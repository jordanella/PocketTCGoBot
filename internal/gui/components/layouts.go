package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// LabelButtonsRow creates a row with labels on the left and buttons right-aligned
// This is the common pattern from your mockups:
// "Instance Name - Index <mumu index>    [ Pause ] [ Stop ] [ Abort ] [Shutdown]"
func LabelButtonsRow(labels fyne.CanvasObject, buttons ...fyne.CanvasObject) *fyne.Container {
	buttonContainer := container.NewHBox(buttons...)

	return container.NewBorder(
		nil, nil,
		labels,
		buttonContainer,
		layout.NewSpacer(),
	)
}

// InlineLabels creates multiple labels displayed inline with separators
// Example: "Instance Name - Index 5" or "Label1 - Label2 - Label3"
func InlineLabels(separator string, labels ...fyne.CanvasObject) *fyne.Container {
	if len(labels) == 0 {
		return container.NewHBox()
	}

	items := []fyne.CanvasObject{}
	for i, label := range labels {
		items = append(items, label)
		if i < len(labels)-1 && separator != "" {
			items = append(items, widget.NewLabel(separator))
		}
	}

	return container.NewHBox(items...)
}

// TwoColumnLayout creates a two-column layout with a resizable split
// Left column is typically a list, right column is detail view
func TwoColumnLayout(leftContent, rightContent fyne.CanvasObject, leftMinWidth float32) *container.Split {
	split := container.NewHSplit(leftContent, rightContent)
	split.Offset = 0.3 // 30% left, 70% right by default

	return split
}

// TabPanel creates a tabbed interface
func TabPanel(tabs ...*container.TabItem) *container.AppTabs {
	return container.NewAppTabs(tabs...)
}

// ConditionalWidget wraps a widget that can be shown/hidden based on a condition
type ConditionalWidget struct {
	widget    fyne.CanvasObject
	condition func() bool
}

// NewConditionalWidget creates a widget that only shows when condition returns true
func NewConditionalWidget(widget fyne.CanvasObject, condition func() bool) *ConditionalWidget {
	return &ConditionalWidget{
		widget:    widget,
		condition: condition,
	}
}

// GetWidget returns the widget if the condition is true, nil otherwise
func (cw *ConditionalWidget) GetWidget() fyne.CanvasObject {
	if cw.condition() {
		return cw.widget
	}
	return nil
}

// ConditionalContainer creates a container that conditionally includes widgets
// This supports your "/" notation for conditional display
func ConditionalContainer(items ...fyne.CanvasObject) *fyne.Container {
	visible := []fyne.CanvasObject{}
	for _, item := range items {
		if item != nil {
			visible = append(visible, item)
		}
	}
	return container.NewHBox(visible...)
}

// SectionHeader creates a section header with optional action buttons
// Example: "Active Groups" or "Inactive Groups" from orchestration mockup
func SectionHeader(title string, actions ...fyne.CanvasObject) *fyne.Container {
	header := Subheading(title)

	if len(actions) > 0 {
		return container.NewBorder(
			nil, nil,
			header,
			container.NewHBox(actions...),
			layout.NewSpacer(),
		)
	}

	return container.NewVBox(header)
}

// FieldRow creates a row with a label and an input field
// Handles the "{field}" pattern from your mockups
func FieldRow(label string, field fyne.CanvasObject) *fyne.Container {
	labelWidget := BoldText(label)
	return container.NewVBox(
		labelWidget,
		field,
	)
}

// FieldRowInline creates an inline field row (label and field side-by-side)
func FieldRowInline(label string, field fyne.CanvasObject) *fyne.Container {
	labelWidget := BoldText(label)
	return container.NewHBox(labelWidget, field)
}

// RequiredFieldRow creates a field row with a required indicator (*)
func RequiredFieldRow(label string, field fyne.CanvasObject, hint string) *fyne.Container {
	labelWidget := BoldText(label + " *")
	hintWidget := Caption(hint)

	return container.NewVBox(
		labelWidget,
		field,
		hintWidget,
	)
}

// ActionBar creates a bottom action bar with buttons
// Supports conditional display and grouping
func ActionBar(leftActions, rightActions []fyne.CanvasObject) *fyne.Container {
	left := container.NewHBox(leftActions...)
	right := container.NewHBox(rightActions...)

	return container.NewBorder(
		nil, nil,
		left,
		right,
		layout.NewSpacer(),
	)
}

// ActionBarSingle creates an action bar with all buttons on the right
func ActionBarSingle(actions ...fyne.CanvasObject) *fyne.Container {
	return container.NewBorder(
		nil, nil,
		nil,
		container.NewHBox(actions...),
		layout.NewSpacer(),
	)
}

// ReorderableRow creates a row with up/down buttons for reordering
// Common pattern in your mockups for sortable lists
func ReorderableRow(content fyne.CanvasObject, onMoveUp, onMoveDown, onRemove func()) *fyne.Container {
	upBtn := widget.NewButton("▲", onMoveUp)
	downBtn := widget.NewButton("▼", onMoveDown)
	removeBtn := widget.NewButton("Remove", onRemove)

	buttons := container.NewHBox(upBtn, downBtn, removeBtn)

	return container.NewBorder(
		nil, nil,
		content,
		buttons,
		layout.NewSpacer(),
	)
}

// ReorderableRowWithToggle creates a reorderable row with enable/disable toggle
func ReorderableRowWithToggle(
	content fyne.CanvasObject,
	enabled bool,
	onMoveUp, onMoveDown, onRemove, onToggle func(),
) *fyne.Container {
	upBtn := widget.NewButton("▲", onMoveUp)
	downBtn := widget.NewButton("▼", onMoveDown)
	removeBtn := widget.NewButton("Remove", onRemove)

	toggleText := "Disable"
	if !enabled {
		toggleText = "Enable"
	}
	toggleBtn := widget.NewButton(toggleText, onToggle)

	buttons := container.NewHBox(upBtn, downBtn, removeBtn, toggleBtn)

	return container.NewBorder(
		nil, nil,
		content,
		buttons,
		layout.NewSpacer(),
	)
}

// TableRow creates a simple table row
// For your account tables with columns like "Account | Packs | Shinedust | Status"
func TableRow(cells ...fyne.CanvasObject) *fyne.Container {
	return container.NewHBox(cells...)
}

// TableHeader creates a table header row with bold text
func TableHeader(headers ...string) *fyne.Container {
	cells := make([]fyne.CanvasObject, len(headers))
	for i, header := range headers {
		cells[i] = BoldText(header)
	}
	return container.NewHBox(cells...)
}

// Spacer creates a vertical spacer of specified height
func Spacer(height float32) *fyne.Container {
	spacer := container.NewVBox()
	spacer.Resize(fyne.NewSize(1, height))
	return spacer
}

// Divider creates a visual divider (horizontal separator)
func Divider() fyne.CanvasObject {
	return widget.NewSeparator()
}

// InfoRow creates an info row with label and value
// Example: "Started: 2 hours ago" or "Pool Progress: 5/10"
func InfoRow(label, value string) *fyne.Container {
	labelWidget := BoldText(label + ":")
	valueWidget := Body(value)
	return container.NewHBox(labelWidget, valueWidget)
}

// InlineInfoRow creates multiple info items in a single row
// Example: "Started <time>   Pool Progress <remaining>/<total>"
func InlineInfoRow(items ...fyne.CanvasObject) *fyne.Container {
	return container.NewHBox(items...)
}
