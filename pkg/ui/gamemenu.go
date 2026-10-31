package ui

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"againrom/pkg/locale"
	"againrom/pkg/render/text"
)

// The in-game menu (0158). Escape raises it over a mission or over the town
// square, the world stops while it is up, and the surface behind it stays drawn
// and darkened.
//
// There are TWO SURFACES AND NOT ONE (MENU-ESC-010, High): the original branches
// on its own UI state word and builds a different panel class with a different
// row list for each. Which one opens is decided by the screen Escape was pressed
// on, which the flow already records in menuBack.

// gameMenuSurface names which panel is up.
type gameMenuSurface int

const (
	// gameMenuMission is the panel raised over a map screen: 340x340 at
	// (100,60), seven rows.
	gameMenuMission gameMenuSurface = iota
	// gameMenuTown is the panel raised at the town square: 340x240 at
	// (100,100), five rows.
	gameMenuTown
)

// gameMenuAction is what a row does. It is a value and not a callback: the
// row order is contract, and a table of closures could be reordered without
// failing anything.
type gameMenuAction int

const (
	gameMenuSave gameMenuAction = iota
	gameMenuLoad
	gameMenuDiplomacy
	gameMenuGameOptions
	gameMenuSoundOptions
	gameMenuQuestObjectives
	gameMenuEndQuest
	gameMenuAbortGame
	gameMenuReturn
	gameMenuPageReturn
	gameMenuToggleTips
	gameMenuToggleSound
	gameMenuVolumeDown
	gameMenuVolumeUp
	gameMenuTestSound
	gameMenuConfirmEndQuest
	gameMenuVictory
	gameMenuExitMain
	gameMenuExitWindows
	gameMenuSpeedDown
	gameMenuSpeedUp
	gameMenuTooltipDelay
	gameMenuDayNight
	gameMenuHealth
	gameMenuDamage
	gameMenuFormation
	gameMenuRetreat
	gameMenuPathfinding
	gameMenuSmoothing
	gameMenuShadows
	gameMenuLighting
	gameMenuAnimation
	gameMenuAutoHealing
	gameMenuLootDots
	gameMenuAutoCast
	gameMenuMusicVolume
	gameMenuEffectsVolume
	gameMenuSpeechVolume
	gameMenuAcknowledgments
	gameMenuMusicTracks
	gameMenuMusicRandom
	gameMenuMusicPlay
	gameMenuMusicStop
	gameMenuMusicUp
	gameMenuMusicDown
	gameMenuMusicScroll
	// gameMenuModScreen opens the mod screen its row names.
	gameMenuModScreen
	// gameMenuModAction opens the confirming page of the mod action its row
	// names; gameMenuModActionConfirm runs it.
	gameMenuModAction
	gameMenuModActionConfirm
	// gameMenuOptionsCancel leaves Game Options without writing its local copy.
	gameMenuOptionsCancel
	gameMenuTimedAutosave
	gameMenuAutosaveMinutes
)

// gameMenuPage is the state within ScreenGameMenu. Keeping every destination
// on this screen keeps the map or town behind it held and darkened for the
// whole nested visit.
type gameMenuPage int

const (
	gameMenuRoot gameMenuPage = iota
	gameMenuGameOptionsPage
	gameMenuSoundOptionsPage
	gameMenuQuestObjectivesPage
	gameMenuDiplomacyPage
	gameMenuEndQuestConfirmation
	gameMenuAbortGameConfirmation
	gameMenuModActionConfirmation
)

// gameMenuRow is one entry: the label as authored, the accelerator to use when
// the label marks none, and what the row does.
//
// THE ACCELERATOR IS NOT A FIELD. MENU-KEY-013's whole content is that the
// letter after the label's `~` wins over the constructor's immediate, and a
// resolved letter stored beside the label is exactly the pair that can come
// to disagree. Fallback is stored because a label with no `~` has nowhere
// else to take one from.
type gameMenuRow struct {
	Label    string
	Fallback byte
	Action   gameMenuAction
	Enabled  bool
	// Literal keeps informational text from treating a `~` in shipped map
	// prose as an accelerator mark.
	Literal bool
	// Status displays a readable value without making it selectable.
	Status bool
	// Mod is the 1-based index of the mod screen a gameMenuModScreen row opens,
	// and zero for every other row.
	Mod int
}

func (r gameMenuRow) text() string {
	if r.Literal {
		return r.Label
	}
	return gameMenuLabelText(r.Label)
}

func (r gameMenuRow) accelerator(selector int) (byte, bool) {
	if r.Literal {
		return 0, false
	}
	return gameMenuAccelerator(r.Label, r.Fallback, selector), true
}

