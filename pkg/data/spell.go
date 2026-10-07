package data

import (
	"fmt"
	"strconv"
	"strings"
)

// lastSpellSlot is the last parameter slot LoadSpells reads: the Defensive
// column. A row that does not reach it is refused rather than padded.
const lastSpellSlot = 18

// The scalar parameter slots named by the cited claims, each the runtime
// column its own field is read from — the same index EntryParams already hands
// cells back at (DAT-SCHEMA-004: param i is CSV column i+1).
// MAGIC-CADENCE-126 anchors slot 0, MAGIC-TARGET-017 anchors slot 4 at
// instruction level (`getParam(row, 4) == 1`, title "Spell Target"), and
// MAGIC-DMG-005 anchors 16 and 17 the same way.
const (
	spellComplicationSlot = 0
	spellManaCostSlot     = 1
	spellSchoolSlot       = 2
	spellTargetSlot       = 4
	spellDeliverySlot     = 5
	spellMaxRangeSlot     = 6
	spellSpeedSlot        = 7
	spellDamageMinSlot    = 16
	spellDamageMaxSlot    = 17
	spellDefensiveSlot    = 18

	// The two shape columns. MAGIC-SHAPE-008 anchors both at instruction level:
	// `getParam(row, 8)`, title 9 "Distribution system", is read at L05102
	// and forks the apply — value 1 builds a PointEffect and requires a
	// target unit, anything else builds an AreaEffect; and `getParam(row,
	// 0x0b)`, title 12 "Area Effect Duaration", is read at L05100 and is the
	// FIRST TERM of that effect's own lifetime.
	//
	// SLOT 8 IS NOT SLOT 4. "Spell Target" above and "Distribution system"
	// here are different columns with different values on shipped rows —
	// Shield carries 2 and 1 — so the shape is a field of its own rather
	// than a second reading of TargetsUnit.
	spellDistributionSlot = 8
	spellRadiusSlot       = 9
	spellAreaDurationSlot = 11
	spellDurationSlot     = 14
)

// SpellEffectKind is the subset of the Magic effect vocabulary used by the
// shipped Spells table. Values are stable at the data/simulation seam; zero is
// deliberately "none".
type SpellEffectKind uint8

const (
	SpellEffectNone SpellEffectKind = iota
	SpellEffectHealth
	SpellEffectSpeed
	SpellEffectScanRange
	SpellEffectAbsorption
	SpellEffectProtectionFire
	SpellEffectProtectionWater
	SpellEffectProtectionAir
	SpellEffectProtectionEarth
)

// SpellEffectMode is the original effect duration-word family. The bit values
// are the decoded values, so mixed modes remain representable.
type SpellEffectMode uint8

const (
	SpellEffectDuration   SpellEffectMode = 1
	SpellEffectContinuous SpellEffectMode = 2
	SpellEffectCharges    SpellEffectMode = 4
	SpellEffectSingleUse  SpellEffectMode = 8
)

// SpellEffect is the first effect record in a shipped spell row. Stone Curse
// carries a second record, but its decoded arm intentionally ignores it.
type SpellEffect struct {
	Kind      SpellEffectKind
	Mode      SpellEffectMode
	Magnitude int32
	Duration  uint16
}

// The two spell ids the game excludes from the shared damage arm BY ID
// ALONE, though both carry a damage-shaped column pair: Heal turns its roll
// into healing and Drain Life moves the roll from the victim to the caster,
// and neither is damage in the sense Damaging names below. MAGIC-DMG-005:
// "the whole arm is skipped when the pair sums to 0 or the spell is 6 (Heal)
// or 11 (Drain Life), both of which carry damage columns and are not
// damage." Named once, here, and read nowhere but LoadSpells' own
// comparison.
const (
	healSpellID      = 6
	drainLifeSpellID = 11
)

