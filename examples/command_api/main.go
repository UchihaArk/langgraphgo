package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// This example demonstrates the Command API for dynamic control flow.
// Nodes can return a Command object to update state and determine the next node dynamically.

func main() {
	g := graph.NewStateGraph()

	// Define schema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("count", graph.OverwriteReducer)
	g.SetSchema(schema)

	// Node A: Decides where to go based on state
	g.AddNode("router", "router", func(ctx context.Context, state any) (any, error) {
		m := state.(map[string]any)
		count := m["count"].(int)

		if count > 5 {
			// Dynamic Goto: Skip "process" and go straight to "end_high"
			return &graph.Command{
				Update: map[string]any{"status": "high"},
				Goto:   "end_high",
			}, nil
		}

		// Normal flow: Update state and let static edges handle it (or Goto "process")
		return &graph.Command{
			Update: map[string]any{"status": "normal"},
			Goto:   "process",
		}, nil
	})

	g.AddNode("process", "process", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing Process Node")
		return map[string]any{"processed": true}, nil
	})

	g.AddNode("end_high", "end_high", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing End High Node")
		return map[string]any{"final": "high value"}, nil
	})

	g.SetEntryPoint("router")
	// Note: We don't strictly need static edges from "router" if it always returns a Command with Goto.
	// But for "process", we need an edge to END.
	g.AddEdge("process", graph.END)
	g.AddEdge("end_high", graph.END)

	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Case 1: Normal Flow
	fmt.Println("--- Case 1: Count = 3 ---")
	res, _ := runnable.Invoke(context.Background(), map[string]any{"count": 3})
	fmt.Printf("Result: %v\n", res)

	// Case 2: High Value Flow (Skip Process)
	fmt.Println("\n--- Case 2: Count = 10 ---")
	res, _ = runnable.Invoke(context.Background(), map[string]any{"count": 10})
	fmt.Printf("Result: %v\n", res)
}
