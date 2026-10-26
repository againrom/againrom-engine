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

// A source Human with General experience grows its pools by the school
// slots alone under the second game's rules: the maximum the first game's
// rules give the same Human whose word holds the school slots' sum.
func TestSourceDerivePoolsFollowTheSecondGamesRule(t *testing.T) {
	hero := classTwoHero(40)
	hero.SkillXP = [6]uint32{2000000, 45000, 0, 0, 0, 0}
	hero.Experience = 2045000
	school := hero
	school.Experience = 45000
	pools := func(s sim.SourceActor, r sim.Rules) uint16 {
		t.Helper()
		got, err := deriveSourceActor(s, 0, r)
		if err != nil {
			t.Fatal(err)
		}
		return SourceHumanState(got, 0).HealthMax
	}
	first, second, want := pools(hero, sim.Rules{}), pools(hero, sim.Rules{}.WithSchoolPoolExperience()), pools(school, sim.Rules{})
	if second != want {
		t.Fatalf("second game health %d, want %d", second, want)
	}
	if first == second {
		t.Fatalf("health %d under both games: the General experience did not reach the first game's pools", first)
	}
}
