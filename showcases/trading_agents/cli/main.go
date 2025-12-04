package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	trading "github.com/smallnest/langgraphgo/showcases/trading_agents"
)

var (
	// Command flags
	command = flag.String("cmd", "analyze", "Command: analyze, recommend, quick")

	// Analysis flags
	symbol        = flag.String("symbol", "", "Stock symbol (required)")
	capital       = flag.Float64("capital", 10000, "Available capital")
	riskTolerance = flag.String("risk-level", "moderate", "Risk tolerance: low, moderate, high")
	timeframe     = flag.String("timeframe", "1D", "Timeframe: 1D, 1H, 5m")

	// API keys
	apiKey   = flag.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY env var)")
	alphaKey = flag.String("alpha-key", "", "Alpha Vantage API key (or set ALPHA_VANTAGE_API_KEY env var)")

	// Output flags
	verbose = flag.Bool("verbose", false, "Verbose output")
	json    = flag.Bool("json", false, "Output in JSON format")
)

func main() {
	flag.Parse()

	// Get API keys from environment if not provided
	if *apiKey == "" {
		*apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if *alphaKey == "" {
		*alphaKey = os.Getenv("ALPHA_VANTAGE_API_KEY")
	}

	if *apiKey == "" {
		log.Fatal("❌ OpenAI API key is required. Set -api-key flag or OPENAI_API_KEY environment variable")
	}

	if *symbol == "" {
		printUsage()
		os.Exit(1)
	}

	// Create configuration
	config := trading.DefaultConfig()
	config.APIKey = *apiKey
	config.AlphaVantageKey = *alphaKey
	config.Verbose = *verbose

	// Create trading agents graph
	graph, err := trading.NewTradingAgentsGraph(config)
	if err != nil {
		log.Fatalf("❌ Failed to create trading agents: %v", err)
	}

	// Execute command
	ctx := context.Background()

	switch *command {
	case "analyze", "a":
		analyzeStock(ctx, graph)
	case "recommend", "r":
		recommendTrade(ctx, graph)
	case "quick", "q":
		quickCheck(ctx, graph)
	default:
		fmt.Printf("❌ Unknown command: %s\n", *command)
		printUsage()
		os.Exit(1)
	}
}

func analyzeStock(ctx context.Context, graph *trading.TradingAgentsGraph) {
	fmt.Printf("📊 Analyzing %s...\n\n", strings.ToUpper(*symbol))

	request := trading.AnalysisRequest{
		Symbol:        strings.ToUpper(*symbol),
		Timeframe:     *timeframe,
		Capital:       *capital,
		RiskTolerance: *riskTolerance,
	}

	start := time.Now()
	result, err := graph.Analyze(ctx, request)
	if err != nil {
		log.Fatalf("❌ Analysis failed: %v", err)
	}
	duration := time.Since(start)

	// Print results
	printHeader("TRADING RECOMMENDATION")
	fmt.Printf("Symbol:          %s\n", result.Symbol)
	fmt.Printf("Recommendation:  %s\n", colorRecommendation(result.Recommendation))
	fmt.Printf("Confidence:      %.1f%%\n", result.Confidence)
	fmt.Printf("Risk Score:      %.1f/100\n", result.RiskScore)

	if result.PositionSize > 0 {
		fmt.Printf("\nPosition Size:   %.2f shares\n", result.PositionSize)
		fmt.Printf("Entry Price:     $%.2f\n", result.Metadata["current_price"].(float64))
		fmt.Printf("Stop Loss:       $%.2f\n", result.StopLoss)
		fmt.Printf("Take Profit:     $%.2f\n", result.TakeProfit)

		invested := result.PositionSize * result.Metadata["current_price"].(float64)
		fmt.Printf("Capital Needed:  $%.2f\n", invested)
	}

	fmt.Printf("\n")
	printHeader("REASONING")
	fmt.Printf("%s\n\n", result.Reasoning)

	if *verbose {
		printHeader("ANALYST REPORTS")
		fmt.Printf("\n📈 Fundamentals:\n%s\n\n", truncate(result.Reports["fundamentals"], 300))
		fmt.Printf("💭 Sentiment:\n%s\n\n", truncate(result.Reports["sentiment"], 300))
		fmt.Printf("📉 Technical:\n%s\n\n", truncate(result.Reports["technical"], 300))
		fmt.Printf("🐂 Bullish View:\n%s\n\n", truncate(result.Reports["bullish"], 300))
		fmt.Printf("🐻 Bearish View:\n%s\n\n", truncate(result.Reports["bearish"], 300))
		fmt.Printf("⚠️  Risk Assessment:\n%s\n\n", truncate(result.Reports["risk"], 300))
	}

	fmt.Printf("⏱️  Analysis completed in %v\n", duration.Round(time.Second))
	fmt.Printf("🕐 Timestamp: %s\n", result.Timestamp.Format(time.RFC3339))

	printDisclaimer()
}

func recommendTrade(ctx context.Context, graph *trading.TradingAgentsGraph) {
	fmt.Printf("💰 Getting trading recommendation for %s...\n\n", strings.ToUpper(*symbol))

	request := trading.AnalysisRequest{
		Symbol:        strings.ToUpper(*symbol),
		Timeframe:     *timeframe,
		Capital:       *capital,
		RiskTolerance: *riskTolerance,
	}

	result, err := graph.Analyze(ctx, request)
	if err != nil {
		log.Fatalf("❌ Recommendation failed: %v", err)
	}

	// Print simplified recommendation
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("  TRADING RECOMMENDATION FOR %s\n", result.Symbol)
	fmt.Printf("═══════════════════════════════════════════\n\n")

	fmt.Printf("Action:     %s\n", colorRecommendation(result.Recommendation))
	fmt.Printf("Confidence: %.0f%% %s\n", result.Confidence, confidenceBar(result.Confidence))
	fmt.Printf("Risk:       %s (%.0f/100)\n", riskLevel(result.RiskScore), result.RiskScore)

	if result.Recommendation == "BUY" && result.PositionSize > 0 {
		currentPrice := result.Metadata["current_price"].(float64)
		investment := result.PositionSize * currentPrice
		potentialProfit := result.PositionSize * (result.TakeProfit - currentPrice)
		potentialLoss := result.PositionSize * (currentPrice - result.StopLoss)

		fmt.Printf("\n💵 Trade Details:\n")
		fmt.Printf("   Buy:         %.2f shares @ $%.2f\n", result.PositionSize, currentPrice)
		fmt.Printf("   Investment:  $%.2f\n", investment)
		fmt.Printf("   Stop Loss:   $%.2f (%.1f%%)\n", result.StopLoss, -100*potentialLoss/investment)
		fmt.Printf("   Take Profit: $%.2f (+%.1f%%)\n", result.TakeProfit, 100*potentialProfit/investment)
		fmt.Printf("\n📊 Risk/Reward:\n")
		fmt.Printf("   Potential Gain:  $%.2f\n", potentialProfit)
		fmt.Printf("   Potential Loss:  $%.2f\n", potentialLoss)
		fmt.Printf("   Risk/Reward:     1:%.2f\n", potentialProfit/potentialLoss)
	}

	fmt.Printf("\n")
	printDisclaimer()
}

func quickCheck(ctx context.Context, graph *trading.TradingAgentsGraph) {
	fmt.Printf("⚡ Quick check for %s...\n", strings.ToUpper(*symbol))

	request := trading.AnalysisRequest{
		Symbol:        strings.ToUpper(*symbol),
		Capital:       *capital,
		RiskTolerance: *riskTolerance,
	}

	result, err := graph.Analyze(ctx, request)
	if err != nil {
		log.Fatalf("❌ Quick check failed: %v", err)
	}

	// Ultra-compact output
	fmt.Printf("\n%s: %s (%.0f%% confidence, risk: %.0f/100)\n",
		result.Symbol,
		colorRecommendation(result.Recommendation),
		result.Confidence,
		result.RiskScore,
	)

	if result.Recommendation == "BUY" {
		fmt.Printf("📈 Entry: $%.2f | Stop: $%.2f | Target: $%.2f\n",
			result.Metadata["current_price"].(float64),
			result.StopLoss,
			result.TakeProfit,
		)
	}
	fmt.Println()
}

func printUsage() {
	fmt.Println("Trading Agents CLI - AI-Powered Stock Analysis")
	fmt.Println("\nUsage:")
	fmt.Println("  trading-cli -cmd <command> -symbol <SYMBOL> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  analyze, a    - Full analysis with detailed reports")
	fmt.Println("  recommend, r  - Trading recommendation with risk/reward")
	fmt.Println("  quick, q      - Quick check for fast decision")
	fmt.Println("\nExamples:")
	fmt.Println("  trading-cli -cmd analyze -symbol AAPL -verbose")
	fmt.Println("  trading-cli -cmd recommend -symbol TSLA -capital 50000 -risk-level low")
	fmt.Println("  trading-cli -cmd quick -symbol GOOGL")
}

func printHeader(title string) {
	fmt.Printf("\n═══════════════════════════════════════════\n")
	fmt.Printf("  %s\n", title)
	fmt.Printf("═══════════════════════════════════════════\n\n")
}

func printDisclaimer() {
	fmt.Println("\n⚠️  DISCLAIMER: This is for educational and research purposes only.")
	fmt.Println("   Not financial advice. Always consult a qualified financial advisor.")
}

func colorRecommendation(rec string) string {
	switch rec {
	case "BUY":
		return "🟢 BUY"
	case "SELL":
		return "🔴 SELL"
	case "HOLD":
		return "🟡 HOLD"
	default:
		return rec
	}
}

func confidenceBar(confidence float64) string {
	bars := int(confidence / 10)
	return strings.Repeat("█", bars) + strings.Repeat("░", 10-bars)
}

func riskLevel(score float64) string {
	if score < 30 {
		return "🟢 LOW"
	} else if score < 70 {
		return "🟡 MODERATE"
	}
	return "🔴 HIGH"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
