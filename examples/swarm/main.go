package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// AgentState defines the state for the swarm
type AgentState struct {
	Messages []llms.MessageContent
	Next     string
}

// HandoffTool defines the tool for handing off control
var HandoffTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "handoff",
		Description: "Hand off control to another agent.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"to": map[string]any{
					"type": "string",
					"enum": []string{"Researcher", "Writer"},
				},
			},
			"required": []string{"to"},
		},
	},
}

func main() {
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	opts := []openai.Option{}
	if base := os.Getenv("OPENAI_API_BASE"); base != "" {
		opts = append(opts, openai.WithBaseURL(base))
	}
	if modelName := os.Getenv("OPENAI_MODEL"); modelName != "" {
		opts = append(opts, openai.WithModel(modelName))
	}

	model, err := openai.New(opts...)
	if err != nil {
		log.Fatal(err)
	}

	// Define the graph
	workflow := graph.NewStateGraph()

	// Define Schema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(schema)

	// Researcher Agent Node
	workflow.AddNode("Researcher", "Researcher", func(ctx context.Context, state any) (any, error) {
		mState := state.(map[string]any)
		messages := mState["messages"].([]llms.MessageContent)

		systemPrompt := "You are a researcher. You can search for information. If you need to write a report, hand off to the Writer. If you are done, just say 'I am done'."
		inputMessages := append([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		}, messages...)

		resp, err := model.GenerateContent(ctx, inputMessages, llms.WithTools([]llms.Tool{HandoffTool}))
		if err != nil {
			return nil, err
		}

		choice := resp.Choices[0]

		// Check for handoff
		if len(choice.ToolCalls) > 0 {
			tc := choice.ToolCalls[0]
			if tc.FunctionCall.Name == "handoff" {
				var args struct {
					To string `json:"to"`
				}
				json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

				// Return tool call message AND update 'next'
				// Note: We should probably append the tool call to history so the next agent sees it?
				// Or maybe just the handoff event.
				// For Swarm, usually we transfer control.

				return map[string]any{
					"messages": []llms.MessageContent{
						{
							Role:  llms.ChatMessageTypeAI,
							Parts: []llms.ContentPart{tc},
						},
						// We also need to add the ToolMessage to complete the turn?
						// Or the next agent will see the tool call and act?
						// Let's add a ToolMessage saying "Handoff to X"
						{
							Role: llms.ChatMessageTypeTool,
							Parts: []llms.ContentPart{
								llms.ToolCallResponse{
									ToolCallID: tc.ID,
									Name:       "handoff",
									Content:    fmt.Sprintf("Handing off to %s", args.To),
								},
							},
						},
					},
					"next": args.To,
				}, nil
			}
		}

		// Normal response
		return map[string]any{
			"messages": []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart(choice.Content)},
				},
			},
			"next": "END", // Or stay?
		}, nil
	})

	// Writer Agent Node
	workflow.AddNode("Writer", "Writer", func(ctx context.Context, state any) (any, error) {
		mState := state.(map[string]any)
		messages := mState["messages"].([]llms.MessageContent)

		systemPrompt := "You are a writer. You write reports based on information. If you need more information, hand off to the Researcher. If you are done, just say 'I am done'."
		inputMessages := append([]llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		}, messages...)

		resp, err := model.GenerateContent(ctx, inputMessages, llms.WithTools([]llms.Tool{HandoffTool}))
		if err != nil {
			return nil, err
		}

		choice := resp.Choices[0]

		if len(choice.ToolCalls) > 0 {
			tc := choice.ToolCalls[0]
			if tc.FunctionCall.Name == "handoff" {
				var args struct {
					To string `json:"to"`
				}
				json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

				return map[string]any{
					"messages": []llms.MessageContent{
						{
							Role:  llms.ChatMessageTypeAI,
							Parts: []llms.ContentPart{tc},
						},
						{
							Role: llms.ChatMessageTypeTool,
							Parts: []llms.ContentPart{
								llms.ToolCallResponse{
									ToolCallID: tc.ID,
									Name:       "handoff",
									Content:    fmt.Sprintf("Handing off to %s", args.To),
								},
							},
						},
					},
					"next": args.To,
				}, nil
			}
		}

		return map[string]any{
			"messages": []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart(choice.Content)},
				},
			},
			"next": "END",
		}, nil
	})

	// Define Edge Logic
	router := func(ctx context.Context, state any) string {
		mState := state.(map[string]any)
		next, ok := mState["next"].(string)
		if !ok || next == "" || next == "END" {
			return graph.END
		}
		return next
	}

	workflow.AddConditionalEdge("Researcher", router)
	workflow.AddConditionalEdge("Writer", router)

	workflow.SetEntryPoint("Researcher")

	app, err := workflow.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Run
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "I need a report on the latest AI trends."),
		},
	}

	res, err := app.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)
	for _, msg := range messages {
		role := msg.Role
		fmt.Printf("%s: ", role)
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case llms.TextContent:
				fmt.Print(p.Text)
			case llms.ToolCall:
				fmt.Printf("[Tool Call: %s]", p.FunctionCall.Name)
			case llms.ToolCallResponse:
				fmt.Printf("[Tool Response: %s]", p.Content)
			default:
				fmt.Printf("[%T]", p)
			}
		}
		fmt.Println()
	}
}
