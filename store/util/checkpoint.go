// Package util provides common utilities for checkpoint store implementations.
package util

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/smallnest/langgraphgo/graph"
)

// Checkpoint wraps graph.Checkpoint for store implementations.
type Checkpoint = graph.Checkpoint

// ExtractMetadataIDs extracts execution_id and thread_id from checkpoint metadata.
// Returns empty strings if the keys are not present or have wrong types.
func ExtractMetadataIDs(checkpoint *Checkpoint) (executionID, threadID string) {
	if checkpoint == nil || checkpoint.Metadata == nil {
		return "", ""
	}

	if id, ok := checkpoint.Metadata["execution_id"].(string); ok {
		executionID = id
	}

	if id, ok := checkpoint.Metadata["thread_id"].(string); ok {
		threadID = id
	}

	return executionID, threadID
}

// MarshalCheckpointData marshals checkpoint state and metadata to JSON.
// Returns the JSON bytes and any error that occurred during marshaling.
func MarshalCheckpointData(checkpoint *Checkpoint) (stateJSON, metadataJSON []byte, err error) {
	stateJSON, err = json.Marshal(checkpoint.State)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	metadataJSON, err = json.Marshal(checkpoint.Metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return stateJSON, metadataJSON, nil
}

// UnmarshalCheckpointData unmarshals state and metadata from JSON into a checkpoint.
// Returns any error that occurred during unmarshaling.
func UnmarshalCheckpointData(stateJSON, metadataJSON []byte, cp *Checkpoint) error {
	if err := json.Unmarshal(stateJSON, &cp.State); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &cp.Metadata); err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return nil
}

// SortCheckpointsByVersion sorts a slice of checkpoints by version in ascending order.
// Modifies the slice in place.
func SortCheckpointsByVersion(checkpoints []*Checkpoint) {
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Version < checkpoints[j].Version
	})
}

// GetLastFromSorted returns the last checkpoint from a sorted slice.
// Returns an error if the slice is empty.
func GetLastFromSorted(checkpoints []*Checkpoint) (*Checkpoint, error) {
	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found")
	}
	return checkpoints[len(checkpoints)-1], nil
}

// ErrCheckpointNotFound creates a "checkpoint not found" error.
func ErrCheckpointNotFound(checkpointID string) error {
	return fmt.Errorf("checkpoint not found: %s", checkpointID)
}

// ErrNoThreadCheckpoints creates a "no checkpoints found for thread" error.
func ErrNoThreadCheckpoints(threadID string) error {
	return fmt.Errorf("no checkpoints found for thread: %s", threadID)
}
