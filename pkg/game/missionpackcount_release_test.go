package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/mapload"
)

// missionPackLayer is the mission pack bar the production App draws while the
// first party member carries a stack of potions of the given size.
func missionPackLayer(t *testing.T, count int) *image.RGBA {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	potion := mapload.ItemInstanceFromCode(0x0e07, f.Table)
	for i := 0; i < count; i++ {
		party[0].CarriedItems = append(party[0].CarriedItems, potion)
		party[0].Carried = append(party[0].Carried, potion.Code)
	}
	f.Carried = party
	app := f.App("1302-pack-count")
	app.Layout(1024, 768)
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
	found := false
	for _, c := range live.invSubject.PackCount {
		found = found || int(c) == count
	}
	if !found {
		t.Fatalf("pack counts = %v, want a stack of %d", live.invSubject.PackCount, count)
	}
	pack, err := app.HeadlessInventoryPack()
	if err != nil {
		t.Fatal(err)
	}
	out := image.NewRGBA(pack.Bounds())
	copy(out.Pix, pack.Pix)
	return out
}

// packCountClassification sorts the pixels in which two pack layers differ.
// A pixel is a face pixel when it holds a level of the ramp whose top is ink,
// and a shadow pixel when it holds the flat shadow colour. It reports whether
// every difference is one of the two, whether a face pixel holds the ramp top,
// and whether each face pixel has a shadow pixel one right and down unless
// another face pixel covers it.
func packCountClassification(a, b *image.RGBA, ink, shadow color.RGBA) (ok, top bool, faces int) {
	face, shade := map[image.Point]bool{}, map[image.Point]bool{}
	for y := 0; y < a.Bounds().Dy(); y++ {
		for x := 0; x < a.Bounds().Dx(); x++ {
			got := a.RGBAAt(x, y)
			if got == b.RGBAAt(x, y) {
				continue
			}
			switch {
			case got == shadow:
				shade[image.Pt(x, y)] = true
			case onRamp(got, ink):
				face[image.Pt(x, y)] = true
				top = top || got == ink
			default:
				return false, false, 0
			}
		}
	}
	for p := range face {
		if !face[p.Add(image.Pt(1, 1))] && !shade[p.Add(image.Pt(1, 1))] {
			return false, top, len(face)
		}
	}
	for p := range shade {
		if !face[p.Sub(image.Pt(1, 1))] {
			return false, top, len(face)
		}
	}
	return len(face) > 0 && len(shade) > 0, top, len(face)
}

func onRamp(c, ink color.RGBA) bool {
	for k := 1; k <= 15; k++ {
		if c == (color.RGBA{R: uint8(int(ink.R) * k / 15), G: uint8(int(ink.G) * k / 15), B: uint8(int(ink.B) * k / 15), A: 255}) {
			return true
		}
	}
	return false
}

// A stack count above one in the mission pack draws in the gold ramp over a
// flat shadow one pixel right and down (SHOP-108). The count is the only
// difference between a stack of one and a stack of twelve, so the difference
// of the two layers is the count's pixels. The loss controls give the same
// layers to the previous flat ink with a black shadow and to a grey ramp.
func TestReleaseMissionPackCountsDrawInTheGoldRampOverAFlatShadow(t *testing.T) {
	one, twelve := missionPackLayer(t, 1), missionPackLayer(t, 12)
	ink, shadow := color.RGBA{R: 185, G: 159, B: 73, A: 255}, color.RGBA{R: 8, G: 8, B: 8, A: 255}
	ok, top, faces := packCountClassification(twelve, one, ink, shadow)
	if !ok || !top || faces < 10 {
		t.Fatalf("the count is not the gold ramp over the flat shadow: ok=%v top=%v face pixels=%d", ok, top, faces)
	}
	for name, c := range map[string]struct{ ink, shadow color.RGBA }{
		"the previous flat ink and black shadow": {color.RGBA{R: 0xc8, G: 0xae, B: 0x54, A: 255}, color.RGBA{A: 255}},
		"a black shadow":                         {ink, color.RGBA{A: 255}},
		"a grey ramp":                            {color.RGBA{R: 160, G: 160, B: 160, A: 255}, shadow},
	} {
		if okc, _, _ := packCountClassification(twelve, one, c.ink, c.shadow); okc {
			t.Errorf("the check cannot tell the claimed count from %s", name)
		}
	}
}
