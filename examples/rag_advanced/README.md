# Advanced RAG Example

This example demonstrates an advanced RAG pipeline with document chunking, reranking, and citation support.

## Overview

The advanced RAG pattern includes:
1. **Document Chunking**: Split large documents into smaller, manageable chunks
2. **Retrieve**: Find relevant chunks from a vector store
3. **Rerank**: Re-score retrieved documents for better relevance
4. **Generate**: Generate answers with the LLM
5. **Format Citations**: Add source citations for transparency

This implementation is suitable for:
- Production RAG systems
- Applications requiring high accuracy
- Systems needing source attribution
- Large document collections

## Features

- **Text Splitting**: Automatic document chunking with overlap
- **Reranking**: Improve retrieval quality with relevance scoring
- **Citations**: Automatic citation generation
- **Higher Quality**: Better embeddings and more sophisticated pipeline
- **Metadata Tracking**: Preserve document metadata through the pipeline

## Running the Example

```bash
cd examples/rag_advanced
go run main.go
```

## Key Components

- **TextSplitter**: Splits documents into chunks with configurable size and overlap
- **Reranker**: Re-scores documents based on query-document relevance
- **Vector Store**: Stores document chunks with embeddings
- **Retriever**: Retrieves top-k most relevant chunks
- **LLM**: DeepSeek-v3 for answer generation
- **Pipeline**: Advanced RAG (Retrieve → Rerank → Generate → Format Citations)

## Pipeline Flow

```
Query → Retrieve Chunks → Rerank by Relevance → Generate Answer → Add Citations → Result
```

## Example Output

The example demonstrates:
- Document chunking statistics
- Retrieved and reranked documents with scores
- Relevance scores for top documents
- Generated answers with citations
- Source attribution

## Best Practices

1. **Chunk Size**: Balance between context and precision (200-500 tokens)
2. **Chunk Overlap**: 10-20% overlap maintains context between chunks
3. **Reranking**: Improves precision by re-scoring with query-document interaction
4. **Citations**: Always include sources for factual accuracy and transparency

## Customization

You can customize:
- Chunk size and overlap
- Number of documents to retrieve and rerank
- Reranking algorithm
- Citation format
- System prompt and generation parameters
