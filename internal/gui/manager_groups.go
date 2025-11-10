package gui

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/bot"
)

// ManagerGroupsTab allows creating and managing multiple bot manager groups
type ManagerGroupsTab struct {
	controller *Controller

	// Manager groups
	groups   map[string]*ManagerGroup
	groupsMu sync.RWMutex

	// UI components
	groupsContainer *fyne.Container
	addGroupBtn     *widget.Button
	refreshBtn      *widget.Button
	statusLabel     *widget.Label

	// Available routines for dropdown (cached from registry)
	availableRoutines []string
	displayToFilename map[string]string
}

// ManagerGroup represents a single manager with its bots and account pool
type ManagerGroup struct {
	Name         string
	Manager      *bot.Manager
	RoutineName  string
	InstanceIDs  []int
	AccountsPath string // Legacy: file-based pool path
	PoolName     string // Name of selected pool from PoolManager
	PoolConfig   accountpool.PoolConfig
	AccountPool  accountpool.AccountPool

	// UI components
	card           *fyne.Container
	statusLabel    *widget.Label
	poolStatsLabel *widget.Label
	startBtn       *widget.Button
	stopBtn        *widget.Button
	refreshPoolBtn *widget.Button

	// Runtime state
	running bool
}

// NewManagerGroupsTab creates a new manager groups tab
func NewManagerGroupsTab(ctrl *Controller) *ManagerGroupsTab {
	return &ManagerGroupsTab{
		controller:        ctrl,
		groups:            make(map[string]*ManagerGroup),
		displayToFilename: make(map[string]string),
	}
}

// Build constructs the UI
func (t *ManagerGroupsTab) Build() fyne.CanvasObject {
	// Load available routines from Controller's registry
	t.loadAvailableRoutines()

	// Header
	header := widget.NewLabelWithStyle("Manager Groups", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	description := widget.NewLabel("Create and manage bot groups with shared configurations and account pools")

	// Control buttons
	t.addGroupBtn = widget.NewButton("Create New Group", func() {
		t.showCreateGroupDialog()
	})
	t.addGroupBtn.Importance = widget.HighImportance

	t.refreshBtn = widget.NewButton("Refresh All", func() {
		t.refreshAllGroups()
	})

	t.statusLabel = widget.NewLabel("No groups created")

	controls := container.NewHBox(
		t.addGroupBtn,
		t.refreshBtn,
		layout.NewSpacer(),
		t.statusLabel,
	)

	// Groups container (will hold group cards)
	t.groupsContainer = container.NewVBox()

	// Wrap in scroll container
	scroll := container.NewVScroll(t.groupsContainer)
	scroll.SetMinSize(fyne.NewSize(800, 500))

	// Main layout
	content := container.NewBorder(
		container.NewVBox(header, description, widget.NewSeparator(), controls),
		nil,
		nil,
		nil,
		scroll,
	)

	// Start periodic refresh
	go t.startPeriodicRefresh()

	return content
}

// loadAvailableRoutines gets list of available routines from Controller's registry
func (t *ManagerGroupsTab) loadAvailableRoutines() {
	routineRegistry := t.controller.GetRoutineRegistry()
	if routineRegistry == nil {
		return
	}

	t.availableRoutines = routineRegistry.ListAvailable()
	t.displayToFilename = make(map[string]string)

	for _, filename := range t.availableRoutines {
		// Get metadata for display name
		metadata := routineRegistry.GetMetadata(filename)
		if metadata != nil {
			if m, ok := metadata.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					displayName := fmt.Sprintf("%s (%s)", name, filename)
					t.displayToFilename[displayName] = filename
				}
			}
		}
	}
}

