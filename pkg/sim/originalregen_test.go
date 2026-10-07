package sim

import (
	"reflect"
	"testing"
)

// Literal outcomes use the signed-word/dword/store rules of
// SAV-REGENWIDTH-528 and SAV-REGENSTORE-529. No expected value calls the
// production formula. The overflow cases distinguish widened multiplication,
// widened accumulator, pre-narrowing clamp and negative-remainder normalization.
func TestOriginalProfile1107SignedWordRegeneration(t *testing.T) {
	for _, tc := range []struct {
		name                                       string
		current, maximum, period, modifier, factor int32
		rest                                       uint8
		want                                       int32
		wantRest                                   uint8
	}{
		{"word before bound", 32766, 32767, 1, 0, 2, 0, 32764, 0},
		{"negative period", 10, 100, -3, 0, 2, 0, -56, 190},
		{"negative modifier", 0, 101, 1, -101, 1, 0, -1, 255},
		{"unsigned remainder reload", -1, 101, 1, -101, 1, 255, 0, 54},
		{"wrapped product", 10, 32767, 1, 32767, 2, 0, 19669, 238},
		{"wrapped accumulator", 32766, 32767, 1, 32668, 2, 0, -12453, 172},
		{"raw pool words", 65530, 65535, 1, 0, 1, 0, -7, 0},
		{"raw signed maximum gate", 1, 65535, 1, 0, 2, 255, 1, 255},
		{"upper bound keeps remainder", 44, 45, 20, 0, 1, 7, 45, 32},
		{"health zero period", 10, 100, 0, 0, 2, 199, 10, 199},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, rest := tc.current, tc.rest
			regenerateCurrentWord(&got, tc.maximum, tc.period, &rest, tc.factor, tc.modifier, 1)
			if got != tc.want || rest != tc.wantRest {
				t.Fatalf("pool/remainder=%d/%d want %d/%d", got, rest, tc.want, tc.wantRest)
			}
		})
	}
}

func TestOriginalProfile1107HealthCrossingStillRunsManaAndNativeDeath(t *testing.T) {
	a := Entity{ID: 1, X: 1, Y: 1, HP: 1, MaxHP: 101, Mana: 0, MaxMana: 101, DyingTime: 30, Speed: 1}
	w := rgWorld(t, a)
	if err := w.ImportOriginalActorProfiles([]OriginalActorProfile{{ID: 1, HealthPeriod: 1, ManaPeriod: 1, HealthRegeneration: -101, ManaRegeneration: -101}}); err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 6, Y: 6}})
	if !w.Entities()[0].HasTarget {
		t.Fatal("new move was not admitted")
	}
	for w.Tick() <= 12 {
		Step(w, nil)
	}
	e := w.Entities()[0]
	if e.HP != -1 || e.HealthHundredths != 254 || e.Mana != -1 || e.ManaHundredths != 255 || e.HasTarget || e.Transit != 0 || e.Decay != DecayFallen {
		t.Fatalf("signed health/mana and native death: %+v", e)
	}
	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal("post-regeneration native save", err)
	}
	for i := 0; i < 96; i++ {
		a, b := StepReported(w, nil), StepReported(&back, nil)
		if !reflect.DeepEqual(a, b) || w.Hash() != back.Hash() {
			t.Fatal("signed post-death continuation", i)
		}
	}
}

func TestOriginalProfile1107ZeroManaFaultPolicyAndNativeRetirement(t *testing.T) {
	w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 100, Mana: 100, MaxMana: 100})
	if err := w.ImportOriginalActorProfiles([]OriginalActorProfile{{ID: 1}}); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	if err := w.ImportOriginalActorPools([]OriginalActorPools{{ID: 1, HP: 20, MaxHP: 100, Mana: 100, MaxMana: 100}}); err == nil || w.Hash() != before {
		t.Fatal("full mana admitted with future zero divisor", err)
	}
	if err := w.ImportOriginalActorPools([]OriginalActorPools{{ID: 1, HP: 20, MaxHP: 100}}); err != nil {
		t.Fatal("saved max0 was checked against fresh max100", err)
	}
	if err := w.ImportOriginalActorProfiles([]OriginalActorProfile{{ID: 1, HealthPeriod: -3, ManaPeriod: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalActorPools([]OriginalActorPools{{ID: 1, HP: 20, MaxHP: 100, Mana: 100, MaxMana: 100}}); err != nil {
		t.Fatal(err)
	}
	invalid := mustMarshal(t, w)
	// Entity+139 is the existing mana period dword; form73's provenance
	// remains source-current. This faulting native candidate must not load.
	start := 34 + 3*8*8
	for j := 0; j < 4; j++ {
		invalid[start+139+j] = 0
	}
	var rejected World
	if err := rejected.UnmarshalBinary(invalid); err == nil {
		t.Fatal("native zero-divisor input admitted")
	}
	// Retirement changes authority, not a claim to original re-derivation.
	if !w.SetCombat(1, CombatBlock{Reach: 1, AttackCharge: 1}) {
		t.Fatal("native rearm refused")
	}
	for i := 0; i < 13; i++ {
		Step(w, nil)
	}
	if e := w.Entities()[0]; e.CurrentProfileBasis != ProfileNativeRetired || e.HP != 20 {
		t.Fatal("retired negative period did not retain native policy", e)
	}
}
