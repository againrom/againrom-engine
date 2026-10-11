package sim

import "testing"

const (
	crtMultiplier = 214013
	crtIncrement  = 2531011
)

// crtStateFor answers an original-stream state whose next len(raws) rand()
// results are raws (one or two values), found by inverting the recurrence.
func crtStateFor(t *testing.T, raws ...int32) uint32 {
	t.Helper()
	inv := uint32(crtMultiplier)
	for range 5 {
		inv *= 2 - crtMultiplier*inv
	}
	back := func(s uint32) uint32 { return (s - crtIncrement) * inv }
	for top := uint32(0); top < 2; top++ {
		for low := uint32(0); low < 1<<16; low++ {
			s1 := top<<31 | uint32(raws[0])<<16 | low
			if len(raws) == 1 {
				return back(s1)
			}
			if int32((s1*crtMultiplier+crtIncrement)>>16&0x7fff) == raws[1] {
				return back(s1)
			}
		}
	}
	t.Fatal("no state yields", raws)
	return 0
}

// originalStream puts w on the original rand() stream at state.
func originalStream(w *World, state uint32) {
	w.rng = rng{state: uint64(state), original: true}
}

// crtAfter is state advanced by n rand() calls.
func crtAfter(state uint32, n int) uint32 {
	for range n {
		state = state*crtMultiplier + crtIncrement
	}
	return state
}

func castDamageWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	return hlWorld(t, 7, Relations{}, nil, ents...)
}

// oldCastDamage is the former 64-bit integer pair, kept as the loss control.
func oldCastDamage(dmin, dmax, power int64) (int64, int64) {
	base := dmin * (power + 30) / 30
	return base, dmax*(power+30)/30 - base
}

func TestCastDamageNarrowsTheRecordToBytes(t *testing.T) {
	target := spEnt(2, 3, 0)
	target.HP, target.MaxHP = 2000, 2000
	w := castDamageWorld(t, spEnt(1, 0, 0), target)
	// 200 * (100/30 + 1) is 866, base byte 98; 210 * f - 98 is 812, spread
	// byte 44. The draws are the physical step's accuracy roll, then raw 32767
	// for the spread, which answers its inclusive top: 98 + 44.
	originalStream(w, crtStateFor(t, 0, 32767))
	rule := SpellRule{ID: 1, Damaging: true, DamageMin: 200, DamageMax: 210}
	w.applySpellDamage(0, 1, rule, 100)
	if got := 2000 - w.entities[1].HP; got != 142 {
		t.Fatalf("damage %d, want 142 = 98 + 44", got)
	}
	if base, spread := oldCastDamage(200, 210, 100); base+spread == 142 {
		t.Fatal("loss control: the integer formula reaches the same damage")
	}
}

func TestCastDamageRoundsTheRecordInBinary64(t *testing.T) {
	// 45 * (12/30 + 1) is 62.99999999999999 in binary64: base 62, not 63.
	// A zero raw draw takes the base itself.
	w := castDamageWorld(t, spEnt(1, 0, 0), spEnt(2, 3, 0))
	originalStream(w, crtStateFor(t, 0, 0))
	w.applySpellDamage(0, 1, SpellRule{ID: 1, Damaging: true, DamageMin: 45, DamageMax: 50}, 12)
	if got := 100 - w.entities[1].HP; got != 62 {
		t.Fatalf("damage %d, want 62", got)
	}
	if base, _ := oldCastDamage(45, 50, 12); base == 62 {
		t.Fatal("loss control: the integer formula gives the same base")
	}
}

func TestCastDamageSignedResistanceAndBias(t *testing.T) {
	for _, tc := range []struct {
		resistance int32
		want       int32
	}{
		{-50, 15},   // (10*150 + 75) / 100: a negative resistance amplifies.
		{25, 8},     // (10*75 + 75) / 100.
		{0, 10},     // zero keeps the value.
		{150, 0},    // a negative component lands as zero.
		{65486, 15}, // read as the signed word -50.
	} {
		target := spEnt(2, 3, 0)
		target.Protection[0] = tc.resistance
		w := castDamageWorld(t, spEnt(1, 0, 0), target)
		w.applySpellDamage(0, 1, SpellRule{ID: 1, Damaging: true, School: 1, DamageMin: 10, DamageMax: 10}, 0)
		if got := 100 - w.entities[1].HP; got != tc.want {
			t.Errorf("resistance %d: damage %d, want %d", tc.resistance, got, tc.want)
		}
	}
	// Loss control: the former 0..100 clamp gave 10 for -50.
	if old := (int64(10)*int64(100-0) + 75) / 100; old == 15 {
		t.Fatal("loss control")
	}
}

func TestCastDamageDrawsThePhysicalStepFirst(t *testing.T) {
	for _, tc := range []struct {
		name       string
		dmin, dmax int32
		bless      bool
		draws      int
	}{
		{"pair", 10, 12, false, 3},
		{"empty pair keeps the physical draws", 0, 0, false, 2},
		{"blessed source rolls Bless", 10, 12, true, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := effectMage(1, 0, 0, 0)
			w := castDamageWorld(t, caster, spEnt(2, 3, 0))
			if tc.bless && !w.attachEffect(1, 1, SpellRule{ID: 23}, EffectBless, 40, 100, EffectDuration) {
				t.Fatal("bless did not attach")
			}
			before := w.rng.state
			w.applySpellDamage(0, 1, SpellRule{ID: 1, Damaging: true, DamageMin: tc.dmin, DamageMax: tc.dmax}, 0)
			if n := cbDraws(t, before, w.rng.state); n != tc.draws {
				t.Fatalf("draws %d, want %d", n, tc.draws)
			}
		})
	}
	// On the original stream the empty physical spread draws nothing: one
	// accuracy draw, one spread draw.
	w := castDamageWorld(t, spEnt(1, 0, 0), spEnt(2, 3, 0))
	originalStream(w, 99)
	w.applySpellDamage(0, 1, SpellRule{ID: 1, Damaging: true, DamageMin: 10, DamageMax: 12}, 0)
	if uint32(w.RandomState()) != crtAfter(99, 2) {
		t.Fatal("original stream did not take exactly two draws")
	}
}

