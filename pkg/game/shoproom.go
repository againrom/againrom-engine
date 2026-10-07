package game

import (
	"fmt"
	"reflect"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Shop controls mutate the trade model and return one UI refresh action.

// shopBuy commits the merchant's places and puts what was bought in the
// pack.
func (t *townScreen) shopBuy() ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation((*townScreen).cityShopBuy)
	}
	if len(t.sess.Shop.Table()) == 0 {
		return ui.TownAction{}
	}
	if !t.shopHasActorContainer() {
		return ui.TownAction{Msg: "no pack"}
	}
	shop := cloneShopMutation(t.sess.Shop)
	bought, spend, ok := shop.Buy(int32(t.sess.Town.Gold()))
	if !ok {
		return ui.TownAction{Msg: "he will not sell you that: check the table and your purse"}
	}
	items := t.shopPackItemInstances()
	units := 0
	for _, item := range bought {
		for n := int32(0); n < item.Count; n++ {
			items = append(items, item.Instance())
			units++
		}
	}
	if !t.setShopPackItemInstances(items) {
		return ui.TownAction{Msg: "cannot change that pack"}
	}
	*t.sess.Shop = *shop
	t.sess.Town.gold -= int(spend)
	return ui.TownInfo(fmt.Sprintf("bought %d item(s) for %d gold; you have %d left",
		units, spend, t.sess.Town.Gold()))
}

// shopSell commits the player's places. The goods left his pack when they
// went on the table, so nothing is taken from it here.
func (t *townScreen) shopSell() ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation((*townScreen).cityShopSell)
	}
	if len(t.sess.Shop.Table()) == 0 {
		return ui.TownAction{}
	}
	var candidate *originalCityTrade
	if t.sess.originalCity != nil {
		candidate, t.sess.originalCity.trade = t.sess.originalCity.trade, nil
	}
	before, gold := t.sess.Shop.Table(), t.sess.Town.Gold()
	paid, ok := t.sess.Shop.Sell()
	if !ok {
		if t.sess.originalCity != nil {
			t.sess.originalCity.trade = candidate
		}
		return ui.TownAction{Msg: "put something of yours on the table first"}
	}
	t.sess.Town.gold += int(paid)
	t.finishOriginalCitySale(candidate, before, gold, int(paid))
	msg := fmt.Sprintf("he pays %d; you have %d", paid, t.sess.Town.Gold())
	if left := len(t.sess.Shop.Table()); left > 0 {
		msg += fmt.Sprintf("   (%d place(s) still on the table)", left)
	}
	return ui.TownInfo(msg)
}

// shopClear returns player places to the selected member's pack and merchant
// places to their source shelves. The current graph path commits the whole
// transfer atomically, including any destination merges.
func (t *townScreen) shopClear() ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation((*townScreen).cityShopClear)
	}
	if !t.shopHasActorContainer() && len(t.sess.Shop.Table()) != 0 {
		return ui.TownAction{Msg: "no pack"}
	}
	shop := cloneShopMutation(t.sess.Shop)
	back := shop.ClearTable()
	if len(back) > 0 {
		items := t.shopPackItemInstances()
		for _, item := range back {
			for n := int32(0); n < item.Count; n++ {
				items = append(items, item.Instance())
			}
		}
		if !t.setShopPackItemInstancesRefresh(items, false) {
			return ui.TownAction{Msg: "cannot change that pack"}
		}
	}
	if t.sess.Shop != nil {
		*t.sess.Shop = *shop
	}
	return ui.TownInfo("the table is cleared")
}

// shopOffTable takes units back off table place i, toward whichever side put
// them there. whole is the shift+click convention: false takes one unit and
// leaves the rest of the place on the table; true takes the whole place,
// exactly as this method always took it before this hotfix (owner, DIV-046,
// DIV-047). A place already down to one unit leaves the table either way,
// because TakeOffTable's own "at or above" test takes it all.
func (t *townScreen) shopOffTable(i int, whole bool) ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopOffTable(i, whole) })
	}
	table := t.sess.Shop.Table()
	if i < 0 || i >= len(table) {
		return ui.TownAction{}
	}
	if table[i].Mine && !t.shopHasActorContainer() {
		return ui.TownAction{Msg: "no pack"}
	}
	n := int32(1)
	if whole {
		n = table[i].Count
	}
	shop := cloneShopMutation(t.sess.Shop)
	place, ok := shop.TakeOffTable(i, n)
	if !ok {
		return ui.TownAction{}
	}
	if !place.Mine {
		shop.returnToShelf(place.From, place.ShopItem)
		*t.sess.Shop = *shop
		return ui.TownInfo("back on his shelf")
	}
	items := t.shopPackItemInstances()
	for k := int32(0); k < place.Count; k++ {
		items = append(items, place.Instance())
	}
	if !t.setShopPackItemInstances(items) {
		return ui.TownAction{Msg: "cannot change that pack"}
	}
	*t.sess.Shop = *shop
	return ui.TownInfo("back in your pack")
}

// shopFindHisPlace finds a matching merchant item and price from the same
// shelf. Unlike a player place, its From value determines the return shelf.
func shopFindHisPlace(table []ShopPlace, code data.ItemCode, price int32, from ShopShelf) (int, bool) {
	return shopFindHisItemPlace(table, ShopItem{Code: code, Price: price}, from)
}

func shopFindHisItemPlace(table []ShopPlace, item ShopItem, from ShopShelf) (int, bool) {
	for i, place := range table {
		if !place.Mine && shopItemEqual(place.ShopItem, item) && place.From == from {
			return i, true
		}
	}
	return -1, false
}

