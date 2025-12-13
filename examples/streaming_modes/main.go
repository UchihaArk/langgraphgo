package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smallnest/langgraphgo/graph"
)

// This example demonstrates Streaming Modes.
// We stream updates from the graph execution.

func main() {
	g := graph.NewStreamingStateGraph()

	g.AddNode("step_1", "step_1", func(ctx context.Context, state any) (any, error) {
		time.Sleep(500 * time.Millisecond) // Simulate work
		return "Result from Step 1", nil
	})

	g.AddNode("step_2", "step_2", func(ctx context.Context, state any) (any, error) {
		time.Sleep(500 * time.Millisecond)
		return "Result from Step 2", nil
	})

	g.SetEntryPoint("step_1")
	g.AddEdge("step_1", "step_2")
	g.AddEdge("step_2", graph.END)

	// Configure for "updates" mode (emit node outputs)
	g.SetStreamConfig(graph.StreamConfig{
		Mode: graph.StreamModeUpdates,
	})

	runnable, err := g.CompileStreaming()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting Stream (Mode: updates)...")
	streamResult := runnable.Stream(context.Background(), nil)

	for event := range streamResult.Events {
		fmt.Printf("[%s] Node: %s, Event: %s, State: %v\n",
			event.Timestamp.Format("15:04:05"),
			event.NodeName,
			event.Event,
			event.State)
	}

	fmt.Println("Stream Finished.")
}
