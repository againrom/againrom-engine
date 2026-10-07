package game

import (
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// The demo install is gameversions/rom1-demo beside the checkouts. The tests
// read it and never write it. demoRootOverride names a different one.
const (
	demoRootOverride = "AGAINROM_DEMO_ASSETS"
	demoShotsDir     = "AGAINROM_DEMO_SHOTS"
)

// demoRoot is the demo install, or the test is skipped when it is absent. It is
// not an environment-gated test: the root is found without any variable, and
// the skip message names no variable.
func demoRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv(demoRootOverride)
	if root == "" {
		root = seatPath("gameversions/rom1-demo")
	}
	if st, err := os.Stat(filepath.Join(root, "main.res")); err != nil || st.IsDir() {
		t.Skipf("no demo base at %s", root)
	}
	return root
}

func demoFront(t *testing.T) *FrontEnd {
	t.Helper()
	f, err := NewFrontEnd(demoRoot(t))
	if err != nil {
		t.Fatalf("NewFrontEnd over the demo base: %v", err)
	}
	cleanupFrontAudio(t, f)
	return f
}

func TestDemoBaseIsDetected(t *testing.T) {
	root := demoRoot(t)
	info := InspectInstall(root)
	if !info.Valid() {
		t.Fatalf("the demo is not a valid install: %+v", info)
	}
	if info.Base.ID() != base.ROM1Demo || !info.Base.Exact {
		t.Fatalf("detected %s, want an exact %s", info.Base, base.ROM1Demo)
	}
	if BaseID(info) != base.ROM1Demo {
		t.Fatalf("BaseID = %q", BaseID(info))
	}
	p := info.Base.Profile
	if !p.Limits.NoCharacterGeneration || p.Mission() != 41 || p.Limits.OriginalSaveRefusal == "" {
		t.Fatalf("the demo profile states %+v", p.Limits)
	}
	front := demoFront(t)
	if front.Base().ID() != base.ROM1Demo || front.ChargenAssets != nil {
		t.Fatalf("front end base %s, generation art %v", front.Base(), front.ChargenAssets != nil)
	}
	lines := strings.Join(front.BaseLines(), "\n")
	for _, want := range []string{"base rom1-demo (Rage of Mages demo 1.01; exact build)", "mission 41", "inn arrays"} {
		if !strings.Contains(lines, want) {
			t.Errorf("base lines lack %q:\n%s", want, lines)
		}
	}
}

// The demo's main menu is composed offscreen from its own archive. The frame is
// written as a PNG under t.TempDir(), or under the directory the shots variable
// names.
func TestDemoBaseMainMenuFrame(t *testing.T) {
	front := demoFront(t)
	app := front.App("demo menu")
	app.Layout(640, 480)
	if app.Screen() != ui.ScreenMenu {
		t.Fatalf("screen = %v, want the menu", app.Screen())
	}
	pix, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("frame: note %q err %v", note, err)
	}
	if pix.Bounds().Dx() != 640 || pix.Bounds().Dy() != 480 {
		t.Fatalf("frame is %v, want 640x480", pix.Bounds())
	}
	colours := map[[4]uint8]bool{}
	for y := 0; y < 480; y += 4 {
		for x := 0; x < 640; x += 4 {
			c := pix.RGBAAt(x, y)
			colours[[4]uint8{c.R, c.G, c.B, c.A}] = true
		}
	}
	if len(colours) < 64 {
		t.Fatalf("the menu frame holds %d distinct colours in a sample grid, want artwork", len(colours))
	}
	dir := os.Getenv(demoShotsDir)
	if dir == "" {
		dir = t.TempDir()
	}
	writeFramePNG(t, filepath.Join(dir, "demo-main-menu.png"), pix)
}

func writeFramePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestDemoBaseNewGameRunsTheFirstMission(t *testing.T) {
	front := demoFront(t)
	app := front.App("demo new game")
	app.Layout(640, 480)
	mission, members := 41, 2
	scenario := HeadlessScenario{Version: HeadlessScenarioVersion, Steps: []HeadlessStep{
		{Command: "assert_state", State: &HeadlessStateAssertion{Screen: "menu"}},
		{Command: "activate", Target: "NEW GAME"},
		{Command: "assert_state", State: &HeadlessStateAssertion{Screen: "map", Mission: &mission}},
		{Command: "wait_ticks", Ticks: 200},
		{Command: "assert_state", State: &HeadlessStateAssertion{Screen: "map", Mission: &mission, MemberCount: &members}},
	}}
	if err := RunHeadlessScenario(front, app, scenario, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
}

// The demo's own save is refused by the engine's original-save reader, and the
// refusal names the base and the limit. It is read through a copy in a
// temporary store.
func TestDemoBaseOriginalSaveIsRefusedWithTheBaseLimit(t *testing.T) {
	root := demoRoot(t)
	saved, err := os.ReadFile(filepath.Join(root, "game9999.sav"))
	if err != nil {
		t.Skipf("the demo base ships no save: %v", err)
	}
	store := t.TempDir()
	if err := os.WriteFile(filepath.Join(store, "game9999.sav"), saved, 0o644); err != nil {
		t.Fatal(err)
	}
	front := demoFront(t)
	app := front.App("demo load")
	app.Layout(640, 480)
	front.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: store}, nil)
	scenario := HeadlessScenario{Version: HeadlessScenarioVersion, Steps: []HeadlessStep{
		{Command: "load", Target: "@first"},
	}}
	err = RunHeadlessScenario(front, app, scenario, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("loading the demo's own save succeeded; the profile's stated limit is stale")
	}
	for _, want := range []string{"inn arrays are 3 NPCs and 2 missions", "base rom1-demo"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q lacks %q", err, want)
		}
	}
	if app.Screen() == ui.ScreenMap {
		t.Fatal("a refused load entered a game")
	}
}
