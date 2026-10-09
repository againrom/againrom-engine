package ui

import (
	"image"
	"image/color"
	"math"
	"strings"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

// The notice: one box of text drawn over the map screen.
//
// THERE IS ONE MECHANISM AND NOT TWO. A mission's dialogue and a mission's
// ending are both a box of text over the map with the same three ways to
// dismiss it, and in the original the ending reuses the script's own window
// message with a sentinel value. So this file holds one layout type, one
// wrap, one composition and one input path; the kinds differ in their
// geometry, their colours and their words, all of which are values of the
// layout below.
//
// EVERYTHING HERE IS A FUNCTION OF ITS ARGUMENTS, exactly as the panel's
// composition is: nothing reads viewer state, a clock, a file or a device,
// and the picture is composed into a plain in-memory image. What is on
// screen and what a test asserts are then the same pixels.
//
// NOTHING HERE NAMES A SIMULATION TYPE. What crosses into this package is a
// string and a kind; what crosses out is a destination and a message. That is
// the same rule every other seam out of this tier keeps.

// NoticeKind is which notice surface a box is. It is an integer and not a
// pair of booleans so that the seam carries one value, and the zero value is the
// dialogue because that is the ordinary case.
type NoticeKind uint8

const (
	// NoticeDialogue is one part of a mission's event text.
	NoticeDialogue NoticeKind = 0
	// NoticeOutcome is the legacy single-button outcome surface. Campaign
	// outcomes use the explicit success and failure kinds below.
	NoticeOutcome NoticeKind = 1
	// NoticeSuccess is the campaign-success panel. Unlike a failure outcome,
	// it offers the two decoded choices Victory and Continue.
	NoticeSuccess NoticeKind = 2
	// NoticeFailure is terminal campaign loss (MISSION-DEFEAT-046).
	NoticeFailure NoticeKind = 3
)

// NoticeStyle is how a layout is painted. The zero value is the plain notice
// every fixture and the outcome surfaces use; the dialogue style is the
// original's dialogue panel.
type NoticeStyle uint8

const (
	// NoticeStylePlain paints a filled, bordered box with ragged text and a
	// filled, bordered button.
	NoticeStylePlain NoticeStyle = 0
	// NoticeStyleDialogue paints the dialogue panel: the nine-piece frame with
	// its shadow band, the portrait surface, the bevelled button and text laid
	// out by paragraph, justified and drawn with a flat shadow (`DLG-PANEL-035`,
	// `DLG-PORTRAIT-036`, `DLG-BUTTON-039`, `DLG-LINE-038`).
	NoticeStyleDialogue NoticeStyle = 1
)

// NoticeAction is the player's decision at an open notice. NoticeAdvance is
// the single-button dialogue/failure action and is also the zero-value default
// used by older headless callers. A success panel resolves that default to its
// initially focused Victory button.
type NoticeAction uint8

const (
	NoticeAdvance NoticeAction = iota
	NoticeVictory
	NoticeContinue
	NoticeExitMain
	NoticeLoadGame
	// NoticeAbandon leaves the running mission without victory for the town it
	// was entered from. A mod's in-game menu entry asks for it.
	NoticeAbandon
	// NoticeRestart reopens the running mission from its entry point.
	NoticeRestart
)

// NoticeDest is where advancing a notice sends the front-end. It is the
// integer half of the advance seam.
//
// The zero value is "stay", which is what an advance that only paged a dialogue
// answers, so a seam that decides nothing cannot accidentally navigate.
type NoticeDest int

const (
	// NoticeStay leaves the map screen exactly where it is.
	NoticeStay      NoticeDest = 0
	NoticeToMenu    NoticeDest = 1
	NoticeToMapList NoticeDest = 2
	// NoticeToMission opens a successor mission directly onto its own map
	// screen — the campaign's own advance. It is APPENDED rather than
	// inserted, for MapAdvance's own reason: no existing value's meaning moves.
	//
	// THE PAYLOAD IS AN OPENER AND NOT A NUMBER. This package cannot name a
	// mission — MapLoader is keyed by row index, not by number — so what
	// crosses beside this destination is the MapOpener the far side already
	// built for it: the same shape OpenMission and the generation screen's own
	// door already accept, rather than a second loader keyed by mission with
	// its own rules about what it opens with. A nil opener or one that fails is
	// answered on the map list, never on a torn-down map screen; this value
	// alone never says which.
	NoticeToMission NoticeDest = 3

	// NoticeToTown returns to the town — where a mission ends once the
	// campaign has reached one. It is APPENDED rather than inserted, for the
	// reason NoticeToMission was: no existing value's meaning moves.
	//
	// IT CARRIES NO PAYLOAD BEYOND THE SENTENCE. The town is a screen the
	// front-end already holds — installed once, and standing whether a
	// mission is running or not — so unlike a successor's map there is
	// nothing to open and no opener to carry. The MapOpener beside it is
	// nil on this destination and is not read.
	//
	// A completed campaign mission can return here. Failure cannot.
	NoticeToTown NoticeDest = 4
	// NoticeToLoad keeps the terminal mission behind the save selection panel.
	NoticeToLoad   NoticeDest = 5
	NoticeToEnding NoticeDest = 6
)

// noticeBreak reports whether b ends a word for the wrap.
//
// SPACE IS THE CLAIM AND THE REST IS OURS. `DLG-WRAP-009` fixes the rect, the
// pitch and the clamp; the break policy is this project's (provenance, "Ours by
// choice"). Authored text carries its own line breaks and this atlas has no
// record for one — every byte below 0x20 selects the space glyph through the
// subscript rule — so a byte that DRAWS as a space and does not BREAK as one
// would turn a two-line authored paragraph into a single unbreakable word.
// Treating the four ASCII layout bytes as the break class is the smallest rule
// that avoids it, and it is ours and disclosed rather than claimed.
func noticeBreak(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

// NoticeLayout is the WHOLE of what a notice looks like: its box in the design
// space, the rectangles inside it, the words on its controls, its line pitch
// and its colours. The composition below reads its appearance from this value
// and from nothing else.
//
// THE GEOMETRY IS IN THE DESIGN SPACE and not in window pixels. The map
// screen draws straight to the window rather than through the menu's
// letterboxed canvas, so the authored 640x480 rectangle is a DESIGN
// rectangle: taken as pixels at the shipped startup scale it would be a
// quarter-size box in a corner instead of the centred panel it is. The
// composition happens at the authored size and the placement scales it,
// which is the one place the two spaces meet.
//
// It is the substitution point the panel's own layout is: the original's window
// art is shipped bitmap this project does not read, so the frame painted here is
// ours and replacing it later is supplying a different value of this type.
type NoticeLayout struct {
	Frame *DialogFrame
	// Style selects how the layout is painted and how its words are laid out.
	Style              NoticeStyle
	DialogueBackdrop   DialogueBackdrop
	DialogueArithmetic DialogueArithmetic
	ButtonState        DialogueButtonState
	// Box is the notice's rectangle in the design space.
	Box image.Rectangle

	// Text and Button are RELATIVE TO Box's origin, which is how the claim
	// states them: the window's child rectangles are panel-relative.
	Text   image.Rectangle
	Button image.Rectangle
	// SecondaryButton is the second control on the campaign-success panel.
	// It is empty on dialogue and failure notices.
	SecondaryButton image.Rectangle
	// SecondaryState is the second button's pointer state.
	SecondaryState DialogueButtonState

	// Portrait is the pane the speaker's face stands in, and
	// TextBesidePortrait is where the words go when it is there — both
	// panel-relative like the two above.
	//
	// BOTH SHAPES OF ONE WINDOW LIVE IN ONE VALUE, which is the original's own
	// arrangement: one routine holds both text rectangles as literals and picks
	// between them on one flag. WithPortrait is that pick. A second layout value
	// beside this one would be a second substitution point carrying its own copy
	// of the box, the button, the pitch and all six colours — every field the
	// contract says the two shapes AGREE on — and keeping two values in step on
	// the fields they must share is a worse job than branching on two they must
	// not.
	//
	// An EMPTY Portrait is a layout that has no pane at all, which is what makes
	// the outcome notice's value carry no portrait fields rather than carry them
	// switched off.
	Portrait           image.Rectangle
	TextBesidePortrait image.Rectangle

	// PortraitAt selects the request point in the physical dialogue surface.
	// PortraitWindow selects the engine canvas's top-down crop. The dialogue
	// adapter converts it to the native source request before descending rows.
	PortraitAt     image.Point
	PortraitWindow image.Rectangle

	// ButtonLabel is the word on the primary button. Installed success panels
	// resolve it from dialogs.bin; other authored layouts supply their own word.
	ButtonLabel string
	// SecondaryButtonLabel is empty where SecondaryButton is empty.
	SecondaryButtonLabel string
	SecondaryDisabled    bool

	// FirstLine is the index of the first wrapped line drawn, and Scrollbar the
	// rectangle (panel-relative) of the scroll bar beside the text, empty for a
	// panel that does not scroll. Only the help panel sets either.
	FirstLine int
	Scrollbar image.Rectangle
	// ScrollbarTopHot and ScrollbarBottomHot are the bar's endcap states.
	ScrollbarTopHot, ScrollbarBottomHot bool

	// Ink is the body text colour when its alpha is nonzero; the dialogue ink
	// otherwise. Only the help panel sets it.
	Ink color.RGBA

	// Pitch is added to the font's height to give the line pitch. The claim's
	// default is 2.
	Pitch int

	Fill         color.RGBA
	Border       color.RGBA
	TextColor    color.RGBA
	ButtonFill   color.RGBA
	ButtonBorder color.RGBA
	ButtonText   color.RGBA

	// PortraitFill and PortraitBorder are what an EMPTY pane is painted with.
	// They are OURS, exactly as the frame and the button's colours are; the
	// installed backdrop (Frame.PortraitBack) covers them under the window.
	//
	// The pane is painted rather than left as window fill so that "no picture
	// could be found for this speaker" and "the pane was never built" look
	// different on screen. They are the same picture otherwise, and the second
	// is a defect.
	PortraitFill   color.RGBA
	PortraitBorder color.RGBA
}

func (l NoticeLayout) WithDialogueButtonState(state DialogueButtonState) NoticeLayout {
	l.ButtonState = state
	return l
}

func (l NoticeLayout) WithDialogueBackdrop(policy DialogueBackdrop) NoticeLayout {
	l.DialogueBackdrop = policy
	return l
}

// WithPortrait is l resolved for one of the window's two shapes: the text
// rectangle set to the applicable one, and the pane emptied when there is not
// one.
//
// IT IS A VALUE AND NOT A MUTATION. The caller holds the layout that carries
// both shapes and gets back the one this window is; nothing about the stored
// layout changes, so two viewers showing opposite shapes cannot reach each
// other.
//
// Asking for a pane of a layout that HAS none — the outcome notice's — yields
// the layout unchanged rather than a window with an empty text rectangle. That
// is what makes "this kind of notice has no portrait" a property of its own
// value instead of a test at every reader.
func (l NoticeLayout) WithPortrait(on bool) NoticeLayout {
	if !on || l.Portrait.Empty() {
		l.Portrait = image.Rectangle{}
		return l
	}
	if !l.TextBesidePortrait.Empty() {
		l.Text = l.TextBesidePortrait
	}
	return l
}

// WithFaceWindow is l resolved for ONE SPEAKER: the window cut from that
// speaker's picture replaced by w, or left at the layout's own default when w
// has no area.
//
// IT IS WithPortrait'S OWN SHAPE for the second thing that varies per notice,
// and for the same reason: the stored layout carries what every speaker shares,
// the caller gets back the one this window is, and nothing about the stored
// value changes. Which of the two applies is decided by the RECORD — 48 of the
// 105 shipped ones state a window and the rest take the engine's default
// (`REG-NPC-089`) — so "this speaker states none" arrives here as the zero
// rectangle and needs no second field to say so.
func (l NoticeLayout) WithFaceWindow(w image.Rectangle) NoticeLayout {
	if w.Dx() > 0 && w.Dy() > 0 {
		l.PortraitWindow = w
	}
	return l
}

// The dialogue notice's geometry, in the design space at 640x480.
//
// THE PANEL IS DRAWN 488x232 AT (76,124), NOT AT THE 580x240 RECTANGLE ITS
// CONSTRUCTOR IS GIVEN: the constructor keeps only that rectangle's size, snaps
// it to the frame's tile grid and centres it on the screen (`DLG-PANEL-035`).
// The box is the drawn size. The frame body is its top-left 480x224 and the 8 px
// to the right and below are the shadow band.
//
// The child rectangles are panel-relative and are the constructor's own
// literals (`DLG-RECT-037`). THERE ARE TWO TEXT RECTANGLES AND NOT ONE: the
// text control is `128,36` WITH the portrait and `48,36` WITHOUT it, 300 and 380
// wide. Both are 135 tall, not the constructor's 136: the control recomputes its
// height from the 15 px font as seven 19 px rows plus 2, and at the 17 px pitch
// that shows seven lines. On both preserved roots every event file mentions a
// speaker, so the original always takes the portrait branch. The portrait child
// is `30,54-118,168` and the button `200,172-280,198`, the same in both shapes.
//
// They are written as origin-and-extent rather than as four edges because that
// is the form the composition uses, and the arithmetic is done once, here,
// rather than at the call sites.
const (
	noticeBoxX, noticeBoxY = 76, 124
	noticeBoxW, noticeBoxH = 488, 232

	// noticeShadow is the width of the shadow band right of and below the body.
	noticeShadow = 8

	noticeTextX, noticeTextY = 48, 36
	noticeTextW, noticeTextH = 380, 135

	noticeFaceTextX, noticeFaceTextY = 128, 36
	noticeFaceTextW, noticeFaceTextH = 300, 135

	noticePortraitX, noticePortraitY = 30, 54
	noticePortraitW, noticePortraitH = 88, 114

	noticeButtonX, noticeButtonY = 200, 172
	noticeButtonW, noticeButtonH = 80, 26

	// noticePitch is the claim's own default: the font's height plus two.
	noticePitch = 2
)

// The engine canvas crop maps to DIALOGUE-064's source request. Canvas
// construction and native exposed rows remain Unknown (DIV-1622).
const (
	noticeFaceAtX, noticeFaceAtY = 8, 7
	noticeFaceW, noticeFaceH     = 72, 96

	noticeFaceDefX, noticeFaceDefY = 36, 8
	noticeFaceDefH                 = 92
)

// NoticeFaceWindow is the window to cut from a speaker's picture when that
// speaker's own record STATES one: 72x96 with its top-left corner at (x, y) in
// the picture's own top-down pixels (`REG-NPC-089`, `REG-NPC-091`).
//
// IT IS EXPORTED BECAUSE THE TWO HALVES LIVE IN TWO TIERS. The origin is a
// registry value and reaches this package through ui.Dialogue; the 72x96 is an
// engine constant about this window and belongs beside the pane it is cut for,
// not beside the registry reader. A caller holding a record that states no
// origin hands over the zero rectangle instead, and the layout's own default
// stands.
//
// NOTHING HERE BOUNDS x OR y. `REG-NPC-089` gives the shipped domain as
// 0 <= X1 <= 88 and 0 <= Y1 <= 144 — the range that keeps the window inside a
// 160x240 picture — and a value outside it simply cuts a window that is partly
// or wholly off the picture, which draws as the pane's own ground. That is the
// same reading every other unresolvable address in this tier already gets.
func NoticeFaceWindow(x, y int) image.Rectangle {
	return image.Rect(x, y, x+noticeFaceW, y+noticeFaceH)
}

// AuthoredDialogueLayout is the dialogue notice this project ships: the
// original's dialogue panel, in the dialogue style.
//
// THE GEOMETRY AND THE PAINT ARE THE CLAIMS' where the style draws them
// (`DLG-PANEL-035`, `DLG-RECT-037`, `DLG-BUTTON-039`, `DLG-LINE-038`). The
// colours below are the plain style's and this project's own: the dialogue style
// reads none of them, but the save dialog is drawn from this value and keeps
// them, and a layout with no frame art falls back to the fill and border.
//
// It is handed out as a fresh value rather than held in a package variable, for
// the panel layout's own reason: a variable is writable from anywhere, and "two
// viewers show the same notice" would then be true by mutation rather than by
// design.
func AuthoredDialogueLayout() NoticeLayout {
	return NoticeLayout{
		Style:              NoticeStyleDialogue,
		Box:                image.Rect(noticeBoxX, noticeBoxY, noticeBoxX+noticeBoxW, noticeBoxY+noticeBoxH),
		Text:               image.Rect(noticeTextX, noticeTextY, noticeTextX+noticeTextW, noticeTextY+noticeTextH),
		Button:             image.Rect(noticeButtonX, noticeButtonY, noticeButtonX+noticeButtonW, noticeButtonY+noticeButtonH),
		Portrait:           image.Rect(noticePortraitX, noticePortraitY, noticePortraitX+noticePortraitW, noticePortraitY+noticePortraitH),
		TextBesidePortrait: image.Rect(noticeFaceTextX, noticeFaceTextY, noticeFaceTextX+noticeFaceTextW, noticeFaceTextY+noticeFaceTextH),
		PortraitAt:         image.Pt(noticeFaceAtX, noticeFaceAtY),
		PortraitWindow:     image.Rect(noticeFaceDefX, noticeFaceDefY, noticeFaceDefX+noticeFaceW, noticeFaceDefY+noticeFaceDefH),
		ButtonLabel:        AuthoredNoticeButton,
		Pitch:              noticePitch,

		Fill:           color.RGBA{R: 0x10, G: 0x12, B: 0x18, A: 0xf2},
		Border:         color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff},
		TextColor:      color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff},
		ButtonFill:     color.RGBA{R: 0x1c, G: 0x1f, B: 0x28, A: 0xff},
		ButtonBorder:   color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff},
		ButtonText:     color.RGBA{R: 0xf2, G: 0xe6, B: 0xc4, A: 0xff},
		PortraitFill:   color.RGBA{A: 0xff},
		PortraitBorder: color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff},
	}
}

