package game

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Inventory popups describe decoded item statistics and live or stored weapon
// spell characteristics. Item instances also state their ordered effects.
// These helpers return plain text for pkg/ui to draw.

// itemInfoLines describes a code's name and decoded equipment statistics.
func itemInfoLines(code data.ItemCode, table *mapload.Table) []string {
	return itemInfoLinesWithWeaponDamage(code, table, nil)
}

// itemInstanceInfoLines is the instance-aware popup boundary. The legacy
// code-only helper above remains useful for callers that genuinely hold only
// a projection; every inventory and shop caller that owns an ItemInstance
// comes through here so ordered effects cannot disappear in UI.
func itemInstanceInfoLines(item sim.ItemInstance, table *mapload.Table, installed ...ui.Words) []string {
	return itemInstanceInfoLinesWithWeaponDamage(item, table, nil, installed...)
}

// weaponCastResolver answers the castSpell Effect a weapon code carries when
// the caller holds the code without its Effects: the spell id and the stored
// power. It names the item's own Effect, never the wearer's state, so a worn,
// a carried and a shelved weapon describe one text (ITEM-211).
type weaponCastResolver func(data.ItemCode) (spell uint16, power int32, ok bool)

// liveWeaponCast answers the castSpell Effect of the weapon entity id wears
// when code is that weapon. A worn item kept without its Effects falls back to
// the entity's item-borne spell pair; an innate spell has no Weapon to describe.
func liveWeaponCast(w *sim.World, id sim.EntityID, code data.ItemCode) (uint16, int32, bool) {
	if w == nil {
		return 0, 0, false
	}
	worn, ok := w.EquippedItems(id)
	if !ok || worn[0].Code != uint16(code) {
		return 0, 0, false
	}
	if spell, power, ok := worn[0].CastSpell(); ok {
		return spell, power, true
	}
	for _, entity := range w.Entities() {
		if entity.ID == id && entity.WeaponSpell != 0 && entity.WeaponSpellSource != sim.WeaponSpellInnate {
			return uint16(entity.WeaponSpell), entity.WeaponSpellLevel, true
		}
	}
	return 0, 0, false
}

// storedWeaponCast answers a party member's stored weapon identity, its
// castSpell name and power, for a code the pack or a town snapshot holds
// without its Effects.
func storedWeaponCast(code data.ItemCode, weapon *data.Weapon, table *mapload.Table) (uint16, int32, bool) {
	if weapon == nil || weapon.Code != code {
		return 0, 0, false
	}
	spellID, ok := mapload.SpellIDByToken(table, weapon.SpellName)
	if !ok {
		return 0, 0, false
	}
	return uint16(spellID), weapon.SpellPower, true
}

func weaponSpellName(spell uint16, table *mapload.Table) string {
	name := fmt.Sprintf("spell %d", spell)
	if table != nil && table.Spells != nil {
		id := int(spell)
		if id > 0 && id < table.Spells.Len() && table.Spells.EntryName(id) != "" {
			name = table.Spells.EntryName(id)
		}
	}
	return name
}

func itemInfoLinesWithWeaponDamage(code data.ItemCode, table *mapload.Table, resolve weaponCastResolver, installed ...ui.Words) []string {
	words := itemWords(installed)
	if resolve != nil {
		if spell, power, ok := resolve(code); ok {
			item := sim.ItemInstance{Code: uint16(code), Effects: []sim.ItemEffect{castSpellEffect(spell, power)}}
			if lines, ok := castWeaponInfoLines(item, table, words); ok {
				return lines
			}
		}
	}
	return itemBaseInfoLines(code, table, &words)
}

func castSpellEffect(spell uint16, power int32) sim.ItemEffect {
	return sim.ItemEffect{Kind: 41, Operand: uint32(spell) | uint32(uint16(int16(power)))<<16}
}

