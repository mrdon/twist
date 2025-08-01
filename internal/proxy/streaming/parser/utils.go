package parser

import (
	"strconv"
	"strings"
)

// ParseUtils provides common parsing utilities
type ParseUtils struct {
	ctx *ParserContext
}

// NewParseUtils creates a new ParseUtils instance
func NewParseUtils(ctx *ParserContext) *ParseUtils {
	return &ParseUtils{ctx: ctx}
}

// StripANSI removes ANSI escape codes from text
func (pu *ParseUtils) StripANSI(text string) string {
	return pu.ctx.AnsiPattern.ReplaceAllString(text, "")
}

// StrToIntSafe converts string to int safely, returning 0 on error
func (pu *ParseUtils) StrToIntSafe(s string) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}

// GetParameter extracts parameter at given position from a line
func (pu *ParseUtils) GetParameter(line string, paramNum int) string {
	params := strings.Fields(line)
	if paramNum >= 0 && paramNum < len(params) {
		return params[paramNum]
	}
	return ""
}

// GetParameterPos finds the position of a parameter in a line
func (pu *ParseUtils) GetParameterPos(line string, paramNum int) int {
	currentParam := 0
	for i, r := range line {
		if r != ' ' && r != '\t' {
			if currentParam == paramNum {
				return i
			}
			// Skip to end of this parameter
			for i < len(line) && line[i] != ' ' && line[i] != '\t' {
				i++
			}
			currentParam++
		}
	}
	return -1
}

// StripChar removes a specific character from the beginning of a string
func (pu *ParseUtils) StripChar(line *string, char rune) {
	if len(*line) > 0 && rune((*line)[0]) == char {
		*line = (*line)[1:]
	}
}