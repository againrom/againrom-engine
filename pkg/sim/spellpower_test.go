package sim

import (
	"math/big"
	"testing"
)

// oldLastingTicks is the duration word as it stood while power stopped at 100,
// kept to prove that no power up to 100 changed.
func oldLastingTicks(power int32, base int32, scale durationScale) uint16 {
	if base <= 0 {
		return 0
	}
	factor := durationSlowQ56[power]
	if scale == durationFast {
		factor = durationFastQ56[power]
	}
	x := new(big.Int).Mul(new(big.Int).SetUint64(uint64(base)*16), new(big.Int).SetUint64(factor))
	x.Rsh(x, 56)
	return uint16(x.Uint64())
}

func oldLastingTicksOverflows(power int32, base int32, scale durationScale) bool {
	factor := durationSlowQ56[power]
	if scale == durationFast {
		factor = durationFastQ56[power]
	}
	x := new(big.Int).Mul(new(big.Int).SetUint64(uint64(base)*16), new(big.Int).SetUint64(factor))
	x.Rsh(x, 56)
	return x.Cmp(big.NewInt(durationTickCeiling)) > 0
}

// exactTicks is floor(base*16*ratio^power) in exact rational arithmetic,
// saturating at the word ceiling.
func exactTicks(power, base int, num, den int64) uint16 {
	v := new(big.Rat).SetInt64(int64(base) * 16)
	r := big.NewRat(num, den)
	for i := 0; i < power; i++ {
		v.Mul(v, r)
	}
	q := new(big.Int).Quo(v.Num(), v.Denom())
	if q.Cmp(big.NewInt(durationTickCeiling)) > 0 {
		return durationTickCeiling
	}
	return uint16(q.Uint64())
}

func TestSpellPowerDurationsAboveHundredMatchTheExactLaw(t *testing.T) {
	for _, base := range []int{1, 3, 5, 10, 30, 160, 347, 1000} {
		for power := 101; power <= spellPowerMax; power++ {
			if got, want := lastingTicks(int32(power), int32(base), durationSlow), exactTicks(power, base, 41, 40); got != want {
				t.Fatalf("slow base %d power %d = %d, want %d", base, power, got, want)
			}
			if got, want := lastingTicks(int32(power), int32(base), durationFast), exactTicks(power, base, 21, 20); got != want {
				t.Fatalf("fast base %d power %d = %d, want %d", base, power, got, want)
			}
		}
	}
}

func TestSpellPowerDurationsUpToHundredAreUnchanged(t *testing.T) {
	for base := int32(1); base <= 340; base++ {
		for power := int32(0); power <= 100; power++ {
			for _, scale := range []durationScale{durationSlow, durationFast} {
				// The word used to wrap above 65535; it saturates now.
				want := oldLastingTicks(power, base, scale)
				if oldLastingTicksOverflows(power, base, scale) {
					want = durationTickCeiling
				}
				if got := lastingTicks(power, base, scale); got != want {
					t.Fatalf("base %d power %d scale %d = %d, was %d", base, power, scale, got, want)
				}
			}
		}
	}
}

func TestSpellPowerDurationSaturatesInsteadOfWrapping(t *testing.T) {
	prev := uint16(0)
	for power := int32(0); power <= spellPowerMax; power++ {
		got := lastingTicks(power, 30, durationSlow)
		if got < prev {
			t.Fatalf("slow duration fell from %d to %d at power %d", prev, got, power)
		}
		prev = got
	}
	for _, power := range []int32{200, 255, 1000} {
		if got := lastingTicks(power, 30, durationSlow); got != 65535 {
			t.Fatalf("protection duration at %d = %d, want 65535", power, got)
		}
	}
	// A base large enough to overflow the word saturates at any power.
	for _, power := range []int32{0, 100} {
		if got := lastingTicks(power, 5000, durationFast); got != 65535 {
			t.Fatalf("oversized base at %d = %d, want 65535", power, got)
		}
	}
}

