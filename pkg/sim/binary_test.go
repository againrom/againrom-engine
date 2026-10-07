package sim

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Nothing in this file may ask the encoder where it put a field, how wide a
// record is, or what the form of a given world comes to. Offsets, widths and
// byte values are written out by hand from the contract; a test that derived
// them would assert the encoder against itself and pass whatever it did. The
// production constants headerLen and entityLen are deliberately not used here
// either, for the same reason: 34 and 100 are spelled out, and so are the grid
// section between them and the route section after them — 4 bytes of cell count
// per entity, and 8 per cell. The header and the record were the same width until
// the domain byte widened the record, which is why both are written as
// literals at every site, so a record that widens again moves one of them alone.

// ---------------------------------------------------------------- offsets

// offsetWorld carries a distinct byte pattern in every field, so that two fields
// swapped, one widened, or the whole body shifted by the version byte all land
// somewhere this file names.
//
// The height is NEGATIVE, which does two things at once: it pins the sign of a
// bounds field, which two positive extents could not, and it leaves the world no
// in-bounds cell — so a grid over these bounds is empty and this stays a fixture
// about offsets rather than one that has to carry W*H cell bytes for an extent
// chosen to be unmistakable rather than small.
//
// The first unit's health pair is positive, so it stays ALIVE and keeps the
// target and the stall count this table pins; the second's is negative on both
// fields, which pins the sign of each and makes that unit dead — it holds no
// target, so nothing of it is cleared.
//
// The two domains are the two NON-ZERO ones, and they differ from each other. A
// domain written as the zero value would be indistinguishable from a byte the
// encoder never wrote, and two equal ones would not catch a record's own domain
// being read out of its neighbour.
func offsetWorld(t *testing.T) *World {
	t.Helper()
	w := mustWorldGrid(t, 0x1122334455667788, Bounds{Width: 0x01020304, Height: -0x05060708},
		ModeOptimised, nil, []Entity{
			{ID: 0x0a0b0c0d, X: 0x11121314, Y: -1, TargetX: 0x21222324, TargetY: -2, Class: 0x31323334, HasTarget: true, Stall: 0x0b,
				HP: 0x41424344, MaxHP: 0x51525354, Domain: DomainAir,
				Speed: 0x61626364, Transit: 0x00ff, TransitTotal: maxTransit, GroupSpeed: 0x81,
				AttackTarget: 0x7f000000, HasAttackTarget: true,
				AttackPhase: AttackCharging, AttackCountdown: 0x00000102,
				AttackCharge: 0x00010203, AttackRelax: 0x00040506,
				ToHit: 0x00070809, Defence: 0x0a0b0c0d, Absorption: -1,
				DamageBase: 0x0e0f1011, DamageSpread: 0x12131415, AlwaysHits: true,
				Group: 0xa1a2a3a4, Owner: 0xb1b2b3b4, Facing: 0xc5,
				DyingTime: 0x76757473, ScanRange: 0xd7,
				KillCreditSource: 0x7f000000, HasKillCredit: true, KillCreditSpell: -7,
				MapUnitID: 0xe3e4, Resistance: [5]uint8{0xa6, 0xa7, 0xa8, 0xa9, 0xaa},
				Withdraw: 0x35363738, Wimpy: -0x01020304, Humanoid: true},
			{ID: 0x7f000000, X: 2, Y: -3, Class: -4, HasTarget: false, HP: -1, MaxHP: -0x02030405,
				Domain: DomainGhost, Speed: 0x71727374, GroupSpeed: 0x91,
				AttackTarget: 0x0a0b0c0d, HasAttackTarget: true,
				AttackPhase: AttackCharging, AttackCountdown: 3,
				AttackCharge: 0x16171819, AttackRelax: -2,
				ToHit: 0x1a1b1c1d, Defence: 0, Absorption: 0x1e1f2021,
				DamageBase: 0x22232425, DamageSpread: 0x26272829,
				Group: 0x0000ffff, Owner: 0x0000fffe, Facing: 0x2a,
				DyingTime: 0x0105, ScanRange: 0x39, KillCreditSpell: 0x2a,
				MapUnitID: 0x00f5, Resistance: [5]uint8{0xb6, 0xb7, 0xb8, 0xb9, 0xba},
				Withdraw: -0x11121314, Wimpy: 0x45464748},
		})
	w.tick = 0x0102030405060708
	return w
}

