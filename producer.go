/*
Ce programme Go, producer.go, est conçu pour fonctionner comme un producteur de messages pour Apache Kafka.
Il envoie des messages JSON sérialisés à un topic Kafka spécifié.

Le programme est configuré pour se connecter à un serveur Kafka fonctionnant sur localhost:9092.
Il envoie en continu des messages prédéfinis au topic orders et attend une confirmation de livraison.

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

// Order représente une commande
type Order struct {
	OrderID  string `json:"order_id"`
	User     string `json:"user"`
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
}

func main() {
	// Configuration du producteur Kafka
	producerConfig := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	}

	// Création du producteur
	producer, err := kafka.NewProducer(producerConfig)
	if err != nil {
		fmt.Printf("❌ Erreur lors de la création du producteur: %v\n", err)
		os.Exit(1)
	}
	defer producer.Close()

	// Canal pour gérer l'interruption (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine pour gérer les événements de livraison
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("❌ La livraison a échoué: %v\n", ev.TopicPartition.Error)
				} else {
					fmt.Printf("✅ Message livré à %s [%d] @ offset %v\n",
						*ev.TopicPartition.Topic,
						ev.TopicPartition.Partition,
						ev.TopicPartition.Offset)
					fmt.Printf("   Contenu: %s\n", string(ev.Value))
				}
			}
		}
	}()

	// Boucle principale pour envoyer des messages
	topic := "orders"
	running := true

	for running {
		select {
		case <-sigChan:
			fmt.Println("\n🔴 Arrêt du producteur")
			running = false
		default:
			// Création d'une nouvelle commande
			order := Order{
				OrderID:  uuid.New().String(),
				User:     "lara",
				Item:     "frozen yogurt",
				Quantity: 10,
			}

			// Sérialisation en JSON
			value, err := json.Marshal(order)
			if err != nil {
				fmt.Printf("❌ Erreur de sérialisation: %v\n", err)
				time.Sleep(2 * time.Second)
				continue
			}

			// Envoi du message
			err = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          value,
			}, nil)

			if err != nil {
				fmt.Printf("❌ Erreur lors de l'envoi: %v\n", err)
			}

			// Attendre 2 secondes avant d'envoyer le prochain message
			time.Sleep(2 * time.Second)
		}
	}

	// S'assurer que tous les messages restants sont envoyés avant de fermer
	fmt.Println("⏳ Envoi des messages restants...")
	producer.Flush(5000) // Attendre jusqu'à 5 secondes
}
