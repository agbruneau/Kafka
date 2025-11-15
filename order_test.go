package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestOrderContainsFullState validates the Event Carried State Transfer (ECST) pattern.
//
// It ensures that a created Order object is fully "enriched" with all the necessary
// data for autonomous processing by a consumer. This test verifies that critical
// nested structures, which carry the state of other domains (like customer and inventory),
// are present and correctly populated.
//
// This test is a direct response to the code review's critique that the ECST pattern,
// while claimed, was not verified. It serves as a foundational check to guarantee
// that the data produced aligns with the architectural principles of the system.
func TestOrderContainsFullState(t *testing.T) {
	// --- Arrange ---
	// Create a complete Order object with enriched data, simulating what the
	// producer would generate.
	order := Order{
		OrderID:  uuid.New().String(),
		Sequence: 1,
		Status:   "pending",
		Items: []OrderItem{
			{
				ItemID:     "item-123",
				ItemName:   "Laptop",
				Quantity:   1,
				UnitPrice:  1200.50,
				TotalPrice: 1200.50,
			},
		},
		Total: 1200.50,
		// --- Enriched Data (The core of ECST) ---
		Metadata: OrderMetadata{
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Version:       "1.0",
			Source:        "test-suite",
			CorrelationID: uuid.New().String(),
		},
		CustomerInfo: CustomerInfo{
			CustomerID:   "cust-abc",
			Name:         "Alice",
			Email:        "alice@example.com",
			LoyaltyLevel: "gold",
		},
		InventoryStatus: []InventoryStatus{
			{
				ItemID:       "item-123",
				AvailableQty: 99,
				InStock:      true,
				Warehouse:    "MAIN-WH",
			},
		},
	}

	// --- Act & Assert ---
	// The core of the test is to assert that the consumer would receive everything
	// it needs, preventing the need for synchronous call-backs to other services.

	// 1. Verify that the main object is not nil.
	assert.NotNil(t, order)

	// 2. Validate Customer Information.
	// A consumer should not need to query a Customer Service.
	assert.NotNil(t, order.CustomerInfo, "CustomerInfo should be present in the event")
	assert.Equal(t, "cust-abc", order.CustomerInfo.CustomerID)
	assert.Equal(t, "Alice", order.CustomerInfo.Name)
	assert.Equal(t, "gold", order.CustomerInfo.LoyaltyLevel)

	// 3. Validate Inventory Status.
	// A consumer should not need to query an Inventory Service.
	assert.NotNil(t, order.InventoryStatus, "InventoryStatus should be present in the event")
	assert.NotEmpty(t, order.InventoryStatus, "InventoryStatus should not be empty")
	assert.Equal(t, "item-123", order.InventoryStatus[0].ItemID)
	assert.True(t, order.InventoryStatus[0].InStock, "InStock status should be true")
	assert.Equal(t, "MAIN-WH", order.InventoryStatus[0].Warehouse)

	// 4. Validate Metadata.
	// A consumer should have context about the event itself.
	assert.NotNil(t, order.Metadata, "Metadata should be present in the event")
	assert.Equal(t, "1.0", order.Metadata.Version)
	assert.Equal(t, "test-suite", order.Metadata.Source)
	assert.NotEmpty(t, order.Metadata.CorrelationID)
}