// itemBaseInfoLines is the name and the base characteristics of an item
// without a castSpell Effect.
func itemBaseInfoLines(code data.ItemCode, table *mapload.Table, words *ui.Words) []string {
	lines := []string{itemName(code, table)}
	if table == nil {
		return lines
	}
	if w, err := data.WeaponFromCode(code, table.Shapes, table.Materials, table.Weapons); err == nil {
		if base := weaponBaseLines(code, w, table, *words); len(base) > 0 {
			return append(lines, base...)
		}
		// A ranged row no character may wield (sutableFor 0) has no base pair
		// in the original. The owner's direction to state a ranged weapon's
		// range keeps that one line for it (DIV-1757).
		if (w.Ranged() || w.AttackType == data.SkillShoot) && uint8(w.Range) > 1 {
			lines = append(lines, fmt.Sprintf("#%s %d", words.ItemStats[38], uint8(w.Range)))
		}
		return lines
	}
	if a, err := data.ArmorFromCode(code, table.Shapes, table.Materials, table.Armors); err == nil {
		lines = append(lines, wornProtectionLines(a.Defence, a.Absorption, words)...)
	} else if table.Shields != nil {
		if s, err := data.ShieldFromCode(code, table.Shapes, table.Materials, table.Shields); err == nil {
			lines = append(lines, wornProtectionLines(s.Defence, s.Absorption, words)...)
		}
	}
	return lines
}

// weaponBaseLines are the base pairs a Weapon writes after its name, in the
// formatter's order (ITEM-211): only when sutableFor has bit 0 (a Staff and a
// Shaman Staff have 2 and get none), the damage pair as `[base, base+spread]`
// (ITEM-PANEL-022), then to-hit and defence when positive and the range when
// it exceeds 1, for any attack type. The elemental pair needs the Weapon's
// +0x66 byte, which the constructors zero (ITEM-157) and no item here carries.
func weaponBaseLines(code data.ItemCode, w data.Weapon, table *mapload.Table, words ui.Words) []string {
	suitable, known := data.SuitabilityFromCode(code, table.Weapons, table.Shields, table.Armors)
	if !known || !suitable.Fighter {
		return nil
	}
	base, spread := int64(w.DamageBase), int64(w.DamageSpread)
	lines := []string{fmt.Sprintf("#%s: %d-%d", words.ItemStats[43], base, base+spread)}
	if w.ToHit > 0 {
		lines = append(lines, fmt.Sprintf("#%s %+d", words.ItemStats[12], uint8(w.ToHit)))
	}
	if w.Defence > 0 {
		lines = append(lines, fmt.Sprintf("#%s %+d", words.ItemStats[15], uint8(w.Defence)))
	}
	if uint8(w.Range) > 1 {
		lines = append(lines, fmt.Sprintf("#%s %d", words.ItemStats[38], uint8(w.Range)))
	}
	return lines
}

// castWeaponInfoLines is the information text of a Weapon with a castSpell
// Effect, in the stream order of ITEM-211: the name; the damage, to-hit,
// defence and range pairs only when sutableFor has bit 0 (a Staff and a
// Shaman Staff have 2 and get none); the magic heading; then each Effect in
// list order. A castSpell Effect gives its casts line and the lines of a Spell
// filled from its own spell id and power bytes (TEXT-127): damage when the
// spread is nonzero, the Weapon's range, duration when nonzero and one caption.
// Nothing here reads the wearer or the Weapon's built Spell.
func castWeaponInfoLines(item sim.ItemInstance, table *mapload.Table, words ui.Words) ([]string, bool) {
	spellID, power, ok := item.CastSpell()
	if !ok || table == nil {
		return nil, false
	}
	code := data.ItemCode(item.Code)
	w, err := data.WeaponFromCode(code, table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		return nil, false
	}
	if tableFiles(table).releasedSpellReadout {
		lines, ok := releasedSpellLines(code, table, spellID, power, words)
		if !ok {
			return nil, false
		}
		for _, effect := range item.Effects {
			switch effect.Kind {
			case 41:
			case 42:
				lines[len(lines)-1] += itemSpellFragment(uint16(effect.Operand), table, words)
			default:
				lines = append(lines, itemEffectInfoLine(effect, table, words))
			}
		}
		return lines, true
	}
	labels := castWeaponWords(words)
	lines := []string{itemName(code, table)}
	lines = append(lines, weaponBaseLines(code, w, table, words)...)
	lines = append(lines, labels.Hover[189])
	rules := mapload.SpellRules(table)
	for _, effect := range item.Effects {
		switch effect.Kind {
		case 41:
			spell, power := uint16(uint8(effect.Operand)), int32(uint8(effect.Operand>>16))
			name := weaponSpellName(spell, table)
			if int(spell) < len(words.ItemSpellNames) && words.ItemSpellNames[spell] != "" {
				name = words.ItemSpellNames[spell]
			}
			lines = append(lines, "#"+labels.Hover[92]+" "+name+labels.Hover[93])
			knower := spellKnower{rule: sim.SpellRule{ID: spell}}
			for _, rule := range rules {
				if rule.ID == spell {
					knower = spellKnower{rule, sim.SpellCharacteristicsAtPower(mapload.TableRules(table), rule, power)}
					break
				}
			}
			knower.c.Range = int64(uint8(w.Range))
			lines = spellDetailsLines([]spellKnower{knower}, lines, &labels, weaponDetails)
		case 42:
			lines[len(lines)-1] += itemSpellFragment(uint16(effect.Operand), table, words)
		default:
			lines = append(lines, itemEffectInfoLine(effect, table, words))
		}
	}
	return lines, true
}

