package tabs

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/emulator"
	"jordanella.com/pocket-tcg-go/internal/gui/components"
)

// AccountPoolsTabV2 manages account pools with inline editing (no wizard)
type AccountPoolsTabV2 struct {
	// Dependencies
	poolManager *accountpool.PoolManager
	db          *sql.DB
	window      fyne.Window
	emulatorMgr *emulator.Manager

	// UI state - pool cards
	poolCards         map[string]*components.AccountPoolCard
	poolCardsMu       sync.RWMutex
	selectedPoolName  string
	poolListContainer *fyne.Container

	// Right panel tabs
	tabContainer *container.AppTabs

	// Current pool being edited (in-memory)
	currentPool *accountpool.UnifiedPoolDefinition
	isDirty     bool

	// Details tab - editable fields
	poolNameLabel    *widget.Label
	descEntry        *widget.Entry
	sortMethodSelect *widget.Select
	retryFailedCheck *widget.Check
	maxFailuresEntry *widget.Entry

	// Details tab - read-only
	totalAccountsValue *widget.Label
	lastUpdatedLabel   *widget.Label

	// Accounts tab
	accountsTable  *widget.Table
	accountsData   [][]string
	accountsDataMu sync.RWMutex

	// Queries tab
	queriesData   []accountpool.QuerySource
	queriesDataMu sync.RWMutex
	queriesList   *widget.List

	// Include tab
	includesData   []string
	includesDataMu sync.RWMutex
	includesList   *widget.List

	// Exclude tab
	excludesData   []string
	excludesDataMu sync.RWMutex
	excludesList   *widget.List

	// UI buttons
	statusLabel *widget.Label
	newBtn      *widget.Button
	refreshBtn  *widget.Button
	saveBtn     *widget.Button
	discardBtn  *widget.Button

	stopRefresh chan bool
}

// NewAccountPoolsTabV2 creates a new account pools tab with inline editing
func NewAccountPoolsTabV2(poolManager *accountpool.PoolManager, db *sql.DB, emulatorMgr *emulator.Manager, window fyne.Window) *AccountPoolsTabV2 {
	return &AccountPoolsTabV2{
		poolManager:   poolManager,
		db:            db,
		emulatorMgr:   emulatorMgr,
		window:        window,
		poolCards:     make(map[string]*components.AccountPoolCard),
		stopRefresh:   make(chan bool),
		queriesData:   []accountpool.QuerySource{},
		includesData:  []string{},
		excludesData:  []string{},
		accountsData:  [][]string{},
	}
}

// Build constructs the tab UI
func (t *AccountPoolsTabV2) Build() fyne.CanvasObject {
	// === LEFT PANEL: Pool List ===
	leftPanel := t.buildLeftPanel()

	// === RIGHT PANEL: Pool Details (Tabbed) ===
	rightPanel := t.buildRightPanel()

	// === SPLIT VIEW ===
	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.3 // 30% for list, 70% for details

	return split
}

// buildLeftPanel creates the pool list panel
func (t *AccountPoolsTabV2) buildLeftPanel() fyne.CanvasObject {
	// Header
	header := components.Heading("Account Pools")

	// Controls
	t.newBtn = components.PrimaryButton("New Pool", func() {
		t.handleNewPool()
	})

	t.refreshBtn = components.SecondaryButton("Refresh", func() {
		t.loadExistingPools()
	})

	t.statusLabel = widget.NewLabel("Loading...")

	controls := container.NewVBox(
		container.NewHBox(t.newBtn, t.refreshBtn),
		t.statusLabel,
	)

	// Pool list container
	t.poolListContainer = container.NewVBox()
	scroll := container.NewVScroll(t.poolListContainer)

	// Layout
	content := container.NewBorder(
		container.NewVBox(header, widget.NewSeparator(), controls, widget.NewSeparator()),
		nil, nil, nil,
		scroll,
	)

	// Load pools
	t.loadExistingPools()

	return content
}

