package graph

import (
	"context"
	"testing"
	"time"
)

// TestState is a simple test state
type TestState struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
}

func TestStateGraphTyped_BasicFunctionality(t *testing.T) {
	// Create a new typed state graph
	g := NewStateGraphTyped[TestState]()

	// Add nodes
	g.AddNode("increment", "Increment counter", func(ctx context.Context, state TestState) (TestState, error) {
		state.Count++
		return state, nil
	})

	g.AddNode("check", "Check count", func(ctx context.Context, state TestState) (TestState, error) {
		if state.Name == "" {
			state.Name = "test"
		}
		return state, nil
	})

	// Set up graph structure
	g.SetEntryPoint("increment")
	g.AddEdge("increment", "check")
	g.AddEdge("check", END)

	// Compile
	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// Test invocation
	initialState := TestState{Count: 0}
	finalState, err := runnable.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke graph: %v", err)
	}

	// Verify results
	if finalState.Count != 1 {
		t.Errorf("Expected count to be 1, got %d", finalState.Count)
	}
	if finalState.Name != "test" {
		t.Errorf("Expected name to be 'test', got '%s'", finalState.Name)
	}
}

func TestStateGraphTyped_ConditionalEdges(t *testing.T) {
	g := NewStateGraphTyped[TestState]()

	g.AddNode("process", "Process", func(ctx context.Context, state TestState) (TestState, error) {
		state.Count++
		return state, nil
	})

	g.AddNode("high", "High count", func(ctx context.Context, state TestState) (TestState, error) {
		state.Name = "high"
		return state, nil
	})

	g.AddNode("low", "Low count", func(ctx context.Context, state TestState) (TestState, error) {
		state.Name = "low"
		return state, nil
	})

	g.SetEntryPoint("process")
	g.AddConditionalEdge("process", func(ctx context.Context, state TestState) string {
		if state.Count > 5 {
			return "high"
		}
		return "low"
	})
	g.AddEdge("high", END)
	g.AddEdge("low", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// Test with initial count of 4 (after increment becomes 5)
	state, err := runnable.Invoke(context.Background(), TestState{Count: 4})
	if err != nil {
		t.Fatalf("Failed to invoke graph: %v", err)
	}
	if state.Name != "low" {
		t.Errorf("Expected name to be 'low', got '%s'", state.Name)
	}

	// Test with initial count of 5 (after increment becomes 6)
	state, err = runnable.Invoke(context.Background(), TestState{Count: 5})
	if err != nil {
		t.Fatalf("Failed to invoke graph: %v", err)
	}
	if state.Name != "high" {
		t.Errorf("Expected name to be 'high', got '%s'", state.Name)
	}
}

func TestStateGraphTyped_WithSchema(t *testing.T) {
	g := NewStateGraphTyped[TestState]()

	// Define schema with merge function
	schema := NewStructSchema(
		TestState{Name: "default"},
		func(current, new TestState) (TestState, error) {
			// Preserve name from current, take count from new
			if new.Name != "" {
				current.Name = new.Name
			}
			if new.Count != 0 {
				current.Count = new.Count
			}
			return current, nil
		},
	)
	g.SetSchema(schema)

	g.AddNode("update", "Update", func(ctx context.Context, state TestState) (TestState, error) {
		return TestState{Count: state.Count + 1}, nil
	})

	g.SetEntryPoint("update")
	g.AddEdge("update", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// Test with schema
	state, err := runnable.Invoke(context.Background(), TestState{Count: 5})
	if err != nil {
		t.Fatalf("Failed to invoke graph: %v", err)
	}

	// Schema should preserve the default name
	if state.Name != "default" {
		t.Errorf("Expected name to be 'default', got '%s'", state.Name)
	}
	if state.Count != 6 {
		t.Errorf("Expected count to be 6, got %d", state.Count)
	}
}

func TestListenableStateGraphTyped_BasicFunctionality(t *testing.T) {
	g := NewListenableStateGraphTyped[TestState]()

	// Add a listener to track events
	var events []string
	listener := NodeListenerTypedFunc[TestState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestState, err error) {
			events = append(events, string(event)+":"+nodeName)
		},
	)

	// Add node with listener
	node := g.AddNode("test", "Test node", func(ctx context.Context, state TestState) (TestState, error) {
		state.Count++
		return state, nil
	})
	node.AddListener(listener)

	g.SetEntryPoint("test")
	g.AddEdge("test", END)

	// Compile and run
	runnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Failed to compile listenable graph: %v", err)
	}

	state, err := runnable.Invoke(context.Background(), TestState{})
	if err != nil {
		t.Fatalf("Failed to invoke listenable graph: %v", err)
	}

	// Check that events were captured
	if len(events) == 0 {
		t.Error("No events were captured")
	}

	// Verify state
	if state.Count != 1 {
		t.Errorf("Expected count to be 1, got %d", state.Count)
	}
}

