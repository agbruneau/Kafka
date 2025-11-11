package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"

	"kafka-demo/internal/bus"
	"kafka-demo/internal/events"
	"kafka-demo/internal/storage"
)

const (
	eventsTopic = "orders.events"
)

func main() {
	output := flag.String("output", getEnv("REPLAYER_OUTPUT", "replay-projections.json"), "fichier de sortie JSON pour les projections reconstruites")
	flag.Parse()

	cfg := bus.Config{
		BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		ClientID:         "orders-replayer",
		DLQTopic:         "orders.dlq",
	}

	messageBus, err := bus.NewBus(cfg)
	if err != nil {
		log.Fatalf("échec création bus: %v", err)
	}
	defer messageBus.Close()

	groupID := "orders-replayer-" + uuid.NewString()
	consumer, err := messageBus.NewConsumer(groupID, "earliest")
	if err != nil {
		log.Fatalf("échec création consumer: %v", err)
	}
	defer consumer.Close()

	if err := consumer.SubscribeTopics([]string{eventsTopic}, nil); err != nil {
		log.Fatalf("abonnement impossible: %v", err)
	}

	store := storage.NewMemoryStore()
	timeoutCount := 0

	log.Println("🟢 Replayer événementiel démarré")

	for {
		msg, err := consumer.ReadMessage(500 * time.Millisecond)
		if err != nil {
			if kafkaErrTimeout(err) {
				timeoutCount++
				if timeoutCount > 10 {
					break
				}
				continue
			}
			log.Printf("lecture Kafka impossible: %v", err)
			continue
		}
		timeoutCount = 0

		evt, err := events.DecodeOrderEvent(msg.Value)
		if err != nil {
			log.Printf("désérialisation impossible: %v", err)
			continue
		}

		store.ApplyEvent(evt)
	}

	data := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"aggregates":   store.ListAggregates(),
	}

	file, err := os.Create(*output)
	if err != nil {
		log.Fatalf("échec création fichier sortie: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(data); err != nil {
		log.Fatalf("écriture JSON impossible: %v", err)
	}

	log.Printf("✅ projections reconstruites écrites dans %s", *output)
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
