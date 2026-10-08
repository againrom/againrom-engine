package sim

import (
	"reflect"
	"testing"
)

func TestRepairNativeSkillsPreservesSavedStateAndActiveTerms(t *testing.T) {
	for _, hp := range []int32{161, 0, -9} {
		for _, active := range []uint8{1, 2} {
			w := trainingWorld(t, 17, 0)
			e := &w.entities[0]
			e.Skill[1], e.XPSlot = 17, active
			e.DamageBase, e.DamageSpread, e.ToHit, e.Defence = 70, 11, 300, 99
			e.HP, e.MaxHP, e.CurrentProfileBasis = hp, 161, ProfileOriginalCurrent
			e.HumanMovement = HumanMovement{Present: true, RawSpeed: 8, NativeSpeed: 7, Capacity: 101}
			e.TurnRemaining, e.TurnTotal, e.DyingTime = 3, 5, 12
			if hp <= 0 {
				e.Decay = DecayFallen
			}
			before := *e
			levels := before.Skill
			levels[1] = 62
			want := before
			want.Skill = levels
			if active == 1 {
				want.ToHit, want.DamageBase = 435, 79
			}
			RefreshBook(w.rules, &want, w.spells)
			if !w.RepairNativeSkillLevels(before.ID, levels) || !reflect.DeepEqual(w.entities[0], want) {
				t.Fatalf("HP %d active %d repaired sheet: got %+v want %+v", hp, active, w.entities[0], want)
			}
			if !w.RepairNativeSkillLevels(before.ID, levels) || !reflect.DeepEqual(w.entities[0], want) {
				t.Fatal("repeated repair changed the saved sheet")
			}
			levels[1] = 1
			if !w.RepairNativeSkillLevels(before.ID, levels) || !reflect.DeepEqual(w.entities[0], want) {
				t.Fatal("repair lowered a saved effective skill")
			}
		}
	}
}

func TestRepairNativeSkillsRejectsSourceActorsAndAbsentIDs(t *testing.T) {
	w := trainingWorld(t, 17, 0)
	levels := w.entities[0].Skill
	levels[3] = 62
	w.entities[0].ActorLoad.Source.Class = 2
	before := w.entities[0]
	if w.RepairNativeSkillLevels(before.ID, levels) || w.RepairNativeSkillLevels(999, levels) || !reflect.DeepEqual(w.entities[0], before) {
		t.Fatal("repair changed a source actor or accepted an absent actor")
	}
}
