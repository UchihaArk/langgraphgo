package graph

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NodeListenerTyped defines the interface for typed node event listeners
type NodeListenerTyped[S any] interface {
	// OnNodeEvent is called when a node event occurs
	OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error)
}

// NodeListenerTypedFunc is a function adapter for NodeListenerTyped
type NodeListenerTypedFunc[S any] func(ctx context.Context, event NodeEvent, nodeName string, state S, err error)

// OnNodeEvent implements the NodeListenerTyped interface
func (f NodeListenerTypedFunc[S]) OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error) {
	f(ctx, event, nodeName, state, err)
}

// StreamEventTyped represents a typed event in the streaming execution
type StreamEventTyped[S any] struct {
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
	listener NodeListenerTyped[S]
}

// ListenableNodeTyped extends NodeTyped with listener capabilities
type ListenableNodeTyped[S any] struct {
	NodeTyped[S]
	listeners []listenerWrapper[S]
	mutex     sync.RWMutex
	nextID    int64
}

// NewListenableNodeTyped creates a new listenable node from a regular typed node
func NewListenableNodeTyped[S any](node NodeTyped[S]) *ListenableNodeTyped[S] {
	return &ListenableNodeTyped[S]{
		NodeTyped: node,
		listeners: make([]listenerWrapper[S], 0),
		nextID:    1,
	}
}

