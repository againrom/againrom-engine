package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestLegacyNativeScalarsCurrentStateThroughSAVE(t *testing.T) {
	f, doc, object := legacyRawFixture(t)
	r := &doc.Objects[object-1]
	for _, value := range []sav.DocumentValueData{{Name: "T18", Value: 0x4321}, {Name: "U8E", Value: 13}, {Name: "UA0", Value: 11}, {Name: "U130", Value: 0xf1234567}, {Name: "U138", Value: 0x87654321}} {
		if err := savedActorSetValue(r, value.Name, value.Value); err != nil {
			t.Fatal(err)
		}
	}
	position, err := savedActorRaw(r, "Block12", 12)
	if err != nil {
		t.Fatal(err)
	}
	position[6], position[7] = 0x52, 0x73
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := nativeSubjectCold(t, f, raw)
	e, ok := cold.live.world.Entity(0)
	if !ok || !e.NativeBasis.ScalarsPresent || e.NativeBasis.ScalarKnown != 1<<sim.ScalarCount-1 || !e.NativeBasis.BlockPresent || e.NativeBasis.BlockKnown != 0x3ff {
		t.Fatal("legacy ordinary scalars/tail unavailable in current World", e.NativeBasis)
	}
	if e.NativeBasis.AttackKnown != 0xffffff || e.NativeBasis.DefenceKnown != 0x3fffff {
		t.Fatal("legacy raw combat bytes were not fully observed", e.NativeBasis)
	}
	for _, block := range []struct {
		name string
		got  []byte
	}{{"UA6", e.NativeBasis.Attack[:]}, {"UBE", e.NativeBasis.Defence[:]}} {
		want, err := savedActorRaw(r, block.name, len(block.got))
		if err != nil {
			t.Fatal(err)
		}
		for n, value := range want {
			if block.got[n] != value {
				t.Fatalf("legacy ordinary %s byte %d: got %#x want %#x", block.name, n, block.got[n], value)
			}
		}
	}
	for slot, name := range [sim.ScalarCount]string{"T0C", "T08", "T18", "T1C", "Reference", "U4B", "U4C", "U6C", "U8E", "U60", "U61", "UA0", "UA4", "U130", "U136", "U138", "U148", "U144", "U50", "U54", "U58"} {
		var want uint32
		if slot >= sim.ScalarU50 {
			p, err := savedActorRaw(r, name, 4)
			if err != nil {
				t.Fatal(err)
			}
			want = binary.LittleEndian.Uint32(p)
		} else {
			var err error
			want, err = savedStructureValue(r, name)
			if err != nil {
				t.Fatal(err)
			}
			if slot == sim.ScalarT08High {
				want >>= 16
			}
		}
		if e.NativeBasis.Scalars[slot] != want {
			t.Fatalf("legacy ordinary %s: got %#x want %#x", name, e.NativeBasis.Scalars[slot], want)
		}
	}
	for n, want := range position[2:] {
		if e.NativeBasis.Block[n] != want {
			t.Fatalf("legacy ordinary Block byte %d: got %#x want %#x", n+2, e.NativeBasis.Block[n], want)
		}
	}
	if e.ActorLoad.Source != (sim.SourceActor{}) || e.SourceBinding != (sim.SourceBinding{}) || e.ActorLoad.Present || e.NativeClass.Present {
		t.Fatal("legacy observation promoted source/load/class admission")
	}
	for _, slot := range []int{sim.ScalarT18, sim.ScalarU8E, sim.ScalarUA0, sim.ScalarU130, sim.ScalarU138} {
		want := map[int]uint32{sim.ScalarT18: 0x4321, sim.ScalarU8E: 13, sim.ScalarUA0: 11, sim.ScalarU130: 0xf1234567, sim.ScalarU138: 0x87654321}[slot]
		if e.NativeBasis.Scalars[slot] != want {
			t.Fatalf("ordinary scalar %d: got %#x want %#x", slot, e.NativeBasis.Scalars[slot], want)
		}
	}
	basis := e.NativeBasis
	basis.Scalars[sim.ScalarT18]++
	basis.Scalars[sim.ScalarU8E] = 29
	basis.Scalars[sim.ScalarUA0], basis.Scalars[sim.ScalarU130], basis.Scalars[sim.ScalarU138] = 31, 0xabcde123, 0xdeadbeef
	basis.Block[4], basis.Block[5] = 0x93, 0xb4
	basis.Attack[22], basis.Defence[4], basis.Defence[16] = 0xd5, 0xe6, 0x57
	if err := cold.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 0, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	written, saved, _ := saveCurrentEffect(t, cold)
	r = &saved.Objects[object-1]
	for _, value := range []sav.DocumentValueData{{Name: "T18", Value: 0x4322}, {Name: "U8E", Value: 29}, {Name: "UA0", Value: 31}, {Name: "U130", Value: 0xabcde123}, {Name: "U138", Value: 0xdeadbeef}} {
		got, err := savedStructureValue(r, value.Name)
		if err != nil || got != value.Value {
			t.Fatalf("changed current %s did not reach ordinary SAVE: got %#x want %#x (%v)", value.Name, got, value.Value, err)
		}
	}
	position, err = savedActorRaw(r, "Block12", 12)
	if err != nil || position[6] != 0x93 || position[7] != 0xb4 {
		t.Fatal("changed current Block tail did not reach ordinary SAVE", position, err)
	}
	again := nativeSubjectCold(t, f, written)
	got, _ := again.live.world.Entity(0)
	for _, slot := range []int{sim.ScalarT18, sim.ScalarU8E, sim.ScalarUA0, sim.ScalarU130, sim.ScalarU138} {
		if got.NativeBasis.Scalars[slot] != basis.Scalars[slot] {
			t.Fatal("cold LOAD lost current native scalar", slot, got.NativeBasis, basis)
		}
	}
	if got.NativeBasis.Block[4] != 0x93 || got.NativeBasis.Block[5] != 0xb4 || got.NativeBasis.Attack[22] != 0xd5 || got.NativeBasis.Defence[4] != 0xe6 || got.NativeBasis.Defence[16] != 0x57 {
		t.Fatal("cold LOAD lost current native raw values", got.NativeBasis)
	}
}

