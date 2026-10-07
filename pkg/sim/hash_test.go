package sim

import "testing"

// fnv1a is FNV-1a 64 written out from its published constants — a SECOND
// implementation, owing nothing to hash/fnv or to anything in this package. It
// is what lets the two pins check each other: the bytes are pinned from the
// format contract, the digest is pinned as a number, and this function says the
// number really is that function of those bytes. Were both pins simply recorded
// from one run of our own code, they would agree with each other no matter what
// the code did.
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

// pinDigest is the pin's other half: the digest of the version-22 world
// binary_test.go pins the bytes of, re-derived from the hand-transcribed bytes
// through this file's own FNV — never captured from the encoder. Nothing about
// it is game-derived — it is a digest of our own encoding of our own state, our
// own generator's seed included.
//
// Version 22 appends THE SACK SECTION, a second counted block sitting between
// the group section and the script section (0103) — nothing before it
// moves — so the version-21 pin 0x0ac517775052f960 does not carry over BY
// ARITHMETIC — nor did the version-20 pin 0xa40e25c8032db332, the version-19
// pin 0xf57a5717a5a538e1, the version-18 pin 0x7a94e926fd68cc91, the
// version-17 pin 0xb942b5d55f697fac, the version-11 pin 0xb56c622a377738cf,
// version 10's 0x9cfe358e8b5c1fae, version 9's 0x78f86b36b0e9819b, version 8's
// 0x5f7130b20e450804, version 7's 0x3386ccb86a0330b8, version 5's
// 0x4835b2e24f792300 or version 4's 0x66b2f30bc889dee4 before it — and the
// reason is worth stating rather than assuming. FNV-1a is a left fold: the
// digest of a prefix determines the digest of an extension of that prefix, and
// nothing else about one form's digest is recoverable from another's. Here
// byte 0 itself moved, so the two forms share no prefix at all and every
// later multiplication carries the difference — there is no arithmetic that
// turns the old number into this one without hashing the new bytes.
//
// Version 23 appends THE REACH, one byte at the tail of every entity record
// (0104) — nor does the version-22 pin 0xbc1dc5d5eef60d85 before it carry
// over BY ARITHMETIC, on the same ground: byte 0 moved again.
//
// Version 24 appends THE POST, two int32 at the tail of every entity record
// (0106) — nor does the version-23 pin 0xc400c725bab385c3 before it carry
// over BY ARITHMETIC, on the same ground: byte 0 moved again.
//
// Version 25 appends THE REGENERATION BLOCK, six fields at the tail of
// every entity record (0109) — nor does the version-24 pin
// 0x612dd5f8c12ccb31 before it carry over BY ARITHMETIC, on the same
// ground: byte 0 moved again.
//
// Version 26 appends THE CARRY SECTION AND THE PURSE, two blocks between
// the sack section and the script section (0112) — nor does the version-25
// pin 0x6ddd5aa12841e3ec before it carry over BY ARITHMETIC, on the same
// ground: byte 0 moved again.
//
// Version 31 appends the compiled check's SECOND PLAYER REFERENCE, a tail
// on the check record alone (0122) — nor does the version-26 pin
// 0xc428573dacfd4a7f before it carry over BY ARITHMETIC, on the same
// ground: byte 0 moved again. It contributes no OTHER byte here: the
// pinned world carries no script, so the check record's new width — 63 to
// 73 — multiplies zero checks and the section stays the same 1421 zeros it
// always was. Versions 27 through 30 are not pinned separately; this story
// never wrote one (0122 D-1).
//
// Version 32 appends THE COMMAND GROUP, one word at the tail of every
// entity record (0117) — nor does the version-31 pin 0xede6d696093ba13e
// before it carry over BY ARITHMETIC, on the same ground: byte 0 moved
// again.
//
// Version 34 appends THE EQUIPMENT SECTION, one block between the carry
// section and the purse section (0124 T2) — nor does the version-32 pin
// 0xb9105ae160cf28d9 before it carry over BY ARITHMETIC, on the same
// ground: byte 0 moved again. Version 33 is not pinned separately; this
// tree never wrote one — it is allocated to a lane running in parallel off
// the same master this one branched from.
//
// Version 35 appends AN ENTITY'S EXPERIENCE FROM USE, six slot experiences,
// a Mind, an experience value, a credited slot and a gains flag at the tail
// of every entity record (0125) — nor does the version-34 pin
// preEquipPinDigest's own successor (binary_test.go's preExperiencePinDigest,
// obtained the same way every pin in this file is: by running the test that
// derives it) carry over BY ARITHMETIC, on the same ground: byte 0 moved
// again. This branch merges onto 0124, so the pin directly beneath this
// story's own bump is 0124's version-34 one, not the version-32 pin an
// unmerged 0125 would have replaced.
//
// Version 36 appends AN ENTITY'S OWN SPELLBOOK, four bytes at the tail of
// every entity record, AND THE SPELL TABLE, a new counted section between
// the purse section and the script section (FR-4b) — nor does the
// version-35 pin (this constant's own predecessor, 0x2cf7c16babbc3952,
// 0125's own landed state) carry over BY ARITHMETIC, on the same ground:
// byte 0 moved again.
//
// Version 38 appends THE INSTANT'S SECOND UNIT REFERENCE, five bytes at the
// tail of every instant record — nor does the version-36 pin (this
// constant's own predecessor, 0x81fdab4547c8625d, 0127's own landed state)
// carry over BY ARITHMETIC, on the same ground: byte 0 moved again. It
// contributes no OTHER byte here: the pinned world carries no script, so the
// instant record's new width — 59 to 64 — multiplies zero instants and
// the section stays the same 1421 zeros it always was. Version 37 is not
// pinned separately; this tree never wrote one — it is allocated to a
// story running in parallel off the same master this one branched from.
//
// Version 39 appends AN ENTITY'S SIX SKILL LEVELS, twenty-four bytes at the
// tail of every entity record — nor does the version-38 pin (this
// constant's own predecessor, 0x920d2c848eafa34b, 0129's own landed state)
// carry over BY ARITHMETIC, on the same ground: byte 0 moved again.
// TestThePinIsThePreStoryPinPlusTheSkill in binary_test.go, run directly on
// this constant's own bytes, is what checks that nothing else moved:
// stripped of exactly the twenty-four bytes per record this story added, it
// must fall back to the version-38 value above.
//
// Version 41 appends A WEAPON'S OWN SPELL, a uint16 and an int32 at the tail
// of every entity record — nor does the version-39 pin (this constant's
// own predecessor, 0x00c2f0597f237b9e, 0135's own landed state) carry over
// BY ARITHMETIC, on the same ground: byte 0 moved again. Version 40 is not
// pinned separately; this tree never wrote one — it is allocated to a
// story running in parallel off the same master this one branched from.
// TestThePinIsThePreStoryPinPlusTheWeaponSpell in binary_test.go, run
// directly on this constant's own bytes, is what checks that nothing else
// moved: stripped of exactly the six bytes per record this story added, it
// must fall back to the version-39 value above.
//
// This value was computed from the hand-transcribed bytes of pinBytes —
// pinHead() followed by pinGroupSection(), the sack section's own bare
// four-byte zero count, the carry section's own three bare four-byte zero
// counts, the equipment section's own three bare EquipSlots-wide zero
// records, the purse section's 200 zeros, the spell table's own bare
// two-byte zero count, the script section's 1421 zeros, the relation's
// 2500, the known-spells tail's own three bare four-byte zero blocks, then
// 0135's own skill tail's three bare twenty-four-byte zero blocks, and then
// 0139's own weapon-spell tail's three bare six-byte zero blocks, each at
// the new end of every entity record — through this file's own fnv1a,
// which TestFNV1aAgreesWithItsPublishedVectors checks against the published
// vectors before anything is measured with it, and the test below checks
// this constant against those bytes through it, so the transcription and
// the number each have to be right and neither was taken from the encoder.
// TestHashIsPinned then checks the constructed pinWorld's own Hash()
// against this same constant, so the transcription and the encoder have to
// agree as well. BOTH STORIES EXTEND THE OLDER PEEL CHAIN, on 0127's own
// precedent rather than 0129's: 0135's own addition sits exactly where the
// entity-record peel chain operates from, and 0139's own tail sits above
// even that, so TestThePinIsThePreStoryPinPlusTheExperience and everything
// beneath it in binary_test.go now run on pinBytesPreSkill and
// rtfBytesPreSkill, which are themselves now built from
// pinBytesPreWeaponSpell and rtfBytesPreWeaponSpell rather than from
// pinBytes and rtfBytes directly (binary_test.go's own doc on each) — those
// derivations are otherwise unchanged and still reach the values they
// always did.
//
// 0156 MOVES IT AGAIN, and this world's script section is a world with no
// script at all — three zero counts and no record — so the compiled
// instant's new item tail contributes no byte and the whole of the bump is
// byte 0. The transcription above carries 0x2e now, and the two derivations
// still check each other: this constant against the transcribed bytes
// through fnv1a, and pinWorld's own Hash() against this constant. The
// version-45 value it replaces was 0xfaa49109d55c12fe.
//
// 1029 MOVES IT AGAIN. This world's entity records each gain two bytes of
// authored map id at +275, all zero because no entity in it was placed by a
// map, and byte 0 carries 0x39. Its script section is still a world with no
// script at all, so the widened check record contributes no byte.
// TestThePinIsThePreviousPinPlusTheAuthoredMapID (binary_test.go) peels that
// tail back off and requires what is left to hash to the value below, which is
// the number this constant carried before this story.
// The version-56 value it replaces was 0x3af68fef64afc381.
//
// 1033 MOVES IT AGAIN (B3). This world declares no structure, so the new
// structure section is its own bare four-byte zero count and byte 0 carries
// 0x3a. TestThePinIsThePreviousPinPlusTheStructureSection (binary_test.go)
// peels that section back off and requires what is left to hash to the
// value below, which is the number this constant carried before this story.
// The version-57 value it replaces was 0x077f2a83b4415c20.
//
// 1039 MOVES IT AGAIN. Five zero resistance bytes are appended to every
// entity in this fixture and byte 0 carries version 59. The version-58 value
// it replaces was 0x86a572758c6baf4f; binary_test.go peels exactly this tail.
//
// 1037 appends the two zero withdrawal thresholds to every fixture entity and
// moves the version byte once. Its independent peel recovers the version-59
// digest above.
//
// 1047 first appends the inactive turn pair (desired facing repeating the
// current facing, remaining progress zero), then its pass-3 correction
// appends zero request-time duration. widenedTurnProgressPin and
// widenedTurnDurationPin independently build the two tails; each peel test
// recovers the prior form rather than treating the new digest as evidence of
// its own shape. 1052 widens each structure record with its immutable map
// shape. This fixture has no structures, so only the form-version byte
// moves. 1063 moves that byte again for the raw cloud counter; this fixture
// has no area record. Form 73 adds one zero provenance byte per entity. The
// form-72 digest is independently retained by
// TestOriginalProfile1107GenuinePredecessorAndIndependentPins. Form76:
// independently transcribed form75 plus three zero 35-byte bindings. Form77:
// independently transcribed form76 plus the five absent-clock bytes. Form78:
// independently transcribed form77 plus the four absent-Group-span bytes,
// through widenedSessionClockPin's own call into widenedSavedGroupPin
// (savedgroups_test.go) rather than a separate pinBytesV77 rung — the
// pinned world carries no saved Group, so the span is explicitly absent
// (savedgroupsbinary.go's own "a zero span is explicitly absent"). Form79:
// independently transcribed form78 plus the four absent-structure-span
// bytes, through widenedSavedGroupPin's own call into
// widenedSavedStructurePin (savedstructures1114_test.go) rather than a
// separate pinBytesV78 rung — the pinned world carries no saved structure,
// so the span is explicitly absent. Form80: independent form79 bytes plus
// four absent-Player-container bytes. Form81: independent form80 bytes plus
// four absent-stride bytes. The previous digest remains a literal control in
// TestNativeStride1115IndependentWireAndHistoricalAbsence. Form82 adds the
// absent fine-motion footer; the genuine form81 digest remains a literal
// control in TestSavedMotion1115LegacyExplicitAbsence. Form83 adds only the
// absent raw-plane footer; the exact form82 digest is retained in
// TestSavedCellPlanes1115Historical82LiteralPinDigests. Form84 adds only the
// absent current-object footer; the form83 digest is retained by
// TestSavedCellPlanes1115Historical83LiteralPinDigests. Form85 adds only the
// absent carried-resume footer; the form84 digest is retained by
// TestSavedCellPlanes1115Historical83LiteralPinDigests's own predecessor
// check on strippedCurrentObjectsPin/widenedCurrentObjectsPin. Form87 adds
// only its four zero counter-footer bytes and version tag.
const pinDigestV91 uint64 = 0xa6bc7f477735bfd6

