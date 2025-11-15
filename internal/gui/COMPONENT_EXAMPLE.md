# Orchestration Card Component - Complete Example

This document provides a complete walkthrough of the `OrchestrationCard` component, demonstrating all the concepts needed to create reusable Fyne components with proper data bindings.

## Component Overview

The `OrchestrationCard` displays information about an orchestration group:
- Group name and status
- Running routine
- Pool progress (remaining/total accounts)
- Active and available instances
- Action buttons (Add Instance, Pause/Resume, Stop, Shutdown)

## File Structure

```
internal/gui/components/
├── orchestration_card.go       # UI rendering and event handling
└── orchestration_card_data.go  # Data bindings and state management
```

## Part 1: Data Layer (`orchestration_card_data.go`)

### Data Structure

```go
type OrchestrationCardData struct {
    // Basic info bindings
    GroupName   binding.String
    Description binding.String
    StartedAt   binding.String

    // Status bindings
    IsActive   binding.Bool
    StatusText binding.String

    // Pool progress bindings
    PoolRemaining binding.Int
    PoolTotal     binding.Int
    PoolProgress  binding.String // Computed: "X/Y"

    // List bindings
    AccountPoolNames    binding.String
    ActiveInstancesList binding.String
    OtherInstancesList  binding.String

    // Metadata
    lastUpdate time.Time
}
```

### Key Concepts

#### 1. Creating Bindings

```go
data := &OrchestrationCardData{
    GroupName:     binding.NewString(),
    IsActive:      binding.NewBool(),
    PoolRemaining: binding.NewInt(),
    // ... etc
}
```

Each binding type corresponds to the data type you want to display:
- `binding.NewString()` for text
- `binding.NewInt()` for numbers
- `binding.NewBool()` for true/false
- `binding.NewStringList()` for lists

#### 2. Computed Bindings

The `PoolProgress` binding is computed from `PoolRemaining` and `PoolTotal`:

```go
func (d *OrchestrationCardData) setupComputedBindings() {
    updatePoolProgress := func() {
        remaining, _ := d.PoolRemaining.Get()
        total, _ := d.PoolTotal.Get()
        d.PoolProgress.Set(fmt.Sprintf("%d/%d", remaining, total))
    }

    // Listen to both dependencies
    d.PoolRemaining.AddListener(binding.NewDataListener(updatePoolProgress))
    d.PoolTotal.AddListener(binding.NewDataListener(updatePoolProgress))

    // Initial calculation
    updatePoolProgress()
}
```

**Why this works:**
- When `PoolRemaining` changes, `updatePoolProgress()` runs
- When `PoolTotal` changes, `updatePoolProgress()` runs
- Both update the same `PoolProgress` binding
- UI automatically reflects the change

#### 3. Thread-Safe Updates

The `UpdateFromGroup()` method reads from the `BotGroup` safely:

```go
func (d *OrchestrationCardData) UpdateFromGroup(group *bot.BotGroup) {
    // Use exported getter methods (they handle locking internally)
    isRunning := group.IsRunning()  // ✅ Safe
    activeBots := group.GetAllBotInfo()  // ✅ Safe (returns copy)

    // Update bindings (safe to call from any thread)
    d.IsActive.Set(isRunning)
    d.PoolRemaining.Set(stats.Available)
    // ... etc
}
```

**Important:** Never access unexported fields directly!
```go
// ❌ WRONG - Race condition
isRunning := group.running

// ✅ RIGHT - Thread-safe
isRunning := group.IsRunning()
```

## Part 2: UI Layer (`orchestration_card.go`)

### Component Structure

```go
type OrchestrationCard struct {
    // Data and model
    data  *OrchestrationCardData
    group *bot.BotGroup

    // Callbacks for actions
    onAddInstance func(*bot.BotGroup)
    onPauseResume func(*bot.BotGroup)
    onStop        func(*bot.BotGroup)
    onShutdown    func(*bot.BotGroup)

    // UI elements that need dynamic updates
    container       *fyne.Container
    statusIndicator *canvas.Circle
    pauseResumeBtn  *widget.Button
}
```

