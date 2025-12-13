package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphResume(t *testing.T) {
	g := NewStateGraph()
	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		return state.(string) + "A", nil
	})
	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		return state.(string) + "B", nil
	})
	g.AddNode("C", "C", func(ctx context.Context, state any) (any, error) {
		return state.(string) + "C", nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("C", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	// Test Resume after InterruptAfter
	t.Run("ResumeAfter", func(t *testing.T) {
		// 1. Run with interrupt after B
		config := &Config{
			InterruptAfter: []string{"B"},
		}
		_, err = runnable.InvokeWithConfig(context.Background(), "Start", config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)
		assert.Equal(t, "StartAB", interrupt.State)
		assert.Equal(t, []string{"C"}, interrupt.NextNodes)

		// 2. Resume from NextNodes with updated state
		// Simulate user modifying state
		updatedState := interrupt.State.(string) + "-Modified"

		resumeConfig := &Config{
			ResumeFrom: interrupt.NextNodes,
		}

		res2, err := runnable.InvokeWithConfig(context.Background(), updatedState, resumeConfig)
		assert.NoError(t, err)
		assert.Equal(t, "StartAB-ModifiedC", res2)
	})

	// Test Resume from InterruptBefore
	t.Run("ResumeBefore", func(t *testing.T) {
		// 1. Run with interrupt before B
		config := &Config{
			InterruptBefore: []string{"B"},
		}
		_, err = runnable.InvokeWithConfig(context.Background(), "Start", config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)
		assert.Equal(t, "StartA", interrupt.State)

		// 2. Resume from interrupted node
		resumeConfig := &Config{
			ResumeFrom: []string{interrupt.Node},
		}

		res2, err := runnable.InvokeWithConfig(context.Background(), interrupt.State, resumeConfig)
		assert.NoError(t, err)
		assert.Equal(t, "StartABC", res2)
	})
}
