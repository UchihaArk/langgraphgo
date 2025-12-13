package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimeConfiguration(t *testing.T) {
	g := NewStateGraph()

	// Define a node that reads config from context
	g.AddNode("reader", "reader", func(ctx context.Context, state any) (any, error) {
		config := GetConfig(ctx)
		if config == nil {
			return "no config", nil
		}

		if val, ok := config.Configurable["model"]; ok {
			return val, nil
		}
		return "key not found", nil
	})

	g.SetEntryPoint("reader")
	g.AddEdge("reader", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	// Test with config
	config := &Config{
		Configurable: map[string]any{
			"model": "gpt-4",
		},
	}

	result, err := runnable.InvokeWithConfig(context.Background(), nil, config)
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4", result)

	// Test without config
	result, err = runnable.Invoke(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, "no config", result)
}

func TestStateGraph_RuntimeConfiguration(t *testing.T) {
	g := NewStateGraph()

	g.AddNode("reader", "reader", func(ctx context.Context, state any) (any, error) {
		config := GetConfig(ctx)
		if config == nil {
			return "no config", nil
		}

		if val, ok := config.Configurable["api_key"]; ok {
			return val, nil
		}
		return "key not found", nil
	})

	g.SetEntryPoint("reader")
	g.AddEdge("reader", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	config := &Config{
		Configurable: map[string]any{
			"api_key": "secret-123",
		},
	}

	result, err := runnable.InvokeWithConfig(context.Background(), nil, config)
	assert.NoError(t, err)
	assert.Equal(t, "secret-123", result)
}
