package graph

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// NodeEvent represents different types of node events
type NodeEvent string

const (
	// NodeEventStart indicates a node has started execution
	NodeEventStart NodeEvent = "start"

	// NodeEventProgress indicates progress during node execution
	NodeEventProgress NodeEvent = "progress"

	// NodeEventComplete indicates a node has completed successfully
	NodeEventComplete NodeEvent = "complete"

	// NodeEventError indicates a node encountered an error
	NodeEventError NodeEvent = "error"

	// EventChainStart indicates the graph execution has started
	EventChainStart NodeEvent = "chain_start"

	// EventChainEnd indicates the graph execution has completed
	EventChainEnd NodeEvent = "chain_end"

	// EventToolStart indicates a tool execution has started
	EventToolStart NodeEvent = "tool_start"

	// EventToolEnd indicates a tool execution has completed
	EventToolEnd NodeEvent = "tool_end"

	// EventLLMStart indicates an LLM call has started
	EventLLMStart NodeEvent = "llm_start"

	// EventLLMEnd indicates an LLM call has completed
	EventLLMEnd NodeEvent = "llm_end"

	// EventToken indicates a generated token (for streaming)
	EventToken NodeEvent = "token"

	// EventCustom indicates a custom user-defined event
	EventCustom NodeEvent = "custom"
)

// NodeListener defines the interface for typed node event listeners
type NodeListener[S any] interface {
	// OnNodeEvent is called when a node event occurs
	OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error)
}

// NodeListenerFunc is a function adapter for NodeListener
type NodeListenerFunc[S any] func(ctx context.Context, event NodeEvent, nodeName string, state S, err error)

// OnNodeEvent implements the NodeListener interface
func (f NodeListenerFunc[S]) OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error) {
	f(ctx, event, nodeName, state, err)
}

// StreamEvent represents a typed event in the streaming execution
type StreamEvent[S any] struct {
	// Timestamp when the event occurred
	Timestamp time.Time

	// NodeName is the name of the node that generated the event
	NodeName string

	// Event is the type of event
	Event NodeEvent

	// State is the current state at the time of the event (typed)
	State S

	// Error contains any error that occurred (if Event is NodeEventError)
	Error error

	// Metadata contains additional event-specific data
	Metadata map[string]any

	// Duration is how long the node took (only for Complete events)
	Duration time.Duration
}

// listenerWrapper wraps a listener with a unique ID for comparison
type listenerWrapper[S any] struct {
	id       string
	listener NodeListener[S]
}

// ListenableNode extends TypedNode with listener capabilities
type ListenableNode[S any] struct {
	TypedNode[S]
	listeners []listenerWrapper[S]
	mutex     sync.RWMutex
	nextID    int64
}

// NewListenableNode creates a new listenable node from a regular typed node
func NewListenableNode[S any](node TypedNode[S]) *ListenableNode[S] {
	return &ListenableNode[S]{
		TypedNode: node,
		listeners: make([]listenerWrapper[S], 0),
		nextID:    1,
	}
}

// AddListener adds a listener to the node and returns the listenable node for chaining
func (ln *ListenableNode[S]) AddListener(listener NodeListener[S]) *ListenableNode[S] {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	id := fmt.Sprintf("listener_%d", ln.nextID)
	ln.nextID++

	ln.listeners = append(ln.listeners, listenerWrapper[S]{
		id:       id,
		listener: listener,
	})
	return ln
}

// AddListenerWithID adds a listener to the node and returns its ID
func (ln *ListenableNode[S]) AddListenerWithID(listener NodeListener[S]) string {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	id := fmt.Sprintf("listener_%d", ln.nextID)
	ln.nextID++

	ln.listeners = append(ln.listeners, listenerWrapper[S]{
		id:       id,
		listener: listener,
	})
	return id
}

// RemoveListener removes a listener from the node by ID
func (ln *ListenableNode[S]) RemoveListener(listenerID string) {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	for i, lw := range ln.listeners {
		if lw.id == listenerID {
			ln.listeners = append(ln.listeners[:i], ln.listeners[i+1:]...)
			break
		}
	}
}

// RemoveListenerByFunc removes a listener from the node by comparing listener values
func (ln *ListenableNode[S]) RemoveListenerByFunc(listener NodeListener[S]) {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	for i, lw := range ln.listeners {
		// Use reflect.DeepEqual for proper interface value comparison
		if reflect.DeepEqual(lw.listener, listener) {
			ln.listeners = append(ln.listeners[:i], ln.listeners[i+1:]...)
			break
		}
	}
}

