package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// SumReducer adds the new integer value to the current one.
func SumReducer(current, new any) (any, error) {
	if current == nil {
		return new, nil
	}
	c, ok1 := current.(int)
	n, ok2 := new.(int)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("expected int, got %T and %T", current, new)
	}
	return c + n, nil
}

func main() {
	// 1. Create a new StateGraph
	g := graph.NewStateGraph()

	// 2. Define the State Schema
	// We use MapSchema to define how different keys in the state map should be updated.
	schema := graph.NewMapSchema()

	// "count" uses SumReducer: values will be added together
	schema.RegisterReducer("count", SumReducer)

	// "logs" uses AppendReducer: new values will be appended to the list
	schema.RegisterReducer("logs", graph.AppendReducer)

	// "status" uses the default OverwriteReducer (no need to register explicitly, but good for clarity)
	// Any key without a registered reducer defaults to overwrite.

	g.SetSchema(schema)

	// 3. Define Nodes
	g.AddNode("node_a", "node_a", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing Node A")
		// Return partial state update
		return map[string]any{
			"count":  1,
			"logs":   []string{"Processed by A"},
			"status": "In Progress (A)",
		}, nil
	})

	g.AddNode("node_b", "node_b", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing Node B")
		return map[string]any{
			"count":  2,
			"logs":   []string{"Processed by B"},
			"status": "In Progress (B)",
		}, nil
	})

	g.AddNode("node_c", "node_c", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing Node C")
		return map[string]any{
			"count":  3,
			"logs":   []string{"Processed by C"},
			"status": "Completed",
		}, nil
	})

	// 4. Define Edges
	g.SetEntryPoint("node_a")
	g.AddEdge("node_a", "node_b")
	g.AddEdge("node_b", "node_c")
	g.AddEdge("node_c", graph.END)

	// 5. Compile
	app, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// 6. Invoke
	// Initial state
	initialState := map[string]any{
		"count":  0,
		"logs":   []string{"Start"},
		"status": "Init",
	}

	fmt.Println("--- Starting Execution ---")
	result, err := app.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	// 7. Inspect Result
	fmt.Println("\n--- Final State ---")
	mState := result.(map[string]any)
	fmt.Printf("Count (Sum): %v\n", mState["count"])         // Should be 0 + 1 + 2 + 3 = 6
	fmt.Printf("Logs (Append): %v\n", mState["logs"])        // Should be ["Start", "Processed by A", "Processed by B", "Processed by C"]
	fmt.Printf("Status (Overwrite): %v\n", mState["status"]) // Should be "Completed"
}
