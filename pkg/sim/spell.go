package sim

import (
	"fmt"
	"math/bits"
	"sort"

	"againrom/pkg/rules"
)

// EffectKind is the installed spell-effect vocabulary this simulation
// consumes. Singular spell arms still select by spell id.
type EffectKind uint8

const (
	EffectNone EffectKind = iota
	EffectHealth
	EffectSpeed
	EffectScanRange
	EffectAbsorption
	EffectProtectionFire
	EffectProtectionWater
	EffectProtectionAir
	EffectProtectionEarth
	EffectBless
	EffectCurse
	EffectInvisible
	EffectHealthRegeneration
	EffectManaRegeneration
)

type EffectMode uint8

const (
	EffectDuration   EffectMode = 1
	EffectContinuous EffectMode = 2
	EffectCharges    EffectMode = 4
	EffectSingleUse  EffectMode = 8
)

// SpellRule is this package's own stdlib-only record of one row of the spell
// table: the id, the mana cost, the school, the maximum range, the damage
// pair, and the two flags a cast reads. It is a PLAIN VALUE — no pointer,
// no slice — so it is copied by assignment and a world holding one holds
// nothing that still reaches into pkg/data, which this package may not
// import (the determinism wall, AGENTS.md).
//
// School selects the target's elemental Protection byte for a damaging
// spell. The field is canonical because all three consumers affect
// deterministic simulation.
type SpellRule struct {
	// Zero retains the timing of native saves written before form95.
	Delivery, EffectSpeed int32
	ID                    uint16
	Complication          uint8
	ManaCost              int32
	School                uint8
	MaxRange              uint8
	DamageMin             int32
	DamageMax             int32
	TargetsUnit           bool
	Damaging              bool
	Defensive             bool
	// Transient book resolution; these fields never enter the table/wire.
	bookInstance  bool
	bookDefensive uint8

	// Restorative is the row's own healing flag, the exact complement of
	// Damaging over the one row that carries damage columns and heals with
	// them. The two are never both true, and a row that is neither is one this
	// build has no arm for and refuses.
	Restorative bool

	// Area is the row's SHAPE: true for a row that lands an area effect
	// standing on a cell, false for one that lands a point effect on a unit. It
	// is the `Distribution system` column reduced to the one bit
	// MAGIC-SHAPE-008 decodes — value 1 is a point effect, anything else an
	// area effect.
	//
	// IT IS NOT TargetsUnit. That is a different column of the same row and
	// the two disagree on shipped rows: Shield is a point effect whose
	// `Spell Target` is 2. TargetsUnit gates a COMMANDED cast's choice of
	// victim; Area decides what an APPLIED cast leaves behind.
	Area         bool
	Distribution uint8
	Radius       uint8

	// AreaDuration is the first term of an area effect's own lifetime in
	// ticks: `(AreaDuration << 4) + (power << 4)/10`, the second term added
	// only for a non-zero power. Read only on a row whose Area is true.
	AreaDuration    int32
	SpellDuration   int32
	EffectKind      EffectKind
	EffectMode      EffectMode
	EffectMagnitude int32
	EffectDuration  uint16

	HealHostile bool     `json:",omitempty"`
	SelfCast    bool     `json:",omitempty"`
	AreaHits    AreaHits `json:",omitempty"`

	// Rays is a mod's Prismatic Spray cap; zero keeps the power rule.
	Rays uint8 `json:",omitempty"`
}

const MaxRays = 100

const PrismaticSpellID = prismaticSpellID

// RayLimit is the one ray-count rule, a maximum.
func (r SpellRule) RayLimit(power int32) int {
	if r.Rays != 0 {
		return int(r.Rays)
	}
	return int(max(min(power/20+2, 7), 0))
}

type AreaHits uint8

const (
	AreaHitsAll AreaHits = iota
	AreaHitsNotOwn
	AreaHitsHostile
)

// GhostTemplate is the definition-table `Units` row a Control Spirit cast
// raises, reduced to the fields the raise reads and carried as a PLAIN VALUE —
// no pointer, no slice — across the pkg/mapload seam exactly as SpellRule is,
// because pkg/sim may not import pkg/data (the determinism wall, AGENTS.md).
//
// Class IS THE FIELD THE DEFECT WAS ABOUT. pkg/game resolves an actor's art
// and its displayed name through classes[e.Class]; a raise that left it at
// zero produced a unit drawn as nothing with an empty name, because no shipped
// units.reg carries a class 0.
//
// The row is the first exact-name `Units` row `Ghost` and supplies every field
// below. ToHit and Defence are persisted in saves but unread by the raise: the
// arm takes the to-hit and defence words, health and the three stats from the
// corpse (MAGIC-249). The row's other columns are Medium (row streaming).
type GhostTemplate struct {
	Class         int32
	TypeID        int32
	Domain        Domain
	Speed         int32
	RotationSpeed int32
	ScanRange     uint8
	Reach         uint8
	TokenSize     uint8
	DyingTime     int32
	XPValue       int32
	Withdraw      int32
	Wimpy         int32
	Humanoid      bool
	NativeBasis   NativeActorBasis

	Protection   [5]int32
	Resistance   [5]uint8
	XPSlot       uint8
	ToHit        int32
	Defence      int32
	Absorption   int32
	DamageBase   int32
	DamageSpread int32
	AttackCharge int32
	AttackRelax  int32
	AlwaysHits   bool
}

// Raisable reports whether the template names a class at all. A world built
// against no definition table, or one whose table ships no `Ghost` row, holds
// the zero value and raises nothing — the cast is refused at admission rather
// than producing an actor no front end can draw.
func (g GhostTemplate) Raisable() bool { return g.Class != 0 }

type durationScale uint8

const (
	durationSlow durationScale = iota
	durationFast
)

// Q56 samples of 1.025^p and 1.05^p for p=0..100. The simulation package
// admits no floating point; fixed samples keep the decoded duration expression
// deterministic while preserving every reachable power.
var durationSlowQ56 = [101]uint64{
	72057594037927936, 73859033888876128, 75705509736098032, 77598147479500464, 79538101166487968,
	81526553695650160, 83564717538041408, 85653835476492448, 87795181363404736, 89990060897489856,
	92239812419927088, 94545807730425264, 96909452923685888, 99332189246778016, 101815493977947472,
	104360881327396144, 106969903360581040, 109644150944595552, 112385254718210432, 115194886086165680,
	118074758238319808, 121026627194277808, 124052292874134736, 127153600195988096, 130332440200887776,
	133590751205909968, 136930519986057696, 140353782985709136, 143862627560351856, 147459193249360640,
	151145673080594624, 154924314907609504, 158797422780299712, 162767358349807200, 166836542308552352,
	171007455866266144, 175282642262922784, 179664708319495840, 184156326027483232, 188760234178170272,
	193479240032624512, 198316221033440128, 203274126559276096, 208355979723257984, 213564879216339424,
	218904001196747904, 224376601226666560, 229986016257333216, 235735666663766528, 241629058330360672,
	247669784788619648, 253861529408335136, 260208067643543488, 266713269334632064, 273381101067997824,
	280215628594697760, 287221019309565184, 294401544792304256, 301761583412111872, 309305622997414592,
	317038263572349952, 324964220161658688, 333088325665700096, 341415533807342592, 349950922152526144,
	358699695206339264, 367667187586497728, 376858867276160128, 386280338958064064, 395937347432015616,
	405835781117816000, 415981675645761344, 426381217536905344, 437040747975327936, 447966766674711104,
	459165935841578880, 470645084237618304, 482411211343558720, 494471491627147648, 506833278917826304,
	519504110890771904, 532491713663041152, 545804006504617088, 559449106667232512, 573435334333913280,
	587771217692260992, 602465498134567552, 617527135587931648, 632965313977629824, 648789446827070592,
	665009182997747328, 681634412572690944, 698675272887008128, 716142154709183232, 734045708576912768,
	752396851291335552, 771206772573618816, 790486941887959296, 810249115435158144, 830505343321037056,
	851267976904062848,
}

var durationFastQ56 = [101]uint64{
	72057594037927936, 75660473739824336, 79443497426815552, 83415672298156336, 87586455913064160,
	91965778708717376, 96564067644153248, 101392271026360912, 106461884577678960, 111784978806562912,
	117374227746891056, 123242939134235616, 129405086090947408, 135875340395494784, 142669107415269536,
	149802562786033024, 157292690925334656, 165157325471601408, 173415191745181472, 182085951332440576,
	191190248899062592, 200749761344015744, 210787249411216544, 221326611881777376, 232392942475866240,
	244012589599659584, 256213219079642560, 269023880033624704, 282475074035305952, 296598827737071296,
	311428769123924864, 327000207580121088, 343350217959127168, 360517728857083520, 378543615299937728,
	397470796064934656, 417344335868181376, 438211552661590464, 460122130294670016, 483128236809403520,
	507284648649873728, 532648881082367424, 559281325136485824, 587245391393310208, 616607660962975744,
	647438044011124480, 679809946211680768, 713800443522264832, 749490465698378112, 786964988983297024,
	826313238432461952, 867628900354085120, 911010345371789312, 956560862640378880, 1004388905772397824,
	1054608351061017728, 1107338768614068736, 1162705707044772096, 1220840992397010944, 1281883042016861440,
	1345977194117704704, 1413276053823589888, 1483939856514769408, 1558136849340507904, 1636043691807533312,
	1717845876397910272, 1803738170217805824, 1893925078728696064, 1988621332665131008, 2088052399298387712,
	2192455019263307264, 2302077770226472704, 2417181658737796096, 2538040741674686464, 2664942778758420480,
	2798189917696342016, 2938099413581158912, 3085004384260217344, 3239254603473228288, 3401217333646889472,
	3571278200329234432, 3749842110345696256, 3937334215862981120, 4134200926656130560, 4340910972988937216,
	4557956521638384128, 4785854347720303616, 5025147065106319360, 5276404418361634816, 5540224639279717376,
	5817235871243703296, 6108097664805889024, 6413502548046183424, 6734177675448493056, 7070886559220917248,
	7424430887181964288, 7795652431541062656, 8185435053118115840, 8594706805774021632, 9024442146062723072,
	9475664253365860352,
}

// spellPowerMax is the highest power a cast reaches: the sum of school skill
// and Mind less 30, clamped to 0..spellPowerMax.
const spellPowerMax = 255

// durationTickCeiling is the largest duration word an effect can hold. A longer
// result saturates there.
const durationTickCeiling = 65535

// lastingTicks is the duration word of a power-scaled effect: base sixteenths
// of a unit times 1.025^power (durationSlow) or 1.05^power (durationFast),
// truncated, saturating at durationTickCeiling.
func lastingTicks(power int32, base int32, scale durationScale) uint16 {
	if base <= 0 {
		return 0
	}
	if power < 0 {
		power = 0
	} else if power > spellPowerMax {
		power = spellPowerMax
	}
	m := uint64(base) * 16
	var v uint64
	if power <= 100 {
		factor := durationSlowQ56[power]
		if scale == durationFast {
			factor = durationFastQ56[power]
		}
		hi, lo := bits.Mul64(m, factor)
		if hi>>56 != 0 {
			return durationTickCeiling
		}
		v = hi<<8 | lo>>56
	} else {
		f := durationSlowAbove[power-101]
		if scale == durationFast {
			f = durationFastAbove[power-101]
		}
		carry, _ := bits.Mul64(f.Frac, m)
		v = f.Whole*m + carry
	}
	if v > durationTickCeiling {
		return durationTickCeiling
	}
	return uint16(v)
}

