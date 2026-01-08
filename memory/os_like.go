package memory

import (
	"context"
	"sync"
	"time"
)

// OSLikeMemory implements OS-inspired memory management with paging and eviction
// Pros: Sophisticated lifecycle management, optimal memory usage
// Cons: Complex implementation, overhead of management
type OSLikeMemory struct {
	// Active memory (like RAM) - recently accessed
	activeMemory map[string]*MemoryPage

	// Cached memory - less recently accessed
	cache map[string]*MemoryPage

	// Archived (like disk) - rarely accessed
	archived map[string]*MemoryPage

	// Configuration
	activeLimit  int           // Max pages in active memory
	cacheLimit   int           // Max pages in cache
	accessWindow time.Duration // Time window for access tracking

	mu sync.RWMutex
}

// MemoryPage represents a page of memory (like OS paging)
type MemoryPage struct {
	ID          string
	Messages    []*Message
	LastAccess  time.Time
	AccessCount int
	Priority    int  // Higher priority = less likely to be evicted
	Dirty       bool // Has been modified
	Size        int  // Token count
}

// OSLikeConfig holds configuration for OS-like memory
type OSLikeConfig struct {
	ActiveLimit  int           // Pages in active memory
	CacheLimit   int           // Pages in cache
	AccessWindow time.Duration // Access tracking window
}

// NewOSLikeMemory creates a new OS-like memory strategy
func NewOSLikeMemory(config *OSLikeConfig) *OSLikeMemory {
	if config == nil {
		config = &OSLikeConfig{
			ActiveLimit:  10,
			CacheLimit:   20,
			AccessWindow: time.Minute * 5,
		}
	}

	if config.ActiveLimit <= 0 {
		config.ActiveLimit = 10
	}
	if config.CacheLimit <= 0 {
		config.CacheLimit = 20
	}
	if config.AccessWindow <= 0 {
		config.AccessWindow = time.Minute * 5
	}

	return &OSLikeMemory{
		activeMemory: make(map[string]*MemoryPage),
		cache:        make(map[string]*MemoryPage),
		archived:     make(map[string]*MemoryPage),
		activeLimit:  config.ActiveLimit,
		cacheLimit:   config.CacheLimit,
		accessWindow: config.AccessWindow,
	}
}

// AddMessage adds a message using OS-like memory management
func (o *OSLikeMemory) AddMessage(ctx context.Context, msg *Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Create or update page
	pageID := o.getPageID(msg)
	page := o.findPage(pageID)

	if page == nil {
		// Create new page
		page = &MemoryPage{
			ID:          pageID,
			Messages:    []*Message{msg},
			LastAccess:  time.Now(),
			AccessCount: 1,
			Priority:    0,
			Dirty:       true,
			Size:        msg.TokenCount,
		}
	} else {
		// Update existing page
		page.Messages = append(page.Messages, msg)
		page.LastAccess = time.Now()
		page.AccessCount++
		page.Dirty = true
		page.Size += msg.TokenCount
	}

	// Add to active memory
	o.activeMemory[pageID] = page

	// Manage memory limits (eviction if needed)
	o.evictIfNeeded()

	return nil
}

// GetContext retrieves messages from memory hierarchy
func (o *OSLikeMemory) GetContext(ctx context.Context, query string) ([]*Message, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	result := make([]*Message, 0)

	// Collect from active memory (highest priority)
	for _, page := range o.activeMemory {
		page.LastAccess = time.Now()
		page.AccessCount++
		result = append(result, page.Messages...)
	}

	// If not enough, fetch from cache
	if len(result) < 10 {
		for pageID, page := range o.cache {
			// "Page in" from cache to active
			page.LastAccess = time.Now()
			page.AccessCount++
			result = append(result, page.Messages...)

			// Promote to active memory
			o.activeMemory[pageID] = page
			delete(o.cache, pageID)

			if len(result) >= 20 {
				break
			}
		}
	}

	return result, nil
}

// Clear removes all memory
func (o *OSLikeMemory) Clear(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.activeMemory = make(map[string]*MemoryPage)
	o.cache = make(map[string]*MemoryPage)
	o.archived = make(map[string]*MemoryPage)

	return nil
}

// GetStats returns OS-like memory statistics
func (o *OSLikeMemory) GetStats(ctx context.Context) (*Stats, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	activeTokens := 0
	cacheTokens := 0
	archivedTokens := 0

	activeCount := 0
	cacheCount := 0
	archivedCount := 0

	for _, page := range o.activeMemory {
		activeTokens += page.Size
		activeCount += len(page.Messages)
	}

	for _, page := range o.cache {
		cacheTokens += page.Size
		cacheCount += len(page.Messages)
	}

	for _, page := range o.archived {
		archivedTokens += page.Size
		archivedCount += len(page.Messages)
	}

	totalMessages := activeCount + cacheCount + archivedCount
	totalTokens := activeTokens + cacheTokens + archivedTokens

	return &Stats{
		TotalMessages:   totalMessages,
		TotalTokens:     totalTokens,
		ActiveMessages:  activeCount,
		ActiveTokens:    activeTokens,
		CompressionRate: float64(activeTokens) / float64(totalTokens),
	}, nil
}

// evictIfNeeded performs eviction when memory limits are exceeded
// Must be called with lock held
func (o *OSLikeMemory) evictIfNeeded() {
	// Evict from active to cache if over limit
	for len(o.activeMemory) > o.activeLimit {
		// Find least recently used page
		lruPage := o.findLRUPage(o.activeMemory)
		if lruPage == nil {
			break
		}

		// Move to cache
		o.cache[lruPage.ID] = lruPage
		delete(o.activeMemory, lruPage.ID)
	}

	// Evict from cache to archive if over limit
	for len(o.cache) > o.cacheLimit {
		// Find least recently used cached page
		lruPage := o.findLRUPage(o.cache)
		if lruPage == nil {
			break
		}

		// Move to archive
		o.archived[lruPage.ID] = lruPage
		delete(o.cache, lruPage.ID)
	}
}

// findLRUPage finds the least recently used page
func (o *OSLikeMemory) findLRUPage(pages map[string]*MemoryPage) *MemoryPage {
	var lruPage *MemoryPage
	var oldestAccess time.Time

	for _, page := range pages {
		if lruPage == nil || page.LastAccess.Before(oldestAccess) {
			lruPage = page
			oldestAccess = page.LastAccess
		}
	}

	return lruPage
}

// findPage finds a page across all memory levels
func (o *OSLikeMemory) findPage(pageID string) *MemoryPage {
	if page, ok := o.activeMemory[pageID]; ok {
		return page
	}
	if page, ok := o.cache[pageID]; ok {
		return page
	}
	if page, ok := o.archived[pageID]; ok {
		return page
	}
	return nil
}

// getPageID determines which page a message belongs to
// Groups messages by time window
func (o *OSLikeMemory) getPageID(msg *Message) string {
	// Group messages into 5-minute pages
	pageTime := msg.Timestamp.Truncate(time.Minute * 5)
	return pageTime.Format("2006-01-02-15:04")
}

// GetMemoryInfo returns detailed information about memory usage
func (o *OSLikeMemory) GetMemoryInfo() map[string]any {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return map[string]any{
		"active_pages":   len(o.activeMemory),
		"cached_pages":   len(o.cache),
		"archived_pages": len(o.archived),
		"active_limit":   o.activeLimit,
		"cache_limit":    o.cacheLimit,
	}
}
