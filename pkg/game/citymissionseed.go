package game

import (
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// cityBaseFunc reads the open town's city graph and document namespace for a
// party: the part of a mission's city seeding that needs the whole game state.
type cityBaseFunc func(party []mapload.PartyMember) (*cityObjectTopology, sav.DocumentData, error)

// seedCurrentCityObjects binds the current city's identities to the exact party
// slots StartMission just created. No old city DTO or donor document is needed.
// A closed town seeds from its own graph; an open town asks base for the graph
// and namespace the current game state projects.
func seedCurrentCityObjects(ms *Mission, town *Town, table *mapload.Table, base cityBaseFunc) error {
	if ms == nil || ms.World == nil || ms.World.SavedObjects() != nil || len(ms.Party) != len(ms.Start.IDs) {
		return fmt.Errorf("current city mission has invalid object construction boundary")
	}
	var graph *cityObjectTopology
	if town != nil {
		graph = town.cityObjects
	}
	var namespace sav.DocumentData
	if town.Open() {
		var err error
		graph, namespace, err = base(ms.Party)
		if err != nil {
			return err
		}
	}
	r, bindings, err := currentCityMissionObjects(graph, ms.Party, ms.Start.IDs, ms.World, table, namespace)
	if err != nil {
		return err
	}
	return ms.World.ImportSavedObjects(r, nil, bindings...)
}

func currentCityMissionObjects(graph *cityObjectTopology, party []mapload.PartyMember, ids []sim.EntityID, world *sim.World, table *mapload.Table, namespaces ...sav.DocumentData) (*sim.SavedObjects, []sim.SavedObjectBinding, error) {
	if world == nil || len(party) != len(ids) {
		return nil, nil, fmt.Errorf("current city mission has unmatched party identities")
	}
	var namespace sav.DocumentData
	if len(namespaces) != 0 {
		namespace = namespaces[0]
	}
	city, err := newCityObjectProjection(graph, party, table)
	if err != nil {
		return nil, nil, err
	}
	if err := city.constructSpellRecords(namespace); err != nil {
		return nil, nil, err
	}
	currentParty := mapload.CloneParty(party)
	for i := range currentParty {
		worn, _ := world.EquippedItems(ids[i])
		equipment := cityMemberEquipment(currentParty[i], table)
		for slot := range equipment {
			if equipment[slot].Code != 0 && equipment[slot].Code == worn[slot].Code {
				equipment[slot].Weight, equipment[slot].WeightPresent = worn[slot].Weight, worn[slot].WeightPresent
				equipment[slot].SourceEquipment = worn[slot].SourceEquipment
			}
			if equipment[slot].SourceEquipment.Class == sim.SourceWeapon {
				equipment[slot].SourceEquipment.Spell = worn[slot].SourceEquipment.Spell
			}
		}
		if currentParty[i].Carry != nil {
			currentParty[i].Carry.EquippedItems = equipment
		} else {
			currentParty[i].WornItems = equipment
		}
	}
	p, err := newCityObjectProjection(city.graph, currentParty, table)
	if err != nil {
		return nil, nil, err
	}
	if err := p.followQuestDocumentCollection(party, ids, world); err != nil {
		return nil, nil, err
	}
	r := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: p.graph.NextID}
	origin := sim.SavedObjectOrigin{Kind: sim.SavedObjectGenerated}
	// Records retained from an earlier seed hold identities of this form.
	retained := map[uint32]bool{}
	for _, row := range p.graph.ItemRecords {
		retained[row.Token.Identity] = true
	}
	for _, row := range p.graph.EffectRecords {
		retained[row.Token.Identity] = true
	}
	for _, row := range p.graph.SpellRecords {
		retained[row.This] = true
	}
	var serial uint32
	token := func() sim.SavedObjectToken {
		serial++
		for retained[0x62000000+serial*16] {
			serial++
		}
		v := sim.SavedObjectToken{Identity: 0x62000000 + serial*16, RuntimeID: serial, T0E: 0x21}
		v.Position[4], v.Position[5] = 128, 128
		return v
	}
	for _, id := range p.graph.Effects {
		if value, live := p.effects[id]; live {
			row := sim.SavedEffectObject{ID: id, Origin: origin, Token: token(), Value: value}
			if source, present := p.graph.EffectRecords[id]; present {
				row.Token = source.Token
				if row.Token.Identity == 0 {
					row.Token.Identity = token().Identity
				}
				if row.Token.RuntimeID == 0 {
					row.Token.RuntimeID = token().RuntimeID
				}
				row.E0C = source.E0C
			}
			r.Effects = append(r.Effects, row)
		}
	}
	spells := map[sim.SavedObjectID]int{}
	for _, id := range p.graph.Spells {
		if value, live := p.spells[id]; live {
			spells[id] = len(r.Spells)
			row := sim.SavedSpellObject{ID: id, Origin: origin, This: token().Identity, Value: value}
			if source, present := p.graph.SpellRecords[id]; present && source.This != 0 {
				row.This = source.This
			}
			r.Spells = append(r.Spells, row)
		}
	}
	items := map[sim.SavedObjectID]sim.SavedItemObject{}
	for _, node := range p.graph.Items {
		value, live := p.items[node.ID]
		if !live {
			continue
		}
		value = value.Clone()
		if node.Spell != 0 {
			value = sim.StackItem(mapload.SourceEquippedItem(value.Instance(), table), value.Count)
		}
		value.ObjectID = node.ID
		row := sim.SavedItemObject{ID: node.ID, Origin: origin, Value: value, Token: token(), Effects: slices.Clone(node.Effects), Spell: node.Spell}
		if source, present := p.graph.ItemRecords[node.ID]; present {
			row.Token = source.Token
			row.F45, row.F46, row.F47, row.F48 = source.F45, source.F46, source.F47, source.F48
			if row.Token.Identity == 0 {
				row.Token.Identity = token().Identity
			}
		}
		if history := value.NativeRecord; history != nil {
			row.Token = history.Token
			row.F45, row.F46, row.F47, row.F48 = history.F45, history.F46, history.F47, history.F48
			row.Value.NativeRecord = nil
		}
		row.Token.T1C = uint32(value.Price)
		_, hasSource := p.graph.ItemRecords[node.ID]
		if value.SourceEquipment.Class != 0 || value.SourceEquipment.DefinitionRow != 0 || !hasSource && value.NativeRecord == nil {
			row.Token.T0C = value.SourceEquipment.DefinitionRow
		}
		if value.Count > 65535 {
			row.Coverage.Unknown |= sim.SavedUnknownCountWidth
		}
		items[row.ID] = row
		r.Items = append(r.Items, row)
	}
	entities := map[sim.EntityID]sim.Entity{}
	for _, entity := range world.Entities() {
		entities[entity.ID] = entity
	}
	byParty, used := map[string]sim.EntityID{}, map[sim.EntityID]bool{}
	for i, member := range party {
		if _, present := entities[ids[i]]; !present || used[ids[i]] {
			return nil, nil, fmt.Errorf("current city party has missing or repeated runtime actor")
		}
		byParty[member.ID], used[ids[i]] = ids[i], true
	}
	var bindings []sim.SavedObjectBinding
	for _, root := range p.graph.Roots {
		entity := byParty[string(root.PartyID)]
		owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: entity}
		container := sim.SavedObjectContainer{Owner: owner, Present: len(root.Pack) != 0, Items: slices.Clone(root.Pack)}
		if load := entities[entity].ActorLoad; load.Present {
			container.Present, container.InsertIndex, container.Accumulator = load.ContainerPresent, load.InsertIndex, load.Accumulator
		}
		for index, id := range root.Pack {
			if id == 0 {
				container.Coverage.Unknown |= sim.SavedUnknownContainerLoad | sim.SavedUnknownMergePolicy
				continue
			}
			bindings = append(bindings, sim.SavedObjectBinding{ID: id, Owner: owner, Index: uint32(index), Value: items[id].Value})
		}
		r.Containers = append(r.Containers, container)
		for slot, id := range root.Worn {
			if id == 0 {
				continue
			}
			owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: entity, Slot: uint32(slot + 1)}
			if err := r.AddItemRoot(id, owner); err != nil {
				return nil, nil, err
			}
			bindings = append(bindings, sim.SavedObjectBinding{ID: id, Owner: owner, Value: items[id].Value})
		}
	}
	for _, book := range p.graph.Books {
		entity := byParty[string(book.PartyID)]
		root := sim.SavedBookRoot{Entity: entity}
		// The mission constructor can legitimately refresh a member's book.
		// A changed slot gets its own current node; another owner is untouched.
		for slot, value := range sim.BookSlotValues(world.Rules(), entities[entity], world.Spells()) {
			if !value.Present {
				continue
			}
			id := book.Slots[slot]
			index, present := spells[id]
			if !present || r.Spells[index].Value != value {
				id, err = mintCityObjectID(&r.NextID)
				if err != nil {
					return nil, nil, err
				}
				spells[id] = len(r.Spells)
				row := sim.SavedSpellObject{ID: id, Origin: origin, This: token().Identity, Value: value}
				if source, present := p.graph.SpellRecords[id]; present && source.This != 0 {
					row.This = source.This
				}
				r.Spells = append(r.Spells, row)
			}
			root.Slots[slot] = id
		}
		r.BookRoots = append(r.BookRoots, root)
	}
	slices.SortFunc(r.Containers, func(a, b sim.SavedObjectContainer) int { return cmp.Compare(a.Owner.Entity, b.Owner.Entity) })
	slices.SortFunc(r.BookRoots, func(a, b sim.SavedBookRoot) int { return cmp.Compare(a.Entity, b.Entity) })
	r.RefreshChildLiveness()
	if err := r.ValidateNoInFlight(); err != nil {
		return nil, nil, err
	}
	return r, bindings, nil
}

