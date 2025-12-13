package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// UserConfig represents custom configuration passed at runtime
type UserConfig struct {
	UserID    string
	RequestID string
	Verbose   bool
}

func main() {
	// Create a new graph
	g := graph.NewStateGraph()

	// Define a node that uses the configuration
	g.AddNode("process", "process", func(ctx context.Context, state any) (any, error) {
		// Retrieve config from context
		config := graph.GetConfig(ctx)

		// Access standard config fields
		if threadID, ok := config.Configurable["thread_id"]; ok {
			fmt.Printf("[Node] Processing for Thread ID: %v\n", threadID)
		}

		// Access custom metadata if available
		if userID, ok := config.Metadata["user_id"]; ok {
			fmt.Printf("[Node] User ID: %v\n", userID)
		}

		if reqID, ok := config.Metadata["request_id"]; ok {
			fmt.Printf("[Node] Request ID: %v\n", reqID)
		}

		return fmt.Sprintf("Processed for %v", config.Metadata["user_id"]), nil
	})

	g.SetEntryPoint("process")
	g.AddEdge("process", graph.END)

	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Prepare runtime configuration
	config := &graph.Config{
		Configurable: map[string]any{
			"thread_id": "thread-123",
		},
		Metadata: map[string]any{
			"user_id":    "alice",
			"request_id": "req-456",
			"verbose":    true,
		},
	}

	fmt.Println("=== Running with Configuration ===")
	res, err := runnable.InvokeWithConfig(context.Background(), "start", config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result: %v\n", res)
}
