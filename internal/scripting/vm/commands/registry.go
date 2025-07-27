package commands

// RegisterAllCommands registers all command groups with the VM
func RegisterAllCommands(vm CommandRegistry) {
	RegisterTextCommands(vm)
	RegisterMathCommands(vm)
	RegisterVariableCommands(vm)
	RegisterControlCommands(vm)
	RegisterDateTimeCommands(vm)
	registerArrayCommandsInternal(vm)
	RegisterScriptCommands(vm)
	RegisterTriggerCommands(vm)
	registerGameCommandsInternal(vm)
	RegisterSystemCommands(vm)
	RegisterFileCommands(vm)
	RegisterComparisonCommands(vm)
	RegisterMiscCommands(vm)
	RegisterNetworkCommands(vm)
}

// Internal function to avoid naming conflicts
func registerArrayCommandsInternal(vm CommandRegistry) {
	// Import from arrays.go
	RegisterArrayCommands(vm)
}

// Internal function to avoid naming conflicts  
func registerGameCommandsInternal(vm CommandRegistry) {
	// Import from game.go
	RegisterGameCommands(vm)
}

// Placeholder functions for other command groups - these would be implemented similarly





func RegisterSystemCommands(vm CommandRegistry) {
	// LOGGING, GETTEXT, GETOUTTEXT commands
}

func RegisterFileCommands(vm CommandRegistry) {
	// LOADTEXT, SAVETEXT, FILEEXISTS, etc.
}

