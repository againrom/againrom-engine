package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Installed map/rules and the ordinary pointer-to-command/session tick path.
// A controlled single mage excludes combat, scripts and inherited pool timers.
func TestReleaseIdleRegenerationAfterManualCast(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[5] = 100
	party := []mapload.PartyMember{{ID: "mage", PlayerCharacter: true, StartingHero: true, Mage: true, Class: 0x18,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << 26,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000}}}
	a := f.App("idle regeneration")
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
	actor.HP, actor.MaxHP, actor.HealthRegenPeriod, actor.HealthHundredths = 10, 100, 100, 0
	actor.Mana, actor.MaxMana, actor.ManaRegenPeriod, actor.ManaHundredths = 1000, 1000, 1000, 0
	actor.AttackCharge, actor.AttackRelax, actor.RotationSpeed, actor.CastWait = 8, 4, 0, 0
	actor.ActionClock = sim.ActionClock{}
	m := live.mission.state.Map
	w, err := sim.NewStructuredWorld(1, live.world.Bounds(), sim.ModeCanonical,
		sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)},
		[]sim.Entity{actor}, nil, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, nil)
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
	entryPointer1084(t, a, 30, 50)
	if live.world.Tick() != 0 || len(live.pending) != 1 {
		t.Fatal("fixture advanced before cast")
	}
	var afterCastMana int32
	for live.world.Tick() < 260 {
		live.tick()
		e, _ := live.entity(id)
		if live.world.Tick() == 9 {
			if e.X != 30 || e.Y != 50 || e.Mana >= 1000 || e.ActionClock.End != 12 {
				t.Fatalf("ordinary Teleport did not create the 8+4 action deadline: at%d,%d mana%d end%d", e.X, e.Y, e.Mana, e.ActionClock.End)
			}
			afterCastMana = e.Mana
		}
	}
	e, _ := live.entity(id)
	// Health ticks12/76 add2, ticks140/204 add6. Mana's six slots12..92
	// add1 each; its ten slots108..252 add3 each, after the cast spent mana.
	if afterCastMana == 0 || e.HP != 26 || e.Mana != afterCastMana+36 {
		t.Fatalf("idle pools HP%d mana%d, want26 and%d", e.HP, e.Mana, afterCastMana+36)
	}
	if _, spell, armed := live.view.QuickSpellState(); spell != 26 || !armed {
		t.Fatal("idle wait lost cast cursor")
	}
	t.Log("pointer Teleport at tick0; end12; HP10->26; mana+36 over260 ticks")
}