// The outcome notice's box, in the design space. EVERY NUMBER HERE IS OURS: the
// spec contracts the dialogue window's geometry and explicitly leaves the
// outcome's box size and placement to this project. It is centred horizontally
// in the design space and sits above the middle, so it does not land on the
// dialogue box it replaces.
const (
	outcomeBoxW, outcomeBoxH = 360, 110
	outcomeBoxX              = (frame.W - outcomeBoxW) / 2
	outcomeBoxY              = 150

	outcomeTextX, outcomeTextY = 24, 24
	outcomeTextW, outcomeTextH = outcomeBoxW - 2*outcomeTextX, 34

	outcomeButtonW, outcomeButtonH = 80, 26
	outcomeButtonX                 = (outcomeBoxW - outcomeButtonW) / 2
	outcomeButtonY                 = outcomeBoxH - outcomeButtonH - 16
)

// AuthoredOutcomeLayout is the outcome notice this project ships.
func AuthoredOutcomeLayout() NoticeLayout {
	return NoticeLayout{
		Box:         image.Rect(outcomeBoxX, outcomeBoxY, outcomeBoxX+outcomeBoxW, outcomeBoxY+outcomeBoxH),
		Text:        image.Rect(outcomeTextX, outcomeTextY, outcomeTextX+outcomeTextW, outcomeTextY+outcomeTextH),
		Button:      image.Rect(outcomeButtonX, outcomeButtonY, outcomeButtonX+outcomeButtonW, outcomeButtonY+outcomeButtonH),
		ButtonLabel: AuthoredNoticeButton,
		Pitch:       noticePitch,

		Fill:         color.RGBA{R: 0x18, G: 0x10, B: 0x12, A: 0xf2},
		Border:       color.RGBA{R: 0xc8, G: 0x9a, B: 0x3a, A: 0xff},
		TextColor:    color.RGBA{R: 0xff, G: 0xf0, B: 0xcc, A: 0xff},
		ButtonFill:   color.RGBA{R: 0x28, G: 0x1c, B: 0x1f, A: 0xff},
		ButtonBorder: color.RGBA{R: 0xc8, G: 0x9a, B: 0x3a, A: 0xff},
		ButtonText:   color.RGBA{R: 0xff, G: 0xf0, B: 0xcc, A: 0xff},
	}
}

