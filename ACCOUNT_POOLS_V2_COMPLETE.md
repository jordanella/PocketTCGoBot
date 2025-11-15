# Account Pools V2 - Complete Implementation

## Overview

Completely redesigned the Account Pools interface to remove the multi-step wizard and implement direct inline editing of all pool properties. This provides a much more intuitive and efficient workflow for managing account pools.

## Key Changes

### 1. Removed Wizard-Based Interface
- **Deleted**: `unified_pool_wizard.go` (5-step wizard)
- **Deleted**: `account_pools.go` (old v1 implementation)
- **Created**: `account_pools_v2.go` (new inline editing implementation)

### 2. New Inline Editing Interface

#### Left Panel: Pool List
- Card-based list of all pools
- Shows pool name, type, account count, last updated
- Click to select a pool
- **New Pool** button creates pools with just a name
- **Refresh** button reloads pool list

#### Right Panel: Tabbed Editor

**Details Tab (Editable)**:
- Description (multi-line text entry)
- Sort Method (dropdown: packs_desc, packs_asc, shinedust_desc, etc.)
- Retry Failed Accounts (checkbox)
- Max Failures (number entry)
- Total Accounts (read-only, with refresh button)
- Save/Discard buttons (enabled only when dirty)
- Delete Pool button

**Accounts Tab (Read-Only)**:
- Table showing accounts in the pool
- Columns: Account ID, Packs, Shinedust, Status
- Populated from pool test results

**Queries Tab (Visual Builder)**:
- List of all queries in the pool
- "+ Add Query" button opens visual builder
- Each query has Edit and Delete buttons
- **Visual Query Builder Dialog**:
  - Query name
  - Add/remove filter conditions (column, comparator, value)
  - Add/remove sort orders (column, direction)
  - Result limit
  - Dynamic UI updates as you add/remove filters

**Include Tab (Manual + Instance Dropdown)**:
- List of manually included accounts
- **Add Manual**: Text entry for account ID
- **From Instance**: Dropdown populated from detected emulator instances
- **Refresh Instances** button to re-detect
- Remove button for each account

**Exclude Tab (Manual + Instance Dropdown)**:
- Same as Include tab but for exclusions

### 3. Smart Change Detection

- Tracks whether pool has unsaved changes (`isDirty` flag)
- Save/Discard buttons disabled until changes made
- Confirmation dialog when switching pools with unsaved changes
- All changes persist immediately to YAML when saved

### 4. Instance Integration

- Emulator manager passed to tab constructor
- Dropdowns populated via `emulatorMgr.DiscoverInstances()`
- Shows "Instance X (Window Title)" for easy identification
- Refresh button to re-detect instances without closing dialog

### 5. Visual Query Builder

**Features**:
- **Filters Section**:
  - Add/remove filter rows dynamically
  - Column dropdown (packs_opened, shinedust, hourglasses, etc.)
  - Comparator dropdown (=, !=, >, >=, <, <=, LIKE)
  - Value text entry
  - All filters combined with AND logic

- **Sorting Section**:
  - Add/remove sort rows dynamically
  - Column dropdown (same as filters)
  - Direction dropdown (asc, desc)
  - Multiple sorts applied in sequence

- **Limit**:
  - Optional result limit
  - 0 = no limit

- **Edit Mode**:
  - Pre-populates with existing query data
  - Same dialog for add and edit

**Database Columns Available**:
- `packs_opened` - Total packs opened
- `shinedust` - Shinedust balance
- `hourglasses` - Hourglass balance
- `pokegold` - PokeGold balance
- `pack_points` - Pack point balance
- `wonder_picks_done` - Wonder picks completed
- `account_level` - Account level
- `last_used_at` - Last time account was used
- `is_active` - Active status
- `is_banned` - Banned status

## Workflow Comparison

### Old Workflow (Wizard-Based)
1. Click "New Pool"
2. Enter pool name in dialog
3. **Step 1/5**: Enter description
4. **Step 2/5**: Configure queries (confusing SQL-like interface)
5. **Step 3/5**: Add manual inclusions
6. **Step 4/5**: Add manual exclusions
7. **Step 5/5**: Configure settings
8. Click "Create Pool"
9. Pool created and wizard closes
10. To edit: No option, had to delete and recreate

### New Workflow (Inline Editing)
1. Click "New Pool"
2. Enter pool name
3. Pool created immediately with defaults
4. Edit any property directly in tabs:
   - Description in Details tab
   - Queries via visual builder
   - Includes/Excludes with instance dropdowns
   - Settings in Details tab
