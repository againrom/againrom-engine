package game

import (
	"encoding/binary"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func city1102Front(t *testing.T, duplicate bool) *FrontEnd {
	t.Helper()
	f, err := sav.Open(city1099Fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	d := p.Data()
	token := make([]byte, 37)
	token[16] = 13
	binary.LittleEndian.PutUint32(token[25:], 5)
	binary.LittleEndian.PutUint32(token[29:], 0x55773322)
	item := sav.CityItemData{Token: token, Fields: []byte{13, 14, 5, 0, 3, 71, 72, 0x54, 0x73, 7, 0, 0x99}}
	d.Objects = append(d.Objects, sav.CityObjectData{Class: "Item", Item: &item})
	idx := uint16(len(d.Objects))
	for _, o := range d.Objects {
		if o.Unit != nil && o.Unit.Name == "Leader" {
			o.Unit.Container = []uint16{idx}
			o.Unit.ContainerTails = [2]uint32{10000, 35}
			if duplicate {
				other := item
				other.Token, other.Fields = append([]byte(nil), item.Token...), append([]byte(nil), item.Fields...)
				binary.LittleEndian.PutUint32(other.Token[29:], 0x55773323)
				other.Fields[11] = 0x88
				d.Objects = append(d.Objects, sav.CityObjectData{Class: "Item", Item: &other})
				o.Unit.Container = append(o.Unit.Container, uint16(len(d.Objects)))
				o.Unit.ContainerTails[1] = 70
			}
		}
	}
	p, err = sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	u := sav.CityUpdate{Money: 3000}
	for _, c := range p.Roster() {
		u.Characters = append(u.Characters, originalCityBaselineUpdate(c))
	}
	raw, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	front := city1099Front(t, raw)
	front.Shop = NewShop(1000)
	s := front.TownScreen().(*townScreen)
	s.room, s.shopMember = roomShop, 0
	return front
}

func city1102Take(t *testing.T, f *FrontEnd, n int) {
	t.Helper()
	s := f.TownScreen().(*townScreen)
	s.room, s.shopMember = roomShop, 0
	for range n {
		s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1})
	}
}

func TestCitySaleCurrentLoadRefreshAndTrainingContinueAcrossSAV(t *testing.T) {
	f := city1102Front(t, false)
	before := trainingPartyMember(t, f, "hero")
	prior := currentCitySaleHuman(t, before)
	sourceID := cityShopRoots(t, f, 0).Pack[0]
	targetID := f.Town.cityObjects.NextID
	city1102Take(t, f, 3)
	staged := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
	wantStaged := prior
	wantStaged.InventoryWeight = 14
	if staged != wantStaged || f.Town.Gold() != 3000 || len(f.Shop.table) != 1 {
		t.Fatal("staging changed current Human fields or committed payment", staged, f.Shop.table)
	}
	if place := f.Shop.table[0]; place.cityID != targetID || place.cityID == sourceID || place.ObjectID != 0 || place.Count != 3 || place.Price != 5 || !place.Mine || f.Town.cityObjects.NextID != targetID+3 {
		t.Fatal("table merge changed its destination, quantity, price or issued floor", f.Shop.table)
	}
	s := f.TownScreen().(*townScreen)
	a := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	// SHOP-SELL-010 rounds the merged place's quantity-times-price once.
	if !strings.Contains(a.Msg, "he pays 8;") || f.Town.Gold() != 3008 {
		t.Fatal(a.Msg, f.Town.Gold())
	}
	after := trainingPartyMember(t, f, "hero")
	wantHuman := prior
	wantHuman.InventoryWeight, wantHuman.Load = 14, 47 // 35-3*7; 40+14/2, same quotient0
	if got := currentCitySaleHuman(t, after); got != wantHuman {
		t.Fatal("equal-quotient sale changed more than inventory weight/load", got, wantHuman)
	}
	if stacks := cityMemberStacks(after, f.Table); len(stacks) != 1 || stacks[0].Count != 2 || stacks[0].Price != 5 || stacks[0].ObjectID != 0 || cityShopRoots(t, f, 0).Pack[0] != sourceID {
		t.Fatal("sale changed the surviving node, current price or count", stacks)
	}
	city1102NativeReload(t, f)
	city1102Take(t, f, 1)
	s = f.TownScreen().(*townScreen)
	a = s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	if !strings.Contains(a.Msg, "he pays 3;") || f.Town.Gold() != 3011 {
		t.Fatal(a.Msg, f.Town.Gold())
	}
	if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 12 for 471" {
		t.Fatal("source school after sale", msg)
	}
	city1102NativeReload(t, f)
	city1102Take(t, f, 1)
	s = f.TownScreen().(*townScreen)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 13 for 518" {
		t.Fatal(msg)
	}
	city1102NativeReload(t, f)
	h := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
	if h.InventoryWeight != 0 || h.Load != 40 || h.Base.Skill[1] != 11 || f.Town.Gold() != 2025 {
		t.Fatal(h, f.Town.Gold())
	}
}

