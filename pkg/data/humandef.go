package data

import "fmt"

// lastHumanSlot is the last parameter slot this loader consumes. Slot 24 —
// serverID, the definition id the search reads — is still fetched on demand by
// other consumers; it gets a case here that stores nothing, so the cursor
// cannot drift past it into slot 25. Slot 25, the column the file titles
// knownSpells, is read here now: a bitmask subscripted by spell id, carried and
// interpreted no further (FR-4a).
//
// It moved from 22 to 23 when the corpse's dwell time gained a consumer, and
// from 23 to 25 when the row's own spellbook did. Each widening pays the same
// cost: a row that used to load now doesn't. A row of exactly 23 parameters was
// refused the day the bound reached 23; rows of 24 or 25 parameters — long
// enough for the old bound, short of the new one — are refused the same way now
// that the bound reaches 25.
const lastHumanSlot = 25

// MinHumanRow is the shortest Humans parameter row NewHumanDef accepts. It is
// EXPORTED so a caller building a synthetic row derives its width instead of
// writing one down: the bound has moved twice, and the second time a literal 24
// in another package's fixture turned a refusal anyone could read into four
// tests asserting against a zero-valued profile. A test that spells the number
// cannot fail usefully when the number changes.
const MinHumanRow = lastHumanSlot + 1

// HumanDef is one Humans row resolved into named fields: what a placed PERSON is
// built with.
//
// It is a second type beside UnitDef and not a second spelling of it. The two
// collections disagree about the thing that matters most: a unit's row carries
// its combat block and its class derives nothing from it, while a human's row
// carries NO damage pair, NO damage-routing selector, NO absorption, NO
// elemental protection and NO damage-kind resistance — five column families that
// exist only on the other collection. A person's blow numbers come out of the
// derived-stat graph instead, which is Combat below, and a loader that read this
// collection the way the other one is read would ship a person who swings for
// nothing and would look finished.
//
// Every field is int32 as the other definition's are, so a negative stays
// visibly negative, and no field ever holds the sentinel.
//
// Some fields are CARRIED AND READ BY NOTHING, and which ones is worth stating
// exactly, because the obvious guess is wrong twice over. The DEFENCE cell is
// replaced by the derivation, so it is carried and dead. The SKILL array is NOT:
// the derivation reads the level of the weapon's own kind out of it, so five of
// the six are live and only slot 0 is not — the streamed to-hit is that slot
// copied, and the derivation replaces the to-hit, while the gate on the active
// skill is on a positive slot so nothing reads the level itself. The mana pair,
// the scan range and the token size have no consumer in this tree at all. All
// of them are carried because the row states them and a later story reading a
// column this one silently discarded would have to decode it again — which is
// exactly what happened to the TYPE ID, the FACE and the GENDER: all three
// were dead here until a placed person had to be drawn, and the third had to
// be added back because it had been dropped rather than carried.
type HumanDef struct {
	Body, Reaction, Mind, Spirit int32

	// Health follows the maximum whenever the maximum is written, so a
	// definition is at full health by construction rather than by two numbers
	// someone keeps equal. The pair is the same shape the other definition's is.
	Health, HealthMax int32
	Mana, ManaMax     int32

	Speed         int32
	RotationSpeed int32
	ScanRange     int32

	// Defence is the ONE combat column this collection streams, and the
	// derivation replaces it. It is carried so the cell is not lost, never read.
	Defence int32

	// Skill is the six levels at their own slots, slot 0 first. Slots 1 to 5 are
	// LIVE — the derivation reads the level of the weapon's own kind — while slot
	// 0 reaches nothing: it is the cell the streamer copies into the live to-hit,
	// which the derivation replaces, and the active-skill gate is on a positive
	// slot.
	Skill [SkillSlots]int32

	TypeID int32
	Face   int32

	Gender int32

	// AttackChargeTime and AttackRelaxTime are the template's own cadence, and
	// unlike everything else a blow reads they SURVIVE the derivation: a weapon
	// assigns over each half only where its own cell is not empty.
	AttackChargeTime int32
	AttackRelaxTime  int32

	TokenSize    int32
	MovementType int32

	// DyingTime is the dwell of the body this person leaves, at this
	// collection's own slot for it. It is the same quantity the other
	// definition carries under the same name, at a different column.
	DyingTime int32

	// KnownSpells is the row's own spellbook: a bitmask subscripted by spell
	// id, bit i set meaning the row knows spell i (FR-4a). The sentinel means
	// no spells stated and lands as an empty book — 0 — never as every spell.
	// This loader carries the mask and interprets no further: no bit is
	// checked against a spell table, because a definition loader holds no
	// table.
	KnownSpells uint32
}

