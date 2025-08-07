# Port Information Integration Project

## Overview

This project implements comprehensive port information parsing, storage, and display functionality matching TWX's port handling capabilities. The system will capture detailed port information from game streams and display it in the TUI trader information panel.

## Current State Analysis

### Existing Infrastructure

#### Database Layer ✅ **COMPLETE**
- `TPort` structure fully defined in `internal/proxy/database/structs.go`
- Port information stored in `sectors` table with dedicated port columns
- Support for all port classes (1-9) with proper type conversion
- Product arrays for buy/sell status, percentages, and quantities
- Build time and port destruction tracking implemented

#### Parser Layer ✅ **MOSTLY COMPLETE**  
- Product parsing implemented in `internal/proxy/streaming/product_parser.go`
- Port build time parsing tested and functional
- Port class detection and conversion working
- Port destruction handling implemented
- Integration with TWX stream processing active

#### Missing Components
- **Database Architecture**: Port data embedded in wide sectors table instead of normalized ports table
- **API Layer**: No detailed port information exposed through ProxyAPI
- **TUI Display**: Trader info panel lacks port information sections  
- **ProxyAPI Methods**: Missing port-specific query endpoints

#### ⚠️ **CRITICAL BUG FOUND**
- **Port Class Mapping Inconsistency**: Two different functions have conflicting class-to-type mappings for classes 4-8:
  - `game_state_converters.go`: Class 4=SSB, 5=SBS, 6=BSS, 7=SSS, 8=BBB
  - `parser_utils.go`: BBB=4, SSS=5, SBS=6, BSS=7 
  - **Impact**: Port parsing and display will show incorrect port types
  - **Fix Required**: Standardize mappings based on official TW specification

### **TW2002 Bible Verification**

From `docs/tw2002-bible-2007.html`, the official specification includes:

**Port Classes:**
- **Class 0**: Special Federation ports (Alpha Centauri, Rylos, StarDock) 
- **Class 1-8**: Standard trading ports using B/S notation (B=Buy, S=Sell)
- **Class 9**: StarDock (special case)

**Commodities:** Three main types as confirmed:
- Fuel Ore (cheapest)
- Organics (medium)  
- Equipment (most expensive)

**Port Notation:** BBS format indicates Buy/Buy/Sell for FuelOre/Organics/Equipment respectively

**Current Implementation Status:**
- ✅ Three commodities correctly implemented
- ✅ Port class 0 and 9 special cases recognized
- ⚠️ **INCONSISTENT**: Classes 4-8 mappings conflict between parser and converter
- ✅ BBS notation format correctly understood

## Implementation Plan

### Phase 0: Critical Bug Fixes 🚨 **URGENT**

**Tasks:**
1. **Resolve Port Class Mapping Inconsistency**
   - Standardize the class-to-string mapping between `game_state_converters.go` and `parser_utils.go`
   - Ensure bidirectional consistency: class → string → class should return original value

2. **Fix Conflicting Mappings**
   - Determine correct mapping for classes 4-8 (SSB, SBS, BSS, SSS, BBB)
   - Update both functions to use identical mappings
   - Add unit tests to prevent regression

**Current Conflicts to Resolve:**
```go
// game_state_converters.go (Class → String)
case 4: return "SSB" 
case 5: return "SBS"
case 6: return "BSS" 
case 7: return "SSS"
case 8: return "BBB"

// parser_utils.go (String → Class)  
case "BBB": return 4  // CONFLICT: Should be 8
case "SSS": return 5  // CONFLICT: Should be 7
case "SBS": return 6  // CONFLICT: Should be 5
case "BSS": return 7  // CONFLICT: Should be 6
```

**Files to modify:**
- `internal/proxy/game_state_converters.go` - Port class to string conversion
- `internal/proxy/streaming/parser_utils.go` - String to port class parsing
- Add comprehensive unit tests for bidirectional mapping

### Phase 1: API Layer Enhancement