// buildRightPanel creates the tabbed details panel
func (t *AccountPoolsTabV2) buildRightPanel() fyne.CanvasObject {
	// Build tabs
	detailsTab := container.NewTabItem("Details", t.buildDetailsTab())
	accountsTab := container.NewTabItem("Accounts", t.buildAccountsTab())
	queriesTab := container.NewTabItem("Queries", t.buildQueriesTab())
	includeTab := container.NewTabItem("Include", t.buildIncludeTab())
	excludeTab := container.NewTabItem("Exclude", t.buildExcludeTab())

	t.tabContainer = container.NewAppTabs(
		detailsTab,
		accountsTab,
		queriesTab,
		includeTab,
		excludeTab,
	)

	// Header with pool name and rename button
	t.poolNameLabel = widget.NewLabel("Select a pool")
	t.poolNameLabel.TextStyle = fyne.TextStyle{Bold: true}

	renameBtn := components.SecondaryButton("Rename", func() {
		t.handleRenamePool()
	})

	header := container.NewBorder(
		nil, nil,
		t.poolNameLabel,
		renameBtn,
		container.NewHBox(),
	)

	return container.NewBorder(
		header,
		nil, nil, nil,
		t.tabContainer,
	)
}

// buildDetailsTab creates the editable details tab
func (t *AccountPoolsTabV2) buildDetailsTab() fyne.CanvasObject {
	// Description
	descLabel := components.Subheading("Description")
	t.descEntry = widget.NewMultiLineEntry()
	t.descEntry.SetPlaceHolder("Enter pool description...")
	t.descEntry.OnChanged = func(string) { t.markDirty() }
	t.descEntry.SetMinRowsVisible(3)

	// Total Accounts (read-only)
	totalLabel := components.BoldText("Total Accounts:")
	t.totalAccountsValue = widget.NewLabel("0")
	t.lastUpdatedLabel = widget.NewLabel("(not loaded)")
	refreshBtn := components.SecondaryButton("Refresh", func() {
		t.handleRefreshPool()
	})

	accountsRow := container.NewHBox(
		totalLabel,
		t.totalAccountsValue,
		t.lastUpdatedLabel,
		refreshBtn,
	)

	// Sort Method
	sortLabel := components.BoldText("Sort Method:")
	t.sortMethodSelect = widget.NewSelect([]string{
		"packs_desc",
		"packs_asc",
		"shinedust_desc",
		"shinedust_asc",
		"random",
	}, func(string) { t.markDirty() })
	t.sortMethodSelect.SetSelected("packs_desc")

	sortRow := container.NewHBox(sortLabel, t.sortMethodSelect)

	// Retry Failed
	t.retryFailedCheck = widget.NewCheck("Retry Failed Accounts", func(bool) { t.markDirty() })

	// Max Failures
	maxFailuresLabel := components.BoldText("Max Failures:")
	t.maxFailuresEntry = widget.NewEntry()
	t.maxFailuresEntry.SetPlaceHolder("3")
	t.maxFailuresEntry.OnChanged = func(string) { t.markDirty() }

	maxFailuresRow := container.NewHBox(maxFailuresLabel, t.maxFailuresEntry)

	// Actions
	t.saveBtn = components.PrimaryButton("Save Changes", func() {
		t.handleSave()
	})
	t.saveBtn.Disable()

	t.discardBtn = components.SecondaryButton("Discard Changes", func() {
		t.handleDiscard()
	})
	t.discardBtn.Disable()

	deleteBtn := components.DangerButton("Delete Pool", func() {
		t.handleDeletePool()
	})

	actions := container.NewHBox(t.saveBtn, t.discardBtn, deleteBtn)

	// Layout
	content := container.NewVBox(
		descLabel,
		t.descEntry,
		widget.NewSeparator(),
		accountsRow,
		widget.NewSeparator(),
		sortRow,
		t.retryFailedCheck,
		maxFailuresRow,
		widget.NewSeparator(),
		actions,
	)

	return container.NewVScroll(content)
}

// buildAccountsTab creates the accounts display tab
func (t *AccountPoolsTabV2) buildAccountsTab() fyne.CanvasObject {
	t.accountsTable = widget.NewTable(
		func() (int, int) {
			t.accountsDataMu.RLock()
			defer t.accountsDataMu.RUnlock()
			return len(t.accountsData), 4
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			t.accountsDataMu.RLock()
			defer t.accountsDataMu.RUnlock()

			if id.Row < len(t.accountsData) && id.Col < len(t.accountsData[id.Row]) {
				label.SetText(t.accountsData[id.Row][id.Col])
			} else {
				label.SetText("")
			}
		},
	)

	headers := []string{"Account", "Packs", "Shinedust", "Status"}
	t.accountsTable.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) {
		if id.Col < len(headers) {
			obj.(*widget.Label).SetText(headers[id.Col])
		}
	}

	t.accountsTable.SetColumnWidth(0, 200)
	t.accountsTable.SetColumnWidth(1, 80)
	t.accountsTable.SetColumnWidth(2, 100)
	t.accountsTable.SetColumnWidth(3, 150)

	return container.NewVScroll(t.accountsTable)
}

