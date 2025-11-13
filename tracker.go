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
var eventLogger *Logger
var instanceID string // Identifiant unique de cette instance pour la scalabilité horizontale

// SystemMetrics représente les métriques système pour l'observabilité
type SystemMetrics struct {
	StartTime           time.Time
	MessagesReceived    int64
	MessagesProcessed   int64
	MessagesFailed      int64
	LastMessageTime     time.Time
	LastProcessedOffset int64
	mu                  sync.RWMutex
}

var systemMetrics = &SystemMetrics{
	StartTime: time.Now(),
}

// EventEntry représente une entrée d'événement (message reçu)
type EventEntry struct {
	Timestamp      string          `json:"timestamp"`
	EventType      string          `json:"event_type"`
	KafkaTopic     string          `json:"kafka_topic,omitempty"`
	KafkaPartition int32           `json:"kafka_partition,omitempty"`
	KafkaOffset    int64           `json:"kafka_offset,omitempty"`
	KafkaKey       string          `json:"kafka_key,omitempty"`
	RawMessage     string          `json:"raw_message"`
	MessageSize    int             `json:"message_size"`
	OrderID        string          `json:"order_id,omitempty"`
	Sequence       int             `json:"sequence,omitempty"`
	Status         string          `json:"status,omitempty"`
	Deserialized   bool            `json:"deserialized"`
	Error          string          `json:"error,omitempty"`
	OrderFull      json.RawMessage `json:"order_full,omitempty"`
}

// initLogger initialise le système de logging
func initLogger() error {
	// Récupérer l'identifiant d'instance depuis la variable d'environnement
	instanceID = os.Getenv("TRACKER_INSTANCE_ID")
	if instanceID == "" {
		// Générer un ID basé sur le PID si non fourni
		instanceID = fmt.Sprintf("instance-%d", os.Getpid())
	}

	// Utiliser des fichiers de logs avec l'ID d'instance pour éviter les conflits
	logFileName := fmt.Sprintf("tracker-%s.log", instanceID)
	eventFileName := fmt.Sprintf("tracker-%s.events", instanceID)

	// Initialiser le logger pour les logs d'observabilité
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le fichier de log: %v", err)
	}

	globalLogger = &Logger{
		file:    logFile,
		encoder: json.NewEncoder(logFile),
	}

	// Initialiser le logger pour les événements (journalisation complète)
	eventFile, err := os.OpenFile(eventFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le fichier d'événements: %v", err)
	}

	eventLogger = &Logger{
		file:    eventFile,
		encoder: json.NewEncoder(eventFile),
	}

	// Vérifier que le fichier a bien été créé
	if eventLogger.file == nil {
		return fmt.Errorf("impossible d'initialiser le fichier d'événements")
	}

	// Log de démarrage du système avec informations d'observabilité
	globalLogger.Log(LogLevelINFO, "Système de journalisation initialisé", map[string]interface{}{
		"instance_id": instanceID,
		"log_file":    logFileName,
		"events_file": eventFileName,
		"start_time":  time.Now().UTC().Format(time.RFC3339),
	})

	// Journaliser un événement de démarrage dans tracker.events pour vérifier que ça fonctionne
	// (Cet événement confirme que le système de journalisation des événements est opérationnel)

	return nil
}

// IncrementMessagesReceived incrémente le compteur de messages reçus
func (sm *SystemMetrics) IncrementMessagesReceived() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.MessagesReceived++
	sm.LastMessageTime = time.Now()
}

// IncrementMessagesProcessed incrémente le compteur de messages traités avec succès
func (sm *SystemMetrics) IncrementMessagesProcessed(offset int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.MessagesProcessed++
	sm.LastProcessedOffset = offset
}

// IncrementMessagesFailed incrémente le compteur de messages en échec
func (sm *SystemMetrics) IncrementMessagesFailed() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.MessagesFailed++
}

// GetMetrics retourne une copie des métriques actuelles
func (sm *SystemMetrics) GetMetrics() SystemMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return SystemMetrics{
		StartTime:           sm.StartTime,
		MessagesReceived:    sm.MessagesReceived,
		MessagesProcessed:   sm.MessagesProcessed,
		MessagesFailed:      sm.MessagesFailed,
		LastMessageTime:     sm.LastMessageTime,
		LastProcessedOffset: sm.LastProcessedOffset,
	}
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
	// Ajouter l'instance ID aux métadonnées si disponible
	if instanceID != "" {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]interface{})
		}
		entry.Metadata["instance_id"] = instanceID
	}

	if err := l.encoder.Encode(entry); err != nil {
		log.Printf("Erreur lors de l'écriture du log: %v", err)
	}

	// Flush pour s'assurer que les logs sont écrits immédiatement
	l.file.Sync()
}

