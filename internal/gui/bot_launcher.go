package gui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/coordinator"
)

// BotLauncherTab allows launching multiple bots with routine selection
type BotLauncherTab struct {
	controller *Controller

	// UI components
	numBotsEntry    *widget.Entry
	botConfigs      []*BotLaunchConfig
	configContainer *fyne.Container
	launchBtn       *widget.Button
	stopBtn         *widget.Button
	setAllBtn       *widget.Button
	statusLabel     *widget.Label

	// Runtime state
	manager           *bot.Manager
	coordinator       *coordinator.BotCoordinator
	runningBots       map[int]*bot.Bot
	availableRoutines []string
	displayToFilename map[string]string // Maps display text -> filename

	// Status polling
	pollingActive bool
	pollingStop   chan struct{}
	pollingWg     sync.WaitGroup
}

// BotLaunchConfig represents configuration for a single bot instance
type BotLaunchConfig struct {
	instance        int
	routineSelect   *widget.Select
	statusLabel     *widget.Label
	statusIndicator *canvas.Circle // Visual state indicator
	selectedRoutine string
	// Individual control buttons
	pauseBtn   *widget.Button
	resumeBtn  *widget.Button
	stopBtn    *widget.Button
	restartBtn *widget.Button
}

// NewBotLauncherTab creates a new bot launcher tab
func NewBotLauncherTab(ctrl *Controller) *BotLauncherTab {
	return &BotLauncherTab{
		controller:        ctrl,
		runningBots:       make(map[int]*bot.Bot),
		displayToFilename: make(map[string]string),
		pollingStop:       make(chan struct{}),
	}
}

// Build constructs the UI
func (t *BotLauncherTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Bot Launcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	description := widget.NewLabel("Configure and launch multiple bot instances with routine selection")

	// Create manager early to access shared registries for routine discovery
	t.initializeManager()

	// Load available routines
	t.loadAvailableRoutines()

	// Number of bots input
	numBotsLabel := widget.NewLabel("Number of Bots:")
	t.numBotsEntry = widget.NewEntry()
	t.numBotsEntry.SetPlaceHolder("e.g., 6")
	t.numBotsEntry.SetText("6")

	generateBtn := widget.NewButton("Generate Bot Configs", func() {
		t.generateBotConfigs()
	})

	numBotsRow := container.NewBorder(nil, nil, numBotsLabel, generateBtn, t.numBotsEntry)

	// Set All button (for setting all bots to same routine)
	t.setAllBtn = widget.NewButton("Set All to...", func() {
		t.showSetAllDialog()
	})
	t.setAllBtn.Disable()

	// Bot configurations container
	t.configContainer = container.NewVBox()

	// Launch/Stop buttons
	t.launchBtn = widget.NewButton("Launch All Bots", func() {
		t.launchAllBots()
	})
	t.launchBtn.Disable()

	t.stopBtn = widget.NewButton("Stop All Bots", func() {
		t.stopAllBots()
	})
	t.stopBtn.Disable()

	// Reload buttons for development
	reloadRoutinesBtn := widget.NewButton("Reload Routines", func() {
		t.reloadRoutines()
	})

	reloadTemplatesBtn := widget.NewButton("Reload Templates", func() {
		t.reloadTemplates()
	})

	buttonsRow := container.NewHBox(
		t.setAllBtn,
		t.launchBtn,
		t.stopBtn,
	)

	devToolsRow := container.NewHBox(
		widget.NewLabel("Dev Tools:"),
		reloadRoutinesBtn,
		reloadTemplatesBtn,
	)

	// Status label
	t.statusLabel = widget.NewLabel("Configure bots to launch")

	// Scrollable content
	content := container.NewVScroll(
		container.NewVBox(
			numBotsRow,
			widget.NewSeparator(),
			buttonsRow,
			devToolsRow,
			widget.NewSeparator(),
			t.configContainer,
			widget.NewSeparator(),
			t.statusLabel,
		),
	)

	// Auto-generate default configs
	t.generateBotConfigs()

	return container.NewBorder(
		container.NewVBox(header, description),
		nil,
		nil,
		nil,
		content,
	)
}

