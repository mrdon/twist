//go:build integration

package scripting

import (
	"strings"
	"testing"
	"time"
)

func TestGetDate_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		getdate $current_date
		echo "Current date: " $current_date
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("GETDATE command failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	// Verify the output contains a date in MM/DD/YYYY format
	output := result.Output[0]
	if !strings.Contains(output, "Current date: ") {
		t.Errorf("Expected 'Current date: ' prefix, got %s", output)
	}

	// Verify it looks like a date (contains slashes)
	if !strings.Contains(output, "/") {
		t.Errorf("Expected date format with slashes, got %s", output)
	}
}

func TestGetDateTime_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		getdatetime $current_datetime
		echo "Current datetime: " $current_datetime
	`

	beforeTime := time.Now()
	result := tester.ExecuteScript(script)
	afterTime := time.Now()
	
	if result.Error != nil {
		t.Fatalf("GETDATETIME command failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	// Verify the output contains a datetime
	output := result.Output[0]
	if !strings.Contains(output, "Current datetime: ") {
		t.Errorf("Expected 'Current datetime: ' prefix, got %s", output)
	}

	// Verify it looks like a datetime (contains both date and time)
	if !strings.Contains(output, "/") || !strings.Contains(output, ":") {
		t.Errorf("Expected datetime format with slashes and colons, got %s", output)
	}
	
	// Extract the datetime part
	parts := strings.Split(output, "Current datetime: ")
	if len(parts) != 2 {
		t.Errorf("Could not extract datetime from: %s", output)
		return
	}
	
	// Parse and verify it's reasonable
	datetimeStr := parts[1]
	parsedTime, err := time.ParseInLocation("01/02/2006 15:04:05", datetimeStr, time.Local)
	if err != nil {
		t.Errorf("Failed to parse datetime %s: %v", datetimeStr, err)
		return
	}

	// Verify the time is within reasonable bounds (allow 5 second tolerance)
	if parsedTime.Before(beforeTime.Add(-5*time.Second)) || parsedTime.After(afterTime.Add(5*time.Second)) {
		t.Errorf("Returned datetime %v is not within expected range %v to %v", 
			parsedTime, beforeTime, afterTime)
	}
}

func TestDateTimeDiff_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	tests := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name:     "Difference in seconds",
			script:   `datetimediff "2023-01-01 12:00:00" "2023-01-01 12:00:10" "seconds" $diff`,
			expected: "10",
		},
		{
			name:     "Difference in minutes", 
			script:   `datetimediff "2023-01-01 12:00:00" "2023-01-01 12:05:00" "minutes" $diff`,
			expected: "5",
		},
		{
			name:     "Difference in hours",
			script:   `datetimediff "2023-01-01 12:00:00" "2023-01-01 15:00:00" "hours" $diff`,
			expected: "3",
		},
		{
			name:     "Difference in days",
			script:   `datetimediff "2023-01-01 12:00:00" "2023-01-03 12:00:00" "days" $diff`,
			expected: "2",
		},
		{
			name:     "Negative difference",
			script:   `datetimediff "2023-01-01 12:00:10" "2023-01-01 12:00:00" "seconds" $diff`,
			expected: "-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullScript := tt.script + `
				echo "Diff: " $diff
			`
			
			result := tester.ExecuteScript(fullScript)
			if result.Error != nil {
				t.Fatalf("Script execution failed: %v", result.Error)
			}

			if len(result.Output) != 1 {
				t.Errorf("Expected 1 output line, got %d", len(result.Output))
			}

			expectedOutput := "Diff: " + tt.expected
			if len(result.Output) > 0 && result.Output[0] != expectedOutput {
				t.Errorf("Expected %s, got %s", expectedOutput, result.Output[0])
			}
		})
	}
}

func TestDateTimeDiff_DifferentFormats_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test with different datetime formats
	tests := []struct {
		name      string
		datetime1 string
		datetime2 string
		expected  string
	}{
		{
			name:      "MM/DD/YYYY format",
			datetime1: "01/01/2023 12:00:00",
			datetime2: "01/01/2023 12:00:05",
			expected:  "5",
		},
		{
			name:      "ISO format",
			datetime1: "2023-01-01T12:00:00",
			datetime2: "2023-01-01T12:00:03",
			expected:  "3",
		},
		{
			name:      "Date only",
			datetime1: "01/01/2023",
			datetime2: "01/02/2023",
			expected:  "86400", // 24 hours in seconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := `datetimediff "` + tt.datetime1 + `" "` + tt.datetime2 + `" "seconds" $diff
				echo "Seconds: " $diff`
				
			result := tester.ExecuteScript(script)
			if result.Error != nil {
				t.Fatalf("Script execution failed: %v", result.Error)
			}

			expectedOutput := "Seconds: " + tt.expected
			if len(result.Output) != 1 || result.Output[0] != expectedOutput {
				t.Errorf("Expected %s, got %v", expectedOutput, result.Output)
			}
		})
	}
}

func TestDateTimeToStr_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Test default format (no format parameter)
	script := `
		datetimetostr "01/01/2023 15:30:45" $formatted
		echo "Formatted: " $formatted
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("DATETIMETOSTR with default format failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	// Should use default format "2006-01-02 15:04:05"
	expected := "Formatted: 2023-01-01 15:30:45"
	if len(result.Output) > 0 && result.Output[0] != expected {
		t.Errorf("Expected %s, got %s", expected, result.Output[0])
	}
}

