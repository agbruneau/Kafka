/*
Ce programme Go (`analyze_logs.go`) est un outil d'analyse des logs d'observabilité
inspiré de la commande `top` d'Ubuntu. Il fournit une analyse en temps réel des
fichiers de log structurés au format JSON avec un affichage interactif.

Caractéristiques :
- Affichage en temps réel sans effacer l'écran (comme `top`)
- Interactions clavier (q: quitter, r: rafraîchir, h: aide)
- Présentation compacte et structurée
- Barres de progression pour les métriques
- Mise à jour automatique toutes les 2 secondes

Il analyse :
- Les logs système (`tracker.log`) : statistiques générales, erreurs, métriques
- Les événements (`tracker.events`) : messages reçus, statistiques métier
*/

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	LOG_FILE        = "tracker.log"
	EVENTS_FILE     = "tracker.events"
	REFRESH_INTERVAL = 2 * time.Second
)

// Codes ANSI pour le contrôle du terminal
const (
	ESC            = "\033"
	ClearScreen    = ESC + "[2J"
	HomeCursor     = ESC + "[H"
	SaveCursor     = ESC + "[s"
	RestoreCursor  = ESC + "[u"
	HideCursor     = ESC + "[?25l"
	ShowCursor     = ESC + "[?25h"
	ClearLine      = ESC + "[2K"
	Bold           = ESC + "[1m"
	Reset          = ESC + "[0m"
	ColorBlue      = ESC + "[34m"
	ColorGreen     = ESC + "[32m"
	ColorRed       = ESC + "[31m"
	ColorYellow    = ESC + "[33m"
	ColorCyan      = ESC + "[36m"
	ColorMagenta   = ESC + "[35m"
	ColorWhite     = ESC + "[37m"
	BgRed          = ESC + "[41m"
	BgGreen        = ESC + "[42m"
	BgYellow       = ESC + "[43m"
	BgBlue         = ESC + "[44m"
)

// Structures pour parser les logs JSON
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

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

type Order struct {
	OrderID        string       `json:"order_id"`
	Total          float64      `json:"total"`
	Currency       string       `json:"currency"`
	PaymentMethod  string       `json:"payment_method"`
	CustomerInfo   CustomerInfo `json:"customer_info"`
	Items          []OrderItem  `json:"items"`
}

type CustomerInfo struct {
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
}

type OrderItem struct {
	ItemName   string  `json:"item_name"`
	Quantity   int     `json:"quantity"`
	TotalPrice float64 `json:"total_price"`
}

// Statistiques collectées
type Stats struct {
	TotalLogs         int
	InfoLogs          int
	ErrorLogs         int
	ShutdownErrors    int
	RealErrors        int
	TotalEvents       int
	ProcessedEvents   int
	FailedEvents      int
	LastMetrics       *LogEntry
	FirstEventTime    time.Time
	LastEventTime     time.Time
	BusinessStats     BusinessStats
	LastErrors        []LogEntry
	LastLogs          []LogEntry
	UpdateTime        time.Time
}

type BusinessStats struct {
	TotalAmount       float64
	AvgAmount         float64
	CustomerOrders    map[string]int
	ProductQuantities map[string]int
	ProductRevenue    map[string]float64
	PaymentMethods    map[string]int
}

var (
	oldTermState *term.State
	keyChan      chan byte
)

