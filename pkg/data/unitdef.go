package data

import "fmt"

// unitSentinel is the value an empty cell of the definition table is stored as.
// Its ONLY effect is to leave a constructor default standing: nothing in a
// loaded definition ever holds it.
const unitSentinel int32 = -1

// lastUnitSlot is the last parameter slot the spawn path consumes. Slots 38 to
// 40 are the three death-gold columns: chance, minimum and random span.
const (
	lastUnitSlot     = 40
	lastRequiredSlot = 37
)

// The three destinations of the damage-routing slot this contract models, and
// the two it refuses. Arms 1 and 2 write a DIFFERENT field pair — a second and
// third damage record nothing here reads — so an entry carrying either is
// refused by name rather than silently given no damage: unexercised on both
// shipped roots, so the cost is measured, and loudly wrong beats quietly wrong.
const (
	damageArmSecondPair int32 = 1
	damageArmThirdPair  int32 = 2
	damageArmAlwaysHits int32 = 3
)

// UnitDef is one Units row resolved into named stat fields: what a placed
// non-hero is built with.
//
// A non-hero has NO derived-stat graph — the one virtual that could hold
// it writes a single mana floor — so these columns are the stats rather
// than inputs to a formula. Every field is int32 as the registry loader's
// are, so a negative stays visibly negative, and no field ever holds the
// sentinel.
//
// Nothing outside the health pair is read by anything in this tree yet: to-hit,
// defence, absorption, the protections, the resistances, the damage pair, speed,
// reach and the attack cadence sit here and wait for the resolvers that consume
// them. They are carried rather than dropped because the row states them and a
// later story reading a column this one silently discarded would have to decode
// it again.
type UnitDef struct {
	Body, Reaction, Mind, Spirit int32

	// Health follows the maximum whenever the maximum is written, so a
	// definition is at full health by construction rather than by two numbers
	// someone keeps equal.
	Health, HealthMax int32
	// HealthRegenPeriod and ManaRegenPeriod are the two periods the constructor
	// defaults to 100 and 50; a hero, on the other arm, always carries those.
	HealthRegenPeriod int32
	Mana, ManaMax     int32
	ManaRegenPeriod   int32

	Speed         int32
	RotationSpeed int32
	ScanRange     int32
	// Withdraw and Wimpy are the two absolute-health thresholds streamed from
	// Units slots 34 and 35. Zero is the constructor default; because a living
	// actor has positive health, that default cannot arm either check.
	Withdraw int32
	Wimpy    int32
	// SeeInvisible is the Units order-block radius streamed from slot 36.
	// It remains separate from ScanRange: ordinary terrain perception first
	// establishes a candidate, then this radius decides whether an invisible
	// candidate survives the target filter.
	SeeInvisible int32

	// Sight and Reach have NO column: the constructor sets both and no slot
	// writes either. Sight stops there — it is purely one of the
	// constructor's own numbers. Reach does not: a placed unit's loader reads
	// its class row's trailing equipment strings and raises this field past the
	// constructor's floor of 1 to the first weapon's range that resolves. It
	// stays exported for exactly that reason — it is written from outside
	// this package, after NewUnitDef returns, not by a case in the slot switch
	// below.
	Sight int32
	Reach int32
	// SkillSlot is the supported active melee kind of the weapon a placement
	// loader folded onto this row. The Units columns do not state it; like
	// Reach and the weapon spell pair, definitionFor fills it from the resolved
	// equipment after NewUnitDef returns. Zero is bare, ranged or unsupported.
	SkillSlot int32

	// SpellName and SpellPower mirror Weapon's own pair: a placed unit whose
	// equipped weapon carries a spell carries it here too, set by the SAME
	// placement loader that raises Reach past the constructor's floor — after
	// NewUnitDef returns, not by a case in the slot switch below, because no
	// Units column states an attachment any more than one states a weapon's
	// range. A row whose weapon names no spell, or that carries no weapon at
	// all, leaves both at their zero value, which UnitDef.Combat below carries
	// through unchanged.
	SpellName  string
	SpellPower int32

	// DamageBase and DamageSpread are the pair the two damage columns build:
	// base is the minimum and spread is maximum minus minimum, which is the
	// form the hit resolver rolls over. AlwaysHits is the mark the third
	// routing arm sets beside them.
	DamageBase   int32
	DamageSpread int32
	// SecondaryDamage is filled by the placed creature's resolved weapon. The
	// Units row itself has no shipped selector-1/2 population, but a Dragon's
	// Flame Thrower reaches this triple through the ordinary weapon fold.
	SecondaryDamage SecondaryDamage
	AlwaysHits      bool

	ToHit      int32
	Defence    int32
	Absorption int32

	// AttackChargeTime and AttackRelaxTime are the template's cadence. A weapon
	// ASSIGNS over both, so an armed class's real cadence is its weapon's —
	// equipment is out of this story's scope and the divergence is disclosed.
	AttackChargeTime int32
	AttackRelaxTime  int32

	// Protection and Resistance are fixed arrays in their own COLUMN ORDER.
	// Protection is Fire through Astral; its component-resolver permutation is
	// kept separate, as protectionNames explains below. Resistance is already
	// Blade through Shooting in the SAME order canonical XPSlot values 1..5
	// select, so resolveBlow reads Resistance[XPSlot-1] without another
	// permutation (1039). The two families therefore do not share one blanket
	// "not resolver order" rule.
	//
	// WHICH FIVE IS WHICH — the two halves are named the opposite way round
	// from the way the field names read, and reading them the intuitive way is
	// a real mistake this comment exists to stop (0140):
	//
	//   Protection ← columns 19…23, titled `prot Fire..Astral` — the five
	//   ELEMENTAL, magic ones. For a hero they are Spirit/2, clamped
	//   (HERO-RESIST-012), so they move with the spread.
	//
	//   Resistance ← columns 24…28, titled `res.Blade..res.Shooting` — the five
	//   WEAPON DAMAGE-KIND ones, and they are NEVER re-derived for a human
	//   (HERO-RESIST-012); the recompute block's own clear is their only writer.
	//
	// Both bindings are UNIT-COMBAT-015's slot → title → field legend.
	Protection [5]int32
	Resistance [5]int32

	// TypeID and Face are the two columns a placement's class keys are searched
	// against; TokenSize is the footprint side and MovementType the domain code.
	// Nothing in this tree reads the last two yet.
	TypeID       int32
	Face         int32
	TokenSize    int32
	MovementType int32

	// DyingTime is how long the body a unit of this row leaves lies before it
	// is torn down, in the ticks whoever consumes it counts. It is the one
	// column here that describes the unit AFTER it stops being one, which is
	// why it sits apart from the stat block above rather than inside it.
	DyingTime int32

	XPValue int32

	// GoldChance, TreasureMin and TreasureMax are the death-gold columns. The
	// simulation keeps them on the spawned entity because death can happen after
	// a save and must make the same draw after restore.
	GoldChance, TreasureMin, TreasureMax int32
}

