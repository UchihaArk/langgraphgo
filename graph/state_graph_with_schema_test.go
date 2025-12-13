package graph

import (
	"context"
	"testing"
)

func TestNewMessageGraph(t *testing.T) {
	g := NewMessageGraph()

	// Verify schema is initialized
	if g.Schema == nil {
		t.Fatal("Schema should be initialized")
	}

	// Verify messages reducer is registered
	mapSchema, ok := g.Schema.(*MapSchema)
	if !ok {
		t.Fatal("Schema should be a MapSchema")
	}

	if mapSchema.Reducers == nil {
		t.Fatal("Reducers map should be initialized")
	}

	if _, exists := mapSchema.Reducers["messages"]; !exists {
		t.Fatal("messages reducer should be registered")
	}

	// Test that the schema works with AddMessages
	g.AddNode("node1", "Test node", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"messages": []map[string]any{
				{"role": "assistant", "content": "Hello"},
			},
		}, nil
	})

	g.AddEdge("node1", END)
	g.SetEntryPoint("node1")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Execute with initial state
	initialState := map[string]any{
		"messages": []map[string]any{
			{"role": "user", "content": "Hi"},
		},
	}

	result, err := runnable.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke: %v", err)
	}

	// Verify messages were merged
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
	}

	messages, ok := resultMap["messages"].([]map[string]any)
	if !ok {
		t.Fatal("messages should be a slice")
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	t.Log("NewStateGraphWithSchema test passed!")
}
