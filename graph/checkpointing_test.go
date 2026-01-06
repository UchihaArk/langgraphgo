package graph_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	st "github.com/smallnest/langgraphgo/store"
	"github.com/smallnest/langgraphgo/store/redis"
)

func TestMemoryCheckpointStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	ctx := context.Background()

	checkpoint := &st.Checkpoint{
		ID:        "test_checkpoint_1",
		NodeName:  testNode,
		State:     "test_state",
		Timestamp: time.Now(),
		Version:   1,
		Metadata: map[string]any{
			"execution_id": "exec_123",
		},
	}

	// Test Save
	err := store.Save(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Test Load
	loaded, err := store.Load(ctx, "test_checkpoint_1")
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	if loaded.ID != checkpoint.ID {
		t.Errorf("Expected ID %s, got %s", checkpoint.ID, loaded.ID)
	}

	if loaded.NodeName != checkpoint.NodeName {
		t.Errorf("Expected NodeName %s, got %s", checkpoint.NodeName, loaded.NodeName)
	}

	if loaded.State != checkpoint.State {
		t.Errorf("Expected State %v, got %v", checkpoint.State, loaded.State)
	}
}

func TestMemoryCheckpointStore_LoadNonExistent(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	ctx := context.Background()

	_, err := store.Load(ctx, "non_existent")
	if err == nil {
		t.Error("Expected error for non-existent checkpoint")
	}

	if !strings.Contains(err.Error(), "checkpoint not found") {
		t.Errorf("Expected 'checkpoint not found' error, got: %v", err)
	}
}

func TestMemoryCheckpointStore_List(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	ctx := context.Background()
	executionID := "exec_123"

	// Save multiple checkpoints
	checkpoints := []*st.Checkpoint{
		{
			ID: "checkpoint_1",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
		},
		{
			ID: "checkpoint_2",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
		},
		{
			ID: "checkpoint_3",
			Metadata: map[string]any{
				"execution_id": "different_exec",
			},
		},
	}

	for _, checkpoint := range checkpoints {
		err := store.Save(ctx, checkpoint)
		if err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// List checkpoints for specific execution
	listed, err := store.List(ctx, executionID)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("Expected 2 checkpoints, got %d", len(listed))
	}

	// Verify correct checkpoints returned
	ids := make(map[string]bool)
	for _, checkpoint := range listed {
		ids[checkpoint.ID] = true
	}

	if !ids["checkpoint_1"] || !ids["checkpoint_2"] {
		t.Error("Wrong checkpoints returned")
	}
}