func TestBuildingTakesTheRecordBytesLessFive(t *testing.T) {
	// 45/50 at power 12: binary64 base 62 and spread 8 (70.0 - 62); the
	// integer formula gives 63 and 7. A zero raw draw lands 62 - 5.
	w := structureAreaWorld(t, []Structure{{ID: 1, Field42: 100, MaxHealth: 100, Width: 1, Height: 1, Attach: 1}})
	originalStream(w, crtStateFor(t, 0))
	w.applyStructureSpellAt(0, 0, SpellRule{ID: 1, Damaging: true, DamageMin: 45, DamageMax: 50}, 12)
	if got := w.structures[0].Field42; got != 43 {
		t.Fatalf("building HP %d, want 43", got)
	}
	if base, _ := oldCastDamage(45, 50, 12); 100-(base-5) == 43 {
		t.Fatal("loss control")
	}
}

func TestFirstGameHealRequestsOneAboveTheBase(t *testing.T) {
	for _, tc := range []struct {
		name       string
		dmin, dmax int32
		raw        int32
		want       int32
	}{
		{"zero spread still draws and heals base+1", 10, 10, 0, 11},
		{"lowest draw heals base+1", 10, 14, 0, 11},
		{"highest draw heals base+spread", 10, 14, 32767, 14},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := spEnt(2, 3, 0)
			target.HP = 50
			w := castDamageWorld(t, spEnt(1, 0, 0), target)
			state := crtStateFor(t, tc.raw)
			originalStream(w, state)
			w.applySpellHealing(1, SpellRule{ID: 6, Restorative: true, DamageMin: tc.dmin, DamageMax: tc.dmax}, 0)
			if got := w.entities[1].HP - 50; got != tc.want {
				t.Fatalf("healed %d, want %d", got, tc.want)
			}
			if uint32(w.RandomState()) != crtAfter(state, 1) {
				t.Fatal("Heal did not take exactly one draw")
			}
			// Loss control: the former draw base + R(spread) healed the base.
			if tc.raw == 0 && tc.dmin == tc.want {
				t.Fatal("loss control")
			}
		})
	}
}

func TestFireSacrificeKeepsLowBytesWithoutLowerClamp(t *testing.T) {
	for _, tc := range []struct {
		hp, mana   int32
		power      uint16
		base, sprd int32
	}{
		{40, 20, 50, 60, 50},
		{200, 100, 0, 255, 45},
		{-5, 0, 30, 251, 30},    // sum -5: base byte 251, spread 30.
		{65531, 0, 30, 251, 30}, // the HP word read signed is -5.
		{400, 300, 100, 255, 255},
	} {
		base, spread := sacrificeDamage(Entity{HP: tc.hp, Mana: tc.mana}, tc.power)
		if base != tc.base || spread != tc.sprd {
			t.Errorf("HP %d mana %d power %d: %d/%d, want %d/%d", tc.hp, tc.mana, tc.power, base, spread, tc.base, tc.sprd)
		}
	}
	// Loss control: the former clamp gave base 0 and spread 25 for sum -5.
	if b, s := sacrificeDamage(Entity{HP: -5}, 30); b == 0 || s == 25 {
		t.Fatal("loss control")
	}
}

func TestPoisonMagnitudeScalesInBinary64(t *testing.T) {
	// -45 * (12/30 + 1) is -62.99999999999999: -62, where the integer
	// formula gives -63.
	if got := PoisonMagnitude(-45, 12); got != -62 {
		t.Fatalf("magnitude %d, want -62", got)
	}
	if -45*(12+30)/30 == -62 {
		t.Fatal("loss control")
	}
	if got := PoisonMagnitude(-2, 100); got != -8 {
		t.Fatalf("installed magnitude at power 100: %d, want -8", got)
	}
}

func TestStoneCurseDurationReadsSignedEarthResistance(t *testing.T) {
	rule := SpellRule{ID: 20, SpellDuration: 10, EffectKind: EffectAbsorption, EffectMagnitude: 5, EffectMode: EffectDuration}
	for _, tc := range []struct {
		resistance int32
		want       uint16
	}{
		{0, 160},
		{25, 120},
		{-50, 240},   // a negative resistance lengthens: 160*150/100.
		{100, 1},     // zero is replaced by one.
		{101, 65535}, // 160*-1/100 is -1, kept as a word.
	} {
		target := spEnt(2, 3, 0)
		target.Protection[3] = tc.resistance
		w := castDamageWorld(t, spEnt(1, 0, 0), target)
		_, _, duration, _ := w.pointEffect(1, rule, 0)
		if duration != tc.want {
			t.Errorf("resistance %d: duration %d, want %d", tc.resistance, duration, tc.want)
		}
	}
}
