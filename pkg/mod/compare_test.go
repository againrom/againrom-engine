package mod

import (
	"reflect"
	"testing"
)

// TestCompareNamesEachDifferenceAsData: every kind carries the mod, the
// setting and both values, and Differences is each one's prose.
func TestCompareNamesEachDifferenceAsData(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Set)
		want []Difference
	}{
		{"base", func(x *Set) { x.Base = "rom1-ru" }, []Difference{{Kind: DifferenceBase, Saved: "rom1-en", Current: "rom1-ru"}}},
		{"missing", func(x *Set) { x.Mods = x.Mods[:1] }, []Difference{{Kind: DifferenceMissing, Mod: "b", Saved: "2"}}},
		{"added", func(x *Set) { x.Mods = append(x.Mods, SetEntry{ID: "c", Version: "4"}) },
			[]Difference{{Kind: DifferenceAdded, Mod: "c", Current: "4"}}},
		{"version", func(x *Set) { x.Mods[1].Version = "3" }, []Difference{{Kind: DifferenceVersion, Mod: "b", Saved: "2", Current: "3"}}},
		{"files", func(x *Set) { x.Mods[1].Digest = "zz" }, []Difference{{Kind: DifferenceFiles, Mod: "b", Saved: "2", Current: "2"}}},
		{"setting", func(x *Set) { x.Mods[0].Settings[0].Value.Int = 120 },
			[]Difference{{Kind: DifferenceSetting, Mod: "a", Setting: "skill_cap", Saved: "150", Current: "120"}}},
		{"setting gone", func(x *Set) { x.Mods[0].Settings = nil },
			[]Difference{{Kind: DifferenceSettingGone, Mod: "a", Setting: "skill_cap", Saved: "150"}}},
		{"setting new", func(x *Set) {
			x.Mods[1].Settings = []SettingValue{{Key: "loud", Value: Value{Kind: KindBool, Bool: true}}}
		}, []Difference{{Kind: DifferenceSettingNew, Mod: "b", Setting: "loud", Current: "true"}}},
		{"order", func(x *Set) { x.Mods[0], x.Mods[1] = x.Mods[1], x.Mods[0] },
			[]Difference{{Kind: DifferenceOrder, Saved: "a, b", Current: "b, a"}}},
	}
	for _, c := range cases {
		have := sampleSet()
		c.mut(&have)
		got := Compare(sampleSet(), have)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Compare = %+v, want %+v", c.name, got, c.want)
		}
		prose := Differences(sampleSet(), have)
		if len(prose) != len(got) {
			t.Fatalf("%s: %d lines of prose for %d differences", c.name, len(prose), len(got))
		}
		for i := range got {
			if prose[i] != got[i].String() {
				t.Errorf("%s: line %q is not the difference's prose %q", c.name, prose[i], got[i].String())
			}
		}
	}
}
