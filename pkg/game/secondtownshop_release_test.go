package game

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/random"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondShopProof is what the continuous run hands the cold process: the
// shop state the SAV holds, the gold, and the stock the live shop held at
// SAVE with its deal undone, which a LOAD must not bring back.
type secondShopProof struct {
	Shop  *currentSecondShop
	Party []currentPartyMember
	Gold  int
	Old   secondshop.Stock
}

// secondShopActivate acts as a player does: a square row by its name; on
// the shop page a press and release at the control a name stands for:
// "ITEM n" the shelf cell of the shown category's n-th entry with units,
// "MINE n" the pack cell of the hero's n-th stack, "SHELF k" the room's
// category rectangle k, and UNDO, BUY, SELL and EXIT the four buttons.
func secondShopActivate(t *testing.T, f *FrontEnd, app *ui.App, rows ...string) {
	t.Helper()
	buttons := map[string]int{"UNDO": 0, "BUY": 1, "SELL": 2, "EXIT": 3}
	for _, row := range rows {
		s, ok := f.TownScreen().(*secondCampaignScreen)
		if !ok || !s.AtTownShop() {
			if err := app.HeadlessActivate(row); err != nil {
				t.Fatalf("town row %q: %v", row, err)
			}
			continue
		}
		kind, index := "button", -1
		var n int
		switch {
		case strings.HasPrefix(row, "ITEM "):
			fmt.Sscanf(row, "ITEM %d", &n)
			if visible := f.Town.second.shop.visibleShelf(); n >= 1 && n <= len(visible) {
				kind, index = "shelf", visible[n-1]-s.shelfBase
			}
		case strings.HasPrefix(row, "MINE "):
			fmt.Sscanf(row, "MINE %d", &n)
			kind, index = "pack", s.packCell(n-1)
		case strings.HasPrefix(row, "SHELF "):
			fmt.Sscanf(row, "SHELF %d", &n)
			kind, index = "shelf_pick", n-1
		default:
			if i, ok := buttons[row]; ok {
				index = i
			}
		}
		x, y, err := app.HeadlessShopPoint(kind, index)
		if err != nil {
			t.Fatalf("shop control %q: %v", row, err)
		}
		shopTap(t, app, image.Pt(x, y), "press", "release")
	}
}

// secondPartyJSON is the party as the document writes it: a decoded member
// holds empty lists where the captured one holds none.
func secondPartyJSON(p []currentPartyMember) string {
	raw, _ := json.Marshal(p)
	return string(raw)
}

// secondShopUndone is the stock s holds once its open deal is undone,
// without changing s.
func secondShopUndone(s secondShop) secondshop.Stock {
	for c := range s.shelves {
		s.shelves[c] = append([]ShopItem(nil), s.shelves[c]...)
	}
	var bag []sim.ItemInstance
	s.undo(&bag)
	return s.stock()
}

