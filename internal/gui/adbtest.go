package gui

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/accounts"
	"jordanella.com/pocket-tcg-go/internal/adb"
	"jordanella.com/pocket-tcg-go/internal/emulator"
)

// ADBTestTab provides ADB testing and diagnostics
type ADBTestTab struct {
	controller *Controller

	// Widgets
	adbPathLabel     *widget.Label
	adbVersionLabel  *widget.Label
	adbStatusLabel   *widget.Label
	testButton       *widget.Button
	findADBButton    *widget.Button
	testResultsLabel *widget.Label
	devicesLabel     *widget.Label
	progressBar      *widget.ProgressBarInfinite
	instanceSelect   *widget.Select
	selectedInstance int
}

// NewADBTestTab creates a new ADB test tab
func NewADBTestTab(ctrl *Controller) *ADBTestTab {
	return &ADBTestTab{
		controller:       ctrl,
		selectedInstance: 1, // Default to instance 1
	}
}

// Build constructs the ADB test UI
func (a *ADBTestTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("ADB Diagnostics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Status labels
	a.adbPathLabel = widget.NewLabel("ADB Path: Not configured")
	a.adbVersionLabel = widget.NewLabel("ADB Version: Unknown")
	a.adbStatusLabel = widget.NewLabel("Status: Not tested")
	a.devicesLabel = widget.NewLabel("Devices: None")
	a.testResultsLabel = widget.NewLabel("")

	// Progress bar (hidden by default)
	a.progressBar = widget.NewProgressBarInfinite()
	a.progressBar.Hide()

	// Instance selector - build options dynamically from MuMu configs
	instanceOptions := a.buildInstanceOptions()
	a.instanceSelect = widget.NewSelect(instanceOptions, func(selected string) {
		// Parse instance number from selection (format: "Instance X: Name (port XXXXX)")
		fmt.Sscanf(selected, "Instance %d", &a.selectedInstance)
		a.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Selected instance %d for testing", a.selectedInstance))
	})
	// Try to select Instance 1 by default
	for _, opt := range instanceOptions {
		if strings.HasPrefix(opt, "Instance 1:") || strings.HasPrefix(opt, "Instance 1 (") {
			a.instanceSelect.SetSelected(opt)
			break
		}
	}
	// If Instance 1 not found, select first option
	if a.instanceSelect.Selected == "" && len(instanceOptions) > 0 {
		a.instanceSelect.SetSelected(instanceOptions[0])
	}

	// Buttons
	a.findADBButton = widget.NewButton("Auto-Detect ADB", func() {
		a.autoDetectADB()
	})

	a.testButton = widget.NewButton("Run Full Test", func() {
		a.runFullTest()
	})

	testDevicesBtn := widget.NewButton("List Devices", func() {
		a.testListDevices()
	})

	testConnectBtn := widget.NewButton("Test Connect", func() {
		a.testConnect(a.selectedInstance)
	})

	killServerBtn := widget.NewButton("Kill ADB Server", func() {
		a.killADBServer()
	})

	launchAppBtn := widget.NewButton("Launch PocketTCG", func() {
		a.launchPocketTCG()
	})

	killAppBtn := widget.NewButton("Kill PocketTCG", func() {
		a.killPocketTCG()
	})

	positionWindowBtn := widget.NewButton("Position Window", func() {
		a.positionInstanceWindow()
	})

	extractOBBBtn := widget.NewButton("Extract OBB Data", func() {
		a.extractOBBData()
	})

	extractAppDataBtn := widget.NewButton("Extract App Data", func() {
		a.extractAppData()
	})

	crawlStorageBtn := widget.NewButton("Crawl Storage", func() {
		a.crawlStorage()
	})

	// Button layout
	buttonGrid := container.NewGridWithColumns(2,
		a.findADBButton,
		a.testButton,
		testDevicesBtn,
		testConnectBtn,
		killServerBtn,
		launchAppBtn,
		killAppBtn,
		positionWindowBtn,
		extractOBBBtn,
		extractAppDataBtn,
		crawlStorageBtn,
	)

	// Instance selection section
	instanceSection := container.NewVBox(
		widget.NewLabelWithStyle("Instance Selection", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(
			widget.NewLabel("Select Instance:"),
			a.instanceSelect,
		),
	)

	// Status section
	statusSection := container.NewVBox(
		widget.NewLabelWithStyle("Current Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		a.adbPathLabel,
		a.adbVersionLabel,
		a.adbStatusLabel,
		a.devicesLabel,
	)

	// Test results section
	resultsSection := container.NewVBox(
		widget.NewLabelWithStyle("Test Results", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		a.testResultsLabel,
	)

	// Update initial status
	a.updateStatus()

	// Layout
	content := container.NewVScroll(
		container.NewVBox(
			header,
			widget.NewSeparator(),
			instanceSection,
			widget.NewSeparator(),
			statusSection,
			widget.NewSeparator(),
			buttonGrid,
			a.progressBar,
			widget.NewSeparator(),
			resultsSection,
		),
	)

	return content
}

// updateStatus updates the status display
func (a *ADBTestTab) updateStatus() {
	cfg := a.controller.GetConfig()
	adbCfg := cfg.ADB()

	if adbCfg.Path == "" {
		a.adbPathLabel.SetText("ADB Path: Not configured")
		a.adbStatusLabel.SetText("Status: Not configured")
	} else {
		a.adbPathLabel.SetText(fmt.Sprintf("ADB Path: %s", adbCfg.Path))
		a.adbStatusLabel.SetText("Status: Configured (not tested)")
	}
}

// autoDetectADB attempts to auto-detect ADB
func (a *ADBTestTab) autoDetectADB() {
	log.Println("[ADBTest] autoDetectADB: Starting")
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", "Searching for ADB..."))
	log.Println("[ADBTest] autoDetectADB: Published initial events")

	go func() {
		log.Println("[ADBTest] autoDetectADB: Goroutine started")
		cfg := a.controller.GetConfig()
		log.Printf("[ADBTest] autoDetectADB: Searching in folder: %s\n", cfg.FolderPath)

		adbPath, err := adb.FindADB(cfg.FolderPath)
		log.Printf("[ADBTest] autoDetectADB: FindADB returned: path=%s, err=%v\n", adbPath, err)

		bus.Publish(HideProgressBar("adbtest"))
		log.Println("[ADBTest] autoDetectADB: Published HideProgressBar")

		if err != nil {
			log.Printf("[ADBTest] autoDetectADB: Error - %v\n", err)
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to find ADB: %v", err)))
			bus.Publish(AddLog(LogLevelError, 0, fmt.Sprintf("ADB auto-detect failed: %v", err)))
			return
		}

		log.Printf("[ADBTest] autoDetectADB: Success - found at %s\n", adbPath)
		bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("✓ Found ADB at: %s", adbPath)))
		bus.Publish(UpdateLabel("adbtest.path", fmt.Sprintf("ADB Path: %s", adbPath)))
		bus.Publish(AddLog(LogLevelInfo, 0, fmt.Sprintf("ADB found at: %s", adbPath)))

		// Update config
		cfg.ADBPath = adbPath
		a.controller.UpdateConfig(cfg)
		log.Println("[ADBTest] autoDetectADB: Completed")
	}()
}

// runFullTest runs a comprehensive ADB test
func (a *ADBTestTab) runFullTest() {
	log.Println("[ADBTest] runFullTest: Starting")
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", "Running full ADB test suite...\n"))
	log.Println("[ADBTest] runFullTest: Published initial events")

	go func() {
		log.Println("[ADBTest] runFullTest: Goroutine started")
		results := []string{}
		cfg := a.controller.GetConfig()
		adbCfg := cfg.ADB()
		log.Printf("[ADBTest] runFullTest: ADB path: %s\n", adbCfg.Path)

		// Test 1: Check if ADB path exists
		log.Println("[ADBTest] runFullTest: Test 1 - ADB Executable Check")
		results = append(results, "Test 1: ADB Executable Check")
		if adbCfg.Path == "" {
			log.Println("[ADBTest] runFullTest: Test 1 - No ADB path configured")
			results = append(results, "  ❌ No ADB path configured")
			bus.Publish(HideProgressBar("adbtest"))
			bus.Publish(UpdateLabel("adbtest.results", strings.Join(results, "\n")))
			return
		}
		results = append(results, fmt.Sprintf("  ✓ ADB path: %s", adbCfg.Path))
		log.Println("[ADBTest] runFullTest: Test 1 - Passed")

		// Test 2: Check ADB version with timeout
		log.Println("[ADBTest] runFullTest: Test 2 - ADB Version Check")
		results = append(results, "\nTest 2: ADB Version Check")
		log.Println("[ADBTest] runFullTest: Test 2 - Calling runADBCommandWithTimeout...")
		version, err := a.runADBCommandWithTimeout("version", 5*time.Second)
		log.Printf("[ADBTest] runFullTest: Test 2 - Returned: err=%v, output=%s\n", err, version)
		if err != nil {
			results = append(results, fmt.Sprintf("  ❌ Failed: %v", err))
		} else {
			// Extract version from output
			lines := strings.Split(version, "\n")
			if len(lines) > 0 {
				versionLine := strings.TrimSpace(lines[0])
				results = append(results, fmt.Sprintf("  ✓ %s", versionLine))
				bus.Publish(UpdateLabel("adbtest.version", fmt.Sprintf("ADB Version: %s", versionLine)))
			}
		}
		log.Println("[ADBTest] runFullTest: Test 2 - Completed")

		// Update intermediate results
		bus.Publish(UpdateLabel("adbtest.results", strings.Join(results, "\n")))

		// Test 3: List devices
		results = append(results, "\nTest 3: Device Detection")
		devices, err := a.runADBCommandWithTimeout("devices", 5*time.Second)
		if err != nil {
			results = append(results, fmt.Sprintf("  ❌ Failed: %v", err))
		} else {
			deviceLines := strings.Split(devices, "\n")
			deviceCount := 0
			for _, line := range deviceLines {
				line = strings.TrimSpace(line)
				if line != "" && line != "List of devices attached" && !strings.HasPrefix(line, "*") {
					deviceCount++
					results = append(results, fmt.Sprintf("  ✓ Device: %s", line))
				}
			}
			if deviceCount == 0 {
				results = append(results, "  ⚠ No devices found")
			}

			bus.Publish(UpdateLabel("adbtest.devices", fmt.Sprintf("Devices: %d connected", deviceCount)))
		}

		// Update intermediate results
		bus.Publish(UpdateLabel("adbtest.results", strings.Join(results, "\n")))

		// Test 4: Test connection to port 16416 (MuMu instance 1)
		// Port = MuMuBasePort + (instanceNum * MuMuPortIncrement) = 16384 + (1 * 32) = 16416
		results = append(results, "\nTest 4: Connection Test (Port 16416)")
		connect, err := a.runADBCommandWithTimeout("connect 127.0.0.1:16416", 10*time.Second)
		if err != nil {
			results = append(results, fmt.Sprintf("  ❌ Failed: %v", err))
		} else {
			if strings.Contains(connect, "connected") || strings.Contains(connect, "already connected") {
				results = append(results, "  ✓ Successfully connected to 127.0.0.1:16416")
			} else {
				results = append(results, fmt.Sprintf("  ⚠ Unexpected response: %s", strings.TrimSpace(connect)))
			}
		}

		results = append(results, "\n=== Test Complete ===")

		// Update final results
		finalResults := strings.Join(results, "\n")
		bus.Publish(HideProgressBar("adbtest"))
		bus.Publish(UpdateLabel("adbtest.results", finalResults))
		bus.Publish(UpdateLabel("adbtest.status", "Status: Test completed"))
		bus.Publish(AddLog(LogLevelInfo, 0, "ADB full test completed"))
	}()
}

// testListDevices lists connected ADB devices
func (a *ADBTestTab) testListDevices() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))

	go func() {
		output, err := a.runADBCommandWithTimeout("devices -l", 5*time.Second)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to list devices: %v", err)))
			return
		}

		bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("Connected Devices:\n\n%s", output)))
		bus.Publish(AddLog(LogLevelInfo, 0, "Listed ADB devices"))
	}()
}

