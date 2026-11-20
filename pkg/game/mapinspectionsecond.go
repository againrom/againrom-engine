package game

import (
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// secondEdits is the edit host of a second-game map. It offers the terrain
// brush, undo, redo and the save: the tools whose written fields the format
// notes read. Every other call is refused with one line, and a tool whose
// host seam is left out here is off.
type secondEdits struct{ s *mapInspectionEdits }

func (c secondEdits) refusal() error {
	return errors.New(c.s.inspector.word("editor.map.secondtool"))
}

func (c secondEdits) Dirty() bool   { return c.s.Dirty() }
func (c secondEdits) CanUndo() bool { return c.s.CanUndo() }
func (c secondEdits) CanRedo() bool { return c.s.CanRedo() }

// Move, Clone and Delete take units, structures and ground sacks; the other
// kinds are refused.
func (c secondEdits) Move(kind ui.RecordKind, index int, cell image.Point) (*ui.InspectionDocument, error) {
	if !c.addable(kind) {
		return nil, c.refusal()
	}
	return c.s.Move(kind, index, cell)
}

func (c secondEdits) Clone(kind ui.RecordKind, index int, cell image.Point) (*ui.InspectionDocument, error) {
	if !c.addable(kind) {
		return nil, c.refusal()
	}
	return c.s.Clone(kind, index, cell)
}

func (c secondEdits) Delete(kind ui.RecordKind, index int) (*ui.InspectionDocument, error) {
	if !c.addable(kind) {
		return nil, c.refusal()
	}
	return c.s.Delete(kind, index)
}

func (c secondEdits) Fields(ui.RecordKind, int) []ui.InspectionField { return nil }

func (c secondEdits) Undo() (*ui.InspectionDocument, error) { return c.s.Undo() }
func (c secondEdits) Redo() (*ui.InspectionDocument, error) { return c.s.Redo() }

func (c secondEdits) SaveAs(path string) (string, error) { return c.s.SaveAs(path) }
func (c secondEdits) SaveFolder() string                 { return c.s.SaveFolder() }
func (c secondEdits) OwnFile() (string, bool)            { return c.s.OwnFile() }

func (c secondEdits) SaveInPlace() (string, error) { return c.s.SaveInPlace() }

func (c secondEdits) BrushTerrains() []ui.BrushTerrain { return c.s.BrushTerrains() }
func (c secondEdits) BrushObjects() []ui.BrushObject   { return c.s.BrushObjects() }
func (c secondEdits) BrushCells(op ui.BrushStroke) []image.Point {
	return c.s.BrushCells(op)
}
func (c secondEdits) BrushMaxRadius() int                  { return c.s.BrushMaxRadius() }
func (c secondEdits) BrushVariants(class int) int          { return c.s.BrushVariants(class) }
func (c secondEdits) BrushObjectIcon(code int) *image.RGBA { return c.s.BrushObjectIcon(code) }

// Paint takes a terrain stroke; the height, object, passability and bridge
// planes are refused.
func (c secondEdits) Paint(op ui.BrushStroke) (*ui.InspectionDocument, error) {
	if op.Plane != ui.BrushPlaneTerrain {
		return nil, c.refusal()
	}
	return c.s.Paint(op)
}

// The palettes list the catalog; a unit, structure or sack is added, every
// other add is refused.
func (c secondEdits) NewRecordStep(r ui.InspectionNewRecord, key string) ui.InspectionNewStep {
	return viewPalettes{c.s}.NewRecordStep(r, key)
}

func (c secondEdits) CheckNewRecord(r ui.InspectionNewRecord, key string) error {
	if !c.addable(r.Kind) {
		return c.refusal()
	}
	return c.s.CheckNewRecord(r, key)
}

func (c secondEdits) NewRecordPreview(kind ui.RecordKind, key string, value int64) *image.RGBA {
	return viewPalettes{c.s}.NewRecordPreview(kind, key, value)
}

func (c secondEdits) AddRecord(r ui.InspectionNewRecord) (*ui.InspectionDocument, error) {
	if !c.addable(r.Kind) || len(r.Values) > 1 && r.Kind != ui.KindSack || len(recordValues(r, "item")) > 1 {
		return nil, c.refusal()
	}
	return c.s.AddRecord(r)
}

// secondRegisterOps are the second game's operations whose first parameter
// indexes the session's map registers: actions 3 and 8, check 19.
var secondRegisterOps = map[[2]uint32]bool{{0, 3}: true, {0, 8}: true, {1, 19}: true}

func (c secondEdits) SetField(kind ui.RecordKind, index int, key string, value int64) (*ui.InspectionDocument, error) {
	return c.SetFields(kind, index, []ui.InspectionFieldValue{{Key: key, Value: value}})
}

// SetFields writes the parameter words of a decoded node, or moves a trigger
// in the stored order. Every other script edit is refused with one line.
func (c secondEdits) SetFields(kind ui.RecordKind, index int, values []ui.InspectionFieldValue) (_ *ui.InspectionDocument, said error) {
	defer c.s.worded(&said)
	k, err := editKind(kind)
	if err != nil || len(values) == 0 {
		return nil, c.refusal()
	}
	switch k {
	case mapedit.KindAction, mapedit.KindCondition:
		return c.setParameters(k, index, values)
	case mapedit.KindTrigger:
		if len(values) != 1 || values[0].Key != "order" || values[0].Text || values[0].New {
			return nil, errors.New(c.s.inspector.word("editor.map.secondscript"))
		}
		to := values[0].Value
		return c.s.change(func(m *mapedit.Editor) (bool, error) {
			if to < 0 || to >= int64(m.Count(k)) {
				return false, wordErr("editor.err.triggerposition", to)
			}
			if to == int64(index) {
				return false, nil
			}
			err := m.Reorder(k, index, int(to))
			return err == nil, err
		})
	}
	return nil, c.refusal()
}

func (c secondEdits) setParameters(k mapedit.Kind, index int, values []ui.InspectionFieldValue) (*ui.InspectionDocument, error) {
	view, err := c.s.model.Map()
	if err != nil {
		return nil, err
	}
	sc, err := view.Script()
	if err != nil {
		return nil, err
	}
	nodes := scriptNodes(sc, k)
	if index < 0 || index >= len(nodes) {
		return nil, wordErr("editor.err.nonode", index)
	}
	n, condition := nodes[index], k == mapedit.KindCondition
	if !mapload.ScriptOperationDecoded(scriptprofile.OpsSecond, condition, n.Opcode) {
		return nil, fmt.Errorf(c.s.inspector.word("editor.map.secondundecoded"), n.Opcode)
	}
	var sets []mapedit.FieldSet
	for _, v := range values {
		slot, err := strconv.Atoi(strings.TrimPrefix(v.Key, "value"))
		if !strings.HasPrefix(v.Key, "value") || err != nil || slot < 0 || slot >= len(n.Type) || n.Type[slot] == 0 || v.Text || v.New {
			return nil, errors.New(c.s.inspector.word("editor.map.secondscript"))
		}
		family := uint32(0)
		if condition {
			family = 1
		}
		if secondRegisterOps[[2]uint32{family, n.Opcode}] && slot == 0 &&
			(v.Value < 0 || v.Value >= scriptprofile.SecondRegisters) {
			return nil, wordErr("editor.script.variableout", v.Value, scriptprofile.SecondRegisters-1)
		}
		if err := scriptCoordinateAllowed(view, k, index, v.Key, v.Value); err != nil {
			return nil, err
		}
		sets = append(sets, mapedit.FieldSet{Key: v.Key, Value: v.Value})
	}
	return c.s.change(func(m *mapedit.Editor) (bool, error) {
		var todo []mapedit.FieldSet
		for _, set := range sets {
			if old, err := m.FieldValue(k, index, set.Key); err != nil {
				return false, err
			} else if old != set.Value {
				todo = append(todo, set)
			}
		}
		if len(todo) == 0 {
			return false, nil
		}
		err := m.SetFields(k, index, todo...)
		return err == nil, err
	})
}

func (c secondEdits) AddKinds() []ui.RecordKind { return viewPalettes{c.s}.AddKinds() }
