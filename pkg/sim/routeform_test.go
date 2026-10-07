package sim

// The route section of the byte form: its bytes, its digest, and everything it
// refuses.
//
// Nothing here asks the encoder where it put a route or what a routed world
// comes to. The pinned form is a literal transcription of the contract's table
// for one small world, written out by hand; the builder beside it is a second
// implementation of the same table, and the two are compared against each other
// so that a malformed case is a spoiled copy of something known good rather
// than of something the encoder happened to produce.
//
// The world here is built through NewWorld and then given its route through
// the field, because no constructor takes one and no exported call sets one.

import (
	"bytes"
	"sort"
	"testing"
)

// ------------------------------------------------- the fixture

// rtfBounds is four by four, and rtfGrid blocks (3,0) and (1,2) — the second of
// which is what the blocked-cell refusal below routes through.
var rtfBounds = Bounds{Width: 4, Height: 4}

func rtfGrid() []byte {
	return []byte{
		0, 0, 0, blockGround,
		0, 0, 0, 0,
		0, blockGround, 0, 0,
		0, 0, 0, 0,
	}
}

// rtfEnts are the three states a route section has to carry: a unit holding a
// target AND a route, a unit holding a target and NO route — which is what every
// unit looks like before its first routing — and a unit holding neither.
//
// The two holding targets carry a health pair, which is what keeps them ALIVE
// and so keeps those targets legal; the third is left at 0/0 — no health system,
// alive — deliberately, because the refusals below spoil that unit with a route
// it may not hold, and a unit that was also dead would be refused twice and
// witness neither rule.
func rtfEnts() []Entity {
	return []Entity{
		{ID: 2, X: 0, Y: 0, TargetX: 2, TargetY: 2, HasTarget: true, HP: 30, MaxHP: 40,
			Speed: 12, Transit: 3, TransitTotal: 22, GroupSpeed: 12, Owner: 1, Facing: 0x20,
			ScanRange: 6},
		{ID: 5, X: 0, Y: 3, TargetX: 3, TargetY: 3, Class: -1, HasTarget: true, Stall: 3, HP: 7, MaxHP: 9,
			Group: 0x11223344, Owner: 0x44332211, Facing: 0xe0, DyingTime: 17, ScanRange: 0xff},
		{ID: 9, X: 3, Y: 3, Speed: 35, TransitTotal: 8, GroupSpeed: 255, Facing: 0x11, DyingTime: 256},
	}
}

// rtfRoute is the one stored route: two cells, ending on unit 2's target.
func rtfRoute() []cell { return []cell{{1, 1}, {2, 2}} }

// rtfWorld is the fixture, at a tick it did not reach by stepping — the tick is
// written through the field for the reason the route is, and a nonzero one keeps
// the header's own bytes in the pin.
func rtfWorld(t *testing.T, routes ...[]cell) *World {
	t.Helper()
	w := mustWorldGrid(t, 0x1234, rtfBounds, ModeCanonical, rtfGrid(), rtfEnts())
	w.tick = 7
	for i, r := range routes {
		w.routes[i] = r
	}
	return w
}

// rtfBytes is the version-22 form of rtfWorld(t, rtfRoute()), transcribed
// from the contract's table: a 34-byte header, three 16-cell planes, three
// 92-byte records — each carrying a thirty-nine-byte attack block of
// zeroes, no unit here being in a fight, and closing on the group word, the
// owner slot and then the facing byte — three routes, one of two cells and
// two empty, then the GROUP SECTION rtfGroupSection supplies, then the SACK
// SECTION — this world names no sack, so it is a bare four-byte zero count
// (0103) — then the CARRY SECTION AND THE PURSE (0112), each entity
// carrying nothing and no gold credited, so three bare four-byte zero counts
// and 200 zeros — then the EQUIPMENT SECTION (0124 T2), between the two,
// this world equipping nothing so it is three bare EquipSlots-wide zero
// records, no count at all — then THE SPELL TABLE (version 36), right
// after the purse, this world naming no table so it is a bare two-byte zero
// count and no records — and then the SCRIPT SECTION, which
// rtfScriptSection supplies. A world mid-cycle is pinned in binary_test.go.
var rtfBytesPre1001Form = append(append(append(append(append(append(append(append(append(append(append(append(append(rtfHead(), rtfGroupSection(rtfEnts())...), make([]byte, sackCountLen)...),
	make([]byte, 3*carryCountLen)...), make([]byte, 3*equipRecordLen)...), make([]byte, 3*treasureRecordLen)...), make([]byte, purseLen)...),
	make([]byte, spellCountLen)...),
	make([]byte, itemWeightCountLen)...),
	make([]byte, 2*castingCountLen)...),
	rtfScriptStateSection()...),
	make([]byte, structureCountLen)...),
	rtfScriptSection()...), make([]byte, relationLen)...)

var rtfBytesV54 = widenedCorpseLootPin(widened1001Pin(rtfBytesPre1001Form, 16, 3), 16, 3)
var rtfBytesV56 = widenedCarriedWeightPin(widenedItemAttributionPin(rtfBytesV54, 16, 3), 16, 3)
var rtfBytesV58 = widenedMapUnitIDPin(rtfBytesV56, 16, 3)
var rtfBytesV59 = widenedWeaponResistancePin(rtfBytesV58, 16, 3)
var rtfBytesV60 = widenedWithdrawalThresholdPin(rtfBytesV59, 16, 3)
var rtfBytesV61 = widenedItemStatePin(rtfBytesV60, 3, len(rtfScriptSection())+relationLen)
var rtfBytesV62 = widenedActionCadenceEntityPin(rtfBytesV61, 16, 3)
var rtfBytesV63 = widenedTurnProgressPin(rtfBytesV62, 16, 3)
var rtfBytesV67 = widenedTurnDurationPin(rtfBytesV63, 16, 3)
var rtfBytesV68 = widenedConsumablePin(rtfBytesV67, 16, 3, len(rtfScriptSection())+relationLen)
var rtfBytesV69 = widenedHumanMovementPin(rtfBytesV68, 16, 3)
var rtfBytesV70 = widenedOriginalDeadPin(rtfBytesV69)
var rtfBytesV71 = widenedSpellbookPin(rtfBytesV70, 16, 3)
var rtfBytesV72 = widenedSecondPhysicalPin(rtfBytesV71, 16, 3)
var rtfBytesV73 = widenedCurrentProfilePin(rtfBytesV72, 16, 3)
var rtfBytesV74 = widenedInstanceWeightPin(rtfBytesV73)
var rtfBytesV75 = widenedActorLoadPin(rtfBytesV74)
var rtfBytesV76 = widenedSourceBindingPin(rtfBytesV75)

// rtfEnts' own highest id is 9 too (rtfEnts above); no originalDead and no
// script, so its floor is exactly that plus one, the same as pinBytes'.
var rtfBytes = widenedEntityIDFloorPin(widenedSessionClockPin(rtfBytesV76), 10)

// rtfTailAfterItemWeights is pinTailAfterItemWeights for the routed world: the
// same four sections, with this world's own script-state and script sections,
// and the structure section (1033 B3) between them.
var rtfTailAfterItemWeights = 2*castingCountLen + len(rtfScriptStateSection()) +
	structureCountLen + len(rtfScriptSection()) + relationLen

// rtfScriptStateSection is 0166's own section (version 50): the per-player
// formation modes and the cell-record tails. It is written out rather than
// appended as zeros, unlike every absent section above it, because the
// formation block's absent case is NOT zero — a world nothing has written
// a mode into carries the default in every slot. This world holds no cell
// tail, so the tail half is a bare zero count.
func rtfScriptStateSection() []byte {
	out := make([]byte, relationSlots, relationSlots+tailCountLen)
	for i := range out {
		out[i] = formationDefault
	}
	return append(out, make([]byte, tailCountLen)...)
}

