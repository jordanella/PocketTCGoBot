package gui

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
)

// AccountPoolsTab manages the Account Pools GUI tab
type AccountPoolsTab struct {
	controller  *Controller
	poolManager *accountpool.PoolManager
	db          *sql.DB

	// UI Components
	content       *fyne.Container
	poolList      *widget.List
	poolsData     []PoolListItem
	selectedIndex int

	// Bindings
	poolNameLabel      binding.String
	poolTypeLabel      binding.String
	poolDescLabel      binding.String
	accountCountLabel  binding.String
	lastRefreshLabel   binding.String

	// Buttons
	refreshBtn *widget.Button
	testBtn    *widget.Button
	editBtn    *widget.Button
	deleteBtn  *widget.Button
	createBtn  *widget.Button
}

// PoolListItem represents a pool in the list view
type PoolListItem struct {
	Name         string
	Type         string
	Description  string
	AccountCount int
	LastRefresh  string
	FilePath     string
}

// NewAccountPoolsTab creates a new Account Pools tab
func NewAccountPoolsTab(controller *Controller, poolManager *accountpool.PoolManager, db *sql.DB) *AccountPoolsTab {
	tab := &AccountPoolsTab{
		controller:        controller,
		poolManager:       poolManager,
		db:                db,
		poolsData:         make([]PoolListItem, 0),
		selectedIndex:     -1,
		poolNameLabel:     binding.NewString(),
		poolTypeLabel:     binding.NewString(),
		poolDescLabel:     binding.NewString(),
		accountCountLabel: binding.NewString(),
		lastRefreshLabel:  binding.NewString(),
	}

	tab.buildUI()
	tab.refreshPoolList()

	return tab
}

// buildUI constructs the UI layout
func (t *AccountPoolsTab) buildUI() {
	// Left panel: Pool list
	leftPanel := t.buildPoolListPanel()

	// Right panel: Pool details and actions
	rightPanel := t.buildDetailsPanel()

	// Main layout: split view
	split := container.NewHSplit(
		leftPanel,
		rightPanel,
	)
	split.Offset = 0.4 // 40% for list, 60% for details

	t.content = container.NewBorder(
		t.buildToolbar(),
		nil,
		nil,
		nil,
		split,
	)
}

// buildToolbar creates the top toolbar
func (t *AccountPoolsTab) buildToolbar() *fyne.Container {
	t.createBtn = widget.NewButton("+ New Pool", t.onCreatePool)

	refreshAllBtn := widget.NewButton("Refresh All", func() {
		t.refreshPoolList()
		t.showInfo("All pools refreshed")
	})

	return container.NewBorder(
		nil,
		nil,
		container.NewHBox(t.createBtn),
		container.NewHBox(refreshAllBtn),
		widget.NewLabel("Account Pools"),
	)
}

