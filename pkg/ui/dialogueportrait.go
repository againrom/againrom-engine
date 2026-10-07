package ui

import (
	"againrom/pkg/render/backdrop"
	"image"
)

// Supplied RGBA canvases use top-down physical rows. The background adapter
// supplies bottom-up rows before the selected suffix reverses them. The final
// keyed surface copy descends rows; native upstream canvas exposure is Unknown.
func drawDialoguePortrait(dst *image.RGBA, l NoticeLayout, face *image.RGBA) {
	p := l.Portrait
	pane := image.Rect(p.Min.X, p.Min.Y, p.Min.X+noticeSurfaceW, p.Min.Y+noticeSurfaceH)
	fill := image.Rect(p.Min.X+noticeFillX, p.Min.Y+noticeFillY, p.Min.X+noticeFillX+noticeFillW, p.Min.Y+noticeFillY+noticeFillH)
	if l.Frame == nil || l.Frame.Portrait == nil {
		fill = pane
	}
	for y := fill.Min.Y; y < fill.Max.Y; y++ {
		for x := fill.Min.X; x < fill.Max.X; x++ {
			c := l.PortraitFill
			if l.Frame == nil && (x == pane.Min.X || y == pane.Min.Y || x == pane.Max.X-1 || y == pane.Max.Y-1) {
				c = l.PortraitBorder
			}
			dst.SetRGBA(x, y, c)
		}
	}
	surface := make([]uint16, noticeSurfaceW*noticeSurfaceH)
	copyWindow := func(pic *image.RGBA, words []uint16, keyed bool) {
		if pic == nil || l.PortraitWindow.Empty() {
			return
		}
		b := pic.Bounds()
		w := l.PortraitWindow.Sub(b.Min)
		source := dialoguePortraitSourceWindow(w, b.Dy())
		copyDialoguePortraitWords(surface, noticeSurfaceW, noticeSurfaceH, noticeSurfaceW, words, b.Dx(), b.Dy(), b.Dx(), l.PortraitAt, source, image.Rect(0, 0, noticeSurfaceW, noticeSurfaceH), keyed)
	}
	var background *image.RGBA
	var backgroundWords []uint16
	if l.Frame != nil && l.Frame.PortraitBack != nil {
		background = l.Frame.PortraitBack
		backgroundWords = dialoguePortraitWords(background, l.DialogueBackdrop.Layout, true)
		reverseDialoguePortraitRows(backgroundWords, background.Bounds().Dx(), background.Bounds().Dy())
		copyWindow(background, backgroundWords, false)
	}
	if face != nil {
		copyWindow(face, dialoguePortraitWords(face, l.DialogueBackdrop.Layout, false), true)
	}
	if background != nil {
		reverseDialoguePortraitRows(backgroundWords, background.Bounds().Dx(), background.Bounds().Dy())
	}
	if l.Frame != nil && l.Frame.Portrait != nil {
		border := l.Frame.Portrait
		for y := 0; y < min(border.Bounds().Dy(), noticeSurfaceH); y++ {
			for x := 0; x < min(border.Bounds().Dx(), noticeSurfaceW); x++ {
				c := border.RGBAAt(border.Bounds().Min.X+x, border.Bounds().Min.Y+y)
				if c.A != 0 {
					surface[y*noticeSurfaceW+x] = dialoguePackedWord(c, l.DialogueBackdrop.Layout)
				}
			}
		}
	}
	screen := make([]uint16, noticeSurfaceW*noticeSurfaceH)
	copyDialoguePortraitWords(screen, noticeSurfaceW, noticeSurfaceH, noticeSurfaceW, surface, noticeSurfaceW, noticeSurfaceH, noticeSurfaceW, image.Point{}, image.Rect(0, 0, noticeSurfaceW, noticeSurfaceH), image.Rect(0, 0, noticeSurfaceW, noticeSurfaceH), true)
	for y := 0; y < noticeSurfaceH; y++ {
		for x := 0; x < noticeSurfaceW; x++ {
			word := screen[y*noticeSurfaceW+x]
			if word != 0 {
				dst.SetRGBA(pane.Min.X+x, pane.Min.Y+y, dialoguePackedColor(word, l.DialogueBackdrop.Layout))
			}
		}
	}
}

func dialoguePortraitSourceWindow(window image.Rectangle, height int) image.Rectangle {
	return image.Rect(window.Min.X, height-window.Max.Y, window.Max.X, height-window.Min.Y)
}

func dialoguePortraitWords(pic *image.RGBA, layout backdrop.Layout, reversed bool) []uint16 {
	b := pic.Bounds()
	out := make([]uint16, b.Dx()*b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := pic.RGBAAt(b.Min.X+x, b.Min.Y+y)
			if c.A == 0 {
				continue
			}
			row := y
			if reversed {
				row = b.Dy() - 1 - y
			}
			out[row*b.Dx()+x] = dialoguePackedWord(c, layout)
		}
	}
	return out
}

func reverseDialoguePortraitRows(words []uint16, width, height int) {
	for y := 0; y < height/2; y++ {
		for x := 0; x < width; x++ {
			a, b := y*width+x, (height-1-y)*width+x
			words[a], words[b] = words[b], words[a]
		}
	}
}

func copyDialoguePortraitWords(dst []uint16, dw, dh, dp int, src []uint16, sw, sh, sp int, at image.Point, window, clip image.Rectangle, keyed bool) bool {
	if dw < 0 || dh < 0 || dp < dw || dp < 0 || sw < 0 || sh < 0 || sp < sw || sp < 0 || dh > len(dst)/max(dp, 1) || sh > len(src)/max(sp, 1) {
		return false
	}
	request := image.Rectangle{Min: at, Max: at.Add(window.Size())}.Intersect(clip).Intersect(image.Rect(0, 0, dw, dh))
	for y := request.Min.Y; y < request.Max.Y; y++ {
		for x := request.Min.X; x < request.Max.X; x++ {
			sx := window.Min.X + x - at.X
			sy := sh - 1 - window.Min.Y - (y - at.Y)
			if sx < 0 || sx >= sw || sy < 0 || sy >= sh {
				continue
			}
			word := src[sy*sp+sx]
			if !keyed || word != 0 {
				dst[y*dp+x] = word
			}
		}
	}
	return true
}

func drawFlippedBorder(dst *image.RGBA, surface image.Rectangle, border *image.RGBA) {
	if border == nil {
		return
	}
	b := border.Bounds()
	for y := 0; y < min(b.Dy(), surface.Dy()); y++ {
		for x := 0; x < min(b.Dx(), surface.Dx()); x++ {
			c := border.RGBAAt(b.Min.X+x, b.Min.Y+y)
			if c.A != 0 {
				dst.SetRGBA(surface.Min.X+x, surface.Max.Y-1-y, c)
			}
		}
	}
}
