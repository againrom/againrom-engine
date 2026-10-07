package ui

import (
	"image"
	"math/rand"
	"slices"
	"testing"
)

func TestShopWantRegionHoldsEveryHit(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	mask := &SlotMask{W: shopFigureRect.Dx(), H: shopFigureRect.Dy()}
	mask.Slot = make([]uint8, mask.W*mask.H)
	for i := range mask.Slot {
		if rng.Intn(3) != 0 {
			mask.Slot[i] = uint8(1 + rng.Intn(12))
		}
	}
	frame := image.Rect(0, 0, shopScreenW, shopScreenH)
	type probe struct {
		want     ShopControl
		grid     bool
		dollSlot int
	}
	probes := []probe{
		{ShopControl{Kind: ShopControlShelfCell, Index: shopShelfN - 1}, true, -1},
		{ShopControl{Kind: ShopControlPackCell, Index: 1}, true, -1},
		{ShopControl{Kind: ShopControlTableCell, Index: shopStripN - 1}, true, -1},
		{ShopControl{Kind: ShopControlDoll, Index: 3}, true, 3},
		{ShopControl{Kind: ShopControlMerchant}, false, -1},
		{ShopControl{Kind: ShopControlButton, Index: 2}, false, -1},
		{ShopControl{Kind: ShopControlShelfPick, Index: 1}, false, -1},
	}
	for _, book := range []bool{false, true} {
		for _, stats := range []bool{!book} {
			view := ShopScreenView{Book: book, SlotMask: mask, OrdinaryDollMask: mask}
			view.Character.Statistics = stats
			for _, pr := range probes {
				full := shopHitPixels(view, pr.want, pr.grid, pr.dollSlot, frame)
				fast := shopHitPixels(view, pr.want, pr.grid, pr.dollSlot, shopWantRegion(pr.want))
				if !slices.Equal(full, fast) {
					t.Fatalf("book %v stats %v %+v: region scan lists %d pixels, frame scan %d", book, stats, pr.want, len(fast), len(full))
				}
			}
		}
	}
}
