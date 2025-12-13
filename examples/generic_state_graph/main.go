package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// UserRequest represents the input to our workflow
type UserRequest struct {
	Name    string
	Age     int
	Country string
}

// WorkflowState represents the state of our workflow with full type safety
type WorkflowState struct {
	Request       UserRequest
	IsAdult       bool
	IsEligible    bool
	Notifications []string
	Result        string
}

func main() {
	fmt.Println("=== Generic StateGraph Example ===")

	// Example 1: Simple type-safe graph
	example1_SimpleGraph()

	fmt.Println("\n" + repeat("=", 50) + "\n")

	// Example 2: Conditional routing with type safety
	example2_ConditionalRouting()

	fmt.Println("\n" + repeat("=", 50) + "\n")

	// Example 3: Using Schema for complex state merging
	example3_WithSchema()
}

// Example 1: Simple type-safe graph
func example1_SimpleGraph() {
	fmt.Println("Example 1: Simple Type-Safe Graph")
	fmt.Println("-----------------------------------")

	// Create a generic graph with WorkflowState type
	g := graph.NewStateGraphTyped[WorkflowState]()

	// Add nodes with full type safety - no type assertions needed!
	g.AddNode("check_age", "Check if user is adult", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
		fmt.Printf("Checking age for %s (%d years old)\n", state.Request.Name, state.Request.Age)
		state.IsAdult = state.Request.Age >= 18
		state.Notifications = append(state.Notifications, fmt.Sprintf("Age check: %s is adult=%v", state.Request.Name, state.IsAdult))
		return state, nil
	})

	g.AddNode("check_eligibility", "Check eligibility", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
		fmt.Printf("Checking eligibility for %s\n", state.Request.Name)
		// Type-safe field access - no casting needed!
		state.IsEligible = state.IsAdult && state.Request.Country == "USA"
		state.Notifications = append(state.Notifications, fmt.Sprintf("Eligibility: %v", state.IsEligible))
		return state, nil
	})

	g.AddNode("finalize", "Generate final result", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
		if state.IsEligible {
			state.Result = fmt.Sprintf("✅ %s is eligible!", state.Request.Name)
		} else {
			state.Result = fmt.Sprintf("❌ %s is not eligible", state.Request.Name)
		}
		fmt.Printf("Final result: %s\n", state.Result)
		return state, nil
	})

	// Define workflow edges
	g.SetEntryPoint("check_age")
	g.AddEdge("check_age", "check_eligibility")
	g.AddEdge("check_eligibility", "finalize")
	g.AddEdge("finalize", graph.END)

	// Compile the graph
	app, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Execute with type-safe input
	initialState := WorkflowState{
		Request: UserRequest{
			Name:    "Alice",
			Age:     25,
			Country: "USA",
		},
		Notifications: []string{},
	}

	// Invoke returns typed result - no type assertion needed!
	finalState, err := app.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	// Type-safe access to result
	fmt.Printf("\nFinal State:\n")
	fmt.Printf("  Result: %s\n", finalState.Result)
	fmt.Printf("  Notifications: %d messages\n", len(finalState.Notifications))
}

// Example 2: Conditional routing with type safety
func example2_ConditionalRouting() {
	fmt.Println("Example 2: Conditional Routing")
	fmt.Println("-------------------------------")

	g := graph.NewStateGraphTyped[WorkflowState]()

	g.AddNode("check_age", "Check age", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
		fmt.Printf("Checking age for %s (%d years old)\n", state.Request.Name, state.Request.Age)
		state.IsAdult = state.Request.Age >= 18
		return state, nil
	})

	g.AddNode("adult_path", "Process adult", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
		fmt.Println("  → Taking adult path")
		state.Result = fmt.Sprintf("%s (adult) - Full access granted", state.Request.Name)
		return state, nil
	})

	g.AddNode("minor_path", "Process minor", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
		fmt.Println("  → Taking minor path")
		state.Result = fmt.Sprintf("%s (minor) - Limited access", state.Request.Name)
		return state, nil
	})

	// Type-safe conditional edge - no type assertions!
	g.SetEntryPoint("check_age")
	g.AddConditionalEdge("check_age", func(ctx context.Context, state WorkflowState) string {
		// Full type safety here
		if state.IsAdult {
			return "adult_path"
		}
		return "minor_path"
	})
	g.AddEdge("adult_path", graph.END)
	g.AddEdge("minor_path", graph.END)

	app, _ := g.Compile()

	// Test with adult
	result1, _ := app.Invoke(context.Background(), WorkflowState{
		Request: UserRequest{Name: "Bob", Age: 30},
	})
	fmt.Printf("Result: %s\n\n", result1.Result)

	// Test with minor
	result2, _ := app.Invoke(context.Background(), WorkflowState{
		Request: UserRequest{Name: "Charlie", Age: 15},
	})
	fmt.Printf("Result: %s\n", result2.Result)
}

// Example 3: Using Schema for complex state merging
func example3_WithSchema() {
	fmt.Println("Example 3: Using Schema for State Merging")
	fmt.Println("-----------------------------------------")

	// Define a state type
	type ProcessState struct {
		Items      []string
		Count      int
		MaxCount   int
		Processing bool
	}

	g := graph.NewStateGraphTyped[ProcessState]()

	// Create a schema with custom merge logic
	schema := graph.NewStructSchema(
		ProcessState{MaxCount: 5, Processing: true}, // Initial values
		func(current, new ProcessState) (ProcessState, error) {
			// Custom merge: append items, sum count, preserve MaxCount and Processing from current
			current.Items = append(current.Items, new.Items...)
			current.Count += new.Count
			// Keep current.MaxCount and current.Processing
			return current, nil
		},
	)
	g.SetSchema(schema)

	// Add processing node
	g.AddNode("process", "Process items", func(ctx context.Context, state ProcessState) (ProcessState, error) {
		item := fmt.Sprintf("item_%d", state.Count+1)
		fmt.Printf("Processing: %s (count: %d/%d)\n", item, state.Count+1, state.MaxCount)

		// Return partial update - schema will merge it!
		return ProcessState{
			Items: []string{item},
			Count: 1, // This will be summed with current count
		}, nil
	})

	// Conditional loop
	g.SetEntryPoint("process")
	g.AddConditionalEdge("process", func(ctx context.Context, state ProcessState) string {
		if state.Count >= state.MaxCount {
			return graph.END
		}
		return "process"
	})

	app, _ := g.Compile()

	// Start with empty state - schema will initialize it
	result, _ := app.Invoke(context.Background(), ProcessState{})

	fmt.Printf("\nFinal State:\n")
	fmt.Printf("  Items processed: %v\n", result.Items)
	fmt.Printf("  Total count: %d\n", result.Count)
	fmt.Printf("  Max count: %d\n", result.MaxCount)
	fmt.Printf("  Processing: %v\n", result.Processing)
}

// Helper for string repetition (Go doesn't have built-in)
type stringHelper string

func (s stringHelper) repeat(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += string(s)
	}
	return result
}

// Add helper method to string type
func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
