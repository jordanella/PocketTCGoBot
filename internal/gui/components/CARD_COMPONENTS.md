# Card Components Guide

Card components provide consistent, rounded-rectangle containers with automatic padding, borders, and easy indentation control for hierarchical layouts.

## Why Use Cards?

Cards help organize content into distinct, visually separated sections. They provide:
- **Visual hierarchy** through indentation
- **Consistent styling** with rounded corners and subtle borders
- **Easy nesting** for complex layouts
- **Elevation appearance** with slightly elevated backgrounds

---

## Basic Card Components

### Card
Basic card with standard padding.

```go
content := widget.NewLabel("Card content")
card := components.Card(content)
```

**When to use:**
- Grouping related content
- Creating visual sections
- Simple containers

---

### CardWithIndent
Card with configurable left margin for indentation.

```go
content := widget.NewLabel("Indented content")
card := components.CardWithIndent(content, 20) // 20px left indent
```

**Parameters:**
- `content`: The widget/container to wrap
- `leftIndent`: Left margin in pixels (float32)

**When to use:**
- Nested/hierarchical content
- Child items under a parent
- Showing relationships between items

**Example indentation levels:**
- `0` - No indent (top level)
- `10` - Slight indent
- `20` - Standard indent (one level)
- `40` - Two levels deep
- `60` - Three levels deep

---

### CardWithOptions
Fully customizable card with all options.

```go
padding := float32(8)
radius := float32(6)
bgColor := color.NRGBA{R: 240, G: 240, B: 250, A: 255}

card := components.CardWithOptions(content, components.CardOptions{
    LeftIndent:      20,
    PaddingOverride: &padding,
    BackgroundColor: &bgColor,
    CornerRadius:    &radius,
})
```

**CardOptions fields:**
- `LeftIndent` (float32): Left margin in pixels
- `PaddingOverride` (*float32): Custom padding (nil = default, 0 = no padding)
- `BackgroundColor` (*color.Color): Custom background (nil = theme default)
- `CornerRadius` (*float32): Border radius (nil = 4px default)
- `Elevation` (int): Reserved for future shadow/depth effects

**When to use:**
- Custom styling requirements
- Special visual effects
- Brand-specific colors

---

## Preset Card Types

### SimpleCard
Alias for `Card()` - basic card with standard padding.

```go
card := components.SimpleCard(content)
```

---

### IndentedCard
Card indented by 20px (one level).

```go
card := components.IndentedCard(content)
```

**When to use:**
- Child items in a list
- Subcategories
- Related but secondary content

---

### CompactCard
Card with reduced padding (4px instead of standard).

```go
card := components.CompactCard(content)
```

**When to use:**
- Dense layouts
- Lists with many items
- Space-constrained areas

---

### NestedCard
Card indented by level (level × 20px).

```go
card := components.NestedCard(content, 0) // No indent
card := components.NestedCard(content, 1) // 20px indent
card := components.NestedCard(content, 2) // 40px indent
card := components.NestedCard(content, 3) // 60px indent
```

**When to use:**
- Tree structures
- Multi-level hierarchies
- Dynamic nesting depth

---

## Card Utilities

### CardList
Wraps multiple items in cards and stacks them vertically.

```go
items := components.CardList(
    widget.NewLabel("Item 1"),
    widget.NewLabel("Item 2"),
    widget.NewLabel("Item 3"),
)
```

**When to use:**
- Quick card-based lists
- Uniform card styling
- Prototyping

---

### CardSection
Card with a subheading title and content.

```go
card := components.CardSection(
    "Settings",
    container.NewVBox(
        widget.NewCheck("Enable feature", nil),
        widget.NewCheck("Auto-save", nil),
    ),
)
```

**When to use:**
- Titled sections
- Settings groups
- Feature categories

---

### CardSectionWithIndent
Titled card with configurable indentation.

```go
card := components.CardSectionWithIndent(
    "Advanced Options",
    settingsForm,
    20, // 20px indent
)
```

**When to use:**
- Nested titled sections
- Sub-settings groups
- Hierarchical configuration

---

## Complete Examples

### Example 1: Simple Information Card

```go
func BuildInfoCard() fyne.CanvasObject {
    title := components.Subheading("Bot Status")
    status := components.Body("Running")
    uptime := components.Caption("Uptime: 2h 15m")

    content := container.NewVBox(
        title,
        status,
        uptime,
    )

    return components.Card(content)
}
```

---

### Example 2: Hierarchical Settings

```go
func BuildSettings() fyne.CanvasObject {
    // Top-level card
    generalSettings := components.CardSection(
        "General Settings",
        container.NewVBox(
            widget.NewCheck("Enable notifications", nil),
            widget.NewCheck("Auto-start", nil),
        ),
    )

    // Nested card (indented)
    advancedSettings := components.CardSectionWithIndent(
        "Advanced",
        container.NewVBox(
            widget.NewCheck("Debug mode", nil),
            widget.NewCheck("Verbose logging", nil),
        ),
        20, // Indent to show it's under General
    )

    return container.NewVBox(
        generalSettings,
        advancedSettings,
    )
}
```

