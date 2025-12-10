package graph

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// StateGraph represents a state-based graph similar to Python's LangGraph StateGraph
type StateGraph struct {
	// nodes is a map of node names to their corresponding Node objects
	nodes map[string]Node

	// edges is a slice of Edge objects representing the connections between nodes
	edges []Edge

	// conditionalEdges contains a map between "From" node, while "To" node is derived based on the condition
	conditionalEdges map[string]func(ctx context.Context, state interface{}) string

	// entryPoint is the name of the entry point node in the graph
	entryPoint string

	// retryPolicy defines retry behavior for failed nodes
	retryPolicy *RetryPolicy

	// stateMerger is an optional function to merge states from parallel execution
	stateMerger StateMerger

	// Schema defines the state structure and update logic
	Schema StateSchema
}

// RetryPolicy defines how to handle node failures
type RetryPolicy struct {
	MaxRetries      int
	BackoffStrategy BackoffStrategy
	RetryableErrors []string
}

// BackoffStrategy defines different backoff strategies
type BackoffStrategy int

const (
	FixedBackoff BackoffStrategy = iota
	ExponentialBackoff
	LinearBackoff
)

// NewStateGraph creates a new instance of StateGraph without a schema.
// For chat-based agents that need message handling, use NewMessageGraph() instead.
func NewStateGraph() *StateGraph {
	return &StateGraph{
		nodes:            make(map[string]Node),
		conditionalEdges: make(map[string]func(ctx context.Context, state interface{}) string),
	}
}

// AddNode adds a new node to the state graph with the given name, description and function
func (g *StateGraph) AddNode(name string, description string, fn func(ctx context.Context, state interface{}) (interface{}, error)) {
	g.nodes[name] = Node{
		Name:        name,
		Description: description,
		Function:    fn,
	}
}

// AddEdge adds a new edge to the state graph between the "from" and "to" nodes
func (g *StateGraph) AddEdge(from, to string) {
	g.edges = append(g.edges, Edge{
		From: from,
		To:   to,
	})
}

// AddConditionalEdge adds a conditional edge where the target node is determined at runtime
func (g *StateGraph) AddConditionalEdge(from string, condition func(ctx context.Context, state interface{}) string) {
	g.conditionalEdges[from] = condition
}

// SetEntryPoint sets the entry point node name for the state graph
func (g *StateGraph) SetEntryPoint(name string) {
	g.entryPoint = name
}

// SetRetryPolicy sets the retry policy for the graph
func (g *StateGraph) SetRetryPolicy(policy *RetryPolicy) {
	g.retryPolicy = policy
}

// SetStateMerger sets the state merger function for the state graph
func (g *StateGraph) SetStateMerger(merger StateMerger) {
	g.stateMerger = merger
}

// SetSchema sets the state schema for the graph
func (g *StateGraph) SetSchema(schema StateSchema) {
	g.Schema = schema
}

// StateRunnable represents a compiled state graph that can be invoked
type StateRunnable struct {
	graph  *StateGraph
	tracer *Tracer
}

// Compile compiles the state graph and returns a StateRunnable instance
func (g *StateGraph) Compile() (*StateRunnable, error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	return &StateRunnable{
		graph:  g,
		tracer: nil, // Initialize with no tracer
	}, nil
}

// SetTracer sets a tracer for observability
func (r *StateRunnable) SetTracer(tracer *Tracer) {
	r.tracer = tracer
}

// WithTracer returns a new StateRunnable with the given tracer
func (r *StateRunnable) WithTracer(tracer *Tracer) *StateRunnable {
	return &StateRunnable{
		graph:  r.graph,
		tracer: tracer,
	}
}