// shopFromShelf moves one unit, or the whole shelf occurrence when requested
// (DIV-046, DIV-047). Compatible merchant lots from that shelf can join an
// existing place, so repeated unit takes do not exhaust table positions.
func (t *townScreen) shopFromShelf(i int, whole bool) ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopStageShelf(i, whole) })
	}
	items := t.sess.Shop.Shelf(t.shopShelf)
	if i < 0 || i >= len(items) {
		return ui.TownAction{}
	}
	st := items[i]
	t.shopRevealTradeTable()
	n := int32(1)
	if whole || st.Count <= 1 {
		n = st.Count
	}
	if idx, found := shopFindHisItemPlace(t.sess.Shop.Table(), st, t.shopShelf); found {
		if _, ok := t.sess.Shop.RemoveFromShelf(t.shopShelf, i, n); !ok {
			return ui.TownAction{}
		}
		if !t.sess.Shop.GrowTablePlace(idx, n) {
			return ui.TownAction{}
		}
		t.startTownResponse(shopItemSpeech(st))
		return ui.TownAction{}
	}
	if !t.sess.Shop.TakeFromShelf(t.shopShelf, i, n) {
		return ui.TownAction{Msg: "the table holds five and no more"}
	}
	t.startTownResponse(shopItemSpeech(st))
	return ui.TownAction{}
}

// shopFindMinePlace matches a player-owned item and unit price. From has no
// meaning on a player place; merchant lots must use shopFindHisPlace instead.
// Each resulting place has one price for its full quantity (DIV-046).
func shopFindMinePlace(table []ShopPlace, code data.ItemCode, price int32) (int, bool) {
	return shopFindMineItemPlace(table, ShopItem{Code: code, Price: price})
}

func shopFindMineItemPlace(table []ShopPlace, item ShopItem) (int, bool) {
	for i, place := range table {
		if place.Mine && shopItemEqual(place.ShopItem, item) {
			return i, true
		}
	}
	return -1, false
}

// shopFromPack moves one unit, or the whole selected stack (DIV-046, DIV-047).
// It joins a compatible player place before consuming another table position.
// The place retains a unit price; its payout uses only the quantity moved.
func (t *townScreen) shopFromPack(i int, whole bool) ui.TownAction {
	stacks := t.shopPackStacks()
	if i < 0 || i >= len(stacks) {
		return ui.TownAction{}
	}
	if t.hasCityShopTopology() {
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopStagePack(i, whole) })
	}
	st := stacks[i]
	t.shopRevealTradeTable()
	n := int32(1)
	if whole || st.Count <= 1 {
		n = int32(st.Count)
	}
	item := st.Instance()
	price := t.shopItemValue(item)
	shopItem := shopItemFromInstance(item, n)
	shopItem.Price = price
	shopItem = t.ownShopItem(shopItem, i)
	candidate := t.prepareOriginalCitySale(item, n)
	shop := cloneShopMutation(t.sess.Shop)

	if idx, found := shopFindMineItemPlace(t.sess.Shop.Table(), shopItem); found {
		if !shop.growOwnedTablePlace(idx, shopItem) {
			return ui.TownAction{}
		}
	} else if !shop.PutOnTable(shopItem) {
		return ui.TownAction{Msg: "the table holds five and no more"}
	}

	items := t.shopPackItemInstances()
	kept := make([]sim.ItemInstance, 0, len(items))
	left := n
	for _, candidate := range items {
		if left > 0 && shopItemMatchesSelection(candidate, item) {
			left--
			continue
		}
		kept = append(kept, candidate)
	}
	if left != 0 || !t.setShopPackItemInstancesRefresh(kept, false) {
		return ui.TownAction{Msg: "cannot change that pack"}
	}
	*t.sess.Shop = *shop
	t.stageOriginalCitySale(candidate)
	if price <= 0 {
		return ui.TownAction{Msg: "he puts no price on that one - it can go on the table but he will not buy it"}
	}
	return ui.TownAction{}
}

// The doll's own moves (1005 round 2: `ITEM-CMD-007`, `ITEM-EQUIP-006`,
// `DIV-087`, `DIV-089`). THE SHOP KEEPS NO sim.World, so none of these reach
// pkg/sim: the party record IS the model here, exactly as shopPackItems'
// own read of it already is, and every write below aliases that record and
// persists the same way shopPackItems' own write does.

// shopWornSlots is the shown-or-named member's own worn array, Carry's over
// a fresh member's own — shopPackItems' own Carry-vs-fresh fallback,
// restated for equipment. Its result aliases the party record.
func (t *townScreen) shopWornSlots(i int) *[sim.EquipSlots]uint16 {
	party := t.shopParty()
	if i < 0 || i >= len(party) {
		return nil
	}
	if party[i].Carry != nil {
		return &party[i].Carry.Equipped
	}
	return &party[i].Worn
}

func (t *townScreen) shopWornItemSlots(i int) *[sim.EquipSlots]sim.ItemInstance {
	party := t.shopParty()
	if i < 0 || i >= len(party) {
		return nil
	}
	m := &party[i]
	if m.Carry != nil {
		if itemEquipmentEmptyGame(m.Carry.EquippedItems) && m.Carry.Equipped != ([sim.EquipSlots]uint16{}) {
			for k, code := range m.Carry.Equipped {
				m.Carry.EquippedItems[k] = sim.PlainItem(code)
			}
		}
		return &m.Carry.EquippedItems
	}
	if itemEquipmentEmptyGame(m.WornItems) && m.Worn != ([sim.EquipSlots]uint16{}) {
		for k, code := range m.Worn {
			m.WornItems[k] = mapload.ItemInstanceFromCode(code, t.in.Table)
		}
	}
	return &m.WornItems
}

func itemEquipmentEmptyGame(items [sim.EquipSlots]sim.ItemInstance) bool {
	for _, item := range items {
		if !item.Empty() {
			return false
		}
	}
	return true
}

func syncShopWornCodes(member *mapload.PartyMember, items [sim.EquipSlots]sim.ItemInstance) {
	if member == nil {
		return
	}
	var codes [sim.EquipSlots]uint16
	for i := range items {
		codes[i] = items[i].Code
	}
	if member.Carry != nil {
		member.Carry.Equipped = codes
	} else {
		member.Worn = codes
	}
}

