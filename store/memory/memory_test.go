package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smallnest/langgraphgo/store"
)

func TestMemoryCheckpointStore_New(t *testing.T) {
	t.Parallel()

	ms := NewMemoryCheckpointStore()

	if ms == nil {
		t.Fatal("Store should not be nil")
	}

	// Verify it implements the interface
	var _ store.CheckpointStore = ms
}

func TestMemoryCheckpointStore_BasicOperations(t *testing.T) {
	t.Parallel()

	t.Run("save and load", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()

		cp := &store.Checkpoint{
			ID:        "user-session-123",
			NodeName:  "auth-handler",
			State:     "waiting_for_2fa",
			Timestamp: time.Now(),
			Version:   1,
			Metadata: map[string]interface{}{
				"user_id":    "alice@example.com",
				"session_id": "sess-abc-123",
				"ip_address": "10.0.0.45",
			},
		}

		// Save it
		err := ms.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		// Load it back
		loaded, err := ms.Load(ctx, cp.ID)
		if err != nil {
			t.Fatalf("Failed to load: %v", err)
		}

		// Verify everything matches
		if loaded.ID != cp.ID {
			t.Errorf("ID mismatch: got %s, want %s", loaded.ID, cp.ID)
		}
		if loaded.NodeName != cp.NodeName {
			t.Errorf("NodeName mismatch: got %s, want %s", loaded.NodeName, cp.NodeName)
		}
		if loaded.State != cp.State {
			t.Errorf("State mismatch: got %s, want %s", loaded.State, cp.State)
		}
		if loaded.Version != cp.Version {
			t.Errorf("Version mismatch: got %d, want %d", loaded.Version, cp.Version)
		}

		// Check some metadata
		if userID, ok := loaded.Metadata["user_id"].(string); !ok || userID != "alice@example.com" {
			t.Error("User ID not preserved correctly")
		}
	})

	t.Run("load missing returns error", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()

		_, err := ms.Load(ctx, "does-not-exist")
		if err == nil {
			t.Error("Expected error for missing checkpoint")
		}
	})

	t.Run("overwrite works", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()

		// Save first version
		cp1 := &store.Checkpoint{
			ID:        "overwrite-test",
			NodeName:  "processor-v1",
			State:     "initial",
			Timestamp: time.Now(),
			Version:   1,
		}
		err := ms.Save(ctx, cp1)
		if err != nil {
			t.Fatalf("Failed to save v1: %v", err)
		}

		// Save second version with same ID
		cp2 := &store.Checkpoint{
			ID:        "overwrite-test",
			NodeName:  "processor-v2",
			State:     "updated",
			Timestamp: time.Now(),
			Version:   2,
		}
		err = ms.Save(ctx, cp2)
		if err != nil {
			t.Fatalf("Failed to save v2: %v", err)
		}

		// Load and verify we get v2
		loaded, err := ms.Load(ctx, "overwrite-test")
		if err != nil {
			t.Fatalf("Failed to load: %v", err)
		}

		if loaded.NodeName != "processor-v2" {
			t.Errorf("Expected v2 processor, got %s", loaded.NodeName)
		}
		if loaded.State != "updated" {
			t.Errorf("Expected updated state, got %s", loaded.State)
		}
		if loaded.Version != 2 {
			t.Errorf("Expected version 2, got %d", loaded.Version)
		}
	})
}

