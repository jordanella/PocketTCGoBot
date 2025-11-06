package gui

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/emulator"
)

// Controller manages the GUI state and bot instances
type Controller struct {
	config *bot.Config
	app    fyne.App
	window fyne.Window

	// Bot instances
	bots   map[int]*bot.Bot
	botsMu sync.RWMutex

	// MuMu instances (detected)
	mumuInstances   []*emulator.MuMuInstance
	mumuInstancesMu sync.RWMutex

	// GUI components
	dashboard  *DashboardTab
	configTab  *ConfigTab
	logTab     *LogTab
	accountTab *AccountTab
	resultsTab *ResultsTab
	controlTab *ControlTab
	adbTestTab *ADBTestTab

	// Content area reference for tab switching
	contentArea *fyne.Container

	// Current selected tab
	currentTab int
	mu         sync.RWMutex

	// Event bus for thread-safe UI updates
	eventBus *EventBus
}

// NewController creates a new GUI controller
func NewController(cfg *bot.Config, app fyne.App, window fyne.Window) *Controller {
	ctrl := &Controller{
		config:        cfg,
		app:           app,
		window:        window,
		bots:          make(map[int]*bot.Bot),
		mumuInstances: make([]*emulator.MuMuInstance, 0),
		currentTab:    0,
		eventBus:      NewEventBus(),
	}

	// Start event bus with app reference for main thread dispatch
	ctrl.eventBus.Start(app)

	// Initialize tabs
	ctrl.dashboard = NewDashboardTab(ctrl)
	ctrl.configTab = NewConfigTab(ctrl)
	ctrl.logTab = NewLogTab(ctrl)
	ctrl.accountTab = NewAccountTab(ctrl)
	ctrl.resultsTab = NewResultsTab(ctrl)
	ctrl.controlTab = NewControlTab(ctrl)
	ctrl.adbTestTab = NewADBTestTab(ctrl)

	// Subscribe event handlers
	ctrl.setupEventHandlers()

	// Detect MuMu instances on startup
	ctrl.RefreshMuMuInstances()

	return ctrl
}

// BuildUI constructs the main UI with horizontal tabs
func (c *Controller) BuildUI() fyne.CanvasObject {
	// Create tab buttons (horizontal navigation)
	tabButtons := container.NewHBox(
		widget.NewButton("Dashboard", func() { c.switchTab(0) }),
		widget.NewButton("Configuration", func() { c.switchTab(1) }),
		widget.NewButton("Event Log", func() { c.switchTab(2) }),
		widget.NewButton("Accounts", func() { c.switchTab(3) }),
		widget.NewButton("Results", func() { c.switchTab(4) }),
		widget.NewButton("Controls", func() { c.switchTab(5) }),
		widget.NewButton("ADB Test", func() { c.switchTab(6) }),
	)

	// Create content area (will switch based on selected tab)
	c.contentArea = container.NewStack(
		c.dashboard.Build(),
		c.configTab.Build(),
		c.logTab.Build(),
		c.accountTab.Build(),
		c.resultsTab.Build(),
		c.controlTab.Build(),
		c.adbTestTab.Build(),
	)

	// Initial state: show dashboard
	c.showTab(0, c.contentArea)

	// Main layout: tabs on top, content below
	return container.NewBorder(
		tabButtons,    // Top
		nil,           // Bottom
		nil,           // Left
		nil,           // Right
		c.contentArea, // Center
	)
}

// switchTab changes the active tab
func (c *Controller) switchTab(tabIndex int) {
	c.mu.Lock()
	c.currentTab = tabIndex
	contentArea := c.contentArea
	c.mu.Unlock()

	// Update visibility
	if contentArea != nil {
		c.showTab(tabIndex, contentArea)
	}
}

// showTab updates which tab content is visible
func (c *Controller) showTab(tabIndex int, contentArea *fyne.Container) {
	if contentArea == nil {
		// Can't refresh without content area reference
		// This should only happen during initial setup
		return
	}

	// Hide all tabs
	for i := 0; i < 7; i++ {
		if i < len(contentArea.Objects) {
			contentArea.Objects[i].Hide()
		}
	}

	// Show selected tab
	if tabIndex >= 0 && tabIndex < len(contentArea.Objects) {
		contentArea.Objects[tabIndex].Show()
	}

	contentArea.Refresh()
}