var pinDigest = entityIDFloorDigest1177(pinBytes, pinDigestV91, 10)

const prePlayerContainersPinDigest uint64 = 0x87a0d78b68e856b2

// TestFNV1aAgreesWithItsPublishedVectors checks the instrument before anything
// is measured with it. If this second implementation were wrong, every
// cross-check below would agree with the pin for the wrong reason.
func TestFNV1aAgreesWithItsPublishedVectors(t *testing.T) {
	cases := []struct {
		in   string
		want uint64
	}{
		{"", 0xcbf29ce484222325},
		{"a", 0xaf63dc4c8601ec8c},
		{"foobar", 0x85944171f73967e8},
	}
	for _, tc := range cases {
		if got := fnv1a([]byte(tc.in)); got != tc.want {
			t.Errorf("fnv1a(%q) = %#016x, want %#016x", tc.in, got, tc.want)
		}
	}
}

func TestHashIsPinned(t *testing.T) {
	requireOriginalDeadLegacyDigest(t, pinWorld(t), 0x522fc66a13b1e0b6)
	requireHumanMovementLegacyDigest(t, pinWorld(t), 0xfef480bf7535ab85)
	if got := pinWorld(t).Hash(); got != pinDigest {
		t.Errorf("the pinned world hashes %#016x, pinned as %#016x", got, pinDigest)
	}
}

