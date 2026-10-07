package sim

// The movement domain: which bytes are domains, which terrain each one may
// cross, and what the byte form does with the field.
//
// The grid bytes this file crosses the three domains against are the three a
// derivation can produce — bit 0 alone, bit 1 alone, and both, the byte a border
// cell comes out as. Every one of the nine answers is written down, so no
// domain's mask is read off another's: a table that asserted "air differs from
// ground" would pass with air and ghost swapped.

import (
	"bytes"
	"strings"
	"testing"
)

// ------------------------------------------------- AC-1: which bytes are domains

// TestTheZeroValueIsTheGroundDomain — AC-1. An entity built naming no domain is
// a ground mover, which is what leaves every world assembled before the field
// existed behaving as it did.
func TestTheZeroValueIsTheGroundDomain(t *testing.T) {
	if DomainGround != 0 {
		t.Errorf("DomainGround is %d, and the contract makes it the zero value", DomainGround)
	}
	var e Entity
	if e.Domain != DomainGround {
		t.Errorf("an Entity built naming no domain is in domain %d", e.Domain)
	}
	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1}})
	if got := w.Entities()[0].Domain; got != DomainGround {
		t.Errorf("a world built from an entity naming no domain holds domain %d", got)
	}
}

// TestTheThreeDomainsAreTheirDocumentedValues — AC-1. The values are contract,
// not an implementation detail: they are the byte the form carries, so a build
// that renumbered them would read every stored world's movers into the wrong
// domain while every test that named the constants went on passing.
func TestTheThreeDomainsAreTheirDocumentedValues(t *testing.T) {
	for _, tc := range []struct {
		d    Domain
		want uint8
	}{
		{DomainGround, 0},
		{DomainGhost, 1},
		{DomainAir, 2},
	} {
		if uint8(tc.d) != tc.want {
			t.Errorf("domain %v is the byte %d, want %d", tc.d, uint8(tc.d), tc.want)
		}
		if !tc.d.defined() {
			t.Errorf("domain %d is not defined", uint8(tc.d))
		}
	}
	// And nothing else is. The whole byte range is swept rather than the first
	// value past the last one, so a build that defined a fourth domain somewhere
	// above fails here instead of quietly accepting it.
	for v := 0; v < 256; v++ {
		d := Domain(v)
		if want := v <= 2; d.defined() != want {
			t.Errorf("Domain(%d).defined() = %v, want %v", v, d.defined(), want)
		}
	}
}

// TestTheConstructorRefusesAnUndefinedDomain — AC-1. Refused, not folded into
// ground: a value normalised at construction would make two entities the byte
// form distinguishes into one world.
func TestTheConstructorRefusesAnUndefinedDomain(t *testing.T) {
	for _, v := range []uint8{3, 4, 127, 128, 255} {
		w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
			[]Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2, Domain: Domain(v)}})
		if err == nil {
			t.Errorf("domain %d was accepted, building %+v", v, w)
		}
		if w != nil {
			t.Errorf("domain %d was refused and a world came back anyway", v)
		}
	}
}

func TestGhostBearingConstructorsValidateTheTemplateDomain(t *testing.T) {
	b := Bounds{Width: 4, Height: 4}
	constructors := []struct {
		name  string
		build func(GhostTemplate) (*World, error)
	}{
		{"summoning", func(g GhostTemplate) (*World, error) {
			return NewSummoningWorld(1, b, ModeCanonical, Terrain{}, nil, nil,
				Relations{}, nil, nil, nil, g)
		}},
		{"structured", func(g GhostTemplate) (*World, error) {
			return NewStructuredWorld(1, b, ModeCanonical, Terrain{}, nil, nil,
				Relations{}, nil, nil, nil, g, nil)
		}},
	}
	for _, ctor := range constructors {
		t.Run(ctor.name, func(t *testing.T) {
			for _, d := range []Domain{DomainGround, DomainGhost, DomainAir} {
				w, err := ctor.build(GhostTemplate{Class: 1, Domain: d})
				if err != nil || w == nil {
					t.Errorf("defined domain %d: world=%v err=%v", uint8(d), w, err)
				}
			}
			for _, d := range []Domain{3, 255} {
				w, err := ctor.build(GhostTemplate{Class: 1, Domain: d})
				if err == nil {
					t.Errorf("undefined domain %d was accepted", uint8(d))
				} else if !strings.Contains(err.Error(), "ghost template: movement domain") {
					t.Errorf("undefined domain %d returned %q", uint8(d), err)
				}
				if w != nil {
					t.Errorf("undefined domain %d returned a world", uint8(d))
				}
			}
		})
	}
}

// ------------------------------------------------- AC-2: the terrain each may cross

