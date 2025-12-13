package graph

// Command allows a node to dynamically update the state and control the flow.
// It can be returned by a node function instead of a direct state update.
type Command struct {
	// Update is the value to update the state with.
	// It will be processed by the schema's reducers.
	Update any

	// Goto specifies the next node(s) to execute.
	// If set, it overrides the graph's edges.
	// Can be a single string (node name) or []string.
	Goto any
}