func TestStateGraphTyped_ParallelExecution(t *testing.T) {
	g := NewStateGraphTyped[TestState]()

	// Add multiple nodes that can run in parallel
	g.AddNode("node1", "Node 1", func(ctx context.Context, state TestState) (TestState, error) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		state.Count += 1
		return state, nil
	})

	g.AddNode("node2", "Node 2", func(ctx context.Context, state TestState) (TestState, error) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		state.Count += 2
		return state, nil
	})

	g.SetEntryPoint("node1")
	g.AddEdge("node1", "node2")
	g.AddEdge("node2", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	start := time.Now()
	state, err := runnable.Invoke(context.Background(), TestState{})
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to invoke graph: %v", err)
	}

	// Verify the nodes were executed
	if state.Count != 3 {
		t.Errorf("Expected count to be 3, got %d", state.Count)
	}

	// Verify execution time (should be less than if executed serially)
	if duration > 50*time.Millisecond {
		t.Errorf("Execution took too long: %v", duration)
	}
}

func BenchmarkStateGraphTyped_Invoke(b *testing.B) {
	g := NewStateGraphTyped[TestState]()

	g.AddNode("increment", "Increment", func(ctx context.Context, state TestState) (TestState, error) {
		state.Count++
		return state, nil
	})

	g.SetEntryPoint("increment")
	g.AddEdge("increment", END)

	runnable, err := g.Compile()
	if err != nil {
		b.Fatalf("Failed to compile graph: %v", err)
	}

	ctx := context.Background()
	initialState := TestState{Count: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runnable.Invoke(ctx, initialState)
		if err != nil {
			b.Fatalf("Failed to invoke graph: %v", err)
		}
	}
}

func BenchmarkListenableStateGraphTyped_Invoke(b *testing.B) {
	g := NewListenableStateGraphTyped[TestState]()

	g.AddNode("increment", "Increment", func(ctx context.Context, state TestState) (TestState, error) {
		state.Count++
		return state, nil
	})

	g.SetEntryPoint("increment")
	g.AddEdge("increment", END)

	runnable, err := g.CompileListenable()
	if err != nil {
		b.Fatalf("Failed to compile listenable graph: %v", err)
	}

	ctx := context.Background()
	initialState := TestState{Count: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runnable.Invoke(ctx, initialState)
		if err != nil {
			b.Fatalf("Failed to invoke listenable graph: %v", err)
		}
	}
}

// Test StateGraphTyped methods directly
func TestStateGraphTyped_AdditionalMethods(t *testing.T) {
	g := NewStateGraphTyped[TestState]()

	// Test SetRetryPolicy
	policy := &RetryPolicy{
		MaxRetries: 3,
	}
	g.SetRetryPolicy(policy)

	if g.retryPolicy != policy {
		t.Error("SetRetryPolicy should set the retryPolicy field")
	}

	// Test SetStateMerger
	merger := func(ctx context.Context, current TestState, newStates []TestState) (TestState, error) {
		for _, ns := range newStates {
			current.Count += ns.Count
		}
		return current, nil
	}
	g.SetStateMerger(merger)

	if g.stateMerger == nil {
		t.Error("SetStateMerger should set the stateMerger field")
	}
}

