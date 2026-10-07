package game

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type SnapshotSAVContainerBinding struct {
	Owner       sim.SavedObjectOwner
	Unavailable string
}

func savedObjectRefs(r *sav.DocumentRecordData, name string) ([]uint16, bool) {
	for _, refs := range r.RefSlots {
		if refs.Name == name {
			return refs.Objects, true
		}
	}
	return nil, false
}

func savedObjectRaw(r *sav.DocumentRecordData, name string, size int) ([]byte, error) {
	for _, raw := range r.Raw {
		if raw.Name == name && len(raw.Bytes) == size {
			return raw.Bytes, nil
		}
	}
	return nil, fmt.Errorf("saved SAV %s lacks %s[%d]", r.Class, name, size)
}

// ITEM-SAVE-014: every Token operand is distinct from the archive object index.
func savedItemToken(r *sav.DocumentRecordData) (sim.SavedObjectToken, error) {
	var token sim.SavedObjectToken
	raw, err := savedObjectRaw(r, "Block12", 12)
	if err != nil {
		return token, err
	}
	copy(token.Position[:], raw)
	values := make([]uint32, 8)
	for i, name := range []string{"RuntimeID", "T0C", "T0E", "T08", "T18", "T1C", "Identity", "Reference"} {
		values[i], err = savedStructureValue(r, name)
		if err != nil {
			return token, err
		}
	}
	token.RuntimeID, token.T0C, token.T0E, token.T08 = values[0], uint8(values[1]), uint16(values[2]), values[3]
	token.T18, token.T1C, token.Identity, token.Reference = uint16(values[4]), values[5], values[6], values[7]
	return token, nil
}

func savedItemClass(class string) bool {
	return class == "Item" || class == "Weapon" || class == "Armor" || class == "Shield"
}

func savedSpellRecord(r *sav.DocumentRecordData) (sim.SavedSpellObject, error) {
	var out sim.SavedSpellObject
	if r.Class != "Spell" {
		return out, fmt.Errorf("saved SAV Weapon child is not Spell")
	}
	v := make([]uint32, 5)
	for i, name := range []string{"S08", "S09", "S0A", "S0C", "This"} {
		var err error
		v[i], err = savedStructureValue(r, name)
		if err != nil {
			return out, err
		}
	}
	out.Value = sim.SourceItemSpell{Present: true, ID: uint8(v[0]), Range: uint8(v[1]), Defensive: uint8(v[2]), ManaCost: uint16(v[3])}
	out.This = v[4]
	return out, nil
}

func savedEffectRecord(r *sav.DocumentRecordData) (sim.SavedEffectObject, error) {
	var out sim.SavedEffectObject
	if r.Class != "Effect" {
		return out, fmt.Errorf("saved SAV Item child is not Effect")
	}
	var err error
	out.Token, err = savedItemToken(r)
	if err != nil {
		return out, err
	}
	v := make([]uint32, 4)
	for i, name := range []string{"E3C", "E3D", "E40", "E0C"} {
		v[i], err = savedStructureValue(r, name)
		if err != nil {
			return out, err
		}
	}
	out.Value, out.E0C = sim.ItemEffect{Kind: uint8(v[0]), Mode: uint8(v[1]), Operand: v[2]}, uint8(v[3])
	return out, nil
}

