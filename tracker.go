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
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// LogLevel représente les niveaux de log disponibles
type LogLevel string

const (
	LogLevelDEBUG LogLevel = "DEBUG"
	LogLevelINFO  LogLevel = "INFO"
	LogLevelWARN  LogLevel = "WARN"
	LogLevelERROR LogLevel = "ERROR"
)

// LogEntry représente une entrée de log structurée
type LogEntry struct {
	Timestamp     string                 `json:"timestamp"`
	Level         LogLevel               `json:"level"`
	Message       string                 `json:"message"`
	Service       string                 `json:"service"`
	OrderID       string                 `json:"order_id,omitempty"`
	Sequence      int                    `json:"sequence,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	EventType     string                 `json:"event_type,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// Logger gère l'écriture des logs dans un fichier
type Logger struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

var globalLogger *Logger

// initLogger initialise le système de logging
func initLogger() error {
	file, err := os.OpenFile("tracker.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le fichier de log: %v", err)
	}

	globalLogger = &Logger{
		file:    file,
		encoder: json.NewEncoder(file),
	}

	// Log de démarrage du système de logging
	globalLogger.Log(LogLevelINFO, "Système de logging initialisé", map[string]interface{}{
		"log_file": "tracker.log",
	})

	return nil
}

// Log écrit une entrée de log structurée
func (l *Logger) Log(level LogLevel, message string, metadata map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Service:   "order-tracker",
		Metadata:  metadata,
	}

	if err := l.encoder.Encode(entry); err != nil {
		log.Printf("Erreur lors de l'écriture du log: %v", err)
	}

	// Flush pour s'assurer que les logs sont écrits immédiatement
	l.file.Sync()
}

// LogOrder écrit un log spécifique pour une commande
func (l *Logger) LogOrder(level LogLevel, message string, order Order) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Level:         level,
		Message:       message,
		Service:       "order-tracker",
		OrderID:       order.OrderID,
		Sequence:      order.Sequence,
		EventType:     order.Metadata.EventType,
		CorrelationID: order.Metadata.CorrelationID,
		Metadata: map[string]interface{}{
			"status":           order.Status,
			"total":            order.Total,
			"currency":         order.Currency,
			"customer_id":      order.CustomerInfo.CustomerID,
			"customer_name":    order.CustomerInfo.Name,
			"items_count":      len(order.Items),
			"payment_method":   order.PaymentMethod,
			"items":            order.Items,
			"inventory_status": order.InventoryStatus,
		},
	}

	if err := l.encoder.Encode(entry); err != nil {
		log.Printf("Erreur lors de l'écriture du log: %v", err)
	}

	l.file.Sync()
}

// LogError écrit un log d'erreur
func (l *Logger) LogError(message string, err error, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["error"] = err.Error()
	l.Log(LogLevelERROR, message, metadata)
}

// Close ferme le fichier de log
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// main initialise et exécute le consommateur Kafka.
// Il configure le consommateur pour se connecter au broker Kafka,
// s'abonne au topic 'orders', et entre dans une boucle de scrutation
// pour recevoir et traiter les messages. La fonction gère également
// les signaux d'arrêt pour une fermeture propre.
func main() {
	// Initialisation du système de logging
	if err := initLogger(); err != nil {
		fmt.Printf("❌ Erreur lors de l'initialisation du logging: %v\n", err)
		os.Exit(1)
	}
	defer globalLogger.Close()

	// Configuration du consommateur
	consumerConfig := kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "order-tracker",
		"auto.offset.reset": "earliest",
	}

	// Création du consommateur
	consumer, err := kafka.NewConsumer(&consumerConfig)
	if err != nil {
		globalLogger.LogError("Erreur lors de la création du consommateur", err, map[string]interface{}{
			"bootstrap_servers": "localhost:9092",
			"group_id":          "order-tracker",
		})
		fmt.Printf("Erreur lors de la création du consommateur: %v\n", err)
		os.Exit(1)
	}
	defer consumer.Close()

	// Abonnement au topic
	err = consumer.SubscribeTopics([]string{"orders"}, nil)
	if err != nil {
		globalLogger.LogError("Erreur lors de l'abonnement au topic", err, map[string]interface{}{
			"topic": "orders",
		})
		fmt.Printf("Erreur lors de l'abonnement au topic: %v\n", err)
		os.Exit(1)
	}

	globalLogger.Log(LogLevelINFO, "Consommateur initialisé et abonné au topic", map[string]interface{}{
		"topic":            "orders",
		"group_id":         "order-tracker",
		"mode":             "Event Carried State Transfer (ECST)",
		"bootstrap_server": "localhost:9092",
	})

	fmt.Println("🟢 Le consommateur est en cours d'exécution et abonné au topic 'orders'")
	fmt.Println("📡 Mode: Event Carried State Transfer (ECST) - État complet dans chaque message")
	fmt.Println("📝 Les logs sont enregistrés dans tracker.log")

	// Gestion de l'interruption propre (Ctrl+C)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Boucle de consommation
	run := true
	for run {
		select {
		case <-sigchan:
			globalLogger.Log(LogLevelINFO, "Arrêt du consommateur demandé", map[string]interface{}{
				"signal": "SIGINT/SIGTERM",
			})
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
				// Log de l'erreur (msg peut être nil en cas d'erreur)
				metadata := make(map[string]interface{})
				if msg != nil {
					metadata["topic"] = msg.TopicPartition.Topic
					metadata["partition"] = msg.TopicPartition.Partition
				}
				globalLogger.LogError("Erreur lors de la lecture du message Kafka", err, metadata)
				fmt.Printf("❌ Erreur: %v\n", err)
				continue
			}

			// Désérialisation du message
			var order Order
			err = json.Unmarshal(msg.Value, &order)
			if err != nil {
				globalLogger.LogError("Erreur lors de la désérialisation du message", err, map[string]interface{}{
					"topic":     *msg.TopicPartition.Topic,
					"partition": msg.TopicPartition.Partition,
					"offset":    msg.TopicPartition.Offset,
					"raw_size":  len(msg.Value),
				})
				fmt.Printf("Erreur lors de la désérialisation: %v\n", err)
				continue
			}

			// Log de la réception de la commande
			globalLogger.LogOrder(LogLevelINFO, "Commande reçue et traitée", order)

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

	// Log de fermeture propre
	globalLogger.Log(LogLevelINFO, "Consommateur arrêté proprement", map[string]interface{}{
		"shutdown_time": time.Now().UTC().Format(time.RFC3339),
	})
}
