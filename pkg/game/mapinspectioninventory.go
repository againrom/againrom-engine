package game

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func carriedField(key string) (int, string, bool) {
	if key == "carryadd" {
		return 0, "add", true
	}
	for _, part := range []string{"recipe", ""} {
		if text, ok := strings.CutPrefix(key, "carry"+part); ok {
			i, err := strconv.Atoi(text)
			if err == nil && i > 0 {
				return i, part, true
			}
		}
	}
	return 0, "", false
}

// stockItemField parses a sack field that names one item: item# (part ""),
// use# and recipe#.
func stockItemField(key string) (int, string, bool) {
	for _, part := range []string{"item", "use", "recipe"} {
		if text, ok := strings.CutPrefix(key, part); ok {
			i, err := strconv.Atoi(text)
			if err == nil && i > 0 {
				return i, strings.TrimPrefix(part, "item"), true
			}
		}
	}
	return 0, "", false
}

// orderItemFields orders one edit's item fields, which item parses, so that
// no write moves the item a later write names: items from the last, an
// item's recipe before its duplicate and its recipe's fields after both; a
// removed item drops its other fields. A unit's worn slots follow its
// carried items.
func orderItemFields(values []ui.InspectionFieldValue, item func(string) (int, string, bool)) []ui.InspectionFieldValue {
	removed := map[int]bool{}
	removedSlot := map[string]bool{}
	for _, value := range values {
		if element, part, ok := item(value.Key); ok && part == "" && value.Value == 0 {
			removed[element] = true
		}
		if slot, part := equipmentField(value.Key); slot > 0 && part == "" && value.Value <= 0 {
			for _, prefix := range []string{"wearrecipe", "wearmaterial", "weartier"} {
				removedSlot[fmt.Sprintf("%s%d", prefix, slot)] = true
			}
		}
	}
	filtered := values[:0]
	for _, value := range values {
		base, _, _ := strings.Cut(value.Key, ".")
		if element, part, ok := item(base); ok && part != "" && removed[element] || removedSlot[base] {
			continue
		}
		filtered = append(filtered, value)
	}
	values = filtered
	rank := func(v ui.InspectionFieldValue) int {
		base, _, _ := strings.Cut(v.Key, ".")
		if _, _, ok := item(base); ok {
			return 1
		}
		if slot, _ := equipmentField(base); slot > 0 {
			return 2
		}
		return 0
	}
	sort.SliceStable(values, func(i, j int) bool {
		ik, _, ic := strings.Cut(values[i].Key, ".")
		jk, _, jc := strings.Cut(values[j].Key, ".")
		a, b := rank(values[i]), rank(values[j])
		if a != b {
			return a < b
		}
		if a == 1 {
			x, xp, _ := item(ik)
			y, yp, _ := item(jk)
			if x != y {
				return x > y
			}
			priority := func(part string, v int64) int {
				if part == "recipe" {
					return 1
				}
				if part == "" && v == -1 {
					return 2
				}
				return 0
			}
			ap, bp := priority(xp, values[i].Value), priority(yp, values[j].Value)
			if ic {
				ap = 3
			}
			if jc {
				bp = 3
			}
			return ap < bp
		}
		if a == 2 {
			x, xp := equipmentField(ik)
			y, yp := equipmentField(jk)
			if x != y {
				return x < y
			}
			priority := map[string]int{"": 0, "material": 1, "tier": 2, "recipe": 3}
			ap, bp := priority[xp], priority[yp]
			if ic {
				ap = 4
			}
			if jc {
				bp = 4
			}
			return ap < bp
		}
		return false
	})
	return values
}

