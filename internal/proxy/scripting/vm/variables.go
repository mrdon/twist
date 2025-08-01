package vm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"twist/internal/proxy/scripting/types"
)

// VariableManager manages script variables with full array support
type VariableManager struct {
	variables     map[string]*types.VarParam // All variables using VarParam system
	scriptID      string                     // Current script ID for database operations
	gameInterface types.GameInterface        // For loading persisted variables
}

// Variable name parsing regex for array indexing: $var[index1][index2]
var varIndexRegex = regexp.MustCompile(`^([^[\]]+)(?:\[([^\[\]]+)\])*$`)

// NewVariableManager creates a new variable manager
func NewVariableManager(gameInterface types.GameInterface) *VariableManager {
	return &VariableManager{
		variables:     make(map[string]*types.VarParam),
		scriptID:      "default",
		gameInterface: gameInterface,
	}
}

// SetScriptID sets the current script ID for database operations
func (vm *VariableManager) SetScriptID(scriptID string) {
	vm.scriptID = scriptID
}

// parseVariableName parses a variable name and extracts indexes and properties
// Examples: 
//   "array[1][2]" returns ("array", ["1", "2"], [])
//   "obj.prop.subprop" returns ("obj", [], ["prop", "subprop"])
//   "array[1].port.class" returns ("array", ["1"], ["port", "class"])
func (vm *VariableManager) parseVariableName(name string) (string, []string, []string) {
	// Remove $ prefix if present
	cleanName := strings.TrimPrefix(name, "$")
	
	var baseName string
	var indexes []string
	var properties []string
	
	// First, handle array indexing
	openBracket := strings.Index(cleanName, "[")
	dotIndex := strings.Index(cleanName, ".")
	
	// Determine what comes first - bracket or dot
	if openBracket == -1 && dotIndex == -1 {
		// Simple variable with no indexing or properties
		// Convert to uppercase for system constant lookup
		return strings.ToUpper(cleanName), nil, nil
	}
	
	var nameAfterBrackets string
	
	if openBracket != -1 && (dotIndex == -1 || openBracket < dotIndex) {
		// Brackets come first or are the only thing
		baseName = strings.ToUpper(cleanName[:openBracket])
		remaining := cleanName[openBracket:]
		
		// Extract all [index] pairs
		for len(remaining) > 0 && strings.HasPrefix(remaining, "[") {
			closeBracket := strings.Index(remaining, "]")
			if closeBracket == -1 {
				break
			}
			
			index := remaining[1:closeBracket]
			indexes = append(indexes, index)
			remaining = remaining[closeBracket+1:]
		}
		nameAfterBrackets = remaining
	} else {
		// Dot comes first or no brackets
		nameAfterBrackets = cleanName
	}
	
	// Now handle dot notation for properties
	if strings.Contains(nameAfterBrackets, ".") {
		if baseName == "" {
			// No brackets were processed, split at first dot
			parts := strings.SplitN(nameAfterBrackets, ".", 2)
			baseName = strings.ToUpper(parts[0])
			nameAfterBrackets = "." + parts[1]
		}
		
		// Extract properties from dot notation
		if strings.HasPrefix(nameAfterBrackets, ".") {
			propStr := strings.TrimPrefix(nameAfterBrackets, ".")
			properties = strings.Split(propStr, ".")
		}
	} else if baseName == "" {
		// No brackets and no dots
		baseName = strings.ToUpper(nameAfterBrackets)
	}
	
	return baseName, indexes, properties
}

// parseVariableNameOld is the old version that only handled arrays for backward compatibility
func (vm *VariableManager) parseVariableNameOld(name string) (string, []string) {
	baseName, indexes, _ := vm.parseVariableName(name)
	return baseName, indexes
}

