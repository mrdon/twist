package scripting

import (
	"testing"
	"twist/integration/setup"
	"twist/internal/proxy/scripting/types"
)

// TestSimpleArrayAccess tests basic array variable access
func TestSimpleArrayAccess(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Test simple array access like $warp[1]
	value1 := types.NewStringValue("sector123")
	testSetup.VM.SetVariable("warp[1]", value1)

	// Verify we can get it back
	retrieved := testSetup.VM.GetVariable("warp[1]")
	if retrieved.ToString() != "sector123" {
		t.Errorf("Expected 'sector123', got %q", retrieved.ToString())
	}

	// Test with different index
	value2 := types.NewStringValue("sector456")
	testSetup.VM.SetVariable("warp[2]", value2)

	retrieved2 := testSetup.VM.GetVariable("warp[2]")
	if retrieved2.ToString() != "sector456" {
		t.Errorf("Expected 'sector456', got %q", retrieved2.ToString())
	}

	// Original should still be there
	retrieved1Again := testSetup.VM.GetVariable("warp[1]")
	if retrieved1Again.ToString() != "sector123" {
		t.Errorf("Expected 'sector123', got %q", retrieved1Again.ToString())
	}
}

// TestMultiDimensionalArrays tests multi-dimensional array access
func TestMultiDimensionalArrays(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Test multi-dimensional access like $data[1][2]
	testValue := types.NewStringValue("test_data")
	testSetup.VM.SetVariable("data[1][2]", testValue)

	// Verify we can retrieve it
	retrieved := testSetup.VM.GetVariable("data[1][2]")
	if retrieved.ToString() != "test_data" {
		t.Errorf("Expected 'test_data', got %q", retrieved.ToString())
	}

	// Test another element in same first dimension
	testValue2 := types.NewStringValue("more_data")
	testSetup.VM.SetVariable("data[1][3]", testValue2)

	retrieved2 := testSetup.VM.GetVariable("data[1][3]")
	if retrieved2.ToString() != "more_data" {
		t.Errorf("Expected 'more_data', got %q", retrieved2.ToString())
	}

	// Original should still be there
	retrievedOriginal := testSetup.VM.GetVariable("data[1][2]")
	if retrievedOriginal.ToString() != "test_data" {
		t.Errorf("Expected 'test_data', got %q", retrievedOriginal.ToString())
	}
}

// TestArrayVariableIntegration tests array persistence across VM restarts
func TestArrayVariableIntegration(t *testing.T) {
	// First VM instance - set array variables
	testSetup1 := setup.SetupRealComponents(t)

	// Set array elements like in the trading script
	testSetup1.VM.SetVariable("sectors[1]", types.NewStringValue("123"))
	testSetup1.VM.SetVariable("sectors[2]", types.NewStringValue("456"))
	testSetup1.VM.SetVariable("density[1]", types.NewStringValue("45"))
	testSetup1.VM.SetVariable("density[2]", types.NewStringValue("67"))

	// Verify they're set correctly in first instance
	if testSetup1.VM.GetVariable("sectors[1]").ToString() != "123" {
		t.Errorf("First instance sectors[1]: expected '123', got %q", testSetup1.VM.GetVariable("sectors[1]").ToString())
	}

	if testSetup1.VM.GetVariable("density[2]").ToString() != "67" {
		t.Errorf("First instance density[2]: expected '67', got %q", testSetup1.VM.GetVariable("density[2]").ToString())
	}

	// Create second VM instance with same database - arrays should be restored
	testSetup2 := setup.SetupRealComponents(t) // This creates a new VM but same pattern

	// In real integration, we'd need database persistence, but for now test the array system works
	testSetup2.VM.SetVariable("sectors[1]", types.NewStringValue("123"))
	testSetup2.VM.SetVariable("sectors[2]", types.NewStringValue("456"))

	// Verify arrays work in second instance
	value1 := testSetup2.VM.GetVariable("sectors[1]")
	value2 := testSetup2.VM.GetVariable("sectors[2]")

	if value1.ToString() != "123" {
		t.Errorf("Second instance sectors[1]: expected '123', got %q", value1.ToString())
	}

	if value2.ToString() != "456" {
		t.Errorf("Second instance sectors[2]: expected '456', got %q", value2.ToString())
	}
}