// TestEachDomainCrossesItsOwnTerrain — AC-2. The nine answers, written out.
//
// The three grid bytes are every byte the derivation can produce: bit 0 for
// water, mountain, the impassable flag and scenery; bit 1 for nothing but the
// border; and both together, which is what a border cell comes out as. So the
// row for bit 0 is the one that says a flyer crosses water, and the row for both
// is the one that says the border stops everything.
func TestEachDomainCrossesItsOwnTerrain(t *testing.T) {
	const (
		clear  = byte(0)
		ground = blockGround
		air    = blockAir
		border = blockGround | blockAir
	)
	for _, tc := range []struct {
		what                  string
		cell                  byte
		wantGround, wantGhost bool
		wantAir               bool
	}{
		{"an open cell", clear, true, true, true},
		{"water, mountain, the impassable flag or scenery", ground, false, true, true},
		{"the air bit alone, which no derivation writes without bit 0", air, true, false, false},
		{"a border cell, both bits", border, false, false, false},
	} {
		w := mustWorldGrid(t, 1, Bounds{Width: 3, Height: 1}, ModeCanonical,
			[]byte{0, tc.cell, 0},
			[]Entity{{ID: 1, X: 0, Y: 0}})
		got := [3]bool{
			w.terrainOpen(DomainGround, 1, 0),
			w.terrainOpen(DomainGhost, 1, 0),
			w.terrainOpen(DomainAir, 1, 0),
		}
		want := [3]bool{tc.wantGround, tc.wantGhost, tc.wantAir}
		if got != want {
			t.Errorf("%s (%#02x): ground/ghost/air may stand there = %v, want %v",
				tc.what, tc.cell, got, want)
		}
	}
}

// TestNoDomainCrossesTheBounds — AC-2. In bounds is asked of every domain: a
// flyer is not off the map's edge merely because no bit is set out there.
func TestNoDomainCrossesTheBounds(t *testing.T) {
	w := mustWorldGrid(t, 1, Bounds{Width: 2, Height: 2}, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 0, Y: 0}})
	for _, d := range []Domain{DomainGround, DomainGhost, DomainAir} {
		for _, c := range []cell{{-1, 0}, {0, -1}, {2, 0}, {0, 2}} {
			if w.terrainOpen(d, c.x, c.y) {
				t.Errorf("domain %d may stand on (%d,%d), which is outside a 2x2 map", uint8(d), c.x, c.y)
			}
		}
	}
}

// ------------------------------------------------- AC-9: the form

// TestEveryDomainRoundTripsThroughTheForm — AC-9. Each of the three is carried
// and read back, and the field is INJECTIVE: three worlds alike but for one
// mover's domain marshal to three different byte strings, so no two forms decode
// to one world.
func TestEveryDomainRoundTripsThroughTheForm(t *testing.T) {
	forms := make(map[string]Domain)
	for _, d := range []Domain{DomainGround, DomainGhost, DomainAir} {
		w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{{ID: 1, X: 1, Y: 1, Domain: d}})
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		if other, seen := forms[string(form)]; seen {
			t.Errorf("domains %d and %d marshal to one form", uint8(other), uint8(d))
		}
		forms[string(form)] = d

		var got World
		if err := got.UnmarshalBinary(form); err != nil {
			t.Fatalf("UnmarshalBinary at domain %d: %v", uint8(d), err)
		}
		if back := got.Entities()[0].Domain; back != d {
			t.Errorf("domain %d read back as %d", uint8(d), uint8(back))
		}
	}
}

// TestUnmarshalRefusesAnUndefinedDomainAndLeavesTheReceiver — AC-9. The decoder
// is the constructor's other half, and a refusal leaves the receiving world
// exactly as it was — not half a world carrying one decoded record.
func TestUnmarshalRefusesAnUndefinedDomainAndLeavesTheReceiver(t *testing.T) {
	src := mustWorld(t, 1, Bounds{Width: 4, Height: 4},
		[]Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}})
	valid, err := src.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// The header is 34 bytes, the THREE planes 16 cells apiece, and the domain
	// sits at each record's +34 — so with a 291-byte record the two records'
	// domain bytes are at 116 and 407. Written out rather than computed off
	// entityLen, which would assert the encoder against itself.
	for _, off := range []int{116, 407} {
		for _, v := range []byte{3, 0xff} {
			w := populated(t)
			before := snap(w)
			if err := w.UnmarshalBinary(withByte(valid, off, v)); err == nil {
				t.Errorf("a domain byte of %d at offset %d was accepted", v, off)
			}
			if got := snap(w); !equalState(got, before) {
				t.Errorf("the receiver changed:\n before %+v\n after  %+v", before, got)
			}
		}
	}
	// And the same form unspoiled is accepted, so the cases above measure the
	// domain check and not something else about this fixture.
	if err := (&World{}).UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

