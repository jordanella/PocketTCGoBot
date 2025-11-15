# Component Examples - Complete Layouts

This document shows complete examples combining text, button, and card components to build real-world UIs.

## Example 1: Simple Settings Page

```go
package main

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "jordanella.com/pocket-tcg-go/internal/gui/components"
)

func BuildSimpleSettingsPage() fyne.CanvasObject {
    // Page header
    header := components.Heading("Settings")
    description := components.Body("Configure your bot preferences")

    // General settings card
    generalCard := components.CardSection(
        "General",
        container.NewVBox(
            widget.NewCheck("Enable notifications", nil),
            widget.NewCheck("Auto-start on launch", nil),
            widget.NewCheck("Minimize to tray", nil),
        ),
    )

    // Actions
    saveBtn := components.PrimaryButton("Save Changes", func() {
        // Save logic
    })
    resetBtn := components.SecondaryButton("Reset to Defaults", func() {
        // Reset logic
    })

    return container.NewVBox(
        header,
        description,
        widget.NewSeparator(),
        generalCard,
        widget.NewSeparator(),
        components.ButtonGroup(saveBtn, resetBtn),
    )
}
```

---

## Example 2: Hierarchical Configuration

```go
func BuildHierarchicalConfig() fyne.CanvasObject {
    // Top-level account settings
    accountCard := components.Card(
        container.NewVBox(
            components.Subheading("Account Settings"),
            components.Body("Manage your account preferences"),
        ),
    )

    // Profile settings (indented under account)
    profileCard := components.CardWithIndent(
        container.NewVBox(
            components.BoldText("Profile"),
            components.Body("Update your profile information"),
            widget.NewEntry(),
        ),
        20, // First level indent
    )

    // Privacy settings (indented under profile)
    privacyCard := components.CardWithIndent(
        container.NewVBox(
            components.BoldText("Privacy"),
            widget.NewCheck("Public profile", nil),
            widget.NewCheck("Show email", nil),
        ),
        40, // Second level indent
    )

    // Security settings (back to first level)
    securityCard := components.CardWithIndent(
        container.NewVBox(
            components.BoldText("Security"),
            widget.NewButton("Change Password", func() {}),
            widget.NewButton("Two-Factor Auth", func() {}),
        ),
        20, // First level indent
    )

    return container.NewVBox(
        components.Heading("Account Management"),
        accountCard,
        profileCard,
        privacyCard,
        securityCard,
    )
}
```

---

## Example 3: Dashboard with Status Cards

```go
func BuildDashboard() fyne.CanvasObject {
    // Page header
    header := components.Heading("Bot Dashboard")
    subtitle := components.Caption("Monitor your bot activity")

    // Status card 1
    activeBotsCard := components.Card(
        container.NewVBox(
            components.Subheading("Active Bots"),
            container.NewHBox(
                components.SizedText("12", 32, true),
                components.Caption("currently running"),
            ),
        ),
    )

    // Status card 2
    completedCard := components.Card(
        container.NewVBox(
            components.Subheading("Tasks Completed"),
            container.NewHBox(
                components.SizedText("847", 32, true),
                components.Caption("today"),
            ),
        ),
    )

    // Status card 3
    errorCard := components.Card(
        container.NewVBox(
            components.Subheading("Errors"),
            container.NewHBox(
                components.SizedText("3", 32, true),
                components.Caption("in last hour"),
            ),
            components.PrimaryButton("View Logs", func() {}),
        ),
    )

    // Layout in a grid-like structure
    statusRow := container.NewHBox(
        activeBotsCard,
        completedCard,
        errorCard,
    )

    return container.NewVBox(
        header,
        subtitle,
        widget.NewSeparator(),
        statusRow,
    )
}
```

---

## Example 4: Form with Validation Hints

