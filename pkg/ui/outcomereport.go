package ui

import "image"

// OutcomeGeometry is one outcome page's rectangles in the 640x480 design
// space (Box) and page-relative (the rest). An empty rectangle keeps the
// authored one; FrameInset and CenterText are the Dialog fields of those names.
type OutcomeGeometry struct {
	Box, Text, Button, Secondary image.Rectangle
	FrameInset                   int
	CenterText                   bool
}

// OutcomeReport is a game's victory and failure page geometry; the zero value
// keeps the authored pages.
type OutcomeReport struct {
	Success, Failure OutcomeGeometry
}

// onto is l with g's rectangles, or base's where g has none.
func (g OutcomeGeometry) onto(l, base Dialog) Dialog {
	pick := func(r, b image.Rectangle) image.Rectangle {
		if r.Empty() {
			return b
		}
		return r
	}
	l.Box = pick(g.Box, base.Box)
	l.Text = pick(g.Text, base.Text)
	l.Button = pick(g.Button, base.Button)
	l.SecondaryButton = pick(g.Secondary, base.SecondaryButton)
	l.FrameInset, l.CenterText = g.FrameInset, g.CenterText
	return l
}

// SecondGameOutcomeReport is the second game's pages: the victory popup built
// 384x180 and snapped to whole frame tiles, centred on the frame, with its
// text and two buttons; the failure page snapped from 576x280 (width and
// buttons only, its height and text stay authored).
func SecondGameOutcomeReport() OutcomeReport {
	pop := newDialogGeometry(384, 180).Rectangle
	fail := newDialogGeometry(576, 280).Rectangle
	authored := AuthoredFailureLayout().Box
	w := fail.Dx()
	return OutcomeReport{
		Success: OutcomeGeometry{
			Box:        pop,
			Text:       image.Rect(40, 32, 232, 80),
			Button:     image.Rect(40, 84, 248, 108),
			Secondary:  image.Rect(40, 108, 248, 132),
			FrameInset: dialogShadowBand,
			CenterText: true,
		},
		Failure: OutcomeGeometry{
			Box:        image.Rect(fail.Min.X, authored.Min.Y, fail.Max.X, authored.Max.Y),
			Button:     image.Rect(w/8, 84, 7*w/8, 108),
			Secondary:  image.Rect(w/8, 108, 7*w/8, 132),
			FrameInset: dialogShadowBand,
		},
	}
}
