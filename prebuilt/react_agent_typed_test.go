package prebuilt

import (
	"context"
	"fmt"
	"testing"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// MockTool for testing
type MockToolForReact struct {
	name        string
	description string
}

func (t *MockToolForReact) Name() string        { return t.name }
func (t *MockToolForReact) Description() string { return t.description }
func (t *MockToolForReact) Call(ctx context.Context, input string) (string, error) {
	return "Result: " + input, nil
}

// MockLLMForReact for testing
type MockLLMForReact struct {
	responses     []llms.ContentChoice
	currentIndex  int
	withToolCalls bool
}

func (m *MockLLMForReact) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.currentIndex >= len(m.responses) {
		m.currentIndex = 0
	}

	choice := m.responses[m.currentIndex]
	m.currentIndex++

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{&choice},
	}, nil
}

// Call implements the deprecated Call method for backward compatibility
func (m *MockLLMForReact) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	// Simple implementation that returns a default response
	if m.currentIndex > 0 && m.currentIndex <= len(m.responses) {
		return m.responses[m.currentIndex-1].Content, nil
	}
	return "Mock response", nil
}

// NewMockLLMWithTextResponse creates a mock LLM that returns text responses
func NewMockLLMWithTextResponse(responses []string) *MockLLMForReact {
	choices := make([]llms.ContentChoice, len(responses))
	for i, resp := range responses {
		choices[i] = llms.ContentChoice{
			Content: resp,
		}
	}

	return &MockLLMForReact{
		responses:     choices,
		currentIndex:  0,
		withToolCalls: false,
	}
}

// NewMockLLMWithToolCalls creates a mock LLM that returns tool calls
func NewMockLLMWithToolCalls(toolCalls []llms.ToolCall) *MockLLMForReact {
	choice := llms.ContentChoice{
		Content: "Using tool",
		ToolCalls: toolCalls,
	}

	return &MockLLMForReact{
		responses:     []llms.ContentChoice{choice},
		currentIndex:  0,
		withToolCalls: true,
	}
}

