package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseSharedDialogFrames1185(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	art := f.gameMenuArt()
	if art == nil || art != f.gameMenuArt() {
		t.Fatal("shared frame not cached")
	}
	for i, p := range art.Pieces {
		if p == nil || p.Bounds().Empty() {
			t.Fatal("missing window piece", i)
		}
	}
	if art.Portrait == nil || art.Portrait.Bounds().Size() != image.Pt(88, 108) {
		t.Fatal("missing installed portrait border")
	}
	if art.Minimap == nil || art.Minimap.Bounds().Size() != image.Pt(160, 158) {
		t.Fatal("missing installed crystal")
	}
	a := f.App("framed load")
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "sample.ags", Label: "Sample save"}} }, nil)
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	pic, note, err := a.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal(note, err)
	}
	checkCorner := func(pic *image.RGBA, at image.Point) {
		checked := 0
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				c := art.Pieces[1].RGBAAt(x, y)
				if c.A == 255 {
					checked++
					if got := pic.RGBAAt(at.X+x, at.Y+y); got != c {
						t.Fatalf("window corner %v: %v want %v", at.Add(image.Pt(x, y)), got, c)
					}
				}
			}
		}
		if checked < 100 {
			t.Fatal("vacuous corner witness")
		}
	}
	checkCorner(pic, image.Pt(80, 56))
	l := f.Words.OnLayout(ui.AuthoredDialogueLayout())
	l.Frame = art
	dialogue := ui.RenderNotice(l, f.Font.Value(), "A conversation.", nil)
	checkCorner(dialogue, image.Point{})
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(a, store, OriginalStore{}, nil)
	releaseLoadScroll(t, f)
	t.Log(fmt.Sprintf("frame=%v portrait=%v crystal=%v; LOAD and dialogue use the same installed corner", art.Pieces[0].Bounds(), art.Portrait.Bounds(), art.Minimap.Bounds()))
	witnessSaveChooserStyle(t, f)
}

