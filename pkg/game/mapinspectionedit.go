package game

import (
	"bytes"
	"errors"
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
	"againrom/pkg/words"
)

// One retained byte-model and checkpoint own the document. A candidate clone
// retains history and is adopted only after its full inspection view succeeds.
type mapInspectionEdits struct {
	unitViews                    *inspectionUnitViews
	inspector                    *MapInspector
	model                        *mapedit.Editor
	source                       string
	saved                        []byte
	workspace                    *editorWorkspaceState
	savedWorkspace               []byte
	workspaceImage               []byte
	workspaceHistory             bool
	page                         string
	undoWorkspace, redoWorkspace []editorWorkspaceSnapshot

	errorsAllowed bool                   // the user confirmed the next save with the report's errors
	saveNote      string                 // what the last save did not write
	reported      []ui.InspectionFinding // the report of the bytes in reportedFor
	reportedFor   []byte
	reportedTexts string // the texts folder stamp the report was made under

	facts    *ui.InspectionMapFacts // the Map panel's facts of the bytes in factsFor
	factsFor []byte

	// batch is the model every edit call writes while AddRecord runs the
	// calls of one new record; nil otherwise.
	batch         *mapedit.Editor
	scriptTargets map[uint32][]ui.InspectionScriptTarget
}

func (s *mapInspectionEdits) Dirty() bool {
	return !bytes.Equal(s.saved, s.model.Bytes()) || !bytes.Equal(s.savedWorkspace, s.workspaceImage)
}
func (s *mapInspectionEdits) CanUndo() bool {
	if s.workspaceHistory {
		return len(s.undoWorkspace) > 0
	}
	return s.model.CanUndo()
}
func (s *mapInspectionEdits) CanRedo() bool {
	if s.workspaceHistory {
		return len(s.redoWorkspace) > 0
	}
	return s.model.CanRedo()
}

func (s *mapInspectionEdits) project(m *mapedit.Editor) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	d, err := s.inspector.inspectModel(m, s.source, s.workspace.Pages)
	if err == nil {
		d.Edits = s
		if d.Script != nil {
			s.scriptTargets = d.Script.Targets
		}
	}
	return d, err
}

func (s *mapInspectionEdits) change(apply func(*mapedit.Editor) (bool, error)) (*ui.InspectionDocument, error) {
	if s.batch != nil {
		_, err := apply(s.batch)
		return nil, err
	}
	next := s.model.Clone()
	changed, err := apply(next)
	if err != nil || !changed {
		return nil, err
	}
	d, err := s.project(next)
	if err == nil {
		s.rememberWorkspace(true)
		s.model = next
	}
	return d, err
}

// editKinds maps an inspection record kind to the model's record section.
// A ground sack and a unit's stock are both type-8 records.
var editKinds = map[string]mapedit.Kind{
	"Unit": mapedit.KindUnit, "Structure": mapedit.KindStructure,
	"Sack": mapedit.KindLoot, "Stock": mapedit.KindLoot,
	"Action": mapedit.KindAction, "Condition": mapedit.KindCondition,
	"Trigger":     mapedit.KindTrigger,
	"Player":      mapedit.KindPlayer,
	"Enchantment": mapedit.KindEnchantment,
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
		return true, s.checkPlacement(k, m, index, image.Rectangle{})
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
		return true, s.checkPlacement(k, m, i, image.Rectangle{})
	})
}

func (s *mapInspectionEdits) Delete(kind string, index int) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := scriptDeleteAllowed(m, k, index); err != nil {
			return false, err
		}
		err := deleteObject(m, k, index)
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
	if err != nil || k == mapedit.KindEnchantment && spellRecordBound(s.model, index) != nil {
		return nil
	}
	var out []ui.InspectionField
	var script alm.Script
	var scriptView *alm.Map
	if scriptKind(k) {
		if view, e := s.model.Map(); e == nil {
			scriptView = view
			script, _ = view.Script()
		}
	}
	template := s.unitTemplate(k, index)
	for _, f := range fields {
		row := ui.InspectionField{Key: f.Key, Label: f.Label, Text: f.Text != nil, Toward: f.Toward}
		if strings.HasPrefix(f.Label, "editor.cast.") {
			row.Label = s.inspector.inspectionUTF8(s.inspector.word(f.Label))
		}
		if f.Check == mapedit.CheckUnitTier {
			if !s.unitMultiTier(index) {
				continue
			}
			row.Label = s.inspector.word("editor.unit.tier")
		}
		if k == mapedit.KindUnit {
			label, offered := template.label(f)
			if !offered {
				continue
			}
			if label != "" {
				row.Label = label
			}
		}
		row.Choices = s.RecordChoices(kind, index, f.Key)
		if f.Text != nil {
			row.TextValue, err = s.model.TextFieldValue(k, index, f.Key)
		} else {
			row.Value, err = s.model.FieldValue(k, index, f.Key)
		}
		if err != nil {
			return nil
		}
		if k == mapedit.KindUnit {
			template.describe(&row)
		}
		if scriptView != nil {
			row.Choices = scriptFieldChoices(kind, f.Key, script, s.inspector.inspectionUTF8)
			if strings.HasPrefix(f.Key, "value") {
				row.Choices = s.scriptParameterChoices(script, k, index, f.Key)
			}
		}
		out = append(out, row)
	}
	if k == mapedit.KindTrigger {
		out = append(out, ui.InspectionField{Key: "order", Label: "Trigger order (0 first)", Value: int64(index)})
	}
	return out
}