// offsetCases is the contract's table, transcribed. It is a partition of the
// whole byte form — checked below — so a field added to the encoding is a
// failure here and not a silent widening.
var offsetCases = []struct {
	field string
	off   int
	width int
	want  []byte
}{
	{"format version", 0, 1, []byte{0x32}},
	{"tick", 1, 8, []byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}},
	{"rng state", 9, 8, []byte{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}},
	{"bounds.Width", 17, 4, []byte{0x04, 0x03, 0x02, 0x01}},
	{"bounds.Height", 21, 4, []byte{0xf8, 0xf8, 0xf9, 0xfa}},
	{"entity count", 25, 4, []byte{0x02, 0x00, 0x00, 0x00}},
	{"routing mode", 29, 1, []byte{0x01}},
	{"grid cell count", 30, 4, []byte{0x00, 0x00, 0x00, 0x00}},

	{"entity 0 ID", 34, 4, []byte{0x0d, 0x0c, 0x0b, 0x0a}},
	{"entity 0 X", 38, 4, []byte{0x14, 0x13, 0x12, 0x11}},
	{"entity 0 Y", 42, 4, []byte{0xff, 0xff, 0xff, 0xff}},
	{"entity 0 TargetX", 46, 4, []byte{0x24, 0x23, 0x22, 0x21}},
	{"entity 0 TargetY", 50, 4, []byte{0xfe, 0xff, 0xff, 0xff}},
	{"entity 0 Class", 54, 4, []byte{0x34, 0x33, 0x32, 0x31}},
	{"entity 0 HasTarget", 58, 1, []byte{0x01}},
	{"entity 0 stall count", 59, 1, []byte{0x0b}},
	{"entity 0 HP", 60, 4, []byte{0x44, 0x43, 0x42, 0x41}},
	{"entity 0 MaxHP", 64, 4, []byte{0x54, 0x53, 0x52, 0x51}},
	{"entity 0 movement domain", 68, 1, []byte{0x02}},
	{"entity 0 Speed", 69, 4, []byte{0x64, 0x63, 0x62, 0x61}},
	{"entity 0 transit owed", 73, 2, []byte{0xff, 0x00}},
	{"entity 0 transit length", 75, 2, []byte{0x00, 0x01}},
	{"entity 0 group speed", 77, 1, []byte{0x81}},
	{"entity 0 attack victim", 78, 4, []byte{0x00, 0x00, 0x00, 0x7f}},
	{"entity 0 attack presence", 82, 1, []byte{0x01}},
	{"entity 0 attack phase", 83, 1, []byte{0x01}},
	{"entity 0 attack countdown", 84, 4, []byte{0x02, 0x01, 0x00, 0x00}},
	{"entity 0 attack charge", 88, 4, []byte{0x03, 0x02, 0x01, 0x00}},
	{"entity 0 attack relax", 92, 4, []byte{0x06, 0x05, 0x04, 0x00}},
	{"entity 0 to-hit", 96, 4, []byte{0x09, 0x08, 0x07, 0x00}},
	{"entity 0 defence", 100, 4, []byte{0x0d, 0x0c, 0x0b, 0x0a}},
	{"entity 0 absorption", 104, 4, []byte{0xff, 0xff, 0xff, 0xff}},
	{"entity 0 damage base", 108, 4, []byte{0x11, 0x10, 0x0f, 0x0e}},
	{"entity 0 damage spread", 112, 4, []byte{0x15, 0x14, 0x13, 0x12}},
	{"entity 0 always hits", 116, 1, []byte{0x01}},
	// The record's version-11 tail, and then version 12's after it. The owner is
	// the last field of the record, so a field appended after it would break
	// this table's partition rather than slide in unnoticed.
	{"entity 0 group", 117, 4, []byte{0xa4, 0xa3, 0xa2, 0xa1}},
	{"entity 0 owner", 121, 4, []byte{0xb4, 0xb3, 0xb2, 0xb1}},
	// AND VERSION 14'S AFTER IT. The facing is the last field of the record, so a
	// field appended after it breaks this table's partition rather than sliding
	// in unnoticed.
	//
	// NEITHER FACING HERE IS A MULTIPLE OF 32, and that is the point of the two
	// values: this build writes only the eight directions, so a byte carried
	// whole and a byte rounded to a direction on the way in or out are
	// indistinguishable on any facing it produces. 0xc5 and 0x2a are the states a
	// decoder may be handed and must give back unaltered.
	{"entity 0 facing", 125, 1, []byte{0xc5}},
	// AND VERSION 17'S AFTER IT, the three the decay ladder adds. This unit is
	// ALIVE, so its stage and its dwell are the pair a living unit carries and
	// the constructor lets it carry no other; its dying time is a stat like its
	// speed and is carried whole whatever its life state.
	{"entity 0 decay stage", 126, 1, []byte{0x00}},
	{"entity 0 dwell", 127, 2, []byte{0x00, 0x00}},
	{"entity 0 dying time", 129, 4, []byte{0x73, 0x74, 0x75, 0x76}},
	// AND VERSION 18'S AFTER THAT, behind the whole decay block. The sight range
	// is the last field of the record now, and the sentence the facing carried
	// moves down to it: a field appended after this one breaks this table's
	// partition rather than sliding in unnoticed. The two values are distinct and
	// neither is a value the other record carries in any field, so a range read
	// out of the neighbouring record lands on a byte this table can name rather
	// than on an equal one.
	{"entity 0 sight range", 133, 1, []byte{0xd7}},
	// AND VERSION 21'S AFTER THAT (0099): the actor state, the ring and the
	// leg, the record's new tail. Both entities in this fixture are built by
	// NewWorld, which writes the state UNCONDITIONALLY at guard and leaves the
	// ring and the leg at their residue zero — a caller cannot hand this
	// constructor a patroller, so the pin's only witness that this tail decodes
	// a real one is AC-3, on a form this file does not build by construction.
	{"entity 0 actor state", 134, 1, []byte{0x0b}},
	{"entity 0 patrol head X", 135, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 patrol head Y", 139, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 patrol tail X", 143, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 patrol tail Y", 147, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 patrol leg", 151, 1, []byte{0x00}},
	// AND VERSION 23'S AFTER THAT (0104): the reach. This entity names none,
	// so the constructor's own floor is what the byte carries — a field
	// appended after this one breaks this table's partition rather than
	// sliding in unnoticed.
	{"entity 0 reach", 152, 1, []byte{0x01}},
	// AND VERSION 24'S AFTER THAT (0106): the post, the record's new last
	// field, carried whole. NewWorld writes it from the entity's own X, Y
	// unconditionally, so it reads back the same bytes entity 0's own X and Y
	// already pinned above — the fixture's own witness that the constructor
	// did not invent a second value.
	{"entity 0 post X", 153, 4, []byte{0x14, 0x13, 0x12, 0x11}},
	{"entity 0 post Y", 157, 4, []byte{0xff, 0xff, 0xff, 0xff}},
	// AND VERSION 25'S AFTER THAT (0109): the regeneration block, the
	// record's new last field. Neither entity in this fixture names a pool
	// or a period, so the constructor leaves all six at their zero value —
	// a caller cannot hand this constructor a mana pool either, exactly as
	// it cannot hand it a patroller above.
	{"entity 0 mana", 161, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 mana maximum", 165, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 health regen period", 169, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 mana regen period", 173, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 health remainder", 177, 1, []byte{0x00}},
	{"entity 0 mana remainder", 178, 1, []byte{0x00}},
	// AND VERSION 27'S AFTER THAT (0117): the command group, the
	// record's new last field. Neither entity in this fixture ever had
	// one written, so the byte carried here is the zero every actor
	// this task can build holds.
	{"entity 0 command group", 179, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	// AND VERSION 35'S AFTER THAT (0125): an entity's experience from use,
	// the record's new last field. Neither entity in this fixture names an
	// experience, a Mind or a class that gains, so every byte here is the
	// zero the constructor leaves it at — a caller cannot hand this
	// constructor an earned slot either, exactly as it cannot hand it a
	// patroller above.
	{"entity 0 skill xp 0", 183, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill xp 1", 187, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill xp 2", 191, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill xp 3", 195, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill xp 4", 199, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill xp 5", 203, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 mind", 207, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 xp value", 211, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 xp slot", 215, 1, []byte{0x00}},
	{"entity 0 gains xp", 216, 1, []byte{0x00}},
	// AND VERSION 36'S AFTER THAT (0127): the known-spells mask, the
	// record's new last field. Neither entity in this fixture is built
	// with one — offsetWorld's own Entity literals name no
	// KnownSpells — so the constructor's own zero value is what the
	// four bytes here carry, on the experience block's own reason
	// above.
	{"entity 0 known spells", 217, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	// AND VERSION 39'S AFTER THAT (0135): the six skill levels. Neither
	// entity in this fixture is built with one — offsetWorld's own Entity
	// literals name no Skill — so the constructor's own zero value is
	// what the twenty-four bytes here carry, on the known-spells mask's
	// own reason above.
	{"entity 0 skill 0", 221, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill 1", 225, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill 2", 229, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill 3", 233, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill 4", 237, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 skill 5", 241, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	// AND VERSION 41'S AFTER THAT (0139): a weapon's own spell, the
	// record's new last field. Neither entity in this fixture carries a
	// WeaponSpell or a WeaponSpellLevel — offsetWorld's own Entity
	// literals name neither — so the constructor's own zero value is what
	// these six bytes carry, on the skill block's own reason above.
	{"entity 0 weapon spell", 245, 2, []byte{0x00, 0x00}},
	{"entity 0 weapon spell level", 247, 4, []byte{0x00, 0x00, 0x00, 0x00}},

	// AND VERSION 45'S AFTER THAT (0154): the autocast pair and the spell
	// effect mark, the record's new last fields. Neither entity in this
	// fixture carries an autocast or a mark — offsetWorld's own Entity
	// literals name none — so the constructor's own zero value is what
	// these five bytes carry, on the weapon spell's own reason above.
	{"entity 0 autocast spell", 251, 2, []byte{0x00, 0x00}},
	{"entity 0 autocast wait", 253, 1, []byte{0x00}},
	// THE MARK'S TICKS ARE A WORD SINCE VERSION 50.
	{"entity 0 spell effect ticks", 254, 2, []byte{0x00, 0x00}},
	{"entity 0 spell effect spell", 256, 1, []byte{0x00}},

	// AND VERSION 48'S AFTER THAT (0164): the off-map bit, the record's new
	// last field. Neither entity in this fixture has been taken off the map —
	// nothing but a script arm sets the bit and this fixture runs no script —
	// so the zero value is what this byte carries, on the autocast pair's own
	// reason above.
	{"entity 0 on-map bit", 257, 1, []byte{0x00}},

	// AND VERSION 50'S AFTER THAT: the escort triple, the record's new last
	// fields. Neither entity in this fixture holds an escort order — nothing
	// but a script group sub-command writes one and this fixture runs no script
	// — so all six bytes are the zero value, on the off-map bit's own reason
	// above.
	{"entity 0 escort target", 258, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 0 escort present", 262, 1, []byte{0x00}},
	{"entity 0 escort range", 263, 1, []byte{0x00}},

	{"entity 1 ID", 264, 4, []byte{0x00, 0x00, 0x00, 0x7f}},
	{"entity 1 X", 268, 4, []byte{0x02, 0x00, 0x00, 0x00}},
	{"entity 1 Y", 272, 4, []byte{0xfd, 0xff, 0xff, 0xff}},
	{"entity 1 TargetX", 276, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 TargetY", 280, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 Class", 284, 4, []byte{0xfc, 0xff, 0xff, 0xff}},
	{"entity 1 HasTarget", 288, 1, []byte{0x00}},
	{"entity 1 stall count", 289, 1, []byte{0x00}},
	{"entity 1 HP", 290, 4, []byte{0xff, 0xff, 0xff, 0xff}},
	{"entity 1 MaxHP", 294, 4, []byte{0xfb, 0xfb, 0xfc, 0xfd}},
	{"entity 1 movement domain", 298, 1, []byte{0x01}},
	{"entity 1 Speed", 299, 4, []byte{0x74, 0x73, 0x72, 0x71}},
	// This unit is DEAD, so its crossing is residue and the constructor drops
	// it — while its speed, like its health, is carried whole. Its GROUP TERM
	// is residue for the same reason and goes with them, though 0x91 was named:
	// a felled member is unlinked from its group.
	{"entity 1 transit owed", 303, 2, []byte{0x00, 0x00}},
	{"entity 1 transit length", 305, 2, []byte{0x00, 0x00}},
	{"entity 1 group speed", 307, 1, []byte{0x00}},
	// Its ATTACK ORDER is residue by the same rule — a victim, a presence, a
	// phase and a count were all named here and none survives the constructor —
	// while the seven numbers a blow reads are stats and are carried whole,
	// negative relax included.
	{"entity 1 attack victim", 308, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 attack presence", 312, 1, []byte{0x00}},
	{"entity 1 attack phase", 313, 1, []byte{0x00}},
	{"entity 1 attack countdown", 314, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 attack charge", 318, 4, []byte{0x19, 0x18, 0x17, 0x16}},
	{"entity 1 attack relax", 322, 4, []byte{0xfe, 0xff, 0xff, 0xff}},
	{"entity 1 to-hit", 326, 4, []byte{0x1d, 0x1c, 0x1b, 0x1a}},
	{"entity 1 defence", 330, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 absorption", 334, 4, []byte{0x21, 0x20, 0x1f, 0x1e}},
	{"entity 1 damage base", 338, 4, []byte{0x25, 0x24, 0x23, 0x22}},
	{"entity 1 damage spread", 342, 4, []byte{0x29, 0x28, 0x27, 0x26}},
	{"entity 1 always hits", 346, 1, []byte{0x00}},
	// THIS UNIT IS DEAD AND ITS GROUP AND OWNER SURVIVE, which is the one place
	// in this table where a field of a felled unit is NOT residue. Its crossing,
	// its group term and its whole attack order were dropped above; membership
	// and roster place are not states an order left behind. A count of a group's
	// living members could never reach zero if the last member's death unlinked
	// it, and a hand-over that skipped the fallen would depend on when it fired.
	{"entity 1 group", 347, 4, []byte{0xff, 0xff, 0x00, 0x00}},
	{"entity 1 owner", 351, 4, []byte{0xfe, 0xff, 0x00, 0x00}},
	// AND SO DOES ITS FACING, for a reason of its own rather than membership's:
	// a body faces the way it fell. That is why the not-alive block clears four
	// fields above and leaves this one.
	{"entity 1 facing", 355, 1, []byte{0x2a}},
	// AND SO DOES ITS DECAY BLOCK — the other half of the pairing rule. This
	// unit is DEAD and was handed over at no stage at all, so the constructor put
	// it where a death would have: the first stage, owing the dwell its own dying
	// time names, which is 0x0105 = 261 and pins the field's WIDTH as well as its
	// place. A byte there would have written 5.
	{"entity 1 decay stage", 356, 1, []byte{0x01}},
	{"entity 1 dwell", 357, 2, []byte{0x05, 0x01}},
	{"entity 1 dying time", 359, 4, []byte{0x05, 0x01, 0x00, 0x00}},
	// AND SO DOES ITS SIGHT RANGE. How far a unit saw is a stat, like the seven
	// numbers a blow reads and like the dying time beside it, and not residue of
	// a state it has left — so a corpse carries the one it had.
	{"entity 1 sight range", 363, 1, []byte{0x39}},
	// AND SO DOES ITS ACTOR STATE, on the sight range's own reason: guard is
	// what the constructor writes into every entity regardless of life state,
	// so a corpse's tail here is the same six bytes entity 0's is — carried,
	// and not a fact this pin can vary on a fixture built by NewWorld.
	{"entity 1 actor state", 364, 1, []byte{0x0b}},
	{"entity 1 patrol head X", 365, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 patrol head Y", 369, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 patrol tail X", 373, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 patrol tail Y", 377, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 patrol leg", 381, 1, []byte{0x00}},
	// AND SO DOES ITS REACH (0104), on the sight range's own reason: this
	// entity names none either, so it is the same constructor floor.
	{"entity 1 reach", 382, 1, []byte{0x01}},
	// AND SO DOES ITS POST (0106), on the reach's own reason: a corpse's post
	// is carried exactly as a living unit's is — nothing about it is residue
	// of an order it has left, and it reads back entity 1's own X and Y.
	{"entity 1 post X", 383, 4, []byte{0x02, 0x00, 0x00, 0x00}},
	{"entity 1 post Y", 387, 4, []byte{0xfd, 0xff, 0xff, 0xff}},
	// AND SO DOES ITS REGENERATION BLOCK (0109), on the reach's own reason:
	// this entity names no pool or period either, so it is the same six zeros
	// — CARRIED on the corpse, and not residue: the not-alive block takes no
	// clause for any of the six.
	{"entity 1 mana", 391, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 mana maximum", 395, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 health regen period", 399, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 mana regen period", 403, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 health remainder", 407, 1, []byte{0x00}},
	{"entity 1 mana remainder", 408, 1, []byte{0x00}},
	{"entity 1 command group", 409, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	// AND SO DOES ITS EXPERIENCE FROM USE (0125), on the regeneration
	// block's own reason: this entity names none either, so it is the same
	// ten zeros — CARRIED on the corpse, and not residue: a dead unit's
	// experience is a fact about it rather than a leftover of an order it
	// has left.
	{"entity 1 skill xp 0", 413, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill xp 1", 417, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill xp 2", 421, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill xp 3", 425, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill xp 4", 429, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill xp 5", 433, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 mind", 437, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 xp value", 441, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 xp slot", 445, 1, []byte{0x00}},
	{"entity 1 gains xp", 446, 1, []byte{0x00}},
	// AND SO DOES ITS KNOWN-SPELLS MASK (0127), on the experience
	// block's own reason: this entity names none either, so it is the
	// same four zeros — CARRIED on the corpse, and not residue: a dead
	// unit's book is a fact about it rather than a leftover of an
	// order it has left.
	{"entity 1 known spells", 447, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	// AND SO DOES ITS SKILL BLOCK (0135), on the known-spells mask's own
	// reason: this entity names none either, so it is the same
	// twenty-four zeros — CARRIED on the corpse, and not residue: a dead
	// unit's skill is a fact about it rather than a leftover of an order
	// it has left.
	{"entity 1 skill 0", 451, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill 1", 455, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill 2", 459, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill 3", 463, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill 4", 467, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 skill 5", 471, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	// AND SO DOES ITS WEAPON SPELL (0139), on the skill block's own
	// reason: this entity carries neither a WeaponSpell nor a
	// WeaponSpellLevel either, so it is the same six zeros — CARRIED on
	// the corpse, and not residue: a dead unit's weapon is a fact about
	// it rather than a leftover of an order it has left.
	{"entity 1 weapon spell", 475, 2, []byte{0x00, 0x00}},
	{"entity 1 weapon spell level", 477, 4, []byte{0x00, 0x00, 0x00, 0x00}},

	// AND SO DO ITS AUTOCAST PAIR AND ITS SPELL EFFECT MARK (0154), on the
	// weapon spell's own reason: this entity carries neither either, so it
	// is the same five zeros — CARRIED on the corpse, and not residue.
	{"entity 1 autocast spell", 481, 2, []byte{0x00, 0x00}},
	{"entity 1 autocast wait", 483, 1, []byte{0x00}},
	{"entity 1 spell effect ticks", 484, 2, []byte{0x00, 0x00}},
	{"entity 1 spell effect spell", 486, 1, []byte{0x00}},
	{"entity 1 on-map bit", 487, 1, []byte{0x00}},
	{"entity 1 escort target", 488, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 escort present", 492, 1, []byte{0x00}},
	{"entity 1 escort range", 493, 1, []byte{0x00}},

	// The routes, in record order, after the last record. Neither unit holds
	// one, so each is a count of zero and no cell — which is what pins the
	// section's PLACE and the width of a count that carries nothing.
	{"entity 0 route cell count", 494, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 route cell count", 498, 4, []byte{0x00, 0x00, 0x00, 0x00}},

	{"group record count", 502, 4, []byte{0x02, 0x00, 0x00, 0x00}},
	{"group 0 owner", 506, 4, []byte{0xfe, 0xff, 0x00, 0x00}},
	{"group 0 group", 510, 4, []byte{0xff, 0xff, 0x00, 0x00}},
	{"group 0 base", 514, 1, []byte{0x08}},
	{"group 0 order", 515, 1, []byte{0x01}},
	{"group 0 commandedX", 516, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"group 0 commandedY", 520, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"group 1 owner", 524, 4, []byte{0xb4, 0xb3, 0xb2, 0xb1}},
	{"group 1 group", 528, 4, []byte{0xa4, 0xa3, 0xa2, 0xa1}},
	{"group 1 base", 532, 1, []byte{0xd7}},
	// group 1's owner, 0xb1b2b3b4, is not SelfSlot either, so it is Guard too
	// — the fixture names no roster slot 1, so Stand Ground has no witness
	// here (pinHead below is where it does).
	{"group 1 order", 533, 1, []byte{0x01}},
	{"group 1 commandedX", 534, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"group 1 commandedY", 538, 4, []byte{0x00, 0x00, 0x00, 0x00}},

	// THE SACK SECTION, after the group section (0103): this world names no
	// sack, so it is a bare count of zero — which is what pins the section's
	// PLACE and the width of a count that carries nothing, on the route
	// counts' own rule above.
	{"sack count", 542, 4, []byte{0x00, 0x00, 0x00, 0x00}},

	// THE CARRY SECTION, after the sack section (0112): this world names no
	// stock, so each entity's own record is a bare count of zero and nothing
	// after it — one per entity, entity order, no section count of its own,
	// on the route counts' own rule above.
	{"entity 0 carried code count", 546, 4, []byte{0x00, 0x00, 0x00, 0x00}},
	{"entity 1 carried code count", 550, 4, []byte{0x00, 0x00, 0x00, 0x00}},
}

func TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth(t *testing.T) {
	w := offsetWorld(t)
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	// A 34-byte header, no grid cells at these bounds, a 294-byte record per
	// entity (1037), a 4-byte route count per entity, the group section this
	// world's two owned entities write — a count and two 18-byte records —
	// the empty sack section this world names none of, the empty carry section
	// (0112) — a bare zero count per entity — the empty equipment section
	// (0124 T2) — a bare EquipSlots-wide zero record per entity, no count at
	// all — the empty purse, the empty spell table — a bare zero count and
	// no records — and the empty structure section (1033 B3) — a bare zero
	// count and no records, on the sack section's own rule.
	if want := 61 + 34 + 0 + relationLen + 2*492 + 2*4 + (4 + 2*18) + sackCountLen + 2*carryCountLen + 2*equipRecordLen + 2*treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(2)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(b) != want {
		t.Fatalf("the form of a 2-entity world is %d bytes, want %d", len(b), want)
	}

	// offsetCases is the pre-1001 transcription. Insert each record's own later
	// tails and move every later field by the amount physically inserted before
	// it. Per record the tails are 30 bytes of protection, footprint, detector
	// and stats, one suppress-corpse-loot byte, six of kill attribution, four
	// bytes of Load and four of Capacity since version 56, two of MapUnitID
	// since version 57, and five weapon-kind resistance bytes since version 59
	// (1039), and eight withdrawal threshold bytes since version 60 (1037),
	// 1045's Humanoid byte and 1047's three-byte turn triple: 64 in all. Record
	// 0 therefore spans [34,328) and record 1 [328,622).
	type fieldOffset struct {
		field string
		off   int
		width int
		want  []byte
	}
	cases := make([]fieldOffset, 0, len(offsetCases)+16)
	inserted0, inserted1 := false, false
	for _, c := range offsetCases {
		if c.off == 0 {
			c.want = []byte{formatVersion}
		}
		if !inserted0 && c.off >= 264 {
			cases = append(cases, fieldOffset{"entity 0 protection, footprint, detector and stat tail", 264, 30, make([]byte, 30)})
			cases = append(cases, fieldOffset{"entity 0 suppress-corpse-loot", 294, 1, []byte{0}})
			cases = append(cases,
				fieldOffset{"entity 0 kill-credit source", 295, 4, []byte{0x00, 0x00, 0x00, 0x7f}},
				fieldOffset{"entity 0 kill-credit presence", 299, 1, []byte{1}},
				fieldOffset{"entity 0 kill-credit spell", 300, 1, []byte{0xf9}},
				fieldOffset{"entity 0 load", 301, 4, []byte{0x00, 0x00, 0x00, 0x00}},
				fieldOffset{"entity 0 capacity", 305, 4, []byte{0x00, 0x00, 0x00, 0x00}},
				fieldOffset{"entity 0 map unit id", 309, 2, []byte{0xe4, 0xe3}},
				fieldOffset{"entity 0 weapon-kind resistance", 311, 5, []byte{0xa6, 0xa7, 0xa8, 0xa9, 0xaa}},
				fieldOffset{"entity 0 withdraw", 316, 4, []byte{0x38, 0x37, 0x36, 0x35}},
				fieldOffset{"entity 0 wimpy", 320, 4, []byte{0xfc, 0xfc, 0xfd, 0xfe}},
				fieldOffset{"entity 0 Humanoid", 324, 1, []byte{1}},
				fieldOffset{"entity 0 desired facing", 325, 1, []byte{0xc5}},
				fieldOffset{"entity 0 turn remainder", 326, 1, []byte{0}},
				fieldOffset{"entity 0 turn total", 327, 1, []byte{0}})
			inserted0 = true
		}
		if !inserted1 && c.off >= 494 {
			cases = append(cases, fieldOffset{"entity 1 protection, footprint, detector and stat tail", 558, 30, make([]byte, 30)})
			cases = append(cases, fieldOffset{"entity 1 suppress-corpse-loot", 588, 1, []byte{0}})
			cases = append(cases,
				fieldOffset{"entity 1 kill-credit source", 589, 4, []byte{0, 0, 0, 0}},
				fieldOffset{"entity 1 kill-credit presence", 593, 1, []byte{0}},
				fieldOffset{"entity 1 kill-credit spell", 594, 1, []byte{0x2a}},
				fieldOffset{"entity 1 load", 595, 4, []byte{0x00, 0x00, 0x00, 0x00}},
				fieldOffset{"entity 1 capacity", 599, 4, []byte{0x00, 0x00, 0x00, 0x00}},
				fieldOffset{"entity 1 map unit id", 603, 2, []byte{0xf5, 0x00}},
				fieldOffset{"entity 1 weapon-kind resistance", 605, 5, []byte{0xb6, 0xb7, 0xb8, 0xb9, 0xba}},
				fieldOffset{"entity 1 withdraw", 610, 4, []byte{0xec, 0xec, 0xed, 0xee}},
				fieldOffset{"entity 1 wimpy", 614, 4, []byte{0x48, 0x47, 0x46, 0x45}},
				fieldOffset{"entity 1 Humanoid", 618, 1, []byte{0}},
				fieldOffset{"entity 1 desired facing", 619, 1, []byte{0x2a}},
				fieldOffset{"entity 1 turn remainder", 620, 1, []byte{0}},
				fieldOffset{"entity 1 turn total", 621, 1, []byte{0}})
			inserted1 = true
		}
		if c.off >= 494 {
			c.off += 128
		} else if c.off >= 264 {
			c.off += 64
		}
		cases = append(cases, fieldOffset{c.field, c.off, c.width, c.want})
	}
	// Append-only version-68 stat/capacity words, independently transcribed.
	prior := cases
	cases = nil
	for _, c := range prior {
		if c.off == 328 {
			cases = append(cases, fieldOffset{"entity 0 potion state", 328, 32, make([]byte, 32)})
		}
		if c.off == 622 {
			cases = append(cases, fieldOffset{"entity 1 potion state", 654, 32, make([]byte, 32)})
		}
		if c.off >= 622 {
			c.off += 64
		} else if c.off >= 328 {
			c.off += 32
		}
		cases = append(cases, c)
	}
	prior = cases
	cases = nil
	for _, c := range prior {
		if c.off == 360 {
			cases = append(cases, fieldOffset{"entity 0 Human movement", 360, 15, make([]byte, 15)})
		}
		if c.off == 686 {
			cases = append(cases, fieldOffset{"entity 1 Human movement", 701, 15, make([]byte, 15)})
		}
		if c.off >= 686 {
			c.off += 30
		} else if c.off >= 360 {
			c.off += 15
		}
		cases = append(cases, c)
	}
	next := 0
	prior = cases
	cases = nil
	for _, c := range prior {
		if c.off == 375 {
			cases = append(cases, fieldOffset{"entity 0 spellbook", 375, 113, make([]byte, 113)}, fieldOffset{"entity 0 second physical", 488, 2, []byte{0, 0}})
		}
		if c.off == 716 {
			cases = append(cases, fieldOffset{"entity 1 spellbook", 831, 113, make([]byte, 113)}, fieldOffset{"entity 1 second physical", 944, 2, []byte{0, 0}})
		}
		if c.off >= 716 {
			c.off += 230
		} else if c.off >= 375 {
			c.off += 115
		}
		cases = append(cases, c)
	}
	prior = cases
	cases = nil
	for _, c := range prior {
		if c.off == 490 {
			cases = append(cases, fieldOffset{"entity 0 current profile basis", 490, 1, []byte{0}})
		}
		if c.off == 946 {
			cases = append(cases, fieldOffset{"entity 1 current profile basis", 947, 1, []byte{0}})
		}
		if c.off >= 946 {
			c.off += 2
		} else if c.off >= 490 {
			c.off++
		}
		cases = append(cases, c)
	}
	prior = cases
	cases = nil
	for _, c := range prior {
		if c.off == 491 {
			cases = append(cases, fieldOffset{"entity 0 source binding", 491, 35, make([]byte, 35)})
		}
		if c.off == 948 {
			cases = append(cases, fieldOffset{"entity 1 source binding", 983, 35, make([]byte, 35)})
		}
		if c.off >= 948 {
			c.off += 70
		} else if c.off >= 491 {
			c.off += 35
		}
		cases = append(cases, c)
	}
	for _, c := range cases {
		if c.off != next {
			t.Errorf("%s starts at %d, but the field before it ends at %d", c.field, c.off, next)
		}
		if len(c.want) != c.width {
			t.Fatalf("%s: the case names width %d and %d byte(s)", c.field, c.width, len(c.want))
		}
		next = c.off + c.width
		if c.off+c.width > len(b) {
			t.Fatalf("%s runs past the %d-byte form", c.field, len(b))
		}
		if got := b[c.off : c.off+c.width]; !bytes.Equal(got, c.want) {
			t.Errorf("%s at [%d:%d) is % x, want % x", c.field, c.off, c.off+c.width, got, c.want)
		}
	}
	// THE EQUIPMENT SECTION, after the carry section (0124 T2): this world
	// equips neither entity, so it is two bare EquipSlots-wide zero records,
	// no count of any kind. It is checked as a span rather than transcribed
	// field by field, on the purse section's own argument below: 48
	// transcribed zeros would check nothing the length and the span do not.
	if rest := b[next : next+2*equipRecordLen]; len(rest) != 2*equipRecordLen {
		t.Errorf("the equipment section is %d byte(s), want %d", len(rest), 2*equipRecordLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("equipment section byte %d is %#02x on a world equipping nothing", i, c)
				break
			}
		}
		next += len(rest)
	}
	// The death-gold section follows equipment, one zero record per entity in
	// this fixture.
	if rest := b[next : next+2*treasureRecordLen]; len(rest) != 2*treasureRecordLen {
		t.Errorf("the death-gold section is %d byte(s), want %d", len(rest), 2*treasureRecordLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("death-gold section byte %d is %#02x on a world with no treasure inputs", i, c)
				break
			}
		}
		next += len(rest)
	}
	// THE PURSE SECTION, after the equipment section (0112): this world credits
	// no gold to any roster slot, so it is relationSlots zeroed dwords. It is
	// checked as a span rather than transcribed field by field, on the
	// script section's and the relation's own argument below: 200
	// transcribed zeros would check nothing the length and the span do not.
	if rest := b[next : next+purseLen]; len(rest) != purseLen {
		t.Errorf("the purse section is %d byte(s), want %d", len(rest), purseLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("purse section byte %d is %#02x on a world crediting no gold", i, c)
				break
			}
		}
		next += len(rest)
	}
	// THE SPELL TABLE, after the purse section and before the script section
	// (version 36): this world names no table, so it is the section's own bare
	// count of zero and no records. It is checked as a span rather than
	// transcribed field by field, on the purse section's own argument above:
	// two transcribed zero bytes would check nothing the length and the span do
	// not.
	if rest := b[next : next+spellCountLen]; len(rest) != spellCountLen {
		t.Errorf("the spell table's count is %d byte(s), want %d", len(rest), spellCountLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("spell table byte %d is %#02x on a world naming no spell", i, c)
				break
			}
		}
		next += len(rest)
	}
	// THE ITEM-WEIGHT SECTION, after the spell table and before the casting
	// section (version 56): this world declares no item weight, so it is the
	// section's own bare count of zero and no records. It is checked as a span
	// on the spell table's own argument above.
	if rest := b[next : next+itemWeightCountLen]; len(rest) != itemWeightCountLen {
		t.Errorf("the item-weight table's count is %d byte(s), want %d", len(rest), itemWeightCountLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("item-weight table byte %d is %#02x on a world declaring no weight", i, c)
				break
			}
		}
		next += len(rest)
	}
	// THE CASTING SECTION, after the item-weight section and before the script
	// section (version 49): this world holds no pending cast and no area
	// effect, so it is the section's own two bare zero counts and no records.
	// It is checked as a span on the spell table's own argument above.
	if rest := b[next : next+2*castingCountLen]; len(rest) != 2*castingCountLen {
		t.Errorf("the casting section is %d byte(s), want %d", len(rest), 2*castingCountLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("casting section byte %d is %#02x on a world holding no cast and no area effect", i, c)
				break
			}
		}
		next += len(rest)
	}
	// THE SCRIPT-STATE SECTION, after the casting section and before the script
	// section (version 50): the per-player formation modes and the cell-record
	// tails. This world holds no tail, so the tail half is a bare zero count;
	// the FORMATION BLOCK IS NOT ZERO — every one of its relationSlots bytes
	// carries the default, which is what a world nothing has written a mode
	// into holds. That is the one span in this form checked against a value
	// rather than against zero, and it is checked that way because a block of
	// zeros would be a world every player of which is never in formation.
	if rest := b[next : next+relationSlots+tailCountLen]; len(rest) != relationSlots+tailCountLen {
		t.Errorf("the script-state section is %d byte(s), want %d", len(rest), relationSlots+tailCountLen)
	} else {
		for i, c := range rest[:relationSlots] {
			if c != formationDefault {
				t.Errorf("formation slot %d is %#02x on a world nothing has written a mode into, want %#02x",
					i, c, formationDefault)
				break
			}
		}
		for i, c := range rest[relationSlots:] {
			if c != 0 {
				t.Errorf("cell-tail count byte %d is %#02x on a world holding no cell tail", i, c)
				break
			}
		}
		next += len(rest)
	}
	// THE STRUCTURE SECTION, after the script-state section and before the
	// script section (version 58, 1033 B3): this world names no structure, so
	// it is the section's own bare count of zero and no records, on the
	// item-weight section's own rule above.
	if rest := b[next : next+structureCountLen]; len(rest) != structureCountLen {
		t.Errorf("the structure section is %d byte(s), want %d", len(rest), structureCountLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("structure section byte %d is %#02x on a world declaring no structure", i, c)
				break
			}
		}
		next += len(rest)
	}
	// THE ITEM-STATE SECTION, after structures and before the script (format
	// 61): this fixture has no sack or carried instances, twelve empty slots
	// per entity, no spell provenance and no item-derived state.
	if rest, want := b[next:next+len(emptyItemStatePin(2))], emptyItemStatePin(2); !bytes.Equal(rest, want) {
		t.Errorf("the empty item-state section is % x, want % x", rest, want)
	} else {
		next += len(rest)
	}
	if !bytes.Equal(b[next:next+4], []byte{0, 0, 0, 0}) {
		t.Fatal("nonempty reserved scroll count")
	}
	next += 4
	// The SCRIPT SECTION closes the form and is a run of zeros on a world running
	// no script — the volatile half and three zero counts. It is checked as a
	// span rather than transcribed field by field, because on this world every
	// one of those fields is zero and a table of 1421 zeros would check nothing
	// the length and the span do not. The partition still has to be exact.
	if rest := b[next : len(b)-65-entityIDFloorLen-spellDeliverySpanLen-relationLen-12]; len(rest) != scriptStateLen+scriptCountsLen {
		t.Errorf("the script section is %d byte(s), want %d", len(rest),
			scriptStateLen+scriptCountsLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("script section byte %d is %#02x on a world running no script", i, c)
				break
			}
		}
		next += len(rest)
	}
	if !bytes.Equal(b[next:next+4], []byte{0, 0, 0, 0}) {
		t.Error("nonempty original dead span in empty fixture")
	}
	next += 4
	// THE RELATION CLOSES THE FORM, behind the script section, and on this world
	if !bytes.Equal(b[next:next+4], []byte{0, 0, 0, 0}) {
		t.Error("nonempty instance-weight span")
	}
	next += 4
	if !bytes.Equal(b[next:next+4], []byte{0, 0, 0, 0}) {
		t.Error("nonempty actor-load span")
	}
	next += 4
	// it is a run of zeros for the same reason: nobody is hostile to anybody. It
	// is checked as a span on the script section's own argument — 2500
	// transcribed zeros would check nothing the length and the span do not — and
	// the partition below is what makes both spans exact rather than approximate.
	if rest := b[next : len(b)-65-entityIDFloorLen-spellDeliverySpanLen]; len(rest) != relationLen {
		t.Errorf("the relation is %d byte(s), want %d", len(rest), relationLen)
	} else {
		for i, c := range rest {
			if c != 0 {
				t.Errorf("relation byte %d is %#02x on a world nobody authored a relation for", i, c)
				break
			}
		}
		next += len(rest)
	}
	if !bytes.Equal(b[next:next+5], []byte{0, 0, 0, 0, 0}) {
		t.Error("native legacy clock suffix is not absent with zero FullTick")
	}
	next += 5
	// Forms78..91 append independent span footers. This fixture has no saved
	// Groups, structures, exact Player registry, strides, fine motion, raw
	// planes, objects, or carried resume state: the newest span is outermost,
	// so it is last here too.
	for _, name := range []string{"saved Group", "saved structure", "saved Group Player", "native stride", "saved motion", "saved cell planes", "saved objects", "carried resume", "action clock", "Group counter", "scorched cells", "structure use", "Player formation", "world-effect continuation", "attack notices"} {
		if !bytes.Equal(b[next:next+4], []byte{0, 0, 0, 0}) {
			t.Errorf("nonempty %s span in empty fixture", name)
		}
		next += 4
	}
	// THE ENTITY ID FLOOR CLOSES THE FORM, outside every span above (form94):
	// the highest id this fixture's own entities hold plus one — entity 1's
	// 0x7f000000, the larger of the two — not zero like every span before it.
	if got, want := binary.LittleEndian.Uint64(b[next:next+entityIDFloorLen]), uint64(0x7f000001); got != want {
		t.Errorf("entity id floor = %#x, want %#x", got, want)
	}
	next += entityIDFloorLen
	if binary.LittleEndian.Uint32(b[next:]) != 0 {
		t.Fatal("unexpected delivery footer")
	}
	next += spellDeliverySpanLen
	if next != len(b) {
		t.Errorf("the fields named here cover %d byte(s) of a %d-byte form", next, len(b))
	}
}

// ---------------------------------------------------------------- the pin

// pinSeed, pinWorld and pinBytes are AC-1's pin: one fixed world, and the
// version-7 byte form it must always have. The bytes are a literal
// transcription of the contract's table for this world — nothing here ran the
// encoder to find them — so this test is a change detector for the encoding and
// is honest as one: it says the form has not moved, and the offset table above
// says the form is right.
//
// The three health pairs are chosen the way the class ids are: a positive pair
// on each unit that must stay ALIVE to keep the target and the stall count this
// pin holds, and on the one holding neither, a pair negative on BOTH fields —
// so the sign of each is pinned and a dead unit's residue is shown to be
// nothing. One MaxHP is the top of the range, one HP is 1, and no two of the six
// values share a byte pattern.
//
// The entities are given out of id order and one of them carries coordinates
// for a target it does not have and a stall count it may not keep either, so the
// pin also holds the constructor's sort and its zeroing of a cleared target's
// whole residue. Their class ids are distinct and one is negative, so the pin
// covers the class field's sign as well as its place. The three domains are one
// each, so the pin holds every byte the field may carry and holds them in three
// different records — a domain read out of the neighbouring record would land on
// the wrong one of the three rather than on an equal value.
//
// The bounds are small because the grid is now IN the form: every in-bounds cell
// costs a byte here, and a pin nobody can read cell by cell is not a
// transcription of anything. The mode is the optimised one and the grid uses
// both defined bits with no uniform row, so neither a mode written as a constant
// nor a grid written in column order can pass. One stall count is the highest
// value a stored world may carry.
//
// No unit here holds a route, and none can: two of the three carry targets that
// are off this map, and a route ending anywhere but on its unit's target is a
// form this build refuses. What the tail of this pin therefore fixes is the
// section's presence and its empty case — three counts of zero. A route with
// CELLS in it is pinned in routeform_test.go, on a world built to carry one.
//
// THE GROUP SECTION 0095 ADDS is exercised without a fourth fixture: id 9 is
// in roster slot 0 and contributes no record; id 1 is alive and alone in its
// group, with a sight range of 255 — the byte's top, reached by geometry; id 4
// is not alive at all, and its group's base is the bare floor. So this world
// holds exactly two records — ascending by OWNER (3, then 4294967295) rather
// than by either entity's id — which is AC-1's ordering and AC-3's two edges
// on one fixture.
const pinSeed = 0x0123456789abcdef

func pinGrid() []byte {
	return []byte{
		0, 1, 0, 2, 3,
		0, 0, 1, 0, 0,
		2, 0, 0, 0, 1,
		0, 3, 0, 1, 0,
	}
}

func pinWorld(t *testing.T) *World {
	t.Helper()
	return mustWorldGrid(t, pinSeed, Bounds{Width: 5, Height: 4}, ModeOptimised, pinGrid(), []Entity{
		{ID: 9, X: 0, Y: 0, TargetX: -1, TargetY: 2147483647, Class: 34, HasTarget: true, Stall: 15,
			HP: 1, MaxHP: 2147483647, Domain: DomainAir,
			Speed: 2147483647, Transit: 0, TransitTotal: maxTransit, GroupSpeed: groupSpeedInit,
			AttackTarget: 1, HasAttackTarget: true, AttackPhase: AttackRelaxing, AttackCountdown: 6,
			AttackCharge: 11, AttackRelax: 4, ToHit: -1, Defence: 2147483647, Absorption: 0,
			DamageBase: 255, DamageSpread: 0, AlwaysHits: true, Group: 7, Owner: 0, Facing: 0,
			ScanRange: 0,
			DyingTime: -1},
		{ID: 1, X: 300, Y: -2, TargetX: 1000, TargetY: -1000, Class: 80, HasTarget: true, Stall: 3,
			HP: 100, MaxHP: 250, Domain: DomainGround,
			Speed: 19, Transit: 7, TransitTotal: 14, GroupSpeed: 19,
			AttackTarget: 9, HasAttackTarget: true, AttackPhase: AttackCharging, AttackCountdown: 2,
			AttackCharge: 5, AttackRelax: 3, ToHit: 40, Defence: 12, Absorption: 7,
			DamageBase: 6, DamageSpread: 4, Group: 4294967295, Owner: 3, Facing: 0x60,
			ScanRange: 255,
			DyingTime: 60},
		{ID: 4, X: -7, Y: 65536, TargetX: 55, TargetY: 66, Class: -5, HasTarget: false, Stall: 7,
			HP: -2000000000, MaxHP: -5, Domain: DomainGhost,
			Speed: -3, Transit: 5, TransitTotal: 9, GroupSpeed: 77,
			AttackTarget: 9, HasAttackTarget: true, AttackPhase: AttackCharging, AttackCountdown: 3,
			AttackCharge: 8, AttackRelax: 4, ToHit: 3, Defence: -1, Absorption: 0,
			DamageBase: 2, DamageSpread: 1, AlwaysHits: true, Group: 258, Owner: 4294967295, Facing: 0xff,
			ScanRange: 0x7b,
			DyingTime: 300},
	})
}

// pinBytes is the transcription; the SCRIPT SECTION this world's form ends
// with is a run of zeros — no script, no register written, no latch set
// — and is appended rather than written out, for the reason
// routeform_test.go's is. The SACK SECTION between the group section and it
// (0103) is likewise appended rather than written out: this world names no
// sack, so it is the section's own bare count of zero and nothing else. THE
// CARRY SECTION AND THE PURSE (0112), right after it, are the same kind of
// absent case: this world names no stock and credits no gold, so the carry
// section is three bare four-byte zero counts — one per entity, entity
// order, no section count of its own — and the purse section is
// relationSlots zeroed dwords. THE EQUIPMENT SECTION (0124 T2), between the
// two, is the same kind of absent case again: this world equips nothing, so
// it is three bare EquipSlots-wide zero records — one per entity, entity
// order, no count of its own at all, unlike the carry section's per-entity
// count. THE SPELL TABLE (version 36), right after the purse, is the same
// kind of absent case once more: this world names no table, so it is the
// section's own bare count of zero and no records.
var pinBytesPre1001Form = append(append(append(append(append(append(append(append(append(append(append(append(append(pinHead(), pinGroupSection()...), make([]byte, sackCountLen)...),
	make([]byte, 3*carryCountLen)...), make([]byte, 3*equipRecordLen)...), make([]byte, 3*treasureRecordLen)...), make([]byte, purseLen)...),
	make([]byte, spellCountLen)...),
	make([]byte, itemWeightCountLen)...),
	make([]byte, 2*castingCountLen)...),
	pinScriptStateSection()...),
	make([]byte, structureCountLen)...),
	make([]byte, scriptStateLen+scriptCountsLen)...),
	make([]byte, relationLen)...)

var pinBytesV54 = widenedCorpseLootPin(widened1001Pin(pinBytesPre1001Form, 20, 3), 20, 3)
var pinBytesV56 = widenedCarriedWeightPin(widenedItemAttributionPin(pinBytesV54, 20, 3), 20, 3)
var pinBytesV58 = widenedMapUnitIDPin(pinBytesV56, 20, 3)
var pinBytesV59 = widenedWeaponResistancePin(pinBytesV58, 20, 3)
var pinBytesV60 = widenedWithdrawalThresholdPin(pinBytesV59, 20, 3)
var pinBytesV61 = widenedItemStatePin(pinBytesV60, 3, scriptStateLen+scriptCountsLen+relationLen)
var pinBytesV62 = widenedActionCadenceEntityPin(pinBytesV61, 20, 3)
var pinBytesV63 = widenedTurnProgressPin(pinBytesV62, 20, 3)
var pinBytesV67 = widenedTurnDurationPin(pinBytesV63, 20, 3)
var pinBytesV68 = widenedConsumablePin(pinBytesV67, 20, 3, scriptStateLen+scriptCountsLen+relationLen)
var pinBytesV69 = widenedHumanMovementPin(pinBytesV68, 20, 3)
var pinBytesV70 = widenedOriginalDeadPin(pinBytesV69)
var pinBytesV71 = widenedSpellbookPin(pinBytesV70, 20, 3)
var pinBytesV72 = widenedSecondPhysicalPin(pinBytesV71, 20, 3)
var pinBytesV73 = widenedCurrentProfilePin(pinBytesV72, 20, 3)
var pinBytesV74 = widenedInstanceWeightPin(pinBytesV73)
var pinBytesV75 = widenedActorLoadPin(pinBytesV74)
var pinBytesV76 = widenedSourceBindingPin(pinBytesV75)

// pinWorld's own highest entity id is 9 (entity list above); no originalDead
// and no script, so its floor is exactly that plus one.
var pinBytes = widenedEntityIDFloorPin(widenedSessionClockPin(pinBytesV76), 10)

func widenedCurrentProfilePin(old []byte, cells, records int) []byte {
	base := 34 + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		at := base + 456*i
		out = append(out, old[at:at+456]...)
		out = append(out, 0)
	}
	out = append(out, old[base+456*records:]...)
	out[0] = 73
	return out
}

func strippedWorldOfCurrentProfile(form []byte) []byte {
	out := strippedInstanceWeightPin(form)
	if out[0] < 73 {
		return out
	}
	cells := int(binary.LittleEndian.Uint32(out[30:34]))
	records := int(binary.LittleEndian.Uint32(out[25:29]))
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		at := base + 457*i + 456
		out = append(out[:at], out[at+1:]...)
	}
	out[0] = 72
	return out
}

// Independent form-72 transcription. Never derive these offsets from the
// production constants: the two bytes follow the unchanged form-71 record.
func widenedSecondPhysicalPin(old []byte, cells, records int) []byte {
	base := 34 + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 454*i
		out = append(out, old[o:o+454]...)
		out = append(out, 0, 0)
	}
	out = append(out, old[base+454*records:]...)
	out[0] = 72
	return out
}

func strippedWorldOfSecondPhysical(form []byte) []byte {
	out := strippedWorldOfCurrentProfile(form)
	if out[0] < 72 {
		return out
	}
	cells := int(binary.LittleEndian.Uint32(out[30:34]))
	records := int(binary.LittleEndian.Uint32(out[25:29]))
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		at := base + 456*i + 454
		out = append(out[:at], out[at+2:]...)
	}
	out[0] = 71
	return out
}

// Independent form-71 transcription: 341 existing bytes, then 113 zero bytes
// representing the pre-instance, table-backed legacy book state.
func widenedSpellbookPin(old []byte, cells, records int) []byte {
	base := 34 + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 341*i
		out = append(out, old[o:o+341]...)
		out = append(out, make([]byte, 113)...)
	}
	out = append(out, old[base+341*records:]...)
	out[0] = 71
	return out
}

func strippedWorldOfSpellbook(form []byte) []byte {
	out := strippedWorldOfSecondPhysical(form)
	if out[0] < 71 {
		return out
	}
	cells := int(binary.LittleEndian.Uint32(out[30:34]))
	records := int(binary.LittleEndian.Uint32(out[25:29]))
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		at := base + 454*i + 341
		out = append(out[:at], out[at+113:]...)
	}
	out[0] = 70
	return out
}

// Independent form-70 transcription: an empty span before the relation.
func widenedOriginalDeadPin(old []byte) []byte {
	at := len(old) - 2500
	out := append([]byte(nil), old[:at]...)
	out = append(out, 0, 0, 0, 0)
	out = append(out, old[at:]...)
	out[0] = 70
	return out
}

func widenedHumanMovementPin(old []byte, cells, records int) []byte {
	base := 34 + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 326*i
		out = append(out, old[o:o+326]...)
		out = append(out, make([]byte, 15)...)
	}
	out = append(out, old[base+326*records:]...)
	out[0] = 69
	return out
}

func strippedWorldOfOriginalDead(form []byte) []byte {
	out := strippedWorldOfSpellbook(form)
	if out[0] >= 70 {
		end := len(out) - 2500
		span := int(binary.LittleEndian.Uint32(out[end-4:])) + 4
		out = append(out[:end-span], out[end:]...)
		out[0] = 69
	}
	return out
}

func strippedWorldOfHumanMovement(form []byte, w *World) []byte {
	out := strippedWorldOfOriginalDead(form)
	if out[0] < 69 {
		return out
	}
	base := 34 + 3*int(gridCells(w.bounds))
	for i := len(w.entities) - 1; i >= 0; i-- {
		at := base + 341*i + 326
		out = append(out[:at], out[at+15:]...)
	}
	out[0] = 68
	return out
}

