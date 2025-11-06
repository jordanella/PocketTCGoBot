package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/database"
)

// DatabasePacksTab displays pack opening results
type DatabasePacksTab struct {
	controller *Controller
	db         *database.DB

	// View mode
	viewMode    string // "cards" or "list"
	viewModeBtn *widget.Button

	// Filters
	filterAccount  *widget.Entry
	filterPackType *widget.Select

	// Content containers
	contentArea *fyne.Container
}

// NewDatabasePacksTab creates a new database packs tab
func NewDatabasePacksTab(ctrl *Controller, db *database.DB) *DatabasePacksTab {
	return &DatabasePacksTab{
		controller: ctrl,
		db:         db,
		viewMode:   "cards",
	}
}

// Build constructs the UI
func (t *DatabasePacksTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Database - Pack Results", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// View toggle button
	t.viewModeBtn = widget.NewButton("Switch to List View", func() {
		t.toggleViewMode()
	})

	// Filters
	t.filterAccount = widget.NewEntry()
	t.filterAccount.SetPlaceHolder("Account ID")

	t.filterPackType = widget.NewSelect([]string{
		"All",
		"standard",
		"premium",
		"promo",
	}, func(string) {
		t.refresh()
	})
	t.filterPackType.SetSelected("All")

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", func() {
		t.refresh()
	})

	// Clear filters button
	clearBtn := widget.NewButton("Clear Filters", func() {
		t.filterAccount.SetText("")
		t.filterPackType.SetSelected("All")
		t.refresh()
	})

	// Toolbar
	toolbar := container.NewHBox(
		t.viewModeBtn,
		widget.NewLabel("Account ID:"),
		t.filterAccount,
		widget.NewLabel("Pack Type:"),
		t.filterPackType,
		refreshBtn,
		clearBtn,
	)

	// Content area
	t.contentArea = container.NewStack()
	t.refresh()

	// Scrollable content
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
func (t *DatabasePacksTab) toggleViewMode() {
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
func (t *DatabasePacksTab) refresh() {
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

	// Get pack results based on filters
	packs, err := t.getFilteredPacks()
	if err != nil {
		if t.controller.window != nil {
			dialog.ShowError(err, t.controller.window)
		}
		return
	}

	if len(packs) == 0 {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("No pack results found"),
		}
		t.contentArea.Refresh()
		return
	}

	// Build view based on mode
	if t.viewMode == "cards" {
		t.contentArea.Objects = []fyne.CanvasObject{
			t.buildCardsView(packs),
		}
	} else {
		t.contentArea.Objects = []fyne.CanvasObject{
			t.buildListView(packs),
		}
	}

	t.contentArea.Refresh()
}

// getFilteredPacks gets pack results based on current filters
func (t *DatabasePacksTab) getFilteredPacks() ([]*database.PackResult, error) {
	query := `
		SELECT id, account_id, activity_log_id, pack_type, pack_name,
		       is_god_pack, card_count, rarity_breakdown, pack_points_earned, opened_at
		FROM pack_results
		ORDER BY opened_at DESC
		LIMIT 500
	`

	rows, err := t.db.Conn().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packs []*database.PackResult
	for rows.Next() {
		pack := &database.PackResult{}
		err := rows.Scan(
			&pack.ID,
			&pack.AccountID,
			&pack.ActivityLogID,
			&pack.PackType,
			&pack.PackName,
			&pack.IsGodPack,
			&pack.CardCount,
			&pack.RarityBreakdown,
			&pack.PackPointsEarned,
			&pack.OpenedAt,
		)
		if err != nil {
			return nil, err
		}

		// Apply client-side filters
		if !t.matchesPackFilters(pack) {
			continue
		}

		packs = append(packs, pack)
	}

	return packs, nil
}

// matchesPackFilters checks if pack matches current filters
func (t *DatabasePacksTab) matchesPackFilters(pack *database.PackResult) bool {
	// Account ID filter
	if t.filterAccount.Text != "" {
		accountIDStr := fmt.Sprintf("%d", pack.AccountID)
		if accountIDStr != t.filterAccount.Text {
			return false
		}
	}

	// Pack type filter
	if t.filterPackType.Selected != "All" {
		if pack.PackType != t.filterPackType.Selected {
			return false
		}
	}

	return true
}

// buildCardsView creates a grid of pack cards
func (t *DatabasePacksTab) buildCardsView(packs []*database.PackResult) fyne.CanvasObject {
	cards := container.NewGridWithColumns(3)

	for _, pack := range packs {
		card := t.createPackCard(pack)
		cards.Add(card)
	}

	return cards
}