```go
func BuildForm() fyne.CanvasObject {
    header := components.Heading("Create New Bot")
    description := components.Body("Fill in the details to create a new bot instance")

    // Bot name field
    nameLabel := components.BoldText("Bot Name *")
    nameEntry := widget.NewEntry()
    nameEntry.SetPlaceHolder("e.g., Premium Farmer 1")
    nameHint := components.Caption("Must be unique")

    // Routine path field
    routineLabel := components.BoldText("Routine Path *")
    routineEntry := widget.NewEntry()
    routineEntry.SetPlaceHolder("/routines/farming/premium.yaml")
    routinePath := components.MonospaceText("./routines/")

    // Instance ID field
    instanceLabel := components.BoldText("Emulator Instance")
    instanceEntry := widget.NewEntry()
    instanceEntry.SetPlaceHolder("1")
    instanceHint := components.Caption("Leave blank for auto-assign")

    // Wrap form in a card
    formCard := components.Card(
        container.NewVBox(
            nameLabel,
            nameEntry,
            nameHint,
            widget.NewSeparator(),
            routineLabel,
            routinePath,
            routineEntry,
            widget.NewSeparator(),
            instanceLabel,
            instanceEntry,
            instanceHint,
        ),
    )

    // Actions
    createBtn := components.PrimaryButton("Create Bot", func() {
        // Validation and creation logic
    })
    cancelBtn := components.SecondaryButton("Cancel", func() {
        // Cancel logic
    })

    return container.NewVBox(
        header,
        description,
        formCard,
        components.ButtonGroup(createBtn, cancelBtn),
    )
}
```

---

## Example 5: Activity Feed

```go
func BuildActivityFeed() fyne.CanvasObject {
    header := components.Heading("Recent Activity")

    // Activity items
    activities := []struct {
        title     string
        details   string
        timestamp string
    }{
        {"Bot Started", "Premium Farmer 1 started successfully", "2 minutes ago"},
        {"Routine Completed", "Daily farming routine finished", "15 minutes ago"},
        {"Error Occurred", "Connection timeout on Instance 3", "1 hour ago"},
        {"Account Added", "New account imported: user@example.com", "2 hours ago"},
    }

    // Build activity cards
    activityCards := container.NewVBox()
    for _, activity := range activities {
        card := components.CompactCard(
            container.NewVBox(
                components.BoldText(activity.title),
                components.Body(activity.details),
                components.Caption(activity.timestamp),
            ),
        )
        activityCards.Add(card)
    }

    // Scrollable container
    scroll := container.NewVScroll(activityCards)
    scroll.SetMinSize(fyne.NewSize(400, 500))

    return container.NewVBox(
        header,
        scroll,
    )
}
```

---

## Example 6: Bot Instance Card

```go
func BuildBotInstanceCard(botName string, status string, uptime string) fyne.CanvasObject {
    // Card header
    title := components.Subheading(botName)

    // Status info
    statusLabel := components.BoldText("Status:")
    statusValue := components.Body(status)

    // Uptime info
    uptimeLabel := components.BoldText("Uptime:")
    uptimeValue := components.MonospaceText(uptime)

    // Last activity
    lastActivity := components.Caption("Last activity: 30 seconds ago")

    // Action buttons
    pauseBtn := components.SecondaryButton("Pause", func() {})
    stopBtn := components.DangerButton("Stop", func() {})

    // Build card content
    content := container.NewVBox(
        title,
        container.NewHBox(statusLabel, statusValue),
        container.NewHBox(uptimeLabel, uptimeValue),
        lastActivity,
        widget.NewSeparator(),
        components.ButtonGroup(pauseBtn, stopBtn),
    )

    return components.Card(content)
}
```

---

## Example 7: Nested Settings with Sections

