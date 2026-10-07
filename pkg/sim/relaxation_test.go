package sim

// SC-9: the grid on which the relaxation order is DISCRIMINATED rather than
// asserted.
//
// Two shapes both satisfy every acceptance criterion this story states, and both
// are deterministic. The contract's one keeps a single label plane and reads it
// where it stands, so a cell lowered earlier in a generation is read at its new
// value by every frontier cell after it. The other — a batched wave — snapshots
// the labels, writes a fresh plane, and tests each offer against the snapshot.
//
// On this grid they part. Generation 1 labels (2,2) at 2 and (2,3) at 3, and
// both neighbour (1,2). Relaxed in frontier order, (2,1) writes a diagonal 6
// into (1,2), (2,2) lowers it to an orthogonal 4, and (2,3)'s diagonal 6 is then
// refused against the 4 that is already there — so (1,2) is 4, the target is 7,
// and the route is (2,2) (1,2) (0,1). A batched wave writes all three offers into
// a fresh plane, (2,3)'s lands last, (1,2) is 6, the target 9, and the route is
// (2,1) (1,0) (0,1): the unit stands on (2,1) after one tick instead of (2,2),
// so the world's byte form and its digest differ from tick 1 onward.
//
// The route is pinned cell by cell and the tick-1 digest with it. The digest is
// derived by hand from the layout the format contract states, through FNV-1a's
// published constants, and never captured from a run of this package.

import "testing"

// rlxBounds and rlxGrid are the fixture: five by five, with the blocks-ground
// bit set at (1,1) and nowhere else.
var rlxBounds = Bounds{Width: 5, Height: 5}

func rlxGrid() []byte {
	g := make([]byte, 25)
	g[1*5+1] = blockGround
	return g
}

// rlxSeed is the world's seed, and it is 1 so that the byte form the digest
// below is taken over can be read off this file: the rng state is the seed and
// nothing in a step consumes it.
const rlxSeed = 1

// rlxUnit starts at (3,2) and is ordered to (0,1).
var rlxUnit = Entity{ID: 1, X: 3, Y: 2}

const rlxTargetX, rlxTargetY = 0, 1

