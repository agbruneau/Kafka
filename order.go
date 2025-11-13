/*
Ce fichier contient les structures de données partagées pour les messages de commande
enrichis avec Event Carried State Transfer (ECST).

Les structures définies ici permettent de transporter l'état complet de la commande
dans chaque message, permettant au consommateur de reconstituer l'état sans appel externe.
*/

package main

// CustomerInfo contient les informations détaillées du client.
type CustomerInfo struct {
	CustomerID   string `json:"customer_id"`   // CustomerID est l'identifiant unique du client.
	Name         string `json:"name"`          // Name est le nom du client.
	Email        string `json:"email"`         // Email est l'adresse email du client.
	Phone        string `json:"phone"`         // Phone est le numéro de téléphone du client.
	Address      string `json:"address"`       // Address est l'adresse de livraison.
	LoyaltyLevel string `json:"loyalty_level"` // LoyaltyLevel est le niveau de fidélité du client.
}

// InventoryStatus représente le statut de l'inventaire pour un article.
type InventoryStatus struct {
	ItemID       string  `json:"item_id"`       // ItemID est l'identifiant unique de l'article.
	ItemName     string  `json:"item_name"`     // ItemName est le nom de l'article.
	AvailableQty int     `json:"available_qty"` // AvailableQty est la quantité disponible en stock.
	ReservedQty  int     `json:"reserved_qty"`  // ReservedQty est la quantité réservée pour cette commande.
	UnitPrice    float64 `json:"unit_price"`    // UnitPrice est le prix unitaire de l'article.
	InStock      bool    `json:"in_stock"`      // InStock indique si l'article est en stock.
	Warehouse    string  `json:"warehouse"`     // Warehouse est l'entrepôt d'origine.
}

// OrderItem représente un article dans la commande avec ses détails.
type OrderItem struct {
	ItemID     string  `json:"item_id"`     // ItemID est l'identifiant unique de l'article.
	ItemName   string  `json:"item_name"`   // ItemName est le nom de l'article.
	Quantity   int     `json:"quantity"`    // Quantity est la quantité commandée.
	UnitPrice  float64 `json:"unit_price"`  // UnitPrice est le prix unitaire.
	TotalPrice float64 `json:"total_price"` // TotalPrice est le prix total pour cet article.
}

// OrderMetadata contient les métadonnées contextuelles de la commande.
type OrderMetadata struct {
	Timestamp     string `json:"timestamp"`      // Timestamp est la date et l'heure de création de la commande (ISO 8601).
	Version       string `json:"version"`        // Version est la version du schéma de message (pour l'évolution du schéma).
	EventType     string `json:"event_type"`      // EventType est le type d'événement (order.created, order.updated, etc.).
	Source        string `json:"source"`         // Source est l'origine de l'événement (service qui a généré l'événement).
	CorrelationID string `json:"correlation_id"` // CorrelationID permet de tracer les événements liés.
}

// Order représente une commande client avec l'état complet (Event Carried State Transfer).
// Cette structure enrichie permet au consommateur de reconstituer l'état sans appel externe.
type Order struct {
	// Identifiants
	OrderID  string `json:"order_id"` // OrderID est l'identifiant unique de la commande.
	Sequence int    `json:"sequence"` // Sequence est un numéro séquentiel pour suivre l'ordre des messages.

	// État complet de la commande
	Status          string           `json:"status"`           // Status est le statut actuel de la commande (pending, confirmed, processing, shipped, delivered, cancelled).
	Items           []OrderItem      `json:"items"`            // Items est la liste complète des articles commandés.
	SubTotal        float64          `json:"sub_total"`        // SubTotal est le sous-total avant taxes et frais.
	Tax             float64          `json:"tax"`              // Tax est le montant des taxes.
	ShippingFee     float64          `json:"shipping_fee"`     // ShippingFee est le montant des frais de livraison.
	Total           float64          `json:"total"`            // Total est le montant total de la commande.
	Currency        string           `json:"currency"`        // Currency est la devise utilisée (EUR, USD, etc.).
	PaymentMethod   string           `json:"payment_method"`   // PaymentMethod est la méthode de paiement utilisée.
	ShippingAddress string           `json:"shipping_address"` // ShippingAddress est l'adresse de livraison complète.

	// Métadonnées contextuelles
	Metadata OrderMetadata `json:"metadata"` // Metadata contient les métadonnées contextuelles de l'événement.

	// Informations client complètes
	CustomerInfo CustomerInfo `json:"customer_info"` // CustomerInfo contient toutes les informations du client.

	// Statut de l'inventaire pour chaque article
	InventoryStatus []InventoryStatus `json:"inventory_status"` // InventoryStatus contient le statut de l'inventaire pour chaque article.
}