// Independent version-68 layout: eight new zero i32 values per actor and an
// empty reserved-scroll count before the existing script section.
func widenedConsumablePin(old []byte, cells, records, scriptTail int) []byte {
	base := 34 + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 294*i
		out = append(out, old[o:o+294]...)
		out = append(out, make([]byte, 32)...)
	}
	at := len(old) - scriptTail
	out = append(out, old[base+294*records:at]...)
	out = append(out, 0, 0, 0, 0)
	out = append(out, old[at:]...)
	out[0] = 68
	return out
}

func strippedWorldOfConsumables(form []byte, w *World) []byte {
	out := strippedWorldOfHumanMovement(form, w)
	if out[0] < 68 {
		return out
	}
	at := len(out) - relationLen - w.scriptSectionLen() - w.scrollSectionLen()
	out = append(out[:at], out[at+w.scrollSectionLen():]...)
	base := 34 + 3*int(gridCells(w.bounds))
	for i := len(w.entities) - 1; i >= 0; i-- {
		at := base + 326*i + 294
		out = append(out[:at], out[at+32:]...)
	}
	out[0] = 67
	return out
}

// widenedActionCadenceEntityPin independently transcribes the only new bytes
// these two long-lived pins carry: neither pin has a spell or book cast, and
// every actor is non-humanoid. The non-zero spell and lifecycle populations
// have their own form-62 witnesses in cadenceform1045_test.go.
func widenedActionCadenceEntityPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + entityLenV61*i
		out = append(out, old[o:o+entityLenV61]...)
		out = append(out, 0)
	}
	out = append(out, old[base+entityLenV61*records:]...)
	out[0] = 62
	return out
}

// widenedTurnProgressPin independently appends version 63's inactive turn
// pair: the desired byte repeats the established current Facing at +91 and the
// remaining count is zero. Reading the facing out of each old record keeps the
// pin honest for the arbitrary non-quantized byte carried by entity 9.
func widenedTurnProgressPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + entityLenV62*i
		out = append(out, old[o:o+entityLenV62]...)
		out = append(out, old[o+91], 0)
	}
	out = append(out, old[base+entityLenV62*records:]...)
	out[0] = 63
	return out
}

// widenedTurnDurationPin independently appends version 64's TurnTotal. Every
// turn in these long-lived pins is inactive, so the canonical total is zero.
func widenedTurnDurationPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + entityLenV63*i
		out = append(out, old[o:o+entityLenV63]...)
		out = append(out, 0)
	}
	out = append(out, old[base+entityLenV63*records:]...)
	out[0] = 67
	return out
}

// emptyItemStatePin is format 61's independent empty-population
// transcription: no sacks; one carried count, twelve empty instances and one
// source tag per entity; then the item-derived scalars and the one
// secondary-damage triple inside its required-zero reserved width.
func emptyItemStatePin(records int) []byte {
	// Keep the independent pin independent of the production section-width
	// constants: 4-byte sack and entity counts, then per entity a 4-byte
	// carried count, twelve 11-byte empty item heads, a one-byte provenance
	// tag and 52 bytes of derived state: twelve scalar bytes, a three-byte
	// secondary-damage triple and thirty-seven required-zero reserved bytes.
	out := make([]byte, 0, 8+records*(4+12*11+1+52))
	out = binary.LittleEndian.AppendUint32(out, 0)
	out = binary.LittleEndian.AppendUint32(out, uint32(records))
	for i := 0; i < records; i++ {
		out = binary.LittleEndian.AppendUint32(out, 0)
	}
	for i := 0; i < records*EquipSlots; i++ {
		out = append(out, make([]byte, itemHeadLen)...)
	}
	out = append(out, make([]byte, records*(1+itemDerivedStateLen))...)
	return out
}

func widenedItemStatePin(old []byte, records, tailAfter int) []byte {
	at := len(old) - tailAfter
	section := emptyItemStatePin(records)
	out := make([]byte, 0, len(old)+len(section))
	out = append(out, old[:at]...)
	out = append(out, section...)
	out = append(out, old[at:]...)
	out[0] = formatVersion
	return out
}

func strippedOfItemState(form []byte, records, tailAfter int) []byte {
	sectionLen := len(emptyItemStatePin(records))
	at := len(form) - tailAfter - sectionLen
	out := append([]byte(nil), form[:at]...)
	out = append(out, form[at+sectionLen:]...)
	out[0] = 60
	return out
}

var (
	pinBytesPreItemState = strippedOfItemState(pinBytesV61, 3, scriptStateLen+scriptCountsLen+relationLen)
	rtfBytesPreItemState = strippedOfItemState(rtfBytesV61, 3, len(rtfScriptSection())+relationLen)
)

// pinTailAfterItemWeights is how many bytes of the pinned form sit AFTER the
// item-weight section: the casting section, the script-state section, the
// structure section (1033 B3), the script section and the relation block, in
// that order. It is the one number strippedOfCarriedWeight cannot derive from
// a record count, so the two pinned worlds each state their own.
var pinTailAfterItemWeights = 2*castingCountLen + len(pinScriptStateSection()) +
	structureCountLen + scriptStateLen + scriptCountsLen + relationLen

// widenedWeaponResistancePin and strippedOfWeaponResistance are version 59's
// independent transcription pair. Five zero bytes are appended to each entity
// because neither pinned world carries a resistance value; offsetWorld is the
// nonzero byte-pattern witness. No whole-form section moves in this version.
func widenedWeaponResistancePin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 277*i
		out = append(out, old[o:o+277]...)
		out = append(out, make([]byte, 5)...)
	}
	out = append(out, old[base+277*records:]...)
	out[0] = formatVersion
	return out
}

func strippedOfWeaponResistance(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := headerLen + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 282*i
		out = append(out[:o+277], out[o+282:]...)
	}
	out[0] = preWeaponResistanceFormVersion
	return out
}

const preWeaponResistanceFormVersion byte = 58

var (
	pinBytesPreWeaponResistance = strippedOfWeaponResistance(pinBytesPreWithdrawalThreshold, 20, 3)
	rtfBytesPreWeaponResistance = strippedOfWeaponResistance(rtfBytesPreWithdrawalThreshold, 16, 3)
)

// TestThePinIsThePreviousPinPlusWeaponResistance proves the form bump
// without deriving its expected bytes from the production encoder.
func TestWeaponResistanceTailPeelsToItsPublishedPredecessor(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"pinned world", pinBytesPreWithdrawalThreshold, 20, 3, 0x86a572758c6baf4f},
		{"routed world", rtfBytesPreWithdrawalThreshold, 16, 3, 0x14a465f5adef7332},
	} {
		stripped := strippedOfWeaponResistance(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 5*tc.records; len(stripped) != want {
			t.Errorf("%s stripped length = %d, want %d", tc.name, len(stripped), want)
		}
		if stripped[0] != preWeaponResistanceFormVersion {
			t.Errorf("%s stripped version = %d, want %d", tc.name, stripped[0], preWeaponResistanceFormVersion)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s stripped hash = %#016x, want version-58 %#016x", tc.name, got, tc.want)
		}
	}
}

// widenedWithdrawalThresholdPin and strippedOfWithdrawalThreshold are version
// 60's independent transcription pair. Eight zero bytes are appended to every
// fixture record; offsetWorld is the separate nonzero byte-pattern witness.
func widenedWithdrawalThresholdPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 282*i
		out = append(out, old[o:o+282]...)
		out = append(out, make([]byte, 8)...)
	}
	out = append(out, old[base+282*records:]...)
	out[0] = formatVersion
	return out
}

func strippedOfWithdrawalThreshold(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := headerLen + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 290*i
		out = append(out[:o+282], out[o+290:]...)
	}
	out[0] = 59
	return out
}

var (
	pinBytesPreWithdrawalThreshold = strippedOfWithdrawalThreshold(pinBytesPreItemState, 20, 3)
	rtfBytesPreWithdrawalThreshold = strippedOfWithdrawalThreshold(rtfBytesPreItemState, 16, 3)
)

func TestWithdrawalThresholdTailPeelsToItsPublishedPredecessor(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"pinned world", pinBytesPreItemState, 20, 3, 0x20d8a806642715a0},
		{"routed world", rtfBytesPreItemState, 16, 3, 0x19a4ce081307f38b},
	} {
		stripped := strippedOfWithdrawalThreshold(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 8*tc.records; len(stripped) != want {
			t.Errorf("%s stripped length = %d, want %d", tc.name, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s stripped hash = %#016x, want version-59 %#016x", tc.name, got, tc.want)
		}
	}
}

// strippedOfTheStructureSection is 1033's own peel (B3). The structure
// section is a whole new tail section rather than a per-record widening —
// mirroring the item-weight section's own introduction, on
// widenedCarriedWeightPin's own comment below — so there is no widening
// counterpart here, only the strip: the section already lives in
// pinBytesPre1001Form and rtfBytesPre1001Form. Neither pinned nor routed
// world declares a structure, so it is its own bare structureCountLen zero
// count and nothing else, sitting at a fixed distance from the end — the
// script section and the relation block after it are both constant width for
// a world running no script.
func strippedOfTheStructureSection(form []byte) []byte {
	out := append([]byte(nil), form...)
	at := len(out) - (scriptStateLen + scriptCountsLen + relationLen) - structureCountLen
	out = append(out[:at], out[at+structureCountLen:]...)
	out[0] = preStructureSectionFormVersion
	return out
}

// preStructureSectionFormVersion is the version this story's own section
// replaces. It is named rather than spelled at each site so that the next
// bump moves one line, on preMapUnitIDFormVersion's own reason below.
const preStructureSectionFormVersion byte = 57

// pinBytesPreStructureSection and rtfBytesPreStructureSection are the two
// pinned forms as version 57 wrote them. EVERY OLDER PEEL STARTS HERE rather
// than at pinBytes, on pinBytesPreMapUnitID's own reason below: a peel
// written for an earlier version walks record offsets and tail widths this
// story's own section moved.
var (
	pinBytesPreStructureSection = strippedOfTheStructureSection(pinBytesPreWeaponResistance)
	rtfBytesPreStructureSection = strippedOfTheStructureSection(rtfBytesPreWeaponResistance)
)

// TestThePinIsThePreviousPinPlusTheStructureSection peels this story's own
// section back off both pinned forms and requires what is left to be the
// form the previous version wrote, byte for byte.
//
// THE NAME CARRIES NO VERSION NUMBER, on
// TestThePinIsThePreviousPinPlusTheAuthoredMapID's own reason below: the
// version belongs in strippedOfTheStructureSection's own constant.
func TestThePinIsThePreviousPinPlusTheStructureSection(t *testing.T) {
	for _, tc := range []struct {
		name string
		form []byte
		want uint64
	}{
		// The two wanted hashes are NOT computed from this peel. They are the
		// pinDigest and rtfDigest constants as the tree carried them at
		// version 57 -- pinDigest 0x077f2a83b4415c20 (hash_test.go) and
		// rtfDigest 0xe7ac2e14cf3c5bb1 (routeform_test.go), both moved below
		// by this story.
		{"pinned world", pinBytesPreWeaponResistance, 0x077f2a83b4415c20},
		{"routed world", rtfBytesPreWeaponResistance, 0xe7ac2e14cf3c5bb1},
	} {
		stripped := strippedOfTheStructureSection(tc.form)
		if want := len(tc.form) - structureCountLen; len(stripped) != want {
			t.Errorf("%s stripped length = %d, want %d", tc.name, len(stripped), want)
		}
		if stripped[0] != preStructureSectionFormVersion {
			t.Errorf("%s stripped version byte = %d, want %d", tc.name, stripped[0], preStructureSectionFormVersion)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s stripped hash = %#016x, want version-57 %#016x", tc.name, got, tc.want)
		}
	}
}

// widenedCarriedWeightPin and strippedOfCarriedWeight are version 56's own
// pair. The version changed the form in two places: the ENTITY RECORD grew
// eight bytes at +267 -- Load then Capacity, both int32 -- and a two-byte
// item-weight count was inserted between the spell table and the casting
// section.
//
// THE TWO HELPERS ARE NOT SYMMETRIC, because this file's own convention is
// that a whole-form section lives in the pinned literal and a widening only
// widens RECORDS. So the item-weight section is written into
// pinBytesPre1001Form beside the spell table's own count, the widening below
// is widenedItemAttributionPin's shape exactly, and only the strip has to
// remove both.
//
// EVERY INSERTED BYTE IS ZERO FOR THE PINNED WORLDS, and that is a fact about
// these worlds rather than a convenience: neither declares an item weight, so
// the new section is its own bare zero count, and neither carries a capacity
// nor holds anything, so every load and every capacity is zero. A world where
// either were nonzero could not be widened this way at all.
func widenedCarriedWeightPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 267*i
		out = append(out, old[o:o+267]...)
		out = append(out, make([]byte, 8)...)
	}
	out = append(out, old[base+267*records:]...)
	out[0] = formatVersion
	return out
}

// widenedMapUnitIDPin and strippedOfTheMapUnitID are 1029's pair. The version
// changed the form in two places, and only one of them touches these pins: the
// ENTITY RECORD grew two bytes at +277, the authored map id as a uint16. The
// other half is the script section's check record, and both pinned worlds
// compile no script at all -- their script sections are three zero counts -- so
// no check record exists here to widen.
//
// EVERY INSERTED BYTE IS ZERO FOR THE PINNED WORLDS, and that is a fact about
// these worlds rather than a convenience: neither is built from a map, so no
// entity in either carries an authored map id. A world where one did could not
// be widened this way.
func widenedMapUnitIDPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 275*i
		out = append(out, old[o:o+275]...)
		out = append(out, make([]byte, 2)...)
	}
	out = append(out, old[base+275*records:]...)
	out[0] = preWeaponResistanceFormVersion
	return out
}

func strippedOfTheMapUnitID(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := headerLen + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 277*i
		out = append(out[:o+275], out[o+277:]...)
	}
	out[0] = preMapUnitIDFormVersion
	return out
}

// preMapUnitIDFormVersion is the version this story's own tail replaced. It is
// named rather than spelled at each site so that the next bump moves one line.
const preMapUnitIDFormVersion byte = 56

// pinBytesPreMapUnitID and rtfBytesPreMapUnitID are the two pinned forms as
// version 56 wrote them. They peel from pinBytesPreStructureSection rather
// than from pinBytes (1033 B3), on pinBytesPreCarriedWeight's own reason
// below: a peel written for an earlier version walks record offsets and tail
// widths that a later version moved.
var (
	pinBytesPreMapUnitID = strippedOfTheMapUnitID(pinBytesPreStructureSection, 20, 3)
	rtfBytesPreMapUnitID = strippedOfTheMapUnitID(rtfBytesPreStructureSection, 16, 3)
)

// TestThePinIsThePreviousPinPlusTheAuthoredMapID peels this story's own tail
// back off both pinned forms and requires what is left to be the form the
// previous version wrote, byte for byte.
//
// THE NAME CARRIES NO VERSION NUMBER, deliberately. Three siblings above it do
// (TestThePinIsTheVersion53PinPlus..., 54, 55) and each was correct when
// written, but a name that spells a version number is a name that has to be
// re-read at every bump to tell whether it still means what it says. The
// numbers belong in strippedOfTheMapUnitID's own constant, which is the one
// place a bump has to move them.
func TestThePinIsThePreviousPinPlusTheAuthoredMapID(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		// The two wanted hashes are NOT computed from this peel. They are the
		// pinDigest and rtfDigest constants as the tree carried them at version
		// 56 -- pinDigest 0x3af68fef64afc381 (hash_test.go) and rtfDigest
		// 0xc182806d328a32f8 (routeform_test.go), both moved before this
		// story, by 1029. So the assertion is that peeling version 57's own
		// tail off the version-57 form (pinBytesPreStructureSection, 1033
		// B3, itself peeled from the current tip above) reproduces the
		// previous version's form byte for byte, measured against a number
		// recorded before 1029 existed.
		{"pinned world", pinBytesPreStructureSection, 20, 3, 0x3af68fef64afc381},
		{"routed world", rtfBytesPreStructureSection, 16, 3, 0xc182806d328a32f8},
	} {
		stripped := strippedOfTheMapUnitID(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 2*tc.records; len(stripped) != want {
			t.Errorf("%s stripped length = %d, want %d", tc.name, len(stripped), want)
		}
		if stripped[0] != preMapUnitIDFormVersion {
			t.Errorf("%s stripped version byte = %d, want %d", tc.name, stripped[0], preMapUnitIDFormVersion)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s stripped hash = %#016x, want the previous version's %#016x", tc.name, got, tc.want)
		}
	}
}

// pinBytesPreCarriedWeight and rtfBytesPreCarriedWeight are the two pinned
// forms as version 55 wrote them. They peel from pinBytesPreMapUnitID rather
// than from pinBytes, on the rule stated there.
var (
	pinBytesPreCarriedWeight = strippedOfCarriedWeight(pinBytesPreMapUnitID, 20, 3, pinTailAfterItemWeights)
	rtfBytesPreCarriedWeight = strippedOfCarriedWeight(rtfBytesPreMapUnitID, 16, 3, rtfTailAfterItemWeights)
)

func strippedOfCarriedWeight(form []byte, cells, records, tailAfter int) []byte {
	out := append([]byte(nil), form...)
	at := len(out) - tailAfter - itemWeightCountLen
	out = append(out[:at], out[at+itemWeightCountLen:]...)
	base := headerLen + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 275*i
		out = append(out[:o+267], out[o+275:]...)
	}
	out[0] = 55
	return out
}

func TestThePinIsTheVersion55PinPlusCarriedWeight(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    []byte
		cells   int
		records int
		tail    int
		want    uint64
	}{
		{"pinned world", pinBytesPreMapUnitID, 20, 3, pinTailAfterItemWeights, 0x830c42ddfeaca97a},
		{"routed world", rtfBytesPreMapUnitID, 16, 3, rtfTailAfterItemWeights, 0x99c2bae8e42ef41f},
	} {
		stripped := strippedOfCarriedWeight(tc.form, tc.cells, tc.records, tc.tail)
		if want := len(tc.form) - 8*tc.records - itemWeightCountLen; len(stripped) != want {
			t.Errorf("%s stripped length = %d, want %d", tc.name, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s stripped hash = %#016x, want version-55 %#016x", tc.name, got, tc.want)
		}
	}
}

func widenedItemAttributionPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 261*i
		out = append(out, old[o:o+261]...)
		out = append(out, make([]byte, 6)...)
	}
	out = append(out, old[base+261*records:]...)
	out[0] = formatVersion
	return out
}

func strippedOfItemAttribution(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := headerLen + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 267*i
		out = append(out[:o+261], out[o+267:]...)
	}
	out[0] = 54
	return out
}

func TestThePinIsTheVersion54PinPlusItemAttribution(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"pinned world", pinBytesPreCarriedWeight, 20, 3, 0x0cdb4dbec102336b},
		{"routed world", rtfBytesPreCarriedWeight, 16, 3, 0x1f218329fc919786},
	} {
		stripped := strippedOfItemAttribution(tc.form, tc.cells, tc.records)
		if len(stripped) != len(tc.form)-6*tc.records {
			t.Errorf("%s stripped length = %d, want %d", tc.name, len(stripped), len(tc.form)-6*tc.records)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s stripped hash = %#016x, want version-54 %#016x", tc.name, got, tc.want)
		}
	}
}

func widenedCorpseLootPin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 260*i
		out = append(out, old[o:o+260]...)
		out = append(out, 0)
	}
	out = append(out, old[base+260*records:]...)
	out[0] = formatVersion
	return out
}

func strippedOfCorpseLoot(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := headerLen + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 261*i
		out = append(out[:o+260], out[o+261:]...)
	}
	out[0] = 53
	return out
}

func TestThePinIsTheVersion53PinPlusCorpseLootSuppression(t *testing.T) {
	for _, tc := range []struct {
		name    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"pinned world", strippedOfItemAttribution(pinBytesPreCarriedWeight, 20, 3), 20, 3, 0x55c345ef6ff2287a},
		{"routed world", strippedOfItemAttribution(rtfBytesPreCarriedWeight, 16, 3), 16, 3, 0x816195816a8c6591},
	} {
		stripped := strippedOfCorpseLoot(tc.form, tc.cells, tc.records)
		if len(stripped) != len(tc.form)-tc.records {
			t.Errorf("%s stripped length = %d, want %d", tc.name, len(stripped), len(tc.form)-tc.records)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s stripped hash = %#016x, want version-53 %#016x", tc.name, got, tc.want)
		}
	}
}

func widened1001Pin(old []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), old[:base]...)
	for i := 0; i < records; i++ {
		o := base + 230*i
		out = append(out, old[o:o+230]...)
		out = append(out, make([]byte, 30)...)
	}
	out = append(out, old[base+230*records:]...)
	out[0] = formatVersion
	return out
}

func strippedOf1001(form []byte, cells, records int) []byte {
	base := headerLen + 3*cells
	out := append([]byte(nil), form[:base]...)
	for i := 0; i < records; i++ {
		o := base + 260*i
		out = append(out, form[o:o+230]...)
	}
	out = append(out, form[base+260*records:]...)
	out[0] = 50
	return out
}

// pinScriptStateSection is 0166's own addition (version 50): the per-player
// formation modes and the cell-record tails. It is WRITTEN OUT rather than
// appended as zeros, unlike every absent section above it, because the
// formation block's absent case is NOT zero — a world nothing has written
// a mode into carries the default in every slot, and a block of zeros would
// be a world every player of which is never in formation. The tail half is
// this world's own bare zero count: it holds no cell tail.
func pinScriptStateSection() []byte {
	out := make([]byte, relationSlots, relationSlots+tailCountLen)
	for i := range out {
		out[i] = formationDefault
	}
	return append(out, make([]byte, tailCountLen)...)
}

// pinGroupSection is 0095's own addition, grown in place by 0096: a count of
// two, then the two records ascending BY OWNER — id 1's group (owner 3)
// before id 4's (owner 4294967295), which is the opposite of the order the
// two entities themselves appear in above. Id 1 is alone and alive in its
// group with a sight range of 255, so its base is that geometry exactly and
// pins the field's top; id 4 is not alive at all, so its group's base is the
// bare floor, 8. Id 9 is in roster slot 0 and contributes no record, which is
// why two records and not three.
//
// THE ORDER AND THE COMMANDED CELL ARE THE NEW TAIL: both records take
// Guard, since neither owner (3, then 4294967295) is SelfSlot — this
// fixture names no roster slot 1, so Stand Ground is pinned elsewhere
// (pinHead below) and not here — and both commanded cells are the origin,
// unread until a command writes one.
func pinGroupSection() []byte {
	return []byte{
		0x02, 0x00, 0x00, 0x00, // two group records

		0x03, 0x00, 0x00, 0x00, // owner 3 — id 1's roster slot
		0xff, 0xff, 0xff, 0xff, // group 4294967295 — id 1's own group word
		0xff, // base 255 — the byte's top, reached by geometry:
		//             id 1 is alone in its group, so the centroid is its own
		//             cell, at distance 0 from itself, plus a sight range of
		//             255
		0x01,                   // order 1: Guard — owner 3 is not SelfSlot
		0x00, 0x00, 0x00, 0x00, // commandedX 0 — the origin, unread
		0x00, 0x00, 0x00, 0x00, // commandedY 0

		0xff, 0xff, 0xff, 0xff, // owner 4294967295 — id 4's roster slot
		0x02, 0x01, 0x00, 0x00, // group 258 — id 4's own group word
		0x08, // base 8 — the guard-range floor: id 4 is this group's only
		//             member and it is not alive, so the group has no LIVING
		//             member and takes the floor bare
		0x01,                   // order 1: Guard — owner 4294967295 is not SelfSlot
		0x00, 0x00, 0x00, 0x00, // commandedX 0
		0x00, 0x00, 0x00, 0x00, // commandedY 0
	}
}

