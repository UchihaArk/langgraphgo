package agents

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// RiskManager assesses and manages trading risks
type RiskManager struct {
	model llms.Model
}

// NewRiskManager creates a new risk manager
func NewRiskManager(apiKey string) (*RiskManager, error) {
	model, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &RiskManager{
		model: model,
	}, nil
}

// AssessRisk performs risk assessment
func (r *RiskManager) AssessRisk(ctx context.Context, state map[string]interface{}) (string, float64, error) {
	symbol := state["symbol"].(string)
	capital := state["capital"].(float64)
	riskTolerance := state["risk_tolerance"].(string)
	marketData := state["market_data"].(map[string]float64)
	technicalIndicators := state["technical_indicators"].(map[string]float64)

	prompt := fmt.Sprintf(`You are a Risk Manager responsible for protecting the firm's capital.

Assess the risk for trading %s:

Portfolio Information:
- Available Capital: $%.2f
- Risk Tolerance: %s
- Current Price: $%.2f

Market Volatility:
- ATR (14-day): %.2f
- Price Range (High-Low): $%.2f
- Volume: %.0f

Technical Risk Indicators:
- RSI: %.2f (Overbought >70, Oversold <30)
- Bollinger Band Width: $%.2f

Provide a comprehensive risk assessment covering:

1. **Market Risk**: Current market volatility and price action risks
2. **Liquidity Risk**: Trading volume and market depth considerations
3. **Volatility Assessment**: Is volatility elevated or normal?
4. **Position Sizing**: Maximum safe position size given risk parameters
5. **Stop-Loss Recommendations**: Appropriate stop-loss levels
6. **Risk Score**: Provide an overall risk score from 0-100 (0=lowest risk, 100=highest risk)
7. **Risk Mitigation**: Specific recommendations to manage identified risks

Be conservative in your assessment. Capital preservation is paramount.

End your response with:
RISK_SCORE: [0-100]`,
		symbol,
		capital,
		riskTolerance,
		marketData["price"],
		technicalIndicators["atr_14"],
		marketData["high"]-marketData["low"],
		marketData["volume"],
		technicalIndicators["rsi_14"],
		technicalIndicators["bb_upper"]-technicalIndicators["bb_lower"],
	)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := r.model.GenerateContent(ctx, messages,
		llms.WithTemperature(0.3), // Lower temperature for risk assessment
		llms.WithMaxTokens(1500),
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to generate risk assessment: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from model")
	}

	analysis := resp.Choices[0].Content

	// Extract risk score
	riskScore := extractRiskScore(analysis)

	return analysis, riskScore, nil
}

// extractRiskScore extracts the risk score from the analysis
func extractRiskScore(analysis string) float64 {
	// Look for RISK_SCORE: XX pattern
	re := regexp.MustCompile(`RISK_SCORE:\s*(\d+)`)
	matches := re.FindStringSubmatch(analysis)
	if len(matches) > 1 {
		if score, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return score
		}
	}

	// Default to moderate risk if not found
	return 50.0
}