// shopPartyMember is the shown-or-named party record itself, at the same
// index shopWornSlots resolves into equipment. Its result aliases the same
// backing slice shopWornSlots' pointer does, so a caller that clears
// member.Weapon writes into the record the doll is drawn from.
func (t *townScreen) shopPartyMember(i int) *mapload.PartyMember {
	party := t.shopParty()
	if i < 0 || i >= len(party) {
		return nil
	}
	return &party[i]
}

// shopMemberPack is member's own container, shopPackItems' own Carry-vs-
// Carried read restated for an arbitrary member rather than the shown one.
func (t *townScreen) shopPackItemInstances() []sim.ItemInstance {
	party := t.shopParty()
	if len(party) == 0 {
		return nil
	}
	m := &party[t.shopMemberIndex()]
	var source []sim.ItemInstance
	if m.Carry != nil {
		if len(m.Carry.ItemInstances) > 0 || len(m.Carry.Items) == 0 {
			source = m.Carry.ItemInstances
		} else {
			source = make([]sim.ItemInstance, len(m.Carry.Items))
			for i, code := range m.Carry.Items {
				source[i] = sim.PlainItem(code)
			}
		}
	} else if len(m.CarriedItems) > 0 || len(m.Carried) == 0 {
		source = m.CarriedItems
	} else {
		source = make([]sim.ItemInstance, len(m.Carried))
		for i, code := range m.Carried {
			source[i] = mapload.ItemInstanceFromCode(code, t.in.Table)
		}
	}
	out := make([]sim.ItemInstance, len(source))
	for i := range source {
		out[i] = source[i].Clone()
	}
	return out
}

func (t *townScreen) setShopPackItemInstances(items []sim.ItemInstance) bool {
	return t.setShopPackItemInstancesRefresh(items, true)
}

func (t *townScreen) shopHasActorContainer() bool {
	m := t.shopPartyMember(t.shopMemberIndex())
	return m == nil || m.Carry == nil || m.Carry.LiveLoad == nil || m.Carry.LiveLoad.Inventory.ContainerPresent
}

func (t *townScreen) setShopPackItemInstancesRefresh(items []sim.ItemInstance, refresh bool) bool {
	party := t.shopParty()
	if len(party) == 0 {
		return false
	}
	i := t.shopMemberIndex()
	if reflect.DeepEqual(t.shopPackItemInstances(), items) {
		return true
	}
	before := mapload.CloneParty([]mapload.PartyMember{party[i]})[0]
	next := mapload.CloneParty([]mapload.PartyMember{before})[0]
	next.RetireOriginalHuman()
	cloned := make([]sim.ItemInstance, len(items))
	codes := make([]uint16, len(items))
	for k := range items {
		cloned[k] = items[k].Clone()
		codes[k] = items[k].Code
	}
	if next.Carry != nil {
		next.Carry.ItemInstances = cloned
		next.Carry.Items = codes
		if err := mapload.UpdatePartyLoad(before, &next, t.in.Table, refresh, false); err != nil {
			return false
		}
	} else {
		next.CarriedItems = cloned
		next.Carried = codes
	}
	party[i] = next
	// A worn layer is a unit the pack backs: a pack that no longer holds it
	// takes it off, so buying the item back does not wear it again.
	party[i].Layers = mapload.MemberLayers(party[i], t.in.Table)
	if t.sess.originalCity != nil {
		t.sess.originalCity.trade = nil
	}
	return true
}

func (t *townScreen) shopWeaponFallbackCode(i int) (code uint16, ok bool) {
	member := t.shopPartyMember(i)
	if member == nil || member.Weapon == nil || member.WeaponMaterialized {
		return 0, false
	}
	return uint16(member.Weapon.Code), true
}

// shopSlot1Code prefers real worn equipment, then the unmaterialized starting
// weapon used by the doll and drag views. viaFallback tells a consumer to
// materialize the weapon instead of clearing an already-empty array slot.
func (t *townScreen) shopSlot1Code(i int, worn *[sim.EquipSlots]uint16) (code uint16, viaFallback bool) {
	if worn[0] != 0 {
		return worn[0], false
	}
	if code, ok := t.shopWeaponFallbackCode(i); ok {
		return code, true
	}
	return 0, false
}

// shopWearInto fills the selected equipment slot and returns displaced items
// to its member's pack. Slot 1 also resolves the starting-weapon fallback.
// Every real slot-1 write sets WeaponMaterialized so that fallback cannot
// reappear after the real weapon is removed.
func (t *townScreen) shopWearInto(slot int, code uint16) {
	t.shopWearItemInto(slot, mapload.ItemInstanceFromCode(code, t.in.Table))
}

func (t *townScreen) shopWearItemInto(slot int, item sim.ItemInstance) {
	if t.hasCityShopTopology() {
		t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) {
			bound, err := n.cityShopBind(shopItemFromInstance(item, 1))
			if err != nil {
				return ui.TownAction{}, err
			}
			if mapload.HasSourceActor(n.sess.Carried[n.shopMemberIndex()]) {
				_, err = n.cityShopSourceEquipment(-1, slot, bound, false)
			} else {
				err = n.cityShopNativeWear(slot, bound)
			}
			return ui.TownAction{}, err
		})
		return
	}
	t.withCityBookMutation(func(n *townScreen) ui.TownAction {
		n.shopWearItemValues(slot, item)
		return ui.TownAction{}
	})
}

func (t *townScreen) shopWearItemValues(slot int, item sim.ItemInstance) {
	_ = t.shopWearItemValuesOrdered(slot, item, nil)
}