func TestMemoryCheckpointStore_Delete(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	ctx := context.Background()

	checkpoint := &st.Checkpoint{
		ID: "test_checkpoint",
	}

	// Save and verify
	err := store.Save(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	_, err = store.Load(ctx, "test_checkpoint")
	if err != nil {
		t.Error("Checkpoint should exist before deletion")
	}

	// Delete
	err = store.Delete(ctx, "test_checkpoint")
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verify deletion
	_, err = store.Load(ctx, "test_checkpoint")
	if err == nil {
		t.Error("Checkpoint should not exist after deletion")
	}
}

func TestMemoryCheckpointStore_Clear(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	ctx := context.Background()
	executionID := "exec_123"

	// Save checkpoints
	checkpoints := []*st.Checkpoint{
		{
			ID: "checkpoint_1",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
		},
		{
			ID: "checkpoint_2",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
		},
		{
			ID: "checkpoint_3",
			Metadata: map[string]any{
				"execution_id": "different_exec",
			},
		},
	}

	for _, checkpoint := range checkpoints {
		err := store.Save(ctx, checkpoint)
		if err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// Clear execution
	err := store.Clear(ctx, executionID)
	if err != nil {
		t.Fatalf("Failed to clear checkpoints: %v", err)
	}

	// Verify clearing
	listed, err := store.List(ctx, executionID)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(listed) != 0 {
		t.Errorf("Expected 0 checkpoints after clear, got %d", len(listed))
	}

	// Verify other execution's checkpoints still exist
	listed, err = store.List(ctx, "different_exec")
	if err != nil {
		t.Fatalf("Failed to list other execution's checkpoints: %v", err)
	}

	if len(listed) != 1 {
		t.Errorf("Expected 1 checkpoint for other execution, got %d", len(listed))
	}
}

func TestFileCheckpointStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := graph.NewFileCheckpointStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file checkpoint store: %v", err)
	}
	ctx := context.Background()

	checkpoint := &st.Checkpoint{
		ID:       "test_checkpoint",
		NodeName: testNode,
		State:    "test_state",
		Version:  1,
	}

	// Test Save
	err = store.Save(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Test Load
	loaded, err := store.Load(ctx, "test_checkpoint")
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	if loaded.ID != checkpoint.ID {
		t.Errorf("Expected ID %s, got %s", checkpoint.ID, loaded.ID)
	}
}

func TestFileCheckpointStore_List(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := graph.NewFileCheckpointStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file checkpoint store: %v", err)
	}
	ctx := context.Background()
	executionID := "exec_123"

	// Save multiple checkpoints
	checkpoints := []*st.Checkpoint{
		{
			ID: "checkpoint_1",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
			Version: 1,
		},
		{
			ID: "checkpoint_2",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
			Version: 2,
		},
		{
			ID: "checkpoint_3",
			Metadata: map[string]any{
				"execution_id": "different_exec",
			},
			Version: 1,
		},
	}

	for _, checkpoint := range checkpoints {
		err := store.Save(ctx, checkpoint)
		if err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// List checkpoints for specific execution
	listed, err := store.List(ctx, executionID)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("Expected 2 checkpoints, got %d", len(listed))
	}

	// Verify correct checkpoints returned
	ids := make(map[string]bool)
	for _, checkpoint := range listed {
		ids[checkpoint.ID] = true
	}

	if !ids["checkpoint_1"] || !ids["checkpoint_2"] {
		t.Error("Wrong checkpoints returned")
	}

	// Verify sorting order
	if listed[0].Version > listed[1].Version {
		t.Error("Checkpoints should be sorted by version ascending")
	}
}

func TestFileCheckpointStore_Delete(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := graph.NewFileCheckpointStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file checkpoint store: %v", err)
	}
	ctx := context.Background()

	checkpoint := &st.Checkpoint{
		ID: "test_checkpoint",
	}

	// Save and verify
	err = store.Save(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	_, err = store.Load(ctx, "test_checkpoint")
	if err != nil {
		t.Error("Checkpoint should exist before deletion")
	}

	// Delete
	err = store.Delete(ctx, "test_checkpoint")
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verify deletion
	_, err = store.Load(ctx, "test_checkpoint")
	if err == nil {
		t.Error("Checkpoint should not exist after deletion")
	}
}

func TestFileCheckpointStore_Clear(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	store, err := graph.NewFileCheckpointStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file checkpoint store: %v", err)
	}
	ctx := context.Background()
	executionID := "exec_123"

	// Save checkpoints
	checkpoints := []*st.Checkpoint{
		{
			ID: "checkpoint_1",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
		},
		{
			ID: "checkpoint_2",
			Metadata: map[string]any{
				"execution_id": executionID,
			},
		},
		{
			ID: "checkpoint_3",
			Metadata: map[string]any{
				"execution_id": "different_exec",
			},
		},
	}

	for _, checkpoint := range checkpoints {
		err := store.Save(ctx, checkpoint)
		if err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}
	}

	// Clear execution
	err = store.Clear(ctx, executionID)
	if err != nil {
		t.Fatalf("Failed to clear checkpoints: %v", err)
	}

	// Verify clearing
	listed, err := store.List(ctx, executionID)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(listed) != 0 {
		t.Errorf("Expected 0 checkpoints after clear, got %d", len(listed))
	}

	// Verify other execution's checkpoints still exist
	listed, err = store.List(ctx, "different_exec")
	if err != nil {
		t.Fatalf("Failed to list other execution's checkpoints: %v", err)
	}

	if len(listed) != 1 {
		t.Errorf("Expected 1 checkpoint for other execution, got %d", len(listed))
	}
}

