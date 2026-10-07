package video

import "image"

const (
	// regionW and regionH are the output region every movie is laid out in.
	regionW, regionH = 640, 360
	// tallMovie is the height whose movie takes its own size as the region.
	tallMovie = 480
)

// presenter turns one decoded frame (palette-indexed plane plus palette) into
// the frame the player blits: the sidecar's palette fade, source origin and
// pan applied, windowed to the output region and doubled when the movie is
// small (VIDEO-071, VIDEO-072). Frames are presented in order, one call each.
type presenter struct {
	w, h         int
	sc           Sidecar
	scale        int
	winW, winH   int
	out          *image.RGBA
	frame        int32
	frames       int32
	srcX, srcY   int32
	fadeIdx      int
	fadeActive   bool
	fadeRemain   int32
	fadeFactor   float32
	fadeDelta    float32
	panIdx       int
	stepX, stepY int32
	scaled       [256][3]byte
	shown        [256][3]byte
}

// presentedSize reports the dimensions of the frames a movie of width w and
// height h is presented at: the output region, doubled for a small movie, or
// the movie's own size for a 480 high one.
func presentedSize(w, h int) (outW, outH, scale int) {
	winW, winH, scale := presentedWindow(w, h)
	return winW * scale, winH * scale, scale
}

func presentedWindow(w, h int) (winW, winH, scale int) {
	doubled := !(2*w > regionW && 2*h > regionH)
	winW, winH, scale = regionW, regionH, 1
	if doubled {
		winW, winH, scale = regionW/2, regionH/2, 2
	}
	if h == tallMovie {
		winW, winH, scale = w, tallMovie, 1
	}
	return
}

func newPresenter(w, h, frames int, sc *Sidecar) *presenter {
	p := &presenter{w: w, h: h, frames: int32(frames)}
	if sc != nil {
		p.sc = *sc
	}
	p.winW, p.winH, p.scale = presentedWindow(w, h)
	p.srcX, p.srcY = p.sc.StartX, p.sc.StartY
	p.out = image.NewRGBA(image.Rect(0, 0, p.winW*p.scale, p.winH*p.scale))
	return p
}

// present renders the next frame. The returned image is reused by the next
// call.
func (p *presenter) present(indices []byte, palette [256][3]byte, newPalette bool) *image.RGBA {
	// The decoder sets its new-palette flag when it opens, so the first frame
	// always sets the display palette (VIDEO-084).
	if p.frame == 0 {
		newPalette = true
	}
	p.armFade()
	p.stepPan()
	// The display palette is set only by an active fade or by a frame that
	// carries a new palette; otherwise the last one set stays (VIDEO-071
	// step 4, reading the decoder's new-palette flag as the gate).
	if p.fadeActive {
		p.applyFade(&palette)
		p.shown = p.scaled
	} else if newPalette {
		p.shown = palette
	}
	p.blit(indices, &p.shown)
	p.frame++
	if p.frame < p.frames {
		p.srcX += p.stepX
		p.srcY += p.stepY
	}
	return p.out
}

// armFade starts the next fade record when its start frame is the current
// frame. Records are consumed in order; a record whose start is already past
// never matches. A record with its end before or at its start has no span to
// fade over and is consumed without effect.
func (p *presenter) armFade() {
	if p.fadeIdx >= len(p.sc.Fades) {
		return
	}
	r := p.sc.Fades[p.fadeIdx]
	if r.Start != p.frame {
		return
	}
	p.fadeIdx++
	if r.End <= r.Start {
		return
	}
	span := r.End - r.Start
	p.fadeActive = true
	p.fadeRemain = span
	p.fadeFactor = r.From
	p.fadeDelta = (r.To - r.From) / float32(span)
}

func (p *presenter) applyFade(src *[256][3]byte) {
	p.fadeFactor += p.fadeDelta
	for i := range src {
		for c := 0; c < 3; c++ {
			p.scaled[i][c] = byte(int32(float32(src[i][c]) * p.fadeFactor))
		}
	}
	p.fadeRemain--
	if p.fadeRemain <= 0 {
		p.fadeActive = false
	}
}

// stepPan loads or clears the pan step for the current frame.
func (p *presenter) stepPan() {
	if p.panIdx >= len(p.sc.Pans) {
		return
	}
	r := p.sc.Pans[p.panIdx]
	if r.Start == p.frame {
		p.stepX, p.stepY = r.StepX, r.StepY
	}
	if r.End == p.frame {
		p.stepX, p.stepY = 0, 0
		p.panIdx++
	}
}

// blit copies the source window at the current origin into the output,
// doubling each source pixel when the movie is small. A source position
// outside the movie leaves its output pixel black.
func (p *presenter) blit(indices []byte, pal *[256][3]byte) {
	out := p.out
	for oy := 0; oy < p.winH*p.scale; oy++ {
		sy := int64(p.srcY) + int64(oy/p.scale)
		row := out.Pix[oy*out.Stride : oy*out.Stride+out.Rect.Dx()*4]
		for ox := 0; ox < p.winW*p.scale; ox++ {
			sx := int64(p.srcX) + int64(ox/p.scale)
			d := ox * 4
			if sx < 0 || sy < 0 || sx >= int64(p.w) || sy >= int64(p.h) {
				row[d], row[d+1], row[d+2], row[d+3] = 0, 0, 0, 255
				continue
			}
			c := pal[indices[int(sy)*p.w+int(sx)]]
			row[d], row[d+1], row[d+2], row[d+3] = c[0], c[1], c[2], 255
		}
	}
}
