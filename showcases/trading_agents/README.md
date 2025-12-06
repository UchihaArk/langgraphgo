# Trading Agents

A multi-agent LLM-powered financial trading framework built with LangGraph Go and LangChain Go. This project is a complete Go implementation inspired by [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents).

## 📊 Overview

Trading Agents simulates a professional trading firm with specialized AI agents working collaboratively to analyze markets and make informed trading decisions. The system combines fundamental analysis, sentiment analysis, technical analysis, and risk management to provide comprehensive trading recommendations.

## 🎯 What Was Built

### 1. **Core Agent System** (7 specialized agents)
- **Fundamentals Analyst**: Analyzes company financials and valuation
- **Sentiment Analyst**: Evaluates social media and public sentiment
- **Technical Analyst**: Performs technical indicator analysis
- **News Analyst**: Monitors news and macroeconomic factors
- **Bullish Researcher**: Provides optimistic perspective
- **Bearish Researcher**: Identifies risks and concerns
- **Risk Manager**: Assesses and manages trading risks
- **Trader**: Synthesizes all reports for final decision

### 2. **Three Complete Interfaces**
- **Backend API Server**: RESTful API with health checks and analysis endpoints
- **CLI Tool**: Command-line interface for terminal-based analysis
- **Web Dashboard**: Modern, responsive web interface

### 3. **Market Data Integration**
- Alpha Vantage API integration
- Real-time quotes and company information
- Technical indicators calculation
- Sentiment data collection

## 🏗️ Architecture

### Agent Teams

1. **Analyst Team**
   - **Fundamentals Analyst**: Evaluates company financials and performance metrics
   - **Sentiment Analyst**: Analyzes social media and public sentiment
   - **News Analyst**: Monitors global news and macroeconomic indicators
   - **Technical Analyst**: Uses technical indicators for price trend analysis

2. **Research Team**
   - **Bullish Researcher**: Advocates for buying opportunities
   - **Bearish Researcher**: Identifies potential risks and selling signals

3. **Trader**
   - Synthesizes all analyst and researcher reports
   - Makes final trading decisions

4. **Risk Management Team**
   - Monitors portfolio exposure
   - Implements risk mitigation strategies
   - Ensures compliance with risk tolerance

### Agent Pipeline Flow
```
1. Data Collection
   ├─> Market quotes
   ├─> Company fundamentals
   ├─> Technical indicators
   └─> Sentiment data

2. Analyst Team (Parallel Conceptually)
   ├─> Fundamentals Analyst
   ├─> Sentiment Analyst
   └─> Technical Analyst

3. Research Team
   ├─> Bullish Researcher
   └─> Bearish Researcher

4. Risk Management
   └─> Risk Manager

5. Final Decision
   └─> Trader (synthesizes all reports)
```

## ✨ Features

- ✅ Multi-agent collaborative analysis
- ✅ Real-time market data integration
- ✅ Backend API server
- ✅ Command-line interface (CLI)
- ✅ Web-based dashboard
- ✅ Comprehensive logging and tracing
- ✅ Configurable risk tolerance
- ✅ Multiple timeframe support
- ✅ Verbose mode for detailed agent reports

## 📁 Project Structure

```
trading_agents/
├── README.md              # This file
├── types.go              # Core type definitions
├── graph.go              # Main trading graph workflow
├── agents/               # Agent implementations
│   ├── fundamentals_analyst.go
│   ├── sentiment_analyst.go
│   ├── technical_analyst.go
│   ├── trader.go
│   ├── risk_manager.go
│   └── researchers.go
├── tools/                # Market data tools
│   └── market_data.go
├── backend/              # API server
│   └── main.go
├── cli/                  # CLI tool
│   └── main.go
├── web/                  # Web interface
│   ├── index.html
│   ├── style.css
│   └── app.js
└── examples/             # Usage examples
    └── simple_analysis.go
```

## 📈 Statistics

- **Total Go Code**: ~2,000 lines
- **Number of Files**: 17
- **Agents Implemented**: 7 specialized agents
- **Interfaces**: 3 (API, CLI, Web)
- **Binary Sizes**:
  - Backend: 9.2 MB
  - CLI: 8.6 MB

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- OpenAI API key (for LLM)
- Alpha Vantage API key (for market data, optional)

### Installation

```bash
# Set environment variables
export OPENAI_API_KEY="your-openai-key"
export ALPHA_VANTAGE_API_KEY="your-alpha-vantage-key"  # Optional

# From the langgraphgo root directory
cd showcases/trading_agents

# Build backend server
go build -o ../../bin/trading-agents ./backend

# Build CLI tool
go build -o ../../bin/trading-cli ./cli
```

