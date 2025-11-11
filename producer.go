/*
Ce programme Go, producer.go, est conçu pour fonctionner comme un producteur de messages pour Apache Kafka.
Il envoie des messages JSON sérialisés à un topic Kafka spécifié.

Le programme est configuré pour se connecter à un serveur Kafka fonctionnant sur localhost:9092.
Il envoie en continu des messages prédéfinis au topic "orders" et attend une confirmation de livraison.

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

// deliveryReport gère les rapports de livraison des messages
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
	/*
		Point d'entrée principal du programme producteur.

		Initialise le producteur Kafka, envoie des messages de commande en boucle
		au topic 'orders' et attend la confirmation de livraison avant de terminer.
	*/

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

	// Gestion des signaux pour un arrêt propre
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Boucle d'envoi de messages
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	running := true
	for running {
		select {
		case <-sigchan:
			fmt.Println("\n🔴 Arrêt du producteur")
			running = false
		case <-ticker.C:
			// Création d'une commande
			order := Order{
				OrderID:  uuid.New().String(),
				User:     "lara",
				Item:     "frozen yogurt",
				Quantity: 10,
			}

			// Sérialisation en JSON
			value, err := json.Marshal(order)
			if err != nil {
				fmt.Printf("Erreur de sérialisation JSON: %v\n", err)
				continue
			}

			// Envoi du message
			topic := "orders"
			err = producer.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
				Value:          value,
			}, deliveryChan)

			if err != nil {
				fmt.Printf("Erreur lors de l'envoi du message: %v\n", err)
			}

			// Traitement des événements en attente
			producer.Flush(1000)
		}
	}

	// S'assurer que tous les messages restants sont envoyés avant de fermer
	fmt.Println("⏳ Envoi des messages restants...")
	producer.Flush(5000)
	close(deliveryChan)
}
