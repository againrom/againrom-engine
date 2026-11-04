package game

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/render/menu"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// TestReleaseRouteWitness walks the install's game from launch to the first
// mission's first ticks through the App's own input, asserts every step
// reaches its screen, and records each step. The game picks the stage table:
// the second game's route, or the first game's.
func TestReleaseRouteWitness(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the route witness needs a lawful install")
	}
	if os.Getenv("AGAINROM_HEADLESS_ONLY") != "1" && os.Getenv("AGAINROM_ROUTE_RENDER_CHILD") != "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReleaseRouteWitness$", "-test.v", "-test.timeout=40m")
		cmd.Env = append(os.Environ(), "AGAINROM_ROUTE_RENDER_CHILD=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("route GPU child: %v\n%s", err, out)
		}
		t.Logf("%s", out)
		return
	}
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	w, stages := newRouteWitness(t, archives.Game(), "")
	f := w.f
	start := time.Now()
	w.walk(stages, "")
	counts := make([]string, 0, len(stages))
	for _, s := range stages {
		counts = append(counts, fmt.Sprintf("%s=%d", s.name, w.steps[s.name]))
	}
	t.Logf("%s: %d steps, %d frames in %v; %s", f.Base().Profile.ID, len(w.rows), w.frames, time.Since(start).Round(time.Millisecond), strings.Join(counts, " "))
	if town := routeTownRows(stages); len(town) > 0 {
		w.townSaves(t, town)
	}
}

// townSaves loads each .sav of AGAINROM_ROUTE_SAVES, in name order, cold
// through the menu's Load window and walks the town rows from there, with
// the file name leading each row and its frames in saves/<file>/. The walk
// buys, hires and saves nothing. Each save's walk opens its own witness on t.
func (w *routeWitness) townSaves(t *testing.T, town []routeStage) {
	dir := os.Getenv("AGAINROM_ROUTE_SAVES")
	if dir == "" {
		w.rows = append(w.rows, strings.Join([]string{fmt.Sprint(len(w.rows) + 1), "town-saves", w.screen(), "", "", "wait", "none: AGAINROM_ROUTE_SAVES names no town save set", "", "", ""}, "\t"))
		w.writeRoute()
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.EqualFold(filepath.Ext(name), ".sav") {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			w.t.Fatal(err)
		}
		originals := filepath.Join(w.t.TempDir(), "originals")
		if err := os.MkdirAll(originals, 0o700); err != nil {
			w.t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(originals, name), payload, 0o600); err != nil {
			w.t.Fatal(err)
		}
		s, _ := newRouteWitness(t, w.f.Archives.Game(), originals)
		if s.dir != "" {
			s.dir = filepath.Join(s.dir, "saves", name)
			if err := os.MkdirAll(s.dir, 0o700); err != nil {
				w.t.Fatal(err)
			}
		}
		s.save, s.stage = name, "load"
		s.wait("menu")
		r, ok := routeFind(s.menuRegions(), "")
		for _, m := range s.menuRegions() {
			b := 0
			fmt.Sscanf(m.key, "%d", &b)
			if s.f.Assets.Binding(b).Action == menu.ActionLoad {
				r, ok = m, true
			}
		}
		if !ok {
			w.t.Fatal("town saves: the menu has no Load button")
		}
		s.click(r.key, r.rect, r.at, 2, nil)
		s.expect("load", "Load")
		before := s.screen()
		if err := s.app.HeadlessActivate("@first"); err != nil {
			w.t.Errorf("town saves: %s: %v", name, err)
		}
		s.until(400, func() bool { return s.app.Screen() == ui.ScreenTown })
		s.record("load "+name, image.Rectangle{}, "activate", s.result(before, ""))
		if s.app.Screen() != ui.ScreenTown {
			w.t.Errorf("town saves: %s loads to %s: %q", name, s.screen(), s.app.HeadlessMessage())
			s.writeRoute()
			continue
		}
		s.dialoguePages("town entry", nil)
		s.walk(town, name)
		w.t.Logf("%s: %d steps", name, len(s.rows))
	}
}

// routeFronts opens each game's front end as its release tests run it.
var routeFronts = map[base.Game]func(*testing.T) *FrontEnd{
	base.GameROM1: func(t *testing.T) *FrontEnd {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		return f
	},
	base.GameROM2: secondGameFront,
}

// newRouteWitness opens game's front end and its App on the install as the
// game runs, with originals, when set, as the original saves' directory,
// and answers the witness and its game's stage table. Outside
// AGAINROM_HEADLESS_ONLY a mission frame is the App's own Draw on the GPU.
func newRouteWitness(t *testing.T, game base.Game, originals string) (*routeWitness, []routeStage) {
	f := routeFronts[game](t)
	size, err := routeWindowSize(os.Getenv("AGAINROM_ROUTE_WINDOW"))
	if err != nil {
		t.Fatal(err)
	}
	w, stages := newRouteWitnessOn(t, f, originals, size, os.Getenv("AGAINROM_ROUTE_WITNESS_DIR"))
	w.emit = func(dir, name string, pix *image.RGBA) { writeTipRender(t, dir, name, pix) }
	if os.Getenv("AGAINROM_HEADLESS_ONLY") != "1" {
		w.render = func(app *ui.App, size image.Point) *image.RGBA {
			tex := ebiten.NewImage(size.X, size.Y)
			defer tex.Dispose()
			app.Draw(tex)
			pix := image.NewRGBA(tex.Bounds())
			tex.ReadPixels(pix.Pix)
			return pix
		}
	}
	return w, stages
}

