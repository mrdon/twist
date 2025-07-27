package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ValueType represents the type of a script value
type ValueType int

const (
	StringType ValueType = iota
	NumberType
	ArrayType
)

// Value represents a variable in the script system
type Value struct {
	Type      ValueType
	String    string
	Number    float64
	Array     map[string]*Value
	ArraySize int // For TWX compatibility - stores the static array size for this dimension
}

// NewStringValue creates a new string value
func NewStringValue(s string) *Value {
	return &Value{
		Type:   StringType,
		String: s,
	}
}

// NewNumberValue creates a new number value
func NewNumberValue(n float64) *Value {
	return &Value{
		Type:   NumberType,
		Number: n,
	}
}

// NewArrayValue creates a new array value
func NewArrayValue() *Value {
	return &Value{
		Type:  ArrayType,
		Array: make(map[string]*Value),
	}
}

// ToString converts the value to a string
func (v *Value) ToString() string {
	if v == nil {
		return ""
	}
	
	switch v.Type {
	case StringType:
		return v.String
	case NumberType:
		// Format like TWX does (integers without decimals)
		if v.Number == float64(int64(v.Number)) {
			return fmt.Sprintf("%.0f", v.Number)
		}
		return fmt.Sprintf("%g", v.Number)
	case ArrayType:
		return "" // Arrays don't convert to strings directly
	default:
		return ""
	}
}

// ToNumber converts the value to a number
func (v *Value) ToNumber() float64 {
	if v == nil {
		return 0
	}
	
	switch v.Type {
	case NumberType:
		return v.Number
	case StringType:
		// Try to parse the string as a number
		if num, err := strconv.ParseFloat(strings.TrimSpace(v.String), 64); err == nil {
			return num
		}
		return 0
	case ArrayType:
		return 0 // Arrays don't convert to numbers
	default:
		return 0
	}
}

// ToBool converts the value to a boolean (TWX style)
func (v *Value) ToBool() bool {
	if v == nil {
		return false
	}
	
	switch v.Type {
	case NumberType:
		return v.Number != 0
	case StringType:
		return v.String != "" && strings.ToUpper(v.String) != "FALSE"
	case ArrayType:
		return len(v.Array) > 0
	default:
		return false
	}
}

// IsNumber checks if the value represents a valid number
func (v *Value) IsNumber() bool {
	if v == nil {
		return false
	}
	
	switch v.Type {
	case NumberType:
		return true
	case StringType:
		_, err := strconv.ParseFloat(strings.TrimSpace(v.String), 64)
		return err == nil
	default:
		return false
	}
}

// GetArrayElement gets an element from an array
func (v *Value) GetArrayElement(index string) *Value {
	if v == nil {
		return NewStringValue("")
	}
	
	// Convert to array if not already (auto-vivification)
	if v.Type != ArrayType {
		v.Type = ArrayType
		v.Array = make(map[string]*Value)
		v.String = ""
		v.Number = 0
	}
	
	if elem, exists := v.Array[index]; exists {
		return elem
	}
	
	// Return empty string for non-existent elements
	return NewStringValue("")
}

// SetArrayElement sets an element in an array
func (v *Value) SetArrayElement(index string, value *Value) {
	if v == nil {
		return
	}
	
	// Convert to array if not already
	if v.Type != ArrayType {
		v.Type = ArrayType
		v.Array = make(map[string]*Value)
		v.String = ""
		v.Number = 0
	}
	
	v.Array[index] = value
}

// Clone creates a deep copy of the value
func (v *Value) Clone() *Value {
	if v == nil {
		return NewStringValue("")
	}
	
	switch v.Type {
	case StringType:
		return NewStringValue(v.String)
	case NumberType:
		return NewNumberValue(v.Number)
	case ArrayType:
		clone := NewArrayValue()
		for k, val := range v.Array {
			clone.Array[k] = val.Clone()
		}
		return clone
	default:
		return NewStringValue("")
	}
}

// GetArraySize returns the size of an array
func (v *Value) GetArraySize() int {
	if v == nil || v.Type != ArrayType {
		return 0
	}
	return len(v.Array)
}

// GetArrayKeys returns the keys of an array in sorted order
func (v *Value) GetArrayKeys() []string {
	if v == nil || v.Type != ArrayType {
		return nil
	}
	
	keys := make([]string, 0, len(v.Array))
	for k := range v.Array {
		keys = append(keys, k)
	}
	
	// Sort keys numerically if they're all numbers
	return keys
}

