package sim

import "testing"

// crtStep is an independent CRT rand(): state*214013+2531011, bits 16..30.
func crtStep(state *uint32) int32 {
	*state = *state*0x343fd + 0x269ec3
	return int32(*state>>16) & 0x7fff
}

// splitMixStep is an independent SplitMix64 draw: the state advances by the
// golden gamma and the finalizer mixes it.
func splitMixStep(state *uint64) uint64 {
	*state += 0x9E3779B97F4A7C15
	z := *state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// TestAIRangeIsTheAIModuleIdiom pins the AI families' draw to MAGIC-285's raw
// form, n*rand()/0x8000 (AI-RANGE-102), over every raw value of an original
// stream, and pins the spell hand-back threshold of MAGIC-AI-012 at exactly 30
// in 100: raw 9830 hands back and raw 9831 does not. The range wrapper the site
// used before answered rand()*101>>15, which hands back below 9734 only.
func TestAIRangeIsTheAIModuleIdiom(t *testing.T) {
	r := rng{original: true, state: 1}
	want := uint32(1)
	for i := 0; i < 1<<15; i++ {
		raw := crtStep(&want)
		n := int32(1 + i%200)
		if got := r.aiRange(n); got != raw*n/0x8000 || uint32(r.state) != want {
			t.Fatalf("draw %d: aiRange(%d) = %d at state %#x, want %d at %#x", i, n, got, r.state, raw*n/0x8000, want)
		}
	}
	handBack := 0
	for raw := int32(0); raw <= 0x7fff; raw++ {
		if raw*100/0x8000 < 30 {
			handBack++
		}
	}
	if handBack != 9831 {
		t.Fatalf("the hand-back admits %d raw values, want 9831", handBack)
	}
	// A seeded stream feeds the same arithmetic the top 15 bits of one draw.
	s := rng{state: 99}
	g := uint64(99)
	for i := 0; i < 1000; i++ {
		raw := int32(splitMixStep(&g) >> 49)
		if got := s.aiRange(100); got != raw*100/0x8000 || s.state != g {
			t.Fatalf("seeded draw %d: aiRange(100) = %d, want %d", i, got, raw*100/0x8000)
		}
	}
}

// TestAIRangeKeepsTheValueIdenticalSites pins the AI sites whose value the
// range wrapper already gave: the roam direction (eight, a power of two), the
// idle turn gate and the per-slot spell draw (raw itself) and the idle turn
// quotient (190). Each answers the same value as the range wrapper did, in
// both modes, so moving them to the idiom moves no World hash.
func TestAIRangeKeepsTheValueIdenticalSites(t *testing.T) {
	for _, original := range []bool{false, true} {
		for seed := uint64(0); seed < 2000; seed++ {
			a, b := rng{original: original, state: seed}, rng{original: original, state: seed}
			if a.aiRange(8) != b.uniform(7) || a.state != b.state {
				t.Fatalf("original %v seed %d: roam direction moved", original, seed)
			}
			if a.raw() != b.uniform(0x7fff) || a.state != b.state {
				t.Fatalf("original %v seed %d: raw draw moved", original, seed)
			}
			if a.aiRange(190) != 190*b.uniform(0x7fff)/32768 || a.state != b.state {
				t.Fatalf("original %v seed %d: idle turn quotient moved", original, seed)
			}
		}
	}
}

// TestAICastPickDrawsOnceOverOneChoice pins MAGIC-AI-012's pick, k =
// rand()*count/0x8000: with one affordable spell the original still draws once.
// The mage has no target, so the cast stops after the pick and the draws
// counted are the AI's own. A Mind above 59 adds the hand-back draw before it.
func TestAICastPickDrawsOnceOverOneChoice(t *testing.T) {
	rule := SpellRule{ID: 1, ManaCost: 1, School: 1, MaxRange: 1, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	for _, tc := range []struct {
		mind  int32
		draws int
	}{{30, 1}, {60, 2}} {
		for _, original := range []bool{false, true} {
			mage := spMage(1, 2, 2, 60, 100, 100, 1<<1)
			mage.Book.State = BookPresent
			mage.Book.Slots[0] = BookSpell{5, 0, 1}
			mage.Owner, mage.Mind, mage.Mana = 2, tc.mind, 3
			w := hlWorld(t, 31, acEnemies(t), []SpellRule{rule}, mage)
			// The first state whose first draw stays above the hand-back.
			state := uint64(1)
			for probe := (rng{original: original, state: state}); probe.raw() < 9831; probe = (rng{original: original, state: state}) {
				state++
			}
			w.rng = rng{original: original, state: state}
			want := rng{original: original, state: w.rng.state}
			for range tc.draws {
				want.raw()
			}
			if w.aiCast(0, nil) {
				t.Fatalf("mind %d original %v: a cast began with no target", tc.mind, original)
			}
			if w.rng.state != want.state {
				t.Fatalf("mind %d original %v: state %#x after the AI's choice, want %d draws to %#x", tc.mind, original, w.rng.state, tc.draws, want.state)
			}
		}
	}
}
