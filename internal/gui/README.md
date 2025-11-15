# GUI Package

Fyne-based graphical user interface for the PocketTCG Bot automation system.

## 📁 Structure

```
internal/gui/
├── components/              # Reusable UI components with data bindings
│   ├── README.md           # Component development guide
│   ├── orchestration_card.go
│   └── orchestration_card_data.go
│
├── tabs/                   # Main application tabs
│   └── orchestration.go    # Orchestration groups management
│
├── database/               # Database visualization tabs
│
├── controller.go           # Main GUI controller (MVC)
├── eventbus.go            # Thread-safe event system
├── theme.go               # Application theming
│
├── GUI_STRUCTURE.md       # Architecture overview
├── COMPONENT_EXAMPLE.md   # Complete component tutorial
└── MIGRATION_GUIDE.md     # Migration plan for existing code
```

## 🚀 Quick Start

### Creating a New Component

```go
import "jordanella.com/pocket-tcg-go/internal/gui/components"

// 1. Create your data model (BotGroup, etc.)
group := &bot.BotGroup{Name: "My Group"}

// 2. Define callbacks
callbacks := components.OrchestrationCardCallbacks{
    OnAddInstance: func(g *bot.BotGroup) {
        // Handle add instance
    },
    // ... other callbacks
}

// 3. Create component
card := components.NewOrchestrationCard(group, callbacks)

// 4. Add to UI
container.Add(card.GetContainer())

// 5. Update periodically
ticker := time.NewTicker(1 * time.Second)
go func() {
    for range ticker.C {
        card.UpdateFromGroup()
    }
}()
```

### Creating a New Tab

```go
package tabs

import "jordanella.com/pocket-tcg-go/internal/gui"

type MyTab struct {
    controller *gui.Controller
}

func NewMyTab(ctrl *gui.Controller) *MyTab {
    return &MyTab{controller: ctrl}
}

func (t *MyTab) Build() fyne.CanvasObject {
    return container.NewVBox(
        widget.NewLabel("My Tab Content"),
    )
}
```

## 📚 Documentation

### For Component Development
- [**Component Development Guide**](components/README.md) - How to create components
- [**Complete Example**](COMPONENT_EXAMPLE.md) - Detailed walkthrough with explanations
- [**Architecture Overview**](GUI_STRUCTURE.md) - High-level design and patterns

### For Migration
- [**Migration Guide**](MIGRATION_GUIDE.md) - How to move existing code to new structure

## 🔑 Key Concepts

### 1. Data Binding

Fyne's data binding system enables reactive UI updates:

```go
// Create binding
nameBinding := binding.NewString()
nameBinding.Set("Initial Value")

// Use in widget
label := widget.NewLabelWithData(nameBinding)

// Update (UI automatically refreshes)
nameBinding.Set("New Value")
```

### 2. Component Architecture

Components are split into two files:

- **`component_data.go`**: Data bindings and state management
- **`component.go`**: UI rendering and event handling

Benefits:
- Clear separation of concerns
- Easier to maintain and test
- Reusable across different contexts

### 3. Thread Safety

Always use exported methods when accessing shared state:

```go
// ❌ WRONG - Race condition
isRunning := group.running

// ✅ RIGHT - Thread-safe
isRunning := group.IsRunning()
```

### 4. Callback Pattern

Components accept callbacks for user actions:

```go
type MyComponentCallbacks struct {
    OnAction func(*Model)
}

func NewMyComponent(model *Model, callbacks MyComponentCallbacks) *MyComponent {
    // Component delegates business logic to callbacks
}
```

This keeps business logic in tabs/controllers, not in components.

### 5. Event Bus

For thread-safe UI updates from background goroutines:

```go
// From any goroutine
ctrl.eventBus.Dispatch(func() {
    // This runs on the main UI thread
    widget.Refresh()
})
```

## 🎨 Component Patterns

### Basic Component

```go
// Data bindings
type MyComponentData struct {
    Title binding.String
    Count binding.Int
}

// Component
type MyComponent struct {
    data      *MyComponentData
    container *fyne.Container
}

// Constructor
func NewMyComponent(model *Model) *MyComponent {
    c := &MyComponent{
        data: NewMyComponentData(model),
    }
    c.container = c.build()
    return c
}

// Build UI
func (c *MyComponent) build() *fyne.Container {
    titleLabel := widget.NewLabelWithData(c.data.Title)
    countLabel := widget.NewLabelWithData(
        binding.IntToStringWithFormat(c.data.Count, "Count: %d"),
    )
    return container.NewVBox(titleLabel, countLabel)
}

// Update from model
func (c *MyComponent) UpdateFromModel(model *Model) {
    c.data.UpdateFromModel(model)
}
```

### Computed Bindings

```go
// Create computed binding that updates when dependencies change
type ComputedData struct {
    Value1   binding.Int
    Value2   binding.Int
    Computed binding.String
}

func (d *ComputedData) setupComputed() {
    update := func() {
        v1, _ := d.Value1.Get()
        v2, _ := d.Value2.Get()
        d.Computed.Set(fmt.Sprintf("%d + %d = %d", v1, v2, v1+v2))
    }

    d.Value1.AddListener(binding.NewDataListener(update))
    d.Value2.AddListener(binding.NewDataListener(update))

    update() // Initial value
}
```

### Dynamic UI Updates

```go
// Listen to binding changes and update UI elements
func (c *MyComponent) setupListeners() {
    c.data.IsActive.AddListener(binding.NewDataListener(func() {
        active, _ := c.data.IsActive.Get()
        if active {
            c.statusIcon.FillColor = color.Green
            c.button.SetText("Deactivate")
        } else {
            c.statusIcon.FillColor = color.Gray
            c.button.SetText("Activate")
        }
        c.statusIcon.Refresh()
    }))
}
```

