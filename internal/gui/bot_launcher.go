package gui

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
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
}

// BotLaunchConfig represents configuration for a single bot instance
type BotLaunchConfig struct {
	instance        int
	routineSelect   *widget.Select
	statusLabel     *widget.Label
	selectedRoutine string
}

// NewBotLauncherTab creates a new bot launcher tab
func NewBotLauncherTab(ctrl *Controller) *BotLauncherTab {
	return &BotLauncherTab{
		controller:        ctrl,
		runningBots:       make(map[int]*bot.Bot),
		displayToFilename: make(map[string]string),
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

	buttonsRow := container.NewHBox(
		t.setAllBtn,
		t.launchBtn,
		t.stopBtn,
	)

	// Status label
	t.statusLabel = widget.NewLabel("Configure bots to launch")

	// Scrollable content
	content := container.NewVScroll(
		container.NewVBox(
			numBotsRow,
			widget.NewSeparator(),
			buttonsRow,
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
func (t *BotLauncherTab) loadAvailableRoutines() {
	t.availableRoutines = []string{"<none>"}
	t.displayToFilename = make(map[string]string)
	t.displayToFilename["<none>"] = "" // Map <none> to empty string

	// If we have a manager with a routine registry, use it
	if t.manager != nil && t.manager.RoutineRegistry() != nil {
		filenames := t.manager.RoutineRegistry().ListAvailable()

		// Build display strings: "DisplayName (filename)"
		for _, filename := range filenames {
			meta := t.manager.RoutineRegistry().GetMetadata(filename)
			displayText := fmt.Sprintf("%s (%s)", meta.DisplayName, meta.Filename)

			// Check if invalid
			if err := t.manager.RoutineRegistry().GetValidationError(filename); err != nil {
				displayText = fmt.Sprintf("⚠️ %s [INVALID]", displayText)
			}

			t.availableRoutines = append(t.availableRoutines, displayText)
			t.displayToFilename[displayText] = filename
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

	return &BotLaunchConfig{
		instance:        instance,
		routineSelect:   routineSelect,
		statusLabel:     widget.NewLabel("Ready"),
		selectedRoutine: "<none>",
	}
}

// createBotConfigCard creates a UI card for a bot configuration
func (t *BotLauncherTab) createBotConfigCard(config *BotLaunchConfig) fyne.CanvasObject {
	instanceLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("Bot %d", config.instance),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	routineLabel := widget.NewLabel("Routine:")

	card := container.NewBorder(
		instanceLabel,
		config.statusLabel,
		routineLabel,
		nil,
		config.routineSelect,
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

	return nil
}

// stopAllBots stops all running bots
func (t *BotLauncherTab) stopAllBots() {
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

	// Update UI state
	for _, config := range t.botConfigs {
		config.statusLabel.SetText("Stopped")
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

// Cleanup performs cleanup when the tab is closed
func (t *BotLauncherTab) Cleanup() {
	t.stopAllBots()
}
