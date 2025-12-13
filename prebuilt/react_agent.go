package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// CreateReactAgent creates a new ReAct agent graph
func CreateReactAgent(model llms.Model, inputTools []tools.Tool) (*graph.StateRunnable, error) {
	// Define the tool executor
	toolExecutor := NewToolExecutor(inputTools)

	// Define the graph
	workflow := graph.NewStateGraph()

	// Define the state schema
	// We use a MapSchema with AppendReducer for messages
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(agentSchema)

	// Define the agent node
	workflow.AddNode("agent", "ReAct agent decision maker", func(ctx context.Context, state any) (any, error) {
		mState, ok := state.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid state type: %T", state)
		}

		messages, ok := mState["messages"].([]llms.MessageContent)
		if !ok {
			return nil, fmt.Errorf("messages key not found or invalid type")
		}

		// Convert tools to ToolInfo for the model
		var toolDefs []llms.Tool
		for _, t := range inputTools {
			toolDefs = append(toolDefs, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"input": map[string]any{
								"type":        "string",
								"description": "The input query for the tool",
							},
						},
						"required":             []string{"input"},
						"additionalProperties": false,
					},
				},
			})
		}

		// We need to pass tools to the model
		opts := []llms.CallOption{
			llms.WithTools(toolDefs),
		}

		resp, err := model.GenerateContent(ctx, messages, opts...)
		if err != nil {
			return nil, err
		}

		choice := resp.Choices[0]

		// Create AIMessage
		aiMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeAI,
		}

		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llms.TextPart(choice.Content))
		}

		// Handle tool calls
		if len(choice.ToolCalls) > 0 {
			for _, tc := range choice.ToolCalls {
				// ToolCall implements ContentPart
				aiMsg.Parts = append(aiMsg.Parts, tc)
			}
		}

		return map[string]any{
			"messages": []llms.MessageContent{aiMsg},
		}, nil
	})

	// Define the tools node
	workflow.AddNode("tools", "Tool execution node", func(ctx context.Context, state any) (any, error) {
		mState, ok := state.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid state")
		}

		messages := mState["messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		if lastMsg.Role != llms.ChatMessageTypeAI {
			return nil, fmt.Errorf("last message is not an AI message")
		}

		var toolMessages []llms.MessageContent

		for _, part := range lastMsg.Parts {
			if tc, ok := part.(llms.ToolCall); ok {
				// Parse arguments to get input
				var args map[string]any
				if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
					// If unmarshal fails, try to use the raw string if it's not JSON object
				}

				inputVal := ""
				if val, ok := args["input"].(string); ok {
					inputVal = val
				} else {
					inputVal = tc.FunctionCall.Arguments
				}

				// Execute tool
				res, err := toolExecutor.Execute(ctx, ToolInvocation{
					Tool:      tc.FunctionCall.Name,
					ToolInput: inputVal,
				})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}

				// Create ToolMessage
				toolMsg := llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: tc.ID,
							Name:       tc.FunctionCall.Name,
							Content:    res,
						},
					},
				}
				toolMessages = append(toolMessages, toolMsg)
			}
		}

		return map[string]any{
			"messages": toolMessages,
		}, nil
	})

	// Define edges
	workflow.SetEntryPoint("agent")

	workflow.AddConditionalEdge("agent", func(ctx context.Context, state any) string {
		mState := state.(map[string]any)
		messages := mState["messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		hasToolCalls := false
		for _, part := range lastMsg.Parts {
			if _, ok := part.(llms.ToolCall); ok {
				hasToolCalls = true
				break
			}
		}

		if hasToolCalls {
			return "tools"
		}
		return graph.END
	})

	workflow.AddEdge("tools", "agent")

	return workflow.Compile()
}