func pinHead() []byte {
	return []byte{
		0x32,                                           // version 50: the last script arms (0166)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 0
		0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01, // rng state = the seed
		0x05, 0x00, 0x00, 0x00, // bounds.Width 5
		0x04, 0x00, 0x00, 0x00, // bounds.Height 4
		0x03, 0x00, 0x00, 0x00, // entity count 3
		0x01,                   // routing mode: optimised
		0x14, 0x00, 0x00, 0x00, // plane cell count 20 — ONE count, all three planes

		0x00, 0x01, 0x00, 0x02, 0x03, // block row 0
		0x00, 0x00, 0x01, 0x00, 0x00, // block row 1
		0x02, 0x00, 0x00, 0x00, 0x01, // block row 2
		0x00, 0x03, 0x00, 0x01, 0x00, // block row 3

		// The cost plane this world names none of, materialised: the uniform
		// default, which is what makes a world with no cost plane and one built
		// over this plane the same twenty bytes and not merely the same routes.
		0x08, 0x08, 0x08, 0x08, 0x08, // cost row 0
		0x08, 0x08, 0x08, 0x08, 0x08, // cost row 1
		0x08, 0x08, 0x08, 0x08, 0x08, // cost row 2
		0x08, 0x08, 0x08, 0x08, 0x08, // cost row 3

		// And the height plane, materialised flat.
		0x00, 0x00, 0x00, 0x00, 0x00, // height row 0
		0x00, 0x00, 0x00, 0x00, 0x00, // height row 1
		0x00, 0x00, 0x00, 0x00, 0x00, // height row 2
		0x00, 0x00, 0x00, 0x00, 0x00, // height row 3

		0x01, 0x00, 0x00, 0x00, // id 1
		0x2c, 0x01, 0x00, 0x00, // X 300
		0xfe, 0xff, 0xff, 0xff, // Y -2
		0xe8, 0x03, 0x00, 0x00, // TargetX 1000
		0x18, 0xfc, 0xff, 0xff, // TargetY -1000
		0x50, 0x00, 0x00, 0x00, // Class 80
		0x01,                   // HasTarget
		0x03,                   // stall count 3
		0x64, 0x00, 0x00, 0x00, // HP 100
		0xfa, 0x00, 0x00, 0x00, // MaxHP 250
		0x00,                   // movement domain: ground, the zero value
		0x13, 0x00, 0x00, 0x00, // Speed 19
		0x07, 0x00, // 7 transit ticks owed
		0x0e, 0x00, // of a transit of 14
		0x13,                   // a group rate term of 19 — this one is alive and keeps it
		0x09, 0x00, 0x00, 0x00, // attacking id 9
		0x01,                   // and holding one, so the presence byte is set
		0x01,                   // phase 1: charging
		0x02, 0x00, 0x00, 0x00, // 2 tick(s) owed, strictly below its own charge
		0x05, 0x00, 0x00, 0x00, // charge 5
		0x03, 0x00, 0x00, 0x00, // relax 3
		0x28, 0x00, 0x00, 0x00, // to-hit 40
		0x0c, 0x00, 0x00, 0x00, // defence 12
		0x07, 0x00, 0x00, 0x00, // absorption 7
		0x06, 0x00, 0x00, 0x00, // damage base 6
		0x04, 0x00, 0x00, 0x00, // damage spread 4
		0x00,                   // and it can miss
		0xff, 0xff, 0xff, 0xff, // group 4294967295, the top of the range
		0x03, 0x00, 0x00, 0x00, // owner: roster slot 3
		0x60, // facing 0x60 — direction 3, south-east: a value this build writes,
		//             being a whole multiple of 32
		0x00, // no decay stage: this one is ALIVE, and the constructor
		//             refuses to let a living unit carry one
		0x00, 0x00, // and no dwell, which only the first stage owes
		0x3c, 0x00, 0x00, 0x00, // a dying time of 60 — CARRIED on the living, a
		//             stat like the speed above it rather than a state of dying
		0xff, // a sight range of 255 — the top of the field, so its width is
		//             pinned on the one record where a narrower one would still
		//             look right
		0x0b, // actor state: guard (0099) — NewWorld writes it into every entity
		//             unconditionally, so this fixture holds no
		//             patroller though nothing here names a state at all
		0x00, 0x00, 0x00, 0x00, // patrol head X — residue, on the stall count's
		0x00, 0x00, 0x00, 0x00, // patrol head Y — own ground: an actor that is
		0x00, 0x00, 0x00, 0x00, // patrol tail X — not patrolling carries no
		0x00, 0x00, 0x00, 0x00, // patrol tail Y — ring, and the constructor
		0x00, // patrol leg: head (0) — clears one it could have been handed
		0x01, // reach 1 — this entity names none, so the constructor's own
		//             floor is what the byte carries (0104)
		0x2c, 0x01, 0x00, 0x00, // post X 300 — PostX, PostY == X, Y, written
		0xfe, 0xff, 0xff, 0xff, // post Y -2 — unconditionally at construction (0106)
		0x00, 0x00, 0x00, 0x00, // mana 0 — this fixture names no mana pool at all
		0x00, 0x00, 0x00, 0x00, // mana maximum 0, so the pool has no system (0109)
		0x00, 0x00, 0x00, 0x00, // health regen period 0 — this fixture names none
		0x00, 0x00, 0x00, 0x00, // mana regen period 0 — nor this one
		0x00,                   // health remainder 0
		0x00,                   // mana remainder 0
		0x00, 0x00, 0x00, 0x00, // command group 0 (0117) — this task writes
		//             nothing here, so every actor, this one included, is decided
		//             for by its placed group alone
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

		0x04, 0x00, 0x00, 0x00, // id 4
		0xf9, 0xff, 0xff, 0xff, // X -7
		0x00, 0x00, 0x01, 0x00, // Y 65536
		0x00, 0x00, 0x00, 0x00, // TargetX 0 — the residue of a target never set
		0x00, 0x00, 0x00, 0x00, // TargetY 0
		0xfb, 0xff, 0xff, 0xff, // Class -5
		0x00,                   // HasTarget false
		0x00,                   // stall count 0 — residue as well, and zeroed with them
		0x00, 0x6c, 0xca, 0x88, // HP -2000000000
		0xfb, 0xff, 0xff, 0xff, // MaxHP -5 — dead, and its residue is nothing
		0x01,                   // movement domain: ghost
		0xfd, 0xff, 0xff, 0xff, // Speed -3 — negative, so this one has no rate
		0x00, 0x00, // no transit ticks owed — cleared with the rest of a
		0x00, 0x00, // corpse's residue, though 5 of 9 were named
		0x00, // and no group term, though 77 was named: a felled member is
		//             unlinked from its group, and the decoder refuses a nonzero
		//             one here
		0x00, 0x00, 0x00, 0x00, // no victim, though id 9 was named
		0x00,                   // no presence
		0x00,                   // phase 0: ready
		0x00, 0x00, 0x00, 0x00, // and nothing owed — the whole order is residue on
		//             a unit that is not alive, exactly as its crossing is
		0x08, 0x00, 0x00, 0x00, // charge 8 — the seven numbers are STATS, not
		0x04, 0x00, 0x00, 0x00, // relax 4, so death does not clear one of them
		0x03, 0x00, 0x00, 0x00, // to-hit 3
		0xff, 0xff, 0xff, 0xff, // defence -1, carried whole and signed
		0x00, 0x00, 0x00, 0x00, // absorption 0
		0x02, 0x00, 0x00, 0x00, // damage base 2
		0x01, 0x00, 0x00, 0x00, // damage spread 1
		0x01,                   // and this corpse would never have missed
		0x02, 0x01, 0x00, 0x00, // group 258 — CARRIED, though this unit is dead:
		//             membership is not residue, and a group whose last member
		//             died would otherwise stop having had one
		0xff, 0xff, 0xff, 0xff, // owner 4294967295 — CARRIED for the same reason and
		//             at the top of the field's range, so the width is pinned on
		//             the one record where a refusal would have been tempting
		0xff, // facing 0xff — CARRIED on the corpse, for a third reason: a body
		//             faces the way it fell. And it is NOT a multiple of 32, the
		//             value this build never writes and must still read back
		//             unaltered — the one shape a decoder that rounded a facing to
		//             a direction would fail on
		0x01, // decay stage 1: this unit is DEAD and was handed over at no
		//             stage at all, so the constructor put it where a death
		//             would have — the other half of the pairing rule the record
		//             above holds
		0x2c, 0x01, // and owing a dwell of 300, which is its own dying time and
		//             is past a byte, so this pins the field's width as well
		0x2c, 0x01, 0x00, 0x00, // dying time 300
		0x7b, // a sight range of 123, CARRIED on the corpse for the dying time's
		//             own reason: how far a unit saw is a stat and not residue of
		//             a state it has left
		0x0b, //
		//             every entity regardless of life state, which is what a
		//             not-alive block that takes no clause for it means
		0x00, 0x00, 0x00, 0x00, // patrol head X
		0x00, 0x00, 0x00, 0x00, // patrol head Y
		0x00, 0x00, 0x00, 0x00, // patrol tail X
		0x00, 0x00, 0x00, 0x00, // patrol tail Y
		0x00,                   // patrol leg: head (0)
		0x01,                   // reach 1
		0xf9, 0xff, 0xff, 0xff, // post X -7 — PostX, PostY == X, Y (0106)
		0x00, 0x00, 0x01, 0x00, // post Y 65536
		0x00, 0x00, 0x00, 0x00, // mana 0 — CARRIED on the corpse, on the group
		0x00, 0x00, 0x00, 0x00, // mana maximum 0 — word's own reason: neither is residue
		0x00, 0x00, 0x00, 0x00, // health regen period 0
		0x00, 0x00, 0x00, 0x00, // mana regen period 0
		0x00,                   // health remainder 0
		0x00,                   // mana remainder 0
		0x00, 0x00, 0x00, 0x00, // command group 0 — CARRIED on the corpse for
		//             the group word's own reason: this task writes it
		//             nowhere, alive or not
		0x00, 0x00, 0x00, 0x00, // SkillXP[0..5] — CARRIED on the corpse for
		0x00, 0x00, 0x00, 0x00, // the group word's own reason: neither is
		0x00, 0x00, 0x00, 0x00, // residue, and this class does not gain
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // Mind 0
		0x00, 0x00, 0x00, 0x00, // XPValue 0
		0x00,                   // XPSlot 0: General
		0x00,                   // GainsXP false
		0x00, 0x00, 0x00, 0x00, // known spells 0 (0127) — CARRIED on the
		//             corpse for the group word's own reason: a dead unit's
		//             book is a fact about it rather than residue
		0x00, 0x00, 0x00, 0x00, // Skill[0..5] (0135) — CARRIED on the corpse
		0x00, 0x00, 0x00, 0x00, // for the group word's own reason: neither is
		0x00, 0x00, 0x00, 0x00, // residue, and this fixture names no levels
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, // weapon spell 0 (0139) — carried on the corpse the same way
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
		0x00, 0x00, 0x00, 0x00, // X 0
		0x00, 0x00, 0x00, 0x00, // Y 0
		0xff, 0xff, 0xff, 0xff, // TargetX -1
		0xff, 0xff, 0xff, 0x7f, // TargetY 2147483647
		0x22, 0x00, 0x00, 0x00, // Class 34
		0x01,                   // HasTarget
		0x0f,                   // stall count 15 — the highest a stored world may carry
		0x01, 0x00, 0x00, 0x00, // HP 1
		0xff, 0xff, 0xff, 0x7f, // MaxHP 2147483647
		0x02,                   // movement domain: air
		0xff, 0xff, 0xff, 0x7f, // Speed 2147483647 — carried whole, never clamped here
		0x00, 0x00, // no transit ticks owed
		0x00, 0x01, // of a transit of 256, the longest the law can make —
		// two bytes, and a byte-wide field could not hold it
		0xfa,                   // a group rate term of 250, the highest a minimum can come out at
		0x01, 0x00, 0x00, 0x00, // attacking id 1 — the two are attacking each other
		0x01,                   // and holding one
		0x02,                   // phase 2: relaxing
		0x06, 0x00, 0x00, 0x00, // 6 tick(s) owed, inside relax plus the jitter's top
		0x0b, 0x00, 0x00, 0x00, // charge 11
		0x04, 0x00, 0x00, 0x00, // relax 4
		0xff, 0xff, 0xff, 0xff, // to-hit -1, so this one needs the roll or the mark
		0xff, 0xff, 0xff, 0x7f, // defence 2147483647, the top of the range
		0x00, 0x00, 0x00, 0x00, // absorption 0
		0xff, 0x00, 0x00, 0x00, // damage base 255
		0x00, 0x00, 0x00, 0x00, // damage spread 0 — a fixed blow, whose roll is
		//             still taken
		0x01,                   // and it always hits
		0x07, 0x00, 0x00, 0x00, // group 7
		0x00, 0x00, 0x00, 0x00, // owner 0 — NO OWNER, which is a value and not a
		//             gap: the slot space is 1-based, so zero is free to mean
		//             absent and no presence byte is spent on it
		0x00, // facing 0 — NORTH, and a direction rather than an absence: the
		//             delta table's own index 0, which is what a unit that has
		//             never turned faces
		0x00,       // no decay stage — alive
		0x00, 0x00, // and no dwell
		0xff, 0xff, 0xff, 0xff, // dying time -1, NEGATIVE and carried whole: the
		//             column's own absent value, which the dwell rule reads as no
		//             dwell rather than as a wrapped sixty-five thousand ticks
		0x00, // a sight range of 0 — A RANGE AND NOT AN ABSENCE, and the two
		//             absent-looking values in a row are why both are said: this
		//             unit marches nowhere and lights only its own cell, and no
		//             rule anywhere substitutes a value for it
		0x0b,                   // actor state: guard — the third and last entity, the same tail
		0x00, 0x00, 0x00, 0x00, // patrol head X
		0x00, 0x00, 0x00, 0x00, // patrol head Y
		0x00, 0x00, 0x00, 0x00, // patrol tail X
		0x00, 0x00, 0x00, 0x00, // patrol tail Y
		0x00,                   // patrol leg: head (0)
		0x01,                   // reach 1
		0x00, 0x00, 0x00, 0x00, // post X 0 — PostX, PostY == X, Y (0106)
		0x00, 0x00, 0x00, 0x00, // post Y 0
		0x00, 0x00, 0x00, 0x00, // mana 0 — this fixture names no mana pool at all
		0x00, 0x00, 0x00, 0x00, // mana maximum 0, so the pool has no system (0109)
		0x00, 0x00, 0x00, 0x00, // health regen period 0 — this fixture names none
		0x00, 0x00, 0x00, 0x00, // mana regen period 0 — nor this one
		0x00,                   // health remainder 0
		0x00,                   // mana remainder 0
		0x00, 0x00, 0x00, 0x00, // command group 0 — id 9 sits in roster slot 0
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
		0x00, 0x00, 0x00, 0x00, // known spells 0 (0127) — id 9's own zero,
		//             on the command group's own reason: this task gives
		//             none of the three any writer
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

		0x00, 0x00, 0x00, 0x00, // id 1 holds no route
		0x00, 0x00, 0x00, 0x00, // id 4 holds no route
		0x00, 0x00, 0x00, 0x00, // id 9 holds no route
	}
}

func TestMarshalledBytesArePinned(t *testing.T) {
	if want := 61 + 34 + 3*20 + relationLen + 3*492 + 3*4 + (4 + 2*18) + sackCountLen + 3*carryCountLen + 3*equipRecordLen + 3*treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(3)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(pinBytes) != want {
		t.Fatalf("the pin is %d bytes; a 3-entity world over 20 cells is %d", len(pinBytes), want)
	}
	got, err := pinWorld(t).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(got, pinBytes) {
		t.Errorf("the pinned world marshals to\n % x\npinned as\n % x\n"+
			"— a changed encoding fails here instead of silently breaking a save", got, pinBytes)
	}
}

// TestThePinnedBytesDecodeBackToThePinnedWorld closes the pin's other direction:
// the bytes were written by hand, so they are only the right ones if the decoder
// recovers the world they were written for.
func TestThePinnedBytesDecodeBackToThePinnedWorld(t *testing.T) {
	var got World
	if err := got.UnmarshalBinary(pinBytes); err != nil {
		t.Fatalf("UnmarshalBinary(pinBytes): %v", err)
	}
	want := snap(pinWorld(t))
	if diff := snap(&got); !equalState(diff, want) {
		t.Errorf("the pinned bytes decode to\n %+v\nwant\n %+v", diff, want)
	}
}

