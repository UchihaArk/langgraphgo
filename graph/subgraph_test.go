package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubgraph(t *testing.T) {
	// 1. Define Child Graph
	child := NewStateGraph()
	child.AddNode("child_A", "child_A", func(ctx context.Context, state any) (any, error) {
		m := state.(map[string]any)
		m["child_visited"] = true
		return m, nil
	})
	child.SetEntryPoint("child_A")
	child.AddEdge("child_A", END)

	// 2. Define Parent Graph
	parent := NewStateGraph()
	parent.AddNode("parent_A", "parent_A", func(ctx context.Context, state any) (any, error) {
		m := state.(map[string]any)
		m["parent_visited"] = true
		return m, nil
	})

	// Add Child Graph as a node
	err := parent.AddSubgraph("child", child)
	assert.NoError(t, err)

	parent.SetEntryPoint("parent_A")
	parent.AddEdge("parent_A", "child")
	parent.AddEdge("child", END)

	// 3. Run Parent Graph
	runnable, err := parent.Compile()
	assert.NoError(t, err)

	res, err := runnable.Invoke(context.Background(), map[string]any{})
	assert.NoError(t, err)

	mRes := res.(map[string]any)
	assert.True(t, mRes["parent_visited"].(bool))
	assert.True(t, mRes["child_visited"].(bool))
}
