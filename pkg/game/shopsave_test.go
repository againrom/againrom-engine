package game

import (
	"bytes"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func shopSaveFixture(t *testing.T, counts ...uint32) *FrontEnd {
	t.Helper()
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	shopSaveCapacity(f)
	item := cityMemberStacks(f.Carried[0], f.Table)[1].Instance()
	for i := range f.Carried {
		if i < len(counts) && counts[i] > 0 {
			cityShopSetTestPack(t, f, i, sim.StackItem(item, counts[i]))
		} else {
			cityShopSetTestPack(t, f, i)
		}
	}
	cityShopGraph(t, f)
	return f
}

func shopSaveCapacity(f *FrontEnd) {
	for i := range f.Carried {
		if carry := f.Carried[i].Carry; carry != nil && carry.LiveLoad != nil {
			carry.LiveLoad.Capacity = 201
			carry.LiveLoad.Movement.Capacity = 201
			carry.LiveLoad.Inventory.Source.Stats[7] = 201
		}
	}
}

func shopSaveStage(t *testing.T, screen *townScreen, member, pack int, whole bool) {
	t.Helper()
	screen.room, screen.shopMember = roomShop, member
	if action := screen.shopFromPack(pack, whole); strings.Contains(action.Msg, "cannot") || len(screen.sess.Shop.table) == 0 {
		t.Fatal("stage owned item", action)
	}
}

func TestShopSaveReturnsWholeGraphToOwnerAcrossColdLoad(t *testing.T) {
	for _, original := range []bool{false, true} {
		t.Run(map[bool]string{false: "current city", true: "original city"}[original], func(t *testing.T) {
			raw, newFront := cityProjectionSource(t)
			f := cityProjectionLoad(t, raw, newFront)
			shopSaveCapacity(f)
			if !original {
				f = cityProjectionLoad(t, cityProjectionSave(t, f), newFront)
				f.originalCity = nil
			}
			before := mapload.CloneParty(f.Carried)
			roots := slices.Clone(cityShopRoots(t, f, 0).Pack)
			graph := f.Town.cityObjects.Clone()
			gold := f.Town.Gold()
			screen := f.TownScreen().(*townScreen)
			shopSaveStage(t, screen, 0, 1, true)
			if len(cityShopRoots(t, f, 0).Pack) != len(roots)-1 {
				t.Fatal("staging did not remove the selected root")
			}
			screen.shopMember = 1
			stagedParty, stagedGraph, stagedShop := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), cloneShopMutation(f.Shop)
			snapshot, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal("city SAVE", err)
			}
			encoded, err := f.ExportCurrentSave(snapshot, "shop ownership")
			if err != nil {
				t.Fatal(err)
			}
			doc, actions, indices := cityProjectionWire(t, encoded)
			var actor uint16
			for _, member := range actions.Party {
				if string(member.ID) != before[0].ID {
					continue
				}
				for _, binding := range actions.Bindings {
					if !binding.Structure && binding.ID == member.Entity {
						actor = binding.Object
					}
				}
			}
			if actor == 0 {
				t.Fatal("saved owner has no ordinary actor")
			}
			var wantRefs []uint16
			for _, id := range roots {
				wantRefs = append(wantRefs, indices[id])
			}
			refs, ok := savedObjectRefs(&doc.Objects[actor-1], "Inventory")
			if !ok || !slices.Equal(refs, wantRefs) {
				t.Fatal("ordinary SAV Inventory omitted the returned current roots", refs, wantRefs)
			}
			cold := cityProjectionLoad(t, encoded, newFront)
			if !reflect.DeepEqual(stagedParty, f.Carried) || !reflect.DeepEqual(stagedGraph, f.Town.cityObjects) || !reflect.DeepEqual(stagedShop, cloneShopMutation(f.Shop)) {
				t.Fatal("read-only capture or SAVE mutated live pending goods", reflect.DeepEqual(stagedParty, f.Carried), reflect.DeepEqual(stagedGraph, f.Town.cityObjects), reflect.DeepEqual(stagedShop, cloneShopMutation(f.Shop)))
			}
			if !slices.Equal(cityShopRoots(t, cold, 0).Pack, roots) || len(cold.Shop.table) != 0 || cold.Town.Gold() != gold {
				t.Fatal("SAVE lost the staged owner's root or paid for an unsettled trade", cityShopRoots(t, cold, 0), gold, cold.Town.Gold())
			}
			if !reflect.DeepEqual(cityMemberStacks(cold.Carried[1], cold.Table), cityMemberStacks(before[1], f.Table)) {
				t.Fatal("SAVE moved staged goods to the selected other hero")
			}
			wantNode, _ := cityMutationSource(graph, roots[1], cityItemLocation{})
			gotNode, err := cityMutationSource(cold.Town.cityObjects, roots[1], cityItemLocation{})
			if err != nil || !reflect.DeepEqual(graph.Items[wantNode], cold.Town.cityObjects.Items[gotNode]) {
				t.Fatal("SAVE changed the returning item's child identities", err)
			}
			for cycle := 0; cycle < 2; cycle++ {
				if !slices.Equal(cityShopRoots(t, cold, 0).Pack, roots) || cold.Town.Gold() != gold {
					t.Fatal("cold SAVE cycle lost or duplicated the returned graph", cycle)
				}
				cold = cityProjectionLoad(t, cityProjectionSave(t, cold), newFront)
			}
			coldScreen := cold.TownScreen().(*townScreen)
			shopSaveStage(t, coldScreen, 0, 1, true)
			if cold.Shop.table[0].cityID != roots[1] {
				t.Fatal("next shop action selected another item identity")
			}
			if action := coldScreen.shopClear(); action.Msg != "the table is cleared" {
				t.Fatal("next shop action failed", action)
			}
		})
	}
}

