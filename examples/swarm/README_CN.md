# Swarm 模式示例

本示例演示如何使用 LangGraphGo 实现多 Agent 协作的 **Swarm** 模式。

## 1. 背景

**Swarm** 模式（受 OpenAI Swarm 框架启发）是一种去中心化的多 Agent 编排方法。与使用中央 Agent 路由任务的 **Supervisor** 模式不同，Swarm 允许 Agent 直接将执行权移交（Handoff）给彼此。这模仿了专家团队的协作方式——根据各自的能力来回传递任务。

## 2. 核心概念

- **Handoff (移交)**: 一个 Agent 将控制权转移给另一个 Agent 的机制。在本例中，它被实现为一个 Agent 可以调用的工具 (`handoff`)。
- **Router (路由)**: 一个条件边函数，它检查状态（特别是 `next` 字段）以确定下一个应该运行哪个 Agent。
- **Shared State (共享状态)**: 所有 Agent 共享相同的对话历史 (`messages`)，使它们能够看到之前发生的事情。

## 3. 工作原理

1.  **定义 Agent**: 我们定义了两个 Agent：`Researcher` (研究员) 和 `Writer` (作家)。
2.  **Handoff 工具**: 我们为两个 Agent 都提供了 `handoff` 工具。该工具接受一个 `to` 参数（例如 "Writer"）。
3.  **执行循环**:
    - `Researcher` 开始运行。它接收用户的请求。
    - 如果它需要写报告，它会调用 `handoff` 工具，参数为 `to="Writer"`。
    - 节点逻辑捕获此工具调用，将状态中的 `next` 字段更新为 "Writer"，然后返回。
4.  **路由**: `router` 函数看到 `next` 是 "Writer"，于是将流程导向 `Writer` 节点。
5.  **完成**: 如果 Agent 决定任务已完成，它会返回正常的响应（无工具调用），路由将导向 `END`。

## 4. 代码亮点

### Handoff 工具定义
```go
var HandoffTool = llms.Tool{
    Name: "handoff",
    // ...
    Parameters: map[string]any{
        "properties": map[string]any{
            "to": map[string]any{
                "enum": []string{"Researcher", "Writer"},
            },
        },
    },
}
```

### 在节点中处理 Handoff
```go
if tc.FunctionCall.Name == "handoff" {
    // ... 解析参数 ...
    return map[string]any{
        // 将工具调用和响应添加到历史记录
        "messages": []llms.MessageContent{ ... },
        // 设置下一个 Agent
        "next": args.To,
    }, nil
}
```

### 条件路由
```go
router := func(ctx context.Context, state any) string {
    mState := state.(map[string]any)
    next := mState["next"].(string)
    if next == "" || next == "END" {
        return graph.END
    }
    return next // 返回 "Researcher" 或 "Writer"
}
workflow.AddConditionalEdge("Researcher", router)
workflow.AddConditionalEdge("Writer", router)
```

## 5. 运行示例

```bash
export OPENAI_API_KEY=your_key
go run main.go
```

**注意**: 本示例需要 OpenAI API Key，因为它使用了函数调用功能。