// rtfGroupSection is 0095's own section, grown in place by 0096: a count,
// then that many (owner, group, base, order, commandedX, commandedY) records
// ascending BY OWNER. Id 9 carries no owner and contributes no record, so
// this world holds two — id 2's, whose owner (1) sorts before id 5's
// (0x44332211), which is the opposite of the order the two entities
// themselves hold in rtfEnts(). Id 2 is alone in its group with a sight range
// of 6, under the floor, so its base is the floor; id 5 is alone with a sight
// range of 255, so its base is that geometry exactly and pins the field's top
// on the SECOND record rather than the first, where a narrower field would
// still look right.
//
// ID 2'S OWNER IS SelfSlot (1), so its group's ORDER is Stand Ground — the
// one witness in this package's byte-form fixtures that the two orders
// differ at all; id 5's owner is not, so its group is Guard, exactly as
// pinBytes' own two records both are (binary_test.go). Both commanded cells
// are the origin, unread until a command writes one.
//
// It is written by hand from the same rule freezeGroups computes, not called
// through it, so a malformed case built from it is a spoiled copy of something
// this file derived on its own — the rule this whole file follows for every
// other section.
func rtfGroupSection(ents []Entity) []byte {
	type key struct{ owner, group uint32 }
	var keys []key
	for _, e := range ents {
		if e.Owner == 0 {
			continue
		}
		found := false
		for _, k := range keys {
			if k.owner == e.Owner && k.group == e.Group {
				found = true
				break
			}
		}
		if !found {
			keys = append(keys, key{e.Owner, e.Group})
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].owner < keys[j].owner ||
			(keys[i].owner == keys[j].owner && keys[i].group < keys[j].group)
	})

	out := []byte{byte(len(keys)), byte(len(keys) >> 8), byte(len(keys) >> 16), byte(len(keys) >> 24)}
	for _, k := range keys {
		// Every group this file ever builds has at most one LIVING member,
		// coincident with its own centroid — true of rtfEnts() and of every
		// variant rtfNotAlive derives from it, and this shortcut is refused
		// nowhere but claims nothing past that: a group with a second living
		// member is not a shape this file constructs.
		var widest uint32
		for _, e := range ents {
			if e.Owner != k.owner || e.Group != k.group || !e.Alive() {
				continue
			}
			if uint32(e.ScanRange) > widest {
				widest = uint32(e.ScanRange)
			}
		}
		base := widest
		if base < minimalGuardRange {
			base = minimalGuardRange
		}
		// The order, from the same rule freezeGroups applies at construction:
		// Stand Ground for SelfSlot's own groups, Guard for every other owner. The
		// commanded cell is the origin, unread.
		order := byte(orderGuard)
		if k.owner == SelfSlot {
			order = orderStandGround
		}
		out = append(out, byte(k.owner), byte(k.owner>>8), byte(k.owner>>16), byte(k.owner>>24))
		out = append(out, byte(k.group), byte(k.group>>8), byte(k.group>>16), byte(k.group>>24))
		out = append(out, byte(base))
		out = append(out, order)
		out = append(out, 0, 0, 0, 0) // commandedX 0
		out = append(out, 0, 0, 0, 0) // commandedY 0
	}
	return out
}

// rtfScriptSection is this world's script section, and it is a RUN OF ZEROS
// rather than a table because that is what it is: the world runs no script, so
// the hundred registers, the thousand latches, the two counters, the outcome and
// the three array counts are every one of them zero. Transcribing 1421 zeros by
// hand would check nothing the length does not; what is pinned is that the
// section is exactly this long and entirely zero, which is the claim — a world
// with no script writes the same bytes a world whose script has done nothing
// writes.
func rtfScriptSection() []byte {
	return make([]byte, scriptStateLen+scriptCountsLen)
}

