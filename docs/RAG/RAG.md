# RAG (Retrieval-Augmented Generation) in LangGraphGo

This document describes the RAG interfaces and implementations in LangGraphGo, inspired by LangChain's RAG patterns.

## Overview

RAG (Retrieval-Augmented Generation) is a technique that combines information retrieval with text generation to produce more accurate, contextual, and grounded responses. LangGraphGo provides a flexible, interface-based RAG system that supports multiple implementation patterns.

## Core Interfaces

### Document

```go
type Document struct {
    PageContent string
    Metadata    map[string]any
}
```

Represents a document with content and metadata.

### DocumentLoader

```go
type DocumentLoader interface {
    Load(ctx context.Context) ([]Document, error)
}
```

Loads documents from various sources (files, databases, APIs, etc.).

### TextSplitter

```go
type TextSplitter interface {
    SplitDocuments(documents []Document) ([]Document, error)
}
```

Splits large documents into smaller chunks for better retrieval and processing.

### Embedder

```go
type Embedder interface {
    EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error)
    EmbedQuery(ctx context.Context, text string) ([]float64, error)
}
```

Generates vector embeddings for text, enabling semantic search.

### VectorStore

```go
type VectorStore interface {
    AddDocuments(ctx context.Context, documents []Document, embeddings [][]float64) error
    SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error)
    SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error)
}
```

Stores and retrieves document embeddings using similarity search.

### Retriever

```go
type Retriever interface {
    GetRelevantDocuments(ctx context.Context, query string) ([]Document, error)
}
```

Retrieves relevant documents for a query (abstracts over different retrieval methods).

### Reranker

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, documents []Document) ([]DocumentWithScore, error)
}
```

Re-scores retrieved documents to improve relevance ranking.

## RAG Pipeline Patterns

### 1. Basic RAG

**Flow**: Retrieve → Generate

The simplest RAG pattern:
- Retrieve top-k relevant documents
- Generate answer using LLM with retrieved context

**Use Cases**:
- Quick prototyping
- Simple Q&A systems
- High-quality document collections

**Example**:
```go
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.LLM = llm

pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildBasicRAG()
```

### 2. Advanced RAG

**Flow**: Retrieve → Rerank → Generate → Format Citations

Enhanced RAG with quality improvements:
- Document chunking for better granularity
- Reranking for improved relevance
- Citation generation for transparency

**Use Cases**:
- Production RAG systems
- Applications requiring high accuracy
- Systems needing source attribution

**Example**:
```go
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.Reranker = reranker
config.LLM = llm
config.UseReranking = true
config.IncludeCitations = true

pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildAdvancedRAG()
```

### 3. Conditional RAG

**Flow**: Retrieve → Rerank → Route (by relevance) → Generate

Intelligent routing based on relevance:
- Conditional edges based on relevance scores
- Fallback search for low-relevance queries
- Adaptive behavior for different query types

**Use Cases**:
- Hybrid search systems
- Variable query types
- Robust production systems

**Example**:
```go
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.Reranker = reranker
config.LLM = llm
config.UseReranking = true
config.UseFallback = true
config.ScoreThreshold = 0.7

pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildConditionalRAG()
```

## Provided Implementations

### SimpleTextSplitter

Splits text into chunks with configurable size and overlap:

```go
splitter := prebuilt.NewSimpleTextSplitter(
    chunkSize: 500,    // Characters per chunk
    chunkOverlap: 50,  // Overlap between chunks
)
chunks, err := splitter.SplitDocuments(documents)
```

**Best Practices**:
- Chunk size: 200-500 tokens for most use cases
- Overlap: 10-20% to maintain context

### InMemoryVectorStore

Simple in-memory vector store for development and testing:

```go
embedder := prebuilt.NewMockEmbedder(128)
vectorStore := prebuilt.NewInMemoryVectorStore(embedder)
vectorStore.AddDocuments(ctx, documents, embeddings)
results, err := vectorStore.SimilaritySearch(ctx, query, k)
```

**Note**: For production, integrate with real vector databases (Pinecone, Weaviate, Chroma, etc.)

### VectorStoreRetriever

Retriever implementation using a vector store:

```go
retriever := prebuilt.NewVectorStoreRetriever(vectorStore, topK)
docs, err := retriever.GetRelevantDocuments(ctx, query)
```

### SimpleReranker

Keyword-based reranker for improving retrieval quality:

```go
reranker := prebuilt.NewSimpleReranker()
rankedDocs, err := reranker.Rerank(ctx, query, documents)
```

**Note**: For production, consider cross-encoder models for better reranking.

### MockEmbedder

Deterministic embedder for testing:

```go
embedder := prebuilt.NewMockEmbedder(dimension)
embeddings, err := embedder.EmbedDocuments(ctx, texts)
```

**Note**: For production, use real embedding models (OpenAI, Cohere, sentence-transformers, etc.)

## RAG State

The RAG pipeline uses a typed state that flows through the graph:

```go
type RAGState struct {
    Query              string                // User query
    Documents          []Document            // Current documents
    RetrievedDocuments []Document            // Initially retrieved docs
    RankedDocuments    []DocumentWithScore   // Reranked docs with scores
    Context            string                // Formatted context for LLM
    Answer             string                // Generated answer
    Citations          []string              // Source citations
    Metadata           map[string]any // Additional metadata
}
```

## Configuration

RAG pipelines are configured using `RAGConfig`:

```go
type RAGConfig struct {
    // Retrieval configuration
    TopK            int     // Number of documents to retrieve
    ScoreThreshold  float64 // Minimum relevance score
    UseReranking    bool    // Whether to use reranking
    UseFallback     bool    // Whether to use fallback search
    
    // Generation configuration
    SystemPrompt    string  // System prompt for LLM
    IncludeCitations bool   // Whether to include citations
    MaxTokens       int     // Max tokens for generation
    Temperature     float64 // LLM temperature
    
    // Components
    Loader      DocumentLoader
    Splitter    TextSplitter
    Embedder    Embedder
    VectorStore VectorStore
    Retriever   Retriever
    Reranker    Reranker
    LLM         llms.Model
}
```

## Advanced Patterns

### Multi-Query RAG

Generate multiple query variations to improve retrieval:

```go
// Implement custom retriever that generates query variations
type MultiQueryRetriever struct {
    baseRetriever Retriever
    llm          llms.Model
}
```

### Hybrid Search

Combine vector search with keyword search:

```go
// Implement custom retriever that merges results
type HybridRetriever struct {
    vectorRetriever  Retriever
    keywordRetriever Retriever
}
```

### Parent Document Retrieval

Retrieve small chunks but provide larger context:

```go
// Store chunk-to-parent mapping in metadata
chunk.Metadata["parent_id"] = parentDoc.ID
```

### Contextual Compression

Compress retrieved documents to extract only relevant parts:

```go
// Implement custom node in pipeline
func compressContext(ctx context.Context, state any) (any, error) {
    // Use LLM to extract relevant parts
}
```

## Best Practices

### 1. Document Preparation

- **Clean text**: Remove noise, formatting artifacts
- **Chunk appropriately**: Balance context and precision
- **Add metadata**: Include source, date, category, etc.
- **Deduplicate**: Remove duplicate or near-duplicate content

### 2. Retrieval

- **Tune top-k**: Start with 3-5, adjust based on results
- **Use metadata filtering**: Filter by date, category, etc.
- **Monitor relevance**: Track retrieval quality metrics
- **Consider hybrid search**: Combine semantic and keyword search

### 3. Reranking

- **Always rerank**: Improves precision significantly
- **Use cross-encoders**: Better than bi-encoders for reranking
- **Limit reranking**: Only rerank top-N candidates (e.g., 20)

### 4. Generation

- **Clear instructions**: Specify how to use context
- **Cite sources**: Always include citations
- **Handle uncertainty**: Instruct LLM to acknowledge limitations
- **Control length**: Set appropriate max tokens

### 5. Evaluation

- **Test retrieval**: Measure precision@k, recall@k
- **Test generation**: Evaluate answer quality, factuality
- **Monitor latency**: Track end-to-end response time
- **Collect feedback**: Use human feedback to improve

## Examples

See the examples directory for complete implementations:

- `examples/rag_basic/` - Basic RAG pipeline
- `examples/rag_advanced/` - Advanced RAG with reranking and citations
- `examples/rag_conditional/` - Conditional RAG with routing
- `examples/rag_pipeline/` - Original RAG pipeline example

## Integration with LangChain

LangGraphGo provides seamless integration with the [langchaingo](https://github.com/tmc/langchaingo) ecosystem through adapter layers. This allows you to use any LangChain component with LangGraphGo's RAG pipeline.

### LangChain Adapters

LangGraphGo includes adapters for the following LangChain components:

#### 1. Document Loaders

Wrap any langchaingo document loader:

```go
import (
    "github.com/tmc/langchaingo/documentloaders"
    "github.com/smallnest/langgraphgo/prebuilt"
)

