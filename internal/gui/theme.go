package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	// DefaultWindowSize is the default window dimensions
	DefaultWindowSize = fyne.NewSize(1200, 800)

	// Colors
	ColorPrimary   = color.NRGBA{R: 63, G: 81, B: 181, A: 255}   // Material Indigo
	ColorSecondary = color.NRGBA{R: 255, G: 64, B: 129, A: 255}  // Material Pink
	ColorSuccess   = color.NRGBA{R: 76, G: 175, B: 80, A: 255}   // Material Green
	ColorWarning   = color.NRGBA{R: 255, G: 152, B: 0, A: 255}   // Material Orange
	ColorError     = color.NRGBA{R: 244, G: 67, B: 54, A: 255}   // Material Red
	ColorInfo      = color.NRGBA{R: 33, G: 150, B: 243, A: 255}  // Material Blue
	ColorBackground = color.NRGBA{R: 18, G: 18, B: 18, A: 255}   // Dark background
)

// BotTheme implements a custom theme for the bot GUI
type BotTheme struct{}

func (t *BotTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return ColorPrimary
	case theme.ColorNameBackground:
		return ColorBackground
	case theme.ColorNameButton:
		return ColorPrimary
	case theme.ColorNameSuccess:
		return ColorSuccess
	case theme.ColorNameWarning:
		return ColorWarning
	case theme.ColorNameError:
		return ColorError
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *BotTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *BotTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *BotTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 18
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNamePadding:
		return 8
	default:
		return theme.DefaultTheme().Size(name)
	}
}