func TestNativeScalarSemanticFieldsWin(t *testing.T) {
	f, raw := nativeSubjectFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	var object uint16
	for _, binding := range a.Bindings {
		if !binding.Structure && binding.ID == 0 {
			object = binding.Object
		}
	}
	e, _ := f.live.world.Entity(0)
	e.NativeBasis.ScalarsPresent, e.NativeBasis.ScalarKnown = true, 1<<sim.ScalarU4C|1<<sim.ScalarU8E|1<<sim.ScalarUA0|1<<sim.ScalarUA4|1<<sim.ScalarU130|1<<sim.ScalarT1C
	e.NativeBasis.Scalars[sim.ScalarU4C], e.NativeBasis.Scalars[sim.ScalarU8E] = 255, 13
	e.NativeBasis.Scalars[sim.ScalarUA0], e.NativeBasis.Scalars[sim.ScalarUA4], e.NativeBasis.Scalars[sim.ScalarU130], e.NativeBasis.Scalars[sim.ScalarT1C] = 31, 0x237a, 0xfedcba98, 999
	e.OffMap, e.ScanRange, e.XPValue, e.HP = false, 7, 17, 43
	next, err := savedActorValueRecord(doc.Objects[object-1], e, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []sav.DocumentValueData{{Name: "U4C", Value: 247}, {Name: "U8E", Value: 13}, {Name: "UA0", Value: 31}, {Name: "UA4", Value: 0x077a}, {Name: "U130", Value: 0xfedcba98}, {Name: "T1C", Value: 17}, {Name: "Health", Value: 43}} {
		got, err := savedStructureValue(&next, value.Name)
		if err != nil || got != value.Value {
			t.Fatalf("current semantic/observed %s: got %#x want %#x (%v)", value.Name, got, value.Value, err)
		}
	}
}

func TestNativeScalarAvailabilityTransport(t *testing.T) {
	f, _ := nativeSubjectFixture(t)
	basis := sim.NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << sim.ScalarT18, BlockPresent: true, BlockKnown: 1 << 4}
	basis.Scalars[sim.ScalarT18], basis.Scalars[sim.ScalarU8E], basis.Block[4], basis.Block[5] = 0x6789, 0xabcdef01, 0x37, 0x59
	if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 0, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	raw, doc, _ := saveCurrentEffect(t, f)
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal(err)
	}
	a := new(currentActionData)
	if err := json.Unmarshal(leaf, a); err != nil {
		t.Fatal(err)
	}
	var metadata *sim.NativeActorBasis
	var object uint16
	for _, actor := range a.Actions.Actors {
		if actor.Entity == 0 && actor.Current != nil {
			metadata = actor.Current.NativeBasis
		}
	}
	for _, binding := range a.Bindings {
		if !binding.Structure && binding.ID == 0 {
			object = binding.Object
		}
	}
	if metadata == nil || metadata.Scalars[sim.ScalarT18] != 0 || metadata.Block[4] != 0 || metadata.Scalars[sim.ScalarU8E] != 0xabcdef01 || metadata.Block[5] != 0x59 {
		t.Fatal("native metadata duplicated known ordinary values or lost unknown residue", metadata)
	}
	cold := nativeSubjectCold(t, f, raw)
	e, _ := cold.live.world.Entity(0)
	if e.NativeBasis != basis {
		t.Fatal("native scalar mask/zero/unknown residue changed", e.NativeBasis, basis)
	}
	if err := savedActorSetValue(&doc.Objects[object-1], "T18", 0x7654); err != nil {
		t.Fatal(err)
	}
	p, err := savedActorRaw(&doc.Objects[object-1], "Block12", 12)
	if err != nil {
		t.Fatal(err)
	}
	p[6] = 0x73
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold = nativeSubjectCold(t, f, raw)
	e, _ = cold.live.world.Entity(0)
	basis.Scalars[sim.ScalarT18], basis.Block[4] = 0x7654, 0x73
	if e.NativeBasis != basis {
		t.Fatal("ordinary current edit failed to replace known metadata value", e.NativeBasis, basis)
	}
	for _, actor := range a.Actions.Actors {
		if actor.Entity == 0 && actor.Current != nil {
			actor.Current.NativeBasis.Scalars[sim.ScalarT18] = 1
		}
	}
	if err := matchCurrentNativeBasis(&doc, a); err == nil {
		t.Fatal("duplicated known scalar payload accepted")
	}
}