func TestMemoryCheckpointStore_List(t *testing.T) {
	t.Parallel()

	t.Run("filters by session_id", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()
		userSession := "web-user-12345"

		checkpoints := []struct {
			id      string
			node    string
			version int
		}{
			{"homepage-visit", "page-renderer", 1},
			{"login-attempt", "auth-handler", 2},
			{"dashboard-view", "dashboard-renderer", 3},
		}

		for _, cp := range checkpoints {
			fullCP := &store.Checkpoint{
				ID:        cp.id,
				NodeName:  cp.node,
				State:     "success",
				Timestamp: time.Now().Add(time.Duration(cp.version) * time.Minute),
				Version:   cp.version,
				Metadata: map[string]interface{}{
					"session_id": userSession,
				},
			}
			err := ms.Save(ctx, fullCP)
			if err != nil {
				t.Fatalf("Failed to save %s: %v", cp.id, err)
			}
		}

		results, err := ms.List(ctx, userSession)
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 checkpoints for user session, got %d", len(results))
		}

		// Verify they're sorted by version
		for i := 1; i < len(results); i++ {
			if results[i-1].Version > results[i].Version {
				t.Error("Checkpoints not sorted by version")
				break
			}
		}
	})

	t.Run("filters by thread_id", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()
		botSession := "api-bot-67890"

		cp := &store.Checkpoint{
			ID:        "api-call",
			NodeName:  "request-handler",
			State:     "success",
			Timestamp: time.Now(),
			Version:   1,
			Metadata: map[string]interface{}{
				"thread_id": botSession,
			},
		}
		err := ms.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		results, err := ms.List(ctx, botSession)
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 checkpoint for bot session, got %d", len(results))
		}

		if results[0].ID != "api-call" {
			t.Errorf("Expected api-call, got %s", results[0].ID)
		}
	})

	t.Run("empty for unknown session", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()

		results, err := ms.List(ctx, "ghost-session")
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 checkpoints, got %d", len(results))
		}
	})

	t.Run("mixed session/thread filters", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()
		userSession := "web-user-12345"
		adminThread := "admin-ops-thread-1"

		// Add a checkpoint that has both
		mixedCP := &store.Checkpoint{
			ID:        "mixed-metadata",
			NodeName:  "hybrid-handler",
			State:     "processing",
			Timestamp: time.Now(),
			Version:   1,
			Metadata: map[string]interface{}{
				"session_id": userSession,
				"thread_id":  adminThread,
			},
		}
		err := ms.Save(ctx, mixedCP)
		if err != nil {
			t.Fatalf("Failed to save mixed: %v", err)
		}

		// Should appear in both session and thread lists
		sessionList, _ := ms.List(ctx, userSession)
		threadList, _ := ms.List(ctx, adminThread)

		if len(sessionList) != 1 {
			t.Errorf("Expected 1 in session list, got %d", len(sessionList))
		}
		if len(threadList) != 1 {
			t.Errorf("Expected 1 in thread list, got %d", len(threadList))
		}
	})
}

func TestMemoryCheckpointStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("delete existing", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()

		// Save a few checkpoints
		ids := []string{"keep-1", "delete-me", "keep-2"}
		for _, id := range ids {
			cp := &store.Checkpoint{
				ID:        id,
				NodeName:  "test-node",
				State:     "test",
				Timestamp: time.Now(),
				Version:   1,
			}
			err := ms.Save(ctx, cp)
			if err != nil {
				t.Fatalf("Failed to save %s: %v", id, err)
			}
		}

		err := ms.Delete(ctx, "delete-me")
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// Verify it's gone
		_, err = ms.Load(ctx, "delete-me")
		if err == nil {
			t.Error("Deleted checkpoint should not load")
		}

		// Verify others are still there
		_, err = ms.Load(ctx, "keep-1")
		if err != nil {
			t.Error("keep-1 should still exist")
		}

		_, err = ms.Load(ctx, "keep-2")
		if err != nil {
			t.Error("keep-2 should still exist")
		}
	})

	t.Run("delete missing is no-op", func(t *testing.T) {
		t.Parallel()

		ms := NewMemoryCheckpointStore()
		ctx := context.Background()

		err := ms.Delete(ctx, "never-existed")
		if err != nil {
			t.Errorf("Should not error for missing checkpoint: %v", err)
		}
	})
}

