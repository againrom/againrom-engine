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

// itemInfoLines is one item code's own popup text: its name, by itemName's
// own rule (world.go), followed by whichever of the two decoded
// characteristic lines its class resolves through — or the name alone when
// neither does.
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

// weaponDamageInterval retains the established internal name while carrying
// the complete simulation-owned item-spell projection. Stone Curse needs its
// duration and Prismatic Spray needs its ray count in addition to damage.
type weaponDamageInterval = sim.WeaponSpellCharacteristics

type weaponDamageResolver func(data.ItemCode) (weaponDamageInterval, bool)

// liveWeaponSpellDamage refuses an interval unless code is the entity's
// currently equipped weapon. The answer then comes from World directly: the
// same entity fields, spell row and arithmetic releaseWeaponSpell uses.
func liveWeaponSpellDamage(w *sim.World, id sim.EntityID, code data.ItemCode) (weaponDamageInterval, bool) {
	if w == nil {
		return weaponDamageInterval{}, false
	}
	equipped, ok := w.Equipped(id)
	if !ok || equipped[0] != uint16(code) {
		return weaponDamageInterval{}, false
	}
	for _, entity := range w.Entities() {
		if entity.ID != id {
			continue
		}
		rule, ok := w.Spell(uint32(entity.WeaponSpell))
		if !ok {
			return weaponDamageInterval{}, false
		}
		return sim.WeaponSpellCharacteristicsFor(w.Rules(), entity, []sim.SpellRule{rule})
	}
	return weaponDamageInterval{}, false
}

// storedWeaponSpellDamage keeps an unequipped starting staff informative in
// the pack and makes town snapshots agree with the mission it will open. The
// spell pair is the party member's stored weapon identity; sim's canonical
// reader still performs every lookup, applicability gate and damage
// calculation. MaxMana is only the reader's mage/non-mage gate here.
func storedWeaponSpellDamage(code data.ItemCode, weapon *data.Weapon, mage bool,
	table *mapload.Table) (weaponDamageInterval, bool) {
	if !mage || weapon == nil || weapon.Code != code {
		return weaponDamageInterval{}, false
	}
	spellID, ok := mapload.SpellIDByToken(table, weapon.SpellName)
	if !ok {
		return weaponDamageInterval{}, false
	}
	rules := mapload.SpellRules(table)
	characteristics, ok := sim.WeaponSpellCharacteristicsFor(mapload.TableRules(table), sim.Entity{
		MaxMana:          1,
		WeaponSpell:      spellID,
		WeaponSpellLevel: weapon.SpellPower,
	}, rules)
	if !ok {
		return weaponDamageInterval{}, false
	}
	return characteristics, true
}

