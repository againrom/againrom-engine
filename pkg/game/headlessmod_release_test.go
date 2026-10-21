package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// headlessEastNames are the names of West and East on this install: Latin on
// an install whose font has no Cyrillic, Cyrillic on the converting one.
func headlessEastNames(t *testing.T) (west, east string) {
	t.Helper()
	if releaseFront(t).textSelector() == text.SelectorConverting {
		return "Запад", "Восток"
	}
	return "West", "East"
}

// headlessEastMods is the fixture mod with West and East renamed.
func headlessEastMods(t *testing.T, west, east string) string {
	t.Helper()
	mods := eastTownMods(t)
	setEastTowns(t, mods, strings.NewReplacer(`"West"`, `"`+west+`"`, `"East"`, `"`+east+`"`).Replace)
	return mods
}

// headlessEastSaves writes a town SAV named "east test" of a party in West,
// mission 20 won so that East is open, into dir.
func headlessEastSaves(t *testing.T, mods, dir string) {
	t.Helper()
	f := townModFront(t, mods, "east-town")
	reachabilityWalkArrive(t, f, "East Hero")
	f.Town.Won(10)
	f.arriveInTown()
	f.Town.Arrive()
	f.Town.Won(20)
	err := f.captureSave(captureCommitted, false, func(s Snapshot, _ string) error {
		encoded, err := encodeSaveLabel("east test", f.textSelector())
		if err != nil {
			return err
		}
		raw, err := f.ExportCurrentSave(s, encoded)
		if err != nil {
			return err
		}
		_, err = SaveStore{Dir: dir}.WriteOriginal("", raw)
		return err
	})
	if err != nil {
		t.Fatalf("town SAVE: %v", err)
	}
}

