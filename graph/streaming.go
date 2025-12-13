package graph

import (
	"context"
	"sync"
	"time"
)

// StreamMode defines the mode of streaming
type StreamMode string

const (
	// StreamModeValues emits the full state after each step
	StreamModeValues StreamMode = "values"
	// StreamModeUpdates emits the updates (deltas) from each node
	StreamModeUpdates StreamMode = "updates"
	// StreamModeMessages emits LLM messages/tokens (if available)
	StreamModeMessages StreamMode = "messages"
	// StreamModeDebug emits all events (default)
	StreamModeDebug StreamMode = "debug"
)

// StreamConfig configures streaming behavior
type StreamConfig struct {
	// BufferSize is the size of the event channel buffer
	BufferSize int

	// EnableBackpressure determines if backpressure handling is enabled
	EnableBackpressure bool

	// MaxDroppedEvents is the maximum number of events to drop before logging
	MaxDroppedEvents int

	// Mode specifies what kind of events to stream
	Mode StreamMode
}

// DefaultStreamConfig returns the default streaming configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		BufferSize:         1000,
		EnableBackpressure: true,
		MaxDroppedEvents:   100,
		Mode:               StreamModeDebug,
	}
}

// StreamResult contains the channels returned by streaming execution
type StreamResult struct {
	// Events channel receives StreamEvent objects in real-time
	Events <-chan StreamEvent

	// Result channel receives the final result when execution completes
	Result <-chan any

	// Errors channel receives any errors that occur during execution
	Errors <-chan error

	// Done channel is closed when streaming is complete
	Done <-chan struct{}

	// Cancel function can be called to stop streaming
	Cancel context.CancelFunc
}

// StreamingListener implements NodeListener for streaming events
type StreamingListener struct {
	eventChan chan<- StreamEvent
	config    StreamConfig
	mutex     sync.RWMutex

	droppedEvents int
	closed        bool
}

// NewStreamingListener creates a new streaming listener
func NewStreamingListener(eventChan chan<- StreamEvent, config StreamConfig) *StreamingListener {
	return &StreamingListener{
		eventChan: eventChan,
		config:    config,
	}
}

// emitEvent sends an event to the channel handling backpressure
func (sl *StreamingListener) emitEvent(event StreamEvent) {
	// Check if listener is closed
	sl.mutex.RLock()
	if sl.closed {
		sl.mutex.RUnlock()
		return
	}
	sl.mutex.RUnlock()

	// Filter based on Mode
	if !sl.shouldEmit(event) {
		return
	}

	// Try to send event without blocking
	select {
	case sl.eventChan <- event:
		// Event sent successfully
	default:
		// Channel is full
		if sl.config.EnableBackpressure {
			sl.handleBackpressure()
		}
		// Drop the event if backpressure handling is disabled or channel is still full
	}
}

func (sl *StreamingListener) shouldEmit(event StreamEvent) bool {
	switch sl.config.Mode {
	case StreamModeDebug:
		return true
	case StreamModeValues:
		// Only emit OnGraphStep events (which contain full state)
		// We use a custom event type for this?
		// Currently OnGraphStep calls emitEvent with what?
		// We need to implement OnGraphStep in StreamingListener.
		// For now, let's assume OnGraphStep emits a special event.
		// If event.Event == "graph_step", return true.
		return event.Event == "graph_step"
	case StreamModeUpdates:
		// Emit node outputs (ToolEnd, ChainEnd, NodeEventComplete)
		return event.Event == EventToolEnd || event.Event == EventChainEnd || event.Event == NodeEventComplete
	case StreamModeMessages:
		// Emit LLM events
		return event.Event == EventLLMEnd || event.Event == EventLLMStart
	default:
		return true
	}
}

// OnNodeEvent implements the NodeListener interface
func (sl *StreamingListener) OnNodeEvent(_ context.Context, event NodeEvent, nodeName string, state any, err error) {
	streamEvent := StreamEvent{
		Timestamp: time.Now(),
		NodeName:  nodeName,
		Event:     event,
		State:     state,
		Error:     err,
		Metadata:  make(map[string]any),
	}
	sl.emitEvent(streamEvent)
}

