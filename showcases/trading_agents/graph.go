package trading_agents

import (
	"context"
	"fmt"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/showcases/trading_agents/agents"
	"github.com/smallnest/langgraphgo/showcases/trading_agents/tools"
)

// TradingAgentsGraph represents the main trading agents graph
type TradingAgentsGraph struct {
	workflow         *graph.StateGraph
	runnable         *graph.StateRunnable
	marketData       *tools.MarketDataProvider
	fundamentals     *agents.FundamentalsAnalyst
	sentiment        *agents.SentimentAnalyst
	technical        *agents.TechnicalAnalyst
	bullishResearch  *agents.BullishResearcher
	bearishResearch  *agents.BearishResearcher
	riskManager      *agents.RiskManager
	trader           *agents.Trader
}

// NewTradingAgentsGraph creates a new trading agents graph
func NewTradingAgentsGraph(config *AgentConfig) (*TradingAgentsGraph, error) {
	// Initialize tools and agents
	marketData := tools.NewMarketDataProvider(config.AlphaVantageKey)

	fundamentals, err := agents.NewFundamentalsAnalyst(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create fundamentals analyst: %w", err)
	}

	sentiment, err := agents.NewSentimentAnalyst(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create sentiment analyst: %w", err)
	}

	technical, err := agents.NewTechnicalAnalyst(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create technical analyst: %w", err)
	}

	bullishResearch, err := agents.NewBullishResearcher(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create bullish researcher: %w", err)
	}

	bearishResearch, err := agents.NewBearishResearcher(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create bearish researcher: %w", err)
	}

	riskManager, err := agents.NewRiskManager(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create risk manager: %w", err)
	}

	trader, err := agents.NewTrader(config.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create trader: %w", err)
	}

	tg := &TradingAgentsGraph{
		workflow:        graph.NewStateGraph(),
		marketData:      marketData,
		fundamentals:    fundamentals,
		sentiment:       sentiment,
		technical:       technical,
		bullishResearch: bullishResearch,
		bearishResearch: bearishResearch,
		riskManager:     riskManager,
		trader:          trader,
	}

	// Build the graph
	if err := tg.buildGraph(); err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	return tg, nil
}

