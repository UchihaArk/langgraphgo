package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// ReactAgentState represents the state for a ReAct agent
type ReactAgentState struct {
	Messages []llms.MessageContent `json:"messages"`
}

// CreateReactAgentTyped creates a new typed ReAct agent graph
func CreateReactAgentTyped(model llms.Model, inputTools []tools.Tool) (*graph.StateRunnableTyped[ReactAgentState], error) {
	// Define the tool executor
	toolExecutor := NewToolExecutor(inputTools)

	// Define the graph
	workflow := graph.NewStateGraphTyped[ReactAgentState]()

	// Define the state schema
	schema := graph.NewStructSchema(
		ReactAgentState{},
		func(current, new ReactAgentState) (ReactAgentState, error) {
			// Append new messages to current messages
			current.Messages = append(current.Messages, new.Messages...)
			return current, nil
		},
	)
	workflow.SetSchema(schema)

	// Define the agent node
	workflow.AddNode("agent", "ReAct agent decision maker", func(ctx context.Context, state ReactAgentState) (ReactAgentState, error) {
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
								"description": fmt.Sprintf("Input for the %s tool", t.Name()),
							},
						},
						"required": []string{"input"},
					},
				},
			})
		}

		// System prompt for ReAct
		systemPrompt := `You are a helpful assistant. Use the provided tools to answer the user's question.
Follow this format:
1. Thought: what should I do next?
2. Action: the action to take (should be one of the provided tools)
3. Observation: the result of the action
4. ... (repeat Thought/Action/Observation as needed)
5. Final Answer: the final answer to the user's question

When you have enough information to answer the user's question, provide the Final Answer without using any tools.`

		// Prepare messages
		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		}
		messages = append(messages, state.Messages...)

		// Call model with tools
		resp, err := model.GenerateContent(ctx, messages,
			llms.WithTools(toolDefs),
			llms.WithToolChoice("auto"),
		)
		if err != nil {
			return ReactAgentState{}, err
		}

		choice := resp.Choices[0]

		// Add the model's response to messages
		newMessages := state.Messages
		newMessages = append(newMessages, llms.TextParts(llms.ChatMessageTypeAI, choice.Content))

		// Check if the model made a tool call
		if len(choice.ToolCalls) > 0 {
			// Execute the tool
			tc := choice.ToolCalls[0]

			// Parse arguments to get the input value
			var args struct {
				Input string `json:"input"`
			}
			inputVal := tc.FunctionCall.Arguments
			if tc.FunctionCall.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)
				inputVal = args.Input
			}

			result, err := toolExecutor.Execute(ctx, ToolInvocation{
				Tool:      tc.FunctionCall.Name,
				ToolInput: inputVal,
			})
			if err != nil {
				return ReactAgentState{}, fmt.Errorf("tool execution failed: %w", err)
			}

			// Add tool response to messages
			toolMessage := llms.TextParts(
				llms.ChatMessageTypeTool,
				fmt.Sprintf("Tool %s returned: %s", tc.FunctionCall.Name, result),
			)
			newMessages = append(newMessages, toolMessage)

			return ReactAgentState{
				Messages: newMessages,
			}, nil
		}

		// No tool call, this should be the final answer
		return ReactAgentState{
			Messages: newMessages,
		}, nil
	})

	// Define the action node (tool executor)
	workflow.AddNode("action", "Execute tools", func(ctx context.Context, state ReactAgentState) (ReactAgentState, error) {
		// Get the last AI message which should contain a tool call
		if len(state.Messages) == 0 {
			return ReactAgentState{}, fmt.Errorf("no messages in state")
		}

		lastMessage := state.Messages[len(state.Messages)-1]
		if lastMessage.Role != llms.ChatMessageTypeAI {
			return ReactAgentState{}, fmt.Errorf("last message is not from AI")
		}

		// This node shouldn't be reached directly anymore since we execute tools in the agent node
		// But keeping it for backward compatibility
		return state, nil
	})

	// Define conditional routing from agent
	workflow.AddConditionalEdge("agent", func(ctx context.Context, state ReactAgentState) string {
		if len(state.Messages) == 0 {
			return "action"
		}

		lastMessage := state.Messages[len(state.Messages)-1]
		if lastMessage.Role == llms.ChatMessageTypeAI {
			// Check if there were tool calls in the AI message
			// This is a simplified check - in practice, we'd need to inspect the message content
			// For now, assume if the last message is a tool response, we should continue
			if len(state.Messages) >= 2 && state.Messages[len(state.Messages)-2].Role == llms.ChatMessageTypeTool {
				// Last tool was executed, continue to agent
				return "agent"
			}
			// Check if the AI message contains tool calls (simplified)
			// In a real implementation, we'd parse the message content for tool calls
			// For now, assume we continue to agent
			return "agent"
		}

		return graph.END
	})

	// Set entry point
	workflow.SetEntryPoint("agent")

	// Compile and return
	return workflow.Compile()
}

