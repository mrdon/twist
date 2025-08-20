package ansi

import (
	"strings"
)

// StreamingStripper removes ANSI escape sequences from streaming text
// It maintains state to handle escape sequences that span across chunks
type StreamingStripper struct {
	state      int    // 0=normal, 1=saw_esc, 2=in_sequence
	ansiBuffer string // Buffer for partial ANSI sequences
}

// NewStreamingStripper creates a new streaming ANSI stripper
func NewStreamingStripper() *StreamingStripper {
	return &StreamingStripper{
		state:      0,
		ansiBuffer: "",
	}
}

// StripChunk processes a chunk of text and returns the ANSI-stripped version
// It handles ANSI escape sequences that may be split across chunks
func (s *StreamingStripper) StripChunk(text string) string {
	var result strings.Builder

	for _, char := range text {
		switch s.state {
		case 0: // Normal state
			if char == '\x1b' {
				s.state = 1
				s.ansiBuffer = string(char)
			} else {
				result.WriteRune(char)
			}

		case 1: // Saw escape character
			s.ansiBuffer += string(char)
			if char == '[' {
				s.state = 2 // Enter ANSI sequence
			} else {
				// Not an ANSI escape, output the buffer and current char
				result.WriteString(s.ansiBuffer)
				s.ansiBuffer = ""
				s.state = 0
			}

		case 2: // In ANSI sequence
			s.ansiBuffer += string(char)
			// Check if this is a terminating character for ANSI sequences
			if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || char == 'm' || char == 'K' || char == 'H' || char == 'J' {
				// End of ANSI sequence, don't output anything from buffer
				s.ansiBuffer = ""
				s.state = 0
			}
			// Continue accumulating sequence characters (numbers, semicolons, etc.)
		}
	}

	return result.String()
}

// Reset resets the stripper state (useful for new connections)
func (s *StreamingStripper) Reset() {
	s.state = 0
	s.ansiBuffer = ""
}

// StripString is a convenience function for stripping ANSI from a complete string
// This is equivalent to creating a new stripper and calling StripChunk once
func StripString(text string) string {
	stripper := NewStreamingStripper()
	return stripper.StripChunk(text)
}