// rlxTick1Digest is FNV-1a over the version-20 byte form of this world at tick 1:
// a 34-byte header carrying tick 1, rng state 1, bounds 5x5, one entity,
// canonical mode and 25 grid cells; the 25 cells with a 1 at index 6 and zero
// everywhere else; then one 92-byte record — id 1 at (2,2), target (0,1) still
// held, class 0, presence 1, stall 0, health 0/0 — no health system, so alive and
// walking — the movement domain 0, the ground mover a world built naming none
// holds, a speed of 0 with no transit, the unrated mover such a world holds too,
// and a group rate term of 0, this order having been a plain one, and a
// THIRTY-NINE-BYTE attack block of zeroes, this unit attacking nobody and
// carrying none of the seven numbers a blow reads, and closing on a GROUP WORD of
// 0 — a real group, and the one an entity built without naming one is in — an
// OWNER SLOT of 0, which is no owner, this world being built from no map, and a
// FACING of 0xc0, WEST: the unit stepped (3,2) to (2,2), and a mover ends a tick
// facing the cell it stepped to; then
// that unit's STORED ROUTE, a count of 2 and the cells (1,2) and (0,1). A hundred
// and seventy bytes before the script section, and 1591 with it.
//
// The route is the far search's answer with its first cell consumed: the unit
// landed on (2,2), which was the route's own cell 0, so that cell and nothing
// else was dropped. It is in the form because it is canonical state, and this
// digest is where a replay would meet it. Then THE GROUP SECTION 0095 adds — a
// count of 0 and no record, this world naming no owned entity — and then the
// SCRIPT SECTION: 1421 zeros, this world running no script and no register,
// latch or counter having moved.
//
// The bytes above were assembled from that description outside this tree and
// hashed there, which is what makes this a derivation rather than a recording.
// None of the values it replaces — version 20's 0x299bb28bd86d0fc1 being the
// last, version 19's 0xc2d6171c57f328be before it, version 18's
// 0x18eabdf2701d7017 before that, version 13's 0x82b1a55740160530 before
// that, version 11's 0x5bcf10d095620c72 before that, version 10's
// 0x1877f13583b8654d before that and version 9's 0x44d5a2db48a59ae4 before
// that — carries over by arithmetic: FNV-1a is a left fold, so only a PREFIX
// of a form determines the digest of an extension of it, and byte 0 itself
// moved every time, so each was derived afresh. This unit is never a
// patroller, so 0099's own tail is guard with an empty ring on this record
// exactly as the constructor writes into any other — the one byte that moved
// between the version-20 form and the version-21 one, 0x299bb28bd86d0fc1,
// was byte 0 itself.
//
// 0103 MOVES IT AGAIN, on the same ground: this world names no owned unit
// and no sack, so its group section and its new sack section are each a bare
// four-byte count of zero, and the byte that moved between the version-21
// form and this one is again byte 0 alone.
//
// 0104 MOVES IT AGAIN: this unit resolves no weapon, so the reach byte its
// record's new tail carries is the constructor's own floor of 1, and the
// byte that moved between the version-22 form (0x040fee15d520e84e) and this
// one is once more byte 0 alone.
//
// 0106 MOVES IT AGAIN: this unit's post is carried WHOLE at the record's new
// tail — NewWorld writes it from rlxUnit's own construction cell (3,2)
// unconditionally, and a walk never moves it, so the two coordinates the
// record's new tail carries are that cell's, not the (2,2) the unit steps
// to on this tick. The byte that moved between the version-23 form
// (0x93a0bf9ca55e5b6c) and this one is once more byte 0 alone.
//
// 0109 MOVES IT AGAIN: this unit names no mana pool and no period, so the
// six fields the record's new tail carries are every one of them zero, and
// the byte that moved between the version-24 form (0x93b55424172ce94a) and
// this one is once more byte 0 alone.
//
// 0112 MOVES IT AGAIN: this world names no stock and credits no gold, so
// the carry section this unit's own record precedes is a bare four-byte
// zero count and the purse section is 200 zeros, both between the sack
// section and the script section, and the byte that moved between the
// version-25 form (0xc5bdb419e5da4b29) and this one is once more byte 0
// alone. This value is taken from a run of this story's own tree, not
// derived by hand, because the two new sections are wide enough that
// transcribing them adds nothing the length does not already check.
//
// 0122 MOVES IT AGAIN: this world compiles no script at all, so the
// compiled check record's widened tail — the two player references T2
// adds — multiplies zero checks and contributes no byte here either; the
// byte that moved between the version-26 form (0x594f3ff7a1a6d7b4) and
// this one is once more byte 0 alone. Taken from a run of this tree, on
// the same ground the version-25 replacement above was.
//
// 0117 MOVES IT AGAIN: this task gives the command group no writer, so the
// one word the record's new tail carries is zero, exactly as every field
// this task adds no writer for already is — and the byte that moved between
// the version-31 form (0xd9d5b0891d76207f) and this one is once more byte 0
// alone.
//
// 0124 T2 MOVES IT AGAIN: this world equips nothing, so the equipment
// section between the carry section and the purse is one bare
// EquipSlots-wide zero record, and the byte that moved between the
// version-32 form (0x35ec61a2904e2ec2) and this one is once more byte 0
// alone. Taken from a run of this story's own tree, on the same ground.
//
// 0125 MOVES IT AGAIN: this unit is wired to no gaining class, so the
// thirty-four bytes the record's new tail carries are every one of them
// zero, exactly as every field this task adds no writer for already is —
// and the byte that moved between the version-34 form (0x356d884bf1c092dc,
// 0124's own above — not the version-32 form an unmerged 0125 would have
// replaced) and this one is once more byte 0 alone.
//
// 0127 MOVES IT AGAIN: this unit is built with no KnownSpells mask and
// this world names no spell table, so the four bytes the record's new
// tail carries and the two-byte spell table right after the purse section
// are each zero, and the byte that moved between the version-35 form
// (0x74d6e30953a4478b, immediately above) and this one is once more byte
// 0 alone. Taken from a run of this story's own tree, on 0112's own
// ground above: the two new shapes are together wide enough that
// transcribing them by hand adds nothing the length check does not
// already give.
//
// 0129 T2 MOVES IT AGAIN: this world compiles no script at all, so the
// instant record's widened tail — 59 to 64 — multiplies zero instants and
// contributes no byte beyond byte 0 itself, and the byte that moved
// between the version-36 form (0x3f55becf5b592d46, immediately above) and
// this one is once more byte 0 alone. Version 37 is not pinned separately;
// this tree never wrote one.
//
// 0135 T1 MOVES IT AGAIN: this unit names no skill level, so the
// twenty-four bytes the record's new tail carries are every one of them
// zero, exactly as every field this task adds no writer for already is —
// and the byte that moved between the version-38 form (0x08b91b9cbbd8eff0,
// immediately above) and this one is once more byte 0 alone. Taken from a
// run of this story's own tree, on 0112's own ground above: the block is
// wide enough that transcribing it by hand adds nothing the length check
// does not already give.
//
// 0139 MOVES IT AGAIN: this unit resolves no weapon, so the weapon-spell
// tail its record's new end carries — a uint16 and an int32 — is zero
// throughout, and the byte that moved between the version-39 form
// (0x0354f3c7321bf817, immediately above) and this one is once more byte 0
// alone. Version 40 is not pinned separately; this tree never wrote one.
//
// 0156 MOVES IT AGAIN: this world runs no script, so the compiled instant's
// new item tail contributes no byte and the byte that moved between the
// version-45 form (0x1d23f05b0ca922af, immediately above) and this one is
// once more byte 0 alone.
//
// 1029 MOVES IT AGAIN: this world is built from no map, so the authored
// map id at the end of the single entity record is zero, and this world
// runs no script, so the widened check record multiplies zero checks. The
// bytes that moved between the version-56 form (0x0c3b081c28a30316) and
// this one are byte 0 and the two zero bytes appended to the one entity
// record. Taken from a run of this branch's tree, on the same ground the
// version-25 replacement above was.
//
// 1033 MOVES IT AGAIN (B3): this world declares no structure, so the new
// structure section is its own bare four-byte zero count. The bytes that
// moved between the version-57 form (0xbfe76df332890129) and this one are
// byte 0 and those four bytes. Taken from a run of this branch's tree, on
// the same ground the version-56 replacement above was.
//
// 1039 MOVES IT AGAIN: this unit carries five zero resistance bytes at the
// record tail. The version-58 value it replaces was 0xc3b5681800df7558.
//
// 1047 MOVES IT AGAIN: this unit's turn is inactive after tick 1 (the entity
// carries RotationSpeed 0, so requestFacing snaps in place with no interval),
// so its two new bytes are the current facing repeated and a zero remainder.
// The pre-story value it replaces was 0x08910f6f91575e0d.
// 1052 moves the form version; this fixture has no structure records. 1063
// moves it again for raw cloud counters; this fixture has no area record.
const rlxTick1Digest uint64 = 0xd1284972bdcf3746

