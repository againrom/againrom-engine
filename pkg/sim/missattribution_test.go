package sim

import "testing"

// A swing writes the last attacker whether it lands, misses or is absorbed to
// nothing: the damage resolver stores the attacker after the damage
// computation with no health or result test (AI-641, HERO-AUTOHIT-031).
func TestEverySwingWritesTheLastAttacker(t *testing.T) {
	for _, tc := range []struct {
		name       string
		second     bool
		absorption int32
		roll       int32 // scripted draw; 0 is the lowest roll, a miss
	}{
		{"miss", false, 0, 0},
		{"absorbed", false, 200, 199},
		{"miss, second game", true, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := cbEnt(1, 5, 5)
			a.Owner, a.TypeID, a.DamageBase, a.Reach = 2, HumanTypeID, 3, 1
			v := cbEnt(2, 5, 6)
			v.Owner, v.TypeID, v.Defence, v.Absorption = 3, 10, 100, tc.absorption
			w := cbWorld(t, 1, a, v)
			if tc.second {
				w.game = newGameProfile(ScriptROM2)
			}
			w.rng.scripted = &scriptedDraws{answers: map[int32][]int32{hitRollSpan: {tc.roll}}}
			w.resolveBlow(0, 1, nil)
			got := cbAt(t, w, 2)
			if got.HP != 100 || !got.HasKillCredit || got.KillCreditSource != 1 {
				t.Fatalf("HP %d credit %v from %d, want a swing with no damage that records attacker 1", got.HP, got.HasKillCredit, got.KillCreditSource)
			}
		})
	}
}
