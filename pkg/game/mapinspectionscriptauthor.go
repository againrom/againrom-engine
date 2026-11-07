package game

import (
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// scriptKind reports a model section that holds script nodes.
func scriptKind(k mapedit.Kind) bool {
	return k == mapedit.KindAction || k == mapedit.KindCondition || k == mapedit.KindTrigger
}

func (s *mapInspectionEdits) AddScript(kind ui.RecordKind) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil || !kind.Scripted() {
		return nil, fmt.Errorf("no script kind %q", kind)
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		_, err := addScriptRecord(m, k)
		return err == nil, err
	})
}

// addScriptRecord places a new script record of kind k: a trigger named
// "Trigger" that fires once, a constant 0 condition or a "Send message 1"
// action under the next free node id. It returns the new node's id.
func addScriptRecord(m *mapedit.Editor, k mapedit.Kind) (uint32, error) {
	var rec []byte
	var id uint32
	limits := scriptprofile.Of(m.Extended())
	if k == mapedit.KindTrigger {
		if n := m.Count(k); n >= limits.Triggers {
			return 0, lifted(&mapedit.ScriptLimitError{Limit: mapedit.LimitTriggers, Max: int64(limits.Triggers), Got: int64(n)},
				fmt.Sprintf("the script already holds %d triggers", limits.Triggers))
		}
		var err error
		if rec, err = m.NewTrigger(alm.ScriptTrigger{Name: "Trigger", Once: 1}); err != nil {
			return 0, err
		}
	} else {
		if k == mapedit.KindCondition && limits.Registers > 0 && m.Count(k) >= limits.Registers {
			return 0, errChecksFull
		}
		var err error
		if id, err = m.NextScriptID(k); err != nil {
			return 0, err
		}
		if id > limits.MaxID {
			return 0, fmt.Errorf("the next node ID exceeds %d", limits.MaxID)
		}
		op, value := uint32(2), uint32(1)
		if k == mapedit.KindCondition {
			op, value = 0x10002, 0
		}
		if rec, err = mapedit.NewScriptNode(mapload.ScriptOperationName(k == mapedit.KindCondition, op), op, id, mapedit.ScriptParam{Type: 1, Value: value}); err != nil {
			return 0, err
		}
	}
	_, err := m.Place(k, rec)
	return id, err
}

func scriptNodes(sc alm.Script, k mapedit.Kind) []alm.ScriptNode {
	if k == mapedit.KindCondition {
		return sc.Conditions
	}
	return sc.Actions
}

func scriptReferenced(sc alm.Script, k mapedit.Kind, id uint32) bool {
	if id == 0 {
		return false
	}
	for _, t := range sc.Triggers {
		if k == mapedit.KindAction {
			for _, v := range t.Acts {
				if v == id {
					return true
				}
			}
		} else {
			for p := range t.Left {
				if t.Left[p] == id || t.Right[p] == id {
					return true
				}
			}
		}
	}
	return false
}

