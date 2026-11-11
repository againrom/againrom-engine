package sim

import "testing"

// blowRollSeed is the first seed whose blow roll, the second draw of a
// resolveBlow with no Bless or Curse, satisfies want.
func blowRollSeed(t *testing.T, want func(roll int64) bool) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 5000; seed++ {
		probe := cbWorld(t, seed, cbEnt(1, 0, 0), cbEnt(2, 1, 0)).rng
		probe.uniform(0)
		if roll := int64(probe.uniform(hitRollSpan)) + hitRollFloor; want(roll) {
			return seed
		}
	}
	t.Fatal("no seed below 5000 gives the wanted roll")
	return 0
}

// TestTheSecondGameMagicPartNeedsAHitByTheRoll is R2-ENGINE-478 step 5 at
// the one resolver: in the second game a blow that lands by the band or by
// AlwaysHits keeps its physical part (absorption skipped under AlwaysHits)
// and loses its magic part; a hit by to-hit plus the roll, or an empty
// physical pair, keeps it. The first game adds it on every hit. The draw
// count is the same in both games.
func TestTheSecondGameMagicPartNeedsAHitByTheRoll(t *testing.T) {
	band := blowRollSeed(t, func(r int64) bool { return r >= autoHitRoll })
	low := blowRollSeed(t, func(r int64) bool { return r < 0 })
	cases := []struct {
		name        string
		seed        uint64
		base, toHit int32
		alwaysHits  bool
		second      uint8
		first, rom2 int32
		wantDraws   int
	}{
		{"band hit", band, 10, 0, false, 0, 73, 93, 3},
		{"band hit with second pair", band, 10, 0, false, 5, 68, 88, 4},
		{"always hits", low, 10, 0, true, 0, 70, 90, 3},
		{"hit by the roll", low, 10, 2000, false, 0, 73, 73, 3},
		{"empty physical pair on a miss", low, 0, 0, false, 0, 80, 80, 3},
	}
	for _, tc := range cases {
		for _, second := range []bool{false, true} {
			name := tc.name + "/first game"
			want := tc.first
			if second {
				name, want = tc.name+"/second game", tc.rom2
			}
			t.Run(name, func(t *testing.T) {
				a := cbEnt(1, 0, 0)
				a.DamageBase, a.ToHit, a.AlwaysHits = tc.base, tc.toHit, tc.alwaysHits
				a.SecondBase = tc.second
				a.SecondaryDamage = SecondaryDamage{Base: 20, Selector: 0}
				v := cbEnt(2, 1, 0)
				v.Defence, v.Absorption = 1000, 3
				w := cbWorld(t, tc.seed, a, v)
				if second {
					w.game = newGameProfile(ScriptROM2)
				}
				before := w.rng.state

				w.resolveBlow(0, 1, nil)

				if got := cbAt(t, w, 2).HP; got != want {
					t.Errorf("target HP = %d, want %d", got, want)
				}
				if got := cbDraws(t, before, w.rng.state); got != tc.wantDraws {
					t.Errorf("resolver draws = %d, want %d", got, tc.wantDraws)
				}
			})
		}
	}
}
