# Script Integration Analysis & Implementation Plan

## Current State Analysis

### Scripting Engine Status: ✅ COMPLETE
Located in `internal/scripting/`, the TWX-compatible scripting engine is fully implemented with:

- **Engine** (`engine.go`): Complete script loading, execution, and management
- **Virtual Machine** (`vm/`): Full TWX command set implementation
- **Triggers** (`triggers/`): Text pattern matching and event handling
- **Game Adapter** (`integration.go`): Database integration for sector/game data
- **Script Manager** (`integration.go:216`): High-level script management interface

### Proxy Data Flow: ⚠️ NO SCRIPTING INTEGRATION
Current data pipeline:
```
Server → Proxy (proxy.go) → Pipeline (pipeline.go) → Telnet Handler → Terminal
                                        ↓
                              Sector Parser (parser/)
```

**Key Integration Points Identified:**
- `proxy.go:35` - Proxy constructor (add ScriptManager)
- `pipeline.go:159` - Text processing point (add script hooks)
- `proxy.go:169` - Input handling (add outgoing text processing)

## TWX Compatibility Requirements

**CRITICAL**: Our implementation must match TWX code `ScriptCmp.pas` as much as possible to ensure compatibility with existing TWX scripts. Key areas for compatibility:

- **Text Processing Pipeline**: Match TWX's text processing order and trigger execution
- **Command Interception**: Replicate TWX's command processing and modification behavior  
- **Script Event Handling**: Follow TWX's script lifecycle and event notification patterns
- **Variable Persistence**: Match TWX's variable storage and retrieval mechanisms
- **Error Handling**: Replicate TWX's script error reporting and recovery behavior

Reference the original TWX `ScriptCmp.pas` implementation when making integration decisions.

## Integration Implementation Phases

### Phase 1: Basic Script Manager Integration
**Goal**: Connect scripting engine to proxy without breaking existing functionality

**Files to Modify:**
- `internal/proxy/proxy.go`

**Changes:**
1. Add ScriptManager field to Proxy struct
2. Initialize ScriptManager in `New()` constructor
3. Wire script manager to use proxy's database instance
4. Add basic script management methods to Proxy

**Estimated Effort**: 2-3 hours

### Phase 2: Incoming Text Processing
**Goal**: Process all incoming game text through scripting triggers

**Files to Modify:**
- `internal/streaming/pipeline.go`

**Changes:**
1. Add ScriptManager reference to Pipeline struct
2. Modify `batchProcessor()` at line 159 to call:
   ```go
   scriptManager.ProcessGameLine(string(decoded))
   ```
3. Handle script processing errors gracefully
4. Ensure script processing doesn't block the pipeline

**Integration Point**: 
```go
// At pipeline.go:159, after sectorParser.ProcessData(decoded)
if p.scriptManager != nil {
    if err := p.scriptManager.ProcessGameLine(string(decoded)); err != nil {
        p.logger.Printf("Script processing error: %v", err)
    }
}
```

**Estimated Effort**: 3-4 hours

### Phase 3: Outgoing Command Processing  
**Goal**: Process all outgoing commands through scripting triggers

**Files to Modify:**
- `internal/proxy/proxy.go`

**Changes:**
1. Modify `handleInput()` method at line 169
2. Add call to `scriptManager.ProcessOutgoingText()` before sending to server
3. Handle script command interception/modification
4. Support script-generated commands

**Integration Point**:
```go
// At proxy.go:179, before writer.WriteString(input)
if p.scriptManager != nil {
    if err := p.scriptManager.ProcessOutgoingText(input); err != nil {
        p.logger.Printf("Outgoing script processing error: %v", err)
    }
}
```

**Estimated Effort**: 2-3 hours

### Phase 4: Script Command Integration
**Goal**: Enable scripts to send commands and receive output

**Files to Modify:**
- `internal/scripting/engine.go`
- `internal/scripting/integration.go`

**Changes:**
1. Wire script engine handlers to proxy:
   - `SetSendHandler()` → proxy command sending
   - `SetOutputHandler()` → terminal output
   - `SetEchoHandler()` → local echo
2. Complete `GameAdapter.SendCommand()` implementation
3. Complete `GameAdapter.GetLastOutput()` implementation

**Estimated Effort**: 4-5 hours

### Phase 5: Script Management UI
**Goal**: Add script control to TUI interface

**Files to Modify:**
- `internal/tui/app.go`
- `internal/tui/components/` (new script panel)

**Changes:**
1. Add script management panel to TUI
2. Display running scripts status
3. Add script load/unload/start/stop controls
4. Show script output/errors in dedicated area

**Estimated Effort**: 6-8 hours

## Integration Challenges & Solutions

### Challenge 1: Threading Safety
**Issue**: Scripts run in goroutines, proxy handles concurrent data
**Solution**: Use channels for script-to-proxy communication, mutex protection for shared state

### Challenge 2: Script Error Handling
**Issue**: Script errors shouldn't crash the proxy
**Solution**: Comprehensive error catching with graceful degradation

### Challenge 3: Performance Impact
**Issue**: Script processing could slow down game data flow
**Solution**: Async script processing with buffering, performance monitoring

### Challenge 4: Configuration Management
**Issue**: Scripts need configuration, startup scripts, etc.
**Solution**: Script directory scanning, auto-load configuration, script state persistence

## Testing Strategy

### Unit Tests
- Script manager initialization
- Text processing pipeline integration
- Command interception and modification
- Error handling and recovery

### Integration Tests
- Full proxy + scripting data flow
- Script trigger activation
- Command sending and receiving
- Multi-script coordination

### Manual Testing
- Load existing TWX scripts
- Verify trigger activation on game text
- Test script-generated commands
- Validate UI script management

## Success Criteria

1. ✅ Existing proxy functionality unchanged
2. ✅ TWX scripts can be loaded and executed
3. ✅ Incoming text triggers work correctly
4. ✅ Scripts can send commands to game server
5. ✅ Script errors don't crash proxy
6. ✅ Performance impact < 10ms per message
7. ✅ UI provides script management controls

## File Dependencies

```
proxy.go
├── imports scripting/integration.go (ScriptManager)
├── uses database.Database (shared instance)
└── coordinates with streaming/pipeline.go

pipeline.go  
├── receives ScriptManager from proxy
├── calls ProcessGameLine() on decoded text
└── handles script processing errors

integration.go
├── implements GameInterface for proxy interaction
├── provides SendCommand() → proxy.SendInput()
└── provides GetLastOutput() → terminal buffer
```

## Rollback Plan

Each phase can be independently disabled via feature flags:
- `ENABLE_SCRIPT_PROCESSING=false` - disable all script processing
- `ENABLE_SCRIPT_UI=false` - disable script management UI
- Configuration-based script auto-loading can be disabled

Phase 1-3 changes are minimal and easily reversible if issues arise.