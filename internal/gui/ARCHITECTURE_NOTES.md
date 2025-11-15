# GUI Architecture Notes

## Component System Summary

We've successfully created a component-based architecture for the Fyne GUI with proper data bindings! 🎉

### What Was Built

#### ✅ **OrchestrationCard Component**
Located in `components/`:
- [orchestration_card.go](components/orchestration_card.go) - UI rendering
- [orchestration_card_data.go](components/orchestration_card_data.go) - Data bindings

**Works with:** `bot.BotGroup` (from `bot.Orchestrator`)

**Features:**
- Reactive data bindings (auto-updating UI)
- Thread-safe updates using exported methods
- Computed bindings (pool progress)
- Callback pattern for actions
- Dynamic UI updates (status indicator colors)

#### ✅ **OrchestrationTab Example**
Located in `tabs/orchestration.go`:
- Shows how to use the OrchestrationCard component
- Demonstrates card lifecycle management
- Implements periodic refresh
- Shows callback implementations

#### ✅ **Comprehensive Documentation**
- [GUI_STRUCTURE.md](GUI_STRUCTURE.md) - Architecture overview
- [components/README.md](components/README.md) - Component development guide
- [COMPONENT_EXAMPLE.md](COMPONENT_EXAMPLE.md) - Complete tutorial
- [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md) - Migration plan
- [README.md](README.md) - Main package documentation

## Architecture Decision: Orchestrator vs Manager

### The Right Abstraction Level

**`bot.Orchestrator` with `bot.BotGroup`** ← Use this for UI!
- High-level coordination of multiple bots
- Manages emulator instances
- Handles account pools
- Tracks bot status and progress
- **This is what users monitor and configure**

**`bot.Manager`** ← Behind-the-scenes
- Low-level bot lifecycle management
- Registry injection
- Internal plumbing
- **Not directly exposed in UI**

### Why OrchestrationCard Works Perfectly

```go
// OrchestrationCard expects bot.BotGroup
type OrchestrationCard struct {
    group *bot.BotGroup  // ✅ Perfect!
    // ...
}

// BotGroup has all the data we need:
type BotGroup struct {
    Name               string
    OrchestrationID    string
    RoutineName        string
    AvailableInstances []int
    ActiveBots         map[int]*BotInfo  // With thread-safe access!
    AccountPoolName    string
    AccountPool        accountpool.AccountPool
    // ... etc
}
```

