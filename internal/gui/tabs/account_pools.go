package tabs

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/gui/components"
)

// AccountPoolsTab manages the account pools view using component library
type AccountPoolsTab struct {
	// Dependencies (injected)
	poolManager *accountpool.PoolManager
	db          *sql.DB
	window      fyne.Window

	// UI state
	poolCards         map[string]*components.AccountPoolCard // key = pool name
	poolCardsMu       sync.RWMutex
	selectedPoolName  string
	poolListContainer *fyne.Container

	// Right panel tabs
	tabContainer *container.AppTabs

	// Details tab elements
	detailsContainer    *fyne.Container
	poolNameLabel       *widget.Label
	descText            *widget.Label
	totalAccountsValue  *widget.Label
	lastUpdatedLabel    *widget.Label
	inclusionsValue     *widget.Label
	exclusionsValue     *widget.Label

	// Accounts tab elements
	accountsTable     *widget.Table
	accountsData      [][]string
	accountsDataMu    sync.RWMutex

	// UI elements
	statusLabel *widget.Label
	newBtn      *widget.Button
	refreshBtn  *widget.Button

	// Refresh control
	stopRefresh chan bool
}

// NewAccountPoolsTab creates a new account pools tab
func NewAccountPoolsTab(poolManager *accountpool.PoolManager, db *sql.DB, window fyne.Window) *AccountPoolsTab {
	return &AccountPoolsTab{
		poolManager:       poolManager,
		db:                db,
		window:            window,
		poolCards:         make(map[string]*components.AccountPoolCard),
		poolListContainer: container.NewVBox(),
		stopRefresh:       make(chan bool),
	}
}

