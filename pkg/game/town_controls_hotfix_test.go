package game

import (
	"againrom/pkg/ui"
	"testing"
)

func TestTavernDoubleClickTalkHireAndDismiss(t *testing.T) {
	f := shellFrontEnd()
	f.Archives = tavernTalkArchives(t)
	f.Font = resolved(missionFont(), nil)
	s := f.townUI
	before := f.Town.Gold()
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, true)
	if s.room != roomTalk || s.npc != 9 || f.Town.Gold() != before || len(f.Carried) != 1 {
		t.Fatal("story double click did not open only its conversation")
	}
	s.room = roomTavern
	for _, hired := range []bool{true, false} {
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 0}, true)
		if s.room != roomTavern || f.Town.MercenaryHired(3) != hired {
			t.Fatalf("mercenary double click: room=%v hired=%v", s.room, f.Town.MercenaryHired(3))
		}
	}
	if len(f.Carried) != 1 || f.Town.Gold() != before {
		t.Fatal("hire/dismiss round trip changed party or purse")
	}
}

func TestShopInventoryArrowActionsMoveOnlyTheViewport(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	s.room = roomShop
	f.Carried[0].Carried = []uint16{0x0e17, 0x0e18, 0x0e19, 0x0e1a, 0x0e1b, 0x0e1c, 0x0e1d}
	n := len(s.shopPackStacks())
	if want := n + 1 - 5; n < 5 || s.packScrollMax() != want {
		t.Fatalf("%d stacks give scroll max %d, want %d", n, s.packScrollMax(), want)
	}
	before := f.Town.Gold()
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackRight})
	if s.packBase != 1 {
		t.Fatal("right arrow did not expose next pack position", s.packBase)
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackLeft})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackLeft})
	if s.packBase != 0 || f.Town.Gold() != before {
		t.Fatal("left arrow did not clamp or changed money")
	}
	for i := 0; i < n+3; i++ {
		s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackRight})
	}
	if s.packBase != s.packScrollMax() {
		t.Fatalf("right arrow ran to %d, want the last full strip at %d", s.packBase, s.packScrollMax())
	}
	if v := s.ShopScreen(); !v.PackBack || v.PackForward {
		t.Fatalf("arrows at the end: back %v forward %v", v.PackBack, v.PackForward)
	}
}

func TestShopInventoryStripDoesNotScrollWithFiveOrFewerElements(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	s.room = roomShop
	f.Carried[0].Carried = []uint16{0x0e17}
	for _, c := range []ui.ShopControlKind{ui.ShopControlPackRight, ui.ShopControlPackLeft} {
		s.ShopClick(ui.ShopControl{Kind: c})
		if v := s.ShopScreen(); s.packBase != 0 || v.PackBack || v.PackForward {
			t.Fatalf("a short pack scrolled: base %d back %v forward %v", s.packBase, v.PackBack, v.PackForward)
		}
	}
}