func TestThePinnedDigestIsFNV1aOfThePinnedBytes(t *testing.T) {
	if got := fnv1a(pinBytes); got != pinDigest {
		t.Errorf("FNV-1a of the pinned bytes is %#016x, but the digest is pinned as %#016x — "+
			"the two pins have to describe the same world", got, pinDigest)
	}
}

func TestHashIsFNV1aOverExactlyTheByteForm(t *testing.T) {
	b, ents := sample()
	worlds := []*World{
		pinWorld(t),
		mustWorld(t, 0, Bounds{}, nil),
		mustWorld(t, 0xffffffffffffffff, b, ents),
		populated(t),
	}
	for i, w := range worlds {
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("world %d: MarshalBinary: %v", i, err)
		}
		if got, want := w.Hash(), fnv1a(form); got != want {
			t.Errorf("world %d hashes %#016x; FNV-1a of its %d-byte form is %#016x",
				i, got, len(form), want)
		}
	}
}

// ---------------------------------------------------------------- AC-6

func hashBase(t *testing.T) *World {
	t.Helper()
	return mustWorldGrid(t, 0x2468ace02468ace0, Bounds{Width: 6, Height: 5}, ModeCanonical, []byte{
		0, 1, 0, 0, 2, 0,
		0, 0, 1, 0, 0, 3,
		2, 0, 0, 1, 0, 0,
		0, 0, 0, 0, 1, 0,
		1, 0, 2, 0, 0, 0,
	}, []Entity{
		{ID: 1, X: 4, Y: 5, TargetX: 9, TargetY: 9, Class: 12, HasTarget: true, Stall: 2,
			HP: 60, MaxHP: 90},
		{ID: 6, X: -2, Y: 7, Class: -3, HP: 12, MaxHP: 12},
	})
}