// Invoke executes the compiled state graph with the given input state
func (r *StateRunnable) Invoke(ctx context.Context, initialState interface{}) (interface{}, error) {
	return r.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig executes the compiled state graph with the given input state and config
func (r *StateRunnable) InvokeWithConfig(ctx context.Context, initialState interface{}, config *Config) (interface{}, error) {
	state := initialState
	currentNodes := []string{r.graph.entryPoint}

	// Handle ResumeFrom
	if config != nil && len(config.ResumeFrom) > 0 {
		currentNodes = config.ResumeFrom
	}

	// Generate run ID for callbacks
	runID := generateRunID()

	// Notify callbacks of graph start
	if config != nil {
		// Inject config into context
		ctx = WithConfig(ctx, config)

		// Inject ResumeValue
		if config.ResumeValue != nil {
			ctx = WithResumeValue(ctx, config.ResumeValue)
		}

		if len(config.Callbacks) > 0 {
			serialized := map[string]interface{}{
				"name": "graph",
				"type": "chain",
			}
			inputs := convertStateToMap(initialState)

			for _, cb := range config.Callbacks {
				cb.OnChainStart(ctx, serialized, inputs, runID, nil, config.Tags, config.Metadata)
			}
		}
	}

	// Start graph tracing if tracer is set
	var graphSpan *TraceSpan
	if r.tracer != nil {
		graphSpan = r.tracer.StartSpan(ctx, TraceEventGraphStart, "graph")
		graphSpan.State = initialState
	}

	for len(currentNodes) > 0 {
		// Filter out END nodes
		activeNodes := make([]string, 0, len(currentNodes))
		for _, node := range currentNodes {
			if node != END {
				activeNodes = append(activeNodes, node)
			}
		}
		currentNodes = activeNodes

		if len(currentNodes) == 0 {
			break
		}

		// Check InterruptBefore
		if config != nil && len(config.InterruptBefore) > 0 {
			for _, node := range currentNodes {
				for _, interrupt := range config.InterruptBefore {
					if node == interrupt {
						return state, &GraphInterrupt{Node: node, State: state}
					}
				}
			}
		}

		// Execute nodes in parallel
		var wg sync.WaitGroup
		results := make([]interface{}, len(currentNodes))
		errorsList := make([]error, len(currentNodes))

		for i, nodeName := range currentNodes {
			node, ok := r.graph.nodes[nodeName]
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeName)
			}

			wg.Add(1)
			go func(index int, n Node, name string) {
				defer wg.Done()

				// Recover from panics in node execution
				defer func() {
					if r := recover(); r != nil {
						errorsList[index] = fmt.Errorf("panic in node %s: %v", name, r)
					}
				}()

				// Start node tracing
				var nodeSpan *TraceSpan
				if r.tracer != nil {
					nodeSpan = r.tracer.StartSpan(ctx, TraceEventNodeStart, name)
					nodeSpan.State = state
				}

				var err error
				var res interface{}

				// Execute node with retry logic
				res, err = r.executeNodeWithRetry(ctx, n, state)

				// End node tracing
				if r.tracer != nil && nodeSpan != nil {
					if err != nil {
						r.tracer.EndSpan(ctx, nodeSpan, res, err)
						// Also emit error event
						errorSpan := r.tracer.StartSpan(ctx, TraceEventNodeError, name)
						errorSpan.Error = err
						errorSpan.State = res
						r.tracer.EndSpan(ctx, errorSpan, res, err)
					} else {
						r.tracer.EndSpan(ctx, nodeSpan, res, nil)
					}
				}

				if err != nil {
					var nodeInterrupt *NodeInterrupt
					if errors.As(err, &nodeInterrupt) {
						nodeInterrupt.Node = name
					}
					errorsList[index] = fmt.Errorf("error in node %s: %w", name, err)
					return
				}

				results[index] = res

				// Notify callbacks of node execution (as tool)
				if config != nil && len(config.Callbacks) > 0 {
					nodeRunID := generateRunID()
					serialized := map[string]interface{}{
						"name": name,
						"type": "tool",
					}
					for _, cb := range config.Callbacks {
						cb.OnToolStart(ctx, serialized, convertStateToString(res), nodeRunID, &runID, config.Tags, config.Metadata)
						cb.OnToolEnd(ctx, convertStateToString(res), nodeRunID)
					}
				}
			}(i, node, nodeName)
		}

		wg.Wait()

		// Check for errors
		for _, err := range errorsList {
			if err != nil {
				// Check for NodeInterrupt
				var nodeInterrupt *NodeInterrupt
				if errors.As(err, &nodeInterrupt) {
					return state, &GraphInterrupt{
						Node:           nodeInterrupt.Node,
						State:          state,
						InterruptValue: nodeInterrupt.Value,
						NextNodes:      []string{nodeInterrupt.Node},
					}
				}

				// Notify callbacks of error
				if config != nil && len(config.Callbacks) > 0 {
					for _, cb := range config.Callbacks {
						cb.OnChainError(ctx, err, runID)
					}
				}
				return nil, err
			}
		}

		// Process results and check for Commands
		var nextNodesFromCommands []string
		processedResults := make([]interface{}, len(results))

		for i, res := range results {
			if cmd, ok := res.(*Command); ok {
				// It's a Command
				processedResults[i] = cmd.Update

				if cmd.Goto != nil {
					switch g := cmd.Goto.(type) {
					case string:
						nextNodesFromCommands = append(nextNodesFromCommands, g)
					case []string:
						nextNodesFromCommands = append(nextNodesFromCommands, g...)
					}
				}
			} else {
				// Regular result
				processedResults[i] = res
			}
		}

		// Merge results
		if r.graph.Schema != nil {
			// If Schema is defined, use it to update state with results
			for _, res := range processedResults {
				var err error
				state, err = r.graph.Schema.Update(state, res)
				if err != nil {
					return nil, fmt.Errorf("schema update failed: %w", err)
				}
			}
		} else if r.graph.stateMerger != nil {
			var err error
			state, err = r.graph.stateMerger(ctx, state, processedResults)
			if err != nil {
				return nil, fmt.Errorf("state merge failed: %w", err)
			}
		} else {
			if len(processedResults) > 0 {
				state = processedResults[len(processedResults)-1]
			}
		}

		// Determine next nodes
		var nextNodesList []string

		if len(nextNodesFromCommands) > 0 {
			// Command.Goto overrides static edges
			// We deduplicate
			seen := make(map[string]bool)
			for _, n := range nextNodesFromCommands {
				if !seen[n] && n != END {
					seen[n] = true
					nextNodesList = append(nextNodesList, n)
				}
			}
		} else {
			// Use static edges
			nextNodesSet := make(map[string]bool)

			for _, nodeName := range currentNodes {
				// First check for conditional edges
				nextNodeFn, hasConditional := r.graph.conditionalEdges[nodeName]
				if hasConditional {
					nextNode := nextNodeFn(ctx, state)
					if nextNode == "" {
						return nil, fmt.Errorf("conditional edge returned empty next node from %s", nodeName)
					}
					nextNodesSet[nextNode] = true
				} else {
					// Then check regular edges
					foundNext := false
					for _, edge := range r.graph.edges {
						if edge.From == nodeName {
							nextNodesSet[edge.To] = true
							foundNext = true
							// Do NOT break here, to allow fan-out (multiple edges from same node)
						}
					}

					if !foundNext {
						return nil, fmt.Errorf("%w: %s", ErrNoOutgoingEdge, nodeName)
					}
				}
			}

			// Update nextNodesList from set
			for node := range nextNodesSet {
				nextNodesList = append(nextNodesList, node)
			}
		}

		// Check InterruptAfter
		if config != nil && len(config.InterruptAfter) > 0 {
			for _, node := range currentNodes {
				for _, interrupt := range config.InterruptAfter {
					if node == interrupt {
						return state, &GraphInterrupt{
							Node:      node,
							State:     state,
							NextNodes: nextNodesList,
						}
					}
				}
			}
		}

		// Keep track of nodes that ran for callbacks
		nodesRan := make([]string, len(currentNodes))
		copy(nodesRan, currentNodes)

		// Update currentNodes
		currentNodes = nextNodesList

		// Cleanup ephemeral state if supported
		if cleaningSchema, ok := r.graph.Schema.(CleaningStateSchema); ok {
			state = cleaningSchema.Cleanup(state)
		}

		// Notify callbacks of step completion
		if config != nil && len(config.Callbacks) > 0 {
			for _, cb := range config.Callbacks {
				if gcb, ok := cb.(GraphCallbackHandler); ok {
					var nodeName string
					if len(nodesRan) == 1 {
						nodeName = nodesRan[0]
					} else {
						nodeName = fmt.Sprintf("step:%v", nodesRan)
					}
					gcb.OnGraphStep(ctx, nodeName, state)
				}
			}
		}
	}

	// End graph tracing
	if r.tracer != nil && graphSpan != nil {
		r.tracer.EndSpan(ctx, graphSpan, state, nil)
	}

	// Notify callbacks of graph end
	if config != nil && len(config.Callbacks) > 0 {
		outputs := convertStateToMap(state)
		for _, cb := range config.Callbacks {
			cb.OnChainEnd(ctx, outputs, runID)
		}
	}

	return state, nil
}

