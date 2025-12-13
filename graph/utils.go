package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// generateRunID generates a unique run ID for callbacks
func generateRunID() string {
	return uuid.New().String()
}

// convertStateToMap converts a state to a map for callbacks
func convertStateToMap(state any) map[string]any {
	// Try to convert to map directly
	if m, ok := state.(map[string]any); ok {
		return m
	}

	// Try to marshal/unmarshal through JSON
	data, err := json.Marshal(state)
	if err != nil {
		return map[string]any{
			"state": fmt.Sprintf("%v", state),
		}
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]any{
			"state": string(data),
		}
	}

	return result
}

// convertStateToString converts a state to a string for callbacks
func convertStateToString(state any) string {
	// Try to marshal to JSON
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Sprintf("%v", state)
	}
	return string(data)
}

type configKey struct{}

// WithConfig adds the config to the context
func WithConfig(ctx context.Context, config *Config) context.Context {
	return context.WithValue(ctx, configKey{}, config)
}

// GetConfig retrieves the config from the context
func GetConfig(ctx context.Context) *Config {
	if config, ok := ctx.Value(configKey{}).(*Config); ok {
		return config
	}
	return nil
}

// SafeGo runs a function in a goroutine with panic recovery.
// It uses a WaitGroup (if provided) and supports a custom panic handler.
func SafeGo(wg *sync.WaitGroup, fn func(), onPanic func(any)) {
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		defer func() {
			if wg != nil {
				wg.Done()
			}
			if r := recover(); r != nil {
				if onPanic != nil {
					onPanic(r)
				} else {
					fmt.Printf("panic recovered in SafeGo: %v\n", r)
				}
			}
		}()
		fn()
	}()
}
