package game

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type panelReleaseState struct {
	gold   int
	worn   [sim.EquipSlots]uint16
	pack   []uint16
	shelf  []uint16
	table  []uint16
	mine   []bool
	member string
}

func panelReleaseSnapshot(s *townScreen) panelReleaseState {
	f := frontOf(s)
	i := s.shopMemberIndex()
	st := panelReleaseState{gold: f.Town.Gold(), worn: *s.shopWornSlots(i), member: f.Carried[i].ID}
	for _, stack := range s.shopPackStacks() {
		for n := uint32(0); n < stack.Count; n++ {
			st.pack = append(st.pack, stack.Code)
		}
	}
	for _, item := range f.Shop.Shelf(shopRoomShelves[s.shopChosen].shelf) {
		for n := int32(0); n < item.Count; n++ {
			st.shelf = append(st.shelf, uint16(item.Code))
		}
	}
	for _, place := range f.Shop.Table() {
		st.table = append(st.table, uint16(place.Code))
		st.mine = append(st.mine, place.Mine)
	}
	return st
}

func (a panelReleaseState) equal(b panelReleaseState) bool {
	return a.gold == b.gold && a.worn == b.worn && a.member == b.member && slices.Equal(a.pack, b.pack) &&
		slices.Equal(a.shelf, b.shelf) && slices.Equal(a.table, b.table) && slices.Equal(a.mine, b.mine)
}

