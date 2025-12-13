# 文件检查点示例

本示例演示如何在 LangGraphGo 中使用 `FileCheckpointStore` 进行持久化状态管理。

## 概述

与应用程序退出时丢失数据的内存检查点存储不同，`FileCheckpointStore` 将执行状态保存到本地文件系统。这允许：
1. **持久化** - 状态在应用程序重启后仍然存在
2. **检查** - 检查点保存为可读的 JSON 文件
3. **恢复** - 工作流可以从特定点恢复（未来功能）

## 演示的功能

- **初始化**：创建临时目录并初始化 `FileCheckpointStore`。
- **配置**：设置带有文件存储的 `CheckpointableStateGraph`。
- **执行**：运行一个在每一步都保存状态的图。
- **验证**：从文件系统中列出已保存的检查点。

## 运行示例

```bash
cd examples/file_checkpointing
go run main.go
```

## 预期输出

该示例将：
1. 创建一个 `./checkpoints` 目录。
2. 执行一个简单的 2 步工作流。
3. 在每一步之后将状态保存为 `./checkpoints` 中的 JSON 文件。
4. 验证这些文件的存在。
5. 使用存储 API 列出检查点。
6. 退出时清理目录（实际应用程序可能会保留它们）。