func TestShopSavePartialStackKeepsOwnerAndUnpaidMerchantGoods(t *testing.T) {
	f := shopSaveFixture(t, 3)
	screen := f.TownScreen().(*townScreen)
	before := cityMemberStacks(f.Carried[0], f.Table)
	gold := f.Town.Gold()
	root := cityShopRoots(t, f, 0).Pack[0]
	shopSaveStage(t, screen, 0, 0, false)
	ownedID := f.Shop.table[0].cityID
	merchant := ShopItem{Code: 0xe0e, Price: 11, Count: 2, Kind: 1, WeightPresent: true}
	f.Shop.shelves[ShelfMagic] = []ShopItem{merchant}
	screen.shopShelf = ShelfMagic
	if action := screen.shopFromShelf(0, true); strings.Contains(action.Msg, "cannot") || len(f.Shop.table) != 2 {
		t.Fatal("stage merchant goods", action)
	}
	merchantID := f.Shop.table[1].cityID
	screen.shopMember = 1
	live := f
	f = cityShopColdLoad(t, f)
	var count uint32
	for _, stack := range cityMemberStacks(f.Carried[0], f.Table) {
		if stack.Code == before[0].Code {
			count += stack.Count
		}
		if stack.Code == uint16(merchant.Code) {
			t.Fatal("SAVE transferred unpaid merchant goods to player")
		}
	}
	if count != before[0].Count || f.Town.Gold() != gold || len(f.Shop.table) != 0 {
		t.Fatal("partial-stack SAVE lost quantity or changed purse", count, before[0].Count, f.Town.Gold())
	}
	if !slices.Contains(cityShopRoots(t, f, 0).Pack, root) {
		t.Fatal("SAVE discarded the surviving stack's identity")
	}
	if ownedID == 0 || merchantID == 0 || len(live.Shop.table) != 2 || len(live.Shop.shelves[ShelfMagic]) != 0 {
		t.Fatal("SAVE changed live staged merchant stock")
	}
	for i := range f.Carried {
		if slices.Contains(cityShopRoots(t, f, i).Pack, merchantID) {
			t.Fatal("merchant identity entered a party root")
		}
	}
	if _, _, err := f.Snapshot(false); err != nil || f.Town.Gold() != gold {
		t.Fatal("repeated SAVE changed settled city", err)
	}
}

