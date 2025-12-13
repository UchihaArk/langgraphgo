package prebuilt

import (
	"context"
	"testing"
)

// MockThoughtState for testing
type MockThoughtState struct {
	id      string
	valid   bool
	isGoal  bool
	desc    string
	hashVal string
}

func (m *MockThoughtState) IsValid() bool {
	return m.valid
}

func (m *MockThoughtState) IsGoal() bool {
	return m.isGoal
}

func (m *MockThoughtState) GetDescription() string {
	return m.desc
}

func (m *MockThoughtState) Hash() string {
	return m.hashVal
}

// MockGenerator for testing
type MockGenerator struct {
	nextStates []ThoughtState
}

func (g *MockGenerator) Generate(ctx context.Context, current ThoughtState) ([]ThoughtState, error) {
	return g.nextStates, nil
}

// MockEvaluator for testing
type MockEvaluator struct {
	score float64
}

func (e *MockEvaluator) Evaluate(ctx context.Context, state ThoughtState, pathLength int) (float64, error) {
	if !state.IsValid() {
		return -1, nil
	}
	return e.score, nil
}

func TestCreateTreeOfThoughtsAgent(t *testing.T) {
	initialState := &MockThoughtState{
		id:      "start",
		valid:   true,
		isGoal:  false,
		desc:    "Start state",
		hashVal: "start",
	}

	config := TreeOfThoughtsConfig{
		Generator:    &MockGenerator{},
		Evaluator:    &MockEvaluator{score: 1.0},
		InitialState: initialState,
		MaxDepth:     5,
		MaxPaths:     3,
		Verbose:      false,
	}

	agent, err := CreateTreeOfThoughtsAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent is nil")
	}
}

func TestTreeOfThoughtsRequiresGenerator(t *testing.T) {
	config := TreeOfThoughtsConfig{
		Evaluator:    &MockEvaluator{},
		InitialState: &MockThoughtState{},
	}

	_, err := CreateTreeOfThoughtsAgent(config)
	if err == nil {
		t.Fatal("Expected error when generator is nil")
	}
}

func TestTreeOfThoughtsRequiresEvaluator(t *testing.T) {
	config := TreeOfThoughtsConfig{
		Generator:    &MockGenerator{},
		InitialState: &MockThoughtState{},
	}

	_, err := CreateTreeOfThoughtsAgent(config)
	if err == nil {
		t.Fatal("Expected error when evaluator is nil")
	}
}

func TestTreeOfThoughtsRequiresInitialState(t *testing.T) {
	config := TreeOfThoughtsConfig{
		Generator: &MockGenerator{},
		Evaluator: &MockEvaluator{},
	}

	_, err := CreateTreeOfThoughtsAgent(config)
	if err == nil {
		t.Fatal("Expected error when initial state is nil")
	}
}

func TestTreeOfThoughtsFindsGoal(t *testing.T) {
	startState := &MockThoughtState{
		id:      "start",
		valid:   true,
		isGoal:  false,
		desc:    "Start",
		hashVal: "start",
	}

	goalState := &MockThoughtState{
		id:      "goal",
		valid:   true,
		isGoal:  true,
		desc:    "Goal",
		hashVal: "goal",
	}

	generator := &MockGenerator{
		nextStates: []ThoughtState{goalState},
	}

	config := TreeOfThoughtsConfig{
		Generator:    generator,
		Evaluator:    &MockEvaluator{score: 1.0},
		InitialState: startState,
		MaxDepth:     5,
		MaxPaths:     3,
		Verbose:      false,
	}

	agent, err := CreateTreeOfThoughtsAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	result, err := agent.Invoke(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	finalState := result.(map[string]any)
	solution, ok := finalState["solution"].(SearchPath)
	if !ok || solution.States == nil {
		t.Fatal("Expected to find a solution")
	}

	if len(solution.States) != 2 {
		t.Errorf("Expected path length 2, got %d", len(solution.States))
	}

	if !solution.States[1].IsGoal() {
		t.Error("Expected final state to be goal")
	}
}

func TestTreeOfThoughtsPrunesInvalidStates(t *testing.T) {
	startState := &MockThoughtState{
		id:      "start",
		valid:   true,
		isGoal:  false,
		desc:    "Start",
		hashVal: "start",
	}

	invalidState := &MockThoughtState{
		id:      "invalid",
		valid:   false,
		isGoal:  false,
		desc:    "Invalid",
		hashVal: "invalid",
	}

	generator := &MockGenerator{
		nextStates: []ThoughtState{invalidState},
	}

	config := TreeOfThoughtsConfig{
		Generator:    generator,
		Evaluator:    &MockEvaluator{score: 1.0},
		InitialState: startState,
		MaxDepth:     2,
		MaxPaths:     3,
		Verbose:      false,
	}

	agent, err := CreateTreeOfThoughtsAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	result, err := agent.Invoke(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	finalState := result.(map[string]any)
	solution := finalState["solution"]

	if solution != nil {
		t.Error("Expected no solution when only invalid states available")
	}
}

func TestSearchPathBasics(t *testing.T) {
	state1 := &MockThoughtState{id: "1", desc: "State 1"}
	state2 := &MockThoughtState{id: "2", desc: "State 2"}

	path := SearchPath{
		States: []ThoughtState{state1, state2},
		Score:  0.5,
	}

	if len(path.States) != 2 {
		t.Errorf("Expected 2 states, got %d", len(path.States))
	}

	if path.Score != 0.5 {
		t.Errorf("Expected score 0.5, got %f", path.Score)
	}
}

func TestDefaultMaxDepth(t *testing.T) {
	config := TreeOfThoughtsConfig{
		Generator:    &MockGenerator{},
		Evaluator:    &MockEvaluator{},
		InitialState: &MockThoughtState{valid: true},
		// MaxDepth not set
	}

	agent, err := CreateTreeOfThoughtsAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent is nil")
	}
}

func TestDefaultMaxPaths(t *testing.T) {
	config := TreeOfThoughtsConfig{
		Generator:    &MockGenerator{},
		Evaluator:    &MockEvaluator{},
		InitialState: &MockThoughtState{valid: true},
		// MaxPaths not set
	}

	agent, err := CreateTreeOfThoughtsAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent is nil")
	}
}