func TestNativeScalarSourceActorGuard(t *testing.T) {
	doc, binding, source := actorProjectionFixture(t, "Unit")
	e := source.Entities()[0]
	e.NativeBasis = sim.NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << sim.ScalarT18}
	e.NativeBasis.Scalars[sim.ScalarT18] = 0x1234
	if e.ActorLoad.Source.Class == 0 {
		t.Fatal("guard fixture lacks actual source admission")
	}
	if _, err := savedActorValueRecord(doc.Objects[binding.ObjectIndex-1], e, false); err == nil {
		t.Fatal("native scalar carrier accepted on source-backed actor")
	}
}

func TestNativeScalarHydratedAdapterRewriteKeepsWireAuthority(t *testing.T) {
	f, _ := nativeSubjectFixture(t)
	basis := sim.NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << sim.ScalarT18, BlockPresent: true, BlockKnown: 1 << 4}
	basis.Scalars[sim.ScalarT18], basis.Scalars[sim.ScalarU8E], basis.Block[4], basis.Block[5] = 0x6789, 0xabcdef01, 0x37, 0x59
	if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 0, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	_, doc, hydrated := saveCurrentEffect(t, f)
	var object uint16
	for _, binding := range hydrated.Bindings {
		if binding.ID == 0 && !binding.Structure {
			object = binding.Object
		}
	}
	if object == 0 {
		t.Fatal("adapter fixture lacks exact ordinary binding")
	}
	if err := savedActorSetValue(&doc.Objects[object-1], "T18", 0x7654); err != nil {
		t.Fatal(err)
	}
	p, err := savedActorRaw(&doc.Objects[object-1], "Block12", 12)
	if err != nil {
		t.Fatal(err)
	}
	p[6] = 0x73
	hydrated.NativeHistoryVersion = 1
	type transport currentActionData
	before, err := json.Marshal((*transport)(hydrated))
	if err != nil {
		t.Fatal("independent adapter ownership snapshot", err)
	}
	leaf, err := json.Marshal(hydrated)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal((*transport)(hydrated))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("adapter serialization mutated hydrated input")
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	again, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal("hydrated adapter rewrite duplicated ordinary values", err)
	}
	basis.Scalars[sim.ScalarT18], basis.Block[4] = 0x7654, 0x73
	for _, actor := range again.Actions.Actors {
		if actor.Entity == 0 && actor.Current != nil && actor.Current.NativeBasis != nil && *actor.Current.NativeBasis != basis {
			t.Fatal("adapter rewrite ignored current ordinary authority", actor.Current.NativeBasis, basis)
		}
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := nativeSubjectCold(t, f, raw)
	e, _ := cold.live.world.Entity(0)
	if e.NativeBasis != basis {
		t.Fatal("adapter rewrite cold LOAD lost scalar availability", e.NativeBasis, basis)
	}
}

