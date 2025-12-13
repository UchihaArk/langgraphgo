# RAG (检索增强生成) 在 LangGraphGo 中的实现

本文档描述了 LangGraphGo 中的 RAG 接口和实现，参考了 LangChain 的 RAG 模式。

## 概述

RAG (Retrieval-Augmented Generation，检索增强生成) 是一种将信息检索与文本生成相结合的技术，用于生成更准确、更具上下文相关性和更有根据的响应。LangGraphGo 提供了一个灵活的、基于接口的 RAG 系统，支持多种实现模式。

## 核心接口

### Document (文档)

```go
type Document struct {
    PageContent string                 // 文档内容
    Metadata    map[string]any // 元数据
}
```

表示包含内容和元数据的文档。

### DocumentLoader (文档加载器)

```go
type DocumentLoader interface {
    Load(ctx context.Context) ([]Document, error)
}
```

从各种来源（文件、数据库、API 等）加载文档。

### TextSplitter (文本分割器)

```go
type TextSplitter interface {
    SplitDocuments(documents []Document) ([]Document, error)
}
```

将大型文档分割成更小的块，以便更好地检索和处理。

### Embedder (嵌入生成器)

```go
type Embedder interface {
    EmbedDocuments(ctx context.Context, texts []string) ([][]float64, error)
    EmbedQuery(ctx context.Context, text string) ([]float64, error)
}
```

为文本生成向量嵌入，实现语义搜索。

### VectorStore (向量存储)

```go
type VectorStore interface {
    AddDocuments(ctx context.Context, documents []Document, embeddings [][]float64) error
    SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error)
    SimilaritySearchWithScore(ctx context.Context, query string, k int) ([]DocumentWithScore, error)
}
```

使用相似度搜索存储和检索文档嵌入。

### Retriever (检索器)

```go
type Retriever interface {
    GetRelevantDocuments(ctx context.Context, query string) ([]Document, error)
}
```

为查询检索相关文档（抽象不同的检索方法）。

### Reranker (重排序器)

```go
type Reranker interface {
    Rerank(ctx context.Context, query string, documents []Document) ([]DocumentWithScore, error)
}
```

对检索到的文档重新评分，以提高相关性排名。

## RAG 流水线模式

### 1. 基础 RAG

**流程**: 检索 → 生成

最简单的 RAG 模式：
- 检索 top-k 相关文档
- 使用 LLM 根据检索到的上下文生成答案

**适用场景**:
- 快速原型开发
- 简单的问答系统
- 高质量文档集合

**示例**:
```go
config := prebuilt.DefaultRAGConfig()
config.Retriever = retriever
config.LLM = llm

pipeline := prebuilt.NewRAGPipeline(config)
pipeline.BuildBasicRAG()
```

### 2. 高级 RAG

**流程**: 检索 → 重排序 → 生成 → 格式化引用

增强的 RAG，具有质量改进：
- 文档分块以获得更好的粒度
- 重排序以提高相关性
- 生成引用以提高透明度

**适用场景**:
- 生产环境的 RAG 系统
- 需要高准确性的应用
- 需要来源归属的系统

**示例**:
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

### 3. 条件 RAG

**流程**: 检索 → 重排序 → 路由（基于相关性）→ 生成

基于相关性的智能路由：
- 基于相关性分数的条件边
- 低相关性查询的后备搜索
- 针对不同查询类型的自适应行为

**适用场景**:
- 混合搜索系统
- 可变查询类型
- 健壮的生产系统

**示例**:
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

## 提供的实现

### SimpleTextSplitter (简单文本分割器)

将文本分割成可配置大小和重叠的块：

```go
splitter := prebuilt.NewSimpleTextSplitter(
    chunkSize: 500,    // 每块字符数
    chunkOverlap: 50,  // 块之间的重叠
)
chunks, err := splitter.SplitDocuments(documents)
```

**最佳实践**:
- 块大小：大多数用例为 200-500 个 token
- 重叠：10-20% 以保持上下文

### InMemoryVectorStore (内存向量存储)

用于开发和测试的简单内存向量存储：

```go
embedder := prebuilt.NewMockEmbedder(128)
vectorStore := prebuilt.NewInMemoryVectorStore(embedder)
vectorStore.AddDocuments(ctx, documents, embeddings)
results, err := vectorStore.SimilaritySearch(ctx, query, k)
```

**注意**: 对于生产环境，请集成真实的向量数据库（Pinecone、Weaviate、Chroma 等）

### VectorStoreRetriever (向量存储检索器)

使用向量存储的检索器实现：

```go
retriever := prebuilt.NewVectorStoreRetriever(vectorStore, topK)
docs, err := retriever.GetRelevantDocuments(ctx, query)
```

### SimpleReranker (简单重排序器)

基于关键词的重排序器，用于提高检索质量：

```go
reranker := prebuilt.NewSimpleReranker()
rankedDocs, err := reranker.Rerank(ctx, query, documents)
```

**注意**: 对于生产环境，考虑使用交叉编码器模型以获得更好的重排序效果。

