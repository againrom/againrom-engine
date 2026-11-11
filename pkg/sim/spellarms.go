package sim

import "slices"

// spellArm is one spell arm's capabilities, one field per question a
// consumer asks of an arm. Every consumer reads a row through spellArmOf
// with the arm a table row runs (SpellRule.arm, World.spellArm); an arm
// with no row answers zero to every question. The area-effect properties
// of an arm are areaArms' rows.
type spellArm struct {
	arm uint16
	// applies: a row with no damage, heal, area or effect record still has
	// an apply; atActor: a book cast of the row aims at an actor.
	applies, atActor bool
	drain            bool // Drain Life moves its roll to the caster
	heals            bool // the damage pair is healing (MAGIC-DMG-005)
	prismatic        bool // Prismatic Spray's fan of victims
	teleport         bool // the caster is relocated
	pointTarget      bool // casts at a cell without being an area
	fastRange        bool // range grows by power/3 (TEXT-096)
	manualOnly       bool // never cast unbidden
	controlSpirit    bool // raises a bones corpse; a body is its target
	summon           bool // creates a spell creature
	stoneCurse       bool // holds its target (MAGIC-SING-019)
	invisibility     bool // hides its target
	attachAward      bool // an attached effect awards 3% of the victim's health
	selfAimed        bool // the cast cursor's self kind by arm
	sparesMages      bool // an unbidden cast never chooses a mage (owner, DIV-4830, DIV-4831)
	own              ownEffect
	opposite         uint16                                       // the arm an attach removes instead (Bless, Curse)
	magnitude        func(rule SpellRule, mag, power int32) int32 // MAGIC-EFFECT-015
	formulaMagnitude bool                                         // a mod magnitude formula applies
	lastingDuration  bool                                         // the duration follows Spell Duration
	durationLaw      durationLaw
	form             spellForm
	autoCast         autoCastPlace
	light            SpellLighting
	creatureAim      creatureAim // the first game's creature slot-cast aim (MAGIC-237, MAGIC-238)
	caption          captionRow
	// burst: each blast ends in one burst record, its picture 2*arm+9
	// (ANIM-111, ANIM-112); speedFlight: a delivery waits the Euclidean
	// distance over the row's effect speed (SAV-1155).
	burst, speedFlight bool
	// second is the second game's answers where its claims differ.
	second secondArmRow
}

// SpellCaption is the effect line a spell's popup states (TEXT-081,
// TEXT-133).
type SpellCaption uint8

const (
	SpellCaptionNone       SpellCaption = iota
	SpellCaptionSpeed                   // Haste and Slow
	SpellCaptionProtection              // the four protections, at most 100
	SpellCaptionLight                   // Light and Darkness
	SpellCaptionBless
	SpellCaptionCurse
	SpellCaptionRays // Prismatic Spray's ray count
	SpellCaptionShield
)

// captionRow is the caption an arm states; negate: the row's magnitude is
// stated with its sign turned.
type captionRow struct {
	kind   SpellCaption
	negate bool
}

// ownEffect is the effect an arm attaches whatever its row's Effects say;
// scaled: its magnitude is ownScaledMagnitude unless a mod formula sets it.
type ownEffect struct {
	kind   EffectKind
	scaled bool
}

// durationLaw is how Spell Duration scales with power.
type durationLaw uint8

const (
	lawSlow      durationLaw = iota
	lawFastThree             // the fast law on a base of 3
	lawSegmented             // the slow law in segments
)

// spellForm is how a cast's delivery is drawn and timed.
type spellForm uint8

const (
	formFlight spellForm = iota
	formBolt             // a bolt drawn straight to the aim
	formFan              // the prismatic fan
)

// autoCastPlace is the arm's place in the auto-cast levels: level 1..3 of
// autoCastTiers (0 none), its order in that level's list, and its role
// in the support sequence (R2-ENGINE-346, R2-ENGINE-347).
type autoCastPlace struct {
	level, order int8
	role         autoCastRole
}

type autoCastRole uint8

const (
	autoCastNoRole autoCastRole = iota
	autoCastHealer
	autoCastShielder
	autoCastHider
)

// SpellLighting is the light an arm's area writes (MAGIC-UNITLIGHT-057).
type SpellLighting uint8

const (
	SpellLightNone SpellLighting = iota
	SpellLightBright
	SpellLightDark
)

