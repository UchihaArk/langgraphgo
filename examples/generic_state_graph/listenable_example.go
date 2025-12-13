package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smallnest/langgraphgo/graph"
)

// CounterState represents the state for our counter example
type CounterState struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
	Logs  []string `json:"logs"`
}

// EventLogger is a typed listener that logs events
type EventLogger struct{}

func (l *EventLogger) OnNodeEvent(ctx context.Context, event graph.NodeEvent, nodeName string, state CounterState, err error) {
	timestamp := time.Now().Format("15:04:05")
	switch event {
	case graph.NodeEventStart:
		fmt.Printf("[%s] 🔵 Node '%s' started (count=%d)\n", timestamp, nodeName, state.Count)
	case graph.NodeEventComplete:
		fmt.Printf("[%s] 🟢 Node '%s' completed (count=%d)\n", timestamp, nodeName, state.Count)
	case graph.NodeEventError:
		fmt.Printf("[%s] 🔴 Node '%s' failed: %v\n", timestamp, nodeName, err)
	case graph.EventChainStart:
		fmt.Printf("[%s] 🚀 Graph execution started\n", timestamp)
	case graph.EventChainEnd:
		fmt.Printf("[%s] 🏁 Graph execution completed\n", timestamp)
	}
}

// ProgressTracker tracks progress and emits progress events
type ProgressTracker struct {
	totalNodes int
	completed  int
}

func (p *ProgressTracker) OnNodeEvent(ctx context.Context, event graph.NodeEvent, nodeName string, state CounterState, err error) {
	switch event {
	case graph.NodeEventStart:
		// Count total nodes (simplified)
		if nodeName == "increment" {
			p.totalNodes = state.Count + 5 // Assuming we'll run 5 increments
		}
	case graph.NodeEventComplete:
		p.completed++
		progress := float64(p.completed) / float64(p.totalNodes) * 100
		fmt.Printf("📊 Progress: %.1f%% (%d/%d)\n", progress, p.completed, p.totalNodes)
	}
}

func main() {
	// Create a typed listenable state graph
	workflow := graph.NewListenableStateGraphTyped[CounterState]()

	// Add global listeners
	logger := &EventLogger{}
	tracker := &ProgressTracker{}
	workflow.AddGlobalListener(logger)
	workflow.AddGlobalListener(tracker)

	// Add increment node with a specific listener
	incrementNode := workflow.AddNode("increment", "Increment counter",
		func(ctx context.Context, state CounterState) (CounterState, error) {
			state.Count++
			logMsg := fmt.Sprintf("Incremented count to %d", state.Count)
			state.Logs = append(state.Logs, logMsg)

			// Simulate some work
			time.Sleep(500 * time.Millisecond)

			return state, nil
		})

	// Add a listener specifically to the increment node
	incrementListener := graph.NodeListenerTypedFunc[CounterState](
		func(ctx context.Context, event graph.NodeEvent, nodeName string, state CounterState, err error) {
			if event == graph.NodeEventComplete {
				fmt.Printf("  ✨ Special notification: Count is now %d!\n", state.Count)
			}
		},
	)
	incrementNode.AddListener(incrementListener)

	// Add check node
	workflow.AddNode("check", "Check if done",
		func(ctx context.Context, state CounterState) (CounterState, error) {
			if state.Count >= 5 {
				state.Logs = append(state.Logs, "Target reached!")
			}
			return state, nil
		})

	// Add print node
	workflow.AddNode("print", "Print result",
		func(ctx context.Context, state CounterState) (CounterState, error) {
			fmt.Printf("\n📋 Final Result:\n")
			fmt.Printf("  Name: %s\n", state.Name)
			fmt.Printf("  Count: %d\n", state.Count)
			fmt.Printf("  Logs: %v\n", state.Logs)
			return state, nil
		})

	// Set up the graph structure
	workflow.SetEntryPoint("increment")

	// Add conditional edge from increment
	workflow.AddConditionalEdge("increment",
		func(ctx context.Context, state CounterState) string {
			if state.Count >= 5 {
				return "print"
			}
			return "increment" // Loop back to increment
		})

	workflow.AddEdge("print", graph.END)

	// Compile the listenable graph
	fmt.Println("🔧 Compiling the graph...")
	runnable, err := workflow.CompileListenable()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	// Execute the graph
	fmt.Println("\n🚀 Starting graph execution...")
	initialState := CounterState{
		Count: 0,
		Name:  "TypedCounter",
		Logs:  []string{},
	}

	finalState, err := runnable.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatalf("Graph execution failed: %v", err)
	}

	fmt.Printf("\n✅ Execution completed successfully!\n")
	fmt.Printf("Final state: %+v\n", finalState)

	// Demonstrate streaming
	fmt.Println("\n\n--- Streaming Example ---")
	streamingRunnable, err := workflow.CompileListenable()
	if err != nil {
		log.Fatalf("Failed to compile streaming graph: %v", err)
	}

	// Create a streaming listener
	streamingListener := &StreamingCounterListener{}
	streamingRunnable.graph.AddGlobalListener(streamingListener)

	// Execute with streaming
	fmt.Println("🎬 Starting streaming execution...")
	eventChan := streamingRunnable.Stream(context.Background(), CounterState{
		Count: 0,
		Name:  "StreamingCounter",
		Logs:  []string{},
	})

	// Process events
	fmt.Println("📡 Receiving events:")
	for event := range eventChan {
		switch event.Event {
		case graph.EventChainStart:
			fmt.Printf("[%s] 🟢 Stream: Chain started\n", event.Timestamp.Format("15:04:05.000"))
		case graph.NodeEventStart:
			fmt.Printf("[%s] 🔵 Stream: Node '%s' started\n",
				event.Timestamp.Format("15:04:05.000"), event.NodeName)
		case graph.NodeEventComplete:
			fmt.Printf("[%s] 🟢 Stream: Node '%s' completed (count=%d)\n",
				event.Timestamp.Format("15:04:05.000"), event.NodeName, event.State.Count)
		case graph.EventChainEnd:
			fmt.Printf("[%s] 🔴 Stream: Chain ended\n", event.Timestamp.Format("15:04:05.000"))
		}
	}
}

// StreamingCounterListener is a listener for streaming events
type StreamingCounterListener struct{}

func (l *StreamingCounterListener) OnNodeEvent(ctx context.Context, event graph.NodeEvent, nodeName string, state CounterState, err error) {
	// This listener will be used with the streaming functionality
	// The actual streaming is handled by the StreamingListenerTyped
}