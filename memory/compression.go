package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CompressionMemory periodically compresses and consolidates memory
// Pros: Maintains long-term context efficiently, removes redundancy
// Cons: Compression requires LLM calls, may lose granular details
type CompressionMemory struct {
	messages           []*Message
	compressedBlocks   []*CompressedBlock
	compressionTrigger int           // Compress after N messages
	consolidateAfter   time.Duration // Consolidate blocks after duration
	lastConsolidation  time.Time
	mu                 sync.RWMutex

	// Compressor compresses a group of messages into a single block
	Compressor func(ctx context.Context, messages []*Message) (*CompressedBlock, error)

	// Consolidator merges multiple compressed blocks
	Consolidator func(ctx context.Context, blocks []*CompressedBlock) (*CompressedBlock, error)
}

// CompressedBlock represents a compressed group of messages
type CompressedBlock struct {
	ID               string    // Unique block ID
	Summary          string    // Compressed summary
	OriginalCount    int       // Number of original messages
	OriginalTokens   int       // Original token count
	CompressedTokens int       // Compressed token count
	TimeRange        TimeRange // Time range of messages
	Topics           []string  // Main topics covered
}

// TimeRange represents a time period
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// CompressionConfig holds configuration for compression memory
type CompressionConfig struct {
	CompressionTrigger int           // Messages before compression
	ConsolidateAfter   time.Duration // Duration before consolidation
	Compressor         func(ctx context.Context, messages []*Message) (*CompressedBlock, error)
	Consolidator       func(ctx context.Context, blocks []*CompressedBlock) (*CompressedBlock, error)
}

// NewCompressionMemory creates a new compression-based memory strategy
func NewCompressionMemory(config *CompressionConfig) *CompressionMemory {
	if config == nil {
		config = &CompressionConfig{
			CompressionTrigger: 20,
			ConsolidateAfter:   time.Hour * 1,
		}
	}

	if config.CompressionTrigger <= 0 {
		config.CompressionTrigger = 20
	}
	if config.ConsolidateAfter <= 0 {
		config.ConsolidateAfter = time.Hour
	}

	compressor := config.Compressor
	if compressor == nil {
		compressor = defaultCompressor
	}

	consolidator := config.Consolidator
	if consolidator == nil {
		consolidator = defaultConsolidator
	}

	return &CompressionMemory{
		messages:           make([]*Message, 0),
		compressedBlocks:   make([]*CompressedBlock, 0),
		compressionTrigger: config.CompressionTrigger,
		consolidateAfter:   config.ConsolidateAfter,
		lastConsolidation:  time.Now(),
		Compressor:         compressor,
		Consolidator:       consolidator,
	}
}

// AddMessage adds a message and triggers compression if needed
func (c *CompressionMemory) AddMessage(ctx context.Context, msg *Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages = append(c.messages, msg)

	// Check if compression is needed
	if len(c.messages) >= c.compressionTrigger {
		if err := c.compress(ctx); err != nil {
			return fmt.Errorf("compression failed: %w", err)
		}
	}

	// Check if consolidation is needed
	if time.Since(c.lastConsolidation) >= c.consolidateAfter {
		if err := c.consolidate(ctx); err != nil {
			return fmt.Errorf("consolidation failed: %w", err)
		}
	}

	return nil
}

// compress compresses current messages into a block
// Must be called with lock held
func (c *CompressionMemory) compress(ctx context.Context) error {
	if len(c.messages) == 0 {
		return nil
	}

	// Compress messages
	block, err := c.Compressor(ctx, c.messages)
	if err != nil {
		return err
	}

	// Store compressed block
	c.compressedBlocks = append(c.compressedBlocks, block)

	// Clear messages
	c.messages = make([]*Message, 0)

	return nil
}

// consolidate merges old compressed blocks
// Must be called with lock held
func (c *CompressionMemory) consolidate(ctx context.Context) error {
	if len(c.compressedBlocks) < 2 {
		c.lastConsolidation = time.Now()
		return nil
	}

	// Consolidate older blocks (keep most recent separate)
	blocksToConsolidate := c.compressedBlocks[:len(c.compressedBlocks)-1]

	if len(blocksToConsolidate) > 0 {
		consolidated, err := c.Consolidator(ctx, blocksToConsolidate)
		if err != nil {
			return err
		}

		// Replace old blocks with consolidated one
		c.compressedBlocks = []*CompressedBlock{consolidated, c.compressedBlocks[len(c.compressedBlocks)-1]}
	}

	c.lastConsolidation = time.Now()
	return nil
}

