package commands

import (
	"strings"
	"twist/internal/debug"
	"twist/internal/proxy/scripting/types"
	"unicode/utf8"
)

// RegisterTextCommands registers all text manipulation commands
func RegisterTextCommands(vm CommandRegistry) {
	vm.RegisterCommand("ECHO", 1, -1, []types.ParameterType{types.ParamValue}, cmdEcho)
	vm.RegisterCommand("CLIENTMESSAGE", 1, 1, []types.ParameterType{types.ParamValue}, cmdClientMessage)
	vm.RegisterCommand("CLEARTEXT", 0, 0, []types.ParameterType{}, cmdClearText)
	vm.RegisterCommand("DISPLAYTEXT", 1, 1, []types.ParameterType{types.ParamValue}, cmdDisplayText)
	vm.RegisterCommand("LEN", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdLen)
	vm.RegisterCommand("MID", 4, 4, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue, types.ParamVar}, cmdMid)
	vm.RegisterCommand("LEFT", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdLeft)
	vm.RegisterCommand("RIGHT", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdRight)
	vm.RegisterCommand("INSTR", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdInStr)
	vm.RegisterCommand("UPPER", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdUpper)
	vm.RegisterCommand("LOWER", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdLower)
	vm.RegisterCommand("TRIM", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdTrim)
	vm.RegisterCommand("STRIPANSI", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdStripANSI)
	vm.RegisterCommand("CHR", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdChr)
	vm.RegisterCommand("ASC", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdAsc)
	vm.RegisterCommand("REPLACE", 4, 4, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue, types.ParamVar}, cmdReplace)
	vm.RegisterCommand("FIND", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdFind)
	vm.RegisterCommand("PADLEFT", 4, 4, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue, types.ParamVar}, cmdPadLeft)
	vm.RegisterCommand("PADRIGHT", 4, 4, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue, types.ParamVar}, cmdPadRight)
	vm.RegisterCommand("GETLENGTH", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdGetLength)
	vm.RegisterCommand("UPPERCASE", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdUppercase)
	vm.RegisterCommand("LOWERCASE", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdLowercase)
	vm.RegisterCommand("CENTER", 4, 4, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamValue, types.ParamVar}, cmdCenter)
	vm.RegisterCommand("REPEAT", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdRepeat)

	// TWX text processing commands
	vm.RegisterCommand("CUTTEXT", 4, 4, []types.ParameterType{types.ParamValue, types.ParamVar, types.ParamValue, types.ParamValue}, cmdCutText)
	vm.RegisterCommand("GETWORD", 3, 4, []types.ParameterType{types.ParamValue, types.ParamVar, types.ParamValue, types.ParamValue}, cmdGetWord)
	vm.RegisterCommand("STRIPTEXT", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdStripText)
}

// CommandRegistry interface for registering commands
type CommandRegistry interface {
	RegisterCommand(name string, minParams, maxParams int, paramTypes []types.ParameterType, handler types.CommandHandler)
}

func cmdEcho(vm types.VMInterface, params []*types.CommandParam) error {
	// Concatenate all parameters like original TWX
	echoText := ""
	for _, param := range params {
		if param.Type == types.ParamVar {
			// Get variable value
			value := vm.GetVariable(param.VarName)
			echoText += value.ToString()
		} else {
			// Use literal value
			echoText += param.Value.ToString()
		}
	}
	// Convert carriage returns to CRLF for proper display (matches TWX ScriptCmd.pas:452)
	echoText = strings.ReplaceAll(echoText, "\r", "\r\n")
	return vm.Echo(echoText)
}

func cmdClientMessage(vm types.VMInterface, params []*types.CommandParam) error {
	message := GetParamString(vm, params[0])
	return vm.ClientMessage(message)
}

func cmdClearText(vm types.VMInterface, params []*types.CommandParam) error {
	// Implementation would depend on game interface
	return nil
}

func cmdDisplayText(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	return vm.Echo(text)
}

