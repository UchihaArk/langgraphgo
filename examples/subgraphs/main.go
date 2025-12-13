package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// This example demonstrates Subgraphs (Composition).
// We create a "Child" graph and use it as a node in a "Parent" graph.

func main() {
	// 1. Define Child Graph
	child := graph.NewStateGraph()
	child.AddNode("child_process", "child_process", func(ctx context.Context, state any) (any, error) {
		m := state.(map[string]any)
		m["child_trace"] = "visited"
		return m, nil
	})
	child.SetEntryPoint("child_process")
	child.AddEdge("child_process", graph.END)

	// 2. Define Parent Graph
	parent := graph.NewStateGraph()
	parent.AddNode("start", "start", func(ctx context.Context, state any) (any, error) {
		return map[string]any{"parent_trace": "started"}, nil
	})

	// Add Child Graph as a node named "nested_graph"
	if err := parent.AddSubgraph("nested_graph", child); err != nil {
		log.Fatal(err)
	}

	parent.AddNode("end", "end", func(ctx context.Context, state any) (any, error) {
		return map[string]any{"parent_trace": "ended"}, nil
	})

	parent.SetEntryPoint("start")
	parent.AddEdge("start", "nested_graph")
	parent.AddEdge("nested_graph", "end")
	parent.AddEdge("end", graph.END)

	// 3. Compile and Run
	runnable, err := parent.Compile()
	if err != nil {
		log.Fatal(err)
	}

	res, err := runnable.Invoke(context.Background(), map[string]any{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final State: %v\n", res)
	// Expected: Contains traces from both parent and child
}
