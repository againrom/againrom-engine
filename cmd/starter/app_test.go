package main

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/game"
)

type fakes struct {
	started    [][]string // exe followed by args
	startErr   error
	ran        [][]string
	runOut     map[string]string // by first argument
	runErr     error
	clipboard  string
	validRoots map[string]game.InstallInfo
}

func (f *fakes) deps() deps {
	return deps{
		run: func(exe string, args []string) (string, error) {
			f.ran = append(f.ran, append([]string{exe}, args...))
			return f.runOut[args[len(args)-1]], f.runErr
		},
		start: func(exe string, args []string) error {
			f.started = append(f.started, append([]string{exe}, args...))
			return f.startErr
		},
		paste: func() string { return f.clipboard },
		inspect: func(root string) game.InstallInfo {
			if info, ok := f.validRoots[root]; ok {
				return info
			}
			return game.InstallInfo{Missing: game.RequiredArchives()}
		},
		async: func(fn func()) { fn() },
	}
}

// rig is a starter folder in a temp directory with a fake game program beside
// it and one valid base.
type rig struct {
	t     *testing.T
	dir   string
	self  string
	ini   string
	root  string
	fakes *fakes
}

func newRig(t *testing.T, iniText string) *rig {
	t.Helper()
	dir := t.TempDir()
	r := &rig{t: t, dir: dir, self: filepath.Join(dir, "starter.exe"), ini: filepath.Join(dir, "starter.ini"), root: filepath.Join(dir, "en")}
	if err := os.MkdirAll(r.root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "againrom.exe"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.fakes = &fakes{
		runOut:     map[string]string{"-version": "againrom 0.9.0 abcdef123456\n", "-check": "line one\r\nline two\r\n"},
		validRoots: map[string]game.InstallInfo{r.root: {Language: "english"}},
	}
	if iniText != "" {
		if err := os.WriteFile(r.ini, []byte(strings.ReplaceAll(iniText, "{root}", r.root)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func (r *rig) app() *app {
	a := newApp(r.ini, r.self, "windows", "0.1.0", r.fakes.deps())
	a.poll()
	return a
}

func writeModFolder(t *testing.T, dir, id, manifest string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id, "mod.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func goodMod(id string) string {
	return "id = \"" + id + "\"\ntitle = \"T " + id + "\"\nversion = \"1.0\"\napplies-to = [\"en\"]\n"
}

// do clicks the first item that carries act.
func do(t *testing.T, a *app, act action) {
	t.Helper()
	for _, it := range a.layout() {
		if it.act == act {
			a.click(image.Pt((it.rect.Min.X+it.rect.Max.X)/2, (it.rect.Min.Y+it.rect.Max.Y)/2))
			return
		}
	}
	t.Fatalf("no item carries %+v", act)
}

func typeInto(t *testing.T, a *app, id fieldID, s string) {
	t.Helper()
	do(t, a, action{aFocus, int(id)})
	if a.focus != id {
		t.Fatalf("focus %v, want %v", a.focus, id)
	}
	a.typed(s)
}

func TestMissingIniStartsWithDefaultsAndNoProblem(t *testing.T) {
	r := newRig(t, "")
	a := r.app()
	if a.statusBad || !reflect.DeepEqual(a.s, defaultSettings()) {
		t.Fatalf("%v %q %+v", a.statusBad, a.status, a.s)
	}
	if a.gameVersion != "0.9.0 abcdef123456" {
		t.Fatalf("game version %q", a.gameVersion)
	}
}

func TestMalformedIniLineIsReportedNotFatal(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\nnot a pair\n[bases]\nen = {root}\n")
	a := r.app()
	if !a.statusBad || !strings.Contains(a.status, "line 3") {
		t.Fatalf("status %q bad=%v", a.status, a.statusBad)
	}
	if got, ok := a.selectedRoot(); !ok || got != r.root {
		t.Fatalf("base lost: %q %v", got, ok)
	}
	a.save()
	data, _ := os.ReadFile(r.ini)
	if !strings.Contains(string(data), "not a pair\n") {
		t.Fatalf("malformed line dropped by save:\n%s", data)
	}
}

func TestIniFolderThatCannotBeWrittenIsReportedAndTheWindowKeepsRunning(t *testing.T) {
	r := newRig(t, "")
	blocker := filepath.Join(r.dir, "blocker")
	if err := os.WriteFile(blocker, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.ini = filepath.Join(blocker, "starter.ini")
	a := r.app()
	if !a.statusBad || !strings.Contains(a.status, "not writable") {
		t.Fatalf("status %q", a.status)
	}
	do(t, a, action{kind: aSave})
	if !a.statusBad || !strings.Contains(a.status, "cannot save") || a.closing {
		t.Fatalf("status %q", a.status)
	}
	if len(a.render().Pix) == 0 {
		t.Fatal("window did not render")
	}
}

func TestAddingABaseByTypingOrPasting(t *testing.T) {
	r := newRig(t, "")
	a := r.app()
	typeInto(t, a, fAddBase, r.root)
	a.enter()
	if got, ok := a.selectedRoot(); !ok || got != r.root || a.s.LastBase != "en" {
		t.Fatalf("not added: %q %v %q", got, ok, a.s.LastBase)
	}
	if a.statusBad || !strings.Contains(a.status, "english") || a.addText != "" {
		t.Fatalf("status %q bad=%v add=%q", a.status, a.statusBad, a.addText)
	}
	// paste a quoted path with a trailing line break, as Explorer copies one
	other := filepath.Join(r.dir, "second")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	r.fakes.clipboard = "\"" + other + "\"\r\n"
	do(t, a, action{aFocus, int(fAddBase)})
	a.pasteText()
	do(t, a, action{kind: aAddBase})
	if got, _ := a.s.basePath("second"); got != other {
		t.Fatalf("pasted base: %+v", a.s.Bases)
	}
	if !a.statusBad || !strings.Contains(a.status, "not an install") {
		t.Fatalf("an invalid root was not flagged: %q", a.status)
	}
	// a path that is not a folder is refused and adds nothing
	typeInto(t, a, fAddBase, filepath.Join(r.dir, "nosuch"))
	a.enter()
	if len(a.s.Bases) != 2 || !a.statusBad {
		t.Fatalf("bases %+v status %q", a.s.Bases, a.status)
	}
	do(t, a, action{aSelectBase, 0})
	if a.s.LastBase != "en" {
		t.Fatal(a.s.LastBase)
	}
	do(t, a, action{kind: aRemoveBase})
	if len(a.s.Bases) != 1 || a.s.LastBase != "second" {
		t.Fatalf("%+v %q", a.s.Bases, a.s.LastBase)
	}
}

func TestModListTickOrderAndBrokenFolders(t *testing.T) {
	r := newRig(t, "")
	mods := filepath.Join(r.dir, "mods")
	writeModFolder(t, mods, "alpha", goodMod("alpha"))
	writeModFolder(t, mods, "beta", goodMod("beta"))
	writeModFolder(t, mods, "broken", "id = nope\n")
	a := r.app()
	rows := a.modRows()
	if len(rows) != 3 || rows[2].problem == "" || rows[0].problem != "" {
		t.Fatalf("rows %+v", rows)
	}
	do(t, a, action{aToggleMod, 1}) // beta first
	do(t, a, action{aToggleMod, 1}) // alpha is now row 1
	if !reflect.DeepEqual(a.s.Enabled, []string{"beta", "alpha"}) {
		t.Fatalf("enabled %v", a.s.Enabled)
	}
	if rows := a.modRows(); rows[0].id != "beta" || rows[1].id != "alpha" || !rows[0].enabled {
		t.Fatalf("enabled mods are not listed first in load order: %+v", rows)
	}
	labels := []string{}
	for _, it := range a.layout() {
		if it.kind == iModRow {
			labels = append(labels, it.text)
		}
	}
	if !strings.HasPrefix(labels[0], "1. beta") || !strings.HasPrefix(labels[1], "2. alpha") {
		t.Fatalf("order not shown: %q", labels)
	}
	do(t, a, action{aToggleMod, 2}) // the broken one
	if a.s.modEnabled("broken") || !a.statusBad {
		t.Fatalf("a broken manifest was ticked: %v %q", a.s.Enabled, a.status)
	}
	// an enabled mod that vanished stays listed, can be unticked, and blocks Play
	a.s.Enabled = append(a.s.Enabled, "ghost")
	if rows := a.modRows(); rows[2].id != "ghost" || rows[2].problem == "" {
		t.Fatalf("%+v", rows)
	}
	a.s.LastBase = "en"
	a.s.Bases = []base{{"en", r.root}}
	a.inspectBase(r.root)
	a.play()
	if len(r.fakes.started) != 0 || !a.statusBad || !strings.Contains(a.status, "ghost") {
		t.Fatalf("started %v status %q", r.fakes.started, a.status)
	}
	do(t, a, action{aToggleMod, 2})
	if a.s.modEnabled("ghost") {
		t.Fatal("ghost still enabled")
	}
}

func TestPlayStartsTheGameWithTheExactArgumentList(t *testing.T) {
	mods := "[mods]\nenabled = alpha\n"
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n"+mods+"[options]\nsound = off\nvolume = 30\nvideo = 4x\nmarkers = true\nextra = -skill axe\n")
	writeModFolder(t, filepath.Join(r.dir, "mods"), "alpha", goodMod("alpha"))
	a := r.app()
	do(t, a, action{kind: aPlay})
	want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-mods", "alpha", "-mods-dir", filepath.Join(r.dir, "mods"),
		"-sound=false", "-volume", "30", "-4x", "-markers", "-skill", "axe"}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("started %q\nwant %q", r.fakes.started, want)
	}
	if a.closing || a.statusBad {
		t.Fatalf("closing=%v status %q", a.closing, a.status)
	}
	a.s.CloseOnPlay = true
	do(t, a, action{kind: aPlay})
	if !a.closing {
		t.Fatal("close-on-play did not close")
	}
}

func TestPlayRefusals(t *testing.T) {
	r := newRig(t, "")
	a := r.app()
	a.play()
	if !a.statusBad || !strings.Contains(a.status, "no base game") {
		t.Fatalf("%q", a.status)
	}
	typeInto(t, a, fAddBase, r.root)
	a.enter()
	a.s.Volume = "loud"
	a.play()
	if len(r.fakes.started) != 0 || !strings.Contains(a.status, "volume") {
		t.Fatalf("%v %q", r.fakes.started, a.status)
	}
	a.s.Volume = ""
	a.s.Againrom = filepath.Join(r.dir, "missing.exe")
	a.play()
	if len(r.fakes.started) != 0 || !strings.Contains(a.status, "game program not found") {
		t.Fatalf("%v %q", r.fakes.started, a.status)
	}
	a.s.Againrom = ""
	r.fakes.startErr = errors.New("boom")
	a.play()
	if !a.statusBad || !strings.Contains(a.status, "boom") {
		t.Fatalf("%q", a.status)
	}
	// a folder that is not an install cannot be played
	bad := filepath.Join(r.dir, "bad")
	os.MkdirAll(bad, 0o755)
	a.s.addBase(bad)
	a.inspectBase(bad)
	a.s.LastBase = "bad"
	r.fakes.startErr = nil
	r.fakes.started = nil
	a.play()
	if len(r.fakes.started) != 0 || !strings.Contains(a.status, "not an install") {
		t.Fatalf("%v %q", r.fakes.started, a.status)
	}
}

func TestCheckRunsTheGameWithCheckAndShowsItsOutput(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	a := r.app()
	r.fakes.ran = nil
	do(t, a, action{kind: aCheck})
	a.poll()
	want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-check"}
	if len(r.fakes.ran) != 1 || !reflect.DeepEqual(r.fakes.ran[0], want) {
		t.Fatalf("ran %q want %q", r.fakes.ran, want)
	}
	if !reflect.DeepEqual(a.output, []string{"line one", "line two"}) || a.statusBad || a.status != "check passed" {
		t.Fatalf("output %q status %q", a.output, a.status)
	}
	r.fakes.runErr = errors.New("exit status 1")
	r.fakes.runOut["-check"] = "againrom: boom\n"
	do(t, a, action{kind: aCheck})
	a.poll()
	if !a.statusBad || !strings.Contains(a.status, "exit status 1") || a.output[0] != "againrom: boom" {
		t.Fatalf("output %q status %q", a.output, a.status)
	}
}

func TestVersionProbe(t *testing.T) {
	r := newRig(t, "")
	r.fakes.runErr = errors.New("exit status 2")
	if a := r.app(); a.gameVersion != "unknown" {
		t.Fatalf("failed probe shows %q", a.gameVersion)
	}
	r.fakes.runErr = nil
	r.fakes.runOut["-version"] = "something else\n"
	if a := r.app(); a.gameVersion != "unknown" {
		t.Fatalf("odd output shows %q", a.gameVersion)
	}
	r.fakes.runOut["-version"] = "againrom 1.2.3 deadbeef0000+dirty\n"
	a := r.app()
	if a.gameVersion != "1.2.3 deadbeef0000+dirty" {
		t.Fatal(a.gameVersion)
	}
	os.Remove(filepath.Join(r.dir, "againrom.exe"))
	if b := r.app(); b.gameVersion != "unknown" {
		t.Fatalf("no program shows %q", b.gameVersion)
	}
	texts := []string{}
	for _, it := range a.layout() {
		if it.kind == iText {
			texts = append(texts, it.text)
		}
	}
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "Starter 0.1.0 (") || !strings.Contains(joined, "launches againrom 1.2.3 deadbeef0000+dirty") {
		t.Fatalf("header texts:\n%s", joined)
	}
}

func TestSaveWritesTheIniAndKeepsUnknownKeys(t *testing.T) {
	r := newRig(t, "; hello\n[starter]\nfuture = 1\n")
	a := r.app()
	typeInto(t, a, fAddBase, r.root)
	a.enter()
	do(t, a, action{kind: aSound})
	do(t, a, action{kind: aMovies})
	typeInto(t, a, fVolume, "55")
	a.backspace()
	a.typed("0")
	do(t, a, action{kind: aSave})
	if a.statusBad {
		t.Fatalf("%q", a.status)
	}
	data, err := os.ReadFile(r.ini)
	if err != nil {
		t.Fatal(err)
	}
	b := newApp(r.ini, r.self, "windows", "0.1.0", r.fakes.deps())
	if b.s.Sound != soundOn || b.s.Movies || b.s.Volume != "50" || b.s.LastBase != "en" || len(b.s.Bases) != 1 {
		t.Fatalf("reloaded %+v from\n%s", b.s, data)
	}
	if !strings.HasPrefix(string(data), "; hello\n[starter]\nfuture = 1\n") {
		t.Fatalf("existing lines changed:\n%s", data)
	}
}

func TestSettingsCyclesAndToggles(t *testing.T) {
	r := newRig(t, "")
	a := r.app()
	var seen []string
	for i := 0; i < 3; i++ {
		do(t, a, action{kind: aSound})
		seen = append(seen, a.s.Sound)
	}
	if !reflect.DeepEqual(seen, []string{soundOn, soundOff, soundDefault}) {
		t.Fatal(seen)
	}
	seen = nil
	for i := 0; i < 3; i++ {
		do(t, a, action{kind: aVideo})
		seen = append(seen, a.s.Video)
	}
	if !reflect.DeepEqual(seen, []string{video4x, video8x, videoNormal}) {
		t.Fatal(seen)
	}
	do(t, a, action{kind: aPicker})
	do(t, a, action{kind: aCloseOnPlay})
	do(t, a, action{kind: aMarkers})
	do(t, a, action{kind: aAcceptUnmarked})
	if !a.s.Picker || !a.s.CloseOnPlay || !a.s.Markers || !a.s.AcceptUnmarked {
		t.Fatalf("%+v", a.s)
	}
}

func TestFieldEditing(t *testing.T) {
	r := newRig(t, "")
	a := r.app()
	typeInto(t, a, fExtra, "ab\tc\n")
	if a.s.Extra != "abc" {
		t.Fatalf("control characters kept: %q", a.s.Extra)
	}
	a.backspace()
	a.backspace()
	a.backspace()
	a.backspace()
	if a.s.Extra != "" {
		t.Fatalf("%q", a.s.Extra)
	}
	a.tab()
	if a.focus != fAgainrom {
		t.Fatalf("tab went to %v", a.focus)
	}
	a.tab()
	if a.focus != fAddBase {
		t.Fatalf("tab wrapped to %v", a.focus)
	}
	a.escape()
	a.typed("ignored")
	if a.addText != "" {
		t.Fatal("typed into nothing")
	}
	a.click(image.Pt(winW-1, winH-1))
	if a.focus != fNone {
		t.Fatal("click elsewhere kept focus")
	}
}

func TestWheelScrollsOnlyOverLists(t *testing.T) {
	r := newRig(t, "")
	mods := filepath.Join(r.dir, "mods")
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		writeModFolder(t, mods, id, goodMod(id))
	}
	a := r.app()
	count := func() (first string, n int) {
		for _, it := range a.layout() {
			if it.kind == iModRow {
				if n == 0 {
					first = it.text
				}
				n++
			}
		}
		return
	}
	if first, n := count(); n != modRows || !strings.HasPrefix(first, "a ") {
		t.Fatalf("%q %d", first, n)
	}
	a.wheel(image.Pt(100, 300), 1)
	if first, _ := count(); !strings.HasPrefix(first, "b ") {
		t.Fatalf("%q", first)
	}
	a.wheel(image.Pt(100, 300), 50)
	if first, n := count(); !strings.HasPrefix(first, "c ") || n != modRows {
		t.Fatalf("scroll not clamped: %q %d", first, n)
	}
	a.wheel(image.Pt(600, 300), -5)
	a.wheel(image.Pt(100, 300), -50)
	if first, _ := count(); !strings.HasPrefix(first, "a ") {
		t.Fatalf("%q", first)
	}
}

func TestScreenshotWritesAPNGWithoutAWindow(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	out := filepath.Join(t.TempDir(), "starter.png")
	var stderr strings.Builder
	if code := run([]string{"-ini", r.ini, "-screenshot", out}, nil, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil || img.Bounds() != image.Rect(0, 0, winW, winH) {
		t.Fatalf("%v %v", img, err)
	}
	// the window is not blank: more than one colour is present
	first := img.At(0, 0)
	varied := false
	for y := 0; y < winH && !varied; y += 3 {
		for x := 0; x < winW; x += 3 {
			if img.At(x, y) != first {
				varied = true
				break
			}
		}
	}
	if !varied {
		t.Fatal("blank window")
	}
}

func TestVersionFlag(t *testing.T) {
	var out, errOut strings.Builder
	if code := run([]string{"-version"}, &out, &errOut); code != 0 || !strings.HasPrefix(out.String(), "starter "+programVersion()+" ") || strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("%d %q %q", code, out.String(), errOut.String())
	}
	if code := run([]string{"-bogus"}, &out, &errOut); code != 2 {
		t.Fatalf("exit %d", code)
	}
}

func TestVersionFileIsValid(t *testing.T) {
	if programVersion() == "unknown" {
		t.Fatalf("VERSION %q", versionFile)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
	a := r.app()
	if string(a.render().Pix) != string(a.render().Pix) {
		t.Fatal("render differs between calls")
	}
}
