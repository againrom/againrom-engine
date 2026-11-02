package base

import "testing"

// Each edition's identity source agrees with its hero identity: the first
// game binds by the role test and carries none, the second binds by party
// identity from a nonzero hero identity.
func TestEditionHeroBandRow(t *testing.T) {
	for _, tc := range []struct {
		game Game
		band HeroBand
	}{{GameROM1, HeroBandRole}, {GameROM2, HeroBandIdentity}} {
		e := tc.game.Edition()
		if e.HeroBand != tc.band {
			t.Errorf("%s: HeroBand = %d, want %d", tc.game, e.HeroBand, tc.band)
		}
		if (e.HeroBand == HeroBandIdentity) != (e.HeroIdentity != 0) {
			t.Errorf("%s: HeroBand %d with HeroIdentity %d", tc.game, e.HeroBand, e.HeroIdentity)
		}
	}
}
