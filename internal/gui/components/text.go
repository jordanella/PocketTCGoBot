package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Text component styles following Material Design typography scale
const (
	// Font sizes
	SizeHeading    float32 = 24
	SizeSubheading float32 = 18
	SizeBody       float32 = 14
	SizeCaption    float32 = 12
)

// Heading creates a large, bold heading text component
// Use for main page titles and primary headers
func Heading(text string) *widget.RichText {
	return widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				SizeName: theme.SizeNameHeadingText,
				TextStyle: fyne.TextStyle{
					Bold: true,
				},
			},
		},
	)
}

// Subheading creates a medium-sized heading text component
// Use for section headers and card titles
func Subheading(text string) *widget.RichText {
	return widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				SizeName: theme.SizeNameSubHeadingText,
				TextStyle: fyne.TextStyle{
					Bold: true,
				},
			},
		},
	)
}

// Body creates standard body text
// Use for descriptions, paragraphs, and general content
func Body(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

// BodyRich creates standard body text with RichText support for custom sizing
// Use when you need more control over text styling
func BodyRich(text string) *widget.RichText {
	return widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				SizeName: theme.SizeNameText,
			},
		},
	)
}

// Caption creates small caption text
// Use for hints, secondary information, and footnotes
func Caption(text string) *widget.RichText {
	return widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				SizeName: theme.SizeNameCaptionText,
				ColorName: theme.ColorNameForeground,
			},
		},
	)
}

// SizedText creates a canvas.Text with a specific size
// Use when you need precise control over font size
// Note: This returns canvas.Text, not a widget, so it won't auto-update with theme changes
func SizedText(text string, size float32, bold bool) *canvas.Text {
	t := canvas.NewText(text, theme.Color(theme.ColorNameForeground))
	t.TextSize = size
	if bold {
		t.TextStyle.Bold = true
	}
	return t
}

// CustomRichText creates a RichText with fully customizable styling
// Use when none of the preset styles fit your needs
type CustomRichTextStyle struct {
	Text      string
	SizeName  fyne.ThemeSizeName
	Bold      bool
	Italic    bool
	Monospace bool
	ColorName fyne.ThemeColorName
}

func CustomRichText(style CustomRichTextStyle) *widget.RichText {
	return widget.NewRichText(
		&widget.TextSegment{
			Text: style.Text,
			Style: widget.RichTextStyle{
				SizeName: style.SizeName,
				TextStyle: fyne.TextStyle{
					Bold:      style.Bold,
					Italic:    style.Italic,
					Monospace: style.Monospace,
				},
				ColorName: style.ColorName,
			},
		},
	)
}

// BoldText creates bold text at standard size
func BoldText(text string) *widget.RichText {
	return widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				SizeName: theme.SizeNameText,
				TextStyle: fyne.TextStyle{
					Bold: true,
				},
			},
		},
	)
}

// MonospaceText creates monospace text for code or technical content
func MonospaceText(text string) *widget.RichText {
	return widget.NewRichText(
		&widget.TextSegment{
			Text: text,
			Style: widget.RichTextStyle{
				SizeName: theme.SizeNameText,
				TextStyle: fyne.TextStyle{
					Monospace: true,
				},
			},
		},
	)
}