// segmentedTicks is the duration word of Invisibility and Stone Curse. Up to
// power 100 it is lastingTicks. Above, the duration is k whole durations at
// power 100 plus the duration at the remaining power, where
// k = floor((power-1)/100), so power 200 holds two durations at 100 and power
// 250 holds two and the duration at 50.
func segmentedTicks(power int32, base int32, scale durationScale) uint16 {
	if power > spellPowerMax {
		power = spellPowerMax
	}
	if power <= 100 {
		return lastingTicks(power, base, scale)
	}
	k := (power - 1) / 100
	total := uint32(k)*uint32(lastingTicks(100, base, scale)) + uint32(lastingTicks(power-100*k, base, scale))
	if total > durationTickCeiling {
		return durationTickCeiling
	}
	return uint16(total)
}

// spellLastingTicks is the power-scaled duration word of a row whose effect
// duration follows its Spell Duration column: Invisibility on the fast law and
// a base of 3, Stone Curse segmented, every other row on the slow law.
func spellLastingTicks(rule SpellRule, power int32) uint16 {
	switch rule.ID {
	case 15:
		return segmentedTicks(power, 3, durationFast)
	case 20:
		return segmentedTicks(power, rule.SpellDuration, durationSlow)
	}
	return lastingTicks(power, rule.SpellDuration, durationSlow)
}

func (w *World) ordinaryEffect(ci, ti int, rule SpellRule, power int32) bool {
	if rule.Delivery == 2 {
		return w.queuePointDelivery(ci, ti, rule, power)
	}
	applied := w.ordinaryEffectPayload(ci, ti, rule, power)
	if applied {
		w.pointAttribution(ci, ti, rule)
	}
	return applied
}

// ordinaryAreaEffect shares the payload with an ordinary point application but
// runs AreaEffect's distinct post-payload attribution programme.
func (w *World) ordinaryAreaEffect(ci, ti int, rule SpellRule, power int32) bool {
	applied := w.ordinaryEffectPayload(ci, ti, rule, power)
	if applied {
		w.areaAttribution(ci, ti, rule)
	}
	return applied
}

func (w *World) ordinaryEffectPayload(ci, ti int, rule SpellRule, power int32) bool {
	if ti < 0 || ti >= len(w.entities) {
		return false
	}
	// One gate serves direct, area, script and item application: restorative
	// Heal may revive above -10, other support rows cannot, damaging rows may
	// finish through -9, and Control Spirit owns its explicit corpse route.
	if !spellTargetable(w.entities[ti], rule) {
		return false
	}
	casterID := ^EntityID(0)
	if ci >= 0 && ci < len(w.entities) {
		casterID = w.entities[ci].ID
	}
	targetID := w.entities[ti].ID
	if rule.Restorative {
		// The apply arm refuses a target the caster's player is hostile to
		// (MAGIC-TARGET-017). The cast itself is admitted and paid, and
		// its marks and event are produced; only the health stays unchanged.
		// A script cast has no caster and no side.
		if !rule.HealHostile && ci >= 0 && ci < len(w.entities) && w.entities[ti].Owner != w.entities[ci].Owner &&
			w.hostileTo(w.entities[ci], w.entities[ti]) {
			return true
		}
		w.applySpellHealing(ti, rule, power)
		return true
	}
	if rule.ID == 11 && !rule.Damaging {
		w.flipOnBlow(ci, ti)
		// Drain Life's own arm caps the roll at victim health+10 and sends
		// that amount to the skill sink before either pool changes. It never
		// enters the resisted direct-damage resolver (MAGIC-ITEMTRAIN-116).
		base, spread := spellDamageUnder(w.rules, rule, power)
		amount := base + int64(w.rng.uniform(int32(max(0, spread))))
		amount = min(amount, int64(w.entities[ti].HP)+10)
		if amount <= 0 {
			return true
		}
		w.awardSpellDamage(ci, ti, rule, amount)
		before := w.entities[ti]
		w.entities[ti].setCurrentHealth(int32(int64(w.entities[ti].HP) - amount))
		w.reportHealthLoss(before, ti)
		w.clearFelled(ti)
		if ci >= 0 {
			hp := int64(w.entities[ci].HP) + amount
			if hp > int64(w.entities[ci].MaxHP) {
				hp = int64(w.entities[ci].MaxHP)
			}
			w.entities[ci].setCurrentHealth(int32(hp))
		}
		return true
	}
	if rule.Damaging {
		w.flipOnBlow(ci, ti)
		before := w.entities[ti].HP
		w.applySpellDamage(ti, rule, power)
		// The direct-damage resolver precedes the envelope tail. The temporary
		// kind is the spell's damage/school kind; a normal point or area tail
		// replaces it with the actual spell id after this payload returns.
		w.resolveDamageAttribution(ci, ti, int8(rule.School))
		if removed := int64(before) - int64(w.entities[ti].HP); removed > 0 {
			w.awardSpellDamage(ci, ti, rule, removed)
		}
		return true
	}
	switch rule.ID {
	case 26:
		if ci < 0 {
			return false
		}
		to := cell{x: w.entities[ti].X, y: w.entities[ti].Y}
		if !w.attachFootprint(ci, to.x, to.y) {
			// The cast has been admitted and paid. A relocation veto does not
			// undo that cast or suppress its presentation (MAGIC-TELEPORT-174).
			return true
		}
		w.entities[ci].X, w.entities[ci].Y = to.x, to.y
		w.invalidateActorMotion(w.entities[ci].ID, "native teleport supersedes original movement")
		w.entities[ci].clearStride()
		w.entities[ci].HasTarget = false
		w.routes[ci] = nil
		return true
	case 25:
		if ci < 0 || w.entities[ti].Alive() || w.entities[ti].Decay != DecayBones {
			return false
		}
		consumed, _ := w.raiseControlSpirit(ci, ti)
		return consumed
	}
	kind, mag, duration, mode := w.pointEffect(ti, rule, power)
	if kind == EffectNone {
		return false
	}
	if mode&effectTimedModes == 0 {
		if _, ok := w.applyEffectDelta(ti, kind, mag); !ok && w.entities[ti].ActorLoad.Source.Class != 0 {
			return false
		}
		w.clearFelled(ti)
		return true
	}
	applied := w.attachEffect(targetID, casterID, rule, kind, mag, duration, mode)
	if applied && (rule.ID == 20 || rule.ID == 27 || rule.ID == 28) {
		w.awardSpellDamage(ci, ti, rule, int64(w.entities[ti].MaxHP)*3/100)
	}
	return applied
}

// MAGIC-ATTACH-016 stores all three low-bit modes. MAGIC-CONSUME-144 advances
// only duration/continuous counters; a charges-only attachment has no clock.
const effectTimedModes = EffectDuration | EffectContinuous | EffectCharges

// pointEffect is the kind, magnitude, duration and mode one ordinary row lands
// on victim ti at power — MAGIC-EFFECT-015's per-arm magnitudes and
// MAGIC-SING-019 (d)'s duration scaling, in one place.
//
// IT IS THE ONE PLACE THEY ARE COMPUTED, so the admission predicate
// (pointEffectRefusal) and the apply (ordinaryEffect) cannot come to disagree
// about whether an effect can land at all. That disagreement is what let a
// book cast pay its mana for a release the apply then refused.
//
// IT READS THE VICTIM AND WRITES NOTHING. Protection[3] enters spell 20's
// duration, so the function takes an index rather than a rule alone; nothing
// else about ti is read and no field of it is touched.
func (w *World) pointEffect(ti int, rule SpellRule, power int32) (EffectKind, int32, uint16, EffectMode) {
	switch rule.ID {
	case 23, 27:
		kind := EffectBless
		if rule.ID == 27 {
			kind = EffectCurse
		}
		mag := power*4/5 + 20
		if m, ok := magnitudeUnder(w.rules, rule, power); ok {
			mag = m
		}
		return kind, mag, spellLastingTicksUnder(w.rules, rule, power), EffectDuration
	case 15:
		return EffectInvisible, 1, spellLastingTicksUnder(w.rules, rule, power), EffectDuration
	}
	kind, mode := rule.EffectKind, rule.EffectMode
	mag, duration := rule.EffectMagnitude, spellPointDurationUnder(w.rules, rule, power)
	switch rule.ID {
	case 5, 10, 16, 22:
		mag = power / 2
	case 18:
		mag = power/10 + 3
	case 24:
		mag = power/15 + 1
	case 7, 28:
		mag = -(power/15 + 1)
	case 8:
		mag = mag * (power + 30) / 30
	case 12:
		mag = power/30 + 1
	case 17:
		mag = -1 - power/30
	case 20:
		// THE MAGNITUDE IS THE ROW'S OWN COLUMN AND NOT A LITERAL HERE.
		// MAGIC-EFFECT-015 names stone_curse as one of the two arms that
		// write no `+0x40`, which is what makes the `Effects` column's own
		// number — the shipped `absorbtion=+5` — the magnitude. A 5 written
		// here read the same on the shipped table and made the column dead,
		// which is the customisation seam contract.md:66 claims.
		//
		// THE DURATION IS SCALED BY protectionEarth (MAGIC-SING-019 (d)),
		// floored at one tick: the only place a resistance shortens an
		// effect instead of reducing damage.
		if p := int32(0); ti >= 0 && ti < len(w.entities) {
			p = w.entities[ti].Protection[3]
			if p <= 0 {
				break
			}
			if p > 100 {
				p = 100
			}
			duration = uint16(uint32(duration) * uint32(100-p) / 100)
			if duration == 0 {
				duration = 1
			}
		}
	}
	if m, ok := magnitudeUnder(w.rules, rule, power); ok && MagnitudeArm(rule.ID) {
		mag = m
	}
	return kind, mag, duration, mode
}

// pointEffectRefusal names the row-specific condition an ordinary apply
// requires of its victim, or the empty string when the apply will land.
//
// Control Spirit still requires a bones corpse. Teleport placement is an
// application-time veto after admission and payment (MAGIC-TELEPORT-174).
//
// IT IS READ-ONLY AND ROLLS NOTHING, which is what lets the same function
// answer for BookSpellRefusal, the lawful real-save instrument, and for the
// pre-cost gate inside castSpell.
func (w *World) pointEffectRefusal(ci, vi int, rule SpellRule, power int32) string {
	if rule.Area || rule.Damaging || rule.Restorative || rule.ID == 11 || rule.ID == 14 {
		return ""
	}
	switch rule.ID {
	case 25:
		if ci < 0 {
			return "control spirit has no caster"
		}
		if w.entities[vi].Alive() || w.entities[vi].Decay != DecayBones {
			return "control spirit target is not a bones corpse"
		}
		if _, available := w.NextEntityID(); !available {
			return "no entity id left to raise into"
		}
		if !w.ghost.Raisable() {
			return "no ghost template loaded"
		}
		return ""
	case 26:
		if ci < 0 {
			return "teleport has no caster"
		}
		return ""
	}
	kind, _, duration, mode := w.pointEffect(vi, rule, power)
	if kind == EffectNone {
		return "row carries no effect kind"
	}
	if mode&effectTimedModes != 0 && duration == 0 {
		return "effect duration is zero"
	}
	return ""
}

