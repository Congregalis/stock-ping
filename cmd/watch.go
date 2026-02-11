package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/congregalis/stock-ping/config"
	"github.com/congregalis/stock-ping/notify"
	"github.com/congregalis/stock-ping/rule"
	"github.com/congregalis/stock-ping/stock"
)

// triggeredState tracks which conditions have been triggered for each symbol
// map[symbol]map[conditionKey]bool - true means condition was triggered in last check
var triggeredState = make(map[string]map[string]bool)

// US Eastern timezone for market hours check
var easternTZ *time.Location

func init() {
	var err error
	easternTZ, err = time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback: use UTC-5 as approximation (doesn't handle DST)
		easternTZ = time.FixedZone("EST", -5*60*60)
	}
}

// isMarketOpen checks if US stock market is currently open
// Regular trading hours: Mon-Fri 9:30 AM - 4:00 PM Eastern Time
func isMarketOpen() bool {
	now := time.Now().In(easternTZ)
	weekday := now.Weekday()

	// Weekend check
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}

	// Time check: 9:30 - 16:00 ET
	hour, min, _ := now.Clock()
	timeInMinutes := hour*60 + min
	marketOpen := 9*60 + 30 // 9:30 AM
	marketClose := 16 * 60  // 4:00 PM

	return timeInMinutes >= marketOpen && timeInMinutes < marketClose
}

// getNextMarketOpen returns the next market open time
func getNextMarketOpen() time.Time {
	now := time.Now().In(easternTZ)
	// Start with today at 9:30 AM ET
	nextOpen := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, easternTZ)

	// If already past 9:30 today, move to tomorrow
	if now.After(nextOpen) {
		nextOpen = nextOpen.Add(24 * time.Hour)
	}

	// Skip weekends
	for nextOpen.Weekday() == time.Saturday || nextOpen.Weekday() == time.Sunday {
		nextOpen = nextOpen.Add(24 * time.Hour)
	}

	return nextOpen
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

// RunWatch executes the watch subcommand
func RunWatch(args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: stock-ping watch\n\n")
		fmt.Fprintf(os.Stderr, "Continuously monitor stocks based on configured rules.\n")
		fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop.\n")
	}

	fs.Parse(args)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Finnhub.APIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: Finnhub API key not configured.\n")
		fmt.Fprintf(os.Stderr, "Please add your API key to ~/.stock-ping.yaml\n")
		os.Exit(1)
	}

	if len(cfg.Rules) == 0 {
		fmt.Fprintf(os.Stderr, "No monitoring rules configured.\n")
		fmt.Fprintf(os.Stderr, "Add rules with: stock-ping config add --symbol AAPL --price-above 200\n")
		os.Exit(1)
	}

	// Create clients
	stockClient := stock.NewClient(cfg.Finnhub.APIKey)
	notifier := notify.NewNotifier(cfg.Bark.ServerURL, cfg.Bark.Key)
	evaluator := rule.NewEvaluator()

	// Print startup message
	fmt.Printf("🔔 Stock Monitor Started (interval: %ds)\n", cfg.Interval)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !notifier.IsConfigured() {
		fmt.Println("⚠️  Warning: Bark not configured, notifications disabled")
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run first check immediately (regardless of market status)
	checkRules(cfg, stockClient, notifier, evaluator)

	// Check if market is currently open
	if !isMarketOpen() {
		nextOpen := getNextMarketOpen()
		waitDuration := time.Until(nextOpen)
		fmt.Printf("\n💤 美股休市中，将在 %s 后自动恢复监控\n", formatDuration(waitDuration))
		fmt.Printf("   下次开盘时间: %s (美东时间)\n", nextOpen.Format("01-02 15:04 Mon"))
		fmt.Println("   程序将继续运行，等待开盘...")

		// Wait for market to open
		waitTimer := time.NewTimer(waitDuration)
		select {
		case <-waitTimer.C:
			fmt.Println("\n🔔 美股开盘，恢复监控!")
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		case <-sigChan:
			waitTimer.Stop()
			fmt.Println("\n👋 Shutting down...")
			return
		}
	}

	// Create ticker for periodic checks
	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	// Main loop
	for {
		select {
		case <-ticker.C:
			// Check if market closed during monitoring
			if !isMarketOpen() {
				nextOpen := getNextMarketOpen()
				waitDuration := time.Until(nextOpen)
				fmt.Printf("\n💤 美股收盘，将在 %s 后自动恢复监控\n", formatDuration(waitDuration))
				fmt.Printf("   下次开盘时间: %s (美东时间)\n", nextOpen.Format("01-02 15:04 Mon"))

				// Stop current ticker and wait for market open
				ticker.Stop()
				waitTimer := time.NewTimer(waitDuration)
				select {
				case <-waitTimer.C:
					fmt.Println("\n🔔 美股开盘，恢复监控!")
					fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
					ticker.Reset(time.Duration(cfg.Interval) * time.Second)
				case <-sigChan:
					waitTimer.Stop()
					fmt.Println("\n👋 Shutting down...")
					return
				}
			}
			checkRules(cfg, stockClient, notifier, evaluator)
		case <-sigChan:
			fmt.Println("\n👋 Shutting down...")
			return
		}
	}
}

