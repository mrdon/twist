package commands

import (
	"fmt"
	"strings"

	"twist/internal/debug"
	"twist/internal/proxy/scripting/types"
)

// RegisterMenuCommands registers all TWX menu commands
func RegisterMenuCommands(vm CommandRegistry) {
	// TWX Menu commands for script-based menu manipulation
	vm.RegisterCommand("ADDMENU", 7, 7, []types.ParameterType{
		types.ParamValue, // parent
		types.ParamValue, // name
		types.ParamValue, // description
		types.ParamValue, // hotkey
		types.ParamValue, // reference
		types.ParamValue, // prompt
		types.ParamValue, // closeMenu
	}, cmdAddMenu)

	vm.RegisterCommand("OPENMENU", 1, 1, []types.ParameterType{
		types.ParamValue, // menuName
	}, cmdOpenMenu)

	vm.RegisterCommand("CLOSEMENU", 1, 1, []types.ParameterType{
		types.ParamValue, // menuName
	}, cmdCloseMenu)

	vm.RegisterCommand("GETMENUVALUE", 2, 2, []types.ParameterType{
		types.ParamValue, // menuName
		types.ParamVar,   // result variable
	}, cmdGetMenuValue)

	vm.RegisterCommand("SETMENUVALUE", 2, 2, []types.ParameterType{
		types.ParamValue, // menuName
		types.ParamValue, // value
	}, cmdSetMenuValue)

	vm.RegisterCommand("SETMENUHELP", 2, 2, []types.ParameterType{
		types.ParamValue, // menuName
		types.ParamValue, // helpText
	}, cmdSetMenuHelp)

	vm.RegisterCommand("SETMENUOPTIONS", 2, 2, []types.ParameterType{
		types.ParamValue, // menuName
		types.ParamValue, // options
	}, cmdSetMenuOptions)

	vm.RegisterCommand("SETMENUKEY", 1, 1, []types.ParameterType{
		types.ParamValue, // newKey
	}, cmdSetMenuKey)
}

// Menu data structures for script-created menus
type ScriptMenu struct {
	Name        string
	Description string
	Hotkey      rune
	Reference   string
	Prompt      string
	CloseMenu   bool
	ScriptOwner string
	Parent      string
	Value       string // Current menu value
	Help        string // Help text
	Options     string // Menu options
}

// Use existing GetParamString from helpers.go

// Helper function to get parameter as rune (for hotkeys)
func GetParamRune(vm types.VMInterface, param *types.CommandParam) rune {
	str := GetParamString(vm, param)
	if len(str) > 0 {
		return rune(strings.ToUpper(str)[0])
	}
	return 0
}

// Helper function to get parameter as boolean
func GetParamBool(vm types.VMInterface, param *types.CommandParam) bool {
	str := strings.ToLower(GetParamString(vm, param))
	return str == "true" || str == "1" || str == "yes" || str == "on"
}

// cmdAddMenu implements the ADDMENU command
// Syntax: ADDMENU parent, name, description, hotkey, reference, prompt, closeMenu
func cmdAddMenu(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdAddMenu: %v", r)
		}
	}()

	if len(params) != 7 {
		return fmt.Errorf("ADDMENU requires 7 parameters")
	}

	parent := GetParamString(vm, params[0])
	name := GetParamString(vm, params[1])
	description := GetParamString(vm, params[2])
	hotkey := GetParamRune(vm, params[3])
	reference := GetParamString(vm, params[4])
	prompt := GetParamString(vm, params[5])
	closeMenu := GetParamBool(vm, params[6])

	// Get the script ID to track menu ownership
	scriptID := ""
	if script := vm.GetCurrentScript(); script != nil {
		scriptID = script.GetID()
	}

	// Create menu data
	menu := &ScriptMenu{
		Name:        name,
		Description: description,
		Hotkey:      hotkey,
		Reference:   reference,
		Prompt:      prompt,
		CloseMenu:   closeMenu,
		ScriptOwner: scriptID,
		Parent:      parent,
	}

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		err := addScriptMenu(menuManager, menu)
		if err != nil {
			return fmt.Errorf("ADDMENU failed: %v", err)
		}
	} else {
		debug.Log("Terminal menu manager not available for ADDMENU")
	}

	return nil
}