// GetContext returns compressed blocks and recent messages
func (c *CompressionMemory) GetContext(ctx context.Context, query string) ([]*Message, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Message, 0)

	// Add compressed blocks as system messages
	for i, block := range c.compressedBlocks {
		blockMsg := &Message{
			ID:      fmt.Sprintf("block_%d", i),
			Role:    "system",
			Content: fmt.Sprintf("[Compressed Memory Block %d]: %s", i+1, block.Summary),
			Metadata: map[string]any{
				"block_id":       block.ID,
				"original_count": block.OriginalCount,
				"topics":         block.Topics,
			},
			TokenCount: block.CompressedTokens,
			Timestamp:  block.TimeRange.End,
		}
		result = append(result, blockMsg)
	}

	// Add recent uncompressed messages
	result = append(result, c.messages...)

	return result, nil
}

// Clear removes all memory
func (c *CompressionMemory) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.messages = make([]*Message, 0)
	c.compressedBlocks = make([]*CompressedBlock, 0)
	c.lastConsolidation = time.Now()
	return nil
}

// GetStats returns compression statistics
func (c *CompressionMemory) GetStats(ctx context.Context) (*Stats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Count original and compressed tokens
	originalTokens := 0
	compressedTokens := 0

	for _, block := range c.compressedBlocks {
		originalTokens += block.OriginalTokens
		compressedTokens += block.CompressedTokens
	}

	// Add current messages
	currentTokens := 0
	for _, msg := range c.messages {
		currentTokens += msg.TokenCount
	}

	totalCompressed := compressedTokens + currentTokens
	totalOriginal := originalTokens + currentTokens

	compressionRate := 1.0
	if totalOriginal > 0 {
		compressionRate = float64(totalCompressed) / float64(totalOriginal)
	}

	totalMessages := 0
	for _, block := range c.compressedBlocks {
		totalMessages += block.OriginalCount
	}
	totalMessages += len(c.messages)

	return &Stats{
		TotalMessages:   totalMessages,
		TotalTokens:     totalOriginal,
		ActiveMessages:  len(c.compressedBlocks) + len(c.messages),
		ActiveTokens:    totalCompressed,
		CompressionRate: compressionRate,
	}, nil
}

// ForceCompression manually triggers compression
func (c *CompressionMemory) ForceCompression(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.compress(ctx)
}

// ForceConsolidation manually triggers consolidation
func (c *CompressionMemory) ForceConsolidation(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consolidate(ctx)
}

// defaultCompressor provides a simple compression function
func defaultCompressor(ctx context.Context, messages []*Message) (*CompressedBlock, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to compress")
	}

	// Calculate statistics
	originalTokens := 0
	for _, msg := range messages {
		originalTokens += msg.TokenCount
	}

	// Create simple summary
	summary := fmt.Sprintf("Compressed %d messages from %s to %s",
		len(messages),
		messages[0].Timestamp.Format("15:04"),
		messages[len(messages)-1].Timestamp.Format("15:04"))

	// Estimate compressed tokens (roughly 1/3 of original)
	compressedTokens := originalTokens / 3

	block := &CompressedBlock{
		ID:               generateID(),
		Summary:          summary,
		OriginalCount:    len(messages),
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		TimeRange: TimeRange{
			Start: messages[0].Timestamp,
			End:   messages[len(messages)-1].Timestamp,
		},
		Topics: []string{"general"},
	}

	return block, nil
}

// defaultConsolidator merges multiple blocks
func defaultConsolidator(ctx context.Context, blocks []*CompressedBlock) (*CompressedBlock, error) {
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no blocks to consolidate")
	}

	// Merge statistics
	totalOriginalCount := 0
	totalOriginalTokens := 0
	totalCompressedTokens := 0
	allTopics := make(map[string]bool)

	var start, end time.Time
	for i, block := range blocks {
		totalOriginalCount += block.OriginalCount
		totalOriginalTokens += block.OriginalTokens
		totalCompressedTokens += block.CompressedTokens

		for _, topic := range block.Topics {
			allTopics[topic] = true
		}

		if i == 0 || block.TimeRange.Start.Before(start) {
			start = block.TimeRange.Start
		}
		if i == 0 || block.TimeRange.End.After(end) {
			end = block.TimeRange.End
		}
	}

	// Collect topics
	topics := make([]string, 0, len(allTopics))
	for topic := range allTopics {
		topics = append(topics, topic)
	}

	// Further compress (estimate 2/3 of combined compressed size)
	finalCompressedTokens := (totalCompressedTokens * 2) / 3

	consolidated := &CompressedBlock{
		ID:               generateID(),
		Summary:          fmt.Sprintf("Consolidated %d blocks covering %d messages", len(blocks), totalOriginalCount),
		OriginalCount:    totalOriginalCount,
		OriginalTokens:   totalOriginalTokens,
		CompressedTokens: finalCompressedTokens,
		TimeRange: TimeRange{
			Start: start,
			End:   end,
		},
		Topics: topics,
	}

	return consolidated, nil
}
