package game

import (
	"os"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondGameRoot returns the asset root when it holds the second game, and
// skips for a root that holds the first.
func secondGameRoot(t *testing.T) *Archives {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the second game's mission needs a lawful install")
	}
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	if archives.Game() != base.GameROM2 {
		t.Skip("the root holds the first game")
	}
	return archives
}

// TestReleaseSecondGameMissionTenOpens runs once against each second-game root.
func TestReleaseSecondGameMissionTenOpens(t *testing.T) {
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	ms, err := StartMission(archives.Containers, 10, defs.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	if ms.Map.Width != 80 || ms.Map.Height != 80 {
		t.Errorf("map is %dx%d, want 80x80", ms.Map.Width, ms.Map.Height)
	}
	if len(ms.Map.Units) != 29 || len(ms.Map.Objects) != 16 {
		t.Errorf("map has %d units and %d objects, want 29 and 16", len(ms.Map.Units), len(ms.Map.Objects))
	}
	w := ms.World
	if got, want := len(w.Entities()), 29+len(party); got != want {
		t.Errorf("world has %d entities, want %d", got, want)
	}
	gaps := w.Script().Unsupported()
	if len(gaps) != 0 {
		t.Errorf("script names unsupported arms: %v", gaps)
	}
	for i := 0; i < 300; i++ {
		sim.Step(w, nil)
	}
	if w.Tick() != 300 || w.Outcome() != sim.OutcomeUndecided {
		t.Errorf("after 300 ticks: tick %d outcome %v", w.Tick(), w.Outcome())
	}
}

// TestReleaseSecondGameCampaignMapsDecode opens every scenario map the root
// carries.
func TestReleaseSecondGameCampaignMapsDecode(t *testing.T) {
	archives := secondGameRoot(t)
	defs, err := LoadDefinitionsFor(archives.Containers, archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	opened := 0
	for n := 1; n <= 300; n++ {
		addr, _ := MissionMap(n)
		if _, err := archives.Containers.ReadFile(addr); err != nil {
			continue
		}
		if _, err := StartMission(archives.Containers, n, defs.Table, mapload.DifficultyNormal, nil); err != nil {
			t.Errorf("mission %d: %v", n, err)
		}
		opened++
	}
	if opened != 46 {
		t.Errorf("%d scenario maps are present, want 46", opened)
	}
	t.Logf("%d scenario maps opened", opened)
}
