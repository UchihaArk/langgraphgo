# RFC: 泛型 StateGraph 设计

## 摘要

本 RFC 提议在 LangGraphGo 中引入**泛型（类型参数化）版本的 StateGraph**，实现编译时类型安全的状态管理，同时保持与现有基于 `any` 的实现的向后兼容性。

## 目录

- [摘要](#摘要)
- [动机](#动机)
- [当前设计分析](#当前设计分析)
- [提议的设计](#提议的设计)
- [API 设计](#api-设计)
- [实现策略](#实现策略)
- [迁移路径](#迁移路径)
- [权衡分析](#权衡分析)
- [类型映射参考](#类型映射参考)
- [示例](#示例)
- [考虑的替代方案](#考虑的替代方案)
- [决策点](#决策点)
- [参考资料](#参考资料)

## 动机

### 当前痛点

1. **类型安全**：开发者必须在节点函数中使用类型断言（`state.(MyState)`），可能导致运行时 panic
2. **IDE 支持**：由于使用 `any` 类型，自动补全和重构支持受限
3. **编译时保证**：类型不匹配只能在运行时被捕获
4. **开发体验**：需要编写冗长的代码和重复的类型断言

### 当前问题示例

```go
// 当前方法 - 需要类型断言
g.AddNode("node1", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(MyState)  // 运行时类型断言 - 可能 panic！
    s.Count++
    return s, nil
})

// 如果传入错误类型会怎样？
initialState := WrongState{} // 编译通过，运行时失败
result, err := app.Invoke(ctx, initialState)
```

### 目标

1. **类型安全**：为 state 启用编译时类型检查
2. **开发体验**：改善 IDE 支持并减少样板代码
3. **向后兼容**：保持现有非泛型 API 功能正常
4. **零性能开销**：泛型不应影响运行时性能
5. **渐进式采用**：允许从 `any` 逐步迁移到泛型类型

## 当前设计分析

### 现有架构

```go
// StateGraph 使用 'any' 获得最大灵活性
type StateGraph struct {
    nodes            map[string]Node
    conditionalEdges map[string]func(ctx context.Context, state any) string
    Schema           StateSchema
}

// 节点函数接受并返回 'any'
type NodeFunc = func(ctx context.Context, state any) (any, error)

// StateSchema 也使用 'any'
type StateSchema interface {
    Init() any
    Update(current, new any) (any, error)
}
```

### 当前使用模式

**模式 1：自定义 Struct（业务逻辑中类型安全）**
```go
type State struct {
    Count  int
    Logs   []string
}

g.AddNode("node", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(State)  // 需要类型断言
    s.Count++
    return s, nil
})
```

**模式 2：map[string]any 配合 Schema（运行时灵活性）**
```go
schema := graph.NewMapSchema()
schema.RegisterReducer("count", SumReducer)

g.AddNode("node", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(map[string]any)  // 类型断言
    s["count"] = s["count"].(int) + 1  // 更多类型断言
    return s, nil
})
```

### 为什么 `map[string]T` 不够用

`map[string]T` 要求所有值都是同一类型 `T`，但真实的 state 有异构类型：

```go
// 用 map[string]T 无法实现：
type State struct {
    Count    int        // int
    Logs     []string   // []string
    Status   string     // string
    Metadata any        // any
}

// map[string]T 只能是以下之一：
map[string]int        // 只能存 int
map[string]string     // 只能存 string
map[string]any        // 与当前设计相同
```

## 提议的设计

### 核心思想

在 StateGraph 层级引入**泛型类型参数**，同时保持现有 API：

```go
// 新：泛型版本
type StateGraphTyped[S any] struct {
    nodes            map[string]NodeTyped[S]
    conditionalEdges map[string]func(ctx context.Context, state S) string
    Schema           StateSchemaTyped[S]
}

// 现有：非泛型版本（保持不变）
type StateGraph struct {
    nodes            map[string]Node
    conditionalEdges map[string]func(ctx context.Context, state any) string
    Schema           StateSchema
}
```

### 类型参数约束

我们使用 `S any`，其中：
- `S` 是**完整的 state 类型**（通常是 struct）
- `any` 约束允许任何类型（structs、maps、primitives）
- 示例：`StateGraphTyped[MyState]`、`StateGraphTyped[map[string]any]`

### 双 API 策略

维护**两个并行的 API**：

1. **非泛型 API**（`StateGraph`）- 用于向后兼容和动态用例
2. **泛型 API**（`StateGraphTyped[S]`）- 用于类型安全的应用

两个 API 可以无冲突地共存。

## API 设计

### 泛型 StateGraph API

```go
// 构造函数
func NewStateGraphTyped[S any]() *StateGraphTyped[S]

// 带类型的节点定义
func (g *StateGraphTyped[S]) AddNode(
    name string,
    description string,
    fn func(ctx context.Context, state S) (S, error),
)

// 带类型的条件边
func (g *StateGraphTyped[S]) AddConditionalEdge(
    from string,
    condition func(ctx context.Context, state S) string,
)

// 类型化 schema
func (g *StateGraphTyped[S]) SetSchema(schema StateSchema[S])

// 编译返回类型化 runnable
func (g *StateGraphTyped[S]) Compile() (*StateRunnableTyped[S], error)
```

### 泛型 StateRunnable API

```go
type StateRunnableTyped[S any] struct {
    graph  *StateGraphTyped[S]
    tracer *Tracer
}

// 使用类型化 state 调用
func (r *StateRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error)

// 使用类型化 state 和配置调用
func (r *StateRunnableTyped[S]) InvokeWithConfig(
    ctx context.Context,
    initialState S,
    config *Config,
) (S, error)
```

### 泛型 StateSchema API

```go
type StateSchemaTyped[S any] interface {
    Init() S
    Update(current, new S) (S, error)
}

// 泛型 reducer
type Reducer[T any] func(current, new T) (T, error)

// 基于 struct 的 schema
type StructSchema[S any] struct {
    InitialValue S
    MergeFunc    func(current, new S) (S, error)
}

func NewStructSchema[S any](initial S, merge func(S, S) (S, error)) *StructSchema[S]
```

### 向后兼容 API（无变化）

```go
// 现有函数保持不变
func NewStateGraph() *StateGraph
func (g *StateGraph) AddNode(name, desc string, fn func(context.Context, any) (any, error))
func (g *StateGraph) Compile() (*StateRunnable, error)
```

## 实现策略

### 阶段 1：核心泛型类型

```go
// 文件：graph/state_graph_typed.go

package graph

// 泛型 StateGraph
type StateGraphTyped[S any] struct {
    nodes            map[string]NodeTyped[S]
    edges            []Edge
    conditionalEdges map[string]func(ctx context.Context, state S) string
    entryPoint       string
    retryPolicy      *RetryPolicy
    stateMerger      StateMergerTyped[S]
    Schema           StateSchemaTyped[S]
}

// 泛型 Node
type NodeTyped[S any] struct {
    Name        string
    Description string
    Function    func(ctx context.Context, state S) (S, error)
}

// 构造函数
func NewStateGraphTyped[S any]() *StateGraphTyped[S] {
    return &StateGraphTyped[S]{
        nodes:            make(map[string]NodeTyped[S]),
        conditionalEdges: make(map[string]func(ctx context.Context, state S) string),
    }
}
```

### 阶段 2：泛型节点操作

```go
// 带类型函数的 AddNode
func (g *StateGraphTyped[S]) AddNode(
    name string,
    description string,
    fn func(ctx context.Context, state S) (S, error),
) {
    g.nodes[name] = NodeTyped[S]{
        Name:        name,
        Description: description,
        Function:    fn,
    }
}

// 带类型条件的 AddConditionalEdge
func (g *StateGraphTyped[S]) AddConditionalEdge(
    from string,
    condition func(ctx context.Context, state S) string,
) {
    g.conditionalEdges[from] = condition
}

// 非类型化边操作（与之前相同）
func (g *StateGraphTyped[S]) AddEdge(from, to string) {
    g.edges = append(g.edges, Edge{From: from, To: to})
}

func (g *StateGraphTyped[S]) SetEntryPoint(name string) {
    g.entryPoint = name
}
```

### 阶段 3：泛型 StateRunnable

```go
// 泛型 StateRunnable
type StateRunnableTyped[S any] struct {
    graph  *StateGraphTyped[S]
    tracer *Tracer
}

// 编译
func (g *StateGraphTyped[S]) Compile() (*StateRunnableTyped[S], error) {
    if g.entryPoint == "" {
        return nil, ErrEntryPointNotSet
    }
    return &StateRunnableTyped[S]{graph: g}, nil
}

// 使用类型化 state 调用
func (r *StateRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error) {
    return r.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig 实现
func (r *StateRunnableTyped[S]) InvokeWithConfig(
    ctx context.Context,
    initialState S,
    config *Config,
) (S, error) {
    state := initialState
    currentNodes := []string{r.graph.entryPoint}

    // ... 与非泛型版本类似的逻辑
    // 但使用类型 S 而不是 any

    return state, nil
}
```

### 阶段 4：泛型 Schema 支持

```go
// 文件：graph/schema_typed.go

// 泛型 StateSchema 接口
type StateSchemaTyped[S any] interface {
    Init() S
    Update(current, new S) (S, error)
}

// 泛型 CleaningStateSchema
type CleaningStateSchemaTyped[S any] interface {
    StateSchemaTyped[S]
    Cleanup(state S) S
}

// 用于基于 struct 的 state 的 StructSchema
type StructSchema[S any] struct {
    InitialValue S
    MergeFunc    func(current, new S) (S, error)
}

func NewStructSchema[S any](initial S, merge func(S, S) (S, error)) *StructSchema[S] {
    return &StructSchema[S]{
        InitialValue: initial,
        MergeFunc:    merge,
    }
}

func (s *StructSchema[S]) Init() S {
    return s.InitialValue
}

func (s *StructSchema[S]) Update(current, new S) (S, error) {
    if s.MergeFunc != nil {
        return s.MergeFunc(current, new)
    }
    // 默认：返回新 state
    return new, nil
}
```

### 阶段 5：辅助实用程序

```go
// 文件：graph/helpers_generic.go

// 结构体状态的默认合并函数，使用反射
func DefaultStructMerge[S any](current, new S) (S, error) {
    // 使用反射将 new 中的非零字段合并到 current 中
    // 这对于大多数结构体类型是一个合理的默认值
    currentVal := reflect.ValueOf(&current).Elem()
    newVal := reflect.ValueOf(new)

    // 检查 S 是否是结构体
    if currentVal.Kind() != reflect.Struct {
        // 对于非结构体类型，只返回 new
        return new, nil
    }

    for i := 0; i < newVal.NumField(); i++ {
        fieldNew := newVal.Field(i)
        if !fieldNew.IsZero() {
            currentField := currentVal.Field(i)
            if currentField.CanSet() {
                currentField.Set(fieldNew)
            }
        }
    }
    return current, nil
}

// 字段级合并器
type FieldMerger[S any] struct {
    InitialValue  S
    FieldMergeFns map[string]func(currentVal, newVal reflect.Value) reflect.Value
}

// 常用合并助手
func AppendSliceMerge(current, new reflect.Value) reflect.Value
func SumIntMerge(current, new reflect.Value) reflect.Value
func OverwriteMerge(current, new reflect.Value) reflect.Value
func KeepCurrentMerge(current, new reflect.Value) reflect.Value
func MaxIntMerge(current, new reflect.Value) reflect.Value
func MinIntMerge(current, new reflect.Value) reflect.Value
```

## 迁移路径

### 策略：渐进式采用

迁移允许开发者逐步采用泛型：

1. **无破坏性变更**：现有代码继续工作
2. **可选迁移**：新项目可以从第一天就使用泛型
3. **按图选择**：在同一代码库中混合使用泛型和非泛型图

### 现有项目的迁移步骤

#### 步骤 1：识别 State 类型

```go
// 之前（隐式）
initialState := State{Count: 0}
app.Invoke(ctx, initialState)  // 可以工作，但 state 是 'any'

// 之后（显式）
var initialState State = State{Count: 0}
```

#### 步骤 2：更改构造函数

```go
// 之前
g := graph.NewStateGraph()

// 之后
g := graph.NewStateGraphTyped[State]()
```

#### 步骤 3：更新节点函数

```go
// 之前 - 带类型断言
g.AddNode("node1", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(State)  // 类型断言
    s.Count++
    return s, nil
})

// 之后 - 类型安全
g.AddNode("node1", "desc", func(ctx context.Context, state State) (State, error) {
    state.Count++  // 不需要类型断言！
    return state, nil
})
```

#### 步骤 4：更新条件边

```go
// 之前
g.AddConditionalEdge("node1", func(ctx context.Context, state any) string {
    s := state.(State)
    if s.Count > 10 {
        return "high"
    }
    return "low"
})

// 之后
g.AddConditionalEdge("node1", func(ctx context.Context, state State) string {
    if state.Count > 10 {
        return "high"
    }
    return "low"
})
```

#### 步骤 5：更新调用

```go
// 之前
result, err := app.Invoke(ctx, initialState)
finalState := result.(State)  // 类型断言

// 之后
finalState, err := app.Invoke(ctx, initialState)  // 类型安全！
// 不需要类型断言
```

### 兼容性矩阵

| 功能 | 非泛型 | 泛型 | 说明 |
|---------|-------------|---------|-------|
| 基本图构建 | ✅ | ✅ | 两者都支持 |
| 自定义 struct state | ✅ | ✅ | 泛型移除断言 |
| map[string]any state | ✅ | ✅ | 使用 `StateGraphTyped[map[string]any]` |
| MapSchema | ✅ | ⚠️ | 有限的泛型支持 |
| 子图 | ✅ | 🚧 | 需要设计工作 |
| Checkpointing | ✅ | ✅ | 类型安全序列化 |
| Streaming | ✅ | ✅ | 完全兼容 |

## 权衡分析

### 优势 ✅

1. **编译时类型安全**
   - 在运行前捕获类型错误
   - 不再因错误的类型断言而 panic

2. **更好的 IDE 支持**
   - state 字段的完整自动补全
   - 更好的重构工具
   - 类型感知文档

3. **更清晰的代码**
   - 没有重复的类型断言
   - 更可读的节点函数
   - 自文档化的类型

4. **性能**
   - 零运行时开销（泛型是编译时的）
   - 编译器优化的潜力

5. **渐进式迁移**
   - 无破坏性变更
   - 按自己的节奏采用

### 劣势 ❌

1. **增加代码复杂性**
   - 两个相似代码的版本
   - 更多文件需要维护
   - 泛型的学习曲线

2. **类型灵活性损失**
   - 每个图锁定到一种 state 类型
   - 不能轻易混合不同的 state 类型
   - 比 `any` 更刚性

3. **编译时间**
   - 泛型代码可能增加编译时间
   - 每次实例化创建新代码

4. **子图复杂性**
   - 父/子图之间的类型兼容性
   - 需要仔细的类型设计

5. **维护负担**
   - 两个 API 需要维护
   - 两套测试
   - 文档重复

### 何时使用每种方法

**使用泛型 StateGraphTyped[S] 当：**
- ✅ 你有定义良好的 state struct
- ✅ 类型安全很重要
- ✅ 构建新项目
- ✅ 你的团队熟悉 Go 泛型

**使用非泛型 StateGraph 当：**
- ✅ 你需要最大的灵活性
- ✅ State 结构是动态的
- ✅ 使用带有复杂 reducer 的 map[string]any
- ✅ 从 Python LangGraph 迁移
- ✅ 原型设计或实验

## 示例

### 示例 1：简单计数器（泛型）

```go
package main

import (
    "context"
    "fmt"
    "github.com/smallnest/langgraphgo/graph"
)

type CounterState struct {
    Count int
    Name  string
}

func main() {
    // 创建泛型图
    g := graph.NewStateGraphTyped[CounterState]()

    // 添加节点 - 完全类型安全！
    g.AddNode("increment", "Increment counter", func(ctx context.Context, state CounterState) (CounterState, error) {
        state.Count++  // 不需要类型断言！
        return state, nil
    })

    g.AddNode("print", "Print result", func(ctx context.Context, state CounterState) (CounterState, error) {
        fmt.Printf("%s: %d\n", state.Name, state.Count)
        return state, nil
    })

    // 添加边
    g.SetEntryPoint("increment")
    g.AddEdge("increment", "print")
    g.AddEdge("print", graph.END)

    // 编译
    app, _ := g.Compile()

    // 调用 - 类型安全！
    initialState := CounterState{Count: 0, Name: "MyCounter"}
    finalState, err := app.Invoke(context.Background(), initialState)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Final count: %d\n", finalState.Count)  // 类型安全访问！
}
```

### 示例 2：条件分支（泛型）

```go
type WorkflowState struct {
    Value    int
    Approved bool
    Result   string
}

func main() {
    g := graph.NewStateGraphTyped[WorkflowState]()

    g.AddNode("check", "Check value", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        // 类型安全的字段访问
        if state.Value > 100 {
            state.Approved = true
        }
        return state, nil
    })

    g.AddNode("process_high", "Process high value", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        state.Result = fmt.Sprintf("High value: %d", state.Value)
        return state, nil
    })

    g.AddNode("process_low", "Process low value", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        state.Result = fmt.Sprintf("Low value: %d", state.Value)
        return state, nil
    })

    // 类型安全的条件边
    g.SetEntryPoint("check")
    g.AddConditionalEdge("check", func(ctx context.Context, state WorkflowState) string {
        if state.Approved {  // 不需要类型断言！
            return "process_high"
        }
        return "process_low"
    })

    g.AddEdge("process_high", graph.END)
    g.AddEdge("process_low", graph.END)

    app, _ := g.Compile()
    result, _ := app.Invoke(context.Background(), WorkflowState{Value: 150})
    fmt.Println(result.Result)  // 类型安全！
}
```

### 示例 3：带 Schema（泛型）

```go
type AgentState struct {
    Messages []string
    Steps    int
    MaxSteps int
}

func main() {
    g := graph.NewStateGraphTyped[AgentState]()

    // 定义合并逻辑
    schema := graph.NewStructSchema(
        AgentState{MaxSteps: 10},
        func(current, new AgentState) (AgentState, error) {
            // 合并 messages（追加）
            current.Messages = append(current.Messages, new.Messages...)
            // 覆盖 steps
            current.Steps = new.Steps
            // 保留初始的 MaxSteps
            return current, nil
        },
    )

    g.SetSchema(schema)

    g.AddNode("process", "Process", func(ctx context.Context, state AgentState) (AgentState, error) {
        return AgentState{
            Messages: []string{fmt.Sprintf("Processed step %d", state.Steps)},
            Steps:    state.Steps + 1,
        }, nil
    })

    g.SetEntryPoint("process")
    g.AddConditionalEdge("process", func(ctx context.Context, state AgentState) string {
        if state.Steps >= state.MaxSteps {
            return graph.END
        }
        return "process"
    })

    app, _ := g.Compile()
    result, _ := app.Invoke(context.Background(), AgentState{})

    fmt.Printf("Executed %d steps\n", result.Steps)
    fmt.Printf("Messages: %v\n", result.Messages)
}
```

### 示例 4：迁移对比

```go
// 之前：非泛型版本
func createGraphOld() *graph.StateRunnable {
    g := graph.NewStateGraph()

    g.AddNode("node1", "desc", func(ctx context.Context, state any) (any, error) {
        s := state.(MyState)  // 类型断言
        s.Count++
        return s, nil
    })

    g.SetEntryPoint("node1")
    g.AddEdge("node1", graph.END)

    app, _ := g.Compile()
    return app
}

func runOld() {
    app := createGraphOld()
    result, _ := app.Invoke(context.Background(), MyState{Count: 0})
    finalState := result.(MyState)  // 类型断言
    fmt.Println(finalState.Count)
}

// 之后：泛型版本
func createGraphNew() *graph.StateRunnableTyped[MyState] {
    g := graph.NewStateGraphTyped[MyState]()

    g.AddNode("node1", "desc", func(ctx context.Context, state MyState) (MyState, error) {
        state.Count++  // 不需要类型断言！
        return state, nil
    })

    g.SetEntryPoint("node1")
    g.AddEdge("node1", graph.END)

    app, _ := g.Compile()
    return app
}

func runNew() {
    app := createGraphNew()
    finalState, _ := app.Invoke(context.Background(), MyState{Count: 0})
    fmt.Println(finalState.Count)  // 不需要类型断言！
}
```

## 类型映射参考

本节提供了非泛型和泛型类型之间的全面映射，以便在迁移过程中参考。

### 核心类型

| 非泛型类型 | 泛型类型 | 描述 |
|-----------|---------|------|
| `StateGraph` | `StateGraphTyped[S any]` | 主要的状态管理图结构 |
| `StateRunnable` | `StateRunnableTyped[S any]` | 已编译的可执行图 |
| `Node` | `NodeTyped[S any]` | 单个图节点 |
| `StateSchema` | `StateSchemaTyped[S any]` | 状态结构和更新逻辑的接口 |
| `StateMerger` | `StateMergerTyped[S any]` | 并行执行时合并状态的函数类型 |

### 模式实现

| 非泛型类型 | 泛型类型 | 描述 |
|-----------|---------|------|
| `StructSchema` | `StructSchema[S any]` | 基于结构体的模式实现 |
| `MapSchema` | N/A | 使用 `StateGraphTyped[map[string]any]` |
| `CleaningStateSchema` | `CleaningStateSchemaTyped[S any]` | 具有清理功能的模式 |
| `FieldMerger` | `FieldMerger[S any]` | 细粒度字段级合并 |

### 监听器类型

| 非泛型类型 | 泛型类型 | 描述 |
|-----------|---------|------|
| `NodeListener` | `NodeListenerTyped[S any]` | 节点事件监听器接口 |
| `NodeListenerFunc` | `NodeListenerTypedFunc[S any]` | 节点监听器的函数适配器 |
| `StreamEvent` | `StreamEventTyped[S any]` | 带类型化状态的事件结构 |
| `ListenableNode` | `ListenableNodeTyped[S any]` | 具有监听器功能的节点 |
| `ListenableStateGraph` | `ListenableStateGraphTyped[S any]` | 带监听器支持的状态图 |
| `ListenableRunnable` | `ListenableRunnableTyped[S any]` | 带监听器的可运行图 |

### 预构建 Agent

| 非泛型类型 | 泛型类型 | 描述 |
|-----------|---------|------|
| N/A | `SupervisorState` | 监督器模式的状态类型 |
| N/A | `ReactAgentState` | ReAct Agent 模式的状态类型 |
| N/A | `CreateSupervisorTyped()` | 创建类型化监督器图 |
| N/A | `CreateReactAgentTyped()` | 创建类型化 ReAct Agent 图 |

### 构造函数

| 非泛型函数 | 泛型函数 | 描述 |
|-----------|---------|------|
| `NewStateGraph()` | `NewStateGraphTyped[S any]()` | 创建新的状态图 |
| `NewStructSchema(initial)` | `NewStructSchema[S any](initial S, merge func(S, S) (S, error))` | 创建结构体模式 |
| `NewListenableStateGraph()` | `NewListenableStateGraphTyped[S any]()` | 创建可监听的图 |
| `NewListenableNode(node)` | `NewListenableNodeTyped[S any](node NodeTyped[S])` | 创建可监听的节点 |

### 方法签名

| 非泛型方法 | 泛型方法 | 描述 |
|-----------|---------|------|
| `AddNode(name, desc string, fn func(context.Context, any) (any, error))` | `AddNode(name, desc string, fn func(context.Context, S) (S, error))` | 向图添加节点 |
| `AddConditionalEdge(from string, condition func(context.Context, any) string)` | `AddConditionalEdge(from string, condition func(context.Context, S) string)` | 添加条件边 |
| `Invoke(ctx context.Context, state any) (any, error)` | `Invoke(ctx context.Context, state S) (S, error)` | 执行图 |
| `OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state any, err error)` | `OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error)` | 处理节点事件 |

### 迁移示例

#### 基本图构建
```go
// 非泛型
g := graph.NewStateGraph()
g.AddNode("node", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(MyState)
    s.Count++
    return s, nil
})

// 泛型
g := graph.NewStateGraphTyped[MyState]()
g.AddNode("node", "desc", func(ctx context.Context, state MyState) (MyState, error) {
    state.Count++
    return state, nil
})
```

#### 模式定义
```go
// 非泛型使用 MapSchema
schema := graph.NewMapSchema()
schema.RegisterReducer("count", graph.SumReducer)

// 泛型使用 StructSchema
schema := graph.NewStructSchema(
    MyState{MaxCount: 10},
    func(current, new MyState) (MyState, error) {
        current.Count += new.Count
        if current.Count > current.MaxCount {
            current.Count = current.MaxCount
        }
        return current, nil
    },
)
```

#### 监听器
```go
// 非泛型监听器
type MyListener struct{}
func (l *MyListener) OnNodeEvent(ctx context.Context, event graph.NodeEvent, nodeName string, state any, err error) {
    s := state.(MyState)
    fmt.Printf("%s: count=%d\n", nodeName, s.Count)
}

// 泛型监听器
type MyListenerTyped struct{}
func (l *MyListenerTyped) OnNodeEvent(ctx context.Context, event graph.NodeEvent, nodeName string, state MyState, err error) {
    fmt.Printf("%s: count=%d\n", nodeName, state.Count)
}
```

## 考虑的替代方案

### 替代方案 1：仅使用类型断言（现状）

**方法**：保持当前设计，在文档中推荐类型断言

**优点**：
- 无需实现工作
- 最大灵活性
- 简单设计

**缺点**：
- 无编译时安全性
- 糟糕的开发体验
- 冗长的代码

**决定**：拒绝 - 未解决核心痛点

### 替代方案 2：代码生成

**方法**：从类型定义生成类型安全的包装器

```go
//go:generate langgraph-gen -type=MyState -output=graph_gen.go
type MyState struct {
    Count int
}
```

**优点**：
- 可以生成最优代码
- 完全类型安全
- 无运行时开销

**缺点**：
- 构建复杂性
- 工具依赖
- 有限的灵活性
- 维护负担

**决定**：拒绝 - 对收益来说过于复杂

### 替代方案 3：基于接口的类型安全

**方法**：将 state 定义为接口，为方法使用类型参数

```go
type State interface {
    GetCount() int
    SetCount(int)
}

g := graph.NewStateGraphTyped[State]()
```

**优点**：
- 接口灵活性
- 类型安全

**缺点**：
- 冗长的接口定义
- 不如 struct 符合人体工程学
- 有限的值访问模式

**决定**：拒绝 - 过于限制性

### 替代方案 4：宏/模板系统

**方法**：使用文本模板或宏生成特定类型的代码

**优点**：
- 完全控制生成的代码
- 可以为特定类型优化

**缺点**：
- 不符合 Go 习惯
- 复杂的构建过程
- 调试困难

**决定**：拒绝 - 不是 Go 的方式

### 替代方案 5：双实现（已选择）

**方法**：维护泛型和非泛型版本

**优点**：
- 向后兼容
- 渐进式迁移
- 两全其美
- 符合 Go 习惯

**缺点**：
- 代码重复
- 维护开销

**决定**：**接受** - 安全性和灵活性的最佳平衡

## 决策点

### 1. 我们应该弃用非泛型 API 吗？

**决定**：**否**

**理由**：
- 许多用例受益于动态类型
- MapSchema 更适合非泛型
- 破坏性变更会损害生态系统

### 2. StateSchema 应该是泛型的吗？

**决定**：**是的，并行的非泛型版本**

**理由**：
- Schema 需要匹配状态类型
- 泛型模式启用类型安全合并
- 为 MapSchema 保留非泛型

### 3. 我们应该支持混合泛型/非泛型图吗？

**决定**：**无直接互操作，但允许适配器**

**理由**：
- 类型安全会受损
- 可以在需要时提供转换助手

### 4. 如何处理子图？

**决定**：**推迟到未来的 RFC**

**理由**：
- 子图需要仔细的类型设计
- 可能需要变异或类型边界
- 最好先把核心做对

### 5. 我们应该添加泛型 MapSchema 吗？

**决定**：**可选的未来增强**

**理由**：
- map[string]T 过于限制性
- map[string]any 适用于 StateGraphTyped[map[string]any]
- 字段级泛型需要复杂的类型系统

### 6. 实现时间表？

**提议的阶段**：
1. **阶段 1（MVP）**：核心泛型 StateGraphTyped[S] 和 StateRunnableTyped[S]
2. **阶段 2**：泛型 StateSchema[S] 和 StructSchema[S]
3. **阶段 3**：文档和示例
4. **阶段 4**：社区反馈和迭代
5. **阶段 5**：高级功能（子图等）

## 参考资料

### 相关 RFC

- [RFC: Channels 架构](./RFC_CHANNELS.md)

### Go 泛型资源

- [Go 泛型提案](https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md)
- [Go 泛型教程](https://go.dev/doc/tutorial/generics)

### Python LangGraph 对比

Python LangGraph 使用带有 TypedDict 的运行时类型：
```python
class State(TypedDict):
    count: int
    logs: list[str]
```

Go 的编译时泛型提供比 Python 的运行时注解更强的保证。

### Go 生态系统中的类似模式

- **泛型 Channel**：标准库 channel 是泛型的（`chan T`）
- **泛型集合**：各种第三方库
- **泛型 Option 类型**：函数式编程库

---

## 结论

本 RFC 提出了一个**务实的方法**，通过泛型为 LangGraphGo 添加类型安全，同时保持使框架有用的灵活性。

**双 API 策略**允许：
- ✅ 新项目立即受益于类型安全
- ✅ 现有项目无需更改即可继续工作
- ✅ 以项目自己的节奏渐进式迁移
- ✅ 需要动态类型的用例继续得到支持

### 下一步

1. **社区审查**：收集关于此设计的反馈
2. **原型实现**：构建泛型 StateGraph 的 MVP
3. **示例迁移**：转换一个 showcase 来演示迁移路径
4. **文档**：创建迁移指南
5. **发布**：作为实验性功能发布，根据反馈迭代

### 开放问题

1. 我们是否应该为泛型图提供类型安全的检查点？
2. 流式传输应如何与泛型类型一起工作？
3. 我们是否应该为常见状态模式（例如，消息列表）添加助手？
4. 处理泛型类型的状态序列化的最佳方式是什么？

**欢迎反馈**：请在 GitHub issues 或 discussions 中分享您对本 RFC 的想法。
