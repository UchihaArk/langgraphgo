package graph

import (
	"reflect"
	"testing"
)

// TestSchemaState for testing schema
type TestSchemaState struct {
	Name  string
	Count int
	Items []string
	Data  map[string]interface{}
}

func TestNewStructSchema(t *testing.T) {
	initial := TestSchemaState{
		Name:  "test",
		Count: 0,
	}

	schema := NewStructSchema(initial, nil)

	if schema.InitialValue.Name != "test" {
		t.Errorf("Expected initial name to be 'test', got '%s'", schema.InitialValue.Name)
	}

	if schema.MergeFunc == nil {
		t.Error("MergeFunc should not be nil when nil is passed")
	}
}

func TestStructSchema_Init(t *testing.T) {
	initial := TestSchemaState{Name: "test", Count: 5}
	schema := NewStructSchema(initial, nil)

	result := schema.Init()

	if result.Name != "test" {
		t.Errorf("Expected name to be 'test', got '%s'", result.Name)
	}

	if result.Count != 5 {
		t.Errorf("Expected count to be 5, got %d", result.Count)
	}
}

func TestStructSchema_Update(t *testing.T) {
	tests := []struct {
		name     string
		initial  TestSchemaState
		new      TestSchemaState
		merge    func(TestSchemaState, TestSchemaState) (TestSchemaState, error)
		expected TestSchemaState
	}{
		{
			name: "default merge (non-zero fields)",
			initial: TestSchemaState{
				Name:  "initial",
				Count: 1,
				Items: []string{"a"},
			},
			new: TestSchemaState{
				Name:  "new",
				Count: 2,
			},
			merge: nil, // Use default merge
			expected: TestSchemaState{
				Name:  "new",      // Overwritten
				Count: 2,         // Overwritten
				Items: []string{"a"}, // Preserved
			},
		},
		{
			name: "custom merge function",
			initial: TestSchemaState{
				Name:  "initial",
				Count: 1,
			},
			new: TestSchemaState{
				Name:  "new",
				Count: 2,
			},
			merge: func(current, new TestSchemaState) (TestSchemaState, error) {
				current.Count += new.Count // Add counts
				return current, nil
			},
			expected: TestSchemaState{
				Name:  "initial", // Preserved
				Count: 3,         // 1 + 2
			},
		},
		{
			name: "zero value fields not merged",
			initial: TestSchemaState{
				Name:  "initial",
				Count: 5,
				Items: []string{"x"},
			},
			new: TestSchemaState{
				Name:  "",        // Zero value
				Count: 0,         // Zero value
				Items: nil,       // Zero value
				Data:  map[string]interface{}{"key": "value"},
			},
			merge: nil,
			expected: TestSchemaState{
				Name:  "initial", // Preserved
				Count: 5,         // Preserved
				Items: []string{"x"}, // Preserved
				Data:  map[string]interface{}{"key": "value"}, // Overwritten
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewStructSchema(tt.initial, tt.merge)
			result, err := schema.Update(tt.initial, tt.new)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Name != tt.expected.Name {
				t.Errorf("Expected name to be '%s', got '%s'", tt.expected.Name, result.Name)
			}

			if result.Count != tt.expected.Count {
				t.Errorf("Expected count to be %d, got %d", tt.expected.Count, result.Count)
			}

			if !reflect.DeepEqual(result.Items, tt.expected.Items) {
				t.Errorf("Expected items to be %v, got %v", tt.expected.Items, result.Items)
			}
		})
	}
}