func checkRules(cfg *config.Config, stockClient *stock.Client, notifier *notify.Notifier, evaluator *rule.Evaluator) {
	now := time.Now().Format("15:04:05")
	fmt.Printf("\n[%s] Checking %d rules...\n", now, len(cfg.Rules))

	for _, r := range cfg.Rules {
		ruleCopy := r // Create a copy for the evaluator

		quote, err := stockClient.GetQuote(r.Symbol, r.Market)
		if err != nil {
			fmt.Printf("  %s ❌ Error: %v\n", r.Symbol, err)
			continue
		}

		result := evaluator.Evaluate(&ruleCopy, quote)

		// Get current triggered conditions as a map
		currentConditions := make(map[string]bool)
		for _, reason := range result.Reasons {
			currentConditions[reason] = true
		}

		// Initialize state for this symbol if not exists
		if triggeredState[r.Symbol] == nil {
			triggeredState[r.Symbol] = make(map[string]bool)
		}

		// Find newly triggered conditions (edge detection)
		var newlyTriggered []string
		for reason := range currentConditions {
			if !triggeredState[r.Symbol][reason] {
				newlyTriggered = append(newlyTriggered, reason)
			}
		}

		// Update state: set current conditions and clear conditions no longer met
		triggeredState[r.Symbol] = currentConditions

		// Format the status line
		changeSign := ""
		if quote.PercentChange >= 0 {
			changeSign = "+"
		}
		status := "✓"
		if result.Triggered() {
			if len(newlyTriggered) > 0 {
				status = "🔔" // New trigger
			} else {
				status = "⚠️" // Still triggered but already notified
			}
		}

		displayName := r.Symbol
		if r.Name != "" {
			displayName = fmt.Sprintf("%s(%s)", r.Symbol, r.Name)
		}

		fmt.Printf("  %s $%.2f (%s%.2f%%) %s\n",
			displayName, quote.CurrentPrice, changeSign, quote.PercentChange, status)

		// Print trigger reasons
		if result.Triggered() {
			for _, reason := range result.Reasons {
				prefix := "→"
				for _, nr := range newlyTriggered {
					if nr == reason {
						prefix = "🆕"
						break
					}
				}
				fmt.Printf("     %s %s\n", prefix, reason)
			}
		}

		// Only send notification for newly triggered conditions
		if len(newlyTriggered) > 0 && notifier.IsConfigured() {
			title, body := result.FormatNotification()
			if err := notifier.SendWithGroup(title, body, "stock-ping"); err != nil {
				fmt.Printf("     ❌ Failed to send notification: %v\n", err)
			} else {
				fmt.Println("     📱 Bark notification sent")
			}
		}

		// Small delay between API calls to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}
}
