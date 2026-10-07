package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type staffSaleLotObservation struct {
	Index        int
	CityID       sim.SavedObjectID
	CityOrigins  []string
	Owners       []shopOwnerQuantity
	Count        int32
	Instance     sim.ItemInstance
	CastSpell    uint16
	CastPower    int32
	HasCastSpell bool
	BookSpell    uint16
	HasBookSpell bool
	Topology     *cityItemTopology
	OwnedSpell   *sim.SavedSpellObject
}

type staffSaleStateObservation struct {
	Gold        int
	Chosen      int
	ChosenShelf string
	ShelfOffset int
	Table       []ShopPlace
	Shelves     [numShopShelves][]staffSaleLotObservation
	ShelfNames  [numShopShelves]string
	Carried     []sim.ItemStack
}

func staffSaleObserveLot(item ShopItem, index int, graph *cityObjectTopology) staffSaleLotObservation {
	v := staffSaleLotObservation{Index: index, CityID: item.cityID,
		CityOrigins: append([]string(nil), item.cityOrigins...), Owners: append([]shopOwnerQuantity(nil), item.owners...),
		Count: item.Count, Instance: item.Instance().Clone()}
	v.CastSpell, v.CastPower, v.HasCastSpell = v.Instance.CastSpell()
	v.BookSpell, v.HasBookSpell = v.Instance.BookSpell()
	id := item.cityID
	if id == 0 {
		id = item.ObjectID
	}
	if graph != nil {
		for _, node := range graph.Items {
			if id != 0 && node.ID == id {
				node.Effects = append([]sim.SavedObjectID(nil), node.Effects...)
				v.Topology = &node
				if spell, ok := graph.SpellRecords[node.Spell]; node.Spell != 0 && ok {
					v.OwnedSpell = &spell
				}
				break
			}
		}
	}
	return v
}

func staffSaleObserve(s *townScreen) staffSaleStateObservation {
	view := s.ShopScreen()
	v := staffSaleStateObservation{Gold: s.sess.Town.Gold(), Chosen: view.Chosen,
		ShelfOffset: view.ShelfOffset, Table: s.sess.Shop.Table(), Carried: s.shopPackStacks()}
	if view.Chosen >= 0 && view.Chosen < len(shopRoomShelves) {
		v.ChosenShelf = shopRoomShelves[view.Chosen].shelf.String()
	}
	for shelf := range v.Shelves {
		v.ShelfNames[shelf] = ShopShelf(shelf).String()
		for i, item := range s.sess.Shop.Shelf(ShopShelf(shelf)) {
			v.Shelves[shelf] = append(v.Shelves[shelf], staffSaleObserveLot(item, i, s.sess.Town.cityObjects))
		}
	}
	return v
}

func staffSaleValueEqual(a, b sim.ItemInstance) bool {
	a.ObjectID, b.ObjectID = 0, 0
	return sim.StackStateEqual(sim.StackItem(a, 1), sim.StackItem(b, 1))
}

func staffSaleCounts(v staffSaleStateObservation, sold sim.ItemInstance, exact bool) [numShopShelves]int64 {
	var counts [numShopShelves]int64
	for shelf, lots := range v.Shelves {
		for _, lot := range lots {
			if lot.Instance.Code == sold.Code && (!exact || staffSaleValueEqual(lot.Instance, sold)) {
				counts[shelf] += int64(lot.Count)
			}
		}
	}
	return counts
}

