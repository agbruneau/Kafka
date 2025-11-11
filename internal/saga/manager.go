package saga

import (
	"sync"
	"time"

	"kafka-demo/internal/events"
)

// State conserve la progression d'une saga.
type State struct {
	SagaID     string
	Current    events.SagaStep
	Finished   bool
	Failed     bool
	UpdatedAt  time.Time
	History    []events.OrderEventType
	LastError  string
	RetryCount int
}

// NextStep calcule l'étape suivante en fonction de l'état courant.
func (s *State) NextStep() events.SagaStep {
	switch s.Current {
	case events.SagaStepValidation:
		return events.SagaStepPayment
	case events.SagaStepPayment:
		return events.SagaStepInventory
	case events.SagaStepInventory:
		return events.SagaStepFinalize
	default:
		return events.SagaStepFinalize
	}
}

// Manager pilote les états de saga en mémoire.
type Manager struct {
	mu    sync.RWMutex
	state map[string]*State
}

// NewManager crée un gestionnaire prêt à l'emploi.
func NewManager() *Manager {
	return &Manager{
		state: make(map[string]*State),
	}
}

// Begin initialise l'état pour une nouvelle saga.
func (m *Manager) Begin(cmd events.OrderCommand) *State {
	m.mu.Lock()
	defer m.mu.Unlock()

	st := &State{
		SagaID:    cmd.SagaID,
		Current:   cmd.Step,
		History:   []events.OrderEventType{},
		UpdatedAt: cmd.IssuedAt,
	}
	m.state[cmd.SagaID] = st
	return st
}

// ApplyEvent met à jour l'état suite à un événement.
func (m *Manager) ApplyEvent(evt events.OrderEvent) *State {
	m.mu.Lock()
	defer m.mu.Unlock()

	st, ok := m.state[evt.SagaID]
	if !ok {
		st = &State{
			SagaID: evt.SagaID,
		}
		m.state[evt.SagaID] = st
	}

	st.History = append(st.History, evt.Type)
	st.UpdatedAt = evt.OccurredAt
	st.Current = evt.Step
	st.RetryCount = evt.RetryCount
	st.LastError = evt.ErrorMessage

	switch evt.Type {
	case events.OrderEventApproved:
		if evt.Step == events.SagaStepFinalize {
			st.Finished = true
		}
	case events.OrderEventRejected, events.OrderEventFailed:
		st.Failed = true
		st.Finished = true
	}

	return st
}

// Get permet de récupérer l'état courant.
func (m *Manager) Get(sagaID string) (*State, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st, ok := m.state[sagaID]
	return st, ok
}

// Reset supprime tous les états existants (utile pour les tests).
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = make(map[string]*State)
}