func rtfHead() []byte {
	return []byte{
		0x32,                                           // version 50: the last script arms (0166)
		0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 7
		0x34, 0x12, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // rng state = the seed
		0x04, 0x00, 0x00, 0x00, // bounds.Width 4
		0x04, 0x00, 0x00, 0x00, // bounds.Height 4
		0x03, 0x00, 0x00, 0x00, // entity count 3
		0x00,                   // routing mode: canonical
		0x10, 0x00, 0x00, 0x00, // plane cell count 16 — ONE count, all three planes

		0x00, 0x00, 0x00, 0x01, // block row 0
		0x00, 0x00, 0x00, 0x00, // block row 1
		0x00, 0x01, 0x00, 0x00, // block row 2
		0x00, 0x00, 0x00, 0x00, // block row 3

		// The cost plane, materialised at the uniform default this world named
		// none of, and the height plane materialised flat beneath it.
		0x08, 0x08, 0x08, 0x08, // cost row 0
		0x08, 0x08, 0x08, 0x08, // cost row 1
		0x08, 0x08, 0x08, 0x08, // cost row 2
		0x08, 0x08, 0x08, 0x08, // cost row 3

		0x00, 0x00, 0x00, 0x00, // height row 0
		0x00, 0x00, 0x00, 0x00, // height row 1
		0x00, 0x00, 0x00, 0x00, // height row 2
		0x00, 0x00, 0x00, 0x00, // height row 3

		0x02, 0x00, 0x00, 0x00, // id 2
		0x00, 0x00, 0x00, 0x00, // X 0
		0x00, 0x00, 0x00, 0x00, // Y 0
		0x02, 0x00, 0x00, 0x00, // TargetX 2
		0x02, 0x00, 0x00, 0x00, // TargetY 2
		0x00, 0x00, 0x00, 0x00, // Class 0
		0x01,                   // HasTarget
		0x00,                   // stall count 0
		0x1e, 0x00, 0x00, 0x00, // HP 30
		0x28, 0x00, 0x00, 0x00, // MaxHP 40
		0x00,                   // movement domain: ground
		0x0c, 0x00, 0x00, 0x00, // Speed 12
		0x03, 0x00, // 3 transit ticks owed
		0x16, 0x00, // of a transit of 22
		0x0c, // a group rate term of 12 — the same as its own speed, which is
		//             what a group of one leaves
		0x00, 0x00, 0x00, 0x00, // no victim
		0x00,                   // no presence
		0x00,                   // phase 0: ready
		0x00, 0x00, 0x00, 0x00, // and nothing owed — this one is not attacking
		0x00, 0x00, 0x00, 0x00, // charge 0
		0x00, 0x00, 0x00, 0x00, // relax 0
		0x00, 0x00, 0x00, 0x00, // to-hit 0
		0x00, 0x00, 0x00, 0x00, // defence 0
		0x00, 0x00, 0x00, 0x00, // absorption 0
		0x00, 0x00, 0x00, 0x00, // damage base 0
		0x00, 0x00, 0x00, 0x00, // damage spread 0
		0x00,                   // and it can miss
		0x00, 0x00, 0x00, 0x00, // group 0 — a real group, and the one an entity
		//             built without naming one is in
		0x01, 0x00, 0x00, 0x00, // owner: roster slot 1
		0x20,       // facing 0x20 — direction 1, north-east
		0x00,       // no decay stage: alive
		0x00, 0x00, // and no dwell
		0x00, 0x00, 0x00, 0x00, // dying time 0 — no dwell either way
		0x06, // a sight range of 6 cells
		0x0b, // actor state: guard (0099) — NewWorld writes it into every
		//             entity unconditionally
		0x00, 0x00, 0x00, 0x00, // patrol head X — residue, on the stall
		0x00, 0x00, 0x00, 0x00, // patrol head Y — count's own ground: an
		0x00, 0x00, 0x00, 0x00, // patrol tail X — actor that is not
		0x00, 0x00, 0x00, 0x00, // patrol tail Y — patrolling carries no ring
		0x00, // patrol leg: head (0)
		0x01, // reach 1 — this entity names none, so the constructor's own
		//             floor is what the byte carries (0104)
		0x00, 0x00, 0x00, 0x00, // post X 0 — the cell this entity was built
		0x00, 0x00, 0x00, 0x00, // post Y 0 — at (0106): PostX, PostY == X, Y
		0x00, 0x00, 0x00, 0x00, // mana 0 — this fixture names no mana pool at all
		0x00, 0x00, 0x00, 0x00, // mana maximum 0, so the pool has no system (0109)
		0x00, 0x00, 0x00, 0x00, // health regen period 0 — this fixture names none
		0x00, 0x00, 0x00, 0x00, // mana regen period 0 — nor this one
		0x00,                   // health remainder 0
		0x00,                   // mana remainder 0
		0x00, 0x00, 0x00, 0x00, // command group 0 (0117) — this task writes
		//             nothing here, so every actor decides by its placed
		//             group alone
		0x00, 0x00, 0x00, 0x00, // SkillXP[0..5] (0125) — this entity earns no
		0x00, 0x00, 0x00, 0x00, // experience at all, so every slot is the
		0x00, 0x00, 0x00, 0x00, // zero the constructor leaves it at
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // Mind 0 — this fixture names none
		0x00, 0x00, 0x00, 0x00, // XPValue 0 — nor this one
		0x00,                   // XPSlot 0: General, this fixture's own zero
		0x00,                   // GainsXP false: this class does not gain
		0x00, 0x00, 0x00, 0x00, // known spells 0 (0127) — this fixture names none
		0x00, 0x00, 0x00, 0x00, // Skill[0..5] (0135) — this fixture names no
		0x00, 0x00, 0x00, 0x00, // levels at all, so every slot is the zero
		0x00, 0x00, 0x00, 0x00, // the constructor leaves it at
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, // weapon spell 0 (0139) — this fixture names none
		0x00, 0x00, 0x00, 0x00, // weapon spell level 0
		0x00, 0x00, // autocast spell 0 (0154) — no autocast, on the weapon
		//             spell's own reserved-row ground
		0x00,       // autocast wait 0
		0x00, 0x00, // spell effect mark 0 ticks — A WORD SINCE VERSION 50 (0166)
		0x00,                   // and no marking spell
		0x00,                   // and on the map (0164) — no fixture here is removed
		0x00, 0x00, 0x00, 0x00, // and escorting nothing (0166): no target,
		0x00, //             no target present,
		0x00, //             and no range

		0x05, 0x00, 0x00, 0x00, // id 5
		0x00, 0x00, 0x00, 0x00, // X 0
		0x03, 0x00, 0x00, 0x00, // Y 3
		0x03, 0x00, 0x00, 0x00, // TargetX 3
		0x03, 0x00, 0x00, 0x00, // TargetY 3
		0xff, 0xff, 0xff, 0xff, // Class -1
		0x01,                   // HasTarget
		0x03,                   // stall count 3
		0x07, 0x00, 0x00, 0x00, // HP 7
		0x09, 0x00, 0x00, 0x00, // MaxHP 9
		0x00,                   // movement domain: ground
		0x00, 0x00, 0x00, 0x00, // Speed 0 — this one has no rate
		0x00, 0x00, // and so no transit either
		0x00, 0x00,
		0x00,                   // and no group term, so no rate from that source either
		0x00, 0x00, 0x00, 0x00, // no victim
		0x00,                   // no presence
		0x00,                   // phase 0: ready
		0x00, 0x00, 0x00, 0x00, // and nothing owed — this one is not attacking
		0x00, 0x00, 0x00, 0x00, // charge 0
		0x00, 0x00, 0x00, 0x00, // relax 0
		0x00, 0x00, 0x00, 0x00, // to-hit 0
		0x00, 0x00, 0x00, 0x00, // defence 0
		0x00, 0x00, 0x00, 0x00, // absorption 0
		0x00, 0x00, 0x00, 0x00, // damage base 0
		0x00, 0x00, 0x00, 0x00, // damage spread 0
		0x00,                   // and it can miss
		0x44, 0x33, 0x22, 0x11, // group 0x11223344 — four distinct bytes, so a
		//             group read out of the neighbouring record lands on a zero
		0x11, 0x22, 0x33, 0x44, // owner 0x44332211 — the group's four bytes reversed,
		//             so the two adjacent words cannot be read for each other
		0xe0,       // facing 0xe0 — direction 7, north-west
		0x00,       // no decay stage: alive
		0x00, 0x00, // and no dwell
		0x11, 0x00, 0x00, 0x00, // dying time 17
		0xff,                   // a sight range of 255 — the top of the field, so its width is pinned
		0x0b,                   // actor state: guard
		0x00, 0x00, 0x00, 0x00, // patrol head X
		0x00, 0x00, 0x00, 0x00, // patrol head Y
		0x00, 0x00, 0x00, 0x00, // patrol tail X
		0x00, 0x00, 0x00, 0x00, // patrol tail Y
		0x00,                   // patrol leg: head (0)
		0x01,                   // reach 1
		0x00, 0x00, 0x00, 0x00, // post X 0
		0x03, 0x00, 0x00, 0x00, // post Y 3 — PostX, PostY == X, Y (0106)
		0x00, 0x00, 0x00, 0x00, // mana 0 — this fixture names no mana pool at all
		0x00, 0x00, 0x00, 0x00, // mana maximum 0, so the pool has no system (0109)
		0x00, 0x00, 0x00, 0x00, // health regen period 0 — this fixture names none
		0x00, 0x00, 0x00, 0x00, // mana regen period 0 — nor this one
		0x00,                   // health remainder 0
		0x00,                   // mana remainder 0
		0x00, 0x00, 0x00, 0x00, // command group 0 — this task writes it
		//             nowhere either
		0x00, 0x00, 0x00, 0x00, // SkillXP[0..5] — CARRIED on this unit for
		0x00, 0x00, 0x00, 0x00, // the group word's own reason: neither is
		0x00, 0x00, 0x00, 0x00, // residue, and this class does not gain
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // Mind 0
		0x00, 0x00, 0x00, 0x00, // XPValue 0
		0x00,                   // XPSlot 0: General
		0x00,                   // GainsXP false
		0x00, 0x00, 0x00, 0x00, // known spells 0 (0127) — CARRIED on this unit
		//             for the group word's own reason: neither is residue
		0x00, 0x00, 0x00, 0x00, // Skill[0..5] (0135) — CARRIED on this unit
		0x00, 0x00, 0x00, 0x00, // for the group word's own reason: neither is
		0x00, 0x00, 0x00, 0x00, // residue, and this fixture names no levels
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, // weapon spell 0 (0139) — carried the same way
		0x00, 0x00, 0x00, 0x00, // weapon spell level 0
		0x00, 0x00, // autocast spell 0 (0154) — no autocast, on the weapon
		//             spell's own reserved-row ground
		0x00,       // autocast wait 0
		0x00, 0x00, // spell effect mark 0 ticks — A WORD SINCE VERSION 50 (0166)
		0x00,                   // and no marking spell
		0x00,                   // and on the map (0164) — no fixture here is removed
		0x00, 0x00, 0x00, 0x00, // and escorting nothing (0166): no target,
		0x00, //             no target present,
		0x00, //             and no range

		0x09, 0x00, 0x00, 0x00, // id 9
		0x03, 0x00, 0x00, 0x00, // X 3
		0x03, 0x00, 0x00, 0x00, // Y 3
		0x00, 0x00, 0x00, 0x00, // TargetX 0 — no target, so no residue
		0x00, 0x00, 0x00, 0x00, // TargetY 0
		0x00, 0x00, 0x00, 0x00, // Class 0
		0x00,                   // HasTarget false
		0x00,                   // stall count 0
		0x00, 0x00, 0x00, 0x00, // HP 0
		0x00, 0x00, 0x00, 0x00, // MaxHP 0 — no health system, and so alive
		0x00,                   // movement domain: ground
		0x23, 0x00, 0x00, 0x00, // Speed 35
		0x00, 0x00, // no transit ticks owed
		0x08, 0x00, // of a transit of 8 — a crossing outlives its own count
		0xff, // a group rate term of 255, the widest byte the field holds,
		//             which only a negative class speed can put there
		0x00, 0x00, 0x00, 0x00, // no victim
		0x00,                   // no presence
		0x00,                   // phase 0: ready
		0x00, 0x00, 0x00, 0x00, // and nothing owed — this one is not attacking
		0x00, 0x00, 0x00, 0x00, // charge 0
		0x00, 0x00, 0x00, 0x00, // relax 0
		0x00, 0x00, 0x00, 0x00, // to-hit 0
		0x00, 0x00, 0x00, 0x00, // defence 0
		0x00, 0x00, 0x00, 0x00, // absorption 0
		0x00, 0x00, 0x00, 0x00, // damage base 0
		0x00, 0x00, 0x00, 0x00, // damage spread 0
		0x00,                   // and it can miss
		0x00, 0x00, 0x00, 0x00, // group 0
		0x00, 0x00, 0x00, 0x00, // owner 0 — no owner at all
		0x11, // facing 0x11 — NOT a multiple of 32, so this record is the one
		//             that fails if a facing is rounded to a direction on the way
		//             in or out rather than carried whole
		0x00, // no decay stage: a health pair of 0/0 is no health system,
		//             which is ALIVE
		0x00, 0x00, // and no dwell
		0x00, 0x01, 0x00, 0x00, // dying time 256 — past a byte, so a record read
		//             one byte narrow lands here
		0x00, // a sight range of 0 — A RANGE AND NOT AN ABSENCE. Such a unit
		//             lights the cell it stands in and no other, and nothing
		//             anywhere supplies a value in its place
		0x0b,                   // actor state: guard
		0x00, 0x00, 0x00, 0x00, // patrol head X
		0x00, 0x00, 0x00, 0x00, // patrol head Y
		0x00, 0x00, 0x00, 0x00, // patrol tail X
		0x00, 0x00, 0x00, 0x00, // patrol tail Y
		0x00,                   // patrol leg: head (0)
		0x01,                   // reach 1
		0x03, 0x00, 0x00, 0x00, // post X 3
		0x03, 0x00, 0x00, 0x00, // post Y 3 — PostX, PostY == X, Y (0106)
		0x00, 0x00, 0x00, 0x00, // mana 0 — this fixture names no mana pool at all
		0x00, 0x00, 0x00, 0x00, // mana maximum 0, so the pool has no system (0109)
		0x00, 0x00, 0x00, 0x00, // health regen period 0 — this fixture names none
		0x00, 0x00, 0x00, 0x00, // mana regen period 0 — nor this one
		0x00,                   // health remainder 0
		0x00,                   // mana remainder 0
		0x00, 0x00, 0x00, 0x00, // command group 0 — id 9 carries no owner
		//             at all and this task writes the field nowhere
		//             regardless
		0x00, 0x00, 0x00, 0x00, // SkillXP[0..5] — id 9's own zero, on the
		0x00, 0x00, 0x00, 0x00, // command group's own reason: this task
		0x00, 0x00, 0x00, 0x00, // gives none of the three any writer
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // Mind 0
		0x00, 0x00, 0x00, 0x00, // XPValue 0
		0x00,                   // XPSlot 0: General
		0x00,                   // GainsXP false
		0x00, 0x00, 0x00, 0x00, // known spells 0 (0127) — id 9's own zero, on
		//             the command group's own reason: this task gives none
		//             of the three any writer
		0x00, 0x00, 0x00, 0x00, // Skill[0..5] (0135) — id 9's own zero, on
		0x00, 0x00, 0x00, 0x00, // the known-spells mask's own reason: this
		0x00, 0x00, 0x00, 0x00, // task gives none of the three any writer
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, // weapon spell 0 (0139) — id 9's own zero, on the same reason
		0x00, 0x00, 0x00, 0x00, // weapon spell level 0
		0x00, 0x00, // autocast spell 0 (0154) — no autocast, on the weapon
		//             spell's own reserved-row ground
		0x00,       // autocast wait 0
		0x00, 0x00, // spell effect mark 0 ticks — A WORD SINCE VERSION 50 (0166)
		0x00,                   // and no marking spell
		0x00,                   // and on the map (0164) — no fixture here is removed
		0x00, 0x00, 0x00, 0x00, // and escorting nothing (0166): no target,
		0x00, //             no target present,
		0x00, //             and no range

		0x02, 0x00, 0x00, 0x00, // id 2's route: two cells
		0x01, 0x00, 0x00, 0x00, // (1,1)
		0x01, 0x00, 0x00, 0x00,
		0x02, 0x00, 0x00, 0x00, // (2,2)
		0x02, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // id 5 holds a target and no route
		0x00, 0x00, 0x00, 0x00, // id 9 holds neither
	}
}

