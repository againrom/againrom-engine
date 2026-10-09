package ui

import (
	"image"
	"image/draw"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
)

// dialogueFrame is the dialogue window's frame over body: the shared window
// row with the dialogue backdrop's pixel format and clip.
func (art *DialogFrame) dialogueFrame(body image.Rectangle, policy DialogueBackdrop) frameSpec {
	s := windowFrame(body, art)
	s.Policy = policy
	return s
}

// DIALOGUE-063: mask remaps precede every normal replacement.
func (art *DialogFrame) drawDialogueBody(dst *image.RGBA, body image.Rectangle) {
	art.drawDialogueBodyWithPolicy(dst, body, DialogueBackdrop{})
}

func (art *DialogFrame) drawDialogueBodyWithPolicy(dst *image.RGBA, body image.Rectangle, policy DialogueBackdrop) {
	drawFrame(dst, art.dialogueFrame(body, policy))
}

func dialoguePolicyClip(p DialogueBackdrop, bounds image.Rectangle) image.Rectangle {
	if p.FrameClipped {
		return bounds.Intersect(p.FrameClip)
	}
	return bounds
}

func dialogueMaskAt(mask *image.RGBA, at image.Point, x, y int) bool {
	p := mask.Rect.Min.Add(image.Pt(x, y).Sub(at))
	return p.In(mask.Bounds()) && mask.RGBAAt(p.X, p.Y).A != 0
}

func remapDialogueMask(dst, mask *image.RGBA, at image.Point, clip image.Rectangle, l *backdrop.Lookup, calls []text.DrawCall) {
	r := image.Rectangle{Min: at, Max: at.Add(mask.Bounds().Size())}.Intersect(dst.Bounds()).Intersect(clip)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if dialogueMaskAt(mask, at, x, y) {
				c := dst.RGBAAt(x, y)
				if c.A != 0 {
					dst.SetRGBA(x, y, l.Color(c))
				}
			}
		}
	}
	remapCapturedMask(calls, r, l, mask, at, false)
}

func (art *DialogFrame) applyDialogueShadows(dst *image.RGBA, body image.Rectangle, policy DialogueBackdrop, calls []text.DrawCall) {
	drawFrameShadows(dst, art.dialogueFrame(body, policy), calls)
}

// ComposeDialogueNotice applies the frame masks to the supplied scene before its image.
func ComposeDialogueNotice(dst *image.RGBA, layout NoticeLayout, font *text.Font, body string, face *image.RGBA, at image.Point) {
	if dst == nil {
		return
	}
	pic := RenderNotice(layout, font, body, face)
	if pic == nil {
		return
	}
	if layout.Style == NoticeStyleDialogue {
		layout.Frame.applyDialogueShadows(dst, image.Rectangle{Max: layout.Box.Size().Sub(image.Pt(noticeShadow, noticeShadow))}.Add(at), layout.DialogueBackdrop, nil)
	}
	composeDialogueImageWithPolicy(dst, pic, at, layout.DialogueBackdrop)
}

func composeDialogueImage(dst, pic *image.RGBA, at image.Point) {
	composeDialogueImageWithPolicy(dst, pic, at, DialogueBackdrop{})
}

func composeDialogueImageWithPolicy(dst, pic *image.RGBA, at image.Point, policy DialogueBackdrop) {
	if dst == nil || pic == nil {
		return
	}
	r := image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}.Intersect(dialoguePolicyClip(policy, dst.Bounds()))
	draw.Draw(dst, r, pic, pic.Bounds().Min.Add(r.Min.Sub(at)), draw.Over)
}

func townDialogueFrame(t TownScreen) (*DialogFrame, image.Rectangle) {
	if provider, ok := t.(interface {
		TownDialogueFrame() (*DialogFrame, image.Rectangle)
	}); ok {
		return provider.TownDialogueFrame()
	}
	return nil, image.Rectangle{}
}
