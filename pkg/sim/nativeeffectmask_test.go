package sim

import "testing"

func nativeEffectMaskCold(t *testing.T, w *World, want NativeActorBasis) {
	t.Helper()
	if w.entities[0].NativeBasis != want {
		t.Fatal("attachment changed an independent observation", w.entities[0].NativeBasis, want)
	}
	actions := w.Actions()
	current := actions.Actors[0].Current
	if current == nil || want.HasValues() && (current.NativeBasis == nil || *current.NativeBasis != want) || !want.HasValues() && current.NativeBasis != nil {
		t.Fatal("current action changed mask values or absence")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || cold.entities[0].NativeBasis != want {
		t.Fatal("native cold form lost the exact mask or hash", err)
	}
	if want.ScalarIsKnown(ScalarU144) {
		lost := want
		lost.Scalars[ScalarU144] ^= 1 << 31
		if err := cold.RestoreNativeActorBases([]NativeActorBasisRecord{{ID: 0, Basis: lost}}); err != nil {
			t.Fatal(err)
		}
		if cold.Hash() == w.Hash() {
			t.Fatal("independent observed mask bit loss did not change the hash")
		}
	}
}

func TestNativeEffectMaskExpiryKeepsUnrelatedObservedBits(t *testing.T) {
	w := nativeBasisWorld(t)
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << ScalarU144}
	basis.Scalars[ScalarU144] = 1<<20 | 1<<31
	w.entities[0].NativeBasis = basis
	// The loaded attachment is already applied; its zero delta is independent.
	w.attached = []attachedEffect{{Target: 0, Spell: 20, Kind: EffectSpeed, Mode: EffectDuration, Remaining: 2}}
	nativeEffectMaskCold(t, w, basis)
	Step(w, nil)
	if len(w.attached) != 1 || w.attached[0].Remaining != 1 {
		t.Fatal("fixture did not retain its first countdown tick")
	}
	nativeEffectMaskCold(t, w, basis)
	Step(w, nil)
	if len(w.attached) != 0 {
		t.Fatal("fixture did not naturally expire")
	}
	basis.Scalars[ScalarU144] &^= 1 << 20
	nativeEffectMaskCold(t, w, basis)
}

func TestNativeEffectMaskNewAttachmentDoesNotPromoteUnknownState(t *testing.T) {
	for _, tc := range []struct {
		name  string
		basis NativeActorBasis
	}{
		{"known", NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << ScalarU144, Scalars: [ScalarCount]uint32{ScalarU144: 1 << 31}}},
		{"knownzero", NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << ScalarU144}},
		{"unknownmask", NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << ScalarU138, Scalars: [ScalarCount]uint32{ScalarU144: 1 << 31}}},
		{"maskzero", NativeActorBasis{ScalarsPresent: true, Scalars: [ScalarCount]uint32{ScalarU144: 1 << 31}}},
		{"absent", NativeActorBasis{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := nativeBasisWorld(t)
			w.entities[0].NativeBasis = tc.basis
			if !w.attachEffect(0, 3, SpellRule{ID: 24}, EffectSpeed, 3, 2, EffectDuration) {
				t.Fatal("new native attachment was refused")
			}
			want := tc.basis
			if want.ScalarIsKnown(ScalarU144) {
				want.Scalars[ScalarU144] |= 1 << 24
			}
			nativeEffectMaskCold(t, w, want)
			Step(w, nil)
			Step(w, nil)
			if len(w.attached) != 0 || w.entities[0].Speed != 10 {
				t.Fatal("new attachment did not expire and undo its delta")
			}
			nativeEffectMaskCold(t, w, tc.basis)
		})
	}
}
