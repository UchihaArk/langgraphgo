package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestCreateSupervisor(t *testing.T) {
	// Setup Mock LLM
	// 1. Route to Agent1
	// 2. Route to Agent2
	// 3. Route to FINISH
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Agent1"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Agent2"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								FunctionCall: &llms.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "FINISH"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	// Setup Mock Agents
	agent1Graph := graph.NewStateGraph()
	agent1Graph.AddNode("run", "run", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"messages": []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeAI, "Agent1 done"),
			},
		}, nil
	})
	agent1Graph.SetEntryPoint("run")
	agent1Graph.AddEdge("run", graph.END)
	agent1, err := agent1Graph.Compile()
	assert.NoError(t, err)

	agent2Graph := graph.NewStateGraph()
	agent2Graph.AddNode("run", "run", func(ctx context.Context, state any) (any, error) {
		return map[string]any{
			"messages": []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeAI, "Agent2 done"),
			},
		}, nil
	})
	agent2Graph.SetEntryPoint("run")
	agent2Graph.AddEdge("run", graph.END)
	agent2, err := agent2Graph.Compile()
	assert.NoError(t, err)

	// Create Supervisor
	members := map[string]*graph.StateRunnable{
		"Agent1": agent1,
		"Agent2": agent2,
	}
	supervisor, err := CreateSupervisor(mockLLM, members)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Start"),
		},
	}

	// Run Supervisor
	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)

	// Expected messages:
	// 0: Human "Start"
	// 1: Agent1 "Agent1 done"
	// 2: Agent2 "Agent2 done"
	// Note: Supervisor itself doesn't add messages in our implementation, only routes.
	// But agents append to messages.

	assert.Equal(t, 3, len(messages))
	assert.Equal(t, "Start", messages[0].Parts[0].(llms.TextContent).Text)
	assert.Equal(t, "Agent1 done", messages[1].Parts[0].(llms.TextContent).Text)
	assert.Equal(t, "Agent2 done", messages[2].Parts[0].(llms.TextContent).Text)

	// Verify "next" state
	assert.Equal(t, "FINISH", mState["next"])
}
