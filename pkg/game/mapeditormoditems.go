package game

import (
	"fmt"
	"image"
	"image/draw"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// The items page's keys: a record's field is "<record>.<id>.<field>", the
// record being an added "item" or "weapon" row by its key or a "change" or
// "material" by its place in the file; the "new." fields add a record from
// the install item chosen.
const (
	itemsNewArmour   = "new.armour"
	itemsNewWeapon   = "new.weapon"
	itemsNewChange   = "new.change"
	itemsNewMaterial = "new.material"
)

// itemSuits are the "for" choices of an added row, by choice value less one.
var itemSuits = []struct {
	suit int32
	word string
}{{mod.SuitAny, "editor.items.for.any"}, {mod.SuitFighter, "editor.items.for.fighter"}, {mod.SuitMage, "editor.items.for.mage"}}

// itemNumberWords are the labels of an added armour row's numbers, which a
// refused number names.
var itemNumberWords = map[string]string{"defence": "editor.items.defence", "absorption": "editor.items.absorption",
	"weight": "editor.items.weight", "price": "editor.items.price"}

// installItem is one row of the install's armour, shield or weapon table as
// the items page offers it: its table, its name as the items file spells it,
// its row, the code class it composes and the code of its first allowed pair,
// and the name the game shows for that code, in UTF-8.
type installItem struct {
	table, name, shown string
	row, class         int
	code               int64
}

// installItems is the one list of the install's item rows the items page
// offers: armour, then shields, then weapons. A name a table holds twice is
// offered once, as the item layer keys its first row; a weapon the game has
// no name for is not offered, and an armour it has none for is its row's name.
func (x *MapInspector) installItems() []installItem {
	t := x.contextBase().table
	if t == nil {
		return nil
	}
	var out []installItem
	for _, c := range []struct {
		table string
		base  data.Collection
		class func([]int32) int
	}{{"armour", t.Armors, data.ArmorShopClass}, {"shield", t.Shields, data.ShieldShopClass}, {"weapon", t.Weapons, data.WeaponShopClass}} {
		if c.base == nil {
			continue
		}
		masks, _ := c.base.(data.MaskTable)
		seen := map[string]bool{}
		for row := 1; row < c.base.Len(); row++ {
			name := c.base.EntryName(row)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			shape, material := 0, 0
			if masks != nil {
				shape, material = firstItemPair(data.ItemMasks(masks, row))
			}
			class := c.class(c.base.EntryParams(row))
			code := data.ComposeItemCode(material, class, shape, row)
			shown, ok := x.installItemName(code)
			if !ok && c.table == "weapon" {
				continue
			}
			if !ok {
				shown = vfs.DecodeText([]byte(name))
			}
			out = append(out, installItem{table: c.table, name: name, shown: shown, row: row, class: class, code: int64(code)})
		}
	}
	return out
}

// installItemName is the name the game shows for an item code, in UTF-8: the
// install's own name, else the one its composer row builds; false for a code
// the game names by no word. The install stores the name in its code page;
// this is where the mod pages decode it, as the engine decodes stored text.
func (x *MapInspector) installItemName(code data.ItemCode) (string, bool) {
	name := mapload.ItemDisplayName(x.table, mapload.ItemSubject{Code: code, Form: mapload.ItemNameWeapon})
	if name == "" {
		return "", false
	}
	return vfs.DecodeText([]byte(name)), true
}

// firstItemPair is the first tier and material masks allow; the first of
// each when they allow none.
func firstItemPair(masks [data.ShopShapes]uint16) (shape, material int) {
	for s, m := range masks {
		for b := range data.ShopMaterials {
			if m&(1<<uint(b)) != 0 {
				return s, b
			}
		}
	}
	return 0, 0
}

// itemSlotName is the items file's name of the slot numbered no; "" for a
// slot a mod item cannot take.
func itemSlotName(no int) string {
	for n, v := range mod.ItemSlots {
		if v == no {
			return n
		}
	}
	return ""
}

// anyItem, armourBase, weaponBase and standInFor are the filters of the
// install items each choice offers: any row, an armour or shield a mod
// item can be worn as, a weapon, and an armour or shield worn in slot.
func anyItem(installItem) bool { return true }

func armourBase(it installItem) bool { return it.table != "weapon" && itemSlotName(it.class) != "" }

func weaponBase(it installItem) bool { return it.table == "weapon" }

func standInFor(slot int) func(installItem) bool {
	return func(it installItem) bool { return it.table != "weapon" && it.class == slot }
}

// installFind is the install item named name; false for none.
func installFind(items []installItem, name string) (installItem, bool) {
	i := slices.IndexFunc(items, func(it installItem) bool { return it.name == name })
	if i < 0 {
		return installItem{}, false
	}
	return items[i], true
}

// itemChoice is a choice field of the install items keep admits, by their
// game names and pictures, then of the weapon rows the mod adds in own; an
// install item's value is its place in items counted from 1, an own row's
// follows them. The record's item named current is the value; prompt
// offers a first choice that picks none.
func (s *mapInspectionEdits) itemChoice(key, label, help, current string, items []installItem, keep func(installItem) bool, own []mod.WeaponRow, prompt bool) ui.InspectionField {
	x := s.inspector
	f := ui.InspectionField{Key: key, Label: label, Meaning: help, Icons: map[int64]*image.RGBA{}}
	if prompt {
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: 0, Label: x.inspectionUTF8(x.word("editor.items.pick"))})
	}
	for i, it := range items {
		if !keep(it) {
			continue
		}
		v := int64(i + 1)
		_, icon := s.ItemLook(it.code)
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: v, Label: x.inspectionUTF8(it.shown)})
		if icon != nil {
			f.Icons[v] = icon
		}
		if it.name == current {
			f.Value = v
		}
	}
	for j, w := range own {
		v := int64(len(items) + 1 + j)
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: v, Label: x.inspectionUTF8(itemPlaceName(w.Name, w.Key))})
		if w.Key == current {
			f.Value = v
		}
	}
	return f
}

