package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	_ "github.com/mattn/go-sqlite3"
	"github.com/smallnest/langgraphgo/graph"
)

// SqliteCheckpointStore implements graph.CheckpointStore using SQLite
type SqliteCheckpointStore struct {
	db        *sql.DB
	tableName string
}

// SqliteOptions configuration for SQLite connection
type SqliteOptions struct {
	Path      string
	TableName string // Default "checkpoints"
}

// NewSqliteCheckpointStore creates a new SQLite checkpoint store
func NewSqliteCheckpointStore(opts SqliteOptions) (*SqliteCheckpointStore, error) {
	db, err := sql.Open("sqlite3", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %w", err)
	}

	tableName := opts.TableName
	if tableName == "" {
		tableName = "checkpoints"
	}

	store := &SqliteCheckpointStore{
		db:        db,
		tableName: tableName,
	}

	if err := store.InitSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// InitSchema creates the necessary table if it doesn't exist
func (s *SqliteCheckpointStore) InitSchema(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			thread_id TEXT,
			node_name TEXT NOT NULL,
			state TEXT NOT NULL,
			metadata TEXT,
			timestamp DATETIME NOT NULL,
			version INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_%s_execution_id ON %s (execution_id);
		CREATE INDEX IF NOT EXISTS idx_%s_thread_id ON %s (thread_id);
	`, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName)

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// Close closes the database connection
func (s *SqliteCheckpointStore) Close() error {
	return s.db.Close()
}

// Save stores a checkpoint
func (s *SqliteCheckpointStore) Save(ctx context.Context, checkpoint *graph.Checkpoint) error {
	stateJSON, err := json.Marshal(checkpoint.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	metadataJSON, err := json.Marshal(checkpoint.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	executionID := ""
	if id, ok := checkpoint.Metadata["execution_id"].(string); ok {
		executionID = id
	}

	threadID := ""
	if id, ok := checkpoint.Metadata["thread_id"].(string); ok {
		threadID = id
	}

	// nolint:gosec // G201: Table name cannot be parameterized, but all values use parameterized queries
	query := fmt.Sprintf(`
		INSERT INTO %s (id, execution_id, thread_id, node_name, state, metadata, timestamp, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			execution_id = excluded.execution_id,
			thread_id = excluded.thread_id,
			node_name = excluded.node_name,
			state = excluded.state,
			metadata = excluded.metadata,
			timestamp = excluded.timestamp,
			version = excluded.version
	`, s.tableName)

	_, err = s.db.ExecContext(ctx, query,
		checkpoint.ID,
		executionID,
		threadID,
		checkpoint.NodeName,
		string(stateJSON),
		string(metadataJSON),
		checkpoint.Timestamp,
		checkpoint.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Load retrieves a checkpoint by ID
func (s *SqliteCheckpointStore) Load(ctx context.Context, checkpointID string) (*graph.Checkpoint, error) {
	// nolint:gosec // G201: Table name cannot be parameterized, but all values use parameterized queries
	query := fmt.Sprintf(`
		SELECT id, node_name, state, metadata, timestamp, version
		FROM %s
		WHERE id = ?
	`, s.tableName)

	var cp graph.Checkpoint
	var stateJSON string
	var metadataJSON string

	err := s.db.QueryRowContext(ctx, query, checkpointID).Scan(
		&cp.ID,
		&cp.NodeName,
		&stateJSON,
		&metadataJSON,
		&cp.Timestamp,
		&cp.Version,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if err := json.Unmarshal([]byte(stateJSON), &cp.State); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal([]byte(metadataJSON), &cp.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &cp, nil
}

// List returns all checkpoints for a given execution
func (s *SqliteCheckpointStore) List(ctx context.Context, executionID string) ([]*graph.Checkpoint, error) {
	// nolint:gosec // G201: Table name cannot be parameterized, but all values use parameterized queries
	query := fmt.Sprintf(`
		SELECT id, node_name, state, metadata, timestamp, version
		FROM %s
		WHERE execution_id = ?
		ORDER BY timestamp ASC
	`, s.tableName)

	rows, err := s.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*graph.Checkpoint
	for rows.Next() {
		var cp graph.Checkpoint
		var stateJSON string
		var metadataJSON string

		err := rows.Scan(
			&cp.ID,
			&cp.NodeName,
			&stateJSON,
			&metadataJSON,
			&cp.Timestamp,
			&cp.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint row: %w", err)
		}

		if err := json.Unmarshal([]byte(stateJSON), &cp.State); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal([]byte(metadataJSON), &cp.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		checkpoints = append(checkpoints, &cp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checkpoint rows: %w", err)
	}

	return checkpoints, nil
}

// Delete removes a checkpoint
func (s *SqliteCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
	// nolint:gosec // G201: Table name cannot be parameterized, but all values use parameterized queries
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.tableName)
	_, err := s.db.ExecContext(ctx, query, checkpointID)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	return nil
}

// Clear removes all checkpoints for an execution
func (s *SqliteCheckpointStore) Clear(ctx context.Context, executionID string) error {
	// nolint:gosec // G201: Table name cannot be parameterized, but all values use parameterized queries
	query := fmt.Sprintf("DELETE FROM %s WHERE execution_id = ?", s.tableName)
	_, err := s.db.ExecContext(ctx, query, executionID)
	if err != nil {
		return fmt.Errorf("failed to clear checkpoints: %w", err)
	}
	return nil
}

// ListByThread returns all checkpoints for a specific thread_id
func (s *SqliteCheckpointStore) ListByThread(ctx context.Context, threadID string) ([]*graph.Checkpoint, error) {
	// nolint:gosec // G201: Table name cannot be parameterized, but all values use parameterized queries
	query := fmt.Sprintf(`
		SELECT id, node_name, state, metadata, timestamp, version
		FROM %s
		WHERE thread_id = ?
		ORDER BY timestamp ASC
	`, s.tableName)

	rows, err := s.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints by thread: %w", err)
	}
	defer rows.Close()

	var checkpoints []*graph.Checkpoint
	for rows.Next() {
		var cp graph.Checkpoint
		var stateJSON string
		var metadataJSON string

		err := rows.Scan(
			&cp.ID,
			&cp.NodeName,
			&stateJSON,
			&metadataJSON,
			&cp.Timestamp,
			&cp.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint row: %w", err)
		}

		if err := json.Unmarshal([]byte(stateJSON), &cp.State); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal([]byte(metadataJSON), &cp.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		checkpoints = append(checkpoints, &cp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating checkpoint rows: %w", err)
	}

	// Sort by version ascending
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Version < checkpoints[j].Version
	})

	return checkpoints, nil
}

// GetLatestByThread returns the latest checkpoint for a thread_id
func (s *SqliteCheckpointStore) GetLatestByThread(ctx context.Context, threadID string) (*graph.Checkpoint, error) {
	checkpoints, err := s.ListByThread(ctx, threadID)
	if err != nil {
		return nil, err
	}

	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for thread: %s", threadID)
	}

	// Return the last one (highest version due to sorting)
	return checkpoints[len(checkpoints)-1], nil
}
