package game

import (
	"cmp"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Coverage records original-constructor uncertainty, not absent current values.
// This export view retains the native registry and emits every current operand.
// Bound stacks retain one identity; their native count width is supplemented.
// Unbound wide stacks occupy adjacent word-count records.
func projectCurrentItemGraph(state *SnapshotSAVDocument, world *sim.World, tables ...*mapload.Table) error {
	return projectCurrentItemGraphMode(state, world, false, tables...)
}

// projectCurrentItemGraphForSave resolves source-only ground roots while
// constructing the explicit current SAV document. Ordinary snapshots retain
// ambiguous source roots so a later producer cannot appear to have selected
// one original owner.
func projectCurrentItemGraphForSave(state *SnapshotSAVDocument, world *sim.World, tables ...*mapload.Table) error {
	return projectCurrentItemGraphMode(state, world, true, tables...)
}

func projectCurrentItemGraphMode(state *SnapshotSAVDocument, world *sim.World, forSave bool, tables ...*mapload.Table) error {
	if err := rejectRetiringSackAliases(state, world); err != nil {
		return err
	}
	if state.Objects == nil {
		state.Objects = &SnapshotSAVObjectBindings{Version: 2}
	}
	if a, err := readCurrentActions(state.Document); err != nil {
		return err
	} else if a != nil {
		a.Ownership = nil
		b, err := json.Marshal(a)
		if err != nil {
			return err
		}
		if err := sav.SetNativeActions(&state.Document.State, b); err != nil {
			return err
		}
	}
	r := world.SavedObjects()
	if r == nil {
		r = &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: 1}
	}
	if err := r.ValidateNoInFlight(); err != nil {
		return err
	}
	nativeContainers := r.Containers
	for _, a := range state.Actors {
		if a.Retired {
			continue
		}
		owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: a.EntityID}
		found := false
		for _, c := range r.Containers {
			if c.Owner == owner {
				found = true
				break
			}
		}
		if !found {
			values, _ := world.CarriedStacks(a.EntityID)
			c := sim.SavedObjectContainer{Owner: owner, Present: true}
			for _, body := range world.OriginalDeadActors() {
				if body.ID == a.EntityID {
					c.Present = body.Source.ContainerPresent
					c.InsertIndex, c.Accumulator = body.Source.ContainerTail[0], int32(body.Source.ContainerTail[1])
				}
			}
			for _, v := range values {
				c.Items = append(c.Items, v.ObjectID)
				c.Accumulator += int32(v.Weight) * int32(v.Count)
			}
			if e, ok := world.Entity(a.EntityID); ok && e.ActorLoad.Present {
				c.Present = e.ActorLoad.ContainerPresent
				c.InsertIndex, c.Accumulator = e.ActorLoad.InsertIndex, e.ActorLoad.Accumulator
			}
			r.Containers = append(r.Containers, c)
		}
	}
	reservation := *state.Document
	reservation.Objects = slices.Clone(reservation.Objects)
	for _, item := range r.Items {
		reservation.Objects = append(reservation.Objects, savedObjectRecord("Item", &item.Token))
	}
	for _, effect := range r.Effects {
		reservation.Objects = append(reservation.Objects, savedCurrentEffectRecord(effect))
	}
	for _, spell := range r.Spells {
		reservation.Objects = append(reservation.Objects, savedCurrentSpellRecord(spell))
	}
	for _, sack := range r.Sacks {
		reservation.Objects = append(reservation.Objects, savedObjectRecord("Sack", &sack.Token))
	}
	keys, err := sav.ReserveDocumentKeys(reservation, 65536)
	if err != nil {
		return err
	}
	b := generatedDocumentBuilder{doc: *state.Document, reservedKeys: keys, runtimeIDs: savedRuntimeIDs(reservation.Objects)}
	if err := b.reserveCurrentWorld(world); err != nil {
		return err
	}
	if len(tables) != 0 {
		b.table = tables[0]
	}
	indices := savedObjectIndices(state.Objects)
	needed := make(map[sim.SavedObjectID]bool)
	for _, row := range r.Items {
		if row.Retired || !currentItemHasOrdinaryRoot(r, row.ID) {
			continue
		}
		for _, id := range row.Effects {
			needed[id] = true
		}
		needed[row.Spell] = true
	}
	for _, book := range r.BookRoots {
		for _, id := range book.Slots {
			needed[id] = true
		}
	}
	put := func(id sim.SavedObjectID, record sav.DocumentRecordData) error {
		index := indices[id]
		if index == 0 {
			var err error
			index, err = b.append(record)
			if err != nil {
				return err
			}
			indices[id] = index
		} else {
			b.doc.Objects[index-1] = record
		}
		return nil
	}
	for _, row := range r.Effects {
		if row.Retired || !needed[row.ID] && indices[row.ID] == 0 {
			continue
		}
		if err := put(row.ID, savedCurrentEffectRecord(row)); err != nil {
			return err
		}
	}
	for _, row := range r.Spells {
		if row.Retired || !needed[row.ID] && indices[row.ID] == 0 {
			continue
		}
		if err := put(row.ID, savedCurrentSpellRecord(row)); err != nil {
			return err
		}
	}
	chunks := func(record sav.DocumentRecordData, count uint32, first uint16) ([]uint16, error) {
		var refs []uint16
		for count != 0 {
			part := min(count, uint32(65535))
			savedObjectSetValue(&record, "F42", part)
			if len(refs) == 0 && first != 0 {
				b.doc.Objects[first-1] = record
				refs = append(refs, first)
			} else {
				copy := record
				copy.Values = append([]sav.DocumentValueData(nil), record.Values...)
				if len(refs) != 0 {
					savedObjectSetValue(&copy, "Identity", b.identity())
					savedObjectSetValue(&copy, "RuntimeID", b.runtime())
				}
				index, err := b.append(copy)
				if err != nil {
					return nil, err
				}
				refs = append(refs, index)
			}
			count -= part
			record.Values = append([]sav.DocumentValueData(nil), record.Values...)
		}
		return refs, nil
	}
	itemRefs := make(map[sim.SavedObjectID][]uint16)
	weights := make(map[uint16]sim.ItemWeight)
	for _, weight := range world.ItemWeights() {
		weights[weight.Code] = weight
	}
	held := make(map[sim.SavedObjectID]bool)
	for _, root := range r.ItemRoots {
		if root.Owner.Kind == sim.SavedOwnerActorWorn {
			held[root.ID] = true
		}
	}
	for _, row := range r.Items {
		if row.Retired || !currentItemHasOrdinaryRoot(r, row.ID) {
			continue
		}
		count := row.Value.Count
		value := currentItemRecordValue(row.Value.Instance(), held[row.ID], weights, b.table)
		row.Value = sim.StackItem(value, min(count, uint32(65535)))
		record, err := savedCurrentItemRecord(row, indices)
		if err != nil {
			return err
		}
		itemRefs[row.ID], err = chunks(record, min(count, uint32(65535)), indices[row.ID])
		if err != nil {
			return err
		}
		indices[row.ID] = itemRefs[row.ID][0]
	}
	retiredSacks, err := projectCurrentUnboundSackRoots(&b, state, world, forSave)
	if err != nil {
		return err
	}
	boundRetired, err := projectCurrentSackRoots(&b, state.Objects, r, indices, put)
	if err != nil {
		return err
	}
	for index := range boundRetired {
		retiredSacks[index] = true
	}
	unbound := func(value sim.ItemStack, owner uint32, held bool) ([]uint16, error) {
		if sim.StackStateEqual(value, sim.ItemStack{}) {
			return []uint16{0}, nil
		}
		if value.ObjectID != 0 {
			refs := itemRefs[value.ObjectID]
			if len(refs) == 0 {
				return nil, fmt.Errorf("current owner names missing Item %d", value.ObjectID)
			}
			return refs, nil
		}
		item := currentItemRecordValue(value.Instance(), held, weights, b.table)
		index, err := b.item(item, min(value.Count, uint32(65535)), owner)
		if err != nil {
			return nil, err
		}
		return chunks(b.doc.Objects[index-1], value.Count, index)
	}
	knownItems := make(map[uint16]sim.SavedObjectID)
	for _, item := range r.Items {
		if object := indices[item.ID]; object != 0 {
			knownItems[object] = item.ID
		}
	}
	for _, a := range state.Actors {
		if a.Retired {
			clearCurrentRetiredActorItems(&b.doc.Objects[a.ObjectIndex-1], a.EntityID, knownItems, r)
			continue
		}
		items, ok := world.EquippedItems(a.EntityID)
		if !ok {
			return fmt.Errorf("current item owner %d is absent", a.EntityID)
		}
		owner, _ := savedStructureValue(&b.doc.Objects[a.ObjectIndex-1], "Reference")
		var worn [sim.EquipSlots]uint16
		for slot, value := range items {
			if value.Empty() {
				continue
			}
			refs, err := unbound(sim.StackItem(value, 1), owner, true)
			if err != nil {
				return err
			}
			worn[slot] = refs[0]
		}
		actor := &b.doc.Objects[a.ObjectIndex-1]
		savedObjectSetRefs(actor, "HeldWeapon", []uint16{worn[0]}, false)
		savedObjectSetRefs(actor, "HeldShield", []uint16{worn[1]}, false)
		if actor.Class != "Unit" {
			worn[0], worn[1] = 0, 0
			savedObjectSetRefs(actor, "Worn", worn[:], false)
		}
	}
	for _, c := range r.Containers {
		var values []sim.ItemStack
		var index uint16
		field, cursor := "Inventory", "Inventory1C"
		switch c.Owner.Kind {
		case sim.SavedOwnerActorPack:
			for _, a := range state.Actors {
				if !a.Retired && a.EntityID == c.Owner.Entity {
					index = a.ObjectIndex
				}
			}
			values, _ = world.CarriedStacks(c.Owner.Entity)
		case sim.SavedOwnerSack:
			index, field, cursor = indices[c.Owner.Object], "Contents", "Contents1C"
			for _, sack := range world.Sacks() {
				if sack.ObjectID != c.Owner.Object {
					continue
				}
				items := currentSackItemInstances(sack)
				at := 0
				for _, id := range c.Items {
					count := uint32(1)
					if id != 0 {
						row, ok := r.Item(id)
						if !ok {
							return fmt.Errorf("current Sack names missing Item %d", id)
						}
						count = row.Value.Count
					}
					if uint64(at)+uint64(count) > uint64(len(items)) {
						return fmt.Errorf("current Sack count exceeds holdings")
					}
					values = append(values, sim.StackItem(items[at], count))
					at += int(count)
				}
				if at != len(items) {
					return fmt.Errorf("current Sack omits holdings")
				}
			}
		}
		if index == 0 || len(values) != len(c.Items) {
			return fmt.Errorf("current container lacks its exact owner or slots")
		}
		if c.Owner.Kind == sim.SavedOwnerActorPack {
			if !c.Present {
				record, err := savedCurrentContainerRecord(b.doc.Objects[index-1], c, nil)
				if err != nil {
					return err
				}
				b.doc.Objects[index-1] = record
				continue
			}
			savedObjectSetValue(&b.doc.Objects[index-1], "HasInventory", 1)
			savedObjectSetValue(&b.doc.Objects[index-1], "Inventory20", uint32(c.Accumulator))
		}
		owner, _ := savedStructureValue(&b.doc.Objects[index-1], "Reference")
		var refs []uint16
		insert := uint64(c.InsertIndex)
		for i, value := range values {
			if value.ObjectID != c.Items[i] {
				return fmt.Errorf("current container slot identity differs")
			}
			parts, err := unbound(value, owner, false)
			if err != nil {
				return err
			}
			refs = append(refs, parts...)
			if uint32(i) < c.InsertIndex {
				insert += uint64(len(parts) - 1)
			}
		}
		if c.Present {
			savedObjectSetRefs(&b.doc.Objects[index-1], field, refs, true)
			savedObjectSetValue(&b.doc.Objects[index-1], cursor, uint32(min(insert, uint64(^uint32(0)))))
		}
	}
	for _, row := range r.Items {
		if refs := itemRefs[row.ID]; len(refs) > 0 {
			indices[row.ID] = refs[0]
		}
	}
	bind := func(rows *[]SnapshotSAVObjectBinding, id sim.SavedObjectID, retired bool) {
		index := indices[id]
		if retired {
			index = 0
		}
		binding := SnapshotSAVObjectBinding{ID: id, ObjectIndex: index}
		for i := range *rows {
			if (*rows)[i].ID == id {
				(*rows)[i] = binding
				return
			}
		}
		*rows = append(*rows, binding)
	}
	for _, row := range r.Items {
		bind(&state.Objects.Items, row.ID, row.Retired)
	}
	for _, row := range r.Effects {
		bind(&state.Objects.Effects, row.ID, row.Retired)
	}
	for _, row := range r.Spells {
		bind(&state.Objects.Spells, row.ID, row.Retired)
	}
	for _, rows := range [][]SnapshotSAVObjectBinding{state.Objects.Items, state.Objects.Effects, state.Objects.Spells} {
		slices.SortFunc(rows, func(a, b SnapshotSAVObjectBinding) int { return cmp.Compare(a.ID, b.ID) })
	}
	state.Objects.Containers = nil
	for _, container := range nativeContainers {
		state.Objects.Containers = append(state.Objects.Containers, SnapshotSAVContainerBinding{Owner: container.Owner})
	}
	// Book roots share the current Spell table with weapons. Connect their
	// current slots before the one graph retirement can remove old children.
	if err := projectCurrentBookGraph(&b, state, world, forSave); err != nil {
		return err
	}
	retired, err := currentUnreachableItems(b.doc, retiredSacks)
	if err != nil {
		return err
	}
	doc, permutation, err := sav.RetireDocumentData(b.doc, retired)
	if err != nil {
		return err
	}
	state.Document = &doc
	return remapSavedSackDocument(state, permutation)
}

