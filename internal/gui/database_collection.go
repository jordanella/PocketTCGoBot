package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/database"
)

// DatabaseCollectionTab displays card collections
type DatabaseCollectionTab struct {
	controller *Controller
	db         *database.DB

	// Filters
	filterAccount *widget.Entry
	filterRarity  *widget.Select
	sortBy        *widget.Select

	// Content containers
	contentArea *fyne.Container
}

// NewDatabaseCollectionTab creates a new database collection tab
func NewDatabaseCollectionTab(ctrl *Controller, db *database.DB) *DatabaseCollectionTab {
	return &DatabaseCollectionTab{
		controller: ctrl,
		db:         db,
	}
}

// Build constructs the UI
func (t *DatabaseCollectionTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Database - Card Collections", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Filters
	t.filterAccount = widget.NewEntry()
	t.filterAccount.SetPlaceHolder("Account ID")

	t.filterRarity = widget.NewSelect([]string{
		"All",
		"Common",
		"Uncommon",
		"Rare",
		"Double Rare",
		"Star",
		"Crown",
		"Promo",
	}, func(string) {
		t.refresh()
	})
	t.filterRarity.SetSelected("All")

	t.sortBy = widget.NewSelect([]string{
		"Card Name",
		"Rarity",
		"Quantity",
		"First Pulled",
		"Last Pulled",
	}, func(string) {
		t.refresh()
	})
	t.sortBy.SetSelected("Card Name")

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", func() {
		t.refresh()
	})

	// Clear filters button
	clearBtn := widget.NewButton("Clear Filters", func() {
		t.filterAccount.SetText("")
		t.filterRarity.SetSelected("All")
		t.sortBy.SetSelected("Card Name")
		t.refresh()
	})

	// Collection stats button
	statsBtn := widget.NewButton("Collection Stats", func() {
		t.showCollectionStats()
	})

	// Toolbar
	toolbar := container.NewHBox(
		widget.NewLabel("Account ID:"),
		t.filterAccount,
		widget.NewLabel("Rarity:"),
		t.filterRarity,
		widget.NewLabel("Sort By:"),
		t.sortBy,
		refreshBtn,
		clearBtn,
		statsBtn,
	)

	// Content area
	t.contentArea = container.NewVBox()
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

// refresh reloads the data
func (t *DatabaseCollectionTab) refresh() {
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

	// Check if account filter is set
	if t.filterAccount == nil || t.filterAccount.Text == "" {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("Please enter an Account ID to view collection"),
		}
		t.contentArea.Refresh()
		return
	}

	// Get collection for account
	accountID := 0
	_, err := fmt.Sscanf(t.filterAccount.Text, "%d", &accountID)
	if err != nil {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("Invalid Account ID"),
		}
		t.contentArea.Refresh()
		return
	}

	collection, err := t.getFilteredCollection(accountID)
	if err != nil {
		if t.controller.window != nil { dialog.ShowError(err, t.controller.window) }
		return
	}

	if len(collection) == 0 {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("No cards in collection"),
		}
		t.contentArea.Refresh()
		return
	}

	// Build grid view
	t.contentArea.Objects = []fyne.CanvasObject{
		t.buildGridView(collection),
	}

	t.contentArea.Refresh()
}

// getFilteredCollection gets collection based on current filters
func (t *DatabaseCollectionTab) getFilteredCollection(accountID int) ([]*database.AccountCollection, error) {
	collection, err := t.db.GetAccountCollection(accountID)
	if err != nil {
		return nil, err
	}

	// Apply rarity filter
	var filtered []*database.AccountCollection
	for _, card := range collection {
		if t.filterRarity.Selected != "All" {
			if card.Rarity != t.filterRarity.Selected {
				continue
			}
		}
		filtered = append(filtered, card)
	}

	// Apply sorting
	// Note: In production, you'd implement proper sorting here
	// For now, the default sort from database is by card_name

	return filtered, nil
}

// buildGridView creates a grid of collection cards
func (t *DatabaseCollectionTab) buildGridView(collection []*database.AccountCollection) fyne.CanvasObject {
	cards := container.NewGridWithColumns(4)

	for _, card := range collection {
		cardWidget := t.createCollectionCard(card)
		cards.Add(cardWidget)
	}

	return cards
}

