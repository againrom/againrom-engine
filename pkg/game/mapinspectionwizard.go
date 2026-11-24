package game

import (
	"errors"
	"image"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// The trigger wizard's writes. A row names its nodes; the host keeps a node
// the row leaves alone, rewrites in place a node no other trigger slot
// names, and makes a new node otherwise, so a change never reaches another
// trigger. A node the trigger named before and no trigger names after is
// deleted, unless deleting it would move a register a variable addresses.

// WriteTrigger writes t as one edit, or with t.Join as part of the history
// entry before it, so one undo takes back both. A refusal for want of a
// register reads in the editor's words.
func (s *mapInspectionEdits) WriteTrigger(t ui.WizardTrigger) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	steps := len(s.undoWorkspace)
	d, err := s.change(func(m *mapedit.Editor) (bool, error) {
		mark := m.Mark()
		if err := writeWizardTrigger(m, t); err != nil {
			return false, s.checkWords(err)
		}
		changed := m.Mark() != mark
		m.Squash(mark)
		return changed, nil
	})
	if t.Join && d != nil && steps > 0 && len(s.undoWorkspace) == steps+1 {
		s.undoWorkspace = s.undoWorkspace[:steps]
	}
	return d, err
}

// checkWords is a refusal the game words, a check register taken or full
// among them, in the editor's words; any other error as it is.
func (s *mapInspectionEdits) checkWords(err error) error {
	var taken registerTaken
	if errors.As(err, &taken) {
		err = lifted(err, "editor.wizard.registers", taken.register, taken.max)
	}
	return s.inspector.inLanguage(err)
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

// wizardRoom refuses a plan the map's trigger form cannot hold: more pairs
// or actions than the profile's, or on an original map a second group.
func wizardRoom(t ui.WizardTrigger, limits scriptprofile.Profile, extended bool) error {
	limit := &mapedit.ScriptLimitError{Limit: mapedit.LimitPairs, Max: int64(limits.Pairs), Got: int64(len(t.Pairs))}
	plan := planScript(t)
	globals, dialogues := mapedit.CountGlobalNodes(plan), mapedit.CountInteractive(plan.Actions)
	switch {
	case len(t.Pairs) > limits.Pairs:
	case len(t.Actions) > limits.Actions:
		limit = &mapedit.ScriptLimitError{Limit: mapedit.LimitActions, Max: int64(limits.Actions), Got: int64(len(t.Actions))}
	case !extended && slices.ContainsFunc(t.Pairs, func(p ui.WizardPair) bool { return p.Group != 0 }):
		limit = &mapedit.ScriptLimitError{Limit: mapedit.LimitGroups, Max: 1, Got: 2}
	case !extended && dialogues > 0:
		n := dialogues
		limit = &mapedit.ScriptLimitError{Limit: mapedit.LimitDialogue, Got: int64(n)}
		return lifted(limit, "editor.script.limit.dialogue", n)
	case !extended && globals > 0:
		limit = &mapedit.ScriptLimitError{Limit: mapedit.LimitGlobal, Got: int64(globals)}
		return lifted(limit, "editor.script.limit.global", globals)
	default:
		return nil
	}
	if extended {
		return lifted(limit, "editor.script.triggerfullx", limits.Pairs, limits.Actions)
	}
	return lifted(limit, "editor.script.triggerfull")
}

// planScript holds the plan's nodes as the script the fold counts: each
// condition side and each action.
func planScript(t ui.WizardTrigger) alm.Script {
	var sc alm.Script
	for _, p := range t.Pairs {
		sc.Conditions = append(sc.Conditions, planNode(p.Left), planNode(p.Right))
	}
	for _, a := range t.Actions {
		sc.Actions = append(sc.Actions, planNode(a))
	}
	return sc
}

// planNode is the node a wizard row writes; a node kept as it stands has no
// opcode and counts for nothing.
func planNode(n ui.WizardNode) alm.ScriptNode {
	if n.Keep {
		return alm.ScriptNode{}
	}
	return alm.ScriptNode{Opcode: n.Opcode, Label: n.Label, Type: n.Types, Value: n.Values}
}

func writeWizardTrigger(m *mapedit.Editor, t ui.WizardTrigger) error {
	extended := m.Extended()
	limits := m.Profile()
	if err := wizardRoom(t, limits, extended); err != nil {
		return err
	}
	_, sc, err := wizardScript(m)
	if err != nil {
		return err
	}
	if t.Index < -1 || t.Index >= len(sc.Triggers) {
		return wordErr("editor.err.notrigger", t.Index+1)
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
	// without a row is cleared. An extended trigger holds the rows alone.
	pairs, acts := limits.Pairs, limits.Actions
	if extended {
		pairs, acts = len(t.Pairs), len(t.Actions)
	}
	next := alm.ScriptTrigger{Left: make([]uint32, pairs), Right: make([]uint32, pairs),
		Cmp: make([]uint32, pairs), Group: make([]uint32, pairs), Acts: make([]uint32, acts)}
	var pairSlots, actSlots []int
	if len(before) > 0 && !extended {
		old := before[0]
		for k := range next.Left {
			next.Left[k], next.Right[k], next.Cmp[k] = old.Pair(k)
		}
		for k := range next.Acts {
			next.Acts[k] = old.Act(k)
		}
		for k := range next.Left {
			if next.Left[k] != 0 || next.Right[k] != 0 {
				pairSlots = append(pairSlots, k)
			}
		}
		for k := range next.Acts {
			if next.Acts[k] != 0 {
				actSlots = append(actSlots, k)
			}
		}
		for _, k := range pairSlots {
			next.Left[k], next.Right[k], next.Cmp[k] = 0, 0, 0
		}
		for _, k := range actSlots {
			next.Acts[k] = 0
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
	for k := 0; len(pairSlots) < len(t.Pairs) && k < pairs; k++ {
		if !used(pairSlots, k) {
			pairSlots = append(pairSlots, k)
		}
	}
	for k := 0; len(actSlots) < len(t.Actions) && k < acts; k++ {
		if !used(actSlots, k) {
			actSlots = append(actSlots, k)
		}
	}
	for k, p := range t.Pairs {
		slot := pairSlots[k]
		keepsCmp := false
		if len(before) > 0 {
			_, _, held := before[0].Pair(slot)
			keepsCmp = p.Cmp == held
		}
		if p.Cmp > limits.MaxCompare && !keepsCmp {
			return wordErr("editor.err.condcompare", k+1, p.Cmp, limits.MaxCompare)
		}
		for side, n := range []ui.WizardNode{p.Left, p.Right} {
			id, err := wizardNode(m, true, n, condRefs, planned)
			if err != nil {
				return wordErr("editor.err.condition", k+1, err)
			}
			if side == 0 {
				next.Left[slot] = id
			} else {
				next.Right[slot] = id
			}
		}
		next.Cmp[slot], next.Group[slot] = p.Cmp, p.Group
	}
	for k, a := range t.Actions {
		id, err := wizardNode(m, false, a, actRefs, planned)
		if err != nil {
			return wordErr("editor.err.action", k+1, err)
		}
		next.Acts[actSlots[k]] = id
	}
	index := t.Index
	if index < 0 {
		if _, err := addScriptRecord(m, mapedit.KindTrigger); err != nil {
			return err
		}
		index = m.Count(mapedit.KindTrigger) - 1
	}
	if t.Once != 0 {
		next.Once = 1
	}
	if extended {
		next = packTrigger(next, true)
	}
	if err := m.SetTriggerSlots(index, next); err != nil {
		return err
	}
	if t.Name != "" {
		if _, err := setTextFieldOn(m, mapedit.KindTrigger, index, "name", t.Name); err != nil {
			return err
		}
	}
	return dropUnnamed(m, before)
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
		return 0, wordErr("editor.err.noop")
	}
	if !condition && n.Opcode >= 0x10000 && n.Opcode != mapload.ScriptConstant {
		return 0, wordErr("editor.err.opaction", n.Opcode)
	}
	var params []mapedit.ScriptParam
	last := 0
	for s := range n.Types {
		if n.Types[s] > 9 {
			return 0, wordErr("editor.err.paramtype", s+1, n.Types[s])
		}
		if n.Types[s] != 0 {
			last = s + 1
		}
	}
	view, err := m.Map()
	if err != nil {
		return 0, err
	}
	// A format whose trigger window writes the narrower node edit holds a row's
	// values to that edit's parameter check, in the same words.
	narrow := !rowFor(m).wholeScripts
	for s := 0; s < last; s++ {
		err := scriptObjectAllowed(view, m.Profile().Ops, n.Types[s], int64(n.Values[s]))
		if narrow {
			err = scriptParamAllowed(view, m.Profile(), condition, n.Opcode, s, n.Types[s], n.Values[s])
		}
		if err != nil {
			return 0, err
		}
		params = append(params, mapedit.ScriptParam{Type: n.Types[s], Value: n.Values[s]})
	}
	if at >= 0 && nodes[at].Opcode == n.Opcode && nodes[at].Type == n.Types && nodes[at].Value == n.Values &&
		(n.Label == "" && !nodes[at].InteractiveMessage() || n.Label == nodes[at].Label) {
		return n.ID, nil // the row names what the node already says
	}
	limits := m.Profile()
	label := mapload.ScriptOperationNameIn(limits.Ops, condition, n.Opcode)
	if n.Label != "" {
		label = n.Label
	}
	vars := variableRegisters(sc)
	if at >= 0 && refs[n.ID] == 1 && planned[[2]uint32{family, n.ID}] == 0x101 && !(condition && vars[uint32(at)]) {
		if nodes[at].Opcode == n.Opcode && n.Label == "" && !nodes[at].InteractiveMessage() {
			label = ""
		}
		if condition && n.Opcode != mapload.ScriptConstant {
			if err := registerCollides(sc, limits, uint32(at)); err != nil {
				return 0, err
			}
		}
		return n.ID, m.SetScriptNode(kind, at, label, n.Opcode, params...)
	}
	if condition {
		if limits.Registers > 0 && len(sc.Conditions)+1 > limits.Registers {
			return 0, errChecksFull(limits)
		}
		if err := registerCollision(sc, limits, 1); err != nil {
			return 0, err
		}
	}
	id, err := m.NextScriptID(kind)
	if err != nil {
		return 0, err
	}
	if maxID := limits.MaxID; id > maxID {
		return 0, wordErr("editor.err.nodeid", maxID)
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
	var ids []goneNode
	for _, tr := range before {
		for k := range tr.Left {
			ids = append(ids, goneNode{true, tr.Left[k]}, goneNode{true, tr.Right[k]})
		}
		for _, id := range tr.Acts {
			ids = append(ids, goneNode{false, id})
		}
	}
	return dropNodes(m, ids, nil)
}

// goneNode is a script node a write may have left with no name.
type goneNode struct {
	condition bool
	id        uint32
}

// dropNodes deletes the nodes of ids that no trigger names and held does not
// hold, the highest index first, each where its deletion moves no variable
// register.
func dropNodes(m *mapedit.Editor, ids []goneNode, held func(g goneNode) bool) error {
	type gone = goneNode
	for {
		_, sc, err := wizardScript(m)
		if err != nil {
			return err
		}
		condRefs, actRefs := wizardRefs(sc)
		best, bestKind := -1, mapedit.KindAction
		for _, g := range ids {
			if g.id == 0 || held != nil && held(g) {
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
func (s *mapInspectionEdits) fixedDropCell(m *mapedit.Editor, cell image.Point) error {
	if cell.X == 0 || cell.Y == 0 {
		return wordErr("editor.err.dropcell", cell.X, cell.Y)
	}
	return s.admitCell(m, cell, "drop cell", ringDrop, admitAdd)
}

// AddDropZone adds a drop zone on cell as the shipped maps write one.
func (s *mapInspectionEdits) AddDropZone(cell image.Point) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := s.fixedDropCell(m, cell); err != nil {
			return false, err
		}
		_, err := m.AddDropZone(uint32(cell.X), uint32(cell.Y))
		return err == nil, err
	})
}

// MoveDropZone moves zone's cell to cell.
func (s *mapInspectionEdits) MoveDropZone(zone int, cell image.Point) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := s.fixedDropCell(m, cell); err != nil {
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
func (s *mapInspectionEdits) DeleteDropZone(zone int) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
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
			return false, wordErr("editor.err.dropaction", zone+1, id)
		}
		for t := len(sc.Triggers) - 1; t >= 0; t-- {
			tr := sc.Triggers[t]
			if !slices.Contains(tr.Acts[:], id) {
				continue
			}
			only := true
			for k := range tr.Pairs() {
				left, right, _ := tr.Pair(k)
				only = only && (left == 0 || right == 0)
			}
			for _, a := range tr.Acts {
				only = only && (a == 0 || a == id)
			}
			if only {
				if err := m.Delete(mapedit.KindTrigger, t); err != nil {
					return false, err
				}
				continue
			}
			next := tr
			next.Acts = slices.Clone(tr.Acts)
			for k, a := range next.Acts {
				if a == id {
					next.Acts[k] = 0
				}
			}
			if err := m.SetTriggerSlots(t, next); err != nil {
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
		return 0, wordErr("editor.err.nodropzone", zone+1)
	}
	return zones[zone], nil
}

var (
	_ ui.InspectionTriggerWizard = (*mapInspectionEdits)(nil)
	_ ui.InspectionDropZones     = (*mapInspectionEdits)(nil)
)
