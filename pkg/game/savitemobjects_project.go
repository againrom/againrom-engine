package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func savedObjectRecord(class string, token *sim.SavedObjectToken) sav.DocumentRecordData {
	r := sav.DocumentRecordData{Class: class}
	if token == nil {
		return r
	}
	r.Raw = []sav.DocumentRawData{{Name: "Block12", Bytes: bytes.Clone(token.Position[:])}}
	for _, f := range []sav.DocumentValueData{{Name: "RuntimeID", Value: token.RuntimeID}, {Name: "T0C", Value: uint32(token.T0C)}, {Name: "T0E", Value: uint32(token.T0E)}, {Name: "T08", Value: token.T08}, {Name: "T18", Value: uint32(token.T18)}, {Name: "T1C", Value: token.T1C}, {Name: "Identity", Value: token.Identity}, {Name: "Reference", Value: token.Reference}} {
		savedObjectSetValue(&r, f.Name, f.Value)
	}
	return r
}

func savedObjectSetValue(r *sav.DocumentRecordData, name string, value uint32) {
	for i := range r.Values {
		if r.Values[i].Name == name {
			r.Values[i].Value = value
			return
		}
	}
	r.Values = append(r.Values, sav.DocumentValueData{Name: name, Value: value})
	slices.SortFunc(r.Values, func(a, b sav.DocumentValueData) int { return strings.Compare(a.Name, b.Name) })
}

func savedObjectSetRefs(r *sav.DocumentRecordData, name string, refs []uint16, counted bool) {
	found := false
	for i := range r.RefSlots {
		if r.RefSlots[i].Name == name {
			r.RefSlots[i].Objects = slices.Clone(refs)
			found = true
		}
	}
	if !found {
		r.RefSlots = append(r.RefSlots, sav.DocumentRefsData{Name: name, Objects: slices.Clone(refs)})
		slices.SortFunc(r.RefSlots, func(a, b sav.DocumentRefsData) int { return strings.Compare(a.Name, b.Name) })
	}
	if counted {
		for i := range r.Counts {
			if r.Counts[i].Name == name {
				r.Counts[i].Count = uint32(len(refs))
				return
			}
		}
		r.Counts = append(r.Counts, sav.DocumentCountData{Name: name, Count: uint32(len(refs))})
		slices.SortFunc(r.Counts, func(a, b sav.DocumentCountData) int { return strings.Compare(a.Name, b.Name) })
	}
}

func savedObjectIndices(m *SnapshotSAVObjectBindings) map[sim.SavedObjectID]uint16 {
	out := make(map[sim.SavedObjectID]uint16)
	for _, rows := range [][]SnapshotSAVObjectBinding{m.Sacks, m.Items, m.Effects, m.Spells} {
		for _, row := range rows {
			out[row.ID] = row.ObjectIndex
		}
	}
	return out
}