func savedItemRecord(doc *sav.DocumentData, index uint16) (sim.SavedItemObject, error) {
	var out sim.SavedItemObject
	if index == 0 || int(index) > len(doc.Objects) || !savedItemClass(doc.Objects[index-1].Class) {
		return out, fmt.Errorf("saved SAV owner reference is not an Item")
	}
	r := &doc.Objects[index-1]
	var err error
	out.Token, err = savedItemToken(r)
	if err != nil {
		return out, err
	}
	v := make([]uint32, 8)
	for i, name := range []string{"F40", "F42", "F44", "F45", "F46", "F47", "F48", "F4A"} {
		v[i], err = savedStructureValue(r, name)
		if err != nil {
			return out, err
		}
	}
	out.Value = sim.ItemStack{Code: uint16(v[0]), Count: v[1], Kind: uint8(v[2]), Price: int32(out.Token.T1C), Weight: int16(v[7]), WeightPresent: true}
	out.F45, out.F46, out.F47, out.F48 = uint8(v[3]), uint8(v[4]), uint8(v[5]), uint16(v[6])
	if out.Value.Code == 0 || out.Value.Count == 0 {
		return out, fmt.Errorf("saved SAV Item has zero code/count")
	}
	refs, ok := savedObjectRefs(r, "Effects")
	if !ok {
		return out, fmt.Errorf("saved SAV Item lacks Effects")
	}
	for _, ref := range refs {
		if ref == 0 || int(ref) > len(doc.Objects) {
			return out, fmt.Errorf("saved SAV Item has a null/unknown Effect")
		}
		child, err := savedEffectRecord(&doc.Objects[ref-1])
		if err != nil || child.E0C != 0 {
			return out, fmt.Errorf("saved SAV Item Effect state is unsupported: %v", err)
		}
		out.Value.Effects = append(out.Value.Effects, child.Value)
	}
	source := &out.Value.SourceEquipment
	switch r.Class {
	case "Weapon":
		source.Class = sim.SourceWeapon
		a, err := savedObjectRaw(r, "W52", 24)
		if err != nil {
			return out, err
		}
		copy(source.Attack[:], a)
		d, err := savedObjectRaw(r, "W6A", 22)
		if err != nil {
			return out, err
		}
		copy(source.Defence[:], d)
		own, err := savedStructureValue(r, "W50")
		if err != nil {
			return out, err
		}
		source.OwnKind = uint8(own)
		spells, ok := savedObjectRefs(r, "WeaponSpell")
		if !ok || len(spells) != 1 {
			return out, fmt.Errorf("saved SAV Weapon lacks its exact Spell slot")
		}
		if spells[0] != 0 {
			child, err := savedSpellRecord(&doc.Objects[spells[0]-1])
			if err != nil {
				return out, err
			}
			source.Spell = child.Value
		}
	case "Armor":
		source.Class = sim.SourceArmor
		d, err := savedObjectRaw(r, "A52", 22)
		if err != nil {
			return out, err
		}
		copy(source.Defence[:], d)
		own, err := savedStructureValue(r, "A50")
		if err != nil {
			return out, err
		}
		source.OwnKind = uint8(own)
	case "Shield":
		source.Class = sim.SourceShield
		d, err := savedObjectRaw(r, "S50", 22)
		if err != nil {
			return out, err
		}
		copy(source.Defence[:], d)
	}
	if source.Class != 0 {
		out.Token.T0C = equipmentDefinitionRow(out.Value.Code, out.Token.T0C)
		source.DefinitionRow = out.Token.T0C
	}
	return out, nil
}

func savedSackAdoptable(doc *sav.DocumentData, record *sav.DocumentRecordData) bool {
	refs, ok := savedObjectRefs(record, "Contents")
	if !ok {
		return false
	}
	for _, ref := range refs {
		if _, err := savedItemRecord(doc, ref); err != nil {
			return false
		}
	}
	return true
}

func savedDocumentIncoming(doc *sav.DocumentData) map[uint16]uint32 {
	incoming := make(map[uint16]uint32)
	var walk func(*sav.DocumentRecordData)
	walk = func(r *sav.DocumentRecordData) {
		for _, refs := range r.RefSlots {
			for _, ref := range refs.Objects {
				if ref != 0 {
					incoming[ref]++
				}
			}
		}
		for i := range r.Inline {
			walk(&r.Inline[i].Record)
		}
		for i := range r.Groups {
			walk(&r.Groups[i])
		}
	}
	for i := range doc.Objects {
		walk(&doc.Objects[i])
	}
	rootsList := [][]uint16{doc.Players, doc.DeadActors}
	if doc.World != nil {
		rootsList = append(rootsList, doc.World.Buildings, doc.World.Effects, doc.World.Sacks)
	}
	for _, roots := range rootsList {
		for _, ref := range roots {
			if ref != 0 {
				incoming[ref]++
			}
		}
	}
	return incoming
}