// protectionNames are the five ELEMENTAL protection columns' own titles, in the
// order UnitDef.Protection holds them (0140).
//
// THEY ARE DECODED, NOT AUTHORED. UNIT-COMBAT-015 publishes the units table's
// slot → title → field legend, and columns 19…23 are titled `prot
// Fire..Astral`; NewUnitDef stores them in that same column order, slot 19 into
// Protection[0] and slot 23 into Protection[4] (the case arms below). So the
// name against index i here is the title of the column index i was filled from,
// and nothing about it is a guess.
//
// THEY ARE WRONG FOR ANY OTHER ORDER, which is the whole reason this is a
// package-level list beside the store rather than five strings written at a
// screen. The damage resolver indexes the same five fields through a different
// permutation (the Protection field's own comment above), so a consumer holding
// a resolver-ordered array must not label it with these.
//
// The other five — `res.Blade..res.Shooting`, columns 24…28, UnitDef.Resistance
// — have published titles too, and deliberately have no list here: nothing in
// this tree has needed to name them, and a list nobody reads is a list nobody
// notices going stale.
var protectionNames = [5]string{"Fire", "Water", "Air", "Earth", "Astral"}

// ProtectionNames returns the five elemental protection columns' titles, in
// UnitDef.Protection's own order. It returns a FRESH slice — data.SkillNames'
// own convention — so the list above cannot be written through it.
func ProtectionNames() []string {
	return append([]string(nil), protectionNames[:]...)
}

