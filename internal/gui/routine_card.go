package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/actions"
)

// RoutineCard represents a clickable card for displaying routine metadata
type RoutineCard struct {
	widget.BaseWidget
	filename    string
	metadata    *actions.RoutineMetadata
	isValid     bool
	validationError error
	isExpanded  bool
	onTapped    func()

	// Cached display elements
	displayName *widget.Label
	fileLabel   *widget.Label
	statusIcon  *widget.Label
	description *widget.RichText
	tagBox      *fyne.Container
	expandedView *fyne.Container
}

// NewRoutineCard creates a new routine card widget
func NewRoutineCard(filename string, metadata *actions.RoutineMetadata, isValid bool, validationError error, onTapped func()) *RoutineCard {
	card := &RoutineCard{
		filename:        filename,
		metadata:        metadata,
		isValid:         isValid,
		validationError: validationError,
		onTapped:        onTapped,
	}
	card.ExtendBaseWidget(card)
	return card
}

// CreateRenderer creates the renderer for the routine card
func (c *RoutineCard) CreateRenderer() fyne.WidgetRenderer {
	// Display name (bold, larger)
	c.displayName = widget.NewLabelWithStyle(
		c.metadata.DisplayName,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	// Filename (faded)
	c.fileLabel = widget.NewLabel(c.filename)
	c.fileLabel.TextStyle = fyne.TextStyle{Italic: true}
	c.fileLabel.Importance = widget.LowImportance

	// Status indicator
	if c.isValid {
		c.statusIcon = widget.NewLabel("✓")
		c.statusIcon.Importance = widget.SuccessImportance
	} else {
		c.statusIcon = widget.NewLabel("⚠️")
		c.statusIcon.Importance = widget.DangerImportance
	}

	// Header row: Display name + filename + status
	headerRow := container.NewBorder(
		nil, nil,
		c.displayName,
		c.statusIcon,
		c.fileLabel,
	)

	// Description (truncated to 2-3 lines)
	descText := c.metadata.Description
	if descText == "" {
		descText = "No description provided"
	}

	// Truncate description if too long
	maxDescLength := 120
	if len(descText) > maxDescLength {
		descText = descText[:maxDescLength] + "..."
	}

	c.description = widget.NewRichTextFromMarkdown(descText)
	c.description.Wrapping = fyne.TextWrapWord

	// Tags as badges
	c.tagBox = c.createTagBadges()

	// Validation error display (if any)
	var errorDisplay fyne.CanvasObject
	if !c.isValid && c.validationError != nil {
		errorText := widget.NewLabel(fmt.Sprintf("Error: %s", c.validationError.Error()))
		errorText.Wrapping = fyne.TextWrapWord
		errorText.Importance = widget.DangerImportance
		errorDisplay = container.NewVBox(
			widget.NewSeparator(),
			errorText,
		)
	}

	// Card content
	cardContent := container.NewVBox(
		headerRow,
		c.description,
	)

	if len(c.metadata.Tags) > 0 {
		cardContent.Add(c.tagBox)
	}

	if errorDisplay != nil {
		cardContent.Add(errorDisplay)
	}

	// Card background
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))

	// Card border
	border := canvas.NewRectangle(theme.Color(theme.ColorNameSeparator))

	objects := []fyne.CanvasObject{
		border,
		bg,
		container.NewPadded(cardContent),
	}

	return &routineCardRenderer{
		card:    c,
		objects: objects,
		bg:      bg,
		border:  border,
		content: cardContent,
	}
}

// createTagBadges creates badge widgets for each tag
func (c *RoutineCard) createTagBadges() *fyne.Container {
	badges := []fyne.CanvasObject{}

	for _, tag := range c.metadata.Tags {
		badge := c.createBadge(tag)
		badges = append(badges, badge)
	}

	return container.New(layout.NewGridWrapLayout(fyne.NewSize(80, 25)), badges...)
}

// createBadge creates a single badge widget
func (c *RoutineCard) createBadge(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)

	// Color based on tag type
	var bgColor fyne.ThemeColorName
	switch strings.ToLower(text) {
	case "sentry":
		bgColor = theme.ColorNameError
	case "navigation":
		bgColor = theme.ColorNamePrimary
	case "combat":
		bgColor = theme.ColorNameWarning
	case "example":
		bgColor = theme.ColorNameDisabled
	default:
		bgColor = theme.ColorNameInputBackground
	}

	bg := canvas.NewRectangle(theme.Color(bgColor))
	bg.CornerRadius = 4

	return container.NewStack(
		bg,
		container.NewPadded(label),
	)
}

// Tapped handles tap events
func (c *RoutineCard) Tapped(_ *fyne.PointEvent) {
	if c.onTapped != nil {
		c.onTapped()
	}
}

// routineCardRenderer renders the routine card
type routineCardRenderer struct {
	card    *RoutineCard
	objects []fyne.CanvasObject
	bg      *canvas.Rectangle
	border  *canvas.Rectangle
	content *fyne.Container
}

func (r *routineCardRenderer) Layout(size fyne.Size) {
	// Border (slightly larger than background for border effect)
	r.border.Resize(size)
	r.border.Move(fyne.NewPos(0, 0))

	// Background (1px smaller on all sides)
	bgSize := fyne.NewSize(size.Width-2, size.Height-2)
	r.bg.Resize(bgSize)
	r.bg.Move(fyne.NewPos(1, 1))

	// Content
	r.objects[2].Resize(size)
	r.objects[2].Move(fyne.NewPos(0, 0))
}

func (r *routineCardRenderer) MinSize() fyne.Size {
	return r.objects[2].MinSize().Add(fyne.NewSize(20, 20))
}

func (r *routineCardRenderer) Refresh() {
	r.bg.FillColor = theme.Color(theme.ColorNameBackground)
	r.border.FillColor = theme.Color(theme.ColorNameSeparator)
	r.bg.Refresh()
	r.border.Refresh()
	canvas.Refresh(r.card)
}

func (r *routineCardRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *routineCardRenderer) Destroy() {}
