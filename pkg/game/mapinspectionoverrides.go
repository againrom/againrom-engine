package game

import (
	"fmt"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// overrideKeys are the unit popup's override fields in window order: the
// record's override block, which the spawner stores after the constructor
// (UNIT-PLACE-034). Health and mana are absent at -1, every other at 0.
var overrideKeys = []string{"hp", "mana", "body", "mind", "spirit", "reaction", "defence", "absorption",
	"skill.blade", "skill.axe", "skill.bludgeon", "skill.pike", "skill.shooting",
	"protection.fire", "protection.water", "protection.air", "protection.earth", "protection.astral"}

// overrideGroup is the unit popup group key shows under; "" for a key that
// is no override.
func overrideGroup(x *MapInspector, key string) string {
	switch {
	case key == "hp" || key == "mana":
		return x.word("editor.unit.group.vitals")
	case key == "body" || key == "mind" || key == "spirit" || key == "reaction":
		return x.word("editor.unit.group.attributes")
	case key == "defence" || key == "absorption":
		return x.word("editor.unit.group.defence")
	case strings.HasPrefix(key, "skill."):
		return x.word("editor.unit.group.skills")
	case strings.HasPrefix(key, "protection."):
		return x.word("editor.unit.group.protections")
	}
	return ""
}

// overrideAbsent is the value that leaves key's template value standing.
func overrideAbsent(key string) int64 {
	if key == "hp" || key == "mana" {
		return -1
	}
	return 0
}

// overrideTemplate is the value the template actor v holds for key.
func overrideTemplate(v unitValues, key string) (int64, bool) {
	stats := map[string]int32{"hp": v.HP, "mana": v.Mana, "body": v.Body, "mind": v.Mind, "spirit": v.Spirit,
		"reaction": v.Reaction, "defence": v.Defence, "absorption": v.Absorption}
	for i, name := range []string{"blade", "axe", "bludgeon", "pike", "shooting"} {
		stats["skill."+name] = v.Skill[i+1]
	}
	for i, name := range []string{"fire", "water", "air", "earth", "astral"} {
		stats["protection."+name] = v.Protection[i]
	}
	n, ok := stats[key]
	return int64(n), ok
}

// overrideName is key's name in the install language.
func (x *MapInspector) overrideName(key string) string {
	switch key {
	case "hp":
		return x.word("editor.unit.override.hp")
	case "mana":
		return x.word("editor.unit.override.mana")
	case "body":
		return x.word("editor.unit.override.body")
	case "mind":
		return x.word("editor.unit.override.mind")
	case "spirit":
		return x.word("editor.unit.override.spirit")
	case "reaction":
		return x.word("editor.unit.override.reaction")
	case "defence":
		return x.word("editor.unit.override.defence")
	case "absorption":
		return x.word("editor.unit.override.absorption")
	case "skill.blade":
		return x.word("editor.unit.override.skill.blade")
	case "skill.axe":
		return x.word("editor.unit.override.skill.axe")
	case "skill.bludgeon":
		return x.word("editor.unit.override.skill.bludgeon")
	case "skill.pike":
		return x.word("editor.unit.override.skill.pike")
	case "skill.shooting":
		return x.word("editor.unit.override.skill.shooting")
	case "protection.fire":
		return x.word("editor.unit.override.protection.fire")
	case "protection.water":
		return x.word("editor.unit.override.protection.water")
	case "protection.air":
		return x.word("editor.unit.override.protection.air")
	case "protection.earth":
		return x.word("editor.unit.override.protection.earth")
	case "protection.astral":
		return x.word("editor.unit.override.protection.astral")
	case "gear":
		return x.word("editor.unit.override.gear")
	}
	return key
}

// hasTail reports that placement u stores anything after its constructor.
func hasTail(u alm.Unit) bool {
	return u.HasCurrentHP || u.Tail.SkipGear || u.Tail != (alm.UnitTail{Mana: u.Tail.Mana})
}

// gearOffered reports that the Human constructor reads u's flag bit 7: the
// type-key and definition-id arms (ALM-FLAGPATH-109).
func gearOffered(u alm.Unit, t *mapload.Table) bool {
	if t == nil {
		return false
	}
	r := mapload.Resolve(u, t)
	return r.Found() && (r.Arm == mapload.ArmHumansByType || r.Arm == mapload.ArmServerID)
}

// templateSheets are m's placements as their templates spawn them: each
// placement without its override block, at Normal difficulty.
func templateSheets(m *alm.Map, t *mapload.Table) ([]unitSheet, error) {
	bare := *m
	bare.Units = make([]alm.Unit, len(m.Units))
	for i, u := range m.Units {
		bare.Units[i] = u.WithoutTail()
	}
	return unitSheets(&bare, t)
}

// unitTemplateView is placement index's view for the popup's fields; its
// in-game sheet is built on the first field that needs it.
type unitTemplateView struct {
	actualBuilt, actualOK bool
	actual                unitValues
	s                     *mapInspectionEdits
	m                     *alm.Map
	model                 *mapedit.Editor
	index                 int
}

// unitTemplate is the field view of record index of kind k; nil for a
// record that is no unit.
func (s *mapInspectionEdits) unitTemplate(k mapedit.Kind, index int, models ...*mapedit.Editor) *unitTemplateView {
	if k != mapedit.KindUnit {
		return nil
	}
	model := s.model
	if len(models) > 0 {
		model = models[0]
	}
	m, err := model.Map()
	if err != nil || index < 0 || index >= len(m.Units) {
		return nil
	}
	return &unitTemplateView{s: s, m: m, model: model, index: index}
}

func (u *unitTemplateView) stock() (int, error) {
	if u == nil || u.model == nil || u.s == nil || u.m == nil || u.index < 0 || u.index >= len(u.m.AuthoredUnits) {
		return -1, fmt.Errorf("native Human placement is unavailable")
	}
	t := u.s.inspector.table
	r := mapload.Resolve(u.m.AuthoredUnits[u.index], t)
	if !t.ClassPlacementFields() || !r.Found() || r.Arm == mapload.ArmUnits {
		return -1, fmt.Errorf("item rows require a native Human placement")
	}
	return u.model.UnitStock(u.index)
}

// label is the unit popup's label for field f. offered is false for a field
// the placement does not take.
func (u *unitTemplateView) label(f mapedit.Field) (label string, offered bool) {
	if u == nil {
		return "", true
	}
	x := u.s.inspector
	switch f.Key {
	case "owner":
		return x.word("editor.unit.field.owner"), true
	case "class":
		return x.word("editor.unit.field.class"), true
	case "definition":
		return x.word("editor.unit.field.definition"), true
	case "group":
		return x.word("editor.unit.field.group"), true
	case "face":
		return x.word("editor.unit.field.face"), x.table.ClassPlacementFields() && gearOffered(u.m.Units[u.index], x.table)
	case "sex":
		return x.word("editor.unit.field.sex"), x.table.ClassPlacementFields() && gearOffered(u.m.Units[u.index], x.table)
	case "npcflag":
		return "", false
	case "npc":
		return x.word("editor.unit.field.npc"), x.table.ClassPlacementFields() && len(x.table.NPCCatalog.Records()) > 0 && mapload.Resolve(u.m.Units[u.index], x.table).Arm != mapload.ArmUnits
	}
	if f.Check == mapedit.CheckUnitGear {
		return x.overrideName(f.Key), gearOffered(u.m.Units[u.index], x.table)
	}
	if _, ok := overrideTemplate(unitValues{}, f.Key); !ok {
		return "", true
	}
	return x.overrideName(f.Key), true
}

// gearError refuses flag bit 7 on a placement whose constructor does not
// read it.
func (x *MapInspector) gearError(u alm.Unit) error {
	if gearOffered(u, x.table) {
		return nil
	}
	return fmt.Errorf("bit 7 reaches only a Human of the type-key or definition-id arm; this placement takes the %s arm", mapload.Resolve(u, x.table).Arm)
}

// overrideLines are the unit popup's map-override section for placement i
// of model: each present override beside its template value, then how a
// field clears. template is the placement's template sheet.
func (x *MapInspector) overrideLines(model *mapedit.Editor, i int, template unitSheet) []ui.InspectionLine {
	enc := x.inspectionUTF8
	out := []ui.InspectionLine{{Text: enc(x.word("editor.unit.overrides")), Role: "heading"}}
	var set []string
	for _, key := range overrideKeys {
		v, err := model.FieldValue(mapedit.KindUnit, i, key)
		if err != nil || v == overrideAbsent(key) {
			continue
		}
		was, _ := overrideTemplate(template.Values, key)
		set = append(set, fmt.Sprintf(x.word("editor.unit.override.item"), x.overrideName(key), v, was))
	}
	if len(set) == 0 {
		set = append(set, x.word("editor.unit.override.none"))
	}
	out = append(out, ui.InspectionLine{Text: enc(strings.Join(set, ", "))})
	if v, err := model.FieldValue(mapedit.KindUnit, i, "gear"); err == nil && v != 0 {
		out = append(out, ui.InspectionLine{Text: enc(x.word("editor.unit.override.skipped"))})
	}
	out = append(out, ui.InspectionLine{Text: enc(x.word("editor.unit.override.note"))})
	return out
}
