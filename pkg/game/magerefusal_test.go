package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// mageRefusalFront installs a World of the given actors, terrain block bytes
// and relations into castOrderFront's mission. The first actor is the
// fixture's caster made a player's mage of reach 1 with no spells and an
// empty pool, so no heal or cast moves it.
func mageRefusalFront(t *testing.T, block []byte, rel sim.Relations, place func(mage *sim.Entity), others ...sim.Entity) *FrontEnd {
	t.Helper()
	f := castOrderFront(t)
	mage := f.live.world.Entities()[0]
	mage.Owner, mage.Reach, mage.KnownSpells, mage.Mana = sim.SelfSlot, 1, 0, 0
	mage.HealthRegenPeriod, mage.ManaRegenPeriod = 0, 0
	place(&mage)
	w, err := sim.NewStockedSpelledWorld(17, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{Block: block},
		append([]sim.Entity{mage}, others...), nil, rel, nil, nil, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	return f
}

// mageRefusalFoe is a sturdy hostile of owner 2 that never strikes.
func mageRefusalFoe(id sim.EntityID, x, y int32) sim.Entity {
	return sim.Entity{ID: id, Owner: 2, TypeID: 1, X: x, Y: y, HP: 30000, MaxHP: 30000, Capacity: 300, TokenSize: 1, Speed: 10,
		ScanRange: 10, Reach: 1}
}

func mageRefusalRelations() sim.Relations {
	var rel sim.Relations
	rel.Set(1, 2, 1)
	rel.Set(2, 1, 2)
	return rel
}

// mageRefusalOrder is the SAV order block of the Unit-family record whose
// order word +0x0c names the record of health hp, and whether one does.
func mageRefusalOrder(doc sav.DocumentData, hp int16) ([]byte, bool) {
	keys := map[uint32]bool{}
	for _, r := range doc.Objects {
		if r.Class == "Unit" || r.Class == "Human" || r.Class == "Humanoid" {
			if int16(attackTargetWireValue(r, "Health")) == hp {
				keys[attackTargetWireValue(r, "Identity")] = true
				keys[attackTargetWireValue(r, "This")] = true
			}
		}
	}
	delete(keys, 0)
	for _, r := range doc.Objects {
		order := attackTargetWireOrder(r)
		if len(order) == 148 && int16(attackTargetWireValue(r, "Health")) != hp && keys[binary.LittleEndian.Uint32(order[0xc:])] {
			return order, true
		}
	}
	return nil, false
}

func mageRefusalSavedOrder(t *testing.T, w *sim.World, id sim.EntityID) sim.SavedActorOrder {
	t.Helper()
	_, orders, _ := w.SavedGroups()
	for _, o := range orders {
		if o.Entity == id {
			return o
		}
	}
	t.Fatalf("actor %d holds no order block", id)
	return sim.SavedActorOrder{}
}

// A player's staff-less mage withdrawing into a walled pocket has its flee
// refused and idles the pick beside it. SAVE writes the order 0 with the pick
// at +0x0c; the cold LOAD holds the same World and continues with it.
func TestARefusedWithdrawalsIdledPickContinuesUnchangedAfterSAVE(t *testing.T) {
	bounds := castOrderFront(t).live.world.Bounds()
	block := make([]byte, bounds.Width*bounds.Height)
	for y := int32(18); y <= 22; y++ {
		for x := int32(15); x <= 19; x++ {
			block[y*bounds.Width+x] = 1
		}
	}
	f := mageRefusalFront(t, block, mageRefusalRelations(), func(m *sim.Entity) {
		m.ID, m.X, m.Y, m.HP, m.MaxHP, m.Withdraw = 1, 20, 20, 30, 100, 30
	}, mageRefusalFoe(2, 21, 20))
	w := f.live.world
	for k := 0; k < 96 && !pendingVictimEntity(t, w, 1).PursuitIdle; k++ {
		sim.Step(w, nil)
	}
	if e := pendingVictimEntity(t, w, 1); !e.PursuitIdle || e.AttackTarget != 2 {
		t.Fatal("fixture: the refused withdrawal never idled its pick")
	}
	raw, doc, _ := saveCurrentEffect(t, f)
	if order, ok := mageRefusalOrder(doc, 30000); !ok || order[8] != 0 {
		t.Fatalf("SAV order block: found %v, order byte %v; want order 0 with the pick at +0x0c", ok, order)
	}
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "idled pick SAVE cut")
	if o := mageRefusalSavedOrder(t, cold.live.world, 1); o.Raw[8] != 0 {
		t.Fatalf("cold LOAD order byte %d, want 0", o.Raw[8])
	}
	for k := range 200 {
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		assertCurrentWorldEqual(t, w, cold.live.world, "idled pick cold continuation")
		if e := pendingVictimEntity(t, w, 1); e.X != 20 || e.Y != 20 || e.AttackPhase != sim.AttackReady {
			t.Fatalf("tick %d: the mage moved or struck", k)
		}
	}
	if pendingVictimEntity(t, w, 2).HP != 30000 {
		t.Fatal("the mage struck its idled pick")
	}
}

