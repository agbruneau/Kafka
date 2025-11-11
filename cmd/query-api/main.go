package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
)

func main() {
	cfg := bus.Config{
		BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		ClientID:         "orders-query-api",
		DLQTopic:         "orders.dlq",
	}

	messageBus, err := bus.NewBus(cfg)
	if err != nil {
		log.Fatalf("échec création bus: %v", err)
	}
	defer messageBus.Close()

	consumer, err := messageBus.NewConsumer("orders-query-api", "earliest")
	if err != nil {
		log.Fatalf("échec création consumer: %v", err)
	}
	defer consumer.Close()

	if err := consumer.SubscribeTopics([]string{aggregatorTopic}, nil); err != nil {
		log.Fatalf("abonnement impossible: %v", err)
	}

	store := storage.NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	go consumeAggregates(ctx, consumer, store, ready)

	select {
	case <-ready:
		log.Println("✅ projections chargées depuis l'agrégateur")
	case <-time.After(5 * time.Second):
		log.Println("⚠️ aucune projection reçue avant timeout")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/aggregates", listAggregatesHandler(store))
	mux.HandleFunc("/aggregates/", aggregateHandler(store))
	mux.HandleFunc("/metrics", metricsHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := getEnv("QUERY_API_PORT", "8080")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		log.Printf("🟢 API de requête disponible sur :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigchan:
		log.Printf("🔴 arrêt demandé (%s)", sig)
	case err := <-errChan:
		log.Printf("❌ serveur HTTP interrompu: %v", err)
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}

func consumeAggregates(ctx context.Context, consumer *kafka.Consumer, store *storage.MemoryStore, ready chan struct{}) {
	var once sync.Once
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := consumer.ReadMessage(500 * time.Millisecond)
			if err != nil {
				if kafkaErrTimeout(err) {
					once.Do(func() { close(ready) })
					continue
				}
				log.Printf("lecture agrégats impossible: %v", err)
				continue
			}

			agg, err := events.DecodeUserAggregateEvent(msg.Value)
			if err != nil {
				log.Printf("désérialisation agrégat impossible: %v", err)
				_ = consumer.StoreOffsets([]kafka.TopicPartition{msg.TopicPartition})
				continue
			}

			store.ApplyAggregateEvent(agg)
			metrics.Default.IncConsumed()
			once.Do(func() { close(ready) })
		}
	}
}

func listAggregatesHandler(store *storage.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := store.ListAggregates()
		writeJSON(w, http.StatusOK, data)
	}
}

func aggregateHandler(store *storage.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Path[len("/aggregates/"):]
		if user == "" {
			http.Error(w, "user requis", http.StatusBadRequest)
			return
		}
		agg, ok := store.GetAggregate(user)
		if !ok {
			http.Error(w, fmt.Sprintf("agrégat introuvable pour %s", user), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, agg)
	}
}

func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, metrics.Default.Snapshot())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("écriture JSON impossible: %v", err)
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
