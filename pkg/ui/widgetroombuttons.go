package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// RoomButtonStyle is a game's room button captions: Font, when set, in
// place of the screen's font, Ink at rest and Hover under the pointer. A
// button that cannot act keeps Ink, or with Lit draws as one that can; its
// press is refused either way. OneLine marks the shop buttons whose caption
// and number draw as one line. LabelShift and ValueShift move the label and
// the value line of a two-line caption.
type RoomButtonStyle struct {
	Font                   *text.Font
	Ink, Hover             color.RGBA
	Lit                    bool
	OneLine                [4]bool
	LabelShift, ValueShift image.Point
}

// disabled reports whether a button that can or cannot act draws disabled.
func (s *RoomButtonStyle) disabled(canAct bool) bool {
	return !canAct && (s == nil || !s.Lit)
}

// oneLine reports whether shop button i draws its caption and number as one
// line.
func (s *RoomButtonStyle) oneLine(i int) bool {
	return s != nil && i >= 0 && i < len(s.OneLine) && s.OneLine[i]
}

// shift moves comp's label lines, the plaques' own shifts kept, and its
// value lines by the style's shifts.
func (s *RoomButtonStyle) shift(comp *panelComposition) {
	if s == nil || s.LabelShift == (image.Point{}) && s.ValueShift == (image.Point{}) {
		return
	}
	label := make([]image.Point, len(comp.Plaques))
	for i := range label {
		if i < len(comp.LabelShift) {
			label[i] = comp.LabelShift[i]
		}
		label[i] = label[i].Add(s.LabelShift)
	}
	comp.LabelShift, comp.ValueShift = label, comp.ValueShift.Add(s.ValueShift)
}

// plaqueInk is the style's ink; base without a style.
func (s *RoomButtonStyle) plaqueInk(base plaqueInk) plaqueInk {
	if s == nil {
		return base
	}
	return plaqueInk{Rest: s.Ink, Hover: s.Hover, Disabled: s.Ink}
}

// font is the style's font; f without one.
func (s *RoomButtonStyle) font(f *text.Font) *text.Font {
	if s == nil || s.Font == nil {
		return f
	}
	return s.Font
}

// RoomButton is one button of a composed room page: its rectangle, its off
// and on pictures, its caption and whether the pointer is over it.
type RoomButton struct {
	Rect    image.Rectangle
	Off, On image.Image
	Caption string
	Hover   bool
}

// roomButtonInk is a composed room page's caption ink without a style.
var roomButtonInk = plaqueInk{Rest: townShellText, Hover: townShellText, Disabled: townShellText}
