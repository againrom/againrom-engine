package game

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// shopShiftDragHelm is shopTable's helm row priced 25, odd, so the whole-stack
// payout ceil(11*25/2) differs from eleven unit payouts (SHOP-SELL-010).
var shopShiftDragHelm = uint16(data.ComposeItemCode(2, 5, 0, 1))

// shopShiftDragApp enters the shop through App's load and town input.
func shopShiftDragApp(t *testing.T, pack []uint16, shelf []ShopItem) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f := shellFrontEnd()
	items := shopTable()
	f.Table.Shapes, f.Table.Materials = items.Shapes, items.Materials
	f.Table.Weapons, f.Table.Armors, f.Table.Shields, f.Table.Magic = items.Weapons, items.Armors, items.Shields, items.Magic
	f.Carried[0].Carry = &mapload.Carry{Items: pack}
	f.Shop = NewShop(0)
	f.Shop.shelves[ShelfArmour] = shelf
	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(int) int { return 0 }
	a := f.App("shop-shift-drag")
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.sav", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) {
		f.townUI.resetForNewGame()
		return nil, true, nil
	})
	for _, step := range []func() error{
		func() error { return a.HeadlessKey("load") },
		func() error { return a.HeadlessActivate("@first") },
		func() error { return a.HeadlessActivate("SHOP") },
		a.HeadlessStep,
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	if f.townUI.room != roomShop {
		t.Fatalf("App entered room %d, want the shop", f.townUI.room)
	}
	f.townUI.CloseTip()
	return f, a, f.townUI
}

func shopShiftDragPack(n int) []uint16 {
	pack := make([]uint16, n)
	for i := range pack {
		pack[i] = shopShiftDragHelm
	}
	return pack
}

// shopPoints resolves the first pack stack, the first table place, the first
// shelf cell and the Buy and Sell buttons through the production hit test.
// They are layout, not stock: every App laid out at 640x480 shares them.
func shopPoints(t *testing.T, a *ui.App) map[string]image.Point {
	t.Helper()
	out := map[string]image.Point{}
	for name, ref := range map[string]struct {
		surface string
		index   int
	}{"pack": {"pack", 1}, "table": {"table", 0}, "shelf": {"shelf", 0}, "buy": {"button", 1}, "sell": {"button", 2}} {
		x, y, err := a.HeadlessShopPoint(ref.surface, ref.index)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = image.Pt(x, y)
	}
	return out
}

// shopDrag presses at from, moves to to and releases there. Each action names
// its own Shift level, as a held key sets it on that frame.
func shopDrag(t *testing.T, a *ui.App, from, to image.Point, press, move, release string) {
	t.Helper()
	for _, e := range []struct {
		action string
		at     image.Point
	}{{press, from}, {move, to}, {release, to}} {
		if err := a.HeadlessPointer(e.action, e.at.X, e.at.Y); err != nil {
			t.Fatal(err)
		}
	}
}

func shopTap(t *testing.T, a *ui.App, at image.Point, press, release string) {
	t.Helper()
	for _, action := range []string{press, release} {
		if err := a.HeadlessPointer(action, at.X, at.Y); err != nil {
			t.Fatal(err)
		}
	}
}

// shopShiftDragCounts is how many helms the pack, the table and the armour
// shelf hold.
func shopShiftDragCounts(f *FrontEnd, s *townScreen) (pack, table, shelf int32) {
	for _, st := range s.shopPackStacks() {
		if st.Code == shopShiftDragHelm {
			pack += int32(st.Count)
		}
	}
	for _, place := range f.Shop.Table() {
		if uint16(place.Code) == shopShiftDragHelm {
			table += place.Count
		}
	}
	for _, item := range f.Shop.Shelf(ShelfArmour) {
		if uint16(item.Code) == shopShiftDragHelm {
			shelf += item.Count
		}
	}
	return pack, table, shelf
}

