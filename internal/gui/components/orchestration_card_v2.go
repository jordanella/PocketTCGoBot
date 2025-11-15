package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
)

// OrchestrationCardV2 represents a card component without data bindings
// This gives more granular control over UI updates
type OrchestrationCardV2 struct {
	// Data and state
	group *bot.BotGroup

	// Callbacks for button actions
	onAddInstance func(*bot.BotGroup)
	onPauseResume func(*bot.BotGroup)
	onStop        func(*bot.BotGroup)
	onShutdown    func(*bot.BotGroup)

	// UI elements that need dynamic updates
	container         *fyne.Container
	statusIndicator   *canvas.Circle
	statusText        *canvas.Text
	descriptionText   *canvas.Text
	startedText       *canvas.Text
	poolProgressText  *canvas.Text
	accountPoolsRow   *fyne.Container
	activeInstanceRow *fyne.Container
	otherInstanceRow  *fyne.Container
	pauseResumeBtn    *widget.Button
}

// NewOrchestrationCardV2 creates a new orchestration card without bindings
func NewOrchestrationCardV2(group *bot.BotGroup, callbacks OrchestrationCardCallbacks) *OrchestrationCardV2 {
	card := &OrchestrationCardV2{
		group:         group,
		onAddInstance: callbacks.OnAddInstance,
		onPauseResume: callbacks.OnPauseResume,
		onStop:        callbacks.OnStop,
		onShutdown:    callbacks.OnShutdown,
	}

	// Build the UI
	card.container = card.build()

	// Initial update
	card.UpdateFromGroup()

	return card
}

// build creates the card UI layout matching the mockup
func (c *OrchestrationCardV2) build() *fyne.Container {
	// === HEADER ROW ===
	// Pattern: "Orchestration Group Name <orchestrationID>                    <active>"
	groupNameLabel := Subheading(c.group.Name)
	orchestrationID := Caption(fmt.Sprintf("<%s>", c.group.OrchestrationID[:8]))
	headerLabels := InlineLabels(" ", groupNameLabel, orchestrationID)

	// Status indicator (will be updated)
	c.statusIndicator = canvas.NewCircle(color.RGBA{150, 150, 150, 255})
	c.statusIndicator.Resize(fyne.NewSize(12, 12))
	c.statusIndicator.StrokeWidth = 2
	c.statusIndicator.StrokeColor = color.RGBA{100, 100, 100, 255}

	c.statusText = canvas.NewText("Stopped", theme.Color(theme.ColorNameForeground))
	c.statusText.TextSize = 14 // Match body text size
	statusBox := container.NewHBox(c.statusIndicator, c.statusText)

	headerRow := LabelButtonsRow(headerLabels, statusBox)

	// === DESCRIPTION ===
	c.descriptionText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.descriptionText.TextSize = 14 // Body text size

	// === INFO ROW ===
	c.startedText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.startedText.TextSize = 14 // Body text size

	c.poolProgressText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.poolProgressText.TextSize = 14 // Body text size

	startedInfo := container.NewHBox(BoldText("Started:"), c.startedText)
	poolProgressInfo := container.NewHBox(BoldText("Pool Progress:"), c.poolProgressText)
	infoRow := InlineInfoRow(startedInfo, poolProgressInfo)

	// === CHIP ROWS ===
	// These will be rebuilt on update for chip support
	c.accountPoolsRow = container.NewHBox(BoldText("Account Pools:"))
	c.activeInstanceRow = container.NewHBox(BoldText("Active Instances:"))
	c.otherInstanceRow = container.NewHBox(BoldText("Other Instances:"))

	// === BUTTONS ===
	addInstanceBtn := SecondaryButton("+ Instance", func() {
		if c.onAddInstance != nil {
			c.onAddInstance(c.group)
		}
	})

	c.pauseResumeBtn = SecondaryButton("Pause", func() {
		if c.onPauseResume != nil {
			c.onPauseResume(c.group)
		}
	})

	stopBtn := SecondaryButton("Stop", func() {
		if c.onStop != nil {
			c.onStop(c.group)
		}
	})

	shutdownBtn := DangerButton("Shutdown", func() {
		if c.onShutdown != nil {
			c.onShutdown(c.group)
		}
	})

	buttonRow := ButtonGroup(addInstanceBtn, c.pauseResumeBtn, stopBtn, shutdownBtn)

	// === ASSEMBLE CARD CONTENT ===
	content := container.NewVBox(
		headerRow,
		c.descriptionText,
		infoRow,
		c.accountPoolsRow,
		c.activeInstanceRow,
		c.otherInstanceRow,
		buttonRow,
	)

	return Card(content)
}