## 🔧 Common Patterns

### Periodic Updates

```go
func (t *MyTab) startPeriodicRefresh() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            for _, component := range t.components {
                component.UpdateFromModel()
            }
        case <-t.stopRefresh:
            return
        }
    }
}
```

### List Management

```go
type ListTab struct {
    items     []*MyComponent
    itemsMu   sync.RWMutex
    container *fyne.Container
}

func (t *ListTab) AddItem(item *MyComponent) {
    t.itemsMu.Lock()
    defer t.itemsMu.Unlock()

    t.items = append(t.items, item)
    t.container.Add(item.GetContainer())
    t.container.Refresh()
}

func (t *ListTab) RemoveItem(item *MyComponent) {
    t.itemsMu.Lock()
    defer t.itemsMu.Unlock()

    t.container.Remove(item.GetContainer())
    // Remove from slice...
    t.container.Refresh()
}
```

### Dialog Patterns

```go
func (t *MyTab) showCreateDialog() {
    nameEntry := widget.NewEntry()
    nameEntry.SetPlaceHolder("Enter name...")

    form := container.NewVBox(
        widget.NewLabel("Name:"),
        nameEntry,
    )

    dialog.NewCustomConfirm(
        "Create Item",
        "Create",
        "Cancel",
        form,
        func(confirmed bool) {
            if confirmed {
                t.createItem(nameEntry.Text)
            }
        },
        t.window,
    ).Show()
}
```

## 🧪 Testing

### Component Tests

```go
func TestComponentCreation(t *testing.T) {
    model := &Model{Name: "Test"}
    component := NewMyComponent(model)

    assert.NotNil(t, component)

    name, _ := component.data.Name.Get()
    assert.Equal(t, "Test", name)
}

func TestComponentUpdate(t *testing.T) {
    model := &Model{Name: "Initial"}
    component := NewMyComponent(model)

    model.Name = "Updated"
    component.UpdateFromModel(model)

    name, _ := component.data.Name.Get()
    assert.Equal(t, "Updated", name)
}
```

### Integration Tests

```go
func TestTabRendering(t *testing.T) {
    app := test.NewApp()
    window := app.NewWindow("Test")

    ctrl := NewController(config, app, window)
    tab := tabs.NewMyTab(ctrl)

    content := tab.Build()
    assert.NotNil(t, content)

    window.SetContent(content)
    // Verify rendering...
}
```

## 🎯 Best Practices

1. **Separation of Concerns**
   - Data logic in `*_data.go`
   - UI logic in `*.go`
   - Business logic in callbacks

2. **Thread Safety**
   - Always use exported getter methods
   - Lock when accessing shared state
   - Use event bus for UI updates from goroutines

3. **Performance**
   - Limit refresh frequency (1-2 seconds is usually enough)
   - Use `GetAllBotInfo()` instead of locking and iterating
   - Avoid blocking the UI thread

4. **Maintainability**
   - Keep components small and focused
   - Document complex bindings
   - Use meaningful variable names

5. **Testing**
   - Test components in isolation
   - Mock callbacks for unit tests
   - Integration tests for full workflows

## 🐛 Common Pitfalls

### 1. Forgetting to Refresh

```go
// ❌ Won't show
circle.FillColor = color.Red

// ✅ Will show
circle.FillColor = color.Red
circle.Refresh()
```

### 2. Race Conditions

```go
// ❌ Race condition
count := len(group.ActiveBots)

// ✅ Thread-safe
count := group.GetActiveBotCount()
```

### 3. Not Using Bindings

```go
// ❌ Manual updates needed everywhere
label := widget.NewLabel("Count: 0")
// Later: label.SetText("Count: 1")
// Later: label.SetText("Count: 2")

// ✅ Automatic updates
countBinding := binding.NewInt()
label := widget.NewLabelWithData(
    binding.IntToStringWithFormat(countBinding, "Count: %d"),
)
countBinding.Set(1) // UI updates automatically
```

### 4. Blocking UI Thread

```go
// ❌ Blocks UI for 5 seconds
button := widget.NewButton("Process", func() {
    time.Sleep(5 * time.Second)
})

// ✅ Non-blocking
button := widget.NewButton("Process", func() {
    go func() {
        time.Sleep(5 * time.Second)
        // Update UI via event bus
    }()
})
```

## 📖 Additional Resources

- [Fyne Documentation](https://docs.fyne.io/)
- [Fyne Data Binding](https://docs.fyne.io/binding/)
- [Fyne Layout Guide](https://docs.fyne.io/container/)
- [Fyne Widget Reference](https://docs.fyne.io/widget/)

## 🤝 Contributing

When adding new components or tabs:

1. Follow the established patterns (see `components/` for examples)
2. Create both `*_data.go` and `*.go` files
3. Use data bindings for reactive updates
4. Accept callbacks for user actions
5. Document complex patterns
6. Add tests
7. Update this README if adding new patterns

## 📝 TODO

- [ ] Migrate existing tabs to `tabs/` directory
- [ ] Migrate existing database tabs to `database/` directory
- [ ] Extract more reusable components
- [ ] Add unit tests for components
- [ ] Add integration tests for tabs
- [ ] Create custom theme
- [ ] Add keyboard shortcuts
- [ ] Improve accessibility

## 📄 License

Part of the PocketTCG Bot project. See root LICENSE file.
