package gui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	_ "jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/cv"
	"jordanella.com/pocket-tcg-go/internal/emulator"
	_ "jordanella.com/pocket-tcg-go/pkg/templates"
)

// ControlTab provides bot control and management
type ControlTab struct {
	controller *Controller

	// Widgets
	instanceSelect     *widget.Select
	startBtn           *widget.Button
	stopBtn            *widget.Button
	pauseBtn           *widget.Button
	resumeBtn          *widget.Button
	statusLabel        *widget.Label
	instanceCountEntry *widget.Entry
	startAllBtn        *widget.Button
	stopAllBtn         *widget.Button

	// Instance mapping for dropdown
	instanceMap map[string]int // Maps display name to instance number
}

// NewControlTab creates a new control tab
func NewControlTab(ctrl *Controller) *ControlTab {
	return &ControlTab{
		controller:  ctrl,
		instanceMap: make(map[string]int),
	}
}

// Build constructs the bot control UI
func (c *ControlTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Bot Controls", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Status label
	c.statusLabel = widget.NewLabel("No bots running")
	c.statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Instance selector
	c.instanceSelect = widget.NewSelect([]string{}, nil)
	c.populateInstanceDropdown() // Populate with player names

	refreshBtn := widget.NewButton("Refresh", func() {
		c.populateInstanceDropdown()
	})

	instanceSelector := container.NewHBox(
		widget.NewLabel("Instance:"),
		c.instanceSelect,
		refreshBtn,
	)

	// Single instance controls
	launchBtn := widget.NewButton("Launch MuMu", func() {
		c.launchMuMuInstance()
	})

	c.startBtn = widget.NewButton("Start Bot", func() {
		c.startInstance()
	})

	c.stopBtn = widget.NewButton("Stop Bot", func() {
		c.stopInstance()
	})

	c.pauseBtn = widget.NewButton("Pause Bot", func() {
		c.pauseInstance()
	})

	c.resumeBtn = widget.NewButton("Resume Bot", func() {
		c.resumeInstance()
	})

	positionBtn := widget.NewButton("Position Window", func() {
		c.positionInstance()
	})

	snapshotBtn := widget.NewButton("Snapshot Screen", func() {
		c.snapshotScreen()
	})

	snapshotRegionBtn := widget.NewButton("Snapshot Region", func() {
		c.snapshotRegion()
	})

	singleControls := container.NewGridWithColumns(4,
		launchBtn,
		positionBtn,
		snapshotBtn,
		snapshotRegionBtn,
		c.startBtn,
		c.stopBtn,
		c.pauseBtn,
		c.resumeBtn,
	)

	singleInstanceSection := container.NewVBox(
		widget.NewLabelWithStyle("Single Instance Control", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		instanceSelector,
		singleControls,
	)

	// Multi-instance controls
	c.instanceCountEntry = widget.NewEntry()
	c.instanceCountEntry.SetText("1")
	c.instanceCountEntry.SetPlaceHolder("Number of instances")

	instanceCountInput := container.NewHBox(
		widget.NewLabel("Number of Instances:"),
		c.instanceCountEntry,
	)

	launchAllBtn := widget.NewButton("Launch All MuMu", func() {
		c.launchAllMuMuInstances()
	})

	c.startAllBtn = widget.NewButton("Start All Bots", func() {
		c.startAllInstances()
	})

	c.stopAllBtn = widget.NewButton("Stop All Bots", func() {
		c.stopAllInstances()
	})

	multiControls := container.NewGridWithColumns(2,
		launchAllBtn,
		c.startAllBtn,
		c.stopAllBtn,
	)

	multiInstanceSection := container.NewVBox(
		widget.NewLabelWithStyle("Multi-Instance Control", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		instanceCountInput,
		multiControls,
	)

	// Quick actions
	quickActionsSection := container.NewVBox(
		widget.NewLabelWithStyle("Quick Actions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewButton("Test Screen Detection", func() {
			c.testScreenDetection()
		}),
		widget.NewButton("Test Window Capture", func() {
			c.testWindowCapture()
		}),
		widget.NewButton("Test ADB Connection", func() {
			c.testADBConnection()
		}),
		widget.NewButton("Test Click at Coordinates", func() {
			c.testClickAtCoordinates()
		}),
		widget.NewButton("Test FindAndClickCenter", func() {
			c.testFindAndClickCenter()
		}),
	)

	// Layout
	content := container.NewVScroll(
		container.NewVBox(
			header,
			c.statusLabel,
			widget.NewSeparator(),
			singleInstanceSection,
			widget.NewSeparator(),
			multiInstanceSection,
			widget.NewSeparator(),
			quickActionsSection,
		),
	)

	return content
}

// startInstance starts a single bot instance
func (c *ControlTab) startInstance() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	// Check if already running
	if _, exists := c.controller.GetBot(instanceNum); exists {
		c.showError(fmt.Sprintf("Instance %d is already running", instanceNum))
		return
	}

	// Create bot
	cfg := c.controller.GetConfig()
	cfg.Instance = instanceNum

	b, err := bot.New(instanceNum, cfg)
	if err != nil {
		c.showError(fmt.Sprintf("Failed to create bot: %v", err))
		return
	}

	// Initialize bot
	if err := b.Initialize(); err != nil {
		c.showError(fmt.Sprintf("Failed to initialize bot: %v", err))
		return
	}

	// Add to controller
	c.controller.AddBot(instanceNum, b)

	// Log
	c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Bot instance started")

	// Update status
	c.updateStatus()

	c.showSuccess(fmt.Sprintf("Instance %d started successfully", instanceNum))
}

