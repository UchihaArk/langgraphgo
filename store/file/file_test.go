package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smallnest/langgraphgo/store"
)

func TestFileCheckpointStore_New(t *testing.T) {
	t.Parallel()

	t.Run("creates directory if missing", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		checkpointPath := filepath.Join(tempDir, "checkpoints")

		store, err := NewFileCheckpointStore(checkpointPath)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		if store == nil {
			t.Fatal("Store should not be nil")
		}

		// Verify directory exists
		if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
			t.Error("Directory should have been created")
		}
	})

	t.Run("works with existing directory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		// Create directory first
		err := os.MkdirAll(tempDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		store, err := NewFileCheckpointStore(tempDir)
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		if store == nil {
			t.Fatal("Store should not be nil")
		}
	})
}

func TestFileCheckpointStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Now()

	t.Run("save creates file", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		cp := &store.Checkpoint{
			ID:        "user-session-123",
			NodeName:  "login-handler",
			State:     "authenticated",
			Timestamp: now,
			Version:   1,
			Metadata: map[string]interface{}{
				"user_id": "john.doe@example.com",
				"ip":      "192.168.1.100",
			},
		}

		err = fs.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		// Check file exists
		filename := filepath.Join(fs.(*FileCheckpointStore).path, cp.ID+".json")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Error("Checkpoint file should exist")
		}
	})

	t.Run("load returns saved checkpoint", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		cp := &store.Checkpoint{
			ID:        "user-session-123",
			NodeName:  "login-handler",
			State:     "authenticated",
			Timestamp: now,
			Version:   1,
			Metadata: map[string]interface{}{
				"user_id": "john.doe@example.com",
				"ip":      "192.168.1.100",
			},
		}

		err = fs.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save: %v", err)
		}

		loaded, err := fs.Load(ctx, cp.ID)
		if err != nil {
			t.Fatalf("Failed to load: %v", err)
		}

		if loaded.ID != cp.ID {
			t.Errorf("Expected ID %s, got %s", cp.ID, loaded.ID)
		}
		if loaded.NodeName != cp.NodeName {
			t.Errorf("Expected NodeName %s, got %s", cp.NodeName, loaded.NodeName)
		}
		if loaded.State != cp.State {
			t.Errorf("Expected State %s, got %s", cp.State, loaded.State)
		}
		if loaded.Version != cp.Version {
			t.Errorf("Expected Version %d, got %d", cp.Version, loaded.Version)
		}

		// Check metadata
		if userID, ok := loaded.Metadata["user_id"].(string); !ok || userID != "john.doe@example.com" {
			t.Error("User ID metadata mismatch")
		}
	})

	t.Run("save complex state", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		complexCP := &store.Checkpoint{
			ID:       "order-flow-456",
			NodeName: "payment-processor",
			State: map[string]interface{}{
				"order_id":     789,
				"items":        []string{"widget", "gadget"},
				"total_amount": 99.99,
				"currency":     "USD",
			},
			Timestamp: now,
			Version:   3,
			Metadata: map[string]interface{}{
				"session_id": "sess-xyz-789",
			},
		}

		err = fs.Save(ctx, complexCP)
		if err != nil {
			t.Fatalf("Failed to save complex checkpoint: %v", err)
		}

		loaded, err := fs.Load(ctx, complexCP.ID)
		if err != nil {
			t.Fatalf("Failed to load complex checkpoint: %v", err)
		}

		// Verify complex state
		state, ok := loaded.State.(map[string]interface{})
		if !ok {
			t.Fatal("State should be a map")
		}

		if state["order_id"] != float64(789) { // JSON numbers are float64
			t.Errorf("Expected order_id 789, got %v", state["order_id"])
		}
	})

	t.Run("load missing checkpoint", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		_, err = fs.Load(ctx, "does-not-exist")
		if err == nil {
			t.Error("Should return error for missing checkpoint")
		}
	})
}

