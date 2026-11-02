package game

import (
	"strconv"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// Every table of the game indexed by a hero's class and sex reads the entry it
// read before the index came from one place.
func TestGameTablesKeepTheirHeroEntries(t *testing.T) {
	names := [4]string{"a", "b", "c", "d"}
	table := &mapload.Table{HeroNames: names}
	in := townInstall{npcName: func(i int, _ string) string { return strconv.Itoa(i) }}
	for _, c := range []struct {
		mage, female bool
		archetype    int
		picture      int
	}{
		{false, false, 0, 0}, {false, true, 1, 2}, {true, false, 2, 1}, {true, true, 3, 3},
	} {
		if got, want := defaultHeroName(table, c.mage, c.female), names[c.picture]; got != want {
			t.Errorf("defaultHeroName(%v, %v) = %q, want %q", c.mage, c.female, got, want)
		}
		p := mapload.PartyMember{FigureDir: string(data.FigureDirFor(c.mage, c.female)), Mage: c.mage, Name: "x"}
		if got, want := in.companionName(p), strconv.Itoa(20+c.archetype); got != want {
			t.Errorf("companionName(%v, %v) = %q, want %q", c.mage, c.female, got, want)
		}
		p.CompanionNPC = 21
		if got, want := companionNameRow(p), 20+c.archetype; got != want {
			t.Errorf("companionNameRow(%v, %v) = %d, want %d", c.mage, c.female, got, want)
		}
	}
}
