package game

import (
	"os"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestDiaryCorpusAudit1135(t *testing.T) {
	sessionCorpusAudit(t)
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name gameversions/saves")
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	f, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	f.SetDeterministicFrames(true)
	sameDiary := func(present bool, got sav.Diary, wantPresent bool, want sav.Diary) bool {
		return present == wantPresent && got.Length == want.Length && sameSavDiaryEntries1135(got.Entries, want.Entries)
	}

	total, worlds, cities, unreadable, nonEmpty, mismatches := 0, 0, 0, 0, 0, 0
	otherPlayers, droppedFiles, saved := 0, 0, 0
	walkErr := walkFirstGameSaves(corpus, func(path, rel string) error {
		total++
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			source, err := sav.Open(raw)
			if err != nil {
				unreadable++
				t.Logf("unreadable, skipped: %v", err)
				return
			}
			if source.World == nil {
				cities++
				t.Log("between-mission save: no world-state store, nothing to compare")
				return
			}
			worlds++
			walked, wantPlayerRec, walkErr := source.PartyWalk()
			if wantPlayerRec == nil {
				t.Fatalf("PartyWalk: no player record, err=%v", walkErr)
			}
			wantPlayer, wantHasPlayer, err := sav.PlayerDiary(wantPlayerRec)
			if err != nil {
				t.Fatalf("PlayerDiary: %v", err)
			}
			// Every Human or Humanoid LOAD builds as an entity carries its
			// Diary, whichever Player owns it (importSavedDiaries).
			graph, err := source.ActorGraph()
			if err != nil {
				t.Fatalf("ActorGraph: %v", err)
			}
			var owners []sav.ActorRecord
			for _, a := range archiveActorsForAudit(graph) {
				if a.Character.HasDiary {
					owners = append(owners, a)
				}
			}
			slices.SortFunc(owners, func(a, b sav.ActorRecord) int { return int(a.ArchiveIndex) - int(b.ArchiveIndex) })
			walkedOff := map[int]bool{}
			for _, c := range walked {
				walkedOff[c.Off] = true
			}
			fileOther := 0
			for _, a := range owners {
				if !walkedOff[a.Off] {
					fileOther++
				}
			}
			otherPlayers += fileOther
			if len(owners) > 0 || wantHasPlayer && len(wantPlayer.Entries) > 0 {
				nonEmpty++
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatalf("DecodeDocumentData: %v", err)
			}
			dropped, err := diariesDroppedAtLoad(&doc)
			if err != nil {
				t.Fatalf("diariesDroppedAtLoad: %v", err)
			}
			if len(dropped) > 0 {
				droppedFiles++
			}

			loaded, err := loadOriginalFront(f, raw)
			if err != nil {
				t.Fatalf("RestoreOriginal: %v", err)
			}
			ms := loaded.CurrentMission()
			report := ms.OriginalSaveReport()
			if !report.DiariesApplied {
				t.Fatal("a world-half save did not apply diaries")
			}
			got := ms.World.SavedDiaries()
			// Another Player's Diary is carried unless it is at its
			// construction default.
			others, err := otherPlayerDiaries(&doc)
			if err != nil {
				t.Fatalf("otherPlayerDiaries: %v", err)
			}
			wantOthers, gotOthers := 0, 0
			for _, d := range others {
				if len(d.Entries) != 0 || d.Length != ms.World.DiaryRows() {
					wantOthers++
				}
			}
			for _, d := range got {
				if d.Owner.PlayerID != 0 {
					gotOthers++
				}
			}
			if gotOthers != wantOthers {
				mismatches++
				t.Errorf("LOAD carries %d other Player Diaries, the file has %d beyond the default", gotOthers, wantOthers)
			}
			// The report counts every Diary the LOAD keeps, another
			// Player's included.
			if report.Diaries != len(got) {
				mismatches++
				t.Errorf("report.Diaries = %d with %d other Player Diaries, len(SavedDiaries()) = %d", report.Diaries, gotOthers, len(got))
			}
			gotBy := map[sim.SavedDiaryOwner]sim.SavedDiary{}
			for _, d := range got {
				gotBy[d.Owner] = d
			}
			want := map[sim.SavedDiaryOwner]sav.Diary{}
			if player := (sim.SavedDiaryOwner{Player: true}); wantHasPlayer && !dropped[player] {
				want[player] = wantPlayer
			}
			droppedActor := map[uint16]bool{}
			for _, a := range owners {
				bound, ok := ms.actorRegistry.actor(a.Off)
				if !ok {
					mismatches++
					t.Errorf("actor %d: LOAD builds it but the registry has no binding", a.ArchiveIndex)
					continue
				}
				if owner := (sim.SavedDiaryOwner{Actor: bound.ID}); dropped[owner] {
					droppedActor[a.ArchiveIndex] = true
				} else {
					want[owner] = a.Character.Diary
				}
			}
			for owner, wd := range want {
				gd, ok := gotBy[owner]
				if !ok {
					mismatches++
					t.Errorf("owner %+v has a file Diary but nothing was carried", owner)
					continue
				}
				if gd.Length != wd.Length || !sameLiveDiaryEntries1135(gd.Entries, wd.Entries) {
					mismatches++
					t.Errorf("owner %+v diary length/entries = %d/%+v, want %d/%+v", owner, gd.Length, gd.Entries, wd.Length, wd.Entries)
				}
			}
			for owner := range gotBy {
				if _, ok := want[owner]; !ok && owner.PlayerID == 0 {
					mismatches++
					t.Errorf("owner %+v carries a Diary the file does not give it", owner)
				}
			}

			// Native export: write the live state back into a fresh decode of
			// this file's own bytes and compare the complete body.
			target, err := sav.Open(raw)
			if err != nil {
				t.Fatalf("sav.Open (export target): %v", err)
			}
			if err := exportOriginalDiaries(target, ms.World); err != nil {
				t.Fatalf("exportOriginalDiaries: %v", err)
			}
			written, err := sav.Open(target.Marshal())
			if err != nil {
				t.Fatalf("sav.Open (re-decode written): %v", err)
			}
			if len(written.Body) != len(source.Body) {
				mismatches++
				t.Errorf("native export changed the body length: %d -> %d", len(source.Body), len(written.Body))
			} else {
				diff := 0
				for i := range source.Body {
					if source.Body[i] != written.Body[i] {
						diff++
					}
				}
				if diff > 0 {
					mismatches++
					t.Errorf("native export of the diary state does not reproduce the source file: %d byte(s) differ", diff)
				}
			}
			_, writtenPlayerRec, _ := written.PartyWalk()
			exportedPlayer, exportedHasPlayer, err := sav.PlayerDiary(writtenPlayerRec)
			if err != nil {
				t.Fatalf("PlayerDiary (re-decode written): %v", err)
			}
			if !sameDiary(exportedHasPlayer, exportedPlayer, wantHasPlayer, wantPlayer) {
				mismatches++
				t.Errorf("native export player diary = present=%v %+v, want present=%v %+v",
					exportedHasPlayer, exportedPlayer, wantHasPlayer, wantPlayer)
			}

			// The player's SAVE writes every carried Diary, joined by Identity:
			// a SAVE renumbers archive indices. A Diary dropped as absent is no
			// oracle: its writer did not hold it (DIV-2885).
			s, _, err := loaded.Snapshot(true)
			if err != nil {
				t.Fatalf("Snapshot: %v", err)
			}
			out, _, err := loaded.playerMissionSave(s, "diary audit")
			if err != nil {
				t.Fatalf("SAVE: %v", err)
			}
			back, err := sav.Open(out)
			if err != nil {
				t.Fatalf("sav.Open (SAVE): %v", err)
			}
			_, savedPlayerRec, _ := back.PartyWalk()
			savedPlayer, savedHasPlayer, err := sav.PlayerDiary(savedPlayerRec)
			if err != nil {
				t.Fatalf("PlayerDiary (SAVE): %v", err)
			}
			if player := (sim.SavedDiaryOwner{Player: true}); !dropped[player] && !sameDiary(savedHasPlayer, savedPlayer, wantHasPlayer, wantPlayer) {
				mismatches++
				t.Errorf("SAVE player diary = present=%v %+v, want present=%v %+v",
					savedHasPlayer, savedPlayer, wantHasPlayer, wantPlayer)
			}
			backDoc, err := sav.DecodeDocumentData(out)
			if err != nil {
				t.Fatalf("DecodeDocumentData (SAVE): %v", err)
			}
			savedOthers, err := otherPlayerDiaries(&backDoc)
			if err != nil {
				t.Fatalf("otherPlayerDiaries (SAVE): %v", err)
			}
			if len(savedOthers) != len(others) {
				mismatches++
				t.Errorf("SAVE wrote %d other Player Diaries, the file has %d", len(savedOthers), len(others))
			} else {
				for i := range others {
					if !sameDiary(true, savedOthers[i], true, others[i]) {
						mismatches++
						t.Errorf("SAVE other Player Diary %d = %+v, want %+v", i, savedOthers[i], others[i])
					}
				}
			}
			savedGraph, err := back.ActorGraph()
			if err != nil {
				t.Fatalf("SAVE's ActorGraph: %v", err)
			}
			byIdentity := map[uint32][]sav.ActorRecord{}
			for _, a := range savedGraph.Actors {
				byIdentity[a.Identity] = append(byIdentity[a.Identity], a)
			}
			for _, a := range owners {
				if droppedActor[a.ArchiveIndex] {
					continue
				}
				sa := byIdentity[a.Identity]
				if a.Identity == 0 || len(sa) != 1 {
					mismatches++
					t.Errorf("actor %d (identity %d): SAVE wrote %d record(s)", a.ArchiveIndex, a.Identity, len(sa))
					continue
				}
				sd := sa[0].Character.Diary
				if !sameDiary(sa[0].Character.HasDiary, sd, true, a.Character.Diary) {
					mismatches++
					t.Errorf("actor %d (identity %d): SAVE diary = present=%v %+v, want %+v",
						a.ArchiveIndex, a.Identity, sa[0].Character.HasDiary, sd, a.Character.Diary)
				}
				saved++
			}
			t.Logf("player diary present=%v length=%d entries=%d; %d actor diaries in the file (%d of other Players), %d dropped as absent, %d carried",
				wantHasPlayer, wantPlayer.Length, len(wantPlayer.Entries), len(owners), fileOther, len(dropped), len(got))
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d with a Diary beyond an empty Player's, %d actor diaries of other Players, %d file(s) with diaries dropped as absent, %d actor diaries checked through SAVE, %d mismatching",
		total, worlds, cities, unreadable, nonEmpty, otherPlayers, droppedFiles, saved, mismatches)
}

// otherPlayerDiaries reads each non-local Player's own Diary in object order.
func otherPlayerDiaries(doc *sav.DocumentData) ([]sav.Diary, error) {
	var out []sav.Diary
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Player" {
			continue
		}
		participant, err := savedStructureValue(r, "Participant")
		if err != nil {
			return nil, err
		}
		if participant == 0 {
			continue
		}
		for _, in := range r.Inline {
			if in.Record.Class == "Diary" {
				d, err := sav.ReadDocumentDiary(in.Record)
				if err != nil {
					return nil, err
				}
				out = append(out, d)
			}
		}
	}
	return out, nil
}
