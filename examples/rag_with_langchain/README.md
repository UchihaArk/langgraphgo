# RAG with LangChain Integration Example

This example demonstrates how to integrate LangChain Go's document loaders and text splitters with LangGraphGo's RAG system.

## Overview

LangChain Go (`github.com/tmc/langchaingo`) provides excellent document loaders for various formats (Text, CSV, PDF, HTML, etc.) and text splitters. This example shows how to use them seamlessly with our RAG pipeline through adapter classes.

## Key Features

- **Direct Integration**: Use LangChain's document loaders without modification
- **Adapter Pattern**: Clean adapters that bridge LangChain and our RAG interfaces
- **Multiple Loaders**: Examples with Text, CSV, and other loaders
- **Text Splitting**: Integration with LangChain's RecursiveCharacterTextSplitter
- **Complete RAG Pipeline**: End-to-end example with retrieval and generation

## Architecture

### Adapter Classes

We provide two adapter classes in `prebuilt/rag_langchain_adapter.go`:

1. **LangChainDocumentLoader**: Adapts `documentloaders.Loader` to our `DocumentLoader` interface
2. **LangChainTextSplitter**: Adapts `textsplitter.TextSplitter` to our `TextSplitter` interface

These adapters handle conversion between:
- `schema.Document` (LangChain) ↔ `prebuilt.Document` (our type)

## Usage

### Basic Document Loading

```go
import (
    "github.com/tmc/langchaingo/documentloaders"
    "github.com/smallnest/langgraphgo/prebuilt"
)

// Create LangChain loader
textReader := strings.NewReader(content)
lcLoader := documentloaders.NewText(textReader)

// Wrap with adapter
loader := prebuilt.NewLangChainDocumentLoader(lcLoader)

// Use with our interface
docs, err := loader.Load(ctx)
```

### Load and Split

```go
import "github.com/tmc/langchaingo/textsplitter"

// Create LangChain text splitter
splitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(200),
    textsplitter.WithChunkOverlap(50),
)

// Load and split in one step
chunks, err := loader.LoadAndSplit(ctx, splitter)
```

### Using Text Splitter Adapter

```go
// Create splitter adapter
lcSplitter := textsplitter.NewRecursiveCharacter(...)
splitterAdapter := prebuilt.NewLangChainTextSplitter(lcSplitter)

// Use with our Document type
chunks, err := splitterAdapter.SplitDocuments(documents)
```

## Running the Example

```bash
cd examples/rag_with_langchain
go run main.go
```

## Examples Included

### 1. Text Loader
Load plain text documents:
```go
textLoader := documentloaders.NewText(reader)
loader := prebuilt.NewLangChainDocumentLoader(textLoader)
docs, _ := loader.Load(ctx)
```

### 2. CSV Loader
Load structured data from CSV:
```go
csvLoader := documentloaders.NewCSV(reader)
loader := prebuilt.NewLangChainDocumentLoader(csvLoader)
docs, _ := loader.Load(ctx)
```

### 3. Text Splitting
Split documents into chunks:
```go
splitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(200),
    textsplitter.WithChunkOverlap(50),
)
chunks, _ := loader.LoadAndSplit(ctx, splitter)
```

### 4. Complete RAG Pipeline
Build a full RAG system with LangChain components:
```go
// Load and split with LangChain
chunks, _ := loader.LoadAndSplit(ctx, splitter)

// Create RAG pipeline
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.LLM = llm

pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildBasicRAG()
runnable, _ := pipeline.Compile()

// Query
result, _ := runnable.Invoke(ctx, prebuilt.RAGState{
    Query: "What is LangGraph?",
})
```

## Supported LangChain Loaders

The adapter works with all LangChain document loaders:

- **Text**: `documentloaders.NewText(reader)`
- **CSV**: `documentloaders.NewCSV(reader, columns...)`
- **HTML**: `documentloaders.NewHTML(reader)`
- **PDF**: `documentloaders.NewPDF(reader, size)`
- **Notion**: `documentloaders.NewNotionDirectory(path)`
- **AssemblyAI**: `documentloaders.NewAssemblyAIAudioTranscript(apiKey)`

## Supported Text Splitters

The adapter works with all LangChain text splitters:

- **RecursiveCharacter**: `textsplitter.NewRecursiveCharacter(opts...)`
- **TokenSplitter**: `textsplitter.NewTokenSplitter(opts...)`
- **MarkdownTextSplitter**: `textsplitter.NewMarkdownTextSplitter(opts...)`

## Benefits of Integration

1. **Rich Ecosystem**: Access to LangChain's extensive loader library
2. **No Duplication**: Reuse well-tested LangChain components
3. **Clean Interface**: Adapters provide clean separation
4. **Type Safety**: Proper type conversion between systems
5. **Flexibility**: Easy to switch between implementations

## Advanced Usage

### Custom Metadata

LangChain documents include metadata that's preserved:

```go
docs, _ := loader.Load(ctx)
for _, doc := range docs {
    fmt.Printf("Source: %v\n", doc.Metadata["source"])
    fmt.Printf("Page: %v\n", doc.Metadata["page"])
}
```

### Score Preservation

Document scores from LangChain are stored in metadata:

```go
// LangChain document with score
schemaDoc := schema.Document{
    PageContent: "content",
    Score: 0.95,
}

// After conversion, score is in metadata
doc := convertSchemaDocuments([]schema.Document{schemaDoc})[0]
score := doc.Metadata["score"].(float32) // 0.95
```

### Combining Loaders

Load from multiple sources:

```go
// Load from text
textDocs, _ := textLoader.Load(ctx)

// Load from CSV
csvDocs, _ := csvLoader.Load(ctx)

// Combine
allDocs := append(textDocs, csvDocs...)
```

## Best Practices

1. **Use Appropriate Loaders**: Choose the right loader for your data format
2. **Configure Splitting**: Adjust chunk size based on your use case
3. **Preserve Metadata**: Ensure important metadata is maintained
4. **Error Handling**: Always check errors from Load operations
5. **Resource Management**: Close readers when done

## Comparison: Direct vs Adapter

### Without Adapter (Manual Conversion)
```go
lcDocs, _ := lcLoader.Load(ctx)
docs := make([]prebuilt.Document, len(lcDocs))
for i, d := range lcDocs {
    docs[i] = prebuilt.Document{
        PageContent: d.PageContent,
        Metadata: d.Metadata,
    }
}
```

### With Adapter (Clean)
```go
loader := prebuilt.NewLangChainDocumentLoader(lcLoader)
docs, _ := loader.Load(ctx)
```

## Troubleshooting

### Import Errors
Ensure you have the required dependencies:
```bash
go get github.com/tmc/langchaingo
```

### Type Conversion Issues
The adapter handles type conversion automatically. If you encounter issues, check that metadata values are compatible types.

### Memory Usage
For large documents, use streaming or chunking:
```go
splitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(500), // Smaller chunks
)
```

## Next Steps

1. Try different LangChain loaders (PDF, HTML, etc.)
2. Experiment with text splitter configurations
3. Build a RAG system with your own documents
4. Integrate with production vector databases
5. Add custom metadata processing

## See Also

- [LangChain Go Documentation](https://github.com/tmc/langchaingo)
- [RAG Documentation](../../docs/RAG/RAG.md)
- [Basic RAG Example](../rag_basic/)
- [Advanced RAG Example](../rag_advanced/)
