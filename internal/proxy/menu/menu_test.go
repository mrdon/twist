package menu

import (
	"strings"
	"testing"
)

func TestNewTerminalMenuItem(t *testing.T) {
	item := NewTerminalMenuItem("Test Menu", "Test Description", 'T')
	
	if item == nil {
		t.Fatal("NewTerminalMenuItem returned nil")
	}
	
	if item.Name != "Test Menu" {
		t.Errorf("Expected Name 'Test Menu', got '%s'", item.Name)
	}
	
	if item.Description != "Test Description" {
		t.Errorf("Expected Description 'Test Description', got '%s'", item.Description)
	}
	
	if item.Hotkey != 'T' {
		t.Errorf("Expected Hotkey 'T', got '%c'", item.Hotkey)
	}
	
	if item.Children == nil {
		t.Error("Children slice should be initialized")
	}
	
	if item.Parameters == nil {
		t.Error("Parameters slice should be initialized")
	}
}

func TestTerminalMenuItemAddChild(t *testing.T) {
	parent := NewTerminalMenuItem("Parent", "Parent Description", 'P')
	child := NewTerminalMenuItem("Child", "Child Description", 'C')
	
	parent.AddChild(child)
	
	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}
	
	if parent.Children[0] != child {
		t.Error("Child not properly added to parent")
	}
	
	if child.Parent != parent {
		t.Error("Parent not properly set on child")
	}
}

func TestTerminalMenuItemAddNilChild(t *testing.T) {
	parent := NewTerminalMenuItem("Parent", "Parent Description", 'P')
	
	// Should not panic or add nil child
	parent.AddChild(nil)
	
	if len(parent.Children) != 0 {
		t.Errorf("Expected 0 children after adding nil, got %d", len(parent.Children))
	}
}

func TestTerminalMenuItemRemoveChild(t *testing.T) {
	parent := NewTerminalMenuItem("Parent", "Parent Description", 'P')
	child1 := NewTerminalMenuItem("Child1", "Child1 Description", '1')
	child2 := NewTerminalMenuItem("Child2", "Child2 Description", '2')
	
	parent.AddChild(child1)
	parent.AddChild(child2)
	
	if len(parent.Children) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(parent.Children))
	}
	
	removed := parent.RemoveChild(child1)
	if !removed {
		t.Error("RemoveChild should return true when child is found")
	}
	
	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child after removal, got %d", len(parent.Children))
	}
	
	if parent.Children[0] != child2 {
		t.Error("Wrong child remained after removal")
	}
	
	if child1.Parent != nil {
		t.Error("Removed child should have nil parent")
	}
}

func TestTerminalMenuItemFindChildByHotkey(t *testing.T) {
	parent := NewTerminalMenuItem("Parent", "Parent Description", 'P')
	child1 := NewTerminalMenuItem("Child1", "Child1 Description", '1')
	child2 := NewTerminalMenuItem("Child2", "Child2 Description", '2')
	
	parent.AddChild(child1)
	parent.AddChild(child2)
	
	found := parent.FindChildByHotkey('1')
	if found != child1 {
		t.Error("FindChildByHotkey should return child1")
	}
	
	found = parent.FindChildByHotkey('2')
	if found != child2 {
		t.Error("FindChildByHotkey should return child2")
	}
	
	found = parent.FindChildByHotkey('3')
	if found != nil {
		t.Error("FindChildByHotkey should return nil for non-existent hotkey")
	}
}

func TestTerminalMenuItemFindChildByName(t *testing.T) {
	parent := NewTerminalMenuItem("Parent", "Parent Description", 'P')
	child1 := NewTerminalMenuItem("Child1", "Child1 Description", '1')
	child2 := NewTerminalMenuItem("Child2", "Child2 Description", '2')
	
	parent.AddChild(child1)
	parent.AddChild(child2)
	
	found := parent.FindChildByName("Child1")
	if found != child1 {
		t.Error("FindChildByName should return child1")
	}
	
	found = parent.FindChildByName("Child2")
	if found != child2 {
		t.Error("FindChildByName should return child2")
	}
	
	found = parent.FindChildByName("NonExistent")
	if found != nil {
		t.Error("FindChildByName should return nil for non-existent name")
	}
}

