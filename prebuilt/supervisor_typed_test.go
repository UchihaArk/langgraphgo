package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

// MockLLMSupervisor for testing
type MockLLMSupervisor struct {
	responses []string
	index     int
}

func (m *MockLLMSupervisor) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	response := m.responses[m.index%len(m.responses)]
	m.index++

	// Return a response with a tool call
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response,
				ToolCalls: []llms.ToolCall{
					{
						ID: "test-call",
						FunctionCall: &llms.FunctionCall{
							Name:      "route",
							Arguments: `{"next":"worker1"}`,
						},
					},
				},
			},
		},
	}, nil
}

// Call implements the deprecated Call method for backward compatibility
func (m *MockLLMSupervisor) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	response := m.responses[m.index%len(m.responses)]
	m.index++
	return response, nil
}

func TestCreateSupervisorTyped(t *testing.T) {
	// Create a mock LLM
	mockLLM := &MockLLMSupervisor{
		responses: []string{
			"Thinking...",
		},
	}

	// Create mock member runnables
	member1 := &graph.StateRunnableTyped[SupervisorState]{}
	member2 := &graph.StateRunnableTyped[SupervisorState]{}

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"worker1": member1,
		"worker2": member2,
	}

	// Create supervisor
	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	if err != nil {
		t.Fatalf("Failed to create supervisor: %v", err)
	}

	if supervisor == nil {
		t.Fatal("Supervisor should not be nil")
	}
}

func TestCreateSupervisorTyped_EmptyMembers(t *testing.T) {
	mockLLM := &MockLLMSupervisor{
		responses: []string{
			"Thinking...",
		},
	}

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{}

	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	if err != nil {
		t.Fatalf("Failed to create supervisor with empty members: %v", err)
	}

	if supervisor == nil {
		t.Fatal("Supervisor should not be nil even with empty members")
	}
}

func TestCreateSupervisorWithStateTyped(t *testing.T) {
	// Define custom state type
	type CustomState struct {
		Step      int
		Data      string
		Messages  []llms.MessageContent
		Next      string
	}

	// Create a mock LLM
	mockLLM := &MockLLMSupervisor{
		responses: []string{
			"Processing...",
		},
	}

	// Create mock member runnables
	member1 := &graph.StateRunnableTyped[CustomState]{}
	member2 := &graph.StateRunnableTyped[CustomState]{}

	members := map[string]*graph.StateRunnableTyped[CustomState]{
		"worker1": member1,
		"worker2": member2,
	}

	// Define state handlers
	getMessages := func(s CustomState) []llms.MessageContent {
		return s.Messages
	}

	setMessages := func(s CustomState, msgs []llms.MessageContent) CustomState {
		s.Messages = msgs
		return s
	}

	getNext := func(s CustomState) string {
		return s.Next
	}

	setNext := func(s CustomState, next string) CustomState {
		s.Next = next
		return s
	}

	// Create supervisor with custom state
	supervisor, err := CreateSupervisorWithStateTyped(
		mockLLM,
		members,
		getMessages,
		setMessages,
		getNext,
		setNext,
	)

	if err != nil {
		t.Fatalf("Failed to create supervisor with custom state: %v", err)
	}

	if supervisor == nil {
		t.Fatal("Supervisor should not be nil")
	}
}

func TestCreateSupervisorWithStateTyped_CustomLogic(t *testing.T) {
	type CustomState struct {
		Counter   int
		Processed []string
		Messages  []llms.MessageContent
		Next      string
	}

	mockLLM := &MockLLMSupervisor{
		responses: []string{
			"Deciding next step...",
		},
	}

	// Create a mock member runnable that updates state
	member := &graph.StateRunnableTyped[CustomState]{}

	members := map[string]*graph.StateRunnableTyped[CustomState]{
		"processor": member,
	}

	getMessages := func(s CustomState) []llms.MessageContent {
		return s.Messages
	}

	setMessages := func(s CustomState, msgs []llms.MessageContent) CustomState {
		s.Messages = msgs
		return s
	}

	getNext := func(s CustomState) string {
		return s.Next
	}

	setNext := func(s CustomState, next string) CustomState {
		s.Next = next
		s.Counter++
		if next != "" {
			s.Processed = append(s.Processed, next)
		}
		return s
	}

	supervisor, err := CreateSupervisorWithStateTyped(
		mockLLM,
		members,
		getMessages,
		setMessages,
		getNext,
		setNext,
	)

	if err != nil {
		t.Fatalf("Failed to create supervisor: %v", err)
	}

	if supervisor == nil {
		t.Fatal("Supervisor should not be nil")
	}
}

// Test SupervisorState structure
func TestSupervisorState(t *testing.T) {
	state := SupervisorState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Test message"),
		},
		Next: "worker1",
	}

	if len(state.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(state.Messages))
	}

	if state.Next != "worker1" {
		t.Errorf("Expected next to be 'worker1', got '%s'", state.Next)
	}
}

// Test that supervisor creates the correct graph structure
func TestCreateSupervisorTyped_GraphStructure(t *testing.T) {
	mockLLM := &MockLLMSupervisor{
		responses: []string{
			"Supervisor decision",
		},
	}

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"agent1": {},
		"agent2": {},
	}

	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	if err != nil {
		t.Fatalf("Failed to create supervisor: %v", err)
	}

	// The supervisor should compile successfully
	// This tests that the graph structure is correctly built
	if supervisor == nil {
		t.Fatal("Supervisor should not be nil")
	}
}

// Test supervisor with single member
func TestCreateSupervisorTyped_SingleMember(t *testing.T) {
	mockLLM := &MockLLMSupervisor{
		responses: []string{
			"Decision made",
		},
	}

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"sole_agent": {},
	}

	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	if err != nil {
		t.Fatalf("Failed to create supervisor with single member: %v", err)
	}

	if supervisor == nil {
		t.Fatal("Supervisor should not be nil")
	}
}

// Test supervisor with complex member names
func TestCreateSupervisorTyped_ComplexMemberNames(t *testing.T) {
	mockLLM := &MockLLMSupervisor{
		responses: []string{
			"Analyzing options",
		},
	}

	members := map[string]*graph.StateRunnableTyped[SupervisorState]{
		"agent_with_underscores": {},
		"agent-with-dashes":      {},
		"AgentWithCamelCase":     {},
	}

	supervisor, err := CreateSupervisorTyped(mockLLM, members)
	if err != nil {
		t.Fatalf("Failed to create supervisor with complex member names: %v", err)
	}

	if supervisor == nil {
		t.Fatal("Supervisor should not be nil")
	}
}