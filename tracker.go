/*
Ce programme Go, `tracker.go`, est un consommateur de messages pour Apache Kafka.
Il est conçu pour suivre les messages d'un topic Kafka spécifié, les désérialiser
et afficher les informations qu'ils contiennent.

Le programme est configuré pour se connecter à un serveur Kafka fonctionnant sur `localhost:9092`
et s'abonner au topic `orders`. Il écoute en continu les nouveaux messages et les
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
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Order représente une commande client avec tous ses détails.
// Cette structure est utilisée pour désérialiser les données JSON reçues de Kafka
// en un objet Go manipulable.
type Order struct {
	OrderID  string `json:"order_id"`  // OrderID est l'identifiant unique de la commande.
	User     string `json:"user"`      // User est l'identifiant du client qui a passé la commande.
	Item     string `json:"item"`      // Item est le nom du produit commandé.
	Quantity int    `json:"quantity"`  // Quantity est le nombre d'unités du produit commandé.
	Sequence int    `json:"sequence"`  // Sequence est un numéro séquentiel pour suivre l'ordre des messages.
}

// main initialise et exécute le consommateur Kafka.
// Il configure le consommateur pour se connecter au broker Kafka,
// s'abonne au topic 'orders', et entre dans une boucle de scrutation
// pour recevoir et traiter les messages. La fonction gère également
// les signaux d'arrêt pour une fermeture propre.
func main() {
	// Configuration du consommateur
	consumerConfig := kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "order-tracker",
		"auto.offset.reset": "earliest",
	}

	// Création du consommateur
	consumer, err := kafka.NewConsumer(&consumerConfig)
	if err != nil {
		fmt.Printf("Erreur lors de la création du consommateur: %v\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Abonnement au topic
	err = consumer.SubscribeTopics([]string{"orders"}, nil)
	if err != nil {
		fmt.Printf("Erreur lors de l'abonnement au topic: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🟢 Le consommateur est en cours d'exécution et abonné au topic 'orders'")

	// Gestion de l'interruption propre (Ctrl+C)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Boucle de consommation
	run := true
	for run {
		select {
		case <-sigchan:
			fmt.Println("\n🔴 Arrêt du consommateur")
			run = false
		default:
			// Poll pour recevoir des messages (timeout de 1 seconde)
			msg, err := consumer.ReadMessage(1000 * time.Millisecond)
			if err != nil {
				// Timeout ou erreur temporaire
				kafkaErr, ok := err.(kafka.Error)
				if ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				fmt.Printf("❌ Erreur: %v\n", err)
				continue
			}

			// Désérialisation du message
			var order Order
			err = json.Unmarshal(msg.Value, &order)
			if err != nil {
				fmt.Printf("Erreur lors de la désérialisation: %v\n", err)
				continue
			}

			// Affichage de la commande
			fmt.Printf("📦 Commande #%d reçue: %d x %s de %s\n", order.Sequence, order.Quantity, order.Item, order.User)
		}
	}
}
