package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/smallnest/langgraphgo/store"
)

// FileCheckpointStore provides file-based checkpoint storage
type FileCheckpointStore struct {
	path  string
	mutex sync.RWMutex
}

// NewFileCheckpointStore creates a new file-based checkpoint store
func NewFileCheckpointStore(path string) (store.CheckpointStore, error) {
	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	return &FileCheckpointStore{
		path: path,
	}, nil
}

// Save implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Save(_ context.Context, checkpoint *store.Checkpoint) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Create filename from ID
	filename := filepath.Join(f.path, fmt.Sprintf("%s.json", checkpoint.ID))

	data, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	return nil
}

// Load implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Load(_ context.Context, checkpointID string) (*store.Checkpoint, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	filename := filepath.Join(f.path, fmt.Sprintf("%s.json", checkpointID))

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
		}
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	var checkpoint store.Checkpoint
	err = json.Unmarshal(data, &checkpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// List implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) List(_ context.Context, executionID string) ([]*store.Checkpoint, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	files, err := os.ReadDir(f.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var checkpoints []*store.Checkpoint

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(f.path, file.Name()))
			if err != nil {
				// Skip unreadable files
				continue
			}

			var checkpoint store.Checkpoint
			if err := json.Unmarshal(data, &checkpoint); err != nil {
				// Skip invalid files
				continue
			}

			// Filter by executionID, threadID, sessionID, or workflowID
			execID, _ := checkpoint.Metadata["execution_id"].(string)
			threadID, _ := checkpoint.Metadata["thread_id"].(string)
			sessionID, _ := checkpoint.Metadata["session_id"].(string)
			workflowID, _ := checkpoint.Metadata["workflow_id"].(string)

			if execID == executionID || threadID == executionID || sessionID == executionID || workflowID == executionID {
				checkpoints = append(checkpoints, &checkpoint)
			}
		}
	}

	// Sort by version (ascending order) so latest is last
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Version < checkpoints[j].Version
	})

	return checkpoints, nil
}

// Delete implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Delete(_ context.Context, checkpointID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	filename := filepath.Join(f.path, fmt.Sprintf("%s.json", checkpointID))

	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, we consider it deleted
			return nil
		}
		return fmt.Errorf("failed to delete checkpoint file: %w", err)
	}

	return nil
}

// Clear implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Clear(ctx context.Context, executionID string) error {
	// We iterate through all files using List (which already filters and reads),
	// but we should probably do a raw read here to avoid overhead if list is slow,
	// however, List Logic is fine for now as it reuses logic.
	// Actually, let's just re-implement simple loop to avoid locking recursion if we called f.Delete inside f.List loop scope if we weren't careful.
	// But List is read-lock. Delete is write-lock. upgrading lock is dangerous.
	// So we should get IDs first, then delete.

	checkpoints, err := f.List(ctx, executionID)
	if err != nil {
		return err
	}

	var errs []error
	for _, cp := range checkpoints {
		if err := f.Delete(ctx, cp.ID); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to clear some checkpoints: %v", errs)
	}

	return nil
}
