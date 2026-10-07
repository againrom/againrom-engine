package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestReleaseShiftF4RevealsMapWithoutQuickSave(t *testing.T) {
	output := os.Getenv("AGAINROM_MAP_REVEAL_WITNESS_DIR")
	if output == "" {
		t.Skip("AGAINROM_MAP_REVEAL_WITNESS_DIR is not set")
	}
	f := releaseFront(t)
	root, err := filepath.EvalSymlinks(f.Archives.Root)
	if err != nil || !filepath.IsAbs(root) || !filepath.IsAbs(output) {
		t.Fatal("absolute resolved asset and witness roots required", err)
	}
	rel, err := filepath.Rel(root, filepath.Clean(output))
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		t.Fatal("map reveal output is inside the installed asset root", output)
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
	f.SetDeterministicFrames(true)
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
	f.Options = OptionsStore{Path: filepath.Join(stateDir, "options.txt")}
	if err := f.Options.setTimedAutosave(ui.TimedAutosaveSettings{Enabled: true, Minutes: 1}); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: filepath.Join(stateDir, "saves")}
	app := f.App("map reveal")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	now := time.Unix(100, 0)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, func() time.Time { return now })
	if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if app.HeadlessNoticeOpen() || app.Screen() != ui.ScreenMap || f.live == nil || !f.live.stopped {
		t.Fatal("mission did not settle on a stopped map")
	}
	viewer := f.live.view
	before := missionAutosaveSampleNow(t, f)
	visible, explored := bytes.Clone(f.live.fog.visible), bytes.Clone(f.live.fog.explored)
	files := mapRevealFiles(t, store.Dir)
	off, err := app.HeadlessMinimap()
	if err != nil || viewer.FogRevealed() {
		t.Fatal("initial production minimap", err)
	}
	for edge := range 2 {
		if err := app.HeadlessKey("shift-f4"); err != nil {
			t.Fatal("ordinary App map chord", err)
		}
		if viewer.FogRevealed() != (edge == 0) {
			t.Fatalf("edge%d reveal=%v, want %v", edge+1, viewer.FogRevealed(), edge == 0)
		}
		if app.Screen() != ui.ScreenMap || f.live.view != viewer || !reflect.DeepEqual(before, missionAutosaveSampleNow(t, f)) ||
			!bytes.Equal(visible, f.live.fog.visible) || !bytes.Equal(explored, f.live.fog.explored) || !reflect.DeepEqual(files, mapRevealFiles(t, store.Dir)) {
			t.Fatal("display chord changed World, fog, view, SAV or ownership sidecars")
		}
		if !viewer.FogRevealed() {
			restored, err := app.HeadlessMinimap()
			if err != nil || restored.Bounds() != off.Bounds() || !bytes.Equal(restored.Pix, off.Pix) {
				t.Fatal("second edge did not restore the exact minimap", err)
			}
			mapRevealPNG(t, filepath.Join(dir, "off-restored.png"), restored)
			continue
		}
		on, err := app.HeadlessMinimap()
		if err != nil || bytes.Equal(off.Pix, on.Pix) {
			t.Fatal("first edge did not visibly reveal a qualified map", err)
		}
		cell, pixel, hidden, full := mapRevealTerrainPixel(t, f, off, on)
		proof := struct {
			AssetRoot, AssetRootSHA256          string
			Cell, Pixel                         image.Point
			Hidden, Revealed                    color.RGBA
			OffSHA256, OnSHA256                 string
			FogVisibleSHA256, FogExploredSHA256 string
			WorldHash                           uint64
			FrozenUnix                          int64
		}{root, rootHash, cell, pixel, hidden, full,
			fmt.Sprintf("%x", sha256.Sum256(off.Pix)), fmt.Sprintf("%x", sha256.Sum256(on.Pix)),
			fmt.Sprintf("%x", sha256.Sum256(visible)), fmt.Sprintf("%x", sha256.Sum256(explored)), before.State.Hash, now.Unix()}
		encoded, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "witness.json"), append(encoded, '\n'), 0600); err != nil {
			t.Fatal("cannot write map reveal witness", err)
		}
		mapRevealPNG(t, filepath.Join(dir, "off.png"), off)
		mapRevealPNG(t, filepath.Join(dir, "on.png"), on)
		t.Logf("asset root %s: cell%v pixel%v %v -> %v; off/on %x/%x; unchanged World%x and %d save files", rootHash, cell, pixel, hidden, full, sha256.Sum256(off.Pix), sha256.Sum256(on.Pix), before.State.Hash, len(files))
	}
	if viewer.FogRevealed() {
		t.Fatal("second edge left reveal on")
	}
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, "quick-save-1.sav"))
	if err != nil {
		t.Fatal("bare F4 missing-write loss control", err)
	}
	owner, err := os.ReadFile(filepath.Join(store.Dir, "quick-save-1"+quickOwnerExtension))
	if err != nil {
		t.Fatal(err)
	}
	if sequence, err := validateQuickSavePair(0, raw, owner); err != nil || sequence != 1 || reflect.DeepEqual(files, mapRevealFiles(t, store.Dir)) {
		t.Fatal("bare F4 did not produce its actual owned SAV", sequence, err)
	}
	t.Logf("bare F4 loss control wrote SAV %x and ownership %x; fixed clock stayed Unix%d", sha256.Sum256(raw), sha256.Sum256(owner), now.Unix())
}

