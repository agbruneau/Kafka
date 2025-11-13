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