// headlessWorkDir is a new folder holding scenario.json, the documented
// example, and the saves folder it names with the party's save in it. A
// positive version replaces the example's version.
func headlessWorkDir(t *testing.T, mods string, version int) string {
	t.Helper()
	dir := t.TempDir()
	raw, err := os.ReadFile("testdata/headless-town.json")
	if err != nil {
		t.Fatal(err)
	}
	if version > 0 {
		raw = bytes.Replace(raw, []byte(`"version": 10`), []byte(`"version": `+strconv.Itoa(version)), 1)
	}
	if err := os.WriteFile(filepath.Join(dir, "scenario.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "saves"), 0o755); err != nil {
		t.Fatal(err)
	}
	headlessEastSaves(t, mods, filepath.Join(dir, "saves"))
	return dir
}

// headlessRun runs the scenario at path on a fresh front end with the fixture
// mod, wired as the program wires its headless route, and returns the front
// end, its application, the machine output lines and the trace. Only the
// first steps steps run when steps is positive.
func headlessRun(t *testing.T, mods, path string, steps int) (*FrontEnd, *ui.App, [][]byte, string, error) {
	t.Helper()
	scenario, err := ReadHeadlessScenario(path)
	if err != nil {
		t.Fatal(err)
	}
	if steps > 0 {
		scenario.Steps = scenario.Steps[:steps]
	}
	f := townModFront(t, mods, "east-town")
	app := f.App("againrom-headless")
	f.ConfigureSaveSeams(app, SaveStore{Dir: scenario.Saves}, OriginalStore{Dir: f.Archives.Root}, nil)
	var out, human bytes.Buffer
	err = RunHeadlessScenario(f, app, scenario, &out, &human)
	return f, app, bytes.Split(bytes.TrimSpace(out.Bytes()), []byte("\n")), human.String(), err
}

func readPNG(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// differingBox is the smallest rectangle holding every pixel at which a and b
// differ, and the whole union when their sizes differ; empty when they are
// the same picture.
func differingBox(a, b image.Image) image.Rectangle {
	if a.Bounds() != b.Bounds() {
		return a.Bounds().Union(b.Bounds())
	}
	var box image.Rectangle
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// checkTownCapture judges a capture event of the documented example: the mod
// town's ID, its name as drawn and the mod path of its square.
func checkTownCapture(line []byte, name string) error {
	var event HeadlessEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return err
	}
	if event.Town == nil {
		return errors.New("the capture names no town")
	}
	// The square draws the mod's picture first and, over it, the install's
	// pictures of the buildings; the mod's is the only mod path.
	mod := slices.DeleteFunc(slices.Clone(event.Town.Art), func(key string) bool { return !strings.HasPrefix(key, "mod:") })
	if event.Town.ID != 2 || event.Town.Name != name || len(event.Town.Art) == 0 ||
		event.Town.Art[0] != "mod:art/east-square.bmp" || len(mod) != 1 {
		return errors.New("the capture names the town " + string(mustJSON(event.Town)))
	}
	return nil
}

// The documented example scenario runs on the install with the fixture mod:
// it loads the party's save, travels to East, and its capture names East's
// ID, its name as the install draws it and the mod picture its square draws.
// The frame commands write PNG files equal to the headless renderer's frame
// for the same state.
func TestReleaseHeadlessExampleNamesAndPicturesTheModTown(t *testing.T) {
	west, east := headlessEastNames(t)
	mods := headlessEastMods(t, west, east)
	dir := headlessWorkDir(t, mods, 0)
	_, app, lines, _, err := headlessRun(t, mods, filepath.Join(dir, "scenario.json"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 9 {
		t.Fatalf("%d events, want 9", len(lines))
	}
	if err := checkTownCapture(lines[3], east); err != nil {
		t.Fatal(err)
	}
	// Loss controls: the same event without its town, or with another name,
	// fails the check.
	var stripped map[string]any
	if err := json.Unmarshal(lines[3], &stripped); err != nil {
		t.Fatal(err)
	}
	delete(stripped, "town")
	if err := checkTownCapture(mustJSON(stripped), east); err == nil {
		t.Fatal("a capture without the town fields passes the check")
	}
	if err := checkTownCapture(lines[3], east+"!"); err == nil {
		t.Fatal("a capture naming another town name passes the check")
	}

	// The last frame is the frame of the final state. The square's is the
	// frame of the state after step 6, reached by running those steps alone
	// and asking the renderer directly.
	mapPNG := readPNG(t, filepath.Join(dir, "east-map.png"))
	squarePNG := readPNG(t, filepath.Join(dir, "east-square.png"))
	final, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("renderer frame: %q %v", note, err)
	}
	if box := differingBox(mapPNG, final); !box.Empty() {
		t.Fatalf("east-map.png differs from the renderer's frame inside %v", box)
	}
	dir2 := headlessWorkDir(t, mods, 0)
	_, app2, _, _, err := headlessRun(t, mods, filepath.Join(dir2, "scenario.json"), 6)
	if err != nil {
		t.Fatal(err)
	}
	squareFrame, note, err := app2.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("renderer square frame: %q %v", note, err)
	}
	if box := differingBox(squarePNG, squareFrame); !box.Empty() {
		t.Fatalf("east-square.png differs from the renderer's frame inside %v", box)
	}
	// Loss controls: the frame of another state, or one changed pixel, is
	// found different.
	if box := differingBox(squarePNG, mapPNG); box.Empty() {
		t.Fatal("the square and the world map from East are one picture")
	}
	altered := image.NewRGBA(squarePNG.Bounds())
	for y := altered.Bounds().Min.Y; y < altered.Bounds().Max.Y; y++ {
		for x := altered.Bounds().Min.X; x < altered.Bounds().Max.X; x++ {
			altered.Set(x, y, squarePNG.At(x, y))
		}
	}
	altered.Pix[0] ^= 0xff
	if box := differingBox(altered, squareFrame); box.Empty() {
		t.Fatal("a changed pixel is not found")
	}
}

// East's name is drawn in the frame: the world map from East, written by the
// frame command with East named one thing and then another, differs inside
// the home scroll alone.
func TestReleaseHeadlessFrameDrawsTheModTownName(t *testing.T) {
	west, east := headlessEastNames(t)
	frames := map[string]image.Image{}
	for _, name := range []string{east, east + "x"} {
		mods := headlessEastMods(t, west, name)
		dir := headlessWorkDir(t, mods, 0)
		if _, _, _, _, err := headlessRun(t, mods, filepath.Join(dir, "scenario.json"), 0); err != nil {
			t.Fatal(err)
		}
		frames[name] = readPNG(t, filepath.Join(dir, "east-map.png"))
	}
	box := differingBox(frames[east], frames[east+"x"])
	if box.Empty() {
		t.Fatal("the frame does not change with the town's name")
	}
	if scroll := ui.WorldMapCardRect(-1); !box.In(scroll) {
		t.Fatalf("the name differs in %v, outside the home scroll %v", box, scroll)
	}
}

// A capture of a version 10 scenario adds the town to the event and nothing
// else; the same steps at version 9 print the event without it.
func TestReleaseHeadlessCaptureIsUnchangedWithoutTheNewVersion(t *testing.T) {
	west, east := headlessEastNames(t)
	mods := headlessEastMods(t, west, east)
	var runs [2][][]byte
	for i, version := range []int{10, 9} {
		dir := headlessWorkDir(t, mods, version)
		path := filepath.Join(dir, "scenario.json")
		// Frames are version 10's: the steps up to the capture are the same
		// file at either version.
		scenario := `{"version": ` + strconv.Itoa(version) + `, "saves": "saves", "steps": [
  {"command": "load", "target": "east test"},
  {"command": "activate", "target": "GATES"},
  {"command": "activate", "target": "town 2"},
  {"command": "capture", "name": "east"}]}`
		if err := os.WriteFile(path, []byte(scenario), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, lines, _, err := headlessRun(t, mods, path, 4)
		if err != nil {
			t.Fatal(err)
		}
		runs[i] = lines
	}
	for i := range runs[0] {
		var ten, nine map[string]any
		if err := json.Unmarshal(runs[0][i], &ten); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(runs[1][i], &nine); err != nil {
			t.Fatal(err)
		}
		if _, has := nine["town"]; has {
			t.Fatalf("event %d of the version 9 run names a town", i+1)
		}
		if _, has := ten["town"]; has != (i == 3) {
			t.Fatalf("event %d of the version 10 run: town present %v", i+1, has)
		}
		delete(ten, "town")
		if !reflect.DeepEqual(ten, nine) {
			t.Fatalf("event %d differs between the versions:\n%s\n%s", i+1, runs[0][i], runs[1][i])
		}
	}
}

// A frame command refuses a path inside the install, inside a preserved
// install, over an existing file, and a scenario naming such a path writes
// nothing, not even for the frame commands before it.
func TestReleaseHeadlessFrameRefusalsWriteNothing(t *testing.T) {
	west, east := headlessEastNames(t)
	mods := headlessEastMods(t, west, east)
	dir := headlessWorkDir(t, mods, 0)
	f := townModFront(t, mods, "east-town")
	root := f.Archives.Root

	existing := filepath.Join(dir, "taken.png")
	if err := os.WriteFile(existing, []byte("taken"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		path string
		want error
	}{
		{"the install", filepath.Join(root, "frame.png"), ErrHeadlessFrameInstall},
		{"a folder of the install", filepath.Join(root, "new-folder", "frame.png"), ErrHeadlessFrameInstall},
		{"an existing file", existing, ErrHeadlessFrameExists},
	} {
		if err := f.checkHeadlessFramePath(c.path); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
	if sibling := filepath.Join(filepath.Dir(root), "ru"); strings.EqualFold(filepath.Base(filepath.Dir(root)), "gameversions") {
		if err := f.checkHeadlessFramePath(filepath.Join(sibling, "frame.png")); !errors.Is(err, ErrHeadlessFrameInstall) {
			t.Errorf("a preserved install: %v", err)
		}
	}

	before := listFiles(t, dir)
	for _, bad := range []string{filepath.Join(root, "frame.png"), existing} {
		scenario := HeadlessScenario{Version: 10, Saves: filepath.Join(dir, "saves"), Steps: []HeadlessStep{
			{Command: "load", Target: "east test"},
			{Command: "frame", Path: filepath.Join(dir, "first.png")},
			{Command: "frame", Path: bad},
		}}
		g := townModFront(t, mods, "east-town")
		app := g.App("againrom-headless")
		g.ConfigureSaveSeams(app, SaveStore{Dir: scenario.Saves}, OriginalStore{Dir: g.Archives.Root}, nil)
		var out, human bytes.Buffer
		err := RunHeadlessScenario(g, app, scenario, &out, &human)
		if err == nil || !strings.Contains(err.Error(), "step 3 (frame)") {
			t.Errorf("%s: %v", bad, err)
		}
		if out.Len() != 0 {
			t.Errorf("%s: the run printed events before refusing: %s", bad, out.String())
		}
	}
	if after := listFiles(t, dir); !slices.Equal(before, after) {
		t.Fatalf("a refused run changed the folder:\n%q\n%q", before, after)
	}
	if got, _ := os.ReadFile(existing); string(got) != "taken" {
		t.Fatalf("the existing file was written over: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "frame.png")); err == nil {
		t.Fatal("a frame was written into the install")
	}
}

// listFiles are the files under dir with their sizes, sorted.
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, path+" "+string(mustJSON(info.Size())))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

func TestReleaseHeadlessPointerActionsReachTheTown(t *testing.T) {
	west, east := headlessEastNames(t)
	mods := headlessEastMods(t, west, east)
	dir := headlessWorkDir(t, mods, 0)
	steps := `{"command": "load", "target": "east test"},
  {"command": "activate", "target": "GATES"},
  {"command": "activate", "target": "town 2"}`
	for _, action := range []string{"wheel-up", "middle-press", "middle-release", "ctrl-hover", "shift-press", "shift-release"} {
		steps += `,
  {"command": "pointer", "action": "` + action + `", "at": {"tip_close": true}}`
	}
	steps += `,
  {"command": "capture", "name": "after"}`
	path := filepath.Join(dir, "pointer.json")
	scenario := `{"version": 10, "saves": "saves", "steps": [` + steps + `]}`
	if err := os.WriteFile(path, []byte(scenario), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, lines, human, err := headlessRun(t, mods, path, 0)
	if err != nil {
		t.Fatalf("%v\n%s", err, human)
	}
	if len(lines) != 10 {
		t.Fatalf("%d events, want 10", len(lines))
	}
	if err := checkTownCapture(lines[9], east); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(scenario, `"wheel-up"`, `"wheel-left"`, 1)
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err = headlessRun(t, mods, path, 0)
	if err == nil || !strings.Contains(err.Error(), "step 4") || !strings.Contains(err.Error(), "wheel-down") {
		t.Fatalf("an action outside the list: %v", err)
	}
}

// A scenario's frame is the same however long the machine took to reach it:
// the town's ambient animation reads the scenario's own clock, not the wall
// clock. The frame is asked for again after the wall clock has moved on, and
// a second run of the same steps is asked as well.
func TestReleaseHeadlessFrameDoesNotDependOnWallClock(t *testing.T) {
	west, east := headlessEastNames(t)
	mods := headlessEastMods(t, west, east)
	frameAfter := func() (*ui.App, image.Image) {
		dir := headlessWorkDir(t, mods, 0)
		_, app, _, _, err := headlessRun(t, mods, filepath.Join(dir, "scenario.json"), 6)
		if err != nil {
			t.Fatal(err)
		}
		pix, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return app, pix
	}
	app, first := frameAfter()
	for _, wait := range []time.Duration{300 * time.Millisecond, 700 * time.Millisecond, 1500 * time.Millisecond} {
		time.Sleep(wait)
		again, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		if box := differingBox(first, again); !box.Empty() {
			t.Fatalf("the frame changed with %v of wall time and no step, inside %v", wait, box)
		}
	}
	_, second := frameAfter()
	if box := differingBox(first, second); !box.Empty() {
		t.Fatalf("two runs of the same steps give different frames inside %v", box)
	}
}

// A scenario of version 9 or lower keeps the clock it always had: the run
// gives the front end no scenario clock, so what it prints is what it printed.
func TestReleaseHeadlessOlderVersionKeepsTheWallClock(t *testing.T) {
	west, east := headlessEastNames(t)
	mods := headlessEastMods(t, west, east)
	dir := headlessWorkDir(t, mods, 9)
	path := filepath.Join(dir, "scenario.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["steps"] = doc["steps"].([]any)[:3]
	if err := os.WriteFile(path, mustJSON(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	f, _, _, _, err := headlessRun(t, mods, path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if f.TownAnimationNow != nil {
		t.Fatal("a version-9 run installed a scenario clock")
	}
}
