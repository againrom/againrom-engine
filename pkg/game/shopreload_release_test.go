package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/random"
	"againrom/pkg/ui"
)

// shopReloadProof is what the continuous run hands the cold process: the
// gold and the hero's items the SAV holds, and the shelves the live shop
// held at SAVE, which a LOAD must not bring back.
type shopReloadProof struct {
	Gold       int
	Worn, Pack []uint16
	Old        [][]string
}

// shopShelvesSummary lists each shelf's elements as code, count and price.
func shopShelvesSummary(s *Shop) [][]string {
	out := make([][]string, numShopShelves)
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		for _, it := range s.Shelf(shelf) {
			out[shelf] = append(out[shelf], fmt.Sprintf("%#x/%d/%d", uint16(it.Code), it.Count, it.Price))
		}
	}
	return out
}

// TestReleaseTownShopReloadRestocks: after a return the hero buys the sword
// 0x1102 and the cheapest other weapon, and sells that one back onto its
// shelf. The town SAV writes no object for any shelf element
// (SHOP-SAVE-015), and a separate process that LOADs it keeps the gold and
// the hero's items and finds the shop generated anew from the campaign's
// seed and the shop stock stream, not the shelves the SAVE saw.
func TestReleaseTownShopReloadRestocks(t *testing.T) {
	if path := os.Getenv("AGAINROM_ROM1_SHOP_INPUT"); path != "" {
		shopReloadCold(t, path)
		return
	}
	f := currentTown(t, nil, nil)
	s := currentTownShop(f, "hero")
	currentTownBuy(t, s, roomWeapons, 0x1102)
	// The second purchase is the cheapest other weapon; it is sold back.
	var soldCode uint16
	price := int32(-1)
	for _, item := range f.Shop.Shelf(ShelfWeapons) {
		if uint16(item.Code) != 0x1102 && item.Price > 0 && (price < 0 || item.Price < price) {
			soldCode, price = uint16(item.Code), item.Price
		}
	}
	if soldCode == 0 || int(price) > f.Town.Gold() {
		t.Fatalf("no affordable second weapon: %#x at %d with %d gold", soldCode, price, f.Town.Gold())
	}
	currentTownBuy(t, s, roomWeapons, soldCode)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: currentTownPackCell(t, s, soldCode)})
	if action := s.shopSell(); !strings.HasPrefix(action.Msg, "he pays") {
		t.Fatalf("sell %#x: %q", soldCode, action.Msg)
	}
	var shelved []uint64
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		for _, it := range f.Shop.Shelf(shelf) {
			if it.CityID != 0 {
				shelved = append(shelved, uint64(it.CityID))
			}
		}
	}
	if len(shelved) == 0 {
		t.Fatal("the sold item holds no object identity on its shelf: the SAV guard below would test nothing")
	}
	raw := currentTownSave(t, f)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range actions.Objects {
		if slices.Contains(shelved, uint64(o.ID)) {
			t.Fatalf("the town SAV writes shelf element %d as object %d: a SAV carries no stock", o.ID, o.Object)
		}
	}
	worn, pack := currentTownMember(t, f, "hero")
	proof := shopReloadProof{Gold: f.Town.Gold(), Worn: worn[:], Pack: pack, Old: shopShelvesSummary(f.Shop)}
	b, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "town.sav")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".json", b, 0o644); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_ROM1_SHOP_INPUT")
	t.Logf("sold %#x onto a shelf; %d shelf objects absent from the %d-byte SAV", soldCode, len(shelved), len(raw))
}

// shopReloadCold is the separate process of TestReleaseTownShopReloadRestocks.
func shopReloadCold(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var proof shopReloadProof
	if err := json.Unmarshal(b, &proof); err != nil {
		t.Fatal(err)
	}
	g := currentTownReload(t, raw)
	worn, pack := currentTownMember(t, g, "hero")
	if g.Town.Gold() != proof.Gold || !slices.Equal(worn[:], proof.Worn) || !slices.Equal(pack, proof.Pack) {
		t.Fatalf("cold LOAD: gold %d/%d, worn %#x/%#x, pack %#x/%#x", g.Town.Gold(), proof.Gold, worn, proof.Worn, pack, proof.Pack)
	}
	ceiling := int32(g.Town.ChapterData().ShopMax)
	fresh := NewShop(ceiling)
	fresh.GenerateWith(g.Table, shopSeed(g.Town.Chapter(), g.Town.finishedCount(), ceiling),
		random.NewService(g.randomService().Session()).Stream(random.ShopStock))
	got := shopShelvesSummary(g.Shop)
	if g.Shop.Ceiling() != ceiling || !reflect.DeepEqual(got, shopShelvesSummary(fresh)) {
		t.Fatalf("cold LOAD: the shop is not the fresh draw at ceiling %d", ceiling)
	}
	// Loss control: the shelves the SAVE saw are not the fresh draw, so a
	// restored stock would fail the check above.
	if reflect.DeepEqual(proof.Old, got) {
		t.Fatal("loss control: the shelves at SAVE equal the fresh draw")
	}
	t.Logf("fresh-process LOAD: gold and items kept; shelves drawn anew, %d/%d/%d/%d elements",
		len(got[0]), len(got[1]), len(got[2]), len(got[3]))
}