func savedObjectRefIDs(ids []sim.SavedObjectID, indices map[sim.SavedObjectID]uint16) ([]uint16, error) {
	var refs []uint16
	for _, id := range ids {
		// Zero explicitly represents an unbound native slot. The paired
		// container coverage prevents this partial graph from claiming export.
		if id == 0 {
			continue
		}
		ref, exists := indices[id]
		if !exists {
			return nil, fmt.Errorf("saved SAV current edge names missing/retired object %d", id)
		}
		if ref == 0 {
			// Native owner validation already excludes a retired target. A live
			// unprojectable Item is omitted only with paired binding/container
			// coverage; its exact ordinal and value remain in the registry.
			continue
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func savedCurrentItemRecord(row sim.SavedItemObject, indices map[sim.SavedObjectID]uint16) (sav.DocumentRecordData, error) {
	class := "Item"
	switch row.Value.SourceEquipment.Class {
	case sim.SourceWeapon:
		class = "Weapon"
	case sim.SourceArmor:
		class = "Armor"
	case sim.SourceShield:
		class = "Shield"
	}
	if class != "Item" {
		// A token minted while the Item was unclassed carries no row.
		row.Token.T0C = equipmentDefinitionRow(row.Value.Code, row.Value.SourceEquipment.DefinitionRow)
	}
	r := savedObjectRecord(class, &row.Token)
	if row.Value.Count > 65535 {
		return r, fmt.Errorf("saved SAV current Item count exceeds original word")
	}
	for _, f := range []sav.DocumentValueData{{Name: "F40", Value: uint32(row.Value.Code)}, {Name: "F42", Value: row.Value.Count}, {Name: "F44", Value: uint32(row.Value.Kind)}, {Name: "F45", Value: uint32(row.F45)}, {Name: "F46", Value: uint32(row.F46)}, {Name: "F47", Value: uint32(row.F47)}, {Name: "F48", Value: uint32(row.F48)}, {Name: "F4A", Value: uint32(uint16(row.Value.Weight))}} {
		savedObjectSetValue(&r, f.Name, f.Value)
	}
	refs, err := savedObjectRefIDs(row.Effects, indices)
	if err != nil {
		return r, err
	}
	savedObjectSetRefs(&r, "Effects", refs, true)
	s := row.Value.SourceEquipment
	switch class {
	case "Weapon":
		r.Raw = append(r.Raw, sav.DocumentRawData{Name: "W52", Bytes: bytes.Clone(s.Attack[:])}, sav.DocumentRawData{Name: "W6A", Bytes: bytes.Clone(s.Defence[:])})
		savedObjectSetValue(&r, "W50", uint32(s.OwnKind))
		ref := uint16(0)
		if row.Spell != 0 {
			ref = indices[row.Spell]
			if ref == 0 {
				return r, fmt.Errorf("saved SAV current Weapon Spell missing")
			}
		}
		savedObjectSetRefs(&r, "WeaponSpell", []uint16{ref}, false)
	case "Armor":
		r.Raw = append(r.Raw, sav.DocumentRawData{Name: "A52", Bytes: bytes.Clone(s.Defence[:])})
		savedObjectSetValue(&r, "A50", uint32(s.OwnKind))
	case "Shield":
		r.Raw = append(r.Raw, sav.DocumentRawData{Name: "S50", Bytes: bytes.Clone(s.Defence[:])})
	}
	slices.SortFunc(r.Raw, func(a, b sav.DocumentRawData) int { return strings.Compare(a.Name, b.Name) })
	return r, nil
}

func savedCurrentEffectRecord(row sim.SavedEffectObject) sav.DocumentRecordData {
	r := savedObjectRecord("Effect", &row.Token)
	for _, f := range []sav.DocumentValueData{{Name: "E3C", Value: uint32(row.Value.Kind)}, {Name: "E3D", Value: uint32(row.Value.Mode)}, {Name: "E40", Value: row.Value.Operand}, {Name: "E0C", Value: uint32(row.E0C)}} {
		savedObjectSetValue(&r, f.Name, f.Value)
	}
	return r
}

func finalizeCurrentTownEffectOwners(doc *sav.DocumentData) {
	keys := make(map[uint32]bool, len(doc.Objects))
	for i := range doc.Objects {
		for _, field := range doc.Objects[i].Values {
			if (field.Name == "Identity" || field.Name == "This") && field.Value != 0 {
				keys[field.Value] = true
			}
		}
	}
	for i := range doc.Objects {
		if doc.Objects[i].Class != "Effect" {
			continue
		}
		for j := range doc.Objects[i].Values {
			field := &doc.Objects[i].Values[j]
			if field.Name == "Reference" && field.Value != 0 && !keys[field.Value] {
				field.Value = 0
			}
		}
	}
}

func savedCurrentSpellRecord(row sim.SavedSpellObject) sav.DocumentRecordData {
	r := savedObjectRecord("Spell", nil)
	for _, f := range []sav.DocumentValueData{{Name: "S08", Value: uint32(row.Value.ID)}, {Name: "S09", Value: uint32(row.Value.Range)}, {Name: "S0A", Value: uint32(row.Value.Defensive)}, {Name: "S0C", Value: uint32(row.Value.ManaCost)}, {Name: "This", Value: row.This}} {
		savedObjectSetValue(&r, f.Name, f.Value)
	}
	return r
}

func savedCurrentSackRecord(row sim.SavedSackObject, c sim.SavedObjectContainer, indices map[sim.SavedObjectID]uint16) (sav.DocumentRecordData, error) {
	r := savedObjectRecord("Sack", &row.Token)
	savedObjectSetValue(&r, "S3C", row.Gold)
	savedObjectSetValue(&r, "Contents1C", c.InsertIndex)
	savedObjectSetValue(&r, "Contents20", uint32(c.Accumulator))
	refs, err := savedObjectRefIDs(c.Items, indices)
	if err != nil {
		return r, err
	}
	savedObjectSetRefs(&r, "Contents", refs, true)
	return r, nil
}

func savedCurrentSackCoverage(row sim.SavedSackObject, c sim.SavedObjectContainer) string {
	if row.Coverage.Unknown&sim.SavedUnknownToken != 0 {
		return savedSackConstructorUnavailable
	}
	if row.Coverage.Unknown != 0 || c.Coverage.Unknown != 0 || row.Coverage.Unsupported != "" || c.Coverage.Unsupported != "" {
		return savedSackContentsUnavailable
	}
	return ""
}

func savedCurrentContainerRecord(r sav.DocumentRecordData, c sim.SavedObjectContainer, indices map[sim.SavedObjectID]uint16) (sav.DocumentRecordData, error) {
	r.Values = slices.Clone(r.Values)
	r.RefSlots = slices.Clone(r.RefSlots)
	r.Counts = slices.Clone(r.Counts)
	if !c.Present {
		savedObjectSetValue(&r, "HasInventory", 0)
		r.Values = slices.DeleteFunc(r.Values, func(v sav.DocumentValueData) bool { return v.Name == "Inventory1C" || v.Name == "Inventory20" })
		r.Counts = slices.DeleteFunc(r.Counts, func(v sav.DocumentCountData) bool { return v.Name == "Inventory" })
		r.RefSlots = slices.DeleteFunc(r.RefSlots, func(v sav.DocumentRefsData) bool { return v.Name == "Inventory" })
		return r, nil
	}
	savedObjectSetValue(&r, "HasInventory", 1)
	savedObjectSetValue(&r, "Inventory1C", c.InsertIndex)
	savedObjectSetValue(&r, "Inventory20", uint32(c.Accumulator))
	refs, err := savedObjectRefIDs(c.Items, indices)
	if err != nil {
		return r, err
	}
	savedObjectSetRefs(&r, "Inventory", refs, true)
	return r, nil
}

func savedCurrentActorItems(state *SnapshotSAVDocument, registry *sim.SavedObjects, indices map[sim.SavedObjectID]uint16) error {
	for _, actor := range state.Actors {
		r := state.Document.Objects[actor.ObjectIndex-1]
		if actor.Retired {
			// Removal retires the native holdings, not the retained actor's
			// other archive fields. Clear only these now-dead ownership edges.
			r.RefSlots, r.Counts = slices.Clone(r.RefSlots), slices.Clone(r.Counts)
			if _, present := savedObjectRefs(&r, "Inventory"); present {
				savedObjectSetRefs(&r, "Inventory", nil, true)
			}
		}
		for _, c := range registry.Containers {
			if c.Owner == (sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: actor.EntityID}) {
				var err error
				r, err = savedCurrentContainerRecord(r, c, indices)
				if err != nil {
					return err
				}
			}
		}
		var worn [sim.EquipSlots]uint16
		for _, item := range registry.ItemRoots {
			if item.Owner.Kind == sim.SavedOwnerActorWorn && item.Owner.Entity == actor.EntityID {
				worn[item.Owner.Slot-1] = indices[item.ID]
			}
		}
		r.RefSlots = slices.Clone(r.RefSlots)
		savedObjectSetRefs(&r, "HeldWeapon", []uint16{worn[0]}, false)
		savedObjectSetRefs(&r, "HeldShield", []uint16{worn[1]}, false)
		if r.Class != "Unit" {
			// Humanoid's twelve armor refs are independent of Unit's held
			// weapon/shield refs. Their first two archive slots remain empty.
			worn[0], worn[1] = 0, 0
			savedObjectSetRefs(&r, "Worn", worn[:], false)
		}
		state.Document.Objects[actor.ObjectIndex-1] = r
	}
	return nil
}

func validateSavedItemBindingWorld(state *SnapshotSAVDocument, world *sim.World) error {
	return validateSavedItemBindingWorldDecoded(state, world, false)
}

// decodedLegacy is confined to UpgradeSaveFormChecked's already-validated
// historical World. Serializing it as a CURRENT World would incorrectly apply
// current area ownership rules before the legacy cell migration can run.
func validateSavedItemBindingWorldDecoded(state *SnapshotSAVDocument, world *sim.World, decodedLegacy bool) error {
	m, r := state.Objects, world.SavedObjects()
	if a, err := readCurrentActions(state.Document); err != nil {
		return err
	} else if a != nil && a.Inventory != nil {
		return validateCurrentObjectBindings(state, world)
	}
	if _, err := cloneSavedObjectBindings(m, state.Document); err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("saved SAV item metadata lacks registry")
	}
	// A current producer may have to materialize concrete equipment operands
	// for an otherwise unbound native Item (for example a weapon held by the
	// generated mission party). The registry keeps that absence explicit so a
	// live-world hash does not change merely because SAV projection ran; the
	// wire record is nevertheless produced from the constructor value. Build
	// the same expected value here before comparing the record.
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
	if !decodedLegacy {
		if _, err := world.MarshalBinary(); err != nil {
			return err
		}
	}
	detached, detachedReasons, err := savedDetachedObjects(world, r)
	if err != nil {
		return err
	}
	if len(r.Sacks) != len(m.Sacks) || len(r.Items) != len(m.Items) || len(r.Effects) != len(m.Effects) || len(r.Spells) != len(m.Spells) || len(r.Containers) != len(m.Containers) {
		return fmt.Errorf("saved SAV object metadata omits native owner")
	}
	indices := savedObjectIndices(m)
	check := func(binding SnapshotSAVObjectBinding, id sim.SavedObjectID, retired bool, coverage string, record sav.DocumentRecordData) error {
		if binding.ID != id || retired != (binding.ObjectIndex == 0) || binding.Unavailable != coverage {
			return fmt.Errorf("saved SAV object %d identity/lifecycle/coverage differs", id)
		}
		if !retired && id != 0 {
			for _, row := range r.Items {
				if row.ID != id || row.Value.SourceEquipment.Class != 0 || (!held[id] && row.Value.WeightPresent) {
					continue
				}
				value := currentItemRecordValue(row.Value.Instance(), held[id], weights, nil)
				row.Value = sim.StackItem(value, row.Value.Count)
				normalized, err := savedCurrentItemRecord(row, indices)
				if err != nil {
					return err
				}
				record = normalized
				break
			}
		}
		if !retired && !reflect.DeepEqual(state.Document.Objects[binding.ObjectIndex-1], record) {
			return fmt.Errorf("saved SAV current object %d differs from native fields/edges", id)
		}
		return nil
	}
	byObject := make(map[uint16]sim.SavedObjectID)
	cellOperands := savedSackCellOperands(world)
	for i, row := range r.Sacks {
		var record sav.DocumentRecordData
		reason := ""
		if !row.Retired {
			c, ok := savedSackContainer(r, row.ID)
			if !ok {
				return fmt.Errorf("saved SAV Sack lacks container")
			}
			var err error
			record, err = savedCurrentSackRecord(row, c, indices)
			if err != nil {
				return err
			}
			reason = savedCurrentSackCoverage(row, c)
			if reason != savedSackConstructorUnavailable && savedCurrentContainerCoverage(c, indices) == savedContainerDetachedUnavailable {
				reason = savedSackContentsUnavailable
			}
			if row.Coverage.Unknown&sim.SavedUnknownIdentity == 0 && !savedSackCellAdmits(cellOperands, binary.LittleEndian.Uint16(row.Token.Position[2:]), row.Token.Identity) {
				return fmt.Errorf("saved SAV Sack lost exact cell identity")
			}
			byObject[m.Sacks[i].ObjectIndex] = row.ID
		}
		if err := check(m.Sacks[i], row.ID, row.Retired, reason, record); err != nil {
			return err
		}
	}
	for i, row := range r.Items {
		retired := row.Retired
		var record sav.DocumentRecordData
		coverage := ""
		if detached[row.ID] {
			coverage = detachedReasons[row.ID]
		} else if !retired {
			var err error
			record, err = savedCurrentItemRecord(row, indices)
			if err != nil {
				return err
			}
			coverage = savedObjectCoverage(row.Coverage)
		}
		if err := check(m.Items[i], row.ID, retired || detached[row.ID], coverage, record); err != nil {
			return err
		}
	}
	for i, row := range r.Effects {
		coverage := ""
		if detached[row.ID] {
			coverage = detachedReasons[row.ID]
		} else if !row.Retired {
			coverage = savedObjectCoverage(row.Coverage)
		}
		if err := check(m.Effects[i], row.ID, row.Retired || detached[row.ID], coverage, savedCurrentEffectRecord(row)); err != nil {
			return err
		}
	}
	for i, row := range r.Spells {
		coverage := ""
		if detached[row.ID] {
			coverage = detachedReasons[row.ID]
		} else if !row.Retired {
			coverage = savedObjectCoverage(row.Coverage)
		}
		if err := check(m.Spells[i], row.ID, row.Retired || detached[row.ID], coverage, savedCurrentSpellRecord(row)); err != nil {
			return err
		}
	}
	for i, c := range r.Containers {
		if m.Containers[i].Owner != c.Owner || m.Containers[i].Unavailable != savedCurrentContainerCoverage(c, indices) {
			return fmt.Errorf("saved SAV container identity/coverage differs")
		}
		if c.Owner.Kind == sim.SavedOwnerActorPack {
			found := false
			for _, e := range world.Entities() {
				if e.ID != c.Owner.Entity {
					continue
				}
				load := e.CurrentActorLoad()
				if load != nil && (load.Inventory.ContainerPresent != c.Present || load.Inventory.InsertIndex != c.InsertIndex || load.Inventory.Accumulator != c.Accumulator) {
					return fmt.Errorf("saved SAV actor %d container differs from native load owner: actor present=%t container=%t index=%d accumulator=%d; registry container=%t index=%d accumulator=%d coverage=%x", e.ID, e.ActorLoad.Present, e.ActorLoad.ContainerPresent, e.ActorLoad.InsertIndex, e.ActorLoad.Accumulator, c.Present, c.InsertIndex, c.Accumulator, c.Coverage.Unknown)
				}
				found = true
			}
			if !found {
				return fmt.Errorf("saved SAV container actor is missing")
			}
		}
	}
	// Compare a detached expected owner graph. Do not repair the incoming DTO.
	copy, err := cloneSavedDocument(state)
	if err != nil {
		return err
	}
	if err := savedCurrentActorItems(copy, r, indices); err != nil {
		return err
	}
	validateEquipment := func(actor SnapshotSAVActor) error {
		items, ok := world.EquippedItems(actor.EntityID)
		if !ok {
			return fmt.Errorf("saved SAV actor %d equipment owner is missing", actor.EntityID)
		}
		record := &state.Document.Objects[actor.ObjectIndex-1]
		checkRef := func(field string, slot int, value sim.ItemInstance) error {
			refs, present := savedObjectRefs(record, field)
			var ref uint16
			if present && slot < len(refs) {
				ref = refs[slot]
			}
			if value.Empty() {
				if ref != 0 {
					return fmt.Errorf("saved SAV actor %d %s slot %d is occupied", actor.EntityID, field, slot)
				}
				return nil
			}
			if value.ObjectID != 0 {
				if ref != indices[value.ObjectID] {
					return fmt.Errorf("saved SAV actor %d %s slot %d identity differs", actor.EntityID, field, slot)
				}
				return nil
			}
			if ref == 0 || int(ref) > len(state.Document.Objects) {
				return fmt.Errorf("saved SAV actor %d %s slot %d lacks current item", actor.EntityID, field, slot)
			}
			code, err := savedStructureValue(&state.Document.Objects[ref-1], "F40")
			if err != nil || uint16(code) != value.Code {
				return fmt.Errorf("saved SAV actor %d %s slot %d value differs", actor.EntityID, field, slot)
			}
			return nil
		}
		if err := checkRef("HeldWeapon", 0, items[0]); err != nil {
			return err
		}
		if err := checkRef("HeldShield", 0, items[1]); err != nil {
			return err
		}
		if state.Document.Objects[actor.ObjectIndex-1].Class != "Unit" {
			for slot := 2; slot < len(items); slot++ {
				value := items[slot]
				if err := checkRef("Worn", slot, value); err != nil {
					return err
				}
			}
		}
		return nil
	}
	stripEquipment := func(record *sav.DocumentRecordData) {
		record.RefSlots = slices.DeleteFunc(slices.Clone(record.RefSlots), func(ref sav.DocumentRefsData) bool {
			return ref.Name == "HeldWeapon" || ref.Name == "HeldShield" || ref.Name == "Worn"
		})
		record.Counts = slices.DeleteFunc(slices.Clone(record.Counts), func(count sav.DocumentCountData) bool { return count.Name == "Worn" })
	}
	for _, actor := range state.Actors {
		actual := state.Document.Objects[actor.ObjectIndex-1]
		expected := copy.Document.Objects[actor.ObjectIndex-1]
		if !actor.Retired {
			if err := validateEquipment(actor); err != nil {
				return err
			}
		}
		stripEquipment(&actual)
		stripEquipment(&expected)
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("saved SAV actor %d item owner/order differs", actor.EntityID)
		}
	}
	var roots []sim.SavedObjectID
	for _, index := range state.Document.World.Sacks {
		if id := byObject[index]; id != 0 {
			roots = append(roots, id)
		}
	}
	if !slices.Equal(roots, r.SackRoots) {
		return fmt.Errorf("saved SAV Sack root aliases/order differ")
	}
	incoming := savedDocumentIncoming(state.Document)
	for _, item := range r.Items {
		if !item.Retired && !detached[item.ID] {
			for _, id := range item.Effects {
				incoming[indices[id]]--
			}
			if item.Spell != 0 {
				incoming[indices[item.Spell]]--
			}
		}
	}
	actors := make(map[sim.EntityID]uint16, len(state.Actors))
	for _, actor := range state.Actors {
		if !actor.Retired {
			actors[actor.EntityID] = actor.ObjectIndex
		}
	}
	for _, root := range r.BookRoots {
		index := actors[root.Entity]
		if index == 0 {
			return fmt.Errorf("saved SAV Book has no exact actor binding")
		}
		refs, present := savedObjectRefs(&state.Document.Objects[index-1], "Spells")
		if !present || len(refs) > len(root.Slots) {
			return fmt.Errorf("saved SAV Book slot presence differs")
		}
		for slot, id := range root.Slots {
			var actual uint16
			if slot < len(refs) {
				actual = refs[slot]
			}
			if actual != indices[id] || id != 0 && actual == 0 {
				return fmt.Errorf("saved SAV Book slot identity differs")
			}
			if id != 0 {
				if incoming[actual] == 0 {
					return fmt.Errorf("saved SAV Book reference count differs")
				}
				incoming[actual]--
			}
		}
	}
	for _, row := range r.Effects {
		if !row.Retired && incoming[indices[row.ID]] != row.ExternalReferences {
			return fmt.Errorf("saved SAV Effect external references differ")
		}
	}
	for _, row := range r.Spells {
		if !row.Retired && incoming[indices[row.ID]] != row.ExternalReferences {
			return fmt.Errorf("saved SAV Spell external references differ")
		}
	}
	return nil
}

