package game

import (
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func legacyRawFixture(t *testing.T) (*FrontEnd, sav.DocumentData, uint16) {
	t.Helper()
	f, raw := nativeSubjectFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal(err)
	}
	var object uint16
	for _, row := range a.Bindings {
		if !row.Structure && row.ID == 0 {
			object = row.Object
		}
	}
	for i := range a.Actions.Actors {
		if a.Actions.Actors[i].Current != nil {
			a.Actions.Actors[i].Current.NativeBasis = nil
		}
	}
	a.NativeBasisWires, a.RemovedNativeBases, a.HeldNativeBasisWires = nil, nil, false
	a.NativeHistoryVersion = 0
	value, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, value); err != nil {
		t.Fatal(err)
	}
	return f, doc, object
}

func TestNativeObservationVersionPreservesCurrentAvailabilityThroughSAVE(t *testing.T) {
	for _, tc := range []struct {
		name   string
		legacy bool
		basis  sim.NativeActorBasis
	}{
		{"modernabsence", false, sim.NativeActorBasis{}},
		{"modernmaskzero", false, sim.NativeActorBasis{BasePresent: true, Base: [24]byte{22: 0xa5}, ModifierPresent: true, Modifier: [64]byte{40: 0xc7}, DefencePresent: true, Defence: [22]byte{16: 50}}},
		{"modernknownzero", false, sim.NativeActorBasis{BasePresent: true, BaseKnown: 1, Base: [24]byte{22: 0xa5}, ModifierPresent: true, ModifierKnown: 1, Modifier: [64]byte{40: 0xc7}, DefencePresent: true, DefenceKnown: 1 << 16}},
		{"legacypresentpartial", true, sim.NativeActorBasis{BasePresent: true, BaseKnown: 1, Base: [24]byte{22: 0xa5}, ModifierPresent: true, Modifier: [64]byte{40: 0xc7}, BodyPresent: true, Body: 9, AttackPresent: true, Attack: [24]byte{23: 0x5a}, DefencePresent: true, Defence: [22]byte{16: 50}, ScalarsPresent: true, Scalars: [sim.ScalarCount]uint32{sim.ScalarU8E: 0xabcdef01}, BlockPresent: true, Block: [10]byte{5: 0x59}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := nativeSubjectFixture(t)
			if tc.basis.HasValues() {
				if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 0, Basis: tc.basis}}); err != nil {
					t.Fatal(err)
				}
			} else {
				actions := f.live.world.Actions()
				for i := range actions.Actors {
					if actions.Actors[i].Entity == 0 && actions.Actors[i].Current != nil {
						actions.Actors[i].Current.NativeBasis = nil
					}
				}
				if err := f.live.world.RestoreActions(actions, nil); err != nil {
					t.Fatal(err)
				}
			}
			raw, doc, _ := saveCurrentEffect(t, f)
			if tc.legacy {
				a, err := readCurrentActions(&doc)
				if err != nil {
					t.Fatal(err)
				}
				a.NativeHistoryVersion = 0
				leaf, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
					t.Fatal(err)
				}
				raw, err = sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
			}
			cold := nativeSubjectCold(t, f, raw)
			e, ok := cold.live.world.Entity(0)
			if !ok || e.NativeBasis != tc.basis || e.ActorLoad.Source != (sim.SourceActor{}) || e.SourceBinding != (sim.SourceBinding{}) {
				t.Fatal("current presence/mask/zero/residue or native admission changed", e.NativeBasis, tc.basis)
			}
			sim.Step(cold.live.world, []sim.Command{sim.MoveTo(0, sim.CellPoint{X: 15, Y: 17})})
			sim.Step(cold.live.world, nil)
			after, _ := cold.live.world.Entity(0)
			if after.NativeBasis != tc.basis {
				t.Fatal("ordinary action/ticks reseeded current availability")
			}
			next, _, _ := saveCurrentEffect(t, cold)
			again := nativeSubjectCold(t, f, next)
			restored, _ := again.live.world.Entity(0)
			if restored.NativeBasis != tc.basis || again.live.world.Hash() != cold.live.world.Hash() {
				t.Fatal("next SAVE/cold LOAD lost represented availability")
			}
		})
	}
}

func TestLegacyNativeRawObservationsParticipateInCurrentState(t *testing.T) {
	f, doc, object := legacyRawFixture(t)
	base, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	before := nativeSubjectCold(t, f, base).live.world.Hash()
	for _, tc := range []struct {
		name     string
		at, size int
	}{{"U114", 22, 24}, {"UD4", 40, 64}, {"UA6", 22, 24}, {"UBE", 4, 22}, {"UBE", 16, 22}} {
		t.Run(tc.name+string(rune('A'+tc.at)), func(t *testing.T) {
			changed, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := savedActorRaw(&changed.Objects[object-1], tc.name, tc.size)
			if err != nil {
				t.Fatal(err)
			}
			raw[tc.at] ^= 73
			encoded, err := sav.EncodeDocumentData(changed)
			if err != nil {
				t.Fatal(err)
			}
			cold := nativeSubjectCold(t, f, encoded)
			if cold.live.world.Hash() == before {
				t.Fatal("actual ordinary observation omitted from current hashed state")
			}
		})
	}
}
