package secondshop

import "againrom/pkg/shopstock"

func (t Tables) ShownPrice(it Item) int32 {
	key, ok := t.NameKey(it)
	if ok && len(it.Effects) == 0 {
		if int(key>>8)&0xf == partMagic && t.MagicItems != nil {
			return param(t.MagicItems, int(key&0xff), colMagic)
		}
		if r, found := t.ItemRecords[key]; found && r.HasPrice {
			return int32(r.Price)
		}
	}
	return it.Price
}

func (t Tables) ShownFlag(it Item) bool {
	key, _ := t.NameKey(it)
	if it.Table == Book {
		key = uint16(partMagic<<8 | bookRow | it.Spell)
		if t.TemplateCode != nil {
			key = t.TemplateCode(key)
		}
	}
	return t.ItemRecords[key].Flags&2 != 0
}

func (t Tables) ShownKind(it Item) int {
	if it.Table == Book {
		return partMagic
	}
	key, _ := t.NameKey(it)
	return int(key>>8) & 0xf
}

func shownGroup(it Item) int {
	switch r := it.Row & 0x1f; {
	case r >= 1 && r <= 6:
		return 1
	case r >= 18 && r <= 19:
		return 2
	case r >= 15 && r <= 17:
		return 3
	case r >= 7 && r <= 12:
		return 5
	case r >= 20 && r <= 22:
		return 6
	default:
		return 7
	}
}

// Sort orders a received shelf. Magic lists keep equal items in their input
// order because their server sort reads unknown stack slots (DIV-4328).
func (t Tables) Sort(list []Item) {
	t.shownOrder(list).Sort(list)
}

func (t Tables) Order(list []Item) []int { return t.shownOrder(list).Indices(list) }

func (t Tables) shownOrder(list []Item) shopstock.ShelfOrder[Item] {
	known := true
	for _, it := range list {
		if it.Table == Magic || it.Table == Book {
			known = false
			break
		}
	}
	return shopstock.ShelfOrder[Item]{Flag: t.ShownFlag, Kind: t.ShownKind, Group: shownGroup,
		Price: func(it Item) int64 { return int64(t.ShownPrice(it)) }, CRT: known}
}
