package graph

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParallelExecution_FanOut(t *testing.T) {
	// Define a simple state as a map to track execution
	type State struct {
		Visited []string
		mu      sync.Mutex
	}

	// Helper to append visited nodes safely
	visit := func(s *State, node string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.Visited = append(s.Visited, node)
	}

	g := NewStateGraph()

	// Node A: Entry point
	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		s := state.(*State)
		visit(s, "A")
		return s, nil
	})

	// Node B: Branch 1
	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		s := state.(*State)
		visit(s, "B")
		time.Sleep(10 * time.Millisecond) // Simulate work
		return s, nil
	})

	// Node C: Branch 2
	g.AddNode("C", "C", func(ctx context.Context, state any) (any, error) {
		s := state.(*State)
		visit(s, "C")
		time.Sleep(10 * time.Millisecond) // Simulate work
		return s, nil
	})

	// Node D: Join point
	g.AddNode("D", "D", func(ctx context.Context, state any) (any, error) {
		s := state.(*State)
		visit(s, "D")
		return s, nil
	})

	g.SetEntryPoint("A")

	// Define Fan-Out: A -> B, A -> C
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")

	// Define Fan-In: B -> D, C -> D
	g.AddEdge("B", "D")
	g.AddEdge("C", "D")

	g.AddEdge("D", END)

	// Compile
	runnable, err := g.Compile()
	assert.NoError(t, err)

	// Execute
	initialState := &State{Visited: []string{}}

	// We need a custom merger for this to work in the new implementation,
	// but for now we just want to see if it runs both B and C.
	// Since the state is a pointer, updates are shared (not thread safe if we didn't use mutex).
	// But the graph execution logic currently only picks ONE path.

	_, err = runnable.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Check results
	// Expected: A, then (B and C in any order), then D (maybe twice or once depending on implementation)
	// Current implementation will likely do: A -> B -> D -> END (skipping C)
	// or A -> C -> D -> END (skipping B)

	visited := initialState.Visited
	assert.Contains(t, visited, "A")
	assert.Contains(t, visited, "D")

	// Both B and C should be visited
	hasB := false
	hasC := false
	for _, v := range visited {
		if v == "B" {
			hasB = true
		}
		if v == "C" {
			hasC = true
		}
	}

	assert.True(t, hasB, "Node B should be visited")
	assert.True(t, hasC, "Node C should be visited")
}

func TestStateGraph_ParallelExecution(t *testing.T) {
	g := NewStateGraph()

	// State is map[string]int
	type State map[string]int

	// Merger function
	merger := func(ctx context.Context, current any, newStates []any) (any, error) {
		merged := make(State)
		// Copy current
		for k, v := range current.(State) {
			merged[k] = v
		}
		// Merge new states
		for _, s := range newStates {
			ns := s.(State)
			for k, v := range ns {
				merged[k] = v
			}
		}
		return merged, nil
	}
	g.SetStateMerger(merger)

	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		s := make(State)
		for k, v := range state.(State) {
			s[k] = v
		}
		s["A"] = 1
		return s, nil
	})

	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		s := make(State)
		for k, v := range state.(State) {
			s[k] = v
		}
		s["B"] = 1
		time.Sleep(10 * time.Millisecond)
		return s, nil
	})

	g.AddNode("C", "C", func(ctx context.Context, state any) (any, error) {
		s := make(State)
		for k, v := range state.(State) {
			s[k] = v
		}
		s["C"] = 1
		time.Sleep(10 * time.Millisecond)
		return s, nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("B", END)
	g.AddEdge("C", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	initialState := make(State)
	result, err := runnable.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	finalState := result.(State)
	assert.Equal(t, 1, finalState["A"])
	assert.Equal(t, 1, finalState["B"])
	assert.Equal(t, 1, finalState["C"])
}