// secondArmRow is an arm's second-game difference: Stone Curse's drawn
// duration and blank effect (R2-ENGINE-025, R2-ENGINE-453); Invisibility's
// P << 4 (R2-ENGINE-022); on every arm, the Effect renewal that keeps the
// larger magnitude and duration (R2-ENGINE-986, R2-ENGINE-454).
type secondArmRow struct {
	drawnDuration, blankCurse, keepsLarger, powerDuration bool
	firstStageDelay                                       uint8
	raise                                                 *corpseRaiseProfile
}

// armSummon is the second game's Summon arm. The first game has no such
// spell, so no first-game id names it.
const armSummon uint16 = 0xfffe

// PrismaticSpellID is the first game's id of the arm that fires rays.
const PrismaticSpellID = 14

var spellArms = [...]spellArm{
	{arm: 1, creatureAim: creatureAimVictim, second: renews},
	{arm: 2, creatureAim: creatureAimVictimCell, burst: true, speedFlight: true, second: renews},
	{arm: 3, creatureAim: creatureAimVictimCell, second: renews},
	{arm: 4, selfAimed: true, second: renews},
	{arm: 5, magnitude: halfPower, formulaMagnitude: true, lastingDuration: true, autoCast: autoCastPlace{3, 2, autoCastNoRole},
		creatureAim: creatureAimCaster, caption: captionRow{kind: SpellCaptionProtection}, second: renews},
	{arm: 6, heals: true, autoCast: autoCastPlace{1, 0, autoCastHealer}, creatureAim: creatureAimCaster, second: renews},
	{arm: 7, magnitude: slowMagnitude, formulaMagnitude: true, creatureAim: creatureAimVictimCell, caption: captionRow{kind: SpellCaptionSpeed},
		second: renews},
	{arm: 8, magnitude: poisonPointMagnitude, formulaMagnitude: true, creatureAim: creatureAimVictimCell, second: renews},
	{arm: 9, creatureAim: creatureAimStepCell, second: renews},
	{arm: 10, magnitude: halfPower, formulaMagnitude: true, lastingDuration: true, autoCast: autoCastPlace{3, 3, autoCastNoRole},
		creatureAim: creatureAimCaster, caption: captionRow{kind: SpellCaptionProtection}, second: renews},
	{arm: 11, applies: true, drain: true, creatureAim: creatureAimVictim, second: renews},
	{arm: 12, magnitude: lightMagnitude, formulaMagnitude: true, light: SpellLightBright, creatureAim: creatureAimVictimCell,
		caption: captionRow{kind: SpellCaptionLight, negate: true}, second: renews},
	{arm: 13, form: formBolt, creatureAim: creatureAimVictim, second: renews},
	{arm: 14, applies: true, prismatic: true, form: formFan, creatureAim: creatureAimVictim, caption: captionRow{kind: SpellCaptionRays},
		second: renews},
	{arm: 15, applies: true, atActor: true, invisibility: true, own: ownEffect{EffectInvisible, false}, lastingDuration: true,
		durationLaw: lawFastThree, autoCast: autoCastPlace{3, 4, autoCastHider}, second: secondArmRow{keepsLarger: true, powerDuration: true},
		creatureAim: creatureAimCaster},
	{arm: 16, magnitude: halfPower, formulaMagnitude: true, lastingDuration: true, autoCast: autoCastPlace{3, 0, autoCastNoRole},
		creatureAim: creatureAimCaster, caption: captionRow{kind: SpellCaptionProtection}, second: renews},
	{arm: 17, magnitude: darknessMagnitude, formulaMagnitude: true, light: SpellLightDark, creatureAim: creatureAimVictimCell,
		caption: captionRow{kind: SpellCaptionLight}, second: renews},
	{arm: 18, applies: true, atActor: true, magnitude: shieldMagnitude, formulaMagnitude: true, lastingDuration: true,
		autoCast: autoCastPlace{1, 1, autoCastShielder}, creatureAim: creatureAimCaster, caption: captionRow{kind: SpellCaptionShield}, second: renews},
	{arm: 19, creatureAim: creatureAimVictimCell, second: renews},
	{arm: 20, applies: true, atActor: true, stoneCurse: true, attachAward: true, lastingDuration: true, durationLaw: lawSegmented,
		second: secondArmRow{drawnDuration: true, blankCurse: true, keepsLarger: true}, creatureAim: creatureAimVictim},
	{arm: 21, creatureAim: creatureAimVictimCell, second: secondArmRow{keepsLarger: true, firstStageDelay: 1}},
	{arm: 22, magnitude: halfPower, formulaMagnitude: true, lastingDuration: true, autoCast: autoCastPlace{3, 1, autoCastNoRole},
		creatureAim: creatureAimCaster, caption: captionRow{kind: SpellCaptionProtection}, second: renews},
	{arm: 23, applies: true, atActor: true, sparesMages: true, own: ownEffect{EffectBless, true}, opposite: 27, formulaMagnitude: true, lastingDuration: true,
		autoCast: autoCastPlace{2, 0, autoCastNoRole}, creatureAim: creatureAimCaster, caption: captionRow{kind: SpellCaptionBless}, second: renews},
	{arm: 24, applies: true, atActor: true, magnitude: hasteMagnitude, formulaMagnitude: true, lastingDuration: true,
		autoCast: autoCastPlace{2, 1, autoCastNoRole}, creatureAim: creatureAimCaster, caption: captionRow{kind: SpellCaptionSpeed}, second: renews},
	{arm: 25, applies: true, atActor: true, controlSpirit: true, second: secondArmRow{keepsLarger: true, raise: &secondCorpseRaise}},
	{arm: 26, applies: true, atActor: true, teleport: true, pointTarget: true, fastRange: true, manualOnly: true,
		creatureAim: creatureAimStepCell, second: renews},
	{arm: 27, applies: true, atActor: true, own: ownEffect{EffectCurse, true}, opposite: 23, attachAward: true, formulaMagnitude: true,
		lastingDuration: true, creatureAim: creatureAimVictim, caption: captionRow{kind: SpellCaptionCurse}, second: renews},
	{arm: 28, applies: true, atActor: true, attachAward: true, magnitude: slowMagnitude, formulaMagnitude: true, lastingDuration: true,
		creatureAim: creatureAimVictim, caption: captionRow{kind: SpellCaptionSpeed}, second: renews},
	{arm: armSummon, applies: true, atActor: true, summon: true, second: renews},
}

