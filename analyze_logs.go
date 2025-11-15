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

func drawBox(content []string, title string, width int, color string) {
	// Top border
	fmt.Printf("%s┌─ %s %s %s─┐%s\n", color, Bold, title, strings.Repeat("─", width-len(title)-5), Reset)

	// Content
	for _, line := range content {
		// Ensure line does not exceed width
		if len(line) > width-2 {
			line = line[:width-5] + "..."
		}
		fmt.Printf("%s│ %s%s%s\n", color, Reset, line, strings.Repeat(" ", width-len(line)-1))
	}

	// Bottom border
	fmt.Printf("%s└%s┘%s\n", color, strings.Repeat("─", width), Reset)
}

func displayReport(stats *Stats) {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width < 80 {
		width = 80
	}
	if width > 120 {
		width = 120
	}

	fmt.Print(HomeCursor)
	// Header
	timeStr := time.Now().Format("15:04:05")
	headerTitle := "ANALYSEUR DE LOGS KAFKA"
	headerPadding := (width - len(headerTitle) - len(timeStr) - 3)
	fmt.Printf("%s%s%s %s %s%s\n", BgBlue+Bold, strings.Repeat(" ", 2), headerTitle, timeStr, strings.Repeat(" ", headerPadding), Reset)

	// Section 1: Stats générales et Erreurs (colonne de gauche)
	// Panneau 1: Statistiques générales
	var generalStatsContent []string
	logLine := fmt.Sprintf("Logs:    %s%d%s Total, %s%d%s Info, %s%d%s Erreurs",
		Bold, stats.TotalLogs, Reset, ColorGreen, stats.InfoLogs, Reset, ColorRed, stats.ErrorLogs, Reset)
	eventLine := fmt.Sprintf("Événements: %s%d%s Reçus, %s%d%s Traités, %s%d%s Échecs",
		Bold, stats.TotalEvents, Reset, ColorGreen, stats.ProcessedEvents, Reset, ColorRed, stats.FailedEvents, Reset)
	generalStatsContent = append(generalStatsContent, logLine, eventLine)
	if stats.LastMetrics != nil {
		if msgPerSec, ok := stats.LastMetrics.Metadata["messages_per_second"].(string); ok {
			if successRate, ok2 := stats.LastMetrics.Metadata["success_rate_percent"].(string); ok2 {
				perfLine := fmt.Sprintf("Perf:    %s%s%s msg/s, %s%s%%%s Taux de succès",
					Bold+ColorCyan, msgPerSec, Reset, Bold+ColorGreen, successRate, Reset)
				generalStatsContent = append(generalStatsContent, perfLine)
			}
		}
	}
	if stats.TotalEvents > 0 {
		successRate := float64(stats.ProcessedEvents) / float64(stats.TotalEvents) * 100
		bar := progressBar(int(successRate), width/2-10, ColorGreen)
		generalStatsContent = append(generalStatsContent, fmt.Sprintf("Taux:    %s %s%.1f%%%s", bar, Bold, successRate, Reset))
	}
	drawBox(generalStatsContent, "Statistiques Générales", width/2, ColorCyan)

	// Panneau 2: État des erreurs
	var errorStatusContent []string
	if stats.ErrorLogs > 0 {
		statusColor := ColorGreen
		statusIcon := "✅"
		statusText := "Aucune erreur réelle"
		if stats.RealErrors > 0 {
			statusColor = ColorRed
			statusIcon = "❌"
			statusText = fmt.Sprintf("%d erreur(s) réelle(s)", stats.RealErrors)
		}
		errorStatusContent = append(errorStatusContent, fmt.Sprintf("%s%s %s%s", Bold+statusColor, statusIcon, statusText, Reset))
		if stats.ShutdownErrors > 0 {
			errorStatusContent = append(errorStatusContent, fmt.Sprintf("  ↳ %s%d erreur(s) d'arrêt (normal)%s", ColorYellow, stats.ShutdownErrors, Reset))
		}
	} else {
		errorStatusContent = append(errorStatusContent, fmt.Sprintf("%s✅ Aucune erreur détectée%s", Bold+ColorGreen, Reset))
	}
	drawBox(errorStatusContent, "État des Erreurs", width/2, ColorRed)

	// Panneau 3: Dernières activités
	var recentActivityContent []string
	if len(stats.LastLogs) > 0 {
		for _, log := range stats.LastLogs {
			levelColor := ColorGreen
			levelIcon := "✓"
			if log.Level == "ERROR" {
				levelColor = ColorRed
				levelIcon = "✗"
			}
			timeStr := log.Timestamp
			if len(timeStr) > 19 {
				timeStr = timeStr[11:19]
			}
			recentActivityContent = append(recentActivityContent, fmt.Sprintf("[%s] %s%s%s[%s]%s %s",
				timeStr, levelColor, levelIcon, Reset, log.Level, Reset, log.Message))
		}
	} else {
		recentActivityContent = append(recentActivityContent, fmt.Sprintf("%sAucune activité récente%s", ColorYellow, Reset))
	}
	drawBox(recentActivityContent, "Dernières Activités", width, ColorYellow)

	// Section 2: Métier (colonne de droite)
	// Move cursor to the right column
	fmt.Printf(ESC+"[%d;%dH", 2, width/2+2)

	var businessStatsContent []string
	if stats.ProcessedEvents > 0 {
		businessStatsContent = append(businessStatsContent, fmt.Sprintf("Chiffre d'affaires: %s%.2f EUR%s | Panier moyen: %s%.2f EUR%s",
			Bold+ColorGreen, stats.BusinessStats.TotalAmount, Reset, Bold+ColorGreen, stats.BusinessStats.AvgAmount, Reset))

		// Top 5 Products by Quantity
		type productCount struct {
			name  string
			count int
		}
		products := make([]productCount, 0, len(stats.BusinessStats.ProductQuantities))
		for name, count := range stats.BusinessStats.ProductQuantities {
			products = append(products, productCount{name, count})
		}
		sort.Slice(products, func(i, j int) bool { return products[i].count > products[j].count })

		var topProducts string
		for i, p := range products {
			if i >= 5 {
				break
			}
			topProducts += fmt.Sprintf("%s%s%s(%d) ", ColorCyan, p.name, Reset, p.count)
		}
		businessStatsContent = append(businessStatsContent, fmt.Sprintf("Top 5 Produits (Qt): %s", topProducts))

		// Top 5 Products by Revenue
		type productRevenue struct {
			name    string
			revenue float64
		}
		productsRev := make([]productRevenue, 0, len(stats.BusinessStats.ProductRevenue))
		for name, rev := range stats.BusinessStats.ProductRevenue {
			productsRev = append(productsRev, productRevenue{name, rev})
		}
		sort.Slice(productsRev, func(i, j int) bool { return productsRev[i].revenue > productsRev[j].revenue })
		var topProductsRev string
		for i, p := range productsRev {
			if i >= 5 {
				break
			}
			topProductsRev += fmt.Sprintf("%s%s%s(%.2f) ", ColorMagenta, p.name, Reset, p.revenue)
		}
		businessStatsContent = append(businessStatsContent, fmt.Sprintf("Top 5 Produits (CA): %s", topProductsRev))

		// Payment Methods
		var paymentMethods string
		for method, count := range stats.BusinessStats.PaymentMethods {
			paymentMethods += fmt.Sprintf("%s%s%s(%d) ", ColorYellow, method, Reset, count)
		}
		businessStatsContent = append(businessStatsContent, fmt.Sprintf("Méthodes Paiement: %s", paymentMethods))
	}
	drawBox(businessStatsContent, "Statistiques Métier", width/2-2, ColorMagenta)

	// Footer
	fmt.Printf(ESC+"[%d;%dH", 25, 0)
	fmt.Println(strings.Repeat("═", width))
	fmt.Printf("%s%sTouches:%s %sq%s:quitter | %sr%s:rafraîchir | %sh%s:aide%s\n",
		ColorYellow, Bold, Reset,
		ColorCyan, Reset,
		ColorCyan, Reset,
		ColorCyan, Reset, Reset)

	// Clear the rest of the screen
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
