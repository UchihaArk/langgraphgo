package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/smallnest/langgraphgo/graph"
)

func main() {
	// Create a temporary directory for checkpoints
	checkpointDir := "./checkpoints"
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(checkpointDir) // Cleanup after run

	fmt.Printf("Using checkpoint directory: %s\n", checkpointDir)

	// Initialize FileCheckpointStore
	store, err := graph.NewFileCheckpointStore(checkpointDir)
	if err != nil {
		log.Fatalf("Failed to create checkpoint store: %v", err)
	}

	// Define a simple graph
	g := graph.NewCheckpointableStateGraph()

	// Add nodes that update state
	g.AddNode("first", "first", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing 'first' node")
		m := state.(map[string]any)
		m["step1"] = "completed"
		return m, nil
	})

	g.AddNode("second", "second", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing 'second' node")
		m := state.(map[string]any)
		m["step2"] = "completed"
		return m, nil
	})

	g.AddEdge("first", "second")
	g.AddEdge("second", graph.END)
	g.SetEntryPoint("first")

	// Set checkpoint config
	g.SetCheckpointConfig(graph.CheckpointConfig{
		Store:    store,
		AutoSave: true,
	})

	// Compile implementation
	runnable, err := g.CompileCheckpointable()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	// Run the graph
	ctx := context.Background()
	initialState := map[string]any{
		"input": "start",
	}

	// Thread ID helps group checkpoints for a specific conversation/execution
	config := &graph.Config{
		Configurable: map[string]any{
			"thread_id": "thread_1",
		},
	}

	fmt.Println("--- Starting Graph Execution ---")
	res, err := runnable.InvokeWithConfig(ctx, initialState, config)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	fmt.Printf("Final Result: %v\n", res)

	// Verify checkpoints were saved
	fmt.Println("\n--- Verifying Checkpoints ---")
	files, err := os.ReadDir(checkpointDir)
	if err != nil {
		log.Fatalf("Failed to read checkpoint directory: %v", err)
	}

	count := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			fmt.Printf("Found checkpoint file: %s\n", file.Name())
			count++
		}
	}

	if count > 0 {
		fmt.Printf("Successfully saved %d checkpoints to %s\n", count, checkpointDir)
	} else {
		log.Fatal("No checkpoints found!")
	}

	// Demonstrate listing via store
	fmt.Println("\n--- Listing Checkpoints from Store ---")
	checkpoints, err := store.List(ctx, "thread_1")
	if err != nil {
		log.Fatalf("Failed to list checkpoints: %v", err)
	}
	for _, cp := range checkpoints {
		fmt.Printf("Checkpoint ID: %s, Node: %s, Version: %d\n", cp.ID, cp.NodeName, cp.Version)
	}
}
