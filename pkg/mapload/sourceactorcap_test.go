package mapload

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

func classTwoHero(level uint16) sim.SourceActor {
	s := sim.SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 60, 10, 18, 0, 0, 500, 100, 100, 100, 0, 0, 50}}
	s.Attack[16] = 1
	binary.LittleEndian.PutUint16(s.Base[4:], 0)
	binary.LittleEndian.PutUint16(s.Base[2+2:], 0)
	binary.LittleEndian.PutUint16(s.Base[2+2*1:], level)
	return s
}

func TestSourceDeriveHoldsAClassTwoHeroAboveOneHundredUnderTheModsRules(t *testing.T) {
	r, err := sim.NewRules(sim.RulesParams{SkillCap: 150})
	if err != nil {
		t.Fatal(err)
	}
	got, err := deriveSourceActor(classTwoHero(120), 0, r)
	if err != nil {
		t.Fatal(err)
	}
	if lvl := binary.LittleEndian.Uint16(got.Attack[2+2*1:]); lvl != 120 {
		t.Fatalf("current level %d, want 120", lvl)
	}
	orig, err := deriveSourceActor(classTwoHero(120), 0, sim.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if lvl := binary.LittleEndian.Uint16(orig.Attack[2+2*1:]); lvl != 100 {
		t.Fatalf("original rules: current level %d, want 100", lvl)
	}
}