// Build constructs the tab UI matching the mockup
func (t *AccountPoolsTab) Build() fyne.CanvasObject {
	// === HEADER ===
	header := components.Heading("Account Pool Management")

	// === CONTROLS ===
	t.newBtn = components.PrimaryButton("New", func() {
		t.handleNewPool()
	})

	t.refreshBtn = components.SecondaryButton("Refresh", func() {
		t.refreshAllPools()
	})

	quickLaunchBtn := components.SecondaryButton("Quick Launch", func() {
		t.handleQuickLaunch()
	})

	t.statusLabel = widget.NewLabel("No pools loaded")

	controls := container.NewHBox(
		t.newBtn,
		t.refreshBtn,
		quickLaunchBtn,
		widget.NewLabel(""), // Spacer
		t.statusLabel,
	)

	// === LEFT PANEL: Pool List ===
	leftPanel := t.buildPoolListPanel()

	// === RIGHT PANEL: Tabbed Details ===
	rightPanel := t.buildDetailsPanel()

	// === MAIN CONTENT ===
	split := container.NewHSplit(
		leftPanel,
		rightPanel,
	)
	split.Offset = 0.3 // 30% for list, 70% for details

	content := container.NewBorder(
		container.NewVBox(
			header,
			widget.NewSeparator(),
			controls,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		split,
	)

	// Load existing pools
	t.loadExistingPools()

	// Start periodic refresh
	go t.startPeriodicRefresh()

	return content
}

// buildPoolListPanel creates the left panel with pool cards
func (t *AccountPoolsTab) buildPoolListPanel() *fyne.Container {
	poolListLabel := components.Subheading("Pool List")

	// Scroll container for pool cards
	scroll := container.NewVScroll(t.poolListContainer)
	scroll.SetMinSize(fyne.NewSize(300, 600))

	// Add new pool button at bottom
	newPoolBtn := components.SecondaryButton("+ New Pool", func() {
		t.handleNewPool()
	})

	return container.NewBorder(
		poolListLabel,
		newPoolBtn,
		nil, nil,
		scroll,
	)
}

// buildDetailsPanel creates the right panel with tabs
func (t *AccountPoolsTab) buildDetailsPanel() *fyne.Container {
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

	// Pool name header
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

// buildDetailsTab creates the Details tab content
func (t *AccountPoolsTab) buildDetailsTab() fyne.CanvasObject {
	// Description
	descLabel := components.Subheading("Description")
	editDescBtn := components.SecondaryButton("Edit", func() {
		t.handleEditDescription()
	})
	descHeader := container.NewBorder(nil, nil, descLabel, editDescBtn, container.NewHBox())

	t.descText = widget.NewLabel("Select a pool to view details")
	t.descText.Wrapping = fyne.TextWrapWord

	// Total Accounts
	totalAccountsLabel := components.BoldText("Total Accounts:")
	t.totalAccountsValue = widget.NewLabel("0")
	t.lastUpdatedLabel = widget.NewLabel("(last updated ...)")
	refreshPoolBtn := components.SecondaryButton("Refresh", func() {
		t.handleRefreshPool()
	})

	accountsRow := container.NewHBox(
		totalAccountsLabel,
		t.totalAccountsValue,
		t.lastUpdatedLabel,
		refreshPoolBtn,
	)

	// Queries section
	queriesLabel := components.Subheading("Queries")
	queriesContent := widget.NewLabel("No queries configured")

	// Inclusions/Exclusions
	inclusionsLabel := components.BoldText("Inclusions:")
	t.inclusionsValue = widget.NewLabel("0")
	inclusionsEditBtn := components.SecondaryButton("Edit", func() {
		// Switch to Include tab
		t.tabContainer.SelectIndex(3)
	})

	exclusionsLabel := components.BoldText("Exclusions:")
	t.exclusionsValue = widget.NewLabel("0")
	exclusionsEditBtn := components.SecondaryButton("Edit", func() {
		// Switch to Exclude tab
		t.tabContainer.SelectIndex(4)
	})

	// Sorting section
	sortingLabel := components.Subheading("Sorting")
	sortingContent := widget.NewLabel("No sorting configured")

	// Limit
	limitLabel := components.BoldText("Limit:")
	limitEntry := widget.NewEntry()
	limitEntry.SetPlaceHolder("No limit")

	// Auto-Refresh
	autoRefreshLabel := components.Subheading("Auto-Refresh")
	enabledCheck := widget.NewCheck("Enabled", func(bool) {})
	frequencyLabel := components.BoldText("Frequency:")
	frequencyEntry := widget.NewEntry()
	frequencyEntry.SetPlaceHolder("60s")

	// Action buttons
	saveBtn := components.PrimaryButton("Save Changes", func() {
		t.handleSaveChanges()
	})
	discardBtn := components.SecondaryButton("Discard Changes", func() {
		t.handleDiscardChanges()
	})
	deleteBtn := components.DangerButton("Delete Pool", func() {
		t.handleDeletePool()
	})

	actions := container.NewHBox(saveBtn, discardBtn, deleteBtn)

	t.detailsContainer = container.NewVBox(
		descHeader,
		t.descText,
		widget.NewSeparator(),
		accountsRow,
		widget.NewSeparator(),
		queriesLabel,
		queriesContent,
		widget.NewSeparator(),
		container.NewHBox(inclusionsLabel, t.inclusionsValue, inclusionsEditBtn),
		container.NewHBox(exclusionsLabel, t.exclusionsValue, exclusionsEditBtn),
		widget.NewSeparator(),
		sortingLabel,
		sortingContent,
		widget.NewSeparator(),
		container.NewHBox(limitLabel, limitEntry),
		widget.NewSeparator(),
		autoRefreshLabel,
		enabledCheck,
		container.NewHBox(frequencyLabel, frequencyEntry),
		widget.NewSeparator(),
		actions,
	)

	return container.NewVScroll(t.detailsContainer)
}

// buildAccountsTab creates the Accounts tab content
func (t *AccountPoolsTab) buildAccountsTab() fyne.CanvasObject {
	// Initialize accounts data
	t.accountsData = [][]string{}

	// Table for displaying accounts
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

	// Column headers
	headers := []string{"Account", "Packs", "Shinedust", "Status"}
	t.accountsTable.UpdateHeader = func(id widget.TableCellID, obj fyne.CanvasObject) {
		if id.Col < len(headers) {
			obj.(*widget.Label).SetText(headers[id.Col])
		}
	}

	// Set column widths
	t.accountsTable.SetColumnWidth(0, 200) // Account
	t.accountsTable.SetColumnWidth(1, 80)  // Packs
	t.accountsTable.SetColumnWidth(2, 100) // Shinedust
	t.accountsTable.SetColumnWidth(3, 150) // Status

	return container.NewVScroll(t.accountsTable)
}

// populateAccountsTable fills the accounts table with data from test result
func (t *AccountPoolsTab) populateAccountsTable(testResult *accountpool.TestResult) {
	if testResult == nil {
		return
	}

	t.accountsDataMu.Lock()
	defer t.accountsDataMu.Unlock()

	// Clear existing data
	t.accountsData = [][]string{}

	// Populate with sample accounts
	for _, acc := range testResult.SampleAccounts {
		row := []string{
			acc.ID,
			fmt.Sprintf("%d", acc.PackCount),
			"N/A", // Shinedust not available in AccountSummary
			string(acc.Status),
		}
		t.accountsData = append(t.accountsData, row)
	}

	// Refresh table
	if t.accountsTable != nil {
		t.accountsTable.Refresh()
	}
}

// buildQueriesTab creates the Queries tab content
func (t *AccountPoolsTab) buildQueriesTab() fyne.CanvasObject {
	label := widget.NewLabel("Query management coming soon")
	label.Alignment = fyne.TextAlignCenter

	addQueryBtn := components.PrimaryButton("+ Query", func() {
		dialog.ShowInformation("Add Query", "Query builder not yet implemented", t.window)
	})

	return container.NewBorder(
		nil,
		addQueryBtn,
		nil, nil,
		container.NewVBox(label),
	)
}

// buildIncludeTab creates the Include tab content
func (t *AccountPoolsTab) buildIncludeTab() fyne.CanvasObject {
	accountEntry := widget.NewEntry()
	accountEntry.SetPlaceHolder("deviceAccount")

	includeBtn := components.PrimaryButton("Include Account", func() {
		if accountEntry.Text != "" {
			dialog.ShowInformation("Include", fmt.Sprintf("Including account: %s", accountEntry.Text), t.window)
		}
	})

	header := container.NewBorder(
		nil, nil,
		accountEntry,
		includeBtn,
		container.NewHBox(),
	)

	// Table for included accounts
	includeTable := widget.NewTable(
		func() (int, int) { return 0, 4 },
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText("No accounts")
		},
	)

	return container.NewBorder(
		header,
		nil, nil, nil,
		container.NewVScroll(includeTable),
	)
}

// buildExcludeTab creates the Exclude tab content
func (t *AccountPoolsTab) buildExcludeTab() fyne.CanvasObject {
	accountEntry := widget.NewEntry()
	accountEntry.SetPlaceHolder("deviceAccount")

	excludeBtn := components.DangerButton("Exclude Account", func() {
		if accountEntry.Text != "" {
			dialog.ShowInformation("Exclude", fmt.Sprintf("Excluding account: %s", accountEntry.Text), t.window)
		}
	})

	header := container.NewBorder(
		nil, nil,
		accountEntry,
		excludeBtn,
		container.NewHBox(),
	)

	// Table for excluded accounts
	excludeTable := widget.NewTable(
		func() (int, int) { return 0, 4 },
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText("No accounts")
		},
	)

	return container.NewBorder(
		header,
		nil, nil, nil,
		container.NewVScroll(excludeTable),
	)
}

