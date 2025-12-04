package agents

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// SentimentAnalyst analyzes market sentiment
type SentimentAnalyst struct {
	model llms.Model
}

// NewSentimentAnalyst creates a new sentiment analyst
func NewSentimentAnalyst(apiKey string) (*SentimentAnalyst, error) {
	model, err := openai.New(openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &SentimentAnalyst{
		model: model,
	}, nil
}

// Analyze performs sentiment analysis
func (s *SentimentAnalyst) Analyze(ctx context.Context, state map[string]interface{}) (string, error) {
	symbol := state["symbol"].(string)
	sentiment := state["social_sentiment"].(map[string]float64)

	prompt := fmt.Sprintf(`You are a Sentiment Analyst specializing in social media and public sentiment analysis.

Analyze the sentiment data for %s:

Social Media Sentiment:
- Twitter Sentiment Score: %.2f (range: -1 to +1)
- Reddit Sentiment Score: %.2f (range: -1 to +1)
- News Sentiment Score: %.2f (range: -1 to +1)
- Overall Sentiment: %.2f (range: -1 to +1)

Sentiment Volume:
- Total Mentions: %.0f
- Positive Mentions: %.0f
- Negative Mentions: %.0f
- Neutral Mentions: %.0f

Provide a comprehensive sentiment analysis covering:
1. **Overall Market Mood**: What is the general sentiment towards this stock?
2. **Sentiment Trends**: Are sentiments improving or deteriorating?
3. **Social Media Analysis**: Key themes and discussions on social platforms
4. **Investor Psychology**: What emotions are driving the market?
5. **Contrarian Indicators**: Any signs of excessive optimism or pessimism?
6. **Trading Implications**: How should this sentiment influence trading decisions?

Be specific and actionable in your analysis.`,
		symbol,
		sentiment["twitter_sentiment"],
		sentiment["reddit_sentiment"],
		sentiment["news_sentiment"],
		sentiment["overall_sentiment"],
		sentiment["sentiment_volume"],
		sentiment["positive_mentions"],
		sentiment["negative_mentions"],
		sentiment["neutral_mentions"],
	)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	resp, err := s.model.GenerateContent(ctx, messages,
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
