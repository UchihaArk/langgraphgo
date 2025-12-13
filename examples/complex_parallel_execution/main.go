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
	// and "branch_status" tracks which branches have completed
	schema := graph.NewMapSchema()
	schema.RegisterReducer("results", graph.AppendReducer)
	g.SetSchema(schema)

	// Define Nodes
	g.AddNode("start", "start", func(ctx context.Context, state any) (any, error) {
		fmt.Println("=== Complex Parallel Execution Start ===")
		fmt.Println("[Start] Initiating fan-out to multiple branches...")
		return map[string]any{
			"timestamp": time.Now().Format("15:04:05.000"),
		}, nil
	})

	// ==== Short Branch (1 node) ====
	g.AddNode("short_branch", "short_branch", func(ctx context.Context, state any) (any, error) {
		fmt.Println("\n[Short Branch] Starting execution...")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("[Short Branch] ✓ Completed (fast path)")
		return map[string]any{
			"results": []string{"Short branch result"},
		}, nil
	})

	// ==== Medium Branch (2 nodes) ====
	g.AddNode("medium_branch_1", "medium_branch_1", func(ctx context.Context, state any) (any, error) {
		fmt.Println("\n[Medium Branch - Step 1/2] Processing...")
		time.Sleep(150 * time.Millisecond)
		fmt.Println("[Medium Branch - Step 1/2] ✓ Completed")
		return map[string]any{
			"medium_temp": "intermediate_data",
		}, nil
	})

	g.AddNode("medium_branch_2", "medium_branch_2", func(ctx context.Context, state any) (any, error) {
		fmt.Println("[Medium Branch - Step 2/2] Processing...")
		time.Sleep(150 * time.Millisecond)
		fmt.Println("[Medium Branch - Step 2/2] ✓ Completed")
		return map[string]any{
			"results": []string{"Medium branch result (2 steps)"},
		}, nil
	})

	// ==== Long Branch (3 nodes) ====
	g.AddNode("long_branch_1", "long_branch_1", func(ctx context.Context, state any) (any, error) {
		fmt.Println("\n[Long Branch - Step 1/3] Initial processing...")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("[Long Branch - Step 1/3] ✓ Completed")
		return map[string]any{
			"long_temp_1": "data_from_step1",
		}, nil
	})

	g.AddNode("long_branch_2", "long_branch_2", func(ctx context.Context, state any) (any, error) {
		fmt.Println("[Long Branch - Step 2/3] Advanced processing...")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("[Long Branch - Step 2/3] ✓ Completed")
		return map[string]any{
			"long_temp_2": "data_from_step2",
		}, nil
	})

	g.AddNode("long_branch_3", "long_branch_3", func(ctx context.Context, state any) (any, error) {
		fmt.Println("[Long Branch - Step 3/3] Final processing...")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("[Long Branch - Step 3/3] ✓ Completed")
		return map[string]any{
			"results": []string{"Long branch result (3 steps)"},
		}, nil
	})

	// ==== Aggregator Node ====
	g.AddNode("aggregator", "aggregator", func(ctx context.Context, state any) (any, error) {
		mState := state.(map[string]any)
		results := mState["results"].([]string)

		fmt.Println("\n=== Aggregation Point ===")
		fmt.Printf("[Aggregator] All branches completed!\n")
		fmt.Printf("[Aggregator] Collected %d results:\n", len(results))
		for i, result := range results {
			fmt.Printf("  %d. %s\n", i+1, result)
		}

		return map[string]any{
			"status":        "all_branches_completed",
			"total_results": len(results),
			"final_message": "Complex parallel execution finished successfully",
		}, nil
	})

	// ==== Define Graph Structure ====
	g.SetEntryPoint("start")

	// Fan-out: Start -> All branch entry points
	g.AddEdge("start", "short_branch")
	g.AddEdge("start", "medium_branch_1")
	g.AddEdge("start", "long_branch_1")

	// Medium branch internal flow
	g.AddEdge("medium_branch_1", "medium_branch_2")

	// Long branch internal flow
	g.AddEdge("long_branch_1", "long_branch_2")
	g.AddEdge("long_branch_2", "long_branch_3")

	// Fan-in: All branch endpoints -> Aggregator
	g.AddEdge("short_branch", "aggregator")
	g.AddEdge("medium_branch_2", "aggregator")
	g.AddEdge("long_branch_3", "aggregator")

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

	fmt.Println("=== Complex Parallel Execution Example ===")
	fmt.Println("Graph Structure:")
	fmt.Println("  start")
	fmt.Println("    ├─> short_branch (1 step) ────────────┐")
	fmt.Println("    ├─> medium_branch_1 -> medium_branch_2 ├─> aggregator -> END")
	fmt.Println("    └─> long_branch_1 -> long_branch_2 -> long_branch_3 ─┘")
	fmt.Println()

	startTime := time.Now()
	res, err := runnable.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(startTime)

	fmt.Println("\n=== Execution Complete ===")
	fmt.Printf("Total execution time: %v\n", elapsed)
	fmt.Printf("Final State: %v\n", res)
}