// GetConfig returns the current configuration
func (c *Controller) GetConfig() *bot.Config {
	return c.config
}

// UpdateConfig updates the configuration
func (c *Controller) UpdateConfig(cfg *bot.Config) {
	c.config = cfg
}

// GetBot returns a bot instance by ID
func (c *Controller) GetBot(instance int) (*bot.Bot, bool) {
	c.botsMu.RLock()
	defer c.botsMu.RUnlock()
	b, ok := c.bots[instance]
	return b, ok
}

// AddBot adds a bot instance
func (c *Controller) AddBot(instance int, b *bot.Bot) {
	c.botsMu.Lock()
	defer c.botsMu.Unlock()
	c.bots[instance] = b
}

// RemoveBot removes a bot instance
func (c *Controller) RemoveBot(instance int) {
	c.botsMu.Lock()
	defer c.botsMu.Unlock()
	if b, ok := c.bots[instance]; ok {
		b.Shutdown()
		delete(c.bots, instance)
	}
}

// GetAllBots returns all bot instances
func (c *Controller) GetAllBots() map[int]*bot.Bot {
	c.botsMu.RLock()
	defer c.botsMu.RUnlock()

	// Return copy to avoid race conditions
	bots := make(map[int]*bot.Bot, len(c.bots))
	for k, v := range c.bots {
		bots[k] = v
	}
	return bots
}

// Shutdown cleans up resources
func (c *Controller) Shutdown() {
	c.botsMu.Lock()
	defer c.botsMu.Unlock()

	for _, b := range c.bots {
		b.Shutdown()
	}
	c.bots = make(map[int]*bot.Bot)

	// Stop event bus
	if c.eventBus != nil {
		c.eventBus.Stop()
	}
}

// setupEventHandlers registers all event handlers
func (c *Controller) setupEventHandlers() {
	// Progress bar events
	c.eventBus.Subscribe(EventTypeProgressBarShow, func(e Event) {
		c.handleProgressBarEvent(e, true)
	})

	c.eventBus.Subscribe(EventTypeProgressBarHide, func(e Event) {
		c.handleProgressBarEvent(e, false)
	})

	// Label update events
	c.eventBus.Subscribe(EventTypeLabelUpdate, func(e Event) {
		c.handleLabelUpdate(e)
	})

	// Log events
	c.eventBus.Subscribe(EventTypeLogAdd, func(e Event) {
		c.handleLogEvent(e)
	})

	// Dialog events
	c.eventBus.Subscribe(EventTypeDialogError, func(e Event) {
		c.handleDialogError(e)
	})

	c.eventBus.Subscribe(EventTypeDialogInfo, func(e Event) {
		c.handleDialogInfo(e)
	})
}

// GetEventBus returns the event bus for publishing events
func (c *Controller) GetEventBus() *EventBus {
	return c.eventBus
}

// handleProgressBarEvent handles progress bar show/hide events
func (c *Controller) handleProgressBarEvent(e Event, show bool) {
	// Route to appropriate tab based on target
	switch e.Target {
	case "adbtest":
		if c.adbTestTab != nil && c.adbTestTab.progressBar != nil {
			if show {
				fyne.Do(func() {
					c.adbTestTab.progressBar.Show()
					c.adbTestTab.progressBar.Start()
				})
			} else {
				fyne.Do(func() {
					c.adbTestTab.progressBar.Stop()
					c.adbTestTab.progressBar.Hide()
				})
			}
			fyne.Do(func() {
				c.adbTestTab.progressBar.Refresh()
			})
		}
	}
}