// renews is the second-game row of an arm with no other second-game
// difference: its Effect renewal keeps the larger magnitude and duration.
var renews = secondArmRow{keepsLarger: true}

func halfPower(_ SpellRule, _, power int32) int32 { return power / 2 }

func slowMagnitude(_ SpellRule, _, power int32) int32 { return -(power/15 + 1) }

func poisonPointMagnitude(rule SpellRule, mag, power int32) int32 {
	if rule.Second {
		return mag * (power + 45) / 45
	}
	return PoisonMagnitude(mag, power)
}

func lightMagnitude(_ SpellRule, _, power int32) int32 { return power/30 + 1 }

func darknessMagnitude(_ SpellRule, _, power int32) int32 { return -1 - power/30 }

func shieldMagnitude(_ SpellRule, _, power int32) int32 { return power/10 + 3 }

func hasteMagnitude(_ SpellRule, _, power int32) int32 { return power/15 + 1 }

// ownScaledMagnitude is the magnitude of an own effect that scales with
// power, at the width its reader computes in.
func ownScaledMagnitude[T int32 | int64](power T) T { return power*4/5 + 20 }

// SpellCaptionOf is the caption a row states and its number for one
// knower's characteristics: a mod magnitude when one is held, else the
// row's own magnitude at the knower's power. The caption is the one of the
// arm the row runs, in both games.
func SpellCaptionOf(rule SpellRule, c SpellCharacteristics) (SpellCaption, int64) {
	row := spellArmOf(rule.arm())
	kind := row.caption.kind
	switch {
	case kind == SpellCaptionNone:
		return kind, 0
	case kind == SpellCaptionRays:
		return kind, int64(rule.RayLimit(c.Power))
	}
	v := int64(c.Magnitude)
	if !c.HasMagnitude {
		if row.own.scaled {
			v = ownScaledMagnitude(int64(c.Power))
		} else {
			v = int64(row.magnitude(rule, 0, c.Power))
		}
		if row.caption.negate {
			v = -v
		}
	} else if kind == SpellCaptionLight {
		v = -v
	}
	if kind == SpellCaptionProtection {
		v = min(v, 100)
	}
	return kind, v
}

