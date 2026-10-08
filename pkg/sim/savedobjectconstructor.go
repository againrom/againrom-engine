package sim

import "fmt"

// ConstructSavedItem constructs a new graph from explicit current operands.
// The caller owns transactional ID/key allocation and the final insertion.
// Zero F45/F46/F47/F48 and Effect E0C are native initialization policy, not a
// claim about original allocation. Each Effect occurrence gets its own node.
func ConstructSavedItem(value ItemStack, mint func() (SavedObjectID, SavedObjectToken, error)) (SavedItemObject, []SavedEffectObject, *SavedSpellObject, error) {
	var out SavedItemObject
	if mint == nil || value.ObjectID != 0 || value.Code == 0 || value.Count == 0 || value.Count > 65535 || !value.WeightPresent || len(value.Effects) > 4096 {
		return out, nil, nil, fmt.Errorf("sim: generated Item lacks bounded explicit operands")
	}
	if err := value.Instance().ValidateWeight(); err != nil {
		return out, nil, nil, err
	}
	if value.SourceEquipment.Class > SourceShield {
		return out, nil, nil, fmt.Errorf("sim: generated Item has unsupported concrete class")
	}
	id, token, err := mint()
	if err != nil {
		return out, nil, nil, err
	}
	if id == 0 || token.Identity == 0 {
		return out, nil, nil, fmt.Errorf("sim: generated Item has no allocated identity")
	}
	token.T0C, token.T1C = value.SourceEquipment.DefinitionRow, uint32(value.Price)
	value = value.Clone()
	value.ObjectID = id
	out = SavedItemObject{ID: id, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, InFlight: 1, Value: value, Token: token}
	if history := value.NativeRecord; history != nil {
		out.Token = history.Token
		out.Token.T0C, out.Token.T1C = token.T0C, token.T1C
		out.F45, out.F46, out.F47, out.F48 = history.F45, history.F46, history.F47, history.F48
		out.Value.NativeRecord = nil
	}
	var effects []SavedEffectObject
	for _, effect := range value.Effects {
		childID, childToken, err := mint()
		if err != nil {
			return SavedItemObject{}, nil, nil, err
		}
		out.Effects = append(out.Effects, childID)
		effects = append(effects, SavedEffectObject{ID: childID, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Token: childToken, Value: effect})
	}
	var spell *SavedSpellObject
	if value.SourceEquipment.Spell.Present {
		if value.SourceEquipment.Class != SourceWeapon {
			return SavedItemObject{}, nil, nil, fmt.Errorf("sim: generated owned Spell has no Weapon")
		}
		childID, childToken, err := mint()
		if err != nil {
			return SavedItemObject{}, nil, nil, err
		}
		out.Spell = childID
		spell = &SavedSpellObject{ID: childID, Origin: SavedObjectOrigin{Kind: SavedObjectGenerated}, Value: value.SourceEquipment.Spell, This: childToken.Identity}
	}
	return out, effects, spell, nil
}

// Acquisition owns native identity. Unbound values keep their absent
// provenance even when the SAV exporter constructs their current operands.
func (w *World) constructSavedAcquisition(item ItemStack) (ItemStack, bool) {
	if w.savedObjects == nil || item.ObjectID != 0 {
		return item, false
	}
	value := w.sourceConstructItem(item.Instance())
	if !item.WeightPresent && value.SourceEquipment.Class == 0 && value.Code&0xf00 >= 0x100 && value.Code&0xf00 <= 0xc00 {
		return item, false
	}
	if !value.WeightPresent {
		weight := w.itemWeightOf(value.Code)
		if weight < -32768 || weight > 32767 {
			return item, false
		}
		value.WeightPresent, value.Weight = true, int16(weight)
	}
	n := w.savedObjects.Clone()
	used := make(map[uint32]bool)
	for _, row := range n.Items {
		used[row.Token.Identity], used[row.Token.Reference] = true, true
	}
	for _, row := range n.Effects {
		used[row.Token.Identity], used[row.Token.Reference] = true, true
	}
	for _, row := range n.Spells {
		used[row.This] = true
	}
	for _, row := range n.Sacks {
		used[row.Token.Identity], used[row.Token.Reference] = true, true
	}
	mint := func() (SavedObjectID, SavedObjectToken, error) {
		id, err := n.mint()
		if err != nil {
			return 0, SavedObjectToken{}, err
		}
		// These are temporary native keys. The document writer performs its
		// full live-key reservation and remint before an original SAV exists.
		key := uint32(0x60000000) + uint32(id)*16
		for used[key] && key != 0 {
			key += 16
		}
		if key == 0 {
			return 0, SavedObjectToken{}, fmt.Errorf("sim: generated item key space exhausted")
		}
		used[key] = true
		token := SavedObjectToken{Identity: key, RuntimeID: key >> 4, T0E: 0x21}
		token.Position[4], token.Position[5] = 0x80, 0x80
		return id, token, nil
	}
	row, effects, spell, err := ConstructSavedItem(StackItem(value, item.Count), mint)
	if err != nil {
		return item, false
	}
	n.Items = append(n.Items, row)
	n.Effects = append(n.Effects, effects...)
	if spell != nil {
		n.Spells = append(n.Spells, *spell)
	}
	if err := n.Validate(); err != nil {
		return item, false
	}
	w.savedObjects = n
	return row.Value.Clone(), true
}