// showCreateGroupDialog shows dialog to create a new group
func (t *ManagerGroupsTab) showCreateGroupDialog() {
	// Form fields
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("e.g., Premium Farmers")

	// Use Entry instead of Select to allow manual routine name entry
	routineEntry := widget.NewEntry()
	routineEntry.SetPlaceHolder("e.g., farm_premium_packs.yaml")

	// Show available routines as helper text
	var availableRoutinesText string
	if len(t.availableRoutines) > 0 {
		availableRoutinesText = fmt.Sprintf("Available: %s", strings.Join(t.availableRoutines, ", "))
	} else {
		availableRoutinesText = "No routines found in ./routines directory"
	}
	routineHelpLabel := widget.NewLabel(availableRoutinesText)
	routineHelpLabel.Wrapping = fyne.TextWrapWord

	instancesEntry := widget.NewEntry()
	instancesEntry.SetPlaceHolder("e.g., 1-4 or 1,2,3,4")
	instancesEntry.SetText("1-4")

	// Pool selection
	var poolOptions []string
	poolOptions = append(poolOptions, "(None - No Account Pool)")
	poolOptions = append(poolOptions, "(Legacy - File Browser)")

	// Get poolManager from Controller
	poolManager := t.controller.poolManager
	if poolManager != nil {
		if err := poolManager.DiscoverPools(); err == nil {
			pools := poolManager.ListPools()
			for _, poolName := range pools {
				poolOptions = append(poolOptions, poolName)
			}
		}
	}

	poolSelect := widget.NewSelect(poolOptions, nil)
	poolSelect.SetSelected("(None - No Account Pool)")

	// Legacy file browser (hidden by default, shown when "Legacy - File Browser" is selected)
	var accountsPath string
	accountsLabel := widget.NewLabel("No directory selected")
	accountsBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				accountsPath = uri.Path()
				accountsLabel.SetText(filepath.Base(accountsPath))
			}
		}, t.controller.window)
	})
	legacyFileContainer := container.NewHBox(accountsBtn, accountsLabel)
	legacyFileContainer.Hide()

	// Legacy pool config fields (hidden by default, shown when using file browser)
	minPacksEntry := widget.NewEntry()
	minPacksEntry.SetPlaceHolder("0")
	minPacksEntry.SetText("0")

	maxPacksEntry := widget.NewEntry()
	maxPacksEntry.SetPlaceHolder("0 = unlimited")
	maxPacksEntry.SetText("0")

	sortMethodSelect := widget.NewSelect([]string{
		"Modified Ascending (oldest first)",
		"Modified Descending (newest first)",
		"Packs Ascending (fewest first)",
		"Packs Descending (most first)",
	}, nil)
	sortMethodSelect.SetSelected("Packs Descending (most first)")

	retryCheck := widget.NewCheck("Retry failed accounts", nil)
	retryCheck.Checked = true

	maxFailuresEntry := widget.NewEntry()
	maxFailuresEntry.SetPlaceHolder("3")
	maxFailuresEntry.SetText("3")

	legacyConfigContainer := container.NewVBox(
		widget.NewLabel("Minimum Packs:"),
		minPacksEntry,
		widget.NewLabel("Maximum Packs:"),
		maxPacksEntry,
		widget.NewLabel("Sort Method:"),
		sortMethodSelect,
		retryCheck,
		widget.NewLabel("Max Retry Attempts:"),
		maxFailuresEntry,
	)
	legacyConfigContainer.Hide()

	// Show/hide legacy fields based on pool selection
	poolSelect.OnChanged = func(selected string) {
		if selected == "(Legacy - File Browser)" {
			legacyFileContainer.Show()
			legacyConfigContainer.Show()
		} else {
			legacyFileContainer.Hide()
			legacyConfigContainer.Hide()
		}
	}

	// Form layout
	form := container.NewVBox(
		widget.NewLabelWithStyle("Group Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Group Name:"),
		nameEntry,
		widget.NewLabel("Routine:"),
		routineEntry,
		routineHelpLabel,
		widget.NewLabel("Bot Instances:"),
		instancesEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Account Pool", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Select Pool:"),
		poolSelect,
		legacyFileContainer,
		legacyConfigContainer,
	)

	// Create dialog
	formDialog := dialog.NewCustomConfirm("Create Manager Group", "Create", "Cancel",
		container.NewVScroll(form),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Validate and create group
			if err := t.createGroup(
				nameEntry.Text,
				routineEntry.Text,
				instancesEntry.Text,
				poolSelect.Selected,
				accountsPath,
				minPacksEntry.Text,
				maxPacksEntry.Text,
				sortMethodSelect.Selected,
				retryCheck.Checked,
				maxFailuresEntry.Text,
			); err != nil {
				dialog.ShowError(err, t.controller.window)
			}
		},
		t.controller.window,
	)

	formDialog.Resize(fyne.NewSize(500, 700))
	formDialog.Show()
}