### Key Concepts

#### 1. Using Data Bindings in Widgets

Instead of:
```go
// ❌ Manual update required
nameLabel := widget.NewLabel(group.Name)
// Later... how do we update it?
nameLabel.SetText(newName) // Must remember to call this!
```

Use data bindings:
```go
// ✅ Automatic updates
nameLabel := widget.NewLabelWithData(c.data.GroupName)
// Later... just update the binding
c.data.GroupName.Set(newName) // UI updates automatically!
```

#### 2. Binding Listeners for Dynamic UI

Some UI changes can't be done through bindings alone. Use listeners:

```go
func (c *OrchestrationCard) setupListeners() {
    c.data.IsActive.AddListener(binding.NewDataListener(func() {
        active, _ := c.data.IsActive.Get()
        if active {
            // Change status indicator to green
            c.statusIndicator.FillColor = color.RGBA{76, 175, 80, 255}
            c.pauseResumeBtn.SetText("Pause")
        } else {
            // Change status indicator to gray
            c.statusIndicator.FillColor = color.RGBA{150, 150, 150, 255}
            c.pauseResumeBtn.SetText("Resume")
        }
        c.statusIndicator.Refresh()
    }))
}
```

**When data changes:**
1. `c.data.IsActive` binding changes
2. Listener function runs
3. UI elements update (color, button text)
4. Call `Refresh()` to redraw

#### 3. Callback Pattern

The component doesn't handle business logic—it delegates to callbacks:

```go
type OrchestrationCardCallbacks struct {
    OnAddInstance func(*bot.BotGroup)
    OnPauseResume func(*bot.BotGroup)
    OnStop        func(*bot.BotGroup)
    OnShutdown    func(*bot.BotGroup)
}

func NewOrchestrationCard(group *bot.BotGroup, callbacks OrchestrationCardCallbacks) *OrchestrationCard {
    card := &OrchestrationCard{
        group:         group,
        onAddInstance: callbacks.OnAddInstance,
        onPauseResume: callbacks.OnPauseResume,
        // ... etc
    }
    // ...
}

// In the button handler:
pauseBtn := widget.NewButton("Pause", func() {
    if c.onPauseResume != nil {
        c.onPauseResume(c.group) // Pass the group to callback
    }
})
```

**Benefits:**
- Component is reusable in different contexts
- Business logic stays in the tab/controller
- Easy to test (mock callbacks)

## Part 3: Using the Component (`tabs/orchestration.go`)

### Creating Cards

```go
// Define what happens when buttons are clicked
callbacks := components.OrchestrationCardCallbacks{
    OnAddInstance: t.handleAddInstance,
    OnPauseResume: t.handlePauseResume,
    OnStop:        t.handleStop,
    OnShutdown:    t.handleShutdown,
}

// Create the card
card := components.NewOrchestrationCard(group, callbacks)

// Add to UI
t.cardsContainer.Add(card.GetContainer())
```

### Implementing Callbacks

```go
func (t *OrchestrationTab) handlePauseResume(group *bot.BotGroup) {
    running := group.IsRunning()

    if running {
        // Pause logic
        dialog.ShowConfirm("Pause?", "Pause this group?", func(ok bool) {
            if ok {
                // Call orchestrator to pause
                t.orchestrator.PauseGroup(group.Name)
            }
        }, t.window)
    } else {
        // Resume logic
        t.orchestrator.ResumeGroup(group.Name)
    }
}
```

### Periodic Updates

Keep cards synchronized with data:

```go
func (t *OrchestrationTab) startPeriodicRefresh() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            t.refreshAllCards()
        case <-t.stopRefresh:
            return
        }
    }
}

func (t *OrchestrationTab) refreshAllCards() {
    t.cardsMu.RLock()
    defer t.cardsMu.RUnlock()

    for _, card := range t.cards {
        card.UpdateFromGroup()
    }
}
```

**Flow:**
1. Timer ticks every second
2. `refreshAllCards()` called
3. Each card's `UpdateFromGroup()` called
4. Data bindings updated
5. UI automatically refreshes

