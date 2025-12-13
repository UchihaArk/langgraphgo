package prebuilt

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/log"
)

// ThoughtState represents a state in the search tree
type ThoughtState interface {
	// IsValid checks if the state is valid (no rule violations)
	IsValid() bool

	// IsGoal checks if this state represents a solution
	IsGoal() bool

	// GetDescription returns a human-readable description of the state
	GetDescription() string

	// Hash returns a unique hash for the state (for cycle detection)
	Hash() string
}

// ThoughtGenerator generates possible next states from a current state
type ThoughtGenerator interface {
	// Generate returns all possible next states from the current state
	Generate(ctx context.Context, current ThoughtState) ([]ThoughtState, error)
}

// ThoughtEvaluator evaluates the quality/promise of a state
type ThoughtEvaluator interface {
	// Evaluate returns a score for the state (higher is better)
	// Returns -1 if the state should be pruned
	Evaluate(ctx context.Context, state ThoughtState, pathLength int) (float64, error)
}

// SearchPath represents a path in the search tree
type SearchPath struct {
	States []ThoughtState
	Score  float64
}

// TreeOfThoughtsConfig configures the Tree of Thoughts search
type TreeOfThoughtsConfig struct {
	// Generator creates new states
	Generator ThoughtGenerator

	// Evaluator scores states
	Evaluator ThoughtEvaluator

	// MaxDepth is the maximum search depth
	MaxDepth int

	// MaxPaths is the maximum number of active paths to maintain
	MaxPaths int

	// Verbose enables detailed logging
	Verbose bool

	// InitialState is the starting state
	InitialState ThoughtState
}

// CreateTreeOfThoughtsAgent creates a Tree of Thoughts search agent
//
// Tree of Thoughts (ToT) is a search-based reasoning framework where problem-solving
// is modeled as a search through a tree. At each step, multiple candidate "thoughts"
// are generated, evaluated for feasibility, and the most promising branches are expanded
// while unpromising ones are pruned.
//
// The ToT pattern involves:
// 1. Decomposition: Break down the problem into steps
// 2. Thought Generation: Generate multiple possible next steps (branches)
// 3. State Evaluation: Evaluate each thought for validity and promise
// 4. Pruning & Expansion: Remove bad branches, expand good ones
// 5. Solution: Continue until a goal state is reached
//
// This pattern is ideal for:
// - Logic puzzles with clear rules and goal states
// - Complex planning problems with constraints
// - Problems where multiple strategies should be explored
func CreateTreeOfThoughtsAgent(config TreeOfThoughtsConfig) (*graph.StateRunnable, error) {
	if config.Generator == nil {
		return nil, fmt.Errorf("generator is required")
	}

	if config.Evaluator == nil {
		return nil, fmt.Errorf("evaluator is required")
	}

	if config.InitialState == nil {
		return nil, fmt.Errorf("initial state is required")
	}

	if config.MaxDepth == 0 {
		config.MaxDepth = 10 // Default max depth
	}

	if config.MaxPaths == 0 {
		config.MaxPaths = 5 // Default max active paths
	}

	// Create the workflow
	workflow := graph.NewStateGraph()

	// Define state schema
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("active_paths", graph.OverwriteReducer)
	agentSchema.RegisterReducer("solution", graph.OverwriteReducer)
	agentSchema.RegisterReducer("visited_states", graph.OverwriteReducer)
	agentSchema.RegisterReducer("iteration", graph.OverwriteReducer)
	workflow.SetSchema(agentSchema)

	// Add initialize node
	workflow.AddNode("initialize", "Initialize search with starting state", func(ctx context.Context, state any) (any, error) {
		return initializeNode(ctx, state, config)
	})

	// Add expand node
	workflow.AddNode("expand", "Expand active paths by generating new thoughts", func(ctx context.Context, state any) (any, error) {
		return expandNode(ctx, state, config)
	})

	// Add evaluate node
	workflow.AddNode("evaluate", "Evaluate and prune paths", func(ctx context.Context, state any) (any, error) {
		return evaluateNode(ctx, state, config)
	})

	// Set entry point
	workflow.SetEntryPoint("initialize")

	// Add edges
	workflow.AddEdge("initialize", "expand")
	workflow.AddConditionalEdge("expand", func(ctx context.Context, state any) string {
		return routeAfterExpand(state, config)
	})
	workflow.AddConditionalEdge("evaluate", func(ctx context.Context, state any) string {
		return routeAfterEvaluate(state, config)
	})

	return workflow.Compile()
}

// initializeNode sets up the initial search state
func initializeNode(ctx context.Context, state any, config TreeOfThoughtsConfig) (any, error) {
	if config.Verbose {
		log.Info("initializing Tree of Thoughts search")
		log.Info("initial state: %s\n", config.InitialState.GetDescription())
	}

	// Create initial path
	initialPath := SearchPath{
		States: []ThoughtState{config.InitialState},
		Score:  0,
	}

	visited := make(map[string]bool)
	visited[config.InitialState.Hash()] = true

	return map[string]any{
		"active_paths":   []SearchPath{initialPath},
		"solution":       nil,
		"visited_states": visited,
		"iteration":      0,
	}, nil
}

