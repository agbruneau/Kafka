package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

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

	// Canal pour arrêter proprement la goroutine de métriques
	metricsStopChan := make(chan struct{})
	go logPeriodicMetrics(metricsStopChan)

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true
	consecutiveErrors := 0
	maxConsecutiveErrors := 3 // Arrêter après 3 erreurs consécutives (probablement Kafka arrêté)

	for run {
		select {
		case <-sigchan:
			logLogger.Log(LogLevelINFO, "Signal d'arrêt reçu, fin de la consommation.", nil)
			run = false
		default:
			msg, err := consumer.ReadMessage(1 * time.Second)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					consecutiveErrors = 0 // Réinitialiser le compteur si c'est juste un timeout
					continue              // Pas de message, c'est normal.
				}

				// Vérifier si c'est une erreur de connexion critique (brokers down)
				kafkaErr, ok := err.(kafka.Error)
				isShutdownError := false
				if ok {
					errorMsg := err.Error()
					if strings.Contains(errorMsg, "brokers are down") ||
						strings.Contains(errorMsg, "Connection refused") ||
						kafkaErr.Code() == kafka.ErrAllBrokersDown {
						isShutdownError = true
						consecutiveErrors++
						if consecutiveErrors >= maxConsecutiveErrors {
							// Logger comme INFO au lieu d'ERROR car c'est un arrêt normal
							logLogger.Log(LogLevelINFO, "Kafka semble être arrêté, arrêt du consommateur", map[string]interface{}{
								"consecutive_errors": consecutiveErrors,
								"reason":             "brokers_unavailable",
							})
							run = false
							break
						}
						// Ne pas logger les erreurs intermédiaires de shutdown pour éviter le bruit
						continue
					}
				}

				// Logger seulement les erreurs qui ne sont pas liées à l'arrêt
				if !isShutdownError {
					logLogger.LogError("Erreur de lecture du message Kafka", err, nil)
					consecutiveErrors++
					if consecutiveErrors >= maxConsecutiveErrors {
						logLogger.LogError("Trop d'erreurs consécutives, arrêt du consommateur", err, map[string]interface{}{
							"consecutive_errors": consecutiveErrors,
						})
						run = false
						break
					}
				}
				continue
			}

			// Réinitialiser le compteur d'erreurs en cas de succès
			consecutiveErrors = 0
			processMessage(msg)
		}
	}

	// Arrêter la goroutine de métriques avant de quitter
	close(metricsStopChan)

	// Log final avant de quitter
	uptime := time.Since(systemMetrics.StartTime)
	logLogger.Log(LogLevelINFO, "Consommateur arrêté proprement", map[string]interface{}{
		"uptime_seconds":           uptime.Seconds(),
		"total_messages_received":  systemMetrics.MessagesReceived,
		"total_messages_processed": systemMetrics.MessagesProcessed,
		"total_messages_failed":    systemMetrics.MessagesFailed,
	})
	fmt.Println("\n🔴 Le consommateur est arrêté.")
}
