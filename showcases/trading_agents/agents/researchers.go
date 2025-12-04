package agents

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// BullishResearcher provides bullish perspective
type BullishResearcher struct {
	model llms.Model
}

// NewBullishResearcher creates a new bullish researcher
func NewBullishResearcher(apiKey string) (*BullishResearcher, error) {
	model, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &BullishResearcher{
		model: model,
	}, nil
}

// Research provides bullish research perspective
func (b *BullishResearcher) Research(ctx context.Context, state map[string]interface{}) (string, error) {
	symbol := state["symbol"].(string)
	fundamentalsReport := state["fundamentals_report"].(string)
	sentimentReport := state["sentiment_report"].(string)
	technicalReport := state["technical_report"].(string)

	prompt := fmt.Sprintf(`You are a Bullish Researcher. Your role is to identify and advocate for buying opportunities.

Review the analyst reports for %s and provide a BULLISH perspective:

=== ANALYST REPORTS ===
FUNDAMENTALS:
%s

SENTIMENT:
%s

TECHNICAL:
%s

=== YOUR TASK ===
Provide a comprehensive BULLISH analysis that:

1. **Identifies Growth Catalysts**: What positive factors could drive price higher?
2. **Highlights Strengths**: What are the strongest bullish signals?
3. **Opportunity Assessment**: Why is this a good buying opportunity?
4. **Risk-Reward**: What is the upside potential?
5. **Bull Case Scenario**: Best-case outcome and probability
6. **Counter-Arguments**: Address potential bearish concerns with rebuttals

Your analysis should be optimistic but grounded in facts from the reports.
Focus on why an investor should be BULLISH on this stock.`,
		symbol,
		fundamentalsReport,
		sentimentReport,
		technicalReport,
	)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := b.model.GenerateContent(ctx, messages,
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(1500),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate research: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return resp.Choices[0].Content, nil
}

// BearishResearcher provides bearish perspective
type BearishResearcher struct {
	model llms.Model
}

// NewBearishResearcher creates a new bearish researcher
func NewBearishResearcher(apiKey string) (*BearishResearcher, error) {
	model, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &BearishResearcher{
		model: model,
	}, nil
}

// Research provides bearish research perspective
func (b *BearishResearcher) Research(ctx context.Context, state map[string]interface{}) (string, error) {
	symbol := state["symbol"].(string)
	fundamentalsReport := state["fundamentals_report"].(string)
	sentimentReport := state["sentiment_report"].(string)
	technicalReport := state["technical_report"].(string)

	prompt := fmt.Sprintf(`You are a Bearish Researcher. Your role is to identify risks and advocate for caution or short positions.

Review the analyst reports for %s and provide a BEARISH perspective:

=== ANALYST REPORTS ===
FUNDAMENTALS:
%s

SENTIMENT:
%s

TECHNICAL:
%s

=== YOUR TASK ===
Provide a comprehensive BEARISH analysis that:

1. **Identifies Risks**: What negative factors could drive price lower?
2. **Highlights Weaknesses**: What are the key bearish signals?
3. **Risk Assessment**: Why should investors be cautious?
4. **Downside Potential**: What is the downside risk?
5. **Bear Case Scenario**: Worst-case outcome and probability
6. **Counter-Arguments**: Address bullish arguments with skepticism

Your analysis should be skeptical but fair and grounded in facts from the reports.
Focus on why an investor should be CAUTIOUS or BEARISH on this stock.`,
		symbol,
		fundamentalsReport,
		sentimentReport,
		technicalReport,
	)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := b.model.GenerateContent(ctx, messages,
		llms.WithTemperature(0.7),
		llms.WithMaxTokens(1500),
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate research: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return resp.Choices[0].Content, nil
}
