/*
Ce programme Go, `producer.go`, est conçu pour fonctionner comme un producteur de messages pour Apache Kafka.
Il envoie des messages JSON sérialisés à un topic Kafka spécifié.

Le programme est configuré pour se connecter à un serveur Kafka fonctionnant sur `localhost:9092`.
Il envoie en continu des messages prédéfinis au topic `orders` et attend une confirmation de livraison.

Fonctionnalités:
- Configuration et initialisation d'un producteur Kafka.
- Envoi de messages en continu au format JSON.
- Rapport de livraison pour confirmer que les messages ont été bien reçus par le broker Kafka.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)


// deliveryReport gère et affiche les rapports de livraison des messages Kafka.
// Elle écoute sur un canal d'événements Kafka et affiche une confirmation
// si le message a été livré avec succès ou une erreur dans le cas contraire.
//
// Paramètres:
//   deliveryChan (chan kafka.Event): Le canal sur lequel les événements de livraison sont envoyés par le producteur Kafka.
func deliveryReport(deliveryChan chan kafka.Event) {
	for e := range deliveryChan {
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("❌ La livraison a échoué: %v\n", m.TopicPartition.Error)
		} else {
			fmt.Printf("✅ Message livré à %s [%d] @ offset %d\n",
				*m.TopicPartition.Topic,
				m.TopicPartition.Partition,
				m.TopicPartition.Offset)
			fmt.Printf("   Contenu: %s\n", string(m.Value))
		}
	}
}

// main initialise et exécute le producteur Kafka.
// Il configure le producteur, gère les signaux d'arrêt (Ctrl+C),
// et entre dans une boucle pour envoyer des messages de commande en continu
// au topic Kafka 'orders'. La fonction assure également que tous les messages
// en attente sont envoyés avant la fermeture du programme.
func main() {
	// Configuration du producteur
	producerConfig := kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	}

	// Création du producteur
	producer, err := kafka.NewProducer(&producerConfig)
	if err != nil {
		fmt.Printf("Erreur lors de la création du producteur: %v\n", err)
		os.Exit(1)
	}
	defer producer.Close()

	// Canal pour les rapports de livraison
	deliveryChan := make(chan kafka.Event, 10000)
	go deliveryReport(deliveryChan)

	// Gestion de l'interruption propre (Ctrl+C)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Topic Kafka
	topic := "orders"

	fmt.Println("🟢 Le producteur est en cours d'exécution...")

	// Templates simplifiés pour générer des commandes enrichies
	type OrderTemplate struct {
		User     string
		Item     string
		Quantity int
		Price    float64
	}

	orderTemplates := []OrderTemplate{
		{User: "client01", Item: "espresso", Quantity: 2, Price: 2.50},
		{User: "client02", Item: "cappuccino", Quantity: 3, Price: 3.20},
		{User: "client03", Item: "latte", Quantity: 4, Price: 3.50},
		{User: "client04", Item: "macchiato", Quantity: 5, Price: 3.00},
		{User: "client05", Item: "flat white", Quantity: 6, Price: 3.30},
		{User: "client06", Item: "mocha", Quantity: 7, Price: 4.00},
		{User: "client07", Item: "americano", Quantity: 8, Price: 2.80},
		{User: "client08", Item: "chai latte", Quantity: 9, Price: 3.80},
		{User: "client09", Item: "matcha", Quantity: 10, Price: 4.50},
		{User: "client10", Item: "smoothie fraise", Quantity: 11, Price: 5.50},
		{User: "client11", Item: "smoothie mangue", Quantity: 2, Price: 5.50},
		{User: "client12", Item: "jus orange", Quantity: 3, Price: 3.00},
		{User: "client13", Item: "jus pomme", Quantity: 4, Price: 3.00},
		{User: "client14", Item: "granite citron", Quantity: 5, Price: 4.50},
		{User: "client15", Item: "soda gingembre", Quantity: 6, Price: 3.50},
		{User: "client16", Item: "milkshake vanille", Quantity: 7, Price: 5.00},
		{User: "client17", Item: "milkshake chocolat", Quantity: 8, Price: 5.00},
		{User: "client18", Item: "wrap poulet", Quantity: 9, Price: 8.50},
		{User: "client19", Item: "wrap legumes", Quantity: 10, Price: 7.50},
		{User: "client20", Item: "salade cesar", Quantity: 11, Price: 9.00},
		{User: "client21", Item: "salade grecque", Quantity: 2, Price: 8.50},
		{User: "client22", Item: "sandwich club", Quantity: 3, Price: 7.00},
		{User: "client23", Item: "bagel saumon", Quantity: 4, Price: 6.50},
		{User: "client24", Item: "croissant", Quantity: 5, Price: 2.20},
		{User: "client25", Item: "pain chocolat", Quantity: 6, Price: 2.50},
	}

	// Boucle d'envoi de messages
	sequence := 1
	templateIndex := 0
	run := true
	for run {
		select {
		case <-sigchan:
			fmt.Println("\n🔴 Arrêt du producteur")
			run = false
		default:
			// Création d'une nouvelle commande enrichie (Event Carried State Transfer)
			template := orderTemplates[templateIndex%len(orderTemplates)]
			orderID := uuid.New().String()
			correlationID := uuid.New().String()
			
			// Calcul des prix
			itemTotal := float64(template.Quantity) * template.Price
			tax := itemTotal * 0.20 // TVA de 20%
			shippingFee := 2.50
			total := itemTotal + tax + shippingFee

			// Création de l'article de commande
			orderItem := OrderItem{
				ItemID:     fmt.Sprintf("item-%s", template.Item),
				ItemName:   template.Item,
				Quantity:   template.Quantity,
				UnitPrice:  template.Price,
				TotalPrice: itemTotal,
			}

			// Création des informations client
			customerInfo := CustomerInfo{
				CustomerID:   template.User,
				Name:         fmt.Sprintf("Client %s", template.User),
				Email:        fmt.Sprintf("%s@example.com", template.User),
				Phone:        fmt.Sprintf("+33 6 %02d %02d %02d %02d", sequence%100, (sequence*2)%100, (sequence*3)%100, (sequence*4)%100),
				Address:      fmt.Sprintf("%d Rue de la Commande, %d000 Paris", sequence, (sequence%20)+1),
				LoyaltyLevel: []string{"bronze", "silver", "gold", "platinum"}[sequence%4],
			}

			// Création du statut d'inventaire
			availableQty := template.Quantity * 3 // Simulation: stock disponible = 3x la quantité commandée
			inventoryStatus := InventoryStatus{
				ItemID:       orderItem.ItemID,
				ItemName:     template.Item,
				AvailableQty: availableQty,
				ReservedQty:  template.Quantity,
				UnitPrice:    template.Price,
				InStock:      availableQty >= template.Quantity,
				Warehouse:    []string{"PARIS-01", "LYON-02", "MARSEILLE-03"}[sequence%3],
			}

			// Création des métadonnées
			metadata := OrderMetadata{
				Timestamp:     time.Now().UTC().Format(time.RFC3339),
				Version:       "1.0",
				EventType:     "order.created",
				Source:        "order-service",
				CorrelationID: correlationID,
			}

			// Création de la commande complète avec état
			order := Order{
				OrderID:         orderID,
				Sequence:        sequence,
				Status:          "pending",
				Items:           []OrderItem{orderItem},
				SubTotal:        itemTotal,
				Tax:             tax,
				ShippingFee:     shippingFee,
				Total:           total,
				Currency:        "EUR",
				PaymentMethod:   []string{"credit_card", "paypal", "bank_transfer"}[sequence%3],
				ShippingAddress: customerInfo.Address,
				Metadata:        metadata,
				CustomerInfo:    customerInfo,
				InventoryStatus: []InventoryStatus{inventoryStatus},
			}

			// Sérialisation en JSON
			value, err := json.Marshal(order)
			if err != nil {
				fmt.Printf("Erreur lors de la sérialisation JSON: %v\n", err)
				continue
			}

			// Production du message
			err = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          value,
			}, deliveryChan)

			if err != nil {
				fmt.Printf("Erreur lors de la production du message: %v\n", err)
			}

			sequence++
			templateIndex++
			// Attendre 2 secondes avant d'envoyer le prochain message
			time.Sleep(2 * time.Second)
		}
	}

	// S'assurer que tous les messages restants sont envoyés avant de fermer
	fmt.Println("⏳ Envoi des messages restants...")
	producer.Flush(15 * 1000) // 15 secondes timeout
}