// buildQueriesTab creates the queries management tab
func (t *AccountPoolsTabV2) buildQueriesTab() fyne.CanvasObject {
	// Queries list
	t.queriesList = widget.NewList(
		func() int {
			t.queriesDataMu.RLock()
			defer t.queriesDataMu.RUnlock()
			return len(t.queriesData)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Query Name"),
				widget.NewButton("Edit", nil),
				widget.NewButton("Delete", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t.queriesDataMu.RLock()
			defer t.queriesDataMu.RUnlock()

			if id < len(t.queriesData) {
				query := t.queriesData[id]
				box := obj.(*fyne.Container)
				box.Objects[0].(*widget.Label).SetText(query.Name)
				box.Objects[1].(*widget.Button).OnTapped = func() {
					t.handleEditQuery(id)
				}
				box.Objects[2].(*widget.Button).OnTapped = func() {
					t.handleDeleteQuery(id)
				}
			}
		},
	)

	addBtn := components.PrimaryButton("+ Add Query", func() {
		t.handleAddQuery()
	})

	return container.NewBorder(
		addBtn,
		nil, nil, nil,
		t.queriesList,
	)
}

// buildIncludeTab creates the include management tab
func (t *AccountPoolsTabV2) buildIncludeTab() fyne.CanvasObject {
	// Include list
	t.includesList = widget.NewList(
		func() int {
			t.includesDataMu.RLock()
			defer t.includesDataMu.RUnlock()
			return len(t.includesData)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Account ID"),
				widget.NewButton("Remove", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t.includesDataMu.RLock()
			defer t.includesDataMu.RUnlock()

			if id < len(t.includesData) {
				accountID := t.includesData[id]
				box := obj.(*fyne.Container)
				box.Objects[0].(*widget.Label).SetText(accountID)
				box.Objects[1].(*widget.Button).OnTapped = func() {
					t.handleRemoveInclude(id)
				}
			}
		},
	)

	// Add controls
	accountEntry := widget.NewEntry()
	accountEntry.SetPlaceHolder("Enter account ID...")

	addManualBtn := components.SecondaryButton("Add Manual", func() {
		if accountEntry.Text != "" {
			t.handleAddInclude(accountEntry.Text)
			accountEntry.SetText("")
		}
	})

	// Instance dropdown (populated from emulator manager)
	instanceSelect := widget.NewSelect([]string{}, func(selected string) {
		if selected != "" {
			t.handleAddInclude(selected)
		}
	})
	t.updateInstanceDropdown(instanceSelect)

	addFromInstanceBtn := components.SecondaryButton("Refresh Instances", func() {
		t.updateInstanceDropdown(instanceSelect)
	})

	controls := container.NewVBox(
		container.NewHBox(widget.NewLabel("Manual:"), accountEntry, addManualBtn),
		container.NewHBox(widget.NewLabel("From Instance:"), instanceSelect, addFromInstanceBtn),
	)

	return container.NewBorder(
		controls,
		nil, nil, nil,
		t.includesList,
	)
}

// buildExcludeTab creates the exclude management tab
func (t *AccountPoolsTabV2) buildExcludeTab() fyne.CanvasObject {
	// Exclude list
	t.excludesList = widget.NewList(
		func() int {
			t.excludesDataMu.RLock()
			defer t.excludesDataMu.RUnlock()
			return len(t.excludesData)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Account ID"),
				widget.NewButton("Remove", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t.excludesDataMu.RLock()
			defer t.excludesDataMu.RUnlock()

			if id < len(t.excludesData) {
				accountID := t.excludesData[id]
				box := obj.(*fyne.Container)
				box.Objects[0].(*widget.Label).SetText(accountID)
				box.Objects[1].(*widget.Button).OnTapped = func() {
					t.handleRemoveExclude(id)
				}
			}
		},
	)

	// Add controls
	accountEntry := widget.NewEntry()
	accountEntry.SetPlaceHolder("Enter account ID...")

	addManualBtn := components.SecondaryButton("Add Manual", func() {
		if accountEntry.Text != "" {
			t.handleAddExclude(accountEntry.Text)
			accountEntry.SetText("")
		}
	})

	// Instance dropdown
	instanceSelect := widget.NewSelect([]string{}, func(selected string) {
		if selected != "" {
			t.handleAddExclude(selected)
		}
	})
	t.updateInstanceDropdown(instanceSelect)

	addFromInstanceBtn := components.SecondaryButton("Refresh Instances", func() {
		t.updateInstanceDropdown(instanceSelect)
	})

	controls := container.NewVBox(
		container.NewHBox(widget.NewLabel("Manual:"), accountEntry, addManualBtn),
		container.NewHBox(widget.NewLabel("From Instance:"), instanceSelect, addFromInstanceBtn),
	)

	return container.NewBorder(
		controls,
		nil, nil, nil,
		t.excludesList,
	)
}

// updateInstanceDropdown populates dropdown with detected emulator instances
func (t *AccountPoolsTabV2) updateInstanceDropdown(dropdown *widget.Select) {
	if t.emulatorMgr == nil {
		dropdown.Options = []string{"No emulator manager"}
		dropdown.Refresh()
		return
	}

	// Discover instances
	if err := t.emulatorMgr.DiscoverInstances(); err != nil {
		fmt.Printf("Warning: Failed to discover instances: %v\n", err)
	}

	instances := t.emulatorMgr.GetAllInstances()
	options := make([]string, 0, len(instances))

	for _, inst := range instances {
		if inst.MuMu != nil {
			label := fmt.Sprintf("Instance %d", inst.MuMu.Index)
			if inst.MuMu.WindowTitle != "" {
				label = fmt.Sprintf("Instance %d (%s)", inst.MuMu.Index, inst.MuMu.WindowTitle)
			}
			options = append(options, label)
		}
	}

	dropdown.Options = options
	dropdown.Refresh()
}

// === EVENT HANDLERS ===

func (t *AccountPoolsTabV2) markDirty() {
	t.isDirty = true
	if t.saveBtn != nil {
		t.saveBtn.Enable()
	}
	if t.discardBtn != nil {
		t.discardBtn.Enable()
	}
}

func (t *AccountPoolsTabV2) clearDirty() {
	t.isDirty = false
	if t.saveBtn != nil {
		t.saveBtn.Disable()
	}
	if t.discardBtn != nil {
		t.discardBtn.Disable()
	}
}

// handleNewPool creates a new pool
func (t *AccountPoolsTabV2) handleNewPool() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter pool name...")

	dialog.ShowCustomConfirm("New Pool", "Create", "Cancel",
		container.NewVBox(
			widget.NewLabel("Enter a name for the new pool:"),
			nameEntry,
		),
		func(create bool) {
			if !create || nameEntry.Text == "" {
				return
			}

			poolName := strings.TrimSpace(nameEntry.Text)

			// Check if exists
			existingPools := t.poolManager.ListPools()
			for _, existing := range existingPools {
				if existing == poolName {
					dialog.ShowError(fmt.Errorf("pool '%s' already exists", poolName), t.window)
					return
				}
			}

			// Create minimal pool definition
			poolDef := &accountpool.PoolDefinition{
				Name: poolName,
				Config: &accountpool.UnifiedPoolDefinition{
					PoolName:    poolName,
					Description: "",
					Queries:     []accountpool.QuerySource{},
					Include:     []string{},
					Exclude:     []string{},
					WatchedPaths: []string{},
					Config: accountpool.UnifiedPoolConfig{
						SortMethod:      "packs_desc",
						RetryFailed:     false,
						MaxFailures:     3,
						RefreshInterval: 0,
					},
				},
			}

			// Save to disk
			if err := t.poolManager.CreatePool(poolDef); err != nil {
				dialog.ShowError(fmt.Errorf("failed to create pool: %w", err), t.window)
				return
			}

			// Refresh list and select new pool
			t.loadExistingPools()
			t.handleSelectPool(poolName)

			dialog.ShowInformation("Success", fmt.Sprintf("Pool '%s' created", poolName), t.window)
		},
		t.window,
	)
}

// handleSelectPool selects a pool and loads its data
func (t *AccountPoolsTabV2) handleSelectPool(poolName string) {
	// Check for unsaved changes
	if t.isDirty {
		dialog.ShowConfirm("Unsaved Changes",
			"You have unsaved changes. Discard them?",
			func(discard bool) {
				if discard {
					t.isDirty = false
					t.selectPoolImpl(poolName)
				}
			},
			t.window,
		)
		return
	}

	t.selectPoolImpl(poolName)
}

func (t *AccountPoolsTabV2) selectPoolImpl(poolName string) {
	t.selectedPoolName = poolName

	// Update card selection states
	t.poolCardsMu.Lock()
	for name, card := range t.poolCards {
		card.SetSelected(name == poolName)
	}
	t.poolCardsMu.Unlock()

	// Load pool data
	t.loadPoolData(poolName)
	t.clearDirty()

	fmt.Printf("Selected pool: %s\n", poolName)
}

// loadPoolData loads pool definition into UI
func (t *AccountPoolsTabV2) loadPoolData(poolName string) {
	poolDef, err := t.poolManager.GetPoolDefinition(poolName)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to load pool: %w", err), t.window)
		return
	}

	t.currentPool = poolDef.Config

	// Update pool name
	t.poolNameLabel.SetText(poolName)

	// Update Details tab
	t.descEntry.SetText(poolDef.Config.Description)
	t.sortMethodSelect.SetSelected(poolDef.Config.Config.SortMethod)
	t.retryFailedCheck.SetChecked(poolDef.Config.Config.RetryFailed)
	t.maxFailuresEntry.SetText(fmt.Sprintf("%d", poolDef.Config.Config.MaxFailures))

	// Update Queries tab
	t.queriesDataMu.Lock()
	t.queriesData = poolDef.Config.Queries
	t.queriesDataMu.Unlock()
	if t.queriesList != nil {
		fyne.Do(func() { t.queriesList.Refresh() })
	}

	// Update Include tab
	t.includesDataMu.Lock()
	t.includesData = poolDef.Config.Include
	t.includesDataMu.Unlock()
	if t.includesList != nil {
		fyne.Do(func() { t.includesList.Refresh() })
	}

	// Update Exclude tab
	t.excludesDataMu.Lock()
	t.excludesData = poolDef.Config.Exclude
	t.excludesDataMu.Unlock()
	if t.excludesList != nil {
		fyne.Do(func() { t.excludesList.Refresh() })
	}

	// Load account count
	t.handleRefreshPool()
}

// handleSave saves the current pool
func (t *AccountPoolsTabV2) handleSave() {
	if t.selectedPoolName == "" || t.currentPool == nil {
		return
	}

	// Build updated pool definition from UI
	maxFailures, _ := strconv.Atoi(t.maxFailuresEntry.Text)
	if maxFailures == 0 {
		maxFailures = 3
	}

	t.currentPool.Description = t.descEntry.Text
	t.currentPool.Config.SortMethod = t.sortMethodSelect.Selected
	t.currentPool.Config.RetryFailed = t.retryFailedCheck.Checked
	t.currentPool.Config.MaxFailures = maxFailures

	// Get queries, includes, excludes from UI
	t.queriesDataMu.RLock()
	t.currentPool.Queries = t.queriesData
	t.queriesDataMu.RUnlock()

	t.includesDataMu.RLock()
	t.currentPool.Include = t.includesData
	t.includesDataMu.RUnlock()

	t.excludesDataMu.RLock()
	t.currentPool.Exclude = t.excludesData
	t.excludesDataMu.RUnlock()

	// Save to disk
	poolDef := &accountpool.PoolDefinition{
		Name:   t.selectedPoolName,
		Config: t.currentPool,
	}

	if err := t.poolManager.UpdatePool(t.selectedPoolName, poolDef); err != nil {
		dialog.ShowError(fmt.Errorf("failed to save pool: %w", err), t.window)
		return
	}

	t.clearDirty()
	dialog.ShowInformation("Saved", fmt.Sprintf("Pool '%s' saved successfully", t.selectedPoolName), t.window)
}

// handleDiscard discards unsaved changes
func (t *AccountPoolsTabV2) handleDiscard() {
	if t.selectedPoolName == "" {
		return
	}

	t.loadPoolData(t.selectedPoolName)
	t.clearDirty()
}

// handleRenamePool renames the current pool
func (t *AccountPoolsTabV2) handleRenamePool() {
	if t.selectedPoolName == "" {
		return
	}

	oldName := t.selectedPoolName
	nameEntry := widget.NewEntry()
	nameEntry.SetText(oldName)

	dialog.ShowCustomConfirm("Rename Pool", "Rename", "Cancel",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Rename pool '%s' to:", oldName)),
			nameEntry,
		),
		func(rename bool) {
			if !rename {
				return
			}

			newName := strings.TrimSpace(nameEntry.Text)
			if newName == "" || newName == oldName {
				return
			}

			// Check if new name exists
			existingPools := t.poolManager.ListPools()
			for _, existing := range existingPools {
				if existing == newName {
					dialog.ShowError(fmt.Errorf("pool '%s' already exists", newName), t.window)
					return
				}
			}

			// Get existing definition
			oldDef, err := t.poolManager.GetPoolDefinition(oldName)
			if err != nil {
				dialog.ShowError(err, t.window)
				return
			}

			// Create new definition
			newDef := &accountpool.PoolDefinition{
				Name:   newName,
				Config: oldDef.Config,
			}
			newDef.Config.PoolName = newName

			// Delete old, create new
			if err := t.poolManager.DeletePool(oldName); err != nil {
				dialog.ShowError(err, t.window)
				return
			}

			if err := t.poolManager.CreatePool(newDef); err != nil {
				dialog.ShowError(err, t.window)
				return
			}

			t.loadExistingPools()
			t.handleSelectPool(newName)
			dialog.ShowInformation("Success", fmt.Sprintf("Pool renamed to '%s'", newName), t.window)
		},
		t.window,
	)
}

