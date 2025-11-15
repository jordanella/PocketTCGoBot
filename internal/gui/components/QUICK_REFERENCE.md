# Component Quick Reference

Visual guide to all available components with usage examples.

## 📝 Text Components

```go
components.Heading("Page Title")           // ██ Large, bold (24px)
components.Subheading("Section Title")     // █  Medium, bold (18px)
components.Body("Regular text here")       // ▌  Standard (14px)
components.Caption("Small hint text")      // ▌  Small (12px)
components.BoldText("Emphasized")          // █  Bold standard
components.MonospaceText("/path/to/file")  // ▌  Fixed-width font
```

## 🔘 Buttons

```go
components.PrimaryButton("Save", fn)       // [▓▓▓▓▓] Highlighted
components.SecondaryButton("Cancel", fn)   // [░░░░░] Standard
components.DangerButton("Delete", fn)      // [▓▓▓▓▓] Red/warning
components.IconButton("Refresh", icon, fn) // [🔄 Refresh]
components.StackedButton("Launch", "Start bots", fn) // [Launch
                                                      //  Start bots]
```

## 🎴 Cards

```go
// Basic card with padding
components.Card(content)
┌─────────────────┐
│                 │
│    Content      │
│                 │
└─────────────────┘

// Indented card (20px)
components.IndentedCard(content)
    ┌─────────────┐
    │   Content   │
    └─────────────┘

// Nested cards
components.NestedCard(content, 0)  // Level 0
┌─────────────────┐
components.NestedCard(content, 1)  // Level 1
    ┌───────────┐
components.NestedCard(content, 2)  // Level 2
        ┌──────┐

// Card with title
components.CardSection("Settings", content)
┌─────────────────┐
│ ■ Settings      │
│                 │
│    Content      │
└─────────────────┘
```

## 🏷️ Chips

```go
// Basic chip
components.Chip("Tag", onClick)
  ┌──────┐
  │ Tag  │
  └──────┘

// Status chips (auto-colored)
components.StatusChip("Active")    // Green
components.StatusChip("Idle")      // Blue
components.StatusChip("Error")     // Red

// Navigation chip (clickable, highlighted)
components.NavigationChip("Go to Pool", onClick)

// Chip list with truncation
components.TruncatedChipList(items, 3, onClick)
┌──────┐ ┌──────┐ ┌──────┐ and 5 more...
│Item 1│ │Item 2│ │Item 3│

// Labeled chip list
components.LabeledChipList("Pools", items, 3, onClick)
Pools: ┌──────┐ ┌──────┐ ┌──────┐
       │Pool A│ │Pool B│ │Pool C│
       └──────┘ └──────┘ └──────┘
```

## 📐 Layout Components

### Label-Buttons Row
```go
components.LabelButtonsRow(labels, btn1, btn2, btn3)

Instance Name - Index 5                    [Pause] [Stop] [Shutdown]
└─ labels (left aligned)    buttons (right aligned) ─┘
```

### Inline Labels
```go
components.InlineLabels(" - ", label1, label2, label3)

Label 1 - Label 2 - Label 3
```

### Two-Column Layout
```go
components.TwoColumnLayout(left, right, 250)

┌─────────┬──────────────────────┐
│  List   │   Details            │
│         │                      │
│ Item 1  │   Name: Item 1       │
│ Item 2  │   Status: Active     │
│ Item 3  │   ...                │
│         │                      │
└─────────┴──────────────────────┘
```

### Reorderable Row
```go
components.ReorderableRow(content, moveUp, moveDown, remove)

Content here                     [▲] [▼] [Remove]
```

### Field Rows
```go
// Vertical field
components.FieldRow("Name", entry)
Name
┌─────────────────┐
│ [input field]   │
└─────────────────┘

// Required field
components.RequiredFieldRow("Name", entry, "Must be unique")
Name *
┌─────────────────┐
│ [input field]   │
└─────────────────┘
Must be unique

// Inline field
components.FieldRowInline("Limit", entry)
Limit ┌──────┐
      │ 100  │
      └──────┘
```

### Action Bar
```go
components.ActionBarSingle(btn1, btn2, btn3)

                                          [Save] [Cancel] [Delete]
└─ Buttons right-aligned ──────────────────────────────────────┘
```

## 📊 Complete Patterns