// cmdOpenMenu implements the OPENMENU command
// Syntax: OPENMENU menuName
func cmdOpenMenu(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdOpenMenu: %v", r)
		}
	}()

	if len(params) != 1 {
		return fmt.Errorf("OPENMENU requires 1 parameter")
	}

	menuName := GetParamString(vm, params[0])

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		err := openScriptMenu(menuManager, menuName)
		if err != nil {
			return fmt.Errorf("OPENMENU failed: %v", err)
		}
	} else {
		debug.Log("Terminal menu manager not available for OPENMENU")
	}

	return nil
}

// cmdCloseMenu implements the CLOSEMENU command
// Syntax: CLOSEMENU menuName
func cmdCloseMenu(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdCloseMenu: %v", r)
		}
	}()

	if len(params) != 1 {
		return fmt.Errorf("CLOSEMENU requires 1 parameter")
	}

	menuName := GetParamString(vm, params[0])

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		err := closeScriptMenu(menuManager, menuName)
		if err != nil {
			return fmt.Errorf("CLOSEMENU failed: %v", err)
		}
	} else {
		debug.Log("Terminal menu manager not available for CLOSEMENU")
	}

	return nil
}

// cmdGetMenuValue implements the GETMENUVALUE command
// Syntax: GETMENUVALUE menuName, resultVar
func cmdGetMenuValue(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdGetMenuValue: %v", r)
		}
	}()

	if len(params) != 2 {
		return fmt.Errorf("GETMENUVALUE requires 2 parameters")
	}

	menuName := GetParamString(vm, params[0])
	resultVar := params[1].VarName

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		value, err := getScriptMenuValue(menuManager, menuName)
		if err != nil {
			return fmt.Errorf("GETMENUVALUE failed: %v", err)
		}

		// Set the result in the variable
		vm.SetVariable(resultVar, &types.Value{
			Type:   types.StringType,
			String: value,
		})
	} else {
		// Set empty value if manager not available
		vm.SetVariable(resultVar, &types.Value{
			Type:   types.StringType,
			String: "",
		})
	}

	return nil
}

// cmdSetMenuValue implements the SETMENUVALUE command
// Syntax: SETMENUVALUE menuName, value
func cmdSetMenuValue(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdSetMenuValue: %v", r)
		}
	}()

	if len(params) != 2 {
		return fmt.Errorf("SETMENUVALUE requires 2 parameters")
	}

	menuName := GetParamString(vm, params[0])
	value := GetParamString(vm, params[1])

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		err := setScriptMenuValue(menuManager, menuName, value)
		if err != nil {
			return fmt.Errorf("SETMENUVALUE failed: %v", err)
		}
	} else {
		debug.Log("Terminal menu manager not available for SETMENUVALUE")
	}

	return nil
}

// cmdSetMenuHelp implements the SETMENUHELP command
// Syntax: SETMENUHELP menuName, helpText
func cmdSetMenuHelp(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdSetMenuHelp: %v", r)
		}
	}()

	if len(params) != 2 {
		return fmt.Errorf("SETMENUHELP requires 2 parameters")
	}

	menuName := GetParamString(vm, params[0])
	helpText := GetParamString(vm, params[1])

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		err := setScriptMenuHelp(menuManager, menuName, helpText)
		if err != nil {
			return fmt.Errorf("SETMENUHELP failed: %v", err)
		}
	} else {
		debug.Log("Terminal menu manager not available for SETMENUHELP")
	}

	return nil
}

// cmdSetMenuOptions implements the SETMENUOPTIONS command
// Syntax: SETMENUOPTIONS menuName, options
func cmdSetMenuOptions(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdSetMenuOptions: %v", r)
		}
	}()

	if len(params) != 2 {
		return fmt.Errorf("SETMENUOPTIONS requires 2 parameters")
	}

	menuName := GetParamString(vm, params[0])
	options := GetParamString(vm, params[1])

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		err := setScriptMenuOptions(menuManager, menuName, options)
		if err != nil {
			return fmt.Errorf("SETMENUOPTIONS failed: %v", err)
		}
	} else {
		debug.Log("Terminal menu manager not available for SETMENUOPTIONS")
	}

	return nil
}

