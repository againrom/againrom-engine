package game

import (
	"testing"

	"againrom/pkg/sim"
)

// TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD: a
// generated mage hero enters campaign mission 10, learns every spell by the
// unlocked #modify cheat, casts Summon from the spell bar at itself
// (R2-ENGINE-455) and then Stone Curse at the creature it made
// (R2-ENGINE-025, R2-ENGINE-454). A named SAV and a cold LOAD give an equal
// World, the creature and the hold included, and the next ticks stay equal.
func TestReleaseSecondCampaignStoneCurseAndSummonSurviveSAVEAndColdLOAD(t *testing.T) {
	f, app := secondGeneratedMage(t, false)
	for _, target := range []string{"TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	for app.HeadlessActivate("notice") == nil {
	}
	for _, target := range []string{"EXIT", "GATES", "mission 10"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
	app.Layout(1024, 768)
	f.live.stopped = true
	mw := f.live
	id := mw.mission.ids[0]
	secondCheatUnlock(t, f, app)
	secondCheatWantReply(t, f, secondCheatLines(t, f, app, "#modify self +spells"), 7, sim.SelfSlot)
	// Automatic casting at a level with no target class casts nothing, so the
	// waits below refill mana for the two casts. Off is not used: a second-game
	// Player set to Off reads the default Average again (DIV-3007).
	mw.pending = append(mw.pending, sim.SetPlayerParameter(sim.SelfSlot, sim.PlayerParameterAutoCast, int32(sim.AutoCastMinimum)))
	hero := releaseEntity(t, mw, id)
	t.Logf("hero %d: mana %d of %d, Astral skill %d, known %#x", id, hero.Mana, hero.MaxMana, hero.Skill[5], hero.KnownSpells)
	for n := 0; n < 80 && releaseEntity(t, mw, id).Mana < min(releaseEntity(t, mw, id).MaxMana, 160); n++ {
		f.LiveAdvance(100)
		if err := mw.world.HeadlessHeal(id); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("after the wait: tick %d, mana %d of %d, regen %d/%d, autocast %#x", mw.world.Tick(), releaseEntity(t, mw, id).Mana, releaseEntity(t, mw, id).MaxMana, releaseEntity(t, mw, id).ManaRegeneration, releaseEntity(t, mw, id).ManaRegenPeriod, mw.world.AutoCast(sim.SelfSlot))
	known := map[sim.EntityID]bool{}
	for _, e := range mw.world.Entities() {
		known[e.ID] = true
	}

	t.Logf("Summon admission: %q, Stone Curse admission at the hero: %q", mw.world.BookSpellRefusal(id, id, 25), mw.world.BookSpellRefusal(id, id, 18))
	mw.pending = append(mw.pending, sim.Cast(id, id, 25))
	f.LiveAdvance(60)
	var creature sim.Entity
	for _, e := range mw.world.Entities() {
		if !known[e.ID] {
			creature = e
		}
	}
	hero = releaseEntity(t, mw, id)
	if creature.ID == 0 || creature.Owner != hero.Owner || creature.Group == hero.Group || creature.Group == 0 {
		t.Fatalf("Summon made actor %d owner %d group %d for hero owner %d group %d", creature.ID, creature.Owner, creature.Group, hero.Owner, hero.Group)
	}
	t.Logf("summoned actor %d type %d at %d,%d (%+d,%+d from the hero), group %d; hero mana %d", creature.ID, creature.TypeID, creature.X, creature.Y, creature.X-hero.X, creature.Y-hero.Y, creature.Group, hero.Mana)

	for try := 0; try < 4 && !mw.world.HasEffectArm(creature.ID, 20); try++ {
		// Tick by tick, so the cast is queued on the first tick the mana
		// allows it, before a support cast can spend it.
		for n := 0; n < 20000 && mw.world.BookSpellRefusal(id, creature.ID, 18) != ""; n++ {
			f.LiveAdvance(1)
		}
		h := releaseEntity(t, mw, id)
		t.Logf("try %d: tick %d, hero hp %d/%d mana %d/%d, cast wait %d, admission %q", try, mw.world.Tick(), h.HP, h.MaxHP, h.Mana, h.MaxMana, h.CastWait, mw.world.BookSpellRefusal(id, creature.ID, 18))
		mw.pending = append(mw.pending, sim.Cast(id, creature.ID, 18))
		f.LiveAdvance(60)
	}
	var curse sim.ActiveEffect
	for _, e := range mw.world.ActiveEffects() {
		if e.Target == creature.ID && e.Spell == 18 {
			curse = e
		}
	}
	if curse.Spell != 18 || curse.Kind != sim.EffectNone || curse.Remaining == 0 {
		t.Fatalf("Stone Curse on the creature: %+v", mw.world.ActiveEffects())
	}
	t.Logf("Stone Curse on actor %d: remaining %d", creature.ID, curse.Remaining)

	out := secondMissionSaveDirectory(t)
	secondMissionNamedSave(t, f, app, out, "Spell rows")
	cold, _ := secondMissionColdAt(t, out, secondSaveFile(t, out, "Spell rows"), 10)
	cold.live.stopped = true
	assertCurrentWorldEqual(t, mw.world, cold.live.world, "cold LOAD")
	if !cold.live.world.HasEffectArm(creature.ID, 20) {
		t.Fatal("cold LOAD lost the hold")
	}
	for range 8 {
		f.LiveAdvance(1)
		cold.LiveAdvance(1)
		assertCurrentWorldEqual(t, mw.world, cold.live.world, "next tick")
	}
	t.Logf("SAV, cold LOAD and 8 next ticks equal; World hash %#x", mw.world.Hash())
}