**Tasks:**
1. **Extend API Structures** (`internal/api/api.go`)
   ```go
   // Enums for type safety
   type ProductType int
   const (
       ProductTypeFuelOre ProductType = iota
       ProductTypeOrganics
       ProductTypeEquipment
   )
   
   type ProductStatus int
   const (
       ProductStatusNone ProductStatus = iota
       ProductStatusBuying
       ProductStatusSelling
   )
   
   type PortClass int
   const (
       PortClassBBS PortClass = iota + 1 // Buy Buy Sell
       PortClassBSB                      // Buy Sell Buy
       PortClassSBB                      // Sell Buy Buy
       PortClassSSB                      // Sell Sell Buy
       PortClassSBS                      // Sell Buy Sell
       PortClassBSS                      // Buy Sell Sell
       PortClassSSS                      // Sell Sell Sell
       PortClassBBB                      // Buy Buy Buy
       PortClassSTD                      // Stardock
   )
   
   type PortInfo struct {
       Name           string        `json:"name"`
       Class          int           `json:"class"`
       ClassType      PortClass     `json:"class_type"`
       BuildTime      int           `json:"build_time"`
       Products       []ProductInfo `json:"products"`
       LastUpdate     time.Time     `json:"last_update"`
       Dead           bool          `json:"dead"`
   }
   
   type ProductInfo struct {
       Type       ProductType   `json:"type"`
       Status     ProductStatus `json:"status"`
       Quantity   int           `json:"quantity"`
       Percentage int           `json:"percentage"`
   }
   ```

2. **Update SectorInfo Structure**
   - Add optional `Port *PortInfo` field to existing SectorInfo
   - Do not maintain backwards compatibility - this is new code, breaking changes are acceptable

3. **Database-to-API Conversion Functions**
   - Create `convertTPortToPortInfo()` function
   - Handle product array conversion to structured format
   - Include proper error handling for empty/null port data

**Files to modify:**
- `internal/api/api.go` - Add new structures
- `internal/api/converters.go` (create if not exists) - Add conversion functions

### Phase 1.5: TUI API Callback Integration ⚠️ **IMPLEMENTED IN THIS SESSION**

**Changes Made:**
- Added `OnPortUpdated(portInfo PortInfo)` method to TuiAPI interface  
- Enhanced PortInfo struct with SectorID field for simplified signature
- Extended TwistApp interface with `HandlePortUpdated` method
- Implemented callback in TuiApiImpl with proper async handling
- Integrated port update notifications in database_integration.go
- Port updates triggered automatically when sector data is saved with port information

**Files Modified:**
- `internal/api/api.go` - Added PortInfo and ProductInfo structures, OnPortUpdated method
- `internal/tui/api/tui_api_impl.go` - Added TwistApp.HandlePortUpdated and OnPortUpdated implementation  
- `internal/proxy/streaming/database_integration.go` - Added port update notifications and helper functions

**API Improvements Needed:**
- Replace ProductInfo.Status string with ProductStatus enum (None, Buying, Selling)
- Replace ProductInfo.Type string with ProductType enum (FuelOre, Organics, Equipment)  
- Replace PortInfo.ClassType string with PortClass enum (BBS, BSB, SBB, SSB, SBS, BSS, SSS, BBB, STD)

**Next Steps:**
- Implement enum types for better type safety
- Test callback integration

### Phase 2: Database Schema Optimization ⚠️ **RECOMMENDED**

**Details**: See `docs/ports-table-proposal.md` for complete analysis and implementation plan.

**Tasks:**
1. Create new ports table schema with proper indexes
2. Implement data migration from sectors.sport_* columns to ports table
3. Update TSector struct and database operations
4. Modify existing code to use separate port queries

**Files to modify:**
- `internal/proxy/database/schema.go` - Add ports table
- `internal/proxy/database/database.go` - Update operations  
- `internal/proxy/database/structs.go` - Modify TSector struct
- `internal/proxy/streaming/converters.go` - Update conversion logic

### Phase 3: ProxyAPI Implementation

**Tasks:**
1. **Extend ProxyAPI Interface** (`internal/api/proxy_api.go`)
   ```go
   // Add to ProxyAPI interface:
   GetPortInfo(sectorNum int) (*PortInfo, error)
   GetPortProducts(sectorNum int) ([]ProductInfo, error)
   GetSectorWithPortInfo(sectorNum int) (*SectorInfo, error) // Enhanced sector query
   ```

2. **Implement ProxyAPI Methods** (`internal/api/proxy_api_impl.go`)
   - `GetPortInfo()`: Query database for detailed port information
   - `GetPortProducts()`: Extract and format product information
   - `GetSectorWithPortInfo()`: Combine sector and port data
   - Handle sectors without ports gracefully (return nil, not error)

3. **Database Integration**
   - Utilize existing database queries from `LoadSector()`
   - Extract port information from TSector.SPort field
   - Implement proper error handling and logging

