package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (x *MapInspector) scriptTargets(d *ui.InspectionDocument, m *alm.Map, sc alm.Script, pages map[string][]byte) map[uint32][]ui.InspectionScriptTarget {
	out := map[uint32][]ui.InspectionScriptTarget{}
	add := func(kind, value uint32, record int, name, choice string, members int) {
		out[kind] = append(out[kind], ui.InspectionScriptTarget{Value: value, Record: record, Name: name, Choice: choice, Members: members})
	}
	counts := map[string]map[uint32]int{"Unit": {}, "Structure": {}}
	for _, u := range m.Units {
		counts["Unit"][uint32(u.UnitID)]++
	}
	for _, o := range m.Objects {
		counts["Structure"][uint32(o.Field12)]++
	}
	groups := map[uint32]int{}
	var order []uint32
	for _, u := range m.Units {
		if _, ok := groups[u.GroupID]; !ok {
			order = append(order, u.GroupID)
		}
		groups[u.GroupID]++
	}
	for _, g := range order {
		add(mapload.ScriptGroup, g, -1, fmt.Sprintf(x.inspectionUTF8(x.word("editor.param.groupchoice")), g, groups[g]), "", groups[g])
	}
	for i, r := range d.Records {
		switch r.Kind {
		case "Unit":
			id := uint32(m.Units[r.Index].UnitID)
			if id > 0 && id < 10001 && counts[r.Kind][id] == 1 {
				add(mapload.ScriptUnit, id, i, r.Label, fmt.Sprintf(x.inspectionUTF8(x.word("editor.param.unitchoice")), id, r.Label), 0)
			}
		case "Structure":
			id := uint32(m.Objects[r.Index].Field12)
			if id > 0 && counts[r.Kind][id] == 1 {
				add(mapload.ScriptStructure, id, i, r.Label, fmt.Sprintf(x.inspectionUTF8(x.word("editor.param.structurechoice")), id, r.Label), 0)
			}
		case "Player":
			add(mapload.ScriptPlayer, uint32(r.Index+1), i, r.Label, fmt.Sprintf(x.inspectionUTF8(x.word("editor.param.playerchoice")), r.Label), 0)
		}
	}
	for id := uint32(10001); id <= 11000; id++ {
		add(mapload.ScriptUnit, id, -1, fmt.Sprintf(x.inspectionUTF8(x.word("editor.param.hero")), id-10000), "", 0)
	}
	seen := map[uint32]bool{}
	for _, nodes := range [][]alm.ScriptNode{sc.Actions, sc.Conditions} {
		for _, n := range nodes {
			for p, tag := range n.Type {
				v := n.Value[p]
				if tag == mapload.ScriptUnit && v > 11000 && !seen[v] {
					seen[v] = true
					add(tag, v, -1, fmt.Sprintf(x.inspectionUTF8(x.word("editor.param.hero")), v-10000), "", 0)
				}
			}
		}
	}
	for _, item := range x.recordCatalog("item") {
		value := uint32(uint16(item.Value - 0xe18))
		name := strings.TrimPrefix(item.Label, fmt.Sprintf("%d: ", item.Value))
		add(mapload.ScriptItem, value, -1, name, fmt.Sprintf("%d: %s", value, name), 0)
	}
	for _, spell := range x.recordCatalog("spell") {
		value := uint32(spell.Value)
		if value < 1 || value > 28 {
			continue
		}
		add(ui.ScriptSpellParameter, value, -1, spell.Label, spell.Label, 0)
		if value != 26 {
			add(ui.ScriptEntrySpellParameter, value, -1, spell.Label, spell.Label, 0)
		}
		switch value {
		case 3, 7, 8, 12, 17, 19:
			add(ui.ScriptAreaSpellParameter, value, -1, spell.Label, spell.Label, 0)
		}
	}
	registers := map[uint32]bool{}
	for _, nodes := range [][]ui.InspectionScriptNode{d.Script.Actions, d.Script.Conditions} {
		for _, n := range nodes {
			if n.AddressesVariable {
				registers[n.VariableRegister] = true
			}
		}
	}
	var vars []uint32
	for r := range registers {
		vars = append(vars, r)
	}
	slices.Sort(vars)
	for i, r := range vars {
		name := x.inspectionUTF8(fmt.Sprintf(x.word("editor.scripts.variable"), r))
		if uint64(r) < uint64(len(sc.Conditions)) && sc.Conditions[r].Opcode == 0x10002 && sc.Conditions[r].Label != "" {
			name = x.inspectionUTF8(sc.Conditions[r].Label)
		}
		add(ui.ScriptVariableParameter, r, -2-i, name, "", 0)
	}
	for r := uint32(0); r < 100; r++ {
		if !registers[r] && uint64(r) >= uint64(len(sc.Conditions)) {
			add(ui.ScriptVariableParameter, r, -1, x.inspectionUTF8(x.word("editor.param.newvariable")), "", 0)
			break
		}
	}
	x.messageTargets(out, pages)
	return out
}

func (s *mapInspectionEdits) scriptParameterChoices(sc alm.Script, k mapedit.Kind, index int, key string) []ui.InspectionChoice {
	if k != mapedit.KindAction && k != mapedit.KindCondition {
		return nil
	}
	slot, err := strconv.Atoi(strings.TrimPrefix(key, "value"))
	if err != nil || slot < 0 || slot >= 10 {
		return nil
	}
	n := scriptNodes(sc, k)[index]
	var out []ui.InspectionChoice
	kind := scriptParameterKind(k == mapedit.KindCondition, n.Opcode, n.Type, slot)
	current, found := int64(int32(n.Value[slot])), false
	for _, target := range s.scriptTargets[kind] {
		out = append(out, ui.InspectionChoice{Value: target.ChoiceValue(), Label: target.ChoiceText()})
		found = found || target.ChoiceValue() == current
	}
	if kind == ui.ScriptMessageParameter && !found {
		out = append(out, ui.InspectionChoice{Value: current, Label: strconv.FormatInt(current, 10)})
	}
	return out
}
