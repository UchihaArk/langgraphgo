package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// This example demonstrates "Time Travel" / Human-in-the-loop (HITL) workflow.
// We run a graph, interrupt it, update the state manually, and resume.

func main() {
	// 1. Setup Checkpointable Graph
	g := graph.NewCheckpointableStateGraph()

	// Schema with integer reducer
	schema := graph.NewMapSchema()
	schema.RegisterReducer("count", func(curr, new any) (any, error) {
		if curr == nil {
			return new, nil
		}
		return curr.(int) + new.(int), nil
	})
	g.SetSchema(schema)

	// Node A: Adds 1
	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Node A executing...")
		return map[string]any{"count": 1}, nil
	})

	// Node B: Adds 10
	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Node B executing...")
		return map[string]any{"count": 10}, nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", graph.END)

	// Configure interrupt before B
	config := &graph.Config{
		InterruptBefore: []string{"B"},
		Configurable: map[string]any{
			"thread_id": "thread_1",
		},
	}

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 2. Run Initial (Interrupts before B)
	fmt.Println("--- Run 1 (Start) ---")
	res, err := runnable.InvokeWithConfig(ctx, map[string]any{"count": 0}, config)
	// Expect interrupt error or partial result?
	// Invoke returns state at interrupt.
	// Note: Invoke returns (state, error). If interrupted, error is GraphInterrupt.
	if err != nil {
		if _, ok := err.(*graph.GraphInterrupt); ok {
			fmt.Println("Graph Interrupted as expected.")
		} else {
			log.Fatal(err)
		}
	}
	fmt.Printf("State at Interrupt: %v\n", res) // Should be count=1 (0+1)

	// 3. Update State (Human Intervention)
	// We decide to change the count to 100 before B runs.
	fmt.Println("\n--- Human Update ---")
	// UpdateState merges. We want to set it to 100.
	// Since our reducer adds, if we pass 99, 1+99=100.
	// Or if we want to OVERWRITE, we need a different reducer or schema logic.
	// For this example, let's just add 50.
	newConfig, err := runnable.UpdateState(ctx, config, map[string]any{"count": 50}, "human")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("State Updated. New Checkpoint created.")

	// 4. Resume Execution
	// We resume from the new checkpoint.
	// Note: We need to clear InterruptBefore to let it proceed, or use ResumeFrom?
	// If we just Invoke with new config (pointing to new checkpoint), it should continue?
	// But we need to tell it to start at B?
	// The checkpoint knows "Next" nodes? Currently our Checkpoint struct doesn't store Next nodes explicitly.
	// But `Invoke` logic determines next nodes from current.
	// If we resume, we usually need to specify `ResumeFrom` or rely on saved state.
	// For now, let's use `ResumeFrom` = "B".

	resumeConfig := &graph.Config{
		Configurable: newConfig.Configurable, // Use the checkpoint ID from UpdateState
		ResumeFrom:   []string{"B"},
	}

	fmt.Println("\n--- Run 2 (Resume) ---")
	finalRes, err := runnable.InvokeWithConfig(ctx, nil, resumeConfig) // State loaded from checkpoint
	if err != nil {
		log.Fatal(err)
	}

	// Final result should be:
	// Initial: 0
	// A: +1 -> 1
	// Update: +50 -> 51
	// B: +10 -> 61
	fmt.Printf("Final Result: %v\n", finalRes)
}
