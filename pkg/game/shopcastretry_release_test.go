package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseCheckshopStaffStockSurvivesPurchaseAndCurrentSAV(t *testing.T) {
	source := os.Getenv("AGAINROM_CHECKSHOP")
	if source == "" {
		t.Skip("AGAINROM_CHECKSHOP is not set")
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	const hash = "a92bf144333cca608cc5d8257cbb551a39fed089f5c9b745f6693c66bf6d1abc"
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != hash {
		t.Fatal("checkshop input identity differs")
	}
	f := shopOrderFront(t)
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("staff stock")
	app.Layout(640, 480)
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(source)}, nil))
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	s := shopOrderEnter(t, f, app)
	if f.Town.Chapter() != 30 || f.Shop.Ceiling() != 1000 || f.Town.Gold() != 4103 {
		t.Fatal("wrong loaded campaign")
	}
	room := shopOrderRoom(t, ShelfMagic)
	stock := shopOrderShelf(t, app, s, room, "original LOAD")
	units := int32(0)
	for _, item := range stock {
		spell, power, cast := item.Instance().CastSpell()
		if !cast || spell != 1 || power <= 0 || item.Price <= 0 || item.Price > 2000 {
			t.Fatalf("invalid low-ceiling staff %+v", item)
		}
		units += item.Count
		t.Logf("staff %#x power %d price %d count %d", item.Code, power, item.Price, item.Count)
	}
	if len(stock) < 2 || units < 2 {
		t.Fatalf("magic shelf has %d rows/%d units, want multiple affordable staffs", len(stock), units)
	}
	bought := stock[0]
	count := func(front *FrontEnd) int {
		n := 0
		for _, member := range front.Carried {
			for _, item := range mapload.MemberCarriedItems(member, front.Table) {
				if item.Code == uint16(bought.Code) && item.Price == bought.Price && reflect.DeepEqual(item.Effects, bought.Effects) {
					n++
				}
			}
		}
		return n
	}
	before, gold := count(f), f.Town.Gold()
	cell := shopOrderShow(t, app, s, room, 0)
	point := shopPointer(t, app)
	point("shelf", cell, "press", "release")
	point("button", 1, "press", "release")
	if count(f) != before+1 || gold-f.Town.Gold() != int(bought.Price) {
		t.Fatal("purchase changed cast, price, quantity or purse")
	}
	name := shopOrderF2Save(t, app, store)
	g := shopOrderFront(t)
	cold := g.App("staff stock cold")
	cold.Layout(640, 480)
	save, list, load := g.SaveSeams(store, OriginalStore{}, nil)
	cold.SetSaveSeams(save, list, load)
	shopOrderLoad(t, cold, list, name)
	if count(g) != before+1 || g.Town.Gold() != f.Town.Gold() {
		t.Fatal("current SAV cold LOAD changed purchased staff or purse")
	}
	shopOrderShelf(t, cold, shopOrderEnter(t, g, cold), room, "current SAV cold LOAD")
	if count(g) != before+1 {
		t.Fatal("post-LOAD shop entry lost the purchased staff")
	}
}