// pickedItem is the name the items file spells for the choice value of an
// itemChoice over the same items, keep and own; false for a value it does
// not offer.
func pickedItem(value string, items []installItem, keep func(installItem) bool, own []mod.WeaponRow) (installItem, string, bool) {
	v, err := strconv.Atoi(value)
	switch {
	case err != nil || v < 1:
		return installItem{}, "", false
	case v <= len(items) && keep(items[v-1]):
		return items[v-1], items[v-1].name, true
	case v > len(items) && v-len(items) <= len(own):
		return installItem{}, own[v-len(items)-1].Key, true
	}
	return installItem{}, "", false
}

// itemNames is the items page's rowText: a name key the project's strings
// lack reads as "?"; the mod's context refuses it in the reader's words.
func (s *mapInspectionEdits) itemNames(d *editorModDocument) (func(key string) (string, bool), error) {
	look, err := d.textLookup(s.inspector.pageLanguage())
	if err != nil {
		return nil, err
	}
	return func(key string) (string, bool) {
		if v, ok := look(key); ok && v != "" && utf8.RuneCountInString(v) <= 40 {
			return v, true
		}
		return "?", true
	}, nil
}

// itemsPicture is the picture the items page shows for a record: the
// draft's picture at rel, else the game item's picture by code; nil for
// none.
func (s *mapInspectionEdits) itemsPicture(rel string, code int64) *image.RGBA {
	if rel != "" {
		d := s.workspace.Project
		if d == nil {
			d = s.inspector.mod.document()
		}
		pic, err := readItemPicture(d.Files[rel])
		if err != nil {
			return nil
		}
		out := image.NewRGBA(image.Rect(0, 0, pic.Bounds().Dx(), pic.Bounds().Dy()))
		draw.Draw(out, out.Bounds(), pic, pic.Bounds().Min, draw.Src)
		return out
	}
	if code > 0 {
		_, icon := s.ItemLook(code)
		return icon
	}
	return nil
}

// rowOff reports that a change of items switches the row key off.
func rowOff(items mod.ItemData, key string) bool {
	off := false
	for _, c := range items.Changes {
		if c.Item == key && c.Off != nil {
			off = *c.Off
		}
	}
	return off
}

// ownItem reports that key names an armour or weapon row the mod adds.
func ownItem(items mod.ItemData, key string) bool {
	return slices.ContainsFunc(items.Rows, func(r mod.ItemRow) bool { return r.Key == key }) ||
		slices.ContainsFunc(items.Weapons, func(w mod.WeaponRow) bool { return w.Key == key })
}

// itemThings is the number of things the items page lists: each added row,
// each change of a game item and each material record. A change of an added
// row is that row's switch, not a thing of its own.
func itemThings(items mod.ItemData) int {
	n := len(items.Rows) + len(items.Weapons) + len(items.Materials)
	for _, c := range items.Changes {
		if !ownItem(items, c.Item) {
			n++
		}
	}
	return n
}

