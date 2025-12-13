package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// CalculatorTool (same as in react_agent example)
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return "Useful for performing basic arithmetic operations. Input should be a string like '2 + 2' or '5 * 10'."
}

func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid input format")
	}
	a, _ := strconv.ParseFloat(parts[0], 64)
	b, _ := strconv.ParseFloat(parts[2], 64)
	op := parts[1]
	var result float64
	switch op {
	case "+":
		result = a + b
	case "-":
		result = a - b
	case "*":
		result = a * b
	case "/":
		result = a / b
	}
	return fmt.Sprintf("%f", result), nil
}

func main() {
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	// Initialize LLM
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

	// 1. Create Math Agent
	mathAgent, err := prebuilt.CreateReactAgent(model, []tools.Tool{CalculatorTool{}})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Create General Agent (just a simple runnable or react agent with no tools)
	generalAgent, err := prebuilt.CreateReactAgent(model, []tools.Tool{})
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create Supervisor
	members := map[string]*graph.StateRunnable{
		"MathExpert":       mathAgent,
		"GeneralAssistant": generalAgent,
	}

	supervisor, err := prebuilt.CreateSupervisor(model, members)
	if err != nil {
		log.Fatal(err)
	}

	// Execute
	query := "Calculate 10 * 5 and then tell me a joke about the result."
	fmt.Printf("User: %s\n", query)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, query),
		},
	}

	// Note: Supervisor loop might run indefinitely if not careful or if LLM doesn't say FINISH.
	// We rely on the supervisor prompt to eventually FINISH.
	// For safety, we might want to add a recursion limit config if available,
	// but here we just run it.

	// We can use a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := supervisor.Invoke(ctx, initialState)
	if err != nil {
		log.Fatal(err)
	}

	// Print Result
	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)

	fmt.Println("\n=== Conversation History ===")
	for _, msg := range messages {
		role := msg.Role
		var content string
		if len(msg.Parts) > 0 {
			if textPart, ok := msg.Parts[0].(llms.TextContent); ok {
				content = textPart.Text
			} else if _, ok := msg.Parts[0].(llms.ToolCall); ok {
				content = "[Tool Call]"
			} else if _, ok := msg.Parts[0].(llms.ToolCallResponse); ok {
				content = "[Tool Response]"
			}
		}
		fmt.Printf("%s: %s\n", role, content)
	}
}
