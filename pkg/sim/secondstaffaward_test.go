package sim

import "testing"

func secondStaffWorld(t *testing.T, victimHP int32, god bool) (*World, SpellRule) {
	t.Helper()
	rule := wpnRule(20, 20, 5)
	rule.School = 1
	rule = secondRow(rule)
	caster := itemTrainingCaster(rule.ID, int(rule.School))
	caster.NativeClass = NativeClass{Present: true}
	caster.Owner = SelfSlot
	victim := killCreditVictim(2, victimHP)
	victim.MaxHP = 100
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)
	w.game = newGameProfile(ScriptROM2)
	w.rom2 = &rom2ScriptState{}
	w.entities[0].Facing = w.headingBetween(w.entities[0], w.entities[1])
	if god && !w.CheatGod(1) {
		t.Fatal("CheatGod refused")
	}
	return w, rule
}

func TestSecondGameStaffSpellDamagePaysTheMage(t *testing.T) {
	for _, god := range []bool{false, true} {
		w, _ := secondStaffWorld(t, 100, god)
		before := spAt(t, w, 1).SkillXP[1]
		orders := []Command{cbOrder(1, 2)}
		for n := 0; n < 60 && spAt(t, w, 2).HP == 100; n++ {
			Step(w, orders)
			orders = nil
		}
		if v := spAt(t, w, 2); v.HP >= 100 {
			t.Fatalf("god %v: the staff did not damage the victim: health %d", god, v.HP)
		}
		if got := spAt(t, w, 1).SkillXP[1]; got <= before {
			t.Errorf("god %v: a damaging staff hit paid no experience: %d, was %d", god, got, before)
		}
	}
}

func TestSecondGameStaffSpellKillPaysTheMage(t *testing.T) {
	for _, god := range []bool{false, true} {
		w, rule := secondStaffWorld(t, 1, god)
		before := spAt(t, w, 1).SkillXP[1]
		orders := []Command{cbOrder(1, 2)}
		for n := 0; n < 60 && !w.entities[1].Dead(); n++ {
			Step(w, orders)
			orders = nil
		}
		v := spAt(t, w, 2)
		dead := w.entities[1].Dead()
		if !dead || !v.HasKillCredit || v.KillCreditSpell != int8(rule.ID) {
			t.Fatalf("god %v: kill attribution: dead %v, credit %v, spell %d", god, dead, v.HasKillCredit, v.KillCreditSpell)
		}
		damage := xpGain(xpRaw(100, 1, 100), 60)
		if got := spAt(t, w, 1).SkillXP[1] - before; got <= int32(damage) {
			t.Errorf("god %v: a staff kill paid %d, want more than the damage's %d", god, got, damage)
		}
	}
}
