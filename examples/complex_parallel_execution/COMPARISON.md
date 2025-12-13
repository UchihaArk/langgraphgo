# 版本对比：main_single_aggregation.go vs main_optimized.go

## 核心区别

这两个版本都试图解决同一个问题：**如何让聚合器只在所有分支完成后才执行一次**，但使用了不同的方法。

## main_single_aggregation.go ✅ 推荐

### 实现方式
使用**应用层逻辑**在聚合器内部判断是否所有分支已完成。

### 关键代码

```go
// 在聚合器内部使用局部变量和条件判断
var aggregationDone bool

g.AddNode("aggregator", "aggregator", func(ctx context.Context, state any) (any, error) {
    mState := state.(map[string]any)
    results := mState["results"].([]string)

    // 只在所有分支完成时输出一次
    if len(results) == expectedBranches && !aggregationDone {
        aggregationDone = true
        fmt.Println("\n=== Aggregation Point ===")
        // ... 输出结果
        return map[string]any{
            "status": "all_branches_completed",
            // ...
        }, nil
    }

    // 还未完成，不输出
    return map[string]any{}, nil
})
```

### 图结构
```
start -> [branches] -> aggregator -> END
```

### 优点
- ✅ **简洁**：不需要额外的节点
- ✅ **轻量**：只需要在聚合器内部添加判断逻辑
- ✅ **易理解**：逻辑集中在一个地方
- ✅ **状态简单**：只使用 `results` 字段

### 缺点
- ❌ 使用闭包变量 `aggregationDone`（不是状态的一部分）
- ❌ 依赖结果数量判断（硬编码 `expectedBranches = 3`）
- ❌ 聚合器仍然被调用多次（只是不输出而已）

## main_optimized.go

### 实现方式
使用**图结构层面**添加同步屏障节点来跟踪分支完成状态。

### 关键代码

```go
// 1. 注册 completed_branches reducer
schema.RegisterReducer("completed_branches", graph.AppendReducer)

// 2. 每个分支完成时标记
g.AddNode("short_branch", "short_branch", func(...) {
    // ...
    return map[string]any{
        "results": []string{"Short branch result"},
        "completed_branches": []string{"short"},  // 标记此分支已完成
    }, nil
})

// 3. 添加同步屏障节点
g.AddNode("sync_barrier", "sync_barrier", func(ctx context.Context, state any) (any, error) {
    mState := state.(map[string]any)
    completed := mState["completed_branches"].([]string)
    totalBranches := mState["total_branches"].(int)

    if len(completed) < totalBranches {
        fmt.Printf("[Sync Barrier] Waiting... (%d/%d branches completed)\n", len(completed), totalBranches)
    } else {
        fmt.Printf("[Sync Barrier] All %d branches completed!\n", totalBranches)
    }

    return map[string]any{}, nil
})

// 4. 图结构：branches -> sync_barrier -> aggregator
```

### 图结构
```
start -> [branches] -> sync_barrier -> aggregator -> END
```

### 优点
- ✅ **状态驱动**：使用状态字段 `completed_branches` 跟踪进度
- ✅ **可观察性好**：sync_barrier 节点提供中间状态反馈
- ✅ **符合图模式**：通过额外节点实现同步
- ✅ **灵活**：可以在 sync_barrier 中添加更多逻辑（超时、重试等）

### 缺点
- ❌ **更复杂**：需要额外的节点和状态字段
- ❌ **冗余**：sync_barrier 和 aggregator 仍然会被调用多次
- ❌ **需要维护**：每个分支都要记得添加 `completed_branches` 标记

## 执行对比

### main_single_aggregation.go 执行流程
```
1. short_branch 完成 -> aggregator (不输出)
2. medium_branch_2 完成 -> aggregator (不输出)
3. long_branch_3 完成 -> aggregator (输出！✓)
```

### main_optimized.go 执行流程
```
1. short_branch 完成 -> sync_barrier (显示 1/3) -> aggregator
2. medium_branch_2 完成 -> sync_barrier (显示 2/3) -> aggregator
3. long_branch_3 完成 -> sync_barrier (显示 3/3 ✓) -> aggregator (最终输出)
```

## 什么时候用哪个？

| 场景 | 推荐版本 |
|------|---------|
| **简单的并行聚合** | main_single_aggregation.go ✅ |
| **需要跟踪进度** | main_optimized.go |
| **分支数量固定** | main_single_aggregation.go ✅ |
| **分支数量动态** | main_optimized.go |
| **需要超时处理** | main_optimized.go |
| **快速原型开发** | main_single_aggregation.go ✅ |
| **生产环境（简单场景）** | main_single_aggregation.go ✅ |
| **生产环境（复杂编排）** | main_optimized.go |

## 总结

- **main_single_aggregation.go**：简单、直接，适合大多数场景 ✅
- **main_optimized.go**：更符合图模式设计理念，适合需要可观察性和扩展性的场景

对于大多数使用场景，**推荐使用 main_single_aggregation.go**，因为它更简洁且易于维护。
