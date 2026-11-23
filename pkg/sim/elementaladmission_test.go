package sim

import (
	"reflect"
	"testing"
)

// TestTheElementalPartNeedsTheHitCompareInTheFirstGame is MAGIC-455 at the
// one blow resolver: the item-effect elemental part is drawn and added only
// when to-hit plus the roll exceeds the defence or the physical pair is
// empty. The auto-hit flag and a roll of 90 or more land the physical part
// alone. The second game's result for the same blows is unchanged
// (R2-ENGINE-1148).
func TestTheElementalPartNeedsTheHitCompareInTheFirstGame(t *testing.T) {
	for _, tc := range []struct {
		name          string
		alwaysHits    bool
		defence       int32
		roll          int32 // answer to the 0..200 draw; the roll is answer-100
		first, second []int32
		hp1, hp2      int32
	}{
		{name: "roll of 90 or more", defence: 1000, roll: 195,
			first: []int32{6, 200}, second: []int32{6, 200}, hp1: 89, hp2: 89},
		{name: "auto-hit flag", alwaysHits: true, defence: 1000, roll: 100,
			first: []int32{6, 200}, second: []int32{6, 200}, hp1: 87, hp2: 87},
		{name: "to-hit plus the roll above the defence", defence: 100, roll: 160,
			first: []int32{6, 200, 4}, second: []int32{6, 200, 4}, hp1: 82, hp2: 82},
	} {
		for _, second := range []bool{false, true} {
			a := cbEnt(1, 0, 0)
			a.ToHit, a.DamageBase, a.DamageSpread = 50, 10, 6
			a.AlwaysHits = tc.alwaysHits
			a.SecondaryDamage = SecondaryDamage{Base: 5, Spread: 4}
			v := cbEnt(2, 1, 0)
			v.Defence, v.Absorption = tc.defence, 2
			w := cbWorld(t, 1, a, v)
			want, wantHP := tc.first, tc.hp1
			if second {
				w.game = newGameProfile(ScriptROM2)
				want, wantHP = tc.second, tc.hp2
			}
			script := &scriptedDraws{answers: map[int32][]int32{6: {3}, 200: {tc.roll}, 4: {2}}}
			w.rng.scripted = script

			w.resolveBlow(0, 1, nil)

			if !reflect.DeepEqual(script.widths, want) {
				t.Errorf("%s (second game %v): draw widths = %v, want %v", tc.name, second, script.widths, want)
			}
			if hp := cbAt(t, w, 2).HP; hp != wantHP {
				t.Errorf("%s (second game %v): target health = %d, want %d", tc.name, second, hp, wantHP)
			}
		}
	}
}
