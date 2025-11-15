# Account Pool YAML Persistence Implementation

## Overview

Implemented automatic YAML persistence for account pool definitions, matching the functionality already present for orchestration groups. Pools can now be created, edited, and deleted through the GUI with automatic file management.

## Changes Made

### 1. YAML Persistence Methods ([internal/accountpool/unified_pool.go](internal/accountpool/unified_pool.go))

#### Added Methods

**`SaveToYAML(dirPath string) error`**
- Creates directory if it doesn't exist
- Sanitizes pool name for filename
- Marshals definition to YAML
- Writes to disk
- Returns error on failure

**`DeleteYAML(dirPath string) error`**
- Deletes YAML file for this pool definition
- Ignores "file not found" errors
- Returns error on failure

**Note:** The `sanitizeFilename` function already existed in `pool_manager.go` and is reused.

### 2. Pool Manager Integration ([internal/accountpool/pool_manager.go](internal/accountpool/pool_manager.go))

The PoolManager already had YAML file management built-in:

- **`CreatePool(def *PoolDefinition) error`** - Creates pool and saves to YAML
- **`UpdatePool(name string, def *PoolDefinition) error`** - Updates pool and saves to YAML
- **`DeletePool(name string) error`** - Deletes pool and removes YAML file
- **`DiscoverPools() error`** - Auto-loads all pools from YAML files on startup

### 3. Pool Creation Wizard Integration ([internal/gui/tabs/](internal/gui/tabs/))

#### Moved Wizard to Tabs Package
- Moved `unified_pool_wizard.go` from `internal/gui/` to `internal/gui/tabs/`
- Changed package from `gui` to `tabs` to avoid import cycles
- Existing wizard already had full 5-step creation flow

#### Implemented Pool Creation ([account_pools.go](internal/gui/tabs/account_pools.go))