// Panel geometry, in the fixed 640x480 frame (MENU-ESC-010, MENU-ART-014).
// The immediates are the decoded ones: the mission panel is pushed as
// 0x190/0x1b8/0x3c/0x64 and the town panel as 0x154/0x1b8/0x64/0x64.
const (
	gameMenuPanelLeft = 100
	gameMenuPanelW    = 340

	gameMenuMissionTop = 60
	gameMenuMissionH   = 340

	gameMenuTownTop = 100
	gameMenuTownH   = 240

	// The row helper's own numbers: the first row's top is 0x28 = 40 below the
	// panel's, the pitch is 0x1e = 30, the left inset is 0x28 = 40 and the
	// right edge is the panel width less 0x30 = 48. So a row is 252x30.
	gameMenuRowTop    = 40
	gameMenuRowPitch  = 30
	gameMenuRowLeft   = 40
	gameMenuRowInsetR = 48
)

// gameMenuBaseRows is how many rows the shipped root list of a surface has, and
// gameMenuBottomMargin the room kept under the last row when mod rows make the
// panel taller than the decoded one.
const gameMenuBottomMargin = 20

func gameMenuBaseRows(s gameMenuSurface) int {
	if s == gameMenuTown {
		return 6
	}
	return 7
}

// gameMenuModRows counts the mod screen rows of a list.
func gameMenuModRows(rows []gameMenuRow) int {
	n := 0
	for _, r := range rows {
		if r.Mod > 0 {
			n++
		}
	}
	return n
}

// gameMenuPanelRectFor is the panel's rectangle for a row list: the decoded one,
// grown for mod rows and centred with its shadow in the native canvas.
func gameMenuPanelRectFor(s gameMenuSurface, rows []gameMenuRow, at ...menuPlacer) image.Rectangle {
	pl := placementOf(at)
	p := pl.panel(s)
	if mods := gameMenuModRows(rows); mods > 0 {
		need := gameMenuRowTop + gameMenuRowPitch*(gameMenuBaseRows(s)+mods) + gameMenuBottomMargin
		if need > p.Dy() {
			p.Max.Y = p.Min.Y + need
			if pl.centred {
				box := windowFrame(p, nil).rect()
				p = p.Add(image.Pt(0, (480-box.Dy())/2-box.Min.Y))
			}
		}
	}
	return p
}

// gameMenuPanelRect is the panel's rectangle in frame coordinates.
func gameMenuPanelRect(s gameMenuSurface, at ...menuPlacer) image.Rectangle {
	return placementOf(at).panel(s)
}

// gameMenuRowRect places row n inside the panel, counting from zero.
func gameMenuRowRect(p image.Rectangle, n int) image.Rectangle {
	top := p.Min.Y + gameMenuRowTop + gameMenuRowPitch*n
	return image.Rect(p.Min.X+gameMenuRowLeft, top,
		p.Max.X-gameMenuRowInsetR, top+gameMenuRowPitch)
}

// gameMenuRowAt reports which row of surface s the frame position p falls on.
//
// BOTH AXES DECIDE, unlike the map list's own hit test. A row here is a drawn
// rectangle inside a panel with a border and a margin, so an x outside it is
// visibly not on the row.
func gameMenuRowAt(panel image.Rectangle, rows int, p image.Point) (int, bool) {
	for n := 0; n < rows; n++ {
		if p.In(gameMenuRowRect(panel, n)) {
			return n, true
		}
	}
	return 0, false
}

// gameMenuAccelerator resolves a row's accelerator (MENU-KEY-013, High).
//
// The decoded routine stores the passed immediate, then walks the label from
// index 1: `~~` is an escape and skips two, and otherwise the first
// character preceded by a single `~` replaces the stored one. This is that
// walk, and it runs over the row's DRAWN label whichever source that label
// came from — this build's own English or the install's `dialogs.txt`
// line.
//
// selector IS THE INSTALL'S OWN LANGUAGE SELECTOR (1 for Russian, 0 for
// English and for every caller with no install font; its locale record
// decides the fold — see gameMenuLower).
//
// TWO RESIDUES REMAIN, both authored and both DIV rows rather than code paths
// here: DIV-140 is which physical key a given rune comes from, which this
// package never decides (the OS keyboard layout does, exactly as EditName
// already assumes); DIV-141 is that two RU labels on the town surface fold to
// the same byte (0x82, `В`), so the second is reachable by pointer only — the
// first match in row order wins, the same rule an ASCII duplicate would meet.
func gameMenuAccelerator(label string, fallback byte, selector int) byte {
	for i := 1; i < len(label); i++ {
		if label[i-1] != '~' {
			continue
		}
		if label[i] == '~' {
			i++ // the escape consumes both, so neither can mark the next byte
			continue
		}
		return gameMenuLower(label[i], selector)
	}
	return gameMenuLower(fallback, selector)
}