// CreateReactAgentWithCustomStateTyped creates a typed ReAct agent with custom state type
func CreateReactAgentWithCustomStateTyped[S any](
	model llms.Model,
	inputTools []tools.Tool,
	getMessages func(S) []llms.MessageContent,
	setMessages func(S, []llms.MessageContent) S,
	hasToolCalls func([]llms.MessageContent) bool,
) (*graph.StateRunnableTyped[S], error) {
	// Define the tool executor
	toolExecutor := NewToolExecutor(inputTools)

	// Define the graph
	workflow := graph.NewStateGraphTyped[S]()

	// Define the agent node
	workflow.AddNode("agent", "ReAct agent decision maker", func(ctx context.Context, state S) (S, error) {
		messages := getMessages(state)

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
								"description": fmt.Sprintf("Input for the %s tool", t.Name()),
							},
						},
						"required": []string{"input"},
					},
				},
			})
		}

		// System prompt for ReAct
		systemPrompt := `You are a helpful assistant. Use the provided tools to answer the user's question.
Follow this format:
1. Thought: what should I do next?
2. Action: the action to take (should be one of the provided tools)
3. Observation: the result of the action
4. ... (repeat Thought/Action/Observation as needed)
5. Final Answer: the final answer to the user's question

When you have enough information to answer the user's question, provide the Final Answer without using any tools.`

		// Prepare messages
		inputMessages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		}
		inputMessages = append(inputMessages, messages...)

		// Call model with tools
		resp, err := model.GenerateContent(ctx, inputMessages,
			llms.WithTools(toolDefs),
			llms.WithToolChoice("auto"),
		)
		if err != nil {
			return state, err
		}

		choice := resp.Choices[0]

		// Add the model's response to messages
		newMessages := messages
		newMessages = append(newMessages, llms.TextParts(llms.ChatMessageTypeAI, choice.Content))

		// Check if the model made a tool call
		if len(choice.ToolCalls) > 0 {
			// Execute the tool
			tc := choice.ToolCalls[0]

			// Parse arguments to get the input value
			var args struct {
				Input string `json:"input"`
			}
			inputVal := tc.FunctionCall.Arguments
			if tc.FunctionCall.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)
				inputVal = args.Input
			}

			result, err := toolExecutor.Execute(ctx, ToolInvocation{
				Tool:      tc.FunctionCall.Name,
				ToolInput: inputVal,
			})
			if err != nil {
				return state, fmt.Errorf("tool execution failed: %w", err)
			}

			// Add tool response to messages
			toolMessage := llms.TextParts(
				llms.ChatMessageTypeTool,
				fmt.Sprintf("Tool %s returned: %s", tc.FunctionCall.Name, result),
			)
			newMessages = append(newMessages, toolMessage)

			return setMessages(state, newMessages), nil
		}

		// No tool call, this should be the final answer
		return setMessages(state, newMessages), nil
	})

	// Define conditional routing from agent
	workflow.AddConditionalEdge("agent", func(ctx context.Context, state S) string {
		messages := getMessages(state)
		if len(messages) == 0 {
			return graph.END
		}

		// Check if the last message contains tool calls
		if hasToolCalls(messages) {
			// Continue with tool execution
			return "agent" // Since we execute tools in the agent node now
		}

		// No tool calls, we're done
		return graph.END
	})

	// Set entry point
	workflow.SetEntryPoint("agent")

	// Compile and return
	return workflow.Compile()
}
