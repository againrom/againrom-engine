package ui

// A corpse cannot be picked, and one that dies while selected is SKIPPED rather
// than dropped.
//
// The two halves are measured separately on purpose. What a tap and a box do is
// a question about the hit test; what a mark and an order do about an id already
// in the set is a question about one filter — and a build that put the filter on
// the marks and left the picks alone would look right in a window and would let
// a click walk a corpse.

import (
	"image"
	"reflect"
	"testing"
)

// dsEntities is the snapshot every case here runs against: one alive unit, one
// DOWNED and one DEAD, each on a cell of its own, plus a second alive unit
// sharing the corpse's cell.
//
// The shared cell is the discriminating one. A tap there must take the living
// unit, and it must do so however the ids compare — id 9 is the corpse and its
// living neighbour is 14, so a walk that took the lower id without first
// dropping the dead would pick the corpse.
func dsEntities() []MapEntity {
	return []MapEntity{
		{ID: 3, Cell: image.Pt(2, 2), Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: 6, Cell: image.Pt(4, 2), Life: LifeDowned, HP: 0, MaxHP: 100},
		{ID: 9, Cell: image.Pt(2, 4), Life: LifeDead, HP: -10, MaxHP: 100},
		{ID: 14, Cell: image.Pt(2, 4), Life: LifeAlive, HP: 40, MaxHP: 100},
	}
}

// dsViewer is a command-mode viewer holding that snapshot, over the same camera
// and grid the rest of this package's pick tests use.
func dsViewer(t *testing.T) *Viewer {
	t.Helper()
	v := commandViewer(t)
	// The map screen's own mode, which is what latches a left drag as a
	// rectangle rather than a pan; the tap cases do not need it and the box case
	// cannot be driven without it.
	v.commandMode = true
	v.SetEntities(dsEntities())
	return v
}

// TestATapTakesTheLivingAndNeverTheDead is AC-11's tap half. Four taps: on the
// alive unit, on the downed one, on the cell a corpse shares with a living unit,
// and on a cell holding a corpse alone — the last of which must clear, exactly
// as a tap on empty ground does, because a corpse is a miss and not a hit.
func TestATapTakesTheLivingAndNeverTheDead(t *testing.T) {
	corpseAlone := []MapEntity{{ID: 9, Cell: image.Pt(2, 4), Life: LifeDead, HP: -10, MaxHP: 100}}

	for _, tc := range []struct {
		name     string
		ents     []MapEntity
		col, row int
		want     selection
	}{
		{"an alive unit", dsEntities(), 2, 2, selection{3}},
		{"a downed unit, which is still commandable", dsEntities(), 4, 2, selection{6}},
		{"a corpse sharing its cell with a living unit", dsEntities(), 2, 4, selection{14}},
		{"a corpse alone on its cell, which is a miss", corpseAlone, 2, 4, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := dsViewer(t)
			v.SetEntities(tc.ents)
			// Something is selected first, so a case whose answer is "nothing" is
			// shown to have CLEARED rather than merely to have left an empty set
			// where an empty set already was.
			v.sel = selection{3, 6, 9, 14}
			x, y := cellPoint(v, tc.col, tc.row)
			tapAt(v, x, y)
			if !reflect.DeepEqual(v.sel, tc.want) {
				t.Errorf("the tap left %v, want %v", v.sel, tc.want)
			}
		})
	}
}

