package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// WeatherTool simulates a weather API with occasional failures
type WeatherTool struct {
	FailureRate float64 // Probability of failure (0.0 to 1.0)
}

func (t WeatherTool) Name() string {
	return "get_weather"
}

func (t WeatherTool) Description() string {
	return "Get the current weather for a location. Input should be a city name like 'New York' or 'London'."
}

func (t WeatherTool) Call(ctx context.Context, input string) (string, error) {
	// Simulate network unreliability
	if rand.Float64() < t.FailureRate {
		return "Error: Connection timeout - weather service unavailable", nil
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "Error: Please provide a city name", nil
	}

	// Simulate weather data
	temps := []int{15, 18, 22, 25, 28, 12, 8}
	conditions := []string{"Sunny", "Cloudy", "Rainy", "Partly Cloudy"}

	temp := temps[rand.Intn(len(temps))]
	condition := conditions[rand.Intn(len(conditions))]

	return fmt.Sprintf("Weather in %s: %d°C, %s", input, temp, condition), nil
}

// CalculatorTool simulates a calculator with validation
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return "Perform basic arithmetic operations. Input should be in format: 'a operator b', e.g., '5 + 3' or '10 * 2'."
}

func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return "Error: Invalid input format. Expected 'a operator b'", nil
	}

	a, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Sprintf("Error: Invalid number '%s'", parts[0]), nil
	}

	b, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return fmt.Sprintf("Error: Invalid number '%s'", parts[2]), nil
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
			return "Error: Division by zero", nil
		}
		result = a / b
	default:
		return fmt.Sprintf("Error: Unknown operator '%s'", op), nil
	}

	return fmt.Sprintf("%.2f", result), nil
}

// DatabaseTool simulates a database query with occasional failures
type DatabaseTool struct {
	FailureRate float64
}

func (t DatabaseTool) Name() string {
	return "query_database"
}

func (t DatabaseTool) Description() string {
	return "Query the user database. Input should describe what user information to retrieve."
}

func (t DatabaseTool) Call(ctx context.Context, input string) (string, error) {
	// Simulate database unreliability
	if rand.Float64() < t.FailureRate {
		return "Error: Database connection failed", nil
	}

	// Simulate query processing delay
	time.Sleep(100 * time.Millisecond)

	// Return simulated data
	return fmt.Sprintf("Database query result: Found 42 users matching '%s'", input), nil
}

func main() {
	// Seed random for consistent demo behavior
	rand.Seed(time.Now().UnixNano())

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create LLM
	model, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	fmt.Println("=== PEV Agent Examples ===")
	fmt.Println("PEV (Plan, Execute, Verify) is a robust, self-correcting agent pattern")
	fmt.Println("that verifies each action and can recover from failures.")

	// Example 1: Simple calculation with reliable tool
	fmt.Println("--- Example 1: Simple Calculation (Reliable Tool) ---")
	runExample1(model)

	fmt.Println("\n" + strings.Repeat("=", 80) + "\n")

	// Example 2: Weather query with unreliable tool
	fmt.Println("--- Example 2: Weather Query (Unreliable Tool) ---")
	runExample2(model)

	fmt.Println("\n" + strings.Repeat("=", 80) + "\n")

	// Example 3: Multi-step task with verification
	fmt.Println("--- Example 3: Multi-Step Task with Verification ---")
	runExample3(model)
}

func runExample1(model llms.Model) {
	fmt.Println("This example demonstrates basic PEV operation with a reliable tool.")

	config := prebuilt.PEVAgentConfig{
		Model:      model,
		Tools:      []tools.Tool{CalculatorTool{}},
		MaxRetries: 3,
		Verbose:    true,
	}

	agent, err := prebuilt.CreatePEVAgent(config)
	if err != nil {
		log.Fatalf("Failed to create PEV agent: %v", err)
	}

	query := "Calculate the result of 15 multiplied by 8"
	fmt.Printf("Query: %s\n\n", query)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(query)},
			},
		},
		"retries": 0,
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatalf("Failed to invoke agent: %v", err)
	}

	printResult(result)
}

func runExample2(model llms.Model) {
	fmt.Println("This example demonstrates PEV's self-correction with an unreliable weather API.")
	fmt.Println("The tool has a 40% failure rate, but PEV will retry until successful.")

	config := prebuilt.PEVAgentConfig{
		Model: model,
		Tools: []tools.Tool{
			WeatherTool{FailureRate: 0.4}, // 40% chance of failure
		},
		MaxRetries: 3,
		Verbose:    true,
	}

	agent, err := prebuilt.CreatePEVAgent(config)
	if err != nil {
		log.Fatalf("Failed to create PEV agent: %v", err)
	}

	query := "What's the weather like in Tokyo?"
	fmt.Printf("Query: %s\n\n", query)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(query)},
			},
		},
		"retries": 0,
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatalf("Failed to invoke agent: %v", err)
	}

	printResult(result)
}

func runExample3(model llms.Model) {
	fmt.Println("This example demonstrates PEV with multiple steps and different tools.")
	fmt.Println("Each step is verified before proceeding to the next.")

	config := prebuilt.PEVAgentConfig{
		Model: model,
		Tools: []tools.Tool{
			CalculatorTool{},
			WeatherTool{FailureRate: 0.2},  // 20% failure rate
			DatabaseTool{FailureRate: 0.3}, // 30% failure rate
		},
		MaxRetries: 3,
		Verbose:    true,
	}

	agent, err := prebuilt.CreatePEVAgent(config)
	if err != nil {
		log.Fatalf("Failed to create PEV agent: %v", err)
	}

	query := "First, calculate 25 times 4. Then, check the weather in Paris."
	fmt.Printf("Query: %s\n\n", query)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(query)},
			},
		},
		"retries": 0,
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatalf("Failed to invoke agent: %v", err)
	}

	printResult(result)
}

func printResult(result any) {
	finalState := result.(map[string]any)

	// Print final answer
	if finalAnswer, ok := finalState["final_answer"].(string); ok {
		fmt.Println("\n=== Final Answer ===")
		fmt.Println(finalAnswer)
		fmt.Println("===================")
	}

	// Print retry count
	retries, _ := finalState["retries"].(int)
	if retries > 0 {
		fmt.Printf("\n⚠️  Total retries: %d\n", retries)
	}

	// Print intermediate steps
	if steps, ok := finalState["intermediate_steps"].([]string); ok && len(steps) > 0 {
		fmt.Println("\n=== Execution Steps ===")
		for _, step := range steps {
			fmt.Println(step)
		}
	}
}
