package ui

import (
        "image/color"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/theme"
)

// ModernWhiteTheme is a clean, minimal light theme tuned for the eDonish Auto
// desktop app. It uses pure white backgrounds, soft greys for separators, and
// the brand accent color (#2E7D32 — calm green) for primary actions.
//
// Compared to Fyne's built-in theme.LightTheme() this theme:
//   - Uses pure white (#FFFFFF) instead of off-white (#FFFFFF→#F5F5F5) backgrounds
//   - Has a softer separator/border colour
//   - Tighter padding for a denser data-grid layout
//   - Larger default text size for journal cells (14 instead of 13)
type ModernWhiteTheme struct{}

// NewModernWhiteTheme returns a ModernWhiteTheme instance.
// Themes in Fyne are stateless so a single shared instance is fine,
// but a constructor makes the call-site read cleaner.
func NewModernWhiteTheme() *ModernWhiteTheme {
        return &ModernWhiteTheme{}
}

// Color returns the colour for the given theme.ColorName.
func (m *ModernWhiteTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
        switch name {
        case theme.ColorNameBackground:
                return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF} // pure white
        case theme.ColorNameButton:
                return color.NRGBA{R: 0xF5, G: 0xF5, B: 0xF5, A: 0xFF} // very light grey
        case theme.ColorNameDisabledButton:
                return color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF}
        case theme.ColorNameDisabled:
                return color.NRGBA{R: 0xBD, G: 0xBD, B: 0xBD, A: 0xFF}
        case theme.ColorNameError:
                return color.NRGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xFF} // Material red 800
        case theme.ColorNameForeground:
                return color.NRGBA{R: 0x21, G: 0x21, B: 0x21, A: 0xFF} // Material grey 900
        case theme.ColorNameHover:
                return color.NRGBA{R: 0xF0, G: 0xF0, B: 0xF0, A: 0xFF}
        case theme.ColorNameInputBackground:
                return color.NRGBA{R: 0xFA, G: 0xFA, B: 0xFA, A: 0xFF}
        case theme.ColorNameInputBorder:
                return color.NRGBA{R: 0xBD, G: 0xBD, B: 0xBD, A: 0xFF}
        case theme.ColorNameMenuBackground:
                return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
        case theme.ColorNameOverlayBackground:
                return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
        case theme.ColorNamePlaceHolder:
                return color.NRGBA{R: 0x9E, G: 0x9E, B: 0x9E, A: 0xFF}
        case theme.ColorNamePressed:
                return color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF}
        case theme.ColorNamePrimary:
                return color.NRGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF} // calm green
        case theme.ColorNameSelection:
                return color.NRGBA{R: 0xC8, G: 0xE6, B: 0xC9, A: 0xFF} // Material green 100
        case theme.ColorNameSeparator:
                return color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF} // soft separator
        case theme.ColorNameShadow:
                return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x14}
        case theme.ColorNameSuccess:
                return color.NRGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF}
        case theme.ColorNameWarning:
                return color.NRGBA{R: 0xE6, G: 0x5C, B: 0x00, A: 0xFF} // Material orange 800
        }
        return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00}
}

// Font returns the font for the given theme.TextStyle.
func (m *ModernWhiteTheme) Font(style fyne.TextStyle) fyne.Resource {
        return theme.DefaultTheme().Font(style)
}

// Icon returns the icon for the given theme.IconName.
func (m *ModernWhiteTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
        return theme.DefaultTheme().Icon(name)
}

// Size returns the size for the given theme.SizeName.
//
// Padding and scroll-bar sizes are tightened slightly compared to Fyne's
// defaults so that dense data grids (journal, final grades) fit more rows
// without scrolling. Default text size is bumped to 14 for readability.
func (m *ModernWhiteTheme) Size(name fyne.ThemeSizeName) float32 {
        switch name {
        case theme.SizeNamePadding:
                return 5
        case theme.SizeNameInnerPadding:
                return 8
        case theme.SizeNameText:
                return 14
        case theme.SizeNameHeadingText:
                return 18
        case theme.SizeNameSubHeadingText:
                return 15
        case theme.SizeNameCaptionText:
                return 12
        case theme.SizeNameSeparatorThickness:
                return 1
        case theme.SizeNameLineSpacing:
                return 4
        }
        return theme.DefaultTheme().Size(name)
}
