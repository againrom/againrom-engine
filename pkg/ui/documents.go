package ui

import (
	"image"
	"image/color"
	"strings"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

// The campaign documents panel (B3, MENU-DOC-009, MISSION-DOC-021,
// DLG-WRAP-009, TEXT-API-007).
//
// The panel is a full 640x480 sheet with three controls on it — a left
// arrow, a right arrow and an OK button — and one document drawn at the
// document rectangle's top-left corner. It is a SCREEN in this build rather
// than a window over the running mission, which is what makes the mission
// behind it stop: the tick is issued on the map arm alone (flow.go's own
// note on the in-game menu). The original runs at one 640x480 surface and
// its panel covers the whole of it, so a screen and a full-screen window are
// the same picture; this build's mission frame is 1024x768, which is why the
// difference had to be decided rather than inherited (DIV-299).

// docSheetRect is the sheet's own extent: the whole frame.
var docSheetRect = image.Rect(0, 0, frame.W, frame.H)

// The three hit rectangles, in frame pixels (MENU-DOC-009).
var (
	docLeftRect  = image.Rect(0, 200, 56, 240)
	docRightRect = image.Rect(576, 200, 636, 240)
	docOKRect    = image.Rect(560, 416, 604, 448)
)

// docContentAt is the document rectangle's top-left corner (92, 72): where a
// picture is blitted at natural size and where the text page's first line
// starts.
var docContentAt = image.Pt(92, 72)

// docContentWidth is the document rectangle's own width, 456 px. It is the
// width a text document is wrapped to.
//
// A PICTURE IS NOT CLIPPED TO IT. The one shipped picture is 464 px wide,
// eight wider than this rectangle, and MENU-DOC-009 records that overhang as
// what the original draws rather than as a defect to correct.
const docContentWidth = 456

// docPageLines is the page step: 0x15 = 21 lines (MENU-DOC-009).
const docPageLines = 21

// docLinePitch is how far apart two drawn lines sit, in pixels: the font's own
// height plus two.
//
// IT IS DLG-WRAP-009's DEFAULT AND NOT A NUMBER CHOSEN HERE. That row reads
// the pitch field's own default as `font height + 2` for a caller that passes
// zero, and MENU-DOC-009 records that this panel's caller passes zero, so the
// pitch is taken from the glyph sprite. font4's records are 16 px tall on both
// shipped roots, which makes the shipped pitch 18; nothing here spells 18.
const docLinePitchPad = 2

// docTextColor is what a page's lines are drawn in. font4's own records carry
// a per-pixel level and no usable colour of its own (DIV-303: every painted
// pixel of all 224 records on the English root indexes a near-white palette
// entry, so the atlas holds a level ramp and nothing else); text.Font.Draw
// scales THIS colour by that level.
//
// THE VALUE IS SAMPLED, NOT WHITE. The original draws this page's letters as
// solid dark brown with no pale interior and no bright fringe — measured by
// counting exact pixel colours over the owner's own screenshots of the
// licensed EN original (review/docs-original/orig-en-edict-{a,b}.png,
// outside this repository): (0x42, 0x2c, 0x10) is the single most common
// non-parchment colour in both, 8292 and 8293 pixels respectively, with no
// nearby shade anywhere near that population — the parchment texture itself
// supplies dozens of close but distinct browns, and this one value recurs
// exactly. A white docTextColor scaled toward black by Shade at a glyph's
// partial levels reads as pale letters with a dark fringe; this colour scaled
// the same way reads as dark letters with a slightly darker edge, which is
// what a solid ink colour under this build's antialiasing model produces.
var docTextColor = color.RGBA{0x42, 0x2c, 0x10, 0xff}

// docParagraphIndent is how far a paragraph's own first line is shifted right
// of the margin, in frame pixels.
//
// BOUNDED BY THE SAME REFERENCE SCREENSHOTS, NOT DECODED (DIV-1387): a
// continuation line's own leftmost ink sits flush with docContentAt.X on
// every measured row; a paragraph's own first line sits measurably right of
// it. The exact width is not established, but the wrap it forces bounds it:
// the Royal Edict text's first paragraph does not fit "646," on its own
// first line while its second paragraph's first line fits an equally long
// run, which under this build's own font metrics requires an indent above 3
// native px and no more than 14 — the first paragraph's first line, measured
// without any indent, has 44px of headroom before "646," and the second
// paragraph's first line has 14px of headroom in total. 12 sits inside that
// band without touching either edge.
const docParagraphIndent = 12

type DocumentPanelArt struct {
	Sheet *image.RGBA
	Left  [3]*image.RGBA
	Right [3]*image.RGBA
	OK    [4]*image.RGBA
}

// DocumentPage is one element of the campaign document collection, already
// resolved to what it draws: a picture at natural size, or a body of text.
//
// THE TEXT ARRIVES UNWRAPPED. Wrapping needs the font, and the font is this
// tier's; the tier that loads the element reads the file whole and hands over
// its bytes, exactly as the original reads the node's whole length and then
// puts it through the wrap (MISSION-DOC-021).
type DocumentPage struct {
	Picture *image.RGBA
	Text    string
}

// DocumentSource is the presentation seam for the campaign document
// collection: what the panel shows when it opens.
//
// IT IS ASKED AT EVERY OPEN and holds nothing. The collection grows as the
// campaign advances, so a panel built once at install time would show the
// collection as it stood when the front end was constructed.
type DocumentSource interface {
	Documents() []DocumentPage
}

// documentPanel is the panel's own state: which element is current, which line
// of it the page starts at, and which control the pointer is over or pressing.
//
// THE TWO POSITION FIELDS ARE THE ORIGINAL'S TWO. `doc` stands for the
// collection index the original's own current-element pointer names, and
// `line` for its `+0x38`, the first line of the page. Paging is one control
// over both, in that order — see step.
type documentPanel struct {
	art   *DocumentPanelArt
	font  *text.Font
	pages []DocumentPage

	// lines is each text page's wrapped lines, computed once when the panel
	// is built. A picture element carries no lines, so its own entry is
	// empty and every page step over it returns false — which is what makes
	// an arrow on a picture move to the next document.
	lines [][]docLine

	doc  int
	line int

	// hover and press are which control the pointer is over and which one a
	// held press began on, or docNoControl. They select the bitmap state and
	// nothing else.
	hover int
	press buttonLatch
}

// pressed is the control the press latched, or docNoControl.
func (p *documentPanel) pressed() int {
	if c, ok := p.press.Latched(); ok {
		return c
	}
	return docNoControl
}

// The three controls, and the absence of one.
const (
	docNoControl = iota
	docControlLeft
	docControlRight
	docControlOK
)

// newDocumentPanel builds the panel over a collection, wrapping every text
// element to the document rectangle's own width under font.
//
// A NIL FONT WRAPS NOTHING and every text element then pages as a single
// empty page, which is what a front end with no font atlas would draw anyway.
func newDocumentPanel(art *DocumentPanelArt, font *text.Font, pages []DocumentPage) *documentPanel {
	p := &documentPanel{art: art, font: font, pages: pages, hover: docNoControl}
	p.lines = make([][]docLine, len(pages))
	for i, page := range pages {
		if page.Text == "" || font == nil {
			continue
		}
		// wrapDocumentText is this screen's own greedy word wrap — DIV-302:
		// the original's break rule itself is not decoded, and this is this
		// build's approximation of it, same as wrapShopTip is everywhere
		// else. It additionally indents and justifies (DIV-1387,
		// docParagraphIndent's own header), which no other screen does; nothing
		// here reaches wrapShopTip or changes what it does for its other
		// callers.
		p.lines[i] = wrapDocumentText(font, page.Text, docContentWidth, docParagraphIndent)
	}
	return p
}

// docLine is one wrapped, drawable line of a text page: the words it holds,
// whether it opens a paragraph (drawn docParagraphIndent right of the margin),
// and whether it is stretched to fill the full width or left at its natural,
// ragged one.
//
// A PARAGRAPH'S OWN LAST LINE IS NEVER JUSTIFIED, including a paragraph that
// is only one line long — the original's own last line of a paragraph ends
// ragged, at its natural width, which is the standard typeset rule this build
// reproduces (Royal Edict reference screenshots, review/docs-original/, DIV-1387).
type docLine struct {
	words   []string
	text    string
	indent  bool
	justify bool
}

// wrapDocumentText splits s into docLines: a blank source line (after
// strings.Fields finds no word on it) starts the next paragraph without
// drawing a line of its own, and every other source line is wrapped
// independently, exactly as wrapShopTip wraps each of strings.Split(s, "\n")'s
// own elements today. The only two differences from that shared wrap are
// paragraph indent on a group's own first line and justification on every
// group but its last, both scoped to this screen alone.
func wrapDocumentText(font *text.Font, s string, width, indent int) []docLine {
	var out []docLine
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		out = append(out, wrapDocumentParagraph(font, words, width, indent)...)
	}
	return out
}