// testConnect tests connection to a specific instance
func (a *ADBTestTab) testConnect(instance int) {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))

	go func() {
		// Calculate port the same way as MuMuInstance does
		// Port = MuMuBasePort + (instanceNum * MuMuPortIncrement)
		// Where MuMuBasePort = 16384, MuMuPortIncrement = 32
		port := 16384 + (instance * 32)
		target := fmt.Sprintf("127.0.0.1:%d", port)

		output, err := a.runADBCommandWithTimeout(fmt.Sprintf("connect %s", target), 10*time.Second)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to connect to %s: %v", target, err)))
			bus.Publish(AddLog(LogLevelError, instance, fmt.Sprintf("ADB connect failed: %v", err)))
			return
		}

		if strings.Contains(output, "connected") || strings.Contains(output, "already connected") {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("✓ Successfully connected to %s\n\n%s", target, output)))
			bus.Publish(AddLog(LogLevelInfo, instance, "ADB connection successful"))
		} else {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("⚠ Unexpected response from %s:\n\n%s", target, output)))
			bus.Publish(AddLog(LogLevelWarn, instance, "ADB connection: unexpected response"))
		}
	}()
}

// killADBServer kills the ADB server
func (a *ADBTestTab) killADBServer() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))

	go func() {
		output, err := a.runADBCommandWithTimeout("kill-server", 5*time.Second)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to kill ADB server: %v", err)))
			return
		}

		bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("✓ ADB server killed\n\n%s", output)))
		bus.Publish(UpdateLabel("adbtest.devices", "Devices: Server killed"))
		bus.Publish(AddLog(LogLevelInfo, 0, "ADB server killed"))
	}()
}

