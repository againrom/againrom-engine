package game

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// fieldChoiceAsk is an editor field whose choices are asked.
type fieldChoiceAsk struct {
	kind   string
	index  int
	key    string
	script *alm.Script
}

// fieldChoiceRow serves its kinds (none: every kind) and its field (a key
// prefix when prefix).
type fieldChoiceRow struct {
	kinds  []string
	field  string
	prefix bool
	script bool
	supply func(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice
}

var scriptRecordKinds = []string{"Action", "Condition", "Trigger"}

// fieldChoiceRows is the field-choice registry, asked in order.
var fieldChoiceRows = []fieldChoiceRow{
	{kinds: scriptRecordKinds, field: "value", prefix: true, script: true, supply: scriptParameterChoices},
	{kinds: scriptRecordKinds, field: "opcode", script: true, supply: scriptOperationChoices},
	{kinds: scriptRecordKinds, field: "type", prefix: true, script: true, supply: enumerationChoices(mapload.ScriptParameterNames[:])},
	{kinds: scriptRecordKinds, field: "cmp", prefix: true, script: true, supply: enumerationChoices(mapload.ScriptComparisons[:])},
	{kinds: scriptRecordKinds, field: "once", script: true, supply: fixedChoices(ui.InspectionChoice{Value: 0, Label: "Repeat while true"}, ui.InspectionChoice{Value: 1, Label: "At most once"})},
	{kinds: []string{"Trigger"}, field: "left", prefix: true, script: true, supply: scriptLinkChoices},
	{kinds: []string{"Trigger"}, field: "right", prefix: true, script: true, supply: scriptLinkChoices},
	{kinds: []string{"Trigger"}, field: "action", prefix: true, script: true, supply: scriptLinkChoices},
	{kinds: scriptRecordKinds, prefix: true, script: true, supply: noChoices},
	{field: "npc", supply: npcFieldChoices},
	{kinds: []string{"Enchantment"}, field: "castkind", supply: castKindChoices},
	{field: "owner", supply: ownerChoices},
	{field: "face", supply: faceChoices},
	{field: "sex", supply: wordChoices("editor.unit.look.male", "editor.unit.look.female")},
	{field: "tier", supply: tierFieldChoices},
	{field: "recipeitem", supply: recipeItemChoices},
	{field: "recipe", prefix: true, supply: recipeChoices},
	{field: "item", supply: catalogChoices("item", "")},
	{field: "item", prefix: true, supply: catalogChoices("item", "Remove this item")},
	{field: "additem", supply: catalogChoices("item", "")},
	{field: "addeffect", supply: catalogChoices("effect", "")},
	{field: "effect", prefix: true, supply: catalogChoices("effect", "Remove this effect")},
	{prefix: true, supply: keyCatalogChoices},
}

func (r fieldChoiceRow) serves(a fieldChoiceAsk) bool {
	if r.script && a.script == nil || len(r.kinds) != 0 && !slices.Contains(r.kinds, a.kind) {
		return false
	}
	if r.prefix {
		return strings.HasPrefix(a.key, r.field)
	}
	return a.key == r.field
}

func (s *mapInspectionEdits) fieldChoices(a fieldChoiceAsk) []ui.InspectionChoice {
	for _, r := range fieldChoiceRows {
		if r.serves(a) {
			return r.supply(s, a)
		}
	}
	return nil
}

func (s *mapInspectionEdits) RecordChoices(kind string, index int, key string) []ui.InspectionChoice {
	return s.fieldChoices(fieldChoiceAsk{kind: kind, index: index, key: key})
}

func noChoices(*mapInspectionEdits, fieldChoiceAsk) []ui.InspectionChoice { return nil }

func fixedChoices(rows ...ui.InspectionChoice) func(*mapInspectionEdits, fieldChoiceAsk) []ui.InspectionChoice {
	return func(*mapInspectionEdits, fieldChoiceAsk) []ui.InspectionChoice { return slices.Clone(rows) }
}

func enumerationChoices(names []string) func(*mapInspectionEdits, fieldChoiceAsk) []ui.InspectionChoice {
	return func(*mapInspectionEdits, fieldChoiceAsk) []ui.InspectionChoice {
		var out []ui.InspectionChoice
		for i, name := range names {
			out = append(out, ui.InspectionChoice{Value: int64(i), Label: name})
		}
		return out
	}
}

func wordChoices(words ...string) func(*mapInspectionEdits, fieldChoiceAsk) []ui.InspectionChoice {
	return func(s *mapInspectionEdits, _ fieldChoiceAsk) []ui.InspectionChoice {
		var out []ui.InspectionChoice
		for i, w := range words {
			out = append(out, ui.InspectionChoice{Value: int64(i), Label: s.inspector.inspectionUTF8(s.inspector.word(w))})
		}
		return out
	}
}

func catalogChoices(category, remove string) func(*mapInspectionEdits, fieldChoiceAsk) []ui.InspectionChoice {
	return func(s *mapInspectionEdits, _ fieldChoiceAsk) []ui.InspectionChoice {
		rows := s.inspector.recordCatalog(category)
		if remove != "" {
			rows = append([]ui.InspectionChoice{{Value: 0, Label: remove}}, rows...)
		}
		return rows
	}
}

func keyCatalogChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	return s.inspector.recordCatalog(a.key)
}

