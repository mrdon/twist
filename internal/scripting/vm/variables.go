package vm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"twist/internal/scripting/types"
)

// VariableManager manages script variables with full array support
type VariableManager struct {
	variables map[string]*types.VarParam // All variables using VarParam system
	scriptID  string                     // Current script ID for database operations
}

// Variable name parsing regex for array indexing: $var[index1][index2]
var varIndexRegex = regexp.MustCompile(`^([^[\]]+)(?:\[([^\[\]]+)\])*$`)

// NewVariableManager creates a new variable manager
func NewVariableManager() *VariableManager {
	return &VariableManager{
		variables: make(map[string]*types.VarParam),
		scriptID:  "default",
	}
}

// SetScriptID sets the current script ID for database operations
func (vm *VariableManager) SetScriptID(scriptID string) {
	vm.scriptID = scriptID
}

// parseVariableName parses a variable name and extracts indexes
// Example: "array[1][2]" returns ("array", ["1", "2"])
func (vm *VariableManager) parseVariableName(name string) (string, []string) {
	// Remove $ prefix if present
	cleanName := strings.TrimPrefix(name, "$")
	
	// Find all bracket pairs
	var baseName string
	var indexes []string
	
	// Simple parsing - find base name and extract indexes
	openBracket := strings.Index(cleanName, "[")
	if openBracket == -1 {
		// No brackets, simple variable
		return cleanName, nil
	}
	
	baseName = cleanName[:openBracket]
	remaining := cleanName[openBracket:]
	
	// Extract all [index] pairs
	for len(remaining) > 0 {
		if !strings.HasPrefix(remaining, "[") {
			break
		}
		
		closeBracket := strings.Index(remaining, "]")
		if closeBracket == -1 {
			break
		}
		
		index := remaining[1:closeBracket]
		indexes = append(indexes, index)
		remaining = remaining[closeBracket+1:]
	}
	
	return baseName, indexes
}

// Get retrieves a variable value by name, supporting array indexing
func (vm *VariableManager) Get(name string) *types.Value {
	baseName, indexes := vm.parseVariableName(name)
	
	// Get or create the base variable
	baseVar, exists := vm.variables[baseName]
	if !exists {
		// Auto-vivification: create new variable
		baseVar = types.NewVarParam(baseName, types.VarParamVariable)
		vm.variables[baseName] = baseVar
	}
	
	// Navigate to the indexed variable
	targetVar := baseVar.GetIndexVar(indexes)
	
	// Convert VarParam to Value for compatibility
	return vm.varParamToValue(targetVar)
}

// Set sets a variable value, supporting array indexing
func (vm *VariableManager) Set(name string, value *types.Value) {
	if value == nil {
		value = &types.Value{
			Type:   types.StringType,
			String: "",
			Number: 0,
		}
	}
	
	baseName, indexes := vm.parseVariableName(name)
	
	// Get or create the base variable
	baseVar, exists := vm.variables[baseName]
	if !exists {
		baseVar = types.NewVarParam(baseName, types.VarParamVariable)
		vm.variables[baseName] = baseVar
	}
	
	// Navigate to the indexed variable
	targetVar := baseVar.GetIndexVar(indexes)
	
	// Set the value
	targetVar.SetValue(vm.valueToString(value))
}

// SetVarParam sets a variable using the VarParam directly
func (vm *VariableManager) SetVarParam(name string, varParam *types.VarParam) {
	baseName, indexes := vm.parseVariableName(name)
	
	if len(indexes) == 0 {
		// Setting base variable
		vm.variables[baseName] = varParam
	} else {
		// Setting indexed variable
		baseVar, exists := vm.variables[baseName]
		if !exists {
			baseVar = types.NewVarParam(baseName, types.VarParamVariable)
			vm.variables[baseName] = baseVar
		}
		
		// Navigate to parent and set the indexed element
		if len(indexes) == 1 {
			baseVar.SetArrayElement(indexes[0], varParam)
		} else {
			parentVar := baseVar.GetIndexVar(indexes[:len(indexes)-1])
			parentVar.SetArrayElement(indexes[len(indexes)-1], varParam)
		}
	}
}

// GetVarParam gets the VarParam directly for advanced operations
func (vm *VariableManager) GetVarParam(name string) *types.VarParam {
	baseName, indexes := vm.parseVariableName(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		baseVar = types.NewVarParam(baseName, types.VarParamVariable)
		vm.variables[baseName] = baseVar
	}
	
	return baseVar.GetIndexVar(indexes)
}

// SetArray initializes an array variable with given dimensions
func (vm *VariableManager) SetArray(name string, dimensions []int) {
	baseName, _ := vm.parseVariableName(name)
	
	varParam := types.NewVarParam(baseName, types.VarParamVariable)
	varParam.SetArray(dimensions)
	vm.variables[baseName] = varParam
}

// SetArrayFromStrings sets array from string list (TWX style)
func (vm *VariableManager) SetArrayFromStrings(name string, strings []string) {
	baseName, _ := vm.parseVariableName(name)
	
	varParam := types.NewVarParam(baseName, types.VarParamVariable)
	varParam.SetArrayFromStrings(strings)
	vm.variables[baseName] = varParam
}

// Exists checks if a variable exists
func (vm *VariableManager) Exists(name string) bool {
	baseName, indexes := vm.parseVariableName(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		return false
	}
	
	if len(indexes) == 0 {
		return true
	}
	
	// Check if indexed element exists
	targetVar := baseVar.GetIndexVar(indexes)
	return targetVar != nil && targetVar.GetValue() != ""
}

