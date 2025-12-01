# Conditional RAG Example

This example demonstrates a conditional RAG pipeline with dynamic routing based on relevance scores.

## Overview

The conditional RAG pattern uses intelligent routing:
1. **Retrieve**: Find potentially relevant documents
2. **Rerank**: Score documents for relevance
3. **Route**: Conditionally choose the next step based on relevance
   - High relevance → Generate answer directly
   - Low relevance → Trigger fallback search
4. **Generate**: Create answer with available context
5. **Format**: Add citations

This implementation is suitable for:
- Hybrid search systems
- Applications with variable query types
- Systems requiring fallback mechanisms
- Production systems needing robustness

## Features

- **Conditional Routing**: Dynamic path selection based on relevance scores
- **Relevance Threshold**: Configurable threshold for routing decisions
- **Fallback Search**: Alternative search when primary retrieval fails
- **Transparency**: Shows which path was taken and why
- **Adaptive Behavior**: Different handling for different query types

## Running the Example

```bash
cd examples/rag_conditional
go run main.go
```

## Key Components

- **Retriever**: Initial document retrieval
- **Reranker**: Scores documents for relevance
- **Conditional Edge**: Routes based on relevance threshold
- **Fallback Search**: Alternative search mechanism
- **LLM**: DeepSeek-v3 for answer generation
- **Pipeline**: Conditional RAG with branching logic

## Pipeline Flow

```
Query → Retrieve → Rerank → Check Relevance Score
                              ↓
                    ┌─────────┴─────────┐
                    ↓                   ↓
            Score >= Threshold    Score < Threshold
                    ↓                   ↓
              Generate           Fallback Search
                    ↓                   ↓
                    └─────────┬─────────┘
                              ↓
                      Format Citations → Result
```

## Routing Logic

The pipeline uses a relevance threshold (default: 0.5) to decide:
- **High Relevance** (≥ 0.5): Documents are relevant, proceed to generation
- **Low Relevance** (< 0.5): Documents may not be relevant, trigger fallback

## Example Queries

The example includes queries that demonstrate both paths:
1. **Relevant queries**: "How does checkpointing work?" → Direct generation
2. **Irrelevant queries**: "What is the weather?" → Fallback search

## Use Cases

1. **Hybrid Search**: Combine vector search with keyword or web search
2. **Quality Control**: Only use retrieved docs if they're truly relevant
3. **Graceful Degradation**: Provide alternative responses for out-of-domain queries
4. **Multi-Source RAG**: Route to different knowledge sources based on query type

## Customization

You can customize:
- Relevance threshold for routing
- Fallback search implementation (e.g., web search, keyword search)
- Routing logic and conditions
- Number of documents to retrieve
- Reranking algorithm

## Best Practices

1. **Tune Threshold**: Adjust based on your use case and quality requirements
2. **Implement Fallback**: Provide meaningful fallback (web search, default response)
3. **Monitor Routing**: Track which path queries take for optimization
4. **Test Edge Cases**: Ensure good behavior for both relevant and irrelevant queries
