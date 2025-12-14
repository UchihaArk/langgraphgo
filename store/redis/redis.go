package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/smallnest/langgraphgo/graph"
)

// RedisCheckpointStore implements graph.CheckpointStore using Redis
type RedisCheckpointStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// RedisOptions configuration for Redis connection
type RedisOptions struct {
	Addr     string
	Password string
	DB       int
	Prefix   string        // Key prefix, default "langgraph:"
	TTL      time.Duration // Expiration for checkpoints, default 0 (no expiration)
}

// NewRedisCheckpointStore creates a new Redis checkpoint store
func NewRedisCheckpointStore(opts RedisOptions) *RedisCheckpointStore {
	client := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})

	prefix := opts.Prefix
	if prefix == "" {
		prefix = "langgraph:"
	}

	return &RedisCheckpointStore{
		client: client,
		prefix: prefix,
		ttl:    opts.TTL,
	}
}

func (s *RedisCheckpointStore) checkpointKey(id string) string {
	return fmt.Sprintf("%scheckpoint:%s", s.prefix, id)
}

func (s *RedisCheckpointStore) executionKey(id string) string {
	return fmt.Sprintf("%sexecution:%s:checkpoints", s.prefix, id)
}

// Save stores a checkpoint
func (s *RedisCheckpointStore) Save(ctx context.Context, checkpoint *graph.Checkpoint) error {
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	key := s.checkpointKey(checkpoint.ID)
	pipe := s.client.Pipeline()

	pipe.Set(ctx, key, data, s.ttl)

	// Index by execution ID if present
	if execID, ok := checkpoint.Metadata["execution_id"].(string); ok && execID != "" {
		execKey := s.executionKey(execID)
		pipe.SAdd(ctx, execKey, checkpoint.ID)
		if s.ttl > 0 {
			pipe.Expire(ctx, execKey, s.ttl)
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint to redis: %w", err)
	}

	return nil
}

// Load retrieves a checkpoint by ID
func (s *RedisCheckpointStore) Load(ctx context.Context, checkpointID string) (*graph.Checkpoint, error) {
	key := s.checkpointKey(checkpointID)
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
		}
		return nil, fmt.Errorf("failed to load checkpoint from redis: %w", err)
	}

	var checkpoint graph.Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// List returns all checkpoints for a given execution
func (s *RedisCheckpointStore) List(ctx context.Context, executionID string) ([]*graph.Checkpoint, error) {
	execKey := s.executionKey(executionID)
	checkpointIDs, err := s.client.SMembers(ctx, execKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints for execution %s: %w", executionID, err)
	}

	if len(checkpointIDs) == 0 {
		return []*graph.Checkpoint{}, nil
	}

	// Fetch all checkpoints
	var keys []string
	for _, id := range checkpointIDs {
		keys = append(keys, s.checkpointKey(id))
	}

	// MGet might fail if some keys are missing (expired), so we handle them individually or filter results
	// But MGet returns nil for missing keys, which is fine.
	results, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch checkpoints: %w", err)
	}

	var checkpoints []*graph.Checkpoint
	for i, result := range results {
		if result == nil {
			continue
		}

		strData, ok := result.(string)
		if !ok {
			continue
		}

		var checkpoint graph.Checkpoint
		if err := json.Unmarshal([]byte(strData), &checkpoint); err != nil {
			// Log error or skip? Skipping for now
			continue
		}
		checkpoints = append(checkpoints, &checkpoint)

		// Sanity check ID - should match if order is preserved
		// If mismatch occurs, it indicates a Redis ordering issue
		_ = checkpointIDs[i] // Acknowledge ID is available for future validation
	}

	return checkpoints, nil
}

// Delete removes a checkpoint
func (s *RedisCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
	// First load to get execution ID for cleanup
	checkpoint, err := s.Load(ctx, checkpointID)
	if err != nil {
		return err // Or ignore if not found?
	}

	key := s.checkpointKey(checkpointID)
	pipe := s.client.Pipeline()

	pipe.Del(ctx, key)

	if execID, ok := checkpoint.Metadata["execution_id"].(string); ok && execID != "" {
		execKey := s.executionKey(execID)
		pipe.SRem(ctx, execKey, checkpointID)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	return nil
}

// Clear removes all checkpoints for an execution
func (s *RedisCheckpointStore) Clear(ctx context.Context, executionID string) error {
	execKey := s.executionKey(executionID)
	checkpointIDs, err := s.client.SMembers(ctx, execKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get checkpoints for clearing: %w", err)
	}

	if len(checkpointIDs) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()

	// Delete all checkpoint keys
	for _, id := range checkpointIDs {
		pipe.Del(ctx, s.checkpointKey(id))
	}

	// Delete execution index
	pipe.Del(ctx, execKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to clear checkpoints: %w", err)
	}

	return nil
}
