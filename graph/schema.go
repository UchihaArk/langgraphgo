package graph

import (
	"fmt"
	"reflect"
)

// Reducer defines how a state value should be updated.
// It takes the current value and the new value, and returns the merged value.
type Reducer func(current, new any) (any, error)

// StateSchema defines the structure and update logic for the graph state.
type StateSchema interface {
	// Init returns the initial state.
	Init() any

	// Update merges the new state into the current state.
	Update(current, new any) (any, error)
}

// MapSchema implements StateSchema for map[string]any.
// It allows defining reducers for specific keys.
type MapSchema struct {
	Reducers map[string]Reducer
}

// NewMapSchema creates a new MapSchema.
func NewMapSchema() *MapSchema {
	return &MapSchema{
		Reducers: make(map[string]Reducer),
	}
}

// RegisterReducer adds a reducer for a specific key.
func (s *MapSchema) RegisterReducer(key string, reducer Reducer) {
	s.Reducers[key] = reducer
}

// Init returns an empty map.
func (s *MapSchema) Init() any {
	return make(map[string]any)
}

// Update merges the new map into the current map using registered reducers.
func (s *MapSchema) Update(current, new any) (any, error) {
	if current == nil {
		current = make(map[string]any)
	}

	currMap, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("current state is not a map[string]any")
	}

	newMap, ok := new.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("new state is not a map[string]any")
	}

	// Create a copy of the current map to avoid mutating it directly
	result := make(map[string]any, len(currMap))
	for k, v := range currMap {
		result[k] = v
	}

	for k, v := range newMap {
		if reducer, ok := s.Reducers[k]; ok {
			// Use reducer
			currVal := result[k]
			mergedVal, err := reducer(currVal, v)
			if err != nil {
				return nil, fmt.Errorf("failed to reduce key %s: %w", k, err)
			}
			result[k] = mergedVal
		} else {
			// Default: Overwrite
			result[k] = v
		}
	}

	return result, nil
}

// Common Reducers

// OverwriteReducer replaces the old value with the new one.
func OverwriteReducer(current, new any) (any, error) {
	return new, nil
}

// AppendReducer appends the new value to the current slice.
// It supports appending a slice to a slice, or a single element to a slice.
func AppendReducer(current, new any) (any, error) {
	if current == nil {
		// If current is nil, start a new slice
		// We need to know the type? We can infer from new.
		newVal := reflect.ValueOf(new)
		if newVal.Kind() == reflect.Slice {
			return new, nil
		}
		// Create slice of type of new
		sliceType := reflect.SliceOf(reflect.TypeOf(new))
		slice := reflect.MakeSlice(sliceType, 0, 1)
		slice = reflect.Append(slice, newVal)
		return slice.Interface(), nil
	}

	currVal := reflect.ValueOf(current)
	newVal := reflect.ValueOf(new)

	if currVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("current value is not a slice")
	}

	if newVal.Kind() == reflect.Slice {
		// Append slice to slice
		if currVal.Type().Elem() != newVal.Type().Elem() {
			// Types don't match, convert both to []any
			result := make([]any, 0, currVal.Len()+newVal.Len())
			for i := 0; i < currVal.Len(); i++ {
				result = append(result, currVal.Index(i).Interface())
			}
			for i := 0; i < newVal.Len(); i++ {
				result = append(result, newVal.Index(i).Interface())
			}
			return result, nil
		}
		return reflect.AppendSlice(currVal, newVal).Interface(), nil
	}

	// Append single element
	return reflect.Append(currVal, newVal).Interface(), nil
}