// createGroup creates a new manager group
func (t *ManagerGroupsTab) createGroup(
	name, routineDisplay, instancesStr, poolSelection, accountsPath,
	minPacksStr, maxPacksStr, sortMethodStr string,
	retryFailed bool, maxFailuresStr string,
) error {
	// Validate inputs
	if name == "" {
		return fmt.Errorf("group name is required")
	}
	if routineDisplay == "" {
		return fmt.Errorf("routine name is required")
	}
	if instancesStr == "" {
		return fmt.Errorf("bot instances are required")
	}

	// Check if group already exists
	t.groupsMu.RLock()
	if _, exists := t.groups[name]; exists {
		t.groupsMu.RUnlock()
		return fmt.Errorf("group '%s' already exists", name)
	}
	t.groupsMu.RUnlock()

	// Get routine filename
	routineName := t.displayToFilename[routineDisplay]
	if routineName == "" {
		routineName = routineDisplay // Fallback to selected value
	}

	// Parse instance IDs
	instanceIDs, err := t.parseInstanceIDs(instancesStr)
	if err != nil {
		return fmt.Errorf("invalid instance IDs: %w", err)
	}

	// Create manager with Controller's registries (MVC: injecting Model into Manager)
	manager := bot.NewManagerWithRegistries(
		t.controller.config,
		t.controller.GetTemplateRegistry(),
		t.controller.GetRoutineRegistry(),
	)

	// Create account pool based on selection
	var pool accountpool.AccountPool
	var poolName string
	var poolConfig accountpool.PoolConfig

	if poolSelection != "(None - No Account Pool)" && poolSelection != "" {
		if poolSelection == "(Legacy - File Browser)" {
			// Legacy file-based pool
			if accountsPath == "" {
				return fmt.Errorf("accounts directory is required for legacy file pool")
			}

			minPacks, _ := strconv.Atoi(minPacksStr)
			maxPacks, _ := strconv.Atoi(maxPacksStr)
			maxFailures, _ := strconv.Atoi(maxFailuresStr)
			if maxFailures == 0 {
				maxFailures = 3
			}

			sortMethod := t.parseSortMethod(sortMethodStr)

			poolConfig = accountpool.PoolConfig{
				MinPacks:        minPacks,
				MaxPacks:        maxPacks,
				SortMethod:      sortMethod,
				RetryFailed:     retryFailed,
				MaxFailures:     maxFailures,
				WaitForAccounts: true,
				MaxWaitTime:     5 * time.Minute,
				BufferSize:      100,
			}

			pool, err = accountpool.NewFileAccountPool(accountsPath, poolConfig)
			if err != nil {
				return fmt.Errorf("failed to create file account pool: %w", err)
			}
		} else {
			// Pool from PoolManager (get from Controller)
			poolManager := t.controller.poolManager
			if poolManager == nil {
				return fmt.Errorf("pool manager not available (database not initialized)")
			}

			pool, err = poolManager.GetPool(poolSelection)
			if err != nil {
				return fmt.Errorf("failed to get pool '%s': %w", poolSelection, err)
			}
			poolName = poolSelection
		}

		manager.SetAccountPool(pool)
	}

	// Create group
	group := &ManagerGroup{
		Name:         name,
		Manager:      manager,
		RoutineName:  routineName,
		InstanceIDs:  instanceIDs,
		AccountsPath: accountsPath, // Only used for legacy file pools
		PoolName:     poolName,     // Name of pool from PoolManager
		PoolConfig:   poolConfig,
		AccountPool:  pool,
		running:      false,
	}

	// Create UI card for group
	group.card = t.createGroupCard(group)

	// Add to groups
	t.groupsMu.Lock()
	t.groups[name] = group
	t.groupsMu.Unlock()

	// Add card to UI
	t.groupsContainer.Add(group.card)
	t.groupsContainer.Refresh()

	t.updateStatusLabel()
	t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Created group '%s' with %d bots running routine '%s'",
		name, len(instanceIDs), routineName))

	return nil
}