// AuthoredSuccessLayout carries the compiled campaign-success geometry from
// MISSION-VICTORY-029. Box is in the 640x480 design space; its controls are
// panel-relative, as every NoticeLayout child rectangle is. The text rectangle
// and paint remain authored because the claim does not establish their pixels.
func AuthoredSuccessLayout() NoticeLayout {
	l := AuthoredOutcomeLayout()
	l.Box = image.Rect(128, 200, 512, 380)
	l.Text = image.Rect(48, 40, 336, 72)
	l.Button = image.Rect(48, 84, 336, 108)
	l.SecondaryButton = image.Rect(48, 108, 336, 132)
	l.ButtonLabel = AuthoredWords().MenuVictory
	l.SecondaryButtonLabel = AuthoredWords().OutcomeContinue
	return l
}

// AuthoredFailureLayout uses the decoded failure choices. Geometry and paint
// remain authored; MISSION-DEFEAT-046 establishes labels and dispatch only.
func AuthoredFailureLayout() NoticeLayout {
	l := AuthoredSuccessLayout()
	l.ButtonLabel = AuthoredWords().MenuExitMain
	l.SecondaryButtonLabel = AuthoredWords().MenuLoad
	l.SecondaryDisabled = true
	return l
}

// AuthoredNoticeBackdrop is the alpha wash for menus, outcomes and custom notices.
func AuthoredNoticeBackdrop() color.RGBA {
	return color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x30}
}

