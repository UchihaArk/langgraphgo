package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/stretchr/testify/assert"
)

func TestRedisCheckpointStore(t *testing.T) {
	// Start miniredis
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	// Create store
	store := NewRedisCheckpointStore(RedisOptions{
		Addr: mr.Addr(),
	})

	ctx := context.Background()
	execID := "exec-123"

	// Create checkpoint
	cp := &graph.Checkpoint{
		ID:        "cp-1",
		NodeName:  "node-a",
		State:     map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
		Version:   1,
		Metadata: map[string]any{
			"execution_id": execID,
		},
	}

	// Test Save
	err = store.Save(ctx, cp)
	assert.NoError(t, err)

	// Test Load
	loaded, err := store.Load(ctx, "cp-1")
	assert.NoError(t, err)
	assert.Equal(t, cp.ID, loaded.ID)
	assert.Equal(t, cp.NodeName, loaded.NodeName)
	// JSON unmarshal converts numbers to float64, so exact map comparison might fail on types if not careful
	// But here we used string, so it should be fine.
	state, ok := loaded.State.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "bar", state["foo"])

	// Test List
	list, err := store.List(ctx, execID)
	assert.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, cp.ID, list[0].ID)

	// Test Delete
	err = store.Delete(ctx, "cp-1")
	assert.NoError(t, err)

	_, err = store.Load(ctx, "cp-1")
	assert.Error(t, err)

	list, err = store.List(ctx, execID)
	assert.NoError(t, err)
	assert.Len(t, list, 0)

	// Test Clear
	// Add multiple checkpoints
	cp2 := &graph.Checkpoint{ID: "cp-2", Metadata: map[string]any{"execution_id": execID}}
	cp3 := &graph.Checkpoint{ID: "cp-3", Metadata: map[string]any{"execution_id": execID}}
	store.Save(ctx, cp2)
	store.Save(ctx, cp3)

	list, err = store.List(ctx, execID)
	assert.NoError(t, err)
	assert.Len(t, list, 2)

	err = store.Clear(ctx, execID)
	assert.NoError(t, err)

	list, err = store.List(ctx, execID)
	assert.NoError(t, err)
	assert.Len(t, list, 0)
}
