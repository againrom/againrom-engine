package data

import "testing"

// humanDeriveInput is Body 30, Reaction 25, Mind 20, Spirit 35, a mage with a
// mana pool, skills 4, 12, 0, 7, 0, 3 and their opening experience, wielding
// slot 1. The expected numbers below are worked by hand from the claims.
func humanDeriveInput() HumanInput {
	in := HumanInput{Stat: [4]int32{30, 25, 20, 35}, Skill: [SkillSlots]int32{4, 12, 0, 7, 0, 3}, Active: 1, ManaPool: true}
	for _, level := range in.Skill {
		in.Experience += SkillXPFor(level)
	}
	return in
}

func deriveHuman(t *testing.T, in HumanInput) HumanOutput {
	t.Helper()
	out, err := DeriveHuman(in)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// HERO-HP-005 and HERO-MP-006: experience 3881 gives log1.1(3881/5000+1) =
// 6.03. Health ftol(30 + 6.03) = 36, then ftol(36 x (1.1^30/100 + 1)) = 42.
// Mana ftol(70 + 2 x 6.03) = 82, then ftol(82 x (1.1^35/100 + 1)) = 105.
func TestDeriveHumanPools(t *testing.T) {
	in := humanDeriveInput()
	if in.Experience != 3881 {
		t.Fatalf("experience %d, want 3881", in.Experience)
	}
	out := deriveHuman(t, in)
	if out.HealthMax != 42 || out.ManaMax != 105 {
		t.Fatalf("pools %d/%d, want 42/105", out.HealthMax, out.ManaMax)
	}
	in.ManaPool, in.Terms.HealthMax, in.Terms.ManaMax = false, 7, 9
	if out := deriveHuman(t, in); out.HealthMax != 49 || out.ManaMax != 9 {
		t.Fatalf("pools without a mana column, with terms: %d/%d, want 49/9", out.HealthMax, out.ManaMax)
	}
}

// HERO-CAP-015: a stat is the smaller of itself and 50 plus its cap term.
func TestDeriveHumanStatCaps(t *testing.T) {
	in := humanDeriveInput()
	in.Stat[0], in.StatCap[0] = 60, 5
	in.Stat[1] = 70
	if out := deriveHuman(t, in); out.Stat[0] != 55 || out.Stat[1] != 50 {
		t.Fatalf("capped stats %v, want 55 and 50", out.Stat)
	}
}

// HERO-SIGHT-007: ftol(((20 + 25)/25 + 4) x 256) = 1484 in 1/256 cell, plus
// the term; capacity 30 x 10 + 1, plus the term.
func TestDeriveHumanSightAndCapacity(t *testing.T) {
	in := humanDeriveInput()
	in.Terms.Sight, in.Terms.Capacity = 2*256+3, 4
	out := deriveHuman(t, in)
	if out.Sight != 1484+515 || out.Sight>>8 != 7 || out.Capacity != 305 {
		t.Fatalf("sight %d, capacity %d; want 1999 and 305", out.Sight, out.Capacity)
	}
}

// SAV-1116, HERO-SPEED-008: Reaction 25 gives 25/5 + 12 = 17, the rider 27.
// The penalty and its floor act before the modifier; the load is compared in
// 32 bits.
func TestDeriveHumanSpeed(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		reaction, body       int32
		rider                bool
		modifier, load       int32
		base, speed, keptMod int32
	}{
		{"unloaded", 25, 30, false, 0, 0, 17, 17, 0},
		{"rider", 25, 30, true, 0, 0, 27, 27, 0},
		{"slow below twelve", 10, 30, false, 2, 0, 10, 12, 2},
		{"overloaded under haste", 15, 26, false, 4, 4680, 15, 10, 4},
		{"overloaded under slow", 15, 26, false, -9, 4680, 15, -3, 0},
		{"load above a word", 25, 30, false, 0, 40000, 17, 6, 0},
	} {
		in := humanDeriveInput()
		in.Stat[0], in.Stat[1], in.Rider, in.Terms.Speed, in.Load = tc.body, tc.reaction, tc.rider, tc.modifier, tc.load
		out := deriveHuman(t, in)
		if out.BaseSpeed != tc.base || out.Speed != tc.speed || out.SpeedModifier != tc.keptMod {
			t.Errorf("%s: base %d, word %d, modifier %d; want %d, %d, %d", tc.name, out.BaseSpeed, out.Speed, out.SpeedModifier, tc.base, tc.speed, tc.keptMod)
		}
	}
}