// Get retrieves a variable value by name, supporting array indexing and object properties
func (vm *VariableManager) Get(name string) *types.Value {
	baseName, indexes, properties := vm.parseVariableName(name)
	
	
	// Get or create the base variable (check user variables first, then system constants)
	baseVar, exists := vm.variables[baseName]
	if !exists {
		// Try to load from database first (for individual array elements)
		if len(indexes) > 0 && vm.gameInterface != nil {
			// For array access like $test[1][2], try loading the specific element
			fullName := baseName
			for _, index := range indexes {
				fullName += "[" + index + "]"
			}
			
			if value, err := vm.gameInterface.LoadScriptVariable(fullName); err == nil {
				// Variable exists in database, return it directly
				return value
			}
		}
		
		// Check if this is a system constant (only if no user variable exists)
		if vm.gameInterface != nil {
			if systemConstants := vm.gameInterface.GetSystemConstants(); systemConstants != nil {
				if constantValue, exists := systemConstants.GetConstant(baseName); exists {
					// Handle array indexing on constants if needed (like LIBPARM[0])
					if len(indexes) > 0 {
						return vm.resolveConstantIndexing(constantValue, indexes)
					}
					// Handle property access on constants if needed (like SECTOR.WARPS)
					if len(properties) > 0 {
						return vm.resolveConstantProperties(baseName, properties)
					}
					return constantValue
				}
			}
		}
		
		// Auto-vivification: create new variable if not found anywhere
		baseVar = types.NewVarParam(baseName, types.VarParamVariable)
		vm.variables[baseName] = baseVar
	}
	
	// Resolve variable references in indexes (e.g. $bestWarp -> "2")
	resolvedIndexes := make([]string, len(indexes))
	for i, index := range indexes {
		if strings.HasPrefix(index, "$") {
			// This is a variable reference, resolve it
			indexValue := vm.Get(index)
			resolvedIndexes[i] = indexValue.ToString()
		} else {
			// This is a literal index
			resolvedIndexes[i] = index
		}
	}
	
	// Navigate to the indexed variable
	targetVar := baseVar.GetIndexVar(resolvedIndexes)
	
	// Handle object property access if present
	if len(properties) > 0 {
		return vm.getObjectProperty(targetVar, properties)
	}
	
	// Convert VarParam to Value for compatibility (GET method)
	return vm.varParamToValue(targetVar)
}

// Set sets a variable value, supporting array indexing and object properties
func (vm *VariableManager) Set(name string, value *types.Value) {
	if value == nil {
		value = &types.Value{
			Type:   types.StringType,
			String: "",
			Number: 0,
		}
	}
	
	baseName, indexes, properties := vm.parseVariableName(name)
	
	// Special handling for ArrayType values
	if value.Type == types.ArrayType && len(indexes) == 0 {
		// Setting a full array - convert Value to VarParam structure
		baseVar := vm.valueToVarParam(baseName, value)
		vm.variables[baseName] = baseVar
		return
	}
	
	// Get or create the base variable
	baseVar, exists := vm.variables[baseName]
	if !exists {
		baseVar = types.NewVarParam(baseName, types.VarParamVariable)
		vm.variables[baseName] = baseVar
	}
	
	// Resolve variable references in indexes (e.g. $bestWarp -> "2")
	resolvedIndexes := make([]string, len(indexes))
	for i, index := range indexes {
		if strings.HasPrefix(index, "$") {
			// This is a variable reference, resolve it
			indexValue := vm.Get(index)
			resolvedIndexes[i] = indexValue.ToString()
		} else {
			// This is a literal index
			resolvedIndexes[i] = index
		}
	}
	
	// Navigate to the indexed variable
	targetVar := baseVar.GetIndexVar(resolvedIndexes)
	
	// Handle object property access if present
	if len(properties) > 0 {
		vm.setObjectProperty(targetVar, properties, value)
		return
	}
	
	// Set the value
	targetVar.SetValue(vm.valueToString(value))
}

// SetVarParam sets a variable using the VarParam directly
func (vm *VariableManager) SetVarParam(name string, varParam *types.VarParam) {
	baseName, indexes := vm.parseVariableNameOld(name)
	
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
	baseName, indexes := vm.parseVariableNameOld(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		baseVar = types.NewVarParam(baseName, types.VarParamVariable)
		vm.variables[baseName] = baseVar
	}
	
	return baseVar.GetIndexVar(indexes)
}

// SetArray initializes an array variable with given dimensions
func (vm *VariableManager) SetArray(name string, dimensions []int) {
	baseName, _ := vm.parseVariableNameOld(name)
	
	varParam := types.NewVarParam(baseName, types.VarParamVariable)
	varParam.SetArray(dimensions)
	vm.variables[baseName] = varParam
}

// SetArrayFromStrings sets array from string list (TWX style)
func (vm *VariableManager) SetArrayFromStrings(name string, strings []string) {
	baseName, _ := vm.parseVariableNameOld(name)
	
	varParam := types.NewVarParam(baseName, types.VarParamVariable)
	varParam.SetArrayFromStrings(strings)
	vm.variables[baseName] = varParam
}

// Exists checks if a variable exists
func (vm *VariableManager) Exists(name string) bool {
	baseName, indexes := vm.parseVariableNameOld(name)
	
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
	baseName, indexes := vm.parseVariableNameOld(name)
	
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
			Type:   types.ArrayType,
			Array:  make(map[string]*types.Value),
			Number: float64(varParam.ArraySize), // Preserve array size
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
	baseName, indexes := vm.parseVariableNameOld(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		return 0
	}
	
	targetVar := baseVar.GetIndexVar(indexes)
	return targetVar.GetElementCount()
}

// GetArrayKeys returns the keys of an array variable
func (vm *VariableManager) GetArrayKeys(name string) []string {
	baseName, indexes := vm.parseVariableNameOld(name)
	
	baseVar, exists := vm.variables[baseName]
	if !exists {
		return nil
	}
	
	targetVar := baseVar.GetIndexVar(indexes)
	return targetVar.GetArrayKeys()
}

