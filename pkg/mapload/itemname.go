package mapload

import (
	"fmt"
	"sort"
	"strings"

	"againrom/pkg/data"
)

// ItemNameForm is what an item with neither a saved nor a shipped name is
// called. The resolver reads it; no caller composes that name itself.
type ItemNameForm uint8

const (
	// ItemNamePlayer is the name the game's composer row builds.
	ItemNamePlayer ItemNameForm = iota
	// ItemNameWeapon is a current weapon's name, which a SAVE carries: the
	// composer row's, except that a code no weapon row resolves is named by
	// nothing rather than by its number.
	ItemNameWeapon
	// ItemNameEditorList is the map editor's choice label, "item 0x0101".
	ItemNameEditorList
	// ItemNameEditorRecord is the map editor's sack record line, which states
	// a code the shipped table does not name as unresolved.
	ItemNameEditorRecord
)

// ItemRowTable is the second game's data.bin table an item row is in.
type ItemRowTable uint8

const (
	ItemRowNone ItemRowTable = iota
	ItemRowArmor
	ItemRowShield
	ItemRowWeapon
	ItemRowMagic
	ItemRowBook
)

// ItemRows are a second-game item's data.bin rows: its equipment or magic
// item row with its material and class, or a book's spell.
type ItemRows struct {
	Table                       ItemRowTable
	Row, Material, Class, Spell int
}

// ItemSubject is the item a name is asked for. Keyless says it has no line
// in the shipped item-name table, so Code is not looked up there.
type ItemSubject struct {
	Code    data.ItemCode
	Saved   string
	Keyless bool
	Rows    *ItemRows
	Form    ItemNameForm
}

// ItemNamer is a game's composer row: the name lines of an item with no
// saved or shipped name.
type ItemNamer func(t *Table, s ItemSubject) []string

// ItemName is a resolved name. Known says it is the item's saved or shipped
// name; Lines holds more than one line only for a composed second-game row
// name (row, material, class).
type ItemName struct {
	Lines []string
	Known bool
}

// Text is the name on one line.
func (n ItemName) Text() string { return strings.Join(n.Lines, " ") }

// ShippedItemName is the install's stored name for code, and whether the
// shipped item-name table holds one.
func ShippedItemName(t *Table, code data.ItemCode) (string, bool) {
	if t == nil {
		return "", false
	}
	return t.Names.NameFor(code)
}

// ShippedItemCode is the lowest nonzero code whose shipped name is name.
func ShippedItemCode(t *Table, name string) (data.ItemCode, bool) {
	for code := 1; code <= 65535; code++ {
		if display, ok := ShippedItemName(t, data.ItemCode(code)); ok && display == name {
			return data.ItemCode(code), true
		}
	}
	return 0, false
}

// ShippedItemCodes are the codes the shipped item-name table holds, in
// ascending order.
func ShippedItemCodes(t *Table) []data.ItemCode {
	if t == nil {
		return nil
	}
	out := make([]data.ItemCode, 0, len(t.Names))
	for code := range t.Names {
		out = append(out, code)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ResolveItemName is the one route that names an item to a player or a map
// maker, in both games: the name the item's own saved record carries, else
// the install's shipped name for its code, else the subject's form, whose
// player and weapon names the table's composer row builds. A subject with
// rows reads an empty shipped line as none.
func ResolveItemName(t *Table, s ItemSubject) ItemName {
	if s.Saved != "" {
		return ItemName{Lines: []string{s.Saved}, Known: true}
	}
	if !s.Keyless {
		if name, ok := ShippedItemName(t, s.Code); ok && (name != "" || s.Rows == nil) {
			return ItemName{Lines: []string{name}, Known: true}
		}
	}
	switch s.Form {
	case ItemNameEditorList:
		return ItemName{Lines: []string{fmt.Sprintf("item %#04x", uint16(s.Code))}}
	case ItemNameEditorRecord:
		return ItemName{Lines: []string{"unresolved item name"}}
	}
	return ItemName{Lines: t.itemNamer()(t, s)}
}

// ItemDisplayName is ResolveItemName's name on one line.
func ItemDisplayName(t *Table, s ItemSubject) string {
	return ResolveItemName(t, s).Text()
}

// itemNamer is the table's composer row. A table no game read names an item
// by the rows its subject carries, else by its code.
func (t *Table) itemNamer() ItemNamer {
	if t == nil || t.ItemNamer == nil {
		return RowItemNamer
	}
	return t.ItemNamer
}

// CodeItemNamer is the first game's composer row: the weapon name the code
// resolves to, else the code's own number.
func CodeItemNamer(t *Table, s ItemSubject) []string {
	if t != nil {
		if w, err := data.WeaponFromCode(s.Code, t.Shapes, t.Materials, t.Weapons); err == nil {
			return []string{w.Name}
		}
	}
	if s.Form == ItemNameWeapon {
		return []string{""}
	}
	return []string{s.Code.Name()}
}

// RowItemNamer is the second game's composer row (DIV-3046): a book is its
// spell's data.bin name, a magic item its row's, equipment its row's,
// material's and class's, one line each; a subject without rows is named as
// CodeItemNamer names it.
func RowItemNamer(t *Table, s ItemSubject) []string {
	if s.Rows == nil {
		return CodeItemNamer(t, s)
	}
	if t == nil {
		return []string{""}
	}
	r := s.Rows
	switch r.Table {
	case ItemRowBook:
		return []string{rowName(t.Spells, r.Spell)}
	case ItemRowMagic:
		return []string{rowName(t.MagicItems, r.Row)}
	}
	rows := map[ItemRowTable]data.Collection{ItemRowArmor: t.Armors, ItemRowShield: t.Shields, ItemRowWeapon: t.Weapons}
	return []string{rowName(rows[r.Table], r.Row), scaleName(t.Materials, r.Material), scaleName(t.Shapes, r.Class)}
}

func rowName(c data.Collection, i int) string {
	if c == nil || i < 0 || i >= c.Len() {
		return ""
	}
	return c.EntryName(i)
}

func scaleName(c data.ScaleTable, i int) string {
	if c == nil || i < 0 || i >= c.Len() {
		return ""
	}
	return c.EntryName(i)
}