// raisedGhost is the actor a Control Spirit cast puts where the corpse stood:
// the world's own GhostTemplate, with the caster's Owner and Group, the
// corpse's cell and facing, and the seven stores the arm takes off the corpse:
// `reaction/2 + 1`, Mind, Spirit, `healthMax/2` with health equal to it, and
// the first word of the to-hit block and of the defence block (MAGIC-249).
//
// IT GAINS NO EXPERIENCE. GainsXP stays false, on the creature arm's own rule
// (pkg/mapload blockFor): a units row derives no class that could earn.
func (w *World) raisedGhost(ci, ti int, id EntityID) (Entity, bool) {
	if !w.ghost.Raisable() {
		return Entity{}, false
	}
	g, src, caster := w.ghost, w.entities[ti], w.entities[ci]
	maxHP := src.MaxHP / 2
	if maxHP < 1 {
		maxHP = 1
	}
	e := Entity{
		ID: id, X: src.X, Y: src.Y, PostX: src.X, PostY: src.Y,
		Owner: caster.Owner, Group: caster.Group,
		// The ghost takes the corpse's facing with no turn in progress;
		// DesiredFacing must equal Facing then or the byte form refuses it.
		Facing: src.Facing, DesiredFacing: src.Facing,
		HP: maxHP, MaxHP: maxHP,
		Class: g.Class, TypeID: g.TypeID, Domain: g.Domain,
		Speed: g.Speed, RotationSpeed: g.RotationSpeed,
		ScanRange: g.ScanRange, Reach: g.Reach,
		TokenSize: g.TokenSize, DyingTime: g.DyingTime, XPValue: g.XPValue,
		Withdraw: g.Withdraw, Wimpy: g.Wimpy,
		Humanoid:     g.Humanoid,
		NativeBasis:  g.NativeBasis,
		Protection:   g.Protection,
		Resistance:   g.Resistance,
		XPSlot:       g.XPSlot,
		ToHit:        src.ToHit,
		Defence:      src.Defence,
		Absorption:   g.Absorption,
		DamageBase:   g.DamageBase,
		DamageSpread: g.DamageSpread,
		AttackCharge: g.AttackCharge,
		AttackRelax:  g.AttackRelax,
		AlwaysHits:   g.AlwaysHits,
		Reaction:     src.Reaction/2 + 1, Mind: src.Mind, Spirit: src.Spirit,
		ActorState: actorStateGuard,
	}
	if e.Reach == 0 {
		e.Reach = 1
	}
	if e.TokenSize == 0 {
		e.TokenSize = 1
	}
	return e, true
}

func spellApplicable(rule SpellRule) bool {
	if rule.Area || rule.Damaging || rule.Restorative || rule.EffectKind != EffectNone {
		return true
	}
	switch rule.ID {
	case 11, 14, 15, 18, 20, 23, 24, 25, 26, 27, 28:
		return true
	}
	return false
}

// normaliseSpells is the constructor's own act over a caller's spell table:
// a COPY, in the caller's own order — unlike normaliseSacks this does not
// sort, because that order is the order a front end reads a book's spells
// out in, and reordering it here would take it away from whoever built the
// table.
//
// IT IS A LINEAR SCAN AND NOT A MAP, on normaliseSacks' own ground
// (sack.go): the table this build ever reads holds 28 rows, and a map here
// would put Go's randomised iteration order on a path that builds canonical
// state, for no benefit a scan does not already give at this size.
func normaliseSpells(in []SpellRule) ([]SpellRule, error) {
	out := append([]SpellRule(nil), in...)
	for i, sp := range out {
		if sp.bookInstance || sp.bookDefensive != 0 {
			return nil, fmt.Errorf("sim: resolved book instance is not a table row")
		}
		if sp.ID == 0 {
			return nil, fmt.Errorf("sim: spell at index %d is id 0, which names no row", i)
		}
		if sp.ManaCost < 0 {
			return nil, fmt.Errorf("sim: spell %d has a negative mana cost %d", sp.ID, sp.ManaCost)
		}
		if sp.HealHostile && !sp.Restorative || sp.SelfCast && !sp.Damaging || sp.AreaHits > AreaHitsHostile || sp.AreaHits != AreaHitsAll && !sp.Area ||
			sp.Rays != 0 && (sp.ID != prismaticSpellID || sp.Rays > MaxRays) {
			return nil, fmt.Errorf("sim: spell %d sets a target filter its row has no arm for", sp.ID)
		}
		if sp.DamageMin < 0 || sp.DamageMax < 0 {
			return nil, fmt.Errorf("sim: spell %d has a negative damage column (%d, %d)",
				sp.ID, sp.DamageMin, sp.DamageMax)
		}
		for j := 0; j < i; j++ {
			if out[j].ID == sp.ID {
				return nil, fmt.Errorf("sim: spell id %d is repeated", sp.ID)
			}
		}
	}
	return out, nil
}

// Spells returns the world's spell table as a fresh copy, in the world's own
// order: mutating the result, or a caller reusing the slice beneath it,
// cannot reach the world it came from — Entities' and Sacks' own rule
// (world.go, sack.go). SpellRule is a plain value, so the slice copy alone
// is the whole of the guarantee; there is no inner slice or pointer a
// shallow copy could still share.
func (w *World) Spells() []SpellRule {
	return append([]SpellRule(nil), w.spells...)
}

// Ghost returns the world's ghost template. GhostTemplate is a plain value
// with no slice or pointer in it, so the return is already a copy and a caller
// that mutates it cannot reach the world it came from — Spells' own rule.
//
// It exists for pkg/mapload's mission REBUILDS (start.go), which read the
// spell table back off the world FromALMWith already built rather than
// resolving the definition table a second time. The template travels the same
// way for the same reason.
func (w *World) Ghost() GhostTemplate { return w.ghost }

func (w *World) findSpell(id uint32) (SpellRule, bool) {
	for _, sp := range w.spells {
		if uint32(sp.ID) == id {
			return sp, true
		}
	}
	return SpellRule{}, false
}

// isMage reports whether e is a mage: an entity with a mana pool (DECODED).
// The predicate is one expression in one place, matching the routine the
// game's own class bit is itself set from — it is derived rather than
// chosen, and there is nowhere else in this package it could come to differ
// from this.
func isMage(e Entity) bool { return e.MaxMana > 0 }

func weaponSpellFor(spells []SpellRule, e Entity) (SpellRule, bool) {
	if e.WeaponSpell == 0 || !isMage(e) {
		return SpellRule{}, false
	}
	for _, sp := range spells {
		if uint32(sp.ID) == uint32(e.WeaponSpell) {
			return sp, true
		}
	}
	return SpellRule{}, false
}

func (w *World) weaponSpell(e Entity) (SpellRule, bool) {
	return weaponSpellFor(w.spells, e)
}

// weaponRiderSpellFor answers the FIGHTER's rider arm of a weapon-borne
// spell (R3-B3, MAGIC-ITEM-007): the exact negation of weaponSpellFor's own
// isMage gate, so a caster's replacement arm and a fighter's rider arm can
// never both claim the same strike. `L05093` is the fighter predicate, and
// `HERO-CLASS-020` fixes it as the exact negation of the mage test
// weaponSpellFor reads.
func weaponRiderSpellFor(spells []SpellRule, e Entity) (SpellRule, bool) {
	if e.WeaponSpell == 0 || isMage(e) {
		return SpellRule{}, false
	}
	for _, sp := range spells {
		if uint32(sp.ID) == uint32(e.WeaponSpell) {
			return sp, true
		}
	}
	return SpellRule{}, false
}

// WeaponSpellCharacteristics is the state-free spell projection used by item
// and character presentation. Damage uses the same powered pair as the live
// apply, duration uses the same pre-target point-effect clock, and MaxRange is
// the raw weapon-release reach rather than a book cast's power-adjusted range.
// RayCount is SpellRule.RayLimit.
type WeaponSpellCharacteristics struct {
	SpellID                uint16
	DamageMin, DamageMax   int64
	DurationTicks          uint16
	MaxRange               uint8
	RayCount               uint8
	HasDamage, HasDuration bool
}

func weaponSpellCharacteristics(r Rules, rule SpellRule, power int32) (WeaponSpellCharacteristics, bool) {
	if rule.ID == 25 || !spellApplicable(rule) {
		return WeaponSpellCharacteristics{}, false
	}
	// Drain Life projects its transfer interval without the damaging flag;
	// Stone Curse projects duration instead of damage.
	if !rule.Damaging && rule.ID != 11 && rule.ID != 20 {
		return WeaponSpellCharacteristics{}, false
	}
	if rule.Damaging && !rule.TargetsUnit && rule.ID != 14 {
		return WeaponSpellCharacteristics{}, false
	}

	out := WeaponSpellCharacteristics{SpellID: rule.ID, MaxRange: rule.MaxRange}
	if rule.Damaging || rule.ID == 11 {
		base, spread := spellDamageUnder(r, rule, power)
		if spread < 0 {
			spread = 0
		}
		out.DamageMin, out.DamageMax, out.HasDamage = base, base+spread, true
	}
	if rule.ID == 20 {
		out.DurationTicks = spellPointDurationUnder(r, rule, power)
		out.HasDuration = true
	}
	if rule.ID == 14 {
		out.RayCount = uint8(rule.RayLimit(power))
	}
	return out, true
}

// WeaponSpellCharacteristicsFor resolves e's caster-item arm and projects the
// exact row at the attachment's unclamped power. A fighter rider is excluded
// by weaponSpellFor's mage gate; it is not the tooltip route described here.
func WeaponSpellCharacteristicsFor(r Rules, e Entity, spells []SpellRule) (WeaponSpellCharacteristics, bool) {
	rule, ok := weaponSpellFor(spells, e)
	if !ok {
		return WeaponSpellCharacteristics{}, false
	}
	return weaponSpellCharacteristics(r, rule, e.WeaponSpellLevel)
}

// WeaponSpellDamageFor reports the damage interval a weapon-borne spell would
// use for e against spells. It is retained for combat and older presentation
// callers; the richer characteristics projection owns lookup and arithmetic.
func WeaponSpellDamageFor(r Rules, e Entity, spells []SpellRule) (base, spread int64, ok bool) {
	characteristics, ok := WeaponSpellCharacteristicsFor(r, e, spells)
	if !ok || !characteristics.HasDamage {
		return 0, 0, false
	}
	return characteristics.DamageMin,
		characteristics.DamageMax - characteristics.DamageMin, true
}

// WeaponSpellDamage reports the exact damage interval used when id's current
// attack is replaced by its weapon-borne spell. It reaches the same
// weaponSpell lookup and spellDamage arithmetic as releaseWeaponSpell, including
// the live path's negative-spread fold and Drain Life's transfer interval.
// A physical attack or a spell without a projected interval answers false.
func (w *World) WeaponSpellDamage(id EntityID) (base, spread int64, ok bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return 0, 0, false
	}
	return WeaponSpellDamageFor(w.rules, w.entities[i], w.spells)
}

// knowsSpell reports whether e's book has learned id: bit id of KnownSpells
// set (FR-4b). id is unsigned throughout its one caller's chain, built from
// a command's Y by an unsigned conversion, so the shift count is never
// negative; and a shift PAST the mask's own 32 bits answers false rather
// than panicking — Go's shift is total over any non-negative count — so an
// id no bit of this mask could ever carry is simply not known.
func knowsSpell(e Entity, id uint32) bool {
	return e.Book.State != BookAbsent && e.KnownSpells&(uint32(1)<<id) != 0
}

func spellPower(level, mind int32) int32 {
	p := level + mind - 30
	switch {
	case p < 0:
		return 0
	case p > spellPowerMax:
		return spellPowerMax
	}
	return p
}

// spellRange is the decoded cast reach after power. Teleport is the sole
// faster-growing arm; every other spell adds power/30 to its installed base.
func spellRange(rule SpellRule, power int32) int64 {
	if rule.bookInstance {
		return int64(rule.MaxRange)
	}
	// MAGIC-ARM-014's first gate is exact: a zero maximum means no range
	// bonus is computed at all. Shield and Fire Sacrifice ship on this arm.
	if rule.MaxRange == 0 {
		return 0
	}
	divisor := int32(30)
	if rule.ID == 26 {
		divisor = 3
	}
	return int64(rule.MaxRange) + int64(power/divisor)
}

