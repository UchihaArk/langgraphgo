package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphInterrupt(t *testing.T) {
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

	// Test InterruptBefore
	t.Run("InterruptBefore", func(t *testing.T) {
		config := &Config{
			InterruptBefore: []string{"B"},
		}
		res, err := runnable.InvokeWithConfig(context.Background(), "Start", config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)
		assert.Equal(t, "StartA", interrupt.State)

		// Result should be the state at interruption
		assert.Equal(t, "StartA", res)
	})

	// Test InterruptAfter
	t.Run("InterruptAfter", func(t *testing.T) {
		config := &Config{
			InterruptAfter: []string{"B"},
		}
		res, err := runnable.InvokeWithConfig(context.Background(), "Start", config)

		assert.Error(t, err)
		var interrupt *GraphInterrupt
		assert.ErrorAs(t, err, &interrupt)
		assert.Equal(t, "B", interrupt.Node)
		assert.Equal(t, "StartAB", interrupt.State)

		// Result should be the state at interruption
		assert.Equal(t, "StartAB", res)
	})
}
