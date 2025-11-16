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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/emulator"
	"jordanella.com/pocket-tcg-go/internal/gui/components"
)

// OrchestrationTabV3 manages orchestration groups with inline editing
type OrchestrationTabV3 struct {
	// Dependencies
	orchestrator *bot.Orchestrator
	emulatorMgr  *emulator.Manager
	window       fyne.Window

	// Current editing state
	currentGroup    *bot.BotGroupDefinition
	currentRunGroup *bot.BotGroup // Runtime group if exists
	isDirty         bool
	selectedName    string

	// Data management
	groupsData   []*bot.BotGroupDefinition
	groupsDataMu sync.RWMutex

	// Left panel: Group list
	groupsList      *widget.List
	selectedIndex   int
	newGroupBtn     *widget.Button
	refreshBtn      *widget.Button
	statusLabel     *widget.Label

	// Right panel: Tabs
	tabs *container.AppTabs

	// Details tab widgets
	nameEntry        *widget.Entry
	descEntry        *widget.Entry
	routineEntry     *widget.Entry
	botCountEntry    *widget.Entry
	poolSelect       *widget.Select

	// Instances tab widgets
	instancesList       *widget.List
	instancesData       []int
	instancesDataMu     sync.RWMutex
	addInstanceDropdown *widget.Select
	addInstanceBtn      *widget.Button
	refreshInstancesBtn *widget.Button

	// Launch Options tab widgets
	validateRoutineCheck   *widget.Check
	validateTemplatesCheck *widget.Check
	validateEmulatorsCheck *widget.Check
	staggerDelayEntry      *widget.Entry
	emulatorTimeoutEntry   *widget.Entry
	conflictResolutionSelect *widget.Select

	// Restart Policy widgets
	restartEnabledCheck    *widget.Check
	maxRetriesEntry        *widget.Entry
	initialDelayEntry      *widget.Entry
	maxDelayEntry          *widget.Entry
	backoffFactorEntry     *widget.Entry
	resetOnSuccessCheck    *widget.Check

	// Status tab widgets
	statusList     *widget.List
	statusData     [][]string
	statusDataMu   sync.RWMutex

	// Action buttons
	saveBtn      *widget.Button
	discardBtn   *widget.Button
	deleteBtn    *widget.Button
	startBtn     *widget.Button
	stopBtn      *widget.Button

	// Refresh control
	stopRefresh chan bool
}

// NewOrchestrationTabV3 creates a new orchestration tab with inline editing
func NewOrchestrationTabV3(orchestrator *bot.Orchestrator, emulatorMgr *emulator.Manager, window fyne.Window) *OrchestrationTabV3 {
	return &OrchestrationTabV3{
		orchestrator:  orchestrator,
		emulatorMgr:   emulatorMgr,
		window:        window,
		selectedIndex: -1,
		groupsData:    make([]*bot.BotGroupDefinition, 0),
		instancesData: make([]int, 0),
		statusData:    make([][]string, 0),
		stopRefresh:   make(chan bool),
	}
}

// Build constructs the tab UI with inline editing
func (t *OrchestrationTabV3) Build() fyne.CanvasObject {
	// Left panel: Group list
	leftPanel := t.buildLeftPanel()

	// Right panel: Tabbed editor
	rightPanel := t.buildRightPanel()

	// Load existing groups (after UI is built)
	t.loadGroupDefinitions()

	// Split layout
	split := container.NewHSplit(leftPanel, rightPanel)
	split.SetOffset(0.3) // 30% left, 70% right

	// Start periodic refresh
	go t.startPeriodicRefresh()

	return split
}