// rtfDigest is FNV-1a over rtfBytes, computed from the transcription above
// outside this tree and checked against it below through hash_test.go's own
// implementation. Two pins that check each other, neither taken from the
// encoder.
//
// The version-38 value it replaces — this constant's own predecessor,
// 0x25661e31ab76e502, 0129's own landed state — does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THAT story's own bytes are AN ENTITY'S SIX SKILL
// LEVELS — twenty-four bytes at the tail of every one of this fixture's
// three records, each carrying its zero value since no entity here names one
// — and TestThePinIsThePreStoryPinPlusTheSkill in binary_test.go, run
// directly on this value's own bytes, is what checks that nothing else
// moved: stripped of exactly the twenty-four bytes per record this story
// added, it must fall back to the version-38 value here.
//
// The version-39 value under THAT — this constant's own predecessor,
// 0xb79e61f2927374bf, 0135's own landed state — does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THIS story's own bytes are A WEAPON'S OWN SPELL
// — a uint16 and an int32 at the tail of every one of this fixture's three
// records, each carrying its zero value since no entity here names a weapon
// or a spell — and TestThePinIsThePreStoryPinPlusTheWeaponSpell in
// binary_test.go, run directly on this value's own bytes, is what checks
// that nothing else moved: stripped of exactly the six bytes per record this
// story added, it must fall back to the version-39 value above.
//
// The version-36 value under THAT — this constant's own predecessor,
// 0x428b254966d424d8, 0127's own landed state — does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THIS story's own bytes are the compiled
// instant's SECOND UNIT REFERENCE — five bytes at the tail of every
// instant record — and they contribute NOTHING here: rtfScriptSection()
// carries no instant, so a record that widens by five bytes still multiplies
// by zero. Only the version byte moved. Version 37 is not pinned separately;
// this tree never wrote one — it is allocated to a story running in
// parallel off the same master this one branched from.
//
// The version-35 value under THAT — this constant's own predecessor before
// 0129, 0xcba5b9ca7ed7891b, 0125's own landed state — does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THAT story's own bytes are AN ENTITY'S OWN
// SPELLBOOK (0127 FR-4b) — one word at the tail of every one of this
// fixture's three records, each carrying its zero value since no entity here
// names a KnownSpells mask — AND THE SPELL TABLE, a new section right
// after the purse section, this fixture's own bare two-byte zero count since
// it names no table — and TestThePinIsThePreStoryPinPlusTheSpells in
// binary_test.go, run on this value's own bytes, is what checks that nothing
// else moved: stripped of exactly the four bytes per record and the section
// this story added, it must fall back to the version-35 value above.
//
// The version-34 value under THAT — binary_test.go's own
// preExperienceRoutedDigest, obtained by running the test that derives it,
// on the same rule every number here was — does not carry over either, on
// the same rule the values before it did not: byte 0 moved, so no prefix
// survives the bump. THAT story's own bytes are AN ENTITY'S EXPERIENCE FROM
// USE (0125) — six slot experiences, a Mind, an experience value, a
// credited slot and a gains flag at the tail of every one of this
// fixture's three records, each carrying its zero value since no entity
// here names an experience, a Mind or a gaining class — and
// TestThePinIsThePreStoryPinPlusTheExperience in binary_test.go, now run
// on strippedOfTheSpells' own output rather than directly on this value's
// own bytes, is what checks that nothing else moved: stripped of exactly
// the thirty-four bytes per record that story added, it must fall back to
// the version-34 value below — 0124's own landed state, not the version-32
// value an unmerged 0125 would have replaced.
//
// The version-32 value under THAT, 0x7cf2b3820d012994, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THIS story's own bytes are THE EQUIPMENT
// SECTION (0124 T2) — three bare EquipSlots-wide zero records, one per
// entity, no count of its own, between the carry section and the purse —
// and TestThePinIsThePreStoryPinPlusTheEquipment in binary_test.go, now run
// on strippedOfTheExperience's own output rather than directly on this
// value's own bytes, is what checks that nothing else moved: stripped of
// exactly the one section this story added, it must fall back to the
// version-32 value here.
//
// The version-31 value under THAT, 0xd1b9e47807319d9f, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THIS story's own bytes are THE COMMAND GROUP —
// one word at the tail of every one of this fixture's three records, each
// carrying its zero value since this task gives the field no writer at
// all — and TestThePinIsThePreStoryPinPlusTheCommandGroup in binary_test.go,
// run directly on this value's own bytes, is what checks that nothing else
// moved: stripped of exactly the four bytes per record this story added, it
// must fall back to the version-31 value here.
//
// The version-26 value under THAT, 0x97b685e24f862f36, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THIS story's own bytes are the compiled check
// record's widened tail (0122) — Player, Player2 and their two presence
// bytes — and they contribute NOTHING here: rtfScriptSection() carries no
// check, so a record that widens by ten bytes still multiplies by zero.
// Only the version byte moved. Versions 27 through 30 are not pinned
// separately; this story never wrote one (0122 D-1).
//
// The version-25 value under THAT, 0x27dac47f6005d3a1, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THIS story's own bytes are THE CARRY SECTION
// AND THE PURSE (0112) — three bare four-byte zero counts, one per entity,
// and 200 zeros, between the sack section and the script section — and
// TestThePinIsThePreStoryPinPlusTheCarryAndPurse in binary_test.go, run on
// strippedOfTheCommandGroup's own output, is what checks that nothing else
// moved: stripped of exactly the two sections this story added, it must
// fall back to the version-25 value here.
//
// The version-24 value under THAT, 0x3f5b7e651f1b6c84, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. THIS story's own bytes are THE REGENERATION
// BLOCK — six fields at the tail of every one of this fixture's three
// records, each carrying its zero value since no entity here names a pool
// or a period — and TestThePinIsThePreStoryPinPlusTheRegen in
// binary_test.go, run directly on this value's own bytes, is what checks
// that nothing else moved: stripped of exactly the eighteen bytes per
// record this story added, it must fall back to the version-24 value here.
//
// The version-23 value under THAT, 0x030da70bda8dd508, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. That story's own bytes are THE POST — two int32
// at the tail of every one of this fixture's three records, each reading
// back that record's own X, Y, since NewWorld writes it from an entity's own
// cell unconditionally and nothing here is a stance command's own re-anchor —
// and TestThePinIsThePreStoryPinPlusThePost in binary_test.go, now run on
// strippedOfTheRegen's own output, is what checks that nothing else moved:
// stripped of exactly the eight bytes per record that story added, it must
// fall back to the version-23 value here.
//
// The version-22 value under THAT, 0x5a8a000b43c044bc, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. That story's own bytes are THE REACH — one byte
// at the tail of every one of this fixture's three records, each carrying
// the constructor's own floor of 1 since no entity here names a weapon — and
// TestThePinIsThePreStoryPinPlusTheReach in binary_test.go, run on
// strippedOfThePost's own output, is what checks that nothing else moved:
// stripped of exactly the one byte per record that story added, it must
// fall back to the version-22 value here.
//
// The version-21 value under THAT, 0xd1d6de06284199ad, does not carry over
// either, on the same rule the values before it did not: byte 0 moved, so no
// prefix survives the bump. That story's own bytes are the SACK SECTION —
// this fixture's own bare four-byte zero count, between the group section
// and the script section — and TestThePinIsThePreStoryPinPlusTheActorState
// in binary_test.go, now run on strippedOfTheReach's and then
// strippedOfTheSacks' own output, is what checks that nothing else moved:
// this story's value, stripped of exactly the section it added and then of
// the actor state's own tail, must fall back to the version-20 value below.
//
// The version-20 value under THAT, 0x54ca89f6d34ff49b, does not carry over
// either, on the same rule: this story's own bytes sit outside the actor
// state's own tail — the record's own width is unmoved by this story — which
// is exactly what stripping the sack section first and then the tail second
// is checked to reach.
//
// The version-19 value before THAT, 0xd5ef1778ab86438e, does not carry over by
// arithmetic — nor did version 18's 0xdb02db658ef7abe7, version 11's
// 0x5e1a6c6e7ea13679, version 10's 0x6b56295a3f0a55f8, version 9's
// 0xef10f5dc60e4207d, version 8's 0x1408d478f23f004a, version 7's
// 0xb0fd8217e5af0510, version 6's 0x695327a5e8b998a9, version 5's
// 0x6fc8c538db18c544 or version 4's 0x79e69cc8097021b9 before any of them.
// FNV-1a is a left fold, so one form's digest determines another's only where
// the first is a PREFIX of the second; byte 0 itself moved here, so the two
// forms share no prefix and the number was derived afresh — over the
// transcribed bytes above, this story's widened group section, and the script
// section's 1421 zeros.
//
// Version 20's group section is the newest thing in the prefix and nothing in
// front of it moved — only the record's own width — so this value was checked
// by stripping the section back off, AT ITS NEW WIDTH, and confirming the
// version-18 value above still falls out (unchanged from before this story,
// since the whole section vanishes either way) — see
// TestThePinIsThePreStoryPinPlusTheGroupSection in binary_test.go, which runs
// this derivation for both pinned fixtures in one place.
//
// That the number is new is not evidence that only the intended bytes moved.
// TestTheRoutedPinIsThePreStoryPinPlusTheOwnerWord and
// TestTheRoutedPinIsThePreStoryPinPlusTheGroupWord are, and they run this
// transcription backwards onto literals two and three stories further back,
// unaffected by this one.
//
// 0156 MOVES IT AGAIN, on pinDigest's own ground: this world writes a script
// section with no record of any kind, so the compiled instant's new item
// tail contributes no byte and the whole of the bump is byte 0. The
// version-45 value it replaces was 0xef2d1ee6b356beb9.
//
// 1029 MOVES IT AGAIN, on pinDigest's own ground: each entity record gains two
// bytes of authored map id at +275, zero for every fixture here because none
// was placed by a map, and this world's script section still carries no record
// of any kind. TestThePinIsThePreviousPinPlusTheAuthoredMapID (binary_test.go)
// peels that tail back off and requires what is left to hash to the value below.
// The version-56 value it replaces was 0xc182806d328a32f8.
//
// 1033 MOVES IT AGAIN (B3). This world declares no structure, so the new
// structure section is its own bare four-byte zero count.
// TestThePinIsThePreviousPinPlusTheStructureSection (binary_test.go) peels
// that section back off and requires what is left to hash to the value
// below. The version-57 value it replaces was 0xe7ac2e14cf3c5bb1.
//
// 1039 MOVES IT AGAIN: every fixture entity appends five zero resistance
// bytes and byte 0 carries version 59. The version-58 value it replaces was
// 0x14a465f5adef7332; the independent peel in binary_test.go proves the delta.
//
// 1047 MOVES IT AGAIN: every fixture entity appends the inactive turn tail
// (current facing repeated, zero remainder, then zero request-time total)
// and byte 0 carries the current version. The pre-story value it replaces
// was 0x88ba00a8b3220e61. 1052 widens structure records. This fixture has no
// structures, so only the current form-version byte moves. 1063 moves it
// again for raw cloud counters; this fixture has no area record. Form 72
// appends two zero bytes per actor; the prior digest stays a control.
// Form76: independently transcribed form75 plus three zero 35-byte bindings.
// Form77: independently transcribed form76 plus the five absent-clock bytes.
// Form78: independently transcribed form77 plus the four absent-Group-span
// bytes, through the same widenedSessionClockPin -> widenedSavedGroupPin
// chain hash_test.go's pinDigest now documents; this routed world carries no
// saved Group either, so the span stays explicitly absent. Form79:
// independently transcribed form78 plus the four absent-structure-span
// bytes, through the same widenedSavedGroupPin -> widenedSavedStructurePin
// chain hash_test.go's pinDigest now documents; this routed world carries no
// saved structure either, so the span stays explicitly absent. Form80:
// independent form79 bytes plus four absent-Player-container bytes. Form81:
// independent form80 plus four absent-stride bytes. The old digest is
// retained in TestNativeStride1115IndependentWireAndHistoricalAbsence.
// Form82 adds only the absent fine-motion footer to that independent pin;
// TestSavedMotion1115LegacyExplicitAbsence retains the genuine form81
// digest. Form83 adds the absent raw-plane footer; its form82 predecessor
// remains a literal control in
// TestSavedCellPlanes1115Historical82LiteralPinDigests. Form84 adds only the
// absent current-object footer; the form83 digest is retained by
// TestSavedCellPlanes1115Historical83LiteralPinDigests. Form85 adds only the
// absent carried-resume footer, through rtfBuild's own version>=85 clause;
// this routed world carries no resume state either, so the span stays
// explicitly absent. Form87 adds only its four zero counter-footer bytes and
// version tag.
const rtfDigestV91 uint64 = 0x45063a17eec8d78f

