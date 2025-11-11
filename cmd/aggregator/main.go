package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"kafka-demo/internal/bus"
	"kafka-demo/internal/events"
	"kafka-demo/internal/metrics"
	"kafka-demo/internal/storage"
)

const (
	aggregatorTopic = "orders.aggregates"
	eventsTopic     = "orders.events"
)

func main() {
	cfg := bus.Config{
		BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		ClientID:         "orders-aggregator",
		DLQTopic:         "orders.dlq",
	}

	messageBus, err := bus.NewBus(cfg)
	if err != nil {
		log.Fatalf("échec création bus: %v", err)
	}
	defer messageBus.Close()

	store := storage.NewMemoryStore()

	log.Println("🟢 Agrégateur d'ordres démarré")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsPort := getEnv("AGGREGATOR_METRICS_PORT", "")
	var metricsServer *http.Server
	if metricsPort != "" {
		metricsServer = metrics.StartServer(":" + metricsPort)
		defer metrics.Shutdown(context.Background(), metricsServer)
	}

	workerCount := getEnvInt("AGGREGATOR_WORKERS", 1)
	if workerCount < 1 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	consumers := make([]*kafka.Consumer, 0, workerCount)

	for i := 0; i < workerCount; i++ {
		consumer, err := messageBus.NewConsumer("orders-aggregator", "earliest")
		if err != nil {
			log.Fatalf("échec création consumer: %v", err)
		}
		if err := consumer.SubscribeTopics([]string{eventsTopic}, nil); err != nil {
			log.Fatalf("abonnement impossible: %v", err)
		}
		consumers = append(consumers, consumer)

		wg.Add(1)
		go func(id int, c *kafka.Consumer) {
			defer wg.Done()
			consumeLoop(ctx, messageBus, store, c, id)
		}(i, consumer)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigchan:
		log.Printf("🔴 Arrêt de l'agrégateur (signal %s)", sig)
	case <-ctx.Done():
	}

	cancel()
	wg.Wait()
	for _, c := range consumers {
		c.Close()
	}
}

func consumeLoop(ctx context.Context, messageBus *bus.Bus, store *storage.MemoryStore, consumer *kafka.Consumer, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := consumer.ReadMessage(500 * time.Millisecond)
			if err != nil {
				if kafkaErrorIsTimeout(err) {
					continue
				}
				log.Printf("[worker %d] lecture Kafka échouée: %v", workerID, err)
				continue
			}
			handleMessage(ctx, messageBus, store, msg)
		}
	}
}

func handleMessage(ctx context.Context, messageBus *bus.Bus, store *storage.MemoryStore, msg *kafka.Message) {
	event, err := events.DecodeOrderEvent(msg.Value)
	if err != nil {
		log.Printf("désérialisation event impossible: %v", err)
		payload := append([]byte(nil), msg.Value...)
		_ = messageBus.SendToDLQ(context.Background(), string(msg.Key), payload, "aggregator_deser_failed")
		return
	}

	store.ApplyEvent(event)
	metrics.Default.IncConsumed()

	agg, ok := store.GetAggregate(event.Snapshot.User)
	if !ok {
		return
	}

	status := ""
	if agg.Metadata != nil {
		status = agg.Metadata["status"]
	}

	snapshot := events.UserAggregateEvent{
		User:        agg.User,
		Orders:      agg.Orders,
		TotalItems:  agg.TotalItems,
		LastUpdated: agg.LastUpdated,
		Status:      status,
		Metadata:    agg.Metadata,
	}

	payload, err := snapshot.Encode()
	if err != nil {
		log.Printf("sérialisation agrégat impossible: %v", err)
		return
	}

	ctxPublish, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = messageBus.Publish(ctxPublish, aggregatorTopic, []byte(agg.User), payload, map[string]string{
		"source_event": string(event.Type),
	})
	cancel()
	if err != nil {
		log.Printf("publication agrégat impossible: %v", err)
	}
}

func kafkaErrorIsTimeout(err error) bool {
	kErr, ok := err.(kafka.Error)
	return ok && kErr.Code() == kafka.ErrTimedOut
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}
