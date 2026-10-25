package game

import (
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

// TestReleaseAStaffedMageCastsFromItsWeaponReach: the ogres2 mage's Fire Arrow
// staff is a Staff, reach 5, and Fire Arrow's range is 7. Ordered alone onto
// the Ogre from beyond its reach, the mage begins its cast inside its weapon
// reach and short of touching the Ogre (MAGIC-409).
func TestReleaseAStaffedMageCastsFromItsWeaponReach(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, w := healerOrderSetup(t, false)
	mage, _ := w.Entity(healerMage)
	ogre, _ := w.Entity(healerOgre)
	if mage.Reach != 5 {
		t.Fatalf("the mage's reach is %d, want the Staff's 5", mage.Reach)
	}
	if sim.InReach(mage, ogre) {
		t.Fatal("fixture: the mage starts within its reach of the Ogre")
	}
	f.live.strike(uint32(healerMage), uint32(healerOgre))
	for range 4000 {
		f.live.tick()
		mage, _ = w.Entity(healerMage)
		ogre, _ = w.Entity(healerOgre)
		if mage.AttackPhase != sim.AttackCasting {
			continue
		}
		if !sim.InReach(mage, ogre) {
			t.Fatalf("the mage began its cast at (%d,%d) outside its weapon reach of the Ogre at (%d,%d)", mage.X, mage.Y, ogre.X, ogre.Y)
		}
		touch := mage
		touch.Reach = 1
		if sim.InReach(touch, ogre) {
			t.Fatalf("the mage walked up to touch the Ogre at (%d,%d) before casting", ogre.X, ogre.Y)
		}
		t.Logf("cast begins with the mage at (%d,%d), the Ogre at (%d,%d)", mage.X, mage.Y, ogre.X, ogre.Y)
		return
	}
	t.Fatal("the mage never began a cast")
}