// buildLeftPanel creates the group list panel
func (t *OrchestrationTabV3) buildLeftPanel() fyne.CanvasObject {
	header := components.Heading("Orchestration Groups")

	t.newGroupBtn = components.PrimaryButton("New Group", func() {
		t.handleNewGroup()
	})

	t.refreshBtn = components.SecondaryButton("Refresh", func() {
		t.loadGroupDefinitions()
	})

	t.statusLabel = widget.NewLabel("No groups")

	controls := container.NewVBox(
		container.NewHBox(t.newGroupBtn, t.refreshBtn),
		t.statusLabel,
		widget.NewSeparator(),
	)

	// Group list
	t.groupsList = widget.NewList(
		func() int {
			t.groupsDataMu.RLock()
			defer t.groupsDataMu.RUnlock()
			return len(t.groupsData)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(""),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t.groupsDataMu.RLock()
			defer t.groupsDataMu.RUnlock()

			if id >= len(t.groupsData) {
				return
			}

			group := t.groupsData[id]
			vbox := obj.(*fyne.Container)
			nameLabel := vbox.Objects[0].(*widget.Label)
			routineLabel := vbox.Objects[1].(*widget.Label)
			instancesLabel := vbox.Objects[2].(*widget.Label)

			nameLabel.SetText(group.Name)
			routineLabel.SetText(fmt.Sprintf("Routine: %s", group.RoutineName))
			instancesLabel.SetText(fmt.Sprintf("Instances: %v | Bots: %d", group.AvailableInstances, group.RequestedBotCount))

			// Highlight selected
			if id == t.selectedIndex {
				nameLabel.TextStyle = fyne.TextStyle{Bold: true}
				nameLabel.Importance = widget.HighImportance
			} else {
				nameLabel.TextStyle = fyne.TextStyle{}
				nameLabel.Importance = widget.MediumImportance
			}
		},
	)

	t.groupsList.OnSelected = func(id widget.ListItemID) {
		t.handleGroupSelected(id)
	}

	content := container.NewBorder(
		container.NewVBox(header, widget.NewSeparator(), controls),
		nil,
		nil,
		nil,
		t.groupsList,
	)

	return content
}

// buildRightPanel creates the tabbed editor panel
func (t *OrchestrationTabV3) buildRightPanel() fyne.CanvasObject {
	// Initialize tabs
	detailsTab := t.buildDetailsTab()
	instancesTab := t.buildInstancesTab()
	launchOptionsTab := t.buildLaunchOptionsTab()
	statusTab := t.buildStatusTab()

	t.tabs = container.NewAppTabs(
		container.NewTabItem("Details", detailsTab),
		container.NewTabItem("Instances", instancesTab),
		container.NewTabItem("Launch Options", launchOptionsTab),
		container.NewTabItem("Status", statusTab),
	)

	// Action buttons
	t.saveBtn = components.PrimaryButton("Save Changes", func() {
		t.handleSaveChanges()
	})
	t.saveBtn.Disable()

	t.discardBtn = components.SecondaryButton("Discard Changes", func() {
		t.handleDiscardChanges()
	})
	t.discardBtn.Disable()

	t.deleteBtn = components.DangerButton("Delete Group", func() {
		t.handleDeleteGroup()
	})

	t.startBtn = components.PrimaryButton("Start Group", func() {
		t.handleStartGroup()
	})

	t.stopBtn = components.DangerButton("Stop Group", func() {
		t.handleStopGroup()
	})

	actionButtons := container.NewHBox(
		t.saveBtn,
		t.discardBtn,
		layout.NewSpacer(),
		t.deleteBtn,
		t.startBtn,
		t.stopBtn,
	)

	content := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), actionButtons),
		nil,
		nil,
		t.tabs,
	)

	return content
}

// buildDetailsTab creates the Details tab
func (t *OrchestrationTabV3) buildDetailsTab() fyne.CanvasObject {
	t.nameEntry = widget.NewEntry()
	t.nameEntry.SetPlaceHolder("Group name")
	t.nameEntry.OnChanged = func(s string) { t.markDirty() }

	t.descEntry = widget.NewMultiLineEntry()
	t.descEntry.SetPlaceHolder("Optional description")
	t.descEntry.OnChanged = func(s string) { t.markDirty() }
	t.descEntry.SetMinRowsVisible(3)

	t.routineEntry = widget.NewEntry()
	t.routineEntry.SetPlaceHolder("e.g., farm_premium_packs.yaml")
	t.routineEntry.OnChanged = func(s string) { t.markDirty() }

	t.botCountEntry = widget.NewEntry()
	t.botCountEntry.SetPlaceHolder("Number of concurrent bots")
	t.botCountEntry.OnChanged = func(s string) { t.markDirty() }

	t.poolSelect = widget.NewSelect([]string{}, func(s string) { t.markDirty() })
	t.poolSelect.PlaceHolder = "Select account pool (optional)"

	// Populate pool dropdown
	t.updatePoolDropdown()

	form := container.NewVBox(
		components.FieldRow("Group Name", t.nameEntry),
		components.FieldRow("Description", t.descEntry),
		components.FieldRow("Routine", t.routineEntry),
		components.FieldRow("Concurrent Bot Count", t.botCountEntry),
		components.FieldRow("Account Pool", t.poolSelect),
	)

	return container.NewVScroll(form)
}

