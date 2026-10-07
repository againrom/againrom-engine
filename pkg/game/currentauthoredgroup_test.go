package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentAuthoredGroupBindsAbsentContainerWithoutGuessing(t *testing.T) {
	for _, change := range []string{"absent carrier", "native container", "missing native container", "ambiguous slot", "missing owner", "unknown owner", "missing native identity"} {
		t.Run(change, func(t *testing.T) {
			first, second := mustNewRecord("Player"), mustNewRecord("Player")
			mustSetValue(&first, "Slot", 1)
			mustSetValue(&first, "This", 100)
			mustSetValue(&second, "Slot", 2)
			mustSetValue(&second, "This", 200)
			state := &SnapshotSAVDocument{Document: &sav.DocumentData{Players: []uint16{1, 2}, Objects: []sav.DocumentRecordData{first, second}},
				GroupBindings: &SnapshotSAVGroupBindings{PlayersPresent: true, PlayersConstructed: true,
					Players: []SnapshotSAVGroupPlayerBinding{{ID: 7, ObjectIndex: 1}, {ID: 9, ObjectIndex: 2}}}}
			group := sim.SavedGroup{ID: 42, Authored: true, Owner: sim.SavedGroupReference{Class: 1, Owner: 2}}
			switch change {
			case "native container":
				state.GroupBindings.PlayersConstructed, group.ContainerID = false, 9
			case "missing native container":
				state.GroupBindings.PlayersConstructed = false
			case "ambiguous slot":
				mustSetValue(&state.Document.Objects[0], "Slot", 2)
			case "missing owner":
				group.Owner.Owner = 3
			case "unknown owner":
				group.Owner = sim.SavedGroupReference{}
			case "missing native identity":
				group.ContainerID = 99
			}
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			binding, err := bindAuthoredSavedGroup(state, group, map[uint32]uint16{7: 1, 9: 2})
			if change == "absent carrier" || change == "native container" {
				if err != nil || binding.ContainerID != group.ContainerID || binding.PlayerObject != 2 || binding.Owner != (SnapshotSAVGroupReferenceBinding{Key: 200, Class: 1, ObjectIndex: 2, Owner: 2}) {
					t.Fatal("current owner did not select its exact ordinary Player", binding, err)
				}
			} else if err == nil {
				t.Fatal("missing or ambiguous authority acquired a Player", binding)
			}
			after, err := json.Marshal(state)
			if err != nil || string(before) != string(after) {
				t.Fatal("binding mutated current document authority", err)
			}
		})
	}
}

func TestCurrentAuthoredGroupAbsentContainerKeepsTwoSAVCycles(t *testing.T) {
	for _, edit := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged", true: "ordinary AI"}[edit], func(t *testing.T) {
			f, snapshot, base := partialCurrentGraph(t)
			groups, _, _ := base.SavedGroups()
			group := groups[0]
			group.ContainerID, group.OwnerID = 0, 0
			world := func(g sim.SavedGroup) *sim.World {
				w, err := sim.NewWorld(11, base.Bounds(), sim.ModeCanonical, nil, base.Entities())
				if err != nil {
					t.Fatal(err)
				}
				if err := w.ImportSavedGroups([]sim.SavedGroup{g}, nil); err != nil {
					t.Fatal(err)
				}
				return w
			}
			initial := world(group)
			var err error
			snapshot.World, err = initial.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			snapshot.SavedDocument = nil
			raw, err := f.ExportCurrentSave(snapshot, "authored Group without native container")
			if err != nil {
				t.Fatal(err)
			}
			if edit {
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				actions, err := readCurrentActions(&doc)
				if err != nil || actions == nil {
					t.Fatal(err)
				}
				for _, binding := range actions.Groups {
					if binding.ID == group.ID && !binding.RootOnly {
						crossingRawField(t, &doc.Objects[binding.Object-1].Groups[binding.Inline], "G3C")[0x12] = 67
					}
				}
				group.AI[0x12] = 67
				raw, err = sav.EncodeDocumentData(doc)
				if err != nil {
					t.Fatal(err)
				}
			}
			want := world(group)
			for cycle := 0; cycle < 2; cycle++ {
				cold := cellStateFront(t)
				cold.Table, cold.Campaign = f.Table, f.Campaign
				open, town, err := cold.RestoreOriginal(raw)
				if err != nil || town {
					t.Fatal(town, err)
				}
				if err := cold.App("current authored Group").OpenMission(open); err != nil {
					t.Fatal(err)
				}
				got, _, present := cold.live.world.SavedGroups()
				_, players := cold.live.world.SavedGroupPlayers()
				_, formations := cold.live.world.SavedPlayerFormations()
				if !present || players || formations || !reflect.DeepEqual(got, []sim.SavedGroup{group}) || cold.live.world.Hash() != want.Hash() {
					t.Fatalf("cycle %d changed current Group state: hash %x/%x; players=%t formations=%t groups=%+v", cycle, want.Hash(), cold.live.world.Hash(), players, formations, got)
				}
				next, _, err := cold.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				raw, err = cold.ExportCurrentSave(next, "next authored Group cycle")
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
