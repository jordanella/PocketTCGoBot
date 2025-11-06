package gui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CardRarity represents card rarity levels
type CardRarity int

const (
	RarityCommon CardRarity = iota
	RarityUncommon
	RarityRare
	RarityUltraRare
	RaritySecret
)

func (r CardRarity) String() string {
	switch r {
	case RarityCommon:
		return "Common"
	case RarityUncommon:
		return "Uncommon"
	case RarityRare:
		return "Rare"
	case RarityUltraRare:
		return "Ultra Rare"
	case RaritySecret:
		return "Secret Rare"
	default:
		return "Unknown"
	}
}

// PackResult represents a pack opening result
type PackResult struct {
	Timestamp time.Time
	Instance  int
	PackType  string
	Cards     []CardResult
}

// CardResult represents a single card pulled
type CardResult struct {
	Name   string
	Rarity CardRarity
}

// ResultsTab displays pack opening results and statistics
type ResultsTab struct {
	controller *Controller

	// Results storage
	results   []PackResult
	resultsMu sync.RWMutex

	// Widgets
	resultsList  *widget.List
	statsLabel   *widget.Label
	clearBtn     *widget.Button
	exportBtn    *widget.Button
}

// NewResultsTab creates a new results tab
func NewResultsTab(ctrl *Controller) *ResultsTab {
	tab := &ResultsTab{
		controller: ctrl,
		results:    make([]PackResult, 0, 1000),
	}

	// Add some sample results for demonstration
	tab.AddResult(PackResult{
		Timestamp: time.Now().Add(-10 * time.Minute),
		Instance:  1,
		PackType:  "Genetic Apex",
		Cards: []CardResult{
			{Name: "Pikachu", Rarity: RarityCommon},
			{Name: "Charmander", Rarity: RarityCommon},
			{Name: "Mewtwo EX", Rarity: RarityUltraRare},
			{Name: "Bulbasaur", Rarity: RarityUncommon},
			{Name: "Squirtle", Rarity: RarityCommon},
		},
	})

	return tab
}

// Build constructs the results viewer UI
func (r *ResultsTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Pack Opening Results", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Statistics label
	r.statsLabel = widget.NewLabel(r.generateStats())

	// Clear button
	r.clearBtn = widget.NewButton("Clear Results", func() {
		r.ClearResults()
	})

	// Export button
	r.exportBtn = widget.NewButton("Export to CSV", func() {
		r.exportResults()
	})

	buttons := container.NewHBox(r.clearBtn, r.exportBtn)

	// Results list
	r.resultsList = widget.NewList(
		func() int {
			r.resultsMu.RLock()
			defer r.resultsMu.RUnlock()
			return len(r.results)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("Timestamp - Pack Type"),
				widget.NewLabel("Cards pulled"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			r.resultsMu.RLock()
			defer r.resultsMu.RUnlock()

			if id >= len(r.results) {
				return
			}

			result := r.results[id]
			box := item.(*fyne.Container)

			// Header
			headerLabel := box.Objects[0].(*widget.Label)
			headerLabel.SetText(fmt.Sprintf("%s - %s (Instance %d)",
				result.Timestamp.Format("15:04:05"),
				result.PackType,
				result.Instance,
			))

			// Cards
			cardsLabel := box.Objects[1].(*widget.Label)
			cardsText := ""
			for i, card := range result.Cards {
				if i > 0 {
					cardsText += ", "
				}
				cardsText += fmt.Sprintf("%s (%s)", card.Name, card.Rarity.String())
			}
			cardsLabel.SetText(cardsText)
		},
	)

	// Layout
	content := container.NewBorder(
		container.NewVBox(header, r.statsLabel, buttons),
		nil,
		nil,
		nil,
		r.resultsList,
	)

	return content
}

// AddResult adds a new pack opening result
func (r *ResultsTab) AddResult(result PackResult) {
	r.resultsMu.Lock()
	defer r.resultsMu.Unlock()

	r.results = append(r.results, result)

	// Update UI
	if r.resultsList != nil {
		r.resultsList.Refresh()
	}
	if r.statsLabel != nil {
		r.statsLabel.SetText(r.generateStats())
	}
}

// ClearResults removes all results
func (r *ResultsTab) ClearResults() {
	r.resultsMu.Lock()
	defer r.resultsMu.Unlock()

	r.results = make([]PackResult, 0, 1000)

	if r.resultsList != nil {
		r.resultsList.Refresh()
	}
	if r.statsLabel != nil {
		r.statsLabel.SetText(r.generateStats())
	}
}

// generateStats creates statistics summary
func (r *ResultsTab) generateStats() string {
	r.resultsMu.RLock()
	defer r.resultsMu.RUnlock()

	if len(r.results) == 0 {
		return "No results yet"
	}

	totalPacks := len(r.results)
	totalCards := 0
	rarityCounts := make(map[CardRarity]int)

	for _, result := range r.results {
		totalCards += len(result.Cards)
		for _, card := range result.Cards {
			rarityCounts[card.Rarity]++
		}
	}

	stats := fmt.Sprintf("Total Packs: %d | Total Cards: %d\n", totalPacks, totalCards)
	stats += fmt.Sprintf("Common: %d | Uncommon: %d | Rare: %d | Ultra Rare: %d | Secret: %d",
		rarityCounts[RarityCommon],
		rarityCounts[RarityUncommon],
		rarityCounts[RarityRare],
		rarityCounts[RarityUltraRare],
		rarityCounts[RaritySecret],
	)

	return stats
}

// exportResults exports results to CSV
func (r *ResultsTab) exportResults() {
	// TODO: Implement CSV export with file dialog
	// For now, just log
	r.controller.logTab.AddLog(LogLevelInfo, 0, "Export functionality coming soon")
}

// GetResults returns all results (for export)
func (r *ResultsTab) GetResults() []PackResult {
	r.resultsMu.RLock()
	defer r.resultsMu.RUnlock()

	results := make([]PackResult, len(r.results))
	copy(results, r.results)
	return results
}