// itemsFields is W's items page: the choices that add a record, then each
// row the mod adds, each change of a game item and each material record.
// Every item is chosen by its game name and picture, every picture through
// the file choice; names are typed per language and numbers as numbers.
func (s *mapInspectionEdits) itemsFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	line := func(key, text string) ui.InspectionField {
		return ui.InspectionField{Key: key, ReadOnly: true, Text: true, TextValue: text, Lines: []ui.InspectionLine{{Text: x.inspectionUTF8(text)}}}
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	read, err := s.heldRows(project, s.itemNames, itemFiles()...)
	if err != nil {
		return []ui.InspectionField{line("error", err.Error())}
	}
	items, sets := read.Items, read.Strings
	install := x.installItems()
	out := []ui.InspectionField{
		s.itemChoice(itemsNewArmour, x.word("editor.items.new.armour"), x.word("editor.items.new.armour.help"), "", install, armourBase, nil, true),
		s.itemChoice(itemsNewWeapon, x.word("editor.items.new.weapon"), x.word("editor.items.new.weapon.help"), "", install, weaponBase, nil, true),
		s.itemChoice(itemsNewChange, x.word("editor.items.new.change"), x.word("editor.items.new.change.help"), "", install, anyItem, nil, true),
		s.itemChoice(itemsNewMaterial, x.word("editor.items.new.material"), x.word("editor.items.new.material.help"), "", install, anyItem, items.Weapons, true),
	}
	heads := map[string]bool{}
	group := func(head string, fields ...ui.InspectionField) {
		// Two records may carry one name (two armours from one game item); a
		// group is one record, so a repeated heading takes its number.
		for n := 2; heads[head]; n++ {
			head = fmt.Sprintf("%s (%d)", head, n)
		}
		heads[head] = true
		for _, f := range fields {
			f.Group = head
			out = append(out, f)
		}
	}
	choice := func(key, label, help string, value int64, words ...string) ui.InspectionField {
		f := ui.InspectionField{Key: key, Label: x.word(label), Meaning: x.word(help), Value: value}
		for i, w := range words {
			f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(x.word(w))})
		}
		return f
	}
	num := func(key, label, help string, v int32) ui.InspectionField {
		f := workspaceText(key, x.word(label), strconv.Itoa(int(v)))
		f.Meaning = x.word(help)
		return f
	}
	names := func(k, nameKey string) []ui.InspectionField {
		var fs []ui.InspectionField
		for _, l := range textLanguages() {
			f := workspaceText(k+"name."+l, fmt.Sprintf(x.word("editor.items.name"), languageLabel(l)), sets[l][nameKey])
			f.Meaning = x.word("editor.items.name.help")
			fs = append(fs, f)
		}
		return fs
	}
	sprite := func(key, rel string, code int64) []ui.InspectionField {
		f := workspaceText(key+"sprite", x.word("editor.items.sprite"), rel)
		f.Meaning, f.Pick = x.word("editor.items.sprite.help"), []string{"png"}
		fs := []ui.InspectionField{f}
		if pic := s.itemsPicture(rel, code); pic != nil {
			fs = append(fs, ui.InspectionField{Key: key + "look", ReadOnly: true, Picture: pic})
		}
		return fs
	}
	switchRow := func(key string, off bool) []ui.InspectionField {
		label := x.word("editor.items.off")
		var fs []ui.InspectionField
		if off {
			label = x.word("editor.items.on")
			fs = append(fs, line(key+"state", x.word("editor.items.isoff")))
		}
		return append(fs, ui.InspectionField{Key: key + "off", Label: label, Action: true, Column: 1})
	}
	remove := func(key string) ui.InspectionField {
		return ui.InspectionField{Key: key + "remove", Label: x.word("editor.items.remove"), Action: true, Column: 1}
	}
	codeOf := func(name string) int64 {
		if it, ok := installFind(install, name); ok {
			return it.code
		}
		return 0
	}
	slots := mod.ItemSlotNames()
	for _, r := range items.Rows {
		k := "item." + r.Key + "."
		slot := ui.InspectionField{Key: k + "slot", Label: x.word("editor.items.slot"), Meaning: x.word("editor.items.slot.help"), Value: int64(slices.Index(slots, r.Slot) + 1)}
		for i, n := range slots {
			slot.Choices = append(slot.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(x.equipmentSlotLabel(n))})
		}
		suit := int64(slices.IndexFunc(itemSuits, func(v struct {
			suit int32
			word string
		}) bool {
			return v.suit == r.Suit
		}) + 1)
		shop := int64(1)
		if r.Stock {
			shop = 2
		}
		fields := append(names(k, r.NameKey), slot,
			s.itemChoice(k+"stand-in", x.word("editor.items.standin"), x.word("editor.items.standin.help"), r.StandIn, install, standInFor(r.SlotNo), nil, false),
			choice(k+"for", "editor.items.for", "editor.items.for.help", suit, "editor.items.for.any", "editor.items.for.fighter", "editor.items.for.mage"),
			choice(k+"shop", "editor.items.shop", "editor.items.shop.help", shop, "editor.items.no", "editor.items.yes"),
			num(k+"defence", "editor.items.defence", "editor.items.defence.help", r.Defence),
			num(k+"absorption", "editor.items.absorption", "editor.items.absorption.help", r.Absorption),
			num(k+"weight", "editor.items.weight", "editor.items.weight.help", r.Weight),
			num(k+"price", "editor.items.price", "editor.items.price.help", r.Price))
		fields = append(fields, sprite(k, r.Sprite, codeOf(r.StandIn))...)
		fields = append(fields, switchRow(k, rowOff(items, r.Key))...)
		group(fmt.Sprintf(x.word("editor.items.armour"), itemPlaceName(r.Name, r.Key)), append(fields, remove(k))...)
	}
	for _, w := range items.Weapons {
		k := "weapon." + w.Key + "."
		fields := append(names(k, w.NameKey), s.itemChoice(k+"base", x.word("editor.items.base"), x.word("editor.items.base.help"), w.Base, install, weaponBase, nil, false))
		for _, wp := range mod.WeaponParams {
			value := ""
			if i := slices.IndexFunc(w.Params, func(v mod.WeaponParamValue) bool { return v.Key == wp.Key }); i >= 0 {
				value = strconv.Itoa(int(w.Params[i].Value))
			}
			f := workspaceText(k+wp.Key, x.word("editor.items.param."+wp.Key), value)
			f.Meaning = x.word("editor.items.param." + wp.Key + ".help")
			fields = append(fields, f)
		}
		fields = append(fields, sprite(k, w.Sprite, codeOf(w.Base))...)
		fields = append(fields, switchRow(k, rowOff(items, w.Key))...)
		group(fmt.Sprintf(x.word("editor.items.weapon"), itemPlaceName(w.Name, w.Key)), append(fields, remove(k))...)
	}
	for i, c := range items.Changes {
		if ownItem(items, c.Item) {
			continue
		}
		k := "change." + strconv.Itoa(i) + "."
		fields := []ui.InspectionField{s.itemChoice(k+"item", x.word("editor.items.pair.item"), x.word("editor.items.new.change.help"), c.Item, install, anyItem, nil, false)}
		it, _ := installFind(install, c.Item)
		for _, n := range []struct {
			field string
			v     *int32
		}{{"defence", c.Defence}, {"absorption", c.Absorption}, {"weight", c.Weight}, {"price", c.Price}} {
			if it.table == "weapon" && (n.field == "defence" || n.field == "absorption") {
				continue
			}
			value := ""
			if n.v != nil {
				value = strconv.Itoa(int(*n.v))
			}
			f := workspaceText(k+n.field, x.word(itemNumberWords[n.field]), value)
			f.Meaning = x.word(itemNumberWords[n.field]+".help") + " " + x.word("editor.items.unchanged")
			fields = append(fields, f)
		}
		fields = append(fields, switchRow(k, c.Off != nil && *c.Off)...)
		name := c.Item
		if it.name != "" {
			name = it.shown
		}
		group(fmt.Sprintf(x.word("editor.items.change"), name), append(fields, remove(k))...)
	}
	t := x.contextBase().table
	for i, m := range items.Materials {
		k := "material." + strconv.Itoa(i) + "."
		allow := int64(2)
		if m.Allow {
			allow = 1
		}
		fields := []ui.InspectionField{s.itemChoice(k+"item", x.word("editor.items.pair.item"), x.word("editor.items.pair.item.help"), m.Item, install, anyItem, items.Weapons, false)}
		shape, material := -1, -1
		if t != nil {
			var f ui.InspectionField
			f, shape = s.scaleChoice(k+"tier", x.word("editor.items.pair.tier"), x.word("editor.items.pair.tier.help"), "tier", t.Shapes, data.ShopShapes, m.Tier)
			fields = append(fields, f)
			f, material = s.scaleChoice(k+"material", x.word("editor.items.pair.material"), x.word("editor.items.pair.material.help"), "material", t.Materials, data.ShopMaterials, m.Material)
			fields = append(fields, f)
		}
		fields = append(fields, choice(k+"allow", "editor.items.allow", "editor.items.allow.help", allow, "editor.items.allow.yes", "editor.items.allow.no"))
		name := m.Item
		if it, ok := installFind(install, m.Item); ok {
			name = it.shown
			if shape >= 0 && material >= 0 {
				fields = append(fields, sprite(k, m.Sprite, int64(data.ComposeItemCode(material, it.class, shape, it.row)))...)
			}
		} else if w := slices.IndexFunc(items.Weapons, func(w mod.WeaponRow) bool { return w.Key == m.Item }); w >= 0 {
			name = itemPlaceName(items.Weapons[w].Name, m.Item)
			fields = append(fields, sprite(k, m.Sprite, 0)...)
		}
		group(fmt.Sprintf(x.word("editor.items.material"), name), append(fields, remove(k))...)
	}
	return out
}

