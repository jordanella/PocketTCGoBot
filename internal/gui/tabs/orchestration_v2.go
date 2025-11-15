package tabs

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/gui/components"
)

// OrchestrationTabV2 manages orchestration groups with mockup-aligned layout
type OrchestrationTabV2 struct {
	// Dependencies (injected)
	orchestrator *bot.Orchestrator
	window       fyne.Window

	// UI state
	cards             map[string]*components.OrchestrationCardV2 // key = group name
	cardsMu           sync.RWMutex
	activeContainer   *fyne.Container
	inactiveContainer *fyne.Container

	// UI elements
	statusLabel *widget.Label
	createBtn   *widget.Button
	refreshBtn  *widget.Button

	// Refresh control
	stopRefresh chan bool
}

// NewOrchestrationTabV2 creates a new orchestration tab with mockup layout
func NewOrchestrationTabV2(orchestrator *bot.Orchestrator, window fyne.Window) *OrchestrationTabV2 {
	return &OrchestrationTabV2{
		orchestrator:      orchestrator,
		window:            window,
		cards:             make(map[string]*components.OrchestrationCardV2),
		stopRefresh:       make(chan bool),
		activeContainer:   container.NewVBox(),
		inactiveContainer: container.NewVBox(),
	}
}

// Build constructs the tab UI matching the mockup
func (t *OrchestrationTabV2) Build() fyne.CanvasObject {
	// === HEADER ===
	header := components.Heading("Orchestration Groups")

	// === CONTROLS ===
	t.createBtn = components.PrimaryButton("Create New Group", func() {
		t.showCreateGroupDialog()
	})

	t.refreshBtn = components.SecondaryButton("Refresh All", func() {
		t.refreshAllCards()
	})

	t.statusLabel = widget.NewLabel("No groups created")

	controls := container.NewHBox(
		t.createBtn,
		t.refreshBtn,
		widget.NewLabel(""), // Spacer
		t.statusLabel,
	)

	// === ACTIVE GROUPS SECTION ===
	activeSection := components.SectionHeader("Active Groups")

	// === INACTIVE GROUPS SECTION ===
	inactiveSection := components.SectionHeader("Inactive Groups")

	// === MAIN CONTENT ===
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		controls,
		widget.NewSeparator(),
		activeSection,
		t.activeContainer,
		widget.NewSeparator(),
		inactiveSection,
		t.inactiveContainer,
	)

	// Scroll container
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(900, 600))

	// Load existing groups
	t.loadExistingGroups()

	// Start periodic refresh
	go t.startPeriodicRefresh()

	return scroll
}

// loadExistingGroups loads any existing orchestration groups
func (t *OrchestrationTabV2) loadExistingGroups() {
	if t.orchestrator == nil {
		return
	}

	// Load all active groups
	activeGroups := t.orchestrator.ListActiveGroups()
	for _, group := range activeGroups {
		t.addGroupCard(group)
	}

	t.updateStatusLabel()
}

// showCreateGroupDialog shows a dialog to create a new orchestration group
func (t *OrchestrationTabV2) showCreateGroupDialog() {
	// Form fields
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("e.g., Premium Farmers")

	routineEntry := widget.NewEntry()
	routineEntry.SetPlaceHolder("e.g., farm_premium_packs.yaml")

	instancesEntry := widget.NewEntry()
	instancesEntry.SetPlaceHolder("e.g., 1,2,3,4")
	instancesEntry.SetText("1,2,3,4")

	botCountEntry := widget.NewEntry()
	botCountEntry.SetPlaceHolder("Number of bots to run concurrently")
	botCountEntry.SetText("2")

	poolEntry := widget.NewEntry()
	poolEntry.SetPlaceHolder("Account pool name (optional)")

	// Form layout using components
	form := container.NewVBox(
		components.Heading("Create Orchestration Group"),
		components.RequiredFieldRow("Group Name", nameEntry, "Must be unique"),
		components.RequiredFieldRow("Routine", routineEntry, "Routine filename"),
		components.FieldRow("Available Emulator Instances (comma-separated)", instancesEntry),
		components.FieldRow("Concurrent Bot Count", botCountEntry),
		components.FieldRow("Account Pool Name", poolEntry),
	)

	// Create dialog
	formDialog := dialog.NewCustomConfirm(
		"Create Orchestration Group",
		"Create",
		"Cancel",
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
				botCountEntry.Text,
				poolEntry.Text,
			); err != nil {
				dialog.ShowError(err, t.window)
			}
		},
		t.window,
	)

	formDialog.Resize(fyne.NewSize(600, 700))
	formDialog.Show()
}

