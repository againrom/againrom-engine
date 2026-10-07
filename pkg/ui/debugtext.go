package ui

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"againrom/pkg/render/debugtext"
	"againrom/pkg/render/text"
)

type debugPicture struct {
	pic   *image.RGBA
	calls []text.DrawCall
}

var debugPictures = struct {
	items map[string]debugPicture
	order []string
}{items: make(map[string]debugPicture)}

func cachedDebugPicture(s string) debugPicture {
	if p, ok := debugPictures.items[s]; ok {
		return p
	}
	var bounds image.Rectangle
	debugtext.Walk(s, 0, 0, func(p debugtext.Placement) {
		bounds = bounds.Union(image.Rect(p.X, p.Y, p.X+debugtext.CellWidth, p.Y+debugtext.LineHeight))
	})
	bounds.Min = image.Point{}
	p := debugPicture{pic: image.NewRGBA(bounds)}
	p.calls = text.Record(func() { debugtext.Draw(p.pic, s, 0, 0) })
	if len(debugPictures.order) == 64 {
		delete(debugPictures.items, debugPictures.order[0])
		debugPictures.order = debugPictures.order[1:]
	}
	debugPictures.items[s] = p
	debugPictures.order = append(debugPictures.order, s)
	return p
}

func drawDebugText(dst *ebiten.Image, log *pixelLog, s string, x, y int) {
	if dst == nil || s == "" {
		return
	}
	if !text.Capturing() {
		ebitenutil.DebugPrintAt(dst, s, x, y)
		if log != nil {
			log.debugPrint(s, x, y)
		}
		return
	}
	p := cachedDebugPicture(s)
	calls := make([]text.DrawCall, len(p.calls))
	for i, c := range p.calls {
		c.Under = slices.Clone(c.Under)
		c.X, c.Y = c.X+x, c.Y+y
		c.Clip = c.Clip.Add(image.Pt(x, y)).Intersect(dst.Bounds())
		c.Erased = true
		if log != nil {
			for n, px := range c.Glyph.Pixels {
				if px.Painted {
					if below, ok := log.value(len(log.ops), c.X+n%c.Glyph.Width, c.Y+n/c.Glyph.Width); ok {
						c.Under[n] = below
					}
				}
			}
		}
		calls[i] = c
	}
	text.Append(calls, 0, 0)
	if log != nil {
		log.over(p.pic, image.Pt(x, y))
	}
}
