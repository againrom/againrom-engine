package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// mageDefendFront is a wounded hero of the player at (10, 10), a player's
// staff-less mage with Heal at (10, 12) defending it, and a hostile beside
// the hero striking it.
func mageDefendFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := castOrderFront(t)
	actors := f.live.world.Entities()
	hero := sim.Entity{ID: 1, Owner: sim.SelfSlot, TypeID: 1, X: 10, Y: 10, HP: 40, MaxHP: 100, Capacity: 300, TokenSize: 1, Speed: 10, ScanRange: 6, Reach: 1}
	mage := actors[0]
	mage.ID, mage.X, mage.Y, mage.Reach, mage.KnownSpells = 2, 10, 12, 1, 1<<6
	mage.HealthRegenPeriod, mage.ManaRegenPeriod = 0, 0
	foe := sim.Entity{ID: 3, Owner: 2, TypeID: 1, X: 11, Y: 10, HP: 30000, MaxHP: 30000, Capacity: 300, TokenSize: 1, Speed: 10,
		ScanRange: 10, Reach: 1, DamageBase: 1, AlwaysHits: true, AttackCharge: 6, AttackRelax: 4, HasAttackTarget: true, AttackTarget: 1}
	var rel sim.Relations
	rel.Set(1, 2, 1)
	rel.Set(2, 1, 2)
	w, err := sim.NewStockedSpelledWorld(17, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, []sim.Entity{hero, mage, foe}, nil, rel, nil, nil, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	sim.Step(w, []sim.Command{sim.GroupDefend(2, 1, 77)})
	return f
}

func mageMelees(e sim.Entity) bool {
	return e.HasAttackTarget && e.AttackTarget == 3 && !e.PursuitIdle || e.AttackPhase != sim.AttackReady
}

func TestADefendingStafflessMageContinuesUnchangedAfterSAVE(t *testing.T) {
	f := mageDefendFront(t)
	w := f.live.world
	for range 4 {
		sim.Step(w, nil)
	}
	raw, _, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "defending mage SAVE cut")
	healed := false
	for k := range 300 {
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		assertCurrentWorldEqual(t, w, cold.live.world, "defending mage cold continuation")
		if mageMelees(pendingVictimEntity(t, w, 2)) {
			t.Fatalf("tick %d: the staff-less defending mage went into melee", k)
		}
		for _, b := range w.Actions().Books {
			if b.Caster == 2 && b.Target == 1 {
				healed = true
			}
		}
	}
	if !healed {
		t.Fatal("the defending mage never healed the wounded hero")
	}
}
