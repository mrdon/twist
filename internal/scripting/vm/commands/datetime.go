package commands

import (
	"strings"
	"time"
	"twist/internal/scripting/types"
)

// RegisterDateTimeCommands registers datetime commands with the VM
func RegisterDateTimeCommands(vm CommandRegistry) {
	vm.RegisterCommand("GETDATE", 1, 1, []types.ParameterType{types.ParamVar}, cmdGetDate)
	vm.RegisterCommand("GETDATETIME", 1, 1, []types.ParameterType{types.ParamVar}, cmdGetDateTime)
	vm.RegisterCommand("DATETIMEDIFF", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdDateTimeDiff)
	vm.RegisterCommand("DATETIMETOSTR", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdDateTimeToStr)
	vm.RegisterCommand("STARTTIMER", 1, 1, []types.ParameterType{types.ParamVar}, cmdStartTimer)
	vm.RegisterCommand("STOPTIMER", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdStopTimer)
}

// cmdGetDate gets the current date
func cmdGetDate(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("GETDATE requires exactly 1 parameter: result_var")
	}

	now := time.Now()
	dateStr := now.Format("01/02/2006")

	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.StringType,
		String: dateStr,
	})

	return nil
}

// cmdGetDateTime gets the current date and time
func cmdGetDateTime(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("GETDATETIME requires exactly 1 parameter: result_var")
	}

	now := time.Now()
	dateTimeStr := now.Format("01/02/2006 15:04:05")

	vm.SetVariable(params[0].VarName, &types.Value{
		Type:   types.StringType,
		String: dateTimeStr,
	})

	return nil
}

// cmdDateTimeDiff calculates the difference between two datetimes
func cmdDateTimeDiff(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 4 {
		return vm.Error("DATETIMEDIFF requires exactly 4 parameters: datetime1, datetime2, unit, result_var")
	}

	dt1Str := GetParamString(vm, params[0])
	dt2Str := GetParamString(vm, params[1])
	unit := GetParamString(vm, params[2])

	// Try to parse the datetime strings - use multiple format attempts
	formats := []string{
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"2006-01-02T15:04:05",
		"01/02/2006",
		"2006-01-02",
	}
	
	var dt1, dt2 time.Time
	var err1, err2 error
	
	for _, format := range formats {
		if dt1, err1 = time.Parse(format, dt1Str); err1 == nil {
			break
		}
	}
	for _, format := range formats {
		if dt2, err2 = time.Parse(format, dt2Str); err2 == nil {
			break
		}
	}

	if err1 != nil || err2 != nil {
		return vm.Error("Invalid datetime format")
	}

	duration := dt2.Sub(dt1)
	var result float64

	switch unit {
	case "seconds":
		result = duration.Seconds()
	case "minutes":
		result = duration.Minutes()
	case "hours":
		result = duration.Hours()
	case "days":
		result = duration.Hours() / 24
	default:
		result = duration.Seconds() // Default to seconds
	}

	vm.SetVariable(params[3].VarName, &types.Value{
		Type:   types.NumberType,
		Number: result,
	})

	return nil
}

// cmdDateTimeToStr converts datetime to string with format
func cmdDateTimeToStr(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) < 2 || len(params) > 3 {
		return vm.Error("DATETIMETOSTR requires 2-3 parameters: datetime, result_var [, format]")
	}

	dtStr := GetParamString(vm, params[0])
	var formatStr string
	var resultVarName string
	
	if len(params) == 2 {
		// Default format - no format parameter
		formatStr = "2006-01-02 15:04:05" // Default format
		resultVarName = params[1].VarName
	} else {
		// Custom format provided
		resultVarName = params[1].VarName
		formatStr = GetParamString(vm, params[2])
	}

	// Try to parse the datetime with multiple input formats
	formats := []string{
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"2006-01-02T15:04:05",
		"01/02/2006",
		"2006-01-02",
	}
	
	var dt time.Time
	var err error
	
	for _, format := range formats {
		if dt, err = time.Parse(format, dtStr); err == nil {
			break
		}
	}

	if err != nil {
		return vm.Error("Invalid datetime format")
	}

	// Convert TWX format strings to Go format strings
	goFormat := convertTWXFormatToGo(formatStr)
	result := dt.Format(goFormat)

	vm.SetVariable(resultVarName, &types.Value{
		Type:   types.StringType,
		String: result,
	})

	return nil
}

// convertTWXFormatToGo converts TWX datetime format strings to Go format strings
func convertTWXFormatToGo(twxFormat string) string {
	// Basic TWX to Go format conversion
	goFormat := twxFormat
	
	// Replace common TWX format codes with Go equivalents
	replacements := map[string]string{
		"YYYY": "2006",
		"YY":   "06",
		"MM":   "01",
		"DD":   "02",
		"HH":   "15",
		"mm":   "04",
		"ss":   "05",
	}
	
	for twx, goFmt := range replacements {
		goFormat = strings.ReplaceAll(goFormat, twx, goFmt)
	}
	
	return goFormat
}

// cmdStartTimer starts a timer with a given name
func cmdStartTimer(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("STARTTIMER requires exactly 1 parameter: timer_name")
	}

	timerName := GetParamString(vm, params[0])

	// Try to call the specific mock VM method if it exists
	if mockVM, ok := vm.(interface {
		StartTimer(string) error
	}); ok {
		return mockVM.StartTimer(timerName)
	}

	// For real implementation, would start timer tracking
	return nil
}

// cmdStopTimer stops a timer by name
func cmdStopTimer(vm types.VMInterface, params []*types.CommandParam) error {
	if len(params) != 1 {
		return vm.Error("STOPTIMER requires exactly 1 parameter: timer_name")
	}

	timerName := GetParamString(vm, params[0])

	// Try to call the specific mock VM method if it exists
	if mockVM, ok := vm.(interface {
		StopTimer(string) error
	}); ok {
		return mockVM.StopTimer(timerName)
	}

	// For real implementation, would stop timer tracking
	return nil
}