// SetNoticeBackdrop replaces the dim drawn over the map behind a notice.
func (v *Viewer) SetNoticeBackdrop(c color.RGBA) { v.noticeBackdrop = c; v.noticeBackdropCustom = true }

// noticeBackdrop is the rectangle to darken this frame and the colour to
// darken it with, or false for a frame that draws no dim at all.
//
// It decides EVERYTHING and the draw decides nothing, which is the shape the
// panel, the readout and the notice's own presentation already keep: the
// whole of what the dim's contract requires is then assertable with no
// window anywhere near it.
//
// THE GATE IS THE POPUP PREDICATE, which is the notice's own open test, font
// and all. A viewer that cannot draw a popup must not darken the map either
// — the player would be left looking at a dimmed map with no box on it and
// no way to un-dim it — and reading the predicate rather than the notice
// directly is what makes a popup added later dim the screen by raising one
// answer, with no second condition here to find and extend.
//
// A fully transparent backdrop draws nothing rather than drawing nothing
// visibly: the frame is then byte-identical to the frame before the dim existed,
// instead of merely looking like it.
//
// THE RECTANGLE IS THE WHOLE DRAWABLE AREA. The map screen fills the window, so
// "the map picture" and "the view" are the same rectangle here; taking it from
// the camera rather than from the notice's letterboxed placement is deliberate,
// because the letterbox is where the BOX goes and the dim is not in the design
// space at all.
func (v *Viewer) noticeBackdropOf() (image.Rectangle, color.RGBA, bool) {
	if !v.popupOpen() || v.noticeBackdrop.A == 0 {
		return image.Rectangle{}, color.RGBA{}, false
	}
	if v.cam.ViewW <= 0 || v.cam.ViewH <= 0 {
		return image.Rectangle{}, color.RGBA{}, false
	}
	return image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH), v.noticeBackdrop, true
}

// NoticeLines is s broken to width, in the font's own advance.
//
// THE RULE, and it has three parts that must be read together:
//
//   - a line is broken at the LAST BREAK BYTE THAT FITS;
//   - a single word wider than the whole width is broken WITHIN the word, at the
//     last byte that fits, because there is no other way to show it at all;
//   - the break byte itself is dropped, so no line begins with one.
//
// IT MEASURES WITH THE FONT'S ADVANCE and never with a character count. The
// atlas is proportional — every record carries its own pen advance — so a count
// would wrap short on a line of narrow letters and overflow on one of wide ones,
// and it would do it differently on either language's atlas.
//
// It is not clamped here. How many of these lines are DRAWN is the area's
// business and is decided by NoticeLayoutOf, so the count this returns is the
// count the text produced and a caller can see how much did not fit.
//
// A nil font, a font with no records, and a non-positive width all yield no
// lines: there is no advance to measure with and no width to measure against.
func NoticeLines(f *text.Font, s string, width int) []string {
	if f == nil || len(f.Glyphs) == 0 || width <= 0 || s == "" {
		return nil
	}
	var out []string
	i := 0
	for {
		// A line never BEGINS with a break byte: leading ones are consumed, so
		// no line is indented by whatever the author happened to type between
		// two words.
		for i < len(s) && noticeBreak(s[i]) {
			i++
		}
		if i >= len(s) {
			return out
		}
		// The longest prefix of the remainder that fits, and the last break byte
		// inside it. Both are found in ONE forward walk over the bytes, adding
		// each byte's own advance, so the measurement here and the pen the draw
		// uses are the same arithmetic — a second measurement through Measure
		// would be a box rather than a pen and would disagree on a glyph whose
		// ink overhangs its advance.
		pen, fit, brk := 0, i, -1
		for j := i; j < len(s); j++ {
			pen += f.Advance(s[j : j+1])
			if pen > width {
				break
			}
			fit = j + 1
			if noticeBreak(s[j]) {
				brk = j
			}
		}
		if fit >= len(s) {
			return append(out, s[i:])
		}

		// A cut is needed. THREE CASES, and only the third breaks a word:
		//
		//   - the byte that did not fit is itself a break, so the last word ends
		//     exactly where the line does and nothing is backed over;
		//   - a break byte lies inside what fits, so the line ends at the LAST
		//     one — the last space that fits;
		//   - there is none, so ONE WORD does not fit alone and is cut where it
		//     stopped fitting.
		end := fit
		if !noticeBreak(s[fit]) {
			if brk > i {
				end = brk
			} else if fit <= i {
				// Narrower than the first glyph. At least one byte is always
				// taken, or this would not terminate.
				end = i + 1
			}
		}
		// Trailing break bytes are not part of the line: they would draw as
		// spaces hanging off its right edge.
		for end > i && noticeBreak(s[end-1]) {
			end--
		}
		if end <= i {
			end = i + 1
		}
		out = append(out, s[i:end])
		i = end
	}
}

