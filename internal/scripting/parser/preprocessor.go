package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// ConditionStruct tracks control flow blocks (IF/WHILE) during preprocessing
// This mirrors the TConditionStruct from TWX ScriptCmp.pas
type ConditionStruct struct {
	ConLabel string // Label to jump to on condition false (IF) or loop continuation (WHILE)
	EndLabel string // Label for end of block
	IsWhile  bool   // true for WHILE blocks, false for IF blocks
	AtElse   bool   // true if we've seen ELSE in this IF block
}

// Preprocessor handles macro expansion (IF/ELSE/END, WHILE/END) 
// This mirrors the functionality in TWX ScriptCmp.pas
type Preprocessor struct {
	ifStack      []*ConditionStruct // Stack of nested control structures
	ifLabelCount int                // Counter for generating unique labels
	output       []string           // Preprocessed output lines
}

// NewPreprocessor creates a new preprocessor instance
func NewPreprocessor() *Preprocessor {
	return &Preprocessor{
		ifStack:      make([]*ConditionStruct, 0),
		ifLabelCount: 0,
		output:       make([]string, 0),
	}
}

// ProcessScript preprocesses a script, expanding IF/ELSE/END and WHILE/END macros
// This mirrors the RecurseCmd functionality from TWX ScriptCmp.pas
func (p *Preprocessor) ProcessScript(lines []string) ([]string, error) {
	p.output = make([]string, 0)
	p.ifStack = make([]*ConditionStruct, 0)
	p.ifLabelCount = 0

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			p.output = append(p.output, line)
			continue
		}

		// Parse the command and parameters
		parts := strings.Fields(line)
		if len(parts) == 0 {
			p.output = append(p.output, line)
			continue
		}

		cmd := strings.ToUpper(parts[0])
		
		// Process macro commands
		switch cmd {
		case "IF":
			if err := p.processIf(parts, lineNum+1); err != nil {
				return nil, err
			}
		case "ELSE":
			if err := p.processElse(parts, lineNum+1); err != nil {
				return nil, err
			}
		case "ELSEIF":
			if err := p.processElseIf(parts, lineNum+1); err != nil {
				return nil, err
			}
		case "WHILE":
			if err := p.processWhile(parts, lineNum+1); err != nil {
				return nil, err
			}
		case "END":
			if err := p.processEnd(parts, lineNum+1); err != nil {
				return nil, err
			}
		default:
			// Not a macro command, pass through unchanged
			p.output = append(p.output, line)
		}
	}

	// Ensure all IF/WHILE blocks are properly closed
	if len(p.ifStack) != 0 {
		return nil, fmt.Errorf("IF/WHILE .. END structure mismatch")
	}

	return p.output, nil
}

// processIf handles IF macro expansion
// Equivalent to the 'IF' case in TWX ScriptCmp.pas RecurseCmd
func (p *Preprocessor) processIf(parts []string, lineNum int) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: no parameters to compare with IF macro", lineNum)
	}

	// Create new condition structure
	conStruct := &ConditionStruct{
		AtElse:  false,
		IsWhile: false,
	}
	
	// Generate unique labels
	p.ifLabelCount++
	conStruct.ConLabel = "::" + strconv.Itoa(p.ifLabelCount)
	p.ifLabelCount++
	conStruct.EndLabel = "::" + strconv.Itoa(p.ifLabelCount)
	
	// Push onto stack
	p.ifStack = append(p.ifStack, conStruct)
	
	// Generate BRANCH command: if condition is false, jump to ConLabel
	condition := strings.Join(parts[1:], " ")
	// Escape any quotes in the condition string
	escapedCondition := strings.ReplaceAll(condition, "\"", "\\\"")
	p.output = append(p.output, fmt.Sprintf("BRANCH \"%s\" %s", escapedCondition, conStruct.ConLabel))
	
	return nil
}

