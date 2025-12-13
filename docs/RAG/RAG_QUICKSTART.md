# RAG Quick Start Guide

This guide will help you get started with RAG in LangGraphGo quickly.

## 5-Minute Quick Start

### 1. Prepare Documents

```go
documents := []prebuilt.Document{
    {
        PageContent: "LangGraph is a library for building stateful, multi-actor applications.",
        Metadata: map[string]any{
            "source": "intro.txt",
        },
    },
    {
        PageContent: "RAG combines information retrieval with text generation.",
        Metadata: map[string]any{
            "source": "rag.txt",
        },
    },
}
```

### 2. Create Vector Store

```go
// Create embedder and vector store
embedder := prebuilt.NewMockEmbedder(128)
vectorStore := prebuilt.NewInMemoryVectorStore(embedder)

// Generate embeddings and add documents
texts := []string{documents[0].PageContent, documents[1].PageContent}
embeddings, _ := embedder.EmbedDocuments(ctx, texts)
vectorStore.AddDocuments(ctx, documents, embeddings)
```

### 3. Create Retriever

```go
retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 2)
```

### 4. Configure and Build RAG Pipeline

```go
// Initialize LLM
llm, _ := openai.New()

// Configure RAG
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.LLM = llm

// Build pipeline
pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildBasicRAG()
runnable, _ := pipeline.Compile()
```

### 5. Execute Query

```go
result, _ := runnable.Invoke(ctx, prebuilt.RAGState{
    Query: "What is LangGraph?",
})

finalState := result.(prebuilt.RAGState)
fmt.Printf("Answer: %s\n", finalState.Answer)
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/smallnest/langgraphgo/prebuilt"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    ctx := context.Background()

    // 1. Prepare documents
    documents := []prebuilt.Document{
        {
            PageContent: "LangGraph is a library for building stateful, multi-actor applications.",
            Metadata:    map[string]any{"source": "intro.txt"},
        },
    }

    // 2. Create vector store
    embedder := prebuilt.NewMockEmbedder(128)
    vectorStore := prebuilt.NewInMemoryVectorStore(embedder)
    
    texts := make([]string, len(documents))
    for i, doc := range documents {
        texts[i] = doc.PageContent
    }
    
    embeddings, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        log.Fatal(err)
    }
    
    err = vectorStore.AddDocuments(ctx, documents, embeddings)
    if err != nil {
        log.Fatal(err)
    }

    // 3. Create retriever
    retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 2)

    // 4. Configure RAG
    llm, err := openai.New(
        openai.WithModel("deepseek-v3"),
        openai.WithBaseURL("https://api.deepseek.com"),
    )
    if err != nil {
        log.Fatal(err)
    }

    config := prebuilt.DefaultRAGConfig()
    config.Retriever = retriever
    config.LLM = llm

    // 5. Build pipeline
    pipeline := prebuilt.NewRAGPipeline(config)
    err = pipeline.BuildBasicRAG()
    if err != nil {
        log.Fatal(err)
    }

    runnable, err := pipeline.Compile()
    if err != nil {
        log.Fatal(err)
    }

    // 6. Execute query
    result, err := runnable.Invoke(ctx, prebuilt.RAGState{
        Query: "What is LangGraph?",
    })
    if err != nil {
        log.Fatal(err)
    }

    finalState := result.(prebuilt.RAGState)
    fmt.Printf("Query: %s\n", finalState.Query)
    fmt.Printf("Answer: %s\n", finalState.Answer)
}
```

## Choosing a RAG Pattern

### Basic RAG - For Quick Prototyping
```go
pipeline.BuildBasicRAG()
```
- Simplest approach
- Retrieve → Generate
- Good for high-quality document collections

### Advanced RAG - For Production
```go
config.UseReranking = true
config.IncludeCitations = true
pipeline.BuildAdvancedRAG()
```
- Includes reranking
- Automatic citations
- Higher accuracy

### Conditional RAG - For Complex Scenarios
```go
config.UseReranking = true
config.UseFallback = true
config.ScoreThreshold = 0.7
pipeline.BuildConditionalRAG()
```
- Intelligent routing
- Fallback search
- Adaptive behavior

## Common Configurations

### Adjust Retrieval Count
```go
config.TopK = 5  // Retrieve top 5 documents
```

### Set Relevance Threshold
```go
config.ScoreThreshold = 0.7  // Minimum relevance score
```

### Customize System Prompt
```go
config.SystemPrompt = "You are a professional AI assistant. Answer based on the provided context."
```

### Enable Citations
```go
config.IncludeCitations = true
```

## Document Chunking

For large documents, use a text splitter:

```go
splitter := prebuilt.NewSimpleTextSplitter(500, 50)
chunks, _ := splitter.SplitDocuments(documents)
```

Parameters:
- `500`: Characters per chunk
- `50`: Overlap between chunks

## Next Steps

1. **Explore Examples**:
   - `examples/rag_basic/` - Basic example
   - `examples/rag_advanced/` - Advanced example
   - `examples/rag_conditional/` - Conditional example

2. **Read Documentation**:
   - `docs/RAG.md` - Complete English documentation
   - `docs/RAG_CN.md` - Complete Chinese documentation

3. **Customize Components**:
   - Implement your own `Retriever`
   - Implement your own `Reranker`
   - Integrate real vector databases

## FAQ

### Q: How to use real embedding models?

A: Integrate LangChain embedding models:
```go
import "github.com/tmc/langchaingo/embeddings"

embedder := embeddings.NewOpenAI()
```

### Q: How to use real vector databases?

A: Implement the `VectorStore` interface or use LangChain vector stores:
```go
import "github.com/tmc/langchaingo/vectorstores"

vectorStore := vectorstores.NewChroma(...)
```

### Q: How to improve retrieval quality?

A: 
1. Use document chunking
2. Enable reranking
3. Tune TopK and threshold
4. Use better embedding models

### Q: How to add metadata filtering?

A: Add metadata to documents, then filter in custom retriever:
```go
doc.Metadata["category"] = "technical"
doc.Metadata["date"] = "2024-01-01"
```

## Performance Optimization Tips

1. **Batch Processing**: Generate all embeddings at once
2. **Caching**: Cache results for common queries
3. **Async**: Process multiple queries in parallel
4. **Indexing**: Use professional vector databases
5. **Limits**: Set reasonable TopK values

## Getting Help

- Check example code
- Read complete documentation
- Review test files for more usage patterns
