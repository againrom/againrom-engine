package game

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/locale"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// One retained byte-model and checkpoint own the document. A candidate clone
// retains history and is adopted only after its full inspection view succeeds.
type mapInspectionEdits struct {
	inspector *MapInspector
	model     *mapedit.Editor
	source    string
	saved     []byte

	reported    []ui.InspectionFinding // the report of the bytes in reportedFor
	reportedFor []byte
}

func (s *mapInspectionEdits) Dirty() bool   { return !bytes.Equal(s.saved, s.model.Bytes()) }
func (s *mapInspectionEdits) CanUndo() bool { return s.model.CanUndo() }
func (s *mapInspectionEdits) CanRedo() bool { return s.model.CanRedo() }

func (s *mapInspectionEdits) project(m *mapedit.Editor) (*ui.InspectionDocument, error) {
	d, err := s.inspector.inspectModel(m, s.source)
	if err == nil {
		d.Edits = s
	}
	return d, err
}

func (s *mapInspectionEdits) change(apply func(*mapedit.Editor) (bool, error)) (*ui.InspectionDocument, error) {
	next := s.model.Clone()
	changed, err := apply(next)
	if err != nil || !changed {
		return nil, err
	}
	d, err := s.project(next)
	if err == nil {
		s.model = next
	}
	return d, err
}

// editKinds maps an inspection record kind to the model's record section.
// A ground sack and a unit's stock are both type-8 records.
var editKinds = map[string]mapedit.Kind{
	"Unit": mapedit.KindUnit, "Structure": mapedit.KindStructure,
	"Sack": mapedit.KindLoot, "Stock": mapedit.KindLoot,
}

func editKind(kind string) (mapedit.Kind, error) {
	k, ok := editKinds[kind]
	if !ok {
		return 0, fmt.Errorf("a %s record cannot be edited", strings.ToLower(kind))
	}
	return k, nil
}

// cellWords keeps the source's fraction within the cell, as a move does.
func cellWords(cell image.Point, x, y uint32) (uint32, uint32) {
	return uint32(cell.X)<<8 | x&255, uint32(cell.Y)<<8 | y&255
}

// Move moves the record onto cell, keeping its fraction within the cell, as
// one edit.
func (s *mapInspectionEdits) Move(kind string, index int, cell image.Point) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := insideMap(m, cell, "destination"); err != nil {
			return false, err
		}
		x, y, err := m.Position(k, index)
		if err != nil {
			return false, err
		}
		nx, ny := cellWords(cell, x, y)
		if x == nx && y == ny {
			return false, nil
		}
		if err := m.Move(k, index, nx, ny); err != nil {
			return false, err
		}
		return true, s.checkPlacement(k, m, index)
	})
}

// Clone copies the record onto cell, keeping the source's fraction within
// the cell, as one edit. The model's section row chooses the copy's identity.
func (s *mapInspectionEdits) Clone(kind string, index int, cell image.Point) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := insideMap(m, cell, "clone cell"); err != nil {
			return false, err
		}
		x, y, err := m.Position(k, index)
		if err != nil {
			return false, err
		}
		nx, ny := cellWords(cell, x, y)
		i, err := m.CloneRecord(k, index, nx, ny)
		if err != nil {
			return false, err
		}
		return true, s.checkPlacement(k, m, i)
	})
}

func (s *mapInspectionEdits) Delete(kind string, index int) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		err := m.Delete(k, index)
		return err == nil, err
	})
}

// AddSack places a new ground sack holding one item on cell, as one edit.
// Its words sit at the cell's centre and it carries no gold (DIV-3134).
func (s *mapInspectionEdits) AddSack(cell image.Point, item int64) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := insideMap(m, cell, "sack cell"); err != nil {
			return false, err
		}
		if item <= 0 || item > math.MaxUint16 {
			return false, fmt.Errorf("item code %d is outside 1..65535", item)
		}
		if err := s.inspector.checkItem(item); err != nil {
			return false, err
		}
		x, y := cellWords(cell, newSackFraction, newSackFraction)
		_, err := m.Place(mapedit.KindLoot, m.GroundSack(x, y, uint16(item)))
		return err == nil, err
	})
}

// newSackFraction is the position within its cell a new sack's words carry.
const newSackFraction = 128

// Fields lists the settable fields of the record with their current values,
// in the order of its section's field table.
func (s *mapInspectionEdits) Fields(kind string, index int) []ui.InspectionField {
	k, err := editKind(kind)
	if err != nil {
		return nil
	}
	fields, err := s.model.Fields(k, index)
	if err != nil {
		return nil
	}
	var out []ui.InspectionField
	for _, f := range fields {
		v, err := s.model.FieldValue(k, index, f.Key)
		if err != nil {
			return nil
		}
		out = append(out, ui.InspectionField{Key: f.Key, Label: f.Label, Value: v})
	}
	return out
}

