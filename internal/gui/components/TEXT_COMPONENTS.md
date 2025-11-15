# Text & Button Components Guide

This guide explains how to use the reusable text and button components for consistent styling across the application.

## Text Components

All text components are in `components/text.go` and provide consistent typography following Material Design principles.

### Basic Text Components

#### Heading
Large, bold text for main page titles and primary headers.

```go
header := components.Heading("Orchestration Groups")
```

**When to use:**
- Main page titles
- Primary section headers
- Dialog titles

---

#### Subheading
Medium-sized bold text for section headers and card titles.

```go
section := components.Subheading("Active Bots")
```

**When to use:**
- Section headers within a page
- Card titles
- Secondary headers

---

#### Body
Standard text for descriptions, paragraphs, and general content. Automatically wraps text.

```go
description := components.Body("This is a description of the feature")
```

**When to use:**
- Descriptions
- Paragraphs
- General content
- Help text

---

#### BodyRich
Body text using RichText for when you need more control over styling.

```go
text := components.BodyRich("Some flexible text")
```

**When to use:**
- When you need RichText features
- When standard Label isn't flexible enough

---

#### Caption
Small text for hints, secondary information, and footnotes.

```go
hint := components.Caption("Last updated: 2 minutes ago")
```

**When to use:**
- Timestamps
- Hints and tips
- Secondary information
- Footnotes
- Status text

---

#### BoldText
Bold text at standard size for emphasis.

```go
emphasized := components.BoldText("Important:")
```

**When to use:**
- Emphasizing specific words
- Labels in forms
- Important callouts

---

#### MonospaceText
Monospace text for code, paths, or technical content.

```go
path := components.MonospaceText("/path/to/file.txt")
```

**When to use:**
- File paths
- Code snippets
- Technical identifiers
- Console output

---

### Advanced Text Components

#### SizedText
Creates canvas.Text with precise size control. Returns `*canvas.Text`, not a widget.

```go
customText := components.SizedText("Custom Size", 20, true)
// true = bold, false = normal weight
```

**Important:** This creates a `canvas.Text`, which doesn't auto-update with theme changes. Use sparingly.

**When to use:**
- When you need exact pixel size control
- Custom UI elements
- Special visual effects

---

#### CustomRichText
Fully customizable RichText for complex styling needs.

```go
custom := components.CustomRichText(components.CustomRichTextStyle{
    Text:      "Custom Styled Text",
    SizeName:  theme.SizeNameHeadingText,
    Bold:      true,
    Italic:    false,
    Monospace: false,
    ColorName: theme.ColorNamePrimary,
})
```

**When to use:**
- None of the preset styles fit
- You need multiple style properties
- Custom color requirements

---

## Button Components

All button components are in `components/buttons.go`.

### Standard Buttons

#### PrimaryButton
High-importance button for main actions (highlighted styling).

```go
createBtn := components.PrimaryButton("Create New Group", func() {
    // Handle creation
})
```

**When to use:**
- Primary action on a page
- Affirmative actions in dialogs
- Main call-to-action

---

#### SecondaryButton
Standard button with default styling.

```go
cancelBtn := components.SecondaryButton("Cancel", func() {
    // Handle cancel
})
```

**When to use:**
- Secondary actions
- Most buttons
- Neutral actions

---

#### DangerButton
Button for destructive actions (red/warning styling).

```go
deleteBtn := components.DangerButton("Delete Group", func() {
    // Handle deletion
})
```

**When to use:**
- Destructive actions (delete, remove, clear)
- Actions that can't be undone
- Warning actions

---

#### IconButton
Button with an icon and text.

```go
import "fyne.io/fyne/v2/theme"

refreshBtn := components.IconButton("Refresh", theme.ViewRefreshIcon(), func() {
    // Handle refresh
})
```

**When to use:**
- Actions that benefit from visual icons
- Toolbar buttons
- Quick actions

---

### Custom Buttons

#### StackedButton
Button with main text and caption beneath it. Returns a `fyne.CanvasObject` (container).

```go
launchBtn := components.StackedButton(
    "Launch",           // Main text (larger, bold)
    "Start all bots",   // Caption text (smaller)
    func() {
        // Handle launch
    },
)
```

