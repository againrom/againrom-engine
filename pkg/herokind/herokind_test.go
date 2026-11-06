package herokind

import (
	"strings"
	"testing"
)

func TestKindIsTheComposedSelector(t *testing.T) {
	for _, c := range []struct {
		mage, female bool
		want         Kind
		words        string
	}{
		{false, false, 0, "male fighter"}, {false, true, 1, "female fighter"},
		{true, false, 2, "male mage"}, {true, true, 3, "female mage"},
	} {
		k := Of(c.mage, c.female)
		if k != c.want || k.Mage() != c.mage || k.Female() != c.female || k.String() != c.words {
			t.Errorf("Of(%v, %v) = %d %q", c.mage, c.female, k, k)
		}
	}
}

func TestSetAxesNameOnlyTheKindsPresent(t *testing.T) {
	for set, want := range [][4]bool{
		{false, false, false, false}, {false, true, false, true},
		{true, false, false, true}, {true, true, false, true},
		{false, true, true, false}, {false, true, true, true},
		{true, true, true, true}, {true, true, true, true},
		{true, false, true, false}, {true, true, true, true},
		{true, false, true, true}, {true, true, true, true},
		{true, true, true, false}, {true, true, true, true},
		{true, true, true, true}, {true, true, true, true},
	} {
		female, male, mage, fighter := Set(set).Axes()
		if got := [4]bool{female, male, mage, fighter}; got != want {
			t.Errorf("set %04b axes %v, want %v", set, got, want)
		}
	}
}

func TestParseNamesTheHeroesOfAClassAndASex(t *testing.T) {
	for _, c := range []struct {
		class, sex string
		want       Set
		words      string
	}{
		{"", "", Every, "every hero"},
		{"fighter", "", SetOf(MaleFighter, FemaleFighter), "a fighter"},
		{"mage", "", SetOf(MaleMage, FemaleMage), "a mage"},
		{"", "male", SetOf(MaleFighter, MaleMage), "a male hero"},
		{"", "female", SetOf(FemaleFighter, FemaleMage), "a female hero"},
		{"mage", "female", SetOf(FemaleMage), "a female mage"},
		{"fighter", "male", SetOf(MaleFighter), "a male fighter"},
	} {
		got, err := Parse(c.class, c.sex)
		if err != nil || got != c.want || got.String() != c.words || Named(c.class, c.sex) != got {
			t.Errorf("Parse(%q, %q) = %b %q %v", c.class, c.sex, got, got, err)
		}
	}
	if got := SetOf(FemaleFighter, MaleMage).String(); got != "a female fighter or a male mage" {
		t.Errorf("mixed set %q", got)
	}
	for _, bad := range [][2]string{{"Mage", ""}, {"", "f"}, {"rogue", "male"}} {
		if _, err := Parse(bad[0], bad[1]); err == nil || !strings.Contains(err.Error(), "neither") || Named(bad[0], bad[1]) != 0 {
			t.Errorf("Parse(%q, %q): %v", bad[0], bad[1], err)
		}
	}
}

func TestChooseTakesTheAlternativeOfTheHeroElseTheDefault(t *testing.T) {
	fighter, _ := Parse("fighter", "")
	maleMage, _ := Parse("mage", "male")
	alternatives := []Set{fighter, maleMage}
	for k, want := range map[Kind]int{MaleFighter: 0, FemaleFighter: 0, MaleMage: 1, FemaleMage: -1} {
		if got := Choose(k, alternatives, -1); got != want {
			t.Errorf("%v chose %d, want %d", k, got, want)
		}
	}
	if got := Choose(FemaleMage, alternatives, 7); got != 7 {
		t.Errorf("the default is %d", got)
	}
	if _, _, ok := Overlap(alternatives); ok {
		t.Error("a fighter and a male mage overlap")
	}
	female, _ := Parse("", "female")
	if i, j, ok := Overlap([]Set{maleMage, fighter, female}); !ok || i != 1 || j != 2 {
		t.Errorf("a fighter and a female hero: %d %d %v", i, j, ok)
	}
	if _, _, ok := Overlap([]Set{Every, maleMage}); !ok {
		t.Error("every hero overlaps nothing")
	}
}

func TestAnOrderPlacesTheFourKinds(t *testing.T) {
	for _, c := range []struct {
		k        Kind
		selector int
		picture  int
		audience int
	}{
		{MaleFighter, 0, 0, 3}, {FemaleFighter, 1, 2, 1}, {MaleMage, 2, 1, 2}, {FemaleMage, 3, 3, 0},
	} {
		if c.k.Index() != c.selector || c.k.In(PictureOrder) != c.picture {
			t.Errorf("%v: Index %d, In(PictureOrder) %d; want %d, %d", c.k, c.k.Index(), c.k.In(PictureOrder), c.selector, c.picture)
		}
		if PictureOrder[c.picture] != c.k {
			t.Errorf("PictureOrder[%d] = %v, want %v", c.picture, PictureOrder[c.picture], c.k)
		}
		if c.k.In(AudienceOrder) != c.audience || AudienceOrder[c.audience] != c.k {
			t.Errorf("%v: In(AudienceOrder) %d, AudienceOrder[%d] %v", c.k, c.k.In(AudienceOrder), c.audience, AudienceOrder[c.audience])
		}
	}
}
