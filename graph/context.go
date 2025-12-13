package graph

import "context"

type resumeValueKey struct{}

// WithResumeValue adds a resume value to the context.
// This value will be returned by Interrupt() when re-executing a node.
func WithResumeValue(ctx context.Context, value any) context.Context {
	return context.WithValue(ctx, resumeValueKey{}, value)
}

// GetResumeValue retrieves the resume value from the context.
func GetResumeValue(ctx context.Context) any {
	return ctx.Value(resumeValueKey{})
}
