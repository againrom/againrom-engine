package mod

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/rules"
)

// writeSpells is the spells kind's writer: source made to hold d.Spells, the
// [global] table and each [[spell]] row kept where it stands with its lines
// but for the fields it changes. A row the source does not hold, its Line
// zero, is appended. The source is read with the settings d was read with,
// so a value a setting gives keeps its text while the edit leaves it.
func writeSpells(_ string, source []byte, d Data) ([]byte, error) {
	setting := settingOf(d.spellSettings)
	before, err := ParseSpells("", SpellsFile, source, setting)
	if err != nil {
		return nil, err
	}
	after := spellSourceRows(d.Spells)
	out, err := writeRows(source, spellSourceRows(before), after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseSpells("", SpellsFile, out, setting)
	if err != nil {
		return nil, err
	}
	if !sameRows(spellSourceRows(back), after) {
		return nil, fmt.Errorf("%s: the written rows do not read back as the rows given", SpellsFile)
	}
	return out, nil
}

// settingOf is the integer setting lookup over values.
func settingOf(values []SettingValue) func(string) (int64, bool) {
	return func(name string) (int64, bool) {
		for _, v := range values {
			if v.Key == name && v.Value.Kind == KindInt {
				return v.Value.Int, true
			}
		}
		return 0, false
	}
}

// spellSourceRows is d's [global] table and rows as the row writer sees
// them, in their file order and a new one after them: the global table
// while the file holds it or it sets a key, each row by its target, a
// repeat numbered.
func spellSourceRows(d SpellData) []sourceRow {
	type record struct {
		line int
		row  sourceRow
	}
	var out []record
	num := func(v int32) string { return strconv.Itoa(int(v)) }
	list := func(vals []int32) string {
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = num(v)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	tables := func(pairs []missionPair, set [5]SpellTable) []missionPair {
		for f := range set {
			if set[f].Set {
				pairs = append(pairs, missionPair{rules.SpellFormula(f).String(), list(set[f].Vals)})
			}
		}
		return pairs
	}
	g := d.Global
	if g.Line != 0 || len(g.Globals()) > 0 {
		var pairs []missionPair
		for _, r := range []struct {
			key string
			v   SpellRatio
		}{{"damage_mul", g.Damage}, {"heal_mul", g.Heal}, {"mana_mul", g.Mana},
			{"range_mul", g.Range}, {"radius_mul", g.Radius}, {"duration_mul", g.Duration}} {
			if r.v.Set {
				pairs = append(pairs, missionPair{r.key, fmt.Sprintf("[%d, %d]", r.v.Num, r.v.Den)})
			}
		}
		pairs = tables(pairs, g.Formulas())
		if g.BookSize.Set {
			pairs = append(pairs, missionPair{"book_size", num(g.BookSize.Val)})
		}
		out = append(out, record{g.Line, sourceRow{table: "global", identity: "global", plain: true, pairs: pairs}})
	}
	for _, r := range d.Rows {
		pairs := []missionPair{{"target", strconv.Quote(r.Target)}}
		ints := func(key string, v SpellInt) {
			if v.Set {
				pairs = append(pairs, missionPair{key, num(v.Val)})
			}
		}
		bools := func(key string, v SpellBool) {
			if v.Set {
				pairs = append(pairs, missionPair{key, strconv.FormatBool(v.Val)})
			}
		}
		ints("mana", r.Mana)
		ints("range", r.Range)
		ints("radius", r.Radius)
		ints("duration", r.Duration)
		ints("area_duration", r.AreaLife)
		if r.DamageMax.Set {
			pairs = append(pairs, missionPair{"damage", fmt.Sprintf("{ min = %d, max = %d }", r.DamageMin.Val, r.DamageMax.Val)})
		}
		if e := r.Effect; e != nil {
			var parts []string
			if e.Kind != "" {
				parts = append(parts, "kind = "+strconv.Quote(e.Kind))
			}
			if e.Mode != "" {
				parts = append(parts, "mode = "+strconv.Quote(e.Mode))
			}
			if e.Magnitude.Set {
				parts = append(parts, "magnitude = "+num(e.Magnitude.Val))
			}
			if e.Duration.Set {
				parts = append(parts, "duration = "+num(e.Duration.Val))
			}
			pairs = append(pairs, missionPair{"effect", "{ " + strings.Join(parts, ", ") + " }"})
		}
		bools("heal_hostile", r.HealHostile)
		bools("self_cast", r.SelfCast)
		if r.AreaHits.Set {
			pairs = append(pairs, missionPair{"area_hits", strconv.Quote(r.AreaHits.Val)})
		}
		ints("rays", r.Rays)
		pairs = tables(pairs, r.Formulas())
		bools(OffField, r.Off)
		ints("book_position", r.BookPosition)
		out = append(out, record{r.Line, sourceRow{table: "spell", identity: "spell " + rules.SpellKey(r.Target), pairs: pairs}})
	}
	slices.SortStableFunc(out, func(a, b record) int {
		if (a.line == 0) != (b.line == 0) {
			if a.line == 0 {
				return 1
			}
			return -1
		}
		return cmp.Compare(a.line, b.line)
	})
	rows, lines := make([]sourceRow, len(out)), make([]int, len(out))
	for i, r := range out {
		rows[i], lines[i] = r.row, r.line
	}
	numberRows(rows, lines)
	return rows
}
