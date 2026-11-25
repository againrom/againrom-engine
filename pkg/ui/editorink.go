package ui

import "image/color"

// The editor's one ink table. Every colour an editor window shows is named
// here; the popup painter reads editorPopupInk and the HTML page writer
// reads the same value, so a page shows the painter's colours.
var (
	editorInk       = color.RGBA{0x20, 0x1b, 0x16, 255}
	editorPanel     = color.RGBA{0x30, 0x29, 0x1f, 255}
	editorPaper     = color.RGBA{0xd6, 0xc4, 0x9b, 255}
	editorLight     = color.RGBA{0xf3, 0xe5, 0xbe, 255}
	editorGold      = color.RGBA{0xb8, 0x95, 0x4f, 255}
	editorWarning   = color.RGBA{0xb8, 0x5e, 0x46, 255}
	editorCondition = color.RGBA{0x78, 0xb6, 0xc2, 255}
	// popupHintInk is the grey of an empty field's hint and a row that is off.
	popupHintInk = color.RGBA{0x8a, 0x80, 0x6c, 255}
	// editorSelectInk is the ground of the selected text in a typed field.
	editorSelectInk = color.RGBA{0x55, 0x65, 0x8c, 255}
	// editorGridInk is the map grid's line.
	editorGridInk = color.RGBA{0x60, 0x58, 0x48, 255}
	editorGhost   = color.RGBA{128, 128, 128, 128}
	// legendMutedInk is the label of a legend row whose player is not shown.
	legendMutedInk = color.RGBA{0x7a, 0x70, 0x5c, 255}
	// The height mode icons' ground and the part a stroke lifts.
	heightIconGround = color.RGBA{150, 108, 60, 255}
	heightIconLifted = color.RGBA{88, 176, 72, 255}
	// The campaign graph's kind inks, its town area's ground, and the zero
	// ink of a box with no band.
	graphMainInk    = color.RGBA{0xd8, 0x6a, 0x52, 255}
	graphSideInk    = color.RGBA{0x7c, 0xb0, 0x5c, 255}
	graphTalkInk    = color.RGBA{0x62, 0x9c, 0xd8, 255}
	graphHandoutInk = color.RGBA{0xa8, 0x84, 0xd0, 255}
	graphFilmInk    = color.RGBA{0xd8, 0x84, 0xb4, 255}
	graphRegionFill = color.RGBA{0x29, 0x23, 0x1a, 255}
	graphNoBand     color.RGBA
)

// areaInks tell overlapping triggers apart; gold is the selection's.
var areaInks = []color.RGBA{
	{0x4f, 0xd6, 0xe0, 255}, {0xe0, 0x6f, 0xd6, 255}, {0x8c, 0xe0, 0x4f, 255}, {0xf0, 0x9a, 0x3e, 255},
	{0x7a, 0x9c, 0xff, 255}, {0xff, 0x7a, 0x8a, 255}, {0xd8, 0xe0, 0x6a, 255}, {0xb0, 0x86, 0xff, 255},
}

// passKindFills are the passability view's fills of a valued cell by how many
// move kinds its value closes: none, one, two, three; passRawFill is a value
// no row names.
var (
	passKindFills = [...]color.RGBA{passOpenFill, {0xf0, 0xb0, 0x20, 0x60}, passClosedFill, {0x80, 0x10, 0x70, 0x80}}
	passRawFill   = color.RGBA{0x30, 0x50, 0xe0, 0x60}
)

// relationInks are the relation matrix's icons: neutral, hostile, ally.
var relationInks = []color.RGBA{{176, 176, 176, 255}, {214, 48, 40, 255}, {64, 184, 72, 255}}

// editorPlayerTints is the one player-tint colour source: the colour of player
// number n is entry n-1, so two players of a map never share one, whatever
// colour field their map records carry.
var editorPlayerTints = [16]color.RGBA{
	{0xe6, 0x19, 0x4b, 255}, // 1 red
	{0x3c, 0xb4, 0x4b, 255}, // 2 green
	{0x43, 0x63, 0xd8, 255}, // 3 blue
	{0xff, 0xe1, 0x19, 255}, // 4 yellow
	{0xf5, 0x82, 0x31, 255}, // 5 orange
	{0x91, 0x1e, 0xb4, 255}, // 6 purple
	{0x42, 0xd4, 0xf4, 255}, // 7 cyan
	{0xf0, 0x32, 0xe6, 255}, // 8 magenta
	{0xbf, 0xef, 0x45, 255}, // 9 lime
	{0xfa, 0xbe, 0xd4, 255}, // 10 pink
	{0x46, 0x99, 0x90, 255}, // 11 teal
	{0xdc, 0xbe, 0xff, 255}, // 12 lavender
	{0x9a, 0x63, 0x24, 255}, // 13 brown
	{0x80, 0x00, 0x00, 255}, // 14 maroon
	{0x00, 0x00, 0x75, 255}, // 15 navy
	{0xa9, 0xa9, 0xa9, 255}, // 16 grey
}

// editorPopupInk is the popup's palette from the table, with the push
// button's own caption and bevel inks the button painter draws.
var editorPopupInk = popupInk{
	Ground: editorPanel, Edge: editorGold, TitleGround: editorInk, Title: editorGold, Label: editorPaper, Value: editorLight,
	Hint: popupHintInk, Button: pushButtonIdleInk, BevelLight: dialogueBevelLight, BevelDark: dialogueBevelDark,
}
