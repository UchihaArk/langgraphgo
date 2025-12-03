# Brave Search Tool Example

This example demonstrates how to use the Brave Search tool with LangGraphGo to create an AI agent that can search the web using Brave's privacy-focused search API.

## Prerequisites

1. Get your Brave Search API key from [Brave Search API](https://api.search.brave.com/)
2. Get an LLM API key (OpenAI or DeepSeek)

## Setup

Set up your environment variables:

```bash
export BRAVE_API_KEY="your-brave-api-key"
export OPENAI_API_KEY="your-openai-api-key"
# OR
export DEEPSEEK_API_KEY="your-deepseek-api-key"
```

## Run

```bash
go run main.go
```

## How It Works

1. **Initialize the LLM**: Creates an OpenAI-compatible LLM client
2. **Initialize the Tool**: Creates a Brave Search tool with custom options:
   - Count: Number of results to return (1-20)
   - Country: Country code for localized results
   - Lang: Language code for search results
3. **Create ReAct Agent**: Combines the LLM and tool into a ReAct agent
4. **Run the Agent**: Sends a query and gets the response

## Customization

You can customize the Brave Search tool with various options:

```go
braveTool, err := tool.NewBraveSearch("",
    tool.WithBraveCount(10),           // Number of results (1-20)
    tool.WithBraveCountry("CN"),       // Country code
    tool.WithBraveLang("zh"),          // Language code
)
```

## Benefits of Brave Search

- Privacy-focused: No tracking or profiling
- Independent: Not reliant on Google or Bing
- Fresh results: Real-time web search
- Global coverage: Supports multiple countries and languages
