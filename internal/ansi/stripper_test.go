package ansi

import (
	"testing"
)

func TestStreamingStripper_BasicStripping(t *testing.T) {
	stripper := NewStreamingStripper()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ansi sequences",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "simple color sequence",
			input:    "\x1b[31mRed Text\x1b[0m",
			expected: "Red Text",
		},
		{
			name:     "game option pattern",
			input:    "\x1b[31m<A> Alien Retribution\x1b[0m",
			expected: "<A> Alien Retribution",
		},
		{
			name:     "multiple sequences",
			input:    "\x1b[31mRed \x1b[32mGreen \x1b[0mNormal",
			expected: "Red Green Normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripper.StripChunk(tt.input)
			if result != tt.expected {
				t.Errorf("StripChunk() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStreamingStripper_ChunkSplitting(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		expected string
	}{
		{
			name:     "ANSI sequence split across chunks",
			chunks:   []string{"\x1b", "[31m", "Red", "\x1b[0m"},
			expected: "Red",
		},
		{
			name:     "game option split across chunks",
			chunks:   []string{"\x1b[", "31m<A> ", "Alien Retribution", "\x1b[0m"},
			expected: "<A> Alien Retribution",
		},
		{
			name:     "escape character at end of chunk",
			chunks:   []string{"Hello \x1b", "[31mRed\x1b[0m"},
			expected: "Hello Red",
		},
		{
			name:     "complex splitting",
			chunks:   []string{"\x1b[31", "m<A> Alien ", "Retri", "bution\x1b[", "0m"},
			expected: "<A> Alien Retribution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stripper := NewStreamingStripper()
			var result string

			for _, chunk := range tt.chunks {
				result += stripper.StripChunk(chunk)
			}

			if result != tt.expected {
				t.Errorf("Final result = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStreamingStripper_Reset(t *testing.T) {
	stripper := NewStreamingStripper()

	// Start processing a sequence
	result1 := stripper.StripChunk("\x1b[31")
	if result1 != "" {
		t.Errorf("Partial sequence should not produce output, got %q", result1)
	}

	// Reset should clear state
	stripper.Reset()

	// Should work normally after reset
	result2 := stripper.StripChunk("Hello")
	if result2 != "Hello" {
		t.Errorf("After reset, normal text should pass through, got %q", result2)
	}
}

func TestStripString(t *testing.T) {
	input := "\x1b[31m<A> Alien Retribution\x1b[0m"
	expected := "<A> Alien Retribution"

	result := StripString(input)
	if result != expected {
		t.Errorf("StripString() = %q, want %q", result, expected)
	}
}
