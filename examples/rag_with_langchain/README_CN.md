# RAG 与 LangChain 集成示例

本示例演示如何将 LangChain Go 的文档加载器和文本分割器与 LangGraphGo 的 RAG 系统集成。

## 概述

LangChain Go (`github.com/tmc/langchaingo`) 提供了优秀的文档加载器，支持多种格式（Text、CSV、PDF、HTML 等）和文本分割器。本示例展示如何通过适配器类无缝地将它们与我们的 RAG 流水线一起使用。

## 主要特性

- **直接集成**: 无需修改即可使用 LangChain 的文档加载器
- **适配器模式**: 清晰的适配器桥接 LangChain 和我们的 RAG 接口
- **多种加载器**: Text、CSV 和其他加载器的示例
- **文本分割**: 与 LangChain 的 RecursiveCharacterTextSplitter 集成
- **完整的 RAG 流水线**: 包含检索和生成的端到端示例

## 架构

### 适配器类

我们在 `prebuilt/rag_langchain_adapter.go` 中提供了两个适配器类：

1. **LangChainDocumentLoader**: 将 `documentloaders.Loader` 适配到我们的 `DocumentLoader` 接口
2. **LangChainTextSplitter**: 将 `textsplitter.TextSplitter` 适配到我们的 `TextSplitter` 接口

这些适配器处理以下类型之间的转换：
- `schema.Document` (LangChain) ↔ `prebuilt.Document` (我们的类型)

## 使用方法

### 基本文档加载

```go
import (
    "github.com/tmc/langchaingo/documentloaders"
    "github.com/smallnest/langgraphgo/prebuilt"
)

// 创建 LangChain 加载器
textReader := strings.NewReader(content)
lcLoader := documentloaders.NewText(textReader)

// 使用适配器包装
loader := prebuilt.NewLangChainDocumentLoader(lcLoader)

// 使用我们的接口
docs, err := loader.Load(ctx)
```

### 加载和分割

```go
import "github.com/tmc/langchaingo/textsplitter"

// 创建 LangChain 文本分割器
splitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(200),
    textsplitter.WithChunkOverlap(50),
)

// 一步完成加载和分割
chunks, err := loader.LoadAndSplit(ctx, splitter)
```

### 使用文本分割器适配器

```go
// 创建分割器适配器
lcSplitter := textsplitter.NewRecursiveCharacter(...)
splitterAdapter := prebuilt.NewLangChainTextSplitter(lcSplitter)

// 使用我们的 Document 类型
chunks, err := splitterAdapter.SplitDocuments(documents)
```

## 运行示例

```bash
cd examples/rag_with_langchain
go run main.go
```

## 包含的示例

### 1. 文本加载器
加载纯文本文档：
```go
textLoader := documentloaders.NewText(reader)
loader := prebuilt.NewLangChainDocumentLoader(textLoader)
docs, _ := loader.Load(ctx)
```

### 2. CSV 加载器
从 CSV 加载结构化数据：
```go
csvLoader := documentloaders.NewCSV(reader)
loader := prebuilt.NewLangChainDocumentLoader(csvLoader)
docs, _ := loader.Load(ctx)
```

### 3. 文本分割
将文档分割成块：
```go
splitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(200),
    textsplitter.WithChunkOverlap(50),
)
chunks, _ := loader.LoadAndSplit(ctx, splitter)
```

### 4. 完整的 RAG 流水线
使用 LangChain 组件构建完整的 RAG 系统：
```go
// 使用 LangChain 加载和分割
chunks, _ := loader.LoadAndSplit(ctx, splitter)

// 创建 RAG 流水线
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.LLM = llm

pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildBasicRAG()
runnable, _ := pipeline.Compile()

// 查询
result, _ := runnable.Invoke(ctx, prebuilt.RAGState{
    Query: "什么是 LangGraph？",
})
```

## 支持的 LangChain 加载器

适配器适用于所有 LangChain 文档加载器：

- **Text**: `documentloaders.NewText(reader)`
- **CSV**: `documentloaders.NewCSV(reader, columns...)`
- **HTML**: `documentloaders.NewHTML(reader)`
- **PDF**: `documentloaders.NewPDF(reader, size)`
- **Notion**: `documentloaders.NewNotionDirectory(path)`
- **AssemblyAI**: `documentloaders.NewAssemblyAIAudioTranscript(apiKey)`

## 支持的文本分割器

适配器适用于所有 LangChain 文本分割器：

- **RecursiveCharacter**: `textsplitter.NewRecursiveCharacter(opts...)`
- **TokenSplitter**: `textsplitter.NewTokenSplitter(opts...)`
- **MarkdownTextSplitter**: `textsplitter.NewMarkdownTextSplitter(opts...)`

## 集成的优势

1. **丰富的生态系统**: 访问 LangChain 广泛的加载器库
2. **无重复**: 重用经过良好测试的 LangChain 组件
3. **清晰的接口**: 适配器提供清晰的分离
4. **类型安全**: 系统之间的正确类型转换
5. **灵活性**: 易于在实现之间切换

## 高级用法

### 自定义元数据

LangChain 文档包含的元数据会被保留：

```go
docs, _ := loader.Load(ctx)
for _, doc := range docs {
    fmt.Printf("来源: %v\n", doc.Metadata["source"])
    fmt.Printf("页码: %v\n", doc.Metadata["page"])
}
```

### 分数保留

LangChain 的文档分数存储在元数据中：

```go
// 带分数的 LangChain 文档
schemaDoc := schema.Document{
    PageContent: "内容",
    Score: 0.95,
}

// 转换后，分数在元数据中
doc := convertSchemaDocuments([]schema.Document{schemaDoc})[0]
score := doc.Metadata["score"].(float32) // 0.95
```

### 组合加载器

从多个来源加载：

```go
// 从文本加载
textDocs, _ := textLoader.Load(ctx)

// 从 CSV 加载
csvDocs, _ := csvLoader.Load(ctx)

// 组合
allDocs := append(textDocs, csvDocs...)
```

## 最佳实践

1. **使用适当的加载器**: 为您的数据格式选择正确的加载器
2. **配置分割**: 根据您的用例调整块大小
3. **保留元数据**: 确保重要的元数据得到维护
4. **错误处理**: 始终检查 Load 操作的错误
5. **资源管理**: 完成后关闭 reader

## 对比：直接使用 vs 适配器

### 不使用适配器（手动转换）
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

### 使用适配器（简洁）
```go
loader := prebuilt.NewLangChainDocumentLoader(lcLoader)
docs, _ := loader.Load(ctx)
```

## 故障排除

### 导入错误
确保您有所需的依赖项：
```bash
go get github.com/tmc/langchaingo
```

### 类型转换问题
适配器自动处理类型转换。如果遇到问题，请检查元数据值是否为兼容类型。

### 内存使用
对于大型文档，使用流式传输或分块：
```go
splitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(500), // 更小的块
)
```

## 下一步

1. 尝试不同的 LangChain 加载器（PDF、HTML 等）
2. 实验文本分割器配置
3. 使用您自己的文档构建 RAG 系统
4. 集成生产向量数据库
5. 添加自定义元数据处理

## 另请参阅

- [LangChain Go 文档](https://github.com/tmc/langchaingo)
- [RAG 文档](../../docs/RAG/RAG_CN.md)
- [基础 RAG 示例](../rag_basic/)
- [高级 RAG 示例](../rag_advanced/)
