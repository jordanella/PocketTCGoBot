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
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

// ManagerGroupsTab allows creating and managing multiple bot manager groups
type ManagerGroupsTab struct {
	controller *Controller

	// Global registries (loaded once, shared by all managers)
	templateRegistry *templates.TemplateRegistry
	routineRegistry  *actions.RoutineRegistry

	// Manager groups
	groups   map[string]*ManagerGroup
	groupsMu sync.RWMutex

	// UI components
	groupsContainer *fyne.Container
	addGroupBtn     *widget.Button
	refreshBtn      *widget.Button
	statusLabel     *widget.Label

	// Available routines for dropdown
	availableRoutines []string
	displayToFilename map[string]string
}

// ManagerGroup represents a single manager with its bots and account pool
type ManagerGroup struct {
	Name         string
	Manager      *bot.Manager
	RoutineName  string
	InstanceIDs  []int
	AccountsPath string
	PoolConfig   accountpool.PoolConfig
	AccountPool  accountpool.AccountPool

	// UI components
	card           *fyne.Container
	statusLabel    *widget.Label
	poolStatsLabel *widget.Label
	startBtn       *widget.Button
	stopBtn        *widget.Button

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
	// Initialize global registries
	t.initializeGlobalRegistries()

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

// initializeGlobalRegistries loads templates and routines once for all managers
func (t *ManagerGroupsTab) initializeGlobalRegistries() {
	// Load templates
	templatesPath := filepath.Join(".", "templates")
	t.templateRegistry = templates.NewTemplateRegistry(templatesPath)
	if err := t.templateRegistry.LoadFromDirectory(filepath.Join(templatesPath, "registry")); err != nil {
		t.controller.logTab.AddLog(LogLevelWarn, 0, fmt.Sprintf("Manager Groups: Failed to load templates: %v", err))
	} else {
		t.controller.logTab.AddLog(LogLevelInfo, 0, "Manager Groups: Loaded global template registry from "+templatesPath)
	}

	// Load routines
	routinesPath := filepath.Join(".", "routines")
	t.routineRegistry = actions.NewRoutineRegistry(routinesPath).WithTemplateRegistry(t.templateRegistry)
	t.controller.logTab.AddLog(LogLevelInfo, 0, "Manager Groups: Loaded global routine registry from "+routinesPath)

	// Load available routines for dropdown
	t.loadAvailableRoutines()
}

// loadAvailableRoutines gets list of available routines
func (t *ManagerGroupsTab) loadAvailableRoutines() {
	if t.routineRegistry == nil {
		return
	}

	t.availableRoutines = t.routineRegistry.ListAvailable()
	t.displayToFilename = make(map[string]string)

	for _, filename := range t.availableRoutines {
		// Get metadata for display name
		metadata := t.routineRegistry.GetMetadata(filename)
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
		widget.NewLabelWithStyle("Account Pool (Optional)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Accounts Directory:"),
		container.NewHBox(accountsBtn, accountsLabel),
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
	name, routineDisplay, instancesStr, accountsPath,
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

	// Parse pool config
	minPacks, _ := strconv.Atoi(minPacksStr)
	maxPacks, _ := strconv.Atoi(maxPacksStr)
	maxFailures, _ := strconv.Atoi(maxFailuresStr)
	if maxFailures == 0 {
		maxFailures = 3
	}

	sortMethod := t.parseSortMethod(sortMethodStr)

	poolConfig := accountpool.PoolConfig{
		MinPacks:        minPacks,
		MaxPacks:        maxPacks,
		SortMethod:      sortMethod,
		RetryFailed:     retryFailed,
		MaxFailures:     maxFailures,
		WaitForAccounts: true,
		MaxWaitTime:     5 * time.Minute,
		BufferSize:      100,
	}

	// Create manager with global registries
	manager := bot.NewManagerWithRegistries(
		t.controller.config,
		t.templateRegistry,
		t.routineRegistry,
	)

	// Create account pool if path provided
	var pool accountpool.AccountPool
	if accountsPath != "" {
		pool, err = accountpool.NewFileAccountPool(accountsPath, poolConfig)
		if err != nil {
			return fmt.Errorf("failed to create account pool: %w", err)
		}
		manager.SetAccountPool(pool)
	}

	// Create group
	group := &ManagerGroup{
		Name:         name,
		Manager:      manager,
		RoutineName:  routineName,
		InstanceIDs:  instanceIDs,
		AccountsPath: accountsPath,
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

	poolLabel := widget.NewLabel("Accounts: None")
	if group.AccountsPath != "" {
		poolLabel.SetText(fmt.Sprintf("Accounts: %s", filepath.Base(group.AccountsPath)))
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

	deleteBtn := widget.NewButton("Delete", func() {
		t.deleteGroup(group)
	})

	controls := container.NewHBox(
		group.startBtn,
		group.stopBtn,
		layout.NewSpacer(),
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
