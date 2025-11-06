package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/accounts"
	"jordanella.com/pocket-tcg-go/internal/emulator"
)

// DashboardTab displays bot instance status
type DashboardTab struct {
	controller *Controller

	// Widgets
	instanceCards *fyne.Container
	//refreshTimer  *time.Ticker
	stopRefresh chan bool
}

// NewDashboardTab creates a new dashboard tab
func NewDashboardTab(ctrl *Controller) *DashboardTab {
	return &DashboardTab{
		controller:  ctrl,
		stopRefresh: make(chan bool),
	}
}

// Build constructs the dashboard UI
func (d *DashboardTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("System Overview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Instance cards container
	d.instanceCards = container.NewVBox()
	d.updateInstanceCards()

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", func() {
		d.controller.RefreshMuMuInstances()
		d.updateInstanceCards()
	})

	// Auto-refresh every 2 seconds
	go d.autoRefresh()

	// Build content sections
	mumuSection := widget.NewLabelWithStyle("MuMu Instances", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	mumuCards := d.buildMuMuInstancesSection()

	botSection := widget.NewLabelWithStyle("Running Bots", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Scrollable content
	content := container.NewVScroll(
		container.NewVBox(
			mumuSection,
			mumuCards,
			widget.NewSeparator(),
			botSection,
			d.instanceCards,
		),
	)

	return container.NewBorder(
		container.NewVBox(header, refreshBtn),
		nil,
		nil,
		nil,
		content,
	)
}

// buildMuMuInstancesSection creates the MuMu instances display
func (d *DashboardTab) buildMuMuInstancesSection() fyne.CanvasObject {
	instances := d.controller.GetMuMuInstances()

	if len(instances) == 0 {
		return widget.NewLabel("No MuMu instances detected")
	}

	// Use grid layout for compact cards
	cards := container.NewGridWithColumns(2)
	for _, inst := range instances {
		card := d.createMuMuInstanceCard(inst)
		cards.Add(card)
	}

	return cards
}

// createMuMuInstanceCard creates a card for a MuMu instance
func (d *DashboardTab) createMuMuInstanceCard(inst *emulator.MuMuInstance) fyne.CanvasObject {
	// Title with version - use window title
	//versionStr := "?"
	//switch inst.Version {
	//case emulator.MuMuV5:
	//	versionStr = "v5"
	//case emulator.MuMuV12:
	//	versionStr = "v12"
	//}

	// Display window title with version
	titleText := inst.WindowTitle
	if titleText == "" {
		titleText = fmt.Sprintf("Instance %d", inst.Index)
	}

	title := widget.NewLabelWithStyle(
		titleText,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	// Compact info
	info := widget.NewLabel(fmt.Sprintf("Port: %d | %dx%d",
		inst.ADBPort, inst.Width, inst.Height))

	// Test ADB button
	testADBBtn := widget.NewButton("Test ADB", func() {
		d.testADBConnection(inst)
	})

	// Extract Account button
	extractAccountBtn := widget.NewButton("Extract Account", func() {
		destFile := fmt.Sprintf("account_%s.xml", inst.WindowTitle)
		err := accounts.ExtractAccount(d.controller.config.ADBPath, inst.ADBPort, destFile)
		if err != nil {
			d.controller.logTab.AddLog(LogLevelError, inst.Index, fmt.Sprintf("Failed to extract account: %v", err))
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Successfully extracted account from instance '%s' to %s.", inst.PlayerName, destFile), d.controller.window)
		}
	})

	buttonRow := container.NewGridWithColumns(2,
		testADBBtn,
		extractAccountBtn,
	)

	// Card with border for visual separation
	card := container.NewVBox(
		title,
		info,
		buttonRow,
	)

	// Add border/padding effect
	return container.NewPadded(card)
}

// updateInstanceCards refreshes the instance status display
func (d *DashboardTab) updateInstanceCards() {
	d.instanceCards.RemoveAll()

	bots := d.controller.GetAllBots()

	if len(bots) == 0 {
		d.instanceCards.Add(widget.NewLabel("No bot instances running"))
		d.instanceCards.Refresh()
		return
	}

	// Use grid layout for compact cards
	grid := container.NewGridWithColumns(2)
	for instance, bot := range bots {
		card := d.createInstanceCard(instance, bot)
		grid.Add(card)
	}

	d.instanceCards.Add(grid)
	d.instanceCards.Refresh()
}

// createInstanceCard creates a status card for a bot instance
func (d *DashboardTab) createInstanceCard(instance int, bot interface{}) fyne.CanvasObject {
	// Title
	titleLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("Bot Instance %d", instance),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	// Compact status info
	statusLabel := widget.NewLabel("Running | Screen: Unknown")

	// Actions
	stopBtn := widget.NewButton("Stop", func() {
		d.controller.RemoveBot(instance)
		d.updateInstanceCards()
	})

	viewLogsBtn := widget.NewButton("View Logs", func() {
		d.controller.switchTab(2)
	})

	actions := container.NewHBox(stopBtn, viewLogsBtn)

	// Compact card layout
	card := container.NewVBox(
		titleLabel,
		statusLabel,
		actions,
	)

	return container.NewPadded(card)
}

// autoRefresh updates the dashboard periodically
func (d *DashboardTab) autoRefresh() {
	// Don't auto-refresh for now to avoid threading issues
	// TODO: Implement proper goroutine-safe updates

	/* DISABLED - causes threading issues
	d.refreshTimer = time.NewTicker(2 * time.Second)
	defer d.refreshTimer.Stop()

	for {
		select {
		case <-d.refreshTimer.C:
			d.updateInstanceCards()
		case <-d.stopRefresh:
			return
		}
	}
	*/
}

// Shutdown stops the auto-refresh
func (d *DashboardTab) Shutdown() {
	close(d.stopRefresh)
}

// testADBConnection tests ADB connection to a specific MuMu instance
func (d *DashboardTab) testADBConnection(inst *emulator.MuMuInstance) {
	cfg := d.controller.GetConfig()

	// Check if ADB path is configured
	adbCfg := cfg.ADB()
	if adbCfg.Path == "" {
		d.controller.logTab.AddLog(LogLevelError, inst.Index, "ADB path not configured")
		return
	}

	d.controller.logTab.AddLog(LogLevelInfo, inst.Index, fmt.Sprintf("Testing ADB connection on port %d...", inst.ADBPort))

	// Create emulator manager
	mgr := emulator.NewManager(cfg.FolderPath, adbCfg.Path)

	// Discover instances (to populate manager state)
	if err := mgr.DiscoverInstances(); err != nil {
		d.controller.logTab.AddLog(LogLevelError, inst.Index, fmt.Sprintf("Failed to discover instances: %v", err))
		return
	}

	// Try to connect to this specific instance
	if err := mgr.ConnectInstance(inst.Index); err != nil {
		d.controller.logTab.AddLog(LogLevelError, inst.Index, fmt.Sprintf("ADB connection failed: %v", err))
		return
	}

	// Get the instance to check connection
	connectedInst, err := mgr.GetInstance(inst.Index)
	if err != nil {
		d.controller.logTab.AddLog(LogLevelError, inst.Index, fmt.Sprintf("Failed to get instance: %v", err))
		return
	}

	if connectedInst.IsConnected {
		d.controller.logTab.AddLog(LogLevelInfo, inst.Index, "ADB connection successful!")

		// Verify the ADB controller is working
		if connectedInst.ADB != nil && connectedInst.ADB.IsConnected() {
			d.controller.logTab.AddLog(LogLevelInfo, inst.Index, "ADB fully functional and ready to use")
			dialog.ShowInformation("Success", fmt.Sprintf("ADB connection to instance %s was successful and ready to use.", inst.PlayerName), d.controller.window)
		} else {
			d.controller.logTab.AddLog(LogLevelWarn, inst.Index, "ADB connected but controller status unclear")
			dialog.ShowInformation("Warning", fmt.Sprintf("ADB connection to instance %s was successful but status unclear.", inst.PlayerName), d.controller.window)
		}

		// Disconnect after test
		mgr.DisconnectInstance(inst.Index)
		d.controller.logTab.AddLog(LogLevelInfo, inst.Index, "ADB test complete, disconnected")

	} else {
		d.controller.logTab.AddLog(LogLevelError, inst.Index, "ADB connection status unclear")

		dialog.ShowInformation("Error", fmt.Sprintf("ADB connection to instance %s status could not be verified.", inst.PlayerName), d.controller.window)
	}
}