func TestCheckpointableRunnable_Basic(t *testing.T) {
	t.Parallel()

	// Create graph
	g := graph.NewListenableStateGraph[map[string]any]()

	g.AddNode("step1", "step1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{"result": "step1_result"}, nil
	})

	g.AddNode("step2", "step2", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{"result": "step2_result"}, nil
	})

	g.AddEdge("step1", "step2")
	g.AddEdge("step2", graph.END)
	g.SetEntryPoint("step1")

	// Compile listenable runnable
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Failed to compile listenable runnable: %v", err)
	}

	// Create checkpointable runnable
	config := graph.DefaultCheckpointConfig()
	checkpointableRunnable := graph.NewCheckpointableRunnable(listenableRunnable, config)

	ctx := context.Background()
	result, err := checkpointableRunnable.Invoke(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	resultMap := result
	if resultMap["result"] != "step2_result" {
		t.Errorf("Expected 'step2_result', got %v", resultMap)
	}

	// Wait for async checkpoint operations
	time.Sleep(100 * time.Millisecond)

	// Check that checkpoints were created
	checkpoints, err := checkpointableRunnable.ListCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 2 {
		t.Errorf("Expected 2 checkpoints (one per completed node), got %d", len(checkpoints))
	}

	// Verify checkpoint contents
	nodeNames := make(map[string]bool)
	for _, checkpoint := range checkpoints {
		nodeNames[checkpoint.NodeName] = true
		if checkpoint.State == nil {
			t.Error("Checkpoint state should not be nil")
		}
		if checkpoint.Timestamp.IsZero() {
			t.Error("Checkpoint timestamp should be set")
		}
	}

	if !nodeNames["step1"] || !nodeNames["step2"] {
		t.Error("Expected checkpoints for both step1 and step2")
	}
}

func TestCheckpointableRunnable_ManualCheckpoint(t *testing.T) {
	t.Parallel()

	g := graph.NewListenableStateGraph[map[string]any]()
	g.AddNode(testNode, testNode, func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{"result": testResult}, nil
	})
	g.AddEdge(testNode, graph.END)
	g.SetEntryPoint(testNode)

	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	config := graph.DefaultCheckpointConfig()
	checkpointableRunnable := graph.NewCheckpointableRunnable(listenableRunnable, config)

	ctx := context.Background()

	// Manual checkpoint save
	err = checkpointableRunnable.SaveCheckpoint(ctx, testNode, map[string]any{"result": "manual_state"})
	if err != nil {
		t.Fatalf("Failed to save manual checkpoint: %v", err)
	}

	checkpoints, err := checkpointableRunnable.ListCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Errorf("Expected 1 checkpoint, got %d", len(checkpoints))
	}

	checkpoint := checkpoints[0]
	if checkpoint.NodeName != testNode {
		t.Errorf("Expected node name 'test_node', got %s", checkpoint.NodeName)
	}

	stateMap, ok := checkpoint.State.(map[string]any)
	if !ok {
		t.Fatalf("Expected state to be map[string]any, got %T", checkpoint.State)
	}
	if stateMap["result"] != "manual_state" {
		t.Errorf("Expected state 'manual_state', got %v", checkpoint.State)
	}
}

func TestCheckpointableRunnable_LoadCheckpoint(t *testing.T) {
	t.Parallel()

	g := graph.NewListenableStateGraph[map[string]any]()
	g.AddNode(testNode, testNode, func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{"result": testResult}, nil
	})
	g.AddEdge(testNode, graph.END)
	g.SetEntryPoint(testNode)

	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	config := graph.DefaultCheckpointConfig()
	checkpointableRunnable := graph.NewCheckpointableRunnable(listenableRunnable, config)

	ctx := context.Background()

	// Save checkpoint
	err = checkpointableRunnable.SaveCheckpoint(ctx, testNode, map[string]any{"result": "saved_state"})
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	checkpoints, err := checkpointableRunnable.ListCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) == 0 {
		t.Fatal("No checkpoints found")
	}

	checkpointID := checkpoints[0].ID

	// Load checkpoint
	loaded, err := checkpointableRunnable.LoadCheckpoint(ctx, checkpointID)
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	stateMap, ok := loaded.State.(map[string]any)
	if !ok {
		t.Fatalf("Expected loaded state to be map[string]any, got %T", loaded.State)
	}
	if stateMap["result"] != "saved_state" {
		t.Errorf("Expected loaded state 'saved_state', got %v", loaded.State)
	}
}

