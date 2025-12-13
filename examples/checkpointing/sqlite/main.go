package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/smallnest/langgraphgo/checkpoint/sqlite"
	"github.com/smallnest/langgraphgo/graph"
)

type ProcessState struct {
	Step    int
	Data    string
	History []string
}

func main() {
	// Check for Sqlite DB path
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = "./checkpoints.db"
	}
	fmt.Printf("Using SQLite database at: %s\n", dbPath)

	// Create a checkpointable graph
	g := graph.NewCheckpointableStateGraph()

	// Initialize Sqlite Checkpoint Store
	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
		Path:      dbPath,
		TableName: "example_checkpoints",
	})
	if err != nil {
		panic(fmt.Errorf("failed to create sqlite store: %w", err))
	}
	defer store.Close()

	// Configure checkpointing
	config := graph.CheckpointConfig{
		Store:          store,
		AutoSave:       true,
		SaveInterval:   2 * time.Second,
		MaxCheckpoints: 5,
	}
	g.SetCheckpointConfig(config)

	// Add processing nodes
	g.AddNode("step1", "step1", func(ctx context.Context, state any) (any, error) {
		s := state.(ProcessState)
		s.Step = 1
		s.Data = s.Data + " → Step1"
		s.History = append(s.History, "Completed Step 1")
		fmt.Println("Executing Step 1...")
		time.Sleep(500 * time.Millisecond) // Simulate work
		return s, nil
	})

	g.AddNode("step2", "step2", func(ctx context.Context, state any) (any, error) {
		s := state.(ProcessState)
		s.Step = 2
		s.Data = s.Data + " → Step2"
		s.History = append(s.History, "Completed Step 2")
		fmt.Println("Executing Step 2...")
		time.Sleep(500 * time.Millisecond) // Simulate work
		return s, nil
	})

	g.AddNode("step3", "step3", func(ctx context.Context, state any) (any, error) {
		s := state.(ProcessState)
		s.Step = 3
		s.Data = s.Data + " → Step3"
		s.History = append(s.History, "Completed Step 3")
		fmt.Println("Executing Step 3...")
		time.Sleep(500 * time.Millisecond) // Simulate work
		return s, nil
	})

	// Build the pipeline
	g.SetEntryPoint("step1")
	g.AddEdge("step1", "step2")
	g.AddEdge("step2", "step3")
	g.AddEdge("step3", graph.END)

	// Compile checkpointable runnable
	runnable, err := g.CompileCheckpointable()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	initialState := ProcessState{
		Step:    0,
		Data:    "Start",
		History: []string{"Initialized"},
	}

	fmt.Println("=== Starting execution with SQLite checkpointing ===")

	// Execute with automatic checkpointing
	result, err := runnable.Invoke(ctx, initialState)
	if err != nil {
		panic(err)
	}

	finalState := result.(ProcessState)
	fmt.Printf("\n=== Execution completed ===\n")
	fmt.Printf("Final Step: %d\n", finalState.Step)
	fmt.Printf("Final Data: %s\n", finalState.Data)
	fmt.Printf("History: %v\n", finalState.History)

	// List saved checkpoints
	checkpoints, err := runnable.ListCheckpoints(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n=== Created %d checkpoints in SQLite ===\n", len(checkpoints))
	for i, cp := range checkpoints {
		fmt.Printf("Checkpoint %d: ID=%s, Time=%v\n", i+1, cp.ID, cp.Timestamp)
	}

	// Demonstrate resuming from a checkpoint
	if len(checkpoints) > 1 {
		fmt.Printf("\n=== Resuming from checkpoint %s ===\n", checkpoints[1].ID)
		resumedState, err := runnable.ResumeFromCheckpoint(ctx, checkpoints[1].ID)
		if err != nil {
			fmt.Printf("Error resuming: %v\n", err)
		} else {
			// Since data is loaded from JSON, it comes back as map[string]any
			// We need to convert it back to ProcessState
			var resumed ProcessState

			// Helper to convert map to struct via JSON
			// In a real app, you might use mapstructure or similar
			importJSON, _ := json.Marshal(resumedState)
			json.Unmarshal(importJSON, &resumed)

			fmt.Printf("Resumed at Step: %d\n", resumed.Step)
			fmt.Printf("Resumed Data: %s\n", resumed.Data)
			fmt.Printf("Resumed History: %v\n", resumed.History)
		}
	}
}
