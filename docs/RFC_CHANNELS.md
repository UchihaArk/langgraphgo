# RFC: Channels Architecture for LangGraphGo

## Summary
This RFC proposes introducing a formal `Channel` architecture to LangGraphGo, aligning it more closely with the Python implementation. Channels provide a flexible and powerful way to manage state updates, synchronization, and data flow between nodes in the graph.

## Motivation
Currently, LangGraphGo uses a `StateSchema` with `Reducer` functions to manage state updates. While effective for simple cases (like `map[string]any` with append logic), it lacks the expressiveness of Python's Channels.

Channels allow for:
- **De-duplication**: Ensuring only unique values are stored.
- **Topic-based Pub/Sub**: Broadcasting values to multiple listeners.
- **Last Value**: Storing only the most recent value (default behavior).
- **History/Windowing**: Keeping a fixed number of past values.
- **Custom Logic**: Defining complex update rules beyond simple reduction.

Formalizing Channels will enable more complex agentic patterns, such as "Swarm" architectures where agents communicate via specific topics, or cyclic graphs with sophisticated state management.

## Proposal

### 1. Define `Channel` Interface

```go
type Channel interface {
    // Update applies a sequence of updates to the channel.
    // It returns the new value of the channel.
    Update(ctx context.Context, updates []any) (any, error)

    // Get returns the current value of the channel.
    Get(ctx context.Context) (any, error)

    // Checkpoint returns a serializable representation of the channel's state.
    Checkpoint() (any, error)

    // Restore restores the channel's state from a checkpoint.
    Restore(checkpoint any) error
}
```

### 2. Standard Channel Implementations

- **`LastValueChannel`**: Stores the last received value. This is the default behavior for most state keys.
- **`BinaryOperatorChannel`**: Applies a binary operator (reducer) to the current value and the update. Equivalent to the current `Reducer` logic.
- **`TopicChannel`**: A pub/sub channel where updates are a list of messages. Useful for "inbox" patterns.
- **`BinopChannel`**: A generic version of `BinaryOperatorChannel` using generics (if Go 1.18+ is fully embraced).

### 3. Integration with `StateGraph`

The `StateGraph` should allow defining the state schema using Channels explicitly.

```go
graph := NewStateGraph()
graph.AddChannel("messages", NewBinaryOperatorChannel(AppendReducer))
graph.AddChannel("user_info", NewLastValueChannel())
```

This replaces or augments the current `MapSchema` approach. `MapSchema` can be refactored to be a collection of Channels.

### 4. Migration Path

1.  Introduce `Channel` interface in `graph` package.
2.  Implement `LastValueChannel` and `BinaryOperatorChannel`.
3.  Refactor `MapSchema` to use Channels internally.
4.  Expose `AddChannel` API on `StateGraph`.
5.  Deprecate `RegisterReducer` in favor of `AddChannel` (or keep it as a shortcut).

## Benefits
- **Parity with Python**: Makes porting graphs easier.
- **Flexibility**: Users can implement custom channels for unique needs.
- **Clarity**: Explicitly defining how each part of the state behaves.

## Drawbacks
- **Complexity**: Adds another abstraction layer.
- **Performance**: Interface overhead (though likely negligible).

## Conclusion
Adopting Channels is a natural evolution for LangGraphGo to support advanced agentic workflows. It provides the necessary primitives for robust state management in complex graphs.
