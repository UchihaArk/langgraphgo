package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== RAG with LangChain Embeddings Example ===")

	// Example 1: Using OpenAI Embeddings
	fmt.Println("Example 1: Using OpenAI Embeddings")
	fmt.Println(strings.Repeat("-", 80))

	// Create OpenAI embeddings client
	// Note: Requires OPENAI_API_KEY environment variable
	var openaiEmbedder *embeddings.EmbedderImpl
	openaiLLM, err := openai.New()
	if err != nil {
		log.Printf("Warning: Failed to create OpenAI LLM: %v", err)
		log.Println("Skipping OpenAI example. Set OPENAI_API_KEY to use OpenAI embeddings.")
	} else {
		openaiEmbedder, err = embeddings.NewEmbedder(openaiLLM)
		if err != nil {
			log.Printf("Warning: Failed to create OpenAI embedder: %v", err)
		} else {
			// Wrap with our adapter
			embedder := prebuilt.NewLangChainEmbedder(openaiEmbedder)

			// Test embedding a single query
			query := "What is machine learning?"
			queryEmb, err := embedder.EmbedQuery(ctx, query)
			if err != nil {
				log.Printf("Failed to embed query: %v", err)
			} else {
				fmt.Printf("Query: %s\n", query)
				fmt.Printf("Embedding dimension: %d\n", len(queryEmb))
				fmt.Printf("First 5 values: %.4f, %.4f, %.4f, %.4f, %.4f\n",
					queryEmb[0], queryEmb[1], queryEmb[2], queryEmb[3], queryEmb[4])
			}

			// Test embedding multiple documents
			texts := []string{
				"Machine learning is a subset of artificial intelligence.",
				"Deep learning uses neural networks with multiple layers.",
				"Natural language processing deals with text and language.",
			}

			docsEmb, err := embedder.EmbedDocuments(ctx, texts)
			if err != nil {
				log.Printf("Failed to embed documents: %v", err)
			} else {
				fmt.Printf("\nEmbedded %d documents\n", len(docsEmb))
				for i, emb := range docsEmb {
					fmt.Printf("Document %d: dimension=%d\n", i+1, len(emb))
				}
			}
		}
	}
	fmt.Println()

	// Example 2: Complete RAG Pipeline with Real Embeddings
	fmt.Println("Example 2: Complete RAG Pipeline with LangChain Embeddings")
	fmt.Println(strings.Repeat("-", 80))

	// For demonstration, we'll use mock embeddings if OpenAI is not available
	var ragEmbedder prebuilt.Embedder
	if openaiEmbedder != nil {
		ragEmbedder = prebuilt.NewLangChainEmbedder(openaiEmbedder)
		fmt.Println("Using OpenAI embeddings for RAG pipeline")
	} else {
		ragEmbedder = prebuilt.NewMockEmbedder(1536) // OpenAI ada-002 dimension
		fmt.Println("Using mock embeddings for RAG pipeline (set OPENAI_API_KEY for real embeddings)")
	}

	// Create sample documents
	documents := []prebuilt.Document{
		{
			PageContent: "LangGraph is a library for building stateful, multi-actor applications with LLMs. " +
				"It provides graph-based workflows with support for cycles and conditional edges.",
			Metadata: map[string]any{
				"source":   "langgraph_intro.txt",
				"category": "Framework",
			},
		},
		{
			PageContent: "RAG (Retrieval-Augmented Generation) combines information retrieval with text generation. " +
				"It uses embeddings to find relevant documents and provides them as context to the LLM.",
			Metadata: map[string]any{
				"source":   "rag_overview.txt",
				"category": "Technique",
			},
		},
		{
			PageContent: "Vector embeddings are numerical representations of text that capture semantic meaning. " +
				"Similar texts have similar embeddings, enabling semantic search and similarity matching.",
			Metadata: map[string]any{
				"source":   "embeddings_guide.txt",
				"category": "Concept",
			},
		},
		{
			PageContent: "OpenAI's text-embedding-ada-002 model generates 1536-dimensional embeddings. " +
				"It's optimized for semantic search and provides high-quality representations of text.",
			Metadata: map[string]any{
				"source":   "openai_embeddings.txt",
				"category": "Model",
			},
		},
	}

	fmt.Printf("Created %d documents\n", len(documents))

	// Create vector store with LangChain embeddings
	vectorStore := prebuilt.NewInMemoryVectorStore(ragEmbedder)

	// Generate embeddings and add documents
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.PageContent
	}

	fmt.Println("Generating embeddings for documents...")
	embeds, err := ragEmbedder.EmbedDocuments(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddDocuments(ctx, documents, embeds)
	if err != nil {
		log.Fatalf("Failed to add documents to vector store: %v", err)
	}
	fmt.Printf("Added %d documents to vector store\n\n", len(documents))

	// Create retriever
	retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 2)

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Configure RAG pipeline
	config := prebuilt.DefaultRAGConfig()
	config.Retriever = retriever
	config.LLM = llm
	config.TopK = 2
	config.SystemPrompt = "You are a helpful AI assistant. Answer questions based on the provided context."

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
		"How do embeddings work?",
		"What is RAG?",
	}

	for i, query := range queries {
		fmt.Printf("=== Query %d ===\n", i+1)
		fmt.Printf("Question: %s\n", query)

		// Show query embedding info
		queryEmb, err := ragEmbedder.EmbedQuery(ctx, query)
		if err != nil {
			log.Printf("Failed to embed query: %v", err)
		} else {
			fmt.Printf("Query embedding dimension: %d\n", len(queryEmb))
		}

		result, err := runnable.Invoke(ctx, prebuilt.RAGState{
			Query: query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		finalState := result.(prebuilt.RAGState)

		fmt.Printf("\nRetrieved %d documents:\n", len(finalState.Documents))
		for j, doc := range finalState.Documents {
			source := "Unknown"
			if s, ok := doc.Metadata["source"]; ok {
				source = fmt.Sprintf("%v", s)
			}
			fmt.Printf("  [%d] %s\n", j+1, source)
			fmt.Printf("      %s\n", truncate(doc.PageContent, 100))
		}

		fmt.Printf("\nAnswer: %s\n", finalState.Answer)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println()
	}

	// Example 3: Comparing Different Embeddings
	fmt.Println("Example 3: Embedding Similarity Comparison")
	fmt.Println(strings.Repeat("-", 80))

	testTexts := []string{
		"Machine learning and artificial intelligence",
		"Deep learning neural networks",
		"The weather is sunny today",
	}

	fmt.Println("Embedding test texts...")
	testEmbeds, err := ragEmbedder.EmbedDocuments(ctx, testTexts)
	if err != nil {
		log.Printf("Failed to embed test texts: %v", err)
	} else {
		// Calculate cosine similarities
		fmt.Println("\nCosine Similarities:")
		for i := 0; i < len(testTexts); i++ {
			for j := i + 1; j < len(testTexts); j++ {
				similarity := cosineSimilarity(testEmbeds[i], testEmbeds[j])
				fmt.Printf("Text %d vs Text %d: %.4f\n", i+1, j+1, similarity)
				fmt.Printf("  \"%s\"\n", truncate(testTexts[i], 50))
				fmt.Printf("  \"%s\"\n", truncate(testTexts[j], 50))
			}
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

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
