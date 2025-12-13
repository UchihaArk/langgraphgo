package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create sample documents
	documents := []prebuilt.Document{
		{
			PageContent: "LangGraph is a library for building stateful, multi-actor applications with LLMs. " +
				"It extends LangChain Expression Language with the ability to coordinate multiple chains " +
				"(or actors) across multiple steps of computation in a cyclic manner.",
			Metadata: map[string]any{
				"source": "langgraph_intro.txt",
				"topic":  "LangGraph",
			},
		},
		{
			PageContent: "RAG (Retrieval-Augmented Generation) is a technique that combines information retrieval " +
				"with text generation. It retrieves relevant documents from a knowledge base and uses them " +
				"to augment the context provided to a language model for generation.",
			Metadata: map[string]any{
				"source": "rag_overview.txt",
				"topic":  "RAG",
			},
		},
		{
			PageContent: "Vector databases store embeddings, which are numerical representations of text. " +
				"They enable efficient similarity search by comparing vector distances. " +
				"Popular vector databases include Pinecone, Weaviate, and Chroma.",
			Metadata: map[string]any{
				"source": "vector_db.txt",
				"topic":  "Vector Databases",
			},
		},
		{
			PageContent: "Text embeddings are dense vector representations of text that capture semantic meaning. " +
				"Models like OpenAI's text-embedding-ada-002 or sentence transformers can generate these embeddings. " +
				"Similar texts have similar embeddings in the vector space.",
			Metadata: map[string]any{
				"source": "embeddings.txt",
				"topic":  "Embeddings",
			},
		},
	}

	// Create embedder and vector store
	embedder := prebuilt.NewMockEmbedder(128)
	vectorStore := prebuilt.NewInMemoryVectorStore(embedder)

	// Generate embeddings and add documents to vector store
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.PageContent
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddDocuments(ctx, documents, embeddings)
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
	config.SystemPrompt = "You are a helpful AI assistant. Answer the question based on the provided context. " +
		"If the context doesn't contain enough information to answer the question, say so."

	// Build basic RAG pipeline
	pipeline := prebuilt.NewRAGPipeline(config)
	err = pipeline.BuildBasicRAG()
	if err != nil {
		log.Fatalf("Failed to build RAG pipeline: %v", err)
	}

	// Compile the pipeline
	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Visualize the pipeline
	exporter := graph.NewExporter(pipeline.GetGraph())
	fmt.Println("=== RAG Pipeline Visualization (Mermaid) ===")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()

	// Test queries
	queries := []string{
		"What is LangGraph?",
		"How does RAG work?",
		"What are vector databases used for?",
	}

	for i, query := range queries {
		fmt.Printf("=== Query %d ===\n", i+1)
		fmt.Printf("Question: %s\n\n", query)

		result, err := runnable.Invoke(ctx, prebuilt.RAGState{
			Query: query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		finalState := result.(prebuilt.RAGState)

		fmt.Println("Retrieved Documents:")
		for j, doc := range finalState.Documents {
			source := "Unknown"
			if s, ok := doc.Metadata["source"]; ok {
				source = fmt.Sprintf("%v", s)
			}
			fmt.Printf("  [%d] %s\n", j+1, source)
			fmt.Printf("      %s...\n", truncate(doc.PageContent, 100))
		}

		fmt.Printf("\nAnswer: %s\n", finalState.Answer)
		fmt.Println("\n" + strings.Repeat("-", 80) + "\n")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
