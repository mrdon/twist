package parser

import (
	"fmt"
	"strings"
)

// NodeType represents the type of an AST node
type NodeType int

const (
	NodeProgram NodeType = iota
	NodeCommand
	NodeLabel
	NodeVariable
	NodeLiteral
	NodeExpression
	NodeIf
	NodeWhile
	NodeAssignment
	NodeArrayAccess
	NodeCompoundAssignment
	NodeIncrementDecrement
	NodeInclude
)

// ASTNode represents a node in the Abstract Syntax Tree
type ASTNode struct {
	Type     NodeType
	Value    string
	Line     int
	Column   int
	Children []*ASTNode
}

// Parser parses tokens into an AST
type Parser struct {
	tokens   []*Token
	position int
	current  *Token
}

// NewParser creates a new parser
func NewParser(tokens []*Token) *Parser {
	p := &Parser{
		tokens:   tokens,
		position: 0,
	}
	if len(tokens) > 0 {
		p.current = tokens[0]
	}
	return p
}

// advance moves to the next token
func (p *Parser) advance() {
	if p.position < len(p.tokens)-1 {
		p.position++
		p.current = p.tokens[p.position]
	}
}

// peek returns the next token without advancing
func (p *Parser) peek() *Token {
	if p.position < len(p.tokens)-1 {
		return p.tokens[p.position+1]
	}
	return &Token{Type: TokenEOF}
}

// expect checks if the current token matches the expected type and advances
func (p *Parser) expect(tokenType TokenType) error {
	if p.current.Type != tokenType {
		return fmt.Errorf("expected %v, got %v at line %d", tokenType, p.current.Type, p.current.Line)
	}
	p.advance()
	return nil
}

// skipNewlines skips newline tokens and comments
func (p *Parser) skipNewlines() {
	for p.current.Type == TokenNewline || p.current.Type == TokenComment {
		p.advance()
	}
}

// Parse parses the tokens into an AST
func (p *Parser) Parse() (*ASTNode, error) {
	program := &ASTNode{
		Type:     NodeProgram,
		Children: make([]*ASTNode, 0),
	}
	
	for p.current.Type != TokenEOF {
		// Skip comments and newlines at the top level
		if p.current.Type == TokenComment || p.current.Type == TokenNewline {
			p.advance()
			continue
		}
		
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		
		if stmt != nil {
			program.Children = append(program.Children, stmt)
		}
		
		p.skipNewlines()
	}
	
	return program, nil
}

// parseStatement parses a single statement
func (p *Parser) parseStatement() (*ASTNode, error) {
	switch p.current.Type {
	case TokenLabel:
		return p.parseLabel()
	case TokenIf:
		return p.parseIf()
	case TokenWhile:
		return p.parseWhile()
	case TokenVariable:
		return p.parseAssignmentOrCommand()
	case TokenCommand:
		return p.parseCommand()
	case TokenGoto, TokenGosub, TokenReturn:
		return p.parseControlFlow()
	case TokenInclude:
		return p.parseInclude()
	default:
		// Try to parse as a command
		return p.parseCommand()
	}
}

// parseLabel parses a label definition
func (p *Parser) parseLabel() (*ASTNode, error) {
	node := &ASTNode{
		Type:   NodeLabel,
		Value:  p.current.Value,
		Line:   p.current.Line,
		Column: p.current.Column,
	}
	p.advance()
	return node, nil
}

