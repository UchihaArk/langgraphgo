# Trading Agents

A multi-agent LLM-powered financial trading framework built with LangGraph Go and LangChain Go. This project is a Go implementation inspired by [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents).

## Overview

Trading Agents simulates a professional trading firm with specialized AI agents working collaboratively to analyze markets and make informed trading decisions.

## Architecture

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

## Features

- ✅ Multi-agent collaborative analysis
- ✅ Real-time market data integration
- ✅ Backend API server
- ✅ Command-line interface (CLI)
- ✅ Web-based dashboard
- ✅ Comprehensive logging and tracing

## Components

```
trading_agents/
├── backend/        # API server implementation
├── cli/            # Command-line interface
├── web/            # Web frontend
├── agents/         # Agent implementations
├── tools/          # Market data and analysis tools
└── config/         # Configuration files
```

## Quick Start

### Prerequisites

- Go 1.21+
- OpenAI API key (for LLM)
- Alpha Vantage API key (for market data)

### Installation

```bash
# Set environment variables
export OPENAI_API_KEY="your-openai-key"
export ALPHA_VANTAGE_API_KEY="your-alpha-vantage-key"

# Build all components
go build -o bin/trading-agents ./showcases/trading_agents/backend
go build -o bin/trading-cli ./showcases/trading_agents/cli
```

### Running the Backend

```bash
./bin/trading-agents --port 8080
```

### Running the CLI

```bash
# Analyze a stock
./bin/trading-cli analyze --symbol AAPL

# Get trading recommendation
./bin/trading-cli recommend --symbol AAPL --capital 10000
```

### Running the Web Interface

```bash
# Start backend first, then
cd showcases/trading_agents/web
# Open index.html in your browser
```

## Usage Examples

### Backend API

```bash
# Health check
curl http://localhost:8080/health

# Analyze stock
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL", "timeframe": "1D"}'

# Get trading recommendation
curl -X POST http://localhost:8080/api/recommend \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL", "capital": 10000, "risk_tolerance": "moderate"}'
```

### CLI

```bash
# Quick analysis
trading-cli analyze --symbol TSLA --verbose

# Detailed recommendation with risk assessment
trading-cli recommend --symbol GOOGL --capital 50000 --risk-level low

# Monitor multiple stocks
trading-cli watch --symbols AAPL,GOOGL,TSLA --interval 5m
```

## Disclaimer

⚠️ **This framework is for research and educational purposes only. It is NOT intended as financial, investment, or trading advice. Always consult with qualified financial professionals before making investment decisions.**

## References

- Original Project: [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents)
- Paper: [arXiv:2412.20138](https://arxiv.org/abs/2412.20138)
- LangGraph Go: [smallnest/langgraphgo](https://github.com/smallnest/langgraphgo)
- LangChain Go: [tmc/langchaingo](https://github.com/tmc/langchaingo)

## License

MIT License - See LICENSE file for details
