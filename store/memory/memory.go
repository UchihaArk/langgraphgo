package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/smallnest/langgraphgo/store"
)

// MemoryCheckpointStore provides in-memory checkpoint storage
type MemoryCheckpointStore struct {
	checkpoints map[string]*store.Checkpoint
	mutex       sync.RWMutex
}

// NewMemoryCheckpointStore creates a new in-memory checkpoint store
func NewMemoryCheckpointStore() store.CheckpointStore {
	return &MemoryCheckpointStore{
		checkpoints: make(map[string]*store.Checkpoint),
	}
}

// Save implements CheckpointStore interface
func (m *MemoryCheckpointStore) Save(_ context.Context, checkpoint *store.Checkpoint) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.checkpoints[checkpoint.ID] = checkpoint
	return nil
}

// Load implements CheckpointStore interface
func (m *MemoryCheckpointStore) Load(_ context.Context, checkpointID string) (*store.Checkpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	checkpoint, exists := m.checkpoints[checkpointID]
	if !exists {
		return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	return checkpoint, nil
}

// List implements CheckpointStore interface
func (m *MemoryCheckpointStore) List(_ context.Context, executionID string) ([]*store.Checkpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var checkpoints []*store.Checkpoint
	for _, checkpoint := range m.checkpoints {
		// Filter by various ID fields that can be used for grouping
		execID, _ := checkpoint.Metadata["execution_id"].(string)
		threadID, _ := checkpoint.Metadata["thread_id"].(string)
		sessionID, _ := checkpoint.Metadata["session_id"].(string)
		workflowID, _ := checkpoint.Metadata["workflow_id"].(string)

		if execID == executionID || threadID == executionID || sessionID == executionID || workflowID == executionID {
			checkpoints = append(checkpoints, checkpoint)
		}
	}

	// Sort by version (ascending order) so latest is last
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Version < checkpoints[j].Version
	})

	return checkpoints, nil
}

// Delete implements CheckpointStore interface
func (m *MemoryCheckpointStore) Delete(_ context.Context, checkpointID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.checkpoints, checkpointID)
	return nil
}

// Clear implements CheckpointStore interface
func (m *MemoryCheckpointStore) Clear(_ context.Context, executionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for id, checkpoint := range m.checkpoints {
		// Check against various ID fields that can be used for grouping
		execID, _ := checkpoint.Metadata["execution_id"].(string)
		threadID, _ := checkpoint.Metadata["thread_id"].(string)
		sessionID, _ := checkpoint.Metadata["session_id"].(string)
		workflowID, _ := checkpoint.Metadata["workflow_id"].(string)

		if execID == executionID || threadID == executionID || sessionID == executionID || workflowID == executionID {
			delete(m.checkpoints, id)
		}
	}

	return nil
}