**Files to modify:**
- `internal/api/proxy_api.go` - Interface definitions
- `internal/api/proxy_api_impl.go` - Method implementations

### Phase 4: TUI Display Enhancement

**Tasks:**
1. **Enhance Trader Info Panel** (`internal/tui/components/panels.go`)
   - Add port information section to `UpdateTraderInfo()`
   - Display port name, class, and build status
   - Show product information with buy/sell indicators
   - Use color coding for different product statuses

2. **Port Information Display Format**
   ```go
   // Example display format:
   info.WriteString("\n[yellow]Port Information[-]\n")
   if portInfo != nil {
       info.WriteString(formatLine("Port Name", portInfo.Name, "cyan"))
       info.WriteString(formatLine("Port Class", fmt.Sprintf("%s (Class %d)", 
           portInfo.ClassType, portInfo.Class), "cyan"))
       
       if portInfo.BuildTime > 0 {
           info.WriteString(formatLine("Build Time", 
               fmt.Sprintf("%d hours remaining", portInfo.BuildTime), "red"))
       }
       
       if portInfo.Dead {
           info.WriteString("[red]PORT DESTROYED[-]\n")
       }
       
       // Product information
       for _, product := range portInfo.Products {
           if product.Status != "None" {
               color := "cyan"
               if product.Status == "Buying" { color = "green" }
               if product.Status == "Selling" { color = "yellow" }
               
               productLine := fmt.Sprintf("%s: %s %s at %d%%", 
                   product.Type, product.Status, 
                   formatNumber(product.Quantity), product.Percentage)
               info.WriteString(fmt.Sprintf("[%s]%s[-]\n", color, productLine))
           }
       }
   } else {
       info.WriteString("[gray]No port in this sector[-]\n")
   }
   ```

3. **Integration with Port Updates**
   - Implement `HandlePortUpdated` method in TUI app to receive port notifications
   - Update port display automatically when port information changes
   - Use existing sector update mechanisms for initial display
   - Handle async updates without blocking UI thread (already implemented via OnPortUpdated callback)

**Files to modify:**
- `internal/tui/components/panels.go` - Display logic
- Potentially `internal/tui/app.go` - If event handling changes needed

### Phase 5: Integration Testing & Polish

**Tasks:**
1. **Integration Testing**
   - Test port information display with various port types
   - Verify build time countdown functionality
   - Test product information accuracy against game data
   - Validate destroyed port handling

2. **Performance Optimization**
   - Profile database queries for port information
   - Optimize TUI update frequency for port data
   - Ensure smooth operation during active trading

3. **Error Handling Enhancement**
   - Implement comprehensive error logging
   - Handle database connection issues gracefully
   - Provide meaningful error messages for debugging

**Files to modify:**
- `integration/` test files - Add port information tests
- Debug logging enhancements

## Implementation Notes

### TWX Compatibility
- Port class mapping matches TWX standards (BBS, SSS, etc.)
- Product parsing handles both standard and alternate formats
- Build time tracking mirrors TWX behavior
- Port destruction detection compatible with TWX logic

### Database Considerations
- Existing `TSector.SPort` structure sufficient for storage
- No database schema changes required
- Leverage existing prepared statements where possible
- Consider caching for frequently accessed port data

### Implementation Requirements
- Port information updates should not block UI
- Database queries optimized for frequent access
- TUI updates should be smooth during active gameplay
- Gracefully handle sectors without ports (common case)
- Log parser errors without disrupting gameplay
- Use debug.Log() for development debugging (remove before final commit)
- **No backwards compatibility required** - this is new functionality, breaking changes are acceptable

## File Locations

### Key Files to Modify
- `internal/api/api.go` - API structures
- `internal/api/proxy_api.go` - Interface definitions  
- `internal/api/proxy_api_impl.go` - Implementation
- `internal/tui/components/panels.go` - UI display

### Supporting Files
- `internal/proxy/database/structs.go` - Database structures (reference only)
- `internal/proxy/streaming/product_parser.go` - Parser logic (reference only)
- `integration/` - Test files for validation

## Dependencies

### Internal Dependencies
- Database layer: `internal/proxy/database/`
- Parser layer: `internal/proxy/streaming/` 
- TUI framework: `internal/tui/`
- API layer: `internal/api/`

### External Dependencies  
- No new external dependencies required
- Utilizes existing sqlite3 and tview frameworks

---

**Total Phases**: 6 (including critical bug fixes and database optimization)