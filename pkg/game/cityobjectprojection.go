package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Values exist only for this detached SAVE construction. The current party is
// their authority; all views of one identity must agree before records exist.
type cityObjectProjection struct {
	graph   *cityObjectTopology
	items   map[sim.SavedObjectID]sim.ItemStack
	effects map[sim.SavedObjectID]sim.ItemEffect
	spells  map[sim.SavedObjectID]sim.SourceItemSpell
	indices map[sim.SavedObjectID]uint16
	chunks  map[sim.SavedObjectID][]uint16
}

func (p *cityObjectProjection) currentItemValue(node cityItemTopology, native sim.ItemStack, table *mapload.Table) (sim.ItemInstance, error) {
	value := currentCityItem(native.Instance(), table)
	if node.Spell == 0 {
		value.SourceEquipment.Spell = sim.SourceItemSpell{}
		return value, nil
	}
	spell, present := p.spells[node.Spell]
	if !present {
		return sim.ItemInstance{}, fmt.Errorf("city current Item %d has no owned Spell value", node.ID)
	}
	value.SourceEquipment.Spell = spell
	return value, nil
}

func cityMemberStacks(p mapload.PartyMember, table *mapload.Table) []sim.ItemStack {
	if p.Carry != nil && p.Carry.OrderedStacks != nil {
		return p.Carry.OrderedStacks
	}
	return sim.FoldItems(mapload.MemberCarriedItems(p, table))
}

func cityMemberBook(p mapload.PartyMember, table *mapload.Table) [28]sim.SourceItemSpell {
	var out [28]sim.SourceItemSpell
	for slot := range out {
		id := uint16(slot + 1)
		if p.KnownSpells&(1<<id) == 0 {
			continue
		}
		v := p.Book.Slots[slot]
		if p.Book.State == sim.BookLegacy {
			rules, _ := nativeCityMemberSpells(p, table)
			for _, r := range rules {
				if r.ID == id {
					v = sim.BookSpell{Range: r.MaxRange, ManaCost: uint16(r.ManaCost)}
					if r.Defensive {
						v.Defensive = 1
					}
				}
			}
		}
		out[slot] = sim.SourceItemSpell{Present: true, ID: uint8(id), Range: v.Range, Defensive: v.Defensive, ManaCost: v.ManaCost}
	}
	return out
}

