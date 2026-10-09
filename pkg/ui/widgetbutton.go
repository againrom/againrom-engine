package ui

import (
	"image"
	"image/color"
	"slices"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
)

// widgetKind names a shared widget builder.
type widgetKind uint8

const (
	widgetPushButton widgetKind = iota + 1
	widgetVScrollBar
	widgetListBox
	widgetSlider
	widgetRadio
	widgetCheck
	widgetEdit
	widgetHoverBox
	widgetFrame
)

// widgetCall is one call of a shared builder: its kind, rectangle and state.
type widgetCall struct {
	kind  widgetKind
	rect  image.Rectangle
	state any
}

// widgetRecord, when set, receives every shared builder call. Only tests set
// it, to prove a screen draws through the shared builders.
var widgetRecord func(widgetCall)

func recordWidget(kind widgetKind, r image.Rectangle, state any) {
	if widgetRecord != nil {
		widgetRecord(widgetCall{kind: kind, rect: r, state: state})
	}
}

// pushButtonRamp is the painter's colour argument (MENU-115). Zero is the
// menu ramp: grey at rest, gold while the enabled button is hovered or
// focused. Nonzero is the dialogue pager's: gold at rest, brown on hover.
type pushButtonRamp uint8

const (
	pushButtonMenu pushButtonRamp = iota
	pushButtonDialogue
)

var (
	pushButtonIdleInk  = color.RGBA{210, 210, 210, 255}
	pushButtonHoverInk = color.RGBA{150, 90, 0, 255}
)

// pushButton is one push button's look: rectangle, caption and the states
// the shared painter reads (MENU-115, DLG-BUTTON-039, DIALOGUE-065).
type pushButton struct {
	Rect  image.Rectangle
	Label string
	Ramp  pushButtonRamp
	// Hover is the pointer over the button; Focus the keyboard focus.
	Hover, Focus bool
	// Pressed is the press latch; Inside the pointer within the rectangle.
	Pressed, Inside bool
	Disabled        bool
	// Literal draws Label as given, with no accelerator marker.
	Literal bool
	// LabelRaise replaces the default vertical anchor when positive.
	LabelRaise int
	Policy     DialogueBackdrop
}

// ink is the label colour the ramp selects for the button's state.
func (b pushButton) ink() color.RGBA {
	if b.Ramp == pushButtonDialogue {
		if b.Hover {
			return pushButtonHoverInk
		}
		return dialogueButtonInk
	}
	if !b.Disabled && (b.Hover || b.Focus) {
		return dialogueButtonInk
	}
	return pushButtonIdleInk
}

// sunk reports the pressed look: the latch held with the pointer inside.
func (b pushButton) sunk() bool { return b.Pressed && b.Inside }

