package game

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// A new map is opened through the same route as an installed one: the source
// "new/<W>x<H>" is read by MapInspector.Open, the catalogue lists one row per
// size of newMapSides, and the editor window, the command line and the tests
// reach the map the same way.
const newMapPrefix = "new/"

// newMapSides is the short list of sides the window offers, square maps. The
// constructor accepts every multiple of 8 from 32 to 256 (DIV-3208).
var newMapSides = []int{32, 48, 64, 80, 128, 256}

func newMapSource(w, h int) string { return fmt.Sprintf("%s%dx%d", newMapPrefix, w, h) }

// NewMapSource is the source of the new map whose size is written "<W>x<H>".
func NewMapSource(size string) string { return newMapPrefix + size }

// newMapRows are the catalogue rows of the new-map sizes.
func newMapRows() []ui.EditorMap {
	rows := make([]ui.EditorMap, len(newMapSides))
	for i, side := range newMapSides {
		rows[i] = ui.EditorMap{Key: newMapSource(side, side), Label: fmt.Sprintf("New map %d x %d", side, side)}
	}
	return rows
}

// newMapSpec reads a "new/<W>x<H>" source into the default spec of that size.
func newMapSpec(source string) (alm.NewMapSpec, error) {
	var w, h int
	var rest string
	if n, _ := fmt.Sscanf(source, newMapPrefix+"%dx%d%s", &w, &h, &rest); n != 2 {
		return alm.NewMapSpec{}, fmt.Errorf("new map %q: want %s<width>x<height>", source, newMapPrefix)
	}
	spec := alm.DefaultNewMap()
	spec.Width, spec.Height = w, h
	return spec, spec.Validate()
}

// openNew builds the new map of source and inspects it as any opened map.
func (x *MapInspector) openNew(source string) (*ui.InspectionDocument, error) {
	spec, err := newMapSpec(source)
	if err != nil {
		return nil, err
	}
	model, err := mapedit.NewMap(spec)
	if err != nil {
		return nil, err
	}
	s := &mapInspectionEdits{inspector: x, model: model, source: source, saved: model.Bytes()}
	return s.project(model)
}

// WriteNewMap writes the blank map of source to the new file path, through the
// fenced writer Save As uses, and returns the file written and the findings
// the map reports. The blank map is the starting file of an author, so the
// findings it holds do not refuse the write (DIV-3211).
func (x *MapInspector) WriteNewMap(source, path string) (string, []ui.InspectionFinding, error) {
	d, err := x.openNew(source)
	if err != nil {
		return "", nil, err
	}
	s := d.Edits.(*mapInspectionEdits)
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	b, err := s.model.SaveBytes()
	if err != nil {
		return "", nil, err
	}
	target, err := writeEditedMap(root, source, path, b)
	if err != nil {
		return "", nil, err
	}
	return target, s.Report(), nil
}

// newUnitFraction is the position within its cell a new unit's words carry,
// the cell's centre as a new sack's.
const newUnitFraction = 128

// AddUnit places a new unit of the class key on cell, owned by the map's
// second roster slot when it has one and the first otherwise, as one edit
// (DIV-3209). The class is checked against the installed tables as a field
// edit's is.
func (s *mapInspectionEdits) AddUnit(cell image.Point, class int64) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := insideMap(m, cell, "unit cell"); err != nil {
			return false, err
		}
		if class < math.MinInt16 || class > math.MaxInt16 {
			return false, fmt.Errorf("class key %d is outside %d..%d", class, math.MinInt16, math.MaxInt16)
		}
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		owner := uint32(min(2, max(1, len(view.Groups))))
		secondary, err := s.inspector.secondaryKey(int16(class))
		if err != nil {
			return false, err
		}
		x, y := cellWords(cell, newUnitFraction, newUnitFraction)
		rec, err := m.NewUnitRecord(x, y, int16(class), secondary, owner)
		if err != nil {
			return false, err
		}
		_, err = m.PlaceUnit(rec)
		return err == nil, err
	})
}

// secondaryKey is the secondary key of the first installed definition row that
// has the class key, so a new unit resolves to a definition (DIV-3209). A key
// below the creature floor is a human, which the class key alone selects.
func (x *MapInspector) secondaryKey(class int16) (uint16, error) {
	if x.table == nil {
		return 0, fmt.Errorf("no installed tables to check class %d against", class)
	}
	best, face := data.NotFound, uint16(0)
	for k := 0; k <= math.MaxUint8; k++ {
		r := mapload.Resolve(alm.Unit{ClassID: class, ClassSubID: uint16(k)}, x.table)
		if r.Found() && (best == data.NotFound || r.Index < best) {
			best, face = r.Index, uint16(k)
		}
	}
	if best == data.NotFound {
		return 0, fmt.Errorf("class %d names no installed definition", class)
	}
	return face, nil
}

// isNewSource reports a source that names a map built from nothing and not
// yet saved.
func isNewSource(source string) bool { return strings.HasPrefix(source, newMapPrefix) }

// unwrittenEntry is the mission of the mod whose map file the mod folder does
// not hold yet: the entry a new map is saved under. It answers only when
// exactly one entry is unwritten, so a new map never chooses between two
// (DIV-3210).
func (m *editorMod) unwrittenEntry() (mod.Mission, bool) {
	var found []mod.Mission
	for _, e := range m.missions.Missions {
		if _, err := os.Lstat(filepath.Join(m.dir, filepath.FromSlash(e.Map))); errors.Is(err, fs.ErrNotExist) {
			found = append(found, e)
		}
	}
	if len(found) != 1 {
		return mod.Mission{}, false
	}
	return found[0], true
}
