package bus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sony/gobreaker"

	"kafka-demo/internal/metrics"
)

// Config regroupe les paramètres d'accès à Kafka.
type Config struct {
	BootstrapServers string
	ClientID         string
	DLQTopic         string
	DeliveryTimeout  time.Duration
	CircuitBreaker   gobreaker.Settings
}

// Bus encapsule producteur et fonctionnalités transverses.
type Bus struct {
	producer     *kafka.Producer
	deliveryChan chan kafka.Event
	breaker      *gobreaker.CircuitBreaker
	cfg          Config
}

// NewBus construit un Bus prêt à publier.
func NewBus(cfg Config) (*Bus, error) {
	if cfg.BootstrapServers == "" {
		return nil, errors.New("bootstrap servers manquants")
	}

	if cfg.DeliveryTimeout == 0 {
		cfg.DeliveryTimeout = 15 * time.Second
	}

	if cfg.CircuitBreaker.Name == "" {
		cfg.CircuitBreaker = gobreaker.Settings{
			Name:        "kafka_producer_breaker",
			MaxRequests: 5,
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
		}
	}

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.BootstrapServers,
		"client.id":         cfg.ClientID,
	})
	if err != nil {
		return nil, fmt.Errorf("création du producteur: %w", err)
	}

	b := &Bus{
		producer:     producer,
		deliveryChan: make(chan kafka.Event, 128),
		breaker:      gobreaker.NewCircuitBreaker(cfg.CircuitBreaker),
		cfg:          cfg,
	}

	go b.handleDeliveryReports()

	return b, nil
}

// Publish envoie un message sur un topic donné avec circuit breaker.
func (b *Bus) Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	start := time.Now()

	_, err := b.breaker.Execute(func() (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          value,
			Key:            key,
			Timestamp:      start,
		}
		for hk, hv := range headers {
			msg.Headers = append(msg.Headers, kafka.Header{Key: hk, Value: []byte(hv)})
		}

		if err := b.producer.Produce(msg, b.deliveryChan); err != nil {
			return nil, err
		}

		return nil, nil
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			metrics.Default.IncBreakerOpen()
		}
		return err
	}

	metrics.Default.IncProduced()
	metrics.Default.ObserveLatency(topic, time.Since(start))
	return nil
}

// SendToDLQ publie un message dans la dead letter queue.
func (b *Bus) SendToDLQ(ctx context.Context, key string, payload []byte, reason string) error {
	if b.cfg.DLQTopic == "" {
		return errors.New("dlq non configurée")
	}
	headers := map[string]string{
		"reason":     reason,
		"published":  time.Now().UTC().Format(time.RFC3339Nano),
		"dlq-origin": "bus",
	}
	err := b.Publish(ctx, b.cfg.DLQTopic, []byte(key), payload, headers)
	if err == nil {
		metrics.Default.IncDLQ()
	}
	return err
}

func (b *Bus) handleDeliveryReports() {
	for evt := range b.deliveryChan {
		switch m := evt.(type) {
		case *kafka.Message:
			if m.TopicPartition.Error != nil {
				log.Printf("❌ livraison échouée topic=%s err=%v", *m.TopicPartition.Topic, m.TopicPartition.Error)
			}
		default:
			log.Printf("événement Kafka inattendu: %v", evt)
		}
	}
}

// Close attend l'envoi des messages restants et ferme le producteur.
func (b *Bus) Close() {
	if b == nil || b.producer == nil {
		return
	}
	b.producer.Flush(int(b.cfg.DeliveryTimeout.Milliseconds()))
	close(b.deliveryChan)
	b.producer.Close()
}

// NewConsumer crée un consommateur prêt à s'abonner.
func (b *Bus) NewConsumer(groupID string, autoOffset string) (*kafka.Consumer, error) {
	if autoOffset == "" {
		autoOffset = "earliest"
	}

	cfg := &kafka.ConfigMap{
		"bootstrap.servers": b.cfg.BootstrapServers,
		"group.id":          groupID,
		"auto.offset.reset": autoOffset,
	}
	if b.cfg.ClientID != "" {
		cfg.SetKey("client.id", b.cfg.ClientID+"-consumer")
	}

	consumer, err := kafka.NewConsumer(cfg)
	if err != nil {
		return nil, fmt.Errorf("création du consommateur: %w", err)
	}

	return consumer, nil
}
