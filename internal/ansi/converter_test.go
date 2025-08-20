package ansi

import (
	"testing"
)

func TestColorConverter_Reset(t *testing.T) {
	converter := NewColorConverter()

	colorTag := converter.ConvertANSIParams("0")

	// Should return tview color tag with default colors
	expected := "[#c0c0c0:#000000]"
	if colorTag != expected {
		t.Errorf("Expected reset tag %s, got %s", expected, colorTag)
	}
}

func TestColorConverter_BasicColors(t *testing.T) {
	// Test red foreground (start fresh)
	converter := NewColorConverter()
	colorTag := converter.ConvertANSIParams("31")
	expected := "[#800000:#000000]"
	if colorTag != expected {
		t.Errorf("Expected red FG tag %s, got %s", expected, colorTag)
	}

	// Test green background (start fresh)
	converter = NewColorConverter()
	colorTag = converter.ConvertANSIParams("42")
	expected = "[#c0c0c0:#008000]"
	if colorTag != expected {
		t.Errorf("Expected green BG tag %s, got %s", expected, colorTag)
	}

	// Test bold (start fresh) - bold makes default gray become white
	converter = NewColorConverter()
	colorTag = converter.ConvertANSIParams("1")
	expected = "[#ffffff:#000000:b]"
	if colorTag != expected {
		t.Errorf("Expected bold tag %s, got %s", expected, colorTag)
	}
}

func TestColorConverter_Combined(t *testing.T) {
	converter := NewColorConverter()

	// Test combined sequence: bold + red foreground + green background
	colorTag := converter.ConvertANSIParams("1;31;42")

	// Bold red should be bright red (#ff0000), green background, with bold attribute
	expected := "[#ff0000:#008000:b]"
	if colorTag != expected {
		t.Errorf("Expected combined tag %s, got %s", expected, colorTag)
	}
}

func TestColorConverter_DirectCall(t *testing.T) {
	converter := NewColorConverter()

	// Test direct method call (no interface needed)
	colorTag := converter.ConvertANSIParams("31")

	expected := "[#800000:#000000]"
	if colorTag != expected {
		t.Errorf("Expected red FG tag %s, got %s", expected, colorTag)
	}
}
