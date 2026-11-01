package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// frameKind is one row of the frame table: every window, panel and border
// the screens draw is one of these kinds, drawn by drawFrame.
type frameKind uint8

const (
	// frameWindow is the lm.256 frames 0..8 window of the shared dialog base
	// (MENU-127, DLG-PANEL-035): Save, Load, the cutscene library, Game and
	// Sound Options, the in-game menu, quest objectives, notices, the
	// dialogue window and mod screens.
	frameWindow frameKind = iota + 1
	// frameTip is the room-tip window, lm.256 frames 9..17 (MENU-127).
	frameTip
	// framePanel is the engine's flat panel: a fill with a one-pixel border
	// (DIV-2722).
	framePanel
	// frameOutline is the panel's one-pixel border alone.
	frameOutline
	// framePortrait is the notice portrait's t_border.256 border, its
	// corners unscaled and its edges tiled (DIV-2723).
	framePortrait
	// frameTint is a translucent fill over the picture with a one-pixel
	// border: the choice box drawn where a choice has no sprite art.
	frameTint
	// frameFill is the panel's fill alone.
	frameFill
	// frameArt is the dialogue portrait's border picture, drawn whole and
	// unscaled from the body's corner, its clear pixels left out.
	frameArt
	// frameScroll is the world map's scroll card: a cap picture above and
	// below and the paper picture repeated between them.
	frameScroll
)

// frameScrollCap is the height of a scroll card's cap pictures.
const frameScrollCap = 12

// frameRow is how drawFrame draws one kind.
type frameRow struct {
	// tiled draws the nine lm pieces: corners, whole edge tiles inward from
	// the near corner and whole fill tiles between them.
	tiled bool
	// cover repeats the last edge and fill tile, clipped, where the body is
	// not whole tiles (DIV-2724).
	cover bool
	// shadow is the band right of and below the body that the mask shadows
	// of pieces 3, 5, 6, 7 and 8 fall into.
	shadow int
	// fill paints the panel's interior; border its one-pixel edge. blend
	// draws the fill over the picture instead of replacing it.
	fill, border, blend bool
	// art draws the whole portrait picture of the spec's art.
	art bool
	// scroll draws the spec's scroll pictures.
	scroll bool
	// corner is the portrait border's unscaled corner.
	corner image.Point
}

var frameRows = [...]frameRow{
	frameWindow:   {tiled: true, cover: true, shadow: 8},
	frameTip:      {tiled: true, shadow: 8},
	framePanel:    {fill: true, border: true},
	frameOutline:  {border: true},
	framePortrait: {corner: image.Pt(22, 27)},
	frameTint:     {fill: true, border: true, blend: true},
	frameFill:     {fill: true},
	frameArt:      {art: true},
	frameScroll:   {scroll: true},
}

// frameShadowTone is the shadow a frame casts where the destination holds
// no picture yet, as on a menu panel composed apart from the scene behind
// it (DIV-2725).
var frameShadowTone = color.RGBA{0, 0, 0, 96}

// frameSides is a set of a border's edges.
type frameSides uint8

const (
	edgeTopBit frameSides = 1 << iota
	edgeBottomBit
	edgeLeftBit
	edgeRightBit
)

// frameSpec is one frame: its kind, its rectangle and its art or colours.
type frameSpec struct {
	Kind frameKind
	// Rect is the frame with its shadow band. With Snap only its size is
	// read: the dialog base snaps the size and centres it (MENU-077).
	Rect image.Rectangle
	Snap bool
	// Art carries the nine pieces of a window or tip, or the portrait border.
	Art *DialogFrame
	// Fill and Border colour a panel or an outline.
	Fill, Border color.RGBA
	// Width is the border's thickness in pixels; zero is one.
	Width int
	// Scroll is a scroll card's top cap, paper and bottom cap.
	Scroll [3]image.Image
	// Sides names the edges the border has; zero is all four.
	Sides frameSides
	// Policy is the dialogue backdrop's pixel format and clip.
	Policy DialogueBackdrop
}

