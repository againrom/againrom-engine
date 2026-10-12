package town

import (
	"image"
	"image/draw"
)

// Size is the scene's view size.
func (s *Scene) Size() image.Point { return s.spec.View.Size.Pt() }

// Paint draws the layers of one group in order onto dst, whose origin is the
// scene's; the square's layers are the group with no name. A layer whose
// condition fails is skipped. Paint changes no state; Advance is the clock.
func (s *Scene) Paint(dst *image.RGBA, group string) {
	if s.host.Art() == nil {
		return
	}
	for _, layer := range s.spec.Layers {
		if layer.Group != group || !s.when(layer.When) {
			continue
		}
		if layer.Actor != "" {
			if a := s.byName[layer.Actor]; a != nil {
				a.draw(s, dst, layer)
			}
			continue
		}
		place(dst, frameAt(s.frames(layer.Art), 0), layer)
	}
}

func (s *Scene) when(w *WhenSpec) bool {
	if w == nil {
		return true
	}
	if len(w.Any) > 0 {
		for i := range w.Any {
			if s.when(&w.Any[i]) {
				return true
			}
		}
		return false
	}
	if w.Running != "" {
		a := s.byName[w.Running]
		if a == nil || !a.runsSheet(w.Running, *w.Sheet) {
			return false
		}
	}
	if w.Hover != "" && (s.hover == nil || s.hover.Name != w.Hover) {
		return false
	}
	if w.If != "" && !s.host.Condition(w.If) {
		return false
	}
	if w.Active != "" {
		a := s.byName[w.Active]
		if a == nil || !a.activeNow() {
			return false
		}
	}
	return true
}

// place draws pic with its top-left at the layer's point, copied or
// composited over by the layer's mode, inside the layer's clip when it has
// one.
func place(dst *image.RGBA, pic image.Image, layer LayerSpec) {
	if pic == nil {
		return
	}
	op := draw.Over
	if layer.Mode == "copy" {
		op = draw.Src
	}
	b := pic.Bounds()
	placed := b.Add(layer.At.Pt().Sub(b.Min))
	if layer.Clip == nil {
		draw.Draw(dst, placed, pic, b.Min, op)
		return
	}
	clip := placed.Intersect(layer.Clip.Rectangle())
	if clip.Empty() {
		return
	}
	draw.Draw(dst, clip, pic, b.Min.Add(clip.Min.Sub(placed.Min)), op)
}
