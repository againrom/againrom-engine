package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// gatedSaveFiles blocks the first temporary-file creation until released, which
// holds an automatic save inside its write.
type gatedSaveFiles struct {
	osSaveFileOps
	entered chan struct{}
	gate    chan struct{}
	once    sync.Once
}

func newGatedSaveFiles() *gatedSaveFiles {
	return &gatedSaveFiles{entered: make(chan struct{}), gate: make(chan struct{})}
}

func (g *gatedSaveFiles) CreateTemp(dir, pattern string) (saveTempFile, error) {
	g.once.Do(func() { close(g.entered) })
	<-g.gate
	return g.osSaveFileOps.CreateTemp(dir, pattern)
}

func asyncExport(t *testing.T, f *FrontEnd, s Snapshot, label string) []byte {
	t.Helper()
	view, detached := f.detachedExporter(s)
	q := newAutosaveQueue()
	var raw []byte
	q.submit(false, false, "", func() autosaveResult {
		var err error
		raw, _, err = view.playerMissionSave(detached, label)
		return autosaveResult{err: err}
	})
	q.wait()
	if q.busy() {
		t.Fatal("queue busy after wait")
	}
	if r := q.take(false); len(r) != 1 || r[0].err != nil {
		t.Fatalf("worker result %+v", r)
	}
	return raw
}

func TestAutosaveWorkerBytesEqualFrameThreadBytesInMission(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("async bytes")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for _, ticks := range []int{0, 16, 37} {
		for range ticks {
			f.live.tick()
		}
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		want, _, err := f.playerMissionSave(s, "timed autosave 1")
		if err != nil {
			t.Fatal(err)
		}
		if got := asyncExport(t, f, s, "timed autosave 1"); !bytes.Equal(got, want) {
			t.Fatalf("mission worker bytes differ from the frame-thread export after %d ticks", ticks)
		}
	}
}

func TestAutosaveWorkerBytesEqualFrameThreadBytesInTown(t *testing.T) {
	f := currentTrainingCity(t)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := f.playerMissionSave(s, "timed autosave 2")
	if err != nil {
		t.Fatal(err)
	}
	if got := asyncExport(t, f, s, "timed autosave 2"); !bytes.Equal(got, want) {
		t.Fatal("town worker bytes differ from the frame-thread export")
	}
}

// The export reads the captured value and the detached front end only. Running
// the mission, spending the purse and replacing the live driver while the
// worker holds the export leaves its bytes those of the capture.
func TestAutosaveExportIgnoresLaterFrameThreadChanges(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("async isolation")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		f.live.tick()
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := f.playerMissionSave(s, "timed autosave 1")
	if err != nil {
		t.Fatal(err)
	}
	view, detached := f.detachedExporter(s)
	q := newAutosaveQueue()
	started, release := make(chan struct{}), make(chan struct{})
	var got []byte
	q.submit(false, false, "", func() autosaveResult {
		close(started)
		<-release
		var err error
		got, _, err = view.playerMissionSave(detached, "timed autosave 1")
		return autosaveResult{err: err}
	})
	<-started
	for range 300 {
		f.live.tick()
	}
	if f.Town != nil {
		f.Town.gold += 5000
	}
	f.Carried = nil
	f.live = nil
	close(release)
	q.wait()
	if !bytes.Equal(got, want) {
		t.Fatal("later frame-thread changes reached the exported bytes")
	}
}

// The export runs on a worker while the frame thread keeps stepping the
// mission and taking new captures. Every worker output equals the export the
// frame thread made of the same capture before the worker started, and the Go
// runtime's concurrent map access check stays silent.
func TestAutosaveExportWhileTheFrameThreadKeepsRunning(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	app := f.App("async stress")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	q := newAutosaveQueue()
	for round := range 12 {
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		want, _, err := f.playerMissionSave(s, "timed autosave 1")
		if err != nil {
			t.Fatal(err)
		}
		view, detached := f.detachedExporter(s)
		var got []byte
		q.submit(false, false, "", func() autosaveResult {
			var err error
			got, _, err = view.playerMissionSave(detached, "timed autosave 1")
			return autosaveResult{err: err}
		})
		for range 40 + round {
			f.live.tick()
			if _, _, err := f.Snapshot(true); err != nil {
				t.Fatal(err)
			}
		}
		q.wait()
		for _, r := range q.take(false) {
			if r.err != nil {
				t.Fatal(r.err)
			}
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("round %d: worker bytes differ from the capture's frame-thread export", round)
		}
	}
}

func TestAutosaveQueueRunsJobsInOrderAndRecoversPanics(t *testing.T) {
	q := newAutosaveQueue()
	var mu sync.Mutex
	var order []int
	for i := range 5 {
		q.submit(false, false, "", func() autosaveResult {
			time.Sleep(time.Millisecond)
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
			return autosaveResult{}
		})
	}
	q.submit(false, false, "", func() autosaveResult { panic("boom") })
	q.wait()
	for i, v := range order {
		if v != i {
			t.Fatal("jobs ran out of order", order)
		}
	}
	results := q.take(false)
	if len(results) != 6 || results[5].err == nil {
		t.Fatalf("results %+v", results)
	}
	// Inline jobs complete before submit returns.
	done := false
	q.submit(true, false, "", func() autosaveResult { done = true; return autosaveResult{} })
	if !done {
		t.Fatal("inline job did not run before submit returned")
	}
}