// buildInstancesTab creates the Instances tab
func (t *OrchestrationTabV3) buildInstancesTab() fyne.CanvasObject {
	// Instance list
	t.instancesList = widget.NewList(
		func() int {
			t.instancesDataMu.RLock()
			defer t.instancesDataMu.RUnlock()
			return len(t.instancesData)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
				layout.NewSpacer(),
				widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t.instancesDataMu.RLock()
			defer t.instancesDataMu.RUnlock()

			if id >= len(t.instancesData) {
				return
			}

			instance := t.instancesData[id]
			hbox := obj.(*fyne.Container)
			label := hbox.Objects[0].(*widget.Label)
			btn := hbox.Objects[2].(*widget.Button)

			label.SetText(fmt.Sprintf("Instance %d", instance))
			btn.OnTapped = func() {
				t.handleRemoveInstance(id)
			}
		},
	)

	// Add instance dropdown
	t.addInstanceDropdown = widget.NewSelect([]string{}, nil)
	t.addInstanceDropdown.PlaceHolder = "Select instance to add"

	t.addInstanceBtn = components.PrimaryButton("Add Instance", func() {
		t.handleAddInstanceFromDropdown()
	})

	t.refreshInstancesBtn = components.SecondaryButton("Refresh Instances", func() {
		t.updateInstanceDropdown()
	})

	// Update dropdown
	t.updateInstanceDropdown()

	addSection := container.NewVBox(
		widget.NewLabel("Add Instance:"),
		t.addInstanceDropdown,
		container.NewHBox(t.addInstanceBtn, t.refreshInstancesBtn),
	)

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Configured Instances:"),
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			addSection,
		),
		nil,
		nil,
		t.instancesList,
	)

	return content
}

