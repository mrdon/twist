# Ports Table Schema Proposal

## Problem Statement

The current schema embeds port data directly in the sectors table with 13 port-related columns. This creates several issues:

1. **Wide Table**: Sectors table has many port columns that are NULL for most sectors (only ~5-10% have ports)
2. **Poor Performance**: Querying port data requires scanning the large sectors table
3. **Complex Queries**: Port-specific queries mix sector and port logic
4. **Future Growth**: Adding port features means more columns in an already wide table
5. **Normalization**: Violates database normalization principles

## Proposed Solution: Separate Ports Table

### New Schema Design

```sql
-- Simplified sectors table (remove port columns)
CREATE TABLE IF NOT EXISTS sectors (
    sector_index INTEGER PRIMARY KEY,
    
    -- Warp array[1..6] (0-indexed in Go)
    warp1 INTEGER DEFAULT 0,
    warp2 INTEGER DEFAULT 0, 
    warp3 INTEGER DEFAULT 0,
    warp4 INTEGER DEFAULT 0,
    warp5 INTEGER DEFAULT 0,
    warp6 INTEGER DEFAULT 0,
    
    -- Basic sector info
    constellation TEXT DEFAULT '',
    beacon TEXT DEFAULT '',
    nav_haz INTEGER DEFAULT 0,
    density INTEGER DEFAULT -1,
    anomaly BOOLEAN DEFAULT FALSE,
    warps INTEGER DEFAULT 0,
    explored INTEGER DEFAULT 0,
    update_time DATETIME,
    
    -- Space objects (TSpaceObject) - keep in sectors as they're sector-specific
    figs_quantity INTEGER DEFAULT 0,
    figs_owner TEXT DEFAULT '',
    figs_type INTEGER DEFAULT 0,
    
    mines_armid_quantity INTEGER DEFAULT 0,
    mines_armid_owner TEXT DEFAULT '',
    
    mines_limpet_quantity INTEGER DEFAULT 0,
    mines_limpet_owner TEXT DEFAULT ''
);

-- New dedicated ports table
CREATE TABLE IF NOT EXISTS ports (
    sector_index INTEGER PRIMARY KEY,  -- 1:1 relationship with sectors
    name TEXT NOT NULL DEFAULT '',
    class_index INTEGER NOT NULL DEFAULT 0,
    dead BOOLEAN DEFAULT FALSE,
    build_time INTEGER DEFAULT 0,
    
    -- Product information (normalized)
    buy_fuel_ore BOOLEAN DEFAULT FALSE,
    buy_organics BOOLEAN DEFAULT FALSE, 
    buy_equipment BOOLEAN DEFAULT FALSE,
    percent_fuel_ore INTEGER DEFAULT 0,
    percent_organics INTEGER DEFAULT 0,
    percent_equipment INTEGER DEFAULT 0,
    amount_fuel_ore INTEGER DEFAULT 0,
    amount_organics INTEGER DEFAULT 0,
    amount_equipment INTEGER DEFAULT 0,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (sector_index) REFERENCES sectors(sector_index) ON DELETE CASCADE
);

-- Optimized indexes for port queries
CREATE INDEX IF NOT EXISTS idx_ports_name ON ports(name) WHERE name != '';
CREATE INDEX IF NOT EXISTS idx_ports_class ON ports(class_index);
CREATE INDEX IF NOT EXISTS idx_ports_building ON ports(build_time) WHERE build_time > 0;
CREATE INDEX IF NOT EXISTS idx_ports_buying_ore ON ports(buy_fuel_ore) WHERE buy_fuel_ore = TRUE;
CREATE INDEX IF NOT EXISTS idx_ports_buying_org ON ports(buy_organics) WHERE buy_organics = TRUE;
CREATE INDEX IF NOT EXISTS idx_ports_buying_equ ON ports(buy_equipment) WHERE buy_equipment = TRUE;
CREATE INDEX IF NOT EXISTS idx_ports_updated ON ports(updated_at);
```

## Benefits