var rtfDigest = entityIDFloorDigest1177(rtfBytes, rtfDigestV91, 10)

const prePlayerContainersRoutedDigest uint64 = 0xb14c31c15c4cdc43

// rtfBuild is a second implementation of the same table: it writes a whole form
// out of the pieces it is given, little-endian by hand. It exists so that a
// malformed case is a deliberate change to something known good — every case
// below differs from the valid form in exactly the one way its name says.
// From version 13 the three planes are written from the block plane's own
// length: this builder takes one plane and materialises the other two exactly as
// the constructor does, so a spoiled copy still differs from the valid form in
// one way only.
func rtfBuild(version byte, tick uint64, seed uint64, b Bounds, mode Mode, grid []byte,
	count uint32, ents []Entity, routes [][]cell) []byte {
	put32 := func(out []byte, v uint32) []byte {
		return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	}
	out := []byte{version}
	for i := 0; i < 8; i++ {
		out = append(out, byte(tick>>(8*i)))
	}
	for i := 0; i < 8; i++ {
		out = append(out, byte(seed>>(8*i)))
	}
	out = put32(out, uint32(b.Width))
	out = put32(out, uint32(b.Height))
	out = put32(out, count)
	out = append(out, byte(mode))
	out = put32(out, uint32(len(grid)))
	out = append(out, grid...)
	// The two newer planes belong to the VERSION being written, not to this
	// builder: the historical form below is built here too, and giving it planes
	// would make it a form no version ever had.
	if version >= 13 {
		for range grid {
			out = append(out, defaultCost)
		}
		out = append(out, make([]byte, len(grid))...)
	}

	for _, e := range ents {
		out = put32(out, uint32(e.ID))
		out = put32(out, uint32(e.X))
		out = put32(out, uint32(e.Y))
		out = put32(out, uint32(e.TargetX))
		out = put32(out, uint32(e.TargetY))
		out = put32(out, uint32(e.Class))
		if e.HasTarget {
			out = append(out, 1)
		} else {
			out = append(out, 0)
		}
		out = append(out, e.Stall)
		out = put32(out, uint32(e.HP))
		out = put32(out, uint32(e.MaxHP))
		out = append(out, byte(e.Domain))
		out = put32(out, uint32(e.Speed))
		out = append(out, byte(e.Transit), byte(e.Transit>>8))
		out = append(out, byte(e.TransitTotal), byte(e.TransitTotal>>8))
		out = append(out, e.GroupSpeed)
		out = put32(out, uint32(e.AttackTarget))
		if e.HasAttackTarget {
			out = append(out, 1)
		} else {
			out = append(out, 0)
		}
		out = append(out, byte(e.AttackPhase))
		out = put32(out, uint32(e.AttackCountdown))
		out = put32(out, uint32(e.AttackCharge))
		out = put32(out, uint32(e.AttackRelax))
		out = put32(out, uint32(e.ToHit))
		out = put32(out, uint32(e.Defence))
		out = put32(out, uint32(e.Absorption))
		out = put32(out, uint32(e.DamageBase))
		out = put32(out, uint32(e.DamageSpread))
		if e.AlwaysHits {
			out = append(out, 1)
		} else {
			out = append(out, 0)
		}
		out = put32(out, e.Group)
		out = put32(out, e.Owner)
		out = append(out, e.Facing)
		// The decay block belongs to the VERSION being written, exactly as the
		// two newer planes above do: the historical form below is built here too,
		// and giving it a ladder would make it a form no version ever had.
		if version >= 17 {
			out = append(out, byte(e.Decay))
			out = append(out, byte(e.Dwell), byte(e.Dwell>>8))
			out = put32(out, uint32(e.DyingTime))
		}
		// And the sight range belongs to ITS version, behind the ladder, for the
		// reason the ladder belongs to 17: a historical form built here with a
		// range would be a form no version ever had.
		if version >= 18 {
			out = append(out, e.ScanRange)
		}
		// And the actor state, the ring and the leg belong to THIS version's own
		// tail (0099): every entity this file builds is handed through NewWorld,
		// whose constructor writes guard into every one of them UNCONDITIONALLY
		// — so this builder writes the same six bytes regardless of what e
		// itself carries, on the rule that a historical form built here with any
		// of this would be a form no version before 21 ever had.
		if version >= 21 {
			out = append(out, actorStateGuard)
			out = put32(out, 0)
			out = put32(out, 0)
			out = put32(out, 0)
			out = put32(out, 0)
			out = append(out, patrolLegHead)
		}
		// And THE REACH belongs to ITS version, at the record's very tail
		// (0104): no fixture this builder is ever given names one, so the
		// constructor's own floor of 1 is the one byte here regardless of
		// what e itself carries — a historical form built here with one
		// would be a form no version before 23 ever had.
		if version >= 23 {
			out = append(out, 1)
		}
		// And THE POST belongs to ITS version, at the record's very tail (0106):
		// every entity this file builds is handed through NewWorld, whose
		// constructor writes PostX, PostY from the entity's own X, Y
		// UNCONDITIONALLY — so this builder writes e's own X, Y here regardless
		// of what e.PostX/e.PostY themselves carry, on the reach byte's own rule:
		// a historical form built here with any of this would be a form no version
		// before 24 ever had.
		if version >= 24 {
			out = put32(out, uint32(e.X))
			out = put32(out, uint32(e.Y))
		}
		// And THE REGENERATION BLOCK belongs to ITS version, at the
		// record's very tail (0109): the constructor writes none of the
		// six unconditionally and only folds an out-of-range remainder,
		// which no fixture this builder is given ever carries, so e's own
		// six fields are exactly what the constructor leaves them at — a
		// historical form built here with any of this would be a form no
		// version before 25 ever had.
		if version >= 25 {
			out = put32(out, uint32(e.Mana))
			out = put32(out, uint32(e.MaxMana))
			out = put32(out, uint32(e.HealthRegenPeriod))
			out = put32(out, uint32(e.ManaRegenPeriod))
			out = append(out, e.HealthHundredths)
			out = append(out, e.ManaHundredths)
		}
		// And THE COMMAND GROUP belongs to ITS version, at the record's very
		// tail (0117): no fixture this builder is ever given carries one —
		// this task gives the field no writer at all — so e's own value is
		// exactly what the zero value already leaves it at, on the
		// regeneration block's own rule: a historical form built here with
		// one would be a form no version before 32 ever had — 0117's own
		// landed state, after 0122's player references took 31.
		if version >= 32 {
			out = put32(out, e.CommandGroup)
		}
		// And AN ENTITY'S EXPERIENCE FROM USE belongs to ITS version, at the
		// record's very tail (0125): no fixture this builder is ever given
		// names an experience, a Mind or a gaining class, so e's own six
		// slots, its Mind, its experience value, its credited slot and its
		// gains flag are exactly what the zero value already leaves them
		// at, on the command group's own rule: a historical form built here
		// with any of this would be a form no version before 35 ever had.
		if version >= 35 {
			for _, xp := range e.SkillXP {
				out = put32(out, uint32(xp))
			}
			out = put32(out, uint32(e.Mind))
			out = put32(out, uint32(e.XPValue))
			out = append(out, e.XPSlot)
			if e.GainsXP {
				out = append(out, 1)
			} else {
				out = append(out, 0)
			}
		}
		// And AN ENTITY'S OWN SPELLBOOK belongs to ITS version, at the
		// record's very tail (0127 FR-4b): no fixture this builder is ever
		// given names a KnownSpells mask, so e's own value is exactly what
		// the zero value already leaves it at, on the experience block's
		// own rule: a historical form built here with one would be a form
		// no version before 36 ever had.
		if version >= 36 {
			out = put32(out, e.KnownSpells)
		}
		// And THE SIX SKILL LEVELS belong to ITS version, at the record's very
		// tail: no fixture this builder is ever given names one, so e's own six
		// slots are exactly what the zero value already leaves them at, on the
		// known-spells mask's own rule: a historical form built here with any of
		// this would be a form no version before 39 ever had.
		if version >= 39 {
			for _, lvl := range e.Skill {
				out = put32(out, uint32(lvl))
			}
		}
		// And A WEAPON'S OWN SPELL belongs to ITS version, at the record's very
		// tail: no fixture this builder is ever given carries a weapon or a spell,
		// so e's own zero values are exactly what the zero value already leaves
		// them at, on the skill block's own rule: a historical form built here
		// with either would be a form no version before 41 ever had.
		if version >= 41 {
			out = append(out, byte(e.WeaponSpell), byte(e.WeaponSpell>>8))
			out = put32(out, uint32(e.WeaponSpellLevel))
		}
		// And THE AUTOCAST PAIR AND THE SPELL EFFECT MARK belong to ITS version,
		// at the record's very tail: no fixture this builder is ever given carries
		// an autocast or a mark, so e's own zero values are exactly what the zero
		// value already leaves them at, on the weapon spell's own rule: a
		// historical form built here with any of them would be a form no version
		// before 45 ever had.
		if version >= 45 {
			out = append(out, byte(e.AutoSpell), byte(e.AutoSpell>>8))
			out = append(out, e.CastWait)
			// THE MARK'S REMAINING TICKS WIDENED TO A WORD AT VERSION 50: one byte
			// before it, two from it on. No fixture this builder is ever given
			// carries a mark, so both arms write the same zeros; the arms are written
			// out because the RECORD'S WIDTH differs between them and a decoder
			// reading the wrong one would start every later field one byte out.
			if version >= 50 {
				out = append(out, byte(e.SpellFX), byte(e.SpellFX>>8))
			} else {
				out = append(out, byte(e.SpellFX))
			}
			out = append(out, e.SpellFXSpell)
		}
		// And THE OFF-MAP BIT belongs to ITS version, at the record's very tail:
		// no fixture this builder is ever given has been taken off the map by a
		// script, and nothing but a script arm can set the bit, so e's own zero
		// value is what is written here — on the autocast pair's own rule: a
		// historical form built here with a set bit would be a form no version
		// before 48 ever had.
		if version >= 48 {
			if e.OffMap {
				out = append(out, 1)
			} else {
				out = append(out, 0)
			}
		}
		// And THE ESCORT TRIPLE belongs to ITS version, at the record's very tail
		// (version 50): no fixture this builder is ever given holds an escort
		// order, and nothing but a script group sub-command can write one, so e's
		// own zero values are what is written here — on the off-map bit's own
		// rule directly above.
		if version >= 50 {
			out = put32(out, uint32(e.EscortTarget))
			if e.HasEscortTarget {
				out = append(out, 1)
			} else {
				out = append(out, 0)
			}
			out = append(out, e.EscortRange)
		}
		if version >= 53 {
			for _, p := range e.Protection {
				out = put32(out, uint32(p))
			}
			out = append(out, e.TokenSize)
			out = append(out, e.SeeInvisible)
			out = put32(out, uint32(e.Reaction))
			out = put32(out, uint32(e.Spirit))
		}
		if version >= 54 {
			if e.SuppressCorpseLoot {
				out = append(out, 1)
			} else {
				out = append(out, 0)
			}
		}
		if version >= 55 {
			out = put32(out, uint32(e.KillCreditSource))
			if e.HasKillCredit {
				out = append(out, 1)
			} else {
				out = append(out, 0)
			}
			out = append(out, byte(e.KillCreditSpell))
		}
		if version >= 56 {
			out = put32(out, uint32(e.Load))
			out = put32(out, uint32(e.Capacity))
		}
		if version >= 57 {
			out = append(out, byte(e.MapUnitID), byte(e.MapUnitID>>8))
		}
		if version >= 59 {
			out = append(out, e.Resistance[:]...)
		}
		if version >= 60 {
			out = put32(out, uint32(e.Withdraw))
			out = put32(out, uint32(e.Wimpy))
		}
		if version >= 62 {
			if e.Humanoid {
				out = append(out, 1)
			} else {
				out = append(out, 0)
			}
		}
		if version >= 63 {
			desired := e.DesiredFacing
			if e.TurnRemaining == 0 {
				desired = e.Facing
			}
			out = append(out, desired, e.TurnRemaining)
		}
		// Version 64 appends the request-time duration. Inactive fixtures carry
		// its canonical zero.
		if version >= 64 {
			out = append(out, e.TurnTotal)
		}
		if version >= 68 {
			for _, n := range e.PotionStats {
				out = put32(out, uint32(n))
			}
			for _, n := range e.PotionHeadroom {
				out = put32(out, uint32(n))
			}
		}
		if version >= 69 {
			out = append(out, make([]byte, 15)...)
		}
		if version >= 71 {
			out = append(out, make([]byte, 113)...)
		}
		if version >= 72 {
			out = append(out, e.SecondBase, e.SecondSpread)
		}
		if version >= 73 {
			out = append(out, byte(e.CurrentProfileBasis))
		}
		if version >= 76 {
			out = append(out, make([]byte, 35)...)
		}
	}
	for _, r := range routes {
		out = put32(out, uint32(len(r)))
		for _, c := range r {
			out = put32(out, uint32(c.x))
			out = put32(out, uint32(c.y))
		}
	}
	// The group section belongs to THIS version, between the routes and the
	// script section — a historical form built here with one would be a form
	// no version before 19 ever had.
	if version >= 19 {
		out = append(out, rtfGroupSection(ents)...)
	}
	// And THE SACK SECTION belongs to ITS version, right after the group
	// section (0103): no fixture this builder is ever given carries a sack,
	// so it is always the section's own bare four-byte zero count — a
	// historical form built here with one would be a form no version before
	// 22 ever had.
	if version >= 22 {
		out = append(out, make([]byte, sackCountLen)...)
	}
	// And THE CARRY SECTION AND THE PURSE belong to THEIR version, right
	// after the sack section (0112): no fixture this builder is ever given
	// carries an item or credits any gold, so the carry section is one bare
	// four-byte zero count per entity — no section count of its own — and
	// the purse section is purseLen zeros — a historical form built here
	// with either would be a form no version before 26 ever had.
	if version >= 26 {
		for range ents {
			out = put32(out, 0)
		}
	}
	// And THE EQUIPMENT SECTION belongs to ITS version, right after the
	// carry section and before the purse (0124 T2): no fixture this builder
	// is ever given equips anything, so it is one bare EquipSlots-wide zero
	// record per entity, no count of its own — a historical form built here
	// with one would be a form no version before 34 ever had.
	if version >= 34 {
		out = append(out, make([]byte, equipRecordLen*len(ents))...)
	}
	if version >= 44 {
		out = append(out, make([]byte, treasureRecordLen*len(ents))...)
	}
	if version >= 26 {
		out = append(out, make([]byte, purseLen)...)
	}
	// And THE SPELL TABLE belongs to ITS version, right after the purse section
	// and before the script section (version 36): no fixture this builder is
	// ever given names a table, so it is always the section's own bare two-byte
	// zero count — a historical form built here with one would be a form no
	// version before 36 ever had.
	if version >= 36 {
		out = append(out, make([]byte, spellCountLen)...)
	}
	// And THE ITEM-WEIGHT SECTION belongs to ITS version, right after the spell
	// table and before the casting section (version 56): no fixture this
	// builder is ever given declares an item weight, so it is always the
	// section's own bare two-byte zero count, on the spell table's own rule
	// above.
	if version >= 56 {
		out = append(out, make([]byte, itemWeightCountLen)...)
	}
	// And THE CASTING SECTION belongs to ITS version, right after the
	// item-weight section and before the script section (version 49): no
	// fixture this builder is ever given holds a pending cast or an area
	// effect, so it is always the section's own two bare zero counts, on the
	// spell table's own rule above.
	if version >= 49 {
		out = append(out, make([]byte, 2*castingCountLen)...)
	}
	// And THE SCRIPT-STATE SECTION belongs to ITS version, right after the
	// casting section and before the script section (version 50). No fixture
	// this builder is ever given carries a cell tail, so the tail half is the
	// section's own bare four-byte zero count; the formation block is
	// relationSlots bytes AT THE DEFAULT, which is not zero — a block of
	// zeros would be a world every player of which is never in formation, and
	// that is a state no constructor produces.
	if version >= 50 {
		modes := make([]byte, relationSlots)
		for i := range modes {
			modes[i] = formationDefault
		}
		out = append(out, modes...)
		out = append(out, make([]byte, tailCountLen)...)
	}
	// And THE STRUCTURE SECTION belongs to ITS version, right after the
	// script-state section and before the script section (version 58, 1033
	// B3): no fixture this builder is ever given declares a structure, so it
	// is always the section's own bare four-byte zero count, on the spell
	// table's own rule above.
	if version >= 58 {
		out = append(out, make([]byte, structureCountLen)...)
	}
	// Format 61 appends canonical sack, carried, equipment, provenance and
	// item-derived state after the structure section. These fixtures carry no
	// instances, so the independent empty transcription is exact.
	if version >= 61 {
		out = append(out, emptyItemStatePin(int(count))...)
	}
	if version >= 68 {
		out = put32(out, 0)
	}
	// The script section, which for a world running none is the fixed volatile
	// half and three zero counts, and then the relation, which for a world where
	// nobody is hostile to anybody is relationLen zeros.
	out = append(out, rtfScriptSection()...)
	if version >= 70 {
		out = put32(out, 0)
	}
	if version >= 74 {
		out = put32(out, 0)
	}
	if version >= 75 {
		out = put32(out, 0)
	}
	out = append(out, make([]byte, relationLen)...)
	if version >= 77 {
		out = append(out, 0, 0, 0, 0, 0)
	}
	if version >= 78 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 79 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 80 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 81 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 82 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 83 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 84 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 85 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 86 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 87 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 88 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 89 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 90 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 91 {
		out = append(out, 0, 0, 0, 0)
	}
	if version >= 93 {
		out = append(out, 0, 0, 0, 0)
	}
	// THE ENTITY ID FLOOR (form94) closes the form, outside every section
	// above: the highest id among the entities this builder was given, plus
	// one — the same value a freshly built world with no removals carries,
	// computed here rather than passed in so this builder stays independent
	// of whatever floor bookkeeping the production encoder does.
	if version >= 94 {
		var floor uint64
		for _, e := range ents {
			if id := uint64(e.ID) + 1; id > floor {
				floor = id
			}
		}
		for i := 0; i < 8; i++ {
			out = append(out, byte(floor>>(8*i)))
		}
	}
	if version >= 95 {
		out = append(out, 0, 0, 0, 0)
	}
	return out
}