func newCityObjectProjection(source *cityObjectTopology, party []mapload.PartyMember, table *mapload.Table) (*cityObjectProjection, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	}
	g := source.Clone()
	if g == nil {
		g = &cityObjectTopology{Version: cityObjectTopologyVersion, NextID: 1}
	}
	p := &cityObjectProjection{graph: g, items: map[sim.SavedObjectID]sim.ItemStack{},
		effects: map[sim.SavedObjectID]sim.ItemEffect{}, spells: map[sim.SavedObjectID]sim.SourceItemSpell{}}
	roots, books := map[string]cityPartyObjectRoots{}, map[string]cityBookTopology{}
	nodes, kinds := map[sim.SavedObjectID]int{}, map[sim.SavedObjectID]uint8{}
	for i, row := range g.Items {
		nodes[row.ID], kinds[row.ID] = i, 1
	}
	for _, id := range g.Effects {
		kinds[id] = 2
	}
	for _, id := range g.Spells {
		kinds[id] = 3
	}
	for _, row := range g.Roots {
		roots[string(row.PartyID)] = row
	}
	for _, row := range g.Books {
		books[string(row.PartyID)] = row
	}
	// Native handles and topology IDs are separate when the handle is absent.
	// Reserve every existing native handle before constructing missing nodes.
	for _, member := range party {
		if err := mapload.ValidatePartyLoad(member); err != nil {
			return nil, fmt.Errorf("city current party load: %w", err)
		}
		reserve := func(id sim.SavedObjectID) error {
			if id == ^sim.SavedObjectID(0) {
				return fmt.Errorf("city item identity is exhausted")
			}
			g.NextID = max(g.NextID, id+1)
			return nil
		}
		for _, v := range cityMemberStacks(member, table) {
			if err := reserve(v.ObjectID); err != nil {
				return nil, err
			}
		}
		for _, v := range cityMemberEquipment(member, table) {
			if err := reserve(v.ObjectID); err != nil {
				return nil, err
			}
		}
	}
	counts := make(map[sim.SavedObjectID]uint32)
	for _, member := range party {
		root := roots[member.ID]
		for at, stack := range cityMemberStacks(member, table) {
			id := stack.ObjectID
			if at < len(root.Pack) && root.Pack[at] != 0 {
				id = root.Pack[at]
			}
			if id == 0 {
				continue
			}
			if prior, ok := counts[id]; ok && prior != stack.Count {
				return nil, fmt.Errorf("city shared Item %d has conflicting current quantities", id)
			}
			counts[id] = stack.Count
		}
	}
	allocate := func(kind uint8) (sim.SavedObjectID, error) {
		id, err := mintCityObjectID(&g.NextID)
		if err != nil {
			return 0, err
		}
		kinds[id] = kind
		switch kind {
		case 2:
			g.Effects = append(g.Effects, id)
		case 3:
			g.Spells = append(g.Spells, id)
		}
		return id, nil
	}
	spell := func(id sim.SavedObjectID, v sim.SourceItemSpell) error {
		if id == 0 || kinds[id] != 3 {
			return fmt.Errorf("city current Spell has no identity")
		}
		if old, ok := p.spells[id]; ok && old != v {
			return fmt.Errorf("city shared Spell %d has conflicting current values", id)
		}
		p.spells[id] = v
		return nil
	}
	item := func(id sim.SavedObjectID, native sim.ItemStack, equipped bool) (sim.SavedObjectID, error) {
		if native.Code == 0 && native.Count == 0 {
			return 0, nil
		}
		if native.Code == 0 || native.Count == 0 {
			return 0, fmt.Errorf("city current Item has incomplete operands")
		}
		if id == 0 {
			id = native.ObjectID
		}
		if id != 0 && kinds[id] != 0 && kinds[id] != 1 {
			return 0, fmt.Errorf("city current Item identity collides with child")
		}
		if id == 0 {
			var err error
			id, err = allocate(1)
			if err != nil {
				return 0, err
			}
		}
		if old, ok := p.items[id]; ok {
			if !sim.StackStateEqual(old, native) {
				return 0, fmt.Errorf("city shared Item %d has conflicting current values", id)
			}
			return id, nil
		}
		value := currentCityItem(native.Instance(), table)
		ownedSpell, err := nativeCityWeaponSpell(value, equipped && native.SourceEquipment.Class == 0, table)
		if err != nil {
			return 0, err
		}
		value.SourceEquipment.Spell = ownedSpell
		index, exists := nodes[id]
		if !exists {
			row := cityItemTopology{ID: id}
			for range value.Effects {
				child, err := allocate(2)
				if err != nil {
					return 0, err
				}
				row.Effects = append(row.Effects, child)
			}
			if value.SourceEquipment.Spell.Present {
				var err error
				row.Spell, err = allocate(3)
				if err != nil {
					return 0, err
				}
			}
			index = len(g.Items)
			nodes[id], kinds[id] = index, 1
			g.Items = append(g.Items, row)
		}
		row := g.Items[index]
		// The party is the current-state producer. A source topology can carry
		// an older child shape after a native item changed; retain identities for
		// surviving positions, mint missing children, and leave dropped nodes
		// unreachable for the transaction cleanup.
		if len(row.Effects) > len(value.Effects) {
			row.Effects = row.Effects[:len(value.Effects)]
		}
		for len(row.Effects) < len(value.Effects) {
			child, err := allocate(2)
			if err != nil {
				return 0, err
			}
			row.Effects = append(row.Effects, child)
		}
		if value.SourceEquipment.Spell.Present && row.Spell == 0 {
			child, err := allocate(3)
			if err != nil {
				return 0, err
			}
			row.Spell = child
		} else if !value.SourceEquipment.Spell.Present {
			row.Spell = 0
		}
		g.Items[index] = row
		for i, child := range row.Effects {
			if child == 0 {
				return 0, fmt.Errorf("city current Effect has no identity")
			}
			v := value.Effects[i]
			if old, ok := p.effects[child]; ok && old != v {
				return 0, fmt.Errorf("city shared Effect %d has conflicting current values", child)
			}
			p.effects[child] = v
		}
		if row.Spell != 0 {
			if err := spell(row.Spell, value.SourceEquipment.Spell); err != nil {
				return 0, err
			}
		}
		p.items[id] = native.Clone()
		return id, nil
	}
	g.Roots, g.Books = nil, nil
	seen := map[string]bool{}
	for _, member := range party {
		if member.ID == "" || seen[member.ID] {
			return nil, fmt.Errorf("city current party lacks unique identity")
		}
		seen[member.ID] = true
		old := roots[member.ID]
		root, book := cityPartyObjectRoots{PartyID: []byte(member.ID)}, cityBookTopology{PartyID: []byte(member.ID)}
		for slot, value := range cityMemberEquipment(member, table) {
			if value.Empty() {
				continue
			}
			var err error
			id := old.Worn[slot]
			if id == 0 {
				id = value.ObjectID
			}
			quantity, present := counts[id]
			if !present {
				quantity = g.wornQuantity(id)
			}
			root.Worn[slot], err = item(old.Worn[slot], sim.StackItem(value, quantity), true)
			if err != nil {
				return nil, err
			}
		}
		for i, value := range cityMemberStacks(member, table) {
			var id sim.SavedObjectID
			if i < len(old.Pack) {
				id = old.Pack[i]
			}
			id, err := item(id, value, false)
			if err != nil {
				return nil, err
			}
			root.Pack = append(root.Pack, id)
		}
		for slot, value := range cityMemberBook(member, table) {
			if !value.Present {
				continue
			}
			id := books[member.ID].Slots[slot]
			if id == 0 {
				var err error
				id, err = allocate(3)
				if err != nil {
					return nil, err
				}
			}
			if err := spell(id, value); err != nil {
				return nil, err
			}
			book.Slots[slot] = id
		}
		g.Roots, g.Books = append(g.Roots, root), append(g.Books, book)
	}
	for i := range g.Items {
		if value, live := p.items[g.Items[i].ID]; live {
			g.Items[i].WornCount = value.Count
		}
	}
	g.trimWornCounts()
	g.sort()
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *cityObjectProjection) emit(b *generatedDocumentBuilder, a *currentActionData) error {
	p.indices, p.chunks = map[sim.SavedObjectID]uint16{}, map[sim.SavedObjectID][]uint16{}
	token := func() sim.SavedObjectToken {
		v := sim.SavedObjectToken{Identity: b.identity(), RuntimeID: b.runtime(), T0E: 0x21}
		v.Position[4], v.Position[5] = 128, 128
		return v
	}
	appendNode := func(id sim.SavedObjectID, record sav.DocumentRecordData, supplement currentActionObject) error {
		index, err := b.append(record)
		if err != nil {
			return err
		}
		p.indices[id] = index
		supplement.ID, supplement.Object = id, index
		a.Objects = append(a.Objects, supplement)
		return nil
	}
	for _, id := range p.graph.Effects {
		if value, ok := p.effects[id]; ok {
			row := sim.SavedEffectObject{ID: id, Token: token(), Value: value}
			if source, present := p.graph.EffectRecords[id]; present {
				row.Token = source.Token
				if row.Token.Identity == 0 {
					row.Token.Identity = b.identity()
				}
				row.E0C = source.E0C
			}
			supplement := currentActionObject{}
			if source, present := p.graph.EffectRecords[id]; present {
				source := source
				supplement.Effect = &source
			}
			if err := appendNode(id, savedCurrentEffectRecord(row), supplement); err != nil {
				return err
			}
		}
	}
	for _, id := range p.graph.Spells {
		if value, ok := p.spells[id]; ok {
			row := sim.SavedSpellObject{ID: id, This: b.identity(), Value: value}
			if source, present := p.graph.SpellRecords[id]; present && source.This != 0 {
				row.This = source.This
			}
			supplement := currentActionObject{}
			if source, present := p.graph.SpellRecords[id]; present {
				source := source
				supplement.Spell = &source
			} else {
				// A Spell created by the current city has no retained source
				// record yet. The SAV record minted above becomes its constructor
				// supplement for the next cold cycle; carry it explicitly instead
				// of asking LOAD to infer private fields from ordinary bytes.
				source := row
				supplement.Spell = &source
			}
			if err := appendNode(id, savedCurrentSpellRecord(row), supplement); err != nil {
				return err
			}
		}
	}
	for _, node := range p.graph.Items {
		native, ok := p.items[node.ID]
		if !ok {
			continue
		}
		value, err := p.currentItemValue(node, native, b.table)
		if err != nil {
			return err
		}
		for remaining := native.Count; remaining != 0; remaining -= min(remaining, uint32(65535)) {
			count := min(remaining, uint32(65535))
			id := node.ID
			if remaining != native.Count {
				var err error
				id, err = mintCityObjectID(&p.graph.NextID)
				if err != nil {
					return err
				}
			}
			var sequence sim.SavedObjectID
			row, _, _, err := sim.ConstructSavedItem(sim.StackItem(value, count), func() (sim.SavedObjectID, sim.SavedObjectToken, error) {
				sequence++
				if sequence == 1 {
					if source, present := p.graph.ItemRecords[node.ID]; present {
						current := source.Token
						if current.Identity == 0 {
							current.Identity = b.identity()
						}
						return sequence, current, nil
					}
				}
				return sequence, token(), nil
			})
			if err != nil {
				return err
			}
			if source, present := p.graph.ItemRecords[node.ID]; present {
				// RuntimeID and other token bytes are loaded document state when
				// the current Item has no owner for them. ConstructSavedItem
				// necessarily writes the current definition row, so retain the
				// source row for an untyped Item whose current definition row is
				// absent. This keeps a zero/opaque source token from becoming a
				// generated identity on a city SAVE/LOAD cycle.
				row.Token.RuntimeID = source.Token.RuntimeID
				if value.SourceEquipment.Class == 0 && value.SourceEquipment.DefinitionRow == 0 {
					row.Token.T0C = source.Token.T0C
				}
			}
			row.ID, row.Effects, row.Spell = id, slices.Clone(node.Effects), node.Spell
			if value.SourceEquipment.Class != 0 && value.NativeRecord == nil {
				if construction, err := nativeCityItemConstructionFor(value.Code, b.table); err == nil {
					row.F45, row.F46, row.F48 = construction.Shape, construction.Material, construction.F48
				}
			}
			if source, present := p.graph.ItemRecords[node.ID]; present {
				row.F45, row.F46, row.F47, row.F48 = source.F45, source.F46, source.F47, source.F48
			}
			record, err := savedCurrentItemRecord(row, p.indices)
			if err != nil {
				return err
			}
			supplement := currentActionObject{}
			if source, present := p.graph.ItemRecords[node.ID]; present && id == node.ID {
				source := source
				source.Value = source.Value.Clone()
				source.Effects = slices.Clone(source.Effects)
				supplement.Item = &source
			}
			if err := appendNode(id, record, supplement); err != nil {
				return err
			}
			p.chunks[node.ID] = append(p.chunks[node.ID], p.indices[id])
			policy := currentItemPolicy(native, p.indices[id])
			if err := captureCurrentItemAbsence(&b.doc, &policy); err != nil {
				return err
			}
			a.Ownership = append(a.Ownership, policy)
		}
	}
	slices.SortFunc(a.Objects, func(x, y currentActionObject) int {
		if x.ID < y.ID {
			return -1
		}
		if x.ID > y.ID {
			return 1
		}
		return 0
	})
	a.Inventory = &currentObjectGraph{Present: true, NextID: p.graph.NextID}
	return nil
}