// handleDeletePool deletes the current pool
func (t *AccountPoolsTabV2) handleDeletePool() {
	if t.selectedPoolName == "" {
		return
	}

	poolName := t.selectedPoolName

	dialog.ShowConfirm("Delete Pool",
		fmt.Sprintf("Are you sure you want to delete pool '%s'?\n\nThis will permanently delete the pool definition file.", poolName),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if err := t.poolManager.DeletePool(poolName); err != nil {
				dialog.ShowError(err, t.window)
				return
			}

			t.selectedPoolName = ""
			t.currentPool = nil
			t.clearDirty()
			t.loadExistingPools()

			// Reset UI
			t.poolNameLabel.SetText("Select a pool")
			t.descEntry.SetText("")

			dialog.ShowInformation("Deleted", fmt.Sprintf("Pool '%s' deleted", poolName), t.window)
		},
		t.window,
	)
}

// handleRefreshPool refreshes pool account count
func (t *AccountPoolsTabV2) handleRefreshPool() {
	if t.selectedPoolName == "" {
		return
	}

	testResult, err := t.poolManager.TestPool(t.selectedPoolName)
	if err != nil {
		t.totalAccountsValue.SetText("Error")
		t.lastUpdatedLabel.SetText(fmt.Sprintf("(error: %v)", err))
		return
	}

	t.totalAccountsValue.SetText(fmt.Sprintf("%d", testResult.AccountsFound))
	t.lastUpdatedLabel.SetText("(just now)")

	// Populate accounts table
	t.accountsDataMu.Lock()
	t.accountsData = [][]string{}
	for _, acc := range testResult.SampleAccounts {
		row := []string{
			acc.ID,
			fmt.Sprintf("%d", acc.PackCount),
			"N/A",
			string(acc.Status),
		}
		t.accountsData = append(t.accountsData, row)
	}
	t.accountsDataMu.Unlock()

	if t.accountsTable != nil {
		fyne.Do(func() { t.accountsTable.Refresh() })
	}
}

