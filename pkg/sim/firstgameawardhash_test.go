package sim

import "testing"

// firstGameAwardFighter is a first-game fighter at Mind 24 trained in blades
// at 20 and skill 5 at 10, as the first game's producers build him.
func firstGameAwardFighter() Entity {
	e := cbEnt(1, 0, 0)
	e.Owner, e.GainsXP, e.Mind, e.XPSlot, e.TypeID, e.Humanoid = SelfSlot, true, 24, 1, HumanTypeID, true
	levels := [skillSlots]int32{0, 20, 0, 0, 0, 10}
	e.NativeClass = NativeClass{Present: true, Fighter: true}
	e.NativeTraining = NativeTraining{Present: true, Levels: levels}
	e.Skill = levels
	for s, level := range levels {
		e.SkillXP[s] = awardXP(level)
	}
	return e
}

// The first game's award for a blow and for kills of two creatures of one
// row leaves the World at its pinned hash: its actors hold no kind word or
// main skill, so its byte form gains no section.
func TestFirstGameAwardsKeepTheirWorldHash(t *testing.T) {
	second := awardVictim(3)
	second.ID, second.X = 3, 2
	w := cbWorld(t, 1, firstGameAwardFighter(), awardVictim(3), second)
	w.game = newGameProfile(ScriptROM1)
	w.awardSkill(0, 0, 50, 1)
	for ti := 1; ti <= 2; ti++ {
		v := &w.entities[ti]
		v.KillCreditSource, v.HasKillCredit = w.entities[0].ID, true
		w.processKillCredit(ti)
	}
	t.Logf("skill experience %v, hash %#016x", w.entities[0].SkillXP, w.Hash())
	if got := w.Hash(); got != firstGameAwardHash {
		t.Errorf("the first game's awards hash to %#016x, want %#016x", got, firstGameAwardHash)
	}
}

const firstGameAwardHash = 0x5f76d79a89621ea4
