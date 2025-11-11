package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// TestDeliveryReportSuccess tests the deliveryReport function for a successful delivery.
func TestDeliveryReportSuccess(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	deliveryChan := make(chan kafka.Event, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		deliveryReport(deliveryChan)
	}()

	topic := "orders"
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: 1,
			Offset:    123,
			Error:     nil,
		},
		Value: []byte("Test message content"),
	}

	deliveryChan <- msg
	close(deliveryChan)

	wg.Wait()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := "Message livré à orders [1] @ offset 123"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain '%s', but got '%s'", expected, output)
	}
}

// TestDeliveryReportFailure tests the deliveryReport function for a failed delivery.
func TestDeliveryReportFailure(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	deliveryChan := make(chan kafka.Event, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		deliveryReport(deliveryChan)
	}()

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Error: fmt.Errorf("Test error"),
		},
	}

	deliveryChan <- msg
	close(deliveryChan)

	wg.Wait()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := "La livraison a échoué: Test error"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain '%s', but got '%s'", expected, output)
	}
}
