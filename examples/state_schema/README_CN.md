# State Schema 示例

本示例演示如何在 LangGraphGo 中使用 **State Schema (状态模式)** 和 **Reducers (归约器)** 来管理复杂的状态更新。

## 1. 背景

在 LangGraph 中，状态不仅仅是一个在每一步都被覆盖的简单变量。它是一个结构化对象，其中不同的字段可以有不同的更新行为。这个概念对应于 Python 库中的 `TypedDict` 和 `Annotated`。

例如：
- 消息列表通常应该被 **追加 (Appended)**。
- 计数器应该被 **累加 (Incremented)**。
- 状态标志应该被 **覆盖 (Overwritten)**。

## 2. 核心概念

- **StateSchema**: 定义状态的结构以及更新如何合并。`graph.MapSchema` 是最常用的实现。
- **Reducer**: 一个函数，接受当前值和新值，并返回合并后的值。
  - `AppendReducer`: 将新项追加到列表中。
  - `OverwriteReducer`: 用新值替换旧值（默认行为）。
  - **Custom Reducer (自定义 Reducer)**: 您可以定义自己的逻辑（例如 `SumReducer`）。

## 3. 工作原理

1.  **定义 Schema**: 我们创建一个 `MapSchema` 并为特定键注册 Reducer。
    - `count`: 使用 `SumReducer`（自定义）来累加值。
    - `logs`: 使用 `AppendReducer` 来累积字符串。
    - `status`: 使用默认的覆盖行为。
2.  **节点**: 每个节点返回一个部分状态更新（包含某些键的 map）。
3.  **执行**: 当节点完成时，运行时使用 Schema 将返回的部分状态合并到全局状态中。

## 4. 代码亮点

### 定义自定义 Reducer
```go
func SumReducer(current, new any) (any, error) {
    // ... 累加整数的逻辑 ...
    return c + n, nil
}
```

### 配置 Schema
```go
schema := graph.NewMapSchema()
schema.RegisterReducer("count", SumReducer)
schema.RegisterReducer("logs", graph.AppendReducer)
g.SetSchema(schema)
```

## 5. 运行示例

```bash
go run main.go
```

**预期输出:**
```text
--- Final State ---
Count (Sum): 6
Logs (Append): [Start Processed by A Processed by B Processed by C]
Status (Overwrite): Completed
```