// createGroupCard creates a UI card for a manager group
func (t *ManagerGroupsTab) createGroupCard(group *ManagerGroup) *fyne.Container {
	// Header
	nameLabel := widget.NewLabelWithStyle(group.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Status indicator
	statusCircle := canvas.NewCircle(color.RGBA{200, 200, 200, 255})
	statusCircle.Resize(fyne.NewSize(12, 12))
	statusCircle.StrokeWidth = 2
	statusCircle.StrokeColor = color.RGBA{100, 100, 100, 255}

	group.statusLabel = widget.NewLabel("Stopped")

	// Info labels
	routineLabel := widget.NewLabel(fmt.Sprintf("Routine: %s", group.RoutineName))
	instancesLabel := widget.NewLabel(fmt.Sprintf("Bots: %v", group.InstanceIDs))

	// Pool information
	poolLabel := widget.NewLabel("Accounts: None")
	if group.PoolName != "" {
		poolLabel.SetText(fmt.Sprintf("Pool: %s", group.PoolName))
	} else if group.AccountsPath != "" {
		poolLabel.SetText(fmt.Sprintf("Accounts: %s (Legacy)", filepath.Base(group.AccountsPath)))
	}

	group.poolStatsLabel = widget.NewLabel("Pool: Not started")

	// Control buttons
	group.startBtn = widget.NewButton("Start", func() {
		t.startGroup(group)
	})
	group.startBtn.Importance = widget.HighImportance

	group.stopBtn = widget.NewButton("Stop", func() {
		t.stopGroup(group)
	})
	group.stopBtn.Importance = widget.DangerImportance
	group.stopBtn.Disable()

	// Refresh pool button (only for named pools from PoolManager)
	group.refreshPoolBtn = widget.NewButton("Refresh Pool", func() {
		t.refreshGroupPool(group)
	})
	if group.PoolName == "" {
		group.refreshPoolBtn.Hide()
	}

	editBtn := widget.NewButton("Edit", func() {
		t.editGroup(group)
	})

	deleteBtn := widget.NewButton("Delete", func() {
		t.deleteGroup(group)
	})

	controls := container.NewHBox(
		group.startBtn,
		group.stopBtn,
		group.refreshPoolBtn,
		layout.NewSpacer(),
		editBtn,
		deleteBtn,
	)

	// Layout
	content := container.NewVBox(
		container.NewHBox(statusCircle, nameLabel, layout.NewSpacer(), group.statusLabel),
		widget.NewSeparator(),
		routineLabel,
		instancesLabel,
		poolLabel,
		group.poolStatsLabel,
		widget.NewSeparator(),
		controls,
	)

	// Card with border
	card := container.NewPadded(content)
	return card
}

// startGroup starts all bots in a group
func (t *ManagerGroupsTab) startGroup(group *ManagerGroup) {
	if group.running {
		return
	}

	t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Starting group '%s'...", group.Name))

	// Create bots
	for _, instanceID := range group.InstanceIDs {
		botInstance, err := group.Manager.CreateBot(instanceID)
		if err != nil {
			t.controller.logTab.AddLog(LogLevelError, instanceID, fmt.Sprintf("Error creating bot: %v", err))
			dialog.ShowError(err, t.controller.window)
			return
		}

		// Start routine in background
		go func(b *bot.Bot, id int) {
			t.controller.logTab.AddLog(LogLevelInfo, id, fmt.Sprintf("Bot %d (Group: %s): Starting routine '%s'",
				id, group.Name, group.RoutineName))

			policy := bot.RestartPolicy{
				Enabled:        true,
				MaxRetries:     5,
				InitialDelay:   10 * time.Second,
				MaxDelay:       5 * time.Minute,
				BackoffFactor:  2.0,
				ResetOnSuccess: true,
			}

			if err := group.Manager.ExecuteWithRestart(id, group.RoutineName, policy); err != nil {
				t.controller.logTab.AddLog(LogLevelError, id, fmt.Sprintf("Bot %d (Group: %s): Failed - %v",
					id, group.Name, err))
			} else {
				t.controller.logTab.AddLog(LogLevelInfo, id, fmt.Sprintf("Bot %d (Group: %s): Completed successfully",
					id, group.Name))
			}
		}(botInstance, instanceID)
	}

	group.running = true
	group.statusLabel.SetText("Running")
	group.startBtn.Disable()
	group.stopBtn.Enable()

	t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Group '%s' started with %d bots", group.Name, len(group.InstanceIDs)))
}

// stopGroup stops all bots in a group
func (t *ManagerGroupsTab) stopGroup(group *ManagerGroup) {
	if !group.running {
		return
	}

	t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Stopping group '%s'...", group.Name))

	group.Manager.ShutdownAll()

	group.running = false
	group.statusLabel.SetText("Stopped")
	group.startBtn.Enable()
	group.stopBtn.Disable()

	t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Group '%s' stopped", group.Name))
}

// deleteGroup removes a group
func (t *ManagerGroupsTab) deleteGroup(group *ManagerGroup) {
	// Confirm deletion
	dialog.ShowConfirm("Delete Group",
		fmt.Sprintf("Are you sure you want to delete group '%s'?", group.Name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Stop if running
			if group.running {
				t.stopGroup(group)
			}

			// Close account pool
			if group.AccountPool != nil {
				group.AccountPool.Close()
			}

			// Remove from map
			t.groupsMu.Lock()
			delete(t.groups, group.Name)
			t.groupsMu.Unlock()

			// Remove card from UI
			t.groupsContainer.Remove(group.card)
			t.groupsContainer.Refresh()

			t.updateStatusLabel()
			t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Deleted group '%s'", group.Name))
		},
		t.controller.window,
	)
}

// editGroup shows a dialog to edit an existing group
func (t *ManagerGroupsTab) editGroup(group *ManagerGroup) {
	if group.running {
		dialog.ShowInformation("Cannot Edit", "Please stop the group before editing", t.controller.window)
		return
	}

	// Form fields - pre-populate with current values
	nameEntry := widget.NewEntry()
	nameEntry.SetText(group.Name)
	nameEntry.Disable() // Can't change name (it's the key)

	routineEntry := widget.NewEntry()
	routineEntry.SetPlaceHolder("e.g., farm_premium_packs.yaml")
	routineEntry.SetText(group.RoutineName)

	// Show available routines as helper text
	var availableRoutinesText string
	if len(t.availableRoutines) > 0 {
		availableRoutinesText = fmt.Sprintf("Available: %s", strings.Join(t.availableRoutines, ", "))
	} else {
		availableRoutinesText = "No routines found in ./routines directory"
	}
	routineHelpLabel := widget.NewLabel(availableRoutinesText)
	routineHelpLabel.Wrapping = fyne.TextWrapWord

	instancesEntry := widget.NewEntry()
	instancesEntry.SetPlaceHolder("e.g., 1-4 or 1,2,3,4")
	instancesEntry.SetText(t.formatInstanceIDs(group.InstanceIDs))

	// Pool selection
	var poolOptions []string
	poolOptions = append(poolOptions, "(None - No Account Pool)")
	poolOptions = append(poolOptions, "(Legacy - File Browser)")

	// Get poolManager from Controller
	poolManager := t.controller.poolManager
	if poolManager != nil {
		if err := poolManager.DiscoverPools(); err == nil {
			pools := poolManager.ListPools()
			for _, poolName := range pools {
				poolOptions = append(poolOptions, poolName)
			}
		}
	}

	poolSelect := widget.NewSelect(poolOptions, nil)

	// Set current pool selection
	if group.PoolName != "" {
		poolSelect.SetSelected(group.PoolName)
	} else if group.AccountsPath != "" {
		poolSelect.SetSelected("(Legacy - File Browser)")
	} else {
		poolSelect.SetSelected("(None - No Account Pool)")
	}

	// Legacy file browser
	var accountsPath string = group.AccountsPath
	accountsLabel := widget.NewLabel("No directory selected")
	if group.AccountsPath != "" {
		accountsLabel.SetText(filepath.Base(group.AccountsPath))
	}

	accountsBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				accountsPath = uri.Path()
				accountsLabel.SetText(filepath.Base(accountsPath))
			}
		}, t.controller.window)
	})
	legacyFileContainer := container.NewHBox(accountsBtn, accountsLabel)

	// Legacy pool config fields
	minPacksEntry := widget.NewEntry()
	minPacksEntry.SetText(strconv.Itoa(group.PoolConfig.MinPacks))

	maxPacksEntry := widget.NewEntry()
	maxPacksEntry.SetText(strconv.Itoa(group.PoolConfig.MaxPacks))

	sortMethodSelect := widget.NewSelect([]string{
		"Modified Ascending (oldest first)",
		"Modified Descending (newest first)",
		"Packs Ascending (fewest first)",
		"Packs Descending (most first)",
	}, nil)
	sortMethodSelect.SetSelected(t.formatSortMethod(group.PoolConfig.SortMethod))

	retryCheck := widget.NewCheck("Retry failed accounts", nil)
	retryCheck.SetChecked(group.PoolConfig.RetryFailed)

	maxFailuresEntry := widget.NewEntry()
	maxFailuresEntry.SetText(strconv.Itoa(group.PoolConfig.MaxFailures))

	legacyConfigContainer := container.NewVBox(
		widget.NewLabel("Minimum Packs:"),
		minPacksEntry,
		widget.NewLabel("Maximum Packs:"),
		maxPacksEntry,
		widget.NewLabel("Sort Method:"),
		sortMethodSelect,
		retryCheck,
		widget.NewLabel("Max Retry Attempts:"),
		maxFailuresEntry,
	)

	// Show/hide legacy fields based on pool selection
	if poolSelect.Selected == "(Legacy - File Browser)" {
		legacyFileContainer.Show()
		legacyConfigContainer.Show()
	} else {
		legacyFileContainer.Hide()
		legacyConfigContainer.Hide()
	}

	poolSelect.OnChanged = func(selected string) {
		if selected == "(Legacy - File Browser)" {
			legacyFileContainer.Show()
			legacyConfigContainer.Show()
		} else {
			legacyFileContainer.Hide()
			legacyConfigContainer.Hide()
		}
	}

	// Form layout
	form := container.NewVBox(
		widget.NewLabelWithStyle("Edit Group", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Group Name (cannot be changed):"),
		nameEntry,
		widget.NewLabel("Routine:"),
		routineEntry,
		routineHelpLabel,
		widget.NewLabel("Bot Instances:"),
		instancesEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Account Pool", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Select Pool:"),
		poolSelect,
		legacyFileContainer,
		legacyConfigContainer,
	)

	// Create dialog
	formDialog := dialog.NewCustomConfirm("Edit Manager Group", "Save", "Cancel",
		container.NewVScroll(form),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Update group
			if err := t.updateGroup(
				group,
				routineEntry.Text,
				instancesEntry.Text,
				poolSelect.Selected,
				accountsPath,
				minPacksEntry.Text,
				maxPacksEntry.Text,
				sortMethodSelect.Selected,
				retryCheck.Checked,
				maxFailuresEntry.Text,
			); err != nil {
				dialog.ShowError(err, t.controller.window)
			}
		},
		t.controller.window,
	)

	formDialog.Resize(fyne.NewSize(500, 700))
	formDialog.Show()
}

