package commands

import (
	"math"
	"math/rand"
	"time"
	"twist/internal/proxy/scripting/types"
)


// RegisterMathCommands registers all mathematical commands
func RegisterMathCommands(vm CommandRegistry) {
	vm.RegisterCommand("ADD", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdAdd)
	vm.RegisterCommand("SUBTRACT", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdSubtract)
	vm.RegisterCommand("MULTIPLY", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdMultiply)
	vm.RegisterCommand("DIVIDE", 2, 2, []types.ParameterType{types.ParamVar, types.ParamValue}, cmdDivide)
	vm.RegisterCommand("MOD", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdMod)
	vm.RegisterCommand("RANDOM", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdRandom)
	vm.RegisterCommand("ABS", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdAbs)
	vm.RegisterCommand("INT", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdInt)
	vm.RegisterCommand("ROUND", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdRound)
	vm.RegisterCommand("SQR", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdSqr)
	vm.RegisterCommand("POWER", 3, 3, []types.ParameterType{types.ParamValue, types.ParamValue, types.ParamVar}, cmdPower)
	vm.RegisterCommand("SIN", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdSin)
	vm.RegisterCommand("COS", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdCos)
	vm.RegisterCommand("TAN", 2, 2, []types.ParameterType{types.ParamValue, types.ParamVar}, cmdTan)
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func cmdAdd(vm types.VMInterface, params []*types.CommandParam) error {
	// TWX-style in-place modification: ADD $var value
	num1 := GetParamNumber(vm, params[0])
	num2 := GetParamNumber(vm, params[1])
	result := &types.Value{
		Type:   types.NumberType,
		Number: num1 + num2,
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}

func cmdSubtract(vm types.VMInterface, params []*types.CommandParam) error {
	// TWX-style in-place modification: SUBTRACT $var value
	num1 := GetParamNumber(vm, params[0])
	num2 := GetParamNumber(vm, params[1])
	result := &types.Value{
		Type:   types.NumberType,
		Number: num1 - num2,
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}

func cmdMultiply(vm types.VMInterface, params []*types.CommandParam) error {
	// TWX-style in-place modification: MULTIPLY $var value
	num1 := GetParamNumber(vm, params[0])
	num2 := GetParamNumber(vm, params[1])
	result := &types.Value{
		Type:   types.NumberType,
		Number: num1 * num2,
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}

func cmdDivide(vm types.VMInterface, params []*types.CommandParam) error {
	// TWX-style in-place modification: DIVIDE $var value
	num1 := GetParamNumber(vm, params[0])
	divisor := GetParamNumber(vm, params[1])
	if divisor == 0 {
		return vm.Error("Division by zero")
	}
	result := &types.Value{
		Type:   types.NumberType,
		Number: num1 / divisor,
	}
	vm.SetVariable(params[0].VarName, result)
	return nil
}

func cmdMod(vm types.VMInterface, params []*types.CommandParam) error {
	num1 := GetParamNumber(vm, params[0])
	divisor := GetParamNumber(vm, params[1])
	if divisor == 0 {
		return vm.Error("Division by zero")
	}
	
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Mod(num1, divisor),
	}
	vm.SetVariable(params[2].VarName, result)
	return nil
}

func cmdRandom(vm types.VMInterface, params []*types.CommandParam) error {
	max := int(GetParamNumber(vm, params[0]))
	if max <= 0 {
		max = 1
	}
	
	randomValue := rng.Intn(max) + 1 // TWX random is 1-based
	result := &types.Value{
		Type:        types.NumberType,
		Number: float64(randomValue),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdAbs(vm types.VMInterface, params []*types.CommandParam) error {
	num := GetParamNumber(vm, params[0])
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Abs(num),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdInt(vm types.VMInterface, params []*types.CommandParam) error {
	num := GetParamNumber(vm, params[0])
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Trunc(num),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdRound(vm types.VMInterface, params []*types.CommandParam) error {
	num := GetParamNumber(vm, params[0])
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Round(num),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdSqr(vm types.VMInterface, params []*types.CommandParam) error {
	value := GetParamNumber(vm, params[0])
	if value < 0 {
		return vm.Error("Square root of negative number")
	}
	
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Sqrt(value),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdPower(vm types.VMInterface, params []*types.CommandParam) error {
	base := GetParamNumber(vm, params[0])
	exponent := GetParamNumber(vm, params[1])
	
	// Handle special cases that might cause errors
	if base == 0 && exponent < 0 {
		return vm.Error("Cannot raise zero to negative power")
	}
	if base < 0 && exponent != math.Trunc(exponent) {
		return vm.Error("Cannot raise negative number to fractional power")
	}
	
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Pow(base, exponent),
	}
	vm.SetVariable(params[2].VarName, result)
	return nil
}

func cmdSin(vm types.VMInterface, params []*types.CommandParam) error {
	// Convert degrees to radians for TWX compatibility
	radians := GetParamNumber(vm, params[0]) * math.Pi / 180
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Sin(radians),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdCos(vm types.VMInterface, params []*types.CommandParam) error {
	// Convert degrees to radians for TWX compatibility
	radians := GetParamNumber(vm, params[0]) * math.Pi / 180
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Cos(radians),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}

func cmdTan(vm types.VMInterface, params []*types.CommandParam) error {
	// Convert degrees to radians for TWX compatibility
	radians := GetParamNumber(vm, params[0]) * math.Pi / 180
	
	// Check for values that would make tan undefined
	cos := math.Cos(radians)
	if math.Abs(cos) < 1e-15 {
		return vm.Error("Tangent undefined at this angle")
	}
	
	result := &types.Value{
		Type:        types.NumberType,
		Number: math.Tan(radians),
	}
	vm.SetVariable(params[1].VarName, result)
	return nil
}