// rtfNotAlive is rtfEnts with the FIRST unit — the one that holds the pinned
// route — put at a health pair that is not alive, and its target either kept or
// struck off. The two shapes are not interchangeable and that is why both are
// built: a unit that is not alive and holds a target is refused before its route
// is ever read, and one that holds no target is refused by the route section's
// own rule. Between them they are the whole of "a route on a unit that is not
// alive", and each says which refusal catches it.
func rtfNotAlive(hp, maxHP int32, keepTarget bool) []Entity {
	ents := rtfEnts()
	ents[0].HP, ents[0].MaxHP = hp, maxHP
	ents[0].HasTarget = keepTarget
	return ents
}

// rtfValid is the well-formed form the refusals below are spoiled copies of,
// built by the second implementation.
func rtfValid(routes ...[]cell) []byte {
	all := make([][]cell, 3)
	copy(all, routes)
	return rtfBuild(formatVersion, 7, 0x1234, rtfBounds, ModeCanonical, rtfGrid(), 3, rtfEnts(), all)
}

// ------------------------------------------------- the pin

func TestARoutedWorldMarshalsToItsPinnedBytes(t *testing.T) {
	if want := 61 + 34 + 3*16 + relationLen + 3*492 + (4 + 2*8) + 4 + 4 + (4 + 2*18) + sackCountLen + 3*carryCountLen + 3*equipRecordLen + 3*treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(3)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(rtfBytes) != want {
		t.Fatalf("the pin is %d bytes; this world's form is %d", len(rtfBytes), want)
	}
	got, err := rtfWorld(t, rtfRoute()).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(got, rtfBytes) {
		t.Errorf("the routed world marshals to\n % x\npinned as\n % x", got, rtfBytes)
	}
	// The builder the refusals are made with agrees with the transcription, so a
	// spoiled copy differs from a known-good form in one way and not two.
	if built := rtfValid(rtfRoute()); !bytes.Equal(built, rtfBytes) {
		t.Errorf("the second implementation writes\n % x\npinned as\n % x", built, rtfBytes)
	}
}

