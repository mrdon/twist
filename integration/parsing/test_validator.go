package parsing

import (
	"reflect"
	"testing"
)

// AssertAPICallsEqual compares expected vs actual API calls
func AssertAPICallsEqual(t *testing.T, mockAPI *MockTuiAPI, expected []string) {
	actual := mockAPI.GetCalls()

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("API calls mismatch:\nExpected: %v\nActual:   %v", expected, actual)
		return
	}

	t.Logf("✓ All %d API calls validated", len(expected))
}