// parseIf parses an if statement
func (p *Parser) parseIf() (*ASTNode, error) {
	node := &ASTNode{
		Type:     NodeIf,
		Line:     p.current.Line,
		Column:   p.current.Column,
		Children: make([]*ASTNode, 0),
	}
	
	p.advance() // skip 'if'
	
	// Parse condition
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	node.Children = append(node.Children, condition)
	
	p.skipNewlines()
	
	// Parse statements until 'elseif', 'else', or 'end'
	for p.current.Type != TokenElseif && p.current.Type != TokenElse && p.current.Type != TokenEnd && p.current.Type != TokenEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			node.Children = append(node.Children, stmt)
		}
		p.skipNewlines()
	}
	
	// Handle elseif/else clauses
	for p.current.Type == TokenElseif {
		p.advance() // skip 'elseif'
		
		elseifCondition, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		
		elseifNode := &ASTNode{
			Type:     NodeIf,
			Value:    "elseif",
			Children: []*ASTNode{elseifCondition},
		}
		
		p.skipNewlines()
		
		for p.current.Type != TokenElseif && p.current.Type != TokenElse && p.current.Type != TokenEnd && p.current.Type != TokenEOF {
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				elseifNode.Children = append(elseifNode.Children, stmt)
			}
			p.skipNewlines()
		}
		
		node.Children = append(node.Children, elseifNode)
	}
	
	// Handle else clause
	if p.current.Type == TokenElse {
		p.advance() // skip 'else'
		p.skipNewlines()
		
		elseNode := &ASTNode{
			Type:     NodeIf,
			Value:    "else",
			Children: make([]*ASTNode, 0),
		}
		
		for p.current.Type != TokenEnd && p.current.Type != TokenEOF {
			stmt, err := p.parseStatement()
			if err != nil {
				return nil, err
			}
			if stmt != nil {
				elseNode.Children = append(elseNode.Children, stmt)
			}
			p.skipNewlines()
		}
		
		node.Children = append(node.Children, elseNode)
	}
	
	// Expect 'end'
	if err := p.expect(TokenEnd); err != nil {
		return nil, err
	}
	
	return node, nil
}

// parseWhile parses a while loop
func (p *Parser) parseWhile() (*ASTNode, error) {
	node := &ASTNode{
		Type:     NodeWhile,
		Line:     p.current.Line,
		Column:   p.current.Column,
		Children: make([]*ASTNode, 0),
	}
	
	p.advance() // skip 'while'
	
	// Parse condition
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	node.Children = append(node.Children, condition)
	
	p.skipNewlines()
	
	// Parse statements until 'end'
	for p.current.Type != TokenEnd && p.current.Type != TokenEOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			node.Children = append(node.Children, stmt)
		}
		p.skipNewlines()
	}
	
	// Expect 'end'
	if err := p.expect(TokenEnd); err != nil {
		return nil, err
	}
	
	return node, nil
}

// parseAssignmentOrCommand parses either a variable assignment or a command starting with a variable
func (p *Parser) parseAssignmentOrCommand() (*ASTNode, error) {
	// Look ahead to see if this is an assignment
	nextToken := p.peek()
	if nextToken.Type == TokenLeftBracket ||
		nextToken.Type == TokenPlusAssign || nextToken.Type == TokenMinusAssign ||
		nextToken.Type == TokenMultiplyAssign || nextToken.Type == TokenDivideAssign ||
		nextToken.Type == TokenConcatAssign {
		return p.parseAssignment()
	}
	
	// Check for Go-style assignment and reject it explicitly
	if nextToken.Type == TokenAssign {
		return nil, fmt.Errorf("invalid syntax: Go-style assignment ':=' not supported in TWX scripts. Use 'setVar %s value' instead at line %d", p.current.Value, nextToken.Line)
	}
	
	// Check for increment/decrement operators
	if nextToken.Type == TokenIncrement || nextToken.Type == TokenDecrement {
		return p.parseIncrementDecrement()
	}
	
	// Otherwise, it's a command
	return p.parseCommand()
}

// parseAssignment parses a variable assignment
func (p *Parser) parseAssignment() (*ASTNode, error) {
	// Parse variable (with possible array access)
	variable, err := p.parseVariableAccess()
	if err != nil {
		return nil, err
	}
	
	// Check what kind of assignment this is
	if p.current.Type == TokenPlusAssign || p.current.Type == TokenMinusAssign ||
		p.current.Type == TokenMultiplyAssign || p.current.Type == TokenDivideAssign ||
		p.current.Type == TokenConcatAssign {
		
		// Compound assignment
		node := &ASTNode{
			Type:     NodeCompoundAssignment,
			Value:    p.current.Value,
			Line:     p.current.Line,
			Column:   p.current.Column,
			Children: []*ASTNode{variable},
		}
		
		p.advance() // skip compound operator
		
		// Parse expression
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, expr)
		
		return node, nil
	}
	
	return nil, fmt.Errorf("expected assignment operator at line %d", p.current.Line)
}