func main() {
	// Vérifier si le terminal supporte les codes ANSI
	if !isTerminal() {
		fmt.Println("❌ Ce programme nécessite un terminal compatible ANSI.")
		os.Exit(1)
	}

	// Sauvegarder l'état du terminal
	var err error
	oldTermState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("❌ Erreur lors de la configuration du terminal: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldTermState)

	// Gestion de l'interruption (Ctrl+C)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Canal pour les touches clavier
	keyChan = make(chan byte, 1)
	go readKeys()

	// Initialisation de l'affichage
	fmt.Print(ClearScreen + HomeCursor + HideCursor)
	defer fmt.Print(ShowCursor + Reset)

	// Boucle principale
	ticker := time.NewTicker(REFRESH_INTERVAL)
	defer ticker.Stop()

	stats := analyzeLogs()
	displayReport(stats)

	for {
		select {
		case <-sigchan:
			cleanup()
			return
		case key := <-keyChan:
			switch key {
			case 'q', 'Q':
				cleanup()
				return
			case 'r', 'R':
				stats = analyzeLogs()
				displayReport(stats)
			case 'h', 'H', '?':
				showHelp()
				time.Sleep(3 * time.Second)
				stats = analyzeLogs()
				displayReport(stats)
			}
		case <-ticker.C:
			stats = analyzeLogs()
			displayReport(stats)
		}
	}
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func readKeys() {
	b := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(b)
		if n > 0 && err == nil {
			select {
			case keyChan <- b[0]:
			default:
			}
		}
	}
}

func cleanup() {
	fmt.Print(ShowCursor + Reset + ClearScreen + HomeCursor)
	fmt.Println("\n👋 Arrêt de l'analyse des logs.")
}

func showHelp() {
	fmt.Print(ClearScreen + HomeCursor)
	fmt.Println(Bold + ColorCyan + "═══════════════════════════════════════════════════════" + Reset)
	fmt.Println(Bold + ColorCyan + "  AIDE - ANALYSE DES LOGS KAFKA" + Reset)
	fmt.Println(Bold + ColorCyan + "═══════════════════════════════════════════════════════" + Reset)
	fmt.Println()
	fmt.Println("  Touches disponibles :")
	fmt.Println("    " + Bold + "q" + Reset + " ou " + Bold + "Q" + Reset + "  : Quitter")
	fmt.Println("    " + Bold + "r" + Reset + " ou " + Bold + "R" + Reset + "  : Rafraîchir immédiatement")
	fmt.Println("    " + Bold + "h" + Reset + " ou " + Bold + "?" + Reset + "  : Afficher cette aide")
	fmt.Println("    " + Bold + "Ctrl+C" + Reset + " : Quitter")
	fmt.Println()
	fmt.Println(ColorYellow + "  Appuyez sur une touche pour continuer..." + Reset)
}

func displayReport(stats *Stats) {
	// Repositionner le curseur en haut
	fmt.Print(HomeCursor)

	// En-tête fixe (comme top)
	timeStr := time.Now().Format("15:04:05")
	headerWidth := 80
	padding := (headerWidth - len("KAFKA LOG ANALYZER") - len(timeStr) - 3) / 2
	header := fmt.Sprintf("%s%s%s KAFKA LOG ANALYZER - %s %s%s%s",
		Bold+ColorCyan+BgBlue,
		strings.Repeat(" ", padding),
		strings.Repeat(" ", padding),
		timeStr,
		strings.Repeat(" ", padding),
		strings.Repeat(" ", padding),
		Reset)
	fmt.Print(header)
	fmt.Print(ClearLine + "\r\n")

	// Ligne de séparation
	fmt.Println(ColorCyan + strings.Repeat("═", 80) + Reset)
	fmt.Println()

	// Statistiques principales en tableau formaté
	fmt.Printf("%s%s%-25s %-25s %-25s%s\n", Bold, ColorGreen, "LOGS", "ÉVÉNEMENTS", "PERFORMANCE", Reset)
	fmt.Println(strings.Repeat("─", 80))

	// Ligne 1: Totaux
	fmt.Printf("  Total:     %s%-15d%s  Reçus:     %s%-15d%s  Débit:     ",
		Bold, stats.TotalLogs, Reset,
		Bold, stats.TotalEvents, Reset)
	if stats.LastMetrics != nil {
		if msgPerSec, ok := stats.LastMetrics.Metadata["messages_per_second"].(string); ok {
			fmt.Printf("%s%s msg/s%s\n", Bold+ColorCyan, msgPerSec, Reset)
		} else {
			fmt.Printf("%sN/A%s\n", ColorYellow, Reset)
		}
	} else {
		fmt.Printf("%sN/A%s\n", ColorYellow, Reset)
	}

	// Ligne 2: INFO / Traités / Taux
	fmt.Printf("  INFO:      %s%-15d%s  Traités:   %s%-15d%s  Taux:      ",
		ColorGreen, stats.InfoLogs, Reset,
		ColorGreen, stats.ProcessedEvents, Reset)
	if stats.LastMetrics != nil {
		if successRate, ok := stats.LastMetrics.Metadata["success_rate_percent"].(string); ok {
			fmt.Printf("%s%s%%%s\n", Bold+ColorGreen, successRate, Reset)
		} else {
			fmt.Printf("%sN/A%s\n", ColorYellow, Reset)
		}
	} else {
		fmt.Printf("%sN/A%s\n", ColorYellow, Reset)
	}

	// Ligne 3: ERROR / Échecs
	fmt.Printf("  ERROR:     %s%-15d%s  Échecs:    %s%-15d%s\n",
		ColorRed, stats.ErrorLogs, Reset,
		ColorRed, stats.FailedEvents, Reset)

	// Barre de progression pour le taux de succès
	if stats.TotalEvents > 0 {
		successRate := float64(stats.ProcessedEvents) / float64(stats.TotalEvents) * 100
		bar := progressBar(int(successRate), 40, ColorGreen)
		fmt.Printf("  Taux de succès: %s %s%.1f%%%s\n", bar, Bold, successRate, Reset)
	}

	fmt.Println()

	// Section Statut des Erreurs
	fmt.Printf("%s%s STATUT DES ERREURS%s\n", Bold, ColorRed, Reset)
	fmt.Println(strings.Repeat("─", 80))
	if stats.ErrorLogs > 0 {
		statusColor := ColorGreen
		statusIcon := "✅"
		statusText := "Aucune erreur réelle"
		if stats.RealErrors > 0 {
			statusColor = ColorRed
			statusIcon = "❌"
			statusText = fmt.Sprintf("%d erreur(s) réelle(s)", stats.RealErrors)
		}
		fmt.Printf("  %s%s %s%s", Bold+statusColor, statusIcon, statusText, Reset)
		if stats.ShutdownErrors > 0 {
			fmt.Printf("  |  %s%d erreur(s) d'arrêt (normal)%s", ColorYellow, stats.ShutdownErrors, Reset)
		}
		fmt.Println()
	} else {
		fmt.Printf("  %s✅ Aucune erreur détectée%s\n", Bold+ColorGreen, Reset)
	}

	// Statistiques métier
	if stats.ProcessedEvents > 0 {
		fmt.Println()
		fmt.Printf("%s%s STATISTIQUES MÉTIER%s\n", Bold, ColorMagenta, Reset)
		fmt.Println(strings.Repeat("─", 80))
		fmt.Printf("  Chiffre d'affaires total: %s%10.2f EUR%s  |  Panier moyen: %s%10.2f EUR%s\n",
			Bold+ColorGreen, stats.BusinessStats.TotalAmount, Reset,
			Bold+ColorGreen, stats.BusinessStats.AvgAmount, Reset)

		// Top clients
		if len(stats.BusinessStats.CustomerOrders) > 0 {
			type customerCount struct {
				id    string
				count int
			}
			customers := make([]customerCount, 0, len(stats.BusinessStats.CustomerOrders))
			for id, count := range stats.BusinessStats.CustomerOrders {
				customers = append(customers, customerCount{id, count})
			}
			sort.Slice(customers, func(i, j int) bool {
				return customers[i].count > customers[j].count
			})

			fmt.Print("  Top 5 clients: ")
			for i, c := range customers {
				if i >= 5 {
					break
				}
				if i > 0 {
					fmt.Print("  |  ")
				}
				fmt.Printf("%s%s%s (%d)", ColorCyan, c.id, Reset, c.count)
			}
			fmt.Println()
		}
	}

	// Dernières activités
	fmt.Println()
	fmt.Printf("%s%s DERNIÈRES ACTIVITÉS%s\n", Bold, ColorYellow, Reset)
	fmt.Println(strings.Repeat("─", 80))
	if len(stats.LastLogs) > 0 {
		for i, log := range stats.LastLogs {
			if i >= 5 {
				break
			}
			levelColor := ColorGreen
			levelIcon := "✓"
			if log.Level == "ERROR" {
				levelColor = ColorRed
				levelIcon = "✗"
			}
			timeStr := log.Timestamp
			if len(timeStr) > 19 {
				timeStr = timeStr[11:19] // Extraire juste l'heure
			}
			fmt.Printf("  [%s] %s%s%s[%s]%s  %s\n",
				timeStr, levelColor, levelIcon, Reset, log.Level, Reset, log.Message)
		}
	} else {
		fmt.Printf("  %sAucune activité récente%s\n", ColorYellow, Reset)
	}

	// Ligne de commandes (comme top)
	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("%s%sTouches:%s  %sq%s:quitter  |  %sr%s:rafraîchir  |  %sh%s:aide%s\n",
		ColorYellow, Bold, Reset,
		ColorCyan, Reset,
		ColorCyan, Reset,
		ColorCyan, Reset, Reset)

	// Effacer le reste de l'écran
	fmt.Print(ClearLine)
}

func progressBar(value, width int, color string) string {
	if value > 100 {
		value = 100
	}
	if value < 0 {
		value = 0
	}
	filled := (value * width) / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return color + bar + Reset
}

func analyzeLogs() *Stats {
	stats := &Stats{
		BusinessStats: BusinessStats{
			CustomerOrders:    make(map[string]int),
			ProductQuantities: make(map[string]int),
			ProductRevenue:    make(map[string]float64),
			PaymentMethods:    make(map[string]int),
		},
		LastErrors: make([]LogEntry, 0),
		LastLogs:   make([]LogEntry, 0),
		UpdateTime: time.Now(),
	}

	// Analyser tracker.log
	if file, err := os.Open(LOG_FILE); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		logs := make([]LogEntry, 0)

		for scanner.Scan() {
			var entry LogEntry
			if json.Unmarshal(scanner.Bytes(), &entry) == nil {
				logs = append(logs, entry)
				stats.TotalLogs++

				if entry.Level == "INFO" {
					stats.InfoLogs++
				} else if entry.Level == "ERROR" {
					stats.ErrorLogs++
					if isShutdownError(entry) {
						stats.ShutdownErrors++
					} else {
						stats.RealErrors++
						stats.LastErrors = append(stats.LastErrors, entry)
					}
				}

				if entry.Message == "Métriques système périodiques" {
					stats.LastMetrics = &entry
				}
			}
		}

		if len(stats.LastErrors) > 5 {
			stats.LastErrors = stats.LastErrors[len(stats.LastErrors)-5:]
		}
		if len(logs) > 5 {
			stats.LastLogs = logs[len(logs)-5:]
		}
	}

	// Analyser tracker.events
	if file, err := os.Open(EVENTS_FILE); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		first := true

		for scanner.Scan() {
			var entry EventEntry
			if json.Unmarshal(scanner.Bytes(), &entry) == nil {
				stats.TotalEvents++

				if entry.Deserialized {
					stats.ProcessedEvents++
					parseBusinessData(entry, stats)
				} else {
					stats.FailedEvents++
				}

				if ts, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
					if first {
						stats.FirstEventTime = ts
						stats.LastEventTime = ts
						first = false
					} else {
						if ts.Before(stats.FirstEventTime) {
							stats.FirstEventTime = ts
						}
						if ts.After(stats.LastEventTime) {
							stats.LastEventTime = ts
						}
					}
				}
			}
		}
	}

	return stats
}

