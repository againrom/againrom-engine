package sim

import (
	"encoding/binary"
	"testing"
)

func secondStaffSourceWorld(t *testing.T, victimHP int32, god bool) *World {
	t.Helper()
	w, _ := secondStaffWorld(t, victimHP, false)
	s := SourceActor{Class: 2, Fighter: false, Stats: [14]uint16{10, 20, 60, 10, 18, 0, 0, 101, 100, 100, 100, 100, 100, 50}, Experience: 2300}
	s.Attack[16] = 1
	binary.LittleEndian.PutUint16(s.Base[2:], 20)
	s.SkillXP[1] = uint32(skillXPFor(20))
	load := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}
	if err := w.RestoreActorLoad(1, load); err != nil {
		t.Fatal(err)
	}
	w.BindSourceDerive(func(n SourceActor, _ int32, _ Rules) (SourceActor, error) { return n, nil })
	if god && !w.CheatGod(1) {
		t.Fatal("CheatGod refused")
	}
	return w
}

func TestSecondGameSourceStaffSpellPaysTheMage(t *testing.T) {
	for _, god := range []bool{false, true} {
		w := secondStaffSourceWorld(t, 100, god)
		before := w.entities[0].ActorLoad.Source.SkillXP[1]
		orders := []Command{cbOrder(1, 2)}
		for n := 0; n < 60 && spAt(t, w, 2).HP == 100; n++ {
			Step(w, orders)
			orders = nil
		}
		c := spAt(t, w, 1)
		if v := spAt(t, w, 2); v.HP >= 100 {
			t.Fatalf("god %v: no damage: health %d, caster phase %v mana %d owner %d ws %d", god, v.HP, c.AttackPhase, c.Mana, c.Owner, c.WeaponSpell)
		}
		if got := c.ActorLoad.Source.SkillXP[1]; got <= before {
			t.Errorf("god %v: source experience %d, was %d (card %d)", god, got, before, c.SkillXP[1])
		}
	}
}