func TestReleaseShopSoldStaffPreservesInstanceAndMagicShelf(t *testing.T) {
	output, source := os.Getenv("AGAINROM_STAFF_SALE_WITNESS_DIR"), os.Getenv("AGAINROM_SAVE_666")
	if output == "" || source == "" {
		t.Skip("AGAINROM_STAFF_SALE_WITNESS_DIR and AGAINROM_SAVE_666 are required")
	}
	if !filepath.IsAbs(output) || !filepath.IsAbs(source) {
		t.Fatal("witness output and source SAV must use absolute paths")
	}
	output = staffSaleWitnessOutput(output, os.Getenv("AGAINROM_ASSETS"))
	if err := os.MkdirAll(filepath.Join(output, "profile", "saves"), 0700); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "1005-doll-and-shop.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Steps) < 31 || scenario.Steps[11].Command != "assert_shop" || scenario.Steps[30].Command != "assert_shop" {
		t.Fatal("doll-and-shop preparation boundary changed")
	}
	scenario.OriginalSaves = filepath.Dir(source)
	scenario.Saves = filepath.Join(output, "profile", "saves")
	f.SetDeterministicFrames(true)
	app := f.App("sold staff identity")
	app.SetSaveSeams(f.SaveSeams(SaveStore{Dir: scenario.Saves}, OriginalStore{Dir: scenario.OriginalSaves}, nil))
	machine, err := os.Create(filepath.Join(output, "preparation.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer machine.Close()
	first := scenario
	first.Steps = scenario.Steps[:12]
	if err := RunHeadlessScenario(f, app, first, machine, io.Discard); err != nil {
		t.Fatal(err)
	}
	s := f.townUI
	member := s.shopMemberIndex()
	if member < 0 || member >= len(f.Carried) {
		t.Fatal("selected staff owner is absent")
	}
	worn := mapload.MemberItemEquipment(f.Carried[member], f.Table)[0]
	wornItem := shopItemFromInstance(worn, 1)
	if f.Town.cityObjects != nil {
		for _, root := range f.Town.cityObjects.Roots {
			if string(root.PartyID) == f.Carried[member].ID {
				wornItem.cityID = root.Worn[0]
			}
		}
	}
	equipped := staffSaleObserveLot(wornItem, 0, f.Town.cityObjects)
	rest := scenario
	rest.Steps = scenario.Steps[12:31]
	if err := RunHeadlessScenario(f, app, rest, machine, io.Discard); err != nil {
		t.Fatal(err)
	}
	places := f.Shop.Table()
	if len(places) != 1 || !places[0].Mine || places[0].Code != 33037 || places[0].Count != 1 {
		t.Fatalf("ordinary preparation produced table %+v", places)
	}
	sold := staffSaleObserveLot(places[0].ShopItem, 0, f.Town.cityObjects)
	tipClosed := false
	if tip := s.ShopScreen().TipPanel; tip.Showing() {
		r := ui.TipPanelCloseRect(tip.Rect)
		p := preCreateWindowPoint(scenario.Window.W, scenario.Window.H, r.Min.Add(r.Size().Div(2)))
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
		if s.ShopScreen().TipPanel.Showing() {
			t.Fatal("painted Close left shop tip showing")
		}
		tipClosed = true
	}
	before := staffSaleObserve(s)
	point := func(kind string, index int, edges ...string) {
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
	point("button", 2, "press", "release")
	afterRelease := staffSaleObserve(s)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	afterTick := staffSaleObserve(s)
	point("shelf_pick", 1, "press", "release")
	afterWeaponsPointer := staffSaleObserve(s)
	beforeCodes, afterCodes := staffSaleCounts(before, sold.Instance, false), staffSaleCounts(afterTick, sold.Instance, false)
	beforeValues, afterValues := staffSaleCounts(before, sold.Instance, true), staffSaleCounts(afterTick, sold.Instance, true)
	var codeDelta, valueDelta [numShopShelves]int64
	var globalCodeDelta int64
	for shelf := range codeDelta {
		codeDelta[shelf] = afterCodes[shelf] - beforeCodes[shelf]
		valueDelta[shelf] = afterValues[shelf] - beforeValues[shelf]
		globalCodeDelta += codeDelta[shelf]
	}
	report := struct {
		AssetRoot           string
		PreparationSteps    string
		OwnerExpectation    string
		ValueComparison     string
		Equipped            staffSaleLotObservation
		Sold                staffSaleLotObservation
		Before              staffSaleStateObservation
		AfterRelease        staffSaleStateObservation
		AfterTick           staffSaleStateObservation
		AfterWeaponsPointer staffSaleStateObservation
		CodeDelta           [numShopShelves]int64
		ExactValueDelta     [numShopShelves]int64
		GlobalCodeDelta     int64
		TipClosed           bool
		AfterMagicPointer   *staffSaleStateObservation
		MerchantLot         *staffSaleLotObservation
		AfterRepurchase     *staffSaleStateObservation
		Repurchased         *staffSaleLotObservation
		RepurchaseLimit     string
		RepurchaseIdentity  string
	}{AssetRoot: os.Getenv("AGAINROM_ASSETS"),
		PreparationSteps: "First31 unchanged actions in one App; observed after12 then continued19",
		OwnerExpectation: "Sold staff returns to Magic",
		ValueComparison:  "Complete persisted value including Price/Effects/SourceEquipment, with ObjectID separately reported",
		Equipped:         equipped, Sold: sold, Before: before, AfterRelease: afterRelease,
		AfterTick: afterTick, AfterWeaponsPointer: afterWeaponsPointer,
		CodeDelta: codeDelta, ExactValueDelta: valueDelta, GlobalCodeDelta: globalCodeDelta, TipClosed: tipClosed}
	t.Cleanup(func() {
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(output, "identity.json"), append(raw, '\n'), 0600); err != nil {
			t.Error(err)
		}
	})
	t.Logf("sold code=%d kind=%d price=%d effects=%+v ownedSpell=%+v chosen=%d/%s; codeDelta=%v exactValueDelta=%v", sold.Instance.Code, sold.Instance.Kind, sold.Instance.Price, sold.Instance.Effects, sold.Instance.SourceEquipment.Spell, afterWeaponsPointer.Chosen, afterWeaponsPointer.ChosenShelf, codeDelta, valueDelta)
	if afterRelease.Gold-before.Gold != 367 || len(afterRelease.Table) != 0 || globalCodeDelta != 1 {
		t.Errorf("sale changed gold by%d/table places%d/global code count%d; want367/0/+1", afterRelease.Gold-before.Gold, len(afterRelease.Table), globalCodeDelta)
	}
	if afterWeaponsPointer.Chosen != 1 || afterWeaponsPointer.ChosenShelf != ShelfWeapons.String() {
		t.Errorf("Weapons pointer selected%d/%s", afterWeaponsPointer.Chosen, afterWeaponsPointer.ChosenShelf)
	}
	for shelf := range valueDelta {
		want := int64(0)
		if ShopShelf(shelf) == ShelfMagic {
			want = 1
		}
		if valueDelta[shelf] != want {
			t.Errorf("sold full value count on%s changed by%d, want%d; actual code deltas%v", ShopShelf(shelf), valueDelta[shelf], want, codeDelta)
		}
	}
	if valueDelta[ShelfMagic] != 1 || globalCodeDelta != 1 {
		report.RepurchaseLimit = "not attempted: sold full value did not gain1 on Magic"
		return
	}
	point("shelf_pick", 2, "press", "release")
	magicView := staffSaleObserve(s)
	report.AfterMagicPointer = &magicView
	if magicView.Chosen != 2 || magicView.ChosenShelf != ShelfMagic.String() {
		t.Fatal("ordinary Magic pointer did not select Magic", magicView.Chosen, magicView.ChosenShelf)
	}
	index := -1
	for i, item := range f.Shop.Shelf(ShelfMagic) {
		if staffSaleValueEqual(item.Instance(), sold.Instance) {
			if index >= 0 {
				t.Fatal("sold full value has more than one Magic lot")
			}
			index = i
		}
	}
	if index < 0 {
		t.Fatal("Magic holds no uniquely observed sold full value")
	}
	stock := f.Shop.Shelf(ShelfMagic)[index]
	if int64(stock.Count) != beforeValues[ShelfMagic]+1 {
		t.Fatal("unique full value lot has no sale quantity gain", stock.Count)
	}
	report.RepurchaseIdentity = "Exact value and quantity control; compatible stock can merge and split. Native instance attribution/alias equality remains Unknown."
	for n := 0; ; n++ {
		view := s.ShopScreen()
		if index >= view.ShelfOffset && index < view.ShelfOffset+len(view.Shelf) {
			break
		}
		if n >= 64 {
			t.Fatal("ordinary wheel did not reveal sold Magic lot", index, view.ShelfOffset)
		}
		x, y, err := app.HeadlessShopPoint("shelf", 0)
		if err != nil {
			t.Fatal(err)
		}
		edge := "wheel-down"
		if index < view.ShelfOffset {
			edge = "wheel-up"
		}
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
		if s.ShopScreen().ShelfOffset == view.ShelfOffset {
			t.Fatal("ordinary wheel left shelf offset unchanged")
		}
	}
	point("shelf", index-s.ShopScreen().ShelfOffset, "press", "release")
	merchant := f.Shop.Table()
	if len(merchant) != 1 || merchant[0].Mine || merchant[0].From != ShelfMagic || merchant[0].Count != 1 || !staffSaleValueEqual(merchant[0].Instance(), sold.Instance) {
		t.Fatal("ordinary shelf click did not stage the sold full value", merchant)
	}
	merchantLot := staffSaleObserveLot(merchant[0].ShopItem, 0, f.Town.cityObjects)
	report.MerchantLot = &merchantLot
	gold := f.Town.Gold()
	point("button", 1, "press", "release")
	purchased := staffSaleObserve(s)
	report.AfterRepurchase = &purchased
	if gold-purchased.Gold != int(sold.Instance.Price) || len(purchased.Table) != 0 {
		t.Fatal("ordinary buy changed payment/table", gold-purchased.Gold, len(purchased.Table))
	}
	var count uint32
	root := cityShopRoots(t, f, s.shopMemberIndex())
	stacks := s.shopPackStacks()
	if len(root.Pack) != len(stacks) {
		t.Fatal("repurchased pack has no matching current object roots")
	}
	for index, stack := range stacks {
		if staffSaleValueEqual(stack.Instance(), sold.Instance) {
			count += stack.Count
			item := shopItemFromInstance(stack.Instance(), int32(stack.Count))
			item.cityID = root.Pack[index]
			v := staffSaleObserveLot(item, index, f.Town.cityObjects)
			report.Repurchased = &v
		}
	}
	if count != 1 {
		t.Fatal("repurchase did not retain one full staff value", count)
	}
	if got := staffSaleCounts(purchased, sold.Instance, true); got != beforeValues {
		t.Error("repurchase left a sold full value on a shelf", got, beforeValues)
	}
	t.Logf("ordinary repurchase retained code%d/Kind%d/effects%+v at gold%d", sold.Instance.Code, sold.Instance.Kind, sold.Instance.Effects, purchased.Gold)
}

func staffSaleWitnessOutput(output, assets string) string {
	rootHash := sha256.Sum256([]byte(filepath.Clean(assets)))
	return filepath.Join(output, hex.EncodeToString(rootHash[:]))
}
