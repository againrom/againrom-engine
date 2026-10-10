package rules

import (
	"math"
	"math/big"
	"testing"
)

// exactSchoolPrice is floor(11^n * 200 / 10^n) in exact integers, the real
// value the original's floating-point price approximates.
func exactSchoolPrice(n int32) int64 {
	ten := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	v := new(big.Int).Exp(big.NewInt(11), big.NewInt(int64(n)), nil)
	v.Mul(v, big.NewInt(200))
	return v.Quo(v, ten).Int64()
}

// TestSchoolRuleOverEveryLevelToTheCap checks the one school rule at every
// level a mod's cap admits. The price is ftol(1.1^n * 200) (TOWN-GENERAL-107,
// HERO-SKILLBUY-076) and equals the exact floor; the trained experience is
// S(n) + 1 (HERO-SKILLBUY-076) with S the curve the skill table holds
// (HERO-XP-077): the original's table to 100, the same curve past it.
func TestSchoolRuleOverEveryLevelToTheCap(t *testing.T) {
	for n := int32(0); n <= MaxSkillCap; n++ {
		price, err := SchoolPrice(n)
		if err != nil || int64(price) != exactSchoolPrice(n) {
			t.Errorf("SchoolPrice(%d) = %d, %v; want %d", n, price, err, exactSchoolPrice(n))
		}
		xp, err := SchoolTrainedXP(n)
		if want := Default().SkillXPExtended(n) + 1; err != nil || xp != want {
			t.Errorf("SchoolTrainedXP(%d) = %d, %v; want %d", n, xp, err, want)
		}
	}
	for n, want := range map[int32]int32{0: 200, 9: 471, 10: 518, 30: 3489, 50: 23478, 100: 2756122} {
		if got, _ := SchoolPrice(n); got != want {
			t.Errorf("SchoolPrice(%d) = %d, want %d", n, got, want)
		}
	}
	for n, want := range map[int32]int32{0: 1, 1: 101, 10: 1594, 11: 1854, 100: 13779613} {
		if got, _ := SchoolTrainedXP(n); got != want {
			t.Errorf("SchoolTrainedXP(%d) = %d, want %d", n, got, want)
		}
	}
}

// TestSchoolRuleStageBoundaries pins the fixed-width stages HERO-SKILLBUY-076
// orders: the trained experience turns negative as a signed dword at level
// 153 and wraps the unsigned dword at 161; the price turns negative at base
// 170 and wraps at 178; the experience and the price leave the signed-qword
// conversion at 386 and 403, where the rule refuses.
func TestSchoolRuleStageBoundaries(t *testing.T) {
	xp := func(n int32) int32 {
		v, err := SchoolTrainedXP(n)
		if err != nil {
			t.Fatalf("SchoolTrainedXP(%d): %v", n, err)
		}
		return v
	}
	price := func(n int32) int32 {
		v, err := SchoolPrice(n)
		if err != nil {
			t.Fatalf("SchoolPrice(%d): %v", n, err)
		}
		return v
	}
	if xp(152) <= 0 || xp(153) >= 0 {
		t.Errorf("signed experience crossing: S(152)+1 = %d, S(153)+1 = %d", xp(152), xp(153))
	}
	if uint32(xp(161)) >= uint32(xp(160)) {
		t.Errorf("unsigned experience wrap: %d then %d", uint32(xp(160)), uint32(xp(161)))
	}
	if price(169) <= 0 || price(170) >= 0 {
		t.Errorf("signed price crossing: %d then %d", price(169), price(170))
	}
	if price(178) <= 0 || uint32(price(178)) >= uint32(price(177)) {
		t.Errorf("unsigned price wrap: %d then %d", uint32(price(177)), uint32(price(178)))
	}
	if _, err := SchoolTrainedXP(385); err != nil {
		t.Errorf("SchoolTrainedXP(385): %v", err)
	}
	if _, err := SchoolTrainedXP(386); err == nil {
		t.Error("SchoolTrainedXP(386) left the signed-qword domain without refusing")
	}
	if _, err := SchoolPrice(402); err != nil {
		t.Errorf("SchoolPrice(402): %v", err)
	}
	if _, err := SchoolPrice(403); err == nil {
		t.Error("SchoolPrice(403) left the signed-qword domain without refusing")
	}
	if _, err := SchoolPrice(math.MaxInt16); err == nil {
		t.Error("SchoolPrice of the largest signed level did not refuse")
	}
}
