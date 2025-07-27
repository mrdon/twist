package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// VarParamType represents the type of a variable parameter
type VarParamType int

const (
	VarParamVariable VarParamType = 1  // $variable
	VarParamConst    VarParamType = 2  // "string constant"
	VarParamSysConst VarParamType = 3  // system constants
)

// VarParam represents a variable parameter that supports arrays and indexing
type VarParam struct {
	Name      string              // Variable name (without $)
	Vars      map[string]*VarParam // Indexed sub-variables for arrays
	ArraySize int                 // Static array size (-1 for dynamic)
	Type      VarParamType        // Parameter type
	Value     string              // Actual value for leaf nodes
}

// NewVarParam creates a new variable parameter
func NewVarParam(name string, paramType VarParamType) *VarParam {
	return &VarParam{
		Name:      name,
		Vars:      make(map[string]*VarParam),
		ArraySize: -1, // Dynamic by default
		Type:      paramType,
		Value:     "",
	}
}

// GetIndexVar gets variable with complex indexing: $array[$index1][$index2]
func (v *VarParam) GetIndexVar(indexes []string) *VarParam {
	if v == nil {
		return NewVarParam("", VarParamVariable)
	}
	
	// If no indexes, return self
	if len(indexes) == 0 {
		return v
	}
	
	// Navigate through the array hierarchy
	current := v
	for i, index := range indexes {
		// Pascal bounds checking for static arrays
		if current.ArraySize > 0 {
			// Static array - perform bounds checking like Pascal
			indexNum, err := strconv.Atoi(index)
			if err != nil || indexNum < 1 || indexNum > current.ArraySize {
				// Create error like Pascal: "Static array index 'X' is out of range (must be 1-Y)"
				panic(fmt.Sprintf("Static array index '%s' is out of range (must be 1-%d)", index, current.ArraySize))
			}
			
			// Access existing static array element
			if indexVar, exists := current.Vars[index]; exists {
				current = indexVar
			} else {
				// This shouldn't happen for properly initialized static arrays
				panic(fmt.Sprintf("Static array element [%s] not properly initialized", index))
			}
		} else {
			// Dynamic array - auto-vivification like before
			// Ensure current is set up as an array
			if current.Vars == nil {
				current.Vars = make(map[string]*VarParam)
			}
			
			// Get or create the indexed variable
			if indexVar, exists := current.Vars[index]; exists {
				current = indexVar
			} else {
				// Auto-vivification: create new variable if it doesn't exist
				newVar := NewVarParam(fmt.Sprintf("%s[%s]", current.Name, index), VarParamVariable)
				current.Vars[index] = newVar
				current = newVar
			}
		}
		
		// For intermediate levels, ensure we can continue indexing
		if i < len(indexes)-1 && current.Vars == nil {
			current.Vars = make(map[string]*VarParam)
		}
	}
	
	return current
}

// SetArray initializes array with given dimensions
func (v *VarParam) SetArray(dimensions []int) {
	if v == nil || len(dimensions) == 0 {
		return
	}
	
	// Clear existing variables
	v.Vars = make(map[string]*VarParam)
	v.Value = ""
	
	// Set the first dimension size
	v.ArraySize = dimensions[0]
	
	// If multi-dimensional, create sub-arrays
	if len(dimensions) > 1 {
		for i := 1; i <= dimensions[0]; i++ {
			indexStr := strconv.Itoa(i)
			subVar := NewVarParam(fmt.Sprintf("%s[%s]", v.Name, indexStr), VarParamVariable)
			subVar.SetArray(dimensions[1:]) // Recursive for remaining dimensions
			v.Vars[indexStr] = subVar
		}
	} else {
		// Single dimension - create variables initialized with "0" like Pascal
		for i := 1; i <= dimensions[0]; i++ {
			indexStr := strconv.Itoa(i)
			subVar := NewVarParam(fmt.Sprintf("%s[%s]", v.Name, indexStr), VarParamVariable)
			subVar.Value = "0" // Pascal default initialization
			v.Vars[indexStr] = subVar
		}
	}
}

