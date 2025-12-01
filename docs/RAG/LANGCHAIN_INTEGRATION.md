# LangChain 集成总结

## 概述

我们为 LangGraphGo 的 RAG 系统创建了与 LangChain Go (`github.com/tmc/langchaingo`) 的无缝集成。

## 集成方式

### 方案选择：适配器模式

**问题**: 是否可以直接集成 `github.com/tmc/langchaingo/documentloaders`？

**答案**: 需要轻量级适配器，原因如下：

1. **类型差异**: 
   - LangChain 使用 `schema.Document` (带 `Score` 字段)
   - 我们使用 `prebuilt.Document` (无 `Score` 字段，分数存在 metadata 中)

2. **接口差异**:
   - LangChain 的 `TextSplitter` 只有 `SplitText(string) ([]string, error)`
   - 我们的 `TextSplitter` 需要 `SplitDocuments([]Document) ([]Document, error)`

3. **优势**:
   - ✅ 保持接口清晰分离
   - ✅ 类型安全的转换
   - ✅ 元数据和分数的正确处理
   - ✅ 易于维护和测试

## 实现的组件

### 1. 适配器文件 (`prebuilt/rag_langchain_adapter.go`)

#### LangChainDocumentLoader
```go
type LangChainDocumentLoader struct {
    loader documentloaders.Loader
}
```

**功能**:
- 包装任何 LangChain 文档加载器
- 实现我们的 `DocumentLoader` 接口
- 自动转换 `schema.Document` ↔ `prebuilt.Document`
- 保留所有元数据
- 将 Score 存储在 metadata 中

**方法**:
- `Load(ctx) ([]Document, error)` - 加载文档
- `LoadAndSplit(ctx, splitter) ([]Document, error)` - 加载并分割

#### LangChainTextSplitter
```go
type LangChainTextSplitter struct {
    splitter textsplitter.TextSplitter
}
```

**功能**:
- 包装任何 LangChain 文本分割器
- 实现我们的 `TextSplitter` 接口
- 调用底层的 `SplitText` 方法
- 为每个块保留原始文档的元数据
- 添加块索引和总数信息

**方法**:
- `SplitDocuments(documents) ([]Document, error)` - 分割文档

### 2. 示例应用 (`examples/rag_with_langchain/`)

完整的示例展示了 5 个用例：

1. **文本加载器**: 从字符串加载文本
2. **加载和分割**: 使用 LangChain 的 RecursiveCharacterTextSplitter
3. **完整 RAG 流水线**: 端到端的检索增强生成
4. **CSV 加载器**: 加载结构化数据
5. **文本分割器适配器**: 独立使用分割器

## 使用方法

### 基本用法

```go
// 1. 创建 LangChain 加载器
textReader := strings.NewReader(content)
lcLoader := documentloaders.NewText(textReader)

// 2. 包装为我们的接口
loader := prebuilt.NewLangChainDocumentLoader(lcLoader)

// 3. 使用
docs, err := loader.Load(ctx)
```

### 加载和分割

```go
// 创建 LangChain 分割器
splitter := textsplitter.NewRecursiveCharacter(
    textsplitter.WithChunkSize(200),
    textsplitter.WithChunkOverlap(50),
)

// 一步完成加载和分割
chunks, err := loader.LoadAndSplit(ctx, splitter)
```

### 在 RAG 流水线中使用

```go
// 加载和分割文档
chunks, _ := loader.LoadAndSplit(ctx, splitter)

// 创建向量存储
embedder := prebuilt.NewMockEmbedder(128)
vectorStore := prebuilt.NewInMemoryVectorStore(embedder)

// 添加文档
texts := extractTexts(chunks)
embeddings, _ := embedder.EmbedDocuments(ctx, texts)
vectorStore.AddDocuments(ctx, chunks, embeddings)

// 创建 RAG 流水线
retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 3)
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.LLM = llm

pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildBasicRAG()
runnable, _ := pipeline.Compile()

// 查询
result, _ := runnable.Invoke(ctx, prebuilt.RAGState{
    Query: "你的问题",
})
```

## 支持的 LangChain 组件

### 文档加载器

所有 LangChain 加载器都可以使用：

| 加载器     | 用途        | 示例                                                |
| ---------- | ----------- | --------------------------------------------------- |
| Text       | 纯文本      | `documentloaders.NewText(reader)`                   |
| CSV        | CSV 文件    | `documentloaders.NewCSV(reader)`                    |
| HTML       | HTML 文档   | `documentloaders.NewHTML(reader)`                   |
| PDF        | PDF 文件    | `documentloaders.NewPDF(reader, size)`              |
| Notion     | Notion 导出 | `documentloaders.NewNotionDirectory(path)`          |
| AssemblyAI | 音频转录    | `documentloaders.NewAssemblyAIAudioTranscript(key)` |

### 文本分割器

所有 LangChain 分割器都可以使用：

| 分割器               | 用途       | 示例                                            |
| -------------------- | ---------- | ----------------------------------------------- |
| RecursiveCharacter   | 通用文本   | `textsplitter.NewRecursiveCharacter(opts...)`   |
| TokenSplitter        | 基于 token | `textsplitter.NewTokenSplitter(opts...)`        |
| MarkdownTextSplitter | Markdown   | `textsplitter.NewMarkdownTextSplitter(opts...)` |

## 类型转换细节

### schema.Document → prebuilt.Document

```go
langchainDoc := schema.Document{
    PageContent: "内容",
    Metadata: map[string]any{"source": "test.txt"},
    Score: 0.95,
}

// 转换后
ourDoc := prebuilt.Document{
    PageContent: "内容",
    Metadata: map[string]interface{}{
        "source": "test.txt",
        "score": float32(0.95),  // Score 存储在 metadata 中
    },
}
```

### prebuilt.Document → schema.Document

```go
ourDoc := prebuilt.Document{
    PageContent: "内容",
    Metadata: map[string]interface{}{
        "source": "test.txt",
        "score": float32(0.95),
    },
}

// 转换后
langchainDoc := schema.Document{
    PageContent: "内容",
    Metadata: map[string]any{"source": "test.txt"},
    Score: 0.95,  // 从 metadata 提取
}
```

## 优势

1. **零学习成本**: 直接使用 LangChain 的文档和示例
2. **丰富的生态**: 访问 LangChain 的所有加载器
3. **类型安全**: 编译时类型检查
4. **元数据保留**: 完整保留所有元数据
5. **清晰分离**: 适配器提供清晰的边界
6. **易于测试**: 可以独立测试适配器

## 文件清单

```
prebuilt/
└── rag_langchain_adapter.go    # 适配器实现

examples/
└── rag_with_langchain/
    ├── main.go                  # 完整示例
    ├── README.md                # 英文文档
    └── README_CN.md             # 中文文档
```

## 编译状态

✅ 所有代码编译成功
✅ 适配器正确实现接口
✅ 示例可以运行

## 下一步建议

1. **添加更多示例**: PDF、HTML 加载器示例
2. **性能优化**: 大文件的流式处理
3. **错误处理**: 更详细的错误信息
4. **测试**: 为适配器添加单元测试
5. **文档**: 添加更多使用场景

## 总结

通过轻量级的适配器模式，我们实现了与 LangChain Go 的完美集成：

- ✅ **无需修改** LangChain 代码
- ✅ **类型安全** 的转换
- ✅ **完整保留** 元数据和分数
- ✅ **简单易用** 的 API
- ✅ **完整示例** 和文档

用户可以直接使用 LangChain 的所有文档加载器和文本分割器，同时享受我们 RAG 系统的所有功能！
