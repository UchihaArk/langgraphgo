package prebuilt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// MockLLM implements llms.Model for testing
type MockLLM struct {
	responses []llms.ContentResponse
	callCount int
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.callCount >= len(m.responses) {
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{Content: "No more responses"},
			},
		}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &resp, nil
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

func TestCreateReactAgent(t *testing.T) {
	// Setup Mock Tool
	mockTool := &MockTool{name: "test-tool"}

	// Setup Mock LLM
	// 1. First call: returns tool call
	// 2. Second call: returns final answer
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "call-1",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "test-tool",
									Arguments: "input-1",
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "Final Answer",
					},
				},
			},
		},
	}

	// Create Agent
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{mockTool})
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Run tool"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)

	// Expected messages:
	// 0: Human "Run tool"
	// 1: AI (ToolCall)
	// 2: Tool (ToolCallResponse)
	// 3: AI "Final Answer"
	assert.Equal(t, 4, len(messages))
	assert.Equal(t, llms.ChatMessageTypeHuman, messages[0].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[1].Role)
	assert.Equal(t, llms.ChatMessageTypeTool, messages[2].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[3].Role)

	// Verify Tool Response
	toolMsg := messages[2]
	assert.Equal(t, 1, len(toolMsg.Parts))
	toolResp, ok := toolMsg.Parts[0].(llms.ToolCallResponse)
	assert.True(t, ok)
	assert.Equal(t, "call-1", toolResp.ToolCallID)
	assert.Equal(t, "Executed test-tool with input-1", toolResp.Content)

	// Verify Final Answer
	finalMsg := messages[3]
	assert.Equal(t, 1, len(finalMsg.Parts))
	textPart, ok := finalMsg.Parts[0].(llms.TextContent)
	assert.True(t, ok)
	assert.Equal(t, "Final Answer", textPart.Text)
}
