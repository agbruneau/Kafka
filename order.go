/*
Ce fichier définit le modèle de données partagé pour les messages de commande.
Il est au cœur de l'approche "Event Carried State Transfer" (ECST) utilisée dans ce projet.

Le principe de l'ECST est d'enrichir chaque événement (ici, une commande) avec
l'état complet nécessaire à son traitement. Cela permet aux services consommateurs
(comme le 'tracker') d'être autonomes et de ne pas avoir à interroger d'autres
services pour obtenir des informations contextuelles (par exemple, les détails du client
ou le statut de l'inventaire).

Ce découplage renforce la résilience et la scalabilité du système.
*/

package main

// CustomerInfo contient les informations détaillées et complètes du client.
// Ces données sont embarquées dans chaque message de commande.
type CustomerInfo struct {
	CustomerID   string `json:"customer_id"`   // Identifiant unique du client.
	Name         string `json:"name"`          // Nom complet du client.
	Email        string `json:"email"`         // Adresse email du client.
	Phone        string `json:"phone"`         // Numéro de téléphone du client.
	Address      string `json:"address"`       // Adresse de livraison principale du client.
	LoyaltyLevel string `json:"loyalty_level"` // Niveau de fidélité (ex: "gold", "silver").
}

// InventoryStatus représente l'état de l'inventaire pour un article spécifique au moment de la commande.
// Ces informations permettent de comprendre le contexte du stock sans interroger un service d'inventaire.
type InventoryStatus struct {
	ItemID       string  `json:"item_id"`       // Identifiant unique de l'article.
	ItemName     string  `json:"item_name"`     // Nom de l'article.
	AvailableQty int     `json:"available_qty"` // Quantité disponible en stock au moment de la commande.
	ReservedQty  int     `json:"reserved_qty"`  // Quantité réservée spécifiquement pour cette commande.
	UnitPrice    float64 `json:"unit_price"`    // Prix unitaire de l'article.
	InStock      bool    `json:"in_stock"`      // Indicateur booléen si l'article était en stock.
	Warehouse    string  `json:"warehouse"`     // Entrepôt d'où l'article est expédié.
}

// OrderItem représente un article individuel au sein d'une commande.
type OrderItem struct {
	ItemID     string  `json:"item_id"`     // Identifiant unique de l'article.
	ItemName   string  `json:"item_name"`   // Nom de l'article.
	Quantity   int     `json:"quantity"`    // Quantité de cet article commandée.
	UnitPrice  float64 `json:"unit_price"`  // Prix unitaire de l'article.
	TotalPrice float64 `json:"total_price"` // Prix total pour cette ligne d'article (Quantity * UnitPrice).
}

// OrderMetadata contient les métadonnées techniques et contextuelles de l'événement de commande.
// Ces informations sont essentielles pour le suivi, le débogage et l'analyse des flux de messages.
type OrderMetadata struct {
	Timestamp     string `json:"timestamp"`      // Horodatage de la création de l'événement au format ISO 8601 (UTC).
	Version       string `json:"version"`        // Version du schéma du message, utile pour la gestion des évolutions.
	EventType     string `json:"event_type"`     // Type d'événement (ex: "order.created", "order.updated").
	Source        string `json:"source"`         // Service ou application ayant généré l'événement (ex: "order-service").
	CorrelationID string `json:"correlation_id"` // Identifiant unique pour suivre un flux de travail à travers plusieurs services.
}

// Order est la structure principale qui représente une commande client complète.
// Elle agrège toutes les informations nécessaires à son traitement en un seul message,
// conformément au principe de l'Event Carried State Transfer.
type Order struct {
	// --- Identifiants principaux ---
	OrderID  string `json:"order_id"` // Identifiant unique de la commande (UUID).
	Sequence int    `json:"sequence"` // Numéro séquentiel pour suivre l'ordre de production des messages.

	// --- État et détails de la commande ---
	Status          string      `json:"status"`           // Statut actuel de la commande (ex: "pending", "shipped").
	Items           []OrderItem `json:"items"`            // Liste complète des articles de la commande.
	SubTotal        float64     `json:"sub_total"`        // Sous-total de la commande (somme des TotalPrice des articles).
	Tax             float64     `json:"tax"`              // Montant des taxes appliquées.
	ShippingFee     float64     `json:"shipping_fee"`     // Frais de livraison.
	Total           float64     `json:"total"`            // Montant total de la commande (SubTotal + Tax + ShippingFee).
	Currency        string      `json:"currency"`         // Devise utilisée pour la transaction (ex: "EUR").
	PaymentMethod   string      `json:"payment_method"`   // Méthode de paiement (ex: "credit_card").
	ShippingAddress string      `json:"shipping_address"` // Adresse de livraison complète pour cette commande.

	// --- Données enrichies (ECST) ---
	Metadata        OrderMetadata     `json:"metadata"`         // Métadonnées techniques de l'événement.
	CustomerInfo    CustomerInfo      `json:"customer_info"`    // Informations complètes sur le client.
	InventoryStatus []InventoryStatus `json:"inventory_status"` // État de l'inventaire pour chaque article au moment de la commande.
}