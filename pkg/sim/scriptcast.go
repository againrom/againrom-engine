package sim

// THE SCRIPT'S OWN CASTS.
//
// Instants 21 and 24 do not order a unit to cast. They build a TEMPORARY
// CASTER — an object with no runtime id, appended to a list that is not the
// actor list, which casts once and is then removed (`TRIG-CAST-033`,
// `TRIG-CASTACTOR-044`). Instant 21 aims it at a cell and instant 24 at a unit;
// the same constructor also serves the cell-entry unit-target helper
// (UNIT-M10CAST-056). cellentry.go supplies that source without instant 21's
// zero-power substitution.
//
// IT IS NOT AN ENTITY IN THIS BUILD, AND THAT IS THE DECODE'S OWN SHAPE. The
// object has no health, no order, no group, no owner and no runtime id, and
// it never occupies a cell. Making it an Entity would put it in the entity
// record, the occupancy plane and the route scratch, and hand every existing
// arm of the step a record it would have to learn to skip.
//
// IT IS HASHED WHILE IT IS LIVE. `TRIG-CAST-033` states that the transient
// actor is not one of the original save format's serialized top-level lists,
// while the effects it leaves behind are — so an original save cannot restore
// an in-flight cast, but a consumer must hash one. This build's byte form is its
// own and carries both (castbinary.go).

// scriptCast is one pending cast the mission script authored.
//
// EVERY FIELD IS THE WIDTH THE CONSTRUCTOR STORES IT AT. Storing any of them
// wider here would let this build cast a spell the original cannot name.
//
// Target and AtUnit are instant 24's own aim. The reference is resolved by
// the binder before it reaches this package, exactly as every other unit
// reference on a compiled record is, so this holds an entity id and not the id
// the map wrote. AtUnit is what separates the two decoded states: set is
// state 0xd, cast at the target; clear is state 0xe, cast at the destination
// bytes.
type scriptCast struct {
	FromX, FromY uint8
	ToX, ToY     uint8
	Spell        uint8
	Power        uint16
	Target       EntityID
	AtUnit       bool
}

// scriptCastDefaultPower is the power the constructor substitutes for an
// authored zero: the immediate 0x63 at L12471, ninety-nine. The arm reads `p5` and calls
// the helper with the authored value when it is non-zero and with this immediate
// when it is zero.
//
// EVERY SHIPPED INSTANT-21 NODE AUTHORS ZERO — all 70 across both preserved
// roots (`TRIG-CASTACTOR-044`) — so the shipped campaign's cell casts run
// entirely on this branch. Instant 24's nodes are not all zero: the shipped ones
// author 0, 1, 99 and 100.
const scriptCastDefaultPower = 99

// scriptCastPower applies that substitution once, so the two instant arms cannot
// come to disagree about it.
func scriptCastPower(p int32) uint16 {
	if uint16(p) == 0 {
		return scriptCastDefaultPower
	}
	return uint16(p)
}

// ScriptCast is one pending cast as a consumer sees it.
type ScriptCast struct {
	FromX, FromY int32
	ToX, ToY     int32
	Spell        uint16
	Power        uint16
	Target       EntityID
	AtUnit       bool
}

// ScriptCasts returns the world's pending script casts, in the order they were
// authored, as a fresh slice.
func (w *World) ScriptCasts() []ScriptCast {
	out := make([]ScriptCast, len(w.casts))
	for i, c := range w.casts {
		out[i] = ScriptCast{
			FromX: int32(c.FromX), FromY: int32(c.FromY),
			ToX: int32(c.ToX), ToY: int32(c.ToY),
			Spell: uint16(c.Spell), Power: c.Power,
			Target: c.Target, AtUnit: c.AtUnit,
		}
	}
	return out
}

// castAtCell is instant 21's whole arm: the six plain parameters are (fromX,
// fromY, toX, toY, spell, power), the order `TRIG-CAST-033` derives from the
// code and the shipped maps' own parameter names — "From X", "From Y", "To
// X", "To Y", "Spell", "Power/skill" — independently corroborate.
func (w *World) castAtCell(in ScriptInstant) {
	w.casts = append(w.casts, scriptCast{
		FromX: uint8(in.Args[0]), FromY: uint8(in.Args[1]),
		ToX: uint8(in.Args[2]), ToY: uint8(in.Args[3]),
		Spell: uint8(in.Args[4]),
		Power: scriptCastPower(in.Args[5]),
	})
}