// updateGroup updates an existing manager group
func (t *ManagerGroupsTab) updateGroup(
	group *ManagerGroup,
	routineDisplay, instancesStr, poolSelection, accountsPath,
	minPacksStr, maxPacksStr, sortMethodStr string,
	retryFailed bool, maxFailuresStr string,
) error {
	// Validate inputs
	if routineDisplay == "" {
		return fmt.Errorf("routine name is required")
	}
	if instancesStr == "" {
		return fmt.Errorf("bot instances are required")
	}

	// Get routine filename
	routineName := t.displayToFilename[routineDisplay]
	if routineName == "" {
		routineName = routineDisplay // Fallback to selected value
	}

	// Parse instance IDs
	instanceIDs, err := t.parseInstanceIDs(instancesStr)
	if err != nil {
		return fmt.Errorf("invalid instance IDs: %w", err)
	}

	// Create new manager with Controller's registries (MVC: injecting Model into Manager)
	newManager := bot.NewManagerWithRegistries(
		t.controller.config,
		t.controller.GetTemplateRegistry(),
		t.controller.GetRoutineRegistry(),
	)

	// Create account pool based on selection
	var pool accountpool.AccountPool
	var poolName string
	var poolConfig accountpool.PoolConfig

	if poolSelection != "(None - No Account Pool)" && poolSelection != "" {
		if poolSelection == "(Legacy - File Browser)" {
			// Legacy file-based pool
			if accountsPath == "" {
				return fmt.Errorf("accounts directory is required for legacy file pool")
			}

			minPacks, _ := strconv.Atoi(minPacksStr)
			maxPacks, _ := strconv.Atoi(maxPacksStr)
			maxFailures, _ := strconv.Atoi(maxFailuresStr)
			if maxFailures == 0 {
				maxFailures = 3
			}

			sortMethod := t.parseSortMethod(sortMethodStr)

			poolConfig = accountpool.PoolConfig{
				MinPacks:        minPacks,
				MaxPacks:        maxPacks,
				SortMethod:      sortMethod,
				RetryFailed:     retryFailed,
				MaxFailures:     maxFailures,
				WaitForAccounts: true,
				MaxWaitTime:     5 * time.Minute,
				BufferSize:      100,
			}

			pool, err = accountpool.NewFileAccountPool(accountsPath, poolConfig)
			if err != nil {
				return fmt.Errorf("failed to create file account pool: %w", err)
			}
		} else {
			// Pool from PoolManager (get from Controller)
			poolManager := t.controller.poolManager
			if poolManager == nil {
				return fmt.Errorf("pool manager not available (database not initialized)")
			}

			pool, err = poolManager.GetPool(poolSelection)
			if err != nil {
				return fmt.Errorf("failed to get pool '%s': %w", poolSelection, err)
			}
			poolName = poolSelection
		}

		newManager.SetAccountPool(pool)
	}

	// Close old pool if different from new
	if group.AccountPool != nil && group.AccountPool != pool {
		group.AccountPool.Close()
	}

	// Update group fields
	group.RoutineName = routineName
	group.InstanceIDs = instanceIDs
	group.AccountsPath = accountsPath
	group.PoolName = poolName
	group.PoolConfig = poolConfig
	group.AccountPool = pool
	group.Manager = newManager

	// Recreate the card to reflect changes
	t.groupsContainer.Remove(group.card)
	group.card = t.createGroupCard(group)
	t.groupsContainer.Add(group.card)
	t.groupsContainer.Refresh()

	t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Updated group '%s'", group.Name))

	return nil
}

