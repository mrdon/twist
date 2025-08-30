# API Phase 5: Module Cleanup and Separation

## Goal

Complete the architectural separation by moving proxy-internal modules to proper locations and establishing clean module boundaries.

## Overview

Phase 5 completes the proxy-TUI API separation with **simple module reorganization**:

1. **Move proxy-internal modules** to `internal/proxy/` package structure
2. **Update import paths** throughout the codebase
3. **Verify clean build** and functionality

**Scope**: This is a **simple refactoring** - just moving files and updating imports. No complex migration or backwards compatibility needed.

## Current State Analysis (Validated)

### ✅ Architecture is Already Clean:

**TUI Module is Fully Separated**:
- ✅ **No forbidden imports**: TUI has zero imports of `internal/database`, `internal/streaming`, or `internal/scripting`
- ✅ **API-only access**: All TUI components use `internal/api` exclusively
- ✅ **Clean build**: Current codebase builds without errors

**API Separation is Complete**:
- ✅ **Connection management**: Working via `api.Connect()` and callbacks
- ✅ **Script management**: Working via `LoadScript()`, `GetScriptStatus()` etc.
- ✅ **Game state access**: Working via `GetCurrentSector()`, `GetSectorInfo()` etc.

### 📊 Simple Cleanup Needed:

**Module Organization**: Core modules scattered in `internal/` instead of organized under `internal/proxy/`
- `internal/streaming/` → `internal/proxy/streaming/`
- `internal/scripting/` → `internal/proxy/scripting/`  
- `internal/database/` → `internal/proxy/database/`

**Import Updates Required**: 38 files with 62 import statements need updating:
- 14 database import statements
- 2 streaming import statements
- 46 scripting import statements

## Scope - Module Restructuring Design

### Target Module Structure (After Phase 5):

```
internal/
├── api/                     # Core API interfaces (unchanged)
│   └── api.go              # ProxyAPI, TuiAPI, data types
├── proxy/                  # Complete proxy package
│   ├── proxy.go            # Core proxy logic
│   ├── proxy_api_impl.go   # ProxyAPI implementation
│   ├── game_state_converters.go  # API data converters
│   ├── database/           # Database management (moved)
│   │   ├── database.go
│   │   ├── migrations.go
│   │   ├── schema.go
│   │   └── structs.go
│   ├── streaming/          # Data streaming (moved)
│   │   ├── pipeline.go
│   │   └── parser/
│   │       ├── parser.go
│   │       ├── port.go
│   │       ├── sector.go
│   │       ├── types.go
│   │       └── utils.go
│   └── scripting/          # Script management (moved)
│       ├── engine.go
│       ├── integration.go
│       ├── manager/
│       ├── parser/
│       ├── triggers/
│       ├── types/
│       └── vm/
├── tui/                    # TUI package (unchanged)
│   ├── api/                # TUI API integration
│   │   ├── proxy_client.go
│   │   └── tui_api_impl.go
│   ├── app.go              # Main TUI app
│   ├── components/         # UI components
│   └── handlers/           # Input handlers
├── ansi/                   # ANSI processing (keep here until shared)
├── telnet/                 # Telnet protocol (keep here until shared)
├── terminal/               # Terminal utilities (keep here until shared)
├── theme/                  # UI theming (keep here until shared)
├── debug/                  # Debug utilities (keep here until shared)
└── components/             # UI components (keep here until shared)
```

### Import Restrictions (Enforced):

**TUI Module** - Can ONLY import:
```go
// ALLOWED imports for TUI
import (
    "twist/internal/api"           // ✅ Core API only
    "twist/internal/theme"         // ✅ UI theming (until shared)
    "twist/internal/ansi"          // ✅ ANSI processing (until shared)
    "twist/internal/terminal"      // ✅ Terminal utilities (until shared)
    "twist/internal/components"    // ✅ UI components (until shared)
    // Standard library packages     // ✅ Always allowed
    // Third-party UI packages       // ✅ tview, etc.
)

// FORBIDDEN imports for TUI (will cause build failure)
import (
    "twist/internal/proxy"           // ❌ NO proxy internals
    "twist/internal/proxy/database"  // ❌ NO database access
    "twist/internal/proxy/streaming" // ❌ NO streaming internals  
    "twist/internal/proxy/scripting" // ❌ NO scripting internals
)
```

