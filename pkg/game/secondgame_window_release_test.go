package game

import (
	"image"
	"os"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/render/menu"
	"againrom/pkg/ui"
)

func secondGameFront(t *testing.T) *FrontEnd {
	t.Helper()
	secondGameRoot(t)
	f, err := NewFrontEnd(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, f)
	f.AmbientSeed = 1
	f.SetDeterministicFrames(true)
	return f
}

var secondOverlaySizes = [menu.ButtonCount]image.Point{
	{104, 96}, {108, 76}, {96, 88}, {100, 100}, {88, 100}, {84, 88}, {96, 84}, {72, 80},
}

func TestReleaseSecondGameMenuDrawsInstalledArt(t *testing.T) {
	f := secondGameFront(t)
	a := f.Assets
	if a.Base.Bounds().Size() != image.Pt(640, 480) || a.Mask.Bounds().Size() != image.Pt(640, 480) {
		t.Fatalf("base %v mask %v, want 640x480", a.Base.Bounds(), a.Mask.Bounds())
	}
	for i, want := range secondOverlaySizes {
		if a.Hover[i].Bounds().Size() != want || a.Pressed[i].Bounds().Size() != want {
			t.Errorf("button %d overlays are %v and %v, want %v", i+1, a.Hover[i].Bounds().Size(), a.Pressed[i].Bounds().Size(), want)
		}
	}
	for i, n := range a.MaskRegions() {
		if n == 0 {
			t.Errorf("button %d has no mask region", i+1)
		}
	}
	baseFrame := a.Compose(menu.State{})
	for b := 1; b <= menu.ButtonCount; b++ {
		for _, pressed := range []bool{false, true} {
			img, at, ok := a.Overlay(menu.State{Selected: b, Pressed: pressed})
			if !ok || at.Size() != secondOverlaySizes[b-1] || img.Bounds().Size() != at.Size() {
				t.Fatalf("button %d overlay: ok %v at %v", b, ok, at)
			}
			hot := image.Rectangle{}
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					if a.ButtonAt(image.Pt(x, y)) == b {
						hot = hot.Union(image.Rect(x, y, x+1, y+1))
					}
				}
			}
			if !hot.In(at.Inset(-8)) {
				t.Errorf("button %d hot region %v lies outside its overlay %v", b, hot, at)
			}
			frame := a.Compose(menu.State{Selected: b, Pressed: pressed})
			caption := image.Rect(232, 200, 412, 280)
			for y := 0; y < 480; y++ {
				for x := 0; x < 640; x++ {
					p := image.Pt(x, y)
					if p.In(at) || p.In(caption) {
						continue
					}
					if frame.RGBAAt(x, y) != baseFrame.RGBAAt(x, y) {
						t.Fatalf("button %d: pixel %v outside overlay and caption differs", b, p)
					}
				}
			}
		}
	}
	app := f.App("second")
	app.Layout(1024, 768)
	pix, _, err := app.HeadlessFrame()
	if err != nil || pix == nil {
		t.Fatalf("menu frame: %v", err)
	}
}

