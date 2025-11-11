package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Counters stocke des métriques globales simples.
type Counters struct {
	mu          sync.RWMutex
	produced    atomic.Int64
	consumed    atomic.Int64
	dlq         atomic.Int64
	breakerOpen atomic.Int64
	latency     map[string]time.Duration
}

// Default est l'instance partagée par défaut.
var Default = NewCounters()

// NewCounters crée un collecteur initialisé.
func NewCounters() *Counters {
	return &Counters{
		latency: make(map[string]time.Duration),
	}
}

func (c *Counters) IncProduced() {
	c.produced.Add(1)
}

func (c *Counters) IncConsumed() {
	c.consumed.Add(1)
}

func (c *Counters) IncDLQ() {
	c.dlq.Add(1)
}

func (c *Counters) IncBreakerOpen() {
	c.breakerOpen.Add(1)
}

// ObserveLatency enregistre une latence pour une clé (topic ou service).
func (c *Counters) ObserveLatency(key string, value time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latency[key] = value
}

// Snapshot renvoie les valeurs actuelles.
func (c *Counters) Snapshot() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]any{
		"produced":     c.produced.Load(),
		"consumed":     c.consumed.Load(),
		"dlq":          c.dlq.Load(),
		"breaker_open": c.breakerOpen.Load(),
		"latency":      c.latency,
	}
}