// TestARouteCrossingAClosedCellReadsBackForEveryDomain — AC-9. A flyer's route
// crosses a cell that blocks ground and reads back, and so does the same route
// on a ground mover: a route is not tested against terrain.
func TestARouteCrossingAClosedCellReadsBackForEveryDomain(t *testing.T) {
	// A 4x1 strip whose middle cell blocks GROUND only — water, not border.
	grid := []byte{0, blockGround, 0, 0}
	route := []cell{{1, 0}, {2, 0}}

	for _, tc := range []struct {
		what   string
		domain Domain
		accept bool
	}{
		{"a flyer over water", DomainAir, true},
		{"a ghost over water", DomainGhost, true},
		{"a ground mover over water", DomainGround, true},
	} {
		w := mustWorldGrid(t, 1, Bounds{Width: 4, Height: 1}, ModeCanonical, grid,
			[]Entity{{ID: 1, X: 0, Y: 0, TargetX: 2, TargetY: 0, HasTarget: true, Domain: tc.domain}})
		w.routes[0] = append([]cell(nil), route...)

		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("%s: MarshalBinary: %v", tc.what, err)
		}
		var got World
		err = got.UnmarshalBinary(form)
		if tc.accept {
			if err != nil {
				t.Errorf("%s: its own form was refused: %v", tc.what, err)
				continue
			}
			if !bytes.Equal(mustForm(t, &got), form) {
				t.Errorf("%s: the decoded world does not marshal back to the same bytes", tc.what)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: a route across a cell its domain cannot cross was accepted", tc.what)
		}
	}
}

// mustForm marshals w, with the never-nil error handled once.
func mustForm(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

// ------------------------------------------------- AC-3: which movers contend

// domRun steps w until nothing holds a target or the tick budget is spent, and
// answers the entities as they end up.
func domRun(t *testing.T, w *World, ticks int) []Entity {
	t.Helper()
	for i := 0; i < ticks; i++ {
		busy := false
		for _, e := range w.Entities() {
			if e.HasTarget {
				busy = true
			}
		}
		if !busy {
			return w.Entities()
		}
		Step(w, nil)
	}
	t.Fatalf("nothing settled in %d ticks", ticks)
	return nil
}

// TestOnlyTheMoversOfOneLayerContend — AC-3. The three pairings, each asserted
// on its own: a flyer and a ground unit ordered onto one cell END ON IT; a ghost
// and a ground unit do not; and two ground units do not either, which is the
// hard collision this story leaves byte for byte alone.
//
// The three cases share a layout so the only thing that differs between them is
// a domain, and the ground pair is in the table rather than assumed: without it,
// "the ghost pair ended distinct" would be evidence about ghosts only if ground
// units were known to do the same, and that is the fact being relied on.
func TestOnlyTheMoversOfOneLayerContend(t *testing.T) {
	const goalX, goalY = 4, 2
	for _, tc := range []struct {
		what   string
		second Domain
		share  bool
	}{
		{"a flyer and a ground unit", DomainAir, true},
		{"a ghost and a ground unit", DomainGhost, false},
		{"two ground units", DomainGround, false},
	} {
		w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
			{ID: 1, X: 0, Y: 2},
			{ID: 2, X: 8, Y: 2, Domain: tc.second},
		})
		Step(w, []Command{
			{Entity: 1, X: goalX, Y: goalY},
			{Entity: 2, X: goalX, Y: goalY},
		})
		ents := domRun(t, w, 64)

		same := ents[0].X == ents[1].X && ents[0].Y == ents[1].Y
		if same != tc.share {
			t.Errorf("%s ordered onto one cell ended at (%d,%d) and (%d,%d); sharing = %v, want %v",
				tc.what, ents[0].X, ents[0].Y, ents[1].X, ents[1].Y, same, tc.share)
		}
		// The lower id takes the ordered cell under every one of the three, so a
		// case that shared could not have done it by both missing.
		if ents[0].X != goalX || ents[0].Y != goalY {
			t.Errorf("%s: the lower id ended at (%d,%d), not on the cell it was sent to",
				tc.what, ents[0].X, ents[0].Y)
		}
	}
}

// TestALayerIsCountedSeparatelyAtEveryCell — AC-3. The predicate under the
// planes, asked directly: a cell holding a flyer is free to a ground mover and
// to a ghost, and closed to a second flyer; a cell holding a ground mover is the
// mirror of that.
func TestALayerIsCountedSeparatelyAtEveryCell(t *testing.T) {
	for _, tc := range []struct {
		what     string
		occupant Domain
		wantFor  map[Domain]bool
	}{
		{"a cell a flyer stands on", DomainAir,
			map[Domain]bool{DomainGround: true, DomainGhost: true, DomainAir: false}},
		{"a cell a ground unit stands on", DomainGround,
			map[Domain]bool{DomainGround: false, DomainGhost: false, DomainAir: true}},
		{"a cell a ghost stands on", DomainGhost,
			map[Domain]bool{DomainGround: false, DomainGhost: false, DomainAir: true}},
	} {
		for asker, want := range tc.wantFor {
			w := mustWorldGrid(t, 1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{
				{ID: 1, X: 2, Y: 2, Domain: tc.occupant},
				{ID: 2, X: 0, Y: 0, Domain: asker},
			})
			s := newRouteScratch(w)
			if got := w.enterable(s, 1, 2, 2); got != want {
				t.Errorf("%s: a mover of domain %d may enter it = %v, want %v",
					tc.what, uint8(asker), got, want)
			}
		}
	}
}

