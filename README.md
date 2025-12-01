# 🦜️🔗 LangGraphGo

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/smallnest/langgraphgo)

[English](./README.md) | [简体中文](./README_CN.md)

> 🔀 **Forked from [paulnegz/langgraphgo](https://github.com/paulnegz/langgraphgo)** - Enhanced with streaming, visualization, observability, and production-ready features.
>
> This fork aims for **feature parity with the Python LangGraph library**, adding support for parallel execution, persistence, advanced state management, pre-built agents, and human-in-the-loop workflows.

## 📦 Installation

```bash
go get github.com/smallnest/langgraphgo
```

## 🚀 Features

- **Core Runtime**:
    - **Parallel Execution**: Concurrent node execution (fan-out) with thread-safe state merging.
    - **Runtime Configuration**: Propagate callbacks, tags, and metadata via `RunnableConfig`.
    - **LangChain Compatible**: Works seamlessly with `langchaingo`.

- **Persistence & Reliability**:
    - **Checkpointers**: Redis, Postgres, and SQLite implementations for durable state.
    - **State Recovery**: Pause and resume execution from checkpoints.

- **Advanced Capabilities**:
    - **State Schema**: Granular state updates with custom reducers (e.g., `AppendReducer`).
    - **Enhanced Streaming**: Real-time event streaming with granular `StreamEvent` types.
    - **Pre-built Agents**: Ready-to-use `ReAct` and `Supervisor` agent factories.

- **Developer Experience**:
    - **Visualization**: Export graphs to Mermaid, DOT, and ASCII with conditional edge support.
    - **Human-in-the-loop (HITL)**: Interrupt execution (`InterruptBefore`/`After`) and resume with updated state (`Command`).
    - **Observability**: Built-in tracing and metrics support.
    - **Tools**: Integrated `Tavily` and `Exa` search tools.

## 🎯 Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()
	model, _ := openai.New()

	// 1. Create Graph
	g := graph.NewMessageGraph()

	// 2. Add Nodes
	g.AddNode("generate", func(ctx context.Context, state interface{}) (interface{}, error) {
		messages := state.([]llms.MessageContent)
		response, _ := model.GenerateContent(ctx, messages)
		return append(messages, llms.TextParts("ai", response.Choices[0].Content)), nil
	})

	// 3. Define Edges
	g.AddEdge("generate", graph.END)
	g.SetEntryPoint("generate")

	// 4. Compile
	runnable, _ := g.Compile()

	// 5. Invoke
	initialState := []llms.MessageContent{
		llms.TextParts("human", "Hello, LangGraphGo!"),
	}
	result, _ := runnable.Invoke(ctx, initialState)
	
	fmt.Println(result)
}
```

## 📚 Examples

- **[Basic LLM](./examples/basic_llm/)** - Simple LangChain integration
- **[RAG Pipeline](./examples/rag_pipeline/)** - Complete retrieval-augmented generation
- **[RAG with LangChain](./examples/rag_with_langchain/)** - LangChain components integration
- **[RAG with VectorStores](./examples/rag_langchain_vectorstore_example/)** - LangChain VectorStore integration (New!)
- **[RAG with Chroma](./examples/rag_chroma_example/)** - Chroma vector database integration (New!)
- **[Tavily Search](./examples/tool_tavily/)** - Tavily search tool integration (New!)
- **[Exa Search](./examples/tool_exa/)** - Exa search tool integration (New!)
- **[Streaming](./examples/streaming_pipeline/)** - Real-time progress updates
- **[Conditional Routing](./examples/conditional_routing/)** - Dynamic path selection
- **[Checkpointing](./examples/checkpointing/)** - Save and resume state
- **[Visualization](./examples/visualization/)** - Export graph diagrams
- **[Listeners](./examples/listeners/)** - Progress, metrics, and logging
- **[Subgraphs](./examples/subgraph/)** - Nested graph composition
- **[Swarm](./examples/swarm/)** - Multi-agent collaboration
- **[State Schema](./examples/state_schema/)** - Complex state management with Reducers
- **[Time Travel](./examples/time_travel/)** - Inspect and modify state history
- **[Dynamic Interrupt](./examples/dynamic_interrupt/)** - Pause execution from within a node
- **[Durable Execution](./examples/durable_execution/)** - Crash recovery and resuming execution

## 🔧 Key Concepts

### Parallel Execution
LangGraphGo automatically executes nodes in parallel when they share the same starting node. Results are merged using the graph's state merger or schema.

```go
g.AddEdge("start", "branch_a")
g.AddEdge("start", "branch_b")
// branch_a and branch_b run concurrently
```

### Human-in-the-loop (HITL)
Pause execution to allow for human approval or input.

```go
config := &graph.Config{
    InterruptBefore: []string{"human_review"},
}

// Execution stops before "human_review" node
state, err := runnable.InvokeWithConfig(ctx, input, config)

// Resume execution
resumeConfig := &graph.Config{
    ResumeFrom: []string{"human_review"},
}
runnable.InvokeWithConfig(ctx, state, resumeConfig)
```

### Pre-built Agents
Quickly create complex agents using factory functions.

```go
// Create a ReAct agent
agent, err := prebuilt.CreateReactAgent(model, tools)

// Create a Supervisor agent
supervisor, err := prebuilt.CreateSupervisor(model, agents)
```

## 🎨 Graph Visualization

```go
exporter := runnable.GetGraph()
fmt.Println(exporter.DrawMermaid()) // Generates Mermaid flowchart
```

## 📈 Performance

- **Graph Operations**: ~14-94μs depending on format
- **Tracing Overhead**: ~4μs per execution
- **Event Processing**: 1000+ events/second
- **Streaming Latency**: <100ms

## 🧪 Testing

```bash
go test ./... -v
```

## 🤝 Contributing

This project is open for contributions! Please check `TASKS.md` for the roadmap and `TODOs.md` for specific items.

## 📄 License

MIT License - see original repository for details.