**When to use:**
- Complex actions that need explanation
- When you want to show both action and effect
- Multi-line button labels

**Example use cases:**
- "Start" / "Begin farming routine"
- "Launch" / "5 instances available"
- "Export" / "Download as CSV"

---

#### CustomButton
Advanced button with full control over text sizing and dual labels. This is a custom widget.

```go
btn := components.NewCustomButton(
    "Main Label",     // Main text
    18,              // Font size for main text
    "Small caption", // Caption text (optional, use "" for none)
    func() {
        // Handle tap
    },
)

// Update labels dynamically
btn.SetLabel("Updated Label")
btn.SetCaption("New caption")
```

**When to use:**
- Need programmatic label updates
- Specific font size requirements
- Dynamic button content

---

### Button Groups

#### ButtonGroup
Horizontal group of related buttons with consistent spacing.

```go
controls := components.ButtonGroup(
    startBtn,
    pauseBtn,
    stopBtn,
)
```

**When to use:**
- Related actions that should be grouped
- Toolbar-style button sets
- Action rows

---

#### ButtonGroupVertical
Vertical stack of related buttons.

```go
actions := components.ButtonGroupVertical(
    editBtn,
    duplicateBtn,
    deleteBtn,
)
```

**When to use:**
- Vertical action menus
- Sidebar actions
- Stacked controls

---

## Complete Examples

### Example 1: Page Header

```go
import "jordanella.com/pocket-tcg-go/internal/gui/components"

func BuildPage() fyne.CanvasObject {
    // Main header
    header := components.Heading("Bot Management")

    // Description
    desc := components.Body(
        "Configure and monitor your bot instances. " +
        "Each bot can run independently or as part of a group.",
    )

    // Hint
    hint := components.Caption("Tip: Use groups to coordinate multiple bots")

    return container.NewVBox(
        header,
        desc,
        hint,
        widget.NewSeparator(),
        // ... rest of page
    )
}
```

---

### Example 2: Form with Styled Labels

```go
func BuildForm() fyne.CanvasObject {
    // Section header
    section := components.Subheading("Configuration")

    // Form fields with bold labels
    nameLabel := components.BoldText("Bot Name:")
    nameEntry := widget.NewEntry()

    pathLabel := components.BoldText("Routine Path:")
    pathEntry := widget.NewEntry()
    pathEntry.SetPlaceHolder("/path/to/routine.yaml")

    // Help text
    pathHelp := components.Caption("Relative to routines directory")

    return container.NewVBox(
        section,
        nameLabel,
        nameEntry,
        pathLabel,
        pathEntry,
        pathHelp,
    )
}
```

---

### Example 3: Action Buttons

```go
func BuildActions() fyne.CanvasObject {
    // Primary action
    startBtn := components.PrimaryButton("Start Bot", func() {
        // Start logic
    })

    // Secondary actions
    pauseBtn := components.SecondaryButton("Pause", func() {
        // Pause logic
    })

    stopBtn := components.SecondaryButton("Stop", func() {
        // Stop logic
    })

    // Destructive action
    deleteBtn := components.DangerButton("Delete", func() {
        // Show confirmation dialog first
        dialog.ShowConfirm("Confirm", "Delete this bot?", func(ok bool) {
            if ok {
                // Delete logic
            }
        }, window)
    })

    // Group them
    return components.ButtonGroup(startBtn, pauseBtn, stopBtn, deleteBtn)
}
```

---

### Example 4: Card with Mixed Typography

```go
func BuildCard() fyne.CanvasObject {
    // Card title
    title := components.Subheading("Premium Farming Group")

    // Status with different text sizes
    statusLabel := components.BoldText("Status:")
    statusValue := components.Body("Running")

    // Monospace for technical info
    instanceLabel := components.BoldText("Instance:")
    instanceValue := components.MonospaceText("emulator-5554")

    // Timestamp
    timestamp := components.Caption("Started: 2 hours ago")

    // Stacked action button
    actionBtn := components.StackedButton(
        "Add Instance",
        "Scale up capacity",
        func() {
            // Handle add
        },
    )

    return container.NewVBox(
        title,
        container.NewHBox(statusLabel, statusValue),
        container.NewHBox(instanceLabel, instanceValue),
        timestamp,
        actionBtn,
    )
}
```