// stopInstance stops a single bot instance
func (c *ControlTab) stopInstance() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	// Check if running
	if _, exists := c.controller.GetBot(instanceNum); !exists {
		c.showError(fmt.Sprintf("Instance %d is not running", instanceNum))
		return
	}

	// Remove bot (will call Shutdown)
	c.controller.RemoveBot(instanceNum)

	// Log
	c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Bot instance stopped")

	// Update status
	c.updateStatus()

	c.showSuccess(fmt.Sprintf("Instance %d stopped successfully", instanceNum))
}

// pauseInstance pauses a bot instance
func (c *ControlTab) pauseInstance() {
	// TODO: Implement pause logic
	c.controller.logTab.AddLog(LogLevelInfo, 0, "Pause functionality coming soon")
}

// resumeInstance resumes a paused bot instance
func (c *ControlTab) resumeInstance() {
	// TODO: Implement resume logic
	c.controller.logTab.AddLog(LogLevelInfo, 0, "Resume functionality coming soon")
}

// startAllInstances starts multiple bot instances
func (c *ControlTab) startAllInstances() {
	count, err := strconv.Atoi(c.instanceCountEntry.Text)
	if err != nil || count < 1 || count > 10 {
		c.showError("Invalid instance count (must be 1-10)")
		return
	}

	started := 0
	for i := 1; i <= count; i++ {
		// Check if already running
		if _, exists := c.controller.GetBot(i); exists {
			continue
		}

		// Create and initialize bot
		cfg := c.controller.GetConfig()
		cfg.Instance = i

		b, err := bot.New(i, cfg)
		if err != nil {
			c.controller.logTab.AddLog(LogLevelError, i, fmt.Sprintf("Failed to create bot: %v", err))
			continue
		}

		if err := b.Initialize(); err != nil {
			c.controller.logTab.AddLog(LogLevelError, i, fmt.Sprintf("Failed to initialize bot: %v", err))
			continue
		}

		c.controller.AddBot(i, b)
		c.controller.logTab.AddLog(LogLevelInfo, i, "Bot instance started")
		started++
	}

	c.updateStatus()
	c.showSuccess(fmt.Sprintf("Started %d instances", started))
}

// stopAllInstances stops all running bot instances
func (c *ControlTab) stopAllInstances() {
	bots := c.controller.GetAllBots()
	stopped := 0

	for instance := range bots {
		c.controller.RemoveBot(instance)
		c.controller.logTab.AddLog(LogLevelInfo, instance, "Bot instance stopped")
		stopped++
	}

	c.updateStatus()
	c.showSuccess(fmt.Sprintf("Stopped %d instances", stopped))
}

