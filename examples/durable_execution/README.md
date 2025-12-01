# Durable Execution Example

This example demonstrates **Durable Execution** in LangGraphGo. It shows how a long-running process can recover from a crash and resume execution from the last saved checkpoint.

## 1. Background

**Durable Execution** ensures that the state of your application is persisted at every step. If the process crashes (e.g., power failure, OOM, deployment), it can be restarted and will automatically pick up where it left off, avoiding data loss and redundant work.

## 2. Key Concepts

- **CheckpointStore**: A persistent storage backend (e.g., File, Postgres, Redis) that saves the graph state.
- **Crash Recovery**: The pattern of checking for existing checkpoints on startup and resuming execution instead of starting fresh.

## 3. How It Works

1.  **File Store**: We implement a simple JSON-based file store (`checkpoints.json`) to persist state.
2.  **Simulation**:
    - The graph has 3 steps.
    - **Step 2** is programmed to crash (exit) if the environment variable `CRASH=true` is set.
3.  **Recovery**:
    - On startup, the program checks `checkpoints.json` for the given `thread_id`.
    - If a checkpoint exists (e.g., from Step 1), it loads the state and determines the next step (Step 2).
    - It then resumes execution from that step.

## 4. Running the Example

**Step 1: Run and Crash**
```bash
export CRASH=true
go run main.go
```
*Output:*
```text
Starting new execution...
Executing Step 1...
Executing Step 2...
!!! CRASHING AT STEP 2 !!!
```

**Step 2: Recover and Finish**
```bash
unset CRASH
go run main.go
```
*Output:*
```text
Found existing checkpoint: ... (Node: step_1)
Resuming execution...
Continuing from step_2...
Executing Step 2...
Executing Step 3...
Final Result: ...
```

**Clean Up**
To start over, delete the `checkpoints.json` file.
```bash
rm checkpoints.json
```