func TestCheckpointableRunnable_ClearCheckpoints(t *testing.T) {
	t.Parallel()

	g := graph.NewListenableStateGraph[map[string]any]()
	g.AddNode(testNode, testNode, func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{"result": testResult}, nil
	})
	g.AddEdge(testNode, graph.END)
	g.SetEntryPoint(testNode)

	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	config := graph.DefaultCheckpointConfig()
	checkpointableRunnable := graph.NewCheckpointableRunnable(listenableRunnable, config)

	ctx := context.Background()

	// Save some checkpoints
	err = checkpointableRunnable.SaveCheckpoint(ctx, "test_node1", map[string]any{"result": "state1"})
	if err != nil {
		t.Fatalf("Failed to save checkpoint 1: %v", err)
	}

	err = checkpointableRunnable.SaveCheckpoint(ctx, "test_node2", map[string]any{"result": "state2"})
	if err != nil {
		t.Fatalf("Failed to save checkpoint 2: %v", err)
	}

	// Verify checkpoints exist
	checkpoints, err := checkpointableRunnable.ListCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 2 {
		t.Errorf("Expected 2 checkpoints, got %d", len(checkpoints))
	}

	// Clear checkpoints
	err = checkpointableRunnable.ClearCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to clear checkpoints: %v", err)
	}

	// Verify checkpoints cleared
	checkpoints, err = checkpointableRunnable.ListCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to list checkpoints after clear: %v", err)
	}

	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints after clear, got %d", len(checkpoints))
	}
}

func TestCheckpointableStateGraph_CompileCheckpointable(t *testing.T) {
	t.Parallel()

	g := graph.NewCheckpointableStateGraph[map[string]any]()

	g.AddNode(testNode, testNode, func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{"result": testResult}, nil
	})
	g.AddEdge(testNode, graph.END)
	g.SetEntryPoint(testNode)

	checkpointableRunnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile checkpointable: %v", err)
	}

	ctx := context.Background()
	result, err := checkpointableRunnable.Invoke(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	resultMap := result
	if resultMap["result"] != testResult {
		t.Errorf("Expected 'test_result', got %v", resultMap)
	}
}

func TestCheckpointableStateGraph_CustomConfig(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	config := graph.CheckpointConfig{
		Store:          store,
		AutoSave:       false,
		SaveInterval:   time.Minute,
		MaxCheckpoints: 5,
	}

	g := graph.NewCheckpointableStateGraphWithConfig[map[string]any](config)

	// Verify config is set
	actualConfig := g.GetCheckpointConfig()
	if actualConfig.AutoSave != false {
		t.Error("Expected AutoSave to be false")
	}

	if actualConfig.SaveInterval != time.Minute {
		t.Error("Expected SaveInterval to be 1 minute")
	}

	if actualConfig.MaxCheckpoints != 5 {
		t.Error("Expected MaxCheckpoints to be 5")
	}
}

// Integration test with comprehensive workflow
//
//nolint:gocognit,cyclop // Comprehensive integration test requires multiple scenarios
func TestCheckpointing_Integration(t *testing.T) {
	t.Parallel()

	// Create checkpointable graph
	g := graph.NewCheckpointableStateGraph[map[string]any]()

	// Build a multi-step pipeline
	g.AddNode("analyze", "analyze", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["analyzed"] = true
		return state, nil
	})

	g.AddNode("process", "process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["processed"] = true
		return state, nil
	})

	g.AddNode("finalize", "finalize", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		state["finalized"] = true
		return state, nil
	})

	g.AddEdge("analyze", "process")
	g.AddEdge("process", "finalize")
	g.AddEdge("finalize", graph.END)
	g.SetEntryPoint("analyze")

	// Compile checkpointable runnable
	runnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Execute with initial state
	initialState := map[string]any{
		"input": "test_data",
	}

	ctx := context.Background()
	result, err := runnable.Invoke(ctx, initialState)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Verify final result
	finalState := result
	if !finalState["analyzed"].(bool) {
		t.Error("Expected analyzed to be true")
	}
	if !finalState["processed"].(bool) {
		t.Error("Expected processed to be true")
	}
	if !finalState["finalized"].(bool) {
		t.Error("Expected finalized to be true")
	}

	// Wait for async checkpoint operations
	time.Sleep(100 * time.Millisecond)

	// Check checkpoints
	checkpoints, err := runnable.ListCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 3 {
		t.Errorf("Expected 3 checkpoints, got %d", len(checkpoints))
	}

	// Verify each checkpoint has the correct state progression
	checkpointsByNode := make(map[string]*st.Checkpoint)
	for _, checkpoint := range checkpoints {
		checkpointsByNode[checkpoint.NodeName] = checkpoint
	}

	fmt.Printf("checkpoints: %+v\n", checkpointsByNode)

	// Check analyze checkpoint
	if analyzeCP, exists := checkpointsByNode["analyze"]; exists {
		state := analyzeCP.State.(map[string]any)
		if !state["analyzed"].(bool) {
			t.Error("Analyze checkpoint should have analyzed=true")
		}
	} else {
		t.Error("Missing checkpoint for analyze node")
	}

	// Check process checkpoint
	if processCP, exists := checkpointsByNode["process"]; exists {
		state := processCP.State.(map[string]any)
		if !state["processed"].(bool) {
			t.Error("Process checkpoint should have processed=true")
		}
	} else {
		t.Error("Missing checkpoint for process node")
	}

	// Check finalize checkpoint
	if finalizeCP, exists := checkpointsByNode["finalize"]; exists {
		state := finalizeCP.State.(map[string]any)
		if !state["finalized"].(bool) {
			t.Error("Finalize checkpoint should have finalized=true")
		}
	} else {
		t.Error("Missing checkpoint for finalize node")
	}
}