func currentCitySaleHuman(t *testing.T, member mapload.PartyMember) data.HumanState {
	t.Helper()
	if member.Carry == nil || member.Carry.LiveLoad == nil {
		t.Fatal("sale fixture lacks current actor operands")
	}
	load := member.Carry.LiveLoad
	return mapload.SourceHumanState(load.Inventory.Source, load.Inventory.Accumulator)
}

func city1102NativeReload(t *testing.T, f *FrontEnd) {
	t.Helper()
	party, graph, gold, shop := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold(), f.cityBookCandidate().Shop
	raw := cityProjectionSave(t, f)
	if f.Town.Gold() != gold || !reflect.DeepEqual(f.Shop, shop) {
		t.Fatal("SAVE changed the live purse or shop")
	}
	fresh := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
	if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold current SAV LOAD", town, err)
	}
	if fresh.Town.Gold() != gold || fresh.Town.cityObjects.NextID != graph.NextID || !reflect.DeepEqual(fresh.Town.cityObjects.Roots, graph.Roots) || !reflect.DeepEqual(fresh.Town.cityObjects.Books, graph.Books) {
		t.Fatal("cold SAV changed the purse, explicit roots or allocation floor")
	}
	for _, before := range party {
		after := trainingPartyMember(t, fresh, before.ID)
		if after.Hero != before.Hero || after.KnownSpells != before.KnownSpells || after.Book != before.Book || !reflect.DeepEqual(after.Carry, before.Carry) ||
			!reflect.DeepEqual(mapload.MemberItemEquipment(after, fresh.Table), mapload.MemberItemEquipment(before, f.Table)) ||
			!reflect.DeepEqual(cityMemberStacks(after, fresh.Table), cityMemberStacks(before, f.Table)) {
			t.Fatalf("cold SAV changed current values for %s\nHero: %+v -> %+v\nCarry: %+v -> %+v", before.ID, before.Hero, after.Hero, before.Carry, after.Carry)
		}
	}
	if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) {
		t.Fatal("SAVE/LOAD preparation changed live party or topology")
	}
	fresh.townUI = nil
	*f = *fresh
}