func scriptOperationChoices(_ *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	var out []ui.InspectionChoice
	for _, row := range mapload.ScriptOperations(a.kind == "Condition") {
		out = append(out, ui.InspectionChoice{Value: int64(row.Opcode), Label: row.Name})
	}
	return out
}

func scriptLinkChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	out := []ui.InspectionChoice{{Value: 0, Label: "Empty"}}
	action := strings.HasPrefix(a.key, "action")
	nodes := a.script.Conditions
	if action {
		nodes = a.script.Actions
	}
	counts := map[uint32]int{}
	for _, n := range nodes {
		counts[n.ID]++
	}
	for _, n := range nodes {
		if n.ID > 0 && counts[n.ID] == 1 && !(action && n.Opcode >= 0x10000) {
			out = append(out, ui.InspectionChoice{Value: int64(n.ID), Label: fmt.Sprintf("%d: %s", n.ID, s.inspector.inspectionUTF8(n.Label))})
		}
	}
	return out
}

func npcFieldChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	m, err := s.model.Map()
	if err != nil || a.index < 0 || a.index >= len(m.AuthoredUnits) {
		return nil
	}
	out := []ui.InspectionChoice{{Value: -1, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.unit.npc.none"))}}
	for _, choice := range s.inspector.npcChoices(math.MaxUint16, false) {
		u := m.AuthoredUnits[a.index]
		u.Flags |= 1
		u.ClassSubID = uint16(choice.Value)
		r := mapload.Resolve(u, s.inspector.table)
		if r.Arm == mapload.ArmNPC && r.Found() {
			out = append(out, choice)
		}
	}
	return out
}

func castKindChoices(s *mapInspectionEdits, _ fieldChoiceAsk) []ui.InspectionChoice {
	var out []ui.InspectionChoice
	for i := 0; i < 4; i++ {
		out = append(out, ui.InspectionChoice{Value: int64(i), Label: s.inspector.inspectionUTF8(s.inspector.word(castWords[fmt.Sprintf("mode%d", i)].Word))})
	}
	return out
}

func ownerChoices(s *mapInspectionEdits, _ fieldChoiceAsk) []ui.InspectionChoice {
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

func faceChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	out := []ui.InspectionChoice{{Value: 0, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.unit.look.inherit"))}}
	m, err := s.model.Map()
	if err != nil || a.index < 0 || a.index >= len(m.Units) {
		return out
	}
	dir, _, ok := mapload.PlacedFigure(m.Units[a.index], s.inspector.table)
	if !ok {
		return out
	}
	cache := fmt.Sprintf("face:%s", dir)
	if s.inspector.recordOptions == nil {
		s.inspector.recordOptions = map[string][]ui.InspectionChoice{}
	}
	if choices, ok := s.inspector.recordOptions[cache]; ok {
		return append(out, choices...)
	}
	var choices []ui.InspectionChoice
	pictures := map[int64]*image.RGBA{}
	for face := int64(1); face <= 127; face++ {
		doll, _ := composeDoll(s.inspector.archives.artSource(), 0, data.Equipment{}, nil, dir, int(face), false)
		if doll.Figure != nil {
			choices = append(choices, ui.InspectionChoice{Value: face, Label: fmt.Sprintf("%d", face)})
			pictures[face] = doll.Figure
		}
	}
	s.inspector.recordOptions[cache] = choices
	if s.inspector.recordPictures == nil {
		s.inspector.recordPictures = map[string]map[int64]*image.RGBA{}
	}
	s.inspector.recordPictures[cache] = pictures
	return append(out, choices...)
}

func tierFieldChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	m, err := s.model.Map()
	if err != nil || a.index < 0 || a.index >= len(m.AuthoredUnits) {
		return nil
	}
	return tierChoices(s.inspector.tiersOf(m.AuthoredUnits[a.index]))
}

func recipeItemChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	m, err := s.model.Map()
	if err != nil {
		return nil
	}
	loot, err := m.Loot()
	if err != nil || a.index < 0 || a.index >= len(loot.Records) {
		return nil
	}
	var out []ui.InspectionChoice
	for i, item := range loot.Records[a.index].Elements {
		name := mapload.ItemDisplayName(s.inspector.table, data.ItemCode(item.ItemCode()), "", fmt.Sprintf("item %#04x", item.ItemCode()))
		out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf("Item %d: %s", i+1, name)})
	}
	return out
}

func recipeChoices(s *mapInspectionEdits, _ fieldChoiceAsk) []ui.InspectionChoice {
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
