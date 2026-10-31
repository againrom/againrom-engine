package game

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The town services along the campaign chain: from a cold LOAD of each town's
// SAV the tavern hires, the shop buys and sells and the school trains through
// the App and the town screen's rooms. Expected rosters, prices and ceilings
// are read through installedCampaign and installedNPCDefs.

// serviceUnitPrice is MERC-PRICE-004's per-chapter factor: a jump table on
// mission/10 - 3, every other mission 0.
var serviceUnitPrice = [...]int{10, 15, 20, 40, 60, 80, 100, 600, 800, 1000, 6000, 8000, 10000}

// serviceMerc is one tavern squad: its type, head count and whole-squad price.
type serviceMerc struct{ typ, count, price int }

// serviceRoster is MERC-SHELF-002's tavern for a chapter reached by the Main
// line: the chapter's Mercenaries, unlocked by an EnableMercenary of a won main
// mission, with a non-empty pool. No squad is hired along the chain, so each
// pool is [General] MercenaryCount (MERC-DEATH-006). Prices follow
// MERC-PRICE-004 with PriceA and PriceB. Sorted by type.
func serviceRoster(t *testing.T, c Campaign, npc *data.NPCDefs, chapter int) []serviceMerc {
	t.Helper()
	unlocked := map[int]bool{}
	for m := 10; m < chapter; m += 10 {
		for _, typ := range c.Chapters[m].EnableMercenary {
			unlocked[typ] = true
		}
	}
	pools := c.MercenaryCount
	factor := serviceUnitPrice[chapter/10-3]
	var out []serviceMerc
	for _, typ := range c.Chapters[chapter].Mercenaries {
		if !unlocked[typ] || typ < 1 || typ > len(pools) || pools[typ-1] <= 0 {
			continue
		}
		p, ok := npc.Mercenary(int32(typ))
		if !ok {
			t.Fatalf("npc.reg holds no PriceA and PriceB for tavern type %d", typ)
		}
		count := pools[typ-1]
		out = append(out, serviceMerc{typ, count, (int(p.PriceA) + count*int(p.PriceB)) * factor})
	}
	slices.SortFunc(out, func(a, b serviceMerc) int { return a.typ - b.typ })
	return out
}

// serviceRosterState names the first difference between the tavern's roster
// and the expected one.
func serviceRosterState(f *FrontEnd, want []serviceMerc) error {
	var got []serviceMerc
	for _, o := range f.TownScreen().(*townScreen).tavernMercenaries() {
		got = append(got, serviceMerc{o.Type, o.Count, o.Price})
	}
	slices.SortFunc(got, func(a, b serviceMerc) int { return a.typ - b.typ })
	if !slices.Equal(got, want) {
		return fmt.Errorf("tavern roster %v, installed %v", got, want)
	}
	return nil
}

// serviceTypeCounts is how many members of each mercenary type a party holds.
func serviceTypeCounts(party []mapload.PartyMember) map[int]int {
	out := map[int]int{}
	for _, p := range party {
		if p.MercenaryType != 0 {
			out[int(p.MercenaryType)]++
		}
	}
	return out
}

func serviceHiredState(party []mapload.PartyMember, hired []serviceMerc) error {
	want := map[int]int{}
	for _, h := range hired {
		want[h.typ] = h.count
	}
	if got := serviceTypeCounts(party); !mapsEqualInt(got, want) {
		return fmt.Errorf("party mercenaries by type %v, hired %v", got, want)
	}
	return nil
}

