package events

import (
	"encoding/json"
	"time"
)

// OrderCommandType représente une intention adressée au domaine de commande.
type OrderCommandType string

const (
	OrderCommandCreate  OrderCommandType = "OrderCreate"
	OrderCommandApprove OrderCommandType = "OrderApprove"
	OrderCommandReject  OrderCommandType = "OrderReject"
)

// OrderEventType capture les états publiés dans le flux d'événements.
type OrderEventType string

const (
	OrderEventCreated  OrderEventType = "OrderCreated"
	OrderEventApproved OrderEventType = "OrderApproved"
	OrderEventRejected OrderEventType = "OrderRejected"
	OrderEventFailed   OrderEventType = "OrderFailed"
)

// SagaStep identifie une étape de saga lors de l'orchestration.
type SagaStep string

const (
	SagaStepValidation SagaStep = "Validation"
	SagaStepPayment    SagaStep = "Payment"
	SagaStepInventory  SagaStep = "Inventory"
	SagaStepFinalize   SagaStep = "Finalize"
)

// OrderSnapshot transporte l'état complet pour Event-Carried State Transfer.
type OrderSnapshot struct {
	OrderID            string            `json:"order_id"`
	User               string            `json:"user"`
	Item               string            `json:"item"`
	Quantity           int               `json:"quantity"`
	Sequence           int               `json:"sequence"`
	PriceCents         int               `json:"price_cents"`
	InventoryAvailable bool              `json:"inventory_available"`
	Attributes         map[string]string `json:"attributes,omitempty"`
}

// OrderCommand est publié sur le topic de commandes (CQRS - côté écriture).
type OrderCommand struct {
	CommandID string            `json:"command_id"`
	SagaID    string            `json:"saga_id"`
	Type      OrderCommandType  `json:"type"`
	Snapshot  OrderSnapshot     `json:"snapshot"`
	Step      SagaStep          `json:"step"`
	IssuedAt  time.Time         `json:"issued_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// OrderEvent représente l'information métier partagée après traitement.
type OrderEvent struct {
	EventID        string            `json:"event_id"`
	CommandID      string            `json:"command_id"`
	SagaID         string            `json:"saga_id"`
	Type           OrderEventType    `json:"type"`
	Snapshot       OrderSnapshot     `json:"snapshot"`
	Step           SagaStep          `json:"step"`
	OccurredAt     time.Time         `json:"occurred_at"`
	RetryCount     int               `json:"retry_count"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	CorrelationKey string            `json:"correlation_key,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Encode renvoie la version JSON de l'événement.
func (e OrderEvent) Encode() ([]byte, error) {
	return json.Marshal(e)
}

// Encode renvoie la version JSON de la commande.
func (c OrderCommand) Encode() ([]byte, error) {
	return json.Marshal(c)
}

// DecodeOrderEvent convertit un flux JSON en événement.
func DecodeOrderEvent(payload []byte) (OrderEvent, error) {
	var evt OrderEvent
	err := json.Unmarshal(payload, &evt)
	return evt, err
}

// DecodeOrderCommand convertit un flux JSON en commande.
func DecodeOrderCommand(payload []byte) (OrderCommand, error) {
	var cmd OrderCommand
	err := json.Unmarshal(payload, &cmd)
	return cmd, err
}

// UserAggregateEvent représente un instantané agrégé par utilisateur.
type UserAggregateEvent struct {
	User        string            `json:"user"`
	Orders      int               `json:"orders"`
	TotalItems  int               `json:"total_items"`
	LastUpdated time.Time         `json:"last_updated"`
	Status      string            `json:"status,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Encode renvoie la version JSON de l'agrégat.
func (a UserAggregateEvent) Encode() ([]byte, error) {
	return json.Marshal(a)
}

// DecodeUserAggregateEvent convertit un flux JSON en agrégat utilisateur.
func DecodeUserAggregateEvent(payload []byte) (UserAggregateEvent, error) {
	var agg UserAggregateEvent
	err := json.Unmarshal(payload, &agg)
	return agg, err
}