// windowFrame is the window whose body is body; its shadow falls outside.
func windowFrame(body image.Rectangle, art *DialogFrame) frameSpec {
	return frameSpec{Kind: frameWindow, Rect: image.Rect(body.Min.X, body.Min.Y,
		body.Max.X+frameRows[frameWindow].shadow, body.Max.Y+frameRows[frameWindow].shadow), Art: art}
}

// valid reports whether the art holds all nine pieces.
func (art *DialogFrame) valid() bool {
	if art == nil {
		return false
	}
	for _, p := range art.Pieces {
		if p == nil || p.Bounds().Empty() {
			return false
		}
	}
	return true
}

// panelFrame is the flat panel over r.
func panelFrame(r image.Rectangle, fill, border color.RGBA) frameSpec {
	return frameSpec{Kind: framePanel, Rect: r, Fill: fill, Border: border}
}

// stripFrame is the editor's bar or strip over r: its ground with a gold
// rule on one side.
func stripFrame(r image.Rectangle, side frameSides) frameSpec {
	s := panelFrame(r, editorPanel, editorGold)
	s.Sides = side
	return s
}

func (s frameSpec) row() frameRow {
	if int(s.Kind) < len(frameRows) {
		return frameRows[s.Kind]
	}
	return frameRow{}
}

// rect is the frame's rectangle after the snap rule.
func (s frameSpec) rect() image.Rectangle {
	if s.Snap {
		return newDialogGeometry(s.Rect.Dx(), s.Rect.Dy()).Rectangle
	}
	return s.Rect
}

// body is the rectangle the frame's pieces cover: the frame without its
// shadow band.
func (s frameSpec) body() image.Rectangle {
	r, band := s.rect(), s.row().shadow
	return image.Rect(r.Min.X, r.Min.Y, r.Max.X-band, r.Max.Y-band)
}

// drawFrame is the one frame builder: the mask shadows first, then the body.
func drawFrame(dst *image.RGBA, s frameSpec) {
	recordWidget(widgetFrame, s.rect(), s)
	if dst == nil {
		return
	}
	planFrameShadows(s, image.Point{}, 1, dst.Bounds(), shadowOverPicture, s.Policy.shadowTable).paint(dst, nil)
	drawFrameBody(dst, s)
}