---

### Example 5: Dialog with Consistent Styling

```go
func ShowCustomDialog(window fyne.Window) {
    // Dialog header
    header := components.Heading("Create New Group")

    // Instructions
    instructions := components.Body(
        "Enter the details for your new orchestration group. " +
        "All fields marked with * are required.",
    )

    // Form section
    formHeader := components.Subheading("Group Settings")

    nameLabel := components.BoldText("Group Name *")
    nameEntry := widget.NewEntry()
    nameHint := components.Caption("Must be unique")

    // Dialog content
    content := container.NewVBox(
        header,
        instructions,
        widget.NewSeparator(),
        formHeader,
        nameLabel,
        nameEntry,
        nameHint,
    )

    // Create dialog
    d := dialog.NewCustomConfirm(
        "Create Group",
        "Create",
        "Cancel",
        content,
        func(confirmed bool) {
            // Handle confirmation
        },
        window,
    )

    d.Show()
}
```

---

## Typography Scale Reference

| Component | Size Name | Typical Size | Bold | Use Case |
|-----------|-----------|--------------|------|----------|
| Heading | `SizeNameHeadingText` | ~24px | Yes | Page titles |
| Subheading | `SizeNameSubHeadingText` | ~18px | Yes | Section headers |
| Body | `SizeNameText` | ~14px | No | Content |
| Caption | `SizeNameCaptionText` | ~12px | No | Secondary info |

---

## Best Practices

### 1. Be Consistent
Always use the same component for the same purpose:
- ✅ Always use `Heading()` for page titles
- ❌ Don't mix `Heading()`, `Subheading()`, and custom sizes for titles

### 2. Use Semantic Components
Choose components based on meaning, not just appearance:
- ✅ Use `Caption()` for timestamps because they're secondary info
- ❌ Don't use `Caption()` just because you want small text

### 3. Leverage the Type System
The components return different types for different use cases:
- `*widget.Label` - For standard text with wrapping
- `*widget.RichText` - For styled text with theme support
- `*canvas.Text` - For precise control (use sparingly)

### 4. Button Importance Hierarchy
On any screen, you should typically have:
- **One** `PrimaryButton` (the main action)
- **Several** `SecondaryButton` (supporting actions)
- **Zero or one** `DangerButton` (destructive action)

### 5. Don't Reinvent
If a preset component fits your needs, use it:
- ✅ `components.Heading("Title")`
- ❌ `components.SizedText("Title", 24, true)` (unless you need canvas.Text)

---

## Migration Guide

### From Old Code to New Components

#### Before:
```go
header := widget.NewLabel("My Title")
header.TextStyle = fyne.TextStyle{Bold: true}

desc := widget.NewLabel("Some description")
desc.Wrapping = fyne.TextWrapWord
```

#### After:
```go
header := components.Heading("My Title")
desc := components.Body("Some description")
```

---

#### Before:
```go
text := canvas.NewText("Custom Size", theme.ForegroundColor())
text.TextSize = 20
text.TextStyle.Bold = true
```

#### After:
```go
text := components.SizedText("Custom Size", 20, true)
```

---

#### Before:
```go
btn := widget.NewButton("Create", handleCreate)
btn.Importance = widget.HighImportance
```

#### After:
```go
btn := components.PrimaryButton("Create", handleCreate)
```

---

## Quick Reference

```go
// TEXT
components.Heading("Page Title")
components.Subheading("Section Title")
components.Body("Regular text")
components.Caption("Small hint")
components.BoldText("Emphasized")
components.MonospaceText("code/path")

// BUTTONS
components.PrimaryButton("Main Action", handler)
components.SecondaryButton("Other Action", handler)
components.DangerButton("Delete", handler)
components.StackedButton("Launch", "Start bots", handler)

// GROUPS
components.ButtonGroup(btn1, btn2, btn3)
components.ButtonGroupVertical(btn1, btn2, btn3)
```
