package game

import (
	"math"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// The camera is additive snapshot state. A payload written before these fields
// existed decodes with CameraSet false, which is what keeps an old save opening
// at the map's start position rather than at cell (0, 0) -- a real map corner.
func TestSnapshotCameraSurvivesTheNativeEncoding(t *testing.T) {
	town := NewTown(saveCampaign())
	var base Snapshot
	snapshotTown(town, &base)

	legacy, err := EncodeSave(base, "no camera")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if back.CameraSet || back.CameraX != 0 || back.CameraY != 0 || back.CameraZoom != 0 {
		t.Fatalf("a save with no camera decoded one: %+v", back)
	}

	withCamera := base
	withCamera.CameraSet, withCamera.CameraX, withCamera.CameraY, withCamera.CameraZoom = true, -12.5, 27, 2
	raw, err := EncodeSave(withCamera, "camera")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err = DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !back.CameraSet || back.CameraX != -12.5 || back.CameraY != 27 || back.CameraZoom != 2 {
		t.Fatalf("camera round trip = %v (%v, %v, %v)", back.CameraSet, back.CameraX, back.CameraY, back.CameraZoom)
	}
	if math.Abs(back.CameraX) > math.MaxInt32 {
		t.Fatal("decoded camera is outside the restorable range")
	}
}

func TestReleaseMissionSaveReopensAtTheSavedCamera(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Saved camera", Choices: []int{0, 0, 0}, Stats: []int{30, 30, 30, 30}})
	app := f.App("saved camera witness")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	start := f.live.view.SaveApplication()
	moved := start
	moved.ViewX, moved.ViewY = math.Round(start.ViewX)+11, math.Round(start.ViewY)+9
	if err := f.live.view.RestoreSaveApplication(moved); err != nil {
		t.Fatal(err)
	}
	want := f.live.view.SaveApplication()
	if want.ViewX == start.ViewX || want.ViewY == start.ViewY {
		t.Fatalf("the witness did not move the camera off %+v", start)
	}
	store, name, raw := menuSAVE(t, f, app, OriginalStore{})
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := readOriginalApplicationState(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if float64(saved.ViewX) != want.ViewX || float64(saved.ViewY) != want.ViewY {
		t.Fatalf("SAV camera (%d, %d), live (%v, %v)", saved.ViewX, saved.ViewY, want.ViewX, want.ViewY)
	}
	fresh := releaseFront(t)
	fresh.SetDeterministicFrames(true)
	reopened := fresh.App("reopened camera witness")
	reopened.Layout(1024, 768)
	save, list, load := fresh.SaveSeams(store, OriginalStore{}, nil)
	reopened.SetSaveSeams(save, list, load)
	groundAppLoad(t, reopened, list, localOriginalSaveToken(name))
	got := fresh.live.view.SaveApplication()
	if got.ViewX != want.ViewX || got.ViewY != want.ViewY || got.Zoom != want.Zoom {
		t.Fatalf("reopened at (%v, %v, %v), want (%v, %v, %v)", got.ViewX, got.ViewY, got.Zoom, want.ViewX, want.ViewY, want.Zoom)
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("live game did not return to the map", app.Screen(), err)
	}
	for _, a := range []*ui.App{app, reopened} {
		for _, event := range []struct {
			action string
			x, y   int
		}{{"right-press", 400, 300}, {"right-move", 464, 332}, {"right-release", 464, 332}} {
			if err := a.HeadlessPointer(event.action, event.x, event.y); err != nil {
				t.Fatal(err)
			}
		}
	}
	for tick := range 16 {
		live, cold := f.live.view.SaveApplication(), fresh.live.view.SaveApplication()
		if live.ViewX != cold.ViewX || live.ViewY != cold.ViewY || live.Zoom != cold.Zoom {
			t.Fatalf("tick %d: live camera (%v, %v), reopened (%v, %v)", tick, live.ViewX, live.ViewY, cold.ViewX, cold.ViewY)
		}
		if tick == 0 && live.ViewX == want.ViewX && live.ViewY == want.ViewY {
			t.Fatal("the pan did not move the camera")
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := reopened.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	lossy := loadAlteredSAV(t, raw, func(doc *sav.DocumentData) bool {
		for i := range doc.State.ValueRecords {
			if doc.State.ValueRecords[i].Path == "/View/X" {
				doc.State.ValueRecords[i].Value.Int32 += 5
				return true
			}
		}
		return false
	})
	if lost := lossy.live.view.SaveApplication(); lost.ViewX != want.ViewX+5 {
		t.Fatalf("altered SAV camera reopened at %v, want %v", lost.ViewX, want.ViewX+5)
	}
	t.Logf("camera %v,%v -> %v,%v; SAV (%d, %d); reopened through the load window; 16-step pan equal; altered camera loads", start.ViewX, start.ViewY, want.ViewX, want.ViewY, saved.ViewX, saved.ViewY)
}
