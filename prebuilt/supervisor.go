package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

// CreateSupervisor creates a supervisor graph that orchestrates multiple agents
func CreateSupervisor(model llms.Model, members map[string]*graph.StateRunnable) (*graph.StateRunnable, error) {
	workflow := graph.NewStateGraph()

	// Define state schema
	// We use MapSchema with AppendReducer for messages
	schema := graph.NewMapSchema()
	schema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(schema)

	// Get member names
	var memberNames []string
	for name := range members {
		memberNames = append(memberNames, name)
	}

	// Define supervisor node
	workflow.AddNode("supervisor", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState, ok := state.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid state type")
		}

		messages, ok := mState["messages"].([]llms.MessageContent)
		if !ok {
			return nil, fmt.Errorf("messages key not found or invalid type")
		}

		// Define routing function
		options := append(memberNames, "FINISH")
		routeTool := llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "route",
				Description: "Select the next role.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"next": map[string]interface{}{
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
			llms.WithToolChoice("auto"), // Let model decide, but prompt strongly encourages it
		)
		if err != nil {
			return nil, err
		}

		choice := resp.Choices[0]
		if len(choice.ToolCalls) == 0 {
			// If no tool call, assume FINISH or error?
			// With ToolChoice("route"), it should call it.\
			return nil, fmt.Errorf("supervisor did not select a next step")
		}

		// Parse selection
		tc := choice.ToolCalls[0]
		var args struct {
			Next string `json:"next"`
		}
		if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
			return nil, fmt.Errorf("failed to parse route arguments: %w", err)
		}

		// We return the decision in a special key "next"
		// We don't append to messages here, or maybe we do?
		// Usually supervisor output is not part of conversation history unless we want to track decisions.
		// For now, let's just return "next" in state for conditional edge.
		return map[string]interface{}{
			"next": args.Next,
		}, nil
	})

	// Add member nodes
	for name, agent := range members {
		// We need to wrap the agent runnable to adapt state if needed
		// But assuming they share the same state schema (messages)
		// We can just call agent.Invoke
		agentName := name
		agentRunnable := agent

		workflow.AddNode(agentName, func(ctx context.Context, state interface{}) (interface{}, error) {
			// Invoke agent
			// We pass the full state
			res, err := agentRunnable.Invoke(ctx, state)
			if err != nil {
				return nil, err
			}

			// Result should be a map with messages
			// We return it to be merged
			return res, nil
		})
	}

	// Define edges
	workflow.SetEntryPoint("supervisor")

	// Conditional edge from supervisor
	workflow.AddConditionalEdge("supervisor", func(ctx context.Context, state interface{}) string {
		mState := state.(map[string]interface{})
		next, ok := mState["next"].(string)
		if !ok {
			return graph.END
		}
		if next == "FINISH" {
			return graph.END
		}
		return next
	})

	// Edges from members back to supervisor
	for _, name := range memberNames {
		workflow.AddEdge(name, "supervisor")
	}

	return workflow.Compile()
}
