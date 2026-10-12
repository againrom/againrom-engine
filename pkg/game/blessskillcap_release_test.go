package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/sim"
)

// TestReleaseBlessAboveTheTrainingCapSurvivesSAVEAndColdLOAD is a re-derive
// of a hero whose skill sits above the training cap. The hero (actor 237 of
// the owner's mission 151 SAV) fights at skill 110, base 100 plus a worn bonus
// of 10; its source block holds 100, the original's domain. A Bless cast by
// the party's mage re-derives the block. It must stay at 100, and an ordinary
// SAV and a cold LOAD must give the same World hash and the same next 32
// ticks as the uninterrupted run. Before the derive kept the block in that
// domain, the live block held 110 and the cold one 100.
func TestReleaseBlessAboveTheTrainingCapSurvivesSAVEAndColdLOAD(t *testing.T) {
	path := os.Getenv("AGAINROM_SEYDEN_SAV")
	if path == "" {
		t.Skip("set AGAINROM_SEYDEN_SAV to the owner ready-to-deliver mission151 SAV")
	}
	raw, err := os.ReadFile(path)
	quest81OK(t, err)
	const sourceSHA = "a1a1f4a154c7f63c393949e5aba615c9a68782aa1592dde7a70832ddf1baf793"
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != sourceSHA {
		t.Fatal("owner ready-to-deliver SAV SHA differs")
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "statue-ready.sav")
	const caster, hero, bless = sim.EntityID(241), sim.EntityID(237), sim.SpellID(23)
	blockSkill := func(e sim.Entity) uint16 { return binary.LittleEndian.Uint16(e.ActorLoad.Source.Attack[4:]) }
	before, ok := f.live.world.Entity(hero)
	if !ok || before.ActorLoad.Source.Class != 2 || blockSkill(before) != 100 || before.Skill[1] <= 100 || f.live.world.HasEffectSpell(hero, uint16(bless)) {
		t.Fatalf("hero %d is not an unblessed Human above the cap: block %d, skills %v", hero, blockSkill(before), before.Skill)
	}
	f.live.pending = append(f.live.pending, sim.Cast(caster, hero, bless))
	f.LiveAdvance(96)
	after, _ := f.live.world.Entity(hero)
	if !f.live.world.HasEffectSpell(hero, uint16(bless)) || blockSkill(after) != 100 || after.Skill[1] != before.Skill[1] || after.ToHit != before.ToHit {
		t.Fatalf("after Bless: effect %v, block %d, skill %d (was %d), to-hit %d (was %d)", f.live.world.HasEffectSpell(hero, uint16(bless)), blockSkill(after), after.Skill[1], before.Skill[1], after.ToHit, before.ToHit)
	}
	store, _, _ := menuSAVE(t, f, app, OriginalStore{})
	cold, _ := campaignCold(t, store)
	quest81Paired(t, f, cold, 32)
	t.Logf("hero %d: block skill 100, live skill %d and to-hit %d kept under Bless; SAV, cold LOAD and 32 next ticks equal", hero, after.Skill[1], after.ToHit)
}
