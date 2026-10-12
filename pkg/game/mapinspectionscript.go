package game

import (
	"fmt"
	"image"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The script arms the editor offers a target for. A win arm is a trigger that
// runs instant 4 and a lose arm one that runs instant 5; a check 18 node loses
// the mission when its unit dies whether or not a trigger names it
// (MISSION-WIN-003, MISSION-VIP-004). The target of an arm is the unit,
// structure or group parameter of a condition node its trigger reads.
const (
	scriptOpWin       = 4
	scriptOpLose      = 5
	scriptCheckVIP    = 18
	scriptUnitParam   = 4
	scriptStructParam = 9
	armWin            = "Win"
	armLose           = "Lose"
	armProtect        = "Protect"
)

// goalRow is one target parameter an arm reads: the arm, the trigger that
// carries it, the condition node and parameter slot that holds it.
type goalRow struct {
	arm           string
	trigger, node int
	slot          int
	param, value  uint32
}

// scriptGoalRows lists the target parameters of the script's win, lose and
// protect arms in trigger order, then check 18 order, and ends with the row
// that adds a protect check (node -1). A trigger names its condition nodes by
// id (ALM-TRIG-047).
func scriptGoalRows(sc alm.Script) []goalRow {
	condition := map[uint32]int{}
	for i, n := range sc.Conditions {
		if _, ok := condition[n.ID]; !ok {
			condition[n.ID] = i
		}
	}
	action := map[uint32]uint32{}
	for _, n := range sc.Actions {
		if _, ok := action[n.ID]; !ok {
			action[n.ID] = n.Opcode
		}
	}
	var rows []goalRow
	type armSlot struct {
		arm        string
		node, slot int
	}
	seen := map[armSlot]bool{}
	add := func(arm string, trigger, node int) {
		n := sc.Conditions[node]
		for slot, tag := range n.Type {
			if tag != scriptUnitParam && tag != scriptGroupParam && tag != scriptStructParam {
				continue
			}
			key := armSlot{arm, node, slot}
			if seen[key] {
				continue
			}
			seen[key] = true
			rows = append(rows, goalRow{arm: arm, trigger: trigger, node: node, slot: slot, param: tag, value: n.Value[slot]})
		}
	}
	for ti, t := range sc.Triggers {
		var arms []string
		for _, id := range t.Acts {
			switch action[id] {
			case scriptOpWin:
				arms = append(arms, armWin)
			case scriptOpLose:
				arms = append(arms, armLose)
			}
		}
		for _, arm := range arms {
			for p := range t.Left {
				for _, id := range []uint32{t.Left[p], t.Right[p]} {
					if i, ok := condition[id]; ok && id != 0 {
						add(arm, ti, i)
					}
				}
			}
		}
	}
	for i, n := range sc.Conditions {
		if n.Opcode == scriptCheckVIP {
			add(armProtect, -1, i)
		}
	}
	return append(rows, goalRow{arm: armProtect, trigger: -1, node: -1, slot: -1, param: scriptUnitParam})
}

// Goals lists the arms of the map's script as the window's script panel shows
// them.
func (s *mapInspectionEdits) Goals() []ui.InspectionGoal {
	m, err := s.model.Map()
	if err != nil {
		return nil
	}
	sc, err := m.Script()
	if err != nil {
		return nil
	}
	var out []ui.InspectionGoal
	for _, g := range scriptGoalRows(sc) {
		out = append(out, goalListing(m, sc, g))
	}
	return out
}

func goalListing(m *alm.Map, sc alm.Script, g goalRow) ui.InspectionGoal {
	pick := "Unit"
	if g.param == scriptStructParam {
		pick = "Structure"
	}
	if g.node < 0 {
		return ui.InspectionGoal{Arm: g.arm, Pick: pick, Index: -1, Text: "add a check that loses the mission when a unit dies"}
	}
	n := sc.Conditions[g.node]
	who := fmt.Sprintf("trigger %q", sc.Triggers[max(g.trigger, 0)].Name)
	if g.trigger < 0 {
		who = fmt.Sprintf("check %d", n.ID)
	}
	target, cell, has := describeTarget(m, g.param, g.value)
	return ui.InspectionGoal{Arm: g.arm, Pick: pick, Kind: "Condition", Index: g.node, Cell: cell, HasCell: has,
		Text: fmt.Sprintf("%s: %s, %s", who, strings.ToLower(inspectionOperation("Condition", n.Opcode)), target)}
}

// describeTarget names what a target value points at on the map, and the cell
// it stands on.
func describeTarget(m *alm.Map, param, value uint32) (string, image.Point, bool) {
	switch param {
	case scriptUnitParam:
		if value >= scriptHeroBand {
			return fmt.Sprintf("party role %d", value), image.Point{}, false
		}
		for _, u := range m.Units {
			if uint32(u.UnitID) == value {
				c := unitCell(u)
				return fmt.Sprintf("unit %d at %d,%d", value, c.X, c.Y), c, true
			}
		}
		return fmt.Sprintf("unit %d (not on the map)", value), image.Point{}, false
	case scriptStructParam:
		for _, o := range m.Objects {
			if uint32(o.Field12) == value {
				c := image.Pt(int(o.X>>8), int(o.Y>>8))
				return fmt.Sprintf("structure %d at %d,%d", value, c.X, c.Y), c, true
			}
		}
		return fmt.Sprintf("structure %d (not on the map)", value), image.Point{}, false
	}
	n := 0
	var at image.Point
	for _, u := range m.Units {
		if u.GroupID == value {
			if n == 0 {
				at = unitCell(u)
			}
			n++
		}
	}
	return fmt.Sprintf("group %d (%d units)", value, n), at, n > 0
}

// goalValue is the value the record picked on the map gives a target
// parameter: a unit's id, a unit's group id or a structure's id word. A value
// another record also holds would name that record in the script, so it is
// refused.
func goalValue(m *alm.Map, param uint32, kind string, index int) (uint32, error) {
	switch {
	case param == scriptStructParam && kind == "Structure" && index < len(m.Objects):
		id := uint32(m.Objects[index].Field12)
		if id == 0 {
			return 0, fmt.Errorf("structure %d has no structure id", index)
		}
		for i, o := range m.Objects {
			if i != index && uint32(o.Field12) == id {
				return 0, fmt.Errorf("structure id %d is also structure %d's", id, i)
			}
		}
		return id, nil
	case param == scriptUnitParam && kind == "Unit" && index < len(m.Units):
		id := uint32(m.Units[index].UnitID)
		if id == 0 {
			return 0, fmt.Errorf("unit %d has no unit id", index)
		}
		for i, u := range m.Units {
			if i != index && uint32(u.UnitID) == id {
				return 0, fmt.Errorf("unit id %d is also unit %d's", id, i)
			}
		}
		return id, nil
	case param == scriptGroupParam && kind == "Unit" && index < len(m.Units):
		if id := m.Units[index].GroupID; id != 0 {
			return id, nil
		}
		return 0, fmt.Errorf("unit %d has no group id", index)
	}
	want := map[uint32]string{scriptUnitParam: "unit", scriptStructParam: "structure", scriptGroupParam: "unit (its group)"}[param]
	return 0, fmt.Errorf("this target takes a %s", want)
}

// SetGoalTarget points goal at the unit or structure record picked on the map,
// as one edit, or adds a protect check for a picked unit.
func (s *mapInspectionEdits) SetGoalTarget(goal int, kind string, index int) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		sc, err := view.Script()
		if err != nil {
			return false, err
		}
		rows := scriptGoalRows(sc)
		if goal < 0 || goal >= len(rows) {
			return false, fmt.Errorf("no script arm %d", goal)
		}
		g := rows[goal]
		value, err := goalValue(view, g.param, kind, index)
		if err != nil {
			return false, err
		}
		if g.node >= 0 {
			if value == g.value {
				return false, nil
			}
			err = m.SetField(mapedit.KindCondition, g.node, fmt.Sprintf("value%d", g.slot), int64(value))
			return err == nil, err
		}
		for _, n := range sc.Conditions {
			if n.Opcode == scriptCheckVIP && n.Type[0] == scriptUnitParam && n.Value[0] == value {
				return false, fmt.Errorf("unit %d is already protected", value)
			}
		}
		if err := registerCollision(sc); err != nil {
			return false, err
		}
		id, err := m.NextScriptID(mapedit.KindCondition)
		if err != nil {
			return false, err
		}
		rec, err := mapedit.NewScriptNode("Protect unit", scriptCheckVIP, id, mapedit.ScriptParam{Type: scriptUnitParam, Value: value})
		if err != nil {
			return false, err
		}
		_, err = m.Place(mapedit.KindCondition, rec)
		return err == nil, err
	})
}

