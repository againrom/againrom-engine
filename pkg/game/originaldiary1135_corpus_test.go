//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestDiaryCorpusAudit1135(t *testing.T) {
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

	total, worlds, cities, unreadable, nonEmpty, mismatches := 0, 0, 0, 0, 0, 0
	walkErr := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".sav" {
			return nil
		}
		total++
		rel, relErr := filepath.Rel(corpus, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
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
			wantChars, wantPlayerRec, walkErr := source.PartyWalk()
			if wantPlayerRec == nil {
				t.Fatalf("PartyWalk: no player record, err=%v", walkErr)
			}
			wantPlayer, wantHasPlayer, err := sav.PlayerDiary(wantPlayerRec)
			if err != nil {
				t.Fatalf("PlayerDiary: %v", err)
			}
			wantByOff := map[int]sav.Character{}
			seen := map[int]bool{}
			for _, c := range wantChars {
				if seen[c.Off] || !c.HasDiary {
					continue
				}
				seen[c.Off] = true
				wantByOff[c.Off] = c
			}
			want := len(wantByOff)
			if wantHasPlayer {
				want++
			}
			if want > 1 {
				nonEmpty++
			}

			ms, report, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
			if err != nil {
				t.Fatalf("ResumeOriginalSave: %v", err)
			}
			if !report.DiariesApplied {
				t.Fatal("a world-half save did not apply diaries")
			}
			got := ms.World.SavedDiaries()
			if report.Diaries != len(got) {
				t.Fatalf("report.Diaries = %d, len(SavedDiaries()) = %d", report.Diaries, len(got))
			}

			gotHasPlayer := false
			// Keyed directly by EntityID, d.Owner.Actor's own type: no reverse
			// lookup is needed on this side, unlike exportOriginalDiaries, which
			// starts from an EntityID with no file record in hand at all.
			gotByEntity := map[sim.EntityID]sim.SavedDiary{}
			for _, d := range got {
				if d.Owner.Player {
					gotHasPlayer = true
					if d.Length != wantPlayer.Length || !sameLiveDiaryEntries1135(d.Entries, wantPlayer.Entries) {
						mismatches++
						t.Errorf("player diary length/entries = %d/%+v, want %d/%+v",
							d.Length, d.Entries, wantPlayer.Length, wantPlayer.Entries)
					}
					continue
				}
				gotByEntity[d.Owner.Actor] = d
			}
			if gotHasPlayer != wantHasPlayer {
				mismatches++
				t.Errorf("player diary present = %v, want %v", gotHasPlayer, wantHasPlayer)
			}
			// Every carried actor diary must trace to a wanted file record, and
			// every wanted record whose actor was admitted must be carried:
			// registryTarget skips a Humanoid the registry never admitted (dead,
			// excluded), which is why this checks admitted membership, not raw
			// wantByOff population, on the same tolerance applyOriginalDiaries
			// itself already applies.
			for off, wc := range wantByOff {
				bound, ok := ms.actorRegistry.actor(off)
				if !ok {
					continue // Not admitted: applyOriginalDiaries skips it too.
				}
				gd, ok := gotByEntity[bound.ID]
				if !ok {
					mismatches++
					t.Errorf("actor at %d (entity %d) has a file diary but nothing was carried", off, bound.ID)
					continue
				}
				if gd.Length != wc.Diary.Length || !sameLiveDiaryEntries1135(gd.Entries, wc.Diary.Entries) {
					mismatches++
					t.Errorf("actor at %d diary length/entries = %d/%+v, want %d/%+v",
						off, gd.Length, gd.Entries, wc.Diary.Length, wc.Diary.Entries)
				}
			}

			// Native export: write the live state back into a fresh decode of
			// this file's own bytes, then compare the complete body, on
			// TestMoverRouteCorpusAudit1134's own whole-body precedent, plus the
			// re-decoded Diary content's own entry values (F-2: Length and
			// len(Entries) alone do not prove a carried value round-trips).
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
			if exportedHasPlayer != wantHasPlayer || exportedPlayer.Length != wantPlayer.Length ||
				!sameSavDiaryEntries1135(exportedPlayer.Entries, wantPlayer.Entries) {
				mismatches++
				t.Errorf("native export player diary = present=%v %+v, want present=%v %+v",
					exportedHasPlayer, exportedPlayer, wantHasPlayer, wantPlayer)
			}
			t.Logf("player diary present=%v length=%d entries=%d; %d actor diaries carried",
				wantHasPlayer, wantPlayer.Length, len(wantPlayer.Entries), len(gotByEntity))
		})
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	t.Logf("audited %d file(s): %d world-half, %d between-mission, %d unreadable, %d with more than one diary, %d mismatching",
		total, worlds, cities, unreadable, nonEmpty, mismatches)
}