type Spell struct {
	// Name is the row's own name, verbatim.
	Name string

	// Complication is slot 0, "Complication Level". It is the retained-cast
	// recovery addend, separate from mana cost. The empty-cell sentinel folds
	// to zero and the simulation seam narrows the installed cell to one byte.
	Complication int32

	// ManaCost is slot 1, "Mana Cost". A negative cell is REFUSED rather than
	// clamped, unlike the three fields below: no shipped row ships one, and a
	// file that did would be describing a free cast this contract does not
	// silently agree to.
	ManaCost int32

	// School is slot 2, "Sphere" — the elemental kind a later resistance arm
	// and skill term will both index. Carried verbatim: this story reads no
	// meaning into the value beyond passing it through.
	School int32

	// TargetsUnit is slot 4, "Spell Target", true iff the cell is exactly 1 —
	// MAGIC-TARGET-017's own routing test. Anything else, including the
	// format's own empty cell, routes at a point instead, which this build
	// does not model and so never targets.
	TargetsUnit bool

	// Columns consumed by the simulation transport, not the client picture.
	Delivery, EffectSpeed int32

	// MaxRange is slot 6, "Max Range". A negative cell — the format's own
	// empty-cell sentinel — lands as 0, never as the row's own reach.
	MaxRange int32

	// DamageMin and DamageMax are slots 16 and 17, "damageMin" and
	// "damageMax". A negative cell lands as 0 on each independently, the
	// same clamp MAGIC-DMG-005's own guard applies before either reaches the
	// roll.
	DamageMin int32
	DamageMax int32

	// Defensive is slot 18, true iff the cell is exactly 1. Carried and read
	// by nothing in this build; a later story reads it rather than the row a
	// second time.
	Defensive bool

	// Area is slot 8, "Distribution system", true iff the cell is anything but
	// 1. A row it is true of builds an AREA EFFECT standing on a cell; a row it
	// is false of builds a POINT EFFECT, which requires a target unit.
	// MAGIC-BURST-031 counts the shipped split at 18 point rows and 10 area
	// rows, the area set being spells 2, 3, 4, 7, 8, 9, 12, 17, 19 and 21.
	//
	// THE COLUMN'S OTHER VALUES ARE NOT CARRIED. MAGIC-SHAPE-008 leaves the
	// value-to-word mapping of that enum explicitly unread, so storing 3
	// against 5 would be storing a number nothing in this tree can
	// interpret. One bit is what the claim decodes and one bit is what is
	// kept.
	Area bool

	// Distribution and Radius retain the two installed shape cells needed by
	// the decoded area painter. Area remains the convenient shape predicate.
	Distribution uint8
	Radius       uint8

	// AreaDuration is slot 11, "Area Effect Duaration", the first term of an
	// area effect's own lifetime in ticks: `(AreaDuration << 4) + (power <<
	// 4)/10` (MAGIC-SHAPE-008 as corrected, carried forward in
	// MAGIC-CEIL-013). A negative cell — the format's own empty-cell
	// sentinel — lands as 0, the same clamp MaxRange takes. Read by nothing
	// on a row whose Area is false.
	AreaDuration int32

	// SpellDuration is the installed duration exponent/base term consumed by
	// the spell-specific lasting arms.
	SpellDuration int32

	// Effects is the row's one trailing string, verbatim and unparsed (SC-3).
	Effects string

	// Effect is the parsed first record of Effects. A row with no applicable
	// record leaves it zero; singular arms are selected by spell id in sim.
	Effect SpellEffect

	// Damaging is true iff this row's damage pair is positive — DamageMin or
	// DamageMax clamped above zero — and the row's id is neither healSpellID
	// nor drainLifeSpellID. Set here, by LoadSpells, and by nothing else.
	Damaging bool

	// Restorative is true iff this row's damage pair is positive and the row's
	// id is healSpellID. It is the EXACT COMPLEMENT of Damaging over the heal
	// row and is computed from the same comparison: heal is refused from the
	// damage arm because its roll is healing, so the one place that refuses it
	// there is the one place that admits it here, and the two flags can never
	// come apart.
	//
	// DRAIN LIFE IS NEITHER. It moves its roll from the victim to the caster,
	// which is a third arm this build does not have, so it stays excluded from
	// both flags exactly as it was.
	Restorative bool
}

