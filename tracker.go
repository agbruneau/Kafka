/*
Ce programme Go (`tracker.go`) est un consommateur de messages pour Apache Kafka.
Son rôle principal est de s'abonner au topic 'orders', de recevoir les messages,
de les traiter et de maintenir une observabilité complète du système.

Il met en œuvre plusieurs fonctionnalités clés :
- **Consommation de messages** : Il se connecte à Kafka et écoute en continu les nouveaux messages.
- **Désérialisation** : Il transforme les messages JSON entrants en structures Go (`Order`).
- **Observabilité avancée** : Il utilise une stratégie de logging à deux fichiers :
  1. `tracker.log`: Pour les logs système structurés (démarrage, arrêt, erreurs, métriques).
     Ce fichier est optimisé pour le monitoring et l'alerte.
  2. `tracker.events`: Pour la journalisation exhaustive de chaque message reçu.
     Ce fichier garantit une traçabilité complète et sert de "log d'audit".
- **Métriques système** : Il collecte et affiche périodiquement des métriques de performance
  (débit, taux de succès, etc.).
- **Arrêt propre (Graceful Shutdown)** : Il gère les signaux d'arrêt (Ctrl+C) pour s'assurer
  que les messages en cours de traitement ne sont pas perdus.
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

// LogLevel définit les niveaux de sévérité pour les logs structurés.
type LogLevel string

const (
	LogLevelINFO  LogLevel = "INFO"
	LogLevelERROR LogLevel = "ERROR"
)

// LogEntry est la structure d'un log écrit dans `tracker.log`.
// Elle est conçue pour être facilement parsable par des outils d'analyse de logs.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventEntry est la structure d'un événement écrit dans `tracker.events`.
// Elle capture toutes les informations relatives à un message Kafka reçu.
type EventEntry struct {
	Timestamp      string          `json:"timestamp"`
	EventType      string          `json:"event_type"`
	KafkaTopic     string          `json:"kafka_topic"`
	KafkaPartition int32           `json:"kafka_partition"`
	KafkaOffset    int64           `json:"kafka_offset"`
	RawMessage     string          `json:"raw_message"`
	MessageSize    int             `json:"message_size"`
	Deserialized   bool            `json:"deserialized"`
	Error          string          `json:"error,omitempty"`
	OrderFull      json.RawMessage `json:"order_full,omitempty"`
}

// Logger gère l'écriture concurrente et sécurisée dans un fichier de log.
type Logger struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

// SystemMetrics collecte les métriques de performance du consommateur.
// L'accès à cette structure est protégé par un mutex pour garantir la sécurité en concurrence.
type SystemMetrics struct {
	mu                sync.RWMutex
	StartTime         time.Time
	MessagesReceived  int64
	MessagesProcessed int64
	MessagesFailed    int64
	LastMessageTime   time.Time
}

var (
	logLogger    *Logger       // Logger pour `tracker.log` (observabilité système).
	eventLogger *Logger       // Logger pour `tracker.events` (traçabilité des messages).
	systemMetrics = &SystemMetrics{StartTime: time.Now()}
)

// newLogger initialise un nouveau Logger pour un fichier donné.
func newLogger(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir le fichier %s: %v", filename, err)
	}
	return &Logger{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// initLoggers configure les deux loggers utilisés par l'application.
func initLoggers() (err error) {
	logLogger, err = newLogger("tracker.log")
	if err != nil {
		return err
	}
	eventLogger, err = newLogger("tracker.events")
	if err != nil {
		return err
	}
	logLogger.Log(LogLevelINFO, "Système de journalisation initialisé", map[string]interface{}{
		"log_file":    "tracker.log",
		"events_file": "tracker.events",
	})
	return nil
}

// Log écrit une entrée structurée dans `tracker.log`.
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
	_ = l.encoder.Encode(entry)
}

// LogError est un raccourci pour écrire un message d'erreur dans `tracker.log`.
func (l *Logger) LogError(message string, err error, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     LogLevelERROR,
		Message:   message,
		Service:   "order-tracker",
		Error:     err.Error(),
		Metadata:  metadata,
	}
	l.mu.Lock()
	_ = l.encoder.Encode(entry)
	l.mu.Unlock()
}

// LogEvent écrit un enregistrement complet de message dans `tracker.events`.
// Cette fonction est appelée pour CHAQUE message reçu, qu'il soit valide ou non.
func (l *Logger) LogEvent(msg *kafka.Message, order *Order, deserializationError error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	eventType := "message.received"
	deserialized := order != nil

	if deserializationError != nil {
		eventType = "message.received.deserialization_error"
	}

	event := EventEntry{
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		EventType:      eventType,
		KafkaTopic:     *msg.TopicPartition.Topic,
		KafkaPartition: msg.TopicPartition.Partition,
		KafkaOffset:    int64(msg.TopicPartition.Offset),
		RawMessage:     string(msg.Value),
		MessageSize:    len(msg.Value),
		Deserialized:   deserialized,
	}

	if deserialized {
		orderJSON, _ := json.Marshal(order)
		event.OrderFull = json.RawMessage(orderJSON)
	}

	if deserializationError != nil {
		event.Error = deserializationError.Error()
	}

	_ = l.encoder.Encode(event)
}

// Close ferme proprement les fichiers de log.
func (l *Logger) Close() {
	if l != nil {
		_ = l.file.Close()
	}
}

// recordMetrics met à jour les compteurs de performance.
func (sm *SystemMetrics) recordMetrics(processed, failed bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.MessagesReceived++
	if processed {
		sm.MessagesProcessed++
	}
	if failed {
		sm.MessagesFailed++
	}
	sm.LastMessageTime = time.Now()
}

// logPeriodicMetrics écrit un résumé des métriques dans `tracker.log` à intervalle régulier.
func logPeriodicMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		systemMetrics.mu.RLock()
		uptime := time.Since(systemMetrics.StartTime)
		var successRate float64
		if systemMetrics.MessagesReceived > 0 {
			successRate = float64(systemMetrics.MessagesProcessed) / float64(systemMetrics.MessagesReceived) * 100
		}
		var messagesPerSecond float64
		if uptime.Seconds() > 0 {
			messagesPerSecond = float64(systemMetrics.MessagesReceived) / uptime.Seconds()
		}
		systemMetrics.mu.RUnlock()

		logLogger.Log(LogLevelINFO, "Métriques système périodiques", map[string]interface{}{
			"uptime_seconds":     uptime.Seconds(),
			"messages_received":  systemMetrics.MessagesReceived,
			"messages_processed": systemMetrics.MessagesProcessed,
			"messages_failed":    systemMetrics.MessagesFailed,
			"success_rate_percent": fmt.Sprintf("%.2f", successRate),
			"messages_per_second":  fmt.Sprintf("%.2f", messagesPerSecond),
		})
	}
}

// main est le point d'entrée du programme consommateur.
//
// Son cycle de vie est le suivant :
// 1. Initialise les loggers pour `tracker.log` et `tracker.events`.
// 2. Configure et crée une instance de consommateur Kafka.
// 3. S'abonne au topic 'orders'.
// 4. Lance une goroutine pour publier des métriques de performance toutes les 30 secondes.
// 5. Met en place la gestion des signaux d'arrêt (Ctrl+C).
// 6. Entre dans une boucle de consommation pour lire les messages de Kafka :
//    a. Pour chaque message, tente de le désérialiser.
//    b. Appelle `LogEvent` pour enregistrer le message dans `tracker.events` (succès ou échec).
//    c. Met à jour les métriques de performance.
//    d. Si la désérialisation échoue, loggue une erreur dans `tracker.log`.
//    e. Si elle réussit, affiche les détails de la commande dans la console.
// 7. Si un signal d'arrêt est reçu, la boucle se termine.
// 8. Loggue un message final avec les statistiques complètes de la session avant de s'arrêter.
func main() {
	if err := initLoggers(); err != nil {
		log.Fatalf("Erreur fatale lors de l'initialisation des loggers: %v", err)
	}
	defer logLogger.Close()
	defer eventLogger.Close()

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "order-tracker-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		logLogger.LogError("Erreur lors de la création du consommateur", err, nil)
		log.Fatalf("Erreur fatale: %v", err)
	}
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{"orders"}, nil)
	if err != nil {
		logLogger.LogError("Erreur lors de l'abonnement au topic", err, map[string]interface{}{"topic": "orders"})
		log.Fatalf("Erreur fatale: %v", err)
	}

	logLogger.Log(LogLevelINFO, "Consommateur démarré et abonné au topic 'orders'", nil)
	fmt.Println("🟢 Le consommateur est en cours d'exécution...")
	fmt.Println("📝 Logs d'observabilité système dans tracker.log")
	fmt.Println("📋 Journalisation complète des messages dans tracker.events")

	go logPeriodicMetrics()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true
	for run {
		select {
		case <-sigchan:
			logLogger.Log(LogLevelINFO, "Signal d'arrêt reçu, fin de la consommation.", nil)
			run = false
		default:
			msg, err := consumer.ReadMessage(1 * time.Second)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue // Pas de message, c'est normal.
				}
				logLogger.LogError("Erreur de lecture du message Kafka", err, nil)
				continue
			}

			var order Order
			deserializationErr := json.Unmarshal(msg.Value, &order)

			// Étape 1: Journaliser l'événement (toujours)
			eventLogger.LogEvent(msg, &order, deserializationErr)

			// Étape 2: Mettre à jour les métriques et traiter le message
			if deserializationErr != nil {
				systemMetrics.recordMetrics(false, true)
				logLogger.LogError("Erreur de désérialisation du message", deserializationErr, map[string]interface{}{
					"kafka_offset": msg.TopicPartition.Offset,
					"raw_message":  string(msg.Value),
				})
			} else {
				systemMetrics.recordMetrics(true, false)
				displayOrder(&order)
			}
		}
	}

	// Log final avant de quitter
	uptime := time.Since(systemMetrics.StartTime)
	logLogger.Log(LogLevelINFO, "Consommateur arrêté proprement", map[string]interface{}{
		"uptime_seconds":          uptime.Seconds(),
		"total_messages_received": systemMetrics.MessagesReceived,
		"total_messages_processed": systemMetrics.MessagesProcessed,
		"total_messages_failed":   systemMetrics.MessagesFailed,
	})
	fmt.Println("\n🔴 Le consommateur est arrêté.")
}

// displayOrder affiche les détails d'une commande formatée dans la console.
func displayOrder(order *Order) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("📦 COMMANDE REÇUE #%d (ID: %s)\n", order.Sequence, order.OrderID)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Client: %s (%s)\n", order.CustomerInfo.Name, order.CustomerInfo.CustomerID)
	fmt.Printf("Statut: %s | Total: %.2f %s\n", order.Status, order.Total, order.Currency)
	fmt.Println("Articles:")
	for _, item := range order.Items {
		fmt.Printf("  - %s (x%d) @ %.2f %s\n", item.ItemName, item.Quantity, item.UnitPrice, order.Currency)
	}
	fmt.Println(strings.Repeat("=", 80))
}