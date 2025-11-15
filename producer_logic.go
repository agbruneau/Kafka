/*
Ce programme Go (`producer.go`) agit comme un producteur de messages pour Apache Kafka.
Son rôle est de simuler la création de commandes enrichies et de les envoyer
de manière continue à un topic Kafka.

Il implémente une logique de production robuste qui met en œuvre plusieurs bonnes pratiques :
- **Event Carried State Transfer** : Il génère des données de commande complètes et autonomes.
- **Publisher/Subscriber** : Il publie des messages dans un topic Kafka.
- **Guaranteed Delivery** : Il utilise un canal de rapport de livraison (`deliveryReport`)
  pour s'assurer que chaque message est bien reçu par le broker Kafka.
- **Graceful Shutdown** : Il intercepte les signaux d'arrêt du système pour terminer proprement
  son exécution, en s'assurant que tous les messages en attente dans le tampon sont
  envoyés avant de quitter (`producer.Flush`).
*/

package main

import (
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

// deliveryReport traite les événements de livraison des messages envoyés par le producteur Kafka.
// Il s'exécute dans une goroutine dédiée pour ne pas bloquer le flux principal de production.
//
// Pour chaque message, il vérifie si la livraison a réussi ou échoué et affiche
// un message de confirmation ou d'erreur en conséquence. C'est un élément clé
// pour s'assurer de la fiabilité de la production.
//
// Paramètres:
//   deliveryChan (chan kafka.Event): Un canal qui reçoit les événements de livraison
//     (kafka.Message) de la part du producteur.
func deliveryReport(deliveryChan chan kafka.Event) {
	for e := range deliveryChan {
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("❌ La livraison du message a échoué: %v\n", m.TopicPartition.Error)
		} else {
			fmt.Printf("✅ Message livré avec succès au topic %s (partition %d) à l'offset %d\n",
				*m.TopicPartition.Topic,
				m.TopicPartition.Partition,
				m.TopicPartition.Offset)
			// Décommenter la ligne suivante pour afficher le contenu de chaque message livré.
			// fmt.Printf("   Contenu: %s\n", string(m.Value))
		}
	}
}

// OrderTemplate defines the structure for generating varied order data.
type OrderTemplate struct {
	User     string
	Item     string
	Quantity int
	Price    float64
}

// createOrder generates a new, fully enriched Order object based on a sequence number
// and a template. This function encapsulates the business logic for creating an order,
// making it independently testable.
//
// By extracting this logic from the main loop, we can write unit tests to verify that
// the generated orders are well-formed and contain all the required enriched data,
// aligning with the Event Carried State Transfer pattern.
//
// Paramètres:
//   sequence (int): The sequential number of the order.
//   template (OrderTemplate): The template containing base data for the order.
//
// Retourne:
//   (Order): A fully populated Order object.
func createOrder(sequence int, template OrderTemplate) Order {
	// Calculs financiers pour la commande.
	itemTotal := float64(template.Quantity) * template.Price
	tax := itemTotal * 0.20 // TVA de 20%
	shippingFee := 2.50
	total := itemTotal + tax + shippingFee

	// Construction de l'objet Order complet avec toutes ses données.
	return Order{
		OrderID:  uuid.New().String(),
		Sequence: sequence,
		Status:   "pending",
		Items: []OrderItem{
			{
				ItemID:     fmt.Sprintf("item-%s", template.Item),
				ItemName:   template.Item,
				Quantity:   template.Quantity,
				UnitPrice:  template.Price,
				TotalPrice: itemTotal,
			},
		},
		SubTotal:        itemTotal,
		Tax:             tax,
		ShippingFee:     shippingFee,
		Total:           total,
		Currency:        "EUR",
		PaymentMethod:   "credit_card",
		ShippingAddress: fmt.Sprintf("%d Rue de la Paix, 75000 Paris", sequence),
		Metadata: OrderMetadata{
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			Version:       "1.1",
			EventType:     "order.created",
			Source:        "producer-service",
			CorrelationID: uuid.New().String(),
		},
		CustomerInfo: CustomerInfo{
			CustomerID:   template.User,
			Name:         fmt.Sprintf("Client %s", template.User),
			Email:        fmt.Sprintf("%s@example.com", template.User),
			Phone:        "+33 6 00 00 00 00",
			Address:      fmt.Sprintf("%d Rue de la Paix, 75000 Paris", sequence),
			LoyaltyLevel: "silver",
		},
		InventoryStatus: []InventoryStatus{
			{
				ItemID:       fmt.Sprintf("item-%s", template.Item),
				ItemName:     template.Item,
				AvailableQty: 100 - template.Quantity,
				ReservedQty:  template.Quantity,
				UnitPrice:    template.Price,
				InStock:      true,
				Warehouse:    "PARIS-01",
			},
		},
	}
}

// main est le point d'entrée du programme producteur.
//
// Son cycle de vie est le suivant :
// 1. Configure et initialise une nouvelle instance de producteur Kafka.
// 2. Lance une goroutine pour gérer les rapports de livraison de manière asynchrone.
// 3. Met en place un canal pour intercepter les signaux d'arrêt du système (Ctrl+C),
//    permettant un arrêt propre.
// 4. Entre dans une boucle infinie pour :
//    a. Générer des données de commande enrichies et complètes.
//    b. Sérialiser la commande en JSON.
//    c. Envoyer le message au topic Kafka 'orders'.
//    d. Marquer une pause de 2 secondes entre chaque envoi.
// 5. Si un signal d'arrêt est reçu, la boucle se termine.
// 6. Avant de quitter, appelle `producer.Flush()` pour s'assurer que tous les messages
//    qui sont encore dans le tampon du producteur sont envoyés à Kafka.