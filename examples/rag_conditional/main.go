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
	llm, err := openai.New(
		openai.WithModel("deepseek-v3"),
		openai.WithBaseURL("https://api.deepseek.com"),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create document corpus focused on specific topics
	documents := []prebuilt.Document{
		{
			PageContent: "LangGraph provides built-in support for checkpointing, which allows you to save and " +
				"restore the state of your graph execution. This is crucial for long-running workflows, " +
				"error recovery, and implementing human-in-the-loop patterns.",
			Metadata: map[string]interface{}{
				"source": "langgraph_checkpointing.txt",
				"topic":  "Checkpointing",
			},
		},
		{
			PageContent: "The StateGraph in LangGraph allows you to define complex workflows with typed state. " +
				"You can add nodes, edges, and conditional edges to create sophisticated control flow. " +
				"The graph compiles into a runnable that can be invoked with initial state.",
			Metadata: map[string]interface{}{
				"source": "langgraph_stategraph.txt",
				"topic":  "StateGraph",
			},
		},
		{
			PageContent: "Human-in-the-loop workflows allow AI systems to pause execution and request human input " +
				"or approval before continuing. LangGraph supports this through interrupts and the Command API, " +
				"enabling you to build interactive AI applications.",
			Metadata: map[string]interface{}{
				"source": "langgraph_hitl.txt",
				"topic":  "Human-in-the-Loop",
			},
		},
	}

	// Create embedder and vector store
	embedder := prebuilt.NewMockEmbedder(128)
	vectorStore := prebuilt.NewInMemoryVectorStore(embedder)

	// Generate embeddings and add documents
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

	// Create retriever and reranker
	retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 2)
	reranker := prebuilt.NewSimpleReranker()

	// Configure conditional RAG pipeline
	config := prebuilt.DefaultRAGConfig()
	config.Retriever = retriever
	config.Reranker = reranker
	config.LLM = llm
	config.TopK = 2
	config.ScoreThreshold = 0.5 // Documents must score above this to skip fallback
	config.UseReranking = true
	config.UseFallback = true
	config.IncludeCitations = true
	config.SystemPrompt = "You are a helpful AI assistant specializing in LangGraph. " +
		"Answer questions based on the provided context. If the context is insufficient, " +
		"acknowledge this and provide general guidance."

	// Build conditional RAG pipeline
	pipeline := prebuilt.NewRAGPipeline(config)
	err = pipeline.BuildConditionalRAG()
	if err != nil {
		log.Fatalf("Failed to build conditional RAG pipeline: %v", err)
	}

	// Compile the pipeline
	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Visualize the pipeline
	exporter := graph.NewExporter(pipeline.GetGraph())
	fmt.Println("=== Conditional RAG Pipeline Visualization (Mermaid) ===")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()
	fmt.Println("This pipeline uses conditional routing:")
	fmt.Println("- If relevance score >= 0.5: proceed to generation")
	fmt.Println("- If relevance score < 0.5: trigger fallback search")
	fmt.Println()

	// Test queries - mix of relevant and less relevant queries
	queries := []struct {
		question string
		expected string
	}{
		{
			question: "How does checkpointing work in LangGraph?",
			expected: "high relevance - should skip fallback",
		},
		{
			question: "What is the StateGraph?",
			expected: "high relevance - should skip fallback",
		},
		{
			question: "How do I implement human-in-the-loop?",
			expected: "high relevance - should skip fallback",
		},
		{
			question: "What is the weather like today?",
			expected: "low relevance - should trigger fallback",
		},
	}

	for i, q := range queries {
		fmt.Printf("=== Query %d ===\n", i+1)
		fmt.Printf("Question: %s\n", q.question)
		fmt.Printf("Expected: %s\n\n", q.expected)

		result, err := runnable.Invoke(ctx, prebuilt.RAGState{
			Query: q.question,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		finalState := result.(prebuilt.RAGState)

		fmt.Println("Retrieved Documents:")
		for j, doc := range finalState.RetrievedDocuments {
			source := "Unknown"
			if s, ok := doc.Metadata["source"]; ok {
				source = fmt.Sprintf("%v", s)
			}
			fmt.Printf("  [%d] %s\n", j+1, source)
		}

		if len(finalState.RankedDocuments) > 0 {
			fmt.Printf("\nRelevance Scores (after reranking):\n")
			for j, rd := range finalState.RankedDocuments {
				fmt.Printf("  [%d] Score: %.4f", j+1, rd.Score)
				if rd.Score >= config.ScoreThreshold {
					fmt.Printf(" ✓ (above threshold)")
				} else {
					fmt.Printf(" ✗ (below threshold)")
				}
				fmt.Println()
			}

			topScore := finalState.RankedDocuments[0].Score
			if topScore >= config.ScoreThreshold {
				fmt.Printf("\n→ High relevance detected (%.4f >= %.2f)\n", topScore, config.ScoreThreshold)
				fmt.Println("→ Proceeding directly to generation")
			} else {
				fmt.Printf("\n→ Low relevance detected (%.4f < %.2f)\n", topScore, config.ScoreThreshold)
				fmt.Println("→ Triggering fallback search")
			}
		}

		// Check if fallback was used
		if finalState.Metadata != nil {
			if fallbackUsed, ok := finalState.Metadata["fallback_used"]; ok && fallbackUsed.(bool) {
				fmt.Println("→ Fallback search was executed")
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

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Println("This example demonstrates conditional RAG with:")
	fmt.Println("1. Document retrieval from vector store")
	fmt.Println("2. Reranking to score relevance")
	fmt.Println("3. Conditional routing based on relevance threshold")
	fmt.Println("4. Fallback search for low-relevance queries")
	fmt.Println("5. Citation generation for transparency")
}