// buildLaunchOptionsTab creates the Launch Options tab
func (t *OrchestrationTabV3) buildLaunchOptionsTab() fyne.CanvasObject {
	// Validation options
	t.validateRoutineCheck = widget.NewCheck("Validate Routine", func(b bool) { t.markDirty() })
	t.validateTemplatesCheck = widget.NewCheck("Validate Templates", func(b bool) { t.markDirty() })
	t.validateEmulatorsCheck = widget.NewCheck("Validate Emulators", func(b bool) { t.markDirty() })

	// Timing options
	t.staggerDelayEntry = widget.NewEntry()
	t.staggerDelayEntry.SetPlaceHolder("e.g., 5s")
	t.staggerDelayEntry.OnChanged = func(s string) { t.markDirty() }

	t.emulatorTimeoutEntry = widget.NewEntry()
	t.emulatorTimeoutEntry.SetPlaceHolder("e.g., 30s")
	t.emulatorTimeoutEntry.OnChanged = func(s string) { t.markDirty() }

	// Conflict resolution
	t.conflictResolutionSelect = widget.NewSelect(
		[]string{"skip", "error", "force"},
		func(s string) { t.markDirty() },
	)
	t.conflictResolutionSelect.PlaceHolder = "Select conflict resolution strategy"

	// Restart policy
	t.restartEnabledCheck = widget.NewCheck("Enable Auto-Restart", func(b bool) { t.markDirty() })

	t.maxRetriesEntry = widget.NewEntry()
	t.maxRetriesEntry.SetPlaceHolder("e.g., 5")
	t.maxRetriesEntry.OnChanged = func(s string) { t.markDirty() }

	t.initialDelayEntry = widget.NewEntry()
	t.initialDelayEntry.SetPlaceHolder("e.g., 10s")
	t.initialDelayEntry.OnChanged = func(s string) { t.markDirty() }

	t.maxDelayEntry = widget.NewEntry()
	t.maxDelayEntry.SetPlaceHolder("e.g., 5m")
	t.maxDelayEntry.OnChanged = func(s string) { t.markDirty() }

	t.backoffFactorEntry = widget.NewEntry()
	t.backoffFactorEntry.SetPlaceHolder("e.g., 2.0")
	t.backoffFactorEntry.OnChanged = func(s string) { t.markDirty() }

	t.resetOnSuccessCheck = widget.NewCheck("Reset on Success", func(b bool) { t.markDirty() })

	form := container.NewVBox(
		widget.NewLabelWithStyle("Validation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.validateRoutineCheck,
		t.validateTemplatesCheck,
		t.validateEmulatorsCheck,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Timing", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		components.FieldRow("Stagger Delay", t.staggerDelayEntry),
		components.FieldRow("Emulator Timeout", t.emulatorTimeoutEntry),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Conflict Resolution", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.conflictResolutionSelect,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Restart Policy", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.restartEnabledCheck,
		components.FieldRow("Max Retries", t.maxRetriesEntry),
		components.FieldRow("Initial Delay", t.initialDelayEntry),
		components.FieldRow("Max Delay", t.maxDelayEntry),
		components.FieldRow("Backoff Factor", t.backoffFactorEntry),
		t.resetOnSuccessCheck,
	)

	return container.NewVScroll(form)
}

// buildStatusTab creates the Status tab showing running bots
func (t *OrchestrationTabV3) buildStatusTab() fyne.CanvasObject {
	t.statusList = widget.NewList(
		func() int {
			t.statusDataMu.RLock()
			defer t.statusDataMu.RUnlock()
			return len(t.statusData)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
				widget.NewLabel(""),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t.statusDataMu.RLock()
			defer t.statusDataMu.RUnlock()

			if id >= len(t.statusData) {
				return
			}

			row := t.statusData[id]
			hbox := obj.(*fyne.Container)
			hbox.Objects[0].(*widget.Label).SetText(row[0]) // Bot ID
			hbox.Objects[1].(*widget.Label).SetText(row[1]) // Instance
			hbox.Objects[2].(*widget.Label).SetText(row[2]) // Status
		},
	)

	header := container.NewHBox(
		widget.NewLabelWithStyle("Bot ID", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Instance", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabelWithStyle("Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	content := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		t.statusList,
	)

	return content
}

// loadGroupDefinitions loads all group definitions
func (t *OrchestrationTabV3) loadGroupDefinitions() {
	if t.orchestrator == nil {
		return
	}

	// Load from orchestrator
	definitions := t.orchestrator.ListGroupDefinitions()

	t.groupsDataMu.Lock()
	t.groupsData = definitions
	t.groupsDataMu.Unlock()

	fyne.Do(func() {
		t.groupsList.Refresh()
		t.updateStatusLabel()
	})
}

// handleGroupSelected handles group selection from list
func (t *OrchestrationTabV3) handleGroupSelected(id widget.ListItemID) {
	// Check for unsaved changes
	if t.isDirty {
		dialog.ShowConfirm(
			"Unsaved Changes",
			"You have unsaved changes. Discard them?",
			func(confirmed bool) {
				if confirmed {
					t.clearDirty()
					t.selectGroup(id)
				} else {
					// Revert selection
					fyne.Do(func() {
						t.groupsList.Select(t.selectedIndex)
					})
				}
			},
			t.window,
		)
		return
	}

	t.selectGroup(id)
}

// selectGroup selects a group and populates the editor
func (t *OrchestrationTabV3) selectGroup(id widget.ListItemID) {
	t.groupsDataMu.RLock()
	if id >= len(t.groupsData) {
		t.groupsDataMu.RUnlock()
		return
	}
	group := t.groupsData[id]
	t.groupsDataMu.RUnlock()

	t.selectedIndex = id
	t.selectedName = group.Name

	// Make a copy for editing
	t.currentGroup = &bot.BotGroupDefinition{}
	*t.currentGroup = *group

	// Check if runtime group exists
	t.currentRunGroup, _ = t.orchestrator.GetGroup(group.Name)

	// Populate fields
	t.populateFields()

	// Update button states
	t.updateButtonStates()

	// Refresh list to update highlighting
	fyne.Do(func() {
		t.groupsList.Refresh()
	})
}

// populateFields populates all editor fields from currentGroup
func (t *OrchestrationTabV3) populateFields() {
	if t.currentGroup == nil {
		return
	}

	// Details tab
	t.nameEntry.SetText(t.currentGroup.Name)
	t.descEntry.SetText(t.currentGroup.Description)
	t.routineEntry.SetText(t.currentGroup.RoutineName)
	t.botCountEntry.SetText(fmt.Sprintf("%d", t.currentGroup.RequestedBotCount))
	t.poolSelect.SetSelected(t.currentGroup.AccountPoolName)

	// Instances tab
	t.instancesDataMu.Lock()
	t.instancesData = make([]int, len(t.currentGroup.AvailableInstances))
	copy(t.instancesData, t.currentGroup.AvailableInstances)
	t.instancesDataMu.Unlock()
	fyne.Do(func() { t.instancesList.Refresh() })

	// Launch Options tab
	t.validateRoutineCheck.SetChecked(t.currentGroup.LaunchOptions.ValidateRoutine)
	t.validateTemplatesCheck.SetChecked(t.currentGroup.LaunchOptions.ValidateTemplates)
	t.validateEmulatorsCheck.SetChecked(t.currentGroup.LaunchOptions.ValidateEmulators)
	t.staggerDelayEntry.SetText(t.currentGroup.LaunchOptions.StaggerDelay.String())
	t.emulatorTimeoutEntry.SetText(t.currentGroup.LaunchOptions.EmulatorTimeout.String())

	// Map conflict resolution enum to string
	conflictStr := "skip"
	switch t.currentGroup.LaunchOptions.OnConflict {
	case bot.ConflictResolutionSkip:
		conflictStr = "skip"
	case bot.ConflictResolutionAbort:
		conflictStr = "error"
	case bot.ConflictResolutionCancel:
		conflictStr = "force"
	}
	t.conflictResolutionSelect.SetSelected(conflictStr)

	// Restart Policy
	t.restartEnabledCheck.SetChecked(t.currentGroup.LaunchOptions.RestartPolicy.Enabled)
	t.maxRetriesEntry.SetText(fmt.Sprintf("%d", t.currentGroup.LaunchOptions.RestartPolicy.MaxRetries))
	t.initialDelayEntry.SetText(t.currentGroup.LaunchOptions.RestartPolicy.InitialDelay.String())
	t.maxDelayEntry.SetText(t.currentGroup.LaunchOptions.RestartPolicy.MaxDelay.String())
	t.backoffFactorEntry.SetText(fmt.Sprintf("%.1f", t.currentGroup.LaunchOptions.RestartPolicy.BackoffFactor))
	t.resetOnSuccessCheck.SetChecked(t.currentGroup.LaunchOptions.RestartPolicy.ResetOnSuccess)

	// Status tab
	t.updateStatusData()
}

// updateStatusData updates the status table from runtime group
func (t *OrchestrationTabV3) updateStatusData() {
	t.statusDataMu.Lock()
	defer t.statusDataMu.Unlock()

	t.statusData = make([][]string, 0)

	if t.currentRunGroup != nil {
		// Get bot states from runtime group
		botInfos := t.currentRunGroup.GetAllBotInfo()
		for instanceID, info := range botInfos {
			status := string(info.Status)

			t.statusData = append(t.statusData, []string{
				fmt.Sprintf("Instance %d", instanceID),
				fmt.Sprintf("Instance %d", instanceID),
				status,
			})
		}
	}

	fyne.Do(func() {
		t.statusList.Refresh()
	})
}

// markDirty marks the group as having unsaved changes
func (t *OrchestrationTabV3) markDirty() {
	t.isDirty = true
	if t.saveBtn != nil {
		t.saveBtn.Enable()
	}
	if t.discardBtn != nil {
		t.discardBtn.Enable()
	}
}

// clearDirty clears the dirty flag
func (t *OrchestrationTabV3) clearDirty() {
	t.isDirty = false
	if t.saveBtn != nil {
		t.saveBtn.Disable()
	}
	if t.discardBtn != nil {
		t.discardBtn.Disable()
	}
}

// updateButtonStates updates action button states based on group state
func (t *OrchestrationTabV3) updateButtonStates() {
	hasGroup := t.currentGroup != nil
	isRunning := t.currentRunGroup != nil && t.currentRunGroup.IsRunning()

	if t.deleteBtn != nil {
		if hasGroup {
			t.deleteBtn.Enable()
		} else {
			t.deleteBtn.Disable()
		}
	}

	if t.startBtn != nil {
		if hasGroup && !isRunning {
			t.startBtn.Enable()
		} else {
			t.startBtn.Disable()
		}
	}

	if t.stopBtn != nil {
		if hasGroup && isRunning {
			t.stopBtn.Enable()
		} else {
			t.stopBtn.Disable()
		}
	}
}

// handleNewGroup creates a new group with just a name
func (t *OrchestrationTabV3) handleNewGroup() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter group name")

	dialog.ShowForm(
		"Create New Group",
		"Create",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Group Name", nameEntry),
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}

			name := strings.TrimSpace(nameEntry.Text)
			if name == "" {
				dialog.ShowError(fmt.Errorf("group name is required"), t.window)
				return
			}

			// Check if exists
			t.groupsDataMu.RLock()
			for _, g := range t.groupsData {
				if g.Name == name {
					t.groupsDataMu.RUnlock()
					dialog.ShowError(fmt.Errorf("group '%s' already exists", name), t.window)
					return
				}
			}
			t.groupsDataMu.RUnlock()

			// Create new definition with defaults
			newGroup := bot.NewBotGroupDefinition(name, "", []int{}, 1)
			newGroup.LaunchOptions = bot.LaunchOptions{
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

			// Save definition
			if err := t.orchestrator.SaveGroupDefinition(newGroup); err != nil {
				dialog.ShowError(fmt.Errorf("failed to save group: %w", err), t.window)
				return
			}

			// Add to list
			t.groupsDataMu.Lock()
			t.groupsData = append(t.groupsData, newGroup)
			newIndex := len(t.groupsData) - 1
			t.groupsDataMu.Unlock()

			// Refresh and select
			fyne.Do(func() {
				t.groupsList.Refresh()
				t.groupsList.Select(newIndex)
			})

			t.updateStatusLabel()
		},
		t.window,
	)
}

