# 从检查点恢复示例

本示例演示如何使用 `FileCheckpointStore` 从特定点恢复图的执行。

## 概述

复杂的工作流可能需要中断（例如，等待人工批准）或可能因外部因素而失败。检查点机制允许您保存图的状态，并从最后保存的状态恢复执行，从而有效地跳过已完成的步骤。

## 演示的功能

1.  **第一阶段：中断/部分执行**
    - 运行一个多步骤的图 (`step1` -> `step2` -> `step3`)。
    - 演示配置中断（或模拟中断）。
    - 在每一步自动保存检查点。

2.  **第二阶段：恢复执行**
    - 识别特定线程的最新检查点。
    - 加载保留的状态。
    - 使用 `ResumeFrom` 配置重新初始化图。
    - 从下一个逻辑步骤 (`step3`) 继续执行，跳过 `step1` 和 `step2` 的重新执行。

## 运行示例

```bash
cd examples/file_checkpointing_resume
go run main.go
```

## 预期输出

您将看到两个执行阶段：
1.  **第一阶段**：执行步骤 1 和步骤 2。
2.  **第二阶段**：检测已保存的状态并立即执行步骤 3，完成工作流。

## 关键逻辑

```go
// 1. 加载最新的检查点
checkpoints, _ := store.List(ctx, threadID)
latestCP := checkpoints[len(checkpoints)-1]

// 2. 配置恢复
config := &graph.Config{
    Configurable: map[string]any{"thread_id": threadID},
    ResumeFrom:   []string{"step3"}, // 从 step3 开始恢复
}

// 3. 使用保留的状态调用
result, err := runnable.InvokeWithConfig(ctx, latestCP.State, config)
```
