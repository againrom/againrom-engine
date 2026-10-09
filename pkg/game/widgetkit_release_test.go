package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/render/text"
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
		rows[i] = ui.SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("slot-%02d", i)}
	}
	orig := OriginalStore{Dir: os.Getenv("AGAINROM_ORIGINAL_SAVES"), Selector: f.textSelector()}
	_, originalList, _ := f.SaveSeams(SaveStore{Dir: stateDir}, orig, nil)
	originalRows := originalList()
	if len(originalRows) == 0 || !IsOriginal(originalRows[0].Name) {
		t.Fatal("widget kit requires an original SAV row")
	}
	rows[13] = originalRows[0]

	load := f.App("widget kit load")
	t.Cleanup(load.StopAudio)
	load.SetCutscenes(nil)
	load.Layout(640, 480)
	load.SetSaveSeams(nil, func() []ui.SaveEntry { return rows }, nil)
	if err := load.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	for range 13 {
		if err := load.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	var loadPicture *image.RGBA
	loadGlyphs := text.Record(func() { loadPicture = shot(t, load, "load-game") })
	assertInstalledLoadDetailLabel(t, f, loadPicture, loadGlyphs, rows[13].Label)
	t.Logf("%s: selected original SAV %s; detail is label %q", lang, rows[13].Name, rows[13].Label)
	assertInstalledChooserWell(t, loadPicture, image.Rect(122, 152, 504, 152+10*(f.Font.Value().Height()+4)+2))
	if !holdsSprite(loadPicture, frames[barThumb]) {
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
	var optionsPicture *image.RGBA
	optionsGlyphs := text.Record(func() { optionsPicture = panel(t, "game-options") })
	assertInstalledOptionsContent(t, f, optionsGlyphs)
	if !holdsSprite(optionsPicture, frames[sliderKnob]) {
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
	var savePicture *image.RGBA
	saveGlyphs := text.Record(func() { savePicture = shot(t, app, "save-dialog") })
	assertInstalledChooserWell(t, savePicture, image.Rect(38, 88, 578, 88+7*(f.Font.Value().Height()+4)+2))
	assertInstalledDialogGlyphsInside(t, saveGlyphs, image.Rect(56, 420, 584, 464), 420)
	if !holdsSprite(savePicture, frames[barThumb]) {
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

func assertInstalledLoadDetailLabel(t *testing.T, f *FrontEnd, pic *image.RGBA, calls []text.DrawCall, label string) {
	t.Helper()
	encoded, err := encodeSaveLabel(label, f.textSelector())
	if err != nil {
		t.Fatal(err)
	}
	font := f.Font.Value()
	y := max(344, 152+10*(font.Height()+4)+4)
	expected := image.NewRGBA(pic.Bounds())
	want := text.Record(func() {
		for i, line := range ui.NoticeLines(font, encoded, 400) {
			if i >= 2 {
				break
			}
			font.Draw(expected, line, 122, y+i*(font.Height()+1), color.RGBA{242, 230, 196, 255})
		}
	})
	var detail []text.DrawCall
	for _, call := range calls {
		if call.Y == y || call.Y == y+font.Height()+1 {
			detail = append(detail, call)
		}
	}
	if len(want) == 0 || len(detail) != len(want) {
		t.Fatalf("LOAD detail drew %d glyphs, want %d label glyphs", len(detail), len(want))
	}
	for i, call := range detail {
		if call.Glyph != want[i].Glyph || call.X != want[i].X || call.Y != want[i].Y || call.Color != want[i].Color {
			t.Fatalf("LOAD detail glyph %d differs from the selected label", i)
		}
	}
	painted := 0
	for py := y; py < min(pic.Bounds().Max.Y, y+2*(font.Height()+1)); py++ {
		for x := 122; x < 522; x++ {
			if pixel := expected.RGBAAt(x, py); pixel.A == 255 {
				painted++
				if pic.RGBAAt(x, py) != pixel {
					t.Fatalf("LOAD label pixel differs at %d,%d", x, py)
				}
			}
		}
	}
	if painted < 20 {
		t.Fatal("LOAD label has no bounded pixel witness")
	}
}

func assertInstalledChooserWell(t *testing.T, pic *image.RGBA, list image.Rectangle) {
	t.Helper()
	r := list.Inset(-1)
	dark, light := color.RGBA{8, 8, 8, 255}, color.RGBA{94, 115, 101, 255}
	for x := r.Min.X + 1; x < r.Max.X-1; x++ {
		if pic.RGBAAt(x, r.Min.Y) != dark || pic.RGBAAt(x, r.Max.Y-1) != light {
			t.Fatalf("chooser lacks sunken horizontal edges at x%d", x)
		}
	}
	for y := r.Min.Y + 1; y < r.Max.Y-1; y++ {
		if pic.RGBAAt(r.Min.X, y) != dark {
			t.Fatalf("chooser lacks sunken left edge at y%d", y)
		}
	}
}

func assertInstalledDialogGlyphsInside(t *testing.T, calls []text.DrawCall, well image.Rectangle, minY int) {
	t.Helper()
	count := 0
	for _, call := range calls {
		if call.Y < minY || call.Glyph == nil || call.Glyph.Width == 0 {
			continue
		}
		for i, pixel := range call.Glyph.Pixels {
			at := image.Pt(call.X+i%call.Glyph.Width, call.Y+i/call.Glyph.Width)
			if !pixel.Painted || !call.Clip.Empty() && !at.In(call.Clip) {
				continue
			}
			count++
			if !at.In(well) {
				t.Fatalf("dialog glyph or shadow enters frame at %v outside %v", at, well)
			}
		}
	}
	if count == 0 {
		t.Fatal("dialog has no captured glyph pixels")
	}
}

func assertInstalledOptionsContent(t *testing.T, f *FrontEnd, calls []text.DrawCall) {
	t.Helper()
	assertInstalledDialogGlyphsInside(t, calls, image.Rect(92, 44, 540, 428), 0)
	content := image.Rectangle{}
	title, ok := LoadTextTable(f.Archives.Containers, DialogsTextPath).At(150)
	if !ok || title == "" {
		t.Fatal("installed options title is absent")
	}
	wantTitleX := 76 + (480-f.Font.Value().Advance(strings.ReplaceAll(title, "~", "")))/2
	titleFound := false
	for _, call := range calls {
		if call.Y == 48 && !call.Flat && !titleFound {
			titleFound = true
			if call.X != wantTitleX {
				t.Fatalf("options title starts at %d, want centred pen %d", call.X, wantTitleX)
			}
		}
		if call.Y >= 70 && call.Y < 400 && call.Clip.Dx() > 50 && call.Clip.Dx() < 480 {
			content = content.Union(call.Clip)
		}
	}
	if !titleFound || content.Empty() || content.Min.X-76 != 556-content.Max.X {
		t.Fatalf("options title=%t content=%v is not centred in the painted body", titleFound, content)
	}
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
