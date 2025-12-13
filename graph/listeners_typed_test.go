package graph

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestListenerState for testing
type TestListenerState struct {
	Name  string
	Count int
	Step  string
}

func TestNodeListenerTypedFunc_OnNodeEvent(t *testing.T) {
	var events []string
	listener := NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {
			events = append(events, string(event)+":"+nodeName)
		},
	)

	ctx := context.Background()
	state := TestListenerState{Name: "test", Count: 1}

	listener.OnNodeEvent(ctx, NodeEventStart, "node1", state, nil)
	listener.OnNodeEvent(ctx, NodeEventComplete, "node1", state, nil)

	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	if events[0] != "start:node1" {
		t.Errorf("Expected 'start:node1', got '%s'", events[0])
	}

	if events[1] != "complete:node1" {
		t.Errorf("Expected 'complete:node1', got '%s'", events[1])
	}
}

func TestNewListenableNodeTyped(t *testing.T) {
	node := NodeTyped[TestListenerState]{
		Name:        "test-node",
		Description: "Test node",
		Function: func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
			return state, nil
		},
	}

	ln := NewListenableNodeTyped(node)

	if ln.Name != "test-node" {
		t.Errorf("Expected name to be 'test-node', got '%s'", ln.Name)
	}

	if ln.Description != "Test node" {
		t.Errorf("Expected description to be 'Test node', got '%s'", ln.Description)
	}

	if ln.Function == nil {
		t.Error("Function should not be nil")
	}

	if ln.listeners == nil {
		t.Error("Listeners slice should be initialized")
	}
}

func TestListenableNodeTyped_AddListener(t *testing.T) {
	ln := NewListenableNodeTyped(NodeTyped[TestListenerState]{
		Name: "test",
		Function: func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
			return state, nil
		},
	})

	initialCount := len(ln.GetListeners())

	listener1 := NodeListenerTypedFunc[TestListenerState](func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {})
	listener2 := NodeListenerTypedFunc[TestListenerState](func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {})

	// Test AddListener returns the node for chaining
	result := ln.AddListener(listener1)
	if result != ln {
		t.Error("AddListener should return the listenable node for chaining")
	}

	ln.AddListener(listener2)

	listeners := ln.GetListeners()
	if len(listeners) != initialCount+2 {
		t.Errorf("Expected %d listeners, got %d", initialCount+2, len(listeners))
	}
}

func TestListenableNodeTyped_RemoveListener(t *testing.T) {
	listener := NodeListenerTypedFunc[TestListenerState](func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {})

	ln := NewListenableNodeTyped(NodeTyped[TestListenerState]{
		Name: "test",
		Function: func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
			return state, nil
		},
	})

	// Add listener and get its ID
	id := ln.AddListenerWithID(listener)

	// Verify listener was added
	if len(ln.GetListeners()) != 1 {
		t.Error("Listener should have been added")
	}

	// Remove the listener by ID
	ln.RemoveListener(id)

	// Verify listener was removed
	if len(ln.GetListeners()) != 0 {
		t.Error("Listener should have been removed")
	}
}

func TestListenableNodeTyped_RemoveListenerByFunc(t *testing.T) {
	listener := NodeListenerTypedFunc[TestListenerState](func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {})

	ln := NewListenableNodeTyped(NodeTyped[TestListenerState]{
		Name: "test",
		Function: func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
			return state, nil
		},
	})

	ln.AddListener(listener)

	// Verify listener was added
	if len(ln.GetListeners()) != 1 {
		t.Error("Listener should have been added")
	}

	// Remove the listener by function reference
	ln.RemoveListenerByFunc(listener)

	// Verify listener was removed
	if len(ln.GetListeners()) != 0 {
		t.Error("Listener should have been removed")
	}
}

func TestListenableNodeTyped_NotifyListeners(t *testing.T) {
	var mu sync.Mutex
	var events []string

	listener1 := NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, "listener1:"+string(event))
		},
	)

	listener2 := NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, "listener2:"+string(event))
		},
	)

	// Listener that panics
	panicListener := NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {
			panic("test panic")
		},
	)

	ln := NewListenableNodeTyped(NodeTyped[TestListenerState]{
		Name: "test-node",
		Function: func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
			return state, nil
		},
	})

	ln.AddListener(listener1)
	ln.AddListener(listener2)
	ln.AddListener(panicListener)

	ctx := context.Background()
	state := TestListenerState{Name: "test"}

	// Notify listeners - should not panic even if one listener panics
	ln.NotifyListeners(ctx, NodeEventStart, state, nil)

	// Wait for all goroutines to complete
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have 2 events (panic listener should not affect others but won't add an event)
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d: %v", len(events), events)
	}
}