// HumanDefaults is the definition a Humans row of all empty cells yields: the
// constructor's defaults, taken through NewHumanDef so there is one statement
// of them. A saved Human whose definition row holds no parameters at all
// (row 0 of the installed collection) is bound to it.
func HumanDefaults() HumanDef {
	d, _ := NewHumanDef("", sentinelRowValues(lastHumanSlot+1))
	return d
}

// NewHumanDef builds a definition from one Humans row: the constructor's
// defaults with slots 0 to 25 written over them IN ASCENDING ORDER.
//
// THE DEFAULTS ARE THE OTHER ARM'S, taken through UnitDefaults rather than
// written out a second time, and that is a choice this tree makes rather than
// something it read. The per-cell law — an empty cell stores nothing and leaves
// the constructor's value standing — is the same law on both arms; which values
// the humans object's own constructor writes is not established, and the two
// arms allocate different sizes. Reusing the base actor's numbers is reachable
// only for a cell a shipped row leaves empty, and having one statement of them
// is what keeps two constants from coming to disagree.
//
// The order is not decoration. Six consecutive slots go to a skill array and the
// slot after the type id is read and dropped, so a loader written as "field i
// takes column i" mis-assigns everything from slot 10 onward — and would look
// right, because the ten fields before it would still agree.
//
// name is used for one thing only: naming the entry in a refusal.
//
// It fails on a row shorter than the streamed slots, yielding the zero value
// rather than a partly filled definition.
func NewHumanDef(name string, params []int32) (HumanDef, error) {
	if len(params) <= lastHumanSlot {
		return HumanDef{}, fmt.Errorf("data: human definition %q carries %d parameter(s); slots 0 to %d are streamed",
			name, len(params), lastHumanSlot)
	}

	u := unitCtorDefaults
	d := HumanDef{
		Body: u.Body, Reaction: u.Reaction, Mind: u.Mind, Spirit: u.Spirit,
		Health: u.Health, HealthMax: u.HealthMax,
		Mana: u.Mana, ManaMax: u.ManaMax,
		Speed: u.Speed, RotationSpeed: u.RotationSpeed, ScanRange: u.ScanRange,
		AttackChargeTime: u.AttackChargeTime, AttackRelaxTime: u.AttackRelaxTime,
		TokenSize: u.TokenSize, MovementType: u.MovementType, Face: u.Face,
		DyingTime: u.DyingTime,
	}

	c := &slotCursor{params: params}
	for slot := 0; slot <= lastHumanSlot; slot++ {
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
			if c.store(&d.ManaMax) {
				d.Mana = d.ManaMax
			}
		case 6:
			c.store(&d.Speed)
		case 7:
			c.store(&d.RotationSpeed)
		case 8:
			c.store(&d.ScanRange)
		case 9:
			c.store(&d.Defence)
		case 10, 11, 12, 13, 14, 15:
			c.store(&d.Skill[slot-10])
		case 16:
			c.store(&d.TypeID)
		case 17:
			c.store(&d.Face)
		case 18:
			// The gender cell. It was read and dropped until 0141, on the ground
			// that no consumer held one; the picture a placed person draws with
			// is that consumer.
			c.store(&d.Gender)
		case 19:
			c.store(&d.AttackChargeTime)
		case 20:
			c.store(&d.AttackRelaxTime)
		case 21:
			c.store(&d.TokenSize)
		case 22:
			c.store(&d.MovementType)
		case 23:
			c.store(&d.DyingTime)
		case 24:
			// Read and DROPPED. serverID is fetched from this same slot on
			// demand elsewhere, so carrying it here would be a second copy of
			// one column with no consumer. It gets a case that stores nothing
			// rather than being skipped by a bound, so the cursor cannot
			// drift.
			c.store(nil)
		case 25:
			// knownSpells: a bitmask subscripted by spell id. store takes a
			// *int32, so land the cell in a local and convert; the sentinel
			// (or any other negative the file never actually carries) maps to
			// 0, an empty book, rather than to a mask with the sign bit set.
			var spells int32
			c.store(&spells)
			if spells > 0 {
				d.KnownSpells = uint32(spells)
			}
		}
	}

	return d, nil
}

