package sim

import "testing"

// The second game's resolver bounds no protection word: a word of 0 leaves
// the roll, a negative word raises it, one above 100 gives 0, and the result
// is held at 0 (R2-ENGINE-1830). The values are literals.
func TestSecondSpellDamageReadsTheProtectionWordUnbounded(t *testing.T) {
	for _, c := range []struct {
		protection int32
		want       int64
	}{
		{0, 40}, {30, 28}, {-50, 60}, {100, 0}, {150, 0},
	} {
		victim := spEnt(2, 1, 0)
		victim.Protection[0] = c.protection
		w := spSecondWorld(t, 3, nil, spEnt(1, 0, 0), victim)
		if got := w.secondSpellDamage(1, 40, 0, 1); got != c.want {
			t.Errorf("protection %d: damage %d, want %d", c.protection, got, c.want)
		}
	}
}

// The blow's magic part reads the target's protection signed in the second
// game, so a negative word raises it; the first game holds a native sheet's
// word to 0..100 (R2-ENGINE-1830).
func TestSecondBlowMagicPartReadsTheProtectionSigned(t *testing.T) {
	for _, c := range []struct {
		name   string
		second bool
		hp     int32
	}{
		{"second game", true, 100 - (20*150+75)/100},
		{"first game", false, 80},
	} {
		a := cbEnt(1, 0, 0)
		a.AlwaysHits, a.SecondaryDamage = true, SecondaryDamage{Base: 20}
		v := cbEnt(2, 1, 0)
		v.Protection[0] = -50
		w := cbWorld(t, 1, a, v)
		if c.second {
			w.game = newGameProfile(ScriptROM2)
		}
		w.resolveBlow(0, 1, nil)
		if got := cbAt(t, w, 2).HP; got != c.hp {
			t.Errorf("%s: target health %d, want %d", c.name, got, c.hp)
		}
	}
}
