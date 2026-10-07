package ui

import (
	"image"
	"image/draw"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
)

// dialogueTile is one blit of one frame piece.
type dialogueTile struct {
	piece int
	at    image.Point
}

// dialogueTiles is the frame's tiling for a body rectangle (`DLG-PANEL-035`):
// four corners, the top and bottom edges in whole tiles of pieces 2 and 7, the
// left and right edges in whole tiles of pieces 4 and 5, and the interior in
// whole tiles of piece 0. Corners are 48 px and the counts are
// (width-96)/96 across and (height-96)/64 down for the shipped pieces.
func (art *DialogFrame) dialogueTiles(body image.Rectangle) []dialogueTile {
	size := func(i int) image.Point { return art.Pieces[i].Bounds().Size() }
	corner := size(1)
	nx := max(0, (body.Dx()-2*corner.X)/size(2).X)
	ny := max(0, (body.Dy()-2*corner.Y)/size(4).Y)
	l, t, r, b := body.Min.X, body.Min.Y, body.Max.X, body.Max.Y
	tiles := []dialogueTile{
		{1, image.Pt(l, t)},
		{3, image.Pt(r-size(3).X, t)},
		{6, image.Pt(l, b-size(6).Y)},
		{8, image.Pt(r-size(8).X, b-size(8).Y)},
	}
	for i := 0; i < nx; i++ {
		tiles = append(tiles,
			dialogueTile{2, image.Pt(l+corner.X+i*size(2).X, t)},
			dialogueTile{7, image.Pt(l+corner.X+i*size(7).X, b-size(7).Y)})
	}
	for j := 0; j < ny; j++ {
		tiles = append(tiles,
			dialogueTile{4, image.Pt(l, t+corner.Y+j*size(4).Y)},
			dialogueTile{5, image.Pt(r-size(5).X, t+corner.Y+j*size(5).Y)})
	}
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			tiles = append(tiles, dialogueTile{0, image.Pt(l+corner.X+i*size(0).X, t+corner.Y+j*size(0).Y)})
		}
	}
	return tiles
}

func (art *DialogFrame) dialogueShadows(body image.Rectangle) []dialogueTile {
	var shadows []dialogueTile
	for _, tile := range art.dialogueTiles(body) {
		switch tile.piece {
		case 3, 5, 6, 7, 8:
			tile.at = tile.at.Add(image.Pt(noticeShadow, noticeShadow))
			shadows = append(shadows, tile)
		}
	}
	return shadows
}

// DIALOGUE-063: mask remaps precede every normal replacement.
func (art *DialogFrame) drawDialogueBody(dst *image.RGBA, body image.Rectangle) {
	art.drawDialogueBodyWithPolicy(dst, body, DialogueBackdrop{})
}

func (art *DialogFrame) drawDialogueBodyWithPolicy(dst *image.RGBA, body image.Rectangle, policy DialogueBackdrop) {
	art.applyDialogueShadows(dst, body, policy, nil)
	art.paintDialogueBody(dst, body, policy)
}

func (art *DialogFrame) paintDialogueBody(dst *image.RGBA, body image.Rectangle, policy DialogueBackdrop) {
	clip := dialoguePolicyClip(policy, dst.Bounds())
	for _, tile := range art.dialogueTiles(body) {
		pic := art.Pieces[tile.piece]
		r := image.Rectangle{Min: tile.at, Max: tile.at.Add(pic.Bounds().Size())}.Intersect(clip)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				c := pic.RGBAAt(pic.Rect.Min.X+x-tile.at.X, pic.Rect.Min.Y+y-tile.at.Y)
				if c.A != 0 {
					dst.SetRGBA(x, y, c)
				}
			}
		}
	}
}

// stampShadow darkens dst under every written pixel of pic placed at at.
func stampShadow(dst *image.RGBA, pic *image.RGBA, at image.Point) {
	l, _ := backdrop.NewLevel(backdrop.RGB565, backdrop.Full, 6)
	remapDialogueMask(dst, pic, at, dst.Bounds(), l, nil)
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
	if dst == nil || !art.valid() {
		return
	}
	l, err := backdrop.NewLevel(policy.Layout, policy.Mode, 6)
	if err != nil {
		panic(err)
	}
	for _, tile := range art.dialogueShadows(body) {
		remapDialogueMask(dst, art.Pieces[tile.piece], tile.at, dialoguePolicyClip(policy, dst.Bounds()), l, calls)
	}
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