// releasedSpellLines is the second game's spell weapon text: the spell the
// attack releases at the stored power, its damage, rays and spell range, with
// no base lines (DIV-1977); the other Effects follow.
func releasedSpellLines(code data.ItemCode, table *mapload.Table, spellID uint16, power int32, words ui.Words) ([]string, bool) {
	spell, ok := sim.WeaponSpellCharacteristicsFor(mapload.TableRules(table), sim.Entity{
		MaxMana: 1, WeaponSpell: spellID, WeaponSpellLevel: power,
	}, mapload.SpellRules(table))
	if !ok {
		return nil, false
	}
	name := weaponSpellName(spell.SpellID, table)
	if int(spell.SpellID) < len(words.ItemSpellNames) && words.ItemSpellNames[spell.SpellID] != "" {
		name = words.ItemSpellNames[spell.SpellID]
	}
	lines := []string{itemName(code, table), words.ItemMagic}
	switch spell.Arm {
	case 20:
		return append(lines,
			fmt.Sprintf("%s %s", name, spellDurationText(spell.DurationTicks, &words)),
			fmt.Sprintf("%s %d", words.ItemRange, spell.MaxRange)), true
	case 14:
		lines = append(lines, fmt.Sprintf("%s %s", words.ItemCasts, name))
		if spell.HasDamage {
			lines = append(lines, fmt.Sprintf("%s %d-%d", words.ItemDamage, spell.DamageMin, spell.DamageMax))
		}
		return append(lines,
			fmt.Sprintf("%s %d", words.ItemRays, spell.RayCount),
			fmt.Sprintf("%s %d", words.ItemRange, spell.MaxRange)), true
	}
	lines = append(lines, fmt.Sprintf("%s %s", words.ItemCasts, name))
	if spell.HasDamage {
		lines = append(lines, fmt.Sprintf("%s %d-%d", words.ItemDamage, spell.DamageMin, spell.DamageMax))
	}
	return append(lines, fmt.Sprintf("%s %d", words.ItemRange, spell.MaxRange)), true
}

func spellDurationSeconds(ticks uint16) string {
	// One nominal game second is sixteen simulation ticks. Round the displayed
	// fraction to hundredths; the stored countdown retains its exact ticks.
	hundredths := (uint32(ticks)*100 + 8) / 16
	if hundredths%100 == 0 {
		return fmt.Sprintf("%d", hundredths/100)
	}
	return strings.TrimRight(fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100), "0")
}

func spellDurationText(ticks uint16, words *ui.Words) string {
	unit := words.DurationUnit
	if unit == "" {
		unit = ui.AuthoredWords().DurationUnit
	}
	return spellDurationSeconds(ticks) + " " + unit
}

// castWeaponWords fills the install's main.txt captions a castSpell Weapon's
// text uses with this build's authored words where no install states them.
func castWeaponWords(words ui.Words) ui.Words {
	for _, f := range []struct {
		index    int
		fallback string
	}{{189, words.ItemMagic}, {92, words.ItemCasts}, {spellLabelDamage, words.ItemDamage}, {spellLabelRange, words.ItemRange}} {
		if words.Hover[f.index] == "" {
			words.Hover[f.index] = f.fallback
		}
	}
	return words
}

