package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (x *MapInspector) equipmentSlotLabel(name string) string {
	switch name {
	case "weapon":
		return x.word("editor.equipment.slot.weapon")
	case "shield":
		return x.word("editor.equipment.slot.shield")
	case "unused":
		return x.word("editor.equipment.slot.unused")
	case "ring":
		return x.word("editor.equipment.slot.ring")
	case "amulet":
		return x.word("editor.equipment.slot.amulet")
	case "head":
		return x.word("editor.equipment.slot.head")
	case "body":
		return x.word("editor.equipment.slot.body")
	case "outer":
		return x.word("editor.equipment.slot.outer")
	case "bracers":
		return x.word("editor.equipment.slot.bracers")
	case "gloves":
		return x.word("editor.equipment.slot.gloves")
	case "legs":
		return x.word("editor.equipment.slot.legs")
	case "boots":
		return x.word("editor.equipment.slot.boots")
	}
	return name
}

func equipmentField(key string) (int, string) {
	for _, part := range []string{"recipe", "material", "tier", ""} {
		if text, ok := strings.CutPrefix(key, "wear"+part); ok {
			slot, _ := strconv.Atoi(text)
			if slot >= 1 && slot <= sim.EquipSlots {
				return slot, part
			}
		}
	}
	return 0, ""
}

func (s *mapInspectionEdits) equipmentFields(u *unitTemplateView) []ui.InspectionField {
	stock, err := u.stock()
	if err != nil {
		return nil
	}
	x, placement := s.inspector, u.m.AuthoredUnits[u.index]
	r := mapload.Resolve(placement, x.table)
	loot, err := u.m.Loot()
	if err != nil {
		return nil
	}
	var elements []alm.LootElement
	if stock >= 0 {
		elements = loot.Records[stock].Elements
	}
	var out []ui.InspectionField
	for _, slot := range mod.EquipmentSlots() {
		value, recipe, elementIndex := int64(-1), int64(0), 0
		if placement.Flags&0x80 != 0 && r.Arm != mapload.ArmNPC {
			value = 0
		}
		for i, element := range elements {
			if target, ok := mapload.EquipTarget(data.ItemCode(element.ItemCode()), x.table); ok && element.Field04 != 0 && target == slot.Number {
				value, recipe = int64(element.ItemCode()), int64(element.TileMarkerIndex)
				elementIndex = i + 1
			}
		}
		choices := []ui.InspectionChoice{{Value: -1, Label: x.inspectionUTF8(x.word("editor.equipment.inherit"))}, {Value: 0, Label: x.inspectionUTF8(x.word("editor.equipment.empty"))}}
		var shut []int64
		if r.Arm == mapload.ArmNPC {
			choices[1].Label, shut = x.inspectionUTF8(x.word("editor.equipment.npcempty")), []int64{0}
		}
		for _, choice := range x.recordCatalog("item") {
			if target, ok := mapload.EquipTarget(data.ItemCode(choice.Value), x.table); ok && target == slot.Number {
				choices = append(choices, choice)
			}
		}
		label := x.equipmentSlotLabel(slot.Name)
		meaning := x.word("editor.equipment.native")
		if sheets := s.unitViewSheets(); u.index < len(sheets) {
			sheet := sheets[u.index]
			meaning = fmt.Sprintf(x.word("editor.equipment.effective"), x.itemList(sheet.Worn[slot.Number-1:slot.Number]))
			if slot.Number == 1 {
				meaning += " / " + fmt.Sprintf(x.word("editor.equipment.carried"), x.itemList(sheet.Carried)) + " / " + x.word("editor.equipment.native")
			}
		}
		field := ui.InspectionField{Key: fmt.Sprintf("wear%d", slot.Number), Label: label, Group: x.word("editor.unit.group.equipment"), Value: value,
			Choices: choices, Shut: shut, Meaning: meaning}
		if (slot.Number == 3 || slot.Number == 11) && len(choices) == 2 && value <= 0 {
			field.ReadOnly = true
			field.Lines = []ui.InspectionLine{{Text: x.inspectionUTF8(label + ": " + x.word("editor.equipment.empty"))}}
		}
		out = append(out, field)
		if value > 0 {
			code := data.ItemCode(value)
			for _, part := range []string{"material", "tier"} {
				var variants []ui.InspectionChoice
				seen := map[int]bool{}
				current := code.A()
				if part == "tier" {
					current = code.C()
				}
				for _, choice := range x.recordCatalog("item") {
					candidate := data.ItemCode(choice.Value)
					v, same := candidate.A(), candidate.C() == code.C()
					if part == "tier" {
						v, same = candidate.C(), candidate.A() == code.A()
					}
					if candidate.B() == code.B() && candidate.D() == code.D() && same && !seen[v] {
						variants, seen[v] = append(variants, ui.InspectionChoice{Value: int64(v), Label: choice.Label}), true
					}
				}
				if len(variants) > 1 {
					word := x.word("editor.equipment.material")
					if part == "tier" {
						word = x.word("editor.equipment.tier")
					}
					out = append(out, ui.InspectionField{Key: fmt.Sprintf("wear%s%d", part, slot.Number), Label: label + " / " + word,
						Group: x.word("editor.unit.group.equipment"), Value: int64(current), Choices: variants})
				}
			}
			out = append(out, ui.InspectionField{Key: fmt.Sprintf("wearrecipe%d", slot.Number), Label: label + " / " + x.word("editor.equipment.magic"),
				Group: x.word("editor.unit.group.equipment"), Value: recipe, Choices: append([]ui.InspectionChoice{{Value: -1, Label: x.inspectionUTF8(x.word("editor.equipment.newrecipe"))}}, s.RecordChoices("Stock", stock, "recipe")...)})
			out = append(out, s.itemRecipeFields(fmt.Sprintf("wearrecipe%d", slot.Number), fmt.Sprintf("wear%d", slot.Number), "equipment", stock, elementIndex)...)
		}
	}
	return out
}

