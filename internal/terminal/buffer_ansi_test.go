package terminal

import (
	"fmt"
	"testing"
	"twist/internal/ansi"
	"unicode/utf8"
)

// testTerminal for capturing sequences and runes
type testTerminal struct {
	*Terminal
	sequences []string
	runes     []rune
	// Buffer to handle partial sequences across chunks
	partialBuffer []byte
}

func newTestTerminal() *testTerminal {
	converter := ansi.NewColorConverter()
	term := NewTerminalWithConverter(80, 24, converter)
	
	return &testTerminal{
		Terminal:      term,
		sequences:     make([]string, 0),
		runes:         make([]rune, 0),
		partialBuffer: make([]byte, 0),
	}
}

// We need to override processDataWithANSI to use our capturing methods
func (tt *testTerminal) processDataWithANSI(data []byte) {
	// Combine any buffered data with the new data
	combinedData := append(tt.partialBuffer, data...)
	tt.partialBuffer = tt.partialBuffer[:0] // Clear buffer
	
	i := 0
	for i < len(combinedData) {
		if combinedData[i] == '\x1b' {
			// Try to find complete escape sequence
			if i+1 < len(combinedData) && combinedData[i+1] == '[' {
				// ANSI escape sequence - find terminator
				end := i + 2
				foundTerminator := false
				for end < len(combinedData) {
					c := combinedData[end]
					if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
						// Found terminator - we have complete sequence
						end++ // include terminator
						foundTerminator = true
						break
					}
					end++
				}
				
				if foundTerminator {
					// Complete sequence found
					sequence := string(combinedData[i:end])
					// Capture the sequence
					tt.sequences = append(tt.sequences, sequence)
					// Process it through the terminal (avoid recursive call)
					tt.Terminal.processANSISequence(combinedData[i:end])
					i = end
				} else {
					// Incomplete sequence - buffer it for next chunk
					tt.partialBuffer = append(tt.partialBuffer, combinedData[i:]...)
					break
				}
			} else if i+1 < len(combinedData) {
				// \x1b followed by non-[ - treat as regular character
				char, size := utf8.DecodeRune(combinedData[i:])
				if char == utf8.RuneError && size == 1 {
					// Skip invalid UTF-8 byte
					i++
				} else {
					tt.runes = append(tt.runes, char)
					tt.Terminal.processRune(char)
					i += size
				}
			} else {
				// Incomplete escape at end of data - buffer it
				tt.partialBuffer = append(tt.partialBuffer, combinedData[i:]...)
				break
			}
		} else {
			// Regular character - handle UTF-8 properly
			char, size := utf8.DecodeRune(combinedData[i:])
			if char == utf8.RuneError && size == 1 {
				// Skip invalid UTF-8 byte
				i++
			} else {
				tt.runes = append(tt.runes, char)
				tt.Terminal.processRune(char)
				i += size
			}
		}
	}
}


func TestANSIChunkSplitting(t *testing.T) {
	// Test every possible split point for various ANSI sequences
	sequences := []struct {
		name string
		seq  string
	}{
		// Color sequences
		{"foreground color", "\x1b[31m"},
		{"background color", "\x1b[40m"},
		{"bright foreground", "\x1b[91m"},
		{"bright background", "\x1b[101m"},
		{"complex color", "\x1b[1;31;40m"},
		{"reset", "\x1b[0m"},
		{"default fg", "\x1b[39m"},
		{"default bg", "\x1b[49m"},
		
		// Cursor movement
		{"cursor home", "\x1b[H"},
		{"cursor position", "\x1b[10;20H"},
		{"cursor up", "\x1b[5A"},
		{"cursor down", "\x1b[3B"},
		{"cursor right", "\x1b[2C"},
		{"cursor left", "\x1b[4D"},
		{"cursor position alt", "\x1b[15;25f"},
		
		// Erase sequences
		{"erase display", "\x1b[2J"},
		{"erase line", "\x1b[K"},
		{"erase from cursor", "\x1b[0J"},
		{"erase to cursor", "\x1b[1J"},
	}

	for _, seq := range sequences {
		t.Run(seq.name, func(t *testing.T) {
			// Test every possible split point in the sequence
			for splitPoint := 1; splitPoint < len(seq.seq); splitPoint++ {
				t.Run(fmt.Sprintf("split_at_%d", splitPoint), func(t *testing.T) {
					term := newTestTerminal()
					
					// Split the sequence at the split point
					part1 := []byte("A" + seq.seq[:splitPoint])  // Add 'A' so we can verify regular chars work
					part2 := []byte(seq.seq[splitPoint:] + "B")  // Add 'B' so we can verify regular chars work
					
					// Process both chunks
					term.processDataWithANSI(part1)
					term.processDataWithANSI(part2)
					
					// Should have exactly one ANSI sequence
					if len(term.sequences) != 1 {
						t.Errorf("expected 1 sequence, got %d: %v", len(term.sequences), term.sequences)
						return
					}
					
					// Should be the complete sequence
					if term.sequences[0] != seq.seq {
						t.Errorf("expected sequence %q, got %q", seq.seq, term.sequences[0])
					}
					
					// Should have exactly 2 regular characters (A and B)
					if len(term.runes) != 2 {
						t.Errorf("expected 2 runes, got %d: %v", len(term.runes), term.runes)
						return
					}
					
					if term.runes[0] != 'A' || term.runes[1] != 'B' {
						t.Errorf("expected runes ['A', 'B'], got %v", term.runes)
					}
				})
			}
		})
	}
}

