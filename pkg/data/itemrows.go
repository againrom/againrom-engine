package data

import "againrom/pkg/namefold"

// ItemRowNames are the two table rows an item code names by name: the kind
// row in the class's collection (field D) and the material row (field A).
type ItemRowNames struct {
	// Equipment says the code's field B is one of the twelve equipment slots,
	// so its kind and material rows are read at all.
	Equipment bool
	// Kind is the name of the code's row in the class's collection (Weapons
	// for slot 1, Shields for slot 2, Armors for the rest); empty when the
	// class has no collection or the row is past its end.
	Kind string
	// Material is the name of the code's Materials row; empty when the row is
	// past the table's end or carries no name.
	Material string
	// KindKey and MaterialKey are the word keys of those names (see RowKey).
	KindKey, MaterialKey string
}

// ItemRows is the one reader of an item code's kind row and material row
// names. A collection or table that is nil reads as no rows.
func ItemRows(c ItemCode, weapons, shields, armors Collection, materials ScaleTable) ItemRowNames {
	slot, ok := EquipSlotFor(c)
	if !ok {
		return ItemRowNames{}
	}
	rows := armors
	switch slot {
	case weaponItemClass:
		rows = weapons
	case shieldItemClass:
		rows = shields
	}
	out := ItemRowNames{Equipment: true}
	if rows != nil && c.D() < rows.Len() {
		out.Kind = rows.EntryName(c.D())
		out.KindKey = RowKey(out.Kind)
	}
	if materials != nil && c.A() < materials.Len() {
		out.Material = materials.EntryName(c.A())
		out.MaterialKey = RowKey(out.Material)
	}
	return out
}

// RowKey is the word key of a table row name written as capitalised ASCII
// words ("Two Handed Sword" is "twohandedsword"), or "" for any other
// spelling, so a name that differs from the shipped one in case, spacing or
// script has none.
func RowKey(name string) string {
	if name == "" {
		return ""
	}
	key := make([]byte, 0, len(name))
	start := true
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c == ' ' && !start:
			start = true
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z' && !start:
			key = append(key, namefold.ArchiveByte(c))
			start = false
		default:
			return ""
		}
	}
	if start {
		return ""
	}
	return string(key)
}