// clampByte is a damage component held in the byte the original's effect
// record carries. A value above 255 saturates, so a power above 100 never
// wraps a large roll to a small one. A negative value keeps its two's
// complement byte, as before.
func clampByte(v int64) uint8 {
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func spellDamage(dmin, dmax, power int32) (base, spread int64) {
	f := int64(power) + 30
	base = int64(dmin) * f / 30
	spread = int64(dmax)*f/30 - base
	return base, spread
}

// SpellCharacteristics is the actor-dependent projection consumed by the
// spellbook popup. It is rebuilt from current canonical actor state and the
// immutable spell row; no UI cache or second copy of power/range/damage
// arithmetic can become stale after training, equipment or an attached effect.
//
// Power is the signed level the caption formulas take. RecordPower is the
// level the popup's spell record is filled at, which differs from Power only
// for a skill and Mind sum below 30 (TEXT-097); Range and Duration are the record's own byte and
// word at RecordPower (TEXT-096). The record's damage bytes are floating-point
// products the presentation tier forms from the row's columns and RecordPower.
type SpellCharacteristics struct {
	SkillLevel, SkillXP           int32
	Power                         int32
	RecordPower                   int32
	ManaCost                      int32
	Range                         int64
	Duration                      uint16
	Radius                        uint8
	DamageFactor, Magnitude       int32
	HasDamageFactor, HasMagnitude bool
}

func SpellCharacteristicsFor(r Rules, e Entity, rule SpellRule) SpellCharacteristics {
	if resolved, ok := BookRuleFor(e, rule); ok {
		rule = resolved
	}
	var level, xp int32
	if rule.School < skillSlots {
		level, xp = e.Skill[rule.School], e.SkillXP[rule.School]
	}
	power := spellPowerUnder(r, rule, level, e.Mind)
	record := spellRecordPowerUnder(r, rule, level, e.Mind)
	c := SpellCharacteristics{SkillLevel: level, SkillXP: xp, Power: power, RecordPower: record,
		ManaCost: rule.ManaCost, Range: spellRecordRangeUnder(r, rule, record),
		Duration: spellRecordDurationUnder(r, rule, record), Radius: rule.Radius}
	c.DamageFactor, c.HasDamageFactor = r.SpellFormulaAt(rules.FormulaDamage, rule.ID, record)
	c.Magnitude, c.HasMagnitude = magnitudeUnder(r, rule, power)
	return c
}

// spellRecordPower is the level the spellbook fold fills a spell record at:
// skill plus mind minus 30, clamped to spellPowerMax like a cast's power. A
// sum below 30 keeps the original byte fold, which wraps it to 226..255 and
// reads 100 (TEXT-097).
func spellRecordPower(level, mind int32) int32 {
	p := level + mind - 30
	switch {
	case p < 0:
		return 100
	case p > spellPowerMax:
		return spellPowerMax
	}
	return p
}

// spellRecordRange is the record's byte +9: the row's range column plus
// power/30 when that column is non-zero, power/3 for Teleport (TEXT-096).
func spellRecordRange(rule SpellRule, power int32) int64 {
	if rule.bookInstance {
		return int64(rule.MaxRange)
	}
	base := int32(rule.MaxRange)
	switch {
	case rule.ID == 26:
		base += power / 3
	case base != 0:
		base += power / 30
	}
	return int64(uint8(base))
}

// spellRecordDuration is the record's word +0x10 in sixteenths of the duration
// unit: the Spell Duration column through the power ramp when it is positive,
// else the Area Effect Duration column plus power*16/10 when that is positive,
// else 0 (TEXT-096). Spell 15 takes the fast ramp on a base of 3.
func spellRecordDuration(rule SpellRule, power int32) uint16 {
	switch {
	case rule.SpellDuration > 0:
		return spellLastingTicks(rule, power)
	case rule.AreaDuration > 0:
		return uint16(int64(rule.AreaDuration)<<4 + (int64(power)<<4)/10)
	}
	return 0
}

// spellPointDuration is the one point-effect duration projection used by both
// application and presentation. Target-specific resistance (Stone Curse's
// target protection adjustment) remains in ordinaryEffect after this base.
func spellPointDuration(rule SpellRule, power int32) uint16 {
	switch rule.ID {
	case 15, 5, 10, 16, 18, 20, 22, 23, 24, 27, 28:
		return spellLastingTicks(rule, power)
	default:
		return rule.EffectDuration
	}
}

func (w *World) castSpell(ci int, victim EntityID, spellID uint32, aimX, aimY int32, obs *castObs) bool {
	caster := &w.entities[ci]
	casterID := caster.ID
	castFromX, castFromY := caster.X, caster.Y
	// 1. held by the world is already true — ci came from the lookup the
	// caller (stepWorld) already made. What is left is alive.
	if !caster.Alive() {
		return false
	}
	// 2. a mage, and one that knows this spell.
	if !bookCaster(*caster) || !knowsSpell(*caster, spellID) {
		return false
	}
	// 3. the spell id must name a row of the world's own table.
	rule, ok := w.bookSpell(*caster, spellID)
	if !ok {
		return false
	}
	// 4. The row must name one of the decoded ordinary effect arms. Point and
	// area delivery both converge here; the applicability table therefore owns
	// protections, modifiers, healing and damage instead of a caller-specific
	// allow-list.
	if !spellApplicable(rule) {
		return false
	}
	if !rule.Area && !rule.TargetsUnit && rule.ID != 15 && rule.ID != 18 && rule.ID != 20 &&
		rule.ID != 23 && rule.ID != 24 && rule.ID != 25 && rule.ID != 26 && rule.ID != 27 && rule.ID != 28 {
		return false
	}
	vi := indexOfEntity(w.entities, victim)
	if vi < 0 || !spellTargetable(w.entities[vi], rule) ||
		prismaticBodyPrimary(w.entities[vi], rule) && !w.bookAlreadyPaid(caster.ID, spellID) {
		return false
	}
	if victim == caster.ID && rule.Damaging && !rule.SelfCast {
		return false
	}
	target := &w.entities[vi]
	healthBefore := target.HP
	// Actor-directed releases revalidate current perception before any cost,
	// turn, event or effect. Once this returns true and the release applies,
	// later visibility is irrelevant to the already committed client object.
	if !w.actorSeesEntity(ci, vi) {
		return false
	}
	// A target without a health system spends nothing. A commanded heal on a
	// target at its maximum is cast: the cost is paid and the apply tail clamps
	// health at the maximum. Only the unbidden target choice skips full targets.
	if rule.Restorative && target.MaxHP <= 0 {
		return false
	}
	var level int32
	if rule.School < skillSlots {
		level = caster.Skill[rule.School]
	}
	power := spellPowerUnder(w.rules, rule, level, caster.Mind)
	// 6. within the decoded power-scaled range, in whole cells, by the same Chebyshev
	// distance the rest of this package measures a separation with —
	// cell.chebyshevTo (route.go), reused rather than restated.
	if (cell{x: caster.X, y: caster.Y}).chebyshevTo(cell{x: target.X, y: target.Y}) > spellRangeUnder(w.rules, rule, power) {
		return false
	}
	// 6b. THE ROW'S OWN TARGET CONDITION, asked BEFORE the cost so that a
	// release the ordinary apply would refuse spends nothing (spec: a refusal,
	// a release-time cancellation and an effect that cannot apply spend no
	// mana, start no recovery and award no training). It is the same predicate
	// bookSpellRefusal asks at admission, so the two cannot disagree.
	if w.pointEffectRefusal(ci, vi, rule, power) != "" {
		return false
	}
	// 6c. pointEffectRefusal answers "" for every area row and this one answers
	// "" for every point row, so the two are one gate over the whole table with
	// no row asked twice. Without it this arm paid, marked both actors and
	// started recovery for a landing whose result it discarded.
	if w.areaLandingRefusal(rule, uint16(power), aimX, aimY) != "" {
		return false
	}
	// 6a. THE CADENCE FLOOR: a caster that cast inside the last castPeriod
	// ticks is refused, so no book cast can emit a projectile before the
	// previous one's swing has run. It is asked BEFORE the cost, so a throttled
	// cast spends nothing and rolls nothing, and it is asked of the book arm
	// alone — releaseWeaponSpell never reaches this function.
	if caster.CastWait > 0 {
		return false
	}
	// 7. a cost the caster cannot afford refuses the WHOLE cast: no mana
	// spent, nothing rolled — the return below is this arm's only one past
	// this point, so nothing after it can leave a half-applied cast.
	if !w.bookAlreadyPaid(caster.ID, spellID) && !bookAffords(*caster, rule) {
		return false
	}
	// 8. the cost is paid, once.
	if !w.bookAlreadyPaid(caster.ID, spellID) {
		debitBook(caster, rule)
	}
	w.refreshAppliedBook(ci, spellID)
	if victim != caster.ID {
		w.removeAttachedSpell(caster.ID, 15)
	}

	var raisedID EntityID
	if rule.ID == 25 {
		raisedID, _ = w.NextEntityID()
	}
	var victims []CellPoint
	if rule.Area {
		w.landAreaAimed(areaAim{Target: victim, Has: true}, rule, uint16(power), caster.ID, true, caster.X, caster.Y, aimX, aimY, caster.Facing, true, obs)
	} else if rule.ID == 14 && rule.Delivery == 2 && w.bookAlreadyPaid(caster.ID, spellID) {
		// Its fan was prepared at admission (MAGIC-CASTCLOCK-171).
	} else if rule.ID == 14 {
		victims = w.applyPrismatic(ci, vi, rule, power)
	} else if !w.ordinaryEffect(ci, vi, rule, power) {
		return false
	}
	if rule.ID == 25 {
		// Control Spirit consumes its target and creates a new actor, so the
		// indices resolved before apply no longer name the two actors. Rebind
		// both before the common observation/training tail.
		ci = indexOfEntity(w.entities, casterID)
		vi = indexOfEntity(w.entities, raisedID)
		w.markSpellEffect(ci, rule.ID)
		if vi >= 0 {
			w.markSpellEffect(vi, rule.ID)
		}
		if w.entities[ci].Alive() {
			w.entities[ci].CastWait = castRecoveryTicks(w.entities[ci])
		}
		if vi >= 0 {
			obs.recordFrom(w, ci, vi, rule, false, castFromX, castFromY)
		}
		w.awardSkill(ci, int32(rule.School), (int64(rule.ManaCost)+1)/2, -1)
		return true
	}

	w.markSpellEffect(ci, rule.ID)
	if rule.Delivery != 2 {
		w.markSpellEffect(vi, rule.ID)
	}
	// AND THE WAIT IS RELOADED ON EVERY APPLIED BOOK CAST, commanded or
	// unbidden alike, which is what makes the floor a property of the caster
	// rather than of the arm that reached it.
	if w.entities[ci].Alive() {
		w.entities[ci].CastWait = castRecoveryTicks(w.entities[ci])
	}
	// AND THE OBSERVATION BESIDE THE MARK, on the same "applied only" terms:
	// the mark says a unit was touched and the observation says by whom, which
	// is the one thing the mark cannot carry (castevent.go). It is recorded
	// HERE, where both indices and the row are in hand, and it changes no state
	// — obs is nil on every unobserved step.
	restored := int32(0)
	if rule.Restorative && w.entities[vi].HP > healthBefore {
		restored = w.entities[vi].HP - healthBefore
	}
	if !(rule.ID == 14 && rule.Delivery == 2 && w.bookAlreadyPaid(casterID, spellID)) {
		obs.recordFromResult(w, ci, vi, rule, false, castFromX, castFromY, restored)
		obs.recordVictims(victims)
	}

	// MAGIC-TRAIN-018
	if rule.ID != 14 {
		w.awardSkill(ci, int32(rule.School), (int64(rule.ManaCost)+1)/2, -1)
	}
	return true
}

// castBookAt is the point-target counterpart of castSpell. Area effects land
// at the point; Teleport moves its caster there. Both use the same book,
// power-scaled range, cadence and cost as a unit-target cast.
func (w *World) castBookAt(ci int, x, y int32, spellID uint32, obs *castObs) bool {
	caster := &w.entities[ci]
	if !caster.Alive() || !bookCaster(*caster) || !knowsSpell(*caster, spellID) {
		return false
	}
	rule, ok := w.bookSpell(*caster, spellID)
	if !ok || (!rule.Area && rule.ID != 26) || !spellApplicable(rule) {
		return false
	}
	var level int32
	if rule.School < skillSlots {
		level = caster.Skill[rule.School]
	}
	power := spellPowerUnder(w.rules, rule, level, caster.Mind)
	// Point orders use the book's range, not the caster's personal sight.
	// The player view gates the clicked ground before producing this order.
	if _, inside := cellIndexIn(w.bounds, x, y); !inside ||
		(cell{x: caster.X, y: caster.Y}).chebyshevTo(cell{x: x, y: y}) > spellRangeUnder(w.rules, rule, power) ||
		caster.CastWait > 0 || !w.bookAlreadyPaid(caster.ID, spellID) && !bookAffords(*caster, rule) {
		return false
	}
	// The area landing's own refusals, asked BEFORE the cost (spec: a refusal
	// and an effect that cannot apply spend no mana and start no recovery).
	// This arm paid and then returned on landArea's false, so a player casting
	// repeatedly at one cell paid the row every time the cell's six slots were
	// already taken. It is pointEffectRefusal's counterpart for the cell form
	// and answers "" for Teleport, whose own destination test is above.
	if w.areaLandingRefusal(rule, uint16(power), x, y) != "" {
		return false
	}
	if !w.bookAlreadyPaid(caster.ID, spellID) {
		debitBook(caster, rule)
	}
	w.refreshAppliedBook(ci, spellID)
	if rule.ID == 26 {
		obs.recordAt(w, ci, rule, x, y)
		if w.attachFootprint(ci, x, y) {
			caster.X, caster.Y = x, y
			w.invalidateActorMotion(caster.ID, "native teleport supersedes original movement")
			caster.clearStride()
			caster.HasTarget = false
			w.routes[ci] = nil
		}
	} else {
		if !w.landArea(rule, uint16(power), caster.ID, true, caster.X, caster.Y, x, y, obs) {
			return false
		}
		obs.recordAt(w, ci, rule, x, y)
	}
	w.markSpellEffect(ci, rule.ID)
	if w.entities[ci].Alive() {
		w.entities[ci].CastWait = castRecoveryTicks(w.entities[ci])
	}
	// THE CELL FORM PAYS THE SAME ONE CAST AWARD THE UNIT FORM DOES (DIV-238).
	// It used to pay for Teleport alone, which is the one row reaching this arm
	// without landing an area effect; every area row landed here and trained
	// through applyAreaCells or not at all.
	w.awardSkill(ci, int32(rule.School), (int64(rule.ManaCost)+1)/2, -1)
	return true
}

func (w *World) applySpellDamage(vi int, rule SpellRule, power int32) {
	target := &w.entities[vi]
	before := *target
	base, spread := spellDamageUnder(w.rules, rule, power)
	if spread < 0 {
		spread = 0
	}
	amount := base + int64(w.rng.uniform(int32(spread)))
	if rule.School >= 1 && rule.School <= 5 {
		p := target.Protection[rule.School-1]
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		amount = (amount*int64(100-p) + 75) / 100
	}
	if hp := int64(target.HP) - amount; hp < int64(minHP) {
		target.setCurrentHealth(minHP)
	} else {
		target.setCurrentHealth(int32(hp))
	}
	w.reportHealthLoss(before, vi)
	// A blow that fells its victim leaves the same residue any other one
	// does — clearFelled is the one rule for that, called here on its own
	// terms (step.go).
	w.clearFelled(vi)
}

func (w *World) weaponSpellApply(ai, ti int, rule SpellRule, power int32, obs *castObs) bool {
	if rule.ID == 14 || !spellApplicable(rule) {
		return false
	}
	if rule.ID == controlSpiritSpellID {
		return w.queueControlSpirit(ai, ti)
	}
	if rule.Area {
		t := w.entities[ti]
		if w.areaLandingRefusal(rule, uint16(power), t.X, t.Y) != "" {
			return false
		}
		a := w.entities[ai]
		return w.landAreaAimed(areaAim{Target: t.ID, Has: true}, rule, uint16(power), a.ID, true, a.X, a.Y, t.X, t.Y, a.Facing, true, obs)
	}
	if w.pointEffectRefusal(ai, ti, rule, power) != "" {
		return false
	}
	return w.ordinaryEffect(ai, ti, rule, power)
}

func (w *World) releaseWeaponSpell(ai, ti int, rule SpellRule, obs *castObs) {
	a, t := &w.entities[ai], &w.entities[ti]
	// 1. the victim must still accept this weapon's spell row.
	if !spellTargetable(*t, rule) || prismaticBodyPrimary(*t, rule) {
		return
	}
	// 2. and must not be the caster itself.
	if t.ID == a.ID {
		return
	}
	// A staff/weapon spell is still one actor-directed release. It uses the
	// same live perception predicate as a book cast and does not fall back to a
	// physical strike when the spell-bearing action loses sight before release.
	if !w.actorSeesEntity(ai, ti) {
		return
	}
	// 3. within the row's MaxRange, in whole cells — the SAME admission
	// closedOn (combat.go) asks while the actor is still approaching, so
	// "walked close enough" and "close enough to release" cannot come apart
	//.
	if (cell{x: a.X, y: a.Y}).chebyshevTo(cell{x: t.X, y: t.Y}) > int64(rule.MaxRange) {
		return
	}
	// 4. and the row must actually land: weaponSpellApply's own refusals.
	if rule.ID == 14 {
		victims := w.applyPrismaticItem(ai, ti, rule, a.WeaponSpellLevel, true)
		if len(victims) == 0 {
			return
		}
		w.markSpellEffect(ai, rule.ID)
		if rule.Delivery != 2 {
			w.markSpellEffect(ti, rule.ID)
		}
		obs.record(w, ai, ti, rule, true)
		obs.recordVictims(victims)
		return
	}
	if !w.weaponSpellApply(ai, ti, rule, a.WeaponSpellLevel, obs) {
		return
	}

	w.markSpellEffect(ai, rule.ID)
	if rule.Delivery != 2 && rule.ID != controlSpiritSpellID {
		w.markSpellEffect(ti, rule.ID)
	}
	obs.record(w, ai, ti, rule, true)

	// There is deliberately no half-mana award here. Event producers inside the
	// apply have already paid any damage or pseudo-damage training they caused.
}

// weaponRiderApply is MAGIC-ITEM-007's rider arm (R3-B3): after a landed
// blow, a FIGHTER whose weapon carries a known row applies it at the struck
// target — the exact complement of releaseWeaponSpell's caster arm
// (MAGIC-AUTOCAST-020). It is called from resolveBlow's own tail
// (combat.go), once the blow's damage has already reduced the target's
// health, and it asks almost none of releaseWeaponSpell's own admission: the
// strike that reached it already proved reach and sight, and `L05098`
// requires only that the weapon carry a spell. The owner-directed -10 cutoff
// is enforced by resolveBlow before this function is reached (`DIV-442`).
//
// weaponSpellApply's own Area branch needs no second target check: Fire Ball is
// a blast and the caller has already retained or refused the target. A point row
// converges on ordinaryEffect's same -10 boundary.
func (w *World) weaponRiderApply(ai, ti int, positiveLivingHit bool, obs *castObs) {
	a := w.entities[ai]
	rule, ok := weaponRiderSpellFor(w.spells, a)
	if !ok {
		return
	}
	if !positiveLivingHit && rule.ID != 2 {
		return
	}
	if !w.weaponSpellApply(ai, ti, rule, a.WeaponSpellLevel, obs) {
		return
	}
	w.markSpellEffect(ai, rule.ID)
	if rule.Delivery != 2 && rule.ID != controlSpiritSpellID {
		w.markSpellEffect(ti, rule.ID)
	}
	obs.record(w, ai, ti, rule, true)
}

// applySpellHealing is applySpellDamage's counterpart for a restorative row:
// the same roll, the same one draw, and the same shared position at the tail
// of a cast — so the two arms cannot come to compute a different number
// for the same row and the same power.
//
// IT DRAWS EXACTLY ONCE, on applySpellDamage's own convention: the spread
// alone, since the base is deterministic at a fixed power and rule. That keeps
// a replay's draw count the same whichever arm a cast took.
//
// THE RESULT IS CLAMPED AT THE TARGET'S MAXIMUM, which is the one thing the
// damage arm does not do. A damage roll saturates at minHP because health is
// carried whole below zero and a wrap would resurrect a corpse; a heal has a
// ceiling instead, because MaxHP is the health the entity was built with.
// Useful-outcome admission above prevents a full-health target from reaching
// this tail, paying mana, training or producing a visual event.
//
// AN ENTITY WITH NO HEALTH SYSTEM IS NOT HEALED. MaxHP not positive is that
// state (Entity.MaxHP's own doc, world.go), and raising such an entity's HP
// toward a maximum it does not have would invent a health system for it. The
// draw still happens, so the two arms keep the same generator cost.
func (w *World) applySpellHealing(vi int, rule SpellRule, power int32) {
	target := &w.entities[vi]
	before := target.HP
	base, spread := spellDamageUnder(w.rules, rule, power)
	if spread < 0 {
		spread = 0
	}
	amount := base + int64(w.rng.uniform(int32(spread)))
	if target.MaxHP <= 0 {
		return
	}
	if hp := int64(target.HP) + amount; hp > int64(target.MaxHP) {
		target.setCurrentHealth(target.MaxHP)
	} else {
		target.setCurrentHealth(int32(hp))
	}
	w.restoreAfterHealthGain(vi, before)
}

// spellFXLife is how many ticks a spell effect mark stands. It is AUTHORED:
// MAGIC-PIC-027 fixes that a direct cast gives the caster a cast action
// rather than a map object, and says nothing about how long that action
// runs. Four ticks is a quarter of the sixteen-tick cycle the mission script
// runs on, which is long enough to read at the pace the game is played at
// and short enough that two casts in a row do not merge into one mark.
const spellFXLife = 4

// castPeriod is the FLOOR ON A BOOK CAST'S WIND-UP. The original applies a
// spell when the actor's charge counter reaches zero, then observes the
// actor's separate recovery counter (MAGIC-CASTTICK-030). Eight is also the
// owner's minimum visible swing, so a shorter or absent installed charge is
// widened to eight rather than making a one-tick projectile feed.
//
// It agrees with the decode rather than contradicting it. `MAGIC-CASTTICK-030`
// fixes the original's own wind-up `actor+0x134` at a default of 8 ticks, and
// `MAGIC-CASTANIM-029` measures the shortest expanded Attack timeline over the
// 34 shipped classes at 8 frames. The floor and both of those are one number.
//
// A WEAPON-BORNE RELEASE IS NOT HELD TO IT. Its cadence is the attack cycle's
// own wind-up, which always has a swing behind it by construction, and refusing
// a release would leave exactly the swing-without-a-projectile his rule forbids.
const castPeriod = 8

type bookPhase uint8

const (
	bookPending bookPhase = iota
	bookCharging
	bookRelaxing
	bookBoundaryOne
	bookBoundaryTwo
	// bookApproach is a creature's retained cast order that is out of range:
	// the caster walks toward its victim and the order stays armed until it
	// is in range or another order replaces it.
	bookApproach
)

// bookCast is one spellbook lifecycle. A retained player or group order re-enters
// after the two boundaries; an unbidden autocast is one-shot and returns target
// selection to its producer after recovery. Pending records survive insufficient
// mana only for retained orders.
type bookCast struct {
	Caster    EntityID
	Target    EntityID
	Spell     uint16
	X, Y      int32
	Remaining uint8
	Phase     bookPhase
	Progress  uint8
	Complete  bool
	AtCell    bool
	Retained  bool
	Paid      bool
}

func castWindupTicks(e Entity) uint8 {
	n := e.AttackCharge
	if n < castPeriod {
		n = castPeriod
	}
	if n > 255 {
		n = 255
	}
	return uint8(n)
}

func castRecoveryTicks(e Entity) uint8 {
	n := e.AttackRelax
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	return uint8(n)
}

func (w *World) bookRecoveryTicks(ci int, rule SpellRule) uint8 {
	n := relaxTicks(w.entities[ci]) + int64(w.rng.uniform(relaxJitter)) + w.humanoidPenalty(ci) + int64(rule.Complication)
	if n > 255 {
		n = 255
	}
	return uint8(n)
}

func (w *World) bookCastIndex(caster EntityID) (int, bool) {
	i := sort.Search(len(w.bookCasts), func(i int) bool { return w.bookCasts[i].Caster >= caster })
	return i, i < len(w.bookCasts) && w.bookCasts[i].Caster == caster
}

func (w *World) bookCastInFlight(caster EntityID) bool {
	i, ok := w.bookCastIndex(caster)
	return ok && w.bookCasts[i].Phase == bookCharging
}

// actorActionBusy is the CAST admission guard: it answers "may a new spell
// begin on this actor now", and nothing else. A spell may not begin while a
// physical or weapon attack cycle is loaded, and may not begin while a book
// cast owns the actor. That is the non-overlap the contract asks for between a
// armed autocast, idle Heal, AI cast, weapon-spell attack and physical attack.
// Manual book commands validate their target, then cancel the old action
// before reaching this ordinary admission path (DIV-1015).
//
// IT IS NOT A MOVEMENT GUARD AND IT IS NOT AN ATTACK-ORDER GUARD. A movement
// order ends whatever fight the unit was in and is never refused; an attack
// order competes with another attack order under orderAttack's own rule,
// which resets the cycle on a different victim. Applying this predicate to
// either of those made a fighting or approaching unit ignore a right-click,
// which neither contract.md nor spec.md sanctions.
//
// The approach half is deliberately absent too. HasAttackTarget with a
// destination is a walk, not an interval: non-overlap between the attack cycle
// and a cast is already total, because the attack advance and the mover both
// stand down for an actor that carries a cast (step.go), and this predicate
// refuses a cast for an actor whose cycle is loaded.
func (w *World) actorActionBusy(i int) bool {
	if w.motionActive(w.entities[i].ID) {
		return true
	}
	if w.stoneCursed(i) {
		return true
	}
	e := w.entities[i]
	// HasAttackTarget is the durable instruction, not by itself an active
	// interval. At Ready/0 the preceding strike has recovered and another
	// attack or spell may win fresh admission before the next wind-up loads.
	if e.AttackPhase != AttackReady || e.AttackCountdown != 0 {
		return true
	}
	return w.actorCastBusy(i)
}

// retainedCastBusy is actorActionBusy with the caller's own retained book
// record excluded. It keeps physical/weapon actions and an older direct-cast
// recovery mutually exclusive with retry admission.
func (w *World) retainedCastBusy(i int) bool {
	if w.motionActive(w.entities[i].ID) {
		return true
	}
	if w.stoneCursed(i) {
		return true
	}
	e := w.entities[i]
	return e.AttackPhase != AttackReady || e.AttackCountdown != 0 || e.CastWait != 0
}

func (w *World) queueBookCast(c bookCast) bool {
	i, exists := w.bookCastIndex(c.Caster)
	if exists {
		return false
	}
	w.bookCasts = append(w.bookCasts, bookCast{})
	copy(w.bookCasts[i+1:], w.bookCasts[i:])
	w.bookCasts[i] = c
	return true
}

// beginBookSpell performs every refusal that can be known at admission without
// rolling or mutating the world, then records the wind-up. Installed rules
// pay at admission; pre-form95 native rules retain payment at release.
func (w *World) beginBookSpell(ci int, victim EntityID, spellID uint32) bool {
	return w.beginBookSpellMode(ci, victim, spellID, true)
}

func (w *World) beginBookSpellOnce(ci int, victim EntityID, spellID uint32) bool {
	return w.beginBookSpellMode(ci, victim, spellID, false)
}

func (w *World) beginBookSpellMode(ci int, victim EntityID, spellID uint32, retained bool) bool {
	caster := w.entities[ci]
	if w.bookSpellRefusal(ci, victim, spellID, false, retained) != "" {
		return false
	}
	rule, _ := w.bookSpell(caster, spellID)
	target := w.entities[indexOfEntity(w.entities, victim)]
	c := bookCast{Caster: caster.ID, Target: victim, Spell: uint16(spellID), X: target.X, Y: target.Y,
		Retained: retained}
	if bookAffords(caster, rule) {
		c.Phase, c.Remaining = bookCharging, castWindupTicks(caster)
	}
	if !w.queueBookCast(c) {
		return false
	}
	w.entities[ci].AdmittedBookSpell = uint16(spellID)
	// Facing is part of admission: a refusal above writes nothing, and target
	// movement during wind-up cannot restart or retarget this action.
	w.turnTowardActor(ci, target.X-caster.X, target.Y-caster.Y)
	if c.Phase == bookCharging {
		w.admitBookPayment(ci, rule)
		w.startSpellAction(ci)
	}
	return true
}

// RestoreOneShotPlayerCasts normalizes retained manual orders written before
// explicit commands became one-shot. A player cast still charging may release
// once. A completed or refused retained order is removed; an unfinished
// recovery becomes the ordinary one-shot CastWait. Non-player retained AI
// orders and already-one-shot records are unchanged.
func (w *World) RestoreOneShotPlayerCasts() int {
	if w == nil {
		return 0
	}
	out := w.bookCasts[:0]
	repaired := 0
	for _, cast := range w.bookCasts {
		ci := indexOfEntity(w.entities, cast.Caster)
		if !cast.Retained || ci < 0 || w.entities[ci].Owner != SelfSlot {
			out = append(out, cast)
			continue
		}
		repaired++
		if cast.Phase == bookCharging {
			cast.Retained = false
			out = append(out, cast)
			continue
		}
		if cast.Phase == bookRelaxing && w.entities[ci].CastWait < cast.Remaining {
			w.entities[ci].CastWait = cast.Remaining
		}
	}
	w.bookCasts = out
	return repaired
}

// BookSpellRefusal is the read-only admission explanation for one explicit
// player cast. No roll, queue or field is changed.
func (w *World) BookSpellRefusal(casterID, victim EntityID, spellID uint32) string {
	ci := indexOfEntity(w.entities, casterID)
	if ci < 0 {
		return "caster absent"
	}
	if w.entities[ci].OffMap {
		return "caster off map"
	}
	return w.bookSpellAdmission(ci, victim, spellID, false, false, false)
}

// BookSpellCellRefusal is BookSpellRefusal for the CELL form: why a cast of
// spellID at (x, y) would be refused, or "" when beginBookSpellAt will accept
// it. No roll, queue or field is changed.
//
// IT IS A SECOND ENTRY POINT AND NOT A WIDENED FIRST ONE, because the two forms
// take different arguments and answer about different admissions: the unit form
// asks about a victim, the cell form about a cell with no victim at all. A
// single signature would have to accept a meaningless victim for half its
// callers.
func (w *World) BookSpellCellRefusal(casterID EntityID, x, y int32, spellID uint32) string {
	ci := indexOfEntity(w.entities, casterID)
	if ci < 0 {
		return "caster absent"
	}
	if w.entities[ci].OffMap {
		return "caster off map"
	}
	return w.bookSpellCellAdmission(ci, x, y, spellID, false, false, false)
}

// bookSpellCellRefusal states beginBookSpellAt's refusals in the order that
// function asks them, so the two cannot report different verdicts about one
// cast. Both read areaLandingRefusal for the landing's own half.
func (w *World) bookSpellCellRefusal(ci int, x, y int32, spellID uint32, retained, allowInsufficient bool) string {
	return w.bookSpellCellAdmission(ci, x, y, spellID, retained, allowInsufficient, true)
}

func (w *World) bookSpellCellAdmission(ci int, x, y int32, spellID uint32, retained, allowInsufficient, checkAction bool) string {
	return w.bookSpellCellAdmissionRange(ci, x, y, spellID, retained, allowInsufficient, checkAction, false)
}

// bookSpellCellAdmissionRange is bookSpellCellAdmission with the range test
// optionally skipped, for the caller that asks whether a cast is refused for
// range alone.
func (w *World) bookSpellCellAdmissionRange(ci int, x, y int32, spellID uint32, retained, allowInsufficient, checkAction, skipRange bool) string {
	caster := w.entities[ci]
	if !caster.Alive() {
		return "caster not alive"
	}
	if !bookCaster(caster) {
		return "caster has no mana pool"
	}
	if !knowsSpell(caster, spellID) {
		return "spell not known"
	}
	rule, ok := w.bookSpell(caster, spellID)
	if !ok {
		return "spell row absent"
	}
	if !rule.Area && rule.ID != 26 {
		return "spell does not target a cell"
	}
	if !spellApplicable(rule) {
		return "spell has no applicable arm"
	}
	if _, inside := cellIndexIn(w.bounds, x, y); !inside {
		return "cell outside the map"
	}
	var level int32
	if rule.School < skillSlots {
		level = caster.Skill[rule.School]
	}
	power := spellPowerUnder(w.rules, rule, level, caster.Mind)
	if !skipRange && (cell{x: caster.X, y: caster.Y}).chebyshevTo(cell{x: x, y: y}) > spellRangeUnder(w.rules, rule, power) {
		return refusalCellOutOfRange
	}
	if r := w.areaLandingRefusal(rule, uint16(power), x, y); r != "" {
		return r
	}
	if w.stoneCursed(ci) || checkAction && (!retained && w.actorActionBusy(ci) || retained && w.retainedCastBusy(ci)) {
		return "caster action busy"
	}
	if !allowInsufficient && !bookAffords(caster, rule) {
		return "insufficient mana"
	}
	return ""
}

// The two admission refusals a creature's drawn cast turns into a walk.
const (
	refusalTargetOutOfRange = "target out of range"
	refusalCellOutOfRange   = "cell out of range"
)

func (w *World) bookSpellRefusal(ci int, victim EntityID, spellID uint32, retained, allowInsufficient bool) string {
	return w.bookSpellAdmission(ci, victim, spellID, retained, allowInsufficient, true)
}

func (w *World) bookSpellAdmission(ci int, victim EntityID, spellID uint32, retained, allowInsufficient, checkAction bool) string {
	return w.bookSpellAdmissionRange(ci, victim, spellID, retained, allowInsufficient, checkAction, false)
}

// bookSpellAdmissionRange is bookSpellAdmission with the range test optionally
// skipped, for the caller that asks whether a cast is refused for range alone.
func (w *World) bookSpellAdmissionRange(ci int, victim EntityID, spellID uint32, retained, allowInsufficient, checkAction, skipRange bool) string {
	caster := w.entities[ci]
	if !caster.Alive() {
		return "caster not alive"
	}
	if !bookCaster(caster) {
		return "caster has no mana pool"
	}
	if !knowsSpell(caster, spellID) {
		return "spell not known"
	}
	rule, ok := w.bookSpell(caster, spellID)
	if !ok {
		return "spell row absent"
	}
	if !spellApplicable(rule) {
		return "spell has no applicable arm"
	}
	if !rule.Area && !rule.TargetsUnit && rule.ID != 15 && rule.ID != 18 && rule.ID != 20 &&
		rule.ID != 23 && rule.ID != 24 && rule.ID != 25 && rule.ID != 26 && rule.ID != 27 && rule.ID != 28 {
		return "spell does not target a unit"
	}
	vi := indexOfEntity(w.entities, victim)
	if vi < 0 {
		return "target absent"
	}
	target := w.entities[vi]
	if !spellTargetable(target, rule) {
		return "target state refuses spell"
	}
	if prismaticBodyPrimary(target, rule) && !w.bookAlreadyPaid(caster.ID, spellID) {
		return "prismatic primary is a body"
	}
	if victim == caster.ID && rule.Damaging && !rule.SelfCast {
		return "damaging self target"
	}
	if rule.Restorative && target.MaxHP <= 0 {
		return "heal target has no health system"
	}
	if !w.actorSeesEntity(ci, vi) {
		return "target not currently visible"
	}
	var level int32
	if rule.School < skillSlots {
		level = caster.Skill[rule.School]
	}
	if !skipRange && (cell{x: caster.X, y: caster.Y}).chebyshevTo(cell{x: target.X, y: target.Y}) >
		spellRangeUnder(w.rules, rule, spellPowerUnder(w.rules, rule, level, caster.Mind)) {
		return refusalTargetOutOfRange
	}
	if r := w.pointEffectRefusal(ci, vi, rule, spellPowerUnder(w.rules, rule, level, caster.Mind)); r != "" {
		return r
	}
	// An area row delivered at a unit lands at that unit's own cell, so the
	// instrument must ask the landing's refusals there too. It reported a cast
	// at a cell already holding six records as admissible.
	if r := w.areaLandingRefusal(rule, uint16(spellPowerUnder(w.rules, rule, level, caster.Mind)), target.X, target.Y); r != "" {
		return r
	}
	if w.stoneCursed(ci) || checkAction && (!retained && w.actorActionBusy(ci) || retained && w.retainedCastBusy(ci)) {
		return "caster action busy"
	}
	if !allowInsufficient && !bookAffords(caster, rule) {
		return "insufficient mana"
	}
	return ""
}

func (w *World) beginBookSpellAt(ci int, x, y int32, spellID uint32) bool {
	return w.beginBookSpellAtMode(ci, x, y, spellID, false)
}

func (w *World) beginBookSpellAtMode(ci int, x, y int32, spellID uint32, retained bool) bool {
	// The refusals are bookSpellCellRefusal's, asked once and stated there
	// rather than here, on beginBookSpell's own shape for the unit form. That
	// is what makes the read-only instrument report what this function will do:
	// a cast the instrument calls admissible and this one drops, or the other
	// way round, is the disagreement the two exist to prevent.
	if w.bookSpellCellRefusal(ci, x, y, spellID, false, retained) != "" {
		return false
	}
	caster := w.entities[ci]
	rule, _ := w.bookSpell(caster, spellID)
	c := bookCast{Caster: caster.ID, Spell: uint16(spellID), X: x, Y: y, AtCell: true, Retained: retained}
	if bookAffords(caster, rule) {
		c.Phase, c.Remaining = bookCharging, castWindupTicks(caster)
	}
	if !w.queueBookCast(c) {
		return false
	}
	w.entities[ci].AdmittedBookSpell = uint16(spellID)
	w.turnToward(ci, x-caster.X, y-caster.Y)
	if c.Phase == bookCharging {
		w.admitBookPayment(ci, rule)
		w.startSpellAction(ci)
	}
	return true
}

// stepBookCasts advances each admitted cast once. The returned set names casts
// that released this tick so the later autocast sweep does not consume one
// recovery tick immediately or begin another cast when recovery is zero.
func (w *World) stepBookCasts(obs *castObs, interrupted map[EntityID]bool) map[EntityID]bool {
	var released map[EntityID]bool
	for i := 0; i < len(w.bookCasts); {
		c := &w.bookCasts[i]
		if interrupted[c.Caster] {
			i++
			continue
		}
		ci := indexOfEntity(w.entities, c.Caster)
		if ci < 0 || !w.entities[ci].Alive() {
			w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
			continue
		}
		if w.stoneCursed(ci) || w.motionActive(w.entities[ci].ID) {
			i++
			continue
		}
		switch c.Phase {
		case bookBoundaryOne:
			c.Phase = bookBoundaryTwo
			i++
			continue
		case bookBoundaryTwo:
			c.Phase = bookPending
			fallthrough
		case bookPending:
			if c.Complete && c.Progress == 3 {
				c.Progress = 0
				i++
				continue
			}
			refusal := w.retainedBookRefusal(ci, *c)
			if refusal == "insufficient mana" {
				if !c.Retained {
					w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
					continue
				}
				if c.Complete {
					c.Progress++
				}
				i++
				continue
			}
			if victim, ok := w.approachVictim(ci, *c); ok && isRangeRefusal(refusal) && w.armCreatureApproach(ci, c, victim) {
				i++
				continue
			}
			if refusal != "" {
				w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
				continue
			}
			c.Phase, c.Remaining, c.Progress, c.Complete = bookCharging,
				castWindupTicks(w.entities[ci]), 0, false
			rule, _ := w.bookSpell(w.entities[ci], uint32(c.Spell))
			w.admitBookPayment(ci, rule)
			w.startSpellAction(ci)
			i++
			continue
		case bookRelaxing:
			if c.Remaining > 0 {
				c.Remaining--
			}
			if c.Remaining == 0 {
				c.Phase = bookBoundaryOne
			}
			i++
			continue
		case bookApproach:
			refusal := w.retainedBookRefusal(ci, *c)
			switch {
			case refusal == "":
				w.startApproachedCast(ci, c)
			case !isRangeRefusal(refusal) || !w.approachHeld(ci, *c):
				w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
				continue
			}
			i++
			continue
		case bookCharging:
		default:
			w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
			continue
		}
		if w.entities[ci].Turning() && c.Target != c.Caster {
			i++
			continue
		}
		if c.Remaining > 0 {
			c.Remaining--
		}
		if c.Remaining > 0 {
			i++
			continue
		}
		state := *c
		refusal := w.retainedBookRefusal(ci, state)
		if refusal == "insufficient mana" {
			if !c.Retained {
				w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
				continue
			}
			c.Phase, c.Progress, c.Complete = bookPending, 0, false
			i++
			continue
		}
		if victim, ok := w.approachVictim(ci, *c); ok && isRangeRefusal(refusal) && w.armCreatureApproach(ci, c, victim) {
			i++
			continue
		}
		if refusal != "" {
			w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
			continue
		}
		applied := false
		if state.AtCell {
			applied = w.castBookAt(ci, state.X, state.Y, uint32(state.Spell), obs)
		} else {
			applied = w.castSpell(ci, state.Target, uint32(state.Spell), state.X, state.Y, obs)
		}
		if !applied {
			w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
			continue
		}
		rule, _ := w.findSpell(uint32(state.Spell))
		recovery := w.bookRecoveryTicks(ci, rule)
		// Application can fell any set of book casters in an area. clearFelled
		// removes their records at the damage site, so the releasing record can
		// move to another index or be removed. Rebind it by caster id before any
		// later mutation. The recovery draw above remains part of every applied
		// book action.
		castIndex, castExists := w.bookCastIndex(state.Caster)
		if !w.entities[ci].Alive() {
			w.clearFelled(ci)
			i, _ = w.bookCastIndex(state.Caster)
			continue
		}
		if !castExists {
			i = castIndex
			continue
		}
		if !state.Retained || w.entities[ci].HasTarget {
			w.entities[ci].CastWait = recovery
			w.bookCasts = append(w.bookCasts[:castIndex], w.bookCasts[castIndex+1:]...)
			if released == nil {
				released = make(map[EntityID]bool)
			}
			released[state.Caster] = true
			i = castIndex
			continue
		}
		w.entities[ci].CastWait = 0
		c = &w.bookCasts[castIndex]
		c.Paid = false
		c.Remaining, c.Progress, c.Complete = recovery, 0, true
		if c.Remaining == 0 {
			c.Phase = bookBoundaryOne
		} else {
			c.Phase = bookRelaxing
		}
		if released == nil {
			released = make(map[EntityID]bool)
		}
		released[state.Caster] = true
		i = castIndex + 1
	}
	return released
}

func (w *World) retainedBookRefusal(ci int, c bookCast) string {
	if c.AtCell {
		return w.bookSpellCellRefusal(ci, c.X, c.Y, uint32(c.Spell), true, c.Paid)
	}
	return w.bookSpellRefusal(ci, c.Target, uint32(c.Spell), true, c.Paid)
}

// markSpellEffect puts the spell effect mark on one entity: the full life,
// and the row's own id so the front end can resolve what drew it. A second
// cast on an entity already marked REPLACES the mark rather than extending
// it, so the mark always names the most recent spell to touch it.
//
// THE ID IS NARROWED TO A BYTE. The id space is 1..28 (MAGIC-SPELL-001), and a
// table row whose id does not fit a byte marks with 0 — "no spell" — rather
// than with some other row's number, which is what a truncation would have
// done.
func (w *World) markSpellEffect(i int, id uint16) {
	e := &w.entities[i]
	e.SpellFX = spellFXLife
	if id > 0xff {
		e.SpellFXSpell = 0
		return
	}
	e.SpellFXSpell = uint8(id)
}

// decaySpellEffects is the mark's whole life past its birth: one tick off
// every live mark, and the spell id cleared with the last of them.
//
// CLEARING THE ID TOO IS WHAT MAKES AC-5 TRUE. A world whose marks have run
// out must be byte-identical to one that never carried any, and a residual id
// beside a zero count would be a byte that remembered a cast nobody can see.
func (w *World) decaySpellEffects() {
	for i := range w.entities {
		e := &w.entities[i]
		if e.SpellFX == 0 {
			e.SpellFXSpell = 0
			continue
		}
		e.SpellFX--
		if e.SpellFX == 0 {
			e.SpellFXSpell = 0
		}
	}
}

func (w *World) stepAutoCasts(released map[EntityID]bool) {
	activity := w.rom2ActivityMask()
	for i := range w.entities {
		if !activity.actorActive(w.entities[i]) {
			continue
		}
		// AN ARMED ROW AUTOMATIC SELECTION MAY NOT RUN IS CLEARED WHERE IT IS
		// FOUND. autoCastable alone would leave the player looking at the
		// dashed border on a cell that never fires — the front end reads
		// Entity.AutoSpell for that border and derives nothing — and a stored
		// setting also reaches this build from an older save and from a
		// script. Clearing is the one act that makes what the player sees and
		// what the simulation will do agree.
		if id := w.entities[i].AutoSpell; id != 0 {
			if rule, ok := w.findSpell(uint32(id)); ok && !autoCastable(rule) {
				w.entities[i].AutoSpell = 0
			}
		}
		if released[w.entities[i].ID] {
			continue
		}
		if w.entities[i].CastWait > 0 {
			continue
		}
		if w.entities[i].HasPendingAttackTarget {
			continue
		}
		w.autoCast(i)
	}
}

// autoCastable reports whether AUTOMATIC selection may pick rule at all. A
// player's own KindCast command reaches castSpell without asking.
//
// TELEPORT (26) IS EXCLUDED (owner). It has no sensible automatic behaviour:
// the shipped row carries `Spell Defensive` and is not restorative, and
// autoCastTarget answers such a row with the CASTER'S OWN id — so an
// unbidden Teleport aims the caster at the cell it already stands on.
// terrainOpen admits that cell, the arm writes the same X and Y back, clears
// the caster's own walk order and route, and the row's 60 mana leaves the
// pool for a move of zero. The route-following autocast that would have
// given the row a use unbidden is deferred by the same ruling and is not
// built here.
func autoCastable(rule SpellRule) bool { return rule.ID != 26 }

func (w *World) ageCastRecovery(released map[EntityID]bool) {
	for i := range w.entities {
		if !w.entities[i].Alive() || released[w.entities[i].ID] || w.entities[i].CastWait == 0 {
			continue
		}
		w.entities[i].CastWait--
	}
}

// autoHealReserve is the share of its own mana pool an entity keeps back
// from an UNBIDDEN, UNARMED heal: a quarter, so a caster that heals the
// party between fights arrives at the next one with at least a quarter of
// its pool. It is AUTHORED — the original has no book-spell autocast to
// take a reserve from — and it is stated as a share rather than as a count
// so it scales with the caster rather than with the shipped Heal row's cost.
//
// IT APPLIES TO THE UNARMED HEAL ALONE. The reserve is measured BEFORE the
// cast: an attempt that would leave the caster under it is not made.
const autoHealReserveNum, autoHealReserveDen = 1, 4

// inCombat is the owner-directed active-engagement predicate for idle healing.
// Mere proximity is not participation: the caster is busy when it holds an
// attack target, is winding up or recovering from a spell, or a living hostile
// is actively attacking it. A movement/order target is handled separately by
// underCommand, so an idle-heal decision cannot consume that order either.
func (w *World) inCombat(ci int) bool {
	e := w.entities[ci]
	if e.HasAttackTarget || e.CastWait != 0 {
		return true
	}
	if _, pending := w.bookCastIndex(e.ID); pending {
		return true
	}
	for i := range w.entities {
		if p := &w.entities[i]; !p.HasAttackTarget || p.AttackTargetKind != AttackTargetUnit || p.AttackTarget != e.ID {
			continue
		}
		t := w.entities[i]
		if t.ID == e.ID || !t.Alive() || !w.hostileTo(e, t) {
			continue
		}
		return true
	}
	return false
}

// knownRestorative is the healing row ci KNOWS, lowest id first: the world's
// own table, filtered by the caster's own KnownSpells mask and by the row's
// own Restorative flag.
//
// NO SPELL ID IS COMPARED TO A LITERAL. Which row heals is the table's own
// flag, set by the table loader from the id it already names there, so this
// function knows nothing about which number Heal is. A caster knowing two
// restorative rows takes the LOWEST id, which is a total order over the
// shipped table and over any table a scenario can author.
func (w *World) knownRestorative(ci int) (SpellRule, bool) {
	e := w.entities[ci]
	var best SpellRule
	found := false
	for _, rule := range w.spells {
		if !rule.Restorative || !knowsSpell(e, uint32(rule.ID)) {
			continue
		}
		rule, _ = BookRuleFor(e, rule)
		if !found || rule.ID < best.ID {
			best, found = rule, true
		}
	}
	return best, found
}

func (w *World) autoCastOrder(ci int) []SpellRule {
	// Retreat owns the actor until replaced. A new automatic cast is not
	// pre-existing action progress and must not starve the next away decision.
	// Keep AutoSpell armed; existing wind-up/recovery advances independently.
	if w.entities[ci].ActorState == actorStateRetreat {
		return nil
	}
	// The sweep now visits every entity rather than only the armed ones, so the
	// cheapest statement of "this one can cast nothing unbidden" stands first:
	// nothing armed and an empty book leaves both arms below with nothing to
	// find, and it is the state every entity in every world built before this
	// story is in.
	if w.entities[ci].AutoSpell == 0 && w.entities[ci].KnownSpells == 0 {
		return nil
	}
	armed, hasArmed := SpellRule{}, false
	if id := w.entities[ci].AutoSpell; id != 0 {
		armed, hasArmed = w.bookSpell(w.entities[ci], uint32(id))
		// A row automatic selection may not pick is not armed as far as this
		// order is concerned (autoCastable, stepAutoCasts above).
		if hasArmed && !autoCastable(armed) {
			hasArmed = false
		}
	}
	if hasArmed && armed.Restorative {
		return []SpellRule{armed}
	}
	var order []SpellRule
	if heal, ok := w.unarmedHeal(ci); ok && !w.inCombat(ci) && !underCommand(w.entities[ci]) {
		order = append(order, heal)
	}
	if hasArmed {
		order = append(order, armed)
	}
	return order
}

// affordsAutoHeal is the reserve, asked of the UNARMED out-of-battle heal
// alone: the cast must leave the caster at or above autoHealReserve of its
// own maximum mana. A caster with no mana maximum keeps no reserve, because
// a share of nothing is nothing and the cost test in castSpell already
// refuses it.
func (w *World) affordsAutoHeal(ci int, rule SpellRule) bool {
	e := w.entities[ci]
	if resolved, ok := BookRuleFor(e, rule); ok {
		rule = resolved
	}
	reserve := int64(e.MaxMana) * autoHealReserveNum / autoHealReserveDen
	return int64(e.Mana)-int64(rule.ManaCost) >= reserve
}

// autoCast is one unbidden attempt. It chooses a target and then reaches
// castSpell — THE SAME ROUTINE A KindCast COMMAND REACHES — so an
// autocast can do nothing a player could not have ordered by hand, and the
// two cannot come to disagree about the refusals, the cost, the roll, the
// mark or the award.
func (w *World) autoCast(ci int) {
	if !isMage(w.entities[ci]) {
		return
	}
	if _, using := w.structureUseIndex(w.entities[ci].ID); using {
		return
	}
	for _, rule := range w.autoCastOrder(ci) {
		target, found := w.autoCastTarget(ci, rule)
		if !found {
			continue
		}
		if w.beginBookSpellOnce(ci, target, uint32(rule.ID)) {
			return
		}
	}
}

// autoCastRefreshNum and autoCastRefreshDen are the owner's own re-cast
// threshold: a standing buff is worth replacing once a tenth or less of a
// fresh cast's duration is left on it. It is AUTHORED — no claim states
// what the original tests before an unbidden buff — and it is stated as a
// share of the duration THIS cast would grant, so a caster whose skill grew
// measures against what it can do now rather than against a constant.
const autoCastRefreshNum, autoCastRefreshDen = 1, 10

// autoCastImproves reports whether an unbidden cast of rule at the actor in
// slot ti would improve that actor's state (owner).
//
// THE DEFECT IT WAS WRITTEN FOR: autocast re-cast a standing buff on every
// tick the recovery allowed, because nothing asked whether the effect was
// already there. Each re-cast removed the record and attached an identical one,
// and each paid training experience — so a mage left standing next to nothing
// levelled its school for free, and the mana went with it.
//
// THE THREE CONDITIONS ARE THE OWNER'S, IN HIS ORDER: the actor does not carry
// the effect; the effect is within a tenth of expiring; or the cast would land
// MORE than the standing record did, which is what a skill increase between the
// two casts buys.
//
// THE THIRD IS MEASURED AS WHAT WOULD LAND, NOT AS THE ROW'S NOMINAL
// MAGNITUDE. attachEffect removes the standing record before it attaches the
// new one, so the state the new cast meets is the state with that record's own
// delta reversed — and a protection already at the 100 clamp lands nothing from
// a second 30, which a nominal comparison would read as an improvement and
// re-cast for ever.
//
// A CONTINUOUS EFFECT IS NEVER AN IMPROVEMENT. Its stored magnitude is
// re-applied on every eighth tick rather than held as a reversible delta, so
// there is no landed amount to compare; only its remaining time can make a
// re-cast useful.
//
// AN UNSTORED EFFECT IS NEVER A CANDIDATE. A row whose mode carries none of the
// three timed bits is applied and forgotten (MAGIC-ATTACH-016), so nothing can
// see that it is already in force and an unbidden cast of it would repeat
// without bound. No shipped row is in that state.
//
// IT WRITES NOTHING. pointEffect is read-only and effectLanding takes the actor
// by value.
func (w *World) autoCastImproves(ti int, rule SpellRule, power int32) bool {
	kind, mag, duration, mode := w.pointEffect(ti, rule, power)
	if kind == EffectNone || duration == 0 || mode&effectTimedModes == 0 {
		return false
	}
	i, standing := effectIndex(w.attached, w.entities[ti].ID, rule.ID)
	if !standing {
		return true
	}
	e := w.attached[i]
	if int64(e.Remaining)*autoCastRefreshDen <= int64(duration)*autoCastRefreshNum {
		return true
	}
	if mode&EffectContinuous != 0 {
		return false
	}
	reversed, _, moves := effectLanding(w.entities[ti], kind, -e.Magnitude)
	if !moves {
		// A kind with no state field of its own — Invisibility's is a
		// parameter other code reads, not a delta. Presence and remaining time
		// are the whole test for it.
		return false
	}
	_, landed, _ := effectLanding(reversed, kind, mag)
	return landed > e.Magnitude
}

func (w *World) autoCastTarget(ci int, rule SpellRule) (EntityID, bool) {
	caster := w.entities[ci]
	if caster.Book.State != BookLegacy {
		var ok bool
		rule, ok = BookRuleFor(caster, rule)
		if !ok {
			return 0, false
		}
	}
	var level int32
	if rule.School < skillSlots {
		level = caster.Skill[rule.School]
	}
	power := spellPowerUnder(w.rules, rule, level, caster.Mind)
	reach := spellRangeUnder(w.rules, rule, power)
	var visible []byte
	var best EntityID
	var bestTier int
	var bestDistance int64
	var bestDeficit int64
	found := false
	for i := range w.entities {
		if (cell{x: caster.X, y: caster.Y}).chebyshevTo(cell{x: w.entities[i].X, y: w.entities[i].Y}) > reach {
			continue
		}
		t := w.entities[i]
		if !spellTargetable(t, rule) || prismaticBodyPrimary(t, rule) {
			continue
		}
		distance := (cell{x: caster.X, y: caster.Y}).chebyshevTo(cell{x: t.X, y: t.Y})
		var key int64
		tier := 0
		if rule.ID == 25 {
			if t.Alive() || t.Decay != DecayBones {
				continue
			}
			key = distance
		} else if rule.Defensive && !rule.Restorative && !rule.Damaging {
			// A LASTING BUFF GOES TO THE FRIENDLY ACTOR IT WOULD ACTUALLY IMPROVE
			// (owner). Own team first, then a locked ally; a neutral is not a
			// candidate, because a buff is a standing mana drain rather than the
			// bounded one an idle heal takes, and the owner asked for allies.
			//
			// THE ROW'S OWN RANGE DECIDES WHETHER THIS IS A SELF-CAST AT ALL.
			// Shield ships MaxRange 0, so spellRange answers 0 and the reach
			// test above leaves the caster as the only candidate; the four
			// protections ship 5 and reach the party. Nothing here names a
			// spell id.
			if t.Owner != caster.Owner && !w.relations.Locked(caster.Owner, t.Owner) {
				continue
			}
			if !w.autoCastImproves(i, rule, power) {
				continue
			}
			if t.Owner != caster.Owner {
				tier = 1
			}
			key = distance
		} else if rule.Damaging || !rule.Restorative {
			if t.ID == caster.ID || !w.hostileTo(caster, t) {
				continue
			}
			// A unit fighting the enemy a player ordered it onto casts at that
			// enemy and at no other, however near another hostile stands.
			if holdsOrderedVictim(caster) && t.ID != caster.AttackTarget {
				continue
			}
			// Nearest wins, so the key ascends with the distance.
			key = distance
		} else {
			if t.Owner != caster.Owner && w.hostileTo(caster, t) && !rule.HealHostile || t.MaxHP <= 0 || t.HP >= t.MaxHP {
				continue
			}
			// Owner priority for idle healing is own team, then declared allies,
			// then neutrals. Inside a tier the decoded spell distance is the
			// stable first key and entity id the final tie-break. Deficit is kept
			// only after distance, so a nearer useful target is not skipped for a
			// farther one whose bar happens to be shorter.
			switch {
			case t.Owner == caster.Owner:
				tier = 0
			case w.relations.Locked(caster.Owner, t.Owner):
				tier = 1
			case w.hostileTo(caster, t):
				tier = 3
			default:
				tier = 2
			}
			key = int64(t.HP) - int64(t.MaxHP)
		}
		if visible == nil {
			visible = w.actorSight(ci)
		}
		if !w.sightShows(visible, cell{x: t.X, y: t.Y}) {
			continue
		}
		better := !found
		switch {
		case rule.Restorative:
			better = better || tier < bestTier || tier == bestTier && (distance < bestDistance ||
				distance == bestDistance && (key < bestDeficit || key == bestDeficit && t.ID < best))
		case rule.Defensive && !rule.Damaging:
			// Tier, then distance, then id. There is no deficit term: a buff
			// either improves the actor or is not a candidate at all, so the
			// gate has already removed every target the restorative arm would
			// have ranked by how far short it fell.
			better = better || tier < bestTier || tier == bestTier && (distance < bestDistance ||
				distance == bestDistance && t.ID < best)
		default:
			better = better || key < bestDistance || key == bestDistance && t.ID < best
		}
		if better {
			best, bestTier, bestDistance, bestDeficit, found = t.ID, tier, distance, key, true
		}
	}
	return best, found
}