```go
func BuildNestedSettings() fyne.CanvasObject {
    header := components.Heading("Advanced Configuration")

    // Bot Settings (Level 0)
    botSection := components.NestedCard(
        components.Subheading("Bot Settings"),
        0,
    )

    // Retry Policy (Level 1)
    retrySection := components.NestedCard(
        container.NewVBox(
            components.BoldText("Retry Policy"),
            widget.NewCheck("Enable auto-retry", nil),
        ),
        1,
    )

    // Backoff Configuration (Level 2)
    backoffSection := components.NestedCard(
        container.NewVBox(
            components.BoldText("Backoff Settings"),
            components.Body("Configure retry backoff behavior"),
            widget.NewSlider(1, 10),
        ),
        2,
    )

    // Timeout Settings (Level 1)
    timeoutSection := components.NestedCard(
        container.NewVBox(
            components.BoldText("Timeout Settings"),
            components.Body("Configure operation timeouts"),
        ),
        1,
    )

    return container.NewVBox(
        header,
        botSection,
        retrySection,
        backoffSection,
        timeoutSection,
    )
}
```

---

## Example 8: Wizard/Multi-Step Form

```go
func BuildWizardStep(stepNumber int, stepName string) fyne.CanvasObject {
    // Step header
    stepTitle := components.Heading(fmt.Sprintf("Step %d: %s", stepNumber, stepName))

    // Step content (example for step 1)
    var stepContent fyne.CanvasObject
    if stepNumber == 1 {
        stepContent = components.Card(
            container.NewVBox(
                components.Subheading("Basic Information"),
                components.BoldText("Group Name"),
                widget.NewEntry(),
                components.BoldText("Description"),
                widget.NewEntry(),
            ),
        )
    } else if stepNumber == 2 {
        stepContent = components.Card(
            container.NewVBox(
                components.Subheading("Select Instances"),
                widget.NewCheck("Instance 1", nil),
                widget.NewCheck("Instance 2", nil),
                widget.NewCheck("Instance 3", nil),
            ),
        )
    }

    // Navigation buttons
    backBtn := components.SecondaryButton("Back", func() {})
    nextBtn := components.PrimaryButton("Next", func() {})

    return container.NewVBox(
        stepTitle,
        components.Caption(fmt.Sprintf("Complete step %d to continue", stepNumber)),
        widget.NewSeparator(),
        stepContent,
        widget.NewSeparator(),
        components.ButtonGroup(backBtn, nextBtn),
    )
}
```

---

## Example 9: Compact List View

```go
func BuildCompactList(items []string) fyne.CanvasObject {
    header := components.Heading("Quick Actions")

    list := container.NewVBox()
    for _, item := range items {
        card := components.CompactCard(
            container.NewHBox(
                components.Body(item),
                widget.NewButton("Go", func() {}),
            ),
        )
        list.Add(card)
    }

    return container.NewVBox(
        header,
        list,
    )
}

// Usage:
// actions := []string{"Start Bot", "View Logs", "Import Account", "Export Data"}
// listView := BuildCompactList(actions)
```

---

## Example 10: Full Page Layout

```go
func BuildFullPage() fyne.CanvasObject {
    // Header section
    header := container.NewVBox(
        components.Heading("Bot Orchestration"),
        components.Body("Manage your bot groups and instances"),
        components.Caption("All times shown in local timezone"),
        widget.NewSeparator(),
    )

    // Main content - 2 column layout
    leftColumn := container.NewVBox(
        components.CardSection(
            "Active Groups",
            container.NewVBox(
                components.Body("Premium Farmers - Running"),
                components.Body("Event Grinders - Paused"),
            ),
        ),
    )

    rightColumn := container.NewVBox(
        components.CardSection(
            "Quick Stats",
            container.NewVBox(
                components.Body("Total Bots: 12"),
                components.Body("Active: 8"),
                components.Body("Errors: 1"),
            ),
        ),
    )

    mainContent := container.NewHBox(
        leftColumn,
        rightColumn,
    )

    // Footer actions
    footer := container.NewVBox(
        widget.NewSeparator(),
        components.ButtonGroup(
            components.PrimaryButton("Create New Group", func() {}),
            components.SecondaryButton("Import Config", func() {}),
            components.SecondaryButton("Refresh", func() {}),
        ),
    )

    // Combine all sections
    return container.NewBorder(
        header,
        footer,
        nil, nil,
        mainContent,
    )
}
```

