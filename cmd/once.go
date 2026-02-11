package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/congregalis/stock-ping/config"
	"github.com/congregalis/stock-ping/stock"
)

// RunOnce executes the once subcommand
func RunOnce(args []string) {
	fs := flag.NewFlagSet("once", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: stock-ping once <SYMBOL>\n\n")
		fmt.Fprintf(os.Stderr, "Query current price for a single stock.\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  stock-ping once AAPL\n")
	}

	fs.Parse(args)

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	symbol := fs.Arg(0)

	// Load config for API key
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Finnhub.APIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: Finnhub API key not configured.\n")
		fmt.Fprintf(os.Stderr, "Please add your API key to ~/.stock-ping.yaml:\n")
		fmt.Fprintf(os.Stderr, "  finnhub:\n")
		fmt.Fprintf(os.Stderr, "    api_key: \"your_api_key\"\n")
		os.Exit(1)
	}

	// Check if we have a rule for this symbol in config to get name and market
	name := ""
	market := ""
	if rule := cfg.GetRule(symbol); rule != nil {
		name = rule.Name
		market = rule.Market
	}

	// Create stock client and fetch quote
	client := stock.NewClient(cfg.Finnhub.APIKey)
	quote, err := client.GetQuote(symbol, market)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching quote: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(quote.FormatQuote(name))
}
