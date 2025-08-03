package theme

import (
	"github.com/gdamore/tcell/v2"
	"twist/internal/components"
)

// Standard ANSI 16-color palette using correct hex values
// This ensures consistent colors regardless of terminal color scheme
var (
	// Basic 8 colors (0-7)
	DOSBlack     = tcell.NewHexColor(0x000000)  // 0: Black
	DOSRed       = tcell.NewHexColor(0x800000)  // 1: Red (Dark Red)
	DOSGreen     = tcell.NewHexColor(0x008000)  // 2: Green (Dark Green)
	DOSBrown     = tcell.NewHexColor(0x808000)  // 3: Yellow/Brown (Dark Yellow)
	DOSBlue      = tcell.NewHexColor(0x000080)  // 4: Blue (Dark Blue)
	DOSMagenta   = tcell.NewHexColor(0x800080)  // 5: Magenta (Dark Magenta)
	DOSCyan      = tcell.NewHexColor(0x008080)  // 6: Cyan (Dark Cyan)
	DOSLightGray = tcell.NewHexColor(0xC0C0C0)  // 7: White/Light Gray
	
	// Bright 8 colors (8-15)
	DOSDarkGray     = tcell.NewHexColor(0x808080)  // 8: Gray (Dark Gray)
	DOSLightRed     = tcell.NewHexColor(0xFF0000)  // 9: Bright Red
	DOSLightGreen   = tcell.NewHexColor(0x00FF00)  // 10: Bright Green
	DOSYellow       = tcell.NewHexColor(0xFFFF00)  // 11: Bright Yellow
	DOSLightBlue    = tcell.NewHexColor(0x0000FF)  // 12: Bright Blue
	DOSLightMagenta = tcell.NewHexColor(0xFF00FF)  // 13: Bright Magenta
	DOSLightCyan    = tcell.NewHexColor(0x00FFFF)  // 14: Bright Cyan
	DOSWhite        = tcell.NewHexColor(0xFFFFFF)  // 15: Bright White
)

// TelixTheme implements the classic Telix DOS terminal theme
type TelixTheme struct{}

// NewTelixTheme creates a new Telix theme instance
func NewTelixTheme() *TelixTheme {
	return &TelixTheme{}
}

// Name returns the theme name
func (t *TelixTheme) Name() string {
	return "telix"
}

// DefaultColors returns the default color scheme
func (t *TelixTheme) DefaultColors() DefaultColors {
	return DefaultColors{
		Background: DOSBlack,
		Foreground: DOSLightGray,
		Waiting:    DOSDarkGray,       // Dark gray for waiting messages
	}
}

// DialogColors returns the dialog color scheme
func (t *TelixTheme) DialogColors() DialogColors {
	return DialogColors{
		Background:  DOSBlue,                            // Main dialog background
		Foreground:  DOSWhite,                           // Text and labels
		Border:      DOSWhite,                           // Border lines
		Title:       DOSWhite,                           // Dialog title
		SelectedBg:  DOSWhite,                           // Selected item background
		SelectedFg:  DOSBlack,                           // Selected item text
		ButtonBg:    DOSLightGray,                       // Button background (light gray)
		ButtonFg:    DOSBlack,                           // Button text (black)
		FieldBg:     tcell.NewHexColor(0x000040),        // Input field background (darker blue)
		FieldFg:     DOSWhite,                           // Input field text (white)
	}
}

// MenuColors returns the menu color scheme
func (t *TelixTheme) MenuColors() MenuColors {
	return MenuColors{
		Background:  DOSBlue,        // Blue background for dropdown
		Foreground:  DOSLightGray,   // Light gray text for normal items
		SelectedBg:  DOSRed,         // Red background for selected items (like reference)
		SelectedFg:  DOSWhite,       // White text on red background
		DisabledFg:  DOSDarkGray,    // Dark gray for disabled items
		Separator:   DOSWhite,       // White separator lines
	}
}

// TerminalColors returns the terminal color scheme
func (t *TelixTheme) TerminalColors() TerminalColors {
	return TerminalColors{
		Background: DOSBlack,
		Foreground: DOSLightGray,      // Light gray text (standard DOS terminal)
		Border:     DOSLightGray,      // Light gray borders
		ScrollBar:  DOSDarkGray,
	}
}

// StatusColors returns the status bar color scheme (menu bar)
func (t *TelixTheme) StatusColors() StatusColors {
	return StatusColors{
		Background:     DOSBlue,        // Use blue #000080 for menu bar background
		Foreground:     DOSLightGray,   // Light gray text on blue background (normal items)
		HighlightBg:    DOSRed,         // Red background for selected menu items (like reference)
		HighlightFg:    DOSWhite,       // White text on red background
		ErrorBg:        DOSRed,
		ErrorFg:        DOSWhite,
		ConnectedFg:    DOSLightGreen,  // Bright green for connected status
		ConnectingFg:   DOSYellow,      // Yellow for connecting status
		DisconnectedFg: DOSLightRed,    // Bright red for disconnected status
	}
}

// PanelColors returns the panel color scheme
func (t *TelixTheme) PanelColors() PanelColors {
	return PanelColors{
		Background: DOSBlack,
		Foreground: DOSLightGray,
		Border:     DOSLightGray,      // Light gray borders (not cyan)
		Title:      DOSLightGray,      // Light gray titles (not cyan)
		HeaderBg:   DOSBlack,          // Black header background (not cyan)
		HeaderFg:   DOSLightGray,      // Light gray text on black header
	}
}

// BorderStyle returns the border styling
func (t *TelixTheme) BorderStyle() BorderStyle {
	return BorderStyle{
		Color:      DOSLightGray,          // Light gray borders
		TitleColor: DOSLightGray,          // Light gray titles
		Padding:    0,
	}
}

// MenuBorderStyle returns the border style for menus
// Using single-line box drawing characters
func (t *TelixTheme) MenuBorderStyle() components.MenuBorderStyle {
	return components.MenuBorderStyleSingle
}

// ANSIColorPalette returns the 16-color ANSI palette for this theme
func (t *TelixTheme) ANSIColorPalette() [16]tcell.Color {
	return [16]tcell.Color{
		DOSBlack,        // 0: Black
		DOSRed,          // 1: Red (Dark Red)
		DOSGreen,        // 2: Green (Dark Green)
		DOSBrown,        // 3: Yellow/Brown (Dark Yellow)
		DOSBlue,         // 4: Blue (Dark Blue)
		DOSMagenta,      // 5: Magenta (Dark Magenta)
		DOSCyan,         // 6: Cyan (Dark Cyan)
		DOSLightGray,    // 7: White/Light Gray
		DOSDarkGray,     // 8: Gray (Dark Gray)
		DOSLightRed,     // 9: Bright Red
		DOSLightGreen,   // 10: Bright Green
		DOSYellow,       // 11: Bright Yellow
		DOSLightBlue,    // 12: Bright Blue
		DOSLightMagenta, // 13: Bright Magenta
		DOSLightCyan,    // 14: Bright Cyan
		DOSWhite,        // 15: Bright White
	}
}