// spellArmSlot indexes the rows of the arms below its length.
var spellArmSlot = func() (out [64]uint8) {
	for i, a := range spellArms {
		if int(a.arm) < len(out) {
			out[a.arm] = uint8(i + 1)
		}
	}
	return out
}()

// spellArmOf is the row of arm, or the zero row of an arm the table lacks.
func spellArmOf(arm uint16) spellArm {
	if int(arm) < len(spellArmSlot) {
		if i := spellArmSlot[arm]; i != 0 {
			return spellArms[i-1]
		}
		return spellArm{}
	}
	for _, a := range spellArms {
		if a.arm == arm {
			return a
		}
	}
	return spellArm{}
}

// secondArm is the row's second-game difference, zero on a first-game row.
func (r SpellRule) secondArm() secondArmRow {
	if !r.Second {
		return secondArmRow{}
	}
	return spellArmOf(r.arm()).second
}

// arm is the arm the row runs: its Arm, or its own id when Arm is unset.
func (r SpellRule) arm() uint16 {
	if r.Arm != 0 {
		return r.Arm
	}
	return r.ID
}

// runsArm reports a table row running arm.
func (r SpellRule) runsArm(arm uint16) bool { return r.arm() == arm }

// spellIDOfArm is the id of the first table row running arm, or arm itself
// when no row runs it.
func (w *World) spellIDOfArm(arm uint16) uint16 {
	for _, r := range w.spells {
		if r.runsArm(arm) {
			return r.ID
		}
	}
	return arm
}

// armSpellID is the row id an attached effect of arm carries. A first-game
// table returns arm itself; a second-game table returns the row running arm,
// or ArmNone, which no effect carries, when no row runs it.
func (w *World) armSpellID(arm uint16) uint16 {
	if len(w.spells) == 0 || !w.spells[0].Second {
		return arm
	}
	for _, r := range w.spells {
		if r.runsArm(arm) {
			return r.ID
		}
	}
	return ArmNone
}

// armWith is the arm of the first row has holds for, or ArmNone.
func armWith(has func(spellArm) bool) uint16 {
	for _, a := range spellArms {
		if has(a) {
			return a.arm
		}
	}
	return ArmNone
}

var (
	hidingArm     = armWith(func(a spellArm) bool { return a.invisibility })
	stoneCurseArm = armWith(func(a spellArm) bool { return a.stoneCurse })
	blessingArm   = armWith(func(a spellArm) bool { return a.own.kind == EffectBless })
	cursingArm    = armWith(func(a spellArm) bool { return a.own.kind == EffectCurse })
	spiritArm     = armWith(func(a spellArm) bool { return a.controlSpirit })
	healingArm    = armWith(func(a spellArm) bool { return a.heals })
	burstArm      = armWith(func(a spellArm) bool { return a.burst })
)

// rowIDBursts and rowIDFliesBySpeed read a row id as an arm in both games,
// as the area walk and the delivery always read it: whatever arm a row
// runs, only the id of Fire Ball's arm bursts and flies by its speed.
func rowIDBursts(id uint16) bool       { return spellArmOf(id).burst }
func rowIDFliesBySpeed(id uint16) bool { return spellArmOf(id).speedFlight }

// autoCastArmsAt is the arms of auto-cast level 1..3, in the level's order.
func autoCastArmsAt(level int8) []uint16 {
	var at []spellArm
	for _, a := range spellArms {
		if a.autoCast.level == level {
			at = append(at, a)
		}
	}
	slices.SortStableFunc(at, func(a, b spellArm) int { return int(a.autoCast.order) - int(b.autoCast.order) })
	out := make([]uint16, len(at))
	for i, a := range at {
		out[i] = a.arm
	}
	return out
}

// autoCastRoleArm is the arm that plays role in the support sequence.
func autoCastRoleArm(role autoCastRole) uint16 {
	return armWith(func(a spellArm) bool { return a.autoCast.role == role })
}

// The effect ids an attached record of an arm carries in the world's table.
func (w *World) hidingSpell() uint16     { return w.armSpellID(hidingArm) }
func (w *World) stoneCurseSpell() uint16 { return w.armSpellID(stoneCurseArm) }
func (w *World) blessingSpell() uint16   { return w.armSpellID(blessingArm) }
func (w *World) cursingSpell() uint16    { return w.armSpellID(cursingArm) }

// controlSpiritSpell is the id of the first row running Control Spirit.
func (w *World) controlSpiritSpell() uint16 { return w.spellIDOfArm(spiritArm) }

