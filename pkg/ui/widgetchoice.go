package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
)

// radiob.256 frame pairs (MENU-123, MENU-124, MENU-125): round radio 0/1,
// standard checkbox 2/3, tip checkbox 4/5.
const (
	radioOffFrame    = 0
	checkOffFrame    = 2
	tipCheckOffFrame = 4
	choiceArtFrames  = 6
)

// choiceKind is which of the three painters a group uses.
type choiceKind uint8

const (
	choiceRadio choiceKind = iota
	choiceCheck
	choiceTipCheck
)

// Label inks. The runtime ramp colours are Unknown (MENU-123..125); the
// engine uses the menu text for the idle ramp and the menu gold for the
// focus or pointer ramp.
var (
	choiceIdleInk = gameMenuText
	choiceHotInk  = loadSelectedText
)

// choiceGroup is one radio group or checkbox group: rows from the top of
// Rect at the kind's pitch, the on state of each row, and the two frames.
type choiceGroup struct {
	Kind   choiceKind
	Rect   image.Rectangle
	Labels []string
	// Selected is a radio group's row; Mask a checkbox group's set bits.
	Selected int
	Mask     uint32
	// Focus is the keyboard focus on the group; Row its keyboard row.
	Focus bool
	Row   int
	// Pointer drives the tip checkbox's per-row label ramp.
	Pointer   image.Point
	PointerOK bool
	Disabled  bool
	Off, On   image.Image
}

func (g choiceGroup) pitch() int {
	if g.Kind == choiceTipCheck {
		return 16
	}
	return 24
}

func (g choiceGroup) on(i int) bool {
	if g.Kind == choiceRadio {
		return i == g.Selected
	}
	return i < 32 && g.Mask&(1<<i) != 0
}

// RowRect is row i's band across the group.
func (g choiceGroup) RowRect(i int) image.Rectangle {
	y := g.Rect.Min.Y + i*g.pitch()
	return image.Rect(g.Rect.Min.X, y, g.Rect.Max.X, y+g.pitch())
}

// RowAt is the row a press at p selects: the signed truncation of
// (y-T)/pitch, for a point inside the group and a row that exists.
func (g choiceGroup) RowAt(p image.Point) (int, bool) {
	if !p.In(g.Rect) {
		return 0, false
	}
	i := (p.Y - g.Rect.Min.Y) / g.pitch()
	return i, i >= 0 && i < len(g.Labels)
}

// ink is row i's label ramp: a radio's selected row under focus, a whole
// checkbox group under focus, or a tip row under the pointer.
func (g choiceGroup) ink(i int) color.RGBA {
	hot := false
	switch g.Kind {
	case choiceRadio:
		hot = g.Focus && i == g.Selected
	case choiceCheck:
		hot = g.Focus
	case choiceTipCheck:
		hot = g.PointerOK && g.Pointer.In(g.RowRect(i))
	}
	if hot {
		return choiceHotInk
	}
	return choiceIdleInk
}

// drawChoiceGroup paints each row's frame with its level-4 shadow at
// (+4,+4) and its label, then remaps a disabled group at level 3 over the
// rectangle grown by one (MENU-123..125). Without a frame it draws a plain
// box of the frame's size.
func drawChoiceGroup(dst *image.RGBA, f *text.Font, g choiceGroup) {
	kind := widgetCheck
	if g.Kind == choiceRadio {
		kind = widgetRadio
	}
	recordWidget(kind, g.Rect, g)
	if dst == nil || g.Rect.Empty() {
		return
	}
	size, labelX, labelY := 24, 30, 5
	if g.Kind == choiceTipCheck {
		size, labelX, labelY = 16, 22, 3
	}
	off, on := rgbaOf(g.Off), rgbaOf(g.On)
	var calls []text.DrawCall
	for i, label := range g.Labels {
		at := image.Pt(g.Rect.Min.X+1, g.Rect.Min.Y+i*g.pitch())
		pic := off
		if g.on(i) {
			pic = on
		}
		if pic != nil && !pic.Bounds().Empty() {
			drawWidgetSprite(dst, pic, at.Sub(pic.Bounds().Min), dst.Bounds())
		} else {
			box := image.Rect(at.X+2, at.Y+2, at.X+size-2, at.Y+size-2)
			tint := color.RGBA{0, 0, 0, 45}
			if g.on(i) {
				tint = color.RGBA{0, 7, 6, 220}
			}
			drawFrame(dst, frameSpec{Kind: frameTint, Rect: box, Fill: tint, Border: color.RGBA{57, 77, 65, 255}})
		}
		if f == nil || label == "" {
			continue
		}
		x, y, ink := g.Rect.Min.X+labelX, g.Rect.Min.Y+i*g.pitch()+labelY, g.ink(i)
		clip := image.Rect(x, g.RowRect(i).Min.Y, g.Rect.Max.X, g.Rect.Max.Y).Intersect(dst.Bounds())
		calls = append(calls, text.Record(func() {
			f.Draw(dst.SubImage(clip).(*image.RGBA), gameMenuLabelText(label), x, y, ink)
		})...)
	}
	if g.Disabled {
		r := g.Rect.Inset(-1)
		if l, err := backdrop.New(backdrop.RGB565, backdrop.Full); err == nil {
			l.Apply(dst, r, dst.Bounds(), 1)
			remapCaptured(calls, r, l, 1)
		}
	}
	text.Append(calls, 0, 0)
}

// rgbaOf returns img as an RGBA picture, copying it when it is another kind.
func rgbaOf(img image.Image) *image.RGBA {
	if img == nil {
		return nil
	}
	if p, ok := img.(*image.RGBA); ok {
		return p
	}
	p := image.NewRGBA(img.Bounds())
	draw.Draw(p, p.Bounds(), img, img.Bounds().Min, draw.Src)
	return p
}
