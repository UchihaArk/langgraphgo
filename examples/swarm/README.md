# Swarm Pattern Example

This example demonstrates how to implement the **Swarm** pattern for multi-agent collaboration using LangGraphGo.

## 1. Background

The **Swarm** pattern (inspired by OpenAI's Swarm framework) is a decentralized approach to multi-agent orchestration. Unlike the **Supervisor** pattern where a central agent routes tasks, Swarm allows agents to hand off execution directly to one another. This mimics how teams of experts work together—passing tasks back and forth based on their specific capabilities.

## 2. Key Concepts

- **Handoff**: The mechanism by which one agent transfers control to another. In this example, it's implemented as a tool (`handoff`) that agents can call.
- **Router**: A conditional edge function that inspects the state (specifically a `next` field) to determine which agent should run next.
- **Shared State**: All agents share the same conversation history (`messages`), allowing them to see what has happened previously.

## 3. How It Works

1.  **Define Agents**: We define two agents: `Researcher` and `Writer`.
2.  **Handoff Tool**: We provide both agents with a `handoff` tool. This tool takes a `to` argument (e.g., "Writer").
3.  **Execution Loop**:
    - The `Researcher` starts. It receives the user's request.
    - If it needs to write a report, it calls the `handoff` tool with `to="Writer"`.
    - The node logic captures this tool call, updates the `next` field in the state to "Writer", and returns.
4.  **Routing**: The `router` function sees that `next` is "Writer" and directs the flow to the `Writer` node.
5.  **Completion**: If an agent decides the task is done, it returns a normal response (no tool call), and the router directs to `END`.

## 4. Code Highlights

### Handoff Tool Definition
```go
var HandoffTool = llms.Tool{
    Name: "handoff",
    // ...
    Parameters: map[string]any{
        "properties": map[string]any{
            "to": map[string]any{
                "enum": []string{"Researcher", "Writer"},
            },
        },
    },
}
```

### Handling Handoffs in Nodes
```go
if tc.FunctionCall.Name == "handoff" {
    // ... parse args ...
    return map[string]any{
        // Add tool call and response to history
        "messages": []llms.MessageContent{ ... },
        // Set the next agent
        "next": args.To,
    }, nil
}
```

### Conditional Routing
```go
router := func(ctx context.Context, state any) string {
    mState := state.(map[string]any)
    next := mState["next"].(string)
    if next == "" || next == "END" {
        return graph.END
    }
    return next // Returns "Researcher" or "Writer"
}
workflow.AddConditionalEdge("Researcher", router)
workflow.AddConditionalEdge("Writer", router)
```

## 5. Running the Example

```bash
export OPENAI_API_KEY=your_key
go run main.go
```

**Note**: This example requires an OpenAI API key as it uses function calling.
