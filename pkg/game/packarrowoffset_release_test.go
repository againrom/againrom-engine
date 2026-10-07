package game

import (
	"fmt"
	"image"
	"os"
	"testing"

	"againrom/pkg/ui"
)

func rowShift(pic, cap *image.RGBA, r image.Rectangle) int {
	best, bestShift := 1e18, 0
	for s := -4; s <= 4; s++ {
		sum, n := 0.0, 0
		for y := r.Min.Y + 4; y < r.Max.Y-4; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				a, b := pic.RGBAAt(x, y), cap.RGBAAt(x, y+s)
				d := absInt(int(a.R)-int(b.R)) + absInt(int(a.G)-int(b.G)) + absInt(int(a.B)-int(b.B))
				sum += float64(d)
				n++
			}
		}
		if m := sum / float64(n); m < best {
			best, bestShift = m, s
		}
	}
	return bestShift
}

func TestReleaseShopPackCornerStatesAlign(t *testing.T) {
	f := releaseFront(t)
	f.Town = NewTown(f.Campaign.Value())
	for _, mission := range f.Campaign.Value().Main {
		if mission < 30 {
			f.Town.Won(mission)
		}
	}
	f.Town.Arrive()
	shop := f.TownScreen().(*townScreen)
	shop.room, shop.shopBook = roomShop, false
	shop.composeShopFaces()
	view := shop.ShopScreen()
	rects := map[string]image.Rectangle{"left": image.Rect(0, 391, 32, 479), "right": image.Rect(432, 391, 464, 479)}
	type state struct {
		name        string
		on, hovered bool
	}
	for side, r := range rects {
		var cap *image.RGBA
		for i, st := range []state{{"cap", false, false}, {"enabled", true, false}, {"hover", true, true}} {
			v := view
			v.PackBack, v.PackForward = st.on, st.on
			pic := ui.ComposeShopScreen(v, r.Min.Add(image.Pt(8, 8)), st.hovered, nil, false)
			if i == 0 {
				cap = pic
			}
			shift := rowShift(pic, cap, r)
			fmt.Printf("shop %s %s: best row shift against the cap %d\n", side, st.name, shift)
			if shift != 0 {
				t.Errorf("%s %s sits %d rows off the cap", side, st.name, shift)
			}
			if out := os.Getenv("AGAINROM_HOVER_PANEL_DIR"); out != "" {
				_ = os.MkdirAll(out, 0o755)
				if err := writeMediaFrame(out, fmt.Sprintf("corner-offset-shop-%s-%s-%s", side, st.name, shopRootName()), pic.SubImage(image.Rect(r.Min.X-40, r.Min.Y-30, r.Max.X+8, r.Max.Y+1)).(*image.RGBA)); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