func (s *mapInspectionEdits) inventoryFields(u *unitTemplateView) []ui.InspectionField {
	stock, err := u.stock()
	if err != nil {
		return nil
	}
	x := s.inspector
	loot, err := u.m.Loot()
	if err != nil {
		return nil
	}
	// The bag's first row opens the unit's sack in the sack's own window;
	// the person window does not show the fields below, which write the
	// same sack.
	sack := ui.InspectionField{Key: "sack", ReadOnly: true, Section: "inventory"}
	if stock >= 0 {
		sack.Label = fmt.Sprintf(x.word("editor.inventory.sack"), len(loot.Records[stock].Elements))
		sack.Link = ui.RecordRef{Kind: ui.KindStock, Index: stock}
	} else {
		sack.Lines = []ui.InspectionLine{{Text: x.inspectionUTF8(x.word("editor.inventory.nosack"))}}
	}
	out := []ui.InspectionField{sack}
	if stock >= 0 {
		ordinal := 0
		for i, item := range loot.Records[stock].Elements {
			if item.Field04 != 0 {
				continue
			}
			ordinal++
			choices := append([]ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.inventory.remove"))}, {Value: -1, Label: x.inspectionUTF8(x.word("editor.inventory.duplicate"))}}, x.recordCatalog("item")...)
			label := fmt.Sprintf(x.word("editor.inventory.item"), ordinal)
			out = append(out, ui.InspectionField{Key: fmt.Sprintf("carry%d", i+1), Label: label, Group: x.word("editor.inventory.added"),
				Value: int64(item.ItemCode()), Choices: choices})
			out = append(out, ui.InspectionField{Key: fmt.Sprintf("carryrecipe%d", i+1), Label: label + " / " + x.word("editor.equipment.magic"), Group: x.word("editor.inventory.added"),
				Parent: fmt.Sprintf("carry%d", i+1), Value: int64(item.TileMarkerIndex), Choices: append([]ui.InspectionChoice{{Value: -1, Label: x.inspectionUTF8(x.word("editor.equipment.newrecipe"))}}, s.RecordChoices(ui.KindStock, stock, "recipe")...)})
			out = append(out, s.itemRecipeFields(fmt.Sprintf("carryrecipe%d", i+1), fmt.Sprintf("carry%d", i+1), "inventory", stock, i+1)...)
		}
	}
	out = append(out, ui.InspectionField{Key: "carryadd", Label: x.word("editor.inventory.add"), Group: x.word("editor.inventory.added"),
		Choices: append([]ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.inventory.choose"))}}, x.recordCatalog("item")...)})
	if sheets := s.unitViewSheets(); u.index < len(sheets) {
		sheet := sheets[u.index]
		lines := []ui.InspectionLine{{Text: fmt.Sprintf(x.inspectionUTF8(x.word("editor.equipment.carried")), x.itemList(sheet.Carried))}}
		lines = append(lines, x.corpseDropLines(sheet)...)
		out = append(out, ui.InspectionField{Key: "holdings", ReadOnly: true, Group: x.word("editor.inventory.after"), Lines: lines})
	}
	return out
}

func (s *mapInspectionEdits) setStockRecipe(m *mapedit.Editor, stock, element int, value int64) (bool, error) {
	view, err := m.Map()
	if err != nil {
		return false, err
	}
	loot, err := view.Loot()
	if err != nil {
		return false, err
	}
	if stock < 0 || stock >= len(loot.Records) || element < 1 || element > len(loot.Records[stock].Elements) {
		return false, fmt.Errorf("item recipe source is unavailable")
	}
	if value == int64(loot.Records[stock].Elements[element-1].TileMarkerIndex) {
		return false, nil
	}
	if value == 0 {
		return s.setField(m, ui.KindStock, mapedit.KindLoot, stock, fmt.Sprintf("recipe%d", element), value, false)
	}
	return s.copyStockRecipe(m, stock, element, value)
}

// copyStockRecipe links element of stock to a recipe of its own: a blank
// one for -1, a copy of recipe value (from 1) otherwise.
func (s *mapInspectionEdits) copyStockRecipe(m *mapedit.Editor, stock, element int, value int64) (bool, error) {
	view, err := m.Map()
	if err != nil {
		return false, err
	}
	if value < -1 || value == 0 || value > int64(len(view.Enchantments)) {
		return false, fmt.Errorf("item recipe is outside the map's native records")
	}
	template := -1
	if value > 0 {
		template = int(value - 1)
	}
	_, err = m.AddItemRecipe(stock, element, template)
	return err == nil, err
}

// duplicateStockItem appends a copy of element of stock as one edit; a
// copied recipe link becomes a recipe of the copy's own, copied from the
// original's. A unit's worn item is not copied: its slot holds one.
func (s *mapInspectionEdits) duplicateStockItem(m *mapedit.Editor, stock, element int) (bool, error) {
	view, err := m.Map()
	if err != nil {
		return false, err
	}
	loot, err := view.Loot()
	if err != nil {
		return false, err
	}
	if stock < 0 || stock >= len(loot.Records) || element < 1 || element > len(loot.Records[stock].Elements) {
		return false, fmt.Errorf("select an item to duplicate")
	}
	item := loot.Records[stock].Elements[element-1]
	if loot.Records[stock].Owner != 0 && item.Field04 != 0 {
		return false, fmt.Errorf("a worn item is not duplicated: its slot holds one item")
	}
	mark := m.Mark()
	if err := m.SetField(mapedit.KindLoot, stock, fmt.Sprintf("duplicate%d", element), 1); err != nil {
		return false, err
	}
	if link := item.TileMarkerIndex; link != 0 {
		if _, err := m.AddItemRecipe(stock, len(loot.Records[stock].Elements)+1, int(link-1)); err != nil {
			return false, err
		}
	}
	m.Squash(mark)
	return true, nil
}

// stockDuplicate is the item choice that duplicates the item.
const stockDuplicate = -1

// setStockItem writes the sack field key that names one item through the
// writers the unit's carried items use: Duplicate, a recipe of the item's
// own (blank, or a copy of recipe N chosen as -(N+1)) and a field of the
// item's recipe, copied first when another item links it. It reports
// false for a field the plain field writer writes.
func (s *mapInspectionEdits) setStockItem(m *mapedit.Editor, stock int, key string, value int64) (bool, bool, error) {
	base, field, sub := strings.Cut(key, ".")
	element, part, ok := stockItemField(base)
	switch {
	case !ok:
		return false, false, nil
	case sub && part == "recipe":
		changed, err := s.setItemRecipeField(m, stock, element, field, value)
		return true, changed, err
	case sub:
		return true, false, fmt.Errorf("unknown item recipe field")
	case part == "" && value == stockDuplicate:
		changed, err := s.duplicateStockItem(m, stock, element)
		return true, changed, err
	case part == "recipe" && value == -1:
		changed, err := s.copyStockRecipe(m, stock, element, -1)
		return true, changed, err
	case part == "recipe" && value < -1:
		template := -value - 1
		if refs, link, err := recipeRefs(m, stock, element); err != nil {
			return true, false, err
		} else if int64(link) == template && refs == 1 {
			return true, false, nil
		}
		changed, err := s.copyStockRecipe(m, stock, element, template)
		return true, changed, err
	}
	return false, false, nil
}

func (s *mapInspectionEdits) setCarried(m *mapedit.Editor, index, element int, part string, value int64) (bool, error) {
	u := s.unitTemplate(mapedit.KindUnit, index, m)
	stock, err := u.stock()
	if err != nil {
		return false, err
	}
	if part == "add" && value == 0 {
		return false, nil
	}
	mark := m.Mark()
	if part == "add" && stock < 0 {
		stock, err = m.AddUnitStock(index)
		if err != nil {
			return false, err
		}
	}
	loot, err := u.m.Loot()
	if err != nil {
		return false, err
	}
	if part != "add" {
		if stock < 0 || element < 1 || element > len(loot.Records[stock].Elements) || loot.Records[stock].Elements[element-1].Field04 != 0 {
			return false, fmt.Errorf("select an authored carried item")
		}
	}
	if part == "recipe" {
		return s.setStockRecipe(m, stock, element, value)
	}
	if part == "add" {
		if value < 1 || value > 65535 {
			return false, fmt.Errorf("carried item code is outside its native word")
		}
		if err := s.inspector.checkItem(value); err != nil {
			return false, err
		}
		if err := s.writeStockItem(m, stock, mapload.ItemInstanceFromCode(uint16(value), s.inspector.table), 0, false); err != nil {
			return false, err
		}
	} else if value == stockDuplicate {
		if _, err := s.duplicateStockItem(m, stock, element); err != nil {
			return false, err
		}
	} else {
		if value < 0 || value > 65535 {
			return false, fmt.Errorf("carried item code is outside its native word")
		}
		if value > 0 {
			if value == int64(loot.Records[stock].Elements[element-1].ItemCode()) {
				return false, nil
			}
			if err := s.inspector.checkItem(value); err != nil {
				return false, err
			}
		}
		if err := m.SetField(mapedit.KindLoot, stock, fmt.Sprintf("item%d", element), value); err != nil {
			return false, err
		}
	}
	m.Squash(mark)
	return true, nil
}
