package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
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

func (s *mapInspectionEdits) MoveUnit(index int, cell image.Point) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		if !cell.In(image.Rect(0, 0, int(view.Width), int(view.Height))) {
			return false, fmt.Errorf("unit destination is outside the map")
		}
		r, err := m.UnitRecord(index)
		if err != nil {
			return false, err
		}
		x, y := binary.LittleEndian.Uint32(r), binary.LittleEndian.Uint32(r[4:])
		nx, ny := uint32(cell.X)<<8|x&255, uint32(cell.Y)<<8|y&255
		if x == nx && y == ny {
			return false, nil
		}
		return true, m.MoveUnit(index, nx, ny)
	})
}

// CloneUnit copies the unit at index onto cell, keeping the source's
// fraction within the cell, as one edit. mapedit.CloneUnit chooses the id.
func (s *mapInspectionEdits) CloneUnit(index int, cell image.Point) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		if !cell.In(image.Rect(0, 0, int(view.Width), int(view.Height))) {
			return false, fmt.Errorf("clone cell %d,%d is outside the map", cell.X, cell.Y)
		}
		r, err := m.UnitRecord(index)
		if err != nil {
			return false, err
		}
		x, y := binary.LittleEndian.Uint32(r), binary.LittleEndian.Uint32(r[4:])
		_, err = m.CloneUnit(index, uint32(cell.X)<<8|x&255, uint32(cell.Y)<<8|y&255)
		return err == nil, err
	})
}

func (s *mapInspectionEdits) DeleteUnit(index int) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		err := m.DeleteUnit(index)
		return err == nil, err
	})
}

// UnitFields lists the settable fields of the unit at index with their current
// values, in the order of mapedit's field table.
func (s *mapInspectionEdits) UnitFields(index int) []ui.InspectionField {
	var out []ui.InspectionField
	for _, f := range mapedit.UnitFields() {
		v, err := s.model.UnitFieldValue(index, f.Key)
		if err != nil {
			return nil
		}
		out = append(out, ui.InspectionField{Key: f.Key, Label: f.Label, Value: v})
	}
	return out
}

// SetUnitField writes one field of the unit at index as one edit, after the
// field's range and its reference check pass on the edited record.
func (s *mapInspectionEdits) SetUnitField(index int, key string, value int64) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		f, ok := mapedit.UnitFieldByKey(key)
		if !ok {
			return false, fmt.Errorf("no unit field %q", key)
		}
		if old, err := m.UnitFieldValue(index, key); err != nil || old == value {
			return false, err
		}
		if err := m.SetUnitField(index, key, value); err != nil {
			return false, err
		}
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		return true, checkUnitField(f, view, view.AuthoredUnits[index], s.inspector.table)
	})
}

// checkUnitField runs a field's reference check against the edited map and
// the installed tables (DIV-3117). The tables are the install's; a row a mod
// adds is not consulted.
func checkUnitField(f mapedit.UnitField, m *alm.Map, u alm.Unit, t *mapload.Table) error {
	switch f.Check {
	case mapedit.CheckRosterSlot:
		// A map without a type-5 record plays with one synthesised side (ALM-REQ-056).
		slots := len(m.Groups)
		if !m.Present(5) {
			slots = 1
		}
		if u.Owner < 1 || int(u.Owner) > slots {
			return fmt.Errorf("owner slot %d is not in the map's roster of %d", u.Owner, slots)
		}
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
	roots := []string{root}
	if root != "" {
		resolvedRoot, err := editorPhysicalDirectory(root)
		if err != nil {
			return "", fmt.Errorf("resolve read-only install: %w", err)
		}
		// A preserved pair is fenced explicitly, independently of the selected
		// language and of whether its archives are presently readable.
		roots = append(roots, preservedPairRoots(resolvedRoot)...)
	}
	for _, protected := range roots {
		if err := refuseOriginalWriteTarget(parent, protected); err != nil {
			return "", fmt.Errorf("Save As: %w", err)
		}
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