func TestReleaseGeneratedStaffPriceReachesPurchaseAndCurrentSAV(t *testing.T) {
	source := os.Getenv("AGAINROM_CHECKSHOP")
	if source == "" {
		t.Skip("AGAINROM_CHECKSHOP is not set")
	}
	raw, err := os.ReadFile(source)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != "a92bf144333cca608cc5d8257cbb551a39fed089f5c9b745f6693c66bf6d1abc" {
		t.Fatalf("checkshop input identity: %v", err)
	}
	f := shopOrderFront(t)
	store := SaveStore{Dir: t.TempDir()}
	app := f.App("generated staff price")
	app.Layout(640, 480)
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(source)}, nil))
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	s := shopOrderEnter(t, f, app)
	var candidate data.ShopCandidate
	for _, c := range shopWeaponPool(f.Table, 1000) {
		if c.Code == 0x810d {
			candidate = c
		}
	}
	if candidate.Code == 0 || candidate.Price != 167 || !candidate.ForcedCast || f.Table.Spells.EntryParams(1)[20] != 50 {
		t.Fatalf("installed staff inputs changed: %+v", candidate)
	}
	generated, ok := shopEnchantedItem(candidate, 1000, f.Table, &fixedDraws{values: []int{0, 9}})
	wantEffect := []sim.ItemEffect{{Kind: 41, Operand: 0x000a0001}}
	if !ok || generated.Price != 1659 || generated.Count != 1 || !reflect.DeepEqual(generated.Effects, wantEffect) {
		t.Fatalf("generated Fire Arrow power 10 = %+v ok=%v, want stored price 1659", generated, ok)
	}
	member := s.shopPartyMember(s.shopMemberIndex())
	if member == nil {
		t.Fatal("no purchase owner")
	}
	owner := member.ID
	loadedWorn := mapload.MemberItemEquipment(*member, f.Table)
	count := func(front *FrontEnd) int {
		for _, p := range front.Carried {
			if p.ID != owner {
				continue
			}
			n := 0
			for _, item := range mapload.MemberCarriedItems(p, front.Table) {
				if item.Code == 0x810d && item.Price == 1659 && reflect.DeepEqual(item.Effects, wantEffect) {
					n++
				}
			}
			return n
		}
		t.Fatal("purchase owner missing")
		return -1
	}
	before, gold := count(f), f.Town.Gold()
	f.Shop.shelves[ShelfMagic] = []ShopItem{generated}
	cell := shopOrderShow(t, app, s, shopOrderRoom(t, ShelfMagic), 0)
	if s.ShopScreen().Shelf[cell].Price != 1659 {
		t.Fatal("merchant plaque has the wrong price")
	}
	point := shopPointer(t, app)
	point("shelf", cell, "press", "release")
	if s.ShopScreen().Buy != 1659 {
		t.Fatal("buy quote has the wrong price")
	}
	point("button", 1, "press", "release")
	if count(f) != before+1 || gold-f.Town.Gold() != 1659 {
		t.Fatal("purchase changed owner, quantity, effect or debit")
	}
	name := shopOrderF2Save(t, app, store)
	saved, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	doc, actions, _ := cityProjectionWire(t, saved)
	var actor uint16
	for _, p := range actions.Party {
		if string(p.ID) == owner {
			for _, binding := range actions.Bindings {
				if !binding.Structure && binding.ID == p.Entity {
					actor = binding.Object
				}
			}
		}
	}
	if actor == 0 || int(actor) > len(doc.Objects) || doc.Objects[actor-1].Class != "Human" {
		t.Fatal("saved buyer has no ordinary Human")
	}
	refs := startingStaffRefs(doc.Objects[actor-1], "Inventory")
	matched := 0
	for _, ref := range refs {
		item, err := savedItemRecord(&doc, ref)
		if err != nil {
			t.Fatal(err)
		}
		if item.Value.Code != 0x810d || item.Value.Price != 1659 || !reflect.DeepEqual(item.Value.Effects, wantEffect) {
			continue
		}
		record := doc.Objects[ref-1]
		if startingStaffScalar(record, "F42") != 1 || startingStaffScalar(record, "F44") != 2 || startingStaffScalar(record, "T1C") != 1659 {
			t.Fatalf("ordinary SAV staff fields = %+v", record.Values)
		}
		effects := startingStaffRefs(record, "Effects")
		if len(effects) != 1 {
			t.Fatal("ordinary SAV effect count", effects)
		}
		e := doc.Objects[effects[0]-1]
		if e.Class != "Effect" || startingStaffScalar(e, "E3C") != 41 || startingStaffScalar(e, "E3D") != 0 || startingStaffScalar(e, "E40") != 0x000a0001 {
			t.Fatalf("ordinary SAV cast = %+v", e)
		}
		matched++
	}
	if matched != before+1 {
		t.Fatalf("saved buyer holds %d exact staffs, want %d", matched, before+1)
	}
	g := shopOrderFront(t)
	cold := g.App("generated staff cold")
	cold.Layout(640, 480)
	save, list, load := g.SaveSeams(store, OriginalStore{}, nil)
	cold.SetSaveSeams(save, list, load)
	shopOrderLoad(t, cold, list, name)
	if count(g) != before+1 || g.Town.Gold() != gold-1659 {
		t.Fatal("cold SAV LOAD changed price, effect, owner, quantity or purse")
	}
	for _, p := range g.Carried {
		if p.ID != owner {
			continue
		}
		worn := mapload.MemberItemEquipment(p, g.Table)
		for i, item := range loadedWorn {
			if item.Code != worn[i].Code || item.Price != worn[i].Price || !reflect.DeepEqual(item.Effects, worn[i].Effects) {
				t.Fatal("LOAD repriced an existing original equipped item")
			}
		}
	}
	next := shopOrderEnter(t, g, cold)
	pack := -1
	for i, stack := range next.shopPackStacks() {
		if stack.Code == 0x810d && stack.Price == 1659 && reflect.DeepEqual(stack.Effects, wantEffect) {
			pack = i
			break
		}
	}
	if pack < 0 {
		t.Fatal("next shop lost the purchased staff")
	}
	saleGold := g.Town.Gold()
	shopOrderSell(t, cold, next, pack)
	if g.Town.Gold()-saleGold != 830 || count(g) != before {
		t.Fatal("loaded staff sale changed the stored-price quote or quantity")
	}
}
