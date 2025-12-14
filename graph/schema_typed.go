package graph

import (
	"fmt"
	"reflect"
)

// StateSchemaTyped defines the structure and update logic for the graph state with type safety.
type StateSchemaTyped[S any] interface {
	// Init returns the initial state.
	Init() S

	// Update merges the new state into the current state.
	Update(current, new S) (S, error)
}

// StructSchema implements StateSchemaTyped for struct-based states.
// It provides a simple and type-safe way to manage struct states.
//
// Example:
//
//	type MyState struct {
//	    Count int
//	    Logs  []string
//	}
//
//	schema := graph.NewStructSchema(
//	    MyState{Count: 0},
//	    func(current, new MyState) (MyState, error) {
//	        // Merge logs (append)
//	        current.Logs = append(current.Logs, new.Logs...)
//	        // Add counts
//	        current.Count += new.Count
//	        return current, nil
//	    },
//	)
type StructSchema[S any] struct {
	InitialValue S
	MergeFunc    func(current, new S) (S, error)
}

// NewStructSchema creates a new StructSchema with the given initial value and merge function.
// If merge function is nil, a default merge function will be used that overwrites non-zero fields.
func NewStructSchema[S any](initial S, merge func(S, S) (S, error)) *StructSchema[S] {
	if merge == nil {
		merge = DefaultStructMerge[S]
	}
	return &StructSchema[S]{
		InitialValue: initial,
		MergeFunc:    merge,
	}
}

// Init returns the initial state.
func (s *StructSchema[S]) Init() S {
	return s.InitialValue
}

// Update merges the new state into the current state using the merge function.
func (s *StructSchema[S]) Update(current, new S) (S, error) {
	if s.MergeFunc != nil {
		return s.MergeFunc(current, new)
	}
	// Default: return new state
	return new, nil
}

// DefaultStructMerge provides a default merge function for struct states.
// It uses reflection to merge non-zero fields from new into current.
// This is a sensible default for most struct types.
//
// Example:
//
//	current := MyState{Count: 5, Name: "Old"}
//	new := MyState{Count: 3}  // Name is zero value
//	result, _ := DefaultStructMerge(current, new)
//	// result.Count = 3 (overwritten)
//	// result.Name = "Old" (preserved, because new.Name was zero)
func DefaultStructMerge[S any](current, new S) (S, error) {
	// Use reflection to merge non-zero fields from new into current
	currentVal := reflect.ValueOf(&current).Elem()
	newVal := reflect.ValueOf(new)

	// Check if S is a struct
	if currentVal.Kind() != reflect.Struct {
		// For non-struct types, just return new
		return new, nil
	}

	for i := 0; i < newVal.NumField(); i++ {
		fieldNew := newVal.Field(i)
		if !fieldNew.IsZero() {
			currentField := currentVal.Field(i)
			if currentField.CanSet() {
				currentField.Set(fieldNew)
			}
		}
	}
	return current, nil
}

// OverwriteStructMerge is a merge function that completely replaces the current state with the new state.
func OverwriteStructMerge[S any](current, new S) (S, error) {
	return new, nil
}

// FieldMerger provides fine-grained control over how individual struct fields are merged.
type FieldMerger[S any] struct {
	InitialValue  S
	FieldMergeFns map[string]func(currentVal, newVal reflect.Value) reflect.Value
}

// NewFieldMerger creates a new FieldMerger with the given initial value.
func NewFieldMerger[S any](initial S) *FieldMerger[S] {
	return &FieldMerger[S]{
		InitialValue:  initial,
		FieldMergeFns: make(map[string]func(currentVal, newVal reflect.Value) reflect.Value),
	}
}

// RegisterFieldMerge registers a custom merge function for a specific field.
func (fm *FieldMerger[S]) RegisterFieldMerge(fieldName string, mergeFn func(currentVal, newVal reflect.Value) reflect.Value) {
	fm.FieldMergeFns[fieldName] = mergeFn
}

// Init returns the initial state.
func (fm *FieldMerger[S]) Init() S {
	return fm.InitialValue
}

// Update merges the new state into the current state using registered field merge functions.
func (fm *FieldMerger[S]) Update(current, new S) (S, error) {
	currentVal := reflect.ValueOf(&current).Elem()
	newVal := reflect.ValueOf(new)

	if currentVal.Kind() != reflect.Struct {
		return new, fmt.Errorf("FieldMerger only works with struct types")
	}

	structType := currentVal.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldName := field.Name

		currentFieldVal := currentVal.Field(i)
		newFieldVal := newVal.Field(i)

		// Check if there's a custom merge function for this field
		if mergeFn, ok := fm.FieldMergeFns[fieldName]; ok {
			if currentFieldVal.CanSet() {
				mergedVal := mergeFn(currentFieldVal, newFieldVal)
				currentFieldVal.Set(mergedVal)
			}
		} else {
			// Default: overwrite if new value is non-zero
			if !newFieldVal.IsZero() && currentFieldVal.CanSet() {
				currentFieldVal.Set(newFieldVal)
			}
		}
	}

	return current, nil
}

// Common merge helpers for FieldMerger

// AppendSliceMerge appends new slice to current slice.
func AppendSliceMerge(current, new reflect.Value) reflect.Value {
	if current.Kind() != reflect.Slice || new.Kind() != reflect.Slice {
		return new
	}
	return reflect.AppendSlice(current, new)
}

// SumIntMerge adds two integer values.
func SumIntMerge(current, new reflect.Value) reflect.Value {
	if current.Kind() == reflect.Int && new.Kind() == reflect.Int {
		return reflect.ValueOf(current.Int() + new.Int()).Convert(current.Type())
	}
	return new
}

// OverwriteMerge always uses the new value.
func OverwriteMerge(current, new reflect.Value) reflect.Value {
	return new
}

// KeepCurrentMerge always keeps the current value (ignores new).
func KeepCurrentMerge(current, new reflect.Value) reflect.Value {
	return current
}

// MaxIntMerge takes the maximum of two integer values.
func MaxIntMerge(current, new reflect.Value) reflect.Value {
	if current.Kind() == reflect.Int && new.Kind() == reflect.Int {
		if current.Int() > new.Int() {
			return current
		}
	}
	return new
}

// MinIntMerge takes the minimum of two integer values.
func MinIntMerge(current, new reflect.Value) reflect.Value {
	if current.Kind() == reflect.Int && new.Kind() == reflect.Int {
		if current.Int() < new.Int() {
			return current
		}
	}
	return new
}