// loadExistingPools loads all pools from the pool manager
func (t *AccountPoolsTab) loadExistingPools() {
	if t.poolManager == nil {
		return
	}

	// Clear existing cards
	t.clearAllCards()

	// Discover pools
	if err := t.poolManager.DiscoverPools(); err != nil {
		fmt.Printf("Warning: Failed to discover pools: %v\n", err)
		return
	}

	// Get pool names
	poolNames := t.poolManager.ListPools()

	// Create cards for each pool
	for _, poolName := range poolNames {
		t.addPoolCard(poolName)
	}

	t.updateStatusLabel()
}

// addPoolCard creates and adds a card for a pool
func (t *AccountPoolsTab) addPoolCard(poolName string) {
	// Get pool definition
	poolDef, err := t.poolManager.GetPoolDefinition(poolName)
	if err != nil {
		fmt.Printf("Warning: Failed to get pool definition for '%s': %v\n", poolName, err)
		return
	}

	// Test pool to get account count
	testResult, err := t.poolManager.TestPool(poolName)
	accountCount := 0
	if err == nil && testResult != nil {
		accountCount = testResult.AccountsFound
	}

	// Create card
	card := components.NewAccountPoolCard(
		poolName,
		"unified",
		accountCount,
		"recently",
		poolDef.Config.Description,
		components.AccountPoolCardCallbacks{
			OnSelect: t.handleSelectPool,
		},
	)

	// Add to tracking
	t.poolCardsMu.Lock()
	t.poolCards[poolName] = card
	t.poolCardsMu.Unlock()

	// Add to UI
	t.poolListContainer.Add(card.GetContainer())
	t.poolListContainer.Refresh()
}