---

## Example 11: Error Display

```go
func BuildErrorDisplay(errorMsg string) fyne.CanvasObject {
    // Custom red-tinted card for errors
    padding := float32(12)
    radius := float32(6)
    errorBg := color.NRGBA{R: 255, G: 235, B: 235, A: 255} // Light red

    errorCard := components.CardWithOptions(
        container.NewVBox(
            components.Subheading("Error Occurred"),
            components.Body(errorMsg),
            components.Caption("Check logs for more details"),
            widget.NewSeparator(),
            components.PrimaryButton("View Logs", func() {}),
        ),
        components.CardOptions{
            PaddingOverride: &padding,
            CornerRadius:    &radius,
            BackgroundColor: &errorBg,
        },
    )

    return errorCard
}
```

---

## Example 12: Info Panel with Icons

```go
func BuildInfoPanel() fyne.CanvasObject {
    // Info sections
    infoItems := []struct {
        title string
        value string
    }{
        {"Bot Version", "v1.2.3"},
        {"Database", "Connected"},
        {"Emulators", "4 available"},
        {"Account Pool", "127 accounts"},
    }

    cards := container.NewVBox()
    for _, item := range infoItems {
        card := components.CompactCard(
            container.NewVBox(
                components.Caption(item.title),
                components.BoldText(item.value),
            ),
        )
        cards.Add(card)
    }

    return container.NewVBox(
        components.Heading("System Info"),
        cards,
    )
}
```

---

## Tips for Combining Components

### 1. Maintain Visual Hierarchy
```go
// ✅ Good: Clear hierarchy
components.Heading("Page")
components.Subheading("Section")
components.Body("Content")
components.Caption("Detail")

// ❌ Bad: Inconsistent hierarchy
components.Heading("Page")
components.Body("Section")  // Should be Subheading
components.Heading("Detail") // Should be Caption
```

### 2. Group Related Actions
```go
// ✅ Good: Related buttons grouped
components.ButtonGroup(
    components.PrimaryButton("Save", save),
    components.SecondaryButton("Cancel", cancel),
)

// ❌ Bad: Scattered buttons
container.NewVBox(
    components.PrimaryButton("Save", save),
    widget.NewLabel("Other content"),
    components.SecondaryButton("Cancel", cancel),
)
```

### 3. Use Cards for Grouping
```go
// ✅ Good: Related content in cards
components.Card(
    container.NewVBox(
        components.Subheading("Settings"),
        setting1,
        setting2,
        setting3,
    ),
)

// ❌ Bad: No visual grouping
container.NewVBox(
    components.Subheading("Settings"),
    setting1,
    setting2,
    setting3,
)
```

### 4. Consistent Spacing
```go
// ✅ Good: Separators between major sections
container.NewVBox(
    section1,
    widget.NewSeparator(),
    section2,
    widget.NewSeparator(),
    actions,
)
```

---

## Quick Component Selection Guide

| Need | Component |
|------|-----------|
| Page title | `components.Heading()` |
| Section title | `components.Subheading()` |
| Description text | `components.Body()` |
| Timestamp/hint | `components.Caption()` |
| File path | `components.MonospaceText()` |
| Primary action | `components.PrimaryButton()` |
| Secondary action | `components.SecondaryButton()` |
| Delete/Remove | `components.DangerButton()` |
| Group content | `components.Card()` |
| Nested content | `components.CardWithIndent()` |
| Titled section | `components.CardSection()` |
| Dense list | `components.CompactCard()` |
| Multi-level tree | `components.NestedCard()` |
