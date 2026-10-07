package game

import (
	"bytes"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The installed MagicItems strings, not the runtime item's output, are the
// independent source for this producer check.
func TestReleaseConsumable1090InstalledRows(t *testing.T) {
	f := releaseFront(t)
	rows := f.Table.MagicItems
	want := []sim.ItemEffect{
		{Kind: 16, Mode: 1, Operand: 50 | 480<<16},
		{Kind: 2, Operand: 1}, {Kind: 4, Operand: 1}, {Kind: 3, Operand: 1}, {Kind: 5, Operand: 1},
		{Kind: 8, Mode: 1, Operand: 100 | 960<<16}, {Kind: 6, Operand: 30}, {Kind: 6, Operand: 100},
		{Kind: 11, Mode: 1, Operand: 100 | 960<<16}, {Kind: 9, Operand: 30}, {Kind: 9, Operand: 100},
		{Kind: 8, Mode: 1, Operand: 250 | 1920<<16}, {Kind: 11, Mode: 1, Operand: 250 | 1920<<16},
	}
	count := 0
	for i := 1; i < rows.Len(); i++ {
		if !strings.HasPrefix(rows.EntryName(i), "Potion") {
			continue
		}
		count++
		item := mapload.ItemInstanceFromCode(uint16(0x0e00|i), f.Table)
		if i < 1 || i > len(want) || item.Kind != 3 || !reflect.DeepEqual(item.Effects, []sim.ItemEffect{want[i-1]}) || item.Price != rows.EntryParams(i)[0] {
			t.Fatalf("row %d %s raw=%q item=%+v", i, rows.EntryName(i), rows.EntryStrings(i), item)
		}
	}
	if count != 13 {
		t.Fatalf("installed Potion population=%d, want 13", count)
	}
}

func usePack1090(t *testing.T, app *ui.App, cell int) {
	usePackCell(t, app, cell)
}

func usePackCell(t *testing.T, app *ui.App, cell int) {
	t.Helper()
	for n := 0; n < 16 && app.HeadlessNoticeOpen(); n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	x, y, err := app.HeadlessPackCellPoint(cell)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release", "press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func shopPointer1090(t *testing.T, app *ui.App, kind string, index int, edges ...string) {
	t.Helper()
	x, y, err := app.HeadlessShopPoint(kind, index)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range edges {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReleaseConsumable1090AppTownSourceUseAndContinuation(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	potions := shopPotionStock(f.Table, rand.New(rand.NewSource(1090)))
	scrolls := shopScrollPool(f.Table, 100000, rand.New(rand.NewSource(1090)))
	if len(potions) != 6 || len(scrolls) != 27 {
		t.Fatalf("producer population potions=%d scrolls=%d", len(potions), len(scrolls))
	}
	// Deterministic display fixture, using the real installed producers. All
	// purchase/use mutations below pass through App pointer dispatch.
	f.Shop.shelves[ShelfBooks] = []ShopItem{potions[0], scrolls[0]}
	f.Town.gold = 100000
	shopPointer1090(t, app, "shelf_pick", 3, "press", "release")
	if s.ShopScreen().Shelf[0].Icon == nil {
		t.Fatal("potion source has no installed art")
	}
	beforeGold, beforeQty := f.Town.Gold(), f.Shop.Shelf(ShelfBooks)[0].Count
	at := releaseShopBuyToPack(t, app, f, uint16(potions[0].Code))
	x, y := at.X, at.Y
	if err := app.HeadlessPointer("press", x, y); err != nil {
		t.Fatal(err)
	}
	dx, dy, err := app.HeadlessShopPoint("doll_box", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"move", "release"} {
		if err := app.HeadlessPointer(edge, dx, dy); err != nil {
			t.Fatal(err)
		}
	}
	p := s.shopPartyMember(s.shopMemberIndex())
	if p.PotionEffect == nil || p.PotionEffect.Kind != sim.EffectHealthRegeneration || p.PotionEffect.Magnitude != 100 || p.PotionEffect.Remaining != 960 || f.Shop.Shelf(ShelfBooks)[0].Count != beforeQty-1 || f.Town.Gold() != beforeGold-int(potions[0].Price) {
		t.Fatalf("town use effect=%+v gold=%d qty=%d", p.PotionEffect, f.Town.Gold(), f.Shop.Shelf(ShelfBooks)[0].Count)
	}
	// Shelf -> table -> pack is purchase, not use. Scroll gesture in the town
	// remains a non-consuming refusal while its target-map route is Unknown.
	shopPointer1090(t, app, "shelf", 1, "press", "release")
	shopPointer1090(t, app, "button", 1, "press", "release")
	items := s.shopPackStacks()
	index := -1
	for i, item := range items {
		if item.Kind == 4 {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("purchased scroll absent from pack")
	}
	before := append([]sim.ItemStack(nil), items...)
	known := p.KnownSpells
	gold := f.Town.Gold()
	shopPointer1090(t, app, "pack", index+1, "press", "release", "press", "release")
	held := f.Shop.Table()
	if len(held) != 1 || !held[0].Mine || held[0].Count != 1 || !sim.ItemEqual(held[0].Instance(), before[index].Instance()) || held[0].Price != before[index].Price || p.KnownSpells != known || f.Town.Gold() != gold {
		t.Fatalf("town scroll refusal lost or learned the staged unit: before=%+v held=%+v", before, held)
	}
	kept := append(append([]sim.ItemStack(nil), before[:index]...), before[index+1:]...)
	if !reflect.DeepEqual(kept, s.shopPackStacks()) {
		t.Fatalf("town scroll refusal changed another pack item: want=%+v after=%+v", kept, s.shopPackStacks())
	}
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	loaded, _, err := DecodeSave(disk)
	if err != nil {
		t.Fatal(err)
	}
	back := releaseFront(t)
	_, town, err := back.Restore(loaded)
	if err != nil || !town {
		t.Fatalf("town restore=%v %v", town, err)
	}
	if !reflect.DeepEqual(back.Carried[0].PotionEffect, p.PotionEffect) {
		t.Fatal("town native timer changed")
	}
	mission := back.App("1090-town-continuation")
	mission.Layout(1024, 768)
	if err := mission.OpenMission(back.MissionOpener(back.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	effects := back.live.world.ActiveEffects()
	if len(effects) != 1 || effects[0].Kind != sim.EffectHealthRegeneration || effects[0].Remaining != 960 {
		t.Fatalf("town to mission effects=%+v", effects)
	}
	potionNative1090(t, back, "town-to-mission")
}

func TestReleaseConsumable1090AppPermanentTimedAndScroll(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	pool := shopScrollPool(f.Table, 100000, rand.New(rand.NewSource(1090)))
	var scroll sim.ItemInstance
	for _, candidate := range pool {
		id, _, ok := sim.ScrollSpell(candidate.Instance())
		if ok && id == 1 {
			scroll = candidate.Instance()
			break
		}
	}
	if scroll.Code == 0 {
		t.Fatal("installed shelf did not construct Fire Arrow scroll")
	}
	if scroll.Kind != 4 || len(scroll.Effects) != 1 || scroll.Effects[0].Kind != 41 || uint16(scroll.Effects[0].Operand) != 1 || int(scroll.Effects[0].Operand>>16) < 1 || int(scroll.Effects[0].Operand>>16) > 100 {
		t.Fatalf("scroll=%+v", scroll)
	}
	basePrice := f.Table.Spells.EntryParams(1)[20]
	if scroll.Price != basePrice*(int32(scroll.Effects[0].Operand>>16)/10+1) {
		t.Fatal("scroll price did not use power")
	}
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	party[0].KnownSpells = 0
	party[0].Carried, party[0].CarriedItems = nil, nil
	for _, code := range []uint16{0xe02, 0xe06, 0xe09} {
		party[0].CarriedItems = append(party[0].CarriedItems, mapload.ItemInstanceFromCode(code, f.Table))
		party[0].Carried = append(party[0].Carried, code)
	}
	party[0].CarriedItems = append(party[0].CarriedItems, scroll, scroll)
	party[0].Carried = append(party[0].Carried, scroll.Code, scroll.Code)
	startingBody := party[0].Hero.Body
	f.Carried = party
	app := f.App("1090-full-consumables")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 32; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
	}
	live := f.live
	id := live.mission.ids[0]
	e, _ := live.entity(id)
	live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	potionNative1090(t, f, "before-permanent")
	usePack1090(t, app, 0)
	for n := 0; n < 32; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
		if e, _ := live.entity(id); e.PotionStats[0] == 1 {
			break
		}
	}
	after, _ := live.entity(id)
	if after.PotionStats[0] != 1 || after.MaxHP <= e.MaxHP || live.chars[id].Body != int(startingBody+1) {
		items, _ := live.world.CarriedItems(id)
		t.Fatalf("permanent live derive: hp=%d->%d gains=%v headroom=%v panelBody=%d baseBody=%d items=%+v pending=%+v", e.MaxHP, after.MaxHP, after.PotionStats, after.PotionHeadroom, live.chars[id].Body, startingBody, items, live.pending)
	}
	potionNative1090(t, f, "after-permanent")
	usePack1090(t, app, 0)
	for n := 0; n < 32; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
		if len(live.world.ActiveEffects()) > 0 {
			break
		}
	}
	first := live.world.ActiveEffects()
	if len(first) != 1 || first[0].Spell != 0 || first[0].Kind != sim.EffectHealthRegeneration || first[0].Magnitude != 100 {
		t.Fatalf("first timed=%+v", first)
	}
	potionNative1090(t, f, "during-timed")
	usePack1090(t, app, 0)
	for n := 0; n < 32; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
		if items, _ := live.world.CarriedItems(id); len(items) == 2 {
			break
		}
	}
	repeated := live.world.ActiveEffects()
	if len(repeated) != 1 || repeated[0].Kind != sim.EffectHealthRegeneration || repeated[0].Remaining < first[0].Remaining {
		t.Fatalf("cross-kind replacement=%+v before=%+v", repeated, first)
	}
	if live.invSubject.Pack[0] == nil {
		t.Fatal("installed scroll art absent")
	}
	// Arming then right cancelling is display-only, not reservation.
	beforeItems, _ := live.world.CarriedItems(id)
	usePack1090(t, app, 0)
	e, _ = live.entity(id)
	live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	x, y, err := app.HeadlessEntityPoint(uint32(id))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"right-press", "right-release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	afterItems, _ := live.world.CarriedItems(id)
	if !reflect.DeepEqual(beforeItems, afterItems) || len(live.world.ScrollCasts()) != 0 {
		t.Fatal("arming cancellation spent item")
	}
	// Use the actual map pointer for the target. A fighter with an empty book
	// casts the installed scroll at himself, a stable source/target witness.
	usePack1090(t, app, 0)
	e, _ = live.entity(id)
	live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	x, y, err = app.HeadlessEntityPoint(uint32(id))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; n < 8 && len(live.world.ScrollCasts()) == 0; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.world.ScrollCasts()) != 1 {
		t.Fatal("map pointer did not reserve scroll")
	}
	potionNative1090(t, f, "scroll-reserved")
	before, _ := live.entity(id)
	for n := 0; n < 96 && len(live.world.ScrollCasts()) != 0; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
	}
	// The sole transport hands off on the first tick; its Point runs next.
	released, _ := live.entity(id)
	if released.HP != before.HP || live.world.PendingSpellDeliveries() != 1 {
		t.Fatalf("scroll release applied early: hp%d->%d deliveries%d", before.HP, released.HP, live.world.PendingSpellDeliveries())
	}
	potionNative1090(t, f, "scroll-in-flight")
	releaseTick := live.world.Tick()
	for n := 0; n < 8 && live.world.PendingSpellDeliveries() != 0; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
		if live.world.Tick() == releaseTick+1 {
			current, _ := live.entity(id)
			pending := live.world.NativeSpellDeliverySaveStates()
			if current.HP != before.HP || len(pending) != 1 || !pending[0].Released {
				t.Fatal("self-targeted scroll lost its undamaged handoff tick")
			}
		}
	}
	if live.world.Tick() != releaseTick+2 || live.world.PendingSpellDeliveries() != 0 {
		t.Fatal("self-targeted scroll did not land after the handoff tick")
	}
	after, _ = live.entity(id)
	left, _ := live.world.CarriedItems(id)
	if len(live.world.ScrollCasts()) != 0 || len(left) != len(beforeItems)-1 || after.HP >= before.HP || after.KnownSpells != 0 || after.Mana != before.Mana {
		t.Fatalf("scroll result hp%d->%d items%d->%d caster=%+v reserved=%+v", before.HP, after.HP, len(beforeItems), len(left), after, live.world.ScrollCasts())
	}
	potionNative1090(t, f, "scroll-completed")
}