// A drag moves the whole stack when Shift is held on its release frame and
// one unit otherwise, through the act the click uses (DIV-1463).
func TestShiftDragMovesTheWholeStack(t *testing.T) {
	shelfStack := []ShopItem{{Code: data.ItemCode(shopShiftDragHelm), Price: 25, Count: 11}}
	_, first, _ := shopShiftDragApp(t, nil, nil)
	at := shopPoints(t, first)

	t.Run("pack stack sold at the stack payout", func(t *testing.T) {
		f, a, s := shopShiftDragApp(t, shopShiftDragPack(11), nil)
		price := s.shopItemValue(sim.PlainItem(shopShiftDragHelm))
		if price != 25 {
			t.Fatalf("fixture helm price %d, want 25", price)
		}
		shopDrag(t, a, at["pack"], at["table"], "shift-press", "shift-move", "shift-release")
		if pack, table, _ := shopShiftDragCounts(f, s); pack != 0 || table != 11 {
			t.Fatalf("after a Shift drag of 11 the pack holds %d and the table %d, want 0 and 11", pack, table)
		}
		gold := f.Town.Gold()
		shopTap(t, a, at["sell"], "press", "release")
		want := (11*price + 1) / 2
		if got := int32(f.Town.Gold() - gold); got != want || want == 11*((price+1)/2) {
			t.Fatalf("the sale paid %d, want the whole-stack ceil(11*%d/2) = %d", got, price, want)
		}
		if pack, table, shelf := shopShiftDragCounts(f, s); pack != 0 || table != 0 || shelf != 11 {
			t.Fatalf("after the sale pack %d, table %d, shelf %d; want 0, 0, 11", pack, table, shelf)
		}
	})

	t.Run("the release level decides", func(t *testing.T) {
		f, a, s := shopShiftDragApp(t, shopShiftDragPack(11), nil)
		for _, step := range []struct {
			press, move, release string
			pack, table          int32
		}{
			{"press", "move", "release", 10, 1},
			{"shift-press", "shift-move", "release", 9, 2},
			{"press", "move", "shift-release", 0, 11},
		} {
			shopDrag(t, a, at["pack"], at["table"], step.press, step.move, step.release)
			if pack, table, _ := shopShiftDragCounts(f, s); pack != step.pack || table != step.table {
				t.Fatalf("%s/%s/%s: pack %d, table %d; want %d, %d", step.press, step.move, step.release,
					pack, table, step.pack, step.table)
			}
		}
	})

	t.Run("every staging arm", func(t *testing.T) {
		for _, tc := range []struct{ from, to string }{
			{"pack", "table"}, {"pack", "shelf"},
			{"shelf", "table"}, {"shelf", "pack"},
			{"table", "pack"}, {"table", "shelf"},
		} {
			for _, edge := range []string{"", "shift-"} {
				pack, shelf := shopShiftDragPack(11), []ShopItem(nil)
				if tc.from == "shelf" {
					pack, shelf = nil, []ShopItem{shelfStack[0].Clone()}
				}
				f, a, s := shopShiftDragApp(t, pack, shelf)
				if tc.from == "table" {
					shopTap(t, a, at["pack"], "shift-press", "shift-release")
				}
				p0, t0, s0 := shopShiftDragCounts(f, s)
				shopDrag(t, a, at[tc.from], at[tc.to], edge+"press", edge+"move", edge+"release")
				p1, t1, s1 := shopShiftDragCounts(f, s)
				moved := map[string]int32{"pack": p0 - p1, "table": t0 - t1, "shelf": s0 - s1}[tc.from]
				want := int32(1)
				if edge != "" {
					want = 11
				}
				if moved != want || p0+t0+s0 != p1+t1+s1 {
					t.Errorf("%s to %s, %q: moved %d, want %d (pack %d->%d, table %d->%d, shelf %d->%d)",
						tc.from, tc.to, edge, moved, want, p0, p1, t0, t1, s0, s1)
				}
			}
		}
	})

	t.Run("buy direction", func(t *testing.T) {
		f, a, s := shopShiftDragApp(t, nil, []ShopItem{shelfStack[0].Clone()})
		gold := f.Town.Gold()
		shopDrag(t, a, at["shelf"], at["table"], "shift-press", "shift-move", "shift-release")
		if _, table, shelf := shopShiftDragCounts(f, s); table != 11 || shelf != 0 {
			t.Fatalf("after a Shift drag of the shelf stack the table holds %d and the shelf %d, want 11 and 0", table, shelf)
		}
		shopTap(t, a, at["buy"], "press", "release")
		if pack, table, _ := shopShiftDragCounts(f, s); pack != 11 || table != 0 || gold-f.Town.Gold() != 11*25 {
			t.Fatalf("after Buy pack %d, table %d, spent %d; want 11, 0, %d", pack, table, gold-f.Town.Gold(), 11*25)
		}
	})
}