func missionStartOrderFixture(t *testing.T) (*FrontEnd, *ui.App, SaveStore, *gatedSaveFiles) {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	f.SetDeterministicFrames(false)
	gate := newGatedSaveFiles()
	store := SaveStore{Dir: t.TempDir(), files: gate}
	app := f.App("autosave order")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	return f, app, store, gate
}

func blocked(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
		t.Fatal("operation finished while an automatic save was in flight")
	case <-time.After(150 * time.Millisecond):
	}
}

// Quick save, quick load, the SAVE seam and exit wait for a save in flight, so
// none of them can finish before it or be overtaken by it.
func TestAutosaveInFlightOrdersLaterSavesLoadAndExit(t *testing.T) {
	for _, step := range []struct {
		name string
		run  func(app *ui.App)
	}{
		{"quick save", func(app *ui.App) { _ = app.HeadlessKey("f4") }},
		{"quick load", func(app *ui.App) { _ = app.HeadlessKey("f9") }},
		{"exit", func(app *ui.App) { app.FlushBackground() }},
	} {
		t.Run(step.name, func(t *testing.T) {
			f, app, store, gate := missionStartOrderFixture(t)
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			<-gate.entered
			done := make(chan struct{})
			go func() { step.run(app); close(done) }()
			blocked(t, done)
			if _, err := os.Stat(filepath.Join(store.Dir, "quick-save-1.sav")); !os.IsNotExist(err) {
				t.Fatal("later save was written before the automatic save finished", err)
			}
			close(gate.gate)
			<-done
			rows, err := store.List()
			if err != nil || len(rows) == 0 || rows[len(rows)-1].Name != "game0000.sav" && rows[0].Name != "game0000.sav" {
				t.Fatalf("automatic save missing after %s: %v %v", step.name, rows, err)
			}
			raw, err := store.Read("game0000.sav")
			if err != nil || len(raw) == 0 {
				t.Fatal("automatic save unreadable", err)
			}
		})
	}
}

// An automatic save that finishes late never replaces a file the player wrote
// after it began: it only ever claims a new name.
func TestAutosaveFinishingLateKeepsEarlierPlayerFiles(t *testing.T) {
	f, app, store, gate := missionStartOrderFixture(t)
	manual := filepath.Join(store.Dir, "game0000.sav")
	if err := os.WriteFile(manual, []byte("player file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	<-gate.entered
	close(gate.gate)
	app.FlushBackground()
	if got, err := os.ReadFile(manual); err != nil || string(got) != "player file" {
		t.Fatal("automatic save replaced a player file", err)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "game0001.sav")); err != nil {
		t.Fatal("automatic save did not claim the next free name", err)
	}
}

// Mission end while the write is in flight changes nothing the worker reads:
// the file holds the captured mission start.
func TestAutosaveInFlightSurvivesLeavingTheMission(t *testing.T) {
	f, app, store, gate := missionStartOrderFixture(t)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	<-gate.entered
	for range 120 {
		f.live.tick()
	}
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	close(gate.gate)
	app.FlushBackground()
	if _, err := store.Read("game0000.sav"); err != nil {
		t.Fatal("first mission's automatic save lost", err)
	}
	if _, err := store.Read("game0001.sav"); err != nil {
		t.Fatal("second mission's automatic save lost", err)
	}
}

type panickingSaveFiles struct{ osSaveFileOps }

// CreateTemp runs with saveTempMu held; the test releases it after the panic.
func (panickingSaveFiles) CreateTemp(dir, pattern string) (saveTempFile, error) {
	panic("injected write panic")
}

func messageText(app *ui.App, f *FrontEnd) string {
	var out string
	for _, line := range f.live.view.MessageLines() {
		out += line.Text
	}
	return out
}

// A job that panics reaches the player through the ordinary failure path.
func TestAutosaveMissionStartPanicIsShownToThePlayer(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	f.SetDeterministicFrames(false)
	store := SaveStore{Dir: t.TempDir(), files: panickingSaveFiles{}}
	app := f.App("mission start panic")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	saveTempMu.Unlock()
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if got := messageText(app, f); !strings.Contains(got, "cannot autosave mission start: autosave stopped: injected write panic") {
		t.Fatalf("panic not shown: %q", got)
	}
}

func TestAutosaveTimedPanicIsShownToThePlayer(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	f.SetDeterministicFrames(false)
	store := SaveStore{Dir: t.TempDir()}
	now := time.Unix(100, 0)
	app := f.App("timed panic")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	// Dropping the cell-plane base from the live driver's map makes the
	// worker's export dereference nil.
	f.Table.Units = nil
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if got := messageText(app, f); !strings.Contains(got, "Timed autosave failed: autosave stopped:") {
		t.Fatalf("panic not shown: %q", got)
	}
}