func TestRouteWitnessWindowGeometry(t *testing.T) {
	for _, value := range []string{"", "640x480", "1024x768", "1280x720"} {
		t.Run(value, func(t *testing.T) {
			size, err := routeWindowSize(value)
			if err != nil {
				t.Fatal(err)
			}
			if value == "" && size != image.Pt(ui.DefaultWindowW, ui.DefaultWindowH) {
				t.Fatalf("default size %v", size)
			}
			app := ui.NewApp("route geometry", nil, nil, nil)
			app.Layout(size.X, size.Y)
			w := &routeWitness{t: t, app: app, size: size}
			for _, p := range []image.Point{{0, 0}, {321, 237}, {639, 479}} {
				window := w.windowPoint(p)
				if got := w.nativePoint(window.X, window.Y); got != p {
					t.Errorf("native %v -> window %v -> native %v", p, window, got)
				}
			}
			src := image.NewRGBA(image.Rect(0, 0, 640, 480))
			src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
			pic := w.fitFrame(src)
			if pic.Bounds().Size() != size {
				t.Fatalf("capture %v, window %v", pic.Bounds(), size)
			}
			p := w.windowPoint(image.Point{})
			if got := pic.RGBAAt(p.X, p.Y); got != src.RGBAAt(0, 0) {
				t.Errorf("pointer %v and captured pixel disagree: %v", p, got)
			}
		})
	}
	for _, value := range []string{"640", "0x480", "640x-1", "640x480x2", "oops", "99999999999999999999x1", "8193x1"} {
		if _, err := routeWindowSize(value); err == nil {
			t.Errorf("accepted invalid window %q", value)
		}
	}
}

// A Fatal on the command's report stops the walk and becomes the error
// WriteRouteWitness answers; an Error is counted and the walk goes on.
func TestRouteReportStopsOnFatal(t *testing.T) {
	var lines []string
	r := &routeLog{logf: func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }}
	err := r.guard(func() {
		r.Errorf("step %d", 1)
		r.Fatalf("stage %s", "shop")
		r.Errorf("never reached")
	})
	if err == nil || !strings.Contains(err.Error(), "stage shop") {
		t.Fatalf("guard answered %v", err)
	}
	if r.errs != 1 || len(lines) != 2 {
		t.Fatalf("errors %d, lines %q", r.errs, lines)
	}
	if err := (&routeLog{logf: func(string, ...any) {}}).guard(func() {}); err != nil {
		t.Fatalf("a clean walk answered %v", err)
	}
}

// The route clicks the mission's usable Pointer itself: a Pointer without an
// action slot, a sized non-Pointer structure and a slot on a structure of no
// size are passed over (DIV-4178).
func TestRouteSignpostTakesTheUsablePointer(t *testing.T) {
	all := []sim.Structure{
		{ID: 1, Kind: 0x4a, Width: 1, Height: 1},
		{ID: 2, Kind: 15, Width: 1, Height: 1},
		{ID: 3, Kind: 0x46, UseAmount: 24},
		{ID: 4, Kind: 0x4a, UseAmount: 24, Width: 1, Height: 1},
		{ID: 5, Kind: 0x4b, UseAmount: 3, Width: 1, Height: 1},
	}
	if s, ok := routeSignpost(all); !ok || s.ID != 4 {
		t.Fatalf("signpost %+v ok %v, want structure 4", s, ok)
	}
	if s, ok := routeSignpost(all[:3]); ok {
		t.Fatalf("no usable Pointer, got %+v", s)
	}
}

// TestReleaseGeneratorFreePoolTooltip rests the pointer on the detailed page's
// free-points readout through the App's own input and reads the hover box:
// the installed text of the readout's hover slot, in the install's language.
func TestReleaseGeneratorFreePoolTooltip(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the free pool tooltip needs a lawful install")
	}
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	w, stages := newRouteWitness(t, archives.Game(), "")
	for _, s := range stages[:2] {
		w.stage = s.name
		s.walk(w)
	}
	w.stage = "generator-detail"
	if got := w.screen(); got != "chargen/detailed" {
		t.Fatalf("the walk ends on %s, want chargen/detailed", got)
	}
	w.closeChargenTip()
	pool := w.f.ChargenSetup().PreCreate.Art.Layout.Detail.Stats.Pool.Rectangle()
	w.hover("free pool readout", pool, routeMiddle(pool))
	lines := w.app.HeadlessTooltipLines()
	t.Logf("%s: free pool readout %v tooltip %q", archives.Game(), pool, lines)
	if len(lines) == 0 || strings.TrimSpace(strings.Join(lines, "")) == "" {
		t.Fatal("the free pool readout shows no tooltip")
	}
}