// createPackCard creates a card widget for a pack result
func (t *DatabasePacksTab) createPackCard(pack *database.PackResult) fyne.CanvasObject {
	// Title
	packName := pack.PackType
	if pack.PackName != nil && *pack.PackName != "" {
		packName = *pack.PackName
	}

	titleLabel := widget.NewLabelWithStyle(packName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// God pack indicator
	godPackText := ""
	if pack.IsGodPack {
		godPackText = " ⭐ GOD PACK!"
	}
	if godPackText != "" {
		godPackLabel := widget.NewLabelWithStyle(godPackText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		titleLabel = godPackLabel
	}

	// Account info
	accountLabel := widget.NewLabel(fmt.Sprintf("Account: %d", pack.AccountID))

	// Timestamp
	timeLabel := widget.NewLabel(fmt.Sprintf("Opened: %s", pack.OpenedAt.Format("01/02 15:04")))

	// Cards pulled
	cardsLabel := widget.NewLabel(fmt.Sprintf("Cards: %d", pack.CardCount))

	// Pack points
	pointsLabel := widget.NewLabel(fmt.Sprintf("Pack Points: %d", pack.PackPointsEarned))

	// Rarity breakdown
	rarityText := "(none)"
	if pack.RarityBreakdown != nil && *pack.RarityBreakdown != "" {
		rarityText = *pack.RarityBreakdown
	}
	rarityLabel := widget.NewLabel(fmt.Sprintf("Rarities: %s", rarityText))

	// View details button
	detailsBtn := widget.NewButton("View Cards", func() {
		t.showPackDetails(pack)
	})

	// Card content
	cardContent := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		accountLabel,
		timeLabel,
		widget.NewSeparator(),
		cardsLabel,
		pointsLabel,
		rarityLabel,
		detailsBtn,
	)

	// Card with border
	return container.NewPadded(
		container.NewBorder(nil, nil, nil, nil, cardContent),
	)
}

// buildListView creates a table of pack results
func (t *DatabasePacksTab) buildListView(packs []*database.PackResult) fyne.CanvasObject {
	// Create table
	table := widget.NewTable(
		func() (int, int) {
			return len(packs) + 1, 7 // +1 for header, 7 columns
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)

			// Header row
			if id.Row == 0 {
				headers := []string{"ID", "Account", "Pack", "Type", "Opened", "Cards", "Points"}
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}

			// Data rows
			pack := packs[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(fmt.Sprintf("%d", pack.ID))
			case 1:
				label.SetText(fmt.Sprintf("%d", pack.AccountID))
			case 2:
				if pack.PackName != nil && *pack.PackName != "" {
					label.SetText(*pack.PackName)
				} else {
					label.SetText(pack.PackType)
				}
			case 3:
				godPack := ""
				if pack.IsGodPack {
					godPack = "⭐"
				}
				label.SetText(pack.PackType + godPack)
			case 4:
				label.SetText(pack.OpenedAt.Format("01/02 15:04"))
			case 5:
				label.SetText(fmt.Sprintf("%d", pack.CardCount))
			case 6:
				label.SetText(fmt.Sprintf("%d", pack.PackPointsEarned))
			}
		},
	)

	// Set column widths
	table.SetColumnWidth(0, 50)  // ID
	table.SetColumnWidth(1, 80)  // Account
	table.SetColumnWidth(2, 150) // Pack
	table.SetColumnWidth(3, 100) // Type
	table.SetColumnWidth(4, 120) // Opened
	table.SetColumnWidth(5, 60)  // Cards
	table.SetColumnWidth(6, 80)  // Points

	// Handle row selection for details
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 { // Skip header
			t.showPackDetails(packs[id.Row-1])
		}
	}

	return table
}

// showPackDetails shows a dialog with pack details including cards pulled
func (t *DatabasePacksTab) showPackDetails(pack *database.PackResult) {
	// Get cards pulled in this pack
	cards, err := t.getCardsInPack(pack.ID)
	if err != nil {
		if t.controller.window != nil {
			dialog.ShowError(err, t.controller.window)
		}
		return
	}

	// Format pack name
	packName := pack.PackType
	if pack.PackName != nil && *pack.PackName != "" {
		packName = *pack.PackName
	}

	// Format rarity breakdown
	rarityText := "(none)"
	if pack.RarityBreakdown != nil && *pack.RarityBreakdown != "" {
		rarityText = *pack.RarityBreakdown
	}

	// Format god pack
	godPackText := "No"
	if pack.IsGodPack {
		godPackText = "YES! ⭐"
	}

	// Build pack info
	packInfo := fmt.Sprintf(`Pack Opening Details

Account ID: %d
Pack Name: %s
Pack Type: %s
God Pack: %s
Opened: %s

Cards Pulled: %d
Pack Points Earned: %d
Rarity Breakdown: %s

Cards in Pack:
`,
		pack.AccountID,
		packName,
		pack.PackType,
		godPackText,
		pack.OpenedAt.Format("2006-01-02 15:04:05"),
		pack.CardCount,
		pack.PackPointsEarned,
		rarityText,
	)

	// Add cards to info
	if len(cards) == 0 {
		packInfo += "\n(No individual card records found)"
	} else {
		packInfo += "\n"
		for _, card := range cards {
			cardName := "(unknown)"
			if card.CardName != nil && *card.CardName != "" {
				cardName = *card.CardName
			}

			// Card type/full art indicators
			indicators := ""
			if card.IsFullArt {
				indicators += " [Full Art]"
			}
			if card.IsEx {
				indicators += " [EX]"
			}

			packInfo += fmt.Sprintf("  - %s [%s]%s\n",
				cardName,
				card.Rarity,
				indicators,
			)
		}
	}

	// Create dialog with scrollable content
	content := container.NewVScroll(widget.NewLabel(packInfo))
	content.SetMinSize(fyne.NewSize(500, 400))

	dialog.ShowCustom(
		"Pack Details",
		"Close",
		content,
		t.controller.window,
	)
}

// getCardsInPack gets all cards pulled in a specific pack
func (t *DatabasePacksTab) getCardsInPack(packID int) ([]*database.CardPulled, error) {
	query := `
		SELECT id, pack_result_id, account_id, card_id, card_name, card_number,
		       rarity, card_type, is_full_art, is_ex, detection_confidence, detected_at
		FROM cards_pulled
		WHERE pack_result_id = ?
		ORDER BY rarity DESC, card_name
	`

	rows, err := t.db.Conn().Query(query, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []*database.CardPulled
	for rows.Next() {
		card := &database.CardPulled{}
		err := rows.Scan(
			&card.ID,
			&card.PackResultID,
			&card.AccountID,
			&card.CardID,
			&card.CardName,
			&card.CardNumber,
			&card.Rarity,
			&card.CardType,
			&card.IsFullArt,
			&card.IsEx,
			&card.DetectionConfidence,
			&card.DetectedAt,
		)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}

	return cards, nil
}
