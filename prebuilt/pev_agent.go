package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/log"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// PEVAgentConfig configures the PEV (Plan, Execute, Verify) agent
type PEVAgentConfig struct {
	// Model is the LLM to use for planning and verification
	Model llms.Model

	// Tools are the available tools that can be executed
	Tools []tools.Tool

	// MaxRetries is the maximum number of retry attempts when verification fails
	MaxRetries int

	// SystemMessage is the system message for the planner
	SystemMessage string

	// VerificationPrompt is the prompt for the verifier
	VerificationPrompt string

	// Verbose enables detailed logging
	Verbose bool
}

// VerificationResult represents the result of verification
type VerificationResult struct {
	IsSuccessful bool   `json:"is_successful"`
	Reasoning    string `json:"reasoning"`
}

// CreatePEVAgent creates a new PEV (Plan, Execute, Verify) Agent that implements
// a robust, self-correcting loop for reliable task execution.
//
// The PEV pattern involves:
// 1. Plan: Break down the user request into executable steps
// 2. Execute: Run each step using available tools
// 3. Verify: Check if the execution was successful
// 4. Retry: If verification fails, re-plan and execute again
//
// This pattern is particularly useful for:
// - High-stakes automation scenarios
// - Systems requiring accuracy verification
// - Situations with unreliable external tools
func CreatePEVAgent(config PEVAgentConfig) (*graph.StateRunnable, error) {
	if config.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3 // Default to 3 retries
	}

	// Default system messages
	if config.SystemMessage == "" {
		config.SystemMessage = buildDefaultPlannerPrompt()
	}

	if config.VerificationPrompt == "" {
		config.VerificationPrompt = buildDefaultVerificationPrompt()
	}

	// Create tool executor
	toolExecutor := NewToolExecutor(config.Tools)

	// Create the workflow
	workflow := graph.NewStateGraph()

	// Define state schema
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	agentSchema.RegisterReducer("plan", graph.OverwriteReducer)
	agentSchema.RegisterReducer("current_step", graph.OverwriteReducer)
	agentSchema.RegisterReducer("last_tool_result", graph.OverwriteReducer)
	agentSchema.RegisterReducer("intermediate_steps", graph.AppendReducer)
	agentSchema.RegisterReducer("retries", graph.OverwriteReducer)
	agentSchema.RegisterReducer("verification_result", graph.OverwriteReducer)
	agentSchema.RegisterReducer("final_answer", graph.OverwriteReducer)
	workflow.SetSchema(agentSchema)

	// Add planner node
	workflow.AddNode("planner", "Create or revise execution plan", func(ctx context.Context, state any) (any, error) {
		return plannerNode(ctx, state, config.Model, config.SystemMessage, config.Verbose)
	})

	// Add executor node
	workflow.AddNode("executor", "Execute the current step using tools", func(ctx context.Context, state any) (any, error) {
		return executorNode(ctx, state, toolExecutor, config.Model, config.Verbose)
	})

	// Add verifier node
	workflow.AddNode("verifier", "Verify the execution result", func(ctx context.Context, state any) (any, error) {
		return verifierNode(ctx, state, config.Model, config.VerificationPrompt, config.Verbose)
	})

	// Add synthesizer node
	workflow.AddNode("synthesizer", "Synthesize final answer from all steps", func(ctx context.Context, state any) (any, error) {
		return synthesizerNode(ctx, state, config.Model, config.Verbose)
	})

	// Set entry point
	workflow.SetEntryPoint("planner")

	// Add conditional edges
	workflow.AddConditionalEdge("planner", func(ctx context.Context, state any) string {
		return routeAfterPlanner(state, config.Verbose)
	})

	workflow.AddConditionalEdge("executor", func(ctx context.Context, state any) string {
		return routeAfterExecutor(state, config.Verbose)
	})

	workflow.AddConditionalEdge("verifier", func(ctx context.Context, state any) string {
		return routeAfterVerifier(state, config.MaxRetries, config.Verbose)
	})

	workflow.AddEdge("synthesizer", graph.END)

	return workflow.Compile()
}

// plannerNode creates or revises an execution plan
func plannerNode(ctx context.Context, state any, model llms.Model, systemMessage string, verbose bool) (any, error) {
	mState := state.(map[string]any)

	retries, _ := mState["retries"].(int)
	messages, ok := mState["messages"].([]llms.MessageContent)
	if !ok || len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in state")
	}

	if verbose {
		if retries == 0 {
			log.Info("planning execution steps...")
		} else {
			log.Info("re-planning (attempt %d)...", retries+1)
		}
	}

	// Build planning prompt
	var promptMessages []llms.MessageContent

	if retries == 0 {
		// Initial planning
		promptMessages = []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(systemMessage)},
			},
		}
		promptMessages = append(promptMessages, messages...)
	} else {
		// Re-planning after verification failure
		lastResult, _ := mState["last_tool_result"].(string)
		verificationResult, _ := mState["verification_result"].(VerificationResult)

		replanPrompt := fmt.Sprintf(`The previous execution failed verification. Please create a revised plan.

Original request:
%s

Previous execution result:
%s

Verification feedback:
%s

Create a new plan that addresses the issues identified.`,
			getOriginalRequest(messages), lastResult, verificationResult.Reasoning)

		promptMessages = []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(systemMessage)},
			},
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(replanPrompt)},
			},
		}
	}

	// Generate plan
	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	planText := resp.Choices[0].Content

	// Parse plan into steps
	steps := parsePlanSteps(planText)

	if verbose {
		log.Info("plan created with %d steps", len(steps))
		for i, step := range steps {
			log.Info("  %d. %s", i+1, step)
		}
		log.Info("")
	}

	return map[string]any{
		"plan":         steps,
		"current_step": 0,
	}, nil
}

