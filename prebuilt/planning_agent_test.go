package prebuilt

import (
	"context"
	"fmt"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// MockPlanningLLM is a mock LLM that returns a workflow plan
type MockPlanningLLM struct {
	planJSON      string
	responses     []llms.ContentResponse
	callCount     int
	capturedCalls [][]llms.MessageContent
}

func (m *MockPlanningLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	m.capturedCalls = append(m.capturedCalls, messages)

	// First call is the planning call
	if m.callCount == 0 {
		m.callCount++
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{
					Content: m.planJSON,
				},
			},
		}, nil
	}

	// Subsequent calls use predefined responses
	if m.callCount-1 < len(m.responses) {
		resp := m.responses[m.callCount-1]
		m.callCount++
		return &resp, nil
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: "No more responses"},
		},
	}, nil
}

func (m *MockPlanningLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

func TestCreatePlanningAgent_SimpleWorkflow(t *testing.T) {
	// Define test nodes
	testNodes := []*graph.Node{
		{
			Name:        "research",
			Description: "Research and gather information",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				mState := state.(map[string]interface{})
				messages := mState["messages"].([]llms.MessageContent)

				researchMsg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Research completed")},
				}

				return map[string]interface{}{
					"messages": append(messages, researchMsg),
				}, nil
			},
		},
		{
			Name:        "analyze",
			Description: "Analyze the gathered information",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				mState := state.(map[string]interface{})
				messages := mState["messages"].([]llms.MessageContent)

				analyzeMsg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Analysis completed")},
				}

				return map[string]interface{}{
					"messages": append(messages, analyzeMsg),
				}, nil
			},
		},
		{
			Name:        "report",
			Description: "Generate a report from the analysis",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				mState := state.(map[string]interface{})
				messages := mState["messages"].([]llms.MessageContent)

				reportMsg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Report generated")},
				}

				return map[string]interface{}{
					"messages": append(messages, reportMsg),
				}, nil
			},
		},
	}

	// Create a workflow plan JSON
	planJSON := `{
		"nodes": [
			{"name": "research", "type": "process"},
			{"name": "analyze", "type": "process"},
			{"name": "report", "type": "process"}
		],
		"edges": [
			{"from": "START", "to": "research"},
			{"from": "research", "to": "analyze"},
			{"from": "analyze", "to": "report"},
			{"from": "report", "to": "END"}
		]
	}`

	// Setup Mock LLM
	mockLLM := &MockPlanningLLM{
		planJSON:  planJSON,
		responses: []llms.ContentResponse{},
	}

	// Create Planning Agent
	agent, err := CreatePlanningAgent(mockLLM, testNodes, []tools.Tool{})
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// Initial State
	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Please research, analyze, and create a report"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Verify Result
	mState := res.(map[string]interface{})
	messages := mState["messages"].([]llms.MessageContent)

	// Expected messages:
	// 0: Human "Please research, analyze, and create a report"
	// 1: AI "Workflow plan created with 3 nodes and 4 edges"
	// 2: AI "Research completed"
	// 3: AI "Analysis completed"
	// 4: AI "Report generated"
	assert.GreaterOrEqual(t, len(messages), 5)

	// Check that all expected messages are present
	assert.Equal(t, llms.ChatMessageTypeHuman, messages[0].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[1].Role)

	// Verify workflow execution
	foundResearch := false
	foundAnalyze := false
	foundReport := false

	for _, msg := range messages {
		if msg.Role == llms.ChatMessageTypeAI {
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					if textPart.Text == "Research completed" {
						foundResearch = true
					}
					if textPart.Text == "Analysis completed" {
						foundAnalyze = true
					}
					if textPart.Text == "Report generated" {
						foundReport = true
					}
				}
			}
		}
	}

	assert.True(t, foundResearch, "Research step should have been executed")
	assert.True(t, foundAnalyze, "Analyze step should have been executed")
	assert.True(t, foundReport, "Report step should have been executed")
}

func TestCreatePlanningAgent_WithVerbose(t *testing.T) {
	// Define a simple test node
	testNodes := []*graph.Node{
		{
			Name:        "simple_task",
			Description: "A simple task",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				return state, nil
			},
		},
	}

	// Create a simple workflow plan
	planJSON := `{
		"nodes": [
			{"name": "simple_task", "type": "process"}
		],
		"edges": [
			{"from": "START", "to": "simple_task"},
			{"from": "simple_task", "to": "END"}
		]
	}`

	// Setup Mock LLM
	mockLLM := &MockPlanningLLM{
		planJSON: planJSON,
	}

	// Create Planning Agent with verbose option
	agent, err := CreatePlanningAgent(mockLLM, testNodes, []tools.Tool{}, WithVerbose(true))
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Execute simple task"),
		},
	}

	// Run Agent (verbose output will be printed to stdout)
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestParseWorkflowPlan_ValidJSON(t *testing.T) {
	planText := `{
		"nodes": [
			{"name": "node1", "type": "process"}
		],
		"edges": [
			{"from": "START", "to": "node1"},
			{"from": "node1", "to": "END"}
		]
	}`

	plan, err := parseWorkflowPlan(planText)
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, 1, len(plan.Nodes))
	assert.Equal(t, 2, len(plan.Edges))
	assert.Equal(t, "node1", plan.Nodes[0].Name)
}

func TestParseWorkflowPlan_MarkdownCodeBlock(t *testing.T) {
	planText := "```json\n" + `{
		"nodes": [
			{"name": "node1", "type": "process"}
		],
		"edges": [
			{"from": "START", "to": "node1"}
		]
	}` + "\n```"

	plan, err := parseWorkflowPlan(planText)
	assert.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, 1, len(plan.Nodes))
	assert.Equal(t, 1, len(plan.Edges))
}

