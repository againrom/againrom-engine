package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// A Hard campaign's mission and town SAVEs carry head 3 and record word 2.
func TestReleaseDifficultyRecordWrittenAndLoadedCold(t *testing.T) {
	f := releaseFront(t)
	app := f.App("difficulty record")
	app.Layout(640, 480)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenPicker {
		if err := app.HeadlessActivate("@first"); err != nil {
			t.Fatal(err)
		}
	}
	if err := headlessCreateCharacter(app, HeadlessCharacter{Name: "Witness", Difficulty: "hard"}); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap || f.Difficulty != mapload.DifficultyHard || f.liveMission != 10 {
		t.Fatalf("new game: screen=%s level=%d mission=%d", app.Screen(), f.Difficulty, f.liveMission)
	}
	save := func(onMap bool) []byte {
		t.Helper()
		s, label, err := f.Snapshot(onMap)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		if head, word := difficultyWords(t, raw); head != 3 || word != 2 {
			t.Fatalf("SAVE on map %v wrote head %d record word %d, want 3 and 2", onMap, head, word)
		}
		return raw
	}
	cold := func(raw []byte, wantTown bool) {
		t.Helper()
		g := releaseFront(t)
		g.SetDeterministicFrames(true)
		open, town, err := g.RestoreOriginal(raw)
		if err != nil || town != wantTown {
			t.Fatalf("cold LOAD: town=%v err=%v", town, err)
		}
		if !town {
			if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
				t.Fatal(err)
			}
		}
		if g.Difficulty != mapload.DifficultyHard {
			t.Fatalf("cold LOAD town=%v at %d, want Hard", town, g.Difficulty)
		}
	}
	cold(save(true), false)
	f.FinishMission(10, f.liveParty, f.live.world, f.live.mission.ids)
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(20)(); err != nil {
		t.Fatal(err)
	}
	f.FinishMission(20, f.liveParty, f.live.world, f.live.mission.ids)
	if !f.Town.Open() {
		t.Fatal("mission 20 did not reach the town")
	}
	cold(save(false), true)
}
