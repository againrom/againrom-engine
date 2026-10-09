package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// priceLabels are the words a price line would start with: the authored word
// and the installed word for stats slot 1.
func priceLabels(f *FrontEnd) []string {
	return []string{ui.AuthoredWords().ItemStats[1], f.Words.ItemStats[1]}
}

// statesPrice reports whether any information line is a price label followed
// by a figure, with or without the drawn-text prefix.
func statesPrice(lines, labels []string) bool {
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		for _, label := range labels {
			if label != "" && strings.HasPrefix(line, label+" ") {
				return true
			}
		}
	}
	return false
}

// unpricedInfo is the information an item composes when its stored price is
// 0: the lines the original composes for it, which state no price.
func unpricedInfo(item sim.ItemInstance, f *FrontEnd, resolve weaponDamageResolver, words ui.Words) []string {
	item = item.Clone()
	item.Price = 0
	return itemInstanceInfoLinesWithWeaponDamage(item, f.Table, resolve, words)
}

// A new campaign gives the created hero the installed access item, whose
// MagicItems row stores price -1 (DAT-DOC-021). Hovering its pack cell draws
// its installed name alone. Each worn item and each item taken off into the
// pack keeps a positive stored price and draws the same lines it draws at price
// 0, none of them a price line (ITEM-PRICETAG-144, DIV-1461). After the hovers
// the stored prices are unchanged, and the access item's T1C in a mission SAVE
// is still -1.
func TestReleaseItemInformationHasNoPriceLine(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	code := uint16(data.QuestDocumentCode)
	row := int(uint8(code))
	if params := f.Table.MagicItems.EntryParams(row); len(params) == 0 || params[0] != -1 {
		t.Fatalf("MagicItems[%d] %q params %v, want price -1", row, f.Table.MagicItems.EntryName(row), params)
	}
	labels := priceLabels(f)
	app := f.App("item information no price line")
	app.Layout(640, 480)
	app.SetTooltipDelayPreference(0, nil)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenPicker {
		if err := app.HeadlessActivate("@first"); err != nil {
			t.Fatal(err)
		}
	}
	if err := headlessCreateCharacter(app, HeadlessCharacter{Name: "Witness"}); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap || f.live == nil || !f.live.invSubjectSet {
		t.Fatalf("character creation ended on %s", app.Screen())
	}
	hero := sim.EntityID(f.live.invSubject.ID)
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	words := f.live.view.Words()
	// The first mission opens a dialogue a few ticks in. It hides every popup and
	// stops the world until dismissed, so each open notice is answered with the
	// Enter key it reads, and the frames run until the opening one has been.
	dismiss := func() bool {
		t.Helper()
		closed := false
		for i := 0; app.HeadlessNoticeOpen(); i++ {
			if i == 16 {
				t.Fatal("a notice stays open after 16 dismissals")
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			closed = true
		}
		return closed
	}
	for i := 0; i < 40 && !dismiss(); i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	// The hover writes the drawn popup to AGAINROM_TOOLTIP_ARTIFACTS when that
	// is set, then requires it to be exactly the expected lines' picture.
	hover := func(name string, x, y int, lines []string) {
		t.Helper()
		dismiss()
		if err := app.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		state, pic := app.HeadlessTooltip()
		if pic != nil {
			tooltipReleasePNG(t, "item-price-"+name, pic)
		}
		want, _, ok := ui.ComposeTooltipHint(lines, f.tipFont(), image.Pt(x, y), image.Rect(0, 0, 640, 480), f.HoverBall())
		if !state.Visible || pic == nil || !ok || pic.Bounds().Size() != want.Bounds().Size() || !bytes.Equal(pic.Pix, want.Pix) {
			t.Fatalf("%s popup (visible %v) is not the drawn lines %q", name, state.Visible, lines)
		}
	}

	stacks, _ := f.live.world.CarriedStacks(hero)
	doc := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == code })
	if doc < 0 || stacks[doc].Price != -1 {
		t.Fatalf("hero pack %+v, want the access item at stored price -1", stacks)
	}
	name := itemName(data.QuestDocumentCode, f.Table)
	x, y, err := app.HeadlessPackCellPoint(doc)
	if err != nil {
		t.Fatal(err)
	}
	hover("access", x, y, []string{name})
	if info := f.live.invSubject.PackInfo[doc]; !slices.Equal(info, []string{name}) {
		t.Fatalf("access item information %q, want [%q] and no price line", info, name)
	}
	t.Logf("%s: stored price -1, information %q", f.Table.MagicItems.EntryName(row), name)

	worn, _ := f.live.world.EquippedItems(hero)
	var prices []int32
	hovered, takeoff := 0, -1
	for i, item := range worn {
		if item.Empty() || item.Price <= 0 {
			continue
		}
		want := unpricedInfo(item, f, f.live.itemWeaponDamage, words)
		info := f.live.invSubject.SlotInfo[i]
		if len(info) == 0 || info[0] != itemName(data.ItemCode(item.Code), f.Table) ||
			!slices.Equal(info, want) || statesPrice(info, labels) {
			t.Fatalf("worn slot %d priced %d: information %q, want the lines %q and no price line", i+1, item.Price, info, want)
		}
		if x, y, err := app.HeadlessDollSlotPoint(i + 1); err == nil {
			hover(fmt.Sprintf("slot%d", i+1), x, y, want)
			hovered++
			if i > 0 {
				takeoff = i
			}
		}
		prices = append(prices, item.Price)
		t.Logf("worn slot %d: stored price %d, information %q", i+1, item.Price, info)
	}
	if hovered == 0 {
		t.Fatal("no worn item with a positive price could be hovered")
	}
	if takeoff < 0 {
		t.Fatal("no worn armour with a positive price could be taken off")
	}

	// A press on a worn slot takes the item off into the pack; the next
	// frames run the command. Its pack cell describes it without a price line.
	taken := worn[takeoff]
	x, y, err = app.HeadlessDollSlotPoint(takeoff + 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	cell := -1
	for step := 0; step < 6 && cell < 0; step++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismiss()
		stacks, _ = f.live.world.CarriedStacks(hero)
		cell = slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == taken.Code && s.Price == taken.Price })
	}
	if cell < 0 {
		t.Fatalf("slot %d item %+v did not reach the pack: %+v", takeoff+1, taken, stacks)
	}
	want := unpricedInfo(stacks[cell].Instance(), f, f.live.itemWeaponDamage, words)
	if info := f.live.invSubject.PackInfo[cell]; !slices.Equal(info, want) || statesPrice(info, labels) {
		t.Fatalf("pack cell %d priced %d: information %q, want the lines %q and no price line", cell, taken.Price, info, want)
	}
	if x, y, err := app.HeadlessPackCellPoint(cell); err != nil {
		t.Fatal(err)
	} else {
		hover("pack", x, y, want)
	}
	t.Logf("pack cell %d: stored price %d, information %q", cell, taken.Price, want)

	stacks, _ = f.live.world.CarriedStacks(hero)
	if doc >= len(stacks) || stacks[doc].Code != code || stacks[doc].Price != -1 {
		t.Fatalf("hero pack after the hovers %+v, want the access item still at -1", stacks)
	}
	if got := stacks[cell].Price; got != taken.Price {
		t.Fatalf("pack cell %d stored price %d after the hovers, want %d", cell, got, taken.Price)
	}
	worn, _ = f.live.world.EquippedItems(hero)
	for i, item := range worn {
		if i != takeoff && !item.Empty() && item.Price > 0 && !slices.Contains(prices, item.Price) {
			t.Fatalf("worn slot %d stored price %d changed by the hovers, was one of %v", i+1, item.Price, prices)
		}
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	written, _, err := f.playerMissionSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := sav.DecodeDocumentData(written)
	if err != nil {
		t.Fatal(err)
	}
	var accessPrices []int32
	for i := range saved.Objects {
		record := &saved.Objects[i]
		if !savedItemClass(record.Class) {
			continue
		}
		c, codeErr := savedStructureValue(record, "F40")
		price, priceErr := savedStructureValue(record, "T1C")
		if codeErr == nil && priceErr == nil && c == uint32(code) {
			accessPrices = append(accessPrices, int32(price))
		}
	}
	if !slices.Equal(accessPrices, []int32{-1}) {
		t.Fatalf("mission SAVE writes access item T1C %v, want [-1]", accessPrices)
	}
}

