package graph

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig configures retry behavior for nodes
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors func(error) bool // Determines if an error should trigger retry
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: func(_ error) bool {
			// By default, retry all errors
			return true
		},
	}
}

// RetryNode wraps a node with retry logic
type RetryNode struct {
	node   Node
	config *RetryConfig
}

// NewRetryNode creates a new retry node
func NewRetryNode(node Node, config *RetryConfig) *RetryNode {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryNode{
		node:   node,
		config: config,
	}
}

// Execute runs the node with retry logic
func (rn *RetryNode) Execute(ctx context.Context, state any) (any, error) {
	var lastErr error
	delay := rn.config.InitialDelay

	for attempt := 1; attempt <= rn.config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		// Execute the node
		result, err := rn.node.Function(ctx, state)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if rn.config.RetryableErrors != nil && !rn.config.RetryableErrors(err) {
			return nil, fmt.Errorf("non-retryable error in %s: %w", rn.node.Name, err)
		}

		// Don't sleep after the last attempt
		if attempt < rn.config.MaxAttempts {
			// Sleep with exponential backoff
			select {
			case <-time.After(delay):
				// Calculate next delay with backoff
				delay = time.Duration(float64(delay) * rn.config.BackoffFactor)
				if delay > rn.config.MaxDelay {
					delay = rn.config.MaxDelay
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
			}
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded for %s: %w",
		rn.config.MaxAttempts, rn.node.Name, lastErr)
}

// AddNodeWithRetry adds a node with retry logic
func (g *StateGraph) AddNodeWithRetry(
	name string,
	description string,
	fn func(context.Context, any) (any, error),
	config *RetryConfig,
) {
	node := Node{
		Name:        name,
		Description: description,
		Function:    fn,
	}
	retryNode := NewRetryNode(node, config)
	g.AddNode(name, description, retryNode.Execute)
}

// TimeoutNode wraps a node with timeout logic
type TimeoutNode struct {
	node    Node
	timeout time.Duration
}

// NewTimeoutNode creates a new timeout node
func NewTimeoutNode(node Node, timeout time.Duration) *TimeoutNode {
	return &TimeoutNode{
		node:    node,
		timeout: timeout,
	}
}

// Execute runs the node with timeout
func (tn *TimeoutNode) Execute(ctx context.Context, state any) (any, error) {
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, tn.timeout)
	defer cancel()

	// Channel for result
	type result struct {
		value any
		err   error
	}
	resultChan := make(chan result, 1)

	// Execute in goroutine
	go func() {
		value, err := tn.node.Function(timeoutCtx, state)
		resultChan <- result{value: value, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		return res.value, res.err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("node %s timed out after %v", tn.node.Name, tn.timeout)
	}
}

// AddNodeWithTimeout adds a node with timeout
func (g *StateGraph) AddNodeWithTimeout(
	name string,
	description string,
	fn func(context.Context, any) (any, error),
	timeout time.Duration,
) {
	node := Node{
		Name:        name,
		Description: description,
		Function:    fn,
	}
	timeoutNode := NewTimeoutNode(node, timeout)
	g.AddNode(name, description, timeoutNode.Execute)
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold int           // Number of failures before opening
	SuccessThreshold int           // Number of successes before closing
	Timeout          time.Duration // Time before attempting to close
	HalfOpenMaxCalls int           // Max calls in half-open state
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	node            Node
	config          CircuitBreakerConfig
	state           CircuitBreakerState
	failures        int
	successes       int
	lastFailureTime time.Time
	halfOpenCalls   int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(node Node, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		node:   node,
		config: config,
		state:  CircuitClosed,
	}
}

// Execute runs the node with circuit breaker logic
func (cb *CircuitBreaker) Execute(ctx context.Context, state any) (any, error) {
	// Check circuit state
	switch cb.state {
	case CircuitClosed:
		// Circuit is closed, proceed normally
	case CircuitOpen:
		// Check if enough time has passed to try again
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCalls = 0
		} else {
			return nil, fmt.Errorf("circuit breaker open for %s", cb.node.Name)
		}
	case CircuitHalfOpen:
		// Check if we've made too many calls in half-open state
		if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			cb.state = CircuitOpen
			return nil, fmt.Errorf("circuit breaker half-open limit reached for %s", cb.node.Name)
		}
		cb.halfOpenCalls++
	}

	// Execute the node
	result, err := cb.node.Function(ctx, state)

	// Update circuit breaker state based on result
	if err != nil {
		cb.failures++
		cb.successes = 0
		cb.lastFailureTime = time.Now()

		if cb.failures >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
		}

		return nil, fmt.Errorf("circuit breaker error in %s: %w", cb.node.Name, err)
	}

	// Success
	cb.successes++
	cb.failures = 0

	if cb.state == CircuitHalfOpen && cb.successes >= cb.config.SuccessThreshold {
		cb.state = CircuitClosed
	}

	return result, nil
}

// AddNodeWithCircuitBreaker adds a node with circuit breaker
func (g *StateGraph) AddNodeWithCircuitBreaker(
	name string,
	description string,
	fn func(context.Context, any) (any, error),
	config CircuitBreakerConfig,
) {
	node := Node{
		Name:        name,
		Description: description,
		Function:    fn,
	}
	cb := NewCircuitBreaker(node, config)
	g.AddNode(name, description, cb.Execute)
}

// RateLimiter implements rate limiting for nodes
type RateLimiter struct {
	node     Node
	maxCalls int
	window   time.Duration
	calls    []time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(node Node, maxCalls int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		node:     node,
		maxCalls: maxCalls,
		window:   window,
		calls:    make([]time.Time, 0, maxCalls),
	}
}

// Execute runs the node with rate limiting
func (rl *RateLimiter) Execute(ctx context.Context, state any) (any, error) {
	now := time.Now()

	// Remove old calls outside the window
	validCalls := make([]time.Time, 0, rl.maxCalls)
	for _, callTime := range rl.calls {
		if now.Sub(callTime) < rl.window {
			validCalls = append(validCalls, callTime)
		}
	}
	rl.calls = validCalls

	// Check if we're at the limit
	if len(rl.calls) >= rl.maxCalls {
		// Calculate when we can make the next call
		oldestCall := rl.calls[0]
		waitTime := rl.window - now.Sub(oldestCall)
		return nil, fmt.Errorf("rate limit exceeded for %s, retry after %v", rl.node.Name, waitTime)
	}

	// Record this call
	rl.calls = append(rl.calls, now)

	// Execute the node
	return rl.node.Function(ctx, state)
}

// AddNodeWithRateLimit adds a node with rate limiting
func (g *StateGraph) AddNodeWithRateLimit(
	name string,
	description string,
	fn func(context.Context, any) (any, error),
	maxCalls int,
	window time.Duration,
) {
	node := Node{
		Name:        name,
		Description: description,
		Function:    fn,
	}
	rl := NewRateLimiter(node, maxCalls, window)
	g.AddNode(name, description, rl.Execute)
}

// ExponentialBackoffRetry implements exponential backoff with jitter
func ExponentialBackoffRetry(
	ctx context.Context,
	fn func() (any, error),
	maxAttempts int,
	baseDelay time.Duration,
) (any, error) {
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		if attempt == maxAttempts-1 {
			return nil, err
		}

		// Calculate delay with exponential backoff and jitter
		delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))

		// Add jitter (±25%)
		//nolint:gosec // Using weak RNG for jitter is acceptable, not security-critical
		jitter := time.Duration(float64(delay) * 0.25 * (2*rand.Float64() - 1))
		delay += jitter

		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("max attempts reached")
}
