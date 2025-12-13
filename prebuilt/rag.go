package prebuilt

import (
	"context"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

// Document represents a document with content and metadata
type Document struct {
	PageContent string
	Metadata    map[string]any
}

// DocumentLoader loads documents from various sources
type DocumentLoader interface {
	Load(ctx context.Context) ([]Document, error)
}

// TextSplitter splits documents into smaller chunks
type TextSplitter interface {
	SplitDocuments(documents []Document) ([]Document, error)
}

// Embedder generates embeddings for text
type Embedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error)
	EmbedQuery(ctx context.Context, text string) ([]float64, error)
}

// VectorStore stores and retrieves document embeddings
type VectorStore interface {
	AddDocuments(ctx context.Context, documents []Document, embeddings [][]float64) error
	SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error)
	SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error)
}

// DocumentWithScore represents a document with its similarity score
type DocumentWithScore struct {
	Document Document
	Score    float64
}

// Retriever retrieves relevant documents for a query
type Retriever interface {
	GetRelevantDocuments(ctx context.Context, query string) ([]Document, error)
}

// Reranker reranks retrieved documents based on relevance
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []Document) ([]DocumentWithScore, error)
}

// RAGState represents the state flowing through a RAG pipeline
type RAGState struct {
	Query              string
	Documents          []Document
	RetrievedDocuments []Document
	RankedDocuments    []DocumentWithScore
	Context            string
	Answer             string
	Citations          []string
	Metadata           map[string]any
}

// RAGConfig configures a RAG pipeline
type RAGConfig struct {
	// Retrieval configuration
	TopK           int     // Number of documents to retrieve
	ScoreThreshold float64 // Minimum relevance score
	UseReranking   bool    // Whether to use reranking
	UseFallback    bool    // Whether to use fallback search

	// Generation configuration
	SystemPrompt     string
	IncludeCitations bool
	MaxTokens        int
	Temperature      float64

	// Components
	Loader      DocumentLoader
	Splitter    TextSplitter
	Embedder    Embedder
	VectorStore VectorStore
	Retriever   Retriever
	Reranker    Reranker
	LLM         llms.Model
}

// DefaultRAGConfig returns a default RAG configuration
func DefaultRAGConfig() *RAGConfig {
	return &RAGConfig{
		TopK:             4,
		ScoreThreshold:   0.7,
		UseReranking:     false,
		UseFallback:      false,
		SystemPrompt:     "You are a helpful assistant. Answer the question based on the provided context. If you cannot answer based on the context, say so.",
		IncludeCitations: true,
		MaxTokens:        1000,
		Temperature:      0.0,
	}
}

// RAGPipeline represents a complete RAG pipeline
type RAGPipeline struct {
	config *RAGConfig
	graph  *graph.StateGraph
}

// NewRAGPipeline creates a new RAG pipeline with the given configuration
func NewRAGPipeline(config *RAGConfig) *RAGPipeline {
	if config == nil {
		config = DefaultRAGConfig()
	}

	return &RAGPipeline{
		config: config,
		graph:  graph.NewStateGraph(),
	}
}

// BuildBasicRAG builds a basic RAG pipeline: Retrieve -> Generate
func (p *RAGPipeline) BuildBasicRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for basic RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for basic RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Build pipeline
	p.graph.SetEntryPoint("retrieve")
	p.graph.AddEdge("retrieve", "generate")
	p.graph.AddEdge("generate", graph.END)

	return nil
}

// BuildAdvancedRAG builds an advanced RAG pipeline: Retrieve -> Rerank -> Generate
func (p *RAGPipeline) BuildAdvancedRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for advanced RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for advanced RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add reranking node if enabled
	if p.config.UseReranking && p.config.Reranker != nil {
		p.graph.AddNode("rerank", "Document reranking node", p.rerankNode)
	}

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Add citation formatting node if enabled
	if p.config.IncludeCitations {
		p.graph.AddNode("format_citations", "Citation formatting node", p.formatCitationsNode)
	}

	// Build pipeline
	p.graph.SetEntryPoint("retrieve")

	if p.config.UseReranking && p.config.Reranker != nil {
		p.graph.AddEdge("retrieve", "rerank")
		p.graph.AddEdge("rerank", "generate")
	} else {
		p.graph.AddEdge("retrieve", "generate")
	}

	if p.config.IncludeCitations {
		p.graph.AddEdge("generate", "format_citations")
		p.graph.AddEdge("format_citations", graph.END)
	} else {
		p.graph.AddEdge("generate", graph.END)
	}

	return nil
}