## Advanced Patterns

### Formatting Lists with Overflow

```go
func formatInstanceList(instances []int, maxVisible int) string {
    if len(instances) <= maxVisible {
        // Show all
        return "Instance 1, Instance 2, Instance 3"
    }

    // Show first N + overflow count
    visible := instances[:maxVisible]
    remaining := len(instances) - maxVisible
    return fmt.Sprintf("Instance 1, Instance 2 and %d more...", remaining)
}
```

### Binding Converters

Fyne provides built-in converters:

```go
// Bool to string
statusLabel := widget.NewLabelWithData(
    binding.BoolToString(c.data.IsActive)
)
// Shows "true" or "false"

// Int to formatted string
countLabel := widget.NewLabelWithData(
    binding.IntToStringWithFormat(c.data.Count, "Count: %d")
)
// Shows "Count: 42"
```

### Custom Computed Bindings

For complex calculations:

```go
type ComputedBinding struct {
    binding.String
    deps []binding.DataItem
}

func NewComputedString(compute func() string, deps ...binding.DataItem) binding.String {
    computed := binding.NewString()

    update := func() {
        computed.Set(compute())
    }

    for _, dep := range deps {
        dep.AddListener(binding.NewDataListener(update))
    }

    update() // Initial value
    return computed
}

// Usage:
progressBinding := NewComputedString(
    func() string {
        rem, _ := c.data.PoolRemaining.Get()
        tot, _ := c.data.PoolTotal.Get()
        pct := (float64(rem) / float64(tot)) * 100
        return fmt.Sprintf("%d/%d (%.1f%%)", rem, tot, pct)
    },
    c.data.PoolRemaining,
    c.data.PoolTotal,
)
```

## Testing Components

### Unit Test Example

```go
func TestOrchestrationCardData_UpdateFromGroup(t *testing.T) {
    // Create test group
    group := &bot.BotGroup{
        Name:        "Test Group",
        RoutineName: "test.yaml",
        ActiveBots:  make(map[int]*bot.BotInfo),
    }

    // Create card data
    data := NewOrchestrationCardData(group)

    // Verify initial values
    name, _ := data.GroupName.Get()
    assert.Equal(t, "Test Group", name)

    // Change group state
    group.Name = "Updated Group"
    data.UpdateFromGroup(group)

    // Verify binding updated
    name, _ = data.GroupName.Get()
    assert.Equal(t, "Updated Group", name)
}
```

## Common Pitfalls

### 1. Forgetting to Call Refresh()

```go
// ❌ Change won't show
c.statusIndicator.FillColor = color.Red

// ✅ Change will show
c.statusIndicator.FillColor = color.Red
c.statusIndicator.Refresh()
```

### 2. Accessing Unexported Fields

```go
// ❌ Race condition
group.running

// ✅ Thread-safe
group.IsRunning()
```

### 3. Not Using Bindings

```go
// ❌ Manual updates everywhere
label := widget.NewLabel("Status: Stopped")
// Later... where do we update this?

// ✅ Automatic updates via bindings
statusBinding := binding.NewString()
label := widget.NewLabelWithData(statusBinding)
statusBinding.Set("Status: Running") // UI updates automatically
```

### 4. Blocking the UI Thread

```go
// ❌ Blocks UI
button := widget.NewButton("Start", func() {
    time.Sleep(10 * time.Second) // UI frozen!
})

// ✅ Non-blocking
button := widget.NewButton("Start", func() {
    go func() {
        time.Sleep(10 * time.Second)
        // Update UI via event bus or bindings
    }()
})
```

## Summary

1. **Separate data from UI**: `*_data.go` and `*.go`
2. **Use bindings**: Let Fyne handle UI updates
3. **Computed bindings**: Derive values from other bindings
4. **Callbacks**: Keep business logic out of components
5. **Thread safety**: Always use exported getter methods
6. **Periodic updates**: Keep UI in sync with data

This pattern makes components:
- ✅ Reusable
- ✅ Maintainable
- ✅ Thread-safe
- ✅ Testable
- ✅ Easy to understand