// ------------------------------------------------- AC-4, AC-5: the soft rule

// domTrack steps w for at most ticks, recording every entity's cell after each
// tick, and stops once nothing holds a target.
func domTrack(t *testing.T, w *World, ticks int) [][]Entity {
	t.Helper()
	var frames [][]Entity
	for i := 0; i < ticks; i++ {
		Step(w, nil)
		frames = append(frames, w.Entities())
		busy := false
		for _, e := range w.Entities() {
			if e.HasTarget {
				busy = true
			}
		}
		if !busy {
			return frames
		}
	}
	t.Fatalf("nothing settled in %d ticks", ticks)
	return nil
}

// TestTwoMovingFlyersInterpenetrate — AC-4. Two flyers ordered onto each other's
// start cells meet on ONE cell and both still arrive. Two ground units on the
// same row also both arrive — by going ROUND each other, since a near search
// that cannot enter its waypoint settles for a cell beside it — but they never
// share one, and that is the contrast: the flyers pass THROUGH, the ground pair
// passes AROUND.
//
// The ground pair is not decoration. "The flyers shared a cell" says nothing
// about a soft rule unless the same layout under the shipped rule does not.
func TestTwoMovingFlyersInterpenetrate(t *testing.T) {
	for _, tc := range []struct {
		what              string
		domain            Domain
		wantShare, arrive bool
	}{
		{"two flyers", DomainAir, true, true},
		{"two ground units", DomainGround, false, true},
	} {
		const west, east, row = 1, 7, 2
		w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
			{ID: 1, X: west, Y: row, Domain: tc.domain},
			{ID: 2, X: east, Y: row, Domain: tc.domain},
		})
		Step(w, []Command{
			{Entity: 1, X: east, Y: row},
			{Entity: 2, X: west, Y: row},
		})
		frames := domTrack(t, w, 64)

		shared := false
		for _, f := range frames {
			if f[0].X == f[1].X && f[0].Y == f[1].Y {
				shared = true
			}
		}
		if shared != tc.wantShare {
			t.Errorf("%s on crossing paths shared a cell = %v, want %v", tc.what, shared, tc.wantShare)
		}
		last := frames[len(frames)-1]
		arrived := last[0].X == east && last[0].Y == row && last[1].X == west && last[1].Y == row
		if arrived != tc.arrive {
			t.Errorf("%s ended at (%d,%d) and (%d,%d); both arrived = %v, want %v",
				tc.what, last[0].X, last[0].Y, last[1].X, last[1].Y, arrived, tc.arrive)
		}
	}
}

// TestAFlyerCrossesARestingFlyersCell — AC-5. A resting flyer stands on the
// straight line to a free target, and the mover flies THROUGH its cell and lands
// exactly on the target: no detour, so its row never changes.
//
// A ground unit on the same layout reaches the same target and NEVER stands on
// the occupied cell: it leaves its row, goes round and comes back. So the two
// halves are told apart by the row and not by the arrival — a hard collision
// costs a detour, a soft one costs nothing, and "no detour" is what makes the
// flyer's straight line evidence about the soft rule.
func TestAFlyerCrossesARestingFlyersCell(t *testing.T) {
	for _, tc := range []struct {
		what                   string
		domain                 Domain
		crosses, moves, detour bool
	}{
		{"a flyer over a resting flyer", DomainAir, true, true, false},
		{"a ground unit past a standing ground unit", DomainGround, false, true, true},
	} {
		const row, sitter, goal = 2, 4, 8
		w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
			{ID: 1, X: 0, Y: row, Domain: tc.domain},
			{ID: 2, X: sitter, Y: row, Domain: tc.domain},
		})
		Step(w, []Command{{Entity: 1, X: goal, Y: row}})
		frames := domTrack(t, w, 64)

		crossed, offRow := false, false
		for _, f := range frames {
			if f[0].X == sitter && f[0].Y == row {
				crossed = true
			}
			if f[0].Y != row {
				offRow = true
			}
		}
		if crossed != tc.crosses {
			t.Errorf("%s stood on the occupied cell = %v, want %v", tc.what, crossed, tc.crosses)
		}
		if offRow != tc.detour {
			t.Errorf("%s left its row = %v, want %v", tc.what, offRow, tc.detour)
		}
		last := frames[len(frames)-1][0]
		arrived := last.X == goal && last.Y == row
		if arrived != tc.moves {
			t.Errorf("%s ended at (%d,%d); reached the free target = %v, want %v",
				tc.what, last.X, last.Y, arrived, tc.moves)
		}
		// The sitter never moved: it was given no order.
		if e := frames[len(frames)-1][1]; e.X != sitter || e.Y != row {
			t.Errorf("%s: the unit standing still ended at (%d,%d)", tc.what, e.X, e.Y)
		}
	}
}