// processElse handles ELSE macro expansion
// Equivalent to the 'ELSE' case in TWX ScriptCmp.pas RecurseCmd
func (p *Preprocessor) processElse(parts []string, lineNum int) error {
	if len(parts) > 1 {
		return fmt.Errorf("line %d: unnecessary parameters after ELSE macro", lineNum)
	}
	if len(p.ifStack) == 0 {
		return fmt.Errorf("line %d: ELSE without IF", lineNum)
	}

	// Get current condition structure
	conStruct := p.ifStack[len(p.ifStack)-1]
	if conStruct.IsWhile {
		return fmt.Errorf("line %d: cannot use ELSE with WHILE", lineNum)
	}
	if conStruct.AtElse {
		return fmt.Errorf("line %d: IF macro syntax error", lineNum)
	}

	conStruct.AtElse = true
	
	// Generate GOTO to end and place the condition label
	p.output = append(p.output, fmt.Sprintf("GOTO %s", conStruct.EndLabel))
	p.output = append(p.output, conStruct.ConLabel)
	
	return nil
}

// processElseIf handles ELSEIF macro expansion
// Equivalent to the 'ELSEIF' case in TWX ScriptCmp.pas RecurseCmd
func (p *Preprocessor) processElseIf(parts []string, lineNum int) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: no parameters to compare with ELSEIF macro", lineNum)
	}
	if len(p.ifStack) == 0 {
		return fmt.Errorf("line %d: ELSEIF without IF", lineNum)
	}

	// Get current condition structure
	conStruct := p.ifStack[len(p.ifStack)-1]
	if conStruct.IsWhile {
		return fmt.Errorf("line %d: cannot use ELSEIF with WHILE", lineNum)
	}
	if conStruct.AtElse {
		return fmt.Errorf("line %d: IF macro syntax error", lineNum)
	}

	// Generate GOTO to end and place the old condition label
	p.output = append(p.output, fmt.Sprintf("GOTO %s", conStruct.EndLabel))
	p.output = append(p.output, conStruct.ConLabel)
	
	// Generate new condition label for this ELSEIF
	p.ifLabelCount++
	conStruct.ConLabel = "::" + strconv.Itoa(p.ifLabelCount)
	
	// Generate new BRANCH command
	condition := strings.Join(parts[1:], " ")
	// Escape any quotes in the condition string
	escapedCondition := strings.ReplaceAll(condition, "\"", "\\\"")
	p.output = append(p.output, fmt.Sprintf("BRANCH \"%s\" %s", escapedCondition, conStruct.ConLabel))
	
	return nil
}

// processWhile handles WHILE macro expansion
// Equivalent to the 'WHILE' case in TWX ScriptCmp.pas RecurseCmd
func (p *Preprocessor) processWhile(parts []string, lineNum int) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: no parameters to compare with WHILE macro", lineNum)
	}

	// Create new condition structure
	conStruct := &ConditionStruct{
		IsWhile: true,
	}
	
	// Generate unique labels
	p.ifLabelCount++
	conStruct.ConLabel = "::" + strconv.Itoa(p.ifLabelCount)
	p.ifLabelCount++
	conStruct.EndLabel = "::" + strconv.Itoa(p.ifLabelCount)
	
	// Push onto stack
	p.ifStack = append(p.ifStack, conStruct)
	
	// Generate loop start label and BRANCH command
	p.output = append(p.output, conStruct.ConLabel)
	condition := strings.Join(parts[1:], " ")
	// Escape any quotes in the condition string
	escapedCondition := strings.ReplaceAll(condition, "\"", "\\\"")
	p.output = append(p.output, fmt.Sprintf("BRANCH \"%s\" %s", escapedCondition, conStruct.EndLabel))
	
	return nil
}

// processEnd handles END macro expansion
// Equivalent to the 'END' case in TWX ScriptCmp.pas RecurseCmd
func (p *Preprocessor) processEnd(parts []string, lineNum int) error {
	if len(parts) > 1 {
		return fmt.Errorf("line %d: unnecessary parameters after END macro", lineNum)
	}
	if len(p.ifStack) == 0 {
		return fmt.Errorf("line %d: END without IF", lineNum)
	}

	// Pop condition structure from stack
	conStruct := p.ifStack[len(p.ifStack)-1]
	p.ifStack = p.ifStack[:len(p.ifStack)-1]
	
	if conStruct.IsWhile {
		// For WHILE: jump back to loop start, then place end label
		p.output = append(p.output, fmt.Sprintf("GOTO %s", conStruct.ConLabel))
	} else {
		// For IF: place the condition label (in case there was no ELSE)
		p.output = append(p.output, conStruct.ConLabel)
	}
	
	// Place the end label
	p.output = append(p.output, conStruct.EndLabel)
	
	return nil
}