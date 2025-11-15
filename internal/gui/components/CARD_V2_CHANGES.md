# OrchestrationCardV2 Changes

## What Changed

The `OrchestrationCardV2` removes data bindings in favor of direct widget updates, giving you more granular control over the UI.

## Key Differences

### Old (`OrchestrationCard`)
```go
// Used data bindings
descLabel := widget.NewLabelWithData(c.data.Description)

// Updates happened through binding.Set()
c.data.Description.Set("New description")
```

**Pros:**
- Automatic UI updates when bindings change
- Less code in UpdateFromGroup()

**Cons:**
- Less control over update timing
- Harder to customize update behavior
- Extra layer of abstraction

### New (`OrchestrationCardV2`)
```go
// Direct widget references
c.descriptionText = widget.NewLabel("")

// Updates happen with SetText()
c.descriptionText.SetText("New description")
```

**Pros:**
- ✅ Direct control over updates
- ✅ No binding overhead
- ✅ Easier to debug
- ✅ More flexible customization
- ✅ **Supports dynamic chip lists**

**Cons:**
- Must manually call SetText() / Refresh()

## Major Improvements

### 1. Dynamic Chip Lists

The V2 card now uses **actual chips** for instances instead of text:

```go
// Active instances as navigation chips
for i := 0; i < len(activeInstances) && i < maxVisible; i++ {
    instanceID := activeInstances[i]
    c.activeInstanceRow.Add(
        NavigationChip(fmt.Sprintf("Instance %d", instanceID), func() {
            // Navigate to instance
        }),
    )
}

// With "and N more..." truncation
if len(activeInstances) > maxVisible {
    remaining := len(activeInstances) - maxVisible
    c.activeInstanceRow.Add(Caption(fmt.Sprintf("and %d more...", remaining)))
}
```

### 2. Clickable Chips

**Account Pools:**
```go
// Now a clickable navigation chip!
c.accountPoolsRow.Add(NavigationChip(c.group.AccountPoolName, func() {
    // Navigate to pool view
}))
```

**Active Instances:**
- Clickable chips for first 3 instances
- "and N more..." for overflow

**Other Instances:**
- Non-clickable chips (visual only)
- "and N more..." for overflow

### 3. Better Update Control

**UpdateFromGroup()** now:
1. Updates status indicator and text
2. Updates description
3. Updates pool progress
4. **Rebuilds account pools row** with chips
5. **Rebuilds active instances row** with chips
6. **Rebuilds other instances row** with chips

Each row is completely rebuilt on update, allowing for dynamic content.

## Migration Guide

### Using the V2 Card

**In tabs/orchestration_v2.go:**
```go
// Already updated to use V2
card := components.NewOrchestrationCardV2(group, callbacks)
```

**No changes needed** - the API is identical:
- Same constructor signature
- Same `UpdateFromGroup()` method
- Same `GetContainer()` method
- Same `GetGroup()` method

### Customizing Chip Behavior

To add navigation handlers, update the chips in `UpdateFromGroup()`:

```go
// In orchestration_card_v2.go, line ~171
c.accountPoolsRow.Add(NavigationChip(c.group.AccountPoolName, func() {
    // TODO: Add your navigation logic here
    // Example: navigateToPoolView(c.group.AccountPoolName)
}))

// For instance chips, line ~188
c.activeInstanceRow.Add(
    NavigationChip(fmt.Sprintf("Instance %d", instanceID), func() {
        // TODO: Add your navigation logic here
        // Example: navigateToInstanceView(instanceID)
    }),
)
```

## Visual Comparison

### Old Card (Text-based)
```
┌────────────────────────────────────────────────┐
│ Premium Farmers <abc123>          Active       │
│ Running routine: farm_premium.yaml             │
│ Started: abc123   Pool Progress: 5/10          │
│ Account Pools: Premium Pool                    │
│ Active Instances: Instance 1, Instance 2       │
│ Other Instances: Instance 3, Instance 4        │
│ [+ Instance] [Pause] [Stop] [Shutdown]         │
└────────────────────────────────────────────────┘
```

