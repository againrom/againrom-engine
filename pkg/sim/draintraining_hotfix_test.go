package sim

import "testing"

func TestDrainStaffTrainsFromVictimAmountBeforeHealing(t *testing.T) {
	for _, hp := range []int32{100, 99, 50} {
		caster := itemTrainingCaster(11, 2)
		caster.HP, caster.MaxHP, caster.WeaponSpellLevel = hp, 100, 0
		victim := spEnt(2, 1, 0)
		victim.Owner, victim.XPValue = 3, 100
		victim.Protection = [5]int32{100, 100, 100, 100, 100}
		rule := SpellRule{ID: 11, School: 2, ManaCost: 100, MaxRange: 5,
			DamageMin: 10, DamageMax: 10, TargetsUnit: true}
		w := spWorld(t, 1, []SpellRule{rule}, caster, victim)
		Step(w, []Command{cbOrder(1, 2)})
		got := spAt(t, w, 1)
		if got.SkillXP[2] != caster.SkillXP[2]+13 || got.HP != min(100, hp+10) || got.Mana != caster.Mana {
			t.Fatalf("starting HP%d: Water XP%d health%d mana%d", hp, got.SkillXP[2], got.HP, got.Mana)
		}
		if spAt(t, w, 2).HP != 90 {
			t.Fatal("Drain entered the resisted direct-damage resolver")
		}
		for i := range got.SkillXP {
			if i != 2 && got.SkillXP[i] != caster.SkillXP[i] {
				t.Fatalf("Drain trained unrelated slot%d", i)
			}
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var back World
		if err := back.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		if back.Hash() != w.Hash() || spAt(t, &back, 1).SkillXP != got.SkillXP {
			t.Fatal("Native checkpoint lost Drain training")
		}
	}
}

func TestDrainCapsAtVictimHealthPlusTenAndUsesPreHitRelationGate(t *testing.T) {
	for _, sameOwner := range []bool{false, true} {
		caster := itemTrainingCaster(11, 2)
		caster.HP, caster.MaxHP = 50, 100
		victim := spEnt(2, 1, 0)
		victim.Owner, victim.XPValue, victim.HP = 3, 100, 0
		if sameOwner {
			victim.Owner = caster.Owner
		}
		rule := SpellRule{ID: 11, School: 2, DamageMin: 40, DamageMax: 40}
		w := spWorld(t, 1, []SpellRule{rule}, caster, victim)
		if !w.weaponSpellApply(0, 1, rule, 0, &castObs{}) {
			t.Fatal("Drain refused a downed target")
		}
		got := spAt(t, w, 1)
		if spAt(t, w, 2).HP != -10 || got.HP != 60 {
			t.Fatal("Drain ignored the victim health+10 cap")
		}
		want := caster.SkillXP[2]
		if !sameOwner {
			want += 13
		}
		if got.SkillXP[2] != want {
			t.Fatalf("same owner%t: Water XP%d, want%d from pre-hit relation", sameOwner, got.SkillXP[2], want)
		}
	}
}