// Hero is the character this definition's row describes: its four statistics and
// its six skill levels, which are the INPUTS of the derived-stat graph.
//
// It is exported because "what character is this row" is a question more than
// one consumer will ask, and it should have exactly one answer. Nothing else in
// a HumanDef reaches that graph — the health, the rate, the cadence, the token
// size and the two art columns are all outside it.
func (d HumanDef) Hero() Hero {
	return Hero{Body: d.Body, Reaction: d.Reaction, Mind: d.Mind, Spirit: d.Spirit, Skill: d.Skill}
}

func (d HumanDef) Derived(w *Weapon) Derived {
	return d.DerivedWithLoadout(Profile{}, Loadout{Weapon: w})
}

// DerivedWithLoadout is Derived generalized to the caller's own Profile and
// full Loadout — the weapon fold AND the worn-set fold together — for a
// caller that has already resolved a real loadout rather than a bare weapon
// (hotfix map-hero-stats, docs/hotfix/LEDGER.md). Derived above is DEFINED
// IN TERMS OF THIS METHOD and not the reverse, so the cadence-override tail
// is stated once and neither name can drift from the other.
//
// THE ONE PRODUCTION CALLER (pkg/mapload blockFor's person arm) passes the
// row's own d.Profile() and a loadout resolved through
// mapload.ResolveEquipmentLoadout over the row's own starting worn set — the
// SAME two inputs a party member's own spawn (mapload.PartySpawnWithTable)
// and a live skill-raise (game.recomputeRaisedSkills) already resolve
// through, so a map-placed person's derived block agrees with what either
// of those would already give him.
func (d HumanDef) DerivedWithLoadout(p Profile, l Loadout) Derived {
	l.RotationSpeed = d.RotationSpeed
	der := d.Hero().Recompute(p, l)
	der.Combat.AttackChargeTime, der.Combat.AttackRelaxTime = d.AttackChargeTime, d.AttackRelaxTime
	if l.Weapon != nil {
		der.Combat.AttackChargeTime = cellOr(l.Weapon.ChargeTime, d.AttackChargeTime)
		der.Combat.AttackRelaxTime = cellOr(l.Weapon.RelaxTime, d.AttackRelaxTime)
	}
	return der
}

// Combat is what a blow reads for this person: the eight numbers his
// statistics and the weapon in his hand derive, plus Reach. A nil weapon is a
// BARE person, which is an ordinary state of a shipped row and not a hole in
// this one.
//
// It is DEFINED IN TERMS of Derived above rather than beside it, and that is
// the whole content of this arm: this collection ships no damage pair, no
// routing selector and no absorption, so the derived-stat graph is not one
// source among several — it is the ONLY source, and a build that read the
// columns and stopped would ship a person who swings for nothing. There is
// one statement of that graph in this tree and this is not a second one.
//
// It differs from a generated character in ONE respect, and Recompute
// supplies none of it: the cadence, which Derived above already folds in —
// see its own doc for why a placed person's cadence survives where a
// generated character's does not.
//
// Nothing else is touched. The damage pair, the to-hit, the defence, the
// absorption and the always-hits mark come out of the derivation exactly as it
// leaves them, and the mark is false because its only source is the OTHER
// collection's routing column, which this one does not have.
//
// It reads nothing but its receiver and its argument: no clock, no generator, no
// global, no file.
func (d HumanDef) Combat(w *Weapon) Combat {
	return d.Derived(w).Combat
}

func (d HumanDef) DerivedMaximum() int32 {
	return d.Hero().Recompute(d.Profile(), Loadout{}).HealthMax
}