// executorNode executes the current step
func executorNode(ctx context.Context, state any, toolExecutor *ToolExecutor, model llms.Model, verbose bool) (any, error) {
	mState := state.(map[string]any)

	plan, ok := mState["plan"].([]string)
	if !ok || len(plan) == 0 {
		return nil, fmt.Errorf("no plan found in state")
	}

	currentStep, _ := mState["current_step"].(int)
	if currentStep >= len(plan) {
		return nil, fmt.Errorf("current step index out of bounds")
	}

	stepDescription := plan[currentStep]

	if verbose {
		log.Info("executing step %d/%d: %s", currentStep+1, len(plan), stepDescription)
	}

	// Use LLM to decide which tool to call
	result, err := executeStep(ctx, stepDescription, toolExecutor, model)
	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}

	if verbose {
		log.Info("result: %s\n", truncateString(result, 200))
	}

	return map[string]any{
		"last_tool_result":   result,
		"intermediate_steps": []string{fmt.Sprintf("Step %d: %s -> %s", currentStep+1, stepDescription, truncateString(result, 100))},
	}, nil
}

// verifierNode verifies the execution result
func verifierNode(ctx context.Context, state any, model llms.Model, verificationPrompt string, verbose bool) (any, error) {
	mState := state.(map[string]any)

	lastResult, ok := mState["last_tool_result"].(string)
	if !ok {
		return nil, fmt.Errorf("no tool result found to verify")
	}

	plan, _ := mState["plan"].([]string)
	currentStep, _ := mState["current_step"].(int)
	stepDescription := plan[currentStep]

	if verbose {
		log.Info("verifying execution result...")
	}

	// Build verification prompt
	verifyPrompt := fmt.Sprintf(`Verify if the following execution was successful:

Intended action:
%s

Execution result:
%s

Determine if this result indicates success or failure. Respond with JSON in this exact format:
{
  "is_successful": true or false,
  "reasoning": "your explanation here"
}`,
		stepDescription, lastResult)

	promptMessages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart(verificationPrompt)},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(verifyPrompt)},
		},
	}

	// Generate verification
	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification: %w", err)
	}

	verificationText := resp.Choices[0].Content

	// Parse verification result
	var verificationResult VerificationResult
	if err := parseVerificationResult(verificationText, &verificationResult); err != nil {
		// If parsing fails, assume failure for safety
		verificationResult = VerificationResult{
			IsSuccessful: false,
			Reasoning:    fmt.Sprintf("Failed to parse verification result: %v", err),
		}
	}

	if verbose {
		if verificationResult.IsSuccessful {
			log.Info("verification passed: %s\n", verificationResult.Reasoning)
		} else {
			log.Error("verification failed: %s\n", verificationResult.Reasoning)
		}
	}

	return map[string]any{
		"verification_result": verificationResult,
	}, nil
}

// synthesizerNode creates the final answer from all intermediate steps
func synthesizerNode(ctx context.Context, state any, model llms.Model, verbose bool) (any, error) {
	mState := state.(map[string]any)

	messages, _ := mState["messages"].([]llms.MessageContent)
	intermediateSteps, _ := mState["intermediate_steps"].([]string)

	if verbose {
		log.Info("synthesizing final answer...")
	}

	originalRequest := getOriginalRequest(messages)

	// Build synthesis prompt
	stepsText := strings.Join(intermediateSteps, "\n")
	synthesisPrompt := fmt.Sprintf(`Based on the following execution steps, provide a final answer to the user's request.

User request:
%s

Execution steps:
%s

Provide a clear, concise final answer that directly addresses the user's request.`,
		originalRequest, stepsText)

	promptMessages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant synthesizing results from a multi-step execution.")},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(synthesisPrompt)},
		},
	}

	// Generate final answer
	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate final answer: %w", err)
	}

	finalAnswer := resp.Choices[0].Content

	if verbose {
		log.Info("final answer generated\n")
	}

	// Create AI message
	aiMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(finalAnswer)},
	}

	return map[string]any{
		"messages":     []llms.MessageContent{aiMsg},
		"final_answer": finalAnswer,
	}, nil
}

// Routing functions

