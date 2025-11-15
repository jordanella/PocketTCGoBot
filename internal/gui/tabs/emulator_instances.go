package tabs

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/emulator"
	"jordanella.com/pocket-tcg-go/internal/gui/components"
)

// EmulatorInstancesTab manages the emulator instances view
type EmulatorInstancesTab struct {
	// Dependencies (injected)
	orchestrator *bot.Orchestrator
	mumuManager  *emulator.MuMuManager
	window       fyne.Window

	// UI state
	groupSections   map[string]*components.GroupSectionCardV2  // key = group name
	instanceCards   map[int]*components.EmulatorInstanceCardV2 // key = instance ID
	groupSectionsMu sync.RWMutex
	instanceCardsMu sync.RWMutex

	// Containers for sections
	activeGroupsContainer      *fyne.Container
	idleInstancesContainer     *fyne.Container
	inactiveInstancesContainer *fyne.Container

	// UI elements
	statusLabel *widget.Label
	refreshBtn  *widget.Button

	// Refresh control
	stopRefresh chan bool
}

// NewEmulatorInstancesTab creates a new emulator instances tab
func NewEmulatorInstancesTab(orchestrator *bot.Orchestrator, mumuManager *emulator.MuMuManager, window fyne.Window) *EmulatorInstancesTab {
	return &EmulatorInstancesTab{
		orchestrator:               orchestrator,
		mumuManager:                mumuManager,
		window:                     window,
		groupSections:              make(map[string]*components.GroupSectionCardV2),
		instanceCards:              make(map[int]*components.EmulatorInstanceCardV2),
		stopRefresh:                make(chan bool),
		activeGroupsContainer:      container.NewVBox(),
		idleInstancesContainer:     container.NewVBox(),
		inactiveInstancesContainer: container.NewVBox(),
	}
}

// Build constructs the tab UI matching the mockup
func (t *EmulatorInstancesTab) Build() fyne.CanvasObject {
	// === HEADER ===
	header := components.Heading("Dashboard")

	// === CONTROLS ===
	t.refreshBtn = components.SecondaryButton("Refresh All", func() {
		t.refreshAll()
	})

	t.statusLabel = widget.NewLabel("Loading...")

	controls := container.NewHBox(
		t.refreshBtn,
		widget.NewLabel(""), // Spacer
		t.statusLabel,
	)

	// === ACTIVE GROUPS SECTION ===
	//activeSection := components.SectionHeader("Active Groups")

	// === IDLE INSTANCES SECTION ===
	//idleSection := components.SectionHeader("Idle Instances")

	// Create the "Idle" parent card
	idleParentCard := components.Card(components.Subheading("Idle"))
	idleContainer := container.NewVBox(
		idleParentCard,
		t.idleInstancesContainer,
	)

	// === INACTIVE INSTANCES SECTION ===
	//inactiveSection := components.SectionHeader("Inactive Instances")

	// Create the "Inactive" parent card
	inactiveParentCard := components.Card(components.Subheading("Inactive"))
	inactiveContainer := container.NewVBox(
		inactiveParentCard,
		t.inactiveInstancesContainer,
	)

	// === MAIN CONTENT ===
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		controls,
		widget.NewSeparator(),
		//activeSection,
		t.activeGroupsContainer,
		widget.NewSeparator(),
		//idleSection,
		idleContainer,
		widget.NewSeparator(),
		//inactiveSection,
		inactiveContainer,
	)

	// Scroll container
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(900, 600))

	// Load existing data
	t.loadExistingData()

	// Start periodic refresh
	go t.startPeriodicRefresh()

	return scroll
}