// wrapDocumentParagraph greedily groups words into lines that each measure no
// wider than width under font — width less indent for the very first group,
// mirroring wrapShopTip's own algorithm otherwise. The first group is marked
// to draw indented; every group but the last is marked to draw justified.
func wrapDocumentParagraph(font *text.Font, words []string, width, indent int) []docLine {
	firstWidth := width - indent
	if firstWidth < 0 {
		firstWidth = 0
	}

	var groups [][]string
	line := words[0]
	group := []string{words[0]}
	for _, w := range words[1:] {
		candidate := line + " " + w
		limit := width
		if len(groups) == 0 {
			limit = firstWidth
		}
		if cw, _ := font.Measure(candidate); cw <= limit {
			line = candidate
			group = append(group, w)
			continue
		}
		groups = append(groups, group)
		line = w
		group = []string{w}
	}
	groups = append(groups, group)

	lines := make([]docLine, len(groups))
	for i, g := range groups {
		lines[i] = docLine{
			words:   g,
			text:    strings.Join(g, " "),
			indent:  i == 0,
			justify: i < len(groups)-1,
		}
	}
	return lines
}

// drawDocumentLine draws one docLine's own words at (x, y): a plain
// font.Draw for a line with nothing to justify (one word, or the last line of
// its paragraph), or each word placed by hand with the line's leftover width
// spread across its own gaps otherwise.
//
// THE SPREAD IS THE STANDARD ONE: extra = width - the line's own natural
// advance, divided evenly over (word count - 1) gaps, with the remainder's
// own pixels — never more than one per gap — added to the leftmost gaps
// first. width is already indent-adjusted by the caller for an indented
// line, so this function reads only the words and the target width.
func drawDocumentLine(dst *image.RGBA, font *text.Font, l docLine, x, y, width int, c color.RGBA) {
	if font == nil {
		return
	}
	if !l.justify || len(l.words) < 2 {
		font.Draw(dst, l.text, x, y, c)
		return
	}
	natural := font.Advance(l.text)
	extra := width - natural
	if extra < 0 {
		extra = 0
	}
	gaps := len(l.words) - 1
	base, rem := extra/gaps, extra%gaps
	space := font.Advance(" ")
	px := x
	for i, w := range l.words {
		font.Draw(dst, w, px, y, c)
		if i == gaps {
			break
		}
		px += font.Advance(w) + space + base
		if i < rem {
			px++
		}
	}
}

