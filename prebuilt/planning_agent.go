package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/log"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// WorkflowPlan represents the parsed workflow plan from LLM
type WorkflowPlan struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

// WorkflowNode represents a node in the workflow plan
type WorkflowNode struct {
	Name string `json:"name"`
	Type string `json:"type"` // "start", "process", "end", "conditional"
}

// WorkflowEdge represents an edge in the workflow plan
type WorkflowEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"` // For conditional edges
}

// CreatePlanningAgent creates an agent that first plans the workflow using LLM,
// then executes according to the generated plan
func CreatePlanningAgent(model llms.Model, nodes []*graph.Node, inputTools []tools.Tool, opts ...CreateAgentOption) (*graph.StateRunnable, error) {
	options := &CreateAgentOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Create a map of node names to nodes for easy lookup
	nodeMap := make(map[string]*graph.Node)
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}

	// Define the workflow
	workflow := graph.NewStateGraph()

	// Define the state schema
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	agentSchema.RegisterReducer("workflow_plan", graph.OverwriteReducer)
	workflow.SetSchema(agentSchema)

	// Add planning node - this is where LLM generates the workflow
	workflow.AddNode("planner", "Generates workflow plan based on user request", func(ctx context.Context, state any) (any, error) {
		mState, ok := state.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid state type: %T", state)
		}

		messages, ok := mState["messages"].([]llms.MessageContent)
		if !ok || len(messages) == 0 {
			return nil, fmt.Errorf("no messages found in state")
		}

		// Build the planning prompt
		nodeDescriptions := buildNodeDescriptions(nodes)
		planningPrompt := buildPlanningPrompt(nodeDescriptions)

		// Prepare messages for LLM
		planningMessages := []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(planningPrompt)},
			},
		}
		planningMessages = append(planningMessages, messages...)

		if options.Verbose {
			log.Info("planning workflow...")
		}

		// Call LLM to generate the plan
		resp, err := model.GenerateContent(ctx, planningMessages)
		if err != nil {
			return nil, fmt.Errorf("failed to generate plan: %w", err)
		}

		planText := resp.Choices[0].Content
		if options.Verbose {
			log.Info("generated plan:\n%s\n", planText)
		}

		// Parse the workflow plan
		workflowPlan, err := parseWorkflowPlan(planText)
		if err != nil {
			return nil, fmt.Errorf("failed to parse workflow plan: %w", err)
		}

		// Store the plan in state
		aiMsg := llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextPart(fmt.Sprintf("Workflow plan created with %d nodes and %d edges", len(workflowPlan.Nodes), len(workflowPlan.Edges)))},
		}

		return map[string]any{
			"messages":      []llms.MessageContent{aiMsg},
			"workflow_plan": workflowPlan,
		}, nil
	})

	// Add executor node - this builds and executes the planned workflow
	workflow.AddNode("executor", "Executes the planned workflow", func(ctx context.Context, state any) (any, error) {
		mState, ok := state.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid state type: %T", state)
		}

		workflowPlan, ok := mState["workflow_plan"].(*WorkflowPlan)
		if !ok {
			return nil, fmt.Errorf("workflow_plan not found in state")
		}

		if options.Verbose {
			log.Info("executing planned workflow...")
		}

		// Build the dynamic workflow
		dynamicWorkflow := graph.NewStateGraph()
		dynamicSchema := graph.NewMapSchema()
		dynamicSchema.RegisterReducer("messages", graph.AppendReducer)
		dynamicWorkflow.SetSchema(dynamicSchema)

		// Add nodes from the plan
		for _, planNode := range workflowPlan.Nodes {
			if planNode.Name == "START" || planNode.Name == "END" {
				continue // Skip special nodes
			}

			actualNode, exists := nodeMap[planNode.Name]
			if !exists {
				return nil, fmt.Errorf("node %s not found in available nodes", planNode.Name)
			}

			// Add the node with its original function
			dynamicWorkflow.AddNode(actualNode.Name, actualNode.Description, actualNode.Function)

			if options.Verbose {
				log.Info("added node: %s", actualNode.Name)
			}
		}

		// Add edges from the plan
		var entryPoint string
		endNodes := make(map[string]bool) // Track nodes that should end

		for _, edge := range workflowPlan.Edges {
			if edge.From == "START" {
				entryPoint = edge.To
				continue
			}
			if edge.To == "END" {
				endNodes[edge.From] = true
				continue // Will be handled after all edges are added
			}

			if edge.Condition != "" {
				// This is a conditional edge
				// For now, we'll add a simple conditional edge
				// In a real implementation, you might want to parse the condition
				dynamicWorkflow.AddConditionalEdge(edge.From, func(ctx context.Context, state any) string {
					// Simple condition evaluation
					// You can enhance this to evaluate the actual condition
					return edge.To
				})
			} else {
				dynamicWorkflow.AddEdge(edge.From, edge.To)
			}

			if options.Verbose {
				log.Info("  added edge: %s -> %s", edge.From, edge.To)
			}
		}

		// Add edges to END for terminal nodes
		for nodeName := range endNodes {
			dynamicWorkflow.AddEdge(nodeName, graph.END)
			if options.Verbose {
				log.Info("  added edge: %s -> END", nodeName)
			}
		}

		if entryPoint == "" {
			return nil, fmt.Errorf("no entry point found in workflow plan")
		}

		dynamicWorkflow.SetEntryPoint(entryPoint)

		// Compile and execute the dynamic workflow
		runnable, err := dynamicWorkflow.Compile()
		if err != nil {
			return nil, fmt.Errorf("failed to compile dynamic workflow: %w", err)
		}

		// Execute the dynamic workflow with current state
		result, err := runnable.Invoke(ctx, mState)
		if err != nil {
			return nil, fmt.Errorf("failed to execute dynamic workflow: %w", err)
		}

		if options.Verbose {
			log.Info("workflow execution completed")
		}

		return result, nil
	})

	// Define edges
	workflow.SetEntryPoint("planner")
	workflow.AddEdge("planner", "executor")
	workflow.AddEdge("executor", graph.END)

	return workflow.Compile()
}