// healSpell is the id of the first row running Heal: an escort's heal and the
// unarmed auto-heal ask a caster's book for it (AI-FOLLOWHEAL-118).
func (w *World) healSpell() uint16 { return w.spellIDOfArm(healingArm) }

// holdsStoneCurse reports a target that carries Stone Curse's effect.
func (w *World) holdsStoneCurse(target EntityID) bool { return w.HasEffectArm(target, stoneCurseArm) }

// StoneCurseSpellArm and InvisibilitySpellArm are the arms presentation asks
// World.HasEffectArm about.
func StoneCurseSpellArm() uint16   { return stoneCurseArm }
func InvisibilitySpellArm() uint16 { return hidingArm }

// spellIDTakesBody reports a stored spell id whose row runs Control Spirit.
func (w *World) spellIDTakesBody(id int64) bool {
	return id >= 0 && id <= 0xffff && spellArmOf(w.spellArm(uint16(id))).controlSpirit
}

// lastBookSpellID is the last id of the first game's spell book walk: its
// table is the rows 1 through 28, each its own arm.
const lastBookSpellID = 28

// bookWalkTakes reports a row id a game's creature book walk takes: the
// first game's rows 1 through 28, every row when the walk takes all
// (gameProfile.bookWalkAll).
func bookWalkTakes(id uint16, all bool) bool {
	return all || id >= 1 && id <= lastBookSpellID
}

// bookWalkFallback is the row a first-game walk with no choice falls back
// to; a walk that takes all has none.
func bookWalkFallback(all bool) (uint32, bool) { return lastBookSpellID, !all }

// secondSpellArms maps a second-game spell id to the first-game arm of the
// same spell (R2-ENGINE-019 names the rows, R2-ENGINE-024 groups them by
// arm). Blizzard runs arm 21's sampled staged program: 32 pulses three area
// updates apart, each one cell drawn x then y as RNG(5)-2 and admitted only
// inside 8..dimension-9 (R2-ENGINE-391, R2-ENGINE-392). Ice Missile and
// Diamond Dust reach the blank-effect default as Fire Arrow does and run its
// damage block alone (R2-ENGINE-024). Summon runs armSummon.
var secondSpellArms = [...]uint16{
	1: 1, 2: 2, 3: 3, 4: 5, 5: ArmNone, 6: 8, 7: 21, 8: 10, 9: 9, 10: 13,
	11: 14, 12: 15, 13: 16, 14: 17, 15: 12, 16: ArmNone, 17: 19, 18: 20, 19: 22,
	20: 23, 21: 24, 22: 25, 23: 26, 24: 6, 25: armSummon, 26: 11, 27: 18, 28: 27,
	29: 28,
}

// SecondGameSpellArm is the arm a second-game row id runs; an id past the
// published table runs none.
func SecondGameSpellArm(id uint16) uint16 {
	if int(id) < len(secondSpellArms) && id != 0 {
		return secondSpellArms[id]
	}
	return ArmNone
}

// The exported capabilities pkg/game, pkg/data and pkg/mapload ask of an arm.

func SpellArmTeleports(arm uint16) bool   { return spellArmOf(arm).teleport }
func SpellArmPointTarget(arm uint16) bool { return spellArmOf(arm).pointTarget }
func SpellArmSelfAimed(arm uint16) bool   { return spellArmOf(arm).selfAimed }
func SpellArmHeals(arm uint16) bool       { return spellArmOf(arm).heals }
func SpellArmDrains(arm uint16) bool      { return spellArmOf(arm).drain }
func SpellArmBolt(arm uint16) bool        { return spellArmOf(arm).form == formBolt }
func SpellArmFiresRays(arm uint16) bool   { return spellArmOf(arm).prismatic }
func SpellArmStoneCurse(arm uint16) bool  { return spellArmOf(arm).stoneCurse }
func SpellArmTakesBody(arm uint16) bool   { return spellArmOf(arm).controlSpirit }

func SpellArmLighting(arm uint16) SpellLighting { return spellArmOf(arm).light }

// MagnitudeArm reports an arm whose magnitude a mod formula may set.
func MagnitudeArm(id uint16) bool { return spellArmOf(id).formulaMagnitude }

// DurationArm reports an arm whose duration follows its Spell Duration.
func DurationArm(id uint16) bool { return spellArmOf(id).lastingDuration }