// parseIncrementDecrement parses increment/decrement operations
func (p *Parser) parseIncrementDecrement() (*ASTNode, error) {
	// Parse variable (with possible array access)
	variable, err := p.parseVariableAccess()
	if err != nil {
		return nil, err
	}
	
	// Check for increment/decrement operator
	if p.current.Type != TokenIncrement && p.current.Type != TokenDecrement {
		return nil, fmt.Errorf("expected ++ or -- at line %d", p.current.Line)
	}
	
	node := &ASTNode{
		Type:     NodeIncrementDecrement,
		Value:    p.current.Value,
		Line:     p.current.Line,
		Column:   p.current.Column,
		Children: []*ASTNode{variable},
	}
	
	p.advance() // skip ++ or --
	
	return node, nil
}

// parseInclude parses an include statement
func (p *Parser) parseInclude() (*ASTNode, error) {
	node := &ASTNode{
		Type:     NodeInclude,
		Line:     p.current.Line,
		Column:   p.current.Column,
		Children: make([]*ASTNode, 0),
	}
	
	p.advance() // skip 'include'
	
	// Parse filename parameter
	if p.current.Type != TokenString && p.current.Type != TokenCommand {
		return nil, fmt.Errorf("INCLUDE expects a filename at line %d", p.current.Line)
	}
	
	filename := &ASTNode{
		Type:   NodeLiteral,
		Value:  p.current.Value,
		Line:   p.current.Line,
		Column: p.current.Column,
	}
	
	node.Children = append(node.Children, filename)
	p.advance()
	
	return node, nil
}

// parseVariableAccess parses a variable with optional array access
func (p *Parser) parseVariableAccess() (*ASTNode, error) {
	if p.current.Type != TokenVariable {
		return nil, fmt.Errorf("expected variable at line %d", p.current.Line)
	}
	
	variable := &ASTNode{
		Type:   NodeVariable,
		Value:  p.current.Value,
		Line:   p.current.Line,
		Column: p.current.Column,
	}
	p.advance()
	
	// Check for array access (supports multi-dimensional arrays)
	if p.current.Type == TokenLeftBracket {
		arrayAccess := &ASTNode{
			Type:     NodeArrayAccess,
			Children: []*ASTNode{variable},
		}
		
		// Parse all bracket pairs for multi-dimensional arrays
		for p.current.Type == TokenLeftBracket {
			p.advance() // skip '['
			
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			arrayAccess.Children = append(arrayAccess.Children, index)
			
			if err := p.expect(TokenRightBracket); err != nil {
				return nil, err
			}
		}
		
		return arrayAccess, nil
	}
	
	return variable, nil
}

// parseCommand parses a command statement
func (p *Parser) parseCommand() (*ASTNode, error) {
	if p.current.Type == TokenEOF {
		return nil, nil
	}
	
	node := &ASTNode{
		Type:     NodeCommand,
		Value:    strings.ToUpper(p.current.Value),
		Line:     p.current.Line,
		Column:   p.current.Column,
		Children: make([]*ASTNode, 0),
	}
	
	p.advance()
	
	// Parse parameters
	for p.current.Type != TokenNewline && p.current.Type != TokenEOF && p.current.Type != TokenComment {
		param, err := p.parseCommandParameter()
		if err != nil {
			return nil, fmt.Errorf("error parsing parameter for command %s: %v", node.Value, err)
		}
		node.Children = append(node.Children, param)
		
		// Skip optional comma
		if p.current.Type == TokenComma {
			p.advance()
		}
	}
	
	return node, nil
}

