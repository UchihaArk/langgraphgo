package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapSchema_Update(t *testing.T) {
	schema := NewMapSchema()
	schema.RegisterReducer("messages", AppendReducer)

	initialState := map[string]any{
		"messages": []string{"hello"},
		"count":    1,
	}

	// Update 1: Append message
	update1 := map[string]any{
		"messages": []string{"world"},
	}

	newState1, err := schema.Update(initialState, update1)
	assert.NoError(t, err)

	state1 := newState1.(map[string]any)
	assert.Equal(t, []string{"hello", "world"}, state1["messages"])
	assert.Equal(t, 1, state1["count"])

	// Update 2: Overwrite count
	update2 := map[string]any{
		"count": 2,
	}

	newState2, err := schema.Update(state1, update2)
	assert.NoError(t, err)

	state2 := newState2.(map[string]any)
	assert.Equal(t, []string{"hello", "world"}, state2["messages"])
	assert.Equal(t, 2, state2["count"])

	// Update 3: Append single element (if supported by AppendReducer logic, currently it supports slice or element)
	// Let's test appending a single string
	update3 := map[string]any{
		"messages": "!",
	}

	newState3, err := schema.Update(state2, update3)
	assert.NoError(t, err)

	state3 := newState3.(map[string]any)
	assert.Equal(t, []string{"hello", "world", "!"}, state3["messages"])
}

func TestStateGraph_Schema(t *testing.T) {
	g := NewStateGraph()

	schema := NewMapSchema()
	schema.RegisterReducer("messages", AppendReducer)
	g.SetSchema(schema)

	g.AddNode("A", "A", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"messages": []string{"A"},
		}, nil
	})

	g.AddNode("B", "B", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"messages": []string{"B"},
		}, nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []string{"start"},
	}

	result, err := runnable.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	finalState := result.(map[string]any)
	assert.Equal(t, []string{"start", "A", "B"}, finalState["messages"])
}
