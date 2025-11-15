package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// Card creates a rounded rectangle container with padding and optional left indent
// This provides a consistent card-like appearance for grouped content
func Card(content fyne.CanvasObject) *fyne.Container {
	return CardWithIndent(content, 0)
}

// CardWithIndent creates a card with a specific left margin for indentation
// leftIndent: the left margin in pixels (e.g., 0, 10, 20)
func CardWithIndent(content fyne.CanvasObject, leftIndent float32) *fyne.Container {
	return CardWithOptions(content, CardOptions{
		LeftIndent: leftIndent,
	})
}

// CardOptions configures the appearance of a card
type CardOptions struct {
	// LeftIndent adds left margin (in pixels) for hierarchical indentation
	LeftIndent float32

	// Padding inside the card (default: standard theme padding)
	// Set to 0 to disable padding
	PaddingOverride *float32

	// Background color (default: theme background color)
	BackgroundColor *color.Color

	// Border radius for rounded corners (default: 4)
	CornerRadius *float32

	// Shadow/elevation (not yet implemented, reserved for future)
	Elevation int
}

// CardWithOptions creates a fully customizable card
func CardWithOptions(content fyne.CanvasObject, opts CardOptions) *fyne.Container {
	// Determine corner radius
	cornerRadius := float32(4)
	if opts.CornerRadius != nil {
		cornerRadius = *opts.CornerRadius
	}

	// Create background rectangle with rounded corners
	bg := canvas.NewRectangle(getBackgroundColor(opts.BackgroundColor))
	bg.CornerRadius = cornerRadius
	bg.StrokeColor = theme.Color(theme.ColorNameSeparator)
	bg.StrokeWidth = 1

	// Apply padding
	paddedContent := content
	if opts.PaddingOverride == nil {
		// Use standard padding
		paddedContent = container.NewPadded(content)
	} else if *opts.PaddingOverride > 0 {
		// Use custom padding
		padding := *opts.PaddingOverride
		paddedContent = container.NewPadded(content)
		paddedContent.(*fyne.Container).Layout = newFixedPaddingLayout(padding)
	}
	// If PaddingOverride == 0, don't add padding

	// Stack background and content
	card := container.NewStack(bg, paddedContent)

	// Apply left indent if specified
	if opts.LeftIndent > 0 {
		return container.NewBorder(
			nil, nil,
			container.NewMax(canvas.NewRectangle(color.Transparent)),
			nil,
			card,
		)
	}

	return card
}

// getBackgroundColor returns the background color or theme default
func getBackgroundColor(colorOverride *color.Color) color.Color {
	if colorOverride != nil {
		return *colorOverride
	}
	// Use a slightly elevated background color for cards
	bgColor := theme.Color(theme.ColorNameBackground)
	// Make it slightly lighter/elevated (subtle effect)
	r, g, b, a := bgColor.RGBA()
	elevated := color.NRGBA{
		R: uint8(min(r>>8+5, 255)),
		G: uint8(min(g>>8+5, 255)),
		B: uint8(min(b>>8+5, 255)),
		A: uint8(a >> 8),
	}
	return elevated
}

// Simple Cards - Preset configurations

// SimpleCard creates a basic card with standard padding
func SimpleCard(content fyne.CanvasObject) *fyne.Container {
	return Card(content)
}

// IndentedCard creates a card indented by 20px (useful for nested content)
func IndentedCard(content fyne.CanvasObject) *fyne.Container {
	return CardWithIndent(content, 20)
}

// CompactCard creates a card with reduced padding
func CompactCard(content fyne.CanvasObject) *fyne.Container {
	padding := float32(4)
	return CardWithOptions(content, CardOptions{
		PaddingOverride: &padding,
	})
}

// NestedCard creates a card indented by a specific level
// level 0 = no indent, level 1 = 20px, level 2 = 40px, etc.
func NestedCard(content fyne.CanvasObject, level int) *fyne.Container {
	indent := float32(level * 20)
	return CardWithIndent(content, indent)
}

// CardList creates multiple cards stacked vertically
func CardList(cards ...fyne.CanvasObject) *fyne.Container {
	// Wrap each item in a card if it isn't already a container
	cardContainers := make([]fyne.CanvasObject, len(cards))
	for i, item := range cards {
		cardContainers[i] = Card(item)
	}
	return container.NewVBox(cardContainers...)
}

// CardSection creates a card with a header and content
func CardSection(title string, content fyne.CanvasObject) *fyne.Container {
	header := Subheading(title)
	section := container.NewVBox(
		header,
		content,
	)
	return Card(section)
}

// CardSectionWithIndent creates an indented card with header and content
func CardSectionWithIndent(title string, content fyne.CanvasObject, indent float32) *fyne.Container {
	header := Subheading(title)
	section := container.NewVBox(
		header,
		content,
	)
	return CardWithIndent(section, indent)
}

// fixedPaddingLayout is a simple layout that adds fixed padding
type fixedPaddingLayout struct {
	padding float32
}

func newFixedPaddingLayout(padding float32) fyne.Layout {
	return &fixedPaddingLayout{padding: padding}
}

func (l *fixedPaddingLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	padding := l.padding * 2 // padding on both sides
	innerSize := fyne.NewSize(
		size.Width-padding,
		size.Height-padding,
	)
	for _, obj := range objects {
		obj.Resize(innerSize)
		obj.Move(fyne.NewPos(l.padding, l.padding))
	}
}

func (l *fixedPaddingLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	for _, obj := range objects {
		objMin := obj.MinSize()
		minSize.Width = max(minSize.Width, objMin.Width)
		minSize.Height = max(minSize.Height, objMin.Height)
	}
	padding := l.padding * 2
	return fyne.NewSize(
		minSize.Width+padding,
		minSize.Height+padding,
	)
}

// Helper functions for min/max
func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