// refreshGroupPool refreshes a pool from PoolManager
func (t *ManagerGroupsTab) refreshGroupPool(group *ManagerGroup) {
	if group.PoolName == "" {
		dialog.ShowInformation("No Pool", "This group does not use a managed pool", t.controller.window)
		return
	}

	// Get poolManager from Controller
	poolManager := t.controller.poolManager
	if poolManager == nil {
		dialog.ShowError(fmt.Errorf("pool manager not available"), t.controller.window)
		return
	}

	// Confirm refresh
	dialog.ShowConfirm("Refresh Pool",
		fmt.Sprintf("Refresh pool '%s'? This will re-execute the pool query to get updated accounts.", group.PoolName),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Refresh the pool
			if err := poolManager.RefreshPool(group.PoolName); err != nil {
				dialog.ShowError(fmt.Errorf("failed to refresh pool: %w", err), t.controller.window)
				return
			}

			// Get updated pool instance
			pool, err := poolManager.GetPool(group.PoolName)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to get refreshed pool: %w", err), t.controller.window)
				return
			}

			// Update group's pool
			oldPool := group.AccountPool
			group.AccountPool = pool
			group.Manager.SetAccountPool(pool)

			// Close old pool if it was different
			if oldPool != nil && oldPool != pool {
				oldPool.Close()
			}

			// Update stats display
			stats := pool.GetStats()
			group.poolStatsLabel.SetText(fmt.Sprintf(
				"Pool: %d total | %d available | %d in use | %d completed | %d failed",
				stats.Total, stats.Available, stats.InUse, stats.Completed, stats.Failed,
			))

			t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Refreshed pool '%s' for group '%s'", group.PoolName, group.Name))
			dialog.ShowInformation("Pool Refreshed", fmt.Sprintf("Pool '%s' refreshed successfully.\nTotal accounts: %d", group.PoolName, stats.Total), t.controller.window)
		},
		t.controller.window,
	)
}