// TestABoxTakesEveryCoveredUnitThatIsNotDead is AC-11's box half: the covered
// units that are not dead, ascending, and never a corpse.
//
// The three releases are the three shapes a battlefield offers. Over everything:
// the corpse is dropped from a set that keeps three others. Over the shared cell
// alone: the living occupant is taken and the corpse beside it is not, which a
// count could not tell apart from taking both. And over a corpse ALONE, where a
// box that caught no unit clears — the same outcome as a release over bare
// ground, since a corpse is no more a candidate than an empty cell is.
func TestABoxTakesEveryCoveredUnitThatIsNotDead(t *testing.T) {
	corpseAlone := []MapEntity{{ID: 9, Cell: image.Pt(2, 4), Life: LifeDead, HP: -10, MaxHP: 100}}

	for _, tc := range []struct {
		name           string
		ents           []MapEntity
		c0, r0, c1, r1 int
		want           selection
	}{
		{"over every unit there is", dsEntities(), 0, 0, 6, 6, selection{3, 6, 14}},
		{"over the cell a corpse shares with a living unit", dsEntities(), 1, 3, 3, 5, selection{14}},
		{"over a corpse alone, which preserves", corpseAlone, 1, 3, 3, 5, selection{3, 6, 9, 14}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := dsViewer(t)
			v.SetEntities(tc.ents)
			// Something is selected first, so a case whose answer is "nothing" is
			// shown to have CLEARED rather than to have left an empty set alone.
			v.sel = selection{3, 6, 9, 14}
			x0, y0, x1, y1 := cellSpan(v, tc.c0, tc.r0, tc.c1, tc.r1)
			dragTo(v, x0, y0, x1, y1)
			if !reflect.DeepEqual(v.sel, tc.want) {
				t.Errorf("the box left %v, want %v", v.sel, tc.want)
			}
		})
	}
}

func TestADeadIdIsSkippedByBothReadersAndStaysInTheSet(t *testing.T) {
	v := dsViewer(t)
	before := selection{3, 6, 9, 14}
	v.sel = append(selection(nil), before...)

	// The marks. One cell each for the three that are not dead, in ascending id,
	// and the corpse's cell appears only because a LIVING unit stands on it.
	wantCells := []image.Point{{X: 2, Y: 2}, {X: 4, Y: 2}, {X: 2, Y: 4}}
	if got := cellsOf(presentSelected(v.sel, v.entities)); !reflect.DeepEqual(got, wantCells) {
		t.Errorf("the marks stand on %v, want %v", got, wantCells)
	}

	// The orders. One per marked unit, all naming the pressed cell, ascending.
	x, y := cellPoint(v, 7, 7)
	ords, ok := tapAt(v, x, y)
	if !ok {
		t.Fatalf("the tap issued nothing at all")
	}
	wantOrders := []order{
		{kind: orderKindMove, entity: 3, x: 7, y: 7},
		{kind: orderKindMove, entity: 6, x: 7, y: 7},
		{kind: orderKindMove, entity: 14, x: 7, y: 7},
	}
	if !reflect.DeepEqual(ords, wantOrders) {
		t.Errorf("the press issued %v, want %v — a dead id is ordered by nothing", ords, wantOrders)
	}

	// And the set is untouched, WHOLE and in order. Nothing but a tap or a
	// release replaces it, and neither a mark nor an order is either.
	if !reflect.DeepEqual(v.sel, before) {
		t.Errorf("the selection is %v after being read twice, want %v — a dead id is skipped, not dropped",
			v.sel, before)
	}
}

func TestOneFilterAnswersBothReaders(t *testing.T) {
	cell := image.Pt(3, 3)
	for _, tc := range []struct {
		name string
		life uint8
		hp   int
	}{
		{"alive", LifeAlive, 100},
		{"downed", LifeDowned, 0},
		{"dead", LifeDead, -30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities([]MapEntity{
				{ID: 4, Cell: image.Pt(1, 1), Life: LifeAlive, HP: 100, MaxHP: 100},
				{ID: 7, Cell: cell, Life: tc.life, HP: tc.hp, MaxHP: 100},
			})
			v.sel = selection{4, 7}

			marked := cellsOf(presentSelected(v.sel, v.entities))
			x, y := cellPoint(v, 6, 6)
			ords, _ := tapAt(v, x, y)
			if len(marked) != len(ords) {
				t.Fatalf("%d mark(s) and %d order(s) — one predicate answers both", len(marked), len(ords))
			}
			// Cell for cell: the k-th mark stands where the k-th order's unit does,
			// which is what makes them the same list rather than two of one length.
			ents := v.entities
			for k, o := range ords {
				var stood image.Point
				for _, e := range ents {
					if e.ID == o.entity {
						stood = e.Cell
					}
				}
				if marked[k] != stood {
					t.Errorf("mark %d stands on %v and order %d names unit %d, standing on %v",
						k, marked[k], k, o.entity, stood)
				}
			}
			wantN := 2
			if tc.life == LifeDead {
				wantN = 1
			}
			if len(ords) != wantN {
				t.Errorf("a %s unit beside a living one yields %d order(s), want %d", tc.name, len(ords), wantN)
			}
		})
	}
}
