package ui

import (
	"image"
	"image/color"
	"math"

	"againrom/pkg/render/text"
)

// The help panel is the Pause panel's class with the install's help text in
// place of the pause line (MENU-052): rectangle 576x384 snapped to 488x360 and
// centred, one OK button, no title, and a body in font 1 that scrolls when the
// wrapped text is taller than the body (TEXT-088). It opens on F1 over the map
// and closes like any notice (MENU-051, MENU-053).
const (
	helpBoxX, helpBoxY = 76, 60
	helpBoxW, helpBoxH = 488, 360

	// The body control's rectangle (40,56)-(448,272); its height is rebuilt
	// from the font as whole rows of height plus 4, plus 2.
	helpBodyX, helpBodyY = 40, 56
	helpBodyW, helpBodyH = 408, 216
	helpRowExtra         = 4
	helpHeightExtra      = 2

	// A scroll bar 24 wide narrows the text by 0x1a (MENU-052); it spans
	// (422,56)-(446,267), the text control's right edge less 2 (MENU-079).
	helpBarW      = 24
	helpBarNarrow = 0x1a
	helpBarLeft   = 422

	// The OK button is 96x24 at (196,300), panel-local (MENU-079).
	helpButtonX, helpButtonY = 196, 300
	helpButtonW, helpButtonH = 96, 24

	// Page Down steps visible-1 lines (MENU-078).
	helpPageDownLess = 1
)

// helpInk is grey ramp entry 15, 15 steps of 14 per channel (MENU-079). The
// shadow at +1,+1 is the font ramp's 8 per channel, the dialogue shadow.
var helpInk = color.RGBA{210, 210, 210, 255}

// helpGeometry is the wrapped text and the body it is shown in.
type helpGeometry struct {
	font    *text.Font
	body    image.Rectangle // panel-relative, the text control's rectangle
	text    image.Rectangle // panel-relative, narrowed beside a scroll bar
	bar     image.Rectangle // panel-relative, empty without a scroll bar
	lines   int
	visible int
}

// helpState is one open help panel.
type helpState struct {
	body   string
	scroll int
	geo    helpGeometry

	// cur is the control's current line: -1 until the first set-position,
	// then equal to scroll (MENU-078).
	cur int
	// okFocus is true while the OK button holds focus; the text control holds
	// it on opening.
	okFocus bool

	hold   helpHold
	grab   int // thumb top to cursor row at the grab
	repeat chargenRepeat
}

type helpScrollArt struct{ frames []*image.RGBA }

// helpGeometryOf wraps s at the body width, adds the scroll bar and rewraps at
// the narrower width when the lines are taller than the body (TEXT-088).
func helpGeometryOf(f *text.Font, s string) helpGeometry {
	g := helpGeometry{font: f}
	if f == nil || f.Height() <= 0 {
		return g
	}
	h := f.Height()
	rows := helpBodyH / (h + helpRowExtra)
	bodyH := rows*(h+helpRowExtra) + helpHeightExtra
	pitch := h + noticePitch
	g.visible = bodyH / pitch
	g.body = image.Rect(helpBodyX, helpBodyY, helpBodyX+helpBodyW, helpBodyY+bodyH)
	g.text = g.body
	n := len(dialogueWrap(f, s, g.body.Dx()))
	if n*pitch > bodyH {
		g.text.Max.X = g.body.Max.X - helpBarNarrow
		g.bar = image.Rect(helpBarLeft, g.body.Min.Y, helpBarLeft+helpBarW, g.body.Max.Y)
		n = len(dialogueWrap(f, s, g.text.Dx()))
	}
	g.lines = n
	return g
}

func (g helpGeometry) maxScroll() int { return max(g.lines-g.visible, 0) }

// OpenHelp opens the help panel over the map and reports whether it opened.
// An empty text, a viewer with no font and a viewer already showing a notice
// open nothing.
func (v *Viewer) OpenHelp(body string) bool {
	if body == "" || v.font == nil || v.NoticeOpen() {
		return false
	}
	v.SetNotice(body, NoticeDialogue)
	v.help = &helpState{body: body, cur: -1, geo: helpGeometryOf(v.font, body)}
	return true
}

// HelpOpen reports whether the help panel is the open notice.
func (v *Viewer) HelpOpen() bool { return v.help != nil && v.NoticeOpen() }

// HelpPanel is the layout the open help panel is drawn with and its body, for a
// witness that composes the panel itself because the map screen draws straight
// onto the display.
func (v *Viewer) HelpPanel() (NoticeLayout, string, bool) {
	if !v.HelpOpen() {
		return NoticeLayout{}, "", false
	}
	return v.noticeLayout(), v.help.body, true
}

// HelpScroll is the index of the first visible line and the largest index.
func (v *Viewer) HelpScroll() (first, last int) {
	if v.help == nil {
		return 0, 0
	}
	return v.help.scroll, v.help.geo.maxScroll()
}

// HelpLines is the number of wrapped lines and how many the body shows.
func (v *Viewer) HelpLines() (lines, visible int) {
	if v.help == nil {
		return 0, 0
	}
	return v.help.geo.lines, v.help.geo.visible
}