// lineCount is how many lines the current element holds — the bound the page
// step is taken against (`+0x18` in the original).
func (p *documentPanel) lineCount() int {
	if p == nil || p.doc < 0 || p.doc >= len(p.lines) {
		return 0
	}
	return len(p.lines[p.doc])
}

// pageCount is how many whole pages the current element's own lines fill. A
// picture element has none.
func (p *documentPanel) pageCount() int {
	n := p.lineCount()
	if n == 0 {
		return 0
	}
	return (n + docPageLines - 1) / docPageLines
}

// pageStep moves the page by dir whole pages and reports whether it moved.
//
// IT IS THE ORIGINAL'S OWN BOUND: a step of 21 lines on the page's first line,
// refused when it would leave the element's own line count. Refusing rather
// than clamping is what makes the last page a partial one and what makes the
// return value mean "this element is exhausted in this direction".
func (p *documentPanel) pageStep(dir int) bool {
	if p == nil {
		return false
	}
	next := p.line + dir*docPageLines
	if next < 0 || next >= p.lineCount() {
		return false
	}
	p.line = next
	return true
}

// step is one arrow press: pages first, and changes document only when the
// page step returns zero (MENU-DOC-009).
//
// THE ORDER IS THE WHOLE BEHAVIOUR. One arrow pair walks pages first and
// documents second; a build that paged documents directly is a different
// screen. A picture element has no lines at all, so its page step always
// refuses and an arrow over it always moves to the next document.
//
// A NEW ELEMENT STARTS AT ITS FIRST LINE IN BOTH DIRECTIONS. Nothing decoded
// says which line the original lands on when it steps BACK into an element,
// and landing on the first line is the one answer that needs no line count
// the panel has not read yet (DIV-300).
//
// The collection's two ends refuse rather than wrap: an arrow at the first
// element's first page and one at the last element's last page both do
// nothing, which is what a bounded step already does inside an element.
func (p *documentPanel) step(dir int) {
	if p == nil || len(p.pages) == 0 {
		return
	}
	if p.pageStep(dir) {
		return
	}
	next := p.doc + dir
	if next < 0 || next >= len(p.pages) {
		return
	}
	p.doc = next
	p.line = 0
}

