package game

import (
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type cityShopSelection struct {
	Location cityItemLocation
	ID       sim.SavedObjectID
	NativeID sim.SavedObjectID
	Count    uint32
}

func (t *townScreen) hasCityShopTopology() bool {
	return t.sess.Town != nil && t.sess.Town.cityObjects != nil
}

func (t *townScreen) withCityShopMutation(action func(*townScreen) (ui.TownAction, error)) ui.TownAction {
	n := *t
	n.sess = t.sess.cityBookCandidate()
	for _, member := range n.sess.Carried {
		if err := mapload.ValidatePartyLoad(member); err != nil {
			return ui.TownAction{Msg: "cannot change city items: " + err.Error()}
		}
		for _, root := range n.sess.Town.cityObjects.Roots {
			if string(root.PartyID) == member.ID && !cityPackRootsCurrent(member, root, n.in.Table) {
				return ui.TownAction{Msg: "cannot change city items: stale pack roots"}
			}
		}
	}
	projection, err := newCityObjectProjection(n.sess.Town.cityObjects, n.sess.Carried, n.in.Table)
	if err == nil {
		err = n.validateCityShopValues(projection)
	}
	if err != nil {
		return ui.TownAction{Msg: "cannot change city items: " + err.Error()}
	}
	n.sess.Town.cityObjects = projection.graph
	result, err := action(&n)
	if err == nil {
		// A worn layer is a unit the pack backs: a pack that gave it up takes
		// it off, so buying the item back does not wear it again.
		for i := range n.sess.Carried {
			n.sess.Carried[i].Layers = mapload.MemberLayers(n.sess.Carried[i], n.in.Table)
		}
		n.sess.Town.cityObjects, err = prepareCityPartyBookChanges(n.sess.Town.cityObjects, t.sess.Carried, n.sess.Carried, n.in.Table)
	}
	if err == nil {
		err = n.validateCityShopRoots()
	}
	if err != nil {
		return ui.TownAction{Msg: "cannot change city items: " + err.Error()}
	}
	t.sess.commitCityBookCandidate(n.sess)
	n.sess = t.sess
	*t = n
	return result
}

// cityPackRootsCurrent reports whether a member's pack root still names the
// member's pack. A member without ordered cells shows its units folded
// (ITEM-MERGE-129). A root holding one record per unit, as a SAV may store
// such a member, names the same units, and the projection folds it.
func cityPackRootsCurrent(member mapload.PartyMember, root cityPartyObjectRoots, table *mapload.Table) bool {
	if len(root.Pack) == len(cityMemberStacks(member, table)) {
		return true
	}
	flat := member.Carry == nil || member.Carry.OrderedStacks == nil
	return flat && len(root.Pack) == len(mapload.MemberCarriedItems(member, table))
}

// cityJoinForm is incoming written in held's stored form (sim.JoinForm). The
// definition tables build a plain equipment value; any other item is built at
// the per-unit weight 1 (ITEM-STACK-003) the city SAV writer gives it. A cell
// incoming joins keeps its held value.
func cityJoinForm(held, incoming sim.ItemInstance, table *mapload.Table) (sim.ItemInstance, bool) {
	return sim.JoinForm(held, incoming, func(v sim.ItemInstance) sim.ItemInstance {
		built := mapload.SourceConstructedItem(v, table)
		if b := data.ItemCode(v.Code).B(); !built.WeightPresent && (b < 1 || b > 12) {
			built.WeightPresent, built.Weight = true, 1
		}
		return built
	})
}

// cityPackJoins reports whether incoming joins the held pack cell.
func cityPackJoins(held, incoming sim.ItemInstance, table *mapload.Table) bool {
	v, ok := cityJoinForm(held, incoming, table)
	return ok && sim.CanMergeItemValues(held, v)
}

// cityShelfJoins reports whether an item returned to a shelf joins the shelf
// cell, which compares without price (DIV-322).
func cityShelfJoins(held, incoming sim.ItemInstance, table *mapload.Table) bool {
	v, ok := cityJoinForm(held, incoming, table)
	return ok && sim.ItemEqual(held, v)
}

func (t *townScreen) validateCityShopRoots() error {
	g := t.sess.Town.cityObjects
	for _, member := range t.sess.Carried {
		root, err := cityMutationParty(g, member.ID)
		if err != nil {
			return err
		}
		if len(root.Pack) != len(cityMemberStacks(member, t.in.Table)) {
			return fmt.Errorf("city pack root count differs from current occurrences")
		}
	}
	p, err := newCityObjectProjection(g, t.sess.Carried, t.in.Table)
	if err != nil {
		return err
	}
	// An Item the operation created must carry the identity the operation
	// minted. A child is different: a weapon's owned Spell exists only while
	// it is worn, so equipping a staff from the pack mints its Spell here.
	for _, row := range p.graph.Items {
		if row.ID >= g.NextID {
			return fmt.Errorf("city operation omitted a new object identity")
		}
	}
	if err := t.validateCityShopValues(p); err != nil {
		return err
	}
	// Only a completed transaction can retire a detached merge input. Party,
	// table and shelf aliases have all contributed their reachable children.
	p.graph.Items = slices.DeleteFunc(p.graph.Items, func(row cityItemTopology) bool {
		_, live := p.items[row.ID]
		return !live
	})
	p.graph.Effects = slices.DeleteFunc(p.graph.Effects, func(id sim.SavedObjectID) bool {
		_, live := p.effects[id]
		return !live
	})
	p.graph.Spells = slices.DeleteFunc(p.graph.Spells, func(id sim.SavedObjectID) bool {
		_, live := p.spells[id]
		return !live
	})
	if err := p.graph.Validate(); err != nil {
		return err
	}
	t.sess.Town.cityObjects = p.graph
	return nil
}

func (t *townScreen) validateCityShopValues(p *cityObjectProjection) error {
	if t.sess.Shop == nil {
		return nil
	}
	check := func(item ShopItem) error {
		if item.cityID == 0 {
			return nil
		}
		if item.Code == 0 || item.Count <= 0 || uint64(item.Count) > sim.MaxOriginalHoldingValues {
			return fmt.Errorf("bound city stock has invalid quantity")
		}
		row, err := cityMutationSource(p.graph, item.cityID, cityItemLocation{})
		if err != nil {
			return err
		}
		stack := sim.StackItem(item.Instance(), uint32(item.Count))
		if before, exists := p.items[item.cityID]; exists && !sim.StackStateEqual(before, stack) {
			return fmt.Errorf("bound city stock disagrees with shared Item %d", item.cityID)
		}
		p.items[item.cityID] = stack
		value := currentCityItem(item.Instance(), t.in.Table)
		node := p.graph.Items[row]
		if len(node.Effects) != len(value.Effects) || (node.Spell != 0) != value.SourceEquipment.Spell.Present {
			return fmt.Errorf("bound city stock has stale child edges")
		}
		for i, id := range node.Effects {
			if old, exists := p.effects[id]; id == 0 || exists && old != value.Effects[i] {
				return fmt.Errorf("bound city stock disagrees with shared Effect")
			}
			p.effects[id] = value.Effects[i]
		}
		if node.Spell != 0 {
			if old, exists := p.spells[node.Spell]; exists && old != value.SourceEquipment.Spell {
				return fmt.Errorf("bound city stock disagrees with shared Spell")
			}
			p.spells[node.Spell] = value.SourceEquipment.Spell
		}
		return nil
	}
	for _, place := range t.sess.Shop.table {
		if err := check(place.ShopItem); err != nil {
			return err
		}
	}
	for _, shelf := range t.sess.Shop.shelves {
		for _, item := range shelf {
			if err := check(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *townScreen) cityShopPackSelection(index int) (cityShopSelection, sim.ItemStack, error) {
	member := t.shopPartyMember(t.shopMemberIndex())
	if member == nil {
		return cityShopSelection{}, sim.ItemStack{}, fmt.Errorf("city item selection has no party member")
	}
	stacks := cityMemberStacks(*member, t.in.Table)
	root, err := cityMutationParty(t.sess.Town.cityObjects, member.ID)
	if err != nil || len(root.Pack) != len(stacks) || index < 0 || index >= len(stacks) {
		return cityShopSelection{}, sim.ItemStack{}, fmt.Errorf("city item selection has stale pack location")
	}
	stack := stacks[index]
	selection := cityShopSelection{Location: cityItemLocation{PartyID: member.ID, Kind: cityItemPack, Index: index}, ID: root.Pack[index], NativeID: stack.ObjectID, Count: stack.Count}
	if stack.Count == 0 || stack.Code == 0 {
		return cityShopSelection{}, sim.ItemStack{}, fmt.Errorf("city item selection has no current quantity")
	}
	if _, err := cityMutationSource(t.sess.Town.cityObjects, selection.ID, selection.Location); err != nil {
		return cityShopSelection{}, sim.ItemStack{}, err
	}
	return selection, stack.Clone(), nil
}

func mintCityShopItem(g *cityObjectTopology, value sim.ItemInstance) (sim.SavedObjectID, error) {
	id, err := mintCityObjectID(&g.NextID)
	if err != nil {
		return 0, err
	}
	row := cityItemTopology{ID: id}
	for range value.Effects {
		child, err := mintCityObjectID(&g.NextID)
		if err != nil {
			return 0, err
		}
		g.Effects, row.Effects = append(g.Effects, child), append(row.Effects, child)
	}
	if value.SourceEquipment.Spell.Present {
		row.Spell, err = mintCityObjectID(&g.NextID)
		if err != nil {
			return 0, err
		}
		g.Spells = append(g.Spells, row.Spell)
	}
	g.Items = append(g.Items, row)
	return id, g.Validate()
}

func (t *townScreen) cityShopBind(item ShopItem) (ShopItem, error) {
	if item.Code == 0 || item.Count <= 0 || uint64(item.Count) > sim.MaxOriginalHoldingValues {
		return ShopItem{}, fmt.Errorf("city item has invalid quantity")
	}
	g := t.sess.Town.cityObjects
	if item.cityID != 0 {
		if _, err := cityMutationSource(g, item.cityID, cityItemLocation{}); err != nil {
			return ShopItem{}, err
		}
		return item.Clone(), nil
	}
	if item.ObjectID == ^sim.SavedObjectID(0) {
		return ShopItem{}, fmt.Errorf("city item native identity is exhausted")
	}
	g.NextID = max(g.NextID, item.ObjectID+1)
	id, err := mintCityShopItem(g, currentCityItem(item.Instance(), t.in.Table))
	if err != nil {
		return ShopItem{}, err
	}
	item.cityID = id
	return item, nil
}

func (t *townScreen) cityShopSetPack(index int, stacks []sim.ItemStack, refresh bool) error {
	before := mapload.CloneParty([]mapload.PartyMember{t.sess.Carried[index]})[0]
	next := mapload.CloneParty([]mapload.PartyMember{before})[0]
	next = mapload.MaterializePartyCarry(next, t.in.Table)
	next.RetireOriginalHuman()
	if err := mapload.UpdatePartyLoadOrdered(before, &next, t.in.Table, refresh, false, stacks); err != nil {
		return err
	}
	t.sess.Carried[index] = next
	if t.sess.originalCity != nil {
		t.sess.originalCity.trade = nil
	}
	return nil
}

func (t *townScreen) cityShopSetNode(id sim.SavedObjectID, value sim.ItemInstance, count uint32, refresh bool) error {
	if id == 0 || value.Empty() || count == 0 || count > sim.MaxOriginalHoldingValues {
		return fmt.Errorf("city node update has invalid current operands")
	}
	for i, member := range t.sess.Carried {
		root, err := cityMutationParty(t.sess.Town.cityObjects, member.ID)
		if err != nil {
			return err
		}
		stacks := slices.Clone(cityMemberStacks(member, t.in.Table))
		if len(root.Pack) != len(stacks) {
			return fmt.Errorf("city node update has stale pack locations")
		}
		next := mapload.CloneParty([]mapload.PartyMember{member})[0]
		worn := mapload.MemberItemEquipment(next, t.in.Table)
		changed := false
		for at, held := range root.Pack {
			if held == id {
				v := value.Clone()
				v.ObjectID = stacks[at].ObjectID
				stacks[at], changed = sim.StackItem(v, count), true
			}
		}
		for at, held := range root.Worn {
			if held == id {
				v := value.Clone()
				v.ObjectID = worn[at].ObjectID
				worn[at], changed = v, true
			}
		}
		if changed {
			next = mapload.MaterializePartyCarry(next, t.in.Table)
			next.RetireOriginalHuman()
			next.Carry.EquippedItems = worn
			for slot, item := range worn {
				next.Carry.Equipped[slot] = item.Code
			}
			if err := mapload.UpdatePartyLoadOrdered(member, &next, t.in.Table, refresh, false, stacks); err != nil {
				return err
			}
			t.sess.Carried[i] = next
		}
	}
	for i := range t.sess.Town.cityObjects.Items {
		if t.sess.Town.cityObjects.Items[i].ID == id {
			t.sess.Town.cityObjects.Items[i].WornCount = count
		}
	}
	t.sess.Town.cityObjects.trimWornCounts()
	if t.sess.Shop != nil {
		update := func(item *ShopItem) {
			if item.cityID == id {
				native, origins, owners := item.ObjectID, item.cityOrigins, resizedShopOwners(item.owners, int32(count))
				*item = shopItemFromInstance(value, int32(count))
				item.cityID, item.cityOrigins, item.ObjectID = id, origins, native
				item.owners = owners
			}
		}
		for i := range t.sess.Shop.table {
			update(&t.sess.Shop.table[i].ShopItem)
		}
		for shelf := range t.sess.Shop.shelves {
			for i := range t.sess.Shop.shelves[shelf] {
				update(&t.sess.Shop.shelves[shelf][i])
			}
		}
	}
	return nil
}

func (t *townScreen) cityShopSplit(item ShopItem, from cityItemLocation) (ShopItem, error) {
	return t.cityShopSplitQuantity(item, from, 1)
}

func (t *townScreen) cityShopSplitQuantity(item ShopItem, from cityItemLocation, count int32) (ShopItem, error) {
	if count <= 0 || count >= item.Count {
		return ShopItem{}, fmt.Errorf("city partial take has no surviving quantity")
	}
	graph, id, err := splitCityItemRoot(t.sess.Town.cityObjects, item.cityID, from, cityItemLocation{})
	if err != nil {
		return ShopItem{}, err
	}
	value := sim.CloneSplitItemValue(item.Instance(), mapload.SpellRules(t.in.Table))
	if item.ObjectID != 0 {
		value.ObjectID = id
	}
	next := shopItemFromInstance(value, count)
	next.cityID, next.cityOrigins = id, slices.Clone(item.cityOrigins)
	next.owners, _ = takeShopOwners(item.owners, count)
	t.sess.Town.cityObjects = graph
	if err := t.cityShopSetNode(item.cityID, item.Instance(), uint32(item.Count-count), false); err != nil {
		return ShopItem{}, err
	}
	return next, nil
}

func (t *townScreen) cityShopTakePack(index int, whole, refresh bool) (ShopItem, error) {
	selected, stack, err := t.cityShopPackSelection(index)
	if err != nil {
		return ShopItem{}, err
	}
	item := shopItemFromInstance(stack.Instance(), int32(stack.Count))
	item.cityID, item.cityOrigins = selected.ID, []string{selected.Location.PartyID}
	if !whole && stack.Count > 1 {
		item, err = t.cityShopSplit(item, selected.Location)
		if err == nil && refresh {
			err = t.cityShopSetPack(t.shopMemberIndex(), cityMemberStacks(t.sess.Carried[t.shopMemberIndex()], t.in.Table), true)
		}
		return item, err
	}
	graph, err := moveCityItemRoot(t.sess.Town.cityObjects, selected.ID, selected.Location, cityItemLocation{})
	if err != nil {
		return ShopItem{}, err
	}
	stacks := slices.Clone(cityMemberStacks(t.sess.Carried[t.shopMemberIndex()], t.in.Table))
	stacks = slices.Delete(stacks, index, index+1)
	if err := t.cityShopSetPack(t.shopMemberIndex(), stacks, refresh); err != nil {
		return ShopItem{}, err
	}
	if load := t.sess.Carried[t.shopMemberIndex()].Carry.LiveLoad; load != nil {
		load.Inventory.InsertIndex = uint32(index)
	}
	t.sess.Town.cityObjects = graph
	return item, nil
}

func (t *townScreen) cityShopTakeShelf(shelf ShopShelf, index int, whole bool) (ShopItem, error) {
	if shelf < 0 || shelf >= numShopShelves || index < 0 || index >= len(t.sess.Shop.shelves[shelf]) {
		return ShopItem{}, fmt.Errorf("city shelf selection is outside stock")
	}
	item, err := t.cityShopBind(t.sess.Shop.shelves[shelf][index])
	if err != nil {
		return ShopItem{}, err
	}
	t.sess.Shop.shelves[shelf][index] = item
	if !whole && item.Count > 1 {
		return t.cityShopSplit(item, cityItemLocation{})
	}
	t.sess.Shop.shelves[shelf] = slices.Delete(t.sess.Shop.shelves[shelf], index, index+1)
	return item, nil
}

func (t *townScreen) cityShopTakeTable(index int, whole bool) (ShopPlace, error) {
	if index < 0 || index >= len(t.sess.Shop.table) {
		return ShopPlace{}, fmt.Errorf("city table selection is outside places")
	}
	place := t.sess.Shop.table[index]
	item, err := t.cityShopBind(place.ShopItem)
	if err != nil {
		return ShopPlace{}, err
	}
	t.sess.Shop.table[index].ShopItem = item
	if !whole && item.Count > 1 {
		item, err = t.cityShopSplit(item, cityItemLocation{})
	} else {
		t.sess.Shop.table = slices.Delete(t.sess.Shop.table, index, index+1)
	}
	place.ShopItem = item
	return place, err
}

// A gameplay merge keeps the explicitly selected destination and its children.
// Removing the detached input occurrence does not remove its other aliases.
func (t *townScreen) cityShopMerge(target, incoming ShopItem) (ShopItem, error) {
	if target.cityID == 0 || incoming.cityID == 0 || target.cityID == incoming.cityID || target.Count <= 0 || incoming.Count <= 0 ||
		uint64(target.Count)+uint64(incoming.Count) > sim.MaxOriginalHoldingValues {
		return ShopItem{}, fmt.Errorf("city merge has invalid identities or quantity")
	}
	for _, item := range []ShopItem{target, incoming} {
		if _, err := cityMutationSource(t.sess.Town.cityObjects, item.cityID, cityItemLocation{}); err != nil {
			return ShopItem{}, err
		}
	}
	target.Count += incoming.Count
	target.owners = append(slices.Clone(target.owners), incoming.owners...)
	for _, id := range incoming.cityOrigins {
		if !slices.Contains(target.cityOrigins, id) {
			target.cityOrigins = append(slices.Clone(target.cityOrigins), id)
		}
	}
	if err := t.cityShopSetNode(target.cityID, target.Instance(), uint32(target.Count), false); err != nil {
		return ShopItem{}, err
	}
	return target, nil
}

func (t *townScreen) cityShopPutTable(incoming ShopPlace) error {
	for i, place := range t.sess.Shop.table {
		if place.cityID == incoming.cityID || place.Mine != incoming.Mine || !place.Mine && place.From != incoming.From ||
			place.Price != incoming.Price || !sim.ItemEqual(place.Instance(), incoming.Instance()) {
			continue
		}
		target, err := t.cityShopBind(place.ShopItem)
		if err != nil {
			return err
		}
		t.sess.Shop.table[i].ShopItem = target
		target, err = t.cityShopMerge(target, incoming.ShopItem)
		if err != nil {
			return err
		}
		t.sess.Shop.table[i].ShopItem = target
		return nil
	}
	if len(t.sess.Shop.table) >= ShopTablePlaces {
		return fmt.Errorf("the table holds five and no more")
	}
	t.sess.Shop.table = append(t.sess.Shop.table, incoming)
	return nil
}

func (t *townScreen) cityShopReturnToShelf(shelf ShopShelf, incoming ShopItem) error {
	incoming.owners = nil
	if shelf < 0 || shelf >= numShopShelves {
		shelf = ShelfMagic
	}
	for i, item := range t.sess.Shop.shelves[shelf] {
		if item.cityID == incoming.cityID || !cityShelfJoins(item.Instance(), incoming.Instance(), t.in.Table) {
			continue
		}
		target, err := t.cityShopBind(item)
		if err != nil {
			return err
		}
		t.sess.Shop.shelves[shelf][i] = target
		target, err = t.cityShopMerge(target, incoming)
		if err != nil {
			return err
		}
		t.sess.Shop.shelves[shelf][i] = target
		return nil
	}
	t.sess.Shop.shelves[shelf] = append(t.sess.Shop.shelves[shelf], incoming.Clone())
	shopSortShelf(t.sess.Shop.shelves[shelf], t.sess.Shop.tbl)
	return nil
}

func (t *townScreen) cityShopStagePack(index int, whole bool) (ui.TownAction, error) {
	item, err := t.cityShopTakePack(index, whole, false)
	if err != nil {
		return ui.TownAction{}, err
	}
	item = t.ownShopItem(item, index)
	price := t.shopItemValue(item.Instance())
	if price != item.Price {
		item.Price = price
		if err := t.cityShopSetNode(item.cityID, item.Instance(), uint32(item.Count), false); err != nil {
			return ui.TownAction{}, err
		}
	}
	if err := t.cityShopPutTable(ShopPlace{ShopItem: item, Mine: true}); err != nil {
		return ui.TownAction{}, err
	}
	t.shopRevealTradeTable()
	return ui.TownAction{}, nil
}

func (t *townScreen) cityShopStageShelf(index int, whole bool) (ui.TownAction, error) {
	item, err := t.cityShopTakeShelf(t.shopShelf, index, whole)
	if err != nil {
		return ui.TownAction{}, err
	}
	if err := t.cityShopPutTable(ShopPlace{ShopItem: item, From: t.shopShelf}); err != nil {
		return ui.TownAction{}, err
	}
	t.shopRevealTradeTable()
	t.startTownResponse(shopItemSpeech(item))
	return ui.TownAction{}, nil
}

func (t *townScreen) cityShopOffTable(index int, whole bool) (ui.TownAction, error) {
	place, err := t.cityShopTakeTable(index, whole)
	if err != nil {
		return ui.TownAction{}, err
	}
	if place.Mine {
		err = t.cityShopPutPack(place.ShopItem, true)
	} else {
		err = t.cityShopReturnToShelf(place.From, place.ShopItem)
	}
	return ui.TownInfo("returned"), err
}

func (t *townScreen) cityShopBuy() (ui.TownAction, error) {
	if len(t.sess.Shop.table) == 0 {
		return ui.TownAction{}, nil
	}
	var spend int64
	for _, place := range t.sess.Shop.table {
		if !place.Mine {
			if place.Count <= 0 || place.Price < 0 {
				return ui.TownAction{}, fmt.Errorf("invalid merchant quantity or price")
			}
			spend += int64(place.Count) * int64(place.Price)
		}
	}
	if spend <= 0 || spend > int64(t.sess.Town.Gold()) || spend > 1<<31-1 {
		return ui.TownAction{}, fmt.Errorf("he will not sell you that: check the table and your purse")
	}
	for i := 0; i < len(t.sess.Shop.table); {
		if t.sess.Shop.table[i].Mine {
			i++
			continue
		}
		place, err := t.cityShopTakeTable(i, true)
		if err != nil {
			return ui.TownAction{}, err
		}
		if err := t.cityShopPutPack(place.ShopItem, true); err != nil {
			return ui.TownAction{}, err
		}
	}
	t.sess.Town.gold -= int(spend)
	return ui.TownInfo(fmt.Sprintf("bought for %d gold; you have %d left", spend, t.sess.Town.Gold())), nil
}

func (t *townScreen) cityShopClear() (ui.TownAction, error) {
	for len(t.sess.Shop.table) != 0 {
		place, err := t.cityShopTakeTable(0, true)
		if err != nil {
			return ui.TownAction{}, err
		}
		if place.Mine {
			err = t.cityShopPutPack(place.ShopItem, false)
		} else {
			err = t.cityShopReturnToShelf(place.From, place.ShopItem)
		}
		if err != nil {
			return ui.TownAction{}, err
		}
	}
	return ui.TownInfo("the table is cleared"), nil
}

func (t *townScreen) cityShopSell() (ui.TownAction, error) {
	if len(t.sess.Shop.table) == 0 {
		return ui.TownAction{}, nil
	}
	var paid int64
	owners := map[string]bool{}
	for _, place := range t.sess.Shop.table {
		if place.Mine && place.Price > 0 {
			if place.Count <= 0 {
				return ui.TownAction{}, fmt.Errorf("invalid sale quantity")
			}
			paid += (int64(place.Count)*int64(place.Price) + 1) / 2
			for _, id := range place.cityOrigins {
				owners[id] = true
			}
		}
	}
	if paid <= 0 || paid > 1<<31-1 || int64(t.sess.Town.Gold())+paid > 1<<31-1 {
		return ui.TownAction{}, fmt.Errorf("sale has no valid payout")
	}
	for i, member := range t.sess.Carried {
		if owners[member.ID] {
			if err := t.cityShopSetPack(i, cityMemberStacks(member, t.in.Table), true); err != nil {
				return ui.TownAction{}, err
			}
		}
	}
	for i := 0; i < len(t.sess.Shop.table); {
		place := t.sess.Shop.table[i]
		if !place.Mine || place.Price <= 0 {
			i++
			continue
		}
		place, err := t.cityShopTakeTable(i, true)
		if err != nil {
			return ui.TownAction{}, err
		}
		if err := t.cityShopReturnToShelf(shopShelfForItem(place.ShopItem, t.sess.Shop.tbl), place.ShopItem); err != nil {
			return ui.TownAction{}, err
		}
	}
	t.sess.Town.gold += int(paid)
	return ui.TownInfo(fmt.Sprintf("he pays %d; you have %d", paid, t.sess.Town.Gold())), nil
}

type cityShopOccurrence struct {
	ID, NativeID sim.SavedObjectID
	Count        uint32
	Ticket       uint32
}

func (t *townScreen) cityShopSourceEquipment(index, slot int, incoming ShopItem, toPack bool) (ShopItem, error) {
	memberIndex := t.shopMemberIndex()
	before := t.sess.Carried[memberIndex]
	g := t.sess.Town.cityObjects
	root, err := cityMutationParty(g, before.ID)
	if err != nil {
		return ShopItem{}, err
	}
	topology := sim.SourceEquipmentTopology{Pack: make([]uint64, len(root.Pack)), External: uint64(incoming.cityID)}
	if incoming.cityID != 0 {
		topology.ExternalCount = uint32(incoming.Count)
	}
	for i, id := range root.Pack {
		topology.Pack[i] = uint64(id)
	}
	for i, id := range root.Worn {
		topology.Worn[i] = uint64(id)
		if id != 0 {
			topology.WornCounts[i], err = t.cityShopNodeCount(id)
			if err != nil {
				return ShopItem{}, err
			}
		}
	}
	if index >= 0 {
		if index >= len(root.Pack) || root.Pack[index] == 0 {
			return ShopItem{}, fmt.Errorf("source equipment lacks selected pack root")
		}
	}
	var receipt sim.SourceEquipmentReceipt
	next, removed, ok := mapload.SourceTownEquipment(before, t.in.Table, index, slot, incoming.Instance(), toPack, sim.SourceEquipmentOperation{Topology: &topology, Receipt: &receipt})
	if !ok {
		return ShopItem{}, fmt.Errorf("cannot change that equipment")
	}
	stacks := cityMemberStacks(before, t.in.Table)
	pack := make([]cityShopOccurrence, len(stacks))
	values := map[sim.SavedObjectID]sim.ItemStack{}
	if len(pack) != len(root.Pack) {
		return ShopItem{}, fmt.Errorf("equipment receipt has stale pack input")
	}
	for i, stack := range stacks {
		pack[i] = cityShopOccurrence{ID: root.Pack[i], NativeID: stack.ObjectID, Count: stack.Count}
		if root.Pack[i] != 0 {
			values[root.Pack[i]] = stack.Clone()
		}
	}
	var worn [sim.EquipSlots]cityShopOccurrence
	for i, value := range mapload.MemberItemEquipment(before, t.in.Table) {
		if !value.Empty() {
			worn[i] = cityShopOccurrence{ID: root.Worn[i], NativeID: value.ObjectID, Count: topology.WornCounts[i]}
			if _, exists := values[root.Worn[i]]; !exists {
				values[root.Worn[i]] = sim.StackItem(value, topology.WornCounts[i])
			}
		}
	}
	if incoming.cityID != 0 {
		values[incoming.cityID] = sim.StackItem(incoming.Instance(), uint32(incoming.Count))
	}
	tickets := map[uint32]cityShopOccurrence{}
	used := map[uint32]bool{}
	changes := map[sim.SavedObjectID]sim.ItemStack{}
	spells := map[sim.SavedObjectID]sim.SourceItemSpell{}
	var external cityShopOccurrence
	for _, event := range receipt.Events {
		at := cityItemLocation{PartyID: before.ID, Index: event.Place.Index}
		switch event.Place.Kind {
		case sim.SourceEquipmentPack:
			at.Kind = cityItemPack
		case sim.SourceEquipmentWorn:
			at.Kind = cityItemWorn
		case sim.SourceEquipmentExternal:
			at = cityItemLocation{}
		}
		switch event.Kind {
		case sim.SourceEquipmentTake:
			if event.Ticket == 0 || tickets[event.Ticket].ID != 0 {
				return ShopItem{}, fmt.Errorf("equipment receipt repeats a take ticket")
			}
			var taken cityShopOccurrence
			switch event.Place.Kind {
			case sim.SourceEquipmentPack:
				if at.Index < 0 || at.Index >= len(pack) {
					return ShopItem{}, fmt.Errorf("equipment receipt take is outside pack")
				}
				taken = pack[at.Index]
				if event.Split != (taken.Count > 1) {
					return ShopItem{}, fmt.Errorf("equipment receipt split disagrees with source count")
				}
				if event.Split {
					var id sim.SavedObjectID
					g, id, err = splitCityItemRoot(g, taken.ID, at, cityItemLocation{})
					for i := range pack {
						if pack[i].ID == taken.ID {
							pack[i].Count--
						}
					}
					source := values[taken.ID]
					source.Count--
					values[taken.ID], changes[taken.ID] = source, source
					for i := range worn {
						if worn[i].ID == taken.ID {
							worn[i].Count = source.Count
						}
					}
					copy := sim.CloneSplitItemValue(source.Instance(), mapload.SpellRules(t.in.Table))
					taken.ID, taken.Count = id, 1
					if taken.NativeID != 0 {
						taken.NativeID = id
						copy.ObjectID = id
					}
					values[id] = sim.StackItem(copy, 1)
				} else {
					g, err = moveCityItemRoot(g, taken.ID, at, cityItemLocation{})
					pack = slices.Delete(pack, at.Index, at.Index+1)
				}
			case sim.SourceEquipmentWorn:
				if at.Index < 0 || at.Index >= len(worn) || event.Split {
					return ShopItem{}, fmt.Errorf("equipment receipt has invalid worn take")
				}
				taken = worn[at.Index]
				g, err = moveCityItemRoot(g, taken.ID, at, cityItemLocation{})
				worn[at.Index] = cityShopOccurrence{}
			case sim.SourceEquipmentExternal:
				if incoming.cityID == 0 || incoming.Count <= 0 || event.Split || event.Place.Index != 0 {
					return ShopItem{}, fmt.Errorf("equipment receipt lacks explicit incoming item")
				}
				taken = cityShopOccurrence{ID: incoming.cityID, NativeID: incoming.ObjectID, Count: uint32(incoming.Count)}
				_, err = cityMutationSource(g, taken.ID, cityItemLocation{})
			default:
				return ShopItem{}, fmt.Errorf("equipment receipt has unknown source place")
			}
			if err != nil {
				return ShopItem{}, err
			}
			if taken.ID == 0 || taken.Count == 0 || event.Count != taken.Count {
				return ShopItem{}, fmt.Errorf("equipment receipt took a null occurrence")
			}
			taken.Ticket = event.Ticket
			tickets[event.Ticket] = taken
		case sim.SourceEquipmentSpellWrite:
			taken := tickets[event.Ticket]
			row, err := cityMutationSource(g, taken.ID, cityItemLocation{})
			if err != nil {
				return ShopItem{}, err
			}
			var child sim.SavedObjectID
			if event.Spell.Present {
				child, err = mintCityObjectID(&g.NextID)
				if err != nil {
					return ShopItem{}, err
				}
				g.Spells = append(g.Spells, child)
			}
			g.Items[row].Spell = child
			spells[taken.ID] = event.Spell
			value := values[taken.ID]
			value.SourceEquipment.Spell = event.Spell
			values[taken.ID], changes[taken.ID] = value, value
		case sim.SourceEquipmentPut:
			taken := tickets[event.Ticket]
			taken.Count = values[taken.ID].Count
			if taken.ID == 0 || used[event.Ticket] || event.Split || event.Count != taken.Count {
				return ShopItem{}, fmt.Errorf("equipment receipt put has no unique detached ticket")
			}
			used[event.Ticket] = true
			if event.Merged {
				if at.Kind != cityItemPack || at.Index < 0 || at.Index >= len(pack) || pack[at.Index].ID == 0 || pack[at.Index].ID == taken.ID ||
					uint64(pack[at.Index].Count)+uint64(taken.Count) > sim.MaxOriginalHoldingValues {
					return ShopItem{}, fmt.Errorf("equipment receipt has invalid merge destination or quantity")
				}
				target := pack[at.Index]
				if _, err := cityMutationSource(g, target.ID, at); err != nil {
					return ShopItem{}, err
				}
				target.Count += taken.Count
				for i := range pack {
					if pack[i].ID == target.ID {
						pack[i].Count, pack[i].Ticket = target.Count, event.Ticket
					}
				}
				value := values[target.ID]
				value.Count = target.Count
				values[target.ID], changes[target.ID] = value, value
				for i := range worn {
					if worn[i].ID == target.ID {
						worn[i].Count = target.Count
					}
				}
				continue
			}
			g, err = moveCityItemRoot(g, taken.ID, cityItemLocation{}, at)
			if err != nil {
				return ShopItem{}, err
			}
			switch event.Place.Kind {
			case sim.SourceEquipmentPack:
				pack = slices.Insert(pack, at.Index, taken)
			case sim.SourceEquipmentWorn:
				worn[at.Index] = taken
			case sim.SourceEquipmentExternal:
				external = taken
			default:
				return ShopItem{}, fmt.Errorf("equipment receipt has unknown destination")
			}
		default:
			return ShopItem{}, fmt.Errorf("equipment receipt has unknown event")
		}
	}
	if len(used) != len(tickets) || len(next.Carry.OrderedStacks) != len(pack) {
		return ShopItem{}, fmt.Errorf("equipment receipt does not cover final occurrences")
	}
	for i, occurrence := range pack {
		value := next.Carry.OrderedStacks[i].Clone()
		if value.Count != occurrence.Count {
			return ShopItem{}, fmt.Errorf("equipment receipt changed an unexplained quantity")
		}
		value.ObjectID = occurrence.NativeID
		next.Carry.OrderedStacks[i] = value
		if occurrence.Ticket != 0 {
			changes[occurrence.ID] = value
		}
	}
	for i, occurrence := range worn {
		value := next.Carry.EquippedItems[i].Clone()
		if value.Empty() != (occurrence.ID == 0) {
			return ShopItem{}, fmt.Errorf("equipment receipt does not cover a worn root")
		}
		value.ObjectID = occurrence.NativeID
		next.Carry.EquippedItems[i] = value
		if occurrence.Ticket != 0 {
			changes[occurrence.ID] = sim.StackItem(value, values[occurrence.ID].Count)
		}
	}
	if external.ID != 0 {
		removed.ObjectID = external.NativeID
		changes[external.ID] = sim.StackItem(removed, external.Count)
	}
	// Restore the flat compatibility view without applying a second load delta.
	if err := mapload.UpdatePartyLoadOrdered(next, &next, t.in.Table, false, false, next.Carry.OrderedStacks); err != nil {
		return ShopItem{}, err
	}
	t.sess.Carried[memberIndex], t.sess.Town.cityObjects = next, g
	ids := make([]sim.SavedObjectID, 0, len(changes))
	for id := range changes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		value := changes[id]
		if spell, ok := spells[id]; ok {
			value.SourceEquipment.Spell = spell
		}
		if err := t.cityShopSetNode(id, value.Instance(), value.Count, true); err != nil {
			return ShopItem{}, err
		}
		if external.ID == id {
			removed = value.Instance()
		}
	}
	if slot == 1 {
		materializeStartingWeapon(&t.sess.Carried[memberIndex])
	}
	if t.sess.originalCity != nil {
		t.sess.originalCity.trade = nil
	}
	t.composeShopFaces()
	out := shopItemFromInstance(removed, int32(external.Count))
	out.cityID, out.cityOrigins = external.ID, []string{before.ID}
	return out, nil
}

func (t *townScreen) cityShopNativeWear(slot int, item ShopItem) error {
	memberIndex := t.shopMemberIndex()
	counts := map[sim.SavedObjectID]uint32{}
	initial, err := cityMutationParty(t.sess.Town.cityObjects, t.sess.Carried[memberIndex].ID)
	if err != nil {
		return err
	}
	for _, id := range initial.Worn {
		if id != 0 {
			counts[id], err = t.cityShopNodeCount(id)
			if err != nil {
				return err
			}
		}
	}
	return t.shopWearItemValuesOrdered(slot, item.Instance(), func(before mapload.PartyMember, old, other sim.ItemInstance, otherSlot int) error {
		root, err := cityMutationParty(t.sess.Town.cityObjects, before.ID)
		if err != nil {
			return err
		}
		g := t.sess.Town.cityObjects
		var returns []ShopItem
		for _, displaced := range []struct {
			slot  int
			value sim.ItemInstance
		}{{slot - 1, old}, {otherSlot - 1, other}} {
			if displaced.value.Empty() {
				continue
			}
			id := root.Worn[displaced.slot]
			from := cityItemLocation{PartyID: before.ID, Kind: cityItemWorn, Index: displaced.slot}
			if id == 0 {
				id, err = mintCityShopItem(g, currentCityItem(displaced.value, t.in.Table))
				from = cityItemLocation{}
				counts[id] = 1
			}
			if err != nil {
				return err
			}
			g, err = moveCityItemRoot(g, id, from, cityItemLocation{})
			if err != nil {
				return err
			}
			returned := shopItemFromInstance(displaced.value, int32(counts[id]))
			returned.cityID = id
			returns = append(returns, returned)
		}
		g, err = moveCityItemRoot(g, item.cityID, cityItemLocation{}, cityItemLocation{PartyID: before.ID, Kind: cityItemWorn, Index: slot - 1})
		if err != nil {
			return err
		}
		root, _ = cityMutationParty(g, before.ID)
		worn := mapload.MemberItemEquipment(t.sess.Carried[memberIndex], t.in.Table)
		for i, value := range worn {
			if !value.Empty() && root.Worn[i] == 0 {
				id, err := mintCityShopItem(g, currentCityItem(value, t.in.Table))
				if err != nil {
					return err
				}
				root.Worn[i] = id
			}
		}
		t.sess.Town.cityObjects = g
		for _, returned := range returns {
			if err := t.cityShopPutPack(returned, false); err != nil {
				return err
			}
		}
		next := mapload.MaterializePartyCarry(t.sess.Carried[memberIndex], t.in.Table)
		if err := mapload.UpdatePartyLoadOrdered(before, &next, t.in.Table, true, true, cityMemberStacks(next, t.in.Table)); err != nil {
			return err
		}
		t.sess.Carried[memberIndex] = next
		return nil
	})
}

func (t *townScreen) cityShopUse(item ShopItem, take func() (ShopItem, error), packIndex int) (ui.TownAction, error) {
	if user, result, refusal, potion := t.shopPotionUser(item.Instance()); potion {
		if user == nil {
			return ui.TownAction{}, fmt.Errorf("%s", refusal.Msg)
		}
		if _, err := take(); err != nil {
			return ui.TownAction{}, err
		}
		if !commitTownPotion(&t.sess.Carried[t.shopMemberIndex()], result) {
			return ui.TownAction{}, fmt.Errorf("cannot use that potion")
		}
		return ui.TownInfo("used"), nil
	}
	if reader, spell, refusal, readable := t.shopBookReader(item.Instance()); readable {
		if reader == nil {
			return ui.TownAction{}, fmt.Errorf("%s", refusal.Msg)
		}
		if _, err := take(); err != nil {
			return ui.TownAction{}, err
		}
		t.shopLearnBook(&t.sess.Carried[t.shopMemberIndex()], spell)
		return ui.TownInfo("learned"), nil
	}
	slot, ok, refusal := t.shopWearInstance(item.Instance())
	if !ok {
		return ui.TownAction{}, fmt.Errorf("%s", refusal.Msg)
	}
	if mapload.HasSourceActor(t.sess.Carried[t.shopMemberIndex()]) && packIndex >= 0 {
		_, err := t.cityShopSourceEquipment(packIndex, slot, ShopItem{}, false)
		return ui.TownInfo("worn"), err
	}
	item, err := take()
	if err != nil {
		return ui.TownAction{}, err
	}
	if mapload.HasSourceActor(t.sess.Carried[t.shopMemberIndex()]) {
		_, err = t.cityShopSourceEquipment(-1, slot, item, false)
	} else {
		err = t.cityShopNativeWear(slot, item)
	}
	return ui.TownInfo("worn"), err
}

func (t *townScreen) cityShopEquipPack(index int) (ui.TownAction, error) {
	_, stack, err := t.cityShopPackSelection(index)
	if err != nil {
		return ui.TownAction{}, err
	}
	return t.cityShopUse(shopItemFromInstance(stack.Instance(), int32(stack.Count)), func() (ShopItem, error) { return t.cityShopTakePack(index, false, true) }, index)
}

func (t *townScreen) cityShopEquipShelf(index int) (ui.TownAction, error) {
	if t.shopChosen < 0 || t.shopChosen >= len(shopRoomShelves) || !shopRoomShelves[t.shopChosen].stocked {
		return ui.TownAction{}, fmt.Errorf("no selected merchant shelf")
	}
	shelf := shopRoomShelves[t.shopChosen].shelf
	if index < 0 || index >= len(t.sess.Shop.shelves[shelf]) {
		return ui.TownAction{}, fmt.Errorf("shelf selection is outside stock")
	}
	item := t.sess.Shop.shelves[shelf][index]
	if item.Price < 0 {
		return ui.TownAction{}, fmt.Errorf("merchant item has no valid price")
	}
	if item.Price > 0 && int64(t.sess.Town.Gold()) < int64(item.Price) {
		return ui.TownAction{}, fmt.Errorf("you cannot afford that")
	}
	action, err := t.cityShopUse(item, func() (ShopItem, error) { return t.cityShopTakeShelf(shelf, index, false) }, -1)
	if err == nil {
		t.sess.Town.gold -= int(item.Price)
		action.Msg = fmt.Sprintf("bought and %s for %d gold; you have %d left", action.Msg, item.Price, t.sess.Town.Gold())
	}
	return action, err
}

func (t *townScreen) cityShopEquipTable(index int) (ui.TownAction, error) {
	if index < 0 || index >= len(t.sess.Shop.table) {
		return ui.TownAction{}, fmt.Errorf("table selection is outside places")
	}
	place := t.sess.Shop.table[index]
	if !place.Mine && place.Price < 0 {
		return ui.TownAction{}, fmt.Errorf("merchant item has no valid price")
	}
	if !place.Mine && place.Price > 0 && int64(t.sess.Town.Gold()) < int64(place.Price) {
		return ui.TownAction{}, fmt.Errorf("you cannot afford that")
	}
	action, err := t.cityShopUse(place.ShopItem, func() (ShopItem, error) {
		p, err := t.cityShopTakeTable(index, false)
		return p.ShopItem, err
	}, -1)
	if err == nil && !place.Mine {
		t.sess.Town.gold -= int(place.Price)
		action.Msg = fmt.Sprintf("bought and %s for %d gold; you have %d left", action.Msg, place.Price, t.sess.Town.Gold())
	}
	return action, err
}

func (t *townScreen) cityShopUnequip(slot int, table bool) (ui.TownAction, error) {
	memberIndex := t.shopMemberIndex()
	if memberIndex < 0 || memberIndex >= len(t.sess.Carried) || slot < 1 || slot > sim.EquipSlots {
		return ui.TownAction{}, fmt.Errorf("invalid equipment selection")
	}
	worn := mapload.MemberItemEquipment(t.sess.Carried[memberIndex], t.in.Table)
	if worn[slot-1].Empty() {
		if slot != 1 {
			return ui.TownAction{}, nil
		}
		if _, ok := t.shopWeaponFallbackCode(memberIndex); !ok {
			return ui.TownAction{}, nil
		}
	}
	var item ShopItem
	var err error
	if mapload.HasSourceActor(t.sess.Carried[memberIndex]) {
		item, err = t.cityShopSourceEquipment(-1, slot, ShopItem{}, !table)
	} else {
		item, err = t.cityShopNativeUnequip(slot, !table)
	}
	if err != nil {
		return ui.TownAction{}, err
	}
	if table {
		item = t.ownShopItem(item, -1)
		if err := t.cityShopPutTable(ShopPlace{ShopItem: item, Mine: true}); err != nil {
			return ui.TownAction{}, err
		}
		t.shopRevealTradeTable()
		return ui.TownInfo("on the table"), nil
	}
	return ui.TownInfo("off, into the pack"), nil
}

func (t *townScreen) cityShopNativeUnequip(slot int, toPack bool) (ShopItem, error) {
	i := t.shopMemberIndex()
	member := t.shopPartyMember(i)
	before := mapload.CloneParty([]mapload.PartyMember{*member})[0]
	worn := t.shopWornItemSlots(i)
	value := worn[slot-1]
	if value.Empty() && slot == 1 {
		if code, ok := t.shopWeaponFallbackCode(i); ok {
			value = mapload.ItemInstanceFromCode(code, t.in.Table)
		}
	}
	if value.Empty() {
		return ShopItem{}, fmt.Errorf("nothing is worn in that slot")
	}
	if toPack && !t.shopHasActorContainer() {
		return ShopItem{}, fmt.Errorf("no pack")
	}
	root, err := cityMutationParty(t.sess.Town.cityObjects, member.ID)
	if err != nil {
		return ShopItem{}, err
	}
	g := t.sess.Town.cityObjects
	item := shopItemFromInstance(value, 1)
	item.cityID, item.cityOrigins = root.Worn[slot-1], []string{member.ID}
	from := cityItemLocation{PartyID: member.ID, Kind: cityItemWorn, Index: slot - 1}
	if item.cityID == 0 {
		item.cityID, err = mintCityShopItem(g, currentCityItem(value, t.in.Table))
		from = cityItemLocation{}
	} else {
		var count uint32
		count, err = t.cityShopNodeCount(item.cityID)
		item.Count = int32(count)
	}
	if err != nil {
		return ShopItem{}, err
	}
	g, err = moveCityItemRoot(g, item.cityID, from, cityItemLocation{})
	if err != nil {
		return ShopItem{}, err
	}
	worn[slot-1] = sim.ItemInstance{}
	syncShopWornCodes(member, *worn)
	t.sess.Town.cityObjects = g
	if toPack {
		if err := t.cityShopPutPack(item, false); err != nil {
			return ShopItem{}, err
		}
	}
	member, worn = t.shopPartyMember(i), t.shopWornItemSlots(i)
	if slot == 1 {
		materializeStartingWeapon(member)
	}
	syncShopWornCodes(member, *worn)
	next := mapload.MaterializePartyCarry(*member, t.in.Table)
	next.RetireOriginalHuman()
	if err := mapload.UpdatePartyLoadOrdered(before, &next, t.in.Table, true, true, cityMemberStacks(next, t.in.Table)); err != nil {
		return ShopItem{}, err
	}
	refreshDerivedPartyBook(&next, t.in.Table)
	t.sess.Carried[i] = next
	t.composeShopFaces()
	return item, nil
}

func (t *townScreen) cityShopPutPack(item ShopItem, refresh bool) error {
	memberIndex := t.shopMemberIndex()
	member := t.shopPartyMember(memberIndex)
	if member == nil || !t.shopHasActorContainer() {
		return fmt.Errorf("no pack")
	}
	var err error
	item, err = t.cityShopBind(item)
	if err != nil {
		return err
	}
	stacks := slices.Clone(cityMemberStacks(*member, t.in.Table))
	root, err := cityMutationParty(t.sess.Town.cityObjects, member.ID)
	if err != nil || len(root.Pack) != len(stacks) {
		return fmt.Errorf("city pack insertion has stale roots")
	}
	for i, stack := range stacks {
		if stack.Code == 0 || stack.Count == 0 || root.Pack[i] == item.cityID || !cityPackJoins(stack.Instance(), item.Instance(), t.in.Table) {
			continue
		}
		target := shopItemFromInstance(stack.Instance(), int32(stack.Count))
		target.cityID = root.Pack[i]
		if _, err := t.cityShopMerge(target, item); err != nil {
			return err
		}
		if refresh {
			return t.cityShopSetPack(memberIndex, cityMemberStacks(t.sess.Carried[memberIndex], t.in.Table), true)
		}
		return nil
	}
	at := len(stacks)
	if member.Carry != nil && member.Carry.LiveLoad != nil {
		at = int(min(member.Carry.LiveLoad.Inventory.InsertIndex, uint32(len(stacks))))
	}
	graph, err := moveCityItemRoot(t.sess.Town.cityObjects, item.cityID, cityItemLocation{}, cityItemLocation{PartyID: member.ID, Kind: cityItemPack, Index: at})
	if err != nil {
		return err
	}
	stacks = slices.Insert(stacks, at, sim.StackItem(item.Instance(), uint32(item.Count)))
	if err := t.cityShopSetPack(memberIndex, stacks, refresh); err != nil {
		return err
	}
	t.sess.Town.cityObjects = graph
	return nil
}
