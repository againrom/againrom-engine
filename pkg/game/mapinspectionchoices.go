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
	kind   ui.RecordKind
	index  int
	key    string
	script *alm.Script
}

// fieldChoiceRow serves its kinds (none: every kind; scripted: the script
// node kinds) and its field (a key prefix when prefix).
type fieldChoiceRow struct {
	kinds    []ui.RecordKind
	scripted bool
	field    string
	prefix   bool
	script   bool
	pad      []image.Point
	items    bool
	supply   func(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice
}

// fieldChoiceRows is the field-choice registry, asked in order.
var fieldChoiceRows = []fieldChoiceRow{
	{scripted: true, field: "value", prefix: true, script: true, supply: scriptParameterChoices},
	{scripted: true, field: "opcode", script: true, supply: scriptOperationChoices},
	{scripted: true, field: "type", prefix: true, script: true, supply: enumerationChoices(mapload.ScriptParameterNames[:])},
	{scripted: true, field: "cmp", prefix: true, script: true, supply: enumerationChoices(mapload.ScriptComparisons[:])},
	{scripted: true, field: "once", script: true, supply: fixedChoices(ui.InspectionChoice{Value: 0, Label: "Repeat while true"}, ui.InspectionChoice{Value: 1, Label: "At most once"})},
	{kinds: []ui.RecordKind{ui.KindTrigger}, field: "left", prefix: true, script: true, supply: scriptLinkChoices},
	{kinds: []ui.RecordKind{ui.KindTrigger}, field: "right", prefix: true, script: true, supply: scriptLinkChoices},
	{kinds: []ui.RecordKind{ui.KindTrigger}, field: "action", prefix: true, script: true, supply: scriptLinkChoices},
	{scripted: true, prefix: true, script: true, supply: noChoices},
	{kinds: []ui.RecordKind{ui.KindEnchantment}, field: "castkind", supply: castKindChoices},
	{kinds: []ui.RecordKind{ui.KindUnit}, field: "definition", supply: unitDefinitionChoices},
	{kinds: []ui.RecordKind{ui.KindStructure}, field: "kind", supply: structureKindChoices},
	{field: "owner", supply: ownerChoices},
	{field: "face", supply: faceChoices},
	{field: "sex", supply: wordChoices("editor.unit.look.male", "editor.unit.look.female")},
	{kinds: []ui.RecordKind{ui.KindUnit}, field: "facing", pad: []image.Point{{0, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}, supply: wordChoices("editor.unit.facing.random",
		"editor.unit.facing.1", "editor.unit.facing.2", "editor.unit.facing.3", "editor.unit.facing.4",
		"editor.unit.facing.5", "editor.unit.facing.6", "editor.unit.facing.7", "editor.unit.facing.8")},
	{field: "recipeitem", supply: recipeItemChoices},
	{kinds: []ui.RecordKind{ui.KindSack, ui.KindStock}, field: "recipe", prefix: true, supply: stockRecipeChoices},
	{field: "recipe", prefix: true, supply: recipeChoices},
	{field: "item", items: true, supply: catalogChoices("item", "")},
	{kinds: []ui.RecordKind{ui.KindSack, ui.KindStock}, field: "item", prefix: true, items: true, supply: stockItemChoices},
	{field: "item", prefix: true, items: true, supply: catalogChoices("item", "Remove this item")},
	{field: "additem", items: true, supply: catalogChoices("item", "")},
	{field: "addeffect", supply: catalogChoices("effect", "")},
	{field: "effect", prefix: true, supply: catalogChoices("effect", "Remove this effect")},
	{prefix: true, supply: keyCatalogChoices},
}

func (r fieldChoiceRow) serves(a fieldChoiceAsk) bool {
	if r.script && a.script == nil || r.scripted && !a.kind.Scripted() || len(r.kinds) != 0 && !slices.Contains(r.kinds, a.kind) {
		return false
	}
	if r.prefix {
		return strings.HasPrefix(a.key, r.field)
	}
	return a.key == r.field
}

func (s *mapInspectionEdits) fieldChoiceOffer(a fieldChoiceAsk) ui.InspectionNewStep {
	for _, r := range fieldChoiceRows {
		if r.serves(a) {
			offer := ui.InspectionNewStep{Choices: r.supply(s, a), ChoicePad: r.pad}
			if r.items {
				offer.Groups, offer.Pictures = s.inspector.itemGroups, s.itemPictures
			}
			return offer
		}
	}
	return ui.InspectionNewStep{}
}

func (s *mapInspectionEdits) fieldChoices(a fieldChoiceAsk) []ui.InspectionChoice {
	return s.fieldChoiceOffer(a).Choices
}

func (s *mapInspectionEdits) RecordChoices(kind ui.RecordKind, index int, key string) []ui.InspectionChoice {
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
	for _, row := range mapload.ScriptOperations(a.kind == ui.KindCondition) {
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
	return append(out, s.inspector.figureFaceChoices(dir)...)
}

func (x *MapInspector) figureFaceChoices(dir data.FigureDir) []ui.InspectionChoice {
	cache := fmt.Sprintf("face:%s", dir)
	if x.recordOptions == nil {
		x.recordOptions = map[string][]ui.InspectionChoice{}
	}
	if choices, ok := x.recordOptions[cache]; ok {
		return choices
	}
	var choices []ui.InspectionChoice
	pictures := map[int64]*image.RGBA{}
	for face := int64(1); face <= 127; face++ {
		doll, _ := composeFigure(x.archives.artSource(), figureOptions{Equipment: data.Equipment{}, Fig: figureID{Dir: dir, Face: int(face)}})
		if doll.Figure != nil {
			choices = append(choices, ui.InspectionChoice{Value: face, Label: fmt.Sprintf("%d", face)})
			pictures[face] = doll.Figure
		}
	}
	x.recordOptions[cache] = choices
	if x.recordPictures == nil {
		x.recordPictures = map[string]map[int64]*image.RGBA{}
	}
	x.recordPictures[cache] = pictures
	return choices
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
		name := mapload.ItemDisplayName(s.inspector.table, mapload.ItemSubject{Code: data.ItemCode(item.ItemCode()), Form: mapload.ItemNameEditorList})
		place := s.inspector.inspectionUTF8(fmt.Sprintf(s.inspector.word("editor.plain.item"), i+1))
		out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: place + ": " + name})
	}
	return out
}