func (p *cityObjectProjection) holdings(b *generatedDocumentBuilder, index uint16, member mapload.PartyMember, locations ...*[]currentPartyHoldingLocation) error {
	var root cityPartyObjectRoots
	var book cityBookTopology
	for _, row := range p.graph.Roots {
		if string(row.PartyID) == member.ID {
			root = row
		}
	}
	for _, row := range p.graph.Books {
		if string(row.PartyID) == member.ID {
			book = row
		}
	}
	var worn [sim.EquipSlots]uint16
	for slot, id := range root.Worn {
		worn[slot] = p.indices[id]
	}
	var refs []uint16
	_, hasWorn := savedObjectRefs(&b.doc.Objects[index-1], "Worn")
	if !hasWorn {
		for slot := 2; slot < len(worn); slot++ {
			if worn[slot] != 0 {
				if len(locations) == 0 || locations[0] == nil {
					return fmt.Errorf("current Unit equipment lacks location policy")
				}
				*locations[0] = append(*locations[0], currentPartyHoldingLocation{Slot: uint8(slot), Inventory: uint32(len(refs))})
				refs = append(refs, worn[slot])
			}
		}
	}
	packedEquipment := len(refs)
	var weight int32
	for _, id := range root.Pack {
		if id == 0 {
			refs = append(refs, 0)
			continue
		}
		refs = append(refs, p.chunks[id]...)
		v := p.items[id]
		weight += int32(currentCityItem(v.Instance(), b.table).Weight) * int32(v.Count)
	}
	var spells []uint16
	for slot, id := range book.Slots {
		if id != 0 {
			for len(spells) <= slot {
				spells = append(spells, 0)
			}
			spells[slot] = p.indices[id]
		}
	}
	rebindCityItemOwners(&b.doc, index, append(slices.Clone(worn[:]), refs...))
	r := &b.doc.Objects[index-1]
	mustSetRefs(r, "HeldWeapon", []uint16{worn[0]})
	mustSetRefs(r, "HeldShield", []uint16{worn[1]})
	worn[0], worn[1] = 0, 0
	if hasWorn {
		mustSetRefs(r, "Worn", worn[:])
	}
	mustSetRefs(r, "Inventory", refs)
	mustSetCount(r, "Inventory", uint32(len(refs)))
	mustSetValue(r, "Inventory20", uint32(weight))
	if member.Carry != nil && member.Carry.LiveLoad != nil {
		load := member.Carry.LiveLoad
		if !load.Inventory.ContainerPresent && len(refs) != packedEquipment {
			return fmt.Errorf("city actor has items in an absent pack")
		}
		present := uint32(0)
		if load.Inventory.ContainerPresent || packedEquipment != 0 {
			present = 1
		}
		mustSetValue(r, "HasInventory", present)
		mustSetValue(r, "Inventory1C", load.Inventory.InsertIndex)
		mustSetValue(r, "Inventory20", uint32(load.Inventory.Accumulator))
		if present == 0 {
			var err error
			*r, err = savedCurrentContainerRecord(*r, sim.SavedObjectContainer{}, nil)
			if err != nil {
				return err
			}
		}
	}
	if len(spells) != 0 {
		mustSetRefs(r, "Spells", spells)
		mustSetCount(r, "Spells", uint32(len(spells)+1))
	}
	return nil
}

