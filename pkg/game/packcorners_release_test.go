package game

import (
	"fmt"
	"image"
	"os"
	"testing"
)

func missionPackLayerAt(t *testing.T, w, h int) *image.RGBA {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("pack-corners")
	app.Layout(w, h)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	id := live.mission.ids[0]
	for n := 0; n < 32; n++ {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	e, _ := live.entity(id)
	live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	pack, err := app.HeadlessInventoryPack()
	if err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(pack.Bounds())
	copy(out.Pix, pack.Pix)
	return out
}

func TestReleaseMissionPackBarCornersAreCapsAtZeroScroll(t *testing.T) {
	for _, size := range [][2]int{{1024, 768}, {1470, 800}} {
		pic := missionPackLayerAt(t, size[0], size[1])
		w := pic.Bounds().Dx()
		cols := (w - 240) / 80
		first := 32 + ((w-240)%80)/2
		for name, x0 := range map[string]int{"left": first - 32, "right": first + 80*cols} {
			gem := 0
			for y := 0; y < 88; y++ {
				for x := x0; x < x0+32 && x < w; x++ {
					c := pic.RGBAAt(x, y)
					if c.R > 60 && int(c.G)*2 < int(c.R) && int(c.B)*2 < int(c.R) {
						gem++
					}
				}
			}
			if gem < 6 {
				t.Errorf("%dx%d %s corner: %d red gem pixels, want the cap drawn", size[0], size[1], name, gem)
			}
		}
		if out := os.Getenv("AGAINROM_HOVER_PANEL_DIR"); out != "" {
			_ = os.MkdirAll(out, 0o755)
			if err := writeMediaFrame(out, fmt.Sprintf("mission-pack-corners-%d-%s", size[0], shopRootName()), pic); err != nil {
				t.Fatal(err)
			}
		}
	}
}
