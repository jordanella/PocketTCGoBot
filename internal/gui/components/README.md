# GUI Components Library

This directory contains reusable UI components for building consistent, maintainable Fyne interfaces.

## Available Component Sets

### 📝 [Text Components](TEXT_COMPONENTS.md)
Typography components for consistent text styling:
- `Heading()` - Page titles (24px, bold)
- `Subheading()` - Section headers (18px, bold)
- `Body()` - Content text (14px)
- `Caption()` - Secondary info (12px)
- `BoldText()`, `MonospaceText()`, and more

### 🔘 [Button Components](TEXT_COMPONENTS.md#button-components)
Button variants for different actions:
- `PrimaryButton()` - Main actions (highlighted)
- `SecondaryButton()` - Standard actions
- `DangerButton()` - Destructive actions
- `StackedButton()` - Multi-line labels
- `ButtonGroup()` - Grouped buttons

### 🎴 [Card Components](CARD_COMPONENTS.md)
Container components with rounded corners and indentation:
- `Card()` - Basic card container
- `CardWithIndent()` - Indented cards for hierarchy
- `NestedCard()` - Multi-level nesting
- `CardSection()` - Cards with titles
- `CompactCard()` - Dense layouts

### 🏷️ Chip/Badge Components
Tag and status display components:
- `Chip()` - Basic clickable chip
- `StatusChip()` - Auto-colored status badges
- `NavigationChip()` - Clickable navigation chips
- `TruncatedChipList()` - Lists with "and N more..."
- `LabeledChipList()` - Label + chips combo

### 📐 [Layout Components](MOCKUP_PATTERNS.md)
Layout patterns from the mockups:
- `LabelButtonsRow()` - Labels left, buttons right
- `InlineLabels()` - Multiple labels with separator
- `TwoColumnLayout()` - Resizable split view
- `ReorderableRow()` - Rows with ▲▼ buttons
- `FieldRow()` - Label + input field
- `ActionBar()` - Bottom action buttons

### 📋 [Mockup Patterns Guide](MOCKUP_PATTERNS.md)
Complete guide to implementing the UI mockups with code examples

## Quick Start

```go
import "jordanella.com/pocket-tcg-go/internal/gui/components"

// Text
header := components.Heading("My Page")
description := components.Body("Page description here")

// Buttons
saveBtn := components.PrimaryButton("Save", handleSave)
cancelBtn := components.SecondaryButton("Cancel", handleCancel)

// Cards
card := components.Card(
    container.NewVBox(
        components.Subheading("Settings"),
        description,
        components.ButtonGroup(saveBtn, cancelBtn),
    ),
)
```

## Component Architecture

### Design Principles

1. **Separation of Concerns**: Complex components are split into files:
   - `*_data.go`: Data bindings and state management
   - `*.go`: UI rendering and event handling
   - Simple components may use a single file

2. **Data Binding**: Components use Fyne's data binding system for reactive updates
   - Changes to bound data automatically update the UI
   - Thread-safe updates via binding interfaces
   - Computed bindings for derived values

3. **Callback Pattern**: Components accept callback functions for user actions
   - Keeps business logic separate from UI code
   - Makes components reusable in different contexts

4. **Consistent Styling**: Preset components follow Material Design principles
   - Typography scale (Heading > Subheading > Body > Caption)
   - Button importance hierarchy (Primary > Secondary > Danger)
   - Visual hierarchy through cards and indentation

## Using the Orchestration Card Component

### Basic Usage

```go
import (
    "jordanella.com/pocket-tcg-go/internal/bot"
    "jordanella.com/pocket-tcg-go/internal/gui/components"
)

// Create a BotGroup (your data model)
group := &bot.BotGroup{
    Name:            "Premium Farmers",
    RoutineName:     "farm_premium.yaml",
    OrchestrationID: "abc-123",
    // ... other fields
}

// Define callbacks for button actions
callbacks := components.OrchestrationCardCallbacks{
    OnAddInstance: func(g *bot.BotGroup) {
        // Handle adding an instance
    },
    OnPauseResume: func(g *bot.BotGroup) {
        // Handle pause/resume
    },
    OnStop: func(g *bot.BotGroup) {
        // Handle stop
    },
    OnShutdown: func(g *bot.BotGroup) {
        // Handle shutdown
    },
}

// Create the card
card := components.NewOrchestrationCard(group, callbacks)

// Get the container for embedding in your layout
container := card.GetContainer()

// Add to your UI
myLayout.Add(container)
```

### Updating Card Data

The card automatically updates when you call `UpdateFromGroup()`:

```go
// Periodically refresh (e.g., in a goroutine)
ticker := time.NewTicker(1 * time.Second)
for range ticker.C {
    card.UpdateFromGroup()
}
```

The `UpdateFromGroup()` method:
- Reads current state from the `BotGroup`
- Updates all data bindings
- UI automatically reflects changes via Fyne's binding system
- Is thread-safe (uses proper locking when accessing group data)

