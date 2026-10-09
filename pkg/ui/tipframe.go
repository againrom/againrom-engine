package ui

import (
	"image"
	"image/draw"
)

func TipPanelBodyRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X, r.Min.Y, r.Max.X-8, r.Max.Y-8)
}

func tipFrameTiles(a *DialogFrame, body image.Rectangle) []dialogueTile {
	if !a.valid() || body.Dx() < 64 || body.Dy() < 64 {
		return nil
	}
	l, t, r, b := body.Min.X, body.Min.Y, body.Max.X, body.Max.Y
	tiles := []dialogueTile{{1, image.Pt(l, t)}, {3, image.Pt(r-32, t)}, {6, image.Pt(l, b-32)}, {8, image.Pt(r-32, b-32)}}
	for x := l + 32; x+48 <= r-32; x += 48 {
		tiles = append(tiles, dialogueTile{2, image.Pt(x, t)}, dialogueTile{7, image.Pt(x, b-32)})
	}
	for y := t + 32; y+32 <= b-32; y += 32 {
		tiles = append(tiles, dialogueTile{4, image.Pt(l, y)}, dialogueTile{5, image.Pt(r-32, y)})
		for x := l + 32; x+48 <= r-32; x += 48 {
			tiles = append(tiles, dialogueTile{0, image.Pt(x, y)})
		}
	}
	return tiles
}

func drawTipFrame(dst *image.RGBA, a *DialogFrame, r image.Rectangle) {
	if dst == nil || !a.valid() {
		return
	}
	body := TipPanelBodyRect(r)
	tiles := tipFrameTiles(a, body)
	for _, tile := range tiles {
		switch tile.piece {
		case 3, 5, 6, 7, 8:
			stampShadow(dst, a.Pieces[tile.piece], tile.at.Add(image.Pt(8, 8)))
		}
	}
	for _, tile := range tiles {
		pic := a.Pieces[tile.piece]
		draw.Draw(dst, image.Rectangle{Min: tile.at, Max: tile.at.Add(pic.Bounds().Size())}.Intersect(body), pic, pic.Bounds().Min, draw.Over)
	}
}