### 1. **Performance Improvements**
- **Faster Port Queries**: Direct queries on ports table instead of scanning sectors
- **Better Indexes**: Port-specific indexes for common query patterns
- **Smaller Sectors Table**: Faster sector queries without port data overhead

### 2. **Storage Efficiency**
- **No NULL Pollution**: Only sectors with ports have port records
- **Better Compression**: Port data clustered together improves compression
- **Cleaner Schema**: Logical separation of concerns

### 3. **Query Simplification**
```sql
-- Old: Complex WHERE clauses to filter port data
SELECT * FROM sectors WHERE sport_name != '' AND sport_class_index > 0;

-- New: Simple port queries
SELECT * FROM ports WHERE class_index > 0;

-- Port with sector info (when needed)
SELECT s.constellation, s.beacon, p.name, p.class_index 
FROM sectors s 
JOIN ports p ON s.sector_index = p.sector_index;
```

### 4. **Future Extensibility**
- Easy to add port-specific features without bloating sectors table
- Port history tracking possible with additional tables
- Better support for advanced port queries and analytics

## Migration Strategy

### Phase 1: Schema Migration
1. Create new `ports` table alongside existing schema
2. Migrate existing port data from sectors to ports table
3. Verify data integrity

### Phase 2: Code Updates
1. Update `TSector` struct to reference port separately
2. Modify database operations to use both tables
3. Update API queries to join sectors and ports when needed

### Phase 3: Cleanup
1. Remove port columns from sectors table
2. Update indexes and constraints
3. Performance testing and optimization

## Code Impact Analysis

### Database Layer Changes
```go
// Updated TSector struct
type TSector struct {
    // ... existing sector fields ...
    
    // Remove embedded SPort - use separate queries
    // SPort TPort `json:"sport"` // REMOVED
    
    // ... other fields ...
}

// Separate port operations
type TPort struct {
    SectorIndex  int       `json:"sector_index"`
    Name         string    `json:"name"`
    ClassIndex   int       `json:"class_index"`
    Dead         bool      `json:"dead"`
    BuildTime    int       `json:"build_time"`
    BuyProduct   [3]bool   `json:"buy_product"`
    ProductPercent [3]int  `json:"product_percent"`
    ProductAmount  [3]int  `json:"product_amount"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

### New Database Operations
```go
// Port-specific operations
func (d *SQLiteDatabase) SavePort(port TPort) error
func (d *SQLiteDatabase) LoadPort(sectorIndex int) (TPort, error)
func (d *SQLiteDatabase) DeletePort(sectorIndex int) error
func (d *SQLiteDatabase) FindPortsByClass(classIndex int) ([]TPort, error)
func (d *SQLiteDatabase) FindPortsBuying(product ProductType) ([]TPort, error)
```

## API Impact

### Enhanced ProxyAPI Methods
```go
// Existing methods work the same but with better performance
func (api *ProxyAPI) GetPortInfo(sectorNum int) (*PortInfo, error) {
    // Now queries ports table directly instead of sectors
}

// New efficient port queries possible
func (api *ProxyAPI) FindPortsByClass(class int) ([]PortInfo, error)
func (api *ProxyAPI) FindPortsBuyingProduct(product string) ([]PortInfo, error)
func (api *ProxyAPI) GetPortsInConstellation(constellation string) ([]PortInfo, error)
```

## Implementation Priority

### High Priority ✅ **RECOMMENDED**
The ports table separation provides immediate benefits:
- Better performance for port queries
- Cleaner database design
- Foundation for port-focused features
- Follows database normalization best practices

### Timeline Estimate
- **Schema Design**: 1 day (design + review)
- **Migration Implementation**: 2 days (code + testing)  
- **Code Updates**: 2-3 days (database layer + API)
- **Testing & Validation**: 1-2 days

**Total**: ~1 week for complete implementation

## Conclusion

Separating ports into their own table is a significant improvement that provides:
- **Immediate performance benefits** for port queries
- **Cleaner, more maintainable code**  
- **Better foundation** for future port features
- **Industry standard database design**

This change aligns with the project's goal of building a robust, performant port information system and should be implemented as part of Phase 2 (ProxyAPI Implementation) in the port-port project.