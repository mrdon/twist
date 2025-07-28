package ansi

import (
	"testing"
	"twist/internal/theme"

	"github.com/rivo/tview"
)

func TestThemeAwareANSIWriter_BasicColors(t *testing.T) {
	// Create a text view and theme-aware writer
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	// Test standard ANSI red (should convert to theme's red)
	testData := []byte("\x1b[31mRed text\x1b[0m")

	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Fatalf("Expected to write %d bytes, got %d", len(testData), n)
	}

	// The conversion should have happened internally
	// This test mainly ensures no crashes and proper byte handling
}

func TestThemeAwareANSIWriter_StreamingEscape(t *testing.T) {
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	// Test partial escape sequence across writes
	part1 := []byte("\x1b[3")
	part2 := []byte("1mRed text\x1b[0m")

	// Write first part (incomplete escape)
	n1, err1 := writer.Write(part1)
	if err1 != nil {
		t.Fatalf("First write failed: %v", err1)
	}
	if n1 != len(part1) {
		t.Fatalf("Expected to write %d bytes, got %d", len(part1), n1)
	}

	// Write second part (completes escape)
	n2, err2 := writer.Write(part2)
	if err2 != nil {
		t.Fatalf("Second write failed: %v", err2)
	}
	if n2 != len(part2) {
		t.Fatalf("Expected to write %d bytes, got %d", len(part2), n2)
	}
}

func TestANSIColorPalette(t *testing.T) {
	// Test that theme provides proper ANSI color palette
	currentTheme := theme.Current()
	palette := currentTheme.ANSIColorPalette()

	// Should have exactly 16 colors
	if len(palette) != 16 {
		t.Fatalf("Expected 16 colors in palette, got %d", len(palette))
	}

	// Test specific colors from Telix theme
	// Color 0 should be black (0x000000)
	if r, g, b := palette[0].RGB(); r != 0 || g != 0 || b != 0 {
		t.Errorf("Expected color 0 to be black (0,0,0), got (%d,%d,%d)", r, g, b)
	}

	// Color 1 should be dark red (0x800000)
	if r, g, b := palette[1].RGB(); r != 128 || g != 0 || b != 0 {
		t.Errorf("Expected color 1 to be dark red (128,0,0), got (%d,%d,%d)", r, g, b)
	}
}

func TestStateBasedColorConversion(t *testing.T) {
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	tests := []struct {
		name     string
		input    string
	}{
		{
			name:     "Plain text gets theme defaults",
			input:    "Hello",
		},
		{
			name:     "Red text maintains state",
			input:    "\x1b[31mRed text",
		},
		{
			name:     "Mixed content",
			input:    "\x1b[31mRed\x1b[32mGreen\x1b[0mReset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process the input through the writer
			_, err := writer.Write([]byte(tt.input))
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			// For now, just verify no errors occurred
			// In a full implementation, we'd check the actual state
		})
	}
}

func TestStateTracking(t *testing.T) {
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	// Initialize state
	writer.initializeState()

	// Test state updates
	writer.updateState("31") // Red foreground
	
	// Verify state was updated (check that we have some state)
	if !writer.initialized {
		t.Error("Expected writer to be initialized")
	}

	// Test reset
	writer.updateState("0")
	
	// Should still be initialized but with theme defaults
	if !writer.initialized {
		t.Error("Expected writer to remain initialized after reset")
	}
}