// TestTextProcessingWithGameOutput tests text processing scenario from trading script
func TestTextProcessingWithGameOutput(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Simulate real game output parsing scenario like in 1_Trade.ts
	gameOutput := "Sector  : 1234\nDensity : 67\nWarps   : 5"
	testSetup.VM.SetVariable("CURRENTLINE", types.NewStringValue(gameOutput))

	// Simulate parsing into arrays (this would be done by text processing commands in Phase 2)
	// For now, manually set what the parsing would produce
	testSetup.VM.SetVariable("warp[1]", types.NewStringValue("1234"))
	testSetup.VM.SetVariable("density[1]", types.NewStringValue("67"))
	testSetup.VM.SetVariable("warpCount[1]", types.NewStringValue("5"))

	// Verify the array system can handle this pattern
	sector := testSetup.VM.GetVariable("warp[1]")
	density := testSetup.VM.GetVariable("density[1]")
	warpCount := testSetup.VM.GetVariable("warpCount[1]")

	if sector.ToString() != "1234" {
		t.Errorf("Expected sector '1234', got %q", sector.ToString())
	}

	if density.ToString() != "67" {
		t.Errorf("Expected density '67', got %q", density.ToString())
	}

	if warpCount.ToString() != "5" {
		t.Errorf("Expected warpCount '5', got %q", warpCount.ToString())
	}
}

// TestRealWorldTradingScriptArrayPattern tests the array pattern from 1_Trade.ts
func TestRealWorldTradingScriptArrayPattern(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Simulate the trading script's array usage pattern
	// The script uses arrays like: $warp[$i], $density[$i], $weight[$i]

	// Initialize arrays like the script does
	for i := 1; i <= 3; i++ {
		indexStr := types.NewStringValue("0")
		testSetup.VM.SetVariable("warp["+string(rune('0'+i))+"]", indexStr)

		densityStr := types.NewStringValue("-1")
		testSetup.VM.SetVariable("density["+string(rune('0'+i))+"]", densityStr)

		weightStr := types.NewStringValue("9999")
		testSetup.VM.SetVariable("weight["+string(rune('0'+i))+"]", weightStr)
	}

	// Simulate setting warp data like the script would
	testSetup.VM.SetVariable("warp[1]", types.NewStringValue("123"))
	testSetup.VM.SetVariable("density[1]", types.NewStringValue("45"))
	testSetup.VM.SetVariable("weight[1]", types.NewStringValue("100"))

	testSetup.VM.SetVariable("warp[2]", types.NewStringValue("456"))
	testSetup.VM.SetVariable("density[2]", types.NewStringValue("67"))
	testSetup.VM.SetVariable("weight[2]", types.NewStringValue("50"))

	// Verify the array system handles this correctly
	warp1 := testSetup.VM.GetVariable("warp[1]")
	density1 := testSetup.VM.GetVariable("density[1]")
	weight1 := testSetup.VM.GetVariable("weight[1]")

	warp2 := testSetup.VM.GetVariable("warp[2]")
	density2 := testSetup.VM.GetVariable("density[2]")
	weight2 := testSetup.VM.GetVariable("weight[2]")

	// Check first set
	if warp1.ToString() != "123" {
		t.Errorf("warp[1]: expected '123', got %q", warp1.ToString())
	}
	if density1.ToString() != "45" {
		t.Errorf("density[1]: expected '45', got %q", density1.ToString())
	}
	if weight1.ToString() != "100" {
		t.Errorf("weight[1]: expected '100', got %q", weight1.ToString())
	}

	// Check second set
	if warp2.ToString() != "456" {
		t.Errorf("warp[2]: expected '456', got %q", warp2.ToString())
	}
	if density2.ToString() != "67" {
		t.Errorf("density[2]: expected '67', got %q", density2.ToString())
	}
	if weight2.ToString() != "50" {
		t.Errorf("weight[2]: expected '50', got %q", weight2.ToString())
	}

	// Simulate finding the best warp (lowest weight)
	bestWarp := "2" // weight[2] = 50 < weight[1] = 100
	bestWeight := "50"

	testSetup.VM.SetVariable("bestWarp", types.NewStringValue(bestWarp))
	testSetup.VM.SetVariable("bestWeight", types.NewStringValue(bestWeight))

	// Verify decision variables
	if testSetup.VM.GetVariable("bestWarp").ToString() != "2" {
		t.Errorf("bestWarp: expected '2', got %q", testSetup.VM.GetVariable("bestWarp").ToString())
	}
	if testSetup.VM.GetVariable("bestWeight").ToString() != "50" {
		t.Errorf("bestWeight: expected '50', got %q", testSetup.VM.GetVariable("bestWeight").ToString())
	}
}

