package game

import (
	"fmt"
	"image"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The trigger wizard's writes. A row names its nodes; the host keeps a node
// the row leaves alone, rewrites in place a node no other trigger slot
// names, and makes a new node otherwise, so a change never reaches another
// trigger. A node the trigger named before and no trigger names after is
// deleted, unless deleting it would move a register a variable addresses.

// WriteTrigger writes t as one edit.
func (s *mapInspectionEdits) WriteTrigger(t ui.WizardTrigger) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		mark := m.Mark()
		if err := writeWizardTrigger(m, t); err != nil {
			return false, err
		}
		changed := m.Mark() != mark
		m.Squash(mark)
		return changed, nil
	})
}

// wizardScript is m's decoded script.
func wizardScript(m *mapedit.Editor) (*alm.Map, alm.Script, error) {
	view, err := m.Map()
	if err != nil {
		return nil, alm.Script{}, err
	}
	sc, err := view.Script()
	return view, sc, err
}

// wizardRefs counts, per node id, the trigger slots naming it: conditions in
// the pairs, actions in the action slots.
func wizardRefs(sc alm.Script) (conds, acts map[uint32]int) {
	conds, acts = map[uint32]int{}, map[uint32]int{}
	for _, tr := range sc.Triggers {
		for k := range tr.Left {
			for _, id := range []uint32{tr.Left[k], tr.Right[k]} {
				if id != 0 {
					conds[id]++
				}
			}
		}
		for _, id := range tr.Acts {
			if id != 0 {
				acts[id]++
			}
		}
	}
	return conds, acts
}

// variableRegisters are the condition registers a variable reads or writes.
func variableRegisters(sc alm.Script) map[uint32]bool {
	out := map[uint32]bool{}
	for _, n := range sc.Actions {
		if scriptAddressesVariable(false, n.Opcode) {
			out[scriptVariable(n)] = true
		}
	}
	for _, n := range sc.Conditions {
		if scriptAddressesVariable(true, n.Opcode) {
			out[scriptVariable(n)] = true
		}
	}
	return out
}

// nodeIndex is the index of the one node of nodes with id, -1 when none or
// several carry it.
func nodeIndex(nodes []alm.ScriptNode, id uint32) int {
	at := -1
	for i, n := range nodes {
		if n.ID == id {
			if at >= 0 {
				return -1
			}
			at = i
		}
	}
	return at
}

