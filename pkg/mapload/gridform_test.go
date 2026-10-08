package mapload_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The fixture's extent. Twenty-four on both axes is 576 cells, of which the
// interior is the 8x8 block at columns and rows 8..15 — small enough to write
// out, large enough to hold every arm on a cell of its own.
const gfW, gfH = 24, 24

// gfPlacement is one cell of the fixture and what is put on it.
type gfPlacement struct {
	x, y    int
	word    uint16
	overlay uint8
}

// gfPlacements draws the map. Every other cell of the tile plane carries the
// open word, so a derivation that only ever looked at a nonzero word would find
// nothing here to look at.
//
// The interior picture, columns 8..15 left to right and rows 8..15 top to
// bottom — `.` open, `#` bit 13, `~` inside the water range, `^` group 7 at a
// blocking blend level, `o` a nonzero overlay code:
//
//	. # . ~ . ^ . o     row 8
//	. . . . . . . .     row 9
//	. . . . . . . .     row 10
//	~ ~ ~ ~ ~ . . .     row 11
//	. . . . . . . .     row 12
//	. . ^ ^ . . . .     row 13
//	. . . . . . . .     row 14
//	. . . . . . o .     row 15
//
// The last two placements are INSIDE the ring, where the border already blocks:
// they say that the five arms are a union in which the border's value covers the
// others, and they change no expected byte.
var gfPlacements = []gfPlacement{
	{x: 9, y: 8, word: 0x2041},
	{x: 11, y: 8, word: 0x0208},
	{x: 13, y: 8, word: 0x01D1},
	{x: 15, y: 8, word: 0x0041, overlay: 0x2a},

	{x: 8, y: 11, word: 0x0208},
	{x: 9, y: 11, word: 0x0208},
	{x: 10, y: 11, word: 0x0208},
	{x: 11, y: 11, word: 0x0208},
	{x: 12, y: 11, word: 0x0208},

	{x: 10, y: 13, word: 0x01D1},
	{x: 11, y: 13, word: 0x01D1},

	{x: 14, y: 15, word: 0x0041, overlay: 0xfa},

	{x: 3, y: 3, word: 0x01D1},
	{x: 20, y: 20, word: 0x0041, overlay: 0x07},
}

// The one placed unit, at the centre of an open interior cell, and the cell it
// stands on written out beside it rather than shifted here.
const (
	gfUnitX, gfUnitY = 0x0980, 0x0980
	gfCellX, gfCellY = 9, 9
	gfClass          = 5
)

// gfMap is that map, built fresh on every call.
func gfMap() *alm.Map {
	m := &alm.Map{
		Width: gfW, Height: gfH,
		Tiles:   make([]uint16, gfW*gfH),
		Overlay: make([]uint8, gfW*gfH),
		Units:   []alm.Unit{{X: gfUnitX, Y: gfUnitY, ClassID: gfClass}},
	}
	for i := range m.Tiles {
		m.Tiles[i] = 0x0041
	}
	for _, p := range gfPlacements {
		m.Tiles[p.y*gfW+p.x] = p.word
		m.Overlay[p.y*gfW+p.x] = p.overlay
	}
	return m
}