// TestAFlyerEntersItsLayerWhenItsOrderEnds — AC-4. The tick a flyer's order ends
// is the tick it becomes visible: before it, a peer's predicate says its cell is
// free; after it, the same predicate says it is taken.
func TestAFlyerEntersItsLayerWhenItsOrderEnds(t *testing.T) {
	w := mustWorldGrid(t, 1, Bounds{Width: 6, Height: 3}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 0, Y: 1, Domain: DomainAir},
		{ID: 2, X: 5, Y: 1, Domain: DomainAir},
	})
	Step(w, []Command{{Entity: 1, X: 2, Y: 1}})

	// While it is walking, its own cell is free to its peer.
	if e := w.Entities()[0]; !e.HasTarget {
		t.Fatalf("the flyer settled in one tick, at (%d,%d) — this fixture needs it still walking", e.X, e.Y)
	}
	moving := w.Entities()[0]
	if s := newRouteScratch(w); !w.enterable(s, 1, moving.X, moving.Y) {
		t.Errorf("a moving flyer's cell (%d,%d) is closed to its peer", moving.X, moving.Y)
	}

	domRun(t, w, 32)
	at := w.Entities()[0]
	if at.HasTarget {
		t.Fatalf("the flyer still holds an order")
	}
	if s := newRouteScratch(w); w.enterable(s, 1, at.X, at.Y) {
		t.Errorf("a resting flyer's cell (%d,%d) is open to its peer", at.X, at.Y)
	}
}

// ------------------------------------------------- AC-6, AC-7, AC-11: where an order ends

func domRestOverlap(ents []Entity) bool {
	for i := range ents {
		for j := i + 1; j < len(ents); j++ {
			a, b := ents[i], ents[j]
			if a.Domain != DomainAir || b.Domain != DomainAir {
				continue
			}
			if !a.HasTarget && !b.HasTarget && a.X == b.X && a.Y == b.Y {
				return true
			}
		}
	}
	return false
}

// TestTwoFlyersOrderedOntoOneCellEndDistinct — AC-6. They interpenetrate while
// moving and end on different cells, and at no tick are two RESTING flyers on
// one cell. The second fixture starts them equidistant, so both would arrive on
// the same tick if nothing separated them.
func TestTwoFlyersOrderedOntoOneCellEndDistinct(t *testing.T) {
	for _, tc := range []struct {
		what string
		a, b int32
		goal int32
		tie  bool
	}{
		{"from different distances, so one arrives first", 0, 7, 4, false},
		{"equidistant, so they arrive on the same tick", 1, 7, 4, true},
	} {
		w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
			{ID: 1, X: tc.a, Y: 2, Domain: DomainAir},
			{ID: 2, X: tc.b, Y: 2, Domain: DomainAir},
		})
		Step(w, []Command{{Entity: 1, X: tc.goal, Y: 2}, {Entity: 2, X: tc.goal, Y: 2}})
		frames := domTrack(t, w, 64)

		for k, f := range frames {
			if domRestOverlap(f) {
				t.Fatalf("%s: two resting flyers share a cell at tick %d", tc.what, k+1)
			}
		}
		// Exactly one of them ends on the cell that was ordered and the other
		// beside it. WHICH one is not id order in general — the nearer flyer gets
		// there first — so the tie-break is asserted only where there is a tie.
		last := frames[len(frames)-1]
		on := 0
		for k, e := range last {
			if e.X == tc.goal && e.Y == 2 {
				on++
				continue
			}
			if d := (cell{e.X, e.Y}).chebyshevTo(cell{tc.goal, 2}); d != 1 {
				t.Errorf("%s: id %d ended at (%d,%d), %d cells from the ordered cell, want it or a neighbour",
					tc.what, k+1, e.X, e.Y, d)
			}
		}
		if on != 1 {
			t.Errorf("%s: %d of the two ended on the ordered cell (%d,2), want exactly one; they are at "+
				"(%d,%d) and (%d,%d)", tc.what, on, tc.goal, last[0].X, last[0].Y, last[1].X, last[1].Y)
		}
		if tc.tie && !(last[0].X == tc.goal && last[0].Y == 2) {
			t.Errorf("%s: the two arrive together and the LOWER id must take the cell; it ended at (%d,%d)",
				tc.what, last[0].X, last[0].Y)
		}
	}
}