// TestTheRelaxationOrderDecidesThisRoute pins the search's answer on the fixture
// directly: the labels the frontier order produces, and the route read back out
// of them.
func TestTheRelaxationOrderDecidesThisRoute(t *testing.T) {
	w := mustWorldGrid(t, rlxSeed, rlxBounds, ModeCanonical, rlxGrid(), []Entity{rlxUnit})
	s := newRouteScratch(w)

	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, rlxTargetX, rlxTargetY)
	checkRoute(t, "(3,2) to (0,1) past the block at (1,1)", got, ok,
		[]cell{{2, 2}, {1, 2}, {0, 1}})

	start := cell{rlxUnit.X, rlxUnit.Y}
	// The two labels the two shapes disagree about, and the one they agree on
	// that gets the disagreement started.
	checkLabel(t, w, s, start, 2, 2, 8)
	checkLabel(t, w, s, start, 2, 3, 12)
	checkLabel(t, w, s, start, 1, 2, 16)
	checkLabel(t, w, s, start, 0, 1, 28)
	// (2,1) and (1,0) are the batched wave's route through this grid. They are
	// labelled here too — the shapes differ in what they COST, not in what they
	// reach — so a case asserting them unlabelled would witness nothing.
	checkLabel(t, w, s, start, 2, 1, 12)
	checkLabel(t, w, s, start, 1, 0, 24)
}

