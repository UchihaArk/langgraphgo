# LangGraphGo 示例

本目录包含演示 LangGraphGo 特性的各种示例。

## 基本概念
- **[基本示例 (Basic Example)](basic_example/)**: 带有硬编码步骤的简单图。
- **[基本 LLM (Basic LLM)](basic_llm/)**: 与 LLM 的集成。
- **[条件路由 (Conditional Routing)](conditional_routing/)**: 基于状态的动态路由。
- **[条件边 (Conditional Edges)](conditional_edges_example/)**: 使用条件边。

## 高级特性
- **[并行执行 (Parallel Execution)](parallel_execution/)**: 带有状态合并的扇出/扇入 (Fan-out/Fan-in) 执行。
- **[配置 (Configuration)](configuration/)**: 使用运行时配置传递元数据和设置。
- **[自定义归约器 (Custom Reducer)](custom_reducer/)**: 为复杂的合并逻辑定义自定义状态归约器。
- **[State Schema](state_schema/)**: 使用 Schema 和 Reducer 管理复杂的状态更新。
- **[子图 (Subgraphs)](subgraph/)**: 在图中组合图。
- **[流式处理 (Streaming)](streaming_pipeline/)**: 流式传输执行事件。
- **[监听器 (Listeners)](listeners/)**: 向图添加事件监听器。

## 持久化 (检查点 Checkpointing)
- **[内存 (Memory)](checkpointing/main.go)**: 内存检查点。
- **[PostgreSQL](checkpointing/postgres/)**: 使用 PostgreSQL 的持久化状态。
- **[SQLite](checkpointing/sqlite/)**: 使用 SQLite 的持久化状态。
- **[Redis](checkpointing/redis/)**: 使用 Redis 的持久化状态。
- **[持久化执行 (Durable Execution)](durable_execution/)**: 崩溃恢复和从检查点恢复执行。

## 人机交互 (Human-in-the-loop)
- **[人工审批 (Human Approval)](human_in_the_loop/)**: 包含中断和人工审批步骤的工作流。
- **[时间旅行 (Time Travel)](time_travel/)**: 检查和修改状态历史 (GetState/UpdateState)。
- **[动态中断 (Dynamic Interrupt)](dynamic_interrupt/)**: 使用 `graph.Interrupt` 在节点内部暂停执行。

## 预构建代理 (Pre-built Agents)
- **[ReAct Agent](react_agent/)**: 使用工具的推理与行动 (Reason and Action) 代理。
- **[Supervisor](supervisor/)**: 使用 Supervisor 进行多代理编排。
- **[Swarm](swarm/)**: 使用切换 (handoffs) 的多代理协作。

## Memory (记忆)
- **[Memory Basic](memory_basic/)**: LangChain Memory 的基本用法。
- **[Memory Chatbot](memory_chatbot/)**: 集成 Memory 的聊天机器人。

## RAG (检索增强生成)
- **[RAG Basic](rag_basic/)**: 基础 RAG 实现。
- **[RAG Pipeline](rag_pipeline/)**: 完整的 RAG 管道。
- **[RAG Advanced](rag_advanced/)**: 高级 RAG 技术。
- **[RAG Conditional](rag_conditional/)**: 条件 RAG 工作流。
- **[RAG with Embeddings](rag_with_embeddings/)**: 使用 Embeddings 的 RAG。
- **[RAG with LangChain](rag_with_langchain/)**: 使用 LangChain 组件的 RAG。
- **[RAG with VectorStores](rag_langchain_vectorstore_example/)**: 使用 LangChain VectorStores 的 RAG。
- **[RAG with Chroma](rag_chroma_example/)**: 使用 Chroma 数据库的 RAG。

## 其他
- **[可视化 (Visualization)](visualization/)**: 生成图的 Mermaid 图表。
- **[LangChain 集成 (LangChain Integration)](langchain_example/)**: 使用 LangChain 工具和模型。
- **[Tavily Search](tool_tavily/)**: 使用 Tavily 搜索工具和 ReAct Agent。
- **[Exa Search](tool_exa/)**: 使用 Exa 搜索工具和 ReAct Agent。