func TestDateTimeToStr_CustomFormat_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	tests := []struct {
		name     string
		input    string
		format   string
		expected string
	}{
		{
			name:     "Year-Month-Day format",
			input:    "01/15/2023 14:30:00",
			format:   "YYYY-MM-DD",
			expected: "2023-01-15",
		},
		{
			name:     "Custom datetime format",
			input:    "12/25/2023 09:15:30",
			format:   "YYYY.MM.DD HH:mm:ss",
			expected: "2023.12.25 09:15:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := `datetimetostr "` + tt.input + `" $result "` + tt.format + `"
				echo "Result: " $result`
				
			result := tester.ExecuteScript(script)
			if result.Error != nil {
				t.Fatalf("Script execution failed: %v", result.Error)
			}

			expectedOutput := "Result: " + tt.expected
			if len(result.Output) != 1 || result.Output[0] != expectedOutput {
				t.Errorf("Expected %s, got %v", expectedOutput, result.Output)
			}
		})
	}
}

func TestStartTimer_StopTimer_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Start a timer
	script := `
		starttimer "test_timer"
		echo "Timer started"
		stoptimer "test_timer"
		echo "Timer stopped"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Timer commands failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Timer started",
		"Timer stopped",
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Timer output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

func TestDateTimeErrors_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	errorTests := []struct {
		name   string
		script string
	}{
		{
			name:   "Invalid datetime format for diff",
			script: `datetimediff "invalid-date" "2023-01-01 12:00:00" "seconds" $diff`,
		},
		{
			name:   "Invalid datetime format for tostr",
			script: `datetimetostr "not-a-date" $result`,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			result := tester.ExecuteScript(tt.script)
			if result.Error == nil {
				t.Errorf("Expected error for %s, but script succeeded", tt.name)
			}
		})
	}
}

func TestDateTimePersistence_CrossInstance_RealIntegration(t *testing.T) {
	// Test that datetime results persist across VM instances
	tester1 := NewIntegrationScriptTester(t)

	// Get current datetime and save it
	script1 := `
		getdatetime $capture_time
		saveVar $capture_time
		echo "Captured: " $capture_time
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Fatalf("Script execution failed: %v", result1.Error)
	}

	if len(result1.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result1.Output))
	}

	// Extract the captured time
	capturedOutput := result1.Output[0]
	if !strings.Contains(capturedOutput, "Captured: ") {
		t.Errorf("Expected 'Captured: ' prefix, got %s", capturedOutput)
	}

	// Create new VM instance sharing same database
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	// Load the saved time and use it
	script2 := `
		loadVar $capture_time
		echo "Loaded: " $capture_time
		datetimetostr $capture_time $formatted_time
		echo "Formatted loaded time: " $formatted_time
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Fatalf("Script execution in second VM failed: %v", result2.Error)
	}

	if len(result2.Output) != 2 {
		t.Errorf("Expected 2 output lines from second script, got %d", len(result2.Output))
	}

	// Verify the loaded time matches what was captured
	if len(result2.Output) > 0 {
		loadedOutput := result2.Output[0]
		if !strings.Contains(loadedOutput, "Loaded: ") {
			t.Errorf("Expected 'Loaded: ' prefix, got %s", loadedOutput)
		}
		
		// The loaded time should match the captured time
		capturedTime := strings.TrimPrefix(capturedOutput, "Captured: ")
		loadedTime := strings.TrimPrefix(loadedOutput, "Loaded: ")
		if capturedTime != loadedTime {
			t.Errorf("Captured time %s doesn't match loaded time %s", capturedTime, loadedTime)
		}
	}
}

func TestComplexDateTimeWorkflow_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	// Complex workflow combining multiple datetime operations
	script := `
		# Get current date and time
		getdate $today
		getdatetime $now
		
		# Calculate a simple time difference (should be 0)
		datetimediff $now $now "seconds" $zero_diff
		
		# Convert current datetime to custom format
		datetimetostr $now $formatted_time "YYYY-MM-DD HH:mm"
		
		echo "Today: " $today
		echo "Now: " $now
		echo "Zero diff: " $zero_diff
		echo "Formatted: " $formatted_time
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Complex datetime workflow failed: %v", result.Error)
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d: %v", len(result.Output), result.Output)
	}

	// Verify expected patterns in output
	patterns := []string{
		"Today: ",
		"Now: ",
		"Zero diff: 0",
		"Formatted: ",
	}

	for i, pattern := range patterns {
		if i < len(result.Output) {
			if !strings.Contains(result.Output[i], pattern) {
				t.Errorf("Output %d: expected to contain %q, got %q", i+1, pattern, result.Output[i])
			}
		}
	}
	
	// Verify date formats
	todayOutput := result.Output[0]
	if !strings.Contains(todayOutput, "/") {
		t.Errorf("Today output should contain date with slashes: %s", todayOutput)
	}

	nowOutput := result.Output[1]
	if !strings.Contains(nowOutput, "/") || !strings.Contains(nowOutput, ":") {
		t.Errorf("Now output should contain datetime with slashes and colons: %s", nowOutput)
	}

	formattedOutput := result.Output[3]
	if !strings.Contains(formattedOutput, "-") || !strings.Contains(formattedOutput, ":") {
		t.Errorf("Formatted output should contain custom format with dashes and colons: %s", formattedOutput)
	}
}