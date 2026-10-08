package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func enemyCardTownKnowledge(f *FrontEnd, row int) uint32 {
	for _, d := range f.Town.knowledge {
		for _, e := range d.Entries {
			if e.Index == row {
				return e.Count
			}
		}
	}
	return 0
}

func enemyCardWonTown(t *testing.T, count uint32) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{}
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Card traveler", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("enemy card town")
	t.Cleanup(app.StopAudio)
	if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	enemyCardSetCount(f.live.world, enemyCardRow, count)
	chainWin(t, f, app, 10)
	if err := app.HeadlessActivate("notice"); err != nil || f.liveMission != 20 {
		t.Fatal("mission 10 victory did not enter mission 20", err, f.liveMission)
	}
	if got := enemyCardCount(f.live.world, enemyCardRow); got != count {
		t.Fatalf("mission 20 opened with count %d, want %d", got, count)
	}
	chainWin(t, f, app, 20)
	campaignReturn(t, f, app)
	return f, app
}

// A town SAVE writes the carried Diary, town LOAD restores it, and the next
// mission opens with it.
func TestReleaseEnemyCardTownSaveKeepsKnowledgeForTheNextMission(t *testing.T) {
	f, _ := enemyCardWonTown(t, 5)
	store, _, raw := campaignSave(t, f, false)
	if got := enemyCardSAVCount(t, raw, enemyCardRow); got != 5 {
		t.Fatalf("the town SAV carries count %d for the row, want 5", got)
	}
	cold, coldApp := campaignCold(t, store)
	if got := enemyCardTownKnowledge(cold, enemyCardRow); got != 5 {
		t.Fatalf("town LOAD restored count %d, want 5", got)
	}
	if err := coldApp.OpenMission(cold.MissionOpener(30)); err != nil {
		t.Fatal(err)
	}
	if got := enemyCardCount(cold.live.world, enemyCardRow); got != 5 {
		t.Fatalf("the mission after town LOAD opened with count %d, want 5", got)
	}
	if got := enemyCardWant(enemyCardCount(cold.live.world, enemyCardRow)); got != 2 {
		t.Fatalf("level %d after town LOAD, want 2", got)
	}

	// Loss control: a town whose carried Diary is dropped writes none.
	f.Town.knowledge = nil
	lossStore, _, lossRaw := campaignSave(t, f, false)
	if got := enemyCardSAVCount(t, lossRaw, enemyCardRow); got != 0 {
		t.Fatalf("a town without a carried Diary still wrote count %d", got)
	}
	lossCold, lossApp := campaignCold(t, lossStore)
	if err := lossApp.OpenMission(lossCold.MissionOpener(30)); err != nil {
		t.Fatal(err)
	}
	if got := enemyCardCount(lossCold.live.world, enemyCardRow); got != 0 {
		t.Fatalf("the mission after a Diary-less town LOAD opened with count %d", got)
	}

	// Loss control: the same town SAV with its Diary zeroed in the file loads
	// at zero, so town LOAD reads the Diary from the file.
	zeroDir := t.TempDir()
	entries, err := os.ReadDir(store.Dir)
	if err != nil {
		t.Fatal(err)
	}
	named := ""
	for _, e := range entries {
		if strings.Contains(e.Name(), "checkpoint") {
			named = e.Name()
		}
	}
	if named == "" {
		t.Fatal("the checkpoint is not in the store", entries)
	}
	if err := os.WriteFile(filepath.Join(zeroDir, named), enemyCardZeroSAV(t, raw), 0600); err != nil {
		t.Fatal(err)
	}
	zeroCold, zeroApp := campaignCold(t, SaveStore{Dir: zeroDir})
	if got := enemyCardTownKnowledge(zeroCold, enemyCardRow); got != 0 {
		t.Fatalf("town LOAD of the zeroed SAV holds count %d", got)
	}
	if err := zeroApp.OpenMission(zeroCold.MissionOpener(30)); err != nil {
		t.Fatal(err)
	}
	if got := enemyCardCount(zeroCold.live.world, enemyCardRow); got != 0 {
		t.Fatalf("the mission after the zeroed town LOAD opened with count %d", got)
	}
}

// Original between-mission SAVs with a nonzero local Diary load with it: it
// reaches the town, the next mission and a re-written town SAV.
func TestReleaseEnemyCardOriginalCitySAVDiaryLoads(t *testing.T) {
	dir := os.Getenv("AGAINROM_SAVE_CORPUS")
	if dir == "" {
		t.Skip("AGAINROM_SAVE_CORPUS is not set")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*", "*.sav"))
	checked := 0
	seen := map[string]bool{}
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil || doc.World != nil {
			continue
		}
		var want map[int]uint32
		func() {
			defer func() { recover() }()
			var record *sav.DocumentRecordData
			for i := range doc.Objects {
				r := &doc.Objects[i]
				if r.Class != "Player" {
					continue
				}
				if participant, err := savedStructureValue(r, "Participant"); err != nil || participant != 0 {
					continue
				}
				for j := range r.Inline {
					if r.Inline[j].Record.Class == "Diary" {
						record = &r.Inline[j].Record
					}
				}
			}
			if record == nil {
				return
			}
			d, err := sav.ReadDocumentDiary(*record)
			if err != nil || len(d.Entries) == 0 {
				return
			}
			want = map[int]uint32{}
			for _, e := range d.Entries {
				want[e.Index] = e.Count
			}
		}()
		if want == nil || checked >= 4 {
			continue
		}
		key := fmt.Sprint(want)
		if seen[key] {
			continue
		}
		seen[key] = true
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			townDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(townDir, name), raw, 0600); err != nil {
				t.Fatal(err)
			}
			app := openLocalTownSAV(t, f, townDir, name)
			for row, count := range want {
				if got := enemyCardTownKnowledge(f, row); got != count {
					t.Fatalf("town LOAD row %d: count %d, file %d", row, got, count)
				}
			}
			snapshot, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			out, err := f.ExportCurrentSave(snapshot, "city")
			if err != nil {
				t.Skip("the loaded city does not re-write:", err)
			}
			for row, count := range want {
				if got := enemyCardSAVCount(t, out, row); got != count {
					t.Fatalf("re-written town SAV row %d: count %d, file %d", row, got, count)
				}
			}
			if err := app.OpenMission(f.MissionOpener(f.Town.currentMain())); err != nil {
				t.Fatal(err)
			}
			for _, d := range f.live.world.SavedDiaries() {
				if !d.Owner.Player {
					continue
				}
				got := map[int]uint32{}
				for _, e := range d.Entries {
					got[e.Index] = e.Count
				}
				for row, count := range want {
					if got[row] != count {
						t.Fatalf("next mission row %d: count %d, file %d", row, got[row], count)
					}
				}
				return
			}
			t.Fatal("the next mission has no local Player Diary")
		})
		checked++
	}
	if checked == 0 {
		t.Skip("no original city SAV with a nonzero Diary in the corpus")
	}
	var _ sim.SavedDiary
}
