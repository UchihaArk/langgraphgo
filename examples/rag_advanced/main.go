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

	fmt.Println("Initializing LLM...")
	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	fmt.Println("LLM Initialized.")

	// Create a larger document corpus
	documents := []prebuilt.Document{
		{
			PageContent: "LangGraph is a library for building stateful, multi-actor applications with LLMs. " +
				"It extends LangChain Expression Language with the ability to coordinate multiple chains " +
				"across multiple steps of computation in a cyclic manner. LangGraph is particularly useful " +
				"for building complex agent workflows and multi-agent systems.",
			Metadata: map[string]interface{}{
				"source":   "langgraph_intro.txt",
				"topic":    "LangGraph",
				"category": "Framework",
			},
		},
		{
			PageContent: "RAG (Retrieval-Augmented Generation) is a technique that combines information retrieval " +
				"with text generation. It retrieves relevant documents from a knowledge base and uses them " +
				"to augment the context provided to a language model for generation. This approach helps " +
				"reduce hallucinations and provides more factual, grounded responses.",
			Metadata: map[string]interface{}{
				"source":   "rag_overview.txt",
				"topic":    "RAG",
				"category": "Technique",
			},
		},
		{
			PageContent: "Vector databases store embeddings, which are numerical representations of text. " +
				"They enable efficient similarity search by comparing vector distances using metrics like " +
				"cosine similarity or Euclidean distance. Popular vector databases include Pinecone, Weaviate, " +
				"Chroma, and Qdrant. These databases are essential for RAG systems.",
			Metadata: map[string]interface{}{
				"source":   "vector_db.txt",
				"topic":    "Vector Databases",
				"category": "Infrastructure",
			},
		},
		{
			PageContent: "Text embeddings are dense vector representations of text that capture semantic meaning. " +
				"Models like OpenAI's text-embedding-ada-002, sentence transformers, or Cohere embeddings " +
				"can generate these embeddings. Similar texts have similar embeddings in the vector space, " +
				"which enables semantic search.",
			Metadata: map[string]interface{}{
				"source":   "embeddings.txt",
				"topic":    "Embeddings",
				"category": "Technique",
			},
		},
		{
			PageContent: "Document reranking is a technique to improve retrieval quality by re-scoring retrieved " +
				"documents based on their relevance to the query. Cross-encoder models are often used for " +
				"reranking as they can better capture query-document interactions compared to bi-encoders " +
				"used for initial retrieval.",
			Metadata: map[string]interface{}{
				"source":   "reranking.txt",
				"topic":    "Reranking",
				"category": "Technique",
			},
		},
		{
			PageContent: "Multi-agent systems involve multiple AI agents working together to solve complex problems. " +
				"Each agent can have specialized roles and capabilities. LangGraph provides excellent support " +
				"for building multi-agent systems with its graph-based architecture and state management.",
			Metadata: map[string]interface{}{
				"source":   "multi_agent.txt",
				"topic":    "Multi-Agent",
				"category": "Architecture",
			},
		},
	}

	// Split documents into smaller chunks
	splitter := prebuilt.NewSimpleTextSplitter(200, 50)
	chunks, err := splitter.SplitDocuments(documents)
	if err != nil {
		log.Fatalf("Failed to split documents: %v", err)
	}

	fmt.Printf("Split %d documents into %d chunks\n\n", len(documents), len(chunks))

	// Create embedder and vector store
	embedder := prebuilt.NewMockEmbedder(256) // Higher dimension for better quality
	vectorStore := prebuilt.NewInMemoryVectorStore(embedder)

	// Generate embeddings and add chunks to vector store
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

	// Create retriever and reranker
	retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 5)
	reranker := prebuilt.NewSimpleReranker()

	// Configure advanced RAG pipeline with reranking and citations
	config := prebuilt.DefaultRAGConfig()
	config.Retriever = retriever
	config.Reranker = reranker
	config.LLM = llm
	config.TopK = 5
	config.UseReranking = true
	config.IncludeCitations = true
	config.SystemPrompt = "You are a knowledgeable AI assistant. Answer questions based on the provided context. " +
		"Always cite your sources using the document numbers provided. If the context doesn't contain " +
		"enough information, acknowledge the limitations and provide what you can."

	// Build advanced RAG pipeline
	pipeline := prebuilt.NewRAGPipeline(config)
	err = pipeline.BuildAdvancedRAG()
	if err != nil {
		log.Fatalf("Failed to build advanced RAG pipeline: %v", err)
	}

	// Compile the pipeline
	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Visualize the pipeline
	exporter := graph.NewExporter(pipeline.GetGraph())
	fmt.Println("=== Advanced RAG Pipeline Visualization (Mermaid) ===")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()

	// Test queries with more complex questions
	queries := []string{
		"What is LangGraph and how is it used in multi-agent systems?",
		"Explain the RAG technique and its benefits",
		"What role do vector databases play in RAG systems?",
		"How does document reranking improve retrieval quality?",
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

		fmt.Println("Retrieved and Reranked Documents:")
		for j, doc := range finalState.Documents {
			source := "Unknown"
			if s, ok := doc.Metadata["source"]; ok {
				source = fmt.Sprintf("%v", s)
			}
			category := "N/A"
			if c, ok := doc.Metadata["category"]; ok {
				category = fmt.Sprintf("%v", c)
			}
			fmt.Printf("  [%d] %s (Category: %s)\n", j+1, source, category)
			fmt.Printf("      %s\n", truncate(doc.PageContent, 120))
		}

		if len(finalState.RankedDocuments) > 0 {
			fmt.Printf("\nRelevance Scores:\n")
			for j, rd := range finalState.RankedDocuments {
				if j >= 3 {
					break // Show top 3 scores
				}
				fmt.Printf("  [%d] Score: %.4f\n", j+1, rd.Score)
			}
		}

		fmt.Printf("\nAnswer: %s\n", finalState.Answer)

		if len(finalState.Citations) > 0 {
			fmt.Println("\nCitations:")
			for _, citation := range finalState.Citations {
				fmt.Printf("  %s\n", citation)
			}
		}

		fmt.Println("\n" + strings.Repeat("=", 100) + "\n")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
