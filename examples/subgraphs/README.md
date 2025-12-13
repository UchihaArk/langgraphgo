# Subgraphs (Composition) Example

## Background

As Agent systems grow in complexity, managing a single monolithic graph becomes unwieldy. **Subgraphs** allow developers to encapsulate logic into smaller, reusable graphs and compose them into larger systems. This is analogous to breaking a large program into functions or microservices. A subgraph appears to the parent graph as just another node, but internally it manages its own state and execution flow.

## Features

*   **Encapsulation**: Hide the complexity of a sub-task (e.g., "Research Topic") behind a single node.
*   **Reusability**: Define a graph once and use it in multiple places or projects.
*   **State Mapping**: Automatically passes the parent's state to the child and merges the child's result back (assuming compatible schemas).

## Implementation Principle

The `AddSubgraph` method in `graph/subgraph.go` wraps a `StateGraph` (or `MessageGraph` for backward compatibility) into a `Subgraph` struct.
1.  The subgraph is compiled into a `Runnable`.
2.  A wrapper function is created that matches the `Node` signature (`func(ctx, state) (any, error)`).
3.  When executed, this wrapper invokes the subgraph's `Runnable.Invoke`.
4.  The result is returned to the parent graph as the node's output.

## Code Walkthrough

In `main.go`:

1.  **Child Graph**:
    Defined as a simple graph that adds a "child_trace" to the state.

2.  **Parent Graph**:
    Defined as a graph that adds "parent_trace".

3.  **Composition**:
    ```go
    parent.AddSubgraph("nested_graph", child)
    ```
    The child graph is registered as a node named "nested_graph".

4.  **Wiring**:
    `start -> nested_graph -> end`.
    The parent graph treats "nested_graph" like any other node.

## How to Run

```bash
go run main.go
```
