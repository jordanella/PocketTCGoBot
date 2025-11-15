package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CustomButton wraps a standard button with custom text sizing capabilities
type CustomButton struct {
	widget.Button
	textSize    float32
	labelText   string
	captionText string

	// Internal UI elements
	mainLabel    *canvas.Text
	captionLabel *canvas.Text
	container    *fyne.Container
}

// NewCustomButton creates a button with a main label and optional caption
// The main label can have a custom size, and the caption appears below it in smaller text
func NewCustomButton(labelText string, labelSize float32, captionText string, tapped func()) *CustomButton {
	btn := &CustomButton{
		textSize:    labelSize,
		labelText:   labelText,
		captionText: captionText,
	}

	btn.ExtendBaseWidget(btn)
	btn.OnTapped = tapped

	return btn
}

// CreateRenderer creates the custom button renderer with stacked text
func (b *CustomButton) CreateRenderer() fyne.WidgetRenderer {
	// Create main label
	b.mainLabel = canvas.NewText(b.labelText, theme.Color(theme.ColorNameForeground))
	b.mainLabel.TextSize = b.textSize
	b.mainLabel.TextStyle.Bold = true
	b.mainLabel.Alignment = fyne.TextAlignCenter

	// Create caption if provided
	var content fyne.CanvasObject
	if b.captionText != "" {
		b.captionLabel = canvas.NewText(b.captionText, theme.Color(theme.ColorNameForeground))
		b.captionLabel.TextSize = SizeCaption
		b.captionLabel.Alignment = fyne.TextAlignCenter

		content = container.NewVBox(
			b.mainLabel,
			b.captionLabel,
		)
	} else {
		content = container.NewVBox(b.mainLabel)
	}

	b.container = container.NewPadded(content)

	return widget.NewSimpleRenderer(b.container)
}

// SetLabel updates the main label text
func (b *CustomButton) SetLabel(text string) {
	b.labelText = text
	if b.mainLabel != nil {
		b.mainLabel.Text = text
		b.mainLabel.Refresh()
	}
}

// SetCaption updates the caption text
func (b *CustomButton) SetCaption(text string) {
	b.captionText = text
	if b.captionLabel != nil {
		b.captionLabel.Text = text
		b.captionLabel.Refresh()
	}
}

// PrimaryButton creates a high-importance button (recommended for main actions)
func PrimaryButton(text string, tapped func()) *widget.Button {
	btn := widget.NewButton(text, tapped)
	btn.Importance = widget.HighImportance
	return btn
}

// SecondaryButton creates a standard button (default styling)
func SecondaryButton(text string, tapped func()) *widget.Button {
	return widget.NewButton(text, tapped)
}

// DangerButton creates a button for destructive actions
func DangerButton(text string, tapped func()) *widget.Button {
	btn := widget.NewButton(text, tapped)
	btn.Importance = widget.DangerImportance
	return btn
}

// IconButton creates a button with an icon and text
func IconButton(text string, icon fyne.Resource, tapped func()) *widget.Button {
	btn := widget.NewButtonWithIcon(text, icon, tapped)
	return btn
}

// LargeButton creates a button with larger text for emphasis
func LargeButton(text string, tapped func()) *widget.Button {
	btn := widget.NewButton(text, tapped)
	// Standard button, but you can wrap it in a container with custom size if needed
	return btn
}

// StackedButton creates a button with a main label and caption beneath it
// Example: "Launch" with "Start all bots" beneath
func StackedButton(mainText string, captionText string, tapped func()) fyne.CanvasObject {
	// Create the text stack
	mainLabel := SizedText(mainText, 16, true)
	mainLabel.Alignment = fyne.TextAlignCenter

	captionLabel := SizedText(captionText, 11, false)
	captionLabel.Alignment = fyne.TextAlignCenter

	textStack := container.NewVBox(
		mainLabel,
		captionLabel,
	)

	// Create a tappable container that acts like a button
	btn := widget.NewButton("", tapped)

	// Overlay the text on the button
	return container.NewStack(
		btn,
		container.NewCenter(container.NewPadded(textStack)),
	)
}

// CompactButton creates a smaller button for tight spaces
func CompactButton(text string, tapped func()) *widget.Button {
	btn := widget.NewButton(text, tapped)
	// Fyne doesn't have built-in compact sizing, but this provides semantic clarity
	return btn
}

// ButtonGroup creates a horizontal group of related buttons with consistent spacing
func ButtonGroup(buttons ...*widget.Button) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, len(buttons))
	for i, btn := range buttons {
		objects[i] = btn
	}
	return container.NewHBox(objects...)
}

// ButtonGroupVertical creates a vertical group of related buttons
func ButtonGroupVertical(buttons ...*widget.Button) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, len(buttons))
	for i, btn := range buttons {
		objects[i] = btn
	}
	return container.NewVBox(objects...)
}