## Creating New Components

Follow this pattern when creating new components:

### 1. Create the Data File (`component_data.go`)

```go
package components

import "fyne.io/fyne/v2/data/binding"

type MyComponentData struct {
    // Define bindings for each piece of data
    Title binding.String
    Count binding.Int
    IsActive binding.Bool

    // Computed bindings
    StatusText binding.String
}

func NewMyComponentData(model *MyModel) *MyComponentData {
    data := &MyComponentData{
        Title:      binding.NewString(),
        Count:      binding.NewInt(),
        IsActive:   binding.NewBool(),
        StatusText: binding.NewString(),
    }

    // Initialize from model
    data.UpdateFromModel(model)

    // Set up computed bindings
    data.setupComputedBindings()

    return data
}

func (d *MyComponentData) setupComputedBindings() {
    // Create bindings that derive from other bindings
    d.IsActive.AddListener(binding.NewDataListener(func() {
        active, _ := d.IsActive.Get()
        if active {
            d.StatusText.Set("Active")
        } else {
            d.StatusText.Set("Inactive")
        }
    }))
}

func (d *MyComponentData) UpdateFromModel(model *MyModel) {
    // Thread-safe read from model
    model.mu.RLock()
    defer model.mu.RUnlock()

    // Update all bindings
    d.Title.Set(model.Title)
    d.Count.Set(model.Count)
    d.IsActive.Set(model.Active)
}
```

### 2. Create the Component File (`component.go`)

```go
package components

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
)

type MyComponent struct {
    data      *MyComponentData
    model     *MyModel
    container *fyne.Container

    // Callbacks
    onAction func(*MyModel)
}

type MyComponentCallbacks struct {
    OnAction func(*MyModel)
}

func NewMyComponent(model *MyModel, callbacks MyComponentCallbacks) *MyComponent {
    c := &MyComponent{
        model:    model,
        data:     NewMyComponentData(model),
        onAction: callbacks.OnAction,
    }

    c.container = c.build()
    c.setupListeners()

    return c
}

func (c *MyComponent) build() *fyne.Container {
    // Create UI using data bindings
    titleLabel := widget.NewLabelWithData(c.data.Title)
    countLabel := widget.NewLabelWithData(
        binding.IntToStringWithFormat(c.data.Count, "Count: %d"),
    )
    statusLabel := widget.NewLabelWithData(c.data.StatusText)

    actionBtn := widget.NewButton("Action", func() {
        if c.onAction != nil {
            c.onAction(c.model)
        }
    })

    return container.NewVBox(
        titleLabel,
        countLabel,
        statusLabel,
        actionBtn,
    )
}

func (c *MyComponent) setupListeners() {
    // Add listeners for dynamic UI updates
    c.data.IsActive.AddListener(binding.NewDataListener(func() {
        // React to state changes
    }))
}

func (c *MyComponent) UpdateFromModel() {
    c.data.UpdateFromModel(c.model)
}

func (c *MyComponent) GetContainer() *fyne.Container {
    return c.container
}
```

## Key Fyne Binding Functions

### Creating Bindings

```go
binding.NewString()           // String data
binding.NewInt()              // Integer data
binding.NewBool()             // Boolean data
binding.NewFloat()            // Float data
binding.NewStringList()       // List of strings
```

### Using Bindings in Widgets

```go
widget.NewLabelWithData(binding.String)
widget.NewEntryWithData(binding.String)
widget.NewCheckWithData("Label", binding.Bool)
```

### Converting Bindings

```go
binding.BoolToString(boolBinding)                    // bool -> string
binding.IntToString(intBinding)                      // int -> string
binding.IntToStringWithFormat(intBinding, "%d")      // int -> formatted string
binding.FloatToStringWithFormat(floatBinding, "%.2f")
```

### Computed Bindings

```go
// Listen for changes and update derived values
myBinding.AddListener(binding.NewDataListener(func() {
    value, _ := myBinding.Get()
    // Update other bindings based on this value
}))
```

## Thread Safety

Always use proper locking when reading from shared data structures:

```go
// WRONG - Race condition
card.Title.Set(model.Title)

// RIGHT - Thread-safe
model.mu.RLock()
title := model.Title
model.mu.RUnlock()
card.Title.Set(title)
```

## Best Practices

1. **Keep Components Small**: Each component should do one thing well
2. **Use Bindings**: Let Fyne's binding system handle UI updates automatically
3. **Separate Concerns**: Data logic in `*_data.go`, UI logic in `*.go`
4. **Thread Safety**: Always lock when reading shared state
5. **Callbacks for Actions**: Don't put business logic in components
6. **Update Periodically**: Call `UpdateFromModel()` regularly for real-time data
7. **Clean Up**: Stop timers and goroutines when components are destroyed

## Examples

See the existing components for reference:
- [orchestration_card.go](orchestration_card.go) - Complete example with bindings
- [orchestration_card_data.go](orchestration_card_data.go) - Data binding patterns
