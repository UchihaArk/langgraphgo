# 重构执行逻辑与恢复功能

我已成功重构了图执行逻辑，消除了代码重复，并确保了标准图和可监听图之间的一致行为。这解决了 `InterruptAfter` 逻辑不一致或缺失的问题。

## 变更内容

### 1. 统一执行逻辑
- 修改了核心执行引擎 `StateRunnable`，使其接受 `nodeRunner` 钩子。
- 更新了用于检查点的 `ListenableRunnable`，使其包装 `StateRunnable` 并通过 `nodeRunner` 注入监听逻辑。
- **结果：** `ListenableRunnable` 不再重新实现复杂的执行循环。它依赖于 `StateRunnable` 中唯一的、健壮的实现。

### 2. 修正中断处理
- 优化了 `StateRunnable.InvokeWithConfig`，改为在通知回调*之后*检查 `InterruptAfter`。
- **结果：** 当图在某一步骤*之后*中断时，该已完成步骤的状态现在会被正确保存到检查点存储中。这允许从*下一步*恢复，而不是重新运行被中断的步骤。

## 验证结果

### 自动化测试
当前的单元测试已通过：
```
ok      github.com/smallnest/langgraphgo/graph  1.270s
```

### 恢复示例
`examples/file_checkpointing_resume` 示例现在运行完美：

```
--- PHASE 1: Running until interruption after Step 2 ---
  [EXEC] Running Step 1
  [EXEC] Running Step 2
  [INFO] Graph interrupted as expected: graph interrupted at node step2

--- PHASE 2: Resuming from checkpoint ---
  [INFO] Resuming from checkpoint: ID=checkpoint_..., Node=step2, Version=2
  [INFO] State at checkpoint: map[input:start step1:done step2:done]
  [EXEC] Running Step 3
  [INFO] Final Result: map[input:start step1:done step2:done step3:done]
  [SUCCESS] Graph successfully resumed and completed all steps.
```
