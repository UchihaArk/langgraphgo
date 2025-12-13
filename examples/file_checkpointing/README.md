# File Checkpointing Example

This example demonstrates how to use `FileCheckpointStore` for persistent state management in LangGraphGo.

## Overview

Unlike the in-memory checkpoint store which loses data when the application exits, the `FileCheckpointStore` saves execution state to the local file system. This allows for:
1. **Persistence** - State survives application restarts
2. **Inspection** - Checkpoints are saved as readable JSON files
3. **Recovery** - Workflows can be resumed from specific points (future feature)

## Features Demonstrated

- **Initialization**: Creating a temporary directory and initializing `FileCheckpointStore`.
- **Configuration**: Setting up a `CheckpointableStateGraph` with file storage.
- **Execution**: Running a graph that saves state at each step.
- **Verification**: Listing saved checkpoints from the file system.

## Running the Example

```bash
cd examples/file_checkpointing
go run main.go
```

## Expected Output

The example will:
1. Create a `./checkpoints` directory.
2. Execute a simple 2-step workflow.
3. Save state after each step to JSON files in `./checkpoints`.
4. Verify the existence of these files.
5. List the checkpoints using the store API.
6. Clean up the directory upon exit (though a real app would likely keep them).

## Code Snippet

```go
// Initialize store
store, err := graph.NewFileCheckpointStore("./checkpoints")

// Configure graph
g := graph.NewCheckpointableStateGraph()
g.SetCheckpointConfig(graph.CheckpointConfig{
    Store:    store,
    AutoSave: true,
})
```