// TestTheTickOneDigestIsTheContractsRoutesFirstCell is the same fact where a
// replay would meet it: after one tick the unit stands on the first cell of the
// contract's route, and the world hashes to the number this file derives from
// the format's own layout. A batched wave puts the unit on (2,1) and moves the
// digest.
func TestTheTickOneDigestIsTheContractsRoutesFirstCell(t *testing.T) {
	w := mustWorldGrid(t, rlxSeed, rlxBounds, ModeCanonical, rlxGrid(), []Entity{rlxUnit})
	Step(w, []Command{{Entity: rlxUnit.ID, X: rlxTargetX, Y: rlxTargetY}})

	// Facing WEST: the step (3,2) to (2,2), which is the facing the digest below
	// is derived over.
	want := Entity{ID: 1, X: 2, Y: 2, TargetX: rlxTargetX, TargetY: rlxTargetY, HasTarget: true,
		Facing: facingOfDir(6), DesiredFacing: facingOfDir(6), ActorState: actorStateGuard, Reach: 1,
		PostX: rlxUnit.X, PostY: rlxUnit.Y}
	if got := w.Entities()[0]; got != want {
		t.Errorf("after one tick the unit is %+v, want %+v", got, want)
	}
	if got := w.Tick(); got != 1 {
		t.Fatalf("the world is at tick %d", got)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if want := 61 + 34 + 3*25 + relationLen + 492 + 4 + 2*8 + groupCountLen + sackCountLen + carryCountLen + equipRecordLen + treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(1)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(form) != want {
		t.Fatalf("the tick-1 form is %d bytes, want %d", len(form), want)
	}
	// The route the form carries, cell for cell: the far search's answer less the
	// cell this tick walked.
	checkRoute(t, "the route stored after one tick", w.routes[0], true, []cell{{1, 2}, {0, 1}})
	requireStructureLegacyDigest(t, w, 0xd45ca209fc6365b1)
	requireOriginalDeadLegacyDigest(t, w, 0x95de18590d4aef56)
	requireSpellbookLegacyDigest(t, w, 0x2a69a9fd4d8f4b25)
	if got := fnv1a(strippedWorldOfSecondPhysical(form)); got != rlxTick1Digest {
		t.Errorf("the tick-1 world hashes %#016x, derived by hand as %#016x\n form % x",
			got, rlxTick1Digest, form)
	}
	requireHumanMovementLegacyDigest(t, w, 0xf4e50a61b5b7f893)
}

// TestTheWalkNeverStandsOnTheBlockedCell carries the fixture to arrival, because
// a route is searched afresh every tick and a pin on tick 1 says nothing about
// tick 2. The unit reaches its order, and at no tick does it stand on a cell
// whose blocks-ground bit is set or outside the map.
func TestTheWalkNeverStandsOnTheBlockedCell(t *testing.T) {
	w := mustWorldGrid(t, rlxSeed, rlxBounds, ModeCanonical, rlxGrid(), []Entity{rlxUnit})
	grid := rlxGrid()

	arrived := 0
	for tick := 1; tick <= 12; tick++ {
		if tick == 1 {
			Step(w, []Command{{Entity: rlxUnit.ID, X: rlxTargetX, Y: rlxTargetY}})
		} else {
			Step(w, nil)
		}
		e := w.Entities()[0]
		if e.X < 0 || e.Y < 0 || e.X >= rlxBounds.Width || e.Y >= rlxBounds.Height {
			t.Fatalf("tick %d: the unit is at (%d,%d), outside the map", tick, e.X, e.Y)
		}
		if grid[e.Y*rlxBounds.Width+e.X]&blockGround != 0 {
			t.Fatalf("tick %d: the unit is standing on the blocked cell (%d,%d)", tick, e.X, e.Y)
		}
		if e.X == rlxTargetX && e.Y == rlxTargetY {
			if e.HasTarget {
				t.Errorf("tick %d: the unit arrived still holding its target", tick)
			}
			arrived = tick
			break
		}
	}
	if arrived != 3 {
		t.Errorf("the unit arrived at tick %d, want 3 — three cells, one per tick", arrived)
	}
}

// ------------------------------------------------- the second grid: the hybrid wave
//
// The grid above kills the batched wave and lets the HYBRID one through, because
// the hybrid keeps the minimum of a generation's offers in the plane exactly as
// the contract does — it differs only in where a frontier cell reads its OWN
// label: from a snapshot taken before the generation began, rather than from the
// plane as it stands. On the 5x5 no frontier cell is lowered by an earlier cell
// of its own frontier, so there is nothing there for a stale read to get wrong.
//
// This grid is where that read decides a route. Eight by eight, blocking (3,2)
// (4,2) (1,3) (2,3); one unit at (0,7) ordered to (4,1).
//
//	. . . . . . . .        Generation 4 labels (2,2) at 14, from (3,3) diagonally.
//	. . . . . . . .        Generation 5's frontier holds (0,2), (1,2), (2,2), … in
//	. . . # # . . .        that order, and (1,2) at 11 lowers (2,2) to an
//	. # # . . . . .        orthogonal 13 BEFORE (2,2) is itself relaxed. The
//	. . . . . . . .        contract reads that 13 and labels (3,1) at 16; the
//	. . . . . . . .        hybrid reads the stale 14 and labels it 17. In the same
//	. . . . . . . .        generation (5,2) at 15 labels the target 18, so
//	@ . . . . . . .        generation 6 stops on it and (3,1) is never corrected.
//
// The walk back from the target at 18 is where the one label costs a route. Under
// the contract (3,1) offers an orthogonal 16+2 = 18, taken first in scan order,
// and (5,2)'s diagonal 15+3 = 18 is then refused as an equal diagonal. Under the
// hybrid (3,1) offers 19, (5,1) at 17 offers an equal orthogonal 19 and displaces
// it, and (5,2)'s 18 is strictly less and takes the cell — so the route leaves
// along the diagonal instead of the west wall.
//
// Both routes cost 18. They part at the FIRST step, (0,6) against (1,6), so the
// unit stands somewhere else after one tick and the digest moves there.

// hybBounds and hybGrid are the fixture: eight by eight with four blocked cells,
// arranged so that the two ways out of (0,7) are a west-wall detour and a clean
// diagonal of equal cost.
var hybBounds = Bounds{Width: 8, Height: 8}

func hybGrid() []byte {
	g := make([]byte, 64)
	for _, c := range []cell{{3, 2}, {4, 2}, {1, 3}, {2, 3}} {
		g[c.y*8+c.x] = blockGround
	}
	return g
}

// hybUnit starts in the bottom-left corner and is ordered to (4,1).
var hybUnit = Entity{ID: 1, X: 0, Y: 7}

const hybTargetX, hybTargetY = 4, 1

// hybTick1Digest is FNV-1a over the version-20 byte form of this world at tick 1:
// a 34-byte header carrying tick 1, rng state 1, bounds 8x8, one entity,
// canonical mode and 64 grid cells; the 64 cells with a 1 at indices 19, 20, 25
// and 26 and zero everywhere else; then one 92-byte record — id 1 at (0,6),
// target (4,1) still held, class 0, presence 1, stall 0, health 0/0, movement
// domain 0, speed 0, no transit, a group rate term of 0, a thirty-nine-byte
// attack block of zeroes, a group word of 0, an owner slot of 0 and a FACING of
// 0 — NORTH, the step (0,7) to (0,6), and the one record in this file whose
// facing byte is the zero it was built with rather than one a step wrote; then its stored
// route, a count of 7 and the cells (0,5) (0,4) (0,3) (1,2) (2,1) (3,1) (4,1) —
// the eight-cell route above with the cell this tick walked dropped from its
// head; then THE GROUP SECTION — a count of 0 and no record, this world naming
// no owned entity — and then the script section's own 1421 zeros.
//
// This world names no owned entity, so its group section is the bare
// four-byte count either way, and the wider record 0096's story adds to that
// section's OWN records moves nothing here — the one byte that moved between
// the version-19 form and the version-20 one was byte 0 itself. The
// version-20 value was 0x413f390a51e41451; this unit is never a patroller,
// so 0099's own tail is guard with an empty ring, and the byte that moved
// between that form and the version-21 one (0x1097247112e3d5e7) is again
// byte 0 alone. The version-19 value
// before it was 0xfab071e2ee5c6a8c, the version-18 value before that was
// 0x40db93b44ac9c083, the version-13 value before that was
// 0x5038edf3ceaee3ca, version 11's was 0x9528b1cc98c786b8, version 10's was
// 0x32a13e90447c7c81 and version 9's was 0x1a0da4948d0a037e; byte 0 moved at
// every one of those, so none of them carries over by arithmetic.
//
// 0103 MOVES IT AGAIN, on the same ground: this world names no owned unit
// and no sack, so its group section and its new sack section are each a bare
// four-byte count of zero, and the byte that moved between the version-21
// form and this one is again byte 0 alone.
//
// 0104 MOVES IT AGAIN: this unit resolves no weapon, so the reach byte its
// record's new tail carries is the constructor's own floor of 1, and the
// byte that moved between the version-22 form (0x714f888047823d62) and this
// one is once more byte 0 alone.
//
// 0106 MOVES IT AGAIN: this unit's post is carried WHOLE at the record's new
// tail — NewWorld writes it from hybUnit's own construction cell (0,7)
// unconditionally, and a walk never moves it, so the two coordinates the
// record's new tail carries are that cell's, not the (0,6) the unit steps
// to on this tick. The byte that moved between the version-23 form
// (0xdef6f721e2103516) and this one is once more byte 0 alone.
//
// 0109 MOVES IT AGAIN: this unit names no mana pool and no period, so the
// six fields the record's new tail carries are every one of them zero, and
// the byte that moved between the version-24 form (0x2d029989474e53b8) and
// this one is once more byte 0 alone.
//
// 0112 MOVES IT AGAIN: this world names no stock and credits no gold, so
// the carry section this unit's own record precedes is a bare four-byte
// zero count and the purse section is 200 zeros, both between the sack
// section and the script section, and the byte that moved between the
// version-25 form (0x58ca3d8590f2cd99) and this one is once more byte 0
// alone. This value is taken from a run of this story's own tree, not
// derived by hand, on rlxTick1Digest's own ground: the two new sections
// are wide enough that transcribing them adds nothing the length does not
// already check.
//
// 0122 MOVES IT AGAIN, on rlxTick1Digest's own ground: this world compiles
// no script, so the check record's widened tail contributes no byte beyond
// byte 0 itself. The version-26 value it replaces was 0x4d243c088ef387f2.
//
// 0117 MOVES IT AGAIN: this task gives the command group no writer, so the
// one word the record's new tail carries is zero, exactly as every field
// this task adds no writer for already is — and the byte that moved between
// the version-31 form (0xba2d48d034daa35f) and this one is once more byte 0
// alone.
//
// 0124 T2 MOVES IT AGAIN, on rlxTick1Digest's own ground: this world equips
// nothing, so the equipment section between the carry section and the
// purse is one bare EquipSlots-wide zero record, and the byte that moved
// between the version-32 form (0xe5b4ffe3d4aaf880) and this one is once
// more byte 0 alone.
//
// 0125 MOVES IT AGAIN: this unit is wired to no gaining class, so the
// thirty-four bytes the record's new tail carries are every one of them
// zero, exactly as every field this task adds no writer for already is —
// and the byte that moved between the version-34 form (0x1fbf92327580b0ca,
// 0124's own above — not the version-32 form an unmerged 0125 would have
// replaced) and this one is once more byte 0 alone.
//
// 0127 MOVES IT AGAIN, on rlxTick1Digest's own ground: this unit is built
// with no KnownSpells mask and this world names no spell table, so the
// four bytes the record's new tail carries and the two-byte spell table
// right after the purse section are each zero. The version-35 value it
// replaces was 0x4d871390c363763b.
//
// 0129 T2 MOVES IT AGAIN, on rlxTick1Digest's own ground: this world
// compiles no script either, so the instant record's widened tail
// contributes no byte beyond byte 0 itself. The version-36 value it
// replaces was 0x5ee03e98767b2ea4.
//
// 0135 T1 MOVES IT AGAIN, on rlxTick1Digest's own ground: this unit names
// no skill level, so the twenty-four bytes the record's new tail carries
// are every one of them zero. The version-38 value it replaces was
// 0x5d5d986482ee6b4e.
//
// 0139 MOVES IT AGAIN, on rlxTick1Digest's own ground: this unit resolves
// no weapon either, so the weapon-spell tail its record's new end carries
// is zero throughout. The version-39 value it replaces was
// 0x35fba89861fdcf37.
//
// 0156 MOVES IT AGAIN, on rlxTick1Digest's own ground: this world runs no
// script either. The version-45 value it replaces was 0xd7bec4d0373b5ed7.
//
// 1029 MOVES IT AGAIN, on rlxTick1Digest's own ground: this world is built
// from no map and runs no script either, so the authored map id at the end
// of its one entity record is zero and the widened check record multiplies
// zero checks. The version-56 value it replaces was 0x3100a6e61214935c.
//
// 1033 MOVES IT AGAIN (B3), on rlxTick1Digest's own ground: this world
// declares no structure either, so the new structure section is its own
// bare four-byte zero count. The version-57 value it replaces was
// 0x35b19645199c9e79.
//
// 1039 MOVES IT AGAIN: this unit carries five zero resistance bytes at the
// record tail. The version-58 value it replaces was 0x72cecb04daf3375e.
//
// 1047 MOVES IT AGAIN: this unit's turn is inactive after tick 1 (RotationSpeed
// 0, an instant snap), so its two new bytes are the current facing repeated
// (0, unmoved from the walk's own north step) and a zero remainder. The
// pre-story value it replaces was 0xefd89ce635e8a2ff.
// 1052 moves the form version; this fixture has no structure records. 1063
// moves it again for raw cloud counters; this fixture has no area record.
const hybTick1Digest uint64 = 0x94458c2bfc89451e

// TestTheStaleSourceLabelWouldCostThisRoute pins the search on the second grid:
// the route cell by cell, and the labels that decide it — including the one the
// two shapes disagree about.
func TestTheStaleSourceLabelWouldCostThisRoute(t *testing.T) {
	w := mustWorldGrid(t, rlxSeed, hybBounds, ModeCanonical, hybGrid(), []Entity{hybUnit})
	s := newRouteScratch(w)

	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, hybTargetX, hybTargetY)
	checkRoute(t, "(0,7) to (4,1) up the west wall", got, ok,
		[]cell{{0, 6}, {0, 5}, {0, 4}, {0, 3}, {1, 2}, {2, 1}, {3, 1}, {4, 1}})

	start := cell{hybUnit.X, hybUnit.Y}
	// (2,2) is the cell lowered inside the generation that relaxes it, and (3,1)
	// is what the two shapes disagree about: 64 here, 68 off a stale read.
	checkLabel(t, w, s, start, 2, 2, 52)
	checkLabel(t, w, s, start, 3, 1, 64)
	// The rest of the walk back's decision at the target: the equal orthogonal
	// that wins it, the runner-up beside it, and the equal diagonal that is
	// refused. Off a stale (3,1) the first of these offers 76 and loses.
	checkLabel(t, w, s, start, 4, 1, 72)
	checkLabel(t, w, s, start, 5, 1, 68)
	checkLabel(t, w, s, start, 5, 2, 60)
	// And the cells the contract's route is actually read back through.
	checkLabel(t, w, s, start, 2, 1, 56)
	checkLabel(t, w, s, start, 1, 2, 44)
	checkLabel(t, w, s, start, 0, 3, 32)
}

