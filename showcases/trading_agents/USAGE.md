# Trading Agents Usage Guide

## Quick Start

### 1. Setup Environment

```bash
# Set your API keys
export OPENAI_API_KEY="your-openai-api-key"
export ALPHA_VANTAGE_API_KEY="your-alpha-vantage-key"  # Optional
```

### 2. Build the Applications

```bash
# From the langgraphgo root directory
cd showcases/trading_agents

# Build backend server
go build -o ../../bin/trading-agents ./backend

# Build CLI tool
go build -o ../../bin/trading-cli ./cli
```

## Using the Backend API

### Start the Server

```bash
./bin/trading-agents --port 8080
```

The server will start on `http://localhost:8080`

### API Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
```

#### Full Analysis
```bash
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "AAPL",
    "capital": 10000,
    "risk_tolerance": "moderate",
    "timeframe": "1D"
  }'
```

#### Quick Check
```bash
curl -X POST http://localhost:8080/api/quick-check \
  -H "Content-Type: application/json" \
  -d '{"symbol": "TSLA"}'
```

## Using the CLI

### Full Analysis
```bash
./bin/trading-cli -cmd analyze -symbol AAPL -verbose
```

### Trading Recommendation
```bash
./bin/trading-cli -cmd recommend -symbol GOOGL -capital 50000 -risk-level low
```

### Quick Check
```bash
./bin/trading-cli -cmd quick -symbol TSLA
```

### CLI Options

- `-cmd` : Command (analyze, recommend, quick)
- `-symbol` : Stock symbol (required)
- `-capital` : Available capital in dollars (default: 10000)
- `-risk-level` : Risk tolerance (low, moderate, high)
- `-timeframe` : Trading timeframe (5m, 1H, 1D, 1W)
- `-verbose` : Show detailed agent reports
- `-json` : Output in JSON format

## Using the Web Interface

### 1. Start the Backend Server

```bash
./bin/trading-agents --port 8080
```

### 2. Open the Web Interface

Simply open the `web/index.html` file in your browser:

```bash
open showcases/trading_agents/web/index.html
```

Or serve it with a simple HTTP server:

```bash
cd showcases/trading_agents/web
python3 -m http.server 3000
# Then open http://localhost:3000 in your browser
```

### 3. Analyze Stocks

1. Enter a stock symbol (e.g., AAPL, GOOGL, TSLA)
2. Set your capital and risk tolerance
3. Click "Analyze Stock"
4. Review the multi-agent analysis and trading recommendation

## Example Workflows

### Day Trading Workflow

```bash
# Quick checks for multiple stocks
./bin/trading-cli -cmd quick -symbol AAPL
./bin/trading-cli -cmd quick -symbol GOOGL
./bin/trading-cli -cmd quick -symbol TSLA

# Full analysis for best candidate
./bin/trading-cli -cmd analyze -symbol AAPL -timeframe 5m -verbose
```

### Investment Analysis Workflow

```bash
# Conservative long-term investment
./bin/trading-cli -cmd recommend \
  -symbol MSFT \
  -capital 100000 \
  -risk-level low \
  -timeframe 1W \
  -verbose
```

### API Integration Example

```python
import requests

# Analyze stock
response = requests.post('http://localhost:8080/api/analyze', json={
    'symbol': 'AAPL',
    'capital': 50000,
    'risk_tolerance': 'moderate'
})

result = response.json()
print(f"Recommendation: {result['recommendation']}")
print(f"Confidence: {result['confidence']}%")
print(f"Position Size: {result['position_size']} shares")
```

## Understanding the Output

### Recommendation Types

- **BUY** 🟢: Strong buying opportunity identified
- **SELL** 🔴: Sell recommendation or short opportunity
- **HOLD** 🟡: Maintain current position or stay on sidelines

### Confidence Score

- **80-100%**: Very high confidence, strong signals across all agents
- **60-80%**: Good confidence, majority of agents agree
- **40-60%**: Moderate confidence, mixed signals
- **Below 40%**: Low confidence, conflicting signals

### Risk Score

- **0-30**: Low risk, conservative trade
- **30-70**: Moderate risk, balanced approach
- **70-100**: High risk, aggressive trade

### Agent Reports

Each analysis includes reports from:

1. **Fundamentals Analyst**: Company financials and valuation
2. **Sentiment Analyst**: Social media and public sentiment
3. **Technical Analyst**: Chart patterns and indicators
4. **Bullish Researcher**: Positive perspective and opportunities
5. **Bearish Researcher**: Risks and cautionary signals
6. **Risk Manager**: Risk assessment and mitigation strategies

The **Trader** synthesizes all reports to make the final recommendation.

## Troubleshooting

### "API key is required" Error

Make sure you've set the OpenAI API key:
```bash
export OPENAI_API_KEY="your-key-here"
```

### "Analysis failed" Error

- Check your internet connection
- Verify the stock symbol is valid
- Ensure the backend server is running (for web interface)

### Backend Server Issues

Check if the server is running:
```bash
curl http://localhost:8080/health
```

View server logs for debugging:
```bash
./bin/trading-agents --verbose
```

## Tips for Best Results

1. **Use Valid Symbols**: Make sure to use correct ticker symbols (e.g., AAPL for Apple, not APPLE)

2. **Set Realistic Capital**: Use actual capital amounts for accurate position sizing

3. **Match Risk Tolerance**: Choose risk level that matches your actual risk tolerance

4. **Review All Reports**: Don't just look at the recommendation - read the detailed analysis

5. **Consider Context**: The analysis is point-in-time. Market conditions change rapidly.

6. **Combine with Research**: Use this as one tool among many in your research process

## Disclaimer

⚠️ **Important**: This tool is for **educational and research purposes only**.

- NOT financial advice
- NOT investment recommendations
- Always consult qualified financial professionals
- Past performance does not guarantee future results
- You are responsible for your own investment decisions

## Getting Help

- Report issues: [GitHub Issues](https://github.com/smallnest/langgraphgo/issues)
- Documentation: See README.md
- Examples: Check the `examples/` directory

## Next Steps

- Try analyzing different stocks
- Experiment with risk tolerance levels
- Compare recommendations across timeframes
- Build your own trading strategies using the API
