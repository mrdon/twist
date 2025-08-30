package internal_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTUIImportRestrictions ensures TUI only imports allowed packages
func TestTUIImportRestrictions(t *testing.T) {
	allowedPrefixes := []string{
		"twist/internal/api",        // Core API only
		"twist/internal/log",        // Debug package (required in all files per CLAUDE.md)
		"twist/internal/theme",      // UI theming (until shared)
		"twist/internal/ansi",       // ANSI processing (until shared)
		"twist/internal/terminal",   // Terminal utilities (until shared)
		"twist/internal/components", // UI components (until shared)
		"twist/internal/tui",        // TUI can import its own subpackages
		"github.com/",               // Third-party packages
		"golang.org/",               // Standard library extensions
	}

	forbiddenPrefixes := []string{
		"twist/internal/proxy",     // No proxy internals at all
		"twist/internal/database",  // No direct database access (old path)
		"twist/internal/streaming", // No streaming internals (old path)
		"twist/internal/scripting", // No scripting internals (old path)
	}

	checkImports(t, "./tui", allowedPrefixes, forbiddenPrefixes)
}

// TestProxyImportRestrictions ensures Proxy doesn't import TUI internals
func TestProxyImportRestrictions(t *testing.T) {
	forbiddenPrefixes := []string{
		"twist/internal/tui/", // No TUI internals (except via API)
	}

	// Proxy can import anything except TUI internals
	checkImports(t, "./proxy", nil, forbiddenPrefixes)
}

func checkImports(t *testing.T, packageDir string, allowedPrefixes, forbiddenPrefixes []string) {
	err := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", path, err)
			return nil
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// Skip standard library and relative imports
			if !strings.Contains(importPath, "twist/internal") {
				continue
			}

			// Check forbidden imports
			for _, forbidden := range forbiddenPrefixes {
				if strings.HasPrefix(importPath, forbidden) {
					t.Errorf("FORBIDDEN import in %s: %s", path, importPath)
				}
			}

			// Check allowed imports (if specified)
			if len(allowedPrefixes) > 0 {
				allowed := false
				for _, prefix := range allowedPrefixes {
					if strings.HasPrefix(importPath, prefix) {
						allowed = true
						break
					}
				}
				if !allowed {
					t.Errorf("DISALLOWED import in %s: %s (not in allowed list)", path, importPath)
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Errorf("Failed to walk directory %s: %v", packageDir, err)
	}
}
