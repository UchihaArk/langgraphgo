package prebuilt

import (
	"context"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
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
				Metadata:    make(map[string]interface{}),
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