// handleSaveChanges saves changes to the current group
func (t *OrchestrationTabV3) handleSaveChanges() {
	if t.currentGroup == nil {
		return
	}

	// Collect values from fields
	name := strings.TrimSpace(t.nameEntry.Text)
	if name == "" {
		dialog.ShowError(fmt.Errorf("group name is required"), t.window)
		return
	}

	routine := strings.TrimSpace(t.routineEntry.Text)
	if routine == "" {
		dialog.ShowError(fmt.Errorf("routine name is required"), t.window)
		return
	}

	botCount, err := strconv.Atoi(strings.TrimSpace(t.botCountEntry.Text))
	if err != nil || botCount <= 0 {
		dialog.ShowError(fmt.Errorf("invalid bot count"), t.window)
		return
	}

	// Check instances
	t.instancesDataMu.RLock()
	if len(t.instancesData) == 0 {
		t.instancesDataMu.RUnlock()
		dialog.ShowError(fmt.Errorf("at least one instance is required"), t.window)
		return
	}
	if botCount > len(t.instancesData) {
		t.instancesDataMu.RUnlock()
		dialog.ShowError(fmt.Errorf("bot count (%d) exceeds available instances (%d)", botCount, len(t.instancesData)), t.window)
		return
	}
	t.instancesDataMu.RUnlock()

	// Update current group
	oldName := t.currentGroup.Name
	t.currentGroup.Name = name
	t.currentGroup.Description = strings.TrimSpace(t.descEntry.Text)
	t.currentGroup.RoutineName = routine
	t.currentGroup.RequestedBotCount = botCount
	t.currentGroup.AccountPoolName = t.poolSelect.Selected

	t.instancesDataMu.RLock()
	t.currentGroup.AvailableInstances = make([]int, len(t.instancesData))
	copy(t.currentGroup.AvailableInstances, t.instancesData)
	t.instancesDataMu.RUnlock()

	// Parse launch options
	t.currentGroup.LaunchOptions.ValidateRoutine = t.validateRoutineCheck.Checked
	t.currentGroup.LaunchOptions.ValidateTemplates = t.validateTemplatesCheck.Checked
	t.currentGroup.LaunchOptions.ValidateEmulators = t.validateEmulatorsCheck.Checked

	if staggerDelay, err := time.ParseDuration(t.staggerDelayEntry.Text); err == nil {
		t.currentGroup.LaunchOptions.StaggerDelay = staggerDelay
	}

	if emulatorTimeout, err := time.ParseDuration(t.emulatorTimeoutEntry.Text); err == nil {
		t.currentGroup.LaunchOptions.EmulatorTimeout = emulatorTimeout
	}

	// Map conflict resolution string to enum
	switch t.conflictResolutionSelect.Selected {
	case "skip":
		t.currentGroup.LaunchOptions.OnConflict = bot.ConflictResolutionSkip
	case "error":
		t.currentGroup.LaunchOptions.OnConflict = bot.ConflictResolutionAbort
	case "force":
		t.currentGroup.LaunchOptions.OnConflict = bot.ConflictResolutionCancel
	default:
		t.currentGroup.LaunchOptions.OnConflict = bot.ConflictResolutionSkip
	}

	// Restart policy
	t.currentGroup.LaunchOptions.RestartPolicy.Enabled = t.restartEnabledCheck.Checked

	if maxRetries, err := strconv.Atoi(t.maxRetriesEntry.Text); err == nil {
		t.currentGroup.LaunchOptions.RestartPolicy.MaxRetries = maxRetries
	}

	if initialDelay, err := time.ParseDuration(t.initialDelayEntry.Text); err == nil {
		t.currentGroup.LaunchOptions.RestartPolicy.InitialDelay = initialDelay
	}

	if maxDelay, err := time.ParseDuration(t.maxDelayEntry.Text); err == nil {
		t.currentGroup.LaunchOptions.RestartPolicy.MaxDelay = maxDelay
	}

	if backoffFactor, err := strconv.ParseFloat(t.backoffFactorEntry.Text, 64); err == nil {
		t.currentGroup.LaunchOptions.RestartPolicy.BackoffFactor = backoffFactor
	}

	t.currentGroup.LaunchOptions.RestartPolicy.ResetOnSuccess = t.resetOnSuccessCheck.Checked

	// Handle rename
	if oldName != name {
		if err := t.orchestrator.DeleteGroupDefinition(oldName); err != nil {
			fmt.Printf("Warning: failed to delete old definition: %v\n", err)
		}
	}

	// Save definition
	if err := t.orchestrator.SaveGroupDefinition(t.currentGroup); err != nil {
		dialog.ShowError(fmt.Errorf("failed to save group: %w", err), t.window)
		return
	}

	// Update in list
	t.groupsDataMu.Lock()
	if t.selectedIndex >= 0 && t.selectedIndex < len(t.groupsData) {
		t.groupsData[t.selectedIndex] = t.currentGroup
	}
	t.groupsDataMu.Unlock()

	t.clearDirty()
	fyne.Do(func() {
		t.groupsList.Refresh()
	})

	dialog.ShowInformation("Saved", fmt.Sprintf("Group '%s' saved successfully", name), t.window)
}