// loadExistingData loads orchestration groups and instances
func (t *EmulatorInstancesTab) loadExistingData() {
	if t.orchestrator == nil {
		return
	}

	// Clear existing containers and tracking maps
	t.clearAllContainers()

	// Discover MuMu instances (running windows)
	emulatorMgr := t.orchestrator.GetEmulatorManager()
	if emulatorMgr != nil {
		if err := emulatorMgr.DiscoverInstances(); err != nil {
			// Log error but continue
			fmt.Printf("Warning: Failed to discover instances: %v\n", err)
		}
	}

	// Get all detected MuMu instances (running windows)
	detectedInstances := make(map[int]string) // instanceID -> window title
	if emulatorMgr != nil {
		for _, inst := range emulatorMgr.GetAllInstances() {
			if inst.MuMu != nil {
				detectedInstances[inst.MuMu.Index] = inst.MuMu.WindowTitle
			}
		}
	}

	// Get all configured instances from config files
	configuredInstances := make(map[int]string) // instanceID -> player name
	if t.mumuManager != nil {
		if configs, err := t.mumuManager.GetAllInstanceConfigs(); err == nil {
			for instanceID, config := range configs {
				configuredInstances[instanceID] = config.PlayerName
			}
		}
	}

	// Get instance assignments from orchestrator (which instances are running bots)
	assignments := t.orchestrator.GetAllInstanceAssignments()

	// Get all instance IDs from groups (which instances belong to groups)
	instanceToGroups := t.orchestrator.GetAllInstanceIDsFromGroups()

	// Load active orchestration groups
	activeGroups := t.orchestrator.ListActiveGroups()
	for _, group := range activeGroups {
		t.addGroupSection(group, assignments)
	}

	// Categorize remaining instances
	// Track which instances we've already shown
	shownInstances := make(map[int]bool)
	for instanceID := range assignments {
		shownInstances[instanceID] = true
	}

	// Idle: detected windows (running) but not running bots
	for instanceID, windowTitle := range detectedInstances {
		if shownInstances[instanceID] {
			continue
		}

		groupNames := instanceToGroups[instanceID]
		// Add to idle section (detected window, may or may not be in groups)
		t.addIdleInstance(instanceID, windowTitle, groupNames)
		shownInstances[instanceID] = true
	}

	// Inactive: configured but not detected (not running windows)
	for instanceID, playerName := range configuredInstances {
		if shownInstances[instanceID] {
			continue
		}

		// Not detected = not running
		name := fmt.Sprintf("Instance %d", instanceID)
		if playerName != "" {
			name = playerName
		}

		groupNames := instanceToGroups[instanceID]
		t.addInactiveInstance(instanceID, name, groupNames)
	}

	t.updateStatusLabel()
}

// addGroupSection creates and adds a group section card
func (t *EmulatorInstancesTab) addGroupSection(group *bot.BotGroup, assignments map[int]*bot.InstanceAssignment) {
	// Create group section card
	groupCard := components.NewGroupSectionCardV2(
		group.Name,
		group.OrchestrationID,
		components.GroupSectionCardCallbacks{
			OnAddInstance: t.handleAddInstance,
		},
	)

	// Set group info
	groupCard.SetDescription(fmt.Sprintf("Running routine: %s", group.RoutineName))
	// TODO: Set actual started time when available from group
	groupCard.SetStartedAt(time.Now().Add(-time.Hour * 2)) // Placeholder

	// Set pool info
	if group.AccountPool != nil {
		stats := group.AccountPool.GetStats()
		groupCard.SetPoolInfo(group.AccountPoolName, stats.Available, stats.Total)
	}

	// Add instance cards for this group's running bots
	botInfos := group.GetAllBotInfo()
	for instanceID, botInfo := range botInfos {
		instanceCard := t.createActiveInstanceCard(instanceID, group.Name, botInfo)
		groupCard.AddInstanceCard(instanceCard)

		// Track the instance card
		t.instanceCardsMu.Lock()
		t.instanceCards[instanceID] = instanceCard
		t.instanceCardsMu.Unlock()
	}

	// Add to tracking
	t.groupSectionsMu.Lock()
	t.groupSections[group.Name] = groupCard
	t.groupSectionsMu.Unlock()

	// Add to UI
	t.activeGroupsContainer.Add(groupCard.GetContainer())
	t.refreshContainers()
}

// addIdleInstance adds an idle instance card
func (t *EmulatorInstancesTab) addIdleInstance(instanceID int, windowTitle string, groupNames []string) {
	instanceCard := components.NewEmulatorInstanceCardV2(
		instanceID,
		windowTitle,
		components.InstanceStateIdle,
		components.EmulatorInstanceCardCallbacks{
			OnQuickStart: t.handleQuickStart,
			OnShutdown:   t.handleShutdown,
		},
	)

	instanceCard.SetAssociatedGroups(groupNames)

	// Track the instance card
	t.instanceCardsMu.Lock()
	t.instanceCards[instanceID] = instanceCard
	t.instanceCardsMu.Unlock()

	// Add to UI
	t.idleInstancesContainer.Add(instanceCard.GetContainer())
}

