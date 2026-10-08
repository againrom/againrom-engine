package game

import (
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type currentObjectGraph struct {
	Version    uint32              `json:",omitempty"`
	ItemRoots  []sim.SavedItemRoot `json:",omitempty"`
	BookActors []sim.EntityID      `json:",omitempty"`
	Present    bool
	NextID     sim.SavedObjectID
	Containers []currentObjectContainer
}

type currentObjectContainer struct {
	Owner    sim.SavedObjectOwner
	Coverage sim.SavedObjectCoverage
}

// Object names the sole value-bearing wire node. Private payloads are allowed
// only for records without a reachable wire node, including retired owners.
type currentOwnedObject struct {
	ID                            sim.SavedObjectID
	Object                        uint16
	Kind                          uint8
	Origin                        sim.SavedObjectOrigin
	Owner                         sim.SavedObjectOwner `json:",omitzero"`
	Retired                       bool                 `json:",omitempty"`
	Coverage                      sim.SavedObjectCoverage
	WeightKnown, EquipmentKnown   bool
	WeightAnchor, EquipmentAnchor *[32]byte             `json:",omitempty"`
	CountLift                     *currentItemCountLift `json:",omitempty"`
	Definition                    sim.SourceWeaponDefinition
	DefinitionRow                 *uint8 `json:",omitempty"`
	EffectsUnsupported            bool
	IdentityPresent               *bool                  `json:",omitempty"`
	IdentityAnchor                *uint32                `json:",omitempty"`
	Item                          *sim.SavedItemObject   `json:",omitempty"`
	Effect                        *sim.SavedEffectObject `json:",omitempty"`
	Spell                         *sim.SavedSpellObject  `json:",omitempty"`
	Range                         *currentSpellRange     `json:",omitempty"`
	Sack                          *sim.SavedSackObject   `json:",omitempty"`
	// TokenRow is an unclassed Item's registry row; its record names the
	// row derived from its code.
	TokenRow                  *uint8    `json:",omitempty"`
	NativeRecordKnown         bool      `json:",omitempty"`
	NativeRecordAnchor        *[32]byte `json:",omitempty"`
	NativeRecordAnchorVersion uint8     `json:",omitempty"`
}

type currentSpellRange struct {
	Wire  sim.SourceItemSpell
	Value uint8
}

func currentItemPolicy(v sim.ItemStack, object uint16) currentOwnedObject {
	row := currentOwnedObject{ID: v.ObjectID, Object: object, Kind: 1, WeightKnown: v.WeightPresent,
		EquipmentKnown: v.SourceEquipment.Class != 0, Definition: v.SourceEquipment.Definition, EffectsUnsupported: v.SourceEquipment.EffectsUnsupported, NativeRecordKnown: v.NativeRecord != nil}
	if v.SourceEquipment.Class == sim.SourceWeapon {
		def := v.SourceEquipment.DefinitionRow
		row.DefinitionRow = &def
	}
	if v.ObjectID != 0 && object != 0 && v.Count > 65535 {
		row.CountLift = &currentItemCountLift{Wire: 65535, Lift: v.Count - 65535}
	}
	return row
}

func readCurrentItem(doc *sav.DocumentData, row currentOwnedObject, table *mapload.Table) (sim.SavedItemObject, error) {
	if err := validateNativeItemRecordPolicy(row); err != nil {
		return sim.SavedItemObject{}, err
	}
	if err := validateCurrentItemCount(row); err != nil {
		return sim.SavedItemObject{}, err
	}
	v, err := savedItemRecord(doc, row.Object)
	if err != nil {
		return v, err
	}
	record := nativeItemRecord(v)
	if lift := row.CountLift; lift != nil && v.Value.Count == uint32(lift.Wire) {
		v.Value.Count += lift.Lift
	}
	if !row.WeightKnown && currentItemAnchorMatches(row.WeightAnchor, v.Value, false) {
		v.Value.WeightPresent, v.Value.Weight = false, 0
	}
	if !row.EquipmentKnown {
		if currentItemAnchorMatches(row.EquipmentAnchor, v.Value, true) {
			v.Value.SourceEquipment = sim.SourceEquipment{}
			if row.TokenRow != nil {
				v.Token.T0C = *row.TokenRow
			}
		} else {
			v.Value.SourceEquipment = mapload.BindSourceItemDefinition(v.Value.Instance(), table).SourceEquipment
		}
	} else {
		v.Value.SourceEquipment.EffectsUnsupported = row.EffectsUnsupported
		if v.Value.SourceEquipment.Class == sim.SourceWeapon {
			if row.DefinitionRow == nil || *row.DefinitionRow == v.Value.SourceEquipment.DefinitionRow {
				v.Value.SourceEquipment.Definition = row.Definition
			} else {
				v.Value.SourceEquipment = mapload.BindSourceItemDefinition(v.Value.Instance(), table).SourceEquipment
			}
		}
	}
	v.ID, v.Value.ObjectID = row.ID, row.ID
	if row.ID == 0 {
		weightEdited := record.Class != 0 && !row.WeightKnown && v.Value.WeightPresent
		if row.NativeRecordKnown || row.NativeRecordAnchor == nil || *row.NativeRecordAnchor != nativeItemAbsenceAnchor(doc, record, row.NativeRecordAnchorVersion) || weightEdited {
			v.Value.NativeRecord = &record
		}
	}
	return v, nil
}

func captureCurrentObjects(state *SnapshotSAVDocument, w *sim.World) (*currentObjectGraph, []currentOwnedObject, error) {
	r := w.SavedObjects()
	g := &currentObjectGraph{Version: sim.SavedObjectsVersion, Present: r != nil}
	var rows []currentOwnedObject
	indices := map[sim.SavedObjectID]uint16{}
	if state.Objects != nil {
		indices = savedObjectIndices(state.Objects)
	}
	seen := map[uint16]bool{}
	itemPolicy := func(v sim.ItemStack, object uint16) currentOwnedObject {
		seen[object] = true
		return currentItemPolicy(v, object)
	}
	if r != nil {
		g.NextID = r.NextID
		for _, root := range r.BookRoots {
			g.BookActors = append(g.BookActors, root.Entity)
		}
		for _, v := range r.Items {
			row := itemPolicy(v.Value, indices[v.ID])
			if v.Value.SourceEquipment.Class == 0 {
				tokenRow := v.Token.T0C
				row.TokenRow = &tokenRow
			}
			identity := v.Token.Identity != 0
			row.IdentityPresent = &identity
			row.Origin, row.Retired, row.Coverage = v.Origin, v.Retired, v.Coverage
			if v.Retired || row.Object == 0 {
				// Child nodes own their scalars even when the Item itself has
				// no ordinary root. Keep only the ordered child identities here.
				v.Value.Effects = nil
				v.Value.SourceEquipment.Spell = sim.SourceItemSpell{}
				row.Object, row.Item = 0, &v
			} else if row.Object == 0 {
				return nil, nil, fmt.Errorf("current Item %d has no wire binding", v.ID)
			}
			rows = append(rows, row)
		}
		for _, v := range r.Effects {
			row := currentOwnedObject{ID: v.ID, Object: indices[v.ID], Kind: 2, Origin: v.Origin, Coverage: v.Coverage}
			identity := v.Token.Identity != 0
			row.IdentityPresent = &identity
			if v.Retired || row.Object == 0 {
				row.Object, row.Effect = 0, &v
			}
			rows = append(rows, row)
		}
		for _, v := range r.Spells {
			row := currentOwnedObject{ID: v.ID, Object: indices[v.ID], Kind: 3, Origin: v.Origin, Coverage: v.Coverage}
			identity := v.This != 0
			row.IdentityPresent = &identity
			if v.Retired || row.Object == 0 {
				row.Object, row.Spell = 0, &v
			} else {
				wire, err := savedSpellRecord(&state.Document.Objects[row.Object-1])
				if err != nil {
					return nil, nil, err
				}
				if wire.Value.Range != v.Value.Range {
					row.Range = &currentSpellRange{Wire: wire.Value, Value: v.Value.Range}
				}
			}
			rows = append(rows, row)
		}
		for _, v := range r.Sacks {
			row := currentOwnedObject{ID: v.ID, Object: indices[v.ID], Kind: 4, Origin: v.Origin, Coverage: v.Coverage}
			identity := v.Token.Identity != 0
			row.IdentityPresent = &identity
			if v.Retired {
				row.Object, row.Sack = 0, &v
			} else if row.Object == 0 {
				return nil, nil, fmt.Errorf("current Sack has no wire binding")
			}
			rows = append(rows, row)
		}
		for _, root := range r.ItemRoots {
			if root.Owner.Kind == sim.SavedOwnerSession {
				g.ItemRoots = append(g.ItemRoots, root)
			}
		}
		for _, c := range r.Containers {
			g.Containers = append(g.Containers, currentObjectContainer{c.Owner, c.Coverage})
		}
	}
	for _, actor := range state.Actors {
		if actor.Retired {
			continue
		}
		if actor.ObjectIndex == 0 || state.Document == nil || int(actor.ObjectIndex) > len(state.Document.Objects) {
			return nil, nil, fmt.Errorf("current actor inventory lost its object")
		}
		record := &state.Document.Objects[actor.ObjectIndex-1]
		pack, _ := w.CarriedStacks(actor.EntityID)
		refs, _ := savedObjectRefs(record, "Inventory")
		at := 0
		for _, v := range pack {
			if sim.StackStateEqual(v, sim.ItemStack{}) {
				if at >= len(refs) || refs[at] != 0 {
					return nil, nil, fmt.Errorf("current null inventory position differs")
				}
				at++
				continue
			}
			remaining := v.Count
			if v.ObjectID != 0 {
				remaining = min(remaining, uint32(65535))
			}
			for ; remaining != 0; remaining -= min(remaining, uint32(65535)) {
				if at >= len(refs) || refs[at] == 0 {
					return nil, nil, fmt.Errorf("current inventory binding is incomplete")
				}
				if !seen[refs[at]] {
					rows = append(rows, itemPolicy(v, refs[at]))
				}
				at++
			}
		}
		if at != len(refs) {
			return nil, nil, fmt.Errorf("current inventory has unmatched positions")
		}
		worn, _ := w.EquippedItems(actor.EntityID)
		for slot, v := range worn {
			if v.Empty() {
				continue
			}
			field, ordinal := "Worn", slot
			if slot == 0 {
				field, ordinal = "HeldWeapon", 0
			}
			if slot == 1 {
				field, ordinal = "HeldShield", 0
			}
			refs, _ := savedObjectRefs(record, field)
			if ordinal >= len(refs) || refs[ordinal] == 0 {
				return nil, nil, fmt.Errorf("current equipment binding is incomplete")
			}
			if !seen[refs[ordinal]] {
				rows = append(rows, itemPolicy(sim.StackItem(v, 1), refs[ordinal]))
			}
		}
	}
	for _, sack := range w.Sacks() {
		object := indices[sack.ObjectID]
		if sack.ObjectID == 0 {
			for _, index := range state.Document.World.Sacks {
				token, _, _, err := savedSackRecord(&state.Document.Objects[index-1])
				if err != nil {
					return nil, nil, err
				}
				if int32(token.Position[2]) == sack.X && int32(token.Position[3]) == sack.Y {
					if object != 0 && object != index {
						return nil, nil, fmt.Errorf("current Sack cell binding is ambiguous")
					}
					object = index
				}
			}
		}
		if object == 0 {
			return nil, nil, fmt.Errorf("current Sack cell has no binding")
		}
		if sack.ObjectID == 0 {
			rows = append(rows, currentOwnedObject{Kind: 4, Object: object})
		}
		refs, _ := savedObjectRefs(&state.Document.Objects[object-1], "Contents")
		items := currentSackItemInstances(sack)
		at := uint64(0)
		for _, ref := range refs {
			v, err := savedItemRecord(state.Document, ref)
			if err != nil {
				return nil, nil, err
			}
			if at >= uint64(len(items)) {
				return nil, nil, fmt.Errorf("current Sack item run exceeds native holdings")
			}
			count := v.Value.Count
			if id := items[at].ObjectID; id != 0 {
				item, ok := r.Item(id)
				if !ok || indices[id] != ref {
					return nil, nil, fmt.Errorf("current Sack child lost its exact Item binding")
				}
				count = item.Value.Count
			}
			if at+uint64(count) > uint64(len(items)) {
				return nil, nil, fmt.Errorf("current Sack item run exceeds native holdings")
			}
			if !seen[ref] {
				rows = append(rows, itemPolicy(sim.StackItem(items[at], count), ref))
			}
			at += uint64(count)
		}
		if at != uint64(len(items)) {
			return nil, nil, fmt.Errorf("current Sack has unmatched holdings")
		}
	}
	for i := range rows {
		if err := captureCurrentItemAbsence(state.Document, &rows[i]); err != nil {
			return nil, nil, err
		}
	}
	return g, rows, nil
}

func restoreCurrentObjects(ms *Mission, g *currentObjectGraph, rows []currentOwnedObject, table *mapload.Table) (map[sim.SavedObjectID]sim.SavedObjectID, error) {
	if g == nil {
		return nil, nil
	}
	doc, old := ms.savedDocument.Document, ms.World.SavedObjects()
	legacy := g.Version == 0 || g.Version == 1
	if !legacy && g.Version != sim.SavedObjectsVersion {
		return nil, fmt.Errorf("invalid current object graph version")
	}
	if legacy && (len(g.ItemRoots) != 0 || len(g.BookActors) != 0) {
		return nil, fmt.Errorf("legacy current graph has explicit roots")
	}
	if old == nil {
		return nil, fmt.Errorf("current ownership lacks imported graph")
	}
	oldIndices := savedObjectIndices(ms.savedDocument.Objects)
	oldByObject := map[uint16]sim.SavedObjectID{}
	for id, object := range oldIndices {
		if object != 0 {
			oldByObject[object] = id
		}
	}
	byObject := map[uint16]sim.SavedObjectID{}
	ids, identities := map[sim.SavedObjectID]bool{}, map[sim.SavedObjectID]sim.SavedObjectID{}
	values := map[sim.SavedObjectID]sim.ItemStack{}
	ordinaryItems := map[uint16]sim.ItemStack{}
	r := &sim.SavedObjects{Version: sim.SavedObjectsVersion, NextID: g.NextID}
	metadata := &SnapshotSAVObjectBindings{Version: 2}
	for _, row := range rows {
		if p := row.Range; p != nil && (row.Kind != 3 || row.Object == 0 || row.ID == 0 || !p.Wire.Present || p.Wire.ID < 1 || p.Wire.ID > 28 || p.Wire.Range == p.Value) {
			return nil, fmt.Errorf("invalid current Spell range operand")
		}
		if err := validateCurrentItemCount(row); err != nil {
			return nil, err
		}
		if err := validateCurrentObjectIdentityAbsence(row, legacy); err != nil {
			return nil, err
		}
		if err := validateCurrentItemAbsence(row); err != nil {
			return nil, err
		}
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
		if row.Sack != nil {
			private++
		}
		if row.Kind < 1 || row.Kind > 4 || int(row.Object) > len(doc.Objects) || (row.Object == 0) != (private == 1) || row.Object != 0 && private != 0 || row.ID != 0 && ids[row.ID] {
			return nil, fmt.Errorf("invalid current object identity or payload")
		}
		if row.ID != 0 {
			ids[row.ID] = true
		}
		if row.Object != 0 {
			if _, exists := byObject[row.Object]; exists {
				return nil, fmt.Errorf("duplicate current object alias")
			}
			byObject[row.Object] = row.ID
			if id := oldByObject[row.Object]; id != 0 {
				identities[id] = row.ID
			}
		}
	}
	privateItems := make(map[sim.SavedObjectID]bool)
	ranges := make(map[sim.SavedObjectID]sim.SourceItemSpell)
	for _, row := range rows {
		b := SnapshotSAVObjectBinding{ID: row.ID, ObjectIndex: row.Object}
		if row.Object == 0 && (row.Item != nil && !row.Item.Retired || row.Effect != nil && !row.Effect.Retired || row.Spell != nil && !row.Spell.Retired) {
			b.Unavailable = savedActiveScrollUnavailable
		}
		switch row.Kind {
		case 1:
			var v sim.SavedItemObject
			if row.Item != nil {
				v = *row.Item
				privateItems[row.ID] = true
				if legacy && row.Owner.Kind != sim.SavedOwnerRetired && row.Owner.Kind != sim.SavedOwnerSession {
					return nil, fmt.Errorf("private current Item is reachable")
				}
				if !legacy && (v.Owner != (sim.SavedObjectOwner{}) || v.InFlight != 0 || v.ID != row.ID || v.Value.ObjectID != row.ID || v.Retired != row.Retired || len(v.Value.Effects) != 0 || v.Value.SourceEquipment.Spell != (sim.SourceItemSpell{})) {
					return nil, fmt.Errorf("private current Item duplicates child values or root policy")
				}
			} else {
				var err error
				v, err = readCurrentItem(doc, row, table)
				if err != nil {
					return nil, err
				}
				refs, _ := savedObjectRefs(&doc.Objects[row.Object-1], "Effects")
				for _, ref := range refs {
					if byObject[ref] == 0 && row.ID != 0 {
						return nil, fmt.Errorf("current Item has unresolved Effect alias")
					}
					v.Effects = append(v.Effects, byObject[ref])
				}
				refs, _ = savedObjectRefs(&doc.Objects[row.Object-1], "WeaponSpell")
				if len(refs) == 1 && refs[0] != 0 {
					v.Spell = byObject[refs[0]]
					if row.ID != 0 && v.Spell == 0 {
						return nil, fmt.Errorf("current Item has unresolved Spell alias")
					}
				}
			}
			v.ID, v.Value.ObjectID, v.Origin, v.Owner, v.Retired, v.Coverage = row.ID, row.ID, row.Origin, row.Owner, row.Retired, row.Coverage
			v.Token.Identity = restoreCurrentObjectIdentity(row, v.Token.Identity)
			if oldID := oldByObject[row.Object]; oldID != 0 {
				values[oldID] = v.Value
			}
			if row.Object != 0 {
				ordinaryItems[row.Object] = v.Value
			}
			if row.ID != 0 {
				r.Items = append(r.Items, v)
				metadata.Items = append(metadata.Items, b)
			}
		case 2:
			var v sim.SavedEffectObject
			if row.Effect != nil {
				v = *row.Effect
			} else {
				var err error
				v, err = savedEffectRecord(&doc.Objects[row.Object-1])
				if err != nil {
					return nil, err
				}
			}
			v.ID, v.Origin, v.Coverage = row.ID, row.Origin, row.Coverage
			v.Token.Identity = restoreCurrentObjectIdentity(row, v.Token.Identity)
			r.Effects = append(r.Effects, v)
			metadata.Effects = append(metadata.Effects, b)
		case 3:
			var v sim.SavedSpellObject
			if row.Spell != nil {
				v = *row.Spell
			} else {
				var err error
				v, err = savedSpellRecord(&doc.Objects[row.Object-1])
				if err != nil {
					return nil, err
				}
				if p := row.Range; p != nil && v.Value == p.Wire {
					v.Value.Range = p.Value
					ranges[row.ID] = v.Value
				}
			}
			v.ID, v.Origin, v.Coverage = row.ID, row.Origin, row.Coverage
			v.This = restoreCurrentObjectIdentity(row, v.This)
			r.Spells = append(r.Spells, v)
			metadata.Spells = append(metadata.Spells, b)
		case 4:
			var v sim.SavedSackObject
			if row.Sack != nil {
				v = *row.Sack
			} else {
				var err error
				v.Token, v.Gold, _, err = savedSackRecord(&doc.Objects[row.Object-1])
				if err != nil {
					return nil, err
				}
			}
			v.ID, v.Origin, v.Coverage = row.ID, row.Origin, row.Coverage
			v.Token.Identity = restoreCurrentObjectIdentity(row, v.Token.Identity)
			if row.ID != 0 {
				r.Sacks = append(r.Sacks, v)
				metadata.Sacks = append(metadata.Sacks, b)
			} else {
				metadata.Unavailable = append(metadata.Unavailable, SnapshotSAVObjectCoverage{ObjectIndex: row.Object, Reason: savedSackNativeUnavailable})
			}
		}
	}
	effects := make(map[sim.SavedObjectID]sim.ItemEffect, len(r.Effects))
	for _, row := range r.Effects {
		effects[row.ID] = row.Value
	}
	spells := make(map[sim.SavedObjectID]sim.SourceItemSpell, len(r.Spells))
	for _, row := range r.Spells {
		spells[row.ID] = row.Value
	}
	itemRanges := make(map[sim.SavedObjectID]sim.SourceItemSpell)
	for i := range r.Items {
		row := &r.Items[i]
		if value, ok := ranges[row.Spell]; ok {
			row.Value.SourceEquipment.Spell = value
			itemRanges[row.ID] = value
		}
		if !privateItems[row.ID] {
			continue
		}
		row.Value.Effects = nil
		for _, id := range row.Effects {
			value, ok := effects[id]
			if !ok {
				return nil, fmt.Errorf("private current Item has an unresolved Effect")
			}
			row.Value.Effects = append(row.Value.Effects, value)
		}
		row.Value.SourceEquipment.Spell = sim.SourceItemSpell{}
		if row.Spell != 0 {
			value, ok := spells[row.Spell]
			if !ok {
				return nil, fmt.Errorf("private current Item has an unresolved Spell")
			}
			row.Value.SourceEquipment.Spell = value
		}
	}
	for id, item := range values {
		if value, ok := itemRanges[item.ObjectID]; ok {
			item.SourceEquipment.Spell = value
			values[id] = item
		}
	}
	for _, c := range g.Containers {
		var v sim.SavedObjectContainer
		found := false
		for _, imported := range old.Containers {
			owner := imported.Owner
			if owner.Kind == sim.SavedOwnerSack {
				owner.Object = identities[owner.Object]
			}
			if owner == c.Owner {
				v, found = imported, true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("current container owner is unresolved: %+v", c.Owner)
		}
		v.Owner, v.Coverage = c.Owner, c.Coverage
		var slots []sim.SavedObjectID
		for _, id := range v.Items {
			mapped, explicit := identities[id]
			// The ordinary document can carry a newly authored Item that was
			// absent from the retained current graph.  Its native slot remains
			// a live zero binding until a producer gives it an identity; the
			// container therefore owns the uncertainty about load and merge
			// policy.  Without carrying this marker forward, ReplaceCurrentObjects
			// rejects the otherwise exact current pack as an uncovered partial
			// graph.
			if mapped == 0 && (c.Owner.Kind == sim.SavedOwnerActorPack || c.Owner.Kind == sim.SavedOwnerSack) {
				v.Coverage.Unknown |= sim.SavedUnknownContainerLoad | sim.SavedUnknownMergePolicy
			}
			count := uint32(1)
			if c.Owner.Kind == sim.SavedOwnerSack && id != 0 && explicit && mapped == 0 {
				value, present := values[id]
				if !present || value.ObjectID != 0 || value.Count == 0 {
					return nil, fmt.Errorf("current unbound Sack slot lacks its ordinary Item value")
				}
				count = value.Count
			}
			if uint64(len(slots))+uint64(count) > sim.MaxSavedObjects {
				return nil, fmt.Errorf("current Sack container exceeds native slot capacity")
			}
			for range count {
				slots = append(slots, mapped)
			}
		}
		v.Items = slots
		r.Containers = append(r.Containers, v)
		metadata.Containers = append(metadata.Containers, SnapshotSAVContainerBinding{Owner: c.Owner})
	}
	if !legacy {
		for _, root := range old.ItemRoots {
			if root.Owner.Kind == sim.SavedOwnerActorWorn && identities[root.ID] != 0 {
				if err := r.AddItemRoot(identities[root.ID], root.Owner); err != nil {
					return nil, err
				}
			}
		}
		for _, root := range g.ItemRoots {
			if root.Owner.Kind != sim.SavedOwnerSession {
				return nil, fmt.Errorf("current root duplicates ordinary ownership")
			}
			if err := r.AddItemRoot(root.ID, root.Owner); err != nil {
				return nil, err
			}
		}
	}

	for _, id := range old.SackRoots {
		if mapped := identities[id]; mapped != 0 {
			r.SackRoots = append(r.SackRoots, mapped)
		}
	}
	incoming := savedDocumentIncoming(doc)
	bookActors := map[sim.EntityID]bool{}
	for _, entity := range g.BookActors {
		if bookActors[entity] {
			return nil, fmt.Errorf("duplicate current book actor")
		}
		bookActors[entity] = true
		var object uint16
		for _, actor := range ms.savedDocument.Actors {
			if actor.EntityID == entity && !actor.Retired {
				if object != 0 {
					return nil, fmt.Errorf("ambiguous current book actor")
				}
				object = actor.ObjectIndex
			}
		}
		if object == 0 || int(object) > len(doc.Objects) {
			return nil, fmt.Errorf("current book actor has no ordinary binding")
		}
		refs, _ := savedObjectRefs(&doc.Objects[object-1], "Spells")
		if len(refs) > 28 {
			return nil, fmt.Errorf("current book exceeds ordinary slot count")
		}
		root := sim.SavedBookRoot{Entity: entity}
		for slot, ref := range refs {
			if id := byObject[ref]; id != 0 {
				root.Slots[slot] = id
				if incoming[ref] == 0 {
					return nil, fmt.Errorf("current book edge has no ordinary owner")
				}
				incoming[ref]--
			}
		}
		r.BookRoots = append(r.BookRoots, root)
	}
	slices.SortFunc(r.BookRoots, func(a, b sim.SavedBookRoot) int { return cmp.Compare(a.Entity, b.Entity) })
	for _, row := range rows {
		if row.Kind == 1 && row.Object != 0 && row.ID != 0 {
			for _, field := range []string{"Effects", "WeaponSpell"} {
				refs, _ := savedObjectRefs(&doc.Objects[row.Object-1], field)
				for _, ref := range refs {
					if ref != 0 {
						incoming[ref]--
					}
				}
			}
		}
	}
	for i := range r.Effects {
		if object := metadata.Effects[i].ObjectIndex; object != 0 {
			r.Effects[i].ExternalReferences = incoming[object]
		}
	}
	for i := range r.Spells {
		if object := metadata.Spells[i].ObjectIndex; object != 0 {
			r.Spells[i].ExternalReferences = incoming[object]
		}
	}
	slices.SortFunc(metadata.Unavailable, func(a, b SnapshotSAVObjectCoverage) int { return int(a.ObjectIndex) - int(b.ObjectIndex) })
	if legacy && g.Present {
		r.Version = 1
		if err := r.MigrateLegacyOwners(); err != nil {
			return nil, err
		}
	}

	if !g.Present {
		if len(r.Items)+len(r.Effects)+len(r.Spells)+len(r.Sacks)+len(r.Containers)+len(r.ItemRoots)+len(r.BookRoots) != 0 || g.NextID != 0 {
			return nil, fmt.Errorf("absent current registry has rows")
		}
		r, metadata = nil, nil
	}
	ground, err := currentUnboundSackValues(doc, rows, ordinaryItems)
	if err != nil {
		return nil, err
	}
	if err := ms.World.ReplaceCurrentObjects(r, identities, values, ground...); err != nil {
		return nil, err
	}
	ms.savedDocument.Objects = metadata
	shared := map[sim.SavedObjectID]sim.SavedObjectID{}
	for id := range ids {
		shared[id] = id
	}
	return shared, nil
}

func validateCurrentObjectBindings(state *SnapshotSAVDocument, world *sim.World) error {
	r, m := world.SavedObjects(), state.Objects
	if r == nil && m == nil {
		return nil
	}
	if r == nil || m == nil {
		return fmt.Errorf("current object registry presence differs")
	}
	if _, err := world.MarshalBinary(); err != nil {
		return err
	}
	if _, err := cloneSavedObjectBindings(m, state.Document); err != nil {
		return err
	}
	if len(r.Items) != len(m.Items) || len(r.Effects) != len(m.Effects) || len(r.Spells) != len(m.Spells) || len(r.Sacks) != len(m.Sacks) || len(r.Containers) != len(m.Containers) {
		return fmt.Errorf("current object bindings omit native rows")
	}
	for i, v := range r.Items {
		if m.Items[i].ID != v.ID {
			return fmt.Errorf("current Item identity differs")
		}
	}
	for i, v := range r.Effects {
		if m.Effects[i].ID != v.ID {
			return fmt.Errorf("current Effect identity differs")
		}
	}
	for i, v := range r.Spells {
		if m.Spells[i].ID != v.ID {
			return fmt.Errorf("current Spell identity differs")
		}
	}
	for i, v := range r.Sacks {
		if m.Sacks[i].ID != v.ID {
			return fmt.Errorf("current Sack identity differs")
		}
	}
	for i, v := range r.Containers {
		if m.Containers[i].Owner != v.Owner {
			return fmt.Errorf("current container owner differs")
		}
	}
	return nil
}