// drawFrameBody draws the frame without its shadows.
func drawFrameBody(dst *image.RGBA, s frameSpec) {
	if dst == nil {
		return
	}
	row, body := s.row(), s.body()
	switch {
	case row.tiled:
		if !s.Art.valid() {
			if s.Kind == frameWindow {
				drawTownShellBox(dst, body, false)
			}
			return
		}
		clip := dialoguePolicyClip(s.Policy, dst.Bounds())
		for _, tile := range frameTiles(s.Art, body, row.cover) {
			pic := s.Art.Pieces[tile.piece]
			r := tile.bounds(pic).Intersect(clip)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					c := pic.RGBAAt(pic.Rect.Min.X+x-tile.at.X, pic.Rect.Min.Y+y-tile.at.Y)
					if c.A != 0 {
						dst.SetRGBA(x, y, c)
					}
				}
			}
		}
	case row.scroll:
		top, paper, bottom := s.Scroll[0], s.Scroll[1], s.Scroll[2]
		if top == nil || paper == nil || bottom == nil || paper.Bounds().Dy() <= 0 {
			return
		}
		mid := image.Rect(body.Min.X, body.Min.Y+frameScrollCap, body.Max.X, body.Max.Y-frameScrollCap)
		for y := mid.Min.Y; y < mid.Max.Y; y += paper.Bounds().Dy() {
			part := image.Rect(mid.Min.X, y, mid.Max.X, min(y+paper.Bounds().Dy(), mid.Max.Y))
			draw.Draw(dst, part, paper, paper.Bounds().Min, draw.Over)
		}
		draw.Draw(dst, image.Rect(body.Min.X, body.Min.Y, body.Max.X, body.Min.Y+frameScrollCap), top, top.Bounds().Min, draw.Over)
		draw.Draw(dst, image.Rect(body.Min.X, body.Max.Y-frameScrollCap, body.Max.X, body.Max.Y), bottom, bottom.Bounds().Min, draw.Over)
	case row.art:
		if s.Art == nil || s.Art.Portrait == nil {
			return
		}
		pic := s.Art.Portrait
		r := body.Intersect(dst.Bounds())
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if c := pic.RGBAAt(pic.Rect.Min.X+x-body.Min.X, pic.Rect.Min.Y+y-body.Min.Y); c.A != 0 {
					dst.SetRGBA(x, y, c)
				}
			}
		}
	case row.corner != image.Point{}:
		if s.Art != nil && s.Art.Portrait != nil {
			drawFrameCorners(dst, body, s.Art.Portrait, row.corner.X, row.corner.Y)
		}
	default:
		r := body.Intersect(dst.Bounds())
		if r.Empty() {
			return
		}
		if row.fill {
			op := draw.Src
			if row.blend {
				op = draw.Over
			}
			draw.Draw(dst, r, &image.Uniform{C: s.Fill}, image.Point{}, op)
		}
		if row.border {
			w, sides := max(s.Width, 1), s.Sides
			if sides == 0 {
				sides = edgeTopBit | edgeBottomBit | edgeLeftBit | edgeRightBit
			}
			ink := &image.Uniform{C: s.Border}
			for _, e := range [...]struct {
				side frameSides
				at   image.Rectangle
			}{
				{edgeTopBit, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+w)},
				{edgeBottomBit, image.Rect(r.Min.X, r.Max.Y-w, r.Max.X, r.Max.Y)},
				{edgeLeftBit, image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Max.Y)},
				{edgeRightBit, image.Rect(r.Max.X-w, r.Min.Y, r.Max.X, r.Max.Y)},
			} {
				if sides&e.side != 0 {
					draw.Draw(dst, e.at, ink, image.Point{}, draw.Src)
				}
			}
		}
	}
}

// DrawPanel draws the flat panel over r: the one frame builder for programs
// outside this package.
func DrawPanel(dst *image.RGBA, r image.Rectangle, fill, border color.RGBA) {
	drawFrame(dst, panelFrame(r, fill, border))
}

// frameStrokeRect is the GPU stroke a frame's border is drawn with.
var frameStrokeRect = vector.StrokeRect

// drawFrameVector draws a flat frame on a GPU destination with the vector
// calls the caller supplies: the fill over the body, then the border
// centred on the pixel ring it covers. It reads the same spec drawFrame
// reads; a caller supplies the calls so tests can observe them.
func drawFrameVector(dst *ebiten.Image, s frameSpec, fill func(*ebiten.Image, float32, float32, float32, float32, color.Color, bool), stroke func(*ebiten.Image, float32, float32, float32, float32, float32, color.Color, bool)) {
	recordWidget(widgetFrame, s.rect(), s)
	row, r := s.row(), s.body()
	if dst == nil || row.tiled || row.corner != (image.Point{}) {
		return
	}
	if row.fill {
		fill(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), s.Fill, false)
	}
	if row.border {
		w := float32(max(s.Width, 1))
		stroke(dst, float32(r.Min.X)+w/2, float32(r.Min.Y)+w/2, float32(r.Dx())-w, float32(r.Dy())-w, w, s.Border, false)
	}
}

// frameShadowLevel is the backdrop level a frame shadow remaps a picture to.
const frameShadowLevel = 6

// shadowTable is the level table a frame shadow remaps through in p's pixel
// format.
func (p DialogueBackdrop) shadowTable() *backdrop.Lookup {
	l, err := backdrop.NewLevel(p.Layout, p.Mode, frameShadowLevel)
	if err != nil {
		panic(err)
	}
	return l
}

