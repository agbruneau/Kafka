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
	"regexp"
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

func stripAnsi(str string) string {
	const ansi = "[\u001B\u009B][[\\]()#;?]*.?[a-zA-Z0-9]"
	var re = regexp.MustCompile(ansi)
	return re.ReplaceAllString(str, "")
}

// formatBox formats content into a box with a title and border, returning it as a slice of strings.
func formatBox(content []string, title string, width int, color string) []string {
	var box []string
	// Top border
	box = append(box, fmt.Sprintf("%s┌─ %s %s %s─┐%s", color, Bold, title, strings.Repeat("─", width-len(title)-5), Reset))

	// Content
	for _, line := range content {
		paddedLine := line + strings.Repeat(" ", width-len(stripAnsi(line))-1)
		box = append(box, fmt.Sprintf("%s│ %s%s%s", color, Reset, paddedLine, Reset))
	}

	// Bottom border
	box = append(box, fmt.Sprintf("%s└%s┘%s", color, strings.Repeat("─", width), Reset))
	return box
}

func displayReport(stats *Stats) {
	width := 80
	height := 24
	fmt.Print(HomeCursor)

	// Header
	timeStr := time.Now().Format("15:04:05")
	headerTitle := "ANALYSEUR DE LOGS KAFKA"
	headerPadding := (width - len(headerTitle) - len(timeStr) - 3)
	header := fmt.Sprintf("%s%s%s %s %s%s", BgBlue+Bold, strings.Repeat(" ", 2), headerTitle, timeStr, strings.Repeat(" ", headerPadding), Reset)
	fmt.Println(header)

	// Prepare content for columns
	colWidth1 := 45
	colWidth2 := width - colWidth1 - 1

	// --- Column 1: Health & Events ---
	var healthContent []string
	healthContent = append(healthContent, fmt.Sprintf("Logs: %s%d%s Total, %s%d%s Info, %s%d%s Err",
		Bold, stats.TotalLogs, Reset, ColorGreen, stats.InfoLogs, Reset, ColorRed, stats.ErrorLogs, Reset))
	if stats.ErrorLogs > 0 {
		statusText := fmt.Sprintf("%s%d erreurs réelles%s", ColorRed, stats.RealErrors, Reset)
		if stats.RealErrors == 0 {
			statusText = fmt.Sprintf("%sAucune erreur réelle%s", ColorGreen, Reset)
		}
		healthContent = append(healthContent, fmt.Sprintf("Statut Err: %s | %s%d arrêt%s",
			statusText, ColorYellow, stats.ShutdownErrors, Reset))
	} else {
		healthContent = append(healthContent, fmt.Sprintf("Statut Err: %s✅ Aucune erreur%s", Bold+ColorGreen, Reset))
	}
	healthContent = append(healthContent, "") // Spacer
	healthContent = append(healthContent, fmt.Sprintf("Événements: %s%d%s Reçus", Bold, stats.TotalEvents, Reset))
	healthContent = append(healthContent, fmt.Sprintf("  %s%d%s Traités, %s%d%s Échecs",
		ColorGreen, stats.ProcessedEvents, Reset, ColorRed, stats.FailedEvents, Reset))
	if stats.TotalEvents > 0 {
		successRate := float64(stats.ProcessedEvents) / float64(stats.TotalEvents) * 100
		bar := progressBar(int(successRate), colWidth1-18, ColorGreen)
		healthContent = append(healthContent, fmt.Sprintf("Taux Succès: %s %s%.1f%%%s", bar, Bold, successRate, Reset))
	}
	healthBox := formatBox(healthContent, "Santé et Événements", colWidth1-2, ColorCyan)

	// --- Column 2: Business Stats ---
	var businessContent []string
	if stats.ProcessedEvents > 0 {
		businessContent = append(businessContent, fmt.Sprintf("CA Total: %s%.2f EUR%s", Bold+ColorGreen, stats.BusinessStats.TotalAmount, Reset))
		businessContent = append(businessContent, fmt.Sprintf("Panier Moyen: %s%.2f EUR%s", Bold, stats.BusinessStats.AvgAmount, Reset))
		businessContent = append(businessContent, "")

		// Top 2 Products by Quantity
		type productCount struct {
			name  string
			count int
		}
		products := make([]productCount, 0, len(stats.BusinessStats.ProductQuantities))
		for name, count := range stats.BusinessStats.ProductQuantities {
			products = append(products, productCount{name, count})
		}
		sort.Slice(products, func(i, j int) bool { return products[i].count > products[j].count })
		businessContent = append(businessContent, Bold+"Top Produits (Qt):"+Reset)
		for i, p := range products {
			if i >= 2 { // Keep it short for the smaller panel
				break
			}
			businessContent = append(businessContent, fmt.Sprintf(" %s- %s: %d%s", ColorCyan, p.name, p.count, Reset))
		}
		businessContent = append(businessContent, "") // Spacer

		// Payment Methods
		businessContent = append(businessContent, Bold+"Méthodes Paiement:"+Reset)
		for method, count := range stats.BusinessStats.PaymentMethods {
			businessContent = append(businessContent, fmt.Sprintf(" %s- %s: %d%s", ColorYellow, method, count, Reset))
		}
	} else {
		businessContent = append(businessContent, "Aucune donnée métier.")
	}
	businessBox := formatBox(businessContent, "Statistiques Métier", colWidth2-2, ColorMagenta)

	// --- Draw Columns ---
	maxBoxHeight := len(healthBox)
	if len(businessBox) > maxBoxHeight {
		maxBoxHeight = len(businessBox)
	}

	for i := 0; i < maxBoxHeight; i++ {
		line1 := ""
		if i < len(healthBox) {
			line1 = healthBox[i]
		} else {
			line1 = strings.Repeat(" ", colWidth1)
		}

		line2 := ""
		if i < len(businessBox) {
			line2 = businessBox[i]
		} else {
			line2 = strings.Repeat(" ", colWidth2)
		}
		fmt.Printf("%s %s\n", line1, line2)
	}

	// --- Bottom Panel: Activity & Errors ---
	var activityContent []string
	if len(stats.LastLogs) > 0 {
		// Last 5 logs
		for _, log := range stats.LastLogs {
			levelColor := ColorGreen
			if log.Level == "ERROR" {
				levelColor = ColorRed
			}
			timeStr := log.Timestamp[11:19]
			msg := fmt.Sprintf("[%s] %s[%s]%s %s", timeStr, levelColor, log.Level, Reset, log.Message)
			if len(stripAnsi(msg)) > width-4 {
				msg = msg[:width-7] + "..."
			}
			activityContent = append(activityContent, msg)
		}
	} else {
		activityContent = append(activityContent, fmt.Sprintf("%sAucune activité récente%s", ColorYellow, Reset))
	}
	activityBox := formatBox(activityContent, "Dernières Activités", width-2, ColorYellow)

	for _, line := range activityBox {
		fmt.Println(line)
	}

	// --- Footer & Screen Clearing ---
	// Clear remaining lines
	linesDrawn := 1 + maxBoxHeight + len(activityBox)
	for i := linesDrawn; i < height-2; i++ {
		fmt.Println(ClearLine)
	}

	fmt.Println(strings.Repeat("═", width))
	fmt.Printf("%s%sTouches:%s %sq%s:quitter | %sr%s:rafraîchir | %sh%s:aide%s\n",
		ColorYellow, Bold, Reset,
		ColorCyan, Reset,
		ColorCyan, Reset,
		ColorCyan, Reset, Reset)
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
