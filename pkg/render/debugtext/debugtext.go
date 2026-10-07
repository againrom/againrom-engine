// Package debugtext supplies the fixed-cell debug face and its shadow.
package debugtext

import (
	"image"
	"image/color"
	"strings"
	"sync"

	"github.com/hajimehoshi/bitmapfont/v4"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"againrom/pkg/render/text"
)

const (
	CellWidth  = 6
	LineHeight = 16
)

// Placement locates the face and disjoint shadow masks of one rune.
type Placement struct {
	Face, Shadow *text.Glyph
	X, Y         int
}

type masks struct {
	face, shadow text.Glyph
}

type source struct {
	atlas *image.RGBA
	mask  [256]masks
}

var cachedSource = sync.OnceValue(buildSource)

func buildSource() *source {
	s := &source{atlas: image.NewRGBA(image.Rect(0, 0, 32*CellWidth, 8*LineHeight))}
	for _, ink := range []struct {
		color  color.RGBA
		offset image.Point
	}{{color.RGBA{A: 128}, image.Pt(1, 1)}, {color.RGBA{255, 255, 255, 255}, image.Point{}}} {
		drawer := font.Drawer{Dst: s.atlas, Src: image.NewUniform(ink.color), Face: bitmapfont.Face}
		for row := 0; row < 8; row++ {
			var line strings.Builder
			for col := 0; col < 32; col++ {
				line.WriteRune(rune(row*32 + col))
			}
			drawer.Dot = fixed.Point26_6{
				X: fixed.I(ink.offset.X),
				Y: bitmapfont.Face.Metrics().Ascent + fixed.I(row*LineHeight+ink.offset.Y),
			}
			drawer.DrawString(line.String())
		}
	}
	for r := range s.mask {
		m := &s.mask[r]
		m.face = text.Glyph{Width: CellWidth, Height: LineHeight, Advance: CellWidth, Pixels: make([]text.Pixel, CellWidth*LineHeight)}
		m.shadow = text.Glyph{Width: CellWidth, Height: LineHeight, Advance: CellWidth, Pixels: make([]text.Pixel, CellWidth*LineHeight)}
		for n := range m.face.Pixels {
			p := s.atlas.RGBAAt(r%32*CellWidth+n%CellWidth, r/32*LineHeight+n/CellWidth)
			switch p {
			case color.RGBA{255, 255, 255, 255}:
				m.face.Pixels[n] = text.Pixel{Level: text.MaxLevel, Painted: true}
			case color.RGBA{A: 128}:
				m.shadow.Pixels[n] = text.Pixel{Level: text.MaxLevel, Painted: true}
			case color.RGBA{}:
			default:
				panic("debug face contains unsupported ink")
			}
		}
	}
	return s
}

// Walk keeps rune advances, including blank cells and unavailable runes.
func Walk(s string, x, y int, visit func(Placement)) {
	if visit == nil || s == "" {
		return
	}
	src := cachedSource()
	px, py := x+1, y
	for _, r := range s {
		if r == '\n' {
			px, py = x+1, py+LineHeight
			continue
		}
		if r < rune(len(src.mask)) {
			m := &src.mask[r]
			visit(Placement{Face: &m.face, Shadow: &m.shadow, X: px, Y: py})
		}
		px += CellWidth
	}
}

// Draw composites the shadow and face through the ordinary glyph capture.
func Draw(dst *image.RGBA, s string, x, y int) {
	if dst == nil {
		return
	}
	Walk(s, x, y, func(p Placement) {
		text.DrawGlyphOver(dst, p.Shadow, p.X, p.Y, color.RGBA{A: 128})
		text.DrawGlyphOver(dst, p.Face, p.X, p.Y, color.RGBA{255, 255, 255, 255})
	})
}
