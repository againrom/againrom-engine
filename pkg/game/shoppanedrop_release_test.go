package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// An item taken off the shown character by a tap and dragged back over each of
// the pane's six rectangles is worn again, while the pane shows the figure and
// while it shows the statistics card, and the release presses no rectangle: the
// picker keeps its member and the pane its mode and its book. An item picked up
// off the figure and released over a rectangle stays worn.
func TestReleaseShopItemReleasedOnEachPaneRectangleIsWorn(t *testing.T) {
	app, s := releaseShopApp(t)
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
	underRect := func(p image.Point) bool {
		for _, r := range rects {
			if p.In(ui.CharacterPaneCornerRect(ui.TownCharacterRegion, r.corner)) {
				return true
			}
		}
		return false
	}
	worn := func() [sim.EquipSlots]uint16 { return *s.shopWornSlots(s.shopMemberIndex()) }
	carried := func(code uint16) (n int32) {
		for _, st := range s.shopPackStacks() {
			if st.Code == code {
				n += int32(st.Count)
			}
		}
		return n
	}
	edges := func(at image.Point, actions ...string) {
		t.Helper()
		for _, action := range actions {
			if err := app.HeadlessPointer(action, at.X, at.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	// packPoint scrolls the pack strip with the wheel until the cell holding
	// code is showing, and answers where that cell stands.
	packPoint := func(code uint16) image.Point {
		t.Helper()
		wx, wy, err := app.HeadlessShopPoint("pack", 1)
		if err != nil {
			t.Fatal(err)
		}
		for range 32 {
			cell, err := frontOf(s).headlessShopCell("pack", code)
			if err != nil {
				t.Fatal(err)
			}
			if x, y, err := app.HeadlessShopPoint("pack", cell); err == nil {
				return image.Pt(x, y)
			}
			wheel := "wheel-down"
			if cell < 1 {
				wheel = "wheel-up"
			}
			edges(image.Pt(wx, wy), wheel)
		}
		t.Fatalf("the pack strip never showed the cell holding %#x", code)
		return image.Point{}
	}

	member := s.shopMemberIndex()
	slot, code := 0, uint16(0)
	for n := sim.EquipSlots; n >= 1 && slot == 0; n-- {
		x, y, err := app.HeadlessShopPoint("doll", n)
		if err == nil && worn()[n-1] != 0 && !underRect(image.Pt(x, y)) {
			slot, code = n, worn()[n-1]
		}
	}
	if slot == 0 {
		t.Fatalf("no worn slot of %#x answers a doll pixel clear of the six rectangles", worn())
	}
	// dollAt is where the slot's own pixels stand while the item is worn.
	dollAt := func() image.Point {
		t.Helper()
		x, y, err := app.HeadlessShopPoint("doll", slot)
		if err != nil {
			t.Fatal(err)
		}
		return image.Pt(x, y)
	}
	t.Logf("member %d wears %#x in slot %d; the doll pixel is %v", member, code, slot, dollAt())

	unequip := func(label string) {
		t.Helper()
		if worn()[slot-1] == 0 {
			return
		}
		edges(dollAt(), "press", "release")
		if worn()[slot-1] != 0 || carried(code) == 0 {
			t.Fatalf("%s: a tap on slot %d left %#x worn and %d carried, want it carried", label, slot, worn()[slot-1], carried(code))
		}
	}
	untouched := func(label string, statistics bool) {
		t.Helper()
		if s.room != roomShop || s.shopMemberIndex() != member || s.townStats != statistics || s.shopBook {
			t.Errorf("%s: room %d, member %d, statistics %v, book %v; want the shop, member %d, statistics %v, no book",
				label, s.room, s.shopMemberIndex(), s.townStats, s.shopBook, member, statistics)
		}
	}

	for _, statistics := range []bool{false, true} {
		mode := map[bool]string{false: "figure", true: "statistics"}[statistics]
		for _, r := range rects {
			label := mode + "/" + r.name
			unequip(label)
			if statistics {
				if err := app.HeadlessKey("tab"); err != nil {
					t.Fatal(err)
				}
			}
			untouched(label+" before", statistics)
			held := carried(code)
			edges(packPoint(code), "press")
			edges(rectAt(r.corner), "move", "release")
			t.Logf("%s: pack cell released at %v: slot %d wears %#x, %d carried (was %d)",
				label, rectAt(r.corner), slot, worn()[slot-1], carried(code), held)
			if worn()[slot-1] != code || carried(code) != held-1 {
				t.Errorf("%s: slot %d wears %#x and %d are carried after the drag, want %#x worn and %d carried",
					label, slot, worn()[slot-1], carried(code), code, held-1)
			}
			untouched(label+" after", statistics)
			if statistics {
				if err := app.HeadlessKey("tab"); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	for _, r := range rects {
		label := "doll/" + r.name
		if worn()[slot-1] != code {
			x, y, err := app.HeadlessShopPoint("doll_box", 0)
			if err != nil {
				t.Fatal(err)
			}
			edges(packPoint(code), "press")
			edges(image.Pt(x, y), "move", "release")
		}
		if worn()[slot-1] != code {
			t.Fatalf("%s setup: slot %d wears %#x, want %#x", label, slot, worn()[slot-1], code)
		}
		before, held := worn(), carried(code)
		edges(dollAt(), "press")
		edges(rectAt(r.corner), "move", "release")
		if worn() != before || carried(code) != held {
			t.Errorf("%s: after the drag worn is %#x and %d carried, want %#x and %d", label, worn(), carried(code), before, held)
		}
		untouched(label, false)
	}
}
