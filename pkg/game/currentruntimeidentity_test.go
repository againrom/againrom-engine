package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentRuntimeFront(t *testing.T, runtime [3]uint32) *FrontEnd {
	t.Helper()
	body, actors := aliasBody1111(true)
	for i, actor := range actors {
		binary.LittleEndian.PutUint32(body[actor.off+12:], runtime[i])
	}
	f := aliasFront1111(t)
	open, town, err := f.RestoreOriginal(savedContainer(body))
	if err != nil || town {
		t.Fatal(town, err)
	}
	app := f.App("current runtime identities")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	view := f.live.view.SaveApplication()
	view.Selection = []uint32{1}
	if err := f.live.view.RestoreSaveApplication(view); err != nil {
		t.Fatal("fixture selection", err)
	}
	return f
}

func currentRuntimeSave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "current runtime identity")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCurrentRuntimeCoordinatesKeepTwoCyclesAndOrdinaryEdits(t *testing.T) {
	for _, test := range []struct {
		name                       string
		ids                        [3]uint32
		runtimeEdit, selectionEdit bool
	}{
		{"unique", [3]uint32{71, 72, 73}, false, false},
		{"duplicate", [3]uint32{77, 77, 77}, false, false},
		{"wide duplicate", [3]uint32{0xabcdef77, 0xabcdef77, 0xfedc0011}, false, false},
		{"ordinary runtime", [3]uint32{77, 77, 77}, true, false},
		{"ordinary selection", [3]uint32{77, 77, 77}, false, true},
		{"wide ordinary runtime and selection", [3]uint32{0xabcdef77, 0xabcdef77, 0xfedc0011}, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := currentRuntimeFront(t, test.ids)
			wantIDs := test.ids
			if test.runtimeEdit {
				wantIDs[1] = 60123 // actor1 is the named hero, the second original object
			}
			want := currentRuntimeFront(t, wantIDs).live.world
			raw := currentRuntimeSave(t, f)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			actions, err := readCurrentActions(&doc)
			if err != nil || actions == nil {
				t.Fatal(err)
			}
			objects := map[sim.EntityID]uint16{}
			seen := map[uint32]bool{}
			for _, binding := range actions.Bindings {
				if binding.Structure || binding.Missing {
					continue
				}
				objects[binding.ID] = binding.Object
				wire, err := savedStructureValue(&doc.Objects[binding.Object-1], "RuntimeID")
				if err != nil || wire == 0 || wire > 65535 || seen[wire] {
					t.Fatal("ordinary runtime namespace is not unique and bounded", wire, err)
				}
				seen[wire] = true
				if test.name == "unique" && actions.Values[binding.ID].RuntimeID != nil {
					t.Fatal("unchanged coordinate acquired redundant policy")
				}
			}
			policy, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			if test.runtimeEdit {
				savedObjectSetValue(&doc.Objects[objects[1]-1], "RuntimeID", wantIDs[1])
			}
			selection := sim.EntityID(1)
			if test.selectionEdit {
				selection = 3
			}
			wire, err := savedStructureValue(&doc.Objects[objects[selection]-1], "RuntimeID")
			if err != nil {
				t.Fatal(err)
			}
			for i := range doc.State.ValueRecords {
				if leaf := &doc.State.ValueRecords[i]; leaf.Path == "/Objects/Selection" {
					leaf.Value.Bytes = binary.LittleEndian.AppendUint32(nil, wire)
				}
			}
			afterPolicy, _, err := sav.NativeActions(doc.State)
			if err != nil || !bytes.Equal(policy, afterPolicy) {
				t.Fatal("ordinary edit changed native policy", err)
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				cold := aliasFront1111(t)
				open, town, err := cold.RestoreOriginal(raw)
				if err != nil || town {
					t.Fatal("cold LOAD", cycle, town, err)
				}
				app := cold.App("cold runtime identity")
				if err := app.OpenMission(open); err != nil {
					t.Fatal(err)
				}
				if got := app.HeadlessSelection(); !reflect.DeepEqual(got, []uint32{uint32(selection)}) || cold.live.world.Hash() != want.Hash() {
					t.Fatalf("cycle%d selection=%v World=%x/%x", cycle, got, cold.live.world.Hash(), want.Hash())
				}
				for tick := 0; tick < 3; tick++ {
					sim.Step(want, nil)
					sim.Step(cold.live.world, nil)
					if cold.live.world.Hash() != want.Hash() {
						t.Fatal("next native tick changed", cycle, tick)
					}
				}
				raw = currentRuntimeSave(t, cold)
			}
		})
	}
}

func TestCurrentRuntimeCoordinateMalformedPolicyIsAtomic(t *testing.T) {
	f := currentRuntimeFront(t, [3]uint32{77, 77, 77})
	raw := currentRuntimeSave(t, f)
	before, _ := f.live.world.MarshalBinary()
	live, town, units := f.live, f.Town, f.Units
	selection := f.live.view.SaveApplication().Selection
	for _, change := range []string{"redundant", "wide anchor", "absent source", "unbound value", "duplicate binding", "unknown nested field"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a.Values[1].RuntimeID == nil {
				t.Fatal("fixture lacks remapped actor", err)
			}
			v := a.Values[1]
			switch change {
			case "redundant":
				v.RuntimeID.Value = v.RuntimeID.Wire
			case "wide anchor":
				v.RuntimeID.Wire = 65536
			case "absent source":
				v.SourceBound = false
			case "unbound value":
				a.Values[999] = v
			case "duplicate binding":
				a.Bindings = append(a.Bindings, a.Bindings[0])
			}
			a.Values[1] = v
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
			if err == nil || marshalErr != nil || !bytes.Equal(before, after) || f.live != live || f.Town != town || f.Units != units || !reflect.DeepEqual(selection, f.live.view.SaveApplication().Selection) {
				t.Fatal("malformed runtime coordinate published candidate", fmt.Sprint(err), marshalErr)
			}
		})
	}
}