// TestEveryFieldChangesTheDigest is AC-6.
func TestEveryFieldChangesTheDigest(t *testing.T) {
	cases := []struct {
		name   string
		change func(*World)
	}{
		{"the tick", func(w *World) { w.tick++ }},
		{"an entity's X", func(w *World) { w.entities[0].X++ }},
		{"an entity's Y", func(w *World) { w.entities[1].Y++ }},
		{"an entity's id", func(w *World) { w.entities[0].ID = 2 }},
		{"an entity's target", func(w *World) { w.entities[0].TargetX++ }},
		{"an entity's class id", func(w *World) { w.entities[0].Class++ }},
		{"a target set where there was none", func(w *World) { w.entities[1].HasTarget = true }},
		{"a target cleared", func(w *World) { w.entities[0].HasTarget = false }},
		{"the RNG state", func(w *World) { w.rng.state++ }},
		{"the bounds width", func(w *World) { w.bounds.Width++ }},
		{"the bounds height", func(w *World) { w.bounds.Height++ }},
		{"an entity's stall count", func(w *World) { w.entities[0].Stall++ }},
		// The health pair, and it is changed on the unit holding NO target: a
		// health that moved the target fields with it would change the digest for
		// a reason this case is not about, and one of the two edits below leaves
		// that unit dead.
		{"an entity's health", func(w *World) { w.entities[1].HP++ }},
		{"an entity's health maximum", func(w *World) { w.entities[1].MaxHP++ }},
		{"an entity killed", func(w *World) { w.entities[1].HP = -1 }},
		{"an entity downed", func(w *World) { w.entities[1].HP = 0 }},
		{"the routing mode", func(w *World) { w.mode = ModeOptimised }},
		{"a grid cell's blocks-ground bit", func(w *World) { w.grid[7] ^= 1 }},
		{"a grid cell's blocks-air bit", func(w *World) { w.grid[7] ^= 2 }},
		{"a grid cell nothing here reads", func(w *World) { w.grid[29] ^= 2 }},
		{"an added entity", func(w *World) {
			w.entities = append(w.entities, Entity{ID: 9, X: 1, Y: 1})
			w.routes = append(w.routes, nil)
		}},
		{"a route where there was none", func(w *World) {
			w.entities[0].TargetX, w.entities[0].TargetY = 4, 4
			w.routes[0] = []cell{{4, 4}}
		}},
		{"one cell of a route", func(w *World) {
			w.entities[0].TargetX, w.entities[0].TargetY = 4, 4
			w.routes[0] = []cell{{3, 4}, {4, 4}}
		}},
		{"a route's length", func(w *World) {
			w.entities[0].TargetX, w.entities[0].TargetY = 4, 4
			w.routes[0] = []cell{{2, 3}, {3, 4}, {4, 4}}
		}},
	}

	unchanged := hashBase(t).Hash()
	got := make([]uint64, len(cases))
	for i, tc := range cases {
		w := hashBase(t)
		tc.change(w)
		got[i] = w.Hash()
		if got[i] == unchanged {
			t.Errorf("changing %s left the digest at %#016x", tc.name, unchanged)
		}
	}
	// And no two changes collide, so the digest says which field moved and not
	// merely that something did.
	for i := range got {
		for j := i + 1; j < len(got); j++ {
			if got[i] == got[j] {
				t.Errorf("%s and %s both hash %#016x", cases[i].name, cases[j].name, got[i])
			}
		}
	}
}