// buildGraph constructs the trading agents workflow
func (tg *TradingAgentsGraph) buildGraph() error {
	// Data Collection Node
	tg.workflow.AddNode("data_collection", "Collect market data and news", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		symbol := s["symbol"].(string)

		// Fetch all market data
		quote, err := tg.marketData.GetQuote(ctx, symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get quote: %w", err)
		}

		companyInfo, err := tg.marketData.GetCompanyOverview(ctx, symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get company info: %w", err)
		}

		indicators, err := tg.marketData.GetTechnicalIndicators(ctx, symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get technical indicators: %w", err)
		}

		sentimentData, err := tg.marketData.GetSentiment(ctx, symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to get sentiment: %w", err)
		}

		s["current_price"] = quote["price"]
		s["market_data"] = quote
		s["company_info"] = companyInfo
		s["technical_indicators"] = indicators
		s["social_sentiment"] = sentimentData
		s["timestamp"] = time.Now()

		return s, nil
	})

	// Fundamentals Analyst Node
	tg.workflow.AddNode("fundamentals_analyst", "Analyze company fundamentals", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		report, err := tg.fundamentals.Analyze(ctx, s)
		if err != nil {
			return nil, err
		}
		s["fundamentals_report"] = report
		return s, nil
	})

	// Sentiment Analyst Node
	tg.workflow.AddNode("sentiment_analyst", "Analyze market sentiment", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		report, err := tg.sentiment.Analyze(ctx, s)
		if err != nil {
			return nil, err
		}
		s["sentiment_report"] = report
		return s, nil
	})

	// Technical Analyst Node
	tg.workflow.AddNode("technical_analyst", "Analyze technical indicators", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		report, err := tg.technical.Analyze(ctx, s)
		if err != nil {
			return nil, err
		}
		s["technical_report"] = report
		return s, nil
	})

	// Bullish Researcher Node
	tg.workflow.AddNode("bullish_researcher", "Provide bullish perspective", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		research, err := tg.bullishResearch.Research(ctx, s)
		if err != nil {
			return nil, err
		}
		s["bullish_research"] = research
		return s, nil
	})

	// Bearish Researcher Node
	tg.workflow.AddNode("bearish_researcher", "Provide bearish perspective", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		research, err := tg.bearishResearch.Research(ctx, s)
		if err != nil {
			return nil, err
		}
		s["bearish_research"] = research
		return s, nil
	})

	// Risk Manager Node
	tg.workflow.AddNode("risk_manager", "Assess trading risks", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		analysis, score, err := tg.riskManager.AssessRisk(ctx, s)
		if err != nil {
			return nil, err
		}
		s["risk_analysis"] = analysis
		s["risk_score"] = score
		return s, nil
	})

	// Trader Decision Node
	tg.workflow.AddNode("trader", "Make final trading decision", func(ctx context.Context, state interface{}) (interface{}, error) {
		s := state.(map[string]interface{})
		decision, err := tg.trader.MakeDecision(ctx, s)
		if err != nil {
			return nil, err
		}

		// Merge decision into state
		for k, v := range decision {
			s[k] = v
		}
		return s, nil
	})

	// Define the workflow
	tg.workflow.SetEntryPoint("data_collection")

	// Data collection flows to all analysts in parallel (conceptually)
	// In practice, we run them sequentially but they're independent
	tg.workflow.AddEdge("data_collection", "fundamentals_analyst")
	tg.workflow.AddEdge("fundamentals_analyst", "sentiment_analyst")
	tg.workflow.AddEdge("sentiment_analyst", "technical_analyst")

	// Analysts feed into researchers
	tg.workflow.AddEdge("technical_analyst", "bullish_researcher")
	tg.workflow.AddEdge("bullish_researcher", "bearish_researcher")

	// Researchers feed into risk manager
	tg.workflow.AddEdge("bearish_researcher", "risk_manager")

	// Risk manager feeds into trader for final decision
	tg.workflow.AddEdge("risk_manager", "trader")

	// Trader makes final decision and ends
	tg.workflow.AddEdge("trader", graph.END)

	// Compile the graph
	runnable, err := tg.workflow.Compile()
	if err != nil {
		return fmt.Errorf("failed to compile graph: %w", err)
	}

	tg.runnable = runnable
	return nil
}

// Analyze executes the full trading analysis pipeline
func (tg *TradingAgentsGraph) Analyze(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error) {
	// Prepare initial state
	initialState := map[string]interface{}{
		"symbol":         request.Symbol,
		"timeframe":      request.Timeframe,
		"capital":        request.Capital,
		"risk_tolerance": request.RiskTolerance,
	}

	// Set defaults
	if initialState["timeframe"] == "" {
		initialState["timeframe"] = "1D"
	}
	if initialState["capital"] == 0.0 {
		initialState["capital"] = 10000.0 // Default $10k
	}
	if initialState["risk_tolerance"] == "" {
		initialState["risk_tolerance"] = "moderate"
	}

	// Execute the graph
	result, err := tg.runnable.Invoke(ctx, initialState)
	if err != nil {
		return nil, fmt.Errorf("failed to execute graph: %w", err)
	}

	// Extract result
	finalState := result.(map[string]interface{})

	// Build response
	response := &AnalysisResponse{
		Symbol:         finalState["symbol"].(string),
		Recommendation: finalState["recommendation"].(string),
		Confidence:     finalState["confidence"].(float64),
		PositionSize:   finalState["position_size"].(float64),
		StopLoss:       finalState["stop_loss"].(float64),
		TakeProfit:     finalState["take_profit"].(float64),
		Reasoning:      finalState["reasoning"].(string),
		RiskScore:      finalState["risk_score"].(float64),
		Reports: map[string]string{
			"fundamentals": finalState["fundamentals_report"].(string),
			"sentiment":    finalState["sentiment_report"].(string),
			"technical":    finalState["technical_report"].(string),
			"bullish":      finalState["bullish_research"].(string),
			"bearish":      finalState["bearish_research"].(string),
			"risk":         finalState["risk_analysis"].(string),
		},
		Metadata: map[string]interface{}{
			"current_price": finalState["current_price"],
			"market_data":   finalState["market_data"],
		},
		Timestamp: finalState["timestamp"].(time.Time),
	}

	return response, nil
}