5. Click "Save Changes" when done
6. Changes persist to YAML immediately

## Benefits

### User Experience
- **Simpler**: No multi-step wizard to navigate
- **Faster**: Create pool with just a name, edit later
- **Clearer**: See all properties at once in tabs
- **Intuitive**: Direct manipulation vs. guided flow

### Developer Experience
- **Less Code**: Removed 800+ lines of wizard code
- **Maintainable**: Single file vs. wizard + integration
- **Extensible**: Easy to add new tabs or fields

### Functionality
- **Instance Integration**: Dropdowns from real detections
- **Visual Query Builder**: No SQL knowledge required
- **Change Detection**: Clear save/discard workflow
- **Edit Support**: Can now edit existing pools

## Technical Implementation

### File Structure
```
internal/gui/tabs/
├── account_pools_v2.go      (new - 1280 lines)
├── account_pools.go          (deleted - old v1)
└── unified_pool_wizard.go    (deleted - 800+ lines)
```

### Key Components

**AccountPoolsTabV2 Struct**:
```go
type AccountPoolsTabV2 struct {
    // Dependencies
    poolManager    *accountpool.PoolManager
    db             *sql.DB
    emulatorMgr    *emulator.Manager  // NEW: for instance dropdowns
    window         fyne.Window

    // State
    currentPool    *accountpool.UnifiedPoolDefinition  // In-memory editing
    isDirty        bool                                // Change detection

    // UI Elements (all editable)
    descEntry        *widget.Entry
    sortMethodSelect *widget.Select
    retryFailedCheck *widget.Check
    maxFailuresEntry *widget.Entry

    // Buttons
    saveBtn     *widget.Button
    discardBtn  *widget.Button

    // Data lists
    queriesData  []accountpool.QuerySource
    includesData []string
    excludesData []string
    accountsData [][]string
}
```

**Query Builder Dialog**:
```go
func (t *AccountPoolsTabV2) showQueryBuilder(
    existingQuery *accountpool.QuerySource,
    onSave func(accountpool.QuerySource),
)
```
- 200+ lines of visual builder UI
- Dynamic filter/sort row management
- Validation and error handling

### Thread Safety

All UI operations wrapped in `fyne.Do()`:
```go
fyne.Do(func() {
    t.queriesList.Refresh()
})
```

All data access protected with mutexes:
```go
t.queriesDataMu.Lock()
defer t.queriesDataMu.Unlock()
```

### YAML Persistence

Changes saved immediately on "Save Changes":
```go
t.poolManager.UpdatePool(poolName, poolDef)
// → Calls savePoolDefinition()
//   → Calls os.WriteFile()
```

## Testing Checklist

- [x] Build compiles without errors
- [ ] Manual GUI testing:
  - [ ] Create new pool
  - [ ] Edit pool description
  - [ ] Add query via visual builder
  - [ ] Edit existing query
  - [ ] Delete query
  - [ ] Add include from instance dropdown
  - [ ] Add exclude manually
  - [ ] Save changes
  - [ ] Discard changes
  - [ ] Rename pool
  - [ ] Delete pool
  - [ ] Pool persists after app restart

## Future Enhancements

### High Priority
- [ ] Query preview/test - show SQL and sample results before saving
- [ ] Duplicate pool functionality
- [ ] Import/export pools

### Medium Priority
- [ ] Query templates (common filters like "premium accounts", "fresh accounts")
- [ ] Bulk operations (add multiple accounts from file)
- [ ] Pool validation warnings

### Low Priority
- [ ] Drag-and-drop query ordering
- [ ] Query statistics (accounts matched per query)
- [ ] Pool usage history

## Migration Notes

### For Users
- Old pools continue to work (YAML format unchanged)
- Wizard removed - all editing now inline
- New features: edit pools, instance dropdowns, visual query builder

### For Developers
- Import changed: `tabs.NewAccountPoolsTabV2()` requires `emulatorMgr`
- Controller field updated: `accountPoolsTab *tabs.AccountPoolsTabV2`
- No API changes to PoolManager or accountpool package

## Summary

Account Pools V2 represents a complete UX overhaul that makes pool management significantly more intuitive and efficient. By removing the wizard and implementing inline editing with a visual query builder, users can now create and modify pools much faster while having better visibility into all pool properties.

The integration with emulator instance detection and the new visual query builder eliminate the need for users to understand internal implementation details, making the feature accessible to a wider range of users.
