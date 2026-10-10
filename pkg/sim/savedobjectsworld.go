package sim

import (
	"encoding/binary"
	"fmt"
)

// SavedSackBinding is an exact import request, not a value-based lookup or
// allocation rule. The caller has already joined the source root and cell key.
// Keyless states that the source cell names no Sack: it has no record or an
// empty Sack slot.
type SavedSackBinding struct {
	ID      SavedObjectID
	X, Y    int32
	Gold    uint32
	Keyless bool
}

func (w *World) SavedObjects() *SavedObjects {
	if w == nil {
		return nil
	}
	return w.savedObjects.Clone()
}

func (w *World) SavedSackCellKey(cell uint16) uint32 {
	if w == nil {
		return 0
	}
	return w.savedSackAtCell(cell)
}

// ImportSavedObjects binds exact native owner/ordinal addresses. Values only
// verify that address; they never select or mint an identity. Sack item indices
// count source stacks, each represented by Value.Count adjacent native units.
func (w *World) ImportSavedObjects(registry *SavedObjects, bindings []SavedSackBinding, items ...SavedObjectBinding) error {
	if w == nil || w.savedObjects != nil || registry == nil {
		return fmt.Errorf("sim: saved object import requires an absent registry")
	}
	registry = registry.Clone()
	if registry.Version == 1 {
		if err := registry.MigrateLegacyOwners(); err != nil {
			return err
		}
	}
	if err := registry.ValidateNoInFlight(); err != nil {
		return err
	}
	n := *w
	n.savedObjects = registry.Clone()
	n.sacks = make([]Sack, len(w.sacks))
	for i, s := range w.sacks {
		n.sacks[i] = makeSack(s.X, s.Y, s.Gold, s.ItemInstances)
		n.sacks[i].ObjectID = s.ObjectID
	}
	n.carried = make([][]ItemStack, len(w.carried))
	n.equipment = make([][EquipSlots]ItemInstance, len(w.equipment))
	for i := range w.carried {
		n.carried[i], n.equipment[i] = cloneStacks(w.carried[i]), cloneEquipment(w.equipment[i])
	}
	seen := make(map[SavedObjectID]bool)
	for _, b := range bindings {
		if b.ID == 0 || seen[b.ID] {
			return fmt.Errorf("sim: duplicate/empty saved Sack binding")
		}
		seen[b.ID] = true
		var row *SavedSackObject
		for i := range n.savedObjects.Sacks {
			if n.savedObjects.Sacks[i].ID == b.ID {
				row = &n.savedObjects.Sacks[i]
				break
			}
		}
		if row == nil || row.Retired || row.Gold != b.Gold || row.Token.Identity == 0 || b.X < 0 || b.X > 255 || b.Y < 0 || b.Y > 255 {
			return fmt.Errorf("sim: invalid saved Sack import row")
		}
		cell := uint16(b.X) | uint16(b.Y)<<8
		source := w.motionCell(cell)
		// Collection identity comes from the explicit cell operand. Static
		// bit five controls gameplay lookup visibility, not this exact join.
		// A keyless binding requires a cell with no record or an empty Sack
		// slot (SAV-SACKREMOVE-591 clears a slot without testing its Sack);
		// the caller joined such a Sack by its unique source cell and gold.
		key := row.Token.Identity
		if b.Keyless {
			key = 0
		}
		slot := uint32(0)
		if source != nil {
			slot = binary.LittleEndian.Uint32(source.Payload[16:])
		}
		if binary.LittleEndian.Uint16(row.Token.Position[2:]) != cell || slot != key || source == nil && !b.Keyless {
			return fmt.Errorf("sim: saved Sack cell key differs from its exact source object")
		}
		found := false
		for i := range n.sacks {
			s := &n.sacks[i]
			if s.X != b.X || s.Y != b.Y {
				continue
			}
			if s.ObjectID != 0 || s.Gold != b.Gold {
				return fmt.Errorf("sim: saved Sack import does not name one native Sack")
			}
			s.ObjectID, found = b.ID, true
		}
		if !found {
			return fmt.Errorf("sim: saved Sack import names no native Sack")
		}
	}
	seenItems := make(map[SavedItemLocation]bool)
	for _, b := range items {
		row := n.savedObjects.item(b.ID)
		at := SavedItemLocation{b.Owner, b.Index}
		if row == nil || seenItems[at] || !n.savedObjects.HasLocation(b.ID, at) || !StackStateEqual(row.Value, b.Value) {
			return fmt.Errorf("sim: invalid explicit saved item binding")
		}
		seenItems[at] = true
		bind := func(value *ItemInstance) error {
			if value.ObjectID != 0 {
				return fmt.Errorf("sim: saved item address is already bound")
			}
			expected := StackItem(*value, b.Value.Count)
			expected.ObjectID = b.ID
			if history := expected.NativeRecord; history != nil {
				token := row.Token
				token.T1C = 0
				historyToken := history.Token
				if expected.SourceEquipment.Class != 0 || expected.SourceEquipment.DefinitionRow != 0 {
					historyToken.T0C = expected.SourceEquipment.DefinitionRow
				}
				if historyToken != token || history.F45 != row.F45 || history.F46 != row.F46 || history.F47 != row.F47 || history.F48 != row.F48 {
					return fmt.Errorf("sim: saved item binding loses current native record")
				}
				expected.NativeRecord = nil
			}
			if !StackStateEqual(expected, b.Value) {
				return fmt.Errorf("sim: explicit saved item value differs")
			}
			value.ObjectID = b.ID
			value.NativeRecord = nil
			return nil
		}
		switch b.Owner.Kind {
		case SavedOwnerActorPack:
			ei := indexOfEntity(n.entities, b.Owner.Entity)
			if ei < 0 || uint64(b.Index) >= uint64(len(n.carried[ei])) {
				return fmt.Errorf("sim: saved pack item ordinal is outside owner")
			}
			st := &n.carried[ei][b.Index]
			if st.ObjectID != 0 || st.Count != b.Value.Count {
				return fmt.Errorf("sim: saved pack quantity/address differs")
			}
			value := st.Instance()
			if err := bind(&value); err != nil {
				return err
			}
			*st = StackItem(value, st.Count)
		case SavedOwnerActorWorn:
			ei := indexOfEntity(n.entities, b.Owner.Entity)
			if ei < 0 || b.Owner.Slot == 0 || b.Owner.Slot > EquipSlots || b.Index != 0 {
				return fmt.Errorf("sim: saved worn item address differs")
			}
			if err := bind(&n.equipment[ei][b.Owner.Slot-1]); err != nil {
				return err
			}
		case SavedOwnerSack:
			c := n.savedObjects.container(b.Owner)
			if c == nil || uint64(b.Index) >= uint64(len(c.Items)) || c.Items[b.Index] != b.ID {
				return fmt.Errorf("sim: saved Sack item ordinal differs")
			}
			var start uint64
			for _, id := range c.Items[:b.Index] {
				if id == 0 {
					start++
				} else {
					start += uint64(n.savedObjects.item(id).Value.Count)
				}
			}
			found := false
			for si := range n.sacks {
				if n.sacks[si].ObjectID != b.Owner.Object {
					continue
				}
				values := n.sacks[si].ItemInstances
				end := start + uint64(b.Value.Count)
				if end > uint64(len(values)) {
					return fmt.Errorf("sim: saved Sack quantity run exceeds native values")
				}
				for at := start; at < end; at++ {
					if err := bind(&values[at]); err != nil {
						return err
					}
				}
				found = true
			}
			if !found {
				return fmt.Errorf("sim: saved item names no native Sack")
			}
		default:
			return fmt.Errorf("sim: unsupported imported item owner")
		}
	}
	if err := n.validateSavedObjects(); err != nil {
		return err
	}
	*w = n
	return nil
}

