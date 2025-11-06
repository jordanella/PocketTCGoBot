package gui

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/config"
)

// ConfigTab allows editing bot configuration
type ConfigTab struct {
	controller *Controller

	// Form widgets
	instanceEntry        *widget.Entry
	adbPathEntry         *widget.Entry
	mumuPathEntry        *widget.Entry
	actionsDelayEntry    *widget.Entry
	screenshotDelayEntry *widget.Entry
	windowWidthEntry     *widget.Entry
	windowHeightEntry    *widget.Entry
	enableLoggingCheck   *widget.Check
	logLevelSelect       *widget.Select
	monitorSelect        *widget.Select
	columnsEntry         *widget.Entry
	rowGapEntry          *widget.Entry
}

// NewConfigTab creates a new configuration tab
func NewConfigTab(ctrl *Controller) *ConfigTab {
	return &ConfigTab{
		controller: ctrl,
	}
}

// Build constructs the configuration UI
func (c *ConfigTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Bot Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Load current config
	cfg := c.controller.GetConfig()

	// Create form widgets
	c.instanceEntry = widget.NewEntry()
	c.instanceEntry.SetText(strconv.Itoa(cfg.Instance))

	// Call methods to get config structs
	adbCfg := cfg.ADB()
	mumuCfg := cfg.MuMu()
	actionsCfg := cfg.Actions()
	loggingCfg := cfg.Logging()

	c.adbPathEntry = widget.NewEntry()
	c.adbPathEntry.SetText(adbCfg.Path)

	adbBrowseBtn := widget.NewButton("Browse", func() {
		c.browseForADBPath()
	})

	adbPathContainer := container.NewBorder(nil, nil, nil, adbBrowseBtn, c.adbPathEntry)

	c.mumuPathEntry = widget.NewEntry()
	c.mumuPathEntry.SetText(mumuCfg.Path)

	mumuBrowseBtn := widget.NewButton("Browse", func() {
		c.browseForMuMuPath()
	})

	mumuPathContainer := container.NewBorder(nil, nil, nil, mumuBrowseBtn, c.mumuPathEntry)

	c.actionsDelayEntry = widget.NewEntry()
	c.actionsDelayEntry.SetText(strconv.Itoa(actionsCfg.DelayBetweenActions))

	c.screenshotDelayEntry = widget.NewEntry()
	c.screenshotDelayEntry.SetText(strconv.Itoa(actionsCfg.ScreenshotDelay))

	c.windowWidthEntry = widget.NewEntry()
	c.windowWidthEntry.SetText(strconv.Itoa(mumuCfg.WindowWidth))

	c.windowHeightEntry = widget.NewEntry()
	c.windowHeightEntry.SetText(strconv.Itoa(mumuCfg.WindowHeight))

	c.enableLoggingCheck = widget.NewCheck("", nil)
	c.enableLoggingCheck.SetChecked(loggingCfg.Enabled)

	c.logLevelSelect = widget.NewSelect([]string{"DEBUG", "INFO", "WARN", "ERROR"}, nil)
	c.logLevelSelect.SetSelected(loggingCfg.Level)

	c.monitorSelect = widget.NewSelect([]string{"0", "1", "2", "3"}, nil)
	c.monitorSelect.SetSelected(strconv.Itoa(cfg.SelectedMonitor))

	c.columnsEntry = widget.NewEntry()
	c.columnsEntry.SetText(strconv.Itoa(cfg.Columns))

	c.rowGapEntry = widget.NewEntry()
	c.rowGapEntry.SetText(strconv.Itoa(cfg.RowGap))

	// Build form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Instance Number", Widget: c.instanceEntry},
			{Text: "ADB Path", Widget: adbPathContainer},
			{Text: "MuMu Path", Widget: mumuPathContainer},
			{Text: "Action Delay (ms)", Widget: c.actionsDelayEntry},
			{Text: "Screenshot Delay (ms)", Widget: c.screenshotDelayEntry},
			{Text: "Window Width", Widget: c.windowWidthEntry},
			{Text: "Window Height", Widget: c.windowHeightEntry},
			{Text: "Window Layout Columns", Widget: c.columnsEntry},
			{Text: "Window Layout Row Gap", Widget: c.rowGapEntry},
			{Text: "Monitor Selection", Widget: c.monitorSelect},
			{Text: "Enable Logging", Widget: c.enableLoggingCheck},
			{Text: "Log Level", Widget: c.logLevelSelect},
		},
		OnSubmit: func() {
			c.saveConfigToFile()
		},
		OnCancel: func() {
			c.loadConfig()
		},
		SubmitText: "Save Configuration",
		CancelText: "Reset",
	}

	// Buttons
	saveBtn := widget.NewButton("Save to File", func() {
		c.saveConfigToFile()
	})

	loadBtn := widget.NewButton("Load from File", func() {
		c.loadConfigFromFile()
	})

	buttons := container.NewHBox(saveBtn, loadBtn)

	// Scrollable content
	content := container.NewVScroll(
		container.NewVBox(
			header,
			form,
			buttons,
		),
	)

	return content
}

