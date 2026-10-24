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