// addInactiveInstance adds an inactive instance card
func (t *EmulatorInstancesTab) addInactiveInstance(instanceID int, name string, groupNames []string) {
	instanceCard := components.NewEmulatorInstanceCardV2(
		instanceID,
		name,
		components.InstanceStateInactive,
		components.EmulatorInstanceCardCallbacks{
			OnQuickStart: t.handleQuickStart,
			OnLaunch:     t.handleLaunch,
		},
	)

	instanceCard.SetAssociatedGroups(groupNames)

	// Track the instance card
	t.instanceCardsMu.Lock()
	t.instanceCards[instanceID] = instanceCard
	t.instanceCardsMu.Unlock()

	// Add to UI
	t.inactiveInstancesContainer.Add(instanceCard.GetContainer())
}

// clearAllContainers clears all containers and tracking maps
func (t *EmulatorInstancesTab) clearAllContainers() {
	// Clear tracking maps
	t.groupSectionsMu.Lock()
	t.groupSections = make(map[string]*components.GroupSectionCardV2)
	t.groupSectionsMu.Unlock()

	t.instanceCardsMu.Lock()
	t.instanceCards = make(map[int]*components.EmulatorInstanceCardV2)
	t.instanceCardsMu.Unlock()

	// Clear containers
	t.activeGroupsContainer.Objects = nil
	t.idleInstancesContainer.Objects = nil
	t.inactiveInstancesContainer.Objects = nil

	t.refreshContainers()
}

// createActiveInstanceCard creates an instance card for an active bot
func (t *EmulatorInstancesTab) createActiveInstanceCard(
	instanceID int,
	groupName string,
	botInfo *bot.BotInfo,
) *components.EmulatorInstanceCardV2 {
	// Create instance card
	instanceCard := components.NewEmulatorInstanceCardV2(
		instanceID,
		fmt.Sprintf("Instance %d", instanceID),
		components.InstanceStateActive,
		components.EmulatorInstanceCardCallbacks{
			OnPause:    t.handlePause,
			OnStop:     t.handleStop,
			OnAbort:    t.handleAbort,
			OnShutdown: t.handleShutdown,
		},
	)

	// Set account info if available
	if botInfo != nil && botInfo.Bot != nil {
		// Try to get account name from bot variables
		if accountID, exists := botInfo.Bot.Variables().Get("device_account_id"); exists && accountID != "" {
			instanceCard.SetAccount(accountID, botInfo.StartedAt)
		}

		// Set routine status
		instanceCard.SetRoutineStatus(string(botInfo.Status))
	}

	return instanceCard
}

// removeGroupSection removes a group section card
func (t *EmulatorInstancesTab) removeGroupSection(groupName string) {
	t.groupSectionsMu.Lock()
	defer t.groupSectionsMu.Unlock()

	if groupCard, exists := t.groupSections[groupName]; exists {
		// Remove from UI
		t.activeGroupsContainer.Remove(groupCard.GetContainer())
		delete(t.groupSections, groupName)
		t.refreshContainers()
	}
}

// refreshContainers refreshes all containers
func (t *EmulatorInstancesTab) refreshContainers() {
	fyne.Do(func() {
		t.activeGroupsContainer.Refresh()
		t.idleInstancesContainer.Refresh()
		t.inactiveInstancesContainer.Refresh()
	})
}

// Card action handlers

func (t *EmulatorInstancesTab) handleAddInstance(groupName string) {
	dialog.ShowInformation(
		"Add Instance",
		fmt.Sprintf("Adding instance to group '%s'", groupName),
		t.window,
	)
	// TODO: Implement instance addition logic
}

func (t *EmulatorInstancesTab) handleQuickStart(instanceID int) {
	dialog.ShowInformation(
		"Quick Start",
		fmt.Sprintf("Quick starting instance %d", instanceID),
		t.window,
	)
	// TODO: Implement quick start logic
}

