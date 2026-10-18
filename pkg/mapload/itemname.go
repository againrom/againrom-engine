package mapload

import "againrom/pkg/data"

// ShippedItemName is the install's stored name for code, and whether the
// shipped item-name table holds one.
func ShippedItemName(t *Table, code data.ItemCode) (string, bool) {
	if t == nil {
		return "", false
	}
	return t.Names.NameFor(code)
}

// ItemDisplayName is the one route that names an item for the player, in
// both games: the name the item's own saved record carries, else the
// install's shipped name for its code, else fallback, the caller's name
// composed from the item's table rows.
func ItemDisplayName(t *Table, code data.ItemCode, saved, fallback string) string {
	if saved != "" {
		return saved
	}
	if name, ok := ShippedItemName(t, code); ok {
		return name
	}
	return fallback
}