// createGroup creates a new orchestration group
func (t *OrchestrationTabV2) createGroup(name, routine, instancesStr, botCountStr, poolName string) error {
	// Validation
	if name == "" {
		return fmt.Errorf("group name is required")
	}
	if routine == "" {
		return fmt.Errorf("routine name is required")
	}

	// Check if group already exists
	t.cardsMu.RLock()
	if _, exists := t.cards[name]; exists {
		t.cardsMu.RUnlock()
		return fmt.Errorf("group '%s' already exists", name)
	}
	t.cardsMu.RUnlock()

	// Parse instances
	instances, err := parseInstances(instancesStr)
	if err != nil {
		return fmt.Errorf("invalid instances: %w", err)
	}
	if len(instances) == 0 {
		return fmt.Errorf("at least one emulator instance is required")
	}

	// Parse bot count
	botCount, err := strconv.Atoi(strings.TrimSpace(botCountStr))
	if err != nil {
		return fmt.Errorf("invalid bot count: %w", err)
	}
	if botCount <= 0 {
		return fmt.Errorf("bot count must be positive")
	}
	if botCount > len(instances) {
		return fmt.Errorf("bot count (%d) exceeds available instances (%d)", botCount, len(instances))
	}

	// Create definition
	definition := bot.NewBotGroupDefinition(name, routine, instances, botCount)
	definition.AccountPoolName = poolName

	// Save definition to orchestrator
	if err := t.orchestrator.SaveGroupDefinition(definition); err != nil {
		return fmt.Errorf("failed to save group definition: %w", err)
	}

	// Create runtime group from definition
	group, err := t.orchestrator.CreateGroupFromDefinition(definition)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	// Add card to UI
	t.addGroupCard(group)
	t.updateStatusLabel()

	dialog.ShowInformation(
		"Group Created",
		fmt.Sprintf("Orchestration group '%s' created successfully", name),
		t.window,
	)

	return nil
}

// addGroupCard creates and adds a card for a bot group
func (t *OrchestrationTabV2) addGroupCard(group *bot.BotGroup) {
	// Create card
	card := components.NewOrchestrationCardV2(group, components.OrchestrationCardCallbacks{
		OnAddInstance: t.handleAddInstance,
		OnPauseResume: t.handlePauseResume,
		OnStop:        t.handleStop,
		OnShutdown:    t.handleShutdown,
	})

	// Add to tracking
	t.cardsMu.Lock()
	t.cards[group.Name] = card
	t.cardsMu.Unlock()

	// Add to appropriate section
	if group.IsRunning() {
		t.activeContainer.Add(card.GetContainer())
	} else {
		t.inactiveContainer.Add(card.GetContainer())
	}

	t.refreshContainers()
}

// removeGroupCard removes a card from the UI
func (t *OrchestrationTabV2) removeGroupCard(groupName string) {
	t.cardsMu.Lock()
	defer t.cardsMu.Unlock()

	if card, exists := t.cards[groupName]; exists {
		// Remove from both containers (it'll only be in one)
		t.activeContainer.Remove(card.GetContainer())
		t.inactiveContainer.Remove(card.GetContainer())
		delete(t.cards, groupName)
		t.refreshContainers()
	}
}

// refreshContainers refreshes both active and inactive containers
func (t *OrchestrationTabV2) refreshContainers() {
	fyne.Do(func() {
		t.activeContainer.Refresh()
		t.inactiveContainer.Refresh()
	})
}

// Card action handlers

func (t *OrchestrationTabV2) handleAddInstance(group *bot.BotGroup) {
	dialog.ShowInformation(
		"Add Instance",
		fmt.Sprintf("Adding instance to group '%s'", group.Name),
		t.window,
	)
	// TODO: Implement instance addition logic
}

func (t *OrchestrationTabV2) handlePauseResume(group *bot.BotGroup) {
	running := group.IsRunning()

	if running {
		// Stop the group (pause functionality)
		dialog.ShowConfirm(
			"Stop Group",
			fmt.Sprintf("Stop group '%s'? You can restart it later.", group.Name),
			func(confirmed bool) {
				if !confirmed {
					return
				}

				if err := t.orchestrator.StopGroup(group.Name); err != nil {
					dialog.ShowError(fmt.Errorf("failed to stop group: %w", err), t.window)
				} else {
					dialog.ShowInformation("Stopped", fmt.Sprintf("Group '%s' stopped successfully", group.Name), t.window)
					// Move card to inactive section
					t.reorganizeCards()
				}
			},
			t.window,
		)
	} else {
		// Launch the group (resume functionality)
		dialog.ShowConfirm(
			"Start Group",
			fmt.Sprintf("Start group '%s'?", group.Name),
			func(confirmed bool) {
				if !confirmed {
					return
				}

				// Use default launch options
				options := bot.LaunchOptions{
					ValidateRoutine:   true,
					ValidateTemplates: true,
					ValidateEmulators: false,
					StaggerDelay:      5 * time.Second,
					EmulatorTimeout:   30 * time.Second,
					OnConflict:        bot.ConflictResolutionSkip,
					RestartPolicy: bot.RestartPolicy{
						Enabled:        true,
						MaxRetries:     5,
						InitialDelay:   10 * time.Second,
						MaxDelay:       5 * time.Minute,
						BackoffFactor:  2.0,
						ResetOnSuccess: true,
					},
				}

				result, err := t.orchestrator.LaunchGroup(group.Name, options)
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to start group: %w", err), t.window)
					return
				}

				// Show result summary
				message := fmt.Sprintf(
					"Group started!\n\nLaunched: %d/%d bots\nConflicts: %d\nErrors: %d",
					result.LaunchedBots,
					result.RequestedBots,
					len(result.Conflicts),
					len(result.Errors),
				)
				dialog.ShowInformation("Group Started", message, t.window)

				// Move card to active section
				t.reorganizeCards()
			},
			t.window,
		)
	}
}