// initializeManager creates the manager with shared registries if not already created
func (t *BotLauncherTab) initializeManager() {
	if t.manager != nil {
		return
	}

	var err error
	t.manager, err = bot.NewManager(t.controller.config)
	if err != nil {
		// Log error but continue - this is for pre-initialization
		fmt.Printf("Warning: Failed to create manager during initialization: %v\n", err)
	}
}

// loadAvailableRoutines loads available routines from the shared registry
// Groups routines by namespace (folder) for better organization
func (t *BotLauncherTab) loadAvailableRoutines() {
	t.availableRoutines = []string{"<none>"}
	t.displayToFilename = make(map[string]string)
	t.displayToFilename["<none>"] = "" // Map <none> to empty string

	// If we have a manager with a routine registry, use it
	if t.manager != nil && t.manager.RoutineRegistry() != nil {
		registry := t.manager.RoutineRegistry()

		// Type assert to access the ListByNamespace method
		if rr, ok := registry.(*actions.RoutineRegistry); ok {
			// Get routines grouped by namespace
			namespaces := rr.ListByNamespace()

			// Sort namespace names for consistent ordering
			var sortedNamespaces []string
			for ns := range namespaces {
				sortedNamespaces = append(sortedNamespaces, ns)
			}
			sort.Strings(sortedNamespaces)

			// Add routines grouped by namespace
			for _, namespace := range sortedNamespaces {
				routines := namespaces[namespace]

				// Add namespace header if not top-level
				if namespace != "" {
					header := fmt.Sprintf("── %s ──", namespace)
					t.availableRoutines = append(t.availableRoutines, header)
					// Map header to empty string (not selectable in practice)
					t.displayToFilename[header] = ""
				}

				// Add routines in this namespace
				for _, filename := range routines {
					metaInterface := registry.GetMetadata(filename)
					meta, ok := metaInterface.(*actions.RoutineMetadata)
					if !ok {
						continue
					}

					// For namespaced routines, show just the base name + full path
					displayText := fmt.Sprintf("%s (%s)", meta.DisplayName, filename)

					// Check if invalid
					if err := registry.GetValidationError(filename); err != nil {
						displayText = fmt.Sprintf("⚠️ %s [INVALID]", displayText)
					}

					t.availableRoutines = append(t.availableRoutines, displayText)
					t.displayToFilename[displayText] = filename
				}
			}
		} else {
			// Fallback: flat list if not using RoutineRegistry
			filenames := registry.ListAvailable()
			for _, filename := range filenames {
				metaInterface := registry.GetMetadata(filename)
				meta, ok := metaInterface.(*actions.RoutineMetadata)
				if !ok {
					continue
				}
				displayText := fmt.Sprintf("%s (%s)", meta.DisplayName, meta.Filename)

				if err := registry.GetValidationError(filename); err != nil {
					displayText = fmt.Sprintf("⚠️ %s [INVALID]", displayText)
				}

				t.availableRoutines = append(t.availableRoutines, displayText)
				t.displayToFilename[displayText] = filename
			}
		}
	} else {
		// Fallback: scan filesystem directly if manager not yet created
		routinesPath := "routines"
		entries, err := filepath.Glob(filepath.Join(routinesPath, "*.yaml"))
		if err != nil {
			return
		}

		for _, path := range entries {
			filename := filepath.Base(path)
			routineName := strings.TrimSuffix(filename, filepath.Ext(filename))
			t.availableRoutines = append(t.availableRoutines, routineName)
			t.displayToFilename[routineName] = routineName
		}
	}
}

// generateBotConfigs creates configuration UI for each bot
func (t *BotLauncherTab) generateBotConfigs() {
	// Parse number of bots
	numBots := 6 // default
	if text := t.numBotsEntry.Text; text != "" {
		fmt.Sscanf(text, "%d", &numBots)
	}

	if numBots < 1 || numBots > 20 {
		dialog.ShowError(fmt.Errorf("number of bots must be between 1 and 20"), t.controller.window)
		return
	}

	// Clear existing configs
	t.configContainer.Objects = nil
	t.botConfigs = nil

	// Create config for each bot
	for i := 1; i <= numBots; i++ {
		config := t.createBotConfig(i)
		t.botConfigs = append(t.botConfigs, config)

		// Add to UI
		card := t.createBotConfigCard(config)
		t.configContainer.Add(card)
	}

	t.configContainer.Refresh()

	// Enable buttons
	t.setAllBtn.Enable()
	t.launchBtn.Enable()

	t.statusLabel.SetText(fmt.Sprintf("Generated %d bot configurations", numBots))
}

