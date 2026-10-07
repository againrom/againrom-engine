package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestMissionStartAutosaveWritesAcceptedTickZero(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("mission start autosave")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	rows, err := store.List()
	if err != nil || len(rows) != 1 {
		t.Fatalf("accepted mission needs one automatic SAV: %v %v", rows, err)
	}
	if rows[0].Name != "game0000.sav" || rows[0].Label != "autosave start mission 10 - mission 10" {
		t.Fatalf("automatic SAV row = %+v", rows[0])
	}
	raw, err := store.Read(rows[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	if err != nil || asciiLabel(file.Label, store.Selector) != "autosave start mission 10" {
		t.Fatal("automatic SAV header label", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.Head.Mission != 10 || doc.Head.CounterA != 0 || f.live.world.Tick() != 0 {
		t.Fatalf("automatic SAV must capture mission 10 at tick 0: %+v %v", doc.Head, err)
	}
}

func TestMissionStartAutosavePreparationAndLoadWriteNothing(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("prepared mission autosave")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	open, err := f.prepareNewGameWith(10, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rows, err := store.List(); err != nil || len(rows) != 0 || f.live != nil {
		t.Fatalf("preparation committed state or a SAV: %v %v", rows, err)
	}
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	rows, err := store.List()
	if err != nil || len(rows) != 1 {
		t.Fatal("adoption did not write one SAV", rows, err)
	}
	raw, err := store.Read(rows[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	g := currentPoolFixtureFront(t, 91, 92)
	cold := g.App("cold mission autosave")
	g.ConfigureSaveSeams(cold, store, OriginalStore{}, nil)
	if err := cold.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	shown := cold.HeadlessRows()
	if len(shown) != 1 || shown[0].Text != "autosave start mission 10 - mission 10" {
		t.Fatalf("LOAD chooser label = %+v", shown)
	}
	if err := cold.HeadlessActivate("@first"); err != nil || cold.Screen() != ui.ScreenMap {
		t.Fatal("cold LOAD", err, cold.Screen())
	}
	if g.live.world.Tick() != 0 || !g.live.mission.resumed {
		t.Fatal("cold LOAD did not restore tick zero")
	}
	cold.Layout(1280, 960)
	if rows, err := store.List(); err != nil || len(rows) != 1 {
		t.Fatal("LOAD or resize wrote another SAV", rows, err)
	}
	if after, err := store.Read(rows[0].Name); err != nil || !bytes.Equal(raw, after) {
		t.Fatal("LOAD overwrote mission-start SAV", err)
	}
}

func TestMissionStartAutosavePreservesSlotsAndDoesNotRetry(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "occupied slot", true: "failed publication"}[fail], func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91, 92)
			store := SaveStore{Dir: t.TempDir()}
			occupied := filepath.Join(store.Dir, "game0000.sav")
			old := []byte("owner slot remains untouched")
			if err := os.WriteFile(occupied, old, 0600); err != nil {
				t.Fatal(err)
			}
			if fail {
				store.Dir = occupied
			}
			app := f.App("mission slot autosave")
			f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
			if err := app.OpenMission(f.MissionOpener(10)); err != nil || app.Screen() != ui.ScreenMap {
				t.Fatal("save failure prevented mission entry", err, app.Screen())
			}
			if f.live.world.Tick() != 0 {
				t.Fatal("entry advanced the world")
			}
			if fail {
				lines := f.live.view.MessageLines()
				var message string
				for _, line := range lines {
					message += line.Text
				}
				if !strings.Contains(message, "cannot autosave mission start:") {
					t.Fatalf("save failure is not visible on map: %+v", lines)
				}
				f.live.view.ClearMessages()
			} else if _, err := os.Stat(filepath.Join(store.Dir, "game0001.sav")); err != nil {
				t.Fatal("next free slot was not used", err)
			}
			app.Layout(1280, 960)
			if fail && len(f.live.view.MessageLines()) != 0 {
				t.Fatal("layout retried failed autosave")
			}
			if after, err := os.ReadFile(occupied); err != nil || !bytes.Equal(old, after) {
				t.Fatal("entry or failed write replaced occupied slot", err)
			}
		})
	}
}

func TestMissionStartAutosaveUnconfiguredAppDoesNotWrite(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("unconfigured mission")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	app.Layout(1280, 960)
	f.live.tick()
	app.Layout(1024, 768)
	if rows, err := store.List(); err != nil || len(rows) != 0 {
		t.Fatal("late configuration or resize wrote a start save", rows, err)
	}
}

func TestMissionStartAutosaveUsesLastSuccessfulSaveDirectory(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	first := SaveStore{Dir: t.TempDir()}
	next := SaveStore{Dir: t.TempDir()}
	app := f.App("mission autosave current directory")
	f.ConfigureSaveSeams(app, first, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("ordinary SAVE dialog", err, app.Screen())
	}
	if err := app.HeadlessSaveEdit(next.Dir, "chosen directory", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if rows, err := first.List(); err != nil || len(rows) != 1 || rows[0].Name != "game0000.sav" {
		t.Fatal("next mission wrote back into the old SAVE directory", rows, err)
	}
	rows, err := next.List()
	if err != nil || len(rows) != 2 {
		t.Fatal("chosen directory lacks manual and automatic saves", rows, err)
	}
	for _, row := range rows {
		if row.Name == "game0000.sav" && row.Label == "autosave start mission 10 - mission 10" {
			return
		}
	}
	t.Fatal("next entry did not use the last successful SAVE directory", rows)
}

func TestMissionStartAutosaveRawMapAndRefusedEntryWriteNothing(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	f.Maps = []MapEntry{{Source: "10.alm", FromArchive: true}}
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("raw map and refused mission")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(-1)); err == nil {
		t.Fatal("invalid mission was accepted")
	}
	if rows, err := store.List(); err != nil || len(rows) != 0 || f.live != nil {
		t.Fatal("refused entry adopted or wrote a mission", rows, err)
	}
	open := func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		return f.loadMap(0)
	}
	if err := app.OpenMission(open); err != nil || app.Screen() != ui.ScreenMap || f.liveMission != 0 {
		t.Fatal("raw-map production loader", err, app.Screen(), f.liveMission)
	}
	app.Layout(1280, 960)
	if rows, err := store.List(); err != nil || len(rows) != 0 {
		t.Fatal("raw map wrote a mission-start save", rows, err)
	}
}
