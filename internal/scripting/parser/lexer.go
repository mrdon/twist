package parser

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// TokenType represents the type of a token
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenNewline
	TokenComment
	
	// Literals
	TokenString
	TokenNumber
	TokenVariable
	TokenLabel
	
	// Keywords
	TokenIf
	TokenElseif
	TokenElse
	TokenEnd
	TokenWhile
	TokenFor
	TokenGoto
	TokenGosub
	TokenReturn
	TokenInclude
	
	// Operators
	TokenAssign    // :=
	TokenEqual     // =
	TokenNotEqual  // <>
	TokenLess      // <
	TokenLessEq    // <=
	TokenGreater   // >
	TokenGreaterEq // >=
	TokenAnd       // AND
	TokenOr        // OR
	TokenNot       // NOT
	TokenPlus      // +
	TokenMinus     // -
	TokenMultiply  // *
	TokenDivide    // /
	TokenModulus   // MOD
	TokenConcat    // &
	
	// Assignment operators
	TokenPlusAssign     // +=
	TokenMinusAssign    // -=
	TokenMultiplyAssign // *=
	TokenDivideAssign   // /=
	TokenConcatAssign   // &=
	TokenIncrement      // ++
	TokenDecrement      // --
	
	// Delimiters
	TokenLeftParen  // (
	TokenRightParen // )
	TokenLeftBracket // [
	TokenRightBracket // ]
	TokenComma      // ,
	TokenQuote      // "
	
	// Commands
	TokenCommand
)

// Token represents a lexical token
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// Lexer tokenizes script source code
type Lexer struct {
	reader *bufio.Reader
	line   int
	column int
	ch     rune
	eof    bool
}

// NewLexer creates a new lexer
func NewLexer(reader io.Reader) *Lexer {
	l := &Lexer{
		reader: bufio.NewReader(reader),
		line:   1,
		column: 0,
	}
	l.nextChar()
	return l
}

// nextChar reads the next character
func (l *Lexer) nextChar() {
	if l.eof {
		return
	}
	
	ch, _, err := l.reader.ReadRune()
	if err != nil {
		l.eof = true
		l.ch = 0
		return
	}
	
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
	
	l.ch = ch
}

// peekChar looks at the next character without consuming it
func (l *Lexer) peekChar() rune {
	if l.eof {
		return 0
	}
	
	ch, _, err := l.reader.ReadRune()
	if err != nil {
		return 0
	}
	
	// Put the character back
	l.reader.UnreadRune()
	return ch
}

// skipWhitespace skips whitespace characters except newlines
func (l *Lexer) skipWhitespace() {
	for !l.eof && unicode.IsSpace(l.ch) && l.ch != '\n' {
		l.nextChar()
	}
}

// readString reads a quoted string
func (l *Lexer) readString() string {
	var result strings.Builder
	l.nextChar() // skip opening quote
	
	for !l.eof && l.ch != '"' {
		if l.ch == '\\' {
			l.nextChar()
			if !l.eof {
				switch l.ch {
				case 'n':
					result.WriteRune('\n')
				case 't':
					result.WriteRune('\t')
				case 'r':
					result.WriteRune('\r')
				case '\\':
					result.WriteRune('\\')
				case '"':
					result.WriteRune('"')
				default:
					result.WriteRune(l.ch)
				}
			}
		} else {
			result.WriteRune(l.ch)
		}
		l.nextChar()
	}
	
	if l.ch == '"' {
		l.nextChar() // skip closing quote
	}
	
	return result.String()
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() string {
	var result strings.Builder
	
	for !l.eof && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_') {
		result.WriteRune(l.ch)
		l.nextChar()
	}
	
	return result.String()
}

// readNumber reads a numeric literal
func (l *Lexer) readNumber() string {
	var result strings.Builder
	
	for !l.eof && (unicode.IsDigit(l.ch) || l.ch == '.') {
		result.WriteRune(l.ch)
		l.nextChar()
	}
	
	return result.String()
}

// readVariable reads a variable name (starting with $)
func (l *Lexer) readVariable() string {
	var result strings.Builder
	result.WriteRune(l.ch) // include the $
	l.nextChar()
	
	for !l.eof && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_') {
		result.WriteRune(l.ch)
		l.nextChar()
	}
	
	return result.String()
}

// readLabel reads a label (starting with :)
func (l *Lexer) readLabel() string {
	var result strings.Builder
	result.WriteRune(l.ch) // include the :
	l.nextChar()
	
	for !l.eof && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_') {
		result.WriteRune(l.ch)
		l.nextChar()
	}
	
	return result.String()
}

// readComment reads a comment line
func (l *Lexer) readComment() string {
	var result strings.Builder
	
	for !l.eof && l.ch != '\n' {
		result.WriteRune(l.ch)
		l.nextChar()
	}
	
	return result.String()
}

