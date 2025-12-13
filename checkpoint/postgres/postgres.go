package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/smallnest/langgraphgo/graph"
)

// DBPool defines the interface for database connection pool
type DBPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Close()
}

// PostgresCheckpointStore implements graph.CheckpointStore using PostgreSQL
type PostgresCheckpointStore struct {
	pool      DBPool
	tableName string
}

// PostgresOptions configuration for Postgres connection
type PostgresOptions struct {
	ConnString string
	TableName  string // Default "checkpoints"
}

// NewPostgresCheckpointStore creates a new Postgres checkpoint store
func NewPostgresCheckpointStore(ctx context.Context, opts PostgresOptions) (*PostgresCheckpointStore, error) {
	pool, err := pgxpool.New(ctx, opts.ConnString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	tableName := opts.TableName
	if tableName == "" {
		tableName = "checkpoints"
	}

	return &PostgresCheckpointStore{
		pool:      pool,
		tableName: tableName,
	}, nil
}

// NewPostgresCheckpointStoreWithPool creates a new Postgres checkpoint store with an existing pool
// Useful for testing with mocks
func NewPostgresCheckpointStoreWithPool(pool DBPool, tableName string) *PostgresCheckpointStore {
	if tableName == "" {
		tableName = "checkpoints"
	}
	return &PostgresCheckpointStore{
		pool:      pool,
		tableName: tableName,
	}
}

// InitSchema creates the necessary table if it doesn't exist
func (s *PostgresCheckpointStore) InitSchema(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			node_name TEXT NOT NULL,
			state JSONB NOT NULL,
			metadata JSONB,
			timestamp TIMESTAMPTZ NOT NULL,
			version INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_%s_execution_id ON %s (execution_id);
	`, s.tableName, s.tableName, s.tableName)

	_, err := s.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

// Close closes the connection pool
func (s *PostgresCheckpointStore) Close() {
	s.pool.Close()
}

// Save stores a checkpoint
func (s *PostgresCheckpointStore) Save(ctx context.Context, checkpoint *graph.Checkpoint) error {
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

	query := fmt.Sprintf(`
		INSERT INTO %s (id, execution_id, node_name, state, metadata, timestamp, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			execution_id = EXCLUDED.execution_id,
			node_name = EXCLUDED.node_name,
			state = EXCLUDED.state,
			metadata = EXCLUDED.metadata,
			timestamp = EXCLUDED.timestamp,
			version = EXCLUDED.version
	`, s.tableName)

	_, err = s.pool.Exec(ctx, query,
		checkpoint.ID,
		executionID,
		checkpoint.NodeName,
		stateJSON,
		metadataJSON,
		checkpoint.Timestamp,
		checkpoint.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Load retrieves a checkpoint by ID
func (s *PostgresCheckpointStore) Load(ctx context.Context, checkpointID string) (*graph.Checkpoint, error) {
	query := fmt.Sprintf(`
		SELECT id, node_name, state, metadata, timestamp, version
		FROM %s
		WHERE id = $1
	`, s.tableName)

	var cp graph.Checkpoint
	var stateJSON []byte
	var metadataJSON []byte

	err := s.pool.QueryRow(ctx, query, checkpointID).Scan(
		&cp.ID,
		&cp.NodeName,
		&stateJSON,
		&metadataJSON,
		&cp.Timestamp,
		&cp.Version,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if err := json.Unmarshal(stateJSON, &cp.State); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &cp.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &cp, nil
}

// List returns all checkpoints for a given execution
func (s *PostgresCheckpointStore) List(ctx context.Context, executionID string) ([]*graph.Checkpoint, error) {
	query := fmt.Sprintf(`
		SELECT id, node_name, state, metadata, timestamp, version
		FROM %s
		WHERE execution_id = $1
		ORDER BY timestamp ASC
	`, s.tableName)

	rows, err := s.pool.Query(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*graph.Checkpoint
	for rows.Next() {
		var cp graph.Checkpoint
		var stateJSON []byte
		var metadataJSON []byte

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

		if err := json.Unmarshal(stateJSON, &cp.State); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &cp.Metadata); err != nil {
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
func (s *PostgresCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", s.tableName)
	_, err := s.pool.Exec(ctx, query, checkpointID)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	return nil
}

// Clear removes all checkpoints for an execution
func (s *PostgresCheckpointStore) Clear(ctx context.Context, executionID string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE execution_id = $1", s.tableName)
	_, err := s.pool.Exec(ctx, query, executionID)
	if err != nil {
		return fmt.Errorf("failed to clear checkpoints: %w", err)
	}
	return nil
}
