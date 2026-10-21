package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mod"
)

// The fixture mod of the install-files tests with a setting and a rule set
// from it, run from its folder and from its packed RES on the first game's
// install: the menu, the sound, the briefing, the town square and world
// map frames, the home title, the town and mission SAV bytes with their mod
// mark, and mission 10's World and its hash are equal. A SAV made with the
// folder loads with the packed mod. A session without the mod differs from
// the folder's, so the comparison can fail; a packed reader that skipped
// files/ would lose the mod's pictures, sound and texts.
func TestReleasePackedModPlaysAsItsFolder(t *testing.T) {
	f, slot := modFilesInstall(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the root holds the second game")
	}
	const id = "packed-art"
	folders, packedMods := t.TempDir(), t.TempDir()
	writeInstallFilesMod(t, f, slot, folders, id, "", "def init(game, settings):\n    game.rules.skill_cap = settings.cap\n")
	settings := "[cap]\nlabel = \"Cap\"\ntype = \"int\"\ndefault = 120\nmin = 100\nmax = 150\n"
	if err := os.WriteFile(filepath.Join(folders, id, mod.SettingsName), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	m, packed, err := mod.Pack(filepath.Join(folders, id))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packedMods, mod.PackedName(m.ID)), packed, 0o644); err != nil {
		t.Fatal(err)
	}
	baseID := BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS")))
	fromFolder := runTownModsFor(t, baseID, folders, id)
	fromRES := runTownModsFor(t, baseID, packedMods, id)
	if fromRES.Ordered[0].Dir != filepath.Join(packedMods, mod.PackedName(id)) {
		t.Fatalf("the packed run read %s", fromRES.Ordered[0].Dir)
	}
	if fromFolder.Set.Canonical() != fromRES.Set.Canonical() || len(fromRES.Files.Files) != 5 || fromRES.Rules.SkillCap() != 120 {
		t.Fatalf("mod sets\n%s\n%s\nfiles %d, skill cap %d", fromFolder.Set.Canonical(), fromRES.Set.Canonical(), len(fromRES.Files.Files), fromRES.Rules.SkillCap())
	}

	folder := modFilesSession(t, &fromFolder, slot)
	if diff := diffModFilesRuns(modFilesSession(t, nil, slot), folder); !slices.Contains(diff, "menu picture") || !slices.Contains(diff, "home title") {
		t.Fatalf("the mod changes only %v; the comparison would prove nothing", diff)
	}
	res := modFilesSession(t, &fromRES, slot)
	if diff := diffModFilesRuns(folder, res); len(diff) != 0 {
		t.Fatalf("the packed mod differs from its folder in %v", diff)
	}
	for _, raw := range [][]byte{res.townSAV, res.missionSAV} {
		if _, ids := withoutModMark(t, raw); !slices.Equal(ids, []string{id}) {
			t.Errorf("a SAV made with the packed mod carries the mark %v", ids)
		}
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "town.sav"), folder.townSAV, 0o644); err != nil {
		t.Fatal(err)
	}
	g := modFilesFront(t, &fromRES)
	_, _, load := g.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	if _, town, err := load("town.sav"); err != nil || !town {
		t.Fatalf("the folder's town SAV under the packed mod: town %v, %v", town, err)
	}
	t.Logf("%s: mod set %s, %d install files", baseID, fromRES.Set.Digest(), len(fromRES.Files.Files))
}