func TestReleaseSecondGameWindowStartsMissionTen(t *testing.T) {
	f := secondGameFront(t)
	if f.Archives.Game() != base.GameROM2 {
		t.Fatal("not a second-game root")
	}
	f.Cutscenes = OpenCutscenes(os.Getenv("AGAINROM_ASSETS"), "video4")
	app := f.App("second")
	app.Layout(1024, 768)
	if app.PlayStartupCutscenes() {
		t.Error("a startup clip played on a second-game root")
	}
	enterSecondCampaignMission(t, app)
	if app.Screen() != ui.ScreenMap || f.live == nil {
		t.Fatalf("screen %v after New Game: %q", app.Screen(), app.HeadlessMessage())
	}
	if f.live.mission.number != 10 {
		t.Fatalf("mission %d, want 10", f.live.mission.number)
	}
	w := f.live.world
	if len(w.Entities()) != 30 {
		t.Errorf("%d entities, want 30", len(w.Entities()))
	}
	for i := 0; i < 20; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if w.Tick() == 0 {
		t.Fatal("world did not advance")
	}
	for pages := 0; f.live.mission.open && f.live.mission.kind == ui.NoticeDialogue; pages++ {
		if pages >= 20 {
			t.Fatal("initial dialogue did not finish")
		}
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.mission.open {
		t.Fatal("unexpected notice before the movement check")
	}
	id := f.live.mission.ids[0]
	e, _ := w.Entity(id)
	cam := f.live.view.Camera()
	cam.CenterOn(float64(e.X*32), float64(e.Y*32))
	x0 := cam.X
	cam.Pan(160, 0)
	if cam.X == x0 {
		t.Error("camera did not pan")
	}
	cam.CenterOn(float64(e.X*32), float64(e.Y*32))
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if sel := app.HeadlessSelection(); len(sel) != 1 || sel[0] != uint32(id) {
		t.Fatalf("selection %v, want [%d]", sel, id)
	}
	gx, gy, err := app.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := w.Entity(id)
	if before.X != e.X || before.Y != e.Y {
		t.Fatalf("selected unit moved without an order: (%d,%d) to (%d,%d)", e.X, e.Y, before.X, before.Y)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, gx, gy); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 120; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := w.Entity(id)
	if after.X == before.X && after.Y == before.Y {
		t.Errorf("selected unit stayed at (%d,%d) after a move click", after.X, after.Y)
	}
}

func TestReleaseSecondGameMenuButtonsDoNotCrash(t *testing.T) {
	f := secondGameFront(t)
	for _, name := range []string{"load game", "hall of fame", "cutscenes", "credits"} {
		app := f.App("second")
		app.Layout(1024, 768)
		if err := app.HeadlessActivate(name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s opens screen %v", name, app.Screen())
		if err := app.HeadlessStep(); err != nil {
			t.Fatalf("%s: step: %v", name, err)
		}
		if _, _, err := app.HeadlessFrame(); err != nil {
			t.Fatalf("%s: frame: %v", name, err)
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatalf("%s: escape: %v", name, err)
		}
		if app.Screen() != ui.ScreenMenu {
			t.Errorf("%s: Escape leaves screen %v", name, app.Screen())
		}
	}
}

func TestReleaseSecondGameWindowArtLoads(t *testing.T) {
	f := secondGameFront(t)
	for name, err := range map[string]error{
		"font": f.Font.Err(), "attack pointer": f.AttackPointer.Err(), "cursors": f.CursorRegistry.Err(),
		"bottom HUD": f.BottomHUDArt.Err(), "command panel": f.CommandPanelArt.Err(), "start weapon": f.StartWeapon.Err(),
	} {
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if f.Campaign.Err() == nil {
		t.Error("the second game's root yielded a campaign registry")
	}
}

func TestReleaseSecondGameMissionEntryAutosaveColdLoads(t *testing.T) {
	f := secondGameFront(t)
	store := SaveStore{Dir: t.TempDir()}
	now := time.Unix(100, 0)
	app := f.App("second")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	enterSecondCampaignMission(t, app)
	if app.Screen() != ui.ScreenMap || f.live.world.Tick() != 0 {
		t.Fatalf("mission entry screen %v tick %d", app.Screen(), f.live.world.Tick())
	}
	row := missionAutosaveRow(t, f, store, 10)
	entries, err := os.ReadDir(store.Dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != row.Name {
		t.Fatalf("mission entry save files %v: %v", entries, err)
	}
	want := secondSaveSampleNow(t, f, app)
	cold, loaded := secondMissionCold(t, store.Dir, row.Name)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "mission entry")
	secondAssertSample(t, want, secondSaveSampleNow(t, cold, loaded))
	for range 16 {
		f.live.tick()
		cold.live.tick()
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "post-load tick")
	}
	if f.live.world.Tick() != 16 || cold.live.world.Tick() != 16 {
		t.Fatal("post-load worlds did not advance sixteen ticks")
	}
	secondAssertSample(t, secondSaveSampleNow(t, f, app), secondSaveSampleNow(t, cold, loaded))
}