func TestListenableNodeTyped_Execute(t *testing.T) {
	var events []string

	ln := NewListenableNodeTyped(NodeTyped[TestListenerState]{
		Name: "test-node",
		Function: func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
			state.Count++
			state.Step = "processed"
			return state, nil
		},
	})

	ln.AddListener(NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {
			events = append(events, string(event))
		},
	))

	ctx := context.Background()
	initialState := TestListenerState{Name: "test", Count: 0}

	result, err := ln.Execute(ctx, initialState)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check the function was executed
	if result.Count != 1 {
		t.Errorf("Expected count to be 1, got %d", result.Count)
	}

	if result.Step != "processed" {
		t.Errorf("Expected step to be 'processed', got '%s'", result.Step)
	}

	// Check events were notified
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 events, got %d", len(events))
	}

	if events[0] != string(NodeEventStart) {
		t.Errorf("Expected first event to be '%s', got '%s'", NodeEventStart, events[0])
	}

	if events[len(events)-1] != string(NodeEventComplete) {
		t.Errorf("Expected last event to be '%s', got '%s'", NodeEventComplete, events[len(events)-1])
	}
}

func TestListenableNodeTyped_Execute_Error(t *testing.T) {
	expectedError := errors.New("test error")

	ln := NewListenableNodeTyped(NodeTyped[TestListenerState]{
		Name: "test-node",
		Function: func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
			return state, expectedError
		},
	})

	var errorEvent NodeEvent
	var actualError error

	ln.AddListener(NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {
			if event == NodeEventError {
				errorEvent = event
				actualError = err
			}
		},
	))

	ctx := context.Background()
	initialState := TestListenerState{Name: "test"}

	_, err := ln.Execute(ctx, initialState)

	if err != expectedError {
		t.Errorf("Expected error '%v', got '%v'", expectedError, err)
	}

	if errorEvent != NodeEventError {
		t.Errorf("Expected error event '%s', got '%s'", NodeEventError, errorEvent)
	}

	if actualError != expectedError {
		t.Errorf("Expected error '%v' in listener, got '%v'", expectedError, actualError)
	}
}

func TestNewListenableStateGraphTyped(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	if g == nil {
		t.Fatal("Graph should not be nil")
	}

	if g.StateGraphTyped == nil {
		t.Error("StateGraphTyped should not be nil")
	}

	if g.listenableNodes == nil {
		t.Error("ListenableNodes map should be initialized")
	}
}

func TestListenableStateGraphTyped_AddNode(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	nodeFunc := func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		state.Count++
		return state, nil
	}

	ln := g.AddNode("test-node", "Test node", nodeFunc)

	if ln == nil {
		t.Fatal("AddNode should return a listenable node")
	}

	// Check node was added to base graph
	if _, ok := g.nodes["test-node"]; !ok {
		t.Error("Node should be added to base graph")
	}

	// Check node was added to listenable nodes map
	if g.listenableNodes["test-node"] == nil {
		t.Error("Node should be added to listenable nodes map")
	}

	// Test chaining
	ln2 := ln.AddListener(NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {},
	))

	if ln2 != ln {
		t.Error("AddListener should return the same node for chaining")
	}
}

func TestListenableStateGraphTyped_GetListenableNode(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	// Test getting non-existent node
	node := g.GetListenableNode("non-existent")
	if node != nil {
		t.Error("GetListenableNode should return nil for non-existent node")
	}

	// Add a node
	ln := g.AddNode("test-node", "Test node", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})

	// Get the node
	node = g.GetListenableNode("test-node")
	if node != ln {
		t.Error("GetListenableNode should return the correct listenable node")
	}
}

func TestListenableStateGraphTyped_AddGlobalListener(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	// Add nodes
	g.AddNode("node1", "Node 1", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})
	g.AddNode("node2", "Node 2", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})

	listener := NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {},
	)

	// Add global listener
	g.AddGlobalListener(listener)

	// Check all nodes have the listener
	for _, node := range g.listenableNodes {
		listeners := node.GetListeners()
		if len(listeners) == 0 {
			t.Error("Global listener should be added to all nodes")
		}
	}
}

func TestListenableStateGraphTyped_RemoveGlobalListener(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	// Add nodes
	g.AddNode("node1", "Node 1", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})
	g.AddNode("node2", "Node 2", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})

	listener := NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {},
	)

	// Add global listener
	g.AddGlobalListener(listener)

	// Verify listener was added to all nodes
	totalListeners := 0
	for _, node := range g.listenableNodes {
		totalListeners += len(node.GetListeners())
	}
	if totalListeners != 2 {
		t.Errorf("Expected 2 total listeners, got %d", totalListeners)
	}

	// Remove global listener
	g.RemoveGlobalListener(listener)

	// Verify listeners were removed from all nodes
	totalListeners = 0
	for _, node := range g.listenableNodes {
		totalListeners += len(node.GetListeners())
	}
	if totalListeners != 0 {
		t.Errorf("Expected 0 total listeners after removal, got %d", totalListeners)
	}
}

