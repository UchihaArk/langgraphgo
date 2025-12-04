package agents

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Trader makes final trading decisions
type Trader struct {
	model llms.Model
}

// NewTrader creates a new trader agent
func NewTrader(apiKey string) (*Trader, error) {
	model, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &Trader{
		model: model,
	}, nil
}

// MakeDecision synthesizes all reports and makes trading decision
func (tr *Trader) MakeDecision(ctx context.Context, state map[string]interface{}) (map[string]interface{}, error) {
	symbol := state["symbol"].(string)
	capital := state["capital"].(float64)
	currentPrice := state["current_price"].(float64)

	fundamentalsReport := state["fundamentals_report"].(string)
	sentimentReport := state["sentiment_report"].(string)
	technicalReport := state["technical_report"].(string)
	bullishResearch := state["bullish_research"].(string)
	bearishResearch := state["bearish_research"].(string)
	riskAnalysis := state["risk_analysis"].(string)

	prompt := fmt.Sprintf(`You are a Senior Trader at a professional trading firm. Your job is to make the final trading decision based on comprehensive analysis from your team.

Symbol: %s
Current Price: $%.2f
Available Capital: $%.2f

=== ANALYST REPORTS ===

FUNDAMENTALS ANALYSIS:
%s

SENTIMENT ANALYSIS:
%s

TECHNICAL ANALYSIS:
%s

=== RESEARCH TEAM REPORTS ===

BULLISH PERSPECTIVE:
%s

BEARISH PERSPECTIVE:
%s

=== RISK MANAGEMENT ===
%s

=== YOUR TASK ===

Based on ALL the above analyses, provide a clear trading decision in the following format:

RECOMMENDATION: [BUY/SELL/HOLD]
CONFIDENCE: [0-100]
POSITION_SIZE: [number of shares, considering the available capital]
STOP_LOSS: [specific price level]
TAKE_PROFIT: [specific price level]

REASONING:
[Provide detailed reasoning that synthesizes all reports. Explain:
- Which factors were most influential in your decision
- How you balanced conflicting views
- Risk-reward assessment
- Time horizon for this trade
- Key triggers that would change your thesis]

Be decisive and specific. Your decision should balance all perspectives while prioritizing capital preservation.`,
		symbol,
		currentPrice,
		capital,
		fundamentalsReport,
		sentimentReport,
		technicalReport,
		bullishResearch,
		bearishResearch,
		riskAnalysis,
	)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := tr.model.GenerateContent(ctx, messages,
		llms.WithTemperature(0.5), // Lower temperature for more consistent decisions
		llms.WithMaxTokens(2000),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate decision: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from model")
	}

	response := resp.Choices[0].Content

	// Parse the response to extract structured data
	decision := parseTraderDecision(response, currentPrice, capital)
	decision["reasoning"] = response

	return decision, nil
}

// parseTraderDecision extracts structured data from trader's response
func parseTraderDecision(response string, currentPrice, capital float64) map[string]interface{} {
	decision := map[string]interface{}{
		"recommendation": "HOLD",
		"confidence":     50.0,
		"position_size":  0.0,
		"stop_loss":      currentPrice * 0.95,
		"take_profit":    currentPrice * 1.10,
	}

	// Extract recommendation
	if reMatch := regexp.MustCompile(`RECOMMENDATION:\s*(BUY|SELL|HOLD)`).FindStringSubmatch(response); len(reMatch) > 1 {
		decision["recommendation"] = reMatch[1]
	}

	// Extract confidence
	if confMatch := regexp.MustCompile(`CONFIDENCE:\s*(\d+)`).FindStringSubmatch(response); len(confMatch) > 1 {
		if conf, err := strconv.ParseFloat(confMatch[1], 64); err == nil {
			decision["confidence"] = conf
		}
	}

	// Extract position size
	if posMatch := regexp.MustCompile(`POSITION_SIZE:\s*([\d.]+)`).FindStringSubmatch(response); len(posMatch) > 1 {
		if size, err := strconv.ParseFloat(posMatch[1], 64); err == nil {
			decision["position_size"] = size
		}
	}

	// Extract stop loss
	if slMatch := regexp.MustCompile(`STOP_LOSS:\s*\$?([\d.]+)`).FindStringSubmatch(response); len(slMatch) > 1 {
		if sl, err := strconv.ParseFloat(slMatch[1], 64); err == nil {
			decision["stop_loss"] = sl
		}
	}

	// Extract take profit
	if tpMatch := regexp.MustCompile(`TAKE_PROFIT:\s*\$?([\d.]+)`).FindStringSubmatch(response); len(tpMatch) > 1 {
		if tp, err := strconv.ParseFloat(tpMatch[1], 64); err == nil {
			decision["take_profit"] = tp
		}
	}

	// Calculate position size if not explicitly provided or if it's 0
	if decision["position_size"].(float64) == 0 && decision["recommendation"].(string) == "BUY" {
		// Use 30% of capital for a BUY recommendation
		positionValue := capital * 0.3
		shares := positionValue / currentPrice
		decision["position_size"] = shares
	}

	return decision
}

// Helper function to extract numeric value from text
func extractNumber(text string) float64 {
	re := regexp.MustCompile(`[\d.]+`)
	match := re.FindString(text)
	if match != "" {
		val, _ := strconv.ParseFloat(match, 64)
		return val
	}
	return 0
}