func equalState(a, b worldState) bool {
	if a.tick != b.tick || a.rng != b.rng || a.bounds != b.bounds || a.mode != b.mode {
		return false
	}
	if !bytes.Equal(a.grid, b.grid) || len(a.entities) != len(b.entities) {
		return false
	}
	for i := range a.entities {
		if a.entities[i] != b.entities[i] {
			return false
		}
	}
	// The routes are compared cell by cell, and their slot count with them: a
	// refusal that left a route behind on the receiver is a change this
	// comparison has to see, and the digest alone could only say something moved.
	if len(a.routes) != len(b.routes) {
		return false
	}
	for i := range a.routes {
		if len(a.routes[i]) != len(b.routes[i]) {
			return false
		}
		for k := range a.routes[i] {
			if a.routes[i][k] != b.routes[i][k] {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------- round trip

func TestUnmarshalReplacesTheWholeReceiver(t *testing.T) {
	// The receiver starts richer than what it is handed, so a decoder that
	// merged instead of replacing would leave entities behind.
	b, ents := sample()
	w := mustWorld(t, 0xfeedface, b, ents)
	Step(w, []Command{{Entity: 2, X: 30, Y: 30}})

	other := mustWorld(t, 5, Bounds{Width: 2, Height: 3}, []Entity{{ID: 100, X: -9, Y: 9}})
	form, err := other.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if err := w.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, want := snap(w), snap(other); !equalState(got, want) {
		t.Fatalf("after unmarshal the receiver is\n %+v\nwant\n %+v", got, want)
	}

	// The world keeps nothing that reaches back into the buffer it was decoded
	// from, so a caller reusing that buffer cannot move a decoded world.
	before := snap(w)
	for i := range form {
		form[i] ^= 0xff
	}
	if got := snap(w); !equalState(got, before) {
		t.Errorf("mutating the source bytes reached the world:\n %+v\n %+v", got, before)
	}
}

func TestMarshalRoundTripsAndReMarshalsIdentically(t *testing.T) {
	w := pinWorld(t)
	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var back World
	if err := back.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, the original %#016x", back.Hash(), w.Hash())
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(again, first) {
		t.Errorf("re-marshalling gave\n % x\nwant\n % x", again, first)
	}
}

// TestAnInterruptedRunContinuesIdentically is AC-5. The schedule places two
// orders after the cut, so the continuation is not a walk that was already
// determined before it.
func TestAnInterruptedRunContinuesIdentically(t *testing.T) {
	const k = 4
	b := Bounds{Width: 16, Height: 16}
	ents := []Entity{{ID: 1, X: 0, Y: 0}, {ID: 3, X: 9, Y: 9}, {ID: 8, X: -2, Y: 5}}
	schedule := [][]Command{
		{{Entity: 1, X: 6, Y: 6}, {Entity: 3, X: 0, Y: 0}},
		nil,
		{{Entity: 8, X: 4, Y: 1}},
		nil,
		{{Entity: 1, X: -3, Y: 12}}, // at tick k, i.e. after the cut
		nil,
		{{Entity: 3, X: 20, Y: 20}}, // and again, later
		nil,
	}

	whole := mustWorld(t, 0x0badc0de, b, ents)
	var atK []byte
	var digestAtK uint64
	for i, cmds := range schedule {
		if i == k {
			var err error
			if atK, err = whole.MarshalBinary(); err != nil {
				t.Fatalf("MarshalBinary at tick %d: %v", k, err)
			}
			digestAtK = whole.Hash()
		}
		Step(whole, cmds)
	}
	if whole.Tick() != uint64(len(schedule)) {
		t.Fatalf("the uninterrupted run reached tick %d", whole.Tick())
	}

	var resumed World
	if err := resumed.UnmarshalBinary(atK); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if resumed.Tick() != k {
		t.Errorf("the resumed world is at tick %d, want %d", resumed.Tick(), k)
	}
	if resumed.Hash() != digestAtK {
		t.Errorf("at the cut the resumed world hashes %#016x, the original %#016x", resumed.Hash(), digestAtK)
	}
	again, err := resumed.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(again, atK) {
		t.Errorf("re-marshalling the resumed world gave\n % x\nwant\n % x", again, atK)
	}

	for _, cmds := range schedule[k:] {
		Step(&resumed, cmds)
	}
	if resumed.Hash() != whole.Hash() {
		t.Errorf("the continued run hashes %#016x, the uninterrupted one %#016x",
			resumed.Hash(), whole.Hash())
	}
}

// TestAReachedTargetEncodesLikeATargetNeverSet: the byte form carries the
// logical world, so an entity that walked onto its target must be
// indistinguishable from one that was placed there with no order at all.
//
// THE PLACED UNIT IS BUILT FACING EAST, and that is the point rather than a
// concession. A facing is not residue of an order — it is where the unit is
// pointing, and a unit that walked one cell east IS pointing east — so the two
// worlds are the same world only if the placed one is given the same facing.
// Written the other way round, this test would have demanded that arriving
// erase a fact about the unit.
func TestAReachedTargetEncodesLikeATargetNeverSet(t *testing.T) {
	b := Bounds{Width: 8, Height: 8}
	walked := mustWorld(t, 77, b, []Entity{{ID: 1, X: 3, Y: 3}})
	Step(walked, []Command{{Entity: 1, X: 4, Y: 3}})

	placed := mustWorld(t, 77, b, []Entity{{ID: 1, X: 4, Y: 3, Facing: facingOfDir(2)}})
	placed.tick = 1 // the two worlds differ in nothing else, so nor may the tick
	// The POST is the one field this pair legitimately differs in otherwise
	// (0106): NewWorld writes it from each entity's own cell at construction,
	// so the walker's is (3,3), where it started, and placed's own literal
	// would freeze at (4,3), where it is handed. A walk never moves the post
	// — only a stance command does — so a unit that walked onto its target
	// and one placed there with no order at all are the same world only once
	// this one difference, which is not what this test is about, is closed by
	// hand.
	placed.entities[0].PostX, placed.entities[0].PostY = 3, 3

	wf, err := walked.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	pf, err := placed.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(wf, pf) {
		t.Errorf("a cleared target left residue in the form:\n % x\n % x", wf, pf)
	}
	if walked.Hash() != placed.Hash() {
		t.Errorf("digests %#016x and %#016x", walked.Hash(), placed.Hash())
	}
}

// TestTheFormFollowsTheSliceLengthNotItsCapacity: nothing in this package hands
// a world a slice with room to spare today, so an encoder that sized itself by
// capacity would be invisible. The world is given one directly.
func TestTheFormFollowsTheSliceLengthNotItsCapacity(t *testing.T) {
	w := mustWorld(t, 7, Bounds{Width: 3, Height: 4}, []Entity{{ID: 1, X: 1}, {ID: 2, Y: 2}})
	tight, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	roomy := make([]Entity, len(w.entities), 16)
	copy(roomy, w.entities)
	w.entities = roomy
	spacious := make([][]cell, len(w.routes), 16)
	copy(spacious, w.routes)
	w.routes = spacious

	got, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if want := 61 + 34 + 3*(3*4) + relationLen + 2*492 + 2*4 + groupCountLen + sackCountLen + 2*carryCountLen + 2*equipRecordLen + 2*treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(2)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(got) != want {
		t.Errorf("a 2-entity world with room for 16 marshals to %d bytes, want %d", len(got), want)
	}
	if !bytes.Equal(got, tight) {
		t.Errorf("spare capacity changed the form:\n % x\n % x", got, tight)
	}
}

// TestTheCountFieldIsFourBytesWide: the count is the one header field whose
// value in every other test here fits in a single byte, so a count written half
// as wide would leave every form above byte-identical. This world's count needs
// the top half of the field, and a 16-bit encoder would have this form declare
// no entities at all.
func TestTheCountFieldIsFourBytesWide(t *testing.T) {
	const n = 1 << 16
	ents := make([]Entity, n)
	for i := range ents {
		ents[i] = Entity{ID: EntityID(i), X: int32(i), Y: -int32(i)}
	}
	w := mustWorld(t, 1, Bounds{Width: 1, Height: 1}, ents)

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if want := 61 + 34 + 3*1 + relationLen + n*492 + n*4 + groupCountLen + sackCountLen + n*carryCountLen + n*equipRecordLen + n*treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(n)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(b) != want {
		t.Fatalf("the form is %d bytes, want %d", len(b), want)
	}
	if got, want := b[25:29], []byte{0x00, 0x00, 0x01, 0x00}; !bytes.Equal(got, want) {
		t.Errorf("the count field is % x, want % x for %d entities", got, want, n)
	}

	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, the original %#016x", back.Hash(), w.Hash())
	}
}

// ---------------------------------------------------------------- refusals

// withU32, withU16 and withByte edit a copy of a valid form. The little-endian
// store is written out here rather than taken from encoding/binary, so a test
// that crafts a bad form does not borrow the encoder's idea of byte order.
func withU32(b []byte, off int, v uint32) []byte {
	out := append([]byte(nil), b...)
	out[off] = byte(v)
	out[off+1] = byte(v >> 8)
	out[off+2] = byte(v >> 16)
	out[off+3] = byte(v >> 24)
	return out
}

func withU16(b []byte, off int, v uint16) []byte {
	out := append([]byte(nil), b...)
	out[off] = byte(v)
	out[off+1] = byte(v >> 8)
	return out
}

func withByte(b []byte, off int, v byte) []byte {
	out := append([]byte(nil), b...)
	out[off] = v
	return out
}

// truncated is b's first n bytes in a buffer of exactly that capacity. A plain
// b[:n] would keep the whole array behind it, and a slice expression may reach
// up to capacity — so a decoder reading past the length it was given would read
// the rest of a valid form and the test would pass. This one faults instead.
func truncated(b []byte, n int) []byte {
	out := make([]byte, n)
	copy(out, b)
	return out
}

// populated is AC-9's receiver: a world with contents, at a tick it reached by
// stepping. A refusal tested on a zero world could not tell "left as it was"
// from "cleared".
func populated(t *testing.T) *World {
	t.Helper()
	b, ents := sample()
	w := mustWorld(t, 0xa5a5a5a5a5a5a5a5, b, ents)
	Step(w, []Command{{Entity: 2, X: 20, Y: 20}, {Entity: 5, X: -6, Y: 20}})
	Step(w, nil)
	return w
}

func TestUnmarshalRefusesAndLeavesTheReceiverExactlyAsItWas(t *testing.T) {
	// A valid two-entity form to spoil: a 34-byte header, three 12-cell planes
	// at [34,70), 294-byte entity records at 70 and 364, then the routes.
	// Within a record the presence byte is at +24, the stall
	// count at +25, HP at +26, MaxHP at +30, the movement domain at +34, the
	// transit pair at +39 and +41 and the group rate term at +43 — so the
	// record offsets used below are 94, 95, 96, 100, 104, 109, 111 and 113 in
	// the first record and 388, 421, 390, 426, 398, 435, 405 and 407 in the
	// second. Spelling both sets keeps a widening of the second record visible;
	// adjusting only the first record's offsets cannot hide it.
	//
	// A comment that names offsets no case uses cannot fail, which is how the
	// paragraph above went stale twice before it was corrected here rather
	// than shifted: it is re-derived from the record's OWN offsets each time
	// rather than adjusted by the widening's delta, which is what keeps a
	// silent off-by-the-tail from surviving a fourth.
	//
	// Both units are built at 0/0: no health system, so both are ALIVE and the
	// first keeps the target and the stall count several cases here spoil. That
	// is also what makes the health spoils below reachable — a unit already dead
	// could not be made dead by one edited field.
	src := mustWorldGrid(t, 3, Bounds{Width: 4, Height: 3}, ModeCanonical,
		[]byte{0, 1, 0, 2, 3, 0, 1, 0, 0, 2, 0, 1}, []Entity{
			{ID: 4, X: 1, Y: 2, TargetX: 6, TargetY: 6, Class: 7, HasTarget: true, Stall: 5},
			{ID: 6, X: -3, Y: 0, Class: -1},
		})
	valid, err := src.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(valid) != 61+34+3*12+relationLen+2*492+2*4+groupCountLen+sackCountLen+2*carryCountLen+2*equipRecordLen+2*treasureRecordLen+purseLen+spellCountLen+itemWeightCountLen+2*castingCountLen+relationSlots+tailCountLen+structureCountLen+len(emptyItemStatePin(2))+4+4+4+4+4+scriptStateLen+scriptCountsLen+entityIDFloorLen+spellDeliverySpanLen {
		t.Fatalf("the fixture form is %d bytes", len(valid))
	}

	cases := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		// Several truncation lengths, not just one. A length guard loosened by
		// a whole record's worth lets a buffer through whose shortfall divides
		// evenly by 100, and a negative record count is not a refusal but a
		// panic — 4 is that length for this form.
		{"one byte", truncated(valid, 1)},
		{"four bytes", truncated(valid, 4)},
		{"twenty bytes", truncated(valid, 20)},
		{"one byte short of a header", truncated(valid, 33)},
		{"header alone, its grid missing", truncated(valid, 34)},
		{"the grid one byte short", truncated(valid, 45)},
		{"header and grid, declaring an entity", withU32(truncated(valid, 46), 25, 1)},
		{"one byte short", truncated(valid, len(valid)-1)},
		{"one byte over", append(append([]byte(nil), valid...), 0x00)},
		{"a whole record short of its own count", truncated(valid, 46+100)},
		{"the records whole and the routes missing", truncated(valid, 46+2*100)},
		{"one route count short", truncated(valid, 46+2*100+4)},
		{"version 0", withByte(valid, 0, 0)},
		{"version 1", withByte(valid, 0, 1)},
		{"version 2", withByte(valid, 0, 2)},
		{"version 3", withByte(valid, 0, 3)},
		{"version 4", withByte(valid, 0, 4)},
		{"version 5", withByte(valid, 0, 5)},
		{"version 6", withByte(valid, 0, 6)},
		{"version 7", withByte(valid, 0, 7)},
		{"version 8", withByte(valid, 0, 8)},
		{"version 9", withByte(valid, 0, 9)},
		{"version 10", withByte(valid, 0, 10)},
		{"version 11", withByte(valid, 0, 11)},
		{"version 12", withByte(valid, 0, 12)},
		{"version 13", withByte(valid, 0, 13)},
		{"count one too many", withU32(valid, 25, 3)},
		{"count one too few", withU32(valid, 25, 1)},
		{"count at the top of its range", withU32(valid, 25, 0xffffffff)},
		{"routing mode 2", withByte(valid, 29, 2)},
		{"routing mode 255", withByte(valid, 29, 0xff)},
		{"grid cell count one too many", withU32(valid, 30, 13)},
		{"grid cell count one too few", withU32(valid, 30, 11)},
		{"grid cell count zero", withU32(valid, 30, 0)},
		{"grid cell count at the top of its range", withU32(valid, 30, 0xffffffff)},
		{"a reserved grid bit in the first cell", withByte(valid, 34, 0x08)},
		{"a reserved grid bit in the last cell", withByte(valid, 45, 0x80)},
		{"every reserved grid bit", withByte(valid, 40, 0xf8)},
		{"stall count at the limit", withByte(valid, 95, 16)},
		{"stall count at the top of its range", withByte(valid, 95, 0xff)},
		{"a stall count on an entity with no target", withByte(valid, 436, 1)},
		{"ids descending", withU32(withU32(valid, 70, 6), 524, 4)},
		{"duplicate ids", withU32(valid, 34+3*12+492, 4)},
		{"presence byte 2", withByte(valid, 94, 2)},
		{"presence byte 255", withByte(valid, 435, 0xff)},

		// The movement domain at each record's +34: the first byte past the three
		// values this build defines, and the top of the field. Both records are
		// spoiled, so a decoder that checked only the first is caught.
		{"movement domain 3 in the first record", withByte(valid, 104, 3)},
		{"movement domain 255 in the first record", withByte(valid, 104, 0xff)},
		{"movement domain 3 in the second record", withByte(valid, 445, 3)},

		// The residue a unit that is not alive may not carry. The first record
		// holds a target and a stall count of 5, so ONE edited health field is
		// what turns each of these from a legal world into a refused one, and the
		// two ways of leaving alive are covered separately: below zero, and at
		// exactly zero against a positive maximum.
		//
		// The last two are the same residue on the SECOND record, which holds no
		// target — so the rule that catches them is the one that already refuses a
		// stall count and a stored route on a unit holding none. That is the whole
		// of why the decoder needs one refusal here and not three: a unit that is
		// not alive cannot carry a stall count or a route without carrying a
		// target too, and a target is what this rule refuses.
		{"a target on a unit killed below zero", withU32(valid, 96, 0xffffffff)},
		{"a target on a unit downed at exactly zero", withU32(valid, 100, 1)},
		{"a target on a unit far below zero", withU32(valid, 96, 0xfffffc18)},
		{"a stall count on a dead unit holding no target",
			withByte(withU32(valid, 437, 0xffffffff), 436, 1)},
		// The GROUP RATE TERM on a unit that is not alive, in both records and by
		// both ways of leaving alive. The second record holds no target, so this
		// is the one piece of residue whose refusal the target rule cannot stand
		// in for — which is why the decoder carries a clause of its own for it.
		{"a group term on a unit killed below zero",
			withByte(withU32(valid, 96, 0xffffffff), 113, 1)},
		{"a group term on a unit downed at exactly zero",
			withByte(withU32(valid, 100, 1), 113, 0xfa)},
		{"a group term on a dead unit holding no target",
			withByte(withU32(valid, 437, 0xffffffff), 454, 1)},

		// The transit pair, at +39 and +41 of the first record. Each is a state
		// no tick can leave: a crossing longer than the law's own longest, an
		// owed count at its own transit's length, one past it, one owed with no
		// transit to owe it to, and a crossing on a unit that is not alive —
		// which is a refusal of its own, since a felled mover holds no target for
		// the rule above to catch it by.
		{"a transit longer than the law can make", withU16(valid, 111, 257)},
		{"a transit at the top of its range", withU16(valid, 111, 0xffff)},
		{"an owed count at its transit's length", withU16(withU16(valid, 111, 9), 109, 9)},
		{"an owed count past its transit's length", withU16(withU16(valid, 111, 9), 109, 10)},
		{"an owed count with no transit", withU16(valid, 109, 1)},
		{"a transit on a unit killed below zero",
			withU16(withU16(withU32(valid, 96, 0xffffffff), 111, 9), 109, 3)},

		// THE ATTACK BLOCK, at +44…+82 of each record — so the first record's at
		// 114…152 and the second's at 408…446. Neither unit in this fixture is
		// attacking, so every one of these is a byte or a word edited into a form
		// that is otherwise exactly the valid one.
		//
		// The two 0/1 bytes first, in both records, because a decoder that read
		// either as truthy would map two forms onto one world. The block's own
		// offsets did not move when the group word arrived: it went past them.
		{"an attack-presence byte of 2", withByte(valid, 118, 2)},
		{"an attack-presence byte at the top of its range", withByte(valid, 459, 0xff)},
		{"an always-hits byte of 2", withByte(valid, 152, 2)},
		{"an always-hits byte at the top of its range", withByte(valid, 493, 0xff)},

		// The phase and the count, which both sides refuse alike.
		{"an attack phase this build does not define", withByte(valid, 119, 3)},
		{"an attack phase at the top of its range", withByte(valid, 460, 0xff)},
		{"a negative count owed", withU32(valid, 120, 0xffffffff)},
		// A charging count at or above its own charge, which is the bound
		// transitFault's own argument gives: the count is loaded from the charge
		// and only falls. The charge here is 0, whose floor is 1, so a count of 5
		// is five past what any tick could have left.
		{"a count owed above its own charge",
			withU32(withByte(withByte(withU32(valid, 114, 6), 94, 1), 95, 1), 120, 5)},
		// And a relaxing one past the relax plus the jitter's own top.
		{"a count owed above its own relax and jitter",
			withU32(withByte(withByte(withU32(valid, 114, 6), 94, 1), 95, 2), 120, 4)},

		// The residue an entity holding no order may not carry — one clause per
		// field, because each is a separate way for a cleared order to leave
		// something behind.
		{"a victim id on a unit holding no attack order", withU32(valid, 114, 6)},
		{"a phase on a unit holding no attack order", withByte(valid, 460, 1)},
		{"a count owed on a unit holding no attack order", withU32(valid, 461, 1)},

		// The order itself: naming its own unit, naming one no record declares,
		// and standing on a unit that is not alive. The last is asked of the
		// SECOND record, which holds no target, so the not-alive rule that
		// already refuses a target cannot stand in for it.
		//
		// THE GROUP WORD IS NOT HERE, and that is the point of saying so: it is
		// the one field of a felled unit this decoder does not refuse, because
		// membership is not something a spent order left behind.
		{"an attack order naming its own unit",
			withByte(withU32(valid, 114, 4), 118, 1)},
		{"an attack order naming a unit the form does not hold",
			withByte(withU32(valid, 114, 99), 118, 1)},
		{"an attack order on a unit killed below zero",
			withByte(withU32(withU32(valid, 437, 0xffffffff), 455, 4), 459, 1)},
		{"an attack order on a unit downed at exactly zero",
			withByte(withU32(withU32(valid, 441, 1), 455, 4), 459, 1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := populated(t)
			before := snap(w)
			beforeForm, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			if before.tick == 0 {
				t.Fatal("the receiver was not populated")
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

	// The one that must NOT be refused, so the table above is not passing by
	// refusing everything.
	w := populated(t)
	if err := w.UnmarshalBinary(valid); err != nil {
		t.Fatalf("the unspoiled form was refused: %v", err)
	}
	if got, want := snap(w), snap(src); !equalState(got, want) {
		t.Errorf("the valid form decoded to\n %+v\nwant\n %+v", got, want)
	}
}

// TestUnmarshalRefusesEveryVersionButTheCurrentOne: one version is defined
// and there is no migration path, so every other value of the first byte is
// an error rather than something to interpret.
//
// REFUSING 19 IS THE SHARPEST OF THEM ALL, sharper than refusing 18 was,
// because a version-19 buffer is the one that would very nearly parse. Its
// header, its three planes, its relation and every offset inside its FIRST
// record — indeed every byte of its group section's first two fields — are
// this version's exactly; the section's own record is nine bytes narrower, so
// the second record and every one after it would be read out of the middle of
// its neighbour while the first record's owner and group survived intact. And
// a version-19 form let through would make a tick answer differently even
// where it did parse: this story's order and commanded cell would then be
// read out of the script section's own volatile half, and a form carrying
// neither is not a gap but the claim that every group's order is whatever the
// decoder happens to invent.
func TestUnmarshalRefusesEveryVersionButTheCurrentOne(t *testing.T) {
	src := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 0, X: 1, Y: 1}})
	valid, err := src.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	for v := 0; v < 256; v++ {
		var w World
		gotErr := w.UnmarshalBinary(withByte(valid, 0, byte(v)))
		if v == int(formatVersion) {
			if gotErr != nil {
				t.Errorf("version %d refused: %v", formatVersion, gotErr)
			}
			continue
		}
		if gotErr == nil {
			t.Errorf("version %d accepted", v)
		}
	}
}

// TestUnmarshalRefusesAWellFormedOlderStream: the sweep above spoils a
// version-4 body's first byte, so it never shows a REAL older stream being
// refused — version 1's 29-byte header and 21-byte records, version 2's 29-byte
// header and 25-byte records, or version 3's 34-byte header, grid and 26-byte
// records with no route section at all. All the streams here are well formed for
// the version they declare, written out by hand from the layout that shipped it.
//
// The version-4 one is the sharpest of them, and it is sharp in a way the
// version-3 case no longer is. THIS DECODER'S LENGTH ARITHMETIC ACCEPTS IT. Its
// one entity holds a one-cell route, so at version 5's wider record the 34 bytes
// taken for that record swallow the route count and the cell's x — read as a
// health pair of 1 and 2 — and the cell's y is left over as a route count of
// zero, which consumes the remainder exactly. Every length test, the declared
// count, the presence byte, the stall rule, the ascending ids and the route
// section all pass. Only the version byte refuses it, which is what makes this
// the case that measures the refusal rather than the arithmetic.
//
// What such a stream does not say is what health its units carry, and "none" is
// a claim about the simulation — a unit with no health system is alive, immune
// to damage and still killable — not a gap for whoever reads the file to fill.
// The version-3 case makes the same point about routes and is kept for it.
//
// The header-only streams are the discriminating ones: their bodies are empty,
// so the length division and the count check would both pass them — an empty
// body divides by anything, and zero records match a zero count. Their refusal
// can come from the version byte alone, ahead of every length test. A version-2
// header is worth its own case for a second reason: at 29 bytes it is SHORTER
// than a version-3 header, so a decoder that reached for the mode byte or the
// grid count before checking the version would read past it.
func TestUnmarshalRefusesAWellFormedOlderStream(t *testing.T) {
	headerOnly := []byte{
		0x01,                                           // version 1
		0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 5
		0x2a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // rng state 42
		0x08, 0x00, 0x00, 0x00, // bounds.Width 8
		0x06, 0x00, 0x00, 0x00, // bounds.Height 6
		0x00, 0x00, 0x00, 0x00, // entity count 0
	}
	oneEntity := []byte{
		0x01,                                           // version 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 0
		0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // rng state 7
		0x08, 0x00, 0x00, 0x00, // bounds.Width 8
		0x06, 0x00, 0x00, 0x00, // bounds.Height 6
		0x01, 0x00, 0x00, 0x00, // entity count 1

		// one shipped 21-byte record: no class field, presence at +20
		0x02, 0x00, 0x00, 0x00, // id 2
		0x03, 0x00, 0x00, 0x00, // X 3
		0x04, 0x00, 0x00, 0x00, // Y 4
		0x00, 0x00, 0x00, 0x00, // TargetX 0
		0x00, 0x00, 0x00, 0x00, // TargetY 0
		0x00, // HasTarget false
	}

	v2HeaderOnly := []byte{
		0x02,                                           // version 2
		0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 9
		0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // rng state 17
		0x08, 0x00, 0x00, 0x00, // bounds.Width 8
		0x06, 0x00, 0x00, 0x00, // bounds.Height 6
		0x00, 0x00, 0x00, 0x00, // entity count 0
	}
	v2OneEntity := []byte{
		0x02,                                           // version 2
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 0
		0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // rng state 7
		0x08, 0x00, 0x00, 0x00, // bounds.Width 8
		0x06, 0x00, 0x00, 0x00, // bounds.Height 6
		0x01, 0x00, 0x00, 0x00, // entity count 1

		// one shipped 25-byte record: a class field, presence at +24, no stall
		0x02, 0x00, 0x00, 0x00, // id 2
		0x03, 0x00, 0x00, 0x00, // X 3
		0x04, 0x00, 0x00, 0x00, // Y 4
		0x00, 0x00, 0x00, 0x00, // TargetX 0
		0x00, 0x00, 0x00, 0x00, // TargetY 0
		0x05, 0x00, 0x00, 0x00, // Class 5
		0x00, // HasTarget false
	}

	// A whole version-3 world: the 34-byte header this build still uses, a
	// 2x2 grid, and one 26-byte record — and nothing after it, because version 3
	// had nothing after it.
	v3OneEntity := []byte{
		0x03,                                           // version 3
		0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 2
		0x0d, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // rng state 13
		0x02, 0x00, 0x00, 0x00, // bounds.Width 2
		0x02, 0x00, 0x00, 0x00, // bounds.Height 2
		0x01, 0x00, 0x00, 0x00, // entity count 1
		0x00,                   // routing mode: canonical
		0x04, 0x00, 0x00, 0x00, // grid cell count 4

		0x00, 0x00, 0x00, 0x00, // the grid

		0x07, 0x00, 0x00, 0x00, // id 7
		0x01, 0x00, 0x00, 0x00, // X 1
		0x00, 0x00, 0x00, 0x00, // Y 0
		0x00, 0x00, 0x00, 0x00, // TargetX 0
		0x01, 0x00, 0x00, 0x00, // TargetY 1
		0x00, 0x00, 0x00, 0x00, // Class 0
		0x01, // HasTarget
		0x02, // stall count 2
	}

	// A whole version-4 world: the 34-byte header, a 4x4 all-passable grid, one
	// 26-byte record, and the route section version 4 introduced — a count of one
	// and the cell (2,0), which is that unit's own target. Well formed for
	// version 4 in every particular, and 88 bytes, which is exactly the length
	// version 5 would read as a 34-byte record with a 4-byte route section after
	// it. See the note above.
	v4OneEntity := []byte{
		0x04,                                           // version 4
		0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // tick 3
		0x15, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // rng state 21
		0x04, 0x00, 0x00, 0x00, // bounds.Width 4
		0x04, 0x00, 0x00, 0x00, // bounds.Height 4
		0x01, 0x00, 0x00, 0x00, // entity count 1
		0x00,                   // routing mode: canonical
		0x10, 0x00, 0x00, 0x00, // grid cell count 16

		0x00, 0x00, 0x00, 0x00, // grid row 0
		0x00, 0x00, 0x00, 0x00, // grid row 1
		0x00, 0x00, 0x00, 0x00, // grid row 2
		0x00, 0x00, 0x00, 0x00, // grid row 3

		0x07, 0x00, 0x00, 0x00, // id 7
		0x01, 0x00, 0x00, 0x00, // X 1
		0x00, 0x00, 0x00, 0x00, // Y 0
		0x02, 0x00, 0x00, 0x00, // TargetX 2
		0x00, 0x00, 0x00, 0x00, // TargetY 0
		0x00, 0x00, 0x00, 0x00, // Class 0
		0x01, // HasTarget
		0x02, // stall count 2

		0x01, 0x00, 0x00, 0x00, // id 7's route: one cell
		0x02, 0x00, 0x00, 0x00, // (2,
		0x00, 0x00, 0x00, 0x00, //  0) — its own target
	}

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"version 1, header alone, zero entities", headerOnly},
		{"version 1, one 21-byte record", oneEntity},
		{"version 2, header alone, zero entities", v2HeaderOnly},
		{"version 2, one 25-byte record", v2OneEntity},
		{"version 3, a whole world with no route section", v3OneEntity},
		{"version 4, a whole world whose length version 5 also accepts", v4OneEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := populated(t)
			before := snap(w)
			if err := w.UnmarshalBinary(tc.data); err == nil {
				t.Fatalf("accepted the older stream % x", tc.data)
			}
			if got := snap(w); !equalState(got, before) {
				t.Errorf("the receiver changed:\n before %+v\n after  %+v", before, got)
			}
		})
	}

	// The version-4 stream's length COINCIDENCE IS GONE at version 6 and stays gone
	// at versions 7 and 8, and that is
	// recorded rather than quietly dropped. Its 88 bytes are a 34-byte header, a
	// 16-cell grid, one 43-byte record and 3 bytes left, where a route count needs
	// 4 — so this decoder's arithmetic turns it away as well as its version byte
	// does. Its case above still witnesses that a well-formed older stream is
	// refused; it no longer isolates WHICH check refuses it.
	if err := (&World{}).UnmarshalBinary(withByte(v4OneEntity, 0, formatVersion)); err == nil {
		t.Errorf("the version-4 stream's length is still one this build accepts — " +
			"the note above about the coincidence being gone is then wrong")
	}

	// What isolates the version byte instead is a body well formed for THIS
	// version and wrong in its first byte alone: every length test, the declared
	// count, the presence byte, the stall rule, the domain and the route section
	// pass, and only the version refuses. The same bytes unspoiled are accepted,
	// which is what makes the pair a measurement of the version check.
	src := mustWorld(t, 5, Bounds{Width: 4, Height: 4}, []Entity{{ID: 7, X: 1, Y: 0, Domain: DomainAir}})
	current, err := src.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if err := (&World{}).UnmarshalBinary(withByte(current, 0, 5)); err == nil {
		t.Errorf("a version-6 body carrying the version byte 5 was accepted")
	}
	if err := (&World{}).UnmarshalBinary(current); err != nil {
		t.Errorf("the same body unspoiled was refused: %v — the pair then measures "+
			"something other than the version byte", err)
	}
}

// ---------------------------------------------------------------------------
// The owner slot's own byte-form witnesses: the two derivations that can still
// fail once the transcriptions and their digests agree with each other.
// ---------------------------------------------------------------------------

// preOwnerFormVersion, preOwnerPinDigest and preOwnerRoutedDigest are the
// version byte and the two digests the pinned worlds had before the owner slot
// existed. They are pasted from the pre-story tree and are the fixed point the
// two derivations below are measured against.
const (
	preOwnerFormVersion  byte   = 11
	preOwnerPinDigest    uint64 = 0xb56c622a377738cf
	preOwnerRoutedDigest uint64 = 0x5e1a6c6e7ea13679
)

// preCostFormVersion and the two digests below are the version byte and the two
// digests the pinned worlds carried before the COST and HEIGHT planes existed.
// They are pasted from the pre-story tree and are the fixed point the derivation
// after them is measured against.
const (
	preCostFormVersion  byte   = 12
	preCostPinDigest    uint64 = 0x140dc9a829f7e3dd
	preCostRoutedDigest uint64 = 0xc3ecca188ad60e2d
)

// preSightFormVersion and the two digests below are the version byte and the two
// digests the pinned worlds carried before THE SIGHT RANGE existed — which is
// the tree as VERSION 17 left it, not as version 16 did. The two stories were
// written in parallel off one master and 0089 landed first, so "pre-sight" is
// 0089's post-landing state and these are its own pinned numbers, pasted from
// the tree this branch merged.
//
// This is the NEWEST peel, so the decay peel and every peel behind it now run on
// its output.
const (
	preSightFormVersion  byte   = 17
	preSightPinDigest    uint64 = 0xb942b5d55f697fac
	preSightRoutedDigest uint64 = 0x5e8ef7f50db81ea7
)

// strippedOfTheSight is this story's inverse over one transcribed form: remove
// the ONE byte each record grew by and put the version byte back.
//
// The removals run BACKWARDS over the records, so each one leaves the earlier
// records where they are. The base is past three planes and the width and the
// offset are written out from the contract, not read off entityLen, so a record
// that widens again fails here rather than following the constant.
func strippedOfTheSight(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 100*i
		out = append(out[:o+99], out[o+100:]...)
	}
	out[0] = preSightFormVersion
	return out
}

// TestThePinIsThePreStoryPinPlusTheSightRange is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its byte in the
// wrong place, both would move together and neither would say so. What cannot be
// faked is the pin read backwards onto a literal that predates the story — and
// here the literal predates it by ONE STORY rather than by the whole history,
// which is what makes this peel the sharpest of them: 0089's numbers were pinned
// before this branch merged, so reaching them proves that everything 0089 wrote
// is exactly where 0089 put it and that this story added one byte per record and
// nothing else.
func TestThePinIsThePreStoryPinPlusTheSightRange(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", pinBytesPreGroups, 20, 3, preSightPinDigest},
		{"the routed world", rtfBytesPreGroups, 16, 3, preSightRoutedDigest},
	} {
		stripped := strippedOfTheSight(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its sight ranges is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its sight ranges removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the one per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// preDecayFormVersion and the two digests below are the version byte and the two
// digests the pinned worlds carried before THE DECAY LADDER existed. They are
// pasted from the pre-story tree, and every derivation below them is measured
// through this one.
const (
	preDecayFormVersion  byte   = 16
	preDecayPinDigest    uint64 = 0x047c5d3c0b15a2d0
	preDecayRoutedDigest uint64 = 0x8ac12c66c9a2fd56
)

// strippedOfTheDecayBlock is this story's inverse over one transcribed form:
// remove the SEVEN bytes each record grew by and put the version byte back.
//
// The removals run BACKWARDS over the records, so each one leaves the earlier
// records where they are. The base is past three planes and the width and the
// offset are written out from the contract, not read off entityLen, so a record
// that widens again fails here rather than following the constant.
func strippedOfTheDecayBlock(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 99*i
		out = append(out[:o+92], out[o+99:]...)
	}
	out[0] = preDecayFormVersion
	return out
}

// TestThePinIsThePreStoryPinPlusTheDecayBlock is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its seven bytes
// in the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the seven each record grew by, with
// the version byte put back, must be the form the tree carried before it.
func TestThePinIsThePreStoryPinPlusTheDecayBlock(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", strippedOfTheSight(pinBytesPreGroups, 20, 3), 20, 3, preDecayPinDigest},
		{"the routed world", strippedOfTheSight(rtfBytesPreGroups, 16, 3), 16, 3, preDecayRoutedDigest},
	} {
		stripped := strippedOfTheDecayBlock(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 7*tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its decay blocks is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its decay blocks removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the seven per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// preRelationFormVersion and the two digests below are the version byte and the
// two digests the pinned worlds carried before THE RELATION existed. They are
// pasted from the pre-story tree and are the fixed point the derivation after
// them is measured against.
const (
	preRelationFormVersion  byte   = 14 // 15 went to another lane; this build never wrote one
	preRelationPinDigest    uint64 = 0xcc2e3f970c53ff2e
	preRelationRoutedDigest uint64 = 0x26357be9f95bb0d4
)

// strippedOfTheRelation is this story's inverse over one transcribed form: drop
// the fixed block the form now ends with and put the version byte back.
//
// It is the NEWEST peel, so every backwards derivation that predates this story
// runs on its output — each strips one story at a time, newest first. It is also
// the SHORTEST of them, and that is the whole content of where this story put
// its bytes: the block closes the form and nothing in front of it moved, so the
// inverse is a truncation rather than a splice. A byte written anywhere else
// would survive this cut and be caught by the peel behind it.
func strippedOfTheRelation(form []byte) []byte {
	out := append([]byte(nil), form[:len(form)-relationLen]...)
	out[0] = preRelationFormVersion
	return out
}

// TestThePinIsThePreStoryPinPlusTheRelation is the derivation for this story: cut
// the block back off the two transcribed forms and the pre-story digests come
// back exactly. Nothing else in the package can fail once this story's new
// digests and its new transcriptions agree with each other — they agree by
// construction, and a byte written in the wrong place would move both together.
func TestThePinIsThePreStoryPinPlusTheRelation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		form []byte
		want uint64
	}{
		{"the pinned world", strippedOfTheDecayBlock(strippedOfTheSight(pinBytesPreGroups, 20, 3), 20, 3), preRelationPinDigest},
		{"the routed world", strippedOfTheDecayBlock(strippedOfTheSight(rtfBytesPreGroups, 16, 3), 16, 3), preRelationRoutedDigest},
	} {
		stripped := strippedOfTheRelation(tc.form)
		if want := len(tc.form) - relationLen; len(stripped) != want {
			t.Errorf("%s stripped of its relation is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its relation removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the block that closes the form", tc.what, got, tc.want)
		}
	}
}

// preFacingFormVersion and the two digests below are the version byte and the two
// digests the pinned worlds carried before the FACING existed. They are pasted
// from the pre-story tree and are the fixed point the derivation after them is
// measured against.
const (
	preFacingFormVersion  byte   = 13
	preFacingPinDigest    uint64 = 0x2a12cf3dcfcd82fc
	preFacingRoutedDigest uint64 = 0xea98475193d1cb4c
)

// strippedOfTheFacing is this story's inverse over one transcribed form: remove
// the ONE byte each record grew by and put the version byte back.
//
// It is the NEWEST peel, so every backwards derivation that predates this story
// now runs on its output — each strips one story at a time, newest first, which
// is the arrangement the plane peel below established and the reason this
// function goes in front of it rather than inside it.
//
// The removals run BACKWARDS over the records, so each one leaves the earlier
// records where they are. The base is past THREE planes and the width and the
// offset are written out from the contract, not read off entityLen, so a record
// that widens again fails here rather than following the constant.
func strippedOfTheFacing(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 92*i
		out = append(out[:o+91], out[o+92:]...)
	}
	out[0] = preFacingFormVersion
	return out
}

// TestThePinIsThePreStoryPinPlusTheFacing is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its byte in the
// wrong place, both would move together and neither would say so. What cannot be
// faked is the pin read backwards onto a literal that predates the story — this
// story's bytes, minus exactly the one byte each record grew by, with the version
// byte put back, must be the form the tree carried before it.
//
// It matters more here than it did for any field before it, because NOTHING IN
// THIS BUILD READS A FACING: every behavioural test in the tree would still pass
// with the field dropped between the constructor and the encoder, and every
// digest that moved would have moved anyway.
func TestThePinIsThePreStoryPinPlusTheFacing(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(pinBytesPreGroups, 20, 3), 20, 3)), 20, 3, preFacingPinDigest},
		{"the routed world", strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(rtfBytesPreGroups, 16, 3), 16, 3)), 16, 3, preFacingRoutedDigest},
	} {
		stripped := strippedOfTheFacing(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its facings is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its facings removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the one per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// TestTwoWorldsDifferingOnlyInAFacingHashDifferently is the assertion that does
// not care what any digest IS, and it is the one this story most needs: no tick
// reads the field, so a facing lost between the constructor and the encoder would
// leave every behavioural test in the tree green.
func TestTwoWorldsDifferingOnlyInAFacingHashDifferently(t *testing.T) {
	t.Parallel()

	build := func(f uint8) *World {
		return mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{
			{ID: 1, X: 1, Y: 1, HP: 5, MaxHP: 5, Facing: f},
			{ID: 2, X: 2, Y: 2, HP: 5, MaxHP: 5, Facing: 0x40},
		})
	}
	a, b := build(0x20), build(0x60)
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in one entity's facing hash alike (%#016x) — "+
			"the field does not reach the byte form", a.Hash())
	}
	if bytes.Equal(mustMarshal(t, a), mustMarshal(t, b)) {
		t.Errorf("two worlds differing only in one entity's facing marshal to the same bytes")
	}
	if build(0x20).Hash() != a.Hash() {
		t.Errorf("one world built twice hashes differently")
	}
	// EVERY BYTE ROUND-TRIPS, including the 248 this build never writes: the
	// field is carried whole and refused nowhere, so a decoder that rounded one
	// to a direction, or a constructor that folded one, fails here.
	for v := 0; v < 256; v++ {
		w := build(uint8(v))
		var got World
		if err := got.UnmarshalBinary(mustMarshal(t, w)); err != nil {
			t.Fatalf("facing %d: UnmarshalBinary: %v", v, err)
		}
		if back := got.Entities()[0].Facing; back != uint8(v) {
			t.Errorf("facing %d read back as %d", v, back)
		}
	}
}

// strippedOfThePlanes is this story's inverse over one transcribed form: cut the
// two plane sections out from behind the block plane and put the version byte
// back.
//
// It is the ONLY check in the package that can still fail once this story's new
// digests and its new transcriptions agree with each other — they agree by
// construction, and a byte written in the wrong place would move both together.
// Run backwards onto a literal that predates the story, a byte written anywhere
// but in those two plane-shaped holes disagrees.
//
// Every backwards derivation that predates this story now runs on ITS OUTPUT
// rather than on the form directly, which is what keeps them measuring their own
// stories: each strips one story at a time, newest first.
func strippedOfThePlanes(form []byte, cells int) []byte {
	out := append([]byte(nil), form[:34+cells]...)
	out = append(out, form[34+3*cells:]...)
	out[0] = preCostFormVersion
	return out
}