func TestTerminalMenuItemGetPath(t *testing.T) {
	root := NewTerminalMenuItem("Root", "Root Description", 'R')
	child := NewTerminalMenuItem("Child", "Child Description", 'C')
	grandchild := NewTerminalMenuItem("Grandchild", "Grandchild Description", 'G')
	
	root.AddChild(child)
	child.AddChild(grandchild)
	
	rootPath := root.GetPath()
	if rootPath != "Root" {
		t.Errorf("Expected root path 'Root', got '%s'", rootPath)
	}
	
	childPath := child.GetPath()
	if childPath != "Root > Child" {
		t.Errorf("Expected child path 'Root > Child', got '%s'", childPath)
	}
	
	grandchildPath := grandchild.GetPath()
	if grandchildPath != "Root > Child > Grandchild" {
		t.Errorf("Expected grandchild path 'Root > Child > Grandchild', got '%s'", grandchildPath)
	}
}

func TestTerminalMenuItemIsRoot(t *testing.T) {
	root := NewTerminalMenuItem("Root", "Root Description", 'R')
	child := NewTerminalMenuItem("Child", "Child Description", 'C')
	
	if !root.IsRoot() {
		t.Error("Root item should report IsRoot() as true")
	}
	
	root.AddChild(child)
	
	if child.IsRoot() {
		t.Error("Child item should report IsRoot() as false")
	}
}

func TestTerminalMenuItemHasChildren(t *testing.T) {
	parent := NewTerminalMenuItem("Parent", "Parent Description", 'P')
	child := NewTerminalMenuItem("Child", "Child Description", 'C')
	
	if parent.HasChildren() {
		t.Error("Empty parent should report HasChildren() as false")
	}
	
	parent.AddChild(child)
	
	if !parent.HasChildren() {
		t.Error("Parent with child should report HasChildren() as true")
	}
}

func TestNewTerminalMenuManager(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	if manager == nil {
		t.Fatal("NewTerminalMenuManager returned nil")
	}
	
	if manager.menuKey != '$' {
		t.Errorf("Expected default menu key '$', got '%c'", manager.menuKey)
	}
	
	if manager.IsActive() {
		t.Error("New manager should not be active")
	}
	
	if manager.activeMenus == nil {
		t.Error("ActiveMenus map should be initialized")
	}
}

func TestTerminalMenuManagerProcessMenuKey(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Mock inject function to capture output
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})
	
	// Test menu activation
	consumed := manager.ProcessMenuKey("$")
	if !consumed {
		t.Error("ProcessMenuKey should consume '$' input")
	}
	
	if !manager.IsActive() {
		t.Error("Manager should be active after processing menu key")
	}
	
	// Test non-menu input
	consumed = manager.ProcessMenuKey("ls")
	if consumed {
		t.Error("ProcessMenuKey should not consume non-menu input")
	}
}

