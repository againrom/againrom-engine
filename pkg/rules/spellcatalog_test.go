package rules

import (
	"slices"
	"testing"
)

// The tables the catalog replaced, written out value for value.
var (
	firstBookBefore  = []uint16{1, 2, 3, 4, 5, 23, 24, 16, 15, 14, 13, 12, 6, 7, 8, 9, 10, 25, 26, 22, 21, 20, 19, 18}
	secondBookBefore = []uint16{1, 2, 3, 4, 24, 20, 21, 26, 13, 12, 11, 10, 5, 6, 7, 8, 25, 22, 23, 27, 19, 18, 17, 16}
	secondArmsBefore = [...]uint16{
		1: 1, 2: 2, 3: 3, 4: 5, 5: SpellArmNone, 6: 8, 7: 21, 8: 10, 9: 9, 10: 13,
		11: 14, 12: 15, 13: 16, 14: 17, 15: 12, 16: SpellArmNone, 17: 19, 18: 20, 19: 22,
		20: 23, 21: 24, 22: 25, 23: 26, 24: 6, 25: SpellArmSummon, 26: 11, 27: 18, 28: 27,
		29: 28,
	}
)

func shopBookBefore(id int) bool {
	excluded := map[int]bool{6: true, 11: true, 12: true, 17: true, 18: true, 27: true}
	return id >= 2 && id <= 27 && !excluded[id]
}

func TestSpellCatalogProfilesReproduceTheReplacedTables(t *testing.T) {
	if SpellIDs != 28 || len(SpellProfile(SpellGameFirst)) != 28 || len(SpellProfile(SpellGameSecond)) != 29 {
		t.Fatalf("profiles have %d and %d spells, SpellIDs %d; want 28, 29, 28",
			len(SpellProfile(SpellGameFirst)), len(SpellProfile(SpellGameSecond)), SpellIDs)
	}
	if got := SpellBookRows(SpellGameFirst); !slices.Equal(got, firstBookBefore) {
		t.Errorf("first game book rows = %v, want %v", got, firstBookBefore)
	}
	if got := SpellBookRows(SpellGameSecond); !slices.Equal(got, secondBookBefore) {
		t.Errorf("second game book rows = %v, want %v", got, secondBookBefore)
	}
	for _, g := range []SpellGame{SpellGameFirst, SpellGameSecond} {
		for id := -5; id <= 70; id++ {
			if got, want := SpellShopBook(g, id), shopBookBefore(id); got != want {
				t.Errorf("game %d book of %d sold = %v, want %v", g, id, got, want)
			}
			if got, want := SpellShopScroll(g, id), id >= 1 && id <= 27; got != want {
				t.Errorf("game %d scroll of %d sold = %v, want %v", g, id, got, want)
			}
		}
	}
	for id := -5; id <= 70; id++ {
		want := SpellArmNone
		if id > 0 && id < len(secondArmsBefore) {
			want = secondArmsBefore[id]
		}
		if got := SpellArm(SpellGameSecond, id); got != want {
			t.Errorf("second game arm of %d = %#x, want %#x", id, got, want)
		}
		if id >= 1 && id <= SpellIDs {
			if got := SpellArm(SpellGameFirst, id); got != uint16(id) {
				t.Errorf("first game arm of %d = %d", id, got)
			}
		}
	}
}

func TestSpellCatalogNamesEachSpellOnce(t *testing.T) {
	names := SpellCatalog()
	if len(names) != 32 {
		t.Fatalf("catalog has %d names, want 28 first-game and 4 second-game-only", len(names))
	}
	for i, s := range SpellProfile(SpellGameFirst) {
		if names[i] != s.Name {
			t.Errorf("catalog name %d = %q, want the first game's %q", i, names[i], s.Name)
		}
	}
	if got, want := names[28:], []string{"Ice_Missile", "Blizzard", "Diamond_Dust", "Summon"}; !slices.Equal(got, want) {
		t.Errorf("second-game-only names = %v, want %v", got, want)
	}
	for _, c := range []struct {
		name          string
		first, second int
	}{
		{"Fire Ball", 2, 2}, {"Fire_Ball", 2, 2}, {"Heal", 6, 24}, {"Drain_Life", 11, 26},
		{"Meteor_Storm", 21, 0}, {"Fire_Sacrifice", 4, 0}, {"Freezing_Cloud", 7, 0},
		{"Ice Missile", 0, 5}, {"Blizzard", 0, 7}, {"Diamond_Dust", 0, 16}, {"Summon", 0, 25},
		{"Slow", 28, 29}, {"Curse", 27, 28}, {"Nothing", 0, 0},
	} {
		for g, want := range []int{c.first, c.second} {
			got, ok := SpellNumber(SpellGame(g), c.name)
			if got != want || ok != (want != 0) {
				t.Errorf("game %d number of %q = %d %v, want %d", g, c.name, got, ok, want)
			}
		}
	}
}
