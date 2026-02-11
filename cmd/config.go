package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/congregalis/stock-ping/config"
)

// RunConfig executes the config subcommand
func RunConfig(args []string) {
	if len(args) < 1 {
		printConfigUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		runConfigList(args[1:])
	case "add":
		runConfigAdd(args[1:])
	case "remove":
		runConfigRemove(args[1:])
	default:
		printConfigUsage()
		os.Exit(1)
	}
}

func printConfigUsage() {
	fmt.Fprintf(os.Stderr, "Usage: stock-ping config <command>\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  list                 List all monitoring rules\n")
	fmt.Fprintf(os.Stderr, "  add                  Add a new monitoring rule\n")
	fmt.Fprintf(os.Stderr, "  remove               Remove a monitoring rule\n")
}

func runConfigList(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.Rules) == 0 {
		fmt.Println("No monitoring rules configured.")
		fmt.Println("Add rules with: stock-ping config add --symbol AAPL --price-above 200")
		return
	}

	fmt.Printf("📋 Monitoring Rules (%d)\n", len(cfg.Rules))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, r := range cfg.Rules {
		displayName := r.Symbol
		if r.Name != "" {
			displayName = fmt.Sprintf("%s (%s)", r.Symbol, r.Name)
		}
		fmt.Printf("%d. %s\n", i+1, displayName)

		if r.PriceAbove != nil {
			fmt.Printf("   • 价格高于 $%.2f\n", *r.PriceAbove)
		}
		if r.PriceBelow != nil {
			fmt.Printf("   • 价格低于 $%.2f\n", *r.PriceBelow)
		}
		if r.ChangeAbove != nil {
			fmt.Printf("   • 涨幅超过 %.2f%%\n", *r.ChangeAbove)
		}
		if r.ChangeBelow != nil {
			fmt.Printf("   • 跌幅超过 %.2f%%\n", *r.ChangeBelow)
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Config file: %s\n", config.DefaultConfigPath())
}

func runConfigAdd(args []string) {
	fs := flag.NewFlagSet("config add", flag.ExitOnError)

	symbol := fs.String("symbol", "", "Stock symbol (required)")
	market := fs.String("market", "US", "Market type (US, CN, TW, CRYPTO, FOREX)")
	name := fs.String("name", "", "Display name (optional)")
	priceAbove := fs.Float64("price-above", 0, "Alert when price is above this value")
	priceBelow := fs.Float64("price-below", 0, "Alert when price is below this value")
	changeAbove := fs.Float64("change-above", 0, "Alert when percent change is above this value")
	changeBelow := fs.Float64("change-below", 0, "Alert when percent change is below this value")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: stock-ping config add [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  stock-ping config add --symbol AAPL --price-above 200\n")
		fmt.Fprintf(os.Stderr, "  stock-ping config add --symbol 600519.SS --market CN --name 茅台 --price-below 1400\n")
	}

	fs.Parse(args)

	if *symbol == "" {
		fmt.Fprintf(os.Stderr, "Error: --symbol is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	// Check that at least one condition is set
	// if *priceAbove == 0 && *priceBelow == 0 && *changeAbove == 0 && *changeBelow == 0 {
	// 	fmt.Fprintf(os.Stderr, "Error: At least one condition is required\n")
	// 	fmt.Fprintf(os.Stderr, "Use --price-above, --price-below, --change-above, or --change-below\n")
	// 	os.Exit(1)
	// }

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create rule
	rule := config.Rule{
		Symbol: *symbol,
		Market: *market,
		Name:   *name,
	}

	if *priceAbove != 0 {
		rule.PriceAbove = priceAbove
	}
	if *priceBelow != 0 {
		rule.PriceBelow = priceBelow
	}
	if *changeAbove != 0 {
		rule.ChangeAbove = changeAbove
	}
	if *changeBelow != 0 {
		rule.ChangeBelow = changeBelow
	}

	// Add rule
	cfg.AddRule(rule)

	// Save config
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Added rule for %s\n", *symbol)
}

func runConfigRemove(args []string) {
	fs := flag.NewFlagSet("config remove", flag.ExitOnError)

	symbol := fs.String("symbol", "", "Stock symbol to remove (required)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: stock-ping config remove --symbol <SYMBOL>\n\n")
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

	// Remove rule
	if !cfg.RemoveRule(*symbol) {
		fmt.Fprintf(os.Stderr, "Rule for %s not found\n", *symbol)
		os.Exit(1)
	}

	// Save config
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Removed rule for %s\n", *symbol)
}

// RunAdd is a top-level alias for RunConfigAdd
func RunAdd(args []string) {
	runConfigAdd(args)
}
