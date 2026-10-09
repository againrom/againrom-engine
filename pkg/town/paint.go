package town

import (
	"image"
	"image/draw"
)

// Size is the view's size.
func (v *View) Size() image.Point { return v.desc.View.Size.Pt() }

// Paint draws every layer in order onto dst, whose origin is the view's.
// Paint changes no state; Advance is the clock.
func (v *View) Paint(dst *image.RGBA) {
	for _, layer := range v.desc.Layers {
		if !v.when(layer.When) {
			continue
		}
		if layer.Actor != "" {
			if a := v.byName[layer.Actor]; a != nil {
				a.draw(v, dst, layer.Actor, layer)
			}
			continue
		}
		put(dst, frameAt(v.frames(layer.Art), 0), layer)
	}
}

func (v *View) when(w *WhenSpec) bool {
	if w == nil {
		return true
	}
	if w.Hover != "" && (v.hover == nil || v.hover.Name != w.Hover) {
		return false
	}
	if w.Active != "" {
		a := v.byName[w.Active]
		if a == nil || !a.activeNow() {
			return false
		}
	}
	return true
}

// put draws pic with its top-left at the layer's point, copied or composited
// over by the layer's mode.
func put(dst *image.RGBA, pic image.Image, layer LayerSpec) {
	if pic == nil {
		return
	}
	op := draw.Over
	if layer.Mode == "copy" {
		op = draw.Src
	}
	b := pic.Bounds()
	draw.Draw(dst, b.Add(layer.At.Pt().Sub(b.Min)), pic, b.Min, op)
}