// secondShopSAVFields answers the sorted field names of the shop the SAV's
// second campaign document holds; nil when it holds none.
func secondShopSAVFields(t *testing.T, doc *sav.DocumentData) []string {
	t.Helper()
	b, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("no current action supplement:", err)
	}
	var a struct {
		Session struct {
			Second struct{ Shop map[string]json.RawMessage }
		}
	}
	if err := json.Unmarshal(b, &a); err != nil {
		t.Fatal(err)
	}
	var out []string
	for k := range a.Session.Second.Shop {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// secondShopSaved checks the SAV holds the captured shop and no stock: its
// shop document has exactly the records, and the hero is written through
// the party member producer.
func secondShopSaved(t *testing.T, raw []byte, saved *currentSecondCampaign) *currentActionData {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if got := actions.Session.Second; !reflect.DeepEqual(got.Shop, saved.Shop) || secondPartyJSON(got.Party) != secondPartyJSON(saved.Party) || len(got.Party) != 1 {
		t.Fatal("the SAV's shop or hero differs from the captured campaign")
	}
	want := []string{"Records"}
	if saved.Shop == nil {
		// A new game's records are not written.
		want = nil
	}
	if fields := secondShopSAVFields(t, &doc); !reflect.DeepEqual(fields, want) {
		t.Fatalf("the SAV's shop holds fields %v, want only Records: a SAV carries no stock", fields)
	}
	return actions
}

// secondShopColdProof is the separate process: the LOAD restores the town's
// shop page, gold, records and the hero's items, and draws the stock anew
// from the town's record on a fresh shop stock stream; the stock the live
// shop held at SAVE does not come back. A purchase then settles at the
// fresh stock's price.
func secondShopColdProof(t *testing.T, path string, id int) {
	t.Helper()
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var proof secondShopProof
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatal(err)
	}
	f, app := secondTownCold(t, filepath.Dir(path), filepath.Base(path))
	c := f.Town.second
	got := captureSecondCampaign(c)
	equal := reflect.DeepEqual(got.Shop, proof.Shop) && secondPartyJSON(got.Party) == secondPartyJSON(proof.Party)
	if c.current != (partyPlace{2, id}) || c.room != secondTownShop || f.Town.gold != proof.Gold || !equal {
		t.Fatalf("cold LOAD: current %v room %d gold %d/%d shop and hero equal %v", c.current, c.room, f.Town.gold, proof.Gold, equal)
	}
	tables, ok := secondShopTables(f.Table)
	if !ok {
		t.Fatal("no shop tables")
	}
	fresh := tables.Fill(c.shop.records[id], random.NewService(f.randomService().Session()).Stream(random.ShopStock).Raw)
	if c.shop.filled != id || !reflect.DeepEqual(c.shop.stock(), fresh) {
		t.Fatalf("cold LOAD: filled %d, stock equals a fresh draw %v", c.shop.filled, reflect.DeepEqual(c.shop.stock(), fresh))
	}
	// Loss control: the stock the SAVE saw is not the fresh draw, so a
	// restored stock would fail the check above.
	if reflect.DeepEqual(proof.Old, fresh) {
		t.Fatal("loss control: the stock at SAVE equals the fresh draw")
	}
	first := c.shop.shelf()[c.shop.visibleShelf()[0]]
	hero := len(c.heroItems())
	secondShopActivate(t, f, app, "UNDO", "ITEM 1", "BUY")
	if f.Town.gold != proof.Gold-int(first.Price) || len(c.heroItems()) < hero || f.TownScreen().Header() != "ROM2 campaign: shop" {
		t.Fatalf("cold purchase at %d: gold %d/%d", first.Price, f.Town.gold, proof.Gold)
	}
	t.Logf("fresh-process LOAD of town %d: gold, records and items kept; stock drawn anew, %d/%d/%d/%d entries", id,
		len(fresh[0]), len(fresh[1]), len(fresh[2]), len(fresh[3]))
}

// secondShopWitness writes the page's headless frame for the owner.
func secondShopWitness(t *testing.T, app *ui.App, name string) {
	t.Helper()
	dir := os.Getenv("AGAINROM_SHOP_WITNESS")
	if dir == "" {
		return
	}
	picture, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal("shop frame", err, note)
	}
	out, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, picture); err != nil {
		t.Fatal(err)
	}
}

// secondShopNamesResolve checks that every item town 1's record admits,
// books aside, has a line in the shipped item-name table under its NameKey;
// the loss control swaps the part and class fields and must lose names.
func secondShopNamesResolve(t *testing.T, f *FrontEnd) {
	t.Helper()
	tables, ok := secondShopTables(f.Table)
	if !ok || len(f.Table.Names) == 0 {
		t.Fatal("no shop tables or item-name table")
	}
	var named, swapped, total int
	for _, c := range secondshop.NewGameRecords[1] {
		for _, it := range tables.Candidates(c, nil) {
			key, ok := tables.NameKey(it)
			if !ok {
				continue
			}
			total++
			if _, ok := f.Table.Names.NameFor(data.ItemCode(key)); ok {
				named++
			}
			if it.Table != secondshop.Magic {
				part := uint16(tables.Part(it))
				alt := key&^0x0fe0 | uint16(it.Class&0xf)<<8 | (part&0x7)<<5
				if _, ok := f.Table.Names.NameFor(data.ItemCode(alt)); ok && alt != key {
					swapped++
				}
			}
		}
	}
	if total == 0 || named != total || swapped >= total/2 {
		t.Fatalf("item names: %d of %d named, %d named with part and class swapped", named, total, swapped)
	}
	t.Logf("item names: %d of %d named; swapped keys name %d", named, total, swapped)
}

