package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

// State represents the workflow state
type State struct {
	Input    string
	Approved bool
	Output   string
}

func main() {
	// Create a new graph
	g := graph.NewStateGraph()

	// Define nodes
	g.AddNode("process_request", "process_request", func(ctx context.Context, state any) (any, error) {
		s := state.(State)
		fmt.Printf("[Process] Processing request: %s\n", s.Input)
		s.Output = "Processed: " + s.Input
		return s, nil
	})

	g.AddNode("human_approval", "human_approval", func(ctx context.Context, state any) (any, error) {
		s := state.(State)
		if s.Approved {
			fmt.Println("[Human] Request APPROVED.")
			s.Output += " (Approved)"
		} else {
			fmt.Println("[Human] Request REJECTED.")
			s.Output += " (Rejected)"
		}
		return s, nil
	})

	g.AddNode("finalize", "finalize", func(ctx context.Context, state any) (any, error) {
		s := state.(State)
		fmt.Printf("[Finalize] Final output: %s\n", s.Output)
		return s, nil
	})

	// Define edges
	g.SetEntryPoint("process_request")
	g.AddEdge("process_request", "human_approval")
	g.AddEdge("human_approval", "finalize")
	g.AddEdge("finalize", graph.END)

	// Compile the graph
	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Initial state
	initialState := State{
		Input:    "Deploy to Production",
		Approved: false,
	}

	// 1. Run with InterruptBefore "human_approval"
	fmt.Println("=== Starting Workflow (Phase 1) ===")
	config := &graph.Config{
		InterruptBefore: []string{"human_approval"},
	}

	res, err := runnable.InvokeWithConfig(context.Background(), initialState, config)

	// We expect an interrupt error
	if err != nil {
		var interrupt *graph.GraphInterrupt
		if errors.As(err, &interrupt) {
			fmt.Printf("Workflow interrupted at node: %s\n", interrupt.Node)
			fmt.Printf("Current State: %+v\n", interrupt.State)
		} else {
			log.Fatalf("Unexpected error: %v", err)
		}
	} else {
		// If it didn't interrupt, that's unexpected for this example
		fmt.Printf("Workflow completed without interrupt: %+v\n", res)
		return
	}

	// Simulate Human Interaction
	fmt.Println("\n=== Human Interaction ===")
	fmt.Println("Reviewing request...")
	fmt.Println("Approving request...")

	// Update state to reflect approval
	// In a real app, you would fetch the saved state, modify it, and pass it back
	// Here we just modify the state we got from the interrupt
	var interrupt *graph.GraphInterrupt
	errors.As(err, &interrupt)

	currentState := interrupt.State.(State)
	currentState.Approved = true // Human approves

	// 2. Resume execution
	fmt.Println("\n=== Resuming Workflow (Phase 2) ===")
	resumeConfig := &graph.Config{
		ResumeFrom: []string{interrupt.Node}, // Resume from the interrupted node
	}

	finalRes, err := runnable.InvokeWithConfig(context.Background(), currentState, resumeConfig)
	if err != nil {
		log.Fatalf("Error resuming workflow: %v", err)
	}

	finalState := finalRes.(State)
	fmt.Printf("Workflow completed successfully.\n")
	fmt.Printf("Final Result: %+v\n", finalState)
}
