package commands

import (
	"twist/internal/log"
	"twist/internal/proxy/scripting/types"
)

// RegisterControlCommands registers all control flow commands
func RegisterControlCommands(vm CommandRegistry) {
	vm.RegisterCommand("GOTO", 1, 1, []types.ParameterType{types.ParamValue}, cmdGoto)
	vm.RegisterCommand("GOSUB", 1, 1, []types.ParameterType{types.ParamValue}, cmdGosub)
	vm.RegisterCommand("RETURN", 0, 0, []types.ParameterType{}, cmdReturn)
}

func cmdGoto(vm types.VMInterface, params []*types.CommandParam) error {
	label := GetParamString(vm, params[0])
	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	log.Info("GOTO command: jumping to label", "script", scriptName, "line", vm.GetCurrentLine(), "label", label)
	return vm.Goto(label)
}

func cmdGosub(vm types.VMInterface, params []*types.CommandParam) error {
	label := GetParamString(vm, params[0])
	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	log.Info("GOSUB command: calling subroutine", "script", scriptName, "line", vm.GetCurrentLine(), "label", label)
	return vm.Gosub(label)
}

func cmdReturn(vm types.VMInterface, params []*types.CommandParam) error {
	scriptName := "unknown"
	if script := vm.GetCurrentScript(); script != nil {
		scriptName = script.GetName()
	}
	log.Info("RETURN command: returning from subroutine", "script", scriptName, "line", vm.GetCurrentLine())
	return vm.Return()
}
