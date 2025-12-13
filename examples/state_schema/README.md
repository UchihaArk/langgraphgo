# State Schema Example

This example demonstrates how to use **State Schema** and **Reducers** in LangGraphGo to manage complex state updates.

## 1. Background

In LangGraph, state is not just a simple variable that gets overwritten at each step. It is a structured object where different fields can have different update behaviors. This concept corresponds to `TypedDict` and `Annotated` in the Python library.

For example:
- A list of messages should typically be **appended** to.
- A counter should be **incremented**.
- A status flag should be **overwritten**.

## 2. Key Concepts

- **StateSchema**: Defines the structure of the state and how updates are merged. `graph.MapSchema` is the most common implementation.
- **Reducer**: A function that takes the current value and a new value, and returns the merged value.
  - `AppendReducer`: Appends new items to a list.
  - `OverwriteReducer`: Replaces the old value with the new one (default).
  - **Custom Reducer**: You can define your own logic (e.g., `SumReducer`).

## 3. How It Works

1.  **Define Schema**: We create a `MapSchema` and register reducers for specific keys.
    - `count`: Uses `SumReducer` (custom) to add values.
    - `logs`: Uses `AppendReducer` to accumulate strings.
    - `status`: Uses default overwrite behavior.
2.  **Nodes**: Each node returns a partial state update (a map with some keys).
3.  **Execution**: When a node finishes, the runtime uses the Schema to merge the returned partial state into the global state.

## 4. Code Highlights

### Defining Custom Reducer
```go
func SumReducer(current, new any) (any, error) {
    // ... logic to add integers ...
    return c + n, nil
}
```

### Configuring Schema
```go
schema := graph.NewMapSchema()
schema.RegisterReducer("count", SumReducer)
schema.RegisterReducer("logs", graph.AppendReducer)
g.SetSchema(schema)
```

## 5. Running the Example

```bash
go run main.go
```

**Expected Output:**
```text
--- Final State ---
Count (Sum): 6
Logs (Append): [Start Processed by A Processed by B Processed by C]
Status (Overwrite): Completed
```
