package sim

import (
	"encoding/binary"
	"testing"
)

func liftHuman(base, bonus, stored int32, active uint8) Entity {
	var e Entity
	s := &e.ActorLoad.Source
	s.Class = 2
	binary.LittleEndian.PutUint16(s.Base[2+2*1:], uint16(int16(base)))
	binary.LittleEndian.PutUint16(s.Modifier[20+2*1:], uint16(int16(bonus)))
	binary.LittleEndian.PutUint16(s.Attack[2+2*1:], uint16(int16(stored)))
	s.Attack[16] = active
	e.Skill[1], e.ToHit, e.DamageBase = stored, 300, 70
	return e
}

func TestAHumanAtTheOriginalClampTakesItsBonusAboveOneHundred(t *testing.T) {
	r150, err := NewRules(RulesParams{SkillCap: 150})
	if err != nil {
		t.Fatal(err)
	}
	for _, rules := range []Rules{{}, r150} {
		e := liftHuman(100, 10, 100, 1)
		e.liftEffectiveSkills(rules)
		if e.Skill[1] != 110 || e.ToHit != 330 || e.DamageBase != 72 {
			t.Fatalf("cap %d: level %d to-hit %d damage %d, want 110 330 72", rules.SkillCap(), e.Skill[1], e.ToHit, e.DamageBase)
		}
		if s := e.ActorLoad.Source; binary.LittleEndian.Uint16(s.Attack[4:]) != 100 {
			t.Fatal("the source block was changed")
		}
	}
	e := liftHuman(100, 10, 100, 2)
	e.liftEffectiveSkills(Rules{})
	if e.Skill[1] != 110 || e.ToHit != 300 || e.DamageBase != 70 {
		t.Fatalf("inactive slot: level %d to-hit %d damage %d, want 110 300 70", e.Skill[1], e.ToHit, e.DamageBase)
	}
}

func TestAHumanWithoutAnExcessBonusIsUntouched(t *testing.T) {
	for _, c := range []struct{ base, bonus, stored int32 }{
		{100, 0, 100},  // the control: no bonus
		{80, 10, 90},   // the stored level is not at the clamp
		{100, -10, 90}, // a penalty
		{0, 0, 100},    // a record whose base block is empty
	} {
		e := liftHuman(c.base, c.bonus, c.stored, 1)
		before := e
		e.liftEffectiveSkills(Rules{})
		if e.Skill != before.Skill || e.ToHit != before.ToHit || e.DamageBase != before.DamageBase {
			t.Fatalf("%+v changed the live values: level %d to-hit %d damage %d", c, e.Skill[1], e.ToHit, e.DamageBase)
		}
	}
	e := liftHuman(100, 900, 100, 1)
	e.liftEffectiveSkills(Rules{})
	if e.Skill[1] != 255 {
		t.Fatalf("level %d, want the bound 255", e.Skill[1])
	}
	e = liftHuman(100, 10, 100, 1)
	e.ActorLoad.Source.Class = 1
	e.liftEffectiveSkills(Rules{})
	if e.Skill[1] != 100 {
		t.Fatal("a unit record was lifted")
	}
}

// liftedAwardWorld holds a source-backed fighter at stored level 100 whose base
// 90 plus a +15 modifier lifts slot 1 to 105. Its experience sits xpLeft short of
// the raise past base 90.
func liftedAwardWorld(t *testing.T, xpLeft int32) *World {
	t.Helper()
	a, b := skAwarder(1, 1), skSource(2, 2)
	a.Mind = 400
	w := cbWorld(t, 1, a, b)
	s := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 60, 10, 18, 0, 0, 101, 100, 100, 100, 0, 0, 50}}
	s.Attack[16] = 1
	binary.LittleEndian.PutUint16(s.Attack[0:], 300)
	s.Attack[14] = 70
	binary.LittleEndian.PutUint16(s.Attack[4:], 100)
	binary.LittleEndian.PutUint16(s.Base[4:], 90)
	binary.LittleEndian.PutUint16(s.Modifier[22:], 15)
	s.SkillXP[1] = uint32(skillXPFor(90) - xpLeft)
	load := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}
	if err := w.RestoreActorLoad(1, load); err != nil {
		t.Fatal(err)
	}
	w.BindSourceDerive(func(n SourceActor, _ int32, r Rules) (SourceActor, error) {
		for j := 1; j <= 5; j++ {
			level := int32(int16(binary.LittleEndian.Uint16(n.Base[2+2*j:]))) + int32(int16(binary.LittleEndian.Uint16(n.Modifier[20+2*j:])))
			binary.LittleEndian.PutUint16(n.Attack[2+2*j:], uint16(max(0, min(level, r.SkillCap()))))
		}
		return n, nil
	})
	if !w.deriveSource(0) {
		t.Fatal("derive refused")
	}
	return w
}

func TestRepeatedAwardsThatRaiseNothingKeepALiftedHumansCombatBlock(t *testing.T) {
	w := liftedAwardWorld(t, 1_000_000)
	e := w.entities[0]
	if e.Skill[1] != 105 || e.ToHit != 315 || e.DamageBase != 71 {
		t.Fatalf("lifted values %d %d %d, want 105 315 71", e.Skill[1], e.ToHit, e.DamageBase)
	}
	for n := 1; n <= 4; n++ {
		if w.awardSkill(0, 0, 1, 1) {
			t.Fatal("a small award raised the level")
		}
		e = w.entities[0]
		if e.Skill[1] != 105 || e.ToHit != 315 || e.DamageBase != 71 {
			t.Fatalf("award %d: %d %d %d, want 105 315 71", n, e.Skill[1], e.ToHit, e.DamageBase)
		}
	}
}

func TestARaiseAfterNonRaisingAwardsLandsOnTheRightLiftedValues(t *testing.T) {
	w := liftedAwardWorld(t, 10_000)
	for n := 0; n < 3; n++ {
		w.awardSkill(0, 0, 1, 1)
	}
	if !w.awardSkill(0, 0, 1_000_000, 1) {
		t.Fatal("the award did not raise")
	}
	e := w.entities[0]
	// Base 91 plus 15 is level 106: three times 6 over the stored 100, and one
	// more fifth of damage.
	if e.Skill[1] != 106 || e.ToHit != 318 || e.DamageBase != 71 {
		t.Fatalf("after the raise %d %d %d, want 106 318 71", e.Skill[1], e.ToHit, e.DamageBase)
	}
}

func TestANativeHeroTrainsItsBaseToTheCapWhileAWornBonusLiftsTheLevel(t *testing.T) {
	for _, bonus := range []int32{10, 0} {
		a := skAwarder(1, 2)
		a.Mind = 400
		a.Skill[3], a.SkillXP[3] = 95+bonus, skillXPFor(95)
		s := skSource(2, 3)
		w := cbWorld(t, 1, a, s)
		if bonus != 0 {
			w.equipment[indexOfEntity(w.entities, a.ID)][11] = ItemInstance{Effects: []ItemEffect{{Kind: 29, Operand: uint32(bonus)}}}
		}
		ai, si := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, s.ID)
		for i := 0; i < 20; i++ {
			w.awardSkill(ai, 0, 1_000_000, si)
		}
		got := cbAt(t, w, a.ID)
		if got.Skill[3] != 100+bonus {
			t.Fatalf("bonus %d: level %d, want %d", bonus, got.Skill[3], 100+bonus)
		}
		if w.awardSkill(ai, 0, 1000, si) {
			t.Fatalf("bonus %d: a base at the training cap raised", bonus)
		}
	}
}