func (t *townScreen) shopWearItemValuesOrdered(slot int, item sim.ItemInstance, complete func(before mapload.PartyMember, old, other sim.ItemInstance, otherSlot int) error) error {
	if member := t.shopPartyMember(t.shopMemberIndex()); member != nil && mapload.HasSourceActor(*member) {
		if next, _, ok := mapload.SourceTownEquipment(*member, t.in.Table, -1, slot, item, false); ok {
			t.commitShopSourceEquipment(member, next, slot)
			return nil
		}
		return fmt.Errorf("cannot wear that")
	}
	if t.sess.originalCity != nil {
		t.sess.originalCity.trade = nil
	}
	i := t.shopMemberIndex()
	worn := t.shopWornItemSlots(i)
	if worn == nil || slot < 1 || slot > sim.EquipSlots {
		return fmt.Errorf("invalid equipment slot")
	}
	member := t.shopPartyMember(i)
	before := mapload.CloneParty([]mapload.PartyMember{*member})[0]
	member.RetireOriginalHuman()
	// other is the item the wear takes off the opposite hand, and otherSlot
	// its one-based slot: a shield leaves when the new weapon fills both hands,
	// and a weapon that fills both hands leaves when the new item is a shield.
	var other sim.ItemInstance
	otherSlot := 0
	// A shield beside the legacy drawn-weapon fallback first makes that weapon
	// real. A weapon that leaves a hand free is worn in slot 1 beside the
	// shield; any other goes to the pack, and the shield is worn alone.
	if slot == 2 && worn[0].Empty() {
		if code, ok := t.shopWeaponFallbackCode(i); ok {
			fallback := mapload.ItemInstanceFromCode(code, t.in.Table)
			materializeStartingWeapon(member)
			if mapload.WeaponAllowsShield(data.ItemCode(code), t.in.Table) {
				worn[0] = fallback
			} else {
				other, otherSlot = fallback, 1
			}
		}
	}
	old := worn[slot-1]
	switch slot {
	case 1:
		if old.Empty() {
			if code, ok := t.shopWeaponFallbackCode(i); ok {
				old = mapload.ItemInstanceFromCode(code, t.in.Table)
			}
		}
		materializeStartingWeapon(member)
		if !mapload.WeaponAllowsShield(data.ItemCode(item.Code), t.in.Table) {
			other, otherSlot = worn[1], 2
			worn[1] = sim.ItemInstance{}
		}
	case 2:
		if !worn[0].Empty() && !mapload.WeaponAllowsShield(data.ItemCode(worn[0].Code), t.in.Table) {
			other, otherSlot = worn[0], 1
			worn[0] = sim.ItemInstance{}
		}
	}
	worn[slot-1] = item.Clone()
	syncShopWornCodes(member, *worn)
	// A restored book is authoritative at the next mint. Commit teaching
	// from this NEW equip now; replaying every worn item would change the
	// saved book, while waiting for mint would lose this later acquisition.
	if member.SpellbookRestored && member.Mage {
		var added [sim.EquipSlots]sim.ItemInstance
		added[0] = item
		_, _, taught := mapload.EquippedPoolEffects(added)
		for id := uint16(1); id <= 28; id++ {
			if taught&(uint32(1)<<id) != 0 {
				learnPartySpell(member, id, t.in.Table)
			}
		}
	}
	if complete != nil {
		if err := complete(before, old, other, otherSlot); err != nil {
			return err
		}
		refreshDerivedPartyBook(member, t.in.Table)
		t.composeShopFaces()
		return nil
	}
	pack := t.shopPackItemInstances()
	changedPack := false
	if !old.Empty() {
		pack = append(pack, old)
		changedPack = true
	}
	if !other.Empty() {
		pack = append(pack, other)
		changedPack = true
	}
	if changedPack {
		t.setShopPackItemInstances(pack)
	}
	mapload.UpdatePartyLoad(before, member, t.in.Table, true, true)
	refreshDerivedPartyBook(member, t.in.Table)
	t.composeShopFaces()
	return nil
}

// shopTakeOnePackUnit removes exactly one unit of code from the shown
// member's own pack, reporting whether one was there to take — shopFromPack's
// own removal loop (its own doc), narrowed to one unit taken by code rather
// than by stack index, since a doll drop names an item and not a position.
func (t *townScreen) shopTakeOnePackUnit(code uint16) bool {
	return t.shopTakeOnePackItem(sim.PlainItem(code))
}

func shopItemMatchesSelection(candidate, selected sim.ItemInstance) bool {
	if selected.ObjectID != 0 {
		return candidate.ObjectID == selected.ObjectID
	}
	return sim.ItemEqual(candidate, selected) && candidate.Price == selected.Price
}

func (t *townScreen) shopTakeOnePackItem(item sim.ItemInstance) bool {
	items := t.shopPackItemInstances()
	for i, candidate := range items {
		if shopItemMatchesSelection(candidate, item) {
			return t.setShopPackItemInstances(append(items[:i:i], items[i+1:]...))
		}
	}
	return false
}

// shopWear is shopEquipFromPack's and shopEquipFromShelf's own shared act:
// EquipTarget's slot, shopUsable's refusal, and the swap — mapWorld's own
// enqueueEquip (world.go) restated over the party record. EITHER REFUSAL
// TOUCHES NOTHING: both are asked before shopWearInto, on equip's own
// totality.
func (t *townScreen) shopWear(code data.ItemCode) (slot int, ok bool, action ui.TownAction) {
	slot, ok = EquipTarget(code, t.in.Table)
	if !ok {
		return 0, false, ui.TownAction{Msg: "he cannot wear that"}
	}
	if !t.shopUsable(code) {
		return 0, false, ui.TownAction{Msg: "not for his class"}
	}
	if !t.shopHasActorContainer() {
		worn := t.shopWornItemSlots(t.shopMemberIndex())
		if worn != nil && (!worn[slot-1].Empty() || slot == 1 && !worn[1].Empty() || slot == 2 && t.shopShieldDisplacesWeapon()) {
			return 0, false, ui.TownAction{Msg: "no pack"}
		}
	}
	return slot, true, ui.TownAction{}
}

