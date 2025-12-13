# 时间旅行 / 人机交互 (HITL) 示例

## 背景

**人机交互 (Human-in-the-loop, HITL)** 是构建可靠 Agent 系统的关键模式。它允许人类在 Agent 继续之前批准行动、纠正错误或提供指导。**时间旅行 (Time Travel)** 对此进行了扩展，允许你重访过去的状态，修改它们（“如果...会怎样？”），并从该点分叉执行。这对于调试和构建协作式 AI 应用至关重要。

## 功能特性

*   **中断**: 在定义的检查点暂停执行 (`InterruptBefore`)。
*   **状态持久化**: Checkpoint 存储完整的状态历史。
*   **状态编辑 (`UpdateState`)**: 允许修改暂停图的状态。这实际上创建了一个新的历史分支。
*   **恢复**: 从修改后的状态继续执行。

## 实现原理

1.  **Checkpointing**: `CheckpointableRunnable` 包装了图的执行。它使用 `CheckpointStore` 在每一步后（或按配置）保存状态。
2.  **中断**: 在执行节点之前，引擎检查该节点是否在 `InterruptBefore` 列表中。如果是，它保存状态并返回 `GraphInterrupt` 错误。
3.  **UpdateState**:
    *   加载最新的 Checkpoint。
    *   使用图的 Schema 合并用户提供的值。
    *   保存一个包含更新后状态和递增版本号的 **新** Checkpoint。
    *   返回指向该新 Checkpoint 的新 `Config`。
4.  **恢复**: 当使用新 Config 调用 `Invoke` 时，它从新 Checkpoint 加载状态并继续执行。

## 代码导读

在 `main.go` 中：

1.  **中断配置**:
    ```go
    config := &graph.Config{
        InterruptBefore: []string{"B"},
        // ...
    }
    ```
    告诉图在执行节点 B 之前停止。

2.  **运行 1**:
    执行节点 A (Count=1) 并停止。

3.  **人工干预**:
    ```go
    runnable.UpdateState(ctx, config, map[string]any{"count": 50}, "human")
    ```
    我们手动将计数设置为 50。系统会合并此值（取决于 Reducer，这里我们假设是覆盖或适合示例的加法逻辑）。

4.  **运行 2**:
    恢复。节点 B 使用 *新* 状态运行。在示例逻辑中，我们看到最终结果反映了手动更改。

## 如何运行

```bash
go run main.go
```
