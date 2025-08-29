package parser

import (
	"strings"
	"testing"
)

func TestLineNumberTracking(t *testing.T) {
	tests := []struct {
		name           string
		script         string
		expectedTokens map[int]int // token index -> expected original line number
	}{
		{
			name: "simple script without macros",
			script: `# Comment on line 1
echo "Line 2"
goto :label
:label
echo "Line 5"`,
			expectedTokens: map[int]int{
				0: 1, // Comment token
				1: 1, // Newline after comment
				2: 2, // "echo" token on line 2
				3: 2, // String token on line 2
				4: 2, // Newline after echo
				5: 3, // "goto" token on line 3
			},
		},
		{
			name: "script with IF macro expansion",
			script: `echo "Line 1"
IF 1 = 1
  echo "Line 3 inside IF"
ELSE
  echo "Line 5 inside ELSE"
END
echo "Line 7 after IF"`,
			expectedTokens: map[int]int{
				0: 1, // "echo" token on original line 1
				// After preprocessing, the IF becomes BRANCH command but should track back to line 2
			},
		},
		{
			name: "script with WHILE macro expansion",
			script: `echo "Line 1"
WHILE 1 = 0
  echo "Line 3 inside WHILE"
END
echo "Line 5 after WHILE"`,
			expectedTokens: map[int]int{
				0: 1, // "echo" token on original line 1
				// After preprocessing, WHILE becomes labels/BRANCH but should track back to line 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Preprocess the script
			lines := strings.Split(tt.script, "\n")
			preprocessor := NewPreprocessor()
			processedLines, err := preprocessor.ProcessScript(lines)
			if err != nil {
				t.Fatalf("Preprocessing failed: %v", err)
			}

			// Step 2: Create lexer with line mappings
			processedSource := strings.Join(processedLines, "\n")
			lexer := NewLexer(strings.NewReader(processedSource), preprocessor.GetLineMappings())

			// Step 3: Tokenize and verify line numbers
			tokens, err := lexer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			// Verify specific tokens have correct original line numbers
			for tokenIndex, expectedLine := range tt.expectedTokens {
				if tokenIndex >= len(tokens) {
					t.Errorf("Token index %d out of range, only have %d tokens", tokenIndex, len(tokens))
					continue
				}
				
				actualLine := tokens[tokenIndex].Line
				if actualLine != expectedLine {
					t.Errorf("Token %d: expected line %d, got line %d (token: %+v)", 
						tokenIndex, expectedLine, actualLine, tokens[tokenIndex])
				}
			}

			// Debug output to see all tokens and their line numbers
			t.Logf("Processed script:\n%s", processedSource)
			t.Logf("Line mappings: %+v", preprocessor.GetLineMappings())
			for i, token := range tokens {
				if i < 10 { // Show first 10 tokens
					t.Logf("Token %d: Line %d, Type %v, Value %q", i, token.Line, token.Type, token.Value)
				}
			}
		})
	}
}

func TestPreprocessorLineMappings(t *testing.T) {
	script := `echo "Line 1"
IF 1 = 1
  echo "Line 3"
END
echo "Line 5"`

	lines := strings.Split(script, "\n")
	preprocessor := NewPreprocessor()
	processedLines, err := preprocessor.ProcessScript(lines)
	if err != nil {
		t.Fatalf("Preprocessing failed: %v", err)
	}

	mappings := preprocessor.GetLineMappings()
	
	// Verify we have mappings
	if len(mappings) == 0 {
		t.Fatal("No line mappings generated")
	}

	// Log the mappings for debugging
	t.Logf("Original script:\n%s", script)
	t.Logf("Processed script:\n%s", strings.Join(processedLines, "\n"))
	t.Logf("Line mappings:")
	for _, mapping := range mappings {
		t.Logf("  Processed line %d -> Original line %d", mapping.ProcessedLine, mapping.OriginalLine)
	}

	// Verify that original lines are preserved
	hasLine1 := false
	hasLine2 := false
	hasLine3 := false
	hasLine5 := false
	
	for _, mapping := range mappings {
		switch mapping.OriginalLine {
		case 1:
			hasLine1 = true
		case 2:
			hasLine2 = true
		case 3:
			hasLine3 = true
		case 5:
			hasLine5 = true
		}
	}

	if !hasLine1 {
		t.Error("Missing mapping for original line 1")
	}
	if !hasLine2 {
		t.Error("Missing mapping for original line 2 (IF)")
	}
	if !hasLine3 {
		t.Error("Missing mapping for original line 3")
	}
	if !hasLine5 {
		t.Error("Missing mapping for original line 5")
	}
}

func TestASTNodeLineNumbers(t *testing.T) {
	script := `echo "Line 1"
IF 1 = 1
  echo "Line 3"
END`

	// Preprocess
	lines := strings.Split(script, "\n")
	preprocessor := NewPreprocessor()
	processedLines, err := preprocessor.ProcessScript(lines)
	if err != nil {
		t.Fatalf("Preprocessing failed: %v", err)
	}

	// Tokenize
	processedSource := strings.Join(processedLines, "\n")
	lexer := NewLexer(strings.NewReader(processedSource), preprocessor.GetLineMappings())
	tokens, err := lexer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	// Parse
	parser := NewParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}

	// Verify AST nodes have line numbers
	if ast.Line == 0 {
		t.Error("Program AST node should have line number")
	}

	// Check that child nodes have line numbers
	for i, child := range ast.Children {
		if child.Line == 0 {
			t.Errorf("Child node %d has no line number: %+v", i, child)
		}
		t.Logf("AST Node %d: Type %v, Line %d, Value %q", i, child.Type, child.Line, child.Value)
	}
}