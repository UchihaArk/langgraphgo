package trading_agents

import "time"

// TradingState represents the state that flows through the agent graph
type TradingState struct {
	// Input
	Symbol        string  `json:"symbol"`
	Timeframe     string  `json:"timeframe"`      // e.g., "1D", "1H", "5m"
	Capital       float64 `json:"capital"`        // Available capital
	RiskTolerance string  `json:"risk_tolerance"` // "low", "moderate", "high"

	// Market Data
	CurrentPrice     float64            `json:"current_price"`
	MarketData       map[string]float64 `json:"market_data"`        // OHLCV and other metrics
	CompanyInfo      map[string]string  `json:"company_info"`       // Company fundamentals
	NewsHeadlines    []NewsItem         `json:"news_headlines"`     // Recent news
	SocialSentiment  map[string]float64 `json:"social_sentiment"`   // Sentiment scores
	TechnicalIndic   map[string]float64 `json:"technical_indicators"` // Technical analysis

	// Analyst Reports
	FundamentalsReport string `json:"fundamentals_report"`
	SentimentReport    string `json:"sentiment_report"`
	NewsReport         string `json:"news_report"`
	TechnicalReport    string `json:"technical_report"`

	// Research Reports
	BullishResearch string `json:"bullish_research"`
	BearishResearch string `json:"bearish_research"`

	// Risk Assessment
	RiskAnalysis string  `json:"risk_analysis"`
	RiskScore    float64 `json:"risk_score"` // 0-100

	// Trading Decision
	Recommendation string  `json:"recommendation"` // "BUY", "SELL", "HOLD"
	Confidence     float64 `json:"confidence"`     // 0-100
	PositionSize   float64 `json:"position_size"`  // Number of shares
	StopLoss       float64 `json:"stop_loss"`      // Stop loss price
	TakeProfit     float64 `json:"take_profit"`    // Take profit price
	Reasoning      string  `json:"reasoning"`      // Detailed explanation

	// Metadata
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewsItem represents a news article
type NewsItem struct {
	Title       string    `json:"title"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	Summary     string    `json:"summary"`
	Sentiment   string    `json:"sentiment"` // "positive", "negative", "neutral"
}

// MarketData represents market data for a symbol
type MarketData struct {
	Symbol    string             `json:"symbol"`
	Price     float64            `json:"price"`
	Change    float64            `json:"change"`
	ChangeP   float64            `json:"change_percent"`
	Volume    int64              `json:"volume"`
	OHLC      OHLC               `json:"ohlc"`
	Timestamp time.Time          `json:"timestamp"`
	Extra     map[string]float64 `json:"extra"`
}

// OHLC represents Open, High, Low, Close prices
type OHLC struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

// AnalysisRequest represents a request to analyze a stock
type AnalysisRequest struct {
	Symbol        string  `json:"symbol"`
	Timeframe     string  `json:"timeframe,omitempty"`
	Capital       float64 `json:"capital,omitempty"`
	RiskTolerance string  `json:"risk_tolerance,omitempty"`
}

// AnalysisResponse represents the response from analysis
type AnalysisResponse struct {
	Symbol         string                 `json:"symbol"`
	Recommendation string                 `json:"recommendation"`
	Confidence     float64                `json:"confidence"`
	PositionSize   float64                `json:"position_size"`
	StopLoss       float64                `json:"stop_loss"`
	TakeProfit     float64                `json:"take_profit"`
	Reasoning      string                 `json:"reasoning"`
	RiskScore      float64                `json:"risk_score"`
	Reports        map[string]string      `json:"reports"`
	Metadata       map[string]interface{} `json:"metadata"`
	Timestamp      time.Time              `json:"timestamp"`
}

// AgentConfig represents configuration for agents
type AgentConfig struct {
	ModelName     string  `json:"model_name"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	Verbose       bool    `json:"verbose"`
	APIKey        string  `json:"-"` // Don't serialize API keys
	AlphaVantageKey string `json:"-"`
}

// DefaultConfig returns default agent configuration
func DefaultConfig() *AgentConfig {
	return &AgentConfig{
		ModelName:   "gpt-4",
		Temperature: 0.7,
		MaxTokens:   2000,
		Verbose:     false,
	}
}
