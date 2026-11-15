package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// unitGoldField is the unit window's Gold row: the gold word of the unit's
// own loot record, which the unit drops when it dies (DIV-4946).
const unitGoldField = "gold"

// goldField is the Gold row of placement u: false where the map's loot head
// carries no gold or the unit has no loot identity of its own.
func (s *mapInspectionEdits) goldField(u *unitTemplateView) (ui.InspectionField, bool) {
	if u == nil || u.m.FormatVersion < alm.LootWideHeadVersion {
		return ui.InspectionField{}, false
	}
	stock, err := u.model.UnitStock(u.index)
	if err != nil {
		return ui.InspectionField{}, false
	}
	x := s.inspector
	row := ui.InspectionField{Key: unitGoldField, Label: x.word("editor.unit.field.gold"), Meaning: x.word("editor.unit.gold.note")}
	if stock >= 0 {
		v, err := u.model.FieldValue(mapedit.KindLoot, stock, "gold")
		if err != nil {
			return ui.InspectionField{}, false
		}
		row.Value = v
	}
	return row, true
}

// setUnitGold writes value into the gold word of unit index's own loot
// record, adding the record when the unit has none and value is not 0.
func setUnitGold(m *mapedit.Editor, index int, value int64) (bool, error) {
	stock, err := m.UnitStock(index)
	if err != nil {
		return false, err
	}
	if stock < 0 {
		if value == 0 {
			return false, nil
		}
		if stock, err = m.AddUnitStock(index); err != nil {
			return false, err
		}
	} else if v, err := m.FieldValue(mapedit.KindLoot, stock, "gold"); err == nil && v == value {
		return false, nil
	}
	return true, m.SetField(mapedit.KindLoot, stock, "gold", value)
}
