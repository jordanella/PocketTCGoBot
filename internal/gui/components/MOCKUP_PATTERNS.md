# GUI Mockup Patterns Guide

This document explains how to implement the UI patterns from the mockups in `gui_mockups/` using the component library.

## Table of Contents
- [Chip Lists (Tags)](#chip-lists-tags)
- [Label-Buttons Rows](#label-buttons-rows)
- [Truncated Lists](#truncated-lists)
- [Two-Column Layouts](#two-column-layouts)
- [Conditional Display](#conditional-display)
- [Reorderable Lists](#reorderable-lists)
- [Field Inputs](#field-inputs)
- [Status Indicators](#status-indicators)
- [Table Layouts](#table-layouts)

---

## Chip Lists (Tags)

### Pattern from Mockups
```
Account Pools <pool A> <pool B>
Active Instances <instance A> <instance B> <instance C>
Tags <example> <event>
```

### Implementation

#### Basic Chip List
```go
// Simple chips (non-interactive)
chips := components.ChipList(
    components.Chip("pool A", nil),
    components.Chip("pool B", nil),
    components.Chip("pool C", nil),
)
```

#### Navigation Chips (Clickable)
```go
// Chips that navigate to other views
pools := []string{"Premium Pool", "Event Pool", "Testing Pool"}
chipList := components.NavigationChipList(pools, 5, func(poolName string) {
    // Navigate to pool view
    showPoolDetails(poolName)
})
```

#### Labeled Chip List
```go
// "Account Pools: <pool A> <pool B>"
row := components.LabeledNavigationChipList(
    "Account Pools",
    []string{"pool A", "pool B", "pool C"},
    3, // Show max 3
    navigateToPool,
)
```

#### Status Chips
```go
// Auto-colored status chips
statusChip := components.StatusChip("Active")  // Green
idleChip := components.StatusChip("Idle")      // Blue
errorChip := components.StatusChip("Error")    // Red
```

---

## Label-Buttons Rows

### Pattern from Mockups
```
|   Instance Name - Index <mumu index>    [ Pause ] [ Stop ] [ Abort ] [Shutdown] |
```

### Implementation

```go
// Labels on left (inline)
labels := components.InlineLabels(
    " - ",
    components.BoldText("Instance Name"),
    components.Body("Index 5"),
)

// Buttons on right
pauseBtn := components.SecondaryButton("Pause", handlePause)
stopBtn := components.SecondaryButton("Stop", handleStop)
abortBtn := components.DangerButton("Abort", handleAbort)
shutdownBtn := components.DangerButton("Shutdown", handleShutdown)

// Combine into row
row := components.LabelButtonsRow(labels, pauseBtn, stopBtn, abortBtn, shutdownBtn)
```

### With Multiple Label Lines
```go
// Line 1: Name and index with buttons
line1 := components.LabelButtonsRow(
    components.InlineLabels(" - ",
        components.BoldText("Instance Name"),
        components.Body("Index 5"),
    ),
    pauseBtn, stopBtn,
)

// Line 2: Additional info
line2 := components.Body("Account deviceAccount since 2 hours ago")

// Line 3: Status
line3 := components.InlineInfoRow(
    components.BoldText("Status:"),
    components.StatusChip("Active"),
)

// Combine in card
card := components.Card(
    container.NewVBox(line1, line2, line3),
)
```

---

## Truncated Lists

### Pattern from Mockups
```
Other Instances <instance A> <instance B> <instance C> and 5 more...
Associated <group A> <group B> <group C> and # more...
```

### Implementation

```go
instances := []string{
    "Instance 1", "Instance 2", "Instance 3",
    "Instance 4", "Instance 5", "Instance 6",
    "Instance 7", "Instance 8",
}

// Show first 3, then "and 5 more..."
truncated := components.TruncatedChipList(instances, 3, func(instance string) {
    // Handle chip click
})

// With label
labeled := components.LabeledChipList(
    "Other Instances",
    instances,
    3,
    handleInstanceClick,
)
```

---

## Two-Column Layouts

### Pattern from Mockups
```
|     Pool List          |   Pool Details (tabs)    |
|------------------------|---------------------------|
| Pool 1                 | Details | Accounts | ... |
| Pool 2                 |                           |
| Pool 3                 |                           |
```

### Implementation

```go
// Left: List of pools
poolList := container.NewVBox(
    components.Card(pool1Summary),
    components.Card(pool2Summary),
    components.Card(pool3Summary),
)
poolScroll := container.NewVScroll(poolList)

// Right: Details with tabs
detailsTabs := components.TabPanel(
    container.NewTabItem("Details", detailsView),
    container.NewTabItem("Accounts", accountsView),
    container.NewTabItem("Queries", queriesView),
)

// Combine with split
layout := components.TwoColumnLayout(poolScroll, detailsTabs, 250)
```

---

## Conditional Display

### Pattern from Mockups
```
[Save /Changes/] /[Save as Copy]/ /[Discard Changes]/ [Cancel]
```
The `/` slashes indicate conditional display (e.g., only show when there are changes).

### Implementation

```go
hasChanges := true // Your state variable

// Create buttons
saveBtn := components.PrimaryButton("Save Changes", handleSave)
saveCopyBtn := components.SecondaryButton("Save as Copy", handleSaveCopy)
discardBtn := components.SecondaryButton("Discard Changes", handleDiscard)
cancelBtn := components.SecondaryButton("Cancel", handleCancel)

// Conditionally include buttons
var buttons []fyne.CanvasObject

// Save button always visible
buttons = append(buttons, saveBtn)

// These only show when hasChanges is true
if hasChanges {
    buttons = append(buttons, saveCopyBtn)
    buttons = append(buttons, discardBtn)
}

// Cancel always visible
buttons = append(buttons, cancelBtn)

actionBar := components.ActionBarSingle(buttons...)
```

### Using ConditionalContainer
```go
actionBar := components.ConditionalContainer(
    saveBtn,
    conditionalWidget(hasChanges, saveCopyBtn),
    conditionalWidget(hasChanges, discardBtn),
    cancelBtn,
)

func conditionalWidget(show bool, widget fyne.CanvasObject) fyne.CanvasObject {
    if show {
        return widget
    }
    return nil
}
```

---

## Reorderable Lists

### Pattern from Mockups
```
{column ▼} {order ▼} [ ▲ ][ ▼ ][ Remove ][ Disable/Enable ]
Instance A [ ▲ ][ ▼ ][ Remove ][ Disable/Enable ]
```

### Implementation

```go
// Simple reorderable row
row := components.ReorderableRow(
    components.Body("Instance A"),
    handleMoveUp,
    handleMoveDown,
    handleRemove,
)

// With enable/disable toggle
row := components.ReorderableRowWithToggle(
    components.Body("Instance A"),
    true, // enabled state
    handleMoveUp,
    handleMoveDown,
    handleRemove,
    handleToggle,
)

// With dropdown fields
columnDropdown := widget.NewSelect(
    []string{"Name", "Status", "Created"},
    onColumnChange,
)
orderDropdown := widget.NewSelect(
    []string{"Ascending", "Descending"},
    onOrderChange,
)

row := components.ReorderableRow(
    container.NewHBox(columnDropdown, orderDropdown),
    handleMoveUp,
    handleMoveDown,
    handleRemove,
)
```

---

## Field Inputs

### Pattern from Mockups
```
Filename {filename.yaml}
Description {description text}
Variable {variable_name}* {{required}}
```

### Basic Fields
```go
// Simple field
filenameField := components.FieldRow(
    "Filename",
    widget.NewEntry(),
)

// Inline field
inlineField := components.FieldRowInline(
    "Limit",
    widget.NewEntry(),
)
```

### Required Fields
```go
// Field with asterisk and hint
nameField := components.RequiredFieldRow(
    "Bot Name",
    widget.NewEntry(),
    "Must be unique",
)
```

### Dropdown Fields
```go
// The ▼ triangle indicates a dropdown
columnDropdown := widget.NewSelect(
    []string{"Column 1", "Column 2", "Column 3"},
    func(selected string) {
        // Handle selection
    },
)

field := components.FieldRow("Column", columnDropdown)
```

---

## Status Indicators

### Pattern from Mockups
```
Status: Active - routine_label
deviceAccount | 12 | 86,500 | Active - routine_label
```

### Implementation

```go
// Simple status
status := components.StatusChip("Active")

// Status with additional info
statusRow := container.NewHBox(
    components.BoldText("Status:"),
    components.StatusChip("Active"),
    components.Body("-"),
    components.Body("routine_label"),
)

// In a table row
row := container.NewHBox(
    components.Body("deviceAccount"),
    components.Body("12"),
    components.Body("86,500"),
    container.NewHBox(
        components.StatusChip("Active"),
        components.Body("- routine_label"),
    ),
)
```

---

## Table Layouts

### Pattern from Mockups
```
Account           | Packs | Shinedust ▼ | Status
------------------------------------------------------------------------------
deviceAccount     |    12 |    86,500   | Active - routine_label  [ Details ]
deviceAccount     |    12 |    73,500   | Idle
```

### Implementation

```go
// Table header
header := components.TableHeader(
    "Account",
    "Packs",
    "Shinedust ▼", // ▼ indicates sortable
    "Status",
)

// Table rows
rows := []fyne.CanvasObject{
    header,
    widget.NewSeparator(),
}

accounts := getAccounts() // Your data
for _, account := range accounts {
    detailsBtn := components.SecondaryButton("Details", func() {
        showAccountDetails(account)
    })

    row := container.NewHBox(
        components.Body(account.Name),
        components.Body(fmt.Sprintf("%d", account.Packs)),
        components.Body(fmt.Sprintf("%s", formatNumber(account.Shinedust))),
        container.NewHBox(
            components.StatusChip(account.Status),
            components.Body(fmt.Sprintf("- %s", account.Routine)),
            detailsBtn,
        ),
    )
    rows = append(rows, row)
}

table := container.NewVBox(rows...)
```

### With Cards (More Visual)
```go
// Each row as a compact card
for _, account := range accounts {
    accountCard := components.CompactCard(
        container.NewVBox(
            container.NewHBox(
                components.BoldText(account.Name),
                components.Body(fmt.Sprintf("Packs: %d", account.Packs)),
                components.Body(fmt.Sprintf("Shinedust: %s", formatNumber(account.Shinedust))),
            ),
            container.NewHBox(
                components.StatusChip(account.Status),
                components.Caption(account.Routine),
            ),
        ),
    )
    rows = append(rows, accountCard)
}
```

---

## Complete Example: Orchestration Group Card

Combining multiple patterns from `emulator_instances.txt`:

```go
func BuildOrchestrationGroupCard(group *bot.BotGroup) fyne.CanvasObject {
    // Header row: name/ID with buttons
    headerRow := components.LabelButtonsRow(
        components.InlineLabels(" ",
            components.Subheading(group.Name),
            components.Caption(fmt.Sprintf("<%s>", group.OrchestrationID)),
        ),
        components.SecondaryButton("+ Instance", handleAddInstance),
    )

    // Description
    description := components.Body(group.Description)

    // Info row: started time and pool progress
    infoRow := components.InlineInfoRow(
        components.BoldText("Started:"),
        components.Caption(formatTime(group.StartedAt)),
        components.BoldText("Account Pool:"),
        components.NavigationChip(group.PoolName, navigateToPool),
        components.Caption(fmt.Sprintf("(%d/%d)", group.Remaining, group.Total)),
    )

    // Main card content
    mainCard := components.Card(
        container.NewVBox(
            headerRow,
            description,
            infoRow,
        ),
    )

    // Instance cards (indented)
    instances := container.NewVBox()
    for _, instance := range group.GetInstances() {
        instanceCard := BuildInstanceCard(instance)
        indented := components.CardWithIndent(instanceCard, 20)
        instances.Add(indented)
    }

    // Combine
    return container.NewVBox(
        mainCard,
        instances,
    )
}

func BuildInstanceCard(instance *Instance) fyne.CanvasObject {
    // Instance header with buttons
    header := components.LabelButtonsRow(
        components.InlineLabels(" - ",
            components.BoldText(instance.Name),
            components.Body(fmt.Sprintf("Index %d", instance.MumuIndex)),
        ),
        components.SecondaryButton("Pause", handlePause),
        components.SecondaryButton("Stop", handleStop),
        components.DangerButton("Abort", handleAbort),
        components.DangerButton("Shutdown", handleShutdown),
    )

    // Account info
    accountInfo := components.Caption(
        fmt.Sprintf("Account %s since %s", instance.Account, formatDuration(instance.InjectionTime)),
    )

    // Status
    statusRow := container.NewHBox(
        components.BoldText("Status:"),
        components.StatusChip(instance.Status),
    )

    return container.NewVBox(
        header,
        accountInfo,
        statusRow,
    )
}
```

---

## Complete Example: Account Pool View

Two-column layout from `account_pools.txt`:

```go
func BuildAccountPoolView() fyne.CanvasObject {
    // LEFT COLUMN: Pool list
    pools := []Pool{} // Your data
    poolCards := container.NewVBox()

    for _, pool := range pools {
        card := components.CompactCard(
            container.NewVBox(
                components.BoldText(pool.Name),
                components.Caption(fmt.Sprintf("<%s>", pool.Type)),
                components.Caption(fmt.Sprintf("%d accounts", pool.Count)),
                components.Caption(fmt.Sprintf("Updated: %s", pool.Updated)),
                components.Body(pool.Description),
            ),
        )
        poolCards.Add(card)
    }

    newPoolBtn := components.PrimaryButton("+ New Pool", handleNewPool)
    poolCards.Add(newPoolBtn)

    leftColumn := container.NewVScroll(poolCards)

    // RIGHT COLUMN: Pool details with tabs
    detailsTab := buildPoolDetailsTab()
    accountsTab := buildPoolAccountsTab()
    queriesTab := buildPoolQueriesTab()
    includeTab := buildPoolIncludeTab()
    excludeTab := buildPoolExcludeTab()

    rightColumn := components.TabPanel(
        container.NewTabItem("Details", detailsTab),
        container.NewTabItem("Accounts", accountsTab),
        container.NewTabItem("Queries", queriesTab),
        container.NewTabItem("Include", includeTab),
        container.NewTabItem("Exclude", excludeTab),
    )

    // Combine with split
    return components.TwoColumnLayout(leftColumn, rightColumn, 250)
}

func buildPoolDetailsTab() fyne.CanvasObject {
    // Description
    descEdit := components.FieldRow(
        "Description",
        widget.NewMultiLineEntry(),
    )

    // Total accounts with refresh
    totalRow := container.NewHBox(
        components.BoldText("Total Accounts:"),
        components.Body("127"),
        components.Caption("(last updated 2024-01-01 12:00:00)"),
        components.SecondaryButton("Refresh", handleRefresh),
    )

    // Queries section
    queriesSection := components.CardSection(
        "Queries",
        container.NewVBox(
            buildQueryRow("query1"),
            buildQueryRow("query2"),
            components.PrimaryButton("+ Query", handleAddQuery),
        ),
    )

    // Inclusions/Exclusions
    inclusionsRow := container.NewHBox(
        components.BoldText("Inclusions:"),
        components.Body("15"),
        components.SecondaryButton("Edit", handleEditInclusions),
    )
    exclusionsRow := container.NewHBox(
        components.BoldText("Exclusions:"),
        components.Body("3"),
        components.SecondaryButton("Edit", handleEditExclusions),
    )

    // Action buttons
    actions := components.ActionBarSingle(
        components.PrimaryButton("Save Changes", handleSave),
        components.SecondaryButton("Discard Changes", handleDiscard),
        components.DangerButton("Delete Pool", handleDelete),
    )

    return container.NewVBox(
        descEdit,
        totalRow,
        widget.NewSeparator(),
        queriesSection,
        inclusionsRow,
        exclusionsRow,
        widget.NewSeparator(),
        actions,
    )
}

func buildQueryRow(queryName string) fyne.CanvasObject {
    return components.ReorderableRow(
        components.Body(queryName),
        handleMoveUp,
        handleMoveDown,
        handleRemoveQuery,
    )
}
```

---

## Quick Pattern Reference

| Mockup Pattern | Component Function |
|---------------|-------------------|
| `<pool A> <pool B>` | `NavigationChipList()` |
| `Status: Active` | `StatusChip()` |
| `Label - Value` | `InlineLabels()` |
| `Label: [ Btn1 ] [ Btn2 ]` | `LabelButtonsRow()` |
| `and 5 more...` | `TruncatedChipList()` |
| `{field}` | `FieldRow()` |
| `{field}*{{required}}` | `RequiredFieldRow()` |
| `[ ▲ ][ ▼ ][ Remove ]` | `ReorderableRow()` |
| `/[Button]/` | Conditional rendering |
| `Column ▼` | `widget.NewSelect()` |
| Two columns | `TwoColumnLayout()` |
| Tabs | `TabPanel()` |
| Indented card | `CardWithIndent()` |

---

## Notes on Mockup Notation

- `<value>` - Dynamic data placeholder
- `{}` - Input field
- `▼` - Dropdown indicator
- `/text/` - Conditionally displayed
- `*` - Required field indicator
- `{{hint}}` - Tooltip/hint text
- `[ Button ]` - Clickable button
- `@modal` - Opens modal dialog
- Indentation - Visual hierarchy (use `CardWithIndent()`)
