# Command API Example

## Background

In complex agent workflows, static graph definitions (edges defined at compile time) are sometimes insufficient. An agent might need to decide *dynamically* where to go next based on the result of a tool call or an LLM decision, potentially skipping intermediate steps or choosing a path that wasn't explicitly wired. The **Command API** provides a mechanism for nodes to return control flow instructions directly.

## Features

*   **Dynamic Routing (`Goto`)**: Allows a node to specify the next node(s) to execute, overriding the graph's static edges.
*   **State Updates (`Update`)**: Allows a node to update the graph state simultaneously with the routing instruction.
*   **Flexibility**: Enables patterns like "early exit", "skip steps", or "dynamic looping" without complex conditional edges.

## Implementation Principle

The `Command` struct is defined as:

```go
type Command struct {
    Update any // State update to apply
    Goto   any // Next node(s) to execute (string or []string)
}
```

When a node returns a `*Command` object:
1.  The `Update` payload is merged into the graph state using the defined Schema/Reducer.
2.  The `Goto` field is inspected. If present, the graph execution engine ignores the static outgoing edges of the current node and instead schedules the node(s) specified in `Goto`.

## Code Walkthrough

In `main.go`:

1.  **Router Node**:
    ```go
    g.AddNode("router", func(ctx context.Context, state any) (any, error) {
        // ... logic to check count ...
        if count > 5 {
            // Dynamic Goto: Skip "process" and go straight to "end_high"
            return &graph.Command{
                Update: map[string]any{"status": "high"},
                Goto:   "end_high",
            }, nil
        }
        // ...
    })
    ```
    This node checks the state. If `count > 5`, it returns a `Command` that directs the flow immediately to `end_high`, bypassing the `process` node.

2.  **Execution**:
    The example runs two cases. In Case 2 (`count=10`), you will see that the "process" node is never executed, proving the dynamic routing worked.

## How to Run

```bash
go run main.go
```