// savedItemOwnershipRefSlots are the only named ref slots this importer's own
// add() ever follows to bind an Item/Weapon/Armor/Shield object to a live
// owner: a Sack's Contents, and a Unit/Humanoid's pack, held and worn slots.
// No other slot in the whole SAV grammar targets an Item-class object except
// Unit's "U68" (ITEM-CASTSTATE-056, SAV-655 "Unit::Serialize" +0x68): "Both
// routes set actor+0x64 = weapon+0x80 and actor+0x68 = weapon, and clear both
// after the attempt... Unit::Serialize writes +0x68 as an archive object
// reference". That field is a transient item-cast borrow of the SAME weapon
// object already held, not a second live owner, so it must not inflate the
// ambiguous-ownership count below.
var savedItemOwnershipRefSlots = map[string]bool{
	"Contents": true, "Inventory": true, "HeldWeapon": true, "HeldShield": true, "Worn": true,
}

// savedItemOwnershipIncoming counts, per object index, how many OWNERSHIP ref
// slots claim it — the same slots importSavedItemObjects itself walks to bind
// an owner. It is the source of truth for "ambiguous live ownership": a
// document-wide reference count (savedDocumentIncoming) also counts non-owning
// slots such as Unit's U68 (see savedItemOwnershipRefSlots), which can alias
// an Item an actor already legitimately owns without creating a second owner.
func savedItemOwnershipIncoming(doc *sav.DocumentData) map[uint16]uint32 {
	incoming := make(map[uint16]uint32)
	var walk func(*sav.DocumentRecordData)
	walk = func(r *sav.DocumentRecordData) {
		for _, refs := range r.RefSlots {
			if !savedItemOwnershipRefSlots[refs.Name] {
				continue
			}
			for _, ref := range refs.Objects {
				if ref != 0 {
					incoming[ref]++
				}
			}
		}
		for i := range r.Inline {
			walk(&r.Inline[i].Record)
		}
		for i := range r.Groups {
			walk(&r.Groups[i])
		}
	}
	for i := range doc.Objects {
		walk(&doc.Objects[i])
	}
	return incoming
}

// The source owner and ordinal choose the object. Native equality is only a
// consistency check after that choice. Original actor stock with LoadState
// preserves these ordered stacks; native LOAD never calls this importer.
func importSavedItemObjects(state *SnapshotSAVDocument, world *sim.World, registry *sim.SavedObjects) ([]sim.SavedObjectBinding, error) {
	return attachSavedItemObjects(state, world, registry, sim.SavedObjectOriginal)
}

