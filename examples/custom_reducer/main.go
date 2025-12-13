package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// Custom Set Reducer
// Merges two values and removes duplicates
func SetReducer(current any, new any) (any, error) {
	// Initialize set with current values
	set := make(map[string]bool)

	// Handle current state
	if current != nil {
		if currentList, ok := current.([]string); ok {
			for _, item := range currentList {
				set[item] = true
			}
		}
	}

	// Merge new values
	if newList, ok := new.([]string); ok {
		for _, item := range newList {
			set[item] = true
		}
	} else if item, ok := new.(string); ok {
		set[item] = true
	}

	// Convert back to slice
	result := make([]string, 0, len(set))
	for item := range set {
		result = append(result, item)
	}

	return result, nil
}

func main() {
	g := graph.NewStateGraph()

	// Define Schema with Custom Reducer
	schema := graph.NewMapSchema()
	schema.RegisterReducer("tags", SetReducer)
	g.SetSchema(schema)

	// Define Nodes
	g.AddNode("start", "start", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"tags": []string{"initial"},
		}, nil
	})

	g.AddNode("tagger_a", "tagger_a", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"tags": []string{"go", "langgraph"},
		}, nil
	})

	g.AddNode("tagger_b", "tagger_b", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"tags": []string{"ai", "agent", "go"}, // "go" is duplicate
		}, nil
	})

	g.SetEntryPoint("start")
	g.AddEdge("start", "tagger_a")
	g.AddEdge("start", "tagger_b")
	g.AddEdge("tagger_a", graph.END)
	g.AddEdge("tagger_b", graph.END)

	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Custom Reducer Example (Set Merge) ===")
	res, err := runnable.Invoke(context.Background(), map[string]any{
		"tags": []string{},
	})
	if err != nil {
		log.Fatal(err)
	}

	mState := res.(map[string]any)
	fmt.Printf("Final Tags: %v\n", mState["tags"])
}
