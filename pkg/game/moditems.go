package game

import (
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
	// shop is the code class a row of the table composes, as the shop walk
	// composes it.
	shop data.ShopClass
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
	mask := data.ItemMasks(masks, row)
	pick := func(s, m int) bool {
		return s >= 0 && s < data.ShopShapes && m >= 0 && mask[s]&(1<<uint(m)) != 0
	}
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
