package tests

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kafka-demo/internal/bus"
	"kafka-demo/internal/events"
)

func TestEventFlowWithDockerCompose(t *testing.T) {
	if testing.Short() {
		t.Skip("tests d'intégration ignorés en mode short")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker non présent dans le PATH")
	}

	composePath := filepath.Join(repoRoot(t), "docker-compose.yaml")

	runCommand(t, "docker", "compose", "-f", composePath, "up", "-d")
	t.Cleanup(func() {
		runCommand(t, "docker", "compose", "-f", composePath, "down", "-v")
	})

	// Laisser le temps à Kafka de démarrer.
	time.Sleep(20 * time.Second)

	for _, topic := range []string{"orders.events", "orders.commands", "orders.dlq", "orders.aggregates"} {
		runCommand(t, "docker", "exec", "kafka",
			"kafka-topics", "--create", "--if-not-exists",
			"--topic", topic, "--bootstrap-server", "localhost:9092",
			"--partitions", "1", "--replication-factor", "1",
		)
	}

	cfg := bus.Config{
		BootstrapServers: "localhost:9092",
		ClientID:         "integration-producer",
		DLQTopic:         "orders.dlq",
	}

	messageBus, err := bus.NewBus(cfg)
	require.NoError(t, err)
	defer messageBus.Close()

	order := events.OrderEvent{
		EventID:   "evt-1",
		CommandID: "cmd-1",
		SagaID:    "saga-1",
		Type:      events.OrderEventCreated,
		Snapshot: events.OrderSnapshot{
			OrderID:            "order-1",
			User:               "integration-user",
			Item:               "espresso",
			Quantity:           2,
			Sequence:           1,
			PriceCents:         350,
			InventoryAvailable: true,
		},
		Step:       events.SagaStepValidation,
		OccurredAt: time.Now().UTC(),
	}

	payload, err := order.Encode()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = messageBus.Publish(ctx, "orders.events", []byte(order.Snapshot.OrderID), payload, map[string]string{
		"type": string(order.Type),
	})
	cancel()
	require.NoError(t, err)

	consumer, err := messageBus.NewConsumer("integration-consumer", "earliest")
	require.NoError(t, err)
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{"orders.events"}, nil)
	require.NoError(t, err)

	msg, err := consumer.ReadMessage(10 * time.Second)
	require.NoError(t, err)

	received, err := events.DecodeOrderEvent(msg.Value)
	require.NoError(t, err)
	require.Equal(t, order.Snapshot.OrderID, received.Snapshot.OrderID)
	require.Equal(t, order.Type, received.Type)
}

func runCommand(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	if runtime.GOOS == "windows" {
		// Garantir l'utilisation de PowerShell pour docker compose sur Windows si nécessaire.
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("commande %s %v impossible: %v\n%s", name, args, err, string(out))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("impossible de localiser le fichier de test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