// handleLabelUpdate handles label update events
func (c *Controller) handleLabelUpdate(e Event) {
	text, ok := e.Data["text"].(string)
	if !ok {
		return
	}

	// Route to appropriate widget based on target
	switch e.Target {
	case "adbtest.results":
		if c.adbTestTab != nil && c.adbTestTab.testResultsLabel != nil {
			fyne.Do(func() {
				c.adbTestTab.testResultsLabel.SetText(text)
				c.adbTestTab.testResultsLabel.Refresh()
			})
		}
	case "adbtest.path":
		if c.adbTestTab != nil && c.adbTestTab.adbPathLabel != nil {
			fyne.Do(func() {
				c.adbTestTab.adbPathLabel.SetText(text)
				c.adbTestTab.adbPathLabel.Refresh()
			})
		}
	case "adbtest.version":
		if c.adbTestTab != nil && c.adbTestTab.adbVersionLabel != nil {
			fyne.Do(func() {
				c.adbTestTab.adbVersionLabel.SetText(text)
				c.adbTestTab.adbVersionLabel.Refresh()
			})
		}
	case "adbtest.status":
		if c.adbTestTab != nil && c.adbTestTab.adbStatusLabel != nil {
			fyne.Do(func() {
				c.adbTestTab.adbStatusLabel.SetText(text)
				c.adbTestTab.adbStatusLabel.Refresh()
			})
		}
	case "adbtest.devices":
		if c.adbTestTab != nil && c.adbTestTab.devicesLabel != nil {
			fyne.Do(func() {
				c.adbTestTab.devicesLabel.SetText(text)
				c.adbTestTab.devicesLabel.Refresh()
			})
		}
	}
}

// handleLogEvent handles log add events
func (c *Controller) handleLogEvent(e Event) {
	level, ok := e.Data["level"].(LogLevel)
	if !ok {
		return
	}

	instance, ok := e.Data["instance"].(int)
	if !ok {
		return
	}

	message, ok := e.Data["message"].(string)
	if !ok {
		return
	}

	if c.logTab != nil {
		c.logTab.AddLog(level, instance, message)
	}
}

// handleDialogError handles error dialog events
func (c *Controller) handleDialogError(e Event) {
	message, ok := e.Data["message"].(string)
	if !ok {
		return
	}

	// Fyne dialogs are safe to call from any goroutine
	// because they queue themselves on the main thread
	dialog.ShowError(fmt.Errorf("%s", message), c.window)
}

// handleDialogInfo handles info dialog events
func (c *Controller) handleDialogInfo(e Event) {
	title, ok := e.Data["title"].(string)
	if !ok {
		return
	}

	message, ok := e.Data["message"].(string)
	if !ok {
		return
	}

	dialog.ShowInformation(title, message, c.window)
}

// RefreshMuMuInstances discovers running MuMu instances
func (c *Controller) RefreshMuMuInstances() {
	cfg := c.config
	adbPath := cfg.ADB().Path
	if adbPath == "" {
		adbPath = "dummy" // Don't need ADB for discovery
	}

	mgr := emulator.NewManager(cfg.FolderPath, adbPath)
	if err := mgr.DiscoverInstances(); err != nil {
		// Log error but don't fail
		if c.logTab != nil {
			c.logTab.AddLog(LogLevelWarn, 0, "Failed to discover MuMu instances")
		}
		return
	}

	instances := mgr.GetAllInstances()

	c.mumuInstancesMu.Lock()
	defer c.mumuInstancesMu.Unlock()

	// Extract MuMu instances
	c.mumuInstances = make([]*emulator.MuMuInstance, 0, len(instances))
	for _, inst := range instances {
		c.mumuInstances = append(c.mumuInstances, inst.MuMu)
	}
}

// GetMuMuInstances returns all detected MuMu instances
func (c *Controller) GetMuMuInstances() []*emulator.MuMuInstance {
	c.mumuInstancesMu.RLock()
	defer c.mumuInstancesMu.RUnlock()

	// Return copy to avoid race conditions
	instances := make([]*emulator.MuMuInstance, len(c.mumuInstances))
	copy(instances, c.mumuInstances)
	return instances
}

// GetEmulatorManager returns an emulator manager (creates on demand)
func (c *Controller) GetEmulatorManager() *emulator.Manager {
	return c.CreateEmulatorManager()
}

// CreateEmulatorManager creates a new emulator manager
func (c *Controller) CreateEmulatorManager() *emulator.Manager {
	cfg := c.GetConfig()
	adbPath := cfg.ADB().Path
	if adbPath == "" {
		adbPath = "dummy"
	}

	return emulator.NewManager(cfg.FolderPath, adbPath)
}