// buildNodeDescriptions creates a formatted string describing all available nodes
func buildNodeDescriptions(nodes []*graph.Node) string {
	var sb strings.Builder
	sb.WriteString("Available nodes:\n")
	for i, node := range nodes {
		sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, node.Name, node.Description))
	}
	return sb.String()
}

// buildPlanningPrompt creates the prompt for the LLM to generate a workflow plan
func buildPlanningPrompt(nodeDescriptions string) string {
	return fmt.Sprintf(`You are a workflow planning assistant. Based on the user's request, create a workflow plan using the available nodes.

%s

Generate a workflow plan in the following JSON format:
{
  "nodes": [
    {"name": "node_name", "type": "process"}
  ],
  "edges": [
    {"from": "START", "to": "first_node"},
    {"from": "first_node", "to": "second_node"},
    {"from": "last_node", "to": "END"}
  ]
}

Rules:
1. The workflow must start with an edge from "START"
2. The workflow must end with an edge to "END"
3. Only use nodes from the available nodes list
4. Each node should appear in the nodes array
5. Create a logical flow based on the user's request
6. Return ONLY the JSON object, no additional text

Example:
{
  "nodes": [
    {"name": "research", "type": "process"},
    {"name": "analyze", "type": "process"}
  ],
  "edges": [
    {"from": "START", "to": "research"},
    {"from": "research", "to": "analyze"},
    {"from": "analyze", "to": "END"}
  ]
}`, nodeDescriptions)
}

// parseWorkflowPlan parses the LLM response to extract the workflow plan
func parseWorkflowPlan(planText string) (*WorkflowPlan, error) {
	// Extract JSON from the response (handle markdown code blocks)
	jsonText := extractJSON(planText)

	var plan WorkflowPlan
	if err := json.Unmarshal([]byte(jsonText), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate the plan
	if len(plan.Nodes) == 0 {
		return nil, fmt.Errorf("workflow plan has no nodes")
	}
	if len(plan.Edges) == 0 {
		return nil, fmt.Errorf("workflow plan has no edges")
	}

	return &plan, nil
}

// extractJSON extracts JSON from a text that might contain markdown code blocks
func extractJSON(text string) string {
	// Try to find JSON in markdown code block
	codeBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*({.*?})\\s*```")
	matches := codeBlockRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find JSON object directly
	jsonRegex := regexp.MustCompile("(?s){.*}")
	matches = jsonRegex.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}

	return text
}
