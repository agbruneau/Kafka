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
	"github.com/stretchr/testify/assert"
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

	expected := "Message livré avec succès au topic orders (partition 1) à l'offset 123"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain '%s', but got '%s'", expected, output)
	}
}

// TestCreateOrderCorrectness verifies that the createOrder function generates a well-formed
// and fully enriched Order object.
//
// This test is crucial for ensuring the quality and consistency of the data being
// produced. By validating the output of the core data generation logic, we can
// prevent issues downstream in the consumer or in any other system that relies
// on this data. It confirms that financial calculations are correct and that all
// necessary metadata and contextual information are included.
func TestCreateOrderCorrectness(t *testing.T) {
	// --- Arrange ---
	// Define a template and sequence number for the test case.
	template := OrderTemplate{
		User:     "test-user",
		Item:     "cappuccino",
		Quantity: 2,
		Price:    3.50,
	}
	sequence := 42

	// --- Act ---
	// Call the function under test to generate an order.
	order := createOrder(sequence, template)

	// --- Assert ---
	// Use a small tolerance for floating-point comparisons to avoid precision issues.
	tolerance := 0.001

	// 1. Basic Information
	assert.Equal(t, sequence, order.Sequence, "Sequence number should match")
	assert.Equal(t, template.User, order.CustomerInfo.CustomerID, "CustomerID should match the template user")

	// 2. Financial Calculations
	expectedSubTotal := 7.00 // 2 * 3.50
	assert.InDelta(t, expectedSubTotal, order.SubTotal, tolerance, "SubTotal should be calculated correctly")

	expectedTax := 1.40 // 7.00 * 0.20
	assert.InDelta(t, expectedTax, order.Tax, tolerance, "Tax should be calculated correctly")

	expectedTotal := 10.90 // 7.00 + 1.40 + 2.50 (shipping)
	assert.InDelta(t, expectedTotal, order.Total, tolerance, "Total should be calculated correctly")

	// 3. Enriched Data (ECST verification)
	assert.Len(t, order.Items, 1, "Order should contain exactly one item")
	assert.Equal(t, template.Item, order.Items[0].ItemName, "Item name should match the template")
	assert.Equal(t, "test-user@example.com", order.CustomerInfo.Email, "Customer email should be formatted correctly")
	assert.Len(t, order.InventoryStatus, 1, "Order should contain exactly one inventory status")
	assert.True(t, order.InventoryStatus[0].InStock, "Inventory status should indicate the item is in stock")
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

	expected := "La livraison du message a échoué: Test error"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain '%s', but got '%s'", expected, output)
	}
}