// handleDiscardChanges discards changes and reloads from saved definition
func (t *OrchestrationTabV3) handleDiscardChanges() {
	if t.selectedIndex >= 0 {
		t.selectGroup(t.selectedIndex)
		t.clearDirty()
	}
}

// handleDeleteGroup deletes the current group
func (t *OrchestrationTabV3) handleDeleteGroup() {
	if t.currentGroup == nil {
		return
	}

	name := t.currentGroup.Name

	dialog.ShowConfirm(
		"Delete Group",
		fmt.Sprintf("Delete group '%s'?\n\nThis will:\n- Stop the group if running\n- Remove the group definition\n- Delete the YAML file", name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Stop if running
			if t.currentRunGroup != nil && t.currentRunGroup.IsRunning() {
				if err := t.orchestrator.StopGroup(name); err != nil {
					fmt.Printf("Warning: failed to stop group: %v\n", err)
				}
			}

			// Delete runtime group
			if err := t.orchestrator.DeleteGroup(name); err != nil {
				fmt.Printf("Warning: failed to delete runtime group: %v\n", err)
			}

			// Delete definition
			if err := t.orchestrator.DeleteGroupDefinition(name); err != nil {
				dialog.ShowError(fmt.Errorf("failed to delete definition: %w", err), t.window)
				return
			}

			// Remove from list
			t.groupsDataMu.Lock()
			if t.selectedIndex >= 0 && t.selectedIndex < len(t.groupsData) {
				t.groupsData = append(t.groupsData[:t.selectedIndex], t.groupsData[t.selectedIndex+1:]...)
			}
			t.groupsDataMu.Unlock()

			// Clear selection
			t.currentGroup = nil
			t.currentRunGroup = nil
			t.selectedIndex = -1
			t.clearDirty()

			fyne.Do(func() {
				t.groupsList.Refresh()
			})

			t.updateStatusLabel()

			dialog.ShowInformation("Deleted", fmt.Sprintf("Group '%s' deleted successfully", name), t.window)
		},
		t.window,
	)
}

