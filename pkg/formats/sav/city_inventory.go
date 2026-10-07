package sav

import (
	"encoding/binary"
	"fmt"
)

// CityItemSale removes Quantity units from one position in the immutable source
// container. Position survives native DTO remapping of archive object indices.
// It never names a newly allocated split object.
type CityItemSale struct {
	Position uint32
	Quantity uint16
}

type CityInventoryItem struct {
	Piece     Piece
	Weight    int16
	Exclusive bool
}

// Inventory returns the original container order, not native display stacks.
// Exclusive means exactly one object reference owns this item and no separate
// identity relation targets it. Ambiguous provenance remains readable, but is
// never eligible for a decrement export.
func (p *CityProvenance) Inventory(identity uint32) ([]CityInventoryItem, error) {
	if p == nil || p.document == nil {
		return nil, fmt.Errorf("sav: missing city inventory document")
	}
	actor := p.document.objects[p.characterSourceIndex[identity]]
	if actor == nil || actor.unit == nil {
		return nil, fmt.Errorf("sav: missing city inventory owner %#x", identity)
	}
	out := make([]CityInventoryItem, len(actor.unit.container))
	for i, object := range actor.unit.container {
		if object == nil || object.item == nil {
			return nil, fmt.Errorf("sav: city inventory position %d is not an item", i)
		}
		v := object.item
		piece := Piece{Class: object.class, Code: binary.LittleEndian.Uint16(v.fields),
			Row: v.token[16], Stack: binary.LittleEndian.Uint16(v.fields[2:]),
			Kind: v.fields[4], Price: int32(binary.LittleEndian.Uint32(v.token[25:])),
			Weight: int16(binary.LittleEndian.Uint16(v.fields[9:]))}
		// The sale comparison must project the same saved equipment operands
		// as the ordinary item reader, not only its appearance code and price.
		switch object.class {
		case "Weapon":
			if len(v.derived) != 47 {
				return nil, fmt.Errorf("sav: city inventory position %d has invalid Weapon fields", i)
			}
			copy(piece.W52[:], v.derived[:24])
			copy(piece.W6A[:], v.derived[24:46])
			piece.W50 = v.derived[46]
			if s := v.weaponExtra; s != nil {
				if s.class != "Spell" || s.spell == nil || len(s.spell.fields) != 9 {
					return nil, fmt.Errorf("sav: city inventory position %d has invalid Weapon Spell", i)
				}
				f := s.spell.fields
				piece.WeaponSpell = &SavedSpell{ID: f[0], Range: f[1], Defensive: f[2], ManaCost: binary.LittleEndian.Uint16(f[3:5]), ArchiveIndex: s.sourceIndex, Key: cityObjectIdentity(s)}
			}
		case "Armor":
			if len(v.derived) != 23 {
				return nil, fmt.Errorf("sav: city inventory position %d has invalid Armor fields", i)
			}
			copy(piece.A52[:], v.derived[:22])
			piece.A50 = v.derived[22]
		case "Shield":
			if len(v.derived) != 22 {
				return nil, fmt.Errorf("sav: city inventory position %d has invalid Shield fields", i)
			}
			copy(piece.S50[:], v.derived)
		}
		for _, effect := range v.effects {
			if effect == nil || effect.effect == nil {
				return nil, fmt.Errorf("sav: city inventory position %d has non-Effect state", i)
			}
			f := effect.effect.fields
			if f[6] != 0 {
				piece.UnsupportedEffectStates = append(piece.UnsupportedEffectStates, f[6])
				continue
			}
			piece.Effects = append(piece.Effects, ItemEffect{Kind: f[0], Mode: f[1], Operand: binary.LittleEndian.Uint32(f[2:])})
		}
		out[i] = CityInventoryItem{Piece: piece, Weight: piece.Weight,
			Exclusive: cityItemExclusive(p.document, object)}
	}
	return out, nil
}

