package data

import "testing"

func capState(base int32) HumanState {
	var h HumanState
	h.Body, h.Reaction, h.Mind, h.Spirit = 40, 40, 40, 40
	h.Capacity, h.HealthMax, h.ManaMax = 500, 100, 100
	h.Health, h.Mana = 100, 100
	h.Base.Skill[1] = uint16(base)
	h.Attack.Active = 1
	return h
}

func TestTheHumanDeriveKeepsTheOriginalCapWithoutARulesValue(t *testing.T) {
	for _, base := range []int32{0, 50, 100, 101, 120} {
		plain, err := capState(base).Derive()
		if err != nil {
			t.Fatal(err)
		}
		explicit := capState(base).WithSkillCap(SkillCap)
		same, err := explicit.Derive()
		if err != nil {
			t.Fatal(err)
		}
		same.skillCap = 0
		if plain != same {
			t.Fatalf("base %d: a zero cap and cap 100 derive differently", base)
		}
		if want := uint16(min(base, 100)); plain.Attack.Skill[1] != want {
			t.Fatalf("base %d: level %d, want %d", base, plain.Attack.Skill[1], want)
		}
	}
}

func TestTheHumanDeriveHoldsALevelAboveOneHundredUnderALargerCap(t *testing.T) {
	h := capState(120).WithSkillCap(150)
	got, err := h.Derive()
	if err != nil {
		t.Fatal(err)
	}
	if got.Attack.Skill[1] != 120 {
		t.Fatalf("level %d, want 120", got.Attack.Skill[1])
	}
	over := capState(170).WithSkillCap(150)
	if got, _ = over.Derive(); got.Attack.Skill[1] != 150 {
		t.Fatalf("level %d, want the cap 150", got.Attack.Skill[1])
	}
	plain, _ := capState(120).Derive()
	if got.Attack.ToHit == plain.Attack.ToHit {
		t.Fatal("the extra levels did not reach the to-hit sum")
	}
}