// followQuestDocumentCollection moves the projected pack roots to where mission
// start put the quest documents. The projection is built from the party as it
// entered; mapload then moved each other member's first quest-document stack
// onto the starting hero, merging it into a held stack or inserting it. The
// world's packs are the authority: a companion's pack must be its entry pack
// less that stack, and the hero's pack its entry pack with document stacks
// grown or inserted. A merged stack's node is no longer held.
func (p *cityObjectProjection) followQuestDocumentCollection(party []mapload.PartyMember, ids []sim.EntityID, world *sim.World) error {
	primary := mapload.QuestDocumentHolder(party)
	if primary < 0 || primary >= len(ids) || primary >= len(party) {
		return nil
	}
	code := uint16(data.QuestDocumentCode)
	roots := map[string]int{}
	for i, row := range p.graph.Roots {
		roots[string(row.PartyID)] = i
	}
	value := func(id sim.SavedObjectID) sim.ItemStack { return p.items[id] }
	same := func(a, b sim.ItemStack) bool { return a.Code == b.Code && a.Count == b.Count }
	var moved []sim.SavedObjectID
	for i, member := range party {
		r, ok := roots[member.ID]
		if i == primary || !ok || i >= len(ids) {
			continue
		}
		pack := p.graph.Roots[r].Pack
		at := slices.IndexFunc(pack, func(id sim.SavedObjectID) bool { return id != 0 && value(id).Code == code })
		if at < 0 {
			continue
		}
		moved = append(moved, pack[at])
		pack = slices.Delete(slices.Clone(pack), at, at+1)
		stacks, _ := world.CarriedStacks(ids[i])
		if len(stacks) != len(pack) {
			return fmt.Errorf("current city mission quest document collection differs from the world")
		}
		for k, id := range pack {
			if !same(value(id), stacks[k]) {
				return fmt.Errorf("current city mission quest document collection differs from the world")
			}
		}
		p.graph.Roots[r].Pack = pack
	}
	if len(moved) == 0 {
		return nil
	}
	r, ok := roots[party[primary].ID]
	if !ok {
		return fmt.Errorf("current city mission quest document holder has no pack root")
	}
	entry := p.graph.Roots[r].Pack
	stacks, _ := world.CarriedStacks(ids[primary])
	var pack []sim.SavedObjectID
	j := 0
	for _, stack := range stacks {
		switch {
		case j < len(entry) && same(value(entry[j]), stack):
			pack = append(pack, entry[j])
			j++
		case j < len(entry) && stack.Code == code && value(entry[j]).Code == code && stack.Count > value(entry[j]).Count:
			grown := value(entry[j])
			for grown.Count < stack.Count && len(moved) != 0 {
				grown.Count += value(moved[0]).Count
				if err := p.retireCollectedDocument(moved[0]); err != nil {
					return err
				}
				moved = moved[1:]
			}
			if grown.Count != stack.Count {
				return fmt.Errorf("current city mission quest document collection differs from the world")
			}
			p.items[entry[j]] = grown
			pack = append(pack, entry[j])
			j++
		case stack.Code == code && len(moved) != 0 && same(value(moved[0]), stack):
			pack = append(pack, moved[0])
			moved = moved[1:]
		default:
			return fmt.Errorf("current city mission quest document collection differs from the world")
		}
	}
	if j != len(entry) || len(moved) != 0 {
		return fmt.Errorf("current city mission quest document collection differs from the world")
	}
	p.graph.Roots[r].Pack = pack
	for i := range p.graph.Items {
		if v, live := p.items[p.graph.Items[i].ID]; live {
			p.graph.Items[i].WornCount = v.Count
		}
	}
	return nil
}

// retireCollectedDocument drops a quest-document node merged into another
// stack. A node with children would leave them unowned, so it is refused.
func (p *cityObjectProjection) retireCollectedDocument(id sim.SavedObjectID) error {
	if node := cityItemTopologyByID(p.graph.Items, id); node != nil && (len(node.Effects) != 0 || node.Spell != 0) {
		return fmt.Errorf("current city mission quest document %d has child objects", id)
	}
	delete(p.items, id)
	return nil
}
