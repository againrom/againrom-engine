package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func savedDiaryRecords(state *SnapshotSAVDocument, world *sim.World) (map[sim.SavedDiaryOwner]*sav.DocumentRecordData, error) {
	if world == nil {
		return nil, fmt.Errorf("SAV diaries have no current world")
	}
	out := map[sim.SavedDiaryOwner]*sav.DocumentRecordData{}
	for i := range state.Document.Objects {
		r := &state.Document.Objects[i]
		if r.Class != "Player" {
			continue
		}
		participant, err := savedStructureValue(r, "Participant")
		if err != nil {
			return nil, err
		}
		if participant != 0 {
			continue
		}
		for j := range r.Inline {
			if r.Inline[j].Record.Class == "Diary" {
				owner := sim.SavedDiaryOwner{Player: true}
				if out[owner] != nil {
					return nil, fmt.Errorf("SAV diaries have more than one local Player")
				}
				out[owner] = &r.Inline[j].Record
			}
		}
	}
	actors := slices.Clone(state.Actors)
	for _, d := range world.OriginalDeadActors() {
		if slices.ContainsFunc(actors, func(b SnapshotSAVActor) bool { return b.EntityID == d.ID }) {
			continue
		}
		var index uint16
		for i := range state.Document.Objects {
			r := &state.Document.Objects[i]
			if r.Class != savedActorClass(d.Source.Class) {
				continue
			}
			key, err := savedStructureValue(r, "Identity")
			if err == nil && key == d.Source.Identity {
				if index != 0 {
					return nil, fmt.Errorf("dead Diary owner has ambiguous identity")
				}
				index = uint16(i + 1)
			}
		}
		if index == 0 {
			return nil, fmt.Errorf("dead Diary owner has no exact object binding")
		}
		actors = append(actors, SnapshotSAVActor{EntityID: d.ID, ObjectIndex: index, Retired: true})
	}
	for _, b := range actors {
		r := &state.Document.Objects[b.ObjectIndex-1]
		if r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		refs, ok := savedObjectRefs(r, "Diary")
		if !ok {
			return nil, fmt.Errorf("SAV actor lacks Diary field")
		}
		if len(refs) != 1 || refs[0] == 0 {
			continue
		}
		if int(refs[0]) > len(state.Document.Objects) {
			return nil, fmt.Errorf("SAV actor Diary is outside graph")
		}
		out[sim.SavedDiaryOwner{Actor: b.EntityID}] = &state.Document.Objects[refs[0]-1]
	}
	return out, nil
}

func importSavedDiaries(ms *Mission, state *SnapshotSAVDocument) error {
	records, err := savedDiaryRecords(state, ms.World)
	if err != nil {
		return err
	}
	var diaries []sim.SavedDiary
	// Player first, then the binding's canonical entity order. Never map order.
	owners := []sim.SavedDiaryOwner{{Player: true}}
	for owner := range records {
		if !owner.Player {
			owners = append(owners, owner)
		}
	}
	slices.SortFunc(owners[1:], func(a, b sim.SavedDiaryOwner) int { return int(a.Actor) - int(b.Actor) })
	for _, owner := range owners {
		if r := records[owner]; r != nil {
			d, err := sav.ReadDocumentDiary(*r)
			if err != nil {
				return err
			}
			diaries = append(diaries, sim.SavedDiary{Owner: owner, Length: d.Length, Entries: savDiaryEntriesToSaved(d.Entries)})
		}
	}
	return ms.World.ImportOriginalDiaries(diaries)
}

func projectSavedDiaries(state *SnapshotSAVDocument, world *sim.World) error {
	records, err := savedDiaryRecords(state, world)
	if err != nil {
		return err
	}
	for _, d := range world.SavedDiaries() {
		r := records[d.Owner]
		if r == nil {
			// A departed actor with no written record takes its Diary with
			// it (DIV-2501); a live owner must have its record.
			if _, live := world.Entity(d.Owner.Actor); !d.Owner.Player && !live {
				continue
			}
			return worldSaveUnsupportedf("current Diary owner has no exact object binding")
		}
		next, err := sav.ProjectDocumentDiary(*r, sav.Diary{Length: d.Length, Entries: savedDiaryEntriesToSav(d.Entries)})
		if err != nil {
			return err
		}
		*r = next
	}
	return nil
}
