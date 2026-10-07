package mapload

import (
	"fmt"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

const (
	fieldMana = 1 << iota
	fieldRange
	fieldRadius
	fieldDuration
	fieldAreaLife
	fieldDamage
	fieldEffectDuration
)

func modSpellIndex(spells []data.Spell, target string) int {
	name := strings.ReplaceAll(target, "_", " ")
	for i, sp := range spells {
		if sp.Name != "" && sp.Name == name {
			return i
		}
	}
	return -1
}

func ModSpellRefusal(t *Table, r mod.SpellRow) (line int, msg string) {
	if t == nil {
		return r.Line, "the game has no spell table"
	}
	spells, err := data.LoadSpells(t.spells())
	if err != nil || len(spells) == 0 {
		return r.Line, "the game has no spell table"
	}
	i := modSpellIndex(spells, r.Target)
	if i < 0 {
		return r.Line, fmt.Sprintf("spell %q is not a row of the spell table", r.Target)
	}
	sp := spells[i]
	if r.DamageMax.Set && sp.DamageMin <= 0 && sp.DamageMax <= 0 {
		return r.DamageMax.Line, fmt.Sprintf("%s has no damage or healing pair to edit", sp.Name)
	}
	if e := r.Effect; e != nil {
		kind, mode := effectKindName(sp.Effect.Kind), effectModeName(sp.Effect.Mode)
		if e.Kind != "" {
			kind = e.Kind
		}
		if e.Mode != "" {
			mode = e.Mode
		}
		if kind != "none" && mode == "" {
			return e.Line, fmt.Sprintf("%s has no effect mode to keep; give effect.mode with effect.kind", sp.Name)
		}
	}
	if r.HealHostile.Set {
		if _, restorative := data.ClassifySpell(i+1, sp.DamageMin, sp.DamageMax); !restorative {
			return r.HealHostile.Line, fmt.Sprintf("%s does not heal, so heal_hostile does not apply to it", sp.Name)
		}
	}
	if r.SelfCast.Set {
		if damaging, _ := data.ClassifySpell(i+1, sp.DamageMin, sp.DamageMax); !damaging {
			return r.SelfCast.Line, fmt.Sprintf("%s does not damage, so self_cast does not apply to it", sp.Name)
		}
	}
	if r.AreaHits.Set && !sp.Area {
		return r.AreaHits.Line, fmt.Sprintf("%s is not an area spell, so area_hits does not apply to it", sp.Name)
	}
	if r.Rays.Set && i+1 != sim.PrismaticSpellID {
		return r.Rays.Line, fmt.Sprintf("%s does not fire rays, so rays does not apply to it", sp.Name)
	}
	return modSpellFormulaRefusal(sp, i+1, r)
}

func modSpellFormulaRefusal(sp data.Spell, id int, r mod.SpellRow) (int, string) {
	if t := r.DamageFactor; t.Set && sp.DamageMin <= 0 && sp.DamageMax <= 0 {
		return t.Line, fmt.Sprintf("%s has no damage or healing pair, so damage_factor does not apply to it", sp.Name)
	}
	if t := r.RangeBonus; t.Set && sp.MaxRange == 0 && (!r.Range.Set || r.Range.Val == 0) {
		return t.Line, fmt.Sprintf("%s has range 0, so range_bonus does not apply to it", sp.Name)
	}
	if t := r.DurationFactor; t.Set && !sim.DurationArm(uint16(id)) {
		return t.Line, fmt.Sprintf("%s has no power-scaled duration, so duration_factor does not apply to it", sp.Name)
	}
	if t := r.Magnitude; t.Set && !sim.MagnitudeArm(uint16(id)) {
		return t.Line, fmt.Sprintf("%s has no power-scaled magnitude, so magnitude does not apply to it", sp.Name)
	}
	return 0, ""
}

func SpellFormulas(t *Table, d mod.SpellData) (rules.SpellFormulaSet, error) {
	var set rules.SpellFormulaSet
	for f, g := range d.Global.Formulas() {
		if g.Set {
			set.Global[f] = g.Vals
		}
	}
	rows := false
	for _, r := range d.Rows {
		for _, tab := range r.Formulas() {
			rows = rows || tab.Set
		}
	}
	if !rows {
		return set, nil
	}
	spells, err := data.LoadSpells(t.spells())
	if err != nil || len(spells) == 0 {
		return set, fmt.Errorf("the game has no spell table")
	}
	for _, r := range d.Rows {
		i := modSpellIndex(spells, r.Target)
		if i < 0 || i+1 > rules.SpellIDs {
			continue
		}
		for f, tab := range r.Formulas() {
			if tab.Set {
				set.Spell[f][i+1] = tab.Vals
			}
		}
	}
	return set, nil
}

func applyModSpellTargets(rules []sim.SpellRule, spells []data.Spell, d mod.SpellData) {
	for _, r := range d.Rows {
		i := modSpellIndex(spells, r.Target)
		if i < 0 || i >= len(rules) {
			continue
		}
		rule := &rules[i]
		rule.HealHostile = r.HealHostile.Set && r.HealHostile.Val && rule.Restorative
		rule.SelfCast = r.SelfCast.Set && r.SelfCast.Val && rule.Damaging
		if r.Rays.Set && rule.ID == sim.PrismaticSpellID {
			rule.Rays, rule.Radius = uint8(r.Rays.Val), 0
		}
		if r.AreaHits.Set && rule.Area {
			switch r.AreaHits.Val {
			case "not_own":
				rule.AreaHits = sim.AreaHitsNotOwn
			case "hostile":
				rule.AreaHits = sim.AreaHitsHostile
			default:
				rule.AreaHits = sim.AreaHitsAll
			}
		}
	}
}

func applyModSpells(spells []data.Spell, d mod.SpellData) {
	if d.Empty() {
		return
	}
	set := make([]uint8, len(spells))
	for _, r := range d.Rows {
		i := modSpellIndex(spells, r.Target)
		if i < 0 {
			continue
		}
		sp := &spells[i]
		if r.Mana.Set {
			sp.ManaCost, set[i] = r.Mana.Val, set[i]|fieldMana
		}
		if r.Range.Set {
			sp.MaxRange, set[i] = r.Range.Val, set[i]|fieldRange
		}
		if r.Radius.Set {
			sp.Radius, set[i] = uint8(r.Radius.Val), set[i]|fieldRadius
		}
		if r.Duration.Set {
			sp.SpellDuration, set[i] = r.Duration.Val, set[i]|fieldDuration
		}
		if r.AreaLife.Set {
			sp.AreaDuration, set[i] = r.AreaLife.Val, set[i]|fieldAreaLife
		}
		if r.DamageMax.Set {
			sp.DamageMin, sp.DamageMax, set[i] = r.DamageMin.Val, r.DamageMax.Val, set[i]|fieldDamage
		}
		if e := r.Effect; e != nil {
			if e.Kind != "" {
				sp.Effect.Kind = effectKindFor(e.Kind)
			}
			if e.Mode != "" {
				sp.Effect.Mode = effectModeFor(e.Mode)
			}
			if e.Magnitude.Set {
				sp.Effect.Magnitude = e.Magnitude.Val
			}
			if e.Duration.Set {
				sp.Effect.Duration, set[i] = uint16(e.Duration.Val), set[i]|fieldEffectDuration
			}
		}
	}
	g := d.Global
	for i := range spells {
		sp := &spells[i]
		if g.Mana.Set && set[i]&fieldMana == 0 {
			sp.ManaCost = scaleSpell(int64(sp.ManaCost), g.Mana, mod.SpellMaxMana)
		}
		if g.Range.Set && set[i]&fieldRange == 0 {
			sp.MaxRange = scaleSpell(int64(sp.MaxRange), g.Range, mod.SpellMaxRange)
		}
		if g.Radius.Set && set[i]&fieldRadius == 0 {
			sp.Radius = uint8(scaleSpell(int64(sp.Radius), g.Radius, mod.SpellMaxRadius))
		}
		if g.Duration.Set {
			if set[i]&fieldDuration == 0 {
				sp.SpellDuration = scaleSpell(int64(sp.SpellDuration), g.Duration, mod.SpellMaxDuration)
			}
			if set[i]&fieldAreaLife == 0 {
				sp.AreaDuration = scaleSpell(int64(sp.AreaDuration), g.Duration, mod.SpellMaxAreaLife)
			}
			if set[i]&fieldEffectDuration == 0 {
				sp.Effect.Duration = uint16(scaleSpell(int64(sp.Effect.Duration), g.Duration, mod.SpellMaxDuration))
			}
		}
		if set[i]&fieldDamage == 0 && (sp.DamageMin > 0 || sp.DamageMax > 0) {
			ratio := g.Damage
			if sp.Restorative {
				ratio = g.Heal
			}
			if ratio.Set {
				sp.DamageMin = scaleSpell(int64(sp.DamageMin), ratio, mod.SpellMaxDamage)
				sp.DamageMax = scaleSpell(int64(sp.DamageMax), ratio, mod.SpellMaxDamage)
			}
		}
		sp.Damaging, sp.Restorative = data.ClassifySpell(i+1, sp.DamageMin, sp.DamageMax)
	}
}

func scaleSpell(v int64, r mod.SpellRatio, max int64) int32 {
	n := v * r.Num / r.Den
	if n > max {
		n = max
	}
	if n < 0 {
		n = 0
	}
	return int32(n)
}

var effectKindsByName = map[string]data.SpellEffectKind{
	"none": data.SpellEffectNone, "health": data.SpellEffectHealth, "speed": data.SpellEffectSpeed,
	"scanrange": data.SpellEffectScanRange, "absorption": data.SpellEffectAbsorption,
	"protectionfire": data.SpellEffectProtectionFire, "protectionwater": data.SpellEffectProtectionWater,
	"protectionair": data.SpellEffectProtectionAir, "protectionearth": data.SpellEffectProtectionEarth,
}

var effectModesByName = map[string]data.SpellEffectMode{
	"duration": data.SpellEffectDuration, "continuous": data.SpellEffectContinuous,
	"charges": data.SpellEffectCharges, "singleuse": data.SpellEffectSingleUse,
}

func effectKindFor(name string) data.SpellEffectKind { return effectKindsByName[name] }
func effectModeFor(name string) data.SpellEffectMode { return effectModesByName[name] }

func effectKindName(k data.SpellEffectKind) string {
	for n, v := range effectKindsByName {
		if v == k {
			return n
		}
	}
	return "none"
}

func effectModeName(m data.SpellEffectMode) string {
	for n, v := range effectModesByName {
		if v == m {
			return n
		}
	}
	return ""
}