// Delete removes a variable
func (vm *VariableManager) Delete(name string) {
	baseName, indexes := vm.parseVariableName(name)
	
	if len(indexes) == 0 {
		// Delete entire variable
		delete(vm.variables, baseName)
	} else {
		// Delete indexed element
		if baseVar, exists := vm.variables[baseName]; exists {
			if len(indexes) == 1 {
				// Delete direct child
				if baseVar.Vars != nil {
					delete(baseVar.Vars, indexes[0])
				}
			} else {
				// Navigate to parent and delete
				parentVar := baseVar.GetIndexVar(indexes[:len(indexes)-1])
				if parentVar != nil && parentVar.Vars != nil {
					delete(parentVar.Vars, indexes[len(indexes)-1])
				}
			}
		}
	}
}

// Clear removes all variables
func (vm *VariableManager) Clear() {
	vm.variables = make(map[string]*types.VarParam)
}

// GetAll returns all variables as Value map for compatibility
func (vm *VariableManager) GetAll() map[string]*types.Value {
	result := make(map[string]*types.Value)
	
	for name, varParam := range vm.variables {
		result[name] = vm.varParamToValue(varParam)
		
		// Add array elements if any
		if varParam.IsArray() {
			vm.addArrayElements(name, varParam, result)
		}
	}
	
	return result
}

// addArrayElements recursively adds array elements to the result map
func (vm *VariableManager) addArrayElements(baseName string, varParam *types.VarParam, result map[string]*types.Value) {
	if varParam == nil || !varParam.IsArray() {
		return
	}
	
	for index, subVar := range varParam.Vars {
		elementName := fmt.Sprintf("%s[%s]", baseName, index)
		result[elementName] = vm.varParamToValue(subVar)
		
		// Recursively add sub-elements
		if subVar.IsArray() {
			vm.addArrayElements(elementName, subVar, result)
		}
	}
}

// Count returns the number of base variables
func (vm *VariableManager) Count() int {
	return len(vm.variables)
}

// GetNames returns all base variable names
func (vm *VariableManager) GetNames() []string {
	names := make([]string, 0, len(vm.variables))
	for name := range vm.variables {
		names = append(names, name)
	}
	return names
}

// varParamToValue converts VarParam to Value for compatibility
func (vm *VariableManager) varParamToValue(varParam *types.VarParam) *types.Value {
	if varParam == nil {
		return &types.Value{
			Type:   types.StringType,
			String: "",
			Number: 0,
		}
	}
	
	if varParam.IsArray() {
		// Convert array to Value with array type
		value := &types.Value{
			Type:  types.ArrayType,
			Array: make(map[string]*types.Value),
		}
		
		for index, subVar := range varParam.Vars {
			value.Array[index] = vm.varParamToValue(subVar)
		}
		
		return value
	}
	
	// Simple value
	valueStr := varParam.GetValue()
	
	// Try to parse as number
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return &types.Value{
			Type:   types.NumberType,
			Number: num,
			String: valueStr,
		}
	}
	
	// String value
	return &types.Value{
		Type:   types.StringType,
		String: valueStr,
	}
}

// valueToString converts Value to string for storage in VarParam
func (vm *VariableManager) valueToString(value *types.Value) string {
	if value == nil {
		return ""
	}
	
	switch value.Type {
	case types.StringType:
		return value.String
	case types.NumberType:
		// Format like TWX does (integers without decimals)
		if value.Number == float64(int64(value.Number)) {
			return fmt.Sprintf("%.0f", value.Number)
		}
		return fmt.Sprintf("%g", value.Number)
	case types.ArrayType:
		// Arrays don't convert to strings directly
		return ""
	default:
		return ""
	}
}

// GetArrayElementCount returns the number of elements in an array variable
func (vm *VariableManager) GetArrayElementCount(name string) int {
	baseName, indexes := vm.parseVariableName(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		return 0
	}
	
	targetVar := baseVar.GetIndexVar(indexes)
	return targetVar.GetElementCount()
}

// GetArrayKeys returns the keys of an array variable
func (vm *VariableManager) GetArrayKeys(name string) []string {
	baseName, indexes := vm.parseVariableName(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		return nil
	}
	
	targetVar := baseVar.GetIndexVar(indexes)
	return targetVar.GetArrayKeys()
}

// IsArray checks if a variable is an array
func (vm *VariableManager) IsArray(name string) bool {
	baseName, indexes := vm.parseVariableName(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		return false
	}
	
	targetVar := baseVar.GetIndexVar(indexes)
	return targetVar.IsArray()
}

// Clone creates a deep copy of all variables
func (vm *VariableManager) Clone() *VariableManager {
	clone := &VariableManager{
		variables: make(map[string]*types.VarParam),
		scriptID:  vm.scriptID,
	}
	
	for name, varParam := range vm.variables {
		clone.variables[name] = varParam.Clone()
	}
	
	return clone
}

// ToJSON serializes all variables to JSON for database persistence
func (vm *VariableManager) ToJSON() (string, error) {
	data := make(map[string]interface{})
	data["scriptID"] = vm.scriptID
	data["variables"] = vm.variables
	
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal variables to JSON: %w", err)
	}
	
	return string(jsonBytes), nil
}

// FromJSON deserializes variables from JSON for database restoration
func (vm *VariableManager) FromJSON(jsonStr string) error {
	if jsonStr == "" {
		return nil
	}
	
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal variables from JSON: %w", err)
	}
	
	if scriptID, ok := data["scriptID"].(string); ok {
		vm.scriptID = scriptID
	}
	
	if variables, ok := data["variables"].(map[string]interface{}); ok {
		vm.variables = make(map[string]*types.VarParam)
		for name, varData := range variables {
			varParam := types.NewVarParam(name, types.VarParamVariable)
			if varDataBytes, err := json.Marshal(varData); err == nil {
				varParam.FromJSON(string(varDataBytes))
				vm.variables[name] = varParam
			}
		}
	}
	
	return nil
}