// testScreenDetection tests screen detection
func (c *ControlTab) testScreenDetection() {
	c.controller.logTab.AddLog(LogLevelInfo, 0, "Screen detection test coming soon")
}

// testWindowCapture tests window capture
func (c *ControlTab) testWindowCapture() {
	c.controller.logTab.AddLog(LogLevelInfo, 0, "Window capture test coming soon")
}

// testADBConnection tests ADB connection
func (c *ControlTab) testADBConnection() {
	cfg := c.controller.GetConfig()

	// Check if ADB path is set
	adbCfg := cfg.ADB()
	if adbCfg.Path == "" {
		c.showError("ADB path is not configured. Please set it in the Configuration tab.")
		c.controller.logTab.AddLog(LogLevelError, 0, "ADB test failed: No ADB path configured")
		return
	}

	c.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Testing ADB at: %s", adbCfg.Path))

	// Try to find ADB
	adbPath := adbCfg.Path
	if cfg.ADBPath == "" {
		// Try auto-detection
		c.controller.logTab.AddLog(LogLevelInfo, 0, "ADB path not explicitly set, searching in MuMu folder...")

		// This will be done during bot initialization
		c.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Will search in: %s", cfg.FolderPath))
		c.controller.logTab.AddLog(LogLevelInfo, 0, "Expected location: vmonitor/bin/adb_server.exe")
	}

	c.showSuccess(fmt.Sprintf("ADB configured at:\n%s\n\nTo fully test, start a bot instance.", adbPath))
}

// testClickAtCoordinates opens a dialog to test clicking at specific coordinates
func (c *ControlTab) testClickAtCoordinates() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	// Check if bot is running for this instance
	b, exists := c.controller.GetBot(instanceNum)
	if !exists {
		c.showError(fmt.Sprintf("Instance %d is not running. Start the bot first.", instanceNum))
		return
	}

	// Create entries for X and Y coordinates
	xEntry := widget.NewEntry()
	xEntry.SetPlaceHolder("X coordinate")
	xEntry.SetText("135")

	yEntry := widget.NewEntry()
	yEntry.SetPlaceHolder("Y coordinate")
	yEntry.SetText("135")

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Enter coordinates to click:"),
		container.NewHBox(widget.NewLabel("X:"), xEntry),
		container.NewHBox(widget.NewLabel("Y:"), yEntry),
	)

	// Create dialog
	dlg := dialog.NewCustomConfirm("Test Click at Coordinates", "Click", "Cancel", form, func(confirmed bool) {
		if !confirmed {
			return
		}

		// Parse coordinates
		x, err := strconv.Atoi(xEntry.Text)
		if err != nil {
			c.showError(fmt.Sprintf("Invalid X coordinate: %v", err))
			return
		}

		y, err := strconv.Atoi(yEntry.Text)
		if err != nil {
			c.showError(fmt.Sprintf("Invalid Y coordinate: %v", err))
			return
		}

		// Perform click in goroutine
		go func() {
			c.controller.logTab.AddLog(LogLevelInfo, instanceNum, fmt.Sprintf("Clicking at coordinates (%d, %d)", x, y))

			if err := b.ADB().Click(x, y); err != nil {
				c.showError(fmt.Sprintf("Failed to click: %v", err))
				c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Click failed: %v", err))
				return
			}

			c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Click successful")
			c.showSuccess(fmt.Sprintf("Clicked at (%d, %d)", x, y))
		}()
	}, c.controller.window)

	dlg.Resize(fyne.NewSize(300, 150))
	dlg.Show()
}

