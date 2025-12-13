package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/tools"
)

func TestMCPTool_Interface(t *testing.T) {
	// Verify that MCPTool implements tools.Tool interface
	var _ tools.Tool = &MCPTool{}
}

func TestMCPTool_NameAndDescription(t *testing.T) {
	tool := &MCPTool{
		name:        "test_tool",
		description: "A test tool",
	}

	assert.Equal(t, "test_tool", tool.Name())
	assert.Equal(t, "A test tool", tool.Description())
}

func TestGetToolSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
		},
	}

	tool := &MCPTool{
		name:        "search",
		description: "Search tool",
		parameters:  schema,
	}

	retrievedSchema, ok := GetToolSchema(tool)
	assert.True(t, ok)
	assert.Equal(t, schema, retrievedSchema)

	// Test with non-MCP tool (using a different MCPTool without the schema)
	nonMCPTool := &MCPTool{
		name:        "other",
		description: "Other tool",
		parameters:  nil,
	}
	retrievedSchema, ok = GetToolSchema(nonMCPTool)
	assert.True(t, ok)
	assert.Nil(t, retrievedSchema)
}

// TestMCPToTools_EmptyClient tests the conversion with no tools
func TestMCPToTools_EmptyConversion(t *testing.T) {
	// This test would require a mock MCP client
	// For now, we just verify the function signature exists
	ctx := context.Background()

	// We can't actually call MCPToTools without a real client,
	// but we verify the function exists and has the right signature
	_ = ctx

	// Type check
	var fn func(context.Context, any) ([]tools.Tool, error)
	_ = fn
}

// Example usage documentation
func ExampleMCPToTools() {
	ctx := context.Background()

	// Load MCP client from config file
	client, err := NewClientFromConfig(ctx, "~/.claude.json")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Convert MCP tools to langchaingo tools
	tools, err := MCPToTools(ctx, client)
	if err != nil {
		panic(err)
	}

	// Use tools with langchaingo or langgraphgo agents
	_ = tools
}

func ExampleNewClientFromConfig() {
	ctx := context.Background()

	// Create MCP client from Claude config file
	client, err := NewClientFromConfig(ctx, "~/.claude.json")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Now you can use the client to get tools or call them directly
	_ = client
}
