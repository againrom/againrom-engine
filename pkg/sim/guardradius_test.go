package sim

// The frozen group record itself (0095 T1): its key set, its base, and what
// a decode does and does not refuse about it.

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// ---------------------------------------------------------------- AC-1

// grdBounds is the small extent every fixture in this file shares.
var grdBounds = Bounds{Width: 8, Height: 8}

// TestABuiltWorldHoldsOneGroupRecordPerOwnedPairAscending is AC-1 whole: an
// entity in slot 0 contributes no record, a group all of whose members are
// dead still holds one, the two records are ascending by (owner, group) — NOT
// by either entity's id or by input order — and building the same entities in
// a different slice order gives the same records.
func TestABuiltWorldHoldsOneGroupRecordPerOwnedPairAscending(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 0, Group: 99, HP: 5, MaxHP: 5}, // slot 0: no group at all
		{ID: 2, X: 1, Y: 1, Owner: 2, Group: 5, HP: 5, MaxHP: 5, ScanRange: 20},
		{ID: 3, X: 2, Y: 2, Owner: 1, Group: 9, HP: -5, MaxHP: 5}, // dead
		{ID: 4, X: 3, Y: 3, Owner: 1, Group: 9, HP: -3, MaxHP: 5}, // dead, same group as 3
	}
	want := []groupAI{
		// ascends before owner 2 though both its members died; owner 1 is
		// SelfSlot, so construction gives it Stand Ground.
		{owner: 1, group: 9, base: minimalGuardRange, order: orderStandGround},
		{owner: 2, group: 5, base: 20, order: orderGuard},
	}

	w := mustWorld(t, 1, grdBounds, ents)
	if got := w.groups; !equalGroups(got, want) {
		t.Fatalf("groups are %+v, want %+v", got, want)
	}

	// Reversed input order must build the identical record set: AC-1's own
	// closing clause.
	reversed := make([]Entity, len(ents))
	for i, e := range ents {
		reversed[len(ents)-1-i] = e
	}
	rw := mustWorld(t, 1, grdBounds, reversed)
	if got := rw.groups; !equalGroups(got, want) {
		t.Fatalf("reversed input built groups %+v, want %+v", got, want)
	}
	if w.Hash() != rw.Hash() {
		t.Errorf("the same entities in two orders hash %#016x and %#016x", w.Hash(), rw.Hash())
	}
}

func equalGroups(got, want []groupAI) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------- AC-2

// TestTheBaseFollowsTheGreatestDistancePlusRangeNotEitherAlone is AC-2: three
// members of one group, each the sole holder of one of three properties — the
// furthest from the centroid, the widest sight range, and the greatest SUM of
// the two — and the base follows only the third.
//
// The centroid is the plain average of three collinear points (Y held at 0,
// so Chebyshev distance is |dx|), chosen so the arithmetic divides evenly:
// members at 130, 95 and 75 average to 100 exactly, giving distances 30, 5
// and 25. Member 1 is the sole furthest (30) but carries the narrowest range
// (5): sum 35. Member 2 is the sole widest range (50) but is the nearest
// (5): sum 55. Member 3 is neither furthest nor widest but carries the
// greatest SUM: 25+40=65, ahead of both 35 and 55 — and every sum here stays
// well under a byte, so AC-3's own narrowing (tested separately) cannot be
// mistaken for this property.
func TestTheBaseFollowsTheGreatestDistancePlusRangeNotEitherAlone(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 130, Y: 0, Owner: 7, Group: 0, HP: 5, MaxHP: 5, ScanRange: 5},
		{ID: 2, X: 95, Y: 0, Owner: 7, Group: 0, HP: 5, MaxHP: 5, ScanRange: 50},
		{ID: 3, X: 75, Y: 0, Owner: 7, Group: 0, HP: 5, MaxHP: 5, ScanRange: 40},
	}

	cx, cy := groupCentroid([]Entity{ents[0], ents[1], ents[2]}, []int{0, 1, 2})
	if cx != 100 || cy != 0 {
		t.Fatalf("fixture's own centroid is (%d,%d), want (100,0) — the arithmetic this test explains "+
			"no longer holds", cx, cy)
	}
	d0 := cellOf(&ents[0]).chebyshevTo(cell{x: cx, y: cy})
	d1 := cellOf(&ents[1]).chebyshevTo(cell{x: cx, y: cy})
	d2 := cellOf(&ents[2]).chebyshevTo(cell{x: cx, y: cy})
	if d0 != 30 || d1 != 5 || d2 != 25 {
		t.Fatalf("fixture's own distances are %d, %d, %d, want 30, 5, 25", d0, d1, d2)
	}
	sum0 := int64(ents[0].ScanRange) + d0
	sum1 := int64(ents[1].ScanRange) + d1
	sum2 := int64(ents[2].ScanRange) + d2
	if !(d0 > d1 && d0 > d2) {
		t.Fatalf("member 0 is not the sole furthest: distances %d, %d, %d", d0, d1, d2)
	}
	if !(ents[1].ScanRange > ents[0].ScanRange && ents[1].ScanRange > ents[2].ScanRange) {
		t.Fatalf("member 1 is not the sole widest range: ranges %d, %d, %d",
			ents[0].ScanRange, ents[1].ScanRange, ents[2].ScanRange)
	}
	if !(sum2 > sum0 && sum2 > sum1) {
		t.Fatalf("member 2 is not the sole greatest sum: sums %d, %d, %d", sum0, sum1, sum2)
	}

	w := mustWorld(t, 1, Bounds{Width: 140, Height: 8}, ents)
	if len(w.groups) != 1 {
		t.Fatalf("groups are %+v, want exactly one", w.groups)
	}
	if got, want := w.groups[0].base, uint8(sum2); got != want {
		t.Errorf("base is %d, want %d — the member with the greatest distance-plus-range, "+
			"not the furthest member (would give %d) nor the widest range (would give %d)",
			got, want, uint8(sum0), uint8(sum1))
	}
}

