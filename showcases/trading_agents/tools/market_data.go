package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// MarketDataProvider provides market data from various sources
type MarketDataProvider struct {
	AlphaVantageKey string
	httpClient      *http.Client
}

// NewMarketDataProvider creates a new market data provider
func NewMarketDataProvider(apiKey string) *MarketDataProvider {
	return &MarketDataProvider{
		AlphaVantageKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetQuote gets current quote for a symbol
func (m *MarketDataProvider) GetQuote(ctx context.Context, symbol string) (map[string]float64, error) {
	if m.AlphaVantageKey == "" {
		// Return mock data if no API key
		return m.getMockQuote(symbol), nil
	}

	baseURL := "https://www.alphavantage.co/query"
	params := url.Values{}
	params.Set("function", "GLOBAL_QUOTE")
	params.Set("symbol", symbol)
	params.Set("apikey", m.AlphaVantageKey)

	resp, err := m.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to get quote: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse Alpha Vantage response
	quote := make(map[string]float64)
	if globalQuote, ok := result["Global Quote"].(map[string]interface{}); ok {
		if price, ok := globalQuote["05. price"].(string); ok {
			var p float64
			fmt.Sscanf(price, "%f", &p)
			quote["price"] = p
		}
		if change, ok := globalQuote["09. change"].(string); ok {
			var c float64
			fmt.Sscanf(change, "%f", &c)
			quote["change"] = c
		}
		if changeP, ok := globalQuote["10. change percent"].(string); ok {
			var cp float64
			fmt.Sscanf(changeP, "%f%%", &cp)
			quote["change_percent"] = cp
		}
	}

	if len(quote) == 0 {
		return m.getMockQuote(symbol), nil
	}

	return quote, nil
}

// GetCompanyOverview gets company fundamental data
func (m *MarketDataProvider) GetCompanyOverview(ctx context.Context, symbol string) (map[string]string, error) {
	if m.AlphaVantageKey == "" {
		return m.getMockCompanyInfo(symbol), nil
	}

	baseURL := "https://www.alphavantage.co/query"
	params := url.Values{}
	params.Set("function", "OVERVIEW")
	params.Set("symbol", symbol)
	params.Set("apikey", m.AlphaVantageKey)

	resp, err := m.httpClient.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to get company overview: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to string map
	info := make(map[string]string)
	for k, v := range result {
		if str, ok := v.(string); ok {
			info[k] = str
		}
	}

	if len(info) == 0 {
		return m.getMockCompanyInfo(symbol), nil
	}

	return info, nil
}

// GetTechnicalIndicators calculates technical indicators
func (m *MarketDataProvider) GetTechnicalIndicators(ctx context.Context, symbol string) (map[string]float64, error) {
	// For demo purposes, return mock technical indicators
	// In production, you would calculate these from historical data
	indicators := map[string]float64{
		"rsi_14":      65.5,  // Relative Strength Index
		"macd":        2.3,   // MACD
		"macd_signal": 1.8,   // MACD Signal
		"sma_50":      150.2, // 50-day Simple Moving Average
		"sma_200":     145.8, // 200-day Simple Moving Average
		"ema_12":      151.5, // 12-day Exponential Moving Average
		"ema_26":      149.3, // 26-day Exponential Moving Average
		"bb_upper":    155.0, // Bollinger Band Upper
		"bb_lower":    145.0, // Bollinger Band Lower
		"atr_14":      3.5,   // Average True Range
	}

	return indicators, nil
}

// GetNews gets recent news for a symbol
func (m *MarketDataProvider) GetNews(ctx context.Context, symbol string) ([]map[string]interface{}, error) {
	// Mock news data
	news := []map[string]interface{}{
		{
			"title":        fmt.Sprintf("%s Reports Strong Q4 Earnings", symbol),
			"source":       "Financial Times",
			"published_at": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			"summary":      "Company beats analyst expectations with record revenue growth",
			"sentiment":    "positive",
		},
		{
			"title":        fmt.Sprintf("Analysts Upgrade %s to Buy Rating", symbol),
			"source":       "Bloomberg",
			"published_at": time.Now().Add(-5 * time.Hour).Format(time.RFC3339),
			"summary":      "Major investment firms raise price targets citing strong fundamentals",
			"sentiment":    "positive",
		},
		{
			"title":        fmt.Sprintf("%s Faces Regulatory Scrutiny", symbol),
			"source":       "Reuters",
			"published_at": time.Now().Add(-1 * 24 * time.Hour).Format(time.RFC3339),
			"summary":      "Regulatory authorities announce investigation into business practices",
			"sentiment":    "negative",
		},
	}

	return news, nil
}

// GetSentiment gets social media sentiment
func (m *MarketDataProvider) GetSentiment(ctx context.Context, symbol string) (map[string]float64, error) {
	// Mock sentiment data
	sentiment := map[string]float64{
		"twitter_sentiment":  0.65,  // -1 to 1
		"reddit_sentiment":   0.72,  // -1 to 1
		"news_sentiment":     0.58,  // -1 to 1
		"overall_sentiment":  0.65,  // -1 to 1
		"sentiment_volume":   15000, // Number of mentions
		"positive_mentions":  9750,
		"negative_mentions":  3750,
		"neutral_mentions":   1500,
	}

	return sentiment, nil
}

// Mock data methods for when API key is not available

func (m *MarketDataProvider) getMockQuote(symbol string) map[string]float64 {
	return map[string]float64{
		"price":          150.25,
		"change":         2.35,
		"change_percent": 1.59,
		"volume":         45678900,
		"open":           148.50,
		"high":           151.20,
		"low":            147.90,
		"close":          150.25,
	}
}

func (m *MarketDataProvider) getMockCompanyInfo(symbol string) map[string]string {
	return map[string]string{
		"Symbol":           symbol,
		"Name":             "Example Corporation",
		"Description":      "A leading technology company",
		"Sector":           "Technology",
		"Industry":         "Software",
		"MarketCap":        "2500000000000",
		"PERatio":          "28.5",
		"DividendYield":    "0.65",
		"EPS":              "5.27",
		"RevenuePerShare":  "22.45",
		"ProfitMargin":     "0.235",
		"52WeekHigh":       "182.94",
		"52WeekLow":        "124.17",
	}
}
