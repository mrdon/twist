# Map Update Architecture for Probe-Discovered Sectors

## Overview

This document outlines the architecture for updating sector maps when probes discover new sector data, while maintaining the distinction between actual player movement and probe exploration.

## Current State Analysis

### Current TUI API Events

The system currently uses `OnCurrentSectorChanged(sectorInfo SectorInfo)` as the primary sector change event, which is designed for player movement. Probe-discovered sectors are currently suppressed from this event to avoid misleading the UI about the player's actual location.

```go
type TuiAPI interface {
    OnCurrentSectorChanged(sectorInfo SectorInfo) // Currently player movement only
    OnTraderDataUpdated(sectorNumber int, traders []TraderInfo)
    OnPlayerStatsUpdated(stats PlayerStatsInfo)
    // ... other events
}
```

### Current SectorInfo Structure

```go
type SectorInfo struct {
    Number        int    `json:"number"`
    NavHaz        int    `json:"nav_haz"`
    HasTraders    int    `json:"has_traders"`
    Constellation string `json:"constellation"`
    Beacon        string `json:"beacon"`
    Warps         []int  `json:"warps"`
    HasPort       bool   `json:"has_port,omitempty"`
    Visited       bool   `json:"visited"`  // True only for player visits (EtHolo)
}
```

### Existing Probe Detection Infrastructure

The system already has excellent probe detection logic:

```go
type TWXParser struct {
    probeMode              bool              // True during probe parsing
    probeDiscoveredSectors map[int]bool      // Track probe-discovered sectors
    // ... other fields
}

func (p *TWXParser) handleProbePrompt(line string) {
    if strings.Contains(line, "Probe entering sector :") {
        p.probeMode = true
        p.probeDiscoveredSectors[targetSector] = true  // Mark as probe-discovered
        // ... warp creation logic
    }
}
```

### Database Exploration Status

The database properly tracks exploration types:

```go
const (
    EtNo      TSectorExploredType = iota  // 0 - Unexplored
    EtCalc                               // 1 - Probe data
    EtDensity                            // 2 - Density scanner data  
    EtHolo                               // 3 - Actually visited by player
)
```

## Proposed Architecture

### Option 2: Separate Probe Discovery Event (Recommended)

Add a new dedicated event for any sector data updates:

```go
type TuiAPI interface {
    OnCurrentSectorChanged(sectorInfo SectorInfo)   // Player movement only
    OnSectorDataUpdated(sectorInfo SectorInfo)      // NEW - Any sector data update
    // ... other events
}
```

#### Updated Event Firing Logic

```go
func (p *TWXParser) sectorCompleted() {
    // ... existing sector saving logic ...
    
    if p.tuiAPI != nil {
        sectorData := p.buildSectorData()
        sectorInfo := p.buildSectorInfo(sectorData)
        
        if p.probeMode || p.probeDiscoveredSectors[p.currentSectorIndex] {
            // Fire sector data update for probe discoveries and other non-player updates
            p.tuiAPI.OnSectorDataUpdated(sectorInfo)
        } else {
            // Fire current sector changed for actual player movement
            p.tuiAPI.OnCurrentSectorChanged(sectorInfo)
            // Also fire sector data update since player movement updates sector data too
            p.tuiAPI.OnSectorDataUpdated(sectorInfo)
        }
    }
}
```

#### Map Component Updates

```go
// In internal/tui/components/sector_map.go
func (smc *SectorMapComponent) UpdateCurrentSectorWithInfo(sectorInfo api.SectorInfo) {
    // Handle player movement - update both sector data and current position
    smc.sectorData[sectorInfo.Number] = sectorInfo
    smc.currentSector = sectorInfo.Number
    smc.refreshMap()
}

func (smc *SectorMapComponent) UpdateSectorDataOnly(sectorInfo api.SectorInfo) {
    // Handle any sector data update - probes, trading, combat, etc.
    // Update sector data without changing current player position
    smc.sectorData[sectorInfo.Number] = sectorInfo
    smc.refreshMap()
}
```

