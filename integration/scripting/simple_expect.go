package scripting

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

// SimpleExpectEngine - minimal expect engine for testing
type SimpleExpectEngine struct {
	t               *testing.T
	outputCapture   []string
	inputSender     func(string)
	timeout         time.Duration
	starReplacement string // What to replace "*" with (e.g., "\r" for client, "\r\n" for server)
}

// SimpleExpectCommand represents a single command
type SimpleExpectCommand struct {
	Type string   // expect, send, assert, timeout, log
	Args []string
	Line int
}

// NewSimpleExpectEngine creates a minimal expect engine
func NewSimpleExpectEngine(t *testing.T, inputSender func(string), starReplacement string) *SimpleExpectEngine {
	return &SimpleExpectEngine{
		t:               t,
		inputSender:     inputSender,
		timeout:         5 * time.Second,
		starReplacement: starReplacement,
		outputCapture:   make([]string, 0),
	}
}

// Run executes a simple expect script (one command per line)
func (e *SimpleExpectEngine) Run(script string) error {
	lines := strings.Split(script, "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse command
		parts := e.parseCommand(line)
		if len(parts) == 0 {
			continue
		}

		cmd := SimpleExpectCommand{
			Type: parts[0],
			Args: parts[1:],
			Line: lineNum + 1,
		}

		e.t.Logf("[%d] %s %q", cmd.Line, cmd.Type, cmd.Args)

		err := e.executeCommand(cmd)
		if err != nil {
			return fmt.Errorf("line %d: %w", cmd.Line, err)
		}
	}

	return nil
}

// parseCommand splits command line respecting quotes
func (e *SimpleExpectEngine) parseCommand(line string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false

	for _, char := range line {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ', '\t':
			if !inQuotes && current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else if inQuotes {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// executeCommand runs a single command
func (e *SimpleExpectEngine) executeCommand(cmd SimpleExpectCommand) error {
	switch cmd.Type {
	case "expect":
		return e.expect(cmd.Args)
	case "send":
		return e.send(cmd.Args)
	case "assert":
		return e.assert(cmd.Args)
	case "timeout":
		return e.setTimeout(cmd.Args)
	case "log":
		return e.log(cmd.Args)
	default:
		return fmt.Errorf("unknown command: %s", cmd.Type)
	}
}

// expect waits for pattern in output
func (e *SimpleExpectEngine) expect(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("expect requires pattern")
	}

	pattern := args[0]
	deadline := time.Now().Add(e.timeout)

	for time.Now().Before(deadline) {
		output := strings.Join(e.outputCapture, "")

		// Try regex first, fall back to literal match
		if matched, _ := regexp.MatchString(pattern, output); matched {
			return nil
		}
		if strings.Contains(output, pattern) {
			return nil
		}

		time.Sleep(10 * time.Millisecond)
	}

	output := strings.Join(e.outputCapture, "")
	return fmt.Errorf("timeout waiting for pattern %q - current output: %q", pattern, output)
}

// send inputs data
func (e *SimpleExpectEngine) send(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("send requires input")
	}

	input := args[0]

	// Process "*" using configured replacement
	input = strings.ReplaceAll(input, "*", e.starReplacement)

	if e.inputSender != nil {
		e.inputSender(input)
	}
	return nil
}

// assert checks that pattern exists in output
func (e *SimpleExpectEngine) assert(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("assert requires pattern")
	}

	pattern := args[0]
	output := strings.Join(e.outputCapture, "")

	// Try regex first, fall back to literal match
	if matched, _ := regexp.MatchString(pattern, output); matched {
		return nil
	}
	if strings.Contains(output, pattern) {
		return nil
	}

	return fmt.Errorf("assertion failed: %q not found in output: %q", pattern, output)
}

// setTimeout changes default timeout
func (e *SimpleExpectEngine) setTimeout(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("timeout requires duration")
	}

	timeout, err := time.ParseDuration(args[0])
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	e.timeout = timeout
	return nil
}

// log outputs a message
func (e *SimpleExpectEngine) log(args []string) error {
	message := ""
	if len(args) > 0 {
		message = args[0]
	}
	e.t.Logf("EXPECT: %s", message)
	return nil
}

// AddOutput feeds output to the engine
func (e *SimpleExpectEngine) AddOutput(output string) {
	e.outputCapture = append(e.outputCapture, output)
}

// GetAllOutput returns all captured output as a single string
func (e *SimpleExpectEngine) GetAllOutput() string {
	return strings.Join(e.outputCapture, "")
}

// ClearOutput clears the output capture buffer
func (e *SimpleExpectEngine) ClearOutput() {
	e.outputCapture = e.outputCapture[:0]
}