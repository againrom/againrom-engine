package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func savDiaryEntriesToSaved(entries []sav.DiaryEntry) []sim.SavedDiaryEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]sim.SavedDiaryEntry, len(entries))
	for i, e := range entries {
		out[i] = sim.SavedDiaryEntry{Index: e.Index, Count: e.Count, Remaining: e.Remaining}
	}
	return out
}

func savedDiaryEntriesToSav(entries []sim.SavedDiaryEntry) []sav.DiaryEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]sav.DiaryEntry, len(entries))
	for i, e := range entries {
		out[i] = sav.DiaryEntry{Index: e.Index, Count: e.Count, Remaining: e.Remaining}
	}
	return out
}

// applyOriginalDiaries is shared by both original LOAD doors, called after
// admitOriginalActorRegistry and before restoreOriginalActors: a
// Humanoid-owned Diary's owner resolves to a live EntityID through
// registryTarget (SAV-669). It must also run before
// restoreOriginalActorStock (originalholdings.go): that call's own staging
// round trip carries whatever this call already applied through the wire
// form's own carried-resume span (Form85) rather than a priming call, but
// the value still has to be on ms.World before that round trip's own encode,
// on applyOriginalProjectiles' own ordering standing
// (originalprojectiles.go).
//
// playerRec and chars are PartyWalk's own two returns (SAV-PLAYER-028,
// SAV-DIARY-042), walked independently of RestoreParty's own
// persistent-filtered, reordered ms.Party: SAV-669 finds every one of 132
// corpus Humanoid-owned Diaries reachable through this same raw list,
// filtered party membership or not, so nothing here depends on which
// characters RestoreParty kept. chars may repeat one archive record more
// than once (PartyWalk's own contract); this dedupes by Character.Off before
// resolving an owner, on f.Party()'s own precedent for the same list.
func applyOriginalDiaries(ms *Mission, playerRec *sav.Record, chars []sav.Character, r *OriginalSaveResume) error {
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original diaries: mission has no world")
	}
	var diaries []sim.SavedDiary
	playerDiary, hasPlayer, err := sav.PlayerDiary(playerRec)
	if err != nil {
		return fmt.Errorf("original diaries: %w", err)
	}
	if hasPlayer {
		diaries = append(diaries, sim.SavedDiary{
			Owner:   sim.SavedDiaryOwner{Player: true},
			Length:  playerDiary.Length,
			Entries: savDiaryEntriesToSaved(playerDiary.Entries),
		})
	}
	seen := make(map[int]bool, len(chars))
	for _, c := range chars {
		if seen[c.Off] || !c.HasDiary {
			continue
		}
		seen[c.Off] = true
		bound, terr := registryTarget(ms, c.Off)
		if terr != nil {
			return fmt.Errorf("original diaries: %w", terr)
		}
		if bound == nil {
			continue // Not an admitted living actor: dead, excluded, or otherwise not joined.
		}
		diaries = append(diaries, sim.SavedDiary{
			Owner:   sim.SavedDiaryOwner{Actor: bound.ID},
			Length:  c.Diary.Length,
			Entries: savDiaryEntriesToSaved(c.Diary.Entries),
		})
	}
	if err := ms.World.ImportOriginalDiaries(diaries); err != nil {
		return fmt.Errorf("original diaries: %w", err)
	}
	if r != nil {
		r.DiariesApplied = true
	}
	return nil
}

// exportOriginalDiaries writes a world's carried Diary list back into a
// decoded original save, the counterpart of the LOAD that carried it. It
// patches each existing Diary record matched by owner: the walked Player's
// inline Diary, and for an actor the Diary of the record whose archive index
// the entity's source binding, or a departed owner's retained source, names.
// Any Player's Human or Humanoid may own one. It has no production caller and
// refuses a save with no world half.
func exportOriginalDiaries(f *sav.File, w *sim.World) error {
	if w == nil {
		return fmt.Errorf("original diaries export: nil world")
	}
	archiveIndexByEntity := make(map[sim.EntityID]uint16, len(w.Entities()))
	for _, d := range w.OriginalDeadActors() {
		if d.Source.ArchiveIndex != 0 {
			archiveIndexByEntity[d.ID] = d.Source.ArchiveIndex
		}
	}
	for _, e := range w.Entities() {
		if e.SourceBinding.ArchiveIndex != 0 {
			archiveIndexByEntity[e.ID] = e.SourceBinding.ArchiveIndex
		}
	}
	return exportDiaries(f, w.SavedDiaries(), archiveIndexByEntity)
}

func exportDiaries(f *sav.File, diaries []sim.SavedDiary, archiveIndexByEntity map[sim.EntityID]uint16) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original diaries export: this save has no world session")
	}
	_, playerRec, err := f.PartyWalk()
	if playerRec == nil {
		if err != nil {
			return fmt.Errorf("original diaries export: %w", err)
		}
		return fmt.Errorf("original diaries export: this save has no player record")
	}
	owners, err := f.DiaryOwners()
	if err != nil {
		return fmt.Errorf("original diaries export: %w", err)
	}
	actorDiaries := map[uint16]*sav.Record{}
	for _, o := range owners {
		if o.Actor && o.Diary != nil {
			actorDiaries[o.ArchiveIndex] = o.Diary
		}
	}
	for _, d := range diaries {
		var rec *sav.Record
		if d.Owner.PlayerID != 0 {
			continue // This export patches the walked Player and actors only.
		}
		if d.Owner.Player {
			refs := playerRec.Refs["Diary"]
			if len(refs) == 0 {
				return fmt.Errorf("original diaries export: player record has no diary to patch")
			}
			rec = refs[0]
		} else {
			archiveIndex, ok := archiveIndexByEntity[d.Owner.Actor]
			if !ok {
				return fmt.Errorf("original diaries export: actor %d has no archive index to resolve", d.Owner.Actor)
			}
			if rec = actorDiaries[archiveIndex]; rec == nil {
				return fmt.Errorf("original diaries export: actor %d (archive index %d) has no diary record to patch", d.Owner.Actor, archiveIndex)
			}
		}
		if err := f.SetDiary(rec, sav.Diary{Length: d.Length, Entries: savedDiaryEntriesToSav(d.Entries), Self: rec.Value["D2C"]}); err != nil {
			return fmt.Errorf("original diaries export: %w", err)
		}
	}
	return nil
}