func TestListenableStateGraphTyped_RemoveGlobalListenerByID(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	// Add nodes
	g.AddNode("node1", "Node 1", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})

	listener := NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {},
	)

	// Add listener to node and get its ID
	id := g.GetListenableNode("node1").AddListenerWithID(listener)

	// Verify listener was added
	if len(g.GetListenableNode("node1").GetListeners()) != 1 {
		t.Error("Listener should be added to node1")
	}

	// Remove the listener by ID from all nodes
	g.RemoveGlobalListenerByID(id)

	// Verify listener was removed
	if len(g.GetListenableNode("node1").GetListeners()) != 0 {
		t.Error("Listener should be removed from node1")
	}
}

func TestListenableStateGraphTyped_CompileListenable(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	// Try to compile without entry point
	_, err := g.CompileListenable()
	if err == nil {
		t.Error("Compile should fail without entry point")
	}

	// Add entry point and node
	g.SetEntryPoint("start")
	g.AddNode("start", "Start node", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		state.Step = "started"
		return state, nil
	})
	g.AddEdge("start", END)

	// Compile successfully
	runnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if runnable == nil {
		t.Fatal("Compile should return a runnable")
	}

	if runnable.graph != g {
		t.Error("Runnable should reference the graph")
	}
}

func TestListenableStateGraphTyped_Invoke(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	var events []string

	// Add node with listener
	ln := g.AddNode("process", "Process", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		state.Count++
		state.Step = "processed"
		return state, nil
	})

	ln.AddListener(NodeListenerTypedFunc[TestListenerState](
		func(ctx context.Context, event NodeEvent, nodeName string, state TestListenerState, err error) {
			events = append(events, string(event)+":"+nodeName)
		},
	))

	g.SetEntryPoint("process")
	g.AddEdge("process", END)

	runnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	ctx := context.Background()
	initialState := TestListenerState{Name: "test", Count: 0}

	result, err := runnable.Invoke(ctx, initialState)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	// Check result
	if result.Count != 1 {
		t.Errorf("Expected count to be 1, got %d", result.Count)
	}

	if result.Step != "processed" {
		t.Errorf("Expected step to be 'processed', got '%s'", result.Step)
	}

	// Check events were notified
	if len(events) < 2 {
		t.Error("Should have received start and complete events")
	}
}

func TestListenableRunnableTyped_InvokeWithConfig(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	g.AddNode("process", "Process", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		state.Count++
		return state, nil
	})

	g.SetEntryPoint("process")
	g.AddEdge("process", END)

	runnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	ctx := context.Background()
	config := &Config{
		Tags: []string{"test"},
	}

	initialState := TestListenerState{Name: "test", Count: 0}

	result, err := runnable.InvokeWithConfig(ctx, initialState, config)

	if err != nil {
		t.Fatalf("InvokeWithConfig failed: %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Expected count to be 1, got %d", result.Count)
	}
}

func TestListenableRunnableTyped_SetTracer(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	g.AddNode("test", "Test", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})

	g.SetEntryPoint("test")
	g.AddEdge("test", END)

	runnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	tracer := &Tracer{}
	runnable.SetTracer(tracer)

	if runnable.runnable.tracer != tracer {
		t.Error("SetTracer should set the tracer on the underlying runnable")
	}
}

func TestListenableRunnableTyped_WithTracer(t *testing.T) {
	g := NewListenableStateGraphTyped[TestListenerState]()

	g.AddNode("test", "Test", func(ctx context.Context, state TestListenerState) (TestListenerState, error) {
		return state, nil
	})

	g.SetEntryPoint("test")
	g.AddEdge("test", END)

	runnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	tracer := &Tracer{}
	newRunnable := runnable.WithTracer(tracer)

	if newRunnable == runnable {
		t.Error("WithTracer should return a new runnable")
	}

	if newRunnable.graph != runnable.graph {
		t.Error("New runnable should reference the same graph")
	}
}

func TestStreamingListenerTyped_OnNodeEvent(t *testing.T) {
	eventChan := make(chan StreamEventTyped[TestListenerState], 10)
	defer close(eventChan)

	listener := &StreamingListenerTyped[TestListenerState]{
		eventChan: eventChan,
	}

	ctx := context.Background()
	state := TestListenerState{Name: "test", Count: 1}

	// Send event
	listener.OnNodeEvent(ctx, NodeEventStart, "test-node", state, nil)

	// Receive event
	select {
	case event := <-eventChan:
		if event.Event != NodeEventStart {
			t.Errorf("Expected event %s, got %s", NodeEventStart, event.Event)
		}
		if event.NodeName != "test-node" {
			t.Errorf("Expected node name 'test-node', got '%s'", event.NodeName)
		}
		if event.State.Name != "test" {
			t.Errorf("Expected state name 'test', got '%s'", event.State.Name)
		}
	default:
		t.Error("Should have received an event")
	}
}