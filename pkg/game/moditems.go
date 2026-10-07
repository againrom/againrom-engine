package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// maxItemRow is the highest row an item code can name: field D of the code is
// five bits wide, so each item table has room for rows 0 to 31.
const maxItemRow = 31

// itemClass is one of the three item tables a code's class field selects.
type itemClass struct {
	name string
	base data.Collection
	rows map[int]data.OverlayRow
	next int
}

func (c *itemClass) lookup(name string) int {
	for i := 1; i < c.base.Len(); i++ {
		if c.base.EntryName(i) == name {
			return i
		}
	}
	return 0
}

func itemFileError(modID, file string, line int, format string, args ...any) error {
	return fmt.Errorf("mod %q: %w", modID, &modItemFileError{file: file, line: line, msg: fmt.Sprintf(format, args...)})
}

type modItemFileError struct {
	file string
	line int
	msg  string
}

func (e *modItemFileError) Error() string { return fmt.Sprintf("%s:%d: %s", e.file, e.line, e.msg) }

// SetModItems adds the items the mods add to the item tables and applies the
// edits to rows the game already has. It runs once, after SetMods and before
// any game is opened.
//
// Each added item takes the lowest free row of its table, in load order; the
// item's code is that row with the iron material and the common shape. The
// name goes into the item-name table, the picture into the file system the
// game reads pictures from, and the save writer and the shop learn the item
// through the table's mod context. A refusal names the mod, the file and the
// line.
func (f *FrontEnd) SetModItems(items mod.ItemData) error {
	if items.Empty() {
		return nil
	}
	t := f.Table
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Armors == nil || t.Shields == nil || t.Weapons == nil {
		return errors.New("the install has no item tables for the mods to extend")
	}
	armors := &itemClass{name: "armour", base: t.Armors, rows: map[int]data.OverlayRow{}, next: t.Armors.Len()}
	shields := &itemClass{name: "shield", base: t.Shields, rows: map[int]data.OverlayRow{}, next: t.Shields.Len()}
	weapons := &itemClass{name: "weapon", base: t.Weapons, rows: map[int]data.OverlayRow{}, next: t.Weapons.Len()}
	classes := []*itemClass{armors, shields, weapons}

	for _, c := range items.Changes {
		if err := applyItemChange(c, classes, t); err != nil {
			return err
		}
	}

	var added []mapload.ModItem
	names := make(data.ItemNames, len(t.Names)+len(items.Rows))
	for k, v := range t.Names {
		names[k] = v
	}
	type picture struct {
		row    mod.ItemRow
		code   data.ItemCode
		in     data.ItemCode
		figure data.ItemCode
		icon   []byte
	}
	var pictures []picture
	for _, r := range items.Rows {
		class := armors
		if r.SlotNo == 2 {
			class = shields
		}
		if class.next > maxItemRow {
			return itemFileError(r.Mod, r.File, r.Line, "no free row: the %s table holds %d rows, the most an item code names, and every one is in use", class.name, maxItemRow+1)
		}
		row := class.next
		class.next++
		in, inRow, err := resolveStandIn(r, class, t)
		if err != nil {
			return err
		}
		code := data.ComposeItemCode(data.ModItemMaterial, r.SlotNo, data.ModItemShape, row)
		class.rows[row] = data.OverlayRow{
			Name:   r.Key,
			Params: data.NewModRowParams(t.Shapes, t.Materials, r.SlotNo, int(r.Suit), r.Price, r.Weight, r.Defence, r.Absorption),
			Raw:    data.ModRowRaw(),
		}
		// Panels draw the install's byte alphabet, not UTF-8.
		shown, err := encodeSaveLabel(r.Name, f.textSelector())
		if err != nil {
			line := r.NameLine
			if line == 0 {
				line = r.Line
			}
			return itemFileError(r.Mod, r.File, line, "the name %q cannot be drawn by this install: %v", r.Name, err)
		}
		names[code] = shown
		p := picture{row: r, code: code, in: in, figure: in}
		if r.Figure != "" {
			if p.figure, err = resolveFigure(r, armors, t); err != nil {
				return err
			}
		}
		if r.Sprite != "" {
			png, _, err := mod.ReadInside(r.Dir, r.Sprite)
			if err != nil {
				return itemFileError(r.Mod, r.File, r.SpriteLine, "sprite %q: %v", r.Sprite, err)
			}
			if p.icon, err = modIconSprite(png); err != nil {
				return itemFileError(r.Mod, r.File, r.SpriteLine, "sprite %q: %v", r.Sprite, err)
			}
		}
		pictures = append(pictures, p)
		price := data.ItemPrice(class.rows[row].Params[2], t.Shapes, t.Materials, data.ModItemShape, data.ModItemMaterial)
		if price != r.Price {
			return itemFileError(r.Mod, r.File, r.Line, "the item table cannot state price %d for this item; it holds %d", r.Price, price)
		}
		added = append(added, mapload.ModItem{
			Mod: r.Mod, Key: r.Key, Code: uint16(code), Row: uint8(row), StandInCode: uint16(in), StandInRow: uint8(inRow),
			Price: price, Stock: r.Stock, Layer: layerDepth(r.Layer), Anchor: uint8(r.AnchorNo),
		})
	}

	overlay := func(c *itemClass) data.Collection {
		if len(c.rows) == 0 {
			return c.base
		}
		return data.NewRowOverlay(c.base, c.rows)
	}
	t.Armors, t.Shields, t.Weapons = overlay(armors), overlay(shields), overlay(weapons)

	for i, r := range items.Rows {
		if err := checkItemNumbers(r, data.ItemCode(added[i].Code), t); err != nil {
			return err
		}
	}
	t.Names = names
	t.Mods.Items = append(t.Mods.Items[:0:0], added...)

	if f.Archives != nil && f.Archives.Containers != nil {
		// The archives are shared by every front end over this install, so the
		// overlay goes on this front end's own filesystem.
		fs := f.Archives.Containers.Fork()
		own := *f.Archives
		own.Containers = fs
		f.Archives = &own
		for _, p := range pictures {
			icon := graphicsPrefix + data.ItemIconPath(p.code)
			if p.icon != nil {
				bytes := p.icon
				fs.Overlay(icon, func() ([]byte, error) { return append([]byte(nil), bytes...), nil })
			} else {
				from := graphicsPrefix + data.ItemIconPath(p.in)
				fs.Overlay(icon, func() ([]byte, error) { return fs.ReadFile(from) })
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
					fs.Overlay(graphicsPrefix+paths[0], func() ([]byte, error) {
						b, err := fs.ReadFile(from)
						if err != nil && alt != "" {
							b, err = fs.ReadFile(alt)
						}
						return b, err
					})
				}
			}
		}
	}
	return nil
}