func TestThePinnedRoutedDigestIsFNV1aOfThePinnedBytes(t *testing.T) {
	requireOriginalDeadLegacyDigest(t, rtfWorld(t, rtfRoute()), 0xc03ff985db44e03d)
	requireHumanMovementLegacyDigest(t, rtfWorld(t, rtfRoute()), 0xa1e77f5301b7df02)
	if got := fnv1a(rtfBytes); got != rtfDigest {
		t.Errorf("FNV-1a of the pinned bytes is %#016x, pinned as %#016x", got, rtfDigest)
	}
	if got := rtfWorld(t, rtfRoute()).Hash(); got != rtfDigest {
		t.Errorf("the routed world hashes %#016x, pinned as %#016x", got, rtfDigest)
	}
}

func TestThePinnedRoutedBytesDecodeBackToTheRoutedWorld(t *testing.T) {
	var got World
	if err := got.UnmarshalBinary(rtfBytes); err != nil {
		t.Fatalf("UnmarshalBinary(rtfBytes): %v", err)
	}
	if diff, want := snap(&got), snap(rtfWorld(t, rtfRoute())); !equalState(diff, want) {
		t.Errorf("the pinned bytes decode to\n %+v\nwant\n %+v", diff, want)
	}
}

// TestAWorldWithNoRouteAndOneWhoseRoutesAreEmptyAreOneWorld: a nil route slot
// and a zero-length one are the same state, so they are the same bytes. There is
// no absent case in the route section any more than there is in the grid.
func TestAWorldWithNoRouteAndOneWhoseRoutesAreEmptyAreOneWorld(t *testing.T) {
	nils := rtfWorld(t)
	empties := rtfWorld(t, []cell{}, []cell{}, []cell{})

	a, err := nils.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	b, err := empties.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("a world of nil routes and one of empty routes marshal to\n % x\n % x", a, b)
	}
	if nils.Hash() != empties.Hash() {
		t.Errorf("digests %#016x and %#016x", nils.Hash(), empties.Hash())
	}
	// And the form really does carry the section: three counts of zero.
	if want := 61 + 34 + 3*16 + relationLen + 3*492 + 3*4 + (4 + 2*18) + sackCountLen + 3*carryCountLen + 3*equipRecordLen + 3*treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(3)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(a) != want {
		t.Errorf("the routeless form is %d bytes, want %d", len(a), want)
	}
}