// ItemLook is an item code's name, the editor list's form of the item-name
// builder, and the icon the inventory shows for it.
func (s *mapInspectionEdits) ItemLook(code int64) (string, *image.RGBA) {
	x := s.inspector
	if x == nil || code <= 0 || code > math.MaxUint16 {
		return "", nil
	}
	subject := editorItemSubject(x.table, uint16(code))
	name := mapload.ItemDisplayName(x.table, subject)
	if subject.Rows != nil {
		name = x.inspectionUTF8(name)
	}
	var src entrySource
	if art := x.archives.artSource(); art != nil {
		src = art
	}
	if x.recordPictures == nil {
		x.recordPictures = map[string]map[int64]*image.RGBA{}
	}
	if x.recordPictures["item"] == nil {
		x.recordPictures["item"] = map[int64]*image.RGBA{}
	}
	icon, _ := itemIcon(src, x.table, uint16(code), x.recordPictures["item"])
	return name, icon
}

var _ ui.InspectionItemLook = (*mapInspectionEdits)(nil)

func recipeChoices(s *mapInspectionEdits, _ fieldChoiceAsk) []ui.InspectionChoice {
	m, err := s.model.Map()
	if err != nil {
		return nil
	}
	x := s.inspector
	out := []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.inventory.norecipe"))}}
	spells := x.recordCatalog("spell")
	for i, recipe := range m.Enchantments {
		if recipe.X == 0 && recipe.Y == 0 {
			id := int64(uint16(recipe.SpellRaw))
			spell := fmt.Sprintf(x.inspectionUTF8(x.word("editor.inventory.spellnumber")), id)
			for _, c := range spells {
				if c.Value == id {
					spell = strings.TrimPrefix(c.Label, fmt.Sprintf("%d: ", id))
					break
				}
			}
			out = append(out, ui.InspectionChoice{Value: int64(i + 1), Label: fmt.Sprintf(x.inspectionUTF8(x.word("editor.inventory.recipe")), i+1, spell, len(recipe.Elements))})
		}
	}
	return out
}

// stockRecipeChoices are a sack item's recipe choices: none, a new recipe
// of the item's own, a copy of each recipe for the item alone, and each
// recipe shared as it stands, under its own label, which the row shows for
// the item's current link. The unit's own recipe rows ask "recipe" and
// take recipeChoices.
func stockRecipeChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	shared := recipeChoices(s, a)
	if a.key == "recipe" || len(shared) == 0 {
		return shared
	}
	x := s.inspector
	out := []ui.InspectionChoice{shared[0], {Value: -1, Label: x.inspectionUTF8(x.word("editor.equipment.newrecipe"))}}
	for _, c := range shared[1:] {
		out = append(out, ui.InspectionChoice{Value: -c.Value - 1, Label: fmt.Sprintf(x.inspectionUTF8(x.word("editor.inventory.recipecopy")), c.Label)})
	}
	return append(out, shared[1:]...)
}

// stockItemChoices are a sack item's choices: remove it, duplicate it (not
// a unit's worn item, whose slot holds one, and not on a format without item
// recipes), or another item.
func stockItemChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	rows := catalogChoices("item", "Remove this item")(s, a)
	element, part, ok := stockItemField(a.key)
	if !ok || part != "" || len(rows) == 0 {
		return rows
	}
	var cur uint16
	if m, err := s.model.Map(); err == nil {
		if loot, err := m.Loot(); err == nil && a.index >= 0 && a.index < len(loot.Records) && element <= len(loot.Records[a.index].Elements) {
			cur = loot.Records[a.index].Elements[element-1].ItemCode()
			if loot.Records[a.index].Owner != 0 && loot.Records[a.index].Elements[element-1].Field04 != 0 {
				return s.withSecondItemChoice(rows, cur)
			}
		}
	}
	if !s.row().spellSets {
		return s.withSecondItemChoice(rows, cur)
	}
	duplicate := ui.InspectionChoice{Value: stockDuplicate, Label: s.inspector.inspectionUTF8(s.inspector.word("editor.inventory.duplicate"))}
	return append([]ui.InspectionChoice{rows[0], duplicate}, rows[1:]...)
}

