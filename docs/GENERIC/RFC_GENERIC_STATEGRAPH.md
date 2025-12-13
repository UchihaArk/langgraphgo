# RFC: Generic StateGraph Design

## Summary

This RFC proposes introducing a **generic (type-parameterized) version of StateGraph** to LangGraphGo, enabling compile-time type safety for state management while maintaining backward compatibility with the existing `any`-based implementation.

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
- [Current Design Analysis](#current-design-analysis)
- [Proposed Design](#proposed-design)
- [API Design](#api-design)
- [Implementation Strategy](#implementation-strategy)
- [Migration Path](#migration-path)
- [Trade-offs Analysis](#trade-offs-analysis)
- [Type Mapping Reference](#type-mapping-reference)
- [Examples](#examples)
- [Alternatives Considered](#alternatives-considered)
- [Decision Points](#decision-points)
- [References](#references)

## Motivation

### Current Pain Points

1. **Type Safety**: Developers must use type assertions (`state.(MyState)`) in node functions, leading to potential runtime panics
2. **IDE Support**: Limited autocomplete and refactoring support due to `any` types
3. **Compile-time Guarantees**: Type mismatches are only caught at runtime
4. **Developer Experience**: Verbose code with repetitive type assertions

### Example of Current Issues

```go
// Current approach - requires type assertion
g.AddNode("node1", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(MyState)  // Runtime type assertion - can panic!
    s.Count++
    return s, nil
})

// What if someone passes the wrong type?
initialState := WrongState{} // Compiles fine, fails at runtime
result, err := app.Invoke(ctx, initialState)
```

### Goals

1. **Type Safety**: Enable compile-time type checking for state
2. **Developer Experience**: Improve IDE support and reduce boilerplate
3. **Backward Compatibility**: Keep existing non-generic API functional
4. **Zero Performance Overhead**: Generics should not impact runtime performance
5. **Progressive Adoption**: Allow gradual migration from `any` to generic types

## Current Design Analysis

### Existing Architecture

```go
// StateGraph uses 'any' for maximum flexibility
type StateGraph struct {
    nodes            map[string]Node
    conditionalEdges map[string]func(ctx context.Context, state any) string
    Schema           StateSchema
}

// Node functions accept and return 'any'
type NodeFunc = func(ctx context.Context, state any) (any, error)

// StateSchema also uses 'any'
type StateSchema interface {
    Init() any
    Update(current, new any) (any, error)
}
```

### Current Usage Patterns

**Pattern 1: Custom Struct (Type-safe in business logic)**
```go
type State struct {
    Count  int
    Logs   []string
}

g.AddNode("node", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(State)  // Type assertion required
    s.Count++
    return s, nil
})
```

**Pattern 2: map[string]any with Schema (Runtime flexibility)**
```go
schema := graph.NewMapSchema()
schema.RegisterReducer("count", SumReducer)

g.AddNode("node", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(map[string]any)  // Type assertion
    s["count"] = s["count"].(int) + 1  // More type assertions
    return s, nil
})
```

### Why `map[string]T` is Not Sufficient

`map[string]T` requires all values to be of the same type `T`, but real state has heterogeneous types:

```go
// This is impossible with map[string]T:
type State struct {
    Count    int        // int
    Logs     []string   // []string
    Status   string     // string
    Metadata any        // any
}

// map[string]T can only be one of:
map[string]int        // Only ints
map[string]string     // Only strings
map[string]any        // Same as current design
```

## Proposed Design

### Core Idea

Introduce **generic type parameters** at the StateGraph level while keeping the existing API:

```go
// New: Generic version
type StateGraphTyped[S any] struct {
    nodes            map[string]Node[S]
    conditionalEdges map[string]func(ctx context.Context, state S) string
    Schema           StateSchema[S]
}

// Existing: Non-generic version (unchanged)
type StateGraph struct {
    nodes            map[string]Node
    conditionalEdges map[string]func(ctx context.Context, state any) string
    Schema           StateSchema
}
```

### Type Parameter Constraint

We use `S any` where:
- `S` is the **complete state type** (typically a struct)
- `any` constraint allows any type (structs, maps, primitives)
- Examples: `StateGraph[MyState]`, `StateGraph[map[string]any]`

### Dual API Strategy

Maintain **two parallel APIs**:

1. **Non-generic API** (`StateGraph`) - for backward compatibility and dynamic use cases
2. **Generic API** (`StateGraphTyped[S]`) - for type-safe applications

Both APIs can coexist without conflicts.

## API Design

### Generic StateGraph API

```go
// Constructor
func NewStateGraphTyped[S any]() *StateGraphTyped[S]

// Node definition with typed function
func (g *StateGraphTyped[S]) AddNode(
    name string,
    description string,
    fn func(ctx context.Context, state S) (S, error),
)

// Conditional edges with typed function
func (g *StateGraphTyped[S]) AddConditionalEdge(
    from string,
    condition func(ctx context.Context, state S) string,
)

// Typed schema
func (g *StateGraphTyped[S]) SetSchema(schema StateSchema[S])

// Compilation returns typed runnable
func (g *StateGraphTyped[S]) Compile() (*StateRunnableTyped[S], error)
```

### Generic StateRunnable API

```go
type StateRunnableTyped[S any] struct {
    graph  *StateGraphTyped[S]
    tracer *Tracer
}

// Invoke with typed state
func (r *StateRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error)

// InvokeWithConfig with typed state
func (r *StateRunnableTyped[S]) InvokeWithConfig(
    ctx context.Context,
    initialState S,
    config *Config,
) (S, error)
```

### Generic StateSchema API

```go
type StateSchema[S any] interface {
    Init() S
    Update(current, new S) (S, error)
}

// Generic reducer
type Reducer[T any] func(current, new T) (T, error)

// Struct-based schema
type StructSchema[S any] struct {
    InitialValue S
    MergeFunc    func(current, new S) (S, error)
}

func NewStructSchema[S any](initial S) *StructSchema[S]
```

### Backward Compatible API (No Changes)

```go
// Existing functions remain unchanged
func NewStateGraph() *StateGraph
func (g *StateGraph) AddNode(name, desc string, fn func(context.Context, any) (any, error))
func (g *StateGraph) Compile() (*StateRunnable, error)
```

## Implementation Strategy

### Phase 1: Core Generic Types

```go
// File: graph/state_graph_generic.go

package graph

// Generic StateGraph
type StateGraphTyped[S any] struct {
    nodes            map[string]Node[S]
    edges            []Edge
    conditionalEdges map[string]func(ctx context.Context, state S) string
    entryPoint       string
    retryPolicy      *RetryPolicy
    Schema           StateSchema[S]
}

// Generic Node
type Node[S any] struct {
    Name        string
    Description string
    Function    func(ctx context.Context, state S) (S, error)
}

// Constructor
func NewStateGraphTyped[S any]() *StateGraphTyped[S] {
    return &StateGraphTyped[S]{
        nodes:            make(map[string]Node[S]),
        conditionalEdges: make(map[string]func(ctx context.Context, state S) string),
    }
}
```

### Phase 2: Generic Node Operations

```go
// AddNode with typed function
func (g *StateGraphTyped[S]) AddNode(
    name string,
    description string,
    fn func(ctx context.Context, state S) (S, error),
) {
    g.nodes[name] = Node[S]{
        Name:        name,
        Description: description,
        Function:    fn,
    }
}

// AddConditionalEdge with typed condition
func (g *StateGraphTyped[S]) AddConditionalEdge(
    from string,
    condition func(ctx context.Context, state S) string,
) {
    g.conditionalEdges[from] = condition
}

// Non-typed edge operations (same as before)
func (g *StateGraphTyped[S]) AddEdge(from, to string) {
    g.edges = append(g.edges, Edge{From: from, To: to})
}

func (g *StateGraphTyped[S]) SetEntryPoint(name string) {
    g.entryPoint = name
}
```

### Phase 3: Generic StateRunnable

```go
// Generic StateRunnable
type StateRunnableTyped[S any] struct {
    graph  *StateGraphTyped[S]
    tracer *Tracer
}

// Compile
func (g *StateGraphTyped[S]) Compile() (*StateRunnableTyped[S], error) {
    if g.entryPoint == "" {
        return nil, ErrEntryPointNotSet
    }
    return &StateRunnableTyped[S]{graph: g}, nil
}

// Invoke with typed state
func (r *StateRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error) {
    return r.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig implementation
func (r *StateRunnableTyped[S]) InvokeWithConfig(
    ctx context.Context,
    initialState S,
    config *Config,
) (S, error) {
    state := initialState
    currentNodes := []string{r.graph.entryPoint}

    // ... similar logic to non-generic version
    // but with type S instead of any

    return state, nil
}
```

### Phase 4: Generic Schema Support

```go
// File: graph/schema_generic.go

// Generic StateSchema interface
type StateSchema[S any] interface {
    Init() S
    Update(current, new S) (S, error)
}

// Generic CleaningStateSchema
type CleaningStateSchema[S any] interface {
    StateSchema[S]
    Cleanup(state S) S
}

// StructSchema for struct-based states
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
    // Default: return new state
    return new, nil
}
```

### Phase 5: Helper Utilities

```go
// File: graph/helpers_generic.go

// Default merge function for struct states using reflection
func DefaultStructMerge[S any](current, new S) (S, error) {
    // Use reflection to merge non-zero fields from new into current
    // This is a sensible default for most struct types
    currentVal := reflect.ValueOf(&current).Elem()
    newVal := reflect.ValueOf(new)

    for i := 0; i < newVal.NumField(); i++ {
        fieldNew := newVal.Field(i)
        if !fieldNew.IsZero() {
            currentVal.Field(i).Set(fieldNew)
        }
    }
    return current, nil
}

// ShallowCopy creates a shallow copy of a struct
func ShallowCopy[S any](state S) S {
    // Implementation using reflection
    // ...
}
```

## Migration Path

### Strategy: Gradual Adoption

The migration allows developers to adopt generics incrementally:

1. **No Breaking Changes**: Existing code continues to work
2. **Opt-in Migration**: New projects can use generics from day one
3. **Per-Graph Choice**: Mix generic and non-generic graphs in same codebase

### Migration Steps for Existing Projects

#### Step 1: Identify State Type

```go
// Before (implicit)
initialState := State{Count: 0}
app.Invoke(ctx, initialState)  // Works, but state is 'any'

// After (explicit)
var initialState State = State{Count: 0}
```

#### Step 2: Change Constructor

```go
// Before
g := graph.NewStateGraph()

// After
g := graph.NewStateGraphTyped[State]()
```

#### Step 3: Update Node Functions

```go
// Before - with type assertion
g.AddNode("node1", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(State)  // Type assertion
    s.Count++
    return s, nil
})

// After - type safe
g.AddNode("node1", "desc", func(ctx context.Context, state State) (State, error) {
    state.Count++  // No type assertion needed!
    return state, nil
})
```

#### Step 4: Update Conditional Edges

```go
// Before
g.AddConditionalEdge("node1", func(ctx context.Context, state any) string {
    s := state.(State)
    if s.Count > 10 {
        return "high"
    }
    return "low"
})

// After
g.AddConditionalEdge("node1", func(ctx context.Context, state State) string {
    if state.Count > 10 {
        return "high"
    }
    return "low"
})
```

#### Step 5: Update Invocation

```go
// Before
result, err := app.Invoke(ctx, initialState)
finalState := result.(State)  // Type assertion

// After
finalState, err := app.Invoke(ctx, initialState)  // Type-safe!
// No type assertion needed
```

### Compatibility Matrix

| Feature | Non-Generic | Generic | Notes |
|---------|-------------|---------|-------|
| Basic graph construction | ✅ | ✅ | Both supported |
| Custom struct state | ✅ | ✅ | Generics remove assertions |
| map[string]any state | ✅ | ✅ | Use `StateGraph[map[string]any]` |
| MapSchema | ✅ | ⚠️ | Limited generic support |
| Subgraphs | ✅ | 🚧 | Needs design work |
| Checkpointing | ✅ | ✅ | Type-safe serialization |
| Streaming | ✅ | ✅ | Fully compatible |

## Trade-offs Analysis

### Advantages ✅

1. **Compile-Time Type Safety**
   - Catch type errors before runtime
   - No more panic from incorrect type assertions

2. **Better IDE Support**
   - Full autocomplete on state fields
   - Better refactoring tools
   - Type-aware documentation

3. **Cleaner Code**
   - No repetitive type assertions
   - More readable node functions
   - Self-documenting types

4. **Performance**
   - Zero runtime overhead (generics are compile-time)
   - Potential for compiler optimizations

5. **Gradual Migration**
   - No breaking changes
   - Adopt at your own pace

### Disadvantages ❌

1. **Increased Code Complexity**
   - Two versions of similar code
   - More files to maintain
   - Learning curve for generics

2. **Type Flexibility Loss**
   - Each graph locked to one state type
   - Can't easily mix different state types
   - More rigid than `any`

3. **Compilation Time**
   - Generic code may increase compile time
   - Each instantiation creates new code

4. **Subgraph Complexity**
   - Type compatibility between parent/child graphs
   - Requires careful type design

5. **Maintenance Burden**
   - Two APIs to maintain
   - Two sets of tests
   - Documentation duplication

### When to Use Each Approach

**Use Generic StateGraphTyped[S] when:**
- ✅ You have a well-defined state struct
- ✅ Type safety is important
- ✅ Building a new project
- ✅ Your team is comfortable with Go generics

**Use Non-Generic StateGraph when:**
- ✅ You need maximum flexibility
- ✅ State structure is dynamic
- ✅ Using map[string]any with complex reducers
- ✅ Migrating from Python LangGraph
- ✅ Prototyping or experimentation

## Examples

### Example 1: Simple Counter (Generic)

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
    // Create generic graph
    g := graph.NewStateGraphTyped[CounterState]()

    // Add nodes - fully type-safe!
    g.AddNode("increment", "Increment counter", func(ctx context.Context, state CounterState) (CounterState, error) {
        state.Count++  // No type assertion!
        return state, nil
    })

    g.AddNode("print", "Print result", func(ctx context.Context, state CounterState) (CounterState, error) {
        fmt.Printf("%s: %d\n", state.Name, state.Count)
        return state, nil
    })

    // Add edges
    g.SetEntryPoint("increment")
    g.AddEdge("increment", "print")
    g.AddEdge("print", graph.END)

    // Compile
    app, _ := g.Compile()

    // Invoke - type-safe!
    initialState := CounterState{Count: 0, Name: "MyCounter"}
    finalState, err := app.Invoke(context.Background(), initialState)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Final count: %d\n", finalState.Count)  // Type-safe access!
}
```

### Example 2: Conditional Branching (Generic)

```go
type WorkflowState struct {
    Value    int
    Approved bool
    Result   string
}

func main() {
    g := graph.NewStateGraphTyped[WorkflowState]()

    g.AddNode("check", "Check value", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        // Type-safe field access
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

    // Type-safe conditional edge
    g.SetEntryPoint("check")
    g.AddConditionalEdge("check", func(ctx context.Context, state WorkflowState) string {
        if state.Approved {  // No type assertion!
            return "process_high"
        }
        return "process_low"
    })

    g.AddEdge("process_high", graph.END)
    g.AddEdge("process_low", graph.END)

    app, _ := g.Compile()
    result, _ := app.Invoke(context.Background(), WorkflowState{Value: 150})
    fmt.Println(result.Result)  // Type-safe!
}
```

### Example 3: With Schema (Generic)

```go
type AgentState struct {
    Messages []string
    Steps    int
    MaxSteps int
}

func main() {
    g := graph.NewStateGraphTyped[AgentState]()

    // Define merge logic
    schema := graph.NewStructSchema(
        AgentState{MaxSteps: 10},
        func(current, new AgentState) (AgentState, error) {
            // Merge messages (append)
            current.Messages = append(current.Messages, new.Messages...)
            // Overwrite steps
            current.Steps = new.Steps
            // Keep MaxSteps from initial
            return current, nil
        },
    )

    g.SetSchema(schema)

    g.AddNode("process", "Process", func(ctx context.Context, state AgentState) (AgentState, error) {
        return AgentState{
            Messages: []string{"Processed step " + fmt.Sprint(state.Steps)},
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

### Example 4: Migration Comparison

```go
// BEFORE: Non-generic version
func createGraphOld() *graph.StateRunnable {
    g := graph.NewStateGraph()

    g.AddNode("node1", "desc", func(ctx context.Context, state any) (any, error) {
        s := state.(MyState)  // Type assertion
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
    finalState := result.(MyState)  // Type assertion
    fmt.Println(finalState.Count)
}

// AFTER: Generic version
func createGraphNew() *graph.StateRunnable[MyState] {
    g := graph.NewStateGraphTyped[MyState]()

    g.AddNode("node1", "desc", func(ctx context.Context, state MyState) (MyState, error) {
        state.Count++  // No type assertion!
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
    fmt.Println(finalState.Count)  // No type assertion!
}
```

## Type Mapping Reference

This section provides a comprehensive mapping between non-generic and generic types for easy reference during migration.

### Core Types

| Non-Generic Type | Generic Type | Description |
|-----------------|-------------|-------------|
| `StateGraph` | `StateGraphTyped[S any]` | Main graph structure for state management |
| `StateRunnable` | `StateRunnableTyped[S any]` | Compiled graph ready for execution |
| `Node` | `NodeTyped[S any]` | Individual graph node |
| `StateSchema` | `StateSchemaTyped[S any]` | Interface for state structure and update logic |
| `StateMerger` | `StateMergerTyped[S any]` | Function type for merging states from parallel execution |

### Schema Implementations

| Non-Generic Type | Generic Type | Description |
|-----------------|-------------|-------------|
| `StructSchema` | `StructSchema[S any]` | Schema implementation for struct-based states |
| `MapSchema` | N/A | Use `StateGraphTyped[map[string]any]` |
| `CleaningStateSchema` | `CleaningStateSchemaTyped[S any]` | Schema with cleanup capabilities |
| `FieldMerger` | `FieldMerger[S any]` | Fine-grained field-level merging |

### Listener Types

| Non-Generic Type | Generic Type | Description |
|-----------------|-------------|-------------|
| `NodeListener` | `NodeListenerTyped[S any]` | Interface for node event listeners |
| `NodeListenerFunc` | `NodeListenerTypedFunc[S any]` | Function adapter for node listeners |
| `StreamEvent` | `StreamEventTyped[S any]` | Event structure with typed state |
| `ListenableNode` | `ListenableNodeTyped[S any]` | Node with listener capabilities |
| `ListenableStateGraph` | `ListenableStateGraphTyped[S any]` | State graph with listener support |
| `ListenableRunnable` | `ListenableRunnableTyped[S any]` | Runnable with listener support |

### Prebuilt Agents

| Non-Generic Type | Generic Type | Description |
|-----------------|-------------|-------------|
| N/A | `SupervisorState` | State type for supervisor pattern |
| N/A | `ReactAgentState` | State type for ReAct agent pattern |
| N/A | `CreateSupervisorTyped()` | Creates typed supervisor graph |
| N/A | `CreateReactAgentTyped()` | Creates typed ReAct agent graph |

### Constructor Functions

| Non-Generic Function | Generic Function | Description |
|---------------------|----------------|-------------|
| `NewStateGraph()` | `NewStateGraphTyped[S any]()` | Creates a new state graph |
| `NewStructSchema(initial)` | `NewStructSchema[S any](initial S, merge func(S, S) (S, error))` | Creates struct schema |
| `NewListenableStateGraph()` | `NewListenableStateGraphTyped[S any]()` | Creates listenable graph |
| `NewListenableNode(node)` | `NewListenableNodeTyped[S any](node NodeTyped[S])` | Creates listenable node |

### Method Signatures

| Non-Generic Method | Generic Method | Description |
|-------------------|---------------|-------------|
| `AddNode(name, desc string, fn func(context.Context, any) (any, error))` | `AddNode(name, desc string, fn func(context.Context, S) (S, error))` | Adds node to graph |
| `AddConditionalEdge(from string, condition func(context.Context, any) string)` | `AddConditionalEdge(from string, condition func(context.Context, S) string)` | Adds conditional edge |
| `Invoke(ctx context.Context, state any) (any, error)` | `Invoke(ctx context.Context, state S) (S, error)` | Executes graph |
| `OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state any, err error)` | `OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error)` | Handles node events |

### Migration Examples

#### Basic Graph Construction
```go
// Non-generic
g := graph.NewStateGraph()
g.AddNode("node", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(MyState)
    s.Count++
    return s, nil
})

// Generic
g := graph.NewStateGraphTyped[MyState]()
g.AddNode("node", "desc", func(ctx context.Context, state MyState) (MyState, error) {
    state.Count++
    return state, nil
})
```

#### Schema Definition
```go
// Non-generic with MapSchema
schema := graph.NewMapSchema()
schema.RegisterReducer("count", graph.SumReducer)

// Generic with StructSchema
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

#### Listeners
```go
// Non-generic listener
type MyListener struct{}
func (l *MyListener) OnNodeEvent(ctx context.Context, event graph.NodeEvent, nodeName string, state any, err error) {
    s := state.(MyState)
    fmt.Printf("%s: count=%d\n", nodeName, s.Count)
}

// Generic listener
type MyListenerTyped struct{}
func (l *MyListenerTyped) OnNodeEvent(ctx context.Context, event graph.NodeEvent, nodeName string, state MyState, err error) {
    fmt.Printf("%s: count=%d\n", nodeName, state.Count)
}
```

## Alternatives Considered

### Alternative 1: Type Assertions Only (Status Quo)

**Approach**: Keep current design, recommend type assertions in documentation

**Pros**:
- No implementation work
- Maximum flexibility
- Simple design

**Cons**:
- No compile-time safety
- Poor developer experience
- Verbose code

**Decision**: Rejected - doesn't address core pain points

### Alternative 2: Code Generation

**Approach**: Generate type-safe wrappers from type definitions

```go
//go:generate langgraph-gen -type=MyState -output=graph_gen.go
type MyState struct {
    Count int
}
```

**Pros**:
- Can generate optimal code
- Full type safety
- No runtime overhead

**Cons**:
- Build complexity
- Tool dependency
- Limited flexibility
- Maintenance burden

**Decision**: Rejected - too complex for the benefit

### Alternative 3: Interface-Based Type Safety

**Approach**: Define state as interface, use type parameters for methods

```go
type State interface {
    GetCount() int
    SetCount(int)
}

g := graph.NewStateGraphTyped[State]()
```

**Pros**:
- Interface flexibility
- Type safety

**Cons**:
- Verbose interface definitions
- Less ergonomic than structs
- Limited value access patterns

**Decision**: Rejected - too restrictive

### Alternative 4: Macro/Template System

**Approach**: Use text templates or macros to generate type-specific code

**Pros**:
- Full control over generated code
- Can optimize for specific types

**Cons**:
- Not idiomatic Go
- Complex build process
- Debugging difficulties

**Decision**: Rejected - not the Go way

### Alternative 5: Dual Implementation (Chosen)

**Approach**: Maintain both generic and non-generic versions

**Pros**:
- Backward compatible
- Gradual migration
- Best of both worlds
- Idiomatic Go

**Cons**:
- Code duplication
- Maintenance overhead

**Decision**: **Accepted** - Best balance of safety and flexibility

## Decision Points

### 1. Should we deprecate non-generic API?

**Decision**: **No**

**Rationale**:
- Many use cases benefit from dynamic types
- MapSchema works better with non-generic
- Breaking change would harm ecosystem

### 2. Should StateSchema be generic?

**Decision**: **Yes, with parallel non-generic version**

**Rationale**:
- Schema needs to match state type
- Generic schema enables type-safe merging
- Keep non-generic for MapSchema

### 3. Should we support mixed generic/non-generic graphs?

**Decision**: **No direct interop, but allow adapters**

**Rationale**:
- Type safety would be compromised
- Can provide conversion helpers if needed

### 4. How to handle subgraphs?

**Decision**: **Defer to future RFC**

**Rationale**:
- Subgraphs require careful type design
- May need variance or type bounds
- Better to get core right first

### 5. Should we add generic MapSchema?

**Decision**: **Optional future enhancement**

**Rationale**:
- map[string]T is too restrictive
- map[string]any works with StateGraph[map[string]any]
- Field-level generics would require complex type system

### 6. Implementation timeline?

**Proposed Phases**:
1. **Phase 1 (MVP)**: Core generic StateGraphTyped[S] and StateRunnableTyped[S]
2. **Phase 2**: Generic StateSchema[S] and StructSchema[S]
3. **Phase 3**: Documentation and examples
4. **Phase 4**: Community feedback and iteration
5. **Phase 5**: Advanced features (subgraphs, etc.)

## References

### Related RFCs

- [RFC: Channels Architecture](./RFC_CHANNELS.md)

### Go Generics Resources

- [Go Generics Proposal](https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md)
- [Go Generics Tutorial](https://go.dev/doc/tutorial/generics)

### Python LangGraph Comparison

Python LangGraph uses runtime typing with TypedDict:
```python
class State(TypedDict):
    count: int
    logs: list[str]
```

Go's compile-time generics provide stronger guarantees than Python's runtime annotations.

### Similar Patterns in Go Ecosystem

- **Generic Channels**: Standard library channels are generic (`chan T`)
- **Generic Collections**: Various third-party libraries
- **Generic Option Types**: Functional programming libraries

---

## Conclusion

This RFC proposes a **pragmatic approach** to adding type safety to LangGraphGo through generics, while maintaining the flexibility that makes the framework useful.

The **dual API strategy** allows:
- ✅ New projects to benefit from type safety immediately
- ✅ Existing projects to continue working without changes
- ✅ Gradual migration at project's own pace
- ✅ Use cases requiring dynamic types to remain supported

### Next Steps

1. **Community Review**: Gather feedback on this design
2. **Prototype Implementation**: Build MVP of generic StateGraph
3. **Example Migration**: Convert one showcase to demonstrate migration path
4. **Documentation**: Create migration guide
5. **Release**: Ship as experimental feature, iterate based on feedback

### Open Questions

1. Should we provide type-safe checkpointing for generic graphs?
2. How should streaming work with generic types?
3. Should we add helpers for common state patterns (e.g., message lists)?
4. What's the best way to handle state serialization for generic types?

**Feedback Welcome**: Please share your thoughts on this RFC in the GitHub issues or discussions.