// ElementalSelectorOrder is the damage resolver's own component-resolver
// permutation the Protection field's comment above promises: it converts a
// saved raw elemental selector 1..5 to the UnitDef.Protection/
// SecondaryDamage.Selector index 0..4 (Fire through Astral in column order).
// HERO-DMG2-029's five-way jump table reads raw 1→+0xc4, 2→+0xca, 3→+0xc8,
// 4→+0xc6, 5→+0xcc, which — at the same two-byte-per-slot column spacing
// sourceactor.go's own Protection decode uses — is Fire, Earth, Air, Water,
// Astral: a permutation, not a plain decrement. Water and Earth (raw 2 and 4)
// trade places against strict column order. The index is ElementalSelectorOrder
// [rawKind-1]; callers hold rawKind in 1..5 before indexing (HumanState.
// ProjectionError, applyOriginalActorProfiles's own range check).
var ElementalSelectorOrder = [5]uint8{0, 3, 2, 1, 4}

// unitCtorDefaults is the constructor's own value, written ONCE and read-only
// thereafter. Every -1 cell in a row leaves exactly what stands here, so this is
// the whole of what "an empty cell keeps the default" means.
//
// The three ones are the container fields the base constructor sets together —
// the footprint side, the movement domain and the tier — and are the reason a
// definition built from an all-sentinel row is not the zero value.
//
// MOVE-LIMIT-033, VERSUS-333
var unitCtorDefaults = UnitDef{
	Body: 30, Reaction: 30, Mind: 20, Spirit: 20,
	Health: 30, HealthMax: 30, HealthRegenPeriod: 100,
	Mana: 0, ManaMax: 0, ManaRegenPeriod: 50,
	Speed: 10, RotationSpeed: 16, ScanRange: 5,
	Sight: 0, Reach: 1,
	AttackChargeTime: 8, AttackRelaxTime: 4,
	TokenSize: 1, MovementType: 1, Face: 1,
	DyingTime: 8,
}

// UnitDefaults returns the constructor's defaults: the definition a row of all
// empty cells yields. It returns a copy, so the value above cannot be moved
// through it.
func UnitDefaults() UnitDef { return unitCtorDefaults }

// slotCursor walks a row's parameter array one slot at a time. It is the ONE
// place the sentinel is tested and the ONLY thing that advances the cursor, so a
// case that stored a -1 cannot be written and a slot cannot be skipped by
// accident.
func sentinelRowValues(n int) []int32 {
	row := make([]int32, n)
	for i := range row {
		row[i] = unitSentinel
	}
	return row
}

type slotCursor struct {
	params []int32
	i      int
}

// store copies the slot under the cursor into *dst and ADVANCES THE CURSOR
// EITHER WAY. It reports whether it wrote: a sentinel cell leaves *dst standing,
// and a nil dst consumes the slot and stores nothing at all.
func (c *slotCursor) store(dst *int32) bool {
	v := c.params[c.i]
	c.i++
	if v == unitSentinel || dst == nil {
		return false
	}
	*dst = v
	return true
}