func TestCheckpointListener_ErrorHandling(t *testing.T) {
	t.Parallel()

	g := graph.NewListenableStateGraph[map[string]any]()

	// Node that will fail
	g.AddNode("failing_node", "failing_node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return nil, fmt.Errorf("simulated failure")
	})

	g.AddEdge("failing_node", graph.END)
	g.SetEntryPoint("failing_node")

	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	config := graph.DefaultCheckpointConfig()
	checkpointableRunnable := graph.NewCheckpointableRunnable(listenableRunnable, config)

	ctx := context.Background()

	// This should fail
	_, err = checkpointableRunnable.Invoke(ctx, map[string]any{})
	if err == nil {
		t.Error("Expected execution to fail")
	}

	// Wait for async operations
	time.Sleep(100 * time.Millisecond)

	// Should not have checkpoints for failed nodes
	checkpoints, err := checkpointableRunnable.ListCheckpoints(ctx)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	// Should have no checkpoints since node failed
	if len(checkpoints) != 0 {
		t.Errorf("Expected no checkpoints for failed execution, got %d", len(checkpoints))
	}
}

// TestAutoResume_WithThreadID tests automatic resume using thread_id
// This matches the Python LangGraph behavior where providing thread_id
// automatically loads and merges the checkpoint state with new input.
func TestAutoResume_WithThreadID(t *testing.T) {
	t.Parallel()

	g := graph.NewCheckpointableStateGraph[map[string]any]()

	// Track execution order
	executionOrder := []string{}
	g.AddNode("step1", "step1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		executionOrder = append(executionOrder, "step1")
		state["step1"] = "done"
		return state, nil
	})

	g.AddNode("step2", "step2", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		executionOrder = append(executionOrder, "step2")
		state["step2"] = "done"
		return state, nil
	})

	g.AddNode("step3", "step3", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		executionOrder = append(executionOrder, "step3")
		state["step3"] = "done"
		return state, nil
	})

	g.AddEdge("step1", "step2")
	g.AddEdge("step2", "step3")
	g.AddEdge("step3", graph.END)
	g.SetEntryPoint("step1")

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	runnable.SetExecutionID("test_auto_resume") // Use consistent execution ID

	ctx := context.Background()
	threadID := "test-thread-auto-resume"

	// First execution - create checkpoint
	result1, err := runnable.InvokeWithConfig(ctx, map[string]any{"input": "first"}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("First execution failed: %v", err)
	}

	// Wait for checkpoint save
	time.Sleep(100 * time.Millisecond)

	// Verify first execution completed all steps
	if result1["step1"] != "done" || result1["step2"] != "done" || result1["step3"] != "done" {
		t.Errorf("First execution incomplete: %v", result1)
	}

	// Second execution - demonstrate checkpoint state loading
	// When using thread_id with an existing checkpoint:
	// - The checkpoint state is loaded and merged with new input
	// - ResumeFrom is set to continue from where it left off
	// Reset execution tracker
	executionOrder = []string{}

	// Note: For completed graphs, you can manually GetState to retrieve the final state
	// This is the recommended pattern for "continuing" a completed conversation
	snapshot, err := runnable.GetState(ctx, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	// Verify we can retrieve the checkpoint state
	if snapshot == nil {
		t.Fatal("Expected non-nil state snapshot")
	}

	t.Logf("Retrieved state snapshot: %+v", snapshot.Values)

	// For a new continuation with additional input, create a new graph execution
	// with the checkpoint state as base
	result2, err := runnable.InvokeWithConfig(ctx, map[string]any{"input": "second"}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("Second execution failed: %v", err)
	}

	// The second execution loads checkpoint state and continues
	t.Logf("Second execution ran nodes: %v", executionOrder)
	// Note: step3 runs again because ResumeFrom is set to the checkpoint node
	// This is expected behavior for continuation
	if result2["input"] != "second" {
		t.Errorf("Input should be 'second', got: %v", result2["input"])
	}
}

