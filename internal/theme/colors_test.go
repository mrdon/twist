package theme

import (
	"testing"
)

func TestANSIColorPalette(t *testing.T) {
	// Test that theme provides proper ANSI color palette
	currentTheme := Current()
	palette := currentTheme.ANSIColorPalette()

	// Should have exactly 16 colors
	if len(palette) != 16 {
		t.Fatalf("Expected 16 colors in palette, got %d", len(palette))
	}

	// Log all colors for debugging
	t.Logf("ANSI Color Palette:")
	for i, color := range palette {
		r, g, b := color.RGB()
		t.Logf("Color %d: RGB(%d,%d,%d) = #%02X%02X%02X", i, r, g, b, r, g, b)
	}

	// Test specific colors from Telix theme
	testCases := []struct {
		index                           int
		name                            string
		expectedR, expectedG, expectedB int32
	}{
		{0, "black", 0, 0, 0},
		{1, "dark red", 128, 0, 0},
		{2, "dark green", 0, 128, 0},
		{3, "brown/dark yellow", 128, 128, 0},
		{4, "dark blue", 0, 0, 128},
		{5, "dark magenta", 128, 0, 128},
		{6, "dark cyan", 0, 128, 128},
		{7, "light gray", 192, 192, 192},
		{8, "dark gray", 128, 128, 128},
		{9, "bright red", 255, 0, 0},
		{10, "bright green", 0, 255, 0},
		{11, "bright yellow", 255, 255, 0},
		{12, "bright blue", 0, 0, 255},
		{13, "bright magenta", 255, 0, 255},
		{14, "bright cyan", 0, 255, 255},
		{15, "white", 255, 255, 255},
	}

	for _, tc := range testCases {
		r, g, b := palette[tc.index].RGB()
		if r != tc.expectedR || g != tc.expectedG || b != tc.expectedB {
			t.Errorf("Color %d (%s): expected RGB(%d,%d,%d), got RGB(%d,%d,%d)",
				tc.index, tc.name, tc.expectedR, tc.expectedG, tc.expectedB, r, g, b)
		}
	}
}

func TestPotentialMissingColors(t *testing.T) {
	currentTheme := Current()
	palette := currentTheme.ANSIColorPalette()

	// Check colors that might represent orange in ANSI art
	t.Logf("Potential orange colors:")
	r3, g3, b3 := palette[3].RGB() // brown/dark yellow
	t.Logf("Color 3 (brown/yellow): #%02X%02X%02X", r3, g3, b3)

	r11, g11, b11 := palette[11].RGB() // bright yellow
	t.Logf("Color 11 (bright yellow): #%02X%02X%02X", r11, g11, b11)

	// Check gray colors
	t.Logf("Gray colors:")
	r7, g7, b7 := palette[7].RGB() // light gray
	t.Logf("Color 7 (light gray): #%02X%02X%02X", r7, g7, b7)

	r8, g8, b8 := palette[8].RGB() // dark gray
	t.Logf("Color 8 (dark gray): #%02X%02X%02X", r8, g8, b8)

	// Note: Orange in ANSI art might use:
	// - True color codes (38;2;255;165;0 for orange)
	// - Color 3 (brown) as closest standard ANSI approximation
	// - Color 11 (bright yellow) as brighter approximation

	// The issue might be that the remote system is sending true color orange
	// codes that we're passing through unchanged, but the terminal isn't
	// displaying them correctly, or they're being converted incorrectly
}
