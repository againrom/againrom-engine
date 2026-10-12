package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/render/menu"
	"againrom/pkg/screenart"
)

// secondAppAssets is appAssets bound through the second game's loader: the
// same synthetic art plus eight captions, so its table is SecondActions.
func secondAppAssets(t *testing.T) *menu.Assets {
	t.Helper()
	first := appAssets(t)
	set := &screenart.Set{Frames: map[string][]image.Image{"base": {first.Base}}, Masks: map[string]*image.Paletted{"mask": first.Mask}}
	for i := 0; i < menu.ButtonCount; i++ {
		set.Frames[fmt.Sprintf("hover/%d", i)] = []image.Image{first.Hover[i]}
		set.Frames[fmt.Sprintf("pressed/%d", i)] = []image.Image{first.Pressed[i]}
		c := image.NewRGBA(image.Rect(0, 0, 180, 80))
		draw.Draw(c, c.Bounds(), image.NewUniform(color.RGBA{0, 0, uint8(i * 20), 255}), image.Point{}, draw.Src)
		set.Frames[fmt.Sprintf("caption/%d", i)] = []image.Image{c}
	}
	a, err := menu.LoadSecond(set)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func clickButton(a *App, button int, now time.Time) {
	x, y := centreOf(button)
	a.step(appInput{CursorX: x, CursorY: y}, now)
	a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, now)
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
}

func buttonOf(t *testing.T, a *menu.Assets, action menu.Action) int {
	t.Helper()
	for b := 1; b <= menu.ButtonCount; b++ {
		if a.Binding(b).Action == action {
			return b
		}
	}
	t.Fatalf("no button binds action %d", action)
	return 0
}

func TestSecondMenuLocksLoadWithoutSave(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	assets := secondAppAssets(t)
	load := buttonOf(t, assets, menu.ActionLoad)
	if load != 5 {
		t.Fatalf("second game's Load is button %d, want 5 (mask 0xc0)", load)
	}
	for _, has := range []bool{false, true} {
		a := NewApp("t", assets, appRows(1), okLoader(t))
		a.Layout(640, 480)
		a.SetSaveSeams(nil, func() []SaveEntry { return nil }, nil)
		a.SetSaveFiles(func() bool { return has })
		x, y := centreOf(load)
		a.step(appInput{CursorX: x, CursorY: y}, now)
		if got, want := a.sel.State().Selected, map[bool]int{false: 0, true: load}[has]; got != want {
			t.Errorf("has save %v: hover over Load selects %d, want %d", has, got, want)
		}
		clickButton(a, load, now)
		if got, want := a.Screen(), map[bool]Screen{false: ScreenMenu, true: ScreenLoad}[has]; got != want {
			t.Errorf("has save %v: Load opens %v, want %v", has, got, want)
		}
		if !has {
			a.step(appInput{Load: true}, now)
			if a.Screen() != ScreenMenu {
				t.Error("the L key opened a locked Load")
			}
			if err := a.HeadlessActivate("load game"); err == nil {
				t.Error("headless activated a locked Load")
			}
		}
	}
	// Loss control: the first game's table never locks Load.
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetSaveSeams(nil, func() []SaveEntry { return nil }, nil)
	a.SetSaveFiles(func() bool { return false })
	clickButton(a, menu.LoadGameButton, now)
	if a.Screen() != ScreenLoad {
		t.Fatalf("first game's Load without a save opens %v, want the LOAD window", a.Screen())
	}
}

func TestSecondMenuMultiplayerButtonsDoNothingAndCutscenesNeedASeenScene(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	assets := secondAppAssets(t)
	for _, b := range []int{2, 6} {
		if got := assets.Binding(b); got.Action != menu.ActionNone || got.Name == "" {
			t.Fatalf("button %d binding %+v, want a named button with no action", b, got)
		}
		a := NewApp("t", assets, appRows(1), okLoader(t))
		a.Layout(640, 480)
		clickButton(a, b, now)
		if a.Screen() != ScreenMenu {
			t.Errorf("button %d opened %v", b, a.Screen())
		}
	}
	cut := buttonOf(t, assets, menu.ActionCutscenes)
	entries := []CutsceneEntry{{Directory: "intro", Title: "Intro"}}
	for _, seen := range [][]string{nil, {"intro"}} {
		a := NewApp("t", assets, appRows(1), okLoader(t))
		a.Layout(640, 480)
		a.SetCutsceneLibrary(entries, seen, nil, CutsceneLibraryWords{})
		clickButton(a, cut, now)
		want := ScreenMenu
		if len(seen) != 0 {
			want = ScreenCutsceneLibrary
		}
		if a.Screen() != want {
			t.Errorf("seen %v: Cut Scenes opens %v, want %v", seen, a.Screen(), want)
		}
	}
	// Loss control: the first game opens its library with nothing seen.
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetCutsceneLibrary(entries, nil, nil, CutsceneLibraryWords{})
	clickButton(a, menu.CutscenesButton, now)
	if a.Screen() != ScreenCutsceneLibrary {
		t.Fatalf("first game's Cut Scenes with nothing seen opens %v", a.Screen())
	}
}