// TestReleaseSecondTownShopDealAndSave opens the town 1 shop from the
// square, buys and sells through the page's rows at the claimed prices,
// saves with a deal open, and has a separate process LOAD the SAV: the gold,
// records and items are kept and the stock is drawn anew (DIV-3045).
func TestReleaseSecondTownShopDealAndSave(t *testing.T) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_SHOP_INPUT"); path != "" {
		secondShopColdProof(t, path, 1)
		return
	}
	f, app := secondTownNew(t, false, false)
	secondShopActivate(t, f, app, "SHOP")
	c := f.Town.second
	if c.room != secondTownShop || f.TownScreen().Header() != "ROM2 campaign: shop" || len(c.shop.shelves[0]) == 0 || c.shop.filled != 1 {
		t.Fatalf("SHOP opened room %d, filled %d, %d entries", c.room, c.shop.filled, len(c.shop.shelves[0]))
	}
	secondShopArtLoaded(t, f.TownScreen().(*secondCampaignScreen))
	secondShopWitness(t, app, "town1-shop-open")
	secondShopNamesResolve(t, f)
	f.Town.gold = 5000
	first := c.shop.shelf()[c.shop.visibleShelf()[0]]
	secondShopActivate(t, f, app, "ITEM 1", "BUY")
	if f.Town.gold != 5000-int(first.Price) || len(c.heroItems()) != 1 || c.heroItems()[0].Count != 1 {
		t.Fatalf("buy of one at %d: gold %d hero %v", first.Price, f.Town.gold, c.heroItems())
	}
	secondShopWitness(t, app, "town1-shop-bought")
	gold := f.Town.gold
	secondShopActivate(t, f, app, "MINE 1", "SELL")
	if f.Town.gold != gold+int(secondshop.SaleCredit(int64(first.Price), 1)) || len(c.heroItems()) != 0 {
		t.Fatalf("sale of one at %d: gold %d -> %d", first.Price, gold, f.Town.gold)
	}
	secondShopActivate(t, f, app, "SHELF 3", "ITEM 1", "BUY", "SHELF 1", "ITEM 2")
	if len(c.heroItems()) != 1 || len(c.shop.buying()) != 1 {
		t.Fatalf("second deal: hero %v buying %v", c.heroItems(), c.shop.buying())
	}
	secondShopWitness(t, app, "town1-shop-deal-open")
	saved, savedGold, old := captureSecondCampaign(c), f.Town.gold, secondShopUndone(c.shop)
	dir := secondMissionSaveDirectory(t)
	raw := secondTownNamedSave(t, f, app, dir, "Shop")
	secondShopSaved(t, raw, saved)
	if len(c.shop.buying()) != 1 {
		t.Fatal("SAVE closed the deal")
	}
	proof, err := json.Marshal(secondShopProof{Shop: saved.Shop, Party: saved.Party, Gold: savedGold, Old: old})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, secondSaveFile(t, dir, "Shop"))
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_SECOND_SHOP_INPUT")
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare || len(c.shop.buying()) != 0 {
		t.Fatalf("escape left room %d with %d pending", c.room, len(c.shop.buying()))
	}
	t.Logf("town 1 shop: %d/%d/%d/%d entries; bought at %d, sold for %d; SAV %d bytes", len(old[0]), len(old[1]),
		len(old[2]), len(old[3]), first.Price, secondshop.SaleCredit(int64(first.Price), 1), len(raw))
}
