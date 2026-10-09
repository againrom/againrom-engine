package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

func TestReleaseMinimapAlwaysVisibleAfterHistoricalUIAndSAVLoad(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("always visible minimap")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	for n := 0; a.HeadlessNoticeOpen() && n < 32; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if a.HeadlessNoticeOpen() {
		t.Fatal("mission notice did not settle")
	}
	if !f.live.stopped {
		if err := a.HeadlessKey("0"); err != nil {
			t.Fatal(err)
		}
	}
	before := f.live.view.SaveApplication()
	world := f.live.world.Hash()
	visible, explored := slices.Clone(f.live.fog.visible), slices.Clone(f.live.fog.explored)
	historical := before
	historical.MinimapOpen = false
	var wire bytes.Buffer
	if err := gob.NewEncoder(&wire).Encode(historical); err != nil {
		t.Fatal(err)
	}
	var old ui.SaveApplicationState
	if err := gob.NewDecoder(&wire).Decode(&old); err != nil || old.MinimapOpen {
		t.Fatal("historical false wire field lost", err)
	}
	if err := f.live.view.RestoreSaveApplication(old); err != nil {
		t.Fatal(err)
	}
	minimapVisibleInstalled(t, f, a, "historical-ui")
	if !reflect.DeepEqual(before, f.live.view.SaveApplication()) || world != f.live.world.Hash() || !bytes.Equal(visible, f.live.fog.visible) || !bytes.Equal(explored, f.live.fog.explored) {
		t.Fatal("normalizing historical switch changed another application field, World or fog")
	}
	s, _, err := f.Snapshot(true)
	if err != nil || s.ApplicationState == nil || !s.ApplicationState.View.MinimapOpen {
		t.Fatal("current capture not visible", err)
	}
	raw, err := f.ExportCurrentSave(s, "always visible minimap")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "visible.sav"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	g, b := minimapColdLoad(t, dir, "visible.sav")
	minimapVisibleInstalled(t, g, b, "cold-sav")
	assertCurrentWorldEqual(t, f.live.world, g.live.world, "minimap cold SAV")
	if !reflect.DeepEqual(before, g.live.view.SaveApplication()) || !bytes.Equal(visible, g.live.fog.visible) || !bytes.Equal(explored, g.live.fog.explored) {
		t.Fatal("cold SAV changed application or fog")
	}
	tick := g.live.world.Tick()
	if err := b.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if err := b.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if g.live.world.Tick() <= tick {
		t.Fatal("post-load simulation did not advance", tick, g.live.world.Tick())
	}
	t.Logf("post-load ticks: %d -> %d", tick, g.live.world.Tick())
	minimapDismissNotices(t, b)
	minimapVisibleInstalled(t, g, b, "post-load")
	if !g.live.view.SaveApplication().MinimapOpen {
		t.Fatal("post-load capture not visible")
	}
	if _, _, err := g.Snapshot(true); err != nil {
		t.Fatal(err)
	}
	t.Log("historical gob UI false -> existing RestoreSaveApplication; ordinary native SAV -> cold App LOAD -> eight post-load frames with simulation advancement; capture true; restore leaves other switches/camera/World/fog unchanged")
}

