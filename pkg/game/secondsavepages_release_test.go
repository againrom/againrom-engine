package game

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/menu"
	"againrom/pkg/ui"
)

// secondPageWitness writes a frame for the owner when a witness directory is
// set.
func secondPageWitness(t *testing.T, name string, pix *image.RGBA) {
	t.Helper()
	root := os.Getenv("AGAINROM_SECOND_SAVE_OUTPUT")
	if root == "" || pix == nil {
		return
	}
	dir := filepath.Join(root, filepath.Base(os.Getenv("AGAINROM_ASSETS")), "pages")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, pix); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".png"), b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func secondFrame(t *testing.T, app *ui.App) *image.RGBA {
	t.Helper()
	pix, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	return pix
}

// secondButtonPoint is a frame point the mask gives to the button bound to
// action.
func secondButtonPoint(t *testing.T, f *FrontEnd, action menu.Action) image.Point {
	t.Helper()
	for y := 0; y < menu.FrameH; y++ {
		for x := 0; x < menu.FrameW; x++ {
			if b := f.Assets.ButtonAt(image.Pt(x, y)); b != 0 && f.Assets.Binding(b).Action == action {
				return image.Pt(x, y)
			}
		}
	}
	t.Fatalf("no mask point for action %d", action)
	return image.Point{}
}

func secondPageApp(t *testing.T, f *FrontEnd, dir string) *ui.App {
	t.Helper()
	app := f.App("second pages")
	app.Layout(640, 480)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	return app
}

