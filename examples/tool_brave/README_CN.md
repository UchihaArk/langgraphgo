# Brave 搜索工具示例

本示例展示如何使用 LangGraphGo 中的 Brave 搜索工具创建一个能够使用 Brave 隐私搜索 API 进行网络搜索的 AI 代理。

## 前置条件

1. 从 [Brave Search API](https://api.search.brave.com/) 获取 API 密钥
2. 获取 LLM API 密钥（OpenAI 或 DeepSeek）

## 设置

配置环境变量：

```bash
export BRAVE_API_KEY="your-brave-api-key"
export OPENAI_API_KEY="your-openai-api-key"
# 或者
export DEEPSEEK_API_KEY="your-deepseek-api-key"
```

## 运行

```bash
go run main.go
```

## 工作原理

1. **初始化 LLM**：创建 OpenAI 兼容的 LLM 客户端
2. **初始化工具**：创建 Brave 搜索工具，可配置以下选项：
   - Count: 返回结果数量（1-20）
   - Country: 本地化结果的国家代码
   - Lang: 搜索结果的语言代码
3. **创建 ReAct Agent**：将 LLM 和工具组合成 ReAct 代理
4. **运行 Agent**：发送查询并获取响应

## 自定义配置

你可以使用多种选项自定义 Brave 搜索工具：

```go
braveTool, err := tool.NewBraveSearch("",
    tool.WithBraveCount(10),           // 结果数量（1-20）
    tool.WithBraveCountry("CN"),       // 国家代码
    tool.WithBraveLang("zh"),          // 语言代码
)
```

## Brave 搜索的优势

- 注重隐私：不追踪或分析用户
- 独立性：不依赖 Google 或 Bing
- 新鲜结果：实时网络搜索
- 全球覆盖：支持多个国家和语言
