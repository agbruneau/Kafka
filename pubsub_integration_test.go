
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCommand is a helper function to execute shell commands and log their output.
// It's used for managing the Docker Compose environment. It prepends 'sudo'
// to Docker commands to handle permission issues in CI/CD environments.
func runCommand(t *testing.T, command string, args ...string) {
	fullCommand := []string{command}
	fullCommand = append(fullCommand, args...)

	// Prepend "sudo" if the command is "docker"
	if command == "docker" {
		fullCommand = append([]string{"sudo"}, fullCommand...)
	}

	cmd := exec.Command(fullCommand[0], fullCommand[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, fmt.Sprintf("Command '%s' failed", strings.Join(fullCommand, " ")))
}

// setupIntegrationTest starts the Docker Compose environment and waits for Kafka to be ready.
func setupIntegrationTest(t *testing.T) {
	t.Log("--- Setting up integration test environment ---")
	runCommand(t, "docker", "compose", "up", "-d")

	// Wait for Kafka to be healthy
	t.Log("Waiting for Kafka to be ready...")
	require.Eventually(t, func() bool {
		cmd := exec.Command("docker", "compose", "exec", "kafka", "kafka-topics", "--bootstrap-server", "localhost:9092", "--list")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Kafka not ready yet, retrying... (error: %v, output: %s)", err, string(output))
			return false
		}
		t.Log("Kafka is ready!")
		return true
	}, 2*time.Minute, 5*time.Second, "Kafka did not become healthy in time")
}

// teardownIntegrationTest stops and removes the Docker Compose environment.
func teardownIntegrationTest(t *testing.T) {
	t.Log("--- Tearing down integration test environment ---")
	runCommand(t, "docker", "compose", "down", "--volumes") // --volumes ensures clean state
}

// TestProducerConsumerIntegration performs an end-to-end test of the PubSub system.
//
// This test validates the entire message flow:
// 1.  It starts the complete environment (Kafka) using Docker Compose.
// 2.  It runs the producer as a separate process to send a single, specific message.
// 3.  It runs the consumer in a goroutine to listen for messages on the 'orders' topic.
// 4.  It captures the output from the consumer's audit log (`tracker.events`).
// 5.  It asserts that the message received and logged by the consumer is identical
//     to the message that was sent by the producer.
// 6.  Finally, it cleans up the environment to ensure no side effects.
//
// This test provides the highest level of confidence that the system components are
// working together correctly.
func TestProducerConsumerIntegration(t *testing.T) {
	// --- Setup ---
	setupIntegrationTest(t)
	defer teardownIntegrationTest(t)

	// Create a unique test order to be sent by the producer
	testOrder := createOrder(999, OrderTemplate{
		User:     "integration-tester",
		Item:     "end-to-end-coffee",
		Quantity: 1,
		Price:    99.99,
	})
	testOrderBytes, err := json.Marshal(testOrder)
	require.NoError(t, err)

	// Context for managing subprocesses
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var receivedOrder Order

	// --- Act ---
	// 1. Run the consumer (tracker) in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		// We expect the consumer to be killed by the context timeout, which is fine
		cmd := exec.CommandContext(ctx, "go", "run", "tracker.go", "order.go")
		cmd.Run()
	}()

	// Give the consumer a moment to start up and subscribe
	time.Sleep(5 * time.Second)

	// 2. Run the producer to send one message
	// We run it as a command-line process to simulate a real-world scenario
	producerCmd := exec.CommandContext(ctx, "go", "run", "producer.go", "order.go")
	producerCmd.Env = append(os.Environ(), "SINGLE_MESSAGE_MODE=true", fmt.Sprintf("SINGLE_MESSAGE_PAYLOAD=%s", string(testOrderBytes)))
	err = producerCmd.Run()
	// This will likely exit with an error due to the context being canceled, which is expected.
	// We are primarily interested in whether it sent the message.

	// 3. Monitor the audit log for the message
	file, err := os.Open("tracker.events")
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, testOrder.OrderID) {
			var event EventEntry
			err := json.Unmarshal([]byte(line), &event)
			require.NoError(t, err)

			err = json.Unmarshal(event.OrderFull, &receivedOrder)
			require.NoError(t, err)

			found = true
			break
		}
	}

	// --- Assert ---
	assert.True(t, found, "The produced message was not found in the consumer's event log")
	assert.Equal(t, testOrder.OrderID, receivedOrder.OrderID, "The OrderID of the received order should match the sent order")
	assert.Equal(t, testOrder.CustomerInfo.CustomerID, receivedOrder.CustomerInfo.CustomerID, "The CustomerID should match")
	assert.Equal(t, testOrder.Items[0].ItemName, receivedOrder.Items[0].ItemName, "The ItemName should match")
}
