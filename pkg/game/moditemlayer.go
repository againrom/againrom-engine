package game

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

// itemRecordKind names an item record in a refusal and an override.
const itemRecordKind = "item"

// itemPair is one (tier, material) bit of a row's masks: the shape index and
// the material index.
type itemPair struct{ shape, material int }

// itemRecord is one record of the items layer step: a shipped armour, shield
// or weapon row, an armour or shield item a mod adds, or a weapon row a mod
// adds as a copy of a shipped one.
type itemRecord struct {
	key   string
	class *itemClass
	// row is the shipped row; zero for an added row until it takes one.
	row    int
	params []int32
	masks  [data.ShopShapes]uint16
	// item is the added armour or shield item; weapon the added weapon row
	// and baseRow the shipped row it copies.
	item    *mod.ItemRow
	weapon  *mod.WeaponRow
	baseRow int
	// pairs are the [[material]] records that allow a pair, for its picture
	// and its shown name.
	pairs map[itemPair]mod.ItemMaterial
}

// shippedItemKey is the layer key of a shipped row. An added key holds no
// space, so the two never meet.
func shippedItemKey(c *itemClass, name string) string { return c.name + " " + name }

// itemLayer0 is the shipped rows of the three tables, each keyed by its table
// and its name; a name a table holds twice keys its first row, the one a
// change by name reaches.
func itemLayer0(classes []*itemClass) []itemRecord {
	var out []itemRecord
	seen := map[string]bool{}
	for _, c := range classes {
		for row := 1; row < c.base.Len(); row++ {
			name := c.base.EntryName(row)
			key := shippedItemKey(c, name)
			if name == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, itemRecord{key: key, class: c, row: row,
				params: slices.Clone(c.base.EntryParams(row)), masks: data.ItemMasks(maskTable(c.base), row)})
		}
	}
	return out
}

// itemModOrder is the load order of the mods that declare items: the order
// the loader read them in, else the order their records first appear.
func itemModOrder(items mod.ItemData) []string {
	var order []string
	seen := map[string]bool{}
	add := func(m string) {
		if !seen[m] {
			seen[m] = true
			order = append(order, m)
		}
	}
	for _, m := range items.Mods {
		add(m)
	}
	for _, r := range items.Rows {
		add(r.Mod)
	}
	for _, w := range items.Weapons {
		add(w.Mod)
	}
	for _, c := range items.Changes {
		add(c.Mod)
	}
	for _, m := range items.Materials {
		add(m.Mod)
	}
	return order
}

