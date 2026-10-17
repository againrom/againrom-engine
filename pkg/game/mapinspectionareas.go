package game

import (
	"fmt"
	"image"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// A trigger area drawn on the map becomes a box check (operation 2): the
// unit, then the box's corners as X, Y, X2, Y2, inclusive
// (TRIG-CHECKGEOM-130). It joins its trigger as one more pair, the box equal
// to a constant 1, as every shipped box pair but one is written.

// boxAreaUnit is the box check's unit: hero ordinal 0, the first player's
// primary character (TRIG-HEROORD-075).
const boxAreaUnit = 10001

// AddBoxArea adds a box check over box's cells, Max exclusive, to trigger
// as one more pair, or to a new trigger when trigger is -1, as one edit.
func (s *mapInspectionEdits) AddBoxArea(box image.Rectangle, trigger int) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		mark := m.Mark()
		if err := addBoxArea(m, box, trigger); err != nil {
			return false, err
		}
		m.Squash(mark)
		return true, nil
	})
}

func addBoxArea(m *mapedit.Editor, box image.Rectangle, trigger int) error {
	view, err := m.Map()
	if err != nil {
		return err
	}
	sc, err := view.Script()
	if err != nil {
		return err
	}
	if box.Empty() || !box.In(image.Rect(0, 0, min(view.Width, 256), min(view.Height, 256))) {
		return fmt.Errorf("the box %v lies outside the map", box)
	}
	pair := 0
	if trigger >= 0 {
		if trigger >= len(sc.Triggers) {
			return fmt.Errorf("no trigger %d", trigger+1)
		}
		pair = -1
		t := sc.Triggers[trigger]
		for k := range t.Left {
			if t.Left[k] == 0 && t.Right[k] == 0 {
				pair = k
				break
			}
		}
		if pair < 0 {
			return fmt.Errorf("trigger %d already has three condition pairs", trigger+1)
		}
	}
	one, found := unsharedConstant(sc, 1)
	if need := 1 + refCount(!found); len(sc.Conditions)+need > 100 {
		return fmt.Errorf("the script already uses 100 check registers")
	}
	if err := registerCollides(sc, uint32(len(sc.Conditions))); err != nil {
		return err
	}
	id, err := m.NextScriptID(mapedit.KindCondition)
	if err != nil {
		return err
	}
	if id > 5000 || !found && id+1 > 5000 {
		return fmt.Errorf("the next node ID exceeds 5000")
	}
	rec, err := mapedit.NewScriptNode(mapload.ScriptOperationName(true, 2), 2, id,
		mapedit.ScriptParam{Type: mapload.ScriptUnit, Value: boxAreaUnit},
		mapedit.ScriptParam{Type: mapload.ScriptX, Value: uint32(box.Min.X)}, mapedit.ScriptParam{Type: mapload.ScriptY, Value: uint32(box.Min.Y)},
		mapedit.ScriptParam{Type: mapload.ScriptX, Value: uint32(box.Max.X - 1)}, mapedit.ScriptParam{Type: mapload.ScriptY, Value: uint32(box.Max.Y - 1)})
	if err != nil {
		return err
	}
	if _, err := m.Place(mapedit.KindCondition, rec); err != nil {
		return err
	}
	if !found {
		one = id + 1
		rec, err := mapedit.NewScriptNode(mapload.ScriptOperationName(true, mapload.ScriptConstant), mapload.ScriptConstant, one, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 1})
		if err != nil {
			return err
		}
		if _, err := m.Place(mapedit.KindCondition, rec); err != nil {
			return err
		}
	}
	if trigger < 0 {
		if _, err := addScriptRecord(m, mapedit.KindTrigger); err != nil {
			return err
		}
		trigger = m.Count(mapedit.KindTrigger) - 1
	}
	return m.SetFields(mapedit.KindTrigger, trigger, mapedit.FieldSet{Key: fmt.Sprintf("left%d", pair), Value: int64(id)},
		mapedit.FieldSet{Key: fmt.Sprintf("right%d", pair), Value: int64(one)}, mapedit.FieldSet{Key: fmt.Sprintf("cmp%d", pair), Value: 0})
}

// unsharedConstant is the id of a constant condition holding value that no
// variable reads or writes: its id names one node, and no set-variable
// action or variable check addresses its register (TRIG-COND-003).
func unsharedConstant(sc alm.Script, value uint32) (uint32, bool) {
	ids := map[uint32]int{}
	for _, n := range sc.Conditions {
		ids[n.ID]++
	}
	variable := map[uint32]bool{}
	for _, n := range sc.Actions {
		if n.Opcode == 3 || n.Opcode == 8 {
			variable[scriptVariable(n)] = true
		}
	}
	for _, n := range sc.Conditions {
		if n.Opcode == 19 {
			variable[scriptVariable(n)] = true
		}
	}
	for i, n := range sc.Conditions {
		if n.Opcode == mapload.ScriptConstant && n.ID != 0 && ids[n.ID] == 1 && !variable[uint32(i)] &&
			(n.Type[0] == mapload.ScriptInt || n.Type[0] == mapload.ScriptConst) && n.Value[0] == value {
			return n.ID, true
		}
	}
	return 0, false
}

// refCount is 1 for true, 0 for false.
func refCount(b bool) int {
	if b {
		return 1
	}
	return 0
}
