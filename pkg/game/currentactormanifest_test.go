package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentManifestFront(t *testing.T, humanoid bool) *FrontEnd {
	t.Helper()
	body, _ := actorRegistryBody1111()
	if humanoid {
		actor := &poolFixtureActor{mapID: 91, cell: 0x100f, hp: 7, maxHP: 31, class: "Humanoid", name: "Exact Humanoid"}
		body = poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{actor}}}}, nil)
		body[actor.off+16] = 254
		binary.LittleEndian.PutUint16(body[actor.off+17:], 3)
	}
	f := actorRegistryFront1111(t)
	open, town, err := f.RestoreOriginal(savedContainer(body))
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := f.App("current actor manifest").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCurrentActorManifestKeepsOrdinaryNamesAndNativeOrder(t *testing.T) {
	for _, humanoid := range []bool{false, true} {
		for _, edit := range []bool{false, true} {
			t.Run(fmt.Sprintf("Humanoid=%t/ordinaryEdit=%t", humanoid, edit), func(t *testing.T) {
				f := currentManifestFront(t, humanoid)
				s, _, err := f.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				want := cloneActorManifest(s.ActorManifest)
				slices.Reverse(want.Actors)
				for i := range want.Actors {
					want.Actors[i].Name = fmt.Sprintf("current %d ", i) + string([]byte{0xff, 0xc0})
				}
				s.ActorManifest = cloneActorManifest(want)
				hash := f.live.world.Hash()
				raw, err := f.ExportCurrentSave(s, "ordinary actor names")
				if err != nil {
					t.Fatal(err)
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				a, err := readCurrentActions(&doc)
				if err != nil || a.Manifest == nil || !a.Manifest.Present || len(a.Manifest.Actors) != len(want.Actors) {
					t.Fatal("missing current manifest policy", err)
				}
				private, _ := json.Marshal(a.Manifest)
				if bytes.Contains(private, []byte("Name")) {
					t.Fatal("ordinary actor name acquired a private copy")
				}
				objects := map[sim.EntityID]uint16{}
				for _, b := range a.Bindings {
					if !b.Structure && !b.Missing {
						objects[b.ID] = b.Object
					}
				}
				leaf, _, err := sav.NativeActions(doc.State)
				if err != nil {
					t.Fatal(err)
				}
				for i := range want.Actors {
					row := &want.Actors[i]
					record := &doc.Objects[objects[row.ID]-1]
					for _, text := range record.Texts {
						if text.Name == "Name" && text.Value != row.Name {
							t.Fatal("SAVE retained an old name", record.Class, text.Value, row.Name)
						}
					}
					if edit {
						row.Name = fmt.Sprintf("ordinary %d ", i) + string([]byte{0xfe, 0xcf})
						mustSetText(record, "Name", row.Name)
					}
				}
				afterLeaf, _, err := sav.NativeActions(doc.State)
				if err != nil || !bytes.Equal(leaf, afterLeaf) {
					t.Fatal("ordinary name edit changed continuation policy", err)
				}
				raw, err = sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
				for cycle := 0; cycle < 2; cycle++ {
					cold := actorRegistryFront1111(t)
					open, town, err := cold.RestoreOriginal(raw)
					if err != nil || town {
						t.Fatal("cold manifest", cycle, town, err)
					}
					if err := cold.App("restored current manifest").OpenMission(open); err != nil {
						t.Fatal(err)
					}
					if cold.live.world.Hash() != hash || !reflect.DeepEqual(cold.live.mission.state.ActorManifest, want) {
						t.Fatalf("cycle%d World=%x/%x manifest=%+v want=%+v", cycle, cold.live.world.Hash(), hash, cold.live.mission.state.ActorManifest, want)
					}
					for _, row := range want.Actors {
						if name, ok := cold.live.actorNames[row.ID]; !ok || name != row.Name {
							t.Fatal("rendered actor name is not current ordinary data", row, name)
						}
					}
					raw = currentRuntimeSave(t, cold)
				}
			})
		}
	}
}

func TestCurrentActorManifestKeepsAbsenceAndEmptyPresence(t *testing.T) {
	for _, present := range []bool{false, true} {
		t.Run(fmt.Sprint(present), func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91, 92)
			if err := f.App("native manifest mode").OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			if present {
				f.live.mission.state.ActorManifest = &SnapshotActorManifest{Version: actorManifestVersion}
			}
			want := cloneActorManifest(f.live.mission.state.ActorManifest)
			hash := f.live.world.Hash()
			for cycle := 0; cycle < 2; cycle++ {
				raw := currentRuntimeSave(t, f)
				cold := currentPoolFixtureFront(t, 91, 92)
				open, town, err := cold.RestoreOriginal(raw)
				if err != nil || town {
					t.Fatal(town, err)
				}
				if err := cold.App("restored native manifest mode").OpenMission(open); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(cold.live.mission.state.ActorManifest, want) || cold.live.world.Hash() != hash {
					t.Fatal("native manifest acquired constructor rows", cycle, cold.live.mission.state.ActorManifest, want)
				}
				f = cold
			}
		})
	}
}

func TestCurrentActorManifestMalformedPolicyIsAtomic(t *testing.T) {
	f := currentManifestFront(t, false)
	raw := currentRuntimeSave(t, f)
	before, _ := f.live.world.MarshalBinary()
	live, town, units := f.live, f.Town, f.Units
	manifest := cloneActorManifest(f.live.mission.state.ActorManifest)
	for _, change := range []string{"duplicate", "missing", "unknown", "absent with rows", "absent source", "absent mode"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "duplicate":
				a.Manifest.Actors = append(a.Manifest.Actors, a.Manifest.Actors[0])
			case "missing":
				a.Manifest.Actors = a.Manifest.Actors[1:]
			case "unknown":
				a.Manifest.Actors[0].Entity = 999
			case "absent with rows":
				a.Manifest.Present = false
			case "absent source":
				a.Manifest.Actors[0].Entity = 0 // native map actor, no source binding
			case "absent mode":
				a.Manifest.Present, a.Manifest.Actors = false, nil
			}
			leaf, _ := json.Marshal(a)
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			candidate, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = f.RestoreOriginal(candidate)
			after, marshalErr := f.live.world.MarshalBinary()
			if err == nil || marshalErr != nil || !bytes.Equal(before, after) || f.live != live || f.Town != town || f.Units != units || !reflect.DeepEqual(manifest, f.live.mission.state.ActorManifest) {
				t.Fatal("malformed manifest published candidate", err, marshalErr)
			}
		})
	}
}