func witnessSaveChooserStyle(t *testing.T, f *FrontEnd) {
	t.Helper()
	output := os.Getenv("AGAINROM_SAVE_CHOOSER_STYLE_WITNESS_DIR")
	root, err := editorPhysicalDirectory(f.Archives.Root)
	if err != nil || !filepath.IsAbs(f.Archives.Root) || !filepath.IsAbs(root) || !filepath.IsAbs(output) {
		t.Fatal("absolute asset root and AGAINROM_SAVE_CHOOSER_STYLE_WITNESS_DIR required", err)
	}
	rel, err := filepath.Rel(root, filepath.Clean(output))
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		t.Fatal("Save style output is inside the installed asset root")
	}
	rootHash := fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.ToSlash(root))))
	dir := filepath.Join(output, rootHash, t.Name())
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	stateDir, err := os.MkdirTemp(dir, "state-")
	if err != nil {
		t.Fatal(err)
	}
	const source = graphicsPrefix + "interface/scrlbars.256"
	raw, err := f.Archives.Containers.ReadFile(source)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != "8451c93d1a3bd772ab0e012bb4e3606cbbb7cb950184ad9ad40f631034fffdfe" {
		t.Fatal("installed scroll source qualification", err)
	}
	frames, err := decodeCursor256Frames(source, raw)
	if err != nil || len(frames) != 26 {
		t.Fatal("installed scroll frames", err)
	}
	for _, index := range []int{16, 18, 19, 20} {
		if frames[index] == nil || frames[index].Bounds() != image.Rect(0, 0, 24, 24) {
			t.Fatal("installed scroll dimensions", index)
		}
	}
	f.Options = OptionsStore{Path: filepath.Join(stateDir, "options.txt")}
	f.SetDeterministicFrames(true)
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
	rows := make([]ui.SaveEntry, 27)
	for i := range rows {
		rows[i] = ui.SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("slot-%02d", i), Note: fmt.Sprintf("detail-%02d", i)}
	}
	manifest := map[string]any{"asset_root_sha256": rootHash, "scroll_sha256": fmt.Sprintf("%x", sha256.Sum256(raw))}
	frame := func(t *testing.T, app *ui.App, name string) *image.RGBA {
		t.Helper()
		pic, note, err := app.HeadlessFrame()
		if err != nil || note != "" || pic.Bounds() != image.Rect(0, 0, 640, 480) {
			t.Fatal("ordinary App chooser frame", note, err)
		}
		file, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr, closeErr := png.Encode(file, pic), file.Close()
		if encodeErr != nil || closeErr != nil {
			t.Fatal(encodeErr, closeErr)
		}
		manifest[name+"_pixels_sha256"] = fmt.Sprintf("%x", sha256.Sum256(pic.Pix))
		return pic
	}
	t.Run("load-style-control", func(t *testing.T) {
		app := f.App("installed Load control")
		t.Cleanup(app.StopAudio)
		app.Layout(640, 480)
		loaded := ""
		app.SetSaveSeams(nil, func() []ui.SaveEntry { return rows }, func(name string) (ui.MapOpener, bool, error) {
			loaded = name
			return nil, false, fmt.Errorf("synthetic target control")
		})
		if err := app.HeadlessKey("load"); err != nil {
			t.Fatal(err)
		}
		initial := frame(t, app, "load-initial")
		manifest["load_initial_frames"] = assertInstalledChooserBar(t, initial, frames, image.Rect(504, 152, 528, 342), 176, 318, 176)
		for range 13 {
			if err := app.HeadlessKey("down"); err != nil {
				t.Fatal(err)
			}
		}
		moved := frame(t, app, "load-selected13")
		manifest["load_selected_frames"] = assertInstalledChooserBar(t, moved, frames, image.Rect(504, 152, 528, 342), 176, 318, 235)
		assertInstalledChooserRows(t, moved, 122, 152, 502, 19, 10)
		manifest["load_selected_glyph_pixels"] = assertChooserGlyphs(t, f, moved, "slot-13", image.Pt(125, 325))
		assertChooserClipping(t, app, moved, image.Rect(504, 152, 528, 342))
		if err := app.HeadlessKey("enter"); err != nil || loaded != "slot-13.sav" {
			t.Fatal("Load exact token changed", loaded, err)
		}
	})
	t.Run("save-load-style", func(t *testing.T) {
		app := f.App("installed Save style")
		t.Cleanup(app.StopAudio)
		app.Layout(640, 480)
		if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
			t.Fatal(err)
		}
		before := f.live.world.Hash()
		prepared := ""
		visibleRows := rows[:7]
		app.SetSaveSeams(func(bool) (string, error) { return "", fmt.Errorf("legacy save must not run") }, nil, nil)
		app.SetSaveDialogSeams(ui.SaveDialogSeams{Directory: stateDir,
			List: func(directory string) (ui.SaveDirectory, error) {
				return ui.SaveDirectory{Path: directory, Entries: visibleRows}, nil
			},
			Prepare: func(r ui.SaveRequest) (ui.PreparedSave, error) {
				prepared = r.Name
				return ui.PreparedSave{}, fmt.Errorf("synthetic target control")
			},
		})
		openMissionGameMenu(t, app)
		if err := app.HeadlessGameMenuAction("save"); err != nil || app.Screen() != ui.ScreenSave {
			t.Fatal("ordinary App Save opening", app.Screen(), err)
		}
		initial := frame(t, app, "save-initial")
		t.Run("row-region", func(t *testing.T) {
			assertInstalledChooserRows(t, initial, 40, 91, 577, 23, 6)
			tile := f.gameMenuArt().Pieces[0]
			for row := range 6 {
				x, y := 550, 101+23*row
				want := image.NewRGBA(image.Rect(0, 0, 1, 1))
				want.SetRGBA(0, 0, tile.RGBAAt((x-24)%tile.Bounds().Dx(), (y-16)%tile.Bounds().Dy()))
				overlay := color.RGBA{0, 0, 0, 45}
				if row == 0 {
					overlay = color.RGBA{0, 7, 6, 220}
				}
				draw.Draw(want, want.Bounds(), image.NewUniform(overlay), image.Point{}, draw.Over)
				if initial.RGBAAt(x, y) != want.RGBAAt(0, 0) {
					t.Fatal("Save row lost installed panel texture", row)
				}
			}
		})
		t.Run("bar-region", func(t *testing.T) {
			manifest["save_initial_frames"] = assertInstalledChooserBar(t, initial, frames, image.Rect(580, 88, 602, 235), 112, 211, 112)
		})
		if err := app.HeadlessSaveSelect(6); err != nil {
			t.Fatal(err)
		}
		moved := frame(t, app, "save-selected6")
		t.Run("selected-bar-region", func(t *testing.T) {
			manifest["save_selected_frames"] = assertInstalledChooserBar(t, moved, frames, image.Rect(580, 88, 602, 235), 112, 211, 187)
		})
		t.Run("selected-font", func(t *testing.T) {
			manifest["save_selected_glyph_pixels"] = assertChooserGlyphs(t, f, moved, "slot-06", image.Pt(44, 209))
		})
		assertChooserClipping(t, app, moved, image.Rect(580, 88, 602, 235))
		state, ok := app.HeadlessSaveState()
		if !ok || state.Request.Name != "slot-06" || prepared != "" {
			t.Fatal("Save style changed the selected token or prepared while drawing", state, prepared)
		}
		for range 3 {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		if f.live.world.Hash() != before {
			t.Fatal("Save chooser paint or idle ticks changed World")
		}
		if err := app.HeadlessSaveAction("save"); err == nil || prepared != "slot-06" {
			t.Fatal("Save exact selected target changed", prepared, err)
		}
		manifest["save_selected_token"] = prepared + ".sav"
		manifest["world_hash_unchanged_after_three_ticks"] = true
		if err := app.HeadlessSaveAction("cancel"); err != nil {
			t.Fatal(err)
		}
		visibleRows = rows[:1]
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		empty := frame(t, app, "save-empty-rows")
		assertInstalledChooserRows(t, empty, 40, 91, 577, 23, 6)
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, 100, 220); err != nil {
				t.Fatal(err)
			}
		}
		if state, _ := app.HeadlessSaveState(); state.Request.Name != "save" {
			t.Fatal("empty styled row activated a target", state)
		}
	})
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil || os.WriteFile(filepath.Join(dir, "manifest.json"), body, 0600) != nil {
		t.Fatal("write private chooser manifest", err)
	}
}

