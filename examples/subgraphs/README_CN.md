# 子图 (Subgraphs) 示例

## 背景

随着 Agent 系统变得越来越复杂，管理单个单体图变得难以驾驭。**子图 (Subgraphs)** 允许开发者将逻辑封装到更小的、可重用的图中，并将它们组合成更大的系统。这类似于将大型程序分解为函数或微服务。子图在父图看来只是另一个节点，但在内部它管理着自己的状态和执行流。

## 功能特性

*   **封装**: 将子任务（例如“研究主题”）的复杂性隐藏在单个节点后面。
*   **可重用性**: 定义一次图，在多个地方或项目中使用。
*   **状态映射**: 自动将父图的状态传递给子图，并将子图的结果合并回来（假设 Schema 兼容）。

## 实现原理

`graph/subgraph.go` 中的 `AddSubgraph` 方法将 `StateGraph`（或 `MessageGraph`，用于向后兼容）包装到 `Subgraph` 结构中。
1.  子图被编译为 `Runnable`。
2.  创建一个匹配 `Node` 签名 (`func(ctx, state) (any, error)`) 的包装函数。
3.  执行时，该包装函数调用子图的 `Runnable.Invoke`。
4.  结果作为节点的输出返回给父图。

## 代码导读

在 `main.go` 中：

1.  **Child Graph**:
    定义为一个简单的图，向状态添加 "child_trace"。

2.  **Parent Graph**:
    定义为一个向状态添加 "parent_trace" 的图。

3.  **组合**:
    ```go
    parent.AddSubgraph("nested_graph", child)
    ```
    子图被注册为名为 "nested_graph" 的节点。

4.  **连线**:
    `start -> nested_graph -> end`。
    父图将 "nested_graph" 视为与其他节点一样。

## 如何运行

```bash
go run main.go
```