// counterpartDir is the figure directory of the same sex and the other class.
func counterpartDir(dir data.FigureDir) data.FigureDir {
	switch dir {
	case data.FigureDirManFighter:
		return data.FigureDirManMage
	case data.FigureDirManMage:
		return data.FigureDirManFighter
	case data.FigureDirWomanFighter:
		return data.FigureDirWomanMage
	}
	return data.FigureDirWomanFighter
}

// layerDepth is the layer placement as a ModItem carries it: negative under the
// anchor, positive over it, zero for an ordinary item.
func layerDepth(layer string) int8 {
	switch layer {
	case mod.LayerUnder:
		return mapload.LayerUnder
	case mod.LayerOver:
		return mapload.LayerOver
	}
	return 0
}

// applyItemChange sets the numbers a change names on the row it names.
func applyItemChange(c mod.ItemChange, classes []*itemClass, t *mapload.Table) error {
	var found []*itemClass
	var rowIdx int
	for _, cl := range classes {
		if i := cl.lookup(c.Item); i != 0 {
			found = append(found, cl)
			rowIdx = i
		}
	}
	switch len(found) {
	case 0:
		return itemFileError(c.Mod, c.File, c.Line, "no row named %q in the armour, shield or weapon table", c.Item)
	case 1:
	default:
		return itemFileError(c.Mod, c.File, c.Line, "%q names a row in more than one item table", c.Item)
	}
	cl := found[0]
	if cl.name == "weapon" && (c.Defence != nil || c.Absorption != nil) {
		return itemFileError(c.Mod, c.File, c.Line, "%q is a weapon; a change can set its price and weight only", c.Item)
	}
	row, ok := cl.rows[rowIdx]
	if !ok {
		row = data.OverlayRow{Name: cl.base.EntryName(rowIdx), Params: cl.base.EntryParams(rowIdx)}
	}
	for _, s := range []struct {
		a data.ItemAttr
		v *int32
	}{{data.AttrPrice, c.Price}, {data.AttrWeight, c.Weight}, {data.AttrDefence, c.Defence}, {data.AttrAbsorption, c.Absorption}} {
		if s.v != nil {
			row.Params = data.WithReferenceColumn(row.Params, t.Shapes, t.Materials, s.a, *s.v)
		}
	}
	cl.rows[rowIdx] = row
	return nil
}

