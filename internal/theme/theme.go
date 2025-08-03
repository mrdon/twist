package theme

import (
	"fmt"
	
	"github.com/gdamore/tcell/v2"
	"twist/internal/components"
)

// DialogColors defines color scheme for dialogs and modals
type DialogColors struct {
	Background      tcell.Color
	Foreground      tcell.Color
	Border          tcell.Color
	Title           tcell.Color
	SelectedBg      tcell.Color
	SelectedFg      tcell.Color
	ButtonBg        tcell.Color
	ButtonFg        tcell.Color
	FieldBg         tcell.Color  // Input field background
	FieldFg         tcell.Color  // Input field text
}

// MenuColors defines color scheme for menus
type MenuColors struct {
	Background      tcell.Color
	Foreground      tcell.Color
	SelectedBg      tcell.Color
	SelectedFg      tcell.Color
	DisabledFg      tcell.Color
	Separator       tcell.Color
}

// TerminalColors defines color scheme for terminal display
type TerminalColors struct {
	Background      tcell.Color
	Foreground      tcell.Color
	Border          tcell.Color
	ScrollBar       tcell.Color
}

// DefaultColors defines default text colors for general use
type DefaultColors struct {
	Background      tcell.Color
	Foreground      tcell.Color
	Waiting         tcell.Color // Color for "Waiting..." messages
}

// StatusColors defines color scheme for status bars
type StatusColors struct {
	Background      tcell.Color
	Foreground      tcell.Color
	HighlightBg     tcell.Color
	HighlightFg     tcell.Color
	ErrorBg         tcell.Color
	ErrorFg         tcell.Color
	ConnectedFg     tcell.Color
	ConnectingFg    tcell.Color
	DisconnectedFg  tcell.Color
}

// PanelColors defines color scheme for side panels
type PanelColors struct {
	Background      tcell.Color
	Foreground      tcell.Color
	Border          tcell.Color
	Title           tcell.Color
	HeaderBg        tcell.Color
	HeaderFg        tcell.Color
}

// BorderStyle defines border styling options
type BorderStyle struct {
	Color       tcell.Color
	TitleColor  tcell.Color
	Padding     int
}


// Theme interface defines all theming properties
type Theme interface {
	// Name returns the theme name
	Name() string
	
	// Color schemes for different components
	DefaultColors() DefaultColors
	DialogColors() DialogColors
	MenuColors() MenuColors
	TerminalColors() TerminalColors
	StatusColors() StatusColors
	PanelColors() PanelColors
	
	// Border styling
	BorderStyle() BorderStyle
	MenuBorderStyle() components.MenuBorderStyle
	
	// ANSI color mapping - returns a 16-color palette (indices 0-15)
	ANSIColorPalette() [16]tcell.Color
}

// ThemeManager manages theme selection and application
type ThemeManager struct {
	currentTheme Theme
	themes       map[string]Theme
}

// NewThemeManager creates a new theme manager
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		themes: make(map[string]Theme),
	}
	
	// Register built-in themes
	tm.RegisterTheme(NewTelixTheme())
	
	// Set default theme
	tm.SetTheme("telix")
	
	return tm
}

// RegisterTheme registers a new theme
func (tm *ThemeManager) RegisterTheme(theme Theme) {
	tm.themes[theme.Name()] = theme
}

// SetTheme sets the current theme by name
func (tm *ThemeManager) SetTheme(name string) error {
	if theme, exists := tm.themes[name]; exists {
		tm.currentTheme = theme
		return nil
	}
	return fmt.Errorf("theme '%s' not found", name)
}

// Current returns the current theme
func (tm *ThemeManager) Current() Theme {
	return tm.currentTheme
}

// Available returns list of available theme names
func (tm *ThemeManager) Available() []string {
	names := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

// Global theme manager instance
var defaultThemeManager = NewThemeManager()

// GetThemeManager returns the global theme manager
func GetThemeManager() *ThemeManager {
	return defaultThemeManager
}

// Current returns the current theme from the global manager
func Current() Theme {
	return defaultThemeManager.Current()
}