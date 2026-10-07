package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Installed terrain, spell rules and book input; one controlled mage isolates
// repeated Teleport from campaign acquisition, AI and mission notice timing.
func TestReleaseRepeatedTeleportKeepsCastMode(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[5] = 100
	party := []mapload.PartyMember{{ID: "mage", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << 26,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}}
	a := f.App("repeated Teleport")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	id := live.mission.ids[0]
	actor, ok := live.entity(id)
	if !ok {
		t.Fatal("missing mage")
	}
	actor.X, actor.Y = 29, 50
	// A previous action's long recovery must not discard the first manual click.
	actor.CastWait = 200
	m := live.mission.state.Map
	w, err := sim.NewStructuredWorld(1, live.world.Bounds(), sim.ModeCanonical,
		sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)},
		[]sim.Entity{actor}, nil, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, live.world.Structures())
	if err != nil {
		t.Fatal(err)
	}
	live.world = w
	live.push()
	inspectionCentre(live, 29, 50)
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	live.push()
	if _, _, err := a.HeadlessSpellPoint(26); err != nil {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	sx, sy, err := a.HeadlessSpellPoint(26)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, sx, sy); err != nil {
			t.Fatal(err)
		}
	}
	for n, cell := range [][2]int{{30, 50}, {29, 50}} {
		inspectionCentre(live, 29, 50)
		entryPointer1084(t, a, cell[0], cell[1])
		if got := live.pending; len(got) != 1 || got[0].Kind != sim.KindCastAt || got[0].Spell != 26 {
			t.Fatalf("cast%d queued %v, want Teleport without another book click", n+1, got)
		}
		before, _ := live.entity(id)
		arrived := false
		for tick := 0; tick < 200; tick++ {
			live.tick()
			after, _ := live.entity(id)
			if after.X == int32(cell[0]) && after.Y == int32(cell[1]) && after.Mana < before.Mana {
				arrived = true
				break // The next manual Teleport arrives during this one's recovery.
			}
		}
		if !arrived {
			after, _ := live.entity(id)
			t.Fatalf("cast%d: no paid arrival at %v; position%d,%d mana%d->%d", n+1, cell, after.X, after.Y, before.Mana, after.Mana)
		}
		if live.world.Tick()%fogPeriod == 0 {
			t.Fatal("arrival fixture hit a periodic fog refresh")
		}
		for i, lit := range live.world.Sight(sim.SelfSlot) {
			if (live.fog.visible[i] != 0) != (lit != 0) || (lit != 0 && live.fog.explored[i] == 0) {
				t.Fatalf("cast%d arrived at tick%d before cell%d visibility/exploration refreshed", n+1, live.world.Tick(), i)
			}
		}
		if _, current, armed := live.view.QuickSpellState(); current != 26 || !armed {
			t.Fatalf("cast%d completed with current%d armed%v", n+1, current, armed)
		}
	}
	gx, gy, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"right-press", "right-release"} {
		if err := a.HeadlessPointer(edge, gx, gy); err != nil {
			t.Fatal(err)
		}
	}
	if _, current, armed := live.view.QuickSpellState(); current != 0 || armed {
		t.Fatal("right click did not cancel the persistent cast mode")
	}
	if selected, ok := live.view.SelectedUnit(); !ok || selected != uint32(id) {
		t.Fatal("cancelling cast deselected the mage")
	}
	t.Log("one book click, two pointer Teleports interrupting recovery, then right-click cancellation")
}
