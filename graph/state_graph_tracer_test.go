package graph

import (
	"context"
	"testing"
)

func TestStateGraph_WithTracer(t *testing.T) {
	// Create a StateGraph
	g := NewStateGraph()

	// Add nodes
	g.AddNode("node1", "First node", func(ctx context.Context, state any) (any, error) {
		return "result1", nil
	})

	g.AddNode("node2", "Second node", func(ctx context.Context, state any) (any, error) {
		return "result2", nil
	})

	g.AddEdge("node1", "node2")
	g.AddEdge("node2", END)
	g.SetEntryPoint("node1")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Create a tracer
	tracer := NewTracer()

	// Test SetTracer method
	runnable.SetTracer(tracer)
	if runnable.tracer != tracer {
		t.Fatal("SetTracer should set the tracer")
	}

	// Test WithTracer method
	runnableWithTracer := runnable.WithTracer(tracer)
	if runnableWithTracer.tracer != tracer {
		t.Fatal("WithTracer should return a new runnable with tracer")
	}

	// Execute the graph with tracer
	_, err = runnableWithTracer.Invoke(context.Background(), "initial")
	if err != nil {
		t.Fatalf("Failed to invoke: %v", err)
	}

	// Verify spans were collected
	spans := tracer.GetSpans()
	if len(spans) == 0 {
		t.Fatal("Tracer should have collected spans")
	}

	// Verify we have graph end and node end spans (events are updated when EndSpan is called)
	var hasGraphEnd, hasNode1End, hasNode2End bool
	for _, span := range spans {
		if span.Event == TraceEventGraphEnd && span.NodeName == "graph" {
			hasGraphEnd = true
		}
		if span.Event == TraceEventNodeEnd && span.NodeName == "node1" {
			hasNode1End = true
		}
		if span.Event == TraceEventNodeEnd && span.NodeName == "node2" {
			hasNode2End = true
		}
	}

	if !hasGraphEnd {
		t.Error("Should have GraphEnd event for graph")
	}
	if !hasNode1End {
		t.Error("Should have NodeEnd event for node1")
	}
	if !hasNode2End {
		t.Error("Should have NodeEnd event for node2")
	}

	t.Logf("StateGraph tracer test passed! Collected %d spans", len(spans))
}