// registerCollision refuses a new check when a script variable already uses
// the register the check would take: the build gives check i register i and a
// variable addresses the same array by its own number (MISSION-SLOT-008).
func registerCollision(sc alm.Script) error {
	next := uint32(len(sc.Conditions))
	for _, set := range []struct {
		nodes   []alm.ScriptNode
		opcodes [2]uint32
	}{{sc.Actions, [2]uint32{3, 8}}, {sc.Conditions, [2]uint32{19, 19}}} {
		for _, n := range set.nodes {
			if n.Opcode != set.opcodes[0] && n.Opcode != set.opcodes[1] {
				continue
			}
			for p, tag := range n.Type {
				if tag == 1 || tag == 5 || tag == 6 || tag == 7 {
					if n.Value[p] == next {
						return fmt.Errorf("script variable %d would share the register of the new check", next)
					}
					break
				}
			}
		}
	}
	return nil
}

// SetDropCell moves the map's drop cell to cell as one edit, or adds a drop
// action when the map has none (MISSION-DROP-002).
func (s *mapInspectionEdits) SetDropCell(cell image.Point) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := insideMap(m, cell, "drop cell"); err != nil {
			return false, err
		}
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		if cells := mapload.DropCells(view); len(cells) > 0 && cells[0] == (mapload.Cell{X: int32(cell.X), Y: int32(cell.Y)}) {
			return false, nil
		}
		err = m.SetDropCell(uint32(cell.X), uint32(cell.Y))
		return err == nil, err
	})
}
