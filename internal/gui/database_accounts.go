package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/database"
)

// DatabaseAccountsTab displays database accounts
type DatabaseAccountsTab struct {
	controller *Controller
	db         *database.DB

	// View mode
	viewMode    string // "cards" or "list"
	viewModeBtn *widget.Button

	// Content containers
	contentArea *fyne.Container
}

// NewDatabaseAccountsTab creates a new database accounts tab
func NewDatabaseAccountsTab(ctrl *Controller, db *database.DB) *DatabaseAccountsTab {
	return &DatabaseAccountsTab{
		controller: ctrl,
		db:         db,
		viewMode:   "cards",
	}
}

// Build constructs the UI
func (t *DatabaseAccountsTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Database - Accounts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// View toggle button
	t.viewModeBtn = widget.NewButton("Switch to List View", func() {
		t.toggleViewMode()
	})

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", func() {
		t.refresh()
	})

	// Toolbar
	toolbar := container.NewHBox(
		t.viewModeBtn,
		refreshBtn,
	)

	// Content area - use Stack instead of VBox to allow content to expand
	t.contentArea = container.NewStack()
	t.refresh()

	// Scrollable content (needed for card view)
	content := container.NewVScroll(t.contentArea)

	return container.NewBorder(
		container.NewVBox(header, toolbar),
		nil,
		nil,
		nil,
		content,
	)
}

// toggleViewMode switches between card and list view
func (t *DatabaseAccountsTab) toggleViewMode() {
	if t.viewMode == "cards" {
		t.viewMode = "list"
		t.viewModeBtn.SetText("Switch to Card View")
	} else {
		t.viewMode = "cards"
		t.viewModeBtn.SetText("Switch to List View")
	}
	t.refresh()
}

// refresh reloads the data
func (t *DatabaseAccountsTab) refresh() {
	// Don't refresh if content area not initialized yet
	if t.contentArea == nil {
		return
	}

	if t.db == nil {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("Database not initialized"),
		}
		t.contentArea.Refresh()
		return
	}

	// Get active accounts
	accounts, err := t.db.ListActiveAccounts()
	if err != nil {
		if t.controller.window != nil {
			dialog.ShowError(err, t.controller.window)
		}
		return
	}

	if len(accounts) == 0 {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("No accounts in database"),
		}
		t.contentArea.Refresh()
		return
	}

	// Build view based on mode
	if t.viewMode == "cards" {
		t.contentArea.Objects = []fyne.CanvasObject{
			t.buildCardsView(accounts),
		}
	} else {
		t.contentArea.Objects = []fyne.CanvasObject{
			t.buildListView(accounts),
		}
	}

	t.contentArea.Refresh()
}

// buildCardsView creates a grid of account cards
func (t *DatabaseAccountsTab) buildCardsView(accounts []*database.Account) fyne.CanvasObject {
	cards := container.NewGridWithColumns(2)

	for _, acc := range accounts {
		card := t.createAccountCard(acc)
		cards.Add(card)
	}

	return cards
}

// createAccountCard creates a card widget for an account
func (t *DatabaseAccountsTab) createAccountCard(acc *database.Account) fyne.CanvasObject {
	// Title
	title := acc.DeviceAccount
	if acc.Username != nil && *acc.Username != "" {
		title = *acc.Username
	}
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Stats
	levelLabel := widget.NewLabel(fmt.Sprintf("Level: %d", acc.AccountLevel))
	packsLabel := widget.NewLabel(fmt.Sprintf("Packs: %d", acc.PacksOpened))
	picksLabel := widget.NewLabel(fmt.Sprintf("Wonder Picks: %d", acc.WonderPicksDone))

	// Resources
	shinedustLabel := widget.NewLabel(fmt.Sprintf("💎 %d", acc.Shinedust))
	hourglassesLabel := widget.NewLabel(fmt.Sprintf("⏳ %d", acc.Hourglasses))
	pokegoldLabel := widget.NewLabel(fmt.Sprintf("🪙 %d", acc.Pokegold))

	// Status
	statusText := "Active"
	if acc.IsBanned {
		statusText = "Banned"
	} else if !acc.IsActive {
		statusText = "Inactive"
	}
	statusLabel := widget.NewLabel(fmt.Sprintf("Status: %s", statusText))

	// Last used
	lastUsedText := "Never"
	if acc.LastUsedAt != nil {
		lastUsedText = acc.LastUsedAt.Format("2006-01-02 15:04")
	}
	lastUsedLabel := widget.NewLabel(fmt.Sprintf("Last Used: %s", lastUsedText))

	// View details button
	detailsBtn := widget.NewButton("Details", func() {
		t.showAccountDetails(acc)
	})

	// Card content
	cardContent := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		levelLabel,
		packsLabel,
		picksLabel,
		widget.NewSeparator(),
		container.NewHBox(shinedustLabel, hourglassesLabel, pokegoldLabel),
		statusLabel,
		lastUsedLabel,
		detailsBtn,
	)

	// Card with border
	return container.NewPadded(
		container.NewBorder(nil, nil, nil, nil, cardContent),
	)
}