// rebindCityItemOwners writes the holder's own owner Reference, its current
// Player key, into each held Item whose loaded Reference names no object of
// the written document. The city Player is constructed with a fresh key, so a
// source Item's Player key would otherwise be left dangling. A zero Reference
// and one that still resolves keep their loaded value.
func rebindCityItemOwners(doc *sav.DocumentData, holder uint16, items []uint16) {
	owner, err := savedStructureValue(&doc.Objects[holder-1], "Reference")
	if err != nil || owner == 0 {
		return
	}
	present := map[uint32]bool{}
	for i := range doc.Objects {
		for _, name := range []string{"Identity", "This"} {
			if v, err := savedStructureValue(&doc.Objects[i], name); err == nil && v != 0 {
				present[v] = true
			}
		}
	}
	for _, index := range items {
		if index == 0 || int(index) > len(doc.Objects) {
			continue
		}
		item := &doc.Objects[index-1]
		if v, err := savedStructureValue(item, "Reference"); err == nil && v != 0 && !present[v] {
			mustSetValue(item, "Reference", owner)
		}
	}
}

func captureCurrentCityTopology(doc *sav.DocumentData, a *currentActionData, actors []cityActorObjectBinding) (*cityObjectTopology, error) {
	owned := a.Ownership
	var next sim.SavedObjectID
	if a.Inventory != nil {
		if !a.Inventory.Present || a.Inventory.NextID == 0 || len(a.Inventory.Containers) != 0 {
			return nil, fmt.Errorf("city topology has invalid allocator policy")
		}
		next = a.Inventory.NextID
	}
	if len(a.Objects) != 0 {
		owned = nil
		for _, row := range a.Objects {
			if row.ID == 0 || row.Object == 0 || int(row.Object) > len(doc.Objects) {
				return nil, fmt.Errorf("city topology has invalid ordinary binding")
			}
			kind := cityRecordKind(doc.Objects[row.Object-1].Class)
			if kind < 1 || kind > 3 || next != 0 && row.ID >= next {
				return nil, fmt.Errorf("city topology binding has wrong kind/allocator")
			}
			owned = append(owned, currentOwnedObject{ID: row.ID, Object: row.Object, Kind: kind})
		}
	}
	g, _, err := captureCityObjectTopology(doc, actors, owned, next)
	if err != nil {
		return nil, err
	}
	// Current wire values and retained constructor records have different
	// owners. Restore only the detached supplement explicitly carried by the
	// action table; never reconstruct it from a current document record.
	for _, row := range a.Objects {
		private := 0
		if row.Item != nil {
			private++
		}
		if row.Effect != nil {
			private++
		}
		if row.Spell != nil {
			private++
		}
		if private > 1 {
			return nil, fmt.Errorf("city topology object has ambiguous constructor supplement")
		}
		if row.Item != nil {
			if row.Item.ID != row.ID || row.Item.Value.ObjectID != row.ID || cityItemTopologyByID(g.Items, row.ID) == nil {
				return nil, fmt.Errorf("city topology Item supplement has wrong identity")
			}
			if g.ItemRecords == nil {
				g.ItemRecords = make(map[sim.SavedObjectID]sim.SavedItemObject)
			}
			source := *row.Item
			source.Value = source.Value.Clone()
			topology := cityItemTopologyByID(g.Items, row.ID)
			source.Effects, source.Spell = slices.Clone(topology.Effects), topology.Spell
			g.ItemRecords[row.ID] = source
		}
		if row.Effect != nil {
			if row.Effect.ID != row.ID || !slices.Contains(g.Effects, row.ID) {
				return nil, fmt.Errorf("city topology Effect supplement has wrong identity")
			}
			if g.EffectRecords == nil {
				g.EffectRecords = make(map[sim.SavedObjectID]sim.SavedEffectObject)
			}
			g.EffectRecords[row.ID] = *row.Effect
		}
		if row.Spell != nil {
			if row.Spell.ID != row.ID || !slices.Contains(g.Spells, row.ID) {
				return nil, fmt.Errorf("city topology Spell supplement has wrong identity")
			}
			if g.SpellRecords == nil {
				g.SpellRecords = make(map[sim.SavedObjectID]sim.SavedSpellObject)
			}
			g.SpellRecords[row.ID] = *row.Spell
		}
	}
	return g, g.Validate()
}

func captureOriginalCityTopology(doc *sav.DocumentData, party []mapload.PartyMember, chars []sav.Character) (*cityObjectTopology, error) {
	if len(party) != len(chars) {
		return nil, fmt.Errorf("city topology lacks original actor/party bindings")
	}
	byKey := map[uint32]uint16{}
	for i, record := range doc.Objects {
		if record.Class != "Human" && record.Class != "Humanoid" && record.Class != "Unit" {
			continue
		}
		key, err := savedStructureValue(&record, "Identity")
		if err != nil || key == 0 || byKey[key] != 0 {
			return nil, fmt.Errorf("city topology has ambiguous original actor identity")
		}
		byKey[key] = uint16(i + 1)
	}
	var bindings []cityActorObjectBinding
	for i, member := range party {
		bindings = append(bindings, cityActorObjectBinding{Object: byKey[chars[i].Key], PartyID: member.ID})
	}
	g, _, err := captureCityObjectTopologyWithRecords(doc, bindings, nil, 0, true)
	return g, err
}
