package sim

import (
	"encoding/binary"
	"testing"
)

// guardCircleWorld is a first-game World whose owner-9 Group stands under the
// guard setter's order 1 with the members given, and one owner-1 unit far off.
func guardCircleWorld(t *testing.T, members ...Entity) (*World, *SavedGroup) {
	t.Helper()
	ents := append([]Entity{engFighter(50, 1, 40, 40)}, members...)
	w := engWorld(t, engRel(t, [3]uint32{9, 1, 1}, [3]uint32{1, 9, 1}), ents...)
	for i := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[i]
		if g.Owner.Owner == 9 {
			if g.AI[0x20] != orderGuard {
				t.Fatalf("the owner-9 Group holds order %d, want Guard", g.AI[0x20])
			}
			return w, g
		}
	}
	t.Fatal("no owner-9 Group")
	return nil, nil
}

func guardLatchState(g *SavedGroup) (radius uint8, latch, count uint32) {
	return g.AI[0x2d], binary.LittleEndian.Uint32(g.AI[0x30:]), binary.LittleEndian.Uint32(g.AI[0x34:])
}

// TestLiveGuardCentreFloorsTheWholeCellMean is AI-739: live Guard averages the
// members' cell bytes and discards the fraction, so two members a cell apart
// centre on the lower cell, where the issue's sub-cell mean rounds up. The
// clip follows the stored centre.
func TestLiveGuardCentreFloorsTheWholeCellMean(t *testing.T) {
	t.Parallel()
	w, g := guardCircleWorld(t, engFighter(1, 9, 10, 10), engFighter(2, 9, 11, 11))
	members, _ := w.savedLiving(g)
	if x, y := groupCentroid(w.entities, members, subCellMean); x != 11 || y != 11 {
		t.Fatalf("the issue's sub-cell mean is %d,%d, want 11,11: the fixture does not separate the rules", x, y)
	}
	w.guardGeometry(g)
	if g.AI[0x28] != 10 || g.AI[0x29] != 10 {
		t.Fatalf("live Guard's centre cell is %d,%d, want the floored mean 10,10", g.AI[0x28], g.AI[0x29])
	}
	if got := binary.LittleEndian.Uint32(g.AI[0x24:]); got != 10<<16|10 {
		t.Fatalf("live Guard's coordinate words are %#x, want 10 and 10", got)
	}
	// A unit exactly the working radius below the floored centre is inside;
	// about the rounded centre it would be one cell out.
	r := int32(g.AI[0x2d])
	at := indexOfEntity(w.entities, 50)
	w.entities[at].X, w.entities[at].Y = 10-r, 10
	if kept := w.clipCircle(orderGuard, w.storedCircle(g, members), []int{at}); len(kept) != 1 {
		t.Fatalf("a unit %d cells from the floored centre was clipped at radius %d", r, r)
	}
}

// TestLiveGuardCountsADyingMemberInItsCentre is AI-739 with TRIG-REAP-017:
// the walk has no living test and a dying member stays in the Group's list
// until its teardown, so it still pulls the centre; a Group of dying members
// still runs Guard's latch block.
func TestLiveGuardCountsADyingMemberInItsCentre(t *testing.T) {
	t.Parallel()
	w, g := guardCircleWorld(t, engFighter(1, 9, 10, 10), engFighter(2, 9, 20, 10))
	w.entities[indexOfEntity(w.entities, 2)].HP = -1
	w.guardGeometry(g)
	if g.AI[0x28] != 15 {
		t.Fatalf("the centre x is %d, want 15 over the living and the dying member", g.AI[0x28])
	}

	w.entities[indexOfEntity(w.entities, 1)].HP = -1
	binary.LittleEndian.PutUint32(g.AI[0x30:], 1)
	w.savedDecision(g, orderGuard, nil)
	if _, latch, count := guardLatchState(g); latch != 1 || count != 1 {
		t.Fatalf("a Group of dying members holds latch %d counter %d after a Guard pass, want 1 and 1", latch, count)
	}
}

