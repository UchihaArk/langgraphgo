package graph

import (
	"context"
	"errors"
	"testing"
)

func TestStateGraph_Interrupt(t *testing.T) {
	// Create a StateGraph
	g := NewStateGraph()

	// Add node that uses Interrupt
	g.AddNode("node1", "Node with interrupt", func(ctx context.Context, state any) (any, error) {
		// Use the Interrupt function
		resumeValue, err := Interrupt(ctx, "waiting for input")
		if err != nil {
			return nil, err
		}
		// If we resumed, return the resume value
		if resumeValue != nil {
			return resumeValue, nil
		}
		return "default", nil
	})

	g.AddEdge("node1", END)
	g.SetEntryPoint("node1")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// First execution should interrupt
	_, err = runnable.Invoke(context.Background(), "initial")

	// Verify we got an interrupt error
	var graphInterrupt *GraphInterrupt
	if err == nil {
		t.Fatal("Expected interrupt error, got nil")
	}

	// Check if it's a NodeInterrupt wrapped in error or GraphInterrupt
	var nodeInterrupt *NodeInterrupt
	if !errors.As(err, &nodeInterrupt) {
		// Try GraphInterrupt
		if !errors.As(err, &graphInterrupt) {
			t.Fatalf("Expected NodeInterrupt or GraphInterrupt error, got: %v", err)
		}
	}

	if graphInterrupt != nil {
		if graphInterrupt.InterruptValue != "waiting for input" {
			t.Errorf("Expected interrupt value 'waiting for input', got: %v", graphInterrupt.InterruptValue)
		}
		t.Logf("Successfully interrupted with GraphInterrupt, value: %v", graphInterrupt.InterruptValue)
	} else {
		if nodeInterrupt.Value != "waiting for input" {
			t.Errorf("Expected interrupt value 'waiting for input', got: %v", nodeInterrupt.Value)
		}
		t.Logf("Successfully interrupted with NodeInterrupt, value: %v", nodeInterrupt.Value)
	}

	t.Log("StateGraph Interrupt test passed!")
}
