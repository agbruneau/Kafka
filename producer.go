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
}

// deliveryReport traite les rapports de livraison des messages Kafka
func deliveryReport(deliveryChan chan kafka.Event) {
	for e := range deliveryChan {
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			fmt.Printf("❌ La livraison a échoué: %v\n", m.TopicPartition.Error)
		} else {
			fmt.Printf("✅ Message livré à %s [%d] @ offset %v\n",
				*m.TopicPartition.Topic,
				m.TopicPartition.Partition,
				m.TopicPartition.Offset)
			fmt.Printf("   Contenu: %s\n", string(m.Value))
		}
	}
}

func main() {
	// Configuration du producteur Kafka
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	})

	if err != nil {
		fmt.Printf("Erreur lors de la création du producteur: %v\n", err)
		os.Exit(1)
	}

	defer producer.Close()

	// Canal pour les rapports de livraison
	deliveryChan := make(chan kafka.Event, 10000)
	go deliveryReport(deliveryChan)

	// Canal pour gérer l'interruption (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Boucle principale d'envoi de messages
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\n🔴 Arrêt du producteur")
			fmt.Println("⏳ Envoi des messages restants...")
			producer.Flush(15 * 1000) // Attendre jusqu'à 15 secondes
			close(deliveryChan)
			return

		case <-ticker.C:
			// Créer une nouvelle commande
			order := Order{
				OrderID:  uuid.New().String(),
				User:     "lara",
				Item:     "frozen yogurt",
				Quantity: 10,
			}

			// Sérialiser en JSON
			value, err := json.Marshal(order)
			if err != nil {
				fmt.Printf("Erreur lors de la sérialisation: %v\n", err)
				continue
			}

			// Envoyer le message au topic Kafka
			topic := "orders"
			err = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &topic,
					Partition: kafka.PartitionAny,
				},
				Value: value,
			}, deliveryChan)

			if err != nil {
				fmt.Printf("Erreur lors de l'envoi: %v\n", err)
			}

			// Traiter les événements en attente
			producer.Flush(1000)
		}
	}
}