// NoticeLayoutOf is the notice's picture as a list of lines, ALREADY CLAMPED
// to what its text area holds.
//
// THE CLAMP IS THE BEHAVIOUR AND NOT A SIMPLIFICATION OF IT. The original's
// control can scroll and this window is given no scrollbar and no arm answering
// one, so the lines past the area are simply not drawn: `min(height/pitch, line
// count)` is the claim's own formula and it is written here once.
func NoticeLayoutOf(l NoticeLayout, f *text.Font, s string) []string {
	pitch := f.Height() + l.Pitch
	if pitch <= 0 {
		return nil
	}
	lines := noticeWrap(l, f, s)
	if n := l.Text.Dy() / pitch; n < len(lines) {
		if n < 0 {
			n = 0
		}
		lines = lines[:n]
	}
	return lines
}

// noticeWrap is the lines l breaks s into at its text width, before any clamp:
// the dialogue style's paragraph wrap, or the plain wrap of NoticeLines. Every
// consumer that counts lines (the draw, the clamp, the pager) goes through it, so
// they cannot disagree about where a line ends.
func noticeWrap(l NoticeLayout, f *text.Font, s string) []string {
	if l.Style != NoticeStyleDialogue {
		return NoticeLines(f, s, l.Text.Dx())
	}
	lines := dialogueWrap(f, s, l.Text.Dx())
	if len(lines) == 0 {
		return nil
	}
	out := make([]string, len(lines))
	for i, ln := range lines {
		out[i] = strings.TrimRight(ln.visible(), " \t")
	}
	return out
}

// RenderNotice composes a notice and returns it as a plain image, or nil when
// there is nothing to compose: no font, a font holding no record, or a box with
// no area.
//
// It is the exported face of the seam, exactly as RenderPanel is: given a
// layout, a font, a string and a picture it is a function of its arguments, with
// no viewer, window or graphics context anywhere near it.
//
// THE LAYOUT ARRIVES ALREADY RESOLVED. Which of the window's two shapes this is
// was settled by WithPortrait before the call, so nothing here tests for a
// speaker, a kind or a file: a pane is drawn when the layout has one, and that
// is the whole of the branch.
//
// face is the picture to stand in the pane, or nil for a pane with nothing in
// it. A face handed to a layout with no pane is drawn nowhere, because there is
// nowhere to draw it.
//
// NOTHING PAINTED LEAVES THE BOX, and no rectangle arithmetic of ours enforces
// that: the destination IS the box, so the font's own bounds test is what drops
// a line a layout placed outside it, and the pane's own draw is clipped by the
// destination it is drawn into.
func RenderNotice(l NoticeLayout, f *text.Font, s string, face *image.RGBA) *image.RGBA {
	if f == nil || f.Height() <= 0 {
		return nil
	}
	box := l.Box.Size()
	if box.X <= 0 || box.Y <= 0 {
		return nil
	}
	if l.Style == NoticeStyleDialogue {
		return renderDialogueNotice(l, f, s, face)
	}
	img := image.NewRGBA(image.Rect(0, 0, box.X, box.Y))
	if l.Frame.valid() {
		l.Frame.Draw(img, img.Bounds())
	} else {
		fillPanelFrame(img, box, l.Fill, l.Border)
	}

	if p := l.Portrait; p.Dx() > 0 && p.Dy() > 0 {
		drawNoticePortrait(img, l, face)
		l.Frame.DrawPortrait(img, l.Portrait)
	}

	if b := l.Button; b.Dx() > 0 && b.Dy() > 0 {
		s := l.ButtonState
		drawPushButton(img, f, pushButton{Rect: b, Label: l.ButtonLabel, Hover: s.Hover, Pressed: s.Pressed,
			Inside: s.Inside, Disabled: s.Disabled, Policy: l.DialogueBackdrop})
	}
	if b := l.SecondaryButton; b.Dx() > 0 && b.Dy() > 0 {
		s := l.SecondaryState
		drawPushButton(img, f, pushButton{Rect: b, Label: l.SecondaryButtonLabel, Hover: s.Hover, Pressed: s.Pressed,
			Inside: s.Inside, Disabled: l.SecondaryDisabled, Policy: l.DialogueBackdrop})
	}

	pitch := f.Height() + l.Pitch
	for i, ln := range NoticeLayoutOf(l, f, s) {
		f.Draw(img, ln, l.Text.Min.X, l.Text.Min.Y+i*pitch, l.TextColor)
	}
	return img
}

// drawNoticePortrait paints the speaker's pane inside the composed box: the
// pane's own frame first, then whatever picture was supplied, centred in it.
//
// THE FRAME IS PAINTED WHETHER OR NOT THERE IS A PICTURE, which is the whole of
// the difference between "this speaker's face could not be found" and "this
// window has no pane". Both would otherwise be a window with words and a button
// and nothing where the face goes, and only one of them is correct.
//
// THE PICTURE IS CLIPPED, never scaled. A face larger than the pane is cropped
// to it rather than resampled: this seam has no resampling rule of its own, and
// inventing one here would put the quality of a picture into the composition.
// Every pixel of the pane's frame outside the picture stays painted, so an
// undersized face reads as a face in a pane.
//
// IT IS ONE BLIT OF ONE WINDOW, and 0141 first drew it as a centred crop of the
// whole picture. The pane is 88x114 and every picture a speaker resolves to is
// 160x240 (`SPR256-PICT-043`), so something has to be cut, and the cut is the
// engine's own: a fixed 72x96 window taken at the speaker's `PortraitX1`/`Y1`
// and blitted to a fixed point inside the pane (`REG-NPC-089`, `REG-NPC-091`,
// and this file's own constants for what the point is measured from). The window
// arrives already resolved, exactly as the text rectangle does — WithFaceWindow
// settles which speaker's it is before the call, so nothing here reads a record.
//
// A WINDOW OFF THE PICTURE IS CUT, NOT REFUSED. The source is intersected with
// the picture's own bounds and whatever survives is drawn where it would have
// landed, so a partly-off window draws its overlapping part in place and a
// wholly-off one draws nothing — the pane's own ground, which is what every
// unresolvable address in this tier already reads as.
//
// IT COMPOSITES OVER THE PANE'S GROUND AND DOES NOT COPY IT: a composed figure
// is mostly transparent canvas. Under the window the ground is the frame's
// PortraitBack cut at the same window, as the original draws it (DIV-1485);
// the black PortraitFill shows only where no backdrop reaches. The blend is
// drawInventoryPicture's own source-over on premultiplied channels (inventory.go),
// stated a second time here rather than shared, because that one centres its
// picture in an area and this one places a window at a point.
func drawNoticePortrait(dst *image.RGBA, l NoticeLayout, face *image.RGBA) {
	p := l.Portrait
	fill := p
	if l.Frame != nil && l.Frame.Portrait != nil {
		fill = image.Rectangle{Min: p.Min.Add(l.PortraitAt), Max: p.Min.Add(l.PortraitAt).Add(image.Pt(noticeFaceW, noticeFaceH))}.Intersect(p)
	}
	for y := fill.Min.Y; y < fill.Max.Y; y++ {
		for x := fill.Min.X; x < fill.Max.X; x++ {
			c := l.PortraitFill
			if l.Frame == nil && (x == p.Min.X || y == p.Min.Y || x == p.Max.X-1 || y == p.Max.Y-1) {
				c = l.PortraitBorder
			}
			dst.SetRGBA(x, y, c)
		}
	}
	if l.Frame != nil {
		drawNoticeWindow(dst, l, l.Frame.PortraitBack)
	}
	drawNoticeWindow(dst, l, face)
}