// SetArrayFromStrings sets array from string list (used in TWX)
func (v *VarParam) SetArrayFromStrings(strings []string) {
	if v == nil {
		return
	}
	
	// Clear existing variables
	v.Vars = make(map[string]*VarParam)
	v.Value = ""
	v.ArraySize = len(strings)
	
	// Set each string value with 1-based indexing (TWX style)
	for i, str := range strings {
		indexStr := strconv.Itoa(i + 1)
		subVar := NewVarParam(fmt.Sprintf("%s[%s]", v.Name, indexStr), VarParamVariable)
		subVar.Value = str
		v.Vars[indexStr] = subVar
	}
}

// GetValue returns the string value of this variable
func (v *VarParam) GetValue() string {
	if v == nil {
		return ""
	}
	return v.Value
}

// SetValue sets the string value of this variable
func (v *VarParam) SetValue(value string) {
	if v != nil {
		v.Value = value
	}
}

// IsArray returns true if this variable has array elements
func (v *VarParam) IsArray() bool {
	return v != nil && len(v.Vars) > 0
}

// GetArraySize returns the declared array size (-1 for dynamic)
func (v *VarParam) GetArraySize() int {
	if v == nil {
		return 0
	}
	return v.ArraySize
}

// GetArrayKeys returns all array indexes as strings
func (v *VarParam) GetArrayKeys() []string {
	if v == nil || v.Vars == nil {
		return nil
	}
	
	keys := make([]string, 0, len(v.Vars))
	for key := range v.Vars {
		keys = append(keys, key)
	}
	return keys
}

// Clone creates a deep copy of the variable parameter
func (v *VarParam) Clone() *VarParam {
	if v == nil {
		return nil
	}
	
	clone := &VarParam{
		Name:      v.Name,
		ArraySize: v.ArraySize,
		Type:      v.Type,
		Value:     v.Value,
		Vars:      make(map[string]*VarParam),
	}
	
	// Deep copy all sub-variables
	for key, subVar := range v.Vars {
		clone.Vars[key] = subVar.Clone()
	}
	
	return clone
}

// ToJSON serializes the variable parameter to JSON for database storage
func (v *VarParam) ToJSON() (string, error) {
	if v == nil {
		return "", nil
	}
	
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VarParam to JSON: %w", err)
	}
	
	return string(data), nil
}

// FromJSON deserializes the variable parameter from JSON
func (v *VarParam) FromJSON(jsonStr string) error {
	if v == nil {
		return fmt.Errorf("cannot deserialize into nil VarParam")
	}
	
	if jsonStr == "" {
		return nil
	}
	
	err := json.Unmarshal([]byte(jsonStr), v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal VarParam from JSON: %w", err)
	}
	
	return nil
}

// GetFullPath returns the full path of this variable (for debugging)
func (v *VarParam) GetFullPath() string {
	if v == nil {
		return ""
	}
	return v.Name
}

// ClearArray removes all array elements
func (v *VarParam) ClearArray() {
	if v != nil {
		v.Vars = make(map[string]*VarParam)
		v.ArraySize = -1
	}
}

// GetArrayElement gets a specific array element by index
func (v *VarParam) GetArrayElement(index string) *VarParam {
	if v == nil || v.Vars == nil {
		return NewVarParam("", VarParamVariable)
	}
	
	if elem, exists := v.Vars[index]; exists {
		return elem
	}
	
	// Auto-vivification: create new element if it doesn't exist
	newVar := NewVarParam(fmt.Sprintf("%s[%s]", v.Name, index), VarParamVariable)
	v.Vars[index] = newVar
	return newVar
}

// SetArrayElement sets a specific array element by index
func (v *VarParam) SetArrayElement(index string, value *VarParam) {
	if v == nil {
		return
	}
	
	if v.Vars == nil {
		v.Vars = make(map[string]*VarParam)
	}
	
	if value != nil {
		// Update name to reflect new position
		value.Name = fmt.Sprintf("%s[%s]", v.Name, index)
	}
	
	v.Vars[index] = value
}

// GetElementCount returns the number of elements in the array
func (v *VarParam) GetElementCount() int {
	if v == nil || v.Vars == nil {
		return 0
	}
	return len(v.Vars)
}

// ToString returns a string representation for debugging
func (v *VarParam) ToString() string {
	if v == nil {
		return "<nil>"
	}
	
	if !v.IsArray() {
		return fmt.Sprintf("%s = %q", v.Name, v.Value)
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s[%d elements]:", v.Name, len(v.Vars)))
	for key, subVar := range v.Vars {
		sb.WriteString(fmt.Sprintf(" [%s]=%q", key, subVar.GetValue()))
	}
	
	return sb.String()
}