func (s *mapInspectionEdits) setEquipment(m *mapedit.Editor, index, slot int, part string, value int64) (bool, error) {
	templateView := s.unitTemplate(mapedit.KindUnit, index, m)
	stock, err := templateView.stock()
	if err != nil {
		return false, err
	}
	view := templateView.m
	u, t := view.AuthoredUnits[index], s.inspector.table
	r := mapload.Resolve(u, t)
	loot, err := view.Loot()
	if err != nil {
		return false, err
	}
	var matches []int
	if stock >= 0 {
		for i, element := range loot.Records[stock].Elements {
			if target, ok := mapload.EquipTarget(data.ItemCode(element.ItemCode()), t); ok && element.Field04 != 0 && target == slot {
				matches = append(matches, i+1)
			}
		}
	}
	if part == "recipe" {
		if len(matches) == 0 {
			return false, fmt.Errorf("choose an explicit worn item before its recipe")
		}
		return s.setStockRecipe(m, stock, matches[len(matches)-1], value)
	}
	oldRecipe := uint32(0)
	if len(matches) > 0 {
		old := loot.Records[stock].Elements[matches[len(matches)-1]-1]
		oldRecipe = old.TileMarkerIndex
		if part == "material" || part == "tier" {
			code := data.ItemCode(old.ItemCode())
			material, tier := code.A(), code.C()
			if value < 0 || part == "material" && value > 15 || part == "tier" && value > 7 {
				return false, fmt.Errorf("item variant is outside its native field")
			}
			if part == "material" {
				material = int(value)
			} else {
				tier = int(value)
			}
			next := data.ComposeItemCode(material, code.B(), tier, code.D())
			if next != code && t.Names[next] == "" {
				return false, fmt.Errorf("item variant is not in the installed catalog")
			}
			value = int64(next)
		}
	} else if part != "" {
		return false, fmt.Errorf("choose an explicit worn item before its variant")
	}
	if value < -1 || value > 65535 || value == 0 && r.Arm == mapload.ArmNPC {
		return false, fmt.Errorf("%s", s.inspector.word("editor.equipment.npcempty"))
	}
	if value > 0 {
		if target, ok := mapload.EquipTarget(data.ItemCode(value), t); !ok || target != slot {
			return false, fmt.Errorf("item %#04x does not fit slot %d", value, slot)
		}
		if len(matches) > 0 && int64(loot.Records[stock].Elements[matches[len(matches)-1]-1].ItemCode()) == value {
			return false, nil
		}
	}
	if len(matches) == 0 && (value == -1 && (u.Flags&0x80 == 0 || r.Arm == mapload.ArmNPC) || value == 0 && u.Flags&0x80 != 0) {
		return false, nil
	}
	mark := m.Mark()
	for i := len(matches) - 1; i >= 0; i-- {
		if err := m.SetField(mapedit.KindLoot, stock, fmt.Sprintf("item%d", matches[i]), 0); err != nil {
			return false, err
		}
	}
	if stock < 0 {
		stock, err = m.AddUnitStock(index)
		if err != nil {
			return false, err
		}
	}
	if value == 0 && u.Flags&0x80 == 0 {
		worn, carried, _, err := mapload.HumanRowEquipment(t.Humans.EntryStrings(r.Index), t)
		if err != nil {
			return false, err
		}
		for i := len(carried) - 1; i >= 0; i-- {
			if err := s.writeStockItem(m, stock, carried[i], 0, true); err != nil {
				return false, err
			}
		}
		for i := len(worn) - 1; i >= 0; i-- {
			if i+1 != slot && !worn[i].Empty() {
				if err := s.writeStockItem(m, stock, worn[i], 1, true); err != nil {
					return false, err
				}
			}
		}
		if err := m.SetField(mapedit.KindUnit, index, "gear", 1); err != nil {
			return false, err
		}
	}
	if value == -1 && u.Flags&0x80 != 0 && r.Arm != mapload.ArmNPC {
		worn, _, _, err := mapload.HumanRowEquipment(t.Humans.EntryStrings(r.Index), t)
		if err != nil {
			return false, err
		}
		if !worn[slot-1].Empty() {
			if err := s.writeStockItem(m, stock, worn[slot-1], 1, false); err != nil {
				return false, err
			}
		}
	}
	if value > 0 {
		if err := s.writeStockItem(m, stock, mapload.ItemInstanceFromCode(uint16(value), t), 1, false); err != nil {
			return false, err
		}
		if oldRecipe != 0 {
			latest, _ := m.Map()
			items, _ := latest.Loot()
			if err := m.SetField(mapedit.KindLoot, stock, fmt.Sprintf("recipe%d", len(items.Records[stock].Elements)), int64(oldRecipe)); err != nil {
				return false, err
			}
		}
	}
	m.Squash(mark)
	return m.Mark() != mark, nil
}

