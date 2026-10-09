package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
)

// scrlbars.256 frames the shared bar and slider painter selects (MENU-117).
// Every frame is 24x24 (MENU-129).
const (
	barTopFrame          = 18
	barTopHotFrame       = 21
	barTrackFrame        = 19
	barBottomFrame       = 20
	barBottomHotFrame    = 23
	barThumbFrame        = 22
	sliderLeftFrame      = 0
	sliderLeftHotFrame   = 3
	sliderTrackFrame     = 7
	sliderRightFrame     = 8
	sliderRightHotFrame  = 11
	sliderKnobFrame      = 10
	scrollArtFrames      = 24
	widgetShadowLevel    = 4
	widgetDisabledLevel  = 3
	widgetSpriteSize     = 24
	listPitchExtra       = 4
	listBottomExtra      = 2
	listFallbackPitch    = 19
	listTextX, listTextY = 3, 2
)

var widgetShadowOffset = image.Pt(4, 4)

// scrollArt reports whether frames hold every scrlbars.256 frame a painter
// selects.
func scrollArt(frames []*image.RGBA) bool {
	if len(frames) < scrollArtFrames {
		return false
	}
	for _, i := range []int{barTopFrame, barTopHotFrame, barTrackFrame, barBottomFrame, barBottomHotFrame, barThumbFrame,
		sliderLeftFrame, sliderLeftHotFrame, sliderTrackFrame, sliderRightFrame, sliderRightHotFrame, sliderKnobFrame} {
		if frames[i] == nil || frames[i].Bounds().Empty() {
			return false
		}
	}
	return true
}

// drawWidgetSprite is one part request: the sprite's shadow at (+4,+4),
// remapping the destination under its mask at level 4, then its body.
func drawWidgetSprite(dst, pic *image.RGBA, at image.Point, clip image.Rectangle) {
	if dst == nil || pic == nil {
		return
	}
	if l, err := backdrop.NewLevel(backdrop.RGB565, backdrop.Full, widgetShadowLevel); err == nil {
		remapDialogueMask(dst, pic, at.Add(widgetShadowOffset), clip, l, nil)
	}
	copyNativeOver(dst, pic, at, clip)
}

// remapDisabled is the level-3 remap a disabled widget requests over r
// after painting.
func remapDisabled(dst *image.RGBA, r image.Rectangle) {
	if l, err := backdrop.New(backdrop.RGB565, backdrop.Full); err == nil {
		l.Apply(dst, r, dst.Bounds(), 1)
	}
}

// vScrollBar is the shared vertical bar (MENU-117, MENU-119): its rectangle,
// position and range count, the endcap states the pointer writes, and the
// enabled flag. The thumb is one fixed frame, not a visible fraction.
type vScrollBar struct {
	Rect              image.Rectangle
	Pos, Count        int
	TopHot, BottomHot bool
	Disabled          bool
}

// travel is the thumb's offset q from the painter's rule.
func (b vScrollBar) travel() int {
	if b.Count < 2 {
		return 0
	}
	w, h := b.Rect.Dx(), b.Rect.Dy()
	return b.Pos * (h - 3*w + 8) / (b.Count - 1)
}

// Thumb is the thumb body's rectangle.
func (b vScrollBar) Thumb() image.Rectangle {
	y := b.Rect.Min.Y + b.Rect.Dx() + b.travel() - 4
	return image.Rect(b.Rect.Min.X, y, b.Rect.Min.X+widgetSpriteSize, y+widgetSpriteSize)
}

// top and bottom are the endcap regions the press handler tests.
func (b vScrollBar) top() image.Rectangle {
	return image.Rect(b.Rect.Min.X, b.Rect.Min.Y, b.Rect.Max.X, b.Rect.Min.Y+b.Rect.Dx())
}

func (b vScrollBar) bottom() image.Rectangle {
	return image.Rect(b.Rect.Min.X, b.Rect.Max.Y-(b.Rect.Dx()-4), b.Rect.Max.X, b.Rect.Max.Y)
}

// withPointer sets the endcap states from pointer membership, as the bar's
// move handler does.
func (b vScrollBar) withPointer(p image.Point, ok bool) vScrollBar {
	b.TopHot = ok && p.In(b.top())
	b.BottomHot = ok && p.In(b.bottom())
	return b
}