// gameMenuLower is the decoded routine's own lowercase step (MENU-KEY-013):
// ASCII 'A'-'Z' always, and — when the selector's locale writes code page
// 866 — CP866 uppercase Cyrillic as well: +0x20 over 0x80..0x8f, +0x50 over
// 0x90..0x9f. Every other byte, and every other selector, is unchanged; a
// byte already in the CP866 lowercase ranges
// (0xa0..0xaf, 0xe0..0xef) is therefore also unchanged, since none of the
// original's own fold clauses reach it.
func gameMenuLower(c byte, selector int) byte {
	c = lowerASCII(c)
	if locale.CodePageOf(selector) != 866 {
		return c
	}
	switch {
	case c >= 0x80 && c <= 0x8f:
		return c + 0x20
	case c >= 0x90 && c <= 0x9f:
		return c + 0x50
	}
	return c
}

// gameMenuLabelText is the label as drawn: the accelerator marks removed,
// and `~~` collapsed to the one literal `~` it stands for (AC-15).
func gameMenuLabelText(label string) string {
	if !strings.Contains(label, "~") {
		return label
	}
	var b strings.Builder
	b.Grow(len(label))
	for i := 0; i < len(label); i++ {
		if label[i] != '~' {
			b.WriteByte(label[i])
			continue
		}
		if i+1 < len(label) && label[i+1] == '~' {
			b.WriteByte('~')
			i++
		}
	}
	return b.String()
}

// gameMenuAcceleratorColumn is the index, in the DRAWN text, of the character
// the accelerator marks, or -1 when the label marks none. The paint uses it to
// underline that character; a test can assert it without a window.
func gameMenuAcceleratorColumn(label string) int {
	col := 0
	for i := 0; i < len(label); i++ {
		if label[i] != '~' {
			col++
			continue
		}
		if i+1 < len(label) && label[i+1] == '~' {
			col++ // the escape draws one literal `~`
			i++
			continue
		}
		if i+1 < len(label) {
			return col
		}
	}
	return -1
}

func lowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// gameMenuPickerRows is what the row list shows: the drawn label, and
// whether the row can be chosen. A disabled row is LISTED.
func gameMenuPickerRows(rows []gameMenuRow) []PickerRow {
	out := make([]PickerRow, len(rows))
	for i, r := range rows {
		out[i] = PickerRow{Text: r.text(), Choosable: r.Enabled}
	}
	return out
}

func literalMenuRow(s string) gameMenuRow {
	return gameMenuRow{Label: s, Literal: true}
}

func pageReturnRow() gameMenuRow {
	return gameMenuRow{Label: "~RETURN TO MENU", Fallback: 'R', Action: gameMenuPageReturn, Enabled: true}
}

// wrapGameMenuText produces rows that fit the menu's 252-pixel text area on
// the fixed-width fallback path. The install font is narrower for the shipped
// prose used here. The bound remains conservative on that path.
func wrapGameMenuText(s string) []string {
	const cols = 38
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var out []string
	line := words[0]
	for _, word := range words[1:] {
		if len([]rune(line))+1+len([]rune(word)) <= cols {
			line += " " + word
			continue
		}
		out = append(out, line)
		line = word
	}
	return append(out, line)
}

// The panel's own colours. They are this project's, not the original's frame art
// (spec AU-1), and they are the dialogue box's values so the two popups read as
// one family.
var (
	gameMenuFill      = color.RGBA{R: 0x18, G: 0x10, B: 0x12, A: 0xf2}
	gameMenuBorder    = color.RGBA{R: 0xc8, G: 0x9a, B: 0x3a, A: 0xff}
	gameMenuFocusFill = color.RGBA{R: 0x3c, G: 0x2c, B: 0x18, A: 0xff}
	gameMenuDisabled  = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x90}
)

// gameMenuTextY is where a row's text starts inside its 30-px rect: the debug
// font's cell is 16 high, so 7 centres it.
const gameMenuTextY = (gameMenuRowPitch - pickerLine) / 2

// gameMenuTextX is where it starts across: one debug-font cell in from the row's
// own left edge.
const gameMenuTextX = 6

// pickerAdvance is the debug font's fixed cell width, which is the same number
// pickerCols is derived from. It positions the accelerator underline on the
// no-font path only; the install font is proportional and measures its own.
const pickerAdvance = 6

// gameMenuText is what the install font draws a row in. It is the notice box's
// letter colour, so the two popups read as one family (0158 spec AU-1); the
// debug-font path takes no colour and draws white.
var gameMenuText = color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff}

// These three calls are the menu painter's vector boundary. Tests replace
// them with recorders so the destination rectangles are observed at the
// production use site. Production keeps the direct ebiten vector calls.
var (
	fillGameMenuRect   = vector.DrawFilledRect
	strokeGameMenuRect = vector.StrokeRect
	strokeGameMenuLine = vector.StrokeLine
	drawGameMenuLabel  = drawMenuLabel
)