func TestDefaultStructMerge(t *testing.T) {
	tests := []struct {
		name     string
		current  TestSchemaState
		new      TestSchemaState
		expected TestSchemaState
	}{
		{
			name: "merge non-zero fields",
			current: TestSchemaState{
				Name:  "current",
				Count: 1,
				Items: []string{"a", "b"},
				Data:  map[string]interface{}{"old": "value"},
			},
			new: TestSchemaState{
				Name:  "new",
				Count: 2,
				Data:  map[string]interface{}{"new": "value"},
			},
			expected: TestSchemaState{
				Name:  "new",                    // Overwritten
				Count: 2,                       // Overwritten
				Items: []string{"a", "b"},       // Preserved (zero in new)
				Data:  map[string]interface{}{"new": "value"}, // Overwritten
			},
		},
		{
			name: "all zero values in new",
			current: TestSchemaState{
				Name:  "current",
				Count: 5,
			},
			new: TestSchemaState{},
			expected: TestSchemaState{
				Name:  "current", // Preserved
				Count: 5,         // Preserved
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DefaultStructMerge(tt.current, tt.new)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Name != tt.expected.Name {
				t.Errorf("Expected name to be '%s', got '%s'", tt.expected.Name, result.Name)
			}

			if result.Count != tt.expected.Count {
				t.Errorf("Expected count to be %d, got %d", tt.expected.Count, result.Count)
			}
		})
	}
}

func TestDefaultStructMerge_NonStruct(t *testing.T) {
	// Test with non-struct type - should return new value
	current := "current"
	new := "new"

	result, err := DefaultStructMerge(current, new)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != new {
		t.Errorf("Expected result to be 'new', got '%s'", result)
	}
}

func TestOverwriteStructMerge(t *testing.T) {
	current := TestSchemaState{
		Name:  "current",
		Count: 1,
		Items: []string{"old"},
	}

	new := TestSchemaState{
		Name:  "new",
		Count: 2,
	}

	result, err := OverwriteStructMerge(current, new)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Name != "new" {
		t.Errorf("Expected name to be 'new', got '%s'", result.Name)
	}

	if result.Count != 2 {
		t.Errorf("Expected count to be 2, got %d", result.Count)
	}

	if result.Items != nil {
		t.Errorf("Expected items to be nil, got %v", result.Items)
	}
}

func TestNewFieldMerger(t *testing.T) {
	initial := TestSchemaState{Name: "test", Count: 5}
	fm := NewFieldMerger(initial)

	if fm.InitialValue.Name != "test" {
		t.Errorf("Expected initial name to be 'test', got '%s'", fm.InitialValue.Name)
	}

	if len(fm.FieldMergeFns) != 0 {
		t.Errorf("Expected FieldMergeFns to be empty, got %d", len(fm.FieldMergeFns))
	}
}

func TestFieldMerger_RegisterFieldMerge(t *testing.T) {
	initial := TestSchemaState{}
	fm := NewFieldMerger(initial)

	mergeFunc := func(current, newVal reflect.Value) reflect.Value {
		return reflect.ValueOf("merged")
	}

	fm.RegisterFieldMerge("Name", mergeFunc)

	if len(fm.FieldMergeFns) != 1 {
		t.Errorf("Expected 1 field merge function, got %d", len(fm.FieldMergeFns))
	}

	if _, ok := fm.FieldMergeFns["Name"]; !ok {
		t.Error("FieldMergeFns should contain 'Name' field")
	}
}

func TestFieldMerger_Init(t *testing.T) {
	initial := TestSchemaState{Name: "test", Count: 10}
	fm := NewFieldMerger(initial)

	result := fm.Init()

	if result.Name != "test" {
		t.Errorf("Expected name to be 'test', got '%s'", result.Name)
	}

	if result.Count != 10 {
		t.Errorf("Expected count to be 10, got %d", result.Count)
	}
}

func TestFieldMerger_Update(t *testing.T) {
	initial := TestSchemaState{Name: "old", Count: 1}
	fm := NewFieldMerger(initial)

	// Register custom merge for Count field - sum the values
	fm.RegisterFieldMerge("Count", SumIntMerge)

	current := TestSchemaState{
		Name:  "current",
		Count: 5,
		Items: []string{"a"},
	}

	new := TestSchemaState{
		Name:  "new",
		Count: 3,
		Items: []string{"b", "c"},
	}

	result, err := fm.Update(current, new)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Name should use default merge (overwrite if non-zero)
	if result.Name != "new" {
		t.Errorf("Expected name to be 'new', got '%s'", result.Name)
	}

	// Count should use custom merge (sum)
	if result.Count != 8 { // 5 + 3
		t.Errorf("Expected count to be 8, got %d", result.Count)
	}

	// Items should use default merge (overwrite if non-zero)
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result.Items))
	}
}

