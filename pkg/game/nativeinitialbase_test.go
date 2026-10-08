package game

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestNativeConstructorBaseWinsStaleDocumentOnlyAtKnownBytes(t *testing.T) {
	f := structureFront(t, true, true)
	f.Table.Units.(dbCollection)[1].params[14] = 17
	if err := f.App("native constructor Base").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	e, ok := f.live.world.Entity(0)
	if !ok || e.ActorLoad.Source.Class != 0 || !e.NativeBasis.BasePresent || e.NativeBasis.BaseKnown != 0x003fffff {
		t.Fatal("resolved native constructor lacks its independent Base prefix", e.NativeBasis)
	}
	_, doc, actions := saveCurrentEffect(t, f)
	var binding SnapshotSAVActor
	for _, row := range actions.Bindings {
		if row.ID == e.ID && !row.Structure && !row.Missing {
			binding = SnapshotSAVActor{EntityID: row.ID, ObjectIndex: row.Object}
		}
	}
	if binding.ObjectIndex == 0 {
		t.Fatal("actual constructor has no ordinary actor root")
	}
	for _, control := range []struct {
		name  string
		basis sim.NativeActorBasis
		known uint32
	}{
		{"constructor", e.NativeBasis, 0x003fffff},
		{"one mask bit absent", func() sim.NativeActorBasis { b := e.NativeBasis; b.BaseKnown &^= 1; return b }(), 0x003ffffe},
		{"present unknown", func() sim.NativeActorBasis { b := e.NativeBasis; b.BaseKnown = 0; return b }(), 0},
		{"component absent", func() sim.NativeActorBasis { b := e.NativeBasis; b.BasePresent = false; b.BaseKnown = 0; return b }(), 0},
	} {
		t.Run(control.name, func(t *testing.T) {
			changed, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			old, err := savedActorRaw(&changed.Objects[binding.ObjectIndex-1], "U114", 24)
			if err != nil {
				t.Fatal(err)
			}
			for i := range old {
				old[i] = byte(0xa0 + i)
			}
			basis := control.basis
			if basis.BasePresent {
				basis.Base[22], basis.Base[23] = 7, 9
			}
			if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: e.ID, Basis: basis}}); err != nil {
				t.Fatal(err)
			}
			if err := projectSavedActorValues(&changed, []SnapshotSAVActor{binding}, f.live.world); err != nil {
				t.Fatal(err)
			}
			got, _ := savedActorRaw(&changed.Objects[binding.ObjectIndex-1], "U114", 24)
			for i, value := range got {
				want := byte(0xa0 + i)
				if control.known&(1<<i) != 0 {
					want = 0
				}
				if value != want {
					t.Fatalf("U114[%d]=%02x, want %02x", i, value, want)
				}
			}
		})
	}
}

func TestNativeConstructorBaseSurvivesOrdinarySaveColdLoadAndAction(t *testing.T) {
	f := structureFront(t, true, true)
	f.Table.Units.(dbCollection)[1].params[14] = 17
	if err := f.App("native constructor Base").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	e, ok := f.live.world.Entity(0)
	if !ok || e.ActorLoad.Source.Class != 0 || e.Skill == ([6]int32{}) || e.NativeBasis.BaseKnown != 0x003fffff {
		t.Fatal("fixture lacks resolved native nonzero skills and independent Base", e)
	}
	raw, doc, actions := saveCurrentEffect(t, f)
	actor := uint16(0)
	for _, row := range actions.Bindings {
		if row.ID == e.ID && !row.Structure && !row.Missing {
			actor = row.Object
		}
	}
	if actor == 0 {
		t.Fatal("current constructor actor root missing")
	}
	base, err := savedActorRaw(&doc.Objects[actor-1], "U114", 24)
	if err != nil || !bytes.Equal(base[:22], make([]byte, 22)) {
		t.Fatal("ordinary U114 copied effective skills into constructor zeros", base, err)
	}
	for _, edited := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged", true: "ordinary known and unknown bytes"}[edited], func(t *testing.T) {
			input := raw
			if edited {
				changed, err := sav.CloneDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				block, _ := savedActorRaw(&changed.Objects[actor-1], "U114", 24)
				block[0], block[22], block[23] = 0x63, 0xa5, 0x5a
				input, err = sav.EncodeDocumentData(changed)
				if err != nil {
					t.Fatal(err)
				}
			}
			cold := coldCurrentBuildings(t, input)
			got, ok := cold.live.world.Entity(e.ID)
			want := e.NativeBasis
			if edited {
				want.Base[0] = 0x63
			}
			if !ok || got.ActorLoad.Source.Class != 0 || got.NativeBasis != want {
				t.Fatal("ordinary LOAD replaced constructor presence/mask/history", got.NativeBasis, want)
			}
			sim.Step(cold.live.world, []sim.Command{sim.MoveTo(e.ID, sim.CellPoint{X: 1, Y: 8})})
			for range 3 {
				cold.live.tick()
			}
			_, next, nextActions := saveCurrentEffect(t, cold)
			for _, row := range nextActions.Bindings {
				if row.ID != e.ID || row.Structure || row.Missing {
					continue
				}
				block, err := savedActorRaw(&next.Objects[row.Object-1], "U114", 24)
				if err != nil || !bytes.Equal(block[:22], want.Base[:22]) || edited && (block[22] != 0xa5 || block[23] != 0x5a) {
					t.Fatal("next ordinary SAVE lost known prefix or retained unknown tail", block, err)
				}
				return
			}
			t.Fatal("next ordinary SAVE lost the actual constructor root")
		})
	}
}