// runADBCommandWithTimeout runs an ADB command with a timeout
func (a *ADBTestTab) runADBCommandWithTimeout(args string, timeout time.Duration) (string, error) {
	log.Printf("[ADBTest] runADBCommandWithTimeout: Starting - args='%s', timeout=%v\n", args, timeout)
	cfg := a.controller.GetConfig()
	adbCfg := cfg.ADB()
	log.Printf("[ADBTest] runADBCommandWithTimeout: ADB path='%s'\n", adbCfg.Path)

	if adbCfg.Path == "" {
		log.Println("[ADBTest] runADBCommandWithTimeout: ERROR - ADB path not configured")
		return "", fmt.Errorf("ADB path not configured")
	}

	// Create context with timeout
	log.Println("[ADBTest] runADBCommandWithTimeout: Creating context with timeout")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Split args and create command
	argList := strings.Fields(args)
	log.Printf("[ADBTest] runADBCommandWithTimeout: Command: %s %v\n", adbCfg.Path, argList)
	cmd := exec.CommandContext(ctx, adbCfg.Path, argList...)

	// Run command
	log.Println("[ADBTest] runADBCommandWithTimeout: Executing command...")
	output, err := cmd.CombinedOutput()
	log.Printf("[ADBTest] runADBCommandWithTimeout: Command completed - err=%v, output length=%d\n", err, len(output))

	if ctx.Err() == context.DeadlineExceeded {
		log.Println("[ADBTest] runADBCommandWithTimeout: Command timed out")
		return "", fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil {
		log.Printf("[ADBTest] runADBCommandWithTimeout: Command failed - %v: %s\n", err, string(output))
		return string(output), fmt.Errorf("%v: %s", err, string(output))
	}

	log.Printf("[ADBTest] runADBCommandWithTimeout: Success - output: %s\n", string(output))
	return string(output), nil
}

// launchPocketTCG launches the PocketTCG app
func (a *ADBTestTab) launchPocketTCG() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("Launching PocketTCG app on Instance %d...", a.selectedInstance)))

	go func() {
		// Connect to selected instance
		port := 16384 + (a.selectedInstance * 32)
		target := fmt.Sprintf("127.0.0.1:%d", port)

		_, err := a.runADBCommandWithTimeout(fmt.Sprintf("connect %s", target), 5*time.Second)
		if err != nil {
			bus.Publish(HideProgressBar("adbtest"))
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to connect to %s: %v", target, err)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("ADB connect failed: %v", err)))
			return
		}

		// Launch the app
		// am start -W -n jp.pokemon.pokemontcgp/com.unity3d.player.UnityPlayerActivity -f 0x10018000
		output, err := a.runADBCommandWithTimeout(
			fmt.Sprintf("-s %s shell am start -W -n jp.pokemon.pokemontcgp/com.unity3d.player.UnityPlayerActivity -f 0x10018000", target),
			15*time.Second,
		)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to launch PocketTCG on Instance %d: %v\n\n%s", a.selectedInstance, err, output)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("Failed to launch PocketTCG: %v", err)))
			return
		}

		bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("✓ PocketTCG launched successfully on Instance %d\n\n%s", a.selectedInstance, output)))
		bus.Publish(AddLog(LogLevelInfo, a.selectedInstance, "PocketTCG launched"))
	}()
}

