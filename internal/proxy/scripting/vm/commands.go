package vm

import (
	"twist/internal/proxy/scripting/types"
	"twist/internal/proxy/scripting/vm/commands"
)

// registerCommands registers all TWX script commands
func (vm *VirtualMachine) registerCommands() {
	commands.RegisterAllCommands(vm)
}

// RegisterCommand implements the CommandRegistry interface
func (vm *VirtualMachine) RegisterCommand(name string, minParams, maxParams int, paramTypes []types.ParameterType, handler types.CommandHandler) {
	vm.registerCommand(name, minParams, maxParams, paramTypes, handler)
}