// TestTheSecondGridsTickOneDigest is the same fact where a replay meets it. Both
// shapes cost 18 and both arrive, so a test comparing costs or arrivals would
// pass under either; what separates them is which cell the unit stands on after
// one tick, and that is a byte of the canonical form.
func TestTheSecondGridsTickOneDigest(t *testing.T) {
	w := mustWorldGrid(t, rlxSeed, hybBounds, ModeCanonical, hybGrid(), []Entity{hybUnit})
	Step(w, []Command{{Entity: hybUnit.ID, X: hybTargetX, Y: hybTargetY}})

	want := Entity{ID: 1, X: 0, Y: 6, TargetX: hybTargetX, TargetY: hybTargetY, HasTarget: true,
		ActorState: actorStateGuard, Reach: 1, PostX: hybUnit.X, PostY: hybUnit.Y}
	if got := w.Entities()[0]; got != want {
		t.Errorf("after one tick the unit is %+v, want %+v", got, want)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if want := 61 + 34 + 3*64 + relationLen + 492 + 4 + 7*8 + groupCountLen + sackCountLen + carryCountLen + equipRecordLen + treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(1)) + 4 + 4 + 4 + 4 + 4 + scriptStateLen + scriptCountsLen + entityIDFloorLen + spellDeliverySpanLen; len(form) != want {
		t.Fatalf("the tick-1 form is %d bytes, want %d", len(form), want)
	}
	checkRoute(t, "the route stored after one tick", w.routes[0], true,
		[]cell{{0, 5}, {0, 4}, {0, 3}, {1, 2}, {2, 1}, {3, 1}, {4, 1}})
	requireStructureLegacyDigest(t, w, 0xd9491a80dad4f963)
	requireOriginalDeadLegacyDigest(t, w, 0xf70d99cafdeb0526)
	requireSpellbookLegacyDigest(t, w, 0x499c6c21294299c7)
	if got := fnv1a(strippedWorldOfSecondPhysical(form)); got != hybTick1Digest {
		t.Errorf("the tick-1 world hashes %#016x, derived by hand as %#016x\n form % x",
			got, hybTick1Digest, form)
	}
	requireHumanMovementLegacyDigest(t, w, 0x7cb0ff201526fbbd)
}