func TestSpellPowerReferenceDurations(t *testing.T) {
	for _, tc := range []struct {
		name  string
		power int32
		base  int32
		scale durationScale
		seg   bool
		want  uint16
	}{
		{"protection 100", 100, 30, durationSlow, false, 5670},
		{"protection 120", 120, 30, durationSlow, false, 9291},
		{"protection 255", 255, 30, durationSlow, false, 65535},
		{"shield 100", 100, 15, durationSlow, false, 2835},
		{"shield 170", 170, 15, durationSlow, false, 15968},
		{"haste 255", 255, 15, durationSlow, false, 65535},
		{"invisibility 100", 100, 3, durationFast, true, 6312},
		{"invisibility 150", 150, 3, durationFast, true, 6862},
		{"invisibility 200", 200, 3, durationFast, true, 12624},
		{"invisibility 255", 255, 3, durationFast, true, 13326},
		{"stone curse 100", 100, 10, durationSlow, true, 1890},
		{"stone curse 150", 150, 10, durationSlow, true, 2439},
		{"stone curse 200", 200, 10, durationSlow, true, 3780},
		{"stone curse 255", 255, 10, durationSlow, true, 4402},
	} {
		got := lastingTicks(tc.power, tc.base, tc.scale)
		if tc.seg {
			got = segmentedTicks(tc.power, tc.base, tc.scale)
		}
		if got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSegmentedDurationIsWholeDurationsAtHundredPlusTheRemainder(t *testing.T) {
	for _, tc := range []struct {
		base  int32
		scale durationScale
	}{{3, durationFast}, {10, durationSlow}, {1, durationSlow}} {
		d := func(p int32) uint16 { return lastingTicks(p, tc.base, tc.scale) }
		if got, want := segmentedTicks(101, tc.base, tc.scale), d(100)+d(1); got != want {
			t.Errorf("base %d at 101 = %d, want %d", tc.base, got, want)
		}
		if got, want := segmentedTicks(200, tc.base, tc.scale), 2*d(100); got != want {
			t.Errorf("base %d at 200 = %d, want %d", tc.base, got, want)
		}
		if got, want := segmentedTicks(201, tc.base, tc.scale), 2*d(100)+d(1); got != want {
			t.Errorf("base %d at 201 = %d, want %d", tc.base, got, want)
		}
		if got, want := segmentedTicks(250, tc.base, tc.scale), 2*d(100)+d(50); got != want {
			t.Errorf("base %d at 250 = %d, want %d", tc.base, got, want)
		}
		for p := int32(0); p <= 100; p++ {
			if segmentedTicks(p, tc.base, tc.scale) != d(p) {
				t.Fatalf("base %d power %d: segmented differs below 101", tc.base, p)
			}
		}
	}
	if got := segmentedTicks(255, 3000, durationFast); got != 65535 {
		t.Errorf("a segmented sum above the word = %d, want 65535", got)
	}
}

func spPowerRows() []SpellRule {
	return []SpellRule{
		{ID: 5, ManaCost: 1, School: 1, MaxRange: 8, TargetsUnit: true, Defensive: true, SpellDuration: 30, EffectKind: EffectProtectionFire, EffectMode: EffectDuration},
		{ID: 15, ManaCost: 1, School: 4, MaxRange: 8, TargetsUnit: true, Defensive: true, SpellDuration: 1, EffectKind: EffectInvisible, EffectMode: EffectDuration},
		{ID: 20, ManaCost: 1, School: 3, MaxRange: 8, TargetsUnit: true, SpellDuration: 10, EffectKind: EffectSpeed, EffectMode: EffectDuration, EffectMagnitude: -5},
		{ID: 24, ManaCost: 1, School: 4, MaxRange: 8, TargetsUnit: true, Defensive: true, SpellDuration: 10, EffectKind: EffectSpeed, EffectMode: EffectDuration},
		{ID: 28, ManaCost: 1, School: 3, MaxRange: 8, TargetsUnit: true, SpellDuration: 10, EffectKind: EffectSpeed, EffectMode: EffectDuration},
	}
}

func attachedRemaining(t *testing.T, w *World, target EntityID, spell uint16) uint16 {
	t.Helper()
	i, ok := effectIndex(w.attached, target, spell)
	if !ok {
		t.Fatalf("no effect of spell %d on %d", spell, target)
	}
	return w.attached[i].Remaining
}

func spPowerCaster(spell uint16) Entity {
	caster := effectMage(1, 2, 2, 1<<spell)
	caster.Mind = 285
	caster.MaxMana = 1000
	caster.Mana = 1000
	return caster
}

// A cast through the book route at skill+Mind of 285 lands the effect at power
// 255, and the stored word is the segmented or saturated duration.
func TestCastAtPower255AttachesTheSegmentedAndSaturatedDurations(t *testing.T) {
	for _, tc := range []struct {
		spell uint16
		want  uint16
		mag   int32
	}{
		{5, 65535, 100},
		{15, 13326, 1},
		{20, 4402, -5},
		{24, 65535, 18},
		{28, 65535, -18},
	} {
		victim := spEnt(2, 3, 2)
		victim.Speed, victim.TokenSize = 30, 1
		w := hlWorld(t, 0x1323, Relations{}, spPowerRows(), spPowerCaster(tc.spell), victim)
		spRunCast(w, Cast(1, 2, SpellID(tc.spell)))
		if got := attachedRemaining(t, w, 2, tc.spell); got != tc.want {
			t.Errorf("spell %d at power 255: remaining %d, want %d", tc.spell, got, tc.want)
		}
		if tc.spell == 5 {
			if got := spAt(t, w, 2).Protection[0]; got != 100 {
				t.Errorf("protection at power 255 reads %d, want the per-element clamp 100", got)
			}
			continue
		}
		if mag, _ := w.attachedMagnitude(2, tc.spell); mag != tc.mag {
			t.Errorf("spell %d at power 255: magnitude %d, want %d", tc.spell, mag, tc.mag)
		}
	}
}

// Stone Curse's duration is cut by the target's earth protection after the
// segments are summed.
func TestStoneCurseCutByEarthProtectionAfterTheSegments(t *testing.T) {
	victim := spEnt(2, 3, 2)
	victim.TokenSize = 1
	victim.Protection[3] = 50
	w := hlWorld(t, 0x1323, Relations{}, spPowerRows(), spPowerCaster(20), victim)
	spRunCast(w, Cast(1, 2, 20))
	if got, want := attachedRemaining(t, w, 2, 20), uint16(4402*50/100); got != want {
		t.Fatalf("stone curse through 50 earth protection = %d, want %d", got, want)
	}
}

// Slow at power 255 (speed -18) and Haste (speed +18) move a unit through the
// ordinary movement path: the slowed unit stays rated, at the floor, and the
// distances covered follow the resulting speed.
func TestHasteAndSlowAtPower255MoveUnitsThroughTheMovementPath(t *testing.T) {
	run := func(spell uint16, speed int32) (int32, int32) {
		victim := spEnt(3, 3, 2)
		victim.Speed, victim.TokenSize = speed, 1
		w, err := NewStockedSpelledWorld(7, Bounds{Width: 60, Height: 60}, ModeCanonical, Terrain{},
			[]Entity{spPowerCaster(spell), victim}, nil, Relations{}, nil, nil, spPowerRows())
		if err != nil {
			t.Fatal(err)
		}
		if spell != 0 {
			spRunCast(w, Cast(1, 3, SpellID(spell)))
		}
		now := spAt(t, w, 3)
		if !rated(now) {
			t.Fatalf("spell %d left speed %d, which is unrated", spell, now.Speed)
		}
		Step(w, []Command{MoveTo(3, CellPoint{X: 3, Y: 55})})
		for n := 0; n < 400; n++ {
			Step(w, nil)
		}
		return now.Speed, spAt(t, w, 3).Y
	}
	baseSpeed, baseY := run(0, 30)
	slowSpeed, slowY := run(28, 30)
	hasteSpeed, hasteY := run(24, 30)
	if baseSpeed != 30 || slowSpeed != 12 || hasteSpeed != 48 {
		t.Fatalf("speeds base %d slow %d haste %d, want 30, 12, 48", baseSpeed, slowSpeed, hasteSpeed)
	}
	if !(slowY > 2 && slowY < baseY && baseY < hasteY) {
		t.Fatalf("rows after 400 ticks: slow %d, base %d, haste %d, want slow < base < haste with slow moving", slowY, baseY, hasteY)
	}
	floorSpeed, floorY := run(28, 5)
	if floorSpeed != minEffectSpeed || floorY <= 2 {
		t.Fatalf("speed-5 unit under power-255 Slow: speed %d, row %d, want the floor %d and some movement", floorSpeed, floorY, minEffectSpeed)
	}
}

func TestBookRootRangeMayBeTheReachAtAnyHigherPower(t *testing.T) {
	w := &World{spells: []SpellRule{{ID: 24, MaxRange: 8}, {ID: 26, MaxRange: 5}}}
	derived := SourceItemSpell{Present: true, ID: 24, Range: 8, ManaCost: 30}
	for _, tc := range []struct {
		name  string
		saved SourceItemSpell
		want  bool
	}{
		{"equal", derived, true},
		{"reach at a higher power", SourceItemSpell{Present: true, ID: 24, Range: 9, ManaCost: 30}, true},
		{"reach at the bound", SourceItemSpell{Present: true, ID: 24, Range: 8 + 255/30, ManaCost: 30}, true},
		{"beyond the bound", SourceItemSpell{Present: true, ID: 24, Range: 8 + 255/30 + 1, ManaCost: 30}, false},
		{"below the derived reach", SourceItemSpell{Present: true, ID: 24, Range: 7, ManaCost: 30}, false},
		{"other field differs", SourceItemSpell{Present: true, ID: 24, Range: 9, ManaCost: 31}, false},
	} {
		if got := w.bookRootValueMatches(tc.saved, derived); got != tc.want {
			t.Errorf("%s: %t, want %t", tc.name, got, tc.want)
		}
	}
}