// scaleChoice is a choice of the first n entries of the tier or material
// scale sc, each by its word on axis, valued by its place counted from 1;
// the entry named current is the value, whose index it returns too.
func (s *mapInspectionEdits) scaleChoice(key, label, help, axis string, sc data.ScaleTable, n int, current string) (ui.InspectionField, int) {
	x := s.inspector
	f := ui.InspectionField{Key: key, Label: label, Meaning: help}
	at := -1
	for i := 0; sc != nil && i < sc.Len() && i < n; i++ {
		name := sc.EntryName(i)
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.itemGroupWord(axis, name, data.RowKey(name))})
		if name == current {
			f.Value, at = int64(i+1), i
		}
	}
	return f, at
}

// shippedItemRow is the install's row named name in the armour, shield or
// weapon table, with the tables' scales; false when no table holds it.
func (x *MapInspector) shippedItemRow(name string) (params []int32, ok bool) {
	t := x.contextBase().table
	if t == nil {
		return nil, false
	}
	for _, c := range []data.Collection{t.Armors, t.Shields, t.Weapons} {
		if c == nil {
			continue
		}
		for row := 1; row < c.Len(); row++ {
			if c.EntryName(row) == name {
				return c.EntryParams(row), true
			}
		}
	}
	return nil, false
}

