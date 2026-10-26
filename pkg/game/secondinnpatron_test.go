package game

import (
	"testing"

	"againrom/pkg/mapload"
)

// An inn patron's own name is the npcnames entry of a key of 1000 or more;
// a smaller key has none and the sheet heads the card with Mercenary over
// the type (R2-ENGINE-581, DIV-4087).
func TestSecondInnPersonName(t *testing.T) {
	tbl := &mapload.Table{NPCNames: make([]string, 2200)}
	tbl.NPCNames[7], tbl.NPCNames[17], tbl.NPCNames[2108] = "Mercenary", "Mercenary", "Peasant"
	for npc, want := range map[int]string{207: "", 517: "", 7: "", 2108: "Peasant", 5000: ""} {
		if got := secondInnPersonName(tbl, npc); got != want {
			t.Errorf("key %d: %q, want %q", npc, got, want)
		}
	}
}

// The SPELLCASTER caption's actor value: the first game's as projected; the
// second game's held at zero without a mana pool (R2-ENGINE-580, DIV-4086).
func TestSpellcasterValue(t *testing.T) {
	for _, c := range []struct {
		files       *gameFiles
		value, mana int32
		want        int32
		what        string
	}{
		{&firstGameFiles, 1079, 0, 1079, "first game, no mana"},
		{&secondGameFiles, 1079, 0, 0, "second game, no mana"},
		{&secondGameFiles, 250536, 604, 250536, "second game, mana"},
	} {
		if got := c.files.spellcasterValue(c.value, c.mana); got != c.want {
			t.Errorf("%s: %d, want %d", c.what, got, c.want)
		}
	}
}