// clearAllCards removes all pool cards
func (t *AccountPoolsTab) clearAllCards() {
	t.poolCardsMu.Lock()
	t.poolCards = make(map[string]*components.AccountPoolCard)
	t.poolCardsMu.Unlock()

	t.poolListContainer.Objects = nil
	t.poolListContainer.Refresh()
}

// Card action handlers

func (t *AccountPoolsTab) handleSelectPool(poolName string) {
	t.selectedPoolName = poolName

	// Update all cards to show selection state
	t.poolCardsMu.RLock()
	for name, card := range t.poolCards {
		card.SetSelected(name == poolName)
	}
	t.poolCardsMu.RUnlock()

	// Update details panel
	t.updateDetailsPanel()

	fmt.Printf("Selected pool: %s\n", poolName)
}

func (t *AccountPoolsTab) handleNewPool() {
	// Show input dialog for pool name
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter pool name...")

	dlg := dialog.NewCustomConfirm("New Pool", "Create", "Cancel",
		container.NewVBox(
			widget.NewLabel("Enter a name for the new pool:"),
			nameEntry,
		),
		func(create bool) {
			if !create {
				return
			}

			poolName := strings.TrimSpace(nameEntry.Text)
			if poolName == "" {
				dialog.ShowError(fmt.Errorf("pool name cannot be empty"), t.window)
				return
			}

			// Check if pool already exists
			existingPools := t.poolManager.ListPools()
			for _, existingName := range existingPools {
				if existingName == poolName {
					dialog.ShowError(fmt.Errorf("pool '%s' already exists", poolName), t.window)
					return
				}
			}

			// Launch wizard
			wizard := NewUnifiedPoolWizard(t.window, poolName, func(poolDef *accountpool.UnifiedPoolDefinition) {
				// Create pool definition
				def := &accountpool.PoolDefinition{
					Name:   poolDef.PoolName,
					Config: poolDef,
				}

				// Save pool (this will auto-save to YAML)
				if err := t.poolManager.CreatePool(def); err != nil {
					dialog.ShowError(fmt.Errorf("failed to create pool: %w", err), t.window)
					return
				}

				// Refresh pool list
				t.loadExistingPools()

				// Select the new pool
				t.handleSelectPool(poolName)

				dialog.ShowInformation("Success", fmt.Sprintf("Pool '%s' created successfully", poolName), t.window)
			})
			wizard.Show()
		},
		t.window,
	)
	dlg.Resize(fyne.NewSize(400, 150))
	dlg.Show()
}

func (t *AccountPoolsTab) handleQuickLaunch() {
	if t.selectedPoolName == "" {
		dialog.ShowInformation("Quick Launch", "Please select a pool first", t.window)
		return
	}
	dialog.ShowInformation("Quick Launch", fmt.Sprintf("Quick launch with pool '%s' not yet implemented", t.selectedPoolName), t.window)
}