func (t *townScreen) shopWearInstance(item sim.ItemInstance) (int, bool, ui.TownAction) {
	if _, layer := mapload.LayerItem(t.in.Table, item.Code); layer {
		return 0, false, ui.TownAction{Msg: "worn from the pack"}
	}
	m := t.shopPartyMember(t.shopMemberIndex())
	if m == nil || !mapload.HasSourceActor(*m) {
		return t.shopWear(data.ItemCode(item.Code))
	}
	item = mapload.SourceConstructedItem(item, t.in.Table)
	// A saved concrete object's own slot is not its appearance-code class.
	slot := 0
	switch item.SourceEquipment.Class {
	case sim.SourceWeapon:
		slot = 1
	case sim.SourceShield:
		slot = 2
	case sim.SourceArmor:
		slot = int(item.SourceEquipment.OwnKind)
	}
	if slot < 1 || slot > sim.EquipSlots {
		return 0, false, ui.TownAction{Msg: "he cannot wear that"}
	}
	if !t.shopUsable(data.ItemCode(item.Code)) {
		return 0, false, ui.TownAction{Msg: "not for his class"}
	}
	return slot, true, ui.TownAction{}
}

func (t *townScreen) commitShopSourceEquipment(member *mapload.PartyMember, next mapload.PartyMember, slot int) {
	if slot == 1 {
		materializeStartingWeapon(&next)
	}
	*member = next
	if t.sess.originalCity != nil {
		t.sess.originalCity.trade = nil
	}
	t.composeShopFaces()
}

// shopShieldDisplacesWeapon reports whether wearing a shield takes the shown
// member's weapon, real or drawn as the starting fallback, off into the pack.
func (t *townScreen) shopShieldDisplacesWeapon() bool {
	i := t.shopMemberIndex()
	worn := t.shopWornItemSlots(i)
	if worn == nil {
		return false
	}
	code := worn[0].Code
	if code == 0 {
		fallback, ok := t.shopWeaponFallbackCode(i)
		if !ok {
			return false
		}
		code = fallback
	}
	return mapload.WeaponBlocksShield(data.ItemCode(code), t.in.Table)
}

// shopBookReader recognizes the shop's non-World equivalent of KindReadBook.
// It only validates; callers remove or buy exactly one source unit before
// committing the spellbook bit, so every refusal is mutation-free.
func (t *townScreen) shopBookReader(item sim.ItemInstance) (*mapload.PartyMember, uint16, ui.TownAction, bool) {
	spell, readable := item.BookSpell()
	if !readable {
		return nil, 0, ui.TownAction{}, false
	}
	member := t.shopPartyMember(t.shopMemberIndex())
	if member == nil || !member.Mage {
		return nil, 0, ui.TownAction{Msg: "only a mage can read that"}, true
	}
	return member, spell, ui.TownAction{}, true
}

func (t *townScreen) shopLearnBook(member *mapload.PartyMember, spell uint16) {
	learnPartySpell(member, spell, t.in.Table)
}

// shopEquipFromPack wears the shown member's own pack strip cell i (spec
// bullet 1: "picked up from the pack bar ... and released on the doll,
// where it is processed exactly as the screen's own equip path processes
// it, wear rule included"). i IS THE STRIP'S OWN RAW INDEX, ShopClick's own
// PackCell arm's numbering — the money cell counted at 0 — so a press on
// the money cell resolves to no stack and is refused the same way an empty
// cell already is.
func (t *townScreen) shopEquipFromPack(i int) ui.TownAction {
	if action, handled := t.shopToggleLayer(i); handled {
		return action
	}
	if t.hasCityShopTopology() {
		if k := t.packBase + i - 1; k < 0 || k >= len(t.shopPackStacks()) {
			return ui.TownAction{}
		}
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopEquipPack(n.packBase + i - 1) })
	}
	return t.withCityBookMutation(func(n *townScreen) ui.TownAction { return n.shopEquipFromPackValues(i) })
}

func (t *townScreen) shopEquipFromPackValues(i int) ui.TownAction {
	stacks := t.shopPackStacks()
	k := t.packBase + i - 1
	if k < 0 || k >= len(stacks) {
		return ui.TownAction{}
	}
	item := stacks[k].Instance()
	if user, result, refusal, potion := t.shopPotionUser(item); potion {
		if user == nil {
			return refusal
		}
		before := mapload.CloneParty([]mapload.PartyMember{*user})[0]
		var trade *originalCityTrade
		if t.sess.originalCity != nil {
			trade = t.sess.originalCity.trade
		}
		if !t.shopTakeOnePackItem(item) {
			return ui.TownAction{}
		}
		if !commitTownPotion(user, result) {
			*user = before
			if t.sess.originalCity != nil {
				t.sess.originalCity.trade = trade
			}
			return ui.TownAction{Msg: "cannot use that potion"}
		}
		return ui.TownInfo("used")
	}
	if reader, spell, refusal, readable := t.shopBookReader(item); readable {
		if reader == nil {
			return refusal
		}
		if !t.shopTakeOnePackItem(item) {
			return ui.TownAction{}
		}
		t.shopLearnBook(reader, spell)
		return ui.TownInfo("learned")
	}
	slot, ok, refusal := t.shopWearInstance(item)
	if !ok {
		return refusal
	}
	member := t.shopPartyMember(t.shopMemberIndex())
	if mapload.HasSourceActor(*member) {
		next, _, ok := mapload.SourceTownEquipment(*member, t.in.Table, k, slot, item, false)
		if !ok {
			return ui.TownAction{Msg: "cannot wear that"}
		}
		t.commitShopSourceEquipment(member, next, slot)
		return ui.TownInfo("worn")
	}
	if !t.shopTakeOnePackItem(item) {
		return ui.TownAction{}
	}
	t.shopWearItemValues(slot, item)
	return ui.TownInfo("worn")
}

