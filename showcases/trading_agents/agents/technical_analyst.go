package agents

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// TechnicalAnalyst performs technical analysis
type TechnicalAnalyst struct {
	model llms.Model
}

// NewTechnicalAnalyst creates a new technical analyst
func NewTechnicalAnalyst(apiKey string) (*TechnicalAnalyst, error) {
	model, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &TechnicalAnalyst{
		model: model,
	}, nil
}

// Analyze performs technical analysis
func (t *TechnicalAnalyst) Analyze(ctx context.Context, state map[string]interface{}) (string, error) {
	symbol := state["symbol"].(string)
	indicators := state["technical_indicators"].(map[string]float64)
	marketData := state["market_data"].(map[string]float64)

	prompt := fmt.Sprintf(`You are a Technical Analyst expert in chart patterns and technical indicators.

Analyze the technical indicators for %s:

Price Action:
- Current Price: $%.2f
- Open: $%.2f
- High: $%.2f
- Low: $%.2f
- Change: %.2f%%

Technical Indicators:
- RSI (14): %.2f
- MACD: %.2f
- MACD Signal: %.2f
- SMA (50): $%.2f
- SMA (200): $%.2f
- EMA (12): $%.2f
- EMA (26): $%.2f
- Bollinger Band Upper: $%.2f
- Bollinger Band Lower: $%.2f
- ATR (14): %.2f

Provide a comprehensive technical analysis covering:
1. **Trend Analysis**: What is the current trend? (uptrend/downtrend/sideways)
2. **Momentum Indicators**: What do RSI and MACD tell us?
3. **Support and Resistance**: Key price levels to watch
4. **Moving Averages**: Price position relative to moving averages
5. **Volatility Assessment**: ATR and Bollinger Bands analysis
6. **Entry/Exit Signals**: Specific price levels for entry and exit
7. **Trading Strategy**: Short-term and medium-term outlook

Provide specific price targets and stop-loss levels.`,
		symbol,
		marketData["price"],
		marketData["open"],
		marketData["high"],
		marketData["low"],
		marketData["change_percent"],
		indicators["rsi_14"],
		indicators["macd"],
		indicators["macd_signal"],
		indicators["sma_50"],
		indicators["sma_200"],
		indicators["ema_12"],
		indicators["ema_26"],
		indicators["bb_upper"],
		indicators["bb_lower"],
		indicators["atr_14"],
	)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := t.model.GenerateContent(ctx, messages,
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(1500),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate analysis: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return resp.Choices[0].Content, nil
}