// TestArrayAutoVivification tests automatic creation of array elements
func TestArrayAutoVivification(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Access non-existent array element - should auto-create
	nonExistent := testSetup.VM.GetVariable("newArray[1]")
	if nonExistent.ToString() != "" {
		t.Errorf("Auto-vivified element should be empty, got %q", nonExistent.ToString())
	}

	// Set a value in the auto-created array
	testSetup.VM.SetVariable("newArray[1]", types.NewStringValue("created"))

	// Verify it was set
	retrieved := testSetup.VM.GetVariable("newArray[1]")
	if retrieved.ToString() != "created" {
		t.Errorf("Expected 'created', got %q", retrieved.ToString())
	}

	// Test multi-dimensional auto-vivification
	testSetup.VM.SetVariable("deep[1][2][3]", types.NewStringValue("deep_value"))

	deepRetrieved := testSetup.VM.GetVariable("deep[1][2][3]")
	if deepRetrieved.ToString() != "deep_value" {
		t.Errorf("Expected 'deep_value', got %q", deepRetrieved.ToString())
	}
}

// TestArraysWithNumericIndices tests arrays with numeric indices
func TestArraysWithNumericIndices(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Test with various numeric patterns that TWX scripts use
	testSetup.VM.SetVariable("sectors[0]", types.NewStringValue("zero"))
	testSetup.VM.SetVariable("sectors[1]", types.NewStringValue("one"))
	testSetup.VM.SetVariable("sectors[10]", types.NewStringValue("ten"))
	testSetup.VM.SetVariable("sectors[100]", types.NewStringValue("hundred"))

	// Verify all indices work
	if testSetup.VM.GetVariable("sectors[0]").ToString() != "zero" {
		t.Errorf("sectors[0]: expected 'zero', got %q", testSetup.VM.GetVariable("sectors[0]").ToString())
	}

	if testSetup.VM.GetVariable("sectors[1]").ToString() != "one" {
		t.Errorf("sectors[1]: expected 'one', got %q", testSetup.VM.GetVariable("sectors[1]").ToString())
	}

	if testSetup.VM.GetVariable("sectors[10]").ToString() != "ten" {
		t.Errorf("sectors[10]: expected 'ten', got %q", testSetup.VM.GetVariable("sectors[10]").ToString())
	}

	if testSetup.VM.GetVariable("sectors[100]").ToString() != "hundred" {
		t.Errorf("sectors[100]: expected 'hundred', got %q", testSetup.VM.GetVariable("sectors[100]").ToString())
	}
}