// GetArrayElementMulti gets an element from a multi-dimensional array
func (v *Value) GetArrayElementMulti(indices []string) *Value {
	current := v
	
	for i, index := range indices {
		if current == nil {
			return NewStringValue("")
		}
		
		// For the last index, get the element
		if i == len(indices)-1 {
			return current.GetArrayElement(index)
		}
		
		// For intermediate indices, get the sub-array
		current = current.GetArrayElement(index)
		if current.Type != ArrayType {
			// Convert to array for auto-vivification
			current.Type = ArrayType
			current.Array = make(map[string]*Value)
			current.String = ""
			current.Number = 0
		}
	}
	
	return NewStringValue("")
}

// SetArrayElementMulti sets an element in a multi-dimensional array
func (v *Value) SetArrayElementMulti(indices []string, value *Value) {
	if v == nil || len(indices) == 0 {
		return
	}
	
	// Single index case
	if len(indices) == 1 {
		v.SetArrayElement(indices[0], value)
		return
	}
	
	// Multi-dimensional case
	current := v
	for i := 0; i < len(indices)-1; i++ {
		next := current.GetArrayElement(indices[i])
		if next.Type != ArrayType {
			next.Type = ArrayType
			next.Array = make(map[string]*Value)
			next.String = ""
			next.Number = 0
		}
		current = next
	}
	
	// Set the final element
	current.SetArrayElement(indices[len(indices)-1], value)
}

// IsArray checks if the value is an array
func (v *Value) IsArray() bool {
	return v != nil && v.Type == ArrayType
}

// ClearArray removes all elements from an array
func (v *Value) ClearArray() {
	if v != nil && v.Type == ArrayType {
		v.Array = make(map[string]*Value)
	}
}

// SetArrayDimensions sets up a multi-dimensional array like TWX's SetArray
func (v *Value) SetArrayDimensions(dimensions []int) {
	if v == nil || len(dimensions) == 0 {
		return
	}
	
	// Convert to array type
	v.Type = ArrayType
	v.Array = make(map[string]*Value)
	v.String = ""
	v.Number = 0
	v.ArraySize = dimensions[0]
	
	// Create elements for the first dimension
	for i := 1; i <= dimensions[0]; i++ {
		indexStr := fmt.Sprintf("%d", i) // TWX uses 1-based indexing
		elem := &Value{
			Type:   StringType,
			String: "",
		}
		
		// If there are more dimensions, recursively set them up
		if len(dimensions) > 1 {
			elem.SetArrayDimensions(dimensions[1:])
		}
		
		v.Array[indexStr] = elem
	}
}

// GetArrayBounds returns the bounds for multi-dimensional arrays
func (v *Value) GetArrayBounds() []int {
	if v == nil || v.Type != ArrayType {
		return []int{}
	}
	
	bounds := []int{v.ArraySize}
	
	// Check if elements have sub-dimensions
	if v.ArraySize > 0 {
		firstKey := "1"
		if firstElem, exists := v.Array[firstKey]; exists && firstElem.Type == ArrayType {
			subBounds := firstElem.GetArrayBounds()
			bounds = append(bounds, subBounds...)
		}
	}
	
	return bounds
}

// GetStaticArrayElement gets an element with bounds checking for static arrays (TWX 1-based indexing)
func (v *Value) GetStaticArrayElement(index int) (*Value, error) {
	if v == nil || v.Type != ArrayType {
		return nil, fmt.Errorf("variable is not an array")
	}
	
	// TWX uses 1-based indexing
	if v.ArraySize > 0 && (index < 1 || index > v.ArraySize) {
		return nil, fmt.Errorf("array index out of bounds")
	}
	
	indexStr := fmt.Sprintf("%d", index)
	if elem, exists := v.Array[indexStr]; exists {
		return elem, nil
	}
	
	// Return empty string for non-existent elements in dynamic arrays
	return NewStringValue(""), nil
}

// SetStaticArrayElement sets an element with bounds checking for static arrays (TWX 1-based indexing)
func (v *Value) SetStaticArrayElement(index int, value *Value) error {
	if v == nil || v.Type != ArrayType {
		return fmt.Errorf("variable is not an array")
	}
	
	// TWX uses 1-based indexing
	if v.ArraySize > 0 && (index < 1 || index > v.ArraySize) {
		return fmt.Errorf("array index out of bounds")
	}
	
	indexStr := fmt.Sprintf("%d", index)
	v.Array[indexStr] = value
	return nil
}