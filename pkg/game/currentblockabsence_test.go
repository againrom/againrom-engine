package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentBlockAbsenceKeepsOrdinaryEditsAcrossTwoCycles(t *testing.T) {
	for _, field := range []string{"Dyn", "Static"} {
		t.Run(field, func(t *testing.T) {
			f := currentRetainedRuntimeFixture(t, []uint32{0xabcdef77})
			doc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || len(a.AbsentBlocks) != len(doc.World.Blocks) || len(a.AbsentBlocks) == 0 {
				t.Fatal("fixture does not cover absent native blocks", err)
			}
			policy, _, _ := sav.NativeActions(doc.State)
			if field == "Dyn" {
				doc.World.Blocks[0].Dyn ^= 1
			} else {
				doc.World.Blocks[0].Static ^= 1
			}
			want := doc.World.Blocks[0]
			afterPolicy, _, _ := sav.NativeActions(doc.State)
			if !bytes.Equal(policy, afterPolicy) {
				t.Fatal("ordinary block edit changed native policy")
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			var previous *sim.World
			for cycle := 0; cycle < 2; cycle++ {
				f = openCurrentRetainedRuntime(t, currentRetainedRuntimeFront(t), raw)
				_, _, blocks, present := f.live.world.SavedActorMotions()
				if !present || len(blocks) != 1 || blocks[0] != (sim.SavedActorBlock{Cell: want.Cell, Dyn: want.Dyn, Static: want.Static}) {
					t.Fatal("ordinary block edit was lost or adopted another absent row", cycle, blocks)
				}
				if previous != nil && previous.Hash() != f.live.world.Hash() {
					t.Fatal("second cycle changed exact edited World")
				}
				previous = f.live.world
				raw = currentRuntimeSave(t, f)
			}
		})
	}
}

func TestCurrentBlockAbsenceKeepsNativeCrossingAndSuccessors(t *testing.T) {
	f, _, _ := crossingOriginalDoor1115(t, false)
	f.live.tick()
	want := f.live.world
	for cycle := 0; cycle < 2; cycle++ {
		raw := currentRuntimeSave(t, f)
		cold := newGroupFront(t, -1)
		open, town, err := cold.RestoreOriginal(raw)
		if err == nil && !town {
			err = cold.App("exact crossing block presence").OpenMission(open)
		}
		if err != nil || town {
			t.Fatal("crossing LOAD", town, err)
		}
		if want.Hash() != cold.live.world.Hash() {
			before, _ := want.MarshalBinary()
			after, _ := cold.live.world.MarshalBinary()
			currentItemWorldDiagnostics(t, before, after)
			t.Fatalf("cycle%d crossing World changed: %x/%x", cycle, want.Hash(), cold.live.world.Hash())
		}
		_, _, blocks, _ := cold.live.world.SavedActorMotions()
		if len(blocks) != 4 {
			t.Fatal("terrain representation became native occupancy rows", len(blocks))
		}
		for tick := 0; tick < 2; tick++ {
			sim.Step(want, nil)
			sim.Step(cold.live.world, nil)
			if want.Hash() != cold.live.world.Hash() {
				t.Fatal("next crossing tick changed", cycle, tick)
			}
		}
		f = cold
	}
}

func TestCurrentBlockAbsenceMalformedPolicyIsAtomic(t *testing.T) {
	f := currentRetainedRuntimeFixture(t, []uint32{71})
	raw := currentRuntimeSave(t, f)
	for _, change := range []string{"duplicate", "empty anchor", "bad framing"} {
		t.Run(change, func(t *testing.T) {
			doc, _ := sav.DecodeDocumentData(raw)
			a, err := readCurrentActions(&doc)
			if err != nil || len(a.AbsentBlocks) == 0 {
				t.Fatal("fixture", err)
			}
			switch change {
			case "duplicate":
				a.AbsentBlocks = append(a.AbsentBlocks, a.AbsentBlocks[0])
			case "empty anchor":
				a.AbsentBlocks[0].Anchor = [16]byte{}
			}
			leaf, _ := json.Marshal(a)
			if change == "bad framing" {
				var data map[string]json.RawMessage
				if err := json.Unmarshal(leaf, &data); err != nil {
					t.Fatal(err)
				}
				data["AbsentBlocks"] = json.RawMessage(`"AQ=="`)
				leaf, _ = json.Marshal(data)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			live, hash := f.live, f.live.world.Hash()
			if _, _, err := f.RestoreOriginal(candidate); err == nil || f.live != live || f.live.world.Hash() != hash {
				t.Fatal("invalid block presence changed the current session", err)
			}
		})
	}
}

func TestCurrentBlockAbsenceFitsCombinedFullCellKeyspace(t *testing.T) {
	f := currentRetainedRuntimeFixture(t, []uint32{0xabcdef77})
	doc, err := sav.DecodeDocumentData(currentRuntimeSave(t, f))
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || len(a.Values) == 0 || len(a.Bindings) == 0 || a.Policy == nil {
		t.Fatal("fixture lacks populated current action sections", err)
	}
	rows := make(currentCellAbsence, 65536)
	blocks := make(currentBlockAbsence, 65536)
	seen := make(map[[16]byte]bool, 65536)
	for i := range rows {
		rows[i] = currentAbsentCellCarrier{Cell: uint16(i), Anchor: currentCellCarrierAnchor(sav.DocumentCellData{Cell: uint16(i), Residue32: uint16(i)}, false)}
		anchor := currentBlockCarrierAnchor(sav.BlockRecord{Dyn: uint8(i), Static: uint8(i >> 8)})
		if seen[anchor] || anchor == ([16]byte{}) {
			t.Fatal("distinct ordinary block bytes share an absence anchor", i)
		}
		seen[anchor] = true
		blocks[i] = currentAbsentBlockCarrier{Cell: uint16(i), Anchor: anchor}
	}
	a.AbsentCellTails, a.AbsentCellRecords, a.AbsentBlocks = rows, rows, blocks
	raw, err := json.Marshal(a)
	if err != nil || len(raw) > sav.MaxNativeActions {
		t.Fatal("combined actual current action leaf exceeds its framing", len(raw), sav.MaxNativeActions, err)
	}
	t.Logf("combined current action leaf=%d limit=%d", len(raw), sav.MaxNativeActions)
	if err := sav.SetNativeActions(&doc.State, raw); err != nil {
		t.Fatal(err)
	}
	back, err := readCurrentActions(&doc)
	if err != nil || !reflect.DeepEqual(back.AbsentCellTails, rows) || !reflect.DeepEqual(back.AbsentCellRecords, rows) || !reflect.DeepEqual(back.AbsentBlocks, blocks) {
		t.Fatal("combined full-keyspace policy changed anchors", err)
	}
}
