package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/smallnest/langgraphgo/prebuilt"
)

// RiverState represents the state of the wolf-goat-cabbage river crossing puzzle
type RiverState struct {
	LeftBank     map[string]bool // Items on the left bank
	RightBank    map[string]bool // Items on the right bank
	BoatLocation string          // "left" or "right"
	LastMove     string          // Description of the last move
}

// NewRiverState creates a new river state
func NewRiverState(left, right map[string]bool, boatLoc, lastMove string) *RiverState {
	return &RiverState{
		LeftBank:     left,
		RightBank:    right,
		BoatLocation: boatLoc,
		LastMove:     lastMove,
	}
}

// IsValid checks if the state is valid (no conflicts)
func (s *RiverState) IsValid() bool {
	// Check left bank
	if s.LeftBank["wolf"] && s.LeftBank["goat"] && !s.LeftBank["farmer"] {
		return false // Wolf eats goat
	}
	if s.LeftBank["goat"] && s.LeftBank["cabbage"] && !s.LeftBank["farmer"] {
		return false // Goat eats cabbage
	}

	// Check right bank
	if s.RightBank["wolf"] && s.RightBank["goat"] && !s.RightBank["farmer"] {
		return false // Wolf eats goat
	}
	if s.RightBank["goat"] && s.RightBank["cabbage"] && !s.RightBank["farmer"] {
		return false // Goat eats cabbage
	}

	return true
}

// IsGoal checks if all items are on the right bank
func (s *RiverState) IsGoal() bool {
	return s.RightBank["farmer"] && s.RightBank["wolf"] &&
		s.RightBank["goat"] && s.RightBank["cabbage"]
}

// GetDescription returns a human-readable description
func (s *RiverState) GetDescription() string {
	leftItems := s.bankToString(s.LeftBank)
	rightItems := s.bankToString(s.RightBank)

	desc := fmt.Sprintf("Left: [%s] | Right: [%s] | Boat: %s",
		leftItems, rightItems, s.BoatLocation)

	if s.LastMove != "" {
		desc += fmt.Sprintf(" | Move: %s", s.LastMove)
	}

	return desc
}

func (s *RiverState) bankToString(bank map[string]bool) string {
	var items []string
	if bank["farmer"] {
		items = append(items, "Farmer")
	}
	if bank["wolf"] {
		items = append(items, "Wolf")
	}
	if bank["goat"] {
		items = append(items, "Goat")
	}
	if bank["cabbage"] {
		items = append(items, "Cabbage")
	}
	return strings.Join(items, ", ")
}

// Hash returns a unique identifier for this state
func (s *RiverState) Hash() string {
	// Create a deterministic hash
	var left, right []string

	for item, present := range s.LeftBank {
		if present {
			left = append(left, item)
		}
	}
	for item, present := range s.RightBank {
		if present {
			right = append(right, item)
		}
	}

	sort.Strings(left)
	sort.Strings(right)

	return fmt.Sprintf("L:%s|R:%s|B:%s",
		strings.Join(left, ","),
		strings.Join(right, ","),
		s.BoatLocation)
}

// RiverPuzzleGenerator generates possible next states
type RiverPuzzleGenerator struct{}

func (g *RiverPuzzleGenerator) Generate(ctx context.Context, current prebuilt.ThoughtState) ([]prebuilt.ThoughtState, error) {
	state := current.(*RiverState)
	var nextStates []prebuilt.ThoughtState

	// Determine which bank the farmer is on
	var fromBank, toBank map[string]bool
	var newBoatLoc string

	if state.BoatLocation == "left" {
		fromBank = state.LeftBank
		toBank = state.RightBank
		newBoatLoc = "right"
	} else {
		fromBank = state.RightBank
		toBank = state.LeftBank
		newBoatLoc = "left"
	}

	// Farmer must be on the same side as the boat
	if !fromBank["farmer"] {
		return nextStates, nil
	}

	// Option 1: Farmer goes alone
	nextStates = append(nextStates, g.createNextState(state, fromBank, toBank, newBoatLoc, "", "Farmer crosses alone"))

	// Option 2: Farmer takes wolf
	if fromBank["wolf"] {
		nextStates = append(nextStates, g.createNextState(state, fromBank, toBank, newBoatLoc, "wolf", "Farmer takes Wolf"))
	}

	// Option 3: Farmer takes goat
	if fromBank["goat"] {
		nextStates = append(nextStates, g.createNextState(state, fromBank, toBank, newBoatLoc, "goat", "Farmer takes Goat"))
	}

	// Option 4: Farmer takes cabbage
	if fromBank["cabbage"] {
		nextStates = append(nextStates, g.createNextState(state, fromBank, toBank, newBoatLoc, "cabbage", "Farmer takes Cabbage"))
	}

	return nextStates, nil
}

