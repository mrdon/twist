package commands

import (
	"strconv"
	"strings"
	"twist/internal/proxy/scripting/types"
)


// RegisterVariableCommands registers all variable manipulation commands
func RegisterVariableCommands(vm CommandRegistry) {
	vm.RegisterCommand("SETVAR", 2, -1, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdSetVar)
	vm.RegisterCommand("ISNUM", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdIsNum)
	vm.RegisterCommand("VAL", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdVal)
	vm.RegisterCommand("STR", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdStr)
	// GETWORD is registered in game.go with correct TWX parameter order
	vm.RegisterCommand("GETWORDPOS", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdGetWordPos)
	vm.RegisterCommand("NUMWORDS", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdNumWords)
}

func cmdSetVar(vm types.VMInterface, params []*types.CommandParam) error {
	// Pascal setVar supports multi-parameter concatenation
	// setVar $result "part1" "part2" "part3" concatenates all parts
	if len(params) > 2 {
		// Concatenate all parameters after the variable (Pascal behavior)
		var concatenated strings.Builder
		for i := 1; i < len(params); i++ {
			concatenated.WriteString(GetParamString(vm, params[i]))
		}
		
		result := &types.Value{
			Type:   types.StringType,
			String: concatenated.String(),
		}
		vm.SetVariable(params[0].VarName, result)
	} else {
		// Single parameter assignment - use GetParamValue to properly resolve variables or literals
		resolvedValue := GetParamValue(vm, params[1])
		vm.SetVariable(params[0].VarName, resolvedValue)
	}
	return nil
}


func cmdIsNum(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	_, err := strconv.ParseFloat(text, 64)
	
	result := &types.Value{
		Type:        types.NumberType,
		Number: 0,
	}
	if err == nil {
		result.Number = 1
	}
	
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdVal(vm types.VMInterface, params []*types.CommandParam) error {
	text := strings.TrimSpace(GetParamString(vm, params[0]))
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		value = 0
	}
	
	result := &types.Value{
		Type:        types.NumberType,
		Number: value,
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdStr(vm types.VMInterface, params []*types.CommandParam) error {
	value := GetParamValue(vm, params[0])
	var text string
	if value.Type == types.NumberType {
		// Format number as string, handling integers specially
		if value.Number == float64(int64(value.Number)) {
			text = strconv.FormatInt(int64(value.Number), 10)
		} else {
			text = strconv.FormatFloat(value.Number, 'f', -1, 64)
		}
	} else {
		text = value.String
	}
	
	result := &types.Value{
		Type:        types.StringType,
		String: text,
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

// cmdGetWord moved to game.go with correct TWX parameter order

func cmdGetWordPos(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	wordNum := int(GetParamNumber(vm, params[1]))
	
	words := SplitWords(text)
	position := 0
	
	if wordNum >= 1 && wordNum <= len(words) {
		// Find the position of the word in the original text
		currentPos := 0
		for i := range words {
			if i == wordNum-1 { // TWX uses 1-based indexing
				position = currentPos + 1 // TWX uses 1-based position
				break
			}
			// Skip over the word and any following whitespace
			for currentPos < len(text) && text[currentPos] != ' ' && text[currentPos] != '\t' {
				currentPos++
			}
			for currentPos < len(text) && (text[currentPos] == ' ' || text[currentPos] == '\t') {
				currentPos++
			}
		}
	}
	
	vm.SetVariable(params[2].VarName, &types.Value{
		Type:        types.NumberType,
		Number: float64(position),
	})
	return nil
}

func cmdNumWords(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	words := SplitWords(text)
	
	vm.SetVariable(params[1].VarName, &types.Value{
		Type:        types.NumberType,
		Number: float64(len(words)),
	})
	return nil
}

// splitWords moved to game.go to avoid duplication