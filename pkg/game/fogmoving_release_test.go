package game

import (
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A party member walking on an installed mission map lights the ground it
// steps onto on the tick it steps, and the ground it left stays lit until the
// period turns over.
func TestReleaseWalkingPartyMemberLightsNewGroundAtOnce(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 60, Spirit: 60}
	party := []mapload.PartyMember{{ID: "walker", PlayerCharacter: true, StartingHero: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100}}}
	a := f.App("fog stamp walk")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	id := live.mission.ids[0]
	start, ok := live.entity(id)
	if !ok {
		t.Fatal("missing party member")
	}
	if live.world.Tick()%fogPeriod != 0 {
		t.Fatal("mission did not open on a period boundary")
	}
	opening := append([]byte(nil), live.fog.visible...)
	live.pending = append(live.pending, sim.MoveTo(id, sim.CellPoint{X: start.X + 5, Y: start.Y}))
	moved := 0
	last := start.X
	for n := 1; n < fogPeriod; n++ {
		live.tick()
		if e, _ := live.entity(id); e.X != last {
			last = e.X
			moved++
		}
		if i := visibleMisses(live); i >= 0 {
			t.Fatalf("world tick %d: cell %d is in the walker's sight and not in the visible layer", live.world.Tick(), i)
		}
		for i, v := range opening {
			if v != 0 && live.fog.visible[i] == 0 {
				t.Fatalf("world tick %d: ground left behind went dark before the period", live.world.Tick())
			}
		}
	}
	if moved == 0 {
		t.Fatal("the party member did not walk")
	}
}
