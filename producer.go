/*
Ce programme Go (`producer.go`) agit comme un producteur de messages pour Apache Kafka.
Son rôle est de simuler la création de commandes enrichies et de les envoyer
de manière continue à un topic Kafka.

Il implémente une logique de production robuste, incluant :
- La connexion à un broker Kafka.
- La génération de données de commande complètes suivant le principe de l'Event Carried State Transfer.
- La sérialisation de ces données en JSON.
- L'envoi asynchrone des messages au topic 'orders'.
- La gestion des rapports de livraison pour confirmer la bonne réception par Kafka.
- Un arrêt propre (graceful shutdown) qui garantit l'envoi de tous les messages en attente.
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
func main() {
	// Configuration du producteur Kafka.
	// "bootstrap.servers" est l'adresse du (ou des) broker(s) Kafka.
	producerConfig := kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	}

	// Crée une nouvelle instance du producteur.
	producer, err := kafka.NewProducer(&producerConfig)
	if err != nil {
		fmt.Printf("Erreur fatale lors de la création du producteur: %v\n", err)
		os.Exit(1)
	}
	defer producer.Close()

	// Crée un canal pour recevoir les rapports de livraison.
	// La goroutine deliveryReport écoutera sur ce canal.
	deliveryChan := make(chan kafka.Event, 10000)
	go deliveryReport(deliveryChan)

	// Met en place la gestion des signaux pour un arrêt propre.
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	topic := "orders"
	fmt.Println("🟢 Le producteur est démarré et prêt à envoyer des messages...")

	// Utilisation de templates pour générer des données de commande variées.
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
	}

	// Boucle principale de production de messages.
	sequence := 1
	run := true
	for run {
		select {
		case <-sigchan:
			// Si un signal d'arrêt est reçu, on sort de la boucle.
			fmt.Println("\n⚠️  Signal d'arrêt reçu. Fin de la production de nouveaux messages...")
			run = false
		default:
			// Création d'une commande enrichie basée sur un template.
			template := orderTemplates[sequence%len(orderTemplates)]
			
			// Calculs financiers pour la commande.
			itemTotal := float64(template.Quantity) * template.Price
			tax := itemTotal * 0.20 // TVA de 20%
			shippingFee := 2.50
			total := itemTotal + tax + shippingFee

			// Construction de l'objet Order complet avec toutes ses données.
			order := Order{
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

			// Sérialisation de l'objet Order en JSON.
			value, err := json.Marshal(order)
			if err != nil {
				fmt.Printf("Erreur de sérialisation JSON: %v\n", err)
				continue
			}

			// Envoi du message au topic Kafka de manière asynchrone.
			err = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          value,
			}, deliveryChan)

			if err != nil {
				fmt.Printf("Erreur lors de la production du message: %v\n", err)
			}

			sequence++
			time.Sleep(2 * time.Second) // Pause entre les envois.
		}
	}

	// Avant de terminer, vider le tampon du producteur pour garantir que tous les
	// messages en attente sont envoyés. C'est une étape cruciale pour l'arrêt propre.
	fmt.Println("⏳ Envoi des messages restants en file d'attente...")
	remainingMessages := producer.Flush(15 * 1000) // Timeout de 15 secondes.
	if remainingMessages > 0 {
		fmt.Printf("⚠️  %d messages n'ont pas pu être envoyés.\n", remainingMessages)
	} else {
		fmt.Println("✅ Tous les messages ont été envoyés avec succès.")
	}
}