// IsArray checks if a variable is an array
func (vm *VariableManager) IsArray(name string) bool {
	baseName, indexes := vm.parseVariableNameOld(name)
	
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

// valueToVarParam converts a Value with ArrayType to VarParam structure
func (vm *VariableManager) valueToVarParam(name string, value *types.Value) *types.VarParam {
	if value.Type != types.ArrayType {
		// For non-arrays, create simple VarParam
		varParam := types.NewVarParam(name, types.VarParamVariable)
		varParam.SetValue(vm.valueToString(value))
		return varParam
	}
	
	// For arrays, create VarParam with array structure
	varParam := types.NewVarParam(name, types.VarParamVariable)
	varParam.Vars = make(map[string]*types.VarParam)
	
	// Set the array size from the Value.Number field (used by cmdArray)
	varParam.ArraySize = int(value.Number)
	
	// Convert all array elements
	for index, elemValue := range value.Array {
		elemName := fmt.Sprintf("%s[%s]", name, index)
		elemVarParam := vm.valueToVarParam(elemName, elemValue)
		varParam.Vars[index] = elemVarParam
	}
	
	return varParam
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

// getObjectProperty retrieves a property value from an object variable (for getSector objects)
func (vm *VariableManager) getObjectProperty(varParam *types.VarParam, properties []string) *types.Value {
	// For object property access like $s.port.class, we check if this is a special object
	// Currently we handle the case where the varParam contains an object structure
	
	// Navigate through the property chain
	current := varParam
	for _, prop := range properties {
		if current == nil {
			return &types.Value{Type: types.StringType, String: ""}
		}
		
		// Check if current VarParam has this property as a sub-variable
		if current.Vars != nil {
			if propVar, exists := current.Vars[strings.ToUpper(prop)]; exists {
				current = propVar
			} else {
				// Property doesn't exist, return empty
				return &types.Value{Type: types.StringType, String: ""}
			}
		} else {
			// No sub-variables, can't access property
			return &types.Value{Type: types.StringType, String: ""}
		}
	}
	
	// Convert the final VarParam to Value
	return vm.varParamToValue(current)
}

// setObjectProperty sets a property value in an object variable (for getSector objects)
func (vm *VariableManager) setObjectProperty(varParam *types.VarParam, properties []string, value *types.Value) {
	// Navigate through the property chain, creating structure as needed
	current := varParam
	
	// Make sure we have a vars map
	if current.Vars == nil {
		current.Vars = make(map[string]*types.VarParam)
	}
	
	// Navigate to the parent of the final property
	for i := 0; i < len(properties)-1; i++ {
		prop := strings.ToUpper(properties[i])
		
		if current.Vars == nil {
			current.Vars = make(map[string]*types.VarParam)
		}
		
		// Get or create the property variable
		if propVar, exists := current.Vars[prop]; exists {
			current = propVar
		} else {
			// Create new property variable
			newVar := types.NewVarParam(prop, types.VarParamVariable)
			current.Vars[prop] = newVar
			current = newVar
		}
	}
	
	// Set the final property
	finalProp := strings.ToUpper(properties[len(properties)-1])
	if current.Vars == nil {
		current.Vars = make(map[string]*types.VarParam)
	}
	
	// Create or update the final property
	if propVar, exists := current.Vars[finalProp]; exists {
		propVar.SetValue(vm.valueToString(value))
	} else {
		newVar := types.NewVarParam(finalProp, types.VarParamVariable)
		newVar.SetValue(vm.valueToString(value))
		current.Vars[finalProp] = newVar
	}
}

// resolveConstantIndexing handles array indexing on system constants (like LIBPARM[0])
func (vm *VariableManager) resolveConstantIndexing(constantValue *types.Value, indexes []string) *types.Value {
	// For now, return empty string for indexed constants we don't specifically handle
	// This can be extended for specific constants that support indexing
	return &types.Value{Type: types.StringType, String: ""}
}

// resolveConstantProperties handles property access on system constants (like SECTOR.WARPS)
func (vm *VariableManager) resolveConstantProperties(baseName string, properties []string) *types.Value {
	// Check if this is a dotted constant name (like SECTOR.WARPS)
	if vm.gameInterface != nil {
		if systemConstants := vm.gameInterface.GetSystemConstants(); systemConstants != nil {
			// Construct the full constant name
			fullConstantName := baseName
			for _, prop := range properties {
				fullConstantName += "." + strings.ToUpper(prop)
			}
			
			if constantValue, exists := systemConstants.GetConstant(fullConstantName); exists {
				return constantValue
			}
		}
	}
	
	// Property not found, return empty string
	return &types.Value{Type: types.StringType, String: ""}
}