// CallbackHandler implementation

func (sl *StreamingListener) OnChainStart(ctx context.Context, serialized map[string]any, inputs map[string]any, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventChainStart,
		Metadata:  metadata,
		State:     inputs,
	})
}

func (sl *StreamingListener) OnChainEnd(ctx context.Context, outputs map[string]any, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventChainEnd,
		State:     outputs,
	})
}

func (sl *StreamingListener) OnChainError(ctx context.Context, err error, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     NodeEventError, // Or specific ChainError?
		Error:     err,
	})
}

func (sl *StreamingListener) OnLLMStart(ctx context.Context, serialized map[string]any, prompts []string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventLLMStart,
		Metadata:  metadata,
		State:     prompts,
	})
}

func (sl *StreamingListener) OnLLMEnd(ctx context.Context, response any, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventLLMEnd,
		State:     response,
	})
}

func (sl *StreamingListener) OnLLMError(ctx context.Context, err error, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     NodeEventError,
		Error:     err,
	})
}

func (sl *StreamingListener) OnToolStart(ctx context.Context, serialized map[string]any, inputStr string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventToolStart,
		Metadata:  metadata,
		State:     inputStr,
	})
}

func (sl *StreamingListener) OnToolEnd(ctx context.Context, output string, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventToolEnd,
		State:     output,
	})
}

func (sl *StreamingListener) OnToolError(ctx context.Context, err error, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     NodeEventError,
		Error:     err,
	})
}

func (sl *StreamingListener) OnRetrieverStart(ctx context.Context, serialized map[string]any, query string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	// Map to custom or tool event?
}

func (sl *StreamingListener) OnRetrieverEnd(ctx context.Context, documents []any, runID string) {
}

func (sl *StreamingListener) OnRetrieverError(ctx context.Context, err error, runID string) {
}

// OnGraphStep implements GraphCallbackHandler
func (sl *StreamingListener) OnGraphStep(ctx context.Context, stepNode string, state any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     "graph_step", // Custom event type
		NodeName:  stepNode,
		State:     state,
	})
}

// Close marks the listener as closed to prevent sending to closed channels
func (sl *StreamingListener) Close() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	sl.closed = true
}

// handleBackpressure manages channel backpressure
func (sl *StreamingListener) handleBackpressure() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	sl.droppedEvents++

	// Could implement more sophisticated backpressure strategies here
	// For now, we just track dropped events
}

// GetDroppedEventsCount returns the number of dropped events
func (sl *StreamingListener) GetDroppedEventsCount() int {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()
	return sl.droppedEvents
}

// StreamingRunnable wraps a ListenableRunnable with streaming capabilities
type StreamingRunnable struct {
	runnable *ListenableRunnable
	config   StreamConfig
}

// NewStreamingRunnable creates a new streaming runnable
func NewStreamingRunnable(runnable *ListenableRunnable, config StreamConfig) *StreamingRunnable {
	return &StreamingRunnable{
		runnable: runnable,
		config:   config,
	}
}

// NewStreamingRunnableWithDefaults creates a streaming runnable with default config
func NewStreamingRunnableWithDefaults(runnable *ListenableRunnable) *StreamingRunnable {
	return NewStreamingRunnable(runnable, DefaultStreamConfig())
}