// TestAutoResume_InterruptAndResume tests the interrupt and resume workflow
// with automatic state loading.
func TestAutoResume_InterruptAndResume(t *testing.T) {
	t.Parallel()

	g := graph.NewCheckpointableStateGraph[map[string]any]()

	executionCount := map[string]int{}
	g.AddNode("step1", "step1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		executionCount["step1"]++
		state["step1"] = "done"
		return state, nil
	})

	g.AddNode("step2", "step2", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		executionCount["step2"]++
		state["step2"] = "done"
		return state, nil
	})

	g.AddNode("step3", "step3", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		executionCount["step3"]++
		state["step3"] = "done"
		return state, nil
	})

	g.AddEdge("step1", "step2")
	g.AddEdge("step2", "step3")
	g.AddEdge("step3", graph.END)
	g.SetEntryPoint("step1")

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}
	runnable.SetExecutionID("test_interrupt_resume")

	ctx := context.Background()
	threadID := "test-thread-interrupt"

	// Phase 1: Run with interrupt after step2
	config1 := graph.WithThreadID(threadID)
	config1.InterruptAfter = []string{"step2"}

	result1, err := runnable.InvokeWithConfig(ctx, map[string]any{"input": "phase1"}, config1)
	// InterruptAfter returns a GraphInterrupt error, which is expected
	if err != nil {
		if _, ok := err.(*graph.GraphInterrupt); !ok {
			t.Fatalf("Phase 1 unexpected error: %v", err)
		}
		// For GraphInterrupt, the result is still valid (contains state at interrupt point)
	}

	// Wait for checkpoint save
	time.Sleep(100 * time.Millisecond)

	// Verify phase1 stopped after step2
	if result1["step1"] != "done" {
		t.Error("step1 should be done")
	}
	if result1["step2"] != "done" {
		t.Error("step2 should be done")
	}
	if result1["step3"] != nil {
		t.Error("step3 should not be done (interrupted)")
	}

	if executionCount["step1"] != 1 || executionCount["step2"] != 1 || executionCount["step3"] != 0 {
		t.Errorf("Unexpected execution counts: %v", executionCount)
	}

	// Phase 2: Resume with just thread_id - should auto-load state and continue
	// Use WithThreadID for simplicity
	// Note: After ResumeFrom step2, step2 and step3 will execute again
	// The checkpoint state is loaded and merged, but ResumeFrom causes re-execution
	result2, err := runnable.InvokeWithConfig(ctx, map[string]any{"input": "phase2"}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("Phase 2 execution failed: %v", err)
	}

	// Verify phase2 completed step3
	if result2["step3"] != "done" {
		t.Errorf("Phase 2 should complete step3: %v", result2)
	}
	if result2["input"] != "phase2" {
		t.Errorf("Input should be 'phase2', got: %v", result2["input"])
	}

	// Step3 should have run
	if executionCount["step3"] < 1 {
		t.Errorf("step3 should run at least once, ran %d times", executionCount["step3"])
	}
}