// Panel releases route by stamp; SAVE and cold LOAD carry the result (DIV-1667).
func TestReleaseShopPanelReleaseRoutesByStamp(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	f.Town.gold = 100000
	if s.shopChosen < 0 || s.shopChosen >= len(shopRoomShelves) {
		t.Fatalf("the room opened on no shelf (%d)", s.shopChosen)
	}
	press := func(at image.Point, actions ...string) {
		t.Helper()
		for _, action := range actions {
			if err := app.HeadlessPointer(action, at.X, at.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	panelAt := func(corner bool) image.Point {
		t.Helper()
		if corner {
			r := ui.CharacterPaneCornerRect(ui.TownCharacterRegion, ui.CharacterPaneMode)
			c := r.Min.Add(r.Size().Div(2))
			return c
		}
		x, y, err := app.HeadlessShopPoint("doll_box", 0)
		if err != nil {
			t.Fatal(err)
		}
		return image.Pt(x, y)
	}
	crossShown := func() bool {
		t.Helper()
		c, ok := app.HeadlessCursor()
		if !ok || c.Pic == nil || c.Hot != image.Pt(38, 36) {
			return false
		}
		red := 0
		for i := 0; i+3 < len(c.Pic.Pix); i += 4 {
			if r, g, b, a := c.Pic.Pix[i], c.Pic.Pix[i+1], c.Pic.Pix[i+2], c.Pic.Pix[i+3]; a > 0 && r > 150 && g < 90 && b < 90 {
				red++
			}
		}
		return red > 0
	}
	pointOf := func(surface string, index int) image.Point {
		t.Helper()
		x, y, err := app.HeadlessShopPoint(surface, index)
		if err != nil {
			t.Fatal(err)
		}
		return image.Pt(x, y)
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
	shelfCell := pointOf("shelf", 0)

	for _, corner := range []bool{false, true} {
		before := panelReleaseSnapshot(s)
		press(shelfCell, "press")
		press(panelAt(corner), "move")
		if !crossShown() {
			t.Errorf("corner %v: no red cross under a shelf object held over the panel", corner)
		}
		press(panelAt(corner), "release")
		if crossShown() {
			t.Errorf("corner %v: the cross outlived the release", corner)
		}
		if after := panelReleaseSnapshot(s); !after.equal(before) {
			t.Errorf("corner %v: a shelf object released over the panel changed state:\nbefore %+v\nafter  %+v", corner, before, after)
		}
	}

	before := panelReleaseSnapshot(s)
	press(shelfCell, "press")
	press(pointOf("table", 0), "move", "release")
	staged := panelReleaseSnapshot(s)
	if len(staged.table) != 1 || staged.mine[0] || staged.equal(before) {
		t.Fatalf("shelf-to-table control staged %+v", staged)
	}
	for _, corner := range []bool{false, true} {
		press(pointOf("table", 0), "press")
		press(panelAt(corner), "move", "release")
		if after := panelReleaseSnapshot(s); !after.equal(staged) {
			t.Errorf("corner %v: a staged merchant place released over the panel changed state:\nbefore %+v\nafter  %+v", corner, staged, after)
		}
	}
	if action := s.shopClear(); action.Msg != "the table is cleared" {
		t.Fatalf("clear: %+v", action)
	}

	slot := 0
	for n := sim.EquipSlots; n >= 1 && slot == 0; n-- {
		if _, _, err := app.HeadlessShopPoint("doll", n); err == nil && s.shopWornSlots(s.shopMemberIndex())[n-1] != 0 {
			slot = n
		}
	}
	if slot == 0 {
		t.Fatal("the shown member wears nothing")
	}
	code := s.shopWornSlots(s.shopMemberIndex())[slot-1]
	press(pointOf("doll", slot), "press", "release")
	if s.shopWornSlots(s.shopMemberIndex())[slot-1] != 0 {
		t.Fatalf("a tap on slot %d left %#x worn", slot, code)
	}
	packCell := func() image.Point {
		t.Helper()
		cell, err := f.headlessShopCell("pack", code)
		if err != nil {
			t.Fatal(err)
		}
		return pointOf("pack", cell)
	}
	unworn := panelReleaseSnapshot(s)
	gold := f.Town.Gold()

	press(packCell(), "press")
	press(panelAt(false), "move")
	if crossShown() {
		t.Error("a red cross shows under an owned pack object held over the panel")
	}
	press(panelAt(false), "release")
	if got := s.shopWornSlots(s.shopMemberIndex())[slot-1]; got != code || f.Town.Gold() != gold {
		t.Fatalf("pack object over the panel: slot %d wears %#x gold %d, want %#x and %d", slot, got, f.Town.Gold(), code, gold)
	}
	press(pointOf("doll", slot), "press", "release")
	press(packCell(), "press")
	press(pointOf("table", 0), "move", "release")
	if len(f.Shop.Table()) != 1 || !f.Shop.Table()[0].Mine {
		t.Fatalf("pack-to-table staged %+v", f.Shop.Table())
	}
	press(pointOf("table", 0), "press")
	press(panelAt(true), "move", "release")
	if got := s.shopWornSlots(s.shopMemberIndex())[slot-1]; got != code || f.Town.Gold() != gold || len(f.Shop.Table()) != 0 {
		t.Fatalf("owned table place over the panel: slot %d wears %#x gold %d table %d, want %#x, %d, 0", slot, got, f.Town.Gold(), len(f.Shop.Table()), code, gold)
	}
	if after := panelReleaseSnapshot(s); after.gold != unworn.gold || slices.Equal(after.pack, unworn.pack) == false && len(after.pack) != len(unworn.pack)-1 {
		t.Errorf("owned release changed the carried goods unexpectedly: %v then %v", unworn.pack, after.pack)
	}

	live := panelReleaseSnapshot(s)
	cold := currentTownReload(t, currentTownSave(t, f))
	var member *mapload.PartyMember
	for i := range cold.Carried {
		if cold.Carried[i].ID == live.member {
			member = &cold.Carried[i]
		}
	}
	if member == nil {
		t.Fatalf("cold LOAD lost member %q", live.member)
	}
	worn, pack := memberItemCodes(*member)
	liveMember := f.Carried[s.shopMemberIndex()]
	wantWorn, wantPack := memberItemCodes(liveMember)
	if cold.Town.Gold() != live.gold || worn != wantWorn || !slices.Equal(pack, wantPack) {
		t.Fatalf("cold LOAD: gold %d worn %#x pack %v, live gold %d worn %#x pack %v",
			cold.Town.Gold(), worn, pack, live.gold, wantWorn, wantPack)
	}
	if worn[slot-1] != code {
		t.Errorf("cold LOAD: slot %d wears %#x, want %#x", slot, worn[slot-1], code)
	}
}

func releaseShopBuyToPack(t *testing.T, app *ui.App, f *FrontEnd, code uint16) image.Point {
	t.Helper()
	point := func(surface string, index int) image.Point {
		t.Helper()
		x, y, err := app.HeadlessShopPoint(surface, index)
		if err != nil {
			t.Fatal(err)
		}
		return image.Pt(x, y)
	}
	step := func(at image.Point, actions ...string) {
		t.Helper()
		for _, action := range actions {
			if err := app.HeadlessPointer(action, at.X, at.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	step(point("shelf", 0), "press")
	step(point("table", 0), "move", "release")
	step(point("button", 1), "press", "release")
	wheel := point("pack", 1)
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
		step(wheel, dir)
	}
	t.Fatalf("the pack strip never showed %#x after the purchase", code)
	return image.Point{}
}