// castAtUnit is instant 24's whole arm. Its packed parameters are (fromX,
// fromY, spell, power): the target is a REFERENCE and references are carried
// beside Args rather than inside the packing, so the spell and the power sit
// at 2 and 3 here where they sit at 4 and 5 above.
//
// A NODE WHOSE REFERENCE DID NOT RESOLVE CREATES NO CAST, which is the treatment
// every other arm of this build gives an unresolved reference: entity id zero is
// a real entity, so the flag and not the value is what says whether one was
// named.
func (w *World) castAtUnit(in ScriptInstant) {
	if !in.HasUnit {
		return
	}
	spell := uint8(in.Args[2])
	if i := indexOfEntity(w.entities, in.Unit); i >= 0 &&
		!w.spellIDTargetable(w.entities[i], uint32(spell)) {
		return
	}
	w.casts = append(w.casts, scriptCast{
		FromX: uint8(in.Args[0]), FromY: uint8(in.Args[1]),
		Spell:  spell,
		Power:  scriptCastPower(in.Args[3]),
		Target: in.Unit, AtUnit: true,
	})
}

// stepScriptCasts resolves every pending cast, in the order they were
// authored, and empties the list.
//
// IT RUNS AT THE HEAD OF THE TICK. The script pass runs on phase 6, after
// the tick's commands, so a cast authored on tick T is walked here at the
// head of T+1 at the earliest — which is the decoded machine's own
// minimum: the actor is appended to a list and a later tick's walker reaches
// it. How many ticks a real cast waits is the spell's own cast time, and no
// claim in this story's pin names the column that comes from (spec SC-3).
//
// EACH RECORD IS RESOLVED ONCE AND REMOVED, whether or not anything landed
// (spec SC-4). The decoded machine retries a cast that returns zero on a later
// actor tick; what makes it return zero is not decoded here, so this build does
// not hold a record it cannot decide.
//
// A successfully resolved record is observed separately from CastEvent. The
// temporary caster has no entity id, so ScriptCastEvent carries its two cells
// instead of manufacturing one.
func (w *World) stepScriptCasts(obs *castObs) {
	if len(w.casts) == 0 {
		return
	}
	casts := w.casts
	w.casts = w.casts[:0]
	for _, c := range casts {
		w.resolveScriptCast(c, obs)
	}
	// The list is emptied whatever the arms above did. Clearing the tail keeps
	// no entity id alive in the slice's spare capacity.
	for i := range casts {
		casts[i] = scriptCast{}
	}
}

// resolveScriptCast lands one cast.
//
// THE FORK IS ON THE SPELL ROW AND NOT ON THE INSTANT. `MAGIC-SHAPE-008`
// puts the shape on the row's own `Distribution system` column: value 1
// builds a PointEffect and REQUIRES A TARGET UNIT, printing "Spell, oops -
// can't cast point effect of x,y" when there is none; anything else builds
// an AreaEffect standing on a cell. So one routine serves both instants and
// they differ only in where the destination comes from — and "instant 21
// cannot cast a point spell" falls out rather than being written.
//
// A SPELL ID THAT NAMES NO ROW LANDS NOTHING. The constructor builds its spell
// object from the id, and a row the world's table does not hold has no columns
// to build one from.
func (w *World) resolveScriptCast(c scriptCast, obs *castObs) {
	rule, ok := w.findSpell(uint32(c.Spell))
	if !ok {
		return
	}
	if rule.Area {
		if x, y, landed := w.landAreaCast(c, rule, obs); landed {
			obs.recordScriptCast(c, rule.ID, x, y)
		}
		return
	}
	if x, y, landed := w.landPointCast(c, rule); landed {
		obs.recordScriptCast(c, rule.ID, x, y)
	}
}

