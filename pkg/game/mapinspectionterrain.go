package game

import (
	"fmt"
	"image"

	"againrom/pkg/mapedit"
	"againrom/pkg/random"
	"againrom/pkg/ui"
)

// seedSource is the random source a whole-map command draws from for seed:
// the brush's own seeded source, so one seed on one map gives one map.
func seedSource(seed uint32) func(int) int { return random.NewGo(int64(seed)).Intn }

// RandomTerrain draws every tile word anew among its look-alikes from seed,
// as one edit (TERR-248).
func (s *mapInspectionEdits) RandomTerrain(seed uint32) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) { return m.RandomizeTerrain(seedSource(seed)) })
}

// RandomHeights lays a random relief over the height plane from seed, as one
// edit (DIV-4013).
func (s *mapInspectionEdits) RandomHeights(seed uint32) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) { return m.RandomizeHeights(seedSource(seed)) })
}

// CopyArea copies the records refs, which may be none, with the tiles,
// heights and objects of the cell rectangle box, as one clip whose origin is
// box's top-left cell, so a paste puts the records where they stood on the
// terrain.
func (s *mapInspectionEdits) CopyArea(refs []ui.RecordRef, box image.Rectangle) (ui.InspectionClip, string, error) {
	clip := &selectionClip{from: s.source}
	if len(refs) > 0 {
		c, _, err := s.CopyRecords(refs)
		if err != nil {
			return nil, "", err
		}
		clip = c.(*selectionClip)
	} else {
		source, err := mapedit.New(s.model.Bytes())
		if err != nil {
			return nil, "", err
		}
		clip.source = source
	}
	p, err := clip.source.CopyPatch(box.Min.X, box.Min.Y, box.Dx(), box.Dy())
	if err != nil {
		return nil, "", err
	}
	clip.patch, clip.origin = &p, box.Min
	return clip, fmt.Sprintf(s.inspector.word("editor.selection.area"), box.Dx(), box.Dy(), len(clip.records)), nil
}

// MoveArea moves the records refs and the objects of the cell rectangle box
// by offset cells as one edit, through the area copy's patch and its paste:
// a patch of the cells box and its moved place span holds their tiles and
// heights as they stand, no object on box and box's objects on the moved
// cells, so its paste writes the object plane alone. The records then move
// as MoveRecords moves them, on the moved objects. The moved box must lie on
// the map.
func (s *mapInspectionEdits) MoveArea(refs []ui.RecordRef, box image.Rectangle, offset image.Point) (ui.SelectionEdit, error) {
	if offset == (image.Point{}) {
		return ui.SelectionEdit{}, nil
	}
	w, h := s.model.GridSize()
	span := box.Union(box.Add(offset))
	if box.Empty() || !span.In(image.Rect(0, 0, w, h)) {
		return ui.SelectionEdit{}, fmt.Errorf("the cells %v moved by %v leave the %dx%d map", box, offset, w, h)
	}
	from, err := s.model.CopyPatch(box.Min.X, box.Min.Y, box.Dx(), box.Dy())
	if err != nil {
		return ui.SelectionEdit{}, err
	}
	p, err := s.model.CopyPatch(span.Min.X, span.Min.Y, span.Dx(), span.Dy())
	if err != nil {
		return ui.SelectionEdit{}, err
	}
	moved := 0
	if p.Objects != nil {
		at := func(c image.Point) int { return (c.Y-span.Min.Y)*span.Dx() + c.X - span.Min.X }
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				p.Objects[at(image.Pt(x, y))] = 0
			}
		}
		for i, b := range from.Objects {
			p.Objects[at(image.Pt(box.Min.X+i%box.Dx(), box.Min.Y+i/box.Dx()).Add(offset))] = b
			if b != 0 {
				moved++
			}
		}
	}
	items, err := s.moveItems(refs, offset)
	if err != nil {
		return ui.SelectionEdit{}, err
	}
	objects := func(m *mapedit.Editor) (bool, error) {
		return m.PastePatch(p, mapedit.Cell{X: span.Min.X, Y: span.Min.Y})
	}
	return s.groupEditOn(objects, items, s.inspector.word("editor.selection.moved"), func(*mapedit.Editor) []string {
		if moved == 0 {
			return nil
		}
		return []string{fmt.Sprintf(s.inspector.word("editor.selection.objects"), moved)}
	}, refs)
}