// testFindAndClickCenter opens a dialog to test FindAndClickCenter with template selection
func (c *ControlTab) testFindAndClickCenter() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	// Check if bot is running for this instance
	b, exists := c.controller.GetBot(instanceNum)
	if !exists {
		c.showError(fmt.Sprintf("Instance %d is not running. Start the bot first.", instanceNum))
		return
	}

	// Create dropdown for template selection
	templateSelect := widget.NewSelect([]string{"Shop", "DailyMissions", "Menu", "Mail"}, nil)
	templateSelect.SetSelected("Menu")

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Select a template to find and click:"),
		templateSelect,
	)

	// Create dialog
	dlg := dialog.NewCustomConfirm("Test FindAndClickCenter", "Find & Click", "Cancel", form, func(confirmed bool) {
		if !confirmed {
			return
		}

		selectedTemplate := templateSelect.Selected
		if selectedTemplate == "" {
			c.showError("Please select a template")
			return
		}

		// Perform FindAndClickCenter in goroutine
		go func() {
			c.controller.logTab.AddLog(LogLevelInfo, instanceNum, fmt.Sprintf("Finding and clicking template: %s", selectedTemplate))

			// Get the appropriate template from the templates package
			// TODO: Update to use new YAML-based routine system
			// This functionality needs to be reimplemented with the new action system
			c.showError("This feature is temporarily disabled during refactoring")
			c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "FindAndClickCenter is temporarily disabled")
			_ = selectedTemplate // Suppress unused variable warning
			_ = b                // Suppress unused variable warning
		}()
	}, c.controller.window)

	dlg.Resize(fyne.NewSize(300, 150))
	dlg.Show()
}

// updateStatus updates the status label
func (c *ControlTab) updateStatus() {
	bots := c.controller.GetAllBots()
	count := len(bots)

	switch count {
	case 0:
		c.statusLabel.SetText("No bots running")
	case 1:
		c.statusLabel.SetText("1 bot running")
	default:
		c.statusLabel.SetText(fmt.Sprintf("%d bots running", count))
	}
}

// showError displays an error dialog
func (c *ControlTab) showError(message string) {
	dialog.ShowError(fmt.Errorf("%s", message), c.controller.window)
}

// showSuccess displays a success dialog
func (c *ControlTab) showSuccess(message string) {
	dialog.ShowInformation("Success", message, c.controller.window)
}

// launchMuMuInstance launches a MuMu instance
func (c *ControlTab) launchMuMuInstance() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	cfg := c.controller.GetConfig()

	// Run in goroutine to avoid blocking UI
	go func() {
		// Create a temporary emulator manager just to launch the instance
		// We don't need full initialization, just the launch capability
		adbPath := cfg.ADB().Path
		if adbPath == "" {
			adbPath = "dummy" // We don't actually need ADB for launching
		}

		mgr := emulator.NewManager(cfg.FolderPath, adbPath)

		// Try to discover instances first to check if it's already running
		if err := mgr.DiscoverInstances(); err != nil {
			c.controller.logTab.AddLog(LogLevelWarn, 0, fmt.Sprintf("Could not check running instances: %v", err))
		}

		// Check if already running
		if mgr.IsInstanceRunning(instanceNum) {
			c.showError(fmt.Sprintf("MuMu instance %d is already running", instanceNum))
			c.controller.logTab.AddLog(LogLevelWarn, instanceNum, "Instance already running")
			return
		}

		// Launch the instance
		c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Launching MuMu instance...")

		if err := mgr.LaunchInstance(instanceNum); err != nil {
			c.showError(fmt.Sprintf("Failed to launch MuMu instance %d: %v", instanceNum, err))
			c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Launch failed: %v", err))
			return
		}

		c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "MuMu instance launched successfully")
		c.showSuccess(fmt.Sprintf("MuMu instance %d launched.\n\nWait a few seconds for it to start, then click 'Start Bot'.", instanceNum))
	}()
}

// launchAllMuMuInstances launches multiple MuMu instances
func (c *ControlTab) launchAllMuMuInstances() {
	count, err := strconv.Atoi(c.instanceCountEntry.Text)
	if err != nil || count < 1 || count > 10 {
		c.showError("Invalid instance count (must be 1-10)")
		return
	}

	cfg := c.controller.GetConfig()

	// Run in goroutine to avoid blocking UI
	go func() {
		adbPath := cfg.ADB().Path
		if adbPath == "" {
			adbPath = "dummy"
		}

		mgr := emulator.NewManager(cfg.FolderPath, adbPath)

		// Discover running instances
		if err := mgr.DiscoverInstances(); err != nil {
			c.controller.logTab.AddLog(LogLevelWarn, 0, fmt.Sprintf("Could not check running instances: %v", err))
		}

		launched := 0
		for i := 1; i <= count; i++ {
			// Check if already running
			if mgr.IsInstanceRunning(i) {
				c.controller.logTab.AddLog(LogLevelInfo, i, "Instance already running, skipping")
				continue
			}

			// Launch the instance
			c.controller.logTab.AddLog(LogLevelInfo, i, "Launching MuMu instance...")

			if err := mgr.LaunchInstance(i); err != nil {
				c.controller.logTab.AddLog(LogLevelError, i, fmt.Sprintf("Launch failed: %v", err))
				continue
			}

			c.controller.logTab.AddLog(LogLevelInfo, i, "MuMu instance launched")
			launched++
		}

		c.showSuccess(fmt.Sprintf("Launched %d MuMu instances.\n\nWait a few seconds for them to start, then click 'Start All Bots'.", launched))
	}()
}

