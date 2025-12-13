package prebuilt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// MockLLMWithInputCapture captures the input messages
type MockLLMWithInputCapture struct {
	CapturedMessages [][]llms.MessageContent
	responses        []llms.ContentResponse
	callCount        int
}

func (m *MockLLMWithInputCapture) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	m.CapturedMessages = append(m.CapturedMessages, messages)
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

func (m *MockLLMWithInputCapture) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

func TestCreateAgent(t *testing.T) {
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

	// Create Agent with System Message
	systemMsg := "You are a helpful assistant."
	agent, err := CreateAgent(mockLLM, []tools.Tool{mockTool}, WithSystemMessage(systemMsg))
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
}

func TestCreateAgent_SystemMessage(t *testing.T) {
	mockTool := &MockTool{name: "test-tool"}
	mockLLM := &MockLLMWithInputCapture{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{Content: "Response"},
				},
			},
		},
	}

	systemMsg := "System Prompt"
	agent, err := CreateAgent(mockLLM, []tools.Tool{mockTool}, WithSystemMessage(systemMsg))
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Hello"),
		},
	}

	_, err = agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify that the first message sent to LLM was the system message
	assert.NotEmpty(t, mockLLM.CapturedMessages)
	firstCallMessages := mockLLM.CapturedMessages[0]
	assert.Equal(t, llms.ChatMessageTypeSystem, firstCallMessages[0].Role)
	assert.Equal(t, "System Prompt", firstCallMessages[0].Parts[0].(llms.TextContent).Text)
	assert.Equal(t, llms.ChatMessageTypeHuman, firstCallMessages[1].Role)
}

func TestCreateAgent_StateModifier(t *testing.T) {
	mockTool := &MockTool{name: "test-tool"}
	mockLLM := &MockLLMWithInputCapture{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{Content: "Response"},
				},
			},
		},
	}

	// State modifier that adds a prefix to the last message
	modifier := func(messages []llms.MessageContent) []llms.MessageContent {
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if len(lastMsg.Parts) > 0 {
				if textPart, ok := lastMsg.Parts[0].(llms.TextContent); ok {
					textPart.Text = "Modified: " + textPart.Text
					lastMsg.Parts[0] = textPart
					messages[len(messages)-1] = lastMsg
				}
			}
		}
		return messages
	}

	agent, err := CreateAgent(mockLLM, []tools.Tool{mockTool}, WithStateModifier(modifier))
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Hello"),
		},
	}

	_, err = agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify that the message sent to LLM was modified
	assert.NotEmpty(t, mockLLM.CapturedMessages)
	firstCallMessages := mockLLM.CapturedMessages[0]
	assert.Equal(t, llms.ChatMessageTypeHuman, firstCallMessages[0].Role)
	assert.Equal(t, "Modified: Hello", firstCallMessages[0].Parts[0].(llms.TextContent).Text)
}