// handleAddQuery adds a new query
func (t *AccountPoolsTabV2) handleAddQuery() {
	dialog.ShowInformation("Add Query", "Query builder UI not yet implemented", t.window)
	// TODO: Implement query builder dialog
}

// handleEditQuery edits an existing query
func (t *AccountPoolsTabV2) handleEditQuery(id int) {
	dialog.ShowInformation("Edit Query", fmt.Sprintf("Edit query %d not yet implemented", id), t.window)
	// TODO: Implement query builder dialog
}

// handleDeleteQuery deletes a query
func (t *AccountPoolsTabV2) handleDeleteQuery(id int) {
	t.queriesDataMu.Lock()
	defer t.queriesDataMu.Unlock()

	if id < len(t.queriesData) {
		t.queriesData = append(t.queriesData[:id], t.queriesData[id+1:]...)
		t.markDirty()
		fyne.Do(func() { t.queriesList.Refresh() })
	}
}

// handleAddInclude adds an account to include list
func (t *AccountPoolsTabV2) handleAddInclude(accountID string) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return
	}

	t.includesDataMu.Lock()
	defer t.includesDataMu.Unlock()

	// Check duplicates
	for _, existing := range t.includesData {
		if existing == accountID {
			return
		}
	}

	t.includesData = append(t.includesData, accountID)
	t.markDirty()
	fyne.Do(func() { t.includesList.Refresh() })
}

