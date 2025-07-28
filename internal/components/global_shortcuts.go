package components

import (
	"strings"
	"sync"
	"github.com/gdamore/tcell/v2"
)

// GlobalShortcutManager manages application-wide keyboard shortcuts
// This allows menu items to register shortcuts that work globally, not just when menus are open
type GlobalShortcutManager struct {
	shortcuts map[string]func() // map of shortcut string to callback function
	mutex     sync.RWMutex      // protect concurrent access
}

// NewGlobalShortcutManager creates a new global shortcut manager
func NewGlobalShortcutManager() *GlobalShortcutManager {
	return &GlobalShortcutManager{
		shortcuts: make(map[string]func()),
	}
}

// RegisterShortcut registers a global shortcut with its callback
func (gsm *GlobalShortcutManager) RegisterShortcut(shortcut string, callback func()) {
	if shortcut == "" {
		return
	}
	
	gsm.mutex.Lock()
	defer gsm.mutex.Unlock()
	
	gsm.shortcuts[normalizeShortcut(shortcut)] = callback
}

// UnregisterShortcut removes a global shortcut
func (gsm *GlobalShortcutManager) UnregisterShortcut(shortcut string) {
	if shortcut == "" {
		return
	}
	
	gsm.mutex.Lock()
	defer gsm.mutex.Unlock()
	
	delete(gsm.shortcuts, normalizeShortcut(shortcut))
}

// HandleKeyEvent checks if a key event matches any registered global shortcuts
func (gsm *GlobalShortcutManager) HandleKeyEvent(event *tcell.EventKey) bool {
	shortcutString := keyEventToString(event)
	if shortcutString == "" {
		return false
	}
	
	gsm.mutex.RLock()
	callback, exists := gsm.shortcuts[normalizeShortcut(shortcutString)]
	gsm.mutex.RUnlock()
	
	if exists {
		callback()
		return true // Event was handled
	}
	return false // Event not handled
}

// ListRegisteredShortcuts returns all currently registered shortcuts (for debugging)
func (gsm *GlobalShortcutManager) ListRegisteredShortcuts() []string {
	gsm.mutex.RLock()
	defer gsm.mutex.RUnlock()
	
	shortcuts := make([]string, 0, len(gsm.shortcuts))
	for shortcut := range gsm.shortcuts {
		shortcuts = append(shortcuts, shortcut)
	}
	return shortcuts
}

// normalizeShortcut converts a shortcut string to a consistent format
func normalizeShortcut(shortcut string) string {
	// Convert to lowercase and trim spaces
	return strings.ToLower(strings.TrimSpace(shortcut))
}