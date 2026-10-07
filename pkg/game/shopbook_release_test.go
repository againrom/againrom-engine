package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseShopBookPurchaseReadAndMissionCarry binds the fourth shelf to
// both lawful installs. It resolves the generic label from the shipped item
// names, generates deterministic stock from installed Spells rows, buys one
// book through the shop controls, reads it through inventory use, and starts
// the next shipped mission with the learned mask.
func TestReleaseShopBookPurchaseReadAndMissionCarry(t *testing.T) {
	f := releaseFront(t)
	// SHOP-CONSUME-073's literal eligible IDs, priced independently from the
	// installed Spells field. The generated pool is not the expected set.
	ids := []uint16{2, 3, 4, 5, 7, 8, 9, 10, 13, 14, 15, 16, 19, 20, 21, 22, 23, 24, 25, 26}
	for _, ceiling := range []int32{1000, 5000, 9_999_999} {
		for _, seed := range []int64{0, 1053} {
			shop := NewShop(ceiling)
			shop.Generate(f.Table, seed)
			seen := map[uint16]int32{}
			for _, item := range shop.Shelf(ShelfBooks) {
				if spell, ok := item.Instance().BookSpell(); ok {
					seen[spell] += item.Count
					params := f.Table.Spells.EntryParams(int(spell))
					if len(params) <= 21 || item.Price != params[21] || item.Price <= 0 || item.Price > ceiling {
						t.Fatalf("ceiling%d seed%d: book%+v lost installed price", ceiling, seed, item)
					}
				}
			}
			for _, spell := range ids {
				params := f.Table.Spells.EntryParams(int(spell))
				if len(params) <= 21 {
					t.Fatalf("installed spell%d has no book price", spell)
				}
				if want := params[21] > 0 && params[21] <= ceiling; (seen[spell] > 0) != want {
					t.Fatalf("ceiling%d seed%d: spell%d count%d price%d", ceiling, seed, spell, seen[spell], params[21])
				}
			}
		}
	}

	label := itemName(shopBookLabelCode, f.Table)
	if label == "" || label == shopBookLabelCode.Name() {
		t.Fatalf("installed generic book name = %q", label)
	}
	for school := 1; school < len(shopBookCodeBySchool); school++ {
		code := shopBookCodeBySchool[school]
		if got := itemName(code, f.Table); got != label {
			t.Fatalf("school %d code %#04x name = %q, want installed generic %q", school, code, got, label)
		}
	}

	// Each school's book picture is the MagicItems row named for the school
	// whose "Protection from" spell the installed Spells rows file under it.
	for spell := 1; spell < f.Table.Spells.Len(); spell++ {
		name := f.Table.Spells.EntryName(spell)
		element, ok := strings.CutPrefix(name, "Protection from ")
		params := f.Table.Spells.EntryParams(spell)
		if !ok || len(params) <= 2 || params[2] <= 0 || int(params[2]) >= len(shopBookCodeBySchool) {
			continue
		}
		code := shopBookCodeBySchool[params[2]]
		if got := strings.ReplaceAll(f.Table.MagicItems.EntryName(int(code)&0xff), "_", " "); got != "Book "+element {
			t.Fatalf("school %d (%s) book code %#04x is row %q, want %q", params[2], name, code, got, "Book "+element)
		}
	}

	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	if len(party) != 1 || !party[0].Mage {
		t.Fatalf("release mage party = %+v", party)
	}
	f.Carried = party
	f.Town = NewTown(f.Campaign.Value())
	f.Town.gold = 9_999_999
	f.Shop = NewShop(9_999_999)
	f.Shop.Generate(f.Table, 1053)
	// The shelf runs cheapest first inside its class and kind (DIV-319), so
	// potions, scrolls and books the mage already knows can stand before the
	// book this buys.
	stock := f.Shop.Shelf(ShelfBooks)
	at := slices.IndexFunc(stock, func(item ShopItem) bool {
		spell, readable := item.Instance().BookSpell()
		return readable && item.Price > 0 && f.Carried[0].KnownSpells&(uint32(1)<<spell) == 0
	})
	if at < 0 {
		t.Fatal("installed Spells rows generated no book the fixture mage lacks")
	}
	want := stock[at]
	spell, _ := want.Instance().BookSpell()

	screen := f.TownScreen().(*townScreen)
	screen.room = roomShop
	if action := screen.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: roomBooks}); screen.ShopScreen().ShelfName != label || action.Msg != "" {
		t.Fatalf("book selector label/message = %q/%q, want installed %q", screen.ShopScreen().ShelfName, action.Msg, label)
	}
	for screen.shelfBase+len(screen.ShopScreen().Shelf) <= at {
		before := screen.shelfBase
		screen.ShopClick(ui.ShopControl{Kind: ui.ShopControlArrowDown})
		if screen.shelfBase == before {
			t.Fatalf("the down arrow stopped at offset %d before book cell %d", before, at)
		}
	}
	if action := screen.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: at - screen.shelfBase}); action.Msg != "" {
		t.Fatalf("staging generated book: %q", action.Msg)
	}
	gold := f.Town.Gold()
	if action := screen.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 1}); action.Msg == "" {
		t.Fatal("buy action returned no result")
	}
	if f.Town.Gold() != gold-int(want.Price) {
		t.Fatalf("gold after book purchase = %d, want %d", f.Town.Gold(), gold-int(want.Price))
	}

	bought := f.Carried[0].CarriedItems
	foundBought := false
	for _, item := range bought {
		if sim.ItemEqual(item, want.Instance()) && item.Price == want.Price {
			foundBought = true
			break
		}
	}
	if !foundBought {
		t.Fatalf("bought pack = %+v, want complete %+v", bought, want.Instance())
	}

	// A native town SAVE used to keep the book's art and price but drop its
	// teaching Effect. Exercise the production save picker before reading it.
	for n := 10; n < 90; n++ {
		if _, ok := f.Campaign.Value().Chapters[n]; ok {
			f.Town.Won(n)
		}
	}
	f.Town.Arrive()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatalf("save purchased book: %q / %v, want native SAV", name, err)
	}
	reloaded := releaseFront(t)
	_, list, load := reloaded.SaveSeams(store, OriginalStore{}, nil)
	entries := list()
	if len(entries) != 1 {
		t.Fatalf("purchased-book saves = %v", entries)
	}
	if open, town, err := load(entries[0].Name); err != nil || !town || open != nil {
		t.Fatalf("reload purchased book: opener=%v town=%v error=%v", open != nil, town, err)
	}
	f.Carried = reloaded.Carried
	foundBought = false
	for _, item := range mapload.MemberCarriedItems(f.Carried[0], f.Table) {
		if item.Code == uint16(want.Code) && item.Kind == want.Kind && item.Price == want.Price && slices.Equal(item.Effects, want.Effects) {
			foundBought = true
		}
	}
	if !foundBought {
		t.Fatalf("native SAVE/LOAD book = %+v, want %+v", mapload.MemberCarriedItems(f.Carried[0], f.Table), want.Instance())
	}

	mission10 := releaseMissionMap(t, f, 10)
	addr10, _ := MissionMap(10)
	ms10, err := StartMissionFrom(mission10, addr10, 10, f.Table, mapload.DifficultyNormal, f.Carried)
	if err != nil {
		t.Fatalf("StartMissionFrom(10): %v", err)
	}
	hero := ms10.Start.IDs[0]
	bookIndex, ok := carriedBookIndex(ms10.World, hero, spell)
	if !ok {
		t.Fatalf("mission 10 hero has no bought spell-%d book", spell)
	}
	mw := openMission(ms10, f.Table, f.Units, worldFixtureViewer(t, mission10),
		f.Archives.Containers, f.Faces, f.NPCFaces)
	mw.enqueueEquip(bookIndex)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindReadBook {
		t.Fatalf("inventory use queued %+v, want KindReadBook", mw.pending)
	}
	mw.tick()
	if got := releaseEntity(t, mw, hero).KnownSpells; got&(uint32(1)<<spell) == 0 {
		t.Fatalf("KnownSpells after reading spell %d = %#x", spell, got)
	}

	nextParty := mapload.CarryParty(ms10.Party, mw.world, ms10.Start.IDs)
	mission20 := releaseMissionMap(t, f, 20)
	addr20, _ := MissionMap(20)
	ms20, err := StartMissionFrom(mission20, addr20, 20, f.Table, mapload.DifficultyNormal, nextParty)
	if err != nil {
		t.Fatalf("StartMissionFrom(20): %v", err)
	}
	if got := releaseEntity(t, &mapWorld{world: ms20.World}, ms20.Start.IDs[0]).KnownSpells; got&(uint32(1)<<spell) == 0 {
		t.Fatalf("mission 20 KnownSpells = %#x, want carried spell %d", got, spell)
	}
}