// newItemRow adds the row "new.armour" or "new.weapon" makes from the
// install item it: an armour row worn where it is and standing in for it,
// or a weapon row copying it. Its key is the record key rule's "armour" or
// "weapon", numbered while held; its name, in every language, is the item's game name under
// a key the one key maker makes.
func (s *mapInspectionEdits) newItemRow(d *mod.ItemData, sets map[string]mod.Strings, it installItem) {
	prefix := "armour"
	if it.table == "weapon" {
		prefix = "weapon"
	}
	key := mod.RecordKey(prefix, func(k string) bool {
		return slices.ContainsFunc(d.Rows, func(r mod.ItemRow) bool { return r.Key == k }) || slices.ContainsFunc(d.Weapons, func(w mod.WeaponRow) bool { return w.Key == k })
	})
	nameKey := makeTextKey(sets, "item."+key)
	name := []rune(it.shown)
	if len(name) > 40 {
		name = name[:40]
	}
	for _, l := range textLanguages() {
		sets[l][nameKey] = string(name)
	}
	if it.table == "weapon" {
		d.Weapons = append(d.Weapons, mod.WeaponRow{Key: key, NameKey: nameKey, Base: it.name})
		return
	}
	d.Rows = append(d.Rows, mod.ItemRow{Key: key, NameKey: nameKey, Slot: itemSlotName(it.class), SlotNo: it.class, StandIn: it.name, Suit: mod.SuitAny})
}

