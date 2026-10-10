package game_test

// The default hero, the one a mission opened without the generator's name
// field starts with, carries the installed name of his picture. Every fixture
// is a synthetic archive with invented names; no game install is read.

import (
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/vfs"
)

// heroNamesFS is an install whose main archive holds an npcnames.txt of the
// given entries, one line each in CRLF text, and whose world and scenario
// archives are the least LoadDefinitions reads.
func heroNamesFS(t *testing.T, entries []string) *vfs.FS {
	t.Helper()
	dir := installDir(t, map[string][]byte{
		game.MainArchive: synth.Archive([]synth.File{
			{Path: "text/npcnames.txt", Data: []byte(strings.Join(entries, "\r\n") + "\r\n")},
		}),
		game.WorldArchive:    synth.Archive([]synth.File{{Path: "data/data.bin", Data: synth.DataBin{}.Bytes()}}),
		game.ScenarioArchive: synth.Archive([]synth.File{{Path: "npc.reg", Data: synth.NPCReg(nil)}}),
	})
	fsys, err := vfs.Open([]string{
		dir + "/" + game.MainArchive, dir + "/" + game.WorldArchive, dir + "/" + game.ScenarioArchive,
	}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return fsys
}

// npcNameEntries is an npcnames.txt of 24 entries where the four hero pictures'
// entries 20, 21, 22 and 23 are the male fighter's, the female fighter's, the
// male mage's and the female mage's, as the generator's name field reads them.
func npcNameEntries(maleFighter, femaleFighter, maleMage, femaleMage string) []string {
	entries := make([]string, 24)
	for i := range entries {
		entries[i] = "filler"
	}
	entries[20], entries[21], entries[22], entries[23] = maleFighter, femaleFighter, maleMage, femaleMage
	return entries
}

// The default hero is the male fighter, or the male mage when a caller asks for
// the mage arm, and carries that picture's installed name byte for byte: the
// RU install's own name is code page 866 bytes, which no pass may re-encode.
func TestMissionPartyNamesTheHeroAfterHisPicture(t *testing.T) {
	const maleFighter, maleMage = "\x84\xa0\xad\xa0\xe1", "\x94\xa5\xe0\xa3\xa0\xe0\xa4"
	d, err := game.LoadDefinitions(heroNamesFS(t, npcNameEntries(maleFighter, "female fighter", maleMage, "female mage")))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	if got := game.MissionParty(nil, data.BodyList{}, d.Table)[0].Name; got != maleFighter {
		t.Errorf("MissionParty names the hero % x, want the male fighter's % x", got, maleFighter)
	}
	if got := game.MissionPartyAs(false, nil, data.BodyList{}, d.Table)[0].Name; got != maleFighter {
		t.Errorf("MissionPartyAs(false) names the hero % x, want the male fighter's % x", got, maleFighter)
	}
	if got := game.MissionPartyAs(true, nil, data.BodyList{}, d.Table)[0].Name; got != maleMage {
		t.Errorf("MissionPartyAs(true) names the hero % x, want the male mage's % x", got, maleMage)
	}
}

// A table that carries no installed name for his picture names him Danath, as
// every default hero was named before the names were read: no table, an
// install with no name file, and a name file that lacks one of the four
// pictures' entries, which the table then treats as no names at all.
func TestMissionPartyNamesTheHeroDanathWithoutInstalledNames(t *testing.T) {
	noMain, err := game.LoadDefinitions(worldFS(t, []synth.File{{Path: "data/data.bin", Data: synth.DataBin{}.Bytes()}}))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	partial, err := game.LoadDefinitions(heroNamesFS(t, npcNameEntries("first", "second", "third", "fourth")[:22]))
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	for name, table := range map[string]*mapload.Table{
		"no table":              nil,
		"no name file":          noMain.Table,
		"a name file too short": partial.Table,
	} {
		for _, mage := range []bool{false, true} {
			if got := game.MissionPartyAs(mage, nil, data.BodyList{}, table)[0].Name; got != "Danath" {
				t.Errorf("%s, mage %v: the hero is named %q, want Danath", name, mage, got)
			}
		}
	}
}