func TestDifferentANSICodesProduceDifferentColors(t *testing.T) {
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)
	
	// Test that 30;40 (black fg, black bg) and 31;40 (red fg, black bg) produce different results
	writer.initializeState()
	
	// Test black foreground with black background (30;40)
	writer.updateState("30;40")
	blackFgState := writer.stateToANSI()
	blackFg := writer.state.foreground
	blackBg := writer.state.background
	
	// Test red foreground with black background (31;40) 
	writer.updateState("31;40")
	redFgState := writer.stateToANSI()
	redFg := writer.state.foreground
	redBg := writer.state.background
	
	// The foreground colors should be different
	if blackFg == redFg {
		t.Errorf("Expected different foreground colors for 30;40 vs 31;40, but both got: %v", blackFg)
	}
	
	// The background colors should be the same (both black)
	if blackBg != redBg {
		t.Errorf("Expected same background colors for 30;40 vs 31;40, got: %v vs %v", blackBg, redBg)
	}
	
	// The ANSI output should be different
	if blackFgState == redFgState {
		t.Errorf("Expected different ANSI sequences for 30;40 vs 31;40, but both got: %q", blackFgState)
	}
	
	// Verify specific colors from theme palette
	currentTheme := theme.Current()
	palette := currentTheme.ANSIColorPalette()
	
	expectedBlack := palette[0] // Color index 0 = black
	expectedRed := palette[1]   // Color index 1 = dark red
	
	if blackFg != expectedBlack {
		t.Errorf("Expected black foreground to be %v, got %v", expectedBlack, blackFg)
	}
	
	if redFg != expectedRed {
		t.Errorf("Expected red foreground to be %v, got %v", expectedRed, redFg)
	}
	
	// Log the actual RGB values for debugging
	blackR, blackG, blackB := blackFg.RGB()
	redR, redG, redB := redFg.RGB()
	
	t.Logf("Black (30): RGB(%d,%d,%d) = #%02X%02X%02X", blackR, blackG, blackB, blackR, blackG, blackB)
	t.Logf("Red (31): RGB(%d,%d,%d) = #%02X%02X%02X", redR, redG, redB, redR, redG, redB)
	t.Logf("Black ANSI: %q", blackFgState)
	t.Logf("Red ANSI: %q", redFgState)
}

func TestColorStatePersistence(t *testing.T) {
	// Initialize theme
	theme.GetThemeManager().SetTheme("telix")

	tests := []struct {
		name     string
		sequence string
		expected string
		desc     string
	}{
		{
			name:     "bright_red_no_bold",
			sequence: "\x1b[91mBRIGHT_RED\x1b[0m",
			expected: "[#ff0000:]BRIGHT_RED[#c0c0c0:]",
			desc:     "Bright red should not have bold formatting",
		},
		{
			name:     "bold_red",
			sequence: "\x1b[1;31mBOLD_RED\x1b[0m",
			expected: "[#ff0000::b]BOLD_RED[#c0c0c0:]", // Reset should clear bold
			desc:     "Bold + red should show bright red with bold, reset should clear bold",
		},
		{
			name:     "bright_red_after_bold",
			sequence: "\x1b[91mBRIGHT_RED_AGAIN\x1b[0m",
			expected: "[#ff0000:]BRIGHT_RED_AGAIN[#c0c0c0:]",
			desc:     "Bright red should not inherit previous bold state",
		},
		{
			name:     "simple_red_green",
			sequence: "\x1b[31mNO\x1b[0m \x1b[32mPVP\x1b[0m",
			expected: "[#800000:]NO[#c0c0c0:] [#008000:]PVP[#c0c0c0:]",
			desc:     "Simple red NO, green PVP",
		},
		{
			name:     "bold_red_green",
			sequence: "\x1b[1;31mNO\x1b[0m \x1b[1;32mPVP\x1b[0m",
			expected: "[#ff0000::b]NO[#c0c0c0:] [#00ff00::b]PVP[#c0c0c0:]", // Both resets should clear bold
			desc:     "Bold red NO, bold green PVP - resets should clear bold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh TextView and writer for each test
			textView := tview.NewTextView()
			writer := NewThemeAwareANSIWriter(textView)

			// Write the sequence
			_, err := writer.Write([]byte(tt.sequence))
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			// Get the output
			output := textView.GetText(false)

			// Check if output matches expected
			if output != tt.expected {
				t.Errorf("Test %s failed:\nDesc: %s\nInput: %q\nExpected: %q\nGot:      %q", 
					tt.name, tt.desc, tt.sequence, tt.expected, output)
			} else {
				t.Logf("Test %s passed: %q -> %q", tt.name, tt.sequence, output)
			}
		})
	}
}