// TestLiveGuardRollsItsWorkingRadiusOnceTheLatchFlips is AI-741: the issue
// leaves working radius B; the first populated pass rolls B + 4 + jitter from
// one draw of the random service, clears the counter and sets the latch; later
// passes count and keep the radius. An empty Group with the latch set rolls
// without the 4 and clears both; the radius wraps as a byte.
func TestLiveGuardRollsItsWorkingRadiusOnceTheLatchFlips(t *testing.T) {
	t.Parallel()
	w, g := guardCircleWorld(t, engFighter(1, 9, 10, 10))
	base := g.AI[0x38]
	if r, latch, count := guardLatchState(g); r != base || latch != 0 || count != 0 {
		t.Fatalf("the issue left radius %d latch %d counter %d, want %d, 0, 0", r, latch, count, base)
	}
	predict := w.rng
	jitter := uint8(predict.aiRange(3) - 1)
	w.guardLatch(g)
	if r, latch, count := guardLatchState(g); r != base+4+jitter || latch != 1 || count != 0 {
		t.Fatalf("the first pass left radius %d latch %d counter %d, want %d, 1, 0", r, latch, count, base+4+jitter)
	}
	if w.rng != predict {
		t.Fatal("the roll did not take exactly one draw of the random service")
	}
	rolled := g.AI[0x2d]
	for pass := uint32(1); pass <= 3; pass++ {
		w.guardLatch(g)
		if r, latch, count := guardLatchState(g); r != rolled || latch != 1 || count != pass {
			t.Fatalf("pass %d left radius %d latch %d counter %d, want %d, 1, %d", pass+1, r, latch, count, rolled, pass)
		}
	}
	if w.rng != predict {
		t.Fatal("a pass with the latch set drew from the random service")
	}

	members := g.Members
	g.Members = nil
	jitter = uint8(predict.aiRange(3) - 1)
	w.guardLatch(g)
	if r, latch, count := guardLatchState(g); r != base+jitter || latch != 0 || count != 0 {
		t.Fatalf("an empty Group with the latch set left radius %d latch %d counter %d, want %d, 0, 0", r, latch, count, base+jitter)
	}
	w.guardLatch(g)
	if _, latch, count := guardLatchState(g); latch != 0 || count != 1 {
		t.Fatalf("an empty Group with the latch clear left latch %d counter %d, want 0 and 1", latch, count)
	}

	g.Members = members
	binary.LittleEndian.PutUint32(g.AI[0x30:], 0)
	g.AI[0x38] = 254
	jitter = uint8(predict.aiRange(3) - 1)
	w.guardLatch(g)
	if want := g.AI[0x38] + 4 + jitter; want >= 254 || g.AI[0x2d] != want {
		t.Fatalf("base 254 rolled radius %d, want the byte-wrapped %d", g.AI[0x2d], want)
	}
}

// TestLiveGuardClipsItsFirstPassAtTheBase is AI-741's order: Guard clips with
// the stored radius before its latch block, so the pass after the issue clips
// at B and only later passes at B + 4 + jitter. A foe B + 2 from the centre,
// in a member's sight, is left alone on the first pass and taken on the next.
func TestLiveGuardClipsItsFirstPassAtTheBase(t *testing.T) {
	t.Parallel()
	w, g := guardCircleWorld(t, engFighter(1, 9, 20, 20), engFighter(2, 9, 20, 20))
	base := int32(g.AI[0x38])
	if base != minimalGuardRange {
		t.Fatalf("the fixture's base is %d, want the floor %d", base, minimalGuardRange)
	}
	// Member 2 steps out after the issue; the centre follows it, the base
	// does not (AI-RADFREEZE-075).
	const reach = int32(3) // inside member 2's sight
	m2 := indexOfEntity(w.entities, 2)
	w.entities[m2].X = 20 + 2*(base+2-reach)
	foe := indexOfEntity(w.entities, 50)
	w.entities[foe].X, w.entities[foe].Y = w.entities[m2].X+reach, 20
	w.entities[foe].HP, w.entities[foe].MaxHP = 1<<20, 1<<20
	members, _ := w.savedLiving(g)
	w.guardGeometry(g)
	if d := int64(w.entities[foe].X) - int64(g.AI[0x28]); d != int64(base)+2 {
		t.Fatalf("the foe stands %d from the centre, want B + 2 = %d", d, base+2)
	}
	w.savedDecision(g, orderGuard, nil)
	for _, id := range []EntityID{1, 2} {
		if _, held := engVictim(w, id); held {
			t.Fatalf("member %d took the foe on the pass that clips at the base %d", id, base)
		}
	}
	if r := int32(g.AI[0x2d]); r < base+3 {
		t.Fatalf("the latch rolled radius %d, want at least B + 3", r)
	}
	if kept := w.clipCircle(orderGuard, w.storedCircle(g, members), []int{foe}); len(kept) != 1 {
		t.Fatal("the rolled radius does not keep the foe for the next pass")
	}
	w.savedDecision(g, orderGuard, nil)
	if v, held := engVictim(w, 2); !held || v != 50 {
		t.Fatalf("member 2 holds %v/%d on the second pass, want the foe 50", held, v)
	}
}
