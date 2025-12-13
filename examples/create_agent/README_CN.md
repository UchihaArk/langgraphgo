# Create Agent 示例

本示例演示如何在 LangGraphGo 中使用 `prebuilt.CreateAgent` 函数，通过函数式选项轻松创建 ReAct 代理。

## 1. 背景

创建代理通常涉及设置包含代理节点、工具执行节点和条件路由逻辑的图。虽然 `CreateReactAgent` 提供了基本实现，但 `CreateAgent` 提供了一种更灵活、更可扩展的方式来构建代理，其设计灵感来自 LangChain 1.0。它支持函数式选项，可以更轻松地配置系统消息、状态修改器以及未来的扩展（如检查点）。

## 2. 核心概念

- **CreateAgent**: 一个工厂函数，用于为代理构建 `StateGraph`。
- **函数式选项 (Functional Options)**: 一种将可选参数（例如 `WithSystemMessage`）传递给工厂函数的模式，使 API 更加整洁且易于扩展。
- **系统消息 (System Message)**: 设定代理行为或角色的预定义指令。
- **状态修改器 (State Modifier)**: 一个函数，用于在消息历史发送给 LLM 之前对其进行拦截和修改。这对于动态提示工程或过滤非常有用。

## 3. 工作原理

1.  **初始化 LLM**: 创建语言模型实例（例如 OpenAI）。
2.  **定义工具**: 创建代理可以使用的工具列表。
3.  **创建 Agent**: 使用模型、工具和选项调用 `prebuilt.CreateAgent`。
    - `WithSystemMessage`: 设置系统提示词。
    - `WithStateModifier`: 允许自定义逻辑来修改消息。
4.  **调用**: 使用初始用户消息运行代理。代理将循环进行推理和工具执行，直到得出最终答案。

## 4. 代码亮点

### 使用选项创建 Agent

```go
agent, err := prebuilt.CreateAgent(model, inputTools,
    // 设置系统消息
    prebuilt.WithSystemMessage("You are a helpful weather assistant. Always be polite."),
    
    // 添加状态修改器以记录或更改消息
    prebuilt.WithStateModifier(func(msgs []llms.MessageContent) []llms.MessageContent {
        log.Printf("Current message count: %d", len(msgs))
        return msgs
    }),
)
```

### 运行 Agent

```go
inputs := map[string]any{
    "messages": []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeHuman, "What is the weather in San Francisco?"),
    },
}

result, err := agent.Invoke(ctx, inputs)
```

## 5. 运行示例

```bash
export OPENAI_API_KEY=your_api_key
go run main.go
```

**预期输出:**
```text
2024/12/01 22:30:00 Starting agent...
2024/12/01 22:30:00 Current message count: 2
Agent Response: The weather in San Francisco is sunny and 25°C.
```
