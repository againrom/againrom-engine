package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"strings"
)

// SHOP-GEN-005 supplies names, not a numeric row range. Resolve each against
// the installed collection so a reordered custom table keeps its identity.
var shopPotionNames = [...]string{
	"Potion Health Regeneration", "Potion Medium Healing", "Potion Big Healing",
	"Potion Mana Regeneration", "Potion Medium Mana", "Potion Big Mana",
}

// SHOP-CONSUME-073: the random value is spell power, not stack quantity.
// Base price admission precedes scaling; the scaled price is not re-filtered.
func shopScrollPool(t *mapload.Table, ceiling int32, r shopDrawSource) []ShopItem {
	if t == nil || t.Spells == nil || t.MagicItems == nil || r == nil || ceiling <= 0 {
		return nil
	}
	schools := [6]string{"", "Fire", "Water", "Air", "Earth", "Astral"}
	var out []ShopItem
	for spell := 1; spell < t.Spells.Len() && spell <= 27; spell++ {
		params := t.Spells.EntryParams(spell)
		if len(params) <= 20 || params[20] <= 0 || params[20] > ceiling || params[2] <= 0 || params[2] >= 6 {
			continue
		}
		var code uint16
		for row := 1; row < t.MagicItems.Len() && row <= 255; row++ {
			if strings.ReplaceAll(t.MagicItems.EntryName(row), "_", " ") == "Scroll "+schools[params[2]] {
				code = uint16(0xe00 | row)
				break
			}
		}
		if code == 0 {
			continue
		}
		limit := min(int64(100), 10*(int64(ceiling)/int64(params[20])))
		power, ok := shopDrawInclusive(r, 1, int(limit))
		if !ok {
			continue
		}
		price := int64(params[20]) * (int64(power)/10 + 1)
		if price > 1<<31-1 {
			continue
		}
		item := sim.ItemInstance{Code: code, Kind: 4, Price: int32(price), Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(spell) | uint32(power)<<16}}}
		out = append(out, shopItemFromInstance(item, 1))
	}
	return out
}

// The constructor's school boundary exclusions are ROM1 data-independent
// rules. Slow lies outside the 1..27 generator, even on a longer custom table.
func shopBookSpellAdmitted(id int) bool {
	return id >= 2 && id <= 27 && id != 6 && id != 11 && id != 12 && id != 17 && id != 18 && id != 27
}

func shopPotionStock(t *mapload.Table, r shopDrawSource) []ShopItem {
	if t == nil || t.MagicItems == nil || r == nil {
		return nil
	}
	var out []ShopItem
	for _, name := range shopPotionNames {
		// Packed class-14 codes have one byte for the definition index.
		for row := 1; row < t.MagicItems.Len() && row <= 255; row++ {
			if t.MagicItems.EntryName(row) != name {
				continue
			}
			item := mapload.ItemInstanceFromCode(uint16(0x0e00|row), t)
			if len(item.Effects) != 0 && item.Price >= 0 {
				count, _ := shopDrawInclusive(r, 51, 100)
				out = append(out, shopItemFromInstance(item, int32(count)))
			}
			break
		}
	}
	return out
}