// TestTheSecondGridIsWalkedCellByCell carries it to arrival. A route is searched
// afresh each tick from wherever the unit now stands, so the walk is only the
// tick-1 route repeated if every later search agrees with it — which is what
// this pins, cell by cell, rather than assuming.
func TestTheSecondGridIsWalkedCellByCell(t *testing.T) {
	w := mustWorldGrid(t, rlxSeed, hybBounds, ModeCanonical, hybGrid(), []Entity{hybUnit})
	walk := []cell{{0, 6}, {0, 5}, {0, 4}, {0, 3}, {1, 2}, {2, 1}, {3, 1}, {4, 1}}

	for i, want := range walk {
		if i == 0 {
			Step(w, []Command{{Entity: hybUnit.ID, X: hybTargetX, Y: hybTargetY}})
		} else {
			Step(w, nil)
		}
		e := w.Entities()[0]
		if (cell{e.X, e.Y}) != want {
			t.Fatalf("tick %d: the unit is at (%d,%d), want (%d,%d)", i+1, e.X, e.Y, want.x, want.y)
		}
		last := i == len(walk)-1
		if e.HasTarget == last {
			t.Errorf("tick %d: the unit holds target %v, want %v", i+1, e.HasTarget, !last)
		}
	}
}

func TestAUnitAlreadyOnItsTargetNeitherSearchesNorSteps(t *testing.T) {
	// Sealed: a one-cell map, so every neighbour of the unit's cell is off it.
	sealed := mustWorldGrid(t, rlxSeed, Bounds{Width: 1, Height: 1}, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 0, Y: 0}})
	Step(sealed, []Command{{Entity: 1, X: 0, Y: 0}})
	if got := (sealed.Entities()[0]); got != (Entity{ID: 1, ActorState: actorStateGuard, Reach: 1}) {
		t.Errorf("the walled-in unit is %+v, want its target cleared with no residue", got)
	}

	// And on open ground, where a search WOULD have found something: the unit
	// does not take a step it has no reason to take.
	open := mustWorldGrid(t, rlxSeed, rlxBounds, ModeCanonical, rlxGrid(),
		[]Entity{{ID: 4, X: 3, Y: 3, TargetX: 3, TargetY: 3, HasTarget: true}})
	Step(open, nil)
	want := Entity{ID: 4, X: 3, Y: 3, ActorState: actorStateGuard, Reach: 1, PostX: 3, PostY: 3}
	if got := open.Entities()[0]; got != want {
		t.Errorf("the unit on its target is %+v, want %+v", got, want)
	}
}
