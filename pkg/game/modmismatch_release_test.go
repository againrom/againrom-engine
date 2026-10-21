package game

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// mismatchMods writes the fixture mods under a new folder: alpha in version,
// with a skill_cap setting its script applies; beta, a second mod that sets a
// rule; and extra mods m01..mNN that set a rule too. Every one changes game
// state, so each is written into a save's mod mark.
func mismatchMods(t *testing.T, version string, extra int) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"alpha/mod.toml":      "id = \"alpha\"\ntitle = \"Alpha\"\nversion = \"" + version + "\"\nentry = \"main.star\"\n",
		"alpha/settings.toml": "[skill_cap]\nlabel = { en = \"Skill cap\" }\ntype = \"int\"\ndefault = 150\nmin = 100\nmax = 150\n",
		"alpha/main.star":     "def init(game, settings):\n    game.rules.skill_cap = settings.skill_cap\n",
		"beta/mod.toml":       "id = \"beta\"\ntitle = \"Beta\"\nversion = \"1.0\"\nentry = \"main.star\"\n",
		"beta/main.star":      "def init(game, settings):\n    game.rules.skill_cap = 140\n",
	}
	for i := 1; i <= extra; i++ {
		id := fmt.Sprintf("m%02d", i)
		files[id+"/mod.toml"] = "id = \"" + id + "\"\ntitle = \"" + id + "\"\nversion = \"2.0\"\nentry = \"main.star\"\n"
		files[id+"/main.star"] = "def init(game, settings):\n    game.rules.skill_cap = 130\n"
	}
	for name, body := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// withMismatchMods runs the mods ids of root on f, with the given -mod-setting
// arguments.
func withMismatchMods(t *testing.T, f *FrontEnd, root string, ids []string, settings ...string) *FrontEnd {
	t.Helper()
	if len(ids) == 0 {
		return f
	}
	entries, err := mod.Resolve(root, ids)
	if err != nil {
		t.Fatal(err)
	}
	var flags []mod.SettingFlag
	for _, s := range settings {
		flag, err := mod.ParseSettingFlag(s)
		if err != nil {
			t.Fatal(err)
		}
		flags = append(flags, flag)
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), flags, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range res.Set.Mods {
		if m.StateFree {
			t.Fatalf("fixture mod %s is state-free, so no mark names it", m.ID)
		}
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	return f
}

// mismatchFrame writes a frame under AGAINROM_MOD_MISMATCH_FRAMES, in a folder
// named for the install, when the variable is set.
func mismatchFrame(t *testing.T, name string, pix *image.RGBA) {
	t.Helper()
	root := os.Getenv("AGAINROM_MOD_MISMATCH_FRAMES")
	if root == "" || pix == nil {
		return
	}
	if !filepath.IsAbs(root) {
		t.Fatal("AGAINROM_MOD_MISMATCH_FRAMES must be absolute")
	}
	dir := filepath.Join(root, filepath.Base(filepath.Clean(os.Getenv("AGAINROM_ASSETS"))))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, pix); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".png"), b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// alertPixels counts the red pixels of r: the alert ink and the shades the
// installed font draws it in.
func alertPixels(pix *image.RGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := pix.RGBAAt(x, y)
			if c.R > 120 && int(c.G)*2 < int(c.R) && int(c.B)*2 < int(c.R) {
				n++
			}
		}
	}
	return n
}

// mismatchWindow chooses the row of app's open Load page that target names
// and returns the window the refusal opened with its frame.
func mismatchWindow(t *testing.T, app *ui.App, target string) (ui.HeadlessModMismatch, *image.RGBA) {
	t.Helper()
	if app.Screen() != ui.ScreenLoad {
		t.Fatalf("Load page %v", app.Screen())
	}
	if err := app.HeadlessActivate(target); err != nil {
		t.Fatal(err)
	}
	w, ok := app.HeadlessModMismatch()
	if !ok || app.Screen() != ui.ScreenLoad {
		t.Fatalf("no mismatch window on %v; the message line says %q", app.Screen(), app.HeadlessMessage())
	}
	if app.HeadlessMessage() != "" {
		t.Fatalf("the refusal also reached the message line: %q", app.HeadlessMessage())
	}
	pix, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal(note, err)
	}
	if n := alertPixels(pix, w.List); n == 0 {
		t.Fatalf("the window's list draws no red: rows %q", w.Rows)
	}
	return w, pix
}

