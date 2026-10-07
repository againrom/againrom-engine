package mapload

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

var effectKinds = map[string]uint8{
	"price": 1, "body": 2, "mind": 3, "reaction": 4, "spirit": 5,
	"health": 6, "healthmax": 7, "healthregeneration": 8,
	"mana": 9, "manamax": 10, "manaregeneration": 11,
	"tohit": 12, "damagemin": 13, "damagemax": 14,
	"defence": 15, "absorbtion": 16, "speed": 17, "rotationspeed": 18,
	"scanrange": 19, "protection0": 20, "protectionfire": 21,
	"protectionwater": 22, "protectionair": 23, "protectionearth": 24,
	"protectionastral": 25, "fighterskill0": 26, "skillblade": 27,
	"skillaxe": 28, "skillbludgeon": 29, "skillpike": 30,
	"skillshooting": 31, "mageskill0": 32, "skillfire": 33,
	"skillwater": 34, "skillair": 35, "skillearth": 36,
	"skillastral": 37, "itemlore": 38, "magiclore": 39,
	"creaturelore": 40, "castspell": 41, "teachspell": 42,
	"damage": 43, "damagefire": 44, "damagewater": 45,
	"damageair": 46, "damageearth": 47, "damageastral": 48,
	"damagebonus": 49,
}

// ParseItemCell separates the first-brace item head from its ordered effect
// tail. Unknown keys, malformed elements and unknown spells reject only their
// own element. A numeric operand whose original CRT result is not established
// is reported instead of being assigned a value this implementation invented.
func ParseItemCell(cell string, table *Table) (head string, effects []sim.ItemEffect, rejected int, err error) {
	open := strings.IndexByte(cell, '{')
	if open < 0 {
		return strings.TrimRightFunc(cell, unicode.IsSpace), nil, 0, nil
	}
	head = strings.TrimRightFunc(cell[:open], unicode.IsSpace)
	tail := cell[open+1:]
	if close := strings.IndexByte(tail, '}'); close >= 0 {
		tail = tail[:close]
	}
	for _, raw := range strings.Split(tail, ",") {
		eq := strings.IndexByte(raw, '=')
		if eq < 0 {
			rejected++
			continue
		}
		kind, ok := effectKinds[strings.ToLower(strings.TrimSpace(raw[:eq]))]
		if !ok || kind == 0 {
			rejected++
			continue
		}
		effect, ok, parseErr := parseItemEffect(kind, raw[eq+1:], table)
		if parseErr != nil {
			return head, effects, rejected, parseErr
		}
		if !ok {
			rejected++
			continue
		}
		effects = append(effects, effect)
	}
	return head, effects, rejected, nil
}

func parseItemEffect(kind uint8, value string, table *Table) (sim.ItemEffect, bool, error) {
	operandText, suffix := value, ""
	if colon := strings.IndexByte(value, ':'); colon >= 0 {
		operandText, suffix = value[:colon], value[colon+1:]
	}
	mode, ticks, err := effectMode(suffix)
	if err != nil {
		return sim.ItemEffect{}, false, err
	}
	effect := sim.ItemEffect{Kind: kind, Mode: mode}
	switch {
	case kind <= 40 || kind == 49:
		n, ok, err := signedDecimalPrefix(operandText)
		if err != nil {
			return effect, false, fmt.Errorf("effect kind %d operand %q: %w", kind, operandText, err)
		}
		if !ok {
			return effect, false, fmt.Errorf("effect kind %d operand %q has an indeterminate original value", kind, operandText)
		}
		if mode == 0 || mode == 8 {
			if n < -1<<31 || n > 1<<31-1 {
				return effect, false, fmt.Errorf("effect kind %d operand %q is outside i32", kind, operandText)
			}
			effect.Operand = uint32(int32(n))
		} else {
			effect.Operand = uint32(uint16(int16(n))) | uint32(ticks)<<16
		}
	case kind == 41:
		spell, ok := SpellIDByToken(table, operandText)
		if !ok {
			return effect, false, nil
		}
		power, err := unsignedPowerPrefix(suffix)
		if err != nil {
			return effect, false, err
		}
		effect.Operand = uint32(spell) | uint32(uint16(int16(power)))<<16
	case kind == 42:
		spell, ok := SpellIDByToken(table, operandText)
		if !ok {
			return effect, false, nil
		}
		effect.Operand = uint32(spell) | uint32(ticks)<<16
	case kind >= 43 && kind <= 48:
		dash := strings.IndexByte(operandText, '-')
		if dash < 0 {
			return effect, true, nil
		}
		low, lowOK, lowErr := signedDecimalPrefix(operandText[:dash])
		high, highOK, highErr := signedDecimalPrefix(operandText[dash+1:])
		if lowErr != nil || highErr != nil || !lowOK || !highOK {
			return effect, false, fmt.Errorf("effect kind %d range %q has an indeterminate original value", kind, operandText)
		}
		base := uint8(low)
		spread := uint8(high - int64(base))
		effect.Operand = uint32(base) | uint32(spread)<<8 | uint32(ticks)<<16
	}
	return effect, true, nil
}