// expandNode generates new thoughts from active paths
func expandNode(ctx context.Context, state any, config TreeOfThoughtsConfig) (any, error) {
	mState := state.(map[string]any)

	activePaths, ok := mState["active_paths"].([]SearchPath)
	if !ok || len(activePaths) == 0 {
		return nil, fmt.Errorf("no active paths to expand")
	}

	visitedStates, _ := mState["visited_states"].(map[string]bool)
	iteration, _ := mState["iteration"].(int)

	if config.Verbose {
		log.Info("iteration %d: expanding %d active paths", iteration+1, len(activePaths))
	}

	var newPaths []SearchPath

	// Expand each active path
	for pathIdx, path := range activePaths {
		currentState := path.States[len(path.States)-1]

		// Check if already at goal
		if currentState.IsGoal() {
			if config.Verbose {
				log.Info("path %d reached goal!", pathIdx)
			}
			return map[string]any{
				"solution": path,
			}, nil
		}

		// Check max depth
		if len(path.States) >= config.MaxDepth {
			if config.Verbose {
				log.Warn("path %d reached max depth, skipping", pathIdx)
			}
			continue
		}

		// Generate next states
		nextStates, err := config.Generator.Generate(ctx, currentState)
		if err != nil {
			if config.Verbose {
				log.Warn("error generating next states for path %d: %v", pathIdx, err)
			}
			continue
		}

		if config.Verbose {
			log.Info("  path %d: generated %d candidate states", pathIdx, len(nextStates))
		}

		// Create new paths for each valid next state
		for _, nextState := range nextStates {
			// Skip if invalid
			if !nextState.IsValid() {
				continue
			}

			// Skip if already visited (cycle detection)
			hash := nextState.Hash()
			if visitedStates[hash] {
				continue
			}

			// Create new path
			newPathStates := make([]ThoughtState, len(path.States))
			copy(newPathStates, path.States)
			newPathStates = append(newPathStates, nextState)

			newPath := SearchPath{
				States: newPathStates,
				Score:  0, // Will be evaluated in next step
			}

			newPaths = append(newPaths, newPath)
			visitedStates[hash] = true
		}
	}

	if config.Verbose {
		log.Info("  Total new paths generated: %d\n", len(newPaths))
	}

	return map[string]any{
		"active_paths":   newPaths,
		"visited_states": visitedStates,
		"iteration":      iteration + 1,
	}, nil
}

// evaluateNode evaluates and prunes paths
func evaluateNode(ctx context.Context, state any, config TreeOfThoughtsConfig) (any, error) {
	mState := state.(map[string]any)

	activePaths, ok := mState["active_paths"].([]SearchPath)
	if !ok || len(activePaths) == 0 {
		return map[string]any{
			"active_paths": []SearchPath{},
		}, nil
	}

	if config.Verbose {
		log.Info("evaluating %d paths", len(activePaths))
	}

	// Evaluate each path
	for i := range activePaths {
		lastState := activePaths[i].States[len(activePaths[i].States)-1]
		score, err := config.Evaluator.Evaluate(ctx, lastState, len(activePaths[i].States))
		if err != nil {
			if config.Verbose {
				log.Warn("error evaluating path %d: %v", i, err)
			}
			score = -1
		}
		activePaths[i].Score = score
	}

	// Prune paths with negative scores
	var prunedPaths []SearchPath
	for _, path := range activePaths {
		if path.Score >= 0 {
			prunedPaths = append(prunedPaths, path)
		}
	}

	if config.Verbose {
		log.Info("  pruned %d paths with negative scores", len(activePaths)-len(prunedPaths))
	}

	// Keep only top MaxPaths paths
	if len(prunedPaths) > config.MaxPaths {
		// Sort by score (descending)
		for i := 0; i < len(prunedPaths)-1; i++ {
			for j := i + 1; j < len(prunedPaths); j++ {
				if prunedPaths[j].Score > prunedPaths[i].Score {
					prunedPaths[i], prunedPaths[j] = prunedPaths[j], prunedPaths[i]
				}
			}
		}
		prunedPaths = prunedPaths[:config.MaxPaths]

		if config.Verbose {
			log.Info("  kept top %d paths", config.MaxPaths)
		}
	}

	if config.Verbose {
		log.Info("  active paths remaining: %d\n", len(prunedPaths))
	}

	return map[string]any{
		"active_paths": prunedPaths,
	}, nil
}

// Routing functions

func routeAfterExpand(state any, config TreeOfThoughtsConfig) string {
	mState := state.(map[string]any)

	// Check if solution found
	if solution, ok := mState["solution"].(SearchPath); ok && solution.States != nil {
		if config.Verbose {
			log.Info("solution found!")
		}
		return graph.END
	}

	// Check if any active paths remain
	activePaths, ok := mState["active_paths"].([]SearchPath)
	if !ok || len(activePaths) == 0 {
		if config.Verbose {
			log.Error("no more paths to explore")
		}
		return graph.END
	}

	// Check iteration limit
	iteration, _ := mState["iteration"].(int)
	if iteration >= config.MaxDepth {
		if config.Verbose {
			log.Warn("reached max iterations (%d)", config.MaxDepth)
		}
		return graph.END
	}

	// Continue to evaluation
	return "evaluate"
}

func routeAfterEvaluate(state any, config TreeOfThoughtsConfig) string {
	mState := state.(map[string]any)

	// Check if any active paths remain
	activePaths, ok := mState["active_paths"].([]SearchPath)
	if !ok || len(activePaths) == 0 {
		if config.Verbose {
			log.Error("no paths remaining after pruning")
		}
		return graph.END
	}

	// Continue expanding
	return "expand"
}

// Helper function to print solution
func PrintSolution(solution any) {
	if solution == nil {
		log.Info("no solution found")
		return
	}

	path, ok := solution.(SearchPath)
	if !ok || len(path.States) == 0 {
		log.Info("no solution found")
		return
	}

	log.Info("=== solution found ===")
	log.Info("path length: %d steps\n", len(path.States)-1)

	for i, state := range path.States {
		if i == 0 {
			log.Info("start: %s", state.GetDescription())
		} else {
			log.Info("step %d: %s", i, state.GetDescription())
		}
	}
	log.Info("======================")
}