func (v *Viewer) helpGeo() helpGeometry {
	if v.help.geo.font != v.font {
		v.help.geo = helpGeometryOf(v.font, v.help.body)
	}
	return v.help.geo
}

func (v *Viewer) setHelpScrollArt(frames []*image.RGBA) {
	v.helpScroll = frames
	if v.help != nil {
		v.noticePic, v.noticeFresh = nil, true
	}
}

// helpApply resolves the dialogue layout for the help panel.
func (v *Viewer) helpApply(l NoticeLayout) NoticeLayout {
	g := v.helpGeo()
	l.Box = image.Rect(helpBoxX, helpBoxY, helpBoxX+helpBoxW, helpBoxY+helpBoxH)
	l.Text, l.Scrollbar = g.text, g.bar
	l.Portrait, l.TextBesidePortrait = image.Rectangle{}, image.Rectangle{}
	l.Button = image.Rect(helpButtonX, helpButtonY, helpButtonX+helpButtonW, helpButtonY+helpButtonH)
	l.Ink = helpInk
	l.FirstLine = min(v.help.scroll, g.maxScroll())
	if len(v.helpScroll) > 0 {
		art := DialogFrame{}
		if l.Frame != nil {
			art = *l.Frame
		}
		art.helpScroll = v.helpScroll
		l.Frame = &art
	}
	return l
}

// helpSetPos is the control's set-position routine: it clamps to 0..lines
// minus visible and stores the result as both the top line and the current
// line (MENU-078).
func (v *Viewer) helpSetPos(p int) {
	if v.help == nil {
		return
	}
	p = min(max(p, 0), v.helpGeo().maxScroll())
	v.help.cur = p
	if p == v.help.scroll {
		return
	}
	v.help.scroll = p
	v.noticeSerial++
	v.noticePic = nil
}

// helpScrollBy moves the first visible line by n, clamped to the body's range.
// The mouse routes use it.
func (v *Viewer) helpScrollBy(n int) {
	if v.help == nil || n == 0 {
		return
	}
	v.helpSetPos(v.help.scroll + n)
}

// helpKeys answers Up, Down, Page Up and Page Down. They act only while the
// text control holds focus; Up from the OK button moves focus to the text and
// scrolls nothing (MENU-078). The first Up does nothing and the first Down only
// resyncs the current line (-1) to the top.
func (v *Viewer) helpKeys(up, down, pageUp, pageDown bool) {
	h := v.help
	if h == nil {
		return
	}
	if h.okFocus {
		if up {
			h.okFocus = false
		}
		return
	}
	g := v.helpGeo()
	switch {
	case pageUp:
		if h.cur == h.scroll {
			v.helpSetPos(h.scroll - g.visible)
		} else {
			v.helpSetPos(h.scroll)
		}
	case pageDown:
		v.helpSetPos(h.scroll + g.visible - helpPageDownLess)
	case up:
		if h.cur > 0 {
			v.helpSetPos(h.cur - 1)
		}
	case down:
		v.helpSetPos(h.cur + 1)
	}
}

// helpTab moves focus between the two tab stops, the text control and the OK
// button (MENU-078).
func (v *Viewer) helpTab(tab bool) {
	if tab && v.help != nil {
		v.help.okFocus = !v.help.okFocus
	}
}

// helpBarParts is the up arrow, track, thumb and down arrow of a scroll bar.
func helpBarParts(bar image.Rectangle, first, total, visible int) (up, down, track, thumb image.Rectangle) {
	up = image.Rect(bar.Min.X, bar.Min.Y, bar.Max.X, bar.Min.Y+bar.Dx())
	down = image.Rect(bar.Min.X, bar.Max.Y-bar.Dx(), bar.Max.X, bar.Max.Y)
	track = image.Rect(bar.Min.X, up.Max.Y, bar.Max.X, down.Min.Y)
	thumb = track
	if total > visible && visible > 0 {
		h := max(track.Dy()*visible/total, bar.Dx())
		span := track.Dy() - h
		top := track.Min.Y + span*first/(total-visible)
		thumb = image.Rect(bar.Min.X, top, bar.Max.X, top+h)
	}
	return up, down, track, thumb
}

// helpHold is the scroll bar part a held primary button owns.
type helpHold int

const (
	helpHoldNone helpHold = iota
	helpHoldUp
	helpHoldDown
	helpHoldPageUp
	helpHoldPageDown
	helpHoldThumb
)

// helpPanelPoint maps a window position to the open help panel's own pixels.
func (v *Viewer) helpPanelPoint(x, y int) (image.Point, bool) {
	_, at, scale, ok := v.noticePresent()
	if !ok {
		return image.Point{}, false
	}
	fx, fy := v.windowToFrame(x, y)
	return image.Pt(int(math.Floor(float64(fx-at.X)/scale)), int(math.Floor(float64(fy-at.Y)/scale))), true
}