// drawPushButton is the one push-button painter. It draws no art and no
// fill: the caption with its shadow, then the two-colour bevel from lines
// and points, raised at rest and swapped while pressed inside, where the
// shadow offset grows from 2 to 4 and the caption stays put. A disabled
// button is remapped at level 3 after painting. A lone `~` underlines the
// next character in the ink (TEXT-079).
func drawPushButton(dst *image.RGBA, f *text.Font, b pushButton) {
	recordWidget(widgetPushButton, b.Rect, b)
	r := b.Rect
	if dst == nil || r.Dx() < 4 || r.Dy() < 4 {
		return
	}
	layout := b.Policy.Layout
	quantize := func(c color.RGBA) color.RGBA {
		return dialoguePackedColor(dialoguePackedWord(c, layout), layout)
	}
	left, top, right, bottom := r.Min.X, r.Min.Y, r.Max.X-1, r.Max.Y-1
	shadow := dialogueButtonShadow
	light, dark := quantize(dialogueBevelLight), quantize(dialogueBevelDark)
	if b.sunk() {
		light, dark = dark, light
		shadow = 4
	}
	var calls []text.DrawCall
	if b.Label != "" && f != nil {
		label := b.Label
		if !b.Literal {
			label = gameMenuLabelText(label)
		}
		x := left + (right-left)/2 + 1 - f.Advance(label)/2
		raise := dialogueLabelRaise
		if b.LabelRaise > 0 {
			raise = b.LabelRaise
		}
		y := top + (bottom-top)/2 - raise
		ink := b.ink()
		flat := quantize(messageShadowColor)
		before := slices.Clone(dst.Pix)
		calls = text.Record(func() { f.DrawFlat(dst, label, x+shadow, y+shadow, flat); f.Draw(dst, label, x, y, ink) })
		copy(dst.Pix, before)
		for i := range calls {
			c := &calls[i]
			if c.Glyph == nil || c.Glyph.Width <= 0 {
				continue
			}
			c.Under = make([]color.RGBA, len(c.Glyph.Pixels))
			c.RasterColors = make([]color.RGBA, len(c.Glyph.Pixels))
			for n, p := range c.Glyph.Pixels {
				at := image.Pt(c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width)
				shade := c.CellColor(p.Level)
				if !c.Flat {
					shade = quantize(shade)
				}
				c.RasterColors[n] = shade
				if p.Painted && at.In(dst.Bounds()) {
					c.Under[n] = dst.RGBAAt(at.X, at.Y)
					dst.SetRGBA(at.X, at.Y, shade)
				}
			}
		}
		if col := gameMenuAcceleratorColumn(b.Label); !b.Literal && col >= 0 && col < len(label) {
			ux, uw := x+f.Advance(label[:col]), f.Advance(label[col:col+1])
			uy := y + f.Height()
			for _, pass := range []struct {
				d int
				c color.RGBA
			}{{shadow, flat}, {0, quantize(ink)}} {
				for px := ux; px < ux+uw; px++ {
					if p := image.Pt(px+pass.d, uy+pass.d); p.In(dst.Bounds()) {
						dst.SetRGBA(p.X, p.Y, pass.c)
					}
				}
			}
		}
	}
	hline := func(y, x0, x1 int, c color.RGBA) {
		for x := x0; x <= x1; x++ {
			if image.Pt(x, y).In(dst.Bounds()) {
				dst.SetRGBA(x, y, c)
			}
		}
	}
	vline := func(x, y0, y1 int, c color.RGBA) {
		for y := y0; y <= y1; y++ {
			if image.Pt(x, y).In(dst.Bounds()) {
				dst.SetRGBA(x, y, c)
			}
		}
	}
	vline(right, top+2, bottom-2, dark)
	vline(right-1, top+1, bottom-1, dark)
	hline(bottom, left+2, right-2, dark)
	hline(bottom-1, left+1, right-1, dark)
	hline(bottom-2, right-2, right-2, dark)
	hline(top, left+2, right-2, light)
	vline(left, top+2, bottom-2, light)
	hline(top+1, left+1, left+1, light)
	if b.Disabled {
		if lookup, err := backdrop.New(layout, b.Policy.Mode); err == nil {
			lookup.Apply(dst, r, dst.Bounds(), 1)
			remapCaptured(calls, r, lookup, 1)
		}
	}
	text.Append(calls, 0, 0)
}

// dialogueButton is the dialogue pager's button: the shared painter with
// the dialogue ramp and the layout's own state and pixel format.
func dialogueButton(b image.Rectangle, label string, l NoticeLayout) pushButton {
	s := l.ButtonState
	return pushButton{Rect: b, Label: label, Ramp: pushButtonDialogue, Hover: s.Hover,
		Pressed: s.Pressed, Inside: s.Inside, Disabled: s.Disabled, Policy: l.DialogueBackdrop}
}

// buttonLatch is the shared button input (MENU-116, DIALOGUE-045): a press
// on an enabled button latches it, and the release activates it only when
// it lands inside the latched button. A release elsewhere clears the latch.
type buttonLatch struct {
	held bool
	id   int
}

// press latches id. A press on no button (ok false) clears the latch.
func (l *buttonLatch) press(id int, ok bool) { l.held, l.id = ok, id }

// pressed reports whether id holds the latch.
func (l buttonLatch) pressed(id int) bool { return l.held && l.id == id }

// release clears the latch and reports the button it activates: the latched
// one, when the release lands inside it.
func (l *buttonLatch) release(id int, inside bool) (int, bool) {
	held, latched := l.held, l.id
	l.held, l.id = false, 0
	return latched, held && inside && latched == id
}

// clear drops the latch without activating anything.
func (l *buttonLatch) clear() { l.held, l.id = false, 0 }

// pointerFrame is the pointer's last frame position, the hover source every
// widget painter reads.
func (a *App) pointerFrame() (image.Point, bool) {
	if a == nil {
		return image.Point{}, false
	}
	return a.windowToNativeFrame(a.pointer.X, a.pointer.Y)
}

// hovered reports whether the pointer is over r.
func (a *App) hovered(r image.Rectangle) bool {
	p, ok := a.pointerFrame()
	return ok && p.In(r)
}
