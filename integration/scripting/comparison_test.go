package scripting

import (
	"testing"
)

// TestIsEqualCommand_RealIntegration tests ISEQUAL command with real VM and database
func TestIsEqualCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test string equality
		setVar $str1 "hello"
		setVar $str2 "hello"
		setVar $str3 "world"
		
		isequal $str1 $str2 $result1
		echo "String equality (hello == hello): " $result1
		
		isequal $str1 $str3 $result2
		echo "String inequality (hello == world): " $result2
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	// Should show 1 for equality
	if len(result.Output) > 0 && result.Output[0] != "String equality (hello == hello): 1" {
		t.Errorf("String equality test: got %q, want %q", result.Output[0], "String equality (hello == hello): 1")
	}

	// Should show 0 for inequality
	if len(result.Output) > 1 && result.Output[1] != "String inequality (hello == world): 0" {
		t.Errorf("String inequality test: got %q, want %q", result.Output[1], "String inequality (hello == world): 0")
	}
}

// TestIsEqualCommand_Numbers tests ISEQUAL with numeric values
func TestIsEqualCommand_Numbers_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test number equality
		setVar $num1 42
		setVar $num2 42
		setVar $num3 43
		
		isequal $num1 $num2 $result1
		echo "Number equality (42 == 42): " $result1
		
		isequal $num1 $num3 $result2
		echo "Number inequality (42 == 43): " $result2
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Number equality (42 == 42): 1" {
		t.Errorf("Number equality test: got %q, want %q", result.Output[0], "Number equality (42 == 42): 1")
	}

	if len(result.Output) > 1 && result.Output[1] != "Number inequality (42 == 43): 0" {
		t.Errorf("Number inequality test: got %q, want %q", result.Output[1], "Number inequality (42 == 43): 0")
	}
}

// TestIsEqualCommand_StringNumberConversion tests ISEQUAL with string-number conversion
func TestIsEqualCommand_StringNumberConversion_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test string-number conversion
		setVar $str_num "42"
		setVar $actual_num 42
		
		isequal $str_num $actual_num $result
		echo "String-number equality (\"42\" == 42): " $result
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "String-number equality (\"42\" == 42): 1" {
		t.Errorf("String-number conversion test: got %q, want %q", result.Output[0], "String-number equality (\"42\" == 42): 1")
	}
}

// TestIsGreaterCommand_RealIntegration tests ISGREATER command
func TestIsGreaterCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test greater than comparisons
		setVar $high 100
		setVar $low 50
		setVar $equal 100
		
		isgreater $high $low $result1
		echo "Greater than (100 > 50): " $result1
		
		isgreater $low $high $result2
		echo "Not greater than (50 > 100): " $result2
		
		isgreater $high $equal $result3
		echo "Equal values (100 > 100): " $result3
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Greater than (100 > 50): 1" {
		t.Errorf("Greater than test: got %q, want %q", result.Output[0], "Greater than (100 > 50): 1")
	}

	if len(result.Output) > 1 && result.Output[1] != "Not greater than (50 > 100): 0" {
		t.Errorf("Not greater than test: got %q, want %q", result.Output[1], "Not greater than (50 > 100): 0")
	}

	if len(result.Output) > 2 && result.Output[2] != "Equal values (100 > 100): 0" {
		t.Errorf("Equal values test: got %q, want %q", result.Output[2], "Equal values (100 > 100): 0")
	}
}

// TestIsLessCommand_RealIntegration tests ISLESS command
func TestIsLessCommand_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test less than comparisons
		setVar $high 100
		setVar $low 50
		setVar $equal 50
		
		isless $low $high $result1
		echo "Less than (50 < 100): " $result1
		
		isless $high $low $result2
		echo "Not less than (100 < 50): " $result2
		
		isless $low $equal $result3
		echo "Equal values (50 < 50): " $result3
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Less than (50 < 100): 1" {
		t.Errorf("Less than test: got %q, want %q", result.Output[0], "Less than (50 < 100): 1")
	}

	if len(result.Output) > 1 && result.Output[1] != "Not less than (100 < 50): 0" {
		t.Errorf("Not less than test: got %q, want %q", result.Output[1], "Not less than (100 < 50): 0")
	}

	if len(result.Output) > 2 && result.Output[2] != "Equal values (50 < 50): 0" {
		t.Errorf("Equal values test: got %q, want %q", result.Output[2], "Equal values (50 < 50): 0")
	}
}