### Running the Backend

```bash
./bin/trading-agents --port 8080
```

The server will start on `http://localhost:8080`

### Running the CLI

```bash
# Analyze a stock
./bin/trading-cli -cmd analyze -symbol AAPL -verbose

# Get trading recommendation
./bin/trading-cli -cmd recommend -symbol AAPL -capital 10000
```

### Running the Web Interface

```bash
# Start backend first, then
cd showcases/trading_agents/web
# Open index.html in your browser
```

Or serve it with a simple HTTP server:

```bash
cd showcases/trading_agents/web
python3 -m http.server 3000
# Then open http://localhost:3000 in your browser
```

## 📚 Usage Guide

### Backend API

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

### CLI Options

- `-cmd` : Command (analyze, recommend, quick)
- `-symbol` : Stock symbol (required)
- `-capital` : Available capital in dollars (default: 10000)
- `-risk-level` : Risk tolerance (low, moderate, high)
- `-timeframe` : Trading timeframe (5m, 1H, 1D, 1W)
- `-verbose` : Show detailed agent reports
- `-json` : Output in JSON format

### CLI Examples

#### Full Analysis
```bash
./bin/trading-cli -cmd analyze -symbol AAPL -verbose
```

#### Trading Recommendation
```bash
./bin/trading-cli -cmd recommend -symbol GOOGL -capital 50000 -risk-level low
```

#### Quick Check
```bash
./bin/trading-cli -cmd quick -symbol TSLA
```

### Example Workflows

#### Day Trading Workflow

```bash
# Quick checks for multiple stocks
./bin/trading-cli -cmd quick -symbol AAPL
./bin/trading-cli -cmd quick -symbol GOOGL
./bin/trading-cli -cmd quick -symbol TSLA

# Full analysis for best candidate
./bin/trading-cli -cmd analyze -symbol AAPL -timeframe 5m -verbose
```

#### Investment Analysis Workflow

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

## 📊 Understanding the Output

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

## 🛠️ Technical Implementation

### State Management
- Map-based state flowing through pipeline
- Each agent enriches the state
- Final state contains all reports and decision

### LLM Integration
- OpenAI GPT-4 for agent reasoning
- Temperature-controlled responses
- Structured output parsing

### LangGraph Go Integration
- State management through graph nodes
- Sequential execution of agent pipeline
- Clean separation of concerns

## 🔧 Troubleshooting

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

## 💡 Tips for Best Results

1. **Use Valid Symbols**: Make sure to use correct ticker symbols (e.g., AAPL for Apple, not APPLE)

2. **Set Realistic Capital**: Use actual capital amounts for accurate position sizing

3. **Match Risk Tolerance**: Choose risk level that matches your actual risk tolerance

4. **Review All Reports**: Don't just look at the recommendation - read the detailed analysis

5. **Consider Context**: The analysis is point-in-time. Market conditions change rapidly.

6. **Combine with Research**: Use this as one tool among many in your research process

## 🎓 Educational Value

This project demonstrates:
- Multi-agent system architecture
- LangGraph Go workflow orchestration
- LangChain Go LLM integration
- RESTful API design
- Modern web interface development
- Production-ready Go application structure

## ⚠️ Important Disclaimers

- **Educational Purpose Only**: Not financial advice
- **Research Tool**: For learning and experimentation
- **No Liability**: Use at your own risk
- **Professional Advice**: Consult qualified financial advisors

**This framework is for research and educational purposes only. It is NOT intended as financial, investment, or trading advice. Always consult with qualified financial professionals before making investment decisions.**

## 🔮 Future Enhancements

Potential improvements:
- Real-time WebSocket streaming
- Historical backtesting
- Portfolio management
- Multiple timeframe analysis
- Machine learning integration
- Database persistence
- User authentication
- Paper trading mode

## 🙏 Acknowledgments

- Inspired by [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents)
- Built with [LangGraph Go](https://github.com/smallnest/langgraphgo)
- Powered by [LangChain Go](https://github.com/tmc/langchaingo)
- Market data from Alpha Vantage

## 📝 References

- [TradingAgents Paper (arXiv:2412.20138)](https://arxiv.org/abs/2412.20138)
- [TradingAgents GitHub](https://github.com/TauricResearch/TradingAgents)
- [LangGraph Go Documentation](https://github.com/smallnest/langgraphgo)
- [LangChain Go Documentation](https://github.com/tmc/langchaingo)

## 📄 License

MIT License - See LICENSE file for details

---

**Project Status**: ✅ Complete and Ready to Use

**Last Updated**: December 2024