// cmdSetMenuKey implements the SETMENUKEY command
// Syntax: SETMENUKEY newKey
func cmdSetMenuKey(vm types.VMInterface, params []*types.CommandParam) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in cmdSetMenuKey: %v", r)
		}
	}()

	if len(params) != 1 {
		return fmt.Errorf("SETMENUKEY requires 1 parameter")
	}

	newKey := GetParamRune(vm, params[0])

	// Get terminal menu manager via game interface
	gameInterface := vm.GetGameInterface()
	if menuManager := getTerminalMenuManager(gameInterface); menuManager != nil {
		err := setMenuKey(menuManager, newKey)
		if err != nil {
			return fmt.Errorf("SETMENUKEY failed: %v", err)
		}
	} else {
		debug.Log("Terminal menu manager not available for SETMENUKEY")
	}

	return nil
}

// Helper functions to interact with the terminal menu manager

func getTerminalMenuManager(gameInterface types.GameInterface) interface{} {
	// Try to get the menu manager via type assertion
	if gameAdapter, ok := gameInterface.(interface {
		GetMenuManager() interface{}
	}); ok {
		return gameAdapter.GetMenuManager()
	}
	debug.Log("GameInterface does not support GetMenuManager()")
	return nil
}

func addScriptMenu(menuManager interface{}, menu *ScriptMenu) error {
	// Type assert to the actual terminal menu manager
	if tmm, ok := menuManager.(interface {
		AddScriptMenu(name, description, parent, reference, prompt, scriptOwner string, hotkey rune, closeMenu bool) error
	}); ok {
		return tmm.AddScriptMenu(
			menu.Name,
			menu.Description,
			menu.Parent,
			menu.Reference,
			menu.Prompt,
			menu.ScriptOwner,
			menu.Hotkey,
			menu.CloseMenu,
		)
	}
	debug.Log("Adding script menu: %s (hotkey: %c)", menu.Name, menu.Hotkey)
	return fmt.Errorf("menu manager interface not available")
}

func openScriptMenu(menuManager interface{}, menuName string) error {
	if tmm, ok := menuManager.(interface {
		OpenScriptMenu(string) error
	}); ok {
		return tmm.OpenScriptMenu(menuName)
	}
	debug.Log("Opening script menu: %s", menuName)
	return fmt.Errorf("menu manager interface not available")
}

func closeScriptMenu(menuManager interface{}, menuName string) error {
	if tmm, ok := menuManager.(interface {
		CloseScriptMenu(string) error
	}); ok {
		return tmm.CloseScriptMenu(menuName)
	}
	debug.Log("Closing script menu: %s", menuName)
	return fmt.Errorf("menu manager interface not available")
}

func getScriptMenuValue(menuManager interface{}, menuName string) (string, error) {
	if tmm, ok := menuManager.(interface {
		GetScriptMenuValue(string) (string, error)
	}); ok {
		return tmm.GetScriptMenuValue(menuName)
	}
	debug.Log("Getting value for menu: %s", menuName)
	return "", fmt.Errorf("menu manager interface not available")
}

func setScriptMenuValue(menuManager interface{}, menuName, value string) error {
	if tmm, ok := menuManager.(interface {
		SetScriptMenuValue(string, string) error
	}); ok {
		return tmm.SetScriptMenuValue(menuName, value)
	}
	debug.Log("Setting value for menu %s: %s", menuName, value)
	return fmt.Errorf("menu manager interface not available")
}

func setScriptMenuHelp(menuManager interface{}, menuName, helpText string) error {
	if tmm, ok := menuManager.(interface {
		SetScriptMenuHelp(string, string) error
	}); ok {
		return tmm.SetScriptMenuHelp(menuName, helpText)
	}
	debug.Log("Setting help for menu %s: %s", menuName, helpText)
	return fmt.Errorf("menu manager interface not available")
}

func setScriptMenuOptions(menuManager interface{}, menuName, options string) error {
	if tmm, ok := menuManager.(interface {
		SetScriptMenuOptions(string, string) error
	}); ok {
		return tmm.SetScriptMenuOptions(menuName, options)
	}
	debug.Log("Setting options for menu %s: %s", menuName, options)
	return fmt.Errorf("menu manager interface not available")
}

func setMenuKey(menuManager interface{}, newKey rune) error {
	if tmm, ok := menuManager.(interface {
		SetMenuKey(rune)
	}); ok {
		tmm.SetMenuKey(newKey)
		return nil
	}
	debug.Log("Setting menu key to: %c", newKey)
	return fmt.Errorf("menu manager interface not available")
}
