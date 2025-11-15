/*
Ce programme Go (`tracker.go`) est un consommateur de messages pour Apache Kafka.
Son rôle principal est de s'abonner au topic 'orders', de recevoir les messages,
de les traiter et de maintenir une observabilité complète du système.

Il met en œuvre plusieurs patrons d'architecture et bonnes pratiques essentiels :
- **Consommation de messages** : Il se connecte à Kafka et écoute en continu les nouveaux messages,
  suivant le modèle Publisher/Subscriber.
- **Désérialisation** : Il transforme les messages JSON entrants en structures Go (`Order`).
- **Observabilité avancée** : Il utilise une stratégie de logging à deux fichiers qui implémente
  deux patrons distincts :
  1. **Application Health Monitoring** (`tracker.log`): Pour les logs système structurés
     (démarrage, arrêt, erreurs, métriques). Ce fichier est optimisé pour le monitoring,
     les dashboards et l'alerte.
  2. **Audit Trail** (`tracker.events`): Pour la journalisation exhaustive de chaque
     message reçu. Ce fichier garantit une traçabilité complète et sert de source de vérité
     immuable pour les données entrantes.
- **Métriques système** : Il collecte et affiche périodiquement des métriques de performance
  (débit, taux de succès, etc.) pour évaluer la santé du service.
- **Graceful Shutdown** : Il gère les signaux d'arrêt (Ctrl+C) pour s'assurer que les messages
  en cours de traitement ne sont pas perdus et que les ressources sont correctement libérées.
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
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
// Elle est conçue pour le patron "Application Health Monitoring".
// Chaque entrée est un log structuré (JSON) contenant des informations sur l'état
// de l'application (démarrage, arrêt, erreurs, métriques). Ce format est optimisé
// pour être ingéré, parsé et visualisé par des outils de monitoring et d'alerte.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EventEntry est la structure d'un événement écrit dans `tracker.events`.
// Elle implémente le patron "Audit Trail" en capturant une copie fidèle et immuable
// de chaque message reçu de Kafka, avec ses métadonnées.
//
// Chaque entrée contient le message brut, le résultat de la tentative de désérialisation,
// et des informations contextuelles comme le topic, la partition et l'offset.
// Ce journal est la source de vérité pour l'audit, le rejeu d'événements et le débogage.
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
// Cette fonction est le cœur de l'implémentation du patron "Audit Trail".
// Elle est appelée pour CHAQUE message reçu, qu'il soit valide ou non, garantissant ainsi
// qu'aucune donnée entrante n'est perdue.
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
// C'est un composant clé du patron "Application Health Monitoring".
// En publiant périodiquement des indicateurs de performance (débit, taux de succès, uptime),
// elle permet de créer des dashboards et des alertes pour surveiller la santé de l'application
// en temps quasi-réel.
func logPeriodicMetrics(stopChan <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return // Arrêt propre de la goroutine
		case <-ticker.C:
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

// processMessage traite un message Kafka individuel.
// Il désérialise, logue, et met à jour les métriques.
func processMessage(msg *kafka.Message) {
	var order Order
	deserializationErr := json.Unmarshal(msg.Value, &order)

	// Étape 1: Journaliser l'événement (toujours).
	// Si la désérialisation échoue, nous passons `nil` pour `order` afin
	// que le journal d'audit reflète correctement l'échec.
	var orderForLog *Order
	if deserializationErr == nil {
		orderForLog = &order
	}
	eventLogger.LogEvent(msg, orderForLog, deserializationErr)

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