// ------------------------------------------------- AC-5

// TestARouteIsCanonicalStateInFormAndDigest is AC-5. Four worlds alike in every
// other field: one holding the pinned route, one differing in a single CELL of
// it, one differing in its LENGTH, and one holding no route at all. All four
// differ pairwise in form and in digest, and the one that is marshalled and read
// back is equal to itself in both.
func TestARouteIsCanonicalStateInFormAndDigest(t *testing.T) {
	worlds := []struct {
		what  string
		route []cell
	}{
		{"the pinned route", rtfRoute()},
		{"one cell of it moved", []cell{{2, 1}, {2, 2}}},
		{"one cell longer", []cell{{0, 1}, {1, 1}, {2, 2}}},
		{"no route at all", nil},
	}

	forms := make([][]byte, len(worlds))
	digests := make([]uint64, len(worlds))
	for i, tc := range worlds {
		w := rtfWorld(t, tc.route)
		f, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: MarshalBinary: %v", tc.what, err)
		}
		forms[i], digests[i] = f, w.Hash()
	}
	for i := range worlds {
		for j := i + 1; j < len(worlds); j++ {
			if bytes.Equal(forms[i], forms[j]) {
				t.Errorf("%s and %s marshal alike", worlds[i].what, worlds[j].what)
			}
			if digests[i] == digests[j] {
				t.Errorf("%s and %s both hash %#016x", worlds[i].what, worlds[j].what, digests[i])
			}
		}
	}

	var back World
	if err := back.UnmarshalBinary(forms[0]); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(again, forms[0]) {
		t.Errorf("the round trip re-marshals to\n % x\nwant\n % x", again, forms[0])
	}
	if back.Hash() != digests[0] {
		t.Errorf("the round trip hashes %#016x, the original %#016x", back.Hash(), digests[0])
	}
}

// ------------------------------------------------- AC-6

// TestUnmarshalRefusesAMalformedRouteAndLeavesTheReceiverAsItWas is AC-6. Each
// case is the valid form with exactly one thing wrong with it, and each is
// checked twice over: the decode fails, and the receiver's whole byte form is
// what it was — a route half-written into a receiver moves bytes a digest could
// only say something about.
func TestUnmarshalRefusesAMalformedRouteAndLeavesTheReceiverAsItWas(t *testing.T) {
	valid := rtfValid(rtfRoute())

	// A version-4 form of this same world: the same header with the shipped
	// version byte, the same grid, the same three records with the eight health
	// bytes struck off the tail of each, and the same route section after them.
	// It is exactly what the previous encoder wrote for this world, and it is
	// well formed for version 4 in every particular — which is why it is refused
	// rather than read as a world whose units have no health.
	//
	// It is DERIVED from the valid form rather than transcribed, like every other
	// case in this table: what each case owes is one deliberate difference from
	// something known good, and the hand-written older streams live in
	// binary_test.go where nothing derives them at all.
	v4 := append([]byte(nil), valid[:34+16]...)
	for i := 0; i < 3; i++ {
		rec := 34 + 3*16 + relationLen + 99*i
		v4 = append(v4, valid[rec:rec+26]...)
	}
	// The route section only — a version-4 form has no script section and no
	// group section, so the tail is cut before both. The bound is the records'
	// own end plus the route section's own length, absolute and independent of
	// len(valid): 0095 sits its own section between the routes and the script,
	// so len(valid)-(scriptStateLen+scriptCountsLen) no longer lands on the
	// routes' own end, and this is what still does.
	v4 = append(v4, valid[34+3*16+3*492:34+3*16+3*492+(4+2*8)+4+4]...)
	v4[0] = 4
	if want := 34 + 16 + 3*26 + (4 + 2*8) + 4 + 4; len(v4) != want {
		t.Fatalf("the version-4 form is %d bytes, want %d", len(v4), want)
	}

	cases := []struct {
		name string
		data []byte
	}{
		{"a route on a unit with no target",
			rtfValid(rtfRoute(), nil, []cell{{3, 2}, {3, 3}})},
		{"a route on a unit killed below zero, its target still held",
			rtfBuild(formatVersion, 7, 0x1234, rtfBounds, ModeCanonical, rtfGrid(), 3,
				rtfNotAlive(-1, 100, true), [][]cell{rtfRoute(), nil, nil})},
		{"a route on a unit downed at zero, holding no target",
			rtfBuild(formatVersion, 7, 0x1234, rtfBounds, ModeCanonical, rtfGrid(), 3,
				rtfNotAlive(0, 100, false), [][]cell{rtfRoute(), nil, nil})},
		{"a route ending anywhere but on its unit's target",
			rtfValid([]cell{{1, 1}, {2, 1}})},
		{"a route cell outside the bounds",
			rtfValid(nil, []cell{{4, 3}, {3, 3}})},
		{"two route cells that are not neighbours",
			rtfValid([]cell{{0, 0}, {2, 2}})},
		{"two route cells that are the same cell",
			rtfValid([]cell{{1, 1}, {1, 1}, {2, 2}})},
		{"a route section one byte long",
			append(append([]byte(nil), valid[:34+3*16+3*492]...), 0x00)},
		{"a route count that overruns the form",
			rtfBuild(formatVersion, 7, 0x1234, rtfBounds, ModeCanonical, rtfGrid(), 3, rtfEnts(),
				[][]cell{{{1, 1}, {2, 2}}, nil, nil})[:len(valid)-8]},
		{"one byte after the last route",
			append(append([]byte(nil), valid...), 0x00)},
		{"the previous version, well formed for itself", v4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := rtfWorld(t, rtfRoute())
			before := snap(w)
			beforeForm, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			if err := w.UnmarshalBinary(tc.data); err == nil {
				t.Fatalf("accepted % x", tc.data)
			}
			if got := snap(w); !equalState(got, before) {
				t.Errorf("the receiver changed:\n before %+v\n after  %+v", before, got)
			}
			afterForm, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			if !bytes.Equal(afterForm, beforeForm) {
				t.Errorf("the receiver's byte form changed under a refusal")
			}
		})
	}

	// The ones that must NOT be refused, so the table above is not passing by
	// refusing everything: the valid form itself, and a unit holding a target
	// with no route — the state every world begins in.
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"the valid form", valid},
		{"a target with no route", rtfValid()},
	} {
		var w World
		if err := w.UnmarshalBinary(tc.data); err != nil {
			t.Errorf("%s was refused: %v", tc.name, err)
		}
	}
}

// TestADecodedRouteIsTheRouteThatWasWritten reads the cells back one at a time.
// The refusals above all end in an error, so none of them says the accepted
// route arrived intact — a decoder that validated a route and then stored a
// different one would pass every case there.
func TestADecodedRouteIsTheRouteThatWasWritten(t *testing.T) {
	var w World
	if err := w.UnmarshalBinary(rtfBytes); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if len(w.routes) != 3 {
		t.Fatalf("the decoded world holds %d route slot(s), want 3", len(w.routes))
	}
	want := rtfRoute()
	if len(w.routes[0]) != len(want) {
		t.Fatalf("unit 2's route is %s, want %s", fmtRoute(w.routes[0]), fmtRoute(want))
	}
	for i := range want {
		if w.routes[0][i] != want[i] {
			t.Fatalf("unit 2's route is %s, want %s", fmtRoute(w.routes[0]), fmtRoute(want))
		}
	}
	for i := 1; i < 3; i++ {
		if len(w.routes[i]) != 0 {
			t.Errorf("unit %d decoded holding the route %s", w.entities[i].ID, fmtRoute(w.routes[i]))
		}
	}

	// And the decoded world keeps nothing that reaches back into the buffer.
	form := append([]byte(nil), rtfBytes...)
	var second World
	if err := second.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	before := snap(&second)
	for i := range form {
		form[i] ^= 0xff
	}
	if got := snap(&second); !equalState(got, before) {
		t.Errorf("mutating the source bytes reached the decoded routes:\n %+v\n %+v", got, before)
	}
}