// LoadSpells turns a parsed Spells collection into one Spell per written
// row, in id order — id being the row's own subscript, walked over entries
// 1..c.Len()-1. Entry 0 is the collection's reserved, never-written slot and
// is skipped rather than loaded as a spell.
//
// A nil collection loads no spells and returns no error: "no table" is a
// state a caller can hold without a branch of its own, the same rule this
// package's other Collection-based searches (defsearch.go) already follow.
//
// It fails, naming the row, on a row too short to reach slot 18 and on a
// negative mana cost — both refused rather than padded or clamped, because a
// row this contract cannot represent should say so rather than ship a spell
// with a silent hole in it.
func LoadSpells(c Collection) ([]Spell, error) {
	if c == nil {
		return nil, nil
	}
	spells := make([]Spell, 0, c.Len())
	for i := 1; i < c.Len(); i++ {
		name := c.EntryName(i)
		p := c.EntryParams(i)
		if len(p) <= lastSpellSlot {
			return nil, fmt.Errorf("data: spell %d %q carries %d parameter(s); slots 0 to %d are read",
				i, name, len(p), lastSpellSlot)
		}

		manaCost := p[spellManaCostSlot]
		if manaCost < 0 {
			return nil, fmt.Errorf("data: spell %d %q: mana cost %d is negative", i, name, manaCost)
		}

		dmin := clampSpellCell(p[spellDamageMinSlot])
		dmax := clampSpellCell(p[spellDamageMaxSlot])

		sp := Spell{
			Name:         name,
			Complication: clampSpellByte(p[spellComplicationSlot]),
			ManaCost:     manaCost,
			School:       p[spellSchoolSlot],
			TargetsUnit:  p[spellTargetSlot] == 1,
			Delivery:     clampSpellCell(p[spellDeliverySlot]),
			EffectSpeed:  clampSpellCell(p[spellSpeedSlot]),
			MaxRange:     clampSpellCell(p[spellMaxRangeSlot]),
			DamageMin:    dmin,
			DamageMax:    dmax,
			Defensive:    p[spellDefensiveSlot] == 1,

			Area:          p[spellDistributionSlot] != 1,
			Distribution:  uint8(clampSpellByte(p[spellDistributionSlot])),
			Radius:        uint8(clampSpellByte(p[spellRadiusSlot])),
			AreaDuration:  clampSpellCell(p[spellAreaDurationSlot]),
			SpellDuration: clampSpellCell(p[spellDurationSlot]),
		}
		sp.Damaging, sp.Restorative = ClassifySpell(i, dmin, dmax)
		if strs := c.EntryStrings(i); len(strs) > 0 {
			sp.Effects = strs[0]
			sp.Effect = parseSpellEffect(strs[0])
		}
		spells = append(spells, sp)
	}
	return spells, nil
}

func ClassifySpell(id int, dmin, dmax int32) (damaging, restorative bool) {
	pair := dmin > 0 || dmax > 0
	return pair && id != healSpellID && id != drainLifeSpellID, pair && id == healSpellID
}

// clampSpellByte admits an installed byte-valued cell into the canonical
// table: the empty sentinel becomes zero and an edited oversized value cannot
// wrap onto another authored value when the map seam narrows it.
func clampSpellByte(v int32) int32 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// parseSpellEffect reads only the grammar the installed Spells collection
// uses: name=magnitude:mode duration. Unknown names and singular text such as
// Invisibility remain zero and are handled by their decoded spell arm.
func parseSpellEffect(s string) SpellEffect {
	first := strings.TrimSpace(strings.SplitN(s, ",", 2)[0])
	eq := strings.IndexByte(first, '=')
	colon := strings.IndexByte(first, ':')
	if eq <= 0 || colon <= eq {
		return SpellEffect{}
	}
	kinds := map[string]SpellEffectKind{
		"health": SpellEffectHealth, "speed": SpellEffectSpeed,
		"scanrange": SpellEffectScanRange, "absorbtion": SpellEffectAbsorption,
		"protectionfire":  SpellEffectProtectionFire,
		"protectionwater": SpellEffectProtectionWater,
		"protectionair":   SpellEffectProtectionAir,
		"protectionearth": SpellEffectProtectionEarth,
	}
	kind := kinds[strings.ToLower(strings.TrimSpace(first[:eq]))]
	if kind == SpellEffectNone {
		return SpellEffect{}
	}
	mag, err := strconv.ParseInt(strings.TrimSpace(first[eq+1:colon]), 10, 32)
	if err != nil {
		return SpellEffect{}
	}
	tail := strings.Fields(strings.TrimSpace(first[colon+1:]))
	if len(tail) != 2 {
		return SpellEffect{}
	}
	modes := map[string]SpellEffectMode{
		"duration": SpellEffectDuration, "continuous": SpellEffectContinuous,
		"charges": SpellEffectCharges, "singleuse": SpellEffectSingleUse,
		"single": SpellEffectSingleUse,
	}
	mode := modes[strings.ToLower(tail[0])]
	dur, err := strconv.ParseUint(tail[1], 10, 16)
	if mode == 0 || err != nil {
		return SpellEffect{}
	}
	return SpellEffect{Kind: kind, Mode: mode, Magnitude: int32(mag), Duration: uint16(dur)}
}

func clampSpellCell(v int32) int32 {
	if v < 0 {
		return 0
	}
	return v
}