// itemPatches turns the mods' item records into layer patches, mod by mod in
// load order and, within a mod, added items, added weapon rows, changes, then
// materials. A record that names no row it may reach is refused here.
func itemPatches(items mod.ItemData, classes []*itemClass, t *mapload.Table) ([]mod.Patch[string, itemRecord], error) {
	armors, shields, weapons := classes[0], classes[1], classes[2]
	var patches []mod.Patch[string, itemRecord]
	// added is the mod that first added each key, so far in load order;
	// weaponKeys are the keys of the added weapon rows among them.
	added := map[string]string{}
	weaponKeys := map[string]bool{}
	for _, id := range itemModOrder(items) {
		for _, r := range items.Rows {
			if r.Mod != id {
				continue
			}
			class := armors
			if r.SlotNo == 2 {
				class = shields
			}
			patches = append(patches, mod.Patch[string, itemRecord]{Source: mod.Source{Mod: r.Mod, File: r.File, Line: r.Line}, Key: r.Key,
				Fields: []mod.PatchField{{Name: "row", Line: r.Line}},
				Apply:  func(rec *itemRecord) { *rec = itemRecord{key: r.Key, class: class, item: &r} }})
			if _, ok := added[r.Key]; !ok {
				added[r.Key] = r.Mod
			}
			delete(weaponKeys, r.Key)
		}
		for _, w := range items.Weapons {
			if w.Mod != id {
				continue
			}
			base := weapons.lookup(w.Base)
			if base == 0 {
				return nil, itemFileError(w.Mod, w.File, w.BaseLine, "base %q: the weapon table has no row named %q", w.Base, w.Base)
			}
			params := slices.Clone(weapons.base.EntryParams(base))
			for _, p := range w.Params {
				for len(params) <= p.Column {
					params = append(params, -1)
				}
				params[p.Column] = p.Value
			}
			masks := data.ItemMasks(maskTable(weapons.base), base)
			patches = append(patches, mod.Patch[string, itemRecord]{Source: mod.Source{Mod: w.Mod, File: w.File, Line: w.Line}, Key: w.Key,
				Fields: []mod.PatchField{{Name: "row", Line: w.Line}},
				Apply: func(rec *itemRecord) {
					*rec = itemRecord{key: w.Key, class: weapons, params: slices.Clone(params), masks: masks, weapon: &w, baseRow: base}
				}})
			if _, ok := added[w.Key]; !ok {
				added[w.Key] = w.Mod
			}
			weaponKeys[w.Key] = true
		}
		for _, c := range items.Changes {
			if c.Mod != id {
				continue
			}
			p, err := itemChangePatch(c, classes, t)
			if err != nil {
				return nil, err
			}
			patches = append(patches, p)
		}
		for _, m := range items.Materials {
			if m.Mod != id {
				continue
			}
			p, err := itemMaterialPatch(m, classes, t, weaponKeys)
			if err != nil {
				return nil, err
			}
			patches = append(patches, p)
		}
	}
	return patches, nil
}

// itemRowKey is the layer key of the shipped row a change or a material
// names, refusing a name no table or more than one table holds.
func itemRowKey(modID, file string, line int, name string, classes []*itemClass) (string, *itemClass, error) {
	var found *itemClass
	for _, cl := range classes {
		if cl.lookup(name) != 0 {
			if found != nil {
				return "", nil, itemFileError(modID, file, line, "%q names a row in more than one item table", name)
			}
			found = cl
		}
	}
	if found == nil {
		return "", nil, itemFileError(modID, file, line, "no row named %q in the armour, shield or weapon table", name)
	}
	return shippedItemKey(found, name), found, nil
}

// itemChangePatch is a change of a shipped row: one field per number it sets.
func itemChangePatch(c mod.ItemChange, classes []*itemClass, t *mapload.Table) (mod.Patch[string, itemRecord], error) {
	key, cl, err := itemRowKey(c.Mod, c.File, c.Line, c.Item, classes)
	if err != nil {
		return mod.Patch[string, itemRecord]{}, err
	}
	if cl.name == "weapon" && (c.Defence != nil || c.Absorption != nil) {
		return mod.Patch[string, itemRecord]{}, itemFileError(c.Mod, c.File, c.Line, "%q is a weapon; a change can set its price and weight only", c.Item)
	}
	type set struct {
		field string
		a     data.ItemAttr
		v     int32
	}
	var sets []set
	var fields []mod.PatchField
	for _, s := range []struct {
		field string
		a     data.ItemAttr
		v     *int32
	}{{"price", data.AttrPrice, c.Price}, {"weight", data.AttrWeight, c.Weight}, {"defence", data.AttrDefence, c.Defence}, {"absorption", data.AttrAbsorption, c.Absorption}} {
		if s.v != nil {
			sets = append(sets, set{s.field, s.a, *s.v})
			fields = append(fields, mod.PatchField{Name: s.field, Line: c.Line})
		}
	}
	return mod.Patch[string, itemRecord]{Source: mod.Source{Mod: c.Mod, File: c.File, Line: c.Line}, Key: key, Fields: fields,
		Apply: func(rec *itemRecord) {
			for _, s := range sets {
				rec.params = data.WithReferenceColumn(rec.params, t.Shapes, t.Materials, s.a, s.v)
			}
		}}, nil
}

