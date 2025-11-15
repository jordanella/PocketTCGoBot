# GUI Structure and Organization

This document describes the organization and architecture of the GUI code.

## Directory Structure

```
internal/gui/
├── components/              # Reusable UI components
│   ├── README.md           # Component documentation
│   ├── orchestration_card.go
│   ├── orchestration_card_data.go
│   └── routine_card.go     # (to be moved)
│
├── tabs/                   # Main tab views
│   ├── orchestration.go    # Orchestration groups tab
│   ├── dashboard.go        # (to be moved)
│   ├── manager_groups.go   # (to be moved)
│   ├── bot_launcher.go     # (to be moved)
│   ├── routines.go         # (to be moved)
│   ├── accounts.go         # (to be moved)
│   ├── account_pools.go    # (to be moved)
│   ├── config.go           # (to be moved)
│   ├── controls.go         # (to be moved)
│   ├── logs.go             # (to be moved)
│   ├── results.go          # (to be moved)
│   └── adbtest.go          # (to be moved)
│
├── database/               # Database-related tabs
│   ├── accounts.go         # (to be moved from database_accounts.go)
│   ├── activity.go         # (to be moved from database_activity.go)
│   ├── errors.go           # (to be moved from database_errors.go)
│   ├── packs.go            # (to be moved from database_packs.go)
│   └── collection.go       # (to be moved from database_collection.go)
│
├── controller.go           # Main GUI controller
├── eventbus.go            # Event bus for thread-safe updates
└── theme.go               # Theme definitions
```

## Architecture

### Controller (MVC Pattern)

The `Controller` serves as the central coordinator:

- **Model**: Business logic (bot.Manager, bot.Orchestrator, registries)
- **View**: Fyne UI components and tabs
- **Controller**: Coordinates between model and view, handles events

### Component-Based Design

Components are self-contained, reusable UI elements:

1. **Components** (`components/`):
   - Reusable across multiple tabs
   - Encapsulate UI logic and state
   - Use Fyne data bindings for reactive updates
   - Accept callbacks for user actions

2. **Tabs** (`tabs/`):
   - Full-screen views in the main window
   - Compose multiple components
   - Handle tab-specific logic
   - Coordinate with controller for business logic

3. **Database Tabs** (`database/`):
   - Specialized tabs for database views
   - Query and display database records
   - Handle data filtering and pagination

## Data Flow

```
User Action → Component Callback → Tab Handler → Controller → Business Logic (Model)
                                                      ↓
                                         Update Model State
                                                      ↓
                                         Periodic Refresh
                                                      ↓
Component.UpdateFromModel() → Update Bindings → Fyne Auto-Updates UI
```

## Creating New Components

### 1. Component Files

Create two files in `components/`:

- `component_name.go` - UI and rendering
- `component_name_data.go` - Data bindings and state

### 2. Use Data Bindings

```go
// In *_data.go
type MyComponentData struct {
    Title binding.String
    Count binding.Int
}

// In component.go
titleLabel := widget.NewLabelWithData(c.data.Title)
```

### 3. Accept Callbacks

```go
type MyComponentCallbacks struct {
    OnAction func(*Model)
}

func NewMyComponent(model *Model, callbacks MyComponentCallbacks) *MyComponent {
    // ...
}
```

### 4. Implement Update Method

```go
func (c *MyComponent) UpdateFromModel() {
    c.data.UpdateFromModel(c.model)
}
```

See [components/README.md](components/README.md) for detailed guidance.

## Creating New Tabs

### 1. Tab File

Create file in `tabs/`:

```go
package tabs

type MyTab struct {
    controller *Controller
    // ... components and state
}

func NewMyTab(ctrl *Controller) *MyTab {
    return &MyTab{controller: ctrl}
}

func (t *MyTab) Build() fyne.CanvasObject {
    // Build UI
    return container.NewVBox(...)
}
```

### 2. Register in Controller

In `controller.go`:

```go
// Add field
myTab *tabs.MyTab

// Initialize in NewController
ctrl.myTab = tabs.NewMyTab(ctrl)

// Add to tabs
tabs.Append(container.NewTabItem("My Tab", ctrl.myTab.Build()))
```

## Event Bus

Use the event bus for thread-safe UI updates from background goroutines:

```go
// From any goroutine
ctrl.eventBus.Dispatch(func() {
    // This runs on the main UI thread
    label.SetText("Updated!")
})
```

## Thread Safety

### Reading Shared State

Always lock when reading shared data:

```go
group.activeBotsMu.RLock()
count := len(group.ActiveBots)
group.activeBotsMu.RUnlock()
```

### Updating UI from Goroutines

Use the event bus or Fyne's main thread dispatch:

```go
// Option 1: Event bus
ctrl.eventBus.Dispatch(func() {
    widget.Refresh()
})

// Option 2: Direct (if you have app reference)
app.QueueEvent(func() {
    widget.Refresh()
})
```

## Migration Plan

To migrate existing code to the new structure:

### Phase 1: Components (Done)
- ✅ Created `components/` directory
- ✅ Created `OrchestrationCard` component
- ✅ Documented component patterns

### Phase 2: New Tabs (In Progress)
- ✅ Created `tabs/` directory
- ✅ Created `OrchestrationTab`
- ⏳ Move existing tabs to `tabs/`

### Phase 3: Database Tabs
- ⏳ Created `database/` directory
- ⏳ Move database tabs to `database/`
- ⏳ Update imports

### Phase 4: Cleanup
- ⏳ Remove old files from root
- ⏳ Update all imports
- ⏳ Test all functionality

## Benefits of New Structure

1. **Better Organization**: Clear separation between components, tabs, and utilities
2. **Reusability**: Components can be used in multiple places
3. **Maintainability**: Easier to find and modify code
4. **Testability**: Components can be tested independently
5. **Scalability**: Easy to add new features without cluttering

## Examples

### Using OrchestrationCard Component

See [tabs/orchestration.go](tabs/orchestration.go) for a complete example of:
- Creating cards from data models
- Setting up callbacks
- Periodic updates
- Managing multiple cards

### Complete Component Pattern

See [components/orchestration_card.go](components/orchestration_card.go) and [components/orchestration_card_data.go](components/orchestration_card_data.go) for:
- Data binding setup
- Computed bindings
- Thread-safe updates
- UI construction
- Callback handling

## Future Improvements

1. **More Components**: Extract common UI patterns into components
2. **Component Library**: Build a library of reusable widgets
3. **Testing**: Add unit tests for components
4. **Documentation**: Add more examples and tutorials
5. **Themes**: Support for custom themes and styling