func TestThePinIsThePreStoryPinPlusTheTwoPlanes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what  string
		form  []byte
		cells int
		want  uint64
	}{
		{"the pinned world", strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(pinBytesPreGroups, 20, 3), 20, 3)), 20, 3), 20, preCostPinDigest},
		{"the routed world", strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(rtfBytesPreGroups, 16, 3), 16, 3)), 16, 3), 16, preCostRoutedDigest},
	} {
		stripped := strippedOfThePlanes(tc.form, tc.cells)
		if want := len(tc.form) - 2*tc.cells; len(stripped) != want {
			t.Errorf("%s stripped of its planes is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its two planes removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside them", tc.what, got, tc.want)
		}
	}
}

// strippedOfTheOwner is this story's inverse over one transcribed form: remove
// the four bytes each record grew by and put the version byte back.
//
// The removals run BACKWARDS over the records, so each one leaves the earlier
// records where they are. The offset and the width are written out from the
// contract, not read off entityLen, so a record that widens again fails here
// rather than following the constant.
func strippedOfTheOwner(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + cells
	for i := records - 1; i >= 0; i-- {
		o := base + 91*i
		out = append(out[:o+87], out[o+91:]...)
	}
	out[0] = preOwnerFormVersion
	return out
}

// TestThePinIsThePreStoryPinPlusTheOwnerWord is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained a byte in the
// wrong place, both would move together and neither would say so. What cannot be
// faked is the pin read backwards onto a literal that predates the story — this
// story's bytes, minus exactly the four each record grew by, with the version
// byte put back, must be the form the tree carried before it.
//
// If a byte was written anywhere but in those three four-byte holes, this
// disagrees, and it is the only check in the package that would.
func TestThePinIsThePreStoryPinPlusTheOwnerWord(t *testing.T) {
	t.Parallel()

	stripped := strippedOfTheOwner(strippedOfThePlanes(strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(pinBytesPreGroups, 20, 3), 20, 3)), 20, 3), 20), 20, 3)
	if want := 34 + 20 + 3*87 + 3*4 + scriptStateLen + scriptCountsLen; len(stripped) != want {
		t.Fatalf("the stripped form is %d byte(s), want the pre-story %d", len(stripped), want)
	}
	if got := fnv1a(stripped); got != preOwnerPinDigest {
		t.Errorf("the pin with its owner slots removed hashes %#016x, want the pre-story %#016x — "+
			"this story wrote a byte outside the three it was supposed to add",
			got, preOwnerPinDigest)
	}
}

// TestTheRoutedPinIsThePreStoryPinPlusTheOwnerWord is the same derivation over
// the second pinned fixture, which carries stored ROUTE CELLS after its records.
// An owner written four bytes wide but in the wrong place would leave the first
// pin's arithmetic intact and this one's broken, because here the records are not
// the last thing in the form.
func TestTheRoutedPinIsThePreStoryPinPlusTheOwnerWord(t *testing.T) {
	t.Parallel()

	if got := fnv1a(strippedOfTheOwner(strippedOfThePlanes(strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(rtfBytesPreGroups, 16, 3), 16, 3)), 16, 3), 16), 16, 3)); got != preOwnerRoutedDigest {
		t.Errorf("the routed pin with its owner slots removed hashes %#016x, want the pre-story "+
			"%#016x — this story wrote a byte outside the three it was supposed to add",
			got, preOwnerRoutedDigest)
	}
}

// TestTwoWorldsDifferingOnlyInAnOwnerHashDifferently is the assertion that does
// not care what any digest IS.
//
// Every pinned digest in this package moved with this story, and a digest that
// changed for the right reason looks exactly like one that changed for the wrong
// one. Worse, NOTHING IN THIS BUILD READS AN OWNER, so every behavioural test in
// the tree would still pass with the field dropped between the constructor and
// the encoder. This is what would notice.
func TestTwoWorldsDifferingOnlyInAnOwnerHashDifferently(t *testing.T) {
	t.Parallel()

	build := func(o uint32) *World {
		return mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{
			{ID: 1, X: 1, Y: 1, HP: 5, MaxHP: 5, Owner: o},
			{ID: 2, X: 2, Y: 2, HP: 5, MaxHP: 5, Owner: 9},
		})
	}
	a, b := build(3), build(4)
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in one entity's owner hash alike (%#016x) — "+
			"the field does not reach the byte form", a.Hash())
	}
	if bytes.Equal(mustMarshal(t, a), mustMarshal(t, b)) {
		t.Errorf("two worlds differing only in one entity's owner marshal to the same bytes")
	}
	// And the same world twice is still the same world, so the difference above
	// is the owner and not the encoder being unstable.
	if build(3).Hash() != a.Hash() {
		t.Errorf("one world built twice hashes differently")
	}
	if build(0).Hash() == build(1).Hash() {
		t.Errorf("a world of unowned units hashes like one owned by roster slot 1")
	}
}

func TestADeadUnitKeepsItsOwnerThroughTheForm(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 0, MaxHP: 10, Owner: 6},
		{ID: 2, X: 2, Y: 2, HP: -40, MaxHP: 10, Owner: 4294967295},
	})
	for _, e := range w.Entities() {
		if e.Alive() {
			t.Fatalf("entity %d is alive; this fixture is about units that are not", e.ID)
		}
	}
	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatalf("a form carrying two owned corpses was refused: %v", err)
	}
	want := map[EntityID]uint32{1: 6, 2: 4294967295}
	for _, e := range back.Entities() {
		if e.Owner != want[e.ID] {
			t.Errorf("entity %d comes back carrying owner %d, want %d", e.ID, e.Owner, want[e.ID])
		}
	}
}

// ---------------------------------------------------------------------------
// The regeneration block's own byte-form witness (0109): the one derivation
// that can still fail once this story's new digests and its new
// transcriptions agree with each other.
// ---------------------------------------------------------------------------

// preRegenFormVersion is the version byte the form carried before THIS
// STORY'S regeneration block existed — version 24, 0106's own landed state.
//
// strippedOfTheRegen is the inverse of the widening that added it: one form
// with the eighteen bytes each record grew by cut off its tail and the
// version byte put back. It is no longer the newest peel — 0112's own carry
// and purse sections are — but it keeps its place ahead of strippedOfThePost
// and everything below it: the strippings come off newest story first,
// exactly as the post's own peel displaced the reach's when IT arrived.
const preRegenFormVersion byte = 24

// strippedOfTheRegen removes the EIGHTEEN bytes each record grew by: every
// entity either pinned fixture builds names no period and no mana pool, so
// the constructor leaves all six fields at their zero value, and this peel's
// own bytes are always eighteen zeros regardless of which entity it was. The
// offset is written out from the contract, not read off entityLen, on
// strippedOfThePost's own rule for its tail.
func strippedOfTheRegen(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 145*i
		out = append(out[:o+127], out[o+145:]...)
	}
	out[0] = preRegenFormVersion
	return out
}

// preRegenPinDigest and preRegenRoutedDigest are the two digests the pinned
// worlds carried before THIS STORY'S regeneration block existed — version
// 24's pinDigest and rtfDigest, pasted from the pre-story tree. Stripping
// the regeneration block this story added must fall back to exactly these,
// which is TestThePinIsThePreStoryPinPlusTheRegen's whole content.
const (
	preRegenPinDigest    uint64 = 0x612dd5f8c12ccb31
	preRegenRoutedDigest uint64 = 0x3f5b7e651f1b6c84
)

// TestThePinIsThePreStoryPinPlusTheRegen is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its block
// in the wrong place, both would move together and neither would say so.
// What cannot be faked is the pin read backwards onto a literal that
// predates the story — this story's bytes, minus exactly the eighteen
// bytes each record grew by, with the version byte put back, must be the
// form the tree carried before it.
func TestThePinIsThePreStoryPinPlusTheRegen(t *testing.T) {
	t.Parallel()

	// 0125's own experience tail, then 0117's own command group tail, then
	// 0112's own carry section together with 0124's own equipment section
	// and the purse — newest peels first, on the same rule every peel here
	// already follows: the strippings come off newest story first. The
	// equipment section sits between the carry section and the purse, so
	// one cut of the combined width removes all three at once, exactly as
	// TestThePinIsThePreStoryPinPlusTheCarryAndPurse's own cut does.
	noCarryPin := strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 20, 3), 34+3*20+3*145+3*4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen)
	noCarryRtf := strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(rtfBytesPreSpells, 16, 3), 16, 3), 34+3*16+3*145+(4+2*8)+4+4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen)

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", noCarryPin, 20, 3, preRegenPinDigest},
		{"the routed world", noCarryRtf, 16, 3, preRegenRoutedDigest},
	} {
		stripped := strippedOfTheRegen(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 18*tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its regeneration blocks is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its regeneration blocks removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the eighteen per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// TestADecodedRegenerationRecordCrossesTheFormWholeAtBothExtremes is AC-6's
// first arm at the field's own edges. The four int32 are carried WHOLE — no
// value refused, folded or clamped — at both int32 extremes, and the two
// remainders at 0 and 99, the whole of their legal range; all six are
// swapped between the two cases so a field read out of its neighbour fails
// here too.
//
// THIS IS THE FOUR INT32 FIELDS' ONLY WITNESS. No construction path this
// task builds ever writes a nonzero mana pool or period — that is a later
// task's own spawn wiring — so a built world's four are always zero and
// could round-trip correctly even with the marshal write missing entirely,
// the digest and every other pinned form moving for an unrelated reason (the
// version bump) and hiding the gap. Only a decode that names nonzero values,
// re-marshalled and compared byte for byte, can catch that — the same gap
// TestADecodedCommandedCellCrossesTheFormWholeAtBothExtremes
// (guardradius_test.go) closes for the commanded cell.
func TestADecodedRegenerationRecordCrossesTheFormWholeAtBothExtremes(t *testing.T) {
	valid, err := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}}).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// The record's own tail, written out from the contract rather than read
	// off entityLen for the offsets themselves: entityLen now measures past
	// the whole record, which is what a caller uses it for elsewhere in this
	// file, not what names one field's own place inside it.
	manaAt := headerLen + 3*16 + 127
	maxManaAt := manaAt + 4
	healthPeriodAt := maxManaAt + 4
	manaPeriodAt := healthPeriodAt + 4
	healthRemAt := manaPeriodAt + 4
	manaRemAt := healthRemAt + 1

	for _, tc := range []struct {
		mana, maxMana, healthPeriod, manaPeriod int32
		healthRem, manaRem                      uint8
	}{
		{math.MinInt32, math.MaxInt32, math.MinInt32, math.MaxInt32, 0, 99},
		{math.MaxInt32, math.MinInt32, math.MaxInt32, math.MinInt32, 99, 0},
	} {
		spoiled := valid
		spoiled = withU32(spoiled, manaAt, uint32(tc.mana))
		spoiled = withU32(spoiled, maxManaAt, uint32(tc.maxMana))
		spoiled = withU32(spoiled, healthPeriodAt, uint32(tc.healthPeriod))
		spoiled = withU32(spoiled, manaPeriodAt, uint32(tc.manaPeriod))
		spoiled = withByte(spoiled, healthRemAt, tc.healthRem)
		spoiled = withByte(spoiled, manaRemAt, tc.manaRem)

		var w World
		if err := w.UnmarshalBinary(spoiled); err != nil {
			t.Fatalf("a regeneration record of %+v was refused: %v", tc, err)
		}
		got := w.Entities()[0]
		if got.Mana != tc.mana || got.MaxMana != tc.maxMana || got.HealthRegenPeriod != tc.healthPeriod ||
			got.ManaRegenPeriod != tc.manaPeriod || got.HealthHundredths != tc.healthRem || got.ManaHundredths != tc.manaRem {
			t.Fatalf("decoded entity is %+v, want the spoiled record %+v", got, tc)
		}
		again, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		if !bytes.Equal(again, spoiled) {
			t.Errorf("%+v: re-marshalling gave\n % x\nwant\n % x", tc, again, spoiled)
		}
	}
}

func TestUnmarshalRefusesARegenRemainderAboveNinetyNine(t *testing.T) {
	valid, err := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}}).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	healthRemAt := headerLen + 3*16 + 143
	manaRemAt := healthRemAt + 1

	for _, tc := range []struct {
		what     string
		form     []byte
		accepted bool
	}{
		{"a health remainder of 100", withByte(valid, healthRemAt, 100), false},
		{"a health remainder of 255", withByte(valid, healthRemAt, 255), false},
		{"a mana remainder of 100", withByte(valid, manaRemAt, 100), false},
		{"a mana remainder of 255", withByte(valid, manaRemAt, 255), false},
		{"a health remainder of 99, the top of its range", withByte(valid, healthRemAt, 99), true},
		{"a mana remainder of 99, the top of its range", withByte(valid, manaRemAt, 99), true},
	} {
		var w World
		err := w.UnmarshalBinary(tc.form)
		if tc.accepted && err != nil {
			t.Errorf("%s was refused: %v", tc.what, err)
		}
		if !tc.accepted && err == nil {
			t.Errorf("%s was accepted", tc.what)
		}
	}
	// And the unspoiled form is accepted, so the cases above measure the
	// remainder check and not something else about this fixture.
	var w World
	if err := w.UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

// ---------------------------------------------------------------------------
// The post's own byte-form witness (0106): the one derivation that can still
// fail once this story's new digests and its new transcriptions agree with
// each other.
// ---------------------------------------------------------------------------

// prePostFormVersion is the version byte the form carried before THIS
// STORY'S post tail existed — version 23, 0104's own landed state.
//
// strippedOfThePost is the inverse of the widening that added it: one form
// with the eight bytes each record grew by cut off its tail and the version
// byte put back. It is no longer the newest peel — 0112's own carry and
// purse sections are, ahead of strippedOfTheRegen above — but it keeps its
// place ahead of strippedOfTheReach and everything below it: the strippings
// come off newest story first, exactly as the reach's own peel displaced
// the sack section's when IT arrived.
const prePostFormVersion byte = 23

// strippedOfThePost removes the EIGHT bytes each record grew by: every
// entity either pinned fixture builds carries its own X, Y as its post —
// NewWorld writes it from the entity's own cell unconditionally — so this
// peel's own bytes are always the coordinate pair already pinned earlier in
// the same record, and nothing about which entity it was varies the check.
// The offset is written out from the contract, not read off entityLen, on
// strippedOfTheReach's own rule for its tail.
func strippedOfThePost(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 127*i
		out = append(out[:o+119], out[o+127:]...)
	}
	out[0] = prePostFormVersion
	return out
}

// prePostPinDigest and prePostRoutedDigest are the two digests the pinned
// worlds carried before THIS STORY'S post tail existed — version 23's
// pinDigest and rtfDigest, pasted from the pre-story tree. Stripping the
// post tail this story added must fall back to exactly these, which is
// TestThePinIsThePreStoryPinPlusThePost's whole content.
const (
	prePostPinDigest    uint64 = 0xc400c725bab385c3
	prePostRoutedDigest uint64 = 0x030da70bda8dd508
)

// TestThePinIsThePreStoryPinPlusThePost is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its pair in
// the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the eight bytes each record grew
// by, with the version byte put back, must be the form the tree carried
// before it.
func TestThePinIsThePreStoryPinPlusThePost(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 20, 3), 34+3*20+3*145+3*4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 20, 3), 20, 3, prePostPinDigest},
		{"the routed world", strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(rtfBytesPreSpells, 16, 3), 16, 3), 34+3*16+3*145+(4+2*8)+4+4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 16, 3), 16, 3, prePostRoutedDigest},
	} {
		stripped := strippedOfThePost(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 8*tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its post tails is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its post tails removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the eight per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The reach's own byte-form witness (0104): the one derivation that can still
// fail once this story's new digests and its new transcriptions agree with
// each other.
// ---------------------------------------------------------------------------

// preReachFormVersion is the version byte the form carried before THIS
// STORY'S reach tail existed — version 22, 0103's own landed state.
//
// strippedOfTheReach is the inverse of the widening that added it: one form
// with the one byte each record grew by cut off its tail and the version
// byte put back. It is no longer the newest peel — 0112's own carry and
// purse sections are, ahead of 0106's own post tail and 0109's own regen
// tail, above it — but it keeps its place ahead of strippedOfTheSacks and
// everything below it: the strippings come off newest story first, exactly
// as the sack section's own peel displaced the actor state's when IT
// arrived.
const preReachFormVersion byte = 22

// strippedOfTheReach removes the ONE byte each record grew by: every entity
// either pinned fixture builds is bare — this task wires no placement to a
// weapon — so the constructor's own floor of 1 is the one byte this peel
// removes from every record, and nothing about which entity it was varies
// the check. The offset is written out from the contract, not read off
// entityLen, on strippedOfTheActorState's own rule for its tail.
func strippedOfTheReach(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 119*i
		out = append(out[:o+118], out[o+119:]...)
	}
	out[0] = preReachFormVersion
	return out
}

// preReachPinDigest and preReachRoutedDigest are the two digests the pinned
// worlds carried before THIS STORY'S reach tail existed — version 22's
// pinDigest and rtfDigest, pasted from the pre-story tree. Stripping the
// reach tail this story added must fall back to exactly these, which is
// TestThePinIsThePreStoryPinPlusTheReach's whole content.
const (
	preReachPinDigest    uint64 = 0xbc1dc5d5eef60d85
	preReachRoutedDigest uint64 = 0x5a8a000b43c044bc
)

// TestThePinIsThePreStoryPinPlusTheReach is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its byte in
// the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the one byte each record grew
// by, with the version byte put back, must be the form the tree carried
// before it.
func TestThePinIsThePreStoryPinPlusTheReach(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 20, 3), 34+3*20+3*145+3*4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 20, 3), 20, 3), 20, 3, preReachPinDigest},
		{"the routed world", strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(rtfBytesPreSpells, 16, 3), 16, 3), 34+3*16+3*145+(4+2*8)+4+4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 16, 3), 16, 3), 16, 3, preReachRoutedDigest},
	} {
		stripped := strippedOfTheReach(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its reach tails is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its reach tails removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the one per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The actor state's own byte-form witness (0099): the one derivation that can
// still fail once this story's new digests and its new transcriptions agree
// with each other.
// ---------------------------------------------------------------------------

// preActorStateFormVersion and the two digests below are the version byte and
// the two digests the pinned worlds carried before THIS STORY'S entity tail
// existed — version 20, 0096's own landed state — pasted from the pre-story
// tree, and the fixed point every OLDER peel in this file, in
// groupform_test.go and in routeform_test.go is now measured through: each of
// those already runs on the output of the peel one story newer than itself,
// and this tail is now the newest one, so pinBytesPreGroups and
// rtfBytesPreGroups below take its place at the head of every one of those
// chains — pushing the group section's own peel down to run on THIS one's
// output rather than on the raw pin.
const (
	preActorStateFormVersion  byte   = 20
	preActorStatePinDigest    uint64 = 0xa40e25c8032db332
	preActorStateRoutedDigest uint64 = 0x54ca89f6d34ff49b
)

// preSackSectionFormVersion is the version byte the form carried before THIS
// STORY'S sack section existed — version 21, 0099's own landed state (0103).
//
// strippedOfTheSacks is the inverse of the widening that added the section:
// one form with the counted block it added between the group section and the
// script section cut out and the version byte put back. It WAS the newest
// peel; 0104's reach tail is, now — see strippedOfTheReach above, which runs
// ahead of this one — but it keeps its place ahead of strippedOfTheActorState
// and everything below it — the strippings come off newest story first,
// exactly as the actor state's own peel displaced the group section's when
// IT arrived.
//
// Unlike strippedOfTheActorState, the section is neither a tail on the
// entity record nor a block closing the form, so the inverse takes the
// section's own offset and length rather than a record count — both are
// properties of the WORLD being measured (how many groups it holds and how
// long its route section ran), on strippedOfTheGroups' own rule, and are
// passed in by the caller rather than read back off the encoder. Every
// fixture in this package builds a world naming no sack, so the section is
// always its own bare four-byte zero count.
const preSackSectionFormVersion byte = 21

func strippedOfTheSacks(form []byte, off, n int) []byte {
	out := append([]byte(nil), form[:off]...)
	out = append(out, form[off+n:]...)
	out[0] = preSackSectionFormVersion
	return out
}

// strippedOfTheActorState is this story's inverse over one transcribed form:
// remove the EIGHTEEN bytes each record grew by and put the version byte
// back.
//
// The removals run BACKWARDS over the records, so each one leaves the earlier
// records where they are. The base is past three planes and the width and the
// offset are written out from the contract, not read off entityLen, so a
// record that widens again fails here rather than following the constant —
// the same rule strippedOfTheSight and strippedOfTheDecayBlock already follow
// for their own tails.
func strippedOfTheActorState(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 118*i
		out = append(out[:o+100], out[o+118:]...)
	}
	out[0] = preActorStateFormVersion
	return out
}

// TestThePinIsThePreStoryPinPlusTheActorState is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its tail in
// the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the eighteen each record grew
// by, with the version byte put back, must be the form the tree carried
// before it. Every entity either fixture builds is guard with an empty ring
// — the only shape NewWorld can produce — so the eighteen bytes this
// peel removes are the same six-field tail on every record, and nothing
// about which entity it was varies the check.
func TestThePinIsThePreStoryPinPlusTheActorState(t *testing.T) {
	t.Parallel()

	// Both fixtures are stripped of 0125's own experience tail, then 0117's
	// own command group tail, then 0112's own carry section together with
	// 0124's own equipment section and the purse, then 0109's regen tail,
	// then 0106's post tail, then 0104's reach tail — each the newest peel
	// in its turn — and then of the sack section, on the same rule the
	// group section's own peel was pushed down the chain when the actor
	// state arrived: the strippings come off newest story first, so what
	// reaches strippedOfTheSacks is a form at its own accustomed width
	// again.
	noSacksPin := strippedOfTheSacks(strippedOfTheReach(strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 20, 3), 34+3*20+3*145+3*4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 20, 3), 20, 3), 20, 3), 34+3*20+3*118+3*4+(4+2*18), sackCountLen)
	noSacksRtf := strippedOfTheSacks(strippedOfTheReach(strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(rtfBytesPreSpells, 16, 3), 16, 3), 34+3*16+3*145+(4+2*8)+4+4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 16, 3), 16, 3), 16, 3), 34+3*16+3*118+(4+2*8)+4+4+(4+2*18), sackCountLen)

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", noSacksPin, 20, 3, preActorStatePinDigest},
		{"the routed world", noSacksRtf, 16, 3, preActorStateRoutedDigest},
	} {
		stripped := strippedOfTheActorState(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 18*tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its actor state tails is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its actor state tails removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the eighteen per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The group section's own byte-form witness: the one derivation that can still
// fail once this story's new digests and its new transcriptions agree with
// each other.
// ---------------------------------------------------------------------------

// preGroupSectionFormVersion and the two digests below are the version byte
// and the two digests the pinned worlds carried before THIS STORY'S group
// section existed — pasted from the pre-story tree, and the fixed point every
// OLDER peel in this file and in groupform_test.go is now measured through:
// each of those already runs on the output of the peel one story newer than
// itself. It is no longer the newest — 0099's actor state tail was, and
// 0103's sack section is now, above both — but it keeps its place in the
// OLDER chain: pinBytesPreGroups and rtfBytesPreGroups below now run it on
// strippedOfTheActorState's output, itself run on strippedOfTheSacks',
// rather than on the raw pin, and everything under it is unaffected.
const (
	preGroupSectionFormVersion  byte   = 18
	preGroupSectionPinDigest    uint64 = 0x7a94e926fd68cc91
	preGroupSectionRoutedDigest uint64 = 0xdb02db658ef7abe7
)

// strippedOfTheGroups is this story's inverse over one transcribed form: cut
// the counted block it added between the routes and the script section, and
// put the version byte back.
//
// Unlike every peel above it, the section is not a tail on the entity record
// and not a block closing the form — it sits in the MIDDLE, so the inverse
// takes the section's own offset and length rather than a cell or record
// count: both are properties of the WORLD being measured (how many groups it
// holds and how long its route section ran), exactly as nstShape's own
// parameters are, and are passed in by the caller rather than read back off
// the encoder.
func strippedOfTheGroups(form []byte, off, n int) []byte {
	out := append([]byte(nil), form[:off]...)
	out = append(out, form[off+n:]...)
	out[0] = preGroupSectionFormVersion
	return out
}

// pinBytesPreGroups and rtfBytesPreGroups chain TWO peels now: 0099's own
// (strippedOfTheActorState, applied first so this story's tail comes off
// before anything older is measured) and then the group section's, exactly as
// it ran before this story. The offsets below are therefore unchanged from
// what they were pre-0099 — the two fixtures' own header, three planes, three
// 100-byte records and route sections, which is what strippedOfTheActorState's
// output is back to: pinBytes carries no route cell at all (3*4), rtfBytes
// carries one 2-cell route and two empty ones ((4+2*8)+4+4) — both written out
// from the contract rather than read back off the forms, on the rule every
// peel here already follows.
// Both then chained THREE peels, for 0103: its own sack section
// (strippedOfTheSacks, applied first so THAT story's section came off
// before anything older was measured), then 0099's actor state, then the
// group section's. Both then chained FOUR, for 0104: its own reach tail
// (strippedOfTheReach) went in ahead of all three. Both then chained FIVE:
// 0106's own post tail (strippedOfThePost) went in ahead of all four, on
// the same rule. Both then chained SIX: 0109's own regeneration block
// (strippedOfTheRegen) went in ahead of all five. Both then chained SEVEN,
// for 0112: its own carry and purse sections (strippedOfTheCarryAndPurse)
// went in ahead of all six. Both then chained EIGHT, for 0117: its own
// command group tail (strippedOfTheCommandGroup) went in ahead of all seven.
// Both then chained NINE, for 0124: its own equipment section rides the
// same carry-and-purse peel as 0112's own sections, on that peel's own
// combined width — the strippings still come off newest story first — and
// nothing below it moves. Both now chain TEN, for 0125: its own experience
// tail (strippedOfTheExperience) goes in ahead of all nine, on the same
// rule.
var (
	pinBytesPreGroups = strippedOfTheGroups(strippedOfTheActorState(
		strippedOfTheSacks(strippedOfTheReach(strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 20, 3), 34+3*20+3*145+3*4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 20, 3), 20, 3), 20, 3), 34+3*20+3*118+3*4+(4+2*18), sackCountLen), 20, 3), 34+3*20+3*100+3*4, 4+2*18)
	rtfBytesPreGroups = strippedOfTheGroups(strippedOfTheActorState(
		strippedOfTheSacks(strippedOfTheReach(strippedOfThePost(strippedOfTheRegen(strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(rtfBytesPreSpells, 16, 3), 16, 3), 34+3*16+3*145+(4+2*8)+4+4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen), 16, 3), 16, 3), 16, 3), 34+3*16+3*118+(4+2*8)+4+4+(4+2*18), sackCountLen), 16, 3), 34+3*16+3*100+(4+2*8)+4+4, 4+2*18)
)