func (t *OrchestrationTabV2) handleStop(group *bot.BotGroup) {
	dialog.ShowConfirm(
		"Stop Group",
		fmt.Sprintf("Are you sure you want to stop group '%s'?", group.Name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if err := t.orchestrator.StopGroup(group.Name); err != nil {
				dialog.ShowError(fmt.Errorf("failed to stop group: %w", err), t.window)
			} else {
				dialog.ShowInformation("Stopped", fmt.Sprintf("Group '%s' stopped successfully", group.Name), t.window)
				t.reorganizeCards()
			}
		},
		t.window,
	)
}

func (t *OrchestrationTabV2) handleShutdown(group *bot.BotGroup) {
	dialog.ShowConfirm(
		"Shutdown Group",
		fmt.Sprintf("Are you sure you want to shutdown and remove group '%s'?\n\nThis will:\n- Stop all bots\n- Release all instances\n- Remove the group from active groups\n- Delete the saved definition", group.Name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// First stop the group if running
			if group.IsRunning() {
				if err := t.orchestrator.StopGroup(group.Name); err != nil {
					dialog.ShowError(fmt.Errorf("failed to stop group: %w", err), t.window)
					return
				}
			}

			// Delete the group
			if err := t.orchestrator.DeleteGroup(group.Name); err != nil {
				dialog.ShowError(fmt.Errorf("failed to delete group: %w", err), t.window)
				return
			}

			// Delete the definition
			if err := t.orchestrator.DeleteGroupDefinition(group.Name); err != nil {
				// Log but don't fail - definition might not exist
				fmt.Printf("Warning: failed to delete definition for '%s': %v\n", group.Name, err)
			}

			// Remove card from UI
			t.removeGroupCard(group.Name)
			t.updateStatusLabel()

			dialog.ShowInformation("Shutdown", fmt.Sprintf("Group '%s' shutdown and removed successfully", group.Name), t.window)
		},
		t.window,
	)
}

// reorganizeCards moves cards between active/inactive sections based on their state
func (t *OrchestrationTabV2) reorganizeCards() {
	t.cardsMu.Lock()
	defer t.cardsMu.Unlock()

	// Clear both containers
	t.activeContainer.Objects = nil
	t.inactiveContainer.Objects = nil

	// Re-add cards to appropriate sections
	for _, card := range t.cards {
		group := card.GetGroup()
		if group.IsRunning() {
			t.activeContainer.Add(card.GetContainer())
		} else {
			t.inactiveContainer.Add(card.GetContainer())
		}
	}

	t.refreshContainers()
}

// refreshAllCards updates all cards from their groups
func (t *OrchestrationTabV2) refreshAllCards() {
	t.cardsMu.RLock()
	defer t.cardsMu.RUnlock()

	for _, card := range t.cards {
		card.UpdateFromGroup()
	}
}

// startPeriodicRefresh updates card data every second
func (t *OrchestrationTabV2) startPeriodicRefresh() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.refreshAllCards()
			// Periodically reorganize in case state changed
			if time.Now().Second()%5 == 0 {
				t.reorganizeCards()
			}
		case <-t.stopRefresh:
			return
		}
	}
}

// Stop stops the periodic refresh
func (t *OrchestrationTabV2) Stop() {
	close(t.stopRefresh)
}

// updateStatusLabel updates the status label with group count
func (t *OrchestrationTabV2) updateStatusLabel() {
	t.cardsMu.RLock()
	count := len(t.cards)
	t.cardsMu.RUnlock()

	if count == 0 {
		t.statusLabel.SetText("No groups created")
	} else {
		t.statusLabel.SetText(fmt.Sprintf("%d group(s) active", count))
	}
}
