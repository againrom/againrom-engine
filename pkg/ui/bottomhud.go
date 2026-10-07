package ui

import (
	"image"
	"image/draw"
)

// BottomHUDArt is immutable installed art for the mission book and pack.
// The wiring tier decodes it once; no archive is read during presentation.
type BottomHUDArt struct {
	Book, UnknownSpell  *image.RGBA
	BookLeft, BookRight [2]*image.RGBA // 800x600, 1024x768
	Pack, PackItem      *image.RGBA
	PackLeft, PackRight *image.RGBA
	PackArrow           [4]*image.RGBA // left/right, normal/pressed
}

func (v *Viewer) SetBottomHUDArt(art *BottomHUDArt) {
	v.bottomHUDArt = art
}

// bookAtlasX centres the 480-pixel atlas in the map band. The right wing
// overlaps its 16-pixel closing seam, as in the owner's original screenshot.
func bookAtlasX(bar image.Rectangle) int {
	return bar.Min.X + (bar.Dx()-bookBarW)/2
}

func drawBookGround(dst *image.RGBA, art *BottomHUDArt) {
	x := bookAtlasX(dst.Bounds())
	if x > 0 {
		family := 0
		if x > 80 {
			family = 1
		}
		// The mission frame is 1024x768. Tiling also keeps custom wider
		// viewer frames covered without scaling a spell or changing its ID.
		tileHUDStrip(dst, image.Rect(0, 0, x, bookBarH), art.BookLeft[family])
		tileHUDRightStrip(dst, image.Rect(x+bookBarW-16, 0, dst.Bounds().Max.X, bookBarH), art.BookRight[family])
	}
	blit(dst, art.Book, x, 0)
}

// drawPackGround repeats the frame's complete 80-pixel bays, including the
// gold edges above and below each empty slot. Cell origins stay those of
// ITEM-STARCOMP-100; the end strips remain the existing scroll hit targets.
func drawPackGround(dst *image.RGBA, bar image.Rectangle, cols int, art *BottomHUDArt) {
	cells := packCellRects(bar, cols)
	if len(cells) == 0 {
		return
	}
	x0, x1 := cells[0].Min.X, cells[len(cells)-1].Max.X
	tileHUDStrip(dst, image.Rect(0, 0, x0-32, packGridH), art.PackLeft)
	for i, cell := range cells {
		// Preserve the two edge bays on each side and repeat the middle
		// bay. The source shading and corner details differ between bays.
		source := i
		if i >= 2 {
			source = 2
			if i >= cols-2 {
				source = 5 - (cols - i)
			}
		}
		draw.Draw(dst, image.Rect(cell.Min.X, 0, cell.Max.X, packGridH), art.Pack, image.Pt(32+80*source, 0), draw.Over)
	}
	draw.Draw(dst, image.Rect(x0-32, 0, x0, packGridH), art.Pack, image.Point{}, draw.Over)
	draw.Draw(dst, image.Rect(x1, 0, x1+48, packGridH), art.Pack, image.Pt(432, 0), draw.Over)
	if x0 > 32 {
		tileHUDRightStrip(dst, image.Rect(x1+32, 0, bar.Max.X, packGridH), art.PackRight)
	}
}

func tileHUDStrip(dst *image.RGBA, box image.Rectangle, src *image.RGBA) {
	if src == nil || src.Bounds().Dx() == 0 {
		return
	}
	for x := box.Min.X; x < box.Max.X; x += src.Bounds().Dx() {
		r := image.Rect(x, box.Min.Y, x+src.Bounds().Dx(), box.Max.Y).Intersect(box)
		draw.Draw(dst, r, src, src.Bounds().Min, draw.Over)
	}
}

// Repeat only the opaque body; the transparent 16-pixel closing seam belongs
// at the outer edge, never in the middle of a widescreen extension.
func tileHUDRightStrip(dst *image.RGBA, box image.Rectangle, src *image.RGBA) {
	if src == nil || src.Bounds().Dx() <= 16 || box.Dx() < 16 {
		return
	}
	b := src.Bounds()
	tileHUDStrip(dst, image.Rect(box.Min.X, box.Min.Y, box.Max.X-16, box.Max.Y), src.SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X-16, b.Max.Y)).(*image.RGBA))
	draw.Draw(dst, image.Rect(box.Max.X-16, box.Min.Y, box.Max.X, box.Max.Y), src, image.Pt(b.Max.X-16, b.Min.Y), draw.Over)
}
