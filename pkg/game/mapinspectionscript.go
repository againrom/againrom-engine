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
	partySlot         = 1 // the roster slot the party plays as
	scriptUnitParam   = 4
	scriptStructParam = 9
	armWin            = "Win"
	armLose           = "Lose"
	armProtect        = "Protect"
)

// goalRow is one target parameter an arm reads: the arm, the trigger that
// carries it, the condition node and parameter slot that holds it. A row with
// node -1 adds a record instead: a protect check, or the arm kind named by
// add.
type goalRow struct {
	arm           string
	trigger, node int
	slot          int
	param, value  uint32
	add           addOp
}

// addOp is what a row with no node adds.
type addOp int

const (
	addNothing addOp = iota
	addProtect
	addWinUnit
	addWinGroup
	addWinStructure
	addLoseUnit
)

// addArms maps the rows that add an arm to the arm table, with the text each
// shows and the parameter type its click names.
var addArms = map[addOp]struct {
	kind  mapedit.ArmKind
	arm   string
	param uint32
	text  string
}{
	addWinUnit:      {mapedit.ArmWinUnit, armWin, scriptUnitParam, "add: the death of a unit wins"},
	addWinGroup:     {mapedit.ArmWinGroup, armWin, scriptGroupParam, "add: a group wiped out wins"},
	addWinStructure: {mapedit.ArmWinStructure, armWin, scriptStructParam, "add: a structure destroyed wins"},
	addLoseUnit:     {mapedit.ArmLoseUnit, armLose, scriptUnitParam, "add: the death of a party unit loses"},
}

// scriptGoalRows lists the target parameters of the script's win, lose and
// protect arms in trigger order, then check 18 order. Rows that add a record
// follow (node -1): the three win arms when the map has no win arm row, the
// lose arm when it has no lose arm row, and always, last, the protect check. A
// trigger names its condition nodes by id (ALM-TRIG-047).
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
	has := map[string]bool{}
	for _, r := range rows {
		has[r.arm] = true
	}
	for _, op := range []addOp{addWinUnit, addWinGroup, addWinStructure, addLoseUnit} {
		if a := addArms[op]; !has[a.arm] {
			rows = append(rows, goalRow{arm: a.arm, trigger: -1, node: -1, slot: -1, param: a.param, add: op})
		}
	}
	return append(rows, goalRow{arm: armProtect, trigger: -1, node: -1, slot: -1, param: scriptUnitParam, add: addProtect})
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
		text := "add a check that loses the mission when a unit dies"
		if g.add != addProtect {
			text = addArms[g.add].text
		}
		return ui.InspectionGoal{Arm: g.arm, Pick: pick, Index: -1, Text: text}
	}
	n := sc.Conditions[g.node]
	who := fmt.Sprintf("check %d", n.ID)
	if g.trigger >= 0 {
		who = fmt.Sprintf("trigger %q", sc.Triggers[g.trigger].Name)
	}
	target, cell, has := describeTarget(m, g.param, g.value)
	openKind, openIndex := "Condition", g.node
	if g.trigger >= 0 {
		openKind, openIndex = "Trigger", g.trigger
	}
	return ui.InspectionGoal{Arm: g.arm, Pick: pick, Kind: "Condition", Index: g.node, Cell: cell, HasCell: has,
		OpenKind: openKind, OpenIndex: openIndex,
		Text: fmt.Sprintf("%s: %s, %s", who, strings.ToLower(inspectionOperation("Condition", n.Opcode)), target)}
}

func (s *mapInspectionEdits) DropGoals() []ui.InspectionGoal {
	m, err := s.model.Map()
	if err != nil {
		return nil
	}
	sc, err := m.Script()
	if err != nil {
		return nil
	}
	var rows []ui.InspectionGoal
	for i, n := range sc.Actions {
		if n.Opcode == 0x10002 {
			rows = append(rows, ui.InspectionGoal{Arm: "Drop", Kind: "Action", Index: i,
				OpenKind: "Action", OpenIndex: i, Cell: image.Pt(int(n.Value[0]), int(n.Value[1])), HasCell: true})
		}
	}
	return rows
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
		if a, ok := addArms[g.add]; ok && g.node < 0 {
			if g.add == addLoseUnit && view.Units[index].Owner != partySlot {
				return false, fmt.Errorf("a lose arm takes a unit of the party's side, roster slot %d; unit %d belongs to slot %d", partySlot, view.Units[index].UnitID, view.Units[index].Owner)
			}
			if g.add == addWinStructure && s.inspector != nil && s.inspector.table != nil {
				if err := structureStartsStanding(view, s.inspector.table, index); err != nil {
					return false, err
				}
			}
			if err := registerCollision(sc, mapedit.ArmConditions); err != nil {
				return false, err
			}
			err = m.AddArm(a.kind, value)
			return err == nil, err
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
		if err := registerCollision(sc, 1); err != nil {
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

// structureStartsStanding refuses a structure the game builds with no health:
// its health check would hold at the first pass and win the mission at once.
func structureStartsStanding(m *alm.Map, t *mapload.Table, index int) error {
	built := mapload.Structures(m, t)
	if index < 0 || index >= len(built) {
		return fmt.Errorf("no structure %d", index)
	}
	if int16(built[index].Field42) <= 0 {
		return fmt.Errorf("structure %d starts the mission with no health, so the arm would win it at once", index)
	}
	return nil
}

// registerCollision refuses new checks when a script variable already uses a
// register one of them would take: the build gives check i register i and a
// variable addresses the same array by its own number (MISSION-SLOT-008).
func registerCollision(sc alm.Script, added int) error {
	for k := range added {
		if err := registerCollides(sc, uint32(len(sc.Conditions)+k)); err != nil {
			return err
		}
	}
	return nil
}

func registerCollides(sc alm.Script, next uint32) error {
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