### Orchestration Group Card
```go
┌────────────────────────────────────────────────────┐
│ Premium Farmers <ID123>              [+ Instance]  │
│ Farms premium packs daily                          │
│ Started: 2h ago  Pool: Premium (5/10)              │
└────────────────────────────────────────────────────┘
    ┌──────────────────────────────────────────────┐
    │ Instance 1 - Index 5    [Pause] [Stop] [⚠️]   │
    │ Account user@example.com since 1h ago        │
    │ Status: Active                               │
    └──────────────────────────────────────────────┘
    ┌──────────────────────────────────────────────┐
    │ Instance 2 - Index 6    [Pause] [Stop] [⚠️]   │
    │ Account user2@example.com since 30m ago      │
    │ Status: Running                              │
    └──────────────────────────────────────────────┘
```

### Account Pool List
```go
Account Pools: ┌──────────┐ ┌───────────┐ ┌──────────┐ and 2 more...
               │ Premium  │ │ Event     │ │ Testing  │
               └──────────┘ └───────────┘ └──────────┘

Active Instances: ┌────┐ ┌────┐ ┌────┐
                  │ 1  │ │ 2  │ │ 3  │
                  └────┘ └────┘ └────┘
```

### Status Display
```go
Status: ┌────────┐ - routine_name
        │ Active │
        └────────┘
        └─green─┘
```

### Table with Cards
```go
Account           Packs    Shinedust    Status
──────────────────────────────────────────────────────
┌──────────────────────────────────────────────────┐
│ user@example.com  12     86,500  ┌────────┐      │
│                                   │ Active │ ...  │
│                                   └────────┘      │
└──────────────────────────────────────────────────┘
┌──────────────────────────────────────────────────┐
│ user2@example.com 12     73,500  ┌──────┐        │
│                                   │ Idle │        │
│                                   └──────┘        │
└──────────────────────────────────────────────────┘
```

## 🎯 Common Use Cases

### Form Layout
```go
form := container.NewVBox(
    components.Heading("Create Bot"),
    components.Body("Fill in the details below"),
    widget.NewSeparator(),

    components.RequiredFieldRow("Bot Name", nameEntry, "Must be unique"),
    components.FieldRow("Description", descEntry),
    components.FieldRowInline("Instance", instanceSelect),

    widget.NewSeparator(),
    components.ActionBarSingle(
        components.PrimaryButton("Create", onCreate),
        components.SecondaryButton("Cancel", onCancel),
    ),
)
```

### Card List with Hierarchy
```go
list := container.NewVBox(
    // Parent card
    components.Card(parentContent),

    // Child cards (indented)
    components.IndentedCard(child1Content),
    components.IndentedCard(child2Content),

    // Grandchild (double indent)
    components.NestedCard(grandchildContent, 2),
)
```

### Info Panel
```go
panel := components.Card(
    container.NewVBox(
        components.Subheading("Bot Info"),

        container.NewHBox(
            components.BoldText("Status:"),
            components.StatusChip("Running"),
        ),

        container.NewHBox(
            components.BoldText("Uptime:"),
            components.Caption("2h 15m"),
        ),

        components.LabeledNavigationChipList(
            "Pools",
            []string{"Premium", "Event"},
            5,
            navigateToPool,
        ),
    ),
)
```

### Two-Column Details View
```go
// Left: List
list := container.NewVBox(
    components.CompactCard(item1),
    components.CompactCard(item2),
    components.CompactCard(item3),
)

// Right: Tabs
tabs := components.TabPanel(
    container.NewTabItem("Details", detailsView),
    container.NewTabItem("Settings", settingsView),
    container.NewTabItem("Logs", logsView),
)

view := components.TwoColumnLayout(
    container.NewVScroll(list),
    tabs,
    250, // left min width
)
```

## 💡 Tips

### Visual Hierarchy
```
Level 0 (Top)     components.Heading()
Level 1           components.Subheading()
Level 2           components.BoldText()
Level 3           components.Body()
Level 4 (Detail)  components.Caption()
```

### Indentation Levels
```
0px   - Top level items
20px  - First level children (components.IndentedCard or level 1)
40px  - Second level (level 2)
60px  - Third level (level 3)
```

### Button Importance
```
One     PrimaryButton    - Main action
Many    SecondaryButton  - Supporting actions
Zero-One DangerButton    - Destructive actions
```

### Color Coding (Status Chips)
```
Green   - Active, Running, Success, Completed
Blue    - Idle, Pending, Waiting, Info
Red     - Error, Failed, Stopped, Offline
Orange  - Warning, Limited
Gray    - Default, Disabled, Unknown
```

## 🔧 Import Statement

```go
import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "jordanella.com/pocket-tcg-go/internal/gui/components"
)
```

## 📚 Documentation Links

- [Text Components](TEXT_COMPONENTS.md) - Typography guide
- [Card Components](CARD_COMPONENTS.md) - Card patterns
- [Mockup Patterns](MOCKUP_PATTERNS.md) - Implementing mockups
- [Complete Examples](EXAMPLES.md) - Real-world examples
