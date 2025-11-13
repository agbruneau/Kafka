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
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

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
	fmt.Println("📡 Mode: Event Carried State Transfer (ECST) - État complet dans chaque message")

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

			// Affichage enrichi de la commande avec l'état complet (Event Carried State Transfer)
			fmt.Println("\n" + strings.Repeat("=", 80))
			fmt.Printf("📦 COMMANDE #%d - État complet reçu (ECST)\n", order.Sequence)
			fmt.Println(strings.Repeat("-", 80))

			// Informations de base
			fmt.Printf("🆔 ID Commande: %s\n", order.OrderID)
			fmt.Printf("📊 Statut: %s\n", order.Status)
			fmt.Printf("🕐 Timestamp: %s\n", order.Metadata.Timestamp)
			fmt.Printf("📌 Version: %s | Type: %s | Source: %s\n", order.Metadata.Version, order.Metadata.EventType, order.Metadata.Source)
			fmt.Printf("🔗 Correlation ID: %s\n", order.Metadata.CorrelationID)

			// Informations client
			fmt.Println("\n👤 INFORMATIONS CLIENT:")
			fmt.Printf("   • ID: %s | Nom: %s\n", order.CustomerInfo.CustomerID, order.CustomerInfo.Name)
			fmt.Printf("   • Email: %s | Téléphone: %s\n", order.CustomerInfo.Email, order.CustomerInfo.Phone)
			fmt.Printf("   • Adresse: %s\n", order.CustomerInfo.Address)
			fmt.Printf("   • Niveau de fidélité: %s\n", order.CustomerInfo.LoyaltyLevel)

			// Articles commandés
			fmt.Println("\n🛒 ARTICLES COMMANDÉS:")
			for i, item := range order.Items {
				fmt.Printf("   %d. %s (ID: %s)\n", i+1, item.ItemName, item.ItemID)
				fmt.Printf("      Quantité: %d | Prix unitaire: %.2f %s | Total: %.2f %s\n",
					item.Quantity, item.UnitPrice, order.Currency, item.TotalPrice, order.Currency)
			}

			// Statut de l'inventaire
			fmt.Println("\n📦 STATUT DE L'INVENTAIRE:")
			for i, inv := range order.InventoryStatus {
				stockStatus := "✅ En stock"
				if !inv.InStock {
					stockStatus = "❌ Rupture de stock"
				}
				fmt.Printf("   %d. %s (ID: %s)\n", i+1, inv.ItemName, inv.ItemID)
				fmt.Printf("      %s | Disponible: %d | Réservé: %d | Entrepôt: %s\n",
					stockStatus, inv.AvailableQty, inv.ReservedQty, inv.Warehouse)
			}

			// Détails financiers
			fmt.Println("\n💰 DÉTAILS FINANCIERS:")
			fmt.Printf("   • Sous-total: %.2f %s\n", order.SubTotal, order.Currency)
			fmt.Printf("   • Taxes (TVA): %.2f %s\n", order.Tax, order.Currency)
			fmt.Printf("   • Frais de livraison: %.2f %s\n", order.ShippingFee, order.Currency)
			fmt.Printf("   • TOTAL: %.2f %s\n", order.Total, order.Currency)
			fmt.Printf("   • Méthode de paiement: %s\n", order.PaymentMethod)
			fmt.Printf("   • Adresse de livraison: %s\n", order.ShippingAddress)

			fmt.Println(strings.Repeat("=", 80))
		}
	}
}