func TestANSIRealWorldCases(t *testing.T) {
	tests := []struct {
		name      string
		chunks    [][]byte
		wantSeqs  []string
		wantRunes []rune
	}{
		{
			name: "normal case - no boundary issues",
			chunks: [][]byte{
				[]byte("Hello \x1b[31mRed\x1b[0m World"),
			},
			wantSeqs:  []string{"\x1b[31m", "\x1b[0m"},
			wantRunes: []rune{'H', 'e', 'l', 'l', 'o', ' ', 'R', 'e', 'd', ' ', 'W', 'o', 'r', 'l', 'd'},
		},
		{
			name: "multiple complete sequences in single chunk",
			chunks: [][]byte{
				[]byte("\x1b[31mRed\x1b[32mGreen\x1b[34mBlue\x1b[0mNormal"),
			},
			wantSeqs:  []string{"\x1b[31m", "\x1b[32m", "\x1b[34m", "\x1b[0m"},
			wantRunes: []rune{'R', 'e', 'd', 'G', 'r', 'e', 'e', 'n', 'B', 'l', 'u', 'e', 'N', 'o', 'r', 'm', 'a', 'l'},
		},
		{
			name: "real problematic case from logs",
			chunks: [][]byte{
				[]byte("content\x1b"),
				[]byte("[40m░\x1b[0mmore"),
			},
			wantSeqs:  []string{"\x1b[40m", "\x1b[0m"},
			wantRunes: []rune{'c', 'o', 'n', 't', 'e', 'n', 't', '░', 'm', 'o', 'r', 'e'},
		},
		{
			name: "multiple sequences with various splits",
			chunks: [][]byte{
				[]byte("\x1b[31mRed\x1b"),
				[]byte("[0mNormal\x1b["),
				[]byte("32mGreen"),
			},
			wantSeqs:  []string{"\x1b[31m", "\x1b[0m", "\x1b[32m"},
			wantRunes: []rune{'R', 'e', 'd', 'N', 'o', 'r', 'm', 'a', 'l', 'G', 'r', 'e', 'e', 'n'},
		},
		{
			name: "cursor positioning split",
			chunks: [][]byte{
				[]byte("Text\x1b["),
				[]byte("10;20H"),
				[]byte("More"),
			},
			wantSeqs:  []string{"\x1b[10;20H"},
			wantRunes: []rune{'T', 'e', 'x', 't', 'M', 'o', 'r', 'e'},
		},
		{
			name: "text without any sequences",
			chunks: [][]byte{
				[]byte("Just"),
				[]byte(" plain"),
				[]byte(" text"),
			},
			wantSeqs:  []string{},
			wantRunes: []rune{'J', 'u', 's', 't', ' ', 'p', 'l', 'a', 'i', 'n', ' ', 't', 'e', 'x', 't'},
		},
		{
			name: "many sequences no splits - single chunk",
			chunks: [][]byte{
				[]byte("\x1b[31mRed\x1b[32mGreen\x1b[34mBlue\x1b[1;33mBoldYellow\x1b[0;40mBgBlack\x1b[H\x1b[2J\x1b[10;20HText\x1b[0mEnd"),
			},
			wantSeqs:  []string{"\x1b[31m", "\x1b[32m", "\x1b[34m", "\x1b[1;33m", "\x1b[0;40m", "\x1b[H", "\x1b[2J", "\x1b[10;20H", "\x1b[0m"},
			wantRunes: []rune{'R', 'e', 'd', 'G', 'r', 'e', 'e', 'n', 'B', 'l', 'u', 'e', 'B', 'o', 'l', 'd', 'Y', 'e', 'l', 'l', 'o', 'w', 'B', 'g', 'B', 'l', 'a', 'c', 'k', 'T', 'e', 'x', 't', 'E', 'n', 'd'},
		},
		{
			name: "many sequences no splits - multiple chunks",
			chunks: [][]byte{
				[]byte("\x1b[31mRed\x1b[32mGreen"),
				[]byte("\x1b[34mBlue\x1b[1;33mBoldYellow"),
				[]byte("\x1b[0;40mBgBlack\x1b[H\x1b[2J"),
				[]byte("\x1b[10;20HText\x1b[0mEnd"),
			},
			wantSeqs:  []string{"\x1b[31m", "\x1b[32m", "\x1b[34m", "\x1b[1;33m", "\x1b[0;40m", "\x1b[H", "\x1b[2J", "\x1b[10;20H", "\x1b[0m"},
			wantRunes: []rune{'R', 'e', 'd', 'G', 'r', 'e', 'e', 'n', 'B', 'l', 'u', 'e', 'B', 'o', 'l', 'd', 'Y', 'e', 'l', 'l', 'o', 'w', 'B', 'g', 'B', 'l', 'a', 'c', 'k', 'T', 'e', 'x', 't', 'E', 'n', 'd'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := newTestTerminal()
			
			// Process each chunk
			for _, chunk := range tt.chunks {
				term.processDataWithANSI(chunk)
			}
			
			// Check sequences
			if len(term.sequences) != len(tt.wantSeqs) {
				t.Errorf("got %d sequences, want %d", len(term.sequences), len(tt.wantSeqs))
				t.Errorf("got sequences: %v", term.sequences)
				t.Errorf("want sequences: %v", tt.wantSeqs)
				return
			}
			
			for i, seq := range term.sequences {
				if seq != tt.wantSeqs[i] {
					t.Errorf("sequence %d: got %q, want %q", i, seq, tt.wantSeqs[i])
				}
			}
			
			// Check runes
			if len(term.runes) != len(tt.wantRunes) {
				t.Errorf("got %d runes, want %d", len(term.runes), len(tt.wantRunes))
				t.Errorf("got runes: %v", term.runes)
				t.Errorf("want runes: %v", tt.wantRunes)
				return
			}
			
			for i, r := range term.runes {
				if r != tt.wantRunes[i] {
					t.Errorf("rune %d: got %q, want %q", i, r, tt.wantRunes[i])
				}
			}
		})
	}
}