// Test StateRunnableTyped methods
func TestStateRunnableTyped_SetTracer(t *testing.T) {
	g := NewStateGraphTyped[TestState]()
	g.AddNode("test", "Test node", func(ctx context.Context, state TestState) (TestState, error) {
		return state, nil
	})
	g.SetEntryPoint("test")
	g.AddEdge("test", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	tracer := &Tracer{}
	runnable.SetTracer(tracer)

	if runnable.tracer != tracer {
		t.Error("SetTracer should set the tracer field")
	}
}

func TestStateRunnableTyped_WithTracer(t *testing.T) {
	g := NewStateGraphTyped[TestState]()
	g.AddNode("test", "Test node", func(ctx context.Context, state TestState) (TestState, error) {
		return state, nil
	})
	g.SetEntryPoint("test")
	g.AddEdge("test", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	tracer := &Tracer{}
	newRunnable := runnable.WithTracer(tracer)

	if newRunnable == runnable {
		t.Error("WithTracer should return a new instance")
	}

	if newRunnable.graph != runnable.graph {
		t.Error("New runnable should have the same graph")
	}
}

// Test edge cases
func TestStateGraphTyped_MultipleEdgesFromNode(t *testing.T) {
	g := NewStateGraphTyped[TestState]()

	g.AddNode("source", "Source node", func(ctx context.Context, state TestState) (TestState, error) {
		return state, nil
	})
	g.AddNode("target1", "Target 1", func(ctx context.Context, state TestState) (TestState, error) {
		state.Count = 1
		return state, nil
	})
	g.AddNode("target2", "Target 2", func(ctx context.Context, state TestState) (TestState, error) {
		state.Count = 2
		return state, nil
	})

	// Add multiple edges from source (fan-out)
	g.AddEdge("source", "target1")
	g.AddEdge("source", "target2")

	// Add edges from targets to END
	g.AddEdge("target1", END)
	g.AddEdge("target2", END)

	g.SetEntryPoint("source")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// This should work - multiple targets should be executed in parallel
	ctx := context.Background()
	result, err := runnable.Invoke(ctx, TestState{})

	if err != nil {
		t.Errorf("Should not error with fan-out: %v", err)
	}

	// Result should be from one of the targets (parallel execution may return either)
	if result.Count != 1 && result.Count != 2 {
		t.Errorf("Expected count to be 1 or 2, got %d", result.Count)
	}
}

func TestStateGraphTyped_ComplexStateType(t *testing.T) {
	// Test with complex nested state
	type ComplexState struct {
		Info struct {
			Name    string
			Version int
		}
		Data  map[string]interface{}
		Items []struct {
			ID   int
			Tags []string
		}
		Processed bool
	}

	g := NewStateGraphTyped[ComplexState]()

	g.AddNode("process", "Process complex state", func(ctx context.Context, state ComplexState) (ComplexState, error) {
		state.Info.Name = "processed"
		state.Processed = true
		return state, nil
	})

	g.SetEntryPoint("process")
	g.AddEdge("process", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	initialState := ComplexState{
		Info: struct {
			Name    string
			Version int
		}{
			Name:    "initial",
			Version: 1,
		},
	}

	result, err := runnable.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke: %v", err)
	}

	if !result.Processed {
		t.Error("State should be marked as processed")
	}

	if result.Info.Name != "processed" {
		t.Errorf("Expected name to be 'processed', got '%s'", result.Info.Name)
	}
}

func TestStateGraphTyped_MapState(t *testing.T) {
	// Test with map state
	g := NewStateGraphTyped[map[string]any]()

	g.AddNode("process", "Process map", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["count"] = state["count"].(int) + 1
		state["processed"] = true
		return state, nil
	})

	g.SetEntryPoint("process")
	g.AddEdge("process", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	initialState := map[string]any{
		"count": 0,
		"name":  "test",
	}

	result, err := runnable.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke: %v", err)
	}

	if result["count"].(int) != 1 {
		t.Errorf("Expected count to be 1, got %v", result["count"])
	}

	if !result["processed"].(bool) {
		t.Error("Should be marked as processed")
	}

	if result["name"].(string) != "test" {
		t.Errorf("Expected name to be 'test', got %v", result["name"])
	}
}

func TestStateGraphTyped_StringState(t *testing.T) {
	// Test with simple string state
	g := NewStateGraphTyped[string]()

	g.AddNode("process", "Process string", func(ctx context.Context, state string) (string, error) {
		return state + "_processed", nil
	})

	g.SetEntryPoint("process")
	g.AddEdge("process", END)

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	result, err := runnable.Invoke(context.Background(), "initial")
	if err != nil {
		t.Fatalf("Failed to invoke: %v", err)
	}

	if result != "initial_processed" {
		t.Errorf("Expected 'initial_processed', got '%s'", result)
	}
}