// A player's staff-less mage defending a unit the hostile beside the mage
// does not threaten idles its pick and steps away from the unit it stands
// beside. A SAVE in the step-away continues as the live World does.
func TestAStafflessDefenderSteppingAwayContinuesUnchangedAfterSAVE(t *testing.T) {
	subject := sim.Entity{ID: 1, Owner: 3, TypeID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, Capacity: 300, TokenSize: 1, Speed: 10, ScanRange: 6, Reach: 1}
	f := mageRefusalFront(t, nil, mageRefusalRelations(), func(m *sim.Entity) {
		m.ID, m.X, m.Y = 2, 10, 11
	}, subject, mageRefusalFoe(3, 11, 12))
	w := f.live.world
	sim.Step(w, []sim.Command{sim.GroupDefend(2, 1, 77)})
	for k := 0; k < 60 && !pendingVictimEntity(t, w, 2).HasTarget; k++ {
		sim.Step(w, nil)
	}
	if e := pendingVictimEntity(t, w, 2); !e.HasTarget {
		t.Fatalf("fixture: the defender is not stepping away: at (%d, %d), victim %v idle %v", e.X, e.Y, e.HasAttackTarget, e.PursuitIdle)
	}
	raw, _, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "step-away SAVE cut")
	left := false
	for k := range 200 {
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		assertCurrentWorldEqual(t, w, cold.live.world, "step-away cold continuation")
		e := pendingVictimEntity(t, w, 2)
		if e.HasAttackTarget && !e.PursuitIdle || e.AttackPhase != sim.AttackReady {
			t.Fatalf("tick %d: the staff-less defender engaged", k)
		}
		if max(e.X-10, 10-e.X, e.Y-10, 10-e.Y) >= 2 {
			left = true
		}
	}
	if !left {
		t.Fatal("the defender never stepped away from the unit it defends")
	}
}

// A player's unit that arrived at the end of a group move acquires where it
// stands with no hostile in sight and holds the order 0 the heal arm leaves, not
// the guard order 0xb. SAVE writes the byte and the cold LOAD continues as
// the live World does.
func TestAnIdlePlayerUnitsOrderZeroSurvivesSAVE(t *testing.T) {
	neutral := mageRefusalFoe(2, 30, 30)
	neutral.Owner = 3
	f := mageRefusalFront(t, nil, mageRefusalRelations(), func(m *sim.Entity) {
		m.ID, m.X, m.Y = 1, 10, 10
	}, neutral)
	w := f.live.world
	sim.Step(w, []sim.Command{sim.GroupMoveTo(1, sim.CellPoint{X: 13, Y: 10}, 78)})
	for k := 0; k < 200 && pendingVictimEntity(t, w, 1).X != 13; k++ {
		sim.Step(w, nil)
	}
	for range 40 {
		sim.Step(w, nil)
	}
	if e := pendingVictimEntity(t, w, 1); e.X != 13 || e.Y != 10 || e.HasTarget {
		t.Fatalf("fixture: the unit did not arrive: at (%d, %d), walking %v", e.X, e.Y, e.HasTarget)
	}
	if o := mageRefusalSavedOrder(t, w, 1); o.Raw[8] != 0 {
		t.Fatalf("the idle player's unit holds state %#x order byte %#x, want order byte 0", o.State, o.Raw[8])
	}
	raw, doc, _ := saveCurrentEffect(t, f)
	found := false
	for _, r := range doc.Objects {
		if order := attackTargetWireOrder(r); len(order) == 148 && int16(attackTargetWireValue(r, "Health")) != 30000 {
			found = true
			if order[8] != 0 {
				t.Fatalf("SAV order byte of the player's unit is %#x, want 0", order[8])
			}
		}
	}
	if !found {
		t.Fatal("SAVE holds no order block of the player's unit")
	}
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "idle unit SAVE cut")
	for range 100 {
		sim.Step(w, nil)
		sim.Step(cold.live.world, nil)
		assertCurrentWorldEqual(t, w, cold.live.world, "idle unit cold continuation")
	}
}