// killPocketTCG kills the PocketTCG app
func (a *ADBTestTab) killPocketTCG() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("Stopping PocketTCG app on Instance %d...", a.selectedInstance)))

	go func() {
		// Connect to selected instance
		port := 16384 + (a.selectedInstance * 32)
		target := fmt.Sprintf("127.0.0.1:%d", port)

		_, err := a.runADBCommandWithTimeout(fmt.Sprintf("connect %s", target), 5*time.Second)
		if err != nil {
			bus.Publish(HideProgressBar("adbtest"))
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to connect to %s: %v", target, err)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("ADB connect failed: %v", err)))
			return
		}

		// Force stop the app
		output, err := a.runADBCommandWithTimeout(
			fmt.Sprintf("-s %s shell am force-stop jp.pokemon.pokemontcgp", target),
			10*time.Second,
		)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to stop PocketTCG on Instance %d: %v\n\n%s", a.selectedInstance, err, output)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("Failed to stop PocketTCG: %v", err)))
			return
		}

		bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("✓ PocketTCG stopped successfully on Instance %d", a.selectedInstance)))
		bus.Publish(AddLog(LogLevelInfo, a.selectedInstance, "PocketTCG stopped"))
	}()
}

// buildInstanceOptions builds the instance dropdown options from MuMu configs
func (a *ADBTestTab) buildInstanceOptions() []string {
	cfg := a.controller.GetConfig()

	// Create MuMu manager to read configs
	mumuMgr := emulator.NewMuMuManager(cfg.FolderPath)

	// Try to read all instance configs
	configs, err := mumuMgr.GetAllInstanceConfigs()
	if err != nil {
		log.Printf("[ADBTest] Failed to read instance configs: %v\n", err)
		// Fall back to default options
		return []string{
			"Instance 0 (port 16384)",
			"Instance 1 (port 16416)",
			"Instance 2 (port 16448)",
			"Instance 3 (port 16480)",
			"Instance 4 (port 16512)",
			"Instance 5 (port 16544)",
		}
	}

	// Build options list with names
	options := []string{}
	instanceNumbers := []int{}

	// Get sorted list of instance numbers
	for instanceNum := range configs {
		instanceNumbers = append(instanceNumbers, instanceNum)
	}
	sort.Ints(instanceNumbers)

	// Build option strings
	for _, instanceNum := range instanceNumbers {
		config := configs[instanceNum]
		port := 16384 + (instanceNum * 32)

		var optionText string
		if config.PlayerName != "" {
			optionText = fmt.Sprintf("Instance %d: %s (port %d)", instanceNum, config.PlayerName, port)
		} else {
			optionText = fmt.Sprintf("Instance %d (port %d)", instanceNum, port)
		}

		options = append(options, optionText)
	}

	// If no instances found, provide defaults
	if len(options) == 0 {
		log.Println("[ADBTest] No instances found in configs, using defaults")
		return []string{
			"Instance 0 (port 16384)",
			"Instance 1 (port 16416)",
			"Instance 2 (port 16448)",
			"Instance 3 (port 16480)",
			"Instance 4 (port 16512)",
			"Instance 5 (port 16544)",
		}
	}

	log.Printf("[ADBTest] Found %d instances with configs\n", len(options))
	return options
}

