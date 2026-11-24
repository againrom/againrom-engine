package game

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// setScriptWords is the node edit of a format whose row does not take whole
// script edits: it writes the label, the operation and the parameter words
// of a decoded node. A trigger takes the
// whole edit through the row's field kinds; any other script edit is refused
// with one line. The caller has worded the error.
func (s *mapInspectionEdits) setScriptWords(kind ui.RecordKind, index int, values []ui.InspectionFieldValue) (*ui.InspectionDocument, error) {
	row := s.row()
	k, err := editKind(kind)
	if err != nil || len(values) == 0 || !row.sets.has(kind) {
		return nil, s.refuse()
	}
	if k == mapedit.KindAction || k == mapedit.KindCondition {
		return s.setParameterWords(k, index, values)
	}
	return nil, s.refuse()
}

// setParameterWords writes the label, the operation and the parameter words
// of one decoded node. A new operation is a decoded row the game's table
// offers, and the slot types are its signature where a claim states one,
// else the stored ones; a slot the new types leave empty holds zero. The
// node is rewritten by the one node writer, then its values are set as
// their types hold them. A register address is held to the map's script
// profile, as the whole-script edit holds it.
func (s *mapInspectionEdits) setParameterWords(k mapedit.Kind, index int, values []ui.InspectionFieldValue) (*ui.InspectionDocument, error) {
	row := s.row()
	view, err := s.model.Map()
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
	profile := s.model.Profile()
	n, condition := nodes[index], k == mapedit.KindCondition
	if !mapload.ScriptOperationDecoded(profile.Ops, condition, n.Opcode) {
		return nil, fmt.Errorf(s.inspector.word(row.undecoded), n.Opcode)
	}
	refusal := errors.New(s.inspector.word(row.scriptRefusal))
	opcode, types, vals := n.Opcode, n.Type, n.Value
	var sets []mapedit.FieldSet
	var names []string
	var set [len(n.Type)]bool
	for _, v := range values {
		switch {
		case v.Key == "name" && v.Text:
			names = append(names, v.TextValue)
			continue
		case v.Text || v.New:
			return nil, refusal
		case v.Key == "opcode":
			if v.Value < 0 || v.Value > math.MaxUint32 {
				return nil, refusal
			}
			opcode = uint32(v.Value)
			continue
		}
		prefix := "value"
		if strings.HasPrefix(v.Key, "type") {
			prefix = "type"
		}
		slot, err := strconv.Atoi(strings.TrimPrefix(v.Key, prefix))
		if !strings.HasPrefix(v.Key, prefix) || err != nil || slot < 0 || slot >= len(n.Type) {
			return nil, refusal
		}
		if prefix == "type" {
			if v.Value < 0 || v.Value > mapload.ScriptStructure {
				return nil, refusal
			}
			types[slot] = uint32(v.Value)
			continue
		}
		vals[slot], set[slot] = uint32(v.Value), true
		sets = append(sets, mapedit.FieldSet{Key: v.Key, Value: v.Value})
	}
	retyped := opcode != n.Opcode || types != n.Type
	if retyped {
		if op, ok := scriptprofile.OpRow(profile.Ops, condition, opcode); !ok || !op.Offered || !op.Decoded {
			return nil, fmt.Errorf(s.inspector.word(row.undecoded), opcode)
		}
		command := uint32(0)
		if !condition && types[0] == mapload.ScriptConst {
			command = vals[0]
		}
		sig, stated := mapload.ScriptSignatureIn(profile.Ops, condition, opcode, command)
		if !stated && opcode != n.Opcode {
			// No claim states the slots of the operation: a node is not changed to it.
			return nil, fmt.Errorf(s.inspector.word(row.unstated), opcode)
		}
		for slot := range types {
			// A slot of a new type takes a value the edit names.
			if stated && types[slot] != sig[slot].Type || !stated && types[slot] != n.Type[slot] ||
				types[slot] != n.Type[slot] && types[slot] != mapload.ScriptNone && !set[slot] {
				return nil, refusal
			}
		}
		if condition && opcode != mapload.ScriptConstant {
			if err := registerCollides(sc, uint32(index)); err != nil {
				return nil, s.checkWords(err)
			}
		}
	}
	for slot, t := range types {
		switch {
		case set[slot] && t == mapload.ScriptNone:
			return nil, refusal
		case !set[slot] && !retyped, t == mapload.ScriptNone:
			continue
		}
		if err := scriptParamAllowed(view, profile, condition, opcode, slot, t, vals[slot]); err != nil {
			return nil, err
		}
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		mark := m.Mark()
		for _, name := range names {
			if _, err := setTextFieldOn(m, k, index, "name", name); err != nil {
				return false, err
			}
		}
		if retyped {
			var params []mapedit.ScriptParam
			for slot := range types {
				params = append(params, mapedit.ScriptParam{Type: types[slot], Value: n.Value[slot]})
				if types[slot] != n.Type[slot] || types[slot] == mapload.ScriptNone {
					params[slot].Value = 0
				}
			}
			if err := m.SetScriptNode(k, index, "", opcode, params...); err != nil {
				return false, err
			}
		}
		var todo []mapedit.FieldSet
		for _, set := range sets {
			if old, err := m.FieldValue(k, index, set.Key); err != nil {
				return false, err
			} else if old != set.Value {
				todo = append(todo, set)
			}
		}
		if len(todo) > 0 {
			if err := m.SetFields(k, index, todo...); err != nil {
				return false, err
			}
		}
		changed := m.Mark() != mark
		m.Squash(mark)
		return changed, nil
	})
}