// buildPoolListPanel creates the left panel with pool list
func (t *AccountPoolsTab) buildPoolListPanel() *fyne.Container {
	t.poolList = widget.NewList(
		func() int {
			return len(t.poolsData)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("Pool Name"),
				widget.NewLabel("Type: SQL | Accounts: 0"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(t.poolsData) {
				return
			}

			pool := t.poolsData[id]
			vbox := obj.(*fyne.Container)

			nameLabel := vbox.Objects[0].(*widget.Label)
			infoLabel := vbox.Objects[1].(*widget.Label)

			nameLabel.SetText(pool.Name)
			infoLabel.SetText(fmt.Sprintf("Type: %s | Accounts: %d", pool.Type, pool.AccountCount))
		},
	)

	t.poolList.OnSelected = func(id widget.ListItemID) {
		t.selectedIndex = id
		t.updateDetailsPanel()
	}

	return container.NewBorder(
		widget.NewLabel("Available Pools"),
		nil,
		nil,
		nil,
		t.poolList,
	)
}

// buildDetailsPanel creates the right panel with pool details
func (t *AccountPoolsTab) buildDetailsPanel() *fyne.Container {
	// Pool info display
	nameLabel := widget.NewLabelWithData(t.poolNameLabel)
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	typeLabel := widget.NewLabelWithData(t.poolTypeLabel)
	descLabel := widget.NewLabelWithData(t.poolDescLabel)
	descLabel.Wrapping = fyne.TextWrapWord

	accountLabel := widget.NewLabelWithData(t.accountCountLabel)
	refreshLabel := widget.NewLabelWithData(t.lastRefreshLabel)

	infoCard := widget.NewCard("Pool Information", "", container.NewVBox(
		container.NewHBox(widget.NewLabel("Name:"), nameLabel),
		container.NewHBox(widget.NewLabel("Type:"), typeLabel),
		container.NewHBox(widget.NewLabel("Description:"), descLabel),
		container.NewHBox(widget.NewLabel("Accounts:"), accountLabel),
		container.NewHBox(widget.NewLabel("Last Refresh:"), refreshLabel),
	))

	// Action buttons
	t.refreshBtn = widget.NewButton("Refresh", t.onRefreshPool)
	t.testBtn = widget.NewButton("Test", t.onTestPool)
	t.editBtn = widget.NewButton("Edit", t.onEditPool)
	t.deleteBtn = widget.NewButton("Delete", t.onDeletePool)

	t.refreshBtn.Disable()
	t.testBtn.Disable()
	t.editBtn.Disable()
	t.deleteBtn.Disable()

	actionsCard := widget.NewCard("Actions", "", container.NewVBox(
		t.refreshBtn,
		t.testBtn,
		t.editBtn,
		t.deleteBtn,
	))

	// Initial message when no pool selected
	noSelectionLabel := widget.NewLabel("Select a pool from the list to view details")
	noSelectionLabel.Alignment = fyne.TextAlignCenter

	return container.NewBorder(
		nil,
		actionsCard,
		nil,
		nil,
		container.NewVBox(
			infoCard,
			noSelectionLabel,
		),
	)
}

// Content returns the tab content
func (t *AccountPoolsTab) Content() fyne.CanvasObject {
	return t.content
}

// refreshPoolList reloads the pool list from PoolManager
func (t *AccountPoolsTab) refreshPoolList() {
	// Discover pools
	if err := t.poolManager.DiscoverPools(); err != nil {
		t.showError("Failed to discover pools", err)
		return
	}

	// Get pool names
	poolNames := t.poolManager.ListPools()

	// Build pool list items
	newPoolsData := make([]PoolListItem, 0, len(poolNames))

	for _, name := range poolNames {
		poolDef, err := t.poolManager.GetPoolDefinition(name)
		if err != nil {
			continue
		}

		item := PoolListItem{
			Name:     poolDef.Name,
			Type:     poolDef.Type,
			FilePath: poolDef.FilePath,
		}

		// Get description and account count based on type
		switch poolDef.Type {
		case "sql":
			queryDef := poolDef.Config.(*accountpool.QueryDefinition)
			item.Description = queryDef.Description

			// Test pool to get account count
			testResult, err := t.poolManager.TestPool(name)
			if err == nil && testResult.Success {
				item.AccountCount = testResult.AccountsFound
			}

		case "file":
			fileConfig := poolDef.Config.(*accountpool.FilePoolConfig)
			item.Description = fmt.Sprintf("Directory: %s", fileConfig.Directory)

			// Test pool to get account count
			testResult, err := t.poolManager.TestPool(name)
			if err == nil && testResult.Success {
				item.AccountCount = testResult.AccountsFound
			}
		}

		newPoolsData = append(newPoolsData, item)
	}

	t.poolsData = newPoolsData
	t.poolList.Refresh()

	// Clear selection if index is out of bounds
	if t.selectedIndex >= len(t.poolsData) {
		t.selectedIndex = -1
		t.poolList.UnselectAll()
		t.updateDetailsPanel()
	}
}

// updateDetailsPanel updates the details panel for selected pool
func (t *AccountPoolsTab) updateDetailsPanel() {
	if t.selectedIndex < 0 || t.selectedIndex >= len(t.poolsData) {
		// No selection
		t.poolNameLabel.Set("No pool selected")
		t.poolTypeLabel.Set("")
		t.poolDescLabel.Set("")
		t.accountCountLabel.Set("")
		t.lastRefreshLabel.Set("")

		t.refreshBtn.Disable()
		t.testBtn.Disable()
		t.editBtn.Disable()
		t.deleteBtn.Disable()
		return
	}

	pool := t.poolsData[t.selectedIndex]

	t.poolNameLabel.Set(pool.Name)
	t.poolTypeLabel.Set(pool.Type)
	t.poolDescLabel.Set(pool.Description)
	t.accountCountLabel.Set(strconv.Itoa(pool.AccountCount))
	t.lastRefreshLabel.Set(pool.LastRefresh)

	t.refreshBtn.Enable()
	t.testBtn.Enable()
	t.editBtn.Enable()
	t.deleteBtn.Enable()
}

// Action handlers

func (t *AccountPoolsTab) onCreatePool() {
	dialog := t.buildCreatePoolDialog()
	dialog.Show()
}

func (t *AccountPoolsTab) onRefreshPool() {
	if t.selectedIndex < 0 {
		return
	}

	pool := t.poolsData[t.selectedIndex]

	// Refresh the pool
	if err := t.poolManager.RefreshPool(pool.Name); err != nil {
		t.showError("Failed to refresh pool", err)
		return
	}

	t.showInfo(fmt.Sprintf("Pool '%s' refreshed successfully", pool.Name))
	t.refreshPoolList()
}

func (t *AccountPoolsTab) onTestPool() {
	if t.selectedIndex < 0 {
		return
	}

	pool := t.poolsData[t.selectedIndex]

	// Test the pool
	testResult, err := t.poolManager.TestPool(pool.Name)
	if err != nil {
		t.showError("Failed to test pool", err)
		return
	}

	// Show test results
	t.showTestResults(pool.Name, testResult)
}

func (t *AccountPoolsTab) onEditPool() {
	if t.selectedIndex < 0 {
		return
	}

	pool := t.poolsData[t.selectedIndex]

	// Get pool definition
	poolDef, err := t.poolManager.GetPoolDefinition(pool.Name)
	if err != nil {
		t.showError("Failed to load pool", err)
		return
	}

	// Show edit dialog based on type
	if poolDef.Type == "sql" {
		t.showSQLPoolEditor(poolDef)
	} else {
		t.showFilePoolEditor(poolDef)
	}
}

func (t *AccountPoolsTab) onDeletePool() {
	if t.selectedIndex < 0 {
		return
	}

	pool := t.poolsData[t.selectedIndex]

	// Confirm deletion
	dialog.ShowConfirm(
		"Delete Pool",
		fmt.Sprintf("Are you sure you want to delete pool '%s'?\n\nThis will remove the pool definition file.", pool.Name),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Delete pool
			if err := t.poolManager.DeletePool(pool.Name); err != nil {
				t.showError("Failed to delete pool", err)
				return
			}

			t.showInfo(fmt.Sprintf("Pool '%s' deleted successfully", pool.Name))
			t.refreshPoolList()
		},
		t.controller.window,
	)
}

