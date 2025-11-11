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

// Order représente une commande à envoyer à Kafka
type Order struct {
	OrderID  string `json:"order_id"`
	User     string `json:"user"`
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
	Sequence int    `json:"sequence"`
}

// deliveryReport traite les rapports de livraison des messages
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

	orderTemplates := []Order{
		{User: "client01", Item: "espresso", Quantity: 2},
		{User: "client02", Item: "cappuccino", Quantity: 3},
		{User: "client03", Item: "latte", Quantity: 4},
		{User: "client04", Item: "macchiato", Quantity: 5},
		{User: "client05", Item: "flat white", Quantity: 6},
		{User: "client06", Item: "mocha", Quantity: 7},
		{User: "client07", Item: "americano", Quantity: 8},
		{User: "client08", Item: "chai latte", Quantity: 9},
		{User: "client09", Item: "matcha", Quantity: 10},
		{User: "client10", Item: "smoothie fraise", Quantity: 11},
		{User: "client11", Item: "smoothie mangue", Quantity: 2},
		{User: "client12", Item: "jus orange", Quantity: 3},
		{User: "client13", Item: "jus pomme", Quantity: 4},
		{User: "client14", Item: "granite citron", Quantity: 5},
		{User: "client15", Item: "soda gingembre", Quantity: 6},
		{User: "client16", Item: "milkshake vanille", Quantity: 7},
		{User: "client17", Item: "milkshake chocolat", Quantity: 8},
		{User: "client18", Item: "wrap poulet", Quantity: 9},
		{User: "client19", Item: "wrap legumes", Quantity: 10},
		{User: "client20", Item: "salade cesar", Quantity: 11},
		{User: "client21", Item: "salade grecque", Quantity: 2},
		{User: "client22", Item: "sandwich club", Quantity: 3},
		{User: "client23", Item: "bagel saumon", Quantity: 4},
		{User: "client24", Item: "croissant", Quantity: 5},
		{User: "client25", Item: "pain chocolat", Quantity: 6},
		{User: "client26", Item: "espresso", Quantity: 5},
		{User: "client27", Item: "cappuccino", Quantity: 6},
		{User: "client28", Item: "latte", Quantity: 7},
		{User: "client29", Item: "macchiato", Quantity: 8},
		{User: "client30", Item: "flat white", Quantity: 9},
		{User: "client31", Item: "mocha", Quantity: 10},
		{User: "client32", Item: "americano", Quantity: 11},
		{User: "client33", Item: "chai latte", Quantity: 12},
		{User: "client34", Item: "matcha", Quantity: 13},
		{User: "client35", Item: "smoothie fraise", Quantity: 14},
		{User: "client36", Item: "smoothie mangue", Quantity: 5},
		{User: "client37", Item: "jus orange", Quantity: 6},
		{User: "client38", Item: "jus pomme", Quantity: 7},
		{User: "client39", Item: "granite citron", Quantity: 8},
		{User: "client40", Item: "soda gingembre", Quantity: 9},
		{User: "client41", Item: "milkshake vanille", Quantity: 10},
		{User: "client42", Item: "milkshake chocolat", Quantity: 11},
		{User: "client43", Item: "wrap poulet", Quantity: 12},
		{User: "client44", Item: "wrap legumes", Quantity: 13},
		{User: "client45", Item: "salade cesar", Quantity: 14},
		{User: "client46", Item: "salade grecque", Quantity: 5},
		{User: "client47", Item: "sandwich club", Quantity: 6},
		{User: "client48", Item: "bagel saumon", Quantity: 7},
		{User: "client49", Item: "croissant", Quantity: 8},
		{User: "client50", Item: "pain chocolat", Quantity: 9},
		{User: "client51", Item: "espresso", Quantity: 8},
		{User: "client52", Item: "cappuccino", Quantity: 9},
		{User: "client53", Item: "latte", Quantity: 10},
		{User: "client54", Item: "macchiato", Quantity: 11},
		{User: "client55", Item: "flat white", Quantity: 12},
		{User: "client56", Item: "mocha", Quantity: 13},
		{User: "client57", Item: "americano", Quantity: 14},
		{User: "client58", Item: "chai latte", Quantity: 15},
		{User: "client59", Item: "matcha", Quantity: 16},
		{User: "client60", Item: "smoothie fraise", Quantity: 17},
		{User: "client61", Item: "smoothie mangue", Quantity: 8},
		{User: "client62", Item: "jus orange", Quantity: 9},
		{User: "client63", Item: "jus pomme", Quantity: 10},
		{User: "client64", Item: "granite citron", Quantity: 11},
		{User: "client65", Item: "soda gingembre", Quantity: 12},
		{User: "client66", Item: "milkshake vanille", Quantity: 13},
		{User: "client67", Item: "milkshake chocolat", Quantity: 14},
		{User: "client68", Item: "wrap poulet", Quantity: 15},
		{User: "client69", Item: "wrap legumes", Quantity: 16},
		{User: "client70", Item: "salade cesar", Quantity: 17},
		{User: "client71", Item: "salade grecque", Quantity: 8},
		{User: "client72", Item: "sandwich club", Quantity: 9},
		{User: "client73", Item: "bagel saumon", Quantity: 10},
		{User: "client74", Item: "croissant", Quantity: 11},
		{User: "client75", Item: "pain chocolat", Quantity: 12},
		{User: "client76", Item: "espresso", Quantity: 11},
		{User: "client77", Item: "cappuccino", Quantity: 12},
		{User: "client78", Item: "latte", Quantity: 13},
		{User: "client79", Item: "macchiato", Quantity: 14},
		{User: "client80", Item: "flat white", Quantity: 15},
		{User: "client81", Item: "mocha", Quantity: 16},
		{User: "client82", Item: "americano", Quantity: 17},
		{User: "client83", Item: "chai latte", Quantity: 18},
		{User: "client84", Item: "matcha", Quantity: 19},
		{User: "client85", Item: "smoothie fraise", Quantity: 20},
		{User: "client86", Item: "smoothie mangue", Quantity: 11},
		{User: "client87", Item: "jus orange", Quantity: 12},
		{User: "client88", Item: "jus pomme", Quantity: 13},
		{User: "client89", Item: "granite citron", Quantity: 14},
		{User: "client90", Item: "soda gingembre", Quantity: 15},
		{User: "client91", Item: "milkshake vanille", Quantity: 16},
		{User: "client92", Item: "milkshake chocolat", Quantity: 17},
		{User: "client93", Item: "wrap poulet", Quantity: 18},
		{User: "client94", Item: "wrap legumes", Quantity: 19},
		{User: "client95", Item: "salade cesar", Quantity: 20},
		{User: "client96", Item: "salade grecque", Quantity: 11},
		{User: "client97", Item: "sandwich club", Quantity: 12},
		{User: "client98", Item: "bagel saumon", Quantity: 13},
		{User: "client99", Item: "croissant", Quantity: 14},
		{User: "client100", Item: "pain chocolat", Quantity: 15},
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
			// Création d'une nouvelle commande
			template := orderTemplates[templateIndex%len(orderTemplates)]
			order := Order{
				OrderID:  uuid.New().String(),
				User:     template.User,
				Item:     template.Item,
				Quantity: template.Quantity,
				Sequence: sequence,
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