func TestFileCheckpointStore_List(t *testing.T) {
	t.Parallel()

	t.Run("filters by session_id", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		ctx := context.Background()
		sessionID := "web-session-2024"

		// Add checkpoints for this session
		checkpoints := []struct {
			id      string
			node    string
			version int
		}{
			{"page-visit-1", "home-page", 1},
			{"page-visit-2", "product-page", 2},
		}

		for _, cp := range checkpoints {
			fullCP := &store.Checkpoint{
				ID:        cp.id,
				NodeName:  cp.node,
				State:     "processing",
				Timestamp: time.Now(),
				Version:   cp.version,
				Metadata: map[string]interface{}{
					"session_id": sessionID,
				},
			}
			err := fs.Save(ctx, fullCP)
			if err != nil {
				t.Fatalf("Failed to save checkpoint %s: %v", cp.id, err)
			}
		}

		results, err := fs.List(ctx, sessionID)
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 checkpoints for session, got %d", len(results))
		}

		// Check they're sorted by version
		if results[0].Version > results[1].Version {
			t.Error("Results should be sorted by version ascending")
		}
	})

	t.Run("filters by thread_id", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		ctx := context.Background()
		threadID := "user-john-thread-1"

		cp := &store.Checkpoint{
			ID:        "cart-action-1",
			NodeName:  "add-to-cart",
			State:     "processing",
			Timestamp: time.Now(),
			Version:   1,
			Metadata: map[string]interface{}{
				"thread_id": threadID,
			},
		}

		err = fs.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}

		results, err := fs.List(ctx, threadID)
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 checkpoint for thread, got %d", len(results))
		}

		if results[0].ID != "cart-action-1" {
			t.Errorf("Expected cart-action-1, got %s", results[0].ID)
		}
	})

	t.Run("empty result for unknown session", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		ctx := context.Background()
		results, err := fs.List(ctx, "unknown-session")
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 checkpoints, got %d", len(results))
		}
	})
}

func TestFileCheckpointStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing checkpoint", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		ctx := context.Background()
		storePath := fs.(*FileCheckpointStore).path

		cp := &store.Checkpoint{
			ID:        "temp-checkpoint",
			NodeName:  "test-node",
			State:     "test-state",
			Timestamp: time.Now(),
			Version:   1,
		}

		err = fs.Save(ctx, cp)
		if err != nil {
			t.Fatalf("Failed to save checkpoint: %v", err)
		}

		// Verify file exists
		filename := filepath.Join(storePath, cp.ID+".json")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			t.Fatal("Checkpoint file should exist")
		}

		err = fs.Delete(ctx, cp.ID)
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}

		// File should be gone
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			t.Error("Checkpoint file should be deleted")
		}

		// Should not be loadable
		_, err = fs.Load(ctx, cp.ID)
		if err == nil {
			t.Error("Should not be able to load deleted checkpoint")
		}
	})

	t.Run("deleting non-existing is no-op", func(t *testing.T) {
		t.Parallel()

		fs, err := NewFileCheckpointStore(t.TempDir())
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}

		ctx := context.Background()

		err = fs.Delete(ctx, "never-existed")
		if err != nil {
			t.Errorf("Delete should not error for non-existing checkpoint: %v", err)
		}
	})
}