func isShutdownError(entry LogEntry) bool {
	errorMsg := strings.ToLower(entry.Error)
	message := strings.ToLower(entry.Message)
	return strings.Contains(errorMsg, "brokers are down") ||
		strings.Contains(message, "kafka semble être arrêté") ||
		strings.Contains(message, "arrêt du consommateur")
}

func parseBusinessData(entry EventEntry, stats *Stats) {
	if len(entry.OrderFull) == 0 {
		return
	}

	var order Order
	if json.Unmarshal(entry.OrderFull, &order) == nil {
		stats.BusinessStats.TotalAmount += order.Total

		if order.CustomerInfo.CustomerID != "" {
			stats.BusinessStats.CustomerOrders[order.CustomerInfo.CustomerID]++
		}

		for _, item := range order.Items {
			stats.BusinessStats.ProductQuantities[item.ItemName] += item.Quantity
			stats.BusinessStats.ProductRevenue[item.ItemName] += item.TotalPrice
		}

		if order.PaymentMethod != "" {
			stats.BusinessStats.PaymentMethods[order.PaymentMethod]++
		}
	}

	if stats.ProcessedEvents > 0 {
		stats.BusinessStats.AvgAmount = stats.BusinessStats.TotalAmount / float64(stats.ProcessedEvents)
	}
}
