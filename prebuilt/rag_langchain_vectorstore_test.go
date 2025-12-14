package prebuilt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

// MockLangChainVectorStore is a mock implementation of vectorstores.VectorStore for testing
type MockLangChainVectorStore struct {
	documents []Document
	// embeddings field removed as it's unused - if needed in future, uncomment:
	// embeddings [][]float64
}

func (m *MockLangChainVectorStore) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) ([]string, error) {
	// Convert and store documents
	for i, doc := range docs {
		m.documents = append(m.documents, Document{
			PageContent: doc.PageContent,
			Metadata:    doc.Metadata,
		})
		// Note: embeddings are available via m.embeddings[i] if needed
		_ = i // Acknowledge index is available
	}

	// Return mock IDs
	ids := make([]string, len(docs))
	for i := range docs {
		ids[i] = string(rune('a' + i))
	}
	return ids, nil
}

func (m *MockLangChainVectorStore) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
	// Return first numDocuments
	var result []schema.Document
	for i := 0; i < numDocuments && i < len(m.documents); i++ {
		result = append(result, schema.Document{
			PageContent: m.documents[i].PageContent,
			Metadata:    m.documents[i].Metadata,
			Score:       float32(1.0 - float64(i)*0.1), // Mock decreasing scores
		})
	}
	return result, nil
}

func TestLangChainVectorStore_AddDocuments(t *testing.T) {
	ctx := context.Background()

	// Create mock store
	mockStore := &MockLangChainVectorStore{}

	// Wrap with adapter
	adapter := NewLangChainVectorStore(mockStore)

	// Create test documents
	docs := []Document{
		{
			PageContent: "Test document 1",
			Metadata:    map[string]any{"source": "test1.txt"},
		},
		{
			PageContent: "Test document 2",
			Metadata:    map[string]any{"source": "test2.txt"},
		},
	}

	// Mock embeddings
	embeddings := [][]float64{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
	}

	// Add documents
	err := adapter.AddDocuments(ctx, docs, embeddings)
	require.NoError(t, err)

	// Verify documents were added
	assert.Equal(t, 2, len(mockStore.documents))
	assert.Equal(t, "Test document 1", mockStore.documents[0].PageContent)
	assert.Equal(t, "Test document 2", mockStore.documents[1].PageContent)
}

func TestLangChainVectorStore_SimilaritySearch(t *testing.T) {
	ctx := context.Background()

	// Create mock store with documents
	mockStore := &MockLangChainVectorStore{
		documents: []Document{
			{PageContent: "Doc 1", Metadata: map[string]any{"id": 1}},
			{PageContent: "Doc 2", Metadata: map[string]any{"id": 2}},
			{PageContent: "Doc 3", Metadata: map[string]any{"id": 3}},
		},
	}

	// Wrap with adapter
	adapter := NewLangChainVectorStore(mockStore)

	// Search
	results, err := adapter.SimilaritySearch(ctx, "test query", 2)
	require.NoError(t, err)

	// Verify results
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "Doc 1", results[0].PageContent)
	assert.Equal(t, "Doc 2", results[1].PageContent)
}

func TestLangChainVectorStore_SimilaritySearchWithScore(t *testing.T) {
	ctx := context.Background()

	// Create mock store with documents
	mockStore := &MockLangChainVectorStore{
		documents: []Document{
			{PageContent: "Doc 1", Metadata: map[string]any{"id": 1}},
			{PageContent: "Doc 2", Metadata: map[string]any{"id": 2}},
			{PageContent: "Doc 3", Metadata: map[string]any{"id": 3}},
		},
	}

	// Wrap with adapter
	adapter := NewLangChainVectorStore(mockStore)

	// Search with scores
	results, err := adapter.SimilaritySearchWithScore(ctx, "test query", 3)
	require.NoError(t, err)

	// Verify results
	assert.Equal(t, 3, len(results))

	// Check first result
	assert.Equal(t, "Doc 1", results[0].Document.PageContent)
	assert.InDelta(t, 1.0, results[0].Score, 0.0001)

	// Check second result
	assert.Equal(t, "Doc 2", results[1].Document.PageContent)
	assert.InDelta(t, 0.9, results[1].Score, 0.0001)

	// Check third result
	assert.Equal(t, "Doc 3", results[2].Document.PageContent)
	assert.InDelta(t, 0.8, results[2].Score, 0.0001)
}

func TestLangChainVectorStore_Integration(t *testing.T) {
	ctx := context.Background()

	// Create mock store
	mockStore := &MockLangChainVectorStore{}
	adapter := NewLangChainVectorStore(mockStore)

	// Add documents
	docs := []Document{
		{PageContent: "LangGraph is a library for building stateful applications", Metadata: map[string]any{"topic": "langgraph"}},
		{PageContent: "Go is a programming language", Metadata: map[string]any{"topic": "golang"}},
		{PageContent: "RAG combines retrieval and generation", Metadata: map[string]any{"topic": "rag"}},
	}

	embeddings := [][]float64{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
		{0.7, 0.8, 0.9},
	}

	err := adapter.AddDocuments(ctx, docs, embeddings)
	require.NoError(t, err)

	// Search
	results, err := adapter.SimilaritySearch(ctx, "what is langgraph", 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))

	// Search with scores
	scoredResults, err := adapter.SimilaritySearchWithScore(ctx, "programming", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, len(scoredResults))

	// Verify scores are in descending order
	for i := 0; i < len(scoredResults)-1; i++ {
		assert.GreaterOrEqual(t, scoredResults[i].Score, scoredResults[i+1].Score)
	}
}