func TestFieldMerger_Update_NonStruct(t *testing.T) {
	fm := NewFieldMerger("not a struct")

	// Should return an error for non-struct types
	_, err := fm.Update("current", "new")
	if err == nil {
		t.Error("Expected error for non-struct type")
	}

	expectedError := "FieldMerger only works with struct types"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// Test helper functions
func TestAppendSliceMerge(t *testing.T) {
	current := reflect.ValueOf([]string{"a", "b"})
	new := reflect.ValueOf([]string{"c", "d"})

	result := AppendSliceMerge(current, new)

	resultSlice := result.Interface().([]string)

	if len(resultSlice) != 4 {
		t.Errorf("Expected 4 elements, got %d", len(resultSlice))
	}

	if resultSlice[0] != "a" || resultSlice[1] != "b" || resultSlice[2] != "c" || resultSlice[3] != "d" {
		t.Errorf("Expected [a b c d], got %v", resultSlice)
	}
}

func TestAppendSliceMerge_NonSlice(t *testing.T) {
	current := reflect.ValueOf("string")
	new := reflect.ValueOf("another")

	result := AppendSliceMerge(current, new)

	// For non-slices, should return new
	if result.String() != "another" {
		t.Errorf("Expected 'another', got '%s'", result.String())
	}
}

func TestSumIntMerge(t *testing.T) {
	current := reflect.ValueOf(5)
	new := reflect.ValueOf(3)

	result := SumIntMerge(current, new)

	if result.Int() != 8 {
		t.Errorf("Expected 8, got %d", result.Int())
	}
}

func TestSumIntMerge_NonInt(t *testing.T) {
	current := reflect.ValueOf("string")
	new := reflect.ValueOf("another")

	result := SumIntMerge(current, new)

	// For non-ints, should return new
	if result.String() != "another" {
		t.Errorf("Expected 'another', got '%s'", result.String())
	}
}

func TestOverwriteMerge(t *testing.T) {
	current := reflect.ValueOf("old")
	new := reflect.ValueOf("new")

	result := OverwriteMerge(current, new)

	if result.String() != "new" {
		t.Errorf("Expected 'new', got '%s'", result.String())
	}
}

func TestKeepCurrentMerge(t *testing.T) {
	current := reflect.ValueOf("current")
	new := reflect.ValueOf("new")

	result := KeepCurrentMerge(current, new)

	if result.String() != "current" {
		t.Errorf("Expected 'current', got '%s'", result.String())
	}
}

func TestMaxIntMerge(t *testing.T) {
	tests := []struct {
		current int
		new     int
		expect  int
	}{
		{5, 3, 5}, // Current is larger
		{3, 5, 5}, // New is larger
		{5, 5, 5}, // Equal
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			current := reflect.ValueOf(tt.current)
			new := reflect.ValueOf(tt.new)

			result := MaxIntMerge(current, new)

			if int(result.Int()) != tt.expect {
				t.Errorf("Expected %d, got %d", tt.expect, int(result.Int()))
			}
		})
	}
}

func TestMinIntMerge(t *testing.T) {
	tests := []struct {
		current int
		new     int
		expect  int
	}{
		{5, 3, 3}, // New is smaller
		{3, 5, 3}, // Current is smaller
		{5, 5, 5}, // Equal
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			current := reflect.ValueOf(tt.current)
			new := reflect.ValueOf(tt.new)

			result := MinIntMerge(current, new)

			if int(result.Int()) != tt.expect {
				t.Errorf("Expected %d, got %d", tt.expect, int(result.Int()))
			}
		})
	}
}