// buildListView creates a table of accounts
func (t *DatabaseAccountsTab) buildListView(accounts []*database.Account) fyne.CanvasObject {
	// Create table
	table := widget.NewTable(
		func() (int, int) {
			return len(accounts) + 1, 8 // +1 for header, 8 columns
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)

			// Header row
			if id.Row == 0 {
				headers := []string{"ID", "Username", "Level", "Packs", "Wonder Picks", "Shinedust", "Status", "Last Used"}
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}

			// Data rows
			acc := accounts[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(fmt.Sprintf("%d", acc.ID))
			case 1:
				if acc.Username != nil && *acc.Username != "" {
					label.SetText(*acc.Username)
				} else {
					label.SetText(acc.DeviceAccount[:min(20, len(acc.DeviceAccount))])
				}
			case 2:
				label.SetText(fmt.Sprintf("%d", acc.AccountLevel))
			case 3:
				label.SetText(fmt.Sprintf("%d", acc.PacksOpened))
			case 4:
				label.SetText(fmt.Sprintf("%d", acc.WonderPicksDone))
			case 5:
				label.SetText(fmt.Sprintf("%d", acc.Shinedust))
			case 6:
				if acc.IsBanned {
					label.SetText("Banned")
				} else if !acc.IsActive {
					label.SetText("Inactive")
				} else {
					label.SetText("Active")
				}
			case 7:
				if acc.LastUsedAt != nil {
					label.SetText(acc.LastUsedAt.Format("01/02 15:04"))
				} else {
					label.SetText("Never")
				}
			}
		},
	)

	// Set column widths
	table.SetColumnWidth(0, 50)   // ID
	table.SetColumnWidth(1, 150)  // Username
	table.SetColumnWidth(2, 60)   // Level
	table.SetColumnWidth(3, 60)   // Packs
	table.SetColumnWidth(4, 100)  // Wonder Picks
	table.SetColumnWidth(5, 80)   // Shinedust
	table.SetColumnWidth(6, 80)   // Status
	table.SetColumnWidth(7, 120)  // Last Used

	// Handle row selection for details
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 { // Skip header
			t.showAccountDetails(accounts[id.Row-1])
		}
	}

	// Return table directly - it will fill the available space
	return table
}

// showAccountDetails shows a dialog with account details
func (t *DatabaseAccountsTab) showAccountDetails(acc *database.Account) {
	// Build detailed info
	details := fmt.Sprintf(`Account ID: %d
Device Account: %s
Username: %s
Friend Code: %s

Level: %d
Packs Opened: %d
Wonder Picks Done: %d

Shinedust: %d
Hourglasses: %d
Pokegold: %d
Pack Points: %d

Created: %s
Last Used: %s
Stamina Recovery: %s

File Path: %s
Active: %t
Banned: %t
Notes: %s`,
		acc.ID,
		acc.DeviceAccount,
		stringOrEmpty(acc.Username),
		stringOrEmpty(acc.FriendCode),
		acc.AccountLevel,
		acc.PacksOpened,
		acc.WonderPicksDone,
		acc.Shinedust,
		acc.Hourglasses,
		acc.Pokegold,
		acc.PackPoints,
		acc.CreatedAt.Format("2006-01-02 15:04:05"),
		timeOrEmpty(acc.LastUsedAt),
		timeOrEmpty(acc.StaminaRecoveryTime),
		acc.FilePath,
		acc.IsActive,
		acc.IsBanned,
		stringOrEmpty(acc.Notes),
	)

	// Create dialog with scrollable content
	content := container.NewVScroll(widget.NewLabel(details))
	content.SetMinSize(fyne.NewSize(500, 400))

	dialog.ShowCustom(
		"Account Details",
		"Close",
		content,
		t.controller.window,
	)
}

// Helper functions
func stringOrEmpty(s *string) string {
	if s == nil {
		return "(none)"
	}
	return *s
}

func timeOrEmpty(t *time.Time) string {
	if t == nil {
		return "(none)"
	}
	return t.Format("2006-01-02 15:04:05")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