func citySalePendingReload(t *testing.T, f *FrontEnd) {
	t.Helper()
	party, graph, gold, shop := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold(), cloneShopMutation(f.Shop)
	if gold != 3000 || len(shop.table) != 1 || !shop.table[0].Mine || shop.table[0].Count != 1 {
		t.Fatal("pending sale fixture has no unpaid unit", gold, shop.table)
	}
	place := shop.table[0]
	wantGraph, wantParty := graph.Clone(), mapload.CloneParty(party)
	root, err := cityMutationParty(wantGraph, "hero")
	if err != nil || len(root.Pack) != 1 || place.cityID == 0 || place.cityID == root.Pack[0] {
		t.Fatal("pending sale fixture lacks distinct returned and surviving roots", err)
	}
	root.Pack = []sim.SavedObjectID{place.cityID, root.Pack[0]}
	for i := range wantParty {
		if wantParty[i].ID != "hero" {
			continue
		}
		carry := wantParty[i].Carry
		if carry == nil || carry.LiveLoad == nil || len(carry.OrderedStacks) != 1 || carry.OrderedStacks[0].Count != 4 || carry.LiveLoad.Inventory.Accumulator != 28 {
			t.Fatal("pending sale fixture lost its four-unit survivor or staged weight")
		}
		carry.OrderedStacks = append([]sim.ItemStack{sim.StackItem(place.Instance(), 1)}, carry.OrderedStacks...)
		carry.ItemInstances = append([]sim.ItemInstance{place.Instance()}, carry.ItemInstances...)
		carry.Items = append([]uint16{uint16(place.Code)}, carry.Items...)
		carry.LiveLoad.Inventory.Accumulator = 35
	}
	raw := cityProjectionSave(t, f)
	fresh := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
	if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold pending-sale SAV LOAD", town, err)
	}
	gotGraph := fresh.Town.cityObjects
	if fresh.Town.Gold() != 3000 || len(fresh.Shop.table) != 0 || gotGraph.NextID != graph.NextID ||
		!reflect.DeepEqual(gotGraph.Roots, wantGraph.Roots) || !reflect.DeepEqual(gotGraph.Books, graph.Books) ||
		!reflect.DeepEqual(gotGraph.Items, graph.Items) || !reflect.DeepEqual(gotGraph.Effects, graph.Effects) || !reflect.DeepEqual(gotGraph.Spells, graph.Spells) {
		t.Fatal("pending SAVE changed payment, returned roots, child graph or allocation floor")
	}
	for _, want := range wantParty {
		after := trainingPartyMember(t, fresh, want.ID)
		if after.Hero != want.Hero || after.KnownSpells != want.KnownSpells || after.Book != want.Book || !reflect.DeepEqual(after.Carry, want.Carry) ||
			!reflect.DeepEqual(mapload.MemberItemEquipment(after, fresh.Table), mapload.MemberItemEquipment(want, f.Table)) ||
			!reflect.DeepEqual(cityMemberStacks(after, fresh.Table), cityMemberStacks(want, f.Table)) {
			t.Fatalf("pending SAVE changed resolved values for %s\nCarry: %+v -> %+v", want.ID, want.Carry, after.Carry)
		}
	}
	if f.Town.Gold() != gold || !reflect.DeepEqual(shop, cloneShopMutation(f.Shop)) || !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) {
		t.Fatal("pending SAVE/LOAD preparation changed live goods, graph or purse")
	}
	fresh.townUI = nil
	*f = *fresh
}

func TestCitySaleCancellationAndCurrentChangesSurviveSAV(t *testing.T) {
	for _, mode := range []string{"clear", "return", "native-edit", "different-object", "incomplete-save", "different-member"} {
		t.Run(mode, func(t *testing.T) {
			f := city1102Front(t, mode == "different-object")
			companion := mapload.CloneParty(f.Carried)[1]
			wantCount, wantAfterSale, wantGold := 4, 3, 3003
			if mode == "different-object" {
				wantCount, wantAfterSale = 9, 8
			}
			city1102Take(t, f, 1)
			s := f.TownScreen().(*townScreen)
			switch mode {
			case "clear":
				s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 0})
				city1102Take(t, f, 1)
			case "return":
				s.ShopClick(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0})
				city1102Take(t, f, 1)
			case "native-edit":
				f.Carried[0].Hero.Body++
			case "different-member":
				s.shopMember = 1
			case "incomplete-save":
				citySalePendingReload(t, f)
				wantCount, wantAfterSale, wantGold = 5, 4, 3000
				s = f.TownScreen().(*townScreen)
				s.room = roomShop
			}
			if action := s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2}); strings.Contains(action.Msg, "cannot") {
				t.Fatal(action)
			}
			if f.Town.Gold() != wantGold || len(mapload.MemberCarriedItems(trainingPartyMember(t, f, "hero"), f.Table)) != wantCount ||
				!reflect.DeepEqual(companion.Carry, trainingPartyMember(t, f, companion.ID).Carry) {
				t.Fatal("sale changed the wrong owner, quantity or purse", f.Town.Gold(), wantGold)
			}
			city1102NativeReload(t, f)
			city1102Take(t, f, 1)
			if action := f.TownScreen().(*townScreen).ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2}); !strings.Contains(action.Msg, "he pays 3;") || f.Town.Gold() != wantGold+3 {
				t.Fatal("cold next sale", action, f.Town.Gold())
			}
			if len(mapload.MemberCarriedItems(trainingPartyMember(t, f, "hero"), f.Table)) != wantAfterSale {
				t.Fatal("cold next sale changed the wrong quantity")
			}
			city1102NativeReload(t, f)
		})
	}
}