// Project all current child and container edges before explicit retirement.
// ITEM-MERGE-129 retires incoming identity; ITEM-EFFSPLIT-074 can add distinct
// child identities. Canonical reindexing and every binding move are one commit.
func projectSavedItemObjects(state *SnapshotSAVDocument, world *sim.World) error {
	next, err := cloneSavedDocument(state)
	if err != nil {
		return err
	}
	r := world.SavedObjects()
	if r == nil {
		return nil
	}
	if _, err := world.MarshalBinary(); err != nil {
		return err
	}
	detached, detachedReasons, err := savedDetachedObjects(world, r)
	if err != nil {
		return err
	}
	var retired []uint16
	prepare := func(rows *[]SnapshotSAVObjectBinding, ids []sim.SavedObjectID, dead []bool, coverage []string) error {
		old := *rows
		if len(old) > len(ids) {
			return fmt.Errorf("saved SAV native object population shrank without retirement")
		}
		for i, id := range ids {
			if i >= len(old) {
				old = append(old, SnapshotSAVObjectBinding{ID: id})
			} else if old[i].ID != id {
				return fmt.Errorf("saved SAV native object identity changed")
			}
			if dead[i] || detached[id] {
				if old[i].ObjectIndex != 0 {
					retired = append(retired, old[i].ObjectIndex)
				}
				old[i].ObjectIndex = 0
				old[i].Unavailable = ""
				if detached[id] {
					old[i].Unavailable = detachedReasons[id]
				}
				continue
			}
			if old[i].ObjectIndex == 0 {
				if i < len(*rows) && !savedDetachedCoverage(old[i].Unavailable) {
					return fmt.Errorf("saved SAV retired object resurrected")
				}
				if len(next.Document.Objects) >= 32767 {
					return fmt.Errorf("saved SAV document object bound exceeded")
				}
				next.Document.Objects = append(next.Document.Objects, sav.DocumentRecordData{})
				old[i].ObjectIndex = uint16(len(next.Document.Objects))
			}
			old[i].Unavailable = coverage[i]
		}
		*rows = old
		return nil
	}
	for group := 0; group < 4; group++ {
		var ids []sim.SavedObjectID
		var dead []bool
		var coverage []string
		var rows *[]SnapshotSAVObjectBinding
		switch group {
		case 0:
			rows = &next.Objects.Items
			for _, row := range r.Items {
				ids = append(ids, row.ID)
				dead = append(dead, row.Retired)
				coverage = append(coverage, savedObjectCoverage(row.Coverage))
			}
		case 1:
			rows = &next.Objects.Effects
			for _, row := range r.Effects {
				ids = append(ids, row.ID)
				dead = append(dead, row.Retired)
				coverage = append(coverage, savedObjectCoverage(row.Coverage))
			}
		case 2:
			rows = &next.Objects.Spells
			for _, row := range r.Spells {
				ids = append(ids, row.ID)
				dead = append(dead, row.Retired)
				coverage = append(coverage, savedObjectCoverage(row.Coverage))
			}
		case 3:
			rows = &next.Objects.Sacks
			for _, row := range r.Sacks {
				ids = append(ids, row.ID)
				dead = append(dead, row.Retired)
				c, _ := savedSackContainer(r, row.ID)
				coverage = append(coverage, savedCurrentSackCoverage(row, c))
			}
		}
		if err := prepare(rows, ids, dead, coverage); err != nil {
			return err
		}
	}
	indices := savedObjectIndices(next.Objects)
	for i, row := range r.Items {
		if !row.Retired && !detached[row.ID] {
			record, err := savedCurrentItemRecord(row, indices)
			if err != nil {
				return err
			}
			next.Document.Objects[next.Objects.Items[i].ObjectIndex-1] = record
		}
	}
	for i, row := range r.Effects {
		if !row.Retired && !detached[row.ID] {
			next.Document.Objects[next.Objects.Effects[i].ObjectIndex-1] = savedCurrentEffectRecord(row)
		}
	}
	for i, row := range r.Spells {
		if !row.Retired && !detached[row.ID] {
			next.Document.Objects[next.Objects.Spells[i].ObjectIndex-1] = savedCurrentSpellRecord(row)
		}
	}
	for i, row := range r.Sacks {
		b := &next.Objects.Sacks[i]
		if b.ID != row.ID {
			return fmt.Errorf("saved SAV Sack identity changed")
		}
		if row.Retired {
			continue
		}
		if b.ObjectIndex == 0 {
			return fmt.Errorf("saved SAV Sack resurrected")
		}
		c, ok := savedSackContainer(r, row.ID)
		if !ok {
			return fmt.Errorf("saved SAV Sack container missing")
		}
		record, err := savedCurrentSackRecord(row, c, indices)
		if err != nil {
			return err
		}
		next.Document.Objects[b.ObjectIndex-1] = record
	}
	if err := savedCurrentActorItems(next, r, indices); err != nil {
		return err
	}
	next.Objects.Containers = nil
	for _, c := range r.Containers {
		next.Objects.Containers = append(next.Objects.Containers, SnapshotSAVContainerBinding{Owner: c.Owner, Unavailable: savedCurrentContainerCoverage(c, indices)})
	}
	next.Document.World.Sacks = slices.DeleteFunc(next.Document.World.Sacks, func(index uint16) bool { return slices.Contains(retired, index) })
	rootCounts := make(map[uint16]int)
	for _, index := range next.Document.World.Sacks {
		rootCounts[index]++
	}
	for _, id := range r.SackRoots {
		index := indices[id]
		if index == 0 {
			return fmt.Errorf("saved SAV native Sack root is missing")
		}
		if rootCounts[index] > 0 {
			rootCounts[index]--
		} else {
			next.Document.World.Sacks = append(next.Document.World.Sacks, index)
		}
	}
	doc, permutation, err := sav.RetireDocumentData(*next.Document, retired)
	if err != nil {
		return fmt.Errorf("current item retirement: %w", err)
	}
	next.Document = &doc
	if err := remapSavedSackDocument(next, permutation); err != nil {
		return err
	}
	if err := validateSavedItemBindingWorld(next, world); err != nil {
		return err
	}
	*state = *next
	return nil
}
