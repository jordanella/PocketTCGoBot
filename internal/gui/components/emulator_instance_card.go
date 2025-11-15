package components

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// EmulatorInstanceCardCallbacks defines callback functions for instance card actions
type EmulatorInstanceCardCallbacks struct {
	OnQuickStart func(instanceID int)
	OnPause      func(instanceID int)
	OnStop       func(instanceID int)
	OnAbort      func(instanceID int)
	OnShutdown   func(instanceID int)
	OnLaunch     func(instanceID int)
}

// InstanceState represents the state of an emulator instance
type InstanceState int

const (
	InstanceStateActive   InstanceState = iota // Running in a group
	InstanceStateIdle                          // Available but not running
	InstanceStateInactive                      // Not in any group
)

// EmulatorInstanceCardV2 represents an instance card component
type EmulatorInstanceCardV2 struct {
	// Data and state
	instanceID    int
	instanceName  string
	state         InstanceState
	accountName   string
	injectionTime time.Time
	routineStatus string
	groupNames    []string // Associated groups

	// Callbacks
	callbacks EmulatorInstanceCardCallbacks

	// UI elements that need dynamic updates
	container        *fyne.Container
	nameText         *canvas.Text
	accountText      *canvas.Text
	statusText       *canvas.Text
	groupsRow        *fyne.Container
	buttonRow        *fyne.Container
	pauseBtn         *widget.Button
	stopBtn          *widget.Button
	abortBtn         *widget.Button
	shutdownBtn      *widget.Button
	quickStartBtn    *widget.Button
	launchBtn        *widget.Button
}

// NewEmulatorInstanceCardV2 creates a new emulator instance card
func NewEmulatorInstanceCardV2(
	instanceID int,
	instanceName string,
	state InstanceState,
	callbacks EmulatorInstanceCardCallbacks,
) *EmulatorInstanceCardV2 {
	card := &EmulatorInstanceCardV2{
		instanceID:   instanceID,
		instanceName: instanceName,
		state:        state,
		callbacks:    callbacks,
	}

	card.container = card.build()
	card.UpdateFromState()

	return card
}

// build creates the card UI layout
func (c *EmulatorInstanceCardV2) build() *fyne.Container {
	// === HEADER ROW ===
	// Pattern: "Instance Name - Index <mumu index>                    [ buttons ]"
	nameLabel := Subheading(fmt.Sprintf("%s - %d", c.instanceName, c.instanceID))
	c.nameText = canvas.NewText(fmt.Sprintf("Index %d", c.instanceID), theme.Color(theme.ColorNameForeground))
	c.nameText.TextSize = 14
	headerLabels := InlineLabels(" ", nameLabel, c.nameText)

	// Create all buttons (will show/hide based on state)
	c.pauseBtn = SecondaryButton("Pause", func() {
		if c.callbacks.OnPause != nil {
			c.callbacks.OnPause(c.instanceID)
		}
	})

	c.stopBtn = SecondaryButton("Stop", func() {
		if c.callbacks.OnStop != nil {
			c.callbacks.OnStop(c.instanceID)
		}
	})

	c.abortBtn = SecondaryButton("Abort", func() {
		if c.callbacks.OnAbort != nil {
			c.callbacks.OnAbort(c.instanceID)
		}
	})

	c.shutdownBtn = DangerButton("Shutdown", func() {
		if c.callbacks.OnShutdown != nil {
			c.callbacks.OnShutdown(c.instanceID)
		}
	})

	c.quickStartBtn = PrimaryButton("Quick Start", func() {
		if c.callbacks.OnQuickStart != nil {
			c.callbacks.OnQuickStart(c.instanceID)
		}
	})

	c.launchBtn = SecondaryButton("Launch", func() {
		if c.callbacks.OnLaunch != nil {
			c.callbacks.OnLaunch(c.instanceID)
		}
	})

	c.buttonRow = container.NewHBox()
	headerRow := LabelButtonsRow(headerLabels, c.buttonRow)

	// === ACCOUNT INFO ===
	c.accountText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.accountText.TextSize = 14

	// === STATUS/GROUPS ROW ===
	// For active: shows status
	// For idle/inactive: shows associated groups
	c.statusText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.statusText.TextSize = 14
	statusLabel := container.NewHBox(BoldText("Status:"), c.statusText)

	c.groupsRow = container.NewHBox(BoldText("Associated:"))

	// === ASSEMBLE CARD CONTENT ===
	content := container.NewVBox(
		headerRow,
		c.accountText,
		statusLabel,
		c.groupsRow,
	)

	return CardWithIndent(content, 20) // Indented under parent group
}

