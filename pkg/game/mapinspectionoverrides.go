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

// unitTemplateView is placement index's view for the popup's field labels;
// its template sheet is built on the first label that needs it.
type unitTemplateView struct {
	actualBuilt, actualOK bool
	actual                unitValues
	s                     *mapInspectionEdits
	m                     *alm.Map
	index                 int
	built                 bool
	v                     unitValues
	ok                    bool
}

// unitTemplate is the label view of record index of kind k; nil for a
// record that is no unit.
func (s *mapInspectionEdits) unitTemplate(k mapedit.Kind, index int) *unitTemplateView {
	if k != mapedit.KindUnit {
		return nil
	}
	m, err := s.model.Map()
	if err != nil || index < 0 || index >= len(m.Units) {
		return nil
	}
	return &unitTemplateView{s: s, m: m, index: index}
}

// label is the unit popup's label for field f: an override field's name
// with its template value in brackets. offered is false for a field the
// placement does not take.
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
	}
	if f.Check == mapedit.CheckUnitGear {
		return x.overrideName(f.Key), gearOffered(u.m.Units[u.index], x.table)
	}
	if _, ok := overrideTemplate(unitValues{}, f.Key); !ok || x.table == nil {
		return "", true
	}
	if !u.built {
		u.built = true
		if sheets := u.s.unitViewSheets(true); u.index < len(sheets) {
			u.v, u.ok = sheets[u.index].Values, true
		}
	}
	if !u.ok {
		return x.overrideName(f.Key), true
	}
	v, _ := overrideTemplate(u.v, f.Key)
	return fmt.Sprintf("%s (%d)", x.overrideName(f.Key), v), true
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