---

### Example 3: Multi-Level Nesting

```go
func BuildNestedCards() fyne.CanvasObject {
    // Level 0: Top level
    topCard := components.NestedCard(
        components.BoldText("Root Item"),
        0,
    )

    // Level 1: First level children
    child1 := components.NestedCard(
        components.Body("Child 1"),
        1,
    )

    // Level 2: Second level children
    grandchild := components.NestedCard(
        components.Body("Grandchild 1.1"),
        2,
    )

    // Level 1: Another first level child
    child2 := components.NestedCard(
        components.Body("Child 2"),
        1,
    )

    return container.NewVBox(
        topCard,
        child1,
        grandchild,
        child2,
    )
}
```

**Output visualization:**
```
┌─────────────────────┐
│ Root Item           │
└─────────────────────┘
    ┌─────────────────┐
    │ Child 1         │
    └─────────────────┘
        ┌─────────────┐
        │ Grandchild  │
        └─────────────┘
    ┌─────────────────┐
    │ Child 2         │
    └─────────────────┘
```

---

### Example 4: Dynamic Indent Levels

```go
func BuildDynamicTree(items []TreeItem) fyne.CanvasObject {
    cards := container.NewVBox()

    for _, item := range items {
        content := widget.NewLabel(item.Name)
        card := components.NestedCard(content, item.Level)
        cards.Add(card)
    }

    return cards
}

type TreeItem struct {
    Name  string
    Level int
}

// Usage:
tree := []TreeItem{
    {"Root", 0},
    {"Child A", 1},
    {"Child A.1", 2},
    {"Child A.2", 2},
    {"Child B", 1},
}
treeView := BuildDynamicTree(tree)
```

---

### Example 5: Card List with Mixed Content

```go
func BuildActivityFeed() fyne.CanvasObject {
    activities := components.CardList(
        container.NewVBox(
            components.BoldText("Bot Started"),
            components.Caption("2 minutes ago"),
        ),
        container.NewVBox(
            components.BoldText("Routine Completed"),
            components.Caption("5 minutes ago"),
        ),
        container.NewVBox(
            components.BoldText("Account Added"),
            components.Caption("10 minutes ago"),
        ),
    )

    return container.NewVBox(
        components.Heading("Activity Feed"),
        activities,
    )
}
```

---

### Example 6: Custom Styled Card

```go
func BuildCustomCard() fyne.CanvasObject {
    // Custom styling
    padding := float32(16)
    radius := float32(8)
    bgColor := color.NRGBA{R: 255, G: 248, B: 220, A: 255} // Light yellow

    content := container.NewVBox(
        components.Subheading("Warning"),
        components.Body("This action cannot be undone"),
    )

    return components.CardWithOptions(content, components.CardOptions{
        PaddingOverride: &padding,
        CornerRadius:    &radius,
        BackgroundColor: &bgColor,
    })
}
```

---

### Example 7: Compact List

```go
func BuildCompactList() fyne.CanvasObject {
    items := []string{"Item 1", "Item 2", "Item 3", "Item 4"}

    cards := container.NewVBox()
    for _, item := range items {
        card := components.CompactCard(widget.NewLabel(item))
        cards.Add(card)
    }

    return cards
}
```

---

### Example 8: Settings Page with Cards

```go
func BuildSettingsPage() fyne.CanvasObject {
    // Account settings section
    accountCard := components.CardSection(
        "Account",
        container.NewVBox(
            components.BoldText("Email:"),
            widget.NewEntry(),
            components.BoldText("Username:"),
            widget.NewEntry(),
        ),
    )

    // Notifications section
    notifCard := components.CardSection(
        "Notifications",
        container.NewVBox(
            widget.NewCheck("Email notifications", nil),
            widget.NewCheck("Push notifications", nil),
            widget.NewCheck("SMS alerts", nil),
        ),
    )

    // Advanced section (indented)
    advancedCard := components.CardSectionWithIndent(
        "Advanced",
        container.NewVBox(
            widget.NewCheck("Developer mode", nil),
            widget.NewCheck("Show debug info", nil),
        ),
        20,
    )

    // Actions
    saveBtn := components.PrimaryButton("Save Changes", func() {
        // Save logic
    })

    return container.NewVBox(
        components.Heading("Settings"),
        accountCard,
        notifCard,
        advancedCard,
        widget.NewSeparator(),
        saveBtn,
    )
}
```

---

### Example 9: Card with No Padding