// helpPartAt is the scroll bar part under panel point p.
func helpPartAt(l NoticeLayout, first, total, visible int, p image.Point) helpHold {
	if l.Scrollbar.Empty() || !p.In(l.Scrollbar) {
		return helpHoldNone
	}
	up, down, _, thumb := helpBarParts(l.Scrollbar, first, total, visible)
	switch {
	case p.In(up):
		return helpHoldUp
	case p.In(down):
		return helpHoldDown
	case p.In(thumb):
		return helpHoldThumb
	case p.Y < thumb.Min.Y:
		return helpHoldPageUp
	}
	return helpHoldPageDown
}

// helpStep applies one step of a held arrow or track part.
func (v *Viewer) helpStep(part helpHold, visible int) {
	switch part {
	case helpHoldUp:
		v.helpScrollBy(-1)
	case helpHoldDown:
		v.helpScrollBy(1)
	case helpHoldPageUp:
		v.helpScrollBy(-visible)
	case helpHoldPageDown:
		v.helpScrollBy(visible)
	}
}

// helpThumbTo scrolls so the thumb's top follows panel row y, less the grab
// offset, over the track's travel.
func (v *Viewer) helpThumbTo(l NoticeLayout, y int) {
	g := v.helpGeo()
	if g.maxScroll() == 0 {
		return
	}
	_, _, track, thumb := helpBarParts(l.Scrollbar, 0, g.lines, g.visible)
	span := track.Dy() - thumb.Dy()
	if span <= 0 {
		return
	}
	top := min(max(y-v.help.grab-track.Min.Y, 0), span)
	v.helpScrollBy((top*g.maxScroll()+span/2)/span - v.help.scroll)
}

// helpPointer drives the open help panel's scroll bar and wheel for one tick.
// A press acts on its own tick: an arrow or the track steps once and the thumb
// is grabbed. A held arrow or track repeats at the character page's held-button
// timing while the cursor stays on the part, the thumb follows the cursor until
// release, and the wheel steps as the mod screens' wheel does. It reports whether the
// tick's press landed on the bar.
func (v *Viewer) helpPointer(in appInput) bool {
	h := v.help
	if h == nil || !v.NoticeOpen() {
		return false
	}
	g := v.helpGeo()
	if in.WheelY != 0 {
		step := modWheelLines
		if in.WheelY > 0 {
			step = -step
		}
		v.helpScrollBy(step)
	}
	l := v.noticeLayout()
	p, ok := v.helpPanelPoint(in.CursorX, in.CursorY)
	if !ok {
		h.hold = helpHoldNone
		return false
	}
	pressed := false
	if in.PrimaryPressed {
		h.hold = helpPartAt(l, h.scroll, g.lines, g.visible, p)
		pressed = h.hold != helpHoldNone
		h.repeat = chargenRepeat{}
		if pressed {
			if h.hold == helpHoldThumb {
				_, _, _, thumb := helpBarParts(l.Scrollbar, h.scroll, g.lines, g.visible)
				h.grab = p.Y - thumb.Min.Y
			} else {
				v.helpStep(h.hold, g.visible)
			}
			h.repeat.tick(in)
		}
		return pressed
	}
	if h.hold == helpHoldNone {
		return false
	}
	if in.PrimaryReleased || !in.Viewer.PrimaryDown {
		h.hold = helpHoldNone
		return false
	}
	if h.hold == helpHoldThumb {
		v.helpThumbTo(l, p.Y)
		return false
	}
	if h.repeat.tick(in) && helpPartAt(l, h.scroll, g.lines, g.visible, p) == h.hold {
		v.helpStep(h.hold, g.visible)
	}
	return false
}

var (
	helpBarTrackInk = color.RGBA{7, 12, 9, 255}
	helpBarKnobInk  = color.RGBA{24, 41, 36, 255}
)

func drawHelpScrollbar(dst *image.RGBA, f *text.Font, l NoticeLayout, first, total, visible int) {
	bar := l.Scrollbar
	up, down, track, thumb := helpBarParts(bar, first, total, visible)
	if l.Frame != nil && drawScrollbarSkin(dst, l.Frame.helpScroll, up, track, down, thumb) {
		return
	}
	for y := track.Min.Y; y < track.Max.Y; y++ {
		for x := track.Min.X; x < track.Max.X; x++ {
			dst.SetRGBA(x, y, helpBarTrackInk)
		}
	}
	for _, knob := range []image.Rectangle{up, down, thumb} {
		for y := knob.Min.Y; y < knob.Max.Y; y++ {
			for x := knob.Min.X; x < knob.Max.X; x++ {
				dst.SetRGBA(x, y, helpBarKnobInk)
			}
		}
		drawDialogueButton(dst, knob, "", f, l)
	}
	arrow := func(r image.Rectangle, pointUp bool) {
		cx, cy := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
		for i := 0; i < 4; i++ {
			y := cy - 2 + i
			if !pointUp {
				y = cy + 1 - i
			}
			for x := cx - i; x <= cx+i; x++ {
				dst.SetRGBA(x, y, dialogueButtonInk)
			}
		}
	}
	arrow(up, true)
	arrow(down, false)
}
