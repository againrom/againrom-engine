package sim

import "testing"

// summonedKill is a second-game World where Summon built one creature, given
// to a computer player and the gold row a death rolls; a human attacker (the
// caster, moved to another owner) is about to kill it.
func summonedKill(t *testing.T) *World {
	t.Helper()
	w := summonedWorld(t, 0x25, 1)
	w.entities[1].GoldChance, w.entities[1].TreasureMin, w.entities[1].TreasureMax = 100, 8, 8
	w.entities[1].Owner = 2
	w.entities[1].TypeID = 0x41
	w.carried[1] = []ItemStack{StackItem(ItemInstance{Code: 0x101, Kind: 1, Price: 27}, 1)}
	a := &w.entities[0]
	a.Humanoid, a.GainsXP, a.XPSlot, a.Owner = true, true, 1, 8
	a.TypeID, a.MaxMana, a.KnownSpells = HumanTypeID, 0, 0
	return w
}

func TestSecondSummonedCreatureCarriesTheMarkThatPaysNothing(t *testing.T) {
	w := summonedKill(t)
	if !w.entities[1].SuppressRewards {
		t.Fatal("Summon's creature carries no summoned mark (R2-ENGINE-455, R2-ENGINE-535)")
	}
	w.payExperience(0, 1, 60, true)
	if w.entities[0].SkillXP != ([skillSlots]int32{}) {
		t.Fatalf("human attacker earned %v from a summoned creature", w.entities[0].SkillXP)
	}
	if w.hasTerminalLoot(1) {
		t.Fatal("summoned creature reports terminal loot")
	}
	w.entities[1].setCurrentHealth(-50)
	w.clearFelled(1)
	w.dropTerminalLoot(1)
	if len(w.sacks) != 0 {
		t.Fatalf("summoned creature dropped %d sacks", len(w.sacks))
	}
	cold := cloneWorld(t, w)
	if !cold.entities[1].SuppressRewards || cold.Hash() != w.Hash() {
		t.Fatal("the summoned mark did not survive SAVE and LOAD")
	}
}

func TestSecondSummonedCreatureWithoutTheMarkPaysAndDrops(t *testing.T) {
	w := summonedKill(t)
	w.entities[1].SuppressRewards = false
	w.payExperience(0, 1, 60, true)
	if w.entities[0].SkillXP == ([skillSlots]int32{}) {
		t.Fatal("unmarked creature paid no experience: the control is not a control")
	}
	if !w.hasTerminalLoot(1) {
		t.Fatal("unmarked creature reports no terminal loot")
	}
	w.entities[1].setCurrentHealth(-50)
	w.clearFelled(1)
	w.dropTerminalLoot(1)
	if len(w.sacks) == 0 {
		t.Fatal("unmarked creature dropped nothing")
	}
}