// parseCommandParameter parses a single command parameter
// This is more restrictive than parseExpression to handle space-separated parameters correctly
func (p *Parser) parseCommandParameter() (*ASTNode, error) {
	// Handle special case: negative numbers in command parameters
	if p.current.Type == TokenMinus && p.peek() != nil && p.peek().Type == TokenNumber {
		// Create unary minus expression for negative numbers
		minusToken := p.current
		p.advance() // skip minus
		numberToken := p.current
		p.advance() // skip number
		
		return &ASTNode{
			Type:     NodeExpression,
			Value:    "-",
			Line:     minusToken.Line,
			Column:   minusToken.Column,
			Children: []*ASTNode{
				{
					Type:   NodeLiteral,
					Value:  numberToken.Value,
					Line:   numberToken.Line,
					Column: numberToken.Column,
				},
			},
		}, nil
	}
	
	// Handle variables (including array access)
	if p.current.Type == TokenVariable {
		return p.parseVariableAccess()
	}
	
	// Handle system constants and identifiers that should be treated as variables
	// Only in command parameter context (not in expressions)
	if p.current.Type == TokenCommand && p.isSystemConstant(p.current.Value) {
		// In TWX, system constants like TRUE, FALSE, CURRENTLINE are variable references
		token := p.current
		p.advance()
		return &ASTNode{
			Type:   NodeVariable,
			Value:  token.Value,
			Line:   token.Line,
			Column: token.Column,
		}, nil
	}
	
	// For other cases, parse as primary expression
	return p.parsePrimaryExpression()
}

// isSystemConstant checks if a token value is a known system constant
func (p *Parser) isSystemConstant(value string) bool {
	switch strings.ToUpper(value) {
	case "TRUE", "FALSE", "CURRENTLINE", "CURRENTANSILINE", "VERSION", "GAME", 
		 "CURRENTSECTOR", "CONNECTED", "DATE", "TIME", "GAMENAME", "TWXVERSION":
		return true
	default:
		return false
	}
}

// parseControlFlow parses control flow statements (goto, gosub, return)
func (p *Parser) parseControlFlow() (*ASTNode, error) {
	node := &ASTNode{
		Type:     NodeCommand,
		Value:    strings.ToUpper(p.current.Value),
		Line:     p.current.Line,
		Column:   p.current.Column,
		Children: make([]*ASTNode, 0),
	}
	
	command := p.current.Value
	p.advance()
	
	// Parse parameters for goto/gosub
	if strings.ToUpper(command) == "GOTO" || strings.ToUpper(command) == "GOSUB" {
		if p.current.Type != TokenNewline && p.current.Type != TokenEOF {
			param, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, param)
		}
	}
	
	return node, nil
}

// parseExpression parses an expression
func (p *Parser) parseExpression() (*ASTNode, error) {
	return p.parseOrExpression()
}

// ParseExpression parses a single expression (public method for external use)
func (p *Parser) ParseExpression() (*ASTNode, error) {
	return p.parseExpression()
}

// parseOrExpression parses OR expressions
func (p *Parser) parseOrExpression() (*ASTNode, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenOr {
		op := p.current.Value
		p.advance()
		
		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}
		
		left = &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{left, right},
		}
	}
	
	return left, nil
}

// parseAndExpression parses AND and XOR expressions
func (p *Parser) parseAndExpression() (*ASTNode, error) {
	left, err := p.parseEqualityExpression()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenAnd || p.current.Type == TokenXor {
		op := p.current.Value
		p.advance()
		
		right, err := p.parseEqualityExpression()
		if err != nil {
			return nil, err
		}
		
		left = &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{left, right},
		}
	}
	
	return left, nil
}

// parseEqualityExpression parses equality expressions (=, <>, etc.)
func (p *Parser) parseEqualityExpression() (*ASTNode, error) {
	left, err := p.parseRelationalExpression()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenEqual || p.current.Type == TokenNotEqual {
		op := p.current.Value
		p.advance()
		
		right, err := p.parseRelationalExpression()
		if err != nil {
			return nil, err
		}
		
		left = &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{left, right},
		}
	}
	
	return left, nil
}