// TestThePinIsThePreStoryPinPlusTheGroupSection is the check a pin and a
// digest computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its section
// in the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the counted block it added
// between the routes and the script section, with the version byte put back,
// must be the form the tree carried before it. It is the ONLY check in the
// package that can still fail once pinDigest, rtfDigest and pinBytesPreGroups
// / rtfBytesPreGroups agree with each other, because it is the one built from
// literals that predate this story rather than from anything this story
// wrote.
func TestThePinIsThePreStoryPinPlusTheGroupSection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what     string
		stripped []byte
		want     uint64
	}{
		{"the pinned world", pinBytesPreGroups, preGroupSectionPinDigest},
		{"the routed world", rtfBytesPreGroups, preGroupSectionRoutedDigest},
	} {
		if got := fnv1a(tc.stripped); got != tc.want {
			t.Errorf("%s with its group section removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the section it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The off-map bit (0164): the one derivation that can still fail once this
// story's new digest and its new transcription agree with each other. It is
// the NEWEST peel, so it runs BEFORE strippedOfTheSpellState and everything
// below it — the strippings come off newest story first, exactly as 0154's
// own tail displaced the treasure peel's when IT arrived.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 0165's OWN PEEL, and it is the NEWEST, so it runs before every peel below
// it — the strippings come off newest story first.
// ---------------------------------------------------------------------------

// preCastingFormVersion is the version byte the form carries once THIS STORY'S
// section is peeled back off — version 48, 0164's own landed state.
const preCastingFormVersion byte = 48

// strippedOfTheCasting removes the CASTING SECTION: the pending- cast count
// and the area-effect count, eight zero bytes on both pinned fixtures, which
// hold neither a pending cast nor an area effect — nothing but a script
// arm creates either and neither fixture runs a script.
//
// THE OFFSET IS TAKEN FROM THE END and not from the front, which is the one
// peel here that does. Every section in front of it is sized by a count and
// the arithmetic would restate ten of them; behind it stand exactly two
// sections of FIXED width on these fixtures — the script section, which is
// its volatile half and three zero counts on a world running no script, and
// the relation, whose length is a compile-time constant. So the section starts
// at a distance from the end that this story cannot get wrong by adding one,
// and a later story appending behind the script section would fail here rather
// than silently peel the wrong eight bytes.
//
// The spell record also grew by four bytes at this version, and this peel does
// NOT undo that: both fixtures name no spell table at all, so there is no
// record to shrink and the count of zero is byte-identical at either width.
func strippedOfTheCasting(form []byte) []byte {
	out := append([]byte(nil), form...)
	o := len(out) - relationLen - (scriptStateLen + scriptCountsLen) - 2*castingCountLen
	out = append(out[:o], out[o+2*castingCountLen:]...)
	out[0] = preCastingFormVersion
	return out
}

// preLastArmsFormVersion is the version byte the form carries once THIS
// STORY'S bytes are peeled back off — version 49, 0165's own landed state.
const preLastArmsFormVersion byte = 49

// strippedOfTheLastArms removes everything 0166 added to the form: the
// SCRIPT-STATE SECTION between the casting section and the script section, and
// the SEVEN bytes each record grew by — six of escort triple at the record's
// tail, and one from widening the spell mark's remaining ticks to a word.
//
// THE SECTION'S OFFSET IS TAKEN FROM THE END, on strippedOfTheCasting's own
// rule and for its reason: every section in front of it is sized by a count and
// the arithmetic would restate eleven of them, where behind it stand exactly two
// sections of fixed width on these fixtures.
//
// THE RECORD OFFSETS ARE WRITTEN OUT FROM THE CONTRACT and not read off
// entityLen, on strippedOfTheOffMap's own rule. No entity either pinned fixture
// builds holds an escort order or carries a spell mark — nothing but a script
// arm writes either and neither fixture runs a script — so every byte this peel
// removes is a zero regardless of which entity it came from, and the peel is a
// statement about the LAYOUT rather than about any value.
//
// THE MARK'S HIGH BYTE IS THE ONE REMOVED, leaving the low byte where version
// 49 put the whole field.
func strippedOfTheLastArms(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	o := len(out) - relationLen - (scriptStateLen + scriptCountsLen) - (relationSlots + tailCountLen)
	out = append(out[:o], out[o+relationSlots+tailCountLen:]...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		r := base + 230*i
		out = append(out[:r+224], out[r+230:]...)
		out = append(out[:r+221], out[r+222:]...)
	}
	out[0] = preLastArmsFormVersion
	return out
}

var (
	pinBytesPreLastArms = strippedOfTheLastArms(strippedOf1001(strippedOfCorpseLoot(strippedOfItemAttribution(pinBytesPreCarriedWeight, 20, 3), 20, 3), 20, 3), 20, 3)
	rtfBytesPreLastArms = strippedOfTheLastArms(strippedOf1001(strippedOfCorpseLoot(strippedOfItemAttribution(rtfBytesPreCarriedWeight, 16, 3), 16, 3), 16, 3), 16, 3)

	pinBytesPreCasting = strippedOfTheCasting(pinBytesPreLastArms)
	rtfBytesPreCasting = strippedOfTheCasting(rtfBytesPreLastArms)
)

// preLastArmsPinDigest and preLastArmsRoutedDigest are the two digests the
// pinned worlds carried before THIS STORY existed — version 49's own pinDigest
// (hash_test.go) and rtfDigest (routeform_test.go), pasted from the pre-story
// tree. Stripping what this story added must fall back to exactly these.
const (
	preLastArmsPinDigest    uint64 = 0x6ab15b9bda10306c
	preLastArmsRoutedDigest uint64 = 0x27295dd98025ede1
)

// TestThePinIsThePreStoryPinPlusTheLastArms is the check a pin and a digest
// computed from that pin cannot make, on TestThePinIsThePreStoryPinPlusThe
// CastingSection's own ground: the two agree by construction, and what cannot
// be faked is the pin read backwards onto a literal that predates the story.
// This literal predates it by ONE story, so reaching it proves that everything
// 0165 wrote is exactly where 0165 put it and that this story added seven bytes
// per record and one section, and nothing else.
func TestThePinIsThePreStoryPinPlusTheLastArms(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		form []byte
		want uint64
	}{
		{"pin", pinBytesPreLastArms, preLastArmsPinDigest},
		{"routed pin", rtfBytesPreLastArms, preLastArmsRoutedDigest},
	} {
		if got := fnv1a(tc.form); got != tc.want {
			t.Errorf("%s stripped of this story's own bytes hashes %#016x, want the pre-story %#016x "+
				"— this story moved a byte outside them", tc.what, got, tc.want)
		}
	}
}

// preCastingPinDigest and preCastingRoutedDigest are the two digests the pinned
// worlds carried before THIS STORY'S section existed — version 48's own
// pinDigest (hash_test.go) and rtfDigest (routeform_test.go), pasted from the
// pre-story tree. Stripping the section this story added must fall back to
// exactly these.
const (
	preCastingPinDigest    uint64 = 0x459b1d6824831221
	preCastingRoutedDigest uint64 = 0x4765e86973904fdc
)

// TestThePinIsThePreStoryPinPlusTheCastingSection is the check a pin and a
// digest computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its section in
// the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — and this literal predates it by ONE story, so reaching it proves
// that everything 0164 wrote is exactly where 0164 put it and that this story
// added eight bytes in one place and nothing else.
func TestThePinIsThePreStoryPinPlusTheCastingSection(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		form []byte
		want uint64
	}{
		// THE FORMS ARE THE PRE-0166 ONES, not the current pins: this peel
		// is one link of a chain and each link removes exactly its own
		// story's bytes from the form the story after it left behind.
		{"the pinned world", pinBytesPreLastArms, preCastingPinDigest},
		{"the routed world", rtfBytesPreLastArms, preCastingRoutedDigest},
	} {
		stripped := strippedOfTheCasting(tc.form)
		if want := len(tc.form) - 2*castingCountLen; len(stripped) != want {
			t.Errorf("%s stripped of its casting section is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its casting section removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the section it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// preOffMapFormVersion is the version byte the form carries once THIS STORY'S
// byte is peeled back off — version 46, 0156's own landed state.
const preOffMapFormVersion byte = 46

// strippedOfTheOffMap removes the ONE byte each record grew by: no entity
// either pinned fixture builds has been taken off the map — nothing but a
// script arm sets the bit and neither fixture runs a script — so this
// peel's own byte is always a zero regardless of which entity it was. The
// offset is written out from the contract, not read off entityLen, on
// strippedOfTheSpellState's own rule for its tail.
func strippedOfTheOffMap(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 223*i
		out = append(out[:o+222], out[o+223:]...)
	}
	out[0] = preOffMapFormVersion
	return out
}

var (
	pinBytesPreOffMap = strippedOfTheOffMap(pinBytesPreCasting, 20, 3)
	rtfBytesPreOffMap = strippedOfTheOffMap(rtfBytesPreCasting, 16, 3)
)

// preOffMapPinDigest and preOffMapRoutedDigest are the two digests the pinned
// worlds carried before THIS STORY'S byte existed — version 46's own
// pinDigest (hash_test.go) and rtfDigest (routeform_test.go), pasted from the
// pre-story tree. Stripping the byte this story added must fall back to
// exactly these.
const (
	preOffMapPinDigest    uint64 = 0xae4c7eafa4cd51f3
	preOffMapRoutedDigest uint64 = 0x85c5e4f52f660a84
)

// TestThePinIsThePreStoryPinPlusTheOffMap is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its byte in
// the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the one byte each record grew by,
// with the version byte put back, must be the form the tree carried before it.
func TestThePinIsThePreStoryPinPlusTheOffMap(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what     string
		stripped []byte
		want     uint64
	}{
		{"the pinned world", pinBytesPreOffMap, preOffMapPinDigest},
		{"the routed world", rtfBytesPreOffMap, preOffMapRoutedDigest},
	} {
		if got := fnv1a(tc.stripped); got != tc.want {
			t.Errorf("%s with its off-map byte removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the one per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The autocast pair and the spell effect mark (0154), one peel further back.
// It now runs on strippedOfTheOffMap's output rather than on the current form,
// which carries a byte version 45 never wrote.
// ---------------------------------------------------------------------------

// preSpellStateFormVersion is the version byte the form carries once THIS
// STORY'S tail is peeled back off — version 44, 0153's own landed state.
const preSpellStateFormVersion byte = 44

// strippedOfTheSpellState removes the FIVE bytes each record grew by: every
// entity either pinned fixture builds carries neither an autocast nor a
// mark, so this peel's own bytes are always five zeros regardless of which
// entity it was. The offset is written out from the contract, not read off
// entityLen, on strippedOfTheWeaponSpell's own rule for its tail.
func strippedOfTheSpellState(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 222*i
		out = append(out[:o+217], out[o+222:]...)
	}
	out[0] = preSpellStateFormVersion
	return out
}

var (
	pinBytesPreSpellState = strippedOfTheSpellState(pinBytesPreOffMap, 20, 3)
	rtfBytesPreSpellState = strippedOfTheSpellState(rtfBytesPreOffMap, 16, 3)
)

// preSpellStatePinDigest and preSpellStateRoutedDigest are the two digests
// the pinned worlds carried before THIS STORY'S tail existed — version 44's
// own pinDigest (hash_test.go) and rtfDigest (routeform_test.go), pasted
// from the pre-story tree. Stripping the tail this story added must fall
// back to exactly these.
const (
	preSpellStatePinDigest    uint64 = 0x7f4abfe618b08d15
	preSpellStateRoutedDigest uint64 = 0x95a70cd91a1f9808
)

// TestThePinIsThePreStoryPinPlusTheSpellState is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its five
// bytes in the wrong place, both would move together and neither would say
// so. What cannot be faked is the pin read backwards onto a literal that
// predates the story — this story's bytes, minus exactly the five bytes each
// record grew by, with the version byte put back, must be the form the tree
// carried before it.
func TestThePinIsThePreStoryPinPlusTheSpellState(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what     string
		stripped []byte
		grew     int
		want     uint64
	}{
		{"the pinned world", pinBytesPreSpellState, 3, preSpellStatePinDigest},
		{"the routed world", rtfBytesPreSpellState, 3, preSpellStateRoutedDigest},
	} {
		if got := fnv1a(tc.stripped); got != tc.want {
			t.Errorf("%s with its autocast and mark tails removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the five per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// strippedOfTheTreasure removes version 44's fixed per-entity death-gold
// section. It runs on strippedOfTheSpellState's output rather than on the
// current form, which now carries a five-byte tail version 44 never wrote.
func strippedOfTheTreasure(form []byte, off, records int) []byte {
	out := append([]byte(nil), form[:off]...)
	out = append(out, form[off+records*treasureRecordLen:]...)
	out[0] = 41
	return out
}

var (
	pinBytesPreTreasure = strippedOfTheTreasure(pinBytesPreSpellState,
		34+3*20+3*217+3*4+(4+2*18)+sackCountLen+3*carryCountLen+3*equipRecordLen, 3)
	rtfBytesPreTreasure = strippedOfTheTreasure(rtfBytesPreSpellState,
		34+3*16+3*217+(4+2*8)+4+4+(4+2*18)+sackCountLen+3*carryCountLen+3*equipRecordLen, 3)
)

// ---------------------------------------------------------------------------
// A weapon's own spell (0139): the one derivation that can still fail once
// this story's new digest and its new transcription agree with each other.
// It is the NEWEST peel now, so it runs BEFORE strippedOfTheSkill and
// everything below it — the strippings come off newest story first, exactly
// as 0135's own skill peel displaced the spellbook peel's when IT arrived.
// ---------------------------------------------------------------------------

// preWeaponSpellFormVersion is the version byte the form carries once THIS
// STORY'S weapon-spell tail is peeled back off — version 39, 0135's own
// landed state: this story's one addition sits above it, so peeling it back
// reaches exactly what 0135 left behind.
const preWeaponSpellFormVersion byte = 39

// strippedOfTheWeaponSpell removes the SIX bytes each record grew by: every
// entity either pinned fixture builds carries no weapon and so no spell —
// this task wires no weapon to a placement — so this peel's own bytes are
// always six zeros regardless of which entity it was. The offset is written
// out from the contract, not read off entityLen, on strippedOfTheSkill's own
// rule for its tail.
func strippedOfTheWeaponSpell(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 217*i
		out = append(out[:o+211], out[o+217:]...)
	}
	out[0] = preWeaponSpellFormVersion
	return out
}

// pinBytesPreWeaponSpell and rtfBytesPreWeaponSpell are the two pinned
// forms with THIS STORY'S own addition peeled back off — version 39,
// 0135's own landed state. strippedOfTheSkill (below) now runs on these
// rather than on pinBytes and rtfBytes directly, on
// TestThePinIsThePreStoryPinPlusTheSpells's own precedent further down:
// those two now carry a six-byte tail 0135 never wrote, which
// strippedOfTheSkill's own fixed offsets do not account for, so the newer
// bytes have to come off first.
var (
	pinBytesPreWeaponSpell = strippedOfTheWeaponSpell(pinBytesPreTreasure, 20, 3)
	rtfBytesPreWeaponSpell = strippedOfTheWeaponSpell(rtfBytesPreTreasure, 16, 3)
)

// preWeaponSpellPinDigest and preWeaponSpellRoutedDigest are the two
// digests the pinned worlds carried before THIS STORY'S weapon-spell tail
// existed — version 39's own pinDigest (hash_test.go) and rtfDigest
// (routeform_test.go), pasted from the pre-story tree. Stripping the
// weapon-spell tail this story added must fall back to exactly these,
// which is TestThePinIsThePreStoryPinPlusTheWeaponSpell's whole content.
const (
	preWeaponSpellPinDigest    uint64 = 0x00c2f0597f237b9e
	preWeaponSpellRoutedDigest uint64 = 0xb79e61f2927374bf
)

// TestThePinIsThePreStoryPinPlusTheWeaponSpell is the check a pin and a
// digest computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its pair in
// the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the six bytes each record grew
// by, with the version byte put back, must be the form the tree carried
// before it.
func TestThePinIsThePreStoryPinPlusTheWeaponSpell(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", pinBytesPreTreasure, 20, 3, preWeaponSpellPinDigest},
		{"the routed world", rtfBytesPreTreasure, 16, 3, preWeaponSpellRoutedDigest},
	} {
		stripped := strippedOfTheWeaponSpell(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 6*tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its weapon-spell tails is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its weapon-spell tails removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the six per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The skill block's own byte-form witness (0135): the one derivation that
// can still fail once this story's new digest and its new transcription
// agree with each other. It was the NEWEST peel until 0139's own
// weapon-spell tail arrived above it; it runs BEFORE strippedOfTheKnownSpells
// and everything below it — the strippings come off newest story first,
// exactly as 0127's own spellbook peel displaced the experience peel's when
// IT arrived.
// ---------------------------------------------------------------------------

// preSkillFormVersion is the version byte the form carries once THIS
// STORY'S skill tail is peeled back off — version 38, 0129's own landed
// state: this story's own addition sits above it, so peeling it back
// reaches exactly what 0129 left behind.
const preSkillFormVersion byte = 38

// strippedOfTheSkill removes the TWENTY-FOUR bytes each record grew by:
// every entity either pinned fixture builds names no skill level at all —
// this task gives Skill no writer outside construction and decode — so
// this peel's own bytes are always twenty-four zeros regardless of which
// entity it was. The offset is written out from the contract, not read off
// entityLen, on strippedOfTheKnownSpells's own rule for its tail.
func strippedOfTheSkill(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 211*i
		out = append(out[:o+187], out[o+211:]...)
	}
	out[0] = preSkillFormVersion
	return out
}

// pinBytesPreSkill and rtfBytesPreSkill are the two pinned forms with THIS
// STORY'S own addition peeled back off — version 38, 0129's own landed
// state. Every older peel below (strippedOfTheKnownSpells and everything
// under it) now runs on these rather than on pinBytes and rtfBytes
// directly, on TestThePinIsThePreStoryPinPlusTheSpells's own precedent
// further down: those two carry a twenty-four-byte tail this story never
// wrote, which every older peel's own fixed offsets do not account for, so
// the newer bytes have to come off first. THE INPUT IS pinBytesPreWeaponSpell
// AND rtfBytesPreWeaponSpell rather than pinBytes and rtfBytes themselves,
// on 0139's own precedent immediately above: those two now also carry the
// six-byte weapon-spell tail 0139 added above this story's own block, which
// this peel's fixed offsets do not account for either, so it has to come
// off first still.
var (
	pinBytesPreSkill = strippedOfTheSkill(pinBytesPreWeaponSpell, 20, 3)
	rtfBytesPreSkill = strippedOfTheSkill(rtfBytesPreWeaponSpell, 16, 3)
)

// preSkillPinDigest and preSkillRoutedDigest are the two digests the
// pinned worlds hash to once THIS STORY'S skill tail is peeled back off —
// version 38's own digests: pinDigest's own predecessor
// (0x920d2c848eafa34b, hash_test.go) and rtfDigest's own predecessor
// (0x25661e31ab76e502, routeform_test.go), pasted from the pre-story tree
// rather than recomputed, on preSpellPinDigest's own precedent below.
const (
	preSkillPinDigest    uint64 = 0x920d2c848eafa34b
	preSkillRoutedDigest uint64 = 0x25661e31ab76e502
)

// TestThePinIsThePreStoryPinPlusTheSkill is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its
// twenty-four bytes in the wrong place, both would move together and
// neither would say so. What cannot be faked is the pin read backwards
// onto a literal that predates the story — this story's bytes, minus
// exactly the twenty-four bytes each record grew by, with the version
// byte put back, must be the form the tree carried before it.
func TestThePinIsThePreStoryPinPlusTheSkill(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what     string
		stripped []byte
		want     uint64
	}{
		{"the pinned world", pinBytesPreSkill, preSkillPinDigest},
		{"the routed world", rtfBytesPreSkill, preSkillRoutedDigest},
	} {
		if got := fnv1a(tc.stripped); got != tc.want {
			t.Errorf("%s with its skill tail removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the block it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------- AC-2

// TestASkillLevelRoundTripsByteIdentically is AC-2's first clause: marshal,
// unmarshal, marshal again over a world whose one entity carries six
// distinct, nonzero skill levels, and the two byte forms must be
// identical — not merely equivalent worlds, the same bytes — on
// TestASpellTableAndAKnownSpellsMaskRoundTripByteIdentically's own pattern
// below.
func TestASkillLevelRoundTripsByteIdentically(t *testing.T) {
	t.Parallel()

	levels := [skillSlots]int32{1, 20, 55, 0, 100, 37}
	w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 1, Y: 1, Skill: levels}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var back World
	if err := back.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	second, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("a world carrying six nonzero skill levels does not round-trip byte-identically:\n % x\n % x",
			first, second)
	}
	if got := back.entities[0].Skill; got != levels {
		t.Errorf("the decoded entity's Skill is %v, want %v", got, levels)
	}
	if got, want := back.Hash(), w.Hash(); got != want {
		t.Errorf("the round-tripped world hashes %#016x, the original %#016x", got, want)
	}
}

// TestTwoWorldsDifferingOnlyInOneSkillLevelHashDifferently is AC-2's second
// clause: two worlds alike in everything else, differing only in one skill
// slot, must marshal to different bytes and hash differently — on
// TestTwoWorldsDifferingOnlyInTheirSpellTablesHashDifferently's own
// argument below: a level dropped between the constructor and the encoder
// would leave every behavioural test in the tree green.
func TestTwoWorldsDifferingOnlyInOneSkillLevelHashDifferently(t *testing.T) {
	t.Parallel()

	build := func(blade int32) *World {
		w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
			[]Entity{{ID: 1, X: 1, Y: 1, Skill: [skillSlots]int32{blade, 0, 0, 0, 0, 0}}})
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		return w
	}
	a, b := build(10), build(11)
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in one skill level hash alike (%#016x) — "+
			"the level does not reach the byte form", a.Hash())
	}
	fa, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary a: %v", err)
	}
	fb, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary b: %v", err)
	}
	if bytes.Equal(fa, fb) {
		t.Errorf("two worlds differing only in one skill level marshal to the same bytes")
	}
}

// ---------------------------------------------------------------------------
// The spellbook's own byte-form witness (0127): the one derivation that can
// still fail once that story's own digest and its own transcription agree
// with each other. It was the NEWEST peel until 0135's own skill block
// arrived above it, and then until 0139's own weapon-spell tail arrived
// above THAT; it runs BEFORE strippedOfTheExperience and everything below
// it — the strippings come off newest story first, exactly as 0125's own
// experience peel displaced the command group peel's when IT arrived.
// ---------------------------------------------------------------------------

// preSpellFormVersion is the version byte the form carries once THIS
// STORY'S known-spells tail and spell table are peeled back off — version
// 35, 0125's own landed state: this story's two additions both sit above
// it, so peeling both back reaches exactly what 0125 left behind.
const preSpellFormVersion byte = 35

// strippedOfTheKnownSpells removes the FOUR bytes each record grew by
// (0127 FR-4b): every entity either pinned fixture builds names no spell
// and the fixtures issue no KindReadBook command, so this peel's own bytes
// are always four zeros regardless of which entity it was. The offset is
// written out from the contract, not
// read off entityLen, on strippedOfTheExperience's own rule for its tail.
func strippedOfTheKnownSpells(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 187*i
		out = append(out[:o+183], out[o+187:]...)
	}
	out[0] = preSpellFormVersion
	return out
}

// strippedOfTheSpellTable removes exactly the section this story added —
// strippedOfTheEquipment's own shape: the section sits in the MIDDLE of
// the form rather than inside each record, so the inverse takes the
// section's own offset and length, properties of the WORLD being
// measured, passed in by the caller rather than read back off the
// encoder.
func strippedOfTheSpellTable(form []byte, off, n int) []byte {
	out := append([]byte(nil), form[:off]...)
	out = append(out, form[off+n:]...)
	out[0] = preSpellFormVersion
	return out
}

// strippedOfTheSpells is this story's own combined peel: the known-spells
// tail off every record, then the spell table section off what is left.
// This story adds two shapes rather than one — a record widening on
// strippedOfTheExperience's own precedent and a new section on
// strippedOfTheEquipment's — so its own peel is the two of theirs in
// sequence, record first so the section's own offset is measured at the
// narrower, post-peel record width.
func strippedOfTheSpells(form []byte, cells, records, off, n int) []byte {
	return strippedOfTheSpellTable(strippedOfTheKnownSpells(form, cells, records), off, n)
}

// pinBytesPreSpells and rtfBytesPreSpells are the two pinned forms with
// 0127's own two additions peeled back off — version 35, 0125's own landed
// state. Every older peel below (strippedOfTheExperience and everything
// under it) now runs on these rather than on pinBytes and rtfBytes
// directly, on TestThePinIsThePreStoryPinPlusTheEquipment's own precedent:
// those two carry a four-byte tail and a two-byte section 0127 never wrote,
// which every older peel's own fixed offsets do not account for, so the
// newer bytes have to come off first. THE INPUT IS pinBytesPreSkill AND
// rtfBytesPreSkill rather than pinBytes and rtfBytes themselves, on this
// story's own precedent immediately above: those two now carry a
// twenty-four-byte tail neither 0127 nor anything below it wrote either, so
// it has to come off first still — and pinBytesPreSkill and rtfBytesPreSkill
// are themselves now built from pinBytesPreWeaponSpell and
// rtfBytesPreWeaponSpell rather than from pinBytes and rtfBytes directly
// (0139's own section above), because 0139 moved the starting point one
// layer further out still: the six-byte weapon-spell tail has to come off
// before the skill block can be measured at ITS OWN fixed offsets.
var (
	pinBytesPreSpells = strippedOfTheSpells(pinBytesPreSkill, 20, 3,
		34+3*20+3*183+3*4+(4+2*18)+sackCountLen+3*carryCountLen+3*equipRecordLen+purseLen, spellCountLen)
	rtfBytesPreSpells = strippedOfTheSpells(rtfBytesPreSkill, 16, 3,
		34+3*16+3*183+(4+2*8)+4+4+(4+2*18)+sackCountLen+3*carryCountLen+3*equipRecordLen+purseLen, spellCountLen)
)

// preSpellPinDigest and preSpellRoutedDigest are the two digests the
// pinned worlds hash to once THIS STORY'S known-spells tail and spell
// table are peeled back off — version 35's own digests: pinDigest's own
// predecessor (0x2cf7c16babbc3952, hash_test.go) and rtfDigest's own
// predecessor (0xcba5b9ca7ed7891b, routeform_test.go), pasted from the
// pre-story tree rather than recomputed, on preCarryPinDigest's own
// precedent above.
const (
	preSpellPinDigest    uint64 = 0x2cf7c16babbc3952
	preSpellRoutedDigest uint64 = 0xcba5b9ca7ed7891b
)

// TestThePinIsThePreStoryPinPlusTheSpells is the check a pin and a digest
// computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its four
// bytes or its section in the wrong place, both would move together and
// neither would say so. What cannot be faked is the pin read backwards
// onto a literal that predates the story — this story's bytes, minus
// exactly the four bytes each record grew by and the section it added
// between the purse and the script section, with the version byte put
// back, must be the form the tree carried before it.
func TestThePinIsThePreStoryPinPlusTheSpells(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what     string
		stripped []byte
		want     uint64
	}{
		{"the pinned world", pinBytesPreSpells, preSpellPinDigest},
		{"the routed world", rtfBytesPreSpells, preSpellRoutedDigest},
	} {
		if got := fnv1a(tc.stripped); got != tc.want {
			t.Errorf("%s with its known-spells tail and spell table removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the two shapes it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------- AC-8, AC-9

// TestASpellTableAndAKnownSpellsMaskRoundTripByteIdentically is AC-8's first
// clause: marshal, unmarshal, marshal again over a world whose spell table
// carries two rows and whose one entity's KnownSpells mask is nonzero, and
// the two byte forms must be identical — not merely equivalent worlds, the
// same bytes — on TestCarriedCodesAndPursesRoundTripByteIdentically's own
// pattern.
func TestASpellTableAndAKnownSpellsMaskRoundTripByteIdentically(t *testing.T) {
	t.Parallel()

	spells := []SpellRule{
		{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true},
		{ID: 6, ManaCost: 12, School: 2, MaxRange: 3, DamageMin: 0, DamageMax: 0, TargetsUnit: false, Damaging: false},
	}
	const knownMask = uint32(1)<<1 | uint32(1)<<6
	w, err := NewSpelledWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 1, Y: 1, KnownSpells: knownMask}}, nil, spells)
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	var back World
	if err := back.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	second, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("a world carrying a spell table and a nonzero KnownSpells mask does not round-trip byte-identically:\n % x\n % x",
			first, second)
	}
	if got := back.entities[0].KnownSpells; got != knownMask {
		t.Errorf("the decoded entity's KnownSpells is %#x, want %#x", got, knownMask)
	}
	if got, want := back.spells, spells; !reflect.DeepEqual(got, want) {
		t.Errorf("the decoded spell table is %+v, want %+v", got, want)
	}
	if got, want := back.Hash(), w.Hash(); got != want {
		t.Errorf("the round-tripped world hashes %#016x, the original %#016x", got, want)
	}
}

// TestTwoWorldsDifferingOnlyInTheirSpellTablesHashDifferently is AC-8's
// second clause: two worlds alike in everything else, differing only in one
// row of the spell table, must marshal to different bytes and hash
// differently — on TestTwoWorldsDifferingOnlyInADecayFieldHashDifferently's
// own argument (decayform_test.go): a table dropped between the constructor
// and the encoder would leave every behavioural test in the tree green.
func TestTwoWorldsDifferingOnlyInTheirSpellTablesHashDifferently(t *testing.T) {
	t.Parallel()

	build := func(cost int32) *World {
		w, err := NewSpelledWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
			[]Entity{{ID: 1, X: 1, Y: 1}}, nil,
			[]SpellRule{{ID: 1, ManaCost: cost, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}})
		if err != nil {
			t.Fatalf("NewSpelledWorld: %v", err)
		}
		return w
	}
	a, b := build(3), build(5)
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in their spell tables hash alike (%#016x) — "+
			"the table does not reach the byte form", a.Hash())
	}
	fa, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary a: %v", err)
	}
	fb, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary b: %v", err)
	}
	if bytes.Equal(fa, fb) {
		t.Errorf("two worlds differing only in their spell tables marshal to the same bytes")
	}
}

