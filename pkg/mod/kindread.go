package mod

import (
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/rules"
)

// The readers of the declared kinds. Each parses one file of one mod,
// refusing a value at its file and line, and adds the records to the set's
// data, refusing what a kept kind refuses across mods.

func readItems(r DataReader, d *Data, rel string, src []byte) error {
	look, err := TextLookup(r.Dir, r.Lang)
	if err != nil {
		return err
	}
	items, err := ParseItems(r.Mod, rel, src, look, r.Lang)
	if err != nil {
		return err
	}
	for i := range items.Rows {
		items.Rows[i].Dir = r.Dir
	}
	for i := range items.Materials {
		items.Materials[i].Dir = r.Dir
	}
	for i := range items.Weapons {
		items.Weapons[i].Dir = r.Dir
	}
	if !slices.Contains(d.Items.Mods, r.Mod) {
		d.Items.Mods = append(d.Items.Mods, r.Mod)
	}
	d.Items.Rows = append(d.Items.Rows, items.Rows...)
	d.Items.Changes = append(d.Items.Changes, items.Changes...)
	d.Items.Materials = append(d.Items.Materials, items.Materials...)
	d.Items.Weapons = append(d.Items.Weapons, items.Weapons...)
	return nil
}

func readCharacters(r DataReader, d *Data, rel string, src []byte) error {
	look, err := TextLookup(r.Dir, r.Lang)
	if err != nil {
		return err
	}
	chars, err := ParseCharacters(r.Mod, rel, src, look, r.Lang)
	if err != nil {
		return err
	}
	d.Characters.Rows = append(d.Characters.Rows, chars.Rows...)
	return nil
}

func readScreens(r DataReader, d *Data, rel string, src []byte) error {
	look, err := TextLookup(r.Dir, r.Lang)
	if err != nil {
		return err
	}
	screens, err := ParseScreens(r.Mod, rel, src, look, r.Lang)
	if err != nil {
		return err
	}
	d.Screens.Screens = append(d.Screens.Screens, screens.Screens...)
	return d.Screens.CheckSlots()
}

func readCompanions(r DataReader, d *Data, rel string, src []byte) error {
	joins, err := ParseCompanions(r.Mod, rel, src)
	if err != nil {
		return err
	}
	d.Companions.Joins = append(d.Companions.Joins, joins.Joins...)
	return nil
}

func readWeaponBodies(r DataReader, d *Data, rel string, src []byte) error {
	weapons, err := ParseWeaponBodies(r.Mod, rel, src)
	if err != nil {
		return err
	}
	d.Bodies.Weapons = append(d.Bodies.Weapons, weapons...)
	return nil
}

func readBodies(r DataReader, d *Data, rel string, src []byte) error {
	bodies, err := ParseBodies(r.Mod, rel, src)
	if err != nil {
		return err
	}
	for i := range bodies {
		bodies[i].Dir = r.Dir
	}
	d.Bodies.Bodies = append(d.Bodies.Bodies, bodies...)
	return nil
}

func readMissions(r DataReader, d *Data, rel string, src []byte) error {
	look, err := TextLookup(r.Dir, r.Lang)
	if err != nil {
		return err
	}
	missions, err := ParseMissions(r.Mod, rel, src, look, r.Lang)
	if err != nil {
		return err
	}
	for _, m := range missions.Missions {
		if !m.Off {
			if err := m.ReadFiles(r.Dir); err != nil {
				return err
			}
		}
		d.Missions.Missions = append(d.Missions.Missions, m)
	}
	return nil
}

// readTowns reads the mod's layer over the town catalog. A description
// that names a file of the mod is read here, so a missing or unreadable one
// refuses the launch at its line; the catalog decoder judges it when the
// layer is applied.
func readTowns(r DataReader, d *Data, rel string, src []byte) error {
	records, rows, err := ParseTowns(r.Mod, rel, src)
	if err != nil {
		return err
	}
	d.Towns.Tavern = append(d.Towns.Tavern, rows...)
	for i := range records {
		t := &records[i]
		t.Dir = r.Dir
		if !slices.ContainsFunc(t.Fields, func(f PatchField) bool { return f.Name == "description" }) ||
			!IsTownDescriptionFile(t.Entry.Description) {
			continue
		}
		data, path, err := ReadInside(r.Dir, t.Entry.Description)
		if err != nil {
			return &ItemFileError{File: rel, Line: t.LineOf("description"), Msg: fmt.Sprintf("description %q: %v", t.Entry.Description, err)}
		}
		t.Description = &TownDescriptionFile{Path: path, Data: data, Dir: r.Dir}
		t.Entry.Description = ModDescriptionName(r.Mod, path)
	}
	d.Towns.Records = append(d.Towns.Records, records...)
	return nil
}