// NewUnitDef builds a definition from one Units row: the constructor's defaults
// with slots 0 to 37 written over them IN ASCENDING ORDER.
//
// The order is not decoration. Two slots go to locals and become a spread, a
// third routes that pair through a switch, and one is read and dropped, so a
// loader written as "field i takes column i" mis-assigns everything from the
// damage pair onward — and would look right, because the eleven fields before it
// would still agree.
//
// name is used for one thing only: naming the entry in a refusal.
//
// It fails on a row shorter than the streamed slots, and on a row whose damage
// selector takes an arm this contract does not model. Both yield the zero value
// rather than a partly filled definition.
func NewUnitDef(name string, params []int32) (UnitDef, error) {
	if len(params) <= lastRequiredSlot {
		return UnitDef{}, fmt.Errorf("data: unit definition %q carries %d parameter(s); slots 0 to %d are required",
			name, len(params), lastRequiredSlot)
	}
	// Older synthetic and authored rows may end at XPValue. Missing treasure
	// cells behave as empty cells; present cells are consumed at their published
	// slots. Padding here keeps the cursor's one-slot-at-a-time invariant.
	row := sentinelRowValues(lastUnitSlot + 1)
	copy(row, params)

	d := unitCtorDefaults
	c := &slotCursor{params: row}

	// The two damage columns land in locals and become a pair; the selector
	// lands in a third. IT IS PRE-SET TO ZERO, which is what makes an absent
	// selector mean arm 0 rather than meaning "-1 <= 0".
	var dmgMin, dmgMax, selector int32

	for slot := 0; slot <= lastUnitSlot; slot++ {
		switch slot {
		case 0:
			c.store(&d.Body)
		case 1:
			c.store(&d.Reaction)
		case 2:
			c.store(&d.Mind)
		case 3:
			c.store(&d.Spirit)
		case 4:
			if c.store(&d.HealthMax) {
				d.Health = d.HealthMax
			}
		case 5:
			c.store(&d.HealthRegenPeriod)
		case 6:
			if c.store(&d.ManaMax) {
				d.Mana = d.ManaMax
			}
		case 7:
			c.store(&d.ManaRegenPeriod)
		case 8:
			c.store(&d.Speed)
		case 9:
			c.store(&d.RotationSpeed)
		case 10:
			c.store(&d.ScanRange)
		case 11:
			c.store(&dmgMin)
		case 12:
			c.store(&dmgMax)
		case 13:
			c.store(&selector)
			switch selector {
			case damageArmSecondPair, damageArmThirdPair:
				return UnitDef{}, fmt.Errorf(
					"data: unit definition %q: damage selector %d routes to a field pair this contract does not model",
					name, selector)
			case damageArmAlwaysHits:
				d.DamageBase, d.DamageSpread = dmgMin, dmgMax-dmgMin
				d.AlwaysHits = true
			default:
				d.DamageBase, d.DamageSpread = dmgMin, dmgMax-dmgMin
			}
		case 14:
			c.store(&d.ToHit)
		case 15:
			c.store(&d.Defence)
		case 16:
			c.store(&d.Absorption)
		case 17:
			c.store(&d.AttackChargeTime)
		case 18:
			c.store(&d.AttackRelaxTime)
		case 19:
			c.store(&d.Protection[0])
		case 20:
			c.store(&d.Protection[1])
		case 21:
			c.store(&d.Protection[2])
		case 22:
			c.store(&d.Protection[3])
		case 23:
			c.store(&d.Protection[4])
		case 24:
			c.store(&d.Resistance[0])
		case 25:
			c.store(&d.Resistance[1])
		case 26:
			c.store(&d.Resistance[2])
		case 27:
			c.store(&d.Resistance[3])
		case 28:
			c.store(&d.Resistance[4])
		case 29:
			c.store(&d.TypeID)
		case 30:
			c.store(&d.Face)
		case 31:
			c.store(&d.TokenSize)
		case 32:
			c.store(&d.MovementType)
		case 33:
			c.store(&d.DyingTime)
		case 34:
			c.store(&d.Withdraw)
		case 35:
			c.store(&d.Wimpy)
		case 36:
			c.store(&d.SeeInvisible)
		case 37:
			c.store(&d.XPValue)
		case 38:
			c.store(&d.GoldChance)
		case 39:
			c.store(&d.TreasureMin)
		case 40:
			c.store(&d.TreasureMax)
		}
	}

	return d, nil
}

// Combat is what a blow reads, as this definition already carries it: the
// eight numbers a non-hero's row states directly, plus Reach and the 0139
// spell pair.
//
// It exists so that both definitions hand a world builder the same type, and so
// the composite literal that fills an entity reads one way whichever collection
// a placement resolved against. A builder that reached into the two definitions'
// own fields instead would have two spellings of one statement, and the second
// one is where a field gets forgotten.
func (d UnitDef) Combat() Combat {
	return Combat{
		DamageBase:       d.DamageBase,
		DamageSpread:     d.DamageSpread,
		SecondaryDamage:  d.SecondaryDamage,
		ToHit:            d.ToHit,
		Defence:          d.Defence,
		Absorption:       d.Absorption,
		AlwaysHits:       d.AlwaysHits,
		AttackChargeTime: d.AttackChargeTime,
		AttackRelaxTime:  d.AttackRelaxTime,
		Reach:            d.Reach,
		SkillSlot:        d.SkillSlot,
		SpellName:        d.SpellName,
		SpellPower:       d.SpellPower,
	}
}

// UnitCapacity is every Unit's carry capacity: the constructor's body x 10 at
// its own default body, set before any template is read and never recomputed
// on the units arm (UNIT-CTOR-004, UNIT-DERIVE-003).
func UnitCapacity() int32 { return unitCtorDefaults.Body * 10 }
