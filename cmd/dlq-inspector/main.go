package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"kafka-demo/internal/bus"
	"kafka-demo/internal/metrics"
)

const (
	dlqTopic = "orders.dlq"
)

func main() {
	cfg := bus.Config{
		BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		ClientID:         "orders-dlq-inspector",
	}

	messageBus, err := bus.NewBus(cfg)
	if err != nil {
		log.Fatalf("échec création bus: %v", err)
	}
	defer messageBus.Close()

	consumer, err := messageBus.NewConsumer("orders-dlq-inspector", "earliest")
	if err != nil {
		log.Fatalf("échec création consumer: %v", err)
	}
	defer consumer.Close()

	if err := consumer.SubscribeTopics([]string{dlqTopic}, nil); err != nil {
		log.Fatalf("abonnement impossible: %v", err)
	}

	metricsPort := getEnv("DLQ_METRICS_PORT", "")
	var metricsServer *http.Server
	if metricsPort != "" {
		metricsServer = metrics.StartServer(":" + metricsPort)
		defer metrics.Shutdown(context.Background(), metricsServer)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("🟢 Inspection DLQ démarrée")

	for {
		select {
		case <-sigchan:
			log.Println("🔴 Arrêt du DLQ inspector")
			return
		default:
			msg, err := consumer.ReadMessage(500 * time.Millisecond)
			if err != nil {
				if kafkaErrTimeout(err) {
					continue
				}
				log.Printf("lecture DLQ impossible: %v", err)
				continue
			}

			headers := map[string]string{}
			for _, h := range msg.Headers {
				headers[h.Key] = string(h.Value)
			}

			log.Printf("📥 DLQ message key=%s headers=%v payload=%s", string(msg.Key), headers, string(msg.Value))
			metrics.Default.IncDLQ()
		}
	}
}

func kafkaErrTimeout(err error) bool {
	kErr, ok := err.(kafka.Error)
	return ok && kErr.Code() == kafka.ErrTimedOut
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
