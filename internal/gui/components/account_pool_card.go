package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// AccountPoolCardCallbacks defines callback functions for pool card actions
type AccountPoolCardCallbacks struct {
	OnSelect func(poolName string)
}

// AccountPoolCard represents a pool card in the list view
type AccountPoolCard struct {
	// Data
	poolName     string
	poolType     string
	accountCount int
	lastUpdated  string
	description  string
	isSelected   bool

	// Callbacks
	callbacks AccountPoolCardCallbacks

	// UI elements
	container     *fyne.Container
	nameText      *canvas.Text
	typeLabel     *canvas.Text
	countText     *canvas.Text
	updatedText   *canvas.Text
	descText      *canvas.Text
	cardContainer *fyne.Container
}

// NewAccountPoolCard creates a new account pool card
func NewAccountPoolCard(
	poolName string,
	poolType string,
	accountCount int,
	lastUpdated string,
	description string,
	callbacks AccountPoolCardCallbacks,
) *AccountPoolCard {
	card := &AccountPoolCard{
		poolName:     poolName,
		poolType:     poolType,
		accountCount: accountCount,
		lastUpdated:  lastUpdated,
		description:  description,
		isSelected:   false,
		callbacks:    callbacks,
	}

	card.container = card.build()
	return card
}

// build creates the card UI layout
func (c *AccountPoolCard) build() *fyne.Container {
	// === HEADER ROW ===
	// Pattern: "Pool Name <type>"
	c.nameText = canvas.NewText(c.poolName, theme.Color(theme.ColorNameForeground))
	c.nameText.TextSize = 16
	c.nameText.TextStyle = fyne.TextStyle{Bold: true}

	c.typeLabel = canvas.NewText(fmt.Sprintf("<%s>", c.poolType), theme.Color(theme.ColorNameForeground))
	c.typeLabel.TextSize = 12

	headerRow := container.NewHBox(
		c.nameText,
		c.typeLabel,
	)

	// === INFO ROW ===
	// Pattern: "<accounts>     <updated>"
	c.countText = canvas.NewText(fmt.Sprintf("%d accounts", c.accountCount), theme.Color(theme.ColorNameForeground))
	c.countText.TextSize = 12

	c.updatedText = canvas.NewText(c.lastUpdated, theme.Color(theme.ColorNameForeground))
	c.updatedText.TextSize = 12

	infoRow := container.NewHBox(
		c.countText,
		canvas.NewText("     ", theme.Color(theme.ColorNameForeground)), // Spacer
		c.updatedText,
	)

	// === DESCRIPTION ROW ===
	c.descText = canvas.NewText(c.description, theme.Color(theme.ColorNameForeground))
	c.descText.TextSize = 12

	// Truncate description to 50 characters
	if len(c.description) > 50 {
		c.descText.Text = c.description[:47] + "..."
	}

	// === ASSEMBLE CARD ===
	content := container.NewVBox(
		headerRow,
		infoRow,
		c.descText,
	)

	// Wrap in card
	c.cardContainer = Card(content)

	// Wrap in button for click handling
	btn := widget.NewButton("", func() {
		c.HandleTap()
	})
	btn.Importance = widget.LowImportance

	// Stack button behind card for click handling
	return container.NewStack(btn, c.cardContainer)
}

// SetSelected sets the selected state
func (c *AccountPoolCard) SetSelected(selected bool) {
	c.isSelected = selected

	// Update visual indication - make name text blue when selected
	if selected {
		c.nameText.Color = theme.Color(theme.ColorNamePrimary)
	} else {
		c.nameText.Color = theme.Color(theme.ColorNameForeground)
	}

	c.nameText.Refresh()
	c.container.Refresh()
}

// UpdateData updates the card data
func (c *AccountPoolCard) UpdateData(accountCount int, lastUpdated string, description string) {
	c.accountCount = accountCount
	c.lastUpdated = lastUpdated
	c.description = description

	c.countText.Text = fmt.Sprintf("%d accounts", accountCount)
	c.updatedText.Text = lastUpdated
	c.descText.Text = description

	if len(description) > 50 {
		c.descText.Text = description[:47] + "..."
	}

	c.countText.Refresh()
	c.updatedText.Refresh()
	c.descText.Refresh()
}

// GetContainer returns the Fyne container for embedding in layouts
func (c *AccountPoolCard) GetContainer() *fyne.Container {
	return c.container
}

// GetPoolName returns the pool name
func (c *AccountPoolCard) GetPoolName() string {
	return c.poolName
}

// HandleTap handles tap events on the card
func (c *AccountPoolCard) HandleTap() {
	if c.callbacks.OnSelect != nil {
		c.callbacks.OnSelect(c.poolName)
	}
}