// createBotConfig creates a configuration object for a bot
func (t *BotLauncherTab) createBotConfig(instance int) *BotLaunchConfig {
	routineSelect := widget.NewSelect(t.availableRoutines, func(selected string) {
		// Update the selected routine
		for _, cfg := range t.botConfigs {
			if cfg.instance == instance {
				cfg.selectedRoutine = selected
				break
			}
		}
	})

	// Default to <none>
	if len(t.availableRoutines) > 0 {
		routineSelect.SetSelected(t.availableRoutines[0])
	}

	// Create status indicator circle (gray initially)
	statusIndicator := canvas.NewCircle(color.RGBA{R: 128, G: 128, B: 128, A: 255})
	statusIndicator.Resize(fyne.NewSize(12, 12))

	config := &BotLaunchConfig{
		instance:        instance,
		routineSelect:   routineSelect,
		statusLabel:     widget.NewLabel("Ready"),
		statusIndicator: statusIndicator,
		selectedRoutine: "<none>",
	}

	// Create individual control buttons (disabled initially)
	config.pauseBtn = widget.NewButton("Pause", func() {
		t.pauseBot(instance)
	})
	config.pauseBtn.Disable()

	config.resumeBtn = widget.NewButton("Resume", func() {
		t.resumeBot(instance)
	})
	config.resumeBtn.Disable()

	config.stopBtn = widget.NewButton("Stop", func() {
		t.stopBot(instance)
	})
	config.stopBtn.Disable()

	config.restartBtn = widget.NewButton("Restart", func() {
		t.restartBot(instance)
	})
	config.restartBtn.Disable()

	return config
}

// createBotConfigCard creates a UI card for a bot configuration
func (t *BotLauncherTab) createBotConfigCard(config *BotLaunchConfig) fyne.CanvasObject {
	instanceLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("Bot %d", config.instance),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	routineLabel := widget.NewLabel("Routine:")

	// Control buttons row
	controlButtons := container.NewHBox(
		config.pauseBtn,
		config.resumeBtn,
		config.stopBtn,
		config.restartBtn,
	)

	// Routine selection row
	routineRow := container.NewBorder(nil, nil, routineLabel, nil, config.routineSelect)

	// Status row with indicator and label
	statusRow := container.NewHBox(
		config.statusIndicator,
		config.statusLabel,
	)

	// Bottom section with status and controls
	bottomSection := container.NewVBox(
		statusRow,
		controlButtons,
	)

	card := container.NewBorder(
		instanceLabel,
		bottomSection,
		nil,
		nil,
		routineRow,
	)

	return container.NewPadded(card)
}

// showSetAllDialog shows a dialog to set all bots to the same routine
func (t *BotLauncherTab) showSetAllDialog() {
	routineSelect := widget.NewSelect(t.availableRoutines, nil)
	if len(t.availableRoutines) > 0 {
		routineSelect.SetSelected(t.availableRoutines[0])
	}

	dialog.ShowCustomConfirm(
		"Set All Routines",
		"Apply",
		"Cancel",
		container.NewVBox(
			widget.NewLabel("Select routine for all bots:"),
			routineSelect,
		),
		func(apply bool) {
			if apply && routineSelect.Selected != "" {
				t.setAllRoutines(routineSelect.Selected)
			}
		},
		t.controller.window,
	)
}

// setAllRoutines sets all bots to the same routine
func (t *BotLauncherTab) setAllRoutines(routine string) {
	for _, config := range t.botConfigs {
		config.routineSelect.SetSelected(routine)
		config.selectedRoutine = routine
	}
	t.statusLabel.SetText(fmt.Sprintf("Set all bots to routine: %s", routine))
}

