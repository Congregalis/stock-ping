package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/congregalis/stock-ping/config"
	"github.com/congregalis/stock-ping/stock"
)

// RunHolding executes the holding subcommand
func RunHolding(args []string) {
	if len(args) < 1 {
		printHoldingUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		runHoldingList(args[1:])
	case "add":
		runHoldingAdd(args[1:])
	case "remove":
		runHoldingRemove(args[1:])
	default:
		printHoldingUsage()
		os.Exit(1)
	}
}

func printHoldingUsage() {
	fmt.Fprintf(os.Stderr, "Usage: stock-ping holding <command>\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  list                 List all holdings\n")
	fmt.Fprintf(os.Stderr, "  add                  Add or update a holding\n")
	fmt.Fprintf(os.Stderr, "  remove               Remove a holding\n")
}

func runHoldingList(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Holdings) == 0 {
		fmt.Println("No holdings configured.")
		fmt.Println("Add holdings with: stock-ping holding add --symbol AAPL --quantity 100 --cost 150.50")
		return
	}

	fmt.Printf("📊 Holdings (%d)\n", len(cfg.Holdings))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var totalCost float64
	for i, h := range cfg.Holdings {
		cost := h.Quantity * h.CostPrice
		totalCost += cost
		fmt.Printf("%d. %s\n", i+1, h.Symbol)
		fmt.Printf("   • 数量: %.2f\n", h.Quantity)
		fmt.Printf("   • 成本价: $%.2f\n", h.CostPrice)
		fmt.Printf("   • 持仓成本: $%.2f\n", cost)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("总持仓成本: $%.2f\n", totalCost)
	fmt.Printf("Config file: %s\n", config.DefaultConfigPath())
}

func runHoldingAdd(args []string) {
	fs := flag.NewFlagSet("holding add", flag.ExitOnError)

	symbol := fs.String("symbol", "", "Stock symbol (required)")
	quantity := fs.Float64("quantity", 0, "Number of shares (required)")
	costPrice := fs.Float64("cost", 0, "Cost price per share (required)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: stock-ping holding add [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  stock-ping holding add --symbol AAPL --quantity 100 --cost 150.50\n")
		fmt.Fprintf(os.Stderr, "  stock-ping holding add --symbol 600519.SS --quantity 10 --cost 1500\n")
	}

	fs.Parse(args)

	if *symbol == "" {
		fmt.Fprintf(os.Stderr, "Error: --symbol is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	if *quantity <= 0 {
		fmt.Fprintf(os.Stderr, "Error: --quantity must be greater than 0\n\n")
		fs.Usage()
		os.Exit(1)
	}

	if *costPrice <= 0 {
		fmt.Fprintf(os.Stderr, "Error: --cost must be greater than 0\n\n")
		fs.Usage()
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check if holding exists for "add position" logic
	if existing := cfg.GetHolding(*symbol); existing != nil {
		currentTotalCost := existing.Quantity * existing.CostPrice
		newAddCost := *quantity * *costPrice
		newTotalQuantity := existing.Quantity + *quantity
		newAvgCost := (currentTotalCost + newAddCost) / newTotalQuantity

		fmt.Printf("ℹ️  Existing holding found: %.2f shares @ $%.2f\n", existing.Quantity, existing.CostPrice)
		fmt.Printf("   Adding: %.2f shares @ $%.2f\n", *quantity, *costPrice)

		// Update variables to new total values
		*quantity = newTotalQuantity
		*costPrice = newAvgCost

		fmt.Printf("   New Position: %.2f shares @ $%.2f\n", *quantity, *costPrice)
	}

	// Create holding
	holding := config.Holding{
		Symbol:    *symbol,
		Quantity:  *quantity,
		CostPrice: *costPrice,
	}

	// Add holding
	cfg.AddHolding(holding)

	// Check if rule exists, if not create it
	if cfg.GetRule(*symbol) == nil {
		fmt.Printf("ℹ️  No monitoring rule found for %s. Creating one...\n", *symbol)

		name, market, err := stock.FetchSymbolDetails(*symbol)
		if err != nil {
			fmt.Printf("⚠️  Failed to fetch symbol details: %v. Using defaults.\n", err)
			name = *symbol
			market = stock.MarketUS
		} else {
			fmt.Printf("✅ Found details: %s (%s)\n", name, market)
		}

		newRule := config.Rule{
			Symbol: *symbol,
			Name:   name,
			Market: market,
		}
		if err := cfg.AddRule(newRule); err != nil {
			fmt.Printf("⚠️  Failed to add rule: %v\n", err)
		}
	}

	// Save config
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	totalCost := *quantity * *costPrice
	fmt.Printf("✅ Added holding for %s: %.2f shares @ $%.2f (total cost: $%.2f)\n",
		*symbol, *quantity, *costPrice, totalCost)
}

func runHoldingRemove(args []string) {
	fs := flag.NewFlagSet("holding remove", flag.ExitOnError)

	symbol := fs.String("symbol", "", "Stock symbol to remove (required)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: stock-ping holding remove --symbol <SYMBOL>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	if *symbol == "" {
		fmt.Fprintf(os.Stderr, "Error: --symbol is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Remove holding
	if !cfg.RemoveHolding(*symbol) {
		fmt.Fprintf(os.Stderr, "Holding for %s not found\n", *symbol)
		os.Exit(1)
	}

	// Save config
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Removed holding for %s\n", *symbol)
}
