package game

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) RecordChoices(kind string, index int, key string) []ui.InspectionChoice {
	if kind == "Enchantment" && key == "castkind" {
		var out []ui.InspectionChoice
		for i := 0; i < 4; i++ {
			out = append(out, ui.InspectionChoice{Value: int64(i), Label: s.inspector.inspectionUTF8(s.inspector.word(castWords[fmt.Sprintf("mode%d", i)].Word))})
		}
		return out
	}
	if key == "owner" {
		m, err := s.model.Map()
		if err != nil {
			return nil
		}
		var out []ui.InspectionChoice
		for i, p := range m.Groups {
			out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("%d: %s", i+1, s.inspector.inspectionUTF8(alm.PlayerNameText.Decode([]byte(p.Name))))})
		}
		return out
	}
	if key == "tier" {
		m, err := s.model.Map()
		if err != nil || index < 0 || index >= len(m.AuthoredUnits) {
			return nil
		}
		return tierChoices(s.inspector.tiersOf(m.AuthoredUnits[index]))
	}
	if key == "recipeitem" {
		m, err := s.model.Map()
		if err != nil {
			return nil
		}
		loot, err := m.Loot()
		if err != nil || index < 0 || index >= len(loot.Records) {
			return nil
		}
		var out []ui.InspectionChoice
		for i, item := range loot.Records[index].Elements {
			name := mapload.ItemDisplayName(s.inspector.table, data.ItemCode(item.ItemCode()), "", fmt.Sprintf("item %#04x", item.ItemCode()))
			out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("Item %d: %s", i+1, name)})
		}
		return out
	}
	if strings.HasPrefix(key, "recipe") {
		m, err := s.model.Map()
		if err != nil {
			return nil
		}
		out := []ui.InspectionChoice{{Value: 0, Label: "No recipe (plain item)"}}
		for i, recipe := range m.Enchantments {
			if recipe.X == 0 && recipe.Y == 0 {
				out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("Recipe %d: spell%d, %d effects", i+1, uint16(recipe.SpellRaw), len(recipe.Elements))})
			}
		}
		return out
	}
	category := key
	if strings.HasPrefix(key, "item") || key == "additem" {
		category = "item"
	}
	if key == "addeffect" || strings.HasPrefix(key, "effect") {
		category = "effect"
	}
	rows := s.inspector.recordCatalog(category)
	if strings.HasPrefix(key, "item") && key != "item" {
		rows = append([]ui.InspectionChoice{{Value: 0, Label: "Remove this item"}}, rows...)
	}
	if category == "effect" && key != "addeffect" {
		rows = append([]ui.InspectionChoice{{Value: 0, Label: "Remove this effect"}}, rows...)
	}
	return rows
}

func (x *MapInspector) recordCatalog(category string) []ui.InspectionChoice {
	if x.recordOptions == nil {
		x.recordOptions = make(map[string][]ui.InspectionChoice)
	}
	if rows, ok := x.recordOptions[category]; ok {
		return slices.Clone(rows)
	}
	var rows []ui.InspectionChoice
	add := func(value int64, name string) {
		if name == "" {
			name = fmt.Sprintf("raw %s %d", category, value)
		}
		rows = append(rows, ui.InspectionChoice{Value: value, Label: fmt.Sprintf("%d: %s", value, name)})
	}
	switch category {
	case "kind":
		if x.structures != nil {
			for id, c := range x.structures.Classes {
				if id > 0 && id <= math.MaxUint8 && c != nil {
					add(int64(id), c.Name)
				}
			}
		}
	case "class":
		if x.units != nil {
			for id, c := range x.units.Classes {
				if id >= math.MinInt16 && id <= math.MaxInt16 && c != nil {
					if _, err := x.secondaryKey(int16(id)); err == nil {
						add(int64(id), c.Name)
					}
				}
			}
		}
	case "item":
		if x.table != nil {
			for code, name := range x.table.Names {
				if code != 0 {
					add(int64(code), name)
				}
			}
		}
	case "definition":
		add(0, "Use class key")
		if x.table != nil && x.table.Humans != nil {
			seen := map[int64]bool{0: true}
			for i := 1; i < x.table.Humans.Len(); i++ {
				p := x.table.Humans.EntryParams(i)
				if len(p) > 0x18 {
					id := int64(uint32(p[0x18]))
					if !seen[id] {
						add(id, x.table.Humans.EntryName(i))
						seen[id] = true
					}
				}
			}
		}
	case "spell":
		add(0, x.inspectionUTF8(x.word("editor.param.nospell")))
		var names *TextTable
		if x.archives != nil {
			names = LoadTextTable(x.archives.Containers, SpellNamesTextPath, InstallTextCode(x.archives.Containers, x.archives.Game().Edition()))
		}
		if x.table != nil && x.table.Spells != nil {
			for i := 1; i < x.table.Spells.Len() && i <= math.MaxInt16; i++ {
				name := x.table.Spells.EntryName(i)
				if x.archives != nil {
					var ok bool
					name, ok = names.At(i - 1)
					if !ok {
						continue
					}
				}
				add(int64(i), name)
			}
		}
	case "effect", "head":
		if category == "head" {
			add(0, "Raw head A 0")
		}
		if x.table != nil && x.table.Magic != nil {
			for i := 1; i < x.table.Magic.Len() && i <= math.MaxUint8; i++ {
				id, nameID := i, i
				if category == "head" {
					if i <= 43 {
						continue
					}
					id = i - 43
				} else if i == 41 {
					nameID = 49
				}
				if nameID < x.table.Magic.Len() {
					add(int64(id), x.table.Magic.EntryName(nameID))
				}
			}
		}
	default:
		return nil
	}
	slices.SortFunc(rows, func(a, b ui.InspectionChoice) int {
		if a.Value < b.Value {
			return -1
		}
		if a.Value > b.Value {
			return 1
		}
		return 0
	})
	x.recordOptions[category] = slices.Clone(rows)
	return rows
}
