package menu

import (
	"strings"
	"testing"
)

func TestTWXMainMenuCreation(t *testing.T) {
	manager := NewTerminalMenuManager()

	// Set up a mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})

	// Activate the main menu
	err := manager.ActivateMainMenu()
	if err != nil {
		t.Fatalf("ActivateMainMenu failed: %v", err)
	}

	// Verify menu is active
	if !manager.IsActive() {
		t.Error("Menu should be active after activation")
	}

	// Verify main menu exists
	mainMenu := manager.GetMenu(TWX_MAIN)
	if mainMenu == nil {
		t.Fatal("TWX_MAIN menu should exist after activation")
	}

	// Verify menu structure
	expectedItems := map[rune]string{
		'B': "Burst Commands",
		'L': "Load Script",
		'T': "Terminate Script",
		'S': "Script Menu",
		'V': "View Data Menu",
		'P': "Port Menu",
	}

	for hotkey, expectedName := range expectedItems {
		child := mainMenu.FindChildByHotkey(hotkey)
		if child == nil {
			t.Errorf("Main menu should have child with hotkey '%c'", hotkey)
			continue
		}

		if child.Name != expectedName {
			t.Errorf("Menu item with hotkey '%c' should be named '%s', got '%s'",
				hotkey, expectedName, child.Name)
		}

		if child.Handler == nil {
			t.Errorf("Menu item '%s' should have a handler", child.Name)
		}
	}

	// Verify output was generated
	if len(capturedOutput) == 0 {
		t.Error("Menu activation should generate output")
	}

	// Check that output contains ANSI formatting
	combinedOutput := strings.Join(capturedOutput, "")
	if !strings.Contains(combinedOutput, "\x1b[") {
		t.Error("Menu output should contain ANSI escape sequences")
	}

	// Check for menu title
	if !strings.Contains(combinedOutput, "TWX Main Menu") {
		t.Error("Menu output should contain title")
	}
}

func TestTWXMainMenuHandlers(t *testing.T) {
	manager := NewTerminalMenuManager()

	// Set up a mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})

	// Test each menu handler independently
	testCases := []struct {
		input       string
		expectedMsg string
		description string
	}{
		{"B", "TWX Burst Menu", "Navigate to Burst Commands submenu"},
		{"L", "Error: No proxy interface available", "Load Script from main menu (no proxy)"},
		{"T", "Error: No proxy interface available", "Terminate Script from main menu (no proxy)"},
		{"S", "TWX Script Menu", "Navigate to Script submenu"},
		{"V", "TWX Data Menu", "Navigate to Data submenu"},
		{"P", "TWX Port Menu", "Port Menu"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Activate fresh main menu for each test
			manager.ActivateMainMenu()
			capturedOutput = nil // Clear activation output

			// Send menu input
			err := manager.MenuText(tc.input)
			if err != nil {
				t.Errorf("MenuText failed for input '%s': %v", tc.input, err)
				return
			}

			// Check that we got expected response
			combinedOutput := strings.Join(capturedOutput, "")
			if !strings.Contains(combinedOutput, tc.expectedMsg) {
				t.Errorf("Expected output containing '%s' for input '%s', got: %s",
					tc.expectedMsg, tc.input, combinedOutput)
			}
		})
	}
}

func TestTWXMainMenuHelp(t *testing.T) {
	manager := NewTerminalMenuManager()

	// Set up a mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})

	// Activate the main menu
	manager.ActivateMainMenu()

	// Clear output and test help
	capturedOutput = nil
	err := manager.MenuText("?")
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	// Check help output
	combinedOutput := strings.Join(capturedOutput, "")

	expectedHelpContent := []string{
		"TWX Main Menu",
		"Burst Commands",
		"Load Script",
		"Script Menu",
		"Data Menu",
	}

	for _, content := range expectedHelpContent {
		if !strings.Contains(combinedOutput, content) {
			t.Errorf("Help output should contain '%s', got: %s", content, combinedOutput)
		}
	}
}

func TestTWXMainMenuCategories(t *testing.T) {
	// Test that the category constants are defined
	categories := []string{
		TWX_MAIN,
		TWX_SCRIPT,
		TWX_DATA,
		TWX_PORT,
		TWX_SETUP,
		TWX_DATABASE,
	}

	expectedValues := []string{
		"Main Menu",
		"Script Menu",
		"Data Menu",
		"Port Menu",
		"Setup Menu",
		"Database Menu",
	}

	for i, category := range categories {
		if category != expectedValues[i] {
			t.Errorf("Category constant mismatch: expected '%s', got '%s'",
				expectedValues[i], category)
		}
	}
}

func TestTWXMainMenuANSIFormatting(t *testing.T) {
	manager := NewTerminalMenuManager()

	// Set up a mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})

	// Activate the main menu
	manager.ActivateMainMenu()

	// Verify ANSI formatting is used
	combinedOutput := strings.Join(capturedOutput, "")

	// Debug: print the actual output
	t.Logf("Actual menu output: %q", combinedOutput)

	// Should contain ANSI color codes
	if !strings.Contains(combinedOutput, "\x1b[") {
		t.Error("Menu output should contain ANSI escape sequences")
	}

	// Should contain menu descriptions (the actual text, not necessarily with parentheses)
	expectedDescriptions := []string{"Burst Commands", "Load Script", "Terminate Script", "Script Menu", "View Data Menu", "Port Menu"}
	for _, desc := range expectedDescriptions {
		if !strings.Contains(combinedOutput, desc) {
			t.Errorf("Menu output should contain description '%s'", desc)
		}
	}

	// Should contain selection prompt
	if !strings.Contains(combinedOutput, "Selection") {
		t.Error("Menu output should contain selection prompt")
	}
}
