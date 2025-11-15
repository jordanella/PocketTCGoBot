# Orchestration Tab Upgrade Guide

The orchestration tab has been upgraded to match the GUI mockup design using the new component library.

## What Changed

### Old Version (`orchestration.go`)
- Single scrolling list of all groups
- Basic card layout with limited styling
- Manual UI construction

### New Version (`orchestration_v2.go`)
- **Separated Active/Inactive sections** matching mockup
- **Uses new component library** for consistent styling
- **Better visual hierarchy** with proper typography
- **Cleaner code** with reusable components

## Key Improvements

### 1. Active/Inactive Sections

**Mockup Pattern:**
```
Active Groups
  ┌─ Group 1 (running) ─┐
  └─────────────────────┘

Inactive Groups
  ┌─ Group 2 (stopped) ─┐
  └─────────────────────┘
```

**Implementation:**
```go
activeSection := components.SectionHeader("Active Groups")
t.activeContainer // Cards for running groups

inactiveSection := components.SectionHeader("Inactive Groups")
t.inactiveContainer // Cards for stopped groups
```

Groups automatically move between sections when started/stopped.

### 2. Updated Card Layout

**Old Card:**
- Basic labels and buttons
- Manual background/border rendering
- Inconsistent spacing

**New Card:**
- Uses `LabelButtonsRow()` for header
- Uses `InlineLabels()` for name + ID
- Uses `BoldText()` for labels
- Uses component buttons (`SecondaryButton`, `DangerButton`)
- Uses `Card()` component for automatic styling

**Example:**
```go
// Header with name + ID on left, status on right
headerRow := LabelButtonsRow(
    InlineLabels(" ", groupName, orchestrationID),
    statusIndicator,
)

// Info row with multiple pieces of data
infoRow := InlineInfoRow(startedInfo, poolProgressInfo)

// Buttons using component library
buttonRow := ButtonGroup(
    SecondaryButton("+ Instance", handler),
    SecondaryButton("Pause", handler),
    SecondaryButton("Stop", handler),
    DangerButton("Shutdown", handler),
)
```

### 3. Better Form Layout

**Old Create Dialog:**
```go
widget.NewLabel("Group Name:")
nameEntry
widget.NewLabel("Routine:")
routineEntry
```

**New Create Dialog:**
```go
components.RequiredFieldRow("Group Name", nameEntry, "Must be unique")
components.RequiredFieldRow("Routine", routineEntry, "Routine filename")
components.FieldRow("Account Pool Name", poolEntry)
```

Benefits:
- Required fields marked with *
- Consistent spacing
- Hints displayed automatically

## Migration Steps

### Option 1: Switch to V2 (Recommended)

1. **Update controller.go** to use new tab:
```go
// Old
orchTab := tabs.NewOrchestrationTab(orchestrator, window)

// New
orchTab := tabs.NewOrchestrationTabV2(orchestrator, window)
```

2. **Test thoroughly** - behavior should be identical, just better organized

### Option 2: Gradual Migration

Keep both versions during transition:
- Use V2 for new development
- Keep V1 for stability
- Compare side-by-side in testing

## API Changes

### Constructor
```go
// Same signature, different implementation
NewOrchestrationTabV2(orchestrator, window) *OrchestrationTabV2
```

### Methods (Unchanged)
- `Build()` - Returns `fyne.CanvasObject`
- `Stop()` - Stops periodic refresh

### Internal Changes
- Added `reorganizeCards()` - moves cards between active/inactive
- Added `activeContainer` and `inactiveContainer` - separate sections
- Improved `refreshAllCards()` - also reorganizes periodically

## Component Usage Examples

### Creating Section Headers
```go
header := components.SectionHeader("Active Groups")
// Or with actions:
header := components.SectionHeader("Settings", editBtn, deleteBtn)
```

### Using Typography Components
```go
title := components.Heading("Page Title")        // Large, bold
section := components.Subheading("Section")      // Medium, bold
body := components.Body("Description text")      // Standard
detail := components.Caption("Last updated...")  // Small
```

### Button Patterns
```go
// Primary action (highlighted)
createBtn := components.PrimaryButton("Create", handler)

// Standard actions
pauseBtn := components.SecondaryButton("Pause", handler)

// Destructive actions (red)
deleteBtn := components.DangerButton("Delete", handler)

// Group related buttons
actions := components.ButtonGroup(btn1, btn2, btn3)
```

### Field Rows
```go
// Simple field
nameField := components.FieldRow("Name", widget.NewEntry())

// Required field with hint
emailField := components.RequiredFieldRow(
    "Email",
    widget.NewEntry(),
    "Must be valid email address",
)

// Inline field (label and input side-by-side)
limitField := components.FieldRowInline("Limit", widget.NewEntry())
```

## Future Enhancements

The V2 version is designed to support future mockup features:

### 1. Chip Lists for Pools/Instances
Currently using labels, can easily convert to chips:
```go
// Current
container.NewHBox(
    BoldText("Account Pools:"),
    widget.NewLabelWithData(c.data.AccountPoolNames),
)

// Future with chips
poolNames := []string{"Premium", "Event", "Testing"}
components.LabeledNavigationChipList(
    "Account Pools",
    poolNames,
    3,
    navigateToPool,
)
```

### 2. Configure Modal
Ready for the "Configure" button pattern:
```go
configureBtn := SecondaryButton("Configure", func() {
    showGroupConfigModal(group)
})
```

### 3. Launch Button for Inactive Groups
Already in mockup, easy to add:
```go
// In card build(), check if inactive:
if !group.IsRunning() {
    launchBtn := PrimaryButton("Launch", handler)
    configureBtn := SecondaryButton("Configure", handler)
    buttonRow := ButtonGroup(launchBtn, configureBtn)
}
```

## Testing Checklist

- [ ] Groups appear in correct section (active vs inactive)
- [ ] Cards move when started/stopped
- [ ] Create dialog works with new field components
- [ ] All buttons function correctly
- [ ] Periodic refresh updates cards
- [ ] Shutdown removes cards properly
- [ ] Status label shows correct count
- [ ] Typography is consistent and readable
- [ ] Spacing and alignment match mockup

## Rollback Plan

If issues arise, reverting is simple:

1. Change controller to use old tab:
```go
orchTab := tabs.NewOrchestrationTab(orchestrator, window)
```

2. Both versions coexist in the same package
3. No data model changes - both use same `Orchestrator` API

## Questions?

See:
- [Component Documentation](../components/README.md)
- [Mockup Patterns Guide](../components/MOCKUP_PATTERNS.md)
- [Quick Reference](../components/QUICK_REFERENCE.md)