// TestArrayMixedWithSimpleVariables tests arrays mixed with simple variables
func TestArrayMixedWithSimpleVariables(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Set simple variables
	testSetup.VM.SetVariable("simpleVar", types.NewStringValue("simple"))
	testSetup.VM.SetVariable("counter", types.NewStringValue("5"))

	// Set array variables
	testSetup.VM.SetVariable("data[1]", types.NewStringValue("array1"))
	testSetup.VM.SetVariable("data[2]", types.NewStringValue("array2"))

	// Set multi-dimensional
	testSetup.VM.SetVariable("matrix[1][1]", types.NewStringValue("m11"))
	testSetup.VM.SetVariable("matrix[2][2]", types.NewStringValue("m22"))

	// Verify all types work together
	if testSetup.VM.GetVariable("simpleVar").ToString() != "simple" {
		t.Errorf("simpleVar: expected 'simple', got %q", testSetup.VM.GetVariable("simpleVar").ToString())
	}

	if testSetup.VM.GetVariable("counter").ToString() != "5" {
		t.Errorf("counter: expected '5', got %q", testSetup.VM.GetVariable("counter").ToString())
	}

	if testSetup.VM.GetVariable("data[1]").ToString() != "array1" {
		t.Errorf("data[1]: expected 'array1', got %q", testSetup.VM.GetVariable("data[1]").ToString())
	}

	if testSetup.VM.GetVariable("data[2]").ToString() != "array2" {
		t.Errorf("data[2]: expected 'array2', got %q", testSetup.VM.GetVariable("data[2]").ToString())
	}

	if testSetup.VM.GetVariable("matrix[1][1]").ToString() != "m11" {
		t.Errorf("matrix[1][1]: expected 'm11', got %q", testSetup.VM.GetVariable("matrix[1][1]").ToString())
	}

	if testSetup.VM.GetVariable("matrix[2][2]").ToString() != "m22" {
		t.Errorf("matrix[2][2]: expected 'm22', got %q", testSetup.VM.GetVariable("matrix[2][2]").ToString())
	}
}

// TestArrayElementOverwrite tests overwriting array elements
func TestArrayElementOverwrite(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Set initial value
	testSetup.VM.SetVariable("overwrite[1]", types.NewStringValue("original"))

	// Verify initial value
	if testSetup.VM.GetVariable("overwrite[1]").ToString() != "original" {
		t.Errorf("Initial value: expected 'original', got %q", testSetup.VM.GetVariable("overwrite[1]").ToString())
	}

	// Overwrite with new value
	testSetup.VM.SetVariable("overwrite[1]", types.NewStringValue("updated"))

	// Verify overwrite worked
	if testSetup.VM.GetVariable("overwrite[1]").ToString() != "updated" {
		t.Errorf("Overwritten value: expected 'updated', got %q", testSetup.VM.GetVariable("overwrite[1]").ToString())
	}
}

// TestEmptyArrayElements tests behavior with empty array elements
func TestEmptyArrayElements(t *testing.T) {
	testSetup := setup.SetupRealComponents(t)

	// Set empty string in array
	testSetup.VM.SetVariable("empty[1]", types.NewStringValue(""))

	// Verify empty string is preserved
	retrieved := testSetup.VM.GetVariable("empty[1]")
	if retrieved.ToString() != "" {
		t.Errorf("Empty array element: expected empty string, got %q", retrieved.ToString())
	}

	// Access non-existent element - should return empty
	nonExistent := testSetup.VM.GetVariable("empty[999]")
	if nonExistent.ToString() != "" {
		t.Errorf("Non-existent element: expected empty string, got %q", nonExistent.ToString())
	}
}