// Create LangChain loader
textLoader := documentloaders.NewText(reader)

// Wrap with adapter
loader := prebuilt.NewLangChainDocumentLoader(textLoader)

// Use in RAG pipeline
docs, err := loader.Load(ctx)
```

**Supported Loaders**:
- Text files
- CSV files
- PDF documents
- HTML pages
- Markdown files
- And more from langchaingo

#### 2. Text Splitters

Wrap any langchaingo text splitter:

```go
import (
    "github.com/tmc/langchaingo/textsplitter"
    "github.com/smallnest/langgraphgo/prebuilt"
)

// Create LangChain splitter
lcSplitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(500),
    textsplitter.WithChunkOverlap(50),
)

// Wrap with adapter
splitter := prebuilt.NewLangChainTextSplitter(lcSplitter)

// Use in RAG pipeline
chunks, err := splitter.SplitDocuments(documents)
```

**Supported Splitters**:
- RecursiveCharacter
- Token-based
- Markdown
- Code splitters

#### 3. Embeddings

Wrap any langchaingo embedder:

```go
import (
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/llms/openai"
    "github.com/smallnest/langgraphgo/prebuilt"
)

// Create LLM
llm, err := openai.New()

// Create LangChain embedder
lcEmbedder, err := embeddings.NewEmbedder(llm)

// Wrap with adapter
embedder := prebuilt.NewLangChainEmbedder(lcEmbedder)

// Use in RAG pipeline
embeddings, err := embedder.EmbedDocuments(ctx, texts)
```

**Supported Embedders**:
- OpenAI
- Cohere
- HuggingFace
- Local models

#### 4. Vector Stores

**NEW**: Wrap any langchaingo vector store:

```go
import (
    "github.com/tmc/langchaingo/vectorstores/chroma"
    "github.com/smallnest/langgraphgo/prebuilt"
)

// Create LangChain vector store
chromaStore, err := chroma.New(
    chroma.WithChromaURL("http://localhost:8000"),
    chroma.WithEmbedder(embedder),
    chroma.WithDistanceFunction("cosine"),
)

// Wrap with adapter
vectorStore := prebuilt.NewLangChainVectorStore(chromaStore)

// Use in RAG pipeline
err = vectorStore.AddDocuments(ctx, documents, embeddings)
results, err := vectorStore.SimilaritySearch(ctx, query, k)
```

**Supported Vector Stores**:
- **Chroma**: Open-source embedding database
- **Weaviate**: Cloud-native vector database
- **Pinecone**: Managed vector database service
- **Qdrant**: Vector similarity search engine
- **Milvus**: Distributed vector database
- **PGVector**: PostgreSQL extension for vectors
- And any other langchaingo vectorstore implementation

### Complete Integration Example

Here's a complete example using LangChain components:

```go
package main

import (
    "context"
    "github.com/smallnest/langgraphgo/prebuilt"
    "github.com/tmc/langchaingo/documentloaders"
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/llms/openai"
    "github.com/tmc/langchaingo/textsplitter"
    "github.com/tmc/langchaingo/vectorstores/chroma"
)