```go
func BuildImageCard() fyne.CanvasObject {
    // For images or content that should touch edges
    zeroPadding := float32(0)

    image := canvas.NewImageFromFile("screenshot.png")
    image.FillMode = canvas.ImageFillContain

    return components.CardWithOptions(image, components.CardOptions{
        PaddingOverride: &zeroPadding,
        CornerRadius:    func() *float32 { r := float32(8); return &r }(),
    })
}
```

---

## Indentation Guide

### Standard Indentation Levels

| Level | Pixels | Use Case |
|-------|--------|----------|
| 0 | 0px | Top-level items, main sections |
| 1 | 20px | Direct children, sub-sections |
| 2 | 40px | Nested children, sub-sub-sections |
| 3 | 60px | Deep nesting, tertiary items |
| 4+ | 80px+ | Very deep hierarchies (use sparingly) |

### Visual Example

```
Card (0px indent)
└─ Card (20px indent)
   └─ Card (40px indent)
      └─ Card (60px indent)
```

### When to Use Indentation

✅ **Good use cases:**
- Parent-child relationships
- Category and subcategories
- Tree structures
- Hierarchical data
- Settings and sub-settings

❌ **Avoid:**
- More than 3-4 levels (gets confusing)
- Unrelated items at different indents
- Indentation without clear hierarchy

---

## Styling Guidelines

### Default Card Appearance

- **Corner Radius**: 4px (slightly rounded)
- **Border**: 1px separator color
- **Background**: Slightly elevated from page background
- **Padding**: Standard theme padding (~8-12px)

### Customization Tips

1. **Consistent Indents**: Use multiples of 20px (0, 20, 40, 60)
2. **Subtle Colors**: Keep background colors subtle for readability
3. **Appropriate Padding**: Use standard padding unless space is constrained
4. **Corner Radius**: 4-8px works well; avoid extreme values

---

## Common Patterns

### Pattern 1: List of Cards

```go
cards := container.NewVBox(
    components.Card(item1),
    components.Card(item2),
    components.Card(item3),
)
```

### Pattern 2: Parent with Indented Children

```go
layout := container.NewVBox(
    components.Card(parent),
    components.IndentedCard(child1),
    components.IndentedCard(child2),
)
```

### Pattern 3: Titled Section

```go
section := components.CardSection("Title", content)
```

### Pattern 4: Scrollable Card List

```go
cards := components.CardList(item1, item2, item3, /*...*/)
scroll := container.NewVScroll(cards)
```

---

## Best Practices

### 1. Use Semantic Indentation
Indent should represent actual hierarchy:
```go
// ✅ Good: Clear parent-child relationship
components.Card(parent)
components.IndentedCard(child)

// ❌ Bad: Random indentation
components.IndentedCard(unrelatedItem)
```

### 2. Consistent Spacing
Use `NewVBox` with cards for vertical spacing:
```go
// ✅ Good: Consistent spacing between cards
container.NewVBox(
    components.Card(item1),
    components.Card(item2),
    components.Card(item3),
)
```

### 3. Don't Over-nest
```go
// ❌ Avoid: Too many levels
components.NestedCard(content, 5) // 100px indent!

// ✅ Better: Max 3 levels
components.NestedCard(content, 2) // 40px indent
```

### 4. Use Appropriate Card Type
```go
// ✅ Use the simplest component that fits
components.Card(simple)                    // Basic card
components.IndentedCard(nested)            // One level down
components.CardSection("Title", complex)   // Needs title

// ❌ Avoid over-engineering
components.CardWithOptions(simple, components.CardOptions{
    LeftIndent: 0,
    // ... many default values
})
```

---

## Quick Reference

```go
// BASIC CARDS
components.Card(content)                          // Standard card
components.CardWithIndent(content, 20)            // Custom indent
components.SimpleCard(content)                    // Alias for Card
components.IndentedCard(content)                  // 20px indent
components.CompactCard(content)                   // Less padding

// NESTED CARDS
components.NestedCard(content, 0)                 // No indent
components.NestedCard(content, 1)                 // 20px indent
components.NestedCard(content, 2)                 // 40px indent

// CARD SECTIONS
components.CardSection("Title", content)          // Card with title
components.CardSectionWithIndent("Title", content, 20)

// CARD UTILITIES
components.CardList(item1, item2, item3)          // Multiple cards

// CUSTOM CARDS
components.CardWithOptions(content, components.CardOptions{
    LeftIndent:      20,
    PaddingOverride: &customPadding,
    BackgroundColor: &customColor,
    CornerRadius:    &customRadius,
})
```

---

## Integration with Text/Button Components

Cards work great with other components:

```go
card := components.Card(
    container.NewVBox(
        components.Heading("Title"),
        components.Body("Description here"),
        components.Caption("Additional info"),
        widget.NewSeparator(),
        components.ButtonGroup(
            components.PrimaryButton("Action", handler),
            components.SecondaryButton("Cancel", handler),
        ),
    ),
)
```
