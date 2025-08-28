package scripting

import (
	"testing"
)

// TestTWXArithmetic_OriginalSyntax tests TWX arithmetic with in-place modification
func TestTWXArithmetic_OriginalSyntax_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test original TWX arithmetic syntax (in-place modification)
		setVar $counter 10
		add $counter 5
		echo "After add: " $counter
		
		subtract $counter 3
		echo "After subtract: " $counter
		
		multiply $counter 2
		echo "After multiply: " $counter
		
		setVar $divisor 4
		divide $counter $divisor
		echo "After divide: " $counter
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX arithmetic script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"After add: 15",      // 10 + 5
		"After subtract: 12", // 15 - 3
		"After multiply: 24", // 12 * 2
		"After divide: 6",    // 24 / 4
	}

	if len(result.Output) != 4 {
		t.Errorf("Expected 4 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("TWX arithmetic output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestTWXArithmetic_WithVariables tests arithmetic with variable operands
func TestTWXArithmetic_WithVariables_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $base 100
		setVar $increment 25
		setVar $multiplier 3
		
		add $base $increment
		echo "Base after increment: " $base
		
		multiply $base $multiplier
		echo "Base after multiply: " $base
		
		setVar $half 2
		divide $base $half
		echo "Base after halve: " $base
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("TWX variable arithmetic script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Base after increment: 125", // 100 + 25
		"Base after multiply: 375",  // 125 * 3
		"Base after halve: 188",     // 375 / 2 = 187.5, rounded to 188
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Variable arithmetic output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestArithmetic_DecimalPrecision tests arithmetic with decimal numbers
func TestArithmetic_DecimalPrecision_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		setVar $price 19.99
		setVar $tax_rate 0.08
		setVar $quantity 3
		
		setVar $subtotal $price
		multiply $subtotal $quantity
		
		setVar $tax $subtotal
		multiply $tax $tax_rate
		
		setVar $total $subtotal
		add $total $tax
		
		echo "Subtotal: " $subtotal
		echo "Tax: " $tax
		echo "Total: " $total
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Decimal arithmetic script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Subtotal: 60",    // 19.99 * 3 = 59.97, rounded to 60
		"Tax: 5",          // 59.97 * 0.08 = 4.7976, rounded to 5
		"Total: 65",       // 59.97 + 4.7976 = 64.7676, rounded to 65
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Decimal arithmetic output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestArithmetic_EdgeCases tests arithmetic edge cases
func TestArithmetic_EdgeCases_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test zero operations
		setVar $zero 0
		setVar $five 5
		
		setVar $result1 $zero
		add $result1 $five
		
		setVar $result2 $zero
		multiply $result2 $five
		
		setVar $result3 $five
		subtract $result3 $zero
		
		echo "0 + 5 = " $result1
		echo "0 multiply 5 = " $result2
		echo "5 - 0 = " $result3
		
		# Test negative numbers
		setVar $negative -10
		setVar $positive 3
		
		setVar $result4 $negative
		add $result4 $positive
		
		setVar $result5 $negative
		multiply $result5 $positive
		
		echo "-10 + 3 = " $result4
		echo "-10 multiply 3 = " $result5
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("Arithmetic edge cases script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"0 + 5 = 5",
		"0 multiply 5 = 0",
		"5 - 0 = 5",
		"-10 + 3 = -7",
		"-10 multiply 3 = -30",
	}

	if len(result.Output) != 5 {
		t.Errorf("Expected 5 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("Edge case output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestSetVarArithmeticExpressions tests SETVAR with complex arithmetic expressions (original issue)
func TestSetVarArithmeticExpressions_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)

	script := `
		# Test the original failing case: setVar $price (($startPrice * $buyPerc) / 100)
		setVar $startPrice 100.0
		setVar $buyPerc 15
		setVar $price (($startPrice * $buyPerc) / 100)
		echo "Price calculation: " $price
		
		# Test more complex arithmetic expressions
		setVar $base 50
		setVar $multiplier 3
		setVar $divisor 2
		setVar $result ((($base * $multiplier) + 10) / $divisor)
		echo "Complex calculation: " $result
		
		# Test with mixed numeric/string variables
		setVar $num1 "25"
		setVar $num2 4
		setVar $mixed ($num1 * $num2)
		echo "Mixed calculation: " $mixed
	`

	result := tester.ExecuteScript(script)
	if result.Error != nil {
		t.Fatalf("SETVAR arithmetic expressions script failed: %v", result.Error)
	}

	expectedOutputs := []string{
		"Price calculation: 15",       // (100.0 * 15) / 100 = 1500 / 100 = 15
		"Complex calculation: 80",     // (((50 * 3) + 10) / 2) = ((150 + 10) / 2) = (160 / 2) = 80
		"Mixed calculation: 100",      // "25" * 4 = 100 (string "25" converted to number)
	}

	if len(result.Output) != 3 {
		t.Errorf("Expected 3 output lines, got %d: %v", len(result.Output), result.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result.Output) && result.Output[i] != expected {
			t.Errorf("SETVAR arithmetic output %d: got %q, want %q", i+1, result.Output[i], expected)
		}
	}
}

// TestArithmetic_CrossInstancePersistence tests arithmetic with persistent variables
func TestArithmetic_CrossInstancePersistence_RealIntegration(t *testing.T) {
	// First instance: Initialize and calculate
	tester1 := NewIntegrationScriptTester(t)

	script1 := `
		setVar $account_balance 1000
		setVar $transaction_amount 150
		
		subtract $account_balance $transaction_amount
		saveVar $account_balance
		
		echo "Balance after transaction: " $account_balance
	`

	result1 := tester1.ExecuteScript(script1)
	if result1.Error != nil {
		t.Fatalf("First arithmetic persistence script failed: %v", result1.Error)
	}

	// Second instance: Load and continue calculating
	tester2 := NewIntegrationScriptTesterWithSharedDB(t, tester1.setupData)

	script2 := `
		loadVar $account_balance
		
		setVar $interest_rate 0.02
		setVar $interest $account_balance
		multiply $interest $interest_rate
		add $account_balance $interest
		
		echo "Balance after interest: " $account_balance
		echo "Interest earned: " $interest
	`

	result2 := tester2.ExecuteScript(script2)
	if result2.Error != nil {
		t.Fatalf("Second arithmetic persistence script failed: %v", result2.Error)
	}

	expectedOutputs := []string{
		"Balance after interest: 867", // 850 + (850 * 0.02) = 850 + 17 = 867
		"Interest earned: 17",         // 850 * 0.02 = 17
	}

	if len(result2.Output) != 2 {
		t.Errorf("Expected 2 output lines, got %d: %v", len(result2.Output), result2.Output)
	}

	for i, expected := range expectedOutputs {
		if i < len(result2.Output) && result2.Output[i] != expected {
			t.Errorf("Arithmetic persistence output %d: got %q, want %q", i+1, result2.Output[i], expected)
		}
	}
}