func main() {
    ctx := context.Background()
    
    // 1. Create LLM
    llm, _ := openai.New(
        openai.WithModel("gpt-4"),
    )
    
    // 2. Load documents with LangChain loader
    textLoader := documentloaders.NewText(reader)
    loader := prebuilt.NewLangChainDocumentLoader(textLoader)
    
    // 3. Split with LangChain splitter
    splitter := textsplitter.NewRecursiveCharacter(
        textsplitter.WithChunkSize(500),
        textsplitter.WithChunkOverlap(50),
    )
    chunks, _ := loader.LoadAndSplit(ctx, splitter)
    
    // 4. Create embedder
    lcEmbedder, _ := embeddings.NewEmbedder(llm)
    embedder := prebuilt.NewLangChainEmbedder(lcEmbedder)
    
    // 5. Create vector store
    chromaStore, _ := chroma.New(
        chroma.WithChromaURL("http://localhost:8000"),
        chroma.WithEmbedder(lcEmbedder),
    )
    vectorStore := prebuilt.NewLangChainVectorStore(chromaStore)
    
    // 6. Add documents
    texts := make([]string, len(chunks))
    for i, chunk := range chunks {
        texts[i] = chunk.PageContent
    }
    embeddings, _ := embedder.EmbedDocuments(ctx, texts)
    vectorStore.AddDocuments(ctx, chunks, embeddings)
    
    // 7. Build RAG pipeline
    retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 3)
    
    config := prebuilt.DefaultRAGConfig()
    config.Retriever = retriever
    config.LLM = llm
    
    pipeline := prebuilt.NewRAGPipeline(config)
    pipeline.BuildBasicRAG()
    runnable, _ := pipeline.Compile()
    
    // 8. Query
    result, _ := runnable.Invoke(ctx, prebuilt.RAGState{
        Query: "What is the main topic?",
    })
}
```

### Vector Store Setup Guides

#### Chroma

```bash
# Start Chroma server
docker run -p 8000:8000 chromadb/chroma

# Use in code
chromaStore, err := chroma.New(
    chroma.WithChromaURL("http://localhost:8000"),
    chroma.WithEmbedder(embedder),
)
```

#### Weaviate

```bash
# Start Weaviate
docker run -d \
  -p 8080:8080 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  semitechnologies/weaviate:latest

# Use in code
weaviateStore, err := weaviate.New(
    weaviate.WithScheme("http"),
    weaviate.WithHost("localhost:8080"),
    weaviate.WithEmbedder(embedder),
)
```

#### Pinecone

```bash
# Set API key
export PINECONE_API_KEY="your-api-key"

# Use in code
pineconeStore, err := pinecone.New(
    pinecone.WithAPIKey(os.Getenv("PINECONE_API_KEY")),
    pinecone.WithEnvironment("us-west1-gcp"),
    pinecone.WithIndexName("my-index"),
    pinecone.WithEmbedder(embedder),
)
```

### Benefits of LangChain Integration

1. **Ecosystem Access**: Use any component from the langchaingo ecosystem
2. **Production Ready**: Battle-tested vector databases and embedders
3. **Flexibility**: Easy to swap components without changing pipeline code
4. **Community Support**: Leverage the LangChain community and documentation
5. **Future Proof**: Automatically get new features from langchaingo updates

### Examples

See these examples for complete implementations:

- `examples/rag_with_langchain/` - Basic LangChain integration
- `examples/rag_langchain_vectorstore_example/` - VectorStore integration with multiple backends
- `examples/rag_chroma_example/` - Chroma-specific example

## Future Enhancements

Planned improvements:

1. **More retrievers**: BM25, TF-IDF, hybrid search
2. **Better rerankers**: Cross-encoder integration
3. **Query transformation**: Multi-query, HyDE, step-back
4. **Contextual compression**: LLM-based context extraction
5. **Evaluation tools**: Built-in metrics and testing
6. **Streaming**: Stream retrieved documents and generation

## References

- [LangChain RAG Tutorial](https://python.langchain.com/docs/tutorials/rag/)
- [RAG Best Practices](https://www.anthropic.com/index/contextual-retrieval)
- [Advanced RAG Techniques](https://arxiv.org/abs/2312.10997)