// handleRemoveInclude removes an account from include list
func (t *AccountPoolsTabV2) handleRemoveInclude(id int) {
	t.includesDataMu.Lock()
	defer t.includesDataMu.Unlock()

	if id < len(t.includesData) {
		t.includesData = append(t.includesData[:id], t.includesData[id+1:]...)
		t.markDirty()
		fyne.Do(func() { t.includesList.Refresh() })
	}
}

// handleAddExclude adds an account to exclude list
func (t *AccountPoolsTabV2) handleAddExclude(accountID string) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return
	}

	t.excludesDataMu.Lock()
	defer t.excludesDataMu.Unlock()

	// Check duplicates
	for _, existing := range t.excludesData {
		if existing == accountID {
			return
		}
	}

	t.excludesData = append(t.excludesData, accountID)
	t.markDirty()
	fyne.Do(func() { t.excludesList.Refresh() })
}

// handleRemoveExclude removes an account from exclude list
func (t *AccountPoolsTabV2) handleRemoveExclude(id int) {
	t.excludesDataMu.Lock()
	defer t.excludesDataMu.Unlock()

	if id < len(t.excludesData) {
		t.excludesData = append(t.excludesData[:id], t.excludesData[id+1:]...)
		t.markDirty()
		fyne.Do(func() { t.excludesList.Refresh() })
	}
}