**`handleNewPool()` implementation:**
1. Shows dialog to enter pool name
2. Validates pool name (non-empty, doesn't exist)
3. Launches `UnifiedPoolWizard` with 5-step process:
   - Step 1: Basic Info (name, description)
   - Step 2: SQL Queries (visual query builder)
   - Step 3: Manual Inclusions
   - Step 4: Manual Exclusions
   - Step 5: Configuration (sort method, retry settings, watched paths)
4. On completion:
   - Creates `PoolDefinition` from wizard output
   - Calls `poolManager.CreatePool()` which auto-saves to YAML
   - Refreshes pool list
   - Selects newly created pool
   - Shows success message

### 4. Pool Deletion ([account_pools.go](internal/gui/tabs/account_pools.go))

**`handleDeletePool()` implementation:**
1. Shows confirmation dialog with warning about file deletion
2. On confirmation:
   - Calls `poolManager.DeletePool()` which removes YAML file
   - Clears selection
   - Refreshes pool list
   - Resets details panel
   - Clears accounts table
   - Shows success message

### 5. Cleanup

**Removed deprecated file:**
- `internal/gui/account_pools.go` - Old account pools implementation that was replaced by the new tab-based version

## File Structure

```
bin/
├── pools/                       # Auto-discovered pool directory
│   ├── Test_Pool.yaml          # Example pool definition
│   └── Premium_Farmers.yaml    # Example pool definition
└── bot-gui.exe
```

## YAML Format

Pools are saved in structured YAML format:

```yaml
pool_name: "Pool Name"
description: "Optional description"
queries:
  - name: "Query Name"
    filters:
      - column: "pack_count"
        comparator: ">="
        value: "10"
    sort:
      - column: "pack_count"
        direction: "desc"
    limit: 100
include:
  - "account001"
  - "account002"
exclude:
  - "banned_account"
watched_paths:
  - "C:/accounts/premium"
config:
  sort_method: "packs_desc"
  retry_failed: true
  max_failures: 3
  refresh_interval: 300
  enabled: true
```

## Workflow Comparison

### Before
1. ❌ No GUI pool creation - had to manually write YAML files
2. ✅ Auto-load on startup
3. ❌ No GUI editing
4. ❌ No GUI deletion - had to manually delete YAML files

### After
1. ✅ **GUI pool creation wizard** - 5-step visual wizard
2. ✅ **Auto-save to YAML** - pools automatically persisted
3. ✅ Auto-load on startup (unchanged)
4. ⏳ GUI editing (TODO - can use wizard for new pools)
5. ✅ **GUI deletion** - removes both pool and YAML file

## Feature Parity with Orchestration Groups

| Feature | Orchestration Groups | Account Pools |
|---------|---------------------|---------------|
| **Auto-load on startup** | ✅ Yes | ✅ Yes |
| **GUI creation wizard** | ✅ Yes | ✅ **Now Yes!** |
| **Auto-save on create** | ✅ Yes | ✅ **Now Yes!** |
| **GUI deletion** | ✅ Yes | ✅ **Now Yes!** |
| **Auto-delete YAML** | ✅ Yes | ✅ **Now Yes!** |
| **YAML location** | `bin/groups/` | `bin/pools/` |
| **GUI editing** | ✅ Yes | ⏳ TODO |

## User Workflow

### Creating a Pool
1. Launch application
2. Go to **Account Pools** tab
3. Click **"New"** button in header
4. Enter pool name in dialog
5. Click **"Create"** - wizard launches
6. **Step 1:** Enter description
7. **Step 2:** Add SQL queries using visual builder (optional)
8. **Step 3:** Add manual account inclusions (optional)
9. **Step 4:** Add manual account exclusions (optional)
10. **Step 5:** Configure sort method, retry settings, watched paths
11. Click **"Create Pool"**
12. Pool appears in list with auto-saved YAML file

### Deleting a Pool
1. Select pool from list
2. Switch to **Details** tab
3. Click **"Delete Pool"** button (red danger button)
4. Confirm deletion
5. Pool removed from list and YAML file deleted

### Using a Pool
1. Create/select a pool in **Account Pools** tab
2. Go to **Orchestration** tab
3. Create new bot group
4. Select the pool from dropdown
5. Configure instances and routine
6. Launch group - accounts are pulled from the pool

## Technical Details

### Import Cycle Resolution
- Initially attempted to import wizard from `gui` package into `tabs` package
- This created an import cycle: `gui` → `tabs` → `gui`
- **Solution:** Moved wizard into `tabs` package to eliminate cycle
- Changed package declaration from `package gui` to `package tabs`

### Duplicate Function Resolution
- Added `sanitizeFilename` function to `unified_pool.go`
- Build error: function already existed in `pool_manager.go`
- **Solution:** Removed duplicate, reused existing function

### File Cleanup
- Removed old `internal/gui/account_pools.go` (deprecated implementation)
- Wizard now lives in `internal/gui/tabs/unified_pool_wizard.go`

## Testing

### Build Status
✅ Compiles successfully
✅ No type errors
✅ No import errors
✅ No duplicate declarations

### Manual Testing Checklist
- [ ] Launch application
- [ ] Create new pool through wizard
- [ ] Verify YAML file created in `bin/pools/`
- [ ] Restart application
- [ ] Verify pool loads from YAML
- [ ] Select pool and view details
- [ ] Delete pool through GUI
- [ ] Verify YAML file removed
- [ ] Test pool in orchestration workflow

## Benefits

1. **User-Friendly**: No need to manually write YAML files
2. **Consistent**: Same persistence model as orchestration groups
3. **Visual**: 5-step wizard with visual query builder
4. **Safe**: Automatic file management prevents orphaned files
5. **Portable**: YAML files can be shared, backed up, version controlled
6. **Reliable**: Validation at every step prevents invalid configurations

## Future Enhancements

### High Priority
1. **Pool Editing**: Implement edit functionality to modify existing pools
   - Reuse wizard but pre-populate with existing values
   - Update YAML file on save
   - Handle pool name changes (delete old file, create new)

### Medium Priority
2. **Import/Export**: Import pools from other sources
3. **Pool Templates**: Predefined configurations for common use cases
4. **Duplicate Pool**: Clone existing pool with modifications

### Low Priority
5. **Pool Validation UI**: Visual feedback for query results before saving
6. **Pool Statistics**: Show pool performance metrics
7. **Pool History**: Track changes to pool configurations

## Dependencies

- `gopkg.in/yaml.v3` - Already in go.mod
- `fyne.io/fyne/v2` - GUI framework

## Documentation

- `POOL_YAML_PERSISTENCE.md` - This file
- `ACCOUNT_POOLS_TAB_REFACTOR.md` - Tab refactor documentation
- `YAML_PERSISTENCE_IMPLEMENTATION.md` - Orchestration group persistence (reference)

## Files Modified/Created

### Modified
- `internal/accountpool/unified_pool.go` - Added YAML persistence methods
- `internal/gui/tabs/account_pools.go` - Implemented create/delete handlers
- `internal/gui/tabs/unified_pool_wizard.go` - Moved from gui package (changed package declaration)

### Deleted
- `internal/gui/account_pools.go` - Old deprecated implementation
- `internal/gui/unified_pool_wizard.go` - Moved to tabs package

### Created
- `POOL_YAML_PERSISTENCE.md` - This documentation
- `bin/pools/test_pool.yaml` - Example pool for testing

## Summary

Account pools now have **full YAML persistence** with **GUI creation and deletion**, achieving feature parity with orchestration groups. Users can create pools through a comprehensive 5-step wizard, and all pool definitions are automatically saved to YAML files that persist across application restarts. The only remaining feature is GUI editing of existing pools, which can be implemented by reusing the wizard with pre-populated values.