// readSettings validates a settings file; the settings themselves are read
// by LoadSettings before the mod's script runs.
func readSettings(_ DataReader, _ *Data, rel string, src []byte) error {
	if _, err := ParseSettings(src); err != nil {
		return fmt.Errorf("%s: %w", rel, err)
	}
	return nil
}

func readSpells(r DataReader, d *Data, rel string, src []byte) error {
	setting := func(name string) (int64, bool) {
		for _, v := range r.Settings {
			if v.Key == name && v.Value.Kind == KindInt {
				return v.Value.Int, true
			}
		}
		return 0, false
	}
	spells, err := ParseSpells(r.Mod, rel, src, setting)
	if err != nil {
		return err
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: rel, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if d.spellGlobals == nil {
		d.spellGlobals, d.spellFields = map[string]string{}, map[string]string{}
	}
	for _, k := range spells.Global.Globals() {
		if first, dup := d.spellGlobals[k.Key]; dup {
			return bad(k.Line, "%s is already set by mod %q", k.Key, first)
		}
		d.spellGlobals[k.Key] = r.Mod
	}
	for _, row := range spells.Rows {
		for _, f := range spellFieldsOf(row) {
			key := strings.ReplaceAll(row.Target, "_", " ") + " " + f.name
			if first, dup := d.spellFields[key]; dup {
				return bad(f.line, "%s of %q is already set by mod %q", f.name, row.Target, first)
			}
			d.spellFields[key] = r.Mod
		}
	}
	g := &d.Spells.Global
	merge := func(dst *SpellRatio, src SpellRatio) {
		if src.Set {
			*dst = src
		}
	}
	merge(&g.Damage, spells.Global.Damage)
	merge(&g.Heal, spells.Global.Heal)
	merge(&g.Mana, spells.Global.Mana)
	merge(&g.Range, spells.Global.Range)
	merge(&g.Radius, spells.Global.Radius)
	merge(&g.Duration, spells.Global.Duration)
	mergeTable := func(dst *SpellTable, src SpellTable) {
		if src.Set {
			*dst = src
		}
	}
	mergeTable(&g.Power, spells.Global.Power)
	mergeTable(&g.DamageFactor, spells.Global.DamageFactor)
	mergeTable(&g.RangeBonus, spells.Global.RangeBonus)
	mergeTable(&g.DurationFactor, spells.Global.DurationFactor)
	d.Spells.Rows = append(d.Spells.Rows, spells.Rows...)
	return nil
}

type spellField struct {
	name string
	line int
}

func spellFieldsOf(r SpellRow) []spellField {
	var out []spellField
	add := func(name string, v SpellInt) {
		if v.Set {
			out = append(out, spellField{name, v.Line})
		}
	}
	add("mana", r.Mana)
	add("range", r.Range)
	add("radius", r.Radius)
	add("rays", r.Rays)
	add("duration", r.Duration)
	add("area_duration", r.AreaLife)
	add("damage", r.DamageMax)
	for f, t := range r.Formulas() {
		if t.Set {
			out = append(out, spellField{rules.SpellFormula(f).String(), t.Line})
		}
	}
	if r.HealHostile.Set {
		out = append(out, spellField{"heal_hostile", r.HealHostile.Line})
	}
	if r.SelfCast.Set {
		out = append(out, spellField{"self_cast", r.SelfCast.Line})
	}
	if r.AreaHits.Set {
		out = append(out, spellField{"area_hits", r.AreaHits.Line})
	}
	if e := r.Effect; e != nil {
		if e.Kind != "" {
			out = append(out, spellField{"effect.kind", e.Line})
		}
		if e.Mode != "" {
			out = append(out, spellField{"effect.mode", e.Line})
		}
		add("effect.magnitude", SpellInt{Set: e.Magnitude.Set, Line: e.Line})
		add("effect.duration", SpellInt{Set: e.Duration.Set, Line: e.Line})
	}
	return out
}