// ---------------------------------------------------------------- AC-3

// TestABaseUnderTheFloorIsRaisedToIt is AC-3's first edge: a group whose
// living geometry is under the guard-range floor takes the floor, and one at
// or above it takes its geometry unchanged.
func TestABaseUnderTheFloorIsRaisedToIt(t *testing.T) {
	under := mustWorld(t, 1, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 3, Group: 0, HP: 5, MaxHP: 5, ScanRange: minimalGuardRange - 1},
	})
	if got := under.groups[0].base; got != minimalGuardRange {
		t.Errorf("a geometry of %d based to %d, want the floor %d",
			minimalGuardRange-1, got, minimalGuardRange)
	}

	at := mustWorld(t, 1, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 3, Group: 0, HP: 5, MaxHP: 5, ScanRange: minimalGuardRange},
	})
	if got := at.groups[0].base; got != minimalGuardRange {
		t.Errorf("a geometry exactly at the floor based to %d, want %d", got, minimalGuardRange)
	}

	over := mustWorld(t, 1, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 3, Group: 0, HP: 5, MaxHP: 5, ScanRange: minimalGuardRange + 5},
	})
	if got := over.groups[0].base; got != minimalGuardRange+5 {
		t.Errorf("a geometry over the floor based to %d, want its own geometry %d",
			got, minimalGuardRange+5)
	}
}

func TestAGroupWithNoLivingMemberTakesTheFloorBare(t *testing.T) {
	w := mustWorld(t, 1, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 4, Group: 0, HP: -1, MaxHP: 5, ScanRange: 255},
		{ID: 2, X: 5, Y: 5, Owner: 4, Group: 0, HP: 0, MaxHP: 5, ScanRange: 255}, // downed, not alive either
	})
	if len(w.groups) != 1 {
		t.Fatalf("groups are %+v, want exactly one", w.groups)
	}
	if got := w.groups[0].base; got != minimalGuardRange {
		t.Errorf("a group with no living member based to %d, want the bare floor %d",
			got, minimalGuardRange)
	}
}

