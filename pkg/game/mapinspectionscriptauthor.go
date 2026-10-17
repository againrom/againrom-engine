package game

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func scriptKind(k mapedit.Kind) bool {
	return k == mapedit.KindAction || k == mapedit.KindCondition || k == mapedit.KindTrigger
}

func (s *mapInspectionEdits) AddScript(kind string) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil || !scriptKind(k) {
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
	if k == mapedit.KindTrigger {
		if m.Count(k) >= 1000 {
			return 0, fmt.Errorf("the script already holds 1000 triggers")
		}
		rec = make([]byte, 184)
		copy(rec, "Trigger")
		binary.LittleEndian.PutUint32(rec[0xb4:], 1)
	} else {
		if k == mapedit.KindCondition && m.Count(k) >= 100 {
			return 0, fmt.Errorf("the script already uses 100 check registers")
		}
		var err error
		if id, err = m.NextScriptID(k); err != nil {
			return 0, err
		}
		if id > 5000 {
			return 0, fmt.Errorf("the next node ID exceeds 5000")
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

func scriptFieldChoices(kind, key string, sc alm.Script) []ui.InspectionChoice {
	var out []ui.InspectionChoice
	switch {
	case key == "opcode":
		for _, row := range mapload.ScriptOperations(kind == "Condition") {
			out = append(out, ui.InspectionChoice{Value: int64(row.Opcode), Label: row.Name})
		}
	case strings.HasPrefix(key, "type"):
		for i, name := range mapload.ScriptParameterNames {
			out = append(out, ui.InspectionChoice{Value: int64(i), Label: name})
		}
	case strings.HasPrefix(key, "cmp"):
		for i, name := range mapload.ScriptComparisons {
			out = append(out, ui.InspectionChoice{Value: int64(i), Label: name})
		}
	case key == "once":
		out = []ui.InspectionChoice{{Value: 0, Label: "Repeat while true"}, {Value: 1, Label: "At most once"}}
	case kind == "Trigger" && (strings.HasPrefix(key, "left") || strings.HasPrefix(key, "right") || strings.HasPrefix(key, "action")):
		out = []ui.InspectionChoice{{Value: 0, Label: "Empty"}}
		nodes := sc.Conditions
		if strings.HasPrefix(key, "action") {
			nodes = sc.Actions
		}
		counts := map[uint32]int{}
		for _, n := range nodes {
			counts[n.ID]++
		}
		for _, n := range nodes {
			if n.ID > 0 && counts[n.ID] == 1 && !(strings.HasPrefix(key, "action") && n.Opcode >= 0x10000) {
				out = append(out, ui.InspectionChoice{Value: int64(n.ID), Label: fmt.Sprintf("%d: %s", n.ID, n.Label)})
			}
		}
	}
	return out
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
	if !(k == mapedit.KindAction && (n.Opcode == 3 || n.Opcode == 8)) && !(k == mapedit.KindCondition && n.Opcode == 19) {
		return nil
	}
	variable := scriptVariable(n)
	if variable >= 100 {
		return fmt.Errorf("variable register %d is outside 0..99", variable)
	}
	if int(variable) < len(sc.Conditions) && sc.Conditions[variable].Opcode != 0x10002 {
		return fmt.Errorf("variable register %d is overwritten by condition %d; use a constant/preset slot or a free register", variable, sc.Conditions[variable].ID)
	}
	return nil
}

func scriptVariable(n alm.ScriptNode) uint32 {
	for p, tag := range n.Type {
		if tag == 1 || tag == 5 || tag == 6 || tag == 7 {
			return n.Value[p]
		}
	}
	return 0
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

func scriptValueChoices(view *alm.Map, k mapedit.Kind, index int, key string) []ui.InspectionChoice {
	if k != mapedit.KindAction && k != mapedit.KindCondition {
		return nil
	}
	sc, err := view.Script()
	if err != nil {
		return nil
	}
	n := scriptNodes(sc, k)[index]
	slot, err := strconv.Atoi(strings.TrimPrefix(key, "value"))
	if err != nil || slot < 0 || slot >= 10 {
		return nil
	}
	var out []ui.InspectionChoice
	add := func(v uint32, label string) { out = append(out, ui.InspectionChoice{Value: int64(v), Label: label}) }
	if slot == 0 && ((k == mapedit.KindAction && (n.Opcode == 3 || n.Opcode == 8)) || (k == mapedit.KindCondition && n.Opcode == 19)) {
		for v := uint32(0); v < 100; v++ {
			if int(v) >= len(sc.Conditions) || sc.Conditions[v].Opcode == 0x10002 {
				add(v, fmt.Sprintf("Variable register %d", v))
			}
		}
		return out
	}
	switch n.Type[slot] {
	case 2:
		seen := map[uint32]bool{}
		for _, u := range view.Units {
			if !seen[u.GroupID] {
				seen[u.GroupID] = true
				add(u.GroupID, fmt.Sprintf("Group %d", u.GroupID))
			}
		}
	case 3:
		for i, p := range view.Groups {
			add(uint32(i+1), fmt.Sprintf("Player %d: %s", i+1, alm.DecodeText([]byte(p.Name))))
		}
	case 4:
		counts := map[uint16]int{}
		for _, u := range view.Units {
			counts[u.UnitID]++
		}
		for _, u := range view.Units {
			if u.UnitID > 0 && counts[u.UnitID] == 1 {
				add(uint32(u.UnitID), fmt.Sprintf("Unit %d", u.UnitID))
			}
		}
	case 9:
		counts := map[uint16]int{}
		for _, u := range view.Objects {
			counts[u.Field12]++
		}
		for _, u := range view.Objects {
			if u.Field12 > 0 && counts[u.Field12] == 1 {
				add(uint32(u.Field12), fmt.Sprintf("Structure %d", u.Field12))
			}
		}
	}
	return out
}