func (w *World) validateSavedObjects() error {
	if err := w.savedObjects.ValidateNoInFlight(); err != nil {
		return err
	}
	if err := w.validateSavedBookRoots(); err != nil {
		return err
	}
	var bindings []SavedObjectBinding
	seenSacks := make(map[SavedObjectID]bool)
	for _, s := range w.sacks {
		if s.ObjectID != 0 {
			if w.savedObjects == nil || seenSacks[s.ObjectID] {
				return fmt.Errorf("sim: Sack identity has no unique registry binding")
			}
			found := false
			for _, row := range w.savedObjects.Sacks {
				if row.ID == s.ObjectID {
					if row.Retired || row.Gold != s.Gold || s.X < 0 || s.X > 255 || s.Y < 0 || s.Y > 255 || binary.LittleEndian.Uint16(row.Token.Position[2:]) != uint16(s.X)|uint16(s.Y)<<8 {
						return fmt.Errorf("sim: saved/native Sack state differs")
					}
					found = true
				}
			}
			if !found {
				return fmt.Errorf("sim: native Sack identity is missing")
			}
			seenSacks[s.ObjectID] = true
		}
		if s.ObjectID == 0 {
			for _, item := range s.ItemInstances {
				if item.ObjectID != 0 {
					return fmt.Errorf("sim: bound Item lacks a Sack root")
				}
			}
			continue
		}
		owner := SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID}
		c := w.savedObjects.container(owner)
		if c == nil {
			return fmt.Errorf("sim: saved Sack lacks its container")
		}
		legacy := len(c.Items) == 0 && c.Coverage.Unknown&SavedUnknownContainerLoad != 0
		for _, item := range s.ItemInstances {
			legacy = legacy && item.ObjectID == 0
		}
		if legacy {
			continue
		}
		at := uint64(0)
		for index, id := range c.Items {
			count := uint64(1)
			if id != 0 {
				count = uint64(w.savedObjects.item(id).Value.Count)
			}
			if count > uint64(len(s.ItemInstances))-at {
				return fmt.Errorf("sim: saved Sack occurrence exceeds units")
			}
			for j := at; j < at+count; j++ {
				item := s.ItemInstances[j]
				if item.ObjectID != id {
					return fmt.Errorf("sim: saved Sack occurrence identity differs")
				}
				if id != 0 && !StackStateEqual(StackItem(item, uint32(count)), w.savedObjects.item(id).Value) {
					return fmt.Errorf("sim: saved Sack occurrence value differs")
				}
			}
			if id != 0 {
				bindings = append(bindings, SavedObjectBinding{ID: id, Owner: owner, Index: uint32(index), Value: StackItem(s.ItemInstances[at], uint32(count))})
			}
			at += count
		}
		if at != uint64(len(s.ItemInstances)) {
			return fmt.Errorf("sim: saved Sack has unmatched native units")
		}

	}
	for i, stocks := range w.carried {
		if w.savedObjects != nil {
			if c := w.savedObjects.container(w.savedPackOwner(i)); c != nil {
				if len(c.Items) != len(stocks) {
					return fmt.Errorf("sim: saved/native pack slot count differs")
				}
				for k, st := range stocks {
					if c.Items[k] != st.ObjectID {
						return fmt.Errorf("sim: saved/native pack slot order differs")
					}
					if st.ObjectID == 0 && !emptyOrderedStack(st) && c.Coverage.Unknown&(SavedUnknownContainerLoad|SavedUnknownMergePolicy) != SavedUnknownContainerLoad|SavedUnknownMergePolicy {
						return fmt.Errorf("sim: unbound saved Pack Item lacks coverage")
					}
				}
				a := w.entities[i].ActorLoad
				if a.Present && (c.Present != a.ContainerPresent || c.InsertIndex != a.InsertIndex || c.Accumulator != a.Accumulator) {
					return fmt.Errorf("sim: saved/native pack bookkeeping differs")
				}
			}
		}
		for k, st := range stocks {
			if st.ObjectID != 0 {
				bindings = append(bindings, SavedObjectBinding{ID: st.ObjectID, Owner: SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: w.entities[i].ID}, Index: uint32(k), Value: st})
			}
		}
	}
	for i, worn := range w.equipment {
		for k, item := range worn {
			if item.ObjectID != 0 {
				bindings = append(bindings, SavedObjectBinding{ID: item.ObjectID, Owner: SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: w.entities[i].ID, Slot: uint32(k + 1)}, Value: w.savedObjects.stackForItem(item)})
			}
		}
	}
	seenSession := make(map[uint64]bool)
	for _, cast := range w.scrollCasts {
		if cast.Item.ObjectID != 0 {
			owner := SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: cast.Reservation}
			if !w.savedObjects.HasRoot(cast.Item.ObjectID, owner) || seenSession[cast.Reservation] {
				return fmt.Errorf("sim: scroll lacks its exact session root")
			}
			bindings = append(bindings, SavedObjectBinding{ID: cast.Item.ObjectID, Owner: owner, Value: w.savedObjects.stackForItem(cast.Item)})
			seenSession[cast.Reservation] = true
		} else if cast.Reservation != 0 {
			return fmt.Errorf("sim: unbound scroll carries reservation")
		}
	}

	if w.savedObjects != nil {
		for _, c := range w.savedObjects.Containers {
			if c.Owner.Kind == SavedOwnerActorPack && indexOfEntity(w.entities, c.Owner.Entity) < 0 {
				return fmt.Errorf("sim: saved container names missing native actor")
			}
		}
		for _, row := range w.savedObjects.Sacks {
			if row.Retired == seenSacks[row.ID] {
				return fmt.Errorf("sim: saved Sack lifecycle differs from the ground population")
			}
		}
		// The World cannot validate a shop/session value it does not own.
		// Game Snapshot must pair these handles with ValidateExternal.
		for _, root := range w.savedObjects.ItemRoots {
			if root.Owner.Kind == SavedOwnerSession && !seenSession[root.Owner.SessionHandle] {
				bindings = append(bindings, SavedObjectBinding{ID: root.ID, Owner: root.Owner, Value: w.savedObjects.item(root.ID).Value})
			}
		}
	}
	return w.savedObjects.CompareLive(bindings)
}

// Native items placed into an imported gold-only Sack remain explicit
// unbound values until their constructor/lifecycle producer is implemented.
func (w *World) noteSavedSackAddition(s Sack, addedItems bool) {
	if s.ObjectID == 0 || w.savedObjects == nil {
		return
	}
	for i := range w.savedObjects.Sacks {
		if w.savedObjects.Sacks[i].ID == s.ObjectID {
			w.savedObjects.Sacks[i].Gold = s.Gold
			if addedItems {
				w.savedObjects.Sacks[i].Coverage.Unknown |= SavedUnknownContainerLoad
			}
		}
	}
	if addedItems {
		if j := w.savedGroundIndex(s.X, s.Y); j >= 0 {
			w.syncSavedSackSlots(j)
			w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerSack, Object: s.ObjectID}).Coverage.Unknown |= SavedUnknownContainerLoad | SavedUnknownMergePolicy
		}
	}
}