func TestCreateReactAgentTyped(t *testing.T) {
	// Create mock tools
	tools := []tools.Tool{
		&MockToolForReact{
			name:        "test_tool",
			description: "A test tool",
		},
		&MockToolForReact{
			name:        "another_tool",
			description: "Another test tool",
		},
	}

	// Create mock LLM with text response (no tool calls)
	mockLLM := NewMockLLMWithTextResponse([]string{
		"The answer is 42",
	})

	// Create ReAct agent
	agent, err := CreateReactAgentTyped(mockLLM, tools)
	if err != nil {
		t.Fatalf("Failed to create ReAct agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

func TestCreateReactAgentTyped_WithTools(t *testing.T) {
	tools := []tools.Tool{
		&MockToolForReact{
			name:        "search",
			description: "Search for information",
		},
	}

	// Create mock LLM with tool call
	mockLLM := NewMockLLMWithToolCalls([]llms.ToolCall{
		{
			ID: "call_1",
			FunctionCall: &llms.FunctionCall{
				Name:      "route",
				Arguments: `{"next":"search"}`,
			},
		},
	})

	// This should not panic even with tool calls
	agent, err := CreateReactAgentTyped(mockLLM, tools)
	if err != nil {
		t.Fatalf("Failed to create ReAct agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

func TestCreateReactAgentTyped_NoTools(t *testing.T) {
	// Create agent with no tools
	tools := []tools.Tool{}

	mockLLM := NewMockLLMWithTextResponse([]string{
		"I don't need tools to answer this",
	})

	agent, err := CreateReactAgentTyped(mockLLM, tools)
	if err != nil {
		t.Fatalf("Failed to create ReAct agent with no tools: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

func TestReactAgentState(t *testing.T) {
	state := ReactAgentState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Hello"),
			llms.TextParts(llms.ChatMessageTypeAI, "Hi there!"),
		},
	}

	if len(state.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(state.Messages))
	}

	if state.Messages[0].Parts[0].(llms.TextContent).Text != "Hello" {
		t.Errorf("Expected first message to be 'Hello'")
	}
}

func TestCreateReactAgentWithCustomStateTyped(t *testing.T) {
	// Define custom state type
	type CustomState struct {
		Messages []llms.MessageContent `json:"messages"`
		Step     int                   `json:"step"`
		Debug    bool                  `json:"debug"`
	}

	// Create mock tools
	tools := []tools.Tool{
		&MockToolForReact{
			name:        "custom_tool",
			description: "A custom tool",
		},
	}

	// Create mock LLM
	mockLLM := NewMockLLMWithTextResponse([]string{
		"Custom processing complete",
	})

	// Define state handlers
	getMessages := func(s CustomState) []llms.MessageContent {
		return s.Messages
	}

	setMessages := func(s CustomState, msgs []llms.MessageContent) CustomState {
		s.Messages = msgs
		s.Step++
		return s
	}

	hasToolCalls := func(msgs []llms.MessageContent) bool {
		// For simplicity, always return false
		return false
	}

	// Create ReAct agent with custom state
	agent, err := CreateReactAgentWithCustomStateTyped(
		mockLLM,
		tools,
		getMessages,
		setMessages,
		hasToolCalls,
	)

	if err != nil {
		t.Fatalf("Failed to create custom ReAct agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

func TestCreateReactAgentWithCustomStateTyped_ComplexState(t *testing.T) {
	// Define complex custom state
	type ComplexState struct {
		Messages     []llms.MessageContent `json:"messages"`
		ToolCalls    []string              `json:"tool_calls"`
		Thoughts     []string              `json:"thoughts"`
		Observations []string              `json:"observations"`
		Complete     bool                  `json:"complete"`
	}

	tools := []tools.Tool{
		&MockToolForReact{
			name:        "complex_tool",
			description: "A complex tool",
		},
	}

	mockLLM := NewMockLLMWithTextResponse([]string{
		"Complex processing done",
	})

	getMessages := func(s ComplexState) []llms.MessageContent {
		return s.Messages
	}

	setMessages := func(s ComplexState, msgs []llms.MessageContent) ComplexState {
		s.Messages = msgs
		return s
	}

	hasToolCalls := func(msgs []llms.MessageContent) bool {
		// Check last message for tool calls
		if len(msgs) > 0 {
			// Simplified check
			return false
		}
		return false
	}

	agent, err := CreateReactAgentWithCustomStateTyped(
		mockLLM,
		tools,
		getMessages,
		setMessages,
		hasToolCalls,
	)

	if err != nil {
		t.Fatalf("Failed to create complex ReAct agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

func TestCreateReactAgentTyped_MultipleToolResponses(t *testing.T) {
	tools := []tools.Tool{
		&MockToolForReact{
			name:        "tool1",
			description: "First tool",
		},
		&MockToolForReact{
			name:        "tool2",
			description: "Second tool",
		},
	}

	// Create mock LLM with multiple responses
	mockLLM := &MockLLMForReact{
		responses: []llms.ContentChoice{
			{Content: "First response"},
			{Content: "Second response"},
			{Content: "Final answer"},
		},
		currentIndex: 0,
	}

	agent, err := CreateReactAgentTyped(mockLLM, tools)
	if err != nil {
		t.Fatalf("Failed to create ReAct agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

func TestCreateReactAgentTyped_ToolCallWithArguments(t *testing.T) {
	tools := []tools.Tool{
		&MockToolForReact{
			name:        "calculator",
			description: "Calculate something",
		},
	}

	// Create mock LLM with tool call and arguments
	mockLLM := NewMockLLMWithToolCalls([]llms.ToolCall{
		{
			ID: "call_calc",
			FunctionCall: &llms.FunctionCall{
				Name:      "calculator",
				Arguments: `{"input":"2+2"}`,
			},
		},
	})

	agent, err := CreateReactAgentTyped(mockLLM, tools)
	if err != nil {
		t.Fatalf("Failed to create ReAct agent with tool arguments: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

// Test edge cases
func TestCreateReactAgentTyped_EmptyToolName(t *testing.T) {
	tools := []tools.Tool{
		&MockToolForReact{
			name:        "", // Empty name
			description: "Tool with empty name",
		},
	}

	mockLLM := NewMockLLMWithTextResponse([]string{
		"Response",
	})

	// Should still create agent even with empty tool name
	agent, err := CreateReactAgentTyped(mockLLM, tools)
	if err != nil {
		t.Fatalf("Failed to create ReAct agent with empty tool name: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}

func TestCreateReactAgentTyped_LargeNumberOfTools(t *testing.T) {
	// Create many tools
	tools := make([]tools.Tool, 100)
	for i := 0; i < 100; i++ {
		tools[i] = &MockToolForReact{
			name:        fmt.Sprintf("tool_%d", i),
			description: fmt.Sprintf("Tool number %d", i),
		}
	}

	mockLLM := NewMockLLMWithTextResponse([]string{
		"Using many tools",
	})

	agent, err := CreateReactAgentTyped(mockLLM, tools)
	if err != nil {
		t.Fatalf("Failed to create ReAct agent with many tools: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}
}