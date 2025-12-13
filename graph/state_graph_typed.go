package graph

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// StateGraphTyped represents a generic state-based graph with compile-time type safety.
// The type parameter S represents the state type, which is typically a struct.
//
// Example usage:
//
//	type MyState struct {
//	    Count int
//	    Name  string
//	}
//
//	g := graph.NewStateGraphTyped[MyState]()
//	g.AddNode("increment", "Increment counter", func(ctx context.Context, state MyState) (MyState, error) {
//	    state.Count++
//	    return state, nil
//	})
type StateGraphTyped[S any] struct {
	// nodes is a map of node names to their corresponding Node objects
	nodes map[string]NodeTyped[S]

	// edges is a slice of Edge objects representing the connections between nodes
	edges []Edge

	// conditionalEdges contains a map between "From" node, while "To" node is derived based on the condition
	conditionalEdges map[string]func(ctx context.Context, state S) string

	// entryPoint is the name of the entry point node in the graph
	entryPoint string

	// retryPolicy defines retry behavior for failed nodes
	retryPolicy *RetryPolicy

	// stateMerger is an optional function to merge states from parallel execution
	stateMerger StateMergerTyped[S]

	// Schema defines the state structure and update logic
	Schema StateSchemaTyped[S]
}

// NodeTyped represents a typed node in the graph.
type NodeTyped[S any] struct {
	Name        string
	Description string
	Function    func(ctx context.Context, state S) (S, error)
}

// StateMergerTyped is a typed function to merge states from parallel execution.
type StateMergerTyped[S any] func(ctx context.Context, currentState S, newStates []S) (S, error)

// NewStateGraphTyped creates a new instance of StateGraphTyped with type safety.
// The type parameter S specifies the state type.
//
// Example:
//
//	g := graph.NewStateGraphTyped[MyState]()
func NewStateGraphTyped[S any]() *StateGraphTyped[S] {
	return &StateGraphTyped[S]{
		nodes:            make(map[string]NodeTyped[S]),
		conditionalEdges: make(map[string]func(ctx context.Context, state S) string),
	}
}

// AddNode adds a new node to the state graph with the given name, description and function.
// The node function is fully typed - no type assertions needed!
//
// Example:
//
//	g.AddNode("process", "Process data", func(ctx context.Context, state MyState) (MyState, error) {
//	    state.Count++  // Type-safe access!
//	    return state, nil
//	})
func (g *StateGraphTyped[S]) AddNode(name string, description string, fn func(ctx context.Context, state S) (S, error)) {
	g.nodes[name] = NodeTyped[S]{
		Name:        name,
		Description: description,
		Function:    fn,
	}
}

// AddEdge adds a new edge to the state graph between the "from" and "to" nodes.
func (g *StateGraphTyped[S]) AddEdge(from, to string) {
	g.edges = append(g.edges, Edge{
		From: from,
		To:   to,
	})
}

// AddConditionalEdge adds a conditional edge where the target node is determined at runtime.
// The condition function is fully typed - no type assertions needed!
//
// Example:
//
//	g.AddConditionalEdge("check", func(ctx context.Context, state MyState) string {
//	    if state.Count > 10 {  // Type-safe access!
//	        return "high"
//	    }
//	    return "low"
//	})
func (g *StateGraphTyped[S]) AddConditionalEdge(from string, condition func(ctx context.Context, state S) string) {
	g.conditionalEdges[from] = condition
}

// SetEntryPoint sets the entry point node name for the state graph.
func (g *StateGraphTyped[S]) SetEntryPoint(name string) {
	g.entryPoint = name
}

// SetRetryPolicy sets the retry policy for the graph.
func (g *StateGraphTyped[S]) SetRetryPolicy(policy *RetryPolicy) {
	g.retryPolicy = policy
}

// SetStateMerger sets the state merger function for the state graph.
func (g *StateGraphTyped[S]) SetStateMerger(merger StateMergerTyped[S]) {
	g.stateMerger = merger
}

