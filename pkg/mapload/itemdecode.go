package mapload

import (
	"fmt"

	"againrom/pkg/data"
)

// ItemDecoding is what an item code says about the item's kind and material,
// read from the table: the kind row's name, the material row's name and, for
// a carried item, its carried kind. Kind is empty when the code names no row.
// KindKey and MaterialKey are the stable word keys of those names ("" when a
// name has none), the same in every language.
type ItemDecoding struct {
	Kind        string
	Material    string
	KindKey     string
	MaterialKey string
	// Equipment says the code is worn (field B is an equipment slot).
	Equipment bool
	// Carried is the carried item kind (3 potion, 4 scroll, 5 book, or the
	// instance's other kind) of a class-14 code; 0 for any other code.
	Carried uint8
}

// carriedKindRows are the kind rows of the carried item kinds, with their
// word keys; every other carried kind is the last row.
var carriedKindRows = map[uint8]struct{ name, key string }{
	3: {"Potions", "potions"},
	4: {"Scrolls", "scrolls"},
	5: {"Books", "books"},
}

const (
	carriedOtherName = "Other items"
	carriedOtherKey  = "otheritems"
	noMaterialName   = "None"
	noMaterialKey    = "none"
)

// DecodeItem is the one reader of an item code's kind and material rows, in
// both games. An equipment code is named by its class collection's row and its
// Materials row ("#n" when that row has no name); a carried code by its
// carried kind; any other code, or a nil table, has no kind and the material
// "None".
func DecodeItem(t *Table, code data.ItemCode) ItemDecoding {
	d := ItemDecoding{Material: noMaterialName, MaterialKey: noMaterialKey}
	if t == nil {
		return d
	}
	if rows := data.ItemRows(code, t.Weapons, t.Shields, t.Armors, t.Materials); rows.Equipment {
		d.Equipment = true
		d.Kind = rows.Kind
		d.KindKey = rows.KindKey
		d.Material = rows.Material
		d.MaterialKey = rows.MaterialKey
		if rows.Material == "" {
			d.Material, d.MaterialKey = fmt.Sprintf("#%d", code.A()), ""
		}
		return d
	}
	if code.B() == data.ItemClassCarried {
		d.Carried = ItemInstanceFromCode(uint16(code), t).Kind
		if row, ok := carriedKindRows[d.Carried]; ok {
			d.Kind, d.KindKey = row.name, row.key
		} else {
			d.Kind, d.KindKey = carriedOtherName, carriedOtherKey
		}
	}
	return d
}