// wornProtectionLines are the base pairs an Armor or Shield writes ahead of its
// Effect pairs: defence always, absorption only above zero, each in the
// formatter's `#label +value` form. The formatter prints one line per pair and
// merges none, so a Defence Effect on an item with base defence gives two
// Defence lines (ITEM-154, ITEM-155).
func wornProtectionLines(defence, absorption int32, words *ui.Words) []string {
	lines := []string{fmt.Sprintf("#%s %+d", words.ItemStats[15], uint8(defence))}
	if absorption > 0 {
		lines = append(lines, fmt.Sprintf("#%s %+d", words.ItemStats[16], uint8(absorption)))
	}
	return lines
}

func itemInstanceInfoLinesWithWeaponDamage(item sim.ItemInstance, table *mapload.Table,
	resolve weaponCastResolver, installed ...ui.Words) []string {
	words := itemWords(installed)
	if _, _, ok := item.CastSpell(); !ok && resolve != nil {
		if spell, power, ok := resolve(data.ItemCode(item.Code)); ok {
			item = item.Clone()
			item.Effects = append(item.Effects, castSpellEffect(spell, power))
		}
	}
	if lines, ok := castWeaponInfoLines(item, table, words); ok {
		return lines
	}
	lines := itemBaseInfoLines(data.ItemCode(item.Code), table, &words)
	// Class-14 spell effects continue the name line (TEXT-090, TEXT-092).
	book := item.Code&0xf00 == 0xe00
	magic := false
	for index, effect := range item.Effects {
		if !book && !magic {
			lines = append(lines, words.ItemMagic)
			magic = true
		}
		if effect.Kind == 42 || book && effect.Kind == 41 {
			id := uint16(effect.Operand)
			scrollID, power, scroll := sim.ScrollSpell(item)
			if index == 0 && scroll {
				id = scrollID
			}
			lines[len(lines)-1] += itemSpellFragment(id, table, words)
			if index == 0 && scroll {
				for _, rule := range mapload.SpellRules(table) {
					if rule.ID == id {
						c := sim.SpellCharacteristicsAtPower(mapload.TableRules(table), rule, power)
						lines = spellDetailsLines([]spellKnower{{rule, c}}, lines, &words, scrollDetails)
						break
					}
				}
			}
			continue
		}
		lines = append(lines, itemEffectInfoLine(effect, table, words))
	}
	// The stored price has no line: the original composes none (ITEM-PRICETAG-144).
	return lines
}

// itemSpellFragment is what the item formatter appends for a teach-spell or, on
// a book, a cast-spell effect: the install's connective, the spell's own name
// from spell.txt and the install's closing word, continuing the line before it.
func itemSpellFragment(spell uint16, table *mapload.Table, words ui.Words) string {
	name := weaponSpellName(spell, table)
	if int(spell) < len(words.ItemSpellNames) && words.ItemSpellNames[spell] != "" {
		name = words.ItemSpellNames[spell]
	}
	return " " + words.ItemSpellOf + " " + name + words.ItemSpellOfSuffix
}

func itemWords(installed []ui.Words) ui.Words {
	words := ui.AuthoredWords()
	if len(installed) > 0 {
		words = installed[0]
	}
	authored := ui.AuthoredWords()
	if words.ItemMagic == "" {
		words.ItemMagic = authored.ItemMagic
	}
	if words.ItemSpellOf == "" {
		words.ItemSpellOf = authored.ItemSpellOf
	}
	if words.DurationUnit == "" {
		words.DurationUnit = authored.DurationUnit
	}
	for _, f := range []struct{ dst, src *string }{
		{&words.ItemCasts, &authored.ItemCasts}, {&words.ItemDamage, &authored.ItemDamage},
		{&words.ItemRange, &authored.ItemRange}, {&words.ItemRays, &authored.ItemRays},
	} {
		if *f.dst == "" {
			*f.dst = *f.src
		}
	}
	for i := range words.ItemStats {
		if words.ItemStats[i] == "" {
			words.ItemStats[i] = authored.ItemStats[i]
		}
	}
	return words
}

