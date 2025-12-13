package graph

import (
	"context"
	"errors"
	"fmt"
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
	State any
	// NextNodes that would have been executed if not interrupted
	NextNodes []string
	// InterruptValue is the value provided by the dynamic interrupt (if any)
	InterruptValue any
}

func (e *GraphInterrupt) Error() string {
	if e.InterruptValue != nil {
		return fmt.Sprintf("graph interrupted at node %s with value: %v", e.Node, e.InterruptValue)
	}
	return fmt.Sprintf("graph interrupted at node %s", e.Node)
}

// Interrupt pauses execution and waits for input.
// If resuming, it returns the value provided in the resume command.
func Interrupt(ctx context.Context, value any) (any, error) {
	if resumeVal := GetResumeValue(ctx); resumeVal != nil {
		return resumeVal, nil
	}
	return nil, &NodeInterrupt{Value: value}
}

// Node represents a node in the graph.
type Node struct {
	// Name is the unique identifier for the node.
	Name string

	// Description describes the functionality of the node.
	Description string

	// Function is the function associated with the node.
	// It takes a context and any state as input and returns the updated state and an error.
	Function func(ctx context.Context, state any) (any, error)
}

// Edge represents an edge in the graph.
type Edge struct {
	// From is the name of the node from which the edge originates.
	From string

	// To is the name of the node to which the edge points.
	To string
}

// StateMerger merges multiple state updates into a single state.
type StateMerger func(ctx context.Context, currentState any, newStates []any) (any, error)

// Runnable is an alias for StateRunnable for backward compatibility.
// All Runnable functionality is now provided by StateRunnable.
type Runnable = StateRunnable

// NewMessageGraph creates a new instance of StateGraph with a default schema
// that handles "messages" using the AddMessages reducer.
// This is the recommended constructor for chat-based agents that use
// map[string]any as state with a "messages" key.
//
// This replaces the old NewMessageGraphWithSchema() function.
func NewMessageGraph() *StateGraph {
	g := &StateGraph{
		nodes:            make(map[string]Node),
		conditionalEdges: make(map[string]func(ctx context.Context, state any) string),
	}

	// Initialize default schema for message handling
	schema := NewMapSchema()
	schema.RegisterReducer("messages", AddMessages)
	g.Schema = schema

	return g
}