// launchAllBots launches all configured bots
func (t *BotLauncherTab) launchAllBots() {
	config := t.controller.config

	// Ensure manager is initialized (should already be done in Build())
	if t.manager == nil {
		var err error
		t.manager, err = bot.NewManager(config)
		if err != nil {
			t.statusLabel.SetText(fmt.Sprintf("Error: Failed to create manager: %v", err))
			dialog.ShowError(err, t.controller.window)
			return
		}
	}

	// Create coordinator for account injection
	t.coordinator = coordinator.NewBotCoordinator(config)

	// Launch each configured bot
	successCount := 0
	for _, botConfig := range t.botConfigs {
		err := t.launchBot(botConfig)
		if err != nil {
			botConfig.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			t.safeLog(LogLevelError, botConfig.instance, fmt.Sprintf("Failed to launch: %v", err))
		} else {
			successCount++
			botConfig.statusLabel.SetText(fmt.Sprintf("Running: %s", botConfig.selectedRoutine))
		}
	}

	// Update UI state
	t.launchBtn.Disable()
	t.stopBtn.Enable()
	t.setAllBtn.Disable()

	t.statusLabel.SetText(fmt.Sprintf("Launched %d/%d bots successfully", successCount, len(t.botConfigs)))

	// Start status polling for real-time updates
	t.startStatusPolling()
}

// launchBot launches a single bot instance
func (t *BotLauncherTab) launchBot(config *BotLaunchConfig) error {
	// Create bot via manager (gets shared registries)
	b, err := t.manager.CreateBot(config.instance)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Store bot reference
	t.runningBots[config.instance] = b

	// Prepare routine request if one is selected
	// Convert display text to filename
	var routineName string
	if config.selectedRoutine != "<none>" && config.selectedRoutine != "" {
		if filename, ok := t.displayToFilename[config.selectedRoutine]; ok {
			routineName = filename
		} else {
			routineName = config.selectedRoutine // Fallback
		}
	}

	// Send to coordinator for account injection and execution
	request := &coordinator.BotRequest{
		Instance:    config.instance,
		RoutineName: routineName,
		Bot:         b,
	}

	// Coordinator will handle account injection and routine execution
	if err := t.coordinator.SubmitBotRequest(request); err != nil {
		return fmt.Errorf("failed to submit to coordinator: %w", err)
	}

	// Log with display name
	displayName := config.selectedRoutine
	if routineName != "" && routineName != config.selectedRoutine {
		displayName = fmt.Sprintf("%s [%s]", config.selectedRoutine, routineName)
	}
	t.safeLog(LogLevelInfo, config.instance, fmt.Sprintf("Launched with routine: %s", displayName))

	// Enable control buttons for this bot
	t.updateBotButtons(config.instance)

	return nil
}

// stopAllBots stops all running bots
func (t *BotLauncherTab) stopAllBots() {
	// Stop status polling first
	t.stopStatusPolling()

	// Stop coordinator
	if t.coordinator != nil {
		t.coordinator.StopAll()
	}

	// Shutdown all bots via manager
	if t.manager != nil {
		t.manager.ShutdownAll()
	}

	// Clear running bots
	t.runningBots = make(map[int]*bot.Bot)

	// Update UI state for all bot configs
	for _, config := range t.botConfigs {
		config.statusLabel.SetText("Stopped")
		config.pauseBtn.Disable()
		config.resumeBtn.Disable()
		config.stopBtn.Disable()
		config.restartBtn.Disable()
	}

	t.launchBtn.Enable()
	t.stopBtn.Disable()
	t.setAllBtn.Enable()

	t.statusLabel.SetText("All bots stopped")
	t.safeLog(LogLevelInfo, 0, "All bots stopped")
}

// safeLog safely logs a message
func (t *BotLauncherTab) safeLog(level LogLevel, instance int, message string) {
	if t.controller.logTab != nil {
		t.controller.logTab.AddLog(level, instance, message)
	}
}

// pauseBot pauses a specific bot instance
func (t *BotLauncherTab) pauseBot(instance int) {
	b, exists := t.runningBots[instance]
	if !exists {
		t.safeLog(LogLevelWarn, instance, "Cannot pause: bot not running")
		return
	}

	if b.RoutineController().Pause() {
		t.safeLog(LogLevelInfo, instance, "Paused")
		t.updateBotButtons(instance)
	} else {
		t.safeLog(LogLevelWarn, instance, "Cannot pause: bot not in running state")
	}
}