func (t *EmulatorInstancesTab) handlePause(instanceID int) {
	dialog.ShowInformation(
		"Pause",
		fmt.Sprintf("Pausing instance %d", instanceID),
		t.window,
	)
	// TODO: Implement pause logic
}

func (t *EmulatorInstancesTab) handleStop(instanceID int) {
	dialog.ShowConfirm(
		"Stop Instance",
		fmt.Sprintf("Stop instance %d?", instanceID),
		func(confirmed bool) {
			if confirmed {
				// TODO: Implement stop logic
				dialog.ShowInformation("Stopped", fmt.Sprintf("Instance %d stopped", instanceID), t.window)
			}
		},
		t.window,
	)
}

func (t *EmulatorInstancesTab) handleAbort(instanceID int) {
	dialog.ShowConfirm(
		"Abort Instance",
		fmt.Sprintf("Abort instance %d? This will force stop the routine.", instanceID),
		func(confirmed bool) {
			if confirmed {
				// TODO: Implement abort logic
				dialog.ShowInformation("Aborted", fmt.Sprintf("Instance %d aborted", instanceID), t.window)
			}
		},
		t.window,
	)
}

func (t *EmulatorInstancesTab) handleShutdown(instanceID int) {
	dialog.ShowConfirm(
		"Shutdown Instance",
		fmt.Sprintf("Shutdown instance %d? This will stop and remove it from the group.", instanceID),
		func(confirmed bool) {
			if confirmed {
				// TODO: Implement shutdown logic
				dialog.ShowInformation("Shutdown", fmt.Sprintf("Instance %d shutdown", instanceID), t.window)
			}
		},
		t.window,
	)
}

func (t *EmulatorInstancesTab) handleLaunch(instanceID int) {
	dialog.ShowConfirm(
		"Launch Emulator",
		fmt.Sprintf("Launch emulator instance %d?", instanceID),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Get emulator manager
			emulatorMgr := t.orchestrator.GetEmulatorManager()
			if emulatorMgr == nil {
				dialog.ShowError(fmt.Errorf("emulator manager not available"), t.window)
				return
			}

			// Launch the instance
			if err := emulatorMgr.LaunchInstance(instanceID); err != nil {
				dialog.ShowError(fmt.Errorf("failed to launch instance %d: %w", instanceID, err), t.window)
				return
			}

			dialog.ShowInformation("Launched", fmt.Sprintf("Instance %d launched successfully", instanceID), t.window)

			// Refresh the view after a delay to show the instance as idle
			go func() {
				time.Sleep(2 * time.Second)
				t.loadExistingData()
			}()
		},
		t.window,
	)
}

// refreshAll updates all cards from their data
func (t *EmulatorInstancesTab) refreshAll() {
	t.groupSectionsMu.RLock()
	defer t.groupSectionsMu.RUnlock()

	for _, groupCard := range t.groupSections {
		groupCard.UpdateFromGroup()
	}

	t.instanceCardsMu.RLock()
	defer t.instanceCardsMu.RUnlock()

	for _, instanceCard := range t.instanceCards {
		instanceCard.UpdateFromState()
	}
}

// startPeriodicRefresh updates card data every second
func (t *EmulatorInstancesTab) startPeriodicRefresh() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.refreshAll()
		case <-t.stopRefresh:
			return
		}
	}
}

// Stop stops the periodic refresh
func (t *EmulatorInstancesTab) Stop() {
	close(t.stopRefresh)
}

// updateStatusLabel updates the status label with counts
func (t *EmulatorInstancesTab) updateStatusLabel() {
	t.groupSectionsMu.RLock()
	groupCount := len(t.groupSections)
	t.groupSectionsMu.RUnlock()

	t.instanceCardsMu.RLock()
	instanceCount := len(t.instanceCards)
	t.instanceCardsMu.RUnlock()

	if groupCount == 0 && instanceCount == 0 {
		t.statusLabel.SetText("No instances")
	} else {
		t.statusLabel.SetText(fmt.Sprintf("%d group(s), %d instance(s)", groupCount, instanceCount))
	}
}
