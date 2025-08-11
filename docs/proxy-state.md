# Proxy State Pattern Implementation Guide

This document provides detailed instructions for implementing the state pattern in `internal/proxy/proxy.go` to eliminate mutex locking.

## Overview

Replace the current mutex-based synchronization with a state pattern where `Proxy` holds an atomic reference to a `ProxyState` interface. This eliminates race conditions by making state immutable.

## Step 1: Define ProxyState Interface

Create the interface in `proxy.go`:

```go
type ProxyState interface {
    // Core operations that vary by connection state
    SendToTUI(output string) error      // Script output → pipeline → TUI
    SendToServer(input string) error    // User input → script processing → server
    IsConnected() bool
    GetParser() *streaming.TWXParser
    
    // Internal operations for I/O handlers
    writeServerData(data string) error      // Direct write to server connection
    readServerData(buffer []byte) (int, error)  // Direct read from server connection
    processServerData(data []byte)          // Process server data → pipeline → TUI
    
    // Resource cleanup
    Close() error
}
```

## Step 2: Implement DisconnectedState

```go
type DisconnectedState struct{}

func NewDisconnectedState() *DisconnectedState {
    return &DisconnectedState{}
}

func (s *DisconnectedState) SendToTUI(output string) error {
    // Drop output when disconnected (current behavior)
    return nil
}

func (s *DisconnectedState) SendToServer(input string) error {
    // No-op when disconnected
    return nil
}

func (s *DisconnectedState) IsConnected() bool {
    return false
}

func (s *DisconnectedState) GetParser() *streaming.TWXParser {
    return nil
}

func (s *DisconnectedState) writeServerData(data string) error {
    return fmt.Errorf("not connected")
}

func (s *DisconnectedState) readServerData(buffer []byte) (int, error) {
    return 0, fmt.Errorf("not connected")
}

func (s *DisconnectedState) processServerData(data []byte) {
    // No-op when disconnected
}

func (s *DisconnectedState) Close() error {
    return nil // Nothing to close
}
```

## Step 3: Implement ConnectedState

```go
type ConnectedState struct {
    // Network components
    conn     net.Conn
    reader   *bufio.Reader
    writer   *bufio.Writer
    pipeline *streaming.Pipeline
    
    // Processing components - always present when connected
    scriptManager *scripting.ScriptManager
    gameDetector  *GameDetector
}

func NewConnectedState(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer, pipeline *streaming.Pipeline, scriptManager *scripting.ScriptManager, gameDetector *GameDetector) *ConnectedState {
    return &ConnectedState{
        conn:          conn,
        reader:        reader, 
        writer:        writer,
        pipeline:      pipeline,
        scriptManager: scriptManager,
        gameDetector:  gameDetector,
    }
}

func (s *ConnectedState) SendToTUI(output string) error {
    data := []byte(output + "\r\n")
    if s.pipeline != nil {
        s.pipeline.InjectTUIData(data)
    }
    return nil
}

func (s *ConnectedState) SendToServer(input string) error {
    // Process through script manager - no nil check needed, always present
    s.scriptManager.ProcessOutgoingText(input)
    
    // Process through game detector - no nil check needed, always present  
    s.gameDetector.ProcessUserInput(input)
    
    // Then write directly to server
    return s.writeServerData(input)
}

func (s *ConnectedState) IsConnected() bool {
    return true
}

func (s *ConnectedState) GetParser() *streaming.TWXParser {
    if s.pipeline == nil {
        return nil
    }
    return s.pipeline.GetParser()
}

func (s *ConnectedState) writeServerData(data string) error {
    _, err := s.writer.WriteString(data)
    if err != nil {
        return err
    }
    return s.writer.Flush()
}

func (s *ConnectedState) readServerData(buffer []byte) (int, error) {
    return s.reader.Read(buffer)
}

func (s *ConnectedState) processServerData(data []byte) {
    if s.pipeline != nil {
        s.pipeline.Write(data)
    }
}

func (s *ConnectedState) Close() error {
    // Stop pipeline first
    if s.pipeline != nil {
        s.pipeline.Stop()
    }
    
    // Close connection
    if s.conn != nil {
        return s.conn.Close()
    }
    return nil
}
```

