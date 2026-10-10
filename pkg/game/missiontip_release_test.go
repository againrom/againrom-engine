package game

import (
	"image"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// missionTipFront is a release front end whose TipsMode word writes to a
// temporary options file and starts set (MENU-135's default).
func missionTipFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := shopOrderFront(t)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.SetTipsOff(false)
	return f
}

// openMissionTipMap opens mission n on a 1024x768 App, whose window pixels
// are frame pixels, and waits for the dialogue its start trigger raises.
func openMissionTipMap(t *testing.T, f *FrontEnd, n int) (*ui.App, *mapWorld) {
	t.Helper()
	app := f.App("mission tips")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(n, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for i := 0; !live.view.NoticeOpen() && i < 64; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	return app, live
}

// pageMissionDialogue presses Enter through an open dialogue and checks that
// no page before the last one changes the mission popup.
func pageMissionDialogue(t *testing.T, app *ui.App, live *mapWorld, what string) {
	t.Helper()
	if !live.view.NoticeOpen() {
		t.Fatalf("%s: no dialogue open", what)
	}
	before := live.view.MissionTip().Text
	for n := 0; live.view.NoticeOpen() && n < 32; n++ {
		if live.mission.part > 0 && live.view.MissionTip().Text != before {
			t.Fatalf("%s: the popup changed on page %d before the dialogue closed", what, live.mission.part)
		}
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if live.view.NoticeOpen() {
		t.Fatalf("%s: the dialogue did not close", what)
	}
}

func missionTipText(t *testing.T, f *FrontEnd, mission, n int) string {
	t.Helper()
	path, ok := MissionTipPath(f.Base().Profile.Edition(), mission, n)
	if !ok {
		t.Fatalf("the edition has no tip %d of mission %d", n, mission)
	}
	text, ok := ReadShopTip(f.Archives.Containers, path, f.textCode())
	if !ok || text == "" {
		t.Fatalf("installed tip %d of mission %d is missing", n, mission)
	}
	return text
}

func wantMissionTip(t *testing.T, f *FrontEnd, live *mapWorld, mission, n int, what string) {
	t.Helper()
	v := live.view.MissionTip()
	if !v.Showing() || v.Rect != ui.MissionTipRect || v.Text != missionTipText(t, f, mission, n) {
		t.Fatalf("%s: popup %q at %v showing %v, want tip %d at %v", what, v.Text, v.Rect, v.Showing(), n, ui.MissionTipRect)
	}
}

// TestReleaseMissionTipsFollowTheirTaggedDialogues: every shipped `tips=`
// block in m10 and m20 raises its tip in the mission popup at (10,20)-(370,188)
// when its dialogue closes, replacing the open one; the start triggers raise
// the first through the script (TRIG-TIPS-087, TRIG-TIPS-088).
func TestReleaseMissionTipsFollowTheirTaggedDialogues(t *testing.T) {
	for _, mission := range []struct {
		n, startTip int
		events      [][2]int // event, tip
	}{
		{10, 2, [][2]int{{16, 3}, {4, 1}, {1, 5}, {2, 7}, {14, 4}, {18, 6}}},
		{20, 1, [][2]int{{7, 2}}},
	} {
		f := missionTipFront(t)
		app, live := openMissionTipMap(t, f, mission.n)
		pageMissionDialogue(t, app, live, "start dialogue")
		wantMissionTip(t, f, live, mission.n, mission.startTip, "start trigger")
		for _, e := range mission.events {
			if !live.openDialogue(e[0]) {
				t.Fatalf("m%d event %02d does not open", mission.n, e[0])
			}
			pageMissionDialogue(t, app, live, "tagged dialogue")
			wantMissionTip(t, f, live, mission.n, e[1], "tagged dialogue")
		}
		t.Logf("m%d: start tip %d and %d tagged dialogues raised their tips", mission.n, mission.startTip, len(mission.events))
	}
}

// TestReleaseMissionTipControlsAndTipsMode: Close deletes the popup on its own
// press and release; the checkbox clears TipsMode at once without deleting the
// open popup; with TipsMode clear a tagged dialogue raises nothing (MENU-135,
// MENU-136, MENU-137).
func TestReleaseMissionTipControlsAndTipsMode(t *testing.T) {
	f := missionTipFront(t)
	app, live := openMissionTipMap(t, f, 10)
	pageMissionDialogue(t, app, live, "start dialogue")
	wantMissionTip(t, f, live, 10, 2, "start trigger")
	box := ui.TipPanelToggleRect(ui.MissionTipRect)
	at := box.Min.Add(image.Pt(2, 2))
	if err := app.HeadlessPointer("press", at.X, at.Y); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("release", at.X, at.Y); err != nil {
		t.Fatal(err)
	}
	if !f.TipsOff() || !live.view.MissionTip().Showing() || live.view.MissionTip().ToggleOn {
		t.Fatal("the checkbox press did not clear TipsMode at once, or deleted the open popup")
	}
	if !live.openDialogue(4) {
		t.Fatal("m10 event 04 does not open")
	}
	pageMissionDialogue(t, app, live, "dialogue with TipsMode clear")
	wantMissionTip(t, f, live, 10, 2, "TipsMode clear keeps the open popup")
	closeAt := ui.TipPanelCloseRect(ui.MissionTipRect).Min.Add(image.Pt(4, 4))
	if err := app.HeadlessPointer("press", closeAt.X, closeAt.Y); err != nil {
		t.Fatal(err)
	}
	if !live.view.MissionTip().Showing() {
		t.Fatal("Close acted on its press")
	}
	if err := app.HeadlessPointer("release", closeAt.X, closeAt.Y); err != nil {
		t.Fatal(err)
	}
	if live.view.MissionTip().Showing() {
		t.Fatal("Close did not delete the popup on its release")
	}
	f.SetTipsOff(false)
	if !live.openDialogue(4) {
		t.Fatal("m10 event 04 does not open")
	}
	pageMissionDialogue(t, app, live, "dialogue with TipsMode set")
	wantMissionTip(t, f, live, 10, 1, "TipsMode set again")
}

// TestReleaseMissionStartTipDoesNotReturnAfterSaveAndLoad: the start trigger's
// fire-once latch rides the SAV, so after F2 SAVE and LOAD the start dialogue
// and its tip do not come back (TRIG-TIPS-088).
func TestReleaseMissionStartTipDoesNotReturnAfterSaveAndLoad(t *testing.T) {
	f := missionTipFront(t)
	app, live := openMissionTipMap(t, f, 10)
	pageMissionDialogue(t, app, live, "start dialogue")
	wantMissionTip(t, f, live, 10, 2, "start trigger")
	for i := 0; i < 64; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if live.view.NoticeOpen() && live.mission.payload != nil {
		t.Fatal("the start dialogue opened a second time in the same run")
	}
	_, raw := ogreF2Save(t, f, app)
	g := missionTipFront(t)
	loaded, _ := openOriginalSAVAppAt(t, g, raw, "tips-current.sav", 1024, 768)
	t.Cleanup(loaded.StopAudio)
	if g.live == nil || g.live.mission.number != 10 {
		t.Fatal("the current SAV did not load mission 10")
	}
	for i := 0; i < 64; i++ {
		if g.live.mission.payload != nil || g.live.view.MissionTip().Showing() {
			t.Fatalf("after LOAD step %d the start dialogue or its tip came back", i)
		}
		if err := loaded.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
}