func assertInstalledChooserRows(t *testing.T, pic *image.RGBA, left, top, right, pitch, count int) {
	t.Helper()
	for row := range count {
		for _, p := range []image.Point{{left, top + pitch*row}, {right - 1, top + pitch*row}, {left, top + pitch*(row+1) - 2}, {right - 1, top + pitch*(row+1) - 2}} {
			if got := pic.RGBAAt(p.X, p.Y); got != (color.RGBA{57, 77, 65, 255}) {
				t.Fatalf("literal chooser row border at%v =%v", p, got)
			}
		}
	}
}

func assertInstalledChooserBar(t *testing.T, pic *image.RGBA, frames []*image.RGBA, strip image.Rectangle, trackY, downY, thumbY int) map[int]int {
	t.Helper()
	counts := map[int]int{}
	for y := strip.Min.Y; y < strip.Max.Y; y++ {
		for x := strip.Min.X; x < strip.Max.X; x++ {
			index, sy := 19, (y-trackY)%24
			switch {
			case y < trackY:
				index, sy = 18, y-strip.Min.Y
			case y >= downY:
				index, sy = 20, y-downY
			case y >= thumbY && y < thumbY+24:
				index, sy = 16, y-thumbY
			}
			sx := (x - strip.Min.X) * 24 / strip.Dx()
			want := frames[index].RGBAAt(sx, sy)
			if want.A != 255 {
				continue
			}
			if got := pic.RGBAAt(x, y); got != want {
				t.Fatalf("installed frame%d source%d,%d at%d,%d =%v want%v", index, sx, sy, x, y, got, want)
			}
			counts[index]++
		}
	}
	for _, index := range []int{16, 18, 19, 20} {
		if counts[index] < 100 {
			t.Fatal("unqualified installed frame population", index, counts)
		}
	}
	return counts
}

func assertChooserGlyphs(t *testing.T, f *FrontEnd, pic *image.RGBA, literal string, at image.Point) int {
	t.Helper()
	font := f.Font.Value()
	expected := map[image.Point]color.RGBA{}
	pen := at.X
	for i := range len(literal) {
		g := &font.Glyphs[int(literal[i])-32]
		for y := range g.Height {
			for x := range g.Width {
				p := g.Pixels[y*g.Width+x]
				if p.Painted {
					expected[image.Pt(pen+x, at.Y+y)] = color.RGBA{uint8(218 * int(p.Level) / 15), uint8(183 * int(p.Level) / 15), uint8(71 * int(p.Level) / 15), 255}
				}
			}
		}
		pen += g.Advance + font.Spacing
	}
	if len(expected) < 20 {
		t.Fatal("vacuous selected installed glyph oracle")
	}
	for p, want := range expected {
		if got := pic.RGBAAt(p.X, p.Y); got != want {
			t.Fatalf("selected literal %q glyph at%v =%v want%v", literal, p, got, want)
		}
	}
	return len(expected)
}

func assertChooserClipping(t *testing.T, app *ui.App, skin *image.RGBA, strip image.Rectangle) {
	t.Helper()
	app.SetCutsceneScrollArt(nil)
	fallback, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal("missing-art control", note, err)
	}
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if !image.Pt(x, y).In(strip) && !bytes.Equal(skin.Pix[skin.PixOffset(x, y):skin.PixOffset(x, y)+4], fallback.Pix[fallback.PixOffset(x, y):fallback.PixOffset(x, y)+4]) {
				t.Fatalf("skin leaked outside literal bar at%d,%d", x, y)
			}
		}
	}
}
