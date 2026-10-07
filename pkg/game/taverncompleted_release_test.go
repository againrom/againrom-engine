package game

import (
	"crypto/sha256"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseCompletedExtraTavernTalkerDisappearsThroughAppAndCurrentSAV(t *testing.T) {
	root := os.Getenv("AGAINROM_TAVERN_DONE_REVIEW_DIR")
	if root == "" {
		t.Skip("AGAINROM_TAVERN_DONE_REVIEW_DIR is not set")
	}
	rel, err := filepath.Rel(reviewDirRoot("a14-tavern-done-talker"), root)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatal("tavern outputs are outside the review directory", root, err)
	}
	root = filepath.Join(root, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"TMP", "TEMP", "TMPDIR"} {
		t.Setenv(key, root)
	}
	f := shopOrderFront(t)
	f.Options = OptionsStore{}
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Tavern Hero", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	if len(f.Carried) != 1 {
		t.Fatal("installed character construction did not create one hero")
	}
	for _, mission := range []int{10, 20, 30} {
		if _, accepted := f.Town.Won(mission); !accepted {
			t.Fatal("controlled prerequisite completion refused", mission)
		}
	}
	f.arriveInTown()
	seed := currentTownSave(t, f)
	f = shopOrderFront(t)
	f.Options = OptionsStore{}
	app := turtleOpenTownApp(t, f, seed, "chapter40.sav")
	t.Cleanup(app.StopAudio)
	if f.Town.Chapter() != 40 || f.Town.progress == nil || f.Town.Done(41) {
		t.Fatal("actual App LOAD did not create unfinished chapter40 extra41")
	}
	s := completedTavernEnter(t, f, app)
	index := completedTavernCell(t, s.TownSurface(), "NPC 90")
	turtleTap(t, app, turtleCellPoint(t, s.TownSurface(), index))
	if !s.TownSurface().Buttons[tavernButtonTalk].Enabled {
		t.Fatal("unfinished installed extra41 has grey Talk")
	}
	r := ui.TownSurfaceButtonRect(ui.TownSurfaceTavern, tavernButtonTalk)
	turtleTap(t, app, r.Min.Add(r.Max).Div(2))
	if path, ok := s.townTextPath(); !ok || s.room != roomTalk || path != "main/text/inn/npc/npc90m41.txt" {
		t.Fatal("pointer Talk did not open installed extra41 text", path)
	}
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != roomTavern || !reflect.DeepEqual(s.innQueue, []int{41}) {
		t.Fatal("installed extra41 did not queue after the conversation", s.room, s.innQueue)
	}
	if err := app.HeadlessKey("escape"); err != nil || s.room != roomSquare {
		t.Fatal("ordinary tavern Exit did not commit the offer", err)
	}
	s = completedTavernEnter(t, f, app)
	index = completedTavernCell(t, s.TownSurface(), "NPC 90")
	turtleTap(t, app, turtleCellPoint(t, s.TownSurface(), index))
	if !s.TownSurface().Buttons[tavernButtonTalk].Enabled || f.Town.progress.record(41) == nil {
		t.Fatal("accepted unfinished extra41 lost its active talker")
	}
	before := tavernLabels(s)
	if want := []string{"Mercenary 6", "Mercenary 14", "NPC 22", "NPC 90"}; !reflect.DeepEqual(before, want) {
		t.Fatal("installed chapter40 witness has a different initial roster", before, want)
	}
	completedTavernFrame(t, app, filepath.Join(root, "before-completion.png"))
	if side, accepted := f.Town.Won(41); !side || !accepted || !f.Town.Done(41) || f.Town.Chapter() != 40 || f.Town.progress.record(41) != nil {
		t.Fatal("controlled extra41 completion did not retain chapter40", side, accepted)
	}
	want := []string{"Mercenary 10", "Mercenary 6", "Mercenary 14", "NPC 22"}
	checkCompletedTavernApp(t, f, app, want)
	completedTavernFrame(t, app, filepath.Join(root, "after-completion.png"))
	store := SaveStore{Dir: filepath.Join(root, "current")}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	name := shopOrderF2Save(t, app, store)
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	g := shopOrderFront(t)
	g.Options = OptionsStore{}
	cold := turtleOpenTownApp(t, g, raw, "completed-extra.sav")
	t.Cleanup(cold.StopAudio)
	if !g.Town.Done(41) || g.Town.Chapter() != 40 || g.Town.progress.record(41) != nil {
		t.Fatal("cold App LOAD lost explicit extra41 completion")
	}
	completedTavernEnter(t, g, cold)
	checkCompletedTavernApp(t, g, cold, want)
	completedTavernFrame(t, cold, filepath.Join(root, "cold-load.png"))
	t.Logf("controlled completion of installed extra41: seed=%x current=%x file=%s", sha256.Sum256(seed), sha256.Sum256(raw), filepath.Join(store.Dir, name))
}

func completedTavernEnter(t *testing.T, f *FrontEnd, app *ui.App) *townScreen {
	t.Helper()
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	if tip := s.TownSurface().Tip; tip.Showing() {
		p := ui.TipPanelCloseRect(tip.Rect).Min.Add(image.Pt(2, 2))
		turtleTap(t, app, p)
	}
	if s.room != roomTavern {
		t.Fatal("App did not enter tavern", s.room)
	}
	return s
}

func completedTavernCell(t *testing.T, v ui.TownSurfaceView, semantic string) int {
	t.Helper()
	for i, cell := range v.Cells {
		if cell.Semantic == semantic {
			return i
		}
	}
	t.Fatal("installed tavern omitted the expected control", semantic)
	return -1
}

func checkCompletedTavernApp(t *testing.T, f *FrontEnd, app *ui.App, want []string) {
	t.Helper()
	s := f.TownScreen().(*townScreen)
	if got := tavernLabels(s); !reflect.DeepEqual(got, want) {
		t.Fatal("completed extra talker remains or unrelated controls changed", got, want)
	}
	beforeState, gold := f.Town.progress.projection(), f.Town.Gold()
	if err := app.HeadlessActivate("NPC 90"); err == nil || s.room != roomTavern {
		t.Fatal("headless activation found the completed extra talker")
	}
	main := completedTavernCell(t, s.TownSurface(), "NPC 22")
	turtleTap(t, app, turtleCellPoint(t, s.TownSurface(), main))
	if !s.TownSurface().Cells[main].Selected || !s.TownSurface().Buttons[tavernButtonTalk].Enabled ||
		len(s.innQueue) != 0 || f.Town.Gold() != gold || !reflect.DeepEqual(beforeState, f.Town.progress.projection()) {
		t.Fatal("selection of the preserved main talker changed state")
	}
}

func completedTavernFrame(t *testing.T, app *ui.App, path string) {
	t.Helper()
	pix, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(file, pix)
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		t.Fatal("tavern frame", err, closeErr)
	}
}