## Step 4: Modify Proxy Struct

```go
type Proxy struct {
    // REMOVE these fields:
    // conn      net.Conn
    // reader    *bufio.Reader
    // writer    *bufio.Writer
    // mu        sync.RWMutex
    // connected bool
    // pipeline  *streaming.Pipeline
    
    // ADD this field:
    state atomic.Value // holds ProxyState
    
    // Keep all other fields unchanged
    outputChan chan string
    inputChan  chan string
    errorChan  chan error
    // ... rest unchanged
}
```

## Step 5: Add State Helper Methods

```go
func (p *Proxy) getState() ProxyState {
    state := p.state.Load()
    if state == nil {
        return NewDisconnectedState()
    }
    return state.(ProxyState)
}

func (p *Proxy) setState(newState ProxyState) {
    // Close old state's resources
    if oldState := p.state.Load(); oldState != nil {
        if closable, ok := oldState.(ProxyState); ok {
            closable.Close()
        }
    }
    
    p.state.Store(newState)
}
```

## Step 6: Update Constructor (New)

In `New()` function around line 90:
```go
p := &Proxy{
    outputChan:   make(chan string, 100),
    inputChan:    make(chan string, 100),
    errorChan:    make(chan error, 100),
    tuiAPI:       tuiAPI,
    // Remove: connected: false, pipeline: nil
}

// Initialize with disconnected state
p.setState(NewDisconnectedState())
```

## Step 7: Update Connect Method

In `Connect()` method around line 221:

**Remove these lines:**
```go
p.mu.Lock()
defer p.mu.Unlock()
if p.connected {
    return fmt.Errorf("already connected")
}
```

**Replace with:**
```go
if p.getState().IsConnected() {
    return fmt.Errorf("already connected")
}
```

**Remove these lines (around 287):**
```go
p.conn = conn
p.reader = bufio.NewReader(conn)
p.writer = bufio.NewWriter(conn)
p.connected = true
```

**Replace these lines (around 299):**
```go
p.pipeline = streaming.NewPipelineWithWriter(...)
p.pipeline.Start()
```

**With:**
```go
reader := bufio.NewReader(conn)
writer := bufio.NewWriter(conn)

writerFunc := func(data []byte) error {
    _, err := writer.Write(data)
    if err != nil {
        return err
    }
    return writer.Flush()
}

pipeline := streaming.NewPipelineWithWriter(p.tuiAPI, p.db, p.scriptManager, p, p.gameDetector, writerFunc)
pipeline.Start()

// Create and set connected state
connectedState := NewConnectedState(conn, reader, writer, pipeline, p.scriptManager, p.gameDetector)
p.setState(connectedState)
```

## Step 8: Update Disconnect Method

In `Disconnect()` method around line 339:

**Replace the entire method body:**
```go
func (p *Proxy) Disconnect() error {
    if !p.getState().IsConnected() {
        return nil
    }

    // Stop all scripts
    if p.scriptManager != nil {
        p.scriptManager.Stop()
    }

    // Close game detector
    if p.gameDetector != nil {
        p.gameDetector.Close()
        p.gameDetector = nil
    }

    // Transition to disconnected state (this will close resources)
    p.setState(NewDisconnectedState())

    return nil
}
```

## Step 9: Update State-Dependent Methods

**IsConnected() (line 373):**
```go
func (p *Proxy) IsConnected() bool {
    return p.getState().IsConnected()
}
```

**SendToTUI() (formerly SendOutput, line 387):**
```go
func (p *Proxy) SendToTUI(output string) {
    err := p.getState().SendToTUI(output)
    if err != nil {
        // Log error but don't fail - maintains current behavior
        debug.Log("SendToTUI error: %v", err)
    }
}
```