// SetSchema sets the state schema for the graph.
func (g *StateGraphTyped[S]) SetSchema(schema StateSchemaTyped[S]) {
	g.Schema = schema
}

// StateRunnableTyped represents a compiled state graph that can be invoked with type safety.
type StateRunnableTyped[S any] struct {
	graph      *StateGraphTyped[S]
	tracer     *Tracer
	nodeRunner func(ctx context.Context, nodeName string, state S) (S, error)
}

// Compile compiles the state graph and returns a StateRunnableTyped instance.
func (g *StateGraphTyped[S]) Compile() (*StateRunnableTyped[S], error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	return &StateRunnableTyped[S]{
		graph:  g,
		tracer: nil, // Initialize with no tracer
	}, nil
}

// SetTracer sets a tracer for observability.
func (r *StateRunnableTyped[S]) SetTracer(tracer *Tracer) {
	r.tracer = tracer
}

// WithTracer returns a new StateRunnableTyped with the given tracer.
func (r *StateRunnableTyped[S]) WithTracer(tracer *Tracer) *StateRunnableTyped[S] {
	return &StateRunnableTyped[S]{
		graph:  r.graph,
		tracer: tracer,
	}
}

// Invoke executes the compiled state graph with the given input state.
// Returns the final state with full type safety - no type assertions needed!
//
// Example:
//
//	initialState := MyState{Count: 0}
//	finalState, err := app.Invoke(ctx, initialState)
//	// finalState is MyState type - no casting needed!
func (r *StateRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error) {
	return r.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig executes the compiled state graph with the given input state and config.
func (r *StateRunnableTyped[S]) InvokeWithConfig(ctx context.Context, initialState S, config *Config) (S, error) {
	state := initialState

	// If schema is defined, merge initialState into schema's initial state
	if r.graph.Schema != nil {
		schemaInit := r.graph.Schema.Init()
		var err error
		state, err = r.graph.Schema.Update(schemaInit, initialState)
		if err != nil {
			var zero S
			return zero, fmt.Errorf("failed to initialize state with schema: %w", err)
		}
	}

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
			serialized := map[string]any{
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
		results, errorsList := r.executeNodesParallel(ctx, currentNodes, state, config, runID)

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
				var zero S
				return zero, err
			}
		}

		// Process results
		processedResults, nextNodesFromCommands := r.processNodeResults(results)

		// Merge results
		var err error
		state, err = r.mergeState(ctx, state, processedResults)
		if err != nil {
			var zero S
			return zero, err
		}

		// Determine next nodes
		nextNodesList, err := r.determineNextNodes(ctx, currentNodes, state, nextNodesFromCommands)
		if err != nil {
			var zero S
			return zero, err
		}

		// Keep track of nodes that ran for callbacks and interrupts
		nodesRan := make([]string, len(currentNodes))
		copy(nodesRan, currentNodes)

		// Update currentNodes
		currentNodes = nextNodesList

		// Cleanup ephemeral state if supported
		if cleaningSchema, ok := r.graph.Schema.(CleaningStateSchemaTyped[S]); ok {
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

		// Check InterruptAfter
		if config != nil && len(config.InterruptAfter) > 0 {
			for _, node := range nodesRan {
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

// executeNodeWithRetry executes a node with retry logic based on the retry policy.
func (r *StateRunnableTyped[S]) executeNodeWithRetry(ctx context.Context, node NodeTyped[S], state S) (S, error) {
	var lastErr error
	var zero S

	maxRetries := 1 // Default: no retries
	if r.graph.retryPolicy != nil {
		maxRetries = r.graph.retryPolicy.MaxRetries + 1 // +1 for initial attempt
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		var result S
		var err error

		if r.nodeRunner != nil {
			result, err = r.nodeRunner(ctx, node.Name, state)
		} else {
			result, err = node.Function(ctx, state)
		}

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
						return zero, ctx.Err()
					}
				}
				continue
			}
		}

		// If not retryable or max retries reached, return error
		break
	}

	return zero, lastErr
}