func potionNative1090(t *testing.T, f *FrontEnd, stage string) {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(disk)
	if err != nil {
		t.Fatal(err)
	}
	restored := releaseFront(t)
	opener, town, err := restored.Restore(decoded)
	if err != nil || town || opener == nil {
		t.Fatalf("%s restore: town=%v err=%v", stage, town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := opener(); err != nil {
		t.Fatal(err)
	}
	form, err := restored.live.world.MarshalBinary()
	if err != nil || !bytes.Equal(form, snapshot.World) {
		t.Fatalf("%s changed on native restore: %v", stage, err)
	}
	var control sim.World
	if err := control.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	for tick := 0; tick < 33; tick++ {
		sim.Step(&control, nil)
		sim.Step(restored.live.world, nil)
		if control.Hash() != restored.live.world.Hash() {
			t.Fatalf("%s diverged at tick %d", stage, tick)
		}
	}
}

func consumableStep1090(app *ui.App) error {
	return stepConsumableApp(app)
}

func stepConsumableApp(app *ui.App) error {
	for n := 0; n < 16 && app.HeadlessNoticeOpen(); n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			return err
		}
	}
	return app.HeadlessStep()
}

func TestReleaseConsumable1090AppInstantPotionAndNative(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	potion := mapload.ItemInstanceFromCode(0x0e07, f.Table)
	party[0].CarriedItems = []sim.ItemInstance{potion, potion}
	party[0].Carried = []uint16{potion.Code, potion.Code}
	f.Carried = party
	app := f.App("1090-instant-potion")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	id := live.mission.ids[0]
	for n := 0; n < 32; n++ {
		if err := consumableStep1090(app); err != nil {
			t.Fatal(err)
		}
	}
	// Controlled damage setup. The subsequent use is real App pack input.
	e, _ := live.entity(id)
	if e.MaxHP <= 60 {
		t.Fatalf("fixture maximum=%d", e.MaxHP)
	}
	headlessDamage(t, live.world, id, e.HP-10)
	sim.Step(live.world, nil)
	live.push()
	live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	for n := 0; n < 16 && app.HeadlessNoticeOpen(); n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if len(live.invSubject.Pack) < 1 || live.invSubject.Pack[0] == nil || live.invSubject.PackCount[0] != 2 {
		items, _ := live.world.CarriedItems(id)
		t.Fatalf("installed potion icon/quantity absent from production pack: pack=%v count=%v items=%+v", live.invSubject.Pack, live.invSubject.PackCount, items)
	}
	pack, err := app.HeadlessInventoryPack()
	if err != nil {
		t.Fatal(err)
	}
	beforePack := append([]byte(nil), pack.Pix...)
	// A cancelled held item is not a use. Cross the drag threshold, cancel
	// with the secondary button, then release the original primary button.
	x, y, err := app.HeadlessPackCellPoint(0)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []struct {
		edge string
		x, y int
	}{
		{"press", x, y}, {"move", x + 30, y - 30},
		{"right-press", x, y}, {"right-release", x, y}, {"release", x, y},
	} {
		if err := app.HeadlessPointer(event.edge, event.x, event.y); err != nil {
			t.Fatal(err)
		}
	}
	if err := consumableStep1090(app); err != nil {
		t.Fatal(err)
	}
	if stacks, _ := live.world.CarriedStacks(id); len(stacks) != 1 || stacks[0].Count != 2 {
		t.Fatalf("cancelled drag spent potion: %+v", stacks)
	}
	potionNative1090(t, f, "before-use")
	for remaining := uint32(1); ; remaining-- {
		for n := 0; n < 16 && app.HeadlessNoticeOpen(); n++ {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
		x, y, err := app.HeadlessPackCellPoint(0)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := live.entity(id)
		for _, edge := range []string{"press", "release", "press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		for tick := 0; tick < 32; tick++ {
			if err := consumableStep1090(app); err != nil {
				t.Fatal(err)
			}
			stacks, _ := live.world.CarriedStacks(id)
			if remaining == 0 && len(stacks) == 0 || remaining == 1 && len(stacks) == 1 && stacks[0].Count == 1 {
				break
			}
		}
		after, _ := live.entity(id)
		stacks, _ := live.world.CarriedStacks(id)
		if after.HP < before.HP+30 || after.KnownSpells != before.KnownSpells || remaining == 0 && len(stacks) != 0 || remaining == 1 && (len(stacks) != 1 || stacks[0].Count != 1) {
			t.Fatalf("remaining=%d hp=%d->%d items=%+v pending=%+v", remaining, before.HP, after.HP, stacks, live.pending)
		}
		potionNative1090(t, f, "after-use")
		pack, err := app.HeadlessInventoryPack()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(pack.Pix, beforePack) {
			t.Fatalf("remaining=%d: composed pack did not change", remaining)
		}
		beforePack = append(beforePack[:0], pack.Pix...)
		if remaining == 0 {
			break
		}
	}
}
