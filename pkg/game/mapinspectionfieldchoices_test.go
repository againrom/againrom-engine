package game

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// referenceFieldChoices states each field's choices without the registry.
func referenceFieldChoices(s *mapInspectionEdits, kind string, index int, key string, script *alm.Script) []ui.InspectionChoice {
	if script == nil {
		return referenceRecordChoices(s, kind, index, key)
	}
	if strings.HasPrefix(key, "value") {
		k, _ := editKind(kind)
		return referenceScriptParameterChoices(s, *script, k, index, key)
	}
	return referenceScriptFieldChoices(kind, key, *script, s.inspector.inspectionUTF8)
}

var addDialogAsks = [][2]string{{"Enchantment", "castkind"}, {"Enchantment", "range"}, {"Enchantment", "spell"},
	{"Unit", "class"}, {"Structure", "kind"}, {"Sack", "item"}, {"Unit", "owner"}, {"Structure", "owner"}}

func checkFieldChoices(t *testing.T, name string, s *mapInspectionEdits, d *ui.InspectionDocument, faceRecords int) (asked int) {
	t.Helper()
	faces := 0
	for _, r := range d.Records {
		k, err := editKind(r.Kind)
		if err != nil {
			continue
		}
		fields, err := s.model.Fields(k, r.Index)
		if err != nil {
			continue
		}
		var script *alm.Script
		if scriptKind(k) {
			if view, e := s.model.Map(); e == nil {
				sc, _ := view.Script()
				script = &sc
			}
		}
		for _, f := range fields {
			if f.Key == "face" {
				if faces >= faceRecords {
					continue
				}
				faces++
				s.inspector.recordOptions, s.inspector.recordPictures = nil, nil
			}
			want := referenceFieldChoices(s, r.Kind, r.Index, f.Key, script)
			if f.Key == "face" {
				s.inspector.recordOptions, s.inspector.recordPictures = nil, nil
			}
			got := s.fieldChoices(fieldChoiceAsk{kind: r.Kind, index: r.Index, key: f.Key, script: script})
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s %d %s: choices %v, want %v", name, r.Kind, r.Index, f.Key, got, want)
			}
			asked++
		}
		if !scriptKind(k) {
			want := referenceRecordChoices(s, r.Kind, r.Index, "recipeitem")
			if got := s.RecordChoices(r.Kind, r.Index, "recipeitem"); !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s %d recipeitem: choices %v, want %v", name, r.Kind, r.Index, got, want)
			}
		}
	}
	for _, a := range addDialogAsks {
		want := referenceRecordChoices(s, a[0], -1, a[1])
		if got := s.RecordChoices(a[0], -1, a[1]); !reflect.DeepEqual(got, want) {
			t.Errorf("%s add %s %s: choices %v, want %v", name, a[0], a[1], got, want)
		}
		asked++
	}
	return asked
}

func TestFieldChoiceRegistryGivesTodaysLists(t *testing.T) {
	e, _ := openGoalMap(t, goalMap(t, true))
	s := objectEdits(t, e)
	rows := make(dbCollection, 31)
	for i := range rows {
		rows[i].name = fmt.Sprintf("Spell name %d", i)
	}
	s.inspector.table = &mapload.Table{Spells: rows}
	s.inspector.recordOptions = nil
	if asked := checkFieldChoices(t, "goal map", s, e.Document(), 4); asked < 40 {
		t.Fatalf("asked %d fields, want at least 40", asked)
	}
}