// frameShadowBlend selects what one shadow tile does to a pixel it covers.
type frameShadowBlend uint8

const (
	// shadowOverPicture remaps a pixel that holds a picture through the
	// level table and gives an empty pixel the shadow tone: a frame
	// composed apart from the scene behind it.
	shadowOverPicture frameShadowBlend = iota
	// shadowRemap remaps every covered pixel through the level table and
	// keeps its alpha: the dialogue window over its backdrop.
	shadowRemap
)

// frameShadowPlan is one frame's shadow pass: the shadow tiles in frame
// coordinates, the mask counting the tiles over each destination pixel, and
// the blend each tile applies. planFrameShadows builds it; paint runs it on
// the CPU and dialogueBackdropState.drawShadowPlan on the GPU.
type frameShadowPlan struct {
	tiles []frameTile
	mask  *image.RGBA
	table *backdrop.Lookup
	tone  color.RGBA
	toned bool
}

// planFrameShadows is the one producer of a frame shadow: s's shadow tiles
// placed at at and scaled by scale into a destination of bounds, clipped by
// s's policy, blended through the table that table returns. A frame without
// shadows, or whose shadows fall outside the clip, plans no mask.
func planFrameShadows(s frameSpec, at image.Point, scale float64, bounds image.Rectangle, blend frameShadowBlend, table func() *backdrop.Lookup) frameShadowPlan {
	row := s.row()
	if !row.tiled || row.shadow == 0 || !s.Art.valid() || scale <= 0 {
		return frameShadowPlan{}
	}
	p := frameShadowPlan{tiles: frameShadowTiles(s.Art, s.body(), row)}
	p.mask = frameShadowMask(s.Art, p.tiles, at, scale, dialoguePolicyClip(s.Policy, bounds))
	if p.mask == nil {
		return p
	}
	p.table = table()
	if blend == shadowOverPicture {
		p.tone, p.toned = frameShadowTone, true
	}
	return p
}

// paint runs the plan on a CPU destination: a pixel takes the blend once per
// tile over it, and captured text under the shadow takes the same.
func (p frameShadowPlan) paint(dst *image.RGBA, calls []text.DrawCall) {
	if dst == nil || p.mask == nil {
		return
	}
	m := p.mask
	for y := m.Rect.Min.Y; y < m.Rect.Max.Y; y++ {
		for x := m.Rect.Min.X; x < m.Rect.Max.X; x++ {
			n := m.RGBAAt(x, y).A
			if n == 0 {
				continue
			}
			c := dst.RGBAAt(x, y)
			for ; n > 0; n-- {
				if p.toned && c.A == 0 {
					c = p.tone
				} else {
					c = p.table.Color(c)
				}
			}
			dst.SetRGBA(x, y, c)
		}
	}
	remapCapturedMask(calls, m.Rect, p.table, m, m.Rect.Min, true)
}

// frameTile is one blit of one frame piece, clipped to clip when that is
// not empty.
type frameTile struct {
	piece int
	at    image.Point
	clip  image.Rectangle
}

func (t frameTile) bounds(pic *image.RGBA) image.Rectangle {
	r := image.Rectangle{Min: t.at, Max: t.at.Add(pic.Bounds().Size())}
	if !t.clip.Empty() {
		r = r.Intersect(t.clip)
	}
	return r
}