// TestReleaseLoadPageListsEveryModMismatch: a mission SAVE made under the
// fixture mod alpha 1.0 is refused on the production Load page under alpha
// 1.1, without alpha, with beta added, with another setting, and with twelve
// more mods; a SAVE made without mods is refused under alpha. Each refusal
// opens the window listing its differences with the mismatched values marked
// red, and Escape leaves the Load page; under alpha 1.0 the save loads.
func TestReleaseLoadPageListsEveryModMismatch(t *testing.T) {
	v10, v11 := mismatchMods(t, "1.0", 12), mismatchMods(t, "1.1", 0)
	save := func(f *FrontEnd, who string) SaveStore {
		t.Helper()
		f.SetDeterministicFrames(true)
		store := SaveStore{Dir: t.TempDir()}
		party := f.ChargenParty(ui.ChargenResult{Name: who, Choices: []int{1, 0, 0}, Stats: []int{31, 27, 24, 29}})
		if err := f.App(who).OpenMission(f.MissionOpenerWith(10, party)); err != nil {
			t.Fatalf("open mission 10: %v", err)
		}
		saveTo, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
		if name, err := saveTo(true); err != nil || !IsOriginal(name) {
			t.Fatalf("mission SAVE wrote %q: %v", name, err)
		}
		return store
	}
	marked := save(withMismatchMods(t, releaseFront(t), v10, []string{"alpha"}), "Marked")
	plain := save(releaseFront(t), "Plain")
	loadPage := func(f *FrontEnd, store SaveStore) *ui.App {
		t.Helper()
		f.SetDeterministicFrames(true)
		app := f.App("mod mismatch")
		app.Layout(640, 480)
		f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
		if err := app.HeadlessKey("load"); err != nil || app.Screen() != ui.ScreenLoad {
			t.Fatal("Load page", err, app.Screen())
		}
		return app
	}
	extras := []string{"alpha"}
	for i := 1; i <= 12; i++ {
		extras = append(extras, fmt.Sprintf("m%02d", i))
	}
	for _, c := range []struct {
		name     string
		save     SaveStore
		root     string
		ids      []string
		settings []string
		rows     int
		last     []string
	}{
		{"version", marked, v11, []string{"alpha"}, nil, 1, []string{"alpha", "[1.0]", "[1.1]"}},
		{"missing", marked, "", nil, nil, 1, []string{"alpha 1.0", "["}},
		{"extra-state-mod", marked, v10, []string{"alpha", "beta"}, nil, 1, []string{"beta 1.0", "["}},
		{"setting", marked, v10, []string{"alpha"}, []string{"alpha.skill_cap=120"}, 1, []string{"alpha", "skill_cap", "[150]", "[120]"}},
		{"unmarked", plain, v10, []string{"alpha"}, nil, 2, []string{"alpha 1.0", "["}},
		{"many", marked, v10, extras, nil, 12, []string{"m12 2.0", "["}},
	} {
		app := loadPage(withMismatchMods(t, releaseFront(t), c.root, c.ids, c.settings...), c.save)
		if len(app.HeadlessRows()) != 1 {
			t.Fatalf("%s: Load rows %v, want the one save", c.name, app.HeadlessRows())
		}
		w, pix := mismatchWindow(t, app, "@first")
		mismatchFrame(t, c.name, pix)
		if len(w.Rows) != c.rows {
			t.Errorf("%s: %d rows %q, want %d", c.name, len(w.Rows), w.Rows, c.rows)
		}
		last := w.Rows[len(w.Rows)-1]
		for _, part := range c.last {
			if !strings.Contains(last, part) {
				t.Errorf("%s: the last row %q does not hold %q; rows %q", c.name, last, part, w.Rows)
			}
		}
		if c.name == "many" {
			if w.Lines <= w.Visible {
				t.Errorf("many: %d lines need no scrolling", w.Lines)
			}
			if err := app.HeadlessKey("end"); err != nil {
				t.Fatal(err)
			}
			if got, _ := app.HeadlessModMismatch(); got.Top != got.Lines-got.Visible || got.Top == 0 {
				t.Errorf("many: End scrolled to line %d of %d", got.Top, got.Lines)
			}
			scrolled, _, err := app.HeadlessFrame()
			if err != nil {
				t.Fatal(err)
			}
			mismatchFrame(t, c.name+"-scrolled", scrolled)
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if _, open := app.HeadlessModMismatch(); open || app.Screen() != ui.ScreenLoad || len(app.HeadlessRows()) != 1 {
			t.Fatalf("%s: Escape left the window open %v on %v", c.name, open, app.Screen())
		}
	}
	app := loadPage(withMismatchMods(t, releaseFront(t), v10, []string{"alpha"}), marked)
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatalf("LOAD under the save's own mods: %v on %v, %q", err, app.Screen(), app.HeadlessMessage())
	}
}

// TestReleaseSecondGameLoadPageListsModMismatches: the second game's Load page
// shows the same window. A SAVE made on the save page under alpha 1.0 is
// refused under alpha 1.1 with the two versions marked red.
func TestReleaseSecondGameLoadPageListsModMismatches(t *testing.T) {
	v10, v11 := mismatchMods(t, "1.0", 0), mismatchMods(t, "1.1", 0)
	dir := t.TempDir()
	app := secondPageApp(t, withMismatchMods(t, secondGameFront(t), v10, []string{"alpha"}), dir)
	enterSecondCampaignMission(t, app)
	secondSaveVia(t, app, "", "Mods")
	cold := secondPageApp(t, withMismatchMods(t, secondGameFront(t), v11, []string{"alpha"}), dir)
	if err := cold.HeadlessActivate("load game"); err != nil {
		t.Fatal(err)
	}
	w, pix := mismatchWindow(t, cold, "Mods")
	if len(w.Rows) != 1 || !strings.Contains(w.Rows[0], "[1.0]") || !strings.Contains(w.Rows[0], "[1.1]") {
		t.Fatalf("rows %q, want alpha's two versions", w.Rows)
	}
	mismatchFrame(t, "load-page-version", pix)
	secondPageWitness(t, "load-page-mod-mismatch", pix)
	if err := cold.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if _, open := cold.HeadlessModMismatch(); open || cold.Screen() != ui.ScreenLoad {
		t.Fatalf("Escape left the window open %v on %v", open, cold.Screen())
	}
}
