package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEphemeralChannel(t *testing.T) {
	g := NewStateGraph()

	schema := NewMapSchema()
	// "temp" is ephemeral, "count" is persistent
	schema.RegisterChannel("temp", OverwriteReducer, true)
	schema.RegisterReducer("count", func(curr, new any) (any, error) {
		if curr == nil {
			return new, nil
		}
		return curr.(int) + new.(int), nil
	})
	g.SetSchema(schema)

	// Node A: Sets temp=1, count=1
	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"temp":  1,
			"count": 1,
		}, nil
	})

	// Node B: Reads temp, adds to count.
	// Since A -> B is one step? No, A and B are separate nodes.
	// If A -> B, B runs in the NEXT super-step.
	// So temp should be CLEARED before B runs?
	// Wait, if temp is ephemeral, it should be available within the SAME super-step (if parallel)
	// but cleared for the NEXT super-step.
	// If A -> B, B is next step. So B should NOT see temp?
	// Let's verify LangGraph behavior.
	// "Ephemeral values are cleared after the step."
	// So if A runs, then step ends. Temp is cleared. B runs. B sees no temp.

	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		mState := state.(map[string]any)
		// temp should be missing or nil
		if _, ok := mState["temp"]; ok {
			return map[string]any{"count": 100}, nil // Error flag
		}
		return map[string]any{"count": 10}, nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	res, err := runnable.Invoke(context.Background(), map[string]any{"count": 0})
	assert.NoError(t, err)

	mRes, ok := res.(map[string]any)
	assert.True(t, ok)

	// Expected:
	// A runs: count=1, temp=1. Step ends. Cleanup -> temp removed.
	// B runs: sees count=1, no temp. Returns count=10.
	// Final: count=11.
	// If B saw temp, it would return count=100 -> Final 101.

	assert.Equal(t, 11, mRes["count"])
	_, hasTemp := mRes["temp"]
	assert.False(t, hasTemp)
}
