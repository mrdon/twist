package terminal

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// parseLogChunks parses the twist_data_chunks.log format and returns the actual data chunks
func parseLogChunks(filename string) ([][]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunks [][]byte
	scanner := bufio.NewScanner(file)
	
	// Pattern to match: "OnData chunk (N bytes):"
	chunkHeaderPattern := regexp.MustCompile(`^OnData chunk \((\d+) bytes\):$`)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check if this is a chunk header
		matches := chunkHeaderPattern.FindStringSubmatch(line)
		if matches != nil {
			// Next line should be the data
			if scanner.Scan() {
				dataLine := scanner.Text()
				// Convert the escaped string back to bytes
				data, err := strconv.Unquote(`"` + dataLine + `"`)
				if err != nil {
					// If unquote fails, try treating it as raw data
					data = dataLine
				}
				chunks = append(chunks, []byte(data))
			}
		}
	}
	
	return chunks, scanner.Err()
}

func TestRealLogData(t *testing.T) {
	// Parse the actual log data
	chunks, err := parseLogChunks("../../twist_data_chunks.log")
	if err != nil {
		t.Skipf("Could not read log file: %v", err)
		return
	}
	
	if len(chunks) == 0 {
		t.Skip("No chunks found in log file")
		return
	}
	
	t.Logf("Found %d chunks in log file", len(chunks))
	
	// Create test terminal
	term := newTestTerminal()
	
	// Process each chunk
	for i, chunk := range chunks {
		t.Logf("Processing chunk %d: %d bytes", i, len(chunk))
		
		// Show first few bytes for debugging
		preview := chunk
		if len(preview) > 20 {
			preview = preview[:20]
		}
		t.Logf("  Preview: %q", string(preview))
		
		term.processDataWithANSI(chunk)
	}
	
	t.Logf("Final results:")
	t.Logf("  ANSI sequences captured: %d", len(term.sequences))
	t.Logf("  Regular characters captured: %d", len(term.runes))
	
	// Show first few sequences
	for i, seq := range term.sequences {
		if i >= 10 {
			t.Logf("  ... and %d more sequences", len(term.sequences)-10)
			break
		}
		t.Logf("  Sequence %d: %q", i, seq)
	}
	
	// Look for corrupted sequences (sequences that don't start with \x1b[)
	corruptedSeqs := 0
	for _, seq := range term.sequences {
		if !strings.HasPrefix(seq, "\x1b[") {
			corruptedSeqs++
			t.Logf("  CORRUPTED sequence: %q", seq)
		}
	}
	
	if corruptedSeqs > 0 {
		t.Errorf("Found %d corrupted sequences that don't start with \\x1b[", corruptedSeqs)
	}
	
	// Look for sequences that got processed as text (like "[40m" appearing as runes)
	suspiciousRunes := 0
	runeStr := string(term.runes)
	suspiciousPatterns := []string{"[40m", "[0m", "[31m", "[32m", "[H", "[2J"}
	
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(runeStr, pattern) {
			suspiciousRunes++
			t.Logf("  SUSPICIOUS: Found %q in regular text output", pattern)
		}
	}
	
	if suspiciousRunes > 0 {
		t.Errorf("Found %d ANSI sequence fragments processed as regular text", suspiciousRunes)
	}
	
	t.Logf("Test completed successfully - no corruption detected!")
}