// drawNoticeWindow composites the layout's window of pic at the pane's picture
// point, clipped to the pane; a nil pic draws nothing. The backdrop and the
// speaker's picture share it, so both are cut at the same window.
func drawNoticeWindow(dst *image.RGBA, l NoticeLayout, pic *image.RGBA) {
	p := l.Portrait
	blitNoticeWindow(dst, p, p.Min.Add(l.PortraitAt), l.PortraitWindow, pic)
}

// blitNoticeWindow composites win of pic over dst with win's top-left corner at
// origin, keeping only what lands inside clip. win is stated in the picture's own
// top-down pixels and is cut, not refused, where it leaves the picture: what
// survives is drawn where it would have landed. A nil pic draws nothing.
func blitNoticeWindow(dst *image.RGBA, clip image.Rectangle, origin image.Point, win image.Rectangle, pic *image.RGBA) {
	if pic == nil {
		return
	}
	// The window is stated in the PICTURE's own top-down pixels, so it is
	// offset by the picture's own origin before it is intersected: a picture
	// handed over as a sub-image carries a non-zero Min and its window is
	// still measured from its own top-left corner.
	fb := pic.Bounds()
	win = win.Add(fb.Min)
	src := win.Intersect(fb)
	if src.Dx() <= 0 || src.Dy() <= 0 {
		return
	}
	at := origin.Add(src.Min.Sub(win.Min))
	for y := 0; y < src.Dy(); y++ {
		dy := at.Y + y
		if dy < clip.Min.Y || dy >= clip.Max.Y {
			continue
		}
		for x := 0; x < src.Dx(); x++ {
			dx := at.X + x
			if dx < clip.Min.X || dx >= clip.Max.X {
				continue
			}
			so := pic.PixOffset(src.Min.X+x, src.Min.Y+y)
			a := pic.Pix[so+3]
			if a == 0 {
				continue
			}
			do := dst.PixOffset(dx, dy)
			if a == 0xff {
				copy(dst.Pix[do:do+4], pic.Pix[so:so+4])
				continue
			}
			inv := uint32(0xff - a)
			for i := 0; i < 4; i++ {
				dst.Pix[do+i] = uint8(uint32(pic.Pix[so+i]) + uint32(dst.Pix[do+i])*inv/0xff)
			}
		}
	}
}

// noticeKey is everything the presented picture is a function of: the words, the
// kind, and which font composed them. The DESIGN-SPACE geometry is fixed by the
// layout, so — unlike the panel's key — the window's area is NOT in here: a
// resize moves and rescales the picture at the draw and does not recompose it.
type noticeKey struct {
	text     string
	kind     NoticeKind
	serial   int
	portrait bool
	face     int
	button   DialogueButtonState
	second   DialogueButtonState
	policy   DialogueBackdrop
}

// Dialogue is one part of a mission's event text as the map screen shows it:
// the words, which of the window's two shapes this file gets, whether this
// part names a speaker, and that speaker's picture.
//
// SPEAKS AND FACE ARE TWO FIELDS AND NOT ONE NULLABLE PICTURE, because the
// window has two different behaviours that would otherwise arrive identically.
// A part naming NOBODY leaves the standing face alone — that is the original's
// behaviour and not an omission. A part naming SOMEBODY whose picture could not
// be found empties the pane, because leaving the previous face up would put one
// speaker's face on another speaker's words. Both are a nil picture; only
// Speaks tells them apart.
//
// PORTRAIT IS THE FILE'S ANSWER AND NOT THIS PART'S. It is carried on every push
// of the same window rather than once at the open, so the seam has one shape
// instead of an open-then-page pair, and a caller cannot page a window into a
// shape it was not opened in.
//
// Face is borrowed, not copied: the composition reads it during the call it is
// rebuilt on. A caller that mutates a picture it has handed over must say so by
// handing it over again.
//
// FaceWindow is the rectangle to cut from Face, in that picture's own top-down
// pixels — the speaker's `PortraitX1`/`PortraitY1` window (`REG-NPC-089`),
// built by NoticeFaceWindow. The ZERO rectangle is what a speaker whose record
// states no window hands over, and it leaves the layout's own default standing;
// it is written with the face and never separately, so a window can never be
// presented over a picture it was not read for.
type Dialogue struct {
	Text       string
	Portrait   bool
	Speaks     bool
	Face       *image.RGBA
	FaceWindow image.Rectangle
}

// SetDialogue explicitly shows a dialogue; repeating it compounds the backdrop.
//
// IT IS ONE STATEMENT and that is the point of the type. The words, the shape
// and the face are written together or not at all, so a picture cannot be
// presented under a shape it was not composed for and no ordering rule exists
// between two setters for a caller to get wrong.
//
// It writes the face ONLY when the part names a speaker. That single condition
// is the whole of "a part without a speaker leaves the previous face standing",
// and it is here — at the one place the state is written — rather than at the
// driver, so every caller gets it.
func (v *Viewer) SetDialogue(d Dialogue) {
	if !v.noticeOpen || v.noticeKind != NoticeDialogue {
		v.dialogueBackdrop.shows = 0
	}
	v.dialogueBackdrop.shows++
	v.noticeKind, v.noticeOpen = NoticeDialogue, true
	v.PageDialogue(d)
}