func TestParseWorkflowPlan_InvalidJSON(t *testing.T) {
	planText := "This is not valid JSON"

	_, err := parseWorkflowPlan(planText)
	assert.Error(t, err)
}

func TestParseWorkflowPlan_EmptyNodes(t *testing.T) {
	planText := `{
		"nodes": [],
		"edges": []
	}`

	_, err := parseWorkflowPlan(planText)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no nodes")
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with text around",
			input:    "Here is the plan:\n{\"key\": \"value\"}\nThat's it!",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.JSONEq(t, tt.expected, result)
		})
	}
}

func TestBuildNodeDescriptions(t *testing.T) {
	nodes := []*graph.Node{
		{Name: "node1", Description: "First node"},
		{Name: "node2", Description: "Second node"},
	}

	result := buildNodeDescriptions(nodes)

	assert.Contains(t, result, "Available nodes:")
	assert.Contains(t, result, "node1: First node")
	assert.Contains(t, result, "node2: Second node")
}

func TestBuildPlanningPrompt(t *testing.T) {
	nodeDescriptions := "1. research: Research node\n2. analyze: Analyze node"

	prompt := buildPlanningPrompt(nodeDescriptions)

	assert.Contains(t, prompt, "workflow planning assistant")
	assert.Contains(t, prompt, nodeDescriptions)
	assert.Contains(t, prompt, "JSON format")
	assert.Contains(t, prompt, "START")
	assert.Contains(t, prompt, "END")
}

func TestCreatePlanningAgent_NodeNotFound(t *testing.T) {
	// Define test nodes
	testNodes := []*graph.Node{
		{
			Name:        "existing_node",
			Description: "An existing node",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				return state, nil
			},
		},
	}

	// Create a plan that references a non-existent node
	planJSON := `{
		"nodes": [
			{"name": "non_existent_node", "type": "process"}
		],
		"edges": [
			{"from": "START", "to": "non_existent_node"},
			{"from": "non_existent_node", "to": "END"}
		]
	}`

	mockLLM := &MockPlanningLLM{
		planJSON: planJSON,
	}

	agent, err := CreatePlanningAgent(mockLLM, testNodes, []tools.Tool{})
	assert.NoError(t, err)

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Execute task"),
		},
	}

	// Run Agent - should fail because node doesn't exist
	_, err = agent.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Example test showing a more complex workflow with conditional edges
func TestCreatePlanningAgent_ComplexWorkflow(t *testing.T) {
	callOrder := []string{}

	testNodes := []*graph.Node{
		{
			Name:        "validate",
			Description: "Validate input data",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				callOrder = append(callOrder, "validate")
				mState := state.(map[string]interface{})
				messages := mState["messages"].([]llms.MessageContent)

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Validation completed")},
				}

				return map[string]interface{}{
					"messages": append(messages, msg),
				}, nil
			},
		},
		{
			Name:        "process",
			Description: "Process the data",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				callOrder = append(callOrder, "process")
				mState := state.(map[string]interface{})
				messages := mState["messages"].([]llms.MessageContent)

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Processing completed")},
				}

				return map[string]interface{}{
					"messages": append(messages, msg),
				}, nil
			},
		},
		{
			Name:        "save",
			Description: "Save the results",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				callOrder = append(callOrder, "save")
				mState := state.(map[string]interface{})
				messages := mState["messages"].([]llms.MessageContent)

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Results saved")},
				}

				return map[string]interface{}{
					"messages": append(messages, msg),
				}, nil
			},
		},
	}

	planJSON := `{
		"nodes": [
			{"name": "validate", "type": "process"},
			{"name": "process", "type": "process"},
			{"name": "save", "type": "process"}
		],
		"edges": [
			{"from": "START", "to": "validate"},
			{"from": "validate", "to": "process"},
			{"from": "process", "to": "save"},
			{"from": "save", "to": "END"}
		]
	}`

	mockLLM := &MockPlanningLLM{
		planJSON: planJSON,
	}

	agent, err := CreatePlanningAgent(mockLLM, testNodes, []tools.Tool{})
	assert.NoError(t, err)

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Validate, process, and save the data"),
		},
	}

	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Verify execution order
	assert.Equal(t, []string{"validate", "process", "save"}, callOrder)

	// Verify messages
	mState := res.(map[string]interface{})
	messages := mState["messages"].([]llms.MessageContent)
	assert.GreaterOrEqual(t, len(messages), 5)
}

// Example usage documentation
func ExampleCreatePlanningAgent() {
	// Define your custom nodes
	nodes := []*graph.Node{
		{
			Name:        "fetch_data",
			Description: "Fetch data from API",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				// Your implementation
				fmt.Println("Fetching data...")
				return state, nil
			},
		},
		{
			Name:        "transform_data",
			Description: "Transform the fetched data",
			Function: func(ctx context.Context, state interface{}) (interface{}, error) {
				// Your implementation
				fmt.Println("Transforming data...")
				return state, nil
			},
		},
	}

	// Create your LLM model (this is a placeholder)
	var model llms.Model // = your actual LLM model

	// Create the planning agent
	agent, _ := CreatePlanningAgent(
		model,
		nodes,
		[]tools.Tool{},
		WithVerbose(true),
	)

	// Use the agent
	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Fetch and transform the data"),
		},
	}

	result, _ := agent.Invoke(context.Background(), initialState)
	fmt.Printf("Result: %v\n", result)

	// Output will show the planning and execution steps
}
