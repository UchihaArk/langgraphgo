# Command API 示例

## 背景

在复杂的 Agent 工作流中，静态的图定义（编译时定义的边）有时是不够的。Agent 可能需要根据工具调用或 LLM 决策的结果，*动态地*决定下一步去哪里，这可能涉及跳过中间步骤或选择一条未显式连接的路径。**Command API** 提供了一种机制，允许节点直接返回控制流指令。

## 功能特性

*   **动态路由 (`Goto`)**: 允许节点指定下一个要执行的节点，覆盖图的静态边。
*   **状态更新 (`Update`)**: 允许节点在发出路由指令的同时更新图的状态。
*   **灵活性**: 支持“提前退出”、“跳过步骤”或“动态循环”等模式，无需编写复杂的条件边。

## 实现原理

`Command` 结构体定义如下：

```go
type Command struct {
    Update any // 要应用的状态更新
    Goto   any // 下一个要执行的节点 (string 或 []string)
}
```

当一个节点返回 `*Command` 对象时：
1.  `Update` 负载会使用定义的 Schema/Reducer 合并到图状态中。
2.  检查 `Goto` 字段。如果存在，图执行引擎将忽略当前节点的静态出边，转而调度 `Goto` 中指定的节点。

## 代码导读

在 `main.go` 中：

1.  **Router 节点**:
    ```go
    g.AddNode("router", func(ctx context.Context, state any) (any, error) {
        // ... 检查 count 的逻辑 ...
        if count > 5 {
            // 动态跳转: 跳过 "process" 直接去 "end_high"
            return &graph.Command{
                Update: map[string]any{"status": "high"},
                Goto:   "end_high",
            }, nil
        }
        // ...
    })
    ```
    该节点检查状态。如果 `count > 5`，它返回一个 `Command`，指示流程立即跳转到 `end_high`，从而绕过 `process` 节点。

2.  **执行**:
    示例运行了两种情况。在情况 2 (`count=10`) 中，你会看到 "process" 节点从未被执行，证明动态路由生效了。

## 如何运行

```bash
go run main.go
```