// drawVScrollBar paints top cap, track tiles, bottom cap and thumb, each a
// shadowed sprite part, then remaps a disabled bar. Without art it paints a
// plain fallback of the same geometry.
func drawVScrollBar(dst *image.RGBA, frames []*image.RGBA, b vScrollBar) {
	recordWidget(widgetVScrollBar, b.Rect, b)
	r := b.Rect
	if r.Empty() {
		return
	}
	if !scrollArt(frames) {
		draw.Draw(dst, r, &image.Uniform{C: helpBarTrackInk}, image.Point{}, draw.Src)
		draw.Draw(dst, b.Thumb().Intersect(r), &image.Uniform{C: helpBarKnobInk}, image.Point{}, draw.Src)
	} else {
		w := r.Dx()
		top, bottom := frames[barTopFrame], frames[barBottomFrame]
		if b.TopHot {
			top = frames[barTopHotFrame]
		}
		if b.BottomHot {
			bottom = frames[barBottomHotFrame]
		}
		clip := dst.Bounds()
		drawWidgetSprite(dst, top, r.Min, clip)
		track := image.Rect(r.Min.X, r.Min.Y+w, r.Max.X, r.Max.Y-w)
		tiles := image.Rect(r.Min.X, track.Min.Y, r.Max.X+widgetShadowOffset.X, track.Max.Y)
		for y := track.Min.Y; y < track.Max.Y; y += widgetSpriteSize {
			drawWidgetSprite(dst, frames[barTrackFrame], image.Pt(r.Min.X, y), tiles)
		}
		drawWidgetSprite(dst, bottom, image.Pt(r.Min.X, r.Max.Y-w), clip)
		drawWidgetSprite(dst, frames[barThumbFrame], b.Thumb().Min, clip)
	}
	if b.Disabled {
		remapDisabled(dst, r)
	}
}

// barRequest is what a bar asks its owner (MENU-119): a line or page step,
// a dragged position, or nothing.
type barRequest uint8

const (
	barNone barRequest = iota
	barLineUp
	barLineDown
	barPageUp
	barPageDown
	barSetPos
)

// press is the bar's down handler: which request a press at p makes, and
// whether it grabbed the thumb.
func (b vScrollBar) press(p image.Point) (barRequest, bool) {
	if !p.In(b.Rect) || b.Disabled {
		return barNone, false
	}
	switch thumb := b.Thumb(); {
	case p.In(b.top()):
		return barLineUp, false
	case p.In(b.bottom()):
		return barLineDown, false
	case p.Y < thumb.Min.Y:
		return barPageUp, false
	case p.Y >= thumb.Max.Y:
		return barPageDown, false
	}
	return barNone, true
}

// dragPos maps a dragged pointer row to a position, the drag arm's rule.
func (b vScrollBar) dragPos(y int) int {
	w, h := b.Rect.Dx(), b.Rect.Dy()
	span := h - 3*(w-4)
	if b.Count < 2 || span <= 0 {
		return 0
	}
	return min(max((b.Count-1)*(y-b.Rect.Min.Y-24)/span, 0), b.Count-1)
}

// scrollBarInput is one bar's held gesture: a held line or page part
// repeats on the character page's held-button timing, and a grabbed thumb
// follows the pointer until release.
type scrollBarInput struct {
	held   barRequest
	drag   bool
	repeat chargenRepeat
}

// step runs one tick of the bar's input and returns the request it makes.
func (s *scrollBarInput) step(b vScrollBar, p image.Point, ok bool, in appInput) (barRequest, int) {
	if in.PrimaryPressed {
		s.held, s.drag, s.repeat = barNone, false, chargenRepeat{}
		if !ok {
			return barNone, 0
		}
		req, grab := b.press(p)
		s.held, s.drag = req, grab
		s.repeat.tick(in)
		return req, 0
	}
	if s.held == barNone && !s.drag {
		return barNone, 0
	}
	if in.PrimaryReleased || !in.Viewer.PrimaryDown {
		s.held, s.drag = barNone, false
		return barNone, 0
	}
	if s.drag {
		if !ok {
			return barNone, 0
		}
		return barSetPos, b.dragPos(p.Y)
	}
	if s.repeat.tick(in) && ok {
		if req, _ := b.press(p); req == s.held {
			return req, 0
		}
	}
	return barNone, 0
}

// active reports whether the bar owns the held button.
func (s scrollBarInput) active() bool { return s.held != barNone || s.drag }

// reset drops the held gesture.
func (s *scrollBarInput) reset() { *s = scrollBarInput{} }

// listBox is the shared list's geometry (MENU-120, MENU-121): rows of font
// height plus 4, the bottom rounded to whole rows plus 2, and the bar 24
// wide at the list's right edge.
type listBox struct {
	Rect  image.Rectangle
	Pitch int
	Rows  int
}