// positionInstance positions a specific MuMu instance window
func (c *ControlTab) positionInstance() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	cfg := c.controller.GetConfig()

	// Run in goroutine to avoid blocking UI
	go func() {
		adbPath := cfg.ADB().Path
		if adbPath == "" {
			adbPath = "dummy"
		}

		mgr := emulator.NewManager(cfg.FolderPath, adbPath)

		// Discover running instances
		c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Discovering instances...")
		if err := mgr.DiscoverInstances(); err != nil {
			c.showError(fmt.Sprintf("Failed to discover instances: %v", err))
			c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Discovery failed: %v", err))
			return
		}

		// Check if instance is running
		if !mgr.IsInstanceRunning(instanceNum) {
			c.showError(fmt.Sprintf("MuMu instance %d is not running", instanceNum))
			c.controller.logTab.AddLog(LogLevelError, instanceNum, "Instance not running")
			return
		}

		// Create window config from bot config
		windowConfig := emulator.NewWindowConfig(
			cfg.Columns,
			cfg.RowGap,
			getScaleParam(cfg.DefaultLanguage),
			cfg.SelectedMonitor,
		)

		// Position the specific instance
		c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Positioning window...")
		if err := mgr.PositionInstance(instanceNum, windowConfig); err != nil {
			c.showError(fmt.Sprintf("Failed to position instance %d: %v", instanceNum, err))
			c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Position failed: %v", err))
			return
		}

		c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Window positioned successfully")
		c.showSuccess(fmt.Sprintf("Instance %d window positioned and resized", instanceNum))
	}()
}

// populateInstanceDropdown populates the instance dropdown with player names
func (c *ControlTab) populateInstanceDropdown() {
	cfg := c.controller.GetConfig()

	// Create emulator manager to read instance configs
	adbPath := cfg.ADB().Path
	if adbPath == "" {
		adbPath = "dummy"
	}

	mgr := emulator.NewManager(cfg.FolderPath, adbPath)

	// Get all instance configurations
	configs, err := mgr.GetAllInstanceConfigs()
	if err != nil {
		c.controller.logTab.AddLog(LogLevelWarn, 0, fmt.Sprintf("Failed to load instance configs: %v", err))
		// Fallback to numbered instances
		c.instanceSelect.Options = []string{"Instance 1", "Instance 2", "Instance 3", "Instance 4", "Instance 5"}
		c.instanceMap = map[string]int{
			"Instance 1": 1,
			"Instance 2": 2,
			"Instance 3": 3,
			"Instance 4": 4,
			"Instance 5": 5,
		}
		if len(c.instanceSelect.Options) > 0 {
			c.instanceSelect.SetSelected(c.instanceSelect.Options[0])
		}
		c.instanceSelect.Refresh()
		return
	}

	// Build dropdown options with player names
	options := []string{}
	newInstanceMap := make(map[string]int)

	// Sort by instance number for consistent ordering
	for i := 0; i <= 10; i++ {
		if config, exists := configs[i]; exists && config.PlayerName != "" {
			displayName := fmt.Sprintf("%d: %s", i, config.PlayerName)
			options = append(options, displayName)
			newInstanceMap[displayName] = i
		} else if i > 0 && i <= 5 {
			// For instances 1-5 without configs, still show them
			displayName := fmt.Sprintf("Instance %d", i)
			options = append(options, displayName)
			newInstanceMap[displayName] = i
		}
	}

	// Update the dropdown
	c.instanceMap = newInstanceMap
	c.instanceSelect.Options = options

	if len(options) > 0 {
		c.instanceSelect.SetSelected(options[0])
	}

	c.instanceSelect.Refresh()
	c.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Loaded %d instance configurations", len(options)))
}

