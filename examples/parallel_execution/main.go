package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smallnest/langgraphgo/graph"
)

func main() {
	// Create a new state graph
	g := graph.NewStateGraph()

	// Define Schema
	// We use a map schema where "results" is a list that accumulates values
	schema := graph.NewMapSchema()
	schema.RegisterReducer("results", graph.AppendReducer)
	g.SetSchema(schema)

	// Define Nodes
	g.AddNode("start", "start", func(ctx context.Context, state any) (any, error) {
		fmt.Println("[Start] Starting execution...")
		return map[string]any{
			"status": "started",
		}, nil
	})

	g.AddNode("branch_a", "branch_a", func(ctx context.Context, state any) (any, error) {
		fmt.Println("  [Branch A] Working...")
		time.Sleep(100 * time.Millisecond)
		return map[string]any{
			"results": []string{"Result from A"},
		}, nil
	})

	g.AddNode("branch_b", "branch_b", func(ctx context.Context, state any) (any, error) {
		fmt.Println("  [Branch B] Working...")
		time.Sleep(200 * time.Millisecond)
		return map[string]any{
			"results": []string{"Result from B"},
		}, nil
	})

	g.AddNode("branch_c", "branch_c", func(ctx context.Context, state any) (any, error) {
		fmt.Println("  [Branch C] Working...")
		time.Sleep(150 * time.Millisecond)
		return map[string]any{
			"results": []string{"Result from C"},
		}, nil
	})

	g.AddNode("aggregator", "aggregator", func(ctx context.Context, state any) (any, error) {
		mState := state.(map[string]any)
		results := mState["results"].([]string)
		fmt.Printf("[Aggregator] Collected %d results: %v\n", len(results), results)
		return map[string]any{
			"status": "finished",
		}, nil
	})

	// Define Edges
	g.SetEntryPoint("start")

	// Fan-out: Start -> A, B, C
	g.AddEdge("start", "branch_a")
	g.AddEdge("start", "branch_b")
	g.AddEdge("start", "branch_c")

	// Fan-in: A, B, C -> Aggregator
	g.AddEdge("branch_a", "aggregator")
	g.AddEdge("branch_b", "aggregator")
	g.AddEdge("branch_c", "aggregator")

	g.AddEdge("aggregator", graph.END)

	// Compile
	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Execute
	initialState := map[string]any{
		"results": []string{},
	}

	fmt.Println("=== Parallel Execution Example ===")
	res, err := runnable.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final State: %v\n", res)
}
