package sav

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// ValidateSpellUpdates checks the alias contract before charging for training.
func (p *CityProvenance) ValidateSpellUpdates(updates []CityCharacterUpdate) error {
	_, err := p.spellUpdates(updates)
	return err
}

func (p *CityProvenance) spellUpdates(updates []CityCharacterUpdate) (map[uint16][5]byte, error) {
	byActor := make(map[uint16][]SavedSpell)
	for _, u := range updates {
		if u.Spells == nil {
			continue
		}
		index, ok := p.characterSourceIndex[u.Identity]
		if !ok {
			return nil, fmt.Errorf("sav: spell update has unknown character %#x", u.Identity)
		}
		if _, duplicate := byActor[index]; duplicate {
			return nil, fmt.Errorf("sav: repeated spell update for %#x", u.Identity)
		}
		descriptor, err := cityCharacterDescriptor(p.document.objects[index], u.Identity, false)
		if err != nil {
			return nil, err
		}
		if len(*u.Spells) != len(descriptor.Spells) {
			return nil, fmt.Errorf("sav: spell update changes membership for %#x", u.Identity)
		}
		for i, s := range *u.Spells {
			want := descriptor.Spells[i]
			if s.Slot != want.Slot || s.ID != want.ID || s.ArchiveIndex != want.ArchiveIndex || s.Key != want.Key {
				return nil, fmt.Errorf("sav: spell update changes slot identity for %#x", u.Identity)
			}
		}
		byActor[index] = *u.Spells
	}
	requests := make(map[uint16][5]byte)
	request := func(spell *cityObject, desired [5]byte) error {
		if before, ok := requests[spell.sourceIndex]; ok && before != desired {
			return fmt.Errorf("sav: conflicting shared Spell %d updates", spell.sourceIndex)
		}
		requests[spell.sourceIndex] = desired
		return nil
	}
	unchanged := func(obj *cityObject) error {
		if obj == nil || obj.spell == nil {
			return nil
		}
		var fields [5]byte
		copy(fields[:], obj.spell.fields)
		return request(obj, fields)
	}
	// Stable order also makes conflict diagnostics independent of map order.
	indices := make([]int, 0, len(p.document.objects))
	for index := range p.document.objects {
		indices = append(indices, int(index))
	}
	sort.Ints(indices)
	for _, index := range indices {
		obj := p.document.objects[uint16(index)]
		if u := obj.unit; u != nil {
			updated, n := byActor[uint16(index)], 0
			for _, spell := range u.spells {
				if spell == nil {
					continue
				}
				if spell.spell == nil || len(spell.spell.fields) != 9 {
					return nil, fmt.Errorf("%w: invalid shared Spell", ErrSpellbook)
				}
				var fields [5]byte
				copy(fields[:], spell.spell.fields)
				if updated != nil {
					s := updated[n]
					fields[1], fields[2] = s.Range, s.Defensive
					binary.LittleEndian.PutUint16(fields[3:], s.ManaCost)
				}
				n++
				if err := request(spell, fields); err != nil {
					return nil, err
				}
			}
			refs := []*cityObject{u.reference74, u.reference78, u.reference68}
			refs = append(refs, u.effects...)
			refs = append(refs, u.container...)
			refs = append(refs, u.equipment...)
			for _, ref := range refs {
				if err := unchanged(ref); err != nil {
					return nil, err
				}
			}
		}
		if item := obj.item; item != nil {
			for _, ref := range append([]*cityObject{item.weaponExtra}, item.effects...) {
				if err := unchanged(ref); err != nil {
					return nil, err
				}
			}
		}
	}
	return requests, nil
}
