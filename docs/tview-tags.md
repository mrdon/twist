# TView Color Tag Optimization

## Problem Analysis

The current terminal rendering system has significant inefficiencies in how it handles colors:

### Current Inefficient Flow
1. **ANSI codes** → **Hex colors** (via ANSIConverter)
2. **Hex colors** stored in every Cell struct (80x25 = 2,000 cells)
3. **Hex colors** → **tview color tags** (via hexToTViewColor)
4. Renders color tag for nearly every character

### Issues with Current Approach
- **Memory waste**: Every cell stores full color state even when unchanged
- **CPU overhead**: Per-character color processing during render
- **Redundant conversion**: ANSI → hex → tview tags
- **Poor data locality**: Color info scattered across 2,000+ cell structs

## Proposed Efficient Architecture

### Key Insight
TView color tags are only needed when colors actually change. We should store data the way we consume it - as a stream of characters with sparse color change markers.

### New Data Structure
```go
type Terminal struct {
    runes [][]rune              // Just the characters (2D grid)
    colorChanges []ColorChange  // Sparse color data
    currentColorTag string      // Track current state for new chars
    // ... other fields
}

type ColorChange struct {
    X, Y int          // Position where color changes
    TViewTag string   // Direct tview color tag: "[red:blue:b]"
}
```

### New Flow
1. **ANSI codes** → **tview color tags** (direct conversion)
2. Store color tag only at positions where color changes
3. Store characters separately in 2D rune array
4. Render by walking runes and inserting color tags at recorded positions

### Benefits
- **Memory**: ~50 color changes per screen vs 2,000 color-storing cells
- **CPU**: No per-character color processing during render
- **Simplicity**: Eliminate hex conversion layer entirely
- **Performance**: Better cache locality, fewer allocations

## Implementation Phases

### Phase 1: Update ANSI Converter ✅ COMPLETED
**Goal**: Make converter output tview tags directly instead of hex colors

**Files modified**:
- `internal/ansi/converter.go`
- `internal/terminal/buffer.go` 
- `internal/terminal/buffer_test.go`
- `internal/ansi/converter_test.go`

**Changes made**:
1. Modified `ConvertANSIParams()` to return tview color tag string
2. Removed unnecessary `ANSIConverter` interface - now uses concrete `*ansi.ColorConverter` type
3. Updated terminal buffer to use direct converter and temporary backward compatibility parsing
4. Simplified tests to use real converter instead of mocks

**Example**:
```go
func (c *ColorConverter) ConvertANSIParams(params string) string {
    // ... existing ANSI processing logic ...
    
    // Instead of returning hex colors:
    // return fgHex, bgHex, bold, underline, reverse
    
    // Return tview tag directly:
    return fmt.Sprintf("[#%02x%02x%02x:#%02x%02x%02x%s]", 
        fgR, fgG, fgB, bgR, bgG, bgB, attributes)
}
```

### Phase 2: Update Terminal Buffer Data Structure ✅ IN PROGRESS
**Goal**: Replace Cell-based storage with rune buffer + color changes

**Files modified**:
- `internal/terminal/buffer.go`
- `internal/terminal/buffer_test.go`

**Changes completed**:
1. Added `runes [][]rune` for character storage
2. Added `colorChanges []ColorChange` for sparse color data  
3. Added `currentColorTag string` for state tracking
4. Updated `putChar()` to store runes in new buffer
5. Updated ANSI processing to record color changes only when they actually change
6. Updated `clear()` function to work with new structure
7. Added getter methods: `GetRunes()`, `GetColorChanges()`
8. Added comprehensive test to verify new data structure works

**Current Status**: Both old and new systems running in parallel for backward compatibility  
**Next Step**: Phase 3 must update the terminal component to use new data structure before Phase 2 cleanup can be completed

**Key functions to update**:
- `putChar()` - store rune, track color changes
- `clear()`, `clearLine()`, etc. - work with rune array
- `processANSISequence()` - store color changes instead of updating cell colors