func mapsEqualInt(a, b map[int]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// serviceOpen enters a building from the square through the App and answers
// any offer conversation that opening starts.
func serviceOpen(t *testing.T, f *FrontEnd, app *ui.App, door string, room townRoom) *townScreen {
	t.Helper()
	if err := app.HeadlessActivate(door); err != nil {
		t.Fatal(door, err)
	}
	s := f.TownScreen().(*townScreen)
	for n := 0; s.room == roomTalk && n < 64; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if s.room != room {
		t.Fatalf("%s opened room %v", door, s.room)
	}
	return s
}

// serviceTavern hires every roster squad the purse allows, in cell order,
// saves the town, loads it cold and enters the chapter's mission by the world
// map; the hired squads are carried, kept and in the mission party.
func serviceTavern(t *testing.T, raw []byte, town chainTown, controls bool) {
	f, app := campaignCold(t, serviceStore(t, raw))
	r, npc := installedCampaign(t, f), installedNPCDefs(t, f)
	want := serviceRoster(t, r, npc, town.chapter)
	s := serviceOpen(t, f, app, "TAVERN", roomTavern)
	chainRequire(t, serviceRosterState(f, want))
	purse := town.gold
	var hired []serviceMerc
	for _, o := range s.tavernMercenaries() {
		var m serviceMerc
		for _, w := range want {
			if w.typ == o.Type {
				m = w
			}
		}
		if m.price > purse {
			continue
		}
		for _, target := range []string{fmt.Sprintf("Mercenary %d", m.typ), f.Words.TavernHire} {
			if err := app.HeadlessActivate(target); err != nil {
				t.Fatal(target, err)
			}
		}
		purse -= m.price
		hired = append(hired, m)
		if f.Town.Gold() != purse || !f.Town.MercenaryHired(m.typ) {
			t.Fatalf("hire of type %d: purse %d hired %t, want %d after price %d", m.typ, f.Town.Gold(), f.Town.MercenaryHired(m.typ), purse, m.price)
		}
	}
	chainRequire(t, serviceHiredState(f.Carried, hired))
	if err := app.HeadlessActivate(f.Words.TavernExit); err != nil {
		t.Fatal(err)
	}
	coldStore, _, _ := campaignSave(t, f, false)
	g, ga := campaignCold(t, coldStore)
	check := func(g *FrontEnd) error {
		if g.Town.Gold() != purse {
			return fmt.Errorf("purse %d after hires, want %d", g.Town.Gold(), purse)
		}
		for _, h := range hired {
			if !g.Town.MercenaryHired(h.typ) {
				return fmt.Errorf("type %d is not hired", h.typ)
			}
		}
		return serviceHiredState(g.Carried, hired)
	}
	chainRequire(t, check(g))
	chainEnterByWorldMap(t, g, ga, town.chapter)
	chainRequirePurse(t, g, purse, "purse on entering the mission after the hires")
	chainRequire(t, serviceHiredState(g.live.mission.party, hired))
	if len(g.live.mission.ids) != len(g.live.mission.party) {
		t.Fatalf("mission party %d members, %d entities", len(g.live.mission.party), len(g.live.mission.ids))
	}
	t.Logf("tavern %d: roster %v, hired %v, purse %d -> %d", town.chapter, want, hired, town.gold, purse)
	if !controls {
		return
	}
	t.Run("loss_controls", func(t *testing.T) {
		if len(want) == 0 || len(hired) == 0 {
			t.Fatal("the control town offers or hires no squad")
		}
		t.Run("roster_unlock_removed", func(t *testing.T) {
			h, _ := campaignCold(t, serviceStore(t, raw))
			h.Town.mercEnabled[want[0].typ] = false
			if serviceRosterState(h, want) == nil {
				t.Fatal("a tavern without an unlocked type passed the roster check")
			}
		})
		t.Run("hire_purse_changed", func(t *testing.T) {
			for _, delta := range []int{-1, 1} {
				g.Town.gold += delta
				failed := check(g) != nil
				g.Town.gold -= delta
				if !failed {
					t.Fatalf("a purse %+d from the hire ledger passed the hire check", delta)
				}
			}
		})
		t.Run("hired_unit_dropped_from_mission", func(t *testing.T) {
			party := slices.DeleteFunc(slices.Clone(g.live.mission.party), func(p mapload.PartyMember) bool {
				return int(p.MercenaryType) == hired[0].typ
			})
			if serviceHiredState(party, hired) == nil {
				t.Fatal("a mission party without the hired squad passed the hire check")
			}
		})
	})
}

// serviceShelfState names the first shelf element the installed data does not
// give: the ceiling is the chapter's ShopMaxPrice (SHOP-MISSION-018), the floor
// 0 (SHOP-MISSION-019); the armour and weapons shelves hold 100 draws each of
// their candidate pools at the candidate's price, the magic shelf at most 20
// draws of the unfiltered pool (SHOP-GEN-005, SHOP-POOL-006).
func serviceShelfState(f *FrontEnd, ceiling int32) error {
	if f.Shop == nil || f.Shop.Ceiling() != ceiling {
		return fmt.Errorf("shop ceiling %d, installed ShopMaxPrice %d", f.Shop.Ceiling(), ceiling)
	}
	armour, weapons := shopArmourPool(f.Table, ceiling), shopWeaponPool(f.Table, ceiling)
	for _, c := range []struct {
		shelf ShopShelf
		pool  []data.ShopCandidate
		units int32
		exact bool
	}{
		{ShelfArmour, shopPlainShelfPool(armour), 100, true},
		{ShelfWeapons, shopPlainShelfPool(weapons), 100, true},
		{ShelfMagic, append(slices.Clone(armour), weapons...), 20, false},
	} {
		price := map[data.ItemCode]int32{}
		for _, p := range c.pool {
			price[p.Code] = p.Price
		}
		units := int32(0)
		for _, item := range f.Shop.Shelf(c.shelf) {
			want, ok := price[item.Code]
			if !ok || item.Count < 1 || c.exact && (item.Price != want || item.Price > ceiling || item.Price < 0) {
				return fmt.Errorf("%s shelf holds %v x%d at %d; installed candidate %t at %d, ceiling %d", c.shelf, item.Code, item.Count, item.Price, ok, want, ceiling)
			}
			units += item.Count
		}
		if c.exact && units != c.units || !c.exact && (units == 0 || units > c.units) {
			return fmt.Errorf("%s shelf holds %d units, want %d", c.shelf, units, c.units)
		}
	}
	return nil
}

// servicePack is the shop member's pack count of each code.
func servicePack(f *FrontEnd, id string) map[uint16]uint32 {
	out := map[uint16]uint32{}
	for _, p := range f.Carried {
		if p.ID == id {
			for _, st := range cityMemberStacks(p, f.Table) {
				out[st.Code] += st.Count
			}
		}
	}
	return out
}

func servicePackEqual(a, b map[uint16]uint32) bool {
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	for k, v := range b {
		if a[k] != v {
			return false
		}
	}
	return true
}

// serviceShop buys the cheapest affordable item of the weapons shelf, or of
// the armour shelf when none is, and sells one unit of another pack stack;
// the purse moves by the item's price and by half the sold unit's price
// rounded up (SHOP-BUY-009, SHOP-SELL-010), and both survive a SAVE and a
// cold LOAD.
func serviceShop(t *testing.T, raw []byte, town chainTown, controls bool) {
	f, app := campaignCold(t, serviceStore(t, raw))
	ceiling := int32(installedCampaign(t, f).Chapters[town.chapter].ShopMax)
	s := serviceOpen(t, f, app, "SHOP", roomShop)
	chainRequire(t, serviceShelfState(f, ceiling))
	member := s.shopPartyMember(s.shopMemberIndex())
	if member == nil {
		t.Fatal("the shop shows no party member")
	}
	id := member.ID
	pack := servicePack(f, id)
	purse := f.Town.Gold()
	if purse != town.gold {
		t.Fatalf("shop purse %d, town %d", purse, town.gold)
	}
	rack, index := -1, -1
	for _, try := range []int{1, 0} {
		items := f.Shop.Shelf(shopRoomShelves[try].shelf)
		for i, item := range items {
			if int(item.Price) <= purse && (index < 0 || item.Price < items[index].Price) {
				rack, index = try, i
			}
		}
		if index >= 0 {
			break
		}
	}
	if index < 0 {
		t.Fatalf("no weapons or armour item is affordable from purse %d", purse)
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: rack})
	bought := f.Shop.Shelf(shopRoomShelves[rack].shelf)[index]
	s.ShopScroll(ui.ShopWheelShelf, index/2)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: index - s.shelfBase})
	if table := f.Shop.Table(); len(table) != 1 || table[0].Mine || table[0].ShopItem.Code != bought.Code || table[0].ShopItem.Count != 1 {
		t.Fatalf("shelf %d item %d did not reach the table: %+v", rack, index, table)
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 1})
	purse -= int(bought.Price)
	pack[uint16(bought.Code)]++
	if f.Town.Gold() != purse || !servicePackEqual(servicePack(f, id), pack) {
		t.Fatalf("buy %v at %d: purse %d pack %v, want %d %v", bought.Code, bought.Price, f.Town.Gold(), servicePack(f, id), purse, pack)
	}
	stacks := s.shopPackStacks()
	// A carried document has no price and the shop will not pay for it.
	sellable := func(st sim.ItemStack) bool { return s.shopItemPrice(data.ItemCode(st.Code)) > 0 }
	sell := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code != uint16(bought.Code) && sellable(st) })
	if sell < 0 {
		sell = slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == uint16(bought.Code) })
	}
	if sell < 0 {
		t.Fatal("the pack holds nothing to sell")
	}
	sold := stacks[sell]
	s.ShopScroll(ui.ShopWheelPack, sell)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(sell)})
	table := f.Shop.Table()
	if len(table) != 1 || !table[0].Mine || uint16(table[0].ShopItem.Code) != sold.Code || table[0].ShopItem.Count != 1 || table[0].ShopItem.Price <= 0 {
		t.Fatalf("pack stack %d (%#x) did not reach the table as one unit: %+v", sell, sold.Code, table)
	}
	price := int(table[0].ShopItem.Price)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	purse += (price + 1) / 2
	pack[sold.Code]--
	if pack[sold.Code] == 0 {
		delete(pack, sold.Code)
	}
	if f.Town.Gold() != purse || !servicePackEqual(servicePack(f, id), pack) || len(f.Shop.Table()) != 0 {
		t.Fatalf("sell %#x at %d: purse %d pack %v table %d, want %d %v", sold.Code, price, f.Town.Gold(), servicePack(f, id), len(f.Shop.Table()), purse, pack)
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 3})
	coldStore, _, _ := campaignSave(t, f, false)
	g, _ := campaignCold(t, coldStore)
	check := func(g *FrontEnd) error {
		if g.Town.Gold() != purse {
			return fmt.Errorf("purse %d after the trade, want %d", g.Town.Gold(), purse)
		}
		if got := servicePack(g, id); !servicePackEqual(got, pack) {
			return fmt.Errorf("member %s pack %v, want %v", id, got, pack)
		}
		return nil
	}
	chainRequire(t, check(g))
	chainRequire(t, serviceShelfState(g, ceiling))
	t.Logf("shop %d: ceiling %d, bought %#x at %d from %s, sold %#x at %d paid %d, purse %d -> %d", town.chapter, ceiling, bought.Code, bought.Price, shopRoomShelves[rack].name, sold.Code, price, (price+1)/2, town.gold, purse)
	if !controls {
		return
	}
	t.Run("loss_controls", func(t *testing.T) {
		t.Run("trade_purse_changed", func(t *testing.T) {
			for _, delta := range []int{-1, 1} {
				g.Town.gold += delta
				failed := check(g) != nil
				g.Town.gold -= delta
				if !failed {
					t.Fatalf("a purse %+d from the trade ledger passed the trade check", delta)
				}
			}
		})
		t.Run("bought_item_removed", func(t *testing.T) {
			saved := pack[uint16(bought.Code)]
			pack[uint16(bought.Code)] = saved + 1
			failed := check(g) != nil
			pack[uint16(bought.Code)] = saved
			if !failed {
				t.Fatal("a pack one unit short of the purchase passed the trade check")
			}
		})
		t.Run("shelf_ceiling_changed", func(t *testing.T) {
			g.Shop.ceiling++
			failed := serviceShelfState(g, ceiling) != nil
			g.Shop.ceiling--
			if !failed {
				t.Fatal("a shop at another ceiling passed the shelf check")
			}
		})
		t.Run("shelf_price_changed", func(t *testing.T) {
			saved := g.Shop.shelves[ShelfWeapons][0].Price
			g.Shop.shelves[ShelfWeapons][0].Price++
			failed := serviceShelfState(g, ceiling) != nil
			g.Shop.shelves[ShelfWeapons][0].Price = saved
			if !failed {
				t.Fatal("a weapons shelf item off its installed price passed the shelf check")
			}
		})
	})
}

