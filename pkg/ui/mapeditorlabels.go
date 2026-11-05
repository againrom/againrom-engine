package ui

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
)

type mapLabelSource uint8

const (
	labelTrigger mapLabelSource = 1 << iota
	labelRecord
	labelGroup
	labelHeight
	labelHeightGesture
)

type labelPopup uint8

const (
	labelPopupNone labelPopup = iota
	labelPopupOmit
	labelPopupLeft
)

type labelEdge uint8

const (
	labelInside labelEdge = iota
	labelClip
	labelUnbounded
)

type mapLabel struct {
	source          mapLabelSource
	index           int
	anchor, offset  image.Point
	above, centered bool
	text            string
	padding         image.Point
	ink, border     color.RGBA
	popup           labelPopup
	edge            labelEdge
	collides        mapLabelSource
	moveDown        bool
	stopAtBottom    bool
	maxTextWidth    int
	layer           int
	rect            image.Rectangle
}

// mapLabels places source rows in dependency order. Painting uses each row's
// layer instead: heights, records, triggers, then group badges.
func (e *MapEditor) mapLabels() []mapLabel {
	popups := e.popupRects()
	rows := e.triggerLabelRows()
	rows = append(rows, e.markerLabelRows()...)
	rows = append(rows, e.groupLabelRows()...)
	rows = append(rows, e.heightLabelRows()...)
	if len(rows) == 0 {
		return nil
	}
	f, area := e.bodyFont(), e.mapArea()
	var out []mapLabel
	for _, l := range rows {
		if l.maxTextWidth > 0 {
			l.text = editorClip(f, l.text, l.maxTextWidth)
		}
		w, h := 2*l.padding.X, fontHeight(f, 15)+2*l.padding.Y
		if f != nil {
			w += f.Advance(l.text)
		}
		at := l.anchor.Add(l.offset)
		if l.above {
			at.Y -= h
		}
		if l.centered {
			at.X -= w / 2
		}
		r := image.Rectangle{Min: at, Max: at.Add(image.Pt(w, h))}
		if l.popup == labelPopupLeft {
			for _, p := range popups {
				if p.Overlaps(r) && r.Min.X < p.Min.X {
					r = r.Sub(image.Pt(r.Max.X-p.Min.X, 0))
				}
			}
		}
		collision := false
		for j := 0; j < len(out) && (!l.stopAtBottom || r.Min.Y < area.Max.Y); {
			if l.collides&out[j].source != 0 && r.Overlaps(out[j].rect) {
				if !l.moveDown {
					collision = true
					break
				}
				r = r.Add(image.Pt(0, h))
				j = 0
			} else {
				j++
			}
		}
		covered := false
		if l.popup != labelPopupNone {
			for _, p := range popups {
				covered = covered || p.Overlaps(r)
			}
		}
		if collision || covered || l.edge == labelInside && !r.In(area) || l.edge == labelClip && !r.Overlaps(area) {
			continue
		}
		l.rect = r
		out = append(out, l)
	}
	return out
}

func (e *MapEditor) mapLabelsOf(source mapLabelSource) []mapLabel {
	var out []mapLabel
	for _, l := range e.mapLabels() {
		if l.source == source {
			out = append(out, l)
		}
	}
	return out
}

func (e *MapEditor) drawMapLabels(pic *image.RGBA) {
	labels := e.mapLabels()
	slices.SortStableFunc(labels, func(a, b mapLabel) int { return a.layer - b.layer })
	f := e.bodyFont()
	for _, l := range labels {
		clip := l.rect
		if l.edge == labelClip {
			clip = clip.Intersect(e.mapArea())
		}
		draw.Draw(pic, clip, image.NewUniform(editorInk), image.Point{}, draw.Src)
		if l.border.A != 0 {
			drawFrame(pic, panelFrame(l.rect, editorInk, l.border))
		}
		if f == nil {
			continue
		}
		x, y := l.rect.Min.X+l.padding.X, l.rect.Min.Y+l.padding.Y
		label := l.text
		if l.edge == labelClip {
			if y < clip.Min.Y || y+f.Height() > clip.Max.Y {
				continue
			}
			for len(label) > 0 && x < clip.Min.X {
				x += f.Advance(label[:1])
				label = label[1:]
			}
			label = editorClip(f, label, clip.Max.X-x)
		}
		f.Draw(pic.SubImage(clip).(*image.RGBA), label, x, y, l.ink)
	}
}
