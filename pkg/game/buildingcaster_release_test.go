package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The second game's mission 72, the installed map: a hero who walks from the
// start to within reach of the tower at (22,111) receives its spell 10. The map
// holds 27 towers' records, which the world holds as 27 casters.
func TestReleaseSecondMissionTowersCastAtTheParty(t *testing.T) {
	f := secondGameFront(t)
	defs, err := LoadDefinitionsFor(f.Archives.Containers, f.Archives.Game())
	if err != nil {
		t.Fatal(err)
	}
	party := MissionPartyAs(false, defs.StartWeapon, defs.Bodies, defs.Table)
	ms, err := StartMission(f.Archives.Containers, 72, defs.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	w := ms.World
	if got := len(w.BuildingCasters()); got != 27 {
		t.Fatalf("mission 72 holds %d casters, want 27", got)
	}
	hero := ms.Start.IDs[0]
	sim.Step(w, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: 23, Y: 109})})
	for range 400 {
		sim.Step(w, nil)
		for _, c := range w.ScriptCasts() {
			if c.FromX == 22 && c.FromY == 111 && c.Spell == 10 && c.Target == hero {
				return
			}
		}
	}
	t.Fatal("the tower at (22,111) did not cast at the hero within 400 ticks")
}
