package prebuilt

import (
	"context"
	"fmt"
	"math"
	"strings"
)

// SimpleTextSplitter splits text into chunks of a given size
type SimpleTextSplitter struct {
	ChunkSize    int
	ChunkOverlap int
	Separator    string
}

// NewSimpleTextSplitter creates a new SimpleTextSplitter
func NewSimpleTextSplitter(chunkSize, chunkOverlap int) *SimpleTextSplitter {
	return &SimpleTextSplitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Separator:    "\n\n",
	}
}

// SplitDocuments splits documents into smaller chunks
func (s *SimpleTextSplitter) SplitDocuments(documents []Document) ([]Document, error) {
	var result []Document

	for _, doc := range documents {
		chunks := s.splitText(doc.PageContent)
		for i, chunk := range chunks {
			newDoc := Document{
				PageContent: chunk,
				Metadata:    make(map[string]any),
			}

			// Copy metadata
			for k, v := range doc.Metadata {
				newDoc.Metadata[k] = v
			}

			// Add chunk metadata
			newDoc.Metadata["chunk_index"] = i
			newDoc.Metadata["total_chunks"] = len(chunks)

			result = append(result, newDoc)
		}
	}

	return result, nil
}

func (s *SimpleTextSplitter) splitText(text string) []string {
	if len(text) <= s.ChunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + s.ChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at a separator
		if end < len(text) {
			lastSep := strings.LastIndex(text[start:end], s.Separator)
			if lastSep > 0 {
				end = start + lastSep + len(s.Separator)
			}
		}

		chunks = append(chunks, strings.TrimSpace(text[start:end]))

		nextStart := end - s.ChunkOverlap
		if nextStart <= start {
			// If overlap would cause us to get stuck or move backwards (because the chunk was small),
			// just move forward to the end of the current chunk.
			nextStart = end
		}

		start = nextStart
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// InMemoryVectorStore is a simple in-memory vector store implementation
type InMemoryVectorStore struct {
	documents  []Document
	embeddings [][]float64
	embedder   Embedder
}

// NewInMemoryVectorStore creates a new InMemoryVectorStore
func NewInMemoryVectorStore(embedder Embedder) *InMemoryVectorStore {
	return &InMemoryVectorStore{
		documents:  make([]Document, 0),
		embeddings: make([][]float64, 0),
		embedder:   embedder,
	}
}

// AddDocuments adds documents with their embeddings to the store
func (s *InMemoryVectorStore) AddDocuments(ctx context.Context, documents []Document, embeddings [][]float64) error {
	if len(documents) != len(embeddings) {
		return fmt.Errorf("number of documents (%d) must match number of embeddings (%d)", len(documents), len(embeddings))
	}

	s.documents = append(s.documents, documents...)
	s.embeddings = append(s.embeddings, embeddings...)

	return nil
}

// SimilaritySearch performs similarity search and returns top k documents
func (s *InMemoryVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error) {
	results, err := s.SimilaritySearchWithScore(ctx, query, k)
	if err != nil {
		return nil, err
	}

	docs := make([]Document, len(results))
	for i, r := range results {
		docs[i] = r.Document
	}

	return docs, nil
}

// SimilaritySearchWithScore performs similarity search and returns documents with scores
func (s *InMemoryVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error) {
	if len(s.documents) == 0 {
		return nil, fmt.Errorf("no documents in vector store")
	}

	// Generate query embedding
	queryEmbedding, err := s.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Calculate similarities
	type docScore struct {
		index int
		score float64
	}

	scores := make([]docScore, len(s.documents))
	for i, docEmb := range s.embeddings {
		similarity := cosineSimilarity(queryEmbedding, docEmb)
		scores[i] = docScore{index: i, score: similarity}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Return top k
	if k > len(scores) {
		k = len(scores)
	}

	results := make([]DocumentWithScore, k)
	for i := 0; i < k; i++ {
		results[i] = DocumentWithScore{
			Document: s.documents[scores[i].index],
			Score:    scores[i].score,
		}
	}

	return results, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SimpleReranker is a simple reranker that scores documents based on keyword matching
type SimpleReranker struct {
	// Can be extended with more sophisticated reranking logic
}

// NewSimpleReranker creates a new SimpleReranker
func NewSimpleReranker() *SimpleReranker {
	return &SimpleReranker{}
}

// Rerank reranks documents based on query relevance
func (r *SimpleReranker) Rerank(ctx context.Context, query string, documents []Document) ([]DocumentWithScore, error) {
	queryTerms := strings.Fields(strings.ToLower(query))

	type docScore struct {
		doc   Document
		score float64
	}

	scores := make([]docScore, len(documents))
	for i, doc := range documents {
		content := strings.ToLower(doc.PageContent)

		// Simple scoring: count query term occurrences
		var score float64
		for _, term := range queryTerms {
			score += float64(strings.Count(content, term))
		}

		// Normalize by document length
		if len(content) > 0 {
			score = score / float64(len(content)) * 1000
		}

		scores[i] = docScore{doc: doc, score: score}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	results := make([]DocumentWithScore, len(scores))
	for i, s := range scores {
		results[i] = DocumentWithScore{
			Document: s.doc,
			Score:    s.score,
		}
	}

	return results, nil
}

// StaticDocumentLoader loads documents from a static list
type StaticDocumentLoader struct {
	Documents []Document
}

// NewStaticDocumentLoader creates a new StaticDocumentLoader
func NewStaticDocumentLoader(documents []Document) *StaticDocumentLoader {
	return &StaticDocumentLoader{
		Documents: documents,
	}
}

// Load returns the static list of documents
func (l *StaticDocumentLoader) Load(ctx context.Context) ([]Document, error) {
	return l.Documents, nil
}

// MockEmbedder is a simple mock embedder for testing
type MockEmbedder struct {
	Dimension int
}

// NewMockEmbedder creates a new MockEmbedder
func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{
		Dimension: dimension,
	}
}

// EmbedDocuments generates mock embeddings for documents
func (e *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i, text := range texts {
		embeddings[i] = e.generateEmbedding(text)
	}
	return embeddings, nil
}

// EmbedQuery generates a mock embedding for a query
func (e *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.generateEmbedding(text), nil
}

func (e *MockEmbedder) generateEmbedding(text string) []float64 {
	// Simple deterministic embedding based on text content
	embedding := make([]float64, e.Dimension)

	for i := 0; i < e.Dimension; i++ {
		var sum float64
		for j, char := range text {
			sum += float64(char) * float64(i+j+1)
		}
		embedding[i] = math.Sin(sum / 1000.0)
	}

	// Normalize
	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}
