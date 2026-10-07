package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// The installed mission's automatic saves reach the disk through the worker,
// and carry exactly the bytes the frame thread would have written for the same
// state.
func TestReleaseAutosaveWorkerWritesTheFrameThreadBytesInMission(t *testing.T) {
	f := releaseFront(t)
	store := SaveStore{Dir: t.TempDir()}
	store.Selector = f.textSelector()
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	now := time.Unix(100, 0)
	app := f.App("async autosave release")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	start, err := store.Read("game0000.sav")
	if err != nil {
		t.Fatal("mission start autosave missing", err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil || f.live.world.Tick() != 0 {
		t.Fatal("tick zero capture", err)
	}
	want, _, err := f.playerMissionSave(s, "autosave start mission 10")
	if err != nil || !bytes.Equal(start, want) {
		t.Fatal("worker mission-start bytes differ from the frame-thread export", err)
	}
	checkNoSharedWritableMemory(t, f, s, &struct{ W *mapWorld }{f.live})

	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	timed, err := os.ReadFile(filepath.Join(store.Dir, "timed-autosave-1.sav"))
	if err != nil {
		t.Fatal("timed autosave missing", err)
	}
	s, _, err = f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	want, _, err = f.playerMissionSave(s, "timed autosave 1")
	if err != nil || !bytes.Equal(timed, want) {
		t.Fatal("worker timed bytes differ from the frame-thread export", err)
	}
	if bytes.Equal(timed, start) {
		t.Fatal("timed autosave equals the mission-start bytes")
	}
	for range 200 {
		f.live.tick()
	}
	if after, err := os.ReadFile(filepath.Join(store.Dir, "timed-autosave-1.sav")); err != nil || !bytes.Equal(after, timed) {
		t.Fatal("later play changed a finished autosave", err)
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatal("autosave left the map")
	}
}

func TestReleaseAutosaveWorkerBytesEqualFrameThreadBytesInTown(t *testing.T) {
	f := currentTown(t, nil, nil)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := f.playerMissionSave(s, "timed autosave 2")
	if err != nil {
		t.Fatal(err)
	}
	if got := asyncExport(t, f, s, "timed autosave 2"); !bytes.Equal(got, want) {
		t.Fatal("installed town worker bytes differ from the frame-thread export")
	}
	checkNoSharedWritableMemory(t, f, s, &struct{ T *Town }{f.Town})
}
