# RAG 快速入门指南

本指南将帮助您快速开始使用 LangGraphGo 的 RAG 功能。

## 5 分钟快速开始

### 1. 准备文档

```go
documents := []prebuilt.Document{
    {
        PageContent: "LangGraph 是一个用于构建有状态、多角色应用的库。",
        Metadata: map[string]any{
            "source": "intro.txt",
        },
    },
    {
        PageContent: "RAG 结合了信息检索和文本生成。",
        Metadata: map[string]any{
            "source": "rag.txt",
        },
    },
}
```

### 2. 创建向量存储

```go
// 创建嵌入器和向量存储
embedder := prebuilt.NewMockEmbedder(128)
vectorStore := prebuilt.NewInMemoryVectorStore(embedder)

// 生成嵌入并添加文档
texts := []string{documents[0].PageContent, documents[1].PageContent}
embeddings, _ := embedder.EmbedDocuments(ctx, texts)
vectorStore.AddDocuments(ctx, documents, embeddings)
```

### 3. 创建检索器

```go
retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 2)
```

### 4. 配置并构建 RAG 流水线

```go
// 初始化 LLM
llm, _ := openai.New()

// 配置 RAG
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.LLM = llm

// 构建流水线
pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildBasicRAG()
runnable, _ := pipeline.Compile()
```

### 5. 执行查询

```go
result, _ := runnable.Invoke(ctx, prebuilt.RAGState{
    Query: "什么是 LangGraph？",
})

finalState := result.(prebuilt.RAGState)
fmt.Printf("答案: %s\n", finalState.Answer)
```

## 完整示例

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

    // 1. 准备文档
    documents := []prebuilt.Document{
        {
            PageContent: "LangGraph 是一个用于构建有状态、多角色应用的库。",
            Metadata:    map[string]any{"source": "intro.txt"},
        },
    }

    // 2. 创建向量存储
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

    // 3. 创建检索器
    retriever := prebuilt.NewVectorStoreRetriever(vectorStore, 2)

    // 4. 配置 RAG
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

    // 5. 构建流水线
    pipeline := prebuilt.NewRAGPipeline(config)
    err = pipeline.BuildBasicRAG()
    if err != nil {
        log.Fatal(err)
    }

    runnable, err := pipeline.Compile()
    if err != nil {
        log.Fatal(err)
    }

    // 6. 执行查询
    result, err := runnable.Invoke(ctx, prebuilt.RAGState{
        Query: "什么是 LangGraph？",
    })
    if err != nil {
        log.Fatal(err)
    }

    finalState := result.(prebuilt.RAGState)
    fmt.Printf("查询: %s\n", finalState.Query)
    fmt.Printf("答案: %s\n", finalState.Answer)
}
```

## 三种 RAG 模式选择

### 基础 RAG - 适合快速原型
```go
pipeline.BuildBasicRAG()
```
- 最简单
- 检索 → 生成
- 适合高质量文档集

### 高级 RAG - 适合生产环境
```go
config.UseReranking = true
config.IncludeCitations = true
pipeline.BuildAdvancedRAG()
```
- 包含重排序
- 自动引用
- 更高准确性

### 条件 RAG - 适合复杂场景
```go
config.UseReranking = true
config.UseFallback = true
config.ScoreThreshold = 0.7
pipeline.BuildConditionalRAG()
```
- 智能路由
- 后备搜索
- 自适应行为

## 常用配置

### 调整检索数量
```go
config.TopK = 5  // 检索前 5 个文档
```

### 设置相关性阈值
```go
config.ScoreThreshold = 0.7  // 最小相关性分数
```

### 自定义系统提示
```go
config.SystemPrompt = "你是一个专业的 AI 助手。基于提供的上下文回答问题。"
```

### 启用引用
```go
config.IncludeCitations = true
```

## 文档分块

对于大型文档，使用文本分割器：

```go
splitter := prebuilt.NewSimpleTextSplitter(500, 50)
chunks, _ := splitter.SplitDocuments(documents)
```

参数：
- `500`: 每块字符数
- `50`: 块之间的重叠

## 下一步

1. **查看完整示例**:
   - `examples/rag_basic/` - 基础示例
   - `examples/rag_advanced/` - 高级示例
   - `examples/rag_conditional/` - 条件示例

2. **阅读详细文档**:
   - `docs/RAG_CN.md` - 完整中文文档
   - `docs/RAG.md` - 完整英文文档

3. **自定义组件**:
   - 实现自己的 `Retriever`
   - 实现自己的 `Reranker`
   - 集成真实的向量数据库

## 常见问题

### Q: 如何使用真实的嵌入模型？

A: 集成 LangChain 的嵌入模型：
```go
import "github.com/tmc/langchaingo/embeddings"

embedder := embeddings.NewOpenAI()
```

### Q: 如何使用真实的向量数据库？

A: 实现 `VectorStore` 接口或使用 LangChain 的向量存储：
```go
import "github.com/tmc/langchaingo/vectorstores"

vectorStore := vectorstores.NewChroma(...)
```

### Q: 如何提高检索质量？

A: 
1. 使用文档分块
2. 启用重排序
3. 调整 TopK 和阈值
4. 使用更好的嵌入模型

### Q: 如何添加元数据过滤？

A: 在文档中添加元数据，然后在自定义检索器中过滤：
```go
doc.Metadata["category"] = "技术"
doc.Metadata["date"] = "2024-01-01"
```

## 性能优化建议

1. **批量处理**: 一次性生成所有嵌入
2. **缓存**: 缓存常见查询的结果
3. **异步**: 并行处理多个查询
4. **索引**: 使用专业的向量数据库
5. **限制**: 设置合理的 TopK 值

## 获取帮助

- 查看示例代码
- 阅读完整文档
- 检查测试文件了解更多用法
