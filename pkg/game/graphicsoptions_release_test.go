package game

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseGraphicsOptions1190InstalledMenusAndColdLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.LoadOptions()
	if len(f.SackFrames) == 0 || len(f.SackBoundaries) != len(f.SackFrames) {
		t.Fatal("installed backpack boundary pair absent", len(f.SackFrames), len(f.SackBoundaries))
	}
	opaque := 0
	for i, b := range f.SackBoundaries {
		if b == nil || b.Palette != f.SackFrames[i].Palette {
			t.Fatal("boundary did not borrow base palette", i)
		}
		for _, p := range b.Pixels {
			if p.Opaque {
				opaque++
			}
		}
	}
	if opaque == 0 {
		t.Fatal("empty boundary sheet")
	}
	a := f.App("graphics options")
	a.SetCutscenes(nil)
	party := f.ChargenParty(ui.ChargenResult{Name: "Graphics witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	if err := a.HeadlessKey("ctrl-o"); err != nil || f.live.view.GraphicsOptions().Smoothing {
		t.Fatal("map shortcut missed smoothing toggle from the default on", err)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	// Map input advances an ordinary frame. Compare the held menu interval,
	// after Ctrl+O and Escape have finished their normal map processing.
	hash := f.live.world.Hash()
	before := bytes.Clone(a.GameMenuPanel().Pix)
	for _, p := range [][2]int{{180, 180}, {180, 206}, {180, 232}, {180, 258}} {
		for _, edge := range []string{"press", "release"} {
			if err := a.HeadlessPointer(edge, p[0], p[1]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if f.live.view.GraphicsOptions().HideShadows || f.live.world.Hash() != hash {
		t.Fatal("a graphics click reached the renderer before OK", f.live.view.GraphicsOptions())
	}
	pic := a.GameMenuPanel()
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	want := ui.GraphicsOptions{Smoothing: true, HideShadows: true, DisableLighting: true, StaticObjects: true}
	if f.live.view.GraphicsOptions() != want || f.live.world.Hash() != hash {
		t.Fatal("installed controls missed renderer or stepped world", f.live.view.GraphicsOptions())
	}
	if bytes.Equal(before, pic.Pix) {
		t.Fatal("checkbox artwork did not reflect choices")
	}
	if dir := os.Getenv("AGAINROM_OPTIONS_ARTIFACTS"); dir != "" {
		dir = filepath.Join(dir, filepath.Base(os.Getenv("AGAINROM_ASSETS")))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		out, err := os.Create(filepath.Join(dir, "graphics-options.png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(out, pic)
		closeErr := out.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	cold.Options = f.Options
	cold.LoadOptions()
	ca := cold.App("cold graphics")
	open, town, err := cold.Restore(s)
	if err != nil || town {
		t.Fatal("native load", err)
	}
	if err := ca.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if cold.live.view.GraphicsOptions() != want || cold.live.world.Hash() != hash {
		t.Fatal("cold LOAD lost process graphics or changed world")
	}
	if err := cold.setGameOption(true, ui.GameOptionAnimation, 1); err != nil {
		t.Fatal(err)
	}
	if err := cold.setGameOption(true, ui.GameOptionLighting, 1); err != nil {
		t.Fatal(err)
	}
	if got := cold.live.view.GraphicsOptions(); got.StaticObjects || got.DisableLighting {
		t.Fatal("animation/light did not re-enable", got)
	}
	t.Logf("installed base/boundary frames%d opaque boundary pixels%d; Ctrl+O, all four pointer switches, unchanged world and cold AGS profile", len(f.SackFrames), opaque)
}
