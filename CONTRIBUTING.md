# Contributing to LangGraphGo

Thank you for your interest in contributing to LangGraphGo! We welcome contributions from the community and are grateful for your support in making this project better.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Pull Request Process](#pull-request-process)
- [Coding Guidelines](#coding-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)
- [Community](#community)

## Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Please be respectful and professional in all interactions.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/langgraphgo.git
   cd langgraphgo
   ```
3. **Add the upstream repository**:
   ```bash
   git remote add upstream https://github.com/smallnest/langgraphgo.git
   ```

## Development Setup

### Prerequisites

- Go 1.25.0 or higher
- Git
- A GitHub account

### Install Dependencies

```bash
go mod download
```

### Verify Your Setup

Run the tests to ensure everything is working:

```bash
go test ./... -v
```

## How to Contribute

### Reporting Bugs

Before creating a bug report:
- Check the [existing issues](https://github.com/smallnest/langgraphgo/issues) to avoid duplicates
- Gather relevant information about your environment

When filing a bug report, include:
- A clear, descriptive title
- Steps to reproduce the issue
- Expected vs. actual behavior
- Go version and operating system
- Relevant code snippets or error messages
- Any additional context

### Suggesting Features

We welcome new feature suggestions! Before creating a feature request:
- Check existing issues to see if it has been discussed
- Consider if it aligns with the project's goal of feature parity with Python LangGraph

When suggesting a feature:
- Create an issue with a clear title
- Describe the feature and its use case
- Explain why it would be valuable
- Provide examples if possible

**Important**: Please create a feature issue first before starting work on a PR.

### Contributing Code

1. **Create an issue first** to discuss your proposed changes
2. **Wait for approval** from maintainers before starting work
3. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```
4. **Make your changes** following our coding guidelines
5. **Write tests** for your changes
6. **Update documentation** if needed
7. **Commit your changes** with clear, descriptive messages
8. **Push to your fork** and create a pull request

## Pull Request Process

### Before Submitting

- [ ] Run tests: `go test ./... -v`
- [ ] Run `go mod tidy` to clean up dependencies
- [ ] Format your code: `go fmt ./...`
- [ ] Lint your code (recommended: `golangci-lint run`)
- [ ] Update documentation if needed
- [ ] Add examples if introducing new features
- [ ] Ensure your commits follow our commit message guidelines

### Commit Message Guidelines

Write clear, concise commit messages:

```
<type>: <subject>

<body (optional)>

<footer (optional)>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

Examples:
```
feat: add support for ephemeral channels

fix: resolve race condition in parallel execution

docs: update README with new streaming examples
```

### PR Checklist

Your pull request should:
- [ ] Reference the related issue (e.g., "Fixes #123")
- [ ] Have a clear title and description
- [ ] Include tests for new functionality
- [ ] Pass all CI checks
- [ ] Be based on the latest `main` branch
- [ ] Have minimal, focused changes (avoid mixing multiple features/fixes)
- [ ] Include updated documentation

### Review Process

1. Maintainers will review your PR as soon as possible
2. Address any feedback or requested changes
3. Once approved, a maintainer will merge your PR
4. Your contribution will be included in the next release

## Coding Guidelines

### Go Style

- Follow the [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Follow common Go idioms and best practices
- Keep functions small and focused
- Use meaningful variable and function names

### Code Organization

- Place new features in appropriate packages
- Keep related code together
- Maintain consistency with existing code structure
- Add comments for exported functions and types

### Error Handling

- Always handle errors explicitly
- Use descriptive error messages
- Wrap errors with context when appropriate
- Avoid panic except in truly exceptional cases

### Examples

Good:
```go
func ProcessState(ctx context.Context, state State) (State, error) {
    if err := validateState(state); err != nil {
        return State{}, fmt.Errorf("invalid state: %w", err)
    }
    // ... processing logic
    return newState, nil
}
```

## Testing

### Writing Tests

- Write tests for all new functionality
- Maintain or improve code coverage
- Use table-driven tests where appropriate
- Test edge cases and error conditions
- Use meaningful test names that describe what is being tested

Example:
```go
func TestGraphExecution(t *testing.T) {
    tests := []struct {
        name    string
        input   any
        want    any
        wantErr bool
    }{
        {
            name:    "successful execution",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test ./graph -v

# Run with race detector
go test ./... -race
```

## Documentation

### Code Documentation

- Document all exported functions, types, and constants
- Use godoc format for comments
- Provide usage examples in comments where helpful
- Keep comments up-to-date with code changes

Example:
```go
// CreateReactAgent creates a ReAct (Reasoning + Acting) agent that can use tools
// to accomplish tasks. The agent follows a thought-action-observation loop.
//
// Parameters:
//   - model: The language model to use for reasoning
//   - tools: Available tools the agent can use
//
// Returns an error if the model or tools are invalid.
func CreateReactAgent(model llms.Model, tools []Tool) (*Graph, error) {
    // implementation
}
```

### User Documentation

- Update README.md when adding significant features
- Add examples to the `/examples` directory for new features
- Include clear comments in example code
- Update the website documentation if applicable

## Community

### Getting Help

- Check the [documentation](http://lango.rpcx.io)
- Search [existing issues](https://github.com/smallnest/langgraphgo/issues)
- Create a new issue with your question

### Staying Updated

- Watch the repository for updates
- Review the changelog for new releases
- Follow discussions in issues and PRs

## Recognition

All contributors will be recognized in our release notes and documentation. Thank you for helping make LangGraphGo better!

## Questions?

If you have any questions about contributing, feel free to:
- Open an issue for discussion
- Check existing documentation
- Review similar past contributions

We appreciate your contributions and look forward to collaborating with you!
