package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// GPTResearcher is the main research orchestrator
type GPTResearcher struct {
	Config         *Config
	PlannerAgent   *PlannerAgent
	ExecutionAgent *ExecutionAgent
	PublisherAgent *PublisherAgent
	Tools          *ToolRegistry
	Graph          *graph.StateRunnable
}

// NewGPTResearcher creates a new GPT Researcher instance
func NewGPTResearcher(config *Config) (*GPTResearcher, error) {
	// Create LLM models
	plannerModel, err := openai.New(
		openai.WithModel(config.Model),
		openai.WithToken(config.OpenAIAPIKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner model: %w", err)
	}

	executionModel, err := openai.New(
		openai.WithModel(config.Model),
		openai.WithToken(config.OpenAIAPIKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution model: %w", err)
	}

	publisherModel, err := openai.New(
		openai.WithModel(config.ReportModel),
		openai.WithToken(config.OpenAIAPIKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher model: %w", err)
	}

	summaryModel, err := openai.New(
		openai.WithModel(config.SummaryModel),
		openai.WithToken(config.OpenAIAPIKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create summary model: %w", err)
	}

	// Create tools
	tools := NewToolRegistry(config, summaryModel)

	// Create agents
	plannerAgent := NewPlannerAgent(plannerModel, config)
	executionAgent := NewExecutionAgent(executionModel, config, tools)
	publisherAgent := NewPublisherAgent(publisherModel, config)

	researcher := &GPTResearcher{
		Config:         config,
		PlannerAgent:   plannerAgent,
		ExecutionAgent: executionAgent,
		PublisherAgent: publisherAgent,
		Tools:          tools,
	}

	// Build the workflow graph
	if err := researcher.buildGraph(); err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	return researcher, nil
}

// buildGraph constructs the research workflow using langgraphgo
func (r *GPTResearcher) buildGraph() error {
	// Create workflow
	workflow := graph.NewStateGraph()

	// Define schema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("questions", graph.AppendReducer)
	schema.RegisterReducer("search_results", graph.AppendReducer)
	schema.RegisterReducer("summaries", graph.AppendReducer)
	schema.RegisterReducer("sources", graph.AppendReducer)
	workflow.SetSchema(schema)

	// Add nodes
	workflow.AddNode("planner", "Generate research questions", r.plannerNode)
	workflow.AddNode("executor", "Execute research and gather information", r.executorNode)
	workflow.AddNode("publisher", "Generate final research report", r.publisherNode)

	// Add edges
	workflow.SetEntryPoint("planner")
	workflow.AddEdge("planner", "executor")
	workflow.AddEdge("executor", "publisher")
	workflow.AddEdge("publisher", graph.END)

	// Compile graph
	compiled, err := workflow.Compile()
	if err != nil {
		return err
	}

	r.Graph = compiled
	return nil
}

// plannerNode is the graph node for the planner agent
func (r *GPTResearcher) plannerNode(ctx context.Context, stateInterface any) (any, error) {
	state := r.interfaceToState(stateInterface)

	if err := r.PlannerAgent.GenerateQuestions(ctx, state); err != nil {
		return nil, err
	}

	return r.stateToInterface(state), nil
}

// executorNode is the graph node for the execution agent
func (r *GPTResearcher) executorNode(ctx context.Context, stateInterface any) (any, error) {
	state := r.interfaceToState(stateInterface)

	if err := r.ExecutionAgent.ExecuteAll(ctx, state); err != nil {
		return nil, err
	}

	return r.stateToInterface(state), nil
}

// publisherNode is the graph node for the publisher agent
func (r *GPTResearcher) publisherNode(ctx context.Context, stateInterface any) (any, error) {
	state := r.interfaceToState(stateInterface)

	if err := r.PublisherAgent.GenerateReport(ctx, state); err != nil {
		return nil, err
	}

	return r.stateToInterface(state), nil
}

// ConductResearch executes the full research workflow
func (r *GPTResearcher) ConductResearch(ctx context.Context, query string) (*ResearchState, error) {
	if r.Config.Verbose {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("GPT RESEARCHER")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("\n📋 Research Query: %s\n", query)
		fmt.Println()
	}

	// Create initial state
	initialState := NewResearchState(query)

	// Convert to map for graph execution
	stateMap := r.stateToInterface(initialState)

	// Execute graph
	result, err := r.Graph.Invoke(ctx, stateMap)
	if err != nil {
		return nil, fmt.Errorf("research workflow failed: %w", err)
	}

	// Convert back to ResearchState
	finalState := r.interfaceToState(result)

	if r.Config.Verbose {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("RESEARCH COMPLETE")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("\nStatistics:\n")
		fmt.Printf("- Research Questions: %d\n", len(finalState.Questions))
		fmt.Printf("- Sources Consulted: %d\n", len(finalState.Sources))
		fmt.Printf("- Summaries Generated: %d\n", len(finalState.Summaries))
		fmt.Printf("- Report Length: %d characters\n", len(finalState.FinalReport))
		fmt.Printf("- Duration: %.1f minutes\n\n", finalState.EndTime.Sub(finalState.StartTime).Minutes())
	}

	return finalState, nil
}

// WriteReport generates and returns the final report
func (r *GPTResearcher) WriteReport(ctx context.Context, state *ResearchState) (string, error) {
	if state.FinalReport != "" {
		return state.FinalReport, nil
	}

	// If report wasn't generated yet, generate it now
	if err := r.PublisherAgent.GenerateReport(ctx, state); err != nil {
		return "", err
	}

	return state.FinalReport, nil
}

// stateToInterface converts ResearchState to map[string]any for graph
func (r *GPTResearcher) stateToInterface(state *ResearchState) map[string]any {
	return map[string]any{
		"query":              state.Query,
		"research_goal":      state.ResearchGoal,
		"questions":          state.Questions,
		"planning_complete":  state.PlanningComplete,
		"search_results":     state.SearchResults,
		"summaries":          state.Summaries,
		"execution_complete": state.ExecutionComplete,
		"final_report":       state.FinalReport,
		"report_complete":    state.ReportComplete,
		"sources":            state.Sources,
		"total_sources":      state.TotalSources,
		"iteration":          state.Iteration,
		"start_time":         state.StartTime,
		"end_time":           state.EndTime,
		"messages":           state.Messages,
	}
}

// interfaceToState converts map[string]any back to ResearchState
func (r *GPTResearcher) interfaceToState(stateInterface any) *ResearchState {
	stateMap, ok := stateInterface.(map[string]any)
	if !ok {
		return NewResearchState("")
	}

	state := &ResearchState{}

	if v, ok := stateMap["query"].(string); ok {
		state.Query = v
	}
	if v, ok := stateMap["research_goal"].(string); ok {
		state.ResearchGoal = v
	}
	if v, ok := stateMap["questions"].([]string); ok {
		state.Questions = v
	} else if v, ok := stateMap["questions"].([]any); ok {
		for _, q := range v {
			if str, ok := q.(string); ok {
				state.Questions = append(state.Questions, str)
			}
		}
	}
	if v, ok := stateMap["planning_complete"].(bool); ok {
		state.PlanningComplete = v
	}
	if v, ok := stateMap["execution_complete"].(bool); ok {
		state.ExecutionComplete = v
	}
	if v, ok := stateMap["report_complete"].(bool); ok {
		state.ReportComplete = v
	}
	if v, ok := stateMap["final_report"].(string); ok {
		state.FinalReport = v
	}
	if v, ok := stateMap["total_sources"].(int); ok {
		state.TotalSources = v
	}
	if v, ok := stateMap["iteration"].(int); ok {
		state.Iteration = v
	}

	// Handle complex types
	if v, ok := stateMap["search_results"].([]SearchResult); ok {
		state.SearchResults = v
	}
	if v, ok := stateMap["summaries"].([]SourceSummary); ok {
		state.Summaries = v
	}
	if v, ok := stateMap["sources"].([]Source); ok {
		state.Sources = v
	}
	if v, ok := stateMap["messages"].([]llms.MessageContent); ok {
		state.Messages = v
	}
	if v, ok := stateMap["start_time"].(time.Time); ok {
		state.StartTime = v
	}
	if v, ok := stateMap["end_time"].(time.Time); ok {
		state.EndTime = v
	}

	return state
}