func TestShopSaveFailedPreparationKeepsGoodsReachable(t *testing.T) {
	f := shopSaveFixture(t, 3)
	screen := f.TownScreen().(*townScreen)
	shopSaveStage(t, screen, 0, 0, true)
	before := sim.StackItem(f.Shop.table[0].Instance(), uint32(f.Shop.table[0].Count))
	gold := f.Town.Gold()
	store := SaveStore{Dir: t.TempDir()}
	_, err := f.SaveDialogSeams(store, OriginalStore{}).Prepare(ui.SaveRequest{Directory: store.Dir, Name: "../bad", Format: ui.SaveSAV})
	if err == nil {
		t.Fatal("invalid save name was accepted")
	}
	if len(f.Shop.table) != 1 || !sim.StackStateEqual(before, sim.StackItem(f.Shop.table[0].Instance(), uint32(f.Shop.table[0].Count))) || f.Town.Gold() != gold {
		t.Fatal("failed SAVE preparation destroyed or paid for staged goods")
	}
}

func TestShopSaveMergedOwnersKeepQuantitiesAndCurrentIdentity(t *testing.T) {
	f := shopSaveFixture(t, 2, 3)
	screen := f.TownScreen().(*townScreen)
	kept, retired := cityShopRoots(t, f, 0).Pack[0], cityShopRoots(t, f, 1).Pack[0]
	shopSaveStage(t, screen, 0, 0, true)
	shopSaveStage(t, screen, 1, 0, true)
	if len(f.Shop.table) != 1 || f.Shop.table[0].Count != 5 || f.Shop.table[0].cityID != kept {
		t.Fatal("fixture did not reach a mixed-owner merge", f.Shop.table)
	}
	if cityItemTopologyByID(f.Town.cityObjects.Items, retired) != nil {
		t.Fatal("fixture did not retire the detached merge input")
	}
	current := f.Shop.table[0].Instance()
	keptChildren := *cityItemTopologyByID(f.Town.cityObjects.Items, kept)
	cold := cityShopColdLoad(t, f)
	for i, count := range []uint32{2, 3} {
		stacks := cityMemberStacks(cold.Carried[i], cold.Table)
		want := current
		if i > 0 {
			want = sim.CloneSplitItemValue(current, mapload.SpellRules(f.Table))
		}
		if len(stacks) != 1 || stacks[0].Count != count || !sim.ItemEqual(stacks[0].Instance(), want) {
			t.Fatal("SAVE redistributed a merged quantity or current value to the wrong owner", i, stacks)
		}
	}
	if got := cityShopRoots(t, cold, 0).Pack; len(got) != 1 || got[0] != kept || !reflect.DeepEqual(*cityItemTopologyByID(cold.Town.cityObjects.Items, kept), keptChildren) {
		t.Fatal("SAVE replaced the current merged destination or its children", got)
	}
	if cityItemTopologyByID(cold.Town.cityObjects.Items, retired) != nil {
		t.Fatal("SAVE resurrected a retired pre-merge identity")
	}
	other := cityItemTopologyByID(cold.Town.cityObjects.Items, cityShopRoots(t, cold, 1).Pack[0])
	if other == nil || other.ID == kept || other.Spell == keptChildren.Spell {
		t.Fatal("owner split aliased the merged object or its owned Spell")
	}
	for _, child := range other.Effects {
		if slices.Contains(keptChildren.Effects, child) {
			t.Fatal("owner split copied an owned Effect reference")
		}
	}
}