### New Card (Chip-based)
```
┌────────────────────────────────────────────────┐
│ Premium Farmers <abc123>          ● Active     │
│ Running routine: farm_premium.yaml             │
│ Started: abc123   Pool Progress: 5/10          │
│ Account Pools: ┌─────────────┐                 │
│                │ Premium Pool│ (clickable)     │
│                └─────────────┘                 │
│ Active Instances: ┌────────┐ ┌────────┐        │
│                   │ Inst 1 │ │ Inst 2 │        │
│                   └────────┘ └────────┘        │
│ Other Instances: ┌────────┐ and 2 more...      │
│                  │ Inst 3 │                    │
│                  └────────┘                    │
│ [+ Instance] [Pause] [Stop] [Shutdown]         │
└────────────────────────────────────────────────┘
```

## Benefits for Future Development

### 1. Easy to Add Features

**Example: Add "Quick Launch" to Other Instances**
```go
// In UpdateFromGroup(), other instances section
for i := 0; i < len(otherInstances) && i < maxVisible; i++ {
    instanceID := otherInstances[i]

    // Create chip with custom content
    chipContent := container.NewVBox(
        widget.NewLabel(fmt.Sprintf("Instance %d", instanceID)),
        widget.NewButton("Quick Launch", func() {
            // Launch this specific instance
        }),
    )

    c.otherInstanceRow.Add(Chip(chipContent, nil))
}
```

### 2. Dynamic Content Based on State

```go
// Show different chips based on instance state
if instance.IsHealthy() {
    chip := NavigationChip("Instance 1", handler)
    chip.BackgroundColor = color.Green
} else {
    chip := NavigationChip("Instance 1 (!))", handler)
    chip.BackgroundColor = color.Yellow
}
```

### 3. Conditional Rows

```go
// Only show account pools if assigned
if c.group.AccountPoolName != "" {
    c.accountPoolsRow.Objects = []fyne.CanvasObject{
        BoldText("Account Pools:"),
        NavigationChip(c.group.AccountPoolName, navigateToPool),
    }
    c.accountPoolsRow.Show()
} else {
    c.accountPoolsRow.Hide()
}
```

## Performance Notes

### Old Card (with bindings)
- Every binding update triggers listener
- Computed bindings add overhead
- Hard to batch updates

### New Card (direct updates)
- Updates only when UpdateFromGroup() called
- Can batch multiple changes
- More predictable performance
- Rebuilding rows is fast (only done on data change)

## Code Locations

| File | Purpose |
|------|---------|
| [orchestration_card_v2.go](orchestration_card_v2.go) | New card implementation |
| [orchestration_card.go](orchestration_card.go) | Old card (still exists) |
| [orchestration_card_data.go](orchestration_card_data.go) | Old data bindings (still exists) |
| [tabs/orchestration_v2.go](../tabs/orchestration_v2.go) | Uses V2 card |
| [tabs/orchestration.go](../tabs/orchestration.go) | Uses old card |

Both versions coexist - you can use either or both.

## Future Enhancements

### 1. Add Pool Chips
When multiple pools are supported:
```go
// Multiple pool chips
for _, poolName := range c.group.AccountPoolNames {
    c.accountPoolsRow.Add(NavigationChip(poolName, navigateToPool))
}
```

### 2. Status Chips for Instances
```go
// Show instance status
chip := container.NewVBox(
    NavigationChip("Instance 1", handler),
    StatusChip("Running"),
)
```

### 3. Removable Chips
```go
// Allow removing instances
c.activeInstanceRow.Add(
    RemovableChip(fmt.Sprintf("Instance %d", instanceID), func() {
        removeInstance(instanceID)
    }),
)
```

## Testing

Both card versions work with the same data model:
```go
// Can test side-by-side
oldCard := components.NewOrchestrationCard(group, callbacks)
newCard := components.NewOrchestrationCardV2(group, callbacks)

// Both will show the same data, different presentation
```

## Summary

The V2 card provides:
- ✅ Direct control over UI updates
- ✅ Clickable chip support
- ✅ Dynamic content rebuilding
- ✅ Better match to mockup design
- ✅ Easier customization
- ✅ More maintainable code

Use V2 for new development. The old card remains for compatibility.
