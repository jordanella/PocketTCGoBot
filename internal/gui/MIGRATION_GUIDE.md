# GUI Restructuring Migration Guide

This guide outlines the plan to migrate the existing GUI code to the new structure.

## Current Structure (Before)

```
internal/gui/
├── *.go (all files in root)
```

## Target Structure (After)

```
internal/gui/
├── components/         # Reusable components
├── tabs/              # Main tab views
├── database/          # Database tabs
├── controller.go
├── eventbus.go
└── theme.go
```

## Migration Steps

### ✅ Phase 1: Create New Structure (COMPLETED)

- ✅ Created `components/` directory
- ✅ Created `tabs/` directory
- ✅ Created `database/` directory
- ✅ Created `OrchestrationCard` component
- ✅ Created `OrchestrationTab` example
- ✅ Documentation written

### ⏳ Phase 2: Move Components

Move component files to `components/`:

```bash
# Move existing component
git mv internal/gui/routine_card.go internal/gui/components/

# Update package declaration in the file
# Change: package gui
# To:     package components
```

Files to move:
- `routine_card.go` → `components/routine_card.go`

### ⏳ Phase 3: Move Tabs

Move tab files to `tabs/`:

```bash
# Example for each tab file
git mv internal/gui/dashboard.go internal/gui/tabs/
git mv internal/gui/manager_groups.go internal/gui/tabs/
git mv internal/gui/bot_launcher.go internal/gui/tabs/
# ... etc
```

Files to move to `tabs/`:
- `dashboard.go`
- `manager_groups.go`
- `bot_launcher.go`
- `routines.go`
- `routines_enhanced.go`
- `accounts.go`
- `account_pools.go`
- `unified_pool_wizard.go`
- `config.go`
- `config_adapter.go`
- `controls.go`
- `logs.go`
- `results.go`
- `adbtest.go`

**Important:** After moving each file:
1. Update package declaration from `package gui` to `package tabs`
2. Add import for `gui` package if needed (for types like `Controller`)
3. Update any struct field types that reference GUI types

Example changes needed:

```go
// Before (in gui/dashboard.go)
package gui

type DashboardTab struct {
    controller *Controller
}

// After (in gui/tabs/dashboard.go)
package tabs

import "jordanella.com/pocket-tcg-go/internal/gui"

type DashboardTab struct {
    controller *gui.Controller
}
```

### ⏳ Phase 4: Move Database Tabs

Move database-related files to `database/`:

```bash
git mv internal/gui/database_accounts.go internal/gui/database/accounts.go
git mv internal/gui/database_activity.go internal/gui/database/activity.go
git mv internal/gui/database_errors.go internal/gui/database/errors.go
git mv internal/gui/database_packs.go internal/gui/database/packs.go
git mv internal/gui/database_collection.go internal/gui/database/collection.go
```

**Important:**
1. Update package to `package database`
2. Remove `database_` prefix from filenames
3. Update imports in files

### ⏳ Phase 5: Update Controller

Update `controller.go` to import from new packages:

```go
package gui

import (
    // ... other imports
    "jordanella.com/pocket-tcg-go/internal/gui/components"
    "jordanella.com/pocket-tcg-go/internal/gui/tabs"
    "jordanella.com/pocket-tcg-go/internal/gui/database"
)

type Controller struct {
    // ... fields

    // Tabs (updated types)
    dashboard        *tabs.DashboardTab
    configTab        *tabs.ConfigTab
    logTab           *tabs.LogTab
    // ... etc

    // Database tabs
    dbAccountsTab   *database.AccountsTab
    dbActivityTab   *database.ActivityTab
    // ... etc
}

// Update initialization
func NewController(...) *Controller {
    // ...
    ctrl.dashboard = tabs.NewDashboardTab(ctrl)
    ctrl.dbAccountsTab = database.NewAccountsTab(ctrl)
    // ... etc
}
```

### ⏳ Phase 6: Update Imports Across Codebase

Search and update imports in other packages:

```bash
# Find files that import the gui package
grep -r "jordanella.com/pocket-tcg-go/internal/gui" --include="*.go"

# Update as needed to point to new subpackages
```

