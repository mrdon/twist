package components

import (
	"strings"
	"github.com/gdamore/tcell/v2"
)

// ShortcutManager handles automatic registration and parsing of keyboard shortcuts
type ShortcutManager struct {
	shortcuts map[string]func() // map of shortcut string to callback function
}

// NewShortcutManager creates a new shortcut manager
func NewShortcutManager() *ShortcutManager {
	return &ShortcutManager{
		shortcuts: make(map[string]func()),
	}
}

// RegisterShortcut registers a shortcut with its callback
func (sm *ShortcutManager) RegisterShortcut(shortcut string, callback func()) {
	if shortcut != "" {
		sm.shortcuts[strings.ToLower(shortcut)] = callback
	}
}

// UnregisterShortcut removes a shortcut
func (sm *ShortcutManager) UnregisterShortcut(shortcut string) {
	if shortcut != "" {
		delete(sm.shortcuts, strings.ToLower(shortcut))
	}
}

// HandleKeyEvent checks if a key event matches any registered shortcuts
func (sm *ShortcutManager) HandleKeyEvent(event *tcell.EventKey) bool {
	shortcutString := keyEventToString(event)
	if callback, exists := sm.shortcuts[strings.ToLower(shortcutString)]; exists {
		callback()
		return true // Event was handled
	}
	return false // Event not handled
}

// keyEventToString converts a tcell.EventKey to a shortcut string
func keyEventToString(event *tcell.EventKey) string {
	var parts []string
	
	// Handle modifiers
	if event.Modifiers()&tcell.ModCtrl != 0 {
		parts = append(parts, "ctrl")
	}
	if event.Modifiers()&tcell.ModAlt != 0 {
		parts = append(parts, "alt")  
	}
	if event.Modifiers()&tcell.ModShift != 0 {
		parts = append(parts, "shift")
	}
	
	// Handle the key itself
	if event.Rune() != 0 {
		// Printable character
		parts = append(parts, string(event.Rune()))
	} else {
		// Special key
		switch event.Key() {
		case tcell.KeyF1:
			parts = append(parts, "f1")
		case tcell.KeyF2:
			parts = append(parts, "f2")
		case tcell.KeyF3:
			parts = append(parts, "f3")
		case tcell.KeyF4:
			parts = append(parts, "f4")
		case tcell.KeyF5:
			parts = append(parts, "f5")
		case tcell.KeyF6:
			parts = append(parts, "f6")
		case tcell.KeyF7:
			parts = append(parts, "f7")
		case tcell.KeyF8:
			parts = append(parts, "f8")
		case tcell.KeyF9:
			parts = append(parts, "f9")
		case tcell.KeyF10:
			parts = append(parts, "f10")
		case tcell.KeyF11:
			parts = append(parts, "f11")
		case tcell.KeyF12:
			parts = append(parts, "f12")
		case tcell.KeyEnter:
			parts = append(parts, "enter")
		case tcell.KeyEscape:
			parts = append(parts, "esc")
		case tcell.KeyTab:
			parts = append(parts, "tab")
		case tcell.KeyBackspace:
			parts = append(parts, "backspace")
		case tcell.KeyDelete:
			parts = append(parts, "delete")
		case tcell.KeyInsert:
			parts = append(parts, "insert")
		case tcell.KeyHome:
			parts = append(parts, "home")
		case tcell.KeyEnd:
			parts = append(parts, "end")
		case tcell.KeyPgUp:
			parts = append(parts, "pageup")
		case tcell.KeyPgDn:
			parts = append(parts, "pagedown")
		case tcell.KeyUp:
			parts = append(parts, "up")
		case tcell.KeyDown:
			parts = append(parts, "down")
		case tcell.KeyLeft:
			parts = append(parts, "left")
		case tcell.KeyRight:
			parts = append(parts, "right")
		default:
			return "" // Unknown key
		}
	}
	
	return strings.Join(parts, "+")
}

// ParseShortcut parses a shortcut string (like "Ctrl+O") and returns the constituent parts
func ParseShortcut(shortcut string) (hasCtrl, hasAlt, hasShift bool, key string) {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "ctrl":
			hasCtrl = true
		case "alt":
			hasAlt = true
		case "shift":
			hasShift = true
		default:
			key = part
		}
	}
	
	return
}