// resumeBot resumes a specific bot instance
func (t *BotLauncherTab) resumeBot(instance int) {
	b, exists := t.runningBots[instance]
	if !exists {
		t.safeLog(LogLevelWarn, instance, "Cannot resume: bot not running")
		return
	}

	if b.RoutineController().Resume() {
		t.safeLog(LogLevelInfo, instance, "Resumed")
		t.updateBotButtons(instance)
	} else {
		t.safeLog(LogLevelWarn, instance, "Cannot resume: bot not in paused state")
	}
}

// stopBot stops a specific bot instance
func (t *BotLauncherTab) stopBot(instance int) {
	b, exists := t.runningBots[instance]
	if !exists {
		t.safeLog(LogLevelWarn, instance, "Bot not running")
		return
	}

	b.RoutineController().ForceStop()
	t.safeLog(LogLevelInfo, instance, "Stopped")

	// Update button states
	t.updateBotButtons(instance)
}

// restartBot restarts a specific bot instance with its last routine
func (t *BotLauncherTab) restartBot(instance int) {
	// Get the bot and its last routine
	lastRoutine, err := t.manager.RestartBot(instance)
	if err != nil {
		t.safeLog(LogLevelError, instance, fmt.Sprintf("Cannot restart: %v", err))
		return
	}

	// Get the bot instance
	b, exists := t.runningBots[instance]
	if !exists {
		t.safeLog(LogLevelError, instance, "Cannot restart: bot not found")
		return
	}

	t.safeLog(LogLevelInfo, instance, fmt.Sprintf("Restarting with routine: %s", lastRoutine))

	// Create a new request for the coordinator
	request := &coordinator.BotRequest{
		Instance:    instance,
		RoutineName: lastRoutine,
		Bot:         b,
	}

	// Submit to coordinator for execution
	if err := t.coordinator.SubmitBotRequest(request); err != nil {
		t.safeLog(LogLevelError, instance, fmt.Sprintf("Failed to restart: %v", err))
		return
	}

	// Update button states
	t.updateBotButtons(instance)
}

// updateBotButtons updates button states based on bot's routine controller state
func (t *BotLauncherTab) updateBotButtons(instance int) {
	// Find the config for this instance
	var config *BotLaunchConfig
	for _, cfg := range t.botConfigs {
		if cfg.instance == instance {
			config = cfg
			break
		}
	}
	if config == nil {
		return
	}

	b, exists := t.runningBots[instance]
	if !exists {
		// Bot not running - all buttons disabled
		config.pauseBtn.Disable()
		config.resumeBtn.Disable()
		config.stopBtn.Disable()
		config.restartBtn.Disable()
		config.statusLabel.SetText("Not Running")
		config.statusIndicator.FillColor = color.RGBA{R: 128, G: 128, B: 128, A: 255} // Gray
		config.statusIndicator.Refresh()
		return
	}

	state := b.RoutineController().GetState()
	hasLastRoutine := b.GetLastRoutine() != ""

	switch state {
	case bot.StateIdle:
		config.pauseBtn.Disable()
		config.resumeBtn.Disable()
		config.stopBtn.Disable()
		// Enable restart if there's a last routine
		if hasLastRoutine {
			config.restartBtn.Enable()
		} else {
			config.restartBtn.Disable()
		}
		config.statusLabel.SetText("Idle")
		config.statusIndicator.FillColor = color.RGBA{R: 200, G: 200, B: 200, A: 255} // Light gray
		config.statusIndicator.Refresh()

	case bot.StateRunning:
		config.pauseBtn.Enable()
		config.resumeBtn.Disable()
		config.stopBtn.Enable()
		config.restartBtn.Disable() // Can't restart while running
		config.statusLabel.SetText("Running")
		config.statusIndicator.FillColor = color.RGBA{R: 0, G: 200, B: 0, A: 255} // Green
		config.statusIndicator.Refresh()

	case bot.StatePaused:
		config.pauseBtn.Disable()
		config.resumeBtn.Enable()
		config.stopBtn.Enable()
		config.restartBtn.Disable() // Can't restart while paused
		config.statusLabel.SetText("Paused")
		config.statusIndicator.FillColor = color.RGBA{R: 255, G: 165, B: 0, A: 255} // Orange
		config.statusIndicator.Refresh()

	case bot.StateStopped:
		config.pauseBtn.Disable()
		config.resumeBtn.Disable()
		config.stopBtn.Disable()
		// Enable restart if there's a last routine
		if hasLastRoutine {
			config.restartBtn.Enable()
		} else {
			config.restartBtn.Disable()
		}
		config.statusLabel.SetText("Stopped")
		config.statusIndicator.FillColor = color.RGBA{R: 200, G: 0, B: 0, A: 255} // Red
		config.statusIndicator.Refresh()

	case bot.StateCompleted:
		config.pauseBtn.Disable()
		config.resumeBtn.Disable()
		config.stopBtn.Disable()
		// Enable restart if there's a last routine
		if hasLastRoutine {
			config.restartBtn.Enable()
		} else {
			config.restartBtn.Disable()
		}
		config.statusLabel.SetText("Completed")
		config.statusIndicator.FillColor = color.RGBA{R: 0, G: 100, B: 200, A: 255} // Blue
		config.statusIndicator.Refresh()
	}
}