func scriptDeleteAllowed(m *mapedit.Editor, k mapedit.Kind, index int) error {
	if !scriptKind(k) || k == mapedit.KindTrigger {
		return nil
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	sc, err := view.Script()
	if err != nil {
		return err
	}
	nodes := scriptNodes(sc, k)
	if index < 0 || index >= len(nodes) {
		return fmt.Errorf("no script node %d", index)
	}
	if scriptReferenced(sc, k, nodes[index].ID) {
		return fmt.Errorf("node ID %d is referenced by a trigger; clear its references first", nodes[index].ID)
	}
	if k == mapedit.KindCondition {
		for _, family := range []struct {
			nodes     []alm.ScriptNode
			condition bool
		}{{sc.Actions, false}, {sc.Conditions, true}} {
			for _, n := range family.nodes {
				if !(family.condition && n.Opcode == 19) && !(!family.condition && (n.Opcode == 3 || n.Opcode == 8)) {
					continue
				}
				v := scriptVariable(n)
				if v >= uint32(index) && v < uint32(len(nodes)) {
					return fmt.Errorf("deleting condition %d would change variable register %d; remove its variable references first", nodes[index].ID, v)
				}
			}
		}
	}
	return nil
}

func scriptFieldAllowed(m *mapedit.Editor, k mapedit.Kind, index int, key string, value int64) error {
	if !scriptKind(k) {
		return nil
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	sc, err := view.Script()
	if err != nil {
		return err
	}
	if k == mapedit.KindTrigger {
		if strings.HasPrefix(key, "left") || strings.HasPrefix(key, "right") || strings.HasPrefix(key, "action") {
			if value == 0 {
				return nil
			}
			nodes := sc.Conditions
			if strings.HasPrefix(key, "action") {
				nodes = sc.Actions
			}
			matches := 0
			for _, n := range nodes {
				if int64(n.ID) == value && !(strings.HasPrefix(key, "action") && n.Opcode >= 0x10000) {
					matches++
				}
			}
			if matches != 1 {
				return fmt.Errorf("reference %d resolves to %d executable nodes", value, matches)
			}
		}
		return nil
	}
	nodes := scriptNodes(sc, k)
	if index < 0 || index >= len(nodes) {
		return fmt.Errorf("no script node %d", index)
	}
	if key == "opcode" {
		found := false
		for _, op := range mapload.ScriptOperations(k == mapedit.KindCondition) {
			if int64(op.Opcode) == value {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("operation %d is not supported for authoring", value)
		}
		if k == mapedit.KindCondition && value != 0x10002 {
			if err := registerCollides(sc, uint32(index)); err != nil {
				return err
			}
		}
	}
	if err := againromValueAllowed(k == mapedit.KindCondition, nodes[index], key, value); err != nil {
		return err
	}
	if key == "id" {
		for i, n := range nodes {
			if i != index && int64(n.ID) == value {
				return fmt.Errorf("node ID %d is already used", value)
			}
		}
		if scriptReferenced(sc, k, nodes[index].ID) {
			return fmt.Errorf("node ID %d is referenced; clear its references before renaming", nodes[index].ID)
		}
	}
	return nil
}

// againromValueAllowed asks sim.AgainromNodeValue about the value a plain
// field key of node n would hold (DIV-4009).
func againromValueAllowed(condition bool, n alm.ScriptNode, key string, value int64) error {
	slot, err := strconv.Atoi(strings.TrimPrefix(key, "value"))
	if !strings.HasPrefix(key, "value") || err != nil || slot < 0 || slot >= len(n.Type) || n.Type[slot] != mapload.ScriptInt {
		return nil
	}
	return sim.AgainromNodeValue(condition, int32(n.Opcode), value)
}

func scriptVariableAllowed(m *mapedit.Editor, k mapedit.Kind, index int) error {
	if k != mapedit.KindAction && k != mapedit.KindCondition {
		return nil
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	sc, err := view.Script()
	if err != nil {
		return err
	}
	n := scriptNodes(sc, k)[index]
	// The set may have made the node Againrom's after its values were
	// checked; its values are checked again on the node the set leaves.
	for slot := range n.Value {
		if err := againromValueAllowed(k == mapedit.KindCondition, n, "value"+strconv.Itoa(slot), int64(int32(n.Value[slot]))); err != nil {
			return err
		}
	}
	if !scriptAddressesVariable(k == mapedit.KindCondition, n.Opcode) {
		return nil
	}
	return scriptRegisterAllowed(sc, scriptVariable(n))
}

func scriptAddressesVariable(condition bool, opcode uint32) bool {
	return condition && opcode == 19 || !condition && (opcode == 3 || opcode == 8)
}

// scriptRegisterAllowed refuses a variable the map's script cannot hold: on
// an extended map one past the variable array; on an original map one past
// the hundred registers, a limit the extended record lifts, or one whose
// register a check result takes.
func scriptRegisterAllowed(sc alm.Script, variable uint32) error {
	if limit := uint32(scriptprofile.Extended.Variables); sc.Extended {
		if variable >= limit {
			return fmt.Errorf("variable %d is outside 0..%d", variable, limit-1)
		}
		return nil
	}
	if limits := scriptprofile.First; variable >= uint32(limits.Registers) {
		return lifted(&mapedit.ScriptLimitError{Limit: mapedit.LimitVariables, Max: int64(limits.Registers), Got: int64(variable)},
			fmt.Sprintf("variable register %d is outside 0..%d", variable, limits.Registers-1))
	}
	if int(variable) < len(sc.Conditions) && sc.Conditions[variable].Opcode != 0x10002 {
		return fmt.Errorf("variable register %d is overwritten by condition %d; use a constant/preset slot or a free register", variable, sc.Conditions[variable].ID)
	}
	return nil
}

func scriptVariable(n alm.ScriptNode) uint32 {
	if p := scriptVariableSlot(n.Type); p >= 0 {
		return n.Value[p]
	}
	return 0
}

func scriptVariableSlot(types [10]uint32) int {
	for p, tag := range types {
		if tag == 1 || tag == 5 || tag == 6 || tag == 7 {
			return p
		}
	}
	return -1
}

func scriptParameterKind(condition bool, opcode uint32, types [10]uint32, slot int) uint32 {
	if !condition && opcode == 2 && slot == 0 && types[slot] == mapload.ScriptInt {
		return ui.ScriptMessageParameter
	}
	if !condition && (types[slot] == 1 || types[slot] == 7) {
		argument := 0
		for _, tag := range types[:slot] {
			switch tag {
			case 2, 3, 4, 8, 9:
			default:
				argument++
			}
		}
		switch {
		case opcode == 21 && argument == 4, opcode == 24 && argument == 2, opcode == 30 && argument == 0:
			return ui.ScriptSpellParameter
		case opcode == 25 && argument == 0:
			return ui.ScriptEntrySpellParameter
		case opcode == 29 && argument == 2:
			return ui.ScriptAreaSpellParameter
		case opcode == 34 && argument == 0:
			return ui.ScriptPropertyParameter
		}
	}
	if scriptAddressesVariable(condition, opcode) && scriptVariableSlot(types) == slot {
		return ui.ScriptVariableParameter
	}
	if types[slot] >= ui.ScriptVariableParameter && types[slot] <= ui.ScriptPropertyParameter {
		return 0
	}
	return types[slot]
}

func scriptCoordinateAllowed(view *alm.Map, k mapedit.Kind, index int, key string, value int64) error {
	if (k != mapedit.KindAction && k != mapedit.KindCondition) || !strings.HasPrefix(key, "value") {
		return nil
	}
	slot, err := strconv.Atoi(strings.TrimPrefix(key, "value"))
	if err != nil || slot < 0 || slot >= 10 {
		return nil
	}
	sc, err := view.Script()
	if err != nil {
		return err
	}
	n := scriptNodes(sc, k)[index]
	limit := 0
	if n.Type[slot] == 5 {
		limit = view.Width
	}
	if n.Type[slot] == 6 {
		limit = view.Height
	}
	if limit > 0 && (value < 0 || value >= int64(limit)) {
		return fmt.Errorf("coordinate %d is outside 0..%d", value, limit-1)
	}
	return nil
}

// LabelVariable sets the label of a variable register; an empty label
// clears it. The labels are workspace edits, saved in the labels file.
func (s *mapInspectionEdits) LabelVariable(register uint32, label string) (*ui.InspectionDocument, error) {
	label = strings.TrimSpace(label)
	if label != "" && !labelAllowed(label) {
		return nil, fmt.Errorf(s.inspector.word("editor.labels.bad"), maxLabelRunes)
	}
	variable := false
	for _, t := range s.scriptTargets[ui.ScriptVariableParameter] {
		variable = variable || t.Value == register
	}
	if !variable {
		return nil, fmt.Errorf(s.inspector.word("editor.labels.notvariable"), register)
	}
	return s.changeWorkspace(func(w *editorWorkspaceState) error {
		if label == "" {
			delete(w.Labels, register)
			if len(w.Labels) == 0 {
				w.Labels = nil
			}
			return nil
		}
		if w.Labels == nil {
			w.Labels = map[uint32]string{}
		}
		w.Labels[register] = label
		return nil
	})
}