func TestShopSaveCapturedReceiptsSurviveLiveSaleAndRepeatedExport(t *testing.T) {
	f := shopSaveFixture(t, 3)
	screen := f.TownScreen().(*townScreen)
	gold := f.Town.Gold()
	shopSaveStage(t, screen, 0, 0, true)
	want := sim.StackItem(f.Shop.table[0].Instance(), uint32(f.Shop.table[0].Count))
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	paid := int(f.Shop.SellPayout())
	if action := screen.shopSell(); len(f.Shop.table) != 0 || f.Town.Gold() != gold+paid {
		t.Fatal("live sale did not settle once", action)
	}
	var previous []byte
	for range 2 {
		raw, err := f.ExportCurrentSave(snapshot, "captured shop")
		if err != nil {
			t.Fatal(err)
		}
		if previous != nil && !bytes.Equal(previous, raw) {
			t.Fatal("repeated export changed the same captured trade")
		}
		previous = raw
		cold := &FrontEnd{InstallResources: f.InstallResources}
		if _, city, err := cold.RestoreOriginal(raw); err != nil || !city {
			t.Fatal(city, err)
		}
		stacks := cityMemberStacks(cold.Carried[0], cold.Table)
		if len(stacks) != 1 || !sim.StackStateEqual(stacks[0], want) || cold.Town.Gold() != gold {
			t.Fatal("captured SAVE read later live sale state", stacks, cold.Town.Gold())
		}
		if len(cityMemberStacks(f.Carried[0], f.Table)) != 0 || f.Town.Gold() != gold+paid {
			t.Fatal("export repeated the live sale or returned its goods")
		}
	}
}

func TestShopSaveMixedEquipmentAndPackKeepCurrentValues(t *testing.T) {
	f, screen, item := sourceCityEquipment(t)
	cityShopGraph(t, f)
	if action := screen.shopEquipFromPack(1); action.Msg != "worn" {
		t.Fatal(action)
	}
	worn := cityShopRoots(t, f, 0).Worn[6]
	gold := f.Town.Gold()
	if action := screen.shopUnequipToTable(7); action.Msg != "on the table" {
		t.Fatal(action)
	}
	shopSaveStage(t, screen, 0, 0, true)
	if len(f.Shop.table) != 1 || f.Shop.table[0].Count != 2 {
		t.Fatal("equipment and pack did not merge on table", f.Shop.table)
	}
	cold := cityShopColdLoad(t, f)
	stacks := cityMemberStacks(cold.Carried[0], cold.Table)
	if len(stacks) != 1 || stacks[0].Count != 2 || !sim.ItemEqual(stacks[0].Instance(), item) || cold.Town.Gold() != gold {
		t.Fatal("mixed equipped/pack goods were lost", stacks, cold.Town.Gold())
	}
	if cityShopRoots(t, cold, 0).Pack[0] != worn || cityShopRoots(t, cold, 0).Worn[6] != 0 {
		t.Fatal("SAVE restored old equipment instead of current merged goods")
	}
	if action := cold.TownScreen().(*townScreen).shopEquipFromPack(1); action.Msg != "worn" {
		t.Fatal("returned equipment cannot be worn", action)
	}
}

func TestShopSavePartialMergedTransferConsumesOwnerQuantities(t *testing.T) {
	f := shopSaveFixture(t, 2, 3)
	screen := f.TownScreen().(*townScreen)
	shopSaveStage(t, screen, 0, 0, true)
	shopSaveStage(t, screen, 1, 0, true)
	if action := screen.shopOffTable(0, false); action.Msg != "returned" || f.Shop.table[0].Count != 4 {
		t.Fatal("partial transfer from merged table", action)
	}
	cold := cityShopColdLoad(t, f)
	for i, want := range []uint32{1, 4} {
		var count uint32
		for _, stack := range cityMemberStacks(cold.Carried[i], cold.Table) {
			count += stack.Count
		}
		if count != want {
			t.Fatal("SAVE ignored a completed partial ownership transfer", i, count, want)
		}
	}
}

