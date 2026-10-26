package data_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/rules"
)

// innMageState is the town-1 inn mage of TestRecomputeSecondGamePoolsLeaveGeneralOut
// as a stored Human: General, Water and Air at 99 with their experience in
// the slots and the six slots' sum in the experience word, as SAVE writes it.
func innMageState() data.HumanState {
	h := data.HumanState{Body: 35, Reaction: 40, Mind: 45, Spirit: 50, ManaMax: 1, TypeID: 0x21}
	for _, slot := range []int{0, 2, 3} {
		h.Attack.Skill[slot], h.Base.Skill[slot] = 99, 99
		h.SkillXP[slot] = uint32(data.SkillXPFor(99))
		h.Experience += h.SkillXP[slot]
	}
	return h
}

func TestStoredHumanPoolsFollowTheGamesRule(t *testing.T) {
	for _, c := range []struct {
		label    string
		state    data.HumanState
		hp, mana uint16
	}{
		{"no rules", innMageState(), 163, 623},
		{"first game", innMageState().WithRules(rules.Rules{}), 163, 623},
		{"first game defaults", innMageState().WithRules(rules.Default()), 163, 623},
		{"second game", innMageState().WithRules(rules.Rules{}.WithSchoolPoolExperience()), 158, 604},
	} {
		n, err := c.state.Derive()
		if err != nil {
			t.Fatal(err)
		}
		if n.HealthMax != c.hp || n.ManaMax != c.mana {
			t.Errorf("%s: pools %d/%d, want %d/%d", c.label, n.HealthMax, n.ManaMax, c.hp, c.mana)
		}
	}
}

// A second-game stored Human derives the native route's maxima.
func TestStoredSecondGameHumanMatchesTheNativeDerive(t *testing.T) {
	r := rules.Rules{}.WithSchoolPoolExperience()
	hero := data.Hero{Body: 35, Reaction: 40, Mind: 45, Spirit: 50, Skill: [data.SkillSlots]int32{99, 0, 99, 99, 0, 0}}
	native := hero.Recompute(data.Profile{HealthColumn: true, ManaColumn: true}, data.Loadout{Rules: r})
	n, err := innMageState().WithRules(r).Derive()
	if err != nil {
		t.Fatal(err)
	}
	if int32(n.HealthMax) != native.HealthMax || int32(n.ManaMax) != native.ManaMax {
		t.Fatalf("stored pools %d/%d, native %d/%d", n.HealthMax, n.ManaMax, native.HealthMax, native.ManaMax)
	}
}

// Under the first game's rules the pools read the stored word even where it
// is not the slots' sum, as before rules reached the stored Human.
func TestStoredFirstGameHumanReadsTheExperienceWord(t *testing.T) {
	h := innMageState()
	h.Experience = 0
	plain, err := h.Derive()
	if err != nil {
		t.Fatal(err)
	}
	ruled, err := h.WithRules(rules.Default()).Derive()
	if err != nil {
		t.Fatal(err)
	}
	if plain != ruled {
		t.Fatal("the first game's rules move a stored derive")
	}
	if plain.HealthMax >= 163 {
		t.Fatalf("health %d: the word was not read", plain.HealthMax)
	}
}

// A derive's result carries no rules: a derive under the second game's rules
// equals the plain derive of a word that holds the school slots' sum.
func TestStoredHumanResultCarriesNoRules(t *testing.T) {
	r := rules.Rules{}.WithSchoolPoolExperience()
	ruled, err := innMageState().WithRules(r).Derive()
	if err != nil {
		t.Fatal(err)
	}
	h := innMageState()
	h.Experience -= h.SkillXP[0]
	plain, err := h.Derive()
	if err != nil {
		t.Fatal(err)
	}
	plain.Experience = ruled.Experience
	if ruled != plain {
		t.Fatal("a ruled derive's result differs from the plain derive of the same pool experience")
	}
}