func TestTerminalMenuManagerMenuText(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	// Mock inject function
	var capturedOutput []string
	manager.SetInjectDataFunc(func(data []byte) {
		capturedOutput = append(capturedOutput, string(data))
	})
	
	// Activate menu first
	manager.ProcessMenuKey("$")
	capturedOutput = nil // Clear initial activation output
	
	// Test help command
	manager.MenuText("?")
	
	found := false
	for _, output := range capturedOutput {
		if strings.Contains(output, "Menu Help") {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Help command should produce help output")
	}
	
	// Test quit command
	capturedOutput = nil
	manager.MenuText("q")
	
	if manager.IsActive() {
		t.Error("Manager should not be active after quit command")
	}
}

func TestTerminalMenuManagerAddCustomMenu(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	menu := manager.AddCustomMenu("TestMenu", nil)
	
	if menu == nil {
		t.Fatal("AddCustomMenu should return menu item")
	}
	
	if menu.Name != "TestMenu" {
		t.Errorf("Expected menu name 'TestMenu', got '%s'", menu.Name)
	}
	
	retrieved := manager.GetMenu("TestMenu")
	if retrieved != menu {
		t.Error("GetMenu should return the same menu item")
	}
}

func TestTerminalMenuManagerRemoveMenu(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	menu := manager.AddCustomMenu("TestMenu", nil)
	if menu == nil {
		t.Fatal("AddCustomMenu failed")
	}
	
	manager.RemoveMenu("TestMenu")
	
	retrieved := manager.GetMenu("TestMenu")
	if retrieved != nil {
		t.Error("Menu should be removed and not retrievable")
	}
}

func TestTerminalMenuManagerSetMenuKey(t *testing.T) {
	manager := NewTerminalMenuManager()
	
	manager.SetMenuKey('#')
	
	if manager.GetMenuKey() != '#' {
		t.Errorf("Expected menu key '#', got '%c'", manager.GetMenuKey())
	}
	
	// Test that new key works
	consumed := manager.ProcessMenuKey("#")
	if !consumed {
		t.Error("ProcessMenuKey should consume new menu key")
	}
}

func TestTerminalMenuItemExecute(t *testing.T) {
	executed := false
	var receivedItem *TerminalMenuItem
	var receivedParams []string
	
	handler := func(item *TerminalMenuItem, params []string) error {
		executed = true
		receivedItem = item
		receivedParams = params
		return nil
	}
	
	item := NewTerminalMenuItem("Test", "Test Description", 'T')
	item.Handler = handler
	
	testParams := []string{"param1", "param2"}
	item.Parameters = testParams
	
	err := item.Execute(testParams)
	
	if err != nil {
		t.Errorf("Execute should not return error, got: %v", err)
	}
	
	if !executed {
		t.Error("Handler should have been executed")
	}
	
	if receivedItem != item {
		t.Error("Handler should receive the correct item")
	}
	
	if len(receivedParams) != len(testParams) {
		t.Error("Handler should receive the correct parameters")
	}
}

func TestTerminalMenuItemClone(t *testing.T) {
	original := NewTerminalMenuItem("Original", "Original Description", 'O')
	original.Reference = "ref1"
	original.Prompt = "Enter value:"
	original.CloseMenu = true
	original.ScriptOwner = "script1"
	original.Parameters = []string{"param1", "param2"}
	
	child := NewTerminalMenuItem("Child", "Child Description", 'C')
	original.AddChild(child)
	
	clone := original.Clone()
	
	if clone == nil {
		t.Fatal("Clone returned nil")
	}
	
	// Test that properties are copied
	if clone.Name != original.Name {
		t.Error("Name not properly cloned")
	}
	
	if clone.Description != original.Description {
		t.Error("Description not properly cloned")
	}
	
	if clone.Hotkey != original.Hotkey {
		t.Error("Hotkey not properly cloned")
	}
	
	if clone.Reference != original.Reference {
		t.Error("Reference not properly cloned")
	}
	
	if clone.Prompt != original.Prompt {
		t.Error("Prompt not properly cloned")
	}
	
	if clone.CloseMenu != original.CloseMenu {
		t.Error("CloseMenu not properly cloned")
	}
	
	if clone.ScriptOwner != original.ScriptOwner {
		t.Error("ScriptOwner not properly cloned")
	}
	
	// Test that slices are copied, not shared
	if len(clone.Parameters) != len(original.Parameters) {
		t.Error("Parameters not properly cloned")
	}
	
	if len(clone.Children) != len(original.Children) {
		t.Error("Children not properly cloned")
	}
	
	// Modify original to ensure independence
	original.Parameters[0] = "modified"
	if clone.Parameters[0] == "modified" {
		t.Error("Parameters slice should be independent")
	}
	
	// Test that parent is not cloned (should be nil)
	if clone.Parent != nil {
		t.Error("Clone should not have a parent")
	}
}