// serviceSchoolPrice is HERO-SKILLBUY-076's price of raising a skill whose
// maintained base is base: ftol(1.1^base * 200).
func serviceSchoolPrice(base int32) int {
	return int(int64(math.Pow(1.1, float64(base)) * 200))
}

// serviceSkillBase is a member's maintained base level of a skill slot.
func serviceSkillBase(p mapload.PartyMember, slot int) int32 {
	if h, ok := p.OriginalHumanState(); ok {
		return int32(int16(h.Base.Skill[slot]))
	}
	return p.Hero.Skill[slot]
}

// serviceSchool trains, through the App, the dearest skill step the purse
// allows over the whole party; the purse falls by the price and the skill
// rises by one, both kept by a SAVE and a cold LOAD.
func serviceSchool(t *testing.T, raw []byte, town chainTown, controls bool) {
	f, app := campaignCold(t, serviceStore(t, raw))
	s := serviceOpen(t, f, app, "SCHOOL", roomSchool)
	party := s.shopParty()
	who, slot, price := -1, 0, 0
	for m := range party {
		for i := 1; i <= 5; i++ {
			if p := serviceSchoolPrice(serviceSkillBase(party[m], i)); p > 0 && p <= town.gold && p > price {
				who, slot, price = m, i, p
			}
		}
	}
	if who < 0 {
		t.Fatalf("no skill of the party is affordable from purse %d", town.gold)
	}
	// A step between a fighter and a mage turns the school's column; the
	// school takes no input until it stands. Its animation advances only on
	// App paints, timed by a clock the test moves.
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	frame := ebiten.NewImage(640, 480)
	settle := func() {
		for i := 0; i < 64 && s.schoolTrainingBusy(); i++ {
			now = now.Add(84 * time.Millisecond)
			app.Draw(frame)
		}
		if s.schoolTrainingBusy() {
			t.Fatal("the school column never came to rest")
		}
	}
	for n := 0; n < len(party) && s.shopMemberIndex() != who; n++ {
		s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
		settle()
	}
	member := s.shopPartyMember(s.shopMemberIndex())
	if s.shopMemberIndex() != who || member == nil {
		t.Fatalf("the school picker did not reach party member %d", who)
	}
	id, mage := member.ID, member.Mage
	level := member.Hero.Skill[slot]
	cell := slot - 1
	if mage {
		cell += 5
	}
	cells := s.TownSurface().Cells
	if cells[cell].Detail != fmt.Sprintf("level %d / %d", level, price) {
		t.Fatalf("school cell %d reads %q, want level %d price %d", cell, cells[cell].Detail, level, price)
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
	if err := app.HeadlessActivate(f.Words.SchoolTrain); err != nil {
		t.Fatal(err)
	}
	purse := town.gold - price
	skill := func(g *FrontEnd) int32 {
		for _, p := range g.Carried {
			if p.ID == id {
				return p.Hero.Skill[slot]
			}
		}
		return -1
	}
	check := func(g *FrontEnd) error {
		if g.Town.Gold() != purse || skill(g) != level+1 {
			return fmt.Errorf("member %s skill %d = %d, purse %d; want %d and %d", id, slot, skill(g), g.Town.Gold(), level+1, purse)
		}
		return nil
	}
	chainRequire(t, check(f))
	coldStore, _, _ := campaignSave(t, f, false)
	g, _ := campaignCold(t, coldStore)
	chainRequire(t, check(g))
	t.Logf("school %d: %s npc %d (mage %t) skill %d %d -> %d for %d, purse %d -> %d", town.chapter, id, member.CompanionNPC, mage, slot, level, level+1, price, town.gold, purse)
	if !controls {
		return
	}
	t.Run("loss_controls", func(t *testing.T) {
		t.Run("trained_skill_removed", func(t *testing.T) {
			for i := range g.Carried {
				if g.Carried[i].ID == id {
					g.Carried[i].Hero.Skill[slot]--
					defer func() { g.Carried[i].Hero.Skill[slot]++ }()
				}
			}
			if check(g) == nil {
				t.Fatal("a member without the trained step passed the training check")
			}
		})
	})
}

// chainServices runs the three services from cold LOADs of one town's SAV.
func chainServices(t *testing.T, raw []byte, town chainTown, controls bool) {
	t.Run("tavern", func(t *testing.T) { serviceTavern(t, raw, town, controls) })
	t.Run("shop", func(t *testing.T) { serviceShop(t, raw, town, controls) })
	t.Run("school", func(t *testing.T) { serviceSchool(t, raw, town, controls) })
}

// chainServiceControlTown is the town whose services also run the loss
// controls.
const chainServiceControlTown = 50

// serviceStore is a fresh save directory holding only the town SAV, so the
// autosaves other subtests write beside it never reach the cold LOAD.
func serviceStore(t *testing.T, raw []byte) SaveStore {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Town.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return SaveStore{Dir: dir}
}
