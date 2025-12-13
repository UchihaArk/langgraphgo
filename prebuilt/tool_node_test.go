package prebuilt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

func TestToolNode(t *testing.T) {
	mockTool := &MockTool{name: "test-tool"}
	toolNode := NewToolNode([]tools.Tool{mockTool})

	// Construct state with AIMessage containing ToolCall
	toolCall := llms.ToolCall{
		ID:   "call_1",
		Type: "function",
		FunctionCall: &llms.FunctionCall{
			Name:      "test-tool",
			Arguments: `{"input": "test-input"}`,
		},
	}

	aiMsg := llms.MessageContent{
		Role: llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{
			toolCall,
		},
	}

	state := map[string]any{
		"messages": []llms.MessageContent{aiMsg},
	}

	// Invoke ToolNode
	res, err := toolNode.Invoke(context.Background(), state)
	assert.NoError(t, err)

	// Verify result
	resMap, ok := res.(map[string]any)
	assert.True(t, ok)

	messages, ok := resMap["messages"].([]llms.MessageContent)
	assert.True(t, ok)
	assert.Len(t, messages, 1)

	toolMsg := messages[0]
	assert.Equal(t, llms.ChatMessageTypeTool, toolMsg.Role)
	assert.Len(t, toolMsg.Parts, 1)

	toolResp, ok := toolMsg.Parts[0].(llms.ToolCallResponse)
	assert.True(t, ok)
	assert.Equal(t, "call_1", toolResp.ToolCallID)
	assert.Equal(t, "test-tool", toolResp.Name)
	assert.Equal(t, "Executed test-tool with test-input", toolResp.Content)
}