// SetField writes one field of the record as one edit, after the field's
// range, its reference check and the record's placement check pass on the
// edited map.
func (s *mapInspectionEdits) SetField(kind string, index int, key string, value int64) (*ui.InspectionDocument, error) {
	if value == ui.NewMessageChoice && kind == "Action" && key == "value0" {
		if view, err := s.model.Map(); err == nil {
			if sc, err := view.Script(); err == nil && index >= 0 && index < len(sc.Actions) && scriptParameterKind(false, sc.Actions[index].Opcode, sc.Actions[index].Type, 0) == ui.ScriptMessageParameter {
				return s.NewMessage(ui.InspectionMessageOwner{Kind: kind, Index: index, Key: key})
			}
		}
	}
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		return s.setFieldOn(m, kind, k, index, key, value)
	})
}

// SetFields writes several fields of one record as one edit: each through
// the checks SetField or SetTextField applies, the trigger order last. A
// refused field refuses the set and the model is unchanged.
func (s *mapInspectionEdits) SetFields(kind string, index int, values []ui.InspectionFieldValue) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	last := func(v ui.InspectionFieldValue) bool {
		return v.Key == "order" || k == mapedit.KindEnchantment && v.Key == "spell" && v.Value == 0
	}
	ordered := make([]ui.InspectionFieldValue, 0, len(values))
	for _, v := range values {
		if !last(v) {
			ordered = append(ordered, v)
		}
	}
	for _, v := range values {
		if last(v) {
			ordered = append(ordered, v)
		}
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		changed, mark := false, m.Mark()
		for _, v := range ordered {
			var c bool
			var err error
			if v.Text {
				c, err = setTextFieldOn(m, k, index, v.Key, v.TextValue)
			} else if v.New {
				c, err = s.setNewNodeOn(m, k, index, v.Key)
			} else {
				c, err = s.setField(m, kind, k, index, v.Key, v.Value, false)
			}
			if err != nil {
				return false, err
			}
			changed = changed || c
		}
		m.Squash(mark)
		if changed {
			// A node's variable register is checked on the node the whole set
			// leaves, not between its fields.
			if err := scriptVariableAllowed(m, k, index); err != nil {
				return false, err
			}
		}
		return changed, nil
	})
}

// setNewNodeOn places a new node of the kind trigger slot key names, as
// AddScript does, and points the slot at it.
func (s *mapInspectionEdits) setNewNodeOn(m *mapedit.Editor, k mapedit.Kind, index int, key string) (bool, error) {
	node := mapedit.KindCondition
	if strings.HasPrefix(key, "action") {
		node = mapedit.KindAction
	}
	if k != mapedit.KindTrigger || !(node == mapedit.KindAction || strings.HasPrefix(key, "left") || strings.HasPrefix(key, "right")) {
		return false, fmt.Errorf("no new node for the field %q", key)
	}
	id, err := addScriptRecord(m, node)
	if err != nil {
		return false, err
	}
	return s.setFieldOn(m, "Trigger", k, index, key, int64(id))
}

// setFieldOn sets one numeric field on the model m after its checks.
func (s *mapInspectionEdits) setFieldOn(m *mapedit.Editor, kind string, k mapedit.Kind, index int, key string, value int64) (bool, error) {
	return s.setField(m, kind, k, index, key, value, true)
}

