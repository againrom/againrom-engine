package game

import (
	"againrom/pkg/sim"
)

// Both initial-world and acquisition graphs use the same exact item operand
// constructor and current record producers. The document owns temporary keys;
// wire remint runs once after the complete graph and external bindings exist.
func (c *generatedDocumentBuilder) spell(value sim.SourceItemSpell) (uint16, error) {
	return c.append(savedCurrentSpellRecord(sim.SavedSpellObject{Value: value, This: c.identity()}))
}
func (c *generatedDocumentBuilder) item(item sim.ItemInstance, count uint32, owner uint32) (uint16, error) {
	var sequence sim.SavedObjectID
	mint := func() (sim.SavedObjectID, sim.SavedObjectToken, error) {
		sequence++
		token := sim.SavedObjectToken{Identity: c.identity(), RuntimeID: c.runtime(), Reference: owner, T0E: 0x21}
		token.Position[4], token.Position[5] = 128, 128
		return sequence, token, nil
	}
	row, effects, spell, err := sim.ConstructSavedItem(sim.StackItem(item, count), mint)
	if err != nil {
		return 0, err
	}
	if item.SourceEquipment.Class != 0 {
		construction, err := nativeCityItemConstructionFor(item.Code, c.table)
		if err == nil {
			row.F45, row.F46, row.F48 = construction.Shape, construction.Material, construction.F48
		}
	}
	indices := map[sim.SavedObjectID]uint16{}
	for _, effect := range effects {
		index, err := c.append(savedCurrentEffectRecord(effect))
		if err != nil {
			return 0, err
		}
		indices[effect.ID] = index
	}
	if spell != nil {
		index, err := c.append(savedCurrentSpellRecord(*spell))
		if err != nil {
			return 0, err
		}
		indices[spell.ID] = index
	}
	record, err := savedCurrentItemRecord(row, indices)
	if err != nil {
		return 0, err
	}
	return c.append(record)
}
