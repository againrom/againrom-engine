package game

import (
	"slices"
	"testing"
)

// heroValues are values of slots 776 and 781: the four the hero is read
// from, then values other than 0 and 1, which read as nonzero.
var heroValues = [][2]int32{{0, 0}, {1, 0}, {0, 1}, {1, 1}, {-1, 7}, {2, 0}, {0, -2}}

// A mod's [[inn]] row that names zero or nonzero on a row chosen by the
// starting hero replaces that list with the slot gate the list held for
// the hero, so the offer list is the one the slot gates give; a row that
// names neither list keeps the row's class and sex.
func TestModGateOnAHeroInnRowReplacesItsSlotGate(t *testing.T) {
	layer := func(id, towns string) *townBook {
		t.Helper()
		dir := t.TempDir()
		writeInnMod(t, dir, id, towns)
		book, overrides, err := layerTownBook(townBooks["rom2"], runTownModsFor(t, "rom2-en", dir, id).Towns, Campaign{}, nil, nil, modTownInstall{})
		if err != nil || len(overrides) != 0 {
			t.Fatalf("%s: layer: %v, overrides %v", id, err, overrides)
		}
		return book
	}
	// offered is the topics of topics the inn of book offers at stage with
	// the hero values v and gate set.
	offered := func(book *townBook, v [2]int32, stage int32, gate int, topics ...int) []int {
		bank := map[int]int32{776: v[0], 781: v[1]}
		if gate > 0 {
			bank[gate] = 1
		}
		c := innAt(2, stage, bank)
		c.offers = book.offerRows()
		var got []int
		for _, o := range c.innOptions() {
			if slices.Contains(topics, o.topic) {
				got = append(got, o.topic)
			}
		}
		return got
	}
	family := []int{74, 75, 76, 77}
	for _, tc := range []struct {
		name, towns string
		// inn80 and fam are the offers of records 81 and 82 at stage 80 and
		// of the 2022 family under slot 959, one per value of heroValues:
		// the base's lists, its rows gating on 776 and 781 in zero and
		// nonzero.
		inn80, fam [][]int
	}{
		{"probe", "[[inn]]\ntopic = 82\nnpc = 2009\nzero = [978]\n\n[[inn]]\ntopic = 74\nnpc = 2022\nzero = [970, 971, 972, 973]\n",
			[][]int{{82}, {81, 82}, {82}, {81, 82}, {81, 82}, {81, 82}, {82}},
			[][]int{{74}, {74, 77}, {74, 75}, {74, 76}, {74, 76}, {74, 77}, {74, 75}}},
		// Row 75 holds 781 in nonzero and 776 in zero: naming zero drops
		// 776 and keeps 781.
		{"zero-only", "[[inn]]\ntopic = 75\nnpc = 2022\nzero = [970, 971, 972, 973]\n",
			[][]int{{82}, {81}, {82}, {81}, {81}, {81}, {82}},
			[][]int{{74}, {77}, {75}, {75, 76}, {75, 76}, {77}, {75}}},
		{"neither", "[[inn]]\ntopic = 76\nnpc = 2022\ntown = 2\n",
			[][]int{{82}, {81}, {82}, {81}, {81}, {81}, {82}},
			[][]int{{74}, {77}, {75}, {76}, {76}, {77}, {75}}},
	} {
		book := layer(tc.name, tc.towns)
		for i, v := range heroValues {
			if got := offered(book, v, 80, 0, 81, 82); !slices.Equal(got, tc.inn80[i]) {
				t.Errorf("%s, 776=%d 781=%d: inn 80 offers %v, want %v", tc.name, v[0], v[1], got, tc.inn80[i])
			}
			if got := offered(book, v, 20, 959, family...); !slices.Equal(got, tc.fam[i]) {
				t.Errorf("%s, 776=%d 781=%d: family offers %v, want %v", tc.name, v[0], v[1], got, tc.fam[i])
			}
		}
	}
	for _, r := range townBooks["rom2"].offerRows() {
		if r.Topic == 75 && (r.Class != "fighter" || r.Sex != "female" || slices.Contains(r.Nonzero, 781)) {
			t.Fatalf("the layer changed the shipped row %+v", r)
		}
	}
}
