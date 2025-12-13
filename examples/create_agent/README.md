# Create Agent Example

This example demonstrates how to use the `prebuilt.CreateAgent` function in LangGraphGo to easily create a ReAct agent with functional options.

## 1. Background

Creating an agent often involves setting up a graph with an agent node, a tool execution node, and conditional routing logic. While `CreateReactAgent` provides a basic implementation, `CreateAgent` offers a more flexible and extensible way to construct agents, inspired by LangChain 1.0's design. It supports functional options for easier configuration of system messages, state modifiers, and future extensions like checkpointing.

## 2. Key Concepts

- **CreateAgent**: A factory function that builds a `StateGraph` for an agent.
- **Functional Options**: A pattern to pass optional parameters (e.g., `WithSystemMessage`) to the factory function, making the API clean and extensible.
- **System Message**: A predefined instruction that sets the behavior or persona of the agent.
- **State Modifier**: A function that intercepts and modifies the message history before it is sent to the LLM. This is useful for dynamic prompt engineering or filtering.

## 3. How It Works

1.  **Initialize LLM**: Create an instance of a language model (e.g., OpenAI).
2.  **Define Tools**: Create a list of tools the agent can use.
3.  **Create Agent**: Call `prebuilt.CreateAgent` with the model, tools, and options.
    - `WithSystemMessage`: Sets the system prompt.
    - `WithStateModifier`: Allows custom logic to modify messages.
4.  **Invoke**: Run the agent with an initial user message. The agent will loop through reasoning and tool execution until a final answer is reached.

## 4. Code Highlights

### Creating the Agent with Options

```go
agent, err := prebuilt.CreateAgent(model, inputTools,
    // Set a system message
    prebuilt.WithSystemMessage("You are a helpful weather assistant. Always be polite."),
    
    // Add a state modifier to log or alter messages
    prebuilt.WithStateModifier(func(msgs []llms.MessageContent) []llms.MessageContent {
        log.Printf("Current message count: %d", len(msgs))
        return msgs
    }),
)
```

### Running the Agent

```go
inputs := map[string]any{
    "messages": []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeHuman, "What is the weather in San Francisco?"),
    },
}

result, err := agent.Invoke(ctx, inputs)
```

## 5. Running the Example

```bash
export OPENAI_API_KEY=your_api_key
go run main.go
```

**Expected Output:**
```text
2024/12/01 22:30:00 Starting agent...
2024/12/01 22:30:00 Current message count: 2
Agent Response: The weather in San Francisco is sunny and 25°C.
```