func attachSavedItemObjects(state *SnapshotSAVDocument, world *sim.World, registry *sim.SavedObjects, origin sim.SavedObjectOriginKind) ([]sim.SavedObjectBinding, error) {
	metadata, doc := state.Objects, state.Document
	incoming := savedDocumentIncoming(doc)
	ownership := savedItemOwnershipIncoming(doc)
	byObject := make(map[uint16]sim.SavedObjectID)
	var bindings []sim.SavedObjectBinding
	mint := func() sim.SavedObjectID { id := registry.NextID; registry.NextID++; return id }
	add := func(index uint16, owner sim.SavedObjectOwner, ordinal uint32, native sim.ItemStack) (sim.SavedObjectID, error) {
		if ownership[index] == 0 {
			return 0, fmt.Errorf("saved SAV Item %d has ambiguous live ownership", index)
		}
		row, err := savedItemRecord(doc, index)
		if err != nil {
			return 0, err
		}
		// Definition is the already-bound installed table operand, not a SAV
		// field. All actual serialized instance operands are compared below.
		row.Value.SourceEquipment.Definition = native.SourceEquipment.Definition
		if owner.Kind == sim.SavedOwnerActorWorn {
			native.Count = row.Value.Count
		}
		if !sim.StackStateEqual(row.Value, native) {
			return 0, fmt.Errorf("saved SAV Item %d differs at explicit owner/ordinal", index)
		}
		if id := byObject[index]; id != 0 {
			previous, ok := registry.Item(id)
			row.Value.ObjectID = id
			if !ok || !sim.StackStateEqual(previous.Value, row.Value) {
				return 0, fmt.Errorf("saved SAV aliased Item value differs")
			}
			if owner.Kind == sim.SavedOwnerActorWorn {
				if err := registry.AddItemRoot(id, owner); err != nil {
					return 0, err
				}
			}
			bindings = append(bindings, sim.SavedObjectBinding{ID: id, Owner: owner, Index: ordinal, Value: previous.Value.Clone()})
			return id, nil
		}
		row.ID, row.Origin = mint(), sim.SavedObjectOrigin{Kind: origin}
		row.Value.ObjectID = row.ID
		byObject[index] = row.ID
		if owner.Kind == sim.SavedOwnerActorWorn {
			if err := registry.AddItemRoot(row.ID, owner); err != nil {
				return 0, err
			}
		}

		refs, _ := savedObjectRefs(&doc.Objects[index-1], "Effects")
		for _, ref := range refs {
			id := byObject[ref]
			if id == 0 {
				child, err := savedEffectRecord(&doc.Objects[ref-1])
				if err != nil {
					return 0, err
				}
				child.ID, child.Origin, child.ExternalReferences = mint(), sim.SavedObjectOrigin{Kind: origin}, incoming[ref]
				id, byObject[ref] = child.ID, child.ID
				registry.Effects = append(registry.Effects, child)
				metadata.Effects = append(metadata.Effects, SnapshotSAVObjectBinding{ID: id, ObjectIndex: ref})
			}
			for i := range registry.Effects {
				if registry.Effects[i].ID == id {
					registry.Effects[i].ExternalReferences--
				}
			}
			row.Effects = append(row.Effects, id)
		}
		spells, _ := savedObjectRefs(&doc.Objects[index-1], "WeaponSpell")
		if len(spells) == 1 && spells[0] != 0 {
			ref := spells[0]
			id := byObject[ref]
			if id == 0 {
				child, err := savedSpellRecord(&doc.Objects[ref-1])
				if err != nil {
					return 0, err
				}
				child.ID, child.Origin, child.ExternalReferences = mint(), sim.SavedObjectOrigin{Kind: origin}, incoming[ref]
				id, byObject[ref] = child.ID, child.ID
				registry.Spells = append(registry.Spells, child)
				metadata.Spells = append(metadata.Spells, SnapshotSAVObjectBinding{ID: id, ObjectIndex: ref})
			}
			for i := range registry.Spells {
				if registry.Spells[i].ID == id {
					registry.Spells[i].ExternalReferences--
				}
			}
			row.Spell = id
		}
		registry.Items = append(registry.Items, row)
		metadata.Items = append(metadata.Items, SnapshotSAVObjectBinding{ID: row.ID, ObjectIndex: index})
		bindings = append(bindings, sim.SavedObjectBinding{ID: row.ID, Owner: owner, Index: ordinal, Value: row.Value.Clone()})
		return row.ID, nil
	}
	nativeSacks := world.Sacks()
	for i, binding := range metadata.Sacks {
		owner := sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: binding.ID}
		var native *sim.Sack
		for k := range nativeSacks {
			if nativeSacks[k].X == int32(registry.Sacks[i].Token.Position[2]) && nativeSacks[k].Y == int32(registry.Sacks[i].Token.Position[3]) {
				native = &nativeSacks[k]
			}
		}
		if native == nil {
			return nil, fmt.Errorf("saved SAV Sack has no addressed native owner")
		}
		refs, _ := savedObjectRefs(&doc.Objects[binding.ObjectIndex-1], "Contents")
		var offset uint64
		for ordinal, ref := range refs {
			row, err := savedItemRecord(doc, ref)
			if err != nil {
				return nil, err
			}
			end := offset + uint64(row.Value.Count)
			if end > uint64(len(native.ItemInstances)) {
				return nil, fmt.Errorf("saved SAV Sack quantity exceeds native owner")
			}
			id, err := add(ref, owner, uint32(ordinal), sim.StackItem(native.ItemInstances[offset], row.Value.Count))
			if err != nil {
				return nil, err
			}
			registry.Containers[i].Items = append(registry.Containers[i].Items, id)
			offset = end
		}
		if offset != uint64(len(native.ItemInstances)) {
			return nil, fmt.Errorf("saved SAV Sack has extra native units")
		}
	}
	for _, actor := range state.Actors {
		if actor.Retired {
			continue
		}
		r := &doc.Objects[actor.ObjectIndex-1]
		pack, ok := world.CarriedStacks(actor.EntityID)
		if !ok {
			return nil, fmt.Errorf("saved SAV pack owner missing")
		}
		worn, ok := world.EquippedItems(actor.EntityID)
		if !ok {
			return nil, fmt.Errorf("saved SAV worn owner missing")
		}
		present, err := savedStructureValue(r, "HasInventory")
		if err != nil {
			return nil, err
		}
		container := sim.SavedObjectContainer{Owner: sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: actor.EntityID}, Present: present != 0}
		refs, _ := savedObjectRefs(r, "Inventory")
		if len(refs) != len(pack) {
			return nil, fmt.Errorf("saved SAV actor %d source pack order was not retained", actor.EntityID)
		}
		if container.Present {
			container.InsertIndex, err = savedStructureValue(r, "Inventory1C")
			if err != nil {
				return nil, err
			}
			acc, err := savedStructureValue(r, "Inventory20")
			if err != nil {
				return nil, err
			}
			container.Accumulator = int32(acc)
		}
		for ordinal, ref := range refs {
			if ref == 0 {
				if !sim.StackStateEqual(pack[ordinal], sim.ItemStack{}) {
					return nil, fmt.Errorf("saved SAV null Pack position carries a native Item")
				}
				container.Items = append(container.Items, 0)
				continue
			}
			id, err := add(ref, container.Owner, uint32(ordinal), pack[ordinal])
			if err != nil {
				return nil, err
			}
			container.Items = append(container.Items, id)
		}
		registry.Containers = append(registry.Containers, container)
		for slot, site := range []string{"HeldWeapon", "HeldShield"} {
			refs, ok := savedObjectRefs(r, site)
			if !ok || len(refs) != 1 {
				return nil, fmt.Errorf("saved SAV actor lacks %s slot", site)
			}
			if refs[0] != 0 {
				if _, err := add(refs[0], sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: actor.EntityID, Slot: uint32(slot + 1)}, 0, sim.StackItem(worn[slot], 1)); err != nil {
					return nil, err
				}
			}
		}
		refs, _ = savedObjectRefs(r, "Worn")
		for slot, ref := range refs {
			if ref != 0 {
				if slot < 2 || slot >= sim.EquipSlots {
					return nil, fmt.Errorf("saved SAV armor overlaps held/native slot")
				}
				if _, err := add(ref, sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: actor.EntityID, Slot: uint32(slot + 1)}, 0, sim.StackItem(worn[slot], 1)); err != nil {
					return nil, err
				}
			}
		}
	}
	// Book slots are exact incoming edges, including aliases with weapons.
	for _, actor := range state.Actors {
		refs, present := savedObjectRefs(&doc.Objects[actor.ObjectIndex-1], "Spells")
		if !present || actor.Retired {
			continue
		}
		if len(refs) > 28 {
			return nil, fmt.Errorf("saved book exceeds native slot count")
		}
		root := sim.SavedBookRoot{Entity: actor.EntityID}
		for slot, ref := range refs {
			if ref == 0 {
				continue
			}
			if byObject[ref] == 0 {
				child, err := savedSpellRecord(&doc.Objects[ref-1])
				if err != nil {
					return nil, err
				}
				child.ID, child.Origin, child.ExternalReferences = mint(), sim.SavedObjectOrigin{Kind: origin}, incoming[ref]
				byObject[ref] = child.ID
				registry.Spells = append(registry.Spells, child)
				metadata.Spells = append(metadata.Spells, SnapshotSAVObjectBinding{ID: child.ID, ObjectIndex: ref})
			}
			root.Slots[slot] = byObject[ref]
			for i := range registry.Spells {
				if registry.Spells[i].ID == root.Slots[slot] {
					registry.Spells[i].ExternalReferences--
					break
				}
			}
		}
		registry.BookRoots = append(registry.BookRoots, root)
	}
	slices.SortFunc(registry.BookRoots, func(a, b sim.SavedBookRoot) int { return cmp.Compare(a.Entity, b.Entity) })
	for _, c := range registry.Containers {
		metadata.Containers = append(metadata.Containers, SnapshotSAVContainerBinding{Owner: c.Owner})
	}
	return bindings, nil
}

