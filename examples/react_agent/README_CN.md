# ReAct Agent 示例

本示例演示如何在 LangGraphGo 中使用预构建的 **ReAct Agent** 工厂。

## 1. 背景

**ReAct** (Reasoning and Acting，推理与行动) 是构建智能 Agent 的基础模式。模型不仅仅是生成文本，而是：
1.  **推理** 问题。
2.  **行动**，调用外部工具（如计算器、搜索引擎）。
3.  **观察** 这些工具的输出。
4.  重复此过程，直到能够回答用户的问题。

LangGraphGo 通过提供 `prebuilt.CreateReactAgent` 函数简化了这一过程，该函数会自动为该循环构建图。

## 2. 核心概念

- **Agent Node**: 调用 LLM 的节点。它接收对话历史，并决定是调用工具还是提供最终答案。
- **Tools Node**: 执行 Agent Node 请求的工具的节点。
- **Conditional Edge (条件边)**: 检查 LLM 输出的逻辑。如果包含工具调用，则路由到 Tools Node；否则，结束执行。
- **Tool Interface**: 工具必须实现 `langchaingo/tools.Tool` 接口 (`Name`, `Description`, `Call`)。

## 3. 工作原理

1.  **定义工具**: 我们定义一个简单的 `CalculatorTool`，它可以执行基本的算术运算。
2.  **创建 Agent**: 我们将 LLM 和工具列表传递给 `prebuilt.CreateReactAgent`。该函数在内部构建状态图。
3.  **调用**: 我们向 Agent 发送一个查询（"What is 25 * 4?"）。
4.  **执行流程**:
    - LLM 看到查询和可用的 `calculator` 工具。
    - 它决定调用 `calculator`，输入为 `25 * 4`。
    - 图路由到工具执行器。
    - 工具返回 `100.000000`。
    - LLM 接收到这个观察结果，并生成最终答案："The answer is 100."

## 4. 代码亮点

### 定义工具
```go
type CalculatorTool struct{}
func (t CalculatorTool) Name() string { return "calculator" }
func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
    // ... 实现 ...
}
```

### 创建 Agent
```go
inputTools := []tools.Tool{CalculatorTool{}}
// 自动构建包含 Agent 和 Tool 节点的图
agent, err := prebuilt.CreateReactAgent(model, inputTools)
```

### 运行 Agent
```go
initialState := map[string]any{
    "messages": []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeHuman, "What is 25 * 4?"),
    },
}
res, err := agent.Invoke(ctx, initialState)
```

## 5. 运行示例

```bash
export OPENAI_API_KEY=your_key
go run main.go
```

**预期输出:**
```text
User: What is 25 * 4?
Agent: The result of 25 * 4 is 100.
```