// TestAutoResume_MergeStates tests that state merging works correctly
// when resuming with new input.
func TestAutoResume_MergeStates(t *testing.T) {
	t.Parallel()

	g := graph.NewCheckpointableStateGraph[map[string]any]()

	g.AddNode("process", "process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		// Initialize messages slice if not exists
		if state["messages"] == nil {
			state["messages"] = []string{}
		}
		messages := state["messages"].([]string)
		messages = append(messages, state["input"].(string))
		state["messages"] = messages
		return state, nil
	})

	g.AddEdge("process", graph.END)
	g.SetEntryPoint("process")

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	ctx := context.Background()
	threadID := "test-thread-merge"

	// First call
	result1, err := runnable.InvokeWithConfig(ctx, map[string]any{"input": "hello"}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	messages1 := result1["messages"].([]string)
	if len(messages1) != 1 || messages1[0] != "hello" {
		t.Errorf("Unexpected messages after first call: %v", messages1)
	}

	// Second call - should merge state (in this implementation, input replaces)
	// For proper append behavior, a reducer would be needed
	result2, err := runnable.InvokeWithConfig(ctx, map[string]any{"input": "world"}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	// The state should be preserved across calls
	messages2 := result2["messages"].([]string)
	if len(messages2) == 0 {
		t.Errorf("Messages should be preserved: %v", messages2)
	}
}

// TestWithThreadID tests the WithThreadID helper function
func TestWithThreadID(t *testing.T) {
	t.Parallel()

	config := graph.WithThreadID("test-thread-123")

	if config == nil {
		t.Fatal("WithThreadID should return non-nil config")
	}

	if config.Configurable == nil {
		t.Fatal("Configurable should not be nil")
	}

	threadID, ok := config.Configurable["thread_id"].(string)
	if !ok {
		t.Fatal("thread_id should be a string")
	}

	if threadID != "test-thread-123" {
		t.Errorf("Expected thread_id 'test-thread-123', got '%s'", threadID)
	}
}

// TestWithInterruptBeforeAfter tests the helper functions for interrupt configuration
func TestWithInterruptBeforeAfter(t *testing.T) {
	t.Parallel()

	// Test WithInterruptBefore
	config1 := graph.WithInterruptBefore("node1", "node2")
	if len(config1.InterruptBefore) != 2 {
		t.Errorf("Expected 2 interrupt-before nodes, got %d", len(config1.InterruptBefore))
	}
	if config1.InterruptBefore[0] != "node1" || config1.InterruptBefore[1] != "node2" {
		t.Error("InterruptBefore nodes not set correctly")
	}

	// Test WithInterruptAfter
	config2 := graph.WithInterruptAfter("node3")
	if len(config2.InterruptAfter) != 1 {
		t.Errorf("Expected 1 interrupt-after node, got %d", len(config2.InterruptAfter))
	}
	if config2.InterruptAfter[0] != "node3" {
		t.Error("InterruptAfter node not set correctly")
	}
}

// TestMaxCheckpoints_AutoCleanup tests that MaxCheckpoints automatically
// removes old checkpoints when the limit is exceeded.
func TestMaxCheckpoints_AutoCleanup(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	g := graph.NewCheckpointableStateGraph[map[string]any]()

	// Add multiple nodes to create multiple checkpoints
	for i := 1; i <= 5; i++ {
		nodeName := fmt.Sprintf("step%d", i)
		g.AddNode(nodeName, nodeName, func(ctx context.Context, state map[string]any) (map[string]any, error) {
			state[nodeName] = "done"
			return state, nil
		})

		if i > 1 {
			prevNode := fmt.Sprintf("step%d", i-1)
			g.AddEdge(prevNode, nodeName)
		}
	}
	g.AddEdge("step5", graph.END)
	g.SetEntryPoint("step1")

	// Set MaxCheckpoints to 3
	config := graph.CheckpointConfig{
		Store:          store,
		AutoSave:       true,
		MaxCheckpoints: 3,
	}
	g.SetCheckpointConfig(config)

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	ctx := context.Background()
	threadID := "test-max-checkpoints"

	// Execute the graph with thread_id
	_, err = runnable.InvokeWithConfig(ctx, map[string]any{"input": "test"}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Wait for async checkpoint operations
	time.Sleep(100 * time.Millisecond)

	// Check that only 3 checkpoints remain (most recent ones)
	checkpoints, err := store.List(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 3 {
		t.Errorf("Expected 3 checkpoints after MaxCheckpoints cleanup, got %d", len(checkpoints))
	}

	// Verify that the remaining checkpoints are the most recent ones (step3, step4, step5)
	nodeNames := make([]string, 0, len(checkpoints))
	for _, cp := range checkpoints {
		nodeNames = append(nodeNames, cp.NodeName)
	}

	// Should contain step3, step4, step5 (the 3 most recent)
	expectedNodes := []string{"step3", "step4", "step5"}
	for _, expected := range expectedNodes {
		found := slices.Contains(nodeNames, expected)
		if !found {
			t.Errorf("Expected checkpoint for %s, but only found: %v", expected, nodeNames)
		}
	}
}

// TestMaxCheckpoints_ZeroOrNegative tests that MaxCheckpoints <= 0 means no limit
func TestMaxCheckpoints_ZeroOrNegative(t *testing.T) {
	t.Parallel()

	store := graph.NewMemoryCheckpointStore()
	g := graph.NewCheckpointableStateGraph[map[string]any]()

	// Add 3 nodes
	for i := 1; i <= 3; i++ {
		nodeName := fmt.Sprintf("step%d", i)
		g.AddNode(nodeName, nodeName, func(ctx context.Context, state map[string]any) (map[string]any, error) {
			state[nodeName] = "done"
			return state, nil
		})

		if i > 1 {
			prevNode := fmt.Sprintf("step%d", i-1)
			g.AddEdge(prevNode, nodeName)
		}
	}
	g.AddEdge("step3", graph.END)
	g.SetEntryPoint("step1")

	// Set MaxCheckpoints to 0 (no limit)
	config := graph.CheckpointConfig{
		Store:          store,
		AutoSave:       true,
		MaxCheckpoints: 0,
	}
	g.SetCheckpointConfig(config)

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	ctx := context.Background()
	threadID := "test-no-limit"

	_, err = runnable.InvokeWithConfig(ctx, map[string]any{}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// All checkpoints should be preserved (no limit)
	checkpoints, err := store.List(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 3 {
		t.Errorf("Expected 3 checkpoints with MaxCheckpoints=0 (no limit), got %d", len(checkpoints))
	}
}

func TestMaxCheckpoints_AutoCleanup_RedisStore(t *testing.T) {
	t.Parallel()

	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		Prefix:   "example_checkpoints:",
		TTL:      1 * time.Hour,
	})
	g := graph.NewCheckpointableStateGraph[map[string]any]()

	// Add multiple nodes to create multiple checkpoints
	for i := 1; i <= 5; i++ {
		nodeName := fmt.Sprintf("step%d", i)
		g.AddNode(nodeName, nodeName, func(ctx context.Context, state map[string]any) (map[string]any, error) {
			state[nodeName] = "done"
			return state, nil
		})

		if i > 1 {
			prevNode := fmt.Sprintf("step%d", i-1)
			g.AddEdge(prevNode, nodeName)
		}
	}
	g.AddEdge("step5", graph.END)
	g.SetEntryPoint("step1")

	// Set MaxCheckpoints to 3
	config := graph.CheckpointConfig{
		Store:          store,
		AutoSave:       true,
		MaxCheckpoints: 3,
	}
	g.SetCheckpointConfig(config)

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	ctx := context.Background()
	threadID := "test-max-checkpoints"

	// Execute the graph with thread_id
	_, err = runnable.InvokeWithConfig(ctx, map[string]any{"input": "test"}, graph.WithThreadID(threadID))
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Wait for async checkpoint operations
	time.Sleep(100 * time.Millisecond)

	// Check that only 3 checkpoints remain (most recent ones)
	checkpoints, err := store.ListByThread(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 3 {
		t.Errorf("Expected 3 checkpoints after MaxCheckpoints cleanup, got %d", len(checkpoints))
	}

	// Verify that the remaining checkpoints are the most recent ones (step3, step4, step5)
	nodeNames := make([]string, 0, len(checkpoints))
	for _, cp := range checkpoints {
		nodeNames = append(nodeNames, cp.NodeName)
	}

	// Should contain step3, step4, step5 (the 3 most recent)
	expectedNodes := []string{"step3", "step4", "step5"}
	for _, expected := range expectedNodes {
		found := slices.Contains(nodeNames, expected)
		if !found {
			t.Errorf("Expected checkpoint for %s, but only found: %v", expected, nodeNames)
		}
	}
	latestByThread, err := store.GetLatestByThread(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to get latest checkpoint by thread: %v", err)
	}
	if latestByThread == nil || latestByThread.NodeName != "step5" {
		t.Errorf("Expected latest checkpoint by thread to be step5")
	}
}
