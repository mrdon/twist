package menu

import (
	"testing"
)

func TestPhase5ScriptMenuFunctionality(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Set up a mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})
	
	// Test AddScriptMenu functionality
	err := manager.AddScriptMenu("TestMenu", "Test Menu Description", "MAIN", "test_ref", "Enter test value:", "test_script", 'M', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	// Verify the menu was created
	if len(manager.scriptMenus) != 1 {
		t.Errorf("Expected 1 script menu, got %d", len(manager.scriptMenus))
	}
	
	menu, exists := manager.scriptMenus["TestMenu"]
	if !exists {
		t.Fatal("TestMenu should exist in scriptMenus")
	}
	
	if menu.Name != "TestMenu" {
		t.Errorf("Expected menu name 'TestMenu', got '%s'", menu.Name)
	}
	
	if menu.Description != "Test Menu Description" {
		t.Errorf("Expected menu description 'Test Menu Description', got '%s'", menu.Description)
	}
	
	if menu.Hotkey != 'M' {
		t.Errorf("Expected hotkey 'M', got '%c'", menu.Hotkey)
	}
	
	if menu.Reference != "test_ref" {
		t.Errorf("Expected reference 'test_ref', got '%s'", menu.Reference)
	}
	
	if menu.ScriptOwner != "test_script" {
		t.Errorf("Expected script owner 'test_script', got '%s'", menu.ScriptOwner)
	}
}

func TestPhase5ScriptMenuValues(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Add a script menu
	err := manager.AddScriptMenu("ValueMenu", "Value Menu", "MAIN", "value_ref", "", "test_script", 'V', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	// Test SetScriptMenuValue
	err = manager.SetScriptMenuValue("ValueMenu", "test_value")
	if err != nil {
		t.Errorf("SetScriptMenuValue failed: %v", err)
	}
	
	// Test GetScriptMenuValue
	value, err := manager.GetScriptMenuValue("ValueMenu")
	if err != nil {
		t.Errorf("GetScriptMenuValue failed: %v", err)
	}
	
	if value != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", value)
	}
}

func TestPhase5ScriptMenuHelp(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Add a script menu
	err := manager.AddScriptMenu("HelpMenu", "Help Menu", "MAIN", "help_ref", "", "test_script", 'H', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	// Test SetScriptMenuHelp
	helpText := "This is test help text"
	err = manager.SetScriptMenuHelp("HelpMenu", helpText)
	if err != nil {
		t.Errorf("SetScriptMenuHelp failed: %v", err)
	}
	
	// Verify help was set
	menu, exists := manager.scriptMenus["HelpMenu"]
	if !exists {
		t.Fatal("HelpMenu should exist")
	}
	
	if menu.Help != helpText {
		t.Errorf("Expected help text '%s', got '%s'", helpText, menu.Help)
	}
}

func TestPhase5ScriptMenuOptions(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Add a script menu
	err := manager.AddScriptMenu("OptionsMenu", "Options Menu", "MAIN", "options_ref", "", "test_script", 'O', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	// Test SetScriptMenuOptions
	options := "Option1,Option2,Option3"
	err = manager.SetScriptMenuOptions("OptionsMenu", options)
	if err != nil {
		t.Errorf("SetScriptMenuOptions failed: %v", err)
	}
	
	// Verify options were set
	menu, exists := manager.scriptMenus["OptionsMenu"]
	if !exists {
		t.Fatal("OptionsMenu should exist")
	}
	
	if menu.Options != options {
		t.Errorf("Expected options '%s', got '%s'", options, menu.Options)
	}
}

func TestPhase5ScriptMenuCleanup(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Add multiple script menus with same owner
	err := manager.AddScriptMenu("Menu1", "Menu 1", "MAIN", "ref1", "", "script_123", 'A', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	err = manager.AddScriptMenu("Menu2", "Menu 2", "MAIN", "ref2", "", "script_123", 'B', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	err = manager.AddScriptMenu("Menu3", "Menu 3", "MAIN", "ref3", "", "other_script", 'C', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	// Verify all menus exist
	if len(manager.scriptMenus) != 3 {
		t.Errorf("Expected 3 script menus, got %d", len(manager.scriptMenus))
	}
	
	// Remove menus by owner
	manager.RemoveScriptMenusByOwner("script_123")
	
	// Verify only the other script's menu remains
	if len(manager.scriptMenus) != 1 {
		t.Errorf("Expected 1 script menu after cleanup, got %d", len(manager.scriptMenus))
	}
	
	_, exists := manager.scriptMenus["Menu3"]
	if !exists {
		t.Error("Menu3 should still exist after cleanup")
	}
	
	_, exists = manager.scriptMenus["Menu1"]
	if exists {
		t.Error("Menu1 should be removed after cleanup")
	}
	
	_, exists = manager.scriptMenus["Menu2"]
	if exists {
		t.Error("Menu2 should be removed after cleanup")
	}
}

func TestPhase5ScriptMenuOpenClose(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Set up a mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})
	
	// Add a script menu
	err := manager.AddScriptMenu("OpenCloseMenu", "Open Close Menu", "MAIN", "oc_ref", "", "test_script", 'O', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	// Test OpenScriptMenu
	err = manager.OpenScriptMenu("OpenCloseMenu")
	if err != nil {
		t.Errorf("OpenScriptMenu failed: %v", err)
	}
	
	// Verify menu is active
	if !manager.IsActive() {
		t.Error("Menu should be active after OpenScriptMenu")
	}
	
	// Test CloseScriptMenu
	err = manager.CloseScriptMenu("OpenCloseMenu")
	if err != nil {
		t.Errorf("CloseScriptMenu failed: %v", err)
	}
	
	// Since there was no parent, the menu should be deactivated
	if manager.IsActive() {
		t.Error("Menu should not be active after CloseScriptMenu with no parent")
	}
}

func TestPhase5MenuKeyConfiguration(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Test default menu key
	if manager.GetMenuKey() != '$' {
		t.Errorf("Expected default menu key '$', got '%c'", manager.GetMenuKey())
	}
	
	// Test SetMenuKey
	manager.SetMenuKey('#')
	if manager.GetMenuKey() != '#' {
		t.Errorf("Expected menu key '#', got '%c'", manager.GetMenuKey())
	}
	
	// Test menu activation with new key
	consumed := manager.ProcessMenuKey("#")
	if !consumed {
		t.Error("New menu key should be processed and consumed")
	}
	
	// Test old key doesn't work
	consumed = manager.ProcessMenuKey("$")
	if consumed {
		t.Error("Old menu key should not be processed")
	}
}

func TestPhase5TwoStageInputCollection(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Set up a mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})
	
	// Add a script menu with prompt
	err := manager.AddScriptMenu("PromptMenu", "Prompt Menu", "MAIN", "prompt_ref", "Enter your choice:", "test_script", 'P', false)
	if err != nil {
		t.Errorf("AddScriptMenu failed: %v", err)
	}
	
	// Verify the menu was created with the prompt
	menu, exists := manager.scriptMenus["PromptMenu"]
	if !exists {
		t.Fatal("PromptMenu should exist")
	}
	
	if menu.Prompt != "Enter your choice:" {
		t.Errorf("Expected prompt 'Enter your choice:', got '%s'", menu.Prompt)
	}
	
	// This test verifies the infrastructure is in place for two-stage input collection
	// The actual implementation may vary based on how menu items are activated
}