// PageDialogue updates an existing page without invoking show again.
func (v *Viewer) PageDialogue(d Dialogue) {
	if !v.noticeOpen || v.noticeKind != NoticeDialogue {
		v.SetDialogue(d)
		return
	}
	v.notice, v.noticeKind, v.noticeOpen = d.Text, NoticeDialogue, true
	v.help = nil
	v.noticeSerial++
	v.noticeButtonState, v.noticeSecondState = DialogueButtonState{}, DialogueButtonState{}
	v.noticePortrait = d.Portrait
	if d.Speaks {
		v.noticeFace, v.noticeFaceWindow = d.Face, d.FaceWindow
		v.noticeFaceSerial++
	}
	v.noticePic = nil
}

// NoticeSpeaker is the shape the open notice was pushed with and the picture
// standing in its pane.
//
// It is NoticeState's counterpart for the half this tier is told about the
// speaker, and it exists for the same reason: the tier that pushes needs to see
// that its value arrived without opening a window.
func (v *Viewer) NoticeSpeaker() (portrait bool, face *image.RGBA) {
	return v.noticePortrait, v.noticeFace
}

// SetNotice opens a notice over the map screen: the words to show and which
// surface they use.
//
// THE VIEWER HOLDS ONLY WHAT IT DRAWS. Which part of an event text this is,
// whether a second announcement was dropped and whether the mission is over are
// the DRIVER'S state and none of it is here — so "a second announcement while
// one is open is discarded" is one test in one place rather than a rule the two
// tiers must agree about.
//
// It writes NOTHING ELSE — no camera, no selection, no clock — and it
// replaces rather than merges, which is what makes an outcome notice
// replacing an open dialogue one statement rather than a teardown and a
// build. IT SETTLES THE SPEAKER TOO, to nothing. This is the plain push —
// the outcome notice's, and any caller with no speaker to state — so it
// opens a window with NO pane and NO face standing. Leaving either as the
// last window left it is how one mission's speaker ends up beside another's
// words, and it would make "a window's shape is decided when it opens" false
// for exactly the paths that do not mention it.
func (v *Viewer) SetNotice(s string, kind NoticeKind) {
	if kind == NoticeDialogue {
		if !v.noticeOpen || v.noticeKind != NoticeDialogue {
			v.dialogueBackdrop.shows = 0
		}
		v.dialogueBackdrop.shows++
	} else {
		v.dialogueBackdrop.shows = 0
	}
	v.notice, v.noticeKind, v.noticeOpen = s, kind, true
	v.help = nil
	v.noticeSerial++
	v.noticeButtonState, v.noticeSecondState = DialogueButtonState{}, DialogueButtonState{}
	v.noticePortrait, v.noticeFace = false, nil
	v.noticeFaceWindow = image.Rectangle{}
	v.noticeFaceSerial++
	v.noticePic = nil
	if kind == NoticeFailure {
		v.noticeTerminal = true
		v.refreshFailureLoad()
	}
}

// The flow supplies the store query; availability is sampled when this panel
// opens, not every paint or pointer event.
func (v *Viewer) setFailureLoadCheck(check func() bool) {
	v.failureLoadCheck = check
	if v.noticeOpen && v.noticeKind == NoticeFailure {
		v.refreshFailureLoad()
	}
}

func (v *Viewer) refreshFailureLoad() {
	v.noticeLayouts[NoticeFailure].SecondaryDisabled = v.failureLoadCheck == nil || !v.failureLoadCheck()
	v.noticePic = nil
}

// ClearNotice closes whatever notice is open. Closing one that is not open is
// nothing at all, which is what lets the driver close unconditionally.
func (v *Viewer) ClearNotice() {
	v.dialogueBackdrop.shows = 0
	v.notice, v.noticeOpen = "", false
	v.help = nil
	v.noticeButtonState, v.noticeSecondState = DialogueButtonState{}, DialogueButtonState{}
	v.noticeTerminal = false
	v.noticePortrait, v.noticeFace = false, nil
	v.noticeFaceWindow = image.Rectangle{}
	v.noticeFaceSerial++
	v.noticePic = nil
}

// NoticeOpen reports whether a notice is TAKING INPUT on the map screen.
//
// Ordinary notices require a font. Terminal failure retains its input gate
// through replacement disclosure pages, even with no font. Return/Escape
// advances those pages back to failure, then exits.
func (v *Viewer) NoticeOpen() bool {
	return v.noticeOpen && (v.font != nil || v.noticeTerminal)
}

// NoticeState is the words and the kind last pushed, and whether one is open at
// all. It mirrors ReadoutState's shape and reason: the tier that pushes needs to
// see that its value arrived without opening a window.
//
// It reports the RAW open flag rather than NoticeOpen's, so a test can tell "no
// notice was pushed" from "one was pushed to a viewer that cannot draw it".
func (v *Viewer) NoticeState() (string, NoticeKind, bool) {
	return v.notice, v.noticeKind, v.noticeOpen
}

// SetNoticeLayouts replaces the dialogue and failure layouts.
func (v *Viewer) SetNoticeLayouts(dialogue, outcome NoticeLayout) {
	v.noticeLayouts[NoticeDialogue] = dialogue
	v.noticeLayouts[NoticeOutcome] = outcome
	v.noticeSerial++
	v.noticePic = nil
}

func (v *Viewer) SetDialogFrame(art *DialogFrame) {
	if v.dialogFrame != art {
		v.noticePic = nil
		v.noticeFresh = true
	}
	for i := range v.noticeLayouts {
		v.noticeLayouts[i].Frame = art
	}
	v.dialogFrame = art
}

// SetSuccessNoticeLayout replaces only the two-button success surface.
func (v *Viewer) SetSuccessNoticeLayout(success NoticeLayout) {
	v.noticeLayouts[NoticeSuccess] = success
	v.noticeSerial++
	v.noticePic = nil
}

// noticeLayout is the layout the open notice is drawn with: the stored value
// for its kind, resolved for BOTH things that vary per notice — which of the
// window's two shapes it is, and which window is cut from this speaker's
// picture.
//
// THE OUTCOME NOTICE IS RESOLVED THE SAME WAY and is unaffected by it: its
// layout has no pane at all, so WithPortrait empties the rectangle and the pane
// is never drawn, whatever window the last speaker left behind.
func (v *Viewer) noticeLayout() NoticeLayout {
	index := int(v.noticeKind)
	if index < 0 || index >= len(v.noticeLayouts) {
		index = 0
	}
	l := v.noticeLayouts[index].WithPortrait(v.noticePortrait).WithFaceWindow(v.noticeFaceWindow)
	if v.noticeKind == NoticeDialogue {
		state := v.noticeButtonState
		state.Disabled = state.Disabled || l.ButtonState.Disabled
		l = l.WithDialogueButtonState(state).WithDialogueBackdrop(v.dialogueBackdrop.policy)
		if v.help != nil {
			l = v.helpApply(l)
		}
	} else {
		state := v.noticeButtonState
		state.Disabled = l.ButtonState.Disabled
		l.ButtonState, l.SecondaryState = state, v.noticeSecondState
	}
	return l
}