// LogOrder écrit un log spécifique pour une commande avec le contenu complet du message
func (l *Logger) LogOrder(level LogLevel, message string, order Order, kafkaMsg *kafka.Message) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Sérialiser la structure Order complète en JSON pour journalisation
	orderJSON, err := json.Marshal(order)
	if err != nil {
		log.Printf("Erreur lors de la sérialisation de la commande: %v", err)
		orderJSON = []byte("{}")
	}

	// Préparer les métadonnées Kafka
	kafkaMetadata := make(map[string]interface{})
	if kafkaMsg != nil {
		if kafkaMsg.TopicPartition.Topic != nil {
			kafkaMetadata["kafka_topic"] = *kafkaMsg.TopicPartition.Topic
		}
		kafkaMetadata["kafka_partition"] = kafkaMsg.TopicPartition.Partition
		kafkaMetadata["kafka_offset"] = kafkaMsg.TopicPartition.Offset
		if kafkaMsg.Key != nil {
			kafkaMetadata["kafka_key"] = string(kafkaMsg.Key)
		}
		// Le timestamp Kafka est disponible via les headers ou peut être omis
		if !kafkaMsg.Timestamp.IsZero() {
			kafkaMetadata["kafka_timestamp"] = kafkaMsg.Timestamp.Format(time.RFC3339)
		}
	}

	// Préparer les métadonnées complètes incluant le message brut et la structure complète
	metadata := map[string]interface{}{
		"status":           order.Status,
		"total":            order.Total,
		"currency":         order.Currency,
		"customer_id":      order.CustomerInfo.CustomerID,
		"customer_name":    order.CustomerInfo.Name,
		"items_count":      len(order.Items),
		"payment_method":   order.PaymentMethod,
		"items":            order.Items,
		"inventory_status": order.InventoryStatus,
		// Ajout de la structure Order complète sérialisée en JSON
		"order_full": json.RawMessage(orderJSON),
		// Métadonnées Kafka
		"kafka": kafkaMetadata,
	}

	// Ajout du message brut reçu de Kafka (pour traçabilité complète)
	if kafkaMsg != nil && kafkaMsg.Value != nil {
		metadata["raw_message"] = string(kafkaMsg.Value)
	}

	entry := LogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Level:         level,
		Message:       message,
		Service:       "order-tracker",
		OrderID:       order.OrderID,
		Sequence:      order.Sequence,
		EventType:     order.Metadata.EventType,
		CorrelationID: order.Metadata.CorrelationID,
		Metadata:      metadata,
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

// LogRawMessage écrit un log pour un message brut reçu de Kafka (même en cas d'erreur de désérialisation)
func (l *Logger) LogRawMessage(level LogLevel, message string, kafkaMsg *kafka.Message, deserializationError error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Préparer les métadonnées Kafka
	kafkaMetadata := make(map[string]interface{})
	if kafkaMsg != nil {
		if kafkaMsg.TopicPartition.Topic != nil {
			kafkaMetadata["kafka_topic"] = *kafkaMsg.TopicPartition.Topic
		}
		kafkaMetadata["kafka_partition"] = kafkaMsg.TopicPartition.Partition
		kafkaMetadata["kafka_offset"] = kafkaMsg.TopicPartition.Offset
		if kafkaMsg.Key != nil {
			kafkaMetadata["kafka_key"] = string(kafkaMsg.Key)
		}
		if !kafkaMsg.Timestamp.IsZero() {
			kafkaMetadata["kafka_timestamp"] = kafkaMsg.Timestamp.Format(time.RFC3339)
		}
	}

	metadata := map[string]interface{}{
		"kafka": kafkaMetadata,
	}

	// Ajouter le message brut
	if kafkaMsg != nil && kafkaMsg.Value != nil {
		metadata["raw_message"] = string(kafkaMsg.Value)
		metadata["raw_message_size"] = len(kafkaMsg.Value)
	}

	// Ajouter l'erreur de désérialisation si présente
	if deserializationError != nil {
		metadata["deserialization_error"] = deserializationError.Error()
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Service:   "order-tracker",
		Metadata:  metadata,
	}

	if deserializationError != nil {
		entry.Error = deserializationError.Error()
	}

	if err := l.encoder.Encode(entry); err != nil {
		log.Printf("Erreur lors de l'écriture du log: %v", err)
	}

	l.file.Sync()
}

