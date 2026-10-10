package data

import (
	"testing"

	"againrom/internal/synth"
)

// HeroArchetype reads each archetype section's five keys by sex and class,
// and refuses a section that lacks one.
func TestHeroArchetypeReadsEachSectionBySexAndClass(t *testing.T) {
	stats := func(name string, body, reaction, mind, spirit, skill int32) synth.RegNode {
		return regDir(name, regInt("Face", 2), regInt("Body", body), regInt("Reaction", reaction),
			regInt("Mind", mind), regInt("Spirit", spirit), regInt("Skill", skill))
	}
	n := npcReg(t,
		stats("MaleFighter", 41, 35, 20, 15, 1),
		stats("MaleMage", 29, 22, 40, 34, 1),
		stats("FemaleFighter", 32, 40, 31, 24, 5),
		regDir("FemaleMage", regInt("Body", 21), regInt("Reaction", 25), regInt("Mind", 33), regInt("Spirit", 41)),
	)
	for _, c := range []struct {
		female, mage bool
		want         HeroArchetype
	}{
		{false, false, HeroArchetype{41, 35, 20, 15, 1, 2}},
		{false, true, HeroArchetype{29, 22, 40, 34, 1, 2}},
		{true, false, HeroArchetype{32, 40, 31, 24, 5, 2}},
	} {
		got, ok := n.HeroArchetype(c.female, c.mage)
		if !ok || got != c.want {
			t.Errorf("female=%v mage=%v: %+v/%v, want %+v", c.female, c.mage, got, ok, c.want)
		}
	}
	if _, ok := n.HeroArchetype(true, true); ok {
		t.Error("a section without Skill answered an archetype")
	}
	var none *NPCDefs
	if _, ok := none.HeroArchetype(false, false); ok {
		t.Error("a nil registry answered an archetype")
	}
}
