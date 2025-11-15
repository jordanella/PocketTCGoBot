package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
)

// OrchestrationCard represents a card component for displaying orchestration group information
type OrchestrationCard struct {
	// Data and state
	data  *OrchestrationCardData
	group *bot.BotGroup

	// Callbacks for button actions
	onAddInstance func(*bot.BotGroup)
	onPauseResume func(*bot.BotGroup)
	onStop        func(*bot.BotGroup)
	onShutdown    func(*bot.BotGroup)

	// UI elements that need to be updated
	container      *fyne.Container
	statusIndicator *canvas.Circle
	pauseResumeBtn *widget.Button
}

// OrchestrationCardCallbacks holds callback functions for card actions
type OrchestrationCardCallbacks struct {
	OnAddInstance func(*bot.BotGroup)
	OnPauseResume func(*bot.BotGroup)
	OnStop        func(*bot.BotGroup)
	OnShutdown    func(*bot.BotGroup)
}

// NewOrchestrationCard creates a new orchestration card component
func NewOrchestrationCard(group *bot.BotGroup, callbacks OrchestrationCardCallbacks) *OrchestrationCard {
	card := &OrchestrationCard{
		group:         group,
		data:          NewOrchestrationCardData(group),
		onAddInstance: callbacks.OnAddInstance,
		onPauseResume: callbacks.OnPauseResume,
		onStop:        callbacks.OnStop,
		onShutdown:    callbacks.OnShutdown,
	}

	// Build the UI
	card.container = card.build()

	// Set up listeners for dynamic UI updates
	card.setupListeners()

	return card
}

// build creates the card UI layout matching the mockup
func (c *OrchestrationCard) build() *fyne.Container {
	// === HEADER ROW ===
	// Pattern: "Orchestration Group Name <orchestrationID>                    <active>"
	// Left: Group name + ID
	groupNameLabel := Subheading("")
	groupNameLabel.Segments[0].(*widget.TextSegment).Text = c.group.Name

	orchestrationID := Caption(fmt.Sprintf("<%s>", c.group.OrchestrationID[:8]))

	headerLabels := InlineLabels(" ", groupNameLabel, orchestrationID)

	// Right: Status indicator
	c.statusIndicator = canvas.NewCircle(color.RGBA{150, 150, 150, 255})
	c.statusIndicator.Resize(fyne.NewSize(12, 12))
	c.statusIndicator.StrokeWidth = 2
	c.statusIndicator.StrokeColor = color.RGBA{100, 100, 100, 255}

	statusLabel := widget.NewLabelWithData(c.data.StatusText)
	statusBox := container.NewHBox(c.statusIndicator, statusLabel)

	headerRow := LabelButtonsRow(headerLabels, statusBox)

	// === DESCRIPTION ===
	descLabel := widget.NewLabelWithData(c.data.Description)
	descLabel.Wrapping = fyne.TextWrapWord

	// === INFO ROW ===
	// Pattern: "Started <started at>   Pool Progress <remaining>/<total>"
	startedInfo := container.NewHBox(
		BoldText("Started:"),
		widget.NewLabelWithData(c.data.StartedAt),
	)

	poolProgressInfo := container.NewHBox(
		BoldText("Pool Progress:"),
		widget.NewLabelWithData(c.data.PoolProgress),
	)

	infoRow := InlineInfoRow(startedInfo, poolProgressInfo)

	// === CHIP ROWS ===
	// Pattern: "Account Pools <pool A> <pool B>"
	// For now using labels, but you can convert to chips when pools are a list
	accountPoolsRow := container.NewHBox(
		BoldText("Account Pools:"),
		widget.NewLabelWithData(c.data.AccountPoolNames),
	)

	// Pattern: "Active Instances <instance A> <instance B> <instance C>"
	activeInstancesRow := container.NewHBox(
		BoldText("Active Instances:"),
		widget.NewLabelWithData(c.data.ActiveInstancesList),
	)

	// Pattern: "Other Instances <instance A> <instance B> <instance C> and # more..."
	otherInstancesRow := container.NewHBox(
		BoldText("Other Instances:"),
		widget.NewLabelWithData(c.data.OtherInstancesList),
	)

	// === BUTTONS ===
	// Pattern: [ + Instance ] [ Pause/Resume ] [ Stop ] [ Shutdown ]
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
		descLabel,
		infoRow,
		accountPoolsRow,
		activeInstancesRow,
		otherInstancesRow,
		buttonRow,
	)

	// Use the new Card component
	return Card(content)
}

// setupListeners sets up data listeners for dynamic UI updates
func (c *OrchestrationCard) setupListeners() {
	// Update status indicator color when active state changes
	c.data.IsActive.AddListener(binding.NewDataListener(func() {
		active, _ := c.data.IsActive.Get()
		if active {
			// Green when active
			c.statusIndicator.FillColor = color.RGBA{76, 175, 80, 255}
			c.pauseResumeBtn.SetText("Pause")
		} else {
			// Gray when stopped
			c.statusIndicator.FillColor = color.RGBA{150, 150, 150, 255}
			c.pauseResumeBtn.SetText("Resume")
		}
		c.statusIndicator.Refresh()
	}))
}

// UpdateFromGroup refreshes all card data from the group state
// Call this periodically or when you know the group state has changed
func (c *OrchestrationCard) UpdateFromGroup() {
	c.data.UpdateFromGroup(c.group)
}

// GetContainer returns the Fyne container for embedding in layouts
func (c *OrchestrationCard) GetContainer() *fyne.Container {
	return c.container
}

// GetGroup returns the underlying BotGroup this card represents
func (c *OrchestrationCard) GetGroup() *bot.BotGroup {
	return c.group
}

// SetCallbacks updates the callback functions
func (c *OrchestrationCard) SetCallbacks(callbacks OrchestrationCardCallbacks) {
	c.onAddInstance = callbacks.OnAddInstance
	c.onPauseResume = callbacks.OnPauseResume
	c.onStop = callbacks.OnStop
	c.onShutdown = callbacks.OnShutdown
}