// pageSeamsFake records what a save page asks its seams.
type pageSeamsFake struct {
	entries   []SaveEntry
	requests  []SaveRequest
	overwrite []bool
	listed    [][]string
	existing  map[string]bool
}

func (p *pageSeamsFake) seams() SaveDialogSeams {
	return SaveDialogSeams{
		Directory: "D",
		List: func(dir string) (SaveDirectory, error) {
			return SaveDirectory{Path: dir, Entries: append([]SaveEntry(nil), p.entries...)}, nil
		},
		Prepare: func(r SaveRequest) (PreparedSave, error) {
			p.requests = append(p.requests, r)
			out := PreparedSave{Paths: []string{r.Name}}
			if p.existing[r.Name] {
				out.Existing = []string{r.Name}
			}
			out.Commit = func(overwrite bool) ([]string, error) {
				p.overwrite = append(p.overwrite, overwrite)
				return out.Paths, nil
			}
			return out, nil
		},
	}
}

func testSavePage(p *pageSeamsFake) SavePage {
	return SavePage{
		Panel: image.Rect(100, 30, 600, 450), List: image.Rect(140, 177, 536, 177), Rows: 10,
		Message: image.Rect(140, 372, 560, 406), NameBox: image.Rect(260, 96, 560, 120),
		EmptySlot: "<<< Empty Slot >>>", Hidden: []string{"game9998.sav", "game9999.sav"},
		Slot: func(dir string, listed []string) string {
			p.listed = append(p.listed, listed)
			return "game0002.sav"
		},
	}
}

func openTestSavePage(t *testing.T, a *App) {
	t.Helper()
	a.flow.screen, a.flow.menuBack = ScreenGameMenu, ScreenMap
	a.flow.openSaveDialog()
	if a.Screen() != ScreenSave || a.flow.pages.save == nil {
		t.Fatalf("save opened %v with page %v", a.Screen(), a.flow.pages.save != nil)
	}
}