// SetField writes one field of the record as one edit, after the field's
// range, its reference check and the record's placement check pass on the
// edited map.
func (s *mapInspectionEdits) SetField(kind string, index int, key string, value int64) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		fields, err := m.Fields(k, index)
		if err != nil {
			return false, err
		}
		var f mapedit.Field
		for _, row := range fields {
			if row.Key == key {
				f = row
			}
		}
		if f.Key == "" {
			return false, fmt.Errorf("no %s field %q", strings.ToLower(kind), key)
		}
		if old, err := m.FieldValue(k, index, key); err != nil || old == value {
			return false, err
		}
		if err := m.SetField(k, index, key, value); err != nil {
			return false, err
		}
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		switch f.Check {
		case mapedit.CheckStructureKind:
			err = s.inspector.checkStructureKind(value)
		case mapedit.CheckItemCode:
			if value != 0 {
				err = s.inspector.checkItem(value)
			}
		case mapedit.CheckRosterSlot, mapedit.CheckClassKey, mapedit.CheckDefinitionID:
			err = checkUnitField(f, view, view.AuthoredUnits[index], s.inspector.table)
		}
		if err != nil {
			return false, err
		}
		return true, s.checkPlacement(k, m, index)
	})
}

func insideMap(m *mapedit.Editor, cell image.Point, what string) error {
	view, err := m.Map()
	if err != nil {
		return err
	}
	if !cell.In(image.Rect(0, 0, int(view.Width), int(view.Height))) {
		return fmt.Errorf("%s %d,%d is outside the map", what, cell.X, cell.Y)
	}
	return nil
}

// checkPlacement refuses a structure whose footprint, as the world builds it
// from the installed tables, leaves the map.
func (s *mapInspectionEdits) checkPlacement(k mapedit.Kind, m *mapedit.Editor, index int) error {
	if k != mapedit.KindStructure {
		return nil
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	st := mapload.Structures(view, s.inspector.table)[index]
	r := structureFootprint(st)
	if !r.In(image.Rect(0, 0, view.Width, view.Height)) {
		return fmt.Errorf("structure footprint %d,%d %dx%d leaves the map", r.Min.X, r.Min.Y, r.Dx(), r.Dy())
	}
	return nil
}

// checkStructureKind refuses a kind the installed structures.reg does not
// hold. A class a mod adds is not consulted.
func (x *MapInspector) checkStructureKind(kind int64) error {
	set := x.structures
	if set == nil || kind <= 0 || kind >= int64(len(set.Classes)) || set.Classes[kind] == nil {
		return fmt.Errorf("structure kind %d is not in the installed structures.reg", kind)
	}
	return nil
}

// checkItem refuses an item code the installed item-name table does not
// name (DIV-3134). An item a mod adds is not consulted.
func (x *MapInspector) checkItem(code int64) error {
	t := x.table
	if t == nil {
		return fmt.Errorf("no installed tables to check item code %d against", code)
	}
	if _, ok := t.Names.NameFor(data.ItemCode(code)); !ok {
		return fmt.Errorf("item code %d names no installed item", code)
	}
	return nil
}

// checkUnitField runs a field's reference check against the edited map and
// the installed tables (DIV-3117). The tables are the install's; a row a mod
// adds is not consulted.
func checkUnitField(f mapedit.UnitField, m *alm.Map, u alm.Unit, t *mapload.Table) error {
	switch f.Check {
	case mapedit.CheckRosterSlot:
		return rosterSlotError(m, u.Owner)
	case mapedit.CheckClassKey:
		if t == nil {
			return fmt.Errorf("no installed tables to check class %d against", u.ClassID)
		}
		if mapload.Resolve(alm.Unit{ClassID: u.ClassID, ClassSubID: u.ClassSubID}, t).Index == data.NotFound {
			return fmt.Errorf("class %d (secondary key %d) names no installed definition", u.ClassID, u.ClassSubID)
		}
	case mapedit.CheckDefinitionID:
		if u.DefID == 0 {
			return nil
		}
		if t == nil {
			return fmt.Errorf("no installed tables to check definition id %d against", u.DefID)
		}
		if mapload.Resolve(alm.Unit{DefID: u.DefID}, t).Index == data.NotFound {
			return fmt.Errorf("definition id %d names no installed definition", u.DefID)
		}
	}
	return nil
}

func (s *mapInspectionEdits) Undo() (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) { return m.Undo(), nil })
}