// gfInterior is the expected interior, cell by cell: rows 8..15 top to bottom,
// columns 8..15 left to right. It is a LITERAL TABLE read off the picture above
// and written from the contract's rules, never from a call into the derivation.
var gfInterior = [8][8]byte{
	{0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	{0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	{0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
}

func gfWantGrid() []byte {
	g := make([]byte, gfW*gfH)
	for y := 0; y < gfH; y++ {
		for x := 0; x < gfW; x++ {
			switch {
			case x < 8 || y < 8 || x >= gfW-8 || y >= gfH-8:
				g[y*gfW+x] = 0x03
			default:
				g[y*gfW+x] = gfInterior[y-8][x-8]
			}
		}
	}
	return g
}

// gfScriptSection is how long pkg/sim's script section is on a world running no
// script: the hundred registers, the thousand latches, the two counters, the
// outcome and three zero array counts. It is written out here rather than
// imported, because this file measures the form against something outside the
// package that writes it.
const gfScriptSection = 4*100 + 1000 + 4 + 4 + 1 + 12

// gfDigest is the digest of the world gfMap builds at tick 0.
//
// IT WAS COMPUTED OUTSIDE THIS TREE. Nothing about it came from running this
// package.
//
// It was re-derived that way when the movement domain widened the record by a
// byte and raised the version to 6, again at 0056, where the movement rate
// widened it by eight more and raised the version to 7, and AGAIN AT 0059, where
// the group rate term widened it by one and raised the version to 8 — derived
// afresh here, from the same description with that one byte added. No value
// carries over by arithmetic: FNV-1a is a left fold and byte 0 itself moved each
// time, so no two of the forms share a prefix. The values before it were
// 0x578ed5f38a8547b1, 0x435d47015db7490a and 0x85f71c84ed1bd55f. A world this
// loader builds names no domain, so the byte the record gained at 6 is the
// GROUND zero, the eight it gained at 7 are the constructor's own default speed,
// which every placement this loader resolves nothing for takes, and the empty
// crossing every unit starts in, and the one it gained at 8 is a ZERO group
// term: this loader issues no order, so no world it builds carries one — the
// loader's whole statement about all three, pinned here.
//
// It is a digest of our own encoding of our own state, this loader's own seed
// included, and nothing in it is game-derived.//
// 0063 MOVED IT AGAIN, and this time the re-derivation could be CHECKED rather
// than only repeated. Version 9 appends the mission script's section and raises
// the version byte; nothing else about this world's form moves, because this
// loader compiles no script — so the section is 1421 zeros. The new value was
// derived by taking the form apart the other way round: strip those 1421 bytes,
// confirm every one of them is zero, put byte 0 back to 8, and hash what is
// left. That hashes to the value below's predecessor, EXACTLY, which is what
// establishes that the version-9 form is the externally-derived version-8 form
// plus a version byte and a zero run — and only then is the whole form hashed
// for the number pinned here.
// 0064 MOVED IT ONCE MORE, and the same check ran a second time and a level
// deeper. Version 10 widens every entity record by the attack block's
// thirty-nine bytes and raises the version byte; this loader gives a placement no
// combat number and issues no order, so all thirty-nine are zero. Taking the form
// apart the same way — lift the thirty-nine out of the one record, confirm every
// one of them is zero, put byte 0 back to 9, and hash what is left — reproduces
// 0x…cf5a755 EXACTLY; and stripping the 1421 zeros from THAT and putting byte 0
// back to 8 reproduces 0x…d8aab382, likewise exactly. So the number below is the
// externally-derived version-8 form plus two version bumps and two runs of
// zeros, and the chain is now checked against TWO numbers neither story's own
// code produced.
//
// The values it replaces: version 9's 0xa1404d649cf5a755, version 8's
// 0xe9eb3c69d8aab382.
//
// 0067 MOVES IT AGAIN — the first of these moves that is DELIBERATE, and the
// first that does not touch the version byte or the length. A placement now
// carries the eight numbers a fight reads; this map's one placement resolves to
// nothing, so the record takes the constructor's charge of 8 and relax of 4 and
// keeps zero on the other six. The form is still 2118 bytes and still opens at
// 10.
//
// The value was DERIVED FORWARD, before the loader was changed: the pre-story
// form was taken, 8 and 4 written into the record's own two cadence slots, and
// the result hashed. So the literal below is not the new loader agreeing with
// itself. It is checked BACKWARD as well, on every run, by the zeroing below.
//
// 0070 MOVES IT AGAIN, and this one is established BACKWARDS ONLY — the method
// the version-9 and version-10 moves above ended up trusting, now used on its
// own. Version 11 appends the group word to every entity record and raises the
// version byte; this loader's one placement carries the map's own group id.
// Rather than reassemble the whole form outside the tree a fifth time, the two
// checks below take it apart: lift the four bytes out of the one record and put
// byte 0 back to 10, and what is left must hash to gfPreGroupDigest — a literal
// older than this story; do that AND zero the eight combat fields, and it must
// hash to gfPreCombatDigest, older than the story before it. Two anchors this
// story's code could not have chosen, so the value below is the form those two
// describe plus the group word and a version byte, and nothing else.
//
// 0071 MOVES IT AGAIN, by the same backwards-only method and with one more
// anchor. Version 12 appends the OWNER SLOT to every entity record and raises the
// version byte; this loader's one placement carries the map's own owner word,
// which this fixture leaves at zero. The chain below is now three deep: lift the
// four owner bytes out and put byte 0 back to 11, and what is left must hash to
// gfPreOwnerDigest; lift the group word out of THAT and put byte 0 back to 10,
// and it must hash to gfPreGroupDigest; zero the eight combat fields in THAT, and
// it must hash to gfPreCombatDigest. Three literals this story's code could not
// have chosen, so the value below is the form those three describe plus one word
// and a version byte, and nothing else.
//
// 0091 MOVES IT AGAIN, by that same backwards-only method and with one more
// anchor at the head of the chain: version 18 appends the SIGHT RANGE behind the
// decay block, this placement resolves to nothing so the range is the
// constructor's own 5, and that value is asserted as a number beside this digest.
//
// 0095 MOVES IT ONCE MORE, and by a DIFFERENT method from every move above:
// the group section is not a tail on this fixture's one record, it is a new
// counted block between the route section and the script section, so the
// derivation reads that section's own count rather than assuming a width.
// This fixture's placement carries no owner, so the section is a bare 4-byte
// zero count and the chain otherwise reaches exactly the value below —
// checked by stripping the section back off the real form and confirming
// that value falls out, not by an external assembly.
//
// 0096 MOVES IT ONCE MORE, on the SAME method 0095's own move used: the group
// section's record widens from 9 bytes to 18 (owner, group, base, order,
// commandedX, commandedY) but stays a counted block at the same position, and
// its own count is still read rather than assumed. This fixture's placement
// carries no owner, so the section is still a bare 4-byte zero count either
// way — the only byte that moved for this fixture is byte 0 itself, checked
// by the same backward chain below still reaching gfPreDecayDigest and every
// older pin unchanged: the peel that lifts this section back off reads its
// own count and multiplies by the record's new width, so it cuts exactly the
// same four bytes here that it always did.
//
// 0099 MOVES IT ONCE MORE, on the same method every peel above already used:
// the one record gains eighteen bytes at its own tail — the actor state,
// the ring and the leg — and this fixture's placement is never a
// patroller, so that tail is guard with an empty ring. The only byte that
// moved for this fixture beyond that tail is byte 0 itself, checked by the
// same backward chain below still reaching gfPreDecayDigest and every older
// pin unchanged.
//
// The values it replaces: version 21's 0xee8dabef04a140eb, version 20's
// 0xc2e68dc02a1a0819, version 19's
// 0x03d2399c2282f2e0, version 18's 0x44820ae7c11c587b, version 17's
// 0xad4fddcd0e315ead, version 13's 0x1c835cfb072e60d1, version 11's
// 0xa34f237fd5ce2305, version 10's 0xb5f72b1253c19040, version 9's
// 0xa1404d649cf5a755, version 8's 0xe9eb3c69d8aab382.
//
// 0103 MOVES IT AGAIN, on the same method every peel above already used:
// a new counted section — the ground sacks — lands between the group
// section and the script section, and this fixture's placement is never a
// ground record, so the section is a bare four-byte zero count. The only
// byte that moved for this fixture beyond that count is byte 0 itself,
// checked by the same backward chain below still reaching gfPreDecayDigest
// and every older pin unchanged.
//
// 0104 MOVES IT AGAIN, on the same method every peel above already used:
// one byte at the very tail of this fixture's one record — the reach —
// and this fixture's placement is never wired to a weapon, so the byte is
// the constructor's own floor of 1. The only byte that moved for this
// fixture beyond that one is byte 0 itself, checked by the same backward
// chain below still reaching gfPreDecayDigest and every older pin
// unchanged. The version-22 value it replaces was 0xa59d2f08ce28ff9a.
//
// 0106 MOVES IT AGAIN, on the same method every peel above already used:
// two int32 at the very tail of this fixture's one record — the post —
// and NewWorld writes it from the placement's own cell unconditionally, so
// the pair is always this fixture's own gfCellX, gfCellY. The only byte
// that moved for this fixture beyond those eight is byte 0 itself, checked
// by the same backward chain below still reaching gfPreDecayDigest and
// every older pin unchanged. The version-23 value it replaces was
// 0x9697af552efac26e.
//
// 0109 T1 MOVES IT AGAIN, on the same method every peel above already used:
// six fields at the very tail of this fixture's one record — the
// regeneration block — and this fixture's placement names no period and no
// mana pool, so all six are zero. The only byte that moved for this
// fixture beyond those six fields is byte 0 itself, checked by the same
// backward chain below still reaching gfPreDecayDigest and every older pin
// unchanged. The version-24 value it replaces was 0x52c151b127913cdd.
//
// 0109 T3 MOVES IT AGAIN, and this one is DELIBERATE, 0067's own kind: a
// placement is now born with its two regeneration periods, and this
// fixture's placement resolves to nothing, so the block's first two fields
// take the base constructor's 100 and 50 and the other four — the mana
// pair and both hundredths remainders — stay zero, exactly as the eight
// combat fields did at 0067 for the same reason. No version moves and no
// width moves, so the backward chain below is unchanged bit for bit; only
// the two bytes T1 left zero move. The value T1 left it at was
// 0x80e347991159a174.
//
// 0112 MOVES IT AGAIN, on the method 0095's own move introduced: the carry
// section and the purse section are two new blocks between the sack section
// and the script section, neither a tail on this fixture's one record nor a
// block closing the form, so the derivation reads the sections it already
// knows how to measure — the group section's own count, then the sack
// section's own walk — and cuts a flat, entity-count-derived span from where
// that walk ends. This fixture's placement carries nothing and no gold, so
// the pair is the one entity's own empty carry record plus the purse
// section's fixed zeroed bytes, checked by strippedOfTheCarryAndPurse below
// still reaching gfPreDecayDigest and every older pin unchanged. The
// version-25 value it replaces was 0x99e7ac5014a3346e.
//
// 0122 MOVES IT AGAIN, and this one moves NOTHING beyond byte 0: a
// compiled check's second player reference is a tail on pkg/sim's own
// check record, and this fixture's placement compiles no script at all, so
// the widened record multiplies zero checks and contributes no further
// byte. The version-26 value it replaces was 0x94f59e656923230d.
//
// 0117 T1 MOVES IT AGAIN, on the same method every peel above already
// used: one word off the very tail of this fixture's one record — the
// command group — and this loader gives no placement one, so the field is
// always zero. The only byte that moved for this fixture beyond that word
// is byte 0 itself, checked by the same backward chain below still
// reaching gfPreDecayDigest and every older pin unchanged. The version-31
// value it replaces was 0xc2dd248bca283f5c.
//
// 0124 T2 (pkg/sim) MOVES IT AGAIN, on the same method: the equipment
// section is a third block, between the carry section and the purse, and
// strippedOfTheCarryAndPurse's own span widened to cover it rather than
// gaining a peel of its own (fromalm_test.go). This fixture's placement
// equips nothing, so the added span is the one entity's own empty
// EquipSlots-wide zero record. The version-32 value it replaces was
// 0x17c610e6355a617b, and the value it moves gfDigest to is
// 0x49e491bbf5fbdb45.
//
// 0125 T2 MOVES IT AGAIN, on the same method every peel above already
// used: six SkillXP fields, a Mind, an experience value, a credited slot
// and a gains flag off the very tail of this fixture's one record — the
// experience-from-use block — and this loader fills none of the five yet,
// so every one of them is always zero. The only byte that moved for this
// fixture beyond those thirty-four is byte 0 itself, checked by the same
// backward chain below still reaching gfPreDecayDigest and every older pin
// unchanged. Its own predecessor is 0124's landed 0x49e491bbf5fbdb45 above
// — not the version-32 value 0x17c610e6355a617b the two stories would each
// have replaced independently had they landed apart — because this branch
// merges onto 0124 rather than the master either was written against.
//
// 0125 T4 MOVES IT AGAIN, and DELIBERATELY — 0067's and 0109 T3's own
// kind, not a version bump: no version moves and no width moves, so the
// backward chain below is unchanged bit for bit and only the bytes T2 left
// zero move. This fixture's one placement resolves to nothing, so the
// unresolved arm now substitutes the whole constructor definition for three
// of the five fields exactly as it already does for the speed and the sight
// range: Mind becomes the constructor's own 20, the experience value stays
// its zero, the credited slot stays SkillGeneral, the gains flag stays false
// — an unresolved placement does not gain — and the six SkillXP integers
// stay zero, nothing having been earned yet. The merged value below is
// obtained by running this test, not composed by hand from either branch's
// own pin.
//
// 0127 MOVES IT AGAIN, on the same method every peel above already used:
// one KnownSpells uint32 off the very tail of this fixture's one record —
// past the experience block — and the spell table's own bare two-byte
// zero count between the purse section and the script section. This
// loader gives no placement a mask and no fixture in this package ever
// names a table, so both are always zero. The only bytes that moved for
// this fixture beyond those six are byte 0 itself, checked by the same
// backward chain below still reaching gfPreDecayDigest and every older pin
// unchanged. Its own predecessor is 0125's landed 0x740dad2e1a8c014c
// above. The new value is obtained by running this test, not composed by
// hand.
//
// 0129 MOVES IT AGAIN, and this is the CHEAPEST move of the lot: a compiled
// instant's second unit reference widened the instant record by five bytes,
// and this fixture's world carries no script at all — no check, no instant,
// no trigger — so not one byte of its own form moved except byte 0 itself.
// The backward chain below is therefore unchanged bit for bit, which is what
// distinguishes this move from the peels above: there is nothing to peel,
// only the version byte to put back. Its own predecessor is 0127's landed
// 0x609c7bcd6263c3d3. The new value is obtained by running this test, not
// composed by hand.
//
// 0135 T1 MOVES IT AGAIN, on the same method every peel above already
// used: six int32 skill levels off the very tail of this fixture's one
// record — past the KnownSpells mask — and this loader gives no
// placement one yet, so every one of them is always zero. The only byte
// that moved for this fixture beyond those twenty-four is byte 0 itself,
// checked by the same backward chain below still reaching
// gfPreDecayDigest and every older pin unchanged. Its own predecessor is
// 0129's landed 0xec0f82c836866e55 above. The new value, 0x08d3d764ddbaf428,
// was obtained by running this test, not composed by hand.
//
// 0139 MOVES IT AGAIN, on the same method 0127's own move above used:
// WeaponSpell (a uint16) and WeaponSpellLevel (an int32) off the very tail
// of this fixture's one record, past the six skill levels. This fixture's
// placement resolves to no unit at all, so it is armed with no weapon and
// carries no spell either — both fields are always zero, on the spellbook
// peel's own reason above. The only byte that moved for this fixture
// beyond those six is byte 0 itself, checked by the same backward chain
// below still reaching gfPreDecayDigest and every older pin unchanged.
// Its own predecessor is 0135's landed 0x08d3d764ddbaf428 above. The new
// value is obtained by running this test, not composed by hand.
//
// 0156 MOVES IT AGAIN, and for this fixture the ONLY byte that moved is
// byte 0: the story adds three bytes to each compiled instant record and
// this fixture's map authors no script action, so its instant count is
// zero. strippedOfTheScriptItem states that, and the backward chain below
// still reaches gfPreDecayDigest and every older pin unchanged — which is
// what makes this re-pin a measurement rather than a paste. Its own
// predecessor is 0154's landed 0x17ef91e58b55af82.
//
// THE ITEM-ATTRIBUTION HOTFIX MOVES IT AGAIN, on the same method: six bytes
// off the very tail of this fixture's one record — the kill-credit source
// EntityID, its presence byte and the signed spell byte — past the corpse-loot
// suppression byte. Nothing in this package applies an effect, so all six are
// the zero the constructor leaves them at. The only byte that moved for this
// fixture beyond those six is byte 0 itself, checked by the same backward chain
// below still reaching gfPreDecayDigest and every older pin unchanged. Its own
// predecessor is 0156's landed 0x29645d4bda039117. The new value was obtained
// by running this test, not composed by hand.
//
// 1029 MOVES IT AGAIN, on the same method: two bytes off the very tail of this
// fixture's one record, the authored map id. gfMap()'s placement leaves
// alm.Unit.UnitID at 0, so both bytes are zero here and the peel is structural
// on the off-map bit's own reason. The story also widens each compiled check
// record by three bytes, and this fixture's map authors no condition, so its
// check count is zero and nothing moves there either: the only bytes that moved
// for this fixture are byte 0 and the two the record grew by. The backward chain
// below still reaches gfPreDecayDigest and every older pin unchanged, which is
// what makes this re-pin a measurement rather than a paste. Its own predecessor
// is the item-attribution hotfix's landed 0x77838696c7b106fd. The new value was
// obtained by running this test, not composed by hand.
//
// 1033 B3 MOVES IT AGAIN, and for this fixture the ONLY byte that moved is
// byte 0: the story adds a whole STRUCTURE SECTION, its own bare four-byte
// zero count, between 0166's own script-state section and the script
// section, and this fixture's map declares no structure. It also widens
// each compiled check and instant record by five bytes, and this fixture's
// map authors no script at all, so its check and instant counts are both
// zero and nothing moves there either. strippedOfTheStructureSection states
// the section's own claim, and the backward chain below still reaches
// gfPreDecayDigest and every older pin unchanged, which is what makes this
// re-pin a measurement rather than a paste. Its own predecessor is 1029's
// landed 0xc6b75e605d34ce74 above. The new value was obtained by running
// this test, not composed by hand.
//
// 1039 MOVES IT AGAIN: the unresolved placement carries five zero resistance
// bytes at the record tail. The version-58 value this replaces was
// 0xe4a2c4b83b0c6ebb; the backward peel below keeps every older pin fixed.
//
// 1047 MOVES IT AGAIN: the record carries the inactive turn pair (desired
// facing repeating current facing, remaining progress zero) at its tail and
// the version byte moves to 63. 1045's value this replaces was
// 0xb4011605065e7834.
//
// 1047 ROUND 2 MOVES IT A SECOND TIME, within the same story: the round-1
// value above, 0x9ca482137b5ca71b, was derived while this fixture's unresolved
// placement still carried RotationSpeed 0 (P1's own finding). Wiring the
// constructor default onto the unresolved arm (mapload's blockFor, per
// unitCtorDefaults) changes the VALUE this record's tail carries, not its
// shape: the record stays the width round 1 left it, no earlier peel's
// boundary moves, and the version byte stays 63. The chain below still
// reaches every older pin unchanged, which is what makes this a measurement.
//
// The pass-3 correction appends zero request-time duration to this inactive
// fixture and advances the form once more. strippedOfTurnDuration removes that
// byte before the historical peel chain, so every older pin remains fixed.
// 1052 widens the fixture's structure records with their map shape and moves
// the current form version.
// 1063 moves only the version byte for raw cloud counters: this fixture holds
// no area record.
const gfDigest uint64 = 0x44c4024682a16076

// gfPreDecayDigest is what this fixture hashed to before a record carried a decay
// ladder, and gfPreDecayFormVersion the version byte it carried then. The digest
// is the number gfDigest USED to be, so the derivation that produced it is not
// lost: it moves one peel down the chain and keeps doing the work it was made
// for.
const (
	gfPreDecayDigest      uint64 = 0xf444843a5e880e90
	gfPreDecayFormVersion byte   = 16
	// gfDecayBlockLen is how many bytes the ladder added to each record: the
	// stage, the dwell and the dying time.
	gfDecayBlockLen = 7
)

// gfPreSkillFormVersion is the version byte the form carries once the tail
// this peel removes is gone — version 38, 0129's own landed state (the
// second unit reference it added sits on the compiled instant record, not
// this one, so this fixture's own form is byte-for-byte the same as it
// always was save for byte 0). It is the NEWEST peel, ahead of the
// KnownSpells mask's, and it is STRUCTURAL rather than a zero check: it
// removes the tail's twenty-four bytes by offset and width alone, on the
// experience block's own reason — this loader gives no placement a skill
// level yet, so the field this peel removes has always been the zero the
// constructor leaves it at.
const gfPreSkillFormVersion byte = 38

// gfSkillLen is the width 0135 added to one entity record: six int32
// skill levels.
const gfSkillLen = 24

// gfPreSpellFormVersion is the version byte the form carries once the tail
// this peel removes is gone — version 35, 0125's own landed state (the
// experience-from-use block it added sits just below this record's own
// tail, untouched by this peel, so the bytes it leaves behind still carry
// it; tagging them 34 would claim a form that predates it). It was the
// newest peel until 0135's own skill block arrived above it, and then
// until 0139's own weapon-spell tail arrived above THAT; it is
// STRUCTURAL rather than a zero check: it removes the tail's four bytes by
// offset and width alone, on the experience block's own reason — this
// loader gives no placement a KnownSpells mask, so the field this peel
// removes has always been the zero the constructor leaves it at.
const gfPreSpellFormVersion byte = 35

// gfKnownSpellsLen is the width 0127 added to one entity record: one
// KnownSpells uint32.
const gfKnownSpellsLen = 4

// gfPreWeaponSpellFormVersion is the version byte the form carries once the
// tail this peel removes is gone — version 39, 0135's own landed state
// (its own skill block sits between the known-spells tail and this one,
// untouched by this peel, so the bytes it leaves behind still carry it;
// tagging them 38 would claim a form that predates it). It is the NEWEST
// peel now, ahead of the skill block's own, and it is STRUCTURAL rather
// than a zero check: it removes the tail's six bytes by offset and width
// alone, on the spellbook peel's own reason — this fixture's placement
// resolves to no unit at all, so no weapon and so no spell ever reaches
// it, and the field this peel removes has always been the zero
// WeaponSpell's own doc names as "none" (pkg/sim/world.go).
const gfPreWeaponSpellFormVersion byte = 39

// gfPreSpellStateFormVersion is the version byte the form carries once the
// tail this peel removes is gone — version 44, 0153's own landed state.
// It is the NEWEST peel now, ahead of the weapon spell's own, and it is
// STRUCTURAL rather than a zero check: it removes the tail's five bytes by
// offset and width alone, on the weapon-spell peel's own reason — nothing
// gives a placement an autocast and no cast has run, so the five bytes this
// peel removes have always been the zeros the constructor leaves them at.
const gfPreSpellStateFormVersion byte = 44

// gfSpellStateLen is the width 0154 added to one entity record: AutoSpell (a
// uint16), CastWait, SpellFX and SpellFXSpell (a byte each).
const gfSpellStateLen = 5

// gfWeaponSpellLen is the width 0139 added to one entity record: WeaponSpell
// (a uint16) and WeaponSpellLevel (an int32).
const gfWeaponSpellLen = 6

// gfPreExperienceFormVersion is the version byte the form carries once the
// tail this peel removes is gone — version 34, 0124's own landed state
// (the equipment section it added sits between the carry section and the
// purse, untouched by this peel, so the bytes it leaves behind still carry
// it; tagging them 32 would claim a form that predates it). It was the
// NEWEST peel until 0127's own spellbook tail arrived above it; it is
// STRUCTURAL rather than a zero check: it removes the tail's thirty-four
// bytes by offset and width alone, so it works the same now that T4 fills
// Mind with the constructor's own 20 for this fixture's unresolved
// placement (the other four of the five stay zero: no experience value, no
// credited slot, no gain, nothing earned) as it did when all five were
// still zero.
const gfPreExperienceFormVersion byte = 34

// gfExperienceLen is the width the experience-from-use block added to one
// entity record: six SkillXP int32, Mind, XPValue, XPSlot and GainsXP.
const gfExperienceLen = 34

// gfPreCommandGroupFormVersion is the version byte the form carried before
// pkg/sim's 0117 COMMAND GROUP existed — version 31, 0122's own landed
// state. It is no longer the newest peel — gfPreExperienceFormVersion
// above is — but it keeps its place ahead of the carry and purse
// sections', and it is STRUCTURAL rather than a zero check: it removes the
// tail's four bytes by offset and width alone, on the regeneration block's
// own reason — this loader gives no placement a command group either, so
// the field this peel removes has always been the zero the constructor
// leaves it at.
const gfPreCommandGroupFormVersion byte = 31

// gfCommandGroupLen is the width the command group added to one entity
// record: one uint32.
const gfCommandGroupLen = 4

// gfPreGroupSectionFormVersion is the version byte the form carried before
// pkg/sim's 0095 GROUP SECTION existed — distinct from gfPreGroupFormVersion
// below, which is the group WORD's own fixed point (version 10) rather than
// this record's (version 18). gfDigest above moves once more for this story,
// on the same terms every earlier move in its own comment already used: this
// fixture's placement carries no owner, so the section it writes is a bare
// 4-byte zero count, and the check below reads that count rather than
// assuming it, so a fixture that ever gained an owned placement would still
// peel correctly.
const gfPreGroupSectionFormVersion byte = 18

// gfPreActorStateFormVersion is the version byte the form carried before
// pkg/sim's 0099 ACTOR STATE TAIL existed, and gfActorStateTailLen is how many
// bytes that story added to the one record this fixture builds: the state
// byte, the two-cell ring and the leg. It WAS the newest peel; pkg/sim's
// 0103 sack section is, now — it comes off before the group section does,
// which comes off before this one does — the strippings come off newest
// story first.
const (
	gfPreActorStateFormVersion byte = 20
	gfActorStateTailLen             = 18
)

// gfPreSackSectionFormVersion is the version byte the form carried before
// pkg/sim's 0103 SACK SECTION existed — version 21, 0099's own landed
// state. gfSackSectionCountLen is that section's own record count width;
// this fixture's placement is never a ground record, so the section it
// writes is a bare four-byte zero count.
const (
	gfPreSackSectionFormVersion byte = 21
	gfSackSectionCountLen            = 4
)

// gfPreReachFormVersion is the version byte the form carried before
// pkg/sim's 0104 REACH TAIL existed — version 22, 0103's own landed state.
// It is no longer the newest peel — gfPrePostFormVersion below is — but it
// keeps its place ahead of the sack section's, which comes off before the
// group section does, which comes off before the actor state does — the
// strippings still come off newest story first. This fixture's placement is
// never wired to a weapon, so the byte the tail carries is always the
// constructor's own floor of 1.
const gfPreReachFormVersion byte = 22

// gfPrePostFormVersion is the version byte the form carried before
// pkg/sim's 0106 POST TAIL existed — version 23, 0104's own landed state.
// It is no longer the newest peel — gfPreRegenFormVersion below is — but it
// keeps its place ahead of the reach's. NewWorld writes the post from a
// placement's own cell unconditionally, so this peel always finds this
// fixture's own gfCellX, gfCellY at the tail of its one record, regardless
// of what a later loader arm supplies.
const gfPrePostFormVersion byte = 23

// gfPostLen is the width the post tail added to one entity record: two
// int32.
const gfPostLen = 8

// gfPreRegenFormVersion is the version byte the form carried before
// pkg/sim's 0109 REGENERATION BLOCK existed — version 24, 0106's own landed
// state. It is the NEWEST peel now, ahead of the post's, and it is
// STRUCTURAL rather than a zero check: it removes the tail's eighteen bytes
// by offset and width alone. Before T3 this fixture's placement named no
// period and no mana pool, so the peel always found six zero bytes there;
// T3 now fills the two periods with the base constructor's 100 and 50 on
// every path, and the peel keeps working unchanged because it never
// inspected the value.
const gfPreRegenFormVersion byte = 24

// gfRegenLen is the width the regeneration block added to one entity
// record: four int32 and two bytes.
const gfRegenLen = 18

// gfPreCostDigest is what this fixture hashed to before a cell carried a cost or
// a height: version 12's number, and a literal that predates this story, so
// nothing in it can be made to agree by changing the code being measured.
//
// gfDigest moved TWICE inside this story and the second move is the one worth
// naming: 0xb9bc03f29f1cd92f was the value while the two planes existed and the
// loader still handed over neither, so both were materialised — an all-8 cost
// plane and a flat height one. It is now the value with this map's OWN planes in
// them, so the difference between those two numbers is the whole of what
// deriving them bought.
const gfPreCostDigest uint64 = 0x9b1a357bd4747326

// gfPreOwnerDigest is what gfDigest's form hashes to once the owner slot is
// lifted out of its record and the version byte put back: the digest this fixture
// carried before the owner story, and a literal that predates it.

const gfPreOwnerDigest uint64 = 0xf3e7191404f35a57

// gfPreGroupDigest is what gfPreOwnerDigest's form hashes to once the group word
// is lifted out of its record as well and the version byte put back: the digest
// this fixture carried before the group story, and a literal that predates it.
const gfPreGroupDigest uint64 = 0x2a3d4e42292adac0

// gfPreRelationDigest is what this fixture hashed to before a world carried a
// RELATION, and gfPreRelationFormVersion the version byte it carried then.
//
// The first of the two is the number gfDigest USED to be — the one derived
// outside this tree — so the external derivation is not lost by this story, it
// moves one peel down the chain and keeps doing the work it was made for. What
// stands at gfDigest now is this tree's own answer, and the two are tied together
// by a truncation of a fixed length.
const (
	gfPreRelationDigest      uint64 = 0x8d318208fb4dbef6
	gfPreRelationFormVersion byte   = 14
)

// gfRelationLen is how long pkg/sim's relation block is, written out here for
// gfScriptSection's reason: the constant is unexported there and these tests
// cannot import it.
const gfRelationLen = 50 * 50

// castingSectionCountLen is the width of each of the casting section's two
// counts, restated here on this file's own rule for every other pkg/sim
// width it names by hand.
const castingSectionCountLen = 4

// gfPreFacingDigest is what this fixture hashed to before an entity carried a
// FACING, and gfPreFacingFormVersion the version byte it carried then. Both are
// literals that predate the facing story, so neither can be made to agree by
// changing the code they measure.
const (
	gfPreFacingDigest      uint64 = 0xef1dcc2a70a1099b
	gfPreFacingFormVersion byte   = 13
)

// gfPreCombatDigest is what gfPreGroupDigest's form hashes to once the eight
// fields' bytes are lifted out of its record as well: the digest this fixture
// carried before the story before that, and a literal that predates both.
const gfPreCombatDigest uint64 = 0x2b7043e15ca64b64

// The two appended words' offsets inside one entity record, the record's own
// width, and the two version bytes that predate them — written out from the
// contract exactly as gfRecordAt is.
const (
	gfGroupWordAt              = 83
	gfOwnerWordAt              = 87
	gfRecordLen                = 91
	gfPreGroupFormVersion byte = 10
	gfPreOwnerFormVersion byte = 11
)

// gfRecordAt is where this fixture's ONE entity record begins: after the form's
// 34-byte header and this map's own grid. Written out from the form's documented
// layout, exactly as the length assertion below is.
const gfRecordAt = 34 + gfW*gfH

// fnv1a is FNV-1a 64 written out from its published constants — a SECOND
// implementation, owing nothing to hash/fnv or to anything in this module. It is
// what makes the pin above check the bytes rather than merely sit beside them:
// the number is pinned as a number, and this function says it is that function
// of the bytes the tree actually produced.
func fnv1a(b []byte) uint64 {
	const (
		offset64 = 0xcbf29ce484222325
		prime64  = 0x100000001b3
	)
	h := uint64(offset64)
	for _, c := range b {
		h ^= uint64(c)
		h *= prime64
	}
	return h
}

// TestTheSecondFNVAgreesWithItsPublishedVectors checks the instrument before
// anything is measured with it. Were this implementation wrong, the cross-check
// below would agree with the pin for the wrong reason.
func TestTheSecondFNVAgreesWithItsPublishedVectors(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want uint64
	}{
		{"", 0xcbf29ce484222325},
		{"a", 0xaf63dc4c8601ec8c},
		{"foobar", 0x85944171f73967e8},
	} {
		if got := fnv1a([]byte(tc.in)); got != tc.want {
			t.Errorf("fnv1a(%q) = %#016x, want %#016x", tc.in, got, tc.want)
		}
	}
}

// TestTheDerivedPlaneIsCarriedHashedAndReadBack is AC-7 (SC-5).
func TestTheDerivedPlaneIsCarriedHashedAndReadBack(t *testing.T) {
	w := mapload.FromALM(gfMap())

	// The section, against the split expectation.
	want := gfWantGrid()
	got := gridSection(t, w)
	if len(got) != len(want) {
		t.Fatalf("the grid section is %d byte(s), want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the grid section differs at cell %d — (%d,%d) is %#02x, want %#02x",
				i, i%gfW, i/gfW, got[i], want[i])
		}
	}

	// The digest, against the number derived outside this tree, and the number
	// against the bytes through the second FNV.
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	legacy := strippedOfConsumables1090(form)
	legacy[0] = 66
	if h := fnv1a(legacy); h != 0xc3dba1f28973088b {
		t.Fatalf("structure-target version changed unit-only state: %#016x", h)
	}
	if h := fnv1a(strippedOfOriginalDead1100(form)); h != 0x2d3b37c6378551e2 {
		t.Fatalf("version69 control changed outside the dead section: %#016x", h)
	}
	if h := fnv1a(strippedOfSpellbook1101(form)); h != 0x114a1b269aa8fae3 {
		t.Fatalf("version70 control changed: %x", h)
	}
	if h := fnv1a(strippedOfSecondPhysical1104(form)); h != gfDigest {
		t.Errorf("the world hashes %#016x, pinned as %#016x", h, gfDigest)
	}
	if h := fnv1a(strippedOfSecondPhysical1104(form)); h != gfDigest {
		t.Errorf("FNV-1a of the %d-byte form is %#016x and the pin is %#016x — the two have to describe "+
			"one world, or the pin was carried over from something else", len(form), h, gfDigest)
	}
	// The chain back through THREE pre-story pins, through the SECOND FNV.
	// First lift the owner slot out of the one record and put the version byte
	// back: what is left must hash to exactly what this fixture hashed to
	// before the owner story. Then lift the group word out of that and put the
	// version byte back again, for what it hashed to before the group story.
	// Then zero the eight combat fields in that, and it must hash to what it
	// hashed to before the story before that. All three literals predate the
	// code being measured, so none can be made to agree by changing it. FIRST
	// the FACING, because the strippings come off newest story first: one byte
	// off each record's tail, and what is left must hash to exactly what this
	// fixture hashed to before the facing story. Then the two planes, and then
	// the three older literals against THAT. Its offset is past THREE planes,
	// unlike every peel below it: those run on a form the planes have already
	// come out of, where the record begins at gfRecordAt. FIRST the RELATION,
	// which is newer still and which closes the form, so its inverse is a
	// truncation: what is left must hash to what this fixture hashed to before
	// the relation story, which is the number an external program derived when
	// it was gfDigest. THE AUTOCAST PAIR AND THE SPELL EFFECT MARK come off
	// first of all now, being the NEWEST peel: AutoSpell (a uint16), CastWait,
	// SpellFX and SpellFXSpell (three bytes) off the very tail of this
	// fixture's one record, past the weapon spell. The peel is structural —
	// it removes those five bytes by offset and width, not by checking their
	// value — on the weapon-spell peel's own reason below: nothing in this
	// tree gives a placement an autocast, and no cast has run, so all five are
	// the zeros the constructor leaves them at.
	//
	// A COMPILED INSTANT'S OWN ITEM comes off BEFORE it, being the newest peel
	// of all (0156): a uint16 code and a presence byte per instant record. This
	// fixture's map authors no script action, so its instant count is zero and
	// the peel removes no bytes — the whole of that undo is the version byte,
	// which is the claim that 0156 wrote nothing into this fixture but the
	// version. strippedOfTheScriptItem carries the note, and it is what
	// preSpellState below is measured on top of. THE OFF-MAP BIT comes off
	// first of all now, being the NEWEST peel: one byte off the very tail of
	// this fixture's one record, past the spell state. The peel is structural
	// — it removes the byte by offset and width, not by checking its value
	// — on the spell state's own reason above: only a mission-script arm sets
	// the bit and no fixture in this package fires one, so it is always the
	// zero the constructor leaves it at.
	//
	// 0166'S OWN BYTES COME OFF IN FRONT OF IT NOW, being the newest peel:
	// the seven each record grew by and the script-state section between
	// the casting section and the script section, both structural on the
	// off-map bit's own reason.
	//
	// 1025'S OWN BYTES COME OFF IN FRONT OF THOSE NOW, being the newest peel:
	// the eight each record grew by -- the carried load and the carrying
	// capacity -- and the item-weight section between the spell table and the
	// casting section, both structural on the off-map bit's own reason.
	// THE STRUCTURE SECTION comes off in front of all of those now, being the
	// newest peel of all (1033 B3): its own bare four-byte zero count,
	// between 0166's own script-state section and the script section. This
	// fixture's map declares no structure, so the peel removes exactly those
	// four bytes and the version byte is put back, on strippedOfTheOffMap's
	// own STRUCTURAL reason.
	preOffMap := strippedOfTheOffMap(strippedOfTheCasting(strippedOfTheLastArms(strippedOfSpellEffects(strippedOfCorpseLoot(strippedOfItemAttribution(strippedOfCarriedWeight(strippedOfTheMapUnitID(strippedOfTheStructureSection(strippedOfWeaponResistance(form), len(w.Structures()))))))))))
	preScriptItem := strippedOfTheScriptItem(preOffMap)
	spellStateAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen + gfRegenLen + gfCommandGroupLen + gfExperienceLen + gfKnownSpellsLen + gfSkillLen + gfWeaponSpellLen
	noSpellState := append([]byte(nil), preScriptItem[:spellStateAt]...)
	noSpellState = append(noSpellState, preScriptItem[spellStateAt+gfSpellStateLen:]...)
	noSpellState[0] = gfPreSpellStateFormVersion

	// A WEAPON'S OWN SPELL comes off next, no longer the newest peel
	// (0139): WeaponSpell (a uint16) and WeaponSpellLevel (an int32) off
	// the very tail of this fixture's one record, past the six skill
	// levels. The peel is structural — it removes those six bytes by
	// offset and width, not by checking their value — on the spellbook
	// peel's own reason below: this fixture's placement resolves to no
	// unit at all (0139's own T3 wires no weapon to a placement that
	// resolves to nothing), so the field is always the zero WeaponSpell's
	// own doc names as "none" (pkg/sim/world.go). It runs on a COPY,
	// noWeaponSpell, so form itself still names the real encoded bytes for
	// the length check at the end of this test.
	preTreasure := strippedOfDeathGold(noSpellState)
	weaponSpellAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen + gfRegenLen + gfCommandGroupLen + gfExperienceLen + gfKnownSpellsLen + gfSkillLen
	noWeaponSpell := append([]byte(nil), preTreasure[:weaponSpellAt]...)
	noWeaponSpell = append(noWeaponSpell, preTreasure[weaponSpellAt+gfWeaponSpellLen:]...)
	noWeaponSpell[0] = gfPreWeaponSpellFormVersion

	// AN ENTITY'S SIX SKILL LEVELS come off next, no longer the newest peel
	// — a weapon's own spell above is: six int32 off the very tail of what
	// is left, past the KnownSpells mask. The peel is structural — it
	// removes those twenty-four bytes by offset and width, not by checking
	// their value — on the experience block's own reason: this loader
	// gives no placement a level yet, so the field is always the zero the
	// constructor leaves it at. It runs on a COPY, noSkill, over
	// noWeaponSpell rather than over form directly, so form itself still
	// names the real encoded bytes for the length check at the end of this
	// test.
	skillAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen + gfRegenLen + gfCommandGroupLen + gfExperienceLen + gfKnownSpellsLen
	noSkill := append([]byte(nil), noWeaponSpell[:skillAt]...)
	noSkill = append(noSkill, noWeaponSpell[skillAt+gfSkillLen:]...)
	noSkill[0] = gfPreSkillFormVersion

	// AN ENTITY'S OWN SPELLBOOK comes off next, no longer the newest peel —
	// the six skill levels above are: one KnownSpells uint32 off the very
	// tail of what is left, past the experience block. The peel is
	// structural — it removes those four bytes by offset and width, not by
	// checking their value — on the experience block's own reason: this
	// loader gives no placement a mask, so the field is always the zero the
	// constructor leaves it at. It runs on a COPY, noKnownSpells, so form
	// itself still names the real encoded bytes for the length check at the
	// end of this test.
	knownSpellsAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen + gfRegenLen + gfCommandGroupLen + gfExperienceLen
	noKnownSpells := append([]byte(nil), noSkill[:knownSpellsAt]...)
	noKnownSpells = append(noKnownSpells, noSkill[knownSpellsAt+gfKnownSpellsLen:]...)
	noKnownSpells[0] = gfPreSpellFormVersion

	// AN ENTITY'S EXPERIENCE FROM USE comes off next, no longer the newest
	// peel — the spellbook's own tail above is: the six slot experiences, a
	// Mind, an experience value, a credited slot and a gains flag off the
	// very tail of what is left, past the command group. The peel is
	// structural — it removes those thirty-four bytes by offset and width,
	// not by checking their value — so it works the same now that T4 fills
	// Mind with the constructor's own 20 (this fixture's placement resolves
	// to nothing; the other four of the five stay the zero the constructor
	// leaves them at) as it did when all five were still zero. It runs on a
	// COPY, noExperience, so form itself still names the real encoded bytes
	// for the length check at the end of this test.
	experienceAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen + gfRegenLen + gfCommandGroupLen
	noExperience := append([]byte(nil), noKnownSpells[:experienceAt]...)
	noExperience = append(noExperience, noKnownSpells[experienceAt+gfExperienceLen:]...)
	noExperience[0] = gfPreExperienceFormVersion

	// THE COMMAND GROUP comes off next, no longer the newest peel —
	// AN ENTITY'S EXPERIENCE FROM USE above is: one word off the very tail
	// of this fixture's one record, past the regeneration block. The peel
	// is structural — it removes those four bytes by offset and width, not
	// by checking their value — on the regeneration block's own reason:
	// this loader gives no placement one, so the field is always the zero
	// the constructor leaves it at. It runs on a COPY, noCommandGroup, so
	// form itself still names the real encoded bytes for the length check
	// at the end of this test.
	commandGroupAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen + gfRegenLen
	noCommandGroup := append([]byte(nil), noExperience[:commandGroupAt]...)
	noCommandGroup = append(noCommandGroup, noExperience[commandGroupAt+gfCommandGroupLen:]...)
	noCommandGroup[0] = gfPreCommandGroupFormVersion

	// THE CARRY, EQUIPMENT, PURSE AND SPELL TABLE SECTIONS come off next,
	// being the newest peel until this story (0112, widened in place by
	// 0124 T2 and again by 0127 — see strippedOfTheCarryAndPurse's own doc
	// in fromalm_test.go): the counted-carry-per-entity block, the
	// equipment block, the fixed purse block and the spell table's own bare
	// count, all between the sack section and the script section. This
	// fixture's placement is given nothing to carry, wears nothing, is
	// credited no gold and knows no spell, so the four strip to a flat,
	// entity-count-derived span — on strippedOfTheCarryAndPurse's own
	// general terms, reused unchanged from fromalm_test.go because it
	// locates the span by measurement rather than by an offset this
	// fixture's own shape would have to restate.
	noCarryAndPurse := strippedOfTheCarryAndPurse(noCommandGroup)

	// THE REGENERATION BLOCK comes off next, being the newest peel until
	// this story (0109): six fields off the very tail of this fixture's one
	// record, past the post. The peel is structural — it removes those
	// eighteen bytes by offset and width, not by checking their value — so
	// it works the same now that T3 fills the first two with the base
	// constructor's 100 and 50 (this fixture's placement resolves to
	// nothing) as it did when all six were still zero. It runs on a COPY,
	// noRegen, so form itself still names the real encoded bytes for the
	// length check at the end of this test.
	regenAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen
	noRegen := append([]byte(nil), noCarryAndPurse[:regenAt]...)
	noRegen = append(noRegen, noCarryAndPurse[regenAt+gfRegenLen:]...)
	noRegen[0] = gfPreRegenFormVersion

	// THE POST comes off next, being the newest peel until this story
	// (0106): eight bytes off the very tail of what is left, past the
	// reach. NewWorld writes it from the placement's own cell
	// unconditionally, so the pair is always this fixture's own gfCellX,
	// gfCellY regardless of what a later loader arm supplies. It runs on a
	// COPY, noPost, so form itself still names the real encoded bytes for
	// the length check at the end of this test.
	postAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1
	noPost := append([]byte(nil), noRegen[:postAt]...)
	noPost = append(noPost, noRegen[postAt+gfPostLen:]...)
	noPost[0] = gfPrePostFormVersion

	// THE REACH comes off next, being the newest peel until this story
	// (0104): one byte off the very tail of what is left, past the actor
	// state, the ring and the leg. This task wires no placement to a
	// weapon, so the byte is the constructor's own floor of 1 regardless of
	// what a later loader arm supplies. It runs on a COPY, noReach, so form
	// itself still names the real encoded bytes for the length check at the
	// end of this test.
	reachAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen
	noReach := append([]byte(nil), noPost[:reachAt]...)
	noReach = append(noReach, noPost[reachAt+1:]...)
	noReach[0] = gfPreReachFormVersion

	// THE SACK SECTION comes off next, being the newest peel until this
	// story (0103): the counted block between the group section and the
	// script section. This fixture places no ground record, so it is a bare
	// four bytes; its own count is read rather than assumed — on the group
	// section's own rule below — past the group section's own count, which
	// is read the same way and is also zero for this fixture. It runs on a
	// COPY, noSacks, so form itself still names the real encoded bytes for
	// the length check at the end of this test.
	sackGroupSectionAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 4
	sackGroupCount := int(binary.LittleEndian.Uint32(noReach[sackGroupSectionAt : sackGroupSectionAt+4]))
	sackSectionAt := sackGroupSectionAt + 4 + 18*sackGroupCount
	sackCount := int(binary.LittleEndian.Uint32(noReach[sackSectionAt : sackSectionAt+4]))
	sackEnd := sackSectionAt + 4
	for i := 0; i < sackCount; i++ {
		items := int(binary.LittleEndian.Uint32(noReach[sackEnd+12 : sackEnd+16]))
		sackEnd += 16 + 2*items
	}
	noSacks := append([]byte(nil), noReach[:sackSectionAt]...)
	noSacks = append(noSacks, noReach[sackEnd:]...)
	noSacks[0] = gfPreSackSectionFormVersion

	// THE ACTOR STATE TAIL comes off next, being the newest peel until this
	// story (0099): eighteen bytes off the very tail of this fixture's one
	// record — the state byte, the two-cell ring and the leg — and the
	// version byte put back to 20. This placement is never a patroller, so
	// the tail is guard with an empty ring, the same shape pkg/sim's own
	// TestThePinIsThePreStoryPinPlusTheActorState pins.
	actorStateAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1
	noActorState := append([]byte(nil), noSacks[:actorStateAt]...)
	noActorState = append(noActorState, noSacks[actorStateAt+gfActorStateTailLen:]...)
	noActorState[0] = gfPreActorStateFormVersion

	// THE GROUP SECTION comes off next, being the newest peel until this
	// story: the counted block between the one route (empty, this fixture's
	// placement never stepped) and the script section. Its own count is read
	// rather than assumed, so a fixture that ever gained an owned placement
	// would still peel correctly; this one names no owner, so the count is 0
	// and the section is its bare four bytes.
	groupSectionAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + 4
	groupCount := int(binary.LittleEndian.Uint32(noActorState[groupSectionAt : groupSectionAt+4]))
	preGroupSection := append([]byte(nil), noActorState[:groupSectionAt]...)
	// The record is 18 bytes since 0096's own version 20 (owner, group, base,
	// order, commandedX, commandedY) — grown from the 9 it opened at, in place.
	// This fixture's placement carries no owner, so groupCount is 0 and the
	// multiplier does not bite here either way; it is kept correct anyway, on
	// this file's own claim that a fixture gaining an owned placement would
	// still peel correctly.
	preGroupSection = append(preGroupSection, noActorState[groupSectionAt+4+18*groupCount:]...)
	preGroupSection[0] = gfPreGroupSectionFormVersion

	// THE SIGHT RANGE comes off next: one byte off the
	// very tail of this fixture's one record, behind the decay block. Nothing is
	// put back in its place, because the peel below is what names a version byte.
	sightAt := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen
	preSight := append([]byte(nil), preGroupSection[:sightAt]...)
	preSight = append(preSight, preGroupSection[sightAt+1:]...)

	// THEN THE DECAY BLOCK: seven bytes off the tail of what is left, and the
	// version byte put back.
	decayAt := 34 + 3*gfW*gfH + gfRecordLen + 1
	preDecay := append([]byte(nil), preSight[:decayAt]...)
	preDecay = append(preDecay, preSight[decayAt+gfDecayBlockLen:]...)
	preDecay[0] = gfPreDecayFormVersion
	if h := fnv1a(preDecay); h != gfPreDecayDigest {
		t.Errorf("lifting the decay block out hashes %#016x, want the pre-story %#016x — "+
			"this story moved a byte outside the seven each record grew by", h, gfPreDecayDigest)
	}
	preRelation := append([]byte(nil), preDecay[:len(preDecay)-gfRelationLen]...)
	preRelation[0] = gfPreRelationFormVersion
	if h := fnv1a(preRelation); h != gfPreRelationDigest {
		t.Errorf("lifting the relation out hashes %#016x, want the pre-story %#016x — "+
			"this story moved a byte outside the block that closes the form", h, gfPreRelationDigest)
	}
	facingAt := 34 + 3*gfW*gfH + gfRecordLen
	preFacing := append([]byte(nil), preRelation...)
	preFacing = append(preFacing[:facingAt], preFacing[facingAt+1:]...)
	preFacing[0] = gfPreFacingFormVersion
	if h := fnv1a(preFacing); h != gfPreFacingDigest {
		t.Errorf("lifting the facing out hashes %#016x, want the pre-story %#016x — "+
			"this story moved a byte outside it", h, gfPreFacingDigest)
	}
	prePlanes := strippedOfThePlanes(preFacing)
	if h := fnv1a(prePlanes); h != gfPreCostDigest {
		t.Errorf("lifting the two planes out hashes %#016x, want the pre-story %#016x — "+
			"this story moved a byte outside them", h, gfPreCostDigest)
	}
	preOwner := append([]byte(nil), prePlanes...)
	preOwner = append(preOwner[:gfRecordAt+gfOwnerWordAt], preOwner[gfRecordAt+gfRecordLen:]...)
	preOwner[0] = gfPreOwnerFormVersion
	if h := fnv1a(preOwner); h != gfPreOwnerDigest {
		t.Errorf("lifting the owner slot out hashes %#016x, want the pre-story %#016x — "+
			"this story moved a byte outside it", h, gfPreOwnerDigest)
	}
	preGroup := append([]byte(nil), preOwner...)
	preGroup = append(preGroup[:gfRecordAt+gfGroupWordAt], preGroup[gfRecordAt+gfOwnerWordAt:]...)
	preGroup[0] = gfPreGroupFormVersion
	if h := fnv1a(preGroup); h != gfPreGroupDigest {
		t.Errorf("lifting the group word out hashes %#016x, want the pre-story %#016x — "+
			"this story moved a byte outside it", h, gfPreGroupDigest)
	}
	stripped := append([]byte(nil), preGroup...)
	for b := gfRecordAt + combatFieldsFrom; b < gfRecordAt+combatFieldsTo; b++ {
		stripped[b] = 0
	}
	if h := fnv1a(stripped); h != gfPreCombatDigest {
		t.Errorf("zeroing the eight combat fields hashes %#016x, want the pre-story %#016x — "+
			"an earlier story's bytes moved too", h, gfPreCombatDigest)
	}
	// The length the external program assembled, stated so a form that grew a
	// section fails here rather than silently moving the digest. carryCountLen
	// and purseLen are the one entity's own empty carry record and the fixed
	// purse block 0112 added, in that order, right after the sack section;
	// gfCommandGroupLen is the word 0117 added to the record's own tail,
	// equipRecordLen is the one entity's own empty equipment record 0124 T2
	// added between the carry section and the purse, gfExperienceLen is
	// the thirty-four bytes 0125 T2 added to the record's own tail, behind
	// the command group, gfKnownSpellsLen is the four bytes 0127 added to
	// the record's own tail behind that, gfSkillLen is the twenty-four
	// bytes 0135 T1 added to the record's own tail behind that,
	// gfWeaponSpellLen is the six bytes 0139 added behind that,
	// gfSpellStateLen is the five bytes 0154 added behind that, and
	// spellCountLen is the spell table's own bare count 0127 added right
	// after the purse section.
	if wantLen := 34 + 3*gfW*gfH + gfRecordLen + 1 + gfDecayBlockLen + 1 + gfActorStateTailLen + 1 + gfPostLen + gfRegenLen + gfCommandGroupLen + gfExperienceLen + gfKnownSpellsLen + gfSkillLen + gfWeaponSpellLen + gfSpellStateLen + offMapLen + lastArmsLen + spellEffectsLen + corpseLootLen + itemAttributionLen + carriedWeightLen + mapUnitIDLen + weaponResistanceLen + withdrawalThresholdLen + 1 + 2 + 1 + 32 + 15 + 113 + 2 + 1 + 35 + 4 + groupSectionCountLen +
		gfSackSectionCountLen + carryCountLen + equipRecordLen + 16 + purseLen + spellCountLen + itemWeightCountLen + 2*castingSectionCountLen + relationSlotsInForm + cellTailCountLen + structureCountLen + 4 + 8 + 189 + gfScriptSection + 4 + 4 + 4 + gfRelationLen + 5 + 4 + 4 + absentSavedPlayerFooterLen + absentNativeStrideFooterLen + absentSavedMotionFooterLen + absentSavedCellFooterLen + absentSavedObjectsFooterLen + absentCarriedResumeFooterLen + 28 + entityIDFloorFooterLen + 4 + 26; len(form) != wantLen {
		t.Errorf("the tick-0 form is %d byte(s), want %d — a 34-byte header, THREE planes of %d cells, "+
			"one %d-byte record, one empty route, the empty group section, the empty sack section, the "+
			"empty carry section, the empty equipment section, the purse section, the empty spell table, "+
			"the structure section, the script section and the relation",
			len(form), wantLen, gfW*gfH, gfRecordLen+2+gfDecayBlockLen+gfActorStateTailLen+1+gfPostLen+gfRegenLen+gfCommandGroupLen+gfExperienceLen+gfKnownSpellsLen+gfSkillLen+gfWeaponSpellLen+gfSpellStateLen+offMapLen+lastArmsLen+spellEffectsLen+corpseLootLen+itemAttributionLen+carriedWeightLen+mapUnitIDLen+weaponResistanceLen+withdrawalThresholdLen+1+2+1)
	}
	// And the byte this story added carries the constructor's own range, this
	// placement having resolved to no entry at all — so the digest above is the
	// merged form plus a value stated here rather than plus whatever was to hand.
	if got := w.Entities()[0].ScanRange; got != 5 {
		t.Errorf("the unresolved placement carries a sight range of %d, want the constructor's 5", got)
	}

	// A tick in which the unit ROUTES, so the round trip below carries a stored
	// route and the decoder's own ground-bit refusal is exercised rather than
	// assumed. The target is across the water run at row 11, so the route it
	// takes is one that had to go round something.
	sim.Step(w, []sim.Command{{Entity: 0, X: 14, Y: 14}})
	const gfTurnBound = 20
	for i := 0; ; i++ {
		if e := w.Entities()[0]; e.X != gfCellX || e.Y != gfCellY {
			break
		}
		if i >= gfTurnBound {
			t.Fatalf("the unit stands on its start cell %v after %d tick(s), so it routed nowhere and the "+
				"round trip below carries no route", [2]int32{gfCellX, gfCellY}, gfTurnBound)
		}
		sim.Step(w, nil)
	}

	walked, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after the tick: %v", err)
	}
	if base := 34 + 3*gfW*gfH + 34 + 4; len(walked) <= base {
		t.Fatalf("the form after the tick is %d byte(s) and an empty-route form is %d — no route is stored, "+
			"so the decode below checks no cell against the grid", len(walked), base)
	}

	// The round trip, compared BYTE FOR BYTE. A digest comparison alone is what
	// this criterion exists to distrust.
	var back sim.World
	if err := back.UnmarshalBinary(walked); err != nil {
		t.Fatalf("the walked world does not read back: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary of the decoded world: %v", err)
	}
	if !bytes.Equal(again, walked) {
		t.Errorf("the decoded world marshals to %d byte(s) against the original's %d; first difference at "+
			"offset %d", len(again), len(walked), firstDifference(again, walked))
	}
	if g := gridSection(t, &back); !bytes.Equal(g, want) {
		t.Errorf("the decoded world's grid section differs from the contract's; first difference at cell %d",
			firstDifference(g, want))
	}
}

func TestOneDerivedCellMovesTheDigest(t *testing.T) {
	base := mapload.FromALM(gfMap())

	m := gfMap()
	const px, py = 12, 12
	if got := gfInterior[py-8][px-8]; got != 0x00 {
		t.Fatalf("cell (%d,%d) is already %#02x in the contract's table; pick an open one", px, py, got)
	}
	m.Tiles[py*gfW+px] = 0x2041 // bit 13

	moved := mapload.FromALM(m)
	if moved.Hash() == base.Hash() {
		t.Errorf("two maps differing in one derived cell both hash %#016x", base.Hash())
	}

	a, b := gridSection(t, base), gridSection(t, moved)
	diffs := 0
	for i := range a {
		if a[i] != b[i] {
			diffs++
		}
	}
	if diffs != 1 {
		t.Errorf("%d grid cells differ between the two maps, want exactly 1 — the digest above moved for "+
			"more than the one cell this test perturbed", diffs)
	}
	if b[py*gfW+px] != 0x01 {
		t.Errorf("the perturbed cell derived as %#02x, want 0x01", b[py*gfW+px])
	}
}