func TestCitySaleNativeHistoryRejectsForgedPositionsAndOverdraw(t *testing.T) {
	for _, mode := range []string{"position", "overdraw", "future-training", "zero", "missing-commit", "training-order", "unknown-policy"} {
		t.Run(mode, func(t *testing.T) {
			f := city1102Front(t, false)
			snapshot, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			for i := range snapshot.OriginalCity.Bindings {
				b := &snapshot.OriginalCity.Bindings[i]
				if b.PartyID != "hero" {
					continue
				}
				b.SalesVersion = 1
				b.Sales = []SnapshotCitySale{{Position: 0, Quantity: 2, Commit: true}}
				switch mode {
				case "position":
					b.Sales[0].Position = 99
				case "overdraw":
					b.Sales[0].Quantity = 6
				case "future-training":
					b.Sales[0].AfterTraining = 1
				case "zero":
					b.Sales[0].Quantity = 0
				case "missing-commit":
					b.Sales[0].Commit = false
				case "training-order":
					b.Training = []uint8{1}
					b.Sales[0].AfterTraining = 1
					b.Sales = append(b.Sales, SnapshotCitySale{Quantity: 1, Commit: true})
				case "unknown-policy":
					b.SalesVersion = 2
				}
			}
			before, graph, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold()
			if _, _, err := f.Restore(snapshot); err == nil {
				t.Fatal("malformed historical sale accepted")
			}
			if !reflect.DeepEqual(before, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || gold != f.Town.Gold() {
				t.Fatal("failed historical restore published party, graph or purse")
			}
		})
	}
}

func TestCitySaleCurrentFieldsNeedNoReplayHistory(t *testing.T) {
	for _, mode := range []string{"load", "item-price", "current-modifier", "missing-retired-basis", "changed-retired-basis"} {
		t.Run(mode, func(t *testing.T) {
			f := city1102Front(t, false)
			city1102Take(t, f, 2)
			f.TownScreen().(*townScreen).ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
			for i := range f.Carried {
				if f.Carried[i].ID != "hero" {
					continue
				}
				m := &f.Carried[i]
				switch mode {
				case "load":
					m.Carry.LiveLoad.Load++
					m.Carry.LiveLoad.Inventory.Source.Stats[6]++
					m.Carry.LiveLoad.Movement.Load++
				case "item-price":
					m.Carry.OrderedStacks[0].Price = 9
					for j := range m.Carry.ItemInstances {
						m.Carry.ItemInstances[j].Price = 9
					}
				case "current-modifier":
					m.Carry.LiveLoad.Inventory.Source.Modifier[40]++
				case "missing-retired-basis":
					m.OriginalHuman = nil
				case "changed-retired-basis":
					m.OriginalHuman.State.Modifier.Attack.Tail[0]++
					m.OriginalHuman.Retired = true
				}
			}
			if binding := f.originalCity.saleBinding("hero"); binding != nil && len(binding.sales) != 0 {
				t.Fatal("current shop synthesized replay history")
			}
			city1102NativeReload(t, f)
			gold := f.Town.Gold()
			city1102Take(t, f, 1)
			wantPaid := 3
			if mode == "item-price" {
				wantPaid = 5
			}
			if a := f.TownScreen().(*townScreen).ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2}); strings.Contains(a.Msg, "cannot") || f.Town.Gold() != gold+wantPaid {
				t.Fatal("next sale ignored current fields", a, f.Town.Gold())
			}
			if got := cityMemberStacks(trainingPartyMember(t, f, "hero"), f.Table); len(got) != 1 || got[0].Count != 2 {
				t.Fatal("next sale changed current count", got)
			}
			city1102NativeReload(t, f)
		})
	}
}