// newListBox places a list of rows rows at the argument rectangle's top
// left, as wide as it, for font f.
func newListBox(r image.Rectangle, rows int, f *text.Font) listBox {
	pitch := listFallbackPitch
	if f != nil && f.Height() > 0 {
		pitch = f.Height() + listPitchExtra
	}
	return listBox{Rect: image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+rows*pitch+listBottomExtra), Pitch: pitch, Rows: rows}
}

// Row is visible row i's rectangle.
func (l listBox) Row(i int) image.Rectangle {
	y := l.Rect.Min.Y + i*l.Pitch
	return image.Rect(l.Rect.Min.X, y, l.Rect.Max.X, y+l.Pitch)
}

// Bar is the attached bar's rectangle.
func (l listBox) Bar() image.Rectangle {
	return image.Rect(l.Rect.Max.X, l.Rect.Min.Y, l.Rect.Max.X+widgetSpriteSize, l.Rect.Max.Y)
}

// RowAt is the visible row under p.
func (l listBox) RowAt(p image.Point) (int, bool) {
	if !p.In(l.Rect) {
		return 0, false
	}
	i := (p.Y - l.Rect.Min.Y) / l.Pitch
	return i, i < l.Rows
}

// listBar is the bar bound to list m: position is the selected index and
// range is the item count (MENU-120).
func listBar(l listBox, m *Picker) vScrollBar {
	b := vScrollBar{Rect: l.Bar()}
	if m != nil {
		b.Pos, b.Count = max(m.Selection(), 0), m.Len()
	}
	return b
}

// listKey is the shared list's four focused keys (MENU-120). Page Up first
// selects the top row, then a page above it; Page Down first selects the last
// visible row, then a page below it. It reports whether a key acted.
func listKey(m *Picker, up, down, pageUp, pageDown bool) bool {
	if m == nil || m.Len() == 0 {
		return false
	}
	top, _ := m.Visible()
	visible := m.visibleRows()
	sel := m.Selection()
	switch {
	case up:
		m.Select(clampIndex(sel-1, m.Len()))
	case down:
		m.Select(clampIndex(sel+1, m.Len()))
	case pageUp:
		target := top - visible
		if sel != top {
			target = top
		}
		m.Select(clampIndex(target, m.Len()))
	case pageDown:
		target := top + 2*visible - 1
		if sel != top+visible-1 {
			target = top + visible - 1
		}
		m.Select(clampIndex(target, m.Len()))
	default:
		return false
	}
	return true
}

// listBarRequest applies a bar request to its list: line and page requests
// act as the arrow and page keys, and a dragged position selects that item.
func listBarRequest(m *Picker, req barRequest, pos int) bool {
	switch req {
	case barLineUp:
		return listKey(m, true, false, false, false)
	case barLineDown:
		return listKey(m, false, true, false, false)
	case barPageUp:
		return listKey(m, false, false, true, false)
	case barPageDown:
		return listKey(m, false, false, false, true)
	case barSetPos:
		return m != nil && m.Select(clampIndex(pos, m.Len()))
	}
	return false
}

var (
	listRowInk      = townShellText
	listSelectedInk = loadSelectedText
	listSelectFill  = color.RGBA{0, 7, 6, 220}
	listSelectEdge  = color.RGBA{57, 77, 65, 255}
)

// listRowText is how a row's text reaches the font: the screen's own
// encoder and fitting.
type listRowText func(row int, width int) string

// drawListBox paints the shared list's visible rows and its bar. The
// selected row is filled and outlined and drawn in the selected ink.
func drawListBox(dst *image.RGBA, f *text.Font, frames []*image.RGBA, l listBox, m *Picker, rowText listRowText, pointer image.Point, pointerOK bool) {
	recordWidget(widgetListBox, l.Rect, l)
	if m != nil && f != nil {
		top, count := m.Visible()
		for i := 0; i < count && i < l.Rows; i++ {
			r := l.Row(i)
			ink := listRowInk
			if top+i == m.Selection() {
				draw.Draw(dst, r, &image.Uniform{C: listSelectFill}, image.Point{}, draw.Over)
				outline(dst, r, listSelectEdge)
				ink = listSelectedInk
			}
			f.Draw(dst.SubImage(r.Inset(1)).(*image.RGBA), rowText(top+i, r.Dx()-2*listTextX), r.Min.X+listTextX, r.Min.Y+listTextY, ink)
		}
	}
	drawVScrollBar(dst, frames, listBar(l, m).withPointer(pointer, pointerOK))
}