// getSelectedInstance returns the instance number from the selected dropdown item
func (c *ControlTab) getSelectedInstance() (int, error) {
	selected := c.instanceSelect.Selected
	if selected == "" {
		return 0, fmt.Errorf("no instance selected")
	}

	instanceNum, exists := c.instanceMap[selected]
	if !exists {
		return 0, fmt.Errorf("invalid instance selection")
	}

	return instanceNum, nil
}

// getScaleParam returns the scale parameter based on language
func getScaleParam(language string) int {
	// This is a helper function that matches the logic in bot.go
	switch language {
	case "en":
		return 270
	case "es":
		return 270
	case "fr":
		return 270
	case "de":
		return 270
	case "it":
		return 270
	case "pt":
		return 270
	default:
		return 270
	}
}

// snapshotScreen captures the full window and saves it as PNG
func (c *ControlTab) snapshotScreen() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	// Check if bot is running for this instance
	b, exists := c.controller.GetBot(instanceNum)
	if !exists {
		c.showError(fmt.Sprintf("Instance %d is not running. Start the bot first.", instanceNum))
		return
	}

	// Create file name entry
	fileNameEntry := widget.NewEntry()
	fileNameEntry.SetText(fmt.Sprintf("snapshot_instance_%d.png", instanceNum))
	fileNameEntry.SetPlaceHolder("File name")

	// Create form
	form := container.NewVBox(
		widget.NewLabel("Enter filename for snapshot:"),
		fileNameEntry,
		widget.NewLabel("(File will be saved in current directory)"),
	)

	// Create dialog
	dlg := dialog.NewCustomConfirm("Snapshot Full Screen", "Save", "Cancel", form, func(confirmed bool) {
		if !confirmed {
			return
		}

		fileName := fileNameEntry.Text
		if fileName == "" {
			c.showError("Please enter a filename")
			return
		}

		// Capture and save in goroutine
		go func() {
			c.controller.logTab.AddLog(LogLevelInfo, instanceNum, "Capturing full screen...")

			// Capture frame
			frame, err := b.CV().CaptureFrame(false)
			if err != nil {
				c.showError(fmt.Sprintf("Failed to capture frame: %v", err))
				c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Capture failed: %v", err))
				return
			}

			// Save to PNG
			if err := savePNG(frame, fileName); err != nil {
				c.showError(fmt.Sprintf("Failed to save PNG: %v", err))
				c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Save failed: %v", err))
				return
			}

			c.controller.logTab.AddLog(LogLevelInfo, instanceNum, fmt.Sprintf("Screenshot saved to: %s", fileName))
			c.showSuccess(fmt.Sprintf("Screenshot saved successfully!\n\nFile: %s\n\nSize: %dx%d",
				fileName, frame.Bounds().Dx(), frame.Bounds().Dy()))
		}()
	}, c.controller.window)

	dlg.Resize(fyne.NewSize(400, 150))
	dlg.Show()
}