// setField is setFieldOn with the node's variable check when variable is set.
func (s *mapInspectionEdits) setField(m *mapedit.Editor, kind string, k mapedit.Kind, index int, key string, value int64, variable bool) (bool, error) {
	if k == mapedit.KindTrigger && key == "order" {
		if value < 0 || value >= int64(m.Count(k)) {
			return false, fmt.Errorf("no trigger position %d", value)
		}
		if value == int64(index) {
			return false, nil
		}
		err := m.Reorder(k, index, int(value))
		return err == nil, err
	}
	if err := scriptFieldAllowed(m, k, index, key, value); err != nil {
		return false, err
	}
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
	if k == mapedit.KindEnchantment && strings.HasPrefix(f.Label, "editor.cast.") {
		if key == "spell" && value == 0 {
			err := m.Delete(k, index)
			return err == nil, err
		}
		if err := spellRecordBound(m, index); err != nil {
			return false, err
		}
	}
	if old, err := m.FieldValue(k, index, key); err != nil || old == value {
		return false, err
	}
	if k == mapedit.KindEnchantment && strings.HasPrefix(f.Label, "editor.cast.") && (value < f.Min || value > f.Max) {
		return false, fmt.Errorf("%s: %d is outside %d..%d", s.inspector.inspectionUTF8(s.inspector.word(f.Label)), value, f.Min, f.Max)
	}
	was := s.footprintOf(m, k, index)
	if err := m.SetField(k, index, key, value); err != nil {
		return false, err
	}
	if k == mapedit.KindEnchantment && strings.HasPrefix(f.Label, "editor.cast.") {
		if err := spellCellAllowed(m, index); err != nil {
			return false, err
		}
	}
	if variable {
		if err := scriptVariableAllowed(m, k, index); err != nil {
			return false, err
		}
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
	case mapedit.CheckRosterSlot:
		err = rosterSlotError(view, uint32(value))
	case mapedit.CheckClassKey, mapedit.CheckDefinitionID:
		err = checkUnitField(f, view, view.AuthoredUnits[index], s.inspector.table)
	case mapedit.CheckUnitTier:
		err = s.inspector.tierError(view.AuthoredUnits[index])
	case mapedit.CheckUnitGear:
		err = s.inspector.gearError(view.AuthoredUnits[index])
	case mapedit.CheckItemRecipe:
		if value != 0 {
			if value > int64(len(view.Enchantments)) {
				err = fmt.Errorf("recipe %d is outside 1..%d", value, len(view.Enchantments))
			} else if r := view.Enchantments[value-1]; r.X != 0 || r.Y != 0 {
				err = fmt.Errorf("recipe %d has cell coordinates and cannot enchant an item", value)
			}
		}
	case mapedit.CheckScriptUnit, mapedit.CheckScriptStructure, mapedit.CheckScriptGroup:
		err = checkScriptReference(f.Check, view, value)
	case mapedit.CheckScriptPlayer:
		err = rosterSlotError(view, uint32(value))
	}
	if err != nil {
		return false, err
	}
	if err := scriptCoordinateAllowed(view, k, index, key, value); err != nil {
		return false, err
	}
	return true, s.checkPlacement(k, m, index, was)
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

// recordFeet are the cells each structure and each unit of a map takes in
// the world the game builds from it: a structure's footprint from the
// installed tables, a unit's square of its class's TileSize at its cell.
type recordFeet struct{ structures, units []image.Rectangle }

func (x *MapInspector) feet(m *alm.Map) recordFeet {
	var f recordFeet
	for _, st := range mapload.Structures(m, x.table) {
		f.structures = append(f.structures, structureFootprint(st))
	}
	for _, u := range m.Units {
		n := 1
		if x.units != nil {
			if c := x.units.Classes[int32(u.ClassID)]; c != nil {
				n = max(1, c.TileSize)
			}
		}
		cell := unitCell(u)
		f.units = append(f.units, image.Rectangle{Min: cell, Max: cell.Add(image.Pt(n, n))})
	}
	return f
}

// of is the footprints of the records of k; none for a kind that takes no
// footprint.
func (f recordFeet) of(k mapedit.Kind) []image.Rectangle {
	switch k {
	case mapedit.KindStructure:
		return f.structures
	case mapedit.KindUnit:
		return f.units
	}
	return nil
}

// footprintOf is the footprint of k's record index on m; empty for a record
// that takes none.
func (s *mapInspectionEdits) footprintOf(m *mapedit.Editor, k mapedit.Kind, index int) image.Rectangle {
	if k != mapedit.KindStructure && k != mapedit.KindUnit {
		return image.Rectangle{}
	}
	view, err := m.Map()
	if err != nil {
		return image.Rectangle{}
	}
	if all := s.inspector.feet(view).of(k); index >= 0 && index < len(all) {
		return all[index]
	}
	return image.Rectangle{}
}

// placementRefusal is a footprint a placement refuses, worded for the
// editor.
type placementRefusal string

func (r placementRefusal) Error() string { return string(r) }

// placedRecord names one record of a kind.
type placedRecord struct {
	kind  mapedit.Kind
	index int
}

// checkPlacement refuses a structure whose footprint, as the world builds it
// from the installed tables, leaves the map, and a structure or unit whose
// footprint shares a cell with another structure's or unit's. was is the
// record's footprint before the edit: a footprint the edit leaves as it was
// is not judged against the others, so a map that already holds an overlap
// keeps it.
func (s *mapInspectionEdits) checkPlacement(k mapedit.Kind, m *mapedit.Editor, index int, was image.Rectangle) error {
	if k != mapedit.KindStructure && k != mapedit.KindUnit {
		return nil
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	return s.placementError(view, s.inspector.feet(view), k, index, was, nil)
}

// placementError is checkPlacement's judgement of k's record index on view,
// whose footprints are feet. A record group holds is not judged against, so
// a group placed together is judged on the settled edit.
func (s *mapInspectionEdits) placementError(view *alm.Map, feet recordFeet, k mapedit.Kind, index int, was image.Rectangle, group map[placedRecord]bool) error {
	all := feet.of(k)
	if index < 0 || index >= len(all) {
		return nil
	}
	r := all[index]
	if k == mapedit.KindStructure && !r.In(image.Rect(0, 0, view.Width, view.Height)) {
		return placementRefusal(fmt.Sprintf(s.inspector.word("editor.placement.outside"), r.Min.X, r.Min.Y, r.Dx(), r.Dy()))
	}
	if r == was {
		return nil
	}
	name := "Unit"
	if k == mapedit.KindStructure {
		name = "Structure"
	}
	for j, o := range all {
		if j != index && !group[placedRecord{k, j}] && r.Overlaps(o) {
			return placementRefusal(fmt.Sprintf(s.inspector.word("editor.placement.overlap"), r.Min.X, r.Min.Y, r.Dx(), r.Dy(), s.recordName(ui.RecordRef{Kind: name, Index: j})))
		}
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
	if _, ok := mapload.ShippedItemName(t, data.ItemCode(code)); !ok {
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
	if len(s.undoWorkspace) > 0 {
		return s.travelWorkspace(true)
	}
	if s.workspaceHistory {
		return nil, nil
	}
	return s.modelHistory(true)
}

func (s *mapInspectionEdits) Redo() (*ui.InspectionDocument, error) {
	if len(s.redoWorkspace) > 0 {
		return s.travelWorkspace(false)
	}
	if s.workspaceHistory {
		return nil, nil
	}
	return s.modelHistory(false)
}

// SaveAs writes the map the model holds. The map file is the save: a part the
// editor cannot write beside it, the texts folder or its pages, is left out
// and named by SaveNote, and its edits stay in the workspace.
func (s *mapInspectionEdits) SaveAs(path string) (string, error) {
	if path != "" && !filepath.IsAbs(path) {
		if maps, _ := s.inspector.mapsFolder(); maps != "" {
			path = filepath.Join(maps, path)
		}
	}
	return s.save(path, false)
}

// OwnFile is the host file Save writes over: the map's own file when it is an
// ordinary existing host file outside the open install and both preserved
// installs.
func (s *mapInspectionEdits) OwnFile() (string, bool) {
	src := s.source
	if src == "" || isNewSource(src) || strings.HasPrefix(src, "scenario/") || strings.HasPrefix(src, "loose/") {
		return "", false
	}
	root := ""
	if s.inspector.archives != nil {
		root = s.inspector.archives.Root
	}
	abs, err := filepath.Abs(src)
	if err != nil || !strings.EqualFold(filepath.Ext(abs), ".alm") || !editorOrdinaryFilename(filepath.Base(abs)) {
		return "", false
	}
	parent, err := editorPhysicalDirectory(filepath.Dir(abs))
	_, grant := s.inspector.mapsFolder()
	if err != nil || fenceEditorWrite(root, parent, "Save", grant) != nil {
		return "", false
	}
	if fi, err := os.Lstat(filepath.Join(parent, filepath.Base(abs))); err != nil || !fi.Mode().IsRegular() {
		return "", false
	}
	return abs, true
}

// SaveInPlace writes the model over the map's own file; OwnFile says whether
// there is one.
func (s *mapInspectionEdits) SaveInPlace() (string, error) {
	path, ok := s.OwnFile()
	if !ok {
		return "", fmt.Errorf("the map has no file of its own to save over")
	}
	return s.save(path, true)
}

func (s *mapInspectionEdits) save(path string, overwrite bool) (string, error) {
	defer func() { s.errorsAllowed = false }()
	s.saveNote = ""
	if err := s.gate(true); err != nil {
		return "", err
	}
	b, err := s.encodeForSave()
	if err != nil {
		return "", err
	}
	root := ""
	if s.inspector.archives != nil {
		root = s.inspector.archives.Root
	}
	pages, err := s.pagesForSave()
	if err != nil {
		s.noteUnwritten(err)
	}
	maps, grant := s.inspector.mapsFolder()
	target, err := writeEditedMapPages(root, s.source, path, b, pages, s.inspector.pageLanguage(), overwrite, grant)
	if err != nil && len(pages) > 0 {
		s.noteUnwritten(err)
		target, err = writeEditedMap(root, s.source, path, b, overwrite, grant)
	}
	if err != nil {
		return "", s.inspector.fenceRefusal(err, maps)
	}
	s.saved, s.source = s.model.Bytes(), target
	if s.saveNote == "" {
		s.checkpointPages()
	}
	return target, nil
}

// encodeForSave is the model's map bytes. When the model cannot encode the
// map, the answer says so and that the edits are kept; nothing is changed.
func (s *mapInspectionEdits) encodeForSave() ([]byte, error) {
	b, err := s.model.SaveBytes()
	if err != nil {
		return nil, fmt.Errorf(words.For(s.inspector.pageLanguage()).Text("editor.save.noencode"), err)
	}
	return b, nil
}

// pagesForSave is the page tree the save writes beside the map. It answers
// no pages and the reason when the texts folder cannot be read.
func (s *mapInspectionEdits) pagesForSave() (map[string][]byte, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	return s.exportPageTree("", nil)
}

// noteUnwritten records that the pages were not written, and why.
func (s *mapInspectionEdits) noteUnwritten(err error) {
	if s.saveNote == "" {
		s.saveNote = fmt.Sprintf(words.For(s.inspector.pageLanguage()).Text("editor.save.pages"), err)
	}
}

// Save As creates a new host file only. It cannot overwrite an input, an
// existing file, or either preserved install, even through an alias. The
// original-format save guard is reused for its physical-directory and archive
// checks, not for any SAV serialization. No directories are created here.
func writeEditedMap(root, source, path string, data []byte, overwrite bool, grants ...*runtimeProfileAccess) (string, error) {
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
	if err := fenceEditorWrite(root, parent, "Save As", grants...); err != nil {
		return "", err
	}
	input := source
	if strings.HasPrefix(source, "scenario/") || isNewSource(source) {
		input = "" // archive address or a map built from nothing, never a writable host path
	} else if strings.HasPrefix(source, "loose/") {
		input = looseHostPath(root, source)
	}
	if input != "" && !overwrite {
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
	if overwrite {
		if err := replaceMapFile(parent, target, data); err != nil {
			return "", err
		}
		return target, nil
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

// replaceMapFile writes data to a temporary file beside target and renames it
// over target, so a crash leaves the old or the new file whole. An existing
// target must be an ordinary file, not a link or a directory.
func replaceMapFile(parent, target string, data []byte) error {
	perm := os.FileMode(0600)
	if fi, err := os.Lstat(target); err == nil {
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("Save cannot replace %s: not an ordinary file", filepath.Base(target))
		}
		perm = fi.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(parent, ".map-save-")
	if err != nil {
		return fmt.Errorf("create temporary map: %w", err)
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	if werr == nil {
		werr = f.Sync()
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmp, perm)
	}
	if werr == nil {
		werr = os.Rename(tmp, target)
	}
	if werr != nil {
		_ = os.Remove(tmp)
		return werr
	}
	return nil
}

// fenceEditorWrite refuses a map write into the resolved directory parent
// when it lies in the install at root or in a preserved pair beside it.
func fenceEditorWrite(root, parent, verb string, grants ...*runtimeProfileAccess) error {
	for _, grant := range grants {
		if allowed, err := grant.allows(parent); err == nil && allowed {
			return nil
		}
	}
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
	if err := refuseOriginalWriteTargets(parent, roots); err != nil {
		return &editorFenceError{verb: verb, folder: parent, cause: err}
	}
	return nil
}

// editorFenceError is a refusal of fenceEditorWrite: the folder refused and
// the fence's own answer.
type editorFenceError struct {
	verb, folder string
	cause        error
}

func (e *editorFenceError) Error() string { return e.verb + ": " + e.cause.Error() }
func (e *editorFenceError) Unwrap() error { return e.cause }

// fenceRefusal answers err in the editor's language when the install fence
// refused the folder, naming the folder and the maps folder to save in; any
// other error is answered as it is.
func (x *MapInspector) fenceRefusal(err error, maps string) error {
	var fence *editorFenceError
	if !errors.As(err, &fence) || !errors.Is(fence.cause, errWriteUnderInstall) {
		return err
	}
	t := words.For(x.pageLanguage())
	if maps == "" {
		return fmt.Errorf(t.Text("editor.save.fenced.plain"), fence.folder)
	}
	return fmt.Errorf(t.Text("editor.save.fenced"), fence.folder, maps)
}

// editorMapsFolder is the folder of the player profile the map editor saves
// maps in, beside the profile's saves.
const editorMapsFolder = "maps"

// mapsFolder is the editor's maps folder, made when absent, and the grant that
// admits writes beneath it when the profile sits inside the install. It
// answers none when the editor has no player profile.
func (x *MapInspector) mapsFolder() (string, *runtimeProfileAccess) {
	if x.testPlay == nil || x.testPlay.Profile == nil {
		return "", nil
	}
	profile, err := x.testPlay.Profile()
	if err != nil {
		return "", nil
	}
	root := ""
	if x.archives != nil {
		root = x.archives.Root
	}
	dir, grant, err := profile.folder(editorMapsFolder, root)
	if err != nil {
		return "", nil
	}
	return dir, grant
}

// SaveFolder is the folder the Save As window starts in: the folder of the
// map's own host file when the editor may write there, else the maps folder.
func (s *mapInspectionEdits) SaveFolder() string {
	maps, grant := s.inspector.mapsFolder()
	src := s.source
	if src != "" && !isNewSource(src) && !strings.HasPrefix(src, "scenario/") && !strings.HasPrefix(src, "loose/") {
		root := ""
		if s.inspector.archives != nil {
			root = s.inspector.archives.Root
		}
		if abs, err := filepath.Abs(src); err == nil {
			if parent, err := editorPhysicalDirectory(filepath.Dir(abs)); err == nil && fenceEditorWrite(root, parent, "Save", grant) == nil {
				return parent
			}
		}
	}
	return maps
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

// checkScriptReference refuses a script target the map does not hold: a
// Target_Unit below the hero band no placed unit carries, a Target_Structure
// no structure carries, or a Target_Group no unit holds (ALM-TRIG-046). A
// binding is a cell spell's object id: the hero band is not exempt, and a
// structure is named by its id word sign-extended, as the game keys it
// (ALM-T9BIND-334).
func checkScriptReference(check mapedit.FieldCheck, m *alm.Map, value int64, binding ...bool) error {
	bind := len(binding) > 0 && binding[0]
	matches := scriptReferenceCount(check, m, value, bind)
	switch check {
	case mapedit.CheckScriptUnit:
		if value >= scriptHeroBand && !bind || matches == 1 {
			return nil
		}
		if matches == 0 {
			return fmt.Errorf("no unit on the map has unit id %d", value)
		}
		return fmt.Errorf("unit id %d resolves to %d map units", value, matches)
	case mapedit.CheckScriptStructure:
		if matches == 1 {
			return nil
		}
		if matches == 0 {
			return fmt.Errorf("no structure on the map has structure id %d", value)
		}
		return fmt.Errorf("structure id %d resolves to %d structures", value, matches)
	}
	if matches > 0 {
		return nil
	}
	return fmt.Errorf("no unit on the map holds group %d", value)
}

// scriptReferenceCount is the number of the map's objects value names.
func scriptReferenceCount(check mapedit.FieldCheck, m *alm.Map, value int64, binding bool) int {
	matches := 0
	switch check {
	case mapedit.CheckScriptUnit:
		for _, u := range m.Units {
			if int64(u.UnitID) == value {
				matches++
			}
		}
	case mapedit.CheckScriptStructure:
		for _, o := range m.Objects {
			id := int64(o.Field12)
			if binding {
				id = int64(structureBindingID(o))
			}
			if id == value {
				matches++
			}
		}
	default:
		for _, u := range m.Units {
			if int64(u.GroupID) == value {
				matches++
			}
		}
	}
	return matches
}
