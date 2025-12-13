package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

// SupervisorState represents the state for a supervisor workflow
type SupervisorState struct {
	Messages []llms.MessageContent `json:"messages"`
	Next     string                `json:"next,omitempty"`
}

// CreateSupervisorTyped creates a typed supervisor graph that orchestrates multiple agents
func CreateSupervisorTyped(model llms.Model, members map[string]*graph.StateRunnableTyped[SupervisorState]) (*graph.StateRunnableTyped[SupervisorState], error) {
	workflow := graph.NewStateGraphTyped[SupervisorState]()

	// Define state schema with merge logic
	schema := graph.NewStructSchema(
		SupervisorState{},
		func(current, new SupervisorState) (SupervisorState, error) {
			// Append new messages to current messages
			current.Messages = append(current.Messages, new.Messages...)
			// Update next if specified
			if new.Next != "" {
				current.Next = new.Next
			}
			return current, nil
		},
	)
	workflow.SetSchema(schema)

	// Get member names
	var memberNames []string
	for name := range members {
		memberNames = append(memberNames, name)
	}

	// Define supervisor node
	workflow.AddNode("supervisor", "Supervisor orchestration node", func(ctx context.Context, state SupervisorState) (SupervisorState, error) {
		// Define routing function
		options := append(memberNames, "FINISH")
		routeTool := llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "route",
				Description: "Select the next role.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"next": map[string]any{
							"type": "string",
							"enum": options,
						},
					},
					"required": []string{"next"},
				},
			},
		}

		// System prompt
		systemPrompt := fmt.Sprintf(
			"You are a supervisor tasked with managing a conversation between the following workers: %s. Given the following user request, respond with the worker to act next. Each worker will perform a task and respond with their results and status. When finished, respond with FINISH. You MUST use the 'route' tool to select the next worker or to finish. Do not provide any other text response.",
			strings.Join(memberNames, ", "),
		)

		// Prepare messages
		inputMessages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		}
		inputMessages = append(inputMessages, state.Messages...)

		// Call model
		resp, err := model.GenerateContent(ctx, inputMessages,
			llms.WithTools([]llms.Tool{routeTool}),
			llms.WithToolChoice("auto"), // Let model decide, but prompt strongly encourages it
		)
		if err != nil {
			return SupervisorState{}, err
		}

		choice := resp.Choices[0]
		if len(choice.ToolCalls) == 0 {
			// If no tool call, assume FINISH or error?
			// With ToolChoice("route"), it should call it.
			return SupervisorState{}, fmt.Errorf("supervisor did not select a next step")
		}

		// Parse selection
		tc := choice.ToolCalls[0]
		var args struct {
			Next string `json:"next"`
		}
		if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
			return SupervisorState{}, fmt.Errorf("failed to parse route arguments: %w", err)
		}

		// Return the decision in the Next field
		return SupervisorState{
			Messages: state.Messages, // Preserve existing messages
			Next:     args.Next,
		}, nil
	})

	// Add member nodes
	for name, runnable := range members {
		// Create a wrapper node that calls the member
		workflow.AddNode(name, fmt.Sprintf("Agent: %s", name), func(ctx context.Context, state SupervisorState) (SupervisorState, error) {
			// Execute the member agent
			result, err := runnable.Invoke(ctx, state)
			if err != nil {
				return SupervisorState{}, fmt.Errorf("agent %s failed: %w", name, err)
			}
			return result, nil
		})
	}

	// Add conditional routing from supervisor
	workflow.AddConditionalEdge("supervisor", func(ctx context.Context, state SupervisorState) string {
		if state.Next == "FINISH" {
			return graph.END
		}
		return state.Next
	})

	// Add edges from members back to supervisor
	for name := range members {
		workflow.AddEdge(name, "supervisor")
	}

	// Set entry point
	workflow.SetEntryPoint("supervisor")

	// Compile and return
	return workflow.Compile()
}

// CreateSupervisorWithStateTyped creates a typed supervisor with custom state type
func CreateSupervisorWithStateTyped[S any](
	model llms.Model,
	members map[string]*graph.StateRunnableTyped[S],
	getMessages func(S) []llms.MessageContent,
	updateMessages func(S, []llms.MessageContent) S,
	getNext func(S) string,
	setNext func(S, string) S,
) (*graph.StateRunnableTyped[S], error) {
	workflow := graph.NewStateGraphTyped[S]()

	// Get member names
	var memberNames []string
	for name := range members {
		memberNames = append(memberNames, name)
	}

	// Define supervisor node
	workflow.AddNode("supervisor", "Supervisor orchestration node", func(ctx context.Context, state S) (S, error) {
		messages := getMessages(state)

		// Define routing function
		options := append(memberNames, "FINISH")
		routeTool := llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "route",
				Description: "Select the next role.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"next": map[string]any{
							"type": "string",
							"enum": options,
						},
					},
					"required": []string{"next"},
				},
			},
		}

		// System prompt
		systemPrompt := fmt.Sprintf(
			"You are a supervisor tasked with managing a conversation between the following workers: %s. Given the following user request, respond with the worker to act next. Each worker will perform a task and respond with their results and status. When finished, respond with FINISH. You MUST use the 'route' tool to select the next worker or to finish. Do not provide any other text response.",
			strings.Join(memberNames, ", "),
		)

		// Prepare messages
		inputMessages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		}
		inputMessages = append(inputMessages, messages...)

		// Call model
		resp, err := model.GenerateContent(ctx, inputMessages,
			llms.WithTools([]llms.Tool{routeTool}),
			llms.WithToolChoice("auto"),
		)
		if err != nil {
			return state, err
		}

		choice := resp.Choices[0]
		if len(choice.ToolCalls) == 0 {
			return state, fmt.Errorf("supervisor did not select a next step")
		}

		// Parse selection
		tc := choice.ToolCalls[0]
		var args struct {
			Next string `json:"next"`
		}
		if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
			return state, fmt.Errorf("failed to parse route arguments: %w", err)
		}

		// Update state with the next decision
		return setNext(state, args.Next), nil
	})

	// Add member nodes
	for name, runnable := range members {
		// Create a wrapper node that calls the member
		workflow.AddNode(name, fmt.Sprintf("Agent: %s", name), func(ctx context.Context, state S) (S, error) {
			// Execute the member agent
			result, err := runnable.Invoke(ctx, state)
			if err != nil {
				return state, fmt.Errorf("agent %s failed: %w", name, err)
			}
			return result, nil
		})
	}

	// Add conditional routing from supervisor
	workflow.AddConditionalEdge("supervisor", func(ctx context.Context, state S) string {
		next := getNext(state)
		if next == "FINISH" {
			return graph.END
		}
		return next
	})

	// Add edges from members back to supervisor
	for name := range members {
		workflow.AddEdge(name, "supervisor")
	}

	// Set entry point
	workflow.SetEntryPoint("supervisor")

	// Compile and return
	return workflow.Compile()
}