// snapshotRegion captures a specific region and saves it as PNG
func (c *ControlTab) snapshotRegion() {
	instanceNum, err := c.getSelectedInstance()
	if err != nil {
		c.showError(fmt.Sprintf("Invalid instance selection: %v", err))
		return
	}

	// Check if bot is running for this instance
	b, exists := c.controller.GetBot(instanceNum)
	if !exists {
		c.showError(fmt.Sprintf("Instance %d is not running. Start the bot first.", instanceNum))
		return
	}

	// Get window dimensions for reference
	width, height := b.CV().GetDimensions()

	// Create entries for region coordinates
	x1Entry := widget.NewEntry()
	x1Entry.SetPlaceHolder("X1")
	x1Entry.SetText("0")

	y1Entry := widget.NewEntry()
	y1Entry.SetPlaceHolder("Y1")
	y1Entry.SetText("0")

	x2Entry := widget.NewEntry()
	x2Entry.SetPlaceHolder("X2")
	x2Entry.SetText(fmt.Sprintf("%d", width/2))

	y2Entry := widget.NewEntry()
	y2Entry.SetPlaceHolder("Y2")
	y2Entry.SetText(fmt.Sprintf("%d", height/2))

	fileNameEntry := widget.NewEntry()
	fileNameEntry.SetText(fmt.Sprintf("region_instance_%d.png", instanceNum))
	fileNameEntry.SetPlaceHolder("File name")

	// Create form
	form := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Window size: %dx%d", width, height)),
		widget.NewLabel("Enter region coordinates (window-relative):"),
		container.NewGridWithColumns(2,
			widget.NewLabel("Top-Left:"),
			widget.NewLabel("Bottom-Right:"),
		),
		container.NewGridWithColumns(4,
			widget.NewLabel("X1:"), x1Entry,
			widget.NewLabel("X2:"), x2Entry,
		),
		container.NewGridWithColumns(4,
			widget.NewLabel("Y1:"), y1Entry,
			widget.NewLabel("Y2:"), y2Entry,
		),
		widget.NewSeparator(),
		widget.NewLabel("Output filename:"),
		fileNameEntry,
	)

	// Create dialog
	dlg := dialog.NewCustomConfirm("Snapshot Region", "Capture", "Cancel", form, func(confirmed bool) {
		if !confirmed {
			return
		}

		// Parse coordinates
		x1, err := strconv.Atoi(x1Entry.Text)
		if err != nil {
			c.showError(fmt.Sprintf("Invalid X1 coordinate: %v", err))
			return
		}

		y1, err := strconv.Atoi(y1Entry.Text)
		if err != nil {
			c.showError(fmt.Sprintf("Invalid Y1 coordinate: %v", err))
			return
		}

		x2, err := strconv.Atoi(x2Entry.Text)
		if err != nil {
			c.showError(fmt.Sprintf("Invalid X2 coordinate: %v", err))
			return
		}

		y2, err := strconv.Atoi(y2Entry.Text)
		if err != nil {
			c.showError(fmt.Sprintf("Invalid Y2 coordinate: %v", err))
			return
		}

		fileName := fileNameEntry.Text
		if fileName == "" {
			c.showError("Please enter a filename")
			return
		}

		// Validate region
		if x1 >= x2 || y1 >= y2 {
			c.showError("Invalid region: X2 must be > X1 and Y2 must be > Y1")
			return
		}

		if x1 < 0 || y1 < 0 || x2 > width || y2 > height {
			c.showError(fmt.Sprintf("Region out of bounds. Valid range: 0--%d (width), 0--%d (height)", width, height))
			return
		}

		// Capture and save in goroutine
		go func() {
			c.controller.logTab.AddLog(LogLevelInfo, instanceNum,
				fmt.Sprintf("Capturing region (%d,%d) to (%d,%d)...", x1, y1, x2, y2))

			// Capture full frame
			frame, err := b.CV().CaptureFrame(false)
			if err != nil {
				c.showError(fmt.Sprintf("Failed to capture frame: %v", err))
				c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Capture failed: %v", err))
				return
			}

			// Crop the region
			region := cv.CropRegion(frame, image.Rect(x1, y1, x2, y2))

			// Save to PNG
			if err := savePNG(region, fileName); err != nil {
				c.showError(fmt.Sprintf("Failed to save PNG: %v", err))
				c.controller.logTab.AddLog(LogLevelError, instanceNum, fmt.Sprintf("Save failed: %v", err))
				return
			}

			c.controller.logTab.AddLog(LogLevelInfo, instanceNum, fmt.Sprintf("Region screenshot saved to: %s", fileName))
			c.showSuccess(fmt.Sprintf("Region screenshot saved!\n\nFile: %s\nRegion: (%d,%d) to (%d,%d)\nSize: %dx%d",
				fileName, x1, y1, x2, y2, x2-x1, y2-y1))
		}()
	}, c.controller.window)

	dlg.Resize(fyne.NewSize(400, 300))
	dlg.Show()
}

// savePNG saves an image to a PNG file
func savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}
