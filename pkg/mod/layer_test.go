package mod

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// layerRow is a record kind for the layer step's own tests: a key and two
// fields.
type layerRow struct {
	Key  string
	Name string
	Size int
}

func rowPatch(m string, line int, key string, set map[string]any) Patch[string, layerRow] {
	p := Patch[string, layerRow]{Source: Source{Mod: m, File: "data/rows.toml", Line: line}, Key: key}
	for _, f := range []string{"name", "size", OffField} {
		v, ok := set[f]
		if !ok {
			continue
		}
		p.Fields = append(p.Fields, PatchField{Name: f, Line: line + 1})
		if f == OffField {
			p.Off = v.(bool)
		}
	}
	p.Apply = func(r *layerRow) {
		r.Key = key
		if v, ok := set["name"]; ok {
			r.Name = v.(string)
		}
		if v, ok := set["size"]; ok {
			r.Size = v.(int)
		}
	}
	return p
}

func mergeRows(patches []Patch[string, layerRow], order Precedence) (*Layered[string, layerRow], error) {
	base := []layerRow{{Key: "a", Name: "A", Size: 1}, {Key: "b", Name: "B", Size: 2}}
	return Merge("row", base, func(r layerRow) string { return r.Key }, patches, order, nil)
}

// A layer adds a record by a new key, changes only the fields it names of
// an existing one, and switches a record off and back on; layer 0 keeps its
// order and added records follow.
func TestLayerAddsChangesAndSwitchesOff(t *testing.T) {
	l, err := mergeRows([]Patch[string, layerRow]{
		rowPatch("m", 1, "a", map[string]any{"size": 9}),
		rowPatch("m", 3, "c", map[string]any{"name": "C", "size": 3}),
		rowPatch("m", 5, "b", map[string]any{OffField: true}),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []layerRow{{Key: "a", Name: "A", Size: 9}, {Key: "c", Name: "C", Size: 3}}
	if !reflect.DeepEqual(l.Records, want) || len(l.Overrides) != 0 {
		t.Fatalf("records %+v overrides %v, want %+v", l.Records, l.Overrides, want)
	}
	if l.Off["b"].Line != 5 || l.Named["c"].Line != 3 {
		t.Fatalf("off %v named %v", l.Off, l.Named)
	}
	l, err = mergeRows([]Patch[string, layerRow]{
		rowPatch("m", 1, "b", map[string]any{OffField: true}),
		rowPatch("n", 1, "b", map[string]any{OffField: false, "size": 5}),
	}, Precedence{"n": {"m"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []layerRow{{Key: "a", Name: "A", Size: 1}, {Key: "b", Name: "B", Size: 5}}; !reflect.DeepEqual(l.Records, want) || len(l.Off) != 0 {
		t.Fatalf("switched back on: %+v off %v", l.Records, l.Off)
	}
	if _, err := mergeRows([]Patch[string, layerRow]{rowPatch("m", 7, "z", map[string]any{OffField: true})}, nil); err == nil ||
		err.Error() != `mod "m": data/rows.toml:7: row z is switched off, but no layer holds it` {
		t.Fatalf("switching off an unknown key: %v", err)
	}
}

// Two mods that set one field: the later wins and is named when it loads
// after the earlier, directly or through another mod; otherwise the merge
// refuses naming both. One new key from two unordered mods is refused the
// same way. Fields two mods set apart need no order.
func TestLayerOrdersTwoModsOnOneField(t *testing.T) {
	two := []Patch[string, layerRow]{
		rowPatch("m", 1, "a", map[string]any{"name": "M"}),
		rowPatch("n", 4, "a", map[string]any{"name": "N"}),
	}
	l, err := mergeRows(two, Precedence{"n": {"x"}, "x": {"m"}})
	if err != nil {
		t.Fatal(err)
	}
	want := Override{Kind: "row", Key: "a", Field: "name", Earlier: "m", Later: Source{Mod: "n", File: "data/rows.toml", Line: 5}}
	if l.Records[0].Name != "N" || !reflect.DeepEqual(l.Overrides, []Override{want}) {
		t.Fatalf("records %+v overrides %+v", l.Records, l.Overrides)
	}
	if got := want.String(); got != `row a name: mod "n" (data/rows.toml:5) overrides mod "m"` {
		t.Fatalf("override line %q", got)
	}
	var le *LayerError
	if _, err := mergeRows(two, nil); !errors.As(err, &le) || le.Mod != "n" || le.Line != 4 ||
		!strings.Contains(err.Error(), `row a name is set by mod "m" and by mod "n"`) {
		t.Fatalf("unordered: %v", err)
	}
	if _, err := mergeRows(two, Precedence{"m": {"n"}}); err == nil {
		t.Fatal("an order the other way round admits the later mod")
	}
	apart := []Patch[string, layerRow]{
		rowPatch("m", 1, "a", map[string]any{"name": "M"}),
		rowPatch("n", 1, "a", map[string]any{"size": 4}),
	}
	if l, err := mergeRows(apart, nil); err != nil || l.Records[0] != (layerRow{Key: "a", Name: "M", Size: 4}) {
		t.Fatalf("fields apart: %+v %v", l, err)
	}
	added := []Patch[string, layerRow]{
		rowPatch("m", 1, "c", map[string]any{"name": "C"}),
		rowPatch("n", 1, "c", map[string]any{"size": 3}),
	}
	if _, err := mergeRows(added, nil); err == nil || !strings.Contains(err.Error(), `row c is added by mod "m" and named by mod "n"`) {
		t.Fatalf("one new key from two mods: %v", err)
	}
	if l, err := mergeRows(added, Precedence{"n": {"m"}}); err != nil || l.Records[2] != (layerRow{Key: "c", Name: "C", Size: 3}) {
		t.Fatalf("an ordered change of an added key: %+v %v", l, err)
	}
}

// The check runs after each mod's last patch, over the records so far, and
// its refusal stops the merge.
func TestLayerChecksAfterEachMod(t *testing.T) {
	var seen []string
	check := func(m string, l *Layered[string, layerRow]) error {
		seen = append(seen, m)
		if len(l.Records) > 3 {
			return errors.New("too many")
		}
		return nil
	}
	patches := []Patch[string, layerRow]{
		rowPatch("m", 1, "c", map[string]any{"name": "C"}),
		rowPatch("m", 3, "d", map[string]any{"name": "D"}),
		rowPatch("n", 1, "a", map[string]any{"size": 3}),
	}
	base := []layerRow{{Key: "a"}, {Key: "b"}}
	if _, err := Merge("row", base, func(r layerRow) string { return r.Key }, patches, nil, check); err == nil || !reflect.DeepEqual(seen, []string{"m"}) {
		t.Fatalf("check after mods %v: %v", seen, err)
	}
}

// Precedence follows requires and load-after among the enabled mods.
func TestPrecedenceOfTheManifests(t *testing.T) {
	entry := func(id string, requires, after []string) Entry {
		return Entry{Manifest: Manifest{ID: id, Requires: requires, LoadAfter: after}}
	}
	p := PrecedenceOf([]Entry{entry("a", nil, nil), entry("b", []string{"a>=1.0"}, nil), entry("c", nil, []string{"b", "absent"})})
	if !p.After("c", "a") || !p.After("b", "a") || p.After("a", "c") || p.After("c", "absent") {
		t.Fatalf("precedence %v", p)
	}
}
