package game

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestNativeInitialModifierOrdinaryBytesAndNextRemoval(t *testing.T) {
	f := structureFront(t, true, true)
	defs := eqDefsTable(t)
	f.Table.Shapes, f.Table.Materials, f.Table.Weapons = defs.Shapes, defs.Materials, defs.Weapons
	u := &f.Table.Units.(dbCollection)[1]
	u.params[14], u.strings = 17, []string{"Bow"}
	if err := f.App("native initial Modifier").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	e, ok := f.live.world.Entity(0)
	want := [64]byte{18: 17, 37: 6, 38: 6, 39: 1}
	if !ok || e.ActorLoad.Source.Class != 0 || !e.NativeBasis.ModifierPresent || e.NativeBasis.ModifierKnown != ^uint64(0) || e.NativeBasis.Modifier != want {
		t.Fatal("actual resolved Unit lacks independent ordered constructor history", e.NativeBasis)
	}
	raw, doc, actions := saveCurrentEffect(t, f)
	var binding SnapshotSAVActor
	for _, row := range actions.Bindings {
		if row.ID == e.ID && !row.Structure && !row.Missing {
			binding = SnapshotSAVActor{EntityID: row.ID, ObjectIndex: row.Object}
		}
	}
	if binding.ObjectIndex == 0 {
		t.Fatal("actual Unit has no ordinary root")
	}
	block, err := savedActorRaw(&doc.Objects[binding.ObjectIndex-1], "UD4", 64)
	if err != nil || !bytes.Equal(block, want[:]) {
		t.Fatal("ordinary UD4 differs from literal constructor events", block, err)
	}
	for _, control := range []struct {
		name    string
		present bool
		known   uint64
	}{
		{"known zeros", true, ^uint64(0)},
		{"one unknown byte", true, ^uint64(0) &^ (uint64(1) << 40)},
		{"present unknown", true, 0},
		{"component absent", false, 0},
	} {
		t.Run(control.name, func(t *testing.T) {
			changed, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			old, _ := savedActorRaw(&changed.Objects[binding.ObjectIndex-1], "UD4", 64)
			for i := range old {
				old[i] = 0xcc
			}
			basis := e.NativeBasis
			basis.ModifierPresent, basis.ModifierKnown = control.present, control.known
			if control.present {
				basis.Modifier[40] = 0xa5
			} else {
				basis.Modifier = [64]byte{}
			}
			if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: e.ID, Basis: basis}}); err != nil {
				t.Fatal(err)
			}
			if err := projectSavedActorValues(&changed, []SnapshotSAVActor{binding}, f.live.world); err != nil {
				t.Fatal(err)
			}
			got, _ := savedActorRaw(&changed.Objects[binding.ObjectIndex-1], "UD4", 64)
			for i, value := range got {
				expected := byte(0xcc)
				if control.known&(uint64(1)<<i) != 0 {
					expected = basis.Modifier[i]
				}
				switch i {
				case 10, 11:
					expected = byte(uint16(e.HealthRegeneration) >> (8 * (i - 10)))
				case 14, 15:
					expected = byte(uint16(e.ManaRegeneration) >> (8 * (i - 14)))
				}
				if value != expected {
					t.Fatalf("UD4[%d]=%02x, want %02x", i, value, expected)
				}
			}
		})
	}
	for _, edited := range []bool{false, true} {
		t.Run(map[bool]string{false: "cold unchanged", true: "cold ordinary selector edit"}[edited], func(t *testing.T) {
			input := raw
			if edited {
				changed, err := sav.CloneDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				b, _ := savedActorRaw(&changed.Objects[binding.ObjectIndex-1], "UD4", 64)
				b[39] = 5
				input, err = sav.EncodeDocumentData(changed)
				if err != nil {
					t.Fatal(err)
				}
			}
			cold := coldNativeConstructor(t, f, input)
			got, ok := cold.live.world.Entity(e.ID)
			expected := e.NativeBasis
			if edited {
				expected.Modifier[39] = 5
			}
			if !ok || got.ActorLoad.Source.Class != 0 || got.NativeBasis != expected {
				t.Fatal("ordinary LOAD lost constructor masks or current edit", got.NativeBasis, expected)
			}
			sim.Step(cold.live.world, []sim.Command{sim.Unequip(e.ID, 1)})
			for range 3 {
				cold.live.tick()
			}
			got, _ = cold.live.world.Entity(e.ID)
			if got.NativeBasis.Modifier != ([64]byte{18: 9}) || got.NativeBasis.ModifierKnown != ^uint64(0) {
				t.Fatal("next actual removal replayed constructor or lost local subtraction/clear", got.NativeBasis)
			}
			nextRaw, next, nextActions := saveCurrentEffect(t, cold)
			r := castOrderRecord(t, next, nextActions, e.ID)
			b, err := savedActorRaw(&r, "UD4", 64)
			if err != nil || !bytes.Equal(b, got.NativeBasis.Modifier[:]) {
				t.Fatal("next SAVE lost post-removal history", b, err)
			}
			last := coldNativeConstructor(t, cold, nextRaw)
			lastEntity, _ := last.live.world.Entity(e.ID)
			if lastEntity.NativeBasis != got.NativeBasis {
				t.Fatal("second cold LOAD lost post-removal history")
			}
		})
	}
}

func coldNativeConstructor(t *testing.T, source *FrontEnd, raw []byte) *FrontEnd {
	t.Helper()
	f := structureFront(t, true, true)
	f.Table = source.Table
	open, town, err := f.RestoreOriginal(raw)
	if err == nil && !town {
		err = f.App("native constructor cold LOAD").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("native constructor cold LOAD", err)
	}
	return f
}