func cityItemExclusive(d *cityDocument, item *cityObject) bool {
	references := 0
	identity := cityObjectIdentity(item)
	count := func(ref *cityObject) {
		if ref == item {
			references++
		}
	}
	for _, o := range d.objects {
		var token []byte
		switch {
		case o.unit != nil:
			u := o.unit
			token = u.token
			count(u.reference74)
			count(u.reference78)
			count(u.reference68)
			for _, list := range [][]*cityObject{u.container, u.equipment, u.effects, u.spells} {
				for _, ref := range list {
					count(ref)
				}
			}
		case o.item != nil:
			token = o.item.token
			count(o.item.weaponExtra)
			for _, ref := range o.item.effects {
				count(ref)
			}
		case o.effect != nil:
			token = o.effect.token
		case o.diary != nil:
			if o.diary.reference == identity {
				return false
			}
		case o.player != nil:
			if o.player.diary.reference == identity || binary.LittleEndian.Uint32(o.player.fixed[43:]) == identity {
				return false
			}
			for _, g := range o.player.groups {
				if g.f44 == identity {
					return false
				}
				for _, ref := range g.actors {
					count(ref)
				}
			}
		}
		if len(token) != 0 && binary.LittleEndian.Uint32(token[33:]) == identity {
			return false
		}
	}
	for _, ref := range d.deadActors {
		count(ref)
	}
	return references == 1
}

// applyCityItemSales preserves each surviving object verbatim except count.
// Load is the stored signed dword minus weight*quantity, not a recomputed sum.
// Arithmetic wraps at the original field width (ITEM-STACK-003).
func applyCityItemSales(d *cityDocument, actor *cityObject, sales []CityItemSale) error {
	if len(sales) == 0 {
		return nil
	}
	if len(sales) > maxCityDataElements || actor.unit.containerFlag != 1 {
		return fmt.Errorf("sav: city sales require a bounded original container")
	}
	u := actor.unit
	counts := make([]uint16, len(u.container))
	changed := make([]bool, len(u.container))
	for i, object := range u.container {
		if object == nil || object.item == nil {
			return fmt.Errorf("sav: invalid city sale container item")
		}
		counts[i] = binary.LittleEndian.Uint16(object.item.fields[2:])
	}
	for _, sale := range sales {
		if sale.Quantity == 0 || uint64(sale.Position) >= uint64(len(counts)) || sale.Quantity > counts[sale.Position] {
			return fmt.Errorf("sav: city sale position/count outside source remainder")
		}
		object := u.container[sale.Position]
		if !cityItemExclusive(d, object) {
			return fmt.Errorf("sav: aliased city sale item")
		}
		counts[sale.Position] -= sale.Quantity
		changed[sale.Position] = true
		weight := int32(int16(binary.LittleEndian.Uint16(object.item.fields[9:])))
		u.containerTails[1] -= uint32(weight * int32(sale.Quantity))
	}
	kept := make([]*cityObject, 0, len(u.container))
	for i, object := range u.container {
		if counts[i] == 0 && changed[i] {
			continue
		}
		binary.LittleEndian.PutUint16(object.item.fields[2:], counts[i])
		kept = append(kept, object)
	}
	u.container = kept
	return nil
}

// Use the archive's actual reference walk to select surviving objects. This
// retains shared children reachable elsewhere and does not invent identities
// for destroyed objects or silently drop dangling numeric identity references
// (remintCityIdentities rejects those afterwards).
func pruneCitySoldObjects(d *cityDocument) error {
	w := newCityArchiveWriter(nil)
	if err := w.referenceList(d.players); err != nil {
		return err
	}
	if err := w.referenceList(d.deadActors); err != nil {
		return err
	}
	d.objects = make(map[uint16]*cityObject, len(w.objects))
	for object, index := range w.objects {
		// Keep the detached source index as the join key for any native
		// continuation carried beside this document. The map key is the new
		// archive index; remapCityCurrentState translates the old key to it
		// after a sale or graph replacement.
		d.objects[index] = object
	}
	return nil
}