### Phase 3: Update Terminal Rendering ✅ COMPLETED
**Goal**: Render by walking runes and inserting color tags at positions

**Files modified**:
- `internal/tui/components/terminal.go`
- `internal/terminal/buffer.go` (scrolling fix)

**Changes completed**:
1. Added new `renderRunesWithColorTags()` function that uses sparse color data
2. Updated `UpdateContent()` to use `GetRunes()` and `GetColorChanges()` instead of `GetCells()`
3. Fixed terminal buffer scrolling to work with new rune buffer and color changes
4. Updated scroll logic to properly adjust color change Y coordinates
5. All tests passing including ANSI colors, scrolling, and persistence

**Performance Improvement**: Instead of checking colors on every cell, now only applies color tags at sparse change points

**New rendering logic**:
```go
func (tc *TerminalComponent) renderRunesWithColorTags() {
    colorIndex := 0
    
    for y, row := range tc.terminal.GetRunes() {
        var lineBuilder strings.Builder
        
        for x, char := range row {
            // Insert color tag if position matches a color change
            if colorIndex < len(tc.terminal.GetColorChanges()) {
                change := tc.terminal.GetColorChanges()[colorIndex]
                if change.Y == y && change.X == x {
                    lineBuilder.WriteString(change.TViewTag)
                    colorIndex++
                }
            }
            
            if char != 0 { // Skip null characters
                lineBuilder.WriteRune(char)
            }
        }
        
        // Add newline except for last row
        if y < len(tc.terminal.GetRunes())-1 {
            lineBuilder.WriteRune('\n')
        }
        
        tc.view.Write([]byte(lineBuilder.String()))
    }
}
```

### Phase 4: Update Terminal Interface
**Goal**: Provide clean getter methods for new data structure

**Files to modify**:
- `internal/terminal/buffer.go`

**Changes**:
1. Add `GetRunes() [][]rune` method
2. Add `GetColorChanges() []ColorChange` method
3. **Remove `GetCells()` method entirely** (no backwards compatibility needed)
4. Update other methods that expose internal structure

### Phase 5: Cleanup and Testing
**Goal**: Remove unused code and verify functionality

**Files to modify**:
- All files that reference the old Cell-based approach

**Changes**:
1. **Remove `Cell` struct definition entirely**
2. **Remove all hex color conversion utilities**
3. **Remove any code that depends on old `GetCells()` API**
4. Update tests to work with new data structure
5. Performance testing to verify improvements

**Note**: Since backwards compatibility is not required, we can aggressively remove old code without deprecation periods.

## Testing Strategy

### Performance Benchmarks
1. **Memory usage**: Before vs after with typical terminal content
2. **Render time**: Measure rendering performance with color-heavy content
3. **Color change frequency**: Verify assumption about sparse color changes

### Functionality Tests
1. **Color rendering**: Ensure colors display correctly
2. **Color transitions**: Test color changes work properly
3. **ANSI sequences**: Verify all supported ANSI codes work
4. **Edge cases**: Empty screens, full-color screens, rapid color changes

## Risk Assessment

### Low Risk
- Performance improvements are measurable
- Data structure change is contained within terminal module
- Rendering logic becomes simpler
- **No backwards compatibility required** - this is an internal optimization

### Medium Risk
- Need to ensure color change tracking is accurate
- ANSI converter changes affect color accuracy
- Cursor positioning with sparse data structure

### Mitigation
- Implement phases incrementally with testing at each step
- **No need to preserve old API** - can make breaking changes freely
- Add extensive logging during transition

## Expected Outcomes

### Performance Improvements
- **Memory**: 60-80% reduction in color-related storage
- **CPU**: 40-60% reduction in rendering overhead
- **Allocations**: Fewer objects created during rendering

### Code Quality
- Simpler rendering logic
- More intuitive data structures
- Elimination of redundant conversion layers