package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// ToolNode is a reusable node that executes tool calls from the last AI message.
// It expects the state to be a map[string]any with a "messages" key containing []llms.MessageContent.
type ToolNode struct {
	Executor *ToolExecutor
}

// NewToolNode creates a new ToolNode with the given tools.
func NewToolNode(inputTools []tools.Tool) *ToolNode {
	return &ToolNode{
		Executor: NewToolExecutor(inputTools),
	}
}

// Invoke executes the tool calls found in the last message.
func (tn *ToolNode) Invoke(ctx context.Context, state any) (any, error) {
	mState, ok := state.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ToolNode expects state to be map[string]any, got %T", state)
	}

	messages, ok := mState["messages"].([]llms.MessageContent)
	if !ok {
		return nil, fmt.Errorf("ToolNode expects 'messages' key to be []llms.MessageContent")
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in state")
	}

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role != llms.ChatMessageTypeAI {
		// If the last message is not from AI, we can't execute tools.
		// In some graphs, this might be valid (e.g. if we just added a user message),
		// but typically ToolNode is called after AI.
		// We'll return empty map (no updates) or error?
		// Official LangGraph ToolNode typically expects to be called when there are tool calls.
		return nil, fmt.Errorf("last message is not an AI message")
	}

	var toolMessages []llms.MessageContent

	for _, part := range lastMsg.Parts {
		if tc, ok := part.(llms.ToolCall); ok {
			// Parse arguments to get input
			var args map[string]any
			// Arguments is a JSON string - ignore error, will use raw string if unmarshal fails
			_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

			inputVal := ""
			if val, ok := args["input"].(string); ok {
				inputVal = val
			} else {
				// Fallback: pass the whole arguments string if "input" key is missing
				// This depends on how the tool expects input.
				inputVal = tc.FunctionCall.Arguments
			}

			// Execute tool
			res, err := tn.Executor.Execute(ctx, ToolInvocation{
				Tool:      tc.FunctionCall.Name,
				ToolInput: inputVal,
			})
			if err != nil {
				res = fmt.Sprintf("Error executing tool %s: %v", tc.FunctionCall.Name, err)
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

	if len(toolMessages) == 0 {
		// No tool calls found
		return map[string]any{}, nil
	}

	return map[string]any{
		"messages": toolMessages,
	}, nil
}