// itemMaterialPatch is one mask bit of a shipped row or of a weapon row a mod
// loaded so far adds: the field "<tier> <material>".
func itemMaterialPatch(m mod.ItemMaterial, classes []*itemClass, t *mapload.Table, weaponKeys map[string]bool) (mod.Patch[string, itemRecord], error) {
	bad := func(line int, format string, args ...any) error {
		return itemFileError(m.Mod, m.File, line, format, args...)
	}
	key := m.Item
	if !weaponKeys[m.Item] {
		var err error
		if key, _, err = itemRowKey(m.Mod, m.File, m.Line, m.Item, classes); err != nil {
			return mod.Patch[string, itemRecord]{}, bad(m.Line, "%q is neither a row of the armour, shield or weapon table nor the key of a weapon row this mod or one it loads after adds", m.Item)
		}
	}
	shape := scaleIndex(t.Shapes, m.Tier, data.ShopShapes)
	if shape < 0 {
		return mod.Patch[string, itemRecord]{}, bad(m.Line, "tier %q is not one of %s", m.Tier, scaleNames(t.Shapes, data.ShopShapes))
	}
	material := scaleIndex(t.Materials, m.Material, data.ShopMaterials)
	if material < 0 {
		return mod.Patch[string, itemRecord]{}, bad(m.Line, "material %q is not one of %s", m.Material, scaleNames(t.Materials, data.ShopMaterials))
	}
	pair := itemPair{shape, material}
	return mod.Patch[string, itemRecord]{Source: mod.Source{Mod: m.Mod, File: m.File, Line: m.Line}, Key: key,
		Fields: []mod.PatchField{{Name: t.Shapes.EntryName(shape) + " " + t.Materials.EntryName(material), Line: m.AllowLine}},
		Apply: func(rec *itemRecord) {
			bit := uint16(1) << uint(material)
			pairs := make(map[itemPair]mod.ItemMaterial, len(rec.pairs)+1)
			for k, v := range rec.pairs {
				pairs[k] = v
			}
			if m.Allow {
				rec.masks[shape] |= bit
				pairs[pair] = m
			} else {
				rec.masks[shape] &^= bit
				delete(pairs, pair)
			}
			rec.pairs = pairs
		}}, nil
}

// scaleIndex is the index of the entry of s named name among its first n
// entries, or -1.
func scaleIndex(s data.ScaleTable, name string, n int) int {
	for i := 0; i < s.Len() && i < n; i++ {
		if s.EntryName(i) == name {
			return i
		}
	}
	return -1
}

// checkWeaponDamage refuses a mod weapon row whose damage-min exceeds its
// damage-max, or whose scaled base or spread leaves 0..255 on an allowed pair.
func checkWeaponDamage(w mod.WeaponRow, params []int32, masks [data.ShopShapes]uint16, t *mapload.Table) error {
	if len(params) <= 7 {
		return nil
	}
	lo, hi := params[6], params[7]
	line := w.Line
	for _, p := range w.Params {
		if p.Key == "damage-min" || p.Key == "damage-max" {
			line = p.Line
		}
	}
	if lo > hi {
		return itemFileError(w.Mod, w.File, line, "weapon row %q: damage-min is %d and damage-max is %d; damage-min must not exceed damage-max", w.Key, lo, hi)
	}
	for shape := 0; shape < data.ShopShapes && shape < t.Shapes.Len(); shape++ {
		for material := 0; material < data.ShopMaterials && material < t.Materials.Len(); material++ {
			if masks[shape]&(1<<uint(material)) == 0 {
				continue
			}
			base, spread := data.WeaponDamageScaled(t.Shapes, t.Materials, shape, material, lo, hi)
			if base < 0 || base > 255 || spread < 0 || spread > 255 {
				return itemFileError(w.Mod, w.File, line, "weapon row %q: damage-min %d and damage-max %d scale to base %d and spread %d at tier %q material %q; both must stay within 0..255",
					w.Key, lo, hi, base, spread, t.Shapes.EntryName(shape), t.Materials.EntryName(material))
			}
		}
	}
	return nil
}

