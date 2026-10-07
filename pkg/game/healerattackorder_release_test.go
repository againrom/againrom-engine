package game

import (
	"os"
	"testing"

	"againrom/pkg/sim"
)

// The ogres2 save is mission 60 with the participant's warrior (86), archer
// (84) and mage (85) standing together. The mage wears a Fire Arrow staff; the
// witness takes it off so the mage has nothing to strike with, wounds the
// warrior, and presses Attack over all three on the first Ogre through the
// map's own attack seam, the one the pointer order calls once per selected unit.
func healerOrderSetup(t *testing.T, takeStaff bool) (*FrontEnd, *sim.World) {
	t.Helper()
	path := os.Getenv("AGAINROM_OGRES2_SAV")
	if path == "" {
		t.Skip("AGAINROM_OGRES2_SAV is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "ogres2.sav")
	t.Cleanup(app.StopAudio)
	w, _ := f.LiveWorld()
	if f.liveMission != turtleMissionNumber {
		t.Fatal("source save is not mission 60", f.liveMission)
	}
	if takeStaff {
		for slot := 1; slot <= sim.EquipSlots; slot++ {
			f.live.pending = append(f.live.pending, sim.Unequip(healerMage, sim.EquipSlot(slot)))
		}
	}
	f.live.pending = append(f.live.pending, sim.Damage(healerWarrior, 120))
	ogreStep(t, f, app)
	mage, _ := w.Entity(healerMage)
	if takeStaff && mage.WeaponSpell != 0 {
		t.Fatal("the mage still holds a weapon spell", mage.WeaponSpell)
	}
	if !takeStaff && mage.WeaponSpell == 0 {
		t.Fatal("the mage holds no weapon spell in the save")
	}
	return f, w
}

const (
	healerArcher  = sim.EntityID(84)
	healerMage    = sim.EntityID(85)
	healerWarrior = sim.EntityID(86)
	healerOgre    = sim.EntityID(42)
)

func healerAttack(t *testing.T, f *FrontEnd, w *sim.World, ids ...sim.EntityID) int32 {
	t.Helper()
	hp := int32(0)
	if e, ok := w.Entity(healerWarrior); ok {
		hp = e.HP
	}
	for _, id := range ids {
		f.live.strike(uint32(id), uint32(healerOgre))
	}
	for range 90 {
		f.live.tick()
	}
	return hp
}

func healerHolds(w *sim.World, id sim.EntityID) bool {
	e, _ := w.Entity(id)
	return e.HasAttackTarget && e.AttackTarget == healerOgre
}

func TestReleaseGroupAttackKeepsAStafflessMageHealing(t *testing.T) {
	f, w := healerOrderSetup(t, true)
	hurt := healerAttack(t, f, w, healerArcher, healerMage, healerWarrior)
	if !healerHolds(w, healerArcher) || !healerHolds(w, healerWarrior) {
		t.Fatal("the archer and the warrior did not take the group's attack order")
	}
	if healerHolds(w, healerMage) {
		t.Fatal("the staffless mage took the group's attack order")
	}
	if e, _ := w.Entity(healerWarrior); e.HP <= hurt {
		t.Fatalf("the warrior's health did not rise from %d (now %d): the mage did not heal", hurt, e.HP)
	}
}

func TestReleaseAStafflessMageOrderedAloneObeys(t *testing.T) {
	f, w := healerOrderSetup(t, true)
	healerAttack(t, f, w, healerMage)
	if !healerHolds(w, healerMage) {
		t.Fatal("the staffless mage ordered alone did not take its attack order")
	}
}

func TestReleaseAStaffedMageTakesAGroupAttackOrder(t *testing.T) {
	f, w := healerOrderSetup(t, false)
	healerAttack(t, f, w, healerArcher, healerMage, healerWarrior)
	if !healerHolds(w, healerMage) {
		t.Fatal("the mage with a staff did not take the group's attack order")
	}
}
