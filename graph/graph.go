package graph

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// END is a special constant used to represent the end node in the graph.
const END = "END"

var (
	// ErrEntryPointNotSet is returned when the entry point of the graph is not set.
	ErrEntryPointNotSet = errors.New("entry point not set")

	// ErrNodeNotFound is returned when a node is not found in the graph.
	ErrNodeNotFound = errors.New("node not found")

	// ErrNoOutgoingEdge is returned when no outgoing edge is found for a node.
	ErrNoOutgoingEdge = errors.New("no outgoing edge found for node")
)

// GraphInterrupt is returned when execution is interrupted by configuration or dynamic interrupt
type GraphInterrupt struct {
	// Node that caused the interruption
	Node string
	// State at the time of interruption
	State interface{}
	// NextNodes that would have been executed if not interrupted
	NextNodes []string
	// InterruptValue is the value provided by the dynamic interrupt (if any)
	InterruptValue interface{}
}

func (e *GraphInterrupt) Error() string {
	if e.InterruptValue != nil {
		return fmt.Sprintf("graph interrupted at node %s with value: %v", e.Node, e.InterruptValue)
	}
	return fmt.Sprintf("graph interrupted at node %s", e.Node)
}

// Interrupt pauses execution and waits for input.
// If resuming, it returns the value provided in the resume command.
func Interrupt(ctx context.Context, value interface{}) (interface{}, error) {
	if resumeVal := GetResumeValue(ctx); resumeVal != nil {
		return resumeVal, nil
	}
	return nil, &NodeInterrupt{Value: value}
}

// Node represents a node in the message graph.
type Node struct {
	// Name is the unique identifier for the node.
	Name string

	// Description describes the functionality of the node.
	Description string

	// Function is the function associated with the node.
	// It takes a context and any state as input and returns the updated state and an error.
	Function func(ctx context.Context, state interface{}) (interface{}, error)
}

// Edge represents an edge in the message graph.
type Edge struct {
	// From is the name of the node from which the edge originates.
	From string

	// To is the name of the node to which the edge points.
	To string
}

// StateMerger merges multiple state updates into a single state.
type StateMerger func(ctx context.Context, currentState interface{}, newStates []interface{}) (interface{}, error)

// MessageGraph represents a message graph.
type MessageGraph struct {
	// nodes is a map of node names to their corresponding Node objects.
	nodes map[string]Node

	// edges is a slice of Edge objects representing the connections between nodes.
	edges []Edge

	// conditionalEdges contains a map between "From" node, while "To" node is derived based on the condition.
	conditionalEdges map[string]func(ctx context.Context, state interface{}) string

	// entryPoint is the name of the entry point node in the graph.
	entryPoint string

	// stateMerger is an optional function to merge states from parallel execution.
	stateMerger StateMerger

	// Schema defines the state structure and update logic
	Schema StateSchema
}

// NewMessageGraph creates a new instance of MessageGraph.
// For backward compatibility, this version does not initialize a schema.
// If you want a MessageGraph with a default schema for message handling,
// use NewMessageGraphWithSchema() instead.
func NewMessageGraph() *MessageGraph {
	return &MessageGraph{
		nodes:            make(map[string]Node),
		conditionalEdges: make(map[string]func(ctx context.Context, state interface{}) string),
	}
}

// NewMessageGraphWithSchema creates a new MessageGraph with a default schema
// that handles "messages" using the AddMessages reducer.
// This is the recommended constructor for chat-based agents that use
// map[string]interface{} as state with a "messages" key.
func NewMessageGraphWithSchema() *MessageGraph {
	g := &MessageGraph{
		nodes:            make(map[string]Node),
		conditionalEdges: make(map[string]func(ctx context.Context, state interface{}) string),
	}

	// Initialize default schema for message handling
	schema := NewMapSchema()
	schema.RegisterReducer("messages", AddMessages)
	g.Schema = schema

	return g
}

// AddNode adds a new node to the message graph with the given name, description and function.
func (g *MessageGraph) AddNode(name string, description string, fn func(ctx context.Context, state interface{}) (interface{}, error)) {
	g.nodes[name] = Node{
		Name:        name,
		Description: description,
		Function:    fn,
	}
}

// AddEdge adds a new edge to the message graph between the "from" and "to" nodes.
func (g *MessageGraph) AddEdge(from, to string) {
	g.edges = append(g.edges, Edge{
		From: from,
		To:   to,
	})
}

// AddConditionalEdge adds a conditional edge where the target node is determined at runtime.
// The condition function receives the current state and returns the name of the next node.
func (g *MessageGraph) AddConditionalEdge(from string, condition func(ctx context.Context, state interface{}) string) {
	g.conditionalEdges[from] = condition
}

// SetEntryPoint sets the entry point node name for the message graph.
func (g *MessageGraph) SetEntryPoint(name string) {
	g.entryPoint = name
}

// SetStateMerger sets the state merger function for the message graph.
func (g *MessageGraph) SetStateMerger(merger StateMerger) {
	g.stateMerger = merger
}

// SetSchema sets the state schema for the message graph.
func (g *MessageGraph) SetSchema(schema StateSchema) {
	g.Schema = schema
}

// Runnable represents a compiled message graph that can be invoked.
type Runnable struct {
	// graph is the underlying MessageGraph object.
	graph *MessageGraph
	// tracer is the optional tracer for observability
	tracer *Tracer
}

// Compile compiles the message graph and returns a Runnable instance.
// It returns an error if the entry point is not set.
func (g *MessageGraph) Compile() (*Runnable, error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	return &Runnable{
		graph:  g,
		tracer: nil, // Initialize with no tracer
	}, nil
}

// SetTracer sets a tracer for observability
func (r *Runnable) SetTracer(tracer *Tracer) {
	r.tracer = tracer
}

// WithTracer returns a new Runnable with the given tracer
func (r *Runnable) WithTracer(tracer *Tracer) *Runnable {
	return &Runnable{
		graph:  r.graph,
		tracer: tracer,
	}
}

// Invoke executes the compiled message graph with the given input state.
// It returns the resulting state and an error if any occurs during the execution.
func (r *Runnable) Invoke(ctx context.Context, initialState interface{}) (interface{}, error) {
	return r.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig executes the compiled message graph with the given input state and config.
// It returns the resulting state and an error if any occurs during the execution.
func (r *Runnable) InvokeWithConfig(ctx context.Context, initialState interface{}, config *Config) (interface{}, error) {
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

				// Pass the current state to the node
				// Note: If state is mutable and shared, this is not thread-safe unless handled by user.
				res, err = n.Function(ctx, state)

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

		// Merge results (using processedResults)
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
			// Default behavior
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

			// Update currentNodes
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