// positionInstanceWindow positions and resizes the selected instance window
func (a *ADBTestTab) positionInstanceWindow() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("Positioning Instance %d window...", a.selectedInstance)))

	go func() {
		cfg := a.controller.GetConfig()

		// Create MuMu manager
		mumuMgr := emulator.NewMuMuManager(cfg.FolderPath)

		// Discover running instances
		_, err := mumuMgr.FindInstances()
		if err != nil {
			bus.Publish(HideProgressBar("adbtest"))
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to discover instances: %v", err)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("Failed to discover instances: %v", err)))
			return
		}

		// Get the specific instance
		instance, err := mumuMgr.GetInstance(a.selectedInstance)
		if err != nil {
			bus.Publish(HideProgressBar("adbtest"))
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Instance %d not found or not running", a.selectedInstance)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, "Instance not found or not running"))
			return
		}

		// Calculate scale param based on language setting
		scaleParam := 277 // Default Scale100
		if cfg.DefaultLanguage == "Scale125" {
			scaleParam = 287
		}

		// Create window config using settings from config
		windowConfig := emulator.NewWindowConfig(
			cfg.Columns,
			cfg.RowGap,
			scaleParam,
			cfg.SelectedMonitor,
		)

		// Position the window
		if err := mumuMgr.PositionWindow(instance, windowConfig); err != nil {
			bus.Publish(HideProgressBar("adbtest"))
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to position Instance %d window: %v", a.selectedInstance, err)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("Failed to position window: %v", err)))
			return
		}

		bus.Publish(HideProgressBar("adbtest"))
		bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("✓ Instance %d window positioned successfully\nPosition: (%d, %d)\nSize: %dx%d",
			a.selectedInstance, instance.X, instance.Y, instance.Width, instance.Height)))
		bus.Publish(AddLog(LogLevelInfo, a.selectedInstance, fmt.Sprintf("Window positioned at (%d, %d) with size %dx%d",
			instance.X, instance.Y, instance.Width, instance.Height)))
	}()
}