// shopEquipFromShelf buys and equips the raw shelf index (DIV-087). Slot,
// class and purse checks precede removal; refusal preserves stock and gold.
// Original shelf-to-doll payment remains unestablished by that divergence.
func (t *townScreen) shopEquipFromShelf(i int) ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopEquipShelf(n.shelfBase + i) })
	}
	return t.withCityBookMutation(func(n *townScreen) ui.TownAction { return n.shopEquipFromShelfValues(i) })
}

func (t *townScreen) shopEquipFromShelfValues(i int) ui.TownAction {
	if t.shopChosen < 0 || t.shopChosen >= len(shopRoomShelves) {
		return ui.TownAction{}
	}
	room := shopRoomShelves[t.shopChosen]
	if !room.stocked {
		return ui.TownAction{}
	}
	items := t.sess.Shop.Shelf(room.shelf)
	k := t.shelfBase + i
	if k < 0 || k >= len(items) {
		return ui.TownAction{}
	}
	item := items[k]
	if user, result, refusal, potion := t.shopPotionUser(item.Instance()); potion {
		if user == nil {
			return refusal
		}
		if item.Price > 0 && int32(t.sess.Town.Gold()) < item.Price {
			return ui.TownAction{Msg: "you cannot afford that"}
		}
		next := mapload.CloneParty([]mapload.PartyMember{*user})[0]
		if !commitTownPotion(&next, result) {
			return ui.TownAction{Msg: "cannot use that potion"}
		}
		if _, ok := t.sess.Shop.RemoveFromShelf(room.shelf, k, 1); !ok {
			return ui.TownAction{}
		}
		t.sess.Town.gold -= int(item.Price)
		*user = next
		return ui.TownInfo("used")
	}
	if reader, spell, refusal, readable := t.shopBookReader(item.Instance()); readable {
		if reader == nil {
			return refusal
		}
		if item.Price > 0 && int32(t.sess.Town.Gold()) < item.Price {
			return ui.TownAction{Msg: "you cannot afford that"}
		}
		if _, ok := t.sess.Shop.RemoveFromShelf(room.shelf, k, 1); !ok {
			return ui.TownAction{}
		}
		t.sess.Town.gold -= int(item.Price)
		t.shopLearnBook(reader, spell)
		return ui.TownInfo(fmt.Sprintf("bought and learned for %d gold; you have %d left", item.Price, t.sess.Town.Gold()))
	}
	slot, ok, refusal := t.shopWearInstance(item.Instance())
	if !ok {
		return refusal
	}
	if item.Price > 0 && int32(t.sess.Town.Gold()) < item.Price {
		return ui.TownAction{Msg: "you cannot afford that"}
	}
	member := t.shopPartyMember(t.shopMemberIndex())
	if mapload.HasSourceActor(*member) {
		next, _, ok := mapload.SourceTownEquipment(*member, t.in.Table, -1, slot, item.Instance(), false)
		if !ok {
			return ui.TownAction{Msg: "cannot wear that"}
		}
		if _, ok := t.sess.Shop.RemoveFromShelf(room.shelf, k, 1); !ok {
			return ui.TownAction{}
		}
		t.sess.Town.gold -= int(item.Price)
		t.commitShopSourceEquipment(member, next, slot)
		return ui.TownInfo(fmt.Sprintf("bought and worn for %d gold; you have %d left", item.Price, t.sess.Town.Gold()))
	}
	// ONE UNIT, NOT THE ELEMENT. A shelf element can hold several units since
	// `DIV-322`, and this gesture buys and wears exactly one: the purse is
	// debited item.Price once, just below, so taking the whole stack would
	// wear one and give the rest away.
	if _, ok := t.sess.Shop.RemoveFromShelf(room.shelf, k, 1); !ok {
		return ui.TownAction{}
	}
	t.sess.Town.gold -= int(item.Price)
	t.shopWearItemValues(slot, item.Instance())
	return ui.TownInfo(fmt.Sprintf("bought and worn for %d gold; you have %d left", item.Price, t.sess.Town.Gold()))
}

func (t *townScreen) shopEquipFromTable(i int) ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopEquipTable(i) })
	}
	return t.withCityBookMutation(func(n *townScreen) ui.TownAction { return n.shopEquipFromTableValues(i) })
}

func (t *townScreen) shopEquipFromTableValues(i int) ui.TownAction {
	table := t.sess.Shop.Table()
	if i < 0 || i >= len(table) {
		return ui.TownAction{}
	}
	place := table[i]
	if user, result, refusal, potion := t.shopPotionUser(place.Instance()); potion {
		if user == nil {
			return refusal
		}
		if !place.Mine && place.Price > 0 && int32(t.sess.Town.Gold()) < place.Price {
			return ui.TownAction{Msg: "you cannot afford that"}
		}
		next := mapload.CloneParty([]mapload.PartyMember{*user})[0]
		if !commitTownPotion(&next, result) {
			return ui.TownAction{Msg: "cannot use that potion"}
		}
		taken, ok := t.sess.Shop.TakeOffTable(i, 1)
		if !ok {
			return ui.TownAction{}
		}
		if !taken.Mine {
			t.sess.Town.gold -= int(taken.Price)
		}
		*user = next
		return ui.TownInfo("used")
	}
	if reader, spell, refusal, readable := t.shopBookReader(place.Instance()); readable {
		if reader == nil {
			return refusal
		}
		if !place.Mine && place.Price > 0 && int32(t.sess.Town.Gold()) < place.Price {
			return ui.TownAction{Msg: "you cannot afford that"}
		}
		taken, ok := t.sess.Shop.TakeOffTable(i, 1)
		if !ok {
			return ui.TownAction{}
		}
		if !taken.Mine {
			t.sess.Town.gold -= int(taken.Price)
		}
		t.shopLearnBook(reader, spell)
		if taken.Mine {
			return ui.TownInfo("learned")
		}
		return ui.TownInfo(fmt.Sprintf("bought and learned for %d gold; you have %d left", taken.Price, t.sess.Town.Gold()))
	}
	slot, ok, refusal := t.shopWearInstance(place.Instance())
	if !ok {
		return refusal
	}
	if !place.Mine && place.Price > 0 && int32(t.sess.Town.Gold()) < place.Price {
		return ui.TownAction{Msg: "you cannot afford that"}
	}
	member := t.shopPartyMember(t.shopMemberIndex())
	if mapload.HasSourceActor(*member) {
		next, _, ok := mapload.SourceTownEquipment(*member, t.in.Table, -1, slot, place.Instance(), false)
		if !ok {
			return ui.TownAction{Msg: "cannot wear that"}
		}
		if _, ok := t.sess.Shop.TakeOffTable(i, 1); !ok {
			return ui.TownAction{}
		}
		if !place.Mine {
			t.sess.Town.gold -= int(place.Price)
		}
		t.commitShopSourceEquipment(member, next, slot)
		return ui.TownInfo("worn")
	}
	taken, ok := t.sess.Shop.TakeOffTable(i, 1)
	if !ok {
		return ui.TownAction{}
	}
	if taken.Mine {
		t.shopWearItemValues(slot, taken.Instance())
		return ui.TownInfo("worn")
	}
	t.sess.Town.gold -= int(taken.Price)
	t.shopWearItemValues(slot, taken.Instance())
	return ui.TownInfo(fmt.Sprintf("bought and worn for %d gold; you have %d left", taken.Price, t.sess.Town.Gold()))
}