func writeWizardTrigger(m *mapedit.Editor, t ui.WizardTrigger) error {
	if len(t.Pairs) > 3 || len(t.Actions) > 4 {
		return fmt.Errorf("a trigger holds three condition pairs and four actions")
	}
	_, sc, err := wizardScript(m)
	if err != nil {
		return err
	}
	if t.Index < -1 || t.Index >= len(sc.Triggers) {
		return fmt.Errorf("no trigger %d", t.Index+1)
	}
	var before []alm.ScriptTrigger
	if t.Index >= 0 {
		before = append(before, sc.Triggers[t.Index])
	}
	condRefs, actRefs := wizardRefs(sc)
	// planned counts each node's slots in the trigger before, times 0x100,
	// plus its rows in the plan: a node rewritten in place is named once by
	// each and by no other trigger.
	planned := map[[2]uint32]int{}
	for _, tr := range before {
		for k := range tr.Left {
			planned[[2]uint32{1, tr.Left[k]}] += 0x100
			planned[[2]uint32{1, tr.Right[k]}] += 0x100
		}
		for _, id := range tr.Acts {
			planned[[2]uint32{0, id}] += 0x100
		}
	}
	for _, p := range t.Pairs {
		planned[[2]uint32{1, p.Left.ID}]++
		planned[[2]uint32{1, p.Right.ID}]++
	}
	for _, a := range t.Actions {
		planned[[2]uint32{0, a.ID}]++
	}
	// Rows fill the trigger's used slots in order, then its empty ones; a
	// slot no row reaches keeps what the map holds, and a used slot left
	// without a row is cleared.
	var pairs [3][3]uint32 // left, right, comparison
	var acts [4]uint32
	var pairSlots, actSlots []int
	if len(before) > 0 {
		old := before[0]
		for k := range old.Left {
			pairs[k] = [3]uint32{old.Left[k], old.Right[k], old.Cmp[k]}
		}
		acts = old.Acts
		for k := range old.Left {
			if old.Left[k] != 0 || old.Right[k] != 0 {
				pairSlots = append(pairSlots, k)
			}
		}
		for k := range old.Acts {
			if old.Acts[k] != 0 {
				actSlots = append(actSlots, k)
			}
		}
		for _, k := range pairSlots {
			pairs[k] = [3]uint32{}
		}
		for _, k := range actSlots {
			acts[k] = 0
		}
	}
	used := func(slots []int, k int) bool {
		for _, s := range slots {
			if s == k {
				return true
			}
		}
		return false
	}
	for k := 0; len(pairSlots) < len(t.Pairs) && k < 3; k++ {
		if !used(pairSlots, k) {
			pairSlots = append(pairSlots, k)
		}
	}
	for k := 0; len(actSlots) < len(t.Actions) && k < 4; k++ {
		if !used(actSlots, k) {
			actSlots = append(actSlots, k)
		}
	}
	for k, p := range t.Pairs {
		slot := pairSlots[k]
		keepsCmp := false
		if len(before) > 0 {
			keepsCmp = p.Left.Keep && p.Right.Keep && p.Cmp == before[0].Cmp[slot]
		}
		if p.Cmp > 5 && !keepsCmp {
			return fmt.Errorf("condition %d: comparison %d is outside 0..5", k+1, p.Cmp)
		}
		for side, n := range []ui.WizardNode{p.Left, p.Right} {
			id, err := wizardNode(m, true, n, condRefs, planned)
			if err != nil {
				return fmt.Errorf("condition %d: %w", k+1, err)
			}
			pairs[slot][side] = id
		}
		pairs[slot][2] = p.Cmp
	}
	for k, a := range t.Actions {
		id, err := wizardNode(m, false, a, actRefs, planned)
		if err != nil {
			return fmt.Errorf("action %d: %w", k+1, err)
		}
		acts[actSlots[k]] = id
	}
	index := t.Index
	if index < 0 {
		if _, err := addScriptRecord(m, mapedit.KindTrigger); err != nil {
			return err
		}
		index = m.Count(mapedit.KindTrigger) - 1
	}
	var sets []mapedit.FieldSet
	for k := 0; k < 3; k++ {
		sets = append(sets, mapedit.FieldSet{Key: fmt.Sprintf("left%d", k), Value: int64(pairs[k][0])},
			mapedit.FieldSet{Key: fmt.Sprintf("right%d", k), Value: int64(pairs[k][1])},
			mapedit.FieldSet{Key: fmt.Sprintf("cmp%d", k), Value: int64(pairs[k][2])})
	}
	for k := 0; k < 4; k++ {
		sets = append(sets, mapedit.FieldSet{Key: fmt.Sprintf("action%d", k), Value: int64(acts[k])})
	}
	once := int64(0)
	if t.Once != 0 {
		once = 1
	}
	sets = append(sets, mapedit.FieldSet{Key: "once", Value: once})
	if err := setChangedFields(m, mapedit.KindTrigger, index, sets); err != nil {
		return err
	}
	if t.Name != "" {
		if _, err := setTextFieldOn(m, mapedit.KindTrigger, index, "name", t.Name); err != nil {
			return err
		}
	}
	return dropUnnamed(m, before)
}

// setChangedFields writes the sets whose value differs from the record's.
func setChangedFields(m *mapedit.Editor, k mapedit.Kind, index int, sets []mapedit.FieldSet) error {
	var changed []mapedit.FieldSet
	for _, set := range sets {
		v, err := m.FieldValue(k, index, set.Key)
		if err != nil {
			return err
		}
		if v != set.Value {
			changed = append(changed, set)
		}
	}
	if len(changed) == 0 {
		return nil
	}
	return m.SetFields(k, index, changed...)
}

