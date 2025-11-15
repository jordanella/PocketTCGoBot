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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/gui/components"
)

// OrchestrationTab manages orchestration groups with the new card component
type OrchestrationTab struct {
	// Dependencies (injected)
	orchestrator *bot.Orchestrator
	window       fyne.Window

	// UI state
	cards          map[string]*components.OrchestrationCard // key = group name
	cardsMu        sync.RWMutex
	cardsContainer *fyne.Container

	// UI elements
	statusLabel *widget.Label
	createBtn   *widget.Button
	refreshBtn  *widget.Button

	// Refresh control
	stopRefresh chan bool
}

// NewOrchestrationTab creates a new orchestration tab
func NewOrchestrationTab(orchestrator *bot.Orchestrator, window fyne.Window) *OrchestrationTab {
	return &OrchestrationTab{
		orchestrator: orchestrator,
		window:       window,
		cards:        make(map[string]*components.OrchestrationCard),
		stopRefresh:  make(chan bool),
	}
}

// Build constructs the tab UI
func (t *OrchestrationTab) Build() fyne.CanvasObject {
	// Header using new component
	header := components.Heading("Orchestration Groups")

	// Description using new component
	description := components.Body(
		"Manage bot groups with coordinated emulator instances and account pools",
	)

	// Control buttons
	t.createBtn = widget.NewButton("Create New Group", func() {
		t.showCreateGroupDialog()
	})
	t.createBtn.Importance = widget.HighImportance

	t.refreshBtn = widget.NewButton("Refresh All", func() {
		t.refreshAllCards()
	})

	t.statusLabel = widget.NewLabel("No groups created")

	controls := container.NewHBox(
		t.createBtn,
		t.refreshBtn,
		layout.NewSpacer(),
		t.statusLabel,
	)

	// Cards container
	t.cardsContainer = container.NewVBox()

	// Scroll container for cards
	scroll := container.NewVScroll(t.cardsContainer)
	scroll.SetMinSize(fyne.NewSize(900, 600))

	// Main layout
	content := container.NewBorder(
		container.NewVBox(
			header,
			description,
			widget.NewSeparator(),
			controls,
		),
		nil, nil, nil,
		scroll,
	)

	// Load existing groups
	t.loadExistingGroups()

	// Start periodic refresh
	go t.startPeriodicRefresh()

	return content
}

// loadExistingGroups loads any existing orchestration groups
func (t *OrchestrationTab) loadExistingGroups() {
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
func (t *OrchestrationTab) showCreateGroupDialog() {
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

	// Form layout
	form := container.NewVBox(
		widget.NewLabelWithStyle("Create Orchestration Group", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Group Name:"),
		nameEntry,
		widget.NewLabel("Routine:"),
		routineEntry,
		widget.NewLabel("Available Emulator Instances (comma-separated):"),
		instancesEntry,
		widget.NewLabel("Concurrent Bot Count:"),
		botCountEntry,
		widget.NewLabel("Account Pool Name:"),
		poolEntry,
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

	formDialog.Resize(fyne.NewSize(500, 600))
	formDialog.Show()
}

// createGroup creates a new orchestration group
func (t *OrchestrationTab) createGroup(name, routine, instancesStr, botCountStr, poolName string) error {
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
func (t *OrchestrationTab) addGroupCard(group *bot.BotGroup) {
	// Create card
	card := components.NewOrchestrationCard(group, components.OrchestrationCardCallbacks{
		OnAddInstance: t.handleAddInstance,
		OnPauseResume: t.handlePauseResume,
		OnStop:        t.handleStop,
		OnShutdown:    t.handleShutdown,
	})

	// Add to tracking
	t.cardsMu.Lock()
	t.cards[group.Name] = card
	t.cardsMu.Unlock()

	// Add to UI (wrapped in fyne.Do for thread safety)
	fyne.Do(func() {
		t.cardsContainer.Add(card.GetContainer())
		t.cardsContainer.Refresh()
	})
}

// parseInstances parses a comma-separated list of instance IDs
func parseInstances(instancesStr string) ([]int, error) {
	instancesStr = strings.TrimSpace(instancesStr)
	if instancesStr == "" {
		return nil, fmt.Errorf("instances cannot be empty")
	}

	parts := strings.Split(instancesStr, ",")
	instances := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid instance ID '%s': %w", part, err)
		}
		if id < 0 {
			return nil, fmt.Errorf("instance ID must be non-negative: %d", id)
		}

		instances = append(instances, id)
	}

	return instances, nil
}

// Card action handlers

func (t *OrchestrationTab) handleAddInstance(group *bot.BotGroup) {
	dialog.ShowInformation(
		"Add Instance",
		fmt.Sprintf("Adding instance to group '%s'", group.Name),
		t.window,
	)
	// TODO: Implement instance addition logic
}

func (t *OrchestrationTab) handlePauseResume(group *bot.BotGroup) {
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
			},
			t.window,
		)
	}
}

func (t *OrchestrationTab) handleStop(group *bot.BotGroup) {
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
			}
		},
		t.window,
	)
}

func (t *OrchestrationTab) handleShutdown(group *bot.BotGroup) {
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
			t.cardsMu.Lock()
			if card, exists := t.cards[group.Name]; exists {
				t.cardsContainer.Remove(card.GetContainer())
				delete(t.cards, group.Name)
			}
			t.cardsMu.Unlock()

			// Refresh UI (wrapped in fyne.Do for thread safety)
			fyne.Do(func() {
				t.cardsContainer.Refresh()
			})
			t.updateStatusLabel()

			dialog.ShowInformation("Shutdown", fmt.Sprintf("Group '%s' shutdown and removed successfully", group.Name), t.window)
		},
		t.window,
	)
}

// refreshAllCards updates all cards from their groups
func (t *OrchestrationTab) refreshAllCards() {
	t.cardsMu.RLock()
	defer t.cardsMu.RUnlock()

	for _, card := range t.cards {
		card.UpdateFromGroup()
	}
}

// startPeriodicRefresh updates card data every second
func (t *OrchestrationTab) startPeriodicRefresh() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.refreshAllCards()
		case <-t.stopRefresh:
			return
		}
	}
}

// Stop stops the periodic refresh
func (t *OrchestrationTab) Stop() {
	close(t.stopRefresh)
}

// updateStatusLabel updates the status label with group count
func (t *OrchestrationTab) updateStatusLabel() {
	t.cardsMu.RLock()
	count := len(t.cards)
	t.cardsMu.RUnlock()

	if count == 0 {
		t.statusLabel.SetText("No groups created")
	} else {
		t.statusLabel.SetText(fmt.Sprintf("%d group(s) active", count))
	}
}
