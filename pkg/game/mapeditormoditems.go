package game

import (
	"bytes"
	"fmt"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// The items page's keys: a record's field is "<record>.<id>.<field>", the
// record being an added "item" or "weapon" row by its key or a "change" or
// "material" by its place in the file; the "new." fields add a record.
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

// itemRows is the open project's draft of the mod's items file: its records
// as the items reader reads them, and its bytes. A name key the project's
// strings lack reads as "?"; the mod's context refuses it in the reader's
// words.
func (s *mapInspectionEdits) itemRows(w *editorWorkspaceState) (mod.ItemData, []byte, error) {
	m := s.inspector.mod
	d := w.Project
	if d == nil {
		d = m.document()
	}
	src := d.Files[mod.ItemsFile]
	lang := s.inspector.pageLanguage()
	look, err := d.textLookup(lang)
	if err != nil {
		return mod.ItemData{}, nil, err
	}
	shown := func(key string) (string, bool) {
		if v, ok := look(key); ok && v != "" && utf8.RuneCountInString(v) <= 40 {
			return v, true
		}
		return "?", true
	}
	items, err := mod.ParseItems(m.id, mod.ItemsFile, src, shown, lang)
	return items, src, err
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

// itemsFields is W's items page: each row the mod adds, each change of a
// shipped row and each material mask, and the rows that add one; none
// with no mod folder open.
func (s *mapInspectionEdits) itemsFields() []ui.InspectionField {
	x := s.inspector
	if x.mod == nil || x.mod.packed || s.prepareWorkspace() != nil {
		return nil
	}
	line := func(key, text string) ui.InspectionField {
		return ui.InspectionField{Key: key, ReadOnly: true, Text: true, TextValue: text, Lines: []ui.InspectionLine{{Text: x.inspectionUTF8(text)}}}
	}
	items, _, err := s.itemRows(s.workspace)
	if err != nil {
		return []ui.InspectionField{line("error", err.Error())}
	}
	var out []ui.InspectionField
	group := func(head string, fields ...ui.InspectionField) {
		for _, f := range fields {
			f.Group = head
			out = append(out, f)
		}
	}
	choice := func(key, label string, value int64, words ...string) ui.InspectionField {
		f := ui.InspectionField{Key: key, Label: x.word(label), Value: value}
		for i, w := range words {
			f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(x.word(w))})
		}
		return f
	}
	num := func(key, label string, v int32) ui.InspectionField {
		return workspaceText(key, x.word(label), strconv.Itoa(int(v)))
	}
	sprite := func(key, rel string) ui.InspectionField {
		f := workspaceText(key, x.word("editor.items.sprite"), rel)
		f.Meaning, f.Pick = x.word("editor.items.sprite.none"), []string{"png"}
		if rel != "" {
			f.Meaning = rel
		}
		return f
	}
	switchRow := func(key string, off bool) []ui.InspectionField {
		label := "editor.items.off"
		var fs []ui.InspectionField
		if off {
			label = "editor.items.on"
			fs = append(fs, line(key+"state", x.word("editor.items.isoff")))
		}
		return append(fs, ui.InspectionField{Key: key + "off", Label: x.word(label), Action: true, Column: 1})
	}
	remove := func(key string) ui.InspectionField {
		return ui.InspectionField{Key: key + "remove", Label: x.word("editor.items.remove"), Action: true, Column: 1}
	}
	own := map[string]bool{}
	slots := mod.ItemSlotNames()
	for _, r := range items.Rows {
		own[r.Key] = true
		k := "item." + r.Key + "."
		name := workspaceText(k+"name", x.word("editor.items.name"), r.NameKey)
		name.Meaning = r.Name
		slot := ui.InspectionField{Key: k + "slot", Label: x.word("editor.items.slot"), Value: int64(slices.Index(slots, r.Slot) + 1)}
		for i, n := range slots {
			slot.Choices = append(slot.Choices, ui.InspectionChoice{Value: int64(i + 1), Label: x.inspectionUTF8(n)})
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
		fields := []ui.InspectionField{name, slot,
			num(k+"defence", "editor.items.defence", r.Defence), num(k+"absorption", "editor.items.absorption", r.Absorption),
			num(k+"weight", "editor.items.weight", r.Weight), num(k+"price", "editor.items.price", r.Price),
			workspaceText(k+"stand-in", x.word("editor.items.standin"), r.StandIn),
			choice(k+"for", "editor.items.for", suit, "editor.items.for.any", "editor.items.for.fighter", "editor.items.for.mage"),
			choice(k+"shop", "editor.items.shop", shop, "editor.items.no", "editor.items.yes"),
			sprite(k+"sprite", r.Sprite)}
		fields = append(fields, switchRow(k, rowOff(items, r.Key))...)
		group(fmt.Sprintf(x.word("editor.items.armour"), r.Key), append(fields, remove(k))...)
	}
	for _, w := range items.Weapons {
		own[w.Key] = true
		k := "weapon." + w.Key + "."
		name := workspaceText(k+"name", x.word("editor.items.name"), w.NameKey)
		name.Meaning = w.Name
		fields := []ui.InspectionField{name, workspaceText(k+"base", x.word("editor.items.base"), w.Base), sprite(k+"sprite", w.Sprite)}
		for _, wp := range mod.WeaponParams {
			value := ""
			if i := slices.IndexFunc(w.Params, func(v mod.WeaponParamValue) bool { return v.Key == wp.Key }); i >= 0 {
				value = strconv.Itoa(int(w.Params[i].Value))
			}
			f := workspaceText(k+wp.Key, wp.Key, value)
			f.Meaning = x.word("editor.items.param")
			fields = append(fields, f)
		}
		fields = append(fields, switchRow(k, rowOff(items, w.Key))...)
		group(fmt.Sprintf(x.word("editor.items.weapon"), w.Key), append(fields, remove(k))...)
	}
	for i, c := range items.Changes {
		if own[c.Item] {
			continue
		}
		k := "change." + strconv.Itoa(i) + "."
		var fields []ui.InspectionField
		for _, n := range []struct {
			field, label string
			v            *int32
		}{{"defence", "editor.items.defence", c.Defence}, {"absorption", "editor.items.absorption", c.Absorption},
			{"weight", "editor.items.weight", c.Weight}, {"price", "editor.items.price", c.Price}} {
			value := ""
			if n.v != nil {
				value = strconv.Itoa(int(*n.v))
			}
			f := workspaceText(k+n.field, x.word(n.label), value)
			f.Meaning = x.word("editor.items.unchanged")
			fields = append(fields, f)
		}
		fields = append(fields, switchRow(k, c.Off != nil && *c.Off)...)
		group(fmt.Sprintf(x.word("editor.items.change"), c.Item), append(fields, remove(k))...)
	}
	for i, m := range items.Materials {
		k := "material." + strconv.Itoa(i) + "."
		allow := int64(2)
		if m.Allow {
			allow = 1
		}
		group(fmt.Sprintf(x.word("editor.items.material"), m.Item, m.Tier, m.Material),
			choice(k+"allow", "editor.items.allow", allow, "editor.items.allow.yes", "editor.items.allow.no"), sprite(k+"sprite", m.Sprite), remove(k))
	}
	for _, n := range []struct{ key, label, hint string }{
		{itemsNewArmour, "editor.items.new.armour", "editor.items.new.hint"},
		{itemsNewWeapon, "editor.items.new.weapon", "editor.items.new.weapon.hint"},
		{itemsNewChange, "editor.items.new.change", "editor.items.new.change.hint"},
		{itemsNewMaterial, "editor.items.new.material", "editor.items.new.material.hint"},
	} {
		f := workspaceText(n.key, x.word(n.label), "")
		f.Meaning = x.word(n.hint)
		if n.key == itemsNewWeapon {
			f.Pick = []string{"png"}
		}
		out = append(out, f)
	}
	return out
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

// newItemRow is the row "new.armour" or "new.weapon" adds under key: an
// armour row of the slot of the install's first armour row, standing in
// for it, or a weapon row copying the first weapon row; its name is the
// text key "item.<key>".
func (x *MapInspector) newItemRow(d *mod.ItemData, key string, weapon bool) {
	t := x.contextBase().table
	first := func(c data.Collection, fit func([]int32) bool) string {
		masks, _ := c.(data.MaskTable)
		for row := 1; c != nil && row < c.Len(); row++ {
			if masks != nil && data.ItemMasks(masks, row) == ([data.ShopShapes]uint16{}) {
				continue
			}
			if n := c.EntryName(row); n != "" && fit(c.EntryParams(row)) {
				return n
			}
		}
		return ""
	}
	name := "item." + key
	if weapon {
		base := ""
		if t != nil {
			base = first(t.Weapons, func([]int32) bool { return true })
		}
		d.Weapons = append(d.Weapons, mod.WeaponRow{Key: key, NameKey: name, Base: base})
		return
	}
	slot, standIn := "body", ""
	if t != nil {
		standIn = first(t.Armors, func(p []int32) bool {
			for n, no := range mod.ItemSlots {
				if len(p) > 4 && int(p[4]) == no {
					slot = n
					return true
				}
			}
			return false
		})
	}
	d.Rows = append(d.Rows, mod.ItemRow{Key: key, NameKey: name, Slot: slot, SlotNo: mod.ItemSlots[slot], StandIn: standIn, Suit: mod.SuitAny})
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
	switch key {
	case itemsNewWeapon:
		return s.newWeapon(value, false)
	case itemsNewArmour:
		return s.editItems(func(d *mod.ItemData, en mod.Strings) error {
			x.newItemRow(d, value, false)
			if _, held := en["item."+value]; !held {
				en["item."+value] = value
			}
			return nil
		})
	case itemsNewChange:
		return s.editItems(func(d *mod.ItemData, _ mod.Strings) error {
			c := mod.ItemChange{Item: value}
			if p, ok := x.shippedItemRow(value); ok && len(p) > 2 {
				t := x.contextBase().table
				price := data.ItemPrice(p[2], t.Shapes, t.Materials, data.ModItemShape, data.ModItemMaterial)
				c.Price = &price
			}
			d.Changes = append(d.Changes, c)
			return nil
		})
	case itemsNewMaterial:
		parts := strings.Split(value, ",")
		if len(parts) != 3 {
			return nil, fmt.Errorf(x.word("editor.items.err.material"), value)
		}
		return s.editItems(func(d *mod.ItemData, _ mod.Strings) error {
			d.Materials = append(d.Materials, mod.ItemMaterial{Item: strings.TrimSpace(parts[0]), Tier: strings.TrimSpace(parts[1]), Material: strings.TrimSpace(parts[2])})
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
	number := func(dst *int32) error {
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf(x.word("editor.items.err.number"), field, value)
		}
		*dst = int32(n)
		return nil
	}
	optional := func(dst **int32) error {
		if value == "" {
			*dst = nil
			return nil
		}
		var n int32
		if err := number(&n); err != nil {
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
	return s.editItems(func(d *mod.ItemData, _ mod.Strings) error {
		switch record {
		case "item":
			i := slices.IndexFunc(d.Rows, func(r mod.ItemRow) bool { return r.Key == id })
			if i < 0 {
				return gone
			}
			r := &d.Rows[i]
			switch field {
			case "name":
				r.NameKey = value
			case "slot":
				n, err := pick(len(mod.ItemSlotNames()))
				if err != nil {
					return err
				}
				r.Slot = mod.ItemSlotNames()[n]
				r.SlotNo = mod.ItemSlots[r.Slot]
			case "defence":
				return number(&r.Defence)
			case "absorption":
				return number(&r.Absorption)
			case "weight":
				return number(&r.Weight)
			case "price":
				return number(&r.Price)
			case "stand-in":
				r.StandIn = value
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
					if err := number(&n); err != nil {
						return err
					}
					w.Params = append(w.Params, mod.WeaponParamValue{WeaponParam: mod.WeaponParams[p], Value: n})
				}
			case field == "name":
				w.NameKey = value
			case field == "base":
				w.Base = value
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
				switch field {
				case "allow":
					v, err := pick(2)
					if err != nil {
						return err
					}
					d.Materials[n].Allow = v == 0
					if !d.Materials[n].Allow {
						d.Materials[n].Sprite, d.Materials[n].Name = "", ""
					}
				case "sprite":
					d.Materials[n].Sprite = ""
				case "remove":
					d.Materials = slices.Delete(d.Materials, n, n+1)
				default:
					return gone
				}
				return nil
			}
			c := &d.Changes[n]
			switch field {
			case "defence":
				return optional(&c.Defence)
			case "absorption":
				return optional(&c.Absorption)
			case "weight":
				return optional(&c.Weight)
			case "price":
				return optional(&c.Price)
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
func (s *mapInspectionEdits) editItems(edit func(*mod.ItemData, mod.Strings) error) (*ui.InspectionDocument, error) {
	return s.changeWorkspace(func(w *editorWorkspaceState) error { return s.writeItemsDraft(w, edit) })
}

// writeItemsDraft makes edit's change of the items file and of its English
// text in the project w: each kind's writer writes its file over the
// draft's bytes, and only a file whose content changed. A file with no
// record left leaves the mod.
func (s *mapInspectionEdits) writeItemsDraft(w *editorWorkspaceState, edit func(*mod.ItemData, mod.Strings) error) error {
	if w.Project == nil {
		w.Project = s.inspector.mod.document()
	}
	items, src, err := s.itemRows(w)
	if err != nil {
		return err
	}
	sets, err := w.Project.strings()
	if err != nil {
		return err
	}
	en := textLanguages()[0]
	if sets[en] == nil {
		sets[en] = mod.Strings{}
	}
	was := maps.Clone(sets[en])
	if err := edit(&items, sets[en]); err != nil {
		return err
	}
	kind, _ := mod.WrittenKind(mod.ItemsFile)
	out, err := kind.Write(mod.ItemsFile, src, mod.Data{Items: items})
	if err != nil {
		return err
	}
	if !bytes.Equal(out, src) {
		w.Project.Files[mod.ItemsFile] = out
		if items.Empty() && strings.TrimSpace(string(out)) == "" {
			s.dropModFile(w, mod.ItemsFile)
		}
	}
	if !maps.Equal(was, sets[en]) {
		rel := mod.StringsFile(en)
		kind, _ := mod.WrittenKind(rel)
		out, err := kind.Write(rel, w.Project.Files[rel], mod.Data{Strings: sets})
		if err != nil {
			return err
		}
		w.Project.Files[rel] = out
	}
	return nil
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
	}, func(d *mod.ItemData, _ mod.Strings, rel string) error {
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
func (s *mapInspectionEdits) importItemSprite(source string, replace bool, again func() (*ui.InspectionDocument, error), edit func(*mod.ItemData, mod.Strings, string) error) (string, *ui.InspectionDocument, error) {
	if err := s.prepareWorkspace(); err != nil {
		return "", nil, err
	}
	rel, data, err := s.modAssetDraft(mod.ItemSprite.Kind, source, modAssetName(source))
	if err != nil {
		return "", nil, err
	}
	return s.importModFile(rel, data, replace, again, func(w *editorWorkspaceState, rel string) error {
		return s.writeItemsDraft(w, func(d *mod.ItemData, en mod.Strings) error { return edit(d, en, rel) })
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
	}, func(d *mod.ItemData, _ mod.Strings, rel string) error {
		i, err := strconv.Atoi(n)
		if err != nil || i < 0 || i >= len(d.Materials) {
			return gone
		}
		d.Materials[i].Sprite, d.Materials[i].Allow = rel, true
		return nil
	})
	return d, err
}

// newWeapon adds a weapon row with the host picture source as its sprite,
// which a weapon the mod adds needs before the game takes it: the picture's
// name less its extension is the row's key.
func (s *mapInspectionEdits) newWeapon(source string, replace bool) (*ui.InspectionDocument, error) {
	_, d, err := s.importItemSprite(source, replace, func() (*ui.InspectionDocument, error) {
		return s.newWeapon(source, true)
	}, func(d *mod.ItemData, en mod.Strings, rel string) error {
		key := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		s.inspector.newItemRow(d, key, true)
		d.Weapons[len(d.Weapons)-1].Sprite = rel
		if _, held := en["item."+key]; !held {
			en["item."+key] = key
		}
		return nil
	})
	return d, err
}