// shopNumberDrawn reports whether the composed shop screen draws the grouped
// figure of price in the cell at r: inside and just around the figure box, the
// pixels that differ from the same screen composed without any text are exactly
// the pixels the price font paints for that figure and for its shadow one pixel
// right and down, and there is at least one.
func shopNumberDrawn(v ui.ShopScreenView, cell ui.ShopCell, r image.Rectangle, price int32) bool {
	font := v.PriceFont
	if font == nil {
		font = v.Font
	}
	if font == nil {
		return false
	}
	_, figure := ui.ShopPricePlacement(v.Art, font, cell, r)
	if figure.Empty() {
		return false
	}
	with := ui.ComposeShopScreen(v, image.Point{}, false, nil, false)
	bare := v
	bare.Font, bare.PriceFont, bare.Character.Font = nil, nil, nil
	without := ui.ComposeShopScreen(bare, image.Point{}, false, nil, false)
	marked := image.NewRGBA(without.Bounds())
	copy(marked.Pix, without.Pix)
	mark := color.RGBA{R: 0xff, B: 0xff, A: 0xff}
	font.DrawFlat(marked, ui.GroupDigits(int64(price)), figure.Min.X+1, figure.Min.Y+1, mark)
	font.Draw(marked, ui.GroupDigits(int64(price)), figure.Min.X, figure.Min.Y, mark)
	painted := 0
	for y := figure.Min.Y - 2; y < figure.Max.Y+3; y++ {
		for x := figure.Min.X - 2; x < figure.Max.X+3; x++ {
			drawn := with.RGBAAt(x, y) != without.RGBAAt(x, y)
			if drawn != (marked.RGBAAt(x, y) != without.RGBAAt(x, y)) {
				return false
			}
			if drawn {
				painted++
			}
		}
	}
	return painted > 0
}