**Pros:**
- Clear separation of concerns
- Backward compatibility maintained  
- Explicit event types
- No need to modify SectorInfo structure
- Simpler implementation

**Cons:**
- Map components need to listen to multiple events

## Implementation Plan

### Phase 1: Core Infrastructure (Option 2 Approach)

1. **Add OnSectorDataUpdated event** in `internal/api/api.go`
   - Add new `OnSectorDataUpdated(sectorInfo SectorInfo)` method to TuiAPI interface

2. **Update TWXParser** in `internal/proxy/streaming/twx_parser.go`
   - Modify `sectorCompleted()` to fire appropriate events based on context
   - Use `OnSectorDataUpdated` for all sector data updates (probes, trading, combat, etc.)
   - Keep `OnCurrentSectorChanged` for actual player movement
   - Fire both events for player movement since it also updates sector data

3. **Update MockTuiAPI** in test files
   - Add `OnSectorDataUpdated` implementation for testing

### Phase 2: Map Component Updates

4. **Update SectorMapComponent** in `internal/tui/components/sector_map.go`
   - Add `UpdateSectorDataOnly()` method for non-movement sector updates
   - Connect to both `OnCurrentSectorChanged` and `OnSectorDataUpdated` events
   - Ensure map refreshes for all sector data changes (probes, trading, combat, etc.)

5. **Update GraphViz map** in `internal/tui/components/sector_map_graphviz.go`
   - Add handler for `OnSectorDataUpdated` events
   - Update visualization for any sector data changes
   - Maintain player position tracking separately from sector data updates

6. **Update Sixel map** in `internal/tui/components/sector_map_sixel.go`
   - Add handler for `OnSectorDataUpdated` events  
   - Update visualization for any sector data changes
   - Optimize rendering for frequent updates

### Phase 3: UI Integration

7. **Update TwistApp handlers** in `internal/tui/app.go`
   - Add `HandleSectorDataUpdated` method for the new event
   - Route sector data update events to map components appropriately
   - Add logging/debugging for sector data update events

8. **Update Panel components** if needed
   - Consider connecting panels to `OnSectorDataUpdated` if they need real-time sector updates
   - Ensure all types of sector data updates are displayed appropriately

## Benefits

### For Users
- **Real-time map updates**: Maps show all sector data changes immediately (probes, trading, combat, etc.)
- **Better situational awareness**: Full sector knowledge without confusion about player location
- **Responsive interface**: Maps update for any sector change, not just player movement

### For Developers
- **Dual event model**: Clear separation between player movement and sector data updates
- **Backward compatibility**: Existing `OnCurrentSectorChanged` behavior unchanged
- **Simple implementation**: No need to modify existing SectorInfo structure
- **Extensible**: Easy to handle any sector update type using `OnSectorDataUpdated`

## Testing Strategy

1. **Unit Tests**: Verify appropriate event firing for different sector update types
2. **Integration Tests**: Test map component updates with various sector data changes
3. **Manual Testing**: Verify map updates for probes, trading, combat, and other activities
4. **Regression Testing**: Ensure existing player movement functionality unchanged

## Migration Path

This architecture maintains backward compatibility while adding new functionality:

1. **Existing components** continue to work unchanged with `OnCurrentSectorChanged`
2. **Gradual enhancement** of map components to also listen to `OnSectorDataUpdated`
3. **No breaking changes** to existing TUI API contracts
4. **Optional adoption** - components can choose whether to handle sector data updates

## Alternative Considerations

### Database-Driven Updates

Instead of event-driven updates, maps could poll the database for sector changes. However, this approach has drawbacks:
- **Performance overhead**: Regular database polling
- **Delayed updates**: Not real-time
- **Complexity**: Requires change detection logic

### Hybrid Approach

Combine events with database queries for complete sector information:
- **Events trigger updates**: Real-time notification of changes
- **Database provides details**: Complete sector information on demand
- **Best of both worlds**: Real-time updates with comprehensive data

This hybrid approach could be considered for future enhancements but adds complexity to the initial implementation.