**Proxy Module** - Can import its own internals:
```go
// ALLOWED imports for Proxy
import (
    "twist/internal/api"              // ✅ Core API
    "twist/internal/proxy/database"   // ✅ Internal database
    "twist/internal/proxy/streaming"  // ✅ Internal streaming
    "twist/internal/proxy/scripting"  // ✅ Internal scripting
    "twist/internal/theme"            // ✅ Utilities (until shared)
    // Standard library packages      // ✅ Always allowed
)

// FORBIDDEN imports for Proxy
import (
    "twist/internal/tui"             // ❌ NO TUI internals (except via API)
)
```

## Implementation Steps (Simple Approach)

### Step 1: Move All Modules at Once

**Move database module:**
```bash
mkdir -p internal/proxy/database
mv internal/database/* internal/proxy/database/
rmdir internal/database
```

**Move streaming module:**
```bash
mkdir -p internal/proxy/streaming/parser
mv internal/streaming/* internal/proxy/streaming/
rmdir internal/streaming
```

**Move scripting module:**
```bash
mkdir -p internal/proxy/scripting
mv internal/scripting/* internal/proxy/scripting/
rmdir internal/scripting
```

### Step 2: Update All Import Statements

**Update all 62 import statements from:**
- `"twist/internal/database"` → `"twist/internal/proxy/database"`
- `"twist/internal/streaming"` → `"twist/internal/proxy/streaming"`  
- `"twist/internal/scripting"` → `"twist/internal/proxy/scripting"`

**Files that need import updates (38 total):**
- All proxy files  
- All scripting internal files
- All streaming internal files
- Integration test files (refactor to use API instead of direct imports)

### Step 3: Add Import Restriction Tests

Create `internal/architecture_test.go`:
```go
package internal

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
        "twist/internal/api",           // Core API only
        "twist/internal/theme",         // UI theming (until shared)
        "twist/internal/ansi",          // ANSI processing (until shared)
        "twist/internal/terminal",      // Terminal utilities (until shared)
        "twist/internal/components",    // UI components (until shared)
        "twist/internal/log",         // Debug utilities (until shared)
        "github.com/",                  // Third-party packages
        "golang.org/",                  // Standard library extensions
    }
    
    forbiddenPrefixes := []string{
        "twist/internal/proxy/",        // No proxy internals
        "twist/internal/database",      // No direct database access
        "twist/internal/streaming",     // No streaming internals
        "twist/internal/scripting",     // No scripting internals
    }
    
    checkImports(t, "internal/tui", allowedPrefixes, forbiddenPrefixes)
}

// TestProxyImportRestrictions ensures Proxy doesn't import TUI internals
func TestProxyImportRestrictions(t *testing.T) {
    forbiddenPrefixes := []string{
        "twist/internal/tui/",          // No TUI internals (except via API)
    }
    
    // Proxy can import anything except TUI internals
    checkImports(t, "internal/proxy", nil, forbiddenPrefixes)
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
```

### Step 4: Test and Verify

```bash
go build ./...                        # Verify build works
go test ./...                         # Verify tests pass
go test -v ./internal -run Architecture # Run import restriction tests
```

## Success Criteria

✅ **Complete Module Organization**: All proxy internals moved to `internal/proxy/`  
✅ **Import Paths Updated**: All 62 import statements updated correctly  
✅ **Clean Build**: Project builds without errors after reorganization  
✅ **Functionality Intact**: All existing functionality still works  

## Summary

Phase 5 is a **straightforward module reorganization**:

1. **Move 3 modules** to `internal/proxy/` package
2. **Update 62 import statements** across 38 files  
3. **Verify build works** after the changes

**Why this is simple:**
- Architecture is already clean (no forbidden imports in TUI)
- Just organizing files into better locations
- No complex migration or backwards compatibility needed
- Standard Go refactoring - move files, update imports, test build

**Result:** Clean module structure that matches the architectural vision from docs/api.md