func cmdLen(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	length := utf8.RuneCountInString(text)
	result := &types.Value{
		Type:   types.NumberType,
		Number: float64(length),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdMid(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	start := int(GetParamNumber(vm, params[1])) - 1
	length := int(GetParamNumber(vm, params[2]))

	if start < 0 {
		start = 0
	}
	if start >= len(text) {
		vm.SetVariable(params[3].VarName, &types.Value{Type: types.StringType, String: ""})
		return nil
	}

	end := start + length
	if end > len(text) {
		end = len(text)
	}

	result := &types.Value{
		Type:   types.StringType,
		String: text[start:end],
	}
	vm.SetVariable(params[3].VarName, result)
	return nil
}

func cmdLeft(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	length := int(GetParamNumber(vm, params[1]))

	if length < 0 {
		length = 0
	}
	if length > len(text) {
		length = len(text)
	}

	result := &types.Value{
		Type:   types.StringType,
		String: text[:length],
	}
	vm.SetVariable(params[2].VarName, result)
	return nil
}

func cmdRight(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	length := int(GetParamNumber(vm, params[1]))

	if length < 0 {
		length = 0
	}
	if length > len(text) {
		length = len(text)
	}

	start := len(text) - length
	result := &types.Value{
		Type:   types.StringType,
		String: text[start:],
	}
	vm.SetVariable(params[2].VarName, result)
	return nil
}

func cmdInStr(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	search := GetParamString(vm, params[1])

	pos := strings.Index(text, search)
	if pos == -1 {
		pos = 0
	} else {
		pos += 1 // TWX uses 1-based indexing
	}

	result := &types.Value{
		Type:   types.NumberType,
		Number: float64(pos),
	}
	vm.SetVariable(params[2].VarName, result)
	return nil
}

func cmdUpper(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	result := &types.Value{
		Type:   types.StringType,
		String: strings.ToUpper(text),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdLower(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	result := &types.Value{
		Type:   types.StringType,
		String: strings.ToLower(text),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdTrim(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	result := &types.Value{
		Type:   types.StringType,
		String: strings.TrimSpace(text),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdStripANSI(vm types.VMInterface, params []*types.CommandParam) error {
	// Simple ANSI stripping - would need proper regex in real implementation
	text := GetParamString(vm, params[0])
	// This is a simplified version - full implementation would use regex
	result := &types.Value{
		Type:   types.StringType,
		String: text, // Placeholder - real implementation would strip ANSI codes
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdChr(vm types.VMInterface, params []*types.CommandParam) error {
	code := int(GetParamNumber(vm, params[0]))
	if code < 0 || code > 255 {
		return vm.Error("Character code out of range")
	}

	result := &types.Value{
		Type:   types.StringType,
		String: string(rune(code)),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdAsc(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	if len(text) == 0 {
		return vm.Error("Empty string")
	}

	result := &types.Value{
		Type:   types.NumberType,
		Number: float64(text[0]),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdReplace(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	oldStr := GetParamString(vm, params[1])
	newStr := GetParamString(vm, params[2])

	result := &types.Value{
		Type:   types.StringType,
		String: strings.ReplaceAll(text, oldStr, newStr),
	}
	vm.SetVariable(params[3].VarName, result)
	return nil
}

func cmdFind(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	search := GetParamString(vm, params[1])

	pos := strings.Index(text, search)
	result := &types.Value{
		Type:   types.NumberType,
		Number: float64(pos + 1), // TWX uses 1-based indexing, 0 for not found
	}
	if pos == -1 {
		result.Number = 0
	}

	vm.SetVariable(params[2].VarName, result)
	return nil
}

func cmdPadLeft(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	width := int(GetParamNumber(vm, params[1]))
	padChar := GetParamString(vm, params[2])

	if len(padChar) == 0 {
		padChar = " "
	}

	if len(text) >= width {
		vm.SetVariable(params[3].VarName, &types.Value{Type: types.StringType, String: text})
		return nil
	}

	padLength := width - len(text)
	padding := strings.Repeat(string(padChar[0]), padLength)

	result := &types.Value{
		Type:   types.StringType,
		String: padding + text,
	}
	vm.SetVariable(params[3].VarName, result)
	return nil
}

func cmdPadRight(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	width := int(GetParamNumber(vm, params[1]))
	padChar := GetParamString(vm, params[2])

	if len(padChar) == 0 {
		padChar = " "
	}

	if len(text) >= width {
		vm.SetVariable(params[3].VarName, &types.Value{Type: types.StringType, String: text})
		return nil
	}

	padLength := width - len(text)
	padding := strings.Repeat(string(padChar[0]), padLength)

	result := &types.Value{
		Type:   types.StringType,
		String: text + padding,
	}
	vm.SetVariable(params[3].VarName, result)
	return nil
}

func cmdGetLength(vm types.VMInterface, params []*types.CommandParam) error {
	return cmdLen(vm, params)
}

func cmdUppercase(vm types.VMInterface, params []*types.CommandParam) error {
	return cmdUpper(vm, params)
}

func cmdLowercase(vm types.VMInterface, params []*types.CommandParam) error {
	return cmdLower(vm, params)
}

func cmdCenter(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	width := int(GetParamNumber(vm, params[1]))
	padChar := GetParamString(vm, params[2])

	if len(padChar) == 0 {
		padChar = " "
	}

	if len(text) >= width {
		vm.SetVariable(params[3].VarName, &types.Value{Type: types.StringType, String: text})
		return nil
	}

	totalPad := width - len(text)
	leftPad := totalPad / 2
	rightPad := totalPad - leftPad

	leftPadding := strings.Repeat(string(padChar[0]), leftPad)
	rightPadding := strings.Repeat(string(padChar[0]), rightPad)

	result := &types.Value{
		Type:   types.StringType,
		String: leftPadding + text + rightPadding,
	}
	vm.SetVariable(params[3].VarName, result)
	return nil
}

func cmdRepeat(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	count := int(GetParamNumber(vm, params[1]))

	if count < 0 {
		count = 0
	}

	result := &types.Value{
		Type:   types.StringType,
		String: strings.Repeat(text, count),
	}
	vm.SetVariable(params[2].VarName, result)
	return nil
}

// cmdCutText implements the cutText command from TWX scripts
// Syntax: cutText <source> <dest> <start> <length>
// Example: cutText CURRENTLINE $location 1 7
func cmdCutText(vm types.VMInterface, params []*types.CommandParam) error {
	text := GetParamString(vm, params[0])
	start := int(GetParamNumber(vm, params[2])) // Pascal uses 1-based indexing directly
	length := int(GetParamNumber(vm, params[3]))

	// Match Pascal behavior: error if start position beyond end of line
	if start > len(text) {
		return vm.Error("CutText: Start position beyond End Of Line")
	}

	// Convert to 0-based for Go string operations, but handle edge cases like Pascal
	if start < 1 {
		start = 1
	}

	startIdx := start - 1 // Convert to 0-based
	endIdx := startIdx + length
	if endIdx > len(text) {
		endIdx = len(text)
	}

	result := &types.Value{
		Type:   types.StringType,
		String: text[startIdx:endIdx],
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

// cmdGetWord implements the getWord command from TWX scripts
// Syntax: getWord <source> <dest> <word_number>
// Example: getWord CURRENTLINE $scanType 4
func cmdGetWord(vm types.VMInterface, params []*types.CommandParam) error {
	// TWX GETWORD syntax: GETWORD <source> <resultVar> <wordNumber> [<default>]
	// params[0] = source text, params[1] = result variable, params[2] = word number, params[3] = optional default
	text := GetParamString(vm, params[0])
	wordNum := int(GetParamNumber(vm, params[2]))

	words := SplitWords(text)

	debug.Log("GETWORD: text=%q, wordNum=%d, words=%v", text, wordNum, words)

	var result string
	if wordNum >= 1 && wordNum <= len(words) {
		result = words[wordNum-1] // TWX uses 1-based indexing
		debug.Log("GETWORD: found word %d = %q", wordNum, result)
	} else {
		// Handle empty result like Pascal implementation
		if len(params) > 3 {
			result = GetParamString(vm, params[3]) // Use provided default
		} else {
			result = "0" // Pascal default when no default provided
		}
		debug.Log("GETWORD: word %d not found, using default = %q", wordNum, result)
	}

	vm.SetVariable(params[1].VarName, &types.Value{
		Type:   types.StringType,
		String: result,
	})
	return nil
}

// cmdStripText implements the stripText command from TWX scripts
// Syntax: stripText <variable> <text_to_remove>
// Example: stripText $line "("
func cmdStripText(vm types.VMInterface, params []*types.CommandParam) error {
	// Strip specific text from variable - removes all occurrences of the specified string
	// Example: stripText $line "(" removes all "(" characters from $line
	currentValue := vm.GetVariable(params[0].VarName)
	text := currentValue.ToString()
	toRemove := GetParamString(vm, params[1])

	// Remove all occurrences of the specified text
	result := &types.Value{
		Type:   types.StringType,
		String: strings.ReplaceAll(text, toRemove, ""),
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}