func TestMemoryCheckpointStore_Clear(t *testing.T) {
	t.Parallel()

	ms := NewMemoryCheckpointStore()
	ctx := context.Background()

	// Create checkpoints for two different workflows
	workflowA := "data-pipeline-2024"
	workflowB := "ml-training-job-999"

	setupData := []struct {
		id       string
		workflow string
		version  int
	}{
		{"extract-step", workflowA, 1},
		{"transform-step", workflowA, 2},
		{"load-step", workflowA, 3},
		{"model-init", workflowB, 1},
		{"training-start", workflowB, 2},
	}

	for _, d := range setupData {
		cp := &store.Checkpoint{
			ID:        d.id,
			NodeName:  "processor",
			State:     "running",
			Timestamp: time.Now(),
			Version:   d.version,
			Metadata: map[string]interface{}{
				"workflow_id": d.workflow,
			},
		}
		err := ms.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save %s: %v", d.id, err)
		}
	}

	// Verify initial state
	aList, _ := ms.List(ctx, workflowA)
	bList, _ := ms.List(ctx, workflowB)
	if len(aList) != 3 || len(bList) != 2 {
		t.Fatalf("Initial setup wrong: a=%d, b=%d", len(aList), len(bList))
	}

	err := ms.Clear(ctx, workflowA)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Workflow A should be empty
	aList, _ = ms.List(ctx, workflowA)
	if len(aList) != 0 {
		t.Errorf("Workflow A should be empty, has %d", len(aList))
	}

	// Workflow B should be untouched
	bList, _ = ms.List(ctx, workflowB)
	if len(bList) != 2 {
		t.Errorf("Workflow B should still have 2, has %d", len(bList))
	}

	// Verify individual checkpoints
	_, err = ms.Load(ctx, "extract-step")
	if err == nil {
		t.Error("extract-step should be cleared")
	}

	_, err = ms.Load(ctx, "model-init")
	if err != nil {
		t.Error("model-init should still exist")
	}
}

func TestMemoryCheckpointStore_ThreadSafety(t *testing.T) {
	t.Parallel()

	ms := NewMemoryCheckpointStore()
	ctx := context.Background()

	// Simulate multiple API endpoints writing checkpoints concurrently
	numGoroutines := 10
	checkpointsPerGoroutine := 5

	done := make(chan bool, numGoroutines)
	errs := make(chan error, numGoroutines)

	// Start multiple "workers"
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < checkpointsPerGoroutine; j++ {
				cp := &store.Checkpoint{
					ID:       fmt.Sprintf("worker-%d-step-%d", workerID, j),
					NodeName: fmt.Sprintf("handler-%d", workerID),
					State:    fmt.Sprintf("processing-step-%d", j),
					Metadata: map[string]interface{}{
						"worker_id":   workerID,
						"step_number": j,
						"timestamp":   time.Now().UnixNano(),
					},
					Timestamp: time.Now(),
					Version:   j + 1,
				}

				// Concurrent save
				if err := ms.Save(ctx, cp); err != nil {
					errs <- fmt.Errorf("worker %d save step %d failed: %v", workerID, j, err)
					return
				}

				// Concurrent load to verify it saved
				loaded, err := ms.Load(ctx, cp.ID)
				if err != nil {
					errs <- fmt.Errorf("worker %d load step %d failed: %v", workerID, j, err)
					return
				}

				if loaded.ID != cp.ID {
					errs <- fmt.Errorf("worker %d step %d ID mismatch", workerID, j)
					return
				}
			}
		}(i)
	}

	// Wait for all workers
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Worker finished
		case err := <-errs:
			t.Errorf("Worker error: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify all checkpoints are there
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < checkpointsPerGoroutine; j++ {
			id := fmt.Sprintf("worker-%d-step-%d", i, j)
			_, err := ms.Load(ctx, id)
			if err != nil {
				t.Errorf("Checkpoint %s missing", id)
			}
		}
	}
}