func (t *AccountPoolsTab) handleRenamePool() {
	if t.selectedPoolName == "" {
		return
	}

	oldPoolName := t.selectedPoolName

	// Show input dialog for new pool name
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter new pool name...")
	nameEntry.SetText(oldPoolName)

	dlg := dialog.NewCustomConfirm("Rename Pool", "Rename", "Cancel",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Rename pool '%s' to:", oldPoolName)),
			nameEntry,
		),
		func(rename bool) {
			if !rename {
				return
			}

			newPoolName := strings.TrimSpace(nameEntry.Text)
			if newPoolName == "" {
				dialog.ShowError(fmt.Errorf("pool name cannot be empty"), t.window)
				return
			}

			if newPoolName == oldPoolName {
				// No change
				return
			}

			// Check if new name already exists
			existingPools := t.poolManager.ListPools()
			for _, existingName := range existingPools {
				if existingName == newPoolName {
					dialog.ShowError(fmt.Errorf("pool '%s' already exists", newPoolName), t.window)
					return
				}
			}

			// Get existing pool definition
			oldDef, err := t.poolManager.GetPoolDefinition(oldPoolName)
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to load pool: %w", err), t.window)
				return
			}

			// Create new definition with new name
			newDef := &accountpool.PoolDefinition{
				Name:   newPoolName,
				Config: oldDef.Config,
			}
			newDef.Config.PoolName = newPoolName

			// Delete old pool
			if err := t.poolManager.DeletePool(oldPoolName); err != nil {
				dialog.ShowError(fmt.Errorf("failed to delete old pool: %w", err), t.window)
				return
			}

			// Create new pool
			if err := t.poolManager.CreatePool(newDef); err != nil {
				dialog.ShowError(fmt.Errorf("failed to create renamed pool: %w", err), t.window)
				return
			}

			// Refresh pool list
			t.loadExistingPools()

			// Select the renamed pool
			t.handleSelectPool(newPoolName)

			dialog.ShowInformation("Success", fmt.Sprintf("Pool renamed from '%s' to '%s'", oldPoolName, newPoolName), t.window)
		},
		t.window,
	)
	dlg.Resize(fyne.NewSize(400, 150))
	dlg.Show()
}

func (t *AccountPoolsTab) handleEditDescription() {
	if t.selectedPoolName == "" {
		return
	}

	poolName := t.selectedPoolName

	// Get existing pool definition
	poolDef, err := t.poolManager.GetPoolDefinition(poolName)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to load pool: %w", err), t.window)
		return
	}

	// Launch wizard with existing data
	wizard := NewUnifiedPoolWizard(t.window, poolName, func(updatedDef *accountpool.UnifiedPoolDefinition) {
		// Update pool definition
		def := &accountpool.PoolDefinition{
			Name:   poolName,
			Config: updatedDef,
		}

		// Update pool (this will save to YAML)
		if err := t.poolManager.UpdatePool(poolName, def); err != nil {
			dialog.ShowError(fmt.Errorf("failed to update pool: %w", err), t.window)
			return
		}

		// Refresh pool list and details
		t.loadExistingPools()
		t.updateDetailsPanel()

		dialog.ShowInformation("Success", fmt.Sprintf("Pool '%s' updated successfully", poolName), t.window)
	})

	// Pre-populate wizard with existing data
	wizard.LoadFromDefinition(poolDef.Config)
	wizard.Show()
}

func (t *AccountPoolsTab) handleRefreshPool() {
	if t.selectedPoolName == "" {
		return
	}

	pool, err := t.poolManager.GetPool(t.selectedPoolName)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get pool: %w", err), t.window)
		return
	}

	if err := pool.Refresh(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to refresh pool: %w", err), t.window)
		return
	}

	dialog.ShowInformation("Refreshed", fmt.Sprintf("Pool '%s' refreshed successfully", t.selectedPoolName), t.window)
	t.updateDetailsPanel()
}

func (t *AccountPoolsTab) handleSaveChanges() {
	if t.selectedPoolName == "" {
		return
	}
	dialog.ShowInformation("Save", "Saving changes not yet implemented", t.window)
}

func (t *AccountPoolsTab) handleDiscardChanges() {
	if t.selectedPoolName == "" {
		return
	}
	t.updateDetailsPanel() // Reload from source
}

