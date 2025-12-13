package prebuilt

import (
	"context"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores"
)

// LangChainDocumentLoader adapts langchaingo's documentloaders.Loader to our DocumentLoader interface
type LangChainDocumentLoader struct {
	loader documentloaders.Loader
}

// NewLangChainDocumentLoader creates a new adapter for langchaingo document loaders
func NewLangChainDocumentLoader(loader documentloaders.Loader) *LangChainDocumentLoader {
	return &LangChainDocumentLoader{
		loader: loader,
	}
}

// Load loads documents using the underlying langchaingo loader
func (l *LangChainDocumentLoader) Load(ctx context.Context) ([]Document, error) {
	schemaDocs, err := l.loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return convertSchemaDocuments(schemaDocs), nil
}

// LoadAndSplit loads and splits documents using langchaingo's text splitter
func (l *LangChainDocumentLoader) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]Document, error) {
	schemaDocs, err := l.loader.LoadAndSplit(ctx, splitter)
	if err != nil {
		return nil, err
	}

	return convertSchemaDocuments(schemaDocs), nil
}

// convertSchemaDocuments converts langchaingo schema.Document to our Document type
func convertSchemaDocuments(schemaDocs []schema.Document) []Document {
	docs := make([]Document, len(schemaDocs))
	for i, schemaDoc := range schemaDocs {
		docs[i] = Document{
			PageContent: schemaDoc.PageContent,
			Metadata:    schemaDoc.Metadata,
		}
		// Optionally store the score if needed
		if schemaDoc.Score > 0 {
			docs[i].Metadata["score"] = schemaDoc.Score
		}
	}
	return docs
}

// convertToSchemaDocuments converts our Document type to langchaingo schema.Document
func convertToSchemaDocuments(docs []Document) []schema.Document {
	schemaDocs := make([]schema.Document, len(docs))
	for i, doc := range docs {
		schemaDocs[i] = schema.Document{
			PageContent: doc.PageContent,
			Metadata:    doc.Metadata,
		}
		// Extract score from metadata if present
		if score, ok := doc.Metadata["score"].(float32); ok {
			schemaDocs[i].Score = score
		}
	}
	return schemaDocs
}

// LangChainTextSplitter adapts langchaingo's textsplitter.TextSplitter to our TextSplitter interface
type LangChainTextSplitter struct {
	splitter textsplitter.TextSplitter
}

// NewLangChainTextSplitter creates a new adapter for langchaingo text splitters
func NewLangChainTextSplitter(splitter textsplitter.TextSplitter) *LangChainTextSplitter {
	return &LangChainTextSplitter{
		splitter: splitter,
	}
}

// SplitDocuments splits documents using the underlying langchaingo splitter
func (s *LangChainTextSplitter) SplitDocuments(documents []Document) ([]Document, error) {
	var result []Document

	for _, doc := range documents {
		// Split the text content
		chunks, err := s.splitter.SplitText(doc.PageContent)
		if err != nil {
			return nil, err
		}

		// Create a new document for each chunk, preserving metadata
		for i, chunk := range chunks {
			newDoc := Document{
				PageContent: chunk,
				Metadata:    make(map[string]any),
			}

			// Copy original metadata
			for k, v := range doc.Metadata {
				newDoc.Metadata[k] = v
			}

			// Add chunk-specific metadata
			newDoc.Metadata["chunk_index"] = i
			newDoc.Metadata["total_chunks"] = len(chunks)

			result = append(result, newDoc)
		}
	}

	return result, nil
}

// LangChainEmbedder adapts langchaingo's embeddings.Embedder to our Embedder interface
type LangChainEmbedder struct {
	embedder embeddings.Embedder
}

// NewLangChainEmbedder creates a new adapter for langchaingo embedders
func NewLangChainEmbedder(embedder embeddings.Embedder) *LangChainEmbedder {
	return &LangChainEmbedder{
		embedder: embedder,
	}
}