// TestAGeometryPastAByteIsNarrowedNotClamped is AC-3's third edge: the
// distance-plus-range summand is narrowed to a byte by truncation, the way
// the field's own width takes it, and NOT clamped to 255.
//
// Three collinear members average to X 1000 exactly (1250+900+850)/3): the
// first stands 250 cells out and carries the top sight range, 255, so its
// summand is 505 — past a byte — and narrows to 505-256=249, not to 255. The
// other two carry no range at all, so their own summands are just their
// distances, 100 and 150, and 249 beats both without needing to be believed
// on its own: a wrong clamp would read 255 here, which is checked against
// directly.
func TestAGeometryPastAByteIsNarrowedNotClamped(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 1250, Y: 0, Owner: 6, Group: 0, HP: 5, MaxHP: 5, ScanRange: 255},
		{ID: 2, X: 900, Y: 0, Owner: 6, Group: 0, HP: 5, MaxHP: 5},
		{ID: 3, X: 850, Y: 0, Owner: 6, Group: 0, HP: 5, MaxHP: 5},
	}
	cx, cy := groupCentroid(ents, []int{0, 1, 2})
	if cx != 1000 || cy != 0 {
		t.Fatalf("fixture's own centroid is (%d,%d), want (1000,0)", cx, cy)
	}
	d := cellOf(&ents[0]).chebyshevTo(cell{x: cx, y: cy})
	if d != 250 {
		t.Fatalf("fixture's own distance is %d, want 250", d)
	}
	raw := d + int64(ents[0].ScanRange)
	if raw != 505 {
		t.Fatalf("fixture's own raw summand is %d, want 505", raw)
	}

	w := mustWorld(t, 1, Bounds{Width: 2000, Height: 8}, ents)
	if len(w.groups) != 1 {
		t.Fatalf("groups are %+v, want exactly one", w.groups)
	}
	if got := w.groups[0].base; got != 249 {
		t.Errorf("base is %d, want 249 (505 narrowed to a byte) — 255 would mean the summand was "+
			"clamped instead of narrowed", got)
	}
}

func TestStepLeavesTheGroupRecordUntouched(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 16, Height: 16}, []Entity{
		{ID: 1, X: 0, Y: 0, TargetX: 8, TargetY: 8, HasTarget: true, Owner: 1, Group: 0,
			HP: 5, MaxHP: 5, Speed: 20, ScanRange: 12},
		{ID: 2, X: 15, Y: 15, Owner: 2, Group: 0, HP: -1, MaxHP: 5}, // dead
	})
	before := append([]groupAI(nil), w.groups...)
	for i := 0; i < 20; i++ {
		Step(w, nil)
	}
	if !equalGroups(w.groups, before) {
		t.Errorf("after 20 ticks the groups are %+v, want the unchanged %+v", w.groups, before)
	}
}

// ---------------------------------------------------------------- AC-6

// TestTwoWorldsDifferingOnlyInOneBaseHashDifferently is the assertion that
// does not care what any digest IS — R-4's own mitigation, since the digest
// pin elsewhere in this package is derived from a hand transcription and
// could agree with a broken encoder by coincidence, but this differential
// could not.
func TestTwoWorldsDifferingOnlyInOneBaseHashDifferently(t *testing.T) {
	build := func(scan uint8) *World {
		return mustWorld(t, 7, grdBounds, []Entity{
			{ID: 1, X: 0, Y: 0, Owner: 1, Group: 0, HP: 5, MaxHP: 5, ScanRange: scan},
			{ID: 2, X: 4, Y: 4, Owner: 2, Group: 0, HP: 5, MaxHP: 5, ScanRange: 9},
		})
	}
	a, b := build(20), build(21)
	if a.groups[0].base == b.groups[0].base {
		t.Fatalf("the fixture's own bases are equal (%d); this test needs them to differ",
			a.groups[0].base)
	}
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in one base hash alike (%#016x) — "+
			"the field does not reach the byte form", a.Hash())
	}
	fa, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	fb, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if bytes.Equal(fa, fb) {
		t.Errorf("two worlds differing only in one base marshal to the same bytes")
	}
	// And the same world twice is still the same world.
	if build(20).Hash() != a.Hash() {
		t.Errorf("one world built twice hashes differently")
	}
}

