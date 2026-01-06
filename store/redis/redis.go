package redis

import (
	"context"
	"encoding/json"
	"errors"
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

func (s *RedisCheckpointStore) threadKey(id string) string {
	return fmt.Sprintf("%sthread:%s:checkpoints", s.prefix, id)
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

	// Index by execution_id if present
	if execID, ok := checkpoint.Metadata["execution_id"].(string); ok && execID != "" {
		execKey := s.executionKey(execID)
		pipe.ZAdd(ctx, execKey, redis.Z{Score: float64(checkpoint.Version), Member: checkpoint.ID})
		if s.ttl > 0 {
			pipe.Expire(ctx, execKey, s.ttl)
		}
	}

	// Index by thread_id if present
	if threadID, ok := checkpoint.Metadata["thread_id"].(string); ok && threadID != "" {
		threadKey := s.threadKey(threadID)
		pipe.ZAdd(ctx, threadKey, redis.Z{Score: float64(checkpoint.Version), Member: checkpoint.ID})
		if s.ttl > 0 {
			pipe.Expire(ctx, threadKey, s.ttl)
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
	checkpointIDs, err := s.client.ZRange(ctx, execKey, 0, -1).Result()
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
	//

	return checkpoints, nil
}

// ListByThread returns all checkpoints for a specific thread_id
func (s *RedisCheckpointStore) ListByThread(ctx context.Context, threadID string) ([]*graph.Checkpoint, error) {
	threadKey := s.threadKey(threadID)
	checkpointIDs, err := s.client.ZRange(ctx, threadKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints for thread %s: %w", threadID, err)
	}

	if len(checkpointIDs) == 0 {
		return []*graph.Checkpoint{}, nil
	}

	// Fetch all checkpoints
	var keys []string
	for _, id := range checkpointIDs {
		keys = append(keys, s.checkpointKey(id))
	}

	results, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch checkpoints: %w", err)
	}

	var checkpoints []*graph.Checkpoint
	for _, result := range results {
		if result == nil {
			continue
		}

		strData, ok := result.(string)
		if !ok {
			continue
		}

		var checkpoint graph.Checkpoint
		if err := json.Unmarshal([]byte(strData), &checkpoint); err != nil {
			continue
		}
		checkpoints = append(checkpoints, &checkpoint)
	}

	return checkpoints, nil
}

// GetLatestByThread returns the latest checkpoint for a thread_id
func (s *RedisCheckpointStore) GetLatestByThread(ctx context.Context, threadID string) (*graph.Checkpoint, error) {
	threadKey := s.threadKey(threadID)

	// get latest checkpoint
	results, err := s.client.ZRevRangeWithScores(ctx, threadKey, 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest checkpoint for thread %s: %w", threadID, err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no checkpoints found for thread: %s", threadID)
	}

	latestCheckpointID := results[0].Member.(string)
	key := s.checkpointKey(latestCheckpointID)

	data, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("checkpoint not found: %s", latestCheckpointID)
		}
		return nil, fmt.Errorf("failed to load checkpoint %s: %w", latestCheckpointID, err)
	}

	var checkpoint graph.Checkpoint
	if err := json.Unmarshal([]byte(data), &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// Delete removes a checkpoint
func (s *RedisCheckpointStore) Delete(ctx context.Context, checkpointID string) error {
	// First load to get execution ID and thread ID for cleanup
	checkpoint, err := s.Load(ctx, checkpointID)
	if err != nil {
		return err // Or ignore if not found?
	}

	key := s.checkpointKey(checkpointID)
	pipe := s.client.Pipeline()

	pipe.Del(ctx, key)

	if execID, ok := checkpoint.Metadata["execution_id"].(string); ok && execID != "" {
		execKey := s.executionKey(execID)
		pipe.ZRem(ctx, execKey, checkpointID)
	}

	if threadID, ok := checkpoint.Metadata["thread_id"].(string); ok && threadID != "" {
		threadKey := s.threadKey(threadID)
		pipe.ZRem(ctx, threadKey, checkpointID)
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
	checkpointIDs, err := s.client.ZRange(ctx, execKey, 0, -1).Result()
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
