package mapedit

import (
	"bytes"
	"fmt"
	"image"
)

// RectangleBrush paints one structure kind as a union of cells. Minimum is
// both the smallest stamp and stored rectangle; MaxSpan is the field bound.
// Delete uses the host's record deletion rule, including attached records.
// Fenced holds cells of other structure kinds resolved by the host.
type RectangleBrush struct {
	Kind    uint32
	Owner   uint16
	Minimum image.Point
	MaxSpan int
	Delete  func(*Editor, int) error
	Fenced  []bool
}

type rectangleRecord struct {
	index int
	rect  image.Rectangle
	x, y  uint32
	met   bool
}

func (e *Editor) rectangleStroke(b Brush, at []Cell, w, h int) (edit, error) {
	p := b.Rectangles
	if p.Minimum.X < 1 || p.Minimum.Y < 1 || p.MaxSpan < max(p.Minimum.X, p.Minimum.Y) || (b.Value != 0 && b.Value != 1) {
		return nil, fmt.Errorf("mapedit: invalid rectangular structure brush")
	}
	if p.Fenced != nil && len(p.Fenced) != w*h {
		return nil, fmt.Errorf("mapedit: a structure plane of %d cells for a %dx%d grid", len(p.Fenced), w, h)
	}
	bounds := image.Rect(0, 0, w, h)
	touched, cells := make([]bool, w*h), make([]bool, w*h)
	for _, c := range b.Footprint(at, w, h) {
		touched[c.Y*w+c.X] = true
	}
	var records []rectangleRecord
	for i := 0; i < e.Count(KindStructure); i++ {
		kind, err := e.FieldValue(KindStructure, i, "kind")
		if err != nil {
			return nil, err
		}
		if uint32(kind) != p.Kind {
			continue
		}
		width, err := e.FieldValue(KindStructure, i, "width")
		if err != nil {
			return nil, err
		}
		height, err := e.FieldValue(KindStructure, i, "height")
		if err != nil {
			return nil, err
		}
		x, y, err := e.Position(KindStructure, i)
		if err != nil {
			return nil, err
		}
		origin := image.Pt(int(x>>8), int(y>>8))
		records = append(records, rectangleRecord{index: i, rect: image.Rectangle{Min: origin, Max: origin.Add(image.Pt(max(1, int(width)), max(1, int(height))))}, x: x, y: y})
	}
	fenced := make([]bool, w*h)
	copy(fenced, p.Fenced)
	for i := range records {
		r := &records[i]
		near := r.rect.Inset(-1).Intersect(bounds)
		for y := near.Min.Y; y < near.Max.Y && !r.met; y++ {
			for x := near.Min.X; x < near.Max.X; x++ {
				if touched[y*w+x] {
					r.met = true
					break
				}
			}
		}
		inside := r.rect.Intersect(bounds)
		for y := inside.Min.Y; y < inside.Max.Y; y++ {
			for x := inside.Min.X; x < inside.Max.X; x++ {
				if r.met {
					cells[y*w+x] = true
				} else {
					fenced[y*w+x] = true
				}
			}
		}
	}
	for _, c := range b.Footprint(at, w, h) {
		cells[c.Y*w+c.X] = b.Value != 0
	}
	for i := range cells {
		cells[i] = cells[i] && !fenced[i]
	}
	rects := rectangleCover(cells, w, h, p.Minimum, p.MaxSpan)
	used := make([]bool, len(rects))
	assigned := make([]int, len(records))
	for i := range assigned {
		assigned[i] = -1
	}
	for i, r := range records {
		if !r.met {
			continue
		}
		for j, rect := range rects {
			if !used[j] && rect == r.rect {
				assigned[i], used[j] = j, true
				break
			}
		}
	}
	for i, r := range records {
		if !r.met || assigned[i] >= 0 {
			continue
		}
		for j, rect := range rects {
			if !used[j] && !rect.Intersect(r.rect).Empty() {
				assigned[i], used[j] = j, true
				break
			}
		}
	}
	next := e.Clone()
	mark := next.Mark()
	for i, r := range records {
		if assigned[i] < 0 {
			continue
		}
		rect := rects[assigned[i]]
		if rect.Min != r.rect.Min {
			if err := next.Move(KindStructure, r.index, uint32(rect.Min.X)<<8|r.x&255, uint32(rect.Min.Y)<<8|r.y&255); err != nil {
				return nil, err
			}
		}
		for _, f := range []FieldSet{{"width", int64(rect.Dx())}, {"height", int64(rect.Dy())}} {
			if err := next.SetField(KindStructure, r.index, f.Key, f.Value); err != nil {
				return nil, err
			}
		}
	}
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if !r.met || assigned[i] >= 0 {
			continue
		}
		var err error
		if p.Delete != nil {
			err = p.Delete(next, r.index)
		} else {
			err = next.Delete(KindStructure, r.index)
		}
		if err != nil {
			return nil, err
		}
	}
	for j, rect := range rects {
		if used[j] {
			continue
		}
		rec, err := next.NewStructureRecord(uint32(rect.Min.X)<<8|128, uint32(rect.Min.Y)<<8|128, p.Kind, p.Owner)
		if err != nil {
			return nil, err
		}
		i, err := next.Place(KindStructure, rec)
		if err != nil {
			return nil, err
		}
		if err := next.SetFields(KindStructure, i, FieldSet{"width", int64(rect.Dx())}, FieldSet{"height", int64(rect.Dy())}); err != nil {
			return nil, err
		}
	}
	if bytes.Equal(e.data, next.data) {
		return nil, nil
	}
	next.Squash(mark)
	return next.log[next.cursor-1], nil
}

// rectangleCover scans minimum-size patches in row order. A full unassigned
// patch grows right, then down, up to limit, using only unassigned cells.
// A short terminal strip borrows from the preceding span when possible.
// Cells no remaining minimum-size patch can hold are dropped.
func rectangleCover(cells []bool, w, h int, minimum image.Point, limit int) []image.Rectangle {
	covered := make([]bool, len(cells))
	full := func(x, y, width, height int) bool {
		for dy := 0; dy < height; dy++ {
			for dx := 0; dx < width; dx++ {
				i := (y+dy)*w + x + dx
				if !cells[i] || covered[i] {
					return false
				}
			}
		}
		return true
	}
	var out []image.Rectangle
	for y := 0; y+minimum.Y <= h; y++ {
		for x := 0; x+minimum.X <= w; x++ {
			if !full(x, y, minimum.X, minimum.Y) {
				continue
			}
			run := minimum.X
			for x+run < w && full(x+run, y, 1, minimum.Y) {
				run++
			}
			width := min(run, limit)
			if tail := run - width; tail > 0 && tail < minimum.X && run-minimum.X >= minimum.X {
				width = run - minimum.X
			}
			run = minimum.Y
			for y+run < h && full(x, y+run, width, 1) {
				run++
			}
			height := min(run, limit)
			if tail := run - height; tail > 0 && tail < minimum.Y && run-minimum.Y >= minimum.Y {
				height = run - minimum.Y
			}
			r := image.Rect(x, y, x+width, y+height)
			out = append(out, r)
			for yy := y; yy < r.Max.Y; yy++ {
				for xx := x; xx < r.Max.X; xx++ {
					covered[yy*w+xx] = true
				}
			}
		}
	}
	return out
}