func (s *mapInspectionEdits) Redo() (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) { return m.Redo(), nil })
}

func (s *mapInspectionEdits) SaveAs(path string) (string, error) {
	if err := s.gate(); err != nil {
		return "", err
	}
	b := s.model.Bytes()
	root := ""
	if s.inspector.archives != nil {
		root = s.inspector.archives.Root
	}
	target, err := writeEditedMap(root, s.source, path, b)
	if err != nil {
		return "", err
	}
	s.saved, s.source = b, target
	return target, nil
}

// Save As creates a new host file only. It cannot overwrite an input, an
// existing file, or either preserved install, even through an alias. The
// original-format save guard is reused for its physical-directory and archive
// checks, not for any SAV serialization. No directories are created here.
func writeEditedMap(root, source, path string, data []byte) (string, error) {
	if strings.TrimSpace(path) == "" || !strings.EqualFold(filepath.Ext(path), ".alm") {
		return "", fmt.Errorf("Save As needs an explicit new .alm path")
	}
	// Require a resolvable existing parent. Unlike the shared save-directory
	// helper, an ambiguous reparse point fails closed for this explicit writer.
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	base := filepath.Base(abs)
	if !editorOrdinaryFilename(base) {
		return "", fmt.Errorf("Save As needs an ordinary new filename, not a device or alternate stream")
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(abs))
	if err != nil {
		return "", fmt.Errorf("resolve map output directory: %w", err)
	}
	target := filepath.Join(parent, base)
	if err := fenceEditorWrite(root, parent, "Save As"); err != nil {
		return "", err
	}
	input := source
	if strings.HasPrefix(source, "scenario/") {
		input = "" // archive address, never a writable host path
	} else if strings.HasPrefix(source, "loose/") {
		input = filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(source, "loose/")))
	}
	if input != "" {
		resolvedInput, err := resolvePathForSaveSafety(input)
		if err != nil {
			return "", err
		}
		if sameSaveName(target, resolvedInput) {
			return "", fmt.Errorf("Save As cannot replace the source map")
		}
		if a, err := os.Stat(input); err == nil {
			if b, err := os.Stat(target); err == nil && os.SameFile(a, b) {
				return "", fmt.Errorf("Save As cannot replace a source alias")
			}
		}
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("create new map: %w", err)
	}
	_, writeErr := f.Write(data)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		// This exact file was created exclusively by this call, never an input.
		_ = os.Remove(target)
		if writeErr != nil {
			return "", writeErr
		}
		return "", closeErr
	}
	return target, nil
}

// fenceEditorWrite refuses a map write into the resolved directory parent
// when it lies in the install at root or in a preserved pair beside it.
func fenceEditorWrite(root, parent, verb string) error {
	roots := []string{root}
	if root != "" {
		resolvedRoot, err := editorPhysicalDirectory(root)
		if err != nil {
			return fmt.Errorf("resolve read-only install: %w", err)
		}
		// A preserved pair is fenced explicitly, independently of the selected
		// language and of whether its archives are presently readable.
		roots = append(roots, preservedPairRoots(resolvedRoot)...)
	}
	for _, protected := range roots {
		if err := refuseOriginalWriteTarget(parent, protected); err != nil {
			return fmt.Errorf("%s: %w", verb, err)
		}
	}
	return nil
}

func editorOrdinaryFilename(name string) bool {
	if !filepath.IsLocal(name) || strings.Contains(name, ":") {
		return false
	}
	stem, _, _ := strings.Cut(name, ".")
	stem = strings.ToUpper(strings.TrimRight(stem, " "))
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "CLOCK$", "CONIN$", "CONOUT$":
		return false
	}
	for _, prefix := range []string{"COM", "LPT"} {
		if strings.HasPrefix(stem, prefix) {
			tail := []rune(strings.TrimPrefix(stem, prefix))
			if len(tail) == 1 {
				r := tail[0]
				if r >= '0' && r <= '9' || r == 0xb9 || r == 0xb2 || r == 0xb3 {
					return false
				}
			}
		}
	}
	return true
}

// preservedPairRoots are the sibling installs, one per locale code, of an
// install kept under a gameversions directory; none for any other root.
func preservedPairRoots(resolvedRoot string) []string {
	parent := filepath.Dir(resolvedRoot)
	if !strings.EqualFold(filepath.Base(parent), "gameversions") {
		return nil
	}
	var roots []string
	for _, l := range locale.All() {
		roots = append(roots, filepath.Join(parent, l.Code))
	}
	return roots
}