// shopUnequipDoll moves the one-based worn slot into the pack. Slot 1 can
// resolve the drawn starting-weapon fallback and permanently materializes it,
// preventing another removal from creating a second starting weapon.
func (t *townScreen) shopUnequipDoll(slot int) ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopUnequip(slot, false) })
	}
	return t.withCityBookMutation(func(n *townScreen) ui.TownAction { return n.shopUnequipDollValues(slot) })
}

func (t *townScreen) shopUnequipDollValues(slot int) ui.TownAction {
	if member := t.shopPartyMember(t.shopMemberIndex()); member != nil && mapload.HasSourceActor(*member) {
		next, _, ok := mapload.SourceTownEquipment(*member, t.in.Table, -1, slot, sim.ItemInstance{}, true)
		if !ok {
			return ui.TownAction{}
		}
		t.commitShopSourceEquipment(member, next, slot)
		return ui.TownInfo("off, into the pack")
	}
	i := t.shopMemberIndex()
	worn := t.shopWornItemSlots(i)
	if worn == nil || slot < 1 || slot > sim.EquipSlots {
		return ui.TownAction{}
	}
	member := t.shopPartyMember(i)
	before := mapload.CloneParty([]mapload.PartyMember{*member})[0]
	item, viaFallback := worn[slot-1], false
	if slot == 1 {
		if item.Empty() {
			if code, ok := t.shopWeaponFallbackCode(i); ok {
				item = mapload.ItemInstanceFromCode(code, t.in.Table)
				viaFallback = true
			}
		}
	}
	if item.Empty() {
		return ui.TownAction{}
	}
	if !t.shopHasActorContainer() {
		return ui.TownAction{Msg: "no pack"}
	}
	// Retire the starting-weapon fallback even when the removed weapon was real;
	// otherwise the next removal could create another unit from the fallback.
	if slot == 1 {
		materializeStartingWeapon(member)
	}
	if !viaFallback {
		worn[slot-1] = sim.ItemInstance{}
	}
	syncShopWornCodes(member, *worn)
	t.setShopPackItemInstances(append(t.shopPackItemInstances(), item))
	mapload.UpdatePartyLoad(before, member, t.in.Table, true, true)
	refreshDerivedPartyBook(member, t.in.Table)
	t.composeShopFaces()
	return ui.TownInfo("off, into the pack")
}

// shopUnequipToTable moves the worn occurrence to a player table place
// (DIV-089). A full table leaves equipment unchanged. Slot 1 resolves and
// retires the starting-weapon fallback, as removal to the pack does.
func (t *townScreen) shopUnequipToTable(slot int) ui.TownAction {
	if t.hasCityShopTopology() {
		return t.withCityShopMutation(func(n *townScreen) (ui.TownAction, error) { return n.cityShopUnequip(slot, true) })
	}
	return t.withCityBookMutation(func(n *townScreen) ui.TownAction { return n.shopUnequipToTableValues(slot) })
}

func (t *townScreen) shopUnequipToTableValues(slot int) ui.TownAction {
	if member := t.shopPartyMember(t.shopMemberIndex()); member != nil && mapload.HasSourceActor(*member) {
		next, item, ok := mapload.SourceTownEquipment(*member, t.in.Table, -1, slot, sim.ItemInstance{}, false)
		if !ok {
			return ui.TownAction{}
		}
		trade := cloneShopMutation(t.sess.Shop)
		shopItem := shopItemFromInstance(item, 1)
		shopItem.Price = t.shopItemValue(item)
		shopItem = t.ownShopItem(shopItem, -1)
		if idx, found := shopFindMineItemPlace(trade.Table(), shopItem); found {
			ok = trade.growOwnedTablePlace(idx, shopItem)
		} else {
			ok = trade.PutOnTable(shopItem)
		}
		if !ok {
			return ui.TownAction{Msg: "the table holds five and no more"}
		}
		t.shopRevealTradeTable()
		*t.sess.Shop = *trade
		t.commitShopSourceEquipment(member, next, slot)
		return ui.TownInfo("on the table")
	}
	i := t.shopMemberIndex()
	worn := t.shopWornItemSlots(i)
	if worn == nil || slot < 1 || slot > sim.EquipSlots {
		return ui.TownAction{}
	}
	member := t.shopPartyMember(i)
	before := mapload.CloneParty([]mapload.PartyMember{*member})[0]
	item, viaFallback := worn[slot-1], false
	if slot == 1 {
		if item.Empty() {
			if code, ok := t.shopWeaponFallbackCode(i); ok {
				item = mapload.ItemInstanceFromCode(code, t.in.Table)
				viaFallback = true
			}
		}
	}
	if item.Empty() {
		return ui.TownAction{}
	}
	t.shopRevealTradeTable()
	price := t.shopItemValue(item)
	shopItem := shopItemFromInstance(item, 1)
	shopItem.Price = price
	shopItem = t.ownShopItem(shopItem, -1)
	if idx, found := shopFindMineItemPlace(t.sess.Shop.Table(), shopItem); found {
		if !t.sess.Shop.growOwnedTablePlace(idx, shopItem) {
			return ui.TownAction{Msg: "the table holds five and no more"}
		}
	} else if !t.sess.Shop.PutOnTable(shopItem) {
		return ui.TownAction{Msg: "the table holds five and no more"}
	}
	// shopUnequipDoll's own rule, on the same slot and for the same reason
	// (R2): a real weapon staged on the table has left the member exactly as
	// one put in his pack has.
	if slot == 1 {
		materializeStartingWeapon(member)
	}
	if !viaFallback {
		worn[slot-1] = sim.ItemInstance{}
	}
	syncShopWornCodes(member, *worn)
	member.RetireOriginalHuman()
	mapload.UpdatePartyLoad(before, member, t.in.Table, true, true)
	refreshDerivedPartyBook(member, t.in.Table)
	t.composeShopFaces()
	return ui.TownInfo("on the table")
}

