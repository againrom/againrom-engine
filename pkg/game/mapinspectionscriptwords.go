package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// setScriptWords is the node edit of a format whose row does not take whole
// script edits: it writes the label and the parameter words of a decoded
// node. A trigger takes the
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

// setParameterWords writes the parameter words of one decoded node; a
// register address is held to the map's script profile, as the whole-script
// edit holds it.
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
	var sets []mapedit.FieldSet
	var names []string
	for _, v := range values {
		if v.Key == "name" && v.Text {
			names = append(names, v.TextValue)
			continue
		}
		slot, err := strconv.Atoi(strings.TrimPrefix(v.Key, "value"))
		if !strings.HasPrefix(v.Key, "value") || err != nil || slot < 0 || slot >= len(n.Type) || n.Type[slot] == 0 || v.Text || v.New {
			return nil, errors.New(s.inspector.word(row.scriptRefusal))
		}
		if scriptAddressesVariable(condition, n.Opcode) && slot == 0 &&
			(v.Value < 0 || v.Value >= int64(profile.Registers)) {
			return nil, wordErr("editor.script.variableout", v.Value, profile.Registers-1)
		}
		if err := scriptCoordinateAllowed(view, profile.Ops, k, index, v.Key, v.Value); err != nil {
			return nil, err
		}
		sets = append(sets, mapedit.FieldSet{Key: v.Key, Value: v.Value})
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		named := false
		for _, name := range names {
			c, err := setTextFieldOn(m, k, index, "name", name)
			if err != nil {
				return false, err
			}
			named = named || c
		}
		var todo []mapedit.FieldSet
		for _, set := range sets {
			if old, err := m.FieldValue(k, index, set.Key); err != nil {
				return false, err
			} else if old != set.Value {
				todo = append(todo, set)
			}
		}
		if len(todo) == 0 {
			return named, nil
		}
		err := m.SetFields(k, index, todo...)
		return err == nil, err
	})
}
