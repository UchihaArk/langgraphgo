# Time Travel / HITL Example

## Background

**Human-in-the-loop (HITL)** is a critical pattern for reliable Agent systems. It allows humans to approve actions, correct errors, or provide guidance before the Agent proceeds. **Time Travel** extends this by allowing you to revisit past states, modify them ("what if?"), and fork execution from that point. This is essential for debugging and for building collaborative AI applications.

## Features

*   **Interrupts**: Pause execution at defined checkpoints (`InterruptBefore`).
*   **State Persistence**: Checkpoints store the complete state history.
*   **State Editing (`UpdateState`)**: Allows modifying the state of a paused graph. This effectively creates a new branch of history.
*   **Resuming**: Continue execution from the modified state.

## Implementation Principle

1.  **Checkpointing**: The `CheckpointableRunnable` wraps the graph execution. It uses a `CheckpointStore` to save the state after every step (or as configured).
2.  **Interrupts**: Before executing a node, the engine checks if the node is in the `InterruptBefore` list. If so, it saves the state and returns a `GraphInterrupt` error.
3.  **UpdateState**:
    *   Loads the latest checkpoint.
    *   Merges the user-provided values using the graph's Schema.
    *   Saves a **new** checkpoint with the updated state and incremented version.
    *   Returns a new `Config` pointing to this new checkpoint.
4.  **Resuming**: When `Invoke` is called with the new Config, it loads the state from the new checkpoint and continues execution.

## Code Walkthrough

In `main.go`:

1.  **Interrupt Config**:
    ```go
    config := &graph.Config{
        InterruptBefore: []string{"B"},
        // ...
    }
    ```
    Tells the graph to stop before executing Node B.

2.  **Run 1**:
    Executes Node A (Count=1) and stops.

3.  **Human Intervention**:
    ```go
    runnable.UpdateState(ctx, config, map[string]any{"count": 50}, "human")
    ```
    We manually set the count to 50. The system merges this (depending on reducer, here we assume overwrite or addition logic tailored for the example).

4.  **Run 2**:
    Resumes. Node B runs with the *new* state (50 + 1 from A? Or just 50 if overwritten). In the example logic, we see the final result reflects the manual change.

## How to Run

```bash
go run main.go
```
