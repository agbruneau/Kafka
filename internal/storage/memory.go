package storage

import (
	"sort"
	"sync"
	"time"

	"kafka-demo/internal/events"
)

// OrderProjection représente l'état consolidé d'une commande.
type OrderProjection struct {
	Snapshot   events.OrderSnapshot
	EventType  events.OrderEventType
	UpdatedAt  time.Time
	LastError  string
	RetryCount int
}

// UserAggregate représente l'agrégation par utilisateur pour CQRS/lecture.
type UserAggregate struct {
	User         string               `json:"user"`
	Orders       int                  `json:"orders"`
	TotalItems   int                  `json:"total_items"`
	LastEvent    events.OrderEvent    `json:"-"`
	LastUpdated  time.Time            `json:"last_updated"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
	LastSnapshot events.OrderSnapshot `json:"-"`
}

// MemoryStore offre un stockage en mémoire thread-safe pour projections et agrégations.
type MemoryStore struct {
	mu         sync.RWMutex
	orders     map[string]OrderProjection
	aggregates map[string]UserAggregate
}

// NewMemoryStore instancie un stockage vide.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		orders:     make(map[string]OrderProjection),
		aggregates: make(map[string]UserAggregate),
	}
}

// ApplyEvent met à jour projections et agrégations.
func (s *MemoryStore) ApplyEvent(evt events.OrderEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	proj := OrderProjection{
		Snapshot:   evt.Snapshot,
		EventType:  evt.Type,
		UpdatedAt:  evt.OccurredAt,
		LastError:  evt.ErrorMessage,
		RetryCount: evt.RetryCount,
	}

	s.orders[evt.Snapshot.OrderID] = proj

	agg := s.aggregates[evt.Snapshot.User]
	agg.User = evt.Snapshot.User
	agg.LastEvent = evt
	agg.LastSnapshot = evt.Snapshot
	agg.LastUpdated = evt.OccurredAt

	switch evt.Type {
	case events.OrderEventCreated:
		agg.Orders++
		agg.TotalItems += evt.Snapshot.Quantity
	case events.OrderEventApproved:
		agg.Metadata = mergeMetadata(agg.Metadata, map[string]string{"status": "approved"})
	case events.OrderEventRejected, events.OrderEventFailed:
		agg.Metadata = mergeMetadata(agg.Metadata, map[string]string{"status": "failed"})
	}

	s.aggregates[evt.Snapshot.User] = agg
}

func mergeMetadata(dst, src map[string]string) map[string]string {
	if dst == nil {
		dst = make(map[string]string)
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// GetOrder récupère la projection d'une commande.
func (s *MemoryStore) GetOrder(orderID string) (OrderProjection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proj, ok := s.orders[orderID]
	return proj, ok
}

// ListAggregates renvoie toutes les agrégations triées par utilisateur.
func (s *MemoryStore) ListAggregates() []UserAggregate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]UserAggregate, 0, len(s.aggregates))
	for _, agg := range s.aggregates {
		out = append(out, agg)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].User < out[j].User
	})
	return out
}

// GetAggregate renvoie l'agrégat pour un utilisateur donné.
func (s *MemoryStore) GetAggregate(user string) (UserAggregate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agg, ok := s.aggregates[user]
	return agg, ok
}

// Reset vide les données.
func (s *MemoryStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders = make(map[string]OrderProjection)
	s.aggregates = make(map[string]UserAggregate)
}

// ApplyAggregateEvent met à jour l'agrégat depuis un événement dédié.
func (s *MemoryStore) ApplyAggregateEvent(evt events.UserAggregateEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agg := s.aggregates[evt.User]
	agg.User = evt.User
	agg.Orders = evt.Orders
	agg.TotalItems = evt.TotalItems
	agg.LastUpdated = evt.LastUpdated
	agg.Metadata = evt.Metadata
	s.aggregates[evt.User] = agg
}
