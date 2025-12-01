package prebuilt

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func TestSimpleTextSplitter(t *testing.T) {
	splitter := NewSimpleTextSplitter(100, 20)

	docs := []Document{
		{
			PageContent: "This is a long document that needs to be split into smaller chunks. " +
				"Each chunk should be around 100 characters with 20 characters overlap. " +
				"This helps maintain context between chunks.",
			Metadata: map[string]interface{}{
				"source": "test.txt",
			},
		},
	}

	chunks, err := splitter.SplitDocuments(docs)
	if err != nil {
		t.Fatalf("Failed to split documents: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	for i, chunk := range chunks {
		t.Logf("Chunk %d: %s", i, chunk.PageContent)
		if source, ok := chunk.Metadata["source"]; !ok || source != "test.txt" {
			t.Errorf("Chunk %d missing source metadata", i)
		}
	}
}

func TestInMemoryVectorStore(t *testing.T) {
	ctx := context.Background()
	embedder := NewMockEmbedder(128)
	vectorStore := NewInMemoryVectorStore(embedder)

	// Create test documents
	docs := []Document{
		{
			PageContent: "LangGraph is a library for building stateful, multi-actor applications with LLMs.",
			Metadata:    map[string]interface{}{"source": "doc1.txt"},
		},
		{
			PageContent: "RAG (Retrieval-Augmented Generation) combines retrieval with generation.",
			Metadata:    map[string]interface{}{"source": "doc2.txt"},
		},
		{
			PageContent: "Vector databases store embeddings for efficient similarity search.",
			Metadata:    map[string]interface{}{"source": "doc3.txt"},
		},
	}

	// Generate embeddings
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.PageContent
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to generate embeddings: %v", err)
	}

	// Add documents to vector store
	err = vectorStore.AddDocuments(ctx, docs, embeddings)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// Test similarity search
	query := "What is RAG?"
	results, err := vectorStore.SimilaritySearchWithScore(ctx, query, 2)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for i, result := range results {
		t.Logf("Result %d (score: %.4f): %s", i+1, result.Score, result.Document.PageContent)
	}
}

func TestSimpleReranker(t *testing.T) {
	ctx := context.Background()
	reranker := NewSimpleReranker()

	docs := []Document{
		{PageContent: "This document talks about cats and dogs."},
		{PageContent: "This document is about machine learning and AI."},
		{PageContent: "This document discusses cats, their behavior and habits."},
	}

	query := "cats behavior"
	results, err := reranker.Rerank(ctx, query, docs)
	if err != nil {
		t.Fatalf("Failed to rerank: %v", err)
	}

	if len(results) != len(docs) {
		t.Errorf("Expected %d results, got %d", len(docs), len(results))
	}

	// The document about cats behavior should be ranked highest
	for i, result := range results {
		t.Logf("Rank %d (score: %.4f): %s", i+1, result.Score, result.Document.PageContent)
	}
}

func TestVectorStoreRetriever(t *testing.T) {
	ctx := context.Background()
	embedder := NewMockEmbedder(128)
	vectorStore := NewInMemoryVectorStore(embedder)

	// Add test documents
	docs := []Document{
		{PageContent: "Document about graphs and networks."},
		{PageContent: "Document about machine learning."},
		{PageContent: "Document about graph neural networks."},
	}

	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.PageContent
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddDocuments(ctx, docs, embeddings)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// Create retriever
	retriever := NewVectorStoreRetriever(vectorStore, 2)

	// Test retrieval
	query := "graph networks"
	results, err := retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		t.Fatalf("Failed to retrieve documents: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for i, doc := range results {
		t.Logf("Retrieved %d: %s", i+1, doc.PageContent)
	}
}

func TestRAGPipelineBasic(t *testing.T) {
	// Skip if no OpenAI API key
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create LLM
	llm, err := openai.New(openai.WithModel("gpt-3.5-turbo"))
	if err != nil {
		t.Skip("Skipping test: OpenAI not configured")
	}

	// Create embedder and vector store
	embedder := NewMockEmbedder(128)
	vectorStore := NewInMemoryVectorStore(embedder)

	// Add test documents
	docs := []Document{
		{
			PageContent: "LangGraph is a library for building stateful, multi-actor applications with LLMs. It extends LangChain with graph-based workflows.",
			Metadata:    map[string]interface{}{"source": "langgraph_docs.txt"},
		},
		{
			PageContent: "RAG (Retrieval-Augmented Generation) is a technique that combines information retrieval with text generation to produce more accurate and contextual responses.",
			Metadata:    map[string]interface{}{"source": "rag_guide.txt"},
		},
	}

	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.PageContent
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddDocuments(ctx, docs, embeddings)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// Create retriever
	retriever := NewVectorStoreRetriever(vectorStore, 2)

	// Create RAG pipeline
	config := DefaultRAGConfig()
	config.Retriever = retriever
	config.LLM = llm

	pipeline := NewRAGPipeline(config)
	err = pipeline.BuildBasicRAG()
	if err != nil {
		t.Fatalf("Failed to build RAG pipeline: %v", err)
	}

	// Compile and run
	runnable, err := pipeline.Compile()
	if err != nil {
		t.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Test query
	result, err := runnable.Invoke(ctx, RAGState{
		Query: "What is LangGraph?",
	})
	if err != nil {
		t.Fatalf("Failed to run pipeline: %v", err)
	}

	finalState := result.(RAGState)
	t.Logf("Query: %s", finalState.Query)
	t.Logf("Answer: %s", finalState.Answer)

	if finalState.Answer == "" {
		t.Error("Expected non-empty answer")
	}
}

func TestRAGPipelineAdvanced(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM for testing
	mockLLM := &mockLLM{}

	// Create embedder and vector store
	embedder := NewMockEmbedder(128)
	vectorStore := NewInMemoryVectorStore(embedder)

	// Add test documents
	docs := []Document{
		{PageContent: "Document 1 about AI", Metadata: map[string]interface{}{"source": "doc1.txt"}},
		{PageContent: "Document 2 about ML", Metadata: map[string]interface{}{"source": "doc2.txt"}},
	}

	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.PageContent
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddDocuments(ctx, docs, embeddings)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// Create retriever and reranker
	retriever := NewVectorStoreRetriever(vectorStore, 2)
	reranker := NewSimpleReranker()

	// Create RAG pipeline with reranking
	config := DefaultRAGConfig()
	config.Retriever = retriever
	config.Reranker = reranker
	config.LLM = mockLLM
	config.UseReranking = true
	config.IncludeCitations = true

	pipeline := NewRAGPipeline(config)
	err = pipeline.BuildAdvancedRAG()
	if err != nil {
		t.Fatalf("Failed to build advanced RAG pipeline: %v", err)
	}

	// Compile and run
	runnable, err := pipeline.Compile()
	if err != nil {
		t.Fatalf("Failed to compile pipeline: %v", err)
	}

	result, err := runnable.Invoke(ctx, RAGState{
		Query: "What is AI?",
	})
	if err != nil {
		t.Fatalf("Failed to run pipeline: %v", err)
	}

	finalState := result.(RAGState)
	t.Logf("Query: %s", finalState.Query)
	t.Logf("Answer: %s", finalState.Answer)
	t.Logf("Citations: %v", finalState.Citations)

	if len(finalState.Citations) == 0 {
		t.Error("Expected citations to be included")
	}
}

// mockLLM is a simple mock LLM for testing
type mockLLM struct{}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: "This is a mock answer based on the provided context.",
			},
		},
	}, nil
}

func (m *mockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "This is a mock response.", nil
}
