package sim

import (
	"reflect"
	"testing"
)

func TestOriginalProfile1107ThirdProtectionPolicyAndNativeContinuation(t *testing.T) {
	for _, tc := range []struct {
		name               string
		attacker, target   CurrentProfileBasis
		protection, wantHP int32
	}{
		{"source negative", ProfileNative, ProfileOriginalCurrent, -50, 9970},
		{"source signed minimum", ProfileNative, ProfileOriginalCurrent, -32768, 3426},
		{"source above hundred", ProfileNative, ProfileOriginalCurrent, 150, 10000},
		{"source signed maximum", ProfileNative, ProfileOriginalCurrent, 32767, 10000},
		{"native negative unchanged", ProfileOriginalCurrent, ProfileNative, -50, 9980},
		{"retired negative unchanged", ProfileOriginalCurrent, ProfileNativeRetired, -50, 9980},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := cbFighter(1, 0, 0, 0, 10)
			a.DamageBase = 0
			a.SecondaryDamage = SecondaryDamage{Base: 20}
			a.CurrentProfileBasis = tc.attacker
			v := cbEnt(2, 1, 0)
			v.HP, v.MaxHP = 10000, 10000
			v.CurrentProfileBasis, v.Protection[0] = tc.target, tc.protection
			w := cbWorld(t, 17, a, v)
			Step(w, []Command{cbOrder(1, 2)})
			if got := cbAt(t, w, 2).HP; got != tc.wantHP {
				t.Fatalf("first third-component blow HP%d want%d", got, tc.wantHP)
			}
			var back World
			if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil || back.Hash() != w.Hash() {
				t.Fatal("native protection-policy roundtrip", err)
			}
			for i := 0; i < 64; i++ {
				left, right := StepReported(w, nil), StepReported(&back, nil)
				if !reflect.DeepEqual(left, right) || w.Hash() != back.Hash() {
					t.Fatal("protection-policy continuation", i)
				}
			}
		})
	}
}
