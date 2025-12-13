package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
)

func main() {
	// Create a simple MessageGraph
	g := graph.NewStateGraph()

	// Define a node that uses dynamic interrupt
	g.AddNode("ask_name", "ask_name", func(ctx context.Context, state any) (any, error) {
		fmt.Println("-> Node 'ask_name' executing...")

		// Call Interrupt to pause execution and wait for input.
		// If a ResumeValue is present in the context (provided during resume),
		// it returns that value. Otherwise, it returns a NodeInterrupt error.
		name, err := graph.Interrupt(ctx, "What is your name?")
		if err != nil {
			return nil, err
		}

		fmt.Printf("-> Node 'ask_name' received input: %v\n", name)
		return fmt.Sprintf("Hello, %v!", name), nil
	})

	g.SetEntryPoint("ask_name")
	g.AddEdge("ask_name", graph.END)

	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// 1. Initial Run
	fmt.Println("--- 1. Initial Execution ---")
	// We pass nil as initial state
	_, err = runnable.Invoke(context.Background(), nil)

	// Check if the execution was interrupted
	var graphInterrupt *graph.GraphInterrupt
	if errors.As(err, &graphInterrupt) {
		fmt.Printf("Graph interrupted at node: %s\n", graphInterrupt.Node)
		fmt.Printf("Interrupt Query: %v\n", graphInterrupt.InterruptValue)

		// Simulate getting input from a user
		userInput := "Alice"
		fmt.Printf("\n[User Input]: %s\n", userInput)

		// 2. Resume Execution
		fmt.Println("\n--- 2. Resuming Execution ---")

		// We provide the user input as ResumeValue in the config
		config := &graph.Config{
			ResumeValue: userInput,
		}

		// Re-run the graph. The 'ask_name' node will run again,
		// but this time graph.Interrupt() will return 'userInput' immediately.
		res, err := runnable.InvokeWithConfig(context.Background(), nil, config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Final Result: %v\n", res)

	} else if err != nil {
		log.Fatalf("Execution failed: %v", err)
	} else {
		fmt.Println("Execution finished without interrupt.")
	}
}