// TestAFlyerOrderedOntoARestingFlyerSettlesBeside — AC-7. The order names the
// substitute from the FIRST tick, so no later tick re-runs the sweep, and the
// mover never stands on the resting flyer's cell at rest.
func TestAFlyerOrderedOntoARestingFlyerSettlesBeside(t *testing.T) {
	const sitX, sitY int32 = 6, 2
	w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 0, Y: 2, Domain: DomainAir},
		{ID: 2, X: sitX, Y: sitY, Domain: DomainAir},
	})
	Step(w, []Command{{Entity: 1, X: sitX, Y: sitY}})

	after := w.Entities()[0]
	if !after.HasTarget {
		t.Fatalf("the order ended in the first tick, at (%d,%d)", after.X, after.Y)
	}
	if after.TargetX == sitX && after.TargetY == sitY {
		t.Errorf("the order still names the resting flyer's cell (%d,%d) after one tick", sitX, sitY)
	}
	settled := cell{after.TargetX, after.TargetY}
	if d := settled.chebyshevTo(cell{sitX, sitY}); d != 1 {
		t.Errorf("the substitute (%d,%d) is %d cells from the cell ordered, want adjacent",
			settled.x, settled.y, d)
	}

	frames := domTrack(t, w, 64)
	for k, f := range frames {
		if domRestOverlap(f) {
			t.Fatalf("two resting flyers share a cell at tick %d", k+1)
		}
		// And the order never goes back to naming the occupied cell.
		if f[0].HasTarget && f[0].TargetX == sitX && f[0].TargetY == sitY {
			t.Fatalf("at tick %d the order names the occupied cell again — a later tick re-ran the sweep", k+1)
		}
	}
	if last := frames[len(frames)-1][0]; last.X != settled.x || last.Y != settled.y {
		t.Errorf("it ended at (%d,%d), not on the cell it settled for (%d,%d)",
			last.X, last.Y, settled.x, settled.y)
	}
}

// TestTheNonSettlingSearchRefusesAnOccupiedGoal — AC-13. The same fixture under
// the mode whose search does not settle: the order ends in the FIRST tick, the
// mover never reaches the peer's cell, and no later tick runs a search — which is
// what the stored-route test would otherwise force every tick, unbounded.
func TestTheNonSettlingSearchRefusesAnOccupiedGoal(t *testing.T) {
	const sitX, sitY int32 = 6, 2
	w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeOptimised, nil, []Entity{
		{ID: 1, X: 0, Y: 2, Domain: DomainAir},
		{ID: 2, X: sitX, Y: sitY, Domain: DomainAir},
	})
	Step(w, []Command{{Entity: 1, X: sitX, Y: sitY}})

	e := w.Entities()[0]
	if e.HasTarget {
		t.Errorf("the order survived the first tick, aiming at (%d,%d)", e.TargetX, e.TargetY)
	}
	if e.X == sitX && e.Y == sitY {
		t.Errorf("the mover reached the resting flyer's cell")
	}
	if len(w.routes[0]) != 0 {
		t.Errorf("it holds a route of %d cell(s) with no order", len(w.routes[0]))
	}
	// A free goal on the same fixture still routes, so the refusal above is the
	// rest term and not the mode refusing everything.
	Step(w, []Command{{Entity: 1, X: 8, Y: 4}})
	if !w.Entities()[0].HasTarget {
		t.Errorf("a free goal was refused too, so the case above measures nothing")
	}
}

// TestAFlyerReOrderedOntoItsOwnCellEndsCounted — AC-11's first half. A
// zero-distance order clears in the tick it is given, and the flyer is in its
// layer at the end of that tick — the re-seed fires on the settle and not only
// on an advancing move.
func TestAFlyerReOrderedOntoItsOwnCellEndsCounted(t *testing.T) {
	w := mustWorldGrid(t, 1, Bounds{Width: 6, Height: 3}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 2, Y: 1, Domain: DomainAir},
		{ID: 2, X: 5, Y: 1, Domain: DomainAir},
	})
	Step(w, []Command{{Entity: 1, X: 2, Y: 1}})

	e := w.Entities()[0]
	if e.HasTarget || e.X != 2 || e.Y != 1 {
		t.Fatalf("after a zero-distance order it is %+v", e)
	}
	if s := newRouteScratch(w); w.restFree(s, 1, 2, 1) {
		t.Errorf("its cell is free to a peer, so the settle did not count it")
	}
}

