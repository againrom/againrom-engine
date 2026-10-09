package ui

import (
	"image"
	"image/draw"
)

// sliderKnobHit is the knob's hit width (MENU-118).
const sliderKnobHit = 16

// hSlider is the shared horizontal slider (MENU-117, MENU-118): rectangle,
// position 0..Max, the endcap states the pointer writes, and the enabled
// flag. The knob is one fixed 24x24 frame.
type hSlider struct {
	Rect              image.Rectangle
	Pos, Max          int
	LeftHot, RightHot bool
	Disabled          bool
}

// span is the knob's travel, W-2H-4.
func (s hSlider) span() int { return s.Rect.Dx() - 2*s.Rect.Dy() - 4 }

// knobLeft is the hit-left K=L+H-4+trunc(p*(W-2H-4)/N), or L+H-4 when N is 0.
func (s hSlider) knobLeft() int {
	k := s.Rect.Min.X + s.Rect.Dy() - 4
	if s.Max != 0 {
		k += s.Pos * s.span() / s.Max
	}
	return k
}

// Knob is the knob's 16-pixel hit rectangle; its body is drawn at (K+1,T).
func (s hSlider) Knob() image.Rectangle {
	k := s.knobLeft()
	return image.Rect(k, s.Rect.Min.Y, k+sliderKnobHit, s.Rect.Max.Y)
}

// left and right are the endcap regions: one height from the left edge, and
// the height minus 4 from the right, mirroring the vertical bar (MENU-119).
func (s hSlider) left() image.Rectangle {
	return image.Rect(s.Rect.Min.X, s.Rect.Min.Y, s.Rect.Min.X+s.Rect.Dy(), s.Rect.Max.Y)
}

func (s hSlider) right() image.Rectangle {
	return image.Rect(s.Rect.Max.X-(s.Rect.Dy()-4), s.Rect.Min.Y, s.Rect.Max.X, s.Rect.Max.Y)
}

// posAt maps a pointer column to clamp(trunc(N*(x-L-H-2)/(W-2H-4)),0,N).
func (s hSlider) posAt(x int) int {
	span := s.span()
	if span <= 0 {
		return 0
	}
	return min(max(s.Max*(x-s.Rect.Min.X-s.Rect.Dy()-2)/span, 0), s.Max)
}

// step is the endcap and arrow-key step, max(trunc(N/16),1).
func (s hSlider) step() int { return max(s.Max/16, 1) }

// withPointer sets the endcap states from pointer membership.
func (s hSlider) withPointer(p image.Point, ok bool) hSlider {
	s.LeftHot = ok && p.In(s.left())
	s.RightHot = ok && p.In(s.right())
	return s
}

// drawHSlider paints left cap, track tiles, right cap and knob, each a
// shadowed sprite part, then remaps a disabled slider. Without art it paints
// a plain fallback of the same geometry.
func drawHSlider(dst *image.RGBA, frames []*image.RGBA, s hSlider) {
	recordWidget(widgetSlider, s.Rect, s)
	r := s.Rect
	if dst == nil || r.Empty() {
		return
	}
	knob := image.Rect(s.knobLeft()+1, r.Min.Y, s.knobLeft()+1+widgetSpriteSize, r.Min.Y+widgetSpriteSize)
	if !scrollArt(frames) {
		draw.Draw(dst, r, &image.Uniform{C: helpBarTrackInk}, image.Point{}, draw.Src)
		draw.Draw(dst, knob.Intersect(r), &image.Uniform{C: helpBarKnobInk}, image.Point{}, draw.Src)
	} else {
		h := r.Dy()
		left, right := frames[sliderLeftFrame], frames[sliderRightFrame]
		if s.LeftHot {
			left = frames[sliderLeftHotFrame]
		}
		if s.RightHot {
			right = frames[sliderRightHotFrame]
		}
		clip := dst.Bounds()
		drawWidgetSprite(dst, left, r.Min, clip)
		track := image.Rect(r.Min.X+h, r.Min.Y, r.Max.X-h, r.Max.Y)
		tiles := image.Rect(track.Min.X, track.Min.Y, track.Max.X, track.Max.Y+widgetShadowOffset.Y)
		for x := track.Min.X; x < track.Max.X; x += widgetSpriteSize {
			drawWidgetSprite(dst, frames[sliderTrackFrame], image.Pt(x, r.Min.Y), tiles)
		}
		drawWidgetSprite(dst, right, image.Pt(r.Max.X-h, r.Min.Y), clip)
		drawWidgetSprite(dst, frames[sliderKnobFrame], knob.Min, clip)
	}
	if s.Disabled {
		remapDisabled(dst, r)
	}
}

// sliderInput is one slider's held gesture: an endcap press steps and
// repeats on the held-button timing; a track press maps x and grabs the
// knob when the point lies in the moved knob; a grabbed knob follows x.
type sliderInput struct {
	held   int
	drag   bool
	repeat chargenRepeat
}

// step runs one tick and returns the slider's new position and whether this
// tick set one.
func (in *sliderInput) step(s hSlider, p image.Point, ok bool, ev appInput) (int, bool) {
	if ev.PrimaryPressed {
		*in = sliderInput{}
		if !ok || !p.In(s.Rect) || s.Disabled {
			return s.Pos, false
		}
		in.repeat.tick(ev)
		switch {
		case p.In(s.left()):
			in.held = -1
			return max(s.Pos-s.step(), 0), true
		case p.In(s.right()):
			in.held = 1
			return min(s.Pos+s.step(), s.Max), true
		}
		s.Pos = s.posAt(p.X)
		in.drag = p.In(s.Knob())
		return s.Pos, true
	}
	if in.held == 0 && !in.drag {
		return s.Pos, false
	}
	if ev.PrimaryReleased || !ev.Viewer.PrimaryDown {
		in.held, in.drag = 0, false
		return s.Pos, false
	}
	if in.drag {
		if !ok {
			return s.Pos, false
		}
		return s.posAt(p.X), true
	}
	if in.repeat.tick(ev) && ok {
		if in.held < 0 && p.In(s.left()) {
			return max(s.Pos-s.step(), 0), true
		}
		if in.held > 0 && p.In(s.right()) {
			return min(s.Pos+s.step(), s.Max), true
		}
	}
	return s.Pos, false
}

// active reports whether the slider owns the held button.
func (in sliderInput) active() bool { return in.held != 0 || in.drag }