// setNoticeButtonStates writes an outcome panel's two button states.
func (v *Viewer) setNoticeButtonStates(primary, second DialogueButtonState) {
	if v.noticeButtonState != primary || v.noticeSecondState != second {
		v.noticeButtonState, v.noticeSecondState = primary, second
		v.noticePic = nil
	}
}

func (v *Viewer) SetDialogueButtonState(state DialogueButtonState) {
	if v.noticeButtonState != state {
		v.noticeButtonState = state
		v.noticePic = nil
	}
}

// Keep native notice pixels inside the mission canvas. Draw scales that canvas
// to the window; enlarging the notice here would resample it twice.
func (v *Viewer) noticePlace() frame.Placement {
	area := v.noticeViewportSize()
	return frame.FitDown(frame.W, frame.H, area.X, area.Y)
}

// noticePresent is the picture to draw this frame, where its top-left corner
// goes in window pixels, and the scale it is drawn at — or false for a frame
// that draws no notice at all.
//
// THE FONT TEST STANDS WITH THE OPEN TEST, for the unit panel's own reason:
// a viewer holding no font draws no notice and fails at nothing (AC-22).
//
// The rebuild rule is the key comparison and nothing else, exactly as the
// panel's is — and because the geometry is authored in the design space, a
// window resize rescales the same picture instead of recomposing it.
func (v *Viewer) noticePresent() (*image.RGBA, image.Point, float64, bool) {
	if !v.noticeOpen || v.font == nil {
		return nil, image.Point{}, 0, false
	}
	place := v.noticePlace()
	if !place.Valid() {
		return nil, image.Point{}, 0, false
	}
	l := v.noticeLayout()
	key := noticeKey{text: v.notice, kind: v.noticeKind, serial: v.noticeSerial,
		portrait: v.noticePortrait, face: v.noticeFaceSerial, button: l.ButtonState, second: l.SecondaryState, policy: l.DialogueBackdrop}
	if v.noticePic == nil || key != v.noticeKey {
		v.noticeText = text.Record(func() {
			v.noticePic, v.noticeFresh = RenderNotice(l, v.font, v.notice, v.noticeFace), true
		})
		v.noticeKey = key
		v.noticeBuilds++
	}
	if v.noticePic == nil {
		return nil, image.Point{}, 0, false
	}
	ox, oy := place.Origin()
	s := place.Scale()
	at := image.Pt(int(ox+float64(l.Box.Min.X)*s), int(oy+float64(l.Box.Min.Y)*s))
	if s == 1 {
		text.Append(v.noticeText, 0, 0)
	}
	return v.noticePic, at, s, true
}

// noticeButtonAt reports whether the window position (x, y) is on the open
// notice's button.
//
// IT ASKS THE SAME PLACEMENT THE DRAW ASKS, and it asks it in the DESIGN space
// rather than the window's: the position is mapped back through the letterbox
// and tested against the authored rectangle, so the hit region is the button
// that was painted whatever the window size is. Comparing scaled window pixels
// instead would be a second rounding of the same numbers, free to disagree with
// the first by a pixel at every scale.
//
// A viewer with no notice open, and one with no font, answer false — the button
// is not on screen to be clicked.
func (v *Viewer) noticeButtonAt(x, y int) bool {
	_, ok := v.noticeActionAt(x, y)
	return ok
}

// noticeDefaultAction is Return's action. The first success child is Victory;
// every single-button notice keeps the ordinary advance action.
func (v *Viewer) noticeDefaultAction() NoticeAction {
	if v != nil && v.noticeKind == NoticeFailure {
		return NoticeExitMain
	}
	if v != nil && v.noticeKind == NoticeSuccess {
		return NoticeVictory
	}
	return NoticeAdvance
}

// noticeEscapeAction is the success panel's direct Continue/close route.
func (v *Viewer) noticeEscapeAction() NoticeAction {
	if v != nil && v.noticeKind == NoticeFailure {
		return NoticeExitMain
	}
	if v != nil && v.noticeKind == NoticeSuccess {
		return NoticeContinue
	}
	return NoticeAdvance
}

func (v *Viewer) noticeDialogueInside(x, y int) bool {
	if !v.NoticeOpen() || v.noticeKind != NoticeDialogue {
		return false
	}
	_, at, scale, ok := v.noticePresent()
	if !ok {
		return false
	}
	fx, fy := v.windowToFrame(x, y)
	p := image.Pt(int(math.Floor(float64(fx-at.X)/scale)), int(math.Floor(float64(fy-at.Y)/scale)))
	return p.In(v.noticeLayout().Button)
}

// noticeActionAt resolves a click against the exact control that was painted.
func (v *Viewer) noticeActionAt(x, y int) (NoticeAction, bool) {
	switch id, _ := v.noticeButtonIDAt(x, y); id {
	case noticePrimaryButton:
		return v.noticeDefaultAction(), true
	case noticeSecondButton:
		if v.noticeKind == NoticeFailure {
			return NoticeLoadGame, true
		}
		return NoticeContinue, true
	}
	return NoticeAdvance, false
}

// The open notice's two buttons, as press-latch ids.
const (
	noticePrimaryButton = iota + 1
	noticeSecondButton
)

// noticeButtonIDAt is the enabled notice button under a window point.
func (v *Viewer) noticeButtonIDAt(x, y int) (int, bool) {
	if !v.NoticeOpen() {
		return 0, false
	}
	_, at, scale, ok := v.noticePresent()
	if !ok {
		return 0, false
	}
	fx, fy := v.windowToFrame(x, y)
	p := image.Pt(int(math.Floor(float64(fx-at.X)/scale)), int(math.Floor(float64(fy-at.Y)/scale)))
	l := v.noticeLayout()
	if p.In(l.Button) && !(l.Style == NoticeStyleDialogue && l.ButtonState.Disabled) {
		return noticePrimaryButton, true
	}
	if !l.SecondaryDisabled && !l.SecondaryButton.Empty() && p.In(l.SecondaryButton) {
		return noticeSecondButton, true
	}
	return 0, false
}