// drawMenuLabel draws one row's text with the install font.
//
// IT COMPOSES INTO AN RGBA AND UPLOADS IT, because text.Font paints pixels and
// an ebiten.Image is not one. The image is the string's own measured box and no
// larger, and at most seven are made per painted frame — the viewer already
// uploads a whole 640x480 frame every frame, so this is small beside what the
// same frame already costs.
func drawMenuLabel(dst *ebiten.Image, f *text.Font, s string, x, y int, c color.RGBA) {
	w, h := f.Measure(s)
	if w <= 0 || h <= 0 {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	f.Draw(img, s, 0, 0, c)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(ebiten.NewImageFromImage(img), op)
}

// paintGameMenu draws the panel and its rows into dst at FRAME COORDINATES.
// It takes a destination because the two surfaces are composed by two
// different paths — the town through the 640x480 canvas, the map screen
// through an overlay placed over the window — and neither may own the
// paint.
//
// It draws nothing outside the panel rect: the dim behind it is the
// caller's, on both paths.
func paintGameMenu(dst *ebiten.Image, f *text.Font, s gameMenuSurface, rows []gameMenuRow, list *Picker, at menuPlacer, logs ...*pixelLog) {
	if dst == nil || list == nil || len(rows) == 0 {
		return
	}
	p := gameMenuPanelRectFor(s, rows, at)
	var log *pixelLog
	if len(logs) > 0 {
		log = logs[0]
	}
	fillGameMenuRect(dst, float32(p.Min.X), float32(p.Min.Y),
		float32(p.Dx()), float32(p.Dy()), gameMenuFill, false)
	if log != nil {
		log.overSolid(p, gameMenuFill)
	}
	strokeGameMenuRect(dst, float32(p.Min.X)+0.5, float32(p.Min.Y)+0.5,
		float32(p.Dx())-1, float32(p.Dy())-1, 1, gameMenuBorder, false)

	sel := list.Selection()
	top, count := list.Visible()
	if top+count > len(rows) {
		count = len(rows) - top
	}
	for slot := 0; slot < count; slot++ {
		n := top + slot
		row := rows[n]
		r := gameMenuRowRect(p, slot)
		if n == sel {
			fillGameMenuRect(dst, float32(r.Min.X), float32(r.Min.Y),
				float32(r.Dx()), float32(r.Dy()), gameMenuFocusFill, false)
			if log != nil {
				log.overSolid(r, gameMenuFocusFill)
			}
		}
		start := markCapture()
		label := row.text()
		tx, ty := r.Min.X+gameMenuTextX, r.Min.Y+gameMenuTextY
		col := -1
		if !row.Literal {
			col = gameMenuAcceleratorColumn(row.Label)
		}
		ux, uw := tx+col*pickerAdvance, pickerAdvance-1
		uy := ty + pickerLine - 3
		if f == nil {
			drawDebugText(dst, log, label, tx, ty)
		} else {
			// THE INSTALL'S FONT DRAWS THE INSTALL'S BYTES. The debug font is ASCII,
			// so a Russian label drawn with it is a row of replacement glyphs — the
			// words would resolve and nobody could read them. This font carries the
			// install's language selector, which is where CP866 is converted
			// (TEXT-DOM-010).
			ty = r.Min.Y + (gameMenuRowPitch-f.Height())/2
			drawGameMenuLabel(dst, f, label, tx, ty, gameMenuText)
			// The underline moves with the font: the mark's column is a BYTE
			// index into the drawn label, and this font is byte-indexed too, so
			// the pen offset of that byte is the advance of everything before
			// it. A column past the label's end marks nothing and is dropped
			// below with every other unmarked row.
			if col >= 0 && col < len(label) {
				ux, uw = tx+f.Advance(label[:col]), f.Advance(label[col:col+1])
			} else {
				uw = 0
			}
			uy = ty + f.Height()
		}

		// The accelerator, underlined where the label marks one.
		if col >= 0 && uw > 0 {
			strokeGameMenuLine(dst, float32(ux), float32(uy), float32(ux+uw), float32(uy), 1, gameMenuBorder, false)
			if log != nil {
				log.unknown(image.Rect(ux, uy-1, ux+uw+1, uy+2))
			}
		}

		// A DISABLED ROW IS DARKENED RATHER THAN RECOLOURED. The debug font draws
		// in one colour and takes no other, so the row's own rectangle carries the
		// state instead of its glyphs.
		if !row.Enabled && !row.Status {
			fillGameMenuRect(dst, float32(r.Min.X), float32(r.Min.Y),
				float32(r.Dx()), float32(r.Dy()), gameMenuDisabled, false)
			text.TintSince(start, r, gameMenuDisabled)
			if log != nil {
				log.overSolid(r, gameMenuDisabled)
			}
		}
	}
}