// getKeywordType returns the token type for a keyword
func getKeywordType(word string) TokenType {
	switch strings.ToUpper(word) {
	case "IF":
		return TokenIf
	case "ELSEIF":
		return TokenElseif
	case "ELSE":
		return TokenElse
	case "END", "ENDIF", "ENDWHILE":
		return TokenEnd
	case "WHILE":
		return TokenWhile
	case "FOR":
		return TokenFor
	case "GOTO":
		return TokenGoto
	case "GOSUB":
		return TokenGosub
	case "RETURN":
		return TokenReturn
	case "INCLUDE":
		return TokenInclude
	case "AND":
		return TokenAnd
	case "OR":
		return TokenOr
	case "NOT":
		return TokenNot
	case "MOD":
		return TokenModulus
	default:
		return TokenCommand
	}
}

// NextToken returns the next token
func (l *Lexer) NextToken() (*Token, error) {
	l.skipWhitespace()
	
	if l.eof {
		return &Token{Type: TokenEOF, Line: l.line, Column: l.column}, nil
	}
	
	token := &Token{Line: l.line, Column: l.column}
	
	switch l.ch {
	case '\n':
		token.Type = TokenNewline
		token.Value = "\n"
		l.nextChar()
		
	case '#':
		token.Type = TokenComment
		token.Value = l.readComment()
		
	case '"':
		token.Type = TokenString
		token.Value = l.readString()
		
	case '$':
		token.Type = TokenVariable
		token.Value = l.readVariable()
		
	case ':':
		// Check if it's an assignment or label
		if l.peekChar() == '=' {
			token.Type = TokenAssign
			token.Value = ":="
			l.nextChar() // skip :
			l.nextChar() // skip =
		} else {
			token.Type = TokenLabel
			token.Value = l.readLabel()
		}
		
	case '=':
		token.Type = TokenEqual
		token.Value = "="
		l.nextChar()
		
	case '<':
		if l.peekChar() == '=' {
			token.Type = TokenLessEq
			token.Value = "<="
			l.nextChar()
			l.nextChar()
		} else if l.peekChar() == '>' {
			token.Type = TokenNotEqual
			token.Value = "<>"
			l.nextChar()
			l.nextChar()
		} else {
			token.Type = TokenLess
			token.Value = "<"
			l.nextChar()
		}
		
	case '>':
		if l.peekChar() == '=' {
			token.Type = TokenGreaterEq
			token.Value = ">="
			l.nextChar()
			l.nextChar()
		} else {
			token.Type = TokenGreater
			token.Value = ">"
			l.nextChar()
		}
		
	case '+':
		if l.peekChar() == '=' {
			token.Type = TokenPlusAssign
			token.Value = "+="
			l.nextChar()
			l.nextChar()
		} else if l.peekChar() == '+' {
			token.Type = TokenIncrement
			token.Value = "++"
			l.nextChar()
			l.nextChar()
		} else {
			token.Type = TokenPlus
			token.Value = "+"
			l.nextChar()
		}
		
	case '-':
		if l.peekChar() == '=' {
			token.Type = TokenMinusAssign
			token.Value = "-="
			l.nextChar()
			l.nextChar()
		} else if l.peekChar() == '-' {
			token.Type = TokenDecrement
			token.Value = "--"
			l.nextChar()
			l.nextChar()
		} else {
			token.Type = TokenMinus
			token.Value = "-"
			l.nextChar()
		}
		
	case '*':
		if l.peekChar() == '=' {
			token.Type = TokenMultiplyAssign
			token.Value = "*="
			l.nextChar()
			l.nextChar()
		} else {
			token.Type = TokenMultiply
			token.Value = "*"
			l.nextChar()
		}
		
	case '/':
		if l.peekChar() == '=' {
			token.Type = TokenDivideAssign
			token.Value = "/="
			l.nextChar()
			l.nextChar()
		} else {
			token.Type = TokenDivide
			token.Value = "/"
			l.nextChar()
		}
		
	case '(':
		token.Type = TokenLeftParen
		token.Value = "("
		l.nextChar()
		
	case ')':
		token.Type = TokenRightParen
		token.Value = ")"
		l.nextChar()
		
	case '[':
		token.Type = TokenLeftBracket
		token.Value = "["
		l.nextChar()
		
	case ']':
		token.Type = TokenRightBracket
		token.Value = "]"
		l.nextChar()
		
	case ',':
		token.Type = TokenComma
		token.Value = ","
		l.nextChar()
		
	case '&':
		if l.peekChar() == '=' {
			token.Type = TokenConcatAssign
			token.Value = "&="
			l.nextChar()
			l.nextChar()
		} else {
			token.Type = TokenConcat
			token.Value = "&"
			l.nextChar()
		}
		
	default:
		if unicode.IsDigit(l.ch) {
			token.Type = TokenNumber
			token.Value = l.readNumber()
		} else if unicode.IsLetter(l.ch) {
			word := l.readIdentifier()
			token.Type = getKeywordType(word)
			token.Value = word
		} else {
			return nil, fmt.Errorf("unexpected character '%c' at line %d, column %d", l.ch, l.line, l.column)
		}
	}
	
	return token, nil
}

// TokenizeAll returns all tokens from the input
func (l *Lexer) TokenizeAll() ([]*Token, error) {
	var tokens []*Token
	
	for {
		token, err := l.NextToken()
		if err != nil {
			return nil, err
		}
		
		tokens = append(tokens, token)
		
		if token.Type == TokenEOF {
			break
		}
	}
	
	return tokens, nil
}