// setItemsField makes one edit of the items page a draft step; a value the
// items reader or the item layer refuses is refused in its words.
func (s *mapInspectionEdits) setItemsField(key, value string) (*ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return nil, err
	}
	x := s.inspector
	if err := x.modWritable(); err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	gone := fmt.Errorf(x.word("editor.items.err.gone"), key)
	install := x.installItems()
	prompts := []string{itemsNewArmour, itemsNewWeapon, itemsNewChange, itemsNewMaterial}
	if value == "0" && slices.Contains(prompts, key) {
		return nil, nil
	}
	switch key {
	case itemsNewArmour, itemsNewWeapon:
		keep := armourBase
		if key == itemsNewWeapon {
			keep = weaponBase
		}
		it, _, ok := pickedItem(value, install, keep, nil)
		if !ok {
			return nil, gone
		}
		return s.editItems(func(d *mod.ItemData, sets map[string]mod.Strings) error {
			s.newItemRow(d, sets, it)
			return nil
		})
	case itemsNewChange:
		_, name, ok := pickedItem(value, install, anyItem, nil)
		if !ok {
			return nil, gone
		}
		return s.editItems(func(d *mod.ItemData, _ map[string]mod.Strings) error {
			c := mod.ItemChange{Item: name}
			if p, ok := x.shippedItemRow(name); ok && len(p) > 2 {
				t := x.contextBase().table
				price := data.ItemPrice(p[2], t.Shapes, t.Materials, data.ModItemShape, data.ModItemMaterial)
				c.Price = &price
			}
			d.Changes = append(d.Changes, c)
			return nil
		})
	case itemsNewMaterial:
		return s.editItems(func(d *mod.ItemData, _ map[string]mod.Strings) error {
			it, name, ok := pickedItem(value, install, anyItem, d.Weapons)
			if !ok {
				return gone
			}
			d.Materials = append(d.Materials, x.newItemMaterial(it, name))
			return nil
		})
	}
	parts := strings.SplitN(key, ".", 3)
	if len(parts) != 3 {
		return nil, gone
	}
	record, id, field := parts[0], parts[1], parts[2]
	if field == "sprite" && value != "" && (record == "item" || record == "weapon") {
		_, d, err := s.ImportItemSprite(id, value, false)
		return d, err
	}
	if field == "sprite" && value != "" && record == "material" {
		return s.materialSprite(id, value, false)
	}
	number := func(dst *int32, label string) error {
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf(x.word("editor.items.err.number"), label, value)
		}
		*dst = int32(n)
		return nil
	}
	optional := func(dst **int32, label string) error {
		if value == "" {
			*dst = nil
			return nil
		}
		var n int32
		if err := number(&n, label); err != nil {
			return err
		}
		*dst = &n
		return nil
	}
	pick := func(n int) (int, error) {
		v, err := strconv.Atoi(value)
		if err != nil || v < 1 || v > n {
			return 0, gone
		}
		return v - 1, nil
	}
	return s.editItems(func(d *mod.ItemData, sets map[string]mod.Strings) error {
		if lang, ok := strings.CutPrefix(field, "name."); ok && (record == "item" || record == "weapon") {
			return setItemRowName(d, sets, record, id, lang, value, gone)
		}
		switch record {
		case "item":
			i := slices.IndexFunc(d.Rows, func(r mod.ItemRow) bool { return r.Key == id })
			if i < 0 {
				return gone
			}
			r := &d.Rows[i]
			switch field {
			case "slot":
				n, err := pick(len(mod.ItemSlotNames()))
				if err != nil {
					return err
				}
				r.Slot = mod.ItemSlotNames()[n]
				r.SlotNo = mod.ItemSlots[r.Slot]
				if j := slices.IndexFunc(install, standInFor(r.SlotNo)); j >= 0 && !standInFor(r.SlotNo)(installOf(install, r.StandIn)) {
					r.StandIn = install[j].name
				}
			case "defence":
				return number(&r.Defence, x.word("editor.items.defence"))
			case "absorption":
				return number(&r.Absorption, x.word("editor.items.absorption"))
			case "weight":
				return number(&r.Weight, x.word("editor.items.weight"))
			case "price":
				return number(&r.Price, x.word("editor.items.price"))
			case "stand-in":
				_, name, ok := pickedItem(value, install, standInFor(r.SlotNo), nil)
				if !ok {
					return gone
				}
				r.StandIn = name
			case "for":
				n, err := pick(len(itemSuits))
				if err != nil {
					return err
				}
				r.Suit = itemSuits[n].suit
			case "shop":
				n, err := pick(2)
				if err != nil {
					return err
				}
				r.Stock = n == 1
			case "sprite":
				r.Sprite = ""
			case "off":
				switchItemRow(d, id)
			case "remove":
				removeItemRow(d, id)
			default:
				return gone
			}
		case "weapon":
			i := slices.IndexFunc(d.Weapons, func(w mod.WeaponRow) bool { return w.Key == id })
			if i < 0 {
				return gone
			}
			w := &d.Weapons[i]
			switch p := slices.IndexFunc(mod.WeaponParams, func(p mod.WeaponParam) bool { return p.Key == field }); {
			case p >= 0:
				w.Params = slices.DeleteFunc(slices.Clone(w.Params), func(v mod.WeaponParamValue) bool { return v.Key == field })
				if value != "" {
					var n int32
					if err := number(&n, x.word("editor.items.param."+field)); err != nil {
						return err
					}
					w.Params = append(w.Params, mod.WeaponParamValue{WeaponParam: mod.WeaponParams[p], Value: n})
				}
			case field == "base":
				_, name, ok := pickedItem(value, install, weaponBase, nil)
				if !ok {
					return gone
				}
				w.Base = name
			case field == "sprite":
				w.Sprite = ""
			case field == "off":
				switchItemRow(d, id)
			case field == "remove":
				removeItemRow(d, id)
			default:
				return gone
			}
		case "change", "material":
			n, err := strconv.Atoi(id)
			if err != nil || n < 0 || record == "change" && n >= len(d.Changes) || record == "material" && n >= len(d.Materials) {
				return gone
			}
			if record == "material" {
				return x.setItemMaterial(d, n, field, value, install, pick, gone)
			}
			c := &d.Changes[n]
			switch field {
			case "item":
				_, name, ok := pickedItem(value, install, anyItem, nil)
				if !ok {
					return gone
				}
				c.Item = name
			case "defence", "absorption", "weight", "price":
				dst := map[string]**int32{"defence": &c.Defence, "absorption": &c.Absorption, "weight": &c.Weight, "price": &c.Price}[field]
				return optional(dst, x.word(itemNumberWords[field]))
			case "off":
				if c.Off != nil && *c.Off {
					c.Off = nil
					if c.Defence == nil && c.Absorption == nil && c.Weight == nil && c.Price == nil {
						d.Changes = slices.Delete(d.Changes, n, n+1)
					}
					return nil
				}
				on := true
				c.Off = &on
			case "remove":
				d.Changes = slices.Delete(d.Changes, n, n+1)
			default:
				return gone
			}
		default:
			return gone
		}
		return nil
	})
}

