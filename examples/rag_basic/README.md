# Basic RAG Example

This example demonstrates a basic Retrieval-Augmented Generation (RAG) pipeline using LangGraphGo.

## Overview

The basic RAG pattern follows these steps:
1. **Retrieve**: Find relevant documents from a vector store based on semantic similarity
2. **Generate**: Use an LLM to generate an answer based on the retrieved context

This is the simplest and most straightforward RAG implementation, suitable for:
- Quick prototyping
- Simple Q&A systems
- Applications with high-quality document collections

## Features

- Vector-based document retrieval using embeddings
- In-memory vector store for fast similarity search
- LLM-based answer generation with context
- Visualization of the RAG pipeline

## Running the Example

```bash
cd examples/rag_basic
go run main.go
```

## Key Components

- **Document Store**: In-memory vector store with mock embeddings
- **Retriever**: Vector store retriever that finds top-k similar documents
- **LLM**: DeepSeek-v3 for answer generation
- **Pipeline**: Basic RAG pipeline (Retrieve → Generate)

## Example Output

The example runs several queries and shows:
- Retrieved documents with sources
- Generated answers based on context
- Pipeline visualization in Mermaid format

## Customization

You can customize:
- Number of documents to retrieve (`TopK`)
- System prompt for the LLM
- Document corpus
- Embedding dimension
