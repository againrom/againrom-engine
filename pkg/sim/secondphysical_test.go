package sim

import (
	"bytes"
	"testing"
)

// Literal scalar answers from HERO-DMG2-029/HERO-CLAMP-030, not the resolver.
// Seed17's primary hit roll is -22. All spreads here are zero on purpose:
// each present component must still consume its own draw.
func TestSecondPhysicalIndependentComponent(t *testing.T) {
	for _, tc := range []struct {
		name               string
		hit, absorb, water int32
		res                uint8
		base               uint8
		wantHP             int32
		draws              int
	}{
		{"primary miss", 0, 0, 25, 0, 20, 85, 3},
		{"primary absorbed", 2000, 1000, 25, 255, 20, 81, 4},
		{"negative protection amplifies", 0, 1000, -25, 255, 20, 75, 3},
		{"over protection clamps", 0, 0, 150, 0, 20, 100, 3},
		{"signed word minimum", 0, 0, -32768, 0, 1, -229, 3},
		{"negative primary cancels", 2000, 0, 25, 255, 20, 91, 4},
		{"total clamps", 2000, 0, 100, 255, 20, 100, 4},
		{"zero pair miss legacy", 0, 0, 25, 0, 0, 100, 2},
		{"zero pair hit legacy", 2000, 0, 25, 0, 0, 89, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := cbEnt(1, 1, 1), cbEnt(2, 2, 1)
			a.DamageBase, a.ToHit, a.XPSlot = 7, tc.hit, 1
			a.SecondBase = tc.base
			a.SecondaryDamage = SecondaryDamage{Base: 8, Selector: 0}
			v.Defence, v.Absorption = 1000, tc.absorb
			v.Protection = [5]int32{50, tc.water, 80, 100, 100}
			v.Resistance[0] = tc.res
			w := cbWorld(t, 17, a, v)
			before := w.rng.state
			w.resolveBlow(0, 1, nil)
			if got := cbAt(t, w, 2).HP; got != tc.wantHP {
				t.Fatalf("HP=%d want%d", got, tc.wantHP)
			}
			if got := cbDraws(t, before, w.rng.state); got != tc.draws {
				t.Fatalf("draws=%d want%d", got, tc.draws)
			}
		})
	}
}

func TestSecondPhysicalSpreadAndDrawOrder(t *testing.T) {
	a, v := cbEnt(1, 1, 1), cbEnt(2, 2, 1)
	a.SecondSpread, a.SecondaryDamage = 255, SecondaryDamage{Spread: 255}
	v.Defence = 1000
	w := cbWorld(t, 17, a, v)
	w.resolveBlow(0, 1, nil)
	// Independent SplitMix64 vector: draw3 maps to 84, draw4 to 57 in 0..255.
	if got := cbAt(t, w, 2).HP; got != -41 {
		t.Fatalf("spread-only ordered components HP=%d want -41", got)
	}
}

func TestSecondPhysicalCommandAndMidChargeContinuation(t *testing.T) {
	a, v := cbEnt(1, 1, 1), cbEnt(2, 2, 1)
	a.Owner, a.Group, a.DamageBase, a.AttackCharge = 1, 1, 7, 12
	a.SecondBase, a.SecondSpread = 20, 7
	v.Defence, v.Protection[1] = 1000, 25
	w := cbWorld(t, 17, a, v)
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	if cbAt(t, w, 1).AttackPhase != AttackCharging {
		t.Fatal("command did not start charge")
	}
	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("continuation differs at %d", i)
		}
		if hp := cbAt(t, w, 2).HP; hp != 100 {
			if hp >= 100 {
				t.Fatal("second pair did not damage target")
			}
			return
		}
	}
	t.Fatal("no actual attack damage")
}

func TestSecondPhysicalCanonicalBytesAndReplacement(t *testing.T) {
	w := cbWorld(t, 17, cbEnt(1, 1, 1))
	base := mustMarshal(t, w)
	hashes := []uint64{w.Hash()}
	for _, pair := range [][2]uint8{{255, 0}, {0, 255}, {21, 9}} {
		if !w.SetCombat(1, CombatBlock{Reach: 1, SecondBase: pair[0], SecondSpread: pair[1]}) {
			t.Fatal("set")
		}
		form := mustMarshal(t, w)
		at := 34 + 3*256 + 454
		if form[at] != pair[0] || form[at+1] != pair[1] || bytes.Equal(base, form) {
			t.Fatal("pair not canonical")
		}
		for _, hash := range hashes {
			if w.Hash() == hash {
				t.Fatal("new byte did not move hash")
			}
		}
		hashes = append(hashes, w.Hash())
		var back World
		if err := back.UnmarshalBinary(form); err != nil || back.Hash() != w.Hash() {
			t.Fatal("roundtrip", err)
		}
	}
	if !w.SetCombat(1, CombatBlock{}) || cbAt(t, w, 1).SecondBase != 0 || cbAt(t, w, 1).SecondSpread != 0 {
		t.Fatal("zero replacement did not clear")
	}
	if !w.SetDerived(1, DerivedBlock{Combat: CombatBlock{SecondBase: 7, SecondSpread: 13}}) || cbAt(t, w, 1).SecondSpread != 13 {
		t.Fatal("full derived replacement lost pair")
	}
}