func (s *mapInspectionEdits) writeStockItem(m *mapedit.Editor, stock int, item sim.ItemInstance, use int64, first bool) error {
	if item.Empty() {
		return nil
	}
	key := "additem"
	if first {
		key = "prependitem"
	}
	if err := m.SetField(mapedit.KindLoot, stock, key, int64(item.Code)); err != nil {
		return err
	}
	view, err := m.Map()
	if err != nil {
		return err
	}
	loot, err := view.Loot()
	if err != nil {
		return err
	}
	element := len(loot.Records[stock].Elements)
	if first {
		element = 1
	}
	if err := m.SetField(mapedit.KindLoot, stock, fmt.Sprintf("use%d", element), use); err != nil {
		return err
	}
	refuse := func() error {
		return fmt.Errorf(s.inspector.word("editor.equipment.recipecannot"), item.Code)
	}
	base := mapload.ItemInstanceFromCode(item.Code, s.inspector.table)
	if len(item.Effects) < len(base.Effects) || !slices.Equal(item.Effects[:len(base.Effects)], base.Effects) {
		return refuse()
	}
	effects := item.Effects[len(base.Effects):]
	if len(effects) > 0 {
		recipe, err := m.AddItemRecipe(stock, element)
		if err != nil {
			return err
		}
		head := slices.IndexFunc(effects, func(effect sim.ItemEffect) bool {
			return effect.Kind == 41 || effect.Kind == 42 && item.Kind == 5
		})
		tail := 0
		for i, effect := range effects {
			if effect.Mode != 0 {
				return refuse()
			}
			if i == 0 && (head == 1 || head < 0 && effect.Kind == 0) {
				if effect.Kind == 41 || effect.Operand > 65535 {
					return refuse()
				}
				a := (int64(effect.Kind) + 256 - 43) % 256
				if a == 0 {
					a = 256
				}
				if err := m.SetFields(mapedit.KindEnchantment, recipe,
					mapedit.FieldSet{Key: "head", Value: a},
					mapedit.FieldSet{Key: "base", Value: int64(uint8(effect.Operand))},
					mapedit.FieldSet{Key: "spread", Value: int64(uint8(effect.Operand >> 8))}); err != nil {
					return err
				}
				continue
			}
			if i == head {
				if i > 1 || effect.Kind == 41 && item.Kind == 5 || uint16(effect.Operand) == 0 {
					return refuse()
				}
				if err := m.SetFields(mapedit.KindEnchantment, recipe,
					mapedit.FieldSet{Key: "spell", Value: int64(uint16(effect.Operand))},
					mapedit.FieldSet{Key: "power", Value: int64(int16(effect.Operand >> 16))}); err != nil {
					return refuse()
				}
				continue
			}
			if effect.Kind == 0 || effect.Kind == 41 {
				return refuse()
			}
			tail++
			if err := m.SetField(mapedit.KindEnchantment, recipe, "addeffect", int64(effect.Kind)); err != nil {
				return err
			}
			if err := m.SetFields(mapedit.KindEnchantment, recipe,
				mapedit.FieldSet{Key: fmt.Sprintf("low%d", tail), Value: int64(uint16(effect.Operand))},
				mapedit.FieldSet{Key: fmt.Sprintf("high%d", tail), Value: int64(effect.Operand >> 16)}); err != nil {
				return err
			}
		}
	}
	view, err = m.Map()
	if err != nil {
		return err
	}
	loot, err = view.Loot()
	if err != nil {
		return err
	}
	loaded, err := mapload.ItemInstanceFromLoot(loot.Records[stock].Elements[element-1], view, s.inspector.table)
	if err != nil || !mapload.ItemRecipeEqual(loaded, item) {
		return refuse()
	}
	return nil
}
