package ui

import "againrom/pkg/ui/widgetkit"

// The kit's widgets under the names this package's screens use. These are
// type aliases and constants, not second builders: each widget has its one
// builder in widgetkit.
type (
	CheckStyle     = widgetkit.CheckStyle
	CheckSpec      = widgetkit.CheckSpec
	TextSelection  = widgetkit.TextSelection
	TextInputRules = widgetkit.TextInputRules
	TextInputKeys  = widgetkit.TextInputKeys
	TextInputState = widgetkit.TextInputState
	WindowRow      = widgetkit.WindowRow
	windowForm     = widgetkit.WindowForm
	frameSides     = widgetkit.FrameSides
)

const (
	CheckArt       = widgetkit.CheckArt
	CheckText      = widgetkit.CheckText
	CheckRadioText = widgetkit.CheckRadioText

	TextCaretSelection      = widgetkit.TextCaretSelection
	TextReplaceAllSelection = widgetkit.TextReplaceAllSelection
	TextStickyAllSelection  = widgetkit.TextStickyAllSelection
	TextDetachedSelection   = widgetkit.TextDetachedSelection

	edgeTopBit    = widgetkit.EdgeTop
	edgeBottomBit = widgetkit.EdgeBottom
	edgeLeftBit   = widgetkit.EdgeLeft
	edgeRightBit  = widgetkit.EdgeRight
)