// === POOL LIST MANAGEMENT ===

func (t *AccountPoolsTabV2) loadExistingPools() {
	if t.poolManager == nil {
		return
	}

	t.clearAllCards()

	if err := t.poolManager.DiscoverPools(); err != nil {
		fmt.Printf("Warning: Failed to discover pools: %v\n", err)
		return
	}

	poolNames := t.poolManager.ListPools()
	for _, poolName := range poolNames {
		t.addPoolCard(poolName)
	}

	t.updateStatusLabel()
}

func (t *AccountPoolsTabV2) addPoolCard(poolName string) {
	poolDef, err := t.poolManager.GetPoolDefinition(poolName)
	if err != nil {
		fmt.Printf("Warning: Failed to get pool definition for '%s': %v\n", poolName, err)
		return
	}

	testResult, err := t.poolManager.TestPool(poolName)
	accountCount := 0
	if err == nil && testResult != nil {
		accountCount = testResult.AccountsFound
	}

	card := components.NewAccountPoolCard(
		poolName,
		"unified",
		accountCount,
		"recently",
		poolDef.Config.Description,
		components.AccountPoolCardCallbacks{
			OnSelect: func(name string) {
				t.handleSelectPool(name)
			},
		},
	)

	t.poolCardsMu.Lock()
	t.poolCards[poolName] = card
	t.poolCardsMu.Unlock()

	fyne.Do(func() {
		t.poolListContainer.Add(card.GetContainer())
		t.poolListContainer.Refresh()
	})
}

func (t *AccountPoolsTabV2) clearAllCards() {
	t.poolCardsMu.Lock()
	t.poolCards = make(map[string]*components.AccountPoolCard)
	t.poolCardsMu.Unlock()

	fyne.Do(func() {
		t.poolListContainer.Objects = nil
		t.poolListContainer.Refresh()
	})
}

func (t *AccountPoolsTabV2) updateStatusLabel() {
	t.poolCardsMu.RLock()
	count := len(t.poolCards)
	t.poolCardsMu.RUnlock()

	fyne.Do(func() {
		if count == 0 {
			t.statusLabel.SetText("No pools created")
		} else if count == 1 {
			t.statusLabel.SetText("1 pool")
		} else {
			t.statusLabel.SetText(fmt.Sprintf("%d pools", count))
		}
	})
}

// Stop stops the tab (cleanup)
func (t *AccountPoolsTabV2) Stop() {
	close(t.stopRefresh)
}
