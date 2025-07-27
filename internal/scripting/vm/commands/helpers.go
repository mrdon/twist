package commands

import (
	"twist/internal/scripting/types"
)

// GetParamValue gets the actual value from a command parameter, handling both variables and literals
func GetParamValue(vm types.VMInterface, param *types.CommandParam) *types.Value {
	if param.Type == types.ParamVar {
		return vm.GetVariable(param.VarName)
	}
	return param.Value
}

// GetParamString gets the string value from a command parameter
func GetParamString(vm types.VMInterface, param *types.CommandParam) string {
	value := GetParamValue(vm, param)
	return value.ToString()
}

// GetParamNumber gets the numeric value from a command parameter  
func GetParamNumber(vm types.VMInterface, param *types.CommandParam) float64 {
	value := GetParamValue(vm, param)
	return value.ToNumber()
}

// SplitWords splits text into words, handling multiple spaces and tabs
func SplitWords(text string) []string {
	var words []string
	var currentWord []rune
	
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if len(currentWord) > 0 {
				words = append(words, string(currentWord))
				currentWord = nil
			}
		} else {
			currentWord = append(currentWord, r)
		}
	}
	
	if len(currentWord) > 0 {
		words = append(words, string(currentWord))
	}
	
	return words
}