// startStatusPolling starts polling bot status for all configured bots
func (t *BotLauncherTab) startStatusPolling() {
	if t.pollingActive {
		return // Already polling
	}

	t.pollingActive = true
	t.pollingWg.Add(1)

	go func() {
		defer t.pollingWg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-t.pollingStop:
				return
			case <-ticker.C:
				// Poll status for all bot configs
				for _, config := range t.botConfigs {
					t.updateBotButtons(config.instance)
				}
			}
		}
	}()
}

// stopStatusPolling stops the status polling goroutine
func (t *BotLauncherTab) stopStatusPolling() {
	if !t.pollingActive {
		return
	}

	t.pollingActive = false
	close(t.pollingStop)
	t.pollingWg.Wait()

	// Recreate the channel for next time
	t.pollingStop = make(chan struct{})
}

// Cleanup performs cleanup when the tab is closed
func (t *BotLauncherTab) Cleanup() {
	t.stopStatusPolling()
	t.stopAllBots()
}

// reloadRoutines reloads all routine files from disk
func (t *BotLauncherTab) reloadRoutines() {
	if t.manager == nil {
		t.updateStatus("Error: Manager not initialized", true)
		return
	}

	t.updateStatus("Reloading routines...", false)

	err := t.manager.ReloadRoutines()
	if err != nil {
		t.updateStatus(fmt.Sprintf("Failed to reload routines: %v", err), true)
		dialog.ShowError(fmt.Errorf("reload failed: %w", err), t.controller.window)
		return
	}

	// Reload the available routines list
	t.loadAvailableRoutines()

	// Update all dropdown menus
	for _, config := range t.botConfigs {
		if config.routineSelect != nil {
			config.routineSelect.Options = t.availableRoutines
			config.routineSelect.Refresh()
		}
	}

	t.updateStatus("✓ Routines reloaded successfully", false)

	// Show success dialog with count
	validCount := len(t.availableRoutines)
	dialog.ShowInformation("Reload Complete",
		fmt.Sprintf("Successfully reloaded %d routine(s)", validCount),
		t.controller.window)
}

// reloadTemplates reloads all template files from disk
func (t *BotLauncherTab) reloadTemplates() {
	if t.manager == nil {
		t.updateStatus("Error: Manager not initialized", true)
		return
	}

	t.updateStatus("Reloading templates...", false)

	err := t.manager.ReloadTemplates()
	if err != nil {
		t.updateStatus(fmt.Sprintf("Failed to reload templates: %v", err), true)
		dialog.ShowError(fmt.Errorf("reload failed: %w", err), t.controller.window)
		return
	}

	t.updateStatus("✓ Templates reloaded successfully", false)

	// Show success dialog
	dialog.ShowInformation("Reload Complete",
		"Successfully reloaded all templates from config/templates/",
		t.controller.window)
}