// page is the lines the current element's current page draws, at most
// docPageLines of them: the original's own `+0x38` to `+0x38 + 0x15`, cut at
// the element's own line count so the last page is a partial one.
func (p *documentPanel) page() []docLine {
	lines := p.lines[p.doc]
	if p.line >= len(lines) {
		return nil
	}
	end := p.line + docPageLines
	if end > len(lines) {
		end = len(lines)
	}
	return lines[p.line:end]
}

// picture is the current element's own bitmap, or nil for a text element.
func (p *documentPanel) picture() *image.RGBA {
	if p == nil || p.doc < 0 || p.doc >= len(p.pages) {
		return nil
	}
	return p.pages[p.doc].Picture
}

// docControlRect is one control's own rectangle, the single table
// documentControlAt and the composer both read. A control with no rectangle is
// the empty one, which no pixel is in.
func docControlRect(control int) image.Rectangle {
	switch control {
	case docControlLeft:
		return docLeftRect
	case docControlRight:
		return docRightRect
	case docControlOK:
		return docOKRect
	}
	return image.Rectangle{}
}

// documentControlAt reports which control stands under frame pixel (x, y), and
// whether any does. The three rectangles do not overlap, so the order of the
// tests is not observable.
func documentControlAt(x, y int) (int, bool) {
	p := image.Pt(x, y)
	switch {
	case p.In(docLeftRect):
		return docControlLeft, true
	case p.In(docRightRect):
		return docControlRight, true
	case p.In(docOKRect):
		return docControlOK, true
	}
	return docNoControl, false
}

// docArrowFrame is which of an arrow's three bitmaps a control in this state
// draws: 0 on leave, 1 on hover, 2 on press (MENU-DOC-009).
func docArrowFrame(control, hover, press int) int {
	switch {
	case press == control:
		return 2
	case hover == control && press == docNoControl:
		return 1
	}
	return 0
}

