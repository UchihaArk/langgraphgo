# ReAct Agent Example

This example demonstrates how to use the pre-built **ReAct Agent** factory in LangGraphGo.

## 1. Background

**ReAct** (Reasoning and Acting) is a fundamental pattern for building intelligent agents. Instead of just generating text, the model:
1.  **Reasons** about the problem.
2.  **Acts** by calling external tools (e.g., calculator, search engine).
3.  **Observes** the output of those tools.
4.  Repeats the process until it can answer the user's question.

LangGraphGo simplifies this by providing a `prebuilt.CreateReactAgent` function that automatically constructs the graph for this loop.

## 2. Key Concepts

- **Agent Node**: The node that calls the LLM. It receives the conversation history and decides whether to call a tool or provide a final answer.
- **Tools Node**: The node that executes the tools requested by the Agent Node.
- **Conditional Edge**: Logic that checks the LLM's output. If it contains a tool call, it routes to the Tools Node; otherwise, it ends execution.
- **Tool Interface**: Tools must implement the `langchaingo/tools.Tool` interface (`Name`, `Description`, `Call`).

## 3. How It Works

1.  **Define Tools**: We define a simple `CalculatorTool` that can perform basic arithmetic.
2.  **Create Agent**: We pass the LLM and the list of tools to `prebuilt.CreateReactAgent`. This function builds the state graph internally.
3.  **Invoke**: We send a query ("What is 25 * 4?") to the agent.
4.  **Execution Flow**:
    - The LLM sees the query and the available `calculator` tool.
    - It decides to call `calculator` with input `25 * 4`.
    - The graph routes to the tool executor.
    - The tool returns `100.000000`.
    - The LLM receives this observation and generates the final answer: "The answer is 100."

## 4. Code Highlights

### Defining a Tool
```go
type CalculatorTool struct{}
func (t CalculatorTool) Name() string { return "calculator" }
func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
    // ... implementation ...
}
```

### Creating the Agent
```go
inputTools := []tools.Tool{CalculatorTool{}}
// Automatically builds the graph with Agent and Tool nodes
agent, err := prebuilt.CreateReactAgent(model, inputTools)
```

### Running the Agent
```go
initialState := map[string]any{
    "messages": []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeHuman, "What is 25 * 4?"),
    },
}
res, err := agent.Invoke(ctx, initialState)
```

## 5. Running the Example

```bash
export OPENAI_API_KEY=your_key
go run main.go
```

**Expected Output:**
```text
User: What is 25 * 4?
Agent: The result of 25 * 4 is 100.
```