// shopRevealTradeTable closes the inspection-only spellbook before an item is
// staged on the same five-place region. The book and the table deliberately
// share one rectangle; leaving the book open while these producer paths ran
// filled invisible places until a sixth item reported a seemingly impossible
// full table. Every production producer converges here: shelf, pack, and doll.
// Closing first also makes a genuine full-table refusal self-explanatory,
// because the five occupied places are visible on the frame that reports it.
func (t *townScreen) shopRevealTradeTable() {
	if t != nil {
		t.shopBook = false
	}
}

// shopParty is the party the shop trades with, materialised so a purchase has
// somewhere to land. NextParty answers a fresh default hero where nothing has
// been carried yet, and that hero is who walks out of the gates — so writing him
// down here is what makes the item he just bought his.
func (t *townScreen) shopParty() []mapload.PartyMember {
	if len(t.sess.Carried) == 0 {
		t.sess.Carried = t.nextParty()
	}
	return t.sess.Carried
}

// shopPackItems is the SHOWN party member's container (SHOP-PICKER-043: the
// grid at (0,390,480,480) is bound to member+0xdc, the shown member's own, and
// one press of the picker rebinds it).
//
// A MEMBER WITH NO Carry KEEPS HIS OWN Carried SLICE and no Carry is created for
// him. mapload.Carry is reached by pointer precisely so that a wholly zero
// Equipped array means "wearing nothing"; minting one here for a member who has
// never finished a mission would strip what he starts wearing.
func (t *townScreen) shopPackItems() []uint16 {
	party := t.shopParty()
	if len(party) == 0 {
		return nil
	}
	m := party[t.shopMemberIndex()]
	if m.Carry != nil {
		return m.Carry.Items
	}
	return m.Carried
}

// shopPackStacks is the pack folded into stacks, which is what a row and a
// table place both address.
func (t *townScreen) shopPackStacks() []sim.ItemStack {
	if t.hasCityShopTopology() {
		if member := t.shopPartyMember(t.shopMemberIndex()); member != nil {
			return cityMemberStacks(*member, t.in.Table)
		}
		return nil
	}
	if m := t.shopPartyMember(t.shopMemberIndex()); m != nil && m.Carry != nil && m.Carry.LiveLoad != nil {
		out := make([]sim.ItemStack, len(m.Carry.OrderedStacks))
		for i, s := range m.Carry.OrderedStacks {
			out[i] = sim.StackItem(s.Instance(), s.Count)
		}
		return out
	}
	var out []sim.ItemStack
	for _, item := range t.shopPackItemInstances() {
		found := -1
		for i := range out {
			if out[i].ObjectID == item.ObjectID && sim.ItemEqual(out[i].Instance(), item) && out[i].Price == item.Price {
				found = i
				break
			}
		}
		if found >= 0 {
			out[found].Count++
		} else {
			out = append(out, sim.StackItem(item, 1))
		}
	}
	return out
}

// shopItemPrice is what one unit of code is worth (SHOP-PRICE-011).
//
// It reads the collection field B names, takes the row's own price cell and
// scales it by the shape and material the code's fields C and A name — the same
// arithmetic the generator prices a drawn item with, so an item bought at N and
// then put back on the table is still worth N.
//
// A CLASS THIS TREE CANNOT PRICE ANSWERS ZERO, which SHOP-SELL-010's own
// `elem+0x1c != 0` test reads as an item the shop will not pay for. Class 14 —
// the carried class, the quest documents and everything the mission scripts hand
// out — is the one a player will meet.
func (t *townScreen) shopItemPrice(c data.ItemCode) int32 {
	return t.shopItemValue(sim.PlainItem(uint16(c)))
}

func (t *townScreen) shopItemValue(item sim.ItemInstance) int32 {
	if item.Price != 0 {
		return item.Price
	}
	c := data.ItemCode(item.Code)
	table := t.in.Table
	if table == nil {
		return 0
	}
	var coll data.Collection
	switch b := c.B(); {
	case b == shopWeaponClass:
		coll = table.Weapons
	case b == shopShieldClass:
		coll = table.Shields
	case b >= 3 && b <= shopArmourMaxSlot:
		coll = table.Armors
	default:
		return mapload.ItemInstanceFromCode(item.Code, table).Price
	}
	if coll == nil || c.D() <= 0 || c.D() >= coll.Len() || coll.EntryName(c.D()) == "" {
		return 0
	}
	p := coll.EntryParams(c.D())
	if len(p) <= 2 || p[2] < 0 {
		return 0
	}
	return data.ItemPrice(p[2], table.Shapes, table.Materials, c.C(), c.A())
}