// docOKFrame is which of OK's four bitmaps this state draws: 0 `Ok_off` on
// leave, 1 `Ok_on` on hover, 3 `Ok_l_on` on press.
//
// `Ok_l_off`, ELEMENT 2, IS LOADED AND NEVER DRAWN, and which of the four is
// the unused one is a place where MENU-DOC-009 disagrees with itself
// (DIV-301). Its closing clause instead says `Ok_on`, element 1, is selected
// by no arm.
//
// THIS BUILD TAKES THE INDEX SENTENCE, and the shipped file names are a second
// reading that agrees with it. The arrow arrays hold three files each, named
// `00`, `01` and `11` — a two-bit code with `10` absent — and the row reads
// them as idle, hover, pressed in that order. The OK array holds the same code
// filled out: `Ok_off` = 00, `Ok_on` = 01, `Ok_l_off` = 10, `Ok_l_on` = 11.
// Selecting 0, 1 and 3 is selecting 00, 01 and 11, which is the arrows' own
// three states, and it leaves unused exactly the combination the arrows do not
// ship. All four files are 4278 bytes and all four are distinct, and each is
// byte-identical between the English and Russian roots (measured over both
// `graphics.res` archives), so no pair is a duplicate that would make the
// choice moot.
//
// The two readings differ by which 44x32 bitmap the button wears while the
// pointer is over it and by nothing else.
func docOKFrame(hover, press int) int {
	switch {
	case press == docControlOK:
		return 3
	case hover == docControlOK && press == docNoControl:
		return 1
	}
	return 0
}

// composeDocumentsPanel paints the whole panel into a fresh 640x480 frame.
//
// THE DRAW ORDER IS THE ORIGINAL'S (MENU-DOC-009): sheet, left arrow, right
// arrow, OK, then the current document at the panel origin. It matters where
// the picture overhangs its own rectangle — the document is drawn last, so
// it lies over the sheet rather than under it.
//
// A NIL PANEL OR NIL ART still returns a frame rather than nothing: a front
// end whose archive did not carry the sheet draws a black screen it can leave
// with OK, not a crash.
func composeDocumentsPanel(p *documentPanel) *image.RGBA {
	dst := image.NewRGBA(docSheetRect)
	if p == nil {
		return dst
	}
	if p.art != nil {
		copyNative(dst, p.art.Sheet, docSheetRect.Min, docSheetRect)
		copyNative(dst, p.art.Left[docArrowFrame(docControlLeft, p.hover, p.pressed())],
			docLeftRect.Min, docSheetRect)
		copyNative(dst, p.art.Right[docArrowFrame(docControlRight, p.hover, p.pressed())],
			docRightRect.Min, docSheetRect)
		copyNative(dst, p.art.OK[docOKFrame(p.hover, p.pressed())], docOKRect.Min, docSheetRect)
	}
	if len(p.pages) == 0 {
		return dst
	}
	if pic := p.picture(); pic != nil {
		copyNative(dst, pic, docContentAt, docSheetRect)
		return dst
	}
	if p.font == nil {
		return dst
	}
	pitch := p.font.Height() + docLinePitchPad
	for i, line := range p.page() {
		x, width := docContentAt.X, docContentWidth
		if line.indent {
			x += docParagraphIndent
			width -= docParagraphIndent
		}
		drawDocumentLine(dst, p.font, line, x, docContentAt.Y+i*pitch, width, docTextColor)
	}
	return dst
}

// HeadlessDocuments is what the campaign documents panel is showing: which
// element of the collection, which page of that element, and how many of
// each there are.
//
// IT IS A READ AND NOTHING ELSE. It exists because a control cannot report that
// the sheet under it was painted with the wrong element: a scenario asserting
// only that the panel is open cannot tell an arrow that pages from one that
// does nothing (AGENTS.md coverage rule 7).
type HeadlessDocuments struct {
	Element  int  `json:"element"`
	Elements int  `json:"elements"`
	Page     int  `json:"page"`
	Pages    int  `json:"pages"`
	Picture  bool `json:"picture"`
}

// HeadlessDocumentState is the open panel's own position, or false when no
// panel is open.
func (a *App) HeadlessDocumentState() (HeadlessDocuments, bool) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenDocuments || a.flow.docPanel == nil {
		return HeadlessDocuments{}, false
	}
	p := a.flow.docPanel
	out := HeadlessDocuments{
		Element:  p.doc,
		Elements: len(p.pages),
		Page:     p.line / docPageLines,
		Pages:    p.pageCount(),
		Picture:  p.picture() != nil,
	}
	return out, true
}