// A town shop states no price in any item information either: each of the
// merchant's four shelves, the table, the shown member's pack and his doll
// draw an item's own lines and no price line (ITEM-PRICETAG-144). Every grid
// still draws its cell's number, the shelf's and the table's at the full price
// and the pack's at half of it, and the stored prices are unchanged. The
// hit-test scan behind each surface's point walks the whole frame, so every
// point is found once.
func TestReleaseShopItemInformationHasNoPriceLine(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	s.CloseTip()
	app.SetTooltipDelayPreference(0, nil)
	labels := priceLabels(f)
	bounds := image.Rect(0, 0, 640, 480)
	hover := func(surface string, p image.Point, info []string, item sim.ItemInstance) {
		t.Helper()
		if item.Price <= 0 {
			t.Fatalf("%s: witness item %+v is not priced", surface, item)
		}
		want := unpricedInfo(item, f, nil, f.Words)
		if len(info) == 0 || !slices.Equal(info, want) || statesPrice(info, labels) {
			t.Fatalf("%s priced %d: information %q, want the lines %q and no price line", surface, item.Price, info, want)
		}
		if err := app.HeadlessPointer("hover", p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		state, pic := app.HeadlessTooltip()
		if pic != nil {
			tooltipReleasePNG(t, "shop-price-"+surface, pic)
		}
		drawn, _, ok := ui.ComposeTooltipHint(want, f.tipFont(), p, bounds, f.HoverBall())
		if !state.Visible || pic == nil || !ok || pic.Bounds().Size() != drawn.Bounds().Size() || !bytes.Equal(pic.Pix, drawn.Pix) {
			t.Fatalf("%s popup (visible %v) is not the drawn lines %q", surface, state.Visible, want)
		}
	}
	found := map[string]image.Point{}
	point := func(kind string, index int) (image.Point, error) {
		key := fmt.Sprintf("%s %d", kind, index)
		if p, ok := found[key]; ok {
			return p, nil
		}
		x, y, err := app.HeadlessShopPoint(kind, index)
		if err != nil {
			return image.Point{}, err
		}
		found[key] = image.Pt(x, y)
		return found[key], nil
	}
	// click presses and releases the pointer on the shop point kind, index.
	click := func(kind string, index int) {
		t.Helper()
		p, err := point(kind, index)
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}

	member := s.shopMemberIndex()
	view := s.ShopScreen()
	dollHovered := 0
	for n := 1; n <= sim.EquipSlots; n++ {
		item, ok := s.shopEquippedItem(member, n)
		if !ok || item.Empty() || item.Price <= 0 {
			continue
		}
		if p, err := point("doll", n); err == nil {
			hover(fmt.Sprintf("doll%d", n), p, view.SlotInfo[n-1], item)
			dollHovered++
		}
	}
	if dollHovered == 0 {
		t.Fatal("no worn item with a positive price could be hovered on the shop doll")
	}

	shelfHovered := 0
	for pick := range shopRoomShelves {
		click("shelf_pick", pick)
		if s.shopChosen != pick {
			t.Fatalf("shelf pick %d chose %d", pick, s.shopChosen)
		}
		items := f.Shop.Shelf(shopRoomShelves[pick].shelf)
		view = s.ShopScreen()
		seen := 0
		for c := range view.Shelf {
			k := s.shelfBase + c
			if k >= len(items) || items[k].Price <= 0 || seen == 2 {
				continue
			}
			seen++
			shelfHovered++
			p, err := point("shelf", c)
			if err != nil {
				t.Fatal(err)
			}
			hover(fmt.Sprintf("shelf%d-%d", pick, c), p, view.Shelf[c].Info, items[k].Instance())
			if view.Shelf[c].Price != items[k].Price || !shopNumberDrawn(view, view.Shelf[c], ui.ShopShelfCellRect(c), items[k].Price) {
				t.Fatalf("shelf %d cell %d: grid number %d, want the stored price %d drawn", pick, c, view.Shelf[c].Price, items[k].Price)
			}
		}
	}
	if shelfHovered == 0 {
		t.Fatal("no shelf item with a positive price could be hovered")
	}

	// A click on a shelf cell puts the item on the merchant's table.
	click("shelf_pick", 0)
	items := f.Shop.Shelf(shopRoomShelves[0].shelf)
	view = s.ShopScreen()
	c := slices.IndexFunc(view.Shelf[:], func(cell ui.ShopCell) bool { return cell.Occupied() && cell.Price > 0 })
	if c < 0 || s.shelfBase+c >= len(items) {
		t.Fatalf("shelf 0 has no priced cell in view: %+v", view.Shelf)
	}
	click("shelf", c)
	table := f.Shop.Table()
	if len(table) == 0 || table[0].Mine || table[0].Price <= 0 {
		t.Fatalf("table after the shelf click %+v, want a merchant item at a positive price", table)
	}
	view = s.ShopScreen()
	p, err := point("table", 0)
	if err != nil {
		t.Fatal(err)
	}
	hover("table", p, view.Table[0].Info, table[0].Instance())
	if view.Table[0].Price != table[0].Price || !shopNumberDrawn(view, view.Table[0], ui.ShopTableCellRect(0), table[0].Price) {
		t.Fatalf("table cell 0: grid number %d, want the stored price %d drawn", view.Table[0].Price, table[0].Price)
	}

	// Taking a worn item off with a tap on the doll puts it in the pack, where
	// the grid draws half of its price and its information states none.
	var taken sim.ItemInstance
	slot := 0
	for n := 1; n <= sim.EquipSlots && slot == 0; n++ {
		if item, ok := s.shopEquippedItem(member, n); ok && !item.Empty() && item.Price > 0 {
			if _, err := point("doll", n); err == nil {
				taken, slot = item, n
			}
		}
	}
	if slot == 0 {
		t.Fatal("no worn item with a positive price could be taken off")
	}
	click("doll", slot)
	stacks := s.shopPackStacks()
	k := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == taken.Code && st.Price == taken.Price })
	if k < 0 {
		t.Fatalf("doll slot %d item %+v did not reach the pack: %+v", slot, taken, stacks)
	}
	view = s.ShopScreen()
	i := k + 1 - s.packBase
	if i < 0 || i >= len(view.Pack) {
		t.Fatalf("pack stack %d is outside the strip at base %d", k, s.packBase)
	}
	if p, err = point("pack", i); err != nil {
		t.Fatal(err)
	}
	hover("pack", p, view.Pack[i].Info, stacks[k].Instance())
	half := shopHalfUp(stacks[k].Price)
	if view.Pack[i].Price != half || !shopNumberDrawn(view, view.Pack[i], ui.ShopPackCellRect(i), half) {
		t.Fatalf("pack cell %d: grid number %d, want half of the stored price %d (%d) drawn", i, view.Pack[i].Price, stacks[k].Price, half)
	}
	if shopNumberDrawn(view, view.Pack[i], ui.ShopPackCellRect(i), half+1) {
		t.Fatalf("pack cell %d: the figure check accepts %d, which the grid does not draw", i, half+1)
	}
	if stacks[k].Price != taken.Price {
		t.Fatalf("pack stack stored price %d, want %d", stacks[k].Price, taken.Price)
	}
	t.Logf("%d shelf items, table, doll and pack hovered; pack cell %d stored price %d draws %d", shelfHovered, i, taken.Price, half)
}
