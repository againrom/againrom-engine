package mod

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// writeItems is the items kind's writer: source made to hold d.Items, each
// record kept where it stands with its lines but for the fields it changes.
// A record the source does not hold, its Line zero, is appended.
func writeItems(_ string, source []byte, d Data) ([]byte, error) {
	lenient := func(string) (string, bool) { return "-", true }
	before, err := ParseItems("", ItemsFile, source, lenient, "")
	if err != nil {
		return nil, err
	}
	after := itemSourceRows(d.Items)
	out, err := writeRows(source, itemSourceRows(before), after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseItems("", ItemsFile, out, lenient, "")
	if err != nil {
		return nil, err
	}
	same := func(a, b sourceRow) bool {
		pairs := func(r sourceRow) map[string]string {
			m := map[string]string{}
			for _, p := range r.pairs {
				m[p.key] = p.value
			}
			return m
		}
		return a.table == b.table && a.identity == b.identity && maps.Equal(pairs(a), pairs(b))
	}
	if !slices.EqualFunc(itemSourceRows(back), after, same) {
		return nil, fmt.Errorf("%s: the written records do not read back as the records given", ItemsFile)
	}
	return out, nil
}

// itemSourceRows is d's records as the row writer sees them, in their file
// order and a new record after them: an added row by its key, a change by
// the row it names and a material by its pair, a repeat numbered.
func itemSourceRows(d ItemData) []sourceRow {
	type record struct {
		line int
		row  sourceRow
	}
	var out []record
	str := strconv.Quote
	num := func(v int32) string { return strconv.Itoa(int(v)) }
	opt := func(pairs []missionPair, key, value string) []missionPair {
		if value != "" {
			pairs = append(pairs, missionPair{key, str(value)})
		}
		return pairs
	}
	for _, r := range d.Rows {
		pairs := []missionPair{{"key", str(r.Key)}, {"name", str(r.NameKey)}, {"slot", str(r.Slot)},
			{"defence", num(r.Defence)}, {"absorption", num(r.Absorption)}, {"weight", num(r.Weight)}, {"price", num(r.Price)},
			{"stand-in", str(r.StandIn)}}
		switch r.Suit {
		case SuitFighter:
			pairs = append(pairs, missionPair{"for", str("fighter")})
		case SuitMage:
			pairs = append(pairs, missionPair{"for", str("mage")})
		}
		if r.Stock {
			pairs = append(pairs, missionPair{"shop", "true"})
		}
		pairs = opt(pairs, "sprite", r.Sprite)
		pairs = opt(pairs, "layer", r.Layer)
		if r.Layer != "" && r.Anchor != r.Slot {
			pairs = opt(pairs, "anchor", r.Anchor)
		}
		pairs = opt(pairs, "figure", r.Figure)
		out = append(out, record{r.Line, sourceRow{table: "item", identity: "item " + r.Key, pairs: pairs}})
	}
	for _, w := range d.Weapons {
		pairs := []missionPair{{"key", str(w.Key)}, {"name", str(w.NameKey)}, {"base", str(w.Base)}}
		pairs = opt(pairs, "sprite", w.Sprite)
		for _, wp := range WeaponParams {
			if i := slices.IndexFunc(w.Params, func(v WeaponParamValue) bool { return v.Key == wp.Key }); i >= 0 {
				pairs = append(pairs, missionPair{wp.Key, num(w.Params[i].Value)})
			}
		}
		out = append(out, record{w.Line, sourceRow{table: "weapon", identity: "weapon " + w.Key, pairs: pairs}})
	}
	for _, c := range d.Changes {
		pairs := []missionPair{{"item", str(c.Item)}}
		for _, f := range []struct {
			key string
			v   *int32
		}{{"defence", c.Defence}, {"absorption", c.Absorption}, {"weight", c.Weight}, {"price", c.Price}} {
			if f.v != nil {
				pairs = append(pairs, missionPair{f.key, num(*f.v)})
			}
		}
		if c.Off != nil {
			pairs = append(pairs, missionPair{"off", strconv.FormatBool(*c.Off)})
		}
		out = append(out, record{c.Line, sourceRow{table: "change", identity: "change " + c.Item, pairs: pairs}})
	}
	for _, m := range d.Materials {
		pairs := []missionPair{{"item", str(m.Item)}, {"tier", str(m.Tier)}, {"material", str(m.Material)}, {"allow", strconv.FormatBool(m.Allow)}}
		pairs = opt(pairs, "sprite", m.Sprite)
		pairs = opt(pairs, "name", m.NameKey)
		out = append(out, record{m.Line, sourceRow{table: "material", identity: "material " + m.Item + "\x00" + m.Tier + "\x00" + m.Material, pairs: pairs}})
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
	rows, seen := make([]sourceRow, len(out)), map[string]int{}
	for i, r := range out {
		rows[i] = r.row
		seen[r.row.identity]++
		rows[i].identity += "\x00" + strconv.Itoa(seen[r.row.identity])
	}
	return rows
}