func TestADecodedBaseCrossesTheFormWholeAtBothEdges(t *testing.T) {
	w := mustWorld(t, 3, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 5, Group: 0, HP: 5, MaxHP: 5, ScanRange: 100},
	})
	if got := w.groups[0].base; got != 100 {
		t.Fatalf("fixture's own base is %d, want 100", got)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	cells := gridCells(grdBounds)
	// header, three planes, one entity record, one (empty) route count, the
	// group count, one owner word and one group word — the base is the next
	// byte, written out from the contract exactly as binary_test.go's own
	// offset tables are.
	baseAt := headerLen + 3*int(cells) + entityLen + routeCountLen + groupCountLen + 4 + 4
	if got := form[baseAt]; got != 100 {
		t.Fatalf("the base byte sits at %d as %d, want 100 — this fixture's own layout assumption is wrong",
			baseAt, got)
	}

	for _, edge := range []byte{0, 255} {
		spoiled := withByte(form, baseAt, edge)
		var back World
		if err := back.UnmarshalBinary(spoiled); err != nil {
			t.Fatalf("a base of %d was refused: %v", edge, err)
		}
		if len(back.groups) != 1 || back.groups[0].base != edge {
			t.Fatalf("decoded groups are %+v, want one record with base %d", back.groups, edge)
		}
		again, err := back.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		if !bytes.Equal(again, spoiled) {
			t.Errorf("base %d: re-marshalling gave\n % x\nwant\n % x", edge, again, spoiled)
		}
	}
}

func TestADecodedCommandedCellCrossesTheFormWholeAtBothExtremes(t *testing.T) {
	w := mustWorld(t, 3, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 5, Group: 0, HP: 5, MaxHP: 5, ScanRange: 100},
	})
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	cells := gridCells(grdBounds)
	// header, three planes, one entity record, one (empty) route count, the
	// group count, one owner word, one group word and the base — the order
	// byte is next, and the commanded cell after it, written out from the
	// contract exactly as binary_test.go's own offset tables are.
	orderAt := headerLen + 3*int(cells) + entityLen + routeCountLen + groupCountLen + 4 + 4 + 1
	cxAt := orderAt + 1
	cyAt := cxAt + 4

	for _, tc := range []struct{ x, y int32 }{
		{math.MinInt32, math.MaxInt32},
		{math.MaxInt32, math.MinInt32},
	} {
		spoiled := withU32(withU32(form, cxAt, uint32(tc.x)), cyAt, uint32(tc.y))
		var back World
		if err := back.UnmarshalBinary(spoiled); err != nil {
			t.Fatalf("a commanded cell of (%d,%d) was refused: %v", tc.x, tc.y, err)
		}
		if len(back.groups) != 1 || back.groups[0].commandedX != tc.x || back.groups[0].commandedY != tc.y {
			t.Fatalf("decoded groups are %+v, want one record with commanded cell (%d,%d)",
				back.groups, tc.x, tc.y)
		}
		again, err := back.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		if !bytes.Equal(again, spoiled) {
			t.Errorf("commanded cell (%d,%d): re-marshalling gave\n % x\nwant\n % x",
				tc.x, tc.y, again, spoiled)
		}
	}
}

func TestUnmarshalRefusesAnOrderOutsideTheSix(t *testing.T) {
	valid := grValid(t)
	cells := gridCells(grdBounds)
	orderAt := headerLen + 3*int(cells) + entityLen + routeCountLen + groupCountLen + 4 + 4 + 1

	for _, bad := range []byte{6, 255} {
		t.Run(fmt.Sprintf("order %d", bad), func(t *testing.T) {
			spoiled := withByte(valid, orderAt, bad)
			var w World
			if err := w.UnmarshalBinary(spoiled); err == nil {
				t.Errorf("an order of %d was accepted", bad)
			}
		})
	}
	// The control: the six orders this build DOES write are not refused, so the
	// cases above are refused for their own value and not because every form
	// here is refused.
	for _, good := range []byte{0, orderGuard, orderSwarm, orderStandGround, orderMove, orderSwarm2} {
		t.Run(fmt.Sprintf("order %d", good), func(t *testing.T) {
			spoiled := withByte(valid, orderAt, good)
			var w World
			if err := w.UnmarshalBinary(spoiled); err != nil {
				t.Errorf("order %d was refused: %v", good, err)
			}
		})
	}
}