// AddListener adds a listener to the node and returns the listenable node for chaining
func (ln *ListenableNodeTyped[S]) AddListener(listener NodeListenerTyped[S]) *ListenableNodeTyped[S] {
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
func (ln *ListenableNodeTyped[S]) AddListenerWithID(listener NodeListenerTyped[S]) string {
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
func (ln *ListenableNodeTyped[S]) RemoveListener(listenerID string) {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	for i, lw := range ln.listeners {
		if lw.id == listenerID {
			ln.listeners = append(ln.listeners[:i], ln.listeners[i+1:]...)
			break
		}
	}
}

// RemoveListenerByFunc removes a listener from the node by comparing pointer values
func (ln *ListenableNodeTyped[S]) RemoveListenerByFunc(listener NodeListenerTyped[S]) {
	ln.mutex.Lock()
	defer ln.mutex.Unlock()

	for i, lw := range ln.listeners {
		// Compare pointer values for reference equality
		if &lw.listener == &listener ||
		   fmt.Sprintf("%p", lw.listener) == fmt.Sprintf("%p", listener) {
			ln.listeners = append(ln.listeners[:i], ln.listeners[i+1:]...)
			break
		}
	}
}

// NotifyListeners notifies all listeners of an event
func (ln *ListenableNodeTyped[S]) NotifyListeners(ctx context.Context, event NodeEvent, state S, err error) {
	ln.mutex.RLock()
	wrappers := make([]listenerWrapper[S], len(ln.listeners))
	copy(wrappers, ln.listeners)
	ln.mutex.RUnlock()

	// Use WaitGroup to synchronize listener notifications
	var wg sync.WaitGroup

	// Notify listeners in separate goroutines to avoid blocking execution
	for _, wrapper := range wrappers {
		wg.Add(1)
		go func(l NodeListenerTyped[S]) {
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
func (ln *ListenableNodeTyped[S]) Execute(ctx context.Context, state S) (S, error) {
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
func (ln *ListenableNodeTyped[S]) GetListeners() []NodeListenerTyped[S] {
	ln.mutex.RLock()
	defer ln.mutex.RUnlock()

	listeners := make([]NodeListenerTyped[S], len(ln.listeners))
	for i, wrapper := range ln.listeners {
		listeners[i] = wrapper.listener
	}
	return listeners
}

// GetListenerIDs returns a copy of the current listener IDs
func (ln *ListenableNodeTyped[S]) GetListenerIDs() []string {
	ln.mutex.RLock()
	defer ln.mutex.RUnlock()

	ids := make([]string, len(ln.listeners))
	for i, wrapper := range ln.listeners {
		ids[i] = wrapper.id
	}
	return ids
}

// ListenableStateGraphTyped extends StateGraphTyped with listener capabilities
type ListenableStateGraphTyped[S any] struct {
	*StateGraphTyped[S]
	listenableNodes map[string]*ListenableNodeTyped[S]
}

// NewListenableStateGraphTyped creates a new typed state graph with listener support
func NewListenableStateGraphTyped[S any]() *ListenableStateGraphTyped[S] {
	return &ListenableStateGraphTyped[S]{
		StateGraphTyped: NewStateGraphTyped[S](),
		listenableNodes: make(map[string]*ListenableNodeTyped[S]),
	}
}

// AddNode adds a node with listener capabilities
func (g *ListenableStateGraphTyped[S]) AddNode(name string, description string, fn func(ctx context.Context, state S) (S, error)) *ListenableNodeTyped[S] {
	node := NodeTyped[S]{
		Name:        name,
		Description: description,
		Function:    fn,
	}

	listenableNode := NewListenableNodeTyped(node)

	// Add to both the base graph and our listenable nodes map
	g.StateGraphTyped.AddNode(name, description, fn)
	g.listenableNodes[name] = listenableNode

	return listenableNode
}

// GetListenableNode returns the listenable node by name
func (g *ListenableStateGraphTyped[S]) GetListenableNode(name string) *ListenableNodeTyped[S] {
	return g.listenableNodes[name]
}

// AddGlobalListener adds a listener to all nodes in the graph
func (g *ListenableStateGraphTyped[S]) AddGlobalListener(listener NodeListenerTyped[S]) {
	for _, node := range g.listenableNodes {
		node.AddListener(listener)
	}
}

// RemoveGlobalListener removes a listener from all nodes in the graph by function reference
func (g *ListenableStateGraphTyped[S]) RemoveGlobalListener(listener NodeListenerTyped[S]) {
	for _, node := range g.listenableNodes {
		node.RemoveListenerByFunc(listener)
	}
}

// RemoveGlobalListenerByID removes a listener from all nodes in the graph by ID
func (g *ListenableStateGraphTyped[S]) RemoveGlobalListenerByID(listenerID string) {
	for _, node := range g.listenableNodes {
		node.RemoveListener(listenerID)
	}
}

// ListenableRunnableTyped wraps a StateRunnableTyped with listener capabilities
type ListenableRunnableTyped[S any] struct {
	graph           *ListenableStateGraphTyped[S]
	listenableNodes map[string]*ListenableNodeTyped[S]
	runnable        *StateRunnableTyped[S]
}

// CompileListenable creates a runnable with listener support
func (g *ListenableStateGraphTyped[S]) CompileListenable() (*ListenableRunnableTyped[S], error) {
	if g.entryPoint == "" {
		return nil, ErrEntryPointNotSet
	}

	runnable, err := g.StateGraphTyped.Compile()
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

	return &ListenableRunnableTyped[S]{
		graph:           g,
		listenableNodes: g.listenableNodes,
		runnable:        runnable,
	}, nil
}

// Invoke executes the graph with listener notifications
func (lr *ListenableRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error) {
	return lr.runnable.Invoke(ctx, initialState)
}

// InvokeWithConfig executes the graph with listener notifications and config
func (lr *ListenableRunnableTyped[S]) InvokeWithConfig(ctx context.Context, initialState S, config *Config) (S, error) {
	if config != nil {
		ctx = WithConfig(ctx, config)
	}
	return lr.runnable.InvokeWithConfig(ctx, initialState, config)
}

// Stream executes the graph with listener notifications and streams events
func (lr *ListenableRunnableTyped[S]) Stream(ctx context.Context, initialState S) <-chan StreamEventTyped[S] {
	eventChan := make(chan StreamEventTyped[S], 100) // Buffered channel

	// Create a streaming listener
	streamListener := &StreamingListenerTyped[S]{
		eventChan: eventChan,
	}

	// Add the listener to all nodes
	lr.graph.AddGlobalListener(streamListener)

	// Start execution in a goroutine
	go func() {
		defer close(eventChan)

		// Send chain start event
		eventChan <- StreamEventTyped[S]{
			Timestamp: time.Now(),
			Event:     EventChainStart,
			State:     initialState,
		}

		// Execute the graph
		_, err := lr.runnable.Invoke(ctx, initialState)

		// Send chain end event
		eventChan <- StreamEventTyped[S]{
			Timestamp: time.Now(),
			Event:     EventChainEnd,
			State:     initialState, // Note: This should be the final state, but we need to capture it
			Error:     err,
		}

		// Remove the listener
		lr.graph.RemoveGlobalListener(streamListener)
	}()

	return eventChan
}

// SetTracer sets a tracer for the underlying runnable
func (lr *ListenableRunnableTyped[S]) SetTracer(tracer *Tracer) {
	lr.runnable.SetTracer(tracer)
}

// WithTracer returns a new ListenableRunnableTyped with the given tracer
func (lr *ListenableRunnableTyped[S]) WithTracer(tracer *Tracer) *ListenableRunnableTyped[S] {
	newRunnable := lr.runnable.WithTracer(tracer)
	return &ListenableRunnableTyped[S]{
		graph:           lr.graph,
		listenableNodes: lr.listenableNodes,
		runnable:        newRunnable,
	}
}

// GetGraph returns an Exporter for visualization
func (lr *ListenableRunnableTyped[S]) GetGraph() *Exporter {
	// For now, return nil as typed graphs cannot be directly exported
	// TODO: Implement typed graph visualization
	return nil
}

// StreamingListenerTyped is a listener that streams node events
type StreamingListenerTyped[S any] struct {
	eventChan chan<- StreamEventTyped[S]
}

// OnNodeEvent implements the NodeListenerTyped interface
func (sl *StreamingListenerTyped[S]) OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error) {
	streamEvent := StreamEventTyped[S]{
		Timestamp: time.Now(),
		NodeName:  nodeName,
		Event:     event,
		State:     state,
		Error:     err,
		Metadata:  make(map[string]any),
	}

	// Send the event if channel is not closed
	select {
	case sl.eventChan <- streamEvent:
	default:
		// Channel is full or closed, drop the event
	}
}