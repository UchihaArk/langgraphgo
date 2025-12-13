package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// CalculatorTool is a simple tool for demonstration
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return "Useful for performing basic arithmetic operations. Input should be a string like '2 + 2' or '5 * 10'."
}

func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	// Very simple parser for demo purposes
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid input format, expected 'a op b'")
	}

	a, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "", err
	}
	b, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return "", err
	}

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
		if b == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return "", fmt.Errorf("unknown operator: %s", op)
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

	// Define Tools
	inputTools := []tools.Tool{
		CalculatorTool{},
	}

	// Create ReAct Agent
	agent, err := prebuilt.CreateReactAgent(model, inputTools)
	if err != nil {
		log.Fatal(err)
	}

	// Execute
	query := "What is 25 * 4?"
	fmt.Printf("User: %s\n", query)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, query),
		},
	}

	res, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	// Print Result
	mState := res.(map[string]any)
	messages := mState["messages"].([]llms.MessageContent)
	lastMsg := messages[len(messages)-1]

	if len(lastMsg.Parts) > 0 {
		if textPart, ok := lastMsg.Parts[0].(llms.TextContent); ok {
			fmt.Printf("Agent: %s\n", textPart.Text)
		}
	}
}
