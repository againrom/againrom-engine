package game

import (
	"image"
	"os"
	"testing"

	"againrom/pkg/ui"
)

// The shop's bottom strip shows its corner pictures at zero scroll: the frame
// caps are painted, not left black.
func TestReleaseShopPackCornersAreDrawnWithoutScrolling(t *testing.T) {
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
	if view.PackBack || view.PackForward {
		t.Skip("the strip can scroll in this fixture")
	}
	pic := ui.ComposeShopScreen(view, image.Point{}, false, nil, false)
	for name, r := range map[string]image.Rectangle{"left": image.Rect(0, 391, 32, 479), "right": image.Rect(432, 391, 464, 479)} {
		lit := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				c := pic.RGBAAt(x, y)
				if int(c.R)+int(c.G)+int(c.B) > 60 {
					lit++
				}
			}
		}
		if lit*3 < r.Dx()*r.Dy() {
			t.Errorf("%s corner: %d of %d pixels lit, want the cap drawn", name, lit, r.Dx()*r.Dy())
		}
	}
	if out := os.Getenv("AGAINROM_HOVER_PANEL_DIR"); out != "" {
		_ = os.MkdirAll(out, 0o755)
		if err := writeMediaFrame(out, "shop-pack-corners-"+shopRootName(), pic); err != nil {
			t.Fatal(err)
		}
	}
}

func shopRootName() string { return os.Getenv("AGAINROM_ASSETS")[len(os.Getenv("AGAINROM_ASSETS"))-2:] }
