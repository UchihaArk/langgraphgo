package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/smallnest/langgraphgo/graph"
)

// --- Simple File-based Checkpoint Store for Demo ---

type DiskStore struct {
	FilePath string
}

func NewDiskStore(path string) *DiskStore {
	return &DiskStore{FilePath: path}
}

func (s *DiskStore) loadAll() map[string]*graph.Checkpoint {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		return make(map[string]*graph.Checkpoint)
	}
	var checkpoints map[string]*graph.Checkpoint
	if err := json.Unmarshal(data, &checkpoints); err != nil {
		return make(map[string]*graph.Checkpoint)
	}
	return checkpoints
}

func (s *DiskStore) saveAll(cps map[string]*graph.Checkpoint) error {
	data, err := json.MarshalIndent(cps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.FilePath, data, 0644)
}

func (s *DiskStore) Save(ctx context.Context, cp *graph.Checkpoint) error {
	cps := s.loadAll()
	cps[cp.ID] = cp
	return s.saveAll(cps)
}

func (s *DiskStore) Load(ctx context.Context, id string) (*graph.Checkpoint, error) {
	cps := s.loadAll()
	if cp, ok := cps[id]; ok {
		return cp, nil
	}
	return nil, fmt.Errorf("checkpoint not found")
}

func (s *DiskStore) List(ctx context.Context, threadID string) ([]*graph.Checkpoint, error) {
	cps := s.loadAll()
	var result []*graph.Checkpoint
	for _, cp := range cps {
		// Check metadata for thread_id
		if tid, ok := cp.Metadata["thread_id"].(string); ok && tid == threadID {
			result = append(result, cp)
		}
	}
	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})
	return result, nil
}

func (s *DiskStore) Delete(ctx context.Context, id string) error {
	cps := s.loadAll()
	delete(cps, id)
	return s.saveAll(cps)
}

func (s *DiskStore) Clear(ctx context.Context, threadID string) error {
	cps := s.loadAll()
	for id, cp := range cps {
		if tid, ok := cp.Metadata["thread_id"].(string); ok && tid == threadID {
			delete(cps, id)
		}
	}
	return s.saveAll(cps)
}

// --- Main Logic ---

func main() {
	storeFile := "checkpoints.json"
	store := NewDiskStore(storeFile)
	threadID := "durable-job-1"

	// 1. Define Graph
	g := graph.NewCheckpointableStateGraph()
	// Use MapSchema for state
	schema := graph.NewMapSchema()
	schema.RegisterReducer("steps", graph.AppendReducer)
	g.SetSchema(schema)

	// Configure Checkpointing
	g.SetCheckpointConfig(graph.CheckpointConfig{
		Store:    store,
		AutoSave: true,
	})

	// Step 1
	g.AddNode("step_1", "step_1", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing Step 1...")
		time.Sleep(500 * time.Millisecond)
		return map[string]any{"steps": []string{"Step 1 Completed"}}, nil
	})

	// Step 2 (Simulate Crash)
	g.AddNode("step_2", "step_2", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing Step 2...")
		time.Sleep(500 * time.Millisecond)

		// Check if we should crash
		if os.Getenv("CRASH") == "true" {
			fmt.Println("!!! CRASHING AT STEP 2 !!!")
			fmt.Println("(Run again without CRASH=true to recover)")
			os.Exit(1)
		}

		return map[string]any{"steps": []string{"Step 2 Completed"}}, nil
	})

	// Step 3
	g.AddNode("step_3", "step_3", func(ctx context.Context, state any) (any, error) {
		fmt.Println("Executing Step 3...")
		time.Sleep(500 * time.Millisecond)
		return map[string]any{"steps": []string{"Step 3 Completed"}}, nil
	})

	g.SetEntryPoint("step_1")
	g.AddEdge("step_1", "step_2")
	g.AddEdge("step_2", "step_3")
	g.AddEdge("step_3", graph.END)

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Check for existing checkpoints to resume
	ctx := context.Background()
	checkpoints, _ := store.List(ctx, threadID)

	var config *graph.Config
	if len(checkpoints) > 0 {
		latest := checkpoints[len(checkpoints)-1]
		fmt.Printf("Found existing checkpoint: %s (Node: %s)\n", latest.ID, latest.NodeName)
		fmt.Println("Resuming execution...")

		// In a real Durable Execution setup, we would want to resume FROM the next node.
		// Currently ResumeFromCheckpoint just loads state.
		// We need to tell the runner to start from the *next* node of the checkpoint.
		// Or, if the checkpoint was saved *after* a node completed, we continue from its edges.

		// For this demo, we'll use the state to determine where we are,
		// but ideally the framework handles this.
		// Since LangGraphGo's ResumeFromCheckpoint is basic, we might need to manually inspect.

		// However, let's try to use the ResumeFrom config if we implemented it.
		// graph.Config has ResumeFrom []string.

		// If latest checkpoint is from "step_1", we want to resume.
		// But wait, if we crashed INSIDE step_2, step_2 didn't finish, so no checkpoint for step_2.
		// So the latest checkpoint is "step_1".
		// So we should resume from the state of "step_1".
		// The graph execution logic should see that "step_1" is done and move to "step_2".

		// To achieve this "continue" behavior, we pass the checkpoint ID.
		// The runner *should* know that if we provide a checkpoint, we are continuing.
		// But currently Invoke() starts from EntryPoint unless ResumeFrom is set.

		// Let's manually set ResumeFrom based on the checkpoint's node.
		// If checkpoint is at "step_1", we want to execute the edges out of "step_1".
		// Which means the next node is "step_2".

		// Actually, if we load the state of "step_1", and we invoke the graph,
		// we want the graph to figure out "what's next".
		// But the graph is stateless.

		// Let's use a trick: We pass the checkpoint state as initial state,
		// and we set ResumeFrom to the *next* nodes.
		// But we need to know the next nodes.

		// For this simple linear graph:
		var nextNode string
		if latest.NodeName == "step_1" {
			nextNode = "step_2"
		} else if latest.NodeName == "step_2" {
			nextNode = "step_3"
		} else {
			// Finished or unknown
			fmt.Println("Job already finished or unknown state.")
			return
		}

		config = &graph.Config{
			Configurable: map[string]any{
				"thread_id":     threadID,
				"checkpoint_id": latest.ID,
			},
			ResumeFrom: []string{nextNode},
		}

		// We use the checkpoint state as input
		// runnable.Invoke(ctx, latest.State, config)
		// Wait, Invoke expects input, not full state?
		// If we provide ResumeFrom, the runner uses that as start nodes.
		// And it uses the provided state.

		fmt.Printf("Continuing from %s...\n", nextNode)
		res, err := runnable.InvokeWithConfig(ctx, latest.State, config)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Final Result: %v\n", res)

	} else {
		fmt.Println("Starting new execution...")
		config = &graph.Config{
			Configurable: map[string]any{
				"thread_id": threadID,
			},
		}
		initialState := map[string]any{"steps": []string{"Start"}}
		res, err := runnable.InvokeWithConfig(ctx, initialState, config)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Final Result: %v\n", res)
	}
}