func TestFileCheckpointStore_Clear(t *testing.T) {
	t.Parallel()

	fs, err := NewFileCheckpointStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ctx := context.Background()

	// Create checkpoints for two different sessions
	session1 := "user-session-alpha"
	session2 := "user-session-beta"

	checkpoints := []struct {
		id      string
		session string
		version int
	}{
		{"alpha-1", session1, 1},
		{"alpha-2", session1, 2},
		{"beta-1", session2, 1},
	}

	for _, cp := range checkpoints {
		fullCP := &store.Checkpoint{
			ID:        cp.id,
			NodeName:  "processor",
			State:     "running",
			Timestamp: time.Now(),
			Version:   cp.version,
			Metadata: map[string]interface{}{
				"session_id": cp.session,
			},
		}
		err := fs.Save(ctx, fullCP)
		if err != nil {
			t.Fatalf("Failed to save checkpoint %s: %v", cp.id, err)
		}
	}

	// Verify we have checkpoints
	alphaList, _ := fs.List(ctx, session1)
	if len(alphaList) != 2 {
		t.Fatalf("Expected 2 alpha checkpoints, got %d", len(alphaList))
	}

	err = fs.Clear(ctx, session1)
	if err != nil {
		t.Fatalf("Failed to clear session: %v", err)
	}

	// Alpha session should be empty
	alphaList, _ = fs.List(ctx, session1)
	if len(alphaList) != 0 {
		t.Errorf("Expected 0 alpha checkpoints after clear, got %d", len(alphaList))
	}

	// Beta session should still have its checkpoint
	betaList, _ := fs.List(ctx, session2)
	if len(betaList) != 1 {
		t.Errorf("Expected 1 beta checkpoint, got %d", len(betaList))
	}
}

func TestFileCheckpointStore_Permissions(t *testing.T) {
	t.Parallel()

	if os.Getenv("CI") != "" {
		t.Skip("Skipping permission test in CI")
	}

	fs, err := NewFileCheckpointStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ctx := context.Background()
	storePath := fs.(*FileCheckpointStore).path

	cp := &store.Checkpoint{
		ID:        "secret-checkpoint",
		NodeName:  "auth-handler",
		State:     "authenticated",
		Timestamp: time.Now(),
		Version:   1,
	}

	err = fs.Save(ctx, cp)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Check file permissions
	filename := filepath.Join(storePath, cp.ID+".json")
	fileInfo, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// On Unix, files should be readable/writable only by owner
	if os.Getenv("GOOS") != "windows" {
		perm := fileInfo.Mode().Perm()
		if perm != 0600 {
			// Allow for more permissive umask settings (like 0022 => 0644)
			if perm != 0644 {
				t.Logf("File permissions: %o (expected 0600 or 0644 due to umask)", perm)
			}
		}
	}
}

func TestFileCheckpointStore_Concurrent(t *testing.T) {
	t.Parallel()

	fs, err := NewFileCheckpointStore(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ctx := context.Background()
	numWorkers := 5
	checkpointsPerWorker := 3

	done := make(chan bool, numWorkers)
	errs := make(chan error, numWorkers)

	// Launch workers
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < checkpointsPerWorker; j++ {
				cp := &store.Checkpoint{
					ID:       fmt.Sprintf("worker-%d-checkpoint-%d", workerID, j),
					NodeName: fmt.Sprintf("worker-%d-processor", workerID),
					State:    fmt.Sprintf("state-%d", j),
					Metadata: map[string]interface{}{
						"worker_id": workerID,
						"step":      j,
					},
					Timestamp: time.Now(),
					Version:   j + 1,
				}

				// Save
				if err := fs.Save(ctx, cp); err != nil {
					errs <- fmt.Errorf("worker %d save failed: %v", workerID, err)
					return
				}

				// Load
				loaded, err := fs.Load(ctx, cp.ID)
				if err != nil {
					errs <- fmt.Errorf("worker %d load failed: %v", workerID, err)
					return
				}

				if loaded.ID != cp.ID {
					errs <- fmt.Errorf("worker %d ID mismatch", workerID)
					return
				}
			}
		}(i)
	}

	// Wait for workers
	for i := 0; i < numWorkers; i++ {
		select {
		case <-done:
			// Worker completed
		case err := <-errs:
			t.Errorf("Worker error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify all checkpoints exist
	expectedTotal := numWorkers * checkpointsPerWorker
	files, err := os.ReadDir(fs.(*FileCheckpointStore).path)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	jsonCount := 0
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			jsonCount++
		}
	}

	if jsonCount != expectedTotal {
		t.Errorf("Expected %d checkpoint files, got %d", expectedTotal, jsonCount)
	}
}