func referenceRecordChoices(s *mapInspectionEdits, kind string, index int, key string) []ui.InspectionChoice {
	if key == "npc" {
		m, err := s.model.Map()
		if err != nil || index < 0 || index >= len(m.AuthoredUnits) {
			return nil
		}
		out := []ui.InspectionChoice{{Value: -1, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.unit.npc.none"))}}
		for _, choice := range s.inspector.npcChoices(math.MaxUint16, false) {
			u := m.AuthoredUnits[index]
			u.Flags |= 1
			u.ClassSubID = uint16(choice.Value)
			r := mapload.Resolve(u, s.inspector.table)
			if r.Arm == mapload.ArmNPC && r.Found() {
				out = append(out, choice)
			}
		}
		return out
	}
	if kind == "Enchantment" && key == "castkind" {
		var out []ui.InspectionChoice
		for i := 0; i < 4; i++ {
			out = append(out, ui.InspectionChoice{Value: int64(i), Label: s.inspector.inspectionUTF8(s.inspector.word(castWords[fmt.Sprintf("mode%d", i)].Word))})
		}
		return out
	}
	if key == "owner" {
		m, err := s.model.Map()
		if err != nil {
			return nil
		}
		var out []ui.InspectionChoice
		for i, p := range m.Groups {
			out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("%d: %s", i+1, s.inspector.inspectionUTF8(alm.PlayerNameText.Decode([]byte(p.Name))))})
		}
		return out
	}
	if key == "face" {
		out := []ui.InspectionChoice{{Value: 0, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.unit.look.inherit"))}}
		m, err := s.model.Map()
		if err != nil || index < 0 || index >= len(m.Units) {
			return out
		}
		dir, _, ok := mapload.PlacedFigure(m.Units[index], s.inspector.table)
		if !ok {
			return out
		}
		cache := fmt.Sprintf("face:%s", dir)
		if s.inspector.recordOptions == nil {
			s.inspector.recordOptions = map[string][]ui.InspectionChoice{}
		}
		if choices, ok := s.inspector.recordOptions[cache]; ok {
			return append(out, choices...)
		}
		var choices []ui.InspectionChoice
		pictures := map[int64]*image.RGBA{}
		for face := int64(1); face <= 127; face++ {
			doll, _ := composeDoll(s.inspector.archives.artSource(), 0, data.Equipment{}, nil, dir, int(face), false)
			if doll.Figure != nil {
				choices = append(choices, ui.InspectionChoice{Value: face, Label: fmt.Sprintf("%d", face)})
				pictures[face] = doll.Figure
			}
		}
		s.inspector.recordOptions[cache] = choices
		if s.inspector.recordPictures == nil {
			s.inspector.recordPictures = map[string]map[int64]*image.RGBA{}
		}
		s.inspector.recordPictures[cache] = pictures
		return append(out, choices...)
	}
	if key == "sex" {
		return []ui.InspectionChoice{{Value: 0, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.unit.look.male"))}, {Value: 1, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.unit.look.female"))}}
	}
	if key == "tier" {
		m, err := s.model.Map()
		if err != nil || index < 0 || index >= len(m.AuthoredUnits) {
			return nil
		}
		return tierChoices(s.inspector.tiersOf(m.AuthoredUnits[index]))
	}
	if key == "recipeitem" {
		m, err := s.model.Map()
		if err != nil {
			return nil
		}
		loot, err := m.Loot()
		if err != nil || index < 0 || index >= len(loot.Records) {
			return nil
		}
		var out []ui.InspectionChoice
		for i, item := range loot.Records[index].Elements {
			name := mapload.ItemDisplayName(s.inspector.table, data.ItemCode(item.ItemCode()), "", fmt.Sprintf("item %#04x", item.ItemCode()))
			out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("Item %d: %s", i+1, name)})
		}
		return out
	}
	if strings.HasPrefix(key, "recipe") {
		m, err := s.model.Map()
		if err != nil {
			return nil
		}
		out := []ui.InspectionChoice{{Value: 0, Label: "No recipe (plain item)"}}
		for i, recipe := range m.Enchantments {
			if recipe.X == 0 && recipe.Y == 0 {
				out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("Recipe %d: spell%d, %d effects", i+1, uint16(recipe.SpellRaw), len(recipe.Elements))})
			}
		}
		return out
	}
	category := key
	if strings.HasPrefix(key, "item") || key == "additem" {
		category = "item"
	}
	if key == "addeffect" || strings.HasPrefix(key, "effect") {
		category = "effect"
	}
	rows := s.inspector.recordCatalog(category)
	if strings.HasPrefix(key, "item") && key != "item" {
		rows = append([]ui.InspectionChoice{{Value: 0, Label: "Remove this item"}}, rows...)
	}
	if category == "effect" && key != "addeffect" {
		rows = append([]ui.InspectionChoice{{Value: 0, Label: "Remove this effect"}}, rows...)
	}
	return rows
}

func referenceScriptFieldChoices(kind, key string, sc alm.Script, text func(string) string) []ui.InspectionChoice {
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
				out = append(out, ui.InspectionChoice{Value: int64(n.ID), Label: fmt.Sprintf("%d: %s", n.ID, text(n.Label))})
			}
		}
	}
	return out
}

func referenceScriptParameterChoices(s *mapInspectionEdits, sc alm.Script, k mapedit.Kind, index int, key string) []ui.InspectionChoice {
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
