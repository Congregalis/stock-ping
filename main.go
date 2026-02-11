package main

import (
	"fmt"
	"os"

	"github.com/congregalis/stock-ping/cmd"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "add":
		cmd.RunAdd(os.Args[2:])
	case "once":
		cmd.RunOnce(os.Args[2:])
	case "watch":
		cmd.RunWatch(os.Args[2:])
	case "dashboard", "ui":
		cmd.RunDashboard(os.Args[2:])
	case "holding":
		cmd.RunHolding(os.Args[2:])
	case "config":
		cmd.RunConfig(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("stock-ping version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("stock-ping - A simple stock monitoring CLI tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  stock-ping <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  add [options]    Quickly add a new monitoring rule")
	fmt.Println("  once <SYMBOL>    Query current price for a single stock")
	fmt.Println("  watch            Continuously monitor stocks (text mode)")
	fmt.Println("  dashboard        Interactive TUI dashboard with hot-reload")
	fmt.Println("  holding          Manage portfolio holdings (add/list/remove)")
	fmt.Println("  config           Manage monitoring rules (add/list/remove)")
	fmt.Println("  version          Show version information")
	fmt.Println("  help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  stock-ping add --symbol AAPL --price-above 200")
	fmt.Println("  stock-ping once AAPL")
	fmt.Println("  stock-ping config add --symbol AAPL --price-above 200")
	fmt.Println("  stock-ping config list")
	fmt.Println("  stock-ping watch")
	fmt.Println("  stock-ping dashboard")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Config file: ~/.stock-ping.yaml")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/congregalis/stock-ping")
}
