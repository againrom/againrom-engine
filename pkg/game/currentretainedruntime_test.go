package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentRetainedRuntimeFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	m := poolFixtureMap(91, 92)
	for off := 20; off+20 <= len(m); {
		size := int(binary.LittleEndian.Uint32(m[off+8:]))
		if binary.LittleEndian.Uint32(m[off+12:]) == 6 {
			for i := 0; i < 2; i++ {
				binary.LittleEndian.PutUint32(m[off+20+i*70+0x14:], 1)
			}
			break
		}
		off += 20 + size
	}
	f.Archives.Containers = poolFixtureFrontMap(t, m).Archives.Containers
	return f
}

func currentRetainedRuntimeFixture(t *testing.T, runtimes []uint32, livingOverride ...*poolFixtureActor) *FrontEnd {
	t.Helper()
	dead := make([]*poolFixtureActor, len(runtimes))
	for i := range runtimes {
		dead[i] = &poolFixtureActor{cell: uint16(0x0807 + i), hp: uint16(65536 - 14), maxHP: 30, stage: 2, human: true, runtime: &runtimes[i]}
	}
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 30, name: "Living binding"}
	if len(livingOverride) != 0 {
		living = livingOverride[0]
	}
	f := currentRetainedRuntimeFront(t)
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}}, dead))
	raw = completeDocumentTail1115(t, f, raw)
	return openCurrentRetainedRuntime(t, f, raw)
}

func openCurrentRetainedRuntime(t *testing.T, f *FrontEnd, raw []byte) *FrontEnd {
	t.Helper()
	open, town, err := f.RestoreOriginal(raw)
	if err == nil && !town {
		err = f.App("retained actor runtime").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("retained body LOAD", town, err)
	}
	return f
}

func TestCurrentRetainedRuntimeKeepsTwoCyclesAndNextDeathTick(t *testing.T) {
	for _, test := range []struct {
		name string
		ids  []uint32
		edit bool
	}{
		{"unique", []uint32{71}, false},
		{"wide", []uint32{0xabcdef77}, false},
		{"duplicate", []uint32{77, 77}, false},
		{"wide duplicate", []uint32{0xabcdef77, 0xabcdef77}, false},
		{"ordinary edit", []uint32{0xabcdef77, 0xabcdef77}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := currentRetainedRuntimeFixture(t, test.ids)
			wantIDs := append([]uint32(nil), test.ids...)
			if test.edit {
				wantIDs[0] = 60123
			}
			want := currentRetainedRuntimeFixture(t, wantIDs).live.world
			raw := currentRuntimeSave(t, f)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || len(doc.DeadActors) != len(test.ids) {
				t.Fatal("retained population", len(doc.DeadActors), err)
			}
			for _, dead := range f.live.world.OriginalDeadActors() {
				v, present := a.Values[dead.ID]
				if present {
					coordinate := v.RuntimeID
					v.RuntimeID = nil
					if coordinate == nil || !reflect.DeepEqual(v, sim.ActorValues{}) {
						t.Fatal("held row carries values beyond its runtime coordinate", dead.ID, v)
					}
				}
				for _, actor := range a.Actions.Actors {
					if actor.Entity == dead.ID {
						t.Fatal("held runtime coordinate fabricated a live action actor")
					}
				}
			}
			policy, _, _ := sav.NativeActions(doc.State)
			if test.edit {
				savedObjectSetValue(&doc.Objects[doc.DeadActors[0]-1], "RuntimeID", wantIDs[0])
			}
			afterPolicy, _, _ := sav.NativeActions(doc.State)
			if !bytes.Equal(policy, afterPolicy) {
				t.Fatal("ordinary edit changed private continuation")
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
				if cold.live.world.Hash() != want.Hash() {
					t.Fatalf("cycle%d exact World changed: %x -> %x", cycle, want.Hash(), cold.live.world.Hash())
				}
				if !reflect.DeepEqual(cold.live.world.OriginalDeadActors(), want.OriginalDeadActors()) || len(cold.live.world.Entities()) != len(want.Entities()) {
					t.Fatal("retained identity or current population changed")
				}
				var wantNext, gotNext sim.World
				wantBytes, _ := want.MarshalBinary()
				gotBytes, _ := cold.live.world.MarshalBinary()
				if err := wantNext.UnmarshalBinary(wantBytes); err != nil {
					t.Fatal(err)
				}
				if err := gotNext.UnmarshalBinary(gotBytes); err != nil {
					t.Fatal(err)
				}
				before := wantNext.OriginalDeadActors()[0].Current.HP
				for tick := 0; tick < 33; tick++ {
					sim.Step(&wantNext, nil)
					sim.Step(&gotNext, nil)
					if gotNext.Hash() != wantNext.Hash() {
						t.Fatalf("cycle%d next death tick%d changed exact World", cycle, tick)
					}
				}
				if wantNext.OriginalDeadActors()[0].Current.HP >= before {
					t.Fatal("fixture did not cross the next dead-manager tick")
				}
				raw = currentRuntimeSave(t, cold)
			}
		})
	}
}

func TestCurrentRetainedRuntimeMalformedPolicyIsAtomic(t *testing.T) {
	f := currentRetainedRuntimeFixture(t, []uint32{0xabcdef77})
	raw := currentRuntimeSave(t, f)
	id := f.live.world.OriginalDeadActors()[0].ID
	before, _ := f.live.world.MarshalBinary()
	live, town, units := f.live, f.Town, f.Units
	for _, change := range []string{"redundant", "wide anchor", "zero anchor", "zero value", "actor operands", "source actor", "unbound value", "duplicate binding", "unknown nested field"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a.Values[id].RuntimeID == nil {
				t.Fatal("fixture coordinate absent", err)
			}
			v := a.Values[id]
			switch change {
			case "redundant":
				v.RuntimeID.Value = v.RuntimeID.Wire
			case "wide anchor":
				v.RuntimeID.Wire = 65536
			case "zero anchor":
				v.RuntimeID.Wire = 0
			case "zero value":
				v.RuntimeID.Value = 0
			case "actor operands":
				v.DyingTime = 1
			case "source actor":
				v.SourceBound = true
			case "unbound value":
				a.Values[999] = v
			case "duplicate binding":
				a.Bindings = append(a.Bindings, a.Bindings[0])
			}
			a.Values[id] = v
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if change == "unknown nested field" {
				leaf = bytes.Replace(leaf, []byte(`"RuntimeID":{`), []byte(`"RuntimeID":{"Unexpected":1,`), 1)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = f.RestoreOriginal(candidate)
			after, marshalErr := f.live.world.MarshalBinary()
			if err == nil || marshalErr != nil || !bytes.Equal(before, after) || f.live != live || f.Town != town || f.Units != units {
				t.Fatal("malformed retained coordinate published a candidate", err, marshalErr)
			}
		})
	}
}