func (t *AccountPoolsTab) handleDeletePool() {
	if t.selectedPoolName == "" {
		return
	}

	poolName := t.selectedPoolName

	dialog.ShowConfirm(
		"Delete Pool",
		fmt.Sprintf("Are you sure you want to delete pool '%s'?\n\nThis will permanently delete the pool definition file.", poolName),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Delete pool (this will also delete the YAML file)
			if err := t.poolManager.DeletePool(poolName); err != nil {
				dialog.ShowError(fmt.Errorf("failed to delete pool: %w", err), t.window)
				return
			}

			// Clear selection
			t.selectedPoolName = ""

			// Refresh pool list
			t.loadExistingPools()

			// Reset details panel
			t.poolNameLabel.SetText("Select a pool")
			t.descText.SetText("Select a pool to view details")
			t.totalAccountsValue.SetText("0")
			t.lastUpdatedLabel.SetText("(last updated ...)")
			t.inclusionsValue.SetText("0")
			t.exclusionsValue.SetText("0")

			// Clear accounts table
			t.accountsDataMu.Lock()
			t.accountsData = [][]string{}
			t.accountsDataMu.Unlock()
			if t.accountsTable != nil {
				t.accountsTable.Refresh()
			}

			dialog.ShowInformation("Success", fmt.Sprintf("Pool '%s' deleted successfully", poolName), t.window)
		},
		t.window,
	)
}

// updateDetailsPanel updates the details panel with selected pool data
func (t *AccountPoolsTab) updateDetailsPanel() {
	if t.selectedPoolName == "" {
		return
	}

	// Update pool name label
	t.poolNameLabel.SetText(t.selectedPoolName)

	// Get pool definition
	poolDef, err := t.poolManager.GetPoolDefinition(t.selectedPoolName)
	if err != nil {
		fmt.Printf("Error loading pool definition: %v\n", err)
		return
	}

	// Update description
	if poolDef.Config != nil {
		t.descText.SetText(poolDef.Config.Description)
	} else {
		t.descText.SetText("No description available")
	}

	// Test pool to get account count and populate accounts table
	testResult, err := t.poolManager.TestPool(t.selectedPoolName)
	if err != nil {
		t.totalAccountsValue.SetText("Error loading")
		t.lastUpdatedLabel.SetText(fmt.Sprintf("(error: %v)", err))
	} else if testResult != nil {
		t.totalAccountsValue.SetText(fmt.Sprintf("%d", testResult.AccountsFound))
		t.lastUpdatedLabel.SetText("(just now)")

		// Populate accounts table
		t.populateAccountsTable(testResult)
	}

	// Get pool to check stats
	pool, err := t.poolManager.GetPool(t.selectedPoolName)
	if err == nil && pool != nil {
		stats := pool.GetStats()
		// TODO: Track manual inclusions/exclusions separately
		// For now, show placeholder values
		t.inclusionsValue.SetText("0")
		t.exclusionsValue.SetText("0")
		_ = stats // Use stats when inclusion/exclusion tracking is implemented
	}

	fmt.Printf("Updated details panel for pool: %s\n", t.selectedPoolName)
}

// refreshAllPools refreshes all pool data
func (t *AccountPoolsTab) refreshAllPools() {
	t.loadExistingPools()
	dialog.ShowInformation("Refreshed", "All pools refreshed", t.window)
}

// startPeriodicRefresh updates pool data periodically
func (t *AccountPoolsTab) startPeriodicRefresh() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Refresh pool data in background
			if t.poolManager != nil {
				t.poolManager.DiscoverPools()
			}
		case <-t.stopRefresh:
			return
		}
	}
}

// Stop stops the periodic refresh
func (t *AccountPoolsTab) Stop() {
	close(t.stopRefresh)
}

// updateStatusLabel updates the status label with pool count
func (t *AccountPoolsTab) updateStatusLabel() {
	t.poolCardsMu.RLock()
	count := len(t.poolCards)
	t.poolCardsMu.RUnlock()

	if count == 0 {
		t.statusLabel.SetText("No pools loaded")
	} else {
		t.statusLabel.SetText(fmt.Sprintf("%d pool(s) available", count))
	}
}