// BuildConditionalRAG builds a RAG pipeline with conditional routing based on relevance
func (p *RAGPipeline) BuildConditionalRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for conditional RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for conditional RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add reranking node
	p.graph.AddNode("rerank", "Document reranking node", p.rerankNode)

	// Add fallback search node if enabled
	if p.config.UseFallback {
		p.graph.AddNode("fallback_search", "Fallback search node", p.fallbackSearchNode)
	}

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Add citation formatting node
	if p.config.IncludeCitations {
		p.graph.AddNode("format_citations", "Citation formatting node", p.formatCitationsNode)
	}

	// Build pipeline with conditional routing
	p.graph.SetEntryPoint("retrieve")
	p.graph.AddEdge("retrieve", "rerank")

	// Conditional edge based on relevance score
	p.graph.AddConditionalEdge("rerank", func(ctx context.Context, state any) string {
		ragState := state.(RAGState)
		if len(ragState.RankedDocuments) > 0 && ragState.RankedDocuments[0].Score >= p.config.ScoreThreshold {
			return "generate"
		}
		if p.config.UseFallback {
			return "fallback_search"
		}
		return "generate"
	})

	if p.config.UseFallback {
		p.graph.AddEdge("fallback_search", "generate")
	}

	if p.config.IncludeCitations {
		p.graph.AddEdge("generate", "format_citations")
		p.graph.AddEdge("format_citations", graph.END)
	} else {
		p.graph.AddEdge("generate", graph.END)
	}

	return nil
}

// Compile compiles the RAG pipeline into a runnable graph
func (p *RAGPipeline) Compile() (*graph.Runnable, error) {
	return p.graph.Compile()
}

// GetGraph returns the underlying graph for visualization
func (p *RAGPipeline) GetGraph() *graph.StateGraph {
	return p.graph
}

// Node implementations

func (p *RAGPipeline) retrieveNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	docs, err := p.config.Retriever.GetRelevantDocuments(ctx, ragState.Query)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	ragState.RetrievedDocuments = docs
	ragState.Documents = docs

	return ragState, nil
}

func (p *RAGPipeline) rerankNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	if p.config.Reranker == nil {
		// If no reranker, just assign scores based on order
		rankedDocs := make([]DocumentWithScore, len(ragState.RetrievedDocuments))
		for i, doc := range ragState.RetrievedDocuments {
			rankedDocs[i] = DocumentWithScore{
				Document: doc,
				Score:    1.0 - float64(i)*0.1, // Simple decreasing score
			}
		}
		ragState.RankedDocuments = rankedDocs
		return ragState, nil
	}

	rankedDocs, err := p.config.Reranker.Rerank(ctx, ragState.Query, ragState.RetrievedDocuments)
	if err != nil {
		return nil, fmt.Errorf("reranking failed: %w", err)
	}

	ragState.RankedDocuments = rankedDocs

	// Update documents with reranked order
	docs := make([]Document, len(rankedDocs))
	for i, rd := range rankedDocs {
		docs[i] = rd.Document
	}
	ragState.Documents = docs

	return ragState, nil
}

func (p *RAGPipeline) fallbackSearchNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	// Placeholder for fallback search (e.g., web search)
	// In a real implementation, this would call an external search API
	ragState.Metadata = map[string]any{
		"fallback_used": true,
	}

	return ragState, nil
}

func (p *RAGPipeline) generateNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	// Build context from retrieved documents
	var contextParts []string
	for i, doc := range ragState.Documents {
		source := "Unknown"
		if s, ok := doc.Metadata["source"]; ok {
			source = fmt.Sprintf("%v", s)
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] Source: %s\nContent: %s", i+1, source, doc.PageContent))
	}
	ragState.Context = strings.Join(contextParts, "\n\n")

	// Build prompt
	prompt := fmt.Sprintf("Context:\n%s\n\nQuestion: %s\n\nAnswer:", ragState.Context, ragState.Query)

	messages := []llms.MessageContent{
		llms.TextParts("system", p.config.SystemPrompt),
		llms.TextParts("human", prompt),
	}

	// Generate answer
	response, err := p.config.LLM.GenerateContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	if len(response.Choices) > 0 {
		ragState.Answer = response.Choices[0].Content
	}

	return ragState, nil
}

func (p *RAGPipeline) formatCitationsNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	// Extract citations from documents
	citations := make([]string, len(ragState.Documents))
	for i, doc := range ragState.Documents {
		source := "Unknown"
		if s, ok := doc.Metadata["source"]; ok {
			source = fmt.Sprintf("%v", s)
		}
		citations[i] = fmt.Sprintf("[%d] %s", i+1, source)
	}
	ragState.Citations = citations

	return ragState, nil
}

// VectorStoreRetriever implements Retriever using a VectorStore
type VectorStoreRetriever struct {
	VectorStore VectorStore
	TopK        int
}

// NewVectorStoreRetriever creates a new VectorStoreRetriever
func NewVectorStoreRetriever(vectorStore VectorStore, topK int) *VectorStoreRetriever {
	return &VectorStoreRetriever{
		VectorStore: vectorStore,
		TopK:        topK,
	}
}

// GetRelevantDocuments retrieves relevant documents for a query
func (r *VectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]Document, error) {
	return r.VectorStore.SimilaritySearch(ctx, query, r.TopK)
}