func TestCitySaleMalformedCurrentPackFailsAtomically(t *testing.T) {
	f := city1102Front(t, false)
	city1102Take(t, f, 2)
	f.TownScreen().(*townScreen).ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Party[0].Carry.ItemInstances = snapshot.Party[0].Carry.ItemInstances[1:]
	snapshot.Party[0].Carry.Items = snapshot.Party[0].Carry.Items[1:]
	party, graph, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold()
	if _, err := f.ExportCurrentSave(snapshot, "malformed current pack"); err == nil {
		t.Fatal("ordered quantity disagreed with flat units but SAVE succeeded")
	}
	if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || gold != f.Town.Gold() {
		t.Fatal("malformed SAVE changed live party, graph or purse")
	}
}

func TestCitySaleLegacyHistoryRemainsRetiredAndNativeTrainingAvailable(t *testing.T) {
	f := city1102Front(t, false)
	b := f.originalCity.saleBinding("hero")
	b.salesVersion = 0
	b.sales = []SnapshotCitySale{{Position: 0, Quantity: 2}}
	m, err := legacyCitySoldMember(*b, b.sales)
	if err != nil {
		t.Fatal(err)
	}
	f.Carried[0] = m
	reloadTrainingCity(t, f)
	if msg := train1099(t, f, "hero", 0); !strings.HasPrefix(msg, "trained ") {
		t.Fatal(msg)
	}
	reloadTrainingCity(t, f)
	// The written pack already holds the sale; nothing is replayed from the
	// legacy history after the SAV reload.
	if binding := f.originalCity.saleBinding("hero"); binding != nil && len(binding.sales) != 0 {
		t.Fatal("legacy history survived the SAV reload")
	}
}

func TestCitySaleChangedQuotientUsesOneConditionalHumanDerive(t *testing.T) {
	f := city1102Front(t, false)
	p := f.originalCity.document.(*sav.CityProvenance)
	d := p.Data()
	for _, o := range d.Objects {
		if u := o.Unit; u != nil && u.Name == "Leader" {
			binary.LittleEndian.PutUint16(u.Scalar2[14:], 10) // stale stored capacity
			binary.LittleEndian.PutUint16(u.RawD4[4:], 65506) // signed speed modifier -30
		}
	}
	p, err := sav.CityFromData(d)
	if err != nil {
		t.Fatal(err)
	}
	u := sav.CityUpdate{Money: 3000}
	for _, c := range p.Roster() {
		u.Characters = append(u.Characters, originalCityBaselineUpdate(c))
	}
	raw, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	f = city1099Front(t, raw)
	f.Shop = NewShop(1000)
	prior := trainingPartyMember(t, f, "hero").OriginalHuman.State
	city1102Take(t, f, 2)
	staged := trainingPartyMember(t, f, "hero")
	if staged.OriginalHuman.State != prior || !staged.OriginalHuman.Retired {
		t.Fatal("pack clicks derived Human")
	}
	f.TownScreen().(*townScreen).ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	h := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
	if h.InventoryWeight != 21 || h.Load != 50 || h.Capacity != 411 || h.Speed != 65525 || h.MoverSpeed != 245 || h.Modifier.Speed != 0 || h.Attack.ToHit != 48 || h.Attack.Skill[1] != 11 {
		t.Fatalf("conditional derive %+v", h)
	}
	if h.Base != prior.Base || h.SkillXP != prior.SkillXP || h.Experience != prior.Experience || h.Attack.Tail != prior.Attack.Tail {
		t.Fatal("sale changed maintained base/XP/tail")
	}
	city1102NativeReload(t, f)
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !strings.HasSuffix(name, ".sav") {
		t.Fatal(name, err)
	}
	saved, err := ReadSaveFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	fresh := city1099Front(t, saved)
	if got := currentCitySaleHuman(t, trainingPartyMember(t, fresh, "hero")); got != h {
		t.Fatal("SAV changed conditional derive", got, h)
	}
}