func routeAfterPlanner(state any, verbose bool) string {
	mState := state.(map[string]any)
	plan, ok := mState["plan"].([]string)

	if !ok || len(plan) == 0 {
		if verbose {
			log.Warn("no plan created, ending")
		}
		return graph.END
	}

	return "executor"
}

func routeAfterExecutor(state any, verbose bool) string {
	// After execution, always verify
	return "verifier"
}

func routeAfterVerifier(state any, maxRetries int, verbose bool) string {
	mState := state.(map[string]any)

	verificationResult, _ := mState["verification_result"].(VerificationResult)
	currentStep, _ := mState["current_step"].(int)
	plan, _ := mState["plan"].([]string)
	retries, _ := mState["retries"].(int)

	if verificationResult.IsSuccessful {
		// Move to next step
		nextStep := currentStep + 1

		if nextStep >= len(plan) {
			// All steps completed successfully
			if verbose {
				log.Info("all steps completed successfully, synthesizing final answer")
			}
			return "synthesizer"
		}

		// Continue to next step
		mState["current_step"] = nextStep
		return "executor"
	}

	// Verification failed
	if retries >= maxRetries {
		if verbose {
			log.Error("max retries (%d) reached, synthesizing with partial results\n", maxRetries)
		}
		return "synthesizer"
	}

	// Retry with re-planning
	mState["retries"] = retries + 1
	mState["plan"] = nil // Clear plan to force re-planning
	return "planner"
}

// Helper functions

func parsePlanSteps(planText string) []string {
	lines := strings.Split(planText, "\n")
	var steps []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove common step prefixes (1., -, *, etc.)
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		// Remove numbered prefixes like "1.", "2.", etc.
		parts := strings.SplitN(line, ".", 2)
		if len(parts) == 2 {
			if _, err := fmt.Sscanf(parts[0], "%d", new(int)); err == nil {
				line = strings.TrimSpace(parts[1])
			}
		}

		if line != "" {
			steps = append(steps, line)
		}
	}

	return steps
}

func executeStep(ctx context.Context, stepDescription string, toolExecutor *ToolExecutor, model llms.Model) (string, error) {
	if toolExecutor == nil || len(toolExecutor.tools) == 0 {
		return fmt.Sprintf("Error: No tools available to execute %s", stepDescription), nil
	}

	// 1. Build tool definitions string
	var toolsInfo strings.Builder
	for name, tool := range toolExecutor.tools {
		toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", name, tool.Description()))
	}

	// 2. Build prompt
	prompt := fmt.Sprintf(`You are an autonomous agent execution step.
Task: %s

Available Tools:
%s

Select the most appropriate tool to execute this task.
Return ONLY a JSON object with the following format:
{
  "tool": "tool_name",
  "tool_input": "input_string"
}
`, stepDescription, toolsInfo.String())

	// 3. Call LLM
	promptMessages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects the best tool for a task.")},
		},
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart(prompt)},
		},
	}

	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return "", fmt.Errorf("failed to generate tool choice: %w", err)
	}

	choiceText := resp.Choices[0].Content

	// 4. Parse response
	var invocation ToolInvocation
	if err := parseToolChoice(choiceText, &invocation); err != nil {
		return "", fmt.Errorf("failed to parse tool choice: %w (Response: %s)", err, choiceText)
	}

	// 5. Execute tool
	return toolExecutor.Execute(ctx, invocation)
}

func parseToolChoice(text string, invocation *ToolInvocation) error {
	// Try to find JSON in the text
	text = strings.TrimSpace(text)

	// Look for JSON object
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("no JSON object found in text")
	}

	jsonText := text[startIdx : endIdx+1]

	if err := json.Unmarshal([]byte(jsonText), invocation); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

func parseVerificationResult(text string, result *VerificationResult) error {
	// Try to find JSON in the text
	text = strings.TrimSpace(text)

	// Look for JSON object
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx == -1 || endIdx == -1 {
		return fmt.Errorf("no JSON object found in text")
	}

	jsonText := text[startIdx : endIdx+1]

	if err := json.Unmarshal([]byte(jsonText), result); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func buildDefaultPlannerPrompt() string {
	return `You are an expert planner that breaks down user requests into concrete, executable steps.

Your task is to:
1. Analyze the user's request carefully
2. Break it down into clear, sequential steps
3. Each step should be specific and actionable
4. Number each step clearly

Format your plan as a numbered list:
1. First step
2. Second step
3. Third step
...

Be concise but specific. Each step should be something that can be executed using available tools.`
}

func buildDefaultVerificationPrompt() string {
	return `You are a verification specialist that checks if executions were successful.

Your task is to:
1. Analyze the intended action and the actual result
2. Determine if the result indicates success or failure
3. Provide clear reasoning for your determination

Indicators of success:
- Valid data returned
- Positive confirmation messages
- Expected format/structure

Indicators of failure:
- Error messages
- Null/empty results when data expected
- Timeout or connection errors
- Invalid or unexpected format

Always respond with JSON in this exact format:
{
  "is_successful": true or false,
  "reasoning": "explain why you determined success or failure"
}`
}
