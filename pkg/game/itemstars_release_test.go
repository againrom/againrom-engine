package game

import (
	"crypto/sha256"
	"image"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseEnchantedItemStarsChangeEveryMeasuredShopGrid(t *testing.T) {
	f := releaseFront(t)
	item := mapload.ItemInstanceFromCode(0x812d, f.Table)
	item.Kind = 2
	item.Effects = []sim.ItemEffect{{Kind: 41, Operand: uint32(13) | uint32(15)<<16}}
	if item.Empty() || !item.HasEnchantment() {
		t.Fatalf("installed Staff witness did not form an enchanted item: %+v", item)
	}

	screen := f.bindTown(&townScreen{})
	cell := screen.shopShelfCell(shopItemFromInstance(item, 1), 1_000_000)
	if cell.Icon == nil || !cell.Star {
		t.Fatalf("installed cell = icon nil %v, star %v; want real base icon plus trail selector", cell.Icon == nil, cell.Star)
	}
	plainCell := cell
	plainCell.Star = false
	plain := ui.ShopScreenView{Art: f.shopArt(), Font: f.Font.Value(), Words: f.Words}
	plain.Shelf[0], plain.Table[0], plain.Pack[0] = plainCell, plainCell, plainCell
	phase0 := plain
	phase0.Shelf[0], phase0.Table[0], phase0.Pack[0] = cell, cell, cell
	phase2 := phase0
	phase2.Shelf[0].StarPhase = 2
	phase2.Table[0].StarPhase = 2
	phase2.Pack[0].StarPhase = 2

	baseFrame := ui.ComposeShopScreen(plain, image.Point{}, false, nil, false)
	firstFrame := ui.ComposeShopScreen(phase0, image.Point{}, false, nil, false)
	nextFrame := ui.ComposeShopScreen(phase2, image.Point{}, false, nil, false)
	// ITEM-STARCOMP-100's three shop origins are (1,32), (32,304), and
	// (32,396) for these first cells. ITEM-STARPIX-098's inclusive 6..74
	// footprint therefore gives these independent half-open rectangles.
	footprints := []struct {
		name string
		r    image.Rectangle
	}{
		{"shelf", image.Rect(7, 38, 76, 107)},
		{"trade table", image.Rect(38, 310, 107, 379)},
		{"shown-member pack", image.Rect(38, 402, 107, 471)},
	}
	for _, footprint := range footprints {
		if n := itemStarReleaseDiff(baseFrame, firstFrame, footprint.r); n == 0 {
			t.Errorf("%s has no changed installed-art pixel at phase 0", footprint.name)
		}
		if n := itemStarReleaseDiff(firstFrame, nextFrame, footprint.r); n == 0 {
			t.Errorf("%s did not animate between phase 0 and phase 2", footprint.name)
		}
	}
	for y := 0; y < baseFrame.Bounds().Dy(); y++ {
		for x := 0; x < baseFrame.Bounds().Dx(); x++ {
			if baseFrame.RGBAAt(x, y) == firstFrame.RGBAAt(x, y) {
				continue
			}
			p := image.Pt(x, y)
			inside := false
			for _, footprint := range footprints {
				inside = inside || p.In(footprint.r)
			}
			if !inside {
				t.Fatalf("star changed pixel %v outside all three decoded footprints", p)
			}
		}
	}

	baseHash := sha256.Sum256(baseFrame.Pix)
	firstHash := sha256.Sum256(firstFrame.Pix)
	nextHash := sha256.Sum256(nextFrame.Pix)
	if baseHash == firstHash || firstHash == nextHash {
		t.Fatalf("installed render hashes did not discriminate plain/phase0/phase2: %x %x %x", baseHash, firstHash, nextHash)
	}
	t.Logf("installed shop render sha256 plain=%x phase0=%x phase2=%x", baseHash, firstHash, nextHash)
}

func itemStarReleaseDiff(a, b *image.RGBA, r image.Rectangle) int {
	r = r.Intersect(a.Bounds()).Intersect(b.Bounds())
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}