func TestReleaseSecondGameMenuLocksLoadUntilASaveFileExists(t *testing.T) {
	f := secondGameFront(t)
	load := secondButtonPoint(t, f, menu.ActionLoad)
	hover := func(app *ui.App, p image.Point) *image.RGBA {
		if err := app.HeadlessPointer("hover", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		return secondFrame(t, app)
	}
	hovered := map[bool]*image.RGBA{}
	for _, tc := range []struct {
		name   string
		file   string
		locked bool
	}{
		{"none", "", true},
		{"other name", "save.sav", true},
		{"game file without a save", "game0042.sav", false},
	} {
		dir := t.TempDir()
		if tc.file != "" {
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte("not a save"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		app := secondPageApp(t, f, dir)
		away := hover(app, image.Pt(2, 2))
		over := hover(app, load)
		secondPageWitness(t, "menu-load-"+map[bool]string{true: "locked", false: "unlocked"}[tc.locked], over)
		if bytes.Equal(away.Pix, over.Pix) {
			t.Errorf("%s: hovered Load frame equals the idle frame; the caption is missing", tc.name)
		}
		hovered[tc.locked] = over
		err := app.HeadlessActivate("load game")
		if (err != nil) != tc.locked {
			t.Errorf("%s: activate Load: %v, want locked %v", tc.name, err, tc.locked)
		}
		if !tc.locked {
			if app.Screen() != ui.ScreenLoad || len(app.HeadlessRows()) != 0 {
				t.Errorf("%s: Load opened %v with rows %v; a file that is no save is not listed", tc.name, app.Screen(), app.HeadlessRows())
			}
			secondPageWitness(t, "load-page-empty", secondFrame(t, app))
		}
	}
	if !bytes.Equal(hovered[true].Pix, hovered[false].Pix) {
		t.Error("a locked Load draws a different hover frame than an unlocked one")
	}
}

func TestReleaseSecondGameMenuButtonsReachTheirScreens(t *testing.T) {
	f := secondGameFront(t)
	for _, tc := range []struct {
		action menu.Action
		name   string
		want   ui.Screen
	}{
		{menu.ActionNone, "multi player", ui.ScreenMenu},
		{menu.ActionNone, "server", ui.ScreenMenu},
		{menu.ActionCredits, "credits", ui.ScreenCredits},
		{menu.ActionHallOfFame, "hall of fame", ui.ScreenEnding},
		{menu.ActionNewGame, "new game", ui.ScreenTown},
	} {
		app := secondPageApp(t, f, t.TempDir())
		if err := app.HeadlessActivate(tc.name); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if app.Screen() != tc.want {
			t.Errorf("%s opens %v, want %v", tc.name, app.Screen(), tc.want)
		}
	}
}

// secondRowStrip is visible list row i's pixels.
func secondRowStrip(pix *image.RGBA, list image.Rectangle, pitch, i int) []byte {
	r := image.Rect(list.Min.X+2, list.Min.Y+i*pitch+1, list.Max.X-2, list.Min.Y+(i+1)*pitch-1)
	var out []byte
	for y := r.Min.Y; y < r.Max.Y; y++ {
		out = append(out, pix.Pix[pix.PixOffset(r.Min.X, y):pix.PixOffset(r.Max.X, y)]...)
	}
	return out
}

// secondSaveVia saves through the F2 page, choosing the last row labelled over
// when it is not empty, and returns to the map.
func secondSaveVia(t *testing.T, app *ui.App, over, label string) {
	t.Helper()
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("save page", err, app.Screen())
	}
	for i, r := range app.HeadlessRows() {
		if over != "" && r.Text == over {
			if err := app.HeadlessSaveSelect(i); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := app.HeadlessSaveEdit("", label, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap {
		t.Fatal("save page stayed open", app.Screen(), app.HeadlessMessage())
	}

}

func TestReleaseSecondGameSavePageWritesSlotsAndLoadPageLoadsThem(t *testing.T) {
	f := secondGameFront(t)
	dir := t.TempDir()
	app := secondPageApp(t, f, dir)
	enterSecondCampaignMission(t, app)
	pages := secondSavePages(f)
	pitch := f.Font.Value().Height() + 4

	save := func(over, label string) {
		t.Helper()
		secondSaveVia(t, app, over, label)
	}
	files := func() []string {
		var out []string
		for _, e := range ownGameSaves(dir, 0, f.savedGame()) {
			out = append(out, e.Name+"="+e.Label)
		}
		return out
	}
	app.FlushBackground()
	start := files()
	if len(start) != 1 {
		t.Fatalf("mission entry wrote %v, want its one autosave", start)
	}

	// The save page shows the Empty Slot row above the listed saves.
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("save page", err, app.Screen())
	}
	rows := app.HeadlessRows()
	if len(rows) != 2 || rows[0].Text != pages.Save.EmptySlot {
		t.Fatalf("save page rows %v, want the Empty Slot and the autosave", rows)
	}
	if strings.ContainsRune(rows[0].Text, '?') || strings.TrimSpace(rows[0].Text) == "" {
		t.Fatalf("Empty Slot row %q does not decode from the installed text", rows[0].Text)
	}
	withSlot := secondFrame(t, app)
	secondPageWitness(t, "save-page", withSlot)
	slotPanel := app.GameMenuPanel()
	if slotPanel == nil {
		t.Fatal("save page has no native menu layer")
	}
	blank := *pages.Save
	blank.EmptySlot = ""
	app.SetSavePages(ui.SavePages{Load: pages.Load, Save: &blank})
	if err := app.HeadlessSaveAction("cancel"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("save page", err, app.Screen())
	}
	blankPanel := app.GameMenuPanel()
	if blankPanel == nil {
		t.Fatal("blank save page has no native menu layer")
	}
	if bytes.Equal(secondRowStrip(slotPanel, pages.Save.List, pitch, 0), secondRowStrip(blankPanel, pages.Save.List, pitch, 0)) {
		t.Fatal("row 0 draws nothing of the Empty Slot text")
	}
	if err := app.HeadlessSaveAction("cancel"); err != nil {
		t.Fatal(err)
	}
	app.SetSavePages(*pages)

	// Row 0 takes the smallest unused number; a chosen row is overwritten
	// without confirmation under the new label.
	save("", "Page one")
	save("", "Page two")
	save("Page one", "Page three")
	got := files()
	want := []string{"game0001.sav=Page three", "game0002.sav=Page two", start[0]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("saves %v, want %v", got, want)
	}
	before := secondSaveSampleNow(t, f, app)

	// A cold start loads the newest row from the main menu's load page.
	cold := secondGameFront(t)
	coldApp := secondPageApp(t, cold, dir)
	if err := coldApp.HeadlessActivate("load game"); err != nil || coldApp.Screen() != ui.ScreenLoad {
		t.Fatal("load page", err, coldApp.Screen())
	}
	var labels []string
	for _, r := range coldApp.HeadlessRows() {
		labels = append(labels, r.Text)
	}
	if want := []string{"Page three", "Page two", ownGameSaves(dir, 0, f.savedGame())[2].Label}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("load rows %q, want %q", labels, want)
	}
	several := secondFrame(t, coldApp)
	secondPageWitness(t, "load-page-several", several)
	for i := 1; i < 3; i++ {
		if bytes.Equal(secondRowStrip(several, pages.Load.List, pitch, i), secondRowStrip(several, pages.Load.List, pitch, 3)) {
			t.Errorf("load row %d draws like an empty row", i)
		}
	}
	if err := coldApp.HeadlessActivate("Page three"); err != nil || coldApp.Screen() != ui.ScreenMap {
		t.Fatal("cold LOAD", err, coldApp.Screen(), coldApp.HeadlessMessage())
	}
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "load page")
	secondAssertSample(t, before, secondSaveSampleNow(t, cold, coldApp))
	for range 16 {
		f.live.tick()
		cold.live.tick()
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "post-load tick")
	}

	// Loss control: the list follows the last write time.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "game0001.sav"), old, old); err != nil {
		t.Fatal(err)
	}
	if rows := ownGameSaves(dir, 0, f.savedGame()); rows[0].Name != "game0002.sav" {
		t.Fatalf("an older game0001.sav still lists first: %v", rows)
	}

	// One save: the load page shows one row.
	one := t.TempDir()
	raw, err := os.ReadFile(filepath.Join(dir, "game0002.sav"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(one, "game0000.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	oneApp := secondPageApp(t, cold, one)
	if err := oneApp.HeadlessActivate("load game"); err != nil || len(oneApp.HeadlessRows()) != 1 {
		t.Fatal("one-save load page", err, oneApp.HeadlessRows())
	}
	secondPageWitness(t, "load-page-one", secondFrame(t, oneApp))
}

func TestReleaseSecondGameSavePageOverwritesTheChosenOfTwoEqualLabels(t *testing.T) {
	f := secondGameFront(t)
	dir := t.TempDir()
	app := secondPageApp(t, f, dir)
	enterSecondCampaignMission(t, app)
	app.FlushBackground()
	save := func(over, label string) {
		t.Helper()
		secondSaveVia(t, app, over, label)
	}

	// Two saves of one label: the chosen lower row is overwritten and the
	// upper row's file keeps its bytes.
	save("", "Twin")
	upper, lower := filepath.Join(dir, "game0002.sav"), filepath.Join(dir, "game0001.sav")
	// The page cannot name a second save "Twin" (the name selects the first),
	// so the pair is made as the engine makes its own: a second file with the label.
	twinRaw, err := os.ReadFile(lower)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(upper, twinRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	lowerWhen, upperWhen := time.Now().Add(-3*time.Hour), time.Now().Add(-2*time.Hour)
	if err := os.Chtimes(lower, lowerWhen, lowerWhen); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(upper, upperWhen, upperWhen); err != nil {
		t.Fatal(err)
	}
	upperBefore, err := os.ReadFile(upper)
	if err != nil {
		t.Fatal(err)
	}
	save("Twin", "Twin")
	upperAfter, err := os.ReadFile(upper)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(upperBefore, upperAfter) {
		t.Error("the upper Twin save changed")
	}
	if info, err := os.Stat(upper); err != nil || !info.ModTime().Equal(upperWhen) {
		t.Error("Ok wrote the first row with the label, not the chosen row", err)
	}
	if info, err := os.Stat(lower); err != nil || info.ModTime().Before(time.Now().Add(-time.Hour)) {
		t.Error("the chosen lower Twin save was not rewritten", err)
	}
	twinCold := secondGameFront(t)
	twinApp := secondPageApp(t, twinCold, dir)
	if err := twinApp.HeadlessActivate("load game"); err != nil || twinApp.Screen() != ui.ScreenLoad {
		t.Fatal("load page", err, twinApp.Screen())
	}
	twins := 0
	for _, r := range twinApp.HeadlessRows() {
		if r.Text == "Twin" {
			twins++
		}
	}
	if twins != 2 {
		t.Fatalf("the load page lists %d Twin rows, want 2", twins)
	}
	if err := twinApp.HeadlessKey("enter"); err != nil || twinApp.Screen() != ui.ScreenMap {
		t.Fatal("cold LOAD of Twin", err, twinApp.Screen(), twinApp.HeadlessMessage())
	}
	assertCurrentWorldEqual(t, f.live.world, twinCold.live.world, "overwritten twin")
}