**SendToServer() (formerly SendDirectToServer, line 405):**
```go
func (p *Proxy) SendToServer(input string) {
    state := p.getState()
    
    if !state.IsConnected() {
        return
    }

    // State handles all processing internally - no nil checks needed
    err := state.SendToServer(input)
    if err != nil {
        p.errorChan <- fmt.Errorf("write error: %w", err)
    }
}
```

**GetParser() (line 665):**
```go
func (p *Proxy) GetParser() *streaming.TWXParser {
    return p.getState().GetParser()
}
```

**injectInboundData() (line 622):**
```go
func (p *Proxy) injectInboundData(data []byte) {
    p.getState().processServerData(data)
}
```

## Step 10: Update I/O Handler Methods

**handleInput() (line 447):**

Remove these lines:
```go
p.mu.RLock()
connected := p.connected && p.writer != nil
p.mu.RUnlock()

if !connected {
    continue
}
```

Replace with:
```go
state := p.getState()
if !state.IsConnected() {
    continue
}
```

Remove these lines:
```go
_, err := p.writer.WriteString(input)
if err != nil {
    p.errorChan <- fmt.Errorf("write error: %w", err)
    continue
}

err = p.writer.Flush()
if err != nil {
    p.errorChan <- fmt.Errorf("flush error: %w", err)
}
```

Replace with:
```go
err := state.writeServerData(input)
if err != nil {
    p.errorChan <- fmt.Errorf("write error: %w", err)
}
```

**handleOutput() (line 574):**

Remove these lines:
```go
p.mu.RLock()
connected := p.connected
reader := p.reader
p.mu.RUnlock()

if !connected {
    break
}

if reader == nil {
    break
}
```

Replace with:
```go
state := p.getState()
if !state.IsConnected() {
    break
}
```

Replace:
```go
n, err := reader.Read(buffer)
```

With:
```go
n, err := state.readServerData(buffer)
```

Replace:
```go
p.pipeline.Write(rawData)
```

With:
```go
state.processServerData(rawData)
```

Remove this section:
```go
p.mu.Lock()
p.connected = false
p.mu.Unlock()
```

Replace with:
```go
p.setState(NewDisconnectedState())
```

## Step 11: Update onDatabaseLoaded Method

In `onDatabaseLoaded()` method around line 729:

**Remove these lines:**
```go
p.mu.Lock()
defer p.mu.Unlock()
```

**Replace the pipeline recreation section (lines 744-757):**
```go
// If we have a connected state, recreate it with new pipeline
currentState := p.getState()
if connectedState, ok := currentState.(*ConnectedState); ok {
    writerFunc := func(data []byte) error {
        _, err := connectedState.writer.Write(data)
        if err != nil {
            return err
        }
        return connectedState.writer.Flush()
    }

    newPipeline := streaming.NewPipelineWithWriter(p.tuiAPI, p.db, p.scriptManager, p, p.gameDetector, writerFunc)
    newPipeline.Start()

    // Create new connected state with same conn/reader/writer but new pipeline
    newConnectedState := NewConnectedState(
        connectedState.conn,
        connectedState.reader,
        connectedState.writer,
        newPipeline,
        connectedState.scriptManager,
        connectedState.gameDetector,
    )
    
    p.setState(newConnectedState)
}
```

## Step 12: Add Import

Add to imports if not already present:
```go
import (
    "sync/atomic"
    // ... other imports
)
```

## Step 13: Testing

After implementation:
1. Run `make test` to ensure all tests pass
2. Test connection/disconnection cycles
3. Test database reloading during active connections
4. Verify no data races with `go test -race`

## Notes

- All mutex locks (`p.mu.Lock()`, `p.mu.RLock()`) should be completely removed
- The `mu sync.RWMutex` field should be removed from the Proxy struct
- State transitions are atomic via `atomic.Value`
- Resource cleanup happens automatically in `setState()` via the `Close()` method
- Each state is immutable - never modify fields after construction
- Remember to update all callers of the renamed methods:
  - `SendOutput` → `SendToTUI` 
  - `SendDirectToServer` → `SendToServer`