// NotifyListeners notifies all listeners of an event
func (ln *ListenableNode[S]) NotifyListeners(ctx context.Context, event NodeEvent, state S, err error) {
	ln.mutex.RLock()
	wrappers := make([]listenerWrapper[S], len(ln.listeners))
	copy(wrappers, ln.listeners)
	ln.mutex.RUnlock()

	// Use WaitGroup to synchronize listener notifications
	var wg sync.WaitGroup

	// Notify listeners in separate goroutines to avoid blocking execution
	for _, wrapper := range wrappers {
		wg.Add(1)
		go func(l NodeListener[S]) {
			defer wg.Done()

			// Protect against panics in listeners
			defer func() {
				if r := recover(); r != nil {
					// Panic recovered, but not logged to avoid dependencies
					_ = r // Acknowledge the panic was caught
				}
			}()

			l.OnNodeEvent(ctx, event, ln.Name, state, err)
		}(wrapper.listener)
	}

	// Wait for all listener notifications to complete
	wg.Wait()
}

// Execute runs the node function with listener notifications
func (ln *ListenableNode[S]) Execute(ctx context.Context, state S) (S, error) {
	// Notify start
	ln.NotifyListeners(ctx, NodeEventStart, state, nil)

	// Execute the node function
	result, err := ln.Function(ctx, state)

	// Notify completion or error
	if err != nil {
		ln.NotifyListeners(ctx, NodeEventError, state, err)
	} else {
		ln.NotifyListeners(ctx, NodeEventComplete, result, nil)
	}

	return result, err
}

// GetListeners returns a copy of the current listeners
func (ln *ListenableNode[S]) GetListeners() []NodeListener[S] {
	ln.mutex.RLock()
	defer ln.mutex.RUnlock()

	listeners := make([]NodeListener[S], len(ln.listeners))
	for i, wrapper := range ln.listeners {
		listeners[i] = wrapper.listener
	}
	return listeners
}

// GetListenerIDs returns a copy of the current listener IDs
func (ln *ListenableNode[S]) GetListenerIDs() []string {
	ln.mutex.RLock()
	defer ln.mutex.RUnlock()

	ids := make([]string, len(ln.listeners))
	for i, wrapper := range ln.listeners {
		ids[i] = wrapper.id
	}
	return ids
}

// ListenableStateGraph extends StateGraph with listener capabilities
type ListenableStateGraph[S any] struct {
	*StateGraph[S]
	listenableNodes map[string]*ListenableNode[S]
}

// NewListenableStateGraph creates a new typed state graph with listener support
func NewListenableStateGraph[S any]() *ListenableStateGraph[S] {
	return &ListenableStateGraph[S]{
		StateGraph:      NewStateGraph[S](),
		listenableNodes: make(map[string]*ListenableNode[S]),
	}
}

// AddNode adds a node with listener capabilities
func (g *ListenableStateGraph[S]) AddNode(name string, description string, fn func(ctx context.Context, state S) (S, error)) *ListenableNode[S] {
	node := TypedNode[S]{
		Name:        name,
		Description: description,
		Function:    fn,
	}

	listenableNode := NewListenableNode(node)

	// Add to both the base graph and our listenable nodes map
	g.StateGraph.AddNode(name, description, fn)
	g.listenableNodes[name] = listenableNode

	return listenableNode
}

// GetListenableNode returns the listenable node by name
func (g *ListenableStateGraph[S]) GetListenableNode(name string) *ListenableNode[S] {
	return g.listenableNodes[name]
}

// AddGlobalListener adds a listener to all nodes in the graph
func (g *ListenableStateGraph[S]) AddGlobalListener(listener NodeListener[S]) {
	for _, node := range g.listenableNodes {
		node.AddListener(listener)
	}
}

// RemoveGlobalListener removes a listener from all nodes in the graph by function reference
func (g *ListenableStateGraph[S]) RemoveGlobalListener(listener NodeListener[S]) {
	for _, node := range g.listenableNodes {
		node.RemoveListenerByFunc(listener)
	}
}

// RemoveGlobalListenerByID removes a listener from all nodes in the graph by ID
func (g *ListenableStateGraph[S]) RemoveGlobalListenerByID(listenerID string) {
	for _, node := range g.listenableNodes {
		node.RemoveListener(listenerID)
	}
}