// frameTiles is the tiling for a body (DLG-PANEL-035, MENU-127): four
// corners, the top and bottom edges in whole tiles of pieces 2 and 7, the
// left and right edges in whole tiles of pieces 4 and 5, and the fill in
// whole tiles of piece 0, counted (width-2c)/edge across and
// (height-2c)/edge down. With cover a remainder gets one more tile,
// clipped short of the far corner. A body smaller than two corners has
// none.
func frameTiles(art *DialogFrame, body image.Rectangle, cover bool) []frameTile {
	size := func(i int) image.Point { return art.Pieces[i].Bounds().Size() }
	corner := size(1)
	if body.Dx() < 2*corner.X || body.Dy() < 2*corner.Y {
		return nil
	}
	l, t, r, b := body.Min.X, body.Min.Y, body.Max.X, body.Max.Y
	spanX, spanY := body.Dx()-2*corner.X, body.Dy()-2*corner.Y
	nx, ny := spanX/size(2).X, spanY/size(4).Y
	across, down := image.Rectangle{}, image.Rectangle{}
	if cover && spanX%size(2).X != 0 {
		nx++
		across = image.Rect(l+corner.X, t, r-corner.X, b)
	}
	if cover && spanY%size(4).Y != 0 {
		ny++
		down = image.Rect(l, t+corner.Y, r, b-corner.Y)
	}
	clipOf := func(i, n int, c image.Rectangle) image.Rectangle {
		if i == n-1 {
			return c
		}
		return image.Rectangle{}
	}
	tiles := []frameTile{
		{piece: 1, at: image.Pt(l, t)},
		{piece: 3, at: image.Pt(r-size(3).X, t)},
		{piece: 6, at: image.Pt(l, b-size(6).Y)},
		{piece: 8, at: image.Pt(r-size(8).X, b-size(8).Y)},
	}
	for i := 0; i < nx; i++ {
		c := clipOf(i, nx, across)
		tiles = append(tiles,
			frameTile{2, image.Pt(l+corner.X+i*size(2).X, t), c},
			frameTile{7, image.Pt(l+corner.X+i*size(7).X, b-size(7).Y), c})
	}
	for j := 0; j < ny; j++ {
		c := clipOf(j, ny, down)
		tiles = append(tiles,
			frameTile{4, image.Pt(l, t+corner.Y+j*size(4).Y), c},
			frameTile{5, image.Pt(r-size(5).X, t+corner.Y+j*size(5).Y), c})
	}
	fill := image.Rect(l+corner.X, t+corner.Y, r-corner.X, b-corner.Y)
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			c := image.Rectangle{}
			if !clipOf(i, nx, across).Empty() || !clipOf(j, ny, down).Empty() {
				c = fill
			}
			tiles = append(tiles, frameTile{0, image.Pt(l+corner.X+i*size(0).X, t+corner.Y+j*size(0).Y), c})
		}
	}
	return tiles
}

// frameShadowTiles are the right column and bottom row of the tiling,
// pieces 3, 5, 6, 7 and 8, moved by the shadow band.
func frameShadowTiles(art *DialogFrame, body image.Rectangle, row frameRow) []frameTile {
	var shadows []frameTile
	d := image.Pt(row.shadow, row.shadow)
	for _, tile := range frameTiles(art, body, row.cover) {
		switch tile.piece {
		case 3, 5, 6, 7, 8:
			tile.at = tile.at.Add(d)
			if !tile.clip.Empty() {
				tile.clip = tile.clip.Add(d)
			}
			shadows = append(shadows, tile)
		}
	}
	return shadows
}

