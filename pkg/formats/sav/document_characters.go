package sav

import (
	"fmt"
	"reflect"
)

// DocumentCharacter is an ordinary actor read at an explicit archive binding.
// Reading it neither chooses party membership nor derives character statistics.
type DocumentCharacter struct {
	Character Character
	Face      uint8
}

// ReadDocumentCharacters retains the requested order, aliases and dead actors.
// Keys may still be zero while a current document is being constructed.
func ReadDocumentCharacters(data DocumentData, indices []uint16) ([]DocumentCharacter, error) {
	if len(data.Objects) > maxDocumentDataObjects || len(indices) > maxDocumentDataElements {
		return nil, fmt.Errorf("sav: current character population exceeds bound")
	}
	if err := (&documentDataBudget{}).check(reflect.ValueOf(data), 0); err != nil {
		return nil, err
	}
	objects := make([]*Record, len(data.Objects))
	for i, r := range data.Objects {
		objects[i] = newRecord(r.Class, 0, 0)
		objects[i].Index = uint16(i + 1)
	}
	for i, r := range data.Objects {
		if err := documentRecordFromData(r, objects[i], objects, false, 0); err != nil {
			return nil, fmt.Errorf("sav: character document object %d: %w", i+1, err)
		}
	}
	players, byKey, owner := []*Record{}, map[uint32]*Record{}, map[*Record]*Record{}
	seen := map[*Record]bool{}
	for _, id := range data.Players {
		if id == 0 {
			continue
		}
		if int(id) > len(objects) || objects[id-1].Class != "Player" {
			return nil, fmt.Errorf("sav: invalid current character Player root")
		}
		p := objects[id-1]
		if seen[p] {
			continue
		}
		seen[p] = true
		players = append(players, p)
		if key := p.value("This"); key != 0 {
			if byKey[key] != nil {
				return nil, fmt.Errorf("sav: ambiguous current character Player key")
			}
			byKey[key] = p
		}
		// The last Group membership wins. Each enclosing Player suffix sets
		// its remaining members' owner independently of their Token reference.
		for _, group := range p.Groups {
			for _, member := range group.Refs["Actors"] {
				owner[member] = p
			}
		}
	}
	var out []DocumentCharacter
	for _, id := range indices {
		if id == 0 || int(id) > len(objects) {
			return nil, fmt.Errorf("sav: current character outside document")
		}
		r := objects[id-1]
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			return nil, fmt.Errorf("sav: current character has class %s", r.Class)
		}
		c, err := actorCharacter(r)
		if err != nil {
			return nil, err
		}
		c.Basis, err = actorBasisFields(r)
		if err != nil {
			return nil, err
		}
		p := owner[r]
		if p == nil {
			p = byKey[r.value("Reference")]
		}
		if p != nil {
			c.Basis.Human.HasOwner = true
			c.Basis.Human.ManaReservePercent = p.value("F58")
		}
		for _, p := range players {
			c.Hero = c.Hero || c.Key != 0 && p.value("Hero") == c.Key
		}
		out = append(out, DocumentCharacter{Character: c, Face: uint8(r.value("U4B"))})
	}
	return out, nil
}
