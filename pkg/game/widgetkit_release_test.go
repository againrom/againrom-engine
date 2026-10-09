package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// TestReleaseWidgetKitScreens renders the screens the shared widget kit
// draws on an installed root: Load Game, the in-game menu, Game Options,
// Sound Options, the cutscene library, the Save dialog and an outcome
// notice. The installed scrlbars.256 knob and thumb frames must stand intact
// on the slider and bar screens. With AGAINROM_WIDGET_KIT_WITNESS_DIR set,
// each frame is written there as a PNG named by the root's language.
func TestReleaseWidgetKitScreens(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	stateDir := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(stateDir, "options.txt")}
	root, err := editorPhysicalDirectory(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("AGAINROM_WIDGET_KIT_WITNESS_DIR")
	if out == "" {
		out = t.TempDir()
	}
	rel, err := filepath.Rel(root, filepath.Clean(out))
	if !filepath.IsAbs(out) || err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		t.Fatal("widget kit output must be absolute and outside the installed root")
	}
	lang := strings.ToLower(filepath.Base(root))
	const source = graphicsPrefix + "interface/scrlbars.256"
	raw, err := f.Archives.Containers.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := decodeCursor256Frames(source, raw)
	if err != nil || len(frames) != 26 {
		t.Fatal("installed scroll frames", len(frames), err)
	}
	const sliderKnob, barThumb = 10, 22
	shot := func(t *testing.T, app *ui.App, name string) *image.RGBA {
		t.Helper()
		pic, note, err := app.HeadlessFrame()
		if err != nil || note != "" || pic.Bounds() != image.Rect(0, 0, 640, 480) {
			t.Fatal(name, note, err)
		}
		writeWidgetKitPNG(t, filepath.Join(out, fmt.Sprintf("%s-%s.png", lang, name)), pic)
		return pic
	}
	rows := make([]ui.SaveEntry, 27)
	for i := range rows {
		rows[i] = ui.SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("slot-%02d", i), Note: fmt.Sprintf("detail-%02d", i)}
	}

	load := f.App("widget kit load")
	t.Cleanup(load.StopAudio)
	load.SetCutscenes(nil)
	load.Layout(640, 480)
	load.SetSaveSeams(nil, func() []ui.SaveEntry { return rows }, nil)
	if err := load.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if !holdsSprite(shot(t, load, "load-game"), frames[barThumb]) {
		t.Fatal("Load Game lost the installed bar thumb")
	}

	for _, key := range []string{"intro", "newgame"} {
		if err := f.Options.EncounterCutscene(key); err != nil {
			t.Fatal(err)
		}
	}
	library := f.App("widget kit library")
	t.Cleanup(library.StopAudio)
	library.Layout(640, 480)
	if err := library.HeadlessActivate("cutscenes"); err != nil {
		t.Fatal(err)
	}
	if len(library.HeadlessRows()) != 2 {
		t.Fatal("cutscene library rows", library.HeadlessRows())
	}
	shot(t, library, "cutscene-library")

	app := f.App("widget kit mission")
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
			return ui.PreparedSave{}, fmt.Errorf("witness writes no save")
		},
	})
	before := f.live.world.Hash()
	openMissionGameMenu(t, app)
	// The map's menu pages draw onto the live canvas; the panel is the
	// composed picture the frame shows over the map.
	panel := func(t *testing.T, name string) *image.RGBA {
		t.Helper()
		pic := app.GameMenuPanel()
		if pic == nil {
			t.Fatal("no menu panel for", name)
		}
		writeWidgetKitPNG(t, filepath.Join(out, fmt.Sprintf("%s-%s.png", lang, name)), pic)
		return pic
	}
	panel(t, "game-menu")
	if err := app.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if !holdsSprite(panel(t, "game-options"), frames[sliderKnob]) {
		t.Fatal("Game Options lost the installed slider knob")
	}
	if err := app.HeadlessGameMenuAction("options-cancel"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	if !holdsSprite(panel(t, "sound-options"), frames[sliderKnob]) {
		t.Fatal("Sound Options lost the installed slider knob")
	}
	if err := app.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	openMissionGameMenu(t, app)
	if err := app.HeadlessGameMenuAction("save"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("Save dialog", app.Screen(), err)
	}
	if !holdsSprite(shot(t, app, "save-dialog"), frames[barThumb]) {
		t.Fatal("the Save dialog lost the installed bar thumb")
	}
	if err := app.HeadlessSaveAction("cancel"); err != nil {
		t.Fatal(err)
	}
	f.live.view.SetNotice(f.Words.MissionWon, ui.NoticeSuccess)
	notice, ok := f.live.view.HeadlessNoticePicture()
	if !ok {
		t.Fatal("the notice drew nothing")
	}
	writeWidgetKitPNG(t, filepath.Join(out, lang+"-notice.png"), notice)
	if f.live.world.Hash() != before {
		t.Fatal("drawing the widget screens changed World")
	}
	t.Logf("%s: seven widget screens written to %s; scroll source sha256 %x", lang, out, sha256.Sum256(raw))
}

// holdsSprite reports whether every opaque pixel of sprite stands unchanged
// at one position of pic.
func holdsSprite(pic, sprite *image.RGBA) bool {
	b := sprite.Bounds()
	var opaque []image.Point
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if sprite.RGBAAt(x, y).A == 255 {
				opaque = append(opaque, image.Pt(x-b.Min.X, y-b.Min.Y))
			}
		}
	}
	if len(opaque) < 100 {
		return false
	}
	for y := 0; y+b.Dy() <= pic.Bounds().Dy(); y++ {
		for x := 0; x+b.Dx() <= pic.Bounds().Dx(); x++ {
			found := true
			for _, p := range opaque {
				if pic.RGBAAt(x+p.X, y+p.Y) != sprite.RGBAAt(b.Min.X+p.X, b.Min.Y+p.Y) {
					found = false
					break
				}
			}
			if found {
				return true
			}
		}
	}
	return false
}

func writeWidgetKitPNG(t *testing.T, path string, pic image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encodeErr, closeErr := png.Encode(file, pic), file.Close()
	if encodeErr != nil || closeErr != nil {
		t.Fatal(encodeErr, closeErr)
	}
}