// installOf is the install item named name; the zero item for none.
func installOf(items []installItem, name string) installItem {
	it, _ := installFind(items, name)
	return it
}

// setItemRowName sets the name of the added row id of record in lang: its
// text under the row's name key, which the one key maker makes when the row
// has none; an empty name leaves the language without the text.
func setItemRowName(d *mod.ItemData, sets map[string]mod.Strings, record, id, lang, value string, gone error) error {
	var nameKey *string
	if record == "item" {
		if i := slices.IndexFunc(d.Rows, func(r mod.ItemRow) bool { return r.Key == id }); i >= 0 {
			nameKey = &d.Rows[i].NameKey
		}
	} else if i := slices.IndexFunc(d.Weapons, func(w mod.WeaponRow) bool { return w.Key == id }); i >= 0 {
		nameKey = &d.Weapons[i].NameKey
	}
	if nameKey == nil || sets[lang] == nil {
		return gone
	}
	if *nameKey == "" {
		*nameKey = makeTextKey(sets, "item."+id)
	}
	if value == "" {
		delete(sets[lang], *nameKey)
		return nil
	}
	sets[lang][*nameKey] = value
	return nil
}

// newItemMaterial is the material record "new.material" adds for the item
// it, named name: the first tier and material its masks do not allow, the
// first of each for a weapon row the mod adds or a row that allows them all.
// It is not allowed yet: a picture chosen for it, or the allow choice, does.
func (x *MapInspector) newItemMaterial(it installItem, name string) mod.ItemMaterial {
	t := x.contextBase().table
	shape, material := 0, 0
	if t != nil && it.name != "" {
		var base data.Collection = t.Armors
		switch it.table {
		case "shield":
			base = t.Shields
		case "weapon":
			base = t.Weapons
		}
		if masks, ok := base.(data.MaskTable); ok {
			m := data.ItemMasks(masks, it.row)
		find:
			for s := 0; s < len(m) && s < t.Shapes.Len(); s++ {
				for b := 0; b < data.ShopMaterials && b < t.Materials.Len(); b++ {
					if m[s]&(1<<uint(b)) == 0 {
						shape, material = s, b
						break find
					}
				}
			}
		}
	}
	m := mod.ItemMaterial{Item: name}
	if t != nil {
		m.Tier, m.Material = t.Shapes.EntryName(shape), t.Materials.EntryName(material)
	}
	return m
}

// setItemMaterial makes one edit of the material record n.
func (x *MapInspector) setItemMaterial(d *mod.ItemData, n int, field, value string, install []installItem, pick func(int) (int, error), gone error) error {
	m := &d.Materials[n]
	t := x.contextBase().table
	switch field {
	case "item":
		_, name, ok := pickedItem(value, install, anyItem, d.Weapons)
		if !ok {
			return gone
		}
		m.Item = name
	case "tier", "material":
		if t == nil {
			return gone
		}
		sc, limit, dst := data.ScaleTable(t.Shapes), data.ShopShapes, &m.Tier
		if field == "material" {
			sc, limit, dst = t.Materials, data.ShopMaterials, &m.Material
		}
		v, err := pick(min(limit, sc.Len()))
		if err != nil {
			return err
		}
		*dst = sc.EntryName(v)
	case "allow":
		v, err := pick(2)
		if err != nil {
			return err
		}
		m.Allow = v == 0
		if !m.Allow {
			m.Sprite, m.Name = "", ""
		}
	case "sprite":
		m.Sprite = ""
	case "remove":
		d.Materials = slices.Delete(d.Materials, n, n+1)
	default:
		return gone
	}
	return nil
}

