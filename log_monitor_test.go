package main

import (
	"testing"
	"time"
)

// TestProcessEventRealTimeMetricsUpdate verifies that processEvent correctly
// updates the metrics in real-time.
func TestProcessEventRealTimeMetricsUpdate(t *testing.T) {
	// Reset metrics for a clean test environment
	monitorMetrics = &Metrics{
		StartTime: time.Now(),
	}

	// Simulate receiving one successful event
	event := MonitorEventEntry{
		Deserialized: true,
	}
	processEvent(event)

	// Assertions
	if monitorMetrics.MessagesReceived != 1 {
		t.Errorf("Expected MessagesReceived to be 1, got %d", monitorMetrics.MessagesReceived)
	}
	if monitorMetrics.MessagesProcessed != 1 {
		t.Errorf("Expected MessagesProcessed to be 1, got %d", monitorMetrics.MessagesProcessed)
	}
	if monitorMetrics.CurrentSuccessRate != 100.0 {
		t.Errorf("Expected CurrentSuccessRate to be 100.0, got %.2f", monitorMetrics.CurrentSuccessRate)
	}
	if monitorMetrics.CurrentMessagesPerSec == 0 {
		t.Errorf("Expected CurrentMessagesPerSec to be updated, but it was 0")
	}

	// Simulate receiving one failed event
	event = MonitorEventEntry{
		Deserialized: false,
	}
	processEvent(event)

	// Assertions
	if monitorMetrics.MessagesReceived != 2 {
		t.Errorf("Expected MessagesReceived to be 2, got %d", monitorMetrics.MessagesReceived)
	}
	if monitorMetrics.MessagesFailed != 1 {
		t.Errorf("Expected MessagesFailed to be 1, got %d", monitorMetrics.MessagesFailed)
	}
	if monitorMetrics.CurrentSuccessRate != 50.0 {
		t.Errorf("Expected CurrentSuccessRate to be 50.0, got %.2f", monitorMetrics.CurrentSuccessRate)
	}
}