**Key advantages:**
1. ✅ No circular imports (bot package doesn't import gui)
2. ✅ Thread-safe methods already exist (`IsRunning()`, `GetAllBotInfo()`)
3. ✅ Right level of abstraction for users
4. ✅ All necessary data in one place

### Existing Tabs

**ManagerGroupsTab** (manager_groups.go)
- Legacy tab using `bot.Manager` directly
- Uses old-style UI (manual widget management)
- Works fine for its use case
- **Keep as-is** - it's a different abstraction level

**Future:** Could create a `BotGroupCard` variant if you want to show manager-level details, but the orchestrator level is more appropriate for most UI needs.

## Component Pattern Summary

### 1. Create Data Bindings (`*_data.go`)

```go
type MyComponentData struct {
    Title binding.String
    Count binding.Int
    // ... all bindable fields
}

func NewMyComponentData(model *MyModel) *MyComponentData {
    data := &MyComponentData{
        Title: binding.NewString(),
        Count: binding.NewInt(),
    }
    data.UpdateFromModel(model)
    data.setupComputedBindings()
    return data
}

func (d *MyComponentData) UpdateFromModel(model *MyModel) {
    // Thread-safe access to model
    d.Title.Set(model.GetTitle())
    d.Count.Set(model.GetCount())
}
```

### 2. Create UI Component (`*.go`)

```go
type MyComponent struct {
    data   *MyComponentData
    model  *MyModel
    container *fyne.Container

    // Callbacks
    onAction func(*MyModel)
}

func NewMyComponent(model *MyModel, callbacks MyComponentCallbacks) *MyComponent {
    c := &MyComponent{
        model: model,
        data:  NewMyComponentData(model),
        onAction: callbacks.OnAction,
    }
    c.container = c.build()
    c.setupListeners()
    return c
}

func (c *MyComponent) build() *fyne.Container {
    // Use data bindings for auto-updating UI
    titleLabel := widget.NewLabelWithData(c.data.Title)
    countLabel := widget.NewLabelWithData(
        binding.IntToStringWithFormat(c.data.Count, "Count: %d"),
    )

    button := widget.NewButton("Action", func() {
        if c.onAction != nil {
            c.onAction(c.model)
        }
    })

    return container.NewVBox(titleLabel, countLabel, button)
}

func (c *MyComponent) setupListeners() {
    // Dynamic UI updates
    c.data.IsActive.AddListener(binding.NewDataListener(func() {
        // React to changes
    }))
}

func (c *MyComponent) UpdateFromModel() {
    c.data.UpdateFromModel(c.model)
}
```

### 3. Use in Tab

```go
// Create component
callbacks := MyComponentCallbacks{
    OnAction: t.handleAction,
}
card := NewMyComponent(model, callbacks)

// Add to UI
container.Add(card.GetContainer())

// Periodic updates
ticker := time.NewTicker(1 * time.Second)
go func() {
    for range ticker.C {
        card.UpdateFromModel()
    }
}()
```

## Key Patterns

### ✅ Data Bindings
```go
// Automatic UI updates
nameBinding := binding.NewString()
label := widget.NewLabelWithData(nameBinding)
nameBinding.Set("New Name") // UI updates automatically!
```

### ✅ Computed Bindings
```go
// Derived values that auto-update
progressBinding := NewComputed(
    func() string {
        cur, _ := current.Get()
        tot, _ := total.Get()
        return fmt.Sprintf("%d/%d", cur, tot)
    },
    current, total,
)
```

### ✅ Thread Safety
```go
// Always use exported methods
isRunning := group.IsRunning()  // ✅ Safe
activeBots := group.GetAllBotInfo()  // ✅ Safe (returns copy)

// Never access unexported fields directly
// isRunning := group.running  // ❌ Race condition!
```

### ✅ Callback Pattern
```go
// Keep business logic out of components
type ComponentCallbacks struct {
    OnAction func(*Model)
}

// Components just trigger callbacks
button := widget.NewButton("Action", func() {
    if c.onAction != nil {
        c.onAction(c.model)  // Delegate to tab/controller
    }
})
```

## Benefits Achieved

1. ✅ **Reusable Components**: OrchestrationCard can be used anywhere
2. ✅ **Automatic UI Updates**: Data bindings handle reactivity
3. ✅ **Thread Safety**: Proper locking and exported methods
4. ✅ **Clean Separation**: Data vs UI vs Business Logic
5. ✅ **Testable**: Components can be tested independently
6. ✅ **Maintainable**: Clear structure and patterns
7. ✅ **Documented**: Extensive guides and examples

## Next Steps

### To Use Orchestration in Your App

1. **Create an Orchestrator** (if you haven't):
   ```go
   orchestrator := bot.NewOrchestrator(
       config,
       templateRegistry,
       routineRegistry,
       emulatorManager,
       poolManager,
   )
   ```

2. **Create a Tab**:
   ```go
   orchestrationTab := tabs.NewOrchestrationTab(orchestrator, window)
   ```

3. **Add to Controller**:
   ```go
   ctrl.orchestrationTab = orchestrationTab
   tabs.Append(container.NewTabItem("Orchestration", orchestrationTab.Build()))
   ```

4. **Create Groups via UI**:
   - Click "Create New Group"
   - Configure routine, instances, pools
   - Cards automatically update with real-time data

### To Create More Components

Follow the patterns in:
- `components/orchestration_card*.go` - Complete example
- `components/README.md` - Development guide
- `COMPONENT_EXAMPLE.md` - Detailed tutorial

Common components to create:
- RoutineCard (already exists, could enhance with bindings)
- AccountCard
- EmulatorInstanceCard
- PoolStatsCard
- LogEntryCard

## Troubleshooting

### Circular Import
**Problem:** `package imports itself`

**Solution:** Never import `gui` in `components`. Components should only import:
- `bot` package (for models like `BotGroup`)
- `fyne` packages
- Standard library

### Race Conditions
**Problem:** Crashes or inconsistent data

**Solution:** Always use exported methods:
```go
// ✅ Good
isRunning := group.IsRunning()

// ❌ Bad
isRunning := group.running
```

### UI Not Updating
**Problem:** Data changes but UI doesn't reflect it

**Solutions:**
1. Check that you're calling `UpdateFromModel()` periodically
2. Ensure bindings are set up correctly
3. For manual updates, call `Refresh()`:
   ```go
   c.statusIndicator.Refresh()
   ```

## File Organization

```
internal/gui/
├── components/              # Reusable components
│   ├── orchestration_card.go
│   ├── orchestration_card_data.go
│   └── README.md
│
├── tabs/                   # Tab implementations
│   └── orchestration.go
│
├── controller.go           # Main controller
├── manager_groups.go       # Legacy manager tab (keep as-is)
├── eventbus.go            # Event system
└── theme.go               # Theming

Documentation:
├── GUI_STRUCTURE.md       # Architecture overview
├── COMPONENT_EXAMPLE.md   # Complete tutorial
├── MIGRATION_GUIDE.md     # Migration plan
├── ARCHITECTURE_NOTES.md  # This file
└── README.md             # Main documentation
```

## Summary

We've built a solid foundation for component-based Fyne UI development:

✅ **OrchestrationCard** component ready to use
✅ **Clear patterns** documented and demonstrated
✅ **Right abstraction** (Orchestrator/BotGroup, not Manager)
✅ **No circular imports** (clean architecture)
✅ **Thread-safe** by design
✅ **Extensively documented**

The Orchestration tab with BotGroup cards is the right UI abstraction for managing your Android game automation bots! 🎮🤖
