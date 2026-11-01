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
	fields := s.Fields(ui.KindEnchantment, int(link-1))
	for i := range fields {
		fields[i].Key = prefix + "." + fields[i].Key
		fields[i].Parent, fields[i].Section = parent, section
		fields[i].Group = s.inspector.word("editor.equipment.magic")
	}
	return fields
}

// stockRecipeFields are the fields of the recipe item element of a sack
// links, keyed recipe#.<field> under the item's number, in the sack's
// window beside the item's recipe row.
func (s *mapInspectionEdits) stockRecipeFields(stock, element int) []ui.InspectionField {
	fields := s.itemRecipeFields(fmt.Sprintf("recipe%d", element), fmt.Sprintf("item%d", element), "", stock, element)
	for i := range fields {
		fields[i].Group = ""
		fields[i].Label = fmt.Sprintf(s.inspector.word("editor.plain.item"), element) + " / " + fields[i].Label
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
	return s.setItemRecipeField(m, stock, element, key, value)
}

// recipeRefs counts the items of every sack that link the recipe element
// of stock links, and names that link (from 1, 0 for none).
func recipeRefs(m *mapedit.Editor, stock, element int) (int, uint32, error) {
	view, err := m.Map()
	if err != nil {
		return 0, 0, err
	}
	loot, err := view.Loot()
	if err != nil {
		return 0, 0, err
	}
	if stock < 0 || stock >= len(loot.Records) || element < 1 || element > len(loot.Records[stock].Elements) {
		return 0, 0, fmt.Errorf("select an item's recipe first")
	}
	link := loot.Records[stock].Elements[element-1].TileMarkerIndex
	if link == 0 || link > uint32(len(view.Enchantments)) {
		return 0, link, nil
	}
	refs := 0
	for _, record := range loot.Records {
		for _, item := range record.Elements {
			if item.TileMarkerIndex == link {
				refs++
			}
		}
	}
	return refs, link, nil
}

// setItemRecipeField writes field key of the recipe element of stock links
// as one edit. A recipe more than one item links is copied for this item
// first, so the other items keep their effects.
func (s *mapInspectionEdits) setItemRecipeField(m *mapedit.Editor, stock, element int, key string, value int64) (bool, error) {
	refs, link, err := recipeRefs(m, stock, element)
	if err != nil {
		return false, err
	}
	if refs == 0 {
		return false, fmt.Errorf("select an item's recipe first")
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
	changed, err := s.setField(m, ui.KindEnchantment, mapedit.KindEnchantment, recipe, key, value, false)
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
