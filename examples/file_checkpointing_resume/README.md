# Resuming from Checkpoint Example

This example demonstrates how to resume graph execution from a specific point using `FileCheckpointStore`.

## Overview

Complex workflows may need to be interrupted (e.g., for human approval) or might fail due to external factors. Checkpointing allows you to save the state of the graph and resume execution from the last saved state, effectively skipping previously completed steps.

## Features Demonstrated

1.  **Phase 1: Interrupted/Partial Execution**
    - Runs a multi-step graph (`step1` -> `step2` -> `step3`).
    - Demonstrates configuring an interruption (or simulating one).
    - Automatically saves checkpoints at each step.

2.  **Phase 2: Resuming Execution**
    - Identifies the latest checkpoint for a specific thread.
    - Loads the preserved state.
    - Re-initializes the graph with `ResumeFrom` configuration.
    - Continues execution from the next logical step (`step3`), skipping re-execution of `step1` and `step2`.

## Running the Example

```bash
cd examples/file_checkpointing_resume
go run main.go
```

## Expected Output

You will see two phases of execution:
1.  **Phase 1**: Executes Step 1 and Step 2.
2.  **Phase 2**: Detects the saved state and immediately executes Step 3, completing the workflow.

## Key Logic

```go
// 1. Load latest checkpoint
checkpoints, _ := store.List(ctx, threadID)
latestCP := checkpoints[len(checkpoints)-1]

// 2. Configure resume
config := &graph.Config{
    Configurable: map[string]interface{}{"thread_id": threadID},
    ResumeFrom:   []string{"step3"}, // Resume starting at step3
}

// 3. Invoke with preserved state
result, err := runnable.InvokeWithConfig(ctx, latestCP.State, config)
```
