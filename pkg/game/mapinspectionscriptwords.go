package game

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// inspectNodeTexts gives each node of the script model the text fields its
// record's field table states beside its name, in UTF-8, numbered from 1.
func (x *MapInspector) inspectNodeTexts(sc *ui.InspectionScriptModel, model *mapedit.Editor) {
	for _, family := range []struct {
		k     mapedit.Kind
		nodes []ui.InspectionScriptNode
	}{{mapedit.KindAction, sc.Actions}, {mapedit.KindCondition, sc.Conditions}} {
		k, nodes := family.k, family.nodes
		for i := range nodes {
			fields, err := model.Fields(k, i)
			if err != nil {
				continue
			}
			rec, err := model.Record(k, i)
			if err != nil {
				continue
			}
			for _, f := range fields {
				if f.Text == nil || f.Key == "name" || f.Offset+f.Width > len(rec) {
					continue
				}
				v := f.Text.Decode(rec[f.Offset : f.Offset+f.Width])
				label := fmt.Sprintf(x.word(f.Label), len(nodes[i].Texts)+1)
				nodes[i].Texts = append(nodes[i].Texts, ui.InspectionField{Key: f.Key, Label: label, Text: true, TextValue: v})
			}
		}
	}
}

// setScriptWords is the node edit of a format whose row does not take whole
// script edits: it writes the label and texts of a node and the operation
// and parameter words of a decoded one. A trigger takes the whole edit
// through the row's field kinds; any other script edit is refused with one
// line. The caller has worded the error.
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
// of one decoded node, and the label and text fields of any node. A new
// operation is a decoded row the game's table offers, and the slot types are its signature where a claim states one,
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
	// A node's name and text fields are its record's own, whatever its
	// operation (R2-ENGINE-1681); its words need a decoded operation.
	texts := !slices.ContainsFunc(values, func(v ui.InspectionFieldValue) bool { return !v.Text || !scriptprofile.NodeTextKey(v.Key) })
	if !texts && !mapload.ScriptOperationDecoded(profile.Ops, condition, n.Opcode) {
		return nil, fmt.Errorf(s.inspector.word(row.undecoded), n.Opcode)
	}
	refusal := errors.New(s.inspector.word(row.scriptRefusal))
	opcode, types, vals := n.Opcode, n.Type, n.Value
	var sets []mapedit.FieldSet
	var named []ui.InspectionFieldValue
	var set [len(n.Type)]bool
	for _, v := range values {
		switch {
		case v.Text && scriptprofile.NodeTextKey(v.Key):
			if _, err := s.model.TextFieldValue(k, index, v.Key); err != nil {
				return nil, refusal
			}
			named = append(named, v)
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
			if err := registerCollides(sc, profile, uint32(index)); err != nil {
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
		for _, v := range named {
			if _, err := setTextFieldOn(m, k, index, v.Key, v.TextValue); err != nil {
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