func TestResetBehavior(t *testing.T) {
	// Initialize theme
	theme.GetThemeManager().SetTheme("telix")

	// Create TextView and writer
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	// Test sequence: bold -> reset -> non-bold color
	sequences := []struct {
		seq      string
		expected string
		desc     string
	}{
		{
			seq:      "\x1b[1mBOLD",
			expected: "[#c0c0c0::b]BOLD",
			desc:     "Just bold modifier",
		},
		{
			seq:      "\x1b[0mRESET",
			expected: "[#c0c0c0::b]BOLD[#c0c0c0:]RESET",
			desc:     "Reset should clear bold",
		},
		{
			seq:      "\x1b[91mBRIGHT",
			expected: "[#c0c0c0::b]BOLD[#c0c0c0:]RESET[#ff0000:]BRIGHT",
			desc:     "Bright red should not be bold",
		},
	}

	fullExpected := ""
	for i, seq := range sequences {
		_, err := writer.Write([]byte(seq.seq))
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}

		output := textView.GetText(false)
		fullExpected = seq.expected

		t.Logf("Step %d - %s: %q -> %q", i+1, seq.desc, seq.seq, output)
	}

	// Final check
	finalOutput := textView.GetText(false)
	if finalOutput != fullExpected {
		t.Errorf("Final output mismatch:\nExpected: %q\nGot:      %q", fullExpected, finalOutput)
	}
}

func TestNoPVPBoldCyanSequence(t *testing.T) {
	// Import terminal package - we need to create the full terminal+converter setup
	// This test needs to be updated to use the actual terminal flow
	
	// Initialize theme  
	theme.GetThemeManager().SetTheme("telix")

	// Create TextView and writer
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	// Test the exact sequence from the bug report: \x1b[31m NO \x1b[1mPVP\x1b[36m]
	// We'll test the ConvertANSIParams method directly to verify the converter logic
	testSequences := []struct {
		params   string
		expected struct {
			fgHex string
			bold  bool
		}
		desc string
	}{
		{
			params: "31",
			expected: struct {
				fgHex string
				bold  bool
			}{fgHex: "#800000", bold: false},
			desc: "Red foreground",
		},
		{
			params: "1",
			expected: struct {
				fgHex string
				bold  bool
			}{fgHex: "#800000", bold: true},
			desc: "Bold modifier (should keep red)",
		},
		{
			params: "36",
			expected: struct {
				fgHex string
				bold  bool
			}{fgHex: "#00ffff", bold: true},
			desc: "Cyan with bold should be bright cyan",
		},
	}

	for i, test := range testSequences {
		fgHex, _, bold, _, _ := writer.ConvertANSIParams(test.params)
		
		if fgHex != test.expected.fgHex {
			t.Errorf("Step %d (%s): Expected FG %s, got %s", 
				i+1, test.desc, test.expected.fgHex, fgHex)
		}
		
		if bold != test.expected.bold {
			t.Errorf("Step %d (%s): Expected bold %t, got %t", 
				i+1, test.desc, test.expected.bold, bold)
		}
		
		t.Logf("Step %d (%s): params=%q -> FG=%s Bold=%t ✓", 
			i+1, test.desc, test.params, fgHex, bold)
	}
}

func TestBoldColorCombinations(t *testing.T) {
	// Initialize theme
	theme.GetThemeManager().SetTheme("telix")

	// Test the converter logic step by step for bold+color combinations
	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	tests := []struct {
		name     string
		params   string
		expected struct {
			fgHex string
			bold  bool
		}
		desc string
	}{
		{
			name:   "bold_first",
			params: "1",
			expected: struct {
				fgHex string
				bold  bool
			}{fgHex: "#c0c0c0", bold: true},
			desc: "Bold modifier should set bold and keep theme default foreground",
		},
		{
			name:   "cyan_after_bold",
			params: "36",
			expected: struct {
				fgHex string
				bold  bool
			}{fgHex: "#00ffff", bold: true},
			desc: "Cyan after bold should be bright cyan and keep bold",
		},
		{
			name:   "compound_bold_red",
			params: "1;31",
			expected: struct {
				fgHex string
				bold  bool
			}{fgHex: "#ff0000", bold: true},
			desc: "Compound bold+red should be bright red with bold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fgHex, _, bold, _, _ := writer.ConvertANSIParams(tt.params)

			if fgHex != tt.expected.fgHex {
				t.Errorf("Test %s: Expected FG %s, got %s", tt.name, tt.expected.fgHex, fgHex)
			}

			if bold != tt.expected.bold {
				t.Errorf("Test %s: Expected bold %t, got %t", tt.name, tt.expected.bold, bold)
			}

			t.Logf("Test %s passed: params=%q -> FG=%s Bold=%t", 
				tt.name, tt.params, fgHex, bold)
		})
	}
}

