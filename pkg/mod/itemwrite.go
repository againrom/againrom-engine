package mod

import (
	"cmp"
	"fmt"
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
	was, err := itemSourceRows(before)
	if err != nil {
		return nil, err
	}
	after, err := itemSourceRows(d.Items)
	if err != nil {
		return nil, err
	}
	out, err := writeRows(source, was, after, func(key string) string { return key })
	if err != nil {
		return nil, err
	}
	back, err := ParseItems("", ItemsFile, out, lenient, "")
	if err != nil {
		return nil, err
	}
	again, err := itemSourceRows(back)
	if err != nil {
		return nil, err
	}
	if !sameRows(again, after) {
		return nil, fmt.Errorf("%s: the written records do not read back as the records given", ItemsFile)
	}
	return out, nil
}

// itemSourceRows is d's records as the row writer sees them, in their file
// order and a new record after them: an added row by its key, a change by
// the row it names and a material by its pair, a repeat numbered.
func itemSourceRows(d ItemData) ([]sourceRow, error) {
	type record struct {
		line int
		row  sourceRow
	}
	var out []record
	var firstErr error
	str := func(key, value string) string {
		if firstErr != nil {
			return ""
		}
		q, err := writeString(value)
		if err != nil {
			firstErr = fmt.Errorf("%s: %w", key, err)
			return ""
		}
		return q
	}
	num := func(v int32) string { return strconv.Itoa(int(v)) }
	opt := func(pairs []missionPair, key, value string) []missionPair {
		if value != "" {
			pairs = append(pairs, missionPair{key, str(key, value)})
		}
		return pairs
	}
	for _, r := range d.Rows {
		pairs := []missionPair{{"key", str("key", r.Key)}, {"name", str("name", r.NameKey)}, {"slot", str("slot", r.Slot)},
			{"defence", num(r.Defence)}, {"absorption", num(r.Absorption)}, {"weight", num(r.Weight)}, {"price", num(r.Price)},
			{"stand-in", str("stand-in", r.StandIn)}}
		switch r.Suit {
		case SuitFighter:
			pairs = append(pairs, missionPair{"for", str("for", "fighter")})
		case SuitMage:
			pairs = append(pairs, missionPair{"for", str("for", "mage")})
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
		if firstErr != nil {
			return nil, firstErr
		}
		out = append(out, record{r.Line, sourceRow{table: "item", identity: "item " + r.Key, pairs: pairs}})
	}
	for _, w := range d.Weapons {
		pairs := []missionPair{{"key", str("key", w.Key)}, {"name", str("name", w.NameKey)}, {"base", str("base", w.Base)}}
		pairs = opt(pairs, "sprite", w.Sprite)
		if firstErr != nil {
			return nil, firstErr
		}
		for _, wp := range WeaponParams {
			if i := slices.IndexFunc(w.Params, func(v WeaponParamValue) bool { return v.Key == wp.Key }); i >= 0 {
				pairs = append(pairs, missionPair{wp.Key, num(w.Params[i].Value)})
			}
		}
		out = append(out, record{w.Line, sourceRow{table: "weapon", identity: "weapon " + w.Key, pairs: pairs}})
	}
	for _, c := range d.Changes {
		pairs := []missionPair{{"item", str("item", c.Item)}}
		if firstErr != nil {
			return nil, firstErr
		}
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
		pairs := []missionPair{{"item", str("item", m.Item)}, {"tier", str("tier", m.Tier)}, {"material", str("material", m.Material)}, {"allow", strconv.FormatBool(m.Allow)}}
		pairs = opt(pairs, "sprite", m.Sprite)
		pairs = opt(pairs, "name", m.NameKey)
		if firstErr != nil {
			return nil, firstErr
		}
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
	rows, lines := make([]sourceRow, len(out)), make([]int, len(out))
	for i, r := range out {
		rows[i], lines[i] = r.row, r.line
	}
	numberRows(rows, lines)
	return rows, nil
}
