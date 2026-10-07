package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func bindCurrentBookPolicies(doc *sav.DocumentData, state *SnapshotSAVDocument, a *currentActionData) error {
	var indices []uint16
	var actors []sim.EntityID
	for _, actor := range state.Actors {
		policy, present := a.Values[actor.EntityID]
		if actor.Retired || !present || !policy.LegacyBook {
			continue
		}
		indices = append(indices, actor.ObjectIndex)
		actors = append(actors, actor.EntityID)
	}
	if len(indices) == 0 {
		return nil
	}
	rows, err := sav.ReadDocumentCharacters(*doc, indices)
	if err != nil {
		return err
	}
	for i, id := range actors {
		c := rows[i].Character
		anchor := sim.BookValueAnchor(importedSpellbook(c.HasSpellbook, c.Spells), c.KnownSpells())
		policy := a.Values[id]
		policy.LegacyBookWire = &anchor
		a.Values[id] = policy
	}
	return nil
}
