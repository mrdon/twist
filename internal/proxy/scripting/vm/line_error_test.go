package vm

import (
	"strings"
	"testing"
	"twist/internal/proxy/scripting/parser"
)

func TestVMErrorReportingWithLineNumbers(t *testing.T) {
	// Create a script that will cause an error
	script := `echo "Line 1 works"
invalid_command "This should error"
echo "Line 3 won't execute"`

	// Parse the script
	lines := strings.Split(script, "\n")
	preprocessor := parser.NewPreprocessor()
	processedLines, err := preprocessor.ProcessScript(lines)
	if err != nil {
		t.Fatalf("Preprocessing failed: %v", err)
	}

	processedSource := strings.Join(processedLines, "\n")
	lexer := parser.NewLexer(strings.NewReader(processedSource), preprocessor.GetLineMappings())
	
	tokens, err := lexer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	parser := parser.NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}

	// Create a VM and try to execute
	mockGameInterface := &MockGameInterface{}
	vm := NewVirtualMachine(mockGameInterface)
	err = vm.LoadScript(ast, nil) // Use nil for script interface
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	// Execute the script - should get an error with line number
	err = vm.Execute()
	
	// We expect an error since invalid_command is not a valid command
	if err == nil {
		t.Fatal("Expected an error from invalid_command, but got none")
	}

	// Check if the error contains line number information
	errorString := err.Error()
	t.Logf("Error message: %s", errorString)

	// The error should mention line 2 (where invalid_command is)
	if !strings.Contains(errorString, "line 2") {
		t.Errorf("Error should mention line 2, but got: %s", errorString)
	}

	// The error should contain the actual command that failed
	if !strings.Contains(strings.ToLower(errorString), "invalid_command") {
		t.Errorf("Error should mention invalid_command, but got: %s", errorString)
	}
}

func TestVMPanicRecoveryWithLineNumbers(t *testing.T) {
	// Create a minimal AST that might cause issues
	ast := &parser.ASTNode{
		Type:     parser.NodeProgram,
		Line:     1,
		Column:   1,
		Children: []*parser.ASTNode{
			{
				Type:   parser.NodeCommand,
				Value:  "test_panic",
				Line:   42, // This line number should show up in panic recovery
				Column: 1,
			},
		},
	}

	vm := NewVirtualMachine(&MockGameInterface{})
	err := vm.LoadScript(ast, nil)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	// This will likely cause a panic due to unimplemented command
	// The panic should be recovered and include line number
	err = vm.Execute()
	
	if err == nil {
		t.Fatal("Expected an error from test_panic command")
	}

	errorString := err.Error()
	t.Logf("Panic error message: %s", errorString)

	// Should include line number from the AST node
	if !strings.Contains(errorString, "line 42") {
		t.Errorf("Error should mention line 42, but got: %s", errorString)
	}
}