// Stream executes the graph with real-time event streaming
func (sr *StreamingRunnable) Stream(ctx context.Context, initialState any) *StreamResult {
	// Create channels
	eventChan := make(chan StreamEvent, sr.config.BufferSize)
	resultChan := make(chan any, 1)
	errorChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Create cancellable context
	streamCtx, cancel := context.WithCancel(ctx)

	// Create streaming listener
	streamingListener := NewStreamingListener(eventChan, sr.config)

	// Add the streaming listener to all nodes
	for _, node := range sr.runnable.listenableNodes {
		node.AddListener(streamingListener)
	}

	// Execute in goroutine
	go func() {
		defer func() {
			// First, close the streaming listener to prevent new events
			streamingListener.Close()

			// Clean up: remove streaming listener from all nodes
			for _, node := range sr.runnable.listenableNodes {
				node.RemoveListener(streamingListener)
			}

			// Give a small delay for any in-flight listener calls to complete
			time.Sleep(10 * time.Millisecond)

			// Now safe to close channels
			close(eventChan)
			close(resultChan)
			close(errorChan)
			close(doneChan)
		}()

		// Create config with streaming listener as callback
		config := &Config{
			Callbacks: []CallbackHandler{streamingListener},
		}

		// Execute the runnable
		result, err := sr.runnable.InvokeWithConfig(streamCtx, initialState, config)

		// Send result or error
		if err != nil {
			select {
			case errorChan <- err:
			case <-streamCtx.Done():
			}
		} else {
			select {
			case resultChan <- result:
			case <-streamCtx.Done():
			}
		}
	}()

	return &StreamResult{
		Events: eventChan,
		Result: resultChan,
		Errors: errorChan,
		Done:   doneChan,
		Cancel: cancel,
	}
}

// StreamingStateGraph extends ListenableStateGraph with streaming capabilities
type StreamingStateGraph struct {
	*ListenableStateGraph
	config StreamConfig
}

// NewStreamingStateGraph creates a new streaming message graph
func NewStreamingStateGraph() *StreamingStateGraph {
	return &StreamingStateGraph{
		ListenableStateGraph: NewListenableStateGraph(),
		config:               DefaultStreamConfig(),
	}
}

// NewStreamingStateGraphWithConfig creates a streaming graph with custom config
func NewStreamingStateGraphWithConfig(config StreamConfig) *StreamingStateGraph {
	return &StreamingStateGraph{
		ListenableStateGraph: NewListenableStateGraph(),
		config:               config,
	}
}

// CompileStreaming compiles the graph into a streaming runnable
func (g *StreamingStateGraph) CompileStreaming() (*StreamingRunnable, error) {
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		return nil, err
	}

	return NewStreamingRunnable(listenableRunnable, g.config), nil
}

// SetStreamConfig updates the streaming configuration
func (g *StreamingStateGraph) SetStreamConfig(config StreamConfig) {
	g.config = config
}

// GetStreamConfig returns the current streaming configuration
func (g *StreamingStateGraph) GetStreamConfig() StreamConfig {
	return g.config
}

// StreamingExecutor provides a high-level interface for streaming execution
type StreamingExecutor struct {
	runnable *StreamingRunnable
}

// NewStreamingExecutor creates a new streaming executor
func NewStreamingExecutor(runnable *StreamingRunnable) *StreamingExecutor {
	return &StreamingExecutor{
		runnable: runnable,
	}
}

// ExecuteWithCallback executes the graph and calls the callback for each event
//
//nolint:cyclop // Complex streaming logic requires multiple conditional paths
func (se *StreamingExecutor) ExecuteWithCallback(
	ctx context.Context,
	initialState any,
	eventCallback func(event StreamEvent),
	resultCallback func(result any, err error),
) error {

	streamResult := se.runnable.Stream(ctx, initialState)
	defer streamResult.Cancel()

	var finalResult any
	var finalError error
	resultReceived := false

	for {
		select {
		case event, ok := <-streamResult.Events:
			if !ok {
				// Events channel closed
				if resultReceived && resultCallback != nil {
					resultCallback(finalResult, finalError)
				}
				return finalError
			}
			if eventCallback != nil {
				eventCallback(event)
			}

		case result := <-streamResult.Result:
			finalResult = result
			resultReceived = true
			// Don't return immediately, wait for events channel to close

		case err := <-streamResult.Errors:
			finalError = err
			resultReceived = true
			// Don't return immediately, wait for events channel to close

		case <-streamResult.Done:
			if resultReceived && resultCallback != nil {
				resultCallback(finalResult, finalError)
			}
			return finalError

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ExecuteAsync executes the graph asynchronously and returns immediately
func (se *StreamingExecutor) ExecuteAsync(ctx context.Context, initialState any) *StreamResult {
	return se.runnable.Stream(ctx, initialState)
}

// GetGraph returns a Exporter for the streaming runnable
func (sr *StreamingRunnable) GetGraph() *Exporter {
	return sr.runnable.GetGraph()
}