// refreshAllGroups updates stats for all groups
func (t *ManagerGroupsTab) refreshAllGroups() {
	t.groupsMu.RLock()
	defer t.groupsMu.RUnlock()

	for _, group := range t.groups {
		if group.AccountPool != nil {
			stats := group.AccountPool.GetStats()
			group.poolStatsLabel.SetText(fmt.Sprintf(
				"Pool: %d total | %d available | %d in use | %d completed | %d failed",
				stats.Total, stats.Available, stats.InUse, stats.Completed, stats.Failed,
			))
		}
	}
}

// startPeriodicRefresh updates group stats every 5 seconds
func (t *ManagerGroupsTab) startPeriodicRefresh() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		t.refreshAllGroups()
	}
}

// Helper functions

func (t *ManagerGroupsTab) parseInstanceIDs(str string) ([]int, error) {
	str = strings.TrimSpace(str)

	// Handle range format: "1-4"
	if strings.Contains(str, "-") {
		parts := strings.Split(str, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range format")
		}
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		if start > end {
			return nil, fmt.Errorf("invalid range: start > end")
		}

		ids := make([]int, 0, end-start+1)
		for i := start; i <= end; i++ {
			ids = append(ids, i)
		}
		return ids, nil
	}

	// Handle comma-separated format: "1,2,3,4"
	parts := strings.Split(str, ",")
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid instance ID: %s", part)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (t *ManagerGroupsTab) parseSortMethod(str string) accountpool.SortMethod {
	switch {
	case strings.Contains(str, "oldest"):
		return accountpool.SortMethodModifiedAsc
	case strings.Contains(str, "newest"):
		return accountpool.SortMethodModifiedDesc
	case strings.Contains(str, "fewest"):
		return accountpool.SortMethodPacksAsc
	case strings.Contains(str, "most"):
		return accountpool.SortMethodPacksDesc
	default:
		return accountpool.SortMethodModifiedAsc
	}
}