// parseRelationalExpression parses relational expressions (<, >, <=, >=)
func (p *Parser) parseRelationalExpression() (*ASTNode, error) {
	left, err := p.parseAdditiveExpression()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenLess || p.current.Type == TokenLessEq || 
		p.current.Type == TokenGreater || p.current.Type == TokenGreaterEq {
		op := p.current.Value
		p.advance()
		
		right, err := p.parseAdditiveExpression()
		if err != nil {
			return nil, err
		}
		
		left = &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{left, right},
		}
	}
	
	return left, nil
}

// parseAdditiveExpression parses addition and subtraction
func (p *Parser) parseAdditiveExpression() (*ASTNode, error) {
	left, err := p.parseConcatenationExpression()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenPlus || p.current.Type == TokenMinus {
		op := p.current.Value
		p.advance()
		
		right, err := p.parseConcatenationExpression()
		if err != nil {
			return nil, err
		}
		
		left = &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{left, right},
		}
	}
	
	return left, nil
}

// parseConcatenationExpression parses string concatenation (&)
func (p *Parser) parseConcatenationExpression() (*ASTNode, error) {
	left, err := p.parseMultiplicativeExpression()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenConcat {
		op := p.current.Value
		p.advance()
		
		right, err := p.parseMultiplicativeExpression()
		if err != nil {
			return nil, err
		}
		
		left = &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{left, right},
		}
	}
	
	return left, nil
}

// parseMultiplicativeExpression parses multiplication, division, and modulus
func (p *Parser) parseMultiplicativeExpression() (*ASTNode, error) {
	left, err := p.parseUnaryExpression()
	if err != nil {
		return nil, err
	}
	
	for p.current.Type == TokenMultiply || p.current.Type == TokenDivide || p.current.Type == TokenModulus {
		op := p.current.Value
		p.advance()
		
		right, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}
		
		left = &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{left, right},
		}
	}
	
	return left, nil
}

// parseUnaryExpression parses unary expressions (NOT, -, +)
func (p *Parser) parseUnaryExpression() (*ASTNode, error) {
	if p.current.Type == TokenNot || p.current.Type == TokenMinus || p.current.Type == TokenPlus {
		op := p.current.Value
		p.advance()
		
		expr, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}
		
		return &ASTNode{
			Type:     NodeExpression,
			Value:    op,
			Children: []*ASTNode{expr},
		}, nil
	}
	
	return p.parsePrimaryExpression()
}

// parsePrimaryExpression parses primary expressions (literals, variables, parentheses)
func (p *Parser) parsePrimaryExpression() (*ASTNode, error) {
	switch p.current.Type {
	case TokenString:
		node := &ASTNode{
			Type:   NodeLiteral,
			Value:  p.current.Value,
			Line:   p.current.Line,
			Column: p.current.Column,
		}
		p.advance()
		return node, nil
		
	case TokenNumber:
		node := &ASTNode{
			Type:   NodeLiteral,
			Value:  p.current.Value,
			Line:   p.current.Line,
			Column: p.current.Column,
		}
		p.advance()
		return node, nil
		
	case TokenVariable:
		return p.parseVariableAccess()
		
	case TokenLabel:
		node := &ASTNode{
			Type:   NodeLiteral,
			Value:  p.current.Value,
			Line:   p.current.Line,
			Column: p.current.Column,
		}
		p.advance()
		return node, nil
		
	case TokenLeftParen:
		p.advance() // skip '('
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if err := p.expect(TokenRightParen); err != nil {
			return nil, err
		}
		return expr, nil
		
	case TokenCommand:
		// Command as expression (for function calls)
		node := &ASTNode{
			Type:     NodeCommand,
			Value:    strings.ToUpper(p.current.Value),
			Line:     p.current.Line,
			Column:   p.current.Column,
			Children: make([]*ASTNode, 0),
		}
		p.advance()
		return node, nil
		
	default:
		// Try to parse as a command
		if p.current.Type != TokenEOF {
			node := &ASTNode{
				Type:   NodeCommand,
				Value:  strings.ToUpper(p.current.Value),
				Line:   p.current.Line,
				Column: p.current.Column,
			}
			p.advance()
			return node, nil
		}
		return nil, fmt.Errorf("unexpected token %v at line %d", p.current.Type, p.current.Line)
	}
}