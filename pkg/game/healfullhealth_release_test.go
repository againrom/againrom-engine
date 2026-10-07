package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseManualHealAtFullHealthHeroIsCast(t *testing.T) {
	const healID = 6
	f := releaseFront(t)
	f.SetTipsOff(true)
	var caster sim.Entity
	for _, choices := range [][]int{{1, 1, 3}, {1, 0, 3}, {0, 1, 3}, {1, 2, 3}} {
		party := f.ChargenParty(ui.ChargenResult{Name: "Heal witness", Choices: choices, Stats: []int{31, 27, 24, 29}})
		a := f.App("heal full health")
		a.SetCutscenes(nil)
		if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
			t.Fatal(err)
		}
		a.Layout(640, 480)
		caster = sim.Entity{}
		for _, e := range f.live.world.Entities() {
			if e.Owner == sim.SelfSlot && e.MaxMana > 0 && e.KnownSpells&(1<<healID) != 0 && e.Mana > 0 && e.HP == e.MaxHP && e.HP > 0 {
				caster = e
				break
			}
		}
		if caster.ID != 0 {
			break
		}
	}
	if caster.ID == 0 {
		t.Fatal("no party mage knowing Heal at full health and with mana")
	}
	w := f.live.world
	if why := w.BookSpellRefusal(caster.ID, caster.ID, healID); why != "" {
		t.Fatalf("Heal at a full-health hero refused: %s", why)
	}
	f.live.attackOrCast(uint32(caster.ID), uint32(caster.ID), healID, 0, 0, false)
	if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindCast {
		t.Fatal("Heal order not queued")
	}
	for range 60 {
		f.live.tick()
	}
	var after sim.Entity
	for _, e := range w.Entities() {
		if e.ID == caster.ID {
			after = e
		}
	}
	if after.Mana >= caster.Mana {
		t.Fatalf("mana %d -> %d: the cast was not paid", caster.Mana, after.Mana)
	}
	if after.HP != after.MaxHP || after.MaxHP < caster.MaxHP {
		t.Fatalf("health %d/%d after the cast, want the maximum, at least %d", after.HP, after.MaxHP, caster.MaxHP)
	}
}
