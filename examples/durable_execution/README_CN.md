# 持久化执行 (Durable Execution) 示例

本示例演示 LangGraphGo 中的 **持久化执行 (Durable Execution)**。它展示了长时间运行的进程如何在崩溃后恢复，并从上次保存的检查点继续执行。

## 1. 背景

**持久化执行** 确保应用程序的状态在每一步都被持久化。如果进程崩溃（例如断电、内存溢出、部署重启），它可以重新启动并自动从中断的地方继续，避免数据丢失和重复工作。

## 2. 核心概念

- **CheckpointStore**: 一个持久化存储后端（例如文件、Postgres、Redis），用于保存图状态。
- **崩溃恢复 (Crash Recovery)**: 启动时检查现有检查点并恢复执行而不是重新开始的模式。

## 3. 工作原理

1.  **文件存储**: 我们实现了一个简单的基于 JSON 的文件存储 (`checkpoints.json`) 来持久化状态。
2.  **模拟**:
    - 图有 3 个步骤。
    - 如果设置了环境变量 `CRASH=true`，**步骤 2** 被编程为崩溃（退出）。
3.  **恢复**:
    - 启动时，程序检查 `checkpoints.json` 中是否存在给定的 `thread_id`。
    - 如果存在检查点（例如来自步骤 1），它加载状态并确定下一步（步骤 2）。
    - 然后它从该步骤恢复执行。

## 4. 运行示例

**步骤 1: 运行并崩溃**
```bash
export CRASH=true
go run main.go
```
*输出:*
```text
Starting new execution...
Executing Step 1...
Executing Step 2...
!!! CRASHING AT STEP 2 !!!
```

**步骤 2: 恢复并完成**
```bash
unset CRASH
go run main.go
```
*输出:*
```text
Found existing checkpoint: ... (Node: step_1)
Resuming execution...
Continuing from step_2...
Executing Step 2...
Executing Step 3...
Final Result: ...
```

**清理**
要重新开始，请删除 `checkpoints.json` 文件。
```bash
rm checkpoints.json
```
