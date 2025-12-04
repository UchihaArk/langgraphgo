package agents

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// FundamentalsAnalyst analyzes company fundamentals
type FundamentalsAnalyst struct {
	model llms.Model
}

// NewFundamentalsAnalyst creates a new fundamentals analyst
func NewFundamentalsAnalyst(apiKey string) (*FundamentalsAnalyst, error) {
	model, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &FundamentalsAnalyst{
		model: model,
	}, nil
}

// Analyze performs fundamental analysis
func (f *FundamentalsAnalyst) Analyze(ctx context.Context, state map[string]interface{}) (string, error) {
	symbol := state["symbol"].(string)
	companyInfo := state["company_info"].(map[string]string)
	marketData := state["market_data"].(map[string]float64)

	// Construct analysis prompt
	prompt := fmt.Sprintf(`You are a Fundamentals Analyst at a professional trading firm.

Analyze the following company fundamentals for %s:

Company Information:
- Name: %s
- Sector: %s
- Industry: %s
- Market Cap: %s
- P/E Ratio: %s
- EPS: %s
- Dividend Yield: %s
- Profit Margin: %s
- Revenue Per Share: %s

Current Market Data:
- Current Price: $%.2f
- 52-Week High: %s
- 52-Week Low: %s

Provide a comprehensive fundamental analysis covering:
1. **Valuation**: Is the stock overvalued or undervalued?
2. **Financial Health**: Assess profitability, revenue growth, and margins
3. **Industry Position**: Competitive advantages and market position
4. **Growth Potential**: Future growth prospects and catalysts
5. **Investment Thesis**: Clear buy/sell/hold recommendation with reasoning

Format your analysis professionally and provide specific insights.`,
		symbol,
		companyInfo["Name"],
		companyInfo["Sector"],
		companyInfo["Industry"],
		companyInfo["MarketCap"],
		companyInfo["PERatio"],
		companyInfo["EPS"],
		companyInfo["DividendYield"],
		companyInfo["ProfitMargin"],
		companyInfo["RevenuePerShare"],
		marketData["price"],
		companyInfo["52WeekHigh"],
		companyInfo["52WeekLow"],
	)

	// Generate analysis
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := f.model.GenerateContent(ctx, messages,
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