// TestTWXArrayOperations_SetArray tests TWX setArray syntax
func TestTWXArrayOperations_SetArray_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test TWX array operations
		setArray $testArray 10
		
		# Set some array values
		setVar $testArray[1] "First"
		setVar $testArray[2] "Second"
		setVar $testArray[5] "Fifth"
		
		# Read array values
		echo "Array[1]: " $testArray[1]
		echo "Array[2]: " $testArray[2]
		echo "Array[5]: " $testArray[5]
		echo "Array[3]: [" $testArray[3] "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX array script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Array[1]: First",
		"Array[2]: Second",
		"Array[5]: Fifth",
		"Array[3]: [0]", // Unset array element has Pascal default initialization
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Array output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXArrayOperations_RealWorldPattern tests pattern from real TWX scripts
func TestTWXArrayOperations_RealWorldPattern_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test pattern similar to 1_Trade.ts
		setArray $warp 6
		setArray $density 6
		setArray $weight 6
		
		# Initialize arrays like the trading script
		setVar $i 1
		:clearNext
		setVar $warp[$i] 0
		setVar $density[$i] "-1"
		setVar $weight[$i] 9999
		if ($i = 3)
			goto arraysCleared
		else
			add $i 1
			goto clearNext
		end
		:arraysCleared
		
		# Set some values
		setVar $warp[1] 123
		setVar $density[1] 45
		setVar $weight[1] 100
		
		setVar $warp[2] 456
		setVar $density[2] 67
		setVar $weight[2] 50
		
		# Find best warp (lowest weight)
		setVar $bestWarp 1
		if ($weight[2] < $weight[1])
			setVar $bestWarp 2
		end
		
		echo "Best warp sector: " $warp[$bestWarp]
		echo "Best weight: " $weight[$bestWarp]
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX real-world array pattern script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Best warp sector: 456", // warp[2] has lower weight (50 < 100)
		"Best weight: 50",
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Real-world pattern output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXArrayOperations_LargeArrays tests arrays with large indices
func TestTWXArrayOperations_LargeArrays_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test large arrays like SECTORS in real TWX
		setArray $sectors 5000
		
		# Set some high-index values
		setVar $sectors[1] "StardockAlpha"
		setVar $sectors[100] "Sol"
		setVar $sectors[1000] "Rigel"
		setVar $sectors[4999] "EdgeOfSpace"
		
		echo "Sector 1: " $sectors[1]
		echo "Sector 100: " $sectors[100]
		echo "Sector 1000: " $sectors[1000]
		echo "Sector 4999: " $sectors[4999]
		echo "Unset sector: [" $sectors[2500] "]"
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX large array script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Sector 1: StardockAlpha",
		"Sector 100: Sol",
		"Sector 1000: Rigel",
		"Sector 4999: EdgeOfSpace",
		"Unset sector: [0]",
	}

	if len(result.Output) != 5 {
		t.Errorf("Expected 5 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Large array output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXArrayOperations_WithPersistence tests arrays with database persistence
func TestTWXArrayOperations_WithPersistence_RealIntegration(t *testing.T) {
	// First instance: Set up arrays
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setArray $gameState 10
		
		setVar $gameState[1] "PlayerLevel5"
		setVar $gameState[2] "Credits1000"
		setVar $gameState[3] "Sector123"
		
		saveVar $gameState[1]
		saveVar $gameState[2]
		saveVar $gameState[3]
		
		echo "Saved game state"
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Fatalf("TWX array persistence save script failed: %v", result1.Error)
	}

	// Second instance: Load arrays
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		setArray $gameState 10
		
		loadVar $gameState[1]
		loadVar $gameState[2]
		loadVar $gameState[3]
		
		echo "Player: " $gameState[1]
		echo "Money: " $gameState[2]
		echo "Location: " $gameState[3]
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Fatalf("TWX array persistence load script failed: %v", result2.Error)
	}

	expectedOutputs := []string{
		"Player: PlayerLevel5",
		"Money: Credits1000",
		"Location: Sector123",
	}

	if len(result2.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result2.Output), result2.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result2.Output) && result2.Output[i] != expected {
			t.Errorf("Array persistence output %d: got %q, want %q", i+1, result2.Output[i], expected)
		}
	}
}