// buildCreatePoolDialog creates the pool creation dialog
func (t *AccountPoolsTab) buildCreatePoolDialog() dialog.Dialog {
	poolNameEntry := widget.NewEntry()
	poolNameEntry.SetPlaceHolder("Enter pool name")

	poolTypeSelect := widget.NewSelect([]string{"SQL Pool", "File Pool"}, nil)
	poolTypeSelect.SetSelected("SQL Pool")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Pool Name", Widget: poolNameEntry},
			{Text: "Pool Type", Widget: poolTypeSelect},
		},
	}

	dlg := dialog.NewCustomConfirm(
		"Create New Pool",
		"Next",
		"Cancel",
		form,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			name := poolNameEntry.Text
			if name == "" {
				dialog.ShowError(fmt.Errorf("pool name is required"), t.controller.window)
				return
			}

			// Open appropriate wizard
			if poolTypeSelect.Selected == "SQL Pool" {
				t.showSQLPoolWizard(name)
			} else {
				t.showFilePoolWizard(name)
			}
		},
		t.controller.window,
	)

	dlg.Resize(fyne.NewSize(400, 200))
	return dlg
}

// Helper methods

func (t *AccountPoolsTab) showInfo(message string) {
	dialog.ShowInformation("Success", message, t.controller.window)
}

func (t *AccountPoolsTab) showError(title string, err error) {
	dialog.ShowError(fmt.Errorf("%s: %v", title, err), t.controller.window)
}