func TestNativeScalarMarshalBoundAndUnbound(t *testing.T) {
	basis := sim.NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1<<sim.ScalarCount - 1, BlockPresent: true, BlockKnown: 0x3ff}
	for slot := range basis.Scalars {
		basis.Scalars[slot] = uint32(slot + 1)
	}
	for n := range basis.Block {
		basis.Block[n] = byte(n + 31)
	}
	a := currentActionData{Version: 1, NativeHistoryVersion: 1,
		Actions:            sim.ActionContinuations{Actors: []sim.ActorContinuation{{Entity: 7, Current: &sim.ActorCurrentContinuation{NativeBasis: &basis}}}},
		Held:               []sim.ActorContinuation{{Entity: 8, Current: &sim.ActorCurrentContinuation{NativeBasis: &basis}}},
		RemovedNativeBases: []sim.NativeActorBasisRecord{{ID: 9, Basis: basis}},
		NativeBasisWires:   []currentNativeBasisWire{{Entity: 7}, {Entity: 8, Unbound: true}, {Entity: 9, Unbound: true}},
	}
	type transport currentActionData
	before, err := json.Marshal(transport(a))
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(transport(a))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("bound/unbound serialization mutated input")
	}
	var decoded transport
	if err := json.Unmarshal(leaf, &decoded); err != nil {
		t.Fatal(err)
	}
	bound := *decoded.Actions.Actors[0].Current.NativeBasis
	if bound.Scalars != ([sim.ScalarCount]uint32{}) || bound.Block != ([10]byte{}) || bound.ScalarKnown != basis.ScalarKnown || bound.BlockKnown != basis.BlockKnown || !bound.ScalarsPresent || !bound.BlockPresent {
		t.Fatal("bound known values duplicated or availability changed", bound)
	}
	if *decoded.Held[0].Current.NativeBasis != basis || decoded.RemovedNativeBases[0].Basis != basis {
		t.Fatal("unbound continuation lost its only scalar/raw values")
	}
	bad := basis
	bad.ScalarKnown |= 1 << sim.ScalarCount
	a.Actions.Actors[0].Current.NativeBasis = &bad
	if _, err := json.Marshal(a); err == nil {
		t.Fatal("malformed known scalar mask accepted by transport")
	}
}

func TestNativeScalarRawDwordsUseLittleEndian(t *testing.T) {
	f, raw := nativeSubjectFixture(t)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	var object uint16
	for _, binding := range a.Bindings {
		if !binding.Structure && binding.ID == 0 {
			object = binding.Object
		}
	}
	e, _ := f.live.world.Entity(0)
	e.NativeBasis.ScalarsPresent = true
	for _, slot := range []int{sim.ScalarU50, sim.ScalarU54, sim.ScalarU58} {
		e.NativeBasis.ScalarKnown |= 1 << slot
		e.NativeBasis.Scalars[slot] = 0x89abcdef + uint32(slot)
	}
	next, err := savedActorValueRecord(doc.Objects[object-1], e, false)
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"U50", "U54", "U58"} {
		p, err := savedActorRaw(&next, name, 4)
		if err != nil || binary.LittleEndian.Uint32(p) != e.NativeBasis.Scalars[sim.ScalarU50+i] {
			t.Fatal("native raw dword not projected", name, p, err)
		}
	}
}

func TestNativeScalarClockAuthority(t *testing.T) {
	f, _ := nativeSubjectFixture(t)
	actor, _ := f.live.world.Entity(0)
	for _, tc := range []struct {
		name     string
		clock    sim.ActionClock
		observed bool
		want     uint32
	}{
		{"absentclock", sim.ActionClock{}, true, 0xdeadbeef},
		{"knownzero", sim.ActionClock{Known: true}, true, 0},
		{"knownwrap", sim.ActionClock{Known: true, End: 0xfffffff0}, true, 0xfffffff0},
		{"absentobservation", sim.ActionClock{}, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := actor
			e.ActionClock = tc.clock
			e.NativeBasis = sim.NativeActorBasis{}
			if tc.observed {
				e.NativeBasis.ScalarsPresent, e.NativeBasis.ScalarKnown = true, 1<<sim.ScalarU138
				e.NativeBasis.Scalars[sim.ScalarU138] = 0xdeadbeef
			}
			world, err := sim.NewWorld(123, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil, []sim.Entity{e})
			if err != nil {
				t.Fatal(err)
			}
			state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{{Class: "Unit", Values: []sav.DocumentValueData{{Name: "U138", Value: 999}}}}}, Actors: []SnapshotSAVActor{{EntityID: e.ID, ObjectIndex: 1}}}
			if err := projectActionClocks(state, world); err != nil {
				t.Fatal(err)
			}
			got, err := savedStructureValue(&state.Document.Objects[0], "U138")
			if err != nil || got != tc.want {
				t.Fatalf("clock/observation precedence: got %#x want %#x (%v)", got, tc.want, err)
			}
		})
	}
}
