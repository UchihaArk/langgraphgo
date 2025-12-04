package main

import (
	"context"
	"fmt"
	"log"
	"os"

	trading "github.com/smallnest/langgraphgo/showcases/trading_agents"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create configuration
	config := trading.DefaultConfig()
	config.APIKey = apiKey
	config.AlphaVantageKey = os.Getenv("ALPHA_VANTAGE_API_KEY")
	config.Verbose = true

	// Create trading agents graph
	fmt.Println("🤖 Initializing Trading Agents...")
	graph, err := trading.NewTradingAgentsGraph(config)
	if err != nil {
		log.Fatalf("Failed to create trading agents: %v", err)
	}

	// Analyze a stock
	symbol := "AAPL"
	if len(os.Args) > 1 {
		symbol = os.Args[1]
	}

	fmt.Printf("\n📊 Analyzing %s...\n\n", symbol)

	request := trading.AnalysisRequest{
		Symbol:        symbol,
		Capital:       10000,
		RiskTolerance: "moderate",
		Timeframe:     "1D",
	}

	// Execute analysis
	ctx := context.Background()
	result, err := graph.Analyze(ctx, request)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	// Print results
	printResults(result)
}

func printResults(result *trading.AnalysisResponse) {
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║      TRADING RECOMMENDATION              ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\nSymbol:          %s\n", result.Symbol)
	fmt.Printf("Recommendation:  %s\n", result.Recommendation)
	fmt.Printf("Confidence:      %.1f%%\n", result.Confidence)
	fmt.Printf("Risk Score:      %.1f/100\n\n", result.RiskScore)

	if result.PositionSize > 0 {
		currentPrice := result.Metadata["current_price"].(float64)
		fmt.Println("Trade Details:")
		fmt.Printf("  Position Size:   %.2f shares\n", result.PositionSize)
		fmt.Printf("  Entry Price:     $%.2f\n", currentPrice)
		fmt.Printf("  Stop Loss:       $%.2f\n", result.StopLoss)
		fmt.Printf("  Take Profit:     $%.2f\n", result.TakeProfit)

		investment := result.PositionSize * currentPrice
		potentialProfit := result.PositionSize * (result.TakeProfit - currentPrice)
		potentialLoss := result.PositionSize * (currentPrice - result.StopLoss)

		fmt.Printf("\n  Investment:      $%.2f\n", investment)
		fmt.Printf("  Potential Gain:  $%.2f (+%.1f%%)\n", potentialProfit, 100*potentialProfit/investment)
		fmt.Printf("  Potential Loss:  $%.2f (%.1f%%)\n", potentialLoss, -100*potentialLoss/investment)
		fmt.Printf("  Risk/Reward:     1:%.2f\n", potentialProfit/potentialLoss)
	}

	fmt.Println("\n╔══════════════════════════════════════════╗")
	fmt.Println("║      REASONING                           ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\n%s\n\n", result.Reasoning)

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║      AGENT REPORTS (SUMMARY)             ║")
	fmt.Println("╚══════════════════════════════════════════╝")

	reports := []struct {
		name   string
		icon   string
		report string
	}{
		{"Fundamentals", "📊", result.Reports["fundamentals"]},
		{"Sentiment", "💭", result.Reports["sentiment"]},
		{"Technical", "📉", result.Reports["technical"]},
		{"Bullish View", "🐂", result.Reports["bullish"]},
		{"Bearish View", "🐻", result.Reports["bearish"]},
		{"Risk Assessment", "⚠️", result.Reports["risk"]},
	}

	for _, r := range reports {
		fmt.Printf("\n%s %s:\n", r.icon, r.name)
		// Print first 200 characters
		summary := r.report
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		fmt.Printf("%s\n", summary)
	}

	fmt.Println("\n⚠️  DISCLAIMER: This is for educational and research purposes only.")
	fmt.Println("   Not financial advice. Consult qualified professionals.")
}