// spellFormFixture is a 0-entity world over one cell, carrying spells: a
// deterministic byte form whose spell table sits at a fixed, hand-computed
// offset — 34-byte header, one plane byte times three, the group section's
// own bare 4-byte zero count, the sack section's own bare 4-byte zero count,
// no carry or equipment records (zero entities), and purseLen zeroed bytes —
// 245, on this file's own convention of writing an offset out from the
// contract rather than reaching for the encoder.
func spellFormFixture(t *testing.T, spells []SpellRule) []byte {
	t.Helper()
	w, err := NewSpelledWorld(1, Bounds{Width: 1, Height: 1}, ModeCanonical, []byte{0}, nil, nil, spells)
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

const spellTableAt = 34 + 3 + groupCountLen + sackCountLen + purseLen

// TestUnmarshalRefusesTheSpellTableShapesNoTableMay carries the reader's own
// refusals: a count the payload cannot supply, a repeated id, id 0, and a
// flags byte setting a bit past the five defined flags. Each case differs
// from a valid two-row form in exactly one field, on
// TestUnmarshalRefusesTheDecayShapesNoTickCanLeave's own pattern
// (decayform_test.go).
func TestUnmarshalRefusesTheSpellTableShapesNoTableMay(t *testing.T) {
	t.Parallel()

	valid := spellFormFixture(t, []SpellRule{
		{ID: 1, ManaCost: 3, School: 1, MaxRange: 7, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true},
		{ID: 2, ManaCost: 5, School: 2, MaxRange: 3, DamageMin: 1, DamageMax: 2, TargetsUnit: false, Damaging: false},
	})
	const (
		record0IDAt    = spellTableAt + spellCountLen
		record0FlagsAt = record0IDAt + 16
		record1IDAt    = record0IDAt + spellRecordLen
	)
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"a spell count the payload cannot supply", withU16(valid, spellTableAt, 0xffff)},
		{"a repeated spell id", withU16(valid, record1IDAt, 1)},
		{"a spell id of 0", withU16(valid, record0IDAt, 0)},
		{"a flags byte setting a bit past the five defined ones", withByte(valid, record0FlagsAt, 0x20)},
		{"a flags byte that is both damaging and restorative", withByte(valid, record0FlagsAt, spellFlagDamaging|spellFlagRestorative)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var w World
			if err := w.UnmarshalBinary(tc.data); err == nil {
				t.Error("accepted")
			}
		})
	}
	// And the unspoiled form is accepted, so the cases above measure the
	// spell table's own checks and not something else about this fixture.
	if err := (&World{}).UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

// ---------------------------------------------------------------------------
// The experience block's own byte-form witness (0125): the one derivation
// that can still fail once that story's own digest and its own
// transcription agree with each other. It was the newest peel until this
// story's own spellbook and spell table arrived above it; it runs BEFORE
// strippedOfTheCommandGroup and everything below it — the strippings come
// off newest story first, exactly as 0117's own command group peel
// displaced the carry-and-purse peel's when IT arrived.
// ---------------------------------------------------------------------------

// preExperienceFormVersion is the version byte the form carries once
// THIS STORY'S experience-from-use tail is peeled back off — version 34,
// 0124's own landed state, not 32: this branch merges onto 0124's
// equipment section, which sits between the carry section and the purse,
// untouched by this peel, so the bytes it leaves behind still carry it.
// Tagging them 32 would claim a form that predates it.
const preExperienceFormVersion byte = 34

// strippedOfTheExperience removes the THIRTY-FOUR bytes each record grew by:
// every entity either pinned fixture builds names no experience, Mind,
// experience value, credited slot or gains flag at all — this task gives
// none of the five a writer outside the constructor's own zero default —
// so this peel's own bytes are always thirty-four zeros regardless of which
// entity it was. The offset is written out from the contract, not read off
// entityLen, on strippedOfTheCommandGroup's own rule for its tail.
func strippedOfTheExperience(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 183*i
		out = append(out[:o+149], out[o+183:]...)
	}
	out[0] = preExperienceFormVersion
	return out
}

// preExperiencePinDigest and preExperienceRoutedDigest are the two digests
// the pinned worlds hash to once THIS STORY'S experience-from-use tail is
// peeled back off — version 34's own digests, 0124's landed state with its
// equipment section still in place, NOT the version-32 pair 0117 left
// behind (preEquipPinDigest and preEquipRoutedDigest below carry those):
// this branch merges onto 0124 rather than the master 0125 was written
// against, so the form directly beneath this story's own tail is 0124's,
// not 0117's. Stripping the tail this story added must fall back to
// exactly these, which is TestThePinIsThePreStoryPinPlusTheExperience's
// whole content — both obtained by running that test, not composed by
// hand.
const (
	preExperiencePinDigest    uint64 = 0xe0e6cb0285f41a27
	preExperienceRoutedDigest uint64 = 0xa4bc8982cd0f7d6e
)

// TestThePinIsThePreStoryPinPlusTheExperience is the check a pin and a
// digest computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its block
// in the wrong place, both would move together and neither would say so.
// What cannot be faked is the pin read backwards onto a literal that
// predates the story — this story's bytes, minus exactly the thirty-four
// bytes each record grew by, with the version byte put back, must be the
// form the tree carried before it.
//
// It matters more here than it did for the command group, because NOTHING
// IN THIS TASK READS AN EXPERIENCE EITHER: every behavioural test in the
// tree would still pass with the five fields dropped between the
// constructor and the encoder, and every digest that moved would have
// moved anyway.
func TestThePinIsThePreStoryPinPlusTheExperience(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world", pinBytesPreSpells, 20, 3, preExperiencePinDigest},
		{"the routed world", rtfBytesPreSpells, 16, 3, preExperienceRoutedDigest},
	} {
		stripped := strippedOfTheExperience(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 34*tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its experience blocks is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its experience blocks removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the thirty-four per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The command group's own byte-form witness (0117): the one derivation that
// can still fail once this story's new digest and its new transcription
// agree with each other. It was the newest peel until 0124's own equipment
// section arrived above it, and then 0125's own experience tail arrived
// above THAT — the strippings come off newest story first, exactly as
// 0112's own carry and purse peel displaced the regeneration block's when
// IT arrived — so it now runs on strippedOfTheEquipment's own output, which
// itself now runs on strippedOfTheExperience's, rather than directly on
// pinBytes and rtfBytes, and it still runs BEFORE strippedOfTheCarryAndPurse
// and everything below it.
// ---------------------------------------------------------------------------

// preCommandGroupFormVersion is the version byte the form carried before
// THIS STORY'S command group tail existed — version 31, 0122's own landed
// state.
const preCommandGroupFormVersion byte = 31

// strippedOfTheCommandGroup removes the FOUR bytes each record grew by:
// every entity either pinned fixture builds names no command group at all
// — this task gives the field no writer — so this peel's own bytes are
// always four zeros regardless of which entity it was. The offset is written
// out from the contract, not read off entityLen, on strippedOfTheRegen's own
// rule for its tail.
func strippedOfTheCommandGroup(form []byte, cells, records int) []byte {
	out := append([]byte(nil), form...)
	base := 34 + 3*cells
	for i := records - 1; i >= 0; i-- {
		o := base + 149*i
		out = append(out[:o+145], out[o+149:]...)
	}
	out[0] = preCommandGroupFormVersion
	return out
}

// preCommandGroupPinDigest and preCommandGroupRoutedDigest are the two
// digests the pinned worlds carried before THIS STORY'S command group tail
// existed — version 31's pinDigest and rtfDigest, the state 0122 landed.
// Stripping the tail this story added must fall back to exactly these,
// which is TestThePinIsThePreStoryPinPlusTheCommandGroup's whole content.
const (
	preCommandGroupPinDigest    uint64 = 0xede6d696093ba13e
	preCommandGroupRoutedDigest uint64 = 0xd1b9e47807319d9f
)

// TestThePinIsThePreStoryPinPlusTheCommandGroup is the check a pin and a
// digest computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its word in
// the wrong place, both would move together and neither would say so. What
// cannot be faked is the pin read backwards onto a literal that predates the
// story — this story's bytes, minus exactly the four bytes each record grew
// by, with the version byte put back, must be the form the tree carried
// before it.
//
// It matters more here than it did for the facing, because NOTHING IN THIS
// TASK READS A COMMAND GROUP EITHER: every behavioural test in the tree
// would still pass with the field dropped between the constructor and the
// encoder, and every digest that moved would have moved anyway.
func TestThePinIsThePreStoryPinPlusTheCommandGroup(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what    string
		form    []byte
		cells   int
		records int
		want    uint64
	}{
		{"the pinned world",
			strippedOfTheEquipment(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 34+3*20+3*149+3*4+(4+2*18)+sackCountLen+3*carryCountLen, 3*equipRecordLen, priorFormVersion),
			20, 3, preCommandGroupPinDigest},
		{"the routed world",
			strippedOfTheEquipment(strippedOfTheExperience(rtfBytesPreSpells, 16, 3), 34+3*16+3*149+(4+2*8)+4+4+(4+2*18)+sackCountLen+3*carryCountLen, 3*equipRecordLen, priorFormVersion),
			16, 3, preCommandGroupRoutedDigest},
	} {
		stripped := strippedOfTheCommandGroup(tc.form, tc.cells, tc.records)
		if want := len(tc.form) - 4*tc.records; len(stripped) != want {
			t.Errorf("%s stripped of its command group words is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its command group words removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the four per record it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// The carry and purse sections' own byte-form witness (0112): the one
// derivation that can still fail once this story's new digests and its new
// transcriptions agree with each other.
// ---------------------------------------------------------------------------

// preCarryFormVersion is the version byte the form carried before THIS
// STORY'S carry and purse sections existed — version 25, 0109's own landed
// state.
//
// strippedOfTheCarryAndPurse is the inverse of the pair this story added:
// one form with the two blocks it wrote between the sack section and the
// script section cut out and the version byte put back. It is no longer the
// newest peel — 0117's own command group tail is — but it keeps its place
// ahead of strippedOfTheRegen and everything below it: the strippings come
// off newest story first, exactly as the regeneration block's own peel was
// displaced when IT arrived.
const preCarryFormVersion byte = 25

// strippedOfTheCarryAndPurse removes exactly the bytes this story added.
// Unlike strippedOfTheRegen's per-record tail, the pair sits in the MIDDLE
// of the form rather than inside each record — strippedOfTheGroups' and
// strippedOfTheSacks' own shape rather than strippedOfThePost's — so the
// inverse takes the section's own offset and length, properties of the
// WORLD being measured, passed in by the caller rather than read back off
// the encoder.
func strippedOfTheCarryAndPurse(form []byte, off, n int) []byte {
	out := append([]byte(nil), form[:off]...)
	out = append(out, form[off+n:]...)
	out[0] = preCarryFormVersion
	return out
}

// preCarryPinDigest and preCarryRoutedDigest are the two digests the pinned
// worlds carried before THIS STORY'S carry and purse sections existed —
// version 25's pinDigest and rtfDigest, pasted from the pre-story tree.
// Stripping the two sections this story added must fall back to exactly
// these, which is TestThePinIsThePreStoryPinPlusTheCarryAndPurse's whole
// content.
const (
	preCarryPinDigest    uint64 = 0x6ddd5aa12841e3ec
	preCarryRoutedDigest uint64 = 0x27dac47f6005d3a1
)

// TestThePinIsThePreStoryPinPlusTheCarryAndPurse is the check a pin and a
// digest computed from that pin cannot make.
//
// Those two agree by construction: had the transcription gained its
// sections in the wrong place, both would move together and neither would
// say so. What cannot be faked is the pin read backwards onto a literal
// that predates the story — this story's bytes, minus exactly the two
// blocks it added between the sack section and the script section, with
// the version byte put back, must be the form the tree carried before it.
func TestThePinIsThePreStoryPinPlusTheCarryAndPurse(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		form []byte
		off  int
		want uint64
	}{
		{"the pinned world", strippedOfTheCommandGroup(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 20, 3), 34 + 3*20 + 3*145 + 3*4 + (4 + 2*18) + sackCountLen, preCarryPinDigest},
		{"the routed world", strippedOfTheCommandGroup(strippedOfTheExperience(rtfBytesPreSpells, 16, 3), 16, 3), 34 + 3*16 + 3*145 + (4 + 2*8) + 4 + 4 + (4 + 2*18) + sackCountLen, preCarryRoutedDigest},
	} {
		stripped := strippedOfTheCarryAndPurse(tc.form, tc.off, 3*carryCountLen+3*equipRecordLen+purseLen)
		if want := len(tc.form) - (3*carryCountLen + 3*equipRecordLen + purseLen); len(stripped) != want {
			t.Errorf("%s stripped of its carry and purse sections is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its carry and purse sections removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the two sections it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

// preScriptPlayerFormVersion is the version byte the form carried before
// THIS STORY'S check-record player references existed — version 26, 0112's
// own landed state, and the version formatVersion moved off when it jumped
// to 31 (0122 D-1: three later numbers are already out to other lanes and
// this one is above all of them, so 27 through 30 are not this story's
// "previous version" and are never built here).
const preScriptPlayerFormVersion byte = 26

// ---------------------------------------------------------------------------
// The equipment section's own byte-form witness (0124 T2): the one
// derivation that can still fail once this story's new digest and its new
// transcription agree with each other, on the carry section's own kind of
// check above rather than a new one.
// ---------------------------------------------------------------------------

// priorFormVersion is the version byte this tree's OWN form carried
// immediately before this story — 32, 0117's own landed state, which this
// tree actually wrote and this pin actually carried, with 0122's and
// 0117's own sections already in it. 0112's own preCarryFormVersion served
// both "the version this tree had" and "the version one below the new
// one" because 0112 bumped 25 straight to 26; this story instead bumps 32
// to 34, skipping one number (33) allocated to another lane running in
// parallel off the same master this one branched from — but 32 answers
// both questions at once regardless, because it is the number that lane's
// own skip left immediately behind 34, and it is the only one of the two
// this tree ever actually held.
const priorFormVersion byte = 32

// strippedOfTheEquipment removes exactly the bytes this story added and
// puts version in the version byte — strippedOfTheCarryAndPurse's own
// shape, widened by one argument because this story's two callers pass two
// different intents through the same parameter: a real reconstruction of
// history here, and a refusal fixture in TestThePreviousVersionFormIsRefused
// below, where 0112's one call needed only one. The section sits in the
// MIDDLE of the form rather than inside each record — strippedOfTheGroups'
// and strippedOfTheSacks' own shape rather than strippedOfThePost's — so
// the inverse takes the section's own offset and length, properties of the
// WORLD being measured, passed in by the caller rather than read back off
// the encoder.
func strippedOfTheEquipment(form []byte, off, n int, version byte) []byte {
	out := append([]byte(nil), form[:off]...)
	out = append(out, form[off+n:]...)
	out[0] = version
	return out
}

// preEquipPinDigest and preEquipRoutedDigest are the two digests the pinned
// worlds carried before THIS STORY'S equipment section existed — the state
// this tree actually held at priorFormVersion, with 0122's check-record
// player references and 0117's command group tail already in it. Stripping
// the section this story added and restoring that byte must fall back to
// exactly these, which is TestThePinIsThePreStoryPinPlusTheEquipment's
// whole content.
const (
	preEquipPinDigest    uint64 = 0xb9105ae160cf28d9
	preEquipRoutedDigest uint64 = 0x7cf2b3820d012994
)

// TestThePinIsThePreStoryPinPlusTheEquipment is the check a pin and a
// digest computed from that pin cannot make, on
// TestThePinIsThePreStoryPinPlusTheCarryAndPurse's own argument: what
// cannot be faked is the pin read backwards onto a literal that predates
// the story — this story's bytes, minus exactly the block it added between
// the carry section and the purse section, with the version byte put back
// to what this tree actually carried, must be the form the tree carried
// before it.
//
// IT RUNS ON strippedOfTheExperience's OWN OUTPUT NOW, not on pinBytes and
// rtfBytes directly (0125): those two carry a thirty-four-byte tail this
// story never wrote, one record widening this peel's own fixed offsets do
// not account for, so the newer tail has to come off first — the
// strippings still come off newest story first — before this section's own
// offset means what it always meant.
func TestThePinIsThePreStoryPinPlusTheEquipment(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		form []byte
		off  int
		want uint64
	}{
		{"the pinned world", strippedOfTheExperience(pinBytesPreSpells, 20, 3),
			34 + 3*20 + 3*149 + 3*4 + (4 + 2*18) + sackCountLen + 3*carryCountLen, preEquipPinDigest},
		{"the routed world", strippedOfTheExperience(rtfBytesPreSpells, 16, 3),
			34 + 3*16 + 3*149 + (4 + 2*8) + 4 + 4 + (4 + 2*18) + sackCountLen + 3*carryCountLen, preEquipRoutedDigest},
	} {
		stripped := strippedOfTheEquipment(tc.form, tc.off, 3*equipRecordLen, priorFormVersion)
		if want := len(tc.form) - 3*equipRecordLen; len(stripped) != want {
			t.Errorf("%s stripped of its equipment section is %d byte(s), want %d",
				tc.what, len(stripped), want)
		}
		if got := fnv1a(stripped); got != tc.want {
			t.Errorf("%s with its equipment section removed hashes %#016x, want the pre-story %#016x — "+
				"this story wrote a byte outside the section it was supposed to add",
				tc.what, got, tc.want)
		}
	}
}

func TestThePreviousVersionFormIsRefused(t *testing.T) {
	t.Parallel()

	valid, err := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}}).MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	for _, tc := range []struct {
		what string
		form []byte
		old  byte
	}{
		{"a current form spoiled at its version byte", withByte(valid, 0, preCarryFormVersion), preCarryFormVersion},
		{"the well-formed previous-version stream this tree emitted a story ago",
			strippedOfTheCarryAndPurse(strippedOfTheCommandGroup(strippedOfTheExperience(pinBytesPreSpells, 20, 3), 20, 3), 34+3*20+3*145+3*4+(4+2*18)+sackCountLen, 3*carryCountLen+3*equipRecordLen+purseLen),
			preCarryFormVersion},
		{"a current form spoiled to the version this story's player references replaced",
			withByte(valid, 0, preScriptPlayerFormVersion), preScriptPlayerFormVersion},
		{"a current form spoiled to the version this story's command group replaced",
			withByte(valid, 0, preCommandGroupFormVersion), preCommandGroupFormVersion},
		{"a current form spoiled to the version this story's equipment replaced",
			withByte(valid, 0, priorFormVersion), priorFormVersion},
		{"the well-formed previous-version stream this tree would emit at 32",
			strippedOfTheEquipment(strippedOfTheExperience(pinBytesPreSpells, 20, 3),
				34+3*20+3*149+3*4+(4+2*18)+sackCountLen+3*carryCountLen, 3*equipRecordLen, priorFormVersion),
			priorFormVersion},
		{"a current form spoiled to the version this story's experience block replaced",
			withByte(valid, 0, preExperienceFormVersion), preExperienceFormVersion},
		{"a current form spoiled to the version this story's spell table replaced",
			withByte(valid, 0, preSpellFormVersion), preSpellFormVersion},
		{"the well-formed previous-version stream this tree would emit at 35",
			pinBytesPreSpells, preSpellFormVersion},
	} {
		var w World
		err := w.UnmarshalBinary(tc.form)
		if err == nil {
			t.Errorf("%s was accepted", tc.what)
			continue
		}
		if !strings.Contains(err.Error(), strconv.Itoa(int(tc.old))) ||
			!strings.Contains(err.Error(), strconv.Itoa(int(formatVersion))) {
			t.Errorf("%s: refusal %q names neither the version refused nor the one this build reads",
				tc.what, err.Error())
		}
	}
}

// ---------------------------------------------------------------------------
// The container and the purse crossing the byte form (0112): what T2 adds
// past the pin and the peel witnesses above — AC-2, AC-3, AC-4.
// ---------------------------------------------------------------------------

// equalCodes is sackform_test.go's own helper, reused here rather than
// redeclared: nothing about it is sack-specific, and this package builds
// as one test binary.

// TestCarriedCodesAndPursesRoundTripByteIdentically is AC-3: marshal,
// unmarshal, marshal again over a world whose entities carry items and
// whose purses hold gold, and the two byte forms must be identical — not
// merely equivalent worlds, the same bytes.
//
// carried and purses are set through the field directly, on
// TestEveryFieldChangesTheDigest's own rule (hash_test.go): no exported
// setter exists for either — Carried and Purse are read-only until T3's
// transfer — so a test inside this package reaches them the only way
// available, exactly as that file's cases reach every other field. Since
// 0138 the field is a list of ELEMENTS, so the direct write states each
// code's count; every count here is 1, which is the state this case always
// measured, and the counted case is TestACountedElementCrossesTheFormAsItsUnits.
func TestCarriedCodesAndPursesRoundTripByteIdentically(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}})
	w.carried[0] = []ItemStack{{Code: 0x0101, Count: 1}, {Code: 0x0202, Count: 1}}
	w.carried[1] = []ItemStack{{Code: 0x0303, Count: 1}}
	w.purses[0] = 500
	w.purses[relationSlots-1] = 999999

	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	second, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("marshal -> unmarshal -> marshal is not byte-identical:\n % x\n % x", first, second)
	}

	// And what came back really does hold what was put in — the round trip
	// is faithful and not merely stable.
	if got, ok := back.Carried(1); !ok || !equalCodes(got, []uint16{0x0101, 0x0202}) {
		t.Errorf("entity 1 decoded carrying %v, want [0x0101 0x0202]", got)
	}
	if got, ok := back.Carried(2); !ok || !equalCodes(got, []uint16{0x0303}) {
		t.Errorf("entity 2 decoded carrying %v, want [0x0303]", got)
	}
	if got := back.Purse(0); got != 500 {
		t.Errorf("purse 0 decoded as %d, want 500", got)
	}
	if got := back.Purse(relationSlots - 1); got != 999999 {
		t.Errorf("purse %d decoded as %d, want 999999", relationSlots-1, got)
	}
}

// TestTwoWorldsDifferingOnlyInOneEntitysCarriedCodesHashDifferently and
// TestTwoWorldsDifferingOnlyInOnePurseHashDifferently are AC-4's two edges:
// the container and the purse are each canonical state on their own, not
// merely alongside each other, so a change to either alone has to move the
// digest.
func TestTwoWorldsDifferingOnlyInOneEntitysCarriedCodesHashDifferently(t *testing.T) {
	t.Parallel()

	base := func(t *testing.T) *World {
		t.Helper()
		return mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}})
	}

	a := base(t)
	b := base(t)
	b.carried[1] = []ItemStack{{Code: 7, Count: 1}}

	if a.Hash() == b.Hash() {
		t.Errorf("worlds differing in one entity's carried codes both hash %#016x", a.Hash())
	}
	formA, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	formB, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if bytes.Equal(formA, formB) {
		t.Errorf("worlds differing in one entity's carried codes marshal alike")
	}
}

func TestTwoWorldsDifferingOnlyInOnePurseHashDifferently(t *testing.T) {
	t.Parallel()

	base := func(t *testing.T) *World {
		t.Helper()
		return mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}})
	}

	a := base(t)
	b := base(t)
	b.purses[3] = 42

	if a.Hash() == b.Hash() {
		t.Errorf("worlds differing in one purse both hash %#016x", a.Hash())
	}
	formA, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	formB, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if bytes.Equal(formA, formB) {
		t.Errorf("worlds differing in one purse marshal alike")
	}
}

func TestUnmarshalRefusesACarriedCodeOfZeroNamingTheEntity(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}})
	w.carried[0] = []ItemStack{{Code: 9, Count: 1}}
	valid, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	codeAt := headerLen + 3*16 + entityLen + routeCountLen + groupCountLen + sackCountLen + carryCountLen
	if got := []byte{valid[codeAt], valid[codeAt+1]}; got[0] != 9 || got[1] != 0 {
		t.Fatalf("codeAt is miscomputed: bytes %d:%d are % x, want the carried code 9 little-endian", codeAt, codeAt+2, got)
	}
	spoiled := withU16(valid, codeAt, 0)

	var back World
	err = back.UnmarshalBinary(spoiled)
	if err == nil {
		t.Fatal("a carried code of zero was accepted")
	}
	if !strings.Contains(err.Error(), "entity record 0") {
		t.Errorf("refusal %q does not name the entity", err.Error())
	}

	// And the unspoiled form is accepted, so the case above measures the
	// zero-code refusal and not something else about this fixture.
	if err := back.UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

// TestUnmarshalRefusesACarriedCountThatOverrunsTheBuffer is the carry
// section's own truncation refusal — decodeRoutes' per-record bounds
// check, restated for a different field: a declared count asking for more
// codes than the buffer holds is refused before a single code past what
// exists is read, rather than panicking or reading into the next entity's
// own record.
func TestUnmarshalRefusesACarriedCountThatOverrunsTheBuffer(t *testing.T) {
	t.Parallel()

	valid := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}})
	form, err := valid.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	countAt := headerLen + 3*16 + entityLen + routeCountLen + groupCountLen + sackCountLen
	if got := form[countAt : countAt+4]; got[0] != 0 || got[1] != 0 || got[2] != 0 || got[3] != 0 {
		t.Fatalf("countAt is miscomputed: bytes %d:%d are % x, want a zero count", countAt, countAt+4, got)
	}
	spoiled := withU32(form, countAt, 0xffffffff)

	var back World
	if err := back.UnmarshalBinary(spoiled); err == nil {
		t.Fatal("a carried code count overrunning the buffer was accepted")
	}
}

func TestTheAuthoredMapIDRoundTripsByteIdentically(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 7, Bounds{Width: 16, Height: 16}, []Entity{
		{ID: 1, X: 1, Y: 1, MapUnitID: 0},
		{ID: 2, X: 2, Y: 2, MapUnitID: 51},
		{ID: 3, X: 3, Y: 3, MapUnitID: 0xffff},
	})
	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	for i, want := range []uint16{0, 51, 0xffff} {
		if got := back.Entities()[i].MapUnitID; got != want {
			t.Errorf("entity %d comes back with map unit id %d, want %d", i, got, want)
		}
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, the original %#016x", back.Hash(), w.Hash())
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(again, first) {
		t.Error("re-marshalling a world carrying authored map ids gave different bytes")
	}
	// Two worlds differing only in one id must not collide, or the encoder
	// could be writing a constant and the read-back above would still hold.
	other := mustWorld(t, 7, Bounds{Width: 16, Height: 16}, []Entity{
		{ID: 1, X: 1, Y: 1, MapUnitID: 0},
		{ID: 2, X: 2, Y: 2, MapUnitID: 52},
		{ID: 3, X: 3, Y: 3, MapUnitID: 0xffff},
	})
	if other.Hash() == w.Hash() {
		t.Errorf("worlds differing only in one entity's authored map id both hash %#016x", w.Hash())
	}
}

// Record widths at the older versions the pinned fixture chain above widens
// through.
const (
	entityLenV61 = 290
	entityLenV62 = 291
	entityLenV63 = 293
)