// TestAFlyerSettlingOnItsOwnCellEndsItsOrder — AC-11, and the case the rest
// filter could break. The mover's own cell is a candidate the picker may return,
// meaning "it can get no nearer"; a filter that took the asker's own presence off
// unconditionally would drive that cell's count below zero and remove it from the
// ring, sending the flyer somewhere further out instead of ending the order.
//
// The fixture seals a flyer into one cell with anti-air terrain and orders it
// away: the wave labels its start and nothing else, so the only candidate any
// ring can hold is that cell.
func TestAFlyerSettlingOnItsOwnCellEndsItsOrder(t *testing.T) {
	const w0, h0 = 5, 5
	grid := make([]byte, w0*h0)
	for y := 0; y < h0; y++ {
		for x := 0; x < w0; x++ {
			if x != 2 || y != 2 {
				grid[y*w0+x] = blockAir | blockGround
			}
		}
	}
	w := mustWorldGrid(t, 1, Bounds{Width: w0, Height: h0}, ModeCanonical, grid,
		[]Entity{{ID: 1, X: 2, Y: 2, Domain: DomainAir}})
	Step(w, []Command{{Entity: 1, X: 4, Y: 4}})

	e := w.Entities()[0]
	if e.HasTarget {
		t.Errorf("a sealed flyer still holds an order aiming at (%d,%d)", e.TargetX, e.TargetY)
	}
	if e.X != 2 || e.Y != 2 || e.Stall != 0 {
		t.Errorf("it is %+v, want its own cell with no residue", e)
	}
	if s := newRouteScratch(w); w.restFree(s, 0, 2, 2) == false {
		t.Errorf("its own cell reads as taken by itself — the self term is subtracted wrongly")
	}
}

func TestAFlyerWhoseSearchFailsDoesNotPublishAnOccupiedRest(t *testing.T) {
	const w0, h0 = 5, 5
	grid := make([]byte, w0*h0)
	for y := 0; y < h0; y++ {
		for x := 0; x < w0; x++ {
			if x != 2 || y != 2 {
				grid[y*w0+x] = blockAir | blockGround
			}
		}
	}
	// Two flyers already sharing the one open cell: a world whose entities share
	// a cell is advanced, not repaired.
	w := mustWorldGrid(t, 1, Bounds{Width: w0, Height: h0}, ModeCanonical, grid, []Entity{
		{ID: 1, X: 2, Y: 2, Domain: DomainAir},
		{ID: 2, X: 2, Y: 2, Domain: DomainAir},
	})
	Step(w, []Command{{Entity: 2, X: 0, Y: 0}})

	ents := w.Entities()
	if !ents[1].HasTarget {
		t.Errorf("the sealed flyer cleared the order whose resting transition was refused")
	}
	if domRestOverlap(ents) {
		t.Errorf("the two flyers at (%d,%d) and (%d,%d) were both published as resting",
			ents[0].X, ents[0].Y, ents[1].X, ents[1].Y)
	}
	s := newRouteScratch(w)
	i, _ := w.cellIndex(2, 2)
	if got := s.at(DomainAir.layer(), i); got != 1 {
		t.Errorf("the occupied cell carries %d resting flyer(s), want only the pre-existing one", got)
	}
}

// TestASettleSkipsARingCellAnotherFlyerRestsOn — AC-7, and the case the settle
// filter is the only thing that answers.
//
// Two flyers rest side by side, and a third is ordered onto the FURTHER of them.
// The picker rings outward from the cell ordered and takes the smallest label —
// the cell cheapest to reach from the mover — and the nearer resting flyer's own
// cell is exactly that: it is the one neighbour of the goal that lies straight
// back along the mover's line, so every other candidate in the ring costs a
// diagonal more.
//
// So a picker that filtered on the label alone would settle the mover ONTO its
// peer. The centre is never probed, which is why AC-7's simpler fixture cannot
// tell the two apart: there the only rest cell in reach is the goal itself.
func TestASettleSkipsARingCellAnotherFlyerRestsOn(t *testing.T) {
	const nearX, farX, row int32 = 5, 6, 2
	w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 0, Y: row, Domain: DomainAir},
		{ID: 2, X: nearX, Y: row, Domain: DomainAir},
		{ID: 3, X: farX, Y: row, Domain: DomainAir},
	})
	Step(w, []Command{{Entity: 1, X: farX, Y: row}})

	e := w.Entities()[0]
	if !e.HasTarget {
		t.Fatalf("the order ended in the first tick, at (%d,%d)", e.X, e.Y)
	}
	settled := cell{e.TargetX, e.TargetY}
	if settled == (cell{nearX, row}) {
		t.Fatalf("it settled for (%d,%d), which the nearer flyer is resting on", settled.x, settled.y)
	}
	if settled == (cell{farX, row}) {
		t.Fatalf("it settled for the cell ordered, which the further flyer is resting on")
	}
	if d := settled.chebyshevTo(cell{farX, row}); d != 1 {
		t.Errorf("it settled for (%d,%d), %d cells from the cell ordered, want a neighbour",
			settled.x, settled.y, d)
	}

	frames := domTrack(t, w, 64)
	for k, f := range frames {
		if domRestOverlap(f) {
			t.Fatalf("two resting flyers share a cell at tick %d", k+1)
		}
	}
	if last := frames[len(frames)-1][0]; last.X != settled.x || last.Y != settled.y {
		t.Errorf("it ended at (%d,%d), not on the cell it settled for (%d,%d)",
			last.X, last.Y, settled.x, settled.y)
	}
}

