package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// shopGoods lists the worn set, pack, chosen shelf and table as one sorted list.
func shopGoods(s *townScreen) []uint16 {
	st := panelReleaseSnapshot(s)
	var out []uint16
	for _, code := range st.worn {
		if code != 0 {
			out = append(out, code)
		}
	}
	out = append(out, st.pack...)
	out = append(out, st.shelf...)
	out = append(out, st.table...)
	slices.Sort(out)
	return out
}

// A shelf object, a merchant table place and a customer table place released
// over each pane rectangle in both modes return, stay and are worn; nothing is
// bought or lost (SHOP-114, SHOP-115, SHOP-116).
func TestReleaseShopPaneReleaseBySourceOverEachRectangle(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	f.Town.gold = 100000
	rects := []struct {
		name   string
		corner ui.CharacterPaneCorner
	}{
		{"backpack", ui.CharacterPaneBackpack}, {"book", ui.CharacterPaneBook}, {"mode", ui.CharacterPaneMode},
		{"previous", ui.CharacterPanePrev}, {"next", ui.CharacterPaneNext}, {"menu", ui.CharacterPaneMenu},
	}
	rectAt := func(c ui.CharacterPaneCorner) image.Point {
		r := ui.CharacterPaneCornerRect(ui.TownCharacterRegion, c)
		return r.Min.Add(r.Size().Div(2))
	}
	edges := func(at image.Point, actions ...string) {
		t.Helper()
		for _, action := range actions {
			if err := app.HeadlessPointer(action, at.X, at.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	pointOf := func(surface string, index int) image.Point {
		t.Helper()
		x, y, err := app.HeadlessShopPoint(surface, index)
		if err != nil {
			t.Fatal(err)
		}
		return image.Pt(x, y)
	}
	packPoint := func(code uint16) image.Point {
		t.Helper()
		wheel := pointOf("pack", 1)
		for range 32 {
			cell, err := f.headlessShopCell("pack", code)
			if err != nil {
				t.Fatal(err)
			}
			if x, y, err := app.HeadlessShopPoint("pack", cell); err == nil {
				return image.Pt(x, y)
			}
			dir := "wheel-down"
			if cell < 1 {
				dir = "wheel-up"
			}
			edges(wheel, dir)
		}
		t.Fatalf("the pack strip never showed %#x", code)
		return image.Point{}
	}
	worn := func() [sim.EquipSlots]uint16 { return *s.shopWornSlots(s.shopMemberIndex()) }
	member := s.shopMemberIndex()
	untouched := func(label string, statistics bool) {
		t.Helper()
		if s.room != roomShop || s.shopMemberIndex() != member || s.townStats != statistics || s.shopBook {
			t.Errorf("%s: room %d, member %d, statistics %v, book %v; want the shop, member %d, statistics %v, no book",
				label, s.room, s.shopMemberIndex(), s.townStats, s.shopBook, member, statistics)
		}
	}

	shelf := shopRoomShelves[s.shopChosen].shelf
	wearable := -1
	for k, item := range f.Shop.Shelf(shelf) {
		if _, ok, _ := s.shopWearInstance(item.Instance()); ok && item.Price > 0 {
			wearable = k
			break
		}
	}
	if wearable < 0 {
		t.Skip("the generated shelf holds nothing the shown member can wear")
	}
	s.shelfBase = wearable

	slot := 0
	for n := sim.EquipSlots; n >= 1 && slot == 0; n-- {
		if _, _, err := app.HeadlessShopPoint("doll", n); err == nil && worn()[n-1] != 0 {
			slot = n
		}
	}
	if slot == 0 {
		t.Fatal("the shown member wears nothing")
	}
	code := worn()[slot-1]

	// Loss control: shopGoods must notice a dropped table place.
	{
		edges(pointOf("shelf", 0), "press")
		edges(pointOf("table", 0), "move", "release")
		staged := shopGoods(s)
		if len(f.Shop.Table()) != 1 {
			t.Fatalf("shelf-to-table staged %+v", f.Shop.Table())
		}
		place, ok := f.Shop.TakeOffTable(0, f.Shop.Table()[0].Count)
		if !ok {
			t.Fatal("loss control: the staged place did not come off the table")
		}
		if slices.Equal(shopGoods(s), staged) {
			t.Fatal("loss control: dropping a table place left the goods list unchanged")
		}
		f.Shop.returnToShelf(place.From, place.ShopItem)
		s.shelfBase = wearable
	}

	for _, statistics := range []bool{false, true} {
		mode := map[bool]string{false: "figure", true: "statistics"}[statistics]
		for _, r := range rects {
			label := mode + "/" + r.name
			if statistics != s.townStats {
				if err := app.HeadlessKey("tab"); err != nil {
					t.Fatal(err)
				}
			}
			untouched(label+" before", statistics)
			at := rectAt(r.corner)

			before, goods := panelReleaseSnapshot(s), shopGoods(s)
			edges(pointOf("shelf", 0), "press")
			edges(at, "move", "release")
			if after := panelReleaseSnapshot(s); !after.equal(before) || !slices.Equal(shopGoods(s), goods) {
				t.Errorf("%s: shelf object released over the pane changed state:\nbefore %+v\nafter  %+v", label, before, after)
			}
			untouched(label+" shelf", statistics)

			edges(pointOf("shelf", 0), "press")
			edges(pointOf("table", 0), "move", "release")
			staged, stagedGoods := panelReleaseSnapshot(s), shopGoods(s)
			if len(staged.table) != 1 || staged.mine[0] {
				t.Fatalf("%s: shelf-to-table staged %+v", label, staged)
			}
			edges(pointOf("table", 0), "press")
			edges(at, "move", "release")
			if after := panelReleaseSnapshot(s); !after.equal(staged) || !slices.Equal(shopGoods(s), stagedGoods) {
				t.Errorf("%s: merchant place released over the pane changed state:\nbefore %+v\nafter  %+v", label, staged, after)
			}
			if worn()[slot-1] == 0 {
				t.Errorf("%s: a merchant place released over the pane unworn slot %d", label, slot)
			}
			untouched(label+" merchant place", statistics)
			s.shopClear()
			if got := panelReleaseSnapshot(s); !got.equal(before) {
				t.Fatalf("%s: clearing the table did not restore the shelf:\nwant %+v\ngot  %+v", label, before, got)
			}

			// The statistics card hides the figure.
			toggle := func() {
				if statistics {
					if err := app.HeadlessKey("tab"); err != nil {
						t.Fatal(err)
					}
				}
			}
			toggle()
			edges(pointOf("doll", slot), "press", "release")
			toggle()
			if worn()[slot-1] != 0 {
				t.Fatalf("%s: a tap on slot %d left %#x worn", label, slot, worn()[slot-1])
			}
			goods, gold := shopGoods(s), f.Town.Gold()
			edges(packPoint(code), "press")
			edges(pointOf("table", 0), "move", "release")
			if len(f.Shop.Table()) != 1 || !f.Shop.Table()[0].Mine {
				t.Fatalf("%s: pack-to-table staged %+v", label, f.Shop.Table())
			}
			edges(pointOf("table", 0), "press")
			edges(at, "move", "release")
			if worn()[slot-1] != code || len(f.Shop.Table()) != 0 || f.Town.Gold() != gold || !slices.Equal(shopGoods(s), goods) {
				t.Errorf("%s: owned place released over the pane: slot %d wears %#x, table %d, gold %d, goods %v; want %#x, 0, %d, %v",
					label, slot, worn()[slot-1], len(f.Shop.Table()), f.Town.Gold(), shopGoods(s), code, gold, goods)
			}
			untouched(label+" owned place", statistics)
		}
	}
}