// Close ferme le fichier de log
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// LogEvent journalise un événement (message reçu) dans tracker.events
func (l *Logger) LogEvent(kafkaMsg *kafka.Message, order *Order, deserializationError error) {
	if l == nil {
		log.Printf("ERREUR: eventLogger est nil - impossible de journaliser l'événement")
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		log.Printf("ERREUR: fichier d'événements non initialisé")
		return
	}

	event := EventEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		EventType:    "message.received",
		Deserialized: order != nil,
	}

	// Métadonnées Kafka
	if kafkaMsg != nil {
		if kafkaMsg.TopicPartition.Topic != nil {
			event.KafkaTopic = *kafkaMsg.TopicPartition.Topic
		}
		event.KafkaPartition = kafkaMsg.TopicPartition.Partition
		event.KafkaOffset = int64(kafkaMsg.TopicPartition.Offset)
		if kafkaMsg.Key != nil {
			event.KafkaKey = string(kafkaMsg.Key)
		}
		if kafkaMsg.Value != nil {
			event.RawMessage = string(kafkaMsg.Value)
			event.MessageSize = len(kafkaMsg.Value)
		} else {
			// Si kafkaMsg existe mais Value est nil, initialiser avec chaîne vide
			event.RawMessage = ""
			event.MessageSize = 0
		}
	} else {
		// Si kafkaMsg est nil (événement système), initialiser avec chaîne vide
		event.RawMessage = ""
		event.MessageSize = 0
		event.EventType = "system.startup"
	}

	// Informations de la commande si désérialisée avec succès
	if order != nil {
		event.OrderID = order.OrderID
		event.Sequence = order.Sequence
		event.Status = order.Status
		// Sérialiser la structure Order complète
		orderJSON, err := json.Marshal(order)
		if err == nil {
			event.OrderFull = json.RawMessage(orderJSON)
		}
	}

	// Erreur de désérialisation si présente
	if deserializationError != nil {
		event.Error = deserializationError.Error()
		event.EventType = "message.received.deserialization_error"
	}

	// Encoder et écrire l'événement
	if err := l.encoder.Encode(event); err != nil {
		log.Printf("ERREUR lors de l'écriture de l'événement dans tracker.events: %v", err)
		return
	}

	// S'assurer que les données sont écrites sur le disque
	if err := l.file.Sync(); err != nil {
		log.Printf("ERREUR lors du flush du fichier tracker.events: %v", err)
	}
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
	defer eventLogger.Close()

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

	// Log d'initialisation du consommateur avec informations système
	globalLogger.Log(LogLevelINFO, "Consommateur Kafka initialisé", map[string]interface{}{
		"instance_id":       instanceID,
		"topic":             "orders",
		"group_id":          "order-tracker",
		"bootstrap_server":  "localhost:9092",
		"mode":              "Event Carried State Transfer (ECST)",
		"pattern":           "Competing Consumers (scalabilité horizontale)",
		"auto_offset_reset": "earliest",
		"start_time":        time.Now().UTC().Format(time.RFC3339),
	})

	fmt.Printf("🟢 Instance %s: Le consommateur est en cours d'exécution et abonné au topic 'orders'\n", instanceID)
	fmt.Println("📡 Mode: Event Carried State Transfer (ECST) - État complet dans chaque message")
	fmt.Printf("🔄 Pattern: Competing Consumers (scalabilité horizontale) - Instance %s\n", instanceID)
	fmt.Printf("📝 Les logs d'observabilité système sont enregistrés dans tracker-%s.log\n", instanceID)
	fmt.Printf("📋 La journalisation complète des événements est dans tracker-%s.events\n", instanceID)

	// Vérification que eventLogger est bien initialisé
	if eventLogger == nil {
		fmt.Println("⚠️  ATTENTION: eventLogger n'est pas initialisé - les événements ne seront pas journalisés!")
	} else if eventLogger.file == nil {
		fmt.Println("⚠️  ATTENTION: fichier tracker.events non initialisé - les événements ne seront pas journalisés!")
	} else {
		fmt.Println("✅ Système de journalisation des événements opérationnel")
	}

	// Gestion de l'interruption propre (Ctrl+C)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Ticker pour les métriques périodiques (toutes les 30 secondes)
	metricsTicker := time.NewTicker(30 * time.Second)
	defer metricsTicker.Stop()

	// Goroutine pour logger les métriques périodiques
	go func() {
		for range metricsTicker.C {
			metrics := systemMetrics.GetMetrics()
			uptime := time.Since(metrics.StartTime)

			// Calculer les taux
			var successRate float64
			if metrics.MessagesReceived > 0 {
				successRate = float64(metrics.MessagesProcessed) / float64(metrics.MessagesReceived) * 100
			}

			var messagesPerSecond float64
			if uptime.Seconds() > 0 {
				messagesPerSecond = float64(metrics.MessagesReceived) / uptime.Seconds()
			}

			globalLogger.Log(LogLevelINFO, "Métriques système", map[string]interface{}{
				"instance_id":           instanceID,
				"uptime_seconds":        int64(uptime.Seconds()),
				"messages_received":     metrics.MessagesReceived,
				"messages_processed":    metrics.MessagesProcessed,
				"messages_failed":       metrics.MessagesFailed,
				"success_rate_percent":  fmt.Sprintf("%.2f", successRate),
				"messages_per_second":   fmt.Sprintf("%.2f", messagesPerSecond),
				"last_message_time":     metrics.LastMessageTime.Format(time.RFC3339),
				"last_processed_offset": metrics.LastProcessedOffset,
			})
		}
	}()

	// Boucle de consommation
	run := true
	shutdownRequested := false
	var shutdownTime time.Time

	for run {
		select {
		case <-sigchan:
			// Signal d'arrêt reçu - continuer à traiter les messages en cours
			if !shutdownRequested {
				shutdownRequested = true
				shutdownTime = time.Now()

				globalLogger.Log(LogLevelINFO, "Signal d'arrêt reçu - traitement des messages en cours", map[string]interface{}{
					"instance_id": instanceID,
					"signal":      "SIGINT/SIGTERM",
				})

				fmt.Println("\n⚠️  Signal d'arrêt reçu - traitement des messages en cours...")
				fmt.Println("   (Les messages en attente seront traités avant l'arrêt)")
			}
		default:
			// Si l'arrêt est demandé et qu'on n'a pas reçu de message depuis 5 secondes, arrêter
			if shutdownRequested {
				timeSinceShutdown := time.Since(shutdownTime)
				if timeSinceShutdown > 5*time.Second {
					// Aucun message reçu depuis 5 secondes après le signal - arrêter proprement
					run = false
					break
				}
			}

			// Poll pour recevoir des messages (timeout de 1 seconde)
			msg, err := consumer.ReadMessage(1000 * time.Millisecond)
			if err != nil {
				// Timeout ou erreur temporaire
				kafkaErr, ok := err.(kafka.Error)
				if ok && kafkaErr.Code() == kafka.ErrTimedOut {
					// Si l'arrêt est demandé et qu'on a un timeout, vérifier si on doit arrêter
					if shutdownRequested {
						timeSinceShutdown := time.Since(shutdownTime)
						// Si on a attendu 3 secondes sans message après le signal, arrêter
						if timeSinceShutdown > 3*time.Second {
							run = false
							break
						}
					}
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

			// IMPORTANT: Journaliser TOUS les messages reçus dans tracker.events
			// pour une traçabilité complète, indépendamment du succès de la désérialisation

			// Désérialisation du message
			var order *Order
			var deserializationErr error
			var tempOrder Order

			deserializationErr = json.Unmarshal(msg.Value, &tempOrder)
			if deserializationErr == nil {
				order = &tempOrder
			}

			// Mettre à jour les métriques
			systemMetrics.IncrementMessagesReceived()

			// Journaliser l'événement dans tracker.events (toujours, même en cas d'erreur)
			if eventLogger != nil {
				eventLogger.LogEvent(msg, order, deserializationErr)
			} else {
				log.Printf("ERREUR CRITIQUE: eventLogger est nil - impossible de journaliser l'événement")
			}

			// tracker.log contient les erreurs ET les métriques d'observabilité
			if deserializationErr != nil {
				// Mettre à jour les métriques d'échec
				systemMetrics.IncrementMessagesFailed()

				// Logger l'erreur dans tracker.log
				globalLogger.LogRawMessage(LogLevelERROR, "Erreur lors de la désérialisation du message", msg, deserializationErr)
				fmt.Printf("Erreur lors de la désérialisation: %v\n", deserializationErr)
				continue
			}

			// Mettre à jour les métriques de succès
			if msg != nil {
				systemMetrics.IncrementMessagesProcessed(int64(msg.TopicPartition.Offset))
				// Si on est en mode shutdown et qu'on a traité un message, réinitialiser le timer
				if shutdownRequested {
					shutdownTime = time.Now()
				}
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

	// Log de fermeture propre avec statistiques finales
	metrics := systemMetrics.GetMetrics()
	uptime := time.Since(metrics.StartTime)

	var successRate float64
	if metrics.MessagesReceived > 0 {
		successRate = float64(metrics.MessagesProcessed) / float64(metrics.MessagesReceived) * 100
	}

	shutdownDuration := time.Duration(0)
	if shutdownRequested {
		shutdownDuration = time.Since(shutdownTime)
	}

	globalLogger.Log(LogLevelINFO, "Consommateur arrêté proprement", map[string]interface{}{
		"instance_id":                instanceID,
		"uptime_seconds":             int64(uptime.Seconds()),
		"total_messages_received":    metrics.MessagesReceived,
		"total_messages_processed":   metrics.MessagesProcessed,
		"total_messages_failed":      metrics.MessagesFailed,
		"final_success_rate_percent": fmt.Sprintf("%.2f", successRate),
		"shutdown_duration_seconds":  int64(shutdownDuration.Seconds()),
		"shutdown_time":              time.Now().UTC().Format(time.RFC3339),
	})

	fmt.Println("✅ Tous les messages en cours ont été traités")
}
