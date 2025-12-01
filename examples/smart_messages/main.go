package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

// This example demonstrates how to use the AddMessages reducer for smart message merging.
// It handles ID-based deduplication and upserts, which is crucial for chat applications.

func main() {
	// 1. Create a StateGraph with AddMessages reducer
	// We use the helper NewMessagesStateGraph which pre-configures "messages" key with AddMessages reducer
	g := graph.NewMessagesStateGraph()

	// 2. Define nodes
	// Node A: Simulates a user message
	g.AddNode("user_input", func(ctx context.Context, state interface{}) (interface{}, error) {
		return map[string]interface{}{
			"messages": []llms.MessageContent{
				{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart("Hello, AI!")}},
			},
		}, nil
	})

	// Node B: Simulates an AI response (initially a placeholder)
	g.AddNode("ai_response", func(ctx context.Context, state interface{}) (interface{}, error) {
		// We use a map with "id" to demonstrate upsert capability
		return map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"id":      "msg_123",
					"role":    "ai",
					"content": "Thinking...",
				},
			},
		}, nil
	})

	// Node C: Simulates updating the previous AI response (Upsert)
	g.AddNode("ai_update", func(ctx context.Context, state interface{}) (interface{}, error) {
		// Same ID "msg_123", different content. This should REPLACE the previous message.
		return map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"id":      "msg_123",
					"role":    "ai",
					"content": "Hello! How can I help you today?",
				},
			},
		}, nil
	})

	// 3. Define edges
	g.SetEntryPoint("user_input")
	g.AddEdge("user_input", "ai_response")
	g.AddEdge("ai_response", "ai_update")
	g.AddEdge("ai_update", graph.END)

	// 4. Compile and Run
	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	// Use []interface{} for messages to allow mixed types (structs and maps)
	// This is necessary because we are mixing llms.MessageContent and map[string]interface{} (for IDs)
	res, err := runnable.Invoke(ctx, map[string]interface{}{"messages": []interface{}{}})
	if err != nil {
		log.Fatal(err)
	}

	// 5. Inspect Result
	mRes := res.(map[string]interface{})
	messages := mRes["messages"].([]interface{}) // Note: Type might be mixed due to map input

	fmt.Printf("Total Messages: %d\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("[%d] %v\n", i, msg)
	}

	// Expected Output:
	// [0] User: Hello, AI!
	// [1] AI: Hello! How can I help you today? (The "Thinking..." message was updated)
}