// createCollectionCard creates a card widget for a collection entry
func (t *DatabaseCollectionTab) createCollectionCard(card *database.AccountCollection) fyne.CanvasObject {
	// Card name
	nameLabel := widget.NewLabelWithStyle(card.CardName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Rarity with emoji indicator
	rarityEmoji := t.getRarityEmoji(card.Rarity)
	rarityLabel := widget.NewLabel(fmt.Sprintf("%s %s", rarityEmoji, card.Rarity))

	// Quantity
	quantityLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("x%d", card.Quantity),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	// First obtained
	firstObtainedLabel := widget.NewLabel(fmt.Sprintf("First: %s", card.FirstObtainedAt.Format("01/02/06")))

	// Last obtained
	lastObtainedLabel := widget.NewLabel(fmt.Sprintf("Last: %s", card.LastObtainedAt.Format("01/02/06")))

	// Card content
	cardContent := container.NewVBox(
		nameLabel,
		rarityLabel,
		widget.NewSeparator(),
		quantityLabel,
		widget.NewSeparator(),
		firstObtainedLabel,
		lastObtainedLabel,
	)

	// Card with padding and border
	return container.NewPadded(
		container.NewBorder(nil, nil, nil, nil, cardContent),
	)
}

// getRarityEmoji returns an emoji for the rarity
func (t *DatabaseCollectionTab) getRarityEmoji(rarity string) string {
	switch rarity {
	case "Common":
		return "⚪"
	case "Uncommon":
		return "🔵"
	case "Rare":
		return "💎"
	case "Double Rare":
		return "💠"
	case "Star":
		return "⭐"
	case "Crown":
		return "👑"
	case "Promo":
		return "🎁"
	default:
		return "❓"
	}
}

// showCollectionStats shows collection statistics dialog
func (t *DatabaseCollectionTab) showCollectionStats() {
	if t.filterAccount.Text == "" {
		dialog.ShowInformation("Collection Stats", "Please enter an Account ID first", t.controller.window)
		return
	}

	accountID := 0
	_, err := fmt.Sscanf(t.filterAccount.Text, "%d", &accountID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid Account ID"), t.controller.window)
		return
	}

	collection, err := t.db.GetAccountCollection(accountID)
	if err != nil {
		if t.controller.window != nil { dialog.ShowError(err, t.controller.window) }
		return
	}

	if len(collection) == 0 {
		dialog.ShowInformation("Collection Stats", "No cards in collection", t.controller.window)
		return
	}

	// Calculate stats
	totalCards := 0
	totalUnique := len(collection)
	rarityBreakdown := make(map[string]int)
	rarityCount := make(map[string]int)

	for _, card := range collection {
		totalCards += card.Quantity
		rarityBreakdown[card.Rarity] += card.Quantity
		rarityCount[card.Rarity]++
	}

	// Build stats text
	statsText := fmt.Sprintf(`Collection Statistics for Account %d

Total Cards: %d
Unique Cards: %d

Rarity Breakdown:
`, accountID, totalCards, totalUnique)

	// Order rarities for display
	rarities := []string{"Common", "Uncommon", "Rare", "Double Rare", "Star", "Crown", "Promo"}
	for _, rarity := range rarities {
		if count, ok := rarityBreakdown[rarity]; ok {
			unique := rarityCount[rarity]
			statsText += fmt.Sprintf("  %s %-15s: %4d cards (%d unique)\n",
				t.getRarityEmoji(rarity),
				rarity,
				count,
				unique,
			)
		}
	}

	// Calculate completion percentage (if we had a master card list)
	// For now, just show what we have
	statsText += fmt.Sprintf("\nCollection Value:\n")
	statsText += fmt.Sprintf("  Average cards per unique: %.1f\n", float64(totalCards)/float64(totalUnique))

	// Create dialog with scrollable content
	content := container.NewVScroll(widget.NewLabel(statsText))
	content.SetMinSize(fyne.NewSize(400, 300))

	dialog.ShowCustom(
		"Collection Statistics",
		"Close",
		content,
		t.controller.window,
	)
}
