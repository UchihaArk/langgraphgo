# Agent Guidelines for LangGraphGo

## Build/Lint/Test Commands
- `make build` - Build all packages
- `make test` - Run all tests; for single test: `go test -v ./path/to/pkg -run TestName`
- `make test-race` - Run with race detector
- `make lint` - Run golangci-lint (also run `make fmt`, `make vet` before committing)

## Code Style
- Use `make fmt` to format code with gofmt before committing
- Exported types/functions: PascalCase; private: camelCase
- Define errors as package-level variables: `var (ErrX = errors.New("..."))`
- Write godoc comments for all exported identifiers
- Tests: Use table-driven tests with `t.Run()`, `t.Parallel()` where appropriate
- Always check errors (linted by golangci-lint)
- Run `make check` (fmt-check, vet, lint) and `make test` before committing

## Package Overview
- graph: Core graph construction and execution engine with state management
- memory: Various memory strategies for conversational AI applications
- prebuilt: Ready-to-use agent implementations (ReAct, Supervisor, Planning, etc.)
- ptc: Programmatic tool calling using generated code (Python, JavaScript, Shell)
- rag: Retrieval-Augmented Generation with vector and GraphRAG support
- store: Checkpoint storage backends (SQLite, PostgreSQL, Redis)
- tool: Collection of tools for web search, file ops, code execution
- log: Simple leveled logging interface for applications
- adapter: Integration adapters for GoSkills, MCP, and external systems