// ListenableRunnable wraps a StateRunnable with listener capabilities
type ListenableRunnable[S any] struct {
	graph           *ListenableStateGraph[S]
	listenableNodes map[string]*ListenableNode[S]
	runnable        *StateRunnable[S]
}

// CompileListenable creates a runnable with listener support
func (g *ListenableStateGraph[S]) CompileListenable() (*ListenableRunnable[S], error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	runnable, err := g.StateGraph.Compile()
	if err != nil {
		return nil, err
	}

	// Configure the runnable to use our listenable nodes
	nodes := g.listenableNodes
	runnable.nodeRunner = func(ctx context.Context, nodeName string, state S) (S, error) {
		node, ok := nodes[nodeName]
		if !ok {
			var zero S
			return zero, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeName)
		}
		return node.Execute(ctx, state)
	}

	return &ListenableRunnable[S]{
		graph:           g,
		listenableNodes: g.listenableNodes,
		runnable:        runnable,
	}, nil
}

// Invoke executes the graph with listener notifications
func (lr *ListenableRunnable[S]) Invoke(ctx context.Context, initialState S) (S, error) {
	return lr.runnable.Invoke(ctx, initialState)
}

// InvokeWithConfig executes the graph with listener notifications and config
func (lr *ListenableRunnable[S]) InvokeWithConfig(ctx context.Context, initialState S, config *Config) (S, error) {
	if config != nil {
		ctx = WithConfig(ctx, config)
	}
	return lr.runnable.InvokeWithConfig(ctx, initialState, config)
}

// Stream executes the graph with listener notifications and streams events
func (lr *ListenableRunnable[S]) Stream(ctx context.Context, initialState S) <-chan StreamEvent[S] {
	eventChan := make(chan StreamEvent[S], 100) // Buffered channel

	// Create a streaming listener
	// Using DefaultStreamConfig from streaming.go
	streamListener := NewStreamingListener(eventChan, DefaultStreamConfig())

	// Add the listener to all nodes
	lr.graph.AddGlobalListener(streamListener)

	// Start execution in a goroutine
	go func() {
		defer func() {
			// Clean up: remove the listener and close the channel
			// We remove the listener first to stop new events
			lr.graph.RemoveGlobalListener(streamListener)
			// Close the listener (sets internal flag)
			streamListener.Close()
			// Close the channel
			close(eventChan)
		}()

		// Send chain start event
		eventChan <- StreamEvent[S]{
			Timestamp: time.Now(),
			Event:     EventChainStart,
			State:     initialState,
		}

		// Execute the graph
		_, err := lr.runnable.Invoke(ctx, initialState)

		// Send chain end event
		eventChan <- StreamEvent[S]{
			Timestamp: time.Now(),
			Event:     EventChainEnd,
			State:     initialState, // Note: This should be the final state
			Error:     err,
		}
	}()

	return eventChan
}

// SetTracer sets a tracer for the underlying runnable
func (lr *ListenableRunnable[S]) SetTracer(tracer *Tracer) {
	lr.runnable.SetTracer(tracer)
}

// GetTracer returns the tracer from the underlying runnable
func (lr *ListenableRunnable[S]) GetTracer() *Tracer {
	return lr.runnable.GetTracer()
}

// WithTracer returns a new ListenableRunnableWith the given tracer
func (lr *ListenableRunnable[S]) WithTracer(tracer *Tracer) *ListenableRunnable[S] {
	newRunnable := lr.runnable.WithTracer(tracer)
	return &ListenableRunnable[S]{
		graph:           lr.graph,
		listenableNodes: lr.listenableNodes,
		runnable:        newRunnable,
	}
}

// GetGraph returns an Exporter for visualization
func (lr *ListenableRunnable[S]) GetGraph() *Exporter[S] {
	// Convert the typed graph to a regular graph for visualization
	regularGraph := lr.convertToRegularGraph()
	return NewExporter[S](regularGraph)
}

// GetListenableGraph returns the underlying ListenableStateGraph
func (lr *ListenableRunnable[S]) GetListenableGraph() *ListenableStateGraph[S] {
	return lr.graph
}

// convertToRegularGraph converts a StateGraph[S] to StateGraph[map[string]any] for visualization
func (lr *ListenableRunnable[S]) convertToRegularGraph() *StateGraph[S] {
	// For visualization of typed graphs, we just return the original graph
	// The Exporter[S] can work with any StateGraph[S]
	return lr.graph.StateGraph
}
