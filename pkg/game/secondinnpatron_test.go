package game

import (
	"strings"
	"testing"

	"againrom/pkg/vfs"
)

// An inn patron's own name is the npcnames entry of a key of 1000 or more;
// a smaller key has none and the sheet heads the card with Mercenary over
// the type (R2-ENGINE-581, DIV-4087).
func TestSecondInnPersonName(t *testing.T) {
	lines := make([]string, 2200)
	lines[6], lines[16] = "Mercenary", "Mercenary"
	lines[2107], lines[2108], lines[2109] = "Peasant", "Neighbour", "Guard"
	fsys, err := vfs.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	fsys.Overlay(npcNamesPath, func() ([]byte, error) { return []byte(strings.Join(lines, "\r\n")), nil })
	in := &InstallResources{Archives: &Archives{Containers: fsys}}
	for npc, want := range map[int]string{
		207: "", 517: "", 7: "", 17: "", 2108: "Peasant", 2109: "Neighbour", 2110: "Guard", 3000: "", 5000: "",
	} {
		if got := secondInnPersonName(in, npc); got != want {
			t.Errorf("key %d: %q, want %q", npc, got, want)
		}
	}
}

func TestSecondInnPatronNamesInstalled(t *testing.T) {
	archives := secondGameRoot(t)
	in := &InstallResources{Archives: archives}
	payload, err := archives.Containers.ReadFile(npcNamesPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(in.textCode().Bytes(payload)), "\r\n", "\n"), "\n")
	for _, npc := range []int{2108, 2110} {
		want := lines[npc-1]
		if want == "" {
			t.Fatalf("NPC %d has an empty installed line", npc)
		}
		got := secondInnPersonName(in, npc)
		t.Logf("NPC %d reads %q (line %d of %d; neighbours %q, %q)", npc, got, npc, len(lines), lines[npc-2], lines[npc])
		if got != want {
			t.Errorf("NPC %d reads %q, want its own line %q", npc, got, want)
		}
	}
}

// The SPELLCASTER caption's actor value: the first game's person's spell mask
// behind a positive mana maximum (TAVERN-035) and a creature's projected value
// (DIV-1809); the second game's projected value held at zero without a mana
// pool, whatever the mask (R2-ENGINE-580, DIV-4086).
func TestSpellcasterValue(t *testing.T) {
	for _, c := range []struct {
		files       *gameFiles
		value, mana int32
		spells      uint32
		person      bool
		want        int32
		what        string
	}{
		{&firstGameFiles, 1079, 0, 0, false, 1079, "first game creature, no mana"},
		{&firstGameFiles, 1079, 30, 0, false, 1079, "first game creature, mana"},
		{&firstGameFiles, 1079, 30, 0, true, 0, "first game person, no spell"},
		{&firstGameFiles, 1079, 0, 1 << 3, true, 0, "first game person, spell, no mana"},
		{&firstGameFiles, 1079, 120, 1<<3 | 1, true, 1 << 3, "first game person, spells"},
		{&firstGameFiles, 1079, 120, 1, true, 0, "first game person, id 0 only"},
		{&secondGameFiles, 1079, 0, 1 << 3, true, 0, "second game, no mana"},
		{&secondGameFiles, 250536, 604, 0, true, 250536, "second game, mana, no spell"},
		{&secondGameFiles, 250536, 604, 1 << 3, false, 250536, "second game creature, mana"},
	} {
		if got := c.files.spellcasterValue(c.value, c.mana, c.spells, c.person); got != c.want {
			t.Errorf("%s: %d, want %d", c.what, got, c.want)
		}
	}
}
