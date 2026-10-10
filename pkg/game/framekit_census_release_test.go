package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// TestReleaseFrameAndButtonCensus composes every installed screen and state
// this build can reach headlessly that draws a window frame, a list or a push
// button, and hashes each picture. With AGAINROM_FRAME_CENSUS_DIR set it
// writes one PNG per picture and a hash list named by the root's language,
// so two builds can be compared picture by picture.
func TestReleaseFrameAndButtonCensus(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	stateDir := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(stateDir, "options.txt")}
	root, err := editorPhysicalDirectory(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	lang := strings.ToLower(filepath.Base(root))
	out := os.Getenv("AGAINROM_FRAME_CENSUS_DIR")
	if out != "" {
		rel, err := filepath.Rel(root, filepath.Clean(out))
		if !filepath.IsAbs(out) || err == nil && !strings.HasPrefix(rel, "..") {
			t.Fatal("census output must be absolute and outside the installed root")
		}
	}
	hashes := map[string]string{}
	record := func(t *testing.T, name string, pic *image.RGBA) {
		t.Helper()
		if pic == nil || pic.Bounds().Empty() {
			t.Fatal("census picture is empty:", name)
		}
		if _, dup := hashes[name]; dup {
			t.Fatal("census name repeated:", name)
		}
		hashes[name] = fmt.Sprintf("%x", sha256.Sum256(pic.Pix))
		if out != "" {
			writeWidgetKitPNG(t, filepath.Join(out, lang, name+".png"), pic)
		}
	}
	frameOf := func(t *testing.T, app *ui.App, name string) {
		t.Helper()
		pic, note, err := app.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatal(name, note, err)
		}
		record(t, name, pic)
	}

	menu := f.App("census menu")
	t.Cleanup(menu.StopAudio)
	menu.Layout(640, 480)
	frameOf(t, menu, "main-menu")

	rows := make([]ui.SaveEntry, 27)
	for i := range rows {
		rows[i] = ui.SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("slot-%02d", i)}
	}
	load := f.App("census load")
	t.Cleanup(load.StopAudio)
	load.SetCutscenes(nil)
	load.Layout(640, 480)
	load.SetSaveSeams(nil, func() []ui.SaveEntry { return rows }, nil)
	if err := load.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	frameOf(t, load, "load-game-top")
	for range 13 {
		if err := load.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	frameOf(t, load, "load-game-row13")
	empty := f.App("census load empty")
	t.Cleanup(empty.StopAudio)
	empty.SetCutscenes(nil)
	empty.Layout(640, 480)
	empty.SetSaveSeams(nil, func() []ui.SaveEntry { return nil }, nil)
	if err := empty.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	frameOf(t, empty, "load-game-empty")

	for _, key := range []string{"intro", "newgame"} {
		if err := f.Options.EncounterCutscene(key); err != nil {
			t.Fatal(err)
		}
	}
	library := f.App("census library")
	t.Cleanup(library.StopAudio)
	library.Layout(640, 480)
	if err := library.HeadlessActivate("cutscenes"); err != nil {
		t.Fatal(err)
	}
	frameOf(t, library, "cutscene-library")
	if err := library.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	frameOf(t, library, "cutscene-library-row1")

	c := ui.NewChargen(f.ChargenSetup())
	record(t, "chargen-precreate-tip", ui.ComposeChargenFrame(c))
	c.CloseTip()
	record(t, "chargen-precreate", ui.ComposeChargenFrame(c))
	c.Forward()
	record(t, "chargen-detailed", ui.ComposeChargenFrame(c))

	for _, room := range []string{"town-square", "tavern", "school", "shop"} {
		v := releaseRoomTipRaster(t, room)
		record(t, "room-tip-"+room, v.with)
		record(t, "room-"+room, v.without)
	}

	shopTalk, _ := openFirstTownShopDialogue(t, f, "census dialogue")
	t.Cleanup(shopTalk.StopAudio)
	record(t, "dialogue-shop", dialogueFrame(t, shopTalk))

	app := f.App("census mission")
	t.Cleanup(app.StopAudio)
	app.Layout(640, 480)
	if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	app.SetSaveDialogSeams(ui.SaveDialogSeams{Directory: stateDir,
		List: func(directory string) (ui.SaveDirectory, error) {
			return ui.SaveDirectory{Path: directory, Entries: rows[:12]}, nil
		},
		Prepare: func(ui.SaveRequest) (ui.PreparedSave, error) {
			return ui.PreparedSave{}, fmt.Errorf("census writes no save")
		},
	})
	before := f.live.world.Hash()
	for _, part := range []struct {
		name string
		get  func() (*image.RGBA, error)
	}{
		{"mission-command-panel", app.HeadlessCommandPanel},
		{"mission-bottom-hud", app.HeadlessBottomHUD},
		{"mission-minimap", app.HeadlessMinimap},
		{"mission-inventory-pack", app.HeadlessInventoryPack},
		{"mission-card", app.HeadlessMissionCard},
	} {
		pic, err := part.get()
		if err != nil {
			t.Fatal(part.name, err)
		}
		record(t, part.name, pic)
	}
	pane, _, err := app.HeadlessCharacterPane()
	if err != nil {
		t.Fatal(err)
	}
	record(t, "mission-character-pane", pane)

	panel := func(t *testing.T, name string) {
		t.Helper()
		pic := app.GameMenuPanel()
		if pic == nil {
			t.Fatal("no menu panel for", name)
		}
		record(t, name, pic)
	}
	openMissionGameMenu(t, app)
	panel(t, "game-menu")
	if err := app.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	panel(t, "game-menu-focus1")
	if err := app.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	panel(t, "game-options")
	if err := app.HeadlessGameMenuAction("options-cancel"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	panel(t, "sound-options")
	if err := app.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	openMissionGameMenu(t, app)
	if err := app.HeadlessGameMenuAction("objectives"); err != nil {
		t.Fatal(err)
	}
	panel(t, "quest-objectives")
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	openMissionGameMenu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("Save dialog", app.Screen(), err)
	}
	frameOf(t, app, "save-dialog")
	if err := app.HeadlessSaveAction("cancel"); err != nil {
		t.Fatal(err)
	}
	f.live.view.SetNotice(f.Words.MissionWon, ui.NoticeSuccess)
	notice, ok := f.live.view.HeadlessNoticePicture()
	if !ok {
		t.Fatal("the notice drew nothing")
	}
	record(t, "notice-success", notice)
	if f.live.world.Hash() != before {
		t.Fatal("drawing the census changed World")
	}

	names := make([]string, 0, len(hashes))
	for name := range hashes {
		names = append(names, name)
	}
	sort.Strings(names)
	var list strings.Builder
	for _, name := range names {
		fmt.Fprintf(&list, "%s %s\n", hashes[name], name)
	}
	t.Logf("%s: %d census pictures\n%s", lang, len(names), list.String())
	if out != "" {
		if err := os.WriteFile(filepath.Join(out, lang+"-hashes.txt"), []byte(list.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