// loadConfig reloads configuration from controller
func (c *ConfigTab) loadConfig() {
	cfg := c.controller.GetConfig()

	// Call methods to get config structs
	adbCfg := cfg.ADB()
	mumuCfg := cfg.MuMu()
	actionsCfg := cfg.Actions()
	loggingCfg := cfg.Logging()

	c.instanceEntry.SetText(strconv.Itoa(cfg.Instance))
	c.adbPathEntry.SetText(adbCfg.Path)
	c.mumuPathEntry.SetText(mumuCfg.Path)
	c.actionsDelayEntry.SetText(strconv.Itoa(actionsCfg.DelayBetweenActions))
	c.screenshotDelayEntry.SetText(strconv.Itoa(actionsCfg.ScreenshotDelay))
	c.windowWidthEntry.SetText(strconv.Itoa(mumuCfg.WindowWidth))
	c.windowHeightEntry.SetText(strconv.Itoa(mumuCfg.WindowHeight))
	c.columnsEntry.SetText(strconv.Itoa(cfg.Columns))
	c.rowGapEntry.SetText(strconv.Itoa(cfg.RowGap))
	c.monitorSelect.SetSelected(strconv.Itoa(cfg.SelectedMonitor))
	c.enableLoggingCheck.SetChecked(loggingCfg.Enabled)
	c.logLevelSelect.SetSelected(loggingCfg.Level)
}

// saveConfig saves configuration to controller
func (c *ConfigTab) saveConfig() {
	cfg := c.controller.GetConfig()

	// Parse and validate inputs
	instance, err := strconv.Atoi(c.instanceEntry.Text)
	if err != nil {
		log.Printf("Invalid instance number: %v", err)
		return
	}

	actionsDelay, err := strconv.Atoi(c.actionsDelayEntry.Text)
	if err != nil {
		log.Printf("Invalid actions delay: %v", err)
		return
	}

	screenshotDelay, err := strconv.Atoi(c.screenshotDelayEntry.Text)
	if err != nil {
		log.Printf("Invalid screenshot delay: %v", err)
		return
	}

	windowWidth, err := strconv.Atoi(c.windowWidthEntry.Text)
	if err != nil {
		log.Printf("Invalid window width: %v", err)
		return
	}

	windowHeight, err := strconv.Atoi(c.windowHeightEntry.Text)
	if err != nil {
		log.Printf("Invalid window height: %v", err)
		return
	}

	columns, err := strconv.Atoi(c.columnsEntry.Text)
	if err != nil {
		log.Printf("Invalid columns: %v", err)
		return
	}

	rowGap, err := strconv.Atoi(c.rowGapEntry.Text)
	if err != nil {
		log.Printf("Invalid row gap: %v", err)
		return
	}

	monitor, err := strconv.Atoi(c.monitorSelect.Selected)
	if err != nil {
		log.Printf("Invalid monitor: %v", err)
		return
	}

	// Update config using setter methods
	cfg.Instance = instance
	cfg.Columns = columns
	cfg.RowGap = rowGap
	cfg.SelectedMonitor = monitor

	cfg.SetADB(bot.ADBConfig{
		Path: c.adbPathEntry.Text,
	})

	cfg.SetMuMu(bot.MuMuConfig{
		Path:         c.mumuPathEntry.Text,
		WindowWidth:  windowWidth,
		WindowHeight: windowHeight,
	})

	cfg.SetActions(bot.ActionsConfig{
		DelayBetweenActions: actionsDelay,
		ScreenshotDelay:     screenshotDelay,
	})

	cfg.SetLogging(bot.LoggingConfig{
		Enabled: c.enableLoggingCheck.Checked,
		Level:   c.logLevelSelect.Selected,
	})

	c.controller.UpdateConfig(cfg)

	log.Println("Configuration updated")
}