// itemWeaponSpellDamage resolves a complete kind-41 item effect without a
// live Entity. Shop cells own the ItemInstance itself, so they can state the
// same spell name, powered damage interval and range before the item is ever
// equipped. The simulation helper remains the sole power arithmetic.
func itemWeaponSpellDamage(item sim.ItemInstance, table *mapload.Table) (weaponDamageInterval, bool) {
	spellID, power, ok := item.CastSpell()
	if !ok {
		return weaponDamageInterval{}, false
	}
	rules := mapload.SpellRules(table)
	characteristics, ok := sim.WeaponSpellCharacteristicsFor(mapload.TableRules(table), sim.Entity{
		MaxMana: 1, WeaponSpell: spellID, WeaponSpellLevel: power,
	}, rules)
	if !ok {
		return weaponDamageInterval{}, false
	}
	return characteristics, true
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

func itemInfoLinesWithWeaponDamage(code data.ItemCode, table *mapload.Table, resolve weaponDamageResolver, installed ...ui.Words) []string {
	words := itemWords(installed)
	lines := []string{itemName(code, table)}
	if table == nil {
		return lines
	}
	if w, err := data.WeaponFromCode(code, table.Shapes, table.Materials, table.Weapons); err == nil {
		// THE SHEET'S OWN COMPOSITION (panel.go's PanelFieldDamage): the
		// first number is the base and the second is base + spread, the
		// roll's own bounds (data.Combat's own doc — the roll is
		// `base + U[0, spread]`, not a minimum and a maximum). Stating the
		// spread itself, as "base + spread", would read as a range the roll
		// cannot produce; the panel and the popup must agree.
		if resolve != nil {
			if spell, ok := resolve(code); ok {
				name := weaponSpellName(spell.SpellID, table)
				if int(spell.SpellID) < len(words.ItemSpellNames) && words.ItemSpellNames[spell.SpellID] != "" {
					name = words.ItemSpellNames[spell.SpellID]
				}
				lines = append(lines, words.ItemMagic)
				switch spell.SpellID { // DIV-1977
				case 20:
					lines = append(lines,
						fmt.Sprintf("%s %s", name, spellDurationText(spell.DurationTicks, &words)),
						fmt.Sprintf("%s %d", words.ItemRange, spell.MaxRange),
					)
				case 14:
					lines = append(lines, fmt.Sprintf("%s %s", words.ItemCasts, name))
					if spell.HasDamage {
						lines = append(lines, fmt.Sprintf("%s %d-%d", words.ItemDamage, spell.DamageMin, spell.DamageMax))
					}
					lines = append(lines,
						fmt.Sprintf("%s %d", words.ItemRays, spell.RayCount),
						fmt.Sprintf("%s %d", words.ItemRange, spell.MaxRange),
					)
				default:
					lines = append(lines, fmt.Sprintf("%s %s", words.ItemCasts, name))
					if spell.HasDamage {
						lines = append(lines, fmt.Sprintf("%s %d-%d", words.ItemDamage, spell.DamageMin, spell.DamageMax))
					}
					lines = append(lines, fmt.Sprintf("%s %d", words.ItemRange, spell.MaxRange))
				}
				return lines
			}
		}
		base, spread := int64(w.DamageBase), int64(w.DamageSpread)
		lines = append(lines, fmt.Sprintf("#%s: %d-%d", words.ItemStats[43], base, base+spread))
		lines = append(lines, fmt.Sprintf("#%s: %d", words.ItemStats[12], w.ToHit))
		lines = append(lines, fmt.Sprintf("#%s: %d", words.ItemStats[15], w.Defence))
		if w.Ranged() || w.AttackType == data.SkillShoot {
			lines = append(lines, weaponRangeLine(w.Range, &words))
		}
		return lines
	}
	if a, err := data.ArmorFromCode(code, table.Shapes, table.Materials, table.Armors); err == nil {
		lines = append(lines, wornProtectionLines(a.Defence, a.Absorption, &words)...)
	} else if table.Shields != nil {
		if s, err := data.ShieldFromCode(code, table.Shapes, table.Materials, table.Shields); err == nil {
			lines = append(lines, wornProtectionLines(s.Defence, s.Absorption, &words)...)
		}
	}
	return lines
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

// weaponRangeLine states a ranged weapon's own range column. A weapon is
// ranged when its attack type is the shooting skill or takes the projectile
// equip arm; a melee row's range column is reach and has no line. The caption is
// the install's spellbook range caption, the one label the installed text
// tables carry for a range; without an install it is the authored "Range".
func weaponRangeLine(rng int32, words *ui.Words) string {
	label := words.Hover[spellLabelRange]
	if label == "" {
		label = "Range"
	}
	return fmt.Sprintf("%s: %d", label, rng)
}

func itemInstanceInfoLinesWithWeaponDamage(item sim.ItemInstance, table *mapload.Table,
	resolve weaponDamageResolver, installed ...ui.Words) []string {
	words := itemWords(installed)
	itemSpell, hasItemSpell := itemWeaponSpellDamage(item, table)
	describedItemSpell := false
	combined := resolve
	if hasItemSpell {
		combined = func(code data.ItemCode) (weaponDamageInterval, bool) {
			if resolve != nil {
				if spell, ok := resolve(code); ok {
					describedItemSpell = true
					return spell, true
				}
			}
			if uint16(code) == item.Code {
				describedItemSpell = true
				return itemSpell, true
			}
			return weaponDamageInterval{}, false
		}
	}
	lines := itemInfoLinesWithWeaponDamage(data.ItemCode(item.Code), table, combined, words)
	// A book is a class-0xe00 item code. Its card has no heading, and the spell it
	// teaches or casts continues its name line (TEXT-090, TEXT-092).
	book := item.Code&0xf00 == 0xe00
	magic := describedItemSpell
	for _, effect := range item.Effects {
		if !book && !magic {
			lines = append(lines, words.ItemMagic)
			magic = true
		}
		if describedItemSpell && effect.Kind == 41 {
			continue
		}
		if effect.Kind == 42 || book && effect.Kind == 41 {
			lines[len(lines)-1] += itemSpellFragment(uint16(effect.Operand), table, words)
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

func slotInfoLinesWithWeaponDamage(eq data.Equipment, table *mapload.Table, resolve weaponDamageResolver) [12][]string {
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
	resolve weaponDamageResolver, installed ...ui.Words) [sim.EquipSlots][]string {
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

func packInfoLinesWithWeaponDamage(stacks []sim.ItemStack, table *mapload.Table, resolve weaponDamageResolver, installed ...ui.Words) [][]string {
	if len(stacks) == 0 {
		return nil
	}
	out := make([][]string, len(stacks))
	for i, s := range stacks {
		out[i] = itemInstanceInfoLinesWithWeaponDamage(s.Instance(), table, resolve, installed...)
	}
	return out
}
