package streaming

import (
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// PARSER UTILITY FUNCTIONS (Mirror TWX Pascal utility functions)
// ============================================================================

// getParameter extracts parameter by index from a delimited string (mirrors Pascal GetParameter)
func (p *TWXParser) getParameter(line string, index int) string {
	if index < 1 {
		return ""
	}

	parts := strings.Fields(line)
	if index > len(parts) {
		return ""
	}

	return parts[index-1] // Pascal uses 1-based indexing
}

// getParameterPos gets the position of a parameter in the string (mirrors Pascal GetParameterPos)
func (p *TWXParser) getParameterPos(line string, index int) int {
	if index < 1 {
		return 0
	}

	parts := strings.Fields(line)
	if index > len(parts) {
		return 0
	}

	// Find position of the parameter in original string
	param := parts[index-1]
	return strings.Index(line, param)
}

// stripChar removes all instances of a character from string (mirrors Pascal StripChar)
func (p *TWXParser) stripChar(s *string, char rune) {
	*s = strings.ReplaceAll(*s, string(char), "")
}

// stripChars removes all instances of multiple characters (mirrors Pascal StripChars)
func (p *TWXParser) stripChars(s *string, chars string) {
	for _, char := range chars {
		p.stripChar(s, char)
	}
}

// split divides a string by delimiter (mirrors Pascal Split function)
func (p *TWXParser) split(line string, delimiter string) []string {
	if delimiter == " " {
		return strings.Fields(line) // More robust than strings.Split for spaces
	}
	return strings.Split(line, delimiter)
}

// parseBoolFromString converts various boolean representations (mirrors Pascal StrToBool)
func (p *TWXParser) parseBoolFromString(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "yes", "true", "1", "on", "y":
		return true
	case "no", "false", "0", "off", "n", "":
		return false
	default:
		return false
	}
}

// extractSectorFromParens extracts sector number from parentheses format like "(1234)"
func (p *TWXParser) extractSectorFromParens(line string) int {
	openParen := strings.Index(line, "(")
	closeParen := strings.Index(line, ")")
	if openParen > 0 && closeParen > openParen {
		sectorStr := strings.TrimSpace(line[openParen+1 : closeParen])
		return p.parseIntSafe(sectorStr)
	}
	return 0
}

// extractNumberFromString finds first number in a string
func (p *TWXParser) extractNumberFromString(s string) int {
	var numStr strings.Builder
	for _, char := range s {
		if char >= '0' && char <= '9' {
			numStr.WriteRune(char)
		} else if numStr.Len() > 0 {
			break // Stop at first non-digit after finding digits
		}
	}
	if numStr.Len() > 0 {
		return p.parseIntSafe(numStr.String())
	}
	return 0
}

// parsePortClass determines port class from description (mirrors Pascal logic)
func (p *TWXParser) parsePortClass(portDesc string) int {
	portDesc = strings.ToUpper(portDesc)

	// Extract class pattern like "Class 9 Port" or trade pattern like "(SSSx3)"
	if strings.Contains(portDesc, "CLASS") {
		parts := strings.Fields(portDesc)
		for i, part := range parts {
			if part == "CLASS" && i+1 < len(parts) {
				return p.parseIntSafe(parts[i+1])
			}
		}
	}

	// Determine class from trade pattern
	if strings.Contains(portDesc, "(") && strings.Contains(portDesc, ")") {
		start := strings.Index(portDesc, "(")
		end := strings.Index(portDesc, ")")
		if end > start {
			pattern := portDesc[start+1 : end]
			return p.classFromTradePattern(pattern)
		}
	}

	return 0 // Unknown class
}

// classFromTradePattern determines port class from trade pattern (mirrors Pascal logic)
func (p *TWXParser) classFromTradePattern(pattern string) int {
	// Map trade patterns to port classes (BWB = 2, SBB = 3, etc.)
	pattern = strings.ToUpper(pattern)

	// Remove multipliers like "x3", "x2"
	if idx := strings.Index(pattern, "X"); idx > 0 {
		pattern = pattern[:idx]
	}

	switch pattern {
	case "BBS":
		return 1
	case "BSB":
		return 2
	case "SBB":
		return 3
	case "SSB":
		return 4
	case "SBS":
		return 5
	case "BSS":
		return 6
	case "SSS":
		return 7
	case "BBB":
		return 8
	case "???":
		return 9
	default:
		return 0
	}
}

// parseIntSafe safely parses an integer, returning 0 on error
func (p *TWXParser) parseIntSafe(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Remove commas
	s = strings.ReplaceAll(s, ",", "")

	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// parseIntSafeWithCommas parses integers that may contain commas
func (p *TWXParser) parseIntSafeWithCommas(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	return p.parseIntSafe(s)
}

// stringContainsWord checks if a string contains a whole word (not partial)
func (p *TWXParser) stringContainsWord(s, word string) bool {
	s = strings.ToLower(s)
	word = strings.ToLower(word)

	index := strings.Index(s, word)
	if index == -1 {
		return false
	}

	// Check if it's a whole word (not part of another word)
	if index > 0 && isAlphaNumeric(rune(s[index-1])) {
		return false
	}

	endIndex := index + len(word)
	if endIndex < len(s) && isAlphaNumeric(rune(s[endIndex])) {
		return false
	}

	return true
}

// isAlphaNumeric checks if a character is alphanumeric
func isAlphaNumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// extractQuotedString extracts text between quotes
func (p *TWXParser) extractQuotedString(s string) string {
	first := strings.Index(s, "\"")
	if first == -1 {
		return ""
	}

	second := strings.Index(s[first+1:], "\"")
	if second == -1 {
		return ""
	}

	return s[first+1 : first+1+second]
}

// splitOnCommaOutsideParens splits a string on commas, but ignores commas inside parentheses
func (p *TWXParser) splitOnCommaOutsideParens(s string) []string {
	var result []string
	var current strings.Builder
	parenLevel := 0

	for _, char := range s {
		switch char {
		case '(':
			parenLevel++
			current.WriteRune(char)
		case ')':
			parenLevel--
			current.WriteRune(char)
		case ',':
			if parenLevel == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

// normalizeSpaces normalizes multiple spaces to single spaces
func (p *TWXParser) normalizeSpaces(s string) string {
	// Replace multiple spaces with single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// timeToString formats time like Pascal TimeToStr (for message history)
func (p *TWXParser) timeToString(t time.Time) string {
	return t.Format("15:04:05")
}

// padLeft pads a string to the left with spaces
func (p *TWXParser) padLeft(s string, length int) string {
	if len(s) >= length {
		return s
	}
	padding := strings.Repeat(" ", length-len(s))
	return padding + s
}

// padRight pads a string to the right with spaces
func (p *TWXParser) padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	padding := strings.Repeat(" ", length-len(s))
	return s + padding
}

// contains performs case-insensitive contains check
func (p *TWXParser) contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// hasPrefix performs case-insensitive prefix check
func (p *TWXParser) hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

// hasSuffix performs case-insensitive suffix check
func (p *TWXParser) hasSuffix(s, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
}