### MockEmbedder (模拟嵌入器)

用于测试的确定性嵌入器：

```go
embedder := prebuilt.NewMockEmbedder(dimension)
embeddings, err := embedder.EmbedDocuments(ctx, texts)
```

**注意**: 对于生产环境，使用真实的嵌入模型（OpenAI、Cohere、sentence-transformers 等）

## RAG 状态

RAG 流水线使用流经图的类型化状态：

```go
type RAGState struct {
    Query              string                // 用户查询
    Documents          []Document            // 当前文档
    RetrievedDocuments []Document            // 最初检索的文档
    RankedDocuments    []DocumentWithScore   // 带分数的重排序文档
    Context            string                // 为 LLM 格式化的上下文
    Answer             string                // 生成的答案
    Citations          []string              // 来源引用
    Metadata           map[string]any // 附加元数据
}
```

## 配置

RAG 流水线使用 `RAGConfig` 进行配置：

```go
type RAGConfig struct {
    // 检索配置
    TopK            int     // 要检索的文档数量
    ScoreThreshold  float64 // 最小相关性分数
    UseReranking    bool    // 是否使用重排序
    UseFallback     bool    // 是否使用后备搜索
    
    // 生成配置
    SystemPrompt    string  // LLM 的系统提示
    IncludeCitations bool   // 是否包含引用
    MaxTokens       int     // 生成的最大 token 数
    Temperature     float64 // LLM 温度
    
    // 组件
    Loader      DocumentLoader
    Splitter    TextSplitter
    Embedder    Embedder
    VectorStore VectorStore
    Retriever   Retriever
    Reranker    Reranker
    LLM         llms.Model
}
```

## 高级模式

### 多查询 RAG

生成多个查询变体以提高检索效果：

```go
// 实现生成查询变体的自定义检索器
type MultiQueryRetriever struct {
    baseRetriever Retriever
    llm          llms.Model
}
```

### 混合搜索

结合向量搜索和关键词搜索：

```go
// 实现合并结果的自定义检索器
type HybridRetriever struct {
    vectorRetriever  Retriever
    keywordRetriever Retriever
}
```

### 父文档检索

检索小块但提供更大的上下文：

```go
// 在元数据中存储块到父文档的映射
chunk.Metadata["parent_id"] = parentDoc.ID
```

### 上下文压缩

压缩检索到的文档以仅提取相关部分：

```go
// 在流水线中实现自定义节点
func compressContext(ctx context.Context, state any) (any, error) {
    // 使用 LLM 提取相关部分
}
```

## 最佳实践

### 1. 文档准备

- **清理文本**: 删除噪音、格式化伪影
- **适当分块**: 平衡上下文和精度
- **添加元数据**: 包括来源、日期、类别等
- **去重**: 删除重复或近似重复的内容

### 2. 检索

- **调整 top-k**: 从 3-5 开始，根据结果调整
- **使用元数据过滤**: 按日期、类别等过滤
- **监控相关性**: 跟踪检索质量指标
- **考虑混合搜索**: 结合语义和关键词搜索

### 3. 重排序

- **始终重排序**: 显著提高精度
- **使用交叉编码器**: 比双编码器更适合重排序
- **限制重排序**: 仅对前 N 个候选项重排序（例如 20 个）

### 4. 生成

- **清晰的指令**: 指定如何使用上下文
- **引用来源**: 始终包含引用
- **处理不确定性**: 指示 LLM 承认局限性
- **控制长度**: 设置适当的最大 token 数

### 5. 评估

- **测试检索**: 测量 precision@k、recall@k
- **测试生成**: 评估答案质量、事实性
- **监控延迟**: 跟踪端到端响应时间
- **收集反馈**: 使用人工反馈进行改进

## 示例

查看示例目录以获取完整实现：

- `examples/rag_basic/` - 基础 RAG 流水线
- `examples/rag_advanced/` - 带重排序和引用的高级 RAG
- `examples/rag_conditional/` - 带路由的条件 RAG
- `examples/rag_pipeline/` - 原始 RAG 流水线示例

## 与 LangChain 的集成

LangGraphGo RAG 接口与 LangChain Go 组件兼容：

```go
import (
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/vectorstores"
)

// 使用 LangChain 嵌入
embedder := embeddings.NewOpenAI()

// 使用 LangChain 向量存储
vectorStore := vectorstores.NewChroma(...)
```

## 未来增强

计划的改进：

1. **更多检索器**: BM25、TF-IDF、混合搜索
2. **更好的重排序器**: 交叉编码器集成
3. **查询转换**: 多查询、HyDE、step-back
4. **上下文压缩**: 基于 LLM 的上下文提取
5. **评估工具**: 内置指标和测试
6. **流式传输**: 流式传输检索到的文档和生成

## 参考资料

- [LangChain RAG 教程](https://python.langchain.com/docs/tutorials/rag/)
- [RAG 最佳实践](https://www.anthropic.com/index/contextual-retrieval)
- [高级 RAG 技术](https://arxiv.org/abs/2312.10997)
