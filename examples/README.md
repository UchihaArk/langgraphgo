# LangGraphGo Examples

This directory contains various examples demonstrating the features of LangGraphGo.

## Basic Concepts
- **[Basic Example](basic_example/)**: Simple graph with hardcoded steps.
- **[Basic LLM](basic_llm/)**: Integration with LLMs.
- **[Conditional Routing](conditional_routing/)**: Dynamic routing based on state.
- **[Conditional Edges](conditional_edges_example/)**: Using conditional edges.

## Advanced Features
- **[Parallel Execution](parallel_execution/)**: Fan-out/Fan-in execution with state merging.
- **[Configuration](configuration/)**: Using runtime configuration to pass metadata and settings.
- **[Custom Reducer](custom_reducer/)**: Defining custom state reducers for complex merge logic.
- **[State Schema](state_schema/)**: Managing complex state updates with Schema and Reducers.
- **[Subgraphs](subgraph/)**: Composing graphs within graphs.
- **[Streaming](streaming_pipeline/)**: Streaming execution events.
- **[Listeners](listeners/)**: Attaching event listeners to the graph.

## Persistence (Checkpointing)
- **[Memory](checkpointing/main.go)**: In-memory checkpointing.
- **[PostgreSQL](checkpointing/postgres/)**: Persistent state using PostgreSQL.
- **[SQLite](checkpointing/sqlite/)**: Persistent state using SQLite.
- **[Redis](checkpointing/redis/)**: Persistent state using Redis.
- **[Durable Execution](durable_execution/)**: Crash recovery and resuming execution from checkpoints.

## Human-in-the-loop
- **[Human Approval](human_in_the_loop/)**: Workflow with interrupts and human approval steps.
- **[Time Travel](time_travel/)**: Inspecting and modifying state history (GetState/UpdateState).
- **[Dynamic Interrupt](dynamic_interrupt/)**: Pausing execution from within a node using `graph.Interrupt`.

## Pre-built Agents
- **[ReAct Agent](react_agent/)**: Reason and Action agent using tools.
- **[Supervisor](supervisor/)**: Multi-agent orchestration using a supervisor.
- **[Swarm](swarm/)**: Multi-agent collaboration using handoffs.

## Memory
- **[Memory Basic](memory_basic/)**: Basic usage of LangChain memory.
- **[Memory Chatbot](memory_chatbot/)**: Chatbot with memory integration.

## RAG (Retrieval Augmented Generation)
- **[RAG Basic](rag_basic/)**: Basic RAG implementation.
- **[RAG Pipeline](rag_pipeline/)**: Complete RAG pipeline.
- **[RAG Advanced](rag_advanced/)**: Advanced RAG techniques.
- **[RAG Conditional](rag_conditional/)**: Conditional RAG workflow.
- **[RAG with Embeddings](rag_with_embeddings/)**: RAG using embeddings.
- **[RAG with LangChain](rag_with_langchain/)**: RAG using LangChain components.
- **[RAG with VectorStores](rag_langchain_vectorstore_example/)**: RAG using LangChain VectorStores.
- **[RAG with Chroma](rag_chroma_example/)**: RAG using Chroma database.

## Other
- **[Visualization](visualization/)**: Generating Mermaid diagrams for graphs.
- **[LangChain Integration](langchain_example/)**: Using LangChain tools and models.
- **[Tavily Search](tool_tavily/)**: Using Tavily search tool with ReAct agent.
- **[Exa Search](tool_exa/)**: Using Exa search tool with ReAct agent.
