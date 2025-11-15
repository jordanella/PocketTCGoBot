package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ChipStyle defines the visual style of a chip
type ChipStyle int

const (
	ChipStyleDefault ChipStyle = iota // Default gray chip
	ChipStylePrimary                  // Primary color chip
	ChipStyleSuccess                  // Green success chip
	ChipStyleWarning                  // Orange warning chip
	ChipStyleDanger                   // Red danger chip
	ChipStyleInfo                     // Blue info chip
)

// Chip creates a clickable chip/badge component
// If tapped is nil, the chip is not clickable
func Chip(text string, tapped func()) *fyne.Container {
	return ChipWithStyle(text, ChipStyleDefault, tapped)
}

// ChipWithStyle creates a chip with a specific style
func ChipWithStyle(text string, style ChipStyle, tapped func()) *fyne.Container {
	bgColor := getChipColor(style)

	// Use smallest text size for compact chips
	label := Caption(text)

	// Create rounded rectangle background with smaller radius
	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 8 // Reduced from 12
	bg.Resize(fyne.NewSize(label.MinSize().Width, label.MinSize().Height))

	// Minimal padding - custom layout
	const chipPadding = 1 // Very small padding
	padded := container.New(&chipPaddingLayout{padding: chipPadding}, label)

	// Stack background and label
	chip := container.NewStack(bg, padded)

	// Make clickable if tapped function provided
	if tapped != nil {
		// Wrap in a tappable widget
		return container.NewMax(
			&tappableContainer{
				Container: chip,
				onTapped:  tapped,
			},
		)
	}

	return chip
}

// chipPaddingLayout provides minimal padding for chips
type chipPaddingLayout struct {
	padding float32
}

func (l *chipPaddingLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	padding := l.padding
	for _, obj := range objects {
		obj.Resize(fyne.NewSize(
			size.Width-padding*2,
			size.Height-padding*2,
		))
		obj.Move(fyne.NewPos(padding, padding))
	}
}

func (l *chipPaddingLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	for _, obj := range objects {
		objMin := obj.MinSize()
		if objMin.Width > minSize.Width {
			minSize.Width = objMin.Width
		}
		if objMin.Height > minSize.Height {
			minSize.Height = objMin.Height
		}
	}
	padding := l.padding * 2
	return fyne.NewSize(minSize.Width+padding, minSize.Height+padding)
}

// StatusChip creates a chip showing status with appropriate color
// Common statuses: "Active", "Idle", "Error", "Pending", "Completed"
func StatusChip(status string) *fyne.Container {
	var style ChipStyle

	// Map common status strings to styles
	switch status {
	case "Active", "Running", "Online", "Completed", "Success":
		style = ChipStyleSuccess
	case "Idle", "Paused", "Pending", "Waiting":
		style = ChipStyleInfo
	case "Error", "Failed", "Offline", "Stopped":
		style = ChipStyleDanger
	case "Warning", "Limited":
		style = ChipStyleWarning
	default:
		style = ChipStyleDefault
	}

	return ChipWithStyle(status, style, nil)
}

// NavigationChip creates a clickable chip that navigates to a view
// Styled as a primary chip to indicate it's interactive
func NavigationChip(text string, onNavigate func()) *fyne.Container {
	return ChipWithStyle(text, ChipStylePrimary, onNavigate)
}

// ChipList creates a horizontal list of chips
func ChipList(chips ...*fyne.Container) *fyne.Container {
	objects := make([]fyne.CanvasObject, len(chips))
	for i, chip := range chips {
		objects[i] = chip
	}
	return container.NewHBox(objects...)
}

// TruncatedChipList creates a chip list that truncates after maxVisible items
// Shows "and N more..." if there are more items
func TruncatedChipList(items []string, maxVisible int, onChipTapped func(string)) *fyne.Container {
	chips := []fyne.CanvasObject{}

	// Show up to maxVisible items
	displayCount := len(items)
	if displayCount > maxVisible {
		displayCount = maxVisible
	}

	for i := 0; i < displayCount; i++ {
		item := items[i]
		var tapped func()
		if onChipTapped != nil {
			tapped = func() { onChipTapped(item) }
		}
		chip := Chip(item, tapped)
		chips = append(chips, chip)
	}

	// Add "and N more..." if truncated
	if len(items) > maxVisible {
		remaining := len(items) - maxVisible
		moreLabel := Caption(fmt.Sprintf("and %d more...", remaining))
		chips = append(chips, moreLabel)
	}

	return container.NewHBox(chips...)
}

