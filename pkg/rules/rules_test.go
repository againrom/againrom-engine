package rules

import (
	"math"
	"math/big"
	"testing"
)

func TestDefaultsAreTheOriginalGame(t *testing.T) {
	r := Default()
	if r.SkillCap() != 100 || !r.IsDefault() {
		t.Fatalf("default cap %d", r.SkillCap())
	}
	var zero Rules
	if !zero.Equal(r) || zero.SkillCap() != 100 || !zero.IsDefault() {
		t.Fatal("zero Rules is not the original game")
	}
	for n := int32(0); n <= 100; n++ {
		if r.SkillXP(n) != rom1SkillXP[n] || zero.SkillXP(n) != rom1SkillXP[n] {
			t.Fatalf("level %d", n)
		}
	}
	if r.SkillXP(-3) != 0 || r.SkillXP(500) != 13779612 {
		t.Fatal("clamp outside the table")
	}
	if rom1SkillXP[10] != 1593 || rom1SkillXP[100] != 13779612 {
		t.Fatal("checkpoints")
	}
}

func TestExactFormulaAgreesWithTheOriginalTable(t *testing.T) {
	// The extension past level 100 uses exact integers; over 0..100 it must
	// reproduce the original table except where the original's float
	// truncation differs, which is listed so a drift is noticed.
	ten, eleven := big.NewInt(1), big.NewInt(1)
	var differ []int
	for n := 0; n <= 100; n++ {
		v := new(big.Int).Sub(eleven, ten)
		v.Mul(v, big.NewInt(1000))
		v.Quo(v, ten)
		if v.Int64() != int64(rom1SkillXP[n]) {
			differ = append(differ, n)
		}
		ten.Mul(ten, big.NewInt(10))
		eleven.Mul(eleven, big.NewInt(11))
	}
	if len(differ) != 0 {
		t.Logf("exact formula differs from the table at levels %v", differ)
	}
}

func TestExtendedTableIsMonotoneAndFitsAnInt32(t *testing.T) {
	r, err := New(Params{SkillCap: MaxSkillCap})
	if err != nil {
		t.Fatal(err)
	}
	prev := int32(-1)
	for n := int32(0); n <= MaxSkillCap; n++ {
		v := r.SkillXP(n)
		if v <= prev && n > 0 {
			t.Fatalf("level %d: %d not above %d", n, v, prev)
		}
		prev = v
	}
	if r.SkillXP(MaxSkillCap) == math.MaxInt32 {
		t.Fatal("the cap's experience saturates")
	}
	if got := skillXPAt(MaxSkillCap + 3); got != math.MaxInt32 {
		t.Logf("level %d still fits: %d", MaxSkillCap+3, got)
	}
	if skillXPAt(400) != math.MaxInt32 {
		t.Fatal("no saturation far past the table")
	}
	if r.SkillLevelFor(r.SkillXP(120)) != 120 || r.SkillLevelFor(r.SkillXP(120)-1) != 119 {
		t.Fatal("inverse")
	}
	for n := int32(0); n <= 100; n++ {
		if r.SkillXP(n) != rom1SkillXP[n] {
			t.Fatalf("the first 101 entries are the original table, level %d", n)
		}
	}
}

func TestNewRefusesOutsideTheDeclaredRange(t *testing.T) {
	for _, c := range []int32{-1, 0, MaxSkillCap + 1, 5000} {
		if _, err := New(Params{SkillCap: c}); err == nil {
			t.Fatalf("cap %d accepted", c)
		}
	}
	for _, c := range []int32{MinSkillCap, 50, 100, 150} {
		r, err := New(Params{SkillCap: c})
		if err != nil || r.SkillCap() != c || r.Params().SkillCap != c {
			t.Fatalf("cap %d: %v", c, err)
		}
	}
}

func TestClampAndImmutability(t *testing.T) {
	r, _ := New(Params{SkillCap: 150})
	if r.ClampSkill(200) != 150 || r.ClampSkill(-4) != 0 || r.ClampSkill(77) != 77 {
		t.Fatal("clamp")
	}
	if r.SkillXPExtended(160) <= r.SkillXP(150) || r.SkillXP(160) != r.SkillXP(150) {
		t.Fatal("extended versus clamped")
	}
	if r.Equal(Default()) || !r.Equal(r) {
		t.Fatal("equal")
	}
}

func TestEffectiveSkillAddsTheBonusToTheHeldBaseWithinTheBound(t *testing.T) {
	r120, err := New(Params{SkillCap: 120})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name        string
		r           Rules
		base, bonus int32
		want        int32
	}{
		{"plain", Rules{}, 60, 0, 60},
		{"bonus under the cap", Rules{}, 60, 20, 80},
		{"bonus over the cap", Rules{}, 100, 10, 110},
		{"no bonus at the cap", Rules{}, 100, 0, 100},
		{"base held to the cap", Rules{}, 130, 10, 110},
		{"base held to a larger cap", r120, 130, 10, 130},
		{"bound", Rules{}, 100, 900, EffectiveSkillBound},
		{"floor", Rules{}, 10, -50, 0},
		{"word wrap", Rules{}, 100, 32767, 0},
	} {
		if got := c.r.EffectiveSkill(c.base, c.bonus); got != c.want {
			t.Errorf("%s: EffectiveSkill(%d, %d) = %d, want %d", c.name, c.base, c.bonus, got, c.want)
		}
	}
	if Default().EffectiveSkillLimit() != EffectiveSkillBound || r120.EffectiveSkillLimit() != EffectiveSkillBound {
		t.Fatal("the effective limit follows the training cap")
	}
}
