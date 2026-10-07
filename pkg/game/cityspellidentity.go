package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func (p *cityObjectProjection) constructSpellRecords(namespace sav.DocumentData) error {
	var missing []sim.SavedObjectID
	for _, id := range p.graph.Spells {
		if _, live := p.spells[id]; live && p.graph.SpellRecords[id].This == 0 {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	keys, err := reserveCurrentCityDocumentKeys(namespace, p.graph, len(missing))
	if err != nil {
		return err
	}
	if p.graph.SpellRecords == nil {
		p.graph.SpellRecords = make(map[sim.SavedObjectID]sim.SavedSpellObject)
	}
	for i, id := range missing {
		row := p.graph.SpellRecords[id]
		row.ID, row.This, row.Value = id, keys[i], p.spells[id]
		p.graph.SpellRecords[id] = row
	}
	return nil
}