func (t *AccountPoolsTab) showTestResults(poolName string, result *accountpool.TestResult) {
	var message string
	if result.Success {
		message = fmt.Sprintf("Pool '%s' tested successfully!\n\nAccounts found: %d\n\nThe pool is ready to use.",
			poolName, result.AccountsFound)
	} else {
		message = fmt.Sprintf("Pool '%s' test failed!\n\nError: %s", poolName, result.Error)
	}

	if result.Success {
		dialog.ShowInformation("Pool Test Results", message, t.controller.window)
	} else {
		dialog.ShowError(fmt.Errorf(message), t.controller.window)
	}
}

// SQL Pool Wizard and Editor
func (t *AccountPoolsTab) showSQLPoolWizard(name string) {
	qb := NewQueryBuilder(t.controller.window, name, func(queryDef *accountpool.QueryDefinition) {
		// Create pool definition
		poolDef := &accountpool.PoolDefinition{
			Name:   queryDef.Name,
			Type:   "sql",
			Config: queryDef,
		}

		// Create pool
		if err := t.poolManager.CreatePool(poolDef); err != nil {
			t.showError("Failed to create pool", err)
			return
		}

		t.showInfo(fmt.Sprintf("SQL pool '%s' created successfully", queryDef.Name))
		t.refreshPoolList()
	})

	qb.Show()
}

func (t *AccountPoolsTab) showSQLPoolEditor(poolDef *accountpool.PoolDefinition) {
	queryDef := poolDef.Config.(*accountpool.QueryDefinition)

	qb := NewQueryBuilder(t.controller.window, queryDef.Name, func(updatedQueryDef *accountpool.QueryDefinition) {
		// Update pool definition
		poolDef.Name = updatedQueryDef.Name
		poolDef.Config = updatedQueryDef

		// Update pool
		if err := t.poolManager.UpdatePool(queryDef.Name, poolDef); err != nil {
			t.showError("Failed to update pool", err)
			return
		}

		t.showInfo(fmt.Sprintf("Pool '%s' updated successfully", updatedQueryDef.Name))
		t.refreshPoolList()
	})

	// TODO: Load existing query definition into query builder

	qb.Show()
}

// File Pool Wizard and Editor
func (t *AccountPoolsTab) showFilePoolWizard(name string) {
	dirEntry := widget.NewEntry()
	dirEntry.SetPlaceHolder("./accounts/manual")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Directory", Widget: dirEntry},
		},
	}

	dialog.ShowCustomConfirm(
		fmt.Sprintf("Create File Pool: %s", name),
		"Create",
		"Cancel",
		form,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			directory := dirEntry.Text
			if directory == "" {
				dialog.ShowError(fmt.Errorf("directory is required"), t.controller.window)
				return
			}

			// Create file pool definition
			fileConfig := &accountpool.FilePoolConfig{
				Name:      name,
				Type:      "file",
				Directory: directory,
				PoolConfig: accountpool.PoolConfig{
					RetryFailed:   false,
					MaxFailures:   1,
					BufferSize:    10,
					RefreshInterval: 0,
				},
			}

			poolDef := &accountpool.PoolDefinition{
				Name:   name,
				Type:   "file",
				Config: fileConfig,
			}

			// Create pool
			if err := t.poolManager.CreatePool(poolDef); err != nil {
				t.showError("Failed to create pool", err)
				return
			}

			t.showInfo(fmt.Sprintf("File pool '%s' created successfully", name))
			t.refreshPoolList()
		},
		t.controller.window,
	)
}

func (t *AccountPoolsTab) showFilePoolEditor(poolDef *accountpool.PoolDefinition) {
	fileConfig := poolDef.Config.(*accountpool.FilePoolConfig)

	dirEntry := widget.NewEntry()
	dirEntry.SetText(fileConfig.Directory)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Directory", Widget: dirEntry},
		},
	}

	dialog.ShowCustomConfirm(
		fmt.Sprintf("Edit File Pool: %s", poolDef.Name),
		"Save",
		"Cancel",
		form,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			directory := dirEntry.Text
			if directory == "" {
				dialog.ShowError(fmt.Errorf("directory is required"), t.controller.window)
				return
			}

			// Update config
			fileConfig.Directory = directory

			// Update pool
			if err := t.poolManager.UpdatePool(poolDef.Name, poolDef); err != nil {
				t.showError("Failed to update pool", err)
				return
			}

			t.showInfo(fmt.Sprintf("Pool '%s' updated successfully", poolDef.Name))
			t.refreshPoolList()
		},
		t.controller.window,
	)
}