// withSecondItemChoice adds the Data.bin row name for a second-game sack item
// code that the shipped item-name table does not list (R2-ENGINE-1513,
// R2-ASSET-319).
func (s *mapInspectionEdits) withSecondItemChoice(rows []ui.InspectionChoice, code uint16) []ui.InspectionChoice {
	x := s.inspector
	subject := editorItemSubject(x.table, code)
	if code == 0 || subject.Rows == nil {
		return rows
	}
	for _, c := range rows {
		if c.Value == int64(code) {
			return rows
		}
	}
	name := x.inspectionUTF8(mapload.ItemDisplayName(x.table, subject))
	return append(rows, ui.InspectionChoice{Value: int64(code), Label: fmt.Sprintf("%d: %s", code, name)})
}

// editorItemSubject is the item the editor names for code: a code the game's
// rows resolve in the table is named by those rows, any other by the editor
// list's form.
func editorItemSubject(t *mapload.Table, code uint16) mapload.ItemSubject {
	s := mapload.ItemSubject{Code: data.ItemCode(code), Form: mapload.ItemNameEditorList}
	if rows := itemRowsOf(t, code); rows.Resolves(t) {
		s.Rows, s.Form = rows, mapload.ItemNamePlayer
	}
	return s
}

// unitDefinitionChoices are a unit's definition choices: on the second game,
// the Units or Humans rows keyed by serverID for the unit's current arm
// (R2-ENGINE-1512, R2-ASSET-316).
func unitDefinitionChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	x := s.inspector
	if x.table == nil || x.table.ClassPlacementFields() {
		return x.recordCatalog("definition")
	}
	m, err := s.model.Map()
	if err != nil || a.index < 0 || a.index >= len(m.Units) {
		return nil
	}
	return x.secondUnitDefinitions(m.Units[a.index].Flags&alm.SecondUnitFlagHuman != 0)
}

func (x *MapInspector) secondUnitDefinitions(human bool) []ui.InspectionChoice {
	key := "second-def-units"
	if human {
		key = "second-def-humans"
	}
	if x.recordOptions == nil {
		x.recordOptions = make(map[string][]ui.InspectionChoice)
	}
	if rows, ok := x.recordOptions[key]; ok {
		return slices.Clone(rows)
	}
	c, serverID := x.table.Units, data.UnitServerID
	if human {
		c, serverID = x.table.Humans, data.HumanServerID
	}
	var rows []ui.InspectionChoice
	seen := map[int64]bool{}
	for i := 1; c != nil && i < c.Len(); i++ {
		name := c.EntryName(i)
		if name == "" {
			continue
		}
		sid, ok := serverID(c.EntryParams(i))
		if !ok {
			continue
		}
		id := int64(uint32(sid))
		if !seen[id] {
			seen[id] = true
			rows = append(rows, ui.InspectionChoice{Value: id, Label: fmt.Sprintf("%d: %s", id, x.inspectionUTF8(name))})
		}
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
	x.recordOptions[key] = slices.Clone(rows)
	return rows
}

// structureKindChoices are a structure's kind choices, keeping a second-game
// extended kind word such as 0x1000021 under its Buildings row name until
// changed (R2-ASSET-317).
func structureKindChoices(s *mapInspectionEdits, a fieldChoiceAsk) []ui.InspectionChoice {
	rows := s.inspector.recordCatalog("kind")
	if !s.inspector.catalogPalettes() || a.index < 0 {
		return rows
	}
	m, err := s.model.Map()
	if err != nil || a.index >= len(m.Objects) {
		return rows
	}
	raw := m.Objects[a.index].Kind
	if raw == uint32(uint8(raw)) {
		return rows
	}
	if row, name, ok := mapload.BuildingName(s.inspector.table, raw); ok {
		rows = append(rows, ui.InspectionChoice{Value: int64(raw), Label: fmt.Sprintf("%d: %s", row, s.inspector.inspectionUTF8(name))})
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
		if x.catalogPalettes() {
			// Every Buildings row, by its own name.
			for row := 1; x.table != nil && row < x.table.Buildings.Len(); row++ {
				if _, name, ok := mapload.BuildingName(x.table, uint32(row)); ok && row <= math.MaxUint8 {
					add(int64(row), x.inspectionUTF8(name))
				}
			}
		} else if x.structures != nil {
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
			for _, code := range mapload.ShippedItemCodes(x.table) {
				if code != 0 {
					add(int64(code), mapload.ItemDisplayName(x.table, mapload.ItemSubject{Code: code, Form: mapload.ItemNameEditorList}))
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
	if category == "item" {
		x.itemGroups = x.itemChoiceGroups(rows)
	}
	return rows
}

// catalogPalettes reports that the install's palettes list every catalog row.
func (x *MapInspector) catalogPalettes() bool {
	return filesOf(x.installGame()).catalogPalettes
}