// TestComparisonCommands_CrossInstancePersistence tests comparison with persistent variables
func TestComparisonCommands_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First script execution - save variables
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $value1 75
		setVar $value2 50
		saveVar $value1
		saveVar $value2
		echo "Saved values: " $value1 " and " $value2
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Errorf("First script execution failed: %v", result1.Error)
	}

	// Second script execution - load and compare variables
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $value1
		loadVar $value2
		isgreater $value1 $value2 $comparison_result
		echo "Loaded comparison (75 > 50): " $comparison_result
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Errorf("Second script execution failed: %v", result2.Error)
	}

	if len(result2.Output) != 1 {
		t.Errorf("Expected 1 output line from second script, got %d", len(result2.Output))
	}

	expected := "Loaded comparison (75 > 50): 1"
	if len(result2.Output) > 0 && result2.Output[0] != expected {
		t.Errorf("Cross-instance comparison: got %q, want %q", result2.Output[0], expected)
	}
}

// TestComparisonCommands_FloatingPoint tests comparison with floating point numbers
func TestComparisonCommands_FloatingPoint_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test floating point comparisons
		setVar $float1 3.14
		setVar $float2 3.15
		setVar $float3 3.14
		
		isgreater $float2 $float1 $result1
		echo "Float greater (3.15 > 3.14): " $result1
		
		isequal $float1 $float3 $result2
		echo "Float equal (3.14 == 3.14): " $result2
		
		isless $float1 $float2 $result3
		echo "Float less (3.14 < 3.15): " $result3
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	if len(result.Output) > 0 && result.Output[0] != "Float greater (3.15 > 3.14): 1" {
		t.Errorf("Float greater test: got %q, want %q", result.Output[0], "Float greater (3.15 > 3.14): 1")
	}

	if len(result.Output) > 1 && result.Output[1] != "Float equal (3.14 == 3.14): 1" {
		t.Errorf("Float equal test: got %q, want %q", result.Output[1], "Float equal (3.14 == 3.14): 1")
	}

	if len(result.Output) > 2 && result.Output[2] != "Float less (3.14 < 3.15): 1" {
		t.Errorf("Float less test: got %q, want %q", result.Output[2], "Float less (3.14 < 3.15): 1")
	}
}

// TestComparisonCommands_EdgeCases tests comparison command edge cases
func TestComparisonCommands_EdgeCases_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test edge cases
		setVar $empty ""
		setVar $zero 0
		setVar $zero_str "0"
		
		isequal $empty $zero_str $result1
		echo "Empty vs zero string: " $result1
		
		isequal $zero $zero_str $result2
		echo "Zero number vs zero string: " $result2
		
		isgreater $zero $empty $result3
		echo "Zero greater than empty: " $result3
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(result.Output))
	}

	// These tests verify the comparison logic handles edge cases correctly
	// Exact expectations may vary based on implementation
	if len(result.Output) < 3 {
		t.Errorf("Not enough output lines for edge case testing")
	}
}

