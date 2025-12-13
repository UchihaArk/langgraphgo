package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/textsplitter"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== RAG with LangChain DocumentLoaders Example ===")

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Example 1: Load from Text
	fmt.Println("Example 1: Loading from Text Files")
	fmt.Println(strings.Repeat("-", 80))

	textContent := `LangGraph is a library for building stateful, multi-actor applications with LLMs.
It extends LangChain Expression Language with the ability to coordinate multiple chains
across multiple steps of computation in a cyclic manner. LangGraph is particularly useful
for building complex agent workflows and multi-agent systems.

Key features of LangGraph include:
- Stateful graph-based workflows
- Support for cycles and conditional edges
- Built-in checkpointing for persistence
- Human-in-the-loop capabilities
- Integration with LangChain components`

	// Create text loader using langchaingo
	textReader := strings.NewReader(textContent)
	textLoader := documentloaders.NewText(textReader)

	// Wrap with our adapter
	loader := prebuilt.NewLangChainDocumentLoader(textLoader)

	// Load documents
	docs, err := loader.Load(ctx)
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}

	fmt.Printf("Loaded %d document(s)\n", len(docs))
	for i, doc := range docs {
		fmt.Printf("Document %d: %d characters\n", i+1, len(doc.PageContent))
	}
	fmt.Println()

	// Example 2: Load and Split using LangChain's RecursiveCharacterTextSplitter
	fmt.Println("Example 2: Loading and Splitting with LangChain TextSplitter")
	fmt.Println(strings.Repeat("-", 80))

	// Create a new text loader
	textReader2 := strings.NewReader(textContent)
	textLoader2 := documentloaders.NewText(textReader2)
	loader2 := prebuilt.NewLangChainDocumentLoader(textLoader2)

	// Create LangChain's RecursiveCharacterTextSplitter
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(200),
		textsplitter.WithChunkOverlap(50),
	)

	// Load and split
	chunks, err := loader2.LoadAndSplit(ctx, splitter)
	if err != nil {
		log.Fatalf("Failed to load and split: %v", err)
	}

	fmt.Printf("Split into %d chunks\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("Chunk %d: %d characters\n", i+1, len(chunk.PageContent))
		fmt.Printf("  Preview: %s...\n", truncate(chunk.PageContent, 80))
	}
	fmt.Println()

	// Example 3: Build RAG Pipeline with LangChain Components
	fmt.Println("Example 3: Complete RAG Pipeline with LangChain Integration")
	fmt.Println(strings.Repeat("-", 80))

	// Create embedder and vector store
	embedder := prebuilt.NewMockEmbedder(128)
	vectorStore := prebuilt.NewInMemoryVectorStore(embedder)

	// Generate embeddings for chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.PageContent
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddDocuments(ctx, chunks, embeddings)
	if err != nil {
		log.Fatalf("Failed to add documents to vector store: %v", err)
	}

	// Create retriever
	retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 3)

	// Configure RAG pipeline
	config := prebuilt.DefaultRAGConfig()
	config.Retriever = retriever
	config.LLM = llm
	config.TopK = 3
	config.SystemPrompt = "You are a helpful AI assistant. Answer questions based on the provided context about LangGraph."

	// Build pipeline
	pipeline := prebuilt.NewRAGPipeline(config)
	err = pipeline.BuildBasicRAG()
	if err != nil {
		log.Fatalf("Failed to build RAG pipeline: %v", err)
	}

	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Visualize pipeline
	exporter := graph.NewExporter(pipeline.GetGraph())
	fmt.Println("Pipeline Visualization:")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()

	// Test queries
	queries := []string{
		"What is LangGraph?",
		"What are the key features of LangGraph?",
		"How does LangGraph support human-in-the-loop?",
	}

	for i, query := range queries {
		fmt.Printf("Query %d: %s\n", i+1, query)

		result, err := runnable.Invoke(ctx, prebuilt.RAGState{
			Query: query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		finalState := result.(prebuilt.RAGState)

		fmt.Printf("Retrieved %d documents:\n", len(finalState.Documents))
		for j, doc := range finalState.Documents {
			fmt.Printf("  [%d] %s\n", j+1, truncate(doc.PageContent, 100))
		}

		fmt.Printf("\nAnswer: %s\n", finalState.Answer)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println()
	}

	// Example 4: Load from CSV (if file exists)
	fmt.Println("Example 4: Loading from CSV (Optional)")
	fmt.Println(strings.Repeat("-", 80))

	csvContent := `title,content,category
LangGraph Basics,LangGraph is a library for building stateful applications,Tutorial
Advanced Features,LangGraph supports cycles and conditional edges,Tutorial
Integration,LangGraph integrates with LangChain components,Guide`

	csvReader := strings.NewReader(csvContent)
	csvLoader := documentloaders.NewCSV(csvReader)
	csvLoaderAdapter := prebuilt.NewLangChainDocumentLoader(csvLoader)

	csvDocs, err := csvLoaderAdapter.Load(ctx)
	if err != nil {
		log.Printf("Failed to load CSV: %v", err)
	} else {
		fmt.Printf("Loaded %d documents from CSV\n", len(csvDocs))
		for i, doc := range csvDocs {
			fmt.Printf("Document %d:\n", i+1)
			fmt.Printf("  Content: %s\n", doc.PageContent)
			fmt.Printf("  Metadata: %v\n", doc.Metadata)
		}
	}
	fmt.Println()

	// Example 5: Using LangChain TextSplitter with our interface
	fmt.Println("Example 5: Using LangChain TextSplitter Adapter")
	fmt.Println(strings.Repeat("-", 80))

	// Create our adapter for LangChain's text splitter
	lcSplitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(150),
		textsplitter.WithChunkOverlap(30),
	)
	splitterAdapter := prebuilt.NewLangChainTextSplitter(lcSplitter)

	// Use it with our Document type
	testDocs := []prebuilt.Document{
		{
			PageContent: textContent,
			Metadata:    map[string]any{"source": "test.txt"},
		},
	}

	splitDocs, err := splitterAdapter.SplitDocuments(testDocs)
	if err != nil {
		log.Printf("Failed to split: %v", err)
	} else {
		fmt.Printf("Split into %d chunks using LangChain splitter\n", len(splitDocs))
		for i, doc := range splitDocs {
			fmt.Printf("Chunk %d: %d chars, source: %v\n",
				i+1, len(doc.PageContent), doc.Metadata["source"])
		}
	}

	fmt.Println("\n=== Example Complete ===")
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