func (g *RiverPuzzleGenerator) createNextState(
	current *RiverState,
	fromBank, toBank map[string]bool,
	newBoatLoc, item, moveDesc string,
) prebuilt.ThoughtState {
	// Copy banks
	newLeft := make(map[string]bool)
	newRight := make(map[string]bool)

	for k, v := range current.LeftBank {
		newLeft[k] = v
	}
	for k, v := range current.RightBank {
		newRight[k] = v
	}

	// Determine which banks to update
	var newFrom, newTo map[string]bool
	if current.BoatLocation == "left" {
		newFrom = newLeft
		newTo = newRight
	} else {
		newFrom = newRight
		newTo = newLeft
	}

	// Move farmer
	newFrom["farmer"] = false
	newTo["farmer"] = true

	// Move item if specified
	if item != "" {
		newFrom[item] = false
		newTo[item] = true
	}

	return NewRiverState(newLeft, newRight, newBoatLoc, moveDesc)
}

// SimpleEvaluator provides a simple heuristic evaluation
type SimpleEvaluator struct{}

func (e *SimpleEvaluator) Evaluate(ctx context.Context, state prebuilt.ThoughtState, pathLength int) (float64, error) {
	riverState := state.(*RiverState)

	// Prune invalid states
	if !riverState.IsValid() {
		return -1, nil
	}

	// Score based on progress: how many items are on the right bank
	score := 0.0
	if riverState.RightBank["farmer"] {
		score += 1.0
	}
	if riverState.RightBank["wolf"] {
		score += 1.0
	}
	if riverState.RightBank["goat"] {
		score += 1.0
	}
	if riverState.RightBank["cabbage"] {
		score += 1.0
	}

	// Penalize longer paths slightly to prefer shorter solutions
	score -= float64(pathLength) * 0.01

	return score, nil
}

func main() {
	fmt.Println("=== Tree of Thoughts: River Crossing Puzzle ===")
	fmt.Println()
	fmt.Println("Problem: A farmer needs to transport a wolf, a goat, and a cabbage across a river.")
	fmt.Println("Rules:")
	fmt.Println("  1. The boat can only carry the farmer and at most one other item")
	fmt.Println("  2. The wolf cannot be left alone with the goat")
	fmt.Println("  3. The goat cannot be left alone with the cabbage")
	fmt.Println()
	fmt.Println("Let's use Tree of Thoughts to find a solution!")

	// Create initial state: everything on the left bank
	initialState := NewRiverState(
		map[string]bool{"farmer": true, "wolf": true, "goat": true, "cabbage": true},
		map[string]bool{"farmer": false, "wolf": false, "goat": false, "cabbage": false},
		"left",
		"Initial state",
	)

	// Configure Tree of Thoughts
	config := prebuilt.TreeOfThoughtsConfig{
		Generator:    &RiverPuzzleGenerator{},
		Evaluator:    &SimpleEvaluator{},
		InitialState: initialState,
		MaxDepth:     10,
		MaxPaths:     10,
		Verbose:      true,
	}

	// Create agent
	agent, err := prebuilt.CreateTreeOfThoughtsAgent(config)
	if err != nil {
		log.Fatalf("Failed to create Tree of Thoughts agent: %v", err)
	}

	// Run search
	fmt.Println("🔍 Starting tree search...")
	result, err := agent.Invoke(context.Background(), map[string]any{})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	// Print solution
	fmt.Println()
	finalState := result.(map[string]any)
	solution := finalState["solution"]

	prebuilt.PrintSolution(solution)

	fmt.Println("\n=== Analysis ===")
	fmt.Println("Tree of Thoughts systematically explored the search space,")
	fmt.Println("evaluating multiple possible paths at each step and pruning")
	fmt.Println("invalid branches (where the wolf would eat the goat, or the")
	fmt.Println("goat would eat the cabbage).")
	fmt.Println()
	fmt.Println("This demonstrates how ToT can solve constraint satisfaction")
	fmt.Println("problems through intelligent search rather than relying on")
	fmt.Println("the LLM's memorized knowledge.")
}
