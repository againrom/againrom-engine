package modrt

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.starlark.net/starlark"

	"againrom/pkg/mod"
	"againrom/pkg/rules"
)

// dataFiles are the data files game.data.add reads, each with the function
// that turns its bytes into game data.
var dataFiles = map[string]func(d *dataValue, rel string, src []byte) error{
	mod.BodiesFile:       (*dataValue).loadBodies,
	mod.WeaponBodiesFile: (*dataValue).loadWeaponBodies,
	mod.CharactersFile:   (*dataValue).loadCharacters,
	mod.CompanionsFile:   (*dataValue).loadCompanions,
	mod.ItemsFile:        (*dataValue).loadItems,
	mod.ScreensFile:      (*dataValue).loadScreens,
	mod.SpellsFile:       (*dataValue).loadSpells,
}

func dataFileNames() string {
	names := make([]string, 0, len(dataFiles))
	for n := range dataFiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// dataValue is game.data: the data files a mod may load.
type dataValue struct {
	sh       *shared
	id       string
	dir      string
	loader   *loader
	loaded   map[string]bool
	frozen   bool
	settings []mod.SettingValue
}

func (d *dataValue) String() string        { return "<game.data>" }
func (d *dataValue) Type() string          { return "game.data" }
func (d *dataValue) Freeze()               { d.frozen = true }
func (d *dataValue) Truth() starlark.Bool  { return true }
func (d *dataValue) Hash() (uint32, error) { return 0, errors.New("unhashable type: game.data") }
func (d *dataValue) AttrNames() []string   { return []string{"add"} }
func (d *dataValue) Attr(name string) (starlark.Value, error) {
	if name == "add" {
		return starlark.NewBuiltin("game.data.add", d.add), nil
	}
	return nil, nil
}

// add is game.data.add(path): it reads a data file of the mod folder and adds
// what it describes to the game. A file is loaded once, during init.
func (d *dataValue) add(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "path", &path); err != nil {
		return nil, err
	}
	if d.frozen {
		return nil, errors.New("game.data.add: data is loaded during init only")
	}
	src, rel, err := d.loader.read(path)
	if err != nil {
		return nil, fmt.Errorf("game.data.add(%q): %w", path, err)
	}
	read, ok := dataFiles[rel]
	if !ok {
		return nil, fmt.Errorf("game.data.add(%q): no data file of that name is loaded (loadable: %s)", path, dataFileNames())
	}
	if d.loaded[rel] {
		return nil, fmt.Errorf("game.data.add(%q): the file is already loaded", path)
	}
	d.loaded[rel] = true
	if err := read(d, rel, src); err != nil {
		return nil, err
	}
	return starlark.None, nil
}

func (d *dataValue) loadItems(rel string, src []byte) error {
	look, err := mod.TextLookup(d.dir, d.sh.lang)
	if err != nil {
		return err
	}
	items, err := mod.ParseItems(d.id, rel, src, look, d.sh.lang)
	if err != nil {
		return err
	}
	for i := range items.Rows {
		items.Rows[i].Dir = d.dir
	}
	d.sh.items.Rows = append(d.sh.items.Rows, items.Rows...)
	d.sh.items.Changes = append(d.sh.items.Changes, items.Changes...)
	return nil
}

func (d *dataValue) loadWeaponBodies(rel string, src []byte) error {
	weapons, err := mod.ParseWeaponBodies(d.id, rel, src)
	if err != nil {
		return err
	}
	d.sh.bodies.Weapons = append(d.sh.bodies.Weapons, weapons...)
	return nil
}

func (d *dataValue) loadBodies(rel string, src []byte) error {
	bodies, err := mod.ParseBodies(d.id, rel, src)
	if err != nil {
		return err
	}
	for _, b := range bodies {
		for _, first := range d.sh.bodies.Bodies {
			if first.Name == b.Name {
				return &mod.ItemFileError{File: rel, Line: b.Line, Msg: fmt.Sprintf(
					"body %q is already supplied by mod %q", b.Name, first.Mod)}
			}
		}
		b.Dir = d.dir
		d.sh.bodies.Bodies = append(d.sh.bodies.Bodies, b)
	}
	return nil
}

func (d *dataValue) loadScreens(rel string, src []byte) error {
	look, err := mod.TextLookup(d.dir, d.sh.lang)
	if err != nil {
		return err
	}
	screens, err := mod.ParseScreens(d.id, rel, src, look, d.sh.lang)
	if err != nil {
		return err
	}
	d.sh.screens.Screens = append(d.sh.screens.Screens, screens.Screens...)
	return d.sh.screens.CheckSlots()
}

func (d *dataValue) loadCompanions(rel string, src []byte) error {
	joins, err := mod.ParseCompanions(d.id, rel, src)
	if err != nil {
		return err
	}
	for _, j := range joins.Joins {
		if first, dup := d.sh.companions.Find(j.Chapter, j.Companion); dup {
			return &mod.ItemFileError{File: rel, Line: j.Line, Msg: fmt.Sprintf(
				"companion %d of chapter %d already has a join condition from mod %q", j.Companion, j.Chapter, first.Mod)}
		}
		d.sh.companions.Joins = append(d.sh.companions.Joins, j)
	}
	return nil
}

func (d *dataValue) loadSpells(rel string, src []byte) error {
	setting := func(name string) (int64, bool) {
		for _, v := range d.settings {
			if v.Key == name && v.Value.Kind == mod.KindInt {
				return v.Value.Int, true
			}
		}
		return 0, false
	}
	spells, err := mod.ParseSpells(d.id, rel, src, setting)
	if err != nil {
		return err
	}
	bad := func(line int, format string, args ...any) error {
		return &mod.ItemFileError{File: rel, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	for _, k := range spells.Global.Globals() {
		if first, dup := d.sh.spellGlobals[k.Key]; dup {
			return bad(k.Line, "%s is already set by mod %q", k.Key, first)
		}
		d.sh.spellGlobals[k.Key] = d.id
	}
	for _, r := range spells.Rows {
		for _, f := range spellFieldsOf(r) {
			key := strings.ReplaceAll(r.Target, "_", " ") + " " + f.name
			if first, dup := d.sh.spellFields[key]; dup {
				return bad(f.line, "%s of %q is already set by mod %q", f.name, r.Target, first)
			}
			d.sh.spellFields[key] = d.id
		}
	}
	g := &d.sh.spells.Global
	merge := func(dst *mod.SpellRatio, src mod.SpellRatio) {
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
	mergeTable := func(dst *mod.SpellTable, src mod.SpellTable) {
		if src.Set {
			*dst = src
		}
	}
	mergeTable(&g.Power, spells.Global.Power)
	mergeTable(&g.DamageFactor, spells.Global.DamageFactor)
	mergeTable(&g.RangeBonus, spells.Global.RangeBonus)
	mergeTable(&g.DurationFactor, spells.Global.DurationFactor)
	d.sh.spells.Rows = append(d.sh.spells.Rows, spells.Rows...)
	return nil
}

type spellField struct {
	name string
	line int
}

func spellFieldsOf(r mod.SpellRow) []spellField {
	var out []spellField
	add := func(name string, v mod.SpellInt) {
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
		add("effect.magnitude", mod.SpellInt{Set: e.Magnitude.Set, Line: e.Line})
		add("effect.duration", mod.SpellInt{Set: e.Duration.Set, Line: e.Line})
	}
	return out
}
