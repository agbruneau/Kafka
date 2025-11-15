package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// newTestLogger creates a logger that writes to a buffer for testing purposes.
func newTestLogger(buf *bytes.Buffer) *Logger {
	return &Logger{
		file:    nil, // Not used in test
		encoder: json.NewEncoder(buf),
		mu:      sync.Mutex{},
	}
}

// TestProcessMessageDeserializationFailure verifies that a message with invalid JSON
// is correctly logged as a deserialization failure.
func TestProcessMessageDeserializationFailure(t *testing.T) {
	// Setup: redirect loggers to buffers
	var eventBuf bytes.Buffer
	eventLogger = newTestLogger(&eventBuf)

	var logBuf bytes.Buffer
	logLogger = newTestLogger(&logBuf)

	// Create a Kafka message with invalid JSON
	topic := "orders"
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 1, Offset: 101},
		Value:          []byte(`{"invalid-json"`),
		Timestamp:      time.Now(),
	}

	// Call the function under test
	processMessage(kafkaMsg)

	// Assert: Check the event log for correct deserialization status
	eventLogOutput := eventBuf.String()
	if !strings.Contains(eventLogOutput, `"deserialized":false`) {
		t.Errorf("Expected event log to contain '\"deserialized\":false', but it did not. Log: %s", eventLogOutput)
	}
	if !strings.Contains(eventLogOutput, `"error":"unexpected end of JSON input"`) {
		t.Errorf("Expected event log to contain the JSON parsing error, but it did not. Log: %s", eventLogOutput)
	}

	// Assert: Check the system log for a specific error message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, `"message":"Erreur de désérialisation du message"`) {
		t.Errorf("Expected system log to contain deserialization error message, but it did not. Log: %s", logOutput)
	}
}

// TestProcessMessageSuccess verifies that a valid message is correctly processed and logged.
func TestProcessMessageSuccess(t *testing.T) {
	// Setup: Redirect loggers to buffers and reset metrics
	var eventBuf bytes.Buffer
	eventLogger = newTestLogger(&eventBuf)
	systemMetrics = &SystemMetrics{StartTime: time.Now()} // Reset metrics

	// Create a valid Order and serialize it to JSON
	order := Order{OrderID: "123", Sequence: 1}
	orderJSON, _ := json.Marshal(order)

	// Create a valid Kafka message
	topic := "orders"
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: 1, Offset: 102},
		Value:          orderJSON,
		Timestamp:      time.Now(),
	}

	// Call the function under test
	processMessage(kafkaMsg)

	// Assert: Check the event log for successful deserialization
	eventLogOutput := eventBuf.String()
	if !strings.Contains(eventLogOutput, `"deserialized":true`) {
		t.Errorf("Expected event log to show deserialized as true, but it did not. Log: %s", eventLogOutput)
	}
	if !strings.Contains(eventLogOutput, `"order_id":"123"`) {
		t.Errorf("Expected event log to contain the full order JSON, but it did not. Log: %s", eventLogOutput)
	}

	// Assert: Check that metrics were updated correctly
	if systemMetrics.MessagesReceived != 1 {
		t.Errorf("Expected MessagesReceived to be 1, got %d", systemMetrics.MessagesReceived)
	}
	if systemMetrics.MessagesProcessed != 1 {
		t.Errorf("Expected MessagesProcessed to be 1, got %d", systemMetrics.MessagesProcessed)
	}
	if systemMetrics.MessagesFailed != 0 {
		t.Errorf("Expected MessagesFailed to be 0, got %d", systemMetrics.MessagesFailed)
	}
}

// TestSystemMetricsConcurrentAccess verifies that the SystemMetrics struct
// can be safely accessed and modified from multiple goroutines.
func TestSystemMetricsConcurrentAccess(t *testing.T) {
	// Reset metrics for the test
	systemMetrics = &SystemMetrics{StartTime: time.Now()}
	var wg sync.WaitGroup
	numGoroutines := 100
	numWritesPerGoroutine := 10

	// Simulate concurrent writes (e.g., from multiple message processing goroutines)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numWritesPerGoroutine; j++ {
				// Alternate between processed and failed messages
				if j%2 == 0 {
					systemMetrics.recordMetrics(true, false) // Processed
				} else {
					systemMetrics.recordMetrics(false, true) // Failed
				}
			}
		}()
	}

	wg.Wait()

	// Assert: Check the final state of the metrics
	expectedTotal := int64(numGoroutines * numWritesPerGoroutine)
	if systemMetrics.MessagesReceived != expectedTotal {
		t.Errorf("Expected total MessagesReceived to be %d, got %d", expectedTotal, systemMetrics.MessagesReceived)
	}

	expectedProcessed := int64(numGoroutines * numWritesPerGoroutine / 2)
	if systemMetrics.MessagesProcessed != expectedProcessed {
		t.Errorf("Expected total MessagesProcessed to be %d, got %d", expectedProcessed, systemMetrics.MessagesProcessed)
	}

	expectedFailed := int64(numGoroutines * numWritesPerGoroutine / 2)
	if systemMetrics.MessagesFailed != expectedFailed {
		t.Errorf("Expected total MessagesFailed to be %d, got %d", expectedFailed, systemMetrics.MessagesFailed)
	}
}