// extractOBBData extracts OBB data from the device using the accounts package workflow
func (a *ADBTestTab) extractOBBData() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("Extracting OBB data from Instance %d...", a.selectedInstance)))

	go func() {
		// Calculate port for this instance
		port := 16384 + (a.selectedInstance * 32)

		// Create extraction directory
		extractDir := fmt.Sprintf("./extracted_obb/instance_%d", a.selectedInstance)

		// Get ADB path from config
		adbPath := a.controller.GetConfig().ADB().Path

		// Use the accounts package extraction function
		err := accounts.ExtractOBBData(adbPath, port, extractDir)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to extract OBB data from Instance %d: %v", a.selectedInstance, err)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("Failed to extract OBB data: %v", err)))
			return
		}

		resultMsg := fmt.Sprintf("✓ OBB data extracted successfully from Instance %d\n\nLocation: %s\n\nCheck the folder for extracted OBB files.",
			a.selectedInstance, extractDir)

		bus.Publish(UpdateLabel("adbtest.results", resultMsg))
		bus.Publish(AddLog(LogLevelInfo, a.selectedInstance, fmt.Sprintf("OBB data extracted to %s", extractDir)))
	}()
}

// extractAppData extracts app data directory from the device using the accounts package workflow
func (a *ADBTestTab) extractAppData() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("Extracting app data from Instance %d...", a.selectedInstance)))

	go func() {
		// Calculate port for this instance
		port := 16384 + (a.selectedInstance * 32)

		// Create extraction directory
		extractDir := fmt.Sprintf("./extracted_app_data/instance_%d", a.selectedInstance)

		// Get ADB path from config
		adbPath := a.controller.GetConfig().ADB().Path

		// Use the accounts package extraction function
		err := accounts.ExtractAppData(adbPath, port, extractDir)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to extract app data from Instance %d: %v", a.selectedInstance, err)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("Failed to extract app data: %v", err)))
			return
		}

		resultMsg := fmt.Sprintf("✓ App data extracted successfully from Instance %d\n\nLocation: %s\n\nThis includes:\n- Databases (user data, cards, collection)\n- Shared Preferences (settings)\n- Cache files",
			a.selectedInstance, extractDir)

		bus.Publish(UpdateLabel("adbtest.results", resultMsg))
		bus.Publish(AddLog(LogLevelInfo, a.selectedInstance, fmt.Sprintf("App data extracted to %s", extractDir)))
	}()
}

// crawlStorage crawls device storage and outputs directory structure to a file
func (a *ADBTestTab) crawlStorage() {
	bus := a.controller.GetEventBus()

	bus.Publish(ShowProgressBar("adbtest"))
	bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("Crawling storage on Instance %d...\n\nThis may take 30-60 seconds...", a.selectedInstance)))

	go func() {
		// Calculate port for this instance
		port := 16384 + (a.selectedInstance * 32)

		// Create output file
		outputFile := fmt.Sprintf("./storage_crawl_instance_%d.txt", a.selectedInstance)

		// Get ADB path from config
		adbPath := a.controller.GetConfig().ADB().Path

		// Use the accounts package crawl function
		err := accounts.CrawlStorage(adbPath, port, outputFile)

		bus.Publish(HideProgressBar("adbtest"))

		if err != nil {
			bus.Publish(UpdateLabel("adbtest.results", fmt.Sprintf("❌ Failed to crawl storage on Instance %d: %v", a.selectedInstance, err)))
			bus.Publish(AddLog(LogLevelError, a.selectedInstance, fmt.Sprintf("Failed to crawl storage: %v", err)))
			return
		}

		resultMsg := fmt.Sprintf("✓ Storage crawl completed for Instance %d\n\nOutput saved to: %s\n\nOpen this file to see the complete directory structure of the device.",
			a.selectedInstance, outputFile)

		bus.Publish(UpdateLabel("adbtest.results", resultMsg))
		bus.Publish(AddLog(LogLevelInfo, a.selectedInstance, fmt.Sprintf("Storage crawl saved to %s", outputFile)))
	}()
}
