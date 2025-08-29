package include

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"twist/internal/proxy/scripting/parser"
)

// IncludeProcessor handles script file inclusion
type IncludeProcessor struct {
	includedFiles map[string]bool
	basePath      string
}

// NewIncludeProcessor creates a new include processor
func NewIncludeProcessor(basePath string) *IncludeProcessor {
	return &IncludeProcessor{
		includedFiles: make(map[string]bool),
		basePath:      basePath,
	}
}

// ProcessIncludes processes all INCLUDE statements in an AST and returns a new AST with included content
func (ip *IncludeProcessor) ProcessIncludes(ast *parser.ASTNode) (*parser.ASTNode, error) {
	if ast == nil {
		return nil, nil
	}

	// Create a new AST for the result
	result := &parser.ASTNode{
		Type:     ast.Type,
		Value:    ast.Value,
		Line:     ast.Line,
		Column:   ast.Column,
		Children: make([]*parser.ASTNode, 0),
	}

	// Process each child node
	for _, child := range ast.Children {
		if child.Type == parser.NodeInclude {
			// Process the include statement
			includedNodes, err := ip.processIncludeNode(child)
			if err != nil {
				return nil, err
			}
			result.Children = append(result.Children, includedNodes...)
		} else {
			// Recursively process other nodes
			processedChild, err := ip.ProcessIncludes(child)
			if err != nil {
				return nil, err
			}
			result.Children = append(result.Children, processedChild)
		}
	}

	return result, nil
}

// processIncludeNode processes a single include node
func (ip *IncludeProcessor) processIncludeNode(node *parser.ASTNode) ([]*parser.ASTNode, error) {
	if len(node.Children) == 0 {
		return nil, fmt.Errorf("INCLUDE statement missing filename at line %d", node.Line)
	}

	filename := node.Children[0].Value

	// Remove quotes if present
	if len(filename) >= 2 && filename[0] == '"' && filename[len(filename)-1] == '"' {
		filename = filename[1 : len(filename)-1]
	}

	// Normalize filename (uppercase like Pascal version)
	normalizedName := strings.ToUpper(filename)

	// Check if already included
	if ip.includedFiles[normalizedName] {
		// Already included, return empty slice
		return []*parser.ASTNode{}, nil
	}

	// Mark as included
	ip.includedFiles[normalizedName] = true

	// Resolve file path
	fullPath := ip.resolveFilePath(filename)

	// Load and parse the included file
	includedAST, err := ip.loadAndParseFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("unable to process include file '%s': %v", filename, err)
	}

	// Recursively process includes in the included file
	processedAST, err := ip.ProcessIncludes(includedAST)
	if err != nil {
		return nil, err
	}

	// Return the children of the processed AST (we don't want the program node wrapper)
	return processedAST.Children, nil
}

// resolveFilePath resolves the include file path
func (ip *IncludeProcessor) resolveFilePath(filename string) string {
	// If filename is absolute, use it as-is
	if filepath.IsAbs(filename) {
		return filename
	}

	// Otherwise, resolve relative to base path
	return filepath.Join(ip.basePath, filename)
}

// loadAndParseFile loads and parses a script file
func (ip *IncludeProcessor) loadAndParseFile(filepath string) (*parser.ASTNode, error) {
	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filepath)
	}

	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %v", filepath, err)
	}
	defer file.Close()

	// Create lexer and parse (no line mappings for included files)
	lexer := parser.NewLexer(file, nil)
	tokens, err := lexer.TokenizeAll()
	if err != nil {
		return nil, fmt.Errorf("tokenization error in %s: %v", filepath, err)
	}

	p := parser.NewParser(tokens)
	ast, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing error in %s: %v", filepath, err)
	}

	return ast, nil
}

// Reset clears the included files cache
func (ip *IncludeProcessor) Reset() {
	ip.includedFiles = make(map[string]bool)
}

// GetIncludedFiles returns a list of included file names
func (ip *IncludeProcessor) GetIncludedFiles() []string {
	files := make([]string, 0, len(ip.includedFiles))
	for file := range ip.includedFiles {
		files = append(files, file)
	}
	return files
}