// resolveStandIn reads the stand-in of an item: an original item of the same
// slot, named as the item tables name it with an optional shape and material
// before it ("Soft Mail", "Hard Leather Soft Mail"). Without a shape and a
// material the first combination the row's own masks allow is taken, so the
// stand-in is an item the shipped game makes.
func resolveStandIn(r mod.ItemRow, class *itemClass, t *mapload.Table) (data.ItemCode, int, error) {
	return resolveOriginalItem(r, class, t, "stand-in", r.StandIn, r.StandInLine, r.SlotNo)
}

// resolveFigure reads the figure of an item: an original armour item of any
// slot whose worn picture the item is drawn with.
func resolveFigure(r mod.ItemRow, armors *itemClass, t *mapload.Table) (data.ItemCode, error) {
	code, _, err := resolveOriginalItem(r, armors, t, "figure", r.Figure, r.FigureLine, 0)
	return code, err
}

// resolveOriginalItem finds the original item a name stands for in class. A
// slot number other than zero is the slot the item must be worn in.
func resolveOriginalItem(r mod.ItemRow, class *itemClass, t *mapload.Table, what, name string, line, slotNo int) (data.ItemCode, int, error) {
	bad := func(format string, args ...any) error {
		return itemFileError(r.Mod, r.File, line, "%s %q: %s", what, name, fmt.Sprintf(format, args...))
	}
	rest := strings.Join(strings.Fields(name), " ")
	shape, material := -1, -1
	for i := 0; i < t.Shapes.Len(); i++ {
		if n := t.Shapes.EntryName(i); n != "" && strings.HasPrefix(rest, n+" ") {
			shape, rest = i, strings.TrimPrefix(rest, n+" ")
			break
		}
	}
	best := 0
	for i := 0; i < t.Materials.Len(); i++ {
		if n := t.Materials.EntryName(i); len(n) > best && strings.HasPrefix(rest, n+" ") {
			material, best = i, len(n)
		}
	}
	if material >= 0 {
		rest = strings.TrimPrefix(rest, t.Materials.EntryName(material)+" ")
	}
	row := class.lookup(rest)
	if row == 0 {
		return 0, 0, bad("the %s table has no row named %q", class.name, rest)
	}
	if class.name == "armour" && slotNo != 0 {
		p := class.base.EntryParams(row)
		if len(p) <= 4 || int(p[4]) != slotNo {
			return 0, 0, bad("%q is not an item of the %s slot", rest, r.Slot)
		}
	}
	masks, ok := class.base.(data.MaskTable)
	if !ok {
		return 0, 0, bad("the %s table carries no material masks", class.name)
	}
	raw := masks.EntryRaw(row)
	mask := func(s int) uint16 {
		if len(raw) < 2*s+2 {
			return 0
		}
		return binary.LittleEndian.Uint16(raw[2*s:])
	}
	pick := func(s, m int) bool { return s >= 0 && m >= 0 && mask(s)&(1<<uint(m)) != 0 }
	switch {
	case shape < 0 && material < 0:
	search:
		for s := 0; s < data.ShopShapes; s++ {
			for m := 0; m < data.ShopMaterials; m++ {
				if pick(s, m) {
					shape, material = s, m
					break search
				}
			}
		}
	case shape < 0:
		for s := 0; s < data.ShopShapes && shape < 0; s++ {
			if pick(s, material) {
				shape = s
			}
		}
	case material < 0:
		for m := 0; m < data.ShopMaterials && material < 0; m++ {
			if pick(shape, m) {
				material = m
			}
		}
	}
	if !pick(shape, material) {
		return 0, 0, bad("the game never makes %q in that shape and material", rest)
	}
	slotField := slotNo
	if slotField == 0 {
		slotField = int(class.base.EntryParams(row)[4])
	}
	return data.ComposeItemCode(material, slotField, shape, row), row, nil
}