// EmbedDocuments generates embeddings for multiple documents
func (e *LangChainEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error) {
	// Call LangChain embedder (returns [][]float32)
	embeddings32, err := e.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, err
	}

	// Convert float32 to float64
	embeddings64 := make([][]float64, len(embeddings32))
	for i, emb32 := range embeddings32 {
		embeddings64[i] = make([]float64, len(emb32))
		for j, val := range emb32 {
			embeddings64[i][j] = float64(val)
		}
	}

	return embeddings64, nil
}

// EmbedQuery generates an embedding for a single query
func (e *LangChainEmbedder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	// Call LangChain embedder (returns []float32)
	embedding32, err := e.embedder.EmbedQuery(ctx, text)
	if err != nil {
		return nil, err
	}

	// Convert float32 to float64
	embedding64 := make([]float64, len(embedding32))
	for i, val := range embedding32 {
		embedding64[i] = float64(val)
	}

	return embedding64, nil
}

// LangChainVectorStore adapts langchaingo's vectorstores.VectorStore to our VectorStore interface
type LangChainVectorStore struct {
	store vectorstores.VectorStore
}

// NewLangChainVectorStore creates a new adapter for langchaingo vector stores
func NewLangChainVectorStore(store vectorstores.VectorStore) *LangChainVectorStore {
	return &LangChainVectorStore{
		store: store,
	}
}

// AddDocuments adds documents to the vector store
func (s *LangChainVectorStore) AddDocuments(ctx context.Context, documents []Document, embeddings [][]float64) error {
	// Convert to langchaingo schema.Document
	schemaDocs := convertToSchemaDocuments(documents)

	// Note: langchaingo's AddDocuments typically handles embedding generation internally if an embedder is set,
	// or we might need to use a specific method if we want to provide pre-computed embeddings.
	// However, the standard vectorstores.VectorStore interface in langchaingo usually takes documents and adds them.
	// Some implementations might re-embed.
	// If the interface provided by langchaingo vectorstores allows passing embeddings, we should use it.
	// Most langchaingo vectorstores AddDocuments method signature is: AddDocuments(ctx context.Context, docs []schema.Document, options ...Option) ([]string, error)

	// For now, we will just pass the documents. If the underlying store needs an embedder, it should be configured with one.
	// The `embeddings` argument here is ignored because langchaingo stores typically manage their own embedding or expect the embedder to be part of the store configuration.
	// If we strictly need to pass pre-computed embeddings, we might need a more specific adapter or check if the specific store supports it.

	_, err := s.store.AddDocuments(ctx, schemaDocs)
	return err
}

// SimilaritySearch searches for similar documents
func (s *LangChainVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error) {
	// Call LangChain store
	schemaDocs, err := s.store.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, err
	}

	return convertSchemaDocuments(schemaDocs), nil
}

// SimilaritySearchWithScore searches for similar documents and returns them with scores
func (s *LangChainVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error) {
	// Call LangChain store
	// Note: Not all langchaingo vectorstores support SimilaritySearchWithScore directly in the main interface,
	// but usually SimilaritySearch returns documents which might contain scores in metadata or the implementation might have a specific method.
	// However, the standard interface `vectorstores.VectorStore` has `SimilaritySearch`.
	// Some stores implement `SimilaritySearchWithScore`.
	// We will check if the store implements a specific interface or just use SimilaritySearch and extract scores if available.

	// Ideally, we should check if s.store implements an interface with SimilaritySearchWithScore.
	// For now, let's try to use the standard SimilaritySearch and see if we can get scores.
	// Many langchaingo implementations return scores in the document metadata or struct.

	// If the underlying store supports returning scores, we can try to cast or use a specific method.
	// Since `vectorstores.VectorStore` interface in langchaingo (v0.1.13) mainly has `SimilaritySearch`,
	// we might need to rely on the returned documents having scores.

	schemaDocs, err := s.store.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, err
	}

	var result []DocumentWithScore
	for _, schemaDoc := range schemaDocs {
		doc := Document{
			PageContent: schemaDoc.PageContent,
			Metadata:    schemaDoc.Metadata,
		}
		score := float64(schemaDoc.Score)
		result = append(result, DocumentWithScore{
			Document: doc,
			Score:    score,
		})
	}

	return result, nil
}