// pairPicture is the inventory picture of one pair a mod allows: the sprite
// the mod gives, else the install's icon of the code itself, else the
// install's icon of from, the same pair of the shipped row an added weapon
// row copies. A pair with none of them is refused at src.
type pairPicture struct {
	src  mod.Source
	what string
	code data.ItemCode
	icon []byte
	from data.ItemCode
}

// pairPicturesOf names the pairs of rec a mod allows and lists their
// pictures: every pair of bits, and every pair whose [[material]] gives a
// sprite. rowSprite is the sprite of every pair of an added weapon row.
func pairPicturesOf(rec itemRecord, bits [data.ShopShapes]uint16, t *mapload.Table, rowSprite []byte,
	shown func(modID, file string, line int, name string) (string, error), names data.ItemNames) ([]pairPicture, error) {
	var out []pairPicture
	for shape := 0; shape < data.ShopShapes; shape++ {
		for material := 0; material < data.ShopMaterials; material++ {
			pair, has := rec.pairs[itemPair{shape, material}]
			newBit := bits[shape]&(1<<uint(material)) != 0
			if !newBit && !has {
				continue
			}
			code := data.ComposeItemCode(material, rec.class.shop(rec.params), shape, rec.row)
			p := pairPicture{code: code, icon: rowSprite,
				what: fmt.Sprintf("%s row %q tier %q material %q", rec.class.name, rowName(rec), t.Shapes.EntryName(shape), t.Materials.EntryName(material))}
			if rec.weapon != nil {
				p.src = mod.Source{Mod: rec.weapon.Mod, File: rec.weapon.File, Line: rec.weapon.Line}
				p.from = data.ComposeItemCode(material, rec.class.shop(rec.params), shape, rec.baseRow)
			}
			if has {
				p.src = mod.Source{Mod: pair.Mod, File: pair.File, Line: pair.Line}
				if pair.Name != "" {
					name, err := shown(pair.Mod, pair.File, pair.NameLine, pair.Name)
					if err != nil {
						return nil, err
					}
					names[code] = name
				}
				if pair.Sprite != "" {
					icon, err := modItemSprite(pair.Mod, pair.File, pair.SpriteLine, pair.Dir, pair.Sprite)
					if err != nil {
						return nil, err
					}
					p.icon = icon
				}
			}
			if newBit || p.icon != nil {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// rowName is the name a mod spells a record's row with.
func rowName(rec itemRecord) string {
	if rec.weapon != nil || rec.item != nil {
		return rec.key
	}
	return rec.class.base.EntryName(rec.row)
}

// modItemSprite is the inventory picture a mod's PNG gives, in the game's
// icon format.
func modItemSprite(modID, file string, line int, dir, rel string) ([]byte, error) {
	png, _, err := mod.ReadInside(dir, rel)
	if err != nil {
		return nil, itemFileError(modID, file, line, "sprite %q: %v", rel, err)
	}
	icon, err := modIconSprite(png)
	if err != nil {
		return nil, itemFileError(modID, file, line, "sprite %q: %v", rel, err)
	}
	return icon, nil
}

// servePairPictures serves the pictures of the pairs the mods allow, and
// refuses a pair whose picture neither the mod nor the install gives. A pair
// of an added weapon row is worn with the figure layers of the pair it copies.
func servePairPictures(files *modFiles, pictures []pairPicture) error {
	fs := files.fs
	for _, p := range pictures {
		icon := graphicsPrefix + data.ItemIconPath(p.code)
		if p.from != 0 {
			for _, dir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirManMage, data.FigureDirWomanFighter, data.FigureDirWomanMage} {
				from := graphicsPrefix + data.ItemFigureLayerPath(dir, p.from)
				if err := files.serve(modFileEntry{mod: p.src.Mod, file: p.src.File, address: graphicsPrefix + data.ItemFigureLayerPath(dir, p.code),
					read: func() ([]byte, error) { return fs.ReadFile(from) }}); err != nil {
					return err
				}
			}
		}
		switch {
		case p.icon != nil:
			if err := files.serveBytes(p.src.Mod, p.src.File, icon, p.icon); err != nil {
				return err
			}
			continue
		case installHas(fs, icon):
			continue
		case p.from != 0 && installHas(fs, graphicsPrefix+data.ItemIconPath(p.from)):
			from := graphicsPrefix + data.ItemIconPath(p.from)
			if err := files.serve(modFileEntry{mod: p.src.Mod, file: p.src.File, address: icon,
				read: func() ([]byte, error) { return fs.ReadFile(from) }}); err != nil {
				return err
			}
			continue
		}
		return p.src.Errorf("%s has no inventory picture: the install holds none at %s and the mod gives no sprite", p.what, icon)
	}
	return nil
}

func installHas(fs *vfs.FS, address string) bool {
	_, err := fs.ReadFile(address)
	return err == nil
}

func scaleNames(s data.ScaleTable, n int) string {
	var names []string
	for i := 0; i < s.Len() && i < n; i++ {
		if name := s.EntryName(i); name != "" {
			names = append(names, fmt.Sprintf("%q", name))
		}
	}
	return strings.Join(names, ", ")
}

// SetModItems builds the item tables the mods declare. The mods' item records
// merge through mod.Merge over layer 0, the shipped armour, shield and weapon
// rows with their material masks; each merged row a mod named is then
// written. It runs once, after SetMods and before any game is opened.
//
// A changed shipped row is an overlay row with its numbers and, when a mask
// bit changed, its masks. Each added item and each added weapon row takes the
// lowest free row of its table, in load order. Names go into the item-name
// table, pictures into the file system the game reads pictures from, and the
// save writer and the shop learn the added codes through the table's mod
// context. A refusal names the mod, the file and the line.
func (f *FrontEnd) SetModItems(items mod.ItemData) error {
	_, err := f.setModItems(items)
	return err
}

// setModItems is SetModItems answering the fields a later mod set over an
// earlier one.
func (f *FrontEnd) setModItems(items mod.ItemData) ([]mod.Override, error) {
	if items.Empty() {
		return nil, nil
	}
	t := f.Table
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Armors == nil || t.Shields == nil || t.Weapons == nil {
		return nil, errors.New("the install has no item tables for the mods to extend")
	}
	armors := &itemClass{name: "armour", base: t.Armors, shop: data.ArmorShopClass, rows: map[int]data.OverlayRow{}, next: t.Armors.Len()}
	shields := &itemClass{name: "shield", base: t.Shields, shop: data.ShieldShopClass, rows: map[int]data.OverlayRow{}, next: t.Shields.Len()}
	weapons := &itemClass{name: "weapon", base: t.Weapons, shop: data.WeaponShopClass, rows: map[int]data.OverlayRow{}, next: t.Weapons.Len()}
	classes := []*itemClass{armors, shields, weapons}

	patches, err := itemPatches(items, classes, t)
	if err != nil {
		return nil, err
	}
	l, err := mod.Merge(itemRecordKind, itemLayer0(classes), func(r itemRecord) string { return r.key }, patches, items.Order, nil)
	if err != nil {
		return nil, err
	}

	var added []mapload.ModItem
	var addedRows []mod.ItemRow
	names := make(data.ItemNames, len(t.Names)+len(items.Rows))
	for k, v := range t.Names {
		names[k] = v
	}
	selector := f.textSelector()
	shown := func(modID, file string, line int, name string) (string, error) {
		// Panels draw the install's byte alphabet, not UTF-8.
		s, err := encodeSaveLabel(name, selector)
		if err != nil {
			return "", itemFileError(modID, file, line, "the name %q cannot be drawn by this install: %v", name, err)
		}
		return s, nil
	}
	type picture struct {
		row    mod.ItemRow
		code   data.ItemCode
		in     data.ItemCode
		figure data.ItemCode
		icon   []byte
	}
	var pictures []picture
	var pairPictures []pairPicture
	bodies, addedWeapons := f.Bodies, 0
	for _, rec := range l.Records {
		if _, named := l.Named[rec.key]; !named {
			continue
		}
		switch {
		case rec.item != nil:
			r := *rec.item
			class := rec.class
			if class.next > maxItemRow {
				return nil, itemFileError(r.Mod, r.File, r.Line, "no free row: the %s table holds %d rows, the most an item code names, and every one is in use", class.name, maxItemRow+1)
			}
			row := class.next
			class.next++
			in, inRow, err := resolveStandIn(r, class, t)
			if err != nil {
				return nil, err
			}
			code := data.ComposeItemCode(data.ModItemMaterial, r.SlotNo, data.ModItemShape, row)
			class.rows[row] = data.OverlayRow{
				Name:   r.Key,
				Params: data.NewModRowParams(t.Shapes, t.Materials, r.SlotNo, int(r.Suit), r.Price, r.Weight, r.Defence, r.Absorption),
				Raw:    data.ModRowRaw(),
			}
			line := r.NameLine
			if line == 0 {
				line = r.Line
			}
			if names[code], err = shown(r.Mod, r.File, line, r.Name); err != nil {
				return nil, err
			}
			p := picture{row: r, code: code, in: in, figure: in}
			if r.Figure != "" {
				if p.figure, err = resolveFigure(r, armors, t); err != nil {
					return nil, err
				}
			}
			if r.Sprite != "" {
				if p.icon, err = modItemSprite(r.Mod, r.File, r.SpriteLine, r.Dir, r.Sprite); err != nil {
					return nil, err
				}
			}
			pictures = append(pictures, p)
			price := data.ItemPrice(class.rows[row].Params[2], t.Shapes, t.Materials, data.ModItemShape, data.ModItemMaterial)
			if price != r.Price {
				return nil, itemFileError(r.Mod, r.File, r.Line, "the item table cannot state price %d for this item; it holds %d", r.Price, price)
			}
			added = append(added, mapload.ModItem{
				Mod: r.Mod, Key: r.Key, Code: uint16(code), Row: uint8(row), StandInCode: uint16(in), StandInRow: uint8(inRow),
				Price: price, Stock: r.Stock, Layer: layerDepth(r.Layer), Anchor: uint8(r.AnchorNo),
			})
			addedRows = append(addedRows, r)
		case rec.weapon != nil:
			w := *rec.weapon
			if weapons.next > maxItemRow {
				return nil, itemFileError(w.Mod, w.File, w.Line, "no free row: the weapon table holds %d rows, the most an item code names, and every one is in use", maxItemRow+1)
			}
			if err := checkWeaponDamage(w, rec.params, rec.masks, t); err != nil {
				return nil, err
			}
			rec.row = weapons.next
			weapons.next++
			bodies = bodies.WithAddedRow(rec.row, rec.baseRow)
			addedWeapons++
			weapons.rows[rec.row] = data.OverlayRow{Name: rec.key, Params: rec.params, Raw: data.ItemMaskRaw(rec.masks)}
			name, err := shown(w.Mod, w.File, w.NameLine, w.Name)
			if err != nil {
				return nil, err
			}
			var sprite []byte
			if w.Sprite != "" {
				if sprite, err = modItemSprite(w.Mod, w.File, w.SpriteLine, w.Dir, w.Sprite); err != nil {
					return nil, err
				}
			}
			for shape := 0; shape < data.ShopShapes && shape < t.Shapes.Len(); shape++ {
				for material := 0; material < data.ShopMaterials && material < t.Materials.Len(); material++ {
					code := data.ComposeItemCode(material, rec.class.shop(rec.params), shape, rec.row)
					names[code] = name
					added = append(added, mapload.ModItem{Mod: w.Mod, Key: w.Key, Code: uint16(code), Row: uint8(rec.row),
						StandInCode: uint16(data.ComposeItemCode(material, rec.class.shop(rec.params), shape, rec.baseRow)), StandInRow: uint8(rec.baseRow),
						Price: data.ItemPrice(rec.params[2], t.Shapes, t.Materials, shape, material)})
				}
			}
			pics, err := pairPicturesOf(rec, rec.masks, t, sprite, shown, names)
			if err != nil {
				return nil, err
			}
			pairPictures = append(pairPictures, pics...)
		default:
			shipped := data.ItemMasks(maskTable(rec.class.base), rec.row)
			if rec.masks == shipped && slices.Equal(rec.params, rec.class.base.EntryParams(rec.row)) && len(rec.pairs) == 0 {
				continue
			}
			row := data.OverlayRow{Name: rec.class.base.EntryName(rec.row), Params: rec.params}
			if rec.masks != shipped {
				row.Raw = data.ItemMaskRaw(rec.masks)
			}
			rec.class.rows[rec.row] = row
			var bits [data.ShopShapes]uint16
			for s := range bits {
				bits[s] = rec.masks[s] &^ shipped[s]
			}
			pics, err := pairPicturesOf(rec, bits, t, nil, shown, names)
			if err != nil {
				return nil, err
			}
			pairPictures = append(pairPictures, pics...)
		}
	}

	overlay := func(c *itemClass) data.Collection {
		if len(c.rows) == 0 {
			return c.base
		}
		return data.NewRowOverlay(c.base, c.rows)
	}
	t.Armors, t.Shields, t.Weapons = overlay(armors), overlay(shields), overlay(weapons)

	for i, r := range addedRows {
		if err := checkItemNumbers(r, data.ItemCode(added[i].Code), t); err != nil {
			return nil, err
		}
	}
	t.Names = names
	t.Mods.Items = append(t.Mods.Items[:0:0], added...)
	if addedWeapons > 0 {
		f.Bodies, t.Mods.Bodies = bodies, bodies
	}

	if f.Archives != nil && f.Archives.Containers != nil {
		files := f.modFileLayer()
		fs := files.fs
		if err := servePairPictures(files, pairPictures); err != nil {
			return nil, err
		}
		if addedWeapons > 0 {
			if err := f.setBodyList(bodies); err != nil {
				return nil, err
			}
		}
		for _, p := range pictures {
			icon := graphicsPrefix + data.ItemIconPath(p.code)
			if p.icon != nil {
				if err := files.serveBytes(p.row.Mod, p.row.File, icon, p.icon); err != nil {
					return nil, err
				}
			} else {
				from := graphicsPrefix + data.ItemIconPath(p.in)
				if err := files.serve(modFileEntry{mod: p.row.Mod, file: p.row.File, address: icon, read: func() ([]byte, error) { return fs.ReadFile(from) }}); err != nil {
					return nil, err
				}
			}
			for _, dir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirManMage, data.FigureDirWomanFighter, data.FigureDirWomanMage} {
				other := counterpartDir(dir)
				for _, paths := range [][3]string{
					{data.ItemFigureLayerPath(dir, p.code), data.ItemFigureLayerPath(dir, p.figure), data.ItemFigureLayerPath(other, p.figure)},
					{data.ItemFigureSecondaryLayerPath(dir, p.code), data.ItemFigureSecondaryLayerPath(dir, p.figure), data.ItemFigureSecondaryLayerPath(other, p.figure)},
				} {
					from, alt := graphicsPrefix+paths[1], ""
					if p.row.Figure != "" {
						alt = graphicsPrefix + paths[2]
					}
					read := func() ([]byte, error) {
						b, err := fs.ReadFile(from)
						if err != nil && alt != "" {
							b, err = fs.ReadFile(alt)
						}
						return b, err
					}
					if err := files.serve(modFileEntry{mod: p.row.Mod, file: p.row.File, address: graphicsPrefix + paths[0], read: read}); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return l.Overrides, nil
}