// handleStartGroup starts the current group
func (t *OrchestrationTabV3) handleStartGroup() {
	if t.currentGroup == nil {
		return
	}

	if t.isDirty {
		dialog.ShowError(fmt.Errorf("please save changes before starting group"), t.window)
		return
	}

	name := t.currentGroup.Name

	dialog.ShowConfirm(
		"Start Group",
		fmt.Sprintf("Start group '%s'?", name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			result, err := t.orchestrator.LaunchGroup(name, t.currentGroup.LaunchOptions)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to start group: %w", err), t.window)
				return
			}

			// Update runtime group reference
			t.currentRunGroup, _ = t.orchestrator.GetGroup(name)

			// Update status
			t.updateStatusData()
			t.updateButtonStates()

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

// handleStopGroup stops the current group
func (t *OrchestrationTabV3) handleStopGroup() {
	if t.currentGroup == nil {
		return
	}

	name := t.currentGroup.Name

	dialog.ShowConfirm(
		"Stop Group",
		fmt.Sprintf("Stop group '%s'?", name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if err := t.orchestrator.StopGroup(name); err != nil {
				dialog.ShowError(fmt.Errorf("failed to stop group: %w", err), t.window)
				return
			}

			// Update runtime group reference
			t.currentRunGroup = nil

			// Update status
			t.updateStatusData()
			t.updateButtonStates()

			dialog.ShowInformation("Stopped", fmt.Sprintf("Group '%s' stopped successfully", name), t.window)
		},
		t.window,
	)
}

// handleAddInstanceFromDropdown adds instance from dropdown selection
func (t *OrchestrationTabV3) handleAddInstanceFromDropdown() {
	selected := t.addInstanceDropdown.Selected
	if selected == "" {
		dialog.ShowError(fmt.Errorf("please select an instance"), t.window)
		return
	}

	// Parse instance number from "Instance X" or "Instance X (Title)"
	var instanceNum int
	if _, err := fmt.Sscanf(selected, "Instance %d", &instanceNum); err != nil {
		dialog.ShowError(fmt.Errorf("failed to parse instance number"), t.window)
		return
	}

	// Check if already added
	t.instancesDataMu.Lock()
	for _, inst := range t.instancesData {
		if inst == instanceNum {
			t.instancesDataMu.Unlock()
			dialog.ShowError(fmt.Errorf("instance %d already added", instanceNum), t.window)
			return
		}
	}

	// Add instance
	t.instancesData = append(t.instancesData, instanceNum)
	t.instancesDataMu.Unlock()

	fyne.Do(func() {
		t.instancesList.Refresh()
	})

	t.markDirty()
}

// handleRemoveInstance removes an instance by index
func (t *OrchestrationTabV3) handleRemoveInstance(id widget.ListItemID) {
	t.instancesDataMu.Lock()
	defer t.instancesDataMu.Unlock()

	if id >= 0 && id < len(t.instancesData) {
		t.instancesData = append(t.instancesData[:id], t.instancesData[id+1:]...)
		fyne.Do(func() {
			t.instancesList.Refresh()
		})
		t.markDirty()
	}
}

// updateInstanceDropdown updates the instance dropdown from emulator manager
func (t *OrchestrationTabV3) updateInstanceDropdown() {
	if t.emulatorMgr == nil {
		t.addInstanceDropdown.Options = []string{"No emulator manager"}
		return
	}

	// Get MuMu manager to access all configured instances (not just running ones)
	mumuMgr := t.emulatorMgr.GetMuMuManager()
	if mumuMgr == nil {
		t.addInstanceDropdown.Options = []string{"No MuMu manager"}
		return
	}

	// Get all instance configs from disk
	configs, err := mumuMgr.GetAllInstanceConfigs()
	if err != nil {
		fmt.Printf("Warning: Failed to get instance configs: %v\n", err)
		t.addInstanceDropdown.Options = []string{"No instances configured"}
		return
	}

	// Build options list from all configured instances
	options := make([]string, 0, len(configs))
	for index, config := range configs {
		label := fmt.Sprintf("Instance %d", index)
		if config.PlayerName != "" {
			label = fmt.Sprintf("Instance %d (%s)", index, config.PlayerName)
		}
		options = append(options, label)
	}

	// Sort by instance number
	// (configs map is already indexed by instance number)
	sortedOptions := make([]string, 0, len(options))
	for i := 0; i < 100; i++ { // Reasonable upper limit
		for index := range configs {
			if index == i {
				label := fmt.Sprintf("Instance %d", index)
				if configs[index].PlayerName != "" {
					label = fmt.Sprintf("Instance %d (%s)", index, configs[index].PlayerName)
				}
				sortedOptions = append(sortedOptions, label)
				break
			}
		}
	}

	t.addInstanceDropdown.Options = sortedOptions
	fyne.Do(func() {
		t.addInstanceDropdown.Refresh()
	})
}

// updatePoolDropdown updates the pool dropdown
func (t *OrchestrationTabV3) updatePoolDropdown() {
	if t.orchestrator == nil {
		t.poolSelect.Options = []string{}
		return
	}

	poolManager := t.orchestrator.GetPoolManager()
	if poolManager == nil {
		t.poolSelect.Options = []string{}
		return
	}

	// Get list of pool names
	poolNames := poolManager.ListPools()
	t.poolSelect.Options = poolNames
	t.poolSelect.Refresh()
}

// updateStatusLabel updates the status label
func (t *OrchestrationTabV3) updateStatusLabel() {
	t.groupsDataMu.RLock()
	count := len(t.groupsData)
	t.groupsDataMu.RUnlock()

	if count == 0 {
		t.statusLabel.SetText("No groups")
	} else {
		t.statusLabel.SetText(fmt.Sprintf("%d group(s)", count))
	}
}

// startPeriodicRefresh updates status data periodically
func (t *OrchestrationTabV3) startPeriodicRefresh() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if t.currentRunGroup != nil {
				t.updateStatusData()
				t.updateButtonStates()
			}
		case <-t.stopRefresh:
			return
		}
	}
}

// Stop stops the periodic refresh
func (t *OrchestrationTabV3) Stop() {
	close(t.stopRefresh)
}