// HERO-DAMAGE-022: the pair is ftol(1.1^Body/20) as two bytes; to-hit is
// ftol((1.1^Body + 1.1^Reaction)/5). The active level adds three times itself
// to to-hit and a fifth to the base.
func TestDeriveHumanDamageAndToHit(t *testing.T) {
	out := deriveHuman(t, humanDeriveInput())
	if out.DamageBase != 2 || out.DamageSpread != 0 || out.ToHit != 41 {
		t.Fatalf("damage %d-%d, to-hit %d; want 2-0 and 41", out.DamageBase, out.DamageSpread, out.ToHit)
	}
	in := humanDeriveInput()
	in.Stat[0], in.StatCap[0], in.Active = 100, 50, 0
	in.Terms.DamageBase, in.Terms.ToHit = 3, 5
	// ftol(1.1^100/20) = 689 stores as the byte 177.
	if out := deriveHuman(t, in); out.DamageBase != 180 || out.DamageSpread != 177 || out.ToHit != 2763 {
		t.Fatalf("Body 100: damage %d-%d, to-hit %d; want 180-177 and 2763", out.DamageBase, out.DamageSpread, out.ToHit)
	}
}

// DIV-2217 and HERO-GENERAL-092: slots 1 to 5 are the trained level held to
// the training cap plus the bonus, within 0..255; slot 0 passes through.
func TestDeriveHumanSkillRestore(t *testing.T) {
	in := humanDeriveInput()
	in.Skill = [SkillSlots]int32{4, 100, 255, 7, 104, 3}
	in.Terms.SkillBonus = [SkillSlots]int32{6, 10, 10, -20, 0, 0}
	out := deriveHuman(t, in)
	if want := [SkillSlots]int32{4, 110, 110, 0, 100, 3}; out.Skill != want {
		t.Fatalf("skills %v, want %v", out.Skill, want)
	}
	in.TrainingCap = 300
	in.Skill[2] = 290
	if out := deriveHuman(t, in); out.Skill[2] != 300 {
		t.Fatalf("a mod cap above 255 bounds at the cap: %d", out.Skill[2])
	}
}

// Defence Reaction/3 and protection Spirit/2 take the terms, then clamp:
// defence and absorption at 0, protection to 0..min(100, Spirit/2 + 70).
func TestDeriveHumanDefenceClamps(t *testing.T) {
	in := humanDeriveInput()
	in.Terms.Defence, in.Terms.Absorption = 6, 2
	in.Terms.Protection = [5]int32{10, 80, -30, 0, 90}
	in.Terms.Resistance = [5]int32{1, 2, 3, 4, 5}
	out := deriveHuman(t, in)
	if out.Defence != 14 || out.Absorption != 2 || out.Protection != [5]int32{27, 87, 0, 17, 87} || out.Resistance != in.Terms.Resistance {
		t.Fatalf("defence %d, absorption %d, protection %v, resistance %v", out.Defence, out.Absorption, out.Protection, out.Resistance)
	}
	in.Terms.Defence, in.Terms.Absorption = -40, -1
	if out := deriveHuman(t, in); out.Defence != 0 || out.Absorption != 0 {
		t.Fatalf("negative defence %d, absorption %d", out.Defence, out.Absorption)
	}
}