// executeNodeWithRetry executes a node with retry logic based on the retry policy
func (r *StateRunnable) executeNodeWithRetry(ctx context.Context, node Node, state interface{}) (interface{}, error) {
	var lastErr error

	maxRetries := 1 // Default: no retries
	if r.graph.retryPolicy != nil {
		maxRetries = r.graph.retryPolicy.MaxRetries + 1 // +1 for initial attempt
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := node.Function(ctx, state)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if r.graph.retryPolicy != nil && attempt < maxRetries-1 {
			if r.isRetryableError(err) {
				// Apply backoff strategy
				delay := r.calculateBackoffDelay(attempt)
				if delay > 0 {
					select {
					case <-time.After(delay):
						// Continue with retry after delay
					case <-ctx.Done():
						// Context cancelled, return immediately
						return nil, ctx.Err()
					}
				}
				continue
			}
		}

		// If not retryable or max retries reached, return error
		break
	}

	return nil, lastErr
}

// isRetryableError checks if an error is retryable based on the retry policy
func (r *StateRunnable) isRetryableError(err error) bool {
	if r.graph.retryPolicy == nil {
		return false
	}

	errorStr := err.Error()
	for _, retryablePattern := range r.graph.retryPolicy.RetryableErrors {
		if contains(errorStr, retryablePattern) {
			return true
		}
	}

	return false
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(substr) > 0 && len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

// findSubstring finds if substr exists in s
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// calculateBackoffDelay calculates the delay for retry based on the backoff strategy
func (r *StateRunnable) calculateBackoffDelay(attempt int) time.Duration {
	if r.graph.retryPolicy == nil {
		return 0
	}

	baseDelay := time.Second // Default 1 second base delay

	switch r.graph.retryPolicy.BackoffStrategy {
	case FixedBackoff:
		return baseDelay
	case ExponentialBackoff:
		// Exponential backoff: 1s, 2s, 4s, 8s, ...
		return baseDelay * time.Duration(1<<attempt)
	case LinearBackoff:
		// Linear backoff: 1s, 2s, 3s, 4s, ...
		return baseDelay * time.Duration(attempt+1)
	default:
		return baseDelay
	}
}