// UpdateFromState refreshes the card based on current state
func (c *EmulatorInstanceCardV2) UpdateFromState() {
	// Update header
	c.nameText.Text = fmt.Sprintf("Index %d", c.instanceID)
	c.nameText.Refresh()

	// Update account info
	if c.accountName != "" {
		elapsed := time.Since(c.injectionTime)
		c.accountText.Text = fmt.Sprintf("Account %s since %s", c.accountName, formatDurationCompact(elapsed))
	} else {
		c.accountText.Text = "No account injected"
	}
	c.accountText.Refresh()

	// Update buttons and status based on state
	c.buttonRow.Objects = nil
	switch c.state {
	case InstanceStateActive:
		// Active: show Pause, Stop, Abort, Shutdown
		c.buttonRow.Add(c.pauseBtn)
		c.buttonRow.Add(c.stopBtn)
		c.buttonRow.Add(c.abortBtn)
		c.buttonRow.Add(c.shutdownBtn)

		// Show status
		c.statusText.Text = c.routineStatus
		c.statusText.Refresh()
		c.groupsRow.Hide()

	case InstanceStateIdle:
		// Idle: show Quick Start, Shutdown
		c.buttonRow.Add(c.quickStartBtn)
		c.buttonRow.Add(c.shutdownBtn)

		// Show associated groups
		c.statusText.Hide()
		c.groupsRow.Show()
		c.updateGroupsRow()

	case InstanceStateInactive:
		// Inactive: show Quick Start, Launch
		c.buttonRow.Add(c.quickStartBtn)
		c.buttonRow.Add(c.launchBtn)

		// Show associated groups
		c.statusText.Hide()
		c.groupsRow.Show()
		c.updateGroupsRow()
	}
	c.buttonRow.Refresh()
}

// updateGroupsRow rebuilds the groups row with chips
func (c *EmulatorInstanceCardV2) updateGroupsRow() {
	c.groupsRow.Objects = []fyne.CanvasObject{BoldText("Associated:")}

	if len(c.groupNames) > 0 {
		maxVisible := 3
		for i := 0; i < len(c.groupNames) && i < maxVisible; i++ {
			groupName := c.groupNames[i]
			c.groupsRow.Add(
				NavigationChip(groupName, func() {
					// Navigate to group
				}),
			)
		}
		if len(c.groupNames) > maxVisible {
			remaining := len(c.groupNames) - maxVisible
			c.groupsRow.Add(Caption(fmt.Sprintf("and %d more...", remaining)))
		}
	} else {
		c.groupsRow.Add(Caption("None"))
	}
	c.groupsRow.Refresh()
}

// SetAccount updates the account information
func (c *EmulatorInstanceCardV2) SetAccount(accountName string, injectionTime time.Time) {
	c.accountName = accountName
	c.injectionTime = injectionTime
	c.UpdateFromState()
}

// SetRoutineStatus updates the routine status (for active instances)
func (c *EmulatorInstanceCardV2) SetRoutineStatus(status string) {
	c.routineStatus = status
	c.UpdateFromState()
}

// SetAssociatedGroups updates the associated groups (for idle/inactive instances)
func (c *EmulatorInstanceCardV2) SetAssociatedGroups(groups []string) {
	c.groupNames = groups
	c.UpdateFromState()
}

// SetState changes the instance state
func (c *EmulatorInstanceCardV2) SetState(state InstanceState) {
	c.state = state
	c.UpdateFromState()
}

// GetContainer returns the Fyne container for embedding in layouts
func (c *EmulatorInstanceCardV2) GetContainer() *fyne.Container {
	return c.container
}

// GetInstanceID returns the instance ID
func (c *EmulatorInstanceCardV2) GetInstanceID() int {
	return c.instanceID
}

// formatDurationCompact formats a duration in compact form (e.g., "2h30m")
func formatDurationCompact(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
