package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamingModes(t *testing.T) {
	g := NewStreamingStateGraph()

	// Setup simple graph
	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		return "A", nil
	})
	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		return "B", nil
	})
	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", END)

	// Test StreamModeValues
	t.Run("Values", func(t *testing.T) {
		g.SetStreamConfig(StreamConfig{
			BufferSize: 100,
			Mode:       StreamModeValues,
		})

		runnable, err := g.CompileStreaming()
		assert.NoError(t, err)

		res := runnable.Stream(context.Background(), "Start")

		var events []StreamEvent
		for event := range res.Events {
			events = append(events, event)
		}

		// Expect "graph_step" events
		// A runs -> state "StartA"
		// B runs -> state "StartAB"
		// (Assuming StateGraph appends strings by default or replaces?
		// StateGraph defaults to simple replacement if no schema?
		// Wait, StateGraph uses ListenableStateGraph which uses StateGraph.
		// StateGraph uses default Node/Edge structs.
		// It does NOT have a default schema/reducer unless set.
		// If no schema/merger, parallel execution takes last result.
		// Sequential A->B: A returns "A". State becomes "A".
		// B returns "B". State becomes "B".

		// Let's verify graph behavior first.
		// A -> "A". B -> "B".
		// Events:
		// 1. graph_step (after A): State "A"
		// 2. graph_step (after B): State "B"

		assert.NotEmpty(t, events)
		for _, e := range events {
			assert.Equal(t, "graph_step", string(e.Event))
		}

		lastEvent := events[len(events)-1]
		assert.Equal(t, "B", lastEvent.State)
	})

	// Test StreamModeUpdates
	t.Run("Updates", func(t *testing.T) {
		g.SetStreamConfig(StreamConfig{
			BufferSize: 100,
			Mode:       StreamModeUpdates,
		})

		runnable, err := g.CompileStreaming()
		assert.NoError(t, err)

		res := runnable.Stream(context.Background(), "Start")

		var events []StreamEvent
		for event := range res.Events {
			events = append(events, event)
		}

		// Expect ToolEnd events (since nodes are treated as tools)
		// A -> "A"
		// B -> "B"

		foundA := false
		foundB := false
		for _, e := range events {
			if e.Event == NodeEventComplete {
				if e.State == "A" {
					foundA = true
				}
				if e.State == "B" {
					foundB = true
				}
			}
		}
		assert.True(t, foundA)
		assert.True(t, foundB)
	})
}