// saveConfigToFile saves configuration to Settings.ini
func (c *ConfigTab) saveConfigToFile() {
	log.Println("[ConfigTab] saveConfigToFile: Starting")
	c.saveConfig() // Save to memory first

	cfg := c.controller.GetConfig()
	log.Printf("[ConfigTab] saveConfigToFile: Saving config to Settings.ini - ADBPath=%s, FolderPath=%s\n", cfg.ADBPath, cfg.FolderPath)

	err := config.SaveToINI(cfg, "Settings.ini")
	if err != nil {
		log.Printf("[ConfigTab] saveConfigToFile: ERROR - %v\n", err)
		bus := c.controller.GetEventBus()
		bus.Publish(ShowErrorDialog(fmt.Sprintf("Failed to save config: %v", err)))
		bus.Publish(AddLog(LogLevelError, 0, fmt.Sprintf("Config save failed: %v", err)))
		return
	}

	log.Println("[ConfigTab] saveConfigToFile: Success")
	bus := c.controller.GetEventBus()
	bus.Publish(ShowInfoDialog("Success", "Configuration saved to Settings.ini"))
	bus.Publish(AddLog(LogLevelInfo, 0, "Configuration saved to Settings.ini"))
}

// loadConfigFromFile loads configuration from Settings.ini
func (c *ConfigTab) loadConfigFromFile() {
	cfg, err := config.LoadFromINI("Settings.ini", c.controller.GetConfig().Instance)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		// TODO: Show error dialog
		return
	}

	c.controller.UpdateConfig(cfg)
	c.loadConfig()

	log.Println("Configuration loaded from Settings.ini")
	// TODO: Show success dialog
}

// showError displays an error dialog
func (c *ConfigTab) showError(message string) {
	dialog.ShowError(
		fmt.Errorf("%s", message),
		c.controller.window,
	)
}

// showSuccess displays a success dialog
func (c *ConfigTab) ShowSuccess(message string) {
	dialog.ShowInformation(
		"Success",
		message,
		c.controller.window,
	)
}

// browseForADBPath opens a file browser for ADB executable
func (c *ConfigTab) browseForADBPath() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			c.showError(fmt.Sprintf("Error selecting file: %v", err))
			return
		}
		if reader == nil {
			return // User cancelled
		}
		defer reader.Close()

		path := reader.URI().Path()
		c.adbPathEntry.SetText(path)
	}, c.controller.window)

	// Set filter for executables
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".exe"}))

	// Try to start in a reasonable location
	cfg := c.controller.GetConfig()
	if cfg.FolderPath != "" {
		// Try to open in MuMu folder
		adbDir := filepath.Join(cfg.FolderPath, "vmonitor", "bin")
		if uri, err := storage.ParseURI("file:///" + filepath.ToSlash(adbDir)); err == nil {
			if lister, err := storage.ListerForURI(uri); err == nil {
				fileDialog.SetLocation(lister)
			}
		}
	}
	fileDialog.Resize(c.controller.window.Canvas().Size())
	fileDialog.Show()
}

// browseForMuMuPath opens a folder browser for MuMu installation
func (c *ConfigTab) browseForMuMuPath() {
	folderDialog := dialog.NewFolderOpen(func(folder fyne.ListableURI, err error) {
		if err != nil {
			c.showError(fmt.Sprintf("Error selecting folder: %v", err))
			return
		}
		if folder == nil {
			return // User cancelled
		}

		path := folder.Path()
		c.mumuPathEntry.SetText(path)
	}, c.controller.window)

	// Try to start in Program Files
	if uri, err := storage.ParseURI("file:///C:/Program Files"); err == nil {
		if lister, err := storage.ListerForURI(uri); err == nil {
			folderDialog.SetLocation(lister)
		}
	}

	folderDialog.Show()
}
