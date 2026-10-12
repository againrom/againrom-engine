package mapload

import (
	"testing"

	"againrom/pkg/data"
)

// TestTrainingCapAttackKeepsTheSourceDomain holds a derived attack block to
// the training cap: an active skill at 110 under a cap of 100 drops to 100
// with its to-hit and damage terms; an inactive skill only drops its level; a
// level at the cap is unchanged.
func TestTrainingCapAttackKeepsTheSourceDomain(t *testing.T) {
	a := data.HumanAttack{ToHit: 129, DamageBase: 73, Active: 1}
	a.Skill[1], a.Skill[2], a.Skill[3] = 110, 120, 100
	got := trainingCapAttack(a, 100)
	if got.ToHit != 99 || got.DamageBase != 71 || got.Skill[1] != 100 || got.Skill[2] != 100 || got.Skill[3] != 100 {
		t.Fatalf("projected %+v", got)
	}
	if again := trainingCapAttack(got, 100); again != got {
		t.Fatalf("a block in the domain changed: %+v", again)
	}
	if wide := trainingCapAttack(a, 150); wide != a {
		t.Fatalf("a higher training cap changed the block: %+v", wide)
	}
}
