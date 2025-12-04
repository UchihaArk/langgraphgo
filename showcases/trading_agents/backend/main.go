package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	trading "github.com/smallnest/langgraphgo/showcases/trading_agents"
)

var (
	port     = flag.Int("port", 8080, "API server port")
	apiKey   = flag.String("api-key", "", "OpenAI API key (or set OPENAI_API_KEY env var)")
	alphaKey = flag.String("alpha-key", "", "Alpha Vantage API key (or set ALPHA_VANTAGE_API_KEY env var)")
	verbose  = flag.Bool("verbose", false, "Enable verbose logging")
)

type Server struct {
	graph  *trading.TradingAgentsGraph
	config *trading.AgentConfig
}

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
		log.Fatal("OpenAI API key is required. Set -api-key flag or OPENAI_API_KEY environment variable")
	}

	// Create configuration
	config := trading.DefaultConfig()
	config.APIKey = *apiKey
	config.AlphaVantageKey = *alphaKey
	config.Verbose = *verbose

	// Create trading agents graph
	graph, err := trading.NewTradingAgentsGraph(config)
	if err != nil {
		log.Fatalf("Failed to create trading agents graph: %v", err)
	}

	server := &Server{
		graph:  graph,
		config: config,
	}

	// Setup routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", server.handleHealth)

	// API endpoints
	mux.HandleFunc("/api/analyze", server.handleAnalyze)
	mux.HandleFunc("/api/quick-check", server.handleQuickCheck)

	// CORS middleware
	handler := corsMiddleware(mux)

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("🚀 Trading Agents API Server starting on %s", addr)
	log.Printf("📊 Endpoints:")
	log.Printf("   GET  /health           - Health check")
	log.Printf("   POST /api/analyze      - Full analysis")
	log.Printf("   POST /api/quick-check  - Quick recommendation")
	log.Printf("\n💡 Example:")
	log.Printf("   curl -X POST http://localhost:%d/api/analyze -H 'Content-Type: application/json' -d '{\"symbol\":\"AAPL\"}'", *port)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req trading.AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Symbol == "" {
		http.Error(w, "Symbol is required", http.StatusBadRequest)
		return
	}

	log.Printf("📈 Analyzing %s...", req.Symbol)

	// Perform analysis
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := s.graph.Analyze(ctx, req)
	if err != nil {
		log.Printf("❌ Analysis failed: %v", err)
		http.Error(w, fmt.Sprintf("Analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Analysis complete for %s: %s (confidence: %.0f%%)",
		result.Symbol, result.Recommendation, result.Confidence)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleQuickCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req trading.AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Symbol == "" {
		http.Error(w, "Symbol is required", http.StatusBadRequest)
		return
	}

	log.Printf("⚡ Quick check for %s...", req.Symbol)

	// Perform analysis
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := s.graph.Analyze(ctx, req)
	if err != nil {
		log.Printf("❌ Quick check failed: %v", err)
		http.Error(w, fmt.Sprintf("Quick check failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return simplified response
	quickResponse := map[string]interface{}{
		"symbol":         result.Symbol,
		"recommendation": result.Recommendation,
		"confidence":     result.Confidence,
		"current_price":  result.Metadata["current_price"],
		"risk_score":     result.RiskScore,
		"reasoning":      truncate(result.Reasoning, 200),
		"timestamp":      result.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quickResponse)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// truncate truncates a string to specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