func TestReleaseOwnerMinimapAlwaysVisibleOnLoadAndF9(t *testing.T) {
	source := os.Getenv("AGAINROM_MINIMAP_OWNER_SAV")
	if source == "" {
		t.Skip("no AGAINROM_MINIMAP_OWNER_SAV selected frozen owner minimap save")
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 364035 || fmt.Sprintf("%x", sha256.Sum256(raw)) != "1b0af0597a690e7e24632d72c356f1dab6925352b540a45f27fd9b3872592bf8" {
		t.Fatal("frozen owner save changed")
	}
	dir := t.TempDir()
	for _, name := range []string{"quick-save-1.sav", "quick-save-1.quick-owner"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(source), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	f, a := minimapColdLoad(t, dir, "quick-save-1.sav")
	minimapVisibleInstalled(t, f, a, "owner-load")
	before := f.live.view.SaveApplication()
	if err := a.HeadlessKey("f9"); err != nil {
		t.Fatal(err)
	}
	minimapVisibleInstalled(t, f, a, "owner-f9")
	if !reflect.DeepEqual(before, f.live.view.SaveApplication()) {
		t.Fatal("F9 changed saved application")
	}
	tick := f.live.world.Tick()
	for range 8 {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Tick() <= tick {
		t.Fatal("owner post-load simulation did not advance", tick, f.live.world.Tick())
	}
	t.Logf("owner post-load ticks: %d -> %d", tick, f.live.world.Tick())
	minimapDismissNotices(t, a)
	minimapVisibleInstalled(t, f, a, "owner-post-load")
}

func minimapColdLoad(t *testing.T, dir, name string) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	a := f.App("visible minimap cold load")
	t.Cleanup(a.StopAudio)
	a.Layout(1024, 768)
	f.ConfigureSaveSeams(a, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	t.Cleanup(a.FlushBackground)
	_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	groundAppLoad(t, a, list, name)
	return f, a
}

func minimapVisibleInstalled(t *testing.T, f *FrontEnd, a *ui.App, stage string) {
	t.Helper()
	pic, err := a.HeadlessMinimap()
	if err != nil {
		t.Fatal(stage, err)
	}
	palette := make(map[color.RGBA]bool)
	for _, word := range f.live.mission.state.Map.Tiles {
		ref := terrain.Resolve(word)
		tile := f.Tiles.Slot(ref.Slot).SubCell(ref.Sub)
		if tile == nil {
			continue
		}
		b := tile.Bounds()
		c := color.RGBAModel.Convert(tile.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)).(color.RGBA)
		if c.R == 0 && c.G == 0 && c.B == 0 {
			continue
		}
		palette[c] = true
		palette[color.RGBA{c.R / 2, c.G / 2, c.B / 2, c.A}] = true
	}
	count := func(img *image.RGBA) int {
		n := 0
		for y := 18; y < 146; y++ {
			for x := 24; x < 152; x++ {
				if palette[img.RGBAAt(x, y)] {
					n++
				}
			}
		}
		return n
	}
	terrainPixels := count(pic)
	if terrainPixels < 100 || count(image.NewRGBA(pic.Bounds())) != 0 {
		t.Fatal(stage, "terrain absent or blank loss control escaped", terrainPixels)
	}
	layers, err := a.HeadlessMissionColumnLayers()
	if err != nil || !slices.Contains(layers, "minimap") {
		t.Fatal(stage, "actual Draw omitted minimap", layers, err)
	}
	matches, opaque := 0, 0
	for y := range pic.Bounds().Dy() {
		for x := range pic.Bounds().Dx() {
			c := pic.RGBAAt(x, y)
			if c.A != 255 {
				continue
			}
			opaque++
			actual, known := a.DialogueBackdropPixel(1024-pic.Bounds().Dx()+x, y)
			if known && actual == c {
				matches++
			}
		}
	}
	if opaque < 26000 || matches != opaque {
		t.Fatal(stage, "final Draw minimap obscured", matches, opaque)
	}
	if !f.live.view.SaveApplication().MinimapOpen {
		t.Fatal(stage, "capture says hidden")
	}
	if dir := os.Getenv("AGAINROM_MINIMAP_VISIBILITY_WITNESS_DIR"); dir != "" {
		if !filepath.IsAbs(dir) || insideDir(os.Getenv("AGAINROM_ASSETS"), dir) {
			t.Fatal("minimap witness directory must be absolute and outside installed assets")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		mapRevealPNG(t, filepath.Join(dir, t.Name()+"-"+stage+".png"), pic)
	}
	t.Logf("%s: terrain=%d blank-loss=0 Draw-match=%d opaque=%d minimap=%x", stage, terrainPixels, matches, opaque, sha256.Sum256(pic.Pix))
}

func minimapDismissNotices(t *testing.T, a *ui.App) {
	t.Helper()
	for n := 0; a.HeadlessNoticeOpen() && n < 32; n++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if a.HeadlessNoticeOpen() {
		t.Fatal("post-load mission notice did not settle")
	}
}