// switchItemRow switches the added row key off with a change, or back on by
// taking the change that switched it off out.
func switchItemRow(d *mod.ItemData, key string) {
	if rowOff(*d, key) {
		d.Changes = slices.DeleteFunc(d.Changes, func(c mod.ItemChange) bool { return c.Item == key })
		return
	}
	off := true
	d.Changes = append(d.Changes, mod.ItemChange{Item: key, Off: &off})
}

// removeItemRow takes the added row key out of the mod with the changes and
// material masks that name it; its name's text stays for the texts page.
func removeItemRow(d *mod.ItemData, key string) {
	d.Rows = slices.DeleteFunc(d.Rows, func(r mod.ItemRow) bool { return r.Key == key })
	d.Weapons = slices.DeleteFunc(d.Weapons, func(w mod.WeaponRow) bool { return w.Key == key })
	d.Changes = slices.DeleteFunc(d.Changes, func(c mod.ItemChange) bool { return c.Item == key })
	d.Materials = slices.DeleteFunc(d.Materials, func(m mod.ItemMaterial) bool { return m.Item == key })
}

// editItems is one draft step of the items file.
func (s *mapInspectionEdits) editItems(edit func(*mod.ItemData, map[string]mod.Strings) error) (*ui.InspectionDocument, error) {
	return s.editKind(func(k kindFiles) error { return s.writeItemsDraft(k.w, edit) })
}

// itemFiles are the items file and the strings files of its names.
func itemFiles() []string { return append([]string{mod.ItemsFile}, textFiles()...) }

// writeItemsDraft makes edit's change of the items file and of the text of
// each language in the project w through editRows.
func (s *mapInspectionEdits) writeItemsDraft(w *editorWorkspaceState, edit func(*mod.ItemData, map[string]mod.Strings) error) error {
	if w.Project == nil {
		w.Project = s.inspector.mod.document()
	}
	return kindFiles{s: s, w: w}.editRows(s.itemNames, itemFiles(), func(d *mod.Data) error { return edit(&d.Items, d.Strings) })
}

// ImportItemSprite copies the host file source into the open mod under
// assets/items, held to the rule the item layer reads a sprite by, and makes
// it the sprite of the added row key, in one draft step through the one
// file import.
func (s *mapInspectionEdits) ImportItemSprite(key, source string, replace bool) (string, *ui.InspectionDocument, error) {
	gone := fmt.Errorf(s.inspector.word("editor.items.err.gone"), key)
	return s.importItemSprite(source, replace, func() (*ui.InspectionDocument, error) {
		_, d, err := s.ImportItemSprite(key, source, true)
		return d, err
	}, func(d *mod.ItemData, rel string) error {
		if i := slices.IndexFunc(d.Rows, func(r mod.ItemRow) bool { return r.Key == key }); i >= 0 {
			d.Rows[i].Sprite = rel
			return nil
		}
		if i := slices.IndexFunc(d.Weapons, func(r mod.WeaponRow) bool { return r.Key == key }); i >= 0 {
			d.Weapons[i].Sprite = rel
			return nil
		}
		return gone
	})
}

// importItemSprite imports the host picture source and makes edit's change
// of the items with its path in the same draft step; again repeats the
// import once a held file may be replaced.
func (s *mapInspectionEdits) importItemSprite(source string, replace bool, again func() (*ui.InspectionDocument, error), edit func(*mod.ItemData, string) error) (string, *ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return "", nil, err
	}
	rel, data, err := s.modAssetDraft(mod.ItemSprite.Kind, source, modAssetName(source))
	if err != nil {
		return "", nil, err
	}
	return s.importModFile(rel, data, replace, again, func(w *editorWorkspaceState, rel string) error {
		return s.writeItemsDraft(w, func(d *mod.ItemData, _ map[string]mod.Strings) error { return edit(d, rel) })
	})
}

// materialSprite imports the host picture source as the sprite of the
// material record at place n of the items file and allows its pair: the
// reader takes a sprite on an allowed pair only, and forbidding the pair
// takes its sprite and name off.
func (s *mapInspectionEdits) materialSprite(n, source string, replace bool) (*ui.InspectionDocument, error) {
	gone := fmt.Errorf(s.inspector.word("editor.items.err.gone"), "material."+n+".sprite")
	_, d, err := s.importItemSprite(source, replace, func() (*ui.InspectionDocument, error) {
		return s.materialSprite(n, source, true)
	}, func(d *mod.ItemData, rel string) error {
		i, err := strconv.Atoi(n)
		if err != nil || i < 0 || i >= len(d.Materials) {
			return gone
		}
		d.Materials[i].Sprite, d.Materials[i].Allow = rel, true
		return nil
	})
	return d, err
}
