//go:build integration

package parsing

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"testing"
	"twist/internal/proxy/database"
	"twist/internal/proxy/streaming"
)

// CreateTestParser creates a TWX parser with mock API for testing
func CreateTestParser(t *testing.T) (*streaming.TWXParser, *MockTuiAPI, database.Database) {
	mockAPI := NewMockTuiAPI(t)
	
	db := database.NewDatabase()
	if err := db.CreateDatabase(":memory:"); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Debug: Verify database is working after creation
	var testCount int
	if err := db.GetDB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sectors'").Scan(&testCount); err != nil {
		t.Fatalf("Failed to verify sectors table after creation: %v", err)
	}
	if testCount == 0 {
		t.Fatalf("sectors table was not created during CreateDatabase")
	}
	
	twxParser := streaming.NewTWXParser(db, mockAPI)
	
	// Debug: Verify database is still working after parser creation
	if err := db.GetDB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sectors'").Scan(&testCount); err != nil {
		t.Fatalf("Failed to verify sectors table after parser creation: %v", err)
	}
	if testCount == 0 {
		t.Fatalf("sectors table disappeared after parser creation")
	}
	
	return twxParser, mockAPI, db
}

// ParseDataChunks reads chunked data from a file and returns byte slices
func ParseDataChunks(filename string) [][]byte {
	file, _ := os.Open(filename)
	defer file.Close()
	var chunks [][]byte
	scanner := bufio.NewScanner(file)
	chunkHeaderPattern := regexp.MustCompile(`^OnData chunk \((\d+) bytes\):$`)
	for scanner.Scan() {
		if matches := chunkHeaderPattern.FindStringSubmatch(scanner.Text()); matches != nil {
			if scanner.Scan() {
				data, _ := strconv.Unquote(`"` + scanner.Text() + `"`)
				chunks = append(chunks, []byte(data))
			}
		}
	}
	return chunks
}

// ProcessRealWorldData processes a transcript file and returns the TUI API calls made
func ProcessRealWorldData(t *testing.T, filename string) []string {
	parser, mockAPI, db := CreateTestParser(t)
	defer db.CloseDatabase()
	for _, chunk := range ParseDataChunks(filename) { parser.ProcessInBound(string(chunk)) }
	
	// Force completion of any pending sector when data stream ends
	parser.Finalize()
	
	return mockAPI.GetCalls()
}

// AssertTuiApiCalls processes a transcript file and asserts it produces the expected TUI API calls
func AssertTuiApiCalls(t *testing.T, filename string, expected []string) {
	calls := ProcessRealWorldData(t, filename)
	
	if len(calls) != len(expected) {
		t.Errorf("Expected %d calls, got %d. Expected: %v, Got: %v", len(expected), len(calls), expected, calls)
		return
	}
	
	for i, expectedCall := range expected {
		if calls[i] != expectedCall {
			t.Errorf("Call %d: expected %q, got %q", i, expectedCall, calls[i])
		}
	}
}