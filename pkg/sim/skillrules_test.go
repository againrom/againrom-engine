package sim

import (
	"encoding/binary"
	"testing"
)

func rulesWithCap(t *testing.T, c int32) Rules {
	t.Helper()
	r, err := NewRules(RulesParams{SkillCap: c})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAwardSkillRunsPastLevel100UnderALargerCap(t *testing.T) {
	a := skAwarder(1, 2)
	a.Mind = 400
	a.Skill[3], a.SkillXP[3] = 100, skillXPFor(100)
	s := skSource(2, 3)
	w := cbWorld(t, 1, a, s)
	w.SetRules(rulesWithCap(t, 150))
	ai, si := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, s.ID)
	for i := 0; i < 20; i++ {
		w.awardSkill(ai, 0, 1_000_000, si)
	}
	got := cbAt(t, w, a.ID)
	if got.Skill[3] <= 100 {
		t.Fatalf("skill stayed at %d under a cap of 150", got.Skill[3])
	}
	if got.SkillXP[3] <= skillXPFor(100) {
		t.Fatalf("experience did not pass the original table: %d", got.SkillXP[3])
	}
}

func TestAwardSkillStopsAtTheRulesCap(t *testing.T) {
	a := skAwarder(1, 2)
	a.Mind = 400
	a.Skill[3], a.SkillXP[3] = 149, w149()
	s := skSource(2, 3)
	w := cbWorld(t, 1, a, s)
	w.SetRules(rulesWithCap(t, 150))
	ai, si := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, s.ID)
	for i := 0; i < 5; i++ {
		w.awardSkill(ai, 0, 1_000_000_000, si)
	}
	if got := cbAt(t, w, a.ID); got.Skill[3] != 150 {
		t.Fatalf("level %d, want exactly the cap 150", got.Skill[3])
	}
	if w.awardSkill(ai, 0, 1000, si) {
		t.Fatal("a slot at the cap raised")
	}
}

func w149() int32 {
	r, _ := NewRules(RulesParams{SkillCap: 150})
	return r.SkillXP(149)
}

func TestAWorldWithoutRulesKeepsTheOriginalCap(t *testing.T) {
	a := skAwarder(1, 2)
	a.Mind = 400
	a.Skill[3], a.SkillXP[3] = 100, skillXPFor(100)
	s := skSource(2, 3)
	beforeA, _, afterA, _, raised := skPay(t, a, s, 0, 1_000_000_000)
	if raised || afterA != beforeA {
		t.Fatal("a default world raised a slot at level 100")
	}
}

func TestDecodingIntoAWorldKeepsItsRules(t *testing.T) {
	a := skAwarder(1, 2)
	s := skSource(2, 3)
	w := cbWorld(t, 1, a, s)
	w.SetRules(rulesWithCap(t, 150))
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if got := w.Rules().SkillCap(); got != 150 {
		t.Fatalf("the cap after a decode is %d, want 150", got)
	}
}

func TestAdmittingActorsKeepsTheWorldsRules(t *testing.T) {
	a := skAwarder(1, 2)
	w := cbWorld(t, 1, a)
	w.SetRules(rulesWithCap(t, 150))
	if err := w.ImportOriginalLivingActors(nil); err != nil {
		t.Fatal(err)
	}
	if got := w.Rules().SkillCap(); got != 150 {
		t.Fatalf("the cap after admission is %d, want 150", got)
	}
}

func TestASourceBackedHeroRaisesPastTheOriginalCapThroughTheDerive(t *testing.T) {
	a, b := skAwarder(1, 1), skSource(2, 2)
	a.Mind = 400
	w := cbWorld(t, 1, a, b)
	w.SetRules(rulesWithCap(t, 150))
	s := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 60, 10, 18, 0, 0, 101, 100, 100, 100, 0, 0, 50}}
	s.Attack[16] = 3
	binary.LittleEndian.PutUint16(s.Attack[8:], 100)
	binary.LittleEndian.PutUint16(s.Base[8:], 100)
	s.SkillXP[3] = uint32(skillXPFor(100))
	load := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}
	if err := w.RestoreActorLoad(1, load); err != nil {
		t.Fatal(err)
	}
	var seenCap int32
	w.BindSourceDerive(func(n SourceActor, _ int32, r Rules) (SourceActor, error) {
		seenCap = r.SkillCap()
		level := int32(int16(binary.LittleEndian.Uint16(n.Base[8:])))
		binary.LittleEndian.PutUint16(n.Attack[8:], uint16(min(level, r.SkillCap())))
		return n, nil
	})
	if !w.awardSkill(0, 0, 100, 1) {
		t.Fatal("a source-backed raise at level 100 was refused under a cap of 150")
	}
	if seenCap != 150 {
		t.Fatalf("the derive was given a cap of %d, want 150", seenCap)
	}
	if got := w.entities[0].Skill[3]; got != 101 {
		t.Fatalf("level %d after the raise, want 101", got)
	}
}

func TestASourceBackedHeroAtOneHundredStaysThereUnderTheOriginalRules(t *testing.T) {
	a, b := skAwarder(1, 1), skSource(2, 2)
	a.Mind = 400
	w := cbWorld(t, 1, a, b)
	s := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 60, 10, 18, 0, 0, 101, 100, 100, 100, 0, 0, 50}}
	s.Attack[16] = 3
	binary.LittleEndian.PutUint16(s.Attack[8:], 100)
	binary.LittleEndian.PutUint16(s.Base[8:], 100)
	s.SkillXP[3] = uint32(skillXPFor(100))
	load := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}
	if err := w.RestoreActorLoad(1, load); err != nil {
		t.Fatal(err)
	}
	w.BindSourceDerive(func(n SourceActor, _ int32, r Rules) (SourceActor, error) {
		level := int32(int16(binary.LittleEndian.Uint16(n.Base[8:])))
		binary.LittleEndian.PutUint16(n.Attack[8:], uint16(min(level, r.SkillCap())))
		return n, nil
	})
	w.awardSkill(0, 0, 100, 1)
	if got := w.entities[0].Skill[3]; got > 100 {
		t.Fatalf("level %d under the original rules, want at most 100", got)
	}
}