func TestSavePageSlotRowsAndOverwriteWithoutConfirmation(t *testing.T) {
	fake := &pageSeamsFake{
		entries: []SaveEntry{
			{Name: "game0001.sav", Label: "Alpha"},
			{Name: "game9999.sav", Label: "Restart"},
			{Name: "game0000.sav", Label: "Beta"},
		},
		existing: map[string]bool{"game0001.sav": true, "game0000.sav": true},
	}
	page := testSavePage(fake)
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetSaveDialogSeams(fake.seams())
	a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &page})

	openTestSavePage(t, a)
	var rows []string
	for _, r := range a.HeadlessRows() {
		rows = append(rows, r.Text)
	}
	if want := []string{"<<< Empty Slot >>>", "Alpha", "Beta"}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows %q, want %q (game9999.sav hidden)", rows, want)
	}
	v := a.flow.pages.save.view()
	if !a.flow.pageButtonDisabled(v, loadOKButton) {
		t.Fatal("Ok is enabled with an empty name")
	}

	// Row 0 with a new name writes the slot's file under the typed label.
	if err := a.HeadlessSaveEdit("", "Gamma", SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[0]; got.Name != "game0002.sav" || got.Label != "Gamma" || got.Directory != "D" {
		t.Fatalf("empty slot request %+v", got)
	}
	if !reflect.DeepEqual(fake.listed[0], []string{"game0001.sav", "game0000.sav"}) {
		t.Fatalf("slot rule saw %q", fake.listed[0])
	}
	if a.Screen() != ScreenGameMenu || fake.overwrite[0] {
		t.Fatalf("after the slot save: screen %v overwrite %v", a.Screen(), fake.overwrite)
	}

	// Choosing a row puts its label in the box; Ok overwrites its file at once.
	openTestSavePage(t, a)
	if err := a.HeadlessSaveSelect(2); err != nil {
		t.Fatal(err)
	}
	if h, _ := a.HeadlessSaveState(); h.Request.Label != "Beta" {
		t.Fatalf("row 2 put %q in the box", h.Request.Label)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[1]; got.Name != "game0000.sav" || got.Label != "Beta" || !fake.overwrite[1] {
		t.Fatalf("row request %+v overwrite %v", got, fake.overwrite)
	}

	// A typed name equal to a listed label selects that row.
	openTestSavePage(t, a)
	if err := a.HeadlessSaveEdit("", "Alpha", SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[2]; got.Name != "game0001.sav" || !fake.overwrite[2] {
		t.Fatalf("matched label request %+v overwrite %v", got, fake.overwrite)
	}

	// A renamed row keeps its own file under the new label.
	openTestSavePage(t, a)
	if err := a.HeadlessSaveSelect(1); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveEdit("", "Delta", SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[3]; got.Name != "game0001.sav" || got.Label != "Delta" {
		t.Fatalf("renamed row request %+v", got)
	}
	if len(fake.requests) != 4 || len(fake.overwrite) != 4 {
		t.Fatalf("%d requests and %d commits, want 4 each", len(fake.requests), len(fake.overwrite))
	}
}

func TestSavePageChosenRowBeatsAnEqualLabelAboveIt(t *testing.T) {
	fake := &pageSeamsFake{
		entries: []SaveEntry{
			{Name: "game0003.sav", Label: "autosave start mission 1"},
			{Name: "game0001.sav", Label: "autosave start mission 1"},
			{Name: "game0002.sav", Label: "Other"},
		},
		existing: map[string]bool{"game0003.sav": true, "game0001.sav": true, "game0002.sav": true},
	}
	page := testSavePage(fake)
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetSaveDialogSeams(fake.seams())
	a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &page})

	// The lower of the two equal rows is chosen: Ok writes its file.
	openTestSavePage(t, a)
	if err := a.HeadlessSaveSelect(2); err != nil {
		t.Fatal(err)
	}
	// The earlier lookup order took the first row with the label before the
	// chosen row; it names the other file, so this case tells the orders apart.
	s := a.flow.pages.save
	first := ""
	for _, e := range s.saves {
		if e.Label == string(s.name) {
			first = e.Name
			break
		}
	}
	if first != "game0003.sav" {
		t.Fatalf("the old lookup order names %q, want the upper row's file", first)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[0]; got.Name != "game0001.sav" || !fake.overwrite[0] {
		t.Fatalf("chosen lower row request %+v overwrite %v, want game0001.sav", got, fake.overwrite)
	}

	// With no row chosen, the first row with the typed label is the file.
	openTestSavePage(t, a)
	if err := a.HeadlessSaveEdit("", "autosave start mission 1", SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[1]; got.Name != "game0003.sav" {
		t.Fatalf("typed name request %+v, want the first row's file", got)
	}

	// A chosen row whose label differs from the typed name gives way to the match.
	openTestSavePage(t, a)
	if err := a.HeadlessSaveSelect(3); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveEdit("", "autosave start mission 1", SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if got := fake.requests[2]; got.Name != "game0003.sav" {
		t.Fatalf("retyped request %+v, want the matching row's file", got)
	}
}

func TestSavePageNameLimitAndOtherDirectory(t *testing.T) {
	fake := &pageSeamsFake{}
	page := testSavePage(fake)
	page.NameLimit = 4
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetSaveDialogSeams(fake.seams())
	a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &page})
	openTestSavePage(t, a)
	if err := a.HeadlessSaveEdit("elsewhere", "x", SaveSAV); err == nil {
		t.Fatal("the save page accepted another directory")
	}
	if err := a.HeadlessSaveEdit("", "abcdef", SaveSAV); err != nil {
		t.Fatal(err)
	}
	if h, _ := a.HeadlessSaveState(); h.Request.Label != "abcd" {
		t.Fatalf("name %q, want it cut at 4 characters", h.Request.Label)
	}
}

func TestLoadPageOKNeedsASave(t *testing.T) {
	fake := &pageSeamsFake{}
	load := testSavePage(fake)
	load.NameBox, load.EmptySlot, load.Slot = image.Rectangle{}, "", nil
	load.OKNeedsSave = true
	for _, saves := range [][]SaveEntry{nil, {{Name: "s", Label: "One"}}} {
		a := newTestApp(t, appRows(1), okLoader(t))
		a.SetSavePages(SavePages{Load: load})
		a.SetSaveSeams(nil, func() []SaveEntry { return saves }, nil)
		a.flow.openLoad(ScreenMenu)
		if got := a.flow.pageButtonDisabled(a.flow.loadView(), loadOKButton); got != (len(saves) == 0) {
			t.Errorf("%d saves: Ok disabled %v", len(saves), got)
		}
	}
	// Loss control: the first game's LOAD window keeps Ok enabled.
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetSaveSeams(nil, func() []SaveEntry { return nil }, nil)
	a.flow.openLoad(ScreenMenu)
	if a.flow.pageButtonDisabled(a.flow.loadView(), loadOKButton) {
		t.Fatal("the first game's Ok is disabled with no save")
	}
}