func itemEffectInfoLine(effect sim.ItemEffect, table *mapload.Table, installed ...ui.Words) string {
	words := itemWords(installed)
	name := fmt.Sprintf("Effect %d", effect.Kind)
	if int(effect.Kind) < len(words.ItemStats) && words.ItemStats[effect.Kind] != "" {
		name = words.ItemStats[effect.Kind]
	}
	suffix := itemEffectModeInfo(effect)
	switch {
	case effect.Kind == 41:
		spell := uint16(effect.Operand)
		name := weaponSpellName(spell, table)
		if int(spell) < len(words.ItemSpellNames) && words.ItemSpellNames[spell] != "" {
			name = words.ItemSpellNames[spell]
		}
		return "#" + words.ItemCasts + " " + name
	case effect.Kind == 42:
		return itemSpellFragment(uint16(effect.Operand), table, words)
	case effect.Kind >= 43 && effect.Kind <= 48:
		base := uint8(effect.Operand)
		spread := uint8(effect.Operand >> 8)
		return fmt.Sprintf("#%s: %d-%d", name, base, uint16(base)+uint16(spread)) + suffix
	default:
		value := int32(effect.Operand)
		if effect.Mode != 0 && effect.Mode != 8 {
			value = int32(int16(effect.Operand))
		}
		switch effect.Mode {
		case 1:
			return fmt.Sprintf("#%s: +%d", name, value) + suffix
		case 2:
			return fmt.Sprintf("#%s: +%d%%", name, value) + suffix
		case 4:
			return fmt.Sprintf("#%s: %5.1f", name, float64(value)) + suffix
		case 8:
			return fmt.Sprintf("#%s %d", name, value) + suffix
		default:
			if value < 0 {
				return fmt.Sprintf("#%s ", name) + fmt.Sprintf("-%d", -value) + suffix
			}
			return fmt.Sprintf("#%s %+d", name, value) + suffix
		}
	}
}

func itemEffectModeInfo(effect sim.ItemEffect) string {
	switch effect.Mode {
	case 0:
		return ""
	case 8:
		return " (single use)"
	case 1, 2, 4:
		name := map[uint8]string{1: "duration", 2: "continuous", 4: "charges"}[effect.Mode]
		return fmt.Sprintf(" (%s %d)", name, uint16(effect.Operand>>16)>>4)
	default:
		return fmt.Sprintf(" (mode %d)", effect.Mode)
	}
}

// slotInfoLines is itemInfoLines run over every OCCUPIED slot of eq, in
// slot order — the worn box's own popup text, one entry per slot index and
// a nil entry for a slot nothing is worn in.
func slotInfoLines(eq data.Equipment, table *mapload.Table) [12][]string {
	return slotInfoLinesWithWeaponDamage(eq, table, nil)
}

func slotInfoLinesWithWeaponDamage(eq data.Equipment, table *mapload.Table, resolve weaponCastResolver) [12][]string {
	var out [12][]string
	for n := 1; n <= data.EquipSlots; n++ {
		occupied, _ := eq.Occupied(n)
		if !occupied {
			continue
		}
		code, _ := eq.Code(n)
		out[n-1] = itemInfoLinesWithWeaponDamage(code, table, resolve)
	}
	return out
}

func slotItemInfoLinesWithWeaponDamage(eq [sim.EquipSlots]sim.ItemInstance, table *mapload.Table,
	resolve weaponCastResolver, installed ...ui.Words) [sim.EquipSlots][]string {
	var out [sim.EquipSlots][]string
	for i, item := range eq {
		if item.Empty() {
			continue
		}
		out[i] = itemInstanceInfoLinesWithWeaponDamage(item, table, resolve, installed...)
	}
	return out
}

// packInfoLines is itemInfoLines run over every carried ELEMENT of stacks,
// in the same order buildInventoryPack already reads that slice in — the
// pack grid's own popup text, one entry per element and PARALLEL TO Pack
// the same way PackCount already is (pkg/ui's own InventorySubject doc).
func packInfoLines(stacks []sim.ItemStack, table *mapload.Table) [][]string {
	return packInfoLinesWithWeaponDamage(stacks, table, nil)
}

func packInfoLinesWithWeaponDamage(stacks []sim.ItemStack, table *mapload.Table, resolve weaponCastResolver, installed ...ui.Words) [][]string {
	if len(stacks) == 0 {
		return nil
	}
	out := make([][]string, len(stacks))
	for i, s := range stacks {
		out[i] = itemInstanceInfoLinesWithWeaponDamage(s.Instance(), table, resolve, installed...)
	}
	return out
}
