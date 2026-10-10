package ui

import (
	"image"
	"image/draw"

	"againrom/pkg/render/text"
)

// A room's button panel is one kind with one builder: the shop's commands,
// the school's, the three- and four-command tavern's and the generator's
// detail commands. Each screen supplies a panelComposition (where the body,
// its seam and each plaque sit), the panelArt the install resolved for it, and
// each button's captions and pointer state; buildButtonPanel turns them into
// the body paint and the push buttons the shared painter draws.

// panelComposition is a button panel's geometry and drawing rules.
type panelComposition struct {
	// Body is where the body picture draws; Seam where the seam picture
	// draws when the body is narrower than its region.
	Body, Seam image.Rectangle
	// BodyOver draws a keyed body over what lies under it; otherwise the body
	// is copied.
	BodyOver bool
	// MissingBox is where the town shell box draws when the body is missing;
	// MissingFrame, when set, is the frame drawn instead.
	MissingBox   image.Rectangle
	MissingFrame *frameSpec
	// Plaques are the buttons' draw and hit rectangles, in panel order.
	Plaques []image.Rectangle
	// Bare draws a plaque with no art as the town shell box.
	Bare bool
	// FitLine trims a single-line caption to its plaque.
	FitLine bool
	Ink     plaqueInk
}

// panelArt is a panel's pictures. Plaques holds each plaque's off and on
// picture: both make a pair; an on picture alone draws over the body only
// while the button is held with the pointer inside (TOWN-260).
type panelArt struct {
	Body, Seam image.Image
	Plaques    [][2]image.Image
}

// panelButton is one button's captions and state. One caption is a line
// centred in the plaque; two are the label and value bands.
type panelButton struct {
	Captions                         []string
	Hover, Pressed, Inside, Disabled bool
}

// buttonPanel is a built panel: its body and its push buttons.
type buttonPanel struct {
	comp    panelComposition
	art     panelArt
	buttons []pushButton
}

// buildButtonPanel builds a panel from its composition, art and button
// states. Buttons beyond the composition's plaques are dropped.
func buildButtonPanel(c panelComposition, art panelArt, buttons []panelButton) buttonPanel {
	n := min(len(buttons), len(c.Plaques))
	out := make([]pushButton, n)
	for i := 0; i < n; i++ {
		r, b := c.Plaques[i], buttons[i]
		face := &plaqueFace{Ink: c.Ink, Sink: plaqueSink, Captions: panelCaptions(r, b.Captions, c.FitLine)}
		var pair [2]image.Image
		if i < len(art.Plaques) {
			pair = art.Plaques[i]
		}
		switch {
		case pair[0] == nil && pair[1] != nil:
			face.Pictures[plaqueDown], face.Over = pair[1], true
		case pair[0] != nil && pair[1] != nil:
			face.Pictures = plaquePair(pair)
		default:
			face.Bare = c.Bare
		}
		out[i] = pushButton{Rect: r, Face: face, Hover: b.Hover, Pressed: b.Pressed, Inside: b.Inside, Disabled: b.Disabled}
	}
	return buttonPanel{comp: c, art: art, buttons: out}
}

// panelCaptions lays out a button's captions in plaque r.
func panelCaptions(r image.Rectangle, captions []string, fitLine bool) []plaqueCaption {
	switch len(captions) {
	case 0:
		return nil
	case 1:
		return []plaqueCaption{{Text: captions[0], Rect: r, Fit: fitLine}}
	}
	return []plaqueCaption{{Text: captions[0], Rect: townButtonLabelRect(r), Fit: true},
		{Text: captions[1], Rect: townButtonValueRect(r), Fit: true}}
}

// drawBody draws the body, or what stands for a missing one, then the seam.
func (p buttonPanel) drawBody(dst *image.RGBA) {
	c := p.comp
	switch body := p.art.Body; {
	case body != nil:
		op := draw.Src
		if c.BodyOver {
			op = draw.Over
		}
		draw.Draw(dst, c.Body, body, body.Bounds().Min, op)
	case c.MissingFrame != nil:
		drawFrame(dst, *c.MissingFrame)
	case !c.MissingBox.Empty():
		drawTownShellBox(dst, c.MissingBox, false)
	}
	if seam := p.art.Seam; seam != nil {
		draw.Draw(dst, c.Seam, seam, seam.Bounds().Min, draw.Over)
	}
}

// drawButtons draws every plaque through the shared painter.
func (p buttonPanel) drawButtons(dst *image.RGBA, f *text.Font) {
	for _, b := range p.buttons {
		drawPushButton(dst, f, b)
	}
}
