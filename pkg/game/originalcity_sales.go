package game

import (
	"fmt"
	"reflect"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// SnapshotCitySale names one original-container removal. Commit ends one Sell
// operation: several pack clicks refresh the Human once, not once per click.
type SnapshotCitySale struct {
	Position      uint32
	Quantity      uint16
	AfterTraining uint32
	Commit        bool
}

// originalCityTrade is transient. A native SAVE during a trade cannot turn its
// uncommitted inventory removal into a completed source sale on later LOAD.
type originalCityTrade struct {
	partyID   string
	sales     []SnapshotCitySale
	staged    mapload.PartyMember
	completed mapload.PartyMember
	table     []ShopPlace
	gold      int
}

func (s *originalCitySaveState) saleBinding(id string) *originalCityBinding {
	if s == nil || s.unavailable != nil {
		return nil
	}
	for i := range s.bindings {
		if s.bindings[i].partyID == id {
			return &s.bindings[i]
		}
	}
	return nil
}

func cloneCitySales(s []SnapshotCitySale) []SnapshotCitySale {
	return append([]SnapshotCitySale(nil), s...)
}

// Interpret immutable source items under the owning baseline's import policy.
// idx is the object's own position in b.inventory. This creates a
// comparison/replay value; it never clears a live party item or changes the
// source weight used by the separate completed-sale load update.
func (b originalCityBinding) inventoryInstance(idx int) sim.ItemInstance {
	p := b.inventory[idx].Piece
	item := originalItemInstance(p, nil)
	if b.partyImportVersion > 0 && b.partyImportVersion < 5 {
		item.SourceEquipment = sim.SourceEquipment{}
	} else {
		// sav.CityProvenance.Inventory builds Piece from the city document's own
		// generic container item object, which never carries the equip-time
		// reached columns (own-kind/attack/defence), a bound weapon spell, or a
		// table-bound Definition — those live only in the character's own
		// baseline holdings decode. It does carry this item's own Effects/
		// UnsupportedEffectStates, so EffectsUnsupported is left sourced from p:
		// blending it away would hide a genuine future city/baseline mismatch
		// instead of reporting one.
		//
		// Class+DefinitionRow is not a unique key: two distinct source stacks
		// can share it (DIV-762 keeps the same weapon row in two materials in
		// distinct stacks), so a keyed search can patch an item from the wrong
		// stack. citySaleRemainders builds expanded by walking b.inventory in
		// container order and repeating each position Stack times, and its
		// DeepEqual against MemberCarriedItems(b.baseline) is what actually
		// proves a patch correct — so the patch takes the baseline entry at
		// this object's own running position instead: the count of source
		// units at every earlier position is this object's first baseline
		// index, and every unit in one stack shares one set of patched columns.
		start := 0
		for _, prior := range b.inventory[:idx] {
			start += int(prior.Piece.Stack)
		}
		if baseline := mapload.MemberCarriedItems(b.baseline, nil); start < len(baseline) {
			source := baseline[start].SourceEquipment
			item.SourceEquipment.OwnKind = source.OwnKind
			item.SourceEquipment.Attack = source.Attack
			item.SourceEquipment.Defence = source.Defence
			item.SourceEquipment.Spell = source.Spell
			item.SourceEquipment.Definition = source.Definition
		}
	}
	if b.partyImportVersion > 0 && b.partyImportVersion < 4 {
		item.Weight, item.WeightPresent = 0, false
	}
	return item
}

// citySaleRemainders verifies the original container-to-native expansion and
// rejects every grouping collision before a position can authorize a removal.
// Equality never conflates source objects merely because code/price agree.
func citySaleRemainders(b originalCityBinding, sales []SnapshotCitySale) ([]uint16, error) {
	if b.inventoryErr != nil {
		return nil, b.inventoryErr
	}
	if len(sales) > maxCityTraining {
		return nil, fmt.Errorf("original city sale history exceeds bound")
	}
	var expanded []sim.ItemInstance
	var elements uint64
	counts := make([]uint16, len(b.inventory))
	for i, source := range b.inventory {
		elements += uint64(source.Piece.Stack) * (1 + uint64(len(source.Piece.Effects)))
		if elements > 1<<16 {
			return nil, fmt.Errorf("original city sale source expansion exceeds bound")
		}
		counts[i] = source.Piece.Stack
		for n := uint16(0); n < source.Piece.Stack; n++ {
			expanded = append(expanded, b.inventoryInstance(i))
		}
	}
	if !reflect.DeepEqual(cloneOriginalItems(expanded), cloneOriginalItems(mapload.MemberCarriedItems(b.baseline, nil))) {
		return nil, fmt.Errorf("original city pack is not its exact source container")
	}
	var training uint32
	for i, sale := range sales {
		if uint64(sale.Position) >= uint64(len(counts)) || sale.Quantity == 0 || sale.Quantity > counts[sale.Position] || sale.AfterTraining > uint32(len(b.training)) || sale.AfterTraining < training ||
			i > 0 && !sales[i-1].Commit && sale.AfterTraining != training || b.salesVersion == 0 && sale.AfterTraining != uint32(len(b.training)) {
			return nil, fmt.Errorf("original city sale has invalid source position, count or training order")
		}
		training = sale.AfterTraining
		source := b.inventory[sale.Position]
		if !source.Exclusive || len(source.Piece.UnsupportedEffectStates) != 0 {
			return nil, fmt.Errorf("original city sale item is aliased or has unsupported effects")
		}
		selected := b.inventoryInstance(int(sale.Position))
		for i := range b.inventory {
			if i != int(sale.Position) && counts[i] != 0 && citySaleGroups(selected, b.inventoryInstance(i)) {
				return nil, fmt.Errorf("original city sale display group spans distinct source objects")
			}
		}
		counts[sale.Position] -= sale.Quantity
	}
	return counts, nil
}

func citySaleGroups(a, b sim.ItemInstance) bool { return a.Price == b.Price && sim.ItemEqual(a, b) }

// legacyCitySoldMember preserves the published WIP history policy. Such native
// records remain retired; loading them must not invent a missing Sell boundary.
func legacyCitySoldMember(b originalCityBinding, sales []SnapshotCitySale) (mapload.PartyMember, error) {
	b.sales = nil
	m, err := b.expectedMember()
	if err != nil {
		return m, err
	}
	if len(sales) == 0 {
		return m, nil
	}
	if _, err := citySaleRemainders(b, sales); err != nil {
		return m, err
	}
	if _, ok := m.OriginalHumanState(); !ok {
		return m, fmt.Errorf("original city sale has no source-backed Human")
	}
	items := mapload.MemberCarriedItems(m, nil)
	for _, sale := range sales {
		selected := b.inventoryInstance(int(sale.Position))
		kept := make([]sim.ItemInstance, 0, len(items))
		left := int(sale.Quantity)
		for _, item := range items {
			if left > 0 && citySaleGroups(selected, item) {
				left--
				continue
			}
			kept = append(kept, item.Clone())
		}
		if left != 0 {
			return m, fmt.Errorf("original city sale exceeds native pack")
		}
		items = kept
	}
	m.RetireOriginalHuman()
	setCityMemberItems(&m, items)
	return m, nil
}

// replayCityMember starts from immutable source fields, then interleaves each
// admitted school operation and completed trade. Unfinished table moves are
// transient and never enter persisted history or SAV admission.
func replayCityMember(b originalCityBinding) (mapload.PartyMember, error) {
	if b.salesVersion == 0 && len(b.sales) != 0 {
		return legacyCitySoldMember(b, b.sales)
	}
	m := mapload.CloneParty([]mapload.PartyMember{b.baseline})[0]
	if len(b.training) == 0 && len(b.sales) == 0 {
		return m, nil
	}
	if len(b.training) > maxCityTraining || b.salesVersion > 1 {
		return m, fmt.Errorf("original city operation history exceeds bound or version")
	}
	if len(b.sales) != 0 {
		if _, err := citySaleRemainders(b, b.sales); err != nil {
			return m, err
		}
	}
	if len(b.sales) != 0 && !b.sales[len(b.sales)-1].Commit {
		return m, fmt.Errorf("original city sale history has no completed Sell")
	}
	h, ok := m.OriginalHumanState()
	if !ok {
		return m, fmt.Errorf("original city operations have no source-backed Human")
	}
	if err := h.ProjectionError(); err != nil {
		return m, err
	}
	items := mapload.MemberCarriedItems(m, nil)
	pos := 0
	for trained := 0; trained <= len(b.training); trained++ {
		for pos < len(b.sales) && int(b.sales[pos].AfterTraining) == trained {
			sale := b.sales[pos]
			source := b.inventory[sale.Position]
			selected := b.inventoryInstance(int(sale.Position))
			kept, left := make([]sim.ItemInstance, 0, len(items)), int(sale.Quantity)
			for _, item := range items {
				if left > 0 && citySaleGroups(selected, item) {
					left--
				} else {
					kept = append(kept, item.Clone())
				}
			}
			if left != 0 {
				return m, fmt.Errorf("original city sale exceeds native pack")
			}
			items = kept
			h.InventoryWeight -= int32(source.Weight) * int32(sale.Quantity)
			if sale.Commit {
				var derived bool
				var err error
				h, derived, err = h.RefreshInventoryLoad()
				if err != nil {
					return m, err
				}
				if err := h.ProjectionError(); err != nil {
					return m, err
				}
				setCityMemberItems(&m, items)
				mapload.ApplyOriginalHuman(&m, h)
				if derived {
					refreshPartyBook(b.gameRules, &m, b.spellRules)
				}
			}
			pos++
		}
		if trained == len(b.training) {
			break
		}
		var err error
		h, err = h.Train(int(b.training[trained]))
		if err != nil {
			return m, err
		}
		if err := h.ProjectionError(); err != nil {
			return m, err
		}
		mapload.ApplyOriginalHuman(&m, h)
		refreshPartyBook(b.gameRules, &m, b.spellRules)
	}
	if err := h.ProjectionError(); err != nil {
		return m, err
	}
	return mapload.CloneParty([]mapload.PartyMember{m})[0], nil
}

func setCityMemberItems(m *mapload.PartyMember, items []sim.ItemInstance) {
	before := mapload.CloneParty([]mapload.PartyMember{*m})[0]
	m.Carry.ItemInstances = cloneOriginalItems(items)
	m.Carry.Items = make([]uint16, len(items))
	for i, item := range items {
		m.Carry.Items[i] = item.Code
	}
	mapload.UpdatePartyLoad(before, m, nil, false, false)
}

func (t *townScreen) prepareOriginalCitySale(item sim.ItemInstance, quantity int32) *originalCityTrade {
	state := t.sess.originalCity
	party := t.shopParty()
	if state == nil || len(party) == 0 || quantity <= 0 || quantity > 65535 {
		return nil
	}
	member := party[t.shopMemberIndex()]
	b := state.saleBinding(member.ID)
	if b == nil || b.baseline.OriginalHuman == nil || b.salesVersion == 0 && len(b.sales) != 0 {
		return nil
	}
	sales := cloneCitySales(b.sales)
	expected, err := b.expectedMember()
	if err != nil {
		return nil
	}
	if pending := state.trade; pending != nil {
		if pending.partyID != member.ID || pending.gold != t.sess.Town.Gold() || !reflect.DeepEqual(pending.table, t.sess.Shop.Table()) {
			return nil
		}
		expected, sales = pending.staged, cloneCitySales(pending.sales)
	} else if len(t.sess.Shop.Table()) != 0 {
		return nil
	}
	if !reflect.DeepEqual(member, expected) {
		return nil
	}
	counts, err := citySaleRemainders(*b, sales)
	if err != nil {
		return nil
	}
	position := -1
	for i := range b.inventory {
		if counts[i] != 0 && citySaleGroups(item, b.inventoryInstance(i)) {
			if position != -1 {
				return nil
			}
			position = i
		}
	}
	if position < 0 {
		return nil
	}
	sales = append(sales, SnapshotCitySale{Position: uint32(position), Quantity: uint16(quantity), AfterTraining: uint32(len(b.training))})
	// Stage the exact native removal with a retired basis. SAV stays closed
	// while goods are on the table. Compute completion before the first write.
	staged := mapload.CloneParty([]mapload.PartyMember{member})[0]
	items, left := mapload.MemberCarriedItems(staged, nil), int(quantity)
	kept := make([]sim.ItemInstance, 0, len(items))
	for _, carried := range items {
		if left > 0 && citySaleGroups(carried, item) {
			left--
		} else {
			kept = append(kept, carried.Clone())
		}
	}
	if left != 0 {
		return nil
	}
	staged.RetireOriginalHuman()
	setCityMemberItems(&staged, kept)
	complete := *b
	complete.sales, complete.salesVersion = cloneCitySales(sales), 1
	complete.sales[len(complete.sales)-1].Commit = true
	completed, err := complete.expectedMember()
	if err != nil {
		return nil
	}
	if err := state.validateBookTraining(member.ID, completed); err != nil {
		return nil
	}
	return &originalCityTrade{partyID: member.ID, sales: sales, staged: staged, completed: completed, gold: t.sess.Town.Gold()}
}

func (t *townScreen) stageOriginalCitySale(candidate *originalCityTrade) {
	if t.sess.originalCity == nil {
		return
	}
	t.sess.originalCity.trade = nil
	if candidate == nil {
		return
	}
	party := t.shopParty()
	if len(party) == 0 || !reflect.DeepEqual(party[t.shopMemberIndex()], candidate.staged) {
		return
	}
	for _, place := range t.sess.Shop.Table() {
		if !place.Mine || place.Price <= 0 {
			return
		}
	}
	candidate.table = t.sess.Shop.Table()
	t.sess.originalCity.trade = candidate
}

func (t *townScreen) finishOriginalCitySale(candidate *originalCityTrade, before []ShopPlace, gold, paid int) {
	if candidate == nil || t.sess.originalCity == nil || gold != candidate.gold || t.sess.Town.Gold() != gold+paid ||
		len(t.sess.Shop.Table()) != 0 || !reflect.DeepEqual(before, candidate.table) {
		return
	}
	b := t.sess.originalCity.saleBinding(candidate.partyID)
	party := t.shopParty()
	if b == nil || len(party) == 0 || party[t.shopMemberIndex()].ID != candidate.partyID {
		return
	}
	for i, member := range party {
		if member.ID == candidate.partyID && reflect.DeepEqual(member, candidate.staged) {
			t.shopParty()[i] = candidate.completed
			b.sales = cloneCitySales(candidate.sales)
			b.sales[len(b.sales)-1].Commit = true
			b.salesVersion = 1
			return
		}
	}
}

func bindCityInventory(document originalCityDocument, b *originalCityBinding) {
	if p, ok := document.(*sav.CityProvenance); ok {
		b.inventory, b.inventoryErr = p.Inventory(b.character.Identity)
	}
}

// A valid source history cannot lend its authority to a different current
// member. Native producers may retire the last valid basis, but may not rewrite
// that basis. Validate before prepareRestore publishes any town or party state.
func (s *originalCitySaveState) validateSalesParty(party []mapload.PartyMember) error {
	if s == nil {
		return nil
	}
	for _, b := range s.bindings {
		if b.salesVersion != 1 || len(b.sales) == 0 {
			continue
		}
		expected, err := b.expectedMember()
		if err != nil {
			return err
		}
		found := false
		for _, member := range party {
			if member.ID != b.partyID {
				continue
			}
			if found || member.OriginalHuman == nil {
				return fmt.Errorf("original city sale has missing or repeated current member")
			}
			found = true
			if member.OriginalHuman.Retired {
				retained := *member.OriginalHuman
				retained.Retired = false
				if !reflect.DeepEqual(&retained, expected.OriginalHuman) {
					return fmt.Errorf("original city sale has corrupt retired Human basis")
				}
			} else if !reflect.DeepEqual(member, expected) {
				return fmt.Errorf("original city sale history differs from current member")
			}
		}
		if !found {
			return fmt.Errorf("original city sale has no current member")
		}
	}
	return nil
}
