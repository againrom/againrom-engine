package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseHealAtAnEnemyIsCastPaidAndChangesNoHealth casts the installed Heal
// at a wounded hostile creature of mission 10. The cast is admitted, the
// caster pays the installed cost, the creature carries the cast mark, and its
// health never rises.
func TestReleaseHealAtAnEnemyIsCastPaidAndChangesNoHealth(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Heal witness", Choices: []int{0, 1, 0}, Stats: []int{20, 25, 40, 40}})
	party[0].KnownSpells |= 1 << 6
	f.Carried = party
	if err := f.App("heal enemy").OpenMission(f.MissionOpenerWith(10, f.Carried)); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	rule, ok := w.Spell(6)
	if !ok || !rule.Restorative {
		t.Fatalf("installed spell 6 is not restorative: %+v", rule)
	}
	var mage, enemy sim.Entity
	for _, e := range w.Entities() {
		switch {
		case mage.ID == 0 && e.Owner == sim.SelfSlot && e.Humanoid:
			mage = e
		case enemy.ID == 0 && e.Owner != sim.SelfSlot && e.Alive() && e.MaxHP > 0 && w.Relations().Hostile(sim.SelfSlot, e.Owner):
			enemy = e
		}
	}
	if mage.ID == 0 || enemy.ID == 0 {
		t.Fatalf("mission 10 has no mage %d or hostile unit %d", mage.ID, enemy.ID)
	}
	placed := false
	for _, d := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if w.HeadlessPlace(enemy.ID, mage.X+d[0], mage.Y+d[1]) == nil {
			placed = true
			break
		}
	}
	if !placed {
		t.Fatal("no free cell beside the mage")
	}
	if err := w.HeadlessDamage(enemy.ID, enemy.MaxHP/2); err != nil {
		t.Fatal(err)
	}
	hurt, _ := liveEntity(f, enemy.ID)
	if hurt.HP >= hurt.MaxHP {
		t.Fatalf("the enemy holds %d of %d health after damage", hurt.HP, hurt.MaxHP)
	}
	before, _ := liveEntity(f, mage.ID)
	if before.Mana < int32(rule.ManaCost) {
		t.Fatalf("the mage holds %d mana, the cast costs %d", before.Mana, rule.ManaCost)
	}

	f.live.pending = append(f.live.pending, sim.Cast(mage.ID, enemy.ID, 6))
	marked, paid := false, false
	for range 64 {
		f.live.tick()
		e, ok := liveEntity(f, enemy.ID)
		if !ok {
			t.Fatal("the enemy left the world")
		}
		if e.HP > hurt.HP {
			t.Fatalf("the enemy health rose from %d to %d", hurt.HP, e.HP)
		}
		if m, _ := liveEntity(f, mage.ID); m.Mana < before.Mana {
			paid = true
		}
		if e.SpellFX > 0 && e.SpellFXSpell == 6 {
			marked = true
			break
		}
	}
	if !paid {
		t.Error("the cast cost the mage no mana")
	}
	if !marked {
		t.Error("the enemy never carried the Heal mark: the cast was not admitted")
	}
}

// The campaign mission whose wounded Ogre lies at zero health under a trigger
// that waits for its health to rise belongs to an owner that is not hostile to
// the party, so its Heal is applied.
func TestReleaseHealRaisesTheDyingOgreOfMission81(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Heal witness", Choices: []int{0, 1, 0}, Stats: []int{20, 25, 40, 40}})
	party[0].KnownSpells |= 1 << 6
	f.Carried = party
	if err := f.App("heal ogre").OpenMission(f.MissionOpenerWith(81, f.Carried)); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	const ogre = sim.EntityID(17)
	o, ok := w.Entity(ogre)
	if !ok || o.HP != 0 || o.MaxHP <= 0 {
		t.Fatalf("mission 81 unit 17 is %+v, want a unit at zero health", o)
	}
	if w.Relations().Hostile(sim.SelfSlot, o.Owner) {
		t.Fatalf("the Ogre's owner %d is hostile to the party", o.Owner)
	}
	var mage sim.Entity
	for _, e := range w.Entities() {
		if e.Owner == sim.SelfSlot && e.Humanoid {
			mage = e
			break
		}
	}
	if mage.ID == 0 {
		t.Fatal("mission 81 has no party mage")
	}
	placed := false
	for _, d := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {2, 0}, {-2, 0}} {
		if w.HeadlessPlace(mage.ID, o.X+d[0], o.Y+d[1]) == nil {
			placed = true
			break
		}
	}
	if !placed {
		t.Fatal("no free cell beside the Ogre")
	}
	f.live.pending = append(f.live.pending, sim.Cast(mage.ID, ogre, 6))
	for range 64 {
		f.live.tick()
		if e, _ := liveEntity(f, ogre); e.HP > 0 {
			return
		}
	}
	t.Fatal("the Ogre's health did not rise above zero")
}