// frameShadowMask is the plan's mask: tiles scaled by scale into a
// destination at at and clipped to clip, as the count of shadow pieces
// covering each pixel.
func frameShadowMask(art *DialogFrame, tiles []frameTile, at image.Point, scale float64, clip image.Rectangle) *image.RGBA {
	transform := func(r image.Rectangle) image.Rectangle {
		return image.Rect(at.X+int(math.Floor(float64(r.Min.X)*scale)), at.Y+int(math.Floor(float64(r.Min.Y)*scale)), at.X+int(math.Ceil(float64(r.Max.X)*scale)), at.Y+int(math.Ceil(float64(r.Max.Y)*scale)))
	}
	var bounds image.Rectangle
	for _, tile := range tiles {
		bounds = bounds.Union(transform(tile.bounds(art.Pieces[tile.piece])))
	}
	bounds = bounds.Intersect(clip)
	if bounds.Empty() {
		return nil
	}
	mask := image.NewRGBA(bounds)
	for _, tile := range tiles {
		pic := art.Pieces[tile.piece]
		r := transform(tile.bounds(pic)).Intersect(bounds)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				sx := int(math.Floor((float64(x-at.X)+0.5)/scale)) - tile.at.X
				sy := int(math.Floor((float64(y-at.Y)+0.5)/scale)) - tile.at.Y
				if sx < 0 || sy < 0 || sx >= pic.Rect.Dx() || sy >= pic.Rect.Dy() || pic.RGBAAt(pic.Rect.Min.X+sx, pic.Rect.Min.Y+sy).A == 0 {
					continue
				}
				i := mask.PixOffset(x, y) + 3
				if mask.Pix[i] == 255 {
					panic("dialogue shadow mask overlap exceeds 255")
				}
				mask.Pix[i]++
			}
		}
	}
	return mask
}

// drawFrameCorners draws border's four cw by ch corners unscaled at the
// panel's corners and tiles its straight edges between them.
func drawFrameCorners(dst *image.RGBA, panel image.Rectangle, border image.Image, cw, ch int) {
	if border == nil {
		return
	}
	b := border.Bounds()
	if cw <= 0 || ch <= 0 || 2*cw >= b.Dx() || 2*ch >= b.Dy() || panel.Dx() < 2*cw || panel.Dy() < 2*ch {
		return
	}
	tl := image.Rect(b.Min.X, b.Min.Y, b.Min.X+cw, b.Min.Y+ch)
	tr := image.Rect(b.Max.X-cw, b.Min.Y, b.Max.X, b.Min.Y+ch)
	bl := image.Rect(b.Min.X, b.Max.Y-ch, b.Min.X+cw, b.Max.Y)
	br := image.Rect(b.Max.X-cw, b.Max.Y-ch, b.Max.X, b.Max.Y)
	top := image.Rect(b.Min.X+cw, b.Min.Y, b.Max.X-cw, b.Min.Y+ch)
	bottom := image.Rect(b.Min.X+cw, b.Max.Y-ch, b.Max.X-cw, b.Max.Y)
	left := image.Rect(b.Min.X, b.Min.Y+ch, b.Min.X+cw, b.Max.Y-ch)
	right := image.Rect(b.Max.X-cw, b.Min.Y+ch, b.Max.X, b.Max.Y-ch)

	draw.Draw(dst, image.Rect(panel.Min.X, panel.Min.Y, panel.Min.X+cw, panel.Min.Y+ch), border, tl.Min, draw.Over)
	draw.Draw(dst, image.Rect(panel.Max.X-cw, panel.Min.Y, panel.Max.X, panel.Min.Y+ch), border, tr.Min, draw.Over)
	draw.Draw(dst, image.Rect(panel.Min.X, panel.Max.Y-ch, panel.Min.X+cw, panel.Max.Y), border, bl.Min, draw.Over)
	draw.Draw(dst, image.Rect(panel.Max.X-cw, panel.Max.Y-ch, panel.Max.X, panel.Max.Y), border, br.Min, draw.Over)

	tileBlitOver(dst, image.Rect(panel.Min.X+cw, panel.Min.Y, panel.Max.X-cw, panel.Min.Y+ch), border, top)
	tileBlitOver(dst, image.Rect(panel.Min.X+cw, panel.Max.Y-ch, panel.Max.X-cw, panel.Max.Y), border, bottom)
	tileBlitOver(dst, image.Rect(panel.Min.X, panel.Min.Y+ch, panel.Min.X+cw, panel.Max.Y-ch), border, left)
	tileBlitOver(dst, image.Rect(panel.Max.X-cw, panel.Min.Y+ch, panel.Max.X, panel.Max.Y-ch), border, right)
}
