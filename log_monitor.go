/*
Ce programme Go (`log_monitor.go`) est un moniteur de logs en temps réel pour le système Kafka Demo.
Il surveille en continu les fichiers tracker.log et tracker.events pour fournir une visualisation
graphique et ergonomique des métriques système et des événements.

Fonctionnalités :
- Surveillance en temps réel des logs tracker.log (métriques système)
- Surveillance en temps réel des événements tracker.events (audit trail)
- Interface graphique interactive avec graphiques et tableaux
- Métriques en temps réel : débit, taux de succès, messages traités
- Affichage des logs et événements récents
- Mise à jour automatique de l'interface
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// MonitorLogEntry représente une entrée structurée du fichier `tracker.log`.
// Elle est utilisée pour désérialiser les lignes de log JSON provenant du monitoring
// de l'état de l'application.
type MonitorLogEntry struct {
	Timestamp string                 `json:"timestamp"` // Horodatage du log.
	Level     string                 `json:"level"`     // Niveau de sévérité (ex: "INFO", "ERROR").
	Message   string                 `json:"message"`   // Message principal du log.
	Service   string                 `json:"service"`   // Nom du service émetteur.
	Error     string                 `json:"error,omitempty"` // Message d'erreur, si applicable.
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // Données contextuelles supplémentaires.
}

// MonitorEventEntry représente une entrée structurée du fichier `tracker.events`.
// Elle est utilisée pour désérialiser les lignes de log JSON qui constituent la piste d'audit
// des messages Kafka reçus.
type MonitorEventEntry struct {
	Timestamp      string          `json:"timestamp"`      // Horodatage de la réception de l'événement.
	EventType      string          `json:"event_type"`     // Type d'événement (ex: "message.received").
	KafkaTopic     string          `json:"kafka_topic"`    // Topic Kafka d'origine.
	KafkaPartition int32           `json:"kafka_partition"`// Partition Kafka d'origine.
	KafkaOffset    int64           `json:"kafka_offset"`   // Offset du message dans la partition.
	RawMessage     string          `json:"raw_message"`    // Contenu brut du message.
	MessageSize    int             `json:"message_size"`   // Taille du message en octets.
	Deserialized   bool            `json:"deserialized"`   // Indique si la désérialisation a réussi.
	Error          string          `json:"error,omitempty"`// Erreur de désérialisation, si applicable.
	OrderFull      json.RawMessage `json:"order_full,omitempty"` // Contenu complet de la commande, si la désérialisation a réussi.
}

// HealthStatus définit les niveaux de santé pour les indicateurs du tableau de bord.
// Il est utilisé pour déterminer la couleur et le texte à afficher pour chaque métrique.
type HealthStatus int

const (
	HealthGood     HealthStatus = iota // Indique une condition saine, typiquement affichée en vert.
	HealthWarning                      // Indique un avertissement, typiquement affiché en jaune.
	HealthCritical                     // Indique un état critique, typiquement affiché en rouge.
)

// Metrics agrège et gère l'état de toutes les métriques collectées par le moniteur.
// L'accès à cette structure est protégé par un RWMutex pour garantir la sécurité
// lors des lectures et écritures concurrentes.
type Metrics struct {
	mu                    sync.RWMutex        // Mutex pour un accès concurrent sécurisé.
	StartTime             time.Time           // Heure de démarrage du moniteur.
	MessagesReceived      int64               // Nombre total de messages reçus.
	MessagesProcessed     int64               // Nombre de messages traités avec succès.
	MessagesFailed        int64               // Nombre de messages qui ont échoué au traitement.
	MessagesPerSecond     []float64           // Historique des débits de messages par seconde pour le graphique.
	SuccessRateHistory    []float64           // Historique des taux de succès pour le graphique.
	RecentLogs            []MonitorLogEntry   // Slice des dernières entrées de log de `tracker.log`.
	RecentEvents          []MonitorEventEntry // Slice des derniers événements de `tracker.events`.
	LastUpdateTime        time.Time           // Heure de la dernière mise à jour des métriques.
	Uptime                time.Duration       // Durée de fonctionnement du moniteur.
	CurrentMessagesPerSec float64             // Valeur actuelle du débit de messages.
	CurrentSuccessRate    float64             // Valeur actuelle du taux de succès.
	ErrorCount            int64               // Nombre total d'erreurs détectées.
	LastErrorTime         time.Time           // Heure de la dernière erreur enregistrée.
}

var monitorMetrics = &Metrics{
	StartTime:          time.Now(),
	RecentLogs:         make([]MonitorLogEntry, 0, 20),
	RecentEvents:       make([]MonitorEventEntry, 0, 20),
	MessagesPerSecond:  make([]float64, 0, 50),
	SuccessRateHistory: make([]float64, 0, 50),
	LastErrorTime:      time.Time{},
}

// monitorFile surveille un fichier en continu, similaire à la commande `tail -f`.
// Il lit les nouvelles lignes ajoutées au fichier et les envoie sur des canaux
// appropriés pour un traitement asynchrone. La fonction gère aussi la recréation
// et la troncature du fichier.
//
// Paramètres:
//   filename (string): Le chemin du fichier à surveiller.
//   logChan (chan<- MonitorLogEntry): Canal pour envoyer les entrées de `tracker.log`.
//   eventChan (chan<- MonitorEventEntry): Canal pour envoyer les entrées de `tracker.events`.
func monitorFile(filename string, logChan chan<- MonitorLogEntry, eventChan chan<- MonitorEventEntry) {
	var file *os.File
	var err error
	var currentPos int64

	// Attendre que le fichier existe
	for {
		file, err = os.Open(filename)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	for {
		// Vérifier si le fichier existe encore
		stat, err := os.Stat(filename)
		if err != nil {
			// Fichier supprimé, attendre qu'il soit recréé
			file.Close()
			for {
				time.Sleep(1 * time.Second)
				file, err = os.Open(filename)
				if err == nil {
					currentPos = 0
					break
				}
			}
			continue
		}

		// Si le fichier a été tronqué, repartir du début
		if stat.Size() < currentPos {
			file.Close()
			file, _ = os.Open(filename)
			currentPos = 0
		}

		// Lire les nouvelles lignes
		if currentPos < stat.Size() {
			file.Seek(currentPos, 0)
			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) == "" {
					continue
				}

				if filename == "tracker.log" {
					var entry MonitorLogEntry
					if err := json.Unmarshal([]byte(line), &entry); err == nil {
						select {
						case logChan <- entry:
						default:
							// Canal plein, ignorer
						}
					}
				} else if filename == "tracker.events" {
					var entry MonitorEventEntry
					if err := json.Unmarshal([]byte(line), &entry); err == nil {
						select {
						case eventChan <- entry:
						default:
							// Canal plein, ignorer
						}
					}
				}
			}

			// Mettre à jour la position actuelle
			newPos, _ := file.Seek(0, os.SEEK_CUR)
			file.Close()
			file, _ = os.Open(filename)
			currentPos = newPos
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// processLog traite une entrée de log provenant de `tracker.log`.
// Elle met à jour l'état global des métriques de manière concurrente-sûre.
//
// Paramètres:
//   entry (MonitorLogEntry): L'entrée de log à traiter.
func processLog(entry MonitorLogEntry) {
	monitorMetrics.mu.Lock()
	defer monitorMetrics.mu.Unlock()

	// Ajouter aux logs récents
	monitorMetrics.RecentLogs = append(monitorMetrics.RecentLogs, entry)
	if len(monitorMetrics.RecentLogs) > 20 {
		monitorMetrics.RecentLogs = monitorMetrics.RecentLogs[1:]
	}

	// Compter les erreurs
	if entry.Level == "ERROR" {
		monitorMetrics.ErrorCount++
		monitorMetrics.LastErrorTime = time.Now()
	}

	// Extraire les métriques périodiques
	if entry.Message == "Métriques système périodiques" && entry.Metadata != nil {
		if msgsReceived, ok := entry.Metadata["messages_received"].(float64); ok {
			monitorMetrics.MessagesReceived = int64(msgsReceived)
		}
		if msgsProcessed, ok := entry.Metadata["messages_processed"].(float64); ok {
			monitorMetrics.MessagesProcessed = int64(msgsProcessed)
		}
		if msgsFailed, ok := entry.Metadata["messages_failed"].(float64); ok {
			monitorMetrics.MessagesFailed = int64(msgsFailed)
		}
		if mpsStr, ok := entry.Metadata["messages_per_second"].(string); ok {
			if mps, err := strconv.ParseFloat(mpsStr, 64); err == nil {
				monitorMetrics.MessagesPerSecond = append(monitorMetrics.MessagesPerSecond, mps)
				if len(monitorMetrics.MessagesPerSecond) > 50 {
					monitorMetrics.MessagesPerSecond = monitorMetrics.MessagesPerSecond[1:]
				}
				monitorMetrics.CurrentMessagesPerSec = mps
			}
		}
		if srStr, ok := entry.Metadata["success_rate_percent"].(string); ok {
			if sr, err := strconv.ParseFloat(srStr, 64); err == nil {
				monitorMetrics.SuccessRateHistory = append(monitorMetrics.SuccessRateHistory, sr)
				if len(monitorMetrics.SuccessRateHistory) > 50 {
					monitorMetrics.SuccessRateHistory = monitorMetrics.SuccessRateHistory[1:]
				}
				monitorMetrics.CurrentSuccessRate = sr
			}
		}
	}

	monitorMetrics.LastUpdateTime = time.Now()
}

// processEvent traite une entrée d'événement provenant de `tracker.events`.
// Elle met à jour l'état global des métriques de manière concurrente-sûre.
//
// Paramètres:
//   entry (MonitorEventEntry): L'événement à traiter.
func processEvent(entry MonitorEventEntry) {
	monitorMetrics.mu.Lock()
	defer monitorMetrics.mu.Unlock()

	// Ajouter aux événements récents
	monitorMetrics.RecentEvents = append(monitorMetrics.RecentEvents, entry)
	if len(monitorMetrics.RecentEvents) > 20 {
		monitorMetrics.RecentEvents = monitorMetrics.RecentEvents[1:]
	}

	// Mettre à jour les compteurs
	if entry.Deserialized {
		monitorMetrics.MessagesProcessed++
	} else {
		monitorMetrics.MessagesFailed++
		monitorMetrics.ErrorCount++
		monitorMetrics.LastErrorTime = time.Now()
	}
	monitorMetrics.MessagesReceived++

	// Recalculer les métriques en temps réel
    uptime := time.Since(monitorMetrics.StartTime)
    if uptime.Seconds() > 0 {
        monitorMetrics.CurrentMessagesPerSec = float64(monitorMetrics.MessagesReceived) / uptime.Seconds()
    }
    if monitorMetrics.MessagesReceived > 0 {
        monitorMetrics.CurrentSuccessRate = float64(monitorMetrics.MessagesProcessed) / float64(monitorMetrics.MessagesReceived) * 100
    }

	monitorMetrics.LastUpdateTime = time.Now()
}

// createMetricsTable initialise et configure le widget de tableau pour les métriques principales.
//
// Retourne:
//   (*widgets.Table): Un pointeur vers le widget de tableau configuré.
func createMetricsTable() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Métrique", "Valeur"},
		{"Messages reçus", "0"},
		{"Messages traités", "0"},
		{"Messages échoués", "0"},
		{"Débit (msg/s)", "0.00"},
		{"Taux de succès", "0.00%"},
		{"Dernière mise à jour", "-"},
	}
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	table.SetRect(0, 0, 50, 9)
	table.ColumnWidths = []int{30, 20}
	return table
}

// createHealthDashboard initialise le widget de tableau pour le tableau de bord de santé.
//
// Retourne:
//   (*widgets.Table): Un pointeur vers le widget de tableau configuré.
func createHealthDashboard() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Indicateur", "Statut"},
		{"Santé globale", "●"},
		{"Taux de succès", "●"},
		{"Débit", "●"},
		{"Erreurs", "●"},
		{"Uptime", "-"},
		{"Qualité", "-"},
	}
	table.TextStyle = ui.NewStyle(ui.ColorWhite)
	table.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	table.SetRect(50, 0, 110, 9)
	table.ColumnWidths = []int{25, 35}
	return table
}

// getHealthStatus évalue le taux de succès et retourne un statut de santé,
// un texte descriptif et une couleur correspondante.
//
// Paramètres:
//   successRate (float64): Le taux de succès en pourcentage.
//
// Retourne:
//   (HealthStatus): Le niveau de santé (Good, Warning, Critical).
//   (string): Le texte à afficher.
//   (ui.Color): La couleur pour l'affichage.
func getHealthStatus(successRate float64) (HealthStatus, string, ui.Color) {
	if successRate >= 95.0 {
		return HealthGood, "● EXCELLENT", ui.ColorGreen
	} else if successRate >= 80.0 {
		return HealthWarning, "● BON", ui.ColorYellow
	} else {
		return HealthCritical, "● CRITIQUE", ui.ColorRed
	}
}

// getThroughputStatus évalue le débit de messages et retourne un statut de santé.
//
// Paramètres:
//   mps (float64): Le nombre de messages par seconde.
//
// Retourne:
//   (HealthStatus): Le niveau de santé.
//   (string): Le texte à afficher.
//   (ui.Color): La couleur pour l'affichage.
func getThroughputStatus(mps float64) (HealthStatus, string, ui.Color) {
	if mps >= 0.3 {
		return HealthGood, "● NORMAL", ui.ColorGreen
	} else if mps >= 0.1 {
		return HealthWarning, "● FAIBLE", ui.ColorYellow
	} else {
		return HealthCritical, "● ARRÊTÉ", ui.ColorRed
	}
}

// getErrorStatus évalue le nombre d'erreurs et le temps écoulé depuis la dernière erreur.
//
// Paramètres:
//   errorCount (int64): Le nombre total d'erreurs.
//   lastErrorTime (time.Time): L'heure de la dernière erreur.
//
// Retourne:
//   (HealthStatus): Le niveau de santé.
//   (string): Le texte à afficher.
//   (ui.Color): La couleur pour l'affichage.
func getErrorStatus(errorCount int64, lastErrorTime time.Time) (HealthStatus, string, ui.Color) {
	timeSinceError := time.Since(lastErrorTime)
	if errorCount == 0 || timeSinceError > 5*time.Minute {
		return HealthGood, "● AUCUNE", ui.ColorGreen
	} else if timeSinceError > 1*time.Minute {
		return HealthWarning, "● RÉCENTES", ui.ColorYellow
	} else {
		return HealthCritical, "● ACTIVES", ui.ColorRed
	}
}

// calculateQualityScore calcule un score de qualité global (0-100) basé sur plusieurs métriques.
//
// Paramètres:
//   successRate (float64): Le taux de succès.
//   mps (float64): Le débit de messages par seconde.
//   errorCount (int64): Le nombre d'erreurs.
//   uptime (time.Duration): La durée de fonctionnement.
//
// Retourne:
//   (float64): Le score de qualité calculé.
func calculateQualityScore(successRate, mps float64, errorCount int64, uptime time.Duration) float64 {
	// Score basé sur le taux de succès (0-50 points)
	successScore := (successRate / 100.0) * 50.0

	// Score basé sur le débit (0-30 points)
	throughputScore := 0.0
	if mps >= 0.5 {
		throughputScore = 30.0
	} else if mps >= 0.3 {
		throughputScore = 25.0
	} else if mps >= 0.1 {
		throughputScore = 15.0
	} else if mps > 0 {
		throughputScore = 10.0
	}

	// Score basé sur les erreurs (0-20 points)
	errorScore := 20.0
	if errorCount > 0 {
		errorPenalty := float64(errorCount) * 2.0
		if errorPenalty > 20.0 {
			errorPenalty = 20.0
		}
		errorScore = 20.0 - errorPenalty
		if errorScore < 0 {
			errorScore = 0
		}
	}

	return successScore + throughputScore + errorScore
}

// createLogList initialise le widget de liste pour afficher les logs récents de `tracker.log`.
//
// Retourne:
//   (*widgets.List): Un pointeur vers le widget de liste configuré.
func createLogList() *widgets.List {
	list := widgets.NewList()
	list.Title = "Logs Récents (tracker.log)"
	list.Rows = []string{"En attente de logs..."}
	list.TextStyle = ui.NewStyle(ui.ColorWhite)
	list.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	list.WrapText = true
	list.SetRect(0, 9, 80, 19)
	return list
}

// createEventList initialise le widget de liste pour afficher les événements récents de `tracker.events`.
//
// Retourne:
//   (*widgets.List): Un pointeur vers le widget de liste configuré.
func createEventList() *widgets.List {
	list := widgets.NewList()
	list.Title = "Événements Récents (tracker.events)"
	list.Rows = []string{"En attente d'événements..."}
	list.TextStyle = ui.NewStyle(ui.ColorWhite)
	list.SelectedRowStyle = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	list.WrapText = true
	list.SetRect(80, 9, 160, 19)
	return list
}

// createMessagesPerSecondChart initialise le widget de graphique pour le débit de messages.
//
// Retourne:
//   (*widgets.Plot): Un pointeur vers le widget de graphique configuré.
func createMessagesPerSecondChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Débit de Messages (msg/s)"
	plot.Data = [][]float64{{}}
	plot.SetRect(0, 19, 80, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorGreen
	plot.Marker = widgets.MarkerDot
	return plot
}

// createSuccessRateChart initialise le widget de graphique pour le taux de succès.
//
// Retourne:
//   (*widgets.Plot): Un pointeur vers le widget de graphique configuré.
func createSuccessRateChart() *widgets.Plot {
	plot := widgets.NewPlot()
	plot.Title = "Taux de Succès (%)"
	plot.Data = [][]float64{{}}
	plot.SetRect(80, 19, 160, 29)
	plot.AxesColor = ui.ColorWhite
	plot.LineColors[0] = ui.ColorBlue
	plot.Marker = widgets.MarkerDot
	return plot
}

// updateUI rafraîchit tous les widgets de l'interface utilisateur avec les dernières métriques.
// Cette fonction est appelée périodiquement pour mettre à jour l'affichage.
//
// Paramètres:
//   table (*widgets.Table): Le widget du tableau des métriques.
//   healthDashboard (*widgets.Table): Le widget du tableau de bord de santé.
//   logList (*widgets.List): Le widget de la liste des logs.
//   eventList (*widgets.List): Le widget de la liste des événements.
//   mpsChart (*widgets.Plot): Le widget du graphique de débit.
//   srChart (*widgets.Plot): Le widget du graphique de taux de succès.
func updateUI(table *widgets.Table, healthDashboard *widgets.Table, logList *widgets.List, eventList *widgets.List, mpsChart *widgets.Plot, srChart *widgets.Plot) {
	monitorMetrics.mu.RLock()
	defer monitorMetrics.mu.RUnlock()

	// Mettre à jour le tableau de métriques
	table.Rows = [][]string{
		{"Métrique", "Valeur"},
		{"Messages reçus", fmt.Sprintf("%d", monitorMetrics.MessagesReceived)},
		{"Messages traités", fmt.Sprintf("%d", monitorMetrics.MessagesProcessed)},
		{"Messages échoués", fmt.Sprintf("%d", monitorMetrics.MessagesFailed)},
		{"Débit (msg/s)", fmt.Sprintf("%.2f", monitorMetrics.CurrentMessagesPerSec)},
		{"Taux de succès", fmt.Sprintf("%.2f%%", monitorMetrics.CurrentSuccessRate)},
		{"Dernière mise à jour", monitorMetrics.LastUpdateTime.Format("15:04:05")},
	}

	// Calculer les indicateurs de santé
	successStatus, successText, successColor := getHealthStatus(monitorMetrics.CurrentSuccessRate)
	throughputStatus, throughputText, throughputColor := getThroughputStatus(monitorMetrics.CurrentMessagesPerSec)
	errorStatus, errorText, errorColor := getErrorStatus(monitorMetrics.ErrorCount, monitorMetrics.LastErrorTime)

	// Déterminer la santé globale (le pire statut)
	globalStatus := successStatus
	globalText := "● EXCELLENT"
	globalColor := ui.ColorGreen
	if throughputStatus > globalStatus {
		globalStatus = throughputStatus
	}
	if errorStatus > globalStatus {
		globalStatus = errorStatus
	}

	switch globalStatus {
	case HealthWarning:
		globalText = "● ATTENTION"
		globalColor = ui.ColorYellow
	case HealthCritical:
		globalText = "● CRITIQUE"
		globalColor = ui.ColorRed
	}

	// Calculer le score de qualité
	qualityScore := calculateQualityScore(
		monitorMetrics.CurrentSuccessRate,
		monitorMetrics.CurrentMessagesPerSec,
		monitorMetrics.ErrorCount,
		monitorMetrics.Uptime,
	)

	qualityText := ""
	qualityColor := ui.ColorWhite
	if qualityScore >= 90 {
		qualityText = fmt.Sprintf("EXCELLENT (%.0f)", qualityScore)
		qualityColor = ui.ColorGreen
	} else if qualityScore >= 70 {
		qualityText = fmt.Sprintf("BON (%.0f)", qualityScore)
		qualityColor = ui.ColorYellow
	} else if qualityScore >= 50 {
		qualityText = fmt.Sprintf("MOYEN (%.0f)", qualityScore)
		qualityColor = ui.ColorYellow
	} else {
		qualityText = fmt.Sprintf("FAIBLE (%.0f)", qualityScore)
		qualityColor = ui.ColorRed
	}

	// Formater l'uptime
	uptimeStr := ""
	if monitorMetrics.Uptime.Hours() >= 1 {
		uptimeStr = fmt.Sprintf("%.1fh", monitorMetrics.Uptime.Hours())
	} else if monitorMetrics.Uptime.Minutes() >= 1 {
		uptimeStr = fmt.Sprintf("%.0fm", monitorMetrics.Uptime.Minutes())
	} else {
		uptimeStr = fmt.Sprintf("%.0fs", monitorMetrics.Uptime.Seconds())
	}

	// Mettre à jour le tableau de bord de santé
	healthDashboard.Rows = [][]string{
		{"Indicateur", "Statut"},
		{"Santé globale", globalText},
		{"Taux de succès", successText},
		{"Débit", throughputText},
		{"Erreurs", errorText},
		{"Uptime", uptimeStr},
		{"Qualité", qualityText},
	}

	// Appliquer les couleurs aux lignes du dashboard
	healthDashboard.RowStyles = make(map[int]ui.Style)
	healthDashboard.RowStyles[0] = ui.NewStyle(ui.ColorYellow, ui.ColorClear, ui.ModifierBold)
	healthDashboard.RowStyles[1] = ui.NewStyle(globalColor, ui.ColorClear, ui.ModifierBold)
	healthDashboard.RowStyles[2] = ui.NewStyle(successColor, ui.ColorClear)
	healthDashboard.RowStyles[3] = ui.NewStyle(throughputColor, ui.ColorClear)
	healthDashboard.RowStyles[4] = ui.NewStyle(errorColor, ui.ColorClear)
	healthDashboard.RowStyles[5] = ui.NewStyle(ui.ColorCyan, ui.ColorClear)
	healthDashboard.RowStyles[6] = ui.NewStyle(qualityColor, ui.ColorClear, ui.ModifierBold)

	// Mettre à jour la liste des logs
	logRows := make([]string, 0, len(monitorMetrics.RecentLogs))
	for i := len(monitorMetrics.RecentLogs) - 1; i >= 0; i-- {
		log := monitorMetrics.RecentLogs[i]
		levelColor := ""
		if log.Level == "ERROR" {
			levelColor = "🔴"
		} else {
			levelColor = "🟢"
		}
		timeStr := log.Timestamp
		if len(timeStr) > 19 {
			timeStr = timeStr[11:19] // Extraire HH:MM:SS
		}
		row := fmt.Sprintf("%s [%s] %s", levelColor, timeStr, log.Message)
		if len(row) > 75 {
			row = row[:72] + "..."
		}
		logRows = append(logRows, row)
	}
	if len(logRows) == 0 {
		logRows = []string{"En attente de logs..."}
	}
	logList.Rows = logRows

	// Mettre à jour la liste des événements
	eventRows := make([]string, 0, len(monitorMetrics.RecentEvents))
	for i := len(monitorMetrics.RecentEvents) - 1; i >= 0; i-- {
		event := monitorMetrics.RecentEvents[i]
		status := "❌"
		if event.Deserialized {
			status = "✅"
		}
		timeStr := event.Timestamp
		if len(timeStr) > 19 {
			timeStr = timeStr[11:19] // Extraire HH:MM:SS
		}
		row := fmt.Sprintf("%s [%s] Offset: %d | %s", status, timeStr, event.KafkaOffset, event.EventType)
		if len(row) > 75 {
			row = row[:72] + "..."
		}
		eventRows = append(eventRows, row)
	}
	if len(eventRows) == 0 {
		eventRows = []string{"En attente d'événements..."}
	}
	eventList.Rows = eventRows

	// Mettre à jour le graphique de débit
	if len(monitorMetrics.MessagesPerSecond) > 0 {
		mpsChart.Data = [][]float64{monitorMetrics.MessagesPerSecond}
	} else {
		mpsChart.Data = [][]float64{{0}}
	}

	// Mettre à jour le graphique de taux de succès
	if len(monitorMetrics.SuccessRateHistory) > 0 {
		srChart.Data = [][]float64{monitorMetrics.SuccessRateHistory}
	} else {
		srChart.Data = [][]float64{{0}}
	}
}

// main est le point d'entrée du programme `log_monitor`.
//
// Son cycle de vie est le suivant :
// 1. Initialise l'interface utilisateur en mode terminal.
// 2. Crée les canaux pour la communication entre les goroutines.
// 3. Lance des goroutines pour surveiller `tracker.log` et `tracker.events`.
// 4. Lance une goroutine pour traiter les logs et événements reçus sur les canaux.
// 5. Initialise tous les widgets de l'interface (tableaux, listes, graphiques).
// 6. Entre dans une boucle principale qui :
//    a. Écoute les événements de l'interface (ex: redimensionnement, 'q' pour quitter).
//    b. Met à jour périodiquement l'interface avec les nouvelles métriques.
//    c. Redessine l'interface.
// 7. À la sortie, ferme proprement l'interface utilisateur.
func main() {
	if err := ui.Init(); err != nil {
		fmt.Printf("Erreur lors de l'initialisation de l'interface: %v\n", err)
		os.Exit(1)
	}
	defer ui.Close()

	// Canaux pour les logs et événements
	logChan := make(chan MonitorLogEntry, 100)
	eventChan := make(chan MonitorEventEntry, 100)

	// Démarrer la surveillance des fichiers
	go monitorFile("tracker.log", logChan, nil)
	go monitorFile("tracker.events", nil, eventChan)

	// Traiter les logs et événements
	go func() {
		for {
			select {
			case log := <-logChan:
				processLog(log)
			case event := <-eventChan:
				processEvent(event)
			}
		}
	}()

	// Créer les widgets
	metricsTable := createMetricsTable()
	healthDashboard := createHealthDashboard()
	logList := createLogList()
	eventList := createEventList()
	mpsChart := createMessagesPerSecondChart()
	srChart := createSuccessRateChart()

	// Gérer le redimensionnement
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	monitorMetrics.StartTime = time.Now()

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				metricsTable.SetRect(0, 0, 50, 9)
				healthDashboard.SetRect(50, 0, 110, 9)
				logList.SetRect(0, 9, 80, 19)
				eventList.SetRect(80, 9, 160, 19)
				mpsChart.SetRect(0, 19, 80, 29)
				srChart.SetRect(80, 19, 160, 29)
				ui.Clear()
			}
		case <-ticker.C:
			monitorMetrics.mu.Lock()
			monitorMetrics.Uptime = time.Since(monitorMetrics.StartTime)
			monitorMetrics.mu.Unlock()
			updateUI(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)
			ui.Render(metricsTable, healthDashboard, logList, eventList, mpsChart, srChart)
		}
	}
}