// wizardNode is the id a row's node has once written: kept, rewritten in
// place when only this slot names it, or new.
func wizardNode(m *mapedit.Editor, condition bool, n ui.WizardNode, refs map[uint32]int, planned map[[2]uint32]int) (uint32, error) {
	_, sc, err := wizardScript(m)
	if err != nil {
		return 0, err
	}
	kind, nodes, family := mapedit.KindAction, sc.Actions, uint32(0)
	if condition {
		kind, nodes, family = mapedit.KindCondition, sc.Conditions, 1
	}
	at := -1
	if n.ID != 0 {
		at = nodeIndex(nodes, n.ID)
	}
	if n.Keep {
		return n.ID, nil // a kept slot stays as the map wrote it
	}
	if n.Opcode == 0 {
		return 0, fmt.Errorf("no operation chosen")
	}
	if !condition && n.Opcode >= 0x10000 && n.Opcode != mapload.ScriptConstant {
		return 0, fmt.Errorf("operation %d does not run as a trigger action", n.Opcode)
	}
	var params []mapedit.ScriptParam
	last := 0
	for s := range n.Types {
		if n.Types[s] > 9 {
			return 0, fmt.Errorf("parameter %d: type %d is outside 0..9", s+1, n.Types[s])
		}
		if n.Types[s] != 0 {
			last = s + 1
		}
	}
	for s := 0; s < last; s++ {
		params = append(params, mapedit.ScriptParam{Type: n.Types[s], Value: n.Values[s]})
	}
	if at >= 0 && nodes[at].Opcode == n.Opcode && nodes[at].Type == n.Types && nodes[at].Value == n.Values {
		return n.ID, nil // the row names what the node already says
	}
	label := mapload.ScriptOperationName(condition, n.Opcode)
	vars := variableRegisters(sc)
	if at >= 0 && refs[n.ID] == 1 && planned[[2]uint32{family, n.ID}] == 0x101 && !(condition && vars[uint32(at)]) {
		if nodes[at].Opcode == n.Opcode {
			label = ""
		}
		if condition && n.Opcode != mapload.ScriptConstant {
			if err := registerCollides(sc, uint32(at)); err != nil {
				return 0, err
			}
		}
		return n.ID, m.SetScriptNode(kind, at, label, n.Opcode, params...)
	}
	if condition {
		if len(sc.Conditions)+1 > 100 {
			return 0, fmt.Errorf("the script already uses 100 check registers")
		}
		if err := registerCollision(sc, 1); err != nil {
			return 0, err
		}
	}
	id, err := m.NextScriptID(kind)
	if err != nil {
		return 0, err
	}
	if id > 5000 {
		return 0, fmt.Errorf("the next node ID exceeds 5000")
	}
	rec, err := mapedit.NewScriptNode(label, n.Opcode, id, params...)
	if err != nil {
		return 0, err
	}
	_, err = m.Place(kind, rec)
	return id, err
}

// dropUnnamed deletes the nodes the triggers before named that no trigger
// names now, the highest index first, each where its deletion moves no
// variable register.
func dropUnnamed(m *mapedit.Editor, before []alm.ScriptTrigger) error {
	type gone struct {
		condition bool
		id        uint32
	}
	var ids []gone
	for _, tr := range before {
		for k := range tr.Left {
			ids = append(ids, gone{true, tr.Left[k]}, gone{true, tr.Right[k]})
		}
		for _, id := range tr.Acts {
			ids = append(ids, gone{false, id})
		}
	}
	for {
		_, sc, err := wizardScript(m)
		if err != nil {
			return err
		}
		condRefs, actRefs := wizardRefs(sc)
		best, bestKind := -1, mapedit.KindAction
		for _, g := range ids {
			if g.id == 0 {
				continue
			}
			nodes, refs, kind := sc.Actions, actRefs, mapedit.KindAction
			if g.condition {
				nodes, refs, kind = sc.Conditions, condRefs, mapedit.KindCondition
			}
			at := nodeIndex(nodes, g.id)
			if at < 0 || refs[g.id] != 0 || scriptDeleteAllowed(m, kind, at) != nil {
				continue
			}
			if kind == mapedit.KindCondition && (bestKind == mapedit.KindAction || at > best) || kind == bestKind && at > best {
				best, bestKind = at, kind
			}
		}
		if best < 0 {
			return nil
		}
		if err := m.Delete(bestKind, best); err != nil {
			return err
		}
	}
}

