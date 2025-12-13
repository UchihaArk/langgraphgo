package graph

import (
	"context"
	"time"
)

// CallbackHandler defines the interface for handling graph execution callbacks
// This matches Python's LangChain callback pattern
type CallbackHandler interface {
	// Chain callbacks (for graph/workflow execution)
	OnChainStart(ctx context.Context, serialized map[string]any, inputs map[string]any, runID string, parentRunID *string, tags []string, metadata map[string]any)
	OnChainEnd(ctx context.Context, outputs map[string]any, runID string)
	OnChainError(ctx context.Context, err error, runID string)

	// LLM callbacks (for AI model calls)
	OnLLMStart(ctx context.Context, serialized map[string]any, prompts []string, runID string, parentRunID *string, tags []string, metadata map[string]any)
	OnLLMEnd(ctx context.Context, response any, runID string)
	OnLLMError(ctx context.Context, err error, runID string)

	// Tool callbacks (for tool/function calls)
	OnToolStart(ctx context.Context, serialized map[string]any, inputStr string, runID string, parentRunID *string, tags []string, metadata map[string]any)
	OnToolEnd(ctx context.Context, output string, runID string)
	OnToolError(ctx context.Context, err error, runID string)

	// Retriever callbacks (for data retrieval operations)
	OnRetrieverStart(ctx context.Context, serialized map[string]any, query string, runID string, parentRunID *string, tags []string, metadata map[string]any)
	OnRetrieverEnd(ctx context.Context, documents []any, runID string)
	OnRetrieverError(ctx context.Context, err error, runID string)
}

// GraphCallbackHandler extends CallbackHandler with graph-specific events
type GraphCallbackHandler interface {
	CallbackHandler
	// OnGraphStep is called after a step (node execution + state update) is completed
	OnGraphStep(ctx context.Context, stepNode string, state any)
}

// Config represents configuration for graph invocation
// This matches Python's config dict pattern
type Config struct {
	// Callbacks to be invoked during execution
	Callbacks []CallbackHandler `json:"callbacks"`

	// Metadata to attach to the execution
	Metadata map[string]any `json:"metadata"`

	// Tags to categorize the execution
	Tags []string `json:"tags"`

	// Configurable parameters for the execution
	Configurable map[string]any `json:"configurable"`

	// RunName for this execution
	RunName string `json:"run_name"`

	// Timeout for the execution
	Timeout *time.Duration `json:"timeout"`

	// InterruptBefore nodes to stop before execution
	InterruptBefore []string `json:"interrupt_before"`

	// InterruptAfter nodes to stop after execution
	InterruptAfter []string `json:"interrupt_after"`

	// ResumeFrom nodes to start execution from (bypassing entry point)
	ResumeFrom []string `json:"resume_from"`

	// ResumeValue provides the value to return from an Interrupt() call when resuming
	ResumeValue any `json:"resume_value"`
}

// NoOpCallbackHandler provides a no-op implementation of CallbackHandler
type NoOpCallbackHandler struct{}

func (n *NoOpCallbackHandler) OnChainStart(ctx context.Context, serialized map[string]any, inputs map[string]any, runID string, parentRunID *string, tags []string, metadata map[string]any) {
}
func (n *NoOpCallbackHandler) OnChainEnd(ctx context.Context, outputs map[string]any, runID string) {
}
func (n *NoOpCallbackHandler) OnChainError(ctx context.Context, err error, runID string) {}
func (n *NoOpCallbackHandler) OnLLMStart(ctx context.Context, serialized map[string]any, prompts []string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
}
func (n *NoOpCallbackHandler) OnLLMEnd(ctx context.Context, response any, runID string) {}
func (n *NoOpCallbackHandler) OnLLMError(ctx context.Context, err error, runID string)  {}
func (n *NoOpCallbackHandler) OnToolStart(ctx context.Context, serialized map[string]any, inputStr string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
}
func (n *NoOpCallbackHandler) OnToolEnd(ctx context.Context, output string, runID string) {}
func (n *NoOpCallbackHandler) OnToolError(ctx context.Context, err error, runID string)   {}
func (n *NoOpCallbackHandler) OnRetrieverStart(ctx context.Context, serialized map[string]any, query string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
}
func (n *NoOpCallbackHandler) OnRetrieverEnd(ctx context.Context, documents []any, runID string) {
}
func (n *NoOpCallbackHandler) OnRetrieverError(ctx context.Context, err error, runID string) {}
