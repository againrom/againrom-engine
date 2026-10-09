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
		loadBar, loadPitch := sharedListBar(f, image.Rect(122, 152, 504, 342), 10)
		manifest["load_initial_frames"] = assertInstalledChooserBar(t, initial, frames, loadBar, 0, len(rows))
		for range 13 {
			if err := app.HeadlessKey("down"); err != nil {
				t.Fatal(err)
			}
		}
		moved := frame(t, app, "load-selected13")
		manifest["load_selected_frames"] = assertInstalledChooserBar(t, moved, frames, loadBar, 13, len(rows))
		assertInstalledChooserRows(t, moved, 122, 152, 504, loadPitch, 9)
		manifest["load_selected_glyph_pixels"] = assertChooserGlyphs(t, f, moved, "slot-13", image.Pt(125, 152+9*loadPitch+2))
		assertChooserClipping(t, app, moved, loadBar)
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
		saveBar, savePitch := sharedListBar(f, image.Rect(38, 88, 578, 235), 7)
		t.Run("row-region", func(t *testing.T) {
			assertInstalledChooserRows(t, initial, 38, 88, 578, savePitch, 0)
			tile := f.gameMenuArt().Pieces[0]
			for row := range 6 {
				x, y := 550, 88+savePitch*row+savePitch/2
				want := image.NewRGBA(image.Rect(0, 0, 1, 1))
				want.SetRGBA(0, 0, tile.RGBAAt((x-24)%tile.Bounds().Dx(), (y-16)%tile.Bounds().Dy()))
				if row == 0 {
					draw.Draw(want, want.Bounds(), image.NewUniform(color.RGBA{0, 7, 6, 220}), image.Point{}, draw.Over)
				}
				if initial.RGBAAt(x, y) != want.RGBAAt(0, 0) {
					t.Fatal("Save row lost installed panel texture", row)
				}
			}
		})
		t.Run("bar-region", func(t *testing.T) {
			manifest["save_initial_frames"] = assertInstalledChooserBar(t, initial, frames, saveBar, 0, len(visibleRows))
		})
		if err := app.HeadlessSaveSelect(6); err != nil {
			t.Fatal(err)
		}
		moved := frame(t, app, "save-selected6")
		t.Run("selected-bar-region", func(t *testing.T) {
			manifest["save_selected_frames"] = assertInstalledChooserBar(t, moved, frames, saveBar, 6, len(visibleRows))
		})
		t.Run("selected-font", func(t *testing.T) {
			manifest["save_selected_glyph_pixels"] = assertChooserGlyphs(t, f, moved, "slot-06", image.Pt(41, 88+6*savePitch+2))
		})
		assertChooserClipping(t, app, moved, saveBar)
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
		assertInstalledChooserRows(t, empty, 38, 88, 578, savePitch, 0)
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

// sharedListBar is the shared list's bar for a list built on arg with rows
// visible rows: pitch font height plus 4 and bottom top+rows*pitch+2
// (MENU-121), the bar one frame wide at the list's right edge.
func sharedListBar(f *FrontEnd, arg image.Rectangle, rows int) (image.Rectangle, int) {
	pitch := f.Font.Value().Height() + 4
	return image.Rect(arg.Max.X, arg.Min.Y, arg.Max.X+24, arg.Min.Y+rows*pitch+2), pitch
}

// assertInstalledChooserRows checks the selected visible row's outline
// corners in the list's selection edge colour.
func assertInstalledChooserRows(t *testing.T, pic *image.RGBA, left, top, right, pitch, row int) {
	t.Helper()
	y0, y1 := top+pitch*row, top+pitch*(row+1)-1
	for _, p := range []image.Point{{left, y0}, {right - 1, y0}, {left, y1}, {right - 1, y1}} {
		if got := pic.RGBAAt(p.X, p.Y); got != (color.RGBA{57, 77, 65, 255}) {
			t.Fatalf("selected chooser row border at%v =%v", p, got)
		}
	}
}

// assertInstalledChooserBar compares the bar with the shared bar's
// composition (MENU-117, MENU-119) at pos of count: top frame 18, track
// frame 19 tiles, bottom frame 20 and the fixed thumb frame 22 at T+W+q-4
// with q=trunc(pos*(H-3W+8)/(count-1)). It compares only opaque frame pixels
// no later part or its (+4,+4) shadow covers.
func assertInstalledChooserBar(t *testing.T, pic *image.RGBA, frames []*image.RGBA, bar image.Rectangle, pos, count int) map[int]int {
	t.Helper()
	top, w, h := bar.Min.Y, bar.Dx(), bar.Dy()
	q := 0
	if count >= 2 {
		q = pos * (h - 3*w + 8) / (count - 1)
	}
	thumbY := top + w + q - 4
	shadowed := image.Rect(bar.Min.X, thumbY, bar.Max.X+4, thumbY+28)
	counts := map[int]int{}
	for y := bar.Min.Y; y < bar.Max.Y; y++ {
		for x := bar.Min.X; x < bar.Max.X; x++ {
			index, sy := 19, (y-top-w)%24
			switch {
			case y >= thumbY && y < thumbY+24:
				index, sy = 22, y-thumbY
			case y < top+w:
				index, sy = 18, y-top
			case y >= bar.Max.Y-w:
				index, sy = 20, y-(bar.Max.Y-w)
			}
			if index != 22 && index != 18 && image.Pt(x, y).In(shadowed) {
				continue
			}
			want := frames[index].RGBAAt(x-bar.Min.X, sy)
			if want.A != 255 {
				continue
			}
			if got := pic.RGBAAt(x, y); got != want {
				t.Fatalf("installed frame%d source%d,%d at%d,%d =%v want%v", index, x-bar.Min.X, sy, x, y, got, want)
			}
			counts[index]++
		}
	}
	for _, index := range []int{18, 19, 20, 22} {
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

func assertChooserClipping(t *testing.T, app *ui.App, skin *image.RGBA, bar image.Rectangle) {
	t.Helper()
	// Each bar part casts its shadow 4 pixels right and down (MENU-117).
	strip := image.Rect(bar.Min.X, bar.Min.Y, bar.Max.X+4, bar.Max.Y+4)
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
