package components

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// GroupSectionCardCallbacks defines callback functions for group section card actions
type GroupSectionCardCallbacks struct {
	OnAddInstance func(groupName string)
}

// GroupSectionCardV2 represents a group section card (parent card containing instances)
type GroupSectionCardV2 struct {
	// Data and state
	groupName       string
	orchestrationID string
	description     string
	startedAt       time.Time
	poolName        string
	poolRemaining   int
	poolTotal       int

	// Callbacks
	callbacks GroupSectionCardCallbacks

	// UI elements that need dynamic updates
	container          *fyne.Container
	descriptionText    *canvas.Text
	startedText        *canvas.Text
	poolText           *canvas.Text
	instancesContainer *fyne.Container
}

// NewGroupSectionCardV2 creates a new group section card
func NewGroupSectionCardV2(
	groupName string,
	orchestrationID string,
	callbacks GroupSectionCardCallbacks,
) *GroupSectionCardV2 {
	card := &GroupSectionCardV2{
		groupName:          groupName,
		orchestrationID:    orchestrationID,
		callbacks:          callbacks,
		instancesContainer: container.NewVBox(),
	}

	card.container = card.build()
	card.UpdateFromGroup()

	return card
}

// build creates the card UI layout
func (c *GroupSectionCardV2) build() *fyne.Container {
	// === HEADER ROW ===
	// Pattern: "Group Name <orchestrationID>                    [ + Instance ]"
	groupNameLabel := Subheading(c.groupName)
	orchestrationIDLabel := Caption(fmt.Sprintf("<%s>", c.orchestrationID[:8]))
	headerLabels := InlineLabels(" ", groupNameLabel, orchestrationIDLabel)

	addInstanceBtn := SecondaryButton("+ Instance", func() {
		if c.callbacks.OnAddInstance != nil {
			c.callbacks.OnAddInstance(c.groupName)
		}
	})

	headerRow := LabelButtonsRow(headerLabels, addInstanceBtn)

	// === DESCRIPTION ===
	c.descriptionText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.descriptionText.TextSize = 14

	// === INFO ROW ===
	c.startedText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.startedText.TextSize = 14

	c.poolText = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	c.poolText.TextSize = 14

	startedInfo := container.NewHBox(BoldText("Started:"), c.startedText)
	poolInfo := container.NewHBox(BoldText("Account Pool:"), c.poolText)
	infoRow := InlineInfoRow(startedInfo, poolInfo)

	// === ASSEMBLE CARD CONTENT ===
	content := container.NewVBox(
		headerRow,
		c.descriptionText,
		infoRow,
	)

	cardContainer := container.NewVBox(
		Card(content),
		c.instancesContainer,
	)

	return cardContainer
}

// UpdateFromGroup refreshes the card from group data
func (c *GroupSectionCardV2) UpdateFromGroup() {
	fyne.Do(func() {
		// Update description
		c.descriptionText.Text = c.description
		c.descriptionText.Refresh()

		// Update started time
		if !c.startedAt.IsZero() {
			elapsed := time.Since(c.startedAt)
			c.startedText.Text = formatDurationCompact(elapsed) + " ago"
		} else {
			c.startedText.Text = "Not started"
		}
		c.startedText.Refresh()

		// Update pool info
		if c.poolName != "" {
			c.poolText.Text = fmt.Sprintf("%s (%d/%d)", c.poolName, c.poolRemaining, c.poolTotal)
		} else {
			c.poolText.Text = "None"
		}
		c.poolText.Refresh()
	})
}

// SetDescription sets the group description
func (c *GroupSectionCardV2) SetDescription(description string) {
	c.description = description
	c.UpdateFromGroup()
}

// SetStartedAt sets the start time
func (c *GroupSectionCardV2) SetStartedAt(startedAt time.Time) {
	c.startedAt = startedAt
	c.UpdateFromGroup()
}

// SetPoolInfo sets the account pool information
func (c *GroupSectionCardV2) SetPoolInfo(poolName string, remaining, total int) {
	c.poolName = poolName
	c.poolRemaining = remaining
	c.poolTotal = total
	c.UpdateFromGroup()
}

// AddInstanceCard adds an instance card to this group section
func (c *GroupSectionCardV2) AddInstanceCard(instanceCard *EmulatorInstanceCardV2) {
	c.instancesContainer.Add(instanceCard.GetContainer())
}

// RemoveInstanceCard removes an instance card from this group section
func (c *GroupSectionCardV2) RemoveInstanceCard(instanceCard *EmulatorInstanceCardV2) {
	c.instancesContainer.Remove(instanceCard.GetContainer())
}

// ClearInstanceCards removes all instance cards
func (c *GroupSectionCardV2) ClearInstanceCards() {
	c.instancesContainer.Objects = nil
	fyne.Do(func() {
		c.instancesContainer.Refresh()
	})
}

// GetContainer returns the Fyne container for embedding in layouts
func (c *GroupSectionCardV2) GetContainer() *fyne.Container {
	return c.container
}

// GetGroupName returns the group name
func (c *GroupSectionCardV2) GetGroupName() string {
	return c.groupName
}