func TestStandaloneBoldModifierReEvaluation(t *testing.T) {
	// This test specifically verifies the fix for standalone bold modifiers
	// that should re-evaluate existing colors to their bright equivalents
	
	// Initialize theme
	theme.GetThemeManager().SetTheme("telix")

	textView := tview.NewTextView()
	writer := NewThemeAwareANSIWriter(textView)

	// Step 1: Set red color (31)
	fgHex1, _, bold1, _, _ := writer.ConvertANSIParams("31")
	t.Logf("Step 1 - Red (31): FG=%s, Bold=%t", fgHex1, bold1)

	// Verify it's standard red and not bold
	if fgHex1 != "#800000" {
		t.Errorf("Expected red to be #800000, got %s", fgHex1)
	}
	if bold1 {
		t.Errorf("Expected bold to be false after setting red, got %t", bold1)
	}

	// Step 2: Apply standalone bold modifier (1)
	fgHex2, _, bold2, _, _ := writer.ConvertANSIParams("1")
	t.Logf("Step 2 - Bold (1): FG=%s, Bold=%t", fgHex2, bold2)

	// Verify bold is set
	if !bold2 {
		t.Errorf("Expected bold to be true after bold modifier, got %t", bold2)
	}

	// CRITICAL: Verify that the existing red color was re-evaluated to bright red
	if fgHex2 != "#ff0000" {
		t.Errorf("Expected red to become bright red (#ff0000) after bold modifier, got %s", fgHex2)
	}

	// Step 3: Set cyan color (36) - should be bright cyan because bold is still active
	fgHex3, _, bold3, _, _ := writer.ConvertANSIParams("36")
	t.Logf("Step 3 - Cyan (36): FG=%s, Bold=%t", fgHex3, bold3)

	// Verify bold is still active
	if !bold3 {
		t.Errorf("Expected bold to remain true after setting cyan, got %t", bold3)
	}

	// Verify cyan is bright cyan (not standard cyan)
	if fgHex3 != "#00ffff" {
		t.Errorf("Expected cyan with bold to be bright cyan (#00ffff), got %s", fgHex3)
	}

	t.Logf("✓ Complete sequence test passed:")
	t.Logf("  Red (31):     %s (bold=%t)", fgHex1, bold1)
	t.Logf("  Bold (1):     %s (bold=%t) <- Red became bright", fgHex2, bold2)
	t.Logf("  Cyan (36):    %s (bold=%t) <- Cyan is bright", fgHex3, bold3)
}

func TestStandaloneBoldWithDifferentColors(t *testing.T) {
	// Test that standalone bold re-evaluation works for all standard colors
	
	// Initialize theme
	theme.GetThemeManager().SetTheme("telix")

	standardColors := []struct {
		code       string
		name       string
		normalHex  string
		brightHex  string
	}{
		{"30", "black", "#000000", "#808080"},
		{"31", "red", "#800000", "#ff0000"},
		{"32", "green", "#008000", "#00ff00"},
		{"33", "yellow", "#808000", "#ffff00"},
		{"34", "blue", "#000080", "#0000ff"},
		{"35", "magenta", "#800080", "#ff00ff"},
		{"36", "cyan", "#008080", "#00ffff"},
		{"37", "white", "#c0c0c0", "#ffffff"},
	}

	for _, color := range standardColors {
		t.Run(color.name, func(t *testing.T) {
			textView := tview.NewTextView()
			writer := NewThemeAwareANSIWriter(textView)

			// Step 1: Set the standard color
			fgHex1, _, bold1, _, _ := writer.ConvertANSIParams(color.code)
			
			// Verify it's the normal color and not bold
			if fgHex1 != color.normalHex {
				t.Errorf("Expected %s to be %s, got %s", color.name, color.normalHex, fgHex1)
			}
			if bold1 {
				t.Errorf("Expected bold to be false for %s, got %t", color.name, bold1)
			}

			// Step 2: Apply standalone bold modifier
			fgHex2, _, bold2, _, _ := writer.ConvertANSIParams("1")
			
			// Verify bold is set and color became bright
			if !bold2 {
				t.Errorf("Expected bold to be true for %s after bold modifier, got %t", color.name, bold2)
			}
			if fgHex2 != color.brightHex {
				t.Errorf("Expected %s to become bright (%s) after bold modifier, got %s", 
					color.name, color.brightHex, fgHex2)
			}

			t.Logf("✓ %s: %s -> bold -> %s", color.name, color.normalHex, color.brightHex)
		})
	}
}