// TestAWorldHoldingNoOwnedEntityHoldsNoGroupAndRoundTrips is AC-7's other
// clause: a world naming no owned entity at all holds zero records, marshals
// to the section's bare empty count, and decodes back to the same nothing.
func TestAWorldHoldingNoOwnedEntityHoldsNoGroupAndRoundTrips(t *testing.T) {
	w := mustWorld(t, 1, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, HP: 5, MaxHP: 5}, // Owner 0: no group at all
	})
	if len(w.groups) != 0 {
		t.Fatalf("groups are %+v, want none", w.groups)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if len(back.groups) != 0 {
		t.Errorf("decoded groups are %+v, want none", back.groups)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(again, form) {
		t.Errorf("re-marshalling gave\n % x\nwant\n % x", again, form)
	}
}

// grValid is a small, well-formed form to spoil: one owned entity, alive,
// alone in its group — so the group section is a count of one and one
// record — over grdBounds.
func grValid(t *testing.T) []byte {
	t.Helper()
	w := mustWorld(t, 9, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 1, Group: 0, HP: 5, MaxHP: 5, ScanRange: 12},
	})
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return form
}

func TestUnmarshalRefusesAMalformedGroupSection(t *testing.T) {
	valid := grValid(t)
	cells := gridCells(grdBounds)
	countAt := headerLen + 3*int(cells) + entityLen + routeCountLen

	cases := []struct {
		name string
		data []byte
	}{
		{"a declared count the buffer cannot hold", withU32(valid, countAt, 0xffffffff)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w World
			if err := w.UnmarshalBinary(tc.data); err == nil {
				t.Errorf("accepted")
			}
		})
	}

	// Two records, out of ascending order and then duplicated: built by hand
	// rather than spoiling grValid, since that fixture holds only one. Its own
	// count sits past TWO entity records and TWO route counts, unlike the
	// single-record fixture's countAt above.
	two := mustWorld(t, 9, grdBounds, []Entity{
		{ID: 1, X: 0, Y: 0, Owner: 1, Group: 0, HP: 5, MaxHP: 5, ScanRange: 12},
		{ID: 2, X: 1, Y: 1, Owner: 2, Group: 0, HP: 5, MaxHP: 5, ScanRange: 12},
	})
	twoForm, err := two.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	twoCountAt := headerLen + 3*int(cells) + entityLen*2 + routeCountLen*2
	rec0At := twoCountAt + groupCountLen
	rec1At := rec0At + groupRecordLen
	// The two records as this form holds them: owner 1 then owner 2,
	// ascending. Swap the two owner words to break the order.
	descending := append([]byte(nil), twoForm...)
	o1 := append([]byte(nil), descending[rec0At:rec0At+4]...)
	o2 := append([]byte(nil), descending[rec1At:rec1At+4]...)
	copy(descending[rec0At:rec0At+4], o2)
	copy(descending[rec1At:rec1At+4], o1)
	if err := (&World{}).UnmarshalBinary(descending); err == nil {
		t.Errorf("a group section out of ascending order was accepted")
	}

	// Duplicated: the second record's owner and group set to the first's.
	duplicated := append([]byte(nil), twoForm...)
	copy(duplicated[rec1At:rec1At+8], duplicated[rec0At:rec0At+8])
	if err := (&World{}).UnmarshalBinary(duplicated); err == nil {
		t.Errorf("a duplicated group pair was accepted")
	}

	// The controls: neither the single-record nor the two-record valid forms
	// may be refused, so the cases above are refused for what they carry and
	// not because every form here is refused.
	if err := (&World{}).UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled single-record form was refused: %v", err)
	}
	if err := (&World{}).UnmarshalBinary(twoForm); err != nil {
		t.Errorf("the unspoiled two-record form was refused: %v", err)
	}
}

func TestAMismatchedGroupSectionKeySetIsNotRefused(t *testing.T) {
	valid := grValid(t)
	cells := gridCells(grdBounds)
	ownerAt := headerLen + 3*int(cells) + entityLen + routeCountLen + groupCountLen

	// The one record's owner set to a slot no entity holds. The entity
	// itself still carries owner 1; only the group section's copy is wrong.
	wrong := withU32(valid, ownerAt, 404)
	var back World
	if err := back.UnmarshalBinary(wrong); err != nil {
		t.Fatalf("a group section naming an owner no entity holds was refused: %v — "+
			"if this now fails, FR-8's entity cross-check has been implemented and this "+
			"test's own comment is stale", err)
	}
	if len(back.groups) != 1 || back.groups[0].owner != 404 {
		t.Errorf("decoded groups are %+v, want the mismatched owner carried through whole", back.groups)
	}
}