// TestDigestsAgreeAtEveryTickWhateverTheInsertionOrder is AC-3: the digest is a
// function of the logical world, so two worlds differing only in the order their
// entities were handed over are the same world at every tick.
func TestDigestsAgreeAtEveryTickWhateverTheInsertionOrder(t *testing.T) {
	b := Bounds{Width: 16, Height: 16}
	ents := []Entity{
		{ID: 3, X: 0, Y: 0, TargetX: 5, TargetY: 2, HasTarget: true},
		{ID: 11, X: 7, Y: 7, TargetX: 1, TargetY: 9, HasTarget: true},
	}
	reversed := []Entity{ents[1], ents[0]}

	forward := mustWorld(t, 0x51ee9, b, ents)
	backward := mustWorld(t, 0x51ee9, b, reversed)

	if forward.Hash() != backward.Hash() {
		t.Fatalf("at tick 0 the digests are %#016x and %#016x", forward.Hash(), backward.Hash())
	}
	cmds := []Command{{Entity: 11, X: 2, Y: 2}, {Entity: 3, X: 12, Y: 4}}
	for i := 1; i <= 8; i++ {
		Step(forward, cmds)
		Step(backward, cmds)
		cmds = nil
		if forward.Hash() != backward.Hash() {
			t.Fatalf("at tick %d the digests are %#016x and %#016x",
				i, forward.Hash(), backward.Hash())
		}
	}
}