// ------------------------------------------------- AC-8, AC-10

// TestAFlyerNeverStandsOnAnAntiAirCell — AC-8. A wall of bit-1 cells crosses the
// straight line to the target with one gap in it. The flyer never stands on a
// walled cell at any tick and still reaches the target, so it went round; and a
// GROUND mover on the same grid walks straight through the wall, which is what
// says the wall is bit 1 and not bit 0.
func TestAFlyerNeverStandsOnAnAntiAirCell(t *testing.T) {
	const w0, h0 = 9, 7
	const wallX, gapY int32 = 4, 6
	grid := make([]byte, w0*h0)
	for y := int32(0); y < h0; y++ {
		if y != gapY {
			grid[y*w0+wallX] = blockAir
		}
	}
	for _, tc := range []struct {
		what      string
		domain    Domain
		wantCross bool
	}{
		{"a flyer", DomainAir, false},
		{"a ground mover", DomainGround, true},
	} {
		w := mustWorldGrid(t, 1, Bounds{Width: w0, Height: h0}, ModeCanonical, grid,
			[]Entity{{ID: 1, X: 0, Y: 0, Domain: tc.domain}})
		Step(w, []Command{{Entity: 1, X: 8, Y: 0}})
		frames := domTrack(t, w, 128)

		crossed := false
		for k, f := range frames {
			e := f[0]
			if e.X == wallX && e.Y != gapY {
				if !tc.wantCross {
					t.Fatalf("%s stood on the walled cell (%d,%d) at tick %d", tc.what, e.X, e.Y, k+1)
				}
				crossed = true
			}
		}
		if crossed != tc.wantCross {
			t.Errorf("%s crossed the wall away from its gap = %v, want %v", tc.what, crossed, tc.wantCross)
		}
		if last := frames[len(frames)-1][0]; last.X != 8 || last.Y != 0 {
			t.Errorf("%s ended at (%d,%d), want the target (8,0)", tc.what, last.X, last.Y)
		}
	}
}

// TestAWorldOfEveryDomainIsDeterministic — AC-10. Two worlds built from the same
// description, carrying all three domains and a soft-collision fixture, advanced
// under one schedule: equal digests and equal byte forms at every tick.
//
// The fixture is one whose outcome the domain rules decide — two flyers
// converging, a ghost and a ground unit contending, a flyer crossing both — so a
// rule that read a map or a clock would show up as a divergence rather than as
// two worlds that never had anything to disagree about.
func TestAWorldOfEveryDomainIsDeterministic(t *testing.T) {
	build := func() *World {
		return mustWorldGrid(t, 0x5eed, Bounds{Width: 12, Height: 9}, ModeCanonical, nil, []Entity{
			{ID: 1, X: 0, Y: 4, Domain: DomainAir},
			{ID: 2, X: 11, Y: 4, Domain: DomainAir},
			{ID: 3, X: 5, Y: 0, Domain: DomainGhost},
			{ID: 4, X: 5, Y: 8, Domain: DomainGround},
			{ID: 5, X: 6, Y: 4, Domain: DomainAir},
		})
	}
	cmds := []Command{
		{Entity: 1, X: 6, Y: 4},
		{Entity: 2, X: 6, Y: 4},
		{Entity: 3, X: 5, Y: 8},
		{Entity: 4, X: 5, Y: 0},
	}

	a, b := build(), build()
	for tick := 0; tick < 40; tick++ {
		if tick == 0 {
			Step(a, cmds)
			Step(b, cmds)
		} else {
			Step(a, nil)
			Step(b, nil)
		}
		if a.Hash() != b.Hash() {
			t.Fatalf("tick %d: digests %#016x and %#016x", tick+1, a.Hash(), b.Hash())
		}
		fa, fb := mustForm(t, a), mustForm(t, b)
		if !bytes.Equal(fa, fb) {
			t.Fatalf("tick %d: the two forms differ", tick+1)
		}
	}
	// The run has to have been a run: a fixture nothing moved in would hash equal
	// for a reason this test is not about.
	if a.Tick() != 40 {
		t.Fatalf("the world is at tick %d", a.Tick())
	}
	moved := 0
	for _, e := range a.Entities() {
		if e.X != build().Entities()[int(e.ID)-1].X || e.Y != build().Entities()[int(e.ID)-1].Y {
			moved++
		}
	}
	if moved < 3 {
		t.Errorf("only %d of the five entities moved over 40 ticks", moved)
	}
}