func (t *ManagerGroupsTab) formatSortMethod(method accountpool.SortMethod) string {
	switch method {
	case accountpool.SortMethodModifiedAsc:
		return "Modified Ascending (oldest first)"
	case accountpool.SortMethodModifiedDesc:
		return "Modified Descending (newest first)"
	case accountpool.SortMethodPacksAsc:
		return "Packs Ascending (fewest first)"
	case accountpool.SortMethodPacksDesc:
		return "Packs Descending (most first)"
	default:
		return "Modified Ascending (oldest first)"
	}
}

func (t *ManagerGroupsTab) formatInstanceIDs(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	// Check if they're consecutive
	isConsecutive := true
	for i := 1; i < len(ids); i++ {
		if ids[i] != ids[i-1]+1 {
			isConsecutive = false
			break
		}
	}

	if isConsecutive && len(ids) > 1 {
		return fmt.Sprintf("%d-%d", ids[0], ids[len(ids)-1])
	}

	// Otherwise, comma-separated
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.Itoa(id)
	}
	return strings.Join(strs, ",")
}

func (t *ManagerGroupsTab) getRoutineDisplayNames() []string {
	names := make([]string, 0, len(t.displayToFilename))
	for display := range t.displayToFilename {
		names = append(names, display)
	}
	return names
}

func (t *ManagerGroupsTab) updateStatusLabel() {
	t.groupsMu.RLock()
	count := len(t.groups)
	t.groupsMu.RUnlock()

	if count == 0 {
		t.statusLabel.SetText("No groups created")
	} else {
		t.statusLabel.SetText(fmt.Sprintf("%d group(s) configured", count))
	}
}