Example:
```go
// Before
import "jordanella.com/pocket-tcg-go/internal/gui"

// After (if using components)
import "jordanella.com/pocket-tcg-go/internal/gui/components"
```

### ⏳ Phase 7: Testing

After migration, test all functionality:

1. ✅ **Build test**: `go build ./...`
2. ✅ **Import test**: Verify no circular imports
3. ✅ **Functionality test**: Run the application
4. ✅ **Tab navigation**: Verify all tabs load
5. ✅ **Component rendering**: Verify all components display
6. ✅ **Actions**: Test button clicks, dialogs, etc.

## Detailed Migration Example: Dashboard Tab

### Step 1: Move the file

```bash
git mv internal/gui/dashboard.go internal/gui/tabs/dashboard.go
```

### Step 2: Update package declaration

```go
// Before
package gui

// After
package tabs
```

### Step 3: Update imports and types

```go
// Before
package gui

type DashboardTab struct {
    controller *Controller
}

func NewDashboardTab(ctrl *Controller) *DashboardTab {
    // ...
}

// After
package tabs

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "jordanella.com/pocket-tcg-go/internal/gui"
)

type DashboardTab struct {
    controller *gui.Controller  // Add gui. prefix
}

func NewDashboardTab(ctrl *gui.Controller) *DashboardTab {  // Add gui. prefix
    // ...
}
```

### Step 4: Update controller.go

```go
import (
    "jordanella.com/pocket-tcg-go/internal/gui/tabs"
)

type Controller struct {
    dashboard *tabs.DashboardTab  // Update type
}

func NewController(...) *Controller {
    ctrl.dashboard = tabs.NewDashboardTab(ctrl)  // Update constructor
}
```

### Step 5: Build and test

```bash
go build ./internal/gui/...
```

## Migration Checklist

Use this checklist when migrating each file:

- [ ] File moved to correct directory
- [ ] Package declaration updated
- [ ] Import statements added (if needed)
- [ ] Type references updated (added `gui.` prefix where needed)
- [ ] Controller.go updated with new import and type
- [ ] Build succeeds: `go build ./internal/gui/...`
- [ ] File added to git: `git add <file>`

## Testing Strategy

### Unit Tests

After migration, verify:

```go
// Test that components can be created
func TestComponentCreation(t *testing.T) {
    group := &bot.BotGroup{Name: "Test"}
    card := components.NewOrchestrationCard(group, components.OrchestrationCardCallbacks{})
    assert.NotNil(t, card)
}

// Test that tabs can be created
func TestTabCreation(t *testing.T) {
    ctrl := &gui.Controller{}
    tab := tabs.NewDashboardTab(ctrl)
    assert.NotNil(t, tab)
}
```

### Integration Tests

Run the full application and verify:

1. All tabs load without errors
2. Components render correctly
3. Buttons and interactions work
4. No runtime panics
5. No console errors

## Rollback Plan

If migration causes issues:

```bash
# Rollback using git
git reset --hard HEAD

# Or rollback specific files
git checkout HEAD -- internal/gui/dashboard.go
```

## Benefits After Migration

1. **Clearer Organization**: Easy to find files
2. **Better IDE Support**: Auto-complete and navigation
3. **Reusable Components**: Share UI patterns
4. **Easier Testing**: Test components in isolation
5. **Faster Onboarding**: New developers understand structure
6. **Scalability**: Easy to add new features

## Current Status

✅ **Completed:**
- Directory structure created
- Example component (`OrchestrationCard`) created
- Example tab (`OrchestrationTab`) created
- Documentation written
- Build verified

⏳ **Remaining:**
- Move existing components to `components/`
- Move existing tabs to `tabs/`
- Move database tabs to `database/`
- Update controller imports
- Update all imports across codebase
- Full testing

## Notes

- **Gradual Migration**: Can be done file-by-file
- **No Breaking Changes**: External API remains the same
- **Git Friendly**: Using `git mv` preserves history
- **Backward Compatible**: Old imports can coexist during migration
