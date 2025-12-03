
---
name: hello_world
description: A fundamental skill that demonstrates the basic execution of a Python script. It serves as a "Hello, World!" example for the skill system, verifying that the environment is correctly set up and that the agent can execute scripts.
version: 1.0.0
license: MIT
---

## Overview

The `hello_world` skill is the simplest possible skill in the ecosystem. Its primary purpose is to validate the operational status of the skill runner and the agent's ability to invoke tools.

## Functionality

When invoked, this skill executes a Python script that prints a greeting message to the standard output. This confirms:
1.  **Script Execution**: The agent can successfully locate and run a Python script.
2.  **Output Capture**: The system can capture the standard output from the script and return it to the agent.
3.  **Tool Integration**: The skill is correctly registered and accessible as a tool.

## Usage

This skill is typically used in the following scenarios:
-   **System Health Check**: To verify that the agent and skill runner are functioning correctly.
-   **Onboarding**: As a first step for developers learning how to create and use skills.
-   **Debugging**: To isolate issues with script execution or tool invocation.

### Example Command

To use this skill, the agent must execute the following command:

```python
scripts/hello.py
```

## Implementation Details

The skill consists of a single Python script `hello.py` which performs a simple print operation. No external dependencies or complex logic are involved, ensuring that any failure is likely due to the environment or configuration rather than the skill itself.