func mapRevealTerrainPixel(t *testing.T, f *FrontEnd, off, on *image.RGBA) (image.Point, image.Point, color.RGBA, color.RGBA) {
	t.Helper()
	m := f.live.mission.state.Map
	g := terrain.Grid{Width: m.Width, Height: m.Height, Block: mapload.PassabilityWith(m, f.Table)}
	window := image.Rect(m.Width, m.Height, 0, 0)
	for row := 0; row < m.Height; row++ {
		for col := 0; col < m.Width; col++ {
			if !g.BorderCell(col, row) {
				window.Min.X, window.Min.Y = min(window.Min.X, col), min(window.Min.Y, row)
				window.Max.X, window.Max.Y = max(window.Max.X, col+1), max(window.Max.Y, row+1)
			}
		}
	}
	if window.Empty() {
		t.Fatal("no installed terrain window")
	}
	den := max(window.Dx(), window.Dy())
	size := image.Pt(window.Dx()*128/den, window.Dy()*128/den)
	art := f.gameMenuArt()
	if art == nil || art.Minimap == nil || art.MinimapSeam == nil {
		t.Fatal("installed minimap frame absent")
	}
	seam := art.MinimapSeam.Bounds().Dx()
	origin := image.Pt(16+seam-seam/2+(128-size.X)/2, 18+(128-size.Y)/2)
	for y := 2; y < size.Y-2; y++ {
		for x := 2; x < size.X-2; x++ {
			col, row := x*den/128, y*den/128
			if 128 >= den {
				col, row = ((x+1)*den-1)/128, ((y+1)*den-1)/128
			}
			cell := window.Min.Add(image.Pt(col, row))
			index := cell.Y*m.Width + cell.X
			state := f.live.fog.project()[index]
			if state == ui.FogVisible {
				continue
			}
			ref := terrain.Resolve(m.Tiles[index])
			input := f.Tiles.Slot(ref.Slot).SubCell(ref.Sub)
			if input == nil {
				continue
			}
			b := input.Bounds()
			full := color.RGBAModel.Convert(input.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)).(color.RGBA)
			if full == (color.RGBA{64, 255, 64, 255}) || full == (color.RGBA{255, 64, 64, 255}) || full == (color.RGBA{168, 255, 168, 255}) {
				continue
			}
			pixel := origin.Add(image.Pt(x, y))
			hidden := color.RGBA{full.R / 2, full.G / 2, full.B / 2, full.A}
			if state == ui.FogUnseen {
				hidden = color.RGBAModel.Convert(art.Minimap.At(pixel.X-seam, pixel.Y)).(color.RGBA)
			}
			if hidden != full && off.RGBAAt(pixel.X, pixel.Y) == hidden && on.RGBAAt(pixel.X, pixel.Y) == full {
				return cell, pixel, hidden, full
			}
		}
	}
	t.Fatal("no changed minimap terrain pixel agrees with its independent installed tile input and raw fog")
	return image.Point{}, image.Point{}, color.RGBA{}, color.RGBA{}
}

func mapRevealFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if os.IsNotExist(err) && path == dir {
			return nil
		}
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err == nil {
			out[strings.TrimPrefix(path, dir+string(filepath.Separator))] = fmt.Sprintf("%x", sha256.Sum256(raw))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func mapRevealPNG(t *testing.T, path string, pic *image.RGBA) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, pic); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}