// A source-only Sack may not be collected merely because its root is stale.
// If it still points at a bound Sack that native state retired, the source
// edge is unresolved and removing both nodes would conceal its aliasing.
func rejectRetiringSackAliases(state *SnapshotSAVDocument, world *sim.World) error {
	if state == nil || state.Document == nil || state.Objects == nil || world == nil {
		return nil
	}
	registry := world.SavedObjects()
	if registry == nil {
		return nil
	}
	retiring := make(map[uint16]bool)
	boundRetiring := make(map[uint16]bool)
	for _, binding := range state.Objects.Sacks {
		if binding.ObjectIndex == 0 {
			continue
		}
		for _, row := range registry.Sacks {
			if row.ID == binding.ID && row.Retired {
				retiring[binding.ObjectIndex] = true
				boundRetiring[binding.ObjectIndex] = true
			}
		}
	}
	if len(retiring) == 0 {
		return nil
	}
	var walk func(uint16, *sav.DocumentRecordData) error
	walk = func(source uint16, record *sav.DocumentRecordData) error {
		for _, refs := range record.RefSlots {
			for _, target := range refs.Objects {
				if retiring[target] && source != target && !boundRetiring[source] {
					return fmt.Errorf("current graph retains retired Sack node %d through source node %d", target, source)
				}
			}
		}
		for i := range record.Inline {
			if err := walk(source, &record.Inline[i].Record); err != nil {
				return err
			}
		}
		for i := range record.Groups {
			if err := walk(source, &record.Groups[i]); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range state.Document.Objects {
		if err := walk(uint16(i+1), &state.Document.Objects[i]); err != nil {
			return err
		}
	}
	return nil
}

// A merged unbound Sack replaces all source roots at its cell with one
// current root. Retaining a stale source row would resurrect old contents.
func projectCurrentUnboundSackRoots(b *generatedDocumentBuilder, state *SnapshotSAVDocument, world *sim.World, reconcileAmbiguousSacks bool) (map[uint16]bool, error) {
	retired := make(map[uint16]bool)
	if b == nil || state == nil || state.Document == nil || world == nil {
		return retired, nil
	}
	bound := make(map[uint16]bool)
	if state.Objects != nil {
		for _, row := range state.Objects.Sacks {
			if row.ObjectIndex != 0 {
				bound[row.ObjectIndex] = true
			}
		}
	}
	ambiguousCells := make(map[uint16]bool)
	for _, index := range b.doc.World.Sacks {
		if bound[index] || index == 0 || int(index) > len(b.doc.Objects) {
			continue
		}
		token, _, _, err := savedSackRecord(&b.doc.Objects[index-1])
		if err != nil {
			return nil, err
		}
		cell := uint16(token.Position[2]) | uint16(token.Position[3])<<8
		ambiguousCells[cell] = ambiguousCells[cell] || countUnboundSackCell(b.doc, bound, cell) > 1
	}
	// A current interaction may have consumed an unbound source Sack before
	// the mission's retained document is snapshotted. Retire that stale root
	// from the wire graph as well; keeping it would resurrect the picked-up
	// ground item on cold LOAD.
	currentCells := make(map[uint16]bool)
	for _, sack := range world.Sacks() {
		if sack.ObjectID == 0 {
			currentCells[uint16(sack.X)|uint16(sack.Y)<<8] = true
		}
	}
	for _, index := range b.doc.World.Sacks {
		if bound[index] || index == 0 || int(index) > len(b.doc.Objects) {
			continue
		}
		token, _, _, err := savedSackRecord(&b.doc.Objects[index-1])
		if err != nil {
			return nil, err
		}
		cell := uint16(token.Position[2]) | uint16(token.Position[3])<<8
		if !currentCells[cell] {
			retired[index] = true
		}
	}
	var generated []uint16
	for _, sack := range world.Sacks() {
		if sack.ObjectID != 0 {
			continue
		}
		cell := uint16(sack.X) | uint16(sack.Y)<<8
		if ambiguousCells[cell] && !reconcileAmbiguousSacks {
			continue
		}
		var stale []uint16
		var sourceKey uint32
		for _, index := range b.doc.World.Sacks {
			if index == 0 || int(index) > len(b.doc.Objects) {
				return nil, fmt.Errorf("current unbound Sack root %d is outside the document", index)
			}
			if bound[index] {
				continue
			}
			token, _, _, err := savedSackRecord(&b.doc.Objects[index-1])
			if err != nil {
				return nil, err
			}
			if uint16(token.Position[2])|uint16(token.Position[3])<<8 == cell {
				stale = append(stale, index)
				retired[index] = true
				if sourceKey == 0 {
					sourceKey = token.Identity
				}
			}
		}
		key, err := currentSackWireKey(world, cell)
		if err != nil {
			return nil, err
		}
		if len(stale) == 1 && (key == 0 || key == sourceKey) && currentUnboundSackMatchesSource(&b.doc, stale[0], sack) {
			delete(retired, stale[0])
			continue
		}
		if key == 0 {
			key = sourceKey
		}
		if key == 0 {
			key = b.identity()
		}
		record := mustNewRecord("Sack")
		mustSetToken(&record, nativeCityToken(key, 0, 0, 0))
		// SAV-1093: every verified original Sack token carries publication
		// mask 2 at Token+0x18.
		mustSetValue(&record, "T18", 2)
		mustSetRaw(&record, "Block12", constructedPositionBlock(sack.X, sack.Y, b.doc.World.TerrainIdentity))
		mustSetValue(&record, "S3C", sack.Gold)
		items := currentSackItemInstances(sack)
		refs := make([]uint16, 0, len(items))
		resolved := make([]sim.ItemInstance, 0, len(items))
		var weight int32
		for _, item := range items {
			value := constructedGeneratedItem(item, b.table)
			index, err := b.item(value, 1, key)
			if err != nil {
				return nil, err
			}
			refs = append(refs, index)
			weight += int32(value.Weight)
			resolved = append(resolved, value)
		}
		mustSetRefs(&record, "Contents", refs)
		mustSetCount(&record, "Contents", uint32(len(refs)))
		mustSetValue(&record, "Contents20", uint32(weight))
		// ITEM-SACK-010: the same gold-plus-items rule the sim's own ground
		// writer recomputes (sim.SackTokenValue), off each item's own resolved
		// price rather than the pre-resolution Price this record's own Contents
		// refs were minted from.
		mustSetValue(&record, "T1C", sim.SackTokenValue(sack.Gold, resolved))
		index, err := b.append(record)
		if err != nil {
			return nil, err
		}
		generated = append(generated, index)
		for i := range b.doc.World.Cells {
			if b.doc.World.Cells[i].Cell == cell {
				b.doc.World.Cells[i].Sack = key
			}
		}
	}
	if len(retired) != 0 {
		b.doc.World.Sacks = slices.DeleteFunc(b.doc.World.Sacks, func(index uint16) bool { return retired[index] })
		if state.Objects != nil {
			state.Objects.Unavailable = slices.DeleteFunc(state.Objects.Unavailable, func(row SnapshotSAVObjectCoverage) bool { return retired[row.ObjectIndex] })
		}
	}
	b.doc.World.Sacks = append(b.doc.World.Sacks, generated...)
	if state.Objects != nil {
		for _, index := range generated {
			state.Objects.Unavailable = append(state.Objects.Unavailable, SnapshotSAVObjectCoverage{ObjectIndex: index, Reason: savedSackNativeUnavailable})
		}
	}
	return retired, nil
}

func currentUnboundSackMatchesSource(doc *sav.DocumentData, index uint16, sack sim.Sack) bool {
	if doc == nil || index == 0 || int(index) > len(doc.Objects) {
		return false
	}
	record := &doc.Objects[index-1]
	token, gold, _, err := savedSackRecord(record)
	if err != nil || gold != sack.Gold || int32(token.Position[2]) != sack.X || int32(token.Position[3]) != sack.Y {
		return false
	}
	refs, ok := savedObjectRefs(record, "Contents")
	if !ok {
		return false
	}
	current := currentSackItemInstances(sack)
	unit := 0
	for _, ref := range refs {
		row, err := savedItemRecord(doc, ref)
		if err != nil || row.Value.Count > uint32(len(current)-unit) {
			return false
		}
		for range row.Value.Count {
			value := row.Value.Clone()
			value.SourceEquipment.Definition = current[unit].SourceEquipment.Definition
			if !sim.StackStateEqual(value, sim.StackItem(current[unit], row.Value.Count)) {
				return false
			}
			unit++
		}
	}
	return unit == len(current)
}

func countUnboundSackCell(doc sav.DocumentData, bound map[uint16]bool, cell uint16) int {
	count := 0
	for _, index := range doc.World.Sacks {
		if bound[index] || index == 0 || int(index) > len(doc.Objects) {
			continue
		}
		token, _, _, err := savedSackRecord(&doc.Objects[index-1])
		if err == nil && uint16(token.Position[2])|uint16(token.Position[3])<<8 == cell {
			count++
		}
	}
	return count
}

// The current cell and motion projections carry the same native Sack key when
// either one was imported. Preserve that key for a generated unbound root so
// a cold LOAD retains the current world's carrier state byte-for-byte.
func currentSackWireKey(world *sim.World, cell uint16) (uint32, error) {
	var key uint32
	accept := func(candidate uint32) error {
		if candidate == 0 {
			return nil
		}
		if key != 0 && key != candidate {
			return fmt.Errorf("current Sack cell %04x has conflicting native keys", cell)
		}
		key = candidate
		return nil
	}
	for _, row := range world.SavedCellRecords() {
		if row.Cell == cell {
			if err := accept(row.Sack); err != nil {
				return 0, err
			}
		}
	}
	_, cells, _, present := world.SavedActorMotions()
	if present {
		for _, row := range cells {
			if row.Cell == cell {
				if err := accept(binary.LittleEndian.Uint32(row.Payload[16:])); err != nil {
					return 0, err
				}
			}
		}
	}
	return key, nil
}

// Remove only known Item edges whose current native owner has gone away.
// Other source references and retained actor operands keep their own authority.
func clearCurrentRetiredActorItems(record *sav.DocumentRecordData, actor sim.EntityID, items map[uint16]sim.SavedObjectID, registry *sim.SavedObjects) {
	for _, field := range []string{"Inventory", "HeldWeapon", "HeldShield", "Worn"} {
		refs, present := savedObjectRefs(record, field)
		if !present {
			continue
		}
		var kept []uint16
		changed := false
		for slot, ref := range refs {
			owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: actor, Slot: uint32(slot + 1)}
			if field == "Inventory" {
				owner.Kind, owner.Slot = sim.SavedOwnerActorPack, 0
			} else if field == "HeldShield" {
				owner.Slot = 2
			}
			id, known := items[ref]
			retained := false
			if known {
				for _, location := range registry.Locations(id) {
					retained = retained || location.Owner == owner
				}
			}
			if known && !retained {
				changed = true
				if field != "Inventory" {
					kept = append(kept, 0)
				}
			} else {
				kept = append(kept, ref)
			}
		}
		if changed {
			savedObjectSetRefs(record, field, kept, field == "Inventory")
		}
	}
}

// The public ground view omits ItemInstances when every current item is plain.
// Expand only that representation; each code still names one unbound unit.
func currentSackItemInstances(sack sim.Sack) []sim.ItemInstance {
	if len(sack.ItemInstances) != 0 {
		return sack.ItemInstances
	}
	items := make([]sim.ItemInstance, len(sack.Items))
	for i, code := range sack.Items {
		items[i] = sim.PlainItem(code)
	}
	return items
}

// Current Sack roots and ordinary records share the same identity table as
// their Items. Only exact retired Sack bindings authorize record removal.
func projectCurrentSackRoots(b *generatedDocumentBuilder, bindings *SnapshotSAVObjectBindings, registry *sim.SavedObjects, indices map[sim.SavedObjectID]uint16, put func(sim.SavedObjectID, sav.DocumentRecordData) error) (map[uint16]bool, error) {
	oldObjects := make(map[uint16]bool, len(bindings.Sacks))
	oldIDs := make(map[sim.SavedObjectID]bool, len(bindings.Sacks))
	oldCoverage := make(map[sim.SavedObjectID]string, len(bindings.Sacks))
	for _, binding := range bindings.Sacks {
		oldIDs[binding.ID] = true
		oldCoverage[binding.ID] = binding.Unavailable
		if binding.ObjectIndex != 0 {
			oldObjects[binding.ObjectIndex] = true
		}
	}
	retired := make(map[uint16]bool)
	var rows []SnapshotSAVObjectBinding
	for _, sack := range registry.Sacks {
		delete(oldIDs, sack.ID)
		index := indices[sack.ID]
		if index != 0 && (int(index) > len(b.doc.Objects) || b.doc.Objects[index-1].Class != "Sack") {
			return nil, fmt.Errorf("current Sack %d has a different ordinary object", sack.ID)
		}
		if sack.Retired {
			if index != 0 {
				retired[index] = true
			}
			indices[sack.ID] = 0
			rows = append(rows, SnapshotSAVObjectBinding{ID: sack.ID})
			continue
		}
		container, ok := savedSackContainer(registry, sack.ID)
		if !ok {
			return nil, fmt.Errorf("current Sack %d lacks its exact container", sack.ID)
		}
		// Native key absence has its own anchored policy. Retaining the key
		// of this exact ordinary object avoids reminting its cell reference.
		if sack.Token.Identity == 0 && index != 0 {
			var err error
			sack.Token.Identity, err = savedStructureValue(&b.doc.Objects[index-1], "Identity")
			if err != nil {
				return nil, err
			}
		}
		record, err := savedCurrentSackRecord(sack, container, indices)
		if err != nil {
			return nil, err
		}
		if err := put(sack.ID, record); err != nil {
			return nil, err
		}
		coverage := oldCoverage[sack.ID]
		rows = append(rows, SnapshotSAVObjectBinding{ID: sack.ID, ObjectIndex: indices[sack.ID], Unavailable: coverage})
	}
	if len(oldIDs) != 0 {
		return nil, fmt.Errorf("current Sack bindings name missing native objects")
	}
	// Preserve unbound ordinary roots in place, and project the exact current
	// registered root sequence, including repeated references to one Sack.
	roots := make([]uint16, 0, len(b.doc.World.Sacks)+len(registry.SackRoots))
	at := 0
	appendRoot := func() error {
		index := indices[registry.SackRoots[at]]
		if index == 0 {
			return fmt.Errorf("current Sack root lacks its ordinary object")
		}
		roots = append(roots, index)
		at++
		return nil
	}
	for _, index := range b.doc.World.Sacks {
		if !oldObjects[index] {
			roots = append(roots, index)
		} else if at < len(registry.SackRoots) {
			if err := appendRoot(); err != nil {
				return nil, err
			}
		}
	}
	for at < len(registry.SackRoots) {
		if err := appendRoot(); err != nil {
			return nil, err
		}
	}
	b.doc.World.Sacks, bindings.Sacks = roots, rows
	return retired, nil
}

func currentItemRecordValue(item sim.ItemInstance, held bool, weights map[uint16]sim.ItemWeight, table *mapload.Table) sim.ItemInstance {
	constructor := mapload.SourceConstructedItem(sim.PlainItem(item.Code), table)
	historyClass := item.NativeRecord != nil && item.NativeRecord.Class != 0
	if item.SourceEquipment.Class == 0 && (held || !item.WeightPresent || historyClass) {
		item.SourceEquipment = weights[item.Code].Constructor
		if item.SourceEquipment.Class == 0 {
			item.SourceEquipment = constructor.SourceEquipment
		}
		if item.SourceEquipment.Class == 0 {
			code := data.ItemCode(item.Code)
			switch code.B() {
			case 1:
				item.SourceEquipment.Class = sim.SourceWeapon
			case 2:
				item.SourceEquipment.Class = sim.SourceShield
			case 3, 4, 5, 6, 7, 8, 9, 10, 11, 12:
				item.SourceEquipment.Class = sim.SourceArmor
			}
			if item.SourceEquipment.Class != 0 {
				item.SourceEquipment.DefinitionRow = uint8(code.D())
			}
		}
		if historyClass {
			item.SourceEquipment.Class = item.NativeRecord.Class
		}
	}
	if !item.WeightPresent {
		item.WeightPresent, item.Weight = true, constructor.Weight
		if weight, present := weights[item.Code]; present {
			item.Weight = int16(weight.Weight)
		}
	}
	return item
}

func currentUnreachableItems(doc sav.DocumentData, retiredSacks map[uint16]bool) ([]uint16, error) {
	seen := map[uint16]bool{}
	var walk func(uint16) error
	var record func(sav.DocumentRecordData) error
	record = func(r sav.DocumentRecordData) error {
		for _, refs := range r.RefSlots {
			for _, ref := range refs.Objects {
				if err := walk(ref); err != nil {
					return err
				}
			}
		}
		for _, child := range r.Inline {
			if err := record(child.Record); err != nil {
				return err
			}
		}
		for _, child := range r.Groups {
			if err := record(child); err != nil {
				return err
			}
		}
		return nil
	}
	walk = func(ref uint16) error {
		if ref == 0 || seen[ref] {
			return nil
		}
		if int(ref) > len(doc.Objects) {
			return fmt.Errorf("current graph reference exceeds Document")
		}
		seen[ref] = true
		return record(doc.Objects[ref-1])
	}
	roots := [][]uint16{doc.Players, doc.DeadActors}
	if doc.World != nil {
		roots = append(roots, doc.World.Buildings, doc.World.Sacks, doc.World.Effects)
	}
	for _, list := range roots {
		for _, ref := range list {
			if err := walk(ref); err != nil {
				return nil, err
			}
		}
	}
	var retired []uint16
	for i, r := range doc.Objects {
		if seen[uint16(i+1)] && retiredSacks[uint16(i+1)] {
			return nil, fmt.Errorf("current graph retains retired Sack node %d", i+1)
		}
		if !seen[uint16(i+1)] {
			if !savedItemClass(r.Class) && r.Class != "Effect" && r.Class != "Spell" && !(r.Class == "Sack" && retiredSacks[uint16(i+1)]) {
				return nil, fmt.Errorf("current graph lost %s node %d", r.Class, i+1)
			}
			retired = append(retired, uint16(i+1))
		}
	}
	return retired, nil
}