// landAreaCast places the area effect an area row leaves behind.
//
// THE DESTINATION IS THE RECORD'S OWN FOR A CELL CAST AND THE TARGET'S CURRENT
// CELL FOR A UNIT CAST. A unit cast whose target the world no longer holds, or
// has reached the -10 ordinary-target boundary, lands nothing.
//
// THE LIFETIME IS `(AreaDuration << 4) + (power << 4)/10`, MAGIC-SHAPE-008's
// corrected expression — the first term is the `Area Effect Duaration` column
// and NOT the distribution value, which is the clause that row was retracted
// for. The second term is added only when the power is non-zero, which is the
// `L05105` branch. Both terms are computed in a width wider than the word the
// result is stored in and then truncated, which is what the original's own
// 16-bit field does.
func (w *World) landAreaCast(c scriptCast, rule SpellRule, obs *castObs) (int32, int32, bool) {
	x, y := int32(c.ToX), int32(c.ToY)
	if c.AtUnit {
		i := indexOfEntity(w.entities, c.Target)
		if i < 0 || !spellTargetable(w.entities[i], rule) {
			return 0, 0, false
		}
		x, y = w.entities[i].X, w.entities[i].Y
	}
	return x, y, w.landArea(rule, c.Power, 0, false, int32(c.FromX), int32(c.FromY), x, y, obs)
}

// landPointCast applies a point row to the cast's target unit.
//
// A CELL CAST LANDS NOTHING HERE. A point effect requires a target unit and a
// destination cell is not one; the original says so itself, by printing an error
// and building nothing.
//
// WHAT THIS BUILD DOES WITH A POINT ROW IS THE TWO ARMS IT ALREADY HAS (spec
// SC-1): a damaging row rolls its damage through applySpellDamage and a
// restorative row heals through applySpellHealing, both at the CAST'S OWN POWER
// rather than at a caster's — there is no caster to read a level or a Mind from,
// and the power is what the script authored or the 99 the constructor
// substituted.
//
// A ROW THAT IS NEITHER USED TO LAND NOTHING, and 0165's own comment recorded
// that as the story's chief limitation. It is no longer true: 1001 wired the
// point arms `MAGIC-CEIL-013` enumerates into the ordinary application path, so
// the six spells the shipped instant-24 nodes name — Stone Curse, Bless and the
// four Protections — now attach their effects here, at the cast's own power,
// through the same owner a commanded cast reaches: this function's own call is
// `ordinaryEffect(-1, i, rule, power)`. Whether every one of the 20 shipped
// instant-24 nodes lands state is a measurement nobody has taken; what changed
// is that the arm exists, not that its coverage was counted.
//
// NONE OF THE EIGHT REFUSALS A COMMANDED CAST TAKES IS ASKED. A script cast has
// no caster to be alive, to be a mage, to know the spell, to stand in range, to
// be off its cadence floor or to pay a mana cost; the constructor builds the
// spell object unconditionally. What is asked is only that the target is a unit
// the world holds and that it is alive.
func (w *World) landPointCast(c scriptCast, rule SpellRule) (int32, int32, bool) {
	if !c.AtUnit {
		return 0, 0, false
	}
	i := indexOfEntity(w.entities, c.Target)
	if i < 0 || !spellTargetable(w.entities[i], rule) || prismaticBodyPrimary(w.entities[i], rule) {
		return 0, 0, false
	}
	x, y := w.entities[i].X, w.entities[i].Y
	power := int32(c.Power)
	if rule.ID == 14 {
		w.applyPrismatic(-1, i, rule, power)
	} else if rule.Delivery == 2 {
		w.queueSpellDelivery(spellDelivery{Target: c.Target, X: x, Y: y, FromX: int32(c.FromX), FromY: int32(c.FromY), Rule: rule, Power: power})
	} else if !w.ordinaryEffect(-1, i, rule, power) {
		return 0, 0, false
	}
	// The mark says a unit was touched by a spell, and it is set only where
	// something was applied — on the target alone, because the caster this
	// mark's other half would name does not exist.
	if rule.Delivery != 2 {
		w.markSpellEffect(i, rule.ID)
	}
	return x, y, true
}