// isRetryableError checks if an error is retryable based on the retry policy.
func (r *StateRunnableTyped[S]) isRetryableError(err error) bool {
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

// calculateBackoffDelay calculates the delay for retry based on the backoff strategy.
func (r *StateRunnableTyped[S]) calculateBackoffDelay(attempt int) time.Duration {
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

// executeNodesParallel executes valid nodes in parallel and returns their results or errors.
func (r *StateRunnableTyped[S]) executeNodesParallel(ctx context.Context, nodes []string, state S, config *Config, runID string) ([]S, []error) {
	var wg sync.WaitGroup
	results := make([]S, len(nodes))
	errorsList := make([]error, len(nodes))

	for i, nodeName := range nodes {
		node, ok := r.graph.nodes[nodeName]
		if !ok {
			errorsList[i] = fmt.Errorf("%w: %s", ErrNodeNotFound, nodeName)
			continue
		}

		// Prepare variables for closure
		idx := i
		n := node
		name := nodeName

		SafeGo(&wg, func() {
			// Start node tracing
			var nodeSpan *TraceSpan
			if r.tracer != nil {
				nodeSpan = r.tracer.StartSpan(ctx, TraceEventNodeStart, name)
				nodeSpan.State = state
			}

			var err error
			var res S

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
				errorsList[idx] = fmt.Errorf("error in node %s: %w", name, err)
				return
			}

			results[idx] = res

			// Notify callbacks of node execution (as tool)
			if config != nil && len(config.Callbacks) > 0 {
				nodeRunID := generateRunID()
				serialized := map[string]any{
					"name": name,
					"type": "tool",
				}
				for _, cb := range config.Callbacks {
					cb.OnToolStart(ctx, serialized, convertStateToString(res), nodeRunID, &runID, config.Tags, config.Metadata)
					cb.OnToolEnd(ctx, convertStateToString(res), nodeRunID)
				}
			}
		}, func(panicVal any) {
			errorsList[idx] = fmt.Errorf("panic in node %s: %v", name, panicVal)
		})
	}
	wg.Wait()
	return results, errorsList
}

// processNodeResults processes the raw results from nodes, handling Commands.
func (r *StateRunnableTyped[S]) processNodeResults(results []S) ([]S, []string) {
	var nextNodesFromCommands []string
	processedResults := make([]S, len(results))

	for i, res := range results {
		// Note: In the generic version, we can't easily support Command[S] without type assertions
		// because S is any type. Commands would need to be handled differently.
		// For now, we just pass through the results as-is.
		// TODO: Consider how to support Command with generics
		processedResults[i] = res
	}
	return processedResults, nextNodesFromCommands
}

// mergeState merges the processed results into the current state.
func (r *StateRunnableTyped[S]) mergeState(ctx context.Context, currentState S, results []S) (S, error) {
	state := currentState
	if r.graph.Schema != nil {
		// If Schema is defined, use it to update state with results
		for _, res := range results {
			var err error
			state, err = r.graph.Schema.Update(state, res)
			if err != nil {
				var zero S
				return zero, fmt.Errorf("schema update failed: %w", err)
			}
		}
	} else if r.graph.stateMerger != nil {
		var err error
		state, err = r.graph.stateMerger(ctx, state, results)
		if err != nil {
			var zero S
			return zero, fmt.Errorf("state merge failed: %w", err)
		}
	} else {
		if len(results) > 0 {
			state = results[len(results)-1]
		}
	}
	return state, nil
}

// determineNextNodes determines the next nodes to execute based on static edges, conditional edges, or commands.
func (r *StateRunnableTyped[S]) determineNextNodes(ctx context.Context, currentNodes []string, state S, nextNodesFromCommands []string) ([]string, error) {
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
					var zero S
					_ = zero
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
	return nextNodesList, nil
}