// checkItemNumbers resolves the code through the tables the game reads and
// refuses an item whose numbers the table cannot state exactly.
func checkItemNumbers(r mod.ItemRow, code data.ItemCode, t *mapload.Table) error {
	var defence, absorption, weight int32
	if r.SlotNo == 2 {
		s, err := data.ShieldFromCode(code, t.Shapes, t.Materials, t.Shields)
		if err != nil {
			return itemFileError(r.Mod, r.File, r.Line, "the item does not resolve: %v", err)
		}
		defence, absorption, weight = s.Defence, s.Absorption, s.Weight
	} else {
		a, err := data.ArmorFromCode(code, t.Shapes, t.Materials, t.Armors)
		if err != nil {
			return itemFileError(r.Mod, r.File, r.Line, "the item does not resolve: %v", err)
		}
		defence, absorption, weight = a.Defence, a.Absorption, a.Weight
	}
	for _, c := range []struct {
		name      string
		got, want int32
	}{{"defence", defence, r.Defence}, {"absorption", absorption, r.Absorption}, {"weight", weight, r.Weight}} {
		if c.got != c.want {
			return itemFileError(r.Mod, r.File, r.Line, "the item table cannot state %s %d for this item; it holds %d", c.name, c.want, c.got)
		}
	}
	return nil
}

// modShopStock is the units the mods put on the armour shelf, one per item that
// asks for a place there. The town-arrival stock always holds them. A tavern
// restock (restocks above zero) offers each one with the chance a shipped
// armour row has of appearing in one generation, decided by a hash of the shop
// seed and the item code, so it takes no draw and the shelf stays reproducible.
func modShopStock(t *mapload.Table, seed int64, restocks uint64, poolSize int) []ShopItem {
	if t == nil {
		return nil
	}
	var out []ShopItem
	for _, m := range t.Mods.Items {
		if m.Stock && modStockOffered(seed, restocks, m.Code, poolSize) {
			out = append(out, shopItemFromInstance(sim.ItemInstance{Code: m.Code, Kind: 1, Price: m.Price}, 1))
		}
	}
	return out
}

// modStockChance is the chance that one given row of a pool of poolSize rows is
// among the draws of one generation: 1 - (1 - 1/poolSize)^draws, by repeated
// multiplication so every platform rounds alike.
func modStockChance(poolSize, draws int) float64 {
	if poolSize <= 1 || draws <= 0 {
		return 1
	}
	miss := 1.0
	for i := 0; i < draws; i++ {
		miss *= 1 - 1/float64(poolSize)
	}
	return 1 - miss
}

func modStockOffered(seed int64, restocks uint64, code uint16, poolSize int) bool {
	if restocks == 0 {
		return true
	}
	h := uint64(seed)
	for _, b := range []byte{byte(code), byte(code >> 8), 0x6d} {
		h ^= uint64(b)
		h *= 1099511628211
	}
	h ^= h >> 29
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 32
	return float64(h>>11)/float64(1<<53) < modStockChance(poolSize, shopShelfDraws[ShelfArmour])
}