func effectMode(suffix string) (uint8, uint16, error) {
	if suffix == "permanent" || suffix == "" {
		return 0, 0, nil
	}
	if suffix == "singleuse" {
		return 8, 0, nil
	}
	for _, candidate := range []struct {
		name string
		mode uint8
	}{{"charges", 4}, {"duration", 1}, {"continuous", 2}} {
		at := strings.Index(suffix, candidate.name)
		if at < 0 {
			continue
		}
		count, ok, err := signedDecimalPrefix(suffix[at+len(candidate.name):])
		if err != nil || !ok {
			return 0, 0, fmt.Errorf("effect mode %q has an indeterminate count", suffix)
		}
		return candidate.mode, uint16(count << 4), nil
	}
	return 0, 0, nil
}

func signedDecimalPrefix(s string) (int64, bool, error) {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	end := 0
	if end < len(s) && (s[end] == '+' || s[end] == '-') {
		end++
	}
	firstDigit := end
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == firstDigit {
		if s == "" {
			return 0, true, nil
		}
		return 0, false, nil
	}
	n, err := strconv.ParseInt(s[:end], 10, 64)
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

func unsignedPowerPrefix(s string) (int64, error) {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	end := 0
	if end < len(s) && s[end] == '+' {
		end++
	}
	firstDigit := end
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == firstDigit {
		return 0, nil
	}
	n, err := strconv.ParseInt(s[:end], 10, 16)
	if err != nil {
		return 0, fmt.Errorf("cast power %q is outside i16", s[:end])
	}
	return n, nil
}

func itemPrice(code data.ItemCode, collection data.Collection, table *Table) int32 {
	if table == nil || collection == nil || code.D() <= 0 || code.D() >= collection.Len() {
		return 0
	}
	params := collection.EntryParams(code.D())
	if len(params) <= 2 {
		return 0
	}
	return data.ItemPrice(params[2], table.Shapes, table.Materials, code.C(), code.A())
}

func equipmentInstance(code data.ItemCode, effects []sim.ItemEffect, collection data.Collection, table *Table) sim.ItemInstance {
	item := sim.ItemInstance{
		Code: uint16(code), Effects: append([]sim.ItemEffect(nil), effects...),
		Price: itemPrice(code, collection, table),
	}
	switch code.B() {
	case 1:
		item.Kind = 2
	case 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12:
		item.Kind = 1
	}
	item.Price = storedItemPrice(item.Price, effects, table)
	return item
}

// storedItemPrice applies the original object's effect-price walk to a base
// equipment price. Ordinary effects contribute their Magic-row point cost as
// one aggregate; castSpell contributes its spell-row scalar separately. The
// arithmetic is deliberately on the complete ordered effect list even though
// price itself is order-independent: this is the same instance list later
// serialized and shown in inventory, not a parallel enchantment summary.
func storedItemPrice(base int32, effects []sim.ItemEffect, table *Table) int32 {
	var points int32
	var cast int64
	var direct int64
	for _, effect := range effects {
		if effect.Kind == 1 {
			direct += int64(itemEffectScalar(effect))
			continue
		}
		if effect.Kind == 41 {
			if table == nil || table.Spells == nil {
				continue
			}
			spell := int(uint16(effect.Operand))
			if spell < 0 || spell >= table.Spells.Len() {
				continue
			}
			params := table.Spells.EntryParams(spell)
			if len(params) <= 20 || params[20] <= 0 {
				continue
			}
			power := int32(int16(effect.Operand >> 16))
			if power < 0 {
				continue
			}
			rank := math.Log(float64(power)/30+1) / math.Log(1.2)
			cast += int64(10 * float64(params[20]) * math.Pow(2, rank))
			continue
		}
		if table == nil || table.Magic == nil || int(effect.Kind) >= table.Magic.Len() {
			continue
		}
		params := table.Magic.EntryParams(int(effect.Kind))
		if len(params) == 0 || params[0] <= 0 {
			continue
		}
		magnitude := itemEffectScalar(effect)
		if effect.Kind >= 44 && effect.Kind <= 48 {
			magnitude = int32(uint8(effect.Operand)) + int32(uint8(effect.Operand>>8))
		}
		points += magnitude * params[0]
	}
	var ordinary int64
	if points > 0 {
		n := float64(points)
		ordinary = int64((math.Pow(1.5, n/70) + 1) * n * 50)
	}
	price := int64(base) + direct + ordinary + cast
	if price >= 9_999_999 {
		return 9_999_999
	}
	if price < -1<<31 {
		return -1 << 31
	}
	return int32(price)
}

func baseItemPrice(code data.ItemCode, table *Table) (int32, bool) {
	if table == nil {
		return 0, false
	}
	var collection data.Collection
	switch code.B() {
	case 1:
		collection = table.Weapons
	case 2:
		collection = table.Shields
	case 3, 4, 5, 6, 7, 8, 9, 10, 11, 12:
		collection = table.Armors
	case data.ItemClassCarried:
		row := int(uint8(code)) // ITEM-CODE-029: class 14 uses the whole low byte.
		if table.MagicItems == nil || row <= 0 || row >= table.MagicItems.Len() {
			return 0, false
		}
		params := table.MagicItems.EntryParams(row)
		if len(params) == 0 {
			return 0, false
		}
		return params[0], true
	default:
		return 0, false
	}
	if table.Shapes == nil || table.Materials == nil || collection == nil || code.D() <= 0 || code.D() >= collection.Len() {
		return 0, false
	}
	params := collection.EntryParams(code.D())
	if len(params) <= 2 {
		return 0, false
	}
	return data.ItemPrice(params[2], table.Shapes, table.Materials, code.C(), code.A()), true
}

// SpellBookCost reads the one installed Spells cell used as a readable
// book's stored value. A missing row or a row too short to carry slot 21 is
// not a zero-price book: it is an incomplete table and reports no answer.
func SpellBookCost(spell uint16, table *Table) (int32, bool) {
	if table == nil || table.Spells == nil || spell == 0 || int(spell) >= table.Spells.Len() {
		return 0, false
	}
	params := table.Spells.EntryParams(int(spell))
	if len(params) <= 21 {
		return 0, false
	}
	return params[21], true
}

func exactSpellBook(item sim.ItemInstance) (uint16, bool) {
	if item.Kind != 5 || len(item.Effects) != 1 {
		return 0, false
	}
	effect := item.Effects[0]
	if effect.Kind != 42 || effect.Mode != 0 || effect.Operand == 0 || effect.Operand >= 32 {
		return 0, false
	}
	return uint16(effect.Operand), true
}

// RepriceItemInstance recomputes an item's stored value from its live base row
// and complete ordered effect list. An incomplete table preserves the stored
// value instead of replacing it with a guessed zero.
func RepriceItemInstance(item sim.ItemInstance, table *Table) int32 {
	if item.Kind == 5 {
		if len(item.Effects) == 0 {
			return 0
		}
		price, ok := SpellBookCost(uint16(item.Effects[0].Operand), table)
		if !ok {
			return item.Price
		}
		return price
	}
	base, ok := baseItemPrice(data.ItemCode(item.Code), table)
	if !ok {
		return item.Price
	}
	if item.Kind == 3 {
		return base // ITEM-MAGVAL-090: ordinary MagicItems keep their row value.
	}
	if item.Kind == 4 {
		var cast []sim.ItemEffect
		for _, e := range item.Effects {
			if e.Kind == 41 {
				cast = append(cast, e)
			}
		}
		if value := storedItemPrice(0, cast, table); value != 0 {
			return value
		}
		return base
	}
	return storedItemPrice(base, item.Effects, table)
}

func itemEquipmentCodes(items [sim.EquipSlots]sim.ItemInstance) (codes [sim.EquipSlots]uint16) {
	for i := range items {
		codes[i] = items[i].Code
	}
	return codes
}

func itemInstanceCodes(items []sim.ItemInstance) []uint16 {
	if len(items) == 0 {
		return nil
	}
	codes := make([]uint16, len(items))
	for i := range items {
		codes[i] = items[i].Code
	}
	return codes
}

func itemEquipmentEmpty(items [sim.EquipSlots]sim.ItemInstance) bool {
	for _, item := range items {
		if !item.Empty() {
			return false
		}
	}
	return true
}

func cloneItemInstances(items []sim.ItemInstance) []sim.ItemInstance {
	if len(items) == 0 {
		return nil
	}
	out := make([]sim.ItemInstance, len(items))
	for i := range items {
		out[i] = items[i].Clone()
	}
	return out
}

func cloneItemEquipment(items [sim.EquipSlots]sim.ItemInstance) (out [sim.EquipSlots]sim.ItemInstance) {
	for i := range items {
		out[i] = items[i].Clone()
	}
	return out
}

// ItemInstanceFromCode constructs the complete base object named by a packed
// code. Equipment price uses its shape/material scale; class 14 uses its signed
// MagicItems value and name-derived kind.
func ItemInstanceFromCode(code uint16, table *Table) sim.ItemInstance {
	item := sim.PlainItem(code)
	c := data.ItemCode(code)
	switch c.B() {
	case 1:
		item.Kind = 2
	case 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12:
		item.Kind = 1
	}
	if table == nil {
		return item
	}
	switch {
	case c.B() == 1:
		item.Price = itemPrice(c, table.Weapons, table)
		return item
	case c.B() == 2:
		item.Price = itemPrice(c, table.Shields, table)
		return item
	case c.B() >= 3 && c.B() <= 12:
		item.Price = itemPrice(c, table.Armors, table)
		return item
	}
	row := int(uint8(c))
	if c.B() != data.ItemClassCarried || table.MagicItems == nil || row <= 0 || row >= table.MagicItems.Len() {
		return item
	}
	name := table.MagicItems.EntryName(row)
	switch {
	case strings.HasPrefix(name, "Potion"):
		item.Kind = 3
	case strings.HasPrefix(name, "Scroll"):
		item.Kind = 4
	case strings.HasPrefix(name, "Book"):
		item.Kind = 5
	}
	params := table.MagicItems.EntryParams(row)
	if len(params) > 0 {
		item.Price = params[0]
	}
	if item.Kind == 3 {
		// The installed Effects string owns the potion's payload. Reuse the
		// ordered item grammar; an indeterminate operand leaves this item
		// unusable instead of constructing a partly applied potion.
		values := table.MagicItems.EntryStrings(row)
		if len(values) > 0 {
			_, effects, rejected, err := ParseItemCell("{"+values[0]+"}", table)
			if err == nil && rejected == 0 {
				item.Effects = effects
			}
		}
	}
	return item
}

func resolveWeaponCell(cell string, table *Table) (data.Weapon, sim.ItemInstance, error) {
	head, effects, _, err := ParseItemCell(cell, table)
	if err != nil {
		return data.Weapon{}, sim.ItemInstance{}, err
	}
	shapes, materials, weapons, ok := table.items()
	if !ok {
		return data.Weapon{}, sim.ItemInstance{}, fmt.Errorf("item table is incomplete")
	}
	w, err := data.ResolveWeapon(head, shapes, materials, weapons)
	if err != nil {
		return data.Weapon{}, sim.ItemInstance{}, err
	}
	for _, effect := range effects {
		if effect.Kind != 41 {
			continue
		}
		spellID := int(uint16(effect.Operand))
		if table.Spells != nil && spellID > 0 && spellID < table.Spells.Len() {
			w.SpellName = strings.ReplaceAll(table.Spells.EntryName(spellID), " ", "_")
			w.SpellPower = int32(int16(effect.Operand >> 16))
		}
		break
	}
	return w, equipmentInstance(w.Code, effects, table.Weapons, table), nil
}

func resolveShieldCell(cell string, table *Table) (data.Shield, sim.ItemInstance, error) {
	head, effects, _, err := ParseItemCell(cell, table)
	if err != nil {
		return data.Shield{}, sim.ItemInstance{}, err
	}
	shapes, materials, shields, ok := table.shieldItems()
	if !ok {
		return data.Shield{}, sim.ItemInstance{}, fmt.Errorf("shield table is incomplete")
	}
	s, err := data.ResolveShield(head, shapes, materials, shields)
	if err != nil {
		return data.Shield{}, sim.ItemInstance{}, err
	}
	return s, equipmentInstance(s.Code, effects, table.Shields, table), nil
}

func resolveArmorCell(cell string, table *Table) (data.Armor, sim.ItemInstance, error) {
	head, effects, _, err := ParseItemCell(cell, table)
	if err != nil {
		return data.Armor{}, sim.ItemInstance{}, err
	}
	shapes, materials, armors, ok := table.armorItems()
	if !ok {
		return data.Armor{}, sim.ItemInstance{}, fmt.Errorf("armor table is incomplete")
	}
	a, err := data.ResolveArmor(head, shapes, materials, armors)
	if err != nil {
		return data.Armor{}, sim.ItemInstance{}, err
	}
	return a, equipmentInstance(a.Code, effects, table.Armors, table), nil
}
