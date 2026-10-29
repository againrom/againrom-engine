package game

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) itemRecipeFields(prefix, parent, section string, stock, element int) []ui.InspectionField {
	view, err := s.model.Map()
	if err != nil {
		return nil
	}
	loot, err := view.Loot()
	if err != nil || stock < 0 || stock >= len(loot.Records) || element < 1 || element > len(loot.Records[stock].Elements) {
		return nil
	}
	link := loot.Records[stock].Elements[element-1].TileMarkerIndex
	if link == 0 || link > uint32(len(view.Enchantments)) {
		return nil
	}
	fields := s.Fields("Enchantment", int(link-1))
	for i := range fields {
		fields[i].Key = prefix + "." + fields[i].Key
		fields[i].Parent, fields[i].Section = parent, section
		fields[i].Group = s.inspector.word("editor.equipment.magic")
	}
	return fields
}

func (s *mapInspectionEdits) setUnitRecipeField(m *mapedit.Editor, index int, prefix, key string, value int64) (bool, error) {
	u := s.unitTemplate(mapedit.KindUnit, index, m)
	stock, err := u.stock()
	if err != nil {
		return false, err
	}
	loot, err := u.m.Loot()
	if err != nil || stock < 0 {
		return false, fmt.Errorf("select an item's recipe first")
	}
	element, part, carried := carriedField(prefix)
	if carried && part != "recipe" {
		return false, fmt.Errorf("unknown item recipe field")
	}
	if !carried {
		slot, part := equipmentField(prefix)
		if slot == 0 || part != "recipe" {
			return false, fmt.Errorf("unknown item recipe field")
		}
		for i, item := range loot.Records[stock].Elements {
			if target, ok := mapload.EquipTarget(data.ItemCode(item.ItemCode()), s.inspector.table); ok && item.Field04 != 0 && target == slot {
				element = i + 1
			}
		}
	}
	if element < 1 || element > len(loot.Records[stock].Elements) || carried && loot.Records[stock].Elements[element-1].Field04 != 0 {
		return false, fmt.Errorf("select an item's recipe first")
	}
	link := loot.Records[stock].Elements[element-1].TileMarkerIndex
	if link == 0 || link > uint32(len(u.m.Enchantments)) {
		return false, fmt.Errorf("select an item's recipe first")
	}
	refs := 0
	for _, record := range loot.Records {
		for _, item := range record.Elements {
			if item.TileMarkerIndex == link {
				refs++
			}
		}
	}
	mark := m.Mark()
	recipe := int(link - 1)
	if old, err := m.FieldValue(mapedit.KindEnchantment, recipe, key); err == nil && old == value {
		return false, nil
	}
	if refs > 1 {
		recipe, err = m.AddItemRecipe(stock, element, recipe)
		if err != nil {
			return false, err
		}
	}
	changed, err := s.setField(m, "Enchantment", mapedit.KindEnchantment, recipe, key, value, false)
	if err != nil {
		return false, err
	}
	m.Squash(mark)
	return changed || refs > 1, nil
}

func unitRecipeField(key string) (string, string, bool) {
	prefix, field, ok := strings.Cut(key, ".")
	return prefix, field, ok && (strings.HasPrefix(prefix, "wearrecipe") || strings.HasPrefix(prefix, "carryrecipe"))
}