func savedObjectCoverage(c sim.SavedObjectCoverage) string {
	if c.Unknown == 0 && c.Unsupported == "" {
		return ""
	}
	return fmt.Sprintf("unknown=%x; %s", uint64(c.Unknown), c.Unsupported)
}

func cloneSavedItemBindings(src *SnapshotSAVObjectBindings, doc *sav.DocumentData, out *SnapshotSAVObjectBindings) error {
	if src.Version == 1 {
		if len(src.Items)+len(src.Effects)+len(src.Spells)+len(src.Containers) != 0 {
			return fmt.Errorf("legacy SAV object metadata carries new owners")
		}
		return nil
	}
	seenID, seenObject := make(map[sim.SavedObjectID]bool), make(map[uint16]bool)
	for _, row := range src.Sacks {
		seenID[row.ID] = true
		if row.ObjectIndex != 0 {
			seenObject[row.ObjectIndex] = true
		}
	}
	for _, group := range []struct {
		in    []SnapshotSAVObjectBinding
		dst   *[]SnapshotSAVObjectBinding
		class string
	}{{src.Items, &out.Items, "Item"}, {src.Effects, &out.Effects, "Effect"}, {src.Spells, &out.Spells, "Spell"}} {
		if len(group.in) > sim.MaxSavedObjects {
			return fmt.Errorf("saved SAV child metadata exceeds bounds")
		}
		for i, row := range group.in {
			if row.ID == 0 || seenID[row.ID] || i > 0 && group.in[i-1].ID >= row.ID || int(row.ObjectIndex) > len(doc.Objects) || len(row.Unavailable) > 600 || bytes.IndexByte([]byte(row.Unavailable), 0) >= 0 || row.ObjectIndex == 0 && row.Unavailable != "" && !savedDetachedCoverage(row.Unavailable) {
				return fmt.Errorf("saved SAV child metadata identity/coverage invalid")
			}
			seenID[row.ID] = true
			if row.ObjectIndex != 0 {
				class := doc.Objects[row.ObjectIndex-1].Class
				if seenObject[row.ObjectIndex] || class != group.class && !(group.class == "Item" && savedItemClass(class)) {
					return fmt.Errorf("saved SAV child metadata class/alias differs")
				}
				seenObject[row.ObjectIndex] = true
			}
		}
		*group.dst = slices.Clone(group.in)
	}
	if len(src.Containers) > sim.MaxSavedObjects {
		return fmt.Errorf("saved SAV container metadata exceeds bounds")
	}
	seenOwner := make(map[sim.SavedObjectOwner]bool)
	for _, c := range src.Containers {
		if seenOwner[c.Owner] || c.Owner.Kind != sim.SavedOwnerActorPack && c.Owner.Kind != sim.SavedOwnerSack || len(c.Unavailable) > 600 || bytes.IndexByte([]byte(c.Unavailable), 0) >= 0 {
			return fmt.Errorf("saved SAV container metadata invalid")
		}
		seenOwner[c.Owner] = true
	}
	out.Containers = slices.Clone(src.Containers)
	return nil
}
