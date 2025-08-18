# Logging Format and Test Infrastructure Refactoring Plan

## Overview
This plan outlines the changes needed to modify the debug logging format and test infrastructure to use directional markers that clearly indicate data flow direction in the proxy system.

## Current State Analysis

### Current LogDataChunk Usage
Based on actual code examination:

1. **internal/debug/logger.go:96** - `LogDataChunk(source string, data []byte)`
   - Writes to `raw.log` file with format: `{source} chunk ({size} bytes):\n{escaped_data}\n`

2. **internal/proxy/proxy_api_impl.go:65** - `debug.LogDataChunk("SendData", data)`  
   - Called when user sends input via `SendData()` (user input going to server)

3. **internal/tui/api/tui_api_impl.go:55** - `debug.LogDataChunk("OnData", data)`
   - Called when data comes from server via pipeline to `OnData()` (server data going to client)

### Current Data Flow
```
User Input: User → ProxyApiImpl.SendData() → LogDataChunk("SendData") → proxy.SendInput() → server
Server Data: Server → pipeline.Write() → pipeline.batchProcessor() → tuiAPI.OnData() → LogDataChunk("OnData") → TuiApiImpl
```

### Current Test Format (probe_test.script)
- Uses `<` for server-to-client data
- Uses `>` for client-to-server data (user input)

## Proposed Changes

### New Directional Format
- `>>` = User typed input (currently logged as "SendData") 
- `<<` = Raw server data with encoded ANSI (before processing) - reflects actual network chunks as received
- `<` = All data sent to TuiApi.OnData (server content with ANSI stripped + menu output)

## Accurate Implementation Plan

### Phase 1: Update LogDataChunk Function
**File**: `internal/debug/logger.go`
- Change `LogDataChunk(source string, data []byte)` to `LogDataChunk(direction string, data []byte)`
- Update format from `{source} chunk ({size} bytes):\n{data}` to `{direction} {escaped_data}\n`
- Match the exact format used in `probe_test.script`

### Phase 2: Update LogDataChunk Call Sites
**File**: `internal/proxy/proxy_api_impl.go:65`
- Change `debug.LogDataChunk("SendData", data)` to `debug.LogDataChunk(">>", data)`

**File**: `internal/tui/api/tui_api_impl.go:55` 
- Change `debug.LogDataChunk("OnData", data)` to `debug.LogDataChunk("<", data)`
- This will log ALL data sent to TUI (server content with ANSI stripped + menu output)

### Phase 3: Add Raw Server Data Logging
**Challenge**: We need to capture raw server data with encoded ANSI before it gets processed.

**Solution**: Add `<<` logging where raw server data first enters the pipeline:
- Add `debug.LogDataChunk("<<", data)` at the beginning of `pipeline.Write(data []byte)` in `pipeline.go`
- This captures the raw server data with encoded ANSI sequences before any processing

### Phase 4: Update Test Infrastructure
**File**: `integration/parsing/probe_test.script`
- Keep `<` as is (this represents processed TUI data for client expect scripts)
- Change all `>` to `>>` (user input)

**File**: `integration/scripting/proxy_tester.go`
- **Line 59**: Update regex from `^([<>])\\s+(.+)$` to `^(<<|>>|<)\\s+(.+)$`
- **Line 95**: Update direction check from `== "<"` to handle both `"<"` and `"<<"` 
- **Line 118**: Update direction check from `== ">"` to `== ">>"`
- **Lines 100, 123**: Update logic to handle `>>` for user input and `<` for display expects
- Add handling for `<<` direction (raw server data) - likely not used in test expects, just parsing
- **Note**: Existing telnet handling works correctly:
  - `processEscapeSequences()` converts `\xff\xfb\x01` → actual telnet bytes
  - `cleanTelnetInput()` strips telnet commands from client input  
  - `strconv.Unquote()` in `generateExpectPattern()` handles escape sequences

**File**: `integration/scripting/proxy_tester_test.go`
- Update test cases to use `>>` for user input
- Keep `<` for client expect patterns (processed TUI data)
- Update expected server/client script outputs in test assertions

## Accurate File Change Summary

1. **internal/debug/logger.go** - Update function signature and format:
   - Change `LogDataChunk(source string, data []byte)` to `LogDataChunk(direction string, data []byte)`
   - Update format from `{source} chunk ({size} bytes):\n{data}` to `{direction} {escaped_data}\n`

2. **internal/proxy/proxy_api_impl.go:65** - Change user input logging:
   - `debug.LogDataChunk("SendData", data)` → `debug.LogDataChunk(">>", data)`

3. **internal/tui/api/tui_api_impl.go:55** - Change TUI data logging:
   - `debug.LogDataChunk("OnData", data)` → `debug.LogDataChunk("<", data)`

4. **internal/proxy/streaming/pipeline.go:216** - Add raw server data logging:
   - Add `debug.LogDataChunk("<<", rawData)` before telnet processing in `batchProcessor()`

5. **integration/parsing/probe_test.script** - Update test script format:
   - Keep `<` as is (processed TUI data for expects)
   - Change `>` to `>>` (user input)

6. **integration/scripting/proxy_tester.go** - Update parsing logic:
   - Line 59: Regex `^([<>])\\s+(.+)$` → `^(<<|>>|<)\\s+(.+)$`
   - Line 95: Handle both `"<"` and `"<<"` directions
   - Line 118: Change `">"` to `">>"`
   - Update `ConvertToExpectScripts()` logic for new directions

7. **integration/scripting/proxy_tester_test.go** - Update test cases:
   - Change test inputs from `">"` to `">>"`  
   - Update expected script outputs in assertions
   - Keep `"<"` for client expect patterns

## Critical Implementation Details

### Data Flow Accuracy
- `>>` should capture user input at `proxy_api_impl.go:65` (SendData)
- `<<` should capture raw network data at `pipeline.go:216` before telnet protocol processing - preserving original network chunk boundaries
- `<` should capture all processed TUI data at `tui_api_impl.go:55` (OnData)

### Exact Implementation Locations
**Raw Server Data (<<)**:
```go
// At pipeline.go:216, in batchProcessor() before telnet processing:
case rawData := <-p.rawDataChan:
    debug.LogDataChunk("<<", rawData)  // ADD THIS LINE - raw network data before telnet processing
    // Process telnet commands immediately
    cleanData := p.telnetHandler.ProcessData(rawData)
    // ... rest of function
```

**Processed TUI Data (<)**:
```go  
// At tui_api_impl.go:55, change existing line:
func (tui *TuiApiImpl) OnData(data []byte) {
    debug.LogDataChunk("<", data)     // CHANGED FROM "OnData" - all processed TUI data
    // ... rest of function
}
```

This ensures:
- Raw network data with ANSI encoding and telnet protocol bytes is logged as `<<` in exact chunks received
- All processed data sent to TUI (server + menu) is logged as `<` after telnet processing and ANSI stripping  
- Client expect scripts use `<` data since that's what gets displayed
- Test scripts can keep using `<` for expects since that represents processed display data
- Network debugging benefits from seeing actual chunk boundaries and telnet protocol data in `<<`
- Existing telnet handling in test infrastructure continues to work without modification

### Implementation Order
1. **First**: Update `LogDataChunk()` function signature and format
2. **Second**: Update all call sites to use new directional markers
3. **Third**: Update test infrastructure parsing and regex
4. **Fourth**: Update test cases and assertions
5. **Fifth**: Run integration tests to verify everything works

### Test Script Compatibility
- Existing scripts use single `<` and `>`
- New scripts will use `<<` and `>>`  
- Need backward compatibility or migration strategy