// fixedDropCell refuses a cell the game would not land the hero on: a zero
// on either axis asks for the random fallback cell (MISSION-DROP-002).
func fixedDropCell(m *mapedit.Editor, cell image.Point) error {
	if cell.X == 0 || cell.Y == 0 {
		return fmt.Errorf("drop cell %d,%d: a zero on either axis makes the game pick a random cell instead; use a cell from 1", cell.X, cell.Y)
	}
	return insideMap(m, cell, "drop cell", ringDrop)
}

// AddDropZone adds a drop zone on cell as the shipped maps write one.
func (s *mapInspectionEdits) AddDropZone(cell image.Point) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := fixedDropCell(m, cell); err != nil {
			return false, err
		}
		_, err := m.AddDropZone(uint32(cell.X), uint32(cell.Y))
		return err == nil, err
	})
}

// MoveDropZone moves zone's cell to cell.
func (s *mapInspectionEdits) MoveDropZone(zone int, cell image.Point) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := fixedDropCell(m, cell); err != nil {
			return false, err
		}
		at, err := dropZoneAction(m, zone)
		if err != nil {
			return false, err
		}
		_, sc, err := wizardScript(m)
		if err != nil {
			return false, err
		}
		n := sc.Actions[at]
		if n.Type[0] == mapload.ScriptX && n.Type[1] == mapload.ScriptY && n.Value[0] == uint32(cell.X) && n.Value[1] == uint32(cell.Y) {
			return false, nil
		}
		params := []mapedit.ScriptParam{{Type: mapload.ScriptX, Value: uint32(cell.X)}, {Type: mapload.ScriptY, Value: uint32(cell.Y)}}
		for k := 2; k < len(n.Type); k++ {
			params = append(params, mapedit.ScriptParam{Type: n.Type[k], Value: n.Value[k]})
		}
		err = m.SetScriptNode(mapedit.KindAction, at, "", n.Opcode, params...)
		return err == nil, err
	})
}

// DeleteDropZone deletes zone's action; a trigger that named only it goes
// with it, and another trigger's slot naming it is cleared.
func (s *mapInspectionEdits) DeleteDropZone(zone int) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		mark := m.Mark()
		at, err := dropZoneAction(m, zone)
		if err != nil {
			return false, err
		}
		_, sc, err := wizardScript(m)
		if err != nil {
			return false, err
		}
		id := sc.Actions[at].ID
		if nodeIndex(sc.Actions, id) < 0 {
			return false, fmt.Errorf("drop zone %d's action ID %d names more than one action", zone+1, id)
		}
		for t := len(sc.Triggers) - 1; t >= 0; t-- {
			tr := sc.Triggers[t]
			if !slices.Contains(tr.Acts[:], id) {
				continue
			}
			only := tr.Left == [3]uint32{} && tr.Right == [3]uint32{}
			for _, a := range tr.Acts {
				only = only && (a == 0 || a == id)
			}
			if only {
				if err := m.Delete(mapedit.KindTrigger, t); err != nil {
					return false, err
				}
				continue
			}
			var sets []mapedit.FieldSet
			for k, a := range tr.Acts {
				if a == id {
					sets = append(sets, mapedit.FieldSet{Key: fmt.Sprintf("action%d", k), Value: 0})
				}
			}
			if err := m.SetFields(mapedit.KindTrigger, t, sets...); err != nil {
				return false, err
			}
		}
		if err := m.Delete(mapedit.KindAction, at); err != nil {
			return false, err
		}
		m.Squash(mark)
		return true, nil
	})
}

// dropZoneAction is the action index of drop zone zone.
func dropZoneAction(m *mapedit.Editor, zone int) (int, error) {
	zones, err := m.DropZones()
	if err != nil {
		return 0, err
	}
	if zone < 0 || zone >= len(zones) {
		return 0, fmt.Errorf("no drop zone %d", zone+1)
	}
	return zones[zone], nil
}

var (
	_ ui.InspectionTriggerWizard = (*mapInspectionEdits)(nil)
	_ ui.InspectionDropZones     = (*mapInspectionEdits)(nil)
)
