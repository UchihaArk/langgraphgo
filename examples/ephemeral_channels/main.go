package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// This example demonstrates Ephemeral Channels (Values).
// "temp_data" is an ephemeral value that is cleared after each step.

func main() {
	g := graph.NewStateGraph()

	schema := graph.NewMapSchema()
	// Register "temp_data" as an ephemeral channel (isEphemeral = true)
	schema.RegisterChannel("temp_data", graph.OverwriteReducer, true)
	// Register "history" as a persistent channel
	schema.RegisterReducer("history", graph.AppendReducer)
	g.SetSchema(schema)

	// Node A: Produces temporary data
	g.AddNode("producer", "producer", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"temp_data": "secret_code_123",
			"history":   []string{"producer_ran"},
		}, nil
	})

	// Node B: Consumes temporary data (if available)
	// Since A -> B is a step transition, and temp_data is ephemeral,
	// B should NOT see temp_data if it runs in the NEXT step.
	// Wait, if A and B run sequentially, they are in different steps?
	// In LangGraph, a "step" usually corresponds to a super-step (parallel execution of nodes).
	// If A -> B, A runs in Step 1. Step 1 ends. Cleanup happens. B runs in Step 2.
	// So B should NOT see "temp_data".
	g.AddNode("consumer", "consumer", func(ctx context.Context, state any) (any, error) {
		m := state.(map[string]any)

		temp, ok := m["temp_data"]
		if ok {
			fmt.Printf("Consumer saw temp_data: %v\n", temp)
		} else {
			fmt.Println("Consumer did NOT see temp_data (Correct for ephemeral)")
		}

		return map[string]any{
			"history": []string{"consumer_ran"},
		}, nil
	})

	g.SetEntryPoint("producer")
	g.AddEdge("producer", "consumer")
	g.AddEdge("consumer", graph.END)

	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	res, err := runnable.Invoke(context.Background(), map[string]any{})
	if err != nil {
		log.Fatal(err)
	}

	mRes := res.(map[string]any)
	fmt.Printf("Final History: %v\n", mRes["history"])
	// "temp_data" should also be gone from final result
	if _, ok := mRes["temp_data"]; !ok {
		fmt.Println("Final state does not contain temp_data")
	}
}