func TestShopSaveLiveBuyAndClearContinueOnce(t *testing.T) {
	for _, action := range []string{"buy", "clear"} {
		t.Run(action, func(t *testing.T) {
			f := shopSaveFixture(t, 2)
			screen := f.TownScreen().(*townScreen)
			gold := f.Town.Gold()
			shopSaveStage(t, screen, 0, 0, true)
			f.Shop.shelves[ShelfMagic] = []ShopItem{{Code: 0xe0e, Kind: 1, Price: 11, Count: 2, WeightPresent: true}}
			screen.shopShelf = ShelfMagic
			screen.shopFromShelf(0, true)
			if len(f.Shop.table) != 2 {
				t.Fatal("mixed trade not staged")
			}
			cold := cityShopColdLoad(t, f)
			if cold.Town.Gold() != gold {
				t.Fatal("SAVE paid merchant")
			}
			if action == "buy" {
				if result := screen.shopBuy(); f.Town.Gold() != gold-22 || len(f.Shop.table) != 1 {
					t.Fatal("post-SAVE live Buy did not charge once", result)
				}
				screen.shopBuy()
				if f.Town.Gold() != gold-22 {
					t.Fatal("repeated Buy charged for already purchased goods")
				}
			}
			if result := screen.shopClear(); result.Msg != "the table is cleared" || len(f.Shop.table) != 0 {
				t.Fatal("post-SAVE live Clear", result)
			}
			screen.shopClear()
			var own, bought uint32
			for _, stack := range cityMemberStacks(f.Carried[0], f.Table) {
				if stack.Code == 0xe0e {
					bought += stack.Count
				} else {
					own += stack.Count
				}
			}
			wantBought := uint32(0)
			if action == "buy" {
				wantBought = 2
			}
			if own != 2 || bought != wantBought {
				t.Fatal("SAVE duplicated or lost live goods before next action", own, bought)
			}
		})
	}
}

func TestShopSaveFreshCityWithoutTopologyRetainsOwners(t *testing.T) {
	f := shopSaveFixture(t, 2, 3)
	f.Town.cityObjects, f.originalCity = nil, nil
	screen := f.TownScreen().(*townScreen)
	shopSaveStage(t, screen, 0, 0, true)
	shopSaveStage(t, screen, 1, 0, true)
	cold := cityShopColdLoad(t, f)
	for i, want := range []uint32{2, 3} {
		var got uint32
		for _, stack := range cityMemberStacks(cold.Carried[i], cold.Table) {
			got += stack.Count
		}
		if got != want {
			t.Fatal("fresh city's unsettled goods changed owners", i, got, want)
		}
	}
}

func TestShopSaveSharedNodeAndDistinctPackOrder(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	shopSaveCapacity(f)
	screen := f.TownScreen().(*townScreen)
	unique := cityMemberStacks(f.Carried[0], f.Table)[1]
	unique.Price++
	if err := screen.cityShopSetNode(cityShopRoots(t, f, 0).Pack[1], unique.Instance(), unique.Count, false); err != nil {
		t.Fatal(err)
	}
	want := f.Town.cityObjects.Clone()
	shared := cityShopRoots(t, f, 0).Pack[0]
	shopSaveStage(t, screen, 0, 0, true)
	shopSaveStage(t, screen, 0, 0, true)
	if len(f.Shop.table) != 2 {
		t.Fatal("distinct fixture items unexpectedly merged")
	}
	screen.shopMember = 1
	cold := cityShopColdLoad(t, f)
	for i := range f.Carried {
		if !reflect.DeepEqual(want.Roots[i], cold.Town.cityObjects.Roots[i]) {
			t.Fatal("SAVE changed current alias relationships or pack order", i, want.Roots[i], cold.Town.cityObjects.Roots[i])
		}
	}
	if node := cityItemTopologyByID(cold.Town.cityObjects.Items, shared); node == nil || !reflect.DeepEqual(node, cityItemTopologyByID(want.Items, shared)) {
		t.Fatal("SAVE copied a shared node instead of returning its occurrence")
	}
}

func TestShopSavePendingGoodsDoNotIntroduceLoadRefreshRefusal(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	before := *f.Carried[0].Carry.LiveLoad
	if before.Capacity != 0 {
		t.Fatal("fixture capacity changed")
	}
	roots := slices.Clone(cityShopRoots(t, f, 0).Pack)
	shopSaveStage(t, f.TownScreen().(*townScreen), 0, 1, true)
	cold := cityShopColdLoad(t, f)
	if !slices.Equal(cityShopRoots(t, cold, 0).Pack, roots) || cold.Carried[0].Carry.LiveLoad.Capacity != before.Capacity || cold.Carried[0].Carry.LiveLoad.Load != before.Load {
		t.Fatal("SAVE invented a load refresh instead of inverse staging")
	}
}