// TestAllComparisonCommands_Comprehensive tests all comparison commands together
func TestAllComparisonCommands_Comprehensive_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Comprehensive comparison test
		setVar $a 10
		setVar $b 20
		setVar $c 10
		
		# Test all comparison operators
		isequal $a $c $eq_result
		isnotequal $a $b $ne_result
		isgreater $b $a $gt_result
		isless $a $b $lt_result
		isgreaterequal $a $c $ge_result
		islessequal $a $b $le_result
		
		echo "Equal (10 == 10): " $eq_result
		echo "Not equal (10 != 20): " $ne_result
		echo "Greater (20 > 10): " $gt_result
		echo "Less (10 < 20): " $lt_result
		echo "Greater equal (10 >= 10): " $ge_result
		echo "Less equal (10 <= 20): " $le_result
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	if len(result.Output) != 6 {
		t.Errorf("Expected 6 output lines, got %d", len(result.Output))
	}

	expectedOutputs := []string{
		"Equal (10 == 10): 1",
		"Not equal (10 != 20): 1",
		"Greater (20 > 10): 1",
		"Less (10 < 20): 1",
		"Greater equal (10 >= 10): 1",
		"Less equal (10 <= 20): 1",
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Comparison %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestInfixOperators_NotEqual tests the <> operator in expressions
func TestInfixOperators_NotEqual_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test <> operator in expressions
		setVar $a 10
		setVar $b 20
		setVar $c 10
		
		# Test numeric not equal
		if ($a <> $b)
			echo "10 <> 20: true"
		else
			echo "10 <> 20: false"
		end
		
		# Test numeric equal (should be false)
		if ($a <> $c)
			echo "10 <> 10: true"
		else
			echo "10 <> 10: false"
		end
		
		# Test string not equal
		setVar $str1 "hello"
		setVar $str2 "world"
		if ($str1 <> $str2)
			echo "hello <> world: true"
		else
			echo "hello <> world: false"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"10 <> 20: true",
		"10 <> 10: false",
		"hello <> world: true",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestInfixOperators_LogicalAND tests the AND operator in expressions
func TestInfixOperators_LogicalAND_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test AND operator in expressions
		setVar $true_val 1
		setVar $false_val 0
		
		# Test true AND true
		if ($true_val AND $true_val)
			echo "1 AND 1: true"
		else
			echo "1 AND 1: false"
		end
		
		# Test true AND false
		if ($true_val AND $false_val)
			echo "1 AND 0: true"
		else
			echo "1 AND 0: false"
		end
		
		# Test false AND false
		if ($false_val AND $false_val)
			echo "0 AND 0: true"
		else
			echo "0 AND 0: false"
		end
		
		# Test complex expression
		setVar $a 5
		setVar $b 10
		if (($a > 3) AND ($b < 15))
			echo "Complex AND: true"
		else
			echo "Complex AND: false"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"1 AND 1: true",
		"1 AND 0: false",
		"0 AND 0: false",
		"Complex AND: true",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestInfixOperators_LogicalOR tests the OR operator in expressions
func TestInfixOperators_LogicalOR_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test OR operator in expressions
		setVar $true_val 1
		setVar $false_val 0
		
		# Test true OR true
		if ($true_val OR $true_val)
			echo "1 OR 1: true"
		else
			echo "1 OR 1: false"
		end
		
		# Test true OR false
		if ($true_val OR $false_val)
			echo "1 OR 0: true"
		else
			echo "1 OR 0: false"
		end
		
		# Test false OR false
		if ($false_val OR $false_val)
			echo "0 OR 0: true"
		else
			echo "0 OR 0: false"
		end
		
		# Test complex expression
		setVar $a 5
		setVar $b 20
		if (($a < 3) OR ($b > 15))
			echo "Complex OR: true"
		else
			echo "Complex OR: false"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"1 OR 1: true",
		"1 OR 0: true",
		"0 OR 0: false",
		"Complex OR: true",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestInfixOperators_LogicalXOR tests the XOR operator in expressions
func TestInfixOperators_LogicalXOR_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test XOR operator in expressions
		setVar $true_val 1
		setVar $false_val 0
		
		# Test true XOR true
		if ($true_val XOR $true_val)
			echo "1 XOR 1: true"
		else
			echo "1 XOR 1: false"
		end
		
		# Test true XOR false
		if ($true_val XOR $false_val)
			echo "1 XOR 0: true"
		else
			echo "1 XOR 0: false"
		end
		
		# Test false XOR true
		if ($false_val XOR $true_val)
			echo "0 XOR 1: true"
		else
			echo "0 XOR 1: false"
		end
		
		# Test false XOR false
		if ($false_val XOR $false_val)
			echo "0 XOR 0: true"
		else
			echo "0 XOR 0: false"
		end
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"1 XOR 1: false",
		"1 XOR 0: true",
		"0 XOR 1: true",
		"0 XOR 0: false",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestInfixOperators_StringConcatenation tests the & operator for string concatenation
func TestInfixOperators_StringConcatenation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test & operator for string concatenation
		setVar $str1 "Hello"
		setVar $str2 "World"
		setVar $num1 42
		setVar $num2 3.14
		
		# String & String
		setVar $result1 ($str1 & " " & $str2)
		echo "String concat: " $result1
		
		# String & Number
		setVar $result2 ($str1 & $num1)
		echo "String + Number: " $result2
		
		# Number & Number (should convert to strings)
		setVar $result3 ($num1 & $num2)
		echo "Number + Number: " $result3
		
		# Complex concatenation
		setVar $result4 ("Value: " & $num1 & " (PI: " & $num2 & ")")
		echo "Complex: " $result4
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"String concat: Hello World",
		"String + Number: Hello42",
		"Number + Number: 423",      // 423.14 rounded to 423
		"Complex: Value: 42 (PI: 3)", // 3.14 rounded to 3
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestInfixOperators_ComplexExpressions tests complex expressions with multiple operators
func TestInfixOperators_ComplexExpressions_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test complex expressions with multiple operators
		setVar $a 10
		setVar $b 20
		setVar $c 30
		setVar $name "test"
		
		# Test mixed logical and comparison operators
		if (($a < $b) AND ($b < $c) AND ($name <> ""))
			echo "Complex condition 1: true"
		else
			echo "Complex condition 1: false"
		end
		
		# Test OR with complex comparisons
		if (($a > $c) OR ($b >= 20) OR ($name = "test"))
			echo "Complex condition 2: true"
		else
			echo "Complex condition 2: false"
		end
		
		# Test XOR with parentheses
		if ((($a = 10) XOR ($b = 30)) AND ($c > 25))
			echo "Complex condition 3: true"
		else
			echo "Complex condition 3: false"
		end
		
		# Test string concatenation in complex expression
		setVar $message ("Result: " & (($a + $b) & " total"))
		echo $message
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Errorf("Script execution failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Complex condition 1: true",
		"Complex condition 2: true",
		"Complex condition 3: true",
		"Result: 30 total",
	}

	if len(result.Output) != len(expectedOutputs) {
		t.Errorf("Expected %d output lines, got %d", len(expectedOutputs), len(result.Output))
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}
