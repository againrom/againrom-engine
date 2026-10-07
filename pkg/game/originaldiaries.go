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
		r.Diaries, r.DiariesApplied = len(diaries), true
	}
	return nil
}

// exportOriginalDiaries writes a world's carried Diary list back into a
// decoded original save's own party graph — the counterpart write to
// applyOriginalDiaries, exportOriginalProjectiles' own building-block role
// for a different section of the same save. Unlike Projectiles, a Diary has
// no flat store to replace outright: it patches each existing Diary record
// found by re-walking f's own current party graph, matched to a carried
// SavedDiary by owner — the Player record for a Player owner, and by
// ArchiveIndex for an Actor owner: Entity.SourceBinding.ArchiveIndex and
// sav.Character.ArchiveIndex (party.go) are the same archive object-counter
// tag (Record.Index) admitOriginalActorRegistry already copies onto
// SourceBinding, and unlike MapUnitID it does not depend on the actor having
// a map placement — a save's own hero carries no MapUnitID
// (actorCharacter's own doc, party.go), and every corpus release fixture
// this story tests against is exactly that case.
//
// NO PRODUCTION CALLER INVOKES THIS. This project has no production mission
// SAVE path yet (owner rule); it is an unshipped round-trip building block,
// checked only by the corpus audit (originaldiary1135_corpus_test.go).
//
// IT NEVER TOUCHES THE BETWEEN-MISSION FORM, on exportOriginalProjectiles'
// own rule: a city save has no world half, so f.World == nil there refuses.
func exportOriginalDiaries(f *sav.File, w *sim.World) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original diaries export: this save has no world session")
	}
	if w == nil {
		return fmt.Errorf("original diaries export: nil world")
	}
	_, playerRec, err := f.PartyWalk()
	if playerRec == nil {
		if err != nil {
			return fmt.Errorf("original diaries export: %w", err)
		}
		return fmt.Errorf("original diaries export: this save has no player record")
	}
	actorRecByArchiveIndex := make(map[uint16]*sav.Record, len(playerRec.Refs["Actors"]))
	for _, a := range playerRec.Refs["Actors"] {
		if a.Index != 0 {
			actorRecByArchiveIndex[a.Index] = a
		}
	}
	archiveIndexByEntity := make(map[sim.EntityID]uint16, len(w.Entities()))
	for _, e := range w.Entities() {
		if e.SourceBinding.ArchiveIndex != 0 {
			archiveIndexByEntity[e.ID] = e.SourceBinding.ArchiveIndex
		}
	}
	for _, d := range w.SavedDiaries() {
		entries := savedDiaryEntriesToSav(d.Entries)
		var refs []*sav.Record
		if d.Owner.Player {
			refs = playerRec.Refs["Diary"]
			if len(refs) == 0 {
				return fmt.Errorf("original diaries export: player record has no diary to patch")
			}
		} else {
			archiveIndex, ok := archiveIndexByEntity[d.Owner.Actor]
			if !ok {
				return fmt.Errorf("original diaries export: actor %d has no archive index to resolve", d.Owner.Actor)
			}
			actorRec, ok := actorRecByArchiveIndex[archiveIndex]
			if !ok {
				return fmt.Errorf("original diaries export: no file record for actor %d (archive index %d)", d.Owner.Actor, archiveIndex)
			}
			refs = actorRec.Refs["Diary"]
			if len(refs) == 0 {
				return fmt.Errorf("original diaries export: actor %d's own record has no diary to patch", d.Owner.Actor)
			}
		}
		if err := f.SetDiary(refs[0], sav.Diary{Length: d.Length, Entries: entries, Self: refs[0].Value["D2C"]}); err != nil {
			return fmt.Errorf("original diaries export: %w", err)
		}
	}
	return nil
}
