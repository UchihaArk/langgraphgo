package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateState(t *testing.T) {
	g := NewCheckpointableStateGraph()

	// Setup schema with reducer
	schema := NewMapSchema()
	schema.RegisterReducer("count", func(curr, new any) (any, error) {
		if curr == nil {
			return new, nil
		}
		return curr.(int) + new.(int), nil
	})
	g.SetSchema(schema)

	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		return map[string]any{"count": 1}, nil
	})
	g.SetEntryPoint("A")
	g.AddEdge("A", END)

	runnable, err := g.CompileCheckpointable()
	assert.NoError(t, err)

	// 1. Run initial graph
	ctx := context.Background()
	res, err := runnable.Invoke(ctx, map[string]any{"count": 10})
	assert.NoError(t, err)

	mRes := res.(map[string]any)
	assert.Equal(t, 11, mRes["count"]) // 10 + 1 = 11

	// 2. Update state manually (Human-in-the-loop)
	// We want to add 5 to the count
	config := &Config{
		Configurable: map[string]any{
			"thread_id": runnable.executionID,
		},
	}

	newConfig, err := runnable.UpdateState(ctx, config, map[string]any{"count": 5}, "human")
	assert.NoError(t, err)
	assert.NotEmpty(t, newConfig.Configurable["checkpoint_id"])

	// 3. Verify state is updated
	snapshot, err := runnable.GetState(ctx, newConfig)
	assert.NoError(t, err)

	mSnap := snapshot.Values.(map[string]any)
	// Should be 11 (previous) + 5 (update) = 16
	assert.Equal(t, 16, mSnap["count"])
}