// UpdateFromGroup refreshes all card data from the group state
func (c *OrchestrationCardV2) UpdateFromGroup() {
	// Update status indicator and text
	isRunning := c.group.IsRunning()

	if isRunning {
		c.statusIndicator.FillColor = color.RGBA{76, 175, 80, 255} // Green
		c.statusText.Text = "Active"
		c.pauseResumeBtn.SetText("Pause")
	} else {
		c.statusIndicator.FillColor = color.RGBA{150, 150, 150, 255} // Gray
		c.statusText.Text = "Stopped"
		c.pauseResumeBtn.SetText("Resume")
	}
	c.statusIndicator.Refresh()
	c.statusText.Refresh()

	// Update description
	c.descriptionText.Text = fmt.Sprintf("Running routine: %s", c.group.RoutineName)
	c.descriptionText.Refresh()

	// Update started time (placeholder - you may want to track actual start time)
	c.startedText.Text = c.group.OrchestrationID[:8]
	c.startedText.Refresh()

	// Update pool progress
	if c.group.AccountPool != nil {
		stats := c.group.AccountPool.GetStats()
		c.poolProgressText.Text = fmt.Sprintf("%d/%d", stats.Available, stats.Total)
	} else {
		c.poolProgressText.Text = "N/A"
	}
	c.poolProgressText.Refresh()

	// Update account pools row
	c.accountPoolsRow.Objects = []fyne.CanvasObject{BoldText("Account Pools:")}
	if c.group.AccountPoolName != "" {
		// Can convert to chip later
		c.accountPoolsRow.Add(NavigationChip(c.group.AccountPoolName, func() {
			// Navigate to pool
		}))
	} else {
		c.accountPoolsRow.Add(Caption("No pool assigned"))
	}
	c.accountPoolsRow.Refresh()

	// Update active instances
	activeBots := c.group.GetAllBotInfo()
	activeInstances := make([]int, 0, len(activeBots))
	for id := range activeBots {
		activeInstances = append(activeInstances, id)
	}

	c.activeInstanceRow.Objects = []fyne.CanvasObject{BoldText("Active Instances:")}
	if len(activeInstances) > 0 {
		// Show first 3 as chips, then "and N more..."
		maxVisible := 3
		for i := 0; i < len(activeInstances) && i < maxVisible; i++ {
			instanceID := activeInstances[i]
			c.activeInstanceRow.Add(
				NavigationChip(fmt.Sprintf("Instance %d", instanceID), func() {
					// Navigate to instance
				}),
			)
		}
		if len(activeInstances) > maxVisible {
			remaining := len(activeInstances) - maxVisible
			c.activeInstanceRow.Add(Caption(fmt.Sprintf("and %d more...", remaining)))
		}
	} else {
		c.activeInstanceRow.Add(Caption("None"))
	}
	c.activeInstanceRow.Refresh()

	// Update other instances (available but not active)
	otherInstances := make([]int, 0)
	for _, availID := range c.group.AvailableInstances {
		isActive := false
		for _, activeID := range activeInstances {
			if availID == activeID {
				isActive = true
				break
			}
		}
		if !isActive {
			otherInstances = append(otherInstances, availID)
		}
	}

	c.otherInstanceRow.Objects = []fyne.CanvasObject{BoldText("Other Instances:")}
	if len(otherInstances) > 0 {
		// Show first 3 as chips, then "and N more..."
		maxVisible := 3
		for i := 0; i < len(otherInstances) && i < maxVisible; i++ {
			instanceID := otherInstances[i]
			c.otherInstanceRow.Add(
				Chip(fmt.Sprintf("Instance %d", instanceID), nil), // Not clickable
			)
		}
		if len(otherInstances) > maxVisible {
			remaining := len(otherInstances) - maxVisible
			c.otherInstanceRow.Add(Caption(fmt.Sprintf("and %d more...", remaining)))
		}
	} else {
		c.otherInstanceRow.Add(Caption("None"))
	}
	c.otherInstanceRow.Refresh()
}

// GetContainer returns the Fyne container for embedding in layouts
func (c *OrchestrationCardV2) GetContainer() *fyne.Container {
	return c.container
}

// GetGroup returns the underlying BotGroup this card represents
func (c *OrchestrationCardV2) GetGroup() *bot.BotGroup {
	return c.group
}

// SetCallbacks updates the callback functions
func (c *OrchestrationCardV2) SetCallbacks(callbacks OrchestrationCardCallbacks) {
	c.onAddInstance = callbacks.OnAddInstance
	c.onPauseResume = callbacks.OnPauseResume
	c.onStop = callbacks.OnStop
	c.onShutdown = callbacks.OnShutdown
}