// NavigationChipList creates a list of navigation chips
func NavigationChipList(items []string, maxVisible int, onNavigate func(string)) *fyne.Container {
	chips := []fyne.CanvasObject{}

	// Show up to maxVisible items
	displayCount := len(items)
	if displayCount > maxVisible {
		displayCount = maxVisible
	}

	for i := 0; i < displayCount; i++ {
		item := items[i]
		chip := NavigationChip(item, func() { onNavigate(item) })
		chips = append(chips, chip)
	}

	// Add "and N more..." if truncated
	if len(items) > maxVisible {
		remaining := len(items) - maxVisible
		moreLabel := Caption(fmt.Sprintf("and %d more...", remaining))
		chips = append(chips, moreLabel)
	}

	return container.NewHBox(chips...)
}

// getChipColor returns the background color for a chip style
func getChipColor(style ChipStyle) color.Color {
	switch style {
	case ChipStylePrimary:
		return theme.Color(theme.ColorNamePrimary)
	case ChipStyleSuccess:
		return color.NRGBA{R: 76, G: 175, B: 80, A: 255} // Green
	case ChipStyleWarning:
		return color.NRGBA{R: 255, G: 152, B: 0, A: 255} // Orange
	case ChipStyleDanger:
		return color.NRGBA{R: 244, G: 67, B: 54, A: 255} // Red
	case ChipStyleInfo:
		return color.NRGBA{R: 33, G: 150, B: 243, A: 255} // Blue
	default:
		// Default gray
		return color.NRGBA{R: 140, G: 140, B: 140, A: 255}
	}
}

// tappableContainer is a helper to make containers tappable
type tappableContainer struct {
	*fyne.Container
	onTapped func()
}

func (t *tappableContainer) Tapped(_ *fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

func (t *tappableContainer) TappedSecondary(_ *fyne.PointEvent) {
	// No secondary tap action
}

// LabeledChipList creates a label followed by a list of chips
// Example: "Account Pools: <pool A> <pool B> <pool C>"
func LabeledChipList(label string, items []string, maxVisible int, onChipTapped func(string)) *fyne.Container {
	labelWidget := BoldText(label + ":")
	chipList := TruncatedChipList(items, maxVisible, onChipTapped)

	return container.NewHBox(
		labelWidget,
		chipList,
	)
}

// LabeledNavigationChipList creates a label followed by navigation chips
func LabeledNavigationChipList(label string, items []string, maxVisible int, onNavigate func(string)) *fyne.Container {
	labelWidget := BoldText(label + ":")
	chipList := NavigationChipList(items, maxVisible, onNavigate)

	return container.NewHBox(
		labelWidget,
		chipList,
	)
}

// RemovableChip creates a chip with a remove button
func RemovableChip(text string, onRemove func()) *fyne.Container {
	label := widget.NewLabel(text)
	removeBtn := widget.NewButton("×", onRemove)
	removeBtn.Importance = widget.LowImportance

	bg := canvas.NewRectangle(getChipColor(ChipStyleDefault))
	bg.CornerRadius = 8 // Match other chips

	content := container.NewHBox(label, removeBtn)

	// Use minimal padding
	const chipPadding = 1
	padded := container.New(&chipPaddingLayout{padding: chipPadding}, content)

	return container.NewStack(bg, padded)
}

// RemovableChipList creates a list of chips with remove buttons
func RemovableChipList(items []string, onRemove func(string)) *fyne.Container {
	chips := []fyne.CanvasObject{}

	for _, item := range items {
		itemCopy := item
		chip := RemovableChip(item, func() { onRemove(itemCopy) })
		chips = append(chips, chip)
	}

	return container.NewHBox(chips...)
}
