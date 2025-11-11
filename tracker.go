/*
Ce programme Go, tracker.go, est un consommateur de messages pour Apache Kafka.
Il est conçu pour suivre les messages d'un topic Kafka spécifié, les désérialiser
et afficher les informations qu'ils contiennent.

Le programme est configuré pour se connecter à un serveur Kafka fonctionnant sur localhost:9092
et s'abonner au topic orders. Il écoute en continu les nouveaux messages et les
affiche dans la console.

Fonctionnalités:
- Configuration et initialisation d'un consommateur Kafka.
- Abonnement à un topic Kafka.
- Boucle de consommation pour recevoir et traiter les messages en temps réel.
- Gestion des erreurs et fermeture propre du consommateur.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Order représente une commande
type Order struct {
	OrderID  string `json:"order_id"`
	User     string `json:"user"`
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
}

func main() {
	// Configuration du consommateur Kafka
	consumerConfig := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "order-tracker",
		"auto.offset.reset": "earliest",
	}

	// Création du consommateur
	consumer, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		fmt.Printf("❌ Erreur lors de la création du consommateur: %v\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Abonnement au topic
	err = consumer.SubscribeTopics([]string{"orders"}, nil)
	if err != nil {
		fmt.Printf("❌ Erreur lors de l'abonnement: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🟢 Le consommateur est en cours d'exécution et abonné au topic 'orders'")

	// Canal pour gérer l'interruption (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	running := true
	for running {
		select {
		case <-sigChan:
			fmt.Println("\n🔴 Arrêt du consommateur")
			running = false
		default:
			// Polling pour recevoir des messages
			ev := consumer.Poll(1000) // Timeout de 1 seconde
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Désérialisation du message
				var order Order
				err := json.Unmarshal(e.Value, &order)
				if err != nil {
					fmt.Printf("❌ Erreur de désérialisation: %v\n", err)
					continue
				}

				// Affichage de la commande
				fmt.Printf("📦 Commande reçue: %d x %s de %s\n",
					order.Quantity, order.Item, order.User)

			case kafka.Error:
				fmt.Printf("❌